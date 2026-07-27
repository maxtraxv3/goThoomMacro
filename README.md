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

### Hotkey System Enhancements

- Macro key/click bindings shown in the Hotkeys window (read-only, with source file:line)
- Bindings refresh automatically on login and macro reload

### UI Changes

- "Combine chat + console" toggle now works correctly in both directions
- WASD movement toggle in Settings > Interface (under Controls)
- Gamepad settings button in Settings > Interface (under Controls)
- Transparent window option with background color picker
- "Reload Macros" in the Actions menu
- "Untested Version" popup removed
- Default timestamp format changed to match text logs (`1/2/06 3:04:05pm`)

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

### Text Logs

- Text log files match the old ClanLord client format and location:
  `Text Logs/<CharName>/CL Log YYYY-MM-DD HH.MM.SS.txt`
- No session header or nested year/month subdirectories

### Bug Fixes

- Click macros (control-click, shift-click2, etc.) now fire correctly
- Backward `goto` always yields (matches original ClanLaw client behavior)
- `@env.textlog` macros work in `@login` (function macros default to unfriendly mode)
- `macroGetKeyByName` handles hyphenated key names (e.g., `shift-right-click`)
- `set`/`setglobal` variable names are not resolved as variables
- `\r` stripping and escape processing in macro text
- `maxCmdsPerFrame` safety limit for unfriendly mode

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

---

## Credits

- Original goThoom client by [Distortions81](https://github.com/Distortions81) ([goThoom](https://github.com/Distortions81/goThoom))
- Clan Lord is a trademark of Delta Tao Research Corporation
- Built in Go with the [Ebiten](https://ebitengine.org/) game library

## License

MIT. Game assets and "Clan Lord" are property of their respective owners; this project ships a client, not server content.
