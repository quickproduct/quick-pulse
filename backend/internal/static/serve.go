// Package static serves the embedded SvelteKit frontend with SPA fallback.
package static

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:build
var staticFS embed.FS

// Handler returns an http.Handler that serves embedded frontend assets and
// falls back to index.html for client-side routes. API/WS/proxy paths are not
// this handler's concern (the router only mounts it on "/").
func Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "build")
	if err != nil {
		panic("static: locate embedded build: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := sub.Open(path); err != nil {
			// Unknown path → serve index.html for the SPA router.
			indexFile, err := sub.Open("index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusNotFound)
				return
			}
			defer indexFile.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, indexFile)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
