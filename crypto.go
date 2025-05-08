package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func newEncryptorKey() []byte {
	keyBuffer := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, keyBuffer)

	return keyBuffer
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

	var (
		buf    = make([]byte, 32*1024)
		stream = cipher.NewCTR(block, iv)
		nw     = block.BlockSize()
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

	var (
		buf    = make([]byte, 32*1024)
		stream = cipher.NewCTR(block, iv)
		nw     = block.BlockSize()
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			written, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += written
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
	}

	return nw, nil
}
