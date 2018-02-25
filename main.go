package main

import (
	"os"

	"github.com/radoondas/netatmobeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
