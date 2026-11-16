#!/bin/sh
# The Caddyfile needs a bcrypt hash, so hash FAI_AUTH_PASSWORD before Caddy reads it.
set -eu

: "${FAI_DOMAIN:?set FAI_DOMAIN in .env (for example fai.example.com)}"
: "${FAI_AUTH_PASSWORD:?set FAI_AUTH_PASSWORD in .env}"

FAI_AUTH_USER="${FAI_AUTH_USER:-admin}"
FAI_AUTH_HASH=$(caddy hash-password --plaintext "$FAI_AUTH_PASSWORD")
export FAI_AUTH_USER FAI_AUTH_HASH

exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
