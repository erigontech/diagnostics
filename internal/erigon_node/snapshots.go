package erigon_node

import (
	"context"
	"encoding/json"
)

type DownloadStatistics struct {
	Progress         string `json:"progress"`
	Downloaded       string `json:"downloaded"`
	Total            string `json:"total"`
	TimeLeft         string `json:"timeLeft"`
	TotalTime        string `json:"totalTime"`
	DownloadRate     string `json:"downloadRate"`
	UploadRate       string `json:"uploadRate"`
	Peers            int32  `json:"peers"`
	Files            int32  `json:"files"`
	Connections      uint64 `json:"connections"`
	Alloc            string `json:"alloc"`
	Sys              string `json:"sys"`
	DownloadFinished bool   `json:"downloadFinished"`
	StagePrefix      string `json:"stagePrefix"`
}

// FindReorgs - Go through "Header" table and look for entries with the same block number but different hashes
func (c *NodeClient) ShanphotSync(ctx context.Context) (DownloadStatistics, error) {
	var syncStats DownloadStatistics

	request, err := c.fetch(ctx, "snapshot-sync", nil)

	if err != nil {
		return syncStats, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return DownloadStatistics{}, err
	}

	if err := json.Unmarshal(result, &syncStats); err != nil {
		return DownloadStatistics{}, err
	}

	return syncStats, nil
}
