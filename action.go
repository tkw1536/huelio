package huelio

import (
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/amimof/huego"
)

type ActionGroup struct {
	ID   int          `json:"id"`
	Data *huego.Group `json:"data"`
}

func (group *ActionGroup) Refresh(bridge *huego.Bridge) (err error) {
	group.Data, err = bridge.GetGroup(group.ID)
	return
}

type ActionLight struct {
	ID   int          `json:"id"`
	Data *huego.Light `json:"data"`
}

func (light *ActionLight) Refresh(bridge *huego.Bridge) (err error) {
	light.Data, err = bridge.GetLight(light.ID)
	return
}

type ActionScene struct {
	ID   string       `json:"id"`
	Data *huego.Scene `json:"data"`
}

func (scene *ActionScene) Refresh(bridge *huego.Bridge) (err error) {
	scene.Data, err = bridge.GetScene(scene.ID)
	return
}

// QueryAction represents the result of a query
type QueryAction struct {
	Group *ActionGroup `json:"group,omitempty"`
	Light *ActionLight `json:"light,omitempty"`

	Scene *ActionScene `json:"scene,omitempty"`
	OnOff string       `json:"onoff,omitempty"`
}

var ErrInvalidAction = errors.New("action.Do: Invalid action")

func (action QueryAction) Do(bridge *huego.Bridge, logger *log.Logger) error {
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
func (res QueryAction) String() string {
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
