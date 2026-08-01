Speech-to-text uses Vosk. Place the native library in this folder:

- Linux x86_64 / arm64:     libvosk.so
- macOS (Intel / Apple):    libvosk.dylib
- Windows:                  libvosk.dll

The library is extracted from the official Vosk release wheel; no source
build is required. See build-scripts/download_vosk.sh (regenerate/download)
and the README section "Speech to Text" for model downloads.

Models go in their own subfolder here, e.g. vosk-model-small-en-us-0.15/,
chosen with the "STT Model" setting. The model downloader fetches models
from the Vosk releases (alphacephei/vosk-model-*) on GitHub.
