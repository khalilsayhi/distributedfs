package main

import (
	"bytes"
	"testing"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "Foo Bar Baz"
	src := bytes.NewReader([]byte(payload))
	encryptDest := new(bytes.Buffer)
	key := newEncryptorKey()

	_, err := copyEncrypt(key, src, encryptDest)
	if err != nil {
		t.Error(err)
	}
	// Decrypt the data
	decryptDest := new(bytes.Buffer)
	_, err = copyDecrypt(key, encryptDest, decryptDest)
	if err != nil {
		t.Error(err)
	}

	if decryptDest.String() != payload {
		t.Errorf("expected: %s got '%s'", payload, decryptDest.String())
	}
}
