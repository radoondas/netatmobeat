package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/radoondas/netatmobeat/beater"
)

func main() {
	err := beat.Run("netatmobeat", "", beater.New)
	if err != nil {
		os.Exit(1)
	}
}
