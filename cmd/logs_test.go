package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessResponseLogPart(t *testing.T) {

	tt := []struct {
		name    string
		result  string
		success bool
		exp     LogPart
	}{
		{
			name:    "should update LogPart with Error containing result value",
			result:  "FAILED\nwhatever reason for it",
			success: false,
			exp: LogPart{
				Error: "FAILED\nwhatever reason for it",
			},
		},
		{
			name:    "should update LogPart with all lines when 'SUCCESS' is not present as a first one",
			result:  "FAILED\nwhatever reason",
			success: true,
			exp: LogPart{
				Success: true,
				Lines:   []string{"FAILED", "whatever reason"},
			},
		},
		{
			name:    "should successfully process result of expected format",
			result:  "SUCCESS\nsth=value",
			success: true,
			exp: LogPart{
				Success: true,
				Lines:   []string{"sth=value"},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var lp LogPart
			lp.processResponse(tc.result, tc.success)

			assert.Equal(t, tc.exp, lp)
		})
	}
}

func TestProcessResponseLogList(t *testing.T) {

	tt := []struct {
		name    string
		result  string
		success bool
		exp     LogList
	}{
		{
			name:    "should contain result as error when no success",
			result:  "FAILED\nwhatever reason for it",
			success: false,
			exp: LogList{
				Error: "FAILED\nwhatever reason for it",
			},
		},
		{
			name:    "should contain result as error when too short result lines",
			result:  "SUCCESS",
			success: true,
			exp: LogList{
				Error: "incorrect response (length of lines should be at least 2): [SUCCESS]",
			},
		},
		{
			name:    "should contain result as error when no success",
			result:  "FAILED\nwhatever reason for it",
			success: true,
			exp: LogList{
				Error: fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", []string{"FAILED", "whatever reason for it"}),
			},
		},
		{
			name:    "should contain error about incorrect terms",
			result:  "SUCCESS\nincorrect terms",
			success: true,
			exp: LogList{
				Error: "incorrect response line (need to have 2 terms divided by |): incorrect terms",
			},
		},
		{
			name:    "should contain error about incorrect size",
			result:  "SUCCESS\nsth | wrongSize",
			success: true,
			exp: LogList{
				Error: "incorrect size: wrongSize",
			},
		},
		{
			name:    "should process data with no error",
			result:  "SUCCESS\nsomeFileName | 1000",
			success: true,
			exp: LogList{
				Success: true,
				List: []LogListItem{
					{
						Filename:    "someFileName",
						Size:        1000,
						PrintedSize: "1000B",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var ll LogList
			ll.processResponse(tc.result, tc.success)

			assert.Equal(t, tc.exp, ll)
		})
	}
}
