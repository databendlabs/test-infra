package labelrunperf

import (
	"regexp"
	"testing"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	githubcli "datafuselabs/test-infra/chatbots/github"
)

func newFakeGithubClient(author string) *githubcli.GithubClient {
	return &githubcli.GithubClient{
		Author: author,
	}
}

func newFakeHandler(author string) *handler {
	return &handler{
		regexp: map[string]*regexp.Regexp{"run_perf": reg, "rerun_perf_all": rerunAll, "rerun_perf": rerun},
		gc:     newFakeGithubClient(author),
		log:    log.With().Str("push", "run-perf").Logger(),
	}
}

func Test_makecomment(t *testing.T) {
	tests := []struct {
		name            string
		labelName       string
		author          string
		expectedComment string
	}{
		{
			name:            "master",
			labelName:       "run-perf master",
			author:          "zhihanz",
			expectedComment: "/run-perf master",
		},
		{
			name:            "rerun",
			labelName:       "rerun-perf master",
			author:          "zhihanz",
			expectedComment: "/rerun-perf master",
		},
		{
			name:            "version",
			labelName:       "run-perf v0.4.11-nightly",
			author:          "zhihanz",
			expectedComment: "/run-perf v0.4.11-nightly",
		},
		{
			name:            "latest",
			labelName:       "run-perf latest",
			author:          "zhihanz",
			expectedComment: "/run-perf latest",
		},
		{
			name:            "foo",
			labelName:       "foo-bar",
			author:          "zhihanz",
			expectedComment: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newFakeHandler(tt.author)
			comment := make_comment(*handler, &github.Label{Name: &tt.labelName})
			assert.Equal(t, comment, tt.expectedComment)
		})
	}
}
