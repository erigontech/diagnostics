package erigon_node

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockNodeClientReader struct {
	mock.Mock
}

func (ra *mockNodeClientReader) Version(ctx context.Context) (Versions, error) {

	panic("TODO")
}

func (ra *mockNodeClientReader) Flags(ctx context.Context) (Flags, error) {

	panic("TODO")
}

func (ra *mockNodeClientReader) CMDLineArgs(ctx context.Context) (CmdLineArgs, error) {

	panic("TODO")
}

func (ra *mockNodeClientReader) FindSyncStages(ctx context.Context) (SyncStageProgress, error) {

	panic("TODO")
}

func (ra *mockNodeClientReader) BodiesDownload(ctx context.Context, w http.ResponseWriter) {

	panic("TODO")
}

func (ra *mockNodeClientReader) HeadersDownload(ctx context.Context, w http.ResponseWriter) {

	panic("TODO")
}

func (ra *mockNodeClientReader) LogFiles(ctx context.Context) (LogFiles, error) {
	return nil, fmt.Errorf("TODO")
}

func (ra *mockNodeClientReader) Log(ctx context.Context, w http.ResponseWriter, file string, offset int64, size int64, download bool) error {
	return fmt.Errorf("TODO")
}

func (c *mockNodeClientReader) DBs(ctx context.Context) (DBs, error) {
	panic("TODO")
}

func (c *mockNodeClientReader) Tables(ctx context.Context, db string) (Tables, error) {
	panic("TODO")
}

func (c *mockNodeClientReader) Table(ctx context.Context, db string, table string) (Results, error) {
	panic("TODO")
}

func (ra *mockNodeClientReader) fetch(ctx context.Context, url string, params url.Values) (*NodeRequest, error) {
	panic("TODO")
	//args := ra.Called(url)
	//return args.Bool(0), args.String(1)
}

/*func (ra *mockNodeClientReader) getResultLines(result string) ([]string, error) {
	args := ra.Called(result)
	return args.Get(0).([]string), args.Error(1)
}*/

func (ra *mockNodeClientReader) FindReorgs(ctx context.Context, w http.ResponseWriter) (Reorg, error) {
	panic("TODO")
}

func (ra *mockNodeClientReader) FindPeers(ctx context.Context) (PeersInfo, error) {
	panic("TODO")
}

func (ra *mockNodeClientReader) Bootnodes(ctx context.Context) (Bootnodes, error) {
	panic("TODO")
}

type remoteCursorDependencies struct {
	remoteApi *mockNodeClientReader
}

func TestInit(t *testing.T) {
	var (
		db                     = "testDb"
		table                  = "testTable"
		initialKey      []byte = nil
		dbPath                 = fmt.Sprintf("/full/path/%s", db)
		lineKey                = "lineKey"
		lineValue              = "lineValue"
		dependencyError        = fmt.Errorf("error")
	)

	tt := []struct {
		name       string
		on         func(*remoteCursorDependencies)
		assert     func(rc *RemoteCursor)
		wantErrMsg string
	}{
		{
			name: "should successfully initialize remote cursor",
			on: func(df *remoteCursorDependencies) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)
				tableLine := fmt.Sprintf("%s | %s", lineKey, lineValue)
				tableLinesResult := fmt.Sprintf("SUCCESS\n%s", tableLine)

				df.remoteApi.On("fetch", "/db/list\n").Return(true, dbListResult)
				df.remoteApi.On("getResultLines", dbListResult).Return([]string{dbPath}, nil)
				df.remoteApi.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey)).Return(true, tableLinesResult)
				df.remoteApi.On("getResultLines", tableLinesResult).Return([]string{tableLine}, nil)
			},
			assert: func(rc *RemoteCursor) {
				assert.Equal(t, dbPath, rc.dbPath)
				assert.Equal(t, table, rc.table)
				//assert.Equal(t, []string{fmt.Sprintf("%s | %s", lineKey, lineValue)}, rc.lines)
			},
		},
		{
			name: "should return database not found error",
			on: func(df *remoteCursorDependencies) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", "notFoundDb")

				df.remoteApi.On("fetch", "/db/list\n").Return(true, dbListResult)
				df.remoteApi.On("getResultLines", dbListResult).Return([]string{"notFoundDb"}, nil)
			},
			wantErrMsg: fmt.Sprintf("database %s not found: %s", db, fmt.Sprintf("SUCCESS\n/full/path/%s", "notFoundDb")),
		},
		{
			name: "should return unable to fetch database list error",
			on: func(df *remoteCursorDependencies) {
				df.remoteApi.On("fetch", "/db/list\n").Return(false, dependencyError.Error())
			},
			wantErrMsg: fmt.Sprintf("unable to fetch database list: %s", dependencyError.Error()),
		},
		{
			name: "should return error when db list result can not be parsed",
			on: func(df *remoteCursorDependencies) {
				dbPathResult := fmt.Sprintf("FAILURE\n/full/path/%s", db)

				df.remoteApi.On("fetch", "/db/list\n").Return(true, dbPathResult)
				df.remoteApi.On("getResultLines", dbPathResult).Return([]string{}, dependencyError)
			},
			wantErrMsg: dependencyError.Error(),
		},
		{
			name: "should return reading table error",
			on: func(df *remoteCursorDependencies) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)

				df.remoteApi.On("fetch", "/db/list\n").Return(true, dbListResult)
				df.remoteApi.On("getResultLines", dbListResult).Return([]string{dbPath}, nil)
				df.remoteApi.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey)).Return(false, "")
			},
			wantErrMsg: fmt.Sprintf("reading %s table: %s", table, ""),
		},
		{
			name: "should return error when table result can not be parsed",
			on: func(df *remoteCursorDependencies) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)
				tableLinesResult := fmt.Sprintf("FAILURE\n%s", "")

				df.remoteApi.On("fetch", "/db/list\n").Return(true, dbListResult)
				df.remoteApi.On("getResultLines", dbListResult).Return([]string{dbPath}, nil)
				df.remoteApi.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey)).Return(true, tableLinesResult)
				df.remoteApi.On("getResultLines", tableLinesResult).Return([]string{}, dependencyError)
			},
			wantErrMsg: dependencyError.Error(),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			nodeClient := &mockNodeClientReader{}
			rc := NewRemoteCursor(nodeClient)

			if tc.on != nil {
				df := &remoteCursorDependencies{
					remoteApi: nodeClient,
				}

				tc.on(df)
			}

			err := rc.Init(context.Background(), db, table, initialKey)

			if tc.wantErrMsg != "" {
				assert.EqualErrorf(t, err, tc.wantErrMsg, "expected error %q, got %s", tc.wantErrMsg, err)
				return
			}

			tc.assert(rc)
		})
	}
}
