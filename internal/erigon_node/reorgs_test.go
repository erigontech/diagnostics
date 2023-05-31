package erigon_node

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ledgerwatch/diagnostics/internal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReorgs(t *testing.T) {
	var (
		db                = "testDb"
		table             = "testTable"
		initialKey []byte = nil
		dbPath            = fmt.Sprintf("/full/path/%s", db)
		lineKey           = "111111111"
		lineValue         = "222222222"
	)

	tt := []struct {
		name       string
		on         func(*remoteCursorDependencies)
		assert     func(map[uint64][]byte, []uint64, []error)
		wantErrMsg string
	}{
		{
			name: "no reorgs found",
			on: func(df *remoteCursorDependencies) {
				dbListResult := fmt.Sprintf("SUCCESS\n/full/path/%s", db)

				lineKey = hex.EncodeToString([]byte(lineKey))
				lineValue = hex.EncodeToString([]byte(lineValue))
				tableLine := fmt.Sprintf("%s | %s", lineKey, lineValue)
				tableLinesResult := fmt.Sprintf("SUCCESS\n%s", tableLine)

				df.remoteApi.On("fetch", "/db/list\n", df.requestChannel).
					Return(true, dbListResult).Once()
				df.remoteApi.On("getResultLines", dbListResult).
					Return([]string{dbPath}, nil).Once()
				df.remoteApi.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", dbPath, table, initialKey), df.requestChannel).
					Return(true, tableLinesResult).Once()
				df.remoteApi.On("getResultLines", tableLinesResult).
					Return([]string{tableLine}, nil).Once()
				df.remoteApi.On("fetch", fmt.Sprintf("/db/read?path=%s&table=%s&key=%s\n", dbPath, table, "313131313131313132"), df.requestChannel).
					Return(true, tableLinesResult)
				df.remoteApi.On("getResultLines", tableLinesResult).
					Return([]string{}, nil).Once()
			},
			assert: func(total map[uint64][]byte, wrongBlocks []uint64, errors []error) {
				assert.Empty(t, errors)
				assert.Len(t, wrongBlocks, 0)
				assert.Len(t, total, 1)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			remoteApi := &mockNodeClientReader{}
			requestChannel := make(chan *internal.NodeRequest)
			rc := NewRemoteCursor(remoteApi, requestChannel)

			if tc.on != nil {
				df := &remoteCursorDependencies{
					remoteApi:      remoteApi,
					requestChannel: requestChannel,
				}

				tc.on(df)
			}

			err := rc.Init(db, table, initialKey)

			if tc.wantErrMsg != "" {
				assert.EqualErrorf(t, err, tc.wantErrMsg, "expected error %q, got %s", tc.wantErrMsg, err)
				return
			}

			handler := NodeClient{}
			tc.assert(handler.findReorgsInternally(context.Background(), nil, rc))
		})
	}
}
