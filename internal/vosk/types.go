package vosk

// TextEvent carries a speech-to-text result from the engine.
type TextEvent struct {
	Text  string
	Final bool
}

// Options configures the speech-to-text engine.
type Options struct {
	// LibPaths lists candidate paths for the native libvosk library
	// (libvosk.so / libvosk.dylib / libvosk.dll). The first existing path
	// that loads successfully is used.
	LibPaths []string
	// ModelPath is the directory of a Vosk model (contains am/final.mdl).
	ModelPath string
	// MicName selects a capture device; empty uses the default device.
	MicName string
	// OnText receives partial and final recognition results.
	OnText func(TextEvent)
}
