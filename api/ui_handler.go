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

var _ http.Handler = &UIHandler{}

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

type UIHandler struct {
	chi.Router
	sessions   sessions.CacheService
	erigonNode erigon_node.Client
}

func (h *UIHandler) GetSession(w http.ResponseWriter, r *http.Request) {
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

func (h *UIHandler) Versions(w http.ResponseWriter, r *http.Request) {
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

func (h *UIHandler) CMDLine(w http.ResponseWriter, r *http.Request) {
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

func (h *UIHandler) Flags(w http.ResponseWriter, r *http.Request) {
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

func (h *UIHandler) Logs(w http.ResponseWriter, r *http.Request) {
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

func (h *UIHandler) Log(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file := path.Base(r.URL.Path)

	if file == "/" || file == "." {
		http.Error(w, fmt.Sprintf("file is required - specify the name of log file to read"), http.StatusBadRequest)
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

	var size int64

	sizeStr := r.URL.Query().Get("size")

	if sizeStr != "" {
		size, err = strconv.ParseInt(sizeStr, 10, 64)

		if err != nil {
			http.Error(w, fmt.Sprintf("size %s is not a Uint64 number: %v", sizeStr, err), http.StatusBadRequest)
			return
		}

		if size < 0 {
			http.Error(w, fmt.Sprintf("size %d must be non-negative", offset), http.StatusBadRequest)
			return
		}
	}

	download := r.URL.Query().Get("download")

	client.Log(r.Context(), w, file, offset, size, len(download) > 0)
}

/*
func (h *UIHandler) LogDownload(w http.ResponseWriter, r *http.Request) {
	// Handles the use case when operator clicks on the link with the log file name, and this initiates the download of this file
	// to the operator's computer (via browser). See LogReader above which is used in http.ServeContent
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	size, err := retrieveSizeStrFrom(r)
	if err != nil {
		api_internal.EncodeError(w, r, err)
	}
	filename := r.Form.Get("file")
	if client == nil {
		api_internal.EncodeError(w, r, diagnostics.AsBadRequestErr(errors.Errorf("ERROR: Node is not allocated\n")))
	}

	cd := mime.FormatMediaType("attachment", map[string]string{"filename": SessionId + "_" + filename})

	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Type", "application/octet-stream")

	logReader := &erigon_node.LogReader{Filename: filename, Client: client, Offset: 0, Total: size, Ctx: r.Context()}
	http.ServeContent(w, r, filename, time.Now(), logReader)
}
*/

func (h *UIHandler) ReOrg(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.FindReorgs(r.Context(), w)
}

func (h *UIHandler) BodiesDownload(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.BodiesDownload(r.Context(), w)
}

func (h *UIHandler) HeadersDownload(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.HeadersDownload(r.Context(), w)
}

func (h *UIHandler) SyncStages(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.FindSyncStages(r.Context(), w)
}

func (h *UIHandler) findNodeClient(w http.ResponseWriter, r *http.Request) (erigon_node.Client, error) {
	sessionId := chi.URLParam(r, SessionId)
	nodeId := chi.URLParam(r, NodeId)

	session, ok := h.sessions.FindNodeSession(nodeId)

	if !ok {
		return nil, fmt.Errorf("Unknown nodeId: %s", nodeId)
	}

	for _, sid := range session.UISessions {
		if sid == sessionId {
			return session.Client, nil
		}
	}

	return nil, fmt.Errorf("Unknown sessionId: %s", sessionId)
}

func NewUIHandler(
	sessions sessions.CacheService,
	erigonNode erigon_node.Client,
) *UIHandler {
	r := &UIHandler{
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

	r.Get("/sessions/{sessionId}/nodes/{nodeId}/reorgs", r.ReOrg)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/bodies/download-stats", r.BodiesDownload)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/headers/download-stats", r.HeadersDownload)
	r.Get("/sessions/{sessionId}/nodes/{nodeId}/sync-stages", r.SyncStages)

	return r
}

const (
	NodeId    = "nodeId"
	SessionId = "sessionId"
)
