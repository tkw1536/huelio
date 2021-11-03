package huelio

import (
	"errors"
	"strconv"
	"sync"

	"github.com/amimof/huego"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/sync/errgroup"
)

// Engine represents a system that can answer queries
type Engine struct {
	m sync.RWMutex // protect the rest of this struct

	err error // error of initializtion

	bridge *huego.Bridge // bridge to use for executing actions

	groups []huego.Group // list of groups
	lights []huego.Light // list of lights
	scenes []huego.Scene // list of scenes
}

// NewEngine creates a new engine and initializes caches
func NewEngine(bridge *huego.Bridge) *Engine {
	engine := &Engine{err: ErrNoBridge}
	engine.m.Lock()
	go engine.useInternal(bridge)

	return engine
}

// Use tells engine to use the provided bridge.
// when bridge is nil, refreshes caches from the current bridge.
func (engine *Engine) Use(bridge *huego.Bridge) error {
	engine.m.Lock()
	return engine.useInternal(bridge)
}

var ErrNoBridge = errors.New("engine: no bridge available")

// useInternal sets bridge.
// The caller must ensure to obtain an appropriate lock on m.
func (engine *Engine) useInternal(bridge *huego.Bridge) error {
	defer engine.m.Unlock()

	if bridge != nil {
		engine.bridge = bridge
	}
	if engine.bridge == nil {
		return ErrNoBridge
	}

	eg := &errgroup.Group{}
	eg.Go(func() (err error) {
		engine.groups, err = engine.bridge.GetGroups()
		return
	})

	eg.Go(func() (err error) {
		engine.lights, err = engine.bridge.GetLights()
		return
	})

	eg.Go(func() (err error) {
		engine.scenes, err = engine.bridge.GetScenes()
		return
	})

	engine.err = eg.Wait()
	return engine.err
}

// Bridge returns the current bridge, if it has been initialized.
func (engine *Engine) Bridge() (*huego.Bridge, error) {
	engine.m.RLock()
	defer engine.m.RUnlock()

	bridge, err := engine.bridge, engine.err
	if bridge == nil && err == nil {
		return nil, ErrNoBridge
	}
	if err != nil {
		return nil, err
	}
	return bridge, nil
}

func (engine *Engine) Query(input string) (results []QueryAction, err error) {
	return engine.Run(ParseQuery(input))
}

// Run runs the provided queries
func (engine *Engine) Run(queries []Query) (results []QueryAction, err error) {
	engine.m.RLock()
	defer engine.m.RUnlock()

	if engine.err != nil {
		return nil, engine.err
	}

	for i, group := range engine.groups {
		theGroup := &ActionGroup{
			ID:   group.ID,
			Data: &engine.groups[i],
		}

		// find queries that match
		qs, ok := filterQueries(queries, func(q Query) bool { return fuzzy.MatchFold(q.Name, group.Name) })
		if !ok {
			continue
		}

		// match "on" and "off" actions
		if anyQueries(qs, func(q Query) bool { return q.Action.IsOnOff("on") }) {
			results = append(results, QueryAction{
				Group: theGroup,
				OnOff: "on",
			})
		}

		if anyQueries(qs, func(q Query) bool { return q.Action.IsOnOff("off") }) {
			results = append(results, QueryAction{
				Group: theGroup,
				OnOff: "off",
			})
		}

		// iterate over scenes in this group!
		gID := strconv.Itoa(group.ID)
		for j, s := range engine.scenes {
			if s.Group != gID {
				continue
			}

			if !anyQueries(qs, func(q Query) bool { return fuzzy.MatchFold(q.Action.Scene, s.Name) }) {
				continue
			}

			theScene := &ActionScene{
				ID:   s.ID,
				Data: &engine.scenes[j],
			}

			results = append(results, QueryAction{
				Group: theGroup,
				Scene: theScene,
			})

		}
	}

	for i, light := range engine.lights {
		// find if this light matches any of the queries
		// and if so, continue checking scenes
		qs, ok := filterQueries(queries, func(q Query) bool { return fuzzy.MatchFold(q.Name, light.Name) })
		if !ok {
			continue
		}
		// match the distinct OnOff Queries

		theLight := &ActionLight{
			ID:   light.ID,
			Data: &engine.lights[i],
		}

		if anyQueries(qs, func(q Query) bool { return q.Action.IsOnOff("on") }) {
			results = append(results, QueryAction{
				Light: theLight,
				OnOff: "on",
			})
		}

		if anyQueries(qs, func(q Query) bool { return q.Action.IsOnOff("off") }) {
			results = append(results, QueryAction{
				Light: theLight,
				OnOff: "off",
			})
		}

		// TODO: Matching scenes
	}

	return results, nil
}
