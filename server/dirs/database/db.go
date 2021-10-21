package database

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type Database struct {
	conn *pgx.Conn
	root uuid.UUID
}

type File struct {
	id            uuid.UUID
	encryptedName string
	parentId      uuid.UUID
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
	log.Print(root)

	db.root = root
	return nil
}

func (db *Database) NewFile(encryptedName string) error {
	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return newFile(context.Background(), tx, encryptedName, db.root)
	})
}

func (db *Database) GetFile(pathNames []string) (*File, error) {
	return getFile(db.conn, pathNames, db.root)
}

func newFile(ctx context.Context, tx pgx.Tx, encryptedName string, parent uuid.UUID) error {
	//sqlFormula := "INSERT INTO file_tree (encrypted_name, parent_id) VALUES ($1, $2);"
	sqlFormula := "INSERT INTO file_tree (encrypted_name, parent_id) VALUES ($1, $2);"
	if _, err := tx.Exec(ctx, sqlFormula, encryptedName, parent); err != nil {
		return err
	}
	return nil
}

func getFile(conn *pgx.Conn, pathNames []string, parent uuid.UUID) (*File, error) {
	if len(pathNames) == 0 {
		return nil, errors.New("pathNames empty")
	}

	f := File{}

	//handle null uuid
	rows, err := conn.Query(context.Background(), "SELECT id, encrypted_name, parent_id FROM file_tree WHERE encrypted_name = $1 AND parent_id = $2;", pathNames[0], parent)
	if err != nil {
		return nil, err
	}

	fileFound := false

	for rows.Next() {
		if err := rows.Scan(&f.id, &f.encryptedName, &f.parentId); err != nil {
			return nil, err
		}
		fileFound = true
	}
	if !fileFound {
		return nil, errors.New("File " + pathNames[0] + " not found")
	}

	rows.Close()

	if len(pathNames) == 1 {
		return &f, nil
	}

	return getFile(conn, pathNames[1:], f.id)
}
