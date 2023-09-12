package erigon_node

type Versions struct {
	Success        bool   `json:"-"`
	Error          string `json:"-"`
	NodeVersion    uint64 `json:"nodeVersion"`
	SupportVersion uint64 `json:"supportVersion"`
	CodeVersion    string `json:"codeVersion"`
	GitCommit      string `json:"gitCommit"`
}

type Flags map[string]interface{}

type CmdLineArgs string

type LogFiles []LogFile

type LogFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type LogContent struct {
	Offset int64  `json:"offset"`
	Size   int64  `json:"size"`
	Chunk  []byte `json:"chunk"`
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
