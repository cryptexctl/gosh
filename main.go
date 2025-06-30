// gosh - Go Shell
// POSIX-compatible shell implementation written from scratch in Go.
// Copyright (c) 2025 gosh project - 0BSD License

package main

import (
	"fmt"
	"os"
	"runtime"

	"gosh/internal/shell"
)

var (
	version   = "1.0.4"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	shell := shell.New()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("gosh %s (built %s, commit %s)\n", version, buildTime, gitCommit)
			fmt.Printf("Go version: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
			os.Exit(0)
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		}
	}

	if err := shell.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`gosh %s - Go Shell

Usage: gosh [options] [script] [args...]

Options:
  -c <cmd>      Execute command and exit
  -i            Interactive mode
  -l, --login   Login shell
  -s            Read from stdin
  --version     Show version
  --help        Show help
  --norc        Skip ~/.goshrc
  --noprofile   Skip profile files
  --posix       POSIX mode
  --debug       Debug mode

Examples:
  gosh                 # Interactive
  gosh -c "echo hi"    # Execute command  
  gosh script.sh       # Run script

`, version)
}
