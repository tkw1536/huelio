package engine

import (
	"context"
	"strconv"
	"sync"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Index holds data indexed from a hue bridge and allows it to be queried.
type Index struct {
	Groups []huego.Group
	Lights []huego.Light
	Scenes []huego.Scene
}

// ErrIndexNilBridge is returened from NewIndex when the provided bridge is not nil
var ErrIndexNilBridge = errors.New("NewIndex: bridge is nil")

// NewIndex indexes the given bridge.
// The request is cancelled if the given context is cancelled.
//
// If bridge is nil, returns [ErrIndexNilBridge].
// If an error occurs while fetching data from the bridge, returns that bridge.
func NewIndex(bridge *huego.Bridge, ctx context.Context) (index Index, err error) {
	if bridge == nil {
		return index, ErrIndexNilBridge
	}

	var eg errgroup.Group

	// fetch all of the things
	eg.Go(func() (err error) {
		index.Groups, err = bridge.GetGroupsContext(ctx)
		return
	})
	eg.Go(func() (err error) {
		index.Lights, err = bridge.GetLightsContext(ctx)
		return
	})
	eg.Go(func() (err error) {
		index.Scenes, err = bridge.GetScenesContext(ctx)
		return
	})

	// wait for the return
	err = eg.Wait()
	return
}

// QueryString passes a set of queries from the index, and passes this to index.Query.
func (index Index) QueryString(input string) ([]Action, []BufferScore, []Score) {
	return index.Query(ParseQuery(input))
}

var stringBufferPool = &sync.Pool{
	New: func() interface{} {
		return new(QueryBuffer[string])
	},
}

var resultsPool = &sync.Pool{
	New: func() interface{} {
		return new(Results)
	},
}

// Query runs a set of queries against this index.
func (index Index) Query(queries []Query) (actions []Action, matchScores []BufferScore, scores []Score) {
	scoring := stringBufferPool.Get().(*QueryBuffer[string])
	defer stringBufferPool.Put(scoring)

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
		if scores, _ := scoring.Finalize(func(q Query) float64 { return q.MatchOnOff(BoolOn) }); len(scores) > 0 {
			results.Add(Action{
				Group: theGroup,
				OnOff: BoolOn,
			}, scores)
		}

		// match the word "off"
		if scores, _ := scoring.Finalize(func(q Query) float64 { return q.MatchOnOff(BoolOff) }); len(scores) > 0 {
			results.Add(Action{
				Group: theGroup,
				OnOff: BoolOff,
			}, scores)
		}

		// match the color
		if scores, _, color := scoring.FinalizeAnnot(func(q Query) (string, float64) { return q.MatchColor() }); len(scores) > 0 {
			results.Add(Action{
				Group: theGroup,
				Color: color,
			}, scores)
		}

		// iterate over scenes in this group!
		gID := strconv.Itoa(g.ID)
		for _, s := range index.Scenes {
			if s.Group != gID {
				continue
			}

			theScene := NewHueScene(s)

			scores, _ := scoring.Finalize(func(q Query) float64 { return q.MatchScene(theScene) })
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
		if scores, _ := scoring.Finalize(func(q Query) float64 { return q.MatchOnOff(BoolOn) }); len(scores) > 0 {
			results.Add(Action{
				Light: theLight,
				OnOff: BoolOn,
			}, scores)
		}

		// match the word "off"
		if scores, _ := scoring.Finalize(func(q Query) float64 { return q.MatchOnOff(BoolOff) }); len(scores) > 0 {
			results.Add(Action{
				Light: theLight,
				OnOff: BoolOff,
			}, scores)
		}

		// match the color
		if scores, _, color := scoring.FinalizeAnnot(func(q Query) (string, float64) { return q.MatchColor() }); len(scores) > 0 {
			results.Add(Action{
				Light: theLight,
				Color: color,
			}, scores)
		}

	}

	return results.Results()
}
