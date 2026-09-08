package khatool

import "testing"

// A display that would rather keep its colour than its vowels drops the marks
// that ride a right-to-left letter, and pointed Hebrew renders one codepoint
// per cell the way pre-shaped Arabic does. Folding is what makes that bearable:
// a point with a presentation form folds INTO its letter and survives as one
// glyph, so only the vowels and accents go.
func TestARidingMarkIsDroppedAndAPointIsFolded(t *testing.T) {
	const (
		vowel  = "ְ" // sheva: no presentation form, so nothing to fold into
		dagesh = "ּ" // a point: folds into its letter
	)
	// drawn renders the answer: the runes actually drawn, marks that were
	// dropped left out.
	drawn := func(s string, folding bool) string {
		out := make([]rune, 0, len(s))
		runes := []rune(s)
		for i, r := range FoldRidingMarks(runes, folding, nil) {
			_ = i
			if r != MarkDropped {
				out = append(out, r)
			}
		}
		return string(out)
	}

	for _, c := range []struct {
		name, text string
		folding    bool
		want       string
	}{
		// The vowel has nothing to fold into, so it goes either way.
		{"a vowel goes", "ש" + vowel, false, "ש"},
		{"and still goes under folding", "ש" + vowel, true, "ש"},

		// The point folds into its letter and survives as one glyph.
		{"a point folds", "ב" + dagesh, true, "בּ"},
		// With nothing to fold into, it goes with the rest.
		{"and goes when nothing folds", "ב" + dagesh, false, "ב"},

		// Both at once: the point folds, the vowel goes.
		{"a point folds and a vowel goes", "ב" + dagesh + vowel, true, "בּ"},

		// A mark on a left-to-right base is not what the reordering miscounts.
		{"an ltr base keeps its mark", "e" + "\u0301", true, "e" + "\u0301"},

		// And plain text is untouched.
		{"nothing to do", "abc שלום", true, "abc שלום"},
	} {
		if got := drawn(c.text, c.folding); got != c.want {
			t.Errorf("%s: %q folding=%v -> %q, want %q",
				c.name, c.text, c.folding, got, c.want)
		}
	}
}

// The answer is one rune per input rune, so a caller that has already worked
// out where each of them goes keeps its own indexing -- the position stays,
// only what is drawn there changes.
func TestTheAnswerKeepsTheCallersIndexing(t *testing.T) {
	runes := []rune("ש" + "ְ" + "abc")
	got := FoldRidingMarks(runes, true, nil)
	if len(got) != len(runes) {
		t.Fatalf("got %d answers for %d runes", len(got), len(runes))
	}
	if got[1] != MarkDropped {
		t.Errorf("the mark at 1 was not marked as dropped: %q", got[1])
	}
	for i := 2; i < len(runes); i++ {
		if got[i] != runes[i] {
			t.Errorf("rune %d changed from %q to %q", i, runes[i], got[i])
		}
	}
}
