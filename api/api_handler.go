package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/go-chi/chi/v5"
	api_internal "github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
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

func (h *APIHandler) Versions(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	versions, err := client.Version(r.Context())

	if err != nil {
		api_internal.EncodeError(w, r, err)
		return
	}

	jsonData, err := json.Marshal(versions)

	if err != nil {
		fmt.Fprintf(w, "Unable to get version: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) CMDLine(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmdLineArgs, err := client.CMDLineArgs(r.Context())

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	jsonData, err := json.Marshal(cmdLineArgs)

	if err != nil {
		fmt.Fprintf(w, "Unable to get version: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) Flags(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	flags, err := client.Flags(r.Context())

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	jsonData, err := json.Marshal(flags)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) Logs(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logs, err := client.LogFiles(r.Context())

	if err != nil {
		api_internal.EncodeError(w, r, err)
		return
	}

	jsonData, err := json.Marshal(logs)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) Log(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

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

func (h *APIHandler) DBs(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbs, err := client.DBs(r.Context())

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

func (h *APIHandler) Tables(w http.ResponseWriter, r *http.Request) {
	db, tables := path.Split(chi.URLParam(r, "*"))

	if tables != "tables" {
		http.Error(w, "unexpected db path format", http.StatusNotFound)
		return
	}

	client, err := h.findNodeClient(w, r)

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
	client, err := h.findNodeClient(w, r)

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
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.BodiesDownload(r.Context(), w)
}

func (h *APIHandler) HeadersDownload(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.HeadersDownload(r.Context(), w)
}

func (h *APIHandler) SyncStages(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

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

func (h *APIHandler) Peers(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	peers, err := client.FindPeers(r.Context())

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch peers: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(peers)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)

}

func (h *APIHandler) Bootnodes(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bootnodes, err := client.Bootnodes(r.Context())

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch bootnodes: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(bootnodes)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)

}

func (h *APIHandler) ShanphotSync(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	snapSync, err := client.ShanphotSync(r.Context())

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch snapshot sync data: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(snapSync)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) ShanphotFilesList(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	snapSync, err := client.ShanphotFiles(r.Context())

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch snapshot files list: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(snapSync)

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *APIHandler) findNodeClient(w http.ResponseWriter, r *http.Request) (erigon_node.Client, error) {
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
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/versions", r.Versions)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/cmdline", r.CMDLine)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/flags", r.Flags)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/logs", r.Logs)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/logs/{file}", r.Log)

	r.Get("/sessions/{sessionId}/nodes/{nodeId}/dbs", r.DBs)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/dbs/*", r.Tables)

	r.Get("/sessions/{sessionId}/nodes/{nodeId}/reorgs", r.ReOrg)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/bodies/download-summary", r.BodiesDownload)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/headers/download-summary", r.HeadersDownload)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/sync-stages", r.SyncStages)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/peers", r.Peers)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/bootnodes", r.Bootnodes)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/snapshot-sync", r.ShanphotSync)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/snapshot-files-list", r.ShanphotFilesList)

	return r
}

const (
	NodeId    = "nodeId"
	SessionId = "sessionId"
)
