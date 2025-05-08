package main

import (
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

// filename => example.txt
// path => TransformFunc(filename) => ROOT/example/txt => 0/1/2/3/4/5/6/7/8/9/a/b/c/d/e/f

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

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeSteam(key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(encKey, r, f)

	return int64(n), err
}

func (s *Store) writeSteam(key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}

	return io.Copy(f, r)
}

func (s *Store) HasKey(key string) bool {
	pathKey := s.KeyTransformeFunc(key)
	_, err := os.Stat(pathKey.FullPathWithRoot(s.Root))

	return !errors.Is(err, fs.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.KeyTransformeFunc(key)
	defer func() {
		log.Printf("Deleted %s", pathKey.FileName)
	}()

	return os.RemoveAll(path.Join(s.Root, pathKey.FirstParentPath()))
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.KeyTransformeFunc(key)

	file, err := os.Open(pathKey.FullPathWithRoot(s.Root))
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (s *Store) openFileForWriting(key string) (*os.File, error) {
	pathKey := s.KeyTransformeFunc(key)
	pathNameWithRoot := path.Join(s.Root, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	return os.Create(pathKey.FullPathWithRoot(s.Root))
}
