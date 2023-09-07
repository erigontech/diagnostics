package erigon_node

type Versions struct {
	Success        bool   `json:"success"`
	Error          string `json:"error"`
	NodeVersion    uint64 `json:"node_version"`
	SupportVersion uint64 `json:"support_version"`
	CodeVersion    string `json:"code_version"`
	GitCommit      string `json:"git_commit"`
}

type Flags struct {
	Success     bool              `json:"success"`
	Error       string            `json:"error"`
	FlagPayload map[string]string `json:"flag_payload"`
}

type CmdLineArgs struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Args    string `json:"args"`
}

type BodyDownload struct {
	Legends       [9]bool
	BlockNum      uint64
	Pre1          []struct{}
	Pre10         []struct{}
	Pre100        []struct{}
	Pre1_000      []struct{}
	Pre10_000     []struct{}
	Pre100_000    []struct{}
	Pre1_000_000  []struct{}
	Pre10_000_000 []struct{}
	States        []SnapshotItem
}

type HeaderDownload struct {
	Legends       [9]bool
	HeaderNum     uint64
	Pre1          []struct{}
	Pre10         []struct{}
	Pre100        []struct{}
	Pre1_000      []struct{}
	Pre10_000     []struct{}
	Pre100_000    []struct{}
	Pre1_000_000  []struct{}
	Pre10_000_000 []struct{}
	States        []SnapshotItem
}

type SnapshotItem struct {
	Id    uint64
	State byte
}

const SuccessLine = "SUCCESS"
