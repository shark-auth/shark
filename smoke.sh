#!/bin/bash
set -e
BASE=http://localhost:8080

echo "== signup =="
TOKEN=$(curl -s -c cj.txt -X POST $BASE/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke'$RANDOM'@test.com","password":"GetCake117$$$"}' | jq -r .token)
echo "token prefix: ${TOKEN:0:40}..."

echo "== Bearer /me =="
curl -s -H "Authorization: Bearer $TOKEN" $BASE/api/v1/auth/me | jq .

echo "== Cookie /me =="
curl -s -b cj.txt $BASE/api/v1/auth/me | jq .

echo "== Garbage Bearer + valid cookie (expect 401) =="
curl -s -o /dev/null -w "HTTP %{http_code}\n" \
  -H "Authorization: Bearer garbage" -b cj.txt $BASE/api/v1/auth/me

echo "== JWKS =="
curl -s $BASE/.well-known/jwks.json | jq '.keys | length, .keys[0].kid, .keys[0].alg'
