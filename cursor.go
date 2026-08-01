package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
)

var (
	customCursorMu      sync.RWMutex
	customCursorNorm    *ebiten.Image
	customCursorMove    *ebiten.Image
	useCustomCursors    bool
	cursorDirty         bool
	prevCursorNormFile  string
	prevCursorMoveFile  string
)

type icoDirEntry struct {
	w, h      byte
	palette   byte
	reserved  byte
	planes    uint16
	bpp       uint16
	size      uint32
	offset    uint32
}

func decodeICO(data []byte) (image.Image, error) {
	var header struct {
		_     uint16
		_     uint16
		count uint16
	}
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	entries := make([]icoDirEntry, header.count)
	const entrySize = 16
	for i := range entries {
		if err := binary.Read(r, binary.LittleEndian, &entries[i]); err != nil {
			return nil, err
		}
	}
	// Try entries in order; prefer PNG data (most modern ICOs embed PNG)
	for _, e := range entries {
		if int(e.offset)+int(e.size) > len(data) {
			continue
		}
		chunk := data[e.offset : e.offset+e.size]
		// Check for PNG signature
		if len(chunk) > 8 && string(chunk[:8]) == "\x89PNG\r\n\x1a\n" {
			img, err := png.Decode(bytes.NewReader(chunk))
			if err == nil {
				return img, nil
			}
		}
	}
	// Fallback: try first entry as raw BMP with ICO header
	if len(entries) > 0 {
		e := entries[0]
		if int(e.offset)+int(e.size) <= len(data) {
			chunk := data[e.offset : e.offset+e.size]
			if len(chunk) > 4 && string(chunk[:4]) == "\x89PNG" {
				return png.Decode(bytes.NewReader(chunk))
			}
		}
	}
	return nil, fmt.Errorf("no supported image found in ICO")
}

func loadICO(path string) *ebiten.Image {
	data, err := os.ReadFile(path)
	if err != nil {
		logError("read ico %q: %v", path, err)
		return nil
	}
	img, err := decodeICO(data)
	if err != nil {
		logError("decode ico %q: %v", path, err)
		return nil
	}
	return scaleCursor(img)
}

func loadImageFile(path string) *ebiten.Image {
	f, err := os.Open(path)
	if err != nil {
		logError("open cursor %q: %v", path, err)
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		logError("decode cursor %q: %v", path, err)
		return nil
	}
	return scaleCursor(img)
}

func scaleCursor(img image.Image) *ebiten.Image {
	const maxSize = 32
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxSize || h > maxSize {
		scale := float64(maxSize) / float64(max(w, h))
		nw := int(float64(w) * scale)
		nh := int(float64(h) * scale)
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
		return ebiten.NewImageFromImage(dst)
	}
	return ebiten.NewImageFromImage(img)
}

func reloadCustomCursors() {
	customCursorMu.Lock()
	defer customCursorMu.Unlock()

	_ = os.MkdirAll(cursorDirPath(), 0o755)

	if customCursorNorm != nil {
		customCursorNorm.Deallocate()
		customCursorNorm = nil
	}
	if customCursorMove != nil {
		customCursorMove.Deallocate()
		customCursorMove = nil
	}

	normalPath := resolveCursorPath(gs.CursorNormalFile)
	movePath := resolveCursorPath(gs.CursorMoveFile)

	useCustomCursors = false

	if normalPath != "" {
		ext := strings.ToLower(filepath.Ext(normalPath))
		if ext == ".ico" {
			customCursorNorm = loadICO(normalPath)
		} else {
			customCursorNorm = loadImageFile(normalPath)
		}
		if customCursorNorm != nil {
			useCustomCursors = true
		} else {
			logError("failed to load normal cursor: %s", normalPath)
		}
	}

	if movePath != "" {
		ext := strings.ToLower(filepath.Ext(movePath))
		if ext == ".ico" {
			customCursorMove = loadICO(movePath)
		} else {
			customCursorMove = loadImageFile(movePath)
		}
		if customCursorMove != nil {
			useCustomCursors = true
		} else {
			logError("failed to load move cursor: %s", movePath)
		}
	}

	if useCustomCursors {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
	}

	prevCursorNormFile = gs.CursorNormalFile
	prevCursorMoveFile = gs.CursorMoveFile
	cursorDirty = false
}

func markCursorDirty() {
	cursorDirty = true
}

// cursorImageExts lists image extensions shown in the cursor dropdown.
var cursorImageExts = []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico"}

// cursorDirPath returns the folder that holds selectable cursor images.
func cursorDirPath() string {
	return filepath.Join(dataDirPath, "cursors")
}

// listCursorFiles returns the names of image files in the cursors folder,
// sorted alphabetically.
func listCursorFiles() []string {
	entries, err := os.ReadDir(cursorDirPath())
	if err != nil {
		return nil
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		for _, c := range cursorImageExts {
			if ext == c {
				files = append(files, e.Name())
				break
			}
		}
	}
	sort.Strings(files)
	return files
}

// resolveCursorPath converts a saved cursor setting into a file path. A bare
// filename is resolved inside the cursors folder; existing absolute or
// relative paths are kept for backward compatibility.
func resolveCursorPath(name string) string {
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) || strings.ContainsRune(name, filepath.Separator) {
		return name
	}
	return filepath.Join(cursorDirPath(), name)
}

func updateCustomCursors() {
	if !cursorDirty {
		return
	}
	SettingsLock.Lock()
	norm := gs.CursorNormalFile
	move := gs.CursorMoveFile
	SettingsLock.Unlock()

	if norm == prevCursorNormFile && move == prevCursorMoveFile {
		cursorDirty = false
		return
	}
	reloadCustomCursors()
}

func drawCustomCursor(screen *ebiten.Image, walk bool) {
	customCursorMu.RLock()
	norm := customCursorNorm
	move := customCursorMove
	use := useCustomCursors
	customCursorMu.RUnlock()

	if !use {
		return
	}

	x, y := ebiten.CursorPosition()
	cx, cy := float64(x), float64(y)

	var img *ebiten.Image
	if walk && move != nil {
		img = move
	} else if norm != nil {
		img = norm
	} else if move != nil {
		img = move
	} else {
		return
	}

	op := &ebiten.DrawImageOptions{}
	b := img.Bounds()
	op.GeoM.Translate(cx-float64(b.Dx())/2, cy-float64(b.Dy())/2)
	screen.DrawImage(img, op)
}
