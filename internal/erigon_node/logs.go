package erigon_node

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledgerwatch/diagnostics/internal"
)

type LogListItem struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	PrintedSize string `json:"printed_size"`
}
type LogList struct {
	Success     bool          `json:"success"`
	Error       string        `json:"error"`
	SessionName string        `json:"session_name"`
	List        []LogListItem `json:"list"`
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
func (c *NodeClient) ProcessLogList(w http.ResponseWriter, template *template.Template, sessionName string, requestChannel chan *internal.NodeRequest) {
	success, result := c.fetch("/logs/list\n", requestChannel)
	var list = LogList{SessionName: sessionName}
	if success {
		lines := strings.Split(result, "\n")
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
	} else {
		list.Error = result
	}
	if err := template.ExecuteTemplate(w, "log_list.html", list); err != nil {
		fmt.Fprintf(w, "Executing log_list template: %v", err)
		return
	}
}

func (c *NodeClient) LogHead(filename string, requestChannel chan *internal.NodeRequest) LogPart {
	success, result := c.fetch(fmt.Sprintf("/logs/read?file=%s&offset=0\n", filename), requestChannel)
	return processLogPart(success, result)
}

func (c *NodeClient) LogTail(filename string, offset uint64, requestChannel chan *internal.NodeRequest) LogPart {
	success, result := c.fetch(fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(filename), offset), requestChannel)
	return processLogPart(success, result)
}

// Produces (into writer w) log part (head or tail) inside the div HTML element, using log_read.html template and LogPart object
func processLogPart(success bool, result string) LogPart {
	var part LogPart
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], SuccessLine) {
			part.Lines = lines[1:]
		} else {
			part.Lines = lines
		}
		part.Success = true
	} else {
		part.Success = false
		part.Error = result
	}

	return part
}

var logReadFirstLine = regexp.MustCompile("^SUCCESS: ([0-9]+)-([0-9]+)/([0-9]+)$")

// Parses the response from the erigon node, which contains a part of a log file.
// It should start with a line of format: SUCCESS from_offset/to_offset/total_size,
// followed by the actual log chunk
func parseLogPart(nodeRequest *internal.NodeRequest, offset uint64) (bool, uint64, uint64, []byte, string) {
	nodeRequest.Lock.Lock()
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
}

// Implements io.ReaderSeeker to be used as parameter to http.ServeContent
type LogReader struct {
	Filename       string // Name of the log files to download
	RequestChannel chan *internal.NodeRequest
	Total          uint64 // Size of the log file to be downloaded. Needs to be known before download
	Offset         uint64 // Current Offset set either by the Seek() or Read() functions
	Ctx            context.Context
}

// Part of the io.Reader interface - emulates reading from the remote logs as if it was from the web server itself
func (lr *LogReader) Read(p []byte) (n int, err error) {
	nodeRequest := &internal.NodeRequest{Url: fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(lr.Filename), lr.Offset)}
	lr.RequestChannel <- nodeRequest
	var total uint64
	var clear bool
	var part []byte
	var errStr string
	for nodeRequest != nil {
		select {
		case <-lr.Ctx.Done():
			return 0, fmt.Errorf("interrupted")
		default:
		}
		clear, _, total, part, errStr = parseLogPart(nodeRequest, lr.Offset)
		if clear {
			nodeRequest = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if errStr != "" {
		return 0, fmt.Errorf(errStr)
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
