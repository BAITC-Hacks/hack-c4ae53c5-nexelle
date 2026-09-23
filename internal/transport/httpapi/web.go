package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WithFrontend serves the built UI and API from the same Go process and origin.
func WithFrontend(api http.Handler, directory string) http.Handler {
	files := http.FileServer(http.Dir(directory))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if _, err := os.Stat(filepath.Join(directory, "index.html")); err != nil {
			http.Error(w, "Build the frontend first: cd web && pnpm install --frozen-lockfile && pnpm build", http.StatusServiceUnavailable)
			return
		}
		files.ServeHTTP(w, r)
	})
}
