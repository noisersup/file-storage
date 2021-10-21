package main

import (
	"log"

	"github.com/noisersup/encryptedfs-api/server/dirs/database"
)

//"github.com/noisersup/encryptedfs-api/logger"
//"github.com/noisersup/encryptedfs-api/server"

func main() {
	//l := logger.Logger{}
	//server.InitServer(&l)
	//database.Test()
	db := database.ConnectDB("postgresql://root@localhost:26257?sslmode=disable", "filestorage", "ef4ebb18-b915-49fe-ba90-443aba9762d2")
	defer db.Close()

	f, err := db.GetFile([]string{"dev", "disk", "by-id"})
	//err := db.NewFile("dev")
	if err != nil {
		log.Fatal(err)
	}
	log.Print(f)
}
