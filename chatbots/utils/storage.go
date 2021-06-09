package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type StorageInterface interface {
	Store(ctx context.Context, info []byte) error
	Retrieve(ctx context.Context) ([]byte, error)
	GetBasePath() string
	SetStoragePath(s string) string
	BuildStoragePath(pr, sha string) (string, error)
}

type FileStorage struct {
	BasePath string // root for all file
	FilePath string //
}

// define the location you want to store your files
func (r *FileStorage) Store(ctx context.Context, data []byte) error {
	err := ioutil.WriteFile(r.FilePath, data, 0666)
	return err
}

// Retrieve will read the file and return the data
func (r *FileStorage) Retrieve(ctx context.Context) ([]byte, error) {
	content, err := ioutil.ReadFile(r.FilePath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (r *FileStorage) GetBasePath() string {
	return r.BasePath
}

func (r *FileStorage) SetStoragePath(s string) string {
	r.FilePath = s
	return s
}

func (r *FileStorage) BuildStoragePath(pr, sha string) (string, error) {
	basePath, err := filepath.Abs(r.GetBasePath())
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		err := os.Mkdir(basePath, 0777)
		if err != nil {
			return "", err
		}
	}
	PRPath := filepath.Join(basePath, pr)
	if _, err := os.Stat(PRPath); os.IsNotExist(err) {
		err := os.Mkdir(PRPath, 0777)
		if err != nil {
			return "", err
		}
	}
	CommitPath := filepath.Join(PRPath, fmt.Sprintf("%s.html", sha))
	return CommitPath, nil
}
