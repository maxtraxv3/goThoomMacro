package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// macroResolveVariable resolves a variable reference like "@my.name" or "@env.debug".
// Returns the value and whether it was found.
func macroResolveVariable(name string) (string, bool) {
	// Check obsolete variable names first
	if newName, ok := obsoleteVarRemap[name]; ok {
		name = newName
	}

	// Handle .num_words suffix on any variable (e.g., @my.name.num_words, @env.textlog.num_words)
	if strings.HasSuffix(name, ".num_words") {
		base := name[:len(name)-len(".num_words")]
		val, ok := macroResolveVariable(base)
		if !ok {
			return "0", false
		}
		return fmt.Sprintf("%d", len(strings.Fields(val))), true
	}

	// Handle .word[N] suffix on any variable (e.g., @my.selected_item.word[1])
	if wordIdx, base, ok := parseWordIndex(name); ok {
		val, _ := macroResolveVariable(base)
		words := strings.Fields(val)
		if wordIdx < len(words) {
			return words[wordIdx], true
		}
		return "", false
	}

	// Extract namespace
	ns, subfield := splitVarRef(name)
	nsID := varNamespaceToID[ns]
	if nsID == 0 {
		// Not a built-in namespace - check user variables (local then global)
		return macroGetUserVariable(name)
	}

	switch nsID {
	case varEnv:
		return macroGetEnvVar(subfield)
	case varMy:
		return macroGetMyVar(subfield)
	case varSelPlayer:
		return macroGetSelPlayerVar(subfield)
	case varClick:
		return macroGetClickVar(subfield)
	case varRandom:
		return fmt.Sprintf("%d", rand.Intn(10000)), true
	}

	return "", false
}

// parseWordIndex checks if name ends with ".word[N]" and returns (index, base, true).
func parseWordIndex(name string) (int, string, bool) {
	idx := strings.Index(name, ".word[")
	if idx < 0 || !strings.HasSuffix(name, "]") {
		return 0, "", false
	}
	nStr := name[idx+6 : len(name)-1]
	n, err := strconv.Atoi(nStr)
	if err != nil {
		return 0, "", false
	}
	return n, name[:idx], true
}

// splitVarRef splits "@namespace.subfield" into ("@namespace", "subfield").
// Also handles "@text.word[0]" style references.
func splitVarRef(name string) (string, string) {
	// Find the first dot after the @ prefix
	if !strings.HasPrefix(name, "@") {
		return name, ""
	}
	dotIdx := strings.Index(name[1:], ".")
	if dotIdx < 0 {
		return name, ""
	}
	return name[:dotIdx+1], name[dotIdx+2:]
}

// macroGetEnvVar returns an environment variable value.
func macroGetEnvVar(subfield string) (string, bool) {
	id := envVarNameToID[strings.ToLower(subfield)]
	switch id {
	case envKeyInterrupts:
		return fmt.Sprintf("%t", macroState.EnvKeyInterrupts), true
	case envClickInterrupts:
		return fmt.Sprintf("%t", macroState.EnvClickInterrupts), true
	case envDebug:
		return fmt.Sprintf("%t", macroState.EnvDebug), true
	case envEcho:
		return fmt.Sprintf("%t", macroState.EnvEcho), true
	case envUnfriendly:
		return fmt.Sprintf("%t", macroState.EnvUnfriendly), true
	case envTextLog:
		return macroState.TextLogBuffer, true
	}
	return "", false
}

// macroGetMyVar returns a player ("@my.*") variable value.
func macroGetMyVar(subfield string) (string, bool) {
	id := playerVarNameToID[subfield]
	switch id {
	case pvarName:
		return playerName, true
	case pvarSimpleName:
		return simplePlayerName(playerName), true
	case playerLeftItem:
		return macroGetEquippedItemName(kItemSlotLeftHand), true
	case playerRightItem:
		return macroGetEquippedItemName(kItemSlotRightHand), true
	case playerRace:
		stateMu.Lock()
		// Race isn't stored in drawState; look for self in liveMobs
		stateMu.Unlock()
		return "", true
	case playerGender:
		return "", true
	case playerHealth:
		stateMu.Lock()
		hp := state.hp
		stateMu.Unlock()
		return fmt.Sprintf("%d", hp), true
	case playerBalance:
		stateMu.Lock()
		bal := state.balance
		stateMu.Unlock()
		return fmt.Sprintf("%d", bal), true
	case playerMagic:
		stateMu.Lock()
		sp := state.sp
		stateMu.Unlock()
		return fmt.Sprintf("%d", sp), true
	case playerSharesIn:
		return macroListShares(false), true
	case playerSharesOut:
		return macroListShares(true), true
	case playerSelectedItem:
		return macroGetSelectedItemName(), true
	default:
		// Equipment slots: playerForehead through playerHead
		if id >= playerForehead && id <= playerHead {
			return macroGetEquipmentSlotVar(id), true
		}
	}
	return "", false
}

// macroGetSelPlayerVar returns a selected player variable value.
func macroGetSelPlayerVar(subfield string) (string, bool) {
	id := playerVarNameToID[subfield]
	switch id {
	case pvarName:
		return macroSelPlayerName, true
	case pvarSimpleName:
		return macroSelPlayerSimpleName, true
	}
	return "", false
}

// macroGetClickVar returns a click variable value.
func macroGetClickVar(subfield string) (string, bool) {
	id := playerVarNameToID[subfield]
	switch id {
	case pvarName:
		return macroClickName, true
	case pvarSimpleName:
		return macroClickSimpleName, true
	}
	// button and chord aren't standard player vars, handle them specially
	switch subfield {
	case "button":
		return fmt.Sprintf("%d", macroClickButton), true
	case "chord":
		return macroClickChord, true
	}
	return "", false
}

// macroResolveBrackets resolves [expr] patterns in a variable name.
// Each expr is looked up as a variable and replaced with its value.
// The brackets themselves are preserved. If the expr is not found as a
// variable, the literal text inside brackets is kept as-is.
// For example, name[thnka] with thnka=3 becomes name[3].
func macroResolveBrackets(name string) string {
	for {
		start := strings.IndexByte(name, '[')
		if start < 0 {
			return name
		}
		end := strings.IndexByte(name[start:], ']')
		if end < 0 {
			return name
		}
		end += start
		inner := name[start+1 : end]
		val, ok := macroFindGlobalVariable(inner)
		if !ok {
			return name
		}
		name = name[:start+1] + val + name[end:]
	}
}

// macroGetUserVariable resolves a user-defined variable (local first, then global).
func macroGetUserVariable(name string) (string, bool) {
	// Check local variables of the currently executing macro
	// (this is called from the exec context, so we check the global state's executing macros)
	// For simplicity, we only check global variables here.
	// Local variable resolution happens during macro execution.
	return macroFindGlobalVariable(name)
}

// macroFindGlobalVariable finds a variable in the global variable list.
func macroFindGlobalVariable(name string) (string, bool) {
	name = macroResolveBrackets(name)
	for m := macroState.GlobalVars; m != nil; m = m.Next {
		if m.VarName == name {
			return m.VarValue, true
		}
	}
	return "", false
}

// macroSetVariable sets a variable value (local if in a macro context, global otherwise).
func macroSetVariable(name, value string, global bool) {
	name = macroResolveBrackets(name)
	if global {
		macroSetGlobalVariable(name, value)
	} else {
		// For top-level set commands, use global scope
		macroSetGlobalVariable(name, value)
	}
}

// macroSetGlobalVariable sets a variable in the global variable list.
func macroSetGlobalVariable(name, value string) {
	// Check obsolete names
	if newName, ok := obsoleteVarRemap[name]; ok {
		name = newName
	}
	// Resolve the value (handle variable references and quotes)
	resolved := macroResolveExpr(value)
	name = macroResolveBrackets(name)
	// Find or create
	for m := macroState.GlobalVars; m != nil; m = m.Next {
		if m.VarName == name {
			m.VarValue = resolved
			return
		}
	}
	macroState.GlobalVars = &Macro{
		Kind:       macroVariable,
		VarName:    name,
		VarValue:   resolved,
		Next:       macroState.GlobalVars,
	}
}

// macroSetLocalVariable sets a variable in the executing macro's local scope.
func macroSetLocalVariable(ex *ExecutingMacro, name, value string) {
	resolved := macroResolveExpr(value)
	name = macroResolveBrackets(name)
	// Find or create in local vars
	for m := ex.Vars; m != nil; m = m.Next {
		if m.VarName == name {
			m.VarValue = resolved
			return
		}
	}
	ex.Vars = &Macro{
		Kind:     macroVariable,
		VarName:  name,
		VarValue: resolved,
		Next:     ex.Vars,
	}
}

// macroGetLocalVariable finds a variable in the executing macro's local scope,
// falling back to global scope.
func macroGetLocalVariable(ex *ExecutingMacro, name string) (string, bool) {
	// Check obsolete names
	if newName, ok := obsoleteVarRemap[name]; ok {
		name = newName
	}
	name = macroResolveBrackets(name)
	// Local scope first
	for m := ex.Vars; m != nil; m = m.Next {
		if m.VarName == name {
			return m.VarValue, true
		}
	}
	// Then global scope
	return macroFindGlobalVariable(name)
}

// macroProcessEscapes converts \r, \n, \t, \", \', \\ in macro text.
func macroProcessEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'r':
				b.WriteByte('\r')
				i++
			case 'n':
				b.WriteByte('\n')
				i++
			case 't':
				b.WriteByte('\t')
				i++
			case '"', '\'', '\\':
				b.WriteByte(s[i+1])
				i++
			default:
				b.WriteByte('\\')
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// macroResolveExpression resolves a macro command parameter, handling
// variable references, quotes, and text operations.
func macroResolveExpression(ex *ExecutingMacro, expr string) string {
	s := strings.TrimSpace(expr)
	if len(s) == 0 {
		return s
	}

	// Strip matching quotes and process escape sequences
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return macroProcessEscapes(s[1 : len(s)-1])
	}

	// Variable reference with @ prefix
	if strings.HasPrefix(s, "@") {
		// Check for text operations: @text.word[N], @text.letter[N], etc.
		if strings.HasPrefix(s, "@text.") {
			return macroResolveTextOp(ex, s)
		}
		val, ok := macroGetLocalVariable(ex, s)
		if ok {
			return val
		}
		// Try built-in namespace variables (@my, @env, @selplayer, @click, @random)
		if val, ok := macroResolveVariable(s); ok {
			return val
		}
		return s
	}

	// Bare variable name (no @, no quotes) — look up as user variable
	if val, ok := macroGetLocalVariable(ex, s); ok {
		return val
	}
	if val, ok := macroFindGlobalVariable(s); ok {
		return val
	}

	return s
}

// macroResolveTextOp handles @text.word[N], @text.letter[N], @text.num_words, @text.num_letters.
func macroResolveTextOp(ex *ExecutingMacro, expr string) string {
	// Get the base text from @text
	textVal, _ := macroGetLocalVariable(ex, "@text")

	suffix := strings.TrimPrefix(expr, "@text.")
	if strings.HasPrefix(suffix, "word[") {
		idxStr := strings.TrimSuffix(strings.TrimPrefix(suffix, "word["), "]")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 {
			return ""
		}
		words := strings.Fields(textVal)
		if idx >= len(words) {
			return ""
		}
		return words[idx]
	}
	if strings.HasPrefix(suffix, "letter[") {
		idxStr := strings.TrimSuffix(strings.TrimPrefix(suffix, "letter["), "]")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 {
			return ""
		}
		runes := []rune(textVal)
		if idx >= len(runes) {
			return ""
		}
		return string(runes[idx])
	}
	if suffix == "num_words" {
		return fmt.Sprintf("%d", len(strings.Fields(textVal)))
	}
	if suffix == "num_letters" {
		return fmt.Sprintf("%d", len([]rune(textVal)))
	}
	return ""
}

// macroGetEquippedItemName returns the name of the item in the given slot.
func macroGetEquippedItemName(slot int) string {
	items := getInventory()
	if clImages == nil {
		return "Nothing"
	}
	for _, it := range items {
		if !it.Equipped {
			continue
		}
		if clImages.ItemSlot(uint32(it.ID)) == slot {
			return it.Name
		}
	}
	return "Nothing"
}

// macroGetEquipmentSlotVar returns the equipped item name for a player variable ID.
func macroGetEquipmentSlotVar(varID int) string {
	slotName := invSlotNames[varID]
	if slotName == "" {
		return ""
	}
	// Map variable ID back to slot constant
	var slot int
	switch varID {
	case playerForehead:
		slot = kItemSlotForehead
	case playerNeck:
		slot = kItemSlotNeck
	case playerShoulders:
		slot = kItemSlotShoulder
	case playerArms:
		slot = kItemSlotArms
	case playerGloves:
		slot = kItemSlotGloves
	case playerFinger:
		slot = kItemSlotFinger
	case playerCoat:
		slot = kItemSlotCoat
	case playerCloak:
		slot = kItemSlotCloak
	case playerTorso:
		slot = kItemSlotTorso
	case playerWaist:
		slot = kItemSlotWaist
	case playerLegs:
		slot = kItemSlotLegs
	case playerFeet:
		slot = kItemSlotFeet
	case playerBothHands:
		slot = kItemSlotBothHands
	case playerHead:
		slot = kItemSlotHead
	}
	return macroGetEquippedItemName(slot)
}

// macroGetSelectedItemName returns the name of the currently selected item.
func macroGetSelectedItemName() string {
	items := getInventory()
	for _, it := range items {
		if it.Equipped {
			return it.Name
		}
	}
	return ""
}

// macroListShares returns a comma-separated list of shared players.
func macroListShares(isOut bool) string {
	// Shares aren't directly accessible from the current state;
	// this is a stub that returns empty.
	return ""
}

// simplePlayerName strips non-alphanumeric characters from a name.
func simplePlayerName(name string) string {
	var b strings.Builder
	for _, ch := range name {
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// macroEvalCondition evaluates a condition like "value1 > value2".
// Returns the boolean result.
func macroEvalCondition(ex *ExecutingMacro, params []*Macro) bool {
	if len(params) < 3 {
		return false
	}
	val1 := macroResolveExpression(ex, params[0].VarName)
	op := params[1].VarName
	val2 := macroResolveExpression(ex, params[2].VarName)

	// Try numeric comparison
	n1, err1 := strconv.Atoi(val1)
	n2, err2 := strconv.Atoi(val2)
	if err1 == nil && err2 == nil {
		switch op {
		case ">":
			return n1 > n2
		case "<":
			return n1 < n2
		case ">=":
			return n1 >= n2
		case "<=":
			return n1 <= n2
		case "==":
			return n1 == n2
		case "!=":
			return n1 != n2
		}
	}

	// String comparison
	switch op {
	case ">":
		return strings.Contains(strings.ToLower(val2), strings.ToLower(val1))
	case "<":
		return strings.Contains(strings.ToLower(val1), strings.ToLower(val2))
	case ">=":
		return strings.Contains(strings.ToLower(val1), strings.ToLower(val2))
	case "<=":
		return strings.Contains(strings.ToLower(val2), strings.ToLower(val1))
	case "==":
		return strings.EqualFold(val1, val2)
	case "!=":
		return !strings.EqualFold(val1, val2)
	}

	return false
}
