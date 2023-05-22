package erigon_node

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type mockRemoteCursor struct {
	mock.Mock
}

type syncStagesDep struct {
	rc *mockRemoteCursor
}

func (mrc *mockRemoteCursor) Init(db string, table string, initialKey []byte) error {
	args := mrc.Called(db, table, initialKey)
	return args.Error(0)
}
func (mrc *mockRemoteCursor) Next() ([]byte, []byte, error) {
	args := mrc.Called()
	return args.Get(0).([]byte), args.Get(1).([]byte), args.Error(2)
}

func encodeBigEndian(n uint64) []byte {
	if n == 0 {
		return []byte{}
	}

	var v [8]byte
	binary.BigEndian.PutUint64(v[:], n)
	return v[:]
}

func TestFetchSyncStageProgress(t *testing.T) {
	var (
		firstStageName      = []byte("first_stage")
		firstStageProgress  = uint64(3458658)
		secondStageName     = []byte("second_stage")
		secondStageProgress = uint64(0)
		db                  = "chaindata"
		table               = "SyncStage"
		depError            = fmt.Errorf("error")
	)

	tt := []struct {
		name       string
		ctx        context.Context
		on         func(*syncStagesDep)
		assert     func(ssp SyncStageProgress)
		wantErrMsg string
	}{
		{
			name: "should successfully fetch and return sync stages with progress",
			ctx:  context.Background(),
			on: func(df *syncStagesDep) {
				df.rc.On("Init", db, table, []byte(nil)).Return(nil)
				df.rc.On("Next").Return(firstStageName, encodeBigEndian(firstStageProgress), nil).Once()
				df.rc.On("Next").Return(secondStageName, encodeBigEndian(secondStageProgress), nil).Once()
				df.rc.On("Next").Return([]byte(nil), []byte(nil), nil).Once()
			},
			assert: func(ssp SyncStageProgress) {
				exp := SyncStageProgress{
					string(firstStageName):  fmt.Sprintf("%d", firstStageProgress),
					string(secondStageName): fmt.Sprintf("%d", secondStageProgress),
				}

				assert.Equal(t, exp, ssp)
			},
		},
		{
			name: "should return could not initialize remote cursor error",
			ctx:  context.Background(),
			on: func(df *syncStagesDep) {
				df.rc.On("Init", db, table, []byte(nil)).Return(depError)
			},
			wantErrMsg: fmt.Sprintf("could not initialize remote cursor: %v", depError),
		},
		{
			name: "should return ctx channel interrupted error",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			on: func(df *syncStagesDep) {
				df.rc.On("Init", db, table, []byte(nil)).Return(nil)
				df.rc.On("Next").Return(firstStageName, encodeBigEndian(firstStageProgress), nil).Once()
			},
			wantErrMsg: "context channel interrupted",
		},
		{
			name: "should return could not unmarshal sync stage data error",
			ctx:  context.Background(),
			on: func(df *syncStagesDep) {
				df.rc.On("Init", db, table, []byte(nil)).Return(nil)
				df.rc.On("Next").Return(firstStageName, []byte{1}, nil).Once()
			},
			wantErrMsg: "could not unmarshal sync stage data: value must be at least 8 bytes, got 1",
		},
		{
			name: "should return could not process remote cursor line error",
			ctx:  context.Background(),
			on: func(df *syncStagesDep) {
				df.rc.On("Init", db, table, []byte(nil)).Return(nil)
				df.rc.On("Next").Return([]byte{}, []byte{}, depError).Once()
			},
			wantErrMsg: fmt.Sprintf("could not process remote cursor line: %v", depError),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rc := &mockRemoteCursor{}
			syncStages := SyncStages{rc: rc}
			if tc.on != nil {
				df := &syncStagesDep{
					rc: rc,
				}

				tc.on(df)
			}

			syncStageProgress, err := syncStages.fetchSyncStageProgress(tc.ctx)

			if tc.wantErrMsg != "" {
				assert.EqualErrorf(t, err, tc.wantErrMsg, "expected error %q, got %s", tc.wantErrMsg, err)
				return
			}

			tc.assert(syncStageProgress)
		})
	}
}
