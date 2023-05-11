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

type SyncStages = map[string]string

const syncStageDb = "chaindata"
const syncStageTable = "SyncStage"
const syncProgressBase = 10

func (uih *UiHandler) findSyncStages(ctx context.Context, w http.ResponseWriter, templ *template.Template, remoteCursor *RemoteCursor) {
	rc, err := remoteCursor.init(syncStageDb, syncStageTable, nil)

	if err != nil {
		fmt.Fprintf(w, "Create remote cursor: %v", err)
		return
	}

	syncStages := make(map[string]string)

	var k, v []byte
	var e error
	for k, v, e = rc.Next(); e == nil && k != nil; k, _, e = rc.Next() {
		select {
		case <-ctx.Done():
			fmt.Fprintf(w, "Interrupted\n")
			return
		default:
		}
		syncStage := string(k)
		syncProgress, unmarshalError := unmarshalData(v)

		if unmarshalError != nil {
			fmt.Printf("Unable to unmarshal sync stage data: %v\n", unmarshalError)
			return
		}

		syncStages[syncStage] = strconv.FormatUint(syncProgress, syncProgressBase)
	}

	if templateErr := templ.ExecuteTemplate(w, "sync_stages.html", syncStages); templateErr != nil {
		fmt.Fprintf(w, "Executing Sync stages template: %v\n", templateErr)
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	if e != nil {
		fmt.Fprintf(w, "Process remote cursor line: %v\n", e)
		return
	}
}

func unmarshalData(data []byte) (uint64, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("value must be at least 8 bytes, got %d", len(data))
	}
	return binary.BigEndian.Uint64(data[:8]), nil
}
