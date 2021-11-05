package huelio

import _ "embed"

//go:generate gogenlicense -p huelio -n LegalNotices -d legal_notices.go github.com/tkw1536/huelio

//go:embed LICENSE
var License string

// LegalText returns legal text to be included in human-readable output using huelio.
func LegalText() string {
	return `
================================================================================
Huelio - Control Hue lights fast
================================================================================
` + License + "\n" + LegalNotices
}
