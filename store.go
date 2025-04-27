package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const DefaultStoreRoot = "khalilnetwork"

func CASKeyTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashString := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashString) / blocksize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i+1)*blocksize
		paths[i] = hashString[from:to]
	}

	return PathKey{
		PathName: filepath.Join(paths...),
		FileName: hashString,
	}
}

type KeyTransformFunc func(string) PathKey
type PathKey struct {
	PathName string
	FileName string
}

func (p PathKey) FirstParentPath() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

func (p PathKey) FullPathWithRoot(root string) string {
	return path.Join(root, p.FullPath())
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

type StoreOpts struct {
	// Root is the root directory where the files will be stored
	Root              string
	KeyTransformeFunc KeyTransformFunc
}
type Store struct {
	StoreOpts
}

var DefaultKeyTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		FileName: key,
	}
}

func NewStore(opts StoreOpts) *Store {
	if opts.KeyTransformeFunc == nil {
		opts.KeyTransformeFunc = DefaultKeyTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = DefaultStoreRoot
	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) HasKey(key string) bool {
	pathKey := s.KeyTransformeFunc(key)
	_, err := os.Stat(pathKey.FullPathWithRoot(s.Root))

	return !errors.Is(err, fs.ErrNotExist)
}

func (s *Store) Delete(key string) error {
	pathKey := s.KeyTransformeFunc(key)
	defer func() {
		log.Printf("Deleted %s", pathKey.FileName)
	}()

	return os.RemoveAll(path.Join(s.Root, pathKey.FirstParentPath()))
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer func(f io.ReadCloser) {
		err := f.Close()
		if err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}(f)

	buf := new(bytes.Buffer)

	_, err = io.Copy(buf, f)

	return buf, err
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.KeyTransformeFunc(key)
	return os.Open(pathKey.FullPathWithRoot(s.Root))
}

func (s *Store) writeSteam(key string, r io.Reader) error {
	pathKey := s.KeyTransformeFunc(key)
	pathNameWithRoot := path.Join(s.Root, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return err
	}

	fullPathWithRoot := path.Join(s.Root, pathKey.FullPath())
	f, err := os.Create(pathKey.FullPathWithRoot(s.Root))
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("Wrote %d bytes to %s", n, fullPathWithRoot)

	return nil
}
