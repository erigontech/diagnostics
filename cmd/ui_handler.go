package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"math/big"
	weakrand "math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/btree"
	lru "github.com/hashicorp/golang-lru/v2"
)

const sessionIdCookieName = "sessionId"
const sessionIdCookieDuration = 30 * 24 * 3600 // 30 days

var uiRegex = regexp.MustCompile("^/ui/(cmd_line|log_list|log_head|log_tail|log_download|versions|reorgs|bodies_download|)$")

func (uih *UiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := uiRegex.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	cookie, err := r.Cookie(sessionIdCookieName)
	var sessionId string
	var uiSession *UiSession
	var sessionFound bool
	if err == nil && cookie.Value != "" {
		sessionId, err = url.QueryUnescape(cookie.Value)
		if err == nil {
			uiSession, sessionFound = uih.findUiSession(sessionId)
		}
	}
	if !sessionFound {
		var e error
		sessionId, uiSession, e = uih.newUiSession()
		if e == nil {
			cookie := http.Cookie{Name: sessionIdCookieName, Value: url.QueryEscape(sessionId), Path: "/", HttpOnly: true, MaxAge: sessionIdCookieDuration}
			http.SetCookie(w, &cookie)
		} else {
			uiSession.appendError(fmt.Sprintf("Creating new UI session: %v", e))
		}
	}
	if err != nil {
		uiSession.appendError(fmt.Sprintf("Cookie handling: %v", err))
	}
	// Try to lookup current session name
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Parsing form: %v", err)
		return
	}
	requestChannel := uih.lookupSession(r, uiSession)
	filename := r.Form.Get("file")
	sizeStr := r.Form.Get("size")
	sessionName := r.Form.Get("current_sessionname")
	switch m[1] {
	case "versions":
		success, result := uih.fetch("/version\n", requestChannel)
		processVersions(w, uih.uiTemplate, success, result)
		return
	case "cmd_line":
		success, result := uih.fetch("/cmdline\n", requestChannel)
		processCmdLineArgs(w, uih.uiTemplate, success, result)
		return
	case "log_list":
		success, result := uih.fetch("/logs/list\n", requestChannel)
		processLogList(w, uih.uiTemplate, success, uiSession.SessionName, result)
		return
	case "log_head":
		success, result := uih.fetch(fmt.Sprintf("/logs/read?file=%s&offset=0\n", url.QueryEscape(filename)), requestChannel)
		processLogPart(w, uih.uiTemplate, success, uiSession.SessionName, result)
		return
	case "log_tail":
		size, err := strconv.ParseUint(sizeStr, 10, 64)
		if err != nil {
			fmt.Fprintf(w, "Parsing size %s: %v", sizeStr, err)
			return
		}
		var offset uint64
		if size > 16*1024 {
			offset = size - 16*1024
		}
		success, result := uih.fetch(fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(filename), offset), requestChannel)
		processLogPart(w, uih.uiTemplate, success, uiSession.SessionName, result)
		return
	case "log_download":
		size, err := strconv.ParseUint(sizeStr, 10, 64)
		if err != nil {
			fmt.Fprintf(w, "Parsing size %s: %v", sizeStr, err)
			return
		}
		transmitLogFile(r.Context(), r, w, sessionName, filename, size, requestChannel)
		return
	case "reorgs":
		uih.findReorgs(r.Context(), w, uih.uiTemplate, requestChannel)
		return
	case "bodies_download":
		uih.bodiesDownload(r.Context(), w, uih.uiTemplate, requestChannel)
		return
	}
	uiSession.lock.Lock()
	defer func() {
		uiSession.Session = false
		uiSession.Errors = nil
		uiSession.NodeS = nil
		uiSession.UiNodes = nil
		uiSession.lock.Unlock()
	}()
	sessionName = r.FormValue("sessionname")
	switch {
	case r.FormValue("new_session") != "":
		// Generate new node session PIN that does not exist yet
		if !uih.validSessionName(sessionName, uiSession) {
			break
		}
		uiSession.Session = true
		uiSession.SessionName = sessionName
		uiSession.SessionPin, uiSession.NodeS, err = uih.allocateNewNodeSession()
		if err != nil {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Generating new node session PIN %v", err))
			break
		}
		uiSession.uiNodeTree.ReplaceOrInsert(UiNodeSession{SessionName: sessionName, SessionPin: uiSession.SessionPin})
	case r.FormValue("resume_session") != "":
		// Resume (take over) node session using known PIN
		pinStr := r.FormValue("pin")
		sessionPin, err := strconv.ParseUint(pinStr, 10, 64)
		if err != nil {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Parsing session PIN %s: %v", pinStr, err))
			break
		}
		var ok bool
		if uiSession.NodeS, ok = uih.findNodeSession(sessionPin); !ok {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", sessionPin))
			break
		}
		if !uih.validSessionName(sessionName, uiSession) {
			break
		}
		uiSession.Session = true
		uiSession.SessionName = sessionName
		uiSession.SessionPin = sessionPin
		uiSession.uiNodeTree.ReplaceOrInsert(UiNodeSession{SessionName: sessionName, SessionPin: uiSession.SessionPin})
	default:
		// Make one of the previously known sessions active
		for k, vs := range r.Form {
			if len(vs) == 1 {
				if v, ok := uiSession.uiNodeTree.Get(UiNodeSession{SessionName: vs[0]}); ok && fmt.Sprintf("pin%d", v.SessionPin) == k {
					if uiSession.NodeS, ok = uih.findNodeSession(v.SessionPin); !ok {
						uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
						uiSession.uiNodeTree.Delete(v)
						break
					}
					uiSession.Session = true
					uiSession.SessionName = vs[0]
					uiSession.SessionPin = v.SessionPin
				}
			}
		}
	}
	// Populate transient field UiNodes to display the buttons (with the labels)
	uiSession.uiNodeTree.Ascend(func(uiNodeSession UiNodeSession) bool {
		uiSession.UiNodes = append(uiSession.UiNodes, uiNodeSession)
		return true
	})
	if err := uih.uiTemplate.ExecuteTemplate(w, "session.html", uiSession); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		return
	}
}

func (uih *UiHandler) lookupSession(r *http.Request, uiSession *UiSession) chan *NodeRequest {
	uiSession.lock.Lock()
	defer uiSession.lock.Unlock()
	uiSession.NodeS = nil
	currentSessionName := r.FormValue("current_sessionname")
	if currentSessionName != "" {
		if v, ok := uiSession.uiNodeTree.Get(UiNodeSession{SessionName: currentSessionName}); ok {
			if uiSession.NodeS, ok = uih.findNodeSession(v.SessionPin); !ok {
				uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
				uiSession.uiNodeTree.Delete(v)
			} else {
				uiSession.Session = true
				uiSession.SessionName = currentSessionName
				uiSession.SessionPin = v.SessionPin
			}
		}
	}
	if uiSession.NodeS != nil {
		return uiSession.NodeS.requestCh
	}
	return nil
}

type UiHandler struct {
	nodeSessionsLock sync.Mutex
	nodeSessions     *lru.ARCCache[uint64, *NodeSession]
	uiSessionsLock   sync.Mutex
	uiSessions       map[string]*UiSession
	uiTemplate       *template.Template
}

func (uih *UiHandler) allocateNewNodeSession() (uint64, *NodeSession, error) {
	uih.nodeSessionsLock.Lock()
	defer uih.nodeSessionsLock.Unlock()
	pin, err := generatePIN()
	if err != nil {
		return pin, nil, err
	}

	for uih.nodeSessions.Contains(pin) {
		pin, err = generatePIN()
		if err != nil {
			return pin, nil, err
		}
	}

	nodeSession := &NodeSession{requestCh: make(chan *NodeRequest, 16)}
	uih.nodeSessions.Add(pin, nodeSession)
	return pin, nodeSession, nil
}

func (uih *UiHandler) findNodeSession(pin uint64) (*NodeSession, bool) {
	nodeSession, ok := uih.nodeSessions.Get(pin)
	return nodeSession, ok
}

func (uih *UiHandler) newUiSession() (string, *UiSession, error) {
	var b [32]byte
	var sessionId string
	_, err := io.ReadFull(rand.Reader, b[:])
	if err == nil {
		sessionId = base64.URLEncoding.EncodeToString(b[:])
	}
	uiSession := &UiSession{uiNodeTree: btree.NewG[UiNodeSession](32, func(a, b UiNodeSession) bool {
		return strings.Compare(a.SessionName, b.SessionName) < 0
	})}
	uih.uiSessionsLock.Lock()
	defer uih.uiSessionsLock.Unlock()
	if sessionId != "" {
		uih.uiSessions[sessionId] = uiSession
	}
	return sessionId, uiSession, err
}

func (uih *UiHandler) findUiSession(sessionId string) (*UiSession, bool) {
	uih.uiSessionsLock.Lock()
	defer uih.uiSessionsLock.Unlock()
	uiSession, ok := uih.uiSessions[sessionId]
	return uiSession, ok
}

func (uih *UiHandler) validSessionName(sessionName string, uiSession *UiSession) bool {
	if sessionName == "" {
		uiSession.Errors = append(uiSession.Errors, "empty session name")
		return false
	}
	if uiSession.uiNodeTree.Has(UiNodeSession{SessionName: sessionName}) {
		uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("session with name [%s] already present, choose another name or close [%s]", sessionName, sessionName))
		return false
	}
	return true
}

func (uih *UiHandler) fetch(url string, requestChannel chan *NodeRequest) (bool, string) {
	if requestChannel == nil {
		return false, fmt.Sprintf("ERROR: Node is not allocated\n")
	}
	// Request command line arguments
	nodeRequest := &NodeRequest{url: url}
	requestChannel <- nodeRequest
	var sb strings.Builder
	var success bool
	for nodeRequest != nil {
		nodeRequest.lock.Lock()
		clear := nodeRequest.served
		if nodeRequest.served {
			if nodeRequest.err == "" {
				sb.Reset()
				sb.Write(nodeRequest.response)
				success = true
			} else {
				success = false
				fmt.Fprintf(&sb, "ERROR: %s\n", nodeRequest.err)
				if nodeRequest.retries < 16 {
					clear = false
				}
			}
		}
		nodeRequest.lock.Unlock()
		if clear {
			nodeRequest = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return success, sb.String()
}

func generatePIN() (uint64, error) {
	if insecure {
		return uint64(weakrand.Int63n(100_000_000)), nil
	}
	max := big.NewInt(100_000_000) // For an 8-digit PIN
	randNum, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return randNum.Uint64(), nil
}
