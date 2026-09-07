package khatool

import (
	"fmt"
	"sync"
	"unicode"
)

// Rides reports whether the rune at index i is drawn INTO the cell before it
// rather than taking one of its own -- a well-formed combining mark, a joiner,
// a zero-width space.
//
// This is the CLUSTER rule, and it belongs to the caller because it has to be
// the same question the caller's own width model answers: two answers means two
// disagreeing pictures of one line, and a reversal made against the wrong one
// puts marks on the wrong cells. A caller with no width model of its own can
// pass nil and take DefaultRides.
type Rides func(runes []rune, i int) bool

// DefaultRides is the rule for a caller with no width model to offer: a
// well-formed combining mark or a format character rides the cell before it,
// and everything else takes a cell of its own.
//
// A caller that paints a control or an ill-formed mark as something VISIBLE --
// a hex substitute, a dotted-circle anchor -- has given that rune a cell, and
// must say so with a rule of its own; this one assumes the plain reading.
func DefaultRides(runes []rune, i int) bool {
	if i < 0 || i >= len(runes) {
		return false
	}
	r := runes[i]
	if r == '\t' || IsControl(r) {
		return false
	}
	if !unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return false // takes a cell of its own
	}
	return !DefectiveMark(PrevBase(runes, i), r)
}

// IsControl reports whether r is a C0 or C1 control character. A control has no
// legitimate glyph, so whatever a caller draws for one takes a cell.
func IsControl(r rune) bool {
	return r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F)
}

// IsMark reports whether r is a combining mark - a codepoint that carries no
// cell of its own and paints into the preceding one.
func IsMark(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me)
}

// scriptCache memoizes scriptOf: a combining mark's script is fixed, and the
// lookup below scans every script table, which is too much to repeat for each
// mark of a fully pointed Hebrew or Arabic line.
var scriptCache sync.Map // rune -> string

// scriptOf names the Unicode script a rune belongs to ("Hebrew", "Han",
// "Common", ...), or "" when no script claims it.
func scriptOf(r rune) string {
	if v, ok := scriptCache.Load(r); ok {
		return v.(string)
	}
	name := ""
	for n, table := range unicode.Scripts {
		if unicode.Is(table, r) {
			name = n
			break
		}
	}
	scriptCache.Store(r, name)
	return name
}

// MarkAnchor is the base character an isolated mark is composed onto.
const MarkAnchor = '◌' // DOTTED CIRCLE

// DefectiveMark reports whether the combining mark r is ill-formed after the
// base character prev. Two cases:
//
//   - No base at all (prev == 0): the mark opens the line with nothing to
//     anchor onto.
//   - The mark is SCRIPT-SPECIFIC and the base belongs to a different script:
//     a Hebrew accent over a CJK ideograph, niqqud on a Latin letter, an NKo
//     tone on punctuation. Unicode calls such a sequence ill-formed, and no
//     shaper will compose it.
//
// Both this library and wcwidth call every combining mark zero-width, which is
// a promise about what a renderer will do: paint the mark INTO the preceding
// cell and advance nothing. A mark with nothing to compose onto, or a base a
// shaper refuses to attach it to, breaks that promise: the fallback is a
// SPACING glyph - .notdef, or the shaper's own dotted-circle plus mark - which
// advances a column nobody budgeted for.
//
// The test is about the SEQUENCE, never about which glyphs happen to be
// installed. Font inventory is deliberately not consulted: the glyph comes from
// whatever font the renderer ends up using, so any answer derived from one
// font's coverage would be about the wrong font. That also means legitimate
// text in a script the caller ships no face for - an NKo mark on an NKo letter
// - is left alone.
//
// General diacritics (script=Inherited/Common - the U+0300..U+036F block, the
// Arabic vowel marks, the kana voicing marks) belong to no script and
// legitimately attach to any base, so they are never defective on this rule.
func DefectiveMark(prev, r rune) bool {
	if !IsMark(r) {
		return false
	}
	if prev == 0 {
		return true // nothing to anchor onto
	}
	if prev == MarkAnchor {
		// A DOTTED CIRCLE the author actually typed is a legitimate base - it is
		// the Unicode character for carrying an isolated mark. So a mark on it
		// is well-formed: it composes onto that circle rather than being lifted
		// onto a second, substitute circle.
		return false
	}
	markScript := scriptOf(r)
	if markScript == "" || markScript == "Inherited" || markScript == "Common" {
		return false // general diacritic: attaches to any base
	}
	// Script-specific mark: well-formed only on a base of its own script.
	return scriptOf(prev) != markScript
}

// PrevBase returns the cluster base for the rune at index i: the nearest
// preceding rune that is not itself a combining mark, or 0 when there is none.
// It is what DefectiveMark wants for prev - a mark rides the last real
// character, not the mark in front of it.
func PrevBase(runes []rune, i int) rune {
	for j := i - 1; j >= 0; j-- {
		if !IsMark(runes[j]) {
			return runes[j]
		}
	}
	return 0
}

// Substitute is the visible stand-in for a rune that must not reach the
// renderer as itself: the C0 controls and DEL as the caret forms every terminal
// user knows (^@, ^I, ^[), and anything else as its codepoint in hex.
//
// C1 is the range that makes this more than a convenience. U+0080..U+009F
// arrives as ordinary two-byte UTF-8 and no width table calls it special, but a
// terminal decoding UTF-8 honours those codepoints as controls, and the range
// holds the string introducers -- DCS, SOS, CSI, ST, OSC, PM, APC. One of them
// emitted from a binary file makes the terminal swallow everything after it as
// a control string, so the rest of the line vanishes until a terminator that
// may never come.
//
// The form is always plain ASCII, so its cell count is its rune count and a
// caller's width model cannot disagree with what gets drawn.
func Substitute(r rune) string {
	v := int(r)
	if v <= 27 {
		switch v {
		case 0:
			return "^@"
		case 27:
			return "^["
		}
		return "^" + string(rune(v+64))
	}
	if v <= 0xFF {
		// One byte of hex reads unambiguously on its own: FE.
		return fmt.Sprintf("%02X", v)
	}
	// Past one byte the digits need a boundary, or a run of substituted
	// codepoints reads as one long number: (0123). Wider planes keep whole byte
	// pairs.
	if v <= 0xFFFF {
		return fmt.Sprintf("(%04X)", v)
	}
	return fmt.Sprintf("(%06X)", v)
}

// MarkForm is what a caller draws for a mark DefectiveMark rejects: a dotted
// circle supplying the base it has none of, with the mark composed onto it.
//
// This is the Unicode convention for showing an isolated combining mark, and it
// is what a shaper already does for a defective cluster -- so the reader sees
// the actual mark rather than a number, and it costs the circle's cell plus
// whatever the mark itself advances (nothing, for a non-spacing mark).
//
// It matters that the pair is drawn rather than left as a bare mark. A mark
// with no base it can attach to is corruption, not text, and it cannot be
// painted as zero-width: a renderer whose shaper rejects the pairing falls back
// to a SPACING glyph that advances a cell nobody budgeted, sliding the rest of
// the line along.
func MarkForm(r rune) string {
	return string(MarkAnchor) + string(r)
}
