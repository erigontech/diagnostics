package web

import (
	"net/http"

	"github.com/ledgerwatch/diagnostics/web/dist"
)

var UI = http.FileServer(http.FS(dist.FS))
