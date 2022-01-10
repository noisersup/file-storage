/*
	Database user authentication operations
*/
package database

import (
	"context"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/jackc/pgx/v4"
)

/*
	Registers new user
	!!! remember to provide bcrypt hash as password argument !!!
*/
func (db *Database) NewUser(username, hashedPassword string) error {
	sqlFormula := "INSERT INTO users (username, password) VALUES ($1,$2);"
	return crdbpgx.ExecuteTx(context.Background(), db.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), sqlFormula, username, hashedPassword); err != nil {
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
