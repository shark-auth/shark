#!/usr/bin/env bash
# Demo 05 — Seed script.
# Starts SharkAuth (dev mode) + the mock protected resource + the
# vanilla bearer contrast server, then runs the defender agent to
# prime state for the attacker scripts.
#
# Prerequisites:
#   - Go binary built: make build  (or: go build ./cmd/sharkauth)
#   - Python deps: pip install shark-auth requests fastapi uvicorn PyJWT cryptography
#   - Port 8080 (SharkAuth), 9000 (protected resource), 9001 (vanilla bearer)
#
# Usage:
#   bash demos/dpop_defense/seed.sh

set -euo pipefail

DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DEMO_DIR/../.." && pwd)"

echo "======================================================"
echo "  Demo 05 — DPoP Attack/Defense Seed"
echo "======================================================"

# 1. Start SharkAuth in the background
echo "[1/4] Starting SharkAuth on :8080 ..."
"$REPO_ROOT/sharkauth" --dev --addr :8080 &
SHARK_PID=$!
trap "kill $SHARK_PID $RESOURCE_PID $VANILLA_PID 2>/dev/null || true" EXIT
sleep 2  # wait for startup

# 2. Start mock protected resource server (DPoP-enforcing)
echo "[2/4] Starting DPoP-enforcing resource server on :9000 ..."
uvicorn demos.dpop_defense.resource_server:app --port 9000 --app-dir "$REPO_ROOT" &
RESOURCE_PID=$!
sleep 1

# 3. Start vanilla bearer contrast server
echo "[3/4] Starting vanilla bearer server on :9001 ..."
uvicorn server:app --port 9001 --app-dir "$DEMO_DIR/vanilla-bearer-server" &
VANILLA_PID=$!
sleep 1

# 4. Run defender agent to prime state
echo "[4/4] Running defender agent (Lin's side) ..."
SHARK_URL=http://localhost:8080 \
API_URL=http://localhost:9000 \
python "$DEMO_DIR/defender/agent.py"

echo ""
echo "======================================================"
echo "  State saved to /tmp/shark_demo_state.json"
echo "  Now run attacker scripts:"
echo ""
echo "  python demos/dpop_defense/attacker/replay.py"
echo "  python demos/dpop_defense/attacker/forge.py"
echo "  python demos/dpop_defense/attacker/jti_replay.py"
echo "  python demos/dpop_defense/attacker/htu_mismatch.py"
echo "  python demos/dpop_defense/attacker/time_travel.py"
echo "  python demos/dpop_defense/attacker/refresh_steal.py"
echo ""
echo "  Then: python demos/dpop_defense/scoreboard.py"
echo "======================================================"
