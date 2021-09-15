// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package runperf

import (
	"fmt"
	"regexp"
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

func newFakeHandler(commentBody, author, authorAssociation string, pr int, region, bucket, endpoint string ) *handler {
	return &handler{
		regexp:   map[string]*regexp.Regexp{"run_perf": reg, "rerun_perf_all": rerunAll, "rerun_perf": rerun},
		gc:       newFakeGithubClient(commentBody, author, authorAssociation, pr),
		log:      log.With().Str("issue comment", "bendbench-local").Logger(),
		Region: region,
		Bucket: bucket,
		Endpoint: endpoint,
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
		expectName string
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
			expectName: "run_perf",
			expectError:         nil,
			lastTag:             "v1.1.1-nightly",
			expectCurrentBranch: "foo",
			StartTime: "1",
			UUID: "12",
			expectRefBranch:     "master",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "foo", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "foo", "REF_BRANCH": "master", "UUID": "12",
				"REGION": "foo-region", "BUCKET": "foo-bucket", "ENDPOINT": "foo-endpoint"},
		},
		{
			name:                "master",
			comment:             "/rerun-perf-all master",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			sha:                 "foo",
			expectName: "rerun_perf_all",
			expectError:         nil,
			lastTag:             "v1.1.1-nightly",
			expectCurrentBranch: "foo",
			StartTime: "1",
			UUID: "12",
			expectRefBranch:     "master",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "foo", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "foo", "REF_BRANCH": "master", "UUID": "12",
				"REGION": "foo-region", "BUCKET": "foo-bucket", "ENDPOINT": "foo-endpoint"},
		},
		{
			name:                "master",
			comment:             "/rerun-perf master",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			sha:                 "foo",
			expectName: "rerun_perf",
			expectError:         nil,
			lastTag:             "v1.1.1-nightly",
			expectCurrentBranch: "foo",
			StartTime: "1",
			UUID: "12",
			expectRefBranch:     "master",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "foo", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "foo", "REF_BRANCH": "master", "UUID": "12",
				"REGION": "foo-region", "BUCKET": "foo-bucket", "ENDPOINT": "foo-endpoint"},
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
			expectName: "run_perf",
			expectError:         nil,
			StartTime: "1",
			expectCurrentBranch: "bar",
			expectRefBranch:     "v1.1.1-nightly",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "bar", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "bar", "REF_BRANCH": "v1.1.1-nightly", "UUID": "12",
				"REGION": "foo-region", "BUCKET": "foo-bucket", "ENDPOINT": "foo-endpoint"},
		},
		{
			name:                "laetst",
			comment:             "/run-perf latest",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			pr:                  233,
			lastTag:             "v1.1.1-nightly",
			sha:                 "bar",
			expectName: "run_perf",
			expectError:         nil,
			StartTime: "1",
			UUID: "12",
			expectCurrentBranch: "bar",
			expectRefBranch:     "v1.1.1-nightly",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "bar", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "bar", "REF_BRANCH": "v1.1.1-nightly", "UUID": "12",
				"REGION": "foo-region", "BUCKET": "foo-bucket", "ENDPOINT": "foo-endpoint"},
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
			expectName: "run_perf",
			expectError:         nil,
			expectCurrentBranch: "bar",
			expectRefBranch:     "v1.2.3-nightly",
			expectPayload:       map[string]string{"CURRENT_BRANCH": "bar", "PR_NUMBER": strconv.Itoa(233),
				"LAST_COMMIT_SHA": "bar", "REF_BRANCH": "v1.2.3-nightly", "UUID": "12",
				"REGION": "foo-region", "BUCKET": "foo-bucket", "ENDPOINT": "foo-endpoint"},
		},
		{
			name:                "empty",
			comment:             "/run-perf-local",
			author:              "zhihanz",
			authorAssociation:   "OWNER",
			expectName: "",
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
			expectName: "",
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
			expectName: "",
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
			expectName: "",
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
			handler := newFakeHandler(tt.comment, tt.author, tt.authorAssociation, tt.pr, "foo-region", "foo-bucket", "foo-endpoint")
			name, err := handlerhelper(handler, tt.sha, tt.lastTag, tt.StartTime, tt.UUID)
			assert.Equal(t, name, tt.expectName)
			assert.Equal(t, err, tt.expectError)
			assert.Equal(t, handler.CurrentBranch, tt.expectCurrentBranch)
			assert.Equal(t, handler.RefBranch, tt.expectRefBranch)
			assert.Equal(t, handler.Payloads, tt.expectPayload)
		})
	}
}
