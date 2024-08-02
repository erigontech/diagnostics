package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/go-chi/chi/v5"

	api_internal "github.com/erigontech/diagnostics/api/internal"
	"github.com/erigontech/diagnostics/internal/erigon_node"
	"github.com/erigontech/diagnostics/internal/sessions"
)

var _ http.Handler = &APIHandler{}

type GetVersionResponse struct {
	NodeVersion uint64 `json:"node_version"`
	CodeVersion string `json:"code_version"`
	GitCommit   string `json:"git_commit"`
}

type SessionResponse struct {
	IsActive   bool                 `json:"is_active"`
	SessionPin uint64               `json:"session_pin"`
	Nodes      []*sessions.NodeInfo `json:"nodes"`
}

type APIHandler struct {
	chi.Router
	sessions   sessions.CacheService
	erigonNode erigon_node.Client
}

func (h *APIHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, SessionId)

	uiSession, ok := h.sessions.FindUISession(id)

	if !ok {
		var err error
		uiSession, err = h.sessions.CreateUISession(id)

		if err != nil {
			api_internal.EncodeError(w, r, err)
			return
		}
	}

	response := SessionResponse{
		IsActive:   uiSession.IsActive(),
		SessionPin: uiSession.SessionPin,
	}

	for _, node := range uiSession.Nodes {
		response.Nodes = append(response.Nodes, node.NodeInfo)
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(w, "Unable to create session: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) Log(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file := path.Base(r.URL.Path)

	if file == "/" || file == "." {
		http.Error(w, "file is required - specify the name of log file to read", http.StatusBadRequest)
		return
	}

	var offset int64

	offsetStr := r.URL.Query().Get("offset")

	if offsetStr != "" {
		offset, err = strconv.ParseInt(offsetStr, 10, 64)

		if err != nil {
			http.Error(w, fmt.Sprintf("offset %s is not a Uint64 number: %v", offsetStr, err), http.StatusBadRequest)
			return
		}

		if offset < 0 {
			http.Error(w, fmt.Sprintf("offset %d must be non-negative", offset), http.StatusBadRequest)
			return
		}
	}

	var limit int64

	limitStr := r.URL.Query().Get("limit")

	if limitStr != "" {
		limit, err = strconv.ParseInt(limitStr, 10, 64)

		if err != nil {
			http.Error(w, fmt.Sprintf("limit %s is not a Uint64 number: %v", limitStr, err), http.StatusBadRequest)
			return
		}

		if limit < 0 {
			http.Error(w, fmt.Sprintf("limit %d must be non-negative", offset), http.StatusBadRequest)
			return
		}
	}

	download := r.URL.Query().Get("download")

	client.Log(r.Context(), w, file, offset, limit, len(download) > 0)
}

func (h *APIHandler) Tables(w http.ResponseWriter, r *http.Request) {
	db, tables := path.Split(chi.URLParam(r, "*"))

	if tables != "tables" {
		http.Error(w, "unexpected db path format", http.StatusNotFound)
		return
	}

	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbs, err := client.Tables(r.Context(), db[:len(db)-1])

	if err != nil {
		api_internal.EncodeError(w, r, err)
		return
	}

	jsonData, err := json.Marshal(dbs)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) ReOrg(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reorgs, err := client.FindReorgs(r.Context(), w)
	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	jsonData, err := json.Marshal(reorgs)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) BodiesDownload(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.BodiesDownload(r.Context(), w)
}

func (h *APIHandler) HeadersDownload(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.HeadersDownload(r.Context(), w)
}

func (h *APIHandler) SyncStages(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	syncStages, err := client.FindSyncStages(r.Context())

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch sync stage progress: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(syncStages)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) findNodeClient(r *http.Request) (erigon_node.Client, error) {
	sessionId := chi.URLParam(r, SessionId)
	nodeId := chi.URLParam(r, NodeId)

	session, ok := h.sessions.FindNodeSession(nodeId)

	if !ok {
		return nil, fmt.Errorf("unknown nodeId: %s", nodeId)
	}

	for _, sid := range session.UISessions {
		if sid == sessionId {
			return session.Client, nil
		}
	}

	return nil, fmt.Errorf("unknown sessionId: %s", sessionId)
}

func (h *APIHandler) UniversalRequest(w http.ResponseWriter, r *http.Request) {
	apiStr := chi.URLParam(r, "*")

	client, err := h.findNodeClient(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := client.GetResponse(r.Context(), apiStr)

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch snapshot files list: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(response)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func NewAPIHandler(
	sessions sessions.CacheService,
	erigonNode erigon_node.Client,
) *APIHandler {
	r := &APIHandler{
		Router:     chi.NewRouter(),
		sessions:   sessions,
		erigonNode: erigonNode,
	}

	r.Get("/sessions/{sessionId}", r.GetSession)

	// Erigon Node data
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/logs/{file}", r.Log)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/dbs/*", r.Tables)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/reorgs", r.ReOrg)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/bodies/download-summary", r.BodiesDownload)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/headers/download-summary", r.HeadersDownload)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/sync-stages", r.SyncStages)
	r.Get("/v2/sessions/{sessionId}/nodes/{nodeId}/*", r.UniversalRequest)

	return r
}

const (
	NodeId    = "nodeId"
	SessionId = "sessionId"
)
