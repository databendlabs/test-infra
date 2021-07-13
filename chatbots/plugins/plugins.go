// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package plugins

import (
	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	githubcli "datafuselabs/test-infra/chatbots/github"
	"datafuselabs/test-infra/chatbots/utils"
)

var (
	IssueCommentHandlers = map[string]IssueCommentHandler{}
)

type Agent struct {
	GithubClient *githubcli.GithubClient
	Logger       zerolog.Logger
	Store        *utils.StorageInterface
	MetaStore 	*utils.MetaStore
}

// IssueCommentHandler defines the function contract for a github.IssueCommentEvent handler.
type IssueCommentHandler func(*Agent, *github.IssueCommentEvent) error

func NewAgent(gitClient *githubcli.GithubClient, metaStore *utils.MetaStore) *Agent {
	return &Agent{
		GithubClient: gitClient,
		Logger:       log.With().Str("test-infra", "agent").Logger(),
		MetaStore: metaStore,
	}
}
func RegisterIssueCommentHandler(name string, fn IssueCommentHandler) {
	IssueCommentHandlers[name] = fn
}
