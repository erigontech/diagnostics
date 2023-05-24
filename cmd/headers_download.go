package cmd

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/btree"
	"golang.org/x/exp/maps"
)

type HeaderDownload struct {
	Legends       [9]bool
	HeaderNum     uint64
	Pre1          []struct{}
	Pre10         []struct{}
	Pre100        []struct{}
	Pre1_000      []struct{}
	Pre10_000     []struct{}
	Pre100_000    []struct{}
	Pre1_000_000  []struct{}
	Pre10_000_000 []struct{}
	States        []SnapshotItem
}

func (uih *UiHandler) headersDownload(ctx context.Context, w http.ResponseWriter, templ *template.Template, requestChannel chan *NodeRequest) {

	snapshot := btree.NewG(16, func(a, b SnapshotItem) bool {
		return a.Id < b.Id
	})
	var tick int64
	sendEvery := time.NewTicker(1000 * time.Millisecond)
	defer sendEvery.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(w, "Interrupted\n")
			return
		default:
		}
		// fmt.Println("Header Download received request")
		// First, fetch list of DB paths
		success, result := uih.remoteApi.fetch(fmt.Sprintf("/headers_download?sincetick=%d\n", tick), requestChannel)
		if !success {
			fmt.Println("Fetching list of changes: ", result)
			fmt.Fprintf(w, "Fetching list of changes: %s", result)
			return
		}

		lines, resultExtractErr := uih.remoteApi.getResultLines(result)
		if resultExtractErr != nil {
			fmt.Println("incorrect response: ", resultExtractErr)
			fmt.Fprintf(w, "incorrect response: %v\n", resultExtractErr)
			return
		}
		// fmt.Println("Result: ", lines)

		var changesMode bool
		var err error
		changes := map[uint64]struct{}{}
		for _, line := range lines {
			switch {
			case len(line) == 0:
				// Skip empty lines
			case strings.HasPrefix(line, "snapshot "):
				fmt.Println("[Header Download] snapshot line found: ", line)
				tick, err = strconv.ParseInt(line[len("snapshot "):], 10, 64)
				if err != nil {
					fmt.Fprintf(w, "parsing snapshot tick [%s]: %v\n", line, err)
					return
				}
				changesMode = false
				snapshot.Clear(true)
			case strings.HasPrefix(line, "changes "):
				fmt.Println("[Header Download] changes line found: ", line)
				tick, err = strconv.ParseInt(line[len("changes "):], 10, 64)
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
					/*
						if _, ok := changes[id]; ok {
							if firstItem, firstOk := snapshot.Min(); firstOk {
								if id < firstItem.Id + VisLimit {
									sendSnapshot(snapshot, w, templ, sendEvery)
									maps.Clear(changes)
								}
							}
						}
					*/
					tick++
				}
				changes[id] = struct{}{}
				if state == 0 {
					snapshot.Delete(SnapshotItem{Id: id})
				} else {
					snapshot.ReplaceOrInsert(SnapshotItem{Id: id, State: byte(state)})
				}
			}
		}
		sendHeadersSnapshot(snapshot, w, templ, sendEvery)
		maps.Clear(changes)
		<-sendEvery.C
	}
}

func sendHeadersSnapshot(snapshot *btree.BTreeG[SnapshotItem], w http.ResponseWriter, templ *template.Template, sendEvery *time.Ticker) {
	//<- sendEvery.C
	var hd HeaderDownload
	first := true
	snapshot.Ascend(func(item SnapshotItem) bool {
		if first {
			first = false
			hd.HeaderNum = item.Id
			pre := int(hd.HeaderNum)
			hd.Pre10_000_000 = make([]struct{}, pre/10_000_000)
			pre %= 10_000_000
			hd.Pre1_000_000 = make([]struct{}, pre/1_000_000)
			pre %= 1_000_000
			hd.Pre100_000 = make([]struct{}, pre/100_000)
			pre %= 100_000
			hd.Pre10_000 = make([]struct{}, pre/10_000)
			pre %= 10_000
			hd.Pre1_000 = make([]struct{}, pre/1_000)
			pre %= 1_000
			hd.Pre100 = make([]struct{}, pre/100)
			pre %= 100
			hd.Pre10 = make([]struct{}, pre/10)
			pre %= 10
			hd.Pre1 = make([]struct{}, pre)
		}
		hd.States = append(hd.States, item)
		hd.Legends[item.State] = true
		return item.Id < hd.HeaderNum+VisLimit // We limit visualisation to VisLimit first blocks
	})
	if err := templ.ExecuteTemplate(w, "headers_download.html", hd); err != nil {
		fmt.Fprintf(w, "Executing headers_download template: %v", err)
		return
	}
}
