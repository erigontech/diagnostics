package erigon_node

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"
)

// Demonstration of the working with the Erigon database remotely on the example of getting information
// about past reorganisation of the chain
const (
	headersDb    = "chaindata"
	headersTable = "Header"
	maxCount     = 1000
)

type Reorg struct {
	TotalScanned int      `json:"TotalScanned"`
	WrongBlocks  []uint64 `json:"WrongBlocks"`
	TimeTook     string   `json:"TimeTook"`
}

// FindReorgs - Go through "Header" table and look for entries with the same block number but different hashes
func (c *NodeClient) FindReorgs(ctx context.Context, writer http.ResponseWriter) (Reorg, error) {
	start := time.Now()
	var err error

	rc := NewRemoteCursor(c)
	if err = rc.Init(ctx, headersDb, headersTable, nil); err != nil {
		fmt.Fprintf(writer, "Create remote cursor: %v", err)
		return Reorg{}, err
	}

	total, wrongBlocks, errors := c.findReorgsInternally(ctx, rc)
	for _, err := range errors {
		if err != nil {
			fmt.Fprintf(writer, "%v\n", err)
		}
	}

	return Reorg{
		TotalScanned: len(total),
		WrongBlocks:  wrongBlocks,
		TimeTook:     time.Since(start).String(),
	}, nil
}

/*func (c *NodeClient) executeFlush(writer io.Writer,
	template *template.Template,
	name string, data any) error {
	if err := template.ExecuteTemplate(writer, name, data); err != nil {
		return err
	}
	if f, ok := writer.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}*/

// findReorgsInternally - searching for reorgs,
// return back total blocks set and wrong blocks
// if there are errors in the middle of processing will return back
// slice of errors
func (c *NodeClient) findReorgsInternally(ctx context.Context, rc *RemoteCursor) (map[uint64][]byte, []uint64, []error) {
	var errors []error
	set := make(map[uint64][]byte)
	var wrongBlocks []uint64

	var k []byte

	var iterator int
	var err error
	for k, _, err = rc.Next(ctx); err == nil && k != nil; k, _, err = rc.Next(ctx) {
		select {
		case <-ctx.Done():
			return nil, nil, []error{fmt.Errorf("Interrupted")}
		default:
		}

		if len(k) == 0 {
			continue
		}

		bn := binary.BigEndian.Uint64(k[:8])
		_, found := set[bn]
		if found {
			//if template != nil {
			//	if err := c.executeFlush(nil, template, "reorg_block.html", bn); err != nil {
			//		errors = append(errors, fmt.Errorf("Executing reorg_block template: %v\n", err))
			//	}
			//}
			wrongBlocks = append(wrongBlocks, bn)
		}
		set[bn] = k

		iterator++
		if iterator%maxCount == 0 {
			fmt.Print(".")
			//if template != nil {
			//	if err := c.executeFlush(nil, template, "reorg_block.html", bn); err != nil {
			//		errors = append(errors, fmt.Errorf("Executing reorg_spacer template: %v\n", err))
			//	}
			//}
		}
	}
	if err != nil {
		errors = append(errors, err)
	}

	return set, wrongBlocks, errors
}
