#!/usr/bin/env bash
# seed.sh — Seeds 5 vault providers + demo user + agent for Token Vault Demo 04
# Usage: SHARK_URL=http://localhost:8000 ADMIN_KEY=demo-admin-key ./seed.sh
set -euo pipefail

SHARK_URL="${SHARK_URL:-http://localhost:8000}"
ADMIN_KEY="${ADMIN_KEY:-demo-admin-key}"
BASE_URL="http://localhost:3000"

auth() { echo "Authorization: Bearer $ADMIN_KEY"; }

echo "======================================================"
echo "  SharkAuth Token Vault Demo — Seeding"
echo "  SHARK_URL=$SHARK_URL"
echo "======================================================"

# ------------------------------------------------------------------
# 1. Wait for Shark to be ready
# ------------------------------------------------------------------
echo ""
echo "[1] Waiting for Shark..."
for i in $(seq 1 20); do
  if curl -sf "$SHARK_URL/health" > /dev/null 2>&1; then
    echo "    Shark is up."
    break
  fi
  sleep 1
done

# ------------------------------------------------------------------
# 2. Create demo user (user_42)
# ------------------------------------------------------------------
echo ""
echo "[2] Creating demo user user_42..."
USER_RESP=$(curl -sf -X POST "$SHARK_URL/api/v1/admin/users" \
  -H "$(auth)" \
  -H "Content-Type: application/json" \
  -d '{"email":"aisha-test@gemma.demo","name":"Aisha Test","external_id":"user_42"}' || echo "{}")
USER_ID=$(echo "$USER_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id','user_42'))" 2>/dev/null || echo "user_42")
echo "    user_id=$USER_ID"

# ------------------------------------------------------------------
# 3. Create agent (gemma-worker)
# ------------------------------------------------------------------
echo ""
echo "[3] Creating agent gemma-worker..."
AGENT_RESP=$(curl -sf -X POST "$SHARK_URL/api/v1/agents" \
  -H "$(auth)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gemma-worker",
    "description": "Gemma AI ops worker — reads Gmail/Slack/GitHub/Notion/Linear",
    "scopes": ["vault:read"],
    "client_secret": "demo-secret"
  }' || echo "{}")
echo "    agent=$(echo "$AGENT_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('client_id','gemma-worker'))" 2>/dev/null || echo "gemma-worker")"

# ------------------------------------------------------------------
# 4. Seed vault providers
# ------------------------------------------------------------------
echo ""
echo "[4] Seeding vault providers..."

seed_provider() {
  local TEMPLATE="$1"
  local CLIENT_ID="${2:-demo-client-id}"
  local CLIENT_SECRET="${3:-demo-client-secret}"
  local SCOPES="$4"

  echo "    Creating provider: $TEMPLATE"
  curl -sf -X POST "$SHARK_URL/api/v1/vault/providers" \
    -H "$(auth)" \
    -H "Content-Type: application/json" \
    -d "{
      \"template\": \"$TEMPLATE\",
      \"client_id\": \"$CLIENT_ID\",
      \"client_secret\": \"$CLIENT_SECRET\",
      \"scopes\": $SCOPES
    }" > /dev/null || echo "      [WARN] $TEMPLATE may already exist or template not found"
}

seed_provider "google_gmail" \
  "${GOOGLE_CLIENT_ID:-demo-google-id}" \
  "${GOOGLE_CLIENT_SECRET:-demo-google-secret}" \
  '["https://www.googleapis.com/auth/gmail.readonly","https://www.googleapis.com/auth/calendar"]'

seed_provider "slack" \
  "${SLACK_CLIENT_ID:-demo-slack-id}" \
  "${SLACK_CLIENT_SECRET:-demo-slack-secret}" \
  '["channels:read","chat:write"]'

seed_provider "github" \
  "${GITHUB_CLIENT_ID:-demo-github-id}" \
  "${GITHUB_CLIENT_SECRET:-demo-github-secret}" \
  '["repo","read:user"]'

seed_provider "notion" \
  "${NOTION_CLIENT_ID:-demo-notion-id}" \
  "${NOTION_CLIENT_SECRET:-demo-notion-secret}" \
  '[]'

seed_provider "linear" \
  "${LINEAR_CLIENT_ID:-demo-linear-id}" \
  "${LINEAR_CLIENT_SECRET:-demo-linear-secret}" \
  '["read","write"]'

# ------------------------------------------------------------------
# 5. Print connect URLs
# ------------------------------------------------------------------
echo ""
echo "[5] Connect URLs (open in browser or paste in demo):"
for PROVIDER in google_gmail slack github notion linear; do
  RETURN_TO="${BASE_URL}/connected?provider=${PROVIDER}"
  echo "    $PROVIDER:"
  echo "      $SHARK_URL/api/v1/vault/connect/$PROVIDER?redirect_uri=$RETURN_TO&user_id=$USER_ID"
done

# ------------------------------------------------------------------
# 6. List available templates
# ------------------------------------------------------------------
echo ""
echo "[6] Available vault templates in this binary:"
curl -sf "$SHARK_URL/api/v1/vault/templates" \
  -H "$(auth)" \
  | python3 -c "
import sys, json
data = json.load(sys.stdin)
templates = data.get('templates', data) if isinstance(data, dict) else data
for t in templates:
    name = t.get('name', t) if isinstance(t, dict) else t
    print(f'    - {name}')
" 2>/dev/null || echo "    [WARN] Could not list templates — check /api/v1/vault/templates endpoint"

echo ""
echo "======================================================"
echo "  Seed complete. Run the demo:"
echo "  1. python demos/token_vault/connect-flow/server.py"
echo "  2. Open http://localhost:3000 to connect providers"
echo "  3. python demos/token_vault/gemma-worker/agent.py"
echo "======================================================"
