package api

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ledgerwatch/diagnostics"
	api_internal "github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
	"github.com/pkg/errors"
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

	fmt.Printf("json data: %s\n", jsonData)
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

	resp := GetVersionResponse{
		NodeVersion: versions.NodeVersion,
		CodeVersion: versions.CodeVersion,
		GitCommit:   versions.GitCommit,
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(w, "Unable to get version: %v", err)
	}

	fmt.Printf("json data: %s\n", jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *UIHandler) CMDLine(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ /*cmdLineArgs :*/, err = client.CMDLineArgs(r.Context())

	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	//if err := h.uiTemplate.ExecuteTemplate(w, "cmd_line.html", cmdLineArgs); err != nil {
	//	fmt.Fprintf(w, "Executing cmd_line template: %v\n", err)
	//	api_internal.EncodeError(w, r, err)
	//}
}

func (h *UIHandler) Flags(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ /*flags*/, err = client.Flags(r.Context())
	if err != nil {
		api_internal.EncodeError(w, r, err)
	}

	//if err := h.uiTemplate.ExecuteTemplate(w, "flags.html", flags); err != nil {
	//	fmt.Fprintf(w, "Executing flags template: %v", err)
	//	internal.EncodeError(w, r, err)
	//}
}

func (h *UIHandler) LogList(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client.ProcessLogList(r.Context(), w, r.FormValue(NodeId))
}

func (h *UIHandler) LogHead(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ /*tail :*/, err = client.LogHead(r.Context(), url.QueryEscape(r.Form.Get("file")))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//return TODO SEMANTIC ERRORING

	//if err := h.uiTemplate.ExecuteTemplate(w, "log_read.html", tail); err != nil {
	//	fmt.Fprintf(w, "Executing log_read template: %v", err)
	//	return
	//}
}

func (h *UIHandler) LogTail(w http.ResponseWriter, r *http.Request) {
	client, err := h.findNodeClient(w, r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	offset, _ := retrieveSizeStrFrom(r)
	//return TODO SEMANTIC ERRORING

	_ /*tail :*/, err = client.LogTail(r.Context(), url.QueryEscape(r.Form.Get("file")), offset)
	//return TODO SEMANTIC ERRORING

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//if err := h.uiTemplate.ExecuteTemplate(w, "log_read.html", tail); err != nil {
	//	fmt.Fprintf(w, "Executing log_read template: %v", err)
	//	return
	//}
}

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

	r.Get("/session/{sessionId}", r.GetSession)

	// Erigon Node data
	r.Get("/session/{sessionId}/node/{nodeId}/versions", r.Versions)
	r.Get("/session/{sessionId}/node/{nodeId}/cmd_line", r.CMDLine)
	r.Get("/session/{sessionId}/node/{nodeId}/flags", r.Flags)

	r.Get("/session/{sessionId}/node/{nodeId}/log_list", r.LogList)
	r.Get("/session/{sessionId}/node/{nodeId}/log_head", r.LogHead)
	r.Get("/session/{sessionId}/node/{nodeId}/log_tail", r.LogTail)
	r.Get("/session/{sessionId}/node/{nodeId}/log_download", r.LogDownload)

	r.Get("/session/{sessionId}/node/{nodeId}/reorgs", r.ReOrg)
	r.Get("/session/{sessionId}/node/{nodeId}/bodies_download", r.BodiesDownload)
	r.Get("/session/{sessionId}/node/{nodeId}/headers_download", r.HeadersDownload)
	r.Get("/session/{sessionId}/node/{nodeId}/sync_stages", r.SyncStages)

	return r
}

const (
	NodeId    = "nodeId"
	SessionId = "sessionId"
)
