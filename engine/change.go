package engine

// BoolOnOff represents turning a scene on or off
type BoolOnOff string

const (
	BoolAny BoolOnOff = ""
	BoolOn  BoolOnOff = "on"
	BoolOff BoolOnOff = "off"
)
