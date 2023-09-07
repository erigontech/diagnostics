package erigon_node

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/ledgerwatch/diagnostics/internal"
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

func (c *NodeClient) FindSyncStages(ctx context.Context, requestChannel chan *internal.NodeRequest) (SyncStageProgress, error) {
	rc := NewRemoteCursor(c, requestChannel)
	syncStages := &SyncStages{rc: rc}

	syncStageProgress, err := syncStages.fetchSyncStageProgress(ctx)
	if err != nil {
		fmt.Printf("Unable to fetch sync stage progress: %v\n", err)
		return nil, err
	}

	return syncStageProgress, nil
}

func (ss *SyncStages) fetchSyncStageProgress(ctx context.Context) (SyncStageProgress, error) {
	if cursorError := ss.rc.Init(syncStageDb, syncStageTable, nil); cursorError != nil {
		return nil, fmt.Errorf("could not initialize remote cursor: %v", cursorError)
	}

	syncStageProgress := make(SyncStageProgress)

	var k, v []byte
	var e error
	for k, v, e = ss.rc.Next(); e == nil && k != nil; k, v, e = ss.rc.Next() {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context channel interrupted")
		default:
		}

		syncStage := string(k)
		syncProgress, unmarshalError := ss.unmarshal(v)

		if unmarshalError != nil {
			return nil, fmt.Errorf("could not unmarshal sync stage data: %v", unmarshalError)
		}

		syncStageProgress[syncStage] = strconv.FormatUint(syncProgress, syncProgressBase)
	}
	if e != nil {
		return nil, fmt.Errorf("could not process remote cursor line: %v", e)
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
