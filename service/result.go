package service

import (
	"encoding/json"

	"github.com/tkw1536/huelio/engine"
)

// result represents a result that is sent to the client
type result struct {
	Results    []engine.Action
	Scores     []engine.Score
	MatchScore []engine.BufferScore

	WithScore bool // if true, marshal the scores and send them to the client
}

const emptyArray = "[]"

// MarshalJSON marshals these results as json
func (r result) MarshalJSON() ([]byte, error) {
	switch {
	case len(r.Results) == 0:
		return []byte(emptyArray), nil

	case !r.WithScore:
		return json.Marshal(r.Results)

	default:
		withScores := make([]struct {
			engine.Action
			Debug struct {
				Scores      engine.Score       `json:"scores"`
				MatchScores engine.BufferScore `json:"matchScores"`
			} `json:"debug"`
		}, len(r.Results))

		for i, a := range r.Results {
			withScores[i].Action = a
			withScores[i].Debug.Scores = r.Scores[i]
			withScores[i].Debug.MatchScores = r.MatchScore[i]
		}

		return json.Marshal(withScores)
	}

}
