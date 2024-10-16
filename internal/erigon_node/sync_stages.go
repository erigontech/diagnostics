package erigon_node

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
)

// Demonstration of the working with the Erigon database remotely on the example of getting information
// about the progress of sync stages
type SyncStages struct {
	rc RemoteDbReader
}

type SyncStageProgress = map[string]string

const syncStageDb = "chaindata"
const syncStageTable = "SyncStage"
const syncProgressBase = 10

func (c *NodeClient) FindSyncStages(ctx context.Context) (SyncStageProgress, error) {
	rc := NewRemoteCursor(c)
	syncStages := &SyncStages{rc: rc}

	return syncStages.fetchSyncStageProgress(ctx)
}

func (ss *SyncStages) fetchSyncStageProgress(ctx context.Context) (SyncStageProgress, error) {
	if cursorError := ss.rc.Init(ctx, syncStageDb, syncStageTable, nil); cursorError != nil {
		return nil, fmt.Errorf("could not initialize remote cursor: %w", cursorError)
	}

	syncStageProgress := make(SyncStageProgress)

	var k, v []byte
	var e error
	for k, v, e = ss.rc.Next(ctx); e == nil && k != nil; k, v, e = ss.rc.Next(ctx) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context channel interrupted")
		default:
		}

		syncStage := string(k)
		syncProgress, unmarshalError := ss.unmarshal(v)

		if unmarshalError != nil {
			return nil, fmt.Errorf("could not unmarshal sync stage data: %w", unmarshalError)
		}

		syncStageProgress[syncStage] = strconv.FormatUint(syncProgress, syncProgressBase)
	}
	if e != nil {
		return nil, fmt.Errorf("could not process remote cursor line: %w", e)
	}

	return syncStageProgress, nil
}

func (ss *SyncStages) unmarshal(data []byte) (uint64, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("value must be at least 8 bytes, got %d", len(data))
	}
	return binary.BigEndian.Uint64(data[:8]), nil
}
