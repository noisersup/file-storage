package database

import (
	"context"
	"fmt"
	"os"

	l "github.com/noisersup/encryptedfs-api/logger"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type Database struct {
	conn *pgx.Conn // database connection
	root uuid.UUID // id of the root directory in database
}

type File struct {
	Id          uuid.UUID
	Name        string
	Hash        string
	parentId    uuid.UUID
	Duplicate   int
	IsDirectory bool
}

// Connects to database with provided data
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

// Close database connection
// ( conn.Close alias )
func (db *Database) Close() error {
	l.Log("Closing database...")
	return db.conn.Close(context.Background())
}

// Fetch root variable in database object from file_Tree_config database
// If not present - creates root entry and insert its id to config db
func (db *Database) fetchRoot() error {
	row := db.conn.QueryRow(context.Background(), "SELECT root FROM file_tree_config;")
	var root uuid.UUID
	err := row.Scan(&root)
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		return err
		//TODO: setRoot
	}

	db.root = root
	l.LogV("root: %s", db.root.String())
	return nil
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

/*
	Gets file id placed on given path

	pathNames array contains filenames of path from the first to last
	ex: /a/b/c/d.conf == {"a","b","c","d.conf"}
	For the best experience use database.PathToArr function
*/
func (db *Database) GetFile(pathNames []string) (*File, error) {
	return getFile(db.conn, pathNames, db.root)
}

// Lists directory with specified id
// (Without arguments it will use root directory id)
func (db *Database) ListDirectory(id ...uuid.UUID) ([]File, error) {
	var dirId uuid.UUID
	if len(id) == 0 {
		dirId = db.root
	} else {
		dirId = id[0]
	}
	return listDirectory(db.conn, dirId)
}

func (db *Database) DeleteFile(pathNames []string) error {
	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return deleteFile(db.conn, context.Background(), tx, pathNames, db.root)
	})
}
