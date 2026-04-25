# init.go

**Path:** `cmd/shark/cmd/init.go`
**Package:** `cmd`
**LOC:** 141
**Tests:** init_test.go

## Purpose
Implements `shark init` — interactive first-boot setup that asks for a base URL, generates a secret, and writes a minimal `sharkauth.yaml`.

## Key types / functions
- `initCmd` (var, line 22) — cobra command. Refuses to run if stdin is not a TTY; refuses to overwrite without `--force`.
- `initAnswers` (struct, line 58) — currently only `BaseURL`.
- `askQuestions` (func, line 62) — prompts via stdin/stdout.
- `printPostInitNotice` (func, line 78) — reminds operator that default email is `shark.email` testing tier.
- `prompt` (func, line 91) — line reader with default fallback.
- `renderYAML` (func, line 112) — renders the minimum config (server.secret, server.base_url, email.provider="shark").
- `randomHexN` (func, line 130) — crypto/rand → hex.

## Imports of note
- `github.com/mattn/go-isatty` — TTY detection.
- `crypto/rand`, `bufio`.

## Wired by / used by
- Registered in `cmd/shark/cmd/root.go:93`.
- Invoked: `shark init`.

## Notes
- File written with mode 0o600.
- Output flag `--out` defaults to `sharkauth.yaml`; `--force` overwrites existing.
