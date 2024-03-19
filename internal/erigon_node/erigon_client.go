package erigon_node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

var _ Client = &NodeClient{}

type Bootnodes []string

type NodeClient struct {
	sync.Mutex
	versions       *Versions
	requestId      uint64
	requestChannel chan *NodeRequest
	nodeId         string
}

func NewClient(nodeId string, requestChannel chan *NodeRequest) Client {
	return &NodeClient{
		nodeId:         nodeId,
		requestChannel: requestChannel,
	}
}

func (c *NodeClient) Version(ctx context.Context) (Versions, error) {
	c.Lock()
	versions := c.versions
	c.Unlock()

	if versions != nil {
		return *versions, nil
	}

	request, err := c.fetch(ctx, "version", nil)

	if err != nil {
		return Versions{}, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return Versions{}, err
	}

	if err := json.Unmarshal(result, &versions); err != nil {
		return Versions{}, err
	}

	c.Lock()
	if c.versions == nil {
		c.versions = versions
	}
	c.Unlock()

	return *versions, nil
}

func (c *NodeClient) checkVersion(ctx context.Context, min uint64) error {
	version, err := c.Version(ctx)

	if err != nil {
		return fmt.Errorf("can't get node version: %w", err)
	}

	if version.NodeVersion < min {
		return fmt.Errorf("required version >= %d. Node version: %d", min, c.versions.NodeVersion)
	}

	return nil
}

func (c *NodeClient) Flags(ctx context.Context) (Flags, error) {
	if err := c.checkVersion(ctx, 2); err != nil {
		return nil, err
	}

	request, err := c.fetch(ctx, "flags", nil)

	if err != nil {
		return nil, err
	}

	var flags Flags

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(result, &flags); err != nil {
		return nil, err
	}

	return flags, nil
}

func (c *NodeClient) CMDLineArgs(ctx context.Context) (CmdLineArgs, error) {
	var args CmdLineArgs

	request, err := c.fetch(ctx, "cmdline", nil)

	if err != nil {
		return args, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(result, &args); err != nil {
		return "", err
	}

	return args, nil
}

func (c *NodeClient) FindPeers(ctx context.Context) (PeersInfo, error) {
	var peers PeersInfo

	request, err := c.fetch(ctx, "peers", nil)

	if err != nil {
		return peers, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(result, &peers); err != nil {
		return nil, err
	}

	return peers, nil
}

func (c *NodeClient) Bootnodes(ctx context.Context) (Bootnodes, error) {
	var bootnodes Bootnodes

	request, err := c.fetch(ctx, "bootnodes", nil)

	if err != nil {
		return bootnodes, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(result, &bootnodes); err != nil {
		return nil, err
	}

	return bootnodes, nil
}

func (c *NodeClient) nextRequestId() string {
	c.Lock()
	id := c.requestId
	c.requestId++
	c.Unlock()
	return strconv.FormatUint(id, 10)
}

func (c *NodeClient) fetch(ctx context.Context, method string, params url.Values) (*NodeRequest, error) {
	if c.requestChannel == nil {
		return nil, fmt.Errorf("ERROR: Node is not allocated")
	}

	nodeRequest := &NodeRequest{
		Responses: make(chan *Response),
		Request: &Request{
			Id:     c.nextRequestId(),
			Method: method,
			Params: &Params{
				NodeId:      c.nodeId,
				QueryParams: params,
			},
		}}

	c.requestChannel <- nodeRequest

	return nodeRequest, nil
}

func (c *NodeClient) GetResponse(ctx context.Context, api string) (interface{}, error) {
	var response interface{}

	request, err := c.fetch(ctx, api, nil)

	if err != nil {
		return response, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return response, err
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return response, err
	}

	return response, nil
}

func NewErigonNodeClient() Client {
	return &NodeClient{}
}

type Client interface {
	// Version retrieves the versions from the Erigon node exposed
	Version(ctx context.Context) (Versions, error)
	// Flags given version has been retrieved, provides the flags
	Flags(ctx context.Context) (Flags, error)
	// CMDLineArgs retrieves the command line arguments provided to run the erigon node
	CMDLineArgs(ctx context.Context) (CmdLineArgs, error)
	FindSyncStages(ctx context.Context) (SyncStageProgress, error)

	LogFiles(ctx context.Context) (LogFiles, error)
	Log(ctx context.Context, w http.ResponseWriter, file string, offset int64, size int64, download bool) error

	DBs(ctx context.Context) (DBs, error)
	Tables(ctx context.Context, db string) (Tables, error)
	Table(ctx context.Context, db string, table string) (Results, error)

	FindReorgs(ctx context.Context, w http.ResponseWriter) (Reorg, error)
	FindPeers(ctx context.Context) (PeersInfo, error)

	Bootnodes(ctx context.Context) (Bootnodes, error)

	ShanphotSync(ctx context.Context) (DownloadStatistics, error)
	ShanphotFiles(ctx context.Context) (ShanshotFilesList, error)

	GetResponse(ctx context.Context, api string) (interface{}, error)

	// TODO: refactor the following methods to follow above pattern where appropriate
	BodiesDownload(ctx context.Context, w http.ResponseWriter)
	HeadersDownload(ctx context.Context, w http.ResponseWriter)

	fetch(ctx context.Context, method string, params url.Values) (*NodeRequest, error)
}
