package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	l "github.com/noisersup/encryptedfs-api/logger"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type Database struct {
	conn *pgx.Conn
	root uuid.UUID
}

type File struct {
	Id          uuid.UUID
	Name        string
	Hash        string
	parentId    uuid.UUID
	Duplicate   int
	IsDirectory bool
}

// Connects to databased with provided data
// and returns database object
func ConnectDB(uri, database string, root string) (*Database, error) {
	config, err := pgx.ParseConfig(os.ExpandEnv(uri))
	if err != nil {
		return nil, err
	}

	config.Database = database

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	db := Database{conn: conn}

	err = db.fetchRoot()
	if err != nil {
		return nil, err
	}

	return &db, nil
}

// Close database connection (conn.Close())
func (db *Database) Close() error {
	l.Log("Closing database...")
	return db.conn.Close(context.Background())
}

func (db *Database) fetchRoot() error {
	row := db.conn.QueryRow(context.Background(), "SELECT root FROM file_tree_config;")
	var root uuid.UUID
	err := row.Scan(&root)
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		return err
		//TODO:setRoot
	}

	db.root = root
	l.LogV("root: %s", db.root.String())
	return nil
}

func getHashOfFile(fileName, key []byte) string {
	hash := sha256.Sum256(append(fileName, key...))
	return fmt.Sprintf("%x", hash)
}

func (db *Database) NewUser(username, hashedPassword string) error {
	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return db.newUser(context.Background(), tx, username, hashedPassword)
	})
}

func (db *Database) newUser(ctx context.Context, tx pgx.Tx, username, hashedPassword string) error {
	sqlFormula := "INSERT INTO users (username, password) VALUES ($1,$2);"

	if _, err := tx.Exec(ctx, sqlFormula, username, hashedPassword); err != nil {
		return err
	}
	return nil
}

func (db *Database) GetPasswordOfUser(username string) (string, error) {
	var expectedPassword string
	err := db.conn.QueryRow(context.Background(), "SELECT password FROM users WHERE username=$1;", username).Scan(&expectedPassword)
	if err != nil {
		return "", err
	}
	return expectedPassword, err
}

// Adds file entry to database
func (db *Database) NewFile(pathNames []string, key []byte, duplicate int, isDirectory bool) error {
	if len(pathNames) == 0 {
		return fmt.Errorf("NewFile: no path provided")
	}
	parentId := db.root

	/*
		Warning!!!
		Be careful with using recursion in go (also in production environments...).
		Go compiler doesn't implement tail call optimization so it is possible to overflow the stack.
	*/
	err := func() error {
		// If only one file in path return from recursion and add it to database
		if len(pathNames) == 1 {
			return nil
		}

		// check if parent of file exists
		f, err := getFile(db.conn, pathNames[:len(pathNames)-1], db.root)
		if err != nil {
			// if parent doesn't exist create it
			if err == FileNotFound {
				err = db.NewFile(pathNames[:len(pathNames)-1], key, 0, true)
				if err != nil {
					if err != FileExists {
						return err
					}
				}

				// we're sure that the parent of file exists (i guess...)
				// now we can get it's database id to link our file to it
				f, err = getFile(db.conn, pathNames[:len(pathNames)-1], db.root)
				if err != nil {
					return err
				}
				parentId = f.Id
				return nil
			}
			return err
		}

		//if parent exists set variable parentId to it's id
		parentId = f.Id
		return nil
	}()

	if err != nil {
		return err
	}

	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return newFile(context.Background(), tx, pathNames[len(pathNames)-1], getHashOfFile([]byte(pathNames[len(pathNames)-1]), key), parentId, duplicate, isDirectory)

	})
}

func (db *Database) GetFile(pathNames []string) (*File, error) {
	return getFile(db.conn, pathNames, db.root)
}

func (db *Database) ListDirectory(id ...uuid.UUID) ([]File, error) {
	var dirId uuid.UUID
	if len(id) == 0 {
		dirId = db.root
	} else {
		dirId = id[0]
	}
	return listDirectory(db.conn, dirId)
}

func listDirectory(conn *pgx.Conn, id uuid.UUID) ([]File, error) {
	files := []File{}
	rows, err := conn.Query(context.Background(), "SELECT id, encrypted_name, hash, parent_id,duplicate, is_directory FROM file_tree WHERE parent_id = $1 ;", id)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		f := File{}
		if err := rows.Scan(&f.Id, &f.Name, &f.Hash, &f.parentId, &f.Duplicate, &f.IsDirectory); err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	if len(files) == 0 {
		return nil, FileNotFound
	}

	rows.Close()

	return files, nil
}

func (db *Database) DeleteFile(pathNames []string) error {
	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return deleteFile(db.conn, context.Background(), tx, pathNames, db.root)
	})
}

func deleteFile(conn *pgx.Conn, ctx context.Context, tx pgx.Tx, pathNames []string, root uuid.UUID) error {
	f, err := getFile(conn, pathNames, root)
	if err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, "DELETE FROM file_tree WHERE id = $1;", f.Id); err != nil {
		return err
	}

	var filePath string

	if f.Duplicate == 0 {
		filePath = fmt.Sprintf("./files/%s", f.Hash)
	} else {
		filePath = fmt.Sprintf("./files/%s%d", f.Hash, f.Duplicate)
	}

	return os.Remove(filePath)
}

func newFile(ctx context.Context, tx pgx.Tx, name string, hash string, parent uuid.UUID, duplicate int, isDirectory bool) error {
	if len(name) > 255 {
		return errors.New("Filename too big")
	}
	log.Print(name, " ", hash)
	sqlFormula := "INSERT INTO file_tree (encrypted_name,hash, parent_id, duplicate, is_directory) VALUES ($1, $2, $3, $4, $5);"
	log.Print(hash)
	if _, err := tx.Exec(ctx, sqlFormula, name, hash, parent, duplicate, isDirectory); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return FileExists
		}
		return err
	}
	return nil
}

var FileNotFound error = errors.New("File not found")
var FileExists error = errors.New("File exists")

func getFile(conn *pgx.Conn, pathNames []string, parent uuid.UUID) (*File, error) {
	if len(pathNames) == 0 {
		return nil, errors.New("pathNames empty")
	}

	f := File{}

	//handle null uuid
	rows, err := conn.Query(context.Background(), "SELECT id, encrypted_name, hash, parent_id,duplicate, is_directory FROM file_tree WHERE encrypted_name = $1 AND parent_id = $2;", pathNames[0], parent)
	if err != nil {
		return nil, err
	}

	fileFound := false

	for rows.Next() {
		if err := rows.Scan(&f.Id, &f.Name, &f.Hash, &f.parentId, &f.Duplicate, &f.IsDirectory); err != nil {
			return nil, err
		}
		fileFound = true
	}

	if !fileFound {
		return nil, FileNotFound
	}

	rows.Close()

	if len(pathNames) == 1 {
		return &f, nil
	}

	return getFile(conn, pathNames[1:], f.Id)
}
