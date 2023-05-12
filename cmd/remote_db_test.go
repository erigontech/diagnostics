package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type mockUiHandler struct {
	mock.Mock
}

type remoteCursorDep struct {
	uih            *mockUiHandler
	requestChannel chan *NodeRequest
}

func (muh *mockUiHandler) fetch(url string, requestChannel chan *NodeRequest) (bool, string) {
	args := muh.Called(url, requestChannel)
	return args.Bool(0), args.String(1)
}
func (muh *mockUiHandler) extractMultilineResult(result string) ([]string, error) {
	args := muh.Called(result)
	return args.Get(0).([]string), args.Error(1)
}

func TestInit(t *testing.T) {
	var (
		db                = "testDb"
		table             = "testTable"
		initialKey []byte = nil
		dbPath            = fmt.Sprintf("/full/path/%s", db)
		lineKey           = "4163636f756e74486973746f7279496e646578"
		lineValue         = "000000000034f9b2"
		depError          = fmt.Errorf("error")
	)

	tt := []struct {
		name       string
		on         func(*remoteCursorDep)
		assert     func(rc *RemoteCursor)
		wantErrMsg string
	}{
		{
			name: "should successfully initialize remote cursor",
			on: func(df *remoteCursorDep) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)
				tableLine := fmt.Sprintf("%s | %s", lineKey, lineValue)
				tableLinesResult := fmt.Sprintf("SUCCESS\n%s", tableLine)

				df.uih.On("fetch", "/db/list\n", df.requestChannel).Return(true, dbListResult)
				df.uih.On("extractMultilineResult", dbListResult).Return([]string{dbPath}, nil)
				df.uih.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey), df.requestChannel).Return(true, tableLinesResult)
				df.uih.On("extractMultilineResult", tableLinesResult).Return([]string{tableLine}, nil)
			},
			assert: func(rc *RemoteCursor) {
				assert.Equal(t, dbPath, rc.dbPath)
				assert.Equal(t, table, rc.table)
				assert.Equal(t, []string{fmt.Sprintf("%s | %s", lineKey, lineValue)}, rc.lines)
			},
		},
		{
			name: "should return database not found error",
			on: func(df *remoteCursorDep) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", "notFoundDb")

				df.uih.On("fetch", "/db/list\n", df.requestChannel).Return(true, dbListResult)
				df.uih.On("extractMultilineResult", dbListResult).Return([]string{"notFoundDb"}, nil)

			},
			wantErrMsg: fmt.Sprintf("database %s not found: %s", db, fmt.Sprintf("SUCCESS\n/full/path/%s", "notFoundDb")),
		},
		{
			name: "should return unable to fetch database list error",
			on: func(df *remoteCursorDep) {
				df.uih.On("fetch", "/db/list\n", df.requestChannel).Return(false, depError.Error())
			},
			wantErrMsg: fmt.Sprintf("unable to fetch database list: %s", depError.Error()),
		},
		{
			name: "should return error when db list multiline result can not be parsed",
			on: func(df *remoteCursorDep) {
				dbPathResult := fmt.Sprintf("FAILURE\n/full/path/%s", db)

				df.uih.On("fetch", "/db/list\n", df.requestChannel).Return(true, dbPathResult)
				df.uih.On("extractMultilineResult", dbPathResult).Return([]string{}, depError)
			},
			wantErrMsg: depError.Error(),
		},
		{
			name: "should return error reading table error",
			on: func(df *remoteCursorDep) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)

				df.uih.On("fetch", "/db/list\n", df.requestChannel).Return(true, dbListResult)
				df.uih.On("extractMultilineResult", dbListResult).Return([]string{dbPath}, nil)
				df.uih.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey), df.requestChannel).Return(false, "")
			},
			wantErrMsg: fmt.Sprintf("reading %s table: %s", table, ""),
		},
		{
			name: "should return error when table read multiline result can not be parsed",
			on: func(df *remoteCursorDep) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)
				tableLinesResult := fmt.Sprintf("FAILURE\n%s", "")

				df.uih.On("fetch", "/db/list\n", df.requestChannel).Return(true, dbListResult)
				df.uih.On("extractMultilineResult", dbListResult).Return([]string{dbPath}, nil)
				df.uih.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey), df.requestChannel).Return(true, tableLinesResult)
				df.uih.On("extractMultilineResult", tableLinesResult).Return([]string{}, depError)
			},
			wantErrMsg: depError.Error(),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			uih := &mockUiHandler{}
			requestChannel := make(chan *NodeRequest)

			rc := NewRemoteCursor(uih, requestChannel)
			if tc.on != nil {
				df := &remoteCursorDep{
					uih:            uih,
					requestChannel: requestChannel,
				}

				tc.on(df)
			}

			err := rc.Init(db, table, initialKey)

			if tc.wantErrMsg != "" {
				assert.EqualErrorf(t, err, tc.wantErrMsg, "expected error %q, got %s", tc.wantErrMsg, err)
				return
			}

			tc.assert(rc)
		})
	}
}
