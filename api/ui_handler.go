package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ledgerwatch/diagnostics"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
	"github.com/pkg/errors"
)

var _ http.Handler = &UIHandler{}

type UIHandler struct {
	chi.Router
	uiSessions sessions.UIService
	uiTemplate *template.Template
	erigonNode erigon_node.Client
}

func (h *UIHandler) UI(w http.ResponseWriter, r *http.Request) {
	if err := h.uiTemplate.ExecuteTemplate(w, "session.html", h.uiSessions); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIHandler) CreateSession(w http.ResponseWriter, r *http.Request) {

	value := r.FormValue(sessionName)

	err := h.uiSessions.Add(value)
	if err != nil {
		fmt.Fprintf(w, "Unable to create session: %v", err)
		internal.EncodeError(w, r, err)
	}

	if err := h.uiTemplate.ExecuteTemplate(w, "session.html", h.uiSessions); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func isActive(activeSessionName, sessionName string) bool {
	return activeSessionName == sessionName
}

func (h *UIHandler) GetAllSessions(w http.ResponseWriter, r *http.Request) {

	uisession, ok := h.uiSessions.(*sessions.UiSession)
	if !ok {
		return
		//fmt.Fprintf(w, "Unable to create session: %v", err)
		//internal.EncodeError(w, r, err)
	}

	var sees internal.AllSessionsJSON

	for key := range uisession.UiNodes {
		sees.Sessions = append(sees.Sessions, internal.SessionJSON{
			IsActive:    isActive(uisession.SessionName, key.SessionName),
			SessionName: key.SessionName,
			SessionPin:  key.SessionPin,
		})
	}

	jsonData, err := json.Marshal(sees)
	if err != nil {
		fmt.Fprintf(w, "Unable to create session: %v", err)
	}

	fmt.Printf("json data: %s\n", jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *UIHandler) CreateSessionNew(w http.ResponseWriter, r *http.Request) {
	sesid := r.Header.Get("session-id")
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		// in case of any error
		return
	}

	value := r.FormValue(sessionName)

	err = h.uiSessions.Add(value)
	if err != nil {
		fmt.Fprintf(w, "Unable to create session: %v", err)
		internal.EncodeError(w, r, err)
	}

	uisession, ok := h.uiSessions.(*sessions.UiSession)
	fmt.Printf("uisession %p\n", uisession)
	if !ok {
		fmt.Fprintf(w, "Unable to create session: %v", err)
		internal.EncodeError(w, r, err)
	}

	jsonData, err := json.Marshal(getCreateSessionResponseFromSession(uisession, sesid))
	if err != nil {
		fmt.Fprintf(w, "Unable to create session: %v", err)
	}

	fmt.Printf("json data: %s\n", jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func getCreateSessionResponseFromSession(sessions *sessions.UiSession, sessionId string) internal.SessionJSON {
	return internal.SessionJSON{
		IsActive:    true,
		SessionName: sessions.SessionName,
		SessionPin:  sessions.SessionPin,
		SessionId:   sessionId,
	}
}

func (h *UIHandler) SwitchSession(w http.ResponseWriter, r *http.Request) {
	h.uiSessions = h.uiSessions.Switch(r.FormValue(sessionName))
	if err := h.uiTemplate.ExecuteTemplate(w, "session.html", h.uiSessions); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIHandler) ResumeSession(w http.ResponseWriter, r *http.Request) {
	pin, err := retrievePinFromURL(r)
	if err != nil {
		log.Print(err)
		internal.EncodeError(w, r, err)
	}

	session, err := h.uiSessions.Resume(pin, r.FormValue(sessionName))
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	if err := h.uiTemplate.ExecuteTemplate(w, "session.html", session); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIHandler) Versions(w http.ResponseWriter, r *http.Request) {
	csn := r.FormValue(currentSessionName)
	requestChannel := h.uiSessions.LookUpSession(csn)
	fmt.Printf("uisession1 %p\n", h.uiSessions.(*sessions.UiSession))

	versions, err := h.erigonNode.Version(r.Context(), requestChannel)
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	if err := h.uiTemplate.ExecuteTemplate(w, "versions.html", versions); err != nil {
		fmt.Fprintf(w, "Executing versions template: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIHandler) V2Versions(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	fmt.Printf("uisession1 %p\n", h.uiSessions.(*sessions.UiSession))

	versions, err := h.erigonNode.Version(r.Context(), requestChannel)
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	flags, err := h.erigonNode.Flags(r.Context(), requestChannel)
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	cmdLineArgs := h.erigonNode.CMDLineArgs(r.Context(), requestChannel)

	syncStages, err := h.erigonNode.FindSyncStages(r.Context(), requestChannel)
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	resp := internal.SessionDataJSON{
		Version:     versions,
		Flags:       flags,
		CmdLineArgs: cmdLineArgs,
		SyncStages:  syncStages,
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
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	cmdLineArgs := h.erigonNode.CMDLineArgs(r.Context(), requestChannel)

	if err := h.uiTemplate.ExecuteTemplate(w, "cmd_line.html", cmdLineArgs); err != nil {
		fmt.Fprintf(w, "Executing cmd_line template: %v\n", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIHandler) Flags(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	flags, err := h.erigonNode.Flags(r.Context(), requestChannel)
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	if err := h.uiTemplate.ExecuteTemplate(w, "flags.html", flags); err != nil {
		fmt.Fprintf(w, "Executing flags template: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIHandler) LogList(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	h.erigonNode.ProcessLogList(w, h.uiTemplate, r.FormValue(currentSessionName), requestChannel)
}

func (h *UIHandler) LogHead(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	tail := h.erigonNode.LogHead(url.QueryEscape(r.Form.Get("file")), requestChannel)
	//return TODO SEMANTIC ERRORING

	if err := h.uiTemplate.ExecuteTemplate(w, "log_read.html", tail); err != nil {
		fmt.Fprintf(w, "Executing log_read template: %v", err)
		return
	}
}

func (h *UIHandler) LogTail(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	offset, _ := retrieveSizeStrFrom(r)
	//return TODO SEMANTIC ERRORING

	tail := h.erigonNode.LogTail(url.QueryEscape(r.Form.Get("file")), offset, requestChannel)
	//return TODO SEMANTIC ERRORING

	if err := h.uiTemplate.ExecuteTemplate(w, "log_read.html", tail); err != nil {
		fmt.Fprintf(w, "Executing log_read template: %v", err)
		return
	}
}

func (h *UIHandler) LogDownload(w http.ResponseWriter, r *http.Request) {
	// Handles the use case when operator clicks on the link with the log file name, and this initiates the download of this file
	// to the operator's computer (via browser). See LogReader above which is used in http.ServeContent
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	size, err := retrieveSizeStrFrom(r)
	if err != nil {
		internal.EncodeError(w, r, err)
	}
	filename := r.Form.Get("file")
	if requestChannel == nil {
		internal.EncodeError(w, r, diagnostics.AsBadRequestErr(errors.Errorf("ERROR: Node is not allocated\n")))
	}

	cd := mime.FormatMediaType("attachment", map[string]string{"filename": sessionName + "_" + filename})

	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Type", "application/octet-stream")

	logReader := &erigon_node.LogReader{Filename: filename, RequestChannel: requestChannel, Offset: 0, Total: size, Ctx: r.Context()}
	http.ServeContent(w, r, filename, time.Now(), logReader)
}

func (h *UIHandler) ReOrg(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	h.erigonNode.FindReorgs(r.Context(), w, h.uiTemplate, requestChannel)
}

func (h *UIHandler) BodiesDownload(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	h.erigonNode.BodiesDownload(r.Context(), w, h.uiTemplate, requestChannel)
}

func (h *UIHandler) HeadersDownload(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	h.erigonNode.HeadersDownload(r.Context(), w, h.uiTemplate, requestChannel)
}

func (h *UIHandler) SyncStages(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	syncStages, err := h.erigonNode.FindSyncStages(r.Context(), requestChannel)
	if err != nil {
		internal.EncodeError(w, r, err)
	}

	if err := h.uiTemplate.ExecuteTemplate(w, "sync_stages.html", syncStages); err != nil {
		fmt.Fprintf(w, "Executing sync_stages template: %v", err)
		return
	}
}

func NewUIHandler(
	uiSession sessions.UIService,
	erigonNode erigon_node.Client,
	uiTemplate *template.Template,
) *UIHandler {
	r := &UIHandler{
		Router:     chi.NewRouter(),
		uiSessions: uiSession,
		erigonNode: erigonNode,
		uiTemplate: uiTemplate,
	}

	r.Get("/", r.UI)
	r.Get("/sessions", r.GetAllSessions)

	// Session Handlers
	r.Post("/", r.CreateSession)
	r.Post("/create", r.CreateSessionNew)
	r.Post("/resume", r.ResumeSession)
	r.Post("/switch", r.SwitchSession)

	// Erigon Node data
	r.Post("/versions", r.Versions)
	r.Post("/v2/versions", r.V2Versions)
	r.Post("/cmd_line", r.CMDLine)
	r.Post("/flags", r.Flags)

	r.Post("/log_list", r.LogList)
	r.Post("/log_head", r.LogHead)
	r.Post("/log_tail", r.LogTail)
	r.Post("/log_download", r.LogDownload)

	r.Post("/reorgs", r.ReOrg)
	r.Post("/bodies_download", r.BodiesDownload)
	r.Post("/headers_download", r.HeadersDownload)
	r.Post("/sync_stages", r.SyncStages)

	return r
}

const (
	currentSessionName = "current_session_name"
	sessionName        = "session_name"
)
