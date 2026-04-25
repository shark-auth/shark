# admin.go

**Path:** `internal/admin/admin.go`  
**Package:** `admin`  
**LOC:** 61  
**Tests:** (none)

## Purpose
Admin dashboard SPA handler. Serves embedded Vite-bundled React dist/ with SPA fallback (unknown paths → index.html). Sets strict CSP with Google Fonts exemption.

## Key types / functions
- `DistFS()` (func, line 29) — returns embedded dist filesystem for sibling packages
- `Handler()` (func, line 33) — returns http.Handler that serves SPA with index.html fallback
- Custom CSP header — default-src 'self', strict script-src, fonts from googleapis.com

## Imports of note
- `embed` — //go:embed dist
- `io/fs` — filesystem operations
- `net/http` — HandlerFunc, FileServer

## Wired by
- `internal/api` mounts Handler() at /admin/*
- Phase 4 feature; dashboard built in separate React repo, bundled at build time

## Notes
- Embedded at build time via //go:embed dist directive
- CSP override: admin route gets Google Fonts exception (style-src 'unsafe-inline' https://fonts.googleapis.com)
- SPA routing: unknown paths serve index.html (client-side routing)
- Static assets (CSS, JS, images) served with cache-friendly filenames
- DistFS exported so internal/api can reach into dist/hosted/assets/ for other routes

