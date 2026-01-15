package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// setupLaunchAgent は、HAMLAB Bridge をログイン時に自動起動するための
// launchctl 設定を設置します。
// すでに launchctl 設定が存在する場合は何も行いません。
// その場合は、osascript を使用してダイアログを表示し、
// ユーザがあった場合にのみ launchctl 設定を設置します。
// 設定は、$HOME/Library/LaunchAgents/jp.hamlab.bridge.plist に配置されます。
func setupLaunchAgent() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	agentDir := filepath.Join(home, "Library", "LaunchAgents")
	agentPath := filepath.Join(agentDir, "jp.hamlab.bridge.plist")

	if _, err := os.Stat(agentPath); err == nil {
		return // すでに有効
	}

	resp := exec.Command(
		"osascript",
		"-e",
		`display dialog "HAMLAB Bridge をログイン時に自動起動しますか？" buttons {"しない","有効にする"} default button 2`,
	).Run()

	if resp != nil {
		return
	}

	os.MkdirAll(agentDir, 0755)

	exec.Command("cp",
		"/Applications/HAMLAB Bridge.app/Contents/Resources/jp.hamlab.bridge.plist",
		agentPath,
	).Run()

	exec.Command("launchctl", "load", agentPath).Run()
}
