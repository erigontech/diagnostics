package erigon_node

import (
	"context"
	"encoding/json"
	"fmt"
)

type Params struct {
	NodeId       string          `json:"nodeId"`
	MethodParams json.RawMessage `json:"methodParams,omitempty"`
}

type Request struct {
	Method string  `json:"method"`
	Id     string  `json:"id"`
	Params *Params `json:"params,omitempty"`
	Notif  bool    `json:"-"`
}

type Response struct {
	Id     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
	Last   bool            `json:"last,omitempty"`
}

type Error struct {
	Code    int64            `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

type NodeRequest struct {
	Request   *Request
	Responses chan *Response
	Retries   uint
}

func (n *NodeRequest) nextResult(ctx context.Context) (bool, json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return false, nil, ctx.Err()
	case response := <-n.Responses:
		if response.Error != nil {
			return false, nil, response.Error
		}

		return !response.Last, response.Result, nil
	}
}
