// Package server — OpenBrowser opens url in the system default browser.
// Cross-platform: xdg-open (Linux), open (macOS), rundll32 (Windows).
// Failure is non-fatal; callers should log and continue.
package server

// OpenBrowser is the exported wrapper around openBrowser.
// It attempts to open url in the system default browser using the platform's
// native launcher command. The browser process is started asynchronously;
// a non-nil error means the launcher itself failed to start.
func OpenBrowser(url string) error {
	return openBrowser(url)
}
