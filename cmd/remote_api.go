package cmd

import (
	"fmt"
	"strings"
	"time"
)

type RemoteApiReader interface {
	fetch(url string, requestChannel chan *NodeRequest) (bool, string)
	getResultLines(result string) ([]string, error)
}

type RemoteApi struct{}

func (ra *RemoteApi) fetch(url string, requestChannel chan *NodeRequest) (bool, string) {
	if requestChannel == nil {
		return false, "ERROR: Node is not allocated\n"
	}
	// Request command line arguments
	nodeRequest := &NodeRequest{url: url}
	requestChannel <- nodeRequest
	var sb strings.Builder
	var success bool
	for nodeRequest != nil {
		nodeRequest.lock.Lock()
		clear := nodeRequest.served
		if nodeRequest.served {
			if nodeRequest.err == "" {
				sb.Reset()
				sb.Write(nodeRequest.response)
				success = true
			} else {
				success = false
				fmt.Fprintf(&sb, "ERROR: %s\n", nodeRequest.err)
				if nodeRequest.retries < 16 {
					clear = false
				}
			}
		}
		nodeRequest.lock.Unlock()
		if clear {
			nodeRequest = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return success, sb.String()
}

func (ra *RemoteApi) getResultLines(result string) ([]string, error) {
	lines := strings.Split(result, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], successLine) {
		return nil, fmt.Errorf("incorrect response (first line needs to be SUCCESS): %s", result)
	}

	return lines[1:], nil
}
