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
[ "$N" = 1 ] && pass "1 key in JWKS" || fail "JWKS keys=$N"
echo "$JWKS" | jq -e '.keys[0].kty=="RSA" and .keys[0].alg=="RS256" and .keys[0].use=="sig"' >/dev/null && pass "kty/alg/use correct" || fail "JWK shape"

KID_JWKS=$(echo "$JWKS" | jq -r '.keys[0].kid')
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
[ "$N" = 2 ] && pass "JWKS has 2 keys post-rotate" || fail "JWKS post-rotate keys=$N"

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
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/sessions)
[ "$CODE" = 200 ] && pass "GET /admin/sessions 200" || fail "GET /admin/sessions $CODE"

# --- 18: Stats + Trends -------------------------------------------------------
section "stats + trends"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/stats)
[ "$CODE" = 200 ] && pass "GET /admin/stats 200" || fail "GET /admin/stats $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" "$BASE/api/v1/admin/stats/trends?days=7")
[ "$CODE" = 200 ] && pass "GET /admin/stats/trends 200" || fail "GET /admin/stats/trends $CODE"

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
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/webhooks)
[ "$CODE" = 200 ] && pass "webhook list 200" || fail "webhook list $CODE"

# Test fire
if [ -n "$WH_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/webhooks/$WH_ID/test)
  [ "$CODE" = 200 ] || [ "$CODE" = 202 ] && pass "webhook test $CODE" || fail "webhook test $CODE"

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
