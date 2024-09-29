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
		return nil, fmt.Errorf("Error fetching profile: %v", err)
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, fmt.Errorf("Error fetching profile content: %v", err)
	}

	var content ProfileContent

	if err := json.Unmarshal(result, &content); err != nil {
		return nil, fmt.Errorf("Error unmarshalling profile content: %v", err)
	}

	//result is a file content so I need to save it to file and return the file path
	tempFile, err := os.CreateTemp("", "profile-*.pprof")
	if err != nil {
		return nil, fmt.Errorf("Error creating temporary file: %v", err)
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
		return nil, fmt.Errorf("Error writing to temporary file: %v", err)
	}

	cmd := exec.Command("go", "tool", "pprof", "-dot", tempFile.Name())
	svgOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Error generating SVG output: %v", err)
	}

	return svgOutput, nil
}
