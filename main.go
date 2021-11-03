package main

import (
	"log"

	"github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server"
	"github.com/noisersup/encryptedfs-api/server/dirs/database"
)

func main() {
	l := logger.Logger{}

	db, err := database.ConnectDB("postgresql://root@localhost:26257?sslmode=disable", "filestorage", "ef4ebb18-b915-49fe-ba90-443aba9762d2")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server.InitServer(&l, db)
}
