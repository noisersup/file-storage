/*
	Database user authentication operations
*/
package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/jackc/pgx/v4"
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

	sqlFormula := "INSERT INTO users (username, password, key) VALUES ($1,$2,$3);"
	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), sqlFormula, username, hashedPassword, keyB64); err != nil {
			return err
		}
		return nil
	})
}

// Returns bcrypted password of provided user
func (db *Database) GetPasswordOfUser(username string) (string, error) {
	var expectedPassword string
	err := db.conn.QueryRow(context.Background(), "SELECT password FROM users WHERE username=$1;", username).Scan(&expectedPassword)
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
	err := db.conn.QueryRow(context.Background(), sqlFormula, username).Scan(&b64Decoded)
	if err != nil {
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(b64Decoded)
	if err != nil {
		return nil, err
	}
	return key, nil
}
