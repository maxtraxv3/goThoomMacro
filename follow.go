package main

import (
	"fmt"
	"strings"
)

// followStopRadius is how close (in world units) the player stops when
// following, so we stand next to the target rather than on top of them.
const followStopRadius = 10

// handleFollowCommand handles "/follow <player>" and "/follow stop". A bare
// "/follow" stops, matching the classic client's behavior.
func handleFollowCommand(args string) {
	fields := strings.Fields(args)
	if len(fields) == 0 || strings.EqualFold(fields[0], "stop") || strings.EqualFold(fields[0], "off") {
		stopFollow()
		return
	}
	name := resolvePlayerName(fields[0])
	if name == "" {
		// Not in the player list; match an on-screen mobile by name.
		name = currentMobileName(fields[0])
	}
	if name == "" {
		consoleMessage(fmt.Sprintf("Could not find '%s' on screen.", fields[0]))
		return
	}
	followTarget = name
	consoleMessage(fmt.Sprintf("Following %s.", name))
}

// stopFollow stops following and halts any command-driven walk.
func stopFollow() {
	if followTarget == "" {
		return
	}
	followTarget = ""
	moveCmdActive = false
	keyStopFrames = 3
	consoleMessage("Stopped following.")
}

// updateFollow steers the command-walk toward the followed player's current
// position. It runs every frame and simply re-aims the shared command-walk
// state; the walk only happens when no manual input is steering.
func updateFollow() {
	if followTarget == "" {
		return
	}
	tH, tV, ok := currentMobilePosition(followTarget)
	if !ok {
		// Target left the screen; stop until they reappear.
		moveCmdActive = false
		return
	}
	if abs16(tH) < followStopRadius && abs16(tV) < followStopRadius {
		// Close enough; stand next to them.
		moveCmdActive = false
		return
	}
	moveCmdActive = true
	moveCmdX = tH
	moveCmdY = tV
}

// currentMobileName returns the exact on-screen descriptor name matching the
// given (case-insensitive) name, or "" when no visible mobile matches.
func currentMobileName(name string) string {
	stateMu.Lock()
	defer stateMu.Unlock()
	for _, m := range state.mobiles {
		if d, found := state.descriptors[m.Index]; found && d.Name != "" && strings.EqualFold(d.Name, name) {
			return d.Name
		}
	}
	return ""
}

// currentMobilePosition returns the on-screen position of the mobile whose
// descriptor name matches, in world coordinates relative to the player.
func currentMobilePosition(name string) (h, v int16, ok bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	for _, m := range state.mobiles {
		if d, found := state.descriptors[m.Index]; found && d.Name != "" && strings.EqualFold(d.Name, name) {
			return m.H, m.V, true
		}
	}
	return 0, 0, false
}

func abs16(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}
