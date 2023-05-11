package cmd

import (
	"sync"

	"github.com/google/btree"
)

type UiSession struct {
	lock        sync.Mutex
	Session     bool
	SessionPin  uint64
	SessionName string
	Errors      []string     // Transient field - only filled for the time of template execution
	NodeS       *NodeSession // Transient field - only filled for the time of template execution
	uiNodeTree  *btree.BTreeG[UiNodeSession]
	UiNodes     []UiNodeSession // Transient field - only filled forthe time of template execution
}

func (uiSession *UiSession) appendError(err string) {
	uiSession.lock.Lock()
	defer uiSession.lock.Unlock()
	uiSession.Errors = append(uiSession.Errors, err)
}

// type UiNodeSession struct {
// 	SessionName string
// 	SessionPin  uint64
// }
