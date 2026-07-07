# Verified facts — P1.M3.T1.S1 (Unicode table alignment, Issue 2)

> Output dir is the orchestrator's `bugfix/001_7fdd8ce7410b/P1M3T1S1`. This is plan
> item **P1.M3.T1.S1** ("Implement rune-count displayWidth and fix padRight,
> wrapWords, column math, comment") — the only subtask of T7 in the build plan;
> here filed under the bugfix effort's M3 ("Unicode Table Alignment"). It fixes QA
> **Issue 2** (Minor, cosmetic).

## 0. Scope & dependency

- **Deliverable:** SURGICAL edits to `internal/ui/ui.go` (add a `displayWidth`
  helper; replace `len()` at 3 display-width sites; rewrite the false ASCII
  comment) + ADD tests to `internal/ui/ui_test.go` (unit tests for the helper +
  padRight + wrapWords, and a rune-based column-alignment integration test).
- **Decision:** `decisions.md §D2` — use stdlib `unicode/utf8.RuneCountInString`
  via a `displayWidth(s string) int` helper. **Do NOT add** `golang.org/x/text/
  width` or `runewidth` (PRD §4/§7.3 keep `gopkg.in/yaml.v3` the ONLY third-party
  dependency). **Zero new external dependency** → `go.mod`/`go.sum` UNCHANGED.
- **Parallel-safe:** the only other in-flight item is P1.M2.T1.S1 (Issue 4), which
  edits `internal/search/search.go`, `internal/search/search_test.go`, `main.go`.
  This subtask edits ONLY `internal/ui/*` — **zero file overlap**, no merge risk.

## 1. The three display-width `len()` sites (confirmed against the working tree)

| Site | File:line | Current (BUG) | Fixed |
|---|---|---|---|
| (a) padRight body | ui.go:133,136 | `if len(s) >= n` / `n-len(s)` | `displayWidth(s)` (both) |
| (b) column-width loop | ui.go:78-82 | `len(s.RelTag)` / `len(name)` | `displayWidth(s.RelTag)` / `displayWidth(name)` |
| (c) wrapWords | ui.go:154 | `len(cur)+1+len(word) <= width` | `displayWidth(cur)+1+displayWidth(word) <= width` |
| comment | ui.go:128-131 | "Operates on byte length ... tags are relative dir paths of the same [ASCII]" | rewrite: rune-count-based + CJK limitation |

**NOT touched (correctly):** `len(skills)==0` (slice length), `len(words)` in
wrapWords (slice length), `len(lines)` (test slice), and the header init
`tagW := len("TAG")` / `nameW := len("NAME")` (ASCII string literals — byte==rune;
the item scopes the column fix to the data-driven `len(s.RelTag)`/`len(name)`
loop only). Verified by scanning every `len(` in ui.go.

## 2. Charset reality — resolves an imprecision in the item's research note

The item's research note says "RelTag and Name ARE charset-constrained (check.go
validName regex)". This is **half right**:

- **Name** (frontmatter): charset-constrained. `check.go:97` defines
  `validName = ^[a-z0-9]+(-[a-z0-9]+)*$` and `check.go:195` applies it to
  `fm.Name` ONLY. So Name is ASCII → byte==display today.
- **RelTag** (the tag = directory name): **NOT charset-constrained.** `validName`
  is never applied to RelTag (grep confirms no `validName.MatchString(*.RelTag)`).
  The bug report (Issue 2) explicitly demonstrates a `café` directory/tag and
  states "directory names (and thus tags) are unrestricted" (§7.1 agrees). So a
  multi-byte TAG is a realistic, reachable misalignment vector — not just
  Description.
- **Description:** free-form prose, length-checked only (no charset rule).

**Conclusion for the PRP:** BOTH RelTag and Description are multi-byte vectors;
Name is ASCII today (but applying `displayWidth` there too is correct, free, and
future-proof). The comment rewrite must NOT repeat the false "tags are ASCII"
claim. The test uses a multi-byte TAG (`café`) per the bug report AND a
multi-byte Description (`—`) to cover both vectors.

## 3. The helper (stdlib, no new dep)

```go
func displayWidth(s string) int { return utf8.RuneCountInString(s) }
```
Import added to ui.go: `unicode/utf8`. Placement: package-level, right before
`padRight` (Go resolves package-level funcs regardless of order; grouping the
width primitive next to its first user reads cleanly). `utf8.RuneCountInString`
counts CODE POINTS, so é=1, —=1, most single-cell emoji=1, but a wide CJK rune
(e.g. 每) also =1 (it renders 2 cells) — the documented limitation (§D2).

## 4. CRITICAL — the existing `colOf` (byte offset) is BLIND to this bug

The item suggests asserting alignment "via the existing colOf() helper ... same
byte offset." **This does not work for multi-byte content**, and an empirical
trace proves why:

- With the **BUG** (byte padding): `padRight(s, tagW)` always emits exactly
  `max(len(s), tagW)` bytes, so every cell in a column has the SAME byte width
  regardless of rune content. Byte offsets of later columns are therefore
  **uniform across rows** → `colOf` (byte) reports the same column for every row
  → a `colOf`-equal assertion **PASSES even with the bug** (false pass).
- With the **FIX** (rune padding): a multi-byte tag padded to N *display* columns
  has MORE bytes than an ASCII tag padded to N columns (e.g. `padRight("café",5)`
  = `"café "` = 6 bytes; `padRight("ascii",5)` = `"ascii"` = 5 bytes). Byte
  offsets of later columns now **differ across rows** → `colOf`-equal would
  **FAIL after the fix** (false fail) — exactly inverted from correctness.

**Verified empirically** (throwaway Go program, café + ascii fixture):
```
runeCol alignment (FIX):  DESCRIPTION=19  café-skill=19  ascii-skill=19   ← aligned (PASS)
runeCol alignment (BUG):  DESCRIPTION=19  café-skill=18  ascii-skill=19   ← café off by 1 (bug DETECTED)
```
So the **rune-based** column offset detects the bug; the byte-based one cannot.

**Fix for the PRP:** add a test-local `runeCol(out, substr) int` helper that
returns the RUNE offset of `substr` within its line (`utf8.RuneCountInString` of
the line prefix), and assert `runeCol(desc) == runeCol("DESCRIPTION")` across all
rows. Unit tests on `displayWidth`/`padRight`/`wrapWords` are the precise guards
(the most robust, no column math at all); the `runeCol` integration test adds
end-to-end confidence. The existing ASCII `TestPrintListColumnsAlignedAcrossRows`
(colOf-based) is UNCHANGED and still passes (byte==rune for ASCII).

## 5. Exact expected values (empirically verified — use verbatim in tests)

```go
displayWidth("café") == 4   // 5 bytes, 4 runes
displayWidth("—")    == 1   // 3 bytes, 1 rune
displayWidth("a—b")  == 3   // 5 bytes, 3 runes
displayWidth("ascii")== 5
displayWidth("")     == 0

padRight("café", 5)  == "café "   // 1 trailing space (5-4 runes)
padRight("éé", 4)    == "éé  "    // 2 trailing spaces (4-2 runes)
padRight("ascii", 3) == "ascii"   // already wider -> no truncation
// (BUG contrast: padRight-by-len("café",5) == "café" — 0 spaces, the bug)

wrapWords("café bar", 8) == []string{"café bar"}   // 1 line: 4+1+3=8 <= 8
// (BUG contrast: by-len sees len("café")=5, 5+1+3=9 > 8 -> 2 lines ["café","bar"])
```

## 6. ASCII regression — nothing breaks

The 3 existing alignment/wrap tests use ASCII content, where byte==rune, so the
fix changes nothing observable for them:
- `TestPrintListColumnsAlignedAcrossRows` (ui_test.go:153, colOf/byte) — still
  passes (ASCII: byte offsets still uniform AND display-aligned).
- `TestPrintListWrapsLongDescription` (`len(ln)-descCol > descWrapWidth` on ASCII
  lines) — still passes (ASCII: rune count == byte count).
- All other ui_test.go tests — unaffected (they check content/presence, not width).

## 7. Imports + scope

- ui.go imports: ADD `unicode/utf8` (stdlib). Full set: fmt, io, strings,
  unicode/utf8, github.com/dabstractor/skpp/internal/discover.
- ui_test.go imports: ADD `unicode/utf8` (for runeCol). Full set: bytes, strings,
  testing, unicode/utf8, github.com/dabstractor/skpp/internal/discover.
- `go.mod`/`go.sum`: UNCHANGED (stdlib only). Verify `git diff --quiet go.mod go.sum`.
- Files touched: EXACTLY `internal/ui/ui.go`, `internal/ui/ui_test.go`. Nothing else.
- Test convention: white-box `package ui` (matches ui_test.go), plain t.Errorf,
  NO testify, NO t.Parallel() (repo convention). runeCol + new tests follow the
  existing helper/test style.

## 8. Confidence

The change is ~4 small `len()`→`displayWidth()` substitutions + a 1-line helper +
a comment rewrite + focused tests. Every expected value in §5 was EXECUTED in a
throwaway Go 1.25 program; the runeCol-detects-the-bug / colOf-is-blind result
(§4) is empirically proven. go.mod-neutral. Risk is limited to transcription
typos, caught by the Level 4 grep contract checks.
