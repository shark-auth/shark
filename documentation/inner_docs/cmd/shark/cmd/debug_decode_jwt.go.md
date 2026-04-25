# debug_decode_jwt.go

**Path:** `cmd/shark/cmd/debug_decode_jwt.go`
**Package:** `cmd`

## Purpose
Implements `shark debug decode-jwt <token>` — decodes a JWT's header and payload locally (no server call, no signature verification). Useful for session debugging.

## Notes
- Local-only operation.
- `--json` emits `{header: {...}, payload: {...}}`.
