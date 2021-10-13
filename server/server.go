package server

import (
	"log"
	"net/http"
	"os"
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
	r.ParseMultipartForm(s.maxUpload)

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Print(err)
		return
	}

	defer file.Close()

	log.Printf("File: %+v\nSize: %+v MIME header: %+v", handler.Filename, handler.Size, handler.Header)

	newFilepath := "./files/" + args[0]

	//Create directory if not exists
	os.MkdirAll(newFilepath, os.ModePerm)

	newFilepath += "/" + handler.Filename + ".bin"

	outfile, err := os.OpenFile(newFilepath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Print(err)
		return
	}

	defer outfile.Close()

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
	f, err := os.OpenFile(filePath, os.O_RDWR, 0777)
	if err != nil {
		log.Print(err)
		return
	}

	outFile, err := os.OpenFile(paths[0], os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Print(err)
		return
	}

	err = decrypt(f, outFile, []byte("2A462D4A614E645267556B5870327354"))
	if err != nil {
		log.Print(err)
		return
	}
	http.ServeFile(w, r, paths[0])
	//Decrypt and servefile
}

// file tree

//user auth

// generating key
