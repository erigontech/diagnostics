package session

import (
	"sync"
)

// Node corresponds to one Erigon node connected via "erigon support" bridge to an operator
type Node struct {
	mx             sync.Mutex
	Connected      bool
	RemoteAddr     string
	SupportVersion uint64        // Version of the erigon support command.
	Requests       chan *Request // Incoming metric requests.
}

func NewNode(len int) *Node {
	return &Node{
		Requests: make(chan *Request, len),
	}
}

func (n *Node) Connect(remoteAddr string) {
	n.mx.Lock()
	defer n.mx.Unlock()

	n.Connected = true
	n.RemoteAddr = remoteAddr
}

func (n *Node) Disconnect() {
	n.mx.Lock()
	defer n.mx.Unlock()

	n.Connected = false
}

type Request struct {
	Mx      sync.Mutex
	URL     string
	Served  bool
	Resp    []byte
	Err     string
	Retries int
}

func NewRequest(url string) *Request {
	return &Request{
		URL: url,
	}
}
