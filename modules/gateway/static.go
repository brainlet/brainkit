package gateway

import (
	"io/fs"
	"net/http"
	"strings"
)

// handleStatic serves files from route.Static at route.Path
// prefix. Trailing slashes are normalized; a request for the
// bare prefix falls back to index.html (the `<prefix>/` entry).
func (gw *Gateway) handleStatic(w http.ResponseWriter, r *http.Request, matched *route) {
	if matched.Static == nil {
		http.NotFound(w, r)
		return
	}
	prefix := matched.Path
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	// Strip the route prefix so the FS sees the asset path only.
	rel := strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(prefix, "/"))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		rel = "index.html"
	}

	// Reject escape attempts early — fs.FS already forbids ".." but
	// reject them up front so we don't rely on the underlying impl.
	if strings.Contains(rel, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Resolve directories to their index.html so `/app/` serves
	// `index.html` out of the FS.
	if info, err := fs.Stat(matched.Static, rel); err == nil && info.IsDir() {
		rel = strings.TrimSuffix(rel, "/") + "/index.html"
	}

	f, err := matched.Static.Open(rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// http.ServeContent gives us range requests + correct
	// Content-Type sniffing from the file name/body.
	rs, ok := f.(readSeeker)
	if !ok {
		// Fall back to reading in memory if the FS doesn't
		// support seeking (go:embed does; os.DirFS does).
		http.ServeFileFS(w, r, matched.Static, rel)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), rs)
}

type readSeeker interface {
	Read(p []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
}
