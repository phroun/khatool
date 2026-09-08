package khatool

import "testing"

// A vowel has no presentation form to fold into, so it survives folding; a
// point has one and does not.
const (
	testVowel = "ְ" // sheva
	testPoint = "ּ" // dagesh
)

// A terminal that reorders what it is sent misplaces a background fill one RUN
// at a time -- what it takes as a unit is what it counts wrongly. So a row of
// chrome and English with one pointed word in it gives up the fill on that
// word and keeps it everywhere else.
func TestOnlyTheRunCarryingMarksGivesUpItsFill(t *testing.T) {
	// marked renders the answer as a picture: '#' where the fill cannot be
	// placed, '.' where it can, one character per rune.
	marked := func(s string, folding bool) string {
		out := make([]rune, 0, len(s))
		for _, give := range ZeroWidthRunsAfterFold([]rune(s), folding, nil) {
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

		// One vowel, and only its own run gives up -- the English on either
		// side of it keeps the fill it would have been given correctly.
		{"one pointed word", "abc ש" + testVowel + "לום xyz", false,
			"....#####...."},

		// The mark is on the run's LAST letter, where a run that stopped at its
		// last strong rune would have left it out.
		{"mark on the last letter", "abc שלום" + testVowel + " xyz", false,
			"....#####...."},

		// Two right-to-left words with only a space between them are one run to
		// such a terminal, so a vowel in either gives up both.
		{"two words, one vowel", "שלום ר" + testVowel + "ב", false,
			"########"},

		// Strong left-to-right content between them makes two runs, and only
		// the one carrying the vowel gives up.
		{"two runs, one vowel", "ש" + testVowel + "לום x רב", false,
			"#####....."},

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

// No run gives up its fill on a line that has nothing to give up. The line can
// say yes where every run says no -- a mark with no right-to-left run to sit in
// is in none of them -- but never the other way round.
func TestNoRunGivesUpWhatTheLineWouldKeep(t *testing.T) {
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
			for i, give := range ZeroWidthRunsAfterFold(runes, folding, nil) {
				if give && !line {
					t.Errorf("%q folding=%v: rune %d gave up its fill on a line "+
						"with nothing to give up", text, folding, i)
				}
			}
		}
	}
}
