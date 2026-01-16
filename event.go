package main

type ADIFEvent struct {
	Type string `json:"type"` // "adif"
	Adif string `json:"adif"`

	QRZ *struct {
		QTH      string `json:"qth"`
		Grid     string `json:"grid"`
		Operator string `json:"operator"`
	} `json:"qrz,omitempty"`

	Geo *struct {
		JCC string `json:"jcc"`
	} `json:"geo,omitempty"`
}

type RigEvent struct {
	Type string `json:"type"` // "rig"
	Rig  string `json:"rig"`  // ICOM / YAESU / KENWOOD
	Freq int64  `json:"freq,omitempty"`
	Mode string `json:"mode,omitempty"`
}

type Payload struct {
	Adif string `json:"adif"`

	QRZ *struct {
		QTH      string `json:"qth"`
		Grid     string `json:"grid"`
		Operator string `json:"operator"`
	} `json:"qrz,omitempty"`

	Geo *struct {
		JCC string `json:"jcc"`
	} `json:"geo,omitempty"`
}
