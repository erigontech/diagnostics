package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type RemoteDbReader interface {
	Init(db string, table string, initialKey []byte) error
	Next() ([]byte, []byte, error)
}

type RemoteCursor struct {
	remoteApi      RemoteApiReader
	requestChannel chan *NodeRequest
	dbPath         string
	table          string
	lines          []string // Parsed response
}

func NewRemoteCursor(remoteApi RemoteApiReader, requestChannel chan *NodeRequest) *RemoteCursor {
	rc := &RemoteCursor{remoteApi: remoteApi, requestChannel: requestChannel}

	return rc
}

func (rc *RemoteCursor) Init(db string, table string, initialKey []byte) error {
	dbPath, dbPathErr := rc.findFullDbPath(db)

	if dbPathErr != nil {
		return dbPathErr
	}

	rc.dbPath = dbPath
	rc.table = table

	if err := rc.nextTableChunk(initialKey); err != nil {
		return err
	}

	return nil
}

func (rc *RemoteCursor) findFullDbPath(db string) (string, error) {
	success, dbListResponse := rc.remoteApi.fetch("/db/list\n", rc.requestChannel)
	if !success {
		return "", fmt.Errorf("unable to fetch database list: %s", dbListResponse)
	}

	lines, err := rc.remoteApi.getResultLines(dbListResponse)
	if err != nil {
		return "", err
	}

	var dbPath string
	for _, line := range lines {
		if strings.HasSuffix(line, fmt.Sprintf("/%s", db)) {
			dbPath = line
		}
	}

	if dbPath == "" {
		return "", fmt.Errorf("database %s not found: %v", db, dbListResponse)
	}

	return dbPath, nil
}

func (rc *RemoteCursor) nextTableChunk(startKey []byte) error {
	success, result := rc.remoteApi.fetch(fmt.Sprintf("/db/read?path=%s&table=%s&key=%x\n", rc.dbPath, rc.table, startKey), rc.requestChannel)
	if !success {
		return fmt.Errorf("reading %s table: %s", rc.table, result)
	}
	lines, err := rc.remoteApi.getResultLines(result)
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

func (rc *RemoteCursor) Next() ([]byte, []byte, error) {
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
		if e = rc.nextTableChunk(advance(k)); e != nil {
			return k, v, e
		}
	}
	return k, v, e
}
