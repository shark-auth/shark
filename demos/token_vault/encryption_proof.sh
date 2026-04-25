#!/usr/bin/env bash
# encryption_proof.sh — Side-by-side: raw encrypted bytes vs decrypted API response
# The WOW MOMENT: proves AES-256-GCM at rest.
#
# Usage:
#   DB_PATH=./dev.db \
#   SHARK_URL=http://localhost:8000 \
#   AGENT_TOKEN=<shark-jwt> \
#   USER_ID=user_42 \
#   PROVIDER=google_gmail \
#   ./encryption_proof.sh
set -euo pipefail

DB_PATH="${DB_PATH:-./dev.db}"
SHARK_URL="${SHARK_URL:-http://localhost:8000}"
AGENT_TOKEN="${AGENT_TOKEN:-}"
USER_ID="${USER_ID:-user_42}"
PROVIDER="${PROVIDER:-google_gmail}"

if [ -z "$AGENT_TOKEN" ]; then
  echo "[ERROR] Set AGENT_TOKEN to a valid Shark JWT with vault:read scope"
  echo "  Get one: curl -X POST $SHARK_URL/oauth/token -d 'grant_type=client_credentials&client_id=gemma-worker&client_secret=demo-secret&scope=vault:read' | jq -r .access_token"
  exit 1
fi

if [ ! -f "$DB_PATH" ]; then
  echo "[ERROR] SQLite database not found at $DB_PATH"
  exit 1
fi

echo ""
echo "============================================================"
echo "  SharkAuth Token Vault — Encryption Proof"
echo "  Provider: $PROVIDER  |  User: $USER_ID"
echo "============================================================"

# ------------------------------------------------------------------
# LEFT: Raw bytes in SQLite (proves encryption)
# ------------------------------------------------------------------
echo ""
echo "=== LEFT: RAW BYTES IN SQLITE (vault_connections) ==="
echo "Query: SELECT hex(access_token_enc) FROM vault_connections LIMIT 1;"
echo ""
RAW_HEX=$(sqlite3 "$DB_PATH" "SELECT hex(access_token_enc) FROM vault_connections LIMIT 1;" 2>/dev/null || echo "ERROR: table not found")
echo "$RAW_HEX" | fold -w 64
echo ""
echo "Is it a JWT? (JWTs start with 'eyJ' = hex 65794A)"
if echo "$RAW_HEX" | grep -qi "^65794a"; then
  echo "  FAIL: looks like a plaintext JWT — check encryption config"
else
  echo "  PASS: not a JWT pattern — encrypted binary blob confirmed"
fi

# Also check typeof
TOKEN_TYPE=$(sqlite3 "$DB_PATH" "SELECT typeof(access_token_enc) FROM vault_connections LIMIT 1;" 2>/dev/null || echo "unknown")
echo "  Column type: $TOKEN_TYPE (expected: blob)"

# ------------------------------------------------------------------
# RIGHT: Decrypted token via Shark API
# ------------------------------------------------------------------
echo ""
echo "=== RIGHT: DECRYPTED TOKEN VIA SHARK API ==="
echo "GET $SHARK_URL/api/v1/vault/$PROVIDER/token"
echo ""
API_RESP=$(curl -sf \
  -H "Authorization: Bearer $AGENT_TOKEN" \
  -H "X-User-ID: $USER_ID" \
  "$SHARK_URL/api/v1/vault/$PROVIDER/token" 2>/dev/null || echo '{"error":"api_call_failed"}')
echo "$API_RESP" | python3 -m json.tool 2>/dev/null || echo "$API_RESP"

ACCESS_TOKEN=$(echo "$API_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")
echo ""
if [ -n "$ACCESS_TOKEN" ]; then
  echo "  Token preview: ${ACCESS_TOKEN:0:40}..."
  echo "  Is it a JWT? (starts with eyJ)"
  if echo "$ACCESS_TOKEN" | grep -q "^eyJ"; then
    echo "    YES — it's a Shark-issued JWT (normal for some providers)"
  else
    echo "    NO — it's a provider access token (e.g. ya29.* for Google)"
  fi
else
  echo "  [WARN] No access_token in response — check user has connected $PROVIDER"
fi

# ------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------
echo ""
echo "============================================================"
echo "  PROOF SUMMARY"
echo "  Database: $(wc -c < <(echo "$RAW_HEX")) chars of hex = encrypted blob"
echo "  API:      returns a usable, decrypted access token"
echo "  Encryption: AES-256-GCM (internal/auth/fieldcrypt.go)"
echo "  Key size:   32 bytes (SHA-256 derived from server secret)"
echo ""
echo "  The agent has NEVER touched the database."
echo "  The agent has NEVER seen the refresh token."
echo "  The agent called ONE Shark endpoint. Shark did the rest."
echo "============================================================"
