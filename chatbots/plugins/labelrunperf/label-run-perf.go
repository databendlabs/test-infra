// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package labelrunperf

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
	pluginName = "label-run-perf"
)

var (
	reg      = regexp.MustCompile(`(?mi)^run-perf\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
	rerunAll = regexp.MustCompile(`(?mi)^rerun-perf-all\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
	rerun    = regexp.MustCompile(`(?mi)^rerun-perf\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
)

func init() {
	log.Info().Msgf("regsited plugin: %s", pluginName)
	plugins.RegisterPushHandler(pluginName, handlePushComment)
}

func handlePushComment(client *plugins.Agent, ic *github.PushEvent) error {
	handler, err := newLabelRunPerf(ic, log.With().Str("push", "label run perf").Logger(), client)
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
func (h handler) verifyUser(pr *github.PullRequest) error {
	var allowed bool
	allowedAssociations := []string{"COLLABORATOR", "MEMBER", "OWNER"}
	for _, a := range allowedAssociations {
		if a == pr.GetAuthorAssociation() {
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
	prs := h.gc.ListAssociatedPR(h.gc.LastSHA)
	for _, i := range prs {
		err := h.verifyUser(i)
		if err != nil {
			continue
		}
		h.commentByLabel(i, h.gc.LastSHA)
	}
	return nil
}

func (h *handler) commentByLabel(pr *github.PullRequest, sha string) error {
	labels := pr.Labels
	if labels == nil || len(labels) == 0 {
		return nil
	}
	for _, l := range labels {
		h.log.Info().Msgf("current label %s in pr %s", l.GetName(), pr.GetNumber())
		comment := make_comment(*h, l)
		if comment == "" {
			continue
		}
		h.gc.Pr = pr.GetNumber()
		h.log.Info().Msgf("post comment for push commit with owner %s repo %s pr %s", pr.GetBase().Repo.Owner, pr.GetBase().Repo.GetName(), pr.GetNumber())
		err := h.gc.PostComment(comment)
		if err != nil {
			h.log.Error().Msgf("cannot post comment %s", err.Error())
		}
		return nil
	}
	return nil
}

func make_comment(h handler, l *github.Label) string {
	m, _ := findMatches(h, l.GetName())
	if m == nil || len(m) < 1 || len(m[0]) < 2 {
		return ""
	}
	return fmt.Sprintf("/%s", m[0][0])
}

func findMatches(h handler, command string) (matches [][]string, dispatch_name string) {
	matches = nil
	for name, regex := range h.regexp {
		matches := regex.FindAllStringSubmatch(command, -1)
		if matches != nil {
			h.log.Log().Msgf("find match in command %s", name)
			return matches, name
		}

	}
	return nil, dispatch_name
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
}

func newLabelRunPerf(e *github.PushEvent, log zerolog.Logger, client *plugins.Agent) (*handler, error) {
	githubCli, err := githubcli.NewGithubClientByPush(context.Background(), e, client.Token)
	if err != nil {
		log.Error().Msgf("Unable to initialize github client given push event %s, %s", e.GetHeadCommit().GetID(), err.Error())

		return nil, err
	}
	regs := make(map[string]*regexp.Regexp)
	regs["run_perf"] = reg
	regs["rerun_perf_all"] = rerunAll
	regs["rerun_perf"] = rerun
	return &handler{
		regexp: regs,
		gc:     githubCli,
		log:    log,
	}, nil
}
