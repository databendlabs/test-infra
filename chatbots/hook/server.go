package hook

import (
	"context"
	"datafuselabs/test-infra/chatbots/plugins"
	_ "datafuselabs/test-infra/chatbots/plugins/builddocker"
	_ "datafuselabs/test-infra/chatbots/plugins/runperf"
	"datafuselabs/test-infra/chatbots/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	githubcli "datafuselabs/test-infra/chatbots/github"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	helloEndpoint           string = "/hello"
	payloadEndpoint         string = "/payload"
	statusEndpoint         string = "/status"
	uploadEndpoint          string = "/upload"
	indexEndpoint				string = "/"
	benchmarkResultEndpoint string = "/benchmark/{pr:.*}/{commit:.*}"
)

type Config struct {
	StorageEndpoint utils.StorageInterface
	ctx             context.Context
	Logger          zerolog.Logger
	GithubToken     string
	WebhookToken    string //
	Address         string // binded address
	TemplateDir		string
	StaticDir 		string
}

type Server struct {
	Config Config
	wg     sync.WaitGroup
}

func NewConfig(StorageBackend utils.StorageInterface, ctx context.Context, Logger zerolog.Logger, GithubToken, WebhookToken, Address, templateDir, staticDir string) Config {
	return Config{
		StorageEndpoint: StorageBackend,
		ctx:             ctx,
		Logger:          Logger,
		GithubToken:     GithubToken,
		WebhookToken:    WebhookToken,
		Address:         Address,
		TemplateDir: 	 templateDir,
		StaticDir: 	   	 staticDir,
	}
}

func NewServer(cfg Config) Server {
	return Server{
		Config: cfg,
	}
}

func (s *Server) Start() {
	s.RegistEndpoints()
	err := http.ListenAndServe(s.Config.Address, nil)
	panic(err)
}

func hello(w http.ResponseWriter, req *http.Request) {
	_, err := fmt.Fprintf(w, "hello\n")
	if err != nil {
		http.Error(w, err.Error(), 404)
	}
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
		log.Info().Msgf("received issue comment %s", *e.Action)
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
		log.Info().Msgf("only issue_comment event is supported now, %T", e)
	}
}

func (s *Server) upload(w http.ResponseWriter, req *http.Request) {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Error().Msgf("Unable to parse file form, %+v", err)
		return
	}

	file, _, err := req.FormFile("upload")
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to process compare result, %v", err.Error())
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
	err = s.Config.StorageEndpoint.Store(s.Config.ctx, pr, sha, "compare.html", fileBytes)
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to store result file, %s", err.Error())
		return
	}
}

// StatusMeta will receive update from github workflow actions
type StatusMeta struct {
	Organization string `json:"org,omitempty"`
	Repository string `json:"repo,omitempty"`
	PRNumber string `json:"pr,omitempty"`
	CommitSHA string `json:"commitSHA,omitempty"`
	RunId string `json:"run_id,omitempty"`
	Author string `json:"author,omitempty"`
	Current *string `json:"current,omitempty"`
	Ref *string `json:"ref,omitempty"`
	Compare string `json:"compare,omitempty"`
	Status string`json:"status,omitempty"`
	Conclusion string `json:"conclusion,omitempty"`
	PRLink string`json:"PRLink,omitempty"`
	CurrentLog string`json:"currentLog,omitempty"`
	RefLog string`json:"refLog,omitempty"`
}

// status endpoint will receive status update from github workflow
func (s *Server) status(w http.ResponseWriter, req *http.Request) {
	// validate github token and webhook token
	// Declare a new Person struct.
	var status StatusMeta

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(req.Body).Decode(&status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	jb, err := json.Marshal(status)

	log.Info().Msgf(status.RunId)
	run_id, err := strconv.Atoi(status.RunId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	state, err := githubcli.GetActionStatus(context.Background(), status.Organization, status.Repository, int64(run_id))
	log.Info().Msgf("current status %s", state)
	// Only store the latest metadata
	err = s.Config.StorageEndpoint.Store(s.Config.ctx, status.PRNumber, status.CommitSHA, "meta.json", jb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

}

type Metas struct {
	Items []StatusMeta
}

func (s *Server) RegistEndpoints() {
	http.HandleFunc(helloEndpoint, hello)
	http.HandleFunc(payloadEndpoint, s.payload)
	http.HandleFunc(uploadEndpoint, s.upload)
	http.HandleFunc(statusEndpoint, s.status)
	m := Metas{Items: []StatusMeta{}}
	http.HandleFunc(indexEndpoint, s.handleSimpleTemplate("index.html", m))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
}
