package main

import (
	"image/color"
	"strings"
	"sync/atomic"
)

// consoleDirty is set by consoleMessage (network goroutine) and consumed by
// Game.Update (Ebiten goroutine) so that updateTextWindow modifications to
// the EUI item tree always happen on the draw goroutine.
var consoleDirty atomic.Bool

const (
	maxMessages = 1000
	sndTink     = 58 // notification sound
)

var consoleLog = messageLog{max: maxMessages}

func consoleMessage(msg string) {
	if msg == "" {
		return
	}
	if wasmPrivacyActive() {
		return
	}
	msg = strings.ReplaceAll(msg, "\r", "")
	if msg == "" {
		return
	}
	if msg == "You have been idle for too long." {
		showNotification(msg)
		playSound([]uint16{sndTink})
	}
	consoleLog.Add(msg)
	appendConsoleLog(msg)
	macroState.TextLogBuffer = msg

	// Defer UI update to Game.Update on the Ebiten goroutine to avoid
	// a data race between the network goroutine and the draw goroutine
	// both accessing list.Contents.
	consoleDirty.Store(true)

	runConsoleTriggers(msg)
}

func getConsoleMessages() []string {
	format := gs.TimestampFormat
	if format == "" {
		format = "1/2/06 3:04:05pm"
	}
	return consoleLog.Entries(format, gs.ConsoleTimestamps)
}

// consoleMessageColor returns a color for a console message based on its type,
// or nil if no color should be applied.
func consoleMessageColor(msg string) *color.RGBA {
	if !gs.ConsoleColors {
		return nil
	}
	// Strip timestamp prefix if present (e.g., "[1/2/06 3:04:05pm] ")
	s := msg
	if idx := strings.Index(s, "] "); idx >= 0 && idx < 30 {
		s = s[idx+2:]
	}
	lower := strings.ToLower(s)

	// Yell: "Hero yells, text"
	if strings.Contains(lower, " yells") {
		c := gs.ConsoleYellColor.ToRGBA()
		return &c
	}
	// Ponder: "Hero ponders" or "Hero ponders, text"
	if strings.Contains(lower, " ponders") {
		c := gs.ConsolePonderColor.ToRGBA()
		return &c
	}
	// Think: "Hero thinks to you, text" / "Hero thinks, text"
	if strings.Contains(lower, "thinks") {
		c := gs.ConsoleThinkColor.ToRGBA()
		return &c
	}
	// Narrate: "(Name): text"
	if strings.HasPrefix(s, "(") && strings.Contains(s, "): ") {
		c := gs.ConsoleNarrateColor.ToRGBA()
		return &c
	}
	// Action: "(text)" or "* text"
	if strings.HasPrefix(s, "(") || strings.HasPrefix(s, "* ") {
		c := gs.ConsoleActionColor.ToRGBA()
		return &c
	}
	// Coin/revive messages
	if strings.Contains(lower, "coin") || strings.Contains(lower, "reviv") {
		c := gs.ConsoleCoinColor.ToRGBA()
		return &c
	}
	return nil
}
