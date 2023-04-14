package cmd

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

func processVersions(w http.ResponseWriter, templ *template.Template, success bool, result string) {
	var versions Versions
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], successLine) {
			versions.Success = true
			if len(lines) < 2 {
				versions.Error = fmt.Sprintf("at least node version needs to be present")
				versions.Success = false
			} else {
				var err error
				versions.NodeVersion, err = strconv.ParseUint(lines[1], 10, 64)
				if err != nil {
					versions.Error = fmt.Sprintf("parsing node version [%s]: %v", lines[1], err)
					versions.Success = false
				} else {
					for idx, line := range lines[2:] {
						switch idx {
						case 0:
							versions.CodeVersion = line
						case 1:
							versions.GitCommit = line
						}
					}
				}
			}
		} else {
			versions.Error = fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", lines)
		}
	} else {
		versions.Error = result
	}
	if err := templ.ExecuteTemplate(w, "versions.html", versions); err != nil {
		fmt.Fprintf(w, "Executing versions template: %v", err)
		return
	}
}
