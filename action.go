package huelio

import (
	"encoding/json"
	"fmt"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
)

// QueryAction represents the result of a query
type QueryAction struct {
	scores      [4]float64
	matchScores [][]float64

	Group *HueGroup `json:"group,omitempty"`
	Light *HueLight `json:"light,omitempty"`

	Scene *HueScene `json:"scene,omitempty"`
	OnOff BoolOnOff `json:"onoff,omitempty"`

	Special *HueSpecial `json:"special,omitempty"`
}

type HueSpecial struct {
	ID   string `json:"id"`
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

var ErrInvalidAction = errors.New("action.Do: Invalid action")

func (action QueryAction) Do(bridge *huego.Bridge) error {
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

// HueGroup represents a hue group
type HueGroup struct {
	ID   int         `json:"id"`
	Data huego.Group `json:"data"`
}

func NewHueGroup(group huego.Group) *HueGroup {
	return &HueGroup{
		ID:   group.ID,
		Data: group,
	}
}

func (group *HueGroup) UnmarshalJSON(data []byte) error {
	id := &struct {
		ID int `json:"id"`
	}{}
	err := json.Unmarshal(data, &id)
	group.ID = id.ID
	return err
}

func (group *HueGroup) Refresh(bridge *huego.Bridge) error {
	data, err := bridge.GetGroup(group.ID)
	if data != nil {
		group.Data = *data
	}
	return err
}

type HueLight struct {
	ID   int         `json:"id"`
	Data huego.Light `json:"data"`
}

func NewHueLight(light huego.Light) *HueLight {
	return &HueLight{
		ID:   light.ID,
		Data: light,
	}
}

func (light *HueLight) UnmarshalJSON(data []byte) error {
	id := &struct {
		ID int `json:"id"`
	}{}
	err := json.Unmarshal(data, &id)
	light.ID = id.ID
	return err
}

func (light *HueLight) Refresh(bridge *huego.Bridge) error {
	data, err := bridge.GetLight(light.ID)
	if data != nil {
		light.Data = *data
	}
	return err
}

type HueScene struct {
	ID   string      `json:"id"`
	Data huego.Scene `json:"data"`
}

func NewHueScene(scene huego.Scene) *HueScene {
	return &HueScene{
		ID:   scene.ID,
		Data: scene,
	}
}

func (scene *HueScene) UnmarshalJSON(data []byte) error {
	id := &struct {
		ID string `json:"id"`
	}{}
	err := json.Unmarshal(data, &id)
	scene.ID = id.ID
	return err
}

func (scene *HueScene) Refresh(bridge *huego.Bridge) error {
	data, err := bridge.GetScene(scene.ID)
	if data != nil {
		scene.Data = *data
	}
	return err
}
