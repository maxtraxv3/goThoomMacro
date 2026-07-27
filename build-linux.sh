#!/usr/bin/env bash
set -euo pipefail

VERSION="${CLIENT_VERSION:-dev}"
echo "Building gothoom for Linux x86_64 (version: $VERSION)"

go build \
  -trimpath \
  -ldflags "-s -w -X main.clientVersion=$VERSION" \
  -o build/gothoom .

echo "Output: build/gothoom"
