#!/usr/bin/env bash
set -euo pipefail

# Downloads the native libvosk library and the default speech-to-text model
# into data/vosk/ for the current platform.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VOSK_DIR="$ROOT_DIR/data/vosk"
mkdir -p "$VOSK_DIR"

VOSK_RELEASE="https://github.com/alphacep/vosk-api/releases/download/v0.3.45"
VOSK_MAC_URL="https://files.pythonhosted.org/packages/44/19/5e8299237bc2005c3d155d3b48adba6fd6484465ad5c970302fc1d37947d/vosk-0.3.44-py3-none-macosx_10_6_universal2.whl"

MODEL="vosk-model-en-us-0.22"
MODEL_URL="https://alphacephei.com/vosk/models/$MODEL.zip"

download_lib() {
  local os
  os="$(uname -s)"
  case "$os" in
    Linux)
      case "$(uname -m)" in
        x86_64) name="vosk-linux-x86_64-0.3.45.zip" ;;
        aarch64) name="vosk-linux-aarch64-0.3.45.zip" ;;
        armv7l) name="vosk-linux-armv7l-0.3.45.zip" ;;
        *) echo "Unsupported arch: $(uname -m)" >&2; exit 1 ;;
      esac
      dest="libvosk.so"
      ;;
    Darwin)
      name="vosk-0.3.44-py3-none-macosx_10_6_universal2.whl"
      dest="libvosk.dylib"
      ;;
    *)
      echo "Unsupported OS: $os" >&2
      exit 1
      ;;
  esac

  if [ -f "$VOSK_DIR/$dest" ]; then
    echo "$dest already present"
    return
  fi

  tmp="$VOSK_DIR/vosk_lib_dl"
  mkdir -p "$tmp"
  case "$os" in
    Darwin)
      echo "Downloading $name..."
      curl -L "$VOSK_MAC_URL" -o "$tmp/$name"
      unzip -oq "$tmp/$name" -d "$tmp"
      mv "$tmp/vosk/libvosk.dyld" "$VOSK_DIR/libvosk.dylib"
      ;;
    Linux)
      echo "Downloading $name..."
      curl -L "$VOSK_RELEASE/$name" -o "$tmp/$name"
      unzip -oq "$tmp/$name" -d "$tmp"
      find "$tmp" -name 'libvosk.so' -exec mv {} "$VOSK_DIR/$dest" \;
      ;;
  esac
  rm -rf "$tmp"
}

if [ ! -d "$VOSK_DIR/$MODEL" ]; then
  tmp="$VOSK_DIR/$MODEL.zip"
  echo "Downloading model $MODEL..."
  curl -L "$MODEL_URL" -o "$tmp"
  unzip -oq "$tmp" -d "$VOSK_DIR"
  rm -f "$tmp"
fi

download_lib

echo "Vosk files ready in $VOSK_DIR"
