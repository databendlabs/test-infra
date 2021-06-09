package hook

import "context"

type StorageInterface interface {
	Store(ctx context.Context, info []byte) error
	Retrieve(ctx context.Context) ([]byte, error)
}
