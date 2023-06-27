package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/ledgerwatch/diagnostics"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
)

var _ http.Handler = &UIHandler{}

type BridgeHandler struct {
	chi.Router
	cache sessions.CacheService
}

func (h BridgeHandler) Bridge(w http.ResponseWriter, r *http.Request) {
	pin, err := retrievePinFromURL(r)
	if err != nil {
		http.Error(w, "Error parsing session PIN", http.StatusBadRequest)
	}

	nodeSession, ok := h.cache.FindNodeSession(pin)
	if !ok {
		log.Printf("Session with specified PIN %d not found\n", pin)
		internal.EncodeError(w, r, diagnostics.BadRequest(errors.Errorf("Session with specified PIN %d not found", pin)))
		return
	}

	//Sends a success Message to the Node client, to receive more information
	flusher, _ := w.(http.Flusher)
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
		log.Printf("Error reading version bytes: %v\n", err)
		internal.EncodeError(w, r, diagnostics.BadRequest(errors.Errorf("Error reading version bytes: %v", err)))
		return
	}

	nodeSession.SupportVersion = binary.BigEndian.Uint64(versionBytes[:])

	// Read data from the request body
	var writeBuf bytes.Buffer
	for request := range nodeSession.RequestCh {
		request.Lock.Lock()
		url := request.Url
		request.Lock.Unlock()
		log.Printf("Sending request %s\n", url)
		writeBuf.Reset()
		fmt.Fprint(&writeBuf, url)
		if _, err := w.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v\n", err)
			request.Lock.Lock()
			request.Served = true
			request.Response = nil
			request.Err = fmt.Sprintf("writing metrics request: %v", err)
			request.Retries++
			if request.Retries < 15 {
				select {
				case nodeSession.RequestCh <- request:
				default:
				}
			}
			request.Lock.Unlock()
			return
		}
		flusher.Flush()
		// Read the response
		var sizeBuf [4]byte
		if _, err := io.ReadFull(r.Body, sizeBuf[:]); err != nil {
			log.Printf("Reading size of metrics response: %v\n", err)
			request.Lock.Lock()
			request.Served = true
			request.Response = nil
			request.Err = fmt.Sprintf("reading size of metrics response: %v", err)
			request.Retries++
			if request.Retries < 15 {
				select {
				case nodeSession.RequestCh <- request:
				default:
				}
			}
			request.Lock.Unlock()
			return
		}
		metricsBuf := make([]byte, binary.BigEndian.Uint32(sizeBuf[:]))
		if _, err := io.ReadFull(r.Body, metricsBuf); err != nil {
			log.Printf("Reading metrics response: %v\n", err)
			request.Lock.Lock()
			request.Served = true
			request.Response = nil
			request.Err = fmt.Sprintf("reading metrics response: %v", err)
			request.Retries++
			if request.Retries < 15 {
				select {
				case nodeSession.RequestCh <- request:
				default:
				}
			}
			request.Lock.Unlock()
			return
		}
		request.Lock.Lock()
		request.Served = true
		request.Response = metricsBuf
		request.Err = ""
		request.Lock.Unlock()
	}

}

func NewBridgeHandler(cacheSvc sessions.CacheService) BridgeHandler {
	r := &BridgeHandler{
		Router: chi.NewRouter(),
		cache:  cacheSvc,
	}

	r.Post("/{pin}", r.Bridge)

	return *r
}
