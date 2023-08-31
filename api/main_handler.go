package api

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/assets"
	"github.com/ledgerwatch/diagnostics/internal/bridge"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
)

type APIServices struct {
	UISessions    sessions.UIService
	ErigonNode    erigon_node.Client
	StoreSession  *sessions.CacheService
	HtmlTemplates *template.Template
}

func NewHandler(services APIServices) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(bridge.Middleware)

	r.Mount(internal.HealthCheckEndPoint, HealthCheckHandler())
	r.Mount(internal.ScriptEndPoint, http.FileServer(http.FS(assets.Scripts)))
	r.Mount(internal.SupportEndPoint, NewBridgeHandler(*services.StoreSession))

	r.Group(func(r chi.Router) {
		session := sessions.Middleware{UIService: services.UISessions, CacheService: *services.StoreSession}
		r.Use(session.Middleware)
		r.Mount(internal.UIEndPoint, NewUIHandler(services.UISessions, services.ErigonNode, services.HtmlTemplates))
		r.Mount(internal.RouterEndPoint, NewUIRouter(services.UISessions, services.ErigonNode))
	})

	return r
}
