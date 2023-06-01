package api

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

func retrievePinFromURL(r *http.Request) (uint64, error) {
	parsedURL, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Println("Error parsing URL:", err)
		return 0, fmt.Errorf("Error parsing URL: %w", err)
	}

	lastPathItem := path.Base(parsedURL.Path)
	pin, err := strconv.ParseUint(lastPathItem, 10, 64)
	if err != nil {
		log.Printf("Error parsing session pin %s: %v\n", lastPathItem, err)
		return 0, fmt.Errorf("Error parsing session pin %s: %w", lastPathItem, err)
	}

	return pin, nil
}

func retrieveSizeStrFrom(r *http.Request) (uint64, error) {
	sizeStr := r.Form.Get("size")
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Parsing size %s: %w", sizeStr, err)
	}

	var offset uint64
	if size > 16*1024 {
		offset = size - 16*1024
	}

	return offset, nil
}
