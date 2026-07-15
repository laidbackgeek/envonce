package main

import (
	"os"

	"github.com/laidbackgeek/envonce/internal/cli"
)

// version is injected at build time by goreleaser ldflags (-X main.version=...);
// it defaults to "dev" (go install / go run).
var version = "dev"

func main() {
	cli.SetVersion(version)
	os.Exit(cli.Execute())
}
