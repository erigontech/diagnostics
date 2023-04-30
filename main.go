package main

import (
	"log"

	"github.com/ledgerwatch/diagnostics/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Printf("Failed to execute command due to error: %s", err.Error())
	}
}
