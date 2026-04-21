#!/usr/bin/env bash
# Hello Agent — end-to-end smoke runner.
#
# Builds shark, starts a dev server, registers an agent via DCR,
# runs the Python demo, and cleans up. Exits 0 on full success.
#
# Deps: go, python3, curl, jq. Python deps: shark-auth (editable from sdk/python).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PORT="${SHARK_PORT:-8080}"
AUTH_URL="http://localhost:${PORT}"
TMPDIR="$(mktemp -d -t hello-agent.XXXXXX)"
PID_FILE="${TMPDIR}/shark.pid"
LOG_FILE="${TMPDIR}/shark.log"

cleanup() {
    local code=$?
    if [[ -f "$PID_FILE" ]]; then
        local pid
        pid="$(cat "$PID_FILE" 2>/dev/null || true)"
        if [[ -n "${pid:-}" ]]; then
            kill "$pid" 2>/dev/null || true
            # give it a moment, then hard-kill
            sleep 1
            kill -9 "$pid" 2>/dev/null || true
        fi
    fi
    rm -rf "$TMPDIR"
    exit $code
}
trap cleanup EXIT INT TERM

need() {
    command -v "$1" >/dev/null 2>&1 || { echo "ERR: missing dependency: $1" >&2; exit 2; }
}
need go
need python3
need curl
need jq

echo "[0/6] tmpdir=$TMPDIR"

echo "[1/6] Building bin/shark ..."
go build -o bin/shark ./cmd/shark

echo "[2/6] Starting shark serve --dev on :${PORT} ..."
(
    cd "$TMPDIR"
    # Absolute path so the server finds the binary regardless of cwd.
    "$ROOT/bin/shark" serve --dev >"$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"
)

# Wait for /healthz up to 30s.
echo "[3/6] Waiting for /healthz ..."
for i in $(seq 1 60); do
    if curl -sf "${AUTH_URL}/healthz" >/dev/null; then
        echo "      healthy after ${i} tries"
        break
    fi
    sleep 0.5
    if [[ $i -eq 60 ]]; then
        echo "ERR: /healthz never returned 200 within 30s" >&2
        echo "---- server log ----" >&2
        tail -60 "$LOG_FILE" >&2 || true
        exit 1
    fi
done

echo "[4/6] Registering agent via DCR (/oauth/register) ..."
DCR_BODY='{"client_name":"hello-agent","grant_types":["client_credentials"],"token_endpoint_auth_method":"client_secret_basic","scope":"openid"}'
DCR_RESP="$(curl -sS -X POST "${AUTH_URL}/oauth/register" \
    -H 'Content-Type: application/json' \
    -d "$DCR_BODY")"
CID="$(echo "$DCR_RESP" | jq -r .client_id)"
CSECRET="$(echo "$DCR_RESP" | jq -r .client_secret)"
if [[ -z "$CID" || "$CID" == "null" ]]; then
    echo "ERR: DCR did not return client_id: $DCR_RESP" >&2
    exit 1
fi
echo "      client_id=$CID"

echo "[5/6] Running examples/hello_agent.py ..."
if ! python3 "$ROOT/examples/hello_agent.py" \
    --auth "$AUTH_URL" \
    --client-id "$CID" \
    --client-secret "$CSECRET"; then
    echo "ERR: hello_agent.py failed" >&2
    echo "---- server log (tail) ----" >&2
    tail -40 "$LOG_FILE" >&2 || true
    exit 1
fi

echo "[6/6] Success. Cleaning up."
