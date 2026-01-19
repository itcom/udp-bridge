package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// submitLogbookAsync は非同期でADIFを各オンラインログサービスに送信します
func submitLogbookAsync(adif string) {
	configLock.RLock()
	qrzEnabled := config.LogbookQRZEnabled
	qrzAPIKey := config.LogbookQRZAPIKey
	hamqthEnabled := config.LogbookHamQTHEnabled
	hamqthCallsign := config.LogbookHamQTHCallsign
	hamqthUser := config.LogbookHamQTHUser
	hamqthPass := config.LogbookHamQTHPass
	eqslEnabled := config.LogbookEQSLEnabled
	eqslUser := config.LogbookEQSLUser
	eqslPass := config.LogbookEQSLPass
	hrdlogEnabled := config.LogbookHRDLogEnabled
	hrdlogCall := config.LogbookHRDLogCallsign
	hrdlogCode := config.LogbookHRDLogCode
	// clublogEnabled := config.LogbookClubLogEnabled
	// clublogEmail := config.LogbookClubLogEmail
	// clublogPass := config.LogbookClubLogPass
	// clublogCall := config.LogbookClubLogCall
	// clublogAPI := config.LogbookClubLogAPI
	configLock.RUnlock()

	// QRZ Logbook
	if qrzEnabled && qrzAPIKey != "" {
		go submitQRZLogbook(adif, qrzAPIKey)
	}

	// HamQTH
	if hamqthEnabled && hamqthCallsign != "" && hamqthUser != "" && hamqthPass != "" {
		go submitHamQTH(adif, hamqthCallsign, hamqthUser, hamqthPass)
	}

	// eQSL
	if eqslEnabled && eqslUser != "" && eqslPass != "" {
		go submitEQSL(adif, eqslUser, eqslPass)
	}

	// HRDLog
	if hrdlogEnabled && hrdlogCall != "" && hrdlogCode != "" {
		go submitHRDLog(adif, hrdlogCall, hrdlogCode)
	}

	// ClubLog (430ssb.net中継経由で実装予定 - ペンディング)
	// if clublogEnabled && clublogEmail != "" && clublogPass != "" && clublogCall != "" {
	// 	go submitClubLog(adif, clublogEmail, clublogPass, clublogCall, clublogAPI)
	// }
}

// submitQRZLogbook はQRZ.com Logbookへ送信します
func submitQRZLogbook(adif, apikey string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[LOGBOOK] QRZ panic:", r)
		}
	}()

	log.Println("[LOGBOOK] QRZ: sending...")
	log.Printf("[LOGBOOK] QRZ: KEY=%s", apikey)
	log.Printf("[LOGBOOK] QRZ: ADIF=%s", adif)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://logbook.qrz.com/api", url.Values{
		"KEY":    {apikey},
		"ACTION": {"INSERT"},
		"ADIF":   {adif},
	})
	if err != nil {
		log.Println("[LOGBOOK] QRZ error:", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取る
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseText := string(body[:n])

	if resp.StatusCode == 200 {
		log.Printf("[LOGBOOK] QRZ: success - response: %s", responseText)
	} else {
		log.Printf("[LOGBOOK] QRZ: failed, status: %d, response: %s", resp.StatusCode, responseText)
	}
}

// submitHamQTH はHamQTHへ送信します
func submitHamQTH(adif, callsign, user, pass string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[LOGBOOK] HamQTH panic:", r)
		}
	}()

	log.Println("[LOGBOOK] HamQTH: sending...")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://www.hamqth.com/qso_realtime.php", url.Values{
		"c":    {callsign},
		"u":    {user},
		"p":    {pass},
		"adif": {adif},
		"prg":  {"hamlab-bridge"},
		"cmd":  {"insert"},
	})
	if err != nil {
		log.Println("[LOGBOOK] HamQTH error:", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取る
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseText := string(body[:n])

	if resp.StatusCode == 200 {
		log.Printf("[LOGBOOK] HamQTH: success (response: %s)", responseText)
	} else {
		log.Printf("[LOGBOOK] HamQTH: failed, status: %d, response: %s", resp.StatusCode, responseText)
	}
}

// submitEQSL はeQSL.ccへ送信します
func submitEQSL(adif, user, pass string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[LOGBOOK] eQSL panic:", r)
		}
	}()

	log.Println("[LOGBOOK] eQSL: sending...")

	// GETリクエストのURLを構築
	params := url.Values{}
	params.Set("ADIFData", adif)
	params.Set("EQSL_USER", user)
	params.Set("EQSL_PSWD", pass)

	reqURL := "https://www.eqsl.cc/qslcard/importADIF.cfm?" + params.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		log.Println("[LOGBOOK] eQSL error:", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取る
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseText := string(body[:n])

	if resp.StatusCode == 200 {
		log.Printf("[LOGBOOK] eQSL: success (response: %s)", responseText)
	} else {
		log.Printf("[LOGBOOK] eQSL: failed, status: %d, response: %s", resp.StatusCode, responseText)
	}
}

// submitHRDLog はHRDLog.netへ送信します
func submitHRDLog(adif, callsign, uploadCode string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[LOGBOOK] HRDLog panic:", r)
		}
	}()

	log.Println("[LOGBOOK] HRDLog: sending...")
	log.Printf("[LOGBOOK] HRDLog: Callsign=%s, UploadCode=%s", callsign, uploadCode)
	log.Printf("[LOGBOOK] HRDLog: ADIF=%s", adif)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://robot.hrdlog.net/NewEntry.aspx", url.Values{
		"Callsign":   {callsign},
		"UploadCode": {uploadCode},
		"ADIF":       {adif},
	})
	if err != nil {
		log.Println("[LOGBOOK] HRDLog error:", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取る
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseText := string(body[:n])

	if resp.StatusCode == 200 {
		log.Printf("[LOGBOOK] HRDLog: success - response: %s", responseText)
	} else {
		log.Printf("[LOGBOOK] HRDLog: failed, status: %d, response: %s", resp.StatusCode, responseText)
	}
}

// submitClubLog はClubLogへ送信します
func submitClubLog(adif, email, password, callsign, apikey string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[LOGBOOK] ClubLog panic:", r)
		}
	}()

	log.Println("[LOGBOOK] ClubLog: sending...")

	client := &http.Client{Timeout: 10 * time.Second}

	values := url.Values{
		"email":    {email},
		"password": {password},
		"callsign": {callsign},
		"adif":     {adif},
	}

	// APIキーが設定されている場合のみ追加
	if strings.TrimSpace(apikey) != "" {
		values.Set("api", apikey)
	}

	resp, err := client.PostForm("https://secure.clublog.org/realtime.php", values)
	if err != nil {
		log.Println("[LOGBOOK] ClubLog error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("[LOGBOOK] ClubLog: success")
	} else {
		log.Println("[LOGBOOK] ClubLog: failed, status:", resp.StatusCode)
	}
}
