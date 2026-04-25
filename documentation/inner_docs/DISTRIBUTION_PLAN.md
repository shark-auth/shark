# Distribution Plan — v0.9.0 launch

**Status:** plan, not yet executed.
**Decision date:** 2026-04-25.
**Owner:** raul.
**Dependencies:** SDK rename (worktree W-C) must land before npm publish; org slugs need confirming.

---

## Decisions you owe before executing

| # | Decision | Why blocking |
|---|---|---|
| 1 | GitHub org slug — `sharkauth` or other? | All goreleaser image refs (`ghcr.io/<org>/shark`) and binary download URLs |
| 2 | PyPI package name — `sharkauth` or `shark-auth`? | Python SDK publish + README install command |
| 3 | npm scope confirmed `@sharkauth` for both `node` and `react`? | After W-C lands |
| 4 | Initial tag — `v0.9.0` or `v0.9.0-rc.0` first then `v0.9.0`? | Goreleaser dry-run vs final |
| 5 | Container registry — GHCR only, or also Docker Hub? | Extra goreleaser block |

---

## Channels

| Channel | Artifact | Surface |
|---|---|---|
| GitHub Releases | `shark_<os>_<arch>` binaries (linux/darwin/windows × amd64/arm64) | `gh release` + `goreleaser` |
| GHCR | `ghcr.io/<org>/shark:v0.9.0`, `:latest` | goreleaser docker block |
| PyPI | `<py-name>` for python SDK | `twine upload` via GitHub Actions |
| npm | `@sharkauth/node`, `@sharkauth/react` | `pnpm publish` via GitHub Actions |
| Homebrew | `brew install <org>/shark/shark` (post-launch) | tap repo, deferred |

---

## Goreleaser plan

File: `.goreleaser.yml` at repo root.

Outline:

```yaml
project_name: shark
before:
  hooks:
    - go mod tidy
    - cd admin && pnpm install --frozen-lockfile && pnpm run build && cd ..
builds:
  - main: ./cmd/shark
    binary: shark
    env:
      - CGO_ENABLED=1   # required by mattn/go-sqlite3
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X github.com/sharkauth/sharkauth/internal/version.Version={{.Version}}
      - -X github.com/sharkauth/sharkauth/internal/version.Commit={{.ShortCommit}}
      - -X github.com/sharkauth/sharkauth/internal/version.Date={{.Date}}
archives:
  - name_template: "shark_{{.Os}}_{{.Arch}}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
dockers:
  - image_templates:
      - "ghcr.io/<org>/shark:{{ .Version }}"
      - "ghcr.io/<org>/shark:latest"
    dockerfile: Dockerfile.release
    extra_files: [config.json]
release:
  github:
    owner: <org>
    name: shark
  draft: false
  prerelease: auto
```

**CGO note:** sqlite3 driver requires CGO. Goreleaser cross-compile with CGO needs Zig or per-arch builders. Two options:
- **A:** Drop `mattn/go-sqlite3`, switch to `modernc.org/sqlite` (pure Go). One-line driver swap. Recommended.
- **B:** Use `goreleaser` with separate matrix builds per OS (slower, more YAML).

Recommend **A** — unblocks single-host goreleaser runs, faster CI.

**Workflow file:** `.github/workflows/release.yml`. Triggers on tag `v*`. Steps:
1. Setup Go 1.22 + pnpm + Node 20
2. Login to GHCR (`docker/login-action`)
3. Run `goreleaser release --clean`
4. Upload checksums to release

Estimated effort: 4-6 hrs (incl. modernc.org/sqlite swap, dry-run cycles).

---

## Python SDK (PyPI)

Path: `sdk/python/`.

Tasks:
1. Confirm package name (`sharkauth` preferred — matches `@sharkauth/*` scope).
2. Add `pyproject.toml` if missing — `setuptools` or `hatch` build backend.
3. Add `[project.urls]` linking GitHub repo.
4. Workflow: `.github/workflows/publish-python.yml` — triggers on tag `v*`. Builds wheel + sdist, uploads via `pypa/gh-action-pypi-publish` with trusted publishers (no API token in repo).
5. Set up trusted publisher in PyPI account (manual, one-time).

Effort: 2-3 hrs once PyPI trusted publisher configured.

---

## TypeScript SDK (npm)

Two packages:
- `sdk/typescript/` → `@sharkauth/node` (already correct name)
- `packages/sharkauth-react/` → `@sharkauth/react` (rename in worktree W-C)

Tasks (after W-C merges):
1. Set `"publishConfig": { "access": "public" }` in both `package.json`.
2. Bump version in both — coordinated via `pnpm changeset` or manual.
3. Workflow: `.github/workflows/publish-npm.yml` — triggers on tag `v*`. Steps:
   - `pnpm install --frozen-lockfile`
   - `pnpm -F @sharkauth/node build`
   - `pnpm -F @sharkauth/react build`
   - `pnpm -F @sharkauth/node publish --no-git-checks --access public`
   - `pnpm -F @sharkauth/react publish --no-git-checks --access public`
4. Need `NPM_TOKEN` repo secret with publish access to `@sharkauth` scope.
5. Verify scope ownership on npmjs.com (manual, one-time).

Effort: 2 hrs once `NPM_TOKEN` set.

---

## Version + tag rehearsal procedure

Before the real `v0.9.0`:

1. **Add VERSION file** at repo root with `0.9.0`.
2. **Add `internal/version/version.go`** exposing `Version`, `Commit`, `Date` vars (set by ldflags). Wire into `shark version` CLI command.
3. **Tag dry-run:** `git tag v0.9.0-rc.0 && git push origin v0.9.0-rc.0`. Verify:
   - Goreleaser builds 6 binaries cleanly
   - GHCR image lands at `ghcr.io/<org>/shark:0.9.0-rc.0`
   - PyPI workflow publishes `<py-name>==0.9.0rc0`
   - npm workflow publishes `@sharkauth/node@0.9.0-rc.0` and `@sharkauth/react@0.9.0-rc.0`
   - GitHub Release is created with checksums + 6 archives
4. **Smoke install** from each channel:
   - `curl -sSL https://github.com/<org>/shark/releases/latest/download/shark_linux_amd64 | tar xz && ./shark version`
   - `docker pull ghcr.io/<org>/shark:0.9.0-rc.0 && docker run --rm ghcr.io/<org>/shark:0.9.0-rc.0 version`
   - `pip install <py-name>==0.9.0rc0 && python -c "import sharkauth; print(sharkauth.__version__)"`
   - `npm install @sharkauth/node@0.9.0-rc.0 && node -e "const s=require('@sharkauth/node'); console.log(s.VERSION)"`
5. **If RC clean:** delete `v0.9.0-rc.0` tag locally + on origin (or leave as rc), create real `v0.9.0` tag for launch.

---

## Install one-liner script (`install.sh`)

Hosted at `https://shark.sh/install` or `https://raw.githubusercontent.com/<org>/shark/main/install.sh`.

```sh
#!/usr/bin/env sh
set -e
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac
URL="https://github.com/<org>/shark/releases/latest/download/shark_${OS}_${ARCH}.tar.gz"
TMP=$(mktemp -d)
curl -fsSL "$URL" | tar xz -C "$TMP"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
mv "$TMP/shark" "$INSTALL_DIR/shark"
chmod +x "$INSTALL_DIR/shark"
echo "shark installed to $INSTALL_DIR/shark"
shark version
```

Effort: 30 min once release artifacts exist.

---

## Sequence on launch day

1. Tag `v0.9.0` → push.
2. CI runs goreleaser → release published with 6 archives + checksums.
3. CI publishes PyPI + npm packages.
4. GHCR image latest pointer updated.
5. Verify install one-liner pulls v0.9.0.
6. README install section already says `curl -sSL https://shark.sh/install | sh` — works.
7. HN post.

---

## Out of scope for this doc

- Homebrew tap (post-launch).
- Windows installer (.msi) (post-launch).
- Linux distro packages (.deb/.rpm) (post-launch).
- Auto-updater in shark CLI (post-launch).
- Signed binaries / cosign attestations (post-launch — security hardening).
