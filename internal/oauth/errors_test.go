package oauth

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestWriteOAuthErrorShapeAndHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteOAuthError(rr, 400, NewOAuthError(ErrInvalidRequest, "missing client_id"))

	if rr.Code != 400 {
		t.Fatalf("status = %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content-type = %q", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("cache-control = %q (RFC 6749 §5.1)", got)
	}
	if got := rr.Header().Get("Pragma"); got != "no-cache" {
		t.Errorf("pragma = %q (RFC 6749 §5.1)", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "invalid_request" {
		t.Errorf("error = %v", body["error"])
	}
	if body["error_description"] != "missing client_id" {
		t.Errorf("error_description = %v", body["error_description"])
	}
	// error_uri must be omitted when unset — strict RFC compliance.
	if _, ok := body["error_uri"]; ok {
		t.Errorf("error_uri should be omitted, got %v", body["error_uri"])
	}
	// No Shark-extension fields leaked into the body.
	for _, banned := range []string{"message", "code", "docs_url", "details"} {
		if _, ok := body[banned]; ok {
			t.Errorf("non-RFC field %q leaked into oauth body", banned)
		}
	}
}

func TestOAuthErrorOmitsEmptyDescription(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteOAuthError(rr, 401, OAuthError{Error: ErrInvalidClient})
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if _, ok := body["error_description"]; ok {
		t.Errorf("error_description should be omitted when unset")
	}
}
