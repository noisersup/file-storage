package server

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"regexp"

	"github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/server/dirs/database"
)

func encryptBytes(input, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(input))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], input)

	return ciphertext, nil
}

func decryptBytes(input, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	if len(input) < aes.BlockSize {
		return nil, errors.New("input too short")
	}

	iv := input[:aes.BlockSize]
	input = input[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(input, input)

	return input, nil
}

func getHashOfFile(fileName, key []byte) string {
	hash := sha256.Sum256(append(fileName, key...))
	return fmt.Sprintf("\r%x", hash)
}

func pathToArr(path string) []string {
	return regexp.MustCompile("([^/]*)").FindAllString(path, -1)
}

// Encrypts a file from multipart reader and stores it in provided directory
func encryptMultipart(r *multipart.Reader, dir string, key []byte, db *database.Database) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	//buf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf := make([]byte, 1024)
	stream := cipher.NewCTR(block, iv)

	part, err := r.NextPart()
	if err != nil {
		log.Print(err)
	}
	partRead := true

	//newFilepath := "./files/" + dir

	hash := getHashOfFile([]byte(part.FileName()), key)

	err = db.NewFile(append(pathToArr(dir), part.FileName()), key)
	if err != nil {
		return err
	}

	//Create directory if not exists
	//os.MkdirAll(newFilepath, os.ModePerm)

	//newFilepath += "/" + handler.Filename + ".bin"
	//newFilepath += "/" + part.FileName() + ".bin"
	//log.Print(newFilepath)

	//overrides old file if exists
	//rmFile(newFilepath)

	outFile, err := os.OpenFile("./files/"+hash, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	l := logger.Logger{}

	defer outFile.Close()
	partRead = true // part of file was read once (for metadata purposes)

	for {
		log.Print("\n")
		log.Print(err)
		if !partRead {
			l.Log("NextPart")
			part, err = r.NextPart()
			l.Log("Part found")
		}
		partRead = false
		if err == io.EOF {
			break
		}

		if err != nil {
			//rmFile(newFilepath)
			return fmt.Errorf("Read %d bytes: %v", len(buf), err)
		}

		l.Log("io.Copy")
		for {
			fmt.Print(".")
			n, err := part.Read(buf)
			if n > 0 {
				stream.XORKeyStream(buf, buf[:n])
				outFile.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
	}
	l.Log("outFile.Write(iv)")
	outFile.Write(iv)
	l.Log("written iv")

	return nil
}

// Decrypts a file from argument, creates chunks od it and send them to writer
func decrypt(file *os.File, part io.Writer, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	finfo, err := file.Stat()
	if err != nil {
		return err
	}

	iv := make([]byte, block.BlockSize())

	length := finfo.Size() - int64(len(iv))

	_, err = file.ReadAt(iv, length)
	if err != nil {
		return err
	}

	d := logger.CreateDots(1000)

	buf := make([]byte, 1024)
	stream := cipher.NewCTR(block, iv)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			d.PrintDots()
			// IV bytes don't belong the original message
			if n > int(length) {
				n = int(length)
			}

			length -= int64(n)
			stream.XORKeyStream(buf, buf[:n])

			if _, err = io.Copy(part, bytes.NewReader(buf[:n])); err != nil {
				return err
			}
			if err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func saveFile(infile multipart.File, outFile *os.File) error {
	buf := make([]byte, 1024)
	for {
		n, err := infile.Read(buf)
		if n > 0 {
			outFile.Write(buf[:n])
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("Read %d bytes: %v", n, err)
		}
	}
	return nil
}

func rmFile(path string) {
	os.Remove(path)
}
