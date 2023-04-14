package cmd

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

func processCmdLineArgs(w http.ResponseWriter, templ *template.Template, success bool, result string) {
	var args CmdLineArgs
	if success {
		if strings.HasPrefix(result, successLine) {
			args.Args = strings.ReplaceAll(result[len(successLine):], "\n", " ")
		} else {
			args.Args = result
		}
		args.Success = true
	} else {
		args.Success = false
		args.Error = result
	}
	if err := templ.ExecuteTemplate(w, "cmd_line.html", args); err != nil {
		fmt.Fprintf(w, "Executing cmd_line template: %v", err)
	}
}
