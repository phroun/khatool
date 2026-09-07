package khatool

import "unicode"

// The counter-flip, for a host that runs its OWN bidi over what it is sent.
//
// A renderer that has already ordered a line holds it in VISUAL order: the
// leftmost cell first, right-to-left words turned over, brackets mirrored. Sent
// to a terminal that does no reordering, that is exactly what appears. Sent to
// one that does -- macOS Terminal.app is the one in common use -- the terminal
// orders it AGAIN, and a line already turned over comes back the wrong way
// round.
//
// So the renderer turns each right-to-left run back before it goes out, and the
// host's own bidi turns it forward again into the picture the renderer meant.
// Mirrored glyphs go back to the characters they mirror, since the host will
// mirror them itself.

// FlipRuns is the order to emit a row of cells in, for a host that applies its
// own bidi to what it receives.
//
// bases is one rune per cell, left to right, as the renderer laid the row out;
// a cell with nothing in it takes zero. Three answers come back, one entry per
// emitted slot:
//
//   - order is which cell's GLYPH goes in that slot.
//   - style is which cell's ATTRIBUTES go there.
//   - mirror is whether that glyph goes out as the character it mirrors.
//
// wordwise selects the run boundary AND the attribute mapping, to match how the
// host segments:
//
//   - Off: a run is a maximal right-to-left span with interior neutrals -- the
//     spaces between words -- absorbed as long as another right-to-left cell
//     follows. The whole span reverses, glyph and attributes together, because
//     the host reverses the parsed cell, colour and all, as a unit.
//   - On: a space is a boundary, so each whitespace-separated right-to-left word
//     is its own run and reverses in place. That host reverses the GLYPHS but
//     paints attributes at the physical column, so the glyphs reverse while the
//     attributes stay in the order they were laid out -- and each one then lands
//     on the letter its glyph settled on.
func FlipRuns(bases []rune, wordwise bool) (order, style []int, mirror []bool) {
	rtl := func(i int) bool { return bases[i] != 0 && IsStrongRTL(bases[i]) }
	strongLTR := func(i int) bool {
		r := bases[i]
		if r == 0 || IsStrongRTL(r) {
			return false
		}
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	order = make([]int, 0, len(bases))
	style = make([]int, 0, len(bases))
	mirror = make([]bool, 0, len(bases))
	for i := 0; i < len(bases); {
		if !rtl(i) {
			order = append(order, i)
			style = append(style, i)
			mirror = append(mirror, false)
			i++
			continue
		}
		// Extend the run. Word-wise stops at the first cell that is not
		// right-to-left (each word reverses in place); otherwise absorb interior
		// neutrals as long as another right-to-left cell follows before any
		// strong left-to-right content.
		end := i
		for j := i + 1; j < len(bases); j++ {
			if rtl(j) {
				end = j
				continue
			}
			if wordwise || strongLTR(j) {
				break
			}
		}
		for j := end; j >= i; j-- {
			order = append(order, j)
			mirror = append(mirror, true)
		}
		if wordwise {
			for j := i; j <= end; j++ { // attributes at the physical column
				style = append(style, j)
			}
		} else {
			for j := end; j >= i; j-- { // attributes reverse with the glyph
				style = append(style, j)
			}
		}
		i = end + 1
	}
	return order, style, mirror
}
