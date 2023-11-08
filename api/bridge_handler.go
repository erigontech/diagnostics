package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/ledgerwatch/diagnostics"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
	"github.com/pkg/errors"
)

var _ http.Handler = &APIHandler{}

type BridgeHandler struct {
	chi.Router
	cache sessions.CacheService
}

func (h BridgeHandler) Bridge(w http.ResponseWriter, r *http.Request) {

	//Sends a success Message to the Node client, to receive more information
	flusher, _ := w.(http.Flusher)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	connectionInfo := struct {
		Version  uint64               `json:"version"`
		Sessions []string             `json:"sessions"`
		Nodes    []*sessions.NodeInfo `json:"nodes"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&connectionInfo)

	if err != nil {
		log.Printf("Error reading connection info: %v\n", err)
		internal.EncodeError(w, r, diagnostics.AsBadRequestErr(errors.Errorf("Error reading connection info: %v", err)))
		return
	}

	requestMap := map[string]*erigon_node.NodeRequest{}
	requestMutex := sync.Mutex{}

	for _, node := range connectionInfo.Nodes {
		nodeSession, ok := h.cache.FindNodeSession(node.Id)

		if !ok {
			nodeSession, err = h.cache.CreateNodeSession(node)

			if err != nil {
				log.Printf("Error creating node session: %v\n", err)
				internal.EncodeError(w, r, diagnostics.AsBadRequestErr(fmt.Errorf("error creating node session: %w", err)))
				return

			}
		}

		nodeSession.AttachSessions(connectionInfo.Sessions)

		nodeSession.Connect(r.RemoteAddr)

		go func() {
			defer nodeSession.Disconnect()

			for {
				var request *erigon_node.NodeRequest
				select {
				case request = <-nodeSession.RequestCh:
				case <-ctx.Done():
					return
				}
				rpcRequest := request.Request

				bytes, err := json.Marshal(rpcRequest)

				if err != nil {
					request.Responses <- &erigon_node.Response{
						Last: true,
						Error: &erigon_node.Error{
							Message: fmt.Errorf("failed to marshal request: %w", err).Error(),
						},
					}
					continue
				}

				//fmt.Printf("Sending request %s\n", string(bytes))

				requestMutex.Lock()
				requestMap[rpcRequest.Id] = request
				requestMutex.Unlock()

				if _, err := w.Write(bytes); err != nil {
					requestMutex.Lock()
					delete(requestMap, rpcRequest.Id)
					requestMutex.Unlock()

					fmt.Println(request.Retries, err)
					request.Retries++
					if request.Retries < 15 {
						select {
						case nodeSession.RequestCh <- request:
						default:
						}
					} else {
						request.Responses <- &erigon_node.Response{
							Last: true,
							Error: &erigon_node.Error{
								Message: fmt.Errorf("failed to write metrics request: %w", err).Error(),
							},
						}
					}
					continue
				}

				flusher.Flush()
			}
		}()
	}

	decoder := json.NewDecoder(r.Body)

	for {
		var response erigon_node.Response

		if err = decoder.Decode(&response); err != nil {
			fmt.Printf("can't read response: %v\n", err)
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				return
			default:
			}
			continue
		}

		requestMutex.Lock()
		request, ok := requestMap[response.Id]
		requestMutex.Unlock()

		if !ok {
			continue
		}

		if response.Error != nil {
			response.Last = true
		}

		request.Responses <- &response

		if response.Last {
			requestMutex.Lock()
			delete(requestMap, response.Id)
			requestMutex.Unlock()
		}
	}
}

func NewBridgeHandler(cacheSvc sessions.CacheService) BridgeHandler {
	r := &BridgeHandler{
		Router: chi.NewRouter(),
		cache:  cacheSvc,
	}

	r.Post("/", r.Bridge)

	return *r
}
