package utils

import (
	"context"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
)

type StorageInterface interface {
	Store(ctx context.Context, pr, sha, filename string, info []byte) error
	Retrieve(ctx context.Context, pr, sha, filename string) ([]byte, error)
	GetBasePath() string
	List(ctx context.Context)([]meta, error)
}

type FileStorage struct {
	BasePath string // root for all file
	FilePath string //
}

// Store define the location you want to store your files
func (r *FileStorage) Store(ctx context.Context, pr, sha, filename string, data []byte) error {
	basePath, err := filepath.Abs(r.GetBasePath())
	if err != nil {
		return err
	}

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		err := os.Mkdir(basePath, 0777)
		if err != nil {
			return err
		}
	}
	PRPath := filepath.Join(basePath, pr)
	if _, err := os.Stat(PRPath); os.IsNotExist(err) {
		err := os.Mkdir(PRPath, 0777)
		if err != nil {
			return err
		}
	}
	SHAPath := filepath.Join(PRPath, sha)
	if _, err := os.Stat(SHAPath); os.IsNotExist(err) {
		err := os.Mkdir(SHAPath, 0777)
		if err != nil {
			return err
		}
	}
	filename = filepath.Join(SHAPath, filename)
	err = ioutil.WriteFile(filename, data, 0666)
	return err
}

// Retrieve will read the file and return the data
func (r *FileStorage) Retrieve(ctx context.Context, pr, sha, filename string) ([]byte, error) {
	address := filepath.Join(r.BasePath, pr, sha, filename)
	content, err := ioutil.ReadFile(address)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (r *FileStorage) GetBasePath() string {
	return r.BasePath
}

// List all available
func (r *FileStorage) List(ctx context.Context)([]meta, error) {
	base := r.GetBasePath()
	ans := make([]meta, 0)
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		m := meta{PrNumber: path}

		err = filepath.Walk(filepath.Join(base, path), func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			m.CommitSHA = path
			ans = append(ans, m.DeepCopy())
			// TODO parsing meta data
			return nil
		})
		return nil
	})
	return ans, err

}


type meta struct {
	RunID string `json:"run_id"`
	CurrentName string `json:"current_name"`
	RefName string `json:"ref_name"`
	CurrentLog string `json:"current_log"`
	RefLog string `json:"ref_log"`
	PrNumber string `json:"pr_number"`
	CommitSHA string `json:"commit_sha"`
}

func (m meta) DeepCopy() meta {
	return meta {
		RunID: m.RunID,
		CurrentName: m.CurrentName,
		RefName: m.RefName,
		CurrentLog: m.CurrentLog,
		RefLog: m.RefLog,
		PrNumber: m.PrNumber,
		CommitSHA: m.CommitSHA,
	}
}