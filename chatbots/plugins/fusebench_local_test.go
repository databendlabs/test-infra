package plugins

import (
	"strconv"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	githubcli "datafuselabs/test-infra/chatbots/github"
)

func newFakeGithubClient(commentBody, author, authorAssociation string, pr_num int) *githubcli.GithubClient {
	return &githubcli.GithubClient{
		AuthorAssociation: authorAssociation,
		CommentBody:       commentBody,
		Author:            author,
		Pr:                pr_num,
	}
}

func newFakeHandler(commentBody, author, authorAssociation string, pr int) *handler {
	return &handler{
		regexp:   reg,
		gc:       newFakeGithubClient(commentBody, author, authorAssociation, pr),
		log:      log.With().Str("issue comment", "fusebench-local").Logger(),
		Payloads: map[string]string{},
	}
}

func Test_handlerhelper(t *testing.T) {
	tests := []struct {
		name              string
		comment           string
		author            string
		authorAssociation string
		pr                int
		sha               string
		expectBranch      string
		expectPayload     map[string]string
	}{
		{
			name:              "master",
			comment:           "/fusebench-local master",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "foo",
			expectBranch:      "master",
			expectPayload:     map[string]string{"BranchName": "master", "PR_NUMBER": strconv.Itoa(233), "LAST_COMMIT_SHA": "foo"},
		},
		{
			name:              "newline",
			comment:           "\r\n/fusebench-local current\t",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "bar",
			expectPayload:     map[string]string{"BranchName": "bar", "PR_NUMBER": strconv.Itoa(233), "LAST_COMMIT_SHA": "bar"},
		},
		{
			name:              "current",
			comment:           "/fusebench-local current",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "bar",
			expectPayload:     map[string]string{"BranchName": "bar", "PR_NUMBER": strconv.Itoa(233), "LAST_COMMIT_SHA": "bar"},
		},
		{
			name:              "release",
			comment:           "/fusebench-local v1.2.3-nightly",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "v1.2.3-nightly",
			expectPayload:     map[string]string{"BranchName": "v1.2.3-nightly", "PR_NUMBER": strconv.Itoa(233), "LAST_COMMIT_SHA": "bar"},
		},
		{
			name:              "empty",
			comment:           "/fusebench-local",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "",
			expectPayload:     map[string]string{},
		},
		{
			name:              "non-sense",
			comment:           "/fusebench-local Wubba-Lubba-Dub-Dub",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "",
			expectPayload:     map[string]string{},
		},
		{
			name:              "non-sense2",
			comment:           "/fusebench-local Wubba Lubba Dub Dub",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "",
			expectPayload:     map[string]string{},
		},
		{
			name:              "non-sense3",
			comment:           "/fusebench-local master main",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "bar",
			expectBranch:      "",
			expectPayload:     map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newFakeHandler(tt.comment, tt.author, tt.authorAssociation, tt.pr)
			assert.NoError(t, handlerhelper(handler, tt.sha))
			assert.Equal(t, handler.BranchName, tt.expectBranch)
			assert.Equal(t, handler.Payloads, tt.expectPayload)
		})
	}
}
