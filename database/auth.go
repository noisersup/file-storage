/*
	Database user authentication operations
*/
package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
	l "github.com/noisersup/encryptedfs-api/logger"
)

/*
	Registers new user
	!!! remember to provide bcrypt hash as password argument !!!
*/
func (db *Database) NewUser(username, hashedPassword string) error {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return err
	}
	keyB64 := base64.StdEncoding.EncodeToString(key)

	var id uuid.UUID

	l.LogV("Inserting into users...")
	sqlFormula := "INSERT INTO users (username, password, key) VALUES ($1,$2,$3) RETURNING id;"
	err = db.pool.QueryRow(context.Background(), sqlFormula, username, hashedPassword, keyB64).Scan(&id)
	if err != nil {
		return err
	}
	l.LogV("SUCCESS!")

	l.LogV("Inserting root %s into files...", id.String())
	r, err := db.pool.Query(context.Background(), "INSERT INTO file_tree (id,parent_id) VALUES ($1,$2);", id, db.root)
	r.Close()

	if err != nil {
		//TODO: RemoveUser
		return err
	}

	l.LogV("SUCCESS!")
	return nil
}

//func (db *Database) RemoveUser(username string) {
//TODO remove user's db entry, his file entries and his files
//}

// Returns bcrypted password of provided user
func (db *Database) GetPasswordOfUser(username string) (string, error) {
	var expectedPassword string
	err := db.pool.QueryRow(context.Background(), "SELECT password FROM users WHERE username=$1;", username).Scan(&expectedPassword)
	if err != nil {
		return "", err
	}
	return expectedPassword, err
}

/*
gets hashing key of user
*/
func (db *Database) GetKey(username string) ([]byte, error) {
	var b64Decoded string
	sqlFormula := "SELECT key FROM users WHERE username=$1;"
	err := db.pool.QueryRow(context.Background(), sqlFormula, username).Scan(&b64Decoded)
	if err != nil {
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(b64Decoded)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (db *Database) GetRoot(username string) (uuid.UUID, error) {
	var root uuid.UUID
	err := db.pool.QueryRow(context.Background(), "SELECT id FROM users where username=$1;", username).Scan(&root)
	if err != nil {
		return uuid.UUID{}, err
	}
	return root, nil
}
