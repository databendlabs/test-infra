// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package runperf

import (
	"context"
	githubcli "datafuselabs/test-infra/chatbots/github"
	"datafuselabs/test-infra/chatbots/plugins"
	"datafuselabs/test-infra/chatbots/utils"
	"datafuselabs/test-infra/pkg/provider"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/google/go-github/v35/github"
	guuid "github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	pluginName = "run-perf"
)

var (
	reg = regexp.MustCompile(`(?mi)^/run-perf\s*(?P<RELEASE>master|main|latest|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$`)
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
	err = handlerhelper(h, lastSHA, lastTag, start, id.String())
	if err != nil {
		if strings.Contains(err.Error(), "is not an owner, member nor a collaborator") {
			h.gc.PostComment(err.Error())
		}
		return err
	}
	wg := new(sync.WaitGroup)

	// Trigger docker build on current and reference branches
	err = h.gc.CreateRepositoryDispatch("build-docker", map[string]string{"REF": h.Payloads["CURRENT_BRANCH"],
																					"PR_NUMBER": h.Payloads["PR_NUMBER"],
																					"LAST_COMMIT_SHA" : h.Payloads["LAST_COMMIT_SHA"],
																					"UUID": id.String()})
	if err != nil {
		h.log.Error().Msgf("cannot create build docker repository dispatch on branch %s, %s", h.Payloads["CURRENT_BRANCH"], err.Error())
		return err
	}

	go h.waitToReady(wg, h.Payloads["CURRENT_BRANCH"], id.String())

	err = h.gc.CreateRepositoryDispatch("build-docker", map[string]string{"REF": h.Payloads["REF_BRANCH"],
		"PR_NUMBER": h.Payloads["PR_NUMBER"],
		"LAST_COMMIT_SHA" : h.Payloads["LAST_COMMIT_SHA"], "UUID": id.String()})

	if err != nil {
		h.log.Error().Msgf("cannot create build docker repository dispatch on branch %s, %s",h.Payloads["REF_BRANCH"], err.Error())
		return err
	}
	go h.waitToReady(wg, h.Payloads["REF_BRANCH"], id.String())

	wg.Wait()
	err = h.gc.CreateRepositoryDispatch("run-perf", h.Payloads)
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

func handlerhelper(h *handler, sha string, lastTag, startTime, uuid string) error {
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
	default:
		h.RefBranch = matches[0][1]
	}
	h.CurrentBranch = sha
	h.log.Info().Msgf("current testing branch: %s, reference branch: %s", h.CurrentBranch, h.RefBranch)
	err = extractPayload(h, sha, startTime, uuid)
	return err
}

func (h *handler) checkStatus(dispatchName, org, repo, pr, commit, uuid string) (string, error) {
	sb, err :=  h.metaStore.GetCopy([]string{dispatchName, org, repo, pr, commit, uuid})
	if err == badger.ErrKeyNotFound {
		return "NOT_FOUND", nil
	}
	if err != nil {
		return "", err
	}
	return string(sb), nil
}
func (h *handler) waitToReady(wg *sync.WaitGroup, branch, id string) {
	wg.Add(1)
	defer wg.Done()
	// wait for 30min until docker is ready
	err := provider.RetryUntilTrue(fmt.Sprintf("build-%s", branch), 180, func()(bool, error) {
		status, err := h.checkStatus("build-docker", h.gc.Owner, h.gc.Repo, h.Payloads["PR_NUMBER"], h.Payloads["LAST_COMMIT_SHA"], id)
		if err != nil {
			return false, err
		}
		h.log.Debug().Msgf("current docker build status: %s", status)
		return status == "SUCCESS", nil
	})
	if err != nil {
		h.log.Error().Msgf("docker build failed repository dispatch on branch %s, %s", h.Payloads["CURRENT_BRANCH"], err.Error())
	}
}

func extractPayload(h *handler, sha, start, uuid string) error {
	h.Payloads["CURRENT_BRANCH"] = h.CurrentBranch
	h.Payloads["PR_NUMBER"] = strconv.Itoa(h.gc.Pr)
	h.Payloads["LAST_COMMIT_SHA"] = sha
	h.Payloads["REF_BRANCH"] = h.RefBranch
	h.Payloads["START_TIME"] = start // for reverse lookup
	h.Payloads["UUID"] = uuid
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

	// define the current branch we want to test
	CurrentBranch string

	// define a series of client-payloads that will be posted to workflow
	Payloads map[string]string

	// define external database store metadata
	metaStore *utils.MetaStore
}

func newRunPerf(e *github.IssueCommentEvent, log zerolog.Logger, client *plugins.Agent) (*handler, error) {
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
		metaStore: client.MetaStore,
	}, nil
}
