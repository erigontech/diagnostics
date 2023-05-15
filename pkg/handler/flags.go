package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type Flags struct {
	Success bool
	Error   string
	Payload map[string]string
}

func processFlags(w http.ResponseWriter, tmpl *template.Template, success bool, result string, versions Versions) {

	if !versions.Success {
		fmt.Fprintf(w, "Unable to process flag due to inability to get node version: %s", versions.Error)
		return
	}
	if versions.NodeVersion < 2 {
		fmt.Fprintf(w, "Flags only support version >= 2. Node version: %d", versions.NodeVersion)
		return
	}

	var flags Flags
	flags.processResponse(result, success)
	if err := tmpl.ExecuteTemplate(w, "flags.html", flags); err != nil {
		fmt.Fprintf(w, "Failed executing flags template: %v", err)
		return
	}
}

func (f *Flags) processResponse(result string, success bool) {
	f.Payload = make(map[string]string)
	if !success {
		f.Error = result
		return
	}

	lines := strings.Split(result, "\n")
	if len(lines) <= 1 || !strings.HasPrefix(lines[0], successLine) {
		f.Error = fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", lines)
		return
	}

	f.Success = true
	for _, l := range lines[1:] {
		if len(l) == 0 {
			continue
		}

		name, val, found := strings.Cut(l, "=")
		if !found {
			f.Error = fmt.Sprintf("failed to parse line %s", l)
			f.Success = false
		} else {
			f.Payload[name] = val
		}
	}
}
