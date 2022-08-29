package engine

import (
	"fmt"

	"github.com/amimof/huego"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/mazznoer/csscolorparser"
	"github.com/pkg/errors"
)

// Action represents  single action
type Action struct {
	Group *HueGroup `json:"group,omitempty"`
	Light *HueLight `json:"light,omitempty"`

	Scene *HueScene `json:"scene,omitempty"`
	OnOff BoolOnOff `json:"onoff,omitempty"`
	Color string    `json:"color,omitempty"`

	Special *HueSpecial `json:"special,omitempty"`
}

func (act Action) ColorXY() (xy []float32) {
	pc, err := csscolorparser.Parse(act.Color)
	if err != nil {
		return
	}

	color, err := colorful.Hex(pc.HexString())
	if err != nil {
		return
	}

	X, Y, Z := color.Xyz()
	xy = []float32{
		float32(X / (X + Y + Z)),
		float32(Y / (X + Y + Z)),
	}

	return
}

// BoolOnOff represents turning a scene on or off
type BoolOnOff string

const (
	BoolAny BoolOnOff = ""
	BoolOn  BoolOnOff = "on"
	BoolOff BoolOnOff = "off"
)

// HueSpecial represents a special action that can be returned by the webserver
type HueSpecial struct {
	ID   string `json:"id"`
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

var ErrInvalidAction = errors.New("action.Do: Invalid action")

func (action Action) Do(bridge *huego.Bridge) error {
	switch {
	case action.Group != nil:
		if err := action.Group.Refresh(bridge); err != nil {
			return errors.Wrap(err, "Unable to find group")
		}
		group := action.Group.Data
		xy := action.ColorXY()
		switch {
		case action.Scene != nil:
			return group.Scene(action.Scene.ID)
		case action.OnOff == "on":
			return group.SetState(huego.State{On: true})
		case action.OnOff == "off":
			return group.SetState(huego.State{On: false})
		case xy != nil:
			return group.SetState(huego.State{On: true, Xy: xy})
		}
	case action.Light != nil:
		if err := action.Light.Refresh(bridge); err != nil {
			return errors.Wrap(err, "Unable to find light")
		}
		light := action.Light.Data

		xy := action.ColorXY()
		switch {
		case xy != nil:
			return light.SetState(huego.State{On: true, Xy: xy})
		case action.OnOff == "on":
			return light.SetState(huego.State{On: true})
		case action.OnOff == "off":
			return light.SetState(huego.State{On: false})
		}
	}
	return ErrInvalidAction
}

// String stringifies the ex
func (res Action) String() string {
	var name string
	switch {
	case res.Group != nil:
		name = fmt.Sprintf("Room %q", res.Group.Data.Name)
	case res.Light != nil:
		name = fmt.Sprintf("Light %q", res.Light.Data.Name)
	default:
		name = "<invalid>"
	}

	var action string
	switch {
	case res.OnOff == "on":
		action = "turn on"
	case res.OnOff == "off":
		action = "turn off"
	case res.Color != "":
		action = "turn " + res.Color
	case res.Scene != nil:
		action = fmt.Sprintf("activate %q", res.Scene.Data.Name)
	}

	return fmt.Sprintf("%s: %s", name, action)
}
