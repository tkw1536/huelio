package engine

import (
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/mazznoer/csscolorparser"
)

// Results represents a sortable set of results
type Results struct {
	actions      []Action
	actionScores []BufferScore

	scores []Score
}

// Reset resets this score to contain empty slices with the given cap.
func (r *Results) Reset(cap int) {
	r.actions = make([]Action, 0, cap)
	r.scores = make([]Score, 0, cap)
	r.actionScores = make([]BufferScore, 0, cap)
}

// Add adds a new action with the provided score to this result
func (r *Results) Add(action Action, actionScore BufferScore) {
	r.actions = append(r.actions, action)
	r.scores = append(r.scores, action.Score(actionScore))
	r.actionScores = append(r.actionScores, actionScore)
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

// MatchColor scores the action in this scene against a color action.
func (query Query) MatchColor() (color string, score float64) {
	c, err := csscolorparser.Parse(query.Action)
	if err != nil || c.A != 1 {
		return "", -1.0
	}
	return c.HexString(), 1.0
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

// Score represents the final score of a single action
//
// The score consists of four components; see the [Score] method of an action details.
// Scores are ordered lexiographically from index 0 to index 3; see the Less method for details.
type Score [4]float64

// Less compares this score to another score using lexiographic ordering.
// An ction
func (s Score) Less(other Score) bool {
	var b float64
	for i, a := range s {
		b = other[i]
		switch {
		case a < b:
			return true
		case a > b:
			return false
		}
	}
	return false
}

// Score returns the score of this action with respect to the given buffer score.
// It returns the four components of a score, which are:
//
// 0. the final buffer score
// 1. a score based on the kind of action this is
// 2. a score based on the original sort of this item
// 3. a score based on the parameters of the action
func (action Action) Score(buffer BufferScore) Score {
	return [4]float64{
		buffer.Final(),
		action.kindScore(),
		action.itemIndexScore(),
		action.actionIndexScore(),
	}
}

// BufferScore represents the (mutable) score of a single action during result computation.
//
// It is represented as an slice of size |scores|*|queries|; meaning
//
//	score[query][sub]
//
// returns the score of the given query, in the given sub score.
//
// Each query being score must have the same amount of objects.
type BufferScore [][]float64

// Final turns this BufferScore into a single float64 score.
//
// The final score is the sum of each maximal score.
// To allow for easier sorting, the total is negated before being returned.
//
// If this BufferScore is invalid, it is negated.
func (b BufferScore) Final() (total float64) {
	// no queries => nothing to be scored
	if len(b) == 0 {
		return 0
	}

	// determine the number of scores
	// if there aren't any => bail out
	scores := len(b[0])
	if scores == 0 {
		return 0
	}

	for i := 0; i < scores; i++ {
		max := b[0][i] // the max for the current element
		for _, q := range b[1:] {
			// quick bounds check; on the first iteration only
			// if the bounds are incorrect, return 0
			if i == 0 && len(q) != scores {
				return 0
			}
			if q[i] > max {
				max = q[i]
			}
		}
		total -= max // add to the total
	}

	return
}

// kind scores for the given action
// they are sorted increasing in the frontend
const (
	GroupOnOffScore float64 = iota
	GroupColorScore
	GroupSceneScore
	LightOnOffScore
	LightColorScore
	SpecialScore
)

// kindScore returns a score based on the kind of action this is
func (action Action) kindScore() float64 {
	isLight := action.Light != nil
	isGroup := action.Group != nil

	isScene := action.Scene != nil
	isColor := action.Color != ""
	isOnOff := action.OnOff != BoolAny

	switch {
	case isGroup && isOnOff:
		return GroupOnOffScore
	case isGroup && isColor:
		return GroupColorScore
	case isGroup && isScene:
		return GroupSceneScore
	case isLight && isOnOff:
		return LightOnOffScore
	case isLight && isColor:
		return LightColorScore
	default:
		return SpecialScore
	}
}

// itemIndexScore returns the index of this score within the index of all possible actions in the hulio api.
func (action Action) itemIndexScore() float64 {
	if action.Group != nil {
		return -float64(action.Group.ID)
	}
	if action.Light != nil {
		return -float64(action.Light.ID)
	}
	return 0
}

// actionIndexScore returns an index of the parameters to the action within the data returned.
func (action Action) actionIndexScore() float64 {
	if action.Scene != nil {
		i, _ := strconv.Atoi(action.Scene.ID)
		return float64(i)
	}
	switch action.OnOff {
	case BoolOn:
		return 1
	case BoolOff:
		return 2
	}

	return 0
}

//
// FOR SORTING
//

// Results returns a li t
func (r *Results) Results() ([]Action, []BufferScore, []Score) {
	sort.Sort(r)
	return r.actions, r.actionScores, r.scores
}

func (r *Results) Swap(i, j int) {
	r.actions[i], r.actions[j] = r.actions[j], r.actions[i]
	r.scores[i], r.scores[j] = r.scores[j], r.scores[i]
	r.actionScores[i], r.actionScores[j] = r.actionScores[j], r.actionScores[i]
}

func (r *Results) Len() int {
	return len(r.actions)
}

func (r *Results) Less(i, j int) bool {
	return r.scores[i].Less(r.scores[j])
}

// QueryBuffer holds a set of queries along with their associated BufferScore.
//
// The Annot type indicates the type of annotations.
type QueryBuffer[Annot any] struct {
	Queries []Query
	Scores  BufferScore
}

// Use resets this QueryBuffer to use the given set of queries.
// The scores are reset to be empty.
func (sq *QueryBuffer[A]) Use(Queries []Query) {
	sq.Queries = make([]Query, len(Queries))
	copy(sq.Queries, Queries)
	sq.Scores = make([][]float64, len(Queries))
}

// Score applies a scoring function to the current set of queries;
// modifying the buffer in place.
//
// When score < 0, a score is considered as non-matching and removed.
//
// The returned boolean indicates if the QueryBuffer has at least one query remaining.
func (qb *QueryBuffer[Annot]) Score(scoring func(q Query) float64) bool {
	queries := qb.Queries[:0]
	scores := qb.Scores[:0] // current scores

	// iterate over the set of queries
	for i, q := range qb.Queries {

		// score the current query; if it does not match, remove the score.
		score := scoring(q)
		if score < 0 {
			continue
		}

		// add the score to the list of scores for the query
		qScores := append(qb.Scores[i], score)
		scores = append(scores, qScores)

		// and add the query back
		queries = append(queries, q)
	}

	// clear up the unusued score memory
	// (so that it can be recovered by the GC)
	for i := len(scores); i < len(qb.Scores); i++ {
		qb.Scores[i] = nil
	}

	// update the queries and scores objects
	qb.Queries = queries
	qb.Scores = scores

	return len(qb.Queries) > 0
}

// Finalize finalizes this QueryBuffer by applying a final score to each query.
// See the Score method on how scoring is handled.
func (sq *QueryBuffer[Annot]) Finalize(scoring func(q Query) float64) (scores BufferScore, queries []Query) {
	var zero Annot
	scores, queries, _ = sq.FinalizeAnnot(func(q Query) (Annot, float64) {
		return zero, scoring(q)
	})
	return
}

// FinalizeAnnot finalizes this QueryBuffer by scoring (like in the Finalize function).
// It additionally returns the first annotation of a query with non-negative score.
func (sq *QueryBuffer[Annot]) FinalizeAnnot(scoring func(q Query) (Annot, float64)) (scores BufferScore, queries []Query, annot Annot) {
	queries = make([]Query, 0, len(sq.Queries))
	scores = make(BufferScore, 0, len(sq.Queries))

	var hadAnnot bool
	for i, q := range sq.Queries {

		// score the current query; discard it if the score is too low
		lAnnot, score := scoring(q)
		if score < 0 {
			continue
		}

		// if we did not yet have an annotation, store it!
		if !hadAnnot {
			hadAnnot = true
			annot = lAnnot
		}

		// append the new score to a copy of the previous scores
		prevScores := sq.Scores[i]
		newScores := make([]float64, len(prevScores), len(prevScores)+1)
		copy(newScores, prevScores)
		newScores = append(newScores, score)

		// use the scores and query
		scores = append(scores, newScores)
		queries = append(queries, q)
	}

	return
}
