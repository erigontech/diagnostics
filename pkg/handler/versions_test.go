package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessResponseVersions(t *testing.T) {
	tt := []struct {
		name    string
		result  string
		success bool
		exp     Versions
	}{
		{
			name:    "should contain result as error when no success",
			result:  "FAILED\nwhatever reason for it",
			success: false,
			exp: Versions{
				Error: "FAILED\nwhatever reason for it",
			},
		},
		{
			name:    "should contain result as error when too short result lines",
			result:  "SUCCESS",
			success: true,
			exp: Versions{
				Error: fmt.Sprintf("incorrect response (at least node version needs to be present): %v", []string{"SUCCESS"}),
			},
		},
		{
			name:    "should contain result as error when first line is not SUCCESS",
			result:  "FAILED\nwhatever reason for it",
			success: true,
			exp: Versions{
				Error: fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", []string{"FAILED", "whatever reason for it"}),
			},
		},
		{
			name:    "should contain error related to parsing version",
			result:  "SUCCESS\nv2",
			success: true,
			exp: Versions{
				Error: fmt.Sprintf("failed parsing node version: strconv.ParseUint: parsing \"v2\": invalid syntax"),
			},
		},
		{
			name:    "should contain node version",
			result:  "SUCCESS\n2",
			success: true,
			exp: Versions{
				Success:     true,
				NodeVersion: 2,
			},
		},
		{
			name:    "should contain node version and code related data (code version and commit) and ignore the rest",
			result:  "SUCCESS\n2\ncodeVersionData\ngitCommitData\ntheRestShouldBeIgnored\nReally",
			success: true,
			exp: Versions{
				Success:     true,
				NodeVersion: 2,
				CodeVersion: "codeVersionData",
				GitCommit:   "gitCommitData",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var ver Versions
			ver.processResponse(tc.result, tc.success)

			assert.Equal(t, tc.exp, ver)
		})
	}
}
