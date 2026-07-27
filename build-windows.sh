#!/usr/bin/env bash
set -euo pipefail

VERSION="${CLIENT_VERSION:-dev}"
echo "Building gothoom for Windows x86_64 (version: $VERSION)"

mkdir -p build

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build \
    -trimpath \
    -ldflags "-s -w -H=windowsgui -X main.clientVersion=$VERSION" \
    -o build/gothoom.exe .

echo "Output: build/gothoom.exe"
