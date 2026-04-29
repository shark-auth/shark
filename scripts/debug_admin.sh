#!/bin/bash
KEY=$(grep -oE 'sk_live_[A-Za-z0-9]+' server.log | head -1)
echo "KEY=[$KEY]  len=${#KEY}"
echo
echo "== curl admin apps =="
curl -s -w "\nHTTP %{http_code}\n" -H "Authorization: Bearer $KEY" http://localhost:8080/api/v1/admin/apps
echo
echo "== DB row =="
sqlite3 sharkauth.db "SELECT id, name, key_prefix, substr(key_hash,1,20) AS hash20, scopes, revoked_at FROM api_keys"
