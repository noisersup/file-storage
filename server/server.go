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
		handle  func(w http.ResponseWriter, r *http.Request, args []string)
	}{
		//{regexp.MustCompile(`^/drive$`), []string{"GET"}, s.getTree}, //tree = encrypted map stored in database map["filename"] = encrypted filename
		{regexp.MustCompile(`^/drive$`), []string{"POST"}, s.uploadFile},
		{regexp.MustCompile(`^/drive/([^/]+)$`), []string{"GET"}, s.getFile},
	}

	hanFunc := func(w http.ResponseWriter, r *http.Request) {
		for _, handler := range handlers {
			match := handler.regex.FindStringSubmatch(r.URL.Path)
			if match == nil {
				continue
			}
			for _, allowed := range handler.methods {
				if r.Method == allowed {
					handler.handle(w, r, match)
					return
				}
			}
		}
		http.NotFound(w, r)
	}

	log.Print("Waiting for connection...")
	log.Fatal(http.ListenAndServe(":8000", http.HandlerFunc(hanFunc)))
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, _ []string) {
	r.ParseMultipartForm(s.maxUpload)

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Print(err)
		return
	}

	defer file.Close()

	log.Printf("File: %+v\nSize: %+v MIME header: %+v", handler.Filename, handler.Size, handler.Header)

	newFilepath := "./files/" + handler.Filename + ".bin"

	outfile, err := os.OpenFile(newFilepath, os.O_RDWR|os.O_CREATE, 0777)
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

func (s *Server) getFile(w http.ResponseWriter, r *http.Request, args []string) {
	filePath := "./files/" + args[1] + ".bin"
	f, err := os.OpenFile(filePath, os.O_RDWR, 0777)
	if err != nil {
		log.Print(err)
		return
	}

	outFile, err := os.OpenFile(args[1], os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Print(err)
		return
	}

	err = decrypt(f, outFile, []byte("2A462D4A614E645267556B5870327354"))
	if err != nil {
		log.Print(err)
		return
	}
	http.ServeFile(w, r, args[1])
	//Decrypt and servefile
}

// file tree

//user auth

// generating key
