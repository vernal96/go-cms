#!/bin/sh
set -eu

base_url=${BASE_URL:-http://localhost:8080}
login=${CMS_LOGIN:-admin}
password=${CMS_PASSWORD:-admin-dev-only-2026}

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }

login_response=$(curl -fsS -X POST "$base_url/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"identifier\":\"$login\",\"password\":\"$password\"}")
token=$(printf '%s' "$login_response" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
[ -n "$token" ] || { echo "login response did not contain access_token" >&2; exit 1; }

curl -fsS "$base_url/healthz" >/dev/null
curl -fsS "$base_url/api/admin/session" -H "Authorization: Bearer $token" >/dev/null
curl -fsS "$base_url/api/admin/navigation" -H "Authorization: Bearer $token" >/dev/null
echo "PASS CMS smoke: health, login, admin session and navigation"
