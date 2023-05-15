package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetResultLines(t *testing.T) {
	tt := []struct {
		name       string
		result     string
		assert     func(lines []string)
		wantErrMsg string
	}{
		{
			name:   "should successfully get data lines from multi-line result",
			result: "SUCCESS\nfirst_line",
			assert: func(lines []string) {
				assert.Equal(t, []string{"first_line"}, lines)
			},
		},
		{
			name:   "should remove the last empty line from the result",
			result: "SUCCESS\nfirst_line\n",
			assert: func(lines []string) {
				assert.Equal(t, []string{"first_line"}, lines)
			},
		},
		{
			name:       "should return first line needs to be SUCCESS error",
			result:     "FAILURE\nfirst_line",
			wantErrMsg: fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %s", "FAILURE\nfirst_line"),
		},
		{
			name:       "should return first line needs to be SUCCESS error when no lines are returned",
			result:     "",
			wantErrMsg: fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %s", ""),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			remoteApi := &RemoteApi{}

			syncStageProgress, err := remoteApi.getResultLines(tc.result)

			if tc.wantErrMsg != "" {
				assert.EqualErrorf(t, err, tc.wantErrMsg, "expected error %q, got %s", tc.wantErrMsg, err)
				return
			}

			tc.assert(syncStageProgress)
		})
	}
}
