package cmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

type NodeRequest struct {
	lock     sync.Mutex
	url      string
	served   bool
	response []byte
	err      string
	retries  int
}

const MaxRequestRetries = 16 // How many time to retry a request to the support

type BridgeHandler struct {
	//cancel context.CancelFunc
	uih *UiHandler
}

var supportUrlRegex = regexp.MustCompile("^/support/([0-9]+)$")

var ErrHTTP2NotSupported = "HTTP2 not supported"

func (bh *BridgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !r.ProtoAtLeast(2, 0) {
		http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
		return
	}
	m := supportUrlRegex.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	pin, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil {
		http.Error(w, "Error parsing session PIN", http.StatusBadRequest)
		log.Printf("Error parsing session pin %s: %v\n", m[1], err)
		return
	}
	nodeSession, ok := bh.uih.findNodeSession(pin)
	if !ok {
		http.Error(w, fmt.Sprintf("Session with specified PIN %d not found", pin), http.StatusBadRequest)
		log.Printf("Session with specified PIN %d not found\n", pin)
		return
	}
	fmt.Fprintf(w, "SUCCESS\n")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()

	nodeSession.connect(r.RemoteAddr)
	defer nodeSession.disconnect()

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var versionBytes [8]byte
	if _, err := io.ReadFull(r.Body, versionBytes[:]); err != nil {
		http.Error(w, fmt.Sprintf("Error reading version bytes: %v", err), http.StatusBadRequest)
		log.Printf("Error reading version bytes: %v\n", err)
		return
	}
	nodeSession.SupportVersion = binary.BigEndian.Uint64(versionBytes[:])

	var writeBuf bytes.Buffer
	for request := range nodeSession.requestCh {
		request.lock.Lock()
		url := request.url
		request.lock.Unlock()
		log.Printf("Sending request %s\n", url)
		writeBuf.Reset()
		fmt.Fprint(&writeBuf, url)
		if _, err := w.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v\n", err)
			request.lock.Lock()
			request.served = true
			request.response = nil
			request.err = fmt.Sprintf("writing metrics request: %v", err)
			request.retries++
			if request.retries < MaxRequestRetries {
				select {
				case nodeSession.requestCh <- request:
				default:
				}
			}
			request.lock.Unlock()
			return
		}
		flusher.Flush()
		// Read the response
		var sizeBuf [4]byte
		if _, err := io.ReadFull(r.Body, sizeBuf[:]); err != nil {
			log.Printf("Reading size of metrics response: %v\n", err)
			request.lock.Lock()
			request.served = true
			request.response = nil
			request.err = fmt.Sprintf("reading size of metrics response: %v", err)
			request.retries++
			if request.retries < MaxRequestRetries {
				select {
				case nodeSession.requestCh <- request:
				default:
				}
			}
			request.lock.Unlock()
			return
		}
		metricsBuf := make([]byte, binary.BigEndian.Uint32(sizeBuf[:]))
		if _, err := io.ReadFull(r.Body, metricsBuf); err != nil {
			log.Printf("Reading metrics response: %v\n", err)
			request.lock.Lock()
			request.served = true
			request.response = nil
			request.err = fmt.Sprintf("reading metrics response: %v", err)
			request.retries++
			if request.retries < MaxRequestRetries {
				select {
				case nodeSession.requestCh <- request:
				default:
				}
			}
			request.lock.Unlock()
			return
		}
		request.lock.Lock()
		request.served = true
		request.response = metricsBuf
		request.err = ""
		request.lock.Unlock()
	}
}
