package main

import "time"

// players maintenance state machine handling /be-who, /be-share and /be-info.
// The /be-who scan is the authoritative source for who is online; after each
// scan completes, players not listed are marked Offline.

// internal phases
const (
	phaseWho = iota
	phaseShare
	phaseInfo
)

var (
	playersPhase   = phaseWho
	playersLastCmd time.Time
	whoRequested   bool
)

// requestPlayersData progresses the maintenance state machine.
func requestPlayersData() {
	now := time.Now()

	if pendingCommand != "" {
		return
	}
	if now.Sub(playersLastCmd) < time.Second {
		return
	}

	switch playersPhase {
	case phaseWho:
		if whoActive {
			if maybeEnqueueWho() {
				playersLastCmd = now
			}
			return
		}
		if !whoRequested {
			pendingCommand = "/be-who"
			whoLastRequest = now
			playersLastCmd = now
			whoRequested = true
			return
		}
		// who scan finished — mark players absent from the scan as offline.
		markWhoAbsentOffline()
		playersPhase = phaseShare
		whoRequested = false
	case phaseShare:
		pendingCommand = "/be-share"
		playersLastCmd = now
		playersPhase = phaseInfo
	case phaseInfo:
		if maybeEnqueueInfo() {
			playersLastCmd = now
		} else if !whoActive {
			playersPhase = phaseWho
		}
	}
}

// markWhoAbsentOffline sets Offline=true for every player whose beWho flag
// is still false after a completed scan.  The current player is exempted.
func markWhoAbsentOffline() {
	playersMu.Lock()
	changed := false
	for name, p := range players {
		if p.beWho {
			continue
		}
		if name == playerName {
			continue
		}
		if !p.Offline {
			p.Offline = true
			changed = true
		}
	}
	playersMu.Unlock()
	if changed {
		playersDirty.Store(true)
	}
}
