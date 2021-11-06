// Package creds implements facilities for managing and generating hue bridge credentials
package creds

import "github.com/amimof/huego"

// Credentials represents credentials to a hue bridge
type Credentials struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
}

// NewBridge creates a new bridge based on credentials
func NewBridge(credentials *Credentials) (*huego.Bridge, error) {
	bridge := huego.New(credentials.Hostname, credentials.Username)
	_, err := bridge.GetCapabilities()
	if err != nil {
		return nil, err
	}
	return bridge, nil
}
