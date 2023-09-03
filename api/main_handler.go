package api

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/assets"
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
	//r.Use(bridge.Middleware)
	r.Use(middleware.RouteHeaders().
		Route("Origin", "*", cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Content-Type", "session-id"},
			AllowCredentials: false, // <----------<<< do not allow credentials
		})).
		Handler)

	r.Mount(internal.HealthCheckEndPoint, HealthCheckHandler())
	r.Mount(internal.ScriptEndPoint, http.FileServer(http.FS(assets.Scripts)))
	r.Mount(internal.SupportEndPoint, NewBridgeHandler(*services.StoreSession))

	r.Group(func(r chi.Router) {
		session := sessions.Middleware{UIService: services.UISessions, CacheService: *services.StoreSession}
		r.Use(session.Middleware)
		r.Mount(internal.UIEndPoint, NewUIHandler(services.UISessions, services.ErigonNode, services.HtmlTemplates))
	})

	return r
}
