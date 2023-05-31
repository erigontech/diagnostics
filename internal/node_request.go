package internal

import "sync"

type NodeRequest struct {
	Lock     sync.Mutex
	Url      string
	Served   bool
	Response []byte
	Err      string
	Retries  int
}
