#!/usr/bin/env bash
# Register 6 agents via POST /api/v1/agents (admin) and write .env.
# Requires: SHARK_ADMIN_KEY env var. SHARK_AUTH_URL defaults to http://localhost:8080.

set -euo pipefail

AUTH="${SHARK_AUTH_URL:-http://localhost:8080}"
ADMIN_KEY="${SHARK_ADMIN_KEY:?SHARK_ADMIN_KEY required}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"

if [[ ! -x "$(command -v jq)" ]]; then
  echo "fatal: jq required" >&2
  exit 1
fi

register() {
  local name="$1" desc="$2" scopes="$3"
  curl -fsS -X POST "$AUTH/api/v1/agents" \
    -H "Authorization: Bearer $ADMIN_KEY" \
    -H "Content-Type: application/json" \
    -d "$(jq -nc \
        --arg name "$name" --arg desc "$desc" \
        --argjson scopes "$(jq -Rc 'split(" ")' <<<"$scopes")" \
        '{
          name: $name,
          description: $desc,
          client_type: "confidential",
          auth_method: "client_secret_basic",
          grant_types: ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
          scopes: $scopes,
          token_lifetime: 900
        }')"
}

extract() {
  jq -r ".$1"
}

write_env() {
  local key_prefix="$1" json="$2"
  local cid csec
  cid=$(echo "$json" | extract client_id)
  csec=$(echo "$json" | extract client_secret)
  echo "${key_prefix}_CLIENT_ID=$cid" >> "$ENV_FILE"
  echo "${key_prefix}_CLIENT_SECRET=$csec" >> "$ENV_FILE"
  echo "  → ${key_prefix} = $cid"
}

cat > "$ENV_FILE" <<EOF
SHARK_AUTH_URL=$AUTH
SHARK_ADMIN_KEY=$ADMIN_KEY

EOF

echo "Registering 7 agents..."

PLATFORM=$(register "supportflow-platform" "SupportFlow application client" \
  "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write")
write_env "PLATFORM" "$PLATFORM"

TRIAGE=$(register "triage-agent" "Routes incoming support tickets" \
  "ticket:read ticket:resolve customer:read crm:write email:draft kb:read calendar:write")
write_env "TRIAGE" "$TRIAGE"

KNOWLEDGE=$(register "knowledge-agent" "Fetches knowledge-base articles" "kb:read")
write_env "KNOWLEDGE" "$KNOWLEDGE"

EMAIL=$(register "email-agent" "Drafts customer email replies" "email:draft email:send")
write_env "EMAIL" "$EMAIL"

GMAIL=$(register "gmail-tool" "SMTP-relay sender (3-deep delegation)" "email:send")
write_env "GMAIL_TOOL" "$GMAIL"

CRM=$(register "crm-agent" "Updates Salesforce records" "crm:write")
write_env "CRM" "$CRM"

FOLLOWUP=$(register "followup-agent" "Schedules check-in calls" "calendar:write")
write_env "FOLLOWUP" "$FOLLOWUP"

echo ""
echo "Seeded. Credentials → $ENV_FILE"
echo ""
echo "NOTE: may_act constraints are NOT seeded automatically — see DEMO_02_DELEGATION_CHAIN.md §11."
echo "      To enforce 'only crm-agent may exchange to crm:write', patch storage with:"
echo "      UPDATE oauth_clients SET metadata = json_set(metadata, '\$.may_act', json('[\"crm-agent\"]'))"
echo "        WHERE client_id = '\$TRIAGE_CLIENT_ID';"
