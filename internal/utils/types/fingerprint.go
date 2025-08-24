package types

type Fingerprint struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	IP      string `json:"ip"`
	Device  string `json:"device"`
}
