package khatool

import "testing"

// A terminal that runs its own bidi counts codepoints where a grid counts
// cells, so a background fill over a line holding combining marks lands on the
// wrong cells. This is how a caller asks whether its line is one of those.
//
// Folding is what makes it worth asking per line: a point that folds into its
// base no longer inflates the count, so a line of pointed consonants comes out
// even and keeps the ordinary fill.
func TestAFoldedLineIsEven(t *testing.T) {
	for _, c := range []struct {
		name             string
		text             string
		plain, afterFold bool
	}{
		// Nothing zero-width either way.
		{"english", "hello", false, false},
		{"bare hebrew", "שלום", false, false},
		// A POINT that has a presentation form: it folds into its base, so a
		// folding caller's line comes out even.
		{"pointed consonant", "שׁ", true, false}, // shin + shin dot
		{"dagesh", "בּ", true, false},            // bet + dagesh
		// A VOWEL has no form to fold into, so it survives and the line is not.
		{"vowel", "לִ", true, true}, // lamed + hiriq
		// A mark with no base at all counts wherever it is.
		{"leading mark", "ִל", true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			runes := []rune(c.text)
			if got := HasZeroWidthAfterFold(runes, false, nil); got != c.plain {
				t.Errorf("unfolded: %v, want %v", got, c.plain)
			}
			if got := HasZeroWidthAfterFold(runes, true, nil); got != c.afterFold {
				t.Errorf("folded: %v, want %v", got, c.afterFold)
			}
		})
	}
}

// The width model is the caller's, for the reason the cluster rule is: two
// answers about which runes take a cell means two disagreeing pictures of one
// line.
func TestTheCallersOwnWidthModelDecides(t *testing.T) {
	// A zero-width joiner is not a MARK, so the plain reading passes it.
	zwj := []rune("ב‍ב")
	if HasZeroWidthAfterFold(zwj, false, nil) {
		t.Error("the plain reading counted a joiner as a mark")
	}
	// A caller whose own width model gives it no cell says so, and is believed.
	mine := func(r rune) bool { return r == '‍' }
	if !HasZeroWidthAfterFold(zwj, false, mine) {
		t.Error("the caller's own width model was not consulted")
	}
	if !HasZeroWidthAfterFold(zwj, true, mine) {
		t.Error("folding lost the caller's own answer")
	}
}
