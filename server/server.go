package server

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	l "github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server/dirs/database"
)

// Server is a structure responsible for handling all http requests.
type Server struct {
	maxUpload int64 //TODO: implement maxuploads
	db        *database.Database
}

func InitServer(db *database.Database) error {
	s := Server{1024 << 20, db}

	//Handle requests
	handlers := []struct {
		regex   *regexp.Regexp
		methods []string
		handle  func(w http.ResponseWriter, r *http.Request, paths []string) // paths are regex matches (in this example they capture the storage server paths)
	}{
		{regexp.MustCompile(`^/drive(?:/(.*[^/]))?$`), []string{"POST"}, s.uploadFile}, // /drive/path/of/target/directory ex. posting d.jpg with /drive/images/ will put to images/d.jpg and /drive/ will result with puting to root dir
		{regexp.MustCompile(`^/drive(?:/(.*[^/]))?$`), []string{"GET"}, s.getFile},
		{regexp.MustCompile(`^/drive/(.*[^/])$`), []string{"DELETE"}, s.deleteFile},
	}

	hanFunc := func(w http.ResponseWriter, r *http.Request) {
		l.Log("%s %s", r.Method, r.URL.Path)
		for _, handler := range handlers {
			match := handler.regex.FindStringSubmatch(r.URL.Path)
			if match == nil {
				continue
			}
			for _, allowed := range handler.methods {
				if r.Method == allowed {
					handler.handle(w, r, match[1:])
					return
				}
			}
		}
		l.Warn("Cannot handle request\n Request: %v", r)
		http.NotFound(w, r)
	}

	port := 8000 //TODO: add custom port

	l.Log("Waiting for connection on port: :%d...", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(hanFunc))
}

//
//
//

// Handler function for GET requests.
// Decrypts file and send it in chunks to user
func (s *Server) getFile(w http.ResponseWriter, r *http.Request, paths []string) {
	l.LogV("Fetching file...")

	path := pathToArr(paths[0])

	if len(path) == 1 && path[0] == "" {
		l.LogV("Listing root directory")
		files, err := s.db.ListDirectory()
		if err != nil {
			if err == database.FileNotFound {
				errResponse(w, http.StatusNotFound, err.Error())
			}
			l.Err(err.Error())
			return
		}
		outFiles := []ListedFile{}
		for _, f := range files {
			outFiles = append(outFiles, ListedFile{f.Name, f.IsDirectory})
		}
		writeResponse(w, ListFilesResponse{outFiles, ""}, http.StatusOK)
		return
	}

	l.LogV("Getting file")
	f, err := s.db.GetFile(path)
	if err != nil {
		errResponse(w, http.StatusNotFound, "File not found")
		l.Err(err.Error())
		return
	}
	if f.IsDirectory {
		l.LogV("Listing directory")
		files, err := s.db.ListDirectory(f.Id)
		if err != nil {
			l.Err(err.Error())
			return
		}
		outFiles := []ListedFile{}
		for _, f := range files {
			outFiles = append(outFiles, ListedFile{f.Name, f.IsDirectory})
		}
		writeResponse(w, ListFilesResponse{outFiles, ""}, http.StatusOK)
		return
	}

	var filePath string
	if f.Duplicate == 0 {
		filePath = fmt.Sprintf("./files/%s", f.Hash)
	} else {
		filePath = fmt.Sprintf("./files/%s%d", f.Hash, f.Duplicate)
	}

	l.LogV("Serving file")
	err, status := serveFile(w, filePath, f.Name)
	if err != nil {
		switch status {
		case http.StatusNotFound:
			errResponse(w, status, "File not found")
			l.Err("File %s not found [error: %s]", filePath, err.Error())
			break
		case http.StatusInternalServerError:
			serverError(w)
			l.Err("getFile Internal error: %s", err.Error())
			break
		default:
			serverError(w)
			l.Warn("getFile Undefined error: %s", err.Error())
		}
	}
	l.LogV("File transfer done!")
}

// Handler function for POST requests.
// Encrypts multipart file and store it in provided by user location
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, args []string) {
	l.LogV("Uploading file...")
	reader, err := r.MultipartReader()
	if err != nil {
		l.Err(err.Error())
		errResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	err = encryptMultipart(reader, args[0], []byte("2A462D4A614E645267556B5870327354"), s.db)
	if err != nil {
		l.Err(err.Error())
		if err == database.FileExists {
			errResponse(w, http.StatusNotFound, "File not found")
		}
		errResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}
	l.LogV("File uploaded!")
}

// Handler function for DELETE requests.
// Finds file on provided by user location
// and removes it
func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request, paths []string) {
	l.LogV("Deleting file...")

	err := s.db.DeleteFile(pathToArr(paths[0]))
	if err != nil {
		l.Err(err.Error())
		return
	}
}

// serveFile decrypts file on provided path and writes it's to ResponseWriter
// Returns error and status code
func serveFile(w http.ResponseWriter, path, name string) (error, int) {
	f, err := os.OpenFile(path, os.O_RDWR, 0777)
	defer f.Close()
	if err != nil {
		return err, http.StatusNotFound
	}

	fi, err := f.Stat()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	//name := fi.Name()[:len(fi.Name())-4]

	// Dont show file on web if it's bigger than ~100MB
	if fi.Size() > 100*1000000 {
		w.Header().Set("Content-Disposition", "attachment; filename="+name)
	} else {
		w.Header().Set("Content-Disposition", "inline; filename="+name)
	}

	ctype := mime.TypeByExtension(filepath.Ext(name))
	w.Header().Set("Content-Type", ctype)

	err = decrypt(f, w, []byte("2A462D4A614E645267556B5870327354"))
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, 200
}

func writeResponse(w http.ResponseWriter, response interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		l.Err("JSON encoding error: %s", err)
	}
}

func serverError(w http.ResponseWriter) {
	errResponse(w, http.StatusInternalServerError, "Server error")
}

func errResponse(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Header().Del("Content-Disposition")
	w.Header().Del("Content-Type")
	writeResponse(w, ErrResponse{msg}, status)
}
