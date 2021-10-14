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

func encrypt(infile multipart.File, outFile *os.File, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Panic(err)
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1024)
	stream := cipher.NewCTR(block, iv)

	iterationCount := 1000

	for {
		n, err := infile.Read(buf)
		if n > 0 {
			iterationCount++
			if iterationCount >= 1000 {
				fmt.Print(".")
				iterationCount = 0
			}
			stream.XORKeyStream(buf, buf[:n])
			outFile.Write(buf[:n])
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("Read %d bytes: %v", n, err)
		}
	}
	log.Print("outfile write")
	outFile.Write(iv)

	return nil
}

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
	log.Print("actual decryption")
	iterationCount := 1000
	for {
		n, err := file.Read(buf)
		if n > 0 {
			iterationCount++
			if iterationCount >= 1000 {
				fmt.Print(".")
				iterationCount = 0
			}
			// The last bytes are the IV, don't belong the original message
			if n > int(length) {
				n = int(length)
			}
			length -= int64(n)
			stream.XORKeyStream(buf, buf[:n])

			if _, err = io.Copy(part, bytes.NewReader(buf[:n])); err != nil {
				return err
			}
			if err != nil {
				log.Print(err)
			}
		}

		if err == io.EOF {
			log.Print(err)
			break
		}

		if err != nil {
			log.Printf("Read %d bytes: %v", n, err)
			break
		}
	}
	return nil
}