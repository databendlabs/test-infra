package main

import (
	"context"
	"datafuselabs/test-infra/chatbots/hook"
	"datafuselabs/test-infra/chatbots/utils"

	"github.com/jnovack/flag"
	"github.com/rs/zerolog/log"
)

var (
	GithubToken  string
	WebhookToken string
	Address      string
	Region       string
	Bucket       string
	Endpoint     string
)

func init() {
	flag.StringVar(&GithubToken, "github-token", "", "chatbot github token")
	flag.StringVar(&WebhookToken, "webhook-token", "", "webhook token for chatbot server")
	flag.StringVar(&Address, "address", "", "address that chatbot server binds to")
	flag.StringVar(&Region, "region", "", "S3 Storage Region")
	flag.StringVar(&Bucket, "bucket", "", "S3 Storage bucket")
	flag.StringVar(&Endpoint, "endpoint", "", "S3 storage endpoint")

}

func main() {
	flag.Parse()
	cfg := hook.NewConfig(
		&utils.FileStorage{
			BasePath: "./tmp",
		},
		context.Background(),
		log.Logger,
		GithubToken,
		WebhookToken,
		Address,
		Region, Bucket, Endpoint,
		"/Users/zhihanzhang/Documents/go/src/test-infra/chatbots/cmd/templates/",
		"/Users/zhihanzhang/Documents/go/src/test-infra/chatbots/cmd/static/",
	)
	server := hook.NewServer(cfg)
	server.Start()
}
