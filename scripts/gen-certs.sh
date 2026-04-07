#!/usr/bin/env bash
# Generate self-signed TLS certificates for nginx
# Usage: ./scripts/gen-certs.sh

set -euo pipefail

CERTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/certs"
mkdir -p "$CERTS_DIR"

if [[ -f "$CERTS_DIR/server.crt" && -f "$CERTS_DIR/server.key" ]]; then
  echo "[gen-certs] Certificates already exist at $CERTS_DIR — skipping."
  echo "            Delete them manually to regenerate."
  exit 0
fi

echo "[gen-certs] Generating self-signed certificate in $CERTS_DIR ..."

openssl req -x509 \
  -newkey rsa:4096 \
  -keyout "$CERTS_DIR/server.key" \
  -out    "$CERTS_DIR/server.crt" \
  -days   825 \
  -nodes \
  -subj   "/C=US/ST=Local/L=Local/O=LLMProxy/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

chmod 600 "$CERTS_DIR/server.key"
chmod 644 "$CERTS_DIR/server.crt"

echo "[gen-certs] Done:"
echo "  Certificate : $CERTS_DIR/server.crt"
echo "  Private key : $CERTS_DIR/server.key"
