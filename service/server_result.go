package service

import (
	"encoding/json"

	"github.com/tkw1536/huelio/engine"
)

// MarshalResult is used to marshal result
type MarshalResult struct {
	Results    []engine.Action
	Scores     []engine.Score
	MatchScore []engine.MatchScore

	// WithScore indicates if the scores should be marshaled
	WithScore bool
}

func (result MarshalResult) MarshalJSON() ([]byte, error) {
	if len(result.Results) == 0 {
		return []byte("[]"), nil
	}
	// marshal without a score => easy
	if !result.WithScore {
		return json.Marshal(result.Results)
	}

	withScores := make([]struct {
		engine.Action
		Debug struct {
			Scores      engine.Score      `json:"scores"`
			MatchScores engine.MatchScore `json:"matchScores"`
		} `json:"debug"`
	}, len(result.Results))

	for i, r := range result.Results {
		withScores[i].Action = r
		withScores[i].Debug.Scores = result.Scores[i]
		withScores[i].Debug.MatchScores = result.MatchScore[i]
	}

	return json.Marshal(withScores)
}
