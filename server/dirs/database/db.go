package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type Database struct {
	conn *pgx.Conn
	root uuid.UUID
}

type File struct {
	id       uuid.UUID
	Name     string
	Hash     string
	parentId uuid.UUID
}

func ConnectDB(uri, database string, root string) *Database {
	config, err := pgx.ParseConfig(os.ExpandEnv(uri))
	if err != nil {
		log.Fatal(err)
	}

	config.Database = database

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	// defer conn.Close(context.Background())
	db := Database{conn: conn}
	err = db.fetchRoot()
	if err != nil {
		log.Fatal(err)
	}

	return &db
}

func (db *Database) Close() error {
	return db.conn.Close(context.Background())
}

func Test() {
	config, err := pgx.ParseConfig(os.ExpandEnv("postgresql://root@localhost:26257?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}

	config.Database = "filestorage"

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	root := uuid.MustParse("ef4ebb18-b915-49fe-ba90-443aba9762d2")
	if err != nil {
		log.Fatal(err)
	}

	var file File

	err = crdbpgx.ExecuteTx(context.Background(), conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		f, err := getFile(conn, []string{"dev", "disks", "by-id"}, root)
		if err != nil {
			log.Fatal(err)
		}
		file = *f
		return nil

	})
	log.Print(file)
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
	return nil
}

/*
/path/to/new/file

get(path,root)
get(to,path.hash)
get(new,to.hash)

^^^^ if !exists then create

get(file,new.hash) if exists then error

create(file,new.hash)
*/

/*
/path/to/new/file

get([path,to,new])
if !exists then
	get([path,to])
	create(new,parent=to)
	if !exists then get(path)
		get([path])
		create(path,parent=db.root)

create(file,parent=new)

*/
func getHashOfFile(fileName, key []byte) string {
	hash := sha256.Sum256(append(fileName, key...))
	return fmt.Sprintf("\r%x", hash)
}

func (db *Database) NewFile(pathNames []string, key []byte) error {
	parentId := db.root

	err := func() error {
		if len(pathNames) > 1 {
			f, err := getFile(db.conn, pathNames[:len(pathNames)-1], db.root)
			if err != nil {
				if err == fileNotFound {
					err = db.NewFile(pathNames[:len(pathNames)-1], key)
					if err != nil {
						if err != fileExists {
							return err
						}
					}
					f, err = getFile(db.conn, pathNames[:len(pathNames)-1], db.root)
					if err != nil {
						return err
					}
					parentId = f.id
				}
				return err
			}

			parentId = f.id
		}
		return nil
	}()
	if err != nil {
		return err
	}

	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return newFile(context.Background(), tx, pathNames[len(pathNames)-1], getHashOfFile([]byte(pathNames[len(pathNames)-1]), key), parentId)
	})
}

func (db *Database) GetFile(pathNames []string) (*File, error) {
	return getFile(db.conn, pathNames, db.root)
}

func newFile(ctx context.Context, tx pgx.Tx, name string, hash string, parent uuid.UUID) error {
	log.Print(name, hash)
	sqlFormula := "INSERT INTO file_tree (encrypted_name,hash, parent_id) VALUES ($1, $2, $3);"
	log.Print(hash)
	if _, err := tx.Exec(ctx, sqlFormula, name, hash, parent); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return fileExists
		}
		return err
	}
	return nil
}

var fileNotFound error = errors.New("File not found")
var fileExists error = errors.New("File exists")

func getFile(conn *pgx.Conn, pathNames []string, parent uuid.UUID) (*File, error) {
	if len(pathNames) == 0 {
		return nil, errors.New("pathNames empty")
	}

	f := File{}

	//handle null uuid
	rows, err := conn.Query(context.Background(), "SELECT id, encrypted_name, hash, parent_id FROM file_tree WHERE encrypted_name = $1 AND parent_id = $2;", pathNames[0], parent)
	if err != nil {
		return nil, err
	}

	fileFound := false

	for rows.Next() {
		if err := rows.Scan(&f.id, &f.Name, &f.Hash, &f.parentId); err != nil {
			return nil, err
		}
		fileFound = true
	}

	if !fileFound {
		return nil, fileNotFound
	}

	rows.Close()

	if len(pathNames) == 1 {
		return &f, nil
	}

	return getFile(conn, pathNames[1:], f.id)
}
