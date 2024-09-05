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

type NodeClient struct {
	lock           sync.Mutex
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

func (c *NodeClient) nextRequestId() string {
	c.lock.Lock()
	id := c.requestId
	c.requestId++
	c.lock.Unlock()
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
	FindSyncStages(ctx context.Context) (SyncStageProgress, error)
	Log(ctx context.Context, w http.ResponseWriter, file string, offset int64, size int64, download bool) error
	Tables(ctx context.Context, db string) (Tables, error)
	Table(ctx context.Context, db string, table string) (Results, error)
	FindReorgs(ctx context.Context, w http.ResponseWriter) (Reorg, error)
	GetResponse(ctx context.Context, api string) (interface{}, error)

	// TODO: refactor the following methods to follow above pattern where appropriate
	BodiesDownload(ctx context.Context, w http.ResponseWriter)
	HeadersDownload(ctx context.Context, w http.ResponseWriter)

	FindProfile(ctx context.Context, profile string) ([]byte, error)

	fetch(ctx context.Context, method string, params url.Values) (*NodeRequest, error)
}
