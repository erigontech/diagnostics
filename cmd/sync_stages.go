package cmd

import (
	"context"
	"encoding/binary"
	"fmt"
	"html/template"
	"net/http"
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

func (uih *UiHandler) findSyncStages(ctx context.Context, w http.ResponseWriter, templ *template.Template, requestChannel chan *NodeRequest) {
	rc := NewRemoteCursor(uih, requestChannel)
	syncStages := &SyncStages{rc: rc}

	syncStageProgress, err := syncStages.fetchSyncStageProgress(ctx)
	if err != nil {
		fmt.Fprintf(w, "Unable to fetch sync stage progress: %v\n", err)
		return
	}

	if templateErr := templ.ExecuteTemplate(w, "sync_stages.html", syncStageProgress); templateErr != nil {
		fmt.Fprintf(w, "Executing Sync stages template: %v\n", templateErr)
		return
	}
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
		syncProgress, unmarshalError := ss.unmarshalData(v)

		if unmarshalError != nil {
			return nil, fmt.Errorf("unable to unmarshal sync stage data: %v", unmarshalError)
		}

		syncStageProgress[syncStage] = strconv.FormatUint(syncProgress, syncProgressBase)
	}
	if e != nil {
		return nil, fmt.Errorf("unable to process remote cursor line: %v", e)
	}

	return syncStageProgress, nil
}

func (ss *SyncStages) unmarshalData(data []byte) (uint64, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("value must be at least 8 bytes, got %d", len(data))
	}
	return binary.BigEndian.Uint64(data[:8]), nil
}
