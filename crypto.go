package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

func generateID() string {
	id := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(id)
}

func hashKey(key string) string {
	hash := md5.Sum([]byte(key))

	return hex.EncodeToString(hash[:])
}
func newEncryptorKey() []byte {
	keyBuffer := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, keyBuffer)

	return keyBuffer
}

func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		nw  = blockSize
	)

	for {
		n, err := src.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		if n > 0 {
			stream.XORKeyStream(buf[:n], buf[:n])
			written, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += written
		}
	}

	return nw, nil
}

func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// Prepend the IV to the file
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)

	return copyStream(stream, block.BlockSize(), src, dst)
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	// Read the IV from the io.Reader
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)

	return copyStream(stream, block.BlockSize(), src, dst)
}
