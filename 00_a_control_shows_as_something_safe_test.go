package khatool

import "testing"

// A control character never reaches the renderer as itself, and the form that
// stands in for it is plain ASCII -- so the cell count a caller budgets for it
// is the rune count of the form, with nothing to shape or measure.
func TestAControlShowsAsSomethingSafe(t *testing.T) {
	cases := []struct {
		r    rune
		want string
	}{
		{0x00, "^@"},       // NUL, which has no letter to caret
		{0x07, "^G"},       // BEL
		{0x09, "^I"},       // TAB
		{0x1B, "^["},       // ESC, whose letter is a bracket
		{0x1C, "1C"},       // past the caret forms, still one byte
		{0x7F, "7F"},       // DEL
		{0x9B, "9B"},       // CSI: the C1 introducer that eats the rest of a line
		{0x2028, "(2028)"}, // two bytes, so the digits need a boundary
		{0x1D173, "(01D173)"},
	}
	for _, c := range cases {
		if got := Substitute(c.r); got != c.want {
			t.Errorf("Substitute(%04X) = %q, want %q", c.r, got, c.want)
		}
	}
	for _, c := range cases {
		for _, r := range Substitute(c.r) {
			if r > 0x7E || r < 0x20 {
				t.Errorf("Substitute(%04X) = %q: %q is not plain ASCII", c.r, c.want, r)
			}
		}
	}
}

// A mark with no base it can attach to is shown on a circle that supplies one,
// which is the Unicode convention and what a shaper does anyway.
func TestAnIsolatedMarkGetsACircleToSitOn(t *testing.T) {
	const hebrewPoint = 'ִ' // HIRIQ, defective on a Latin base
	if !DefectiveMark('a', hebrewPoint) {
		t.Fatal("a Hebrew point on a Latin base is defective")
	}
	got := MarkForm(hebrewPoint)
	want := string([]rune{MarkAnchor, hebrewPoint})
	if got != want {
		t.Errorf("MarkForm = %q, want %q", got, want)
	}
	// The mark composes onto the circle rather than replacing it: both runes
	// are there, in that order, and nothing else is.
	if r := []rune(got); len(r) != 2 || r[0] != MarkAnchor || r[1] != hebrewPoint {
		t.Errorf("MarkForm = %v, want the anchor then the mark", []rune(got))
	}
}
