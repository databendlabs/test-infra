package hook

import (
	"context"
	"datafuselabs/test-infra/chatbots/plugins"
	"fmt"
	"net/http"
	"sync"

	githubcli "datafuselabs/test-infra/chatbots/github"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"
)

const (
	helloEndpoint           string = "/hello"
	payloadEndpoint         string = "/payload"
	benchmarkResultEndpoint string = "/benchmark/{pr:.*}/{commit:.*}"
)

type Config struct {
	StorageEndpoint StorageInterface
	ctx             context.Context
	Logger          zerolog.Logger
	GithubToken     string
	WebhookToken    string //
	Address         string // binded address
}

type Server struct {
	Config Config
	wg     sync.WaitGroup
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

func (s *Server) Start() {
	s.RegistEndpoints()
	http.ListenAndServe(s.Config.Address, nil)
}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

func (s *Server) payload(w http.ResponseWriter, req *http.Request) {
	// validate github token and webhook token
	if s.Config.GithubToken == "" {
		fmt.Fprintf(w, "Unable to fetch the github token for the chatbot\n")
	}
	payload, err := github.ValidatePayload(req, []byte(s.Config.WebhookToken))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "unable to read webhook body", http.StatusBadRequest)
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(req), payload)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "unable to parse webhook", http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:
		if *e.Action != "created" {
			http.Error(w, "issue_comment type must be 'created'", http.StatusOK)
			return
		}
		for name, handler := range plugins.IssueCommentHandlers {
			s.wg.Add(1)
			go func(n string, h plugins.IssueCommentHandler) {
				defer s.wg.Done()
				client, err := githubcli.NewGithubClient(context.Background(), e)
				if err != nil {
					s.Config.Logger.Error().Msgf("Cannot build github client given event %s, %s", *e.Action, err.Error())
				}
				agent := plugins.NewAgent(client)
				err = h(agent, e)
				if err != nil {
					s.Config.Logger.Error().Msgf("Cannot process handler %s, %s", n, err.Error())
				}
			}(name, handler)
		}
	}
}

func (s *Server) RegistEndpoints() {
	http.HandleFunc(helloEndpoint, hello)
	http.HandleFunc(payloadEndpoint, s.payload)
}
