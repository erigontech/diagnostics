package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
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

const (
	wsReadBuffer       = 1024
	wsWriteBuffer      = 1024
	wsPingInterval     = 60 * time.Second
	wsPingWriteTimeout = 5 * time.Second
	wsMessageSizeLimit = 32 * 1024 * 1024
)

var wsBufferPool = new(sync.Pool)

func (h BridgeHandler) Bridge(w http.ResponseWriter, r *http.Request) {

	//Sends a success Message to the Node client, to receive more information
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()

	upgrader := websocket.Upgrader{
		EnableCompression: true,
		ReadBufferSize:    wsReadBuffer,
		WriteBufferSize:   wsWriteBuffer,
		WriteBufferPool:   wsBufferPool,
	}

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		internal.EncodeError(w, r, diagnostics.AsBadRequestErr(errors.Errorf("Error upgrading websocket: %v", err)))
		return
	}

	connectionInfo := struct {
		Version  uint64               `json:"version"`
		Sessions []string             `json:"sessions"`
		Nodes    []*sessions.NodeInfo `json:"nodes"`
	}{}

	_, message, err := conn.ReadMessage()

	if err != nil {
		internal.EncodeError(w, r, diagnostics.AsBadRequestErr(errors.Errorf("Error reading connection info: %v", err)))
		return
	}

	err = json.Unmarshal(message, &connectionInfo)

	if err != nil {
		log.Printf("Error reading connection info: %v\n", err)
		internal.EncodeError(w, r, diagnostics.AsBadRequestErr(errors.Errorf("Error unmarshaling connection info: %v", err)))
		return
	}

	requestMap := map[string]*erigon_node.NodeRequest{}
	requestMutex := sync.Mutex{}

	wg := &sync.WaitGroup{}
	defer wg.Wait()
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

		wg.Add(1)
		go func() {
			defer wg.Done()
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

				if err := conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
					requestMutex.Lock()
					delete(requestMap, rpcRequest.Id)
					requestMutex.Unlock()
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
			}
		}()
	}

	for {
		var response erigon_node.Response

		_, message, err := conn.ReadMessage()

		if err != nil {
			fmt.Printf("can't read response: %v\n", err)
			continue
		}

		if err = json.Unmarshal(message, &response); err != nil {
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

	r.Get("/", r.Bridge)

	return *r
}
