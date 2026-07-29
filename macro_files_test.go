package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testDataDir() string {
	_, src, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(src), "data", "Macros")
}

func TestAllMacroFilesParse(t *testing.T) {
	dir := testDataDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".txt") {
			continue
		}
		if strings.EqualFold(name, "Macro Instructions.txt") {
			continue
		}
		if strings.Contains(name, " - Copy") {
			continue
		}
		fpath := filepath.Join(dir, name)
		t.Run(name, func(t *testing.T) {
			saved := saveMacroState()
			defer restoreMacroState(saved)
			resetMacroState()

			macroParseFile(fpath)

			if macroState.Expressions == nil &&
				macroState.Replacements == nil &&
				macroState.Keys == nil &&
				macroState.Clicks == nil &&
				macroState.Functions == nil &&
				macroState.IncludeFiles == nil &&
				macroState.GlobalVars == nil {
				t.Errorf("parsing %s produced no macros", fpath)
			}
		})
	}
}
