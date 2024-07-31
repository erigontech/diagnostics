package api

import (
	"net/http"

	"github.com/erigontech/erigonwatch"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
)

type APIServices struct {
	ErigonNode   erigon_node.Client
	StoreSession sessions.CacheService
}

func NewHandler(services APIServices) http.Handler {
	supportedSubpaths := []string{
		"sentry-network",
		"sentinel-network",
		"downloader",
		"logs",
		"chain",
		"data",
		"debug",
		"testing",
		"performance",
		"documentation",
		"issues",
		"admin",
	}

	r := chi.NewRouter()
	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RouteHeaders().
		Route("Origin", "*", cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Content-Type", "session-id"},
			AllowCredentials: false, // <----------<<< do not allow credentials
		})).
		Handler)

	r.Mount(internal.HealthCheckEndPoint, HealthCheckHandler())
	r.Mount(internal.BridgeEndPoint, NewBridgeHandler(services.StoreSession))

	assets, _ := erigonwatch.UIFiles()
	fs := http.FileServer(http.FS(assets))

	r.Mount("/", fs)

	for _, subpath := range supportedSubpaths {
		addhandler(r, "/"+subpath, fs)
	}

	r.Group(func(r chi.Router) {
		session := sessions.Middleware{CacheService: services.StoreSession}
		r.Use(session.Middleware)
		r.Mount("/api", NewAPIHandler(services.StoreSession, services.ErigonNode))
	})

	return r
}

func addhandler(r *chi.Mux, path string, handler http.Handler) {
	r.Handle(path, http.StripPrefix(path, handler))
}
