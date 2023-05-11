package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type RemoteCursor struct {
	uih            *UiHandler
	requestChannel chan *NodeRequest
	dbPath         string
	table          string
	lines          []string // Parsed response
}

func NewRemoteCursor(uih *UiHandler, requestChannel chan *NodeRequest) *RemoteCursor {
	rc := &RemoteCursor{uih: uih, requestChannel: requestChannel}

	return rc
}

func (rc *RemoteCursor) init(db string, table string, initialKey []byte) (*RemoteCursor, error) {
	dbPath, dbPathErr := rc.findFullDbPath(db)

	if dbPathErr != nil {
		return nil, dbPathErr
	}

	rc.dbPath = dbPath
	rc.table = table

	if err := rc.nextTableChunk(initialKey); err != nil {
		return nil, err
	}

	return rc, nil
}

func (rc *RemoteCursor) findFullDbPath(db string) (string, error) {
	success, dbListResponse := rc.uih.fetch("/db/list\n", rc.requestChannel)
	if !success {
		return "", fmt.Errorf("fetching list of db paths: %s", dbListResponse)
	}

	lines := strings.Split(dbListResponse, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], successLine) {
		return "", fmt.Errorf("incorrect response (first line needs to be SUCCESS): %v", lines)
	}

	var dbPath string
	for _, line := range lines[1:] {
		if strings.HasSuffix(line, fmt.Sprintf("/%s", db)) {
			dbPath = line
		}
	}

	if dbPath == "" {
		return "", fmt.Errorf("db path %s not found: %v", db, lines)
	}

	return dbPath, nil
}

func (rc *RemoteCursor) nextTableChunk(startKey []byte) error {
	success, result := rc.uih.fetch(fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", rc.dbPath, rc.table, startKey), rc.requestChannel)
	if !success {
		return fmt.Errorf("reading %s table: %s", rc.table, result)
	}
	lines := strings.Split(result, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], successLine) {
		return fmt.Errorf("incorrect response (first line needs to be SUCCESS): %v", lines)
	}
	lines = lines[1:]
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
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

func (rc *RemoteCursor) Next() ([]byte, []byte, error) {
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
		if e = rc.nextTableChunk(advance(k)); e != nil {
			return k, v, e
		}
	}
	return k, v, e
}
