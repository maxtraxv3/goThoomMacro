package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// macroInit initializes the macro system. Called once on startup.
func macroInit() {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	if macroState.Initialized {
		return
	}

	// Create Macros directory
	macroDir := filepath.Join(dataDirPath, "Macros")
	os.MkdirAll(macroDir, 0755)

	// Write default macro file if missing
	defaultPath := filepath.Join(macroDir, "Default")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		os.WriteFile(defaultPath, []byte(macroDefaultContent()), 0644)
	}

	macroState.Initialized = true
}

// macroLoadCharacter loads macros for the current character.
func macroLoadCharacter(charName string) {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	// Kill any existing macros
	macroKillLocked()

	if charName == "" {
		return
	}

	macroDir := filepath.Join(dataDirPath, "Macros")
	os.MkdirAll(macroDir, 0755)

	// Find the character's macro file, case-insensitive
	charFile := macroFindCharFile(macroDir, charName)
	if charFile == "" {
		// No matching file — nothing to load
		logWarn("[macro] no macro file for character %q", charName)
		return
	}
	macroState.CurrentCharFile = charFile

	// Parse the character's macro file
	logWarn("[macro] loading macros from %s", filepath.Base(charFile))
	macroParseFile(charFile)

	// Log summary
	countFuncs := 0
	for m := macroState.Functions; m != nil; m = m.Next {
		countFuncs++
	}
	countExprs := 0
	for m := macroState.Expressions; m != nil; m = m.Next {
		countExprs++
	}
	countKeys := 0
	for m := macroState.Keys; m != nil; m = m.Next {
		countKeys++
	}
	countClicks := 0
	for m := macroState.Clicks; m != nil; m = m.Next {
		countClicks++
	}
	countRepl := 0
	for m := macroState.Replacements; m != nil; m = m.Next {
		countRepl++
	}
	countGlob := 0
	for m := macroState.GlobalVars; m != nil; m = m.Next {
		countGlob++
	}
	logWarn("[macro] %d functions, %d expressions, %d keys, %d clicks, %d replacements, %d globals",
		countFuncs, countExprs, countKeys, countClicks, countRepl, countGlob)

	// Execute @login function if it exists
	if !macroState.LoginExecuted {
		macroExecLogin()
		macroState.LoginExecuted = true
	}
}

// macroFindCharFile searches macroDir for a file matching name (case-insensitive).
func macroFindCharFile(macroDir, name string) string {
	target := strings.ToLower(name)
	entries, err := os.ReadDir(macroDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(e.Name()) == target {
			return filepath.Join(macroDir, e.Name())
		}
	}
	return ""
}

// macroFindIncludeOnDisk searches dir for a file matching name (case-insensitive).
func macroFindIncludeOnDisk(dir, name string) string {
	// If exact path exists, use it
	exact := filepath.Join(dir, name)
	if _, err := os.Stat(exact); err == nil {
		return exact
	}
	target := strings.ToLower(name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(e.Name()) == target {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// macroReload reloads all macros for the current character.
func macroReload() {
	charName := ""
	if macroState.CurrentCharFile != "" {
		charName = strings.TrimSuffix(
			filepath.Base(macroState.CurrentCharFile),
			filepath.Ext(macroState.CurrentCharFile),
		)
	}
	macroState.mu.Lock()
	macroState.LoginExecuted = false
	macroState.mu.Unlock()

	macroStopAll()
	macroLoadCharacter(charName)
	refreshHotkeysList()
	macroShowInfo("Macros reloaded", true)
}

// macroKill destroys all macro definitions and stops execution.
func macroKill() {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()
	macroKillLocked()
}

func macroKillLocked() {
	macroState.Expressions = nil
	macroState.Replacements = nil
	macroState.Keys = nil
	macroState.Clicks = nil
	macroState.Functions = nil
	macroState.IncludeFiles = nil
	macroState.GlobalVars = nil

	// Stop all executing macros
	for ex := macroState.Executing; ex != nil; ex = ex.Next {
		macroFinish(ex)
	}
	macroState.Executing = nil
}

// macroExecLogin executes the @login function macro if it exists.
func macroExecLogin() {
	// Must be called with lock held
	for m := macroState.Functions; m != nil; m = m.Next {
		if m.Name == "@login" {
			logWarn("[macro] executing @login")
			macroStart(m, macroFunction, "")
			return
		}
	}
	logWarn("[macro] no @login function found")
}

// macroExecFunc executes a function macro by name (without the @ prefix).
func macroExecFunc(name string) {
	if !strings.HasPrefix(name, "@") {
		name = "@" + name
	}
	macroState.mu.Lock()
	defer macroState.mu.Unlock()
	for m := macroState.Functions; m != nil; m = m.Next {
		if m.Name == name {
			ex := macroStart(m, macroFunction, "")
			for {
				if macroContinueOne(ex) {
					macroFinish(ex)
					if macroState.Executing == ex {
						macroState.Executing = ex.Next
					}
					break
				}
				if ex.Buffer != "" && strings.Contains(ex.Buffer, "\r") {
					cmd := strings.Split(ex.Buffer, "\r")[0]
					if cmd != "" {
						enqueueCommand(cmd)
					}
					ex.Buffer = ""
					continue
				}
				break
			}
			return
		}
	}
	macroShowInfo(fmt.Sprintf("macro function %q not found", name), false)
}

// macroFindExpressionMacro finds an expression macro matching the given text.
func macroFindExpressionMacro(text string) *Macro {
	for m := macroState.Expressions; m != nil; m = m.Next {
		if m.Attributes&attrIgnoreCase != 0 {
			if strings.EqualFold(m.Expression, text) {
				return m
			}
		} else {
			if m.Expression == text {
				return m
			}
		}
	}
	return nil
}

// macroFindKeyMacro finds a key macro matching the given key and modifiers.
func macroFindKeyMacro(key int, mods uint) *Macro {
	for m := macroState.Keys; m != nil; m = m.Next {
		mk := m.Key
		mm := m.Modifiers &^ 0x0040 // mask out capslock
		if mk == key && mm == (mods&^0x0040) {
			return m
		}
	}
	return nil
}

// macroFindClickMacro finds a click macro matching the given button and modifiers.
func macroFindClickMacro(button int, mods uint) *Macro {
	for m := macroState.Clicks; m != nil; m = m.Next {
		if m.Key == button && m.Modifiers == mods {
			return m
		}
	}
	return nil
}

// macroDoKey handles a key event, returning true if the key was consumed.
func macroDoKey(key int, mods uint) bool {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	// Ctrl+Escape always stops all macros
	if key == 0x1B && mods&0x0002 != 0 {
		macroStopAll()
		return true
	}

	// Don't intercept if no macros loaded
	if macroState.Keys == nil && macroState.Clicks == nil {
		return false
	}

	m := macroFindKeyMacro(key, mods)
	if m == nil {
		return false
	}

	ex := macroStart(m, macroKey, "")

	// Run the macro until it hits a pause or finishes
	for {
		if macroContinueOne(ex) {
			macroFinish(ex)
			// Remove from executing list
			if macroState.Executing == ex {
				macroState.Executing = ex.Next
			}
			break
		}
		if ex.Buffer != "" && strings.Contains(ex.Buffer, "\r") {
			// Send the buffered command
			cmd := strings.Split(ex.Buffer, "\r")[0]
			if cmd != "" {
				enqueueCommand(cmd)
			}
			ex.Buffer = ""
			// Continue after the command is sent
			continue
		}
		break
	}

	return m.Attributes&attrNoOverride == 0
}

// macroDoClick handles a click event, returning true if the click was consumed.
func macroDoClick(button int, mods uint, clickedName string) bool {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	// Stop macros if click_interrupts is set
	if macroState.EnvClickInterrupts {
		macroStopAllLocked()
	}

	m := macroFindClickMacro(button, mods)
	if m == nil {
		return false
	}

	macroClickName = clickedName
	macroClickSimpleName = simplePlayerName(clickedName)
	macroClickButton = button
	macroClickChord = ""

	ex := macroStart(m, macroKey, "")

	// Run until pause or finish
	for {
		if macroContinueOne(ex) {
			macroFinish(ex)
			if macroState.Executing == ex {
				macroState.Executing = ex.Next
			}
			break
		}
		if ex.Buffer != "" && strings.Contains(ex.Buffer, "\r") {
			cmd := strings.Split(ex.Buffer, "\r")[0]
			if cmd != "" {
				enqueueCommand(cmd)
			}
			ex.Buffer = ""
			continue
		}
		break
	}

	return m.Attributes&attrNoOverride == 0
}

// macroDoText handles text input, checking for expression macros.
// Returns true if the text was handled by a macro.
func macroDoText(text string) bool {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	if macroState.Expressions == nil {
		return false
	}

	// Extract the first word
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}
	firstWord := words[0]

	m := macroFindExpressionMacro(firstWord)
	if m == nil {
		return false
	}

	// The rest of the text (after the expression) becomes @text
	rest := strings.TrimPrefix(text, firstWord)
	rest = strings.TrimSpace(rest)

	ex := macroStart(m, macroExpression, rest)

	// Run until pause or finish
	for {
		if macroContinueOne(ex) {
			macroFinish(ex)
			if macroState.Executing == ex {
				macroState.Executing = ex.Next
			}
			break
		}
		if ex.Buffer != "" && strings.Contains(ex.Buffer, "\r") {
			cmd := strings.Split(ex.Buffer, "\r")[0]
			if cmd != "" {
				enqueueCommand(cmd)
			}
			ex.Buffer = ""
			continue
		}
		break
	}

	return true
}

// macroDoReplacement checks if the current word being typed is a replacement macro.
// Returns the replacement text if found, or empty string.
func macroDoReplacement(word string) string {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	if macroState.Replacements == nil {
		return ""
	}

	for m := macroState.Replacements; m != nil; m = m.Next {
		if m.Attributes&attrIgnoreCase != 0 {
			if strings.EqualFold(m.Replace, word) {
				return macroGetReplacementBody(m)
			}
		} else {
			if m.Replace == word {
				return macroGetReplacementBody(m)
			}
		}
	}
	return ""
}

// macroGetReplacementBody builds the replacement text from a replacement macro.
func macroGetReplacementBody(m *Macro) string {
	// Replacement macros can't contain \r (pauses forbidden)
	var buf strings.Builder
	for cmd := m.Commands; cmd != nil; cmd = cmd.Next {
		if cmd.CommandKind == cmdText {
			buf.WriteString(cmd.VarName)
			for _, p := range cmd.Params {
				buf.WriteString(" ")
				buf.WriteString(p.VarName)
			}
		}
	}
	return buf.String()
}

// macroDefaultContent returns the default macro file content.
func macroDefaultContent() string {
	return `// Default macros - included by all character macro files
// These are shorthand expression macros for common commands.

"??"    "/help "    @text "\r"
"aa"    "/action "  @text "\r"
"gg"    "/give "    @text "\r"
"ii"    "/info "    @text "\r"
"mm"    "/money\r"
"pp"    "/ponder "  @text "\r"
"rr"    "/report "  @text "\r"
"ss"    "/speak "   @text "\r"
"tt"    "/think "   @text "\r"
"ww"    "/whisper " @text "\r"
"yy"    "/yell "    @text "\r"
`
}

// ebitenKeyToMacroKey converts an ebiten key to a macro key code.
// Returns 0 if the key has no macro equivalent.
func ebitenKeyToMacroKey(k ebiten.Key) int {
	switch k {
	case ebiten.KeyEscape:
		return 0x1B
	case ebiten.KeyF1:
		return 0x0105
	case ebiten.KeyF2:
		return 0x0106
	case ebiten.KeyF3:
		return 0x0107
	case ebiten.KeyF4:
		return 0x0108
	case ebiten.KeyF5:
		return 0x0109
	case ebiten.KeyF6:
		return 0x010A
	case ebiten.KeyF7:
		return 0x010B
	case ebiten.KeyF8:
		return 0x010C
	case ebiten.KeyF9:
		return 0x010D
	case ebiten.KeyF10:
		return 0x010E
	case ebiten.KeyF11:
		return 0x010F
	case ebiten.KeyF12:
		return 0x0110
	case ebiten.KeyF13:
		return 0x0111
	case ebiten.KeyF14:
		return 0x0112
	case ebiten.KeyF15:
		return 0x0113
	case ebiten.KeyF16:
		return 0x0114
	case ebiten.KeyMinus:
		return '-'
	case ebiten.KeyBackspace:
		return 0x08
	case ebiten.KeyTab:
		return '\t'
	case ebiten.KeyEnter:
		return 0x0D
	case ebiten.KeySpace:
		return ' '
	case ebiten.KeyHome:
		return 0x01
	case ebiten.KeyPageUp:
		return 0x0B
	case ebiten.KeyDelete:
		return 0x7F
	case ebiten.KeyEnd:
		return 0x04
	case ebiten.KeyPageDown:
		return 0x0C
	case ebiten.KeyArrowUp:
		return 0x1E
	case ebiten.KeyArrowDown:
		return 0x1F
	case ebiten.KeyArrowLeft:
		return 0x1C
	case ebiten.KeyArrowRight:
		return 0x1D
	case ebiten.KeyA:
		return 'a'
	case ebiten.KeyB:
		return 'b'
	case ebiten.KeyC:
		return 'c'
	case ebiten.KeyD:
		return 'd'
	case ebiten.KeyE:
		return 'e'
	case ebiten.KeyF:
		return 'f'
	case ebiten.KeyG:
		return 'g'
	case ebiten.KeyH:
		return 'h'
	case ebiten.KeyI:
		return 'i'
	case ebiten.KeyJ:
		return 'j'
	case ebiten.KeyK:
		return 'k'
	case ebiten.KeyL:
		return 'l'
	case ebiten.KeyM:
		return 'm'
	case ebiten.KeyN:
		return 'n'
	case ebiten.KeyO:
		return 'o'
	case ebiten.KeyP:
		return 'p'
	case ebiten.KeyQ:
		return 'q'
	case ebiten.KeyR:
		return 'r'
	case ebiten.KeyS:
		return 's'
	case ebiten.KeyT:
		return 't'
	case ebiten.KeyU:
		return 'u'
	case ebiten.KeyV:
		return 'v'
	case ebiten.KeyW:
		return 'w'
	case ebiten.KeyX:
		return 'x'
	case ebiten.KeyY:
		return 'y'
	case ebiten.KeyZ:
		return 'z'
	case ebiten.KeyDigit0:
		return '0'
	case ebiten.KeyDigit1:
		return '1'
	case ebiten.KeyDigit2:
		return '2'
	case ebiten.KeyDigit3:
		return '3'
	case ebiten.KeyDigit4:
		return '4'
	case ebiten.KeyDigit5:
		return '5'
	case ebiten.KeyDigit6:
		return '6'
	case ebiten.KeyDigit7:
		return '7'
	case ebiten.KeyDigit8:
		return '8'
	case ebiten.KeyDigit9:
		return '9'
	}
	return 0
}

// ebitenButtonToMacroClick converts an ebiten mouse button to a macro click code.
func ebitenButtonToMacroClick(b ebiten.MouseButton) int {
	switch b {
	case ebiten.MouseButtonLeft:
		return 1024 // click
	case ebiten.MouseButtonRight:
		return 1025 // click2 / right-click
	case ebiten.MouseButtonMiddle:
		return 1026 // click3
	}
	return 0
}

// macroCurrentMods returns the current modifier bitmask for the macro system.
func macroCurrentMods() uint {
	var mods uint
	if ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		mods |= 0x0001 // shift
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		mods |= 0x0002 // control
	}
	if ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight) {
		mods |= 0x0004 // command
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) || ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
		mods |= 0x0008 // option
	}
	return mods
}

// macroProcessKeyEvents checks for key-based macro triggers. Called each frame
// before checkHotkeys. Returns true if a macro consumed the key.
func macroProcessKeyEvents() bool {
	if macroState.Keys == nil {
		return false
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		macroKey := ebitenKeyToMacroKey(k)
		if macroKey == 0 {
			continue
		}
		mods := macroCurrentMods()
		if macroDoKey(macroKey, mods) {
			return true
		}
	}
	return false
}

// macroProcessClickEvents checks for click-based macro triggers. Called each
// frame from the click handling code. clickedName is the player name if a
// player was clicked, empty otherwise.
func macroProcessClickEvents(clickedName string) bool {
	if macroState.Clicks == nil {
		return false
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mods := macroCurrentMods()
		if macroDoClick(ebitenButtonToMacroClick(ebiten.MouseButtonLeft), mods, clickedName) {
			return true
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mods := macroCurrentMods()
		if macroDoClick(ebitenButtonToMacroClick(ebiten.MouseButtonRight), mods, clickedName) {
			return true
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		mods := macroCurrentMods()
		if macroDoClick(ebitenButtonToMacroClick(ebiten.MouseButtonMiddle), mods, clickedName) {
			return true
		}
	}
	return false
}
