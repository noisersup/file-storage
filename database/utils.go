package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/models"
)

// Converts path 			("/a/b/c/d.conf")
// to array with filenames {"a","b","c","d.conf"}
func PathToArr(path string) []string {
	if len(path) == 0 {
		return []string{}
	}
	return regexp.MustCompile("([^/]*)").FindAllString(path, -1)
}

// returns sha256 checksum of filename and user access key
func getHashOfFile(fileName, key []byte) string {
	hash := sha256.Sum256(append(fileName, key...))
	return fmt.Sprintf("%x", hash)
}

/*

	File database related errors

*/
var FileNotFound error = errors.New("File not found")
var FileExists error = errors.New("File exists")

/*

	File database functions

*/

// Get metadata of specified file from database
func getFile(pool *pgxpool.Pool, pathNames []string, parent uuid.UUID) (*models.File, error) {
	if len(pathNames) == 0 {
		return nil, errors.New("pathNames empty")
	}

	f := models.File{}

	// Get metadata of first file from pathNames
	sqlQuery := "SELECT id, encrypted_name, hash, parent_id,duplicate, is_directory FROM file_tree WHERE encrypted_name = $1 AND parent_id = $2;"
	rows, err := pool.Query(context.Background(), sqlQuery, pathNames[0], parent)
	if err != nil {
		rows.Close()
		return nil, err
	}

	fileFound := false

	for rows.Next() {
		if err := rows.Scan(&f.Id, &f.Name, &f.Hash, &f.ParentId, &f.Duplicate, &f.IsDirectory); err != nil {
			return nil, err
		}
		fileFound = true
	}

	rows.Close()

	if !fileFound {
		return nil, FileNotFound
	}

	// Closes recursion
	// if it's last file in path returns it
	if len(pathNames) == 1 {
		return &f, nil
	}

	return getFile(pool, pathNames[1:], f.Id)
}

// deletes file entry from database and removes it from disk
func deleteFile(pool *pgxpool.Pool, ctx context.Context, tx pgx.Tx, pathNames []string, root uuid.UUID) error {
	f, err := getFile(pool, pathNames, root)
	if err != nil {
		return err
	}

	if f.IsDirectory {
		childs, _ := listDirectory(pool, f.Id)
		for _, ch := range childs {
			err = deleteFile(pool, ctx, tx, append(pathNames, ch.Name), root)
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					logger.Warn(err.Error())
				} else {
					return err
				}
			}
		}
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

	if f.IsDirectory {
		return nil
	}

	return os.Remove(filePath)
}

// Creates empty file
func newRootEntry(ctx context.Context, tx pgx.Tx) error {
	sqlFormula := "INSERT INTO file_tree (encrypted_name) VALUES ($1) RETURNING id;"
	if _, err := tx.Exec(ctx, sqlFormula, "root"); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return FileExists
		}
		return err
	}
	return nil
}

// Creates new file entry in database
func newFile(ctx context.Context, tx pgx.Tx, name string, hash string, parent uuid.UUID, duplicate int, isDirectory bool) error {
	log.Print("newFile: ", name)
	if len(name) > 255 {
		return errors.New("Filename too big")
	}

	sqlFormula := "INSERT INTO file_tree (encrypted_name, hash, parent_id, duplicate, is_directory) VALUES ($1, $2, $3, $4, $5);"
	if _, err := tx.Exec(ctx, sqlFormula, name, hash, parent, duplicate, isDirectory); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return FileExists
		}
		return err
	}
	return nil
}

// List directory with specified id
func listDirectory(pool *pgxpool.Pool, id uuid.UUID) ([]models.File, error) {
	files := []models.File{}
	sqlFormula := "SELECT id, encrypted_name, hash, parent_id,duplicate, is_directory FROM file_tree WHERE parent_id = $1 ;"
	rows, err := pool.Query(context.Background(), sqlFormula, id)

	if err != nil {
		rows.Close()
		return nil, err
	}

	for rows.Next() {
		f := models.File{}
		if err := rows.Scan(&f.Id, &f.Name, &f.Hash, &f.ParentId, &f.Duplicate, &f.IsDirectory); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	rows.Close()

	if len(files) == 0 {
		return nil, FileNotFound
	}

	return files, nil
}
