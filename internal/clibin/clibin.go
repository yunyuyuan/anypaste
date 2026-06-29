// Package clibin embeds the cross-compiled CLI binaries (copied to ./bin at
// build time) so the server can hand them out at /cli/ and stay a single
// self-contained binary — no sidecar cli-dist directory required.
package clibin

import (
	"embed"
	"io/fs"
)

// all: also embeds dotfiles. A placeholder bin/.gitkeep is committed so the
// module builds even without a CLI build; the Docker build and release workflow
// populate ./bin from build-cli.sh output before `go build`.
//
//go:embed all:bin
var embedded embed.FS

// FS returns the embedded CLI binaries (the contents of bin/). It is empty
// except for the placeholder until a build populates bin/.
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "bin")
}
