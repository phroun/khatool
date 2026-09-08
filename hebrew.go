// Hebrew point folding: the points that Unicode also encodes as part of a
// single Alphabetic-Presentation-Form glyph are folded into that glyph, so a
// renderer that mispositions a free-standing point -- drawing a dagesh, dot or
// rafe a cell off its base -- has nothing left to misposition.
//
// It is the Hebrew counterpart of Shape: both substitute a presentation form so
// that a renderer with no shaper of its own is handed one glyph where it would
// otherwise have to compose two.
package khatool

import "unicode"

// dageshForm maps a Hebrew base letter + dagesh/mapiq to its presentation form.
// Letters with no such form (het, ayin, the final mem/nun/tsadi) are absent;
// their dagesh is dropped by PrecomposeCluster.
var dageshForm = map[rune]rune{
	0x05D0: 0xFB30, // alef + mapiq
	0x05D1: 0xFB31, // bet
	0x05D2: 0xFB32, // gimel
	0x05D3: 0xFB33, // dalet
	0x05D4: 0xFB34, // he + mapiq
	0x05D5: 0xFB35, // vav
	0x05D6: 0xFB36, // zayin
	0x05D8: 0xFB38, // tet
	0x05D9: 0xFB39, // yod
	0x05DA: 0xFB3A, // final kaf
	0x05DB: 0xFB3B, // kaf
	0x05DC: 0xFB3C, // lamed
	0x05DE: 0xFB3E, // mem
	0x05E0: 0xFB40, // nun
	0x05E1: 0xFB41, // samekh
	0x05E3: 0xFB43, // final pe
	0x05E4: 0xFB44, // pe
	0x05E6: 0xFB46, // tsadi
	0x05E7: 0xFB47, // qof
	0x05E8: 0xFB48, // resh
	0x05E9: 0xFB49, // shin (bare dagesh, no dot)
	0x05EA: 0xFB4A, // tav
}

// rafeForm maps a Hebrew base letter + rafe to its presentation form (only bet,
// kaf and pe have one).
var rafeForm = map[rune]rune{
	0x05D1: 0xFB4C, // bet
	0x05DB: 0xFB4D, // kaf
	0x05E4: 0xFB4E, // pe
}

// Folds reports whether r is a Hebrew point that folds into its base's
// presentation form: the dagesh/mapiq, shin dot, sin dot, rafe, or the
// holam-haser-for-vav. Vowels and accents do not fold.
func Folds(r rune) bool {
	switch r {
	case 0x05BC, 0x05C1, 0x05C2, 0x05BF, 0x05BA:
		return true
	}
	return false
}

// PrecomposeCluster folds a Hebrew cluster — a base rune followed by its
// combining marks — for a terminal that mishandles free-standing points. It
// returns the runes to emit: the base with its folding points folded into one
// presentation-form glyph, followed by the vowels/accents that ride normally;
// and ok=true when a fold happened.
//
// An isolated point anchored on a dotted circle (◌ + shin/sin dot, or ◌ +
// holam-haser) is shown on its faux base — the shin-with-dot or vav-with-holam
// glyph — since those points have no meaningful isolated rendering. A dagesh
// whose letter has no presentation form is dropped. ok=false when nothing folds
// (no such point, or a non-Hebrew base), and the caller emits the cluster as-is.
func PrecomposeCluster(runes []rune) ([]rune, bool) {
	if len(runes) < 2 {
		return nil, false
	}
	base := runes[0]

	if base == MarkAnchor {
		switch runes[1] {
		case 0x05C1:
			return []rune{0xFB2A}, true // shin with shin dot
		case 0x05C2:
			return []rune{0xFB2B}, true // shin with sin dot
		case 0x05BA:
			return []rune{0xFB4B}, true // vav with holam
		}
		return nil, false
	}
	if base < 0x05D0 || base > 0x05EA {
		return nil, false // not a Hebrew base letter
	}

	var hasDagesh, hasShinDot, hasSinDot, hasRafe, hasHolamHaser bool
	vowels := make([]rune, 0, len(runes)-1)
	for _, m := range runes[1:] {
		switch {
		case m == 0x05BC:
			hasDagesh = true
		case m == 0x05C1:
			hasShinDot = true
		case m == 0x05C2:
			hasSinDot = true
		case m == 0x05BF:
			hasRafe = true
		case m == 0x05BA && base == 0x05D5:
			hasHolamHaser = true
		default:
			vowels = append(vowels, m)
		}
	}
	if !hasDagesh && !hasShinDot && !hasSinDot && !hasRafe && !hasHolamHaser {
		return nil, false
	}

	pre := base // fall back to the bare letter, dropping an un-formable point
	switch {
	case base == 0x05E9 && hasDagesh && hasShinDot:
		pre = 0xFB2C // shin with dagesh and shin dot
	case base == 0x05E9 && hasDagesh && hasSinDot:
		pre = 0xFB2D // shin with dagesh and sin dot
	case base == 0x05E9 && hasShinDot:
		pre = 0xFB2A
	case base == 0x05E9 && hasSinDot:
		pre = 0xFB2B
	case hasHolamHaser:
		pre = 0xFB4B
	case hasDagesh:
		if f, ok := dageshForm[base]; ok {
			pre = f
		}
	case hasRafe:
		if f, ok := rafeForm[base]; ok {
			pre = f
		}
	}
	return append([]rune{pre}, vowels...), true
}

// ComposedBase folds base + its folding points into the single presentation-form
// glyph, ignoring any vowels. It is PrecomposeCluster's base rune alone — for
// callers that fold the base but handle the vowels themselves (e.g. drift, which
// moves the vowels to another cell). Returns the base unchanged, ok=false, when
// nothing folds.
func ComposedBase(runes []rune) (rune, bool) {
	folded, ok := PrecomposeCluster(runes)
	if !ok {
		if len(runes) > 0 {
			return runes[0], false
		}
		return 0, false
	}
	return folded[0], true
}

// HasZeroWidthAfterFold reports whether the runes still carry a ZERO-WIDTH one
// once each cluster has been folded into its presentation form.
//
// It answers a question about what a renderer can safely paint OVER. A terminal
// that runs its own bidi counts codepoints where a grid counts cells, so a
// background fill -- a selection bar, a highlight -- over a line holding
// combining marks lands on the wrong cells and half-vanishes. Foreground colour
// and weight ride each glyph through that reordering intact, so a caller with
// such a line reaches for those instead.
//
// Folding is what makes it worth asking per line rather than per script. A
// point that folds into its base no longer inflates the codepoint count, so a
// line of pointed consonants comes out even and keeps the ordinary fill; only
// marks that survive the fold -- vowels, accents, points with no form to fold
// into -- force the other treatment.
//
// zeroWidth is the caller's own width model, for the reason Rides is: two
// answers about which runes take a cell means two disagreeing pictures of one
// line. nil takes every non-spacing mark, which is the plain reading.
func HasZeroWidthAfterFold(runes []rune, folding bool, zeroWidth func(rune) bool) bool {
	if zeroWidth == nil {
		zeroWidth = func(r rune) bool { return unicode.In(r, unicode.Mn, unicode.Me) }
	}
	if !folding {
		for _, r := range runes {
			if zeroWidth(r) {
				return true
			}
		}
		return false
	}
	for i := 0; i < len(runes); {
		// A leading zero-width mark with no base of its own still counts.
		if zeroWidth(runes[i]) {
			return true
		}
		// Gather the zero-width marks riding this base into one cluster.
		j := i + 1
		for j < len(runes) && zeroWidth(runes[j]) {
			j++
		}
		folded, ok := PrecomposeCluster(runes[i:j])
		if !ok {
			folded = runes[i:j] // nothing folds: the cluster stands as written
		}
		for _, fr := range folded[1:] { // marks left over after the base
			if zeroWidth(fr) {
				return true
			}
		}
		i = j
	}
	return false
}

// UnplaceableFillAfterFold marks, for each rune, whether a background fill on
// its cell can still be placed by a terminal that reorders what it is sent.
//
// It is HasZeroWidthAfterFold's question asked per POSITION rather than of a
// whole line, because such a terminal misplaces the fill from one point
// onwards rather than everywhere. The damage starts at the first right-to-left
// run still carrying a zero-width mark after folding, and runs to the end of
// the line: that run's fill slides off the cells it was meant for, and
// everything after it slides with it. What comes BEFORE is placed correctly
// and keeps its fill, which is the whole point of asking -- a row of chrome
// and English with one pointed word in it has no business losing the fill on
// the half that would have been right.
//
// The trailing half matters as much as the run itself. A lone selected full
// stop after a pointed word is in no run of its own, and left with a fill it
// lands somewhere else entirely: one stray cell of colour behind a letter that
// was never selected.
//
// A run is what such a terminal takes as that unit: a span opened by a strong
// right-to-left rune, carrying the marks that ride its letters, and absorbing
// what is neither strong nor a mark -- the spaces between words -- as long as
// another strong right-to-left rune follows before any strong left-to-right
// one. Runes outside every run are false: there is nothing to reorder and the
// fill lands where it was put.
//
// zeroWidth is the caller's own width model, as it is for HasZeroWidthAfterFold
// and for Rides, so one line does not get two disagreeing pictures of itself.
// nil takes every non-spacing mark.
func UnplaceableFillAfterFold(runes []rune, folding bool, zeroWidth func(rune) bool) []bool {
	if zeroWidth == nil {
		zeroWidth = func(r rune) bool { return unicode.In(r, unicode.Mn, unicode.Me) }
	}
	strongLTR := func(r rune) bool {
		if IsStrongRTL(r) || zeroWidth(r) {
			return false
		}
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	out := make([]bool, len(runes))
	for i := 0; i < len(runes); {
		if !IsStrongRTL(runes[i]) {
			i++
			continue
		}
		// The run reaches as far as its last strong rune, plus the marks riding
		// that rune; a neutral between two of them is inside it, and strong
		// left-to-right content ends it.
		end := i
	run:
		for j := i + 1; j < len(runes); j++ {
			switch {
			case IsStrongRTL(runes[j]):
				end = j
			case zeroWidth(runes[j]):
				if j == end+1 {
					end = j // a mark riding the run's last letter
				}
			case strongLTR(runes[j]):
				break run
			}
		}
		if HasZeroWidthAfterFold(runes[i:end+1], folding, zeroWidth) {
			// From here to the end of the line. The run's own fill is
			// misplaced, and everything after it is carried along.
			for k := i; k < len(runes); k++ {
				out[k] = true
			}
			return out
		}
		i = end + 1
	}
	return out
}
