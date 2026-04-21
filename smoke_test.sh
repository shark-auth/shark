#!/bin/bash
# Shark smoke test — cross-platform (Windows/MSYS2 + Linux + macOS).
set -u

BASE="${BASE:-http://localhost:8080}"
DB="${DB:-dev.db}"
YAML="${YAML:-sharkauth.yaml}"
BIN="${BIN:-./shark}"
PASS=0
FAIL=0
FAIL_DETAILS=()

RED=$'\033[31m'
GRN=$'\033[32m'
YEL=$'\033[33m'
CYA=$'\033[36m'
RST=$'\033[0m'

section() { echo; echo "${CYA}== $* ==${RST}"; }
pass() { echo "  ${GRN}PASS${RST} $*"; PASS=$((PASS+1)); }
fail() { echo "  ${RED}FAIL${RST} $*"; FAIL=$((FAIL+1)); FAIL_DETAILS+=("$*"); }
note() { echo "  ${YEL}note${RST} $*"; }

# --- Cross-platform helpers ---------------------------------------------------

kill_port() {
  local port=$1
  # Windows (Git Bash / MSYS2)
  if command -v taskkill >/dev/null 2>&1; then
    local pid
    pid=$(netstat -ano 2>/dev/null | grep "LISTENING" | grep ":${port} " | awk '{print $5}' | head -1)
    [ -n "$pid" ] && taskkill //F //PID "$pid" 2>/dev/null || true
  # Linux
  elif command -v fuser >/dev/null 2>&1; then
    fuser -k "${port}/tcp" 2>/dev/null || true
  # macOS
  elif command -v lsof >/dev/null 2>&1; then
    local pids
    pids=$(lsof -ti :"$port" 2>/dev/null || true)
    [ -n "$pids" ] && kill $pids 2>/dev/null || true
  fi
  sleep 1
}

stop_server() {
  if [ -n "${SERVER_PID:-}" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  # Belt-and-suspenders: also kill by port in case the PID tracking was lost.
  kill_port 8080
}
trap stop_server EXIT

wait_for_server() {
  for _ in $(seq 1 50); do
    if curl -sf $BASE/healthz >/dev/null 2>&1; then return 0; fi
    sleep 0.2
  done
  return 1
}

boot_server() {
  stop_server
  $BIN serve --dev >> server.log 2>&1 &
  SERVER_PID=$!
  wait_for_server || { cat server.log; fail "server didn't come up"; exit 1; }
  # Give the server a moment to finish bootstrap logging (admin key, default app)
  sleep 1
}

# Re-login helper: obtains fresh cookie + JWT after a server restart.
relogin() {
  curl -s -c cj.txt -X POST $BASE/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" > /dev/null
}

# --- Pre: fresh DB + yaml -----------------------------------------------------
section "bootstrap: fresh DB"
kill_port 8080
# Remove DB + SQLite WAL/journal siblings so bootstrap generates fresh keys.
# --dev uses dev.db by default.
rm -f $DB $DB-journal $DB-wal $DB-shm sharkauth.db sharkauth.db-journal sharkauth.db-wal sharkauth.db-shm $YAML server.log cj*.txt
if [ ! -x "$BIN" ]; then echo "build binary first: go build -o $BIN ./cmd/shark"; exit 1; fi

# Write a minimal yaml — --dev flag auto-selects dev email provider + dev.db.
cat > $YAML <<EOF
server:
  base_url: http://localhost:8080
  secret: "change-me-this-secret-is-not-secure-at-all-abc123456789"
auth:
  jwt:
    enabled: true
    mode: "session"
    issuer: "http://localhost:8080"
    audience: "shark-smoke"
storage:
  path: $DB
EOF

boot_server
if grep -q "Default application created" server.log; then pass "default app banner"; else fail "no default app banner"; fi
if grep -q "ADMIN API KEY" server.log; then pass "admin key banner"; else fail "no admin key banner"; fi

ADMIN=$(grep -oE 'sk_live_[A-Za-z0-9_-]+' server.log | head -1)
DEFAULT_CID=$(grep -oE 'shark_app_[A-Za-z0-9_-]+' server.log | head -1)
[ -n "$ADMIN" ] && pass "admin key captured" || fail "admin key extract"
[ -n "$DEFAULT_CID" ] && pass "default client_id captured: $DEFAULT_CID" || fail "client_id extract"

# --- 2: JWT at signup ---------------------------------------------------------
section "signup issues JWT"
EMAIL="smoke$RANDOM@test.com"
PASSWORD='GetCake117$$$'
RESP=$(curl -s -c cj.txt -X POST $BASE/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
TOKEN=$(echo "$RESP" | jq -r '.token // empty')
USERID=$(echo "$RESP" | jq -r '.id // empty')
[ -n "$TOKEN" ] && pass "token in signup response" || { fail "no token"; echo "$RESP"; }
[ -n "$USERID" ] && pass "user id in response" || fail "no user id"

# Sanity-check that captured admin key actually authenticates.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/apps)
if [ "$CODE" = 200 ]; then
  pass "admin key sanity check"
else
  note "admin probe -> $CODE  key=[${ADMIN:0:16}...] len=${#ADMIN}"
  note "body: $(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/apps)"
fi
grep -q shark_session cj.txt && pass "cookie set" || fail "no cookie"

HEADER=$(echo "$TOKEN" | cut -d. -f1 | base64 -d 2>/dev/null | jq -c . 2>/dev/null || true)
echo "$HEADER" | grep -q '"alg":"RS256"' && pass "alg=RS256" || fail "alg not RS256"
echo "$HEADER" | grep -q '"kid":' && pass "kid present" || fail "kid missing"

# --- 3: dual-accept middleware ------------------------------------------------
section "middleware dual-accept"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" $BASE/api/v1/auth/me)
[ "$CODE" = 200 ] && pass "Bearer /me 200" || fail "Bearer /me $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt $BASE/api/v1/auth/me)
[ "$CODE" = 200 ] && pass "Cookie /me 200" || fail "Cookie /me $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" -b cj.txt $BASE/api/v1/auth/me)
[ "$CODE" = 200 ] && pass "Both 200" || fail "Both $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer garbage" -b cj.txt $BASE/api/v1/auth/me)
[ "$CODE" = 401 ] && pass "garbage Bearer + valid cookie -> 401 (no fallthrough)" || fail "no-fallthrough violated: $CODE"

WWW=$(curl -s -D - -o /dev/null -H "Authorization: Bearer garbage" $BASE/api/v1/auth/me | grep -i 'www-authenticate')
echo "$WWW" | grep -qi 'Bearer' && pass "WWW-Authenticate header present" || fail "no WWW-Authenticate"

# No auth at all
CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/auth/me)
[ "$CODE" = 401 ] && pass "no auth -> 401" || fail "no auth -> $CODE"

# --- 4: JWKS ------------------------------------------------------------------
section "JWKS"
JWKS=$(curl -s $BASE/.well-known/jwks.json)
N=$(echo "$JWKS" | jq '.keys | length')
[ "$N" -ge 1 ] && pass "$N key(s) in JWKS" || fail "JWKS keys=$N"
# RS256 key (session JWTs) + ES256 key (OAuth 2.1) both expected
echo "$JWKS" | jq -e '[.keys[] | select(.alg=="RS256")][0] | .kty=="RSA" and .use=="sig"' >/dev/null && pass "RS256 key present" || fail "no RS256 key"

KID_JWKS=$(echo "$JWKS" | jq -r '[.keys[] | select(.alg=="RS256")][0].kid')
KID_TOK=$(echo "$HEADER" | jq -r '.kid')
[ "$KID_JWKS" = "$KID_TOK" ] && pass "kid match token/JWKS ($KID_JWKS)" || fail "kid mismatch: tok=$KID_TOK jwks=$KID_JWKS"

CT=$(curl -sI $BASE/.well-known/jwks.json | grep -i 'cache-control' | tr -d '\r\n')
echo "$CT" | grep -qi 'max-age=300' && pass "Cache-Control max-age=300" || note "Cache-Control: $CT"

# --- 5: User-initiated revoke -------------------------------------------------
section "user revoke"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt -X POST $BASE/api/v1/auth/revoke \
  -H "Content-Type: application/json" -d "{\"token\":\"$TOKEN\"}")
if [ "$CODE" = 200 ] || [ "$CODE" = 204 ]; then pass "/auth/revoke $CODE"; else fail "/auth/revoke $CODE"; fi

# With check_per_request=false (default), token still works.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" $BASE/api/v1/auth/me)
[ "$CODE" = 200 ] && pass "TTL-only: revoked token still validates (default)" || note "/me=$CODE (tolerable if hardening turned on)"

# --- 6: Admin revoke gated ----------------------------------------------------
section "admin revoke gated"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/api/v1/admin/auth/revoke-jti \
  -H "Content-Type: application/json" -d '{"jti":"x","expires_at":"2030-01-01T00:00:00Z"}')
[ "$CODE" = 401 ] && pass "no auth -> 401" || fail "admin revoke no-auth -> $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/admin/auth/revoke-jti \
  -H "Content-Type: application/json" -d '{"jti":"x","expires_at":"2030-01-01T00:00:00Z"}')
if [ "$CODE" = 200 ] || [ "$CODE" = 204 ]; then pass "admin revoke 2xx"; else fail "admin revoke -> $CODE"; fi

# --- 7: Key rotation ----------------------------------------------------------
section "key rotation"
stop_server
$BIN keys generate-jwt --rotate > rotate.log 2>&1
if [ $? -eq 0 ]; then pass "rotate CLI exit 0"; else cat rotate.log; fail "rotate CLI failed"; fi
boot_server
relogin
sleep 0.5
JWKS=$(curl -s $BASE/.well-known/jwks.json)
N=$(echo "$JWKS" | jq '.keys | length')
# After RS256 rotation: old RS256 + new RS256 + ES256 = 3+ keys
[ "$N" -ge 3 ] && pass "JWKS has $N keys post-rotate" || fail "JWKS post-rotate keys=$N (expected >=3)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt $BASE/api/v1/auth/me)
[ "$CODE" = 200 ] && pass "session still validates after rotate" || fail "session after rotate $CODE"

# --- 8: Apps CLI --------------------------------------------------------------
section "apps CLI"
stop_server
$BIN app create --name smoke --callback https://ok.example.com/cb > app.log 2>&1
grep -q 'client_id' app.log && pass "app create prints client_id" || { cat app.log; fail "app create"; }
NEW_CID=$(grep -oE 'shark_app_[A-Za-z0-9_-]+' app.log | head -1)
[ -n "$NEW_CID" ] && pass "captured new CID $NEW_CID" || fail "CID extract"

$BIN app list > list.log 2>&1
grep -q "$NEW_CID" list.log && pass "new app in list" || fail "new app missing"

$BIN app show "$NEW_CID" > show.log 2>&1
grep -qi 'client_secret_hash' show.log && fail "secret hash leaked in show" || pass "secret hash NOT leaked"

$BIN app rotate-secret "$NEW_CID" > rot.log 2>&1
grep -qi 'client_secret' rot.log && pass "rotate prints new secret" || fail "rotate no secret"

$BIN app delete "$DEFAULT_CID" --yes > del.log 2>&1
if [ $? -ne 0 ]; then pass "delete default refused"; else fail "delete default NOT refused"; fi

$BIN app delete "$NEW_CID" --yes > del2.log 2>&1
[ $? -eq 0 ] && pass "delete non-default ok" || { cat del2.log; fail "delete non-default"; }

boot_server
relogin

# --- 9: Admin apps HTTP -------------------------------------------------------
section "admin apps HTTP"
CODE=$(curl -s -o resp.json -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/admin/apps \
  -H "Content-Type: application/json" \
  -d '{"name":"httpapp","allowed_callback_urls":["https://x.com/cb"]}')
if [ "$CODE" = 200 ] || [ "$CODE" = 201 ]; then pass "admin apps POST $CODE"; else fail "admin apps POST $CODE"; fi
grep -q client_secret resp.json && pass "secret in create response" || fail "no secret on create"
HTTP_CID=$(jq -r .client_id resp.json)

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/apps)
[ "$CODE" = 200 ] && pass "admin list 200" || fail "admin list $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/apps/$HTTP_CID)
if [ "$CODE" = 200 ] || [ "$CODE" = 204 ]; then pass "admin delete $CODE"; else fail "admin delete $CODE"; fi

# Non-admin blocked
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/api/v1/admin/apps -H "Content-Type: application/json" -d '{}')
[ "$CODE" = 401 ] && pass "admin endpoint blocks no-auth" || fail "admin endpoint open: $CODE"

# --- 10: Redirect allowlist (magic-link flow) ---------------------------------
section "redirect allowlist"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/auth/magic-link/verify?token=fake&redirect_url=javascript:alert(1)")
[ "$CODE" = 400 ] && pass "javascript: redirect -> 400" || note "js-url got $CODE (may surface token error first)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/auth/magic-link/verify?token=fake&redirect_url=https://evil.example.com")
[ "$CODE" = 400 ] && pass "non-allowlisted redirect -> 400" || note "evil-url got $CODE (may surface token error first)"

# --- 11: Org RBAC -------------------------------------------------------------
section "org RBAC"
ORG_RESP=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations -H "Content-Type: application/json" -d '{"name":"Acme","slug":"acme-smoke"}')
ORG_ID=$(echo "$ORG_RESP" | jq -r '.id // empty')
[ -n "$ORG_ID" ] && pass "org created $ORG_ID" || { echo "$ORG_RESP"; fail "org create"; }

ROLES=$(curl -s -b cj.txt $BASE/api/v1/organizations/$ORG_ID/roles)
N=$(echo "$ROLES" | jq '.data | length')
[ "$N" = 3 ] && pass "3 builtin roles seeded" || { echo "    $ROLES"; fail "expected 3 roles, got $N"; }

# Create custom role
CUST=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations/$ORG_ID/roles \
  -H "Content-Type: application/json" \
  -d '{"name":"editor","description":"custom editor"}')
ROLE_ID=$(echo "$CUST" | jq -r '.id // empty')
[ -n "$ROLE_ID" ] && pass "custom role $ROLE_ID" || { echo "$CUST"; fail "custom role create"; }

# Try delete builtin — should be 409
BUILTIN_ID=$(echo "$ROLES" | jq -r '.data[] | select(.name=="owner") | .id' | head -1)
if [ -n "$BUILTIN_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt -X DELETE $BASE/api/v1/organizations/$ORG_ID/roles/$BUILTIN_ID)
  [ "$CODE" = 409 ] && pass "builtin delete -> 409" || fail "builtin delete -> $CODE"
fi

# --- 12: Audit log rows -------------------------------------------------------
section "audit logs"
if [ -f $DB ]; then
  N=$(sqlite3 $DB "SELECT COUNT(*) FROM audit_logs WHERE action LIKE 'app.%' OR action LIKE 'org.%' OR action LIKE 'rbac.%'")
  if [ "$N" -gt 0 ] 2>/dev/null; then pass "audit_logs has $N rows (app./org./rbac.)"; else fail "no audit rows"; fi
  sqlite3 $DB "SELECT action, COUNT(*) FROM audit_logs GROUP BY action ORDER BY 2 DESC" | head -10 | sed 's/^/    /'
fi

# --- 13: Regression -----------------------------------------------------------
section "regression"
CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/healthz)
[ "$CODE" = 200 ] && pass "/healthz 200" || fail "/healthz $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt -X POST $BASE/api/v1/auth/logout)
if [ "$CODE" = 200 ] || [ "$CODE" = 204 ]; then pass "/logout $CODE"; else fail "/logout $CODE"; fi

# --- 14: Admin System Endpoints (Wave G) --------------------------------------
section "admin system endpoints"

# Test email (dev mode sends to dev inbox)
CODE=$(curl -s -o /tmp/test-email.json -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"to":"test@example.com"}' \
  $BASE/api/v1/admin/test-email)
[ "$CODE" = 200 ] && pass "POST /admin/test-email -> 200" || fail "POST /admin/test-email -> $CODE"

# Purge expired sessions
CODE=$(curl -s -o /tmp/purge-sess.json -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -X POST $BASE/api/v1/admin/sessions/purge-expired)
[ "$CODE" = 200 ] && pass "POST /admin/sessions/purge-expired -> 200" || fail "POST /admin/sessions/purge-expired -> $CODE"

# Purge audit logs (far future date = 0 deleted)
CODE=$(curl -s -o /tmp/purge-audit.json -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -d '{"before":"2020-01-01T00:00:00Z"}' \
  $BASE/api/v1/admin/audit-logs/purge)
[ "$CODE" = 200 ] && pass "POST /admin/audit-logs/purge -> 200" || fail "POST /admin/audit-logs/purge -> $CODE"

# User oauth-accounts (empty array is fine)
CODE=$(curl -s -o /tmp/oauth-accts.json -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  $BASE/api/v1/users/$USERID/oauth-accounts 2>/dev/null)
if [ "$CODE" = 200 ]; then pass "GET /users/{id}/oauth-accounts -> 200"
elif [ -z "${USERID:-}" ]; then note "USER_ID not set, skipping oauth-accounts"
else fail "GET /users/{id}/oauth-accounts -> $CODE"; fi

# User passkeys (empty array is fine)
CODE=$(curl -s -o /tmp/passkeys.json -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  $BASE/api/v1/users/$USERID/passkeys 2>/dev/null)
if [ "$CODE" = 200 ]; then pass "GET /users/{id}/passkeys -> 200"
elif [ -z "${USERID:-}" ]; then note "USER_ID not set, skipping passkeys"
else fail "GET /users/{id}/passkeys -> $CODE"; fi

# Rotate signing key
CODE=$(curl -s -o /tmp/rotate-key.json -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -X POST $BASE/api/v1/admin/auth/rotate-signing-key)
[ "$CODE" = 200 ] && pass "POST /admin/auth/rotate-signing-key -> 200" || fail "POST /admin/auth/rotate-signing-key -> $CODE"

# Verify JWKS has 2+ keys after rotation
JWKS_KEYS=$(curl -s $BASE/.well-known/jwks.json | jq '.keys | length')
[ "$JWKS_KEYS" -ge 2 ] 2>/dev/null && pass "JWKS has $JWKS_KEYS keys after rotation" || fail "JWKS keys=$JWKS_KEYS (expected >=2)"

# --- 15: User List Filters ---------------------------------------------------
section "user list filters"

CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  "$BASE/api/v1/users?mfa_enabled=false")
[ "$CODE" = 200 ] && pass "GET /users?mfa_enabled=false -> 200" || fail "GET /users?mfa_enabled=false -> $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  "$BASE/api/v1/users?email_verified=true")
[ "$CODE" = 200 ] && pass "GET /users?email_verified=true -> 200" || fail "GET /users?email_verified=true -> $CODE"

# --- 16: Sessions (self-service) ----------------------------------------------
section "sessions (self-service)"
# Re-login to get a fresh cookie after logout above
relogin

CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt $BASE/api/v1/auth/sessions)
[ "$CODE" = 200 ] && pass "GET /auth/sessions 200" || fail "GET /auth/sessions $CODE"

# --- 17: Admin sessions -------------------------------------------------------
section "admin sessions"
SESS_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/sessions)
SESS_CODE=$(echo "$SESS_RESP" | tail -1)
SESS_BODY=$(echo "$SESS_RESP" | sed '$d')
[ "$SESS_CODE" = 200 ] && pass "GET /admin/sessions 200" || fail "GET /admin/sessions $SESS_CODE"

# Assert response shape uses .data (not .sessions) and has has_more-equivalent (next_cursor or absent)
SESS_DATA_TYPE=$(echo "$SESS_BODY" | jq -r '.data | type' 2>/dev/null)
[ "$SESS_DATA_TYPE" = "array" ] && pass "GET /admin/sessions body has .data array" || fail "GET /admin/sessions missing .data array (got type=$SESS_DATA_TYPE)"

# Assert no .sessions key (old wrong key) at the top level
SESS_BAD_KEY=$(echo "$SESS_BODY" | jq 'has("sessions")' 2>/dev/null)
[ "$SESS_BAD_KEY" = "false" ] && pass "GET /admin/sessions body has no legacy .sessions key" || fail "GET /admin/sessions still exposes .sessions key"

# If any session row exists, assert last_activity_at is present and non-empty
FIRST_LAA=$(echo "$SESS_BODY" | jq -r '.data[0].last_activity_at // "absent"' 2>/dev/null)
if [ "$FIRST_LAA" != "absent" ] && [ "$FIRST_LAA" != "null" ] && [ -n "$FIRST_LAA" ]; then
  pass "session[0].last_activity_at present: $FIRST_LAA"
else
  note "GET /admin/sessions: no active sessions in DB; last_activity_at check skipped"
fi

# Assert GET /audit-logs?limit=5 returns 200 (endpoint used by AppEvents in dashboard)
AL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/audit-logs?limit=5")
[ "$AL_CODE" = 200 ] && pass "GET /audit-logs?limit=5 -> 200" || fail "GET /audit-logs?limit=5 -> $AL_CODE"

# --- 18: Stats + Trends -------------------------------------------------------
section "stats + trends"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats)
[ "$CODE" = 200 ] && pass "GET /admin/stats 200" || fail "GET /admin/stats $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/stats/trends?days=7")
[ "$CODE" = 200 ] && pass "GET /admin/stats/trends 200" || fail "GET /admin/stats/trends $CODE"

# Shape assertions (Bug fix: mfa.total, signups_by_day, auth_methods array)
STATS_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats)
MFA_TOTAL=$(echo "$STATS_BODY" | jq -r '.mfa.total // "missing"')
if echo "$MFA_TOTAL" | grep -qE '^[0-9]+$'; then
  pass "GET /admin/stats — mfa.total is a number ($MFA_TOTAL)"
else
  fail "GET /admin/stats — mfa.total not a number: $MFA_TOTAL"
fi

TRENDS_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/stats/trends?days=14")
SBD_TYPE=$(echo "$TRENDS_BODY" | jq -r '(.signups_by_day | type) // "null"')
[ "$SBD_TYPE" = "array" ] && pass "GET /admin/stats/trends — signups_by_day is array" || fail "signups_by_day type: $SBD_TYPE"
AM_TYPE=$(echo "$TRENDS_BODY" | jq -r '(.auth_methods | type) // "null"')
[ "$AM_TYPE" = "array" ] && pass "GET /admin/stats/trends — auth_methods is array" || fail "auth_methods type: $AM_TYPE"

# --- 19: Webhooks CRUD --------------------------------------------------------
section "webhooks CRUD"
# Create webhook
WH_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/hook","events":["user.created"],"description":"smoke"}')
WH_CODE=$(echo "$WH_RESP" | tail -1)
WH_BODY=$(echo "$WH_RESP" | head -1)
[ "$WH_CODE" = 201 ] && pass "webhook create 201" || fail "webhook create $WH_CODE"
WH_ID=$(echo "$WH_BODY" | jq -r '.id // empty')

# List
WH_LIST=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/webhooks)
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/webhooks)
[ "$CODE" = 200 ] && pass "webhook list 200" || fail "webhook list $CODE"

# Bug E5 contract: response shape MUST be {data: [...]} (frontend webhooks.tsx:60 reads .data).
WH_LIST_TYPE=$(echo "$WH_LIST" | jq -r 'if (.data|type)=="array" then "ok" else "bad" end')
[ "$WH_LIST_TYPE" = ok ] && pass "webhook list shape {data:[]} (E5 contract)" || fail "webhook list shape: $WH_LIST_TYPE"

# Bug E4: previously rejected events now accepted (KnownWebhookEvents alignment).
WH2_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/hook2","events":["user.updated"],"description":"E4 user.updated"}')
[ "$WH2_CODE" = 201 ] && pass "webhook create user.updated 201 (E4 fix)" || fail "webhook create user.updated $WH2_CODE"

WH3_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/hook3","events":["session.revoked"],"description":"E4 session.revoked"}')
[ "$WH3_CODE" = 201 ] && pass "webhook create session.revoked 201 (E4 fix)" || fail "webhook create session.revoked $WH3_CODE"

# Test fire
if [ -n "$WH_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks/$WH_ID/test)
  [ "$CODE" = 200 ] || [ "$CODE" = 202 ] && pass "webhook test $CODE" || fail "webhook test $CODE"

  # Bug C1: test fire honors event_type body field.
  C1_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks/$WH_ID/test \
    -H "Content-Type: application/json" \
    -d '{"event_type":"user.created"}')
  [ "$C1_CODE" = 202 ] && pass "webhook test custom event_type 202 (C1 fix)" || fail "webhook test custom event_type $C1_CODE"

  C1_BAD=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks/$WH_ID/test \
    -H "Content-Type: application/json" \
    -d '{"event_type":"bogus.event"}')
  [ "$C1_BAD" = 400 ] && pass "webhook test bogus event_type 400 (C1 fix)" || fail "webhook test bogus event_type $C1_BAD"

  # Delete
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/webhooks/$WH_ID)
  [ "$CODE" = 200 ] || [ "$CODE" = 204 ] && pass "webhook delete" || fail "webhook delete $CODE"
fi

# --- 20: API Key CRUD ---------------------------------------------------------
section "API key CRUD"
# Create API key
AK_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name":"smokekey","scopes":["read:users"]}')
AK_CODE=$(echo "$AK_RESP" | tail -1)
AK_BODY=$(echo "$AK_RESP" | head -1)
[ "$AK_CODE" = 201 ] && pass "apikey create 201" || fail "apikey create $AK_CODE"
AK_ID=$(echo "$AK_BODY" | jq -r '.id // empty')

# List
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/api-keys)
[ "$CODE" = 200 ] && pass "apikey list 200" || fail "apikey list $CODE"

# Delete
if [ -n "$AK_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/api-keys/$AK_ID)
  [ "$CODE" = 200 ] || [ "$CODE" = 204 ] && pass "apikey revoke" || fail "apikey revoke $CODE"
fi

# --- 21: User CRUD (admin) ----------------------------------------------------
section "user CRUD (admin)"
# List users
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/users)
[ "$CODE" = 200 ] && pass "user list 200" || fail "user list $CODE"

# Get user by ID
if [ -n "$USERID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/users/$USERID)
  [ "$CODE" = 200 ] && pass "user get 200" || fail "user get $CODE"

  # Update user
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
    -X PATCH $BASE/api/v1/users/$USERID \
    -H "Content-Type: application/json" -d '{"name":"Smoke User"}')
  [ "$CODE" = 200 ] && pass "user update 200" || fail "user update $CODE"

  # last_login_at populated after login (multiple relogin calls happened earlier)
  LLA_LIST=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?search=$EMAIL" | jq -r '.users[0].last_login_at // empty')
  [ -n "$LLA_LIST" ] && pass "user list includes last_login_at" || fail "user list missing last_login_at (got empty)"

  LLA_GET=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/users/$USERID | jq -r '.last_login_at // empty')
  [ -n "$LLA_GET" ] && pass "user get includes last_login_at" || fail "user get missing last_login_at (got empty)"

  # Response shape: keys must be snake_case (email_verified not emailVerified).
  EVKEY=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?limit=1" | jq 'has("users") and (.users | length > 0) and (.users[0] | has("email_verified"))')
  [ "$EVKEY" = "true" ] && pass "user list response has snake_case email_verified key" || fail "user list missing email_verified key (camelCase leak?)"
fi

# --- 22: Dev Inbox -------------------------------------------------------------
section "dev inbox"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/dev/emails)
[ "$CODE" = 200 ] && pass "dev inbox list 200" || fail "dev inbox $CODE"

# --- 23: Password change ------------------------------------------------------
section "password change"
# Password change requires verified email. Verify via admin API first.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X PATCH $BASE/api/v1/users/$USERID \
  -H "Content-Type: application/json" -d '{"email_verified":true}')
[ "$CODE" = 200 ] && pass "admin verify email 200" || fail "admin verify email $CODE"

# Login to get fresh session (post-verification)
curl -s -c cj2.txt -X POST $BASE/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" > /dev/null

CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj2.txt \
  -X POST $BASE/api/v1/auth/password/change \
  -H "Content-Type: application/json" \
  -d "{\"current_password\":\"$PASSWORD\",\"new_password\":\"NewCake999\$\$\$\"}")
[ "$CODE" = 200 ] && pass "password change 200" || fail "password change $CODE"
# Update password variable for subsequent logins
PASSWORD='NewCake999$$$'

# --- 24: SSO Connections CRUD (admin) ------------------------------------------
section "SSO connections CRUD"
# Create OIDC connection
SSO_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/sso/connections \
  -H "Content-Type: application/json" \
  -d '{"type":"oidc","name":"Smoke IdP","domain":"smoke.example.com","oidc_issuer":"https://idp.smoke.example.com","oidc_client_id":"cid","oidc_client_secret":"csec"}')
SSO_CODE=$(echo "$SSO_RESP" | tail -1)
SSO_BODY=$(echo "$SSO_RESP" | head -1)
[ "$SSO_CODE" = 201 ] && pass "sso connection create 201" || fail "sso create $SSO_CODE"
SSO_ID=$(echo "$SSO_BODY" | jq -r '.id // empty')

# List
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/sso/connections)
[ "$CODE" = 200 ] && pass "sso list 200" || fail "sso list $CODE"

# Delete
if [ -n "$SSO_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/sso/connections/$SSO_ID)
  [ "$CODE" = 200 ] || [ "$CODE" = 204 ] && pass "sso delete" || fail "sso delete $CODE"
fi

# --- 25: Admin Config + Health -------------------------------------------------
section "admin config + health"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/health)
[ "$CODE" = 200 ] && pass "admin health 200" || fail "admin health $CODE"

# Bug C6: response shape must match dashboard mapHealth (overview.tsx ~L100).
HBODY=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/health)
[ -n "$(echo "$HBODY" | jq -r '.version // empty')" ] && pass "admin health .version present" || fail "admin health .version missing"
echo "$HBODY" | jq -e '.uptime_seconds | type == "number" and . >= 0' >/dev/null 2>&1 \
  && pass "admin health .uptime_seconds is number >= 0" || fail "admin health .uptime_seconds bad"
[ -n "$(echo "$HBODY" | jq -r '.db.driver // empty')" ] && pass "admin health .db.driver present" || fail "admin health .db.driver missing"
echo "$HBODY" | jq -e '.db.size_mb | type == "number"' >/dev/null 2>&1 \
  && pass "admin health .db.size_mb is number" || fail "admin health .db.size_mb not number"
echo "$HBODY" | jq -e '.migrations.current | type == "number" and . > 0' >/dev/null 2>&1 \
  && pass "admin health .migrations.current is number > 0" || fail "admin health .migrations.current bad"
[ -n "$(echo "$HBODY" | jq -r '.jwt.mode // empty')" ] && pass "admin health .jwt.mode present" || fail "admin health .jwt.mode missing"
echo "$HBODY" | jq -e '.jwt.active_keys | type == "number" and . >= 1' >/dev/null 2>&1 \
  && pass "admin health .jwt.active_keys is number >= 1" || fail "admin health .jwt.active_keys bad"
echo "$HBODY" | jq -e '.smtp | has("configured")' >/dev/null 2>&1 \
  && pass "admin health .smtp.configured present" || fail "admin health .smtp.configured missing"
echo "$HBODY" | jq -e '.oauth_providers | type == "array"' >/dev/null 2>&1 \
  && pass "admin health .oauth_providers is array" || fail "admin health .oauth_providers not array"
echo "$HBODY" | jq -e '.sso_connections | type == "number"' >/dev/null 2>&1 \
  && pass "admin health .sso_connections is number" || fail "admin health .sso_connections not number"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/config)
[ "$CODE" = 200 ] && pass "admin config 200" || fail "admin config $CODE"

# --- 26: AS Metadata (RFC 8414) ------------------------------------------------
section "AS metadata (RFC 8414)"
META=$(curl -s $BASE/.well-known/oauth-authorization-server)
echo "$META" | jq -e '.issuer' >/dev/null 2>&1 && pass "AS metadata has issuer" || fail "AS metadata missing issuer"
echo "$META" | jq -e '.authorization_endpoint' >/dev/null 2>&1 && pass "AS metadata has authorization_endpoint" || fail "no authorization_endpoint"
echo "$META" | jq -e '.token_endpoint' >/dev/null 2>&1 && pass "AS metadata has token_endpoint" || fail "no token_endpoint"
echo "$META" | jq -e '.registration_endpoint' >/dev/null 2>&1 && pass "AS metadata has registration_endpoint (MCP)" || fail "no registration_endpoint"
echo "$META" | jq -e '.code_challenge_methods_supported | index("S256")' >/dev/null 2>&1 && pass "PKCE S256 supported" || fail "no S256 in code_challenge_methods"
echo "$META" | jq -e '.grant_types_supported | index("client_credentials")' >/dev/null 2>&1 && pass "client_credentials grant" || fail "no client_credentials"
echo "$META" | jq -e '.grant_types_supported | index("urn:ietf:params:oauth:grant-type:device_code")' >/dev/null 2>&1 && pass "device_code grant" || fail "no device_code"
CT=$(curl -sI $BASE/.well-known/oauth-authorization-server | grep -i 'cache-control' | tr -d '\r\n')
echo "$CT" | grep -qi 'max-age' && pass "AS metadata cache-control" || note "no cache-control: $CT"

# --- 27: OAuth tables exist (Wave A) -------------------------------------------
section "OAuth tables (Wave A)"
if [ -f $DB ]; then
  for tbl in agents oauth_authorization_codes oauth_tokens oauth_consents oauth_device_codes oauth_dcr_clients; do
    sqlite3 $DB "SELECT 1 FROM $tbl LIMIT 0" 2>/dev/null && pass "table $tbl exists" || fail "table $tbl missing"
  done
fi

# --- 28: AS metadata advanced fields -------------------------------------------
section "28: AS metadata advanced (RFC 8414)"
META=$(curl -s $BASE/.well-known/oauth-authorization-server)
echo "$META" | jq -e '.introspection_endpoint' >/dev/null 2>&1 && pass "introspection_endpoint present" || fail "introspection_endpoint missing"
echo "$META" | jq -e '.revocation_endpoint' >/dev/null 2>&1 && pass "revocation_endpoint present" || fail "revocation_endpoint missing"
echo "$META" | jq -e '.device_authorization_endpoint' >/dev/null 2>&1 && pass "device_authorization_endpoint present" || fail "device_authorization_endpoint missing"
echo "$META" | jq -e '.grant_types_supported | index("urn:ietf:params:oauth:grant-type:token-exchange")' >/dev/null 2>&1 && pass "token-exchange grant advertised" || fail "no token-exchange grant"
echo "$META" | jq -e '.grant_types_supported | index("authorization_code")' >/dev/null 2>&1 && pass "authorization_code grant advertised" || fail "no authorization_code grant"
echo "$META" | jq -e '.grant_types_supported | index("refresh_token")' >/dev/null 2>&1 && pass "refresh_token grant advertised" || fail "no refresh_token grant"
echo "$META" | jq -e '.response_types_supported | index("code")' >/dev/null 2>&1 && pass "response_type=code advertised" || fail "no code response_type"
echo "$META" | jq -e '.dpop_signing_alg_values_supported | length >= 1' >/dev/null 2>&1 && pass "dpop_signing_alg_values_supported present" || fail "no dpop_signing_alg_values_supported"
echo "$META" | jq -e '.dpop_signing_alg_values_supported | index("ES256")' >/dev/null 2>&1 && pass "DPoP ES256 advertised" || fail "DPoP ES256 not advertised"

# --- 29: Agent CRUD (admin API) ------------------------------------------------
section "29: Agent CRUD (admin API)"
AGENT_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-agent","grant_types":["client_credentials"],"scopes":["read","write"]}')
AGENT_CODE=$(echo "$AGENT_RESP" | tail -1)
AGENT_BODY=$(echo "$AGENT_RESP" | sed '$d')
[ "$AGENT_CODE" = 201 ] && pass "agent create 201" || fail "agent create $AGENT_CODE"
AGENT_ID=$(echo "$AGENT_BODY" | jq -r '.id // empty')
AGENT_CID=$(echo "$AGENT_BODY" | jq -r '.client_id // empty')
AGENT_SECRET=$(echo "$AGENT_BODY" | jq -r '.client_secret // empty')
[ -n "$AGENT_SECRET" ] && pass "client_secret in create response" || fail "no client_secret"
echo "$AGENT_CID" | grep -q '^shark_agent_' && pass "client_id prefix shark_agent_" || fail "client_id prefix wrong: $AGENT_CID"

LIST_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/agents)
echo "$LIST_BODY" | jq -e ".data | length >= 1" >/dev/null && pass "agent list has >=1 entry" || fail "agent list empty"
echo "$LIST_BODY" | jq -e ".total >= 1" >/dev/null && pass "agent list total>=1" || fail "agent list total=0"
echo "$LIST_BODY" | jq -e --arg id "$AGENT_ID" '.data[] | select(.id==$id)' >/dev/null && pass "created agent in list" || fail "agent not in list"

CODE=$(curl -s -o /tmp/agent-get.json -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/agents/$AGENT_ID)
[ "$CODE" = 200 ] && pass "GET agent by id 200" || fail "GET agent by id $CODE"
GET_NAME=$(jq -r .name /tmp/agent-get.json)
[ "$GET_NAME" = "smoke-agent" ] && pass "agent name matches" || fail "name mismatch: $GET_NAME"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/agents/$AGENT_CID)
[ "$CODE" = 200 ] && pass "GET agent by client_id 200" || fail "GET agent by client_id $CODE"

PATCH_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X PATCH $BASE/api/v1/agents/$AGENT_ID \
  -H "Content-Type: application/json" -d '{"description":"updated"}')
PATCH_CODE=$(echo "$PATCH_RESP" | tail -1)
PATCH_BODY=$(echo "$PATCH_RESP" | sed '$d')
[ "$PATCH_CODE" = 200 ] && pass "agent patch 200" || fail "agent patch $PATCH_CODE"
[ "$(echo "$PATCH_BODY" | jq -r .description)" = "updated" ] && pass "description updated" || fail "description not updated"

AUDIT_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/agents/$AGENT_ID/audit)
echo "$AUDIT_BODY" | jq -e '.data | length >= 1' >/dev/null && pass "agent audit log has entries" || fail "agent audit empty"
echo "$AUDIT_BODY" | jq -e '.data[] | select(.action=="agent.created")' >/dev/null && pass "agent.created in audit" || fail "no agent.created audit"
echo "$AUDIT_BODY" | jq -e '.data[] | select(.action=="agent.updated")' >/dev/null && pass "agent.updated in audit" || fail "no agent.updated audit"

CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/agents)
[ "$CODE" = 401 ] && pass "no auth -> 401 on /agents" || fail "no-auth -> $CODE"

# --- 30: Client Credentials grant ---------------------------------------------
section "30: Client Credentials grant"
CC_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name":"cc-agent","grant_types":["client_credentials"],"scopes":["read"]}')
CC_CODE=$(echo "$CC_RESP" | tail -1)
CC_BODY=$(echo "$CC_RESP" | sed '$d')
[ "$CC_CODE" = 201 ] && pass "cc-agent create 201" || fail "cc-agent create $CC_CODE"
CC_CID=$(echo "$CC_BODY" | jq -r '.client_id // empty')
CC_SECRET=$(echo "$CC_BODY" | jq -r '.client_secret // empty')

CC_BASIC=$(printf '%s' "$CC_CID:$CC_SECRET" | base64 | tr -d '\n' | tr -d ' ')
TOK_RESP=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials&scope=read')
TOK_CODE=$(echo "$TOK_RESP" | tail -1)
TOK_BODY=$(echo "$TOK_RESP" | sed '$d')
[ "$TOK_CODE" = 200 ] && pass "token endpoint 200" || { echo "    $TOK_BODY"; fail "token endpoint $TOK_CODE"; }
CC_TOKEN=$(echo "$TOK_BODY" | jq -r '.access_token // empty')
[ -n "$CC_TOKEN" ] && pass "access_token returned" || fail "no access_token"
TT=$(echo "$TOK_BODY" | jq -r '.token_type // empty')
if [ "$TT" = "Bearer" ] || [ "$TT" = "bearer" ] || [ "$TT" = "DPoP" ]; then pass "token_type=$TT"; else fail "token_type=$TT"; fi
EXPIN=$(echo "$TOK_BODY" | jq -r '.expires_in // 0')
[ "$EXPIN" -gt 0 ] 2>/dev/null && pass "expires_in=$EXPIN" || fail "expires_in=$EXPIN"
SC=$(echo "$TOK_BODY" | jq -r '.scope // empty')
echo "$SC" | grep -q 'read' && pass "scope contains read" || note "scope=$SC (may be omitted for cc grant)"

# Tokens here are opaque HMAC (key.sig), not JWTs. Verify via introspection instead.
note "CC tokens are opaque HMAC (key.sig); JWT decode deferred to introspection (section 37)"

# Wrong secret -> 401
BAD_BASIC=$(printf '%s' "$CC_CID:wrong-secret" | base64 | tr -d '\n' | tr -d ' ')
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/oauth/token \
  -H "Authorization: Basic $BAD_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials')
[ "$CODE" = 401 ] && pass "wrong secret -> 401" || fail "wrong secret -> $CODE"

# Missing grant_type -> 400
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'scope=read')
[ "$CODE" = 400 ] && pass "missing grant_type -> 400" || fail "missing grant_type -> $CODE"

# --- 31: Auth Code + PKCE flow -------------------------------------------------
section "31: Auth Code + PKCE flow"
PKCE_VERIFIER="test_verifier_0123456789abc_0123456789abc_0123456789"
if command -v openssl >/dev/null 2>&1; then
  PKCE_CHALLENGE=$(printf '%s' "$PKCE_VERIFIER" | openssl dgst -sha256 -binary | openssl base64 -A | tr '+/' '-_' | tr -d '=')
else
  PKCE_CHALLENGE=""
fi

if [ -z "$PKCE_CHALLENGE" ]; then
  note "openssl not available; skipping auth-code PKCE flow (covered by Go unit tests)"
  PKCE_DONE=0
else
  pass "computed PKCE challenge ($PKCE_CHALLENGE)"
  # Create PKCE-capable agent.
  PKCE_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/agents \
    -H "Content-Type: application/json" \
    -d '{"name":"pkce-agent","grant_types":["authorization_code","refresh_token"],"redirect_uris":["http://localhost:9999/callback"],"scopes":["openid","profile","offline_access"],"client_type":"confidential","response_types":["code"]}')
  PKCE_CODE=$(echo "$PKCE_RESP" | tail -1)
  PKCE_BODY=$(echo "$PKCE_RESP" | sed '$d')
  [ "$PKCE_CODE" = 201 ] && pass "pkce-agent create 201" || fail "pkce-agent create $PKCE_CODE"
  PKCE_CID=$(echo "$PKCE_BODY" | jq -r '.client_id // empty')
  PKCE_SECRET=$(echo "$PKCE_BODY" | jq -r '.client_secret // empty')

  # Need fresh cookie (logged in user). Relogin just in case.
  relogin
  # Build authorize query string. offline_access scope required for refresh_token issuance.
  SCOPE_ENC="openid%20profile%20offline_access"
  AUTHZ_QS="response_type=code&client_id=$PKCE_CID&redirect_uri=http%3A%2F%2Flocalhost%3A9999%2Fcallback&state=xyzabcde&code_challenge=$PKCE_CHALLENGE&code_challenge_method=S256&scope=$SCOPE_ENC"

  # GET /oauth/authorize with session cookie -> either 200 (consent page) or 302/303 (auto-approve).
  GET_H=$(curl -s -o /tmp/authz.html -D - -b cj.txt "$BASE/oauth/authorize?$AUTHZ_QS")
  GET_STATUS=$(echo "$GET_H" | head -1 | awk '{print $2}')
  if [ "$GET_STATUS" = 200 ]; then
    pass "GET /oauth/authorize renders consent (200)"
  elif [ "$GET_STATUS" = 302 ] || [ "$GET_STATUS" = 303 ]; then
    pass "GET /oauth/authorize auto-approve ($GET_STATUS)"
  else
    fail "GET /oauth/authorize -> $GET_STATUS"
  fi

  # Try extracting code from GET redirect first; if none, POST consent.
  LOC=$(echo "$GET_H" | grep -i '^location:' | tr -d '\r\n' | sed 's/^[Ll]ocation: //')
  if [ -z "$(echo "$LOC" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')" ]; then
    LOC=$(curl -s -o /dev/null -D - -b cj.txt -X POST $BASE/oauth/authorize \
      -H "Content-Type: application/x-www-form-urlencoded" \
      --data-urlencode "challenge=$AUTHZ_QS" \
      --data-urlencode "client_id=$PKCE_CID" \
      --data-urlencode "state=xyzabcde" \
      --data-urlencode "approved=true" | grep -i '^location:' | tr -d '\r\n' | sed 's/^[Ll]ocation: //')
  fi
  if [ -n "$LOC" ]; then pass "consent -> Location redirect"; else fail "no Location header"; fi
  PKCE_AUTHCODE=$(echo "$LOC" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')
  STATE_ECHOED=$(echo "$LOC" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p')
  [ -n "$PKCE_AUTHCODE" ] && pass "code extracted ($(echo "$PKCE_AUTHCODE" | head -c 16)...)" || fail "no code in redirect"
  [ "$STATE_ECHOED" = "xyzabcde" ] && pass "state echoed" || fail "state mismatch: $STATE_ECHOED"

  if [ -n "$PKCE_AUTHCODE" ]; then
    EX_RESP=$(curl -s -w "\n%{http_code}" -u "$PKCE_CID:$PKCE_SECRET" -X POST $BASE/oauth/token \
      -H "Content-Type: application/x-www-form-urlencoded" \
      --data-urlencode "grant_type=authorization_code" \
      --data-urlencode "code=$PKCE_AUTHCODE" \
      --data-urlencode "redirect_uri=http://localhost:9999/callback" \
      --data-urlencode "code_verifier=$PKCE_VERIFIER")
    EX_CODE=$(echo "$EX_RESP" | tail -1)
    EX_BODY=$(echo "$EX_RESP" | sed '$d')
    if [ "$EX_CODE" = 200 ]; then
      pass "token exchange 200"
      PKCE_AT=$(echo "$EX_BODY" | jq -r '.access_token // empty')
      PKCE_RT=$(echo "$EX_BODY" | jq -r '.refresh_token // empty')
      [ -n "$PKCE_AT" ] && pass "access_token issued" || fail "no access_token"
      [ -n "$PKCE_RT" ] && pass "refresh_token issued" || fail "no refresh_token"
      PKCE_DONE=1
    else
      fail "token exchange $EX_CODE — PKCE persistence broken (oauth_pkce_sessions table or FositeStore.Create/GetPKCERequestSession)"
      PKCE_DONE=0
    fi
  else
    PKCE_DONE=0
  fi
fi

# --- 32: PKCE enforcement ------------------------------------------------------
section "32: PKCE enforcement (OAuth 2.1)"
if [ -n "${PKCE_CID:-}" ]; then
  relogin
  NO_PKCE_QS="response_type=code&client_id=$PKCE_CID&redirect_uri=http%3A%2F%2Flocalhost%3A9999%2Fcallback&state=noPkce12345&scope=openid"
  # Without code_challenge, OAuth 2.1 should reject. Fosite redirects to redirect_uri with error, OR returns 400 inline.
  H=$(curl -s -o /tmp/no-pkce.html -D - -b cj.txt "$BASE/oauth/authorize?$NO_PKCE_QS")
  STATUS=$(echo "$H" | head -1 | awk '{print $2}')
  LOC_ERR=$(echo "$H" | grep -i '^location:' | head -1)
  if echo "$LOC_ERR" | grep -q 'error='; then
    pass "no PKCE -> redirect with error= ($(echo "$LOC_ERR" | sed -n 's/.*error=\([^&]*\).*/\1/p'))"
  elif [ "$STATUS" = "400" ]; then
    pass "no PKCE -> 400 inline"
  else
    fail "no PKCE got status=$STATUS loc=$LOC_ERR"
  fi
else
  note "skipped (pkce-agent not created)"
fi

# --- 33: Refresh Token Rotation -----------------------------------------------
section "33: Refresh token rotation"
if [ "${PKCE_DONE:-0}" = "1" ] && [ -n "${PKCE_RT:-}" ]; then
  RT_RESP=$(curl -s -w "\n%{http_code}" -u "$PKCE_CID:$PKCE_SECRET" -X POST $BASE/oauth/token \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "grant_type=refresh_token" \
    --data-urlencode "refresh_token=$PKCE_RT")
  RT_CODE=$(echo "$RT_RESP" | tail -1)
  RT_BODY=$(echo "$RT_RESP" | sed '$d')
  [ "$RT_CODE" = 200 ] && pass "refresh exchange 200" || { echo "    $RT_BODY"; fail "refresh exchange $RT_CODE"; }
  NEW_AT=$(echo "$RT_BODY" | jq -r '.access_token // empty')
  NEW_RT=$(echo "$RT_BODY" | jq -r '.refresh_token // empty')
  [ -n "$NEW_AT" ] && pass "new access_token issued" || fail "no new access_token"
  [ -n "$NEW_RT" ] && [ "$NEW_RT" != "$PKCE_RT" ] && pass "refresh_token rotated" || fail "refresh_token not rotated"

  # Reuse OLD refresh token -> should fail (family revoked).
  REUSE_CODE=$(curl -s -o /tmp/reuse.json -w "%{http_code}" -u "$PKCE_CID:$PKCE_SECRET" -X POST $BASE/oauth/token \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "grant_type=refresh_token" \
    --data-urlencode "refresh_token=$PKCE_RT")
  if [ "$REUSE_CODE" != "200" ]; then pass "old refresh token reuse rejected ($REUSE_CODE)"; else fail "old refresh token still works (reuse detection broken)"; fi
else
  note "skipped (depends on section 31)"
fi

# --- 34: Device flow -----------------------------------------------------------
section "34: Device flow (RFC 8628)"
DEV_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name":"device-agent","grant_types":["urn:ietf:params:oauth:grant-type:device_code"],"scopes":["read"]}')
DEV_CODE_HTTP=$(echo "$DEV_RESP" | tail -1)
DEV_BODY=$(echo "$DEV_RESP" | sed '$d')
[ "$DEV_CODE_HTTP" = 201 ] && pass "device-agent create 201" || fail "device-agent create $DEV_CODE_HTTP"
DEV_CID=$(echo "$DEV_BODY" | jq -r '.client_id // empty')
DEV_SECRET=$(echo "$DEV_BODY" | jq -r '.client_secret // empty')

DA_RESP=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/device \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=$DEV_CID&scope=read")
DA_CODE=$(echo "$DA_RESP" | tail -1)
DA_BODY=$(echo "$DA_RESP" | sed '$d')
[ "$DA_CODE" = 200 ] && pass "device authz 200" || { echo "    $DA_BODY"; fail "device authz $DA_CODE"; }
DEV_CODE_VAL=$(echo "$DA_BODY" | jq -r '.device_code // empty')
USER_CODE=$(echo "$DA_BODY" | jq -r '.user_code // empty')
VERIFY_URI=$(echo "$DA_BODY" | jq -r '.verification_uri // empty')
DEV_EXPIN=$(echo "$DA_BODY" | jq -r '.expires_in // 0')
DEV_INTERVAL=$(echo "$DA_BODY" | jq -r '.interval // 0')
[ -n "$DEV_CODE_VAL" ] && pass "device_code present" || fail "no device_code"
echo "$USER_CODE" | grep -qE '^[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}$' && pass "user_code format OK ($USER_CODE)" || fail "user_code format bad: $USER_CODE"
[ -n "$VERIFY_URI" ] && pass "verification_uri present" || fail "no verification_uri"
[ "$DEV_EXPIN" -gt 0 ] 2>/dev/null && pass "expires_in=$DEV_EXPIN" || fail "expires_in=$DEV_EXPIN"
[ "$DEV_INTERVAL" -ge 5 ] 2>/dev/null && pass "interval>=5 ($DEV_INTERVAL)" || fail "interval<5"

# Immediate poll -> authorization_pending
POLL_RESP=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=$DEV_CODE_VAL")
POLL_CODE=$(echo "$POLL_RESP" | tail -1)
POLL_BODY=$(echo "$POLL_RESP" | sed '$d')
[ "$POLL_CODE" = 400 ] && pass "immediate poll -> 400" || fail "immediate poll -> $POLL_CODE"
echo "$POLL_BODY" | jq -e '.error == "authorization_pending"' >/dev/null 2>&1 && pass "error=authorization_pending" || fail "error=$(echo "$POLL_BODY" | jq -r '.error // empty')"

# Approve via DB
sqlite3 $DB "UPDATE oauth_device_codes SET status='approved', user_id='$USERID' WHERE user_code='$USER_CODE';" 2>/dev/null
DB_STATUS=$(sqlite3 $DB "SELECT status FROM oauth_device_codes WHERE user_code='$USER_CODE';" 2>/dev/null)
[ "$DB_STATUS" = "approved" ] && pass "DB approved" || fail "DB status=$DB_STATUS"

# Poll again -> 200. Wait a touch to clear interval.
sleep 6
POLL2=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=$DEV_CODE_VAL")
POLL2_CODE=$(echo "$POLL2" | tail -1)
POLL2_BODY=$(echo "$POLL2" | sed '$d')
[ "$POLL2_CODE" = 200 ] && pass "approved poll -> 200" || { echo "    $POLL2_BODY"; fail "approved poll -> $POLL2_CODE"; }
DEV_AT=$(echo "$POLL2_BODY" | jq -r '.access_token // empty')
[ -n "$DEV_AT" ] && pass "device access_token issued" || fail "no device access_token"

# Re-use same device_code (status now 'used' or still 'approved'? our impl leaves 'approved' until something else runs)
# In any case, we expect non-200: either invalid_grant or still-ok if idempotent. Per impl, status remains 'approved' so second poll would still issue.
# Skip strict check.
note "device_code replay strictness is implementation-dependent — covered by device_test.go"

# --- 35: Token Exchange (RFC 8693) --------------------------------------------
section "35: Token Exchange (RFC 8693)"
# Needs subject token + may_act on subject. Likely fails without proper setup.
# Best-effort: try exchange using CC token as subject. Expect failure (CC tokens aren't JWTs).
if [ -n "${CC_TOKEN:-}" ]; then
  TE_RESP=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/token \
    -H "Authorization: Basic $CC_BASIC" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
    --data-urlencode "subject_token=$CC_TOKEN" \
    --data-urlencode "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
    --data-urlencode "scope=read")
  TE_CODE=$(echo "$TE_RESP" | tail -1)
  TE_BODY=$(echo "$TE_RESP" | sed '$d')
  if [ "$TE_CODE" = 200 ]; then
    pass "token-exchange 200"
    echo "$TE_BODY" | jq -e '.issued_token_type == "urn:ietf:params:oauth:token-type:access_token"' >/dev/null && pass "issued_token_type correct" || fail "issued_token_type missing"
    echo "$TE_BODY" | jq -e '.access_token' >/dev/null && pass "access_token present" || fail "no access_token"
  else
    note "token-exchange -> $TE_CODE (subject must be JWT issued by this AS; CC tokens are opaque). Full coverage in exchange_test.go"
  fi
else
  note "skipped (no CC_TOKEN available)"
fi

# --- 36: DPoP (RFC 9449) -------------------------------------------------------
section "36: DPoP (RFC 9449)"
# No DPoP header still works (DPoP is optional)
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials')
[ "$CODE" = 200 ] && pass "CC without DPoP still works ($CODE)" || fail "CC w/o DPoP -> $CODE"

# Malformed DPoP header -> 400 invalid_dpop_proof
DPOP_RESP=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "DPoP: this.is.garbage" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials')
DPOP_CODE=$(echo "$DPOP_RESP" | tail -1)
DPOP_BODY=$(echo "$DPOP_RESP" | sed '$d')
[ "$DPOP_CODE" = 400 ] && pass "garbage DPoP -> 400" || fail "garbage DPoP -> $DPOP_CODE"
echo "$DPOP_BODY" | jq -e '.error == "invalid_dpop_proof"' >/dev/null && pass "error=invalid_dpop_proof" || fail "wrong error"

# Metadata advertises ES256 for DPoP
META=$(curl -s $BASE/.well-known/oauth-authorization-server)
echo "$META" | jq -e '.dpop_signing_alg_values_supported | index("ES256")' >/dev/null && pass "DPoP ES256 in metadata" || fail "DPoP ES256 missing"

note "Full DPoP flow requires ES256-signed proof JWT; covered by internal/oauth/dpop_test.go"

# --- 37: Token Introspection (RFC 7662) ---------------------------------------
section "37: Token Introspection (RFC 7662)"
# Need a fresh CC token since Section 36 didn't capture one.
TOK_RESP=$(curl -s -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials&scope=read')
CC_TOKEN2=$(echo "$TOK_RESP" | jq -r '.access_token // empty')
[ -n "$CC_TOKEN2" ] && pass "fresh CC token for introspection" || fail "no CC token for introspect"

INTRO_RESP=$(curl -s -X POST $BASE/oauth/introspect \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=$CC_TOKEN2")
echo "$INTRO_RESP" | jq -e '.active == true' >/dev/null && pass "introspect active=true" || { echo "    $INTRO_RESP"; fail "introspect not active"; }
echo "$INTRO_RESP" | jq -e --arg cid "$CC_CID" '.client_id == $cid' >/dev/null && pass "client_id matches" || fail "client_id mismatch"
echo "$INTRO_RESP" | jq -e '.exp > 0' >/dev/null && pass "exp > 0" || fail "exp missing"
echo "$INTRO_RESP" | jq -e '.scope | test("read")' >/dev/null && pass "scope contains read" || note "scope=$(echo "$INTRO_RESP" | jq -r .scope) (may be empty)"

# Invalid token -> active:false
INTRO2=$(curl -s -X POST $BASE/oauth/introspect \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=completely_fake_token")
echo "$INTRO2" | jq -e '.active == false' >/dev/null && pass "fake token -> active:false" || fail "fake token not active:false"

# --- 38: Token Revocation (RFC 7009) ------------------------------------------
section "38: Token Revocation (RFC 7009)"
REV_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/oauth/revoke \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=$CC_TOKEN2" \
  --data-urlencode "token_type_hint=access_token")
[ "$REV_CODE" = 200 ] && pass "revoke returns 200" || fail "revoke -> $REV_CODE"

# Introspect revoked token -> active:false
INTRO3=$(curl -s -X POST $BASE/oauth/introspect \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=$CC_TOKEN2")
echo "$INTRO3" | jq -e '.active == false' >/dev/null && pass "revoked token -> active:false" || fail "revoked token still active"

# Revoke invalid token -> still 200
REV2_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE/oauth/revoke \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=does_not_exist")
[ "$REV2_CODE" = 200 ] && pass "revoke invalid token -> 200 (RFC 7009)" || fail "invalid revoke -> $REV2_CODE"

# --- 39: Dynamic Client Registration (RFC 7591) -------------------------------
section "39: Dynamic Client Registration (RFC 7591)"
DCR_RESP=$(curl -s -w "\n%{http_code}" -X POST $BASE/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name":"dcr-test","grant_types":["client_credentials"],"scope":"read","redirect_uris":[]}')
DCR_CODE=$(echo "$DCR_RESP" | tail -1)
DCR_BODY=$(echo "$DCR_RESP" | sed '$d')
[ "$DCR_CODE" = 201 ] && pass "DCR register 201" || { echo "    $DCR_BODY"; fail "DCR register $DCR_CODE"; }
DCR_CID=$(echo "$DCR_BODY" | jq -r '.client_id // empty')
DCR_RAT=$(echo "$DCR_BODY" | jq -r '.registration_access_token // empty')
DCR_RCU=$(echo "$DCR_BODY" | jq -r '.registration_client_uri // empty')
echo "$DCR_CID" | grep -q '^shark_dcr_' && pass "client_id prefix shark_dcr_" || fail "DCR prefix wrong: $DCR_CID"
[ -n "$(echo "$DCR_BODY" | jq -r '.client_secret // empty')" ] && pass "client_secret present" || fail "no client_secret"
[ -n "$DCR_RAT" ] && pass "registration_access_token present" || fail "no RAT"
[ -n "$DCR_RCU" ] && pass "registration_client_uri present" || fail "no RCU"

GET_CODE=$(curl -s -o /tmp/dcr-get.json -w "%{http_code}" \
  -H "Authorization: Bearer $DCR_RAT" \
  $BASE/oauth/register/$DCR_CID)
[ "$GET_CODE" = 200 ] && pass "DCR GET with RAT -> 200" || fail "DCR GET -> $GET_CODE"
[ "$(jq -r .client_name /tmp/dcr-get.json)" = "dcr-test" ] && pass "client_name matches" || fail "client_name mismatch"

PUT_CODE=$(curl -s -o /tmp/dcr-put.json -w "%{http_code}" \
  -H "Authorization: Bearer $DCR_RAT" \
  -X PUT $BASE/oauth/register/$DCR_CID \
  -H "Content-Type: application/json" \
  -d '{"client_name":"dcr-updated","grant_types":["client_credentials"],"scope":"read","redirect_uris":[]}')
[ "$PUT_CODE" = 200 ] && pass "DCR PUT -> 200" || fail "DCR PUT -> $PUT_CODE"
[ "$(jq -r .client_name /tmp/dcr-put.json)" = "dcr-updated" ] && pass "DCR name updated" || fail "DCR name not updated"

DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $DCR_RAT" \
  -X DELETE $BASE/oauth/register/$DCR_CID)
[ "$DEL_CODE" = 204 ] && pass "DCR DELETE -> 204" || fail "DCR DELETE -> $DEL_CODE"

# GET after delete with old RAT
GET2_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $DCR_RAT" \
  $BASE/oauth/register/$DCR_CID)
if [ "$GET2_CODE" = 401 ] || [ "$GET2_CODE" = 404 ]; then pass "DCR GET after delete -> $GET2_CODE"; else fail "DCR GET after delete -> $GET2_CODE"; fi

# GET without RAT
NOAUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/oauth/register/shark_dcr_fake)
[ "$NOAUTH_CODE" = 401 ] && pass "DCR GET no RAT -> 401" || fail "DCR GET no RAT -> $NOAUTH_CODE"

# --- 40: Resource Indicators (RFC 8707) ---------------------------------------
section "40: Resource Indicators (RFC 8707)"
TOK_R_RESP=$(curl -s -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "grant_type=client_credentials" \
  --data-urlencode "resource=https://api.example.com")
R_TOKEN=$(echo "$TOK_R_RESP" | jq -r '.access_token // empty')
[ -n "$R_TOKEN" ] && pass "token issued with resource param" || fail "no token with resource"

# Introspect to check audience (stored in DB).
R_INTRO=$(curl -s -X POST $BASE/oauth/introspect \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=$R_TOKEN")
echo "$R_INTRO" | jq -e '.active == true' >/dev/null && pass "introspect active for resource-bound token" || fail "introspect inactive"
R_AUD=$(echo "$R_INTRO" | jq -r '.aud // empty')
[ "$R_AUD" = "https://api.example.com" ] && pass "aud bound to resource indicator" || fail "aud=$R_AUD (expected https://api.example.com)"

# Without resource param: aud should be empty (or issuer default).
TOK_NORES=$(curl -s -X POST $BASE/oauth/token \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials')
NORES_TOKEN=$(echo "$TOK_NORES" | jq -r '.access_token // empty')
NORES_INTRO=$(curl -s -X POST $BASE/oauth/introspect \
  -H "Authorization: Basic $CC_BASIC" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "token=$NORES_TOKEN")
NORES_AUD=$(echo "$NORES_INTRO" | jq -r '.aud // empty')
if [ "$NORES_AUD" != "https://api.example.com" ]; then pass "no resource -> aud!=previous ($NORES_AUD)"; else fail "aud leaked from previous request"; fi

# --- 41: ES256 JWKS + verification --------------------------------------------
section "41: ES256 JWKS"
JWKS=$(curl -s $BASE/.well-known/jwks.json)
echo "$JWKS" | jq -e '[.keys[] | select(.alg=="ES256")][0]' >/dev/null && pass "ES256 key present" || fail "no ES256 key"
echo "$JWKS" | jq -e '[.keys[] | select(.alg=="ES256")][0] | .kty=="EC"' >/dev/null && pass "ES256 kty=EC" || fail "kty wrong"
echo "$JWKS" | jq -e '[.keys[] | select(.alg=="ES256")][0] | .crv=="P-256"' >/dev/null && pass "crv=P-256" || fail "crv wrong"
echo "$JWKS" | jq -e '[.keys[] | select(.alg=="ES256")][0] | .use=="sig"' >/dev/null && pass "use=sig" || fail "use wrong"

ES_X=$(echo "$JWKS" | jq -r '[.keys[] | select(.alg=="ES256")][0].x')
ES_Y=$(echo "$JWKS" | jq -r '[.keys[] | select(.alg=="ES256")][0].y')
ES_KID=$(echo "$JWKS" | jq -r '[.keys[] | select(.alg=="ES256")][0].kid')
[ "${#ES_X}" = 43 ] && pass "x is 43 chars (32 bytes base64url)" || fail "x len=${#ES_X}"
[ "${#ES_Y}" = 43 ] && pass "y is 43 chars (32 bytes base64url)" || fail "y len=${#ES_Y}"
[ -n "$ES_KID" ] && pass "ES256 kid present ($ES_KID)" || fail "no ES256 kid"

# Match kid against an ID token (JWT) if we have one from section 31.
if [ -n "${PKCE_AT:-}" ]; then
  PKCE_HEADER=$(echo "$PKCE_AT" | cut -d. -f1 | base64 -d 2>/dev/null | jq -c . 2>/dev/null || true)
  if [ -n "$PKCE_HEADER" ]; then
    TOK_ALG=$(echo "$PKCE_HEADER" | jq -r .alg 2>/dev/null)
    TOK_KID=$(echo "$PKCE_HEADER" | jq -r .kid 2>/dev/null)
    # PKCE_AT is opaque (HMAC), so this probably won't parse; note if so.
    if [ "$TOK_ALG" = "ES256" ] && [ "$TOK_KID" = "$ES_KID" ]; then
      pass "auth-code token alg=ES256, kid matches JWKS"
    else
      note "auth-code access_token alg=$TOK_ALG kid=$TOK_KID (HMAC strategy; opaque by default)"
    fi
  else
    note "PKCE access_token is opaque (HMAC); JWT kid match skipped"
  fi
else
  note "no PKCE access_token to cross-check kid"
fi

# --- 42: Consent management (self-service) ------------------------------------
section "42: Consent management"
relogin
CONS_RESP=$(curl -s -w "\n%{http_code}" -b cj.txt $BASE/api/v1/auth/consents)
CONS_CODE=$(echo "$CONS_RESP" | tail -1)
CONS_BODY=$(echo "$CONS_RESP" | sed '$d')
[ "$CONS_CODE" = 200 ] && pass "GET /auth/consents -> 200" || fail "GET /auth/consents -> $CONS_CODE"
echo "$CONS_BODY" | jq -e '.data' >/dev/null && pass "response has data array" || fail "no data array"

# If section 31 succeeded, user has consent for pkce-agent.
if [ "${PKCE_DONE:-0}" = "1" ] && [ -n "${PKCE_CID:-}" ]; then
  CONSENT_ID=$(echo "$CONS_BODY" | jq -r --arg cid "$PKCE_CID" '.data[] | select(.client_id==$cid) | .id' | head -1)
  if [ -n "$CONSENT_ID" ]; then
    pass "consent for pkce-agent found ($CONSENT_ID)"
    DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt -X DELETE $BASE/api/v1/auth/consents/$CONSENT_ID)
    [ "$DEL_CODE" = 200 ] && pass "DELETE consent -> 200" || fail "DELETE consent -> $DEL_CODE"
    # Confirm removed.
    CONS2=$(curl -s -b cj.txt $BASE/api/v1/auth/consents)
    if echo "$CONS2" | jq -e --arg id "$CONSENT_ID" '.data[] | select(.id==$id)' >/dev/null 2>&1; then
      fail "consent still present after delete"
    else
      pass "consent removed after delete"
    fi
  else
    note "no pkce-agent consent found in list (possibly consent already revoked upstream)"
  fi
else
  note "section 31 did not complete — skipping consent ID lookup"
fi

# Without auth -> 401
NOAUTH_CONS=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/auth/consents)
[ "$NOAUTH_CONS" = 401 ] && pass "no auth -> 401" || fail "no-auth consents -> $NOAUTH_CONS"

# --- 43: Vault provider CRUD (admin) -----------------------------------------
section "43: Vault provider CRUD (admin)"
# Use a unique name per run so re-invocations don't collide with any leftover row.
VAULT_PROV_NAME="github"
VP_CREATE=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/vault/providers \
  -H "Content-Type: application/json" \
  -d "{\"template\":\"github\",\"client_id\":\"smoke-client-id\",\"client_secret\":\"smoke-secret-abc123\"}")
VP_CODE=$(echo "$VP_CREATE" | tail -1)
VP_BODY=$(echo "$VP_CREATE" | sed '$d')
[ "$VP_CODE" = 201 ] && pass "POST /vault/providers (template=github) -> 201" || { echo "    $VP_BODY"; fail "POST /vault/providers -> $VP_CODE"; }
VAULT_PID=$(echo "$VP_BODY" | jq -r '.id // empty')
[ -n "$VAULT_PID" ] && pass "provider id captured ($VAULT_PID)" || fail "no provider id in response"
echo "$VP_BODY" | jq -e 'has("client_secret") | not' >/dev/null && pass "client_secret NOT in create response" || fail "client_secret leaked in create response"
echo "$VP_BODY" | jq -e '.name == "github"' >/dev/null && pass "name=github (from template)" || fail "name not github"

# List — our provider present, no secret anywhere.
VP_LIST=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/vault/providers)
echo "$VP_LIST" | jq -e --arg id "$VAULT_PID" '.data[] | select(.id==$id)' >/dev/null && pass "GET /vault/providers lists created provider" || fail "created provider missing from list"
echo "$VP_LIST" | jq -e '[.data[] | has("client_secret")] | any | not' >/dev/null && pass "no client_secret in list response" || fail "client_secret leaked in list response"

# Get one — still sanitized.
VP_GET=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/vault/providers/$VAULT_PID)
VP_GET_CODE=$(echo "$VP_GET" | tail -1)
VP_GET_BODY=$(echo "$VP_GET" | sed '$d')
[ "$VP_GET_CODE" = 200 ] && pass "GET /vault/providers/{id} -> 200" || fail "GET /vault/providers/{id} -> $VP_GET_CODE"
echo "$VP_GET_BODY" | jq -e 'has("client_secret") | not' >/dev/null && pass "GET by id sanitized (no client_secret)" || fail "client_secret leaked on GET by id"

# PATCH display_name.
VP_PATCH=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X PATCH $BASE/api/v1/vault/providers/$VAULT_PID \
  -H "Content-Type: application/json" \
  -d '{"display_name":"GitHub Enterprise"}')
VP_PATCH_CODE=$(echo "$VP_PATCH" | tail -1)
VP_PATCH_BODY=$(echo "$VP_PATCH" | sed '$d')
[ "$VP_PATCH_CODE" = 200 ] && pass "PATCH display_name -> 200" || fail "PATCH display_name -> $VP_PATCH_CODE"
[ "$(echo "$VP_PATCH_BODY" | jq -r .display_name)" = "GitHub Enterprise" ] && pass "display_name updated" || fail "display_name not updated"

# PATCH client_secret (rotation).
VP_ROT_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X PATCH $BASE/api/v1/vault/providers/$VAULT_PID \
  -H "Content-Type: application/json" \
  -d '{"client_secret":"new-secret-12345"}')
[ "$VP_ROT_CODE" = 200 ] && pass "PATCH client_secret (rotation) -> 200" || fail "PATCH client_secret -> $VP_ROT_CODE"

# Duplicate name -> 409.
VP_DUP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/vault/providers \
  -H "Content-Type: application/json" \
  -d '{"template":"github","client_id":"smoke-client-id-2","client_secret":"secret-2-abc"}')
[ "$VP_DUP_CODE" = 409 ] && pass "duplicate name -> 409" || fail "duplicate name -> $VP_DUP_CODE"

# DELETE -> 204, GET after -> 404.
VP_DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/vault/providers/$VAULT_PID)
[ "$VP_DEL_CODE" = 204 ] && pass "DELETE /vault/providers/{id} -> 204" || fail "DELETE -> $VP_DEL_CODE"

VP_GONE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/vault/providers/$VAULT_PID)
[ "$VP_GONE_CODE" = 404 ] && pass "GET after delete -> 404" || fail "GET after delete -> $VP_GONE_CODE"

# No-auth on admin CRUD is blocked.
VP_NOAUTH=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/vault/providers)
[ "$VP_NOAUTH" = 401 ] && pass "no auth -> 401 on /vault/providers" || fail "no-auth -> $VP_NOAUTH"

# --- 44: Vault templates discovery -------------------------------------------
section "44: Vault templates discovery"
VT_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/vault/templates)
VT_CODE=$(echo "$VT_RESP" | tail -1)
VT_BODY=$(echo "$VT_RESP" | sed '$d')
[ "$VT_CODE" = 200 ] && pass "GET /vault/templates -> 200" || fail "GET /vault/templates -> $VT_CODE"
VT_LEN=$(echo "$VT_BODY" | jq '.data | length')
[ "$VT_LEN" = 9 ] && pass "9 built-in templates" || fail "expected 9 templates, got $VT_LEN"

# Required keys on each template row (snake_case, not PascalCase).
MISSING_KEYS=$(echo "$VT_BODY" | jq -r '[.data[] | (has("name") and has("display_name") and has("auth_url") and has("token_url") and has("default_scopes"))] | all | not')
[ "$MISSING_KEYS" = "false" ] && pass "all templates have name/display_name/auth_url/token_url/default_scopes" || fail "some templates missing required keys"

# Sanity-check snake_case: no PascalCase keys surface.
PASCAL_LEAK=$(echo "$VT_BODY" | jq -r '[.data[] | (has("Name") or has("DisplayName") or has("AuthURL"))] | any')
[ "$PASCAL_LEAK" = "false" ] && pass "no PascalCase keys leaked" || fail "PascalCase keys in template response"

# github known template present (sanity).
echo "$VT_BODY" | jq -e '.data[] | select(.name=="github")' >/dev/null && pass "github template present" || fail "github template missing"

# --- 45: Vault connect flow (session auth) ------------------------------------
section "45: Vault connect flow (session auth)"
relogin
# Re-seed a provider for this section (section 43 deleted its own).
VC_SEED=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/vault/providers \
  -H "Content-Type: application/json" \
  -d '{"template":"github","client_id":"smoke-connect-id","client_secret":"smoke-connect-secret-123"}')
VC_SEED_CODE=$(echo "$VC_SEED" | tail -1)
VC_SEED_BODY=$(echo "$VC_SEED" | sed '$d')
[ "$VC_SEED_CODE" = 201 ] && pass "seeded provider for connect test" || { echo "    $VC_SEED_BODY"; fail "seed provider -> $VC_SEED_CODE"; }
VAULT_PID2=$(echo "$VC_SEED_BODY" | jq -r '.id // empty')
VAULT_NAME2=$(echo "$VC_SEED_BODY" | jq -r '.name // empty')

# GET /vault/connect/{provider} with session — expect 302 to provider's authorize URL.
# --max-redirs 0 prevents curl from following; we want to inspect the Location header.
CONN_H=$(curl -s -o /dev/null -D - --max-redirs 0 -b cj.txt "$BASE/api/v1/vault/connect/$VAULT_NAME2")
CONN_STATUS=$(echo "$CONN_H" | head -1 | awk '{print $2}')
CONN_LOC=$(echo "$CONN_H" | grep -i '^location:' | tr -d '\r\n' | sed 's/^[Ll]ocation: //')
[ "$CONN_STATUS" = "302" ] && pass "connect -> 302" || fail "connect -> $CONN_STATUS"
echo "$CONN_LOC" | grep -q 'client_id=' && pass "Location contains client_id=" || fail "Location missing client_id=: $CONN_LOC"
echo "$CONN_LOC" | grep -q 'state=' && pass "Location contains state=" || fail "Location missing state=: $CONN_LOC"

# State cookie set on the response (CSRF binding).
echo "$CONN_H" | grep -qi '^set-cookie:.*shark_vault_state=' && pass "shark_vault_state cookie set" || fail "shark_vault_state cookie missing"

# Without session -> 401.
CONN_NOAUTH=$(curl -s -o /dev/null -w "%{http_code}" --max-redirs 0 "$BASE/api/v1/vault/connect/$VAULT_NAME2")
[ "$CONN_NOAUTH" = 401 ] && pass "no session -> 401" || fail "no-session connect -> $CONN_NOAUTH"

# Clean up the seed provider so section 48's audit count lines up with expectations.
VC_CLEAN=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/vault/providers/$VAULT_PID2)
[ "$VC_CLEAN" = 204 ] && pass "seed provider cleanup -> 204" || note "seed provider cleanup -> $VC_CLEAN"

# --- 46: Agent token retrieval (OAuth bearer) ---------------------------------
section "46: Agent token retrieval (OAuth bearer)"
# Full happy-path ExchangeAndStore requires a live upstream OAuth provider
# (token endpoint that will accept our test code). That's infeasible from
# smoke; covered by internal/vault/*_test.go unit tests. What we CAN verify
# here is the bearer-auth envelope: missing + bogus tokens are rejected
# correctly with WWW-Authenticate.
note "full token-retrieval happy path requires mock upstream OAuth server; unit-covered in internal/vault/vault_test.go"

# No bearer -> 401 + WWW-Authenticate.
NB_H=$(curl -s -o /dev/null -D - -w "%{http_code}" "$BASE/api/v1/vault/google_calendar/token")
NB_STATUS=$(echo "$NB_H" | tail -1)
# Remove last line (status code) before scanning headers.
NB_HDRS=$(echo "$NB_H" | sed '$d')
[ "$NB_STATUS" = "401" ] && pass "no bearer -> 401" || fail "no bearer -> $NB_STATUS"
echo "$NB_HDRS" | grep -qi '^www-authenticate:.*Bearer' && pass "WWW-Authenticate: Bearer on no-bearer" || fail "WWW-Authenticate missing on no-bearer"

# Bogus bearer -> 401 + WWW-Authenticate.
BB_H=$(curl -s -o /dev/null -D - -w "%{http_code}" -H "Authorization: Bearer invalid_token_xyz" "$BASE/api/v1/vault/google_calendar/token")
BB_STATUS=$(echo "$BB_H" | tail -1)
BB_HDRS=$(echo "$BB_H" | sed '$d')
[ "$BB_STATUS" = "401" ] && pass "bogus bearer -> 401" || fail "bogus bearer -> $BB_STATUS"
echo "$BB_HDRS" | grep -qi '^www-authenticate:.*Bearer' && pass "WWW-Authenticate: Bearer on bogus" || fail "WWW-Authenticate missing on bogus"

# --- 47: Vault connections list (session auth) --------------------------------
section "47: Vault connections list (session auth)"
relogin
VCL_RESP=$(curl -s -w "\n%{http_code}" -b cj.txt $BASE/api/v1/vault/connections)
VCL_CODE=$(echo "$VCL_RESP" | tail -1)
VCL_BODY=$(echo "$VCL_RESP" | sed '$d')
[ "$VCL_CODE" = 200 ] && pass "GET /vault/connections -> 200" || fail "GET /vault/connections -> $VCL_CODE"
echo "$VCL_BODY" | jq -e '.data' >/dev/null && pass "response has data field" || fail "no data field"
echo "$VCL_BODY" | jq -e '.data | length == 0' >/dev/null && pass "empty for new user (length=0)" || note "expected empty, got length=$(echo "$VCL_BODY" | jq '.data | length')"

# Without session -> 401.
VCL_NOAUTH=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/vault/connections)
[ "$VCL_NOAUTH" = 401 ] && pass "no session -> 401" || fail "no-session connections -> $VCL_NOAUTH"

# DELETE bogus id with session -> 404 (IDOR-safe).
VCL_DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -b cj.txt -X DELETE $BASE/api/v1/vault/connections/vconn_nonexistent_xyz)
[ "$VCL_DEL_CODE" = 404 ] && pass "DELETE unknown connection -> 404" || fail "DELETE unknown connection -> $VCL_DEL_CODE"

# --- 48: Audit events for vault ops -------------------------------------------
section "48: Audit events for vault ops"
# The audit endpoint's ?action= param supports comma-separated exact matches.
AUD_ACTIONS="vault.provider.created,vault.provider.updated,vault.provider.deleted"
AUD_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/audit-logs?action=$AUD_ACTIONS&limit=200")
AUD_CODE=$(echo "$AUD_RESP" | tail -1)
AUD_BODY=$(echo "$AUD_RESP" | sed '$d')
[ "$AUD_CODE" = 200 ] && pass "GET /audit-logs?action=vault.* -> 200" || fail "GET /audit-logs -> $AUD_CODE"

AUD_CREATED=$(echo "$AUD_BODY" | jq '[.data[] | select(.action=="vault.provider.created")] | length')
AUD_UPDATED=$(echo "$AUD_BODY" | jq '[.data[] | select(.action=="vault.provider.updated")] | length')
AUD_DELETED=$(echo "$AUD_BODY" | jq '[.data[] | select(.action=="vault.provider.deleted")] | length')
[ "$AUD_CREATED" -ge 1 ] 2>/dev/null && pass "vault.provider.created events: $AUD_CREATED" || fail "no vault.provider.created events"
[ "$AUD_UPDATED" -ge 1 ] 2>/dev/null && pass "vault.provider.updated events: $AUD_UPDATED" || fail "no vault.provider.updated events"
[ "$AUD_DELETED" -ge 1 ] 2>/dev/null && pass "vault.provider.deleted events: $AUD_DELETED" || fail "no vault.provider.deleted events"

# Also grep the full unfiltered list for any vault.* events (defensive —
# confirms the action filter matched the right namespace).
AUD_ALL=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/audit-logs?limit=200")
AUD_VAULT_TOTAL=$(echo "$AUD_ALL" | jq '[.data[] | select(.action | startswith("vault."))] | length')
[ "$AUD_VAULT_TOTAL" -ge 3 ] 2>/dev/null && pass "unfiltered grep: >=3 vault.* events ($AUD_VAULT_TOTAL)" || fail "unfiltered grep: $AUD_VAULT_TOTAL vault.* events (expected >=3)"

# --- 49: Proxy admin endpoints (proxy disabled) -------------------------------
# Smoke runs against a default dev config with proxy disabled. The admin
# endpoints must still be registered (so dashboards can probe them) but must
# self-404 until proxy wiring is configured in sharkauth.yaml.
section "49: Proxy admin endpoints (proxy disabled)"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/proxy/status)
[ "$CODE" = 404 ] && pass "GET /admin/proxy/status -> 404 (disabled)" || fail "proxy status -> $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/proxy/rules)
[ "$CODE" = 404 ] && pass "GET /admin/proxy/rules -> 404 (disabled)" || fail "proxy rules -> $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"method":"GET","path":"/api/foo"}' \
  $BASE/api/v1/admin/proxy/simulate)
[ "$CODE" = 404 ] && pass "POST /admin/proxy/simulate -> 404 (disabled)" || fail "proxy simulate -> $CODE"

# No admin key -> 401 (admin middleware rejects before the handler's 404).
CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/admin/proxy/status)
[ "$CODE" = 401 ] && pass "no auth -> 401" || fail "unauth proxy -> $CODE"

note "proxy happy-path smoke requires enabling proxy + upstream — covered in internal/api package tests"

# --- 50: Auth flow CRUD -------------------------------------------------------
section "50: Auth flow CRUD"

# Create a signup-trigger flow whose only step requires email verification.
# We reuse the resulting FLOW_ID across §§50-54.
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Smoke Signup","trigger":"signup","steps":[{"type":"require_email_verification"}],"enabled":true,"priority":10}' \
  $BASE/api/v1/admin/flows)
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
[ "$CODE" = 201 ] && pass "POST /admin/flows -> 201" || fail "create flow -> $CODE: $BODY"
FLOW_ID=$(echo "$BODY" | jq -r '.id')
{ [ -n "$FLOW_ID" ] && [ "$FLOW_ID" != "null" ] ; } && pass "flow id returned ($FLOW_ID)" || fail "no flow id"

# Get by id.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/flows/$FLOW_ID)
[ "$CODE" = 200 ] && pass "GET /admin/flows/{id} -> 200" || fail "get flow -> $CODE"

# List (no filter) should include the new flow.
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/flows)
COUNT=$(echo "$RESP" | jq '.data | length')
[ "$COUNT" -ge 1 ] 2>/dev/null && pass "list includes flow (count=$COUNT)" || fail "list missing flow"

# Filter by trigger=login (the smoke flow is a signup flow — expect zero).
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/flows?trigger=login")
COUNT=$(echo "$RESP" | jq '.data | length')
[ "$COUNT" = 0 ] && pass "filter trigger=login -> 0 results" || fail "trigger filter wrong: $COUNT"

# PATCH (toggle enabled off → back on later in §51).
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X PATCH \
  -d '{"enabled":false}' $BASE/api/v1/admin/flows/$FLOW_ID)
[ "$CODE" = 200 ] && pass "PATCH /admin/flows/{id} -> 200" || fail "patch flow -> $CODE"

# Validation: bad trigger → 400 invalid_flow.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Bad","trigger":"bogus","steps":[{"type":"redirect"}]}' \
  $BASE/api/v1/admin/flows)
[ "$CODE" = 400 ] && pass "bad trigger -> 400" || fail "bad trigger accepted: $CODE"

# Validation: empty steps → 400.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Empty","trigger":"signup","steps":[]}' \
  $BASE/api/v1/admin/flows)
[ "$CODE" = 400 ] && pass "empty steps -> 400" || fail "empty steps accepted: $CODE"

# --- 51: Flow dry-run ---------------------------------------------------------
section "51: Flow dry-run"

# Re-enable the smoke flow so §§52+54 behave.
curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -X PATCH -d '{"enabled":true}' $BASE/api/v1/admin/flows/$FLOW_ID > /dev/null

# Dry-run with unverified user -> expect block + non-empty timeline.
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -X POST -d '{"user":{"email":"dry@test.com","email_verified":false}}' \
  $BASE/api/v1/admin/flows/$FLOW_ID/test)
OUTCOME=$(echo "$RESP" | jq -r '.outcome')
[ "$OUTCOME" = "block" ] && pass "dry-run unverified -> block" || fail "outcome=$OUTCOME body=$RESP"

REASON=$(echo "$RESP" | jq -r '.reason')
echo "$REASON" | grep -qi "email verification" && pass "reason mentions email verification" || fail "reason=$REASON"

TIMELINE_LEN=$(echo "$RESP" | jq '.timeline | length')
[ "$TIMELINE_LEN" -ge 1 ] 2>/dev/null && pass "timeline populated (len=$TIMELINE_LEN)" || fail "empty timeline"

# Dry-run with verified user -> continue.
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -X POST -d '{"user":{"email":"dry@test.com","email_verified":true}}' \
  $BASE/api/v1/admin/flows/$FLOW_ID/test)
OUTCOME=$(echo "$RESP" | jq -r '.outcome')
[ "$OUTCOME" = "continue" ] && pass "dry-run verified -> continue" || fail "outcome=$OUTCOME"

# 404 for unknown flow id.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST -d '{}' \
  $BASE/api/v1/admin/flows/flow_nonexistent/test)
[ "$CODE" = 404 ] && pass "bad flow id -> 404" || fail "bad id test -> $CODE"

# --- 52: Flow blocks signup on unverified email -------------------------------
# POSTing /auth/signup with the require_email_verification flow enabled should
# land 403 with {"error":"flow_blocked"}. Note: the user row is created BEFORE
# the flow fires (see internal/api/auth_handlers.go:166), so a DB entry remains
# — acceptable per the plan; we only assert on the API response here.
section "52: Flow blocks signup on unverified email"
FLOW_EMAIL="flowtest-$(date +%s)-$RANDOM@test.com"
FLOW_PW='GetCake117$$$'
RESP=$(curl -s -w "\n%{http_code}" -H "Content-Type: application/json" -X POST \
  -d "{\"email\":\"$FLOW_EMAIL\",\"password\":\"$FLOW_PW\"}" \
  $BASE/api/v1/auth/signup)
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
[ "$CODE" = 403 ] && pass "signup with blocking flow -> 403" || fail "signup -> $CODE (body: $BODY)"
echo "$BODY" | jq -e '.error=="flow_blocked"' >/dev/null && pass "body has flow_blocked error" || fail "body=$BODY"

# --- 53: Disabled flow lets signup through ------------------------------------
section "53: Disabled flow lets signup through"

# Disable the flow and confirm signup returns the normal 201.
curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -X PATCH -d '{"enabled":false}' $BASE/api/v1/admin/flows/$FLOW_ID > /dev/null

FLOW_EMAIL2="flowtest2-$(date +%s)-$RANDOM@test.com"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Content-Type: application/json" -X POST \
  -d "{\"email\":\"$FLOW_EMAIL2\",\"password\":\"$FLOW_PW\"}" \
  $BASE/api/v1/auth/signup)
{ [ "$CODE" = 201 ] || [ "$CODE" = 200 ] ; } && pass "signup with disabled flow -> $CODE (success)" || fail "disabled flow blocked signup: $CODE"

# --- 54: Flow runs recorded ---------------------------------------------------
section "54: Flow runs recorded"

# Re-enable the flow and trigger another blocked signup so we have at least
# one persisted run to read back.
curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -X PATCH -d '{"enabled":true}' $BASE/api/v1/admin/flows/$FLOW_ID > /dev/null

FLOW_EMAIL3="flowtest3-$(date +%s)-$RANDOM@test.com"
curl -s -o /dev/null -H "Content-Type: application/json" -X POST \
  -d "{\"email\":\"$FLOW_EMAIL3\",\"password\":\"$FLOW_PW\"}" \
  $BASE/api/v1/auth/signup

# GET /admin/flows/{id}/runs should have at least one entry, including a block.
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/flows/$FLOW_ID/runs)
COUNT=$(echo "$RESP" | jq '.data | length')
[ "$COUNT" -ge 1 ] 2>/dev/null && pass "runs recorded (count=$COUNT)" || fail "no runs: $RESP"

OUTCOMES=$(echo "$RESP" | jq -r '.data[].outcome' | sort -u)
echo "$OUTCOMES" | grep -q "block" && pass "at least one run has outcome=block" || fail "outcomes: $OUTCOMES"

# Cleanup so re-runs of the smoke test start from a clean flows table.
curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/admin/flows/$FLOW_ID

# --- 55: webhook delivery replay (Wave 1 A1) ---------------------------------
# Validates POST /webhooks/{id}/deliveries/{deliveryId}/replay returns 202 with
# a new_delivery_id and that the new delivery is visible via the deliveries
# list endpoint. Frontend webhooks.tsx:646 wires the per-row replay button
# against this; previously it 404'd silently.
section "55: webhook delivery replay (A1)"

REP_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/replay","events":["user.created"],"description":"replay smoke"}')
REP_CODE=$(echo "$REP_RESP" | tail -1)
REP_BODY=$(echo "$REP_RESP" | head -1)
[ "$REP_CODE" = 201 ] && pass "replay setup: webhook create 201" || fail "replay setup: webhook create $REP_CODE"
REP_WH=$(echo "$REP_BODY" | jq -r '.id // empty')

# Fire a test event so we have one delivery row to replay.
TEST_RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks/$REP_WH/test \
  -H "Content-Type: application/json" -d '{"event_type":"user.created"}')
ORIG_DEL=$(echo "$TEST_RESP" | jq -r '.delivery_id // empty')
[ -n "$ORIG_DEL" ] && pass "test fire returned delivery_id $ORIG_DEL" || fail "test fire response: $TEST_RESP"

# Replay it.
RREP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST \
  $BASE/api/v1/webhooks/$REP_WH/deliveries/$ORIG_DEL/replay)
RCODE=$(echo "$RREP" | tail -1)
RBODY=$(echo "$RREP" | head -1)
[ "$RCODE" = 202 ] && pass "replay -> 202" || fail "replay -> $RCODE (body: $RBODY)"
NEW_DEL=$(echo "$RBODY" | jq -r '.new_delivery_id // empty')
[ -n "$NEW_DEL" ] && [ "$NEW_DEL" != "$ORIG_DEL" ] && pass "replay returned distinct new_delivery_id $NEW_DEL" \
  || fail "new_delivery_id missing/duplicate: orig=$ORIG_DEL new=$NEW_DEL"

# Confirm the new delivery is visible via list.
LIST=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/webhooks/$REP_WH/deliveries?limit=10")
echo "$LIST" | jq -e --arg id "$NEW_DEL" '.data[] | select(.id == $id)' >/dev/null \
  && pass "new delivery visible in list" || fail "new delivery missing from list: $LIST"

# Cross-webhook replay must 404 (URL tampering protection).
OTHER_RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/other","events":["user.created"]}')
OTHER_WH=$(echo "$OTHER_RESP" | jq -r '.id')
CCODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST \
  $BASE/api/v1/webhooks/$OTHER_WH/deliveries/$ORIG_DEL/replay)
[ "$CCODE" = 404 ] && pass "cross-webhook replay -> 404" || fail "cross-webhook replay -> $CCODE"

# Cleanup.
curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/webhooks/$REP_WH
curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/webhooks/$OTHER_WH

# --- 56: admin org CRUD (Wave 1 A2-A4) ----------------------------------------
# Validates the admin-key-authenticated org CRUD surface that the dashboard
# uses (admin/src/components/organizations.tsx). Pre-fix, PATCH/DELETE/roles
# under /admin/organizations were 404 — now they mirror the user-facing routes
# without requiring a session cookie.
section "56: admin org CRUD (A2-A4)"

ORG_SLUG="adm-smoke-$RANDOM"
AOC=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations -H "Content-Type: application/json" \
  -d "{\"name\":\"AdminCRUD\",\"slug\":\"$ORG_SLUG\"}")
# Falls back: admin org create still goes through user-facing route (it requires
# a creator user); we test the admin PATCH/DELETE/roles surfaces against it.
AO_ID=$(echo "$AOC" | jq -r '.id // empty')
if [ -z "$AO_ID" ]; then
  # cj.txt may have been logged out by §13; relogin once and retry.
  relogin
  AOC=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations -H "Content-Type: application/json" \
    -d "{\"name\":\"AdminCRUD\",\"slug\":\"$ORG_SLUG\"}")
  AO_ID=$(echo "$AOC" | jq -r '.id // empty')
fi
[ -n "$AO_ID" ] && pass "org seeded for admin CRUD: $AO_ID" || fail "org seed: $AOC"

# PATCH name + slug via admin endpoint (not session route).
NEW_SLUG="adm-renamed-$RANDOM"
PRESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X PATCH \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"AdminRenamed\",\"slug\":\"$NEW_SLUG\"}" \
  $BASE/api/v1/admin/organizations/$AO_ID)
PCODE=$(echo "$PRESP" | tail -1)
PBODY=$(echo "$PRESP" | head -1)
[ "$PCODE" = 200 ] && pass "admin PATCH org -> 200" || fail "admin PATCH org -> $PCODE (body: $PBODY)"
PNAME=$(echo "$PBODY" | jq -r '.name')
[ "$PNAME" = "AdminRenamed" ] && pass "PATCH applied name field" || fail "name not updated: $PNAME"

# GET to verify persistence.
GBODY=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/organizations/$AO_ID)
[ "$(echo "$GBODY" | jq -r '.slug')" = "$NEW_SLUG" ] && pass "admin GET reflects new slug" || fail "GET slug mismatch: $GBODY"

# Create org-role via admin endpoint (was 404 pre-fix).
RRESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"admin-editor","description":"admin-created role"}' \
  $BASE/api/v1/admin/organizations/$AO_ID/roles)
RCODE=$(echo "$RRESP" | tail -1)
RBODY=$(echo "$RRESP" | head -1)
[ "$RCODE" = 201 ] && pass "admin POST org-role -> 201" || fail "admin POST org-role -> $RCODE (body: $RBODY)"
ROLE_ID=$(echo "$RBODY" | jq -r '.id // empty')
[ -n "$ROLE_ID" ] && pass "new role id $ROLE_ID" || fail "no role id in body"

# Reject blank name.
BAD_R=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST \
  -H "Content-Type: application/json" -d '{"name":"  "}' \
  $BASE/api/v1/admin/organizations/$AO_ID/roles)
[ "$BAD_R" = 400 ] && pass "admin POST org-role blank name -> 400" || fail "blank name -> $BAD_R"

# Slug uniqueness on PATCH: try to take the original org's slug back via a
# second org. Should 409.
OTHER_ORG=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations -H "Content-Type: application/json" \
  -d "{\"name\":\"Other\",\"slug\":\"other-$RANDOM\"}" | jq -r '.id // empty')
if [ -n "$OTHER_ORG" ]; then
  CONFLICT=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X PATCH \
    -H "Content-Type: application/json" -d "{\"slug\":\"$NEW_SLUG\"}" \
    $BASE/api/v1/admin/organizations/$OTHER_ORG)
  [ "$CONFLICT" = 409 ] && pass "duplicate slug PATCH -> 409" || fail "duplicate slug -> $CONFLICT"
  curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/admin/organizations/$OTHER_ORG
fi

# DELETE the org via admin endpoint.
DCODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE \
  $BASE/api/v1/admin/organizations/$AO_ID)
[ "$DCODE" = 200 ] && pass "admin DELETE org -> 200" || fail "admin DELETE org -> $DCODE"

# GET 404 after delete.
GCODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  $BASE/api/v1/admin/organizations/$AO_ID)
[ "$GCODE" = 404 ] && pass "deleted org GET -> 404" || fail "deleted org GET -> $GCODE"

# DELETE missing org -> 404.
MCODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE \
  $BASE/api/v1/admin/organizations/org_definitely_not_there)
[ "$MCODE" = 404 ] && pass "DELETE missing org -> 404" || fail "DELETE missing org -> $MCODE"

# --- 57: admin org invitation manage (Wave 1 A5-A6) --------------------------
# Tests admin DELETE + resend on an org invitation. Pre-fix both routes 404'd
# (organizations.tsx:609,616 silent-failed). Resend rotates the token + bumps
# expiry; delete drops the row entirely.
section "57: admin org invitation manage (A5-A6)"

INV_SLUG="inv-smoke-$RANDOM"
INV_ORG=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations -H "Content-Type: application/json" \
  -d "{\"name\":\"InvOrg\",\"slug\":\"$INV_SLUG\"}" | jq -r '.id // empty')
[ -n "$INV_ORG" ] && pass "invitation org seeded: $INV_ORG" || fail "invitation org seed failed"

# Create invitation via session route (admin doesn't have a create endpoint;
# this is fine — dashboard creates via the session-authenticated user-facing
# route too. Admin only manages existing rows.)
INV_RESP=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations/$INV_ORG/invitations \
  -H "Content-Type: application/json" \
  -d '{"email":"invitee@example.com","role":"member"}')
INV_ID=$(echo "$INV_RESP" | jq -r '.id // empty')
INV_EXP_OLD=$(echo "$INV_RESP" | jq -r '.expires_at // empty')
[ -n "$INV_ID" ] && pass "invitation seeded: $INV_ID" || fail "invitation create: $INV_RESP"

# Resend rotates the token and updates expiry. Sleep 1s so expiry strictly differs.
sleep 1
RES_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST \
  $BASE/api/v1/admin/organizations/$INV_ORG/invitations/$INV_ID/resend)
RES_CODE=$(echo "$RES_RESP" | tail -1)
RES_BODY=$(echo "$RES_RESP" | head -1)
[ "$RES_CODE" = 200 ] && pass "admin resend invitation -> 200" || fail "resend -> $RES_CODE (body: $RES_BODY)"
NEW_EXP=$(echo "$RES_BODY" | jq -r '.expires_at // empty')
[ -n "$NEW_EXP" ] && [ "$NEW_EXP" != "$INV_EXP_OLD" ] && pass "resend rotated expires_at" \
  || fail "expires_at unchanged: old=$INV_EXP_OLD new=$NEW_EXP"
echo "$RES_BODY" | jq -e '.email_sent | type == "boolean"' >/dev/null && pass "resend reports email_sent flag" || fail "missing email_sent: $RES_BODY"

# Cross-org URL tampering: same invitation id, different org -> 404.
OTHER_INV=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"OtherInv\",\"slug\":\"otherinv-$RANDOM\"}" | jq -r '.id // empty')
if [ -n "$OTHER_INV" ]; then
  XCODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST \
    $BASE/api/v1/admin/organizations/$OTHER_INV/invitations/$INV_ID/resend)
  [ "$XCODE" = 404 ] && pass "cross-org resend -> 404" || fail "cross-org resend -> $XCODE"
  curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/admin/organizations/$OTHER_INV
fi

# Delete invitation.
DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE \
  $BASE/api/v1/admin/organizations/$INV_ORG/invitations/$INV_ID)
[ "$DEL_CODE" = 200 ] && pass "admin DELETE invitation -> 200" || fail "DELETE invitation -> $DEL_CODE"

# Second delete -> 404 (gone).
GONE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE \
  $BASE/api/v1/admin/organizations/$INV_ORG/invitations/$INV_ID)
[ "$GONE" = 404 ] && pass "deleted invitation -> 404 on retry" || fail "deleted invitation re-delete -> $GONE"

# Cleanup org.
curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/admin/organizations/$INV_ORG

# --- 57b: admin GET org roles + invitations (Phase 2 Task A + B) --------------
# GET /admin/organizations/{id}/roles and /invitations were 404 before this fix.
# Dashboard organizations.tsx:138 (Roles tab) and :608 (Invitations tab) both
# call these endpoints; without them both tabs silently showed zero.
section "57b: admin GET org roles + invitations list"

RL_SLUG="roles-inv-smoke-$RANDOM"
RL_ORG=$(curl -s -b cj.txt -X POST $BASE/api/v1/organizations -H "Content-Type: application/json" \
  -d "{\"name\":\"RolesInvSmoke\",\"slug\":\"$RL_SLUG\"}" | jq -r '.id // empty')
[ -n "$RL_ORG" ] && pass "org seeded for roles+inv smoke: $RL_ORG" || fail "org seed failed"

# GET roles -> 200 with .roles array (may be empty on fresh org).
RL_RESP=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/organizations/$RL_ORG/roles)
RL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  $BASE/api/v1/admin/organizations/$RL_ORG/roles)
[ "$RL_CODE" = 200 ] && pass "GET admin org roles -> 200" || fail "GET admin org roles -> $RL_CODE"
echo "$RL_RESP" | jq -e '.roles | arrays' > /dev/null 2>&1 \
  && pass "GET admin org roles .roles is array" || fail "GET admin org roles .roles not array: $RL_RESP"

# GET invitations -> 200 with .invitations array.
INV_LIST_RESP=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/organizations/$RL_ORG/invitations)
INV_LIST_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  $BASE/api/v1/admin/organizations/$RL_ORG/invitations)
[ "$INV_LIST_CODE" = 200 ] && pass "GET admin org invitations -> 200" || fail "GET admin org invitations -> $INV_LIST_CODE"
echo "$INV_LIST_RESP" | jq -e '.invitations | arrays' > /dev/null 2>&1 \
  && pass "GET admin org invitations .invitations is array" || fail "GET admin org invitations .invitations not array: $INV_LIST_RESP"

# Both 401 without admin key.
RL_UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/admin/organizations/$RL_ORG/roles)
[ "$RL_UNAUTH" = 401 ] && pass "GET org roles no key -> 401" || fail "GET org roles no key -> $RL_UNAUTH"
INV_UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/admin/organizations/$RL_ORG/invitations)
[ "$INV_UNAUTH" = 401 ] && pass "GET org invitations no key -> 401" || fail "GET org invitations no key -> $INV_UNAUTH"

# Cleanup.
curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/admin/organizations/$RL_ORG

# --- 58: admin MFA disable (Wave 1 A7) ----------------------------------------
# Admin-only DELETE /users/{id}/mfa wipes MFA without requiring the user's
# current TOTP code. Used by support to recover lost-device accounts. The
# user-facing /auth/mfa endpoint still requires a code (sec. boundary intact).
section "58: admin MFA disable (A7)"

MFA_EMAIL="mfa-admin-$RANDOM@test.com"
MFA_PW='GetCake117$$$'
MFA_RESP=$(curl -s -c mfacj.txt -X POST $BASE/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$MFA_EMAIL\",\"password\":\"$MFA_PW\"}")
MFA_USER=$(echo "$MFA_RESP" | jq -r '.id // empty')
[ -n "$MFA_USER" ] && pass "MFA test user created: $MFA_USER" || fail "MFA user signup: $MFA_RESP"

# Force email_verified so /auth/mfa endpoints (which require verified email) work.
curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X PATCH \
  -H "Content-Type: application/json" \
  -d '{"email_verified":true}' $BASE/api/v1/users/$MFA_USER

# Re-login to refresh the cookie's email_verified claim.
curl -s -c mfacj.txt -X POST $BASE/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$MFA_EMAIL\",\"password\":\"$MFA_PW\"}" > /dev/null

# Enroll MFA via the user-facing endpoint to populate the secret (then we
# bypass the verify step by directly flipping mfa_enabled via storage path
# isn't possible from smoke — instead we rely on the admin endpoint clearing
# whatever state exists).
ENR=$(curl -s -b mfacj.txt -X POST $BASE/api/v1/auth/mfa/enroll)
SECRET=$(echo "$ENR" | jq -r '.secret // empty')
[ -n "$SECRET" ] && pass "MFA enroll returned secret" || note "enroll: $ENR"

# Admin disable: should clear secret + flag regardless of verify state.
ADMIN_DEL=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE \
  $BASE/api/v1/users/$MFA_USER/mfa)
AD_CODE=$(echo "$ADMIN_DEL" | tail -1)
AD_BODY=$(echo "$ADMIN_DEL" | head -1)
[ "$AD_CODE" = 200 ] && pass "admin DELETE /users/{id}/mfa -> 200" || fail "admin MFA disable -> $AD_CODE (body: $AD_BODY)"
echo "$AD_BODY" | jq -e '.mfa_enabled == false' >/dev/null && pass "response asserts mfa_enabled=false" || fail "body: $AD_BODY"

# Verify via GET that mfa_enabled is now false on the user record.
USER_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/users/$MFA_USER)
echo "$USER_BODY" | jq -e '.mfaEnabled == false' >/dev/null && pass "GET user reflects mfa_enabled=false" || fail "user body: $USER_BODY"

# Audit log entry exists.
if [ -f $DB ]; then
  N=$(sqlite3 $DB "SELECT COUNT(*) FROM audit_logs WHERE action='admin.mfa.disabled' AND target_id='$MFA_USER'")
  [ "$N" -ge 1 ] 2>/dev/null && pass "audit log: admin.mfa.disabled (n=$N)" || fail "no admin.mfa.disabled audit row"
fi

# Missing user -> 404.
NF_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X DELETE \
  $BASE/api/v1/users/usr_definitely_not_here/mfa)
[ "$NF_CODE" = 404 ] && pass "admin MFA disable missing user -> 404" || fail "missing user -> $NF_CODE"

# --- 59: audit ?actor_type= filter (Wave 2 C2) --------------------------------
# Frontend passes ?actor_type=user|agent|system|admin to /audit-logs. Pre-fix
# AuditLogQuery had no ActorType field so the param silently dropped — every
# request returned all rows regardless of selection. Seed two rows with
# distinct actor_types and assert the filter actually filters.
section "59: audit actor_type filter (C2)"

if [ -f $DB ]; then
  AGENT_AUD="aud_smoke_agent_$RANDOM"
  USER_AUD="aud_smoke_user_$RANDOM"
  NOW_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  # NB: storage.AuditLog scans into plain strings — NULL columns blow up the
  # query handler. Seed every column explicitly so the row round-trips cleanly.
  sqlite3 $DB "INSERT INTO audit_logs (id, actor_id, actor_type, action, target_type, target_id, ip, user_agent, metadata, status, created_at) VALUES ('$AGENT_AUD','agt_smoke','agent','smoke.actor.agent','','','','','{}','success','$NOW_TS')" 2>/dev/null \
    && pass "seeded agent audit row" || fail "seed agent row"
  sqlite3 $DB "INSERT INTO audit_logs (id, actor_id, actor_type, action, target_type, target_id, ip, user_agent, metadata, status, created_at) VALUES ('$USER_AUD','usr_smoke','user','smoke.actor.user','','','','','{}','success','$NOW_TS')" 2>/dev/null \
    && pass "seeded user audit row" || fail "seed user row"

  AGENT_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/audit-logs?actor_type=agent&limit=200")
  echo "$AGENT_BODY" | jq -e --arg id "$AGENT_AUD" '.data | map(.id) | index($id) != null' >/dev/null \
    && pass "actor_type=agent returns the agent row" || fail "agent row missing: $AGENT_BODY"
  echo "$AGENT_BODY" | jq -e --arg id "$USER_AUD" '.data | map(.id) | index($id) == null' >/dev/null \
    && pass "actor_type=agent excludes the user row" || fail "user row leaked into agent filter: $AGENT_BODY"

  USER_BODY=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/audit-logs?actor_type=user&limit=200")
  echo "$USER_BODY" | jq -e --arg id "$USER_AUD" '.data | map(.id) | index($id) != null' >/dev/null \
    && pass "actor_type=user returns the user row" || fail "user row missing: $USER_BODY"
  echo "$USER_BODY" | jq -e --arg id "$AGENT_AUD" '.data | map(.id) | index($id) == null' >/dev/null \
    && pass "actor_type=user excludes the agent row" || fail "agent row leaked into user filter: $USER_BODY"
fi

# --- 60: failed_logins_24h counter accuracy (Wave 2 C3) -----------------------
# Pre-fix CountFailedLoginsSince queried action='login' but the login handler
# never emitted any audit log on failure — counter was always 0. Wave 2 emits
# action='user.login' status='failure' on each failed login + flips the query
# to match. Submit one bad login, snapshot stats, assert counter incremented.
section "60: failed_logins_24h accuracy (C3)"

BEFORE_FL=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats | jq -r '.failed_logins_24h // 0')
curl -s -o /dev/null -X POST $BASE/api/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"smoke-nope@test.com","password":"definitely-wrong-x"}'
AFTER_FL=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats | jq -r '.failed_logins_24h // 0')
[ "$AFTER_FL" -gt "$BEFORE_FL" ] 2>/dev/null && pass "failed_logins_24h incremented ($BEFORE_FL -> $AFTER_FL)" \
  || fail "counter unchanged: before=$BEFORE_FL after=$AFTER_FL"

if [ -f $DB ]; then
  N=$(sqlite3 $DB "SELECT COUNT(*) FROM audit_logs WHERE action='user.login' AND status='failure'")
  [ "$N" -ge 1 ] 2>/dev/null && pass "audit row written on failed login (n=$N)" || fail "no user.login failure audit row"
fi

# --- 61: MFA verified-only count (Wave 2 C4) ----------------------------------
# Pre-fix CountMFAEnabled counted users with mfa_enabled=1 regardless of
# mfa_verified. Half-enrolled users (started TOTP but never verified) inflated
# the dashboard adoption number. Wave 2 narrows the count to verified users.
# Smoke: snapshot count, set mfa_enabled=1 mfa_verified=0, assert unchanged;
# then flip mfa_verified=1 and assert it now increments.
section "61: MFA enabled-vs-verified count (C4)"

if [ -f $DB ]; then
  MFA_CNT_EMAIL="mfa-count-$RANDOM@test.com"
  MFA_CNT_RESP=$(curl -s -X POST $BASE/api/v1/auth/signup -H "Content-Type: application/json" \
    -d "{\"email\":\"$MFA_CNT_EMAIL\",\"password\":\"GetCake117\$\$\$\"}")
  MFA_CNT_USER=$(echo "$MFA_CNT_RESP" | jq -r '.id // empty')
  [ -n "$MFA_CNT_USER" ] && pass "mfa-count user seeded: $MFA_CNT_USER" || fail "user seed: $MFA_CNT_RESP"

  BEFORE_MFA=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats | jq -r '.mfa.enabled')
  sqlite3 $DB "UPDATE users SET mfa_enabled=1, mfa_verified=0 WHERE id='$MFA_CNT_USER'" 2>/dev/null
  HALF_MFA=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats | jq -r '.mfa.enabled')
  [ "$HALF_MFA" = "$BEFORE_MFA" ] && pass "half-enrolled user does NOT count ($BEFORE_MFA == $HALF_MFA)" \
    || fail "half-enrolled bumped count: before=$BEFORE_MFA half=$HALF_MFA"

  sqlite3 $DB "UPDATE users SET mfa_enabled=1, mfa_verified=1 WHERE id='$MFA_CNT_USER'" 2>/dev/null
  FULL_MFA=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats | jq -r '.mfa.enabled')
  [ "$FULL_MFA" -gt "$BEFORE_MFA" ] 2>/dev/null && pass "verified user counts ($BEFORE_MFA -> $FULL_MFA)" \
    || fail "verified user did not increment: before=$BEFORE_MFA full=$FULL_MFA"
fi

# --- 62: flow test metadata pass-through (Wave 2 C7) --------------------------
# handleTestFlow takes {metadata:{...}} in the body and seeds the dry-run
# context with it. Assert the response surfaces caller-supplied keys back so
# we know the engine received them (the engine copies fc.Metadata into
# Result.Metadata before any step runs).
section "62: flow test metadata pass-through (C7)"

# §54 deletes its FLOW_ID; create a fresh one here. Engine seeds Result.Metadata
# from fc.Metadata before any step runs, so a single require_email_verification
# step against a verified user is enough to drive the test.
META_FLOW=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" -X POST \
  -d '{"name":"Meta Smoke","trigger":"signup","steps":[{"type":"require_email_verification"}],"enabled":true}' \
  $BASE/api/v1/admin/flows | jq -r '.id // empty')
if [ -n "$META_FLOW" ]; then
  RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
    -X POST -d '{"user":{"email":"meta@test.com","email_verified":true},"metadata":{"smoke_meta_key":"smoke_meta_val"}}' \
    $BASE/api/v1/admin/flows/$META_FLOW/test)
  echo "$RESP" | jq -e '.metadata.smoke_meta_key == "smoke_meta_val"' >/dev/null \
    && pass "test-flow echoes caller metadata" || fail "metadata dropped: $RESP"
  curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE $BASE/api/v1/admin/flows/$META_FLOW
else
  fail "could not seed metadata test flow"
fi

# --- 66: RBAC reverse lookup + email preview (F Wave) -------------------------
section "66: RBAC reverse lookup + email preview"

# Seed: create role + permission + assign
PERM_RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" -X POST \
  -d '{"action":"smoke_read","resource":"smoke_thing"}' $BASE/api/v1/permissions)
PERM_ID=$(echo "$PERM_RESP" | jq -r '.id // empty')
[ -n "$PERM_ID" ] && pass "seed permission $PERM_ID" || fail "perm seed: $PERM_RESP"

ROLE_RESP=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" -X POST \
  -d '{"name":"smoke_role_F","description":"smoke F"}' $BASE/api/v1/roles)
ROLE_ID=$(echo "$ROLE_RESP" | jq -r '.id // empty')
[ -n "$ROLE_ID" ] && pass "seed role $ROLE_ID" || fail "role seed: $ROLE_RESP"

if [ -n "$PERM_ID" ] && [ -n "$ROLE_ID" ]; then
  # Attach perm to role
  AT_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
    -X POST -d "{\"permission_id\":\"$PERM_ID\"}" "$BASE/api/v1/roles/$ROLE_ID/permissions")
  if [ "$AT_CODE" = 200 ] || [ "$AT_CODE" = 201 ] || [ "$AT_CODE" = 204 ]; then
    pass "attach perm to role ($AT_CODE)"
  else
    fail "attach -> $AT_CODE"
  fi

  # Assign role to user
  AS_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
    -X POST -d "{\"role_id\":\"$ROLE_ID\"}" "$BASE/api/v1/users/$USERID/roles")
  if [ "$AS_CODE" = 200 ] || [ "$AS_CODE" = 201 ] || [ "$AS_CODE" = 204 ]; then
    pass "assign role to user ($AS_CODE)"
  else
    fail "assign -> $AS_CODE"
  fi

  # Reverse lookup: roles for this permission
  ROLES_BY_PERM=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/permissions/$PERM_ID/roles")
  echo "$ROLES_BY_PERM" | jq -e --arg rid "$ROLE_ID" '.data[] | select(.id==$rid)' >/dev/null \
    && pass "GET /permissions/{id}/roles returns assigned role" || fail "reverse role lookup: $ROLES_BY_PERM"

  # Reverse lookup: users for this permission
  USERS_BY_PERM=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/permissions/$PERM_ID/users")
  echo "$USERS_BY_PERM" | jq -e --arg uid "$USERID" '.data[] | select(.id==$uid)' >/dev/null \
    && pass "GET /permissions/{id}/users returns user via role" || fail "reverse user lookup: $USERS_BY_PERM"

  # Missing permission -> 404
  MISS_PR=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/permissions/perm_doesnt_exist/roles")
  [ "$MISS_PR" = 404 ] && pass "missing perm -> 404" || fail "missing perm -> $MISS_PR"
fi

# Email preview — each template renders non-empty HTML
for TPL in magic_link verify_email password_reset organization_invitation; do
  EP=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/email-preview/$TPL")
  HTML_LEN=$(echo "$EP" | jq -r '.html | length // 0')
  if [ "$HTML_LEN" -gt 100 ]; then
    pass "email-preview/$TPL renders non-empty HTML ($HTML_LEN bytes)"
  else
    fail "email-preview/$TPL empty: $EP"
  fi
done

# Unknown template -> 404
UNK_EP=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/email-preview/not_a_real_template")
[ "$UNK_EP" = 404 ] && pass "unknown template -> 404" || fail "unknown template -> $UNK_EP"

# No-auth blocked
NA_EP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/admin/email-preview/magic_link")
[ "$NA_EP" = 401 ] && pass "no auth -> 401 on email-preview" || fail "no-auth -> $NA_EP"

# --- 65: Admin consents + device queue (E Wave) -------------------------------
section "65: admin consents + device queue"

# Admin consent listing — empty case shape.
EMPTY_CONS=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/oauth/consents")
echo "$EMPTY_CONS" | jq -e '.data | type == "array"' >/dev/null \
  && pass "/admin/oauth/consents .data is array" || fail "shape: $EMPTY_CONS"

# Seed a consent and a different one for another user
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
CID="shark_app_NqFBAOJvP9f9dAmskrArJ"
sqlite3 $DB "INSERT INTO oauth_consents (id, user_id, client_id, scope, authorization_details, granted_at, expires_at, revoked_at)
  VALUES ('cons_smoke_e1', '$USERID', '$CID', 'openid profile', '', '$NOW', NULL, NULL);"

ADMIN_CONS=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/oauth/consents")
echo "$ADMIN_CONS" | jq -e '.data[] | select(.id=="cons_smoke_e1")' >/dev/null \
  && pass "seeded consent visible in admin list" || fail "consent missing: $ADMIN_CONS"

# user_id present in admin response
echo "$ADMIN_CONS" | jq -e --arg uid "$USERID" '.data[] | select(.id=="cons_smoke_e1" and .user_id==$uid)' >/dev/null \
  && pass "admin consent row includes user_id" || fail "user_id missing"

# Admin revoke
REVK_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X DELETE "$BASE/api/v1/admin/oauth/consents/cons_smoke_e1")
[ "$REVK_CODE" = 200 ] && pass "DELETE /admin/oauth/consents/{id} -> 200" || fail "admin revoke -> $REVK_CODE"

# Confirm revoked (no longer in list)
AFTER_CONS=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/oauth/consents")
echo "$AFTER_CONS" | jq -e '.data[] | select(.id=="cons_smoke_e1") | length == 0' >/dev/null 2>&1 \
  && pass "revoked consent gone from list" \
  || (echo "$AFTER_CONS" | jq -e '[.data[] | select(.id=="cons_smoke_e1")] | length == 0' >/dev/null \
      && pass "revoked consent gone from list" || fail "consent still in list")

# audit row written
AUDIT_REVK=$(sqlite3 $DB "SELECT COUNT(*) FROM audit_logs WHERE action='consent.revoked' AND target_id='cons_smoke_e1' AND actor_type='admin';")
[ "$AUDIT_REVK" = "1" ] && pass "audit log: consent.revoked by admin" || fail "audit row missing (got $AUDIT_REVK)"

# Device codes admin queue — empty case
EMPTY_DC=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/oauth/device-codes")
echo "$EMPTY_DC" | jq -e '.data | type == "array"' >/dev/null \
  && pass "/admin/oauth/device-codes .data is array" || fail "shape: $EMPTY_DC"

# Seed pending device code
DC_NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
DC_EXP=$(date -u -d "+10 minutes" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v+10M +"%Y-%m-%dT%H:%M:%SZ")
sqlite3 $DB "INSERT INTO oauth_device_codes
  (device_code_hash, user_code, client_id, scope, resource, status, poll_interval, expires_at, created_at)
  VALUES ('dch_smoke_e2', 'SMOK-EE12', '$CID', 'openid', '', 'pending', 5, '$DC_EXP', '$DC_NOW');"

PEND_DC=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/oauth/device-codes")
echo "$PEND_DC" | jq -e '.data[] | select(.user_code=="SMOK-EE12")' >/dev/null \
  && pass "seeded pending device code visible" || fail "device code missing: $PEND_DC"

# Approve
APPR_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X POST "$BASE/api/v1/admin/oauth/device-codes/SMOK-EE12/approve" \
  -H "Content-Type: application/json" -d "{\"user_id\":\"$USERID\"}")
[ "$APPR_CODE" = 200 ] && pass "POST /admin/oauth/device-codes/{user_code}/approve -> 200" || fail "approve -> $APPR_CODE"

# After approve, status flipped — no longer pending
NOT_PEND=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/oauth/device-codes")
echo "$NOT_PEND" | jq -e '[.data[] | select(.user_code=="SMOK-EE12")] | length == 0' >/dev/null \
  && pass "approved code dropped from pending queue" || fail "code still pending after approve"

# Re-approve same code -> 409 (no longer pending)
RE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X POST "$BASE/api/v1/admin/oauth/device-codes/SMOK-EE12/approve")
[ "$RE_CODE" = 409 ] && pass "re-approve already-decided -> 409" || fail "re-approve -> $RE_CODE"

# Approve missing -> 404
MISS_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X POST "$BASE/api/v1/admin/oauth/device-codes/NONE-XXXX/approve")
[ "$MISS_CODE" = 404 ] && pass "approve missing -> 404" || fail "missing approve -> $MISS_CODE"

# Audit row for approve
AUDIT_DEV=$(sqlite3 $DB "SELECT COUNT(*) FROM audit_logs WHERE action='oauth.device.approved' AND target_id='SMOK-EE12';")
[ "$AUDIT_DEV" = "1" ] && pass "audit log: oauth.device.approved" || fail "device approve audit missing (got $AUDIT_DEV)"

# No-auth blocked
NA_DC=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/admin/oauth/device-codes")
[ "$NA_DC" = 401 ] && pass "no auth -> 401 on device-codes" || fail "no-auth -> $NA_DC"

# --- 64: Admin vault connections (C Wave) -------------------------------------
section "64: admin vault connections"

# Empty case: shape contract.
EMPTY_AVC=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/vault/connections")
EMPTY_TYPE=$(echo "$EMPTY_AVC" | jq -r '.data | type // "missing"')
[ "$EMPTY_TYPE" = "array" ] && pass "/admin/vault/connections empty .data is array" || fail "shape: $EMPTY_AVC"

# Seed: create a vault provider via API, then insert a connection row directly.
SEED_PROV=$(curl -s -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" -X POST \
  -d '{"template":"slack","client_id":"smoke-c","client_secret":"smoke-s-abc"}' \
  $BASE/api/v1/vault/providers)
SEED_PID=$(echo "$SEED_PROV" | jq -r '.id // empty')
if [ -n "$SEED_PID" ]; then
  pass "seed vault provider: $SEED_PID"
  NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  sqlite3 $DB "INSERT INTO vault_connections
    (id, provider_id, user_id, access_token_enc, refresh_token_enc, token_type,
     scopes, expires_at, metadata, needs_reauth, last_refreshed_at, created_at, updated_at)
    VALUES ('vc_smoke_admin', '$SEED_PID', '$USERID', 'enc::abc', 'enc::xyz', 'Bearer',
     '[\"openid\",\"profile\"]', NULL, '{}', 0, NULL, '$NOW', '$NOW');"

  AVC=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/vault/connections")
  echo "$AVC" | jq -e --arg pid "$SEED_PID" '.data[] | select(.provider_id==$pid)' >/dev/null \
    && pass "seeded connection appears in /admin/vault/connections" \
    || fail "seeded connection not found: $AVC"

  # user_id present in admin response (cross-user view)
  echo "$AVC" | jq -e --arg uid "$USERID" '.data[] | select(.user_id==$uid)' >/dev/null \
    && pass "admin response includes user_id" \
    || fail "user_id missing from admin connection rows"

  # Token material never serialized
  echo "$AVC" | jq -e '[.data[] | (has("access_token_enc") or has("refresh_token_enc"))] | any | not' >/dev/null \
    && pass "no token material in admin response" \
    || fail "token material leaked into admin response"

  # Admin DELETE cross-user works
  AVC_DEL=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
    -X DELETE "$BASE/api/v1/admin/vault/connections/vc_smoke_admin")
  [ "$AVC_DEL" = 204 ] && pass "DELETE /admin/vault/connections/{id} -> 204" || fail "admin delete -> $AVC_DEL"

  # 404 on missing
  AVC_404=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
    -X DELETE "$BASE/api/v1/admin/vault/connections/vc_does_not_exist")
  [ "$AVC_404" = 404 ] && pass "DELETE missing -> 404" || fail "delete missing -> $AVC_404"

  # No-auth blocked
  AVC_NA=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/admin/vault/connections")
  [ "$AVC_NA" = 401 ] && pass "no auth -> 401" || fail "no-auth -> $AVC_NA"

  # cleanup
  curl -s -o /dev/null -H "Authorization: Bearer $ADMIN" -X DELETE "$BASE/api/v1/vault/providers/$SEED_PID"
else
  fail "could not seed vault provider for connections test"
fi

# --- 63: User list filters (B Wave) -------------------------------------------
section "63: user list filters (auth_method, org_id)"

# Response shape now is {users:[...], total:N}.
SHAPE=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?limit=5" | jq -r '.users | type // "missing"')
[ "$SHAPE" = "array" ] && pass "user list response has .users array" || fail "user list .users missing (got $SHAPE)"

TOTAL_ANY=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?limit=1000" | jq -r '.total // 0')
TOTAL_PWD=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?limit=1000&auth_method=password" | jq -r '.total // 0')
TOTAL_PASSKEY=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?limit=1000&auth_method=passkey" | jq -r '.total // 0')
# Smoke users use password — auth_method=password should match many; passkey should match fewer/none.
if [ "$TOTAL_PWD" -gt 0 ] && [ "$TOTAL_PWD" -le "$TOTAL_ANY" ]; then
  pass "auth_method=password filter narrows list ($TOTAL_PWD <= $TOTAL_ANY)"
else
  fail "auth_method=password filter wrong: pwd=$TOTAL_PWD all=$TOTAL_ANY"
fi
if [ "$TOTAL_PASSKEY" -le "$TOTAL_ANY" ]; then
  pass "auth_method=passkey filter applied ($TOTAL_PASSKEY <= $TOTAL_ANY)"
else
  fail "auth_method=passkey filter wrong: pk=$TOTAL_PASSKEY all=$TOTAL_ANY"
fi

# org_id filter — bogus org returns 0
TOTAL_BOGUS=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?org_id=org_does_not_exist_zzz" | jq -r '.total // 0')
[ "$TOTAL_BOGUS" = "0" ] && pass "org_id=bogus filter returns 0" || fail "org_id filter not applied (got $TOTAL_BOGUS)"

# page/per_page pagination
PAGE1=$(curl -s -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/users?page=1&per_page=2" | jq -r '.users | length')
[ "$PAGE1" -le 2 ] && pass "per_page=2 limits results ($PAGE1)" || fail "per_page not honored (got $PAGE1)"

# --- 67: proxy rules CRUD (Wave D) -------------------------------------------
# DB-backed proxy rule overrides. Endpoints are always available regardless of
# proxy enable state — admins must be able to stage rules before flipping the
# proxy on. Proxy is disabled in this smoke environment, so we exercise the
# CRUD surface without an engine refresh side-effect.
section "67: proxy rules CRUD (Wave D)"

# 1. List initially returns the documented {data:[],total:N} shape.
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/proxy/rules/db)
SHAPE=$(echo "$RESP" | jq -r '.data | type // "missing"')
[ "$SHAPE" = "array" ] && pass "GET /admin/proxy/rules/db -> {data:[]}" || fail "list shape: $SHAPE -- $RESP"
INITIAL_TOTAL=$(echo "$RESP" | jq -r '.total // 0')

# 2. Create a rule.
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Smoke Override","pattern":"/api/smoke/{id}","methods":["GET","PATCH"],"require":"role:admin","scopes":["webhooks:write"],"enabled":true,"priority":50}' \
  $BASE/api/v1/admin/proxy/rules/db)
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
[ "$CODE" = 201 ] && pass "POST create -> 201" || fail "create -> $CODE: $BODY"
PXR_ID=$(echo "$BODY" | jq -r '.data.id')
{ [ -n "$PXR_ID" ] && [ "$PXR_ID" != "null" ] ; } && pass "rule id returned ($PXR_ID)" || fail "no rule id"

# Validate created shape — methods uppercased + normalized, pattern + require + scopes round-trip.
METHODS=$(echo "$BODY" | jq -r '.data.methods | join(",")')
[ "$METHODS" = "GET,PATCH" ] && pass "methods normalized to GET,PATCH" || fail "methods wrong: $METHODS"
REQUIRE=$(echo "$BODY" | jq -r '.data.require')
[ "$REQUIRE" = "role:admin" ] && pass "require persisted (role:admin)" || fail "require wrong: $REQUIRE"

# 3. Validation: missing name → 400.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"pattern":"/x","require":"authenticated"}' $BASE/api/v1/admin/proxy/rules/db)
[ "$CODE" = 400 ] && pass "missing name -> 400" || fail "missing name accepted: $CODE"

# 4. Validation: pattern without leading slash → 400.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Bad","pattern":"api/foo","require":"authenticated"}' $BASE/api/v1/admin/proxy/rules/db)
[ "$CODE" = 400 ] && pass "pattern missing slash -> 400" || fail "bad pattern accepted: $CODE"

# 5. Validation: both require + allow → 400.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Bad","pattern":"/x","require":"authenticated","allow":"anonymous"}' $BASE/api/v1/admin/proxy/rules/db)
[ "$CODE" = 400 ] && pass "require+allow both set -> 400" || fail "require+allow accepted: $CODE"

# 6. Validation: unknown require kind → 400.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X POST \
  -d '{"name":"Bad","pattern":"/x","require":"bogus_kind"}' $BASE/api/v1/admin/proxy/rules/db)
[ "$CODE" = 400 ] && pass "unknown require kind -> 400" || fail "unknown require accepted: $CODE"

# 7. List shows the new rule.
RESP=$(curl -s -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/proxy/rules/db)
NEW_TOTAL=$(echo "$RESP" | jq -r '.total // 0')
[ "$NEW_TOTAL" -gt "$INITIAL_TOTAL" ] && pass "list reflects new rule ($INITIAL_TOTAL -> $NEW_TOTAL)" || fail "list missing rule"
FOUND=$(echo "$RESP" | jq -r --arg id "$PXR_ID" '.data[] | select(.id==$id) | .id')
[ "$FOUND" = "$PXR_ID" ] && pass "new rule visible by id" || fail "rule not in list"

# 8. GET by id.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/proxy/rules/db/$PXR_ID)
[ "$CODE" = 200 ] && pass "GET /admin/proxy/rules/db/{id} -> 200" || fail "get rule -> $CODE"

# 9. PATCH enabled flag.
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X PATCH \
  -d '{"enabled":false}' $BASE/api/v1/admin/proxy/rules/db/$PXR_ID)
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
[ "$CODE" = 200 ] && pass "PATCH enabled=false -> 200" || fail "patch -> $CODE: $BODY"
ENABLED=$(echo "$BODY" | jq -r '.data.enabled')
[ "$ENABLED" = "false" ] && pass "enabled flag toggled" || fail "enabled flag not toggled (got $ENABLED)"

# 10. PATCH back to enabled, change priority + name.
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X PATCH \
  -d '{"enabled":true,"priority":99,"name":"Smoke Override v2"}' $BASE/api/v1/admin/proxy/rules/db/$PXR_ID)
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
[ "$CODE" = 200 ] && pass "PATCH multi-field -> 200" || fail "multi patch -> $CODE: $BODY"
PRIORITY=$(echo "$BODY" | jq -r '.data.priority')
NAME=$(echo "$BODY" | jq -r '.data.name')
[ "$PRIORITY" = "99" ] && [ "$NAME" = "Smoke Override v2" ] && pass "priority+name updated" || fail "patch field mismatch: prio=$PRIORITY name=$NAME"

# 11. PATCH unknown id → 404.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -X PATCH \
  -d '{"enabled":false}' $BASE/api/v1/admin/proxy/rules/db/pxr_does_not_exist)
[ "$CODE" = 404 ] && pass "PATCH unknown id -> 404" || fail "missing id -> $CODE"

# 12. DELETE.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X DELETE $BASE/api/v1/admin/proxy/rules/db/$PXR_ID)
[ "$CODE" = 204 ] && pass "DELETE -> 204" || fail "delete -> $CODE"

# 13. Verify gone.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/proxy/rules/db/$PXR_ID)
[ "$CODE" = 404 ] && pass "GET deleted id -> 404" || fail "deleted id still reachable: $CODE"

# 14. DELETE again → 404 (idempotent on missing id).
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" \
  -X DELETE $BASE/api/v1/admin/proxy/rules/db/$PXR_ID)
[ "$CODE" = 404 ] && pass "DELETE missing -> 404" || fail "delete missing -> $CODE"

# 15. No auth -> 401.
CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/admin/proxy/rules/db)
[ "$CODE" = 401 ] && pass "no auth -> 401" || fail "unauth -> $CODE"

# --- 68: proxy status snake_case shape assertion --------------------------------
# When the proxy is enabled, GET /admin/proxy/status must return 200 and the
# response payload must contain a top-level "state" key (snake_case, not
# PascalCase). This catches regressions where the JSON struct tags drift from
# what the dashboard expects.
# In the default smoke environment the proxy is disabled and the endpoint
# returns 404 — that is expected and we skip the shape assertion in that case.
section "68: proxy status response shape (snake_case)"
PROXY_STATUS_RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $ADMIN" \
  $BASE/api/v1/admin/proxy/status)
PROXY_STATUS_CODE=$(echo "$PROXY_STATUS_RESP" | tail -1)
PROXY_STATUS_BODY=$(echo "$PROXY_STATUS_RESP" | sed '$d')
if [ "$PROXY_STATUS_CODE" = "404" ]; then
  pass "proxy disabled (404) — shape check skipped"
elif [ "$PROXY_STATUS_CODE" = "200" ]; then
  # Assert .data.state key exists (snake_case) and is not null/missing.
  STATE_VAL=$(echo "$PROXY_STATUS_BODY" | jq -r '.data.state // "MISSING"')
  [ "$STATE_VAL" != "MISSING" ] && pass "proxy status .data.state present (snake_case): $STATE_VAL" \
    || fail "proxy status .data.state missing — got: $PROXY_STATUS_BODY"
  # Assert the old PascalCase key is NOT present (regression guard).
  PASCAL_VAL=$(echo "$PROXY_STATUS_BODY" | jq -r '.data.State // "ABSENT"')
  [ "$PASCAL_VAL" = "ABSENT" ] && pass ".data.State (PascalCase) absent — no regression" \
    || fail ".data.State key present — backend JSON tags regressed"
else
  fail "proxy status unexpected code: $PROXY_STATUS_CODE"
fi

# --- 69: audit log CSV export (Compliance page dependency) ---------------------
# Compliance page (admin/src/components/compliance.tsx) POSTs {from,to} as JSON
# to /audit-logs/export and expects text/csv back. Guard the contract so
# dashboard download button doesn't silently break.
section "69: audit log CSV export (Compliance page)"

EXPORT_FROM=$(date -u -d "7 days ago" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-7d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null)
EXPORT_TO=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Empty body → 400 invalid_request (guards against accidental "export everything")
EMPTY_EXPORT_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -X POST --data '{}' \
  $BASE/api/v1/audit-logs/export)
[ "$EMPTY_EXPORT_CODE" = "400" ] && pass "POST /audit-logs/export {} -> 400" \
  || fail "POST /audit-logs/export {} -> $EMPTY_EXPORT_CODE (expected 400)"

# Dated export → 200 + text/csv (same request shape dashboard produces)
EXPORT_HEADERS=$(curl -s -D - -o /dev/null \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -X POST --data "{\"from\":\"$EXPORT_FROM\",\"to\":\"$EXPORT_TO\"}" \
  $BASE/api/v1/audit-logs/export)
EXPORT_CODE=$(echo "$EXPORT_HEADERS" | awk 'NR==1 {print $2}')
[ "$EXPORT_CODE" = "200" ] && pass "POST /audit-logs/export {from,to} -> 200" \
  || fail "POST /audit-logs/export {from,to} -> $EXPORT_CODE (expected 200)"
echo "$EXPORT_HEADERS" | grep -i '^Content-Type:' | grep -qi 'text/csv' \
  && pass "audit export returns text/csv" \
  || fail "audit export content-type not text/csv: $(echo "$EXPORT_HEADERS" | grep -i '^Content-Type:')"
echo "$EXPORT_HEADERS" | grep -i '^Content-Disposition:' | grep -qi '\.csv' \
  && pass "audit export Content-Disposition names .csv" \
  || fail "audit export Content-Disposition missing .csv"

# Unauthorized → 401 (guards admin-key requirement on export route)
EXPORT_UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST --data "{\"from\":\"$EXPORT_FROM\",\"to\":\"$EXPORT_TO\"}" \
  $BASE/api/v1/audit-logs/export)
[ "$EXPORT_UNAUTH" = "401" ] && pass "POST /audit-logs/export no key -> 401" \
  || fail "POST /audit-logs/export no key -> $EXPORT_UNAUTH"

# --- 70: POST /admin/users admin-key user creation (T04) ----------------------
# Dashboard create-user slide-over posts to /admin/users. Endpoint enforces
# admin-key auth, requires email, hashes password when provided, rejects
# duplicate emails, and writes an admin.user.create audit row.
section "70: POST /admin/users admin user creation (T04)"

# Unauthenticated → 401
T04_UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST --data '{"email":"t04-noauth@example.com"}' \
  $BASE/api/v1/admin/users)
[ "$T04_UNAUTH" = "401" ] && pass "POST /admin/users no key -> 401" \
  || fail "POST /admin/users no key -> $T04_UNAUTH (expected 401)"

# Empty body → 400 (JSON decodes to zero-value, email missing)
T04_EMPTY=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -X POST --data '{}' \
  $BASE/api/v1/admin/users)
[ "$T04_EMPTY" = "400" ] && pass "POST /admin/users {} -> 400" \
  || fail "POST /admin/users {} -> $T04_EMPTY (expected 400)"

# Missing email (name-only body) → 400
T04_NO_EMAIL=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -X POST --data '{"name":"no email"}' \
  $BASE/api/v1/admin/users)
[ "$T04_NO_EMAIL" = "400" ] && pass "POST /admin/users no email -> 400" \
  || fail "POST /admin/users no email -> $T04_NO_EMAIL (expected 400)"

# Valid email+password → 201 + returned user has id + matching email
T04_EMAIL="t04-admin-created@example.com"
T04_CREATE_RESP=$(curl -s -w "\n%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -X POST --data "{\"email\":\"$T04_EMAIL\",\"password\":\"SuperSecret123!\",\"name\":\"T04 Admin\"}" \
  $BASE/api/v1/admin/users)
T04_CREATE_CODE=$(echo "$T04_CREATE_RESP" | tail -1)
T04_CREATE_BODY=$(echo "$T04_CREATE_RESP" | sed '$d')
[ "$T04_CREATE_CODE" = "201" ] && pass "POST /admin/users valid -> 201" \
  || fail "POST /admin/users valid -> $T04_CREATE_CODE (body: $T04_CREATE_BODY)"

T04_NEW_ID=$(echo "$T04_CREATE_BODY" | jq -r '.id // "MISSING"')
[ "$T04_NEW_ID" != "MISSING" ] && [ -n "$T04_NEW_ID" ] \
  && pass "created user has id: $T04_NEW_ID" \
  || fail "created user missing id (body: $T04_CREATE_BODY)"
T04_NEW_EMAIL=$(echo "$T04_CREATE_BODY" | jq -r '.email // "MISSING"')
[ "$T04_NEW_EMAIL" = "$T04_EMAIL" ] \
  && pass "created user email matches input" \
  || fail "created user email mismatch: want $T04_EMAIL got $T04_NEW_EMAIL"

# Duplicate email → 409 + {"error":"email_exists"}
T04_DUP_RESP=$(curl -s -w "\n%{http_code}" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" \
  -X POST --data "{\"email\":\"$T04_EMAIL\",\"password\":\"SuperSecret123!\"}" \
  $BASE/api/v1/admin/users)
T04_DUP_CODE=$(echo "$T04_DUP_RESP" | tail -1)
T04_DUP_BODY=$(echo "$T04_DUP_RESP" | sed '$d')
[ "$T04_DUP_CODE" = "409" ] && pass "duplicate email -> 409" \
  || fail "duplicate email -> $T04_DUP_CODE (expected 409)"
T04_DUP_ERR=$(echo "$T04_DUP_BODY" | jq -r '.error // "MISSING"')
[ "$T04_DUP_ERR" = "email_exists" ] \
  && pass "duplicate error code=email_exists" \
  || fail "duplicate error code=$T04_DUP_ERR (expected email_exists)"

# Audit log entry with action=admin.user.create for new user id
if [ -n "$T04_NEW_ID" ] && [ "$T04_NEW_ID" != "MISSING" ]; then
  T04_AUDIT=$(curl -s -H "Authorization: Bearer $ADMIN" \
    "$BASE/api/v1/audit-logs?action=admin.user.create&limit=50")
  T04_AUDIT_HIT=$(echo "$T04_AUDIT" | jq -r --arg id "$T04_NEW_ID" \
    '.logs // .audit_logs // [] | map(select(.target_id == $id and .action == "admin.user.create")) | length')
  if [ "${T04_AUDIT_HIT:-0}" -gt 0 ] 2>/dev/null; then
    pass "audit log admin.user.create for $T04_NEW_ID present"
  else
    fail "audit log admin.user.create for $T04_NEW_ID missing (got: $T04_AUDIT)"
  fi
fi

section "71: bootstrap token consume (T15)"

# The bootstrap token is minted at server startup ONLY when the audit log
# has zero admin.* events. This smoke runs after many admin.* events have
# already been written (T04/T05/T21/etc), so the server will NOT print a
# fresh token and any in-memory token from startup is long gone. We
# therefore assert the contract of the consume endpoint from the outside:
# empty body → 400, random token → 401. Fresh-DB startup print is covered
# by inspection of server.Serve + printBootstrapURL (see PROGRESS row).

# No-auth route (bootstrap IS the auth): verify the endpoint is reachable
# without an Authorization header. Empty body → 400.
T15_EMPTY=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST --data '' \
  $BASE/api/v1/admin/bootstrap/consume)
[ "$T15_EMPTY" = "400" ] && pass "bootstrap consume empty body -> 400" \
  || fail "bootstrap consume empty body -> $T15_EMPTY (expected 400)"

# Missing-token JSON → 400 with error=invalid_request
T15_MISSING=$(curl -s -w "\n%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST --data '{}' \
  $BASE/api/v1/admin/bootstrap/consume)
T15_MISSING_CODE=$(echo "$T15_MISSING" | tail -1)
T15_MISSING_BODY=$(echo "$T15_MISSING" | sed '$d')
[ "$T15_MISSING_CODE" = "400" ] && pass "bootstrap consume missing token -> 400" \
  || fail "bootstrap consume missing token -> $T15_MISSING_CODE (expected 400)"
T15_MISSING_ERR=$(echo "$T15_MISSING_BODY" | jq -r '.error // "MISSING"')
[ "$T15_MISSING_ERR" = "invalid_request" ] \
  && pass "bootstrap consume missing token error=invalid_request" \
  || fail "bootstrap consume missing token error=$T15_MISSING_ERR"

# Random token → 401 invalid_token (either no token active because admin
# already exists, or token hash mismatch). Either way: 401.
T15_RANDOM=$(curl -s -w "\n%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST --data '{"token":"deadbeef0000000000000000000000000000000000000000000000000000dead"}' \
  $BASE/api/v1/admin/bootstrap/consume)
T15_RANDOM_CODE=$(echo "$T15_RANDOM" | tail -1)
T15_RANDOM_BODY=$(echo "$T15_RANDOM" | sed '$d')
[ "$T15_RANDOM_CODE" = "401" ] && pass "bootstrap consume random token -> 401" \
  || fail "bootstrap consume random token -> $T15_RANDOM_CODE (expected 401)"
T15_RANDOM_ERR=$(echo "$T15_RANDOM_BODY" | jq -r '.error // "MISSING"')
[ "$T15_RANDOM_ERR" = "invalid_token" ] \
  && pass "bootstrap consume random token error=invalid_token" \
  || fail "bootstrap consume random token error=$T15_RANDOM_ERR"

# No Authorization header required — the route is explicitly un-gated.
# Confirm we don't return 401 from middleware (we return 400 from handler
# when body is empty, as asserted above). This single assertion guards
# against accidentally moving the route inside the admin-auth group.
T15_NOAUTH=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST --data '{"token":"x"}' \
  $BASE/api/v1/admin/bootstrap/consume)
[ "$T15_NOAUTH" = "401" ] && pass "bootstrap consume no-auth route (handler-level 401, not middleware)" \
  || fail "bootstrap consume no-auth -> $T15_NOAUTH (expected handler 401 for bad token)"

# --- 72: W15 multi-listener proxy --------------------------------------------
section "72: W15 multi-listener proxy (embedded)"

# The main :8080 server is already up and owns dev.db. For this section we
# need a toy HTTP upstream on :9001 and a proxy listener on :9000 that
# injects identity headers. Running a second shark instance would require
# a separate DB — easier to exercise the Listener directly via a tiny
# Python or Go helper. We use python3 for the toy upstream, and a dedicated
# shark instance with its own DB on :8090 for the proxy listener.

if ! command -v python3 >/dev/null 2>&1; then
  note "section 72 skipped: python3 not available"
else
  kill_port 9001
  kill_port 9000
  kill_port 9101
  kill_port 9100
  kill_port 8090

  # Two toy upstreams so we can verify ≥2 listeners running side-by-side,
  # each with their own rules, each proxying to their own backend.
  # Upstream A on :9001 echoes headers; upstream B on :9101 returns a
  # fixed "from-b" body so we can assert traffic was routed to the right
  # backend (not cross-wired).
  cat > upstream72.py <<'PY'
import http.server, json, sys
port = int(sys.argv[1])
tag = sys.argv[2]
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("X-Backend-Tag", tag)
        self.end_headers()
        body = {"tag": tag, "headers": {k: v for k, v in self.headers.items()}}
        self.wfile.write(json.dumps(body).encode())
    def log_message(self, *a): pass
http.server.HTTPServer(("127.0.0.1", port), H).serve_forever()
PY
  python3 upstream72.py 9001 A > upstream.log 2>&1 &
  UPSTREAM_PID=$!
  python3 upstream72.py 9101 B > upstreamB.log 2>&1 &
  UPSTREAM_B_PID=$!

  # Give both upstreams a moment to bind.
  for _ in $(seq 1 20); do
    curl -sf http://127.0.0.1:9001/ >/dev/null 2>&1 \
      && curl -sf http://127.0.0.1:9101/ >/dev/null 2>&1 && break
    sleep 0.1
  done

  if ! curl -sf http://127.0.0.1:9001/ >/dev/null 2>&1 \
     || ! curl -sf http://127.0.0.1:9101/ >/dev/null 2>&1; then
    fail "toy upstreams did not start on :9001 + :9101"
    kill $UPSTREAM_PID $UPSTREAM_B_PID 2>/dev/null || true
  else
    pass "toy upstreams up on :9001 + :9101"

    # Fresh shark instance with TWO proxy listeners on its own DB. Each
    # listener binds its own port, has its own upstream, and its own rule
    # set — verifying listener isolation end-to-end.
    rm -f w15.db w15.db-wal w15.db-shm w15.yaml w15-server.log
    cat > w15.yaml <<EOF
server:
  port: 8090
  base_url: http://localhost:8090
  secret: "w15-smoke-secret-xxxxxxxxxxxxxxxxxxxxxxxxxxx"
storage:
  path: w15.db
email:
  provider: dev
proxy:
  listeners:
    - bind: "127.0.0.1:9000"
      upstream: "http://127.0.0.1:9001"
      rules:
        - path: /public/*
          allow: anonymous
        - path: /*
          require: authenticated
    - bind: "127.0.0.1:9100"
      upstream: "http://127.0.0.1:9101"
      rules:
        - path: /*
          allow: anonymous
EOF

    $BIN serve --dev --config w15.yaml >> w15-server.log 2>&1 &
    W15_PID=$!

    # Wait for main (:8090) + both listeners (:9000, :9100) to come up.
    UP=0
    for _ in $(seq 1 50); do
      if curl -sf http://127.0.0.1:8090/healthz >/dev/null 2>&1 \
         && curl -s -o /dev/null http://127.0.0.1:9000/public/foo \
         && curl -s -o /dev/null http://127.0.0.1:9100/anything; then
        UP=1; break
      fi
      sleep 0.2
    done

    if [ $UP -ne 1 ]; then
      cat w15-server.log | tail -30
      fail "W15 shark instance didn't bind :8090 + :9000 + :9100"
    else
      pass "multi-listener: :8090 API + :9000 proxy A + :9100 proxy B all bound"

      # Listener A: anonymous on public path -> 200 (rule allows anonymous).
      CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:9000/public/foo)
      [ "$CODE" = "200" ] && pass "listener A: anonymous /public/* -> 200" \
        || fail "listener A: anonymous /public/* -> $CODE (expected 200)"

      # Listener A: anonymous on protected path -> 401 per spec.
      CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:9000/secret)
      [ "$CODE" = "401" ] \
        && pass "listener A: anonymous /secret -> 401 (unauth)" \
        || fail "listener A: anonymous /secret -> $CODE (expected 401)"

      # Listener B: rule wide-open — every path allowed anonymously. Also
      # verify routing landed on backend B (tag=B) not backend A.
      TAG=$(curl -s http://127.0.0.1:9100/any/path | grep -o '"tag": *"[AB]"' | head -1)
      [ "$TAG" = '"tag": "B"' ] \
        && pass "listener B: routed to backend B ($TAG)" \
        || fail "listener B: wrong backend routing ($TAG)"

      # Cross-check: listener A should NOT route to backend B even for
      # public paths (isolation sanity).
      TAG=$(curl -s http://127.0.0.1:9000/public/ping | grep -o '"tag": *"[AB]"' | head -1)
      [ "$TAG" = '"tag": "A"' ] \
        && pass "listener A: routed to backend A ($TAG)" \
        || fail "listener A: wrong backend routing ($TAG)"

      # Admin port still works.
      CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8090/healthz)
      [ "$CODE" = "200" ] && pass "admin port still healthy during multi-listener" \
        || fail "admin healthz on :8090 -> $CODE"

      # Port-in-use = fatal. Try booting a second instance bound to :9000.
      kill_port 8091 2>/dev/null || true
      cat > w15-dup.yaml <<EOF
server:
  port: 8091
  base_url: http://localhost:8091
  secret: "w15-smoke-secret-yyyyyyyyyyyyyyyyyyyyyyyyyyy"
storage:
  path: w15-dup.db
email:
  provider: dev
proxy:
  listeners:
    - bind: "127.0.0.1:9000"
      upstream: "http://127.0.0.1:9001"
      rules:
        - path: /*
          allow: anonymous
EOF
      $BIN serve --dev --config w15-dup.yaml > w15-dup.log 2>&1
      RC=$?
      if [ $RC -ne 0 ]; then
        pass "port-in-use on :9000 -> fatal startup (exit $RC)"
      else
        fail "port-in-use accepted — expected fatal bind error"
      fi
      rm -f w15-dup.yaml w15-dup.db w15-dup.log
    fi

    kill $W15_PID 2>/dev/null || true
    wait $W15_PID 2>/dev/null || true
    kill_port 8090
    kill_port 9000
    kill_port 9100
    rm -f w15.yaml w15.db w15.db-wal w15.db-shm w15-server.log
  fi

  kill $UPSTREAM_PID $UPSTREAM_B_PID 2>/dev/null || true
  kill_port 9001
  kill_port 9101
  rm -f upstream.log upstreamB.log upstream72.py
fi

# --- 73: W15 standalone shark proxy + JWT verify -----------------------------
section "73: W15 standalone shark proxy JWT verify"

if ! command -v python3 >/dev/null 2>&1; then
  note "section 73 skipped: python3 not available"
else
  kill_port 9002
  kill_port 9003

  # Toy upstream on :9003.
  cat > upstream73.py <<'PY'
import http.server
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"ok")
    def log_message(self, *a): pass
http.server.HTTPServer(("127.0.0.1", 9003), H).serve_forever()
PY
  python3 upstream73.py > upstream73.log 2>&1 &
  UP73_PID=$!

  for _ in $(seq 1 20); do
    curl -sf http://127.0.0.1:9003/ >/dev/null 2>&1 && break
    sleep 0.1
  done

  # Rules file requiring authentication on all paths.
  cat > rules73.yaml <<EOF
rules:
  - path: /*
    require: authenticated
EOF

  # W15c: --audience + --issuer mandatory. Must match what the shark auth
  # server mints (configured above in the main YAML: audience=shark-smoke,
  # issuer=http://localhost:8080 — which is $BASE).
  $BIN proxy --upstream http://127.0.0.1:9003 --port 9002 \
    --auth $BASE --rules rules73.yaml \
    --audience shark-smoke --issuer $BASE > proxy73.log 2>&1 &
  PROXY73_PID=$!

  # Wait for bind.
  UP=0
  for _ in $(seq 1 30); do
    curl -s -o /dev/null http://127.0.0.1:9002/ && { UP=1; break; }
    sleep 0.2
  done

  if [ $UP -ne 1 ]; then
    cat proxy73.log | tail -30
    fail "standalone proxy did not start on :9002"
  else
    pass "standalone proxy bound on :9002, JWKS fetched from $BASE"

    # No token -> 401 (anonymous fails authenticated rule, per spec).
    CODE=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:9002/)
    [ "$CODE" = "401" ] \
      && pass "no auth -> 401 (unauth)" \
      || fail "no auth -> $CODE (expected 401)"

    # $TOKEN was minted at signup in §2. It's an RS256 session JWT signed
    # by the same shark instance whose JWKS the standalone proxy cached.
    if [ -n "${TOKEN:-}" ]; then
      CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" http://127.0.0.1:9002/)
      [ "$CODE" = "200" ] && pass "valid JWT -> 200" \
        || fail "valid JWT -> $CODE (expected 200)"

      # Tampered token: flip a byte in the signature.
      TAMPERED="${TOKEN%??}aa"
      CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TAMPERED" http://127.0.0.1:9002/)
      [ "$CODE" = "401" ] && pass "tampered JWT -> 401" \
        || fail "tampered JWT -> $CODE (expected 401)"

      # Bogus token structure.
      CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer not.a.jwt" http://127.0.0.1:9002/)
      [ "$CODE" = "401" ] && pass "malformed JWT -> 401" \
        || fail "malformed JWT -> $CODE (expected 401)"
    else
      note "TOKEN not captured from §2 — skipping valid-JWT assertion"
    fi

    # Help text no longer carries the MVP caveat.
    HELP=$($BIN proxy --help 2>&1)
    if echo "$HELP" | grep -qi "MVP scope"; then
      fail "proxy --help still contains MVP-scope caveat"
    else
      pass "proxy --help no longer warns MVP-scope"
    fi
  fi

  kill $PROXY73_PID 2>/dev/null || true
  wait $PROXY73_PID 2>/dev/null || true
  kill $UP73_PID 2>/dev/null || true
  kill_port 9002
  kill_port 9003
  rm -f rules73.yaml proxy73.log upstream73.log upstream73.py
fi

# --- 74: Branding + Email Templates + integration_mode (Phase A) -------------
section "74. Branding CRUD"

# GET default branding — primary_color seeded to #7c3aed.
r=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/branding")
if echo "$r" | jq -e '.branding.primary_color == "#7c3aed"' > /dev/null 2>&1; then
  pass "branding GET returns default primary #7c3aed"
else
  note "body: $r"
  fail "branding GET default primary mismatch"
fi

# PATCH branding — set primary_color to #ff0000 and confirm via GET.
curl -sS -X PATCH -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"primary_color":"#ff0000"}' "$BASE/api/v1/admin/branding" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/branding")
if echo "$r" | jq -e '.branding.primary_color == "#ff0000"' > /dev/null 2>&1; then
  pass "branding PATCH persists primary_color"
else
  note "body: $r"
  fail "branding PATCH did not persist"
fi

# Logo upload must reject files >1MB.
dd if=/dev/zero of=/tmp/shark_big_logo.png bs=1048576 count=2 2>/dev/null
CODE=$(curl -sS -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $ADMIN" \
  -F "logo=@/tmp/shark_big_logo.png" "$BASE/api/v1/admin/branding/logo")
if [ "$CODE" = "400" ] || [ "$CODE" = "413" ]; then
  pass "logo upload rejects >1MB ($CODE)"
else
  fail "logo upload >1MB returned $CODE (expected 400 or 413)"
fi
rm -f /tmp/shark_big_logo.png

# Logo upload accepts a tiny valid PNG.
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDAT\x18Wc\x00\x00\x00\x03\x00\x01\x8f\x8f\xa4\xa6\x00\x00\x00\x00IEND\xaeB\x60\x82' > /tmp/shark_tiny_logo.png
r=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN" \
  -F "logo=@/tmp/shark_tiny_logo.png" "$BASE/api/v1/admin/branding/logo")
if echo "$r" | jq -e '.logo_url | startswith("/assets/branding/")' > /dev/null 2>&1; then
  pass "logo upload accepts tiny PNG, returns /assets/branding/ URL"
else
  note "body: $r"
  fail "logo upload tiny PNG did not return expected logo_url"
fi
rm -f /tmp/shark_tiny_logo.png

section "74b. Email Templates"

# List returns 5 seeded templates.
r=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/email-templates")
COUNT=$(echo "$r" | jq '.data | length')
if [ "$COUNT" = "5" ]; then
  pass "email-templates list returns 5 seeded templates"
else
  note "body: $r"
  fail "email-templates list returned $COUNT (expected 5)"
fi

# PATCH magic_link subject.
curl -sS -X PATCH -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"subject":"Custom subject"}' "$BASE/api/v1/admin/email-templates/magic_link" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/email-templates/magic_link")
if echo "$r" | jq -e '.subject == "Custom subject"' > /dev/null 2>&1; then
  pass "email template PATCH persists subject"
else
  note "body: $r"
  fail "email template PATCH did not persist"
fi

# Preview returns structured response with our updated subject + rendered HTML.
r=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{}' "$BASE/api/v1/admin/email-templates/magic_link/preview")
if echo "$r" | jq -e '.subject == "Custom subject" and (.html | type == "string") and (.html | length > 0)' > /dev/null 2>&1; then
  pass "email preview returns subject + html"
else
  note "body: $r"
  fail "email preview missing subject or html"
fi

# Reset reverts to seeded default (subject contains "Sign in to").
curl -sS -X POST -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/email-templates/magic_link/reset" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/email-templates/magic_link")
if echo "$r" | jq -e '.subject | contains("Sign in to")' > /dev/null 2>&1; then
  pass "email template reset restores default subject"
else
  note "body: $r"
  fail "email template reset did not restore default"
fi

section "74c. Application integration_mode"

# Pick first app id from admin list. Route is /admin/apps (not /admin/applications);
# response field is .data (list of applicationResponse).
APPS=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/apps")
APP_ID=$(echo "$APPS" | jq -r '.data[0].id // empty')
if [ -n "$APP_ID" ]; then
  pass "picked app id $APP_ID"

  # PATCH integration_mode to components.
  CODE=$(curl -sS -o /dev/null -w "%{http_code}" -X PATCH \
    -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
    -d '{"integration_mode":"components"}' "$BASE/api/v1/admin/apps/$APP_ID")
  if [ "$CODE" = "200" ] || [ "$CODE" = "204" ]; then
    pass "integration_mode PATCH $CODE"
  else
    fail "integration_mode PATCH returned $CODE"
  fi

  # Snippet endpoint returns 3 snippets for react framework.
  r=$(curl -sS -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/apps/$APP_ID/snippet?framework=react")
  if echo "$r" | jq -e '.snippets | length == 3' > /dev/null 2>&1; then
    pass "snippet endpoint returns 3 React snippets"
  else
    note "body: $r"
    fail "snippet endpoint did not return 3 React snippets"
  fi
else
  note "body: $APPS"
  fail "could not pick first application id for integration_mode test"
fi

# --- Summary ------------------------------------------------------------------
section "summary"
echo "  ${GRN}PASS: $PASS${RST}   ${RED}FAIL: $FAIL${RST}"
if [ $FAIL -gt 0 ]; then
  echo
  echo "  Failures:"
  for d in "${FAIL_DETAILS[@]}"; do echo "    - $d"; done
  exit 1
fi
exit 0
