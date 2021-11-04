package huelio

import (
	"strconv"
	"sync"

	"github.com/amimof/huego"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"
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
	go engine.Use(bridge)

	return engine
}

var ErrNoBridge = errors.New("engine: No Bridge")

// Use tells engine to use the provided bridge.
// when bridge is nil, refreshes caches from the current bridge.
func (engine *Engine) Use(bridge *huego.Bridge) error {
	// if the bridge is nil, use the bridge from the engine
	if bridge == nil {
		func() {
			engine.m.RLock()
			defer engine.m.RUnlock()

			bridge = engine.bridge
		}()
	}

	// if no bridge exists anywhere, we're done.
	// no need to lock writes.
	if bridge == nil {
		return ErrNoBridge
	}

	// fetch all the lights, groups, and scenes!
	eg := &errgroup.Group{}

	var groups []huego.Group
	var lights []huego.Light
	var scenes []huego.Scene

	eg.Go(func() (err error) {
		groups, err = bridge.GetGroups()
		return
	})
	eg.Go(func() (err error) {
		lights, err = bridge.GetLights()
		return
	})
	eg.Go(func() (err error) {
		scenes, err = bridge.GetScenes()
		return
	})

	err := eg.Wait()

	// store the results
	engine.m.Lock()
	defer engine.m.Unlock()

	engine.bridge = bridge
	engine.err = err
	engine.groups = groups
	engine.lights = lights
	engine.scenes = scenes

	return err
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

	var scoring ScoreQueries
	for _, g := range engine.groups {
		scoring.Use(queries)

		if !scoring.Score(func(q Query) float64 {
			return ScoreText(q.Name, g.Name)
		}) {
			continue
		}

		theGroup := NewHueGroup(g)

		// match the word "on"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 {
			if q.Change.IsOnOff(BoolOn) {
				return 0
			} else {
				return -1
			}
		}); len(scores) > 0 {
			results = append(results, QueryAction{
				matchScores: scores,
				Group:       theGroup,
				OnOff:       BoolOn,
			})
		}

		// match the word "off"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 {
			if q.Change.IsOnOff(BoolOff) {
				return 0
			} else {
				return -1
			}
		}); len(scores) > 0 {
			results = append(results, QueryAction{
				matchScores: scores,
				Group:       theGroup,
				OnOff:       BoolOff,
			})
		}

		// iterate over scenes in this group!
		gID := strconv.Itoa(g.ID)
		for _, s := range engine.scenes {
			if s.Group != gID {
				continue
			}

			if scores, _ := scoring.ScoreFinal(func(q Query) float64 {
				return ScoreText(q.Change.Scene, s.Name)
			}); len(scores) > 0 {
				results = append(results, QueryAction{
					matchScores: scores,
					Group:       theGroup,
					Scene:       NewHueScene(s),
				})
			}
		}
	}

	for _, l := range engine.lights {
		scoring.Use(queries)

		if !scoring.Score(func(q Query) float64 {
			return ScoreText(q.Name, l.Name)
		}) {
			continue
		}

		theLight := NewHueLight(l)

		// match the word "on"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 {
			if q.Change.IsOnOff(BoolOn) {
				return 0
			} else {
				return -1
			}
		}); len(scores) > 0 {
			results = append(results, QueryAction{
				matchScores: scores,
				Light:       theLight,
				OnOff:       BoolOn,
			})
		}

		// match the word "off"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 {
			if q.Change.IsOnOff(BoolOff) {
				return 0
			} else {
				return -1
			}
		}); len(scores) > 0 {
			results = append(results, QueryAction{
				matchScores: scores,
				Light:       theLight,
				OnOff:       BoolOff,
			})
		}

	}

	ResultSlice(results).Sort()

	return results, nil
}

func ScoreText(source, target string) float64 {
	score := float64(fuzzy.RankMatchNormalizedFold(source, target))
	return score / float64(len(target))
}
