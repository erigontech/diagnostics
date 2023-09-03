package sessions

import (
	"fmt"
	"net/http"
	"net/url"
)

const sessionIdCookieName = "sessionId"

const sessionIdCookieDuration = 30 * 24 * 3600 // 30 days

type Middleware struct {
	UIService
	CacheService
}

func (s *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Looking up Cookie and retrieves session from cache
		cookie, err := r.Cookie(sessionIdCookieName)
		var sessionId string
		var sessionFound bool
		var uiSession *UiSession
		if err == nil && cookie.Value != "" {
			sessionId, err = url.QueryUnescape(cookie.Value)
			if err == nil {
				uiSession, sessionFound = s.FindUISession(sessionId)
			} else { //If cookie does not contain a session-id look for session-id in header
				sessionId = r.Header.Get("session-id")
				if sessionId != "" {
					uiSession, sessionFound = s.FindUISession(sessionId)
				}
			}
		} else { //If cookie is empty look for session-id in header
			sessionId = r.Header.Get("session-id")
			if sessionId != "" {
				uiSession, sessionFound = s.FindUISession(sessionId)
			}
		}

		// Creates new session and sets cookie
		if !sessionFound {
			fmt.Println("Creating new session")
			sessionId, uiSession, err = s.Generate()
			if err == nil {
				cookie := http.Cookie{Name: sessionIdCookieName, Value: url.QueryEscape(sessionId), Path: "/", HttpOnly: true, MaxAge: sessionIdCookieDuration}
				http.SetCookie(w, &cookie)
			} else {
				uiSession.AppendError(fmt.Sprintf("Creating new UI session: %v", err))
			}
		}
		if err != nil {
			uiSession.AppendError(fmt.Sprintf("Cookie handling: %v", err))
		}

		r.Header.Add("session-id", sessionId)

		next.ServeHTTP(w, r)
	})
}
