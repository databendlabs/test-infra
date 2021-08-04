// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package runperf

import (
	"context"
	githubcli "datafuselabs/test-infra/chatbots/github"
	"datafuselabs/test-infra/chatbots/plugins"
	"fmt"
	"github.com/google/go-github/v35/github"
	guuid "github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	pluginName = "run-perf"
)

var (
	reg = regexp.MustCompile(`(?mi)^/run-perf\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
	rerunAll = regexp.MustCompile(`(?mi)^/rerun-perf-all\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
	rerun = regexp.MustCompile(`(?mi)^/rerun-perf\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
)

func init() {
	log.Info().Msgf("regsited plugin: %s", pluginName)
	plugins.RegisterIssueCommentHandler(pluginName, handleIssueComment)
}

func handleIssueComment(client *plugins.Agent, ic *github.IssueCommentEvent) error {
	handler, err := newRunPerf(ic, log.With().Str("issue comment", "fusebench-local").Logger(), client)
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
	lastTag, err := h.gc.GetLatestTag()
	id := guuid.New()
	if err != nil {
		return err
	}
	start := strconv.Itoa(int(time.Now().Unix()))
	name, err := handlerhelper(h, lastSHA, lastTag, start, id.String())
	if err != nil {
		if strings.Contains(err.Error(), "is not an owner, member nor a collaborator") {
			err := h.gc.PostComment(err.Error())
			if err != nil {
				return err
			}
		}
		return err
	}

	err = h.gc.CreateRepositoryDispatch(name, h.Payloads)
	if err != nil {
		h.log.Error().Msgf("cannot create run-perf repository dispatch, %s", err.Error())
		return err
	}
	err = h.gc.PostComment(fmt.Sprintf("run performance on sha %s reference on %s", h.Payloads["CURRENT_BRANCH"], h.Payloads["REF_BRANCH"]))
	if err != nil {
		return err
	}
	err = h.gc.UpdateStatus("run-perf-status", "pending", "")
	if err != nil {
		h.log.Error().Msgf("cannot update status, %s", err.Error())
		return err
	}
	return nil
}

func findMatches(h *handler, command string) (matches [][]string, dispatch_name string) {
	matches = nil
	for name, regex := range h.regexp {
		matches := regex.FindAllStringSubmatch(command, -1)
		if matches != nil {
			h.log.Log().Msgf("start to run command %s", name)
			return matches, name
		}

	}
	return nil, dispatch_name
}

func handlerhelper(h *handler, sha string, lastTag, startTime, uuid string) (string, error) {
	command := extractCommand(h.gc.CommentBody)
	h.log.Log().Msgf(command)
	matches, name := findMatches(h, command)
	if matches == nil {
		return "", fmt.Errorf("there is no matching regex")
	}
	err := h.verifyUser()
	if err != nil {
		return "", err
	}

	if len(matches) < 1 || len(matches[0]) < 2 {
		return "", nil
	}
	switch matches[0][1] {
	case "latest":
		h.RefBranch = lastTag
	default:
		h.RefBranch = matches[0][1]
	}
	h.CurrentBranch = sha
	h.log.Info().Msgf("current testing branch: %s, reference branch: %s", h.CurrentBranch, h.RefBranch)
	err = extractPayload(h, sha, startTime, uuid)
	return name, err
}

func extractPayload(h *handler, sha, start, uuid string) error {
	h.Payloads["CURRENT_BRANCH"] = h.CurrentBranch
	h.Payloads["PR_NUMBER"] = strconv.Itoa(h.gc.Pr)
	h.Payloads["LAST_COMMIT_SHA"] = sha
	h.Payloads["REF_BRANCH"] = h.RefBranch
	h.Payloads["REGION"] = h.Region
	h.Payloads["BUCKET"] = h.Bucket
	h.Payloads["ENDPOINT"] = h.Endpoint
	h.Payloads["UUID"] = uuid
	return nil

}

// handler is a struct that contains data about a github event and provides functions to help handle it.
type handler struct {

	// regexp is the regular expression describing the command. It must have an optional 'un' prefix
	// as the first subgroup and the arguments to the command as the second subgroup.
	regexp map[string]*regexp.Regexp
	// gc is the githubClient to use for creating response comments in the event of a failure.
	gc *githubcli.GithubClient

	// log define structed logging interface.
	log zerolog.Logger

	// define the branch that the current branch will be tested with
	RefBranch string

	// define the current branch we want to test
	CurrentBranch string

	// define a series of client-payloads that will be posted to workflow
	Payloads map[string]string

	// define S3 storage configuration
	Region string
	Bucket string
	Endpoint string
}

func newRunPerf(e *github.IssueCommentEvent, log zerolog.Logger, client *plugins.Agent) (*handler, error) {
	githubCli, err := githubcli.NewGithubClient(context.Background(), e, client.Token)
	if err != nil {
		log.Error().Msgf("Unable to initialize github client given issue comment event %s, %s", e.GetComment().String(), err.Error())

		return nil, err
	}
	regs := make(map[string]*regexp.Regexp)
	regs["run_perf"] = reg
	regs["rerun_perf_all"] = rerunAll
	regs["rerun_perf"] = rerun
	return &handler{
		regexp:   regs,
		gc:       githubCli,
		log:      log,
		Payloads: make(map[string]string),
		Region: client.Region,
		Bucket: client.Bucket,
		Endpoint: client.Endpoint,
	}, nil
}
