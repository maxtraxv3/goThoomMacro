package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// clientVersion is the semver version of this fork, set at build time via:
//
//	-ldflags "-X main.clientVersion=v1.0.0"
//
// If unset, auto-update is disabled.
var clientVersion = "dev"

const githubReleasesURL = "https://api.github.com/repos/maxtraxv3/goThoomMacro/releases/latest"

type githubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// semverParts splits a "v1.2.3" or "1.2.3" string into major, minor, patch.
// Returns -1 for any unparseable part.
func semverParts(tag string) (int, int, int) {
	tag = strings.TrimPrefix(tag, "v")
	parts := strings.SplitN(tag, ".", 3)
	major, minor, patch := -1, -1, -1
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}
	return major, minor, patch
}

// semverNewer returns true if 'a' is strictly newer than 'b'.
func semverNewer(a, b string) bool {
	am, ai, ap := semverParts(a)
	bm, bi, bp := semverParts(b)
	if am != bm {
		return am > bm
	}
	if ai != bi {
		return ai > bi
	}
	return ap > bp
}

// checkClientUpdate checks GitHub for a newer release of this fork.
// If found, it prompts the user and auto-updates.
func checkClientUpdate() {
	if clientVersion == "dev" || clientVersion == "" {
		return
	}

	resp, err := http.Get(githubReleasesURL)
	if err != nil {
		log.Printf("client update check: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("client update check: HTTP %d", resp.StatusCode)
		return
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Printf("client update check: %v", err)
		return
	}

	if release.TagName == "" {
		return
	}

	if !semverNewer(release.TagName, clientVersion) {
		consoleMessage(fmt.Sprintf("Client is up to date (%s)", clientVersion))
		return
	}

	consoleMessage(fmt.Sprintf("Client update available: %s (current: %s)", release.TagName, clientVersion))

	// Find the platform-appropriate binary asset.
	binaryName := runtimeBinaryName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName || asset.Name == binaryName+".exe" {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	// Fallback: if no exact match, look for any asset matching the OS/arch pattern.
	if downloadURL == "" {
		pattern := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		for _, asset := range release.Assets {
			if strings.Contains(strings.ToLower(asset.Name), pattern) {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}
	if downloadURL == "" {
		log.Printf("client update: no matching binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
		go showNotification(fmt.Sprintf("Update %s available but no binary for %s/%s", release.TagName, runtime.GOOS, runtime.GOARCH))
		return
	}

	// Ask the user before updating.
	go func() {
		for !uiReady {
			time.Sleep(100 * time.Millisecond)
		}
		showPopup(
			"Client Update",
			fmt.Sprintf("A new version (%s) is available.\nCurrent: %s\n\nUpdate now?", release.TagName, clientVersion),
			[]popupButton{
				{Text: "Later"},
				{Text: "Update Now", Action: func() {
					go func() {
						if err := downloadAndRestart(downloadURL); err != nil {
							logError("client update failed: %v", err)
							go showNotification(fmt.Sprintf("Update failed: %v", err))
						}
					}()
				}},
			},
		)
	}()
}

// runtimeBinaryName returns the expected name of the running binary.
func runtimeBinaryName() string {
	name := "gothoom"
	if runtime.GOOS == "windows" {
		name = "gothoom.exe"
	}
	return name
}

// downloadAndRestart downloads the new binary, replaces the current one, and
// restarts the process.
func downloadAndRestart(url string) error {
	if downloadStatus != nil {
		downloadStatus("Downloading client update...")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	// Download to a temporary file next to the current binary.
	dir := filepath.Dir(exe)
	tmp := filepath.Join(dir, ".gothoom.update.tmp")

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %s", resp.Status)
	}

	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp: %w", err)
	}

	// Make executable.
	if err := os.Chmod(tmp, 0755); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("chmod: %w", err)
	}

	// Replace the current binary.
	// On Linux/macOS: rename current to .old, rename new to current.
	// On Windows: can't rename over a running exe, so we rename new to .new
	// and let the launcher handle it.
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename current: %w", err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		// Try to restore.
		os.Rename(old, exe)
		os.Remove(tmp)
		return fmt.Errorf("rename new: %w", err)
	}

	consoleMessage(fmt.Sprintf("Updated to %s — restarting...", releaseTag(url)))

	// Restart the process.
	if err := restartProcess(exe); err != nil {
		logError("restart failed: %v — please restart manually", err)
		os.Exit(0)
	}
	return nil
}

// restartProcess replaces the current process with a fresh copy of itself.
func restartProcess(exe string) error {
	// Clean up .old from previous update.
	old := exe + ".old"
	_ = os.Remove(old)

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	// Exit the old process after the new one starts.
	os.Exit(0)
	return nil
}

func releaseTag(downloadURL string) string {
	// Best-effort: extract tag from GitHub URL path.
	// e.g. .../releases/download/v1.2.3/gothoom
	i := strings.LastIndex(downloadURL, "/releases/download/")
	if i < 0 {
		return "latest"
	}
	rest := downloadURL[i+len("/releases/download/"):]
	j := strings.IndexByte(rest, '/')
	if j < 0 {
		return rest
	}
	return rest[:j]
}
