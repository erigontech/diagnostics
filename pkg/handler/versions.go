package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Versions struct {
	Success        bool
	Error          string
	NodeVersion    uint64
	SupportVersion uint64
	CodeVersion    string
	GitCommit      string
}

func processVersions(w http.ResponseWriter, templ *template.Template, success bool, result string, skipUpdateHTML ...bool) Versions {
	var ver Versions
	ver.processResponse(result, success)

	if len(skipUpdateHTML) == 0 || !skipUpdateHTML[0] {
		if err := templ.ExecuteTemplate(w, "versions.html", ver); err != nil {
			fmt.Fprintf(w, "Failed executing versions template: %v", err)
		}
	}

	return ver
}

func (v *Versions) processResponse(result string, success bool) {
	if !success {
		v.Error = result
		return
	}

	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		v.Error = fmt.Sprintf("incorrect response (at least node version needs to be present): %v", lines)
		return
	}
	if !strings.HasPrefix(lines[0], successLine) {
		v.Error = fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", lines)
		return
	}

	nodeVer, err := strconv.ParseUint(lines[1], 10, 64)
	if err != nil {
		v.Error = fmt.Sprintf("failed parsing node version: %v", err)
		return
	}

	v.NodeVersion = nodeVer
	v.Success = true
	for i, l := range lines[2:] {
		switch i {
		case 0:
			v.CodeVersion = l
		case 1:
			v.GitCommit = l
		}
	}
	return
}
