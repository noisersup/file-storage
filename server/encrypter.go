package server

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
)

// Encrypts a file from multipart reader and stores it in provided directory
func encrypt(r *multipart.Reader, dir string, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	stream := cipher.NewCTR(block, iv)
	log.Print(buf.Len())

	iterationCount := 0 // For output purposes

	part, err := r.NextPart()
	if err != nil {
		log.Print(err)
	}
	partRead := true

	newFilepath := "./files/" + dir

	//Create directory if not exists
	os.MkdirAll(newFilepath, os.ModePerm)

	//newFilepath += "/" + handler.Filename + ".bin"
	newFilepath += "/" + part.FileName() + ".bin"
	log.Print(newFilepath)

	//overrides old file if exists
	rmFile(newFilepath)

	outFile, err := os.OpenFile(newFilepath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}

	defer outFile.Close()
	partRead = true // part of file was read once (for metadata purposes)

	for {
		if !partRead {
			part, err = r.NextPart()
		}
		partRead = false
		if err == io.EOF {
			break
		}

		if err != nil {
			rmFile(newFilepath)
			return fmt.Errorf("Read %d bytes: %v", buf.Len(), err)
		}
		iterationCount++            // For output purposes
		if iterationCount >= 1000 { //
			fmt.Print(".")     //	//
			iterationCount = 0 //	//
		} //						//

		io.Copy(buf, part)
		stream.XORKeyStream(buf.Bytes(), buf.Bytes()[:buf.Len()])
		outFile.Write(buf.Bytes()[:buf.Len()])
	}
	outFile.Write(iv)

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

	buf := make([]byte, 1024)
	stream := cipher.NewCTR(block, iv)
	iterationCount := 0 // for output purposes
	for {
		n, err := file.Read(buf)
		if n > 0 {
			iterationCount++            // for output purposes
			if iterationCount >= 1000 { //
				fmt.Print(".")     //	//
				iterationCount = 0 //	//
			} // 						//

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
