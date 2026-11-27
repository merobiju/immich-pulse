#!/usr/bin/env bash
set -euo pipefail
REPO="merobiju/immich-pulse"
BINARY_NAME="immich-pulse"
INSTALL_DIR="/usr/local/bin"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7l) ARCH="arm" ;;
  i386|i686) ARCH="386" ;;
  *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}-${OS}-${ARCH}"
echo "⚡ Downloading ImmichPulse for ${OS}/${ARCH}..."
TMP_FILE=$(mktemp)
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
  echo "⚠️ Pre-built binary not found. Building from source..."
  if command -v go >/dev/null 2>&1; then
    go build -ldflags="-s -w" -o "$TMP_FILE" .
  else
    echo "❌ Go not found and binary download failed."
    exit 1
  fi
fi
chmod +x "$TMP_FILE"
echo "📦 Installing to ${INSTALL_DIR}..."
sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
sudo ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/immich"
echo "✅ Installed! Run 'immich' or 'immich-pulse' to launch."
