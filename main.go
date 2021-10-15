package main

import (
	"github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server"
)

func main() {
	l := logger.Logger{}
	server.InitServer(&l)
}
