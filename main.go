package main

import (
	"github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server"
	"github.com/noisersup/encryptedfs-api/server/dirs/database"
)

//"github.com/noisersup/encryptedfs-api/logger"
//"github.com/noisersup/encryptedfs-api/server"

func main() {
	l := logger.Logger{}
	//database.Test()
	db := database.ConnectDB("postgresql://root@localhost:26257?sslmode=disable", "filestorage", "ef4ebb18-b915-49fe-ba90-443aba9762d2")
	defer db.Close()

	server.InitServer(&l, db)

	//f, err := db.GetFile([]string{"dev", "disks", "by-id"})
	//err := db.NewFile([]string{"ledu", "michu", "leduchosky"}, []byte("2A462D4A614E645267556B5870327354"))
	/*if err != nil {
		log.Fatal(err)
	}*/
}
