package main

import (
	"os"

	"github.com/radoondas/netatmobeat/cmd"

	_ "github.com/radoondas/netatmobeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
