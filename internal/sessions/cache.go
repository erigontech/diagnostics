package sessions

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/ledgerwatch/diagnostics/internal"
	"log"
)

var _ CacheService = &Cache{}

type Cache struct {
	NodeSessions *lru.ARCCache[uint64, *NodeSession]
	UISessions   *lru.ARCCache[string, *UiSession]
}

func (s *Cache) AddUISession(sessionId string, uiSession *UiSession) {
	s.UISessions.Add(sessionId, uiSession)
}

func (s *Cache) FindNodeSession(pin uint64) (*NodeSession, bool) {
	return s.NodeSessions.Get(pin)
}

func (s *Cache) FindUISession(sessionId string) (*UiSession, bool) {
	return s.UISessions.Get(sessionId)
}

func (s *Cache) AllocateNewNodeSession() (uint64, *NodeSession, error) {
	pin, err := generatePIN()
	if err != nil {
		return pin, nil, err
	}

	for s.NodeSessions.Contains(pin) {
		pin, err = generatePIN()
		if err != nil {
			return pin, nil, err
		}
	}

	nodeSession := &NodeSession{RequestCh: make(chan *internal.NodeRequest, 16)}
	s.NodeSessions.Add(pin, nodeSession)
	return pin, nodeSession, nil
}

func NewCache(maxNodeSessions int, maxUISessions int) CacheService {
	ns, err := lru.NewARC[uint64, *NodeSession](maxNodeSessions)
	if err != nil {
		log.Printf("failed to create nodeSessions: %v", err)
	}

	uis, err := lru.NewARC[string, *UiSession](maxUISessions)
	if err != nil {
		log.Printf("failed to create uiSessions: %v", err)
	}

	return &Cache{
		NodeSessions: ns,
		UISessions:   uis,
	}

}

type CacheService interface {
	// FindNodeSession retrieves node session from cache
	FindNodeSession(pin uint64) (*NodeSession, bool)
	// FindUISession retrieves the diagnostics UI session based on the session ID
	FindUISession(sessionId string) (*UiSession, bool)
	// AllocateNewNodeSession creates a new node session and inserts it in to the cache
	AllocateNewNodeSession() (uint64, *NodeSession, error)
	// AddUISession inserts in to the cache the specified UI session
	AddUISession(sessionId string, uiSession *UiSession)
}
