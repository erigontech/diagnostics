package sessions

import (
	"strconv"
	"sync"
)

type UISession struct {
	lock       sync.Mutex
	SessionPin uint64
	Store      CacheService
	Nodes      map[string]*NodeSession
}

func (s *UISession) Attach(ns *NodeSession) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Nodes[ns.NodeInfo.Id] = ns
}

func (s *UISession) Detach(nodeId string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.Nodes, nodeId)
}

func (s *UISession) IsActive() bool {
	return s != nil && len(s.Nodes) > 0
}

func NewUISession(sessionId string, store CacheService) (*UISession, error) {
	pin, err := strconv.ParseUint(sessionId, 10, 64)

	if err != nil {
		return nil, err
	}

	return &UISession{Store: store, Nodes: map[string]*NodeSession{}, SessionPin: pin}, nil
}
