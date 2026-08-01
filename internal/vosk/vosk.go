//go:build cgo

package vosk

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gen2brain/malgo"
)

const sampleRate = 16000

// Auto-gain settings: Vosk needs reasonably loud normalized audio, but many
// mics deliver a weak signal. AGC amplifies quiet input toward agcTarget.
const (
	agcTarget  = 0.5
	agcMaxGain = 25.0
)

// Noise gate: Vosk's acoustic model tends to hallucinate filler words ("the",
// "and", "um") from mic noise and room tone, no matter the model size. A stateful
// gate opens when a chunk is above the gate level, then stays open for
// hangoverChunks (a short "hangover") so the tails of words aren't chopped. While
// the gate is closed the recognizer is fed silence instead, keeping hallucinated
// words out of results.
//
// The gate level adapts: the quietest recent level is tracked as the noise floor
// and the gate sits a few times above it, so it works across quiet and noisy
// rooms. The floor only updates while the gate is closed, so speech can never
// push it up.
const (
	noiseGateRms    = 0.01
	noiseGateFactor = 3.0
	hangoverChunks  = 10 // ~400ms at 40ms chunks
)

var (
	engMu        sync.Mutex
	engModel     unsafe.Pointer
	engRec       unsafe.Pointer
	micMu        sync.Mutex
	micCtx       *malgo.AllocatedContext
	engDev       *malgo.Device
	listening    atomic.Bool
	level        atomic.Uint64
	gain         atomic.Uint64
	chunks       atomic.Uint64
	totalSamples atomic.Uint64
	workerStart  time.Time
	agcPeak      float64
	feedCh       chan []float32
	stopCh       chan struct{}
	workerDone   chan struct{}
)

// Available reports whether a loadable libvosk library exists at any of the
// given candidate paths.
func Available(libPaths []string) bool {
	if libHandle != nil {
		return true
	}
	for _, p := range libPaths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// ModelReady reports whether a Vosk model directory is present and looks
// loadable (contains am/final.mdl).
func ModelReady(modelPath string) bool {
	if modelPath == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(modelPath, "am", "final.mdl"))
	return err == nil && !info.IsDir()
}

// ListMics returns the names of all capture devices detected on the system.
func ListMics() []string {
	ctx, err := ensureCtx()
	if err != nil {
		return nil
	}
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(infos))
	for i := range infos {
		names = append(names, infos[i].Name())
	}
	return names
}

// ensureCtx lazily creates a single shared malgo context that lives for the
// duration of the process. Creating and destroying a context per call proved
// unreliable inside the Ebiten runtime, so the context is created once and
// never uninitialized (the OS reclaims it on exit).
func ensureCtx() (*malgo.AllocatedContext, error) {
	micMu.Lock()
	defer micMu.Unlock()
	if micCtx == nil {
		ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
		if err != nil {
			return nil, err
		}
		micCtx = ctx
	}
	return micCtx, nil
}

// Start begins listening with the given options. Only one capture session may
// run at a time; Start is a no-op if already listening. The model and
// recognizer are cached, so repeated Start/Stop pairs are cheap (push-to-talk).
func Start(opts Options) error {
	engMu.Lock()
	defer engMu.Unlock()

	if listening.Load() {
		return nil
	}
	if err := ensureLib(opts.LibPaths); err != nil {
		return err
	}
	if engModel == nil {
		if !ModelReady(opts.ModelPath) {
			return fmt.Errorf("vosk model not found at %s", opts.ModelPath)
		}
		if err := initModel(opts); err != nil {
			return err
		}
	}

	ctx, err := ensureCtx()
	if err != nil {
		return fmt.Errorf("init audio context: %w", err)
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = 1
	cfg.SampleRate = sampleRate
	cfg.PeriodSizeInMilliseconds = 40

	if opts.MicName != "" {
		if id, ok := findMicID(ctx.Context, opts.MicName); ok {
			// The device ID must live in C memory: Go 1.26's cgo checks
			// reject Go pointers stored inside structs passed to C unless
			// pinned, and miniaudio only reads the ID during InitDevice.
			cid := C.malloc(C.size_t(len(id)))
			if cid != nil {
				*(*malgo.DeviceID)(cid) = id
				defer C.free(cid)
				cfg.Capture.DeviceID = cid
			}
		}
	}

	cb := malgo.DeviceCallbacks{
		Data: func(_out, in []byte, _frames uint32) {
			if len(in) == 0 {
				return
			}
			chunk := make([]float32, len(in)/4)
			var sum float64
			for i := range chunk {
				v := *(*float32)(unsafe.Pointer(&in[i*4]))
				chunk[i] = v
				sum += float64(v) * float64(v)
			}
			// RMS level of the current chunk for the live mic meter.
			level.Store(math.Float64bits(math.Sqrt(sum / float64(len(chunk)))))
			select {
			case feedCh <- chunk:
			default:
			}
		},
	}

	dev, err := malgo.InitDevice(ctx.Context, cfg, cb)
	if err != nil {
		return fmt.Errorf("init mic device: %w", err)
	}
	level.Store(0)
	if err := dev.Start(); err != nil {
		dev.Uninit()
		return fmt.Errorf("start mic: %w", err)
	}

	feedCh = make(chan []float32, 8)
	stopCh = make(chan struct{})
	workerDone = make(chan struct{})

	engDev = dev
	listening.Store(true)
	chunks.Store(0)
	totalSamples.Store(0)
	agcPeak = 0
	workerStart = time.Now()
	go worker(opts.OnText)
	return nil
}

// Stop stops listening but keeps the model and recognizer loaded for a fast
// restart.
func Stop() {
	engMu.Lock()
	defer engMu.Unlock()

	if !listening.Load() {
		return
	}
	listening.Store(false)

	close(stopCh)
	if engDev != nil {
		_ = engDev.Stop()
	}
	<-workerDone
	close(feedCh)
	level.Store(0)

	if engDev != nil {
		engDev.Uninit()
	}
	engDev = nil
}

// Close fully tears down the engine, freeing the model and recognizer. Call it
// when speech-to-text is permanently disabled or the app exits.
func Close() {
	Stop()
	engMu.Lock()
	defer engMu.Unlock()
	if engRec != nil {
		recognizerFree(engRec)
		engRec = nil
	}
	if engModel != nil {
		modelFree(engModel)
		engModel = nil
	}
	if libHandle != nil {
		closeLibrary()
		libHandle = nil
	}
}

// Listening reports whether a capture session is currently active.
func Listening() bool {
	return listening.Load()
}

// Level returns the RMS level (0..1) of the most recent mic audio chunk.
// It stays at 0 while no audio is being captured.
func Level() float64 {
	return math.Float64frombits(level.Load())
}

// ChunkCount returns how many audio chunks have been fed to the recognizer.
// A rising count confirms the worker is receiving mic audio.
func ChunkCount() uint64 {
	return chunks.Load()
}

// SampleRate reports the measured audio rate (samples per second) reaching the
// recognizer. Vosk expects 16000; a wildly different value (e.g. 48000) means
// the mic delivers a different rate than configured.
func SampleRate() float64 {
	total := totalSamples.Load()
	if workerStart.IsZero() {
		return 0
	}
	secs := time.Since(workerStart).Seconds()
	if secs <= 0 {
		return 0
	}
	return float64(total) / secs
}

// Gain reports the current auto-gain multiplier applied to the mic audio.
func Gain() float64 {
	return math.Float64frombits(gain.Load())
}

func initModel(opts Options) error {
	model := modelNew(opts.ModelPath)
	if model == nil {
		return fmt.Errorf("failed to load vosk model")
	}

	rec := recognizerNew(model, sampleRate)
	if rec == nil {
		modelFree(model)
		return fmt.Errorf("failed to create vosk recognizer")
	}
	engModel, engRec = model, rec
	return nil
}

func ensureLib(libPaths []string) error {
	if libHandle != nil {
		return nil
	}
	for _, p := range libPaths {
		h := loadLibrary(p)
		if h == nil {
			continue
		}
		libHandle = h
		setLogLevel(1)
		return nil
	}
	return fmt.Errorf("libvosk native library not found (checked %v)", libPaths)
}

func findMicID(ctx malgo.Context, name string) (malgo.DeviceID, bool) {
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return malgo.DeviceID{}, false
	}
	for _, info := range infos {
		if strings.EqualFold(strings.TrimSpace(info.Name()), strings.TrimSpace(name)) {
			return info.ID, true
		}
	}
	return malgo.DeviceID{}, false
}

func worker(onText func(TextEvent)) {
	defer close(workerDone)
	lastPartial := time.Time{}
	var inSpeech bool
	var silentChunks int
	noiseFloor := noiseGateRms
	for {
		select {
		case <-stopCh:
			if engRec != nil && onText != nil {
				txt := finalResult(engRec)
				if s := parseResultText(txt); s != "" {
					onText(TextEvent{Text: s, Final: true})
				}
			}
			return
		case chunk, ok := <-feedCh:
			if !ok {
				return
			}
			if len(chunk) == 0 {
				continue
			}
			chunks.Add(1)
			totalSamples.Add(uint64(len(chunk)))
			// Adaptive gain so quiet mics still register with Vosk.
			var peak float64
			var sumSq float64
			for _, v := range chunk {
				f := float64(v)
				if a := math.Abs(f); a > peak {
					peak = a
				}
				sumSq += f * f
			}
			rms := math.Sqrt(sumSq / float64(len(chunk)))
			// Stateful noise gate: open during speech, hold through short gaps,
			// and adapt the gate level to the room's noise floor.
			gate := noiseGateRms
			if nf := noiseFloor * noiseGateFactor; nf > gate {
				gate = nf
			}
			if rms < gate {
				if rms < noiseFloor {
					noiseFloor = noiseFloor*0.5 + rms*0.5
				} else {
					noiseFloor += (rms - noiseFloor) * 0.05
				}
				silentChunks++
				if silentChunks > hangoverChunks {
					inSpeech = false
				}
			} else {
				inSpeech = true
				silentChunks = 0
			}
			if peak > agcPeak {
				agcPeak = agcPeak*0.5 + peak*0.5
			} else {
				agcPeak = agcPeak*0.99 + peak*0.01
			}
			g := agcTarget / (agcPeak + 1e-4)
			if g > agcMaxGain {
				g = agcMaxGain
			}
			if g < 1 {
				g = 1
			}
			gain.Store(math.Float64bits(g))
			for i := range chunk {
				v := chunk[i] * float32(g)
				if !inSpeech {
					v = 0
				}
				if v > 1 {
					v = 1
				} else if v < -1 {
					v = -1
				}
				// Vosk's float API (accept_waveform_f) expects samples at int16
				// scale (±32767), not the ±1 range the mic delivers. Without
				// this the audio is ~32768x too quiet and Vosk hears silence.
				chunk[i] = v * 32768
			}
			if acceptWaveformF(engRec, chunk) {
				txt := result(engRec)
				if s := parseResultText(txt); s != "" && onText != nil {
					onText(TextEvent{Text: s, Final: true})
				}
				lastPartial = time.Time{}
			} else if onText != nil && time.Since(lastPartial) >= 300*time.Millisecond {
				lastPartial = time.Now()
				txt := partialResult(engRec)
				if s := parseResultText(txt); s != "" {
					onText(TextEvent{Text: s})
				}
			}
		}
	}
}

// parseResultText extracts recognized text from a Vosk result. Final results
// use the "text" field while live partial results use "partial", so both are
// handled here.
func parseResultText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var v struct {
		Text    string `json:"text"`
		Partial string `json:"partial"`
	}
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		if v.Text != "" {
			return v.Text
		}
		return v.Partial
	}
	return s
}
