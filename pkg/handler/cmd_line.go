package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type CmdLineArgs struct {
	Success bool
	Error   string
	Args    string
}

func processCmdLineArgs(w http.ResponseWriter, tmpl *template.Template, success bool, result string) {
	var args CmdLineArgs
	if success {
		if strings.HasPrefix(result, successLine) {
			result = strings.ReplaceAll(result[len(successLine):], "\n", " ")
		}
		args.Args = result
		args.Success = true
	} else {
		args.Error = result
	}
	if err := tmpl.ExecuteTemplate(w, "cmd_line.html", args); err != nil {
		fmt.Fprintf(w, "Failed executing cmd_line template: %v\n", err)
	}
}
