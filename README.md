# goThoom (Macro Fork)

An alternate version of [goThoom](https://github.com/Distortions81/goThoom) by [Distortions81](https://github.com/Distortions81), an open-source client for the classic **[Clan Lord](https://www.deltatao.com/clanlord/)** MMORPG.

This fork builds on the original goThoom client with a focus on full Clan Lord macro system compatibility and quality-of-life improvements. All original features and credits belong to Distortions81.

---

## Changes on top of the original goThoom

### Full Clan Lord Macro System

- File-based macro language with expressions, replacements, key/click/function macros
- Variables (`@my.*`, `@selplayer.*`, `@env.*`, `@click.*`, `@text.*`, `@random`)
- Flow control: `goto`, `if`/`else if`/`else`/`end if`, `random`/`end random`
- Timing: `pause`, `timing`, `atime` with ack-frame synchronization
- `set`/`setglobal` variables, `call` function macros
- Case-insensitive file lookups (works on Linux)
- Drop-in compatible with existing Clan Lord macro files
- Macro hotkeys displayed in the Hotkeys UI with source file and line number attribution

### Client Slash Commands

Client-side slash commands are handled locally (never sent to the server) and work from the chat input bar, the macro/script command queue, and speech-to-text:

- `/move <n|ne|e|se|s|sw|w|nw|stop> [run|walk]` — walks in the given compass direction. A bare `/move` or `/move stop` stops. Append `run` for full-speed movement (defaults to the keyboard walk speed). The walk persists until stopped, so you can move in one direction while chatting.
- `/select <player|@next|@prev|@first|@last>` — selects a player (by exact, prefix, or substring name) or cycles through the online player list.
- `/selectitem <name>` — selects an inventory item by exact base name, then prefix/substring match; a bare `/selectitem` clears the selection.
- `/label <player> [label]` — assigns a friend label (0-10, or a label name) to a player; label `0` clears it.
- `/wholabel [label]` — lists friends, optionally filtered to a specific label.
- `/pref <movement hold|toggle> | <shownames|timestamps|message fallen|message clanning> <on|off> | <soundvolume|bardvolume|maxnight> <0-100>` — changes client settings from the command line; preferences without a client equivalent report that they are not implemented.
- `/record [on|off]` — toggles movie recording (a bare `/record` toggles).
- `/notts add|remove <name> | list` — manages the speech-to-text (TTS) blocklist.

### Hotkey System Enhancements

- Macro key/click bindings shown in the Hotkeys window (read-only, with source file:line)
- Bindings refresh automatically on login and macro reload

### UI Changes

- "Combine chat + console" toggle now works correctly in both directions
- Players window title shows live counts that stay visible regardless of scroll position: `Players   Online: N   Shared: N   Sharing: N`
- Inventory window title shows slot usage that stays visible regardless of scroll position: `Inventory   Slots: N/M`, plus `(N free)` when fewer than 5 slots remain
- WASD movement toggle in Settings > Interface (under Controls)
- Gamepad settings button in Settings > Interface (under Controls)
- Transparent window option with background color picker
- "Reload Macros" in the Actions menu
- "Untested Version" popup removed
- Default timestamp format changed to match text logs (`1/2/06 3:04:05pm`)
- Color-coded console messages: yells, ponders, thinks, narrates, actions, coins — each with individually customizable color via HSV color picker in Settings. Click "Customize Colors..." after enabling. Defaults: yell=yellow, action=red, think/narrate=green, ponder=purple, coin=gold. Enabled by default, toggle in Settings.
- TTS blocklist management in Advanced Settings (add blocked speakers, "More Piper voices..." link to download additional voices)

### Speech to Text (Vosk)

Offline speech-to-text powered by Vosk, with no cloud service:

- **Dictate to chat**: say what you want, and it's sent as a chat message
- **Voice commands**: say a phrase from `data/stt_commands.txt` to run a command or macro function (e.g. `sit down = @sit`)
- **Mic selection**: pick your microphone in Settings (the OS default is sometimes wrong)
- **Model selection**: choose your Vosk model in Settings > Speech to Text. The default full US English model (`vosk-model-en-us-0.22`, ~1.8 GB) recognizes most reliably; smaller models such as `vosk-model-en-us-0.22-lgraph` (~130 MB) and `vosk-model-small-en-us-0.15` (~40 MB) are available for faster downloads at the cost of accuracy
- **Activation**: "Start/Stop listening" button in Settings, plus an optional hotkey — either push-to-talk (hold) or toggle
- **Auto-gain**: quiet microphones are boosted automatically so Vosk hears them clearly
- **Noise gate**: an adaptive gate feeds Vosk silence between speech, and standalone filler words ("the", "and", "um") are dropped, preventing hallucinated words from reaching chat
- **Mishear aliases**: add phrases to `data/stt_commands.txt` that map common mishearings to the same command (e.g. `poe's=\POSE {text}`)
- **Setup**: tick "Download Vosk files" in the Downloads window, or run `build-scripts/download_vosk.sh`. This fetches the native `libvosk` library into `data/vosk/` and a model (default `vosk-model-en-us-0.22`, ~1.8 GB)
- Hotkey format matches macro key names, e.g. `shift-f9` or `command-space`
- Only enabled when the `STTEnabled` setting is on; requires cgo (unavailable in the web/wasm build)

### Gamepad / Controller Support

- Full gamepad input with per-stick walk and cursor movement
- Configurable deadzones for both sticks
- Button-to-command mapping (16 mappable buttons): enter a command like `/sit` or a macro function name
- Click bindings (Click1/2/3) via gamepad buttons
- Auto-sizing Gamepad window with visual controller layout
- Visual controller display with:
  - Controller body outline with grips
  - Analog sticks with deadzone rings and live position dots (green = active, red = in deadzone)
  - D-pad cross that lights up per direction
  - ABXY face buttons in standard diamond layout
  - LB/RB bumpers, LT/RT triggers, Start/Back, L3/R3
  - Every button shows its name and ebiten button number
  - Live axis bar gauges showing all axis values with center markers
- Button tester: press any button and see its name + number for easy mapping
- Stick mapping dropdowns auto-detect axis count from the connected controller
- Auto-detection of controller connect/disconnect with refresh button
- Numpad keys (`Numpad0`-`Numpad9`, `NumpadEnter`, `NumpadAdd`, etc.) mapped for macros
- Higher mouse buttons (`click4`-`click8`) detected

### Text Logs

- Text log files match the old ClanLord client format and location:
  `Text Logs/<CharName>/CL Log YYYY-MM-DD HH.MM.SS.txt`
- No session header or nested year/month subdirectories

### Hotkey Conflict Detection

- Hotkeys that conflict with macro key bindings or enabled script hotkeys are automatically disabled at runtime
- Conflicts resolve automatically when the conflicting macro or script is removed/disabled
- Conflict-disabled hotkeys are visually marked in the Hotkeys UI (outlined buttons)

### Bug Fixes

- `/move` (and the macro `move` command) now actually moves the character when issued from chat, macros, or speech-to-text: the command walk is now routed through the same movement input as keyboard and mouse steering, instead of being dropped
- Click macros (control-click, shift-click2, etc.) now fire correctly
- Backward `goto` always yields (matches original ClanLaw client behavior)
- `@env.textlog` macros work in `@login` (function macros default to unfriendly mode)
- `macroGetKeyByName` handles hyphenated key names (e.g., `shift-right-click`)
- `set`/`setglobal` variable names are not resolved as variables
- `\r` stripping and escape processing in macro text
- `maxCmdsPerFrame` safety limit for unfriendly mode
- TTS now works when "Combine chat + console" is enabled (chat bubbles always route through `chatMessage()`)
- TTS skips consecutive duplicate messages (same text won't be spoken twice in a row)
- Auto-blocks your own character from TTS on login (added to TTS blocklist)
- TTS blocklist management UI in Advanced Settings (add names, view blocked list, link to download more Piper voices)
- Numpad keys (numpad-0 through numpad-9) now work in key macros — `keyNameToCode` was missing digit entries, causing `numpad-1` etc. to silently fail to parse
- Inventory item names bug fixed: item custom names (e.g., whatzit creature info) now display correctly in the inventory
- Equipped location text (e.g., `[Feet]`) in the inventory window is no longer cut off and is now fully readable
- Speech-to-text (Vosk) uses a prebuilt native library loaded at runtime, so no Vosk source build is needed
- Speech-to-text now hears audio correctly: mic samples were being fed at ±1.0 instead of the int16 scale (±32767) Vosk's float API expects, which made the input ~32768x too quiet (Vosk heard silence)

### Auto-Update (Client Binary)

- Checks GitHub releases (`maxtraxv3/goThoomMacro`) for newer semver-tagged releases
- Prompts with "Update Now" / "Later" popup when a new version is found
- Downloads the platform-appropriate binary, replaces the running executable, and restarts
- Enabled only when built with `-ldflags "-X main.clientVersion=v1.0.0"` (set to your semver tag)
- Disabled by default when `clientVersion` is `dev`

### Game Asset Updates (CL_Images / CL_Sounds)

- Game assets (images, sounds) are downloaded from the official Clan Lord servers (Delta Tao)
- Incremental patching enabled by default (`-experimental`): downloads only the diff between your current version and the latest, saving bandwidth
- Falls back to full archive download if incremental patch is unavailable
- Skips download entirely if local files are already up to date or newer than the server version
- Remote `versions.json` is checked at startup to get the actual latest CL version, so updates aren't held back by outdated game server responses

---

## Original goThoom Features

All original features are preserved, including:

- Smooth motion interpolation and animation frame blending
- Texture processing for CRT-like dithering correction
- High-quality audio resampling and music synthesis
- Built-in AI text-to-speech
- Dark/light/fun themes with custom palette support
- WASD and modern input schemes
- Scripting system via yaegi Go interpreter
- Cross-platform: Windows, macOS, Linux

---

## Building

1. Install system packages:
   ```bash
   sudo apt-get install -y build-essential libgl1-mesa-dev libglu1-mesa-dev \
     xorg-dev libxrandr-dev libasound2-dev libgtk-3-dev xdg-utils
   ```
2. Install Go 1.26.5 from [go.dev](https://go.dev/dl/).
3. Download and extract the prebuilt dependency bundle:
   ```bash
   curl -LO https://m45sci.xyz/u/dist/goThoom/gothoom_deps.tar.gz
   tar -xzf gothoom_deps.tar.gz
   ```
4. Fetch Go modules and build:
   ```bash
   cd gothoom
   go mod download
   go build
   ```
5. To enable auto-update, pass the version via ldflags:
   ```bash
   go build -ldflags "-s -w -X main.clientVersion=v1.0.0" -o gothoom .
   ```
   If `clientVersion` is left as the default (`dev`), auto-update is disabled.

### Speech to Text assets

Speech-to-text needs the Vosk model and native library in `data/vosk/`. Tick
"Download Vosk files" in the client's Downloads window, or run:

```bash
build-scripts/download_vosk.sh
```

The library is extracted from official Vosk release archives; no source build
is required.

### Cross-compilation (Linux, Windows, macOS, RPi, Web)

Use `build-scripts/build_binaries.sh` or `build-scripts/build_binaries_local.sh` to build for all targets at once. These scripts:
- Install any missing build dependencies (osxcross for macOS, mingw-64 for Windows via cgo, etc.)
- Build binaries for `linux:amd64`, `linux:arm64` (Raspberry Pi), `windows:amd64`, `darwin:arm64` (Apple Silicon), `darwin:amd64` (Intel), and `js:wasm`
- **Auto-cleanup**: any packages installed by the script that were not already on your system are automatically removed after the build completes

---

## Credits

- Original goThoom client by [Distortions81](https://github.com/Distortions81) ([goThoom](https://github.com/Distortions81/goThoom))
- Clan Lord is a trademark of Delta Tao Research Corporation
- Built in Go with the [Ebiten](https://ebitengine.org/) game library

## Third-Party Libraries

- **Ebitengine** — Apache-2.0 — Hajime Hoshi (https://github.com/hajimehoshi/ebiten)
- **go-humanize** — MIT — Dustin Sallings (https://github.com/dustin/go-humanize)
- **piper** — MIT — Amity Bell (https://github.com/fresh-cut/piper)
- **gopacket** — BSD-3-Clause — Google, Inc.; Andreas Krennmair (https://github.com/google/gopacket)
- **durafmt** — MIT — Wesley Hill (https://github.com/hako/durafmt)
- **rich-go** — MIT — Hugo Lageneste (https://github.com/hugolgst/rich-go)
- **sizedwaitgroup** — MIT — Rémy Mathieu (https://github.com/remeh/sizedwaitgroup)
- **go-meltysynth** — MIT — Nobuaki Tanaka (https://github.com/sinshu/go-meltysynth)
- **dialog** — ISC — sqweek and contributors (https://github.com/sqweek/dialog)
- **dark-mode-go** — MIT — Thiago Kenji Okada (https://github.com/thiagokokada/dark-mode-go)
- **clipboard** — MIT — Changkun Ou (https://github.com/golang-design/clipboard)
- **x/crypto** — BSD-3-Clause — The Go Authors (https://github.com/golang/crypto)
- **x/text** — BSD-3-Clause — The Go Authors (https://github.com/golang/text)
- **x/time** — BSD-3-Clause — The Go Authors (https://github.com/golang/time)
- **beeep** — BSD-2-Clause — Milan Nikolic (https://github.com/gen2brain/beeep)
- **malgo** — The Unlicense (public domain) — Eugene Pirogov; wraps miniaudio (https://github.com/gen2brain/malgo)
- **browser** — BSD-2-Clause — Dave Cheney (https://github.com/pkg/browser)
- **open-golang** — MIT — skratchdot (https://github.com/skratchdot/open-golang)
- **spellchecker** — MIT — cyradin (https://github.com/f1monkey/spellchecker)
- **yaegi** — Apache-2.0 — Traefik Labs (https://github.com/traefik/yaegi)
- **x/image** — BSD-3-Clause — The Go Authors (https://github.com/golang/image)

### Speech-to-Text

- **Vosk** — Apache-2.0 — Alpha Cephei (https://github.com/alphacep/vosk-api)
- **Vosk models** (small-en-us-0.15, en-us-0.22-lgraph, en-us-0.22) — Apache-2.0 — Alpha Cephei (https://alphacephei.com/vosk/models)
- **miniaudio** — Public Domain / MIT-0 — David Reid (https://github.com/mackron/miniaudio)

## License

MIT. Game assets and "Clan Lord" are property of their respective owners; this project ships a client, not server content.
