package plugins

import (
	"context"
	githubcli "datafuselabs/test-infra/chatbots/github"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	pluginName = "fusebench-local"
)

var (
	reg = regexp.MustCompile(`(?mi)^/fusebench-local\s*(?P<RELEASE>master|main|current|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
)

func init() {
	RegisterIssueCommentHandler(pluginName, handleIssueComment)
}

func handleIssueComment(client *Agent, ic *github.IssueCommentEvent) error {
	handler, err := newOKToFusebench(ic, log.With().Str("issue comment", "fusebench-local").Logger())
	if err != nil {
		return err
	}
	err = handle(handler)
	if err != nil {
		return err
	}
	return nil
}

func extractCommand(s string) string {
	s = strings.TrimLeft(s, "\r\n\t ")
	if i := strings.Index(s, "\n"); i != -1 {
		s = s[:i]
	}
	s = strings.TrimRight(s, "\r\n\t ")
	return s
}

// Verify if user is allowed to perform activity.
func (h handler) verifyUser() error {
	var allowed bool
	allowedAssociations := []string{"COLLABORATOR", "MEMBER", "OWNER"}
	for _, a := range allowedAssociations {
		if a == h.gc.AuthorAssociation {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("@%s is not a org, member nor a collaborator and cannot execute fusebench.", h.gc.Author)
	}
	h.log.Info().Msgf("author is a owner, member or collaborator")
	return nil
}

// ok-to-fusebench <branch-name> will run fusebench test given branch reference
func handle(h *handler) error {
	if h == nil {
		return nil
	}
	h.log.Info().Msgf(h.gc.GetIssueState())
	lastSHA := h.gc.GetLastCommitSHA()
	err := handlerhelper(h, lastSHA)
	if err != nil {
		if strings.Contains(err.Error(), "is not a org, member nor a collaborator") {
			h.gc.PostComment(err.Error())
		}
		return err
	}
	return nil
}

func handlerhelper(h *handler, sha string) error {
	command := extractCommand(h.gc.CommentBody)
	matches := h.regexp.FindAllStringSubmatch(command, -1)
	if matches == nil {
		return nil
	}
	err := h.verifyUser()
	if err != nil {
		return err
	}

	if len(matches) < 1 || len(matches[0]) < 2 {
		return nil
	}
	switch matches[0][1] {
	case "current":
		h.BranchName = sha
	default:
		h.BranchName = matches[0][1]
	}
	h.log.Info().Msgf("current testing branch: %s", h.BranchName)
	err = extractPayload(h, sha)
	return err
}

func extractPayload(h *handler, sha string) error {
	h.Payloads["BranchName"] = h.BranchName
	h.Payloads["PR_NUMBER"] = strconv.Itoa(h.gc.Pr)
	h.Payloads["LAST_COMMIT_SHA"] = sha
	return nil

}

// handler is a struct that contains data about a github event and provides functions to help handle it.
type handler struct {

	// regexp is the regular expression describing the command. It must have an optional 'un' prefix
	// as the first subgroup and the arguments to the command as the second subgroup.
	regexp *regexp.Regexp
	// gc is the githubClient to use for creating response comments in the event of a failure.
	gc *githubcli.GithubClient

	// log define structed logging interface.
	log zerolog.Logger

	// define the branch that will be tested through fusebench
	BranchName string

	// define a series of client-payloads that will be posted to workflow
	Payloads map[string]string
}

func newOKToFusebench(e *github.IssueCommentEvent, log zerolog.Logger) (*handler, error) {
	githubCli, err := githubcli.NewGithubClient(context.Background(), e)
	if err != nil {
		log.Error().Msgf("Unable to initialize github client given issue comment event %s, %s", e.GetComment().String(), err.Error())

		return nil, err
	}
	return &handler{
		regexp:   reg,
		gc:       githubCli,
		log:      log,
		Payloads: make(map[string]string),
	}, nil
}
