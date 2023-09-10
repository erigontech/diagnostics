package erigon_node

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type RemoteDbReader interface {
	Init(ctx context.Context, db string, table string, initialKey []byte) error
	Next(ctx context.Context) ([]byte, []byte, error)
}

type RemoteCursor struct {
	nodeClient Client
	dbPath     string
	table      string
	lines      []string // Parsed response
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
	request, err := rc.nodeClient.fetch(ctx, "db_list", nil)

	if err != nil {
		return "", fmt.Errorf("unable to fetch database list: %s", err)
	}

	_, result, err := request.nextResult(ctx)

	var lines []string

	err = json.Unmarshal(result, lines)

	if err != nil {
		return "", err
	}
	// fmt.Println("lines: ", lines)

	var dbPath string

	for _, line := range lines {
		if strings.HasSuffix(line, fmt.Sprintf("/%s", db)) {
			dbPath = line
		}
	}

	if dbPath == "" {
		return "", fmt.Errorf("database %s not found in: %v", db, lines)
	}

	return dbPath, nil
}

func (rc *RemoteCursor) nextTableChunk(ctx context.Context, startKey []byte) error {
	request, err := rc.nodeClient.fetch(ctx, "dbs/"+rc.dbPath+"/tables/"+rc.table+"/"+string(startKey), nil)

	if err != nil {
		return fmt.Errorf("reading %s table: %w", rc.table, err)
	}

	_, result, err := request.nextResult(ctx)

	var lines []string

	err = json.Unmarshal(result, lines)

	if err != nil {
		return err
	}

	rc.lines = lines
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

	if len(rc.lines) == 0 {
		return nil, nil, nil
	}
	line := rc.lines[0]
	sepIndex := strings.Index(line, " | ")
	if sepIndex == -1 {
		return nil, nil, fmt.Errorf("could not find key-value separator | in the result line: %v", line)
	}
	var k, v []byte
	var e error
	if k, e = hex.DecodeString(line[:sepIndex]); e != nil {
		return nil, nil, fmt.Errorf("could not parse the key [%s]: %v", line[:sepIndex], e)
	}
	if v, e = hex.DecodeString(line[sepIndex+3:]); e != nil {
		return nil, nil, fmt.Errorf("could not parse the value [%s]: %v", line[sepIndex+3:], e)
	}
	rc.lines = rc.lines[1:]

	if len(rc.lines) == 0 {
		if e = rc.nextTableChunk(ctx, advance(k)); e != nil {
			return k, v, e
		}
	}
	return k, v, e
}
