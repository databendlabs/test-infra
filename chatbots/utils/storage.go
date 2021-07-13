package utils

import (
	"context"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type StorageInterface interface {
	Store(ctx context.Context, owner, repo, pr, sha, uuid, filename string, info []byte) error
	Retrieve(ctx context.Context, owner, repo, pr, sha, uuid, filename string) ([]byte, error)
	GetBasePath() string
	GetFilePath(owner, repo, pr, sha, uuid, filename string) string
}

type COSStorage struct {
	url string
	secretId string
	secretKey string
	client *cos.Client
}

func (r *COSStorage) Store(ctx context.Context, owner, repo, pr, sha, uuid, filename string, data []byte) error {
	location := filepath.Join(owner, repo, pr, sha, uuid, filename)
	f := strings.NewReader(string(data))
	_, err :=  r.client.Object.Put(ctx, location, f, nil)
	return err
}

func (r *COSStorage) Retrieve(ctx context.Context, owner, repo, pr, sha, uuid, filename string) ([]byte, error) {
	location := filepath.Join(owner, repo, pr, sha, uuid, filename)
	resp, err :=  r.client.Object.Get(ctx, location,nil)
	if err != nil {
		return nil, err
	}
	bs, _ := ioutil.ReadAll(resp.Body)
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (r *COSStorage) GetURL(ctx context.Context, owner, repo, pr, sha, uuid, filename string) string {
	location := filepath.Join(owner, repo, pr, sha, uuid, filename)
	u :=  r.client.Object.GetObjectURL(location)
	return u.String()
}

func (r *COSStorage) GetBasePath() string {
	return r.url
}

type FileStorage struct {
	BasePath string // root for all file
	FilePath string //
}

// Store define the location you want to store your files
// TODO refactor
func (r *FileStorage) Store(ctx context.Context, owner, repo, pr, sha, uuid, filename string, data []byte) error {
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
func (r *FileStorage) Retrieve(ctx context.Context, owner, repo, pr, sha, uuid, filename string) ([]byte, error) {
	address := filepath.Join(r.BasePath, owner, repo, pr, sha, uuid, filename)
	content, err := ioutil.ReadFile(address)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (r *FileStorage) GetBasePath() string {
	return r.BasePath
}

func (r *FileStorage) GetFilePath(owner, repo, pr, sha, uuid, filename string) string {
	address := filepath.Join(r.BasePath, owner, repo, pr, sha, uuid, filename)
	return address
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

// MetaStore backed by badgerDB (for actions' metadata management)
// Tentative storage schema:
// build-docker: mainly need for UUID based Get and Update
// dispatchName/org/repo/pr/commitSHA/UUID/Status
// run-perf: need two kinds of api, one for timestamp based reverse iteration
// one for status management
// report/StartTime/DispatchName/Org/Repo/PR/commitSHA// -> meta json
type MetaStore struct {
	path string
	DB *badger.DB
}

func (m *MetaStore) Store(keys []string, value []byte) error {
	key := strings.Join(keys, "/")
	err := m.DB.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value)
		err := txn.SetEntry(e)
		return err
	})
	return err
}

func (m *MetaStore) GetCopy(keys []string) ([]byte, error) {
	key := strings.Join(keys, "/")
	var value []byte
	err := m.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		// Alternatively, you could also use item.ValueCopy().
		value, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})
	return value, err
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