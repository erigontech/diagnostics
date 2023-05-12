package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractMultilineResult(t *testing.T) {
	tt := []struct {
		name       string
		result     string
		assert     func(lines []string)
		wantErrMsg string
	}{
		{
			name:   "should successfully extract data lines from multi line result",
			result: "SUCCESS\nfirst_line",
			assert: func(lines []string) {
				assert.Equal(t, []string{"first_line"}, lines)
			},
		},
		{
			name:       "should return first line needs to be success error when first line is not success",
			result:     "FAILURE\nfirst_line",
			wantErrMsg: fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %s", "FAILURE\nfirst_line"),
		},
		{
			name:       "should return first line needs to be success error when no lines are returned",
			result:     "",
			wantErrMsg: fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %s", ""),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			remoteApi := &RemoteApi{}

			syncStageProgress, err := remoteApi.extractMultilineResult(tc.result)

			if tc.wantErrMsg != "" {
				assert.EqualErrorf(t, err, tc.wantErrMsg, "expected error %q, got %s", tc.wantErrMsg, err)
				return
			}

			tc.assert(syncStageProgress)
		})
	}
}
