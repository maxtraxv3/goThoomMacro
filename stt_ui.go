package main

import (
	"fmt"
	"strings"

	"gothoom/eui"
	"gothoom/internal/vosk"
)

var (
	sttTestWin   *eui.WindowData
	sttTestList  *eui.ItemData
	sttTestLines []string
)

func makeSTTTestWindow() {
	if sttTestWin != nil {
		return
	}
	sttTestWin, sttTestList, _ = makeTextWindow("STT Test", eui.HZoneCenter, eui.VZoneMiddleTop, false)
	// No AutoSize: updateTextWindow's size math combined with AutoSize causes
	// the window to grow on every refresh. A fixed size keeps it stable.
	sttTestWin.Size = eui.Point{X: 380, Y: 300}

	flow := sttTestList.Parent

	toggleBtn, toggleEvents := eui.NewButton()
	toggleBtn.Text = "Start/Stop listening"
	toggleBtn.Size.Y = 20
	toggleBtn.Fixed = true
	toggleEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			logWarn("stt diag: test window Start/Stop button clicked")
			sttToggle()
			sttTestDirty.Store(true)
		}
	}
	flow.PrependItem(toggleBtn)

	sttTestWin.OnResize = func() { updateSTTTestWindow() }
}

// updateSTTTestWindow rebuilds the STT test window content. It is called from
// updateSTT whenever recognition state changes.
func updateSTTTestWindow() {
	if sttTestWin == nil || !sttTestWin.IsOpen() {
		return
	}
	sttTestLines = sttTestStatusLines()
	updateTextWindow(sttTestWin, sttTestList, nil, sttTestLines, 14, "", nil)
	sttTestWin.Refresh()
}

func sttTestStatusLines() []string {
	heard := sttLiveText
	if heard == "" {
		heard = "(nothing yet)"
	}
	last := sttLastFinal
	if last == "" {
		last = "(nothing yet)"
	}
	match := sttLastMatch
	if match == "" {
		match = "waiting..."
	}
	mic := gs.STTMicrophone
	if mic == "" {
		mic = "(system default)"
	}
	lvl := vosk.Level()
	pct := int(lvl * 100)
	if pct > 100 {
		pct = 100
	}
	bar := strings.Repeat("=", int(lvl*30))
	return []string{
		"Status: " + sttStatusText(),
		"",
		"Mic:    " + mic,
		"",
		"Level:  " + fmt.Sprintf("%d%% %s", pct, bar),
		"",
		fmt.Sprintf("Gain:   %.1fx   Rate: %.0f Hz", vosk.Gain(), vosk.SampleRate()),
		"",
		fmt.Sprintf("Chunks: %d", vosk.ChunkCount()),
		"",
		"Heard:  " + heard,
		"",
		"Last:   " + last,
		"",
		"Match:  " + match,
	}
}

func openSTTTestWindow(anchor *eui.ItemData) {
	if sttTestWin == nil {
		makeSTTTestWindow()
	}
	if sttTestWin.Open {
		sttTestWin.Close()
		return
	}
	if anchor != nil {
		sttTestWin.MarkOpenNear(anchor)
	} else {
		sttTestWin.MarkOpen()
	}
	updateSTTTestWindow()
}
