package erigon_node

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (c *NodeClient) BodiesDownload(ctx context.Context, w http.ResponseWriter) {
	var tick int64
	sendEvery := time.NewTicker(1000 * time.Millisecond)
	defer sendEvery.Stop()

	for {

		// First, fetch list of DB paths
		request, err := c.fetch(ctx, "block_body_download", url.Values{"sinceTick": []string{strconv.FormatInt(tick, 10)}})

		if err != nil {
			fmt.Fprintf(w, "Fetching list of changes: %s", err)
			return
		}

		select {
		case <-ctx.Done():
			fmt.Fprintf(w, "Interrupted\n")
			return
		default:
		}

		more, _ /*result*/, err := request.nextResult(ctx)

		if err != nil {
			fmt.Fprintf(w, "Fetching list of changes: %s", err)
			return
		}

		/*
			var changesMode bool

			changes := map[uint64]struct{}{}
			switch {
			case len(line) == 0:
				// Skip empty lines
			case strings.HasPrefix(line, "snapshot "):
				tick, err = strconv.ParseInt(line[len("snapshot "):], 10, 64)
				if err != nil {
					fmt.Fprintf(w, "parsing snapshot tick [%s]: %v\n", line, err)
					return
				}
				changesMode = false
				snapshot.Clear(true)
			case strings.HasPrefix(line, "changes "):
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
									sendSnapshot(snapshot, w, template, sendEvery)
									maps.Clear(changes)
								}
							}
						}
		*/ /*
					tick++
				}
				changes[id] = struct{}{}
				if state == 0 {
					snapshot.Delete(SnapshotItem{Id: id})
				} else {
					snapshot.ReplaceOrInsert(SnapshotItem{Id: id, State: byte(state)})
				}
			}

			sendSnapshot(snapshot, w)
			maps.Clear(changes)
		*/

		if !more {
			break
		}

		<-sendEvery.C
	}
}

/*func sendSnapshot(snapshot *btree.BTreeG[SnapshotItem], w http.ResponseWriter) {
	//<- sendEvery.C
	var bd BodyDownload
	first := true
	snapshot.Ascend(func(item SnapshotItem) bool {
		if first {
			first = false
			bd.BlockNum = item.Id
			pre := int(bd.BlockNum)
			bd.Pre10_000_000 = make([]struct{}, pre/10_000_000)
			pre %= 10_000_000
			bd.Pre1_000_000 = make([]struct{}, pre/1_000_000)
			pre %= 1_000_000
			bd.Pre100_000 = make([]struct{}, pre/100_000)
			pre %= 100_000
			bd.Pre10_000 = make([]struct{}, pre/10_000)
			pre %= 10_000
			bd.Pre1_000 = make([]struct{}, pre/1_000)
			pre %= 1_000
			bd.Pre100 = make([]struct{}, pre/100)
			pre %= 100
			bd.Pre10 = make([]struct{}, pre/10)
			pre %= 10
			bd.Pre1 = make([]struct{}, pre)
		}
		bd.States = append(bd.States, item)
		bd.Legends[item.State] = true
		return item.Id < bd.BlockNum+VisLimit // We limit visualisation to VisLimit first blocks
	})
	//if err := templ.ExecuteTemplate(w, "body_download.html", bd); err != nil {
	//	fmt.Fprintf(w, "Executing body_download template: %v", err)
	//	return
	//}
}*/

const VisLimit = 1000
