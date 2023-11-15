package bridge

import (
	"net/http"
)

var ErrHTTP2NotSupported = "HTTP2 not supported"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
