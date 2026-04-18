#!/bin/bash
# Shark Phase 3 smoke test. See SMOKE_TEST.md.
set -u

BASE="${BASE:-http://localhost:8080}"
DB="${DB:-sharkauth.db}"
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

stop_server() {
  if [ -n "${SERVER_PID:-}" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
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
  $BIN serve >> server.log 2>&1 &
  SERVER_PID=$!
  wait_for_server || { cat server.log; fail "server didn't come up"; exit 1; }
}

# --- Pre: fresh DB + yaml -----------------------------------------------------
section "bootstrap: fresh DB"
# Kill ANY process on :8080 (handles prior manual `./shark serve`)
if command -v fuser >/dev/null 2>&1; then
  fuser -k 8080/tcp 2>/dev/null || true
elif command -v lsof >/dev/null 2>&1; then
  PIDS=$(lsof -ti :8080 2>/dev/null || true)
  [ -n "$PIDS" ] && kill $PIDS 2>/dev/null || true
fi
sleep 0.5
# Remove DB + SQLite WAL/journal siblings so bootstrap generates fresh keys
rm -f $DB $DB-journal $DB-wal $DB-shm $YAML server.log cj*.txt
if [ ! -x "$BIN" ]; then echo "build binary first: go build -o $BIN ./cmd/shark"; exit 1; fi

# Write a minimal yaml matching `shark init` output.
cat > $YAML <<EOF
server:
  base_url: http://localhost:8080
  secret: "change-me-this-secret-is-not-secure-at-all-abc123456789"
  listen_addr: ":8080"
auth:
  jwt:
    enabled: true
    mode: "session"
    audience: "shark"
storage:
  path: $DB
email:
  provider: "shark_relay"
EOF

boot_server
sleep 1
if grep -q "Default application created" server.log; then pass "default app banner"; else fail "no default app banner"; fi
if grep -q "ADMIN API KEY" server.log; then pass "admin key banner"; else fail "no admin key banner"; fi

ADMIN=$(grep -oE 'sk_live_[A-Za-z0-9]+' server.log | head -1)
DEFAULT_CID=$(grep -oE 'shark_app_[A-Za-z0-9_-]+' server.log | head -1)
[ -n "$ADMIN" ] && pass "admin key captured" || fail "admin key extract"
[ -n "$DEFAULT_CID" ] && pass "default client_id captured: $DEFAULT_CID" || fail "client_id extract"

# --- 2: JWT at signup ---------------------------------------------------------
section "signup issues JWT"
EMAIL="smoke$RANDOM@test.com"
RESP=$(curl -s -c cj.txt -X POST $BASE/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"GetCake117\$\$\$\"}")
TOKEN=$(echo "$RESP" | jq -r '.token // empty')
USERID=$(echo "$RESP" | jq -r '.id // empty')
[ -n "$TOKEN" ] && pass "token in signup response" || { fail "no token"; echo "$RESP"; }
[ -n "$USERID" ] && pass "user id in response" || fail "no user id"

# Sanity-check that captured admin key actually authenticates.
CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" $BASE/api/v1/admin/apps)
if [ "$CODE" = 200 ]; then
  pass "admin key sanity check"
else
  note "admin probe → $CODE  key=[${ADMIN:0:16}...] len=${#ADMIN}"
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
[ "$CODE" = 401 ] && pass "garbage Bearer + valid cookie → 401 (no fallthrough)" || fail "no-fallthrough violated: $CODE"

WWW=$(curl -s -D - -o /dev/null -H "Authorization: Bearer garbage" $BASE/api/v1/auth/me | grep -i 'www-authenticate')
echo "$WWW" | grep -qi 'Bearer' && pass "WWW-Authenticate header present" || fail "no WWW-Authenticate"

# No auth at all
CODE=$(curl -s -o /dev/null -w "%{http_code}" $BASE/api/v1/auth/me)
[ "$CODE" = 401 ] && pass "no auth → 401" || fail "no auth → $CODE"

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
[ "$CODE" = 401 ] && pass "no auth → 401" || fail "admin revoke no-auth → $CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN" -X POST $BASE/api/v1/admin/auth/revoke-jti \
  -H "Content-Type: application/json" -d '{"jti":"x","expires_at":"2030-01-01T00:00:00Z"}')
if [ "$CODE" = 200 ] || [ "$CODE" = 204 ]; then pass "admin revoke 2xx"; else fail "admin revoke → $CODE"; fi

# --- 7: Key rotation ----------------------------------------------------------
section "key rotation"
stop_server
$BIN keys generate-jwt --rotate > rotate.log 2>&1
if [ $? -eq 0 ]; then pass "rotate CLI exit 0"; else cat rotate.log; fail "rotate CLI failed"; fi
boot_server
sleep 0.5
JWKS=$(curl -s $BASE/.well-known/jwks.json)
N=$(echo "$JWKS" | jq '.keys | length')
[ "$N" = 2 ] && pass "JWKS has 2 keys post-rotate" || fail "JWKS post-rotate keys=$N"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" $BASE/api/v1/auth/me)
[ "$CODE" = 200 ] && pass "old token still validates (retired-key window)" || fail "old token /me $CODE"

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

# --- 10: Redirect allowlist (magic-link flow, §4.4) ---------------------------
section "redirect allowlist"
# Send magic link with bad redirect, then consume with bad redirect — expect 400.
# Send magic link API differs per handler wiring. Hit the verify endpoint directly with an unknown token to exercise validator.
# Instead test the redirect validator side: try /api/v1/auth/magic-link/verify with redirect_url set.
# Since we can't easily generate a real token here, we drive the validator error path:
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/auth/magic-link/verify?token=fake&redirect_url=javascript:alert(1)")
[ "$CODE" = 400 ] && pass "javascript: redirect → 400" || note "js-url got $CODE (may surface token error first)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/auth/magic-link/verify?token=fake&redirect_url=https://evil.example.com")
[ "$CODE" = 400 ] && pass "non-allowlisted redirect → 400" || note "evil-url got $CODE (may surface token error first)"

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
  [ "$CODE" = 409 ] && pass "builtin delete → 409" || fail "builtin delete → $CODE"
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
