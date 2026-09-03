//go:build embed

// The `embed` build tag mounts the built web UI into the binary:
//
//	make build-web   # pnpm build + copy web/dist into embed/public
//	go build -tags embed ./cmd/agentforum
//
// Plain builds (no tag) serve the /v1 API only; the tag keeps the UI out of
// test binaries and CI runs that do not build the frontend.
package server

import (
	"embed"
	"io/fs"
)

//go:embed embed/public
var embedded embed.FS

func init() {
	sub, err := fs.Sub(embedded, "embed/public")
	if err != nil {
		panic(err)
	}
	spaFS = sub
}
