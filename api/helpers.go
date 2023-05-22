package api

import (
	"fmt"
	"github.com/ledgerwatch/diagnostics"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

func retrievePinFromURL(r *http.Request) (uint64, error) {
	parsedURL, err := url.Parse(r.URL.Path)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return 0, err
	}

	lastPathItem := path.Base(parsedURL.Path)
	pin, err := strconv.ParseUint(lastPathItem, 10, 64)
	if err != nil {
		log.Printf("Errir parsing session pin %s: %v\n", lastPathItem, err)
		return 0, err
	}

	return pin, nil
}

func retrieveSizeStrFrom(r *http.Request) (uint64, error) {
	sizeStr := r.Form.Get("size")
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return 0, diagnostics.AsBadRequestErr(errors.Errorf("Parsing size %s: %v", sizeStr, err))
	}

	var offset uint64
	if size > 16*1024 {
		offset = size - 16*1024
	}

	return offset, nil
}
