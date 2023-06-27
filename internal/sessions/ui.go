package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/btree"
	"github.com/ledgerwatch/diagnostics"
	"github.com/ledgerwatch/diagnostics/internal"
)

type UiSession struct {
	lock        sync.Mutex
	Session     bool
	SessionPin  uint64
	SessionName string

	uiNodeTree *btree.BTreeG[UINodeSession]
	Store      CacheService

	Errors  []string               // Transient field - only filled for the time of template execution
	NodeS   *NodeSession           // Transient field - only filled for the time of template execution
	UiNodes map[UINodeSession]bool // Transient field - only filled forthe time of template execution
}

func (s *UiSession) AppendError(err string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Errors = append(s.Errors, err)
}

func (s *UiSession) ValidSessionName(sessionName string, uiSession *UiSession) bool {
	if sessionName == "" {
		uiSession.Errors = append(uiSession.Errors, "empty session name")
		return false
	}

	if uiSession.uiNodeTree.Has(UINodeSession{SessionName: sessionName}) {
		uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("session with name [%s] already present, choose another name or close [%s]", sessionName, sessionName))
		return false
	}

	return true
}

func (s *UiSession) Generate() (string, *UiSession, error) {
	var b [32]byte
	var sessionId string
	_, err := io.ReadFull(rand.Reader, b[:])
	if err == nil {
		sessionId = base64.URLEncoding.EncodeToString(b[:])
	}

	s.uiNodeTree = btree.NewG(32, func(a, b UINodeSession) bool {
		return strings.Compare(a.SessionName, b.SessionName) < 0
	})

	if sessionId != "" {
		s.Store.AddUISession(sessionId, s)
	}

	return sessionId, s, err
}

func (s *UiSession) Add(sessionName string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.ValidSessionName(sessionName, s) {
		return diagnostics.BadRequest(fmt.Errorf("invalid session name %s", sessionName))
	}

	s.Session = true
	s.SessionName = sessionName
	var err error
	s.SessionPin, s.NodeS, err = s.Store.AllocateNewNodeSession()
	if err != nil {
		s.Errors = append(s.Errors, fmt.Sprintf("Generating new node session PIN %v", err))
		return diagnostics.BadRequest(err)
	}

	s.uiNodeTree.ReplaceOrInsert(UINodeSession{SessionName: sessionName, SessionPin: s.SessionPin})

	s.uiNodeTree.Ascend(func(uiNodeSession UINodeSession) bool {
		if s.UiNodes == nil {
			s.UiNodes = make(map[UINodeSession]bool)
		}
		if _, ok := s.UiNodes[uiNodeSession]; !ok {
			s.UiNodes[uiNodeSession] = true
		}

		return true
	})

	return nil
}

func (s *UiSession) LookUpSession(currentSessionName string) chan *internal.NodeRequest {
	s.NodeS = nil
	if currentSessionName != "" {
		s.updateSession(currentSessionName)
	}

	if s.NodeS != nil {
		return s.NodeS.RequestCh
	}

	return nil
}

func (s *UiSession) Switch(sessionName string) *UiSession {
	if sessionName != "" {
		s.updateSession(sessionName)
	}

	s.uiNodeTree.Ascend(func(uiNodeSession UINodeSession) bool {
		if s.UiNodes == nil {
			s.UiNodes = make(map[UINodeSession]bool)
		}
		if _, ok := s.UiNodes[uiNodeSession]; !ok {
			s.UiNodes[uiNodeSession] = true
		}

		return true
	})

	return s
}

func (s *UiSession) updateSession(currentSessionName string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	v, ok := s.uiNodeTree.Get(UINodeSession{SessionName: currentSessionName})
	if ok {
		var ok bool
		if s.NodeS, ok = s.Store.FindNodeSession(v.SessionPin); !ok {
			s.Errors = append(s.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
			s.uiNodeTree.Delete(v)
		} else {
			s.Session = true
			s.SessionName = currentSessionName
			s.SessionPin = v.SessionPin
		}
	}
}

func (s *UiSession) Resume(pin uint64, sessionName string) (*UiSession, error) {
	var ok bool
	if s.NodeS, ok = s.Store.FindNodeSession(pin); !ok {
		s.Errors = append(s.Errors, fmt.Sprintf("Session %d is not found", pin))
		return nil, diagnostics.NotFound(fmt.Errorf("session %d is not found", pin))
	}

	if !s.ValidSessionName(sessionName, s) {
		return nil, diagnostics.BadRequest(fmt.Errorf("invalid session name %s", sessionName))
	}

	s.Session = true
	s.SessionName = sessionName
	s.SessionPin = pin
	s.uiNodeTree.ReplaceOrInsert(UINodeSession{SessionName: sessionName, SessionPin: s.SessionPin})

	return s, nil
}

func NewUISession(store CacheService) UIService {
	return &UiSession{Store: store}
}

type UIService interface {
	// AppendError appends error to the error array for current session
	AppendError(err string)
	// ValidSessionName validates that name is set and returns true if the name exists in the nodeTree
	ValidSessionName(sessionName string, uiSession *UiSession) bool
	// Generate creates a new UI session and initializes the UI-Node tree and inserts it in to the cache
	Generate() (string, *UiSession, error)
	// LookUpSession checks if session name exits in node tree, if so proceeds in retrieving it from cache
	LookUpSession(currentSessionName string) chan *internal.NodeRequest
	// Add provided a session name creates a session and adds it to the UI-Node tree and to the cache
	Add(sessionName string) error
	// Resume takes over a diagnostics UI session by the provided pin and session name
	Resume(pin uint64, sessionName string) (*UiSession, error)
	// Switch looks up for requested session and updates the session details
	Switch(sessionName string) *UiSession
}

type UINodeSession struct {
	SessionName string
	SessionPin  uint64
}
