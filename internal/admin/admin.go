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

// adminCSP relaxes the strict default CSP from mw.SecurityHeaders() for the
// admin dashboard route. Scoped narrowly:
//   - script-src: 'self' for vendored React/Babel + component JSX files;
//     'unsafe-inline' for the <script type="text/babel"> blocks in index.html;
//     'unsafe-eval' because Babel-standalone compiles JSX via eval.
//   - style-src: 'self' plus 'unsafe-inline' (the bundle uses inline styles)
//     and fonts.googleapis.com for the @font-face stylesheet.
//   - font-src: 'self' plus fonts.gstatic.com for the actual font binaries.
//   - img-src: 'self' plus data: for inline avatars.
//   - connect-src: 'self' — XHR/fetch stays same-origin to the Shark API.
//   - frame-ancestors: 'none' — preserve clickjacking protection.
const adminCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
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
		// route only. The admin bundle runs React + Babel-standalone client-side
		// (Babel needs `eval` to compile JSX) and loads Google Fonts. The rest
		// of the API keeps the strict `default-src 'self'` policy.
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
