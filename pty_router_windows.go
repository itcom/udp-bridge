//go:build windows

package main

// GetPTYPath returns empty string on Windows (PTY not supported)
func GetPTYPath() string {
	return ""
}

// startRigWatcherWithPTY is not supported on Windows
func startRigWatcherWithPTY() {
	// PTY is not available on Windows
}
