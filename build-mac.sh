#!/usr/bin/env bash
set -euo pipefail

VERSION="${CLIENT_VERSION:-dev}"
ARCH="${GOARCH:-arm64}"

if [ "$ARCH" = "arm64" ]; then
  echo "Building gothoom for macOS Apple Silicon (version: $VERSION)"
else
  echo "Building gothoom for macOS Intel (version: $VERSION)"
fi

mkdir -p build

# Requires osxcross (see build-scripts/build_binaries.sh for setup).
CC=""
CXX=""
case "$ARCH" in
  arm64) CC="oa64-clang"; CXX="oa64-clang++" ;;
  amd64) CC="o64-clang"; CXX="o64-clang++" ;;
esac

EXTRA_ARGS=()
if command -v go-winres >/dev/null 2>&1; then
  :
fi

# Use metal tags for macOS.
CGO_ENABLED=1 GOOS=darwin GOARCH="$ARCH" \
  CC="$CC" CXX="$CXX" \
  go build \
    -trimpath \
    -tags metal \
    -ldflags "-s -w -X main.clientVersion=$VERSION" \
    -o build/gothoom .

echo "Output: build/gothoom"
echo ""
echo "To create a .app bundle, see build-scripts/build_binaries.sh"
