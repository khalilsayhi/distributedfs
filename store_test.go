package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestKeyTransformFunc(t *testing.T) {
	key := "mybestpicture"
	pathname := CASKeyTransformFunc(key)
	fmt.Println(pathname)
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		KeyTransformeFunc: CASKeyTransformFunc,
	}
	store := NewStore(
		opts,
	)

	key := "mybestpicture"
	data := []byte("test data")
	err := store.writeSteam(key, bytes.NewReader(data))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !store.HasKey(key) {
		t.Errorf("expected key %s to exist", key)
	}

	readData, err := store.Read(key)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	b, err := io.ReadAll(readData)
	if string(b) != string(data) {
		t.Errorf("expected %s, got %s", data, b)
	}

	if err = store.Delete(key); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if store.HasKey(key) {
		t.Errorf("expected key %s to not exist", key)
	}
}

func TestDelete(t *testing.T) {
	opts := StoreOpts{
		KeyTransformeFunc: CASKeyTransformFunc,
	}
	store := NewStore(
		opts,
	)

	key := "mybestpicture"
	data := []byte("test data")
	err := store.writeSteam(key, bytes.NewReader(data))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = store.Delete(key)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
