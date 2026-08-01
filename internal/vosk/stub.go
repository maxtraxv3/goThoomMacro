//go:build !cgo

package vosk

import "fmt"

// This file provides no-op implementations so the package compiles on builds
// without cgo (e.g. js/wasm). Speech-to-text is unavailable in those builds.

// Available always returns false without cgo.
func Available(libPaths []string) bool { return false }

// ModelReady always returns false without cgo.
func ModelReady(modelPath string) bool { return false }

// ListMics returns no devices without cgo.
func ListMics() []string { return nil }

// Start always fails without cgo.
func Start(opts Options) error {
	return fmt.Errorf("speech-to-text unavailable in this build (requires cgo)")
}

// Stop is a no-op without cgo.
func Stop() {}

// Close is a no-op without cgo.
func Close() {}

// Listening always returns false without cgo.
func Listening() bool { return false }

// Level always returns 0 without cgo.
func Level() float64 { return 0 }

// ChunkCount always returns 0 without cgo.
func ChunkCount() uint64 { return 0 }

// SampleRate always returns 0 without cgo.
func SampleRate() float64 { return 0 }

// Gain always returns 1 without cgo.
func Gain() float64 { return 1 }
