package khatool

import "testing"

// A vowel has no presentation form to fold into, so it survives folding; a
// point has one and does not.
const (
	testVowel = "ְ" // sheva
	testPoint = "ּ" // dagesh
)

// A terminal that reorders what it is sent misplaces a background fill from one
// point onwards: the first right-to-left run still carrying a mark, and
// everything after it. What comes before is placed correctly and keeps its
// fill, so a row of chrome and English with one pointed word in it gives up
// only from that word on.
func TestTheFillIsLostFromThePointedRunOnwards(t *testing.T) {
	// marked renders the answer as a picture: '#' where the fill cannot be
	// placed, '.' where it can, one character per rune.
	marked := func(s string, folding bool) string {
		out := make([]rune, 0, len(s))
		for _, give := range UnplaceableFillAfterFold([]rune(s), folding, nil, nil) {
			if give {
				out = append(out, '#')
			} else {
				out = append(out, '.')
			}
		}
		return string(out)
	}

	for _, c := range []struct {
		name    string
		text    string
		folding bool
		want    string
	}{
		// Nothing right-to-left: nothing is reordered, so nothing is in doubt.
		{"english alone", "hello", false, "....."},

		// A right-to-left run with no marks is reordered but counted correctly.
		{"hebrew, no marks", "abc שלום xyz", false, "............"},

		// One vowel: the English BEFORE it keeps the fill it would have been
		// given correctly, and everything from the run on is carried off.
		{"one pointed word", "abc ש" + testVowel + "לום xyz", false,
			"....#########"},

		// The mark is on the run's LAST letter, where a run that stopped at its
		// last strong rune would have left it out.
		{"mark on the last letter", "abc שלום" + testVowel + " xyz", false,
			"....#########"},

		// Two right-to-left words with only a space between them are one run to
		// such a terminal, so a vowel in either gives up both.
		{"two words, one vowel", "שלום ר" + testVowel + "ב", false,
			"########"},

		// A vowel in the first of two runs takes the second with it: the
		// damage runs to the end of the line, not to the end of the run.
		{"two runs, one vowel", "ש" + testVowel + "לום x רב", false,
			"##########"},

		// A point that folds into its base leaves the count, so with folding on
		// the run comes out even and keeps the ordinary fill.
		{"folding point, folding on", "abc ש" + testPoint + "לום", true,
			"........."},

		// And with folding off it does not, being still free-standing.
		{"folding point, folding off", "abc ש" + testPoint + "לום", false,
			"....#####"},
	} {
		if got := marked(c.text, c.folding); got != c.want {
			t.Errorf("%s: %q\n got %s\nwant %s", c.name, c.text, got, c.want)
		}
	}
}

// Nothing gives up its fill on a line that has nothing to give up. The line can
// say yes where every position says no -- a mark with no right-to-left run to
// sit in opens none -- but never the other way round.
func TestNothingGivesUpWhatTheLineWouldKeep(t *testing.T) {
	for _, text := range []string{
		"hello",
		"abc שלום xyz",
		"abc ש" + testVowel + "לום xyz",
		testVowel,
		"שלום ר" + testVowel + "ב",
		"abc ש" + testPoint + "לום",
	} {
		for _, folding := range []bool{false, true} {
			runes := []rune(text)
			line := HasZeroWidthAfterFold(runes, folding, nil)
			for i, give := range UnplaceableFillAfterFold(runes, folding, nil, nil) {
				if give && !line {
					t.Errorf("%q folding=%v: rune %d gave up its fill on a line "+
						"with nothing to give up", text, folding, i)
				}
			}
		}
	}
}

// "After" means after in the STREAM the cells go out in, not after in the rune
// array. On a line whose base direction is right-to-left those are opposite
// ends: the pointed run is drawn at the LEFT, and what follows it on the screen
// is what came BEFORE it in the text. Marking the array's tail there gives up
// the wrong half of the line.
func TestAfterMeansAfterOnTheScreen(t *testing.T) {
	// Four runes: a pointed Hebrew letter, then three Latin ones. Under a
	// right-to-left base the Hebrew is drawn leftmost and the Latin follows it
	// across the screen, so the whole row gives up its fill.
	runes := []rune("ש" + testVowel + "abc")
	rtl := []int{0, 1, 2, 3, 4} // ש vowel a b c, left to right on screen

	got := UnplaceableFillAfterFold(runes, false, nil, rtl)
	for i, give := range got {
		if !give {
			t.Errorf("rune %d kept its fill, but it is drawn after the pointed run", i)
		}
	}

	// The same runes drawn the other way round -- the Latin first, the Hebrew
	// last -- give up only from the Hebrew on.
	ltr := []int{2, 3, 4, 0, 1} // a b c ש vowel
	got = UnplaceableFillAfterFold(runes, false, nil, ltr)
	for _, i := range []int{2, 3, 4} {
		if got[i] {
			t.Errorf("rune %d gave up its fill, but it is drawn before the pointed run", i)
		}
	}
	for _, i := range []int{0, 1} {
		if !got[i] {
			t.Errorf("rune %d kept its fill, but it is the pointed run itself", i)
		}
	}
}
