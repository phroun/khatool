package khatool

import (
	"strings"
	"testing"
)

// emit renders a row the way a host that applies its own bidi would be sent it:
// the glyphs in FlipRuns' order, mirrored where it says.
func emit(row string, wordwise bool) string {
	bases := []rune(row)
	order, _, mirror := FlipRuns(bases, wordwise)
	var b strings.Builder
	for k, i := range order {
		r := bases[i]
		if mirror[k] {
			r = Mirror(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// A row already laid out for the screen is turned BACK before it goes to a host
// that will order it itself -- so the host's own bidi turns it forward again
// into the picture the renderer meant.
func TestAFlippedRowComesBackTheRightWay(t *testing.T) {
	// The renderer's picture of "abc שלום xyz": the Hebrew turned over.
	const laidOut = "abc םולש xyz"
	// What goes out: the Hebrew back in the order the host will reverse.
	const sent = "abc שלום xyz"

	if got := emit(laidOut, false); got != sent {
		t.Errorf("whole-run flip of %q gave %q, want %q", laidOut, got, sent)
	}
	// Flipping twice is the picture again, which is what the host does to it.
	if got := emit(emit(laidOut, false), false); got != laidOut {
		t.Errorf("flipped twice, %q became %q", laidOut, got)
	}
	// Nothing right-to-left, nothing to do.
	if got := emit("plain text", false); got != "plain text" {
		t.Errorf("a row with no right-to-left content became %q", got)
	}
}

// The space between two right-to-left words is INSIDE the run for a host that
// reverses a whole parsed span, and a boundary for one that reverses word by
// word. Which it is decides where the words land.
func TestTheSpaceBetweenTwoWordsPicksTheRun(t *testing.T) {
	// Two Hebrew words as the renderer laid them out, second word leftmost.
	laidOut := "םולש בוט"

	whole := emit(laidOut, false)
	if strings.Count(whole, " ") != 1 {
		t.Fatalf("whole-run flip lost the space: %q", whole)
	}
	// The whole span reversed, space and all: what was leftmost is now last.
	if got := emit(whole, false); got != laidOut {
		t.Errorf("whole-run flip is not its own reverse: %q -> %q", laidOut, got)
	}

	// Word-wise, each word turns in place, so the words stay where they are.
	word := emit(laidOut, true)
	lw, rw := strings.Fields(laidOut), strings.Fields(word)
	if len(rw) != 2 || len(lw) != 2 {
		t.Fatalf("expected two words either way: %q -> %q", laidOut, word)
	}
	if reverse(lw[0]) != rw[0] || reverse(lw[1]) != rw[1] {
		t.Errorf("word-wise flip moved the words rather than turning each: %q -> %q",
			laidOut, word)
	}
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// A bracket the renderer mirrored goes out as the character it mirrors: the
// host mirrors it again on the way in.
func TestAMirroredBracketGoesOutAsItself(t *testing.T) {
	// "(שלום)" laid out: the run reversed, both brackets mirrored.
	laidOut := "(םולש)"
	sent := emit(laidOut, false)
	if strings.ContainsRune(sent, ')') && strings.IndexRune(sent, ')') < strings.IndexRune(sent, '(') {
		t.Errorf("the brackets were sent still mirrored: %q", sent)
	}
	if got := emit(sent, false); got != laidOut {
		t.Errorf("mirroring did not come back: %q -> %q -> %q", laidOut, sent, got)
	}
}

// Word-wise hosts paint attributes at the physical column, so the attribute
// order does not reverse with the glyphs; a whole-run host reverses both.
func TestAttributesFollowTheHostsOwnPainting(t *testing.T) {
	bases := []rune("םולש")

	_, style, _ := FlipRuns(bases, false)
	for k, i := range style {
		if want := len(bases) - 1 - k; i != want {
			t.Errorf("whole-run: attribute slot %d takes cell %d, want %d", k, i, want)
		}
	}

	_, style, _ = FlipRuns(bases, true)
	for k, i := range style {
		if i != k {
			t.Errorf("word-wise: attribute slot %d takes cell %d, want the physical %d",
				k, i, k)
		}
	}
}
