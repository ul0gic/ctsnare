package main

import (
	"os"

	"github.com/ul0gic/cert-hunter/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
