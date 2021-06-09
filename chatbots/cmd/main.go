package main

import (
	"context"
	"datafuselabs/test-infra/chatbots/hook"
	"datafuselabs/test-infra/chatbots/utils"

	"github.com/jnovack/flag"
	"github.com/rs/zerolog/log"
)

var (
	GITHUB_TOKEN  string
	WEBHOOK_TOKEN string
	Address       string
)

func init() {
	flag.StringVar(&GITHUB_TOKEN, "github-token", "", "chatbot github token")
	flag.StringVar(&WEBHOOK_TOKEN, "webhook-token", "", "webhook token for chatbot server")
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
		GITHUB_TOKEN,
		WEBHOOK_TOKEN,
		Address,
	)
	server := hook.NewServer(cfg)
	server.Start()
}
