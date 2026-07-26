package main

import (
	"image/color"
	"path/filepath"
	"strings"

	"gothoom/eui"
)

// macroUIState holds UI-related macro state.
var (
	macroIndicator *eui.ItemData
	macroReloadBtn *eui.ItemData
)

// macroDrawIndicator draws a grey border around the text input when a macro is executing.
// Called from the console UI draw path.
func macroDrawIndicator() bool {
	macroState.mu.Lock()
	running := macroState.Executing != nil
	macroState.mu.Unlock()
	return running
}

// macroMakeReloadButton creates a "Reload Macros" button for the options/help area.
func macroMakeReloadButton() *eui.ItemData {
	btn, events := eui.NewButton()
	btn.Text = "Reload Macros"
	events.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			macroReload()
		}
	}
	macroReloadBtn = btn
	return btn
}

// macroAddReloadToMenu adds the Reload Macros button to an existing window's contents.
func macroAddReloadToMenu(items []*eui.ItemData) []*eui.ItemData {
	return append(items, macroMakeReloadButton())
}

// macroSetupUI sets up the macro system's UI elements.
// Called during UI initialization.
func macroSetupUI() {
	// The reload button is added to the help/options window by the caller
}

// macroHandleConsoleDraw adds the macro execution indicator to the console.
// Returns a color for the text input border when a macro is running.
func macroHandleConsoleDraw() color.Color {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()
	if macroState.Executing != nil {
		return color.RGBA{0x80, 0x80, 0x80, 0xff} // grey border
	}
	return nil
}

// macroGetStatus returns a status string about currently executing macros.
func macroGetStatus() string {
	macroState.mu.Lock()
	defer macroState.mu.Unlock()

	count := 0
	for ex := macroState.Executing; ex != nil; ex = ex.Next {
		count++
	}
	if count == 0 {
		return ""
	}
	if count == 1 {
		return "1 macro executing"
	}
	return strings.Repeat("macro", count) // simplified
}

// macroMakeWindow creates the macros information window.
func macroMakeWindow() *eui.WindowData {
	win := eui.NewWindow()
	win.Title = "Macros"
	win.Closable = true
	win.Movable = true
	win.AutoSize = true

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	win.AddItem(flow)

	addText := func(text string) {
		t, _ := eui.NewText()
		t.Text = text
		flow.AddItem(t)
	}

	addText("Macro System")

	macroDir := filepath.Join(dataDirPath, "Macros")
	addText("Macro folder: " + macroDir)
	addText("")
	addText("Expression macros are triggered by typing the")
	addText("expression text and pressing Enter.")
	addText("")
	addText("Key macros are triggered by pressing")
	addText("the assigned key combination.")
	addText("")
	addText("Edit macro files in the Macros folder")
	addText("to add or modify macros.")
	addText("")

	flow.AddItem(macroMakeReloadButton())

	return win
}
