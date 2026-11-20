#!/usr/bin/env bash
set -e
echo "⚡ Installing ImmichPulse TUI Dashboard..."
if ! command -v go &> /dev/null; then
    sudo apt-get update && sudo apt-get install -y golang-go
fi
go mod tidy
CGO_ENABLED=0 go build -ldflags="-s -w" -o immich-pulse main.go
sudo mv immich-pulse /usr/local/bin/immich-pulse
sudo chown root:root /usr/local/bin/immich-pulse
sudo chmod 755 /usr/local/bin/immich-pulse
if [ ! -e /usr/local/bin/immich ] || [ -L /usr/local/bin/immich ]; then
    sudo ln -sf /usr/local/bin/immich-pulse /usr/local/bin/immich
fi
echo "✅ Installed! Run 'immich' or 'immich-pulse' anytime."
