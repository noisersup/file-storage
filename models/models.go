package models

import "github.com/google/uuid"

type File struct {
	Id          uuid.UUID
	Name        string
	Hash        string
	ParentId    uuid.UUID
	Duplicate   int
	IsDirectory bool
}

type Database interface {
	Close() error
	NewFile(pathNames []string, key []byte, duplicate int, isDirectory bool, userRoot uuid.UUID) error
	GetFile(pathNames []string, userRoot uuid.UUID) (*File, error)
	ListDirectory(id ...uuid.UUID) ([]File, error)
	DeleteFile(pathNames []string, userRoot uuid.UUID) error
	NewUser(username, hashedPassword string) error
	GetPasswordOfUser(username string) (string, error)
	GetKey(username string) ([]byte, error)
	GetRoot(username string) (uuid.UUID, error)
}
