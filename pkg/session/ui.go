package session

import (
	"strings"
	"sync"

	"github.com/google/btree"
)

type UI struct {
	Mx          sync.Mutex
	Session     bool
	SessionPin  uint64
	SessionName string
	Errors      []string // Transient field - only filled for the time of template execution
	Node        *Node    // Transient field - only filled for the time of template execution
	UINodeTree  *btree.BTreeG[UINodeSession]
	UINodes     []UINodeSession // Transient field - only filled for the time of template execution
}

func NewUI(degree int) *UI {
	return &UI{
		UINodeTree: btree.NewG(degree, func(a, b UINodeSession) bool {
			return strings.Compare(a.SessionName, b.SessionName) < 0
		}),
	}
}

func (ui *UI) AppendError(err string) {
	ui.Mx.Lock()
	defer ui.Mx.Unlock()

	ui.Errors = append(ui.Errors, err)
}

type UINodeSession struct {
	SessionName string
	SessionPin  uint64
}
