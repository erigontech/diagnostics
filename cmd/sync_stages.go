package cmd

import (
	"context"
	"encoding/binary"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

// Demonstration of the working with the Erigon database remotely on the example of getting information
// about the progress of sync stages

type SyncStages = map[string]string

func (uih *UiHandler) getSyncStages(ctx context.Context, w http.ResponseWriter, templ *template.Template, requestChannel chan *NodeRequest) {
	success, result := uih.fetch("/db/list\n", requestChannel)
	if !success {
		fmt.Fprintf(w, "Fetching list of db paths: %s", result)
		return
	}
	lines := strings.Split(result, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], successLine) {
		fmt.Fprintf(w, "Incorrect response (first line needs to be SUCCESS): %v", lines)
		return
	}
	var chaindataPath string
	for _, line := range lines[1:] {
		if strings.HasSuffix(line, "/chaindata") {
			chaindataPath = line
		}
	}
	if chaindataPath == "" {
		fmt.Fprintf(w, "DB path chaindata not found: %v", lines)
		return
	}

	rc, err := NewRemoteCursor(chaindataPath, "SyncStage", requestChannel, nil)
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
		syncProgress, err := unmarshalData(v)
		if err != nil {
			fmt.Printf("Unable to unmarshal sync stage data: %v\n", err)

			return
		}

		syncStages[syncStage] = strconv.FormatUint(syncProgress, 10)
	}

	if err := templ.ExecuteTemplate(w, "sync_stages.html", syncStages); err != nil {
		fmt.Fprintf(w, "Executing Sync stages template update: %v\n", err)
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
