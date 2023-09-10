package erigon_node

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *NodeClient) LogFiles(ctx context.Context) (LogFiles, error) {
	request, err := c.fetch(ctx, "logs", nil)

	if err != nil {
		return nil, err
	}

	var files LogFiles

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(result, &files); err != nil {
		return nil, err
	}

	return files, nil

}

func (c *NodeClient) Log(ctx context.Context, w http.ResponseWriter, file string, offset int64, size int64, download bool) error {
	var params url.Values

	if offset > 0 || size > 0 {
		params = url.Values{
			"offset": []string{strconv.FormatInt(offset, 10)},
		}

		if size > 0 {
			params.Set("size", strconv.FormatInt(size, 10))
		}
	}

	request, err := c.fetch(ctx, "logs/"+file, params)

	if err != nil {
		return err
	}

	for {
		more, result, err := request.nextResult(ctx)

		if err != nil {
			return err
		}

		var content LogContent

		if err := json.Unmarshal(result, &content); err != nil {
			return err
		}

		if _, err := w.Write(content.Chunk); err != nil {
			return err
		}

		if !more {
			break
		}
	}

	return nil
}
