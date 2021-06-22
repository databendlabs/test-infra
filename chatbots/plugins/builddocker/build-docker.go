// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package builddocker

import (
	"context"
	githubcli "datafuselabs/test-infra/chatbots/github"
	"datafuselabs/test-infra/chatbots/plugins"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	pluginName = "build-docker"
)

var (
	reg = regexp.MustCompile(`(?mi)^/build-docker\s*(?P<RELEASE>master|main|latest|current|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
)

func init() {
	log.Info().Msgf("registed plugin: %s", pluginName)
	plugins.RegisterIssueCommentHandler(pluginName, handleIssueComment)
}

func handleIssueComment(client *plugins.Agent, ic *github.IssueCommentEvent) error {
	handler, err := newRunPerf(ic, log.With().Str("issue comment", "build-docker").Logger())
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
	if err != nil {
		return err
	}
	err = handlerhelper(h, lastSHA, lastTag)
	if err != nil {
		if strings.Contains(err.Error(), "is not a org, member nor a collaborator") {
			h.gc.PostComment(err.Error())
		}
		return err
	}
	err = h.gc.CreateRepositoryDispatch("build_docker", h.Payloads)
	if err != nil {
		h.log.Error().Msgf("cannot create run-perf repository dispatch, %s", err.Error())
		return err
	}
	// err = h.gc.PostComment(fmt.Sprintf("run performance on sha %s reference on %s", h.Payloads["CURRENT_BRANCH"], h.Payloads["REF_BRANCH"]))
	// if err != nil {
	// 	return err
	// }

	return nil
}

func handlerhelper(h *handler, sha string, lastTag string) error {
	command := extractCommand(h.gc.CommentBody)
	h.log.Log().Msgf(command)
	matches := h.regexp.FindAllStringSubmatch(command, -1)
	if matches == nil {
		return fmt.Errorf("there is no matching regex")
	}
	err := h.verifyUser()
	if err != nil {
		return err
	}

	if len(matches) < 1 || len(matches[0]) < 2 {
		return nil
	}
	switch matches[0][1] {
	case "latest":
		h.RefBranch = lastTag
	case "current":
		h.RefBranch = sha
	default:
		h.RefBranch = matches[0][1]
	}
	h.log.Info().Msgf("build image on branch: %s", h.RefBranch)
	err = extractPayload(h, sha)
	return err
}

func extractPayload(h *handler, sha string) error {
	h.Payloads["REF"] = h.RefBranch
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

	// define the branch that the current branch will be tested with
	RefBranch string

	// define a series of client-payloads that will be posted to workflow
	Payloads map[string]string
}

func newRunPerf(e *github.IssueCommentEvent, log zerolog.Logger) (*handler, error) {
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
