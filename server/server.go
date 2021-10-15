package server

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/noisersup/encryptedfs-api/logger"
)

// Server is a structure responsible for handling all http requests.
type Server struct {
	maxUpload int64 //TODO: implement maxuploads
	l         *logger.Logger
}

func InitServer(l *logger.Logger) {
	s := Server{1024 << 20, l}

	//Handle requests
	handlers := []struct {
		regex   *regexp.Regexp
		methods []string
		handle  func(w http.ResponseWriter, r *http.Request, paths []string) // paths are regex matches (in this example they capture the storage server paths)
	}{
		{regexp.MustCompile(`^/drive(?:/(.*[^/]))?$`), []string{"POST"}, s.uploadFile}, // /drive/path/of/target/directory ex. posting d.jpg with /drive/images/ will put to images/d.jpg and /drive/ will result with puting to root dir
		{regexp.MustCompile(`^/drive/(.*[^/])$`), []string{"GET"}, s.getFile},
	}

	hanFunc := func(w http.ResponseWriter, r *http.Request) {
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
		l.SWarn("hanFunc", "Cannot handle request\n Request: %v", r)
		http.NotFound(w, r)
	}

	port := 8000 //TODO: add custom port

	l.Log("Waiting for connection on port: :%d...", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(hanFunc))
	if err != nil {
		l.SFatal("InitServer", err.Error())
	}
}

// Handler function for POST requests.
// Encrypts multipart file and store it in provided by user location
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, args []string) {
	s.l.Log("Uploading file...")
	reader, err := r.MultipartReader()
	if err != nil {
		log.Print(err)
		return
	}

	err = encryptMultipart(reader, args[0], []byte("2A462D4A614E645267556B5870327354"))
	if err != nil {
		log.Print(err)
		return
	}
	s.l.Log("File uploaded!")
}

// Handler function for GET requests.
// Decrypts file and send it in chunks to user
func (s *Server) getFile(w http.ResponseWriter, r *http.Request, paths []string) {
	s.l.Log("Fetching file...")
	filePath := fmt.Sprintf("./files/%s.bin", paths[0])

	err, status := serveFile(w, filePath)
	if err != nil {
		switch status {
		case http.StatusNotFound:
			errResponse(w, status, "File not found")
			s.l.Log("File %s not found [error: %s]", filePath, err.Error())
			break
		case http.StatusInternalServerError:
			serverError(w)
			s.l.SErr("getFile", "Internal error: %s", err.Error())
			break
		default:
			serverError(w)
			s.l.SWarn("getFile", "Undefined error: %s", err.Error())
		}
	}
	s.l.Log("File transfer done!")
}

// serveFile decrypts file on provided path and writes it's to ResponseWriter
// Returns error and status code
func serveFile(w http.ResponseWriter, path string) (error, int) {
	f, err := os.OpenFile(path, os.O_RDWR, 0777)
	defer f.Close()
	if err != nil {
		return err, http.StatusNotFound
	}

	fi, err := f.Stat()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	name := fi.Name()[:len(fi.Name())-4]

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

func serverError(w http.ResponseWriter) {
	errResponse(w, http.StatusInternalServerError, "Server error")
}

func errResponse(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Header().Del("Content-Disposition")
	w.Header().Del("Content-Type")
	w.Write([]byte("error: " + msg))
}

// file tree

//user auth

// generating key
