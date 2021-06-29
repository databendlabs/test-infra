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
)

func init() {
	flag.StringVar(&GithubToken, "github-token", "", "chatbot github token")
	flag.StringVar(&WebhookToken, "webhook-token", "", "webhook token for chatbot server")
	flag.StringVar(&Address, "address", "", "address that chatbot server binds to")

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
	)
	server := hook.NewServer(cfg)
	server.Start()
}
