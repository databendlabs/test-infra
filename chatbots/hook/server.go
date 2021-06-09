package hook

import (
	"context"
	"datafuselabs/test-infra/chatbots/plugins"
	"datafuselabs/test-infra/chatbots/utils"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	githubcli "datafuselabs/test-infra/chatbots/github"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	helloEndpoint           string = "/hello"
	payloadEndpoint         string = "/payload"
	uploadEndpoint          string = "/upload"
	benchmarkResultEndpoint string = "/benchmark/{pr:.*}/{commit:.*}"
)

type Config struct {
	StorageEndpoint utils.StorageInterface
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

func NewConfig(StorageBackend utils.StorageInterface, ctx context.Context, Logger zerolog.Logger, GithubToken, WebhookToken, Address string) Config {
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
	default:
		log.Info().Msgf("only issue_comment event is supported now")
	}
}

func (s *Server) upload(w http.ResponseWriter, req *http.Request) {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	req.ParseMultipartForm(10 << 20)

	file, _, err := req.FormFile("upload")
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to process compare result")
		http.Error(w, err.Error(), 403)
		return
	}
	pr := req.FormValue("PR")
	sha := req.FormValue("SHA")
	s.Config.Logger.Info().Msgf("received SHA %s from PR %s", sha, pr)
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to read result")
		http.Error(w, err.Error(), 403)
		return
	}
	filePath, err := s.buildFilePath(pr, sha)
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to build storage file path")
		return
	}
	filePath = s.Config.StorageEndpoint.SetStoragePath(filePath)
	s.Config.Logger.Info().Msgf("storage path: %s", filePath)
	err = s.Config.StorageEndpoint.Store(s.Config.ctx, fileBytes)
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to store result file, %s", err.Error())
		return
	}
}

func (s *Server) buildFilePath(pr, sha string) (string, error) {
	file, err := s.Config.StorageEndpoint.BuildStoragePath(pr, sha)
	if err != nil {
		return "", err
	}
	result := s.Config.StorageEndpoint.SetStoragePath(file)
	return result, nil
}

func (s *Server) RegistEndpoints() {
	http.HandleFunc(helloEndpoint, hello)
	http.HandleFunc(payloadEndpoint, s.payload)
	http.HandleFunc(uploadEndpoint, s.upload)
}
