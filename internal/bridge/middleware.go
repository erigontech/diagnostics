package bridge

import "net/http"

var ErrHTTP2NotSupported = "HTTP2 not supported"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !r.ProtoAtLeast(2, 0) {
			http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
			return
		}

		_, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
			return
		}

		next.ServeHTTP(w, r)
	})
}
