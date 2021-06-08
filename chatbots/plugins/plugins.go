package plugins

import (
	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"

	githubcli "datafuselabs/test-infra/chatbots/github"
)

var (
	issueCommentHandlers = map[string]IssueCommentHandler{}
)

type Agent struct {
	GithubClient *githubcli.GithubClient
	Logger       *zerolog.Event
}

// IssueCommentHandler defines the function contract for a github.IssueCommentEvent handler.
type IssueCommentHandler func(*Agent, *github.IssueCommentEvent) error

func NewAgent(gitClient *githubcli.GithubClient, logger *zerolog.Logger) *Agent {
	return &Agent{
		GithubClient: gitClient,
		Logger:       logger.Info().Str("type", "plugins"),
	}
}
func RegisterIssueCommentHandler(name string, fn IssueCommentHandler) {
	issueCommentHandlers[name] = fn
}
