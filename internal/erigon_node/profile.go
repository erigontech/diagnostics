package erigon_node

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type ProfileContent struct {
	Chunk []byte `json:"chunk"`
}

func (c *NodeClient) FindProfile(ctx context.Context, profile string) ([]byte, error) {
	request, err := c.fetch(ctx, profile, nil)

	if err != nil {
		return nil, err
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, err
	}

	var content ProfileContent

	if err := json.Unmarshal(result, &content); err != nil {
		return nil, err
	}

	//result is a file content so I need to save it to file and return the file path
	tempFile, err := os.CreateTemp("", "profile-*.pprof")
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := tempFile.Close(); err != nil {
			fmt.Printf("Error closing temporary file: %v\n", err)
		}

		// Remove the file after closing it
		if err := os.Remove(tempFile.Name()); err != nil {
			fmt.Printf("Error removing temporary file: %v\n", err)
		}
	}()

	if _, err := tempFile.Write(content.Chunk); err != nil {
		return nil, err
	}

	cmd := exec.Command("go", "tool", "pprof", "-png", tempFile.Name())
	svgOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return svgOutput, nil
}
