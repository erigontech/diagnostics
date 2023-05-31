package sessions

import (
	"github.com/ledgerwatch/diagnostics/internal"
	"sync"
)

// NodeSession corresponds to one Erigon node connected via "erigon support" bridge to an operator
type NodeSession struct {
	lock           sync.Mutex
	Connected      bool
	RemoteAddr     string
	SupportVersion uint64                     // Versions of the erigon support command
	RequestCh      chan *internal.NodeRequest // Channel for incoming metrics requests
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

type NodeService interface {
	// Connect sets the appropriate fields for connect
	Connect(remoteAddr string)
	// Disconnect unsets the connect field
	Disconnect()
}
