// Package admin embeds the SharkAuth admin dashboard (Phase 4) and exposes
// an http.Handler that serves the bundle with SPA fallback to index.html.
package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

// adminCSP defines the Content-Security-Policy for the admin dashboard route.
// The dashboard is a Vite-bundled React SPA — no runtime compilation, so
// 'unsafe-eval' is not needed. 'unsafe-inline' stays for style-src because
// the bundle uses inline styles extensively.
const adminCSP = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'"

// Handler returns an http.Handler that serves the embedded admin dashboard.
// Unknown paths fall back to index.html so client-side routing works.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Override the strict default CSP set by mw.SecurityHeaders() for this
		// route only. The admin dashboard is a Vite-built React SPA that loads
		// Google Fonts. The rest of the API keeps the strict policy.
		w.Header().Set("Content-Security-Policy", adminCSP)

		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(sub, clean); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
