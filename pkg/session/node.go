package session

import "sync"

// Node corresponds to one Erigon node connected via "erigon support" bridge to an operator
type Node struct {
	mx             sync.Mutex
	Connected      bool
	RemoteAddr     string
	SupportVersion uint64        // Version of the erigon support command.
	requests       chan *Request // Incoming metric requests.
}

func NewNode(len int) *Node {
	return &Node{
		requests: make(chan *Request, len),
	}
}

func (n *Node) connect(remoteAddr string) {
	n.mx.Lock()
	defer n.mx.Unlock()

	n.Connected = true
	n.RemoteAddr = remoteAddr
}

func (n *Node) disconnect() {
	n.mx.Lock()
	defer n.mx.Unlock()

	n.Connected = false
}

type Request struct {
	lock     sync.Mutex
	url      string
	served   bool
	response []byte
	err      string
	retries  int
}

func NewRequest() *Request {
	return &Request{}
}
