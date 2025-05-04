package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestStore(t *testing.T) {
	store := newStore()
	defer teardown(t, store)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key_%d", i)
		data := []byte("test data")

		_, err := store.writeSteam(key, bytes.NewReader(data))
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !store.HasKey(key) {
			t.Errorf("expected key %s to exist", key)
		}

		_, readData, err := store.Read(key)
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
}

func newStore() *Store {
	opts := StoreOpts{
		KeyTransformeFunc: CASKeyTransformFunc,
	}
	return NewStore(
		opts,
	)
}

func teardown(t *testing.T, store *Store) {
	if err := store.Clear(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
