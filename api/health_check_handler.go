package api

import (
	"encoding/json"
	"net/http"
)

func HealthCheckHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode("Running!")
	})
}
