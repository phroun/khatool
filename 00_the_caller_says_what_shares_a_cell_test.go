package khatool

import (
	"reflect"
	"testing"
)

// Whether two runes share a cell is the caller's answer, not this library's,
// and the reversal has to be made against the answer the caller gave: a mark
// that rides its base travels with it when a run turns over, and one the caller
// draws in a cell of its own turns over on its own like any other cell.
//
// The same line, read under two rules, comes out in two orders -- which is the
// whole reason the rule is a parameter and not a table in here.
func TestTheCallersRuleDecidesWhatTurnsOverTogether(t *testing.T) {
	// A Hebrew letter carrying a point, then a second letter. Reading right to
	// left, the second letter comes first.
	const bet, hiriq, alef = 'ב', 'ִ', 'א'
	runes := []rune{bet, hiriq, alef}

	// Riding: the point stays immediately after the letter it composes onto, so
	// the pair moves as one and the terminal still lands it on that cell.
	if got := Order(runes, true, DefaultRides).Perm; !reflect.DeepEqual(got, []int{2, 0, 1}) {
		t.Errorf("under the default rule the line orders %v, want the point to travel with its letter", got)
	}

	// A caller that gives every rune a cell of its own -- one drawing the point
	// as a spacing substitute, say -- gets the point reversed like any other
	// cell, ahead of the letter it came after.
	alone := func(runes []rune, i int) bool { return false }
	if got := Order(runes, true, alone).Perm; !reflect.DeepEqual(got, []int{2, 1, 0}) {
		t.Errorf("under a rule where nothing rides, the line orders %v, want every cell turned over on its own", got)
	}

	// And nil is the default rule, not "nothing rides".
	if got := Order(runes, true, nil).Perm; !reflect.DeepEqual(got, []int{2, 0, 1}) {
		t.Errorf("a nil rule orders %v, want what DefaultRides gives", got)
	}
}
