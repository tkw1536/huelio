package engine

import (
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

// Ranking represents a set of results
type Results struct {
	actions     []Action
	scores      []Score
	matchScores []MatchScore
}

// Reset resets this ranking slice for the provided cap
func (r *Results) Reset(cap int) {
	r.actions = make([]Action, 0, cap)
	r.scores = make([]Score, 0, cap)
	r.matchScores = make([]MatchScore, 0, cap)
}

// Add adds a new result to this result set
func (r *Results) Add(action Action, matchScore MatchScore) {
	r.actions = append(r.actions, action)
	r.scores = append(r.scores, action.score(matchScore))
	r.matchScores = append(r.matchScores, matchScore)
}

//
// INDIVIDUAL SCORE
//

// MatchGroup scores the object stored in this room against a group.
func (query Query) MatchGroup(room *HueGroup) float64 {
	return scoreText(query.Name, room.Data.Name)
}

// MatchLight scores the object stored in this room against a light.
func (query Query) MatchLight(light *HueLight) float64 {
	return scoreText(query.Name, light.Data.Name)
}

// MatchLights scores the object stored in this room against a scene action.
func (query Query) MatchScene(scene *HueScene) float64 {
	return scoreText(query.Action, scene.Data.Name)
}

// MatchOnOff scores the action stored in this scene against an on/off action.
func (query Query) MatchOnOff(onoff BoolOnOff) float64 {
	return scoreText(query.Action, string(onoff))
}

// scoreText is the main scoring function.
// It scores a source text against a target match.
//
// If a match exists, the value will be between 0 and 1.
// The lower the score, the closer the match.
//
// If the source does not overlap with the target, the value returned will be -1.
func scoreText(source, target string) float64 {
	if len(target) == 0 {
		return 0
	}
	// RankMatchNormalizedFold computes the edit distance between source and target.
	// it will be -1 if there is no match.
	score := float64(fuzzy.RankMatchNormalizedFold(source, target))
	if score == -1 {
		return -1
	}
	return score / float64(len(target))
}

//
// COMPUTING OVERALL SCORES
//

// Score represents the score of a single result
type Score [4]float64

// MatchScore represents the score of a single action
type MatchScore [][]float64

// AsFloat64 returns the score of this action as a single float64
func (m MatchScore) AsFloat64() (total float64) {
	if len(m) == 0 {
		return 0 // should never happen
	}

	subScoreCount := len(m[0])

	maxes := make([]float64, subScoreCount)
	copy(maxes, m[0])

	for i, v := range m[0] {
		maxes[i] = v
	}

	for _, score := range m[1:] {
		if len(score) != subScoreCount {
			return 0 // mismatching length, shouldn't happen
		}
		for i, v := range score {
			if v > maxes[i] {
				maxes[i] = v
			}
		}
	}

	// sum the up, and invert them for proper summing
	for _, v := range maxes {
		total -= v
	}
	return
}

// score returns the score of a single action with the provided match
func (action Action) score(matchScore MatchScore) Score {
	return [4]float64{
		matchScore.AsFloat64(),
		action.kindScore(),
		action.itemIndexScore(),
		action.actionIndexScore(),
	}
}

func (action Action) kindScore() float64 {
	isLight := action.Light != nil
	isGroup := action.Group != nil

	isScene := action.Scene != nil
	isOnOff := action.OnOff != BoolAny

	switch {
	case isGroup && isOnOff:
		return 3
	case isGroup && isScene:
		return 2
	case isLight && isOnOff:
		return 1
	default:
		return 0
	}
}

func (action Action) itemIndexScore() float64 {
	if action.Group != nil {
		return -float64(action.Group.ID)
	}
	if action.Light != nil {
		return -float64(action.Light.ID)
	}
	return 0
}

func (action Action) actionIndexScore() float64 {
	if action.Scene != nil {
		i, _ := strconv.Atoi(action.Scene.ID)
		return float64(i)
	}
	switch action.OnOff {
	case BoolOn:
		return 2
	case BoolOff:
		return 1
	}

	return 0
}

// lessThan checks if s shoudld be ordered before other!
func (s Score) lessThan(other Score) bool {
	var b float64
	for i, a := range s {
		b = other[i]
		if a < b {
			return true
		}
	}
	return false
}

//
// FOR SORTING
//

// Actions returns the sorted version of results
func (r *Results) Results() ([]Action, []MatchScore, []Score) {
	sort.Sort(r)
	return r.actions, r.matchScores, r.scores
}

func (r *Results) Swap(i, j int) {
	r.actions[i], r.actions[j] = r.actions[j], r.actions[i]
	r.scores[i], r.scores[j] = r.scores[j], r.scores[i]
	r.matchScores[i], r.matchScores[j] = r.matchScores[j], r.matchScores[i]
}

func (r *Results) Len() int {
	return len(r.actions)
}

func (r *Results) Less(i, j int) bool {
	return r.scores[i].lessThan(r.scores[j])
}

// ScoreQueries represents a buffer of queries to be scored and filtered
type ScoreQueries struct {
	Queries []Query
	Scores  MatchScore
}

func (sq *ScoreQueries) Use(Queries []Query) {
	sq.Queries = make([]Query, len(Queries))
	copy(sq.Queries, Queries)
	sq.Scores = make([][]float64, len(Queries))
}

// Score applies scoring to the current list of queries.
// It modifies the scoring in place.
//
// When score < 0, considers the scores as non-matching
func (sq *ScoreQueries) Score(scoring func(q Query) float64) bool {
	queries := sq.Queries[:0]
	scores := sq.Scores[:0]

	for i, q := range sq.Queries {
		score := scoring(q)
		if score < 0 {
			continue
		}
		queries = append(queries, q)
		prevScores := sq.Scores[i]
		scores = append(scores, append(prevScores, score))
	}

	sq.Queries = queries
	sq.Scores = scores

	return len(sq.Queries) > 0
}

func (sq ScoreQueries) ScoreFinal(scoring func(q Query) float64) (scores MatchScore, queries []Query) {
	queries = make([]Query, 0, len(sq.Queries))
	scores = make([][]float64, 0, len(sq.Queries))

	for i, q := range sq.Queries {
		score := scoring(q)
		if score < 0 {
			continue
		}
		queries = append(queries, q)

		prevScores := sq.Scores[i]
		newScores := make([]float64, len(prevScores), len(prevScores)+1)
		copy(newScores, prevScores)
		newScores = append(newScores, score)

		scores = append(scores, newScores)
	}

	return
}
