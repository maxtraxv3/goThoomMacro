package main

import "strings"

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

	updateConsoleWindow()

	runConsoleTriggers(msg)
}

func getConsoleMessages() []string {
	format := gs.TimestampFormat
	if format == "" {
		format = "1/2/06 3:04:05pm"
	}
	return consoleLog.Entries(format, gs.ConsoleTimestamps)
}
