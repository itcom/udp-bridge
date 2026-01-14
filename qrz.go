package main

import (
	"encoding/xml"
	"errors"
	"log"
	"net/http"
	"net/url"
)

const qrzEndpoint = "https://xmldata.qrz.com/xml/current/"

type qrzSession struct {
	Key string `xml:"Key"`
}

type qrzResponse struct {
	Session  qrzSession `xml:"Session"`
	Callsign qrzCall    `xml:"Callsign"`
}

type qrzCall struct {
	Call  string `xml:"call"`
	Fname string `xml:"fname"`
	Name  string `xml:"name"`
	Addr2 string `xml:"addr2"`
	Grid  string `xml:"grid"`
}

var qrzKey string

// ensureQRZLogin logs in to the QRZ server with the current configuration's user and password,
// and sets the session key if it is not already set. If the login fails, an error is returned.
func ensureQRZLogin() error {
	if qrzKey != "" {
		return nil
	}
	key, err := qrzLogin()
	if err != nil {
		return err
	}
	qrzKey = key
	return nil
}

// qrzLogin logs in to the QRZ server with the current configuration's user and password, and returns the session key.
// If the login fails, an error is returned.
func qrzLogin() (string, error) {
	configLock.RLock()
	user, pass := config.QRZUser, config.QRZPass
	configLock.RUnlock()

	log.Println("[QRZ] login start:", user)

	resp, err := http.PostForm(qrzEndpoint, url.Values{
		"username": {user},
		"password": {pass},
		"agent":    {"HAMLAB-Bridge"},
	})
	if err != nil {
		log.Println("[QRZ] login http error:", err)
		return "", err
	}
	defer resp.Body.Close()

	var r qrzResponse
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Println("[QRZ] login xml error:", err)
		return "", err
	}

	if r.Session.Key == "" {
		log.Println("[QRZ] login failed: empty key")
		return "", errors.New("QRZ login failed")
	}

	log.Println("[QRZ] login success, key obtained")
	return r.Session.Key, nil
}

// qrzLookup looks up a callsign in the QRZ database with the given session key and callsign.
// If the lookup fails, an error is returned.
// If the lookup is successful, the returned qrzCall object contains the QRZ data for the callsign.
// If the QRZ data is empty, an error is returned.
func qrzLookup(key, call string) (*qrzCall, error) {
	log.Println("[QRZ] lookup:", call)

	resp, err := http.PostForm(qrzEndpoint, url.Values{
		"s":        {key},
		"callsign": {call},
	})
	if err != nil {
		log.Println("[QRZ] lookup http error:", err)
		return nil, err
	}
	defer resp.Body.Close()

	var r qrzResponse
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Println("[QRZ] lookup xml error:", err)
		return nil, err
	}

	if r.Callsign.Call == "" {
		log.Println("[QRZ] lookup no data:", call)
		return nil, errors.New("no qrz data")
	}

	log.Printf(
		"[QRZ] result call=%s addr2=%q grid=%q\n",
		r.Callsign.Call,
		r.Callsign.Addr2,
		r.Callsign.Grid,
	)

	return &r.Callsign, nil
}
