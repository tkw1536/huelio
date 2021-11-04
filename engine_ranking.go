package huelio

import (
	"sort"
	"strconv"
)

type ResultSlice []QueryAction

func (r ResultSlice) Len() int           { return len(r) }
func (r ResultSlice) Less(i, j int) bool { return r[i].LesserPriority(r[j]) }
func (r ResultSlice) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ResultSlice) Sort()              { sort.Sort(r) }

// LesserPriority checjk
func (action QueryAction) LesserPriority(other QueryAction) bool {
	scoresA := other.Scores()
	scoresB := action.Scores()

	var b float64
	for i, a := range scoresA {
		b = scoresB[i]
		if a < b {
			return true
		}
	}
	return false
}

func (action QueryAction) Scores() (scores [4]float64) {
	scores[0] = action.MatchScore()
	scores[1] = action.KindScore()
	scores[2] = action.ItemIndexScore()
	scores[3] = action.ActionIndexScore()
	return
}

func (action QueryAction) MatchScore() (score float64) {
	if len(action.Score) == 0 {
		return 0
	}
	score0 := action.Score[0][0]
	score1 := action.Score[0][1]
	for _, s := range action.Score {
		if s[0] > score0 {
			score0 = s[0]
		}
		if s[1] > score1 {
			score1 = s[1]
		}
	}
	return -(score0 + score1)
}

func (action QueryAction) KindScore() float64 {
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

func (action QueryAction) ItemIndexScore() float64 {
	if action.Group != nil {
		return -float64(action.Group.ID)
	}
	if action.Light != nil {
		return -float64(action.Light.ID)
	}
	return 0
}

func (action QueryAction) ActionIndexScore() float64 {
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

type ScoreQueries struct {
	Queries []Query
	Scores  [][]float64
}

// Use prepares ScoreQueries to use the specefied queries
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

func (sq ScoreQueries) ScoreFinal(scoring func(q Query) float64) (scores [][]float64, queries []Query) {
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
