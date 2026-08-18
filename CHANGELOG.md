# Changelog

## v1.0.4

### New Features

- **Selection color setting**: The `/select` player marker (circle on the ground) and players window highlight now use a dedicated "Selection" color (blue default) instead of the global accent color. Customize via Settings > Customize Colors, or reset to defaults.
- **`/select` clears on miss**: Using `/select <name>` with a name that doesn't match any player now clears the current selection instead of silently doing nothing.

### Bug Fixes

- **Lightning bolt rendering**: Added support for the `kPictDefFlagRandomAnimation` (0x0004) flag from the old ClanLord client. Images with this flag (lightning effects) now pick a random animation frame each draw call instead of cycling deterministically, matching the erratic flickering behavior of the original client.

## v1.0.3

### New Features

- **Text selection and copy**: Click and drag to select text in the console, chat, and any text window. Press Ctrl+C to copy the selected text to the clipboard.
- **Clickable URLs**: HTTP and HTTPS links in console and chat messages are now highlighted with a blue underline and open in your default browser when clicked.
- **Clan marker color**: Messages with clan bullet markers (`•` or `*`) are colored with a new "Clan" color setting (brown-red default). Customize via Settings > Customize Colors, or reset to defaults.
- **Player list accuracy**: The players window no longer hides players after a timeout. Players are now marked offline only after a `/who` scan confirms they have logged off (matching old Clan Lord client behavior).

### Bug Fixes

- **`/move` command actually works**: Command-walk (from chat, macros, or speech-to-text) now routes through the same movement system as keyboard and mouse steering, instead of being silently dropped.
- **Macro keys in chat input**: Typing in the chat input bar no longer accidentally triggers macro hotkeys. A new `macroKeyJustFired` flag suppresses text triggers during the same frame a macro key is processed.
- **`/move` and macro `move` cancel on click**: Clicking in the game world now cancels an active `/move` or macro `move` command, just as it cancels manual keyboard movement.
- **Player list logoff detection**: Players who log off are now detected when the message "is not in the lands" appears, in addition to the existing "has departed" check.

## v1.0.2

_(Previous releases — see git history for details.)_
