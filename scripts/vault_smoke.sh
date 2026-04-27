#!/usr/bin/env bash
# vault_smoke.sh — real-token smoke harness for SharkAuth Vault providers.
#
# USAGE:
#   SHARK_BASE_URL=https://auth.example.com \
#   SHARK_ADMIN_KEY=sk_live_... \
#   VAULT_GMAIL_CLIENT_ID=xxx VAULT_GMAIL_CLIENT_SECRET=yyy \
#   bash scripts/vault_smoke.sh gmail
#
# REQUIRED ENV (all providers):
#   SHARK_BASE_URL    — base URL of your SharkAuth server (no trailing slash)
#   SHARK_ADMIN_KEY   — admin API key (sk_live_...)
#
# REQUIRED ENV per provider (VAULT_{PROVIDER^^}_CLIENT_ID / _CLIENT_SECRET):
#   gmail      → VAULT_GMAIL_CLIENT_ID, VAULT_GMAIL_CLIENT_SECRET
#   slack      → VAULT_SLACK_CLIENT_ID, VAULT_SLACK_CLIENT_SECRET
#   github     → VAULT_GITHUB_CLIENT_ID, VAULT_GITHUB_CLIENT_SECRET
#   notion     → VAULT_NOTION_CLIENT_ID, VAULT_NOTION_CLIENT_SECRET
#   microsoft  → VAULT_MICROSOFT_CLIENT_ID, VAULT_MICROSOFT_CLIENT_SECRET
#   linear     → VAULT_LINEAR_CLIENT_ID, VAULT_LINEAR_CLIENT_SECRET
#   jira       → VAULT_JIRA_CLIENT_ID, VAULT_JIRA_CLIENT_SECRET
#
# MANUAL STEPS:
#   1. Run this script — it will print an authorization URL.
#   2. Open the URL in a browser and complete the OAuth consent flow.
#   3. The script polls /vault/connections until the connection appears.
#   4. Once found, it decrypts the token and makes a live API call.
#
# EXIT CODES: 0 = pass, 1 = fail

set -euo pipefail

PROVIDER="${1:-}"
if [[ -z "$PROVIDER" ]]; then
  echo "ERROR: PROVIDER argument required. Usage: $0 <provider>" >&2
  echo "       Supported: gmail, slack, github, notion, microsoft, linear, jira" >&2
  exit 1
fi

BASE="${SHARK_BASE_URL:?SHARK_BASE_URL must be set}"
TOKEN="${SHARK_ADMIN_KEY:?SHARK_ADMIN_KEY must be set}"
PROVIDER_UPPER="${PROVIDER^^}"

CLIENT_ID_VAR="VAULT_${PROVIDER_UPPER}_CLIENT_ID"
CLIENT_SECRET_VAR="VAULT_${PROVIDER_UPPER}_CLIENT_SECRET"
CLIENT_ID="${!CLIENT_ID_VAR:?${CLIENT_ID_VAR} must be set}"
CLIENT_SECRET="${!CLIENT_SECRET_VAR:?${CLIENT_SECRET_VAR} must be set}"

AUTH_HEADER="Authorization: Bearer ${TOKEN}"
CONTENT_JSON="Content-Type: application/json"

log()  { echo "[vault_smoke] $*"; }
fail() { echo "[vault_smoke] FAIL: $*" >&2; exit 1; }
pass() { echo "[vault_smoke] PASS: $*"; }

# ---------------------------------------------------------------------------
# Step 1 — Register / upsert vault provider
# ---------------------------------------------------------------------------
log "Registering provider '${PROVIDER}'…"

PROVIDER_BODY=$(cat <<JSON
{
  "provider": "${PROVIDER}",
  "client_id": "${CLIENT_ID}",
  "client_secret": "${CLIENT_SECRET}"
}
JSON
)

PROVIDER_RESP=$(curl -s -w "\n%{http_code}" -X POST \
  -H "${AUTH_HEADER}" -H "${CONTENT_JSON}" \
  -d "${PROVIDER_BODY}" \
  "${BASE}/api/v1/vault/providers")

PROVIDER_STATUS=$(echo "${PROVIDER_RESP}" | tail -1)
PROVIDER_BODY_OUT=$(echo "${PROVIDER_RESP}" | head -n -1)

if [[ "$PROVIDER_STATUS" != "200" && "$PROVIDER_STATUS" != "201" && "$PROVIDER_STATUS" != "409" ]]; then
  fail "POST /api/v1/vault/providers → HTTP ${PROVIDER_STATUS}: ${PROVIDER_BODY_OUT}"
fi
log "Provider registered (HTTP ${PROVIDER_STATUS})."

PROVIDER_ID=$(echo "${PROVIDER_BODY_OUT}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
if [[ -z "$PROVIDER_ID" ]]; then
  PROVIDER_ID=$(echo "${PROVIDER_BODY_OUT}" | grep -o '"provider_id":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
fi
log "Provider ID: ${PROVIDER_ID:-<unknown — will poll by provider name>}"

# ---------------------------------------------------------------------------
# Step 2 — Print authorize URL for browser consent
# ---------------------------------------------------------------------------
log "Fetching authorize URL…"

AUTHZ_RESP=$(curl -s -w "\n%{http_code}" \
  -H "${AUTH_HEADER}" \
  "${BASE}/api/v1/vault/providers/${PROVIDER}/authorize")

AUTHZ_STATUS=$(echo "${AUTHZ_RESP}" | tail -1)
AUTHZ_BODY=$(echo "${AUTHZ_RESP}" | head -n -1)

if [[ "$AUTHZ_STATUS" != "200" ]]; then
  fail "GET /api/v1/vault/providers/${PROVIDER}/authorize → HTTP ${AUTHZ_STATUS}: ${AUTHZ_BODY}"
fi

AUTHORIZE_URL=$(echo "${AUTHZ_BODY}" | grep -o '"url":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
if [[ -z "$AUTHORIZE_URL" ]]; then
  AUTHORIZE_URL=$(echo "${AUTHZ_BODY}" | grep -o 'https://[^"]*' | head -1 || true)
fi

echo ""
echo "============================================================"
echo " MANUAL ACTION REQUIRED"
echo " Open this URL in your browser and complete the OAuth flow:"
echo ""
echo "  ${AUTHORIZE_URL}"
echo ""
echo " Press ENTER when done (or Ctrl+C to abort)."
echo "============================================================"
read -r _

# ---------------------------------------------------------------------------
# Step 3 — Poll for connection
# ---------------------------------------------------------------------------
log "Polling /vault/connections for provider '${PROVIDER}'…"

CONNECTION_ID=""
for i in $(seq 1 30); do
  CONN_RESP=$(curl -s -H "${AUTH_HEADER}" \
    "${BASE}/api/v1/vault/connections?provider=${PROVIDER}")

  CONNECTION_ID=$(echo "${CONN_RESP}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
  if [[ -n "$CONNECTION_ID" ]]; then
    log "Connection found: ${CONNECTION_ID} (attempt ${i}/30)"
    break
  fi
  log "Attempt ${i}/30 — no connection yet, sleeping 5s…"
  sleep 5
done

if [[ -z "$CONNECTION_ID" ]]; then
  fail "No connection appeared after 150s. Did the browser flow complete?"
fi

# ---------------------------------------------------------------------------
# Step 4 — Fetch + decrypt token, print first 20 chars
# ---------------------------------------------------------------------------
log "Fetching decrypted token for connection ${CONNECTION_ID}…"

TOKEN_RESP=$(curl -s -w "\n%{http_code}" -X POST \
  -H "${AUTH_HEADER}" -H "${CONTENT_JSON}" \
  -d '{}' \
  "${BASE}/api/v1/vault/connections/${CONNECTION_ID}/token")

TOKEN_STATUS=$(echo "${TOKEN_RESP}" | tail -1)
TOKEN_BODY=$(echo "${TOKEN_RESP}" | head -n -1)

if [[ "$TOKEN_STATUS" != "200" ]]; then
  fail "POST /vault/connections/${CONNECTION_ID}/token → HTTP ${TOKEN_STATUS}: ${TOKEN_BODY}"
fi

ACCESS_TOKEN=$(echo "${TOKEN_BODY}" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
if [[ -z "$ACCESS_TOKEN" ]]; then
  ACCESS_TOKEN=$(echo "${TOKEN_BODY}" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
fi

if [[ -z "$ACCESS_TOKEN" ]]; then
  fail "Could not extract access_token from response: ${TOKEN_BODY}"
fi

PREVIEW="${ACCESS_TOKEN:0:20}"
log "Token prefix (first 20 chars): ${PREVIEW}…"

if [[ ${#ACCESS_TOKEN} -lt 10 ]]; then
  fail "Token too short — likely invalid: '${ACCESS_TOKEN}'"
fi

# ---------------------------------------------------------------------------
# Step 5 — Real API test call per provider
# ---------------------------------------------------------------------------
log "Making live API call for provider '${PROVIDER}'…"

API_STATUS=0
case "$PROVIDER" in
  gmail)
    API_RESP=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      "https://gmail.googleapis.com/gmail/v1/users/me/profile")
    API_STATUS="$API_RESP"
    ;;
  slack)
    API_RESP=$(curl -s -w "\n%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      "https://slack.com/api/auth.test")
    API_STATUS=$(echo "${API_RESP}" | tail -1)
    SLACK_OK=$(echo "${API_RESP}" | head -n -1 | grep -o '"ok":true' || true)
    if [[ -z "$SLACK_OK" ]]; then
      fail "Slack auth.test returned ok!=true: $(echo "${API_RESP}" | head -n -1)"
    fi
    ;;
  github)
    API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      -H "Accept: application/vnd.github+json" \
      "https://api.github.com/user")
    ;;
  notion)
    API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      -H "Notion-Version: 2022-06-28" \
      "https://api.notion.com/v1/users/me")
    ;;
  microsoft)
    API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      "https://graph.microsoft.com/v1.0/me")
    ;;
  linear)
    API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      -H "Content-Type: application/json" \
      -d '{"query":"{viewer{id}}"}' \
      "https://api.linear.app/graphql")
    ;;
  jira)
    API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      "https://api.atlassian.com/oauth/token/accessible-resources")
    ;;
  *)
    fail "Unknown provider '${PROVIDER}'. Supported: gmail, slack, github, notion, microsoft, linear, jira"
    ;;
esac

if [[ "$API_STATUS" != "200" ]]; then
  fail "Live ${PROVIDER} API call returned HTTP ${API_STATUS} (expected 200)."
fi

pass "Live ${PROVIDER} API call succeeded (HTTP 200)."

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "=============================="
echo " vault_smoke SUMMARY"
echo " Provider:     ${PROVIDER}"
echo " Connection:   ${CONNECTION_ID}"
echo " Token prefix: ${PREVIEW}…"
echo " Live API:     HTTP 200 OK"
echo " Result:       PASS"
echo "=============================="
exit 0
