package server

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/noisersup/encryptedfs-api/auth"
	"github.com/noisersup/encryptedfs-api/database"
	l "github.com/noisersup/encryptedfs-api/logger"
)

// Server is a structure responsible for handling all http requests.
type Server struct {
	maxUpload int64 //TODO: implement maxuploads
	db        *database.Database
	auth      *auth.Auth
}

func InitServer(db *database.Database) error {
	a, err := auth.InitAuth(db)
	if err != nil {
		return err
	}
	s := Server{1024 << 20, db, a}

	//Handle requests
	handlers := []struct {
		regex      *regexp.Regexp
		methods    []string
		handle     func(w http.ResponseWriter, r *http.Request, paths []string, user string) // paths are regex matches (in this example they capture the storage server paths)
		authNeeded bool
	}{
		{regexp.MustCompile(`^/drive(?:/(.*[^/]))?$`), []string{"POST"}, s.uploadFile, true}, // /drive/path/of/target/directory ex. posting d.jpg with /drive/images/ will put to images/d.jpg and /drive/ will result with puting to root dir
		{regexp.MustCompile(`^/drive(?:/(.*[^/]))?$`), []string{"GET"}, s.getFile, true},
		{regexp.MustCompile(`^/drive/(.*[^/])$`), []string{"DELETE"}, s.deleteFile, true},
		{regexp.MustCompile(`^/signin$`), []string{"POST"}, s.signIn, false},
		{regexp.MustCompile(`^/signup$`), []string{"POST"}, s.signUp, false},
		{regexp.MustCompile(`^/refresh$`), []string{"POST"}, s.refresh, false},
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
					var authResp string
					if handler.authNeeded {
						authResp = a.Authorize(w, r)
						if authResp == "" {
							return
						}
					}
					handler.handle(w, r, match[1:], authResp)
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
func (s *Server) getFile(w http.ResponseWriter, r *http.Request, paths []string, user string) {
	l.Log(user)
	l.LogV("Fetching file...")

	path := database.PathToArr(paths[0])

	userRoot, err := s.db.GetRoot(user)
	if err != nil {
		l.Err("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	if len(path) == 1 && path[0] == "" {
		l.LogV("Listing root directory")
		files, err := s.db.ListDirectory(userRoot)
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
	f, err := s.db.GetFile(path, userRoot)
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

	key, err := s.db.GetKey(user)
	if err != nil {
		l.Err("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	l.LogV("Serving file")
	err, status := serveFile(w, filePath, f.Name, key)
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
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, args []string, user string) {
	l.LogV("Uploading file...")
	reader, err := r.MultipartReader()
	if err != nil {
		l.Err(err.Error())
		errResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	key, err := s.db.GetKey(user)
	if err != nil {
		l.Err("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	userRoot, err := s.db.GetRoot(user)
	if err != nil {
		l.Err("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	err = encryptMultipart(reader, args[0], key, s.db, userRoot)
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
func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request, paths []string, user string) {
	l.LogV("Deleting file...")

	userRoot, err := s.db.GetRoot(user)
	if err != nil {
		l.Err("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	err = s.db.DeleteFile(database.PathToArr(paths[0]), userRoot)
	if err != nil {
		l.Err(err.Error())
		return
	}
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) signUp(w http.ResponseWriter, r *http.Request, _ []string, _ string) {
	var credentials Credentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	status := s.auth.Signup(credentials.Username, credentials.Password)
	log.Print(status)
	w.WriteHeader(status)
}

func (s *Server) signIn(w http.ResponseWriter, r *http.Request, _ []string, _ string) {
	var credentials Credentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionToken, status := s.auth.Signin(credentials.Username, credentials.Password)
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(120 * time.Second),
	})
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request, _ []string, _ string) {
	s.auth.Refresh(w, r)
}

// serveFile decrypts file on provided path and writes it's to ResponseWriter
// Returns error and status code
func serveFile(w http.ResponseWriter, path, name string, key []byte) (error, int) {
	f, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return err, http.StatusNotFound
	}
	defer f.Close()

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

	err = decrypt(f, w, key)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, 200
}

func writeResponse(w http.ResponseWriter, response interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode) // TODO: http: superfluous response.WriteHeader (server.go:299)
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
