package main

import (
	"os"

	"github.com/ul0gic/ctsnare/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
