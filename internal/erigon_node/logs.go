package erigon_node

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LogListItem struct {
	Filename    string
	Size        int64
	PrintedSize string
}
type LogList struct {
	Success     bool
	Error       string
	SessionName string
	List        []LogListItem
}
type LogPart struct {
	Success bool
	Error   string
	Lines   []string
}

func MBToGB(b uint64) (float64, int) {
	const unit = 1024
	if b < unit {
		return float64(b), 0
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return float64(b) / float64(div), exp
}

func ByteCount(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	bGb, exp := MBToGB(b)
	return fmt.Sprintf("%.1f%cB", bGb, "KMGTPE"[exp])
}

// Produces (into the writer w) the list of available lots inside the div element, using log_list.html template and LogList object
func (c *NodeClient) ProcessLogList(ctx context.Context, w http.ResponseWriter, sessionName string) error {
	request, err := c.fetch(ctx, "logs_list", nil)

	var list = LogList{SessionName: sessionName}

	if err != nil {
		return fmt.Errorf("Can't get log list", err)
	}

	_, result, err := request.nextResult(ctx)

	var lines []string

	err = json.Unmarshal(result, lines)

	if err != nil {
		return err
	}

	if len(lines) > 0 && strings.HasPrefix(lines[0], SuccessLine) {
		list.Success = true
		for _, line := range lines[1:] {
			if len(line) == 0 {
				// skip empty line (usually at the end)
				continue
			}
			terms := strings.Split(line, " | ")
			if len(terms) != 2 {
				list.Error = fmt.Sprintf("incorrect response line (need to have 2 terms divided by |): %v", line)
				list.Success = false
				break
			}
			size, err := strconv.ParseUint(terms[1], 10, 64)
			if err != nil {
				list.Error = fmt.Sprintf("incorrect size: %v", terms[1])
				list.Success = false
				break
			}
			list.List = append(list.List, LogListItem{Filename: terms[0], Size: int64(size), PrintedSize: ByteCount(size)})
		}
	} else {
		list.Error = fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", lines)
	}

	//if err := template.ExecuteTemplate(w, "log_list.html", list); err != nil {
	//	fmt.Fprintf(w, "Executing log_list template: %v", err)
	//	return
	//}

	return nil
}

func (c *NodeClient) LogHead(ctx context.Context, filename string) (LogPart, error) {
	request, err := c.fetch(ctx, "logs_read", LogParams{url.QueryEscape(filename), 0})

	if err != nil {
		return LogPart{}, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return LogPart{}, err
	}

	return processLogPart(result)
}

func (c *NodeClient) LogTail(ctx context.Context, filename string, offset uint64) (LogPart, error) {
	request, err := c.fetch(ctx, "logs_read", LogParams{url.QueryEscape(filename), offset})

	if err != nil {
		return LogPart{}, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return LogPart{}, err
	}

	return processLogPart(result)
}

// Produces (into writer w) log part (head or tail) inside the div HTML element, using log_read.html template and LogPart object
func processLogPart(result json.RawMessage) (LogPart, error) {
	var part LogPart

	err := json.Unmarshal(result, &part.Lines)

	if err != nil {
		return LogPart{}, err
	}

	return part, nil
}

var logReadFirstLine = regexp.MustCompile("^SUCCESS: ([0-9]+)-([0-9]+)/([0-9]+)$")

// Parses the response from the erigon node, which contains a part of a log file.
// It should start with a line of format: SUCCESS from_offset/to_offset/total_size,
// followed by the actual log chunk
func parseLogPart(offset uint64) (bool, uint64, uint64, []byte, string) {
	/*nodeRequest.Lock.Lock()
	defer nodeRequest.Lock.Unlock()
	if !nodeRequest.Served {
		return false, 0, 0, nil, ""
	}
	clear := nodeRequest.Retries >= 16
	if nodeRequest.Err != "" {
		return clear, 0, 0, nil, nodeRequest.Err
	}
	firstLineEnd := bytes.IndexByte(nodeRequest.Response, '\n')
	if firstLineEnd == -1 {
		return clear, 0, 0, nil, "could not find first line in log part response"
	}
	m := logReadFirstLine.FindSubmatch(nodeRequest.Response[:firstLineEnd])
	if m == nil {
		return clear, 0, 0, nil, fmt.Sprintf("first line needs to have format SUCCESS: from-to/total, was [%sn", nodeRequest.Response[:firstLineEnd])
	}
	from, err := strconv.ParseUint(string(m[1]), 10, 64)
	if err != nil {
		return clear, 0, 0, nil, fmt.Sprintf("parsing from: %v", err)
	}
	if from != offset {
		return clear, 0, 0, nil, fmt.Sprintf("Unexpected from %d, wanted %d", from, offset)
	}
	to, err := strconv.ParseUint(string(m[2]), 10, 64)
	if err != nil {
		return clear, 0, 0, nil, fmt.Sprintf("parsing to: %v", err)
	}
	total, err := strconv.ParseUint(string(m[3]), 10, 64)
	if err != nil {
		return clear, 0, 0, nil, fmt.Sprintf("parsing total: %v", err)
	}
	return true, to, total, nodeRequest.Response[firstLineEnd+1:], ""
	*/

	return true, 0, 0, []byte{}, ""
}

// Implements io.ReaderSeeker to be used as parameter to http.ServeContent
type LogReader struct {
	Filename string // Name of the log files to download
	Client   Client
	Total    uint64 // Size of the log file to be downloaded. Needs to be known before download
	Offset   uint64 // Current Offset set either by the Seek() or Read() functions
	Ctx      context.Context
}

type LogParams struct {
	File   string `json:"file"`
	Offset uint64 `json:"offset"`
}

// Part of the io.Reader interface - emulates reading from the remote logs as if it was from the web server itself
func (lr *LogReader) Read(p []byte) (n int, err error) {
	request, err := lr.Client.fetch(lr.Ctx, "logs_read", LogParams{url.QueryEscape(lr.Filename), lr.Offset})

	if err != nil {
		return 0, err
	}

	var total uint64
	var part []byte

	for {
		select {
		case <-lr.Ctx.Done():
			return 0, fmt.Errorf("interrupted")
		default:
		}

		more, _ /*result*/, err := request.nextResult(lr.Ctx)

		if err != nil {
			return 0, err
		}

		//clear, _, total, part, errStr = parseLogPart(nodeRequest, lr.Offset)

		if !more {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	lr.Total = total
	copied := copy(p, part)
	lr.Offset += uint64(copied)
	if lr.Offset == total {
		return copied, io.EOF
	}

	return copied, nil
}

// Part of the io.Seeker interface. Please note io.SeekEnd - this is used by http.ServeContent to establish content length
func (lr *LogReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		lr.Offset = uint64(offset)
	case io.SeekCurrent:
		lr.Offset = uint64(int64(lr.Offset) + offset)
	case io.SeekEnd:
		if lr.Total > 0 {
			lr.Offset = uint64(int64(lr.Total) + offset)
		} else {
			lr.Offset = 0
		}
	}
	return int64(lr.Offset), nil
}
