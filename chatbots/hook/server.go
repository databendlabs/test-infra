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
	MetaStorage utils.MetaStore
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
				agent := plugins.NewAgent(client, &s.Config.MetaStorage)
				err = h(agent, e)
				if err != nil {
					s.Config.Logger.Error().Msgf("Cannot process handler %s, %s", n, err.Error())
				}
			}(name, handler)
		}
	default:
		log.Debug().Msgf("only issue_comment event is supported now, %T", e)
	}
}

func (s *Server) processReqFile(req *http.Request, fileName string) error{
	file, _, err := req.FormFile(fileName)
	if err == http.ErrMissingFile {
		return nil
	}
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to process file %s result, %v", fileName, err.Error())
		return fmt.Errorf("unable to process file %s result, %v", fileName, err.Error())
	}
	owner := req.FormValue("OWNER")
	repo := req.FormValue("REPO")
	pr := req.FormValue("PR")
	sha := req.FormValue("SHA")
	uuid := req.FormValue("UUID")

	s.Config.Logger.Info().Msgf("received SHA %s from PR %s", sha, pr)
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to read result")
		return fmt.Errorf("unable to read file %s result, %v", fileName, err.Error())
	}
	err = s.Config.StorageEndpoint.Store(s.Config.ctx, owner, repo, pr, sha, uuid, fileName, fileBytes)
	if err != nil {
		s.Config.Logger.Error().Msgf("unable to store result file, %s", err.Error())
		return fmt.Errorf("unable to store file %s result, %v", fileName, err.Error())
	}
	return nil
}

// handle three type of upload fields
// log file
// compare file
func (s *Server) upload(w http.ResponseWriter, req *http.Request) {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	err := req.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Error().Msgf("Unable to parse file form, %+v", err)
		return
	}
	files := []string{"compare.html", "current.log", "ref.log"}
	for _, f := range files {
		err = s.processReqFile(req, f)
		if err != nil {
			http.Error(w, err.Error(), 503)
			return
		}
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
	UUID string `json:"uuid,omitempty"`
	DispatchName string `json:"dispatch_name,omitempty"`
	Current string `json:"current,omitempty"`
	Ref string `json:"ref,omitempty"`
	Compare string `json:"compare,omitempty"`
	Status string`json:"status,omitempty"`
	Conclusion string `json:"conclusion,omitempty"`
	PRLink string`json:"PRLink,omitempty"`
	CurrentLog string`json:"currentLog,omitempty"`
	RefLog string`json:"refLog,omitempty"`
	StartTime string `json:"start_time,omitempty"`
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	switch status.DispatchName {
	case "build-docker":
		err := s.HandleStatus(status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	case "run-perf":
		err := s.HandleStatus(status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		err = s.HandleReport(status, jb)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	default:
		http.Error(w, fmt.Sprintf("Not support dispatch %s for now", status.DispatchName), http.StatusBadRequest)
		return
	}
}

func (s *Server) HandleStatus(meta StatusMeta) error {
	return s.Config.MetaStorage.Store([]string{meta.DispatchName,
		meta.Organization, meta.Repository,
		meta.PRNumber, meta.CommitSHA, meta.UUID}, []byte(meta.Status))
}

func (s *Server) HandleReport(meta StatusMeta, jb []byte) error {
	key := []string{"report", meta.StartTime, meta.Organization, meta.Repository, meta.PRNumber, meta.CommitSHA, meta.UUID}
	return s.Config.MetaStorage.Store(key, jb)
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
