package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ledgerwatch/diagnostics/api/internal"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
	"github.com/ledgerwatch/erigonwatch"
)

type APIServices struct {
	ErigonNode   erigon_node.Client
	StoreSession sessions.CacheService
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
	r.Mount(internal.BridgeEndPoint, NewBridgeHandler(services.StoreSession))

	assets, _ := erigonwatch.UIFiles()
	fs := http.FileServer(http.FS(assets))

	r.Mount("/", fs)
	r.HandleFunc("/snapshot-sync", index)
	r.HandleFunc("/sentry-network", index)
	r.HandleFunc("/sentinel-network", index)
	r.HandleFunc("/logs", index)
	r.HandleFunc("/chain", index)
	r.HandleFunc("/data", index)
	r.HandleFunc("/debug", index)
	r.HandleFunc("/testing", index)
	r.HandleFunc("/performance", index)
	r.HandleFunc("/documentation", index)
	r.HandleFunc("/admin", index)
	r.HandleFunc("/downloader", index)

	r.Group(func(r chi.Router) {
		session := sessions.Middleware{CacheService: services.StoreSession}
		r.Use(session.Middleware)
		r.Mount("/api", NewAPIHandler(services.StoreSession, services.ErigonNode))
	})

	return r
}

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./../../web/dist/index.html")
}
