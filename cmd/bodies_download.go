package cmd

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/btree"
	"golang.org/x/exp/maps"
)

type SnapshotItem struct {
	id    uint64
	state byte
}

func (uih *UiHandler) bodiesDownload(ctx context.Context, w http.ResponseWriter, templ *template.Template, requestChannel chan *NodeRequest) {
	snapshot := btree.NewG[SnapshotItem](16, func(a, b SnapshotItem) bool {
		return a.id < b.id
	})
	var tick int
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(w, "Interrupted\n")
			return
		default:
		}
		// First, fetch list of DB paths
		success, result := uih.fetch(fmt.Sprintf("/block_body_download?sincetick=%d\n", tick), requestChannel)
		if !success {
			fmt.Fprintf(w, "Fetching list of changes: %s", result)
			return
		}
		lines := strings.Split(result, "\n")
		if len(lines) == 0 || !strings.HasPrefix(lines[0], successLine) {
			fmt.Fprintf(w, "incorrect response (first line needs to be SUCCESS)\n")
			return
		}
		lines = lines[1:]
		var changesMode bool
		var tick int64
		var err error
		changes := map[uint64]struct{}{}
		for len(lines) > 0 {
			line := lines[0]
			switch {
			case len(line) == 0:
				// Skip empty lines
			case strings.HasPrefix(line, "snapshot "):
				tick, err = strconv.ParseInt(line[len("snapshot: "):], 10, 64)
				if err != nil {
					fmt.Fprintf(w, "parsing snapshot tick [%s]: %v\n", line, err)
					return
				}
				changesMode = false
				snapshot.Clear(true)
			case strings.HasPrefix(line, "changes "):
				tick, err = strconv.ParseInt(line[len("changes: "):], 10, 64)
				if err != nil {
					fmt.Fprintf(w, "parsing changes tick [%s]: %v\n", line, err)
					return
				}
				changesMode = true
			default:
				splits := strings.Split(line, ",")
				if len(splits) != 2 {
					fmt.Fprintf(w, "snapshot or change line must have 2 comma-separated items [%s]\n", line)
					return
				}
				id, err := strconv.ParseUint(splits[0], 10, 64)
				if err != nil {
					fmt.Fprintf(w, "parsing id [%s]: %v\n", splits[0], err)
					return
				}
				state, err := strconv.ParseUint(splits[1], 10, 8)
				if err != nil {
					fmt.Fprintf(w, "parsing state [%s]: %v\n", splits[1], err)
					return
				}
				if changesMode {
					if _, ok := changes[id]; ok {
						sendSnapshot(snapshot, w, templ)
						maps.Clear(changes)
					}
					tick++
				}
				changes[id] = struct{}{}
				if state == 0 {
					snapshot.Delete(SnapshotItem{id: id})
				} else {
					snapshot.ReplaceOrInsert(SnapshotItem{id: id, state: byte(state)})
				}
			}
		}
	}
}

func sendSnapshot(snapshot *btree.BTreeG[SnapshotItem], w http.ResponseWriter, templ *template.Template) {

}
