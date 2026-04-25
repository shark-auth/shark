package api

import (
	_ "embed"
	"net/http"
)

//go:embed docs/openapi.bundled.yaml
var openAPISpec []byte

// handleAPIDocs serves the Scalar API reference UI at /api/docs.
// The bundled OpenAPI spec is embedded at build time; no runtime file I/O.
func (s *Server) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(scalarHTML)) //nolint:errcheck
}

// handleAPISpec serves the raw bundled OpenAPI YAML at /api/docs/openapi.yaml.
func (s *Server) handleAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	w.Write(openAPISpec) //nolint:errcheck
}

// scalarHTML is the Scalar CDN shell. The spec is loaded from /api/docs/openapi.yaml
// so the page works without inlining the entire YAML into the HTML.
const scalarHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>SharkAuth API Reference</title>
  <link rel="icon" type="image/svg+xml" href="/assets/branding/favicon.svg" />
  <style>
    body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <script
    id="api-reference"
    data-url="/api/docs/openapi.yaml"
    data-configuration='{
      "theme": "default",
      "layout": "sidebar",
      "defaultHttpClient": {"targetKey": "shell", "clientKey": "curl"},
      "metaData": {"title": "SharkAuth API Reference"}
    }'
  ></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>
`
