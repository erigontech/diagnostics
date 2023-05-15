package handler

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

	"github.com/ledgerwatch/diagnostics/pkg/session"
)

type NodeRequest struct {
	mx       sync.Mutex
	url      string
	served   bool
	response []byte
	err      string
	retries  int
}

const MaxRequestRetries = 16 // How many time to retry a request to the support

type BridgeHandler struct {
	//cancel context.CancelFunc
	uih *UIHandler
}

func NewBridgeHandler(uih *UIHandler) *BridgeHandler {
	return &BridgeHandler{uih}
}

var supportURLRegex = regexp.MustCompile("^/support/([0-9]+)$")

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

	m := supportURLRegex.FindStringSubmatch(r.URL.Path)
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

	nodeSession.Connect(r.RemoteAddr)
	defer nodeSession.Disconnect()

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
	processNodeRequests(*nodeSession, writeBuf, flusher, w, r)
}

func processNodeRequests(n session.Node, writeBuf bytes.Buffer, flusher http.Flusher, w http.ResponseWriter, r *http.Request) {
	for req := range n.Requests {
		req.Mx.Lock()
		url := req.URL
		req.Mx.Unlock()

		log.Printf("Sending request %s\n", url)
		writeBuf.Reset()
		fmt.Fprint(&writeBuf, url)
		if _, err := w.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v\n", err)
			req.Mx.Lock()
			req.Served = true
			req.Resp = nil
			req.Err = fmt.Sprintf("writing metrics request: %v", err)
			req.Retries++
			if req.Retries < 16 {
				select {
				case n.Requests <- req:
				default:
				}
			}
			req.Mx.Unlock()
			return
		}

		flusher.Flush()
		// Read the response
		var sizeBuf [4]byte
		if _, err := io.ReadFull(r.Body, sizeBuf[:]); err != nil {
			log.Printf("Reading size of metrics response: %v\n", err)
			req.Mx.Lock()
			req.Served = true
			req.Resp = nil
			req.Err = fmt.Sprintf("reading size of metrics response: %v", err)
			req.Retries++
			if req.Retries < 16 {
				select {
				case n.Requests <- req:
				default:
				}
			}
			req.Mx.Unlock()
			return
		}

		metricsBuf := make([]byte, binary.BigEndian.Uint32(sizeBuf[:]))
		if _, err := io.ReadFull(r.Body, metricsBuf); err != nil {
			log.Printf("Reading metrics response: %v\n", err)
			req.Mx.Lock()
			req.Served = true
			req.Resp = nil
			req.Err = fmt.Sprintf("reading metrics response: %v", err)
			req.Retries++
			if req.Retries < 16 {
				select {
				case n.Requests <- req:
				default:
				}
			}
			req.Mx.Unlock()
			return
		}

		req.Mx.Lock()
		req.Served = true
		req.Resp = metricsBuf
		req.Err = ""
		req.Mx.Unlock()
	}
}
