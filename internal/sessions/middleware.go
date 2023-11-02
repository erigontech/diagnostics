package sessions

import (
	"net/http"
)

const sessionIdCookieName = "sessionId"

const sessionIdCookieDuration = 30 * 24 * 3600 // 30 days

type Middleware struct {
	CacheService
}

func (s *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO this needs to validate the that the session id in the URL is accessible
		next.ServeHTTP(w, r)
	})
}
