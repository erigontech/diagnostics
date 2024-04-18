package sessions

import (
	"encoding/json"
	"sync"

	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
)

type ports struct {
	Discovery uint32 `json:"discovery,omitempty"`
	Listener  uint32 `json:"listener,omitempty"`
}

type enode struct {
	Enode        string `json:"enode,omitempty"`
	Enr          string `json:"enr,omitempty"`
	Ports        *ports `json:"ports,omitempty"`
	ListenerAddr string `json:"listener_addr,omitempty"`
}

type NodeInfo struct {
	Id        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Protocols json.RawMessage `json:"protocols,omitempty"`
	Enodes    []enode         `json:"enodes,omitempty"`
}

// NodeSession corresponds to one Erigon node connected via "erigon support" bridge to an operator
type NodeSession struct {
	lock         sync.Mutex
	Connected    bool
	RemoteAddr   string
	Client       erigon_node.Client
	RequestCh    chan *erigon_node.NodeRequest // Channel for incoming metrics requests
	UISessions   []string
	SessionCache *Cache
	NodeInfo     *NodeInfo
}

func (ns *NodeSession) Connect(remoteAddr string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.Connected = true
	ns.RemoteAddr = remoteAddr
}

func (ns *NodeSession) Disconnect() {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.Connected = false
}

func NewNodeSession() NodeService {
	return &NodeSession{}
}

func (ns *NodeSession) AttachSessions(sessions []string) error {
	for _, session := range sessions {
		ns.UISessions = append(ns.UISessions, session)

		nodes, ok := ns.SessionCache.uiNodeMap[session]

		if !ok {
			nodes = map[string]*NodeSession{}
			ns.SessionCache.uiNodeMap[session] = nodes
		}

		nodes[ns.NodeInfo.Id] = ns

		if uiSession, ok := ns.SessionCache.UISessions.Get(session); ok {
			uiSession.Attach(ns)
		}
	}

	return nil
}

type NodeService interface {
	// Connect sets the appropriate fields for connect
	Connect(remoteAddr string)
	// Disconnect unsets the connect field
	Disconnect()
}
