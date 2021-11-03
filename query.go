package huelio

import (
	"fmt"
	"strings"
)

// Query desribes a query for the huelio system
type Query struct {
	// Name of the object (light or scene) to modify
	Name string

	// Action is the Action to perform
	Action Action
}

func (q Query) String() string {
	return fmt.Sprintf("%q: %s", q.Name, q.Action)
}

// anyQueries checks if any of the queries matches the provided predicate
func anyQueries(queries []Query, pred func(q Query) bool) bool {
	for _, q := range queries {
		if pred(q) {
			return true
		}
	}
	return false
}

// filterQueries finds all queries that match a given predicate
func filterQueries(queries []Query, pred func(q Query) bool) (matches []Query, any bool) {
	matches = make([]Query, 0, len(queries))
	for _, q := range queries {
		if pred(q) {
			matches = append(matches, q)
		}
	}
	return matches, len(matches) != 0
}

// ParseQuery parses a query.
// It can contain the following syntaxes:
// ["turn"] "off"|"on" ROOM|LIGHT
// ["turn"] ROOM|LIGHT "to" ACTION
// ROOM|LIGHT|ACTION
func ParseQuery(value string) (passes []Query) {
	fields := strings.Fields(strings.ToLower(value)) // TODO: parse me!
	if len(fields) == 0 {
		return nil
	}

	// compute one-based indexes of everything
	// we use 1-based indexes to avoid having to initialize non-existing values.
	fieldIndexes := make(map[string]int, len(fields)+3)
	for i, f := range fields {
		if _, ok := fieldIndexes[f]; !ok {
			fieldIndexes[f] = i + 1
		}
	}

	//lint:ignore S1005 ignore non-existence
	turnIndex, _ := fieldIndexes["turn"]
	//lint:ignore S1005 ignore non-existence
	onIndex, _ := fieldIndexes["on"]
	//lint:ignore S1005 ignore non-existence
	offIndex, _ := fieldIndexes["off"]
	//lint:ignore S1005 ignore non-existence
	toIndex, _ := fieldIndexes["to"]

	// "turn" "on" | "off" ROOM|LIGHT
	if turnIndex == 1 && (onIndex == 2 || offIndex == 2) && len(fields) >= 3 {
		passes = append(passes, Query{
			Name:   strings.Join(fields[2:], " "),
			Action: ParseAction(fields[1]),
		})
	}

	// "on" | "off" light
	if (onIndex == 1 || offIndex == 1) && len(fields) >= 2 {
		passes = append(passes, Query{
			Name:   strings.Join(fields[1:], " "),
			Action: ParseAction(fields[0]),
		})
	}

	// [turn] light "on" | "off"
	if onIndex == len(fields) || offIndex == len(fields) && len(fields) >= 2 {
		last := len(fields) - 1
		var name string
		if turnIndex == 1 {
			name = strings.Join(fields[1:last], " ")
		} else {
			name = strings.Join(fields[:last], " ")
		}
		passes = append(passes, Query{
			Name:   name,
			Action: ParseAction(fields[last]),
		})
	}

	// [turn] ROOM|LIGHT to ACTION
	if toIndex > 0 && len(fields) > toIndex {
		action := ParseAction(strings.Join(fields[toIndex:], " "))
		if turnIndex == 0 {
			passes = append(passes, Query{
				Name:   strings.Join(fields[1:toIndex-1], " "),
				Action: action,
			})
		} else {
			passes = append(passes, Query{
				Name:   strings.Join(fields[:toIndex-1], " "),
				Action: action,
			})
		}
	}

	// ROOM | LIGHT
	passes = append(passes, Query{
		Name:   strings.Join(fields, " "),
		Action: ParseAction(""),
	})

	// ACTION
	passes = append(passes, Query{
		Name:   "",
		Action: ParseAction(strings.Join(fields, " ")),
	})

	return
}

// Action describes an action to perform
type Action struct {
	// Exactly one of these must be non-zero
	Scene string // turn on a special scene

	OnOff string // turn the light on or off
}

func (act Action) IsOnOff(onoff string) bool {
	return act.Scene == "" && (act.OnOff == "" || act.OnOff == onoff)
}

func (act Action) String() string {
	if act.Scene != "" {
		return fmt.Sprintf("Activate Scene %q", act.Scene)
	}
	if act.OnOff == "on" {
		return "turn on"
	}
	if act.OnOff == "off" {
		return "turn off"
	}
	return "<invalid>"
}

// ParseAction parses a string into an action.
//
// This currently only catches special actions
func ParseAction(value string) Action {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "on", "off":
		return Action{OnOff: value}
	// todo: parse colors etc
	default:
		return Action{Scene: value}
	}
}
