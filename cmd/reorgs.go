package cmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Demonstration of the working with the Erigon database remotely on the example of getting information
// about past reorganisation of the chain

func advance(key []byte) bool {
	if len(key) == 0 {
		return false
	}
	i := len(key) - 1
	for i >= 0 && key[i] == 0xff {
		i--
	}
	if i < 0 {
		return false
	}
	key[i]++
	return true
}

func (uih *UiHandler) findReorgs(ctx context.Context, r *http.Request, w http.ResponseWriter, sessionName string, requestChannel chan *NodeRequest) {
	start := time.Now()
	// First, fetch list of DB paths
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
	// Go through "Header" table and look for entries with the same block number but different hashes
	var key []byte
	var prevK []byte
	reorgCount := 0
	for {
		success, result = uih.fetch(fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", chaindataPath, "Header", key), requestChannel)
		if !success {
			fmt.Fprintf(w, "Reading Headers table: %s", result)
			return
		}
		lines := strings.Split(result, "\n")
		if len(lines) == 0 || !strings.HasPrefix(lines[0], successLine) {
			fmt.Fprintf(w, "Incorrect response (first line needs to be SUCCESS): %v", lines)
			return
		}
		lineCount := 0
		for _, line := range lines[1:] {
			if len(line) == 0 {
				// Skip empty line (at the end)
				continue
			}
			sepIndex := strings.Index(line, " | ")
			if sepIndex == -1 {
				fmt.Fprintf(w, "Could not find key-value separator | in the result line: %v", line)
				return
			}
			var k []byte
			var e error
			if k, e = hex.DecodeString(line[:sepIndex]); e != nil {
				fmt.Fprintf(w, "Could not parse the key [%s]: %v", line[:sepIndex], e)
				return
			}
			if _, e = hex.DecodeString(line[sepIndex+3:]); e != nil {
				fmt.Fprintf(w, "Could not parse the value [%s]: %v", line[sepIndex+3:], e)
				return
			}
			if len(k) >= 8 && len(prevK) >= 8 && bytes.Equal(k[:8], prevK[:8]) {
				bn := binary.BigEndian.Uint64(k[:8])
				fmt.Fprintf(w, "Reorg at block [%d]\n", bn)
				reorgCount++
			}
			prevK = k
			lineCount++
		}
		if lineCount == 0 {
			// No more records
			break
		}
		if !advance(prevK) {
			break
		}
		key = prevK
	}
	fmt.Fprintf(w, "Reorg count: %d, produced in %s\n", reorgCount, time.Since(start))
}
