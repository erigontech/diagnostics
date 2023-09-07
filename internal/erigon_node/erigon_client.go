package erigon_node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var _ Client = &NodeClient{}

type NodeClient struct {
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
	request, err := c.fetch(ctx, "version", nil)

	var versions Versions

	if err != nil {
		return versions, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		versions.Error = err.Error()
		return versions, nil
	}

	if err := json.Unmarshal(result, &versions); err != nil {
		versions.Error = err.Error()
	}

	c.versions = &versions

	return versions, nil
}

func (c *NodeClient) Flags(ctx context.Context) (Flags, error) {
	// If Versions have not been retrieved, go ahead and retrieve them
	if c.versions == nil {
		_, err := c.Version(ctx)
		if err != nil {
			//fmt.Fprintf(w, "Unable to process flag due to inability to get node version: %s", versions.Error)
			return Flags{}, err
		}
	}

	if c.versions.NodeVersion < 2 {
		//fmt.Fprintf(w, "Flags only support version >= 2. Node version: %d", versions.NodeVersion)
		// TODO semeantic error propagate
		return Flags{}, nil
	}

	// Retrieving the data from the node
	request, err := c.fetch(ctx, "flags", nil)

	if err != nil {
		return Flags{}, err
	}

	var flags Flags

	_, result, err := request.nextResult(ctx)

	if err != nil {
		if err := json.Unmarshal(result, &flags.FlagPayload); err != nil {
			flags.Error = err.Error()
		}
	} else {
		flags.Error = err.Error()
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

	if err == nil {
		var cargs []string
		if err := json.Unmarshal(result, &cargs); err != nil {
			args.Success = false
			args.Error = err.Error()
		} else {
			args.Success = true
			args.Args = strings.Join(cargs, " ")
		}
	} else {
		args.Success = false
		args.Error = err.Error()
	}

	return args, nil
}

func (c *NodeClient) nextRequestId() string {
	id := strconv.FormatUint(c.requestId, 10)
	c.requestId++
	return id
}

func (c *NodeClient) fetch(ctx context.Context, method string, params interface{}) (*NodeRequest, error) {
	if c.requestChannel == nil {
		return nil, fmt.Errorf("ERROR: Node is not allocated")
	}

	jsonMsg, err := json.Marshal(params)

	if err != nil {
		return nil, err
	}

	nodeRequest := &NodeRequest{
		Responses: make(chan *Response),
		Request: &Request{
			Id:     c.nextRequestId(),
			Method: method,
			Params: &Params{
				NodeId:       c.nodeId,
				MethodParams: jsonMsg,
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

	// TODO: refactor the following methods to follow above pattern where appropriate
	FindSyncStages(ctx context.Context, w http.ResponseWriter)
	BodiesDownload(ctx context.Context, w http.ResponseWriter)
	HeadersDownload(ctx context.Context, w http.ResponseWriter)
	FindReorgs(ctx context.Context, w http.ResponseWriter)
	ProcessLogList(ctx context.Context, w http.ResponseWriter, sessionName string) error
	LogHead(ctx context.Context, filename string) (LogPart, error)
	LogTail(ctx context.Context, filename string, offset uint64) (LogPart, error)

	fetch(ctx context.Context, method string, params interface{}) (*NodeRequest, error)
}
