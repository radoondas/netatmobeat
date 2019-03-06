package cmd

import (
	"github.com/radoondas/netatmobeat/beater"

	cmd "github.com/elastic/beats/libbeat/cmd"
)

// Name of the beat (netatmobeat).
var Name = "netatmobeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, "", beater.New)
