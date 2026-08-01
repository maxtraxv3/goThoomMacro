package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"gothoom/eui"
	"gothoom/internal/vosk"
)

//go:embed data/stt_commands.txt
var defaultSTTCommands []byte

const sttCommandsFile = "stt_commands.txt"

var (
	sttCommands   map[string]string
	sttCommandsMu sync.RWMutex

	sttHotkeyCode int
	sttHotkeyMods uint

	sttStarted bool
	sttHeld    bool
	sttErr     string
	sttErrMu   sync.RWMutex

	// Recognition state shown in the STT test window.
	sttLiveText  string
	sttLastFinal string
	sttLastMatch string
	sttTestDirty atomic.Bool
	sttTestFrame int
)

// sttTextCh carries recognition results from the Vosk worker goroutine to the
// game Update goroutine, which drains it in updateSTT.
var sttTextCh = make(chan vosk.TextEvent, 16)

func sttDataDir() string {
	return filepath.Join(dataDirPath, "vosk")
}

func sttModelDir() string {
	return filepath.Join(sttDataDir(), gs.STTModel)
}

func sttLibPaths() []string {
	if isWASM {
		return nil
	}
	dir := sttDataDir()
	paths := make([]string, 0, 3)
	switch runtime.GOOS {
	case "darwin":
		paths = append(paths, filepath.Join(dir, "libvosk.dylib"))
	case "windows":
		paths = append(paths, filepath.Join(dir, "libvosk.dll"))
	default:
		paths = append(paths, filepath.Join(dir, "libvosk.so"))
	}
	paths = append(paths,
		filepath.Join(dir, "libvosk.dylib"),
		filepath.Join(dir, "libvosk.dll"),
		filepath.Join(dir, "libvosk.so"),
	)
	return paths
}

func init() {
	loadSTTCommands()
}

func loadSTTCommands() {
	path := filepath.Join(dataDirPath, sttCommandsFile)
	var b []byte
	if isWASM {
		b = append([]byte(nil), defaultSTTCommands...)
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_ = os.WriteFile(path, defaultSTTCommands, 0o644)
		}
		var err error
		b, err = os.ReadFile(path)
		if err != nil {
			logError("read stt_commands: %v", err)
			return
		}
	}
	m := parseSTTCommands(string(b))
	sttCommandsMu.Lock()
	sttCommands = m
	sttCommandsMu.Unlock()
}

// parseSTTCommands parses the phrase=command file. Lines starting with '#'
// and inline comments (everything after the first '#') are ignored, and
// leading/trailing whitespace is trimmed.
func parseSTTCommands(content string) map[string]string {
	m := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx >= 0 {
			phrase := strings.TrimSpace(line[:idx])
			cmd := strings.TrimSpace(line[idx+1:])
			// Strip inline comments: everything from the first '#' is ignored.
			if c := strings.Index(phrase, "#"); c >= 0 {
				phrase = strings.TrimSpace(phrase[:c])
			}
			if c := strings.Index(cmd, "#"); c >= 0 {
				cmd = strings.TrimSpace(cmd[:c])
			}
			if phrase != "" && cmd != "" {
				m[strings.ToLower(phrase)] = cmd
			}
		}
	}
	return m
}

// sttToggle starts or stops listening, called by the settings UI button.
func sttToggle() {
	if sttStarted {
		logWarn("stt diag: toggle -> stop")
		vosk.Stop()
		sttStarted = false
		return
	}
	logWarn("stt diag: toggle -> start")
	sttStart()
}

func sttStart() {
	if sttStarted {
		return
	}
	if !gs.STTEnabled {
		sttSetErr("speech-to-text is disabled in settings")
		return
	}
	if !vosk.Available(sttLibPaths()) {
		sttSetErr("libvosk library not found - see data/vosk/README.txt")
		return
	}
	if !vosk.ModelReady(sttModelDir()) {
		sttSetErr("vosk model not found at " + sttModelDir())
		return
	}
	err := vosk.Start(vosk.Options{
		LibPaths:  sttLibPaths(),
		ModelPath: sttModelDir(),
		MicName:   gs.STTMicrophone,
		OnText:    onSTTText,
	})
	if err != nil {
		sttSetErr(err.Error())
		return
	}
	sttSetErr("")
	sttStarted = true
	logWarn("stt diag: started listening")
	sttTestDirty.Store(true)
}

func sttStop() {
	if sttStarted {
		logWarn("stt diag: sttStop()")
		vosk.Stop()
		sttStarted = false
		sttTestDirty.Store(true)
	}
}

// sttStatusText summarizes the current STT state for display.
func sttStatusText() string {
	switch {
	case !gs.STTEnabled:
		return "Disabled"
	case sttStarted:
		return "Listening"
	case sttError() != "":
		return sttError()
	case !vosk.Available(sttLibPaths()):
		return "libvosk not found - see data/vosk/README.txt"
	case !vosk.ModelReady(sttModelDir()):
		return "Model missing - download in the Downloads window"
	default:
		return "Ready"
	}
}

func sttSetErr(msg string) {
	sttErrMu.Lock()
	sttErr = msg
	sttErrMu.Unlock()
}

func sttError() string {
	sttErrMu.RLock()
	defer sttErrMu.RUnlock()
	return sttErr
}

func onSTTText(ev vosk.TextEvent) {
	select {
	case sttTextCh <- ev:
	default:
	}
}

func processSTTText(ev vosk.TextEvent) {
	txt := strings.TrimSpace(ev.Text)
	sttLiveText = txt
	sttTestDirty.Store(true)
	if txt == "" {
		return
	}
	if !ev.Final {
		return
	}
	if isNoiseWord(txt) {
		return
	}
	sttLastFinal = txt
	if phrase, cmd, rest, ok := sttFindCommand(txt); ok {
		// {text} is replaced with everything spoken after the matched phrase.
		cmd = sttTemplate(cmd, rest)
		sttLastMatch = fmt.Sprintf("matched %q -> %s", phrase, cmd)
		logDebug("stt command: %q -> %q", txt, cmd)
		dispatchSTTCommand(cmd)
		return
	}
	if gs.STTDictateToChat {
		sttLastMatch = "no command matched - dictating to chat"
		logDebug("stt dictate: %q", txt)
		enqueueCommand(txt)
		return
	}
	sttLastMatch = "no command matched - enable dictation to send it"
}

// sttTemplate substitutes the free text captured after the matched phrase into
// the command template's {text} placeholder.
func sttTemplate(cmd, rest string) string {
	return strings.ReplaceAll(cmd, "{text}", rest)
}

// isNoiseWord reports whether a recognized utterance is just a single filler
// word that Vosk hallucinated from noise/silence. These are dropped instead of
// being dictated to chat or run as a command.
func isNoiseWord(txt string) bool {
	t := strings.ToLower(strings.TrimSpace(txt))
	t = strings.Trim(t, ".,!?")
	switch t {
	case "the", "a", "an", "and", "um", "uh", "erm", "hmm", "ah", "oh", "like", "i":
		return true
	}
	return false
}

// sttFindCommand matches the spoken text against the phrase table and returns
// the matched phrase, the configured command, and the text spoken after the
// phrase. The phrase must appear at the start of what you said (followed by a
// space or end of the utterance) and the longest phrase wins, so e.g. "think to
// clan" beats "think to" beats "think".
func sttFindCommand(txt string) (phrase, cmd, rest string, ok bool) {
	lower := strings.ToLower(strings.TrimSpace(txt))
	sttCommandsMu.RLock()
	defer sttCommandsMu.RUnlock()
	bestLen := -1
	for p, c := range sttCommands {
		if !strings.HasPrefix(lower, p) {
			continue
		}
		if len(lower) > len(p) && lower[len(p)] != ' ' {
			continue
		}
		if len(p) > bestLen {
			bestLen = len(p)
			phrase = p
			cmd = c
			rest = strings.TrimSpace(txt[len(p):])
			ok = true
		}
	}
	return phrase, cmd, rest, ok
}

// dispatchSTTCommand runs a recognized voice command. Commands starting with
// "@hotkey" trigger a hotkey by combo, "@name" runs a macro function, "/name"
// routes to built-in or script-registered client commands, and anything else
// is sent to the server.
func dispatchSTTCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	lower := strings.ToLower(cmd)
	switch {
	case strings.HasPrefix(lower, "@hotkey"):
		combo := strings.TrimSpace(cmd[len("@hotkey"):])
		if !runHotkeyByCombo(combo) {
			logWarn("stt hotkey not found: %q", combo)
		}
	case strings.HasPrefix(lower, "@"):
		name := strings.TrimSpace(cmd[1:])
		if macroFuncExists(name) {
			macroExecFunc(name)
		} else {
			enqueueCommand(cmd)
		}
	case strings.HasPrefix(lower, "/"):
		if !handleClientCommand(cmd) {
			enqueueCommand(cmd)
		}
	default:
		enqueueCommand(cmd)
	}
}

// sttRefreshHotkey re-parses the hotkey setting into cached key code and mods.
func sttRefreshHotkey() {
	name := strings.TrimSpace(gs.STTHotkey)
	if name == "" {
		sttHotkeyCode = 0
		sttHotkeyMods = 0
		return
	}
	code, mods := macroGetKeyByName(name)
	sttHotkeyCode = code
	sttHotkeyMods = mods
}

// updateSTT is called once per frame from the game Update loop.
func updateSTT() {
	// Refresh the live mic level in the test window periodically while
	// listening; the dirty flag covers recognition events and state changes.
	sttTestFrame++
	if sttStarted && sttTestFrame%6 == 0 && sttTestWin != nil && sttTestWin.IsOpen() {
		updateSTTTestWindow()
	}
	if sttTestDirty.CompareAndSwap(true, false) {
		updateSTTTestWindow()
	}
	if isWASM {
		sttStop()
		return
	}
	refreshHotkeyIfChanged()
	for {
		select {
		case ev := <-sttTextCh:
			processSTTText(ev)
			continue
		default:
		}
		break
	}

	if !gs.STTEnabled {
		if sttStarted {
			logWarn("stt diag: auto-stop because !gs.STTEnabled")
			vosk.Stop()
			sttStarted = false
			vosk.Close()
		}
		return
	}

	if sttHotkeyCode == 0 {
		return
	}

	mods := macroCurrentMods()
	keyPressed, keyReleased := false, false
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		if ebitenKeyToMacroKey(k) == sttHotkeyCode {
			keyPressed = true
		}
	}
	for _, k := range inpututil.AppendJustReleasedKeys(nil) {
		if ebitenKeyToMacroKey(k) == sttHotkeyCode {
			keyReleased = true
		}
	}

	if gs.STTHotkeyToggle {
		if keyPressed && mods == sttHotkeyMods {
			sttToggle()
		}
		return
	}

	// Push-to-talk: listen while the combo is held.
	if keyPressed && mods == sttHotkeyMods {
		sttHeld = true
		sttStart()
	}
	if keyReleased && sttHeld {
		sttHeld = false
		sttStop()
	}
}

var cachedHotkeyName string

func refreshHotkeyIfChanged() {
	if cachedHotkeyName != gs.STTHotkey {
		cachedHotkeyName = gs.STTHotkey
		sttRefreshHotkey()
	}
}

// updateSTTStatus refreshes the settings-window status text for the STT
// section. It is assigned during window construction.
var updateSTTStatus func()

// sttRefreshMicList repopulates a microphone dropdown with the system default
// option plus the detected capture devices, preserving the current selection.
func sttRefreshMicList(dd *eui.ItemData) {
	if isWASM {
		dd.Options = []string{"(system default)"}
		dd.Selected = 0
		return
	}
	mics := vosk.ListMics()
	opts := make([]string, 0, len(mics)+1)
	opts = append(opts, "(system default)")
	opts = append(opts, mics...)
	dd.Options = opts
	sel := 0
	for i, name := range mics {
		if name == gs.STTMicrophone {
			sel = i + 1
			break
		}
	}
	dd.Selected = sel
}

// sttRefreshModelList repopulates the model dropdown with every model folder
// found under data/vosk, preserving the current selection.
func sttRefreshModelList(dd *eui.ItemData) {
	dir := sttDataDir()
	models := []string{}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() && vosk.ModelReady(filepath.Join(dir, e.Name())) {
				models = append(models, e.Name())
			}
		}
	}
	if len(models) == 0 {
		models = []string{"(no models in data/vosk)"}
	}
	dd.Options = models
	sel := 0
	for i, name := range models {
		if name == gs.STTModel {
			sel = i
			break
		}
	}
	dd.Selected = sel
}

const (
	voskLibBase     = "https://github.com/alphacep/vosk-api/releases/download/v0.3.45/"
	voskMacURL      = "https://files.pythonhosted.org/packages/44/19/5e8299237bc2005c3d155d3b48adba6fd6484465ad5c970302fc1d37947d/vosk-0.3.44-py3-none-macosx_10_6_universal2.whl"
	voskModelURLFmt = "https://alphacephei.com/vosk/models/%s.zip"
)

func voskModelURL(model string) string {
	return fmt.Sprintf(voskModelURLFmt, model)
}

func voskLibArchiveName() string {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "vosk-linux-x86_64-0.3.45.zip"
		case "arm64":
			return "vosk-linux-aarch64-0.3.45.zip"
		case "arm":
			return "vosk-linux-armv7l-0.3.45.zip"
		}
	case "windows":
		return "vosk-win64-0.3.45.zip"
	}
	return ""
}

// downloadVoskFiles fetches the native libvosk library and the configured
// model into the vosk data directory.
func downloadVoskFiles(getVosk bool) error {
	if !getVosk || isWASM {
		return nil
	}
	dir := sttDataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logError("create %v: %v", dir, err)
		return err
	}
	if !vosk.Available(sttLibPaths()) {
		if err := downloadVoskLib(dir); err != nil {
			return err
		}
	}
	if !vosk.ModelReady(sttModelDir()) {
		if err := downloadVoskModel(dir); err != nil {
			return err
		}
	}
	return nil
}

func downloadVoskLib(dir string) error {
	if runtime.GOOS == "darwin" {
		archPath := filepath.Join(dir, "vosk_macos.zip")
		if err := downloadFile(voskMacURL, archPath); err != nil {
			logError("download libvosk: %v", err)
			return fmt.Errorf("download libvosk: %w", err)
		}
		defer os.Remove(archPath)
		if err := extractArchive(archPath, dir); err != nil {
			logError("extract libvosk: %v", err)
			return fmt.Errorf("extract libvosk: %w", err)
		}
		if err := placeVoskLib(dir); err != nil {
			return err
		}
		_ = os.RemoveAll(filepath.Join(dir, "vosk"))
		return nil
	}
	name := voskLibArchiveName()
	if name == "" {
		return fmt.Errorf("no libvosk available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	archPath := filepath.Join(dir, name)
	if err := downloadFile(voskLibBase+name, archPath); err != nil {
		logError("download %v: %v", name, err)
		return fmt.Errorf("download libvosk: %w", err)
	}
	defer os.Remove(archPath)
	if err := extractArchive(archPath, dir); err != nil {
		logError("extract %v: %v", name, err)
		return fmt.Errorf("extract libvosk: %w", err)
	}
	return placeVoskLib(dir)
}

// placeVoskLib moves the extracted libvosk into the vosk directory root with
// the canonical per-platform name.
func placeVoskLib(dir string) error {
	dstName := "libvosk.so"
	switch runtime.GOOS {
	case "darwin":
		dstName = "libvosk.dylib"
	case "windows":
		dstName = "libvosk.dll"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	srcNames := []string{"libvosk.so", "libvosk.dll", "libvosk.dyld", "libvosk.dylib"}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		for _, src := range srcNames {
			srcPath := filepath.Join(dir, e.Name(), src)
			if info, err := os.Stat(srcPath); err == nil && !info.IsDir() {
				if err := os.Rename(srcPath, filepath.Join(dir, dstName)); err != nil {
					return err
				}
				_ = os.RemoveAll(filepath.Join(dir, e.Name()))
				return nil
			}
		}
	}
	return fmt.Errorf("libvosk not found after extraction")
}

func downloadVoskModel(dir string) error {
	model := gs.STTModel
	if model == "" {
		return fmt.Errorf("no STT model configured")
	}
	archPath := filepath.Join(dir, model+".zip")
	if err := downloadFile(voskModelURL(model), archPath); err != nil {
		logError("download %v: %v", model, err)
		return fmt.Errorf("download vosk model: %w", err)
	}
	defer os.Remove(archPath)
	if err := extractArchive(archPath, dir); err != nil {
		logError("extract %v: %v", model, err)
		return fmt.Errorf("extract vosk model: %w", err)
	}
	return nil
}
