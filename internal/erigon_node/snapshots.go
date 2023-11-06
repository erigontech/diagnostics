package erigon_node

import (
	"context"
	"encoding/json"
)

type DownloadStatistics map[string]interface{}

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
