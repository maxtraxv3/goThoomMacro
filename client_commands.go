package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// runClientCommand executes a client-side slash command. It returns true when
// the command was handled locally; otherwise the caller should send the text
// to the server. This is the single dispatch point used by chat input, the
// command queue (macros/scripts), and speech-to-text. The command set mirrors
// the classic client's Commands_cl.cp.
func runClientCommand(txt string) bool {
	if !strings.HasPrefix(txt, "/") {
		return false
	}
	lower := strings.ToLower(txt)
	switch {
	case hasCommandPrefix(lower, "/selectitem"):
		handleSelectItem(strings.TrimSpace(txt[len("/selectitem"):]))
		return true
	case hasCommandPrefix(lower, "/select"):
		handleSelect(strings.TrimSpace(txt[len("/select"):]))
		return true
	case hasCommandPrefix(lower, "/move"):
		handleMoveCommand(strings.TrimSpace(txt[len("/move"):]))
		return true
	case hasCommandPrefix(lower, "/follow"):
		handleFollowCommand(strings.TrimSpace(txt[len("/follow"):]))
		return true
	case hasCommandPrefix(lower, "/label"):
		handleLabelCommand(strings.TrimSpace(txt[len("/label"):]))
		return true
	case hasCommandPrefix(lower, "/wholabel"):
		handleWhoLabelCommand(strings.TrimSpace(txt[len("/wholabel"):]))
		return true
	case hasCommandPrefix(lower, "/pref"):
		handlePrefCommand(strings.TrimSpace(txt[len("/pref"):]))
		return true
	case hasCommandPrefix(lower, "/record"):
		handleRecordCommand(strings.TrimSpace(txt[len("/record"):]))
		return true
	case hasCommandPrefix(lower, "/notts"):
		handleNoTTSCommand(strings.TrimSpace(txt[len("/notts"):]))
		return true
	}

	// Script-registered commands.
	parts := strings.SplitN(strings.TrimPrefix(txt, "/"), " ", 2)
	name := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}
	if handler, ok := scriptCommands[name]; ok && handler != nil {
		owner := scriptCommandOwners[name]
		if scriptDisabled[owner] {
			// Disabled script commands fall through so the server still
			// receives the user's input.
			return false
		}
		scriptLogEvent(owner, "Command", args)
		go handler(args)
		return true
	}
	return false
}

// hasCommandPrefix matches a lowercased command at a word boundary so that
// "/select" does not accidentally match "/selected".
func hasCommandPrefix(lower, prefix string) bool {
	if !strings.HasPrefix(lower, prefix) {
		return false
	}
	rest := lower[len(prefix):]
	return rest == "" || rest[0] == ' ' || rest[0] == '\t'
}

// handleMoveCommand handles "/move <dir> [run|walk]" and "/move stop". A bare
// "/move" stops, matching the classic client's "\move".
func handleMoveCommand(args string) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		sendWalkCommand(0, 0, false)
		return
	}
	dir := strings.ToLower(fields[0])
	if dir == "stop" {
		sendWalkCommand(0, 0, false)
		return
	}
	fast := false
	if len(fields) > 1 && strings.EqualFold(fields[1], "run") {
		fast = true
	}
	var dx, dy int
	switch dir {
	case "e", "east":
		dx, dy = 1, 0
	case "ne", "northeast":
		dx, dy = 1, -1
	case "n", "north":
		dx, dy = 0, -1
	case "nw", "northwest":
		dx, dy = -1, -1
	case "w", "west":
		dx, dy = -1, 0
	case "sw", "southwest":
		dx, dy = -1, 1
	case "s", "south":
		dx, dy = 0, 1
	case "se", "southeast":
		dx, dy = 1, 1
	default:
		consoleMessage("Usage: /move <n|ne|e|se|s|sw|w|nw|stop> [run|walk]")
		return
	}
	sendWalkCommand(dx, dy, fast)
}

// handleLabelCommand handles "/label <player> [label]" where label may be a
// number (0-10) or a label name. With no label it defaults to label 1,
// matching the classic client.
func handleLabelCommand(args string) {
	fields := strings.Fields(args)
	if len(fields) == 0 || strings.EqualFold(fields[0], "?") || strings.EqualFold(fields[0], "help") {
		consoleMessage("Usage: /label <player> [label]")
		return
	}
	name := resolvePlayerName(fields[0])
	if name == "" {
		consoleMessage(fmt.Sprintf("No player named '%s' found in the player list.", fields[0]))
		return
	}
	label := 1
	if len(fields) > 1 {
		labelStr := fields[1]
		if n, err := strconv.Atoi(labelStr); err == nil {
			if n < 0 || n > len(labelColors) {
				consoleMessage(fmt.Sprintf("Label number must be 0-%d.", len(labelColors)))
				return
			}
			label = n
		} else {
			found := false
			for i := 1; i <= len(labelColors); i++ {
				if strings.EqualFold(labelName(i), labelStr) {
					label = i
					found = true
					break
				}
			}
			if !found {
				consoleMessage(fmt.Sprintf("Unknown label '%s'.", labelStr))
				return
			}
		}
	}
	setPlayerLabel(name, label, true)
	if label == 0 {
		consoleMessage(fmt.Sprintf("Cleared label for %s.", name))
	} else {
		consoleMessage(fmt.Sprintf("Labeled %s: %s", name, labelName(label)))
	}
}

// handleWhoLabelCommand handles "/wholabel [label]" and lists friends, either
// all or just those carrying a specific label.
func handleWhoLabelCommand(args string) {
	label := 0
	a := strings.TrimSpace(args)
	if a != "" && !strings.EqualFold(a, "?") && !strings.EqualFold(a, "help") {
		if n, err := strconv.Atoi(a); err == nil {
			label = n
		} else {
			found := false
			for i := 1; i <= len(labelColors); i++ {
				if strings.EqualFold(labelName(i), a) {
					label = i
					found = true
					break
				}
			}
			if !found {
				consoleMessage(fmt.Sprintf("Unknown label '%s'.", a))
				return
			}
		}
	}
	ps := getPlayers()
	names := make([]string, 0, len(ps))
	for _, p := range ps {
		if p.Name == "" || p.IsNPC || p.FriendLabel <= 0 {
			continue
		}
		if label > 0 && p.FriendLabel != label {
			continue
		}
		names = append(names, p.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		consoleMessage("No friends found.")
		return
	}
	if label > 0 {
		consoleMessage(fmt.Sprintf("%s (%d): %s", labelName(label), len(names), strings.Join(names, ", ")))
	} else {
		consoleMessage(fmt.Sprintf("Friends (%d): %s", len(names), strings.Join(names, ", ")))
	}
}

// handleRecordCommand handles "/record [on|off]". With no argument it toggles
// movie recording, matching the classic client's "\record".
func handleRecordCommand(args string) {
	switch strings.ToLower(strings.TrimSpace(args)) {
	case "":
		toggleRecording()
		return
	case "on", "true", "1", "yes":
		if recorder != nil {
			consoleMessage("A movie is already being recorded.")
			return
		}
		if clmov != "" || playingMovie || pcapPath != "" || fake {
			consoleMessage("Cannot record a movie while playing one.")
			return
		}
		if tcpConn == nil {
			recordingMovie = true
			consoleMessage("recording will start on connect")
			updateRecordButton()
			return
		}
		startRecording()
	case "off", "false", "0", "no":
		if recorder == nil {
			consoleMessage("No movie is being recorded.")
			return
		}
		stopRecording()
	default:
		consoleMessage("Usage: /record [on|off]")
	}
}

// handlePrefCommand handles "/pref <subcommand> [value]". The subset that maps
// to client settings is applied; anything without a corresponding setting
// reports that it is not implemented.
func handlePrefCommand(args string) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		consoleMessage("Usage: /pref <movement hold|toggle> | <shownames|timestamps|message fallen|message clanning> <on|off> | <soundvolume|bardvolume|maxnight> <0-100>")
		return
	}
	sub := strings.ToLower(fields[0])
	value := ""
	if len(fields) > 1 {
		value = fields[1]
	}

	switch sub {
	case "movement":
		if strings.EqualFold(value, "hold") {
			consoleMessage("/pref movement hold is not implemented in this client.")
		} else if strings.EqualFold(value, "toggle") {
			consoleMessage("/pref movement toggle is not implemented in this client.")
		} else {
			consoleMessage("Usage: /pref movement <hold|toggle>")
		}
		return

	case "message":
		if len(fields) < 3 {
			consoleMessage("Usage: /pref message <fallen|clanning> <on|off>")
			return
		}
		if _, ok := parsePrefBool(fields[2]); !ok {
			consoleMessage("Usage: /pref message <fallen|clanning> <on|off>")
			return
		}
		switch strings.ToLower(fields[1]) {
		case "fallen":
			consoleMessage("Fallen message suppression is not implemented in this client.")
		case "clanning":
			consoleMessage("Clanning message suppression is not implemented in this client.")
		default:
			consoleMessage("Usage: /pref message <fallen|clanning> <on|off>")
		}
		return

	case "shownames":
		on, ok := parsePrefBool(value)
		if !ok {
			consoleMessage("Usage: /pref shownames <on|off>")
			return
		}
		gs.hideMobiles = !on
		consoleMessage(fmt.Sprintf("Showing names: %t", on))
		return

	case "timestamps":
		on, ok := parsePrefBool(value)
		if !ok {
			consoleMessage("Usage: /pref timestamps <on|off>")
			return
		}
		gs.ChatTimestamps = on
		gs.ConsoleTimestamps = on
		consoleMessage(fmt.Sprintf("Timestamps: %t", on))
		return

	case "brightcolors":
		if _, ok := parsePrefBool(value); !ok {
			consoleMessage("Usage: /pref brightcolors <on|off>")
			return
		}
		consoleMessage("/pref brightcolors is not implemented in this client.")
		return

	case "newlog":
		if _, ok := parsePrefBool(value); !ok {
			consoleMessage("Usage: /pref newlog <on|off>")
			return
		}
		consoleMessage("/pref newlog is not implemented in this client.")
		return

	case "soundvolume":
		n, ok := parsePrefPercent(value)
		if !ok {
			consoleMessage("Usage: /pref soundvolume <0-100>")
			return
		}
		gs.MasterVolume = float64(n) / 100.0
		updateSoundVolume()
		consoleMessage(fmt.Sprintf("Sound volume: %d", n))
		return

	case "bardvolume":
		n, ok := parsePrefPercent(value)
		if !ok {
			consoleMessage("Usage: /pref bardvolume <0-100>")
			return
		}
		gs.MusicVolume = float64(n) / 100.0
		updateSoundVolume()
		consoleMessage(fmt.Sprintf("Bard volume: %d", n))
		return

	case "maxnight":
		n, ok := parsePrefPercent(value)
		if !ok {
			consoleMessage("Usage: /pref maxnight <0-100>")
			return
		}
		// Match the prefs dialog: round to the nearest 25%.
		n = (n / 25) * 25
		gs.MaxNightLevel = n
		consoleMessage(fmt.Sprintf("Max night level: %d", n))
		return
	}

	consoleMessage("Usage: /pref <movement hold|toggle> | <shownames|timestamps|message fallen|message clanning> <on|off> | <soundvolume|bardvolume|maxnight> <0-100>")
}

// parsePrefBool parses an on/off boolean value.
func parsePrefBool(s string) (bool, bool) {
	switch strings.ToLower(s) {
	case "on", "true", "1", "yes":
		return true, true
	case "off", "false", "0", "no":
		return false, true
	}
	return false, false
}

// parsePrefPercent parses an integer 0-100.
func parsePrefPercent(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 || n > 100 {
		return 0, false
	}
	return n, true
}
