package cmd

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"mime"
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

func processLogList(w http.ResponseWriter, templ *template.Template, success bool, sessionName string, result string) {
	var list = LogList{SessionName: sessionName}
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], successLine) {
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
	if err := templ.ExecuteTemplate(w, "log_list.html", list); err != nil {
		fmt.Fprintf(w, "Executing log_list template: %v", err)
		return
	}
}

func processLogPart(w http.ResponseWriter, templ *template.Template, success bool, sessionName string, result string) {
	var part LogPart
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], successLine) {
			part.Lines = lines[1:]
		} else {
			part.Lines = lines
		}
		part.Success = true
	} else {
		part.Success = false
		part.Error = result
	}
	if err := templ.ExecuteTemplate(w, "log_read.html", part); err != nil {
		fmt.Fprintf(w, "Executing log_read template: %v", err)
		return
	}
}

var logReadFirstLine = regexp.MustCompile("^SUCCESS: ([0-9]+)-([0-9]+)/([0-9]+)$")

func parseLogPart(nodeRequest *NodeRequest, offset uint64) (bool, uint64, uint64, []byte, string) {
	nodeRequest.lock.Lock()
	defer nodeRequest.lock.Unlock()
	if !nodeRequest.served {
		return false, 0, 0, nil, ""
	}
	clear := nodeRequest.retries >= 16
	if nodeRequest.err != "" {
		return clear, 0, 0, nil, nodeRequest.err
	}
	firstLineEnd := bytes.IndexByte(nodeRequest.response, '\n')
	if firstLineEnd == -1 {
		return clear, 0, 0, nil, fmt.Sprintf("could not find first line in log part response")
	}
	m := logReadFirstLine.FindSubmatch(nodeRequest.response[:firstLineEnd])
	if m == nil {
		return clear, 0, 0, nil, fmt.Sprintf("first line needs to have format SUCCESS: from-to/total, was [%sn", nodeRequest.response[:firstLineEnd])
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
	return true, to, total, nodeRequest.response[firstLineEnd+1:], ""
}

// Implements io.ReaderSeeker to be used as parameter to http.ServeContent
type LogReader struct {
	filename       string // Name of the log files to download
	requestChannel chan *NodeRequest
	total          uint64 // Size of the log file to be downloaded. Needs to be known before download
	offset         uint64 // Current offset set either by the Seek() or Read() functions
	ctx            context.Context
}

func (lr *LogReader) Read(p []byte) (n int, err error) {
	nodeRequest := &NodeRequest{url: fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(lr.filename), lr.offset)}
	lr.requestChannel <- nodeRequest
	var total uint64
	var clear bool
	var part []byte
	var errStr string
	for nodeRequest != nil {
		select {
		case <-lr.ctx.Done():
			return 0, fmt.Errorf("interrupted")
		default:
		}
		clear, _, total, part, errStr = parseLogPart(nodeRequest, lr.offset)
		if clear {
			nodeRequest = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if errStr != "" {
		return 0, fmt.Errorf(errStr)
	}
	lr.total = total
	copied := copy(p, part)
	lr.offset += uint64(copied)
	if lr.offset == total {
		return copied, io.EOF
	}
	return copied, nil
}

func (lr *LogReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		lr.offset = uint64(offset)
	case io.SeekCurrent:
		lr.offset = uint64(int64(lr.offset) + offset)
	case io.SeekEnd:
		if lr.total > 0 {
			lr.offset = uint64(int64(lr.total) + offset)
		} else {
			lr.offset = 0
		}
	}
	return int64(lr.offset), nil
}

func transmitLogFile(ctx context.Context, r *http.Request, w http.ResponseWriter, sessionName string, filename string, size uint64, requestChannel chan *NodeRequest) {
	if requestChannel == nil {
		fmt.Fprintf(w, "ERROR: Node is not allocated\n")
		return
	}
	cd := mime.FormatMediaType("attachment", map[string]string{"filename": sessionName + "_" + filename})
	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Type", "application/octet-stream")
	logReader := &LogReader{filename: filename, requestChannel: requestChannel, offset: 0, total: size, ctx: ctx}
	http.ServeContent(w, r, filename, time.Now(), logReader)
}
