package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
)

var _ http.Handler = &UIRouter{}

type UIRouter struct {
	chi.Router
	uiSessions sessions.UIService
	erigonNode erigon_node.Client
}

func (h *UIRouter) CreateSession(w http.ResponseWriter, r *http.Request) {
	err := h.uiSessions.Add(r.FormValue(sessionName))
	if err != nil {
		fmt.Fprintf(w, "Unable to create session: %v", err)
		internal.EncodeError(w, r, err)
	}
}

func (h *UIRouter) SwitchSession(w http.ResponseWriter, r *http.Request) {
	h.uiSessions = h.uiSessions.Switch(r.FormValue(sessionName))
}

func (h *UIRouter) ResumeSession(w http.ResponseWriter, r *http.Request) {
	pin, err := retrievePinFromURL(r)
	if err != nil {
		log.Print(err)
		internal.EncodeError(w, r, err)
	}

	_, err = h.uiSessions.Resume(pin, r.FormValue(sessionName))
	if err != nil {
		internal.EncodeError(w, r, err)
	}

}

func (h *UIRouter) Fetch(w http.ResponseWriter, r *http.Request) {
	requestChannel := h.uiSessions.LookUpSession(r.FormValue(currentSessionName))
	ok, result := h.erigonNode.Fetch(r.URL.Path, requestChannel)

	if !ok {
		internal.EncodeError(w, r, fmt.Errorf("Fetch failed"))
	}

	_, err := w.Write([]byte(result))

	if err != nil {
		internal.EncodeError(w, r, err)
	}
}

func NewUIRouter(
	uiSession sessions.UIService,
	erigonNode erigon_node.Client,
) *UIRouter {
	r := &UIRouter{
		Router:     chi.NewRouter(),
		uiSessions: uiSession,
		erigonNode: erigonNode,
	}

	r.Get("/", r.Fetch)

	// Session Handlers
	r.Post("/", r.CreateSession)
	r.Post("/resume", r.ResumeSession)
	r.Post("/switch", r.SwitchSession)

	return r
}
