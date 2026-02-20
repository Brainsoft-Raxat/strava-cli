package main

import "github.com/Brainsoft-Raxat/strava-cli/cmd"

// version is overridden at build time via:
//
//	go build -ldflags "-X main.version=<tag>"
//
// The Makefile does this automatically via `git describe --tags`.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
