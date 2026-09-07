// Package khatool turns a line of Hebrew or Arabic text into the order and the
// glyphs a renderer needs when it can only draw one cell at a time: the visual
// ordering of the Unicode bidirectional algorithm (UAX #9, via
// golang.org/x/text), and the presentation forms that stand in for shaping.
//
// ORDERING. A line starts in a base direction the caller states — forced, not
// guessed from the first strong character. Runs of RTL text are resolved by the
// full bidi classification and reversed; numbers inside an RTL region keep
// their own left-to-right digit order while the region as a whole mirrors (the
// L2 reordering step applied over the resolved runs). Positions in the text
// stay strictly LOGICAL: only the layout is visual, so a caller's own cursor,
// selection and hit-testing arithmetic is left alone.
//
// OUTPUT. Arabic letters become their contextual presentation forms (Shape) and
// Hebrew points fold into theirs (Precompose): standalone glyphs a renderer can
// reverse, clip and colour cell by cell without a shaper of its own, and which
// a terminal cannot misplace.
//
// The library holds no opinion about how wide a glyph is. Where it has to know
// which runes share a cell — to keep a mark with its base through a reversal —
// it asks the caller (see Rides).
package khatool

import (
	xbidi "golang.org/x/text/unicode/bidi"
)

// Marker slot values that may appear in Layout.Perm when the layout was
// computed with direction markers: a negative entry is a synthetic
// one-column glyph at a fragment's edge rather than a real rune. The LTR/RTL
// markers point at a fragment's leading (reading-start) edge; MarkerEnd sits
// at the same fragment's trailing edge, showing where the foreign-direction
// run has ended.
const (
	MarkerLTR = -1 // ">" — an LTR fragment begins here (reading rightward)
	MarkerRTL = -2 // "<" — an RTL fragment begins here (reading leftward)
	MarkerEnd = -3 // "|" — the fragment's reading end
)

// Layout is the computed visual arrangement of one line.
type Layout struct {
	// Perm maps visual slot -> logical rune index: painting the line means
	// drawing runes[Perm[0]], runes[Perm[1]], ... left to right. When Marked,
	// negative MarkerLTR/MarkerRTL entries are synthetic direction-marker
	// cells (one column each) at fragment leading edges.
	Perm []int
	// RTL reports, per LOGICAL rune index, whether that rune lives in a
	// right-to-left run (used for bracket mirroring and the rtl command).
	RTL []bool
	// Marked records that the layout was computed with direction markers:
	// Perm may hold marker slots, and explicit direction-control characters
	// (normally zero-width) render one column wide as their own marker.
	Marked bool
	// Glyph holds the shaped glyph per LOGICAL rune index (Arabic cursive
	// presentation forms and lam-alef ligatures), or nil when the line has no
	// Arabic. A LigatureAbsorbed entry takes no cell (see Shape). Consumers
	// paint Glyph[i] and treat an absorbed entry as zero-width.
	Glyph []rune
}

// IsDirectionControl reports whether r is an explicit Unicode direction
// control (LRM, RLM, ALM, the embedding/override controls, or the isolate
// controls). Under showBidi these render as a visible one-column marker; a
// fragment led by one gets no additional synthetic marker — the character
// represents the transition itself.
func IsDirectionControl(r rune) bool {
	switch {
	case r == 0x200E || r == 0x200F || r == 0x061C:
		return true
	case r >= 0x202A && r <= 0x202E:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}

// classOf returns the bidi class of a rune.
func classOf(r rune) xbidi.Class {
	p, _ := xbidi.LookupRune(r)
	return p.Class()
}

// IsStrongRTL reports whether a rune is a strong right-to-left character
// (Hebrew, Arabic, and their presentation forms — bidi classes R and AL). It
// is the cheap question a caller asks of a finished row to find its RTL runs.
func IsStrongRTL(r rune) bool {
	if r < 0x0590 {
		return false
	}
	switch classOf(r) {
	case xbidi.R, xbidi.AL:
		return true
	}
	return false
}

// needsLayout reports whether a line contains any character that could make
// visual order differ from logical order under the given base direction. A
// pure-LTR line under an LTR base needs no layout at all — the fast path for
// virtually every line in an LTR document.
func needsLayout(runes []rune, baseRTL bool) bool {
	if baseRTL {
		return len(runes) > 0
	}
	for _, r := range runes {
		// Cheap ASCII rejection first: no ASCII character is R/AL/AN or an
		// explicit directional control.
		if r < 0x0590 {
			continue
		}
		if IsDirectionControl(r) {
			return true
		}
		switch classOf(r) {
		case xbidi.R, xbidi.AL, xbidi.AN, xbidi.Control:
			return true
		}
	}
	return false
}

// run is one directional run in logical order.
type run struct {
	start, end int // logical rune indices, inclusive
	rtl        bool
}

// resolveRuns computes the line's directional runs (logical order) under the
// base direction, using the full UAX #9 resolution.
//
// The base direction must be FORCED, not auto-detected: a caller states the
// direction a line begins in, and it holds regardless of the line's first
// strong character. x/text only forces RTL (DefaultDirection(LeftToRight)
// leaves the paragraph level auto-detected by first strong), so the LTR case
// prepends a zero-width LRM sentinel — its class L pins the paragraph level
// to 0 — and shifts the run indices back afterwards.
func resolveRuns(runes []rune, baseRTL bool) []run {
	var p xbidi.Paragraph
	var err error
	sentinel := 0
	if baseRTL {
		_, err = p.SetString(string(runes), xbidi.DefaultDirection(xbidi.RightToLeft))
	} else {
		_, err = p.SetString("‎" + string(runes))
		sentinel = 1
	}
	if err != nil {
		return nil
	}
	o, err := p.Order()
	if err != nil {
		return nil
	}
	runs := make([]run, 0, o.NumRuns())
	for i := 0; i < o.NumRuns(); i++ {
		r := o.Run(i)
		start, end := r.Pos()
		start -= sentinel
		end -= sentinel
		if end < 0 {
			continue // the run was only the sentinel
		}
		if start < 0 {
			start = 0
		}
		runs = append(runs, run{start: start, end: end, rtl: r.Direction() == xbidi.RightToLeft})
	}
	return runs
}

// numericRun reports whether every rune in the (parity-LTR) run is numeric
// content in the bidi sense — the digits and separators that take a HIGHER
// embedding level inside an RTL region rather than breaking it (UAX #9 gives
// EN/AN level base+2, so the surrounding region still mirrors around them).
func numericRun(runes []rune, rn run) bool {
	for i := rn.start; i <= rn.end; i++ {
		switch classOf(runes[i]) {
		case xbidi.EN, xbidi.AN, xbidi.ES, xbidi.ET, xbidi.CS, xbidi.NSM, xbidi.BN:
		default:
			return false
		}
	}
	return true
}

// appendReversed appends logical indices start..end to perm with the run's
// contents reversed cluster-wise: a base rune keeps its riders (combining
// marks, joiners) immediately after it, so they still compose onto the right
// cell once the run is turned over.
//
// What counts as a rider is the CALLER's question, not this file's, because
// whoever draws the line measures its cells by the same rule and the two
// pictures of a line have to be the same one. The case that proves it: a
// renderer that draws an ill-formed mark as a spacing substitute (a
// dotted-circle anchor, say) has given that mark a cell of its own, so it is
// its own cluster — glued to whatever preceded it, it would be dragged along in
// the reversal and land on the wrong side of a space, reading after the space
// instead of before it.
func appendReversed(perm []int, runes []rune, start, end int, rides Rides) []int {
	// Split into clusters: each cluster is a rune plus its zero-width
	// followers.
	type cluster struct{ s, e int }
	clusters := make([]cluster, 0, end-start+1)
	for i := start; i <= end; {
		j := i + 1
		for j <= end && rides(runes, j) {
			j++
		}
		clusters = append(clusters, cluster{s: i, e: j - 1})
		i = j
	}
	for i := len(clusters) - 1; i >= 0; i-- {
		for k := clusters[i].s; k <= clusters[i].e; k++ {
			perm = append(perm, k)
		}
	}
	return perm
}

// Order returns the visual layout of a line under the base direction, or nil
// when visual order equals logical order (the common pure-LTR case — the caller
// should use its ordinary sequential path).
//
// rides says which runes share a cell with the one before them, so a mark stays
// with its base when a run is turned over; nil takes DefaultRides.
//
// Reordering applies UAX #9's L2 step over the resolved runs:
//   - LTR base: maximal regions of RTL runs (absorbing numeric runs flanked by
//     RTL on both sides) mirror as a whole — run sequence reversed, RTL run
//     contents reversed, numeric runs kept digit-order.
//   - RTL base: the entire run sequence is reversed (the line reads from the
//     right), RTL run contents reversed, LTR/numeric run contents kept.
func Order(runes []rune, baseRTL bool, rides Rides) *Layout {
	return order(runes, baseRTL, false, rides)
}

// OrderMarked is Order with direction markers: a synthetic one-column marker
// slot is injected at the leading edge of every fragment except the
// line-initial fragment when it is in the natural (base) direction — and except
// fragments led by an explicit direction-control character, which represents the
// transition itself (rendered one column wide by consumers when Layout.Marked
// is set). It is how a caller shows the reader where the direction turned.
func OrderMarked(runes []rune, baseRTL bool, rides Rides) *Layout {
	return order(runes, baseRTL, true, rides)
}

func order(runes []rune, baseRTL bool, marked bool, rides Rides) *Layout {
	if rides == nil {
		rides = DefaultRides
	}
	if len(runes) == 0 || !needsLayout(runes, baseRTL) {
		return nil
	}
	runs := resolveRuns(runes, baseRTL)
	if runs == nil {
		return nil
	}

	rtl := make([]bool, len(runes))
	anyRTL := false
	for _, rn := range runs {
		for i := rn.start; i <= rn.end; i++ {
			rtl[i] = rn.rtl
		}
		if rn.rtl {
			anyRTL = true
		}
	}
	if !baseRTL && !anyRTL && !(marked && len(runs) > 1) {
		return nil // resolved to pure LTR after all
	}

	// markedRun reports whether a fragment gets a marker: every fragment
	// after the first, plus a line-initial fragment in the foreign
	// direction. A fragment led by an explicit direction control speaks for
	// itself (the control renders as the marker).
	logicalIdx := make(map[int]int, len(runs)) // run start -> logical order
	for i, rn := range runs {
		logicalIdx[rn.start] = i
	}
	markedRun := func(rn run) bool {
		if !marked {
			return false
		}
		if IsDirectionControl(runes[rn.start]) {
			return false
		}
		if logicalIdx[rn.start] == 0 {
			return rn.rtl != baseRTL
		}
		return true
	}

	perm := make([]int, 0, len(runes)+2*len(runs))
	emit := func(rn run) {
		if rn.rtl {
			// Leading edge of an RTL fragment is its RIGHTMOST cell: the
			// begin marker follows the reversed content, and the end marker
			// (the fragment's reading end) precedes it on the left.
			if markedRun(rn) {
				perm = append(perm, MarkerEnd)
			}
			perm = appendReversed(perm, runes, rn.start, rn.end, rides)
			if markedRun(rn) {
				perm = append(perm, MarkerRTL)
			}
		} else {
			if markedRun(rn) {
				perm = append(perm, MarkerLTR)
			}
			for i := rn.start; i <= rn.end; i++ {
				perm = append(perm, i)
			}
			if markedRun(rn) {
				perm = append(perm, MarkerEnd)
			}
		}
	}

	if baseRTL {
		// Whole line mirrors: visit runs in reverse order.
		for i := len(runs) - 1; i >= 0; i-- {
			emit(runs[i])
		}
		return &Layout{Perm: perm, RTL: rtl, Marked: marked, Glyph: Shape(runes)}
	}

	// LTR base: mirror each maximal RTL region (RTL runs plus numeric runs
	// enclosed between RTL runs) as a unit; everything else stays in place.
	for i := 0; i < len(runs); {
		if !runs[i].rtl {
			emit(runs[i])
			i++
			continue
		}
		// Region of runs [i, j): RTL runs, or numeric runs with an RTL run
		// later in the region (checked by lookahead).
		j := i + 1
		for j < len(runs) {
			if runs[j].rtl {
				j++
				continue
			}
			// A numeric run continues the region only when another RTL run
			// follows it directly after any numeric neighbors.
			if numericRun(runes, runs[j]) {
				k := j + 1
				for k < len(runs) && !runs[k].rtl && numericRun(runes, runs[k]) {
					k++
				}
				if k < len(runs) && runs[k].rtl {
					j = k + 1
					continue
				}
			}
			break
		}
		// Mirror the region: run sequence reversed; RTL contents reversed.
		for k := j - 1; k >= i; k-- {
			emit(runs[k])
		}
		i = j
	}
	return &Layout{Perm: perm, RTL: rtl, Marked: marked, Glyph: Shape(runes)}
}

// RTLAt reports whether the rune at logical index idx sits in an RTL run
// (the registered rtl command). For an index at or past the end of the line
// the direction of the last rune applies; an empty line reports the base
// direction.
func RTLAt(runes []rune, idx int, baseRTL bool) bool {
	if len(runes) == 0 {
		return baseRTL
	}
	if !needsLayout(runes, baseRTL) {
		return false
	}
	runs := resolveRuns(runes, baseRTL)
	if runs == nil {
		return baseRTL
	}
	if idx >= len(runes) {
		idx = len(runes) - 1
	}
	if idx < 0 {
		idx = 0
	}
	for _, rn := range runs {
		if idx >= rn.start && idx <= rn.end {
			return rn.rtl
		}
	}
	return baseRTL
}

// Mirror returns the paired counterpart of a bracket rune (for painting
// brackets inside RTL runs), or the rune unchanged.
func Mirror(r rune) rune {
	p, _ := xbidi.LookupRune(r)
	if !p.IsBracket() {
		return r
	}
	// ReverseString of a single rune applies x/text's full mirroring table.
	rs := []rune(xbidi.ReverseString(string(r)))
	if len(rs) == 1 {
		return rs[0]
	}
	return r
}
