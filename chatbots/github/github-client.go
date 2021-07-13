// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v35/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	Clt               *github.Client
	Owner             string
	Repo              string
	Pr                int
	Author            string
	CommentBody       string
	AuthorAssociation string
	State             string
	LastTag           string
	Ctx               context.Context
}

func NewGithubClient(ctx context.Context, e *github.IssueCommentEvent) (*GithubClient, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		return nil, fmt.Errorf("env var missing")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return &GithubClient{
		Clt:               github.NewClient(tc),
		Owner:             *e.GetRepo().Owner.Login,
		Repo:              *e.GetRepo().Name,
		Pr:                *e.GetIssue().Number,
		Author:            *e.Sender.Login,
		AuthorAssociation: *e.GetComment().AuthorAssociation,
		CommentBody:       *e.GetComment().Body,
		Ctx:               ctx,
		State:             e.GetIssue().GetState(),
	}, nil
}

func (c GithubClient) PostComment(commentBody string) error {
	issueComment := &github.IssueComment{Body: github.String(commentBody)}
	_, _, err := c.Clt.Issues.CreateComment(c.Ctx, c.Owner, c.Repo, c.Pr, issueComment)
	return err
}

func (c GithubClient) GetIssueState() string {
	return c.State
}

func (c GithubClient) CreateLabel(labelName string) error {
	benchmarkLabel := []string{labelName}
	_, _, err := c.Clt.Issues.AddLabelsToIssue(c.Ctx, c.Owner, c.Repo, c.Pr, benchmarkLabel)
	return err
}

func (c GithubClient) GetLastCommitSHA() string {
	// https://developer.github.com/v3/pulls/#list-commits-on-a-pull-request
	listops := &github.ListOptions{Page: 1, PerPage: 250}
	l, _, _ := c.Clt.PullRequests.ListCommits(c.Ctx, c.Owner, c.Repo, c.Pr, listops)
	if len(l) == 0 {
		return ""
	}
	return l[len(l)-1].GetSHA()
}

func (c GithubClient) CreateRepositoryDispatch(eventType string, clientPayload map[string]string) error {
	allArgs, err := json.Marshal(clientPayload)
	if err != nil {
		return fmt.Errorf("%v: could not encode client payload", err)
	}
	cp := json.RawMessage(string(allArgs))

	rd := github.DispatchRequestOptions{
		EventType:     eventType,
		ClientPayload: &cp,
	}

	log.Printf("creating repository_dispatch with payload: %v", string(allArgs))
	_, _, err = c.Clt.Repositories.Dispatch(c.Ctx, c.Owner, c.Repo, rd)
	return err
}

func (c GithubClient) GetLatestTag() (string, error) {
	listops := &github.ListOptions{Page: 1, PerPage: 250}
	tags, _, err := c.Clt.Repositories.ListTags(context.Background(), c.Owner, c.Repo, listops)
	if err != nil {
		return "", err
	}
	if len(tags) > 0 {
		c.LastTag = tags[0].GetName()
	} else {
		return "", fmt.Errorf("%s owned by %s has no tags", c.Repo, c.Owner)
	}
	return c.LastTag, nil
}

func GetActionStatus(ctx context.Context, owner, repo string, run_id int64) (*github.WorkflowRun, error) {
	clt, err := buildClient(ctx)
	if err != nil {
		return nil, err
	}
	w, _, err := clt.Actions.GetWorkflowRunByID(context.Background(), owner, repo, run_id)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func buildClient(ctx context.Context) (*github.Client, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		return nil, fmt.Errorf("env var missing")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}