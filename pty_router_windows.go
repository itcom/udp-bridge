//go:build windows
// +build windows

package main

import "log"

// PTY is not supported on Windows.
// These are stub functions to allow compilation.

// GetPTYPaths returns empty slice on Windows (PTY not supported)
func GetPTYPaths() []string {
	return []string{}
}

// GetPTYPath returns empty string on Windows (PTY not supported)
func GetPTYPath() string {
	return ""
}

// startRigWatcherWithPTY is not supported on Windows.
// Logs a warning and does nothing (caller should fall back to non-PTY mode).
func startRigWatcherWithPTY() {
	log.Println("[RIG-PTY] PTY mode is not supported on Windows, ignoring UsePTY setting")
	// Do not call startRigWatcher() here to avoid infinite loop.
	// The rig watcher will not start in PTY mode on Windows.
	// User should disable UsePTY in settings on Windows.
}

// broadcastPTYPaths is a no-op on Windows
func broadcastPTYPaths() {
	// PTY not supported on Windows
}

// restartRigWatcherWithPTY is not supported on Windows.
// Falls back to normal rig watcher restart.
func restartRigWatcherWithPTY() {
	log.Println("[RIG-PTY] PTY mode is not supported on Windows, using normal restart")
	restartRigWatcher()
}
