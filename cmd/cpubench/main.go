package main

import (
	"io"
	"os"

	"github.com/example/cpu-benchmark/internal/cli"
)

var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return cli.Run(args, stdout, stderr)
}
