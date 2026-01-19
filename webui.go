package main

import (
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strconv"
)

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"inc": func(i int) int { return i + 1 },
	"safeIndex": func(slice []string, i int) string {
		if i >= 0 && i < len(slice) {
			return slice[i]
		}
		return ""
	},
}).Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>HAMLAB Bridge 設定</title>
<style>
* {
  box-sizing: border-box;
}
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: #f5f5f5;
  margin: 0;
  padding: 20px;
}
.container {
  max-width: 500px;
  margin: 0 auto;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
  padding: 24px;
}
h1 {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 24px 0;
  color: #333;
  text-align: center;
}
.form-group {
  margin-bottom: 16px;
}
label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: #555;
  margin-bottom: 6px;
}
input[type="text"],
input[type="password"],
select {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid #ddd;
  border-radius: 6px;
  transition: border-color 0.2s;
  background: #fff;
}
input[type="text"]:focus,
input[type="password"]:focus,
select:focus {
  outline: none;
  border-color: #007aff;
}
.checkbox-group {
  margin: 20px 0;
  padding: 16px;
  background: #f9f9f9;
  border-radius: 8px;
}
.checkbox-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
}
.checkbox-item:first-child {
  padding-top: 0;
}
.checkbox-item:last-child {
  padding-bottom: 0;
}
.checkbox-item input[type="checkbox"] {
  width: 18px;
  height: 18px;
  margin-right: 10px;
  cursor: pointer;
}
.checkbox-item span {
  font-size: 14px;
  color: #333;
  cursor: pointer;
}
button {
  width: 100%;
  padding: 12px;
  font-size: 15px;
  font-weight: 500;
  color: #fff;
  background: #007aff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}
button:hover {
  background: #0056b3;
}
.message {
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 20px;
  font-size: 14px;
  display: flex;
  align-items: center;
}
.message.success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}
.message .icon {
  margin-right: 8px;
  font-size: 16px;
}
.version {
  text-align: center;
  font-size: 11px;
  color: #999;
  margin-top: 16px;
}
.pty-path {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #eee;
}
.pty-path label {
  display: block;
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
}
.pty-path input {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  font-family: monospace;
  background: #f0f0f0;
  border: 1px solid #ddd;
  border-radius: 4px;
  cursor: pointer;
}
.port-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}
.port-row .port-num {
  min-width: 20px;
  font-size: 13px;
  color: #666;
}
.port-row select {
  flex: 2;
}
.port-row select.baud {
  flex: 1;
}
.broadcast-mode {
  margin-top: 12px;
  padding: 12px;
  background: #f0f8ff;
  border-radius: 6px;
}
.broadcast-mode label {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  cursor: pointer;
}
.broadcast-mode label:last-child {
  margin-bottom: 0;
}
.broadcast-mode input[type="radio"] {
  margin-right: 8px;
}
.broadcast-mode .mode-desc {
  font-size: 11px;
  color: #666;
  margin-left: 24px;
}
</style>
</head>
<body>
<div class="container">
  <h1>HAMLAB Bridge 設定</h1>
  {{if .Saved}}
  <div class="message success">
    <span class="icon">✓</span>
    設定を保存しました
  </div>
  {{end}}
  <form method="post">
    <div class="form-group">
      <label for="user">QRZ.com ユーザー名</label>
      <input type="text" id="user" name="user" value="{{.Config.QRZUser}}" autocomplete="username">
    </div>
    <div class="form-group">
      <label for="pass">QRZ.com パスワード</label>
      <input type="password" id="pass" name="pass" value="{{.Config.QRZPass}}" autocomplete="current-password">
    </div>
    <div class="checkbox-group">
      <label class="checkbox-item">
        <input type="checkbox" name="use_qrz" {{if .Config.UseQRZ}}checked{{end}}>
        <span>QRZ.com から情報を補完</span>
      </label>
      <label class="checkbox-item">
        <input type="checkbox" name="use_geo" {{if .Config.UseGeo}}checked{{end}}>
        <span>JCC / 住所を自動補完</span>
      </label>
    </div>
    <div class="checkbox-group">
      <label class="checkbox-item">
        <input type="checkbox" name="use_rig" {{if .Config.UseRig}}checked{{end}}>
        <span>無線機（CAT / CI-V）から周波数・モードを取得</span>
      </label>
      {{if .HasPTY}}
      <label class="checkbox-item">
        <input type="checkbox" name="use_pty" {{if .Config.UsePTY}}checked{{end}}>
        <span>PTYルーター（WSJT-X等と共有）</span>
      </label>
      <div style="font-size:11px;color:#888;margin-top:4px;padding-left:28px;">※ PTYルーター・ポート・ボーレートの変更は再起動後に反映</div>
      {{else}}
      <div style="font-size:11px;color:#888;margin-top:4px;padding-left:28px;">※ ポート・ボーレートの変更は再起動後に反映</div>
      {{end}}
    </div>
    <div class="form-group">
      <label>CAT / CI-V ポート（最大5つ）</label>
      {{range $i, $rp := .Config.RigPorts}}
      <div class="port-row">
        <span class="port-num">{{inc $i}}</span>
        <select name="rig_port_{{$i}}">
          <option value="">-- 未使用 --</option>
          {{range $.Ports}}
          <option value="{{.}}"{{if eq . $rp.Port}} selected{{end}}>{{.}}</option>
          {{end}}
        </select>
        <select name="rig_baud_{{$i}}" class="baud">
          {{range $.Bauds}}
          <option value="{{.}}"{{if eq . $rp.Baud}} selected{{end}}>{{.}}</option>
          {{end}}
        </select>
      </div>
      {{if and $.HasPTY (safeIndex $.PTYPaths $i)}}
      <div class="pty-path" style="margin-left:28px;margin-bottom:12px;">
        <input type="text" value="{{safeIndex $.PTYPaths $i}}" readonly onclick="this.select()" title="PTYパス (ポート{{inc $i}})">
      </div>
      {{end}}
      {{end}}
      <div class="broadcast-mode">
        <label>
          <input type="radio" name="broadcast_mode" value="all"{{if eq .Config.RigBroadcastMode "all"}} checked{{end}}>
          全ポートを監視（自動選択）
        </label>
        <div class="mode-desc">データが来たポートを自動でbroadcast</div>
        <label>
          <input type="radio" name="broadcast_mode" value="single"{{if eq .Config.RigBroadcastMode "single"}} checked{{end}}>
          選択したポートのみ
        </label>
        <div class="mode-desc" style="display:flex;align-items:center;gap:8px;">
          ポート番号:
          <select name="selected_rig_index" style="width:60px;padding:4px;">
            {{range $i, $_ := .Config.RigPorts}}
            <option value="{{$i}}"{{if eq $i $.Config.SelectedRigIndex}} selected{{end}}>{{inc $i}}</option>
            {{end}}
          </select>
        </div>
      </div>
    </div>
    <div class="checkbox-group">
      <div style="font-weight:600;margin-bottom:12px;color:#333;">📚 Logbook連携</div>
      <label class="checkbox-item">
        <input type="checkbox" name="logbook_qrz_enabled" {{if .Config.LogbookQRZEnabled}}checked{{end}}>
        <span>QRZ.com Logbook</span>
      </label>
      <div class="form-group" style="margin-left:28px;">
        <label for="logbook_qrz_apikey">API Key</label>
        <input type="password" id="logbook_qrz_apikey" name="logbook_qrz_apikey" value="{{.Config.LogbookQRZAPIKey}}">
      </div>
      <label class="checkbox-item">
        <input type="checkbox" name="logbook_hamqth_enabled" {{if .Config.LogbookHamQTHEnabled}}checked{{end}}>
        <span>HamQTH</span>
      </label>
      <div class="form-group" style="margin-left:28px;">
        <label for="logbook_hamqth_callsign">Callsign</label>
        <input type="text" id="logbook_hamqth_callsign" name="logbook_hamqth_callsign" value="{{.Config.LogbookHamQTHCallsign}}">
        <label for="logbook_hamqth_user">ユーザー名</label>
        <input type="text" id="logbook_hamqth_user" name="logbook_hamqth_user" value="{{.Config.LogbookHamQTHUser}}">
        <label for="logbook_hamqth_pass">パスワード</label>
        <input type="password" id="logbook_hamqth_pass" name="logbook_hamqth_pass" value="{{.Config.LogbookHamQTHPass}}">
      </div>
      <label class="checkbox-item">
        <input type="checkbox" name="logbook_eqsl_enabled" {{if .Config.LogbookEQSLEnabled}}checked{{end}}>
        <span>eQSL.cc</span>
      </label>
      <div class="form-group" style="margin-left:28px;">
        <label for="logbook_eqsl_user">ユーザー名</label>
        <input type="text" id="logbook_eqsl_user" name="logbook_eqsl_user" value="{{.Config.LogbookEQSLUser}}">
        <label for="logbook_eqsl_pass">パスワード</label>
        <input type="password" id="logbook_eqsl_pass" name="logbook_eqsl_pass" value="{{.Config.LogbookEQSLPass}}">
      </div>
      <label class="checkbox-item">
        <input type="checkbox" name="logbook_hrdlog_enabled" {{if .Config.LogbookHRDLogEnabled}}checked{{end}}>
        <span>HRDLog.net</span>
      </label>
      <div class="form-group" style="margin-left:28px;">
        <label for="logbook_hrdlog_callsign">Callsign</label>
        <input type="text" id="logbook_hrdlog_callsign" name="logbook_hrdlog_callsign" value="{{.Config.LogbookHRDLogCallsign}}">
        <label for="logbook_hrdlog_code">Upload Code</label>
        <input type="password" id="logbook_hrdlog_code" name="logbook_hrdlog_code" value="{{.Config.LogbookHRDLogCode}}">
      </div>
    </div>
    <button type="submit">保存</button>
  </form>
  <div class="version">HAMLAB Bridge v0.4.0</div>
</div>
</body>
</html>
`))

type PageData struct {
	Config   Config
	Saved    bool
	PTYPaths []string
	Ports    []string
	Bauds    []int
	HasPTY   bool
}

var defaultBauds = []int{4800, 9600, 19200, 38400, 57600, 115200}

// startWebUI starts a web server on localhost:17801 that serves a settings page.
// The page allows the user to set QRZ user and password, and to toggle the use of QRZ and geo lookup.
// When the form is submitted, the settings are saved and the user is redirected back to the settings page.
func startWebUI() {

	http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if r.Method == "POST" {
			_ = r.ParseForm()

			configLock.Lock()
			// 変更検知のため古い設定を保存
			oldUseRig := config.UseRig
			oldUsePTY := config.UsePTY
			oldPorts := make([]RigPortConfig, len(config.RigPorts))
			copy(oldPorts, config.RigPorts)
			oldBroadcastMode := config.RigBroadcastMode
			oldSelectedIndex := config.SelectedRigIndex

			config.QRZUser = r.FormValue("user")
			config.QRZPass = r.FormValue("pass")
			config.UseQRZ = r.FormValue("use_qrz") != ""
			config.UseGeo = r.FormValue("use_geo") != ""
			config.UseRig = r.FormValue("use_rig") != ""
			config.UsePTY = r.FormValue("use_pty") != ""

			// 複数ポート設定の読み取り
			for i := 0; i < 5; i++ {
				portKey := "rig_port_" + strconv.Itoa(i)
				baudKey := "rig_baud_" + strconv.Itoa(i)
				config.RigPorts[i].Port = r.FormValue(portKey)
				if v := r.FormValue(baudKey); v != "" {
					if baud, err := strconv.Atoi(v); err == nil {
						config.RigPorts[i].Baud = baud
					}
				}
			}

			// 後方互換性: RigPorts[0]をRigPort/RigBaudにも反映
			config.RigPort = config.RigPorts[0].Port
			config.RigBaud = config.RigPorts[0].Baud

			// ブロードキャストモード
			config.RigBroadcastMode = r.FormValue("broadcast_mode")
			if config.RigBroadcastMode == "" {
				config.RigBroadcastMode = "all"
			}

			// 選択ポートインデックス
			if v := r.FormValue("selected_rig_index"); v != "" {
				if idx, err := strconv.Atoi(v); err == nil && idx >= 0 && idx < 5 {
					config.SelectedRigIndex = idx
				}
			}

			// Logbook連携設定
			config.LogbookQRZEnabled = r.FormValue("logbook_qrz_enabled") != ""
			config.LogbookQRZAPIKey = r.FormValue("logbook_qrz_apikey")
			config.LogbookHamQTHEnabled = r.FormValue("logbook_hamqth_enabled") != ""
			config.LogbookHamQTHCallsign = r.FormValue("logbook_hamqth_callsign")
			config.LogbookHamQTHUser = r.FormValue("logbook_hamqth_user")
			config.LogbookHamQTHPass = r.FormValue("logbook_hamqth_pass")
			config.LogbookEQSLEnabled = r.FormValue("logbook_eqsl_enabled") != ""
			config.LogbookEQSLUser = r.FormValue("logbook_eqsl_user")
			config.LogbookEQSLPass = r.FormValue("logbook_eqsl_pass")
			config.LogbookHRDLogEnabled = r.FormValue("logbook_hrdlog_enabled") != ""
			config.LogbookHRDLogCallsign = r.FormValue("logbook_hrdlog_callsign")
			config.LogbookHRDLogCode = r.FormValue("logbook_hrdlog_code")
			// ClubLogは430ssb.net中継経由で実装予定（ペンディング）

			// リグ設定の変更をチェック
			rigSettingsChanged := false
			if oldUseRig != config.UseRig || oldUsePTY != config.UsePTY {
				rigSettingsChanged = true
			}
			if oldBroadcastMode != config.RigBroadcastMode || oldSelectedIndex != config.SelectedRigIndex {
				rigSettingsChanged = true
			}
			for i := range config.RigPorts {
				if oldPorts[i].Port != config.RigPorts[i].Port || oldPorts[i].Baud != config.RigPorts[i].Baud {
					rigSettingsChanged = true
					break
				}
			}

			saveConfig()
			configLock.Unlock()

			// リグ設定が変更された場合は再起動（非同期）
			if rigSettingsChanged && config.UseRig {
				log.Println("[CONFIG] rig settings changed, restarting rig watcher...")
				go func() {
					if config.UsePTY && (runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
						restartRigWatcherWithPTY()
					} else {
						restartRigWatcher()
					}
					// Auto Information を再有効化（WSJT-X等がAI0;を送るため）
					SendAI1()
				}()
			}

			http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
			return
		}

		configLock.RLock()
		data := PageData{
			Config:   config,
			Saved:    r.URL.Query().Get("saved") == "1",
			PTYPaths: GetPTYPaths(),
			Ports:    listSerialPorts(),
			Bauds:    defaultBauds,
			HasPTY:   runtime.GOOS == "darwin" || runtime.GOOS == "linux",
		}
		configLock.RUnlock()

		_ = tmpl.Execute(w, data)
	})

	http.ListenAndServe("127.0.0.1:17801", nil)
	log.Println("Settings UI: http://127.0.0.1:17801/settings")
}
