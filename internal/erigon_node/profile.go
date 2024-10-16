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
		return nil, fmt.Errorf("fetching profile: %w", err)
	}

	_, result, err := request.nextResult(ctx)

	if err != nil {
		return nil, fmt.Errorf("fetching profile content: %w", err)
	}

	var content ProfileContent

	if err := json.Unmarshal(result, &content); err != nil {
		return nil, fmt.Errorf("unmarshalling profile content: %w", err)
	}

	//result is a file content so I need to save it to file and return the file path
	tempFile, err := os.CreateTemp("", "profile-*.pprof")
	if err != nil {
		return nil, fmt.Errorf("creating temporary file: %w", err)
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
		return nil, fmt.Errorf("writing to temporary file: %w", err)
	}

	cmd := exec.Command("go", "tool", "pprof", "-dot", tempFile.Name()) //nolint:gosec
	svgOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generating SVG output: %w", err)
	}

	return svgOutput, nil
}
