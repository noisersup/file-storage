package main

import (
	"flag"

	l "github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server"
	"github.com/noisersup/encryptedfs-api/server/dirs/database"
)

func main() {
	v := flag.Bool("v", false, "verbose output")
	flag.Parse()

	l.Verbose = *v

	dbPayload := "postgresql://root@localhost:26257?sslmode=disable"
	dbName := "filestorage"

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
