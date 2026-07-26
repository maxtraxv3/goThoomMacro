package main

import (
	"strings"
	"testing"
)

type savedMacroState struct {
	expressions  *Macro
	replacements *Macro
	keys         *Macro
	clicks       *Macro
	functions    *Macro
	includeFiles *Macro
	globalVars   *Macro
}

func saveMacroState() savedMacroState {
	return savedMacroState{
		expressions:  macroState.Expressions,
		replacements: macroState.Replacements,
		keys:         macroState.Keys,
		clicks:       macroState.Clicks,
		functions:    macroState.Functions,
		includeFiles: macroState.IncludeFiles,
		globalVars:   macroState.GlobalVars,
	}
}

func restoreMacroState(s savedMacroState) {
	macroState.Expressions = s.expressions
	macroState.Replacements = s.replacements
	macroState.Keys = s.keys
	macroState.Clicks = s.clicks
	macroState.Functions = s.functions
	macroState.IncludeFiles = s.includeFiles
	macroState.GlobalVars = s.globalVars
}

func resetMacroState() {
	macroState.Expressions = nil
	macroState.Replacements = nil
	macroState.Keys = nil
	macroState.Clicks = nil
	macroState.Functions = nil
	macroState.IncludeFiles = nil
	macroState.GlobalVars = nil
}

func TestExpressionMacroParsing(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := `"??"  "/help "    @text "\r"`
	p := &macroParser{
		fileName: "test",
		lines:    input + "\n",
	}
	p.parse()

	m := macroState.Expressions
	if m == nil {
		t.Fatal("no expression macros found")
	}
	if m.Expression != "??" {
		t.Errorf("expression = %q, want %q", m.Expression, "??")
	}
	if m.Commands == nil {
		t.Fatal("expression macro has nil commands")
	}
	count := 0
	for c := m.Commands; c != nil; c = c.Next {
		count++
	}
	if count != 3 {
		t.Errorf("command count = %d, want 3", count)
	}
}

func TestReplacementMacroParsing(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := "'haha' \"LOL\""
	p := &macroParser{
		fileName: "test",
		lines:    input + "\n",
	}
	p.parse()

	m := macroState.Replacements
	if m == nil {
		t.Fatal("no replacement macros found")
	}
	if m.Replace != "haha" {
		t.Errorf("replace = %q, want %q", m.Replace, "haha")
	}
	if m.Commands == nil {
		t.Fatal("replacement macro has nil commands")
	}
}

func TestSetGlobalTopLevel(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := "SETGLOBAL AT 1\nSETGLOBAL tradername \"\"\n"
	p := &macroParser{
		fileName: "test",
		lines:    input,
	}
	p.parse()

	val, ok := macroFindGlobalVariable("AT")
	if !ok || val != "1" {
		t.Errorf("global AT = %q (ok=%v), want %q", val, ok, "1")
	}
}

func TestFunctionMacroWithBlockBody(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := "@login\n{\n    set actname \"(\"\n    set actname + @my.name\n}\n"
	p := &macroParser{
		fileName: "test",
		lines:    input,
	}
	p.parse()

	var loginFn *Macro
	for m := macroState.Functions; m != nil; m = m.Next {
		if m.Name == "@login" {
			loginFn = m
			break
		}
	}
	if loginFn == nil {
		t.Fatal("@login function not found")
	}
	count := 0
	for c := loginFn.Commands; c != nil; c = c.Next {
		count++
	}
	if count != 2 {
		t.Errorf("@login command count = %d, want 2", count)
	}
}

func TestExpressionMacroWithBlockBody(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := "\"wa\"\n{\n    random\n    \"/action waves. \\r\"\n    or\n    \"/action waves \\r\"\n    end random\n}\n"
	p := &macroParser{
		fileName: "test",
		lines:    input,
	}
	p.parse()

	var waExpr *Macro
	for m := macroState.Expressions; m != nil; m = m.Next {
		if m.Expression == "wa" {
			waExpr = m
			break
		}
	}
	if waExpr == nil {
		t.Fatal("'wa' expression not found")
	}
	count := 0
	for c := waExpr.Commands; c != nil; c = c.Next {
		count++
	}
	if count != 5 {
		t.Errorf("'wa' command count = %d, want 5", count)
	}
}

func TestSplitLineQuotedStrings(t *testing.T) {
	words := splitLine(`"/help "    @text "\r"`)
	if len(words) != 3 {
		t.Errorf("splitLine got %d words, want 3: %v", len(words), words)
	}
	expected := []string{`"/help "`, "@text", `"\r"`}
	for i, w := range words {
		if w != expected[i] {
			t.Errorf("word[%d] = %q, want %q", i, w, expected[i])
		}
	}
}

func TestGetWordQuotedString(t *testing.T) {
	word, rest := getWord(`"??"  "/help "    @text "\r"`)
	if word != "\"??\"" {
		t.Errorf("word = %q, want %q", word, "\"??\"")
	}
	if !strings.HasPrefix(rest, "\"/help \"") {
		t.Errorf("rest = %q, should start with quoted /help", rest)
	}
}

func TestGotoFindsEarlierLabel(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := `@loopfunc
{
    label top
    set x 1
    set y 2
    goto top
}
`
	p := &macroParser{fileName: "test", lines: input}
	p.parse()

	var fn *Macro
	for m := macroState.Functions; m != nil; m = m.Next {
		if m.Name == "@loopfunc" {
			fn = m
			break
		}
	}
	if fn == nil {
		t.Fatal("@loopfunc not found")
	}

	ex := macroStart(fn, macroFunction, "")
	ex.Unfriendly = false // override: test step-by-step execution
	defer macroFinish(ex)

	// Walk forward one command at a time (not unfriendly = yields each step)
	// List: label top -> set x 1 -> set y 2 -> goto top
	// Step 1: execute label top (no-op)
	macroContinueOne(ex)
	// Step 2: execute set x 1
	macroContinueOne(ex)
	// Step 3: execute set y 2
	macroContinueOne(ex)
	// Step 4: execute goto top — should find "label top" via CommandsHead
	macroContinueOne(ex)
	// After goto, cursor should be at set x 1 again
	if ex.Mark == nil || ex.Mark.Commands == nil {
		t.Fatal("macro finished or no commands after goto")
	}
	cmd := ex.Mark.Commands
	if cmd.CommandKind != cmdSetVariable {
		t.Fatalf("after goto, expected cmdSetVariable, got kind=%d", cmd.CommandKind)
	}
}

func TestNumWordsResolution(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	playerName = "Aria Windrunner"
	defer func() { playerName = "" }()

	val, ok := macroResolveVariable("@my.name.num_words")
	if !ok {
		t.Fatal("could not resolve @my.name.num_words")
	}
	if val != "2" {
		t.Errorf("@my.name.num_words = %q, want %q", val, "2")
	}
}

func TestWordIndexOnUserVariable(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	macroSetGlobalVariable("mydata", "hello world foo bar")

	val, ok := macroResolveVariable("mydata.word[1]")
	if !ok {
		t.Fatal("could not resolve mydata.word[1]")
	}
	if val != "world" {
		t.Errorf("mydata.word[1] = %q, want %q", val, "world")
	}
}

func TestNestedIfWithOuterElse(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	// Pattern from /atz: inner if/endif then outer else
	input := `@testatz
{
    if trademe == 0
        set result "on"
    if AT == 0
        set result "on+warn"
    end if
    else
        set result "off"
    end if
}
`
	p := macroParser{fileName: "test", lines: input}
	p.parse()

	var fn *Macro
	for m := macroState.Functions; m != nil; m = m.Next {
		if m.Name == "@testatz" {
			fn = m
			break
		}
	}
	if fn == nil {
		t.Fatal("@testatz function not found")
	}

	// Command chain: if trademe, set "on", if AT, set "on+warn",
	//   end if (inner), else, set "off", end if (outer)
	cmdKinds := []int{}
	for c := fn.Commands; c != nil; c = c.Next {
		cmdKinds = append(cmdKinds, c.CommandKind)
	}
	expected := []int{
		cmdIf, cmdSetVariable,
		cmdIf, cmdSetVariable, cmdEndIf,
		cmdElse, cmdSetVariable, cmdEndIf,
	}
	if len(cmdKinds) != len(expected) {
		t.Fatalf("command count = %d, want %d; kinds = %v", len(cmdKinds), len(expected), cmdKinds)
	}
	for i, k := range cmdKinds {
		if k != expected[i] {
			t.Errorf("cmdKinds[%d] = %d, want %d", i, k, expected[i])
		}
	}

	tests := []struct {
		name   string
		tradAT string // "trademe AT" e.g. "0,0" or "0,1" or "1,0"
		want   string
	}{
		{"trademe=0 AT=0", "0,0", "on+warn"},
		{"trademe=0 AT=1", "0,1", "on"},
		{"trademe=1 AT=0", "1,0", "off"},
		{"trademe=1 AT=1", "1,1", "off"},
	}

	for _, tt := range tests {
		parts := strings.Split(tt.tradAT, ",")
		macroSetGlobalVariable("trademe", parts[0])
		macroSetGlobalVariable("AT", parts[1])
		ex := macroStart(fn, macroFunction, "")
		ex.Unfriendly = true
		macroContinueOne(ex)
		macroFinish(ex)

		got, _ := macroGetLocalVariable(ex, "result")
		if got != tt.want {
			t.Errorf("%s: result=%q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestIfElseIfElseChain(t *testing.T) {
	saved := saveMacroState()
	defer restoreMacroState(saved)
	resetMacroState()

	input := `@testbranch
{
    set result "none"
    if AT == 1
        set result "one"
    else if AT == 2
        set result "two"
    else
        set result "other"
    end if
}
`
	p := macroParser{fileName: "test", lines: input}
	p.parse()

	var fn *Macro
	for m := macroState.Functions; m != nil; m = m.Next {
		if m.Name == "@testbranch" {
			fn = m
			break
		}
	}
	if fn == nil {
		t.Fatal("@testbranch function not found")
	}

	// Verify command chain structure
	cmdKinds := []int{}
	for c := fn.Commands; c != nil; c = c.Next {
		cmdKinds = append(cmdKinds, c.CommandKind)
	}
	if len(cmdKinds) != 8 {
		t.Fatalf("command count = %d, want 8; kinds = %v", len(cmdKinds), cmdKinds)
	}
	expected := []int{cmdSetVariable, cmdIf, cmdSetVariable, cmdElseIf, cmdSetVariable, cmdElse, cmdSetVariable, cmdEndIf}
	for i, k := range cmdKinds {
		if k != expected[i] {
			t.Errorf("cmdKinds[%d] = %d, want %d", i, k, expected[i])
		}
	}

	tests := []struct {
		atVal string
		want  string
	}{
		{"1", "one"},
		{"2", "two"},
		{"3", "other"},
		{"99", "other"},
	}

	for _, tt := range tests {
		macroSetGlobalVariable("AT", tt.atVal)
		ex := macroStart(fn, macroFunction, "")
		ex.Unfriendly = true
		macroContinueOne(ex)
		macroFinish(ex)

		got, _ := macroGetLocalVariable(ex, "result")
		if got != tt.want {
			t.Errorf("AT=%s: result=%q, want %q", tt.atVal, got, tt.want)
		}
	}
}

func TestClickMacroLookup(t *testing.T) {
	s := saveMacroState()
	defer restoreMacroState(s)
	resetMacroState()

	// Parse a minimal click macro file
	input := `shift-click2
{
    if shiftc2 == 0
        "/share test\r"
        end if
}
control-click2
{
    "/sell 0 test\r"
}
control-click
{
    "/use test\r"
}
control-shift-click2
{
    "/whisper test\r"
}
`
	p := &macroParser{lines: input}
	p.parse()

	// Verify macros were added to Clicks list
	count := 0
	for m := macroState.Clicks; m != nil; m = m.Next {
		count++
	}
	if count != 4 {
		t.Fatalf("expected 4 click macros, got %d", count)
	}

	tests := []struct {
		name    string
		button  int
		mods    uint
		wantNil bool
	}{
		{"shift-click2", 1025, 0x0001, false},
		{"control-click2", 1025, 0x0002, false},
		{"control-click", 1024, 0x0002, false},
		{"control-shift-click2", 1025, 0x0003, false},
		{"wrong button", 1024, 0x0001, true},
		{"wrong mods", 1025, 0x0004, true},
		{"no mods", 1025, 0x0000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := macroFindClickMacro(tt.button, tt.mods)
			if tt.wantNil {
				if m != nil {
					t.Errorf("expected nil, got macro with key=%d mods=%d", m.Key, m.Modifiers)
				}
			} else {
				if m == nil {
					t.Errorf("expected to find click macro for button=%d mods=%d", tt.button, tt.mods)
				}
			}
		})
	}
}
