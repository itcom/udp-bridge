package main

import (
	"html/template"
	"net/http"
)

var tmpl = template.Must(template.New("").Parse(`
<h2>HAMLAB Bridge 設定</h2>
<form method="post">
QRZ User:<br><input name="user" value="{{.QRZUser}}"><br>
QRZ Pass:<br><input name="pass" type="password" value="{{.QRZPass}}"><br><br>

<label><input type="checkbox" name="use_qrz" {{if .UseQRZ}}checked{{end}}> QRZ補完</label><br>
<label><input type="checkbox" name="use_geo" {{if .UseGeo}}checked{{end}}> JCC/住所補完</label><br><br>

<button>保存</button>
</form>
`))

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

			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		configLock.RLock()
		defer configLock.RUnlock()
		_ = tmpl.Execute(w, config)
	})

	http.ListenAndServe("127.0.0.1:17801", nil)
}
