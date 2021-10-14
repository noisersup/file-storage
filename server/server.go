package server

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

type Server struct {
	maxUpload int64
}

func InitServer() {
	s := Server{1024 << 20}

	handlers := []struct {
		regex   *regexp.Regexp
		methods []string
		handle  func(w http.ResponseWriter, r *http.Request, paths []string)
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
		http.NotFound(w, r)
	}

	log.Print("Waiting for connection...")
	log.Fatal(http.ListenAndServe(":8000", http.HandlerFunc(hanFunc)))
}
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, args []string) {
	log.Print("uploadFile")
	r.ParseMultipartForm(s.maxUpload)

	log.Print("FormFile")
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Print(err)
		return
	}

	defer file.Close()

	log.Printf("File: %+v\nSize: %+v MIME header: %+v", handler.Filename, handler.Size, handler.Header)

	newFilepath := "./files/" + args[0]

	//Create directory if not exists
	log.Print("Make dir")
	os.MkdirAll(newFilepath, os.ModePerm)

	newFilepath += "/" + handler.Filename + ".bin"

	log.Print("Open File")
	outfile, err := os.OpenFile(newFilepath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Print(err)
		return
	}

	defer outfile.Close()

	log.Print("encrypt")
	err = encrypt(file, outfile, []byte("2A462D4A614E645267556B5870327354"))
	//err = saveFile(file, outfile)
	if err != nil {
		log.Print(err)
		err := os.Remove(newFilepath)
		if err != nil {
			log.Print(err)
		}
		return
	}
}

func (s *Server) getFile(w http.ResponseWriter, r *http.Request, paths []string) {
	filePath := "./files/" + paths[0] + ".bin"

	err, status := serveFile(w, filePath)
	if err != nil {
		switch status {
		case http.StatusNotFound:
			errResponse(w, status, "File not found")
			break
		case http.StatusInternalServerError:
			serverError(w)
			break
		default:
			errResponse(w, status, "")
		}
	}
}

func serveFile(w http.ResponseWriter, path string) (error, int) {
	log.Print("Open file")
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

	if fi.Size() > 100*1000000 {
		w.Header().Set("Content-Disposition", "attachment; filename="+name)
	} else {
		w.Header().Set("Content-Disposition", "inline; filename="+name)
	}

	ctype := mime.TypeByExtension(filepath.Ext(name))
	w.Header().Set("Content-Type", ctype)

	log.Print("decrypt")
	err = decrypt(f, w, []byte("2A462D4A614E645267556B5870327354"))
	if err != nil {
		return err, http.StatusInternalServerError
	}
	log.Print("decryption ended")
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
