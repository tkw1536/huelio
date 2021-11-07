package engine

import (
	"strconv"
	"sync"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Index provides cached data contained in a hue bridge
type Index struct {
	Groups []huego.Group
	Lights []huego.Light
	Scenes []huego.Scene
}

var ErrIndexNilBridge = errors.New("NewIndex: bridge is nil")

// NewIndex creates an index of a given bridge
func NewIndex(bridge *huego.Bridge) (*Index, error) {
	if bridge == nil {
		return nil, ErrIndexNilBridge
	}

	eg := &errgroup.Group{}
	index := &Index{}

	eg.Go(func() (err error) {
		index.Groups, err = bridge.GetGroups()
		return
	})
	eg.Go(func() (err error) {
		index.Lights, err = bridge.GetLights()
		return
	})
	eg.Go(func() (err error) {
		index.Scenes, err = bridge.GetScenes()
		return
	})

	return index, eg.Wait()
}

// QueryString sends a string query to this index
func (index Index) QueryString(input string) ([]Action, []Score) {
	return index.Query(ParseQuery(input))
}

var scoringPool = &sync.Pool{
	New: func() interface{} {
		return new(ScoreQueries)
	},
}

var resultsPool = &sync.Pool{
	New: func() interface{} {
		return new(Results)
	},
}

// Query queries an index for a set of queries
// It returns a list of matching actions
func (index Index) Query(queries []Query) (actions []Action, scores []Score) {
	scoring := scoringPool.Get().(*ScoreQueries)
	defer scoringPool.Put(scoring)

	results := resultsPool.Get().(*Results)
	results.Reset(len(index.Groups) + len(index.Lights)) // todo: do we want to cache this?
	defer resultsPool.Put(results)

	for _, g := range index.Groups {
		scoring.Use(queries)

		theGroup := NewHueGroup(g)

		if !scoring.Score(func(q Query) float64 { return q.MatchGroup(theGroup) }) {
			groupPool.Put(theGroup)
			continue
		}

		// match the word "on"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 { return q.MatchOnOff(BoolOn) }); len(scores) > 0 {
			results.Add(Action{
				Group: theGroup,
				OnOff: BoolOn,
			}, scores)
		}

		// match the word "off"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 { return q.MatchOnOff(BoolOff) }); len(scores) > 0 {
			results.Add(Action{
				Group: theGroup,
				OnOff: BoolOff,
			}, scores)
		}

		// iterate over scenes in this group!
		gID := strconv.Itoa(g.ID)
		for _, s := range index.Scenes {
			if s.Group != gID {
				continue
			}

			theScene := NewHueScene(s)

			scores, _ := scoring.ScoreFinal(func(q Query) float64 { return q.MatchScene(theScene) })
			if len(scores) == 0 {
				scenePool.Put(theScene)
				continue
			}

			results.Add(Action{
				Group: theGroup,
				Scene: theScene,
			}, scores)
		}
	}

	for _, l := range index.Lights {
		scoring.Use(queries)

		theLight := NewHueLight(l)

		if !scoring.Score(func(q Query) float64 { return q.MatchLight(theLight) }) {
			lightPool.Put(theLight)
			continue
		}

		// match the word "on"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 { return q.MatchOnOff(BoolOn) }); len(scores) > 0 {
			results.Add(Action{
				Light: theLight,
				OnOff: BoolOn,
			}, scores)
		}

		// match the word "off"
		if scores, _ := scoring.ScoreFinal(func(q Query) float64 { return q.MatchOnOff(BoolOff) }); len(scores) > 0 {
			results.Add(Action{
				Light: theLight,
				OnOff: BoolOff,
			}, scores)
		}

	}

	return results.Results()
}
