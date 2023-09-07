package internal

import "github.com/ledgerwatch/diagnostics/internal/erigon_node"

type SessionDataJSON struct {
	Version     erigon_node.Versions    `json:"version"`
	Flags       erigon_node.Flags       `json:"flags"`
	CmdLineArgs erigon_node.CmdLineArgs `json:"cmd_line_args"`
	SyncStages  map[string]string       `json:"sync_stages"`
}

type SessionJSON struct {
	IsActive    bool   `json:"is_active"`
	SessionName string `json:"session_name"`
	SessionPin  uint64 `json:"session_pin"`
	SessionId   string `json:"session_id"`
}

type AllSessionsJSON struct {
	Sessions []SessionJSON `json:"sessions"`
}

type SnapshotItemJSON struct {
	Id    uint64 `json:"id"`
	State byte   `json:"state"`
}

type BodyDownloadJSON struct {
	Legends       [9]bool            `json:"legends"`
	BlockNum      uint64             `json:"block_num"`
	Pre1          []struct{}         `json:"pre_1"`
	Pre10         []struct{}         `json:"pre_10"`
	Pre100        []struct{}         `json:"pre_100"`
	Pre1_000      []struct{}         `json:"pre_1_000"`
	Pre10_000     []struct{}         `json:"pre_10_000"`
	Pre100_000    []struct{}         `json:"pre_100_000"`
	Pre1_000_000  []struct{}         `json:"pre_1_000_000"`
	Pre10_000_000 []struct{}         `json:"pre_10_000_000"`
	States        []SnapshotItemJSON `json:"states"`
}

type HeaderDownloadJSON struct {
	Legends       [9]bool            `json:"legends"`
	HeaderNum     uint64             `json:"header_num"`
	Pre1          []struct{}         `json:"pre_1"`
	Pre10         []struct{}         `json:"pre_10"`
	Pre100        []struct{}         `json:"pre_100"`
	Pre1_000      []struct{}         `json:"pre_1_000"`
	Pre10_000     []struct{}         `json:"pre_10_000"`
	Pre100_000    []struct{}         `json:"pre_100_000"`
	Pre1_000_000  []struct{}         `json:"pre_1_000_000"`
	Pre10_000_000 []struct{}         `json:"pre_10_000_000"`
	States        []SnapshotItemJSON `json:"states"`
}

type LogListItemJSON struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	PrintedSize string `json:"printed_size"`
}

type LogListJSON struct {
	Success     bool              `json:"success"`
	Error       string            `json:"error"`
	SessionName string            `json:"session_name"`
	List        []LogListItemJSON `json:"list"`
}
