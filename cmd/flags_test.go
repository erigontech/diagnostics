package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessResponseFlags(t *testing.T) {

	tt := []struct {
		name    string
		result  string
		success bool
		exp     Flags
	}{
		{
			name:    "should return flags with Error containing result value",
			result:  "FAILED\nwhatever reason for it",
			success: false,
			exp: Flags{
				Error:   "FAILED\nwhatever reason for it",
				Payload: map[string]string{},
			},
		},
		{
			name:    "should return early with too short result",
			result:  "",
			success: true,
			exp: Flags{
				Error:   fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", []string{}),
				Payload: map[string]string{},
			},
		},
		{
			name:    "should return early with missing SUCCESS as a first line",
			result:  "FAILED\nwhatever reason",
			success: true,
			exp: Flags{
				Error:   fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", []string{"FAILED", "whatever reason"}),
				Payload: map[string]string{},
			},
		},
		{
			name:    "should successfully process result of expected format",
			result:  "SUCCESS\nsth=value",
			success: true,
			exp: Flags{
				Success: true,
				Payload: map[string]string{
					"sth": "value",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var flags Flags
			flags.processResponse(tc.result, tc.success)

			assert.Equal(t, tc.exp, flags)
		})
	}
}
