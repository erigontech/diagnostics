package sessions

import (
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/erigontech/diagnostics/internal/erigon_node"
)

var _ CacheService = &Cache{}

type Cache struct {
	NodeSessions *lru.Cache[string, *NodeSession]
	UISessions   *lru.Cache[string, *UISession]
	uiNodeMap    map[string]map[string]*NodeSession
}

func (s *Cache) CreateUISession(sessionId string) (*UISession, error) {
	session, err := NewUISession(sessionId, s)

	if err != nil {
		return nil, err
	}

	s.UISessions.Add(sessionId, session)

	for _, node := range s.uiNodeMap[sessionId] {
		session.Attach(node)
	}

	return session, nil
}

func (s *Cache) FindNodeSession(sessionId string) (*NodeSession, bool) {
	return s.NodeSessions.Get(sessionId)
}

func (s *Cache) FindUISession(sessionId string) (*UISession, bool) {
	return s.UISessions.Get(sessionId)
}

func (s *Cache) CreateNodeSession(node *NodeInfo) (*NodeSession, error) {
	requestCh := make(chan *erigon_node.NodeRequest)

	nodeSession := &NodeSession{
		RequestCh:    requestCh,
		Client:       erigon_node.NewClient(node.Id, requestCh),
		SessionCache: s,
		NodeInfo:     node,
	}

	s.NodeSessions.Add(node.Id, nodeSession)
	return nodeSession, nil
}

func NewCache(maxNodeSessions int, maxUISessions int) (CacheService, error) {

	uis, err := lru.New[string, *UISession](maxUISessions)

	if err != nil {
		return nil, err
	}

	cache := &Cache{
		UISessions: uis,
		uiNodeMap:  map[string]map[string]*NodeSession{},
	}

	cache.NodeSessions, err = lru.NewWithEvict[string, *NodeSession](maxNodeSessions, func(key string, value *NodeSession) {

		for _, session := range value.UISessions {
			if nodes, ok := cache.uiNodeMap[session]; ok {
				delete(nodes, key)

				if len(nodes) == 0 {
					delete(cache.uiNodeMap, session)
				}
			}

			if uiSession, ok := cache.UISessions.Get(session); ok {
				uiSession.Detach(value.NodeInfo.Id)
			}
		}
	})

	if err != nil {
		return nil, err
	}

	return cache, nil
}

type CacheService interface {
	// FindNodeSession retrieves node session from cache
	FindNodeSession(nodeId string) (*NodeSession, bool)
	// FindUISession retrieves the diagnostics UI session based on the session ID
	FindUISession(sessionId string) (*UISession, bool)
	// AllocateNewNodeSession creates a new node session and inserts it in to the cache
	CreateNodeSession(node *NodeInfo) (*NodeSession, error)
	// AddUISession inserts in to the cache the specified UI session
	CreateUISession(sessionId string) (*UISession, error)
}
