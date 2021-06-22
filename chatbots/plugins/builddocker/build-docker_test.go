// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package builddocker

import (
	"fmt"
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
		log:      log.With().Str("issue comment", "build-docker").Logger(),
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
		lastTag           string
		expectError       error
		expectRefBranch   string
		expectPayload     map[string]string
	}{
		{
			name:              "master",
			comment:           "/build-docker master",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "foo",
			expectError:       nil,
			lastTag:           "v1.1.1-nightly",
			expectRefBranch:   "master",
			expectPayload:     map[string]string{"REF": "master", "LAST_COMMIT_SHA": "foo"},
		},
		{
			name:              "master",
			comment:           "/build-docker current",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			sha:               "foo",
			expectError:       nil,
			lastTag:           "v1.1.1-nightly",
			expectRefBranch:   "foo",
			expectPayload:     map[string]string{"REF": "foo", "LAST_COMMIT_SHA": "foo"},
		},
		{
			name:              "newline",
			comment:           "\r\n/build-docker latest\t",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			lastTag:           "v1.1.1-nightly",
			sha:               "bar",
			expectError:       nil,
			expectRefBranch:   "v1.1.1-nightly",
			expectPayload:     map[string]string{"REF": "v1.1.1-nightly", "LAST_COMMIT_SHA": "bar"},
		},
		{
			name:              "laetst",
			comment:           "/build-docker latest",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			lastTag:           "v1.1.1-nightly",
			sha:               "bar",
			expectError:       nil,
			expectRefBranch:   "v1.1.1-nightly",
			expectPayload:     map[string]string{"REF": "v1.1.1-nightly", "LAST_COMMIT_SHA": "bar"},
		},
		{
			name:              "release",
			comment:           "/build-docker v1.2.3-nightly",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			pr:                233,
			lastTag:           "v1.1.1-nightly",
			sha:               "bar",
			expectError:       nil,
			expectRefBranch:   "v1.2.3-nightly",
			expectPayload:     map[string]string{"REF": "v1.2.3-nightly", "LAST_COMMIT_SHA": "bar"},
		},
		{
			name:              "empty",
			comment:           "/build-docker",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			expectError:       fmt.Errorf("there is no matching regex"),
			pr:                233,
			sha:               "bar",
			expectRefBranch:   "",
			expectPayload:     map[string]string{},
		},
		{
			name:              "non-sense",
			comment:           "/build-docker Wubba-Lubba-Dub-Dub",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			expectError:       fmt.Errorf("there is no matching regex"),
			pr:                233,
			sha:               "bar",
			expectRefBranch:   "",
			expectPayload:     map[string]string{},
		},
		{
			name:              "non-sense2",
			comment:           "/build-docker Wubba Lubba Dub Dub",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			expectError:       fmt.Errorf("there is no matching regex"),
			pr:                233,
			sha:               "bar",
			expectRefBranch:   "",
			expectPayload:     map[string]string{},
		},
		{
			name:              "non-sense3",
			comment:           "/build-dockermaster main",
			author:            "zhihanz",
			authorAssociation: "OWNER",
			expectError:       fmt.Errorf("there is no matching regex"),
			pr:                233,
			sha:               "bar",
			expectRefBranch:   "",
			expectPayload:     map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newFakeHandler(tt.comment, tt.author, tt.authorAssociation, tt.pr)
			assert.Equal(t, handlerhelper(handler, tt.sha, tt.lastTag), tt.expectError)
			assert.Equal(t, handler.RefBranch, tt.expectRefBranch)
			assert.Equal(t, handler.Payloads, tt.expectPayload)
		})
	}
}
