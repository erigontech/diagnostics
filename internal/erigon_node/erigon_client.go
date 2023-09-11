package erigon_node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var _ Client = &NodeClient{}

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
		fmt.Println("Res", string(result), err)
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

func (c *NodeClient) getResultLines(result string) ([]string, error) {
	lines := strings.Split(result, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], SuccessLine) {
		return nil, fmt.Errorf("incorrect response (first line needs to be SUCCESS): %s", result)
	}

	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	return lines[1:], nil
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

	LogFiles(ctx context.Context) (LogFiles, error)
	Log(ctx context.Context, w http.ResponseWriter, file string, offset int64, size int64, download bool) error

	// TODO: refactor the following methods to follow above pattern where appropriate
	FindSyncStages(ctx context.Context, w http.ResponseWriter)
	BodiesDownload(ctx context.Context, w http.ResponseWriter)
	HeadersDownload(ctx context.Context, w http.ResponseWriter)
	FindReorgs(ctx context.Context, w http.ResponseWriter)

	fetch(ctx context.Context, method string, params url.Values) (*NodeRequest, error)
}
