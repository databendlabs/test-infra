package hook

import (
	"context"

	"github.com/rs/zerolog"
)

const (
	helloEndpoint   string = "hello"
	configEndpoint  string = "configs"
	payloadEndpoint string = "payload"
)

type Config struct {
	StorageEndpoint StorageInterface
	ctx             context.Context
	Logger          zerolog.Logger
	GithubToken     string
	WebhookToken    string //
	Address         string // binded address
}
type StorageInterface interface {
	Store(ctx context.Context, info []byte) error
	Retrieve(ctx context.Context) ([]byte, error)
}

type Server struct {
	Config Config
}

func NewConfig(StorageBackend StorageInterface, ctx context.Context, Logger zerolog.Logger, GithubToken, WebhookToken, Address string) Config {
	return Config{
		StorageEndpoint: StorageBackend,
		ctx:             ctx,
		Logger:          Logger,
		GithubToken:     GithubToken,
		WebhookToken:    WebhookToken,
		Address:         Address,
	}
}

func NewServer(cfg Config) Server {
	return Server{
		Config: cfg,
	}
}
