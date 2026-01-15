package main

import (
	"html/template"
	"net/http"
)

var tmpl = template.Must(template.New("").Parse(`<!DOCTYPE html>
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
  max-width: 400px;
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
input[type="password"] {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid #ddd;
  border-radius: 6px;
  transition: border-color 0.2s;
}
input[type="text"]:focus,
input[type="password"]:focus {
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
    <button type="submit">保存</button>
  </form>
  <div class="version">HAMLAB Bridge v0.1.0</div>
</div>
</body>
</html>
`))

type PageData struct {
	Config Config
	Saved  bool
}

// startWebUI starts a web server on localhost:17801 that serves a settings page.
// The page allows the user to set QRZ user and password, and to toggle the use of QRZ and geo lookup.
// When the form is submitted, the settings are saved and the user is redirected back to the settings page.
func startWebUI() {
	http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if r.Method == "POST" {
			_ = r.ParseForm()

			configLock.Lock()
			config.QRZUser = r.FormValue("user")
			config.QRZPass = r.FormValue("pass")
			config.UseQRZ = r.FormValue("use_qrz") != ""
			config.UseGeo = r.FormValue("use_geo") != ""
			saveConfig()
			configLock.Unlock()

			http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
			return
		}

		configLock.RLock()
		data := PageData{
			Config: config,
			Saved:  r.URL.Query().Get("saved") == "1",
		}
		configLock.RUnlock()

		_ = tmpl.Execute(w, data)
	})

	http.ListenAndServe("127.0.0.1:17801", nil)
}
