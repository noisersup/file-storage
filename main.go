package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/noisersup/encryptedfs-api/database"
	l "github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server"
)

func main() {
	v := flag.Bool("v", false, "verbose output")
	flag.Parse()

	dbName := getEnv("DB_NAME", "filestorage")
	user := getEnv("DB_USER", "root")
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "26257")

	l.Verbose = *v

	dbPayload := fmt.Sprintf("postgresql://%s@%s:%s?sslmode=disable", user, host, port)

	l.LogV("Connecting to database %s with payload: %s", dbName, dbPayload)

	db, err := database.ConnectDB(dbPayload, dbName, "ef4ebb18-b915-49fe-ba90-443aba9762d2")
	if err != nil {
		l.Fatal(err.Error())
	}
	defer db.Close()

	if err = server.InitServer(db); err != nil {
		l.Fatal(err.Error())
	}
}

func getEnv(envName, defValue string) string {
	env := os.Getenv(envName)
	if env == "" {
		return defValue
	}
	return env
}
