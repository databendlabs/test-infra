package utils

import (
	"context"
	"io/ioutil"
)

type StorageInterface interface {
	Store(ctx context.Context, info []byte) error
	Retrieve(ctx context.Context) ([]byte, error)
}

type FileStorage struct {
	Path string //location for the file
}

// define the location you want to store your files
func (r *FileStorage) Store(ctx context.Context, data []byte) error {
	err := ioutil.WriteFile(r.Path, data, 0644)
	return err
}

// Retrieve will read the file and return the data
func (r *FileStorage) Retrieve(ctx context.Context) ([]byte, error) {
	content, err := ioutil.ReadFile(r.Path)
	if err != nil {
		return nil, err
	}
	return content, nil
}
