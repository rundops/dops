package web

import (
	"io/fs"
	"net/http"
	"os"
	"path"
)

// SPAHandler returns an http.Handler that serves the embedded Vue SPA.
// Static files are served directly; all other paths fall back to index.html
// for client-side routing.
func SPAHandler() http.Handler {
	distSub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("embedded dist/ not found: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distSub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path and try to find the file.
		filePath := path.Clean(r.URL.Path)
		if filePath == "/" {
			filePath = "index.html"
		} else {
			filePath = filePath[1:] // strip leading /
		}

		// If the file exists in the embedded FS, serve it directly.
		if _, err := fs.Stat(distSub, filePath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found — SPA fallback: serve index.html.
		indexData, err := fs.ReadFile(distSub, "index.html")
		if os.IsNotExist(err) {
			// No index.html means frontend hasn't been built yet.
			http.Error(w, "Web UI not built. Run 'make web' first.", http.StatusServiceUnavailable)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexData)
	})
}
