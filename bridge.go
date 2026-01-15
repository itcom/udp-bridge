package main

import (
	"encoding/json"
	"log"
	"net"

	"regexp"

	"strings"
	"time"
)

var reGrid = regexp.MustCompile(`(?i)<gridsquare:\d+>([A-Za-z0-9]+)`)
var reCall = regexp.MustCompile(`(?i)<call:\d+>([A-Za-z0-9/]+)`)
var reQSODate = regexp.MustCompile(`(?i)<qso_date:\d+>(\d{8})`)

var qrzc = newQRZCache(24 * time.Hour)

// startBridge starts the HAMLAB Bridge. It starts a WebSocket server on localhost:17800 and a UDP server on localhost:2333.
// The WebSocket server listens for incoming WSJT-X/JTDX messages and broadcasts them to connected WebSocket clients.
// The UDP server listens for incoming WSJT-X/JTDX messages and broadcasts them to connected WebSocket clients.
func startBridge() {

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2333")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, _, _ := conn.ReadFromUDP(buf)
		adif := string(buf[:n])
		log.Println("[QRZ] adif :", adif)

		call := extractCall(adif)

		if call == "" || !hasQSODate(adif) {
			continue
		}

		var qrzQTH, qrzGrid string

		configLock.RLock()
		useQRZ := config.UseQRZ
		configLock.RUnlock()

		qrzOperator := ""

		log.Println("[BRIDGE] QRZ enabled:", useQRZ, "call:", call)
		if useQRZ && call != "" {

			var qrz *qrzCall
			portable := isPortableCall(call)

			// ① キャッシュ確認
			if cached, ok := qrzc.get(call); ok {
				log.Println("[QRZ] cache hit:", call)
				qrz = cached
			} else {
				log.Println("[QRZ] cache miss, lookup:", call)
				if ensureQRZLogin() == nil {
					if r, err := qrzLookup(qrzKey, call); err == nil {
						qrz = r
						qrzc.set(call, r)
						log.Println("[QRZ] cache store:", call)
					} else {
						log.Println("[BRIDGE] QRZ lookup error:", err)
					}
				}
			}

			if qrz != nil {
				log.Println("[BRIDGE] QRZ used")

				// ★ NAME → operator（/P でも使う）
				fullName := strings.TrimSpace(qrz.Fname + " " + qrz.Name)
				if fullName != "" {
					qrzOperator = fullName
				}

				// ★ /P 等は QTH / Grid を使わない
				if !portable {
					qrzQTH = qrz.Addr2

					if usableQRZGrid(qrz.Grid) {
						qrzGrid = qrz.Grid
					}
				}
			}
		}

		grid := extractGridFromADIF(adif)
		finalGrid := betterGrid(grid, qrzGrid)

		jcc := ""
		if len(finalGrid) >= 6 {
			jcc, _ = geoLookup(finalGrid)
		}

		payload := ADIFEvent{
			Type: "adif",
			Adif: adif,
		}

		if jcc != "" {
			payload.Geo = &struct {
				JCC string `json:"jcc"`
			}{
				JCC: jcc,
			}
		}

		if qrzQTH != "" || qrzGrid != "" || qrzOperator != "" {
			payload.QRZ = &struct {
				QTH      string `json:"qth"`
				Grid     string `json:"grid"`
				Operator string `json:"operator"`
			}{
				QTH:      qrzQTH,
				Grid:     finalGrid,
				Operator: qrzOperator,
			}
		}

		b, _ := json.Marshal(payload)
		broadcast(string(b))

	}
}

// extractGridFromADIF takes an ADIF string and returns the grid extracted from it.
// The grid is extracted by searching for a substring that matches the regular expression
// `<GRIDSQUARE:\d+>([A-Za-z0-9]+)`. If a match is found, the second group of the match
// (i.e. the grid itself) is returned. If no match is found, an empty string is returned.
func extractGridFromADIF(adif string) string {
	m := reGrid.FindStringSubmatch(adif)
	if len(m) > 1 {
		return strings.ToUpper(m[1])
	}
	return ""
}

// extractCall takes an ADIF string and returns the callsign extracted from it.
// The callsign is extracted by searching for a substring that matches the regular expression
// `<CALLSIGN:\d+>([A-Za-z0-9]+)`. If a match is found, the second group of the match
// (i.e. the callsign itself) is returned and converted to uppercase. If no match is found, an empty string is returned.
func extractCall(adif string) string {
	m := reCall.FindStringSubmatch(adif)
	if len(m) > 1 {
		return strings.ToUpper(m[1])
	}
	return ""
}

// hasQSODate takes an ADIF string and returns true if the string contains a QSO date (in the format
// <YYYYMMDD>), and false otherwise.
// The QSO date is extracted by searching for a substring that matches the regular expression
// `<\d{8}>`. If a match is found, true is returned. If no match is found, false is returned.
func hasQSODate(adif string) bool {
	return reQSODate.MatchString(adif)
}

// betterGrid takes two grids and returns the better one.
// If the newer grid is longer than the original grid and starts with the original grid,
// it returns the newer grid. Otherwise, it returns the original grid.
func betterGrid(orig, newer string) string {
	if len(newer) > len(orig) && strings.HasPrefix(newer, orig) {
		return newer
	}
	return orig
}

// isPortableCall returns true if the given callsign is a portable callsign,
// and false otherwise. A callsign is considered portable if it contains a slash (/).
func isPortableCall(call string) bool {
	return strings.Contains(call, "/")
}

// usableQRZGrid returns true if the given grid is usable as a QRZ grid,
// and false otherwise. A grid is considered usable if it is 6 characters or longer.
func usableQRZGrid(grid string) bool {
	// 6桁以上のみ許可
	return len(grid) >= 6
}
