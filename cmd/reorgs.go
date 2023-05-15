package cmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

// Demonstration of the working with the Erigon database remotely on the example of getting information
// about past reorganisation of the chain
const headersDb = "chaindata"
const headersTable = "Header"

func (uih *UiHandler) findReorgs(ctx context.Context, w http.ResponseWriter, templ *template.Template, requestChannel chan *NodeRequest) {
	start := time.Now()

	rc := NewRemoteCursor(uih.remoteApi, requestChannel)

	if err := rc.Init(headersDb, headersTable, nil); err != nil {
		fmt.Fprintf(w, "Create remote cursor: %v", err)
		return
	}

	// Go through "Header" table and look for entries with the same block number but different hashes
	var prevK []byte
	reorgCount := 0
	var k []byte
	var e error
	var count int
	for k, _, e = rc.Next(); e == nil && k != nil; k, _, e = rc.Next() {
		select {
		case <-ctx.Done():
			fmt.Fprintf(w, "Interrupted\n")
			return
		default:
		}
		if len(k) >= 8 && len(prevK) >= 8 && bytes.Equal(k[:8], prevK[:8]) {
			bn := binary.BigEndian.Uint64(k[:8])
			if err := templ.ExecuteTemplate(w, "reorg_block.html", bn); err != nil {
				fmt.Fprintf(w, "Executing reorg_block template: %v\n", err)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			reorgCount++
		}
		prevK = k
		count++
		if count%1000 == 0 {
			if err := templ.ExecuteTemplate(w, "reorg_spacer.html", nil); err != nil {
				fmt.Fprintf(w, "Executing reorg_spacer template: %v\n", err)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
	if e != nil {
		fmt.Fprintf(w, "Process remote cursor line: %v\n", e)
		return
	}
	fmt.Fprintf(w, "Reorg count: %d, produced in %s\n", reorgCount, time.Since(start))
}
