package erigon_node

type Versions struct {
	Success        bool   `json:"-"`
	Error          string `json:"-"`
	NodeVersion    uint64 `json:"nodeVersion"`
	SupportVersion uint64 `json:"supportVersion"`
	CodeVersion    string `json:"codeVersion"`
	GitCommit      string `json:"gitCommit"`
}

type PeerNetworkInfo struct {
	LocalAddress  string `json:"localAddress"`  // Local endpoint of the TCP data connection
	RemoteAddress string `json:"remoteAddress"` // Remote endpoint of the TCP data connection
	Inbound       bool   `json:"inbound"`
	Trusted       bool   `json:"trusted"`
	Static        bool   `json:"static"`
}

type PeerInfo struct {
	ENR           string                 `json:"enr,omitempty"` // Ethereum Node Record
	Enode         string                 `json:"enode"`         // Node URL
	ID            string                 `json:"id"`            // Unique node identifier
	Name          string                 `json:"name"`          // Name of the node, including client type, version, OS, custom data
	ErrorCount    int                    `json:"errorCount"`    // Number of errors
	LastSeenError string                 `json:"lastSeenError"` // Last seen error
	Type          string                 `json:"type"`          // Type of connection
	Caps          []string               `json:"caps"`          // Protocols advertised by this peer
	Network       PeerNetworkInfo        `json:"network"`
	Protocols     map[string]interface{} `json:"protocols"` // Sub-protocol specific metadata fields
	BytesIn       int                    `json:"bytesIn"`   // Number of bytes received from the peer
	BytesOut      int                    `json:"bytesOut"`  // Number of bytes sent to the peer
}

type PeersInfo []PeerInfo

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
