// Package api — snippet handler (A8).
//
// handleAppSnippet renders a framework-specific "paste this into your app"
// code snippet bundle for a registered application. Returns the install
// command, provider setup, and a page-usage example with the real ClientID
// and auth base URL substituted in so copy-paste just works.
package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleAppSnippet serves GET /api/v1/admin/apps/{id}/snippet.
//
// Query param: ?framework=react (default). Any other framework returns 501
// with `framework_not_supported`. Unknown app id returns 404.
func (s *Server) handleAppSnippet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "react"
	}
	if framework != "react" {
		writeJSON(w, http.StatusNotImplemented, errPayload(
			"framework_not_supported",
			framework+" support coming soon. React available now.",
		))
		return
	}

	app, err := s.getAppByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "app not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	authURL := ""
	if s.Config != nil {
		authURL = s.Config.Server.BaseURL
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"framework": "react",
		"snippets": []map[string]string{
			{
				"label": "Install",
				"lang":  "bash",
				"code":  "npm install @sharkauth/react",
			},
			{
				"label": "Provider setup",
				"lang":  "tsx",
				"code": fmt.Sprintf(
					"import { SharkProvider } from '@sharkauth/react'\n\n"+
						"<SharkProvider publishableKey=%q authUrl=%q>\n"+
						"  <App/>\n"+
						"</SharkProvider>",
					app.ClientID, authURL,
				),
			},
			{
				"label": "Page usage",
				"lang":  "tsx",
				"code": "import { SignIn, UserButton, SignedIn, SignedOut } from '@sharkauth/react'\n\n" +
					"<SignedOut><SignIn/></SignedOut>\n" +
					"<SignedIn><UserButton/></SignedIn>",
			},
		},
	})
}
