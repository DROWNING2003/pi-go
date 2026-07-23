package main

import (
	"os"

	"github.com/DROWNING2003/pi-go/packages/coding-agent/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
