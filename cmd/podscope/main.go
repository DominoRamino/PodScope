package main

import (
	"os"

	"github.com/podscope/podscope/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
