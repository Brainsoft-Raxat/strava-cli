package main

import (
	"runtime/debug"

	"github.com/Brainsoft-Raxat/strava-cli/cmd"
)

// version is overridden at build time via:
//
//	go build -ldflags "-X main.version=<tag>"
//
// The Makefile and GoReleaser do this automatically.
// When installed via `go install`, the module version from BuildInfo is used instead.
var version = "dev"

func main() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok &&
			info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	cmd.SetVersion(version)
	cmd.Execute()
}
