package engine

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/tkw1536/huelio/logging"
)

var queryLogger zerolog.Logger

func init() {
	logging.ComponentLogger("engine.Query", &queryLogger)
}

// Query describes a query for the huelio system.
type Query struct {
	// Name is the name of an action to pattern match against
	Name string
	// Action is the name of an action (scene or on/off) to pattern match against
	Action string
}

func (q Query) String() string {
	return fmt.Sprintf("name={%s} change={%s}", q.Name, q.Action)
}

// ParseQuery generates a set of queries from an input string
func ParseQuery(value string) (passes []Query) {
	// split into whitespace-delimited fields
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}

	// generate a set of "passes" of the fields.
	//
	// each reading pass consists of a contigous query.Name and query.Action part.
	// the action or the name may come first, but both must be used.
	// furthermore, each part may be empty.
	//
	// this means that there are 2*(len(fields) + 1) passes.

	passes = make([]Query, 2*(len(fields)+1))

	var first, second string
	for i := 0; i <= len(fields); i++ {
		first = strings.Join(fields[:i], " ")
		second = strings.Join(fields[i:], " ")

		passes[2*i].Name = first
		passes[2*i].Action = second

		passes[2*i+1].Name = second
		passes[2*i+1].Action = first
	}

	return
}
