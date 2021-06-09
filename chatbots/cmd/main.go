package main

import (
	"datafuselabs/test-infra/chatbots/hook"

	"github.com/jnovack/flag"
)

var (
	GITHUB_TOKEN  string
	WEBHOOK_TOKEN string
)

func init() {
	flag.StringVar(&GITHUB_TOKEN, "github-token", "", "chatbot github token")
	flag.StringVar(&WEBHOOK_TOKEN, "webhook-token", "", "webhook token for chatbot server")

}

func main() {
	flag.Parse()
	cfg := hook.NewConfig
}
