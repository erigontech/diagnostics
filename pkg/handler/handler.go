package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/ledgerwatch/diagnostics/assets"
	"github.com/ledgerwatch/diagnostics/pkg/session"
	"github.com/pkg/errors"
)

const sessionIDCookieName = "sessionId"
const sessionIdCookieDuration = 30 * 24 * 3600 // 30 days

var uiRegex = regexp.MustCompile("^/ui/(cmd_line|flags|log_list|log_head|log_tail|log_download|versions|reorgs|bodies_download|)$")

type UIHandler struct {
	nodeSessions *lru.ARCCache[uint64, *session.Node]
	uiSessions   *lru.ARCCache[string, *session.UI]
	uiTemplate   *template.Template
}

type UIHandlerConf struct {
	MaxNodeSessions int
	MaxUISessions   int
	UITmplPath      string
}

func NewUIHandler(cfg UIHandlerConf) (*UIHandler, error) {
	uiTmpl, err := template.ParseFS(assets.Templates, cfg.UITmplPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing session.html template")
	}
	ns, err := lru.NewARC[uint64, *session.Node](cfg.MaxNodeSessions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create nodeSessions")
	}
	uis, err := lru.NewARC[string, *session.UI](cfg.MaxUISessions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create uiSessions")
	}

	return &UIHandler{
		nodeSessions: ns,
		uiSessions:   uis,
		uiTemplate:   uiTmpl,
	}, nil
}

func (uih *UIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := uiRegex.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}

	cookie, err := r.Cookie(sessionIDCookieName)
	var sessionId string
	var uiSession *session.UI
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
			cookie := http.Cookie{Name: sessionIDCookieName, Value: url.QueryEscape(sessionId), Path: "/", HttpOnly: true, MaxAge: sessionIdCookieDuration}
			http.SetCookie(w, &cookie)
		} else {
			uiSession.AppendError(fmt.Sprintf("Creating new UI session: %v", e))
		}
	}
	if err != nil {
		uiSession.AppendError(fmt.Sprintf("Cookie handling: %v", err))
	}
	// Try to lookup current session name
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Parsing form: %v", err)
		return
	}
	requests := uih.lookupSession(r, uiSession)
	filename := r.Form.Get("file")
	sizeStr := r.Form.Get("size")
	sessionName := r.Form.Get("current_sessionname")
	switch m[1] {
	case "versions":
		result, ok := uih.fetch("/version\n", requests)
		processVersions(w, uih.uiTemplate, ok, result)
		return
	case "cmd_line":
		result, ok := uih.fetch("/cmdline\n", requests)
		processCmdLineArgs(w, uih.uiTemplate, ok, result)
		return
	case "flags":
		result, ok := uih.fetch("/version\n", requests)
		// TODO: boolean should be configureable.
		versions := processVersions(w, uih.uiTemplate, ok, result, true)
		result, ok = uih.fetch("/flags\n", requests)
		processFlags(w, uih.uiTemplate, ok, result, versions)
		return
	case "log_list":
		success, result := uih.fetch("/logs/list\n", requests)
		processLogList(w, uih.uiTemplate, result, uiSession.SessionName, success)
		return
	case "log_head":
		path := fmt.Sprintf("/logs/read?file=%s&offset=0\n", url.QueryEscape(filename))
		success, result := uih.fetch(path, requests)
		processLogPart(w, uih.uiTemplate, result, uiSession.SessionName, success)
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

		path := fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(filename), offset)
		result, ok := uih.fetch(path, requests)
		processLogPart(w, uih.uiTemplate, ok, uiSession.SessionName, result)
		return
	case "log_download":
		size, err := strconv.ParseUint(sizeStr, 10, 64)
		if err != nil {
			fmt.Fprintf(w, "Parsing size %s: %v", sizeStr, err)
			return
		}
		transmitLogFile(r.Context(), r, w, sessionName, filename, size, requests)
		return
	case "reorgs":
		uih.findReorgs(r.Context(), w, uih.uiTemplate, requests)
		return
	case "bodies_download":
		uih.bodiesDownload(r.Context(), w, uih.uiTemplate, requests)
		return
	}
	uiSession.Mx.Lock()
	defer func() {
		uiSession.Session = false
		uiSession.Errors = nil
		uiSession.Node = nil
		uiSession.UINodes = nil
		uiSession.Mx.Unlock()
	}()
	sessionName = r.FormValue("sessionname")
	switch {
	case r.FormValue("new_session") != "":
		// Generate new node session PIN that does not exist yet
		if !uih.validSessionName(sessionName, uiSession) {
			break
		}

		// TODO: set security somewhere above in config.
		insecure := true
		uiSession.Session = true
		uiSession.SessionName = sessionName
		uiSession.SessionPin, uiSession.Node, err = uih.allocateNewNodeSession(insecure)
		if err != nil {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Generating new node session PIN %v", err))
			break
		}

		uiSession.UINodeTree.ReplaceOrInsert(session.UINodeSession{SessionName: sessionName, SessionPin: uiSession.SessionPin})

	case r.FormValue("resume_session") != "":
		// Resume (take over) node session using known PIN.
		pinStr := r.FormValue("pin")
		sessionPin, err := strconv.ParseUint(pinStr, 10, 64)
		if err != nil {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Parsing session PIN %s: %v", pinStr, err))
			break
		}

		var ok bool
		if uiSession.Node, ok = uih.findNodeSession(sessionPin); !ok {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", sessionPin))
			break
		}
		if !uih.validSessionName(sessionName, uiSession) {
			break
		}

		uiSession.Session = true
		uiSession.SessionName = sessionName
		uiSession.SessionPin = sessionPin
		uiSession.UINodeTree.ReplaceOrInsert(session.UINodeSession{SessionName: sessionName, SessionPin: uiSession.SessionPin})
	default:
		// Make one of the previously known sessions active
		for k, vs := range r.Form {
			if len(vs) == 1 {
				if v, ok := uiSession.UINodeTree.Get(session.UINodeSession{SessionName: vs[0]}); ok && fmt.Sprintf("pin%d", v.SessionPin) == k {
					if uiSession.Node, ok = uih.findNodeSession(v.SessionPin); !ok {
						uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
						uiSession.UINodeTree.Delete(v)
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
	uiSession.UINodeTree.Ascend(func(uiNodeSession session.UINodeSession) bool {
		uiSession.UINodes = append(uiSession.UINodes, uiNodeSession)
		return true
	})
	if err := uih.uiTemplate.ExecuteTemplate(w, "session.html", uiSession); err != nil {
		fmt.Fprintf(w, "Failed executing template: %v", err)
		return
	}
}

func (uih *UIHandler) lookupSession(r *http.Request, uiSession *session.UI) chan *session.Request {
	uiSession.Mx.Lock()
	defer uiSession.Mx.Unlock()

	uiSession.Node = nil
	currentSessionName := r.FormValue("current_sessionname")
	if currentSessionName != "" {
		if v, ok := uiSession.UINodeTree.Get(session.UINodeSession{SessionName: currentSessionName}); ok {
			if uiSession.Node, ok = uih.findNodeSession(v.SessionPin); !ok {
				uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d not found", v.SessionPin))
				uiSession.UINodeTree.Delete(v)
			} else {
				uiSession.Session = true
				uiSession.SessionName = currentSessionName
				uiSession.SessionPin = v.SessionPin
			}
		}
	}
	if uiSession.Node != nil {
		return uiSession.Node.Requests
	}
	return nil
}

func (uih *UIHandler) allocateNewNodeSession(insecure bool) (uint64, *session.Node, error) {
	pin, err := generatePIN(insecure)
	if err != nil {
		return pin, nil, err
	}

	for uih.nodeSessions.Contains(pin) {
		pin, err = generatePIN(insecure)
		if err != nil {
			return pin, nil, err
		}
	}

	// TODO: prob could be good to have the "16" parameterized in conf.
	nodeSession := session.NewNode(16)
	uih.nodeSessions.Add(pin, nodeSession)
	return pin, nodeSession, nil
}

func (uih *UIHandler) findNodeSession(pin uint64) (*session.Node, bool) {
	return uih.nodeSessions.Get(pin)
}

func (uih *UIHandler) newUiSession() (string, *session.UI, error) {
	var b [32]byte
	var sessionID string
	_, err := io.ReadFull(rand.Reader, b[:])
	if err == nil {
		sessionID = base64.URLEncoding.EncodeToString(b[:])
	}

	// TODO: might be worth considering parameterizing degree
	deg := 32
	uiSession := session.NewUI(deg)
	if sessionID != "" {
		uih.uiSessions.Add(sessionID, uiSession)
	}
	return sessionID, uiSession, err
}

func (uih *UIHandler) findUiSession(sessionId string) (*session.UI, bool) {
	return uih.uiSessions.Get(sessionId)
}

func (uih *UIHandler) validSessionName(sessionName string, uiSession *session.UI) bool {
	if sessionName == "" {
		uiSession.Errors = append(uiSession.Errors, "empty session name")
		return false
	}
	if uiSession.UINodeTree.Has(session.UINodeSession{SessionName: sessionName}) {
		uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("session with name [%s] already present, choose another name or close [%s]", sessionName, sessionName))
		return false
	}
	return true
}

func (uih *UIHandler) fetch(url string, requests chan *session.Request) (string, bool) {
	if requests == nil {
		return "ERROR: Node is not allocated\n", false
	}
	// Request command line arguments
	nodeReq := session.NewRequest(url)
	requests <- nodeReq
	var sb strings.Builder
	var success bool
	for nodeReq != nil {
		nodeReq.Mx.Lock()
		clear := nodeReq.Served
		if nodeReq.Served {
			// TODO: could be handled by a separate method
			if nodeReq.Err == "" {
				sb.Reset()
				sb.Write(nodeReq.Resp)
				success = true
			} else {
				success = false
				fmt.Fprintf(&sb, "ERROR: %s\n", nodeReq.Err)
				if nodeReq.Retries < 16 {
					clear = false
				}
			}
			// till here
		}
		nodeReq.Mx.Unlock()
		if clear {
			nodeReq = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return sb.String(), success
}

func generatePIN(insecure bool) (uint64, error) {
	if insecure {
		return uint64(mathrand.Int63n(100_000_000)), nil
	}
	max := big.NewInt(100_000_000) // For an 8-digit PIN
	randNum, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}

	return randNum.Uint64(), nil
}
