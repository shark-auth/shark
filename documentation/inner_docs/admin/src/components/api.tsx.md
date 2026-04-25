# api.tsx

**Path:** `admin/src/components/api.tsx`
**Type:** API client module
**LOC:** 82

## Purpose
HTTP client wrapper for SharkAuth admin API. Manages auth headers, error handling, 401 redirects.

## Exports
- `API` (object) — request methods
- `useAPI(path, deps?)` (hook) — fetch with caching

## API methods
- `API.get(path)` — GET request
- `API.post(path, body)` — POST request
- `API.patch(path, body)` — PATCH request
- `API.put(path, body)` — PUT request
- `API.del(path)` — DELETE request
- `API.request(method, path, body)` — generic request
- `API.postFormData(path, form)` — POST multipart form

## useAPI hook
- Signature: `useAPI(path, deps?)` → `{ data, loading, error, refresh, setData }`
- Auto-fetches on mount and path change
- Refresh function to manually re-fetch
- Set data to manually update cache

## Auth
- Bearer token from localStorage (`shark_admin_key`)
- 401 triggers cleanup and `shark-auth-expired` event

## Error handling
- Throws on network error
- Parses JSON error body, falls back to HTTP status text
- 204 No Content returns null

## Base URL
- All requests prefixed with `/api/v1`

## Notes
- `// @ts-nocheck` to skip type checking
- Handles both JSON and FormData request bodies
