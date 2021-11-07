package engine

import (
	"strconv"

	"github.com/amimof/huego"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Index is a data structure used to query the index
type Index struct {
	Groups []huego.Group
	Lights []huego.Light
	Scenes []huego.Scene
}

var ErrIndexNilBridge = errors.New("NewIndex: bridge is nil")

// NewIndex indexes the provided bridge
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
func (index Index) QueryString(input string) []Action {
	return index.Query(ParseQuery(input))
}

// Query sends a list of queries to this index
func (index Index) Query(queries []Query) (results []Action) {
	var scoring ScoreQueries
	for _, g := range index.Groups {
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
			results = append(results, Action{
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
			results = append(results, Action{
				matchScores: scores,
				Group:       theGroup,
				OnOff:       BoolOff,
			})
		}

		// iterate over scenes in this group!
		gID := strconv.Itoa(g.ID)
		for _, s := range index.Scenes {
			if s.Group != gID {
				continue
			}

			if scores, _ := scoring.ScoreFinal(func(q Query) float64 {
				return ScoreText(q.Change.Scene, s.Name)
			}); len(scores) > 0 {
				results = append(results, Action{
					matchScores: scores,
					Group:       theGroup,
					Scene:       NewHueScene(s),
				})
			}
		}
	}

	for _, l := range index.Lights {
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
			results = append(results, Action{
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
			results = append(results, Action{
				matchScores: scores,
				Light:       theLight,
				OnOff:       BoolOff,
			})
		}

	}

	ResultSlice(results).Sort()

	return results
}

func ScoreText(source, target string) float64 {
	score := float64(fuzzy.RankMatchNormalizedFold(source, target))
	return score / float64(len(target))
}
