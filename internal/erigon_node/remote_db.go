package erigon_node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Tables []Table

type Table struct {
	Name  string `json:"name"`
	Count uint64 `json:"count"`
	Size  uint64 `json:"size"`
}

type Results struct{}

func (c *NodeClient) Tables(ctx context.Context, db string) (Tables, error) {
	request, err := c.fetch(ctx, "dbs/"+db+"/tables", nil)

	if err != nil {
		return nil, err
	}

	var tables Tables

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(result, &tables); err != nil {
		return nil, err
	}

	return tables, nil
}

func (c *NodeClient) Table(ctx context.Context, db string, table string) (Results, error) {
	return Results{}, fmt.Errorf("TODO")
}

type RemoteDbReader interface {
	Init(ctx context.Context, db string, table string, initialKey []byte) error
	Next(ctx context.Context) ([]byte, []byte, error)
}

type results [][2][]byte

func (r *results) UnmarshalJSON(json []byte) (err error) {

	*r = nil

	result := [2][]byte{}

	i := 0
	ri := 0

	for ; i < len(json); i++ {
		if i == 0 {
			if json[i] != byte('{') {
				return fmt.Errorf(`unexpected json start '%c'`, json[i])
			}

			continue
		}

		if json[i] == byte('}') {
			break
		}

		switch json[i] {
		case byte('"'):
			length, value := tostr(json[i:])

			i += length

			if result[ri], err = base64.URLEncoding.DecodeString(value); err != nil {
				return err
			}

			switch ri {
			case 0:
				ri++
			case 1:
				*r = append(*r, result)
				result = [2][]byte{}
				ri = 0
			}
		case byte(':'), byte(','):
			continue
		default:
			return fmt.Errorf("unexpected char '%c' at %d", json[i], i)
		}
	}

	return nil
}

func tostr(json []byte) (pos int, str string) {
	// expects that the lead character is a '"'
	for i := 1; i < len(json); i++ {
		if json[i] == byte('"') {
			return i, string(json[1:i])
		}
	}
	return len(json), string(json[1:])
}

type RemoteCursor struct {
	nodeClient Client
	dbPath     string
	table      string
	results    results
}

func NewRemoteCursor(nodeClient Client) *RemoteCursor {
	rc := &RemoteCursor{nodeClient: nodeClient}

	return rc
}

func (rc *RemoteCursor) Init(ctx context.Context, db string, table string, initialKey []byte) error {
	dbPath, dbPathErr := rc.findFullDbPath(ctx, db)

	if dbPathErr != nil {
		return dbPathErr
	}

	rc.dbPath = dbPath
	rc.table = table
	fmt.Println("Remote Cursor", rc.dbPath, rc.table)

	if err := rc.nextTableChunk(ctx, initialKey); err != nil {
		return err
	}

	return nil
}

func (rc *RemoteCursor) findFullDbPath(ctx context.Context, db string) (string, error) {
	request, err := rc.nodeClient.fetch(ctx, "dbs", nil)

	if err != nil {
		return "", fmt.Errorf("unable to fetch database list: %s", err)
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return "", fmt.Errorf("unable to fetch database list: %s", err)
	}

	var lines []string

	err = json.Unmarshal(result, &lines)

	if err != nil {
		return "", err
	}

	var dbPath string

	for _, line := range lines {
		if line == db {
			dbPath = line
		}
	}

	if dbPath == "" {
		return "", fmt.Errorf("database %s not found in: %v", db, lines)
	}

	return dbPath, nil
}

func (rc *RemoteCursor) nextTableChunk(ctx context.Context, startKey []byte) error {
	request, err := rc.nodeClient.fetch(ctx, "dbs/"+rc.dbPath+"/tables/"+rc.table+"/"+base64.URLEncoding.EncodeToString(startKey)+"?limit=256", nil)

	if err != nil {
		return fmt.Errorf("reading %s table: %w", rc.table, err)
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return err
	}

	var results struct {
		Offset  int64   `json:"offset"`
		Limit   int64   `json:"limit"`
		Count   int64   `json:"count"`
		Results results `json:"results"`
	}

	err = json.Unmarshal(result, &results)

	if err != nil {
		return err
	}

	rc.results = results.Results
	return nil
}

func advance(key []byte) []byte {
	if len(key) == 0 {
		return []byte{0}
	}
	i := len(key) - 1
	for i >= 0 && key[i] == 0xff {
		i--
	}
	var key1 []byte
	if i < 0 {
		key1 = make([]byte, len(key)+1)
	} else {
		key1 = make([]byte, len(key))
	}
	copy(key1, key)
	if i >= 0 {
		key1[i]++
	}
	return key1
}

func (rc *RemoteCursor) Next(ctx context.Context) ([]byte, []byte, error) {
	if rc.dbPath == "" || rc.table == "" {
		return nil, nil, fmt.Errorf("cursor not initialized")
	}

	if len(rc.results) == 0 {
		return nil, nil, nil
	}
	result := rc.results[0]

	rc.results = rc.results[1:]

	if len(rc.results) == 0 {
		if e := rc.nextTableChunk(ctx, advance(result[0])); e != nil {
			return result[0], result[1], e
		}
	}
	return result[0], result[1], nil
}
