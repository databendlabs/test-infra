package main

import (
	"context"
	"datafuselabs/test-infra/chatbots/hook"
	"datafuselabs/test-infra/chatbots/utils"
	"strings"

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
	TemplateDir string
	StaticDir string
)

func init() {
	flag.StringVar(&GithubToken, "github-token", "", "chatbot github token")
	flag.StringVar(&WebhookToken, "webhook-token", "", "webhook token for chatbot server")
	flag.StringVar(&Address, "address", "", "address that chatbot server binds to")
	flag.StringVar(&Region, "region", "", "S3 Storage Region")
	flag.StringVar(&Bucket, "bucket", "", "S3 Storage bucket")
	flag.StringVar(&Endpoint, "endpoint", "", "S3 storage endpoint")
	flag.StringVar(&TemplateDir, "template-dir", "", "dashboard template dir")
	flag.StringVar(&StaticDir, "static-dir", "", "dashboard static file dir")

}

func main() {
	flag.Parse()
	if !strings.HasPrefix(Endpoint, "http://") && !strings.HasPrefix(Endpoint, "https://") {
		Endpoint = "https://" + Endpoint
	}
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
		TemplateDir,
		StaticDir,
	)
	server := hook.NewServer(cfg)
	server.Start()
}
