// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package runperf

import (
	"fmt"
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
		name                string
		comment             string
		author              string
		authorAssociation   string
		pr                  int
		sha                 string
		StartTime 			string
		UUID string
		lastTag             string
		expectError         error
		expectCurrentBranch string
		expectRefBranch     string
		expectPayload       map[string]string
	}{
		{
			name:                "master",
			comment:             "/run-perf master",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			sha:                 "foo",
			expectError:         nil,
			lastTag:             "v1.1.1-nightly",
			expectCurrentBranch: "foo",
			StartTime: "1",
			UUID: "12",
			expectRefBranch:     "master",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "foo", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "foo", "REF_BRANCH": "master", "START_TIME": "1", "UUID": "12"},
		},
		{
			name:                "newline",
			comment:             "\r\n/run-perf latest\t",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			lastTag:             "v1.1.1-nightly",
			sha:                 "bar",
			UUID: "12",
			expectError:         nil,
			StartTime: "1",
			expectCurrentBranch: "bar",
			expectRefBranch:     "v1.1.1-nightly",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "bar", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "bar", "REF_BRANCH": "v1.1.1-nightly", "START_TIME": "1", "UUID": "12"},
		},
		{
			name:                "laetst",
			comment:             "/run-perf latest",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			lastTag:             "v1.1.1-nightly",
			sha:                 "bar",
			expectError:         nil,
			StartTime: "1",
			UUID: "12",
			expectCurrentBranch: "bar",
			expectRefBranch:     "v1.1.1-nightly",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "bar", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "bar", "REF_BRANCH": "v1.1.1-nightly", "START_TIME": "1", "UUID": "12"},
		},
		{
			name:                "release",
			comment:             "/run-perf v1.2.3-nightly",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			lastTag:             "v1.1.1-nightly",
			sha:                 "bar",
			StartTime: "1",
			UUID: "12",
			expectError:         nil,
			expectCurrentBranch: "bar",
			expectRefBranch:     "v1.2.3-nightly",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "bar", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "bar", "REF_BRANCH": "v1.2.3-nightly", "START_TIME": "1", "UUID": "12"},
		},
		{
			name:                "empty",
			comment:             "/run-perf-local",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			expectError:         fmt.Errorf("there is no matching regex"),
			pr:                  233,
			StartTime: "1",
			sha:                 "bar",
			expectCurrentBranch: "",
			expectRefBranch:     "",
			expectPayload:       map[string]string{},
		},
		{
			name:                "non-sense",
			comment:             "/run-perf Wubba-Lubba-Dub-Dub",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			expectError:         fmt.Errorf("there is no matching regex"),
			pr:                  233,
			StartTime: "1",
			sha:                 "bar",
			expectCurrentBranch: "",
			expectRefBranch:     "",
			expectPayload:       map[string]string{},
		},
		{
			name:                "non-sense2",
			comment:             "/run-perf Wubba Lubba Dub Dub",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			expectError:         fmt.Errorf("there is no matching regex"),
			pr:                  233,
			StartTime: "1",
			sha:                 "bar",
			expectCurrentBranch: "",
			expectRefBranch:     "",
			expectPayload:       map[string]string{},
		},
		{
			name:                "non-sense3",
			comment:             "/run-perf master main",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			expectError:         fmt.Errorf("there is no matching regex"),
			pr:                  233,
			StartTime: "1",
			sha:                 "bar",
			expectCurrentBranch: "",
			expectRefBranch:     "",
			expectPayload:       map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newFakeHandler(tt.comment, tt.author, tt.authorAssociation, tt.pr)
			assert.Equal(t, handlerhelper(handler, tt.sha, tt.lastTag, tt.StartTime, tt.UUID), tt.expectError)
			assert.Equal(t, handler.CurrentBranch, tt.expectCurrentBranch)
			assert.Equal(t, handler.RefBranch, tt.expectRefBranch)
			assert.Equal(t, handler.Payloads, tt.expectPayload)
		})
	}
}
