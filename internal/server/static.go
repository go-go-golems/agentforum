package server

import (
	"io/fs"
	"net/http"
	"strings"
)

// spaFS is the embedded UI filesystem when the binary is built with the
// `embed` tag (see embed.go); nil in plain builds, which serve the API only.
var spaFS fs.FS

// mountSPA serves the web UI at "/" with an SPA fallback: client-side
// routes (/s/eng, /t/th_…) that match no file serve index.html so deep
// links survive a refresh. /v1 and /healthz are always routed first
// (more specific patterns win on the mux).
func (s *Server) mountSPA() {
	if spaFS == nil {
		return
	}
	s.router.Handle("/", spaHandler(spaFS))
}

func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(fsys, path); err != nil {
			// no such asset: serve the SPA shell so the client router
			// can resolve the path
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
