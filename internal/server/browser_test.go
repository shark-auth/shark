package server

import (
	"runtime"
	"testing"
)

// TestOpenBrowserNoPanic verifies that OpenBrowser does not panic on the
// current platform's command resolution path. We use a deliberately
// invalid URL so no actual browser window is opened; the test just
// confirms the code path runs without panicking.
//
// Note: the child process may fail to start (e.g. xdg-open not installed
// in CI), so we only check that the function returns without panicking —
// the error value is intentionally ignored.
func TestOpenBrowserNoPanic(t *testing.T) {
	t.Logf("platform: %s", runtime.GOOS)
	// We do NOT assert err == nil because the launcher binary may not be
	// present in CI (e.g. xdg-open on headless Linux). The important
	// invariant is "no panic".
	_ = OpenBrowser("http://127.0.0.1:0/test-no-actual-open")
}

// TestOpenBrowserEmptyURL verifies that passing an empty URL does not panic.
func TestOpenBrowserEmptyURL(t *testing.T) {
	_ = OpenBrowser("")
}
