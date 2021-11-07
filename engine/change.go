package engine

import (
	"fmt"
	"strings"
)

// Change describes a change to perform for the specified name
type Change struct {
	Scene string    // turn on a special scene
	OnOff BoolOnOff // turn the light on or off
}

// ParseChange parses a value into a string.
func ParseChange(value string) Change {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case string(BoolOn), string(BoolOff):
		return Change{OnOff: BoolOnOff(value)}

	// TODO: parse colors
	default:
		return Change{Scene: value}
	}
}

func (act Change) IsOnOff(onoff BoolOnOff) bool {
	return act.Scene == "" && (act.OnOff == BoolAny || act.OnOff == onoff)
}

func (act Change) String() string {
	if act.Scene != "" {
		return fmt.Sprintf("Scene %q", act.Scene)
	}
	if act.OnOff == BoolOn {
		return "turn on"
	}
	if act.OnOff == BoolOff {
		return "turn off"
	}
	return "<invalid>"
}

// BoolOnOff represents turning a scene on or off
type BoolOnOff string

const (
	BoolAny BoolOnOff = ""
	BoolOn  BoolOnOff = "on"
	BoolOff BoolOnOff = "off"
)
