package cmd

import (
	"github.com/radoondas/netatmobeat/beater"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
)

// Name of the beat (netatmobeat).
var Name = "netatmobeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmdWithSettings(beater.New, instance.Settings{Name: Name})
