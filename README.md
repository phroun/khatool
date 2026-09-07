# khatool

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **khatool** — *KittyTK's Hebrew & Arabic Text Output & Ordering Library ("חתול" — cat)*

*If you use this, please support me on ko-fi:  [https://ko-fi.com/jeffday](https://ko-fi.com/F2F61JR2B4)*

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/F2F61JR2B4)

Hebrew and Arabic for renderers that draw one cell at a time. It answers two
questions — **what order** the runes of a line are drawn in, and **which glyphs**
to draw — and nothing else: it has no opinion about fonts, cells, terminals or
documents, and its only dependency is `golang.org/x/text`.

It's a sibling to the rest of the line: **Mew** (text editor), **PurfecTerm**
(terminal emulator), **PawScript** (language), **KittyTK** (UI toolkit) and
**ifitfits** (viewport tiling). The name is a pun on חתול — *khatul*, cat.

## Model

- **Ordering** is the Unicode bidirectional algorithm (UAX #9, resolved by
  `x/text`) with the base direction **forced**: a caller states the direction a
  line begins in, and it holds regardless of the line's first strong character.
  RTL runs reverse, numbers inside an RTL region keep their own digit order
  while the region mirrors around them, and the result comes back as a
  permutation — visual slot to logical index. **Positions stay logical**, so a
  caller's cursor, selection and hit-testing arithmetic is untouched.
- **Output** is presentation-form substitution, one per script: Arabic letters
  become their contextual joining forms (`Shape`, including the mandatory
  lam-alef ligature), and Hebrew points fold into the letter that carries them
  (`PrecomposeCluster`). Both hand a renderer a single standalone glyph where it
  would otherwise have to shape or compose — which is exactly what a cell
  renderer, a bitmap font, or a terminal cannot be trusted to do.
- **Cell width is the caller's business.** Where the ordering has to know which
  runes share a cell — so a mark travels with its base when a run turns over —
  it asks, through a `Rides` function the caller supplies. Two answers to that
  question means two disagreeing pictures of one line, so the answer belongs to
  whoever will draw it. `nil` takes `DefaultRides`.

## Use

```go
import "github.com/phroun/khatool"

runes := []rune(line)
lay := khatool.Order(runes, baseRTL, nil) // nil: the default cluster rule
if lay == nil {
    draw(runes) // visual order is logical order — the common case
    return
}
for _, i := range lay.Perm {              // left to right on the screen
    r := runes[i]
    if lay.Glyph != nil {                 // Arabic shaped in logical order
        if r = lay.Glyph[i]; r == khatool.LigatureAbsorbed {
            continue                      // lam-alef: the pair took one cell
        }
    }
    if lay.RTL[i] {
        r = khatool.Mirror(r)             // brackets face the way the run reads
    }
    drawCell(r)
}
```

A caller that draws ill-formed marks or control characters as something visible
gives them cells of their own, and says so with a rule:

```go
lay := khatool.Order(runes, baseRTL, func(runes []rune, i int) bool {
    return myWidth(runes[i]) == 0 && !khatool.DefectiveMark(khatool.PrevBase(runes, i), runes[i])
})
```

## Test

```
go test ./...
```

## License

MIT — see [LICENSE](LICENSE).
