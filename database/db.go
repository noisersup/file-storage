package database

import (
	"context"
	"fmt"
	"os"

	l "github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/models"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool // database connection
	root uuid.UUID     // id of the root directory in database
}

// Connects to database with provided data
// and returns database object
func ConnectDB(uri, database string, root string) (*Database, error) {
	config, err := pgxpool.ParseConfig(os.ExpandEnv(uri))
	if err != nil {
		return nil, err
	}

	config.ConnConfig.Database = database

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	db := Database{pool: pool}

	err = db.fetchRoot()
	if err != nil {
		return nil, err
	}

	return &db, nil
}

// Close database connection
// ( conn.Close alias )
func (db *Database) Close() {
	l.Log("Closing database...")
	db.pool.Close()
	l.Log("All database connections closed.")
}

// Fetch root variable in database object from file_Tree_config database
// If not present - creates root entry and insert its id to config db
func (db *Database) fetchRoot() error {
	row := db.pool.QueryRow(context.Background(), "SELECT root FROM file_tree_config;")
	var root uuid.UUID
	err := row.Scan(&root)
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		l.Log("root entry not found, creating one...")
		return db.setRoot()
	}

	db.root = root
	l.LogV("root: %s", db.root.String())
	return nil
}

func (db *Database) setRoot() error {
	sqlFormula := "INSERT INTO file_tree (encrypted_name) VALUES ($1) RETURNING id;"

	var id uuid.UUID

	l.LogV("Inserting root to file_tree")
	row := db.pool.QueryRow(context.Background(), sqlFormula, "root")
	err := row.Scan(&id)
	if err != nil {
		return err
	}
	l.LogV("SUCCESS!")

	l.LogV("Removing all from file_tree_config")
	r, err := db.pool.Query(context.Background(), "DELETE FROM file_tree_config WHERE TRUE;")
	r.Close()
	if err != nil {
		return err
	}

	l.LogV("Inserting root to file_tree_config")
	r, err = db.pool.Query(context.Background(), "INSERT INTO file_tree_config (root) VALUES ($1)", id)
	r.Close()
	if err != nil {
		return err
	}

	l.LogV("SUCCESS!")
	return nil
}

// Adds file entry to database
func (db *Database) NewFile(pathNames []string, key []byte, duplicate int, isDirectory bool, userRoot uuid.UUID) error {
	if len(pathNames) == 0 {
		return fmt.Errorf("NewFile: no path provided")
	}
	parentId := userRoot
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
		f, err := getFile(db.pool, pathNames[:len(pathNames)-1], userRoot)
		if err != nil {
			// if parent doesn't exist create it
			if err == FileNotFound {
				err = db.NewFile(pathNames[:len(pathNames)-1], key, 0, true, userRoot)
				if err != nil {
					if err != FileExists {
						return err
					}
				}

				// we're sure that the parent of file exists (i guess...)
				// now we can get it's database id to link our file to it
				f, err = getFile(db.pool, pathNames[:len(pathNames)-1], userRoot)
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
	return crdbpgx.ExecuteTx(context.Background(), db.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return newFile(context.Background(), tx, pathNames[len(pathNames)-1], getHashOfFile([]byte(pathNames[len(pathNames)-1]), key), parentId, duplicate, isDirectory)
	})
}

/*
	Gets file id placed on given path

	pathNames array contains filenames of path from the first to last
	ex: /a/b/c/d.conf == {"a","b","c","d.conf"}
	For the best experience use database.PathToArr function
*/
func (db *Database) GetFile(pathNames []string, userRoot uuid.UUID) (*models.File, error) {
	return getFile(db.pool, pathNames, userRoot)
}

// Lists directory with specified id
// (Without arguments it will use root directory id)
// WARNING!!! Remember to not use this function without any arguments as an output for an user!!!

func (db *Database) ListDirectory(id ...uuid.UUID) ([]models.File, error) {
	var dirId uuid.UUID
	if len(id) == 0 {
		dirId = db.root
	} else {
		dirId = id[0]
	}
	return listDirectory(db.pool, dirId)
}

func (db *Database) DeleteFile(pathNames []string, userRoot uuid.UUID) error {
	return crdbpgx.ExecuteTx(context.Background(), db.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return deleteFile(db.pool, context.Background(), tx, pathNames, userRoot)
	})
}
