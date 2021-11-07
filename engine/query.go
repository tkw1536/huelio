package engine

import (
	"fmt"
	"strings"
	"sync"
)

// Query describes an change for the huelio system.
type Query struct {
	// Name is the name of a Hue Object to fuzzy match against
	Name string
	// Change is a change to perform to the specified object
	Change Change
}

func (q Query) String() string {
	return fmt.Sprintf("%q: %s", q.Name, q.Change)
}

// ParseQuery parses a query from the string
func ParseQuery(value string) (passes []Query) {
	// split the string into fields, normalizing to lower cases
	fields := strings.Fields(strings.ToLower(value))
	if len(fields) == 0 {
		return nil
	}

	// get a map of indexes to grab from the pool
	indexes := makeQueryIndexes(fields)
	defer queryIndexPool.Put(indexes)

	// "turn" "on" | "off" ROOM|LIGHT
	if indexes["turn"] == 0 && (indexes[string(BoolOn)] == 2 || indexes[string(BoolOff)] == 2) && len(fields) >= 3 {
		passes = append(passes, Query{
			Name:   strings.Join(fields[2:], " "),
			Change: ParseChange(fields[1]),
		})
	}

	// "on" | "off" light
	if (indexes[string(BoolOn)] == 1 || indexes[string(BoolOff)] == 1) && len(fields) >= 2 {
		passes = append(passes, Query{
			Name:   strings.Join(fields[1:], " "),
			Change: ParseChange(fields[0]),
		})
	}

	// [turn] light "on" | "off"
	if indexes[string(BoolOn)] == len(fields) || indexes[string(BoolOff)] == len(fields) && len(fields) >= 2 {
		last := len(fields) - 1
		var name string
		if indexes["turn"] == 1 {
			name = strings.Join(fields[1:last], " ")
		} else {
			name = strings.Join(fields[:last], " ")
		}
		passes = append(passes, Query{
			Name:   name,
			Change: ParseChange(fields[last]),
		})
	}

	// [turn] ROOM|LIGHT to ACTION
	if indexes["to"] > 1 && len(fields) > indexes["to"]+1 {
		action := ParseChange(strings.Join(fields[indexes["to"]+1:], " "))
		if indexes["turn"] == 0 {
			passes = append(passes, Query{
				Name:   strings.Join(fields[1:indexes["to"]], " "),
				Change: action,
			})
		} else {
			passes = append(passes, Query{
				Name:   strings.Join(fields[:indexes["to"]], " "),
				Change: action,
			})
		}
	}

	for i := 0; i <= len(fields); i++ {
		passes = append(passes, Query{
			Name:   strings.Join(fields[:i], " "),
			Change: ParseChange(strings.Join(fields[i:], " ")),
		})
	}

	return
}

var parseQueryPrebuilts = map[string]struct{}{
	"turn":          {},
	string(BoolOn):  {},
	string(BoolOff): {},
	"to":            {},
}

var queryIndexPool = &sync.Pool{
	New: func() interface{} {
		return make(map[string]int, len(parseQueryPrebuilts))
	},
}

// makeQueryIndexes computes indexes of parseQueryPrebuilts in values.
//
// The returned map, if not used otherwise, should be returned to queryIndexPool by the caller.
func makeQueryIndexes(values []string) map[string]int {
	// set the required indexes to 0
	indexes := queryIndexPool.Get().(map[string]int)
	for k := range parseQueryPrebuilts {
		indexes[k] = 0
	}

	for i, v := range values {
		if _, ok := parseQueryPrebuilts[v]; !ok {
			continue
		}
		if indexes[v] == 0 {
			indexes[v] = i
		}
	}

	return indexes
}
