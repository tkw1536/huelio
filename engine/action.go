package engine

import (
	"fmt"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
)

// Action represents  single action
type Action struct {
	Group *HueGroup `json:"group,omitempty"`
	Light *HueLight `json:"light,omitempty"`

	Scene *HueScene `json:"scene,omitempty"`
	OnOff BoolOnOff `json:"onoff,omitempty"`

	Special *HueSpecial `json:"special,omitempty"`
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
		switch {
		case action.Scene != nil:
			return group.Scene(action.Scene.ID)
		case action.OnOff == "on":
			return group.SetState(huego.State{On: true})
		case action.OnOff == "off":
			return group.SetState(huego.State{On: false})
		}
	case action.Light != nil:
		if err := action.Light.Refresh(bridge); err != nil {
			return errors.Wrap(err, "Unable to find light")
		}
		light := action.Light.Data
		switch action.OnOff {
		case "on":
			return light.SetState(huego.State{On: true})
		case "off":
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
	case res.Scene != nil:
		action = fmt.Sprintf("activate %q", res.Scene.Data.Name)
	}

	return fmt.Sprintf("%s: %s", name, action)
}
