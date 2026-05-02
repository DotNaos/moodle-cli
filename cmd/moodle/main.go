package main

import (
	"os"

	"github.com/DotNaos/moodle-services/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
