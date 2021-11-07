package engine

import (
	"encoding/json"
	"sync"

	"github.com/amimof/huego"
)

// HueGroup represents a hue group
type HueGroup struct {
	ID   int         `json:"id"`
	Data huego.Group `json:"data"`
}

// NewHueGroup creates a new hue group.
//
// Any internal call to this function should take care to place any unused instance back into groupPool.
func NewHueGroup(group huego.Group) *HueGroup {
	g := groupPool.Get().(*HueGroup)
	g.ID = group.ID
	g.Data = group
	return g
}

var groupPool = &sync.Pool{
	New: func() interface{} {
		return new(HueGroup)
	},
}

// UnmarshalJSON implements json.Unmarshaler.
//
// For safety reasons this only unmarshals the ID from the data source.
// Any call should be followed by a call to Refresh.
func (group *HueGroup) UnmarshalJSON(data []byte) error {
	id := &struct {
		ID int `json:"id"`
	}{}
	err := json.Unmarshal(data, &id)
	group.ID = id.ID
	return err
}

// Refresh reloads data about this group from a bridge
func (group *HueGroup) Refresh(bridge *huego.Bridge) error {
	data, err := bridge.GetGroup(group.ID)
	if data != nil {
		group.Data = *data
	}
	return err
}

// HueLight represents a hue light
type HueLight struct {
	ID   int         `json:"id"`
	Data huego.Light `json:"data"`
}

// NewHueLight creates a new hue light.
//
// Any internal call to this function should take care to place any unused instance back into lightPool.
func NewHueLight(light huego.Light) *HueLight {
	l := lightPool.Get().(*HueLight)
	l.ID = light.ID
	l.Data = light
	return l
}

var lightPool = &sync.Pool{
	New: func() interface{} {
		return new(HueLight)
	},
}

// UnmarshalJSON implements json.Unmarshaler
//
// For safety reasons this only unmarshals the ID from the data source.
// Any call should be followed by a call to Refresh.
func (light *HueLight) UnmarshalJSON(data []byte) error {
	id := &struct {
		ID int `json:"id"`
	}{}
	err := json.Unmarshal(data, &id)
	light.ID = id.ID
	return err
}

// Refresh reloads data about this light from a bridge
func (light *HueLight) Refresh(bridge *huego.Bridge) error {
	data, err := bridge.GetLight(light.ID)
	if data != nil {
		light.Data = *data
	}
	return err
}

// HueScene represents a hue scene
type HueScene struct {
	ID   string      `json:"id"`
	Data huego.Scene `json:"data"`
}

// NewHueScene creates a new hue scene.
//
// Any internal call to this function should take care to place any unused instance back into scenePool.
func NewHueScene(scene huego.Scene) *HueScene {
	s := scenePool.Get().(*HueScene)
	s.ID = scene.ID
	s.Data = scene
	return s
}

var scenePool = &sync.Pool{
	New: func() interface{} {
		return new(HueScene)
	},
}

// UnmarshalJSON implements json.Unmarshaler.
//
// For safety reasons this only unmarshals the ID from the data source.
// Any call should be followed by a call to Refresh.
func (scene *HueScene) UnmarshalJSON(data []byte) error {
	id := &struct {
		ID string `json:"id"`
	}{}
	err := json.Unmarshal(data, &id)
	scene.ID = id.ID
	return err
}

// Refresh reloads data about this scene from a bridge
func (scene *HueScene) Refresh(bridge *huego.Bridge) error {
	data, err := bridge.GetScene(scene.ID)
	if data != nil {
		scene.Data = *data
	}
	return err
}
