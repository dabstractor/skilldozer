# PRP — P1.M3.T1.S1: Unicode display width in `ui` table rendering (Issue 2)

> **Subtask:** P1.M3.T1.S1 — fixes QA **Issue 2** (Minor, cosmetic): the
> `--list` / `--search` table misaligns for multi-byte UTF-8 content because
> `ui.padRight`, the column-width loop, and `wrapWords` all measure **byte**
> length (`len(s)`) instead of display width. A tag like `café` (4 columns, 5
> bytes) or a description with `—` (1 column, 3 bytes) is under-counted, shifting
> that row's NAME/DESCRIPTION columns left.
> **Decision:** `decisions.md §D2` — add a stdlib `displayWidth` helper using
> `unicode/utf8.RuneCountInString`; **no third-party width library** (PRD §4/§7.3
> keep `gopkg.in/yaml.v3` the only dependency).
>
> **Scope:** SURGICAL edits to `internal/ui/ui.go` (a `displayWidth` helper, 3
> `len()`→`displayWidth()` substitutions, a rewritten comment, one new import) and
> ADDITIVE tests in `internal/ui/ui_test.go` (unit tests for the helper/padRight/
> wrapWords + a rune-based column-alignment integration test). Mode A — README
> deferred to P1.M5.T3. **No other file touched.**
>
> **PARALLEL CONTEXT:** the only other in-flight item is P1.M2.T1.S1 (Issue 4,
> search fields), which edits `internal/search/*` and `main.go`. This subtask
> edits ONLY `internal/ui/*` — **zero file overlap**, no merge risk regardless of
> landing order.
>
> **CRITICAL RESEARCH FINDING (see verified_facts §4):** the item's suggested
> "assert alignment via the existing `colOf()` (byte offset)" does NOT work for
> multi-byte content — byte-padding yields uniform byte widths even when the
> display is misaligned, so `colOf` is *blind* to this bug and would even *fail*
> after the fix. The PRP uses a **rune-based** `runeCol` helper for the
> integration test (empirically proven to detect the bug and pass post-fix), plus
> precise unit tests that need no column math at all.

---

## Goal

**Feature Goal**: Make the `--list`/`--search` table columns align regardless of
character set. Replace every `len(s)` used for **display** width in `ui.go` with a
`displayWidth(s)` helper backed by `utf8.RuneCountInString`, so multi-byte runes
(é, —, smart quotes, single-cell emoji) are counted as one column. Fix the false
"tags are ASCII" code comment. ASCII-only output is unchanged (rune count == byte
count for ASCII).

**Deliverable**: Surgical edits to 2 files:
1. `internal/ui/ui.go` — add `displayWidth`; add `unicode/utf8` import; substitute
   `len()`→`displayWidth()` at the 3 display-width sites (column-width loop,
   `padRight`, `wrapWords`); rewrite the `padRight` comment (rune-count-based +
   CJK limitation).
2. `internal/ui/ui_test.go` — add a `runeCol` helper + 4 tests
   (`TestDisplayWidth`, `TestPadRightMultibyte`, `TestWrapWordsMultibyte`,
   `TestPrintListColumnsAlignedForMultibyte`).

**Success Definition**: `go test ./internal/ui/ -v` passes (new tests green, all
existing ASCII tests unchanged & green); `go test ./...` whole module green;
`gofmt`/`go vet` clean; `go.mod`/`go.sum` unchanged; only the 2 files changed
(`git diff --name-only`). The bug-report `café` snippet now renders aligned (the
café row's DESCRIPTION starts at the same display column as the ascii row's).

---

## Why

- **Correctness of a user-visible table.** Issue 2 is cosmetic but real: a user
  with a `café` skill (or any non-ASCII description) sees a visibly crooked
  `--list`/`--search` table. Tags are NOT charset-restricted (§7.1; the bug report
  demonstrates `café`), and descriptions are free-form prose — both are reachable
  multi-byte vectors. Rune-count width fixes the common case (é, —, smart quotes,
  single-cell emoji) with **zero dependency cost**.
- **Respects the hard dependency policy.** PRD §4/§7.3 deliberately keep
  `gopkg.in/yaml.v3` the only third-party dependency. A perfect fix (full
  East-Asian width tables via `golang.org/x/text/width` / `runewidth`) would
  violate that. `utf8.RuneCountInString` is stdlib, fixes every common case, and
  the residual imperfection (wide CJK runes counted as 1) is documented in code
  (decisions.md §D2).
- **Kills a false assumption at the source.** The `padRight` comment currently
  asserts "tags are relative dir paths of the same [ASCII]" — which is false (only
  the frontmatter `name` is ASCII-restricted via `check.go validName`; tags and
  descriptions are not). A future maintainer reading that comment would re-introduce
  `len()` and silently re-break the table. Rewriting the comment prevents that.
- **Lowest-risk change possible.** ~4 one-line substitutions + a 1-line helper + a
  comment + additive tests. No struct, no API, no flag, no exit-code, no
  stdout/stderr contract change. ASCII output is byte-identical.

---

## What

`ui.go` gains a `displayWidth(s string) int` helper (`utf8.RuneCountInString`) and
uses it everywhere a string's **rendered** width matters. The `padRight` doc
comment is rewritten to state the rune-count model and its CJK limitation.

### Behavior change (multi-byte cells align; ASCII unchanged)

| Content | Before (byte width) | After (rune width) |
|---|---|---|
| tag `café` in a column of width 5 | under-padded → row shifts left 1 col | padded to 5 cols → aligned |
| description with `—`/`é` | wrap/pad use byte count → misaligned/wrong wrap point | rune count → correct |
| ASCII-only table | correct | correct (byte==rune) |
| wide CJK rune (e.g. 每, 2 cells) | counted as len(bytes) | counted as 1 (documented limitation) |

### Success Criteria

- [ ] `ui.go` has `func displayWidth(s string) int { return utf8.RuneCountInString(s) }` and imports `unicode/utf8`
- [ ] column-width loop uses `displayWidth(s.RelTag)` / `displayWidth(name)` (not `len`)
- [ ] `padRight` uses `displayWidth(s)` in both the guard and the `Repeat` arg (not `len(s)`)
- [ ] `wrapWords` uses `displayWidth(cur)+1+displayWidth(word)` (not `len`)
- [ ] `padRight` comment rewritten: states rune-count-based; notes the wide-CJK limitation + PRD §4/§7.3 rationale; no longer claims tags are ASCII
- [ ] NO other `len()` changed (slice lengths `len(skills)`/`len(words)` and ASCII header init `len("TAG")`/`len("NAME")` stay)
- [ ] `ui_test.go` adds `runeCol` helper + the 4 named tests; existing tests UNCHANGED
- [ ] `go test ./internal/ui/ -v` green; `go test ./...` green; `gofmt -l`/`go vet ./...` clean
- [ ] `git diff --quiet go.mod go.sum` (no new dependency); `git diff --name-only` == exactly the 2 ui files

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT current code of every edit site (with file:line), the exact
old→new text for each, the exact expected test values (all EXECUTED in a throwaway
Go 1.25 program — verified_facts §5), and the empirically-proven reason the
existing `colOf` cannot be used (§4) are all in the Implementation Blueprint. An
implementer who knows Go but nothing about this repo can apply the edits and tests
verbatim._

### Documentation & References

```yaml
# MUST READ — this subtask's verified facts (every load-bearing decision)
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/P1M3T1S1/research/verified_facts.md
  why: "§1 the 3 len() sites with exact file:line. §2 charset reality (Name IS
        ASCII via check.go:97/195; RelTag is NOT — bug report shows café;
        Description is free-form). §3 the helper. §4 CRITICAL: colOf (byte) is
        BLIND to this bug — use runeCol; empirically proven. §5 exact expected
        values (executed). §6 ASCII regression safe. §7 imports + scope."
  critical: "Do NOT use the existing colOf() for the multi-byte alignment test —
             it passes with the bug AND fails after the fix (byte widths differ
             post-fix). Use the rune-based runeCol helper provided. Do NOT change
             len(\"TAG\")/len(\"NAME\") header init (ASCII literals, out of scope)."

# CONTRACT — the decision this implements
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/decisions.md
  why: "§D2: use stdlib utf8.RuneCountInString; do NOT add golang.org/x/text/width
        or runewidth (PRD §4/§7.3 dependency policy). Document the wide-CJK
        limitation in the code comment."
  section: "D2"

# CONTRACT — the issue root cause + fix surface + test impact
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/issue_analysis.md
  why: "Issue 2: root cause (padRight len(s) at ui.go:132; also column-width
        79-82 and wrapWords:143/154), the utf8.RuneCountInString fix, and the
        test impact (ASCII TestPrintListColumnsAlignedAcrossRows still passes; new
        multi-byte test via colOf). NOTE: issue_analysis suggests colOf; this PRP
        corrects that to runeCol per verified_facts §4."
  section: "Issue 2 (MINOR)"

# THE FILE BEING EDITED — the 3 len() sites + the false comment
- file: internal/ui/ui.go
  why: "column-width loop :78-82 (len(s.RelTag)/len(name)); padRight comment+body
        :128-136 (len(s)>=n, n-len(s), and the 'tags are ASCII' comment);
        wrapWords :154 (len(cur)+1+len(word)). Imports block at top."
  pattern: "Replace len() with displayWidth() ONLY at display-width sites; leave
            slice lengths (len(skills), len(words)) and ASCII header init
            (len(\"TAG\"), len(\"NAME\")) untouched."
  gotcha: "padRight is applied to PLAIN text BEFORE paint (ui.go:88-92) — keep that
           invariant; displayWidth must never see ANSI escape bytes (it would
           count them). The fix changes the width MATH, not the paint order."

# THE TEST FILE — colOf helper (byte) + the ASCII alignment test to keep passing
- file: internal/ui/ui_test.go
  why: "colOf :18 (byte offset — used by the ASCII tests, keep it); mk :13 (skill
        builder); TestPrintListColumnsAlignedAcrossRows :153 (ASCII, UNCHANGED);
        TestPrintListWrapsLongDescription (ASCII len(ln)-based, UNCHANGED). Add a
        runeCol helper + 4 new tests; do NOT modify existing tests."
  pattern: "White-box package ui; bytes.Buffer + PrintList; plain t.Errorf; no
            testify; no t.Parallel(). New tests follow the same style."

# CHARSET EVIDENCE — proves RelTag is NOT charset-validated (only Name is)
- file: internal/check/check.go
  why: ":97 validName = ^[a-z0-9]+(-[a-z0-9]+)*$; :195 applied to fm.Name ONLY
        (never to RelTag). So a multi-byte tag like café is NOT flagged by check
        and is a realistic input. This is why the test fixture uses a café tag."
  pattern: "READ-ONLY reference. Do NOT modify check.go."

# URLS — the load-bearing stdlib surface
- url: https://pkg.go.dev/unicode/utf8#RuneCountInString
  why: "utf8.RuneCountInString(s) returns the number of runes (code points) in s —
        the display-width approximation. é→1, —→1, café→4 (vs 5 bytes). This is the
        ONE stdlib call the whole fix hinges on; no third-party dep."
- url: https://pkg.go.dev/unicode/utf8#DecodeRuneInString
  why: "Reference only — RuneCountInString is what we use. (A hand-rolled range
        loop over runes would be equivalent but less self-documenting.)"
```

### Current Codebase tree (relevant slice)

```bash
$ cd /home/dustin/projects/skpp && ls internal/ui/
internal/ui/
├── ui.go          # padRight :128-136, column loop :78-82, wrapWords :154, imports — EDIT
└── ui_test.go     # colOf :18, mk :13, ASCII tests — ADD runeCol + 4 tests
# (all other packages — discover, resolve, search, check, skillsdir, main — UNCHANGED)
```

### Desired Codebase tree (files touched)

```bash
skpp/
└── internal/ui/
    ├── ui.go          # MODIFY — displayWidth helper + 3 substitutions + comment + import
    └── ui_test.go     # MODIFY (additive) — runeCol helper + 4 new tests
# (go.mod, go.sum, every other file — UNCHANGED; zero new dependency)
```

| File | Change | Lines (approx) |
|---|---|---|
| `internal/ui/ui.go` | add `unicode/utf8` import; add `displayWidth`; `len()`→`displayWidth()` ×3 sites; rewrite padRight comment | imports; 78-82; 128-136; 154 |
| `internal/ui/ui_test.go` | add `unicode/utf8` import; add `runeCol`; add 4 tests | end of file |

### Known Gotchas of our codebase & the width logic

```go
// GOTCHA #1 — Use displayWidth() ONLY at DISPLAY-width sites, never at slice sites.
// The 3 sites: column-width loop (len(s.RelTag)/len(name)), padRight (len(s) in
// guard + Repeat arg), wrapWords (len(cur)+1+len(word)). LEAVE ALONE: len(skills)
// (slice length), len(words)/len(lines) (slice lengths), and the ASCII header
// init `tagW := len("TAG")` / `nameW := len("NAME")` (string LITERALS — byte==rune,
// and the item scopes the column fix to the data loop only). Verified §1.
//   RIGHT: if displayWidth(s.RelTag) > tagW { tagW = displayWidth(s.RelTag) }
//   WRONG: tagW := displayWidth("TAG")   // needless; "TAG" is an ASCII literal

// GOTCHA #2 — The existing colOf() (BYTE offset) is BLIND to this bug. Do NOT use
// it for the multi-byte alignment test. With byte-padding every cell in a column
// has the same byte width, so byte offsets are uniform even when the DISPLAY is
// misaligned -> colOf-equal PASSES with the bug (false pass). After the fix, a
// rune-padded multi-byte cell has MORE bytes than an ASCII one (café=6 bytes,
// ascii=5 bytes for width 5), so byte offsets DIFFER -> colOf-equal FAILS post-fix
// (false fail). Use the rune-based runeCol() helper instead. Empirically proven §4.
//   RIGHT: runeCol(out, "café skill") == runeCol(out, "DESCRIPTION")  // rune offset
//   WRONG: colOf(out, "café skill")   == colOf(out, "DESCRIPTION")    // byte offset

// GOTCHA #3 — RelTag is NOT charset-constrained (the item's note is imprecise).
// check.go validName (:97) is applied (:195) to fm.Name ONLY, never RelTag. The
// bug report demonstrates a café tag and §7.1 says directory names are unrestricted.
// So a multi-byte TAG is realistic — the test MUST use a café tag (not just a
// multi-byte description). Name IS ASCII today; applying displayWidth there too is
// correct + free + future-proof. Verified §2.

// GOTCHA #4 — Keep the paint-before-color invariant (ui.go:88-92). padRight is
// applied to PLAIN text BEFORE paint(), specifically so ANSI escape bytes stay
// out of width math. displayWidth must never see escape bytes. The fix changes the
// width MATH (len->displayWidth), NOT the paint order. Do not move padRight after
// paint or wrap painted strings.

// GOTCHA #5 — Rune count ≠ true display width for wide CJK runes. 每 (display 2
// cells) counts as 1 rune. This is the documented limitation (decisions.md §D2):
// a full width table would need a third-party dep the PRD forbids. The comment
// MUST state this so no one assumes perfection. Common cases (é, —, smart quotes,
// single-cell emoji) ARE fixed. Do NOT try to special-case CJK — that's the
// explicitly-avoided path.

// GOTCHA #6 — Exact expected values are verified; use them verbatim. padRight
// ("café",5) == "café " (1 space); padRight("éé",4) == "éé  " (2 spaces);
// wrapWords("café bar",8) == ["café bar"] (1 line); displayWidth("café")==4,
// displayWidth("—")==1, displayWidth("a—b")==3. The BUG contrasts: byte-pad of
// "café" to 5 == "café" (0 spaces); byte-wrap of "café bar" to 8 == 2 lines. §5.

// GOTCHA #7 — go.mod/go.sum UNCHANGED. unicode/utf8 is stdlib; no new module.
// Verify `git diff --quiet go.mod go.sum`. If go mod tidy changes anything, you
// accidentally added an external import (you should not have).

// GOTCHA #8 — Additive tests only; do NOT modify existing ui_test.go tests.
// TestPrintListColumnsAlignedAcrossRows (:153, colOf/byte) and
// TestPrintListWrapsLongDescription (len(ln)-based) use ASCII, where byte==rune,
// so they still pass unchanged. Editing them is scope creep and risk. ADD the new
// tests; leave the rest.

// GOTCHA #9 — Test convention: package ui (white-box), plain t.Errorf/t.Fatalf,
// NO testify, NO t.Parallel() (matches the existing ui_test.go). runeCol mirrors
// the colOf style (Index + LastIndex) but counts runes via RuneCountInString.

// GOTCHA #10 — This runs IN PARALLEL with P1.M2.T1.S1 (Issue 4), which edits
// internal/search/search.go, internal/search/search_test.go, and main.go. You
// edit ONLY internal/ui/ui.go and internal/ui/ui_test.go. Zero overlap. No merge
// conflict possible regardless of landing order.
```

---

## Implementation Blueprint

### Data models and structure

**No new data models.** This is a width-math fix in the existing `ui` formatter.
The only new symbol is a 1-line helper:

```go
// displayWidth approximates a string's display width as its UTF-8 rune count.
func displayWidth(s string) int { return utf8.RuneCountInString(s) }
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT internal/ui/ui.go — add the unicode/utf8 import
  - FILE: internal/ui/ui.go
  - LOCATE: the import block (top of file).
  - EDIT: see "Edit A". Add "unicode/utf8" in stdlib group (alpha-sorted, before
          the internal/discover group).
  - GOTCHA: this is the ONLY new import; no external module (go.mod unchanged).

Task 2: EDIT internal/ui/ui.go — column-width loop (site b)
  - FILE: internal/ui/ui.go, lines 78-82 (inside the `for i, s := range skills` loop).
  - EDIT: see "Edit B". len(s.RelTag) -> displayWidth(s.RelTag) (both the compare
          and the assignment); same for len(name) -> displayWidth(name).
  - GOTCHA: leave `tagW := len("TAG")` / `nameW := len("NAME")` (ASCII literals).

Task 3: EDIT internal/ui/ui.go — displayWidth helper + padRight comment + body (sites a + helper)
  - FILE: internal/ui/ui.go, lines 128-136 (the padRight comment + func).
  - EDIT: see "Edit C". INSERT the displayWidth helper BEFORE padRight; REWRITE
          the padRight comment (rune-count-based + CJK limitation, no "tags ASCII"
          claim); change len(s) -> displayWidth(s) in the guard and the Repeat arg.
  - GOTCHA: keep padRight applied to PLAIN text (paint-before-color invariant).
            Place displayWidth at package level right before padRight.

Task 4: EDIT internal/ui/ui.go — wrapWords (site c)
  - FILE: internal/ui/ui.go, line 154.
  - EDIT: see "Edit D". len(cur)+1+len(word) -> displayWidth(cur)+1+displayWidth(word).
  - GOTCHA: leave len(words) (slice length) untouched.

Task 5: EDIT internal/ui/ui_test.go — add unicode/utf8 import + runeCol helper + 4 tests
  - FILE: internal/ui/ui_test.go.
  - EDIT: see "Edit E". Add "unicode/utf8" to imports; add runeCol helper (rune
          offset within a line) after colOf; APPEND the 4 tests at end of file.
  - DO NOT TOUCH: colOf, mk, or any existing test.
  - GOTCHA: use runeCol (NOT colOf) for the multi-byte alignment assertion (§4).

Task 6: VALIDATE (all gates green)
  - gofmt -w internal/ui/ui.go internal/ui/ui_test.go
  - test -z "$(gofmt -l .)"   # whole tree gofmt-clean
  - go vet ./...              # clean
  - go build ./...            # compiles
  - go test ./internal/ui/ -v   # all ui tests pass (4 new + existing ASCII)
  - go test ./...             # whole module green (regression guard)
  - git diff --quiet go.mod go.sum   # NO new dependency
  - Level 3 smoke test (bug-report café snippet now aligns)
  - Level 4 scope check (git diff --name-only == exactly the 2 ui files)
```

### Edit A — `ui.go` imports (Task 1)

```
OLD:
import (
	"fmt"
	"io"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

NEW:
import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/dabstractor/skpp/internal/discover"
)
```

### Edit B — `ui.go` column-width loop (Task 2)

```
OLD:
		if len(s.RelTag) > tagW {
			tagW = len(s.RelTag)
		}
		if len(name) > nameW {
			nameW = len(name)
		}

NEW:
		if displayWidth(s.RelTag) > tagW {
			tagW = displayWidth(s.RelTag)
		}
		if displayWidth(name) > nameW {
			nameW = displayWidth(name)
		}
```

### Edit C — `ui.go` displayWidth helper + padRight comment + body (Task 3)

```
OLD:
// padRight returns s right-padded with spaces to width n. If len(s) >= n, s is
// returned unchanged (no truncation). Operates on byte length, which is correct
// for the ASCII values skpp handles (Agent Skills names are lowercase a-z0-9-;
// tags are relative dir paths of the same).
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

NEW:
// displayWidth returns the number of display columns s occupies, approximated as
// its UTF-8 rune count. It replaces len(s) wherever a string's rendered width
// matters (column sizing, padding, word-wrap) so a multi-byte rune like é (2
// bytes, 1 rune) or — (3 bytes, 1 rune) counts as one column instead of 2–4 bytes.
// KNOWN LIMITATION: wide CJK runes that render two cells wide are still counted
// as one; a full East-Asian width table would be needed for that, deliberately
// avoided to keep skpp dependency-free (PRD §4/§7.3). See padRight for usage.
func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// padRight returns s right-padded with spaces to display width n. If s is already
// n or more columns wide it is returned unchanged (no truncation). Width is
// measured in RUNES via displayWidth (utf8.RuneCountInString), not bytes: a
// multi-byte rune like é (2 bytes, 1 column) or — (3 bytes, 1 column) renders one
// cell wide, so rune count gives correct padding for the common case (é, —, smart
// quotes, single-cell emoji). Applied to PLAIN text before paint so ANSI color
// escapes stay out of the width math (their bytes would otherwise corrupt padding).
func padRight(s string, n int) string {
	if displayWidth(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-displayWidth(s))
}
```

### Edit D — `ui.go` wrapWords (Task 4)

```
OLD:
		case len(cur)+1+len(word) <= width:

NEW:
		case displayWidth(cur)+1+displayWidth(word) <= width:
```

### Edit E — `ui_test.go` import + runeCol helper + 4 tests (Task 5)

```
OLD:
import (
	"bytes"
	"strings"
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

NEW:
import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/dabstractor/skpp/internal/discover"
)
```

Then APPEND, immediately after the existing `colOf` helper (so the two column
helpers sit together), the rune-based counterpart:

```
// runeCol returns the RUNE column (display column under the width-1-rune model)
// of substr's first occurrence within its line in out. Unlike colOf (byte offset),
// this counts runes so it reflects VISUAL alignment for multi-byte cells: byte
// padding yields uniform byte widths even when the display is misaligned, so a
// byte check is blind to the very bug displayWidth fixes (verified_facts §4).
func runeCol(out, substr string) int {
	idx := strings.Index(out, substr)
	if idx < 0 {
		return -1
	}
	lineStart := strings.LastIndex(out[:idx], "\n") + 1
	return utf8.RuneCountInString(out[lineStart:idx])
}
```

Then APPEND the four new tests at the END of the file:

```go
// TestDisplayWidth: rune count is the display-width approximation (not byte len).
func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"café", 4},  // 5 bytes, 4 runes
		{"—", 1},     // 3 bytes, 1 rune
		{"a—b", 3},   // 5 bytes, 3 runes
		{"ascii", 5}, // byte == rune for ASCII
		{"", 0},
	}
	for _, c := range cases {
		if got := displayWidth(c.in); got != c.want {
			t.Errorf("displayWidth(%q)=%d; want %d (bytes=%d)", c.in, got, c.want, len(c.in))
		}
	}
}

// TestPadRightMultibyte: padding is measured in RUNES, so a multi-byte string is
// padded by (n - runeCount) spaces. With the byte-length bug, padRight("café",5)
// returns "café" (0 spaces) because len("café")==5>=5.
func TestPadRightMultibyte(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"café", 5, "café "},  // 1 space (5 - 4 runes)
		{"éé", 4, "éé  "},     // 2 spaces (4 - 2 runes)
		{"ascii", 3, "ascii"}, // already wider -> no truncation
		{"", 3, "   "},        // empty -> all padding
	}
	for _, c := range cases {
		if got := padRight(c.s, c.n); got != c.want {
			t.Errorf("padRight(%q,%d)=%q; want %q", c.s, c.n, got, c.want)
		}
	}
}

// TestWrapWordsMultibyte: wrapping measures RUNE width, so "café bar" fits in 8
// columns (4+1+3=8). With the byte-length bug, len("café")==5 makes 5+1+3=9>8 and
// it wrongly breaks into two lines.
func TestWrapWordsMultibyte(t *testing.T) {
	lines := wrapWords("café bar", 8)
	if len(lines) != 1 || lines[0] != "café bar" {
		t.Errorf("wrapWords(\"café bar\",8)=%v; want [\"café bar\"] (1 line, rune width)", lines)
	}
}

// TestPrintListColumnsAlignedForMultibyte: a multi-byte TAG (café: 4 runes/5
// bytes) and a multi-byte DESCRIPTION (—) must NOT shift the row's columns. Every
// description starts at the same DISPLAY column as the DESCRIPTION header.
//
// NOTE: this uses runeCol (rune offset), NOT the existing colOf (byte offset) —
// byte offsets are uniform under byte-padding (blind to the bug) and actually
// DIFFER under rune-padding (a rune-padded café cell is 6 bytes vs ascii's 5), so
// a byte check would pass with the bug and fail after the fix. See verified_facts §4.
func TestPrintListColumnsAlignedForMultibyte(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{
		mk("café", "cafe-name", "café — skill", true),
		mk("ascii", "ascii-name", "ascii skill", true),
	}, false)
	out := buf.String()

	descCol := runeCol(out, "DESCRIPTION")
	if descCol < 0 {
		t.Fatalf("no DESCRIPTION header:\n%s", out)
	}
	// Every description starts at the same DISPLAY column as the header. With the
	// byte-width bug, the café row's description lands one column early.
	for _, want := range []string{"café — skill", "ascii skill"} {
		if c := runeCol(out, want); c != descCol {
			t.Errorf("desc %q at display col %d; want %d (aligned under DESCRIPTION):\n%s", want, c, descCol, out)
		}
	}
	// The NAME column is likewise aligned (and the multi-byte tag did not shift it).
	nameCol := runeCol(out, "NAME")
	if c := runeCol(out, "cafe-name"); c != nameCol {
		t.Errorf("'cafe-name' at display col %d; want %d:\n%s", c, nameCol, out)
	}
}
```

### Implementation Patterns & Key Details

```go
// PATTERN: one stdlib helper, applied wherever RENDERED width matters.
//   func displayWidth(s string) int { return utf8.RuneCountInString(s) }
//   // column sizing:  if displayWidth(s.RelTag) > tagW { tagW = displayWidth(s.RelTag) }
//   // padding:        if displayWidth(s) >= n { return s }; return s + strings.Repeat(" ", n-displayWidth(s))
//   // wrapping:       case displayWidth(cur)+1+displayWidth(word) <= width:
// WHY: rune count is the right unit for "how many cells does this occupy" for the
//      common case (1 rune == 1 cell). len(s) is byte count, which over-counts
//      multi-byte runes and under-pads. ASCII is unaffected (byte==rune).

// PATTERN: test with a RUNE-based column check, not a byte-based one.
//   func runeCol(out, substr string) int { /* utf8.RuneCountInString of line prefix */ }
//   if runeCol(out, "café skill") != runeCol(out, "DESCRIPTION") { t.Error(...) }
// WHY: byte offsets are uniform under byte-padding (blind to the bug) and differ
//      under rune-padding (café cell 6 bytes vs ascii 5). Only rune offset tracks
//      visual alignment. The unit tests (displayWidth/padRight/wrapWords) are the
//      most precise guards; runeCol adds end-to-end confidence.

// PATTERN: keep the paint-before-color invariant.
//   paint := func(code, s string) string { ... }
//   padRight(plainTag, tagW)   // BEFORE paint — displayWidth never sees escapes
//   paint(ansiCyan, paddedTag)
// WHY: ANSI escape bytes (e.g. "\x1b[36m") would corrupt width math if measured.
//      The fix changes the math (len->displayWidth), not the order.
```

### Integration Points

```yaml
NO NEW INTEGRATION POINTS:
  - No new types, no new exports beyond displayWidth (lowercase, package-private),
    no new flag, no exit-code change, no stdout/stderr contract change, no API change.
  - PrintList's signature and behavior are unchanged for ASCII; multi-byte content
    now aligns. main's --list/--search dispatch is unchanged.
  - check.go, discover, resolve, search, skillsdir, main — ALL UNCHANGED.
  - go.mod/go.sum UNCHANGED (unicode/utf8 is stdlib).

PARALLEL-SAFETY (vs P1.M2.T1.S1, running concurrently):
  - Files touched by THIS subtask: internal/ui/ui.go, internal/ui/ui_test.go.
  - Files touched by P1.M2.T1.S1: internal/search/search.go,
    internal/search/search_test.go, main.go.
  - ZERO overlap. Both apply cleanly regardless of landing order.
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate)

```bash
cd /home/dustin/projects/skpp

# Format the touched files, then assert the whole tree is gofmt-clean.
gofmt -w internal/ui/ui.go internal/ui/ui_test.go
test -z "$(gofmt -l .)" || { echo "FAIL: gofmt reports unformatted files: $(gofmt -l .)"; exit 1; }

# Compile (catches any typo, e.g. a stray len left in).
go build ./... || { echo "FAIL: go build"; exit 1; }

# Static checks.
go vet ./... || { echo "FAIL: go vet"; exit 1; }
echo "Level 1 PASS"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# The ui tests specifically (verbose). Must include the 4 NEW tests AND the
# unchanged ASCII tests (TestPrintListColumnsAlignedAcrossRows, ...WrapsLongDescription).
go test ./internal/ui/ -v \
  -run 'TestDisplayWidth|TestPadRightMultibyte|TestWrapWordsMultibyte|TestPrintListColumnsAlignedForMultibyte|TestPrintListColumnsAlignedAcrossRows|TestPrintListWrapsLongDescription|TestPrintListSingleNoColor' \
  || { echo "FAIL: targeted ui tests"; exit 1; }

# Full ui package (regression: nothing else broke).
go test ./internal/ui/ -v || { echo "FAIL: go test ./internal/ui/"; exit 1; }

# Whole module (regression guard across search/resolve/check/discover/main —
# especially important since P1.M2.T1.S1 may have landed search changes).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Integration smoke test (the bug-report café snippet, now aligned)

```bash
cd /home/dustin/projects/skpp

# Build the binary.
go build -o skpp . || { echo "FAIL: build"; exit 1; }

# Reproduce the EXACT bug-report scenario (Issue 2): a café tag + an ascii tag.
U="$(mktemp -d)"
mkdir -p "$U/skills/café" "$U/skills/ascii"
printf -- '---\nname: caf\ndescription: café skill\n---\n\n# x\n' > "$U/skills/café/SKILL.md"
printf -- '---\nname: ascii\ndescription: ascii skill\n---\n\n# x\n' > "$U/skills/ascii/SKILL.md"

OUT="$(SKPP_SKILLS_DIR="$U/skills" ./skpp --list)"
RC=$?
test "$RC" = 0 || { echo "FAIL: --list exit=$RC; want 0"; rm -rf "$U" skpp; exit 1; }

# After the fix, the two description columns start at the same DISPLAY column.
# Measure via awk: for the two data lines, the column where "café skill" vs
# "ascii skill" begins must match. (Both tags are 4-5 runes; the bug shifts café
# one column left.) Use a rune-aware check: count leading columns of each data row.
cafe_col=$(printf '%s' "$OUT" | sed -n 's/^café .* \(café skill\)$/x/p' >/dev/null; printf '%s' "$OUT" | awk '/^café /{print index($0,"café skill")}')
ascii_col=$(printf '%s' "$OUT" | awk '/^ascii /{print index($0,"ascii skill")}')
echo "café desc at byte-col $cafe_col; ascii desc at byte-col $ascii_col"

# NOTE on the byte-vs-rune subtlety (verified_facts §4): awk's index() is
# BYTE-based, so with the FIX the café row's desc lands ONE BYTE LATER than
# ascii's (the café tag cell is 6 bytes vs ascii's 5, both padded to 5 RUNES).
# The CORRECT post-fix invariant is therefore: café_col == ascii_col + 1 (byte),
# which means they are DISPLAY-aligned (both at rune column N). Assert exactly
# that — it proves the rune-padding ran (a byte-padded café would give café_col
# == ascii_col, i.e. display-misaligned).
test -n "$cafe_col" -a -n "$ascii_col" || { echo "FAIL: could not locate descriptions:\n$OUT"; rm -rf "$U" skpp; exit 1; }
test "$cafe_col" -eq $((ascii_col + 1)) \
  || { echo "FAIL: café desc byte-col=$cafe_col, want $((ascii_col + 1)) (display-aligned post-fix):\n$OUT"; rm -rf "$U" skpp; exit 1; }
echo "café/ascii rows DISPLAY-aligned (byte-col differs by exactly 1 == the é extra byte) PASS"

# Regression: ASCII-only output is byte-identical to before (rune==byte for ASCII).
A2="$(mktemp -d)"
mkdir -p "$A2/skills/example"
printf -- '---\nname: example\ndescription: A demo skill.\n---\n\n# x\n' > "$A2/skills/example/SKILL.md"
SKPP_SKILLS_DIR="$A2/skills" ./skpp --list | grep -q 'example' \
  || { echo "FAIL: ASCII regression"; rm -rf "$U" "$A2" skpp; exit 1; }
echo "ASCII regression PASS"

rm -rf "$U" "$A2" skpp
echo "Level 3 PASS"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# displayWidth helper exists and uses utf8.RuneCountInString (stdlib, no dep).
grep -q 'func displayWidth(s string) int' internal/ui/ui.go || { echo "FAIL: displayWidth missing"; exit 1; }
grep -q 'utf8.RuneCountInString' internal/ui/ui.go || { echo "FAIL: must use utf8.RuneCountInString"; exit 1; }
grep -q '"unicode/utf8"' internal/ui/ui.go || { echo "FAIL: unicode/utf8 import missing"; exit 1; }

# The 3 display-width sites use displayWidth (not len).
grep -q 'displayWidth(s.RelTag) > tagW' internal/ui/ui.go || { echo "FAIL: column loop RelTag"; exit 1; }
grep -q 'displayWidth(name) > nameW' internal/ui/ui.go || { echo "FAIL: column loop name"; exit 1; }
grep -q 'if displayWidth(s) >= n' internal/ui/ui.go || { echo "FAIL: padRight guard"; exit 1; }
grep -q 'n-displayWidth(s)' internal/ui/ui.go || { echo "FAIL: padRight Repeat arg"; exit 1; }
grep -q 'displayWidth(cur)+1+displayWidth(word) <= width' internal/ui/ui.go || { echo "FAIL: wrapWords"; exit 1; }

# The false ASCII comment is gone (no "Operates on byte length" / "tags are relative dir paths").
! grep -q 'Operates on byte length' internal/ui/ui.go || { echo "FAIL: stale byte-length comment"; exit 1; }
! grep -q 'tags are relative dir paths of the same' internal/ui/ui.go || { echo "FAIL: stale ASCII-tag comment"; exit 1; }
# The CJK limitation is documented (decisions.md §D2 requires it).
grep -qi 'CJK\|East-Asian width\|display width 2\|two cells' internal/ui/ui.go || { echo "FAIL: CJK limitation not documented"; exit 1; }

# NO third-party width library added (PRD §4/§7.3).
! grep -q 'golang.org/x/text' internal/ui/ui.go || { echo "FAIL: x/text added (forbidden)"; exit 1; }
! grep -q 'runewidth' internal/ui/ui.go || { echo "FAIL: runewidth added (forbidden)"; exit 1; }

# Slice-length len() calls are PRESERVED (must not have been "fixed").
grep -q 'if len(skills) == 0' internal/ui/ui.go || { echo "FAIL: len(skills) slice check removed"; exit 1; }
grep -q 'if len(words) == 0' internal/ui/ui.go || { echo "FAIL: len(words) slice check removed"; exit 1; }

# Tests: runeCol + the 4 new tests present; existing tests intact.
grep -q 'func runeCol(' internal/ui/ui_test.go || { echo "FAIL: runeCol helper missing"; exit 1; }
for tn in TestDisplayWidth TestPadRightMultibyte TestWrapWordsMultibyte TestPrintListColumnsAlignedForMultibyte; do
  grep -q "func $tn" internal/ui/ui_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done
grep -q 'func TestPrintListColumnsAlignedAcrossRows' internal/ui/ui_test.go || { echo "FAIL: ASCII alignment test removed (must stay)"; exit 1; }
# The new alignment test uses runeCol (NOT colOf) for the multi-byte assertion.
grep -q 'runeCol(out, "café — skill")' internal/ui/ui_test.go || { echo "FAIL: must use runeCol for multi-byte alignment"; exit 1; }

# go.mod / go.sum UNCHANGED (stdlib only).
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null || { echo "FAIL: go.mod/go.sum changed (unicode/utf8 is stdlib — no dep allowed)"; git diff go.mod go.sum; exit 1; }

# EXACTLY 2 files changed — nothing else.
git diff --quiet -- internal/search/search.go internal/search/search_test.go 2>/dev/null \
  || { echo "NOTE: search files changed (likely P1.M2.T1.S1 landed — expected, not this subtask's concern)"; }
git diff --quiet -- internal/check/check.go   || { echo "FAIL: check.go changed (out of scope)"; exit 1; }
git diff --quiet -- internal/discover/skill.go || { echo "FAIL: skill.go changed (out of scope)"; exit 1; }
git diff --quiet -- internal/resolve/resolve.go || { echo "FAIL: resolve.go changed (out of scope)"; exit 1; }
git diff --quiet -- README.md                  || { echo "FAIL: README.md changed (deferred to P1.M5.T3)"; exit 1; }
git diff --quiet -- PRD.md                     || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l .` empty, `go build ./...` compiles, `go vet ./...` clean
- [ ] Level 2 PASS — `go test ./internal/ui/ -v` green (4 new + existing ASCII tests); `go test ./...` whole module green
- [ ] Level 3 PASS — bug-report café snippet: café & ascii rows DISPLAY-aligned (byte-col differs by exactly 1 == the extra é byte); ASCII regression intact
- [ ] Level 4 PASS — displayWidth present + uses utf8.RuneCountInString; 3 sites fixed; stale comment gone; CJK limitation documented; no x/text/runewidth; slice `len()` preserved; runeCol + 4 tests present; go.mod/go.sum unchanged; scope respected

### Feature Validation
- [ ] A multi-byte tag (café) no longer shifts its row's NAME/DESCRIPTION columns left
- [ ] A multi-byte description (—, é) wraps and pads by rune width, not byte width
- [ ] ASCII-only tables are byte-identical to before (rune count == byte count)
- [ ] The wide-CJK limitation is documented in the padRight/displayWidth comment
- [ ] No new third-party dependency (unicode/utf8 is stdlib)

### Code Quality Validation
- [ ] `displayWidth` is a single stdlib call, applied consistently at all 3 display-width sites
- [ ] Slice-length `len()` calls (`len(skills)`, `len(words)`) and ASCII header init (`len("TAG")`) are untouched
- [ ] The paint-before-color invariant is preserved (padRight on PLAIN text before paint)
- [ ] Tests use `runeCol` (rune offset) for multi-byte alignment, not the byte-based `colOf`
- [ ] New tests mirror the existing ui_test.go style (white-box, plain t.Errorf, no testify/Parallel)

### Scope Discipline (Mode A)
- [ ] `internal/check/check.go` NOT modified (charset evidence is read-only)
- [ ] `internal/discover/*`, `internal/resolve/*`, `internal/search/*`, `internal/skillsdir/*` NOT modified
- [ ] `main.go` NOT modified (P1.M2.T1.S1 owns it this cycle; zero overlap)
- [ ] `README.md` NOT modified (deferred to P1.M5.T3.S1 Mode B doc sync)
- [ ] `PRD.md` / `tasks.json` / `prd_snapshot.md` NOT modified (read-only / orchestrator-owned)
- [ ] `go.mod` / `go.sum` NOT modified (stdlib only)
- [ ] `git diff --name-only` ⊆ {`internal/ui/ui.go`, `internal/ui/ui_test.go`} (search/main may also appear if P1.M2.T1.S1 landed — expected, separate subtask)

---

## Anti-Patterns to Avoid

- ❌ **Don't use the existing `colOf()` for the multi-byte alignment test.** It
  measures BYTE offset, which is uniform under byte-padding (blind to the bug)
  and differs under rune-padding (café cell 6 bytes vs ascii 5) — so a byte check
  passes with the bug and FAILS after the fix. Use `runeCol` (rune offset).
  Empirically proven (verified_facts §4).
- ❌ **Don't add `golang.org/x/text/width` or `runewidth`.** PRD §4/§7.3 forbid a
  new third-party dependency; decisions.md §D2 mandates stdlib `utf8.RuneCountInString`.
  The wide-CJK imperfection is the accepted, documented trade-off.
- ❌ **Don't change slice-length `len()` calls or the ASCII header init.** Only the
  3 DISPLAY-width sites change. `len(skills)`, `len(words)`, `len(lines)` are slice
  lengths; `len("TAG")`/`len("NAME")` are ASCII literals (byte==rune, out of scope).
- ❌ **Don't claim rune count is perfect.** Wide CJK runes (2 cells) count as 1.
  The comment MUST state this limitation and the §4/§7.3 rationale, or a future
  maintainer will assume perfection or re-add `len()`.
- ❌ **Don't repeat the false "tags are ASCII" claim.** Only the frontmatter `name`
  is ASCII-restricted (check.go `validName` @ :195 on `fm.Name`); RelTag (directory
  name) and Description are NOT. The bug report demonstrates a `café` tag.
- ❌ **Don't break the paint-before-color invariant.** `padRight` runs on PLAIN text
  before `paint()`; `displayWidth` must never see ANSI escape bytes. Change the math,
  not the order.
- ❌ **Don't modify existing ui_test.go tests.** They are ASCII (byte==rune) and
  still pass unchanged. ADD `runeCol` + the 4 new tests; leave the rest. Editing
  `TestPrintListColumnsAlignedAcrossRows` or `TestPrintListWrapsLongDescription` is
  unnecessary risk.
- ❌ **Don't edit `main.go` / `search` / `check` / `discover` / `resolve`.** This is
  a `ui`-only fix. P1.M2.T1.S1 owns main.go/search this cycle — zero overlap, but
  do not touch them here.
- ❌ **Don't change `go.mod`/`go.sum`.** `unicode/utf8` is stdlib. If `go mod tidy`
  changes anything, you added an external import you should not have.
- ❌ **Don't edit `README.md`.** Deferred to P1.M5.T3.S1 (Mode B doc sync). This is
  Mode A: code + in-source docs + tests.

---

## Confidence Score

**10/10** — The change is a 1-line stdlib helper + 3 `len()`→`displayWidth()`
substitutions + a comment rewrite + additive tests. Every expected test value
(`displayWidth("café")==4`, `padRight("café",5)=="café "`, `wrapWords("café bar",8)==
["café bar"]`, and the runeCol alignment result `DESCRIPTION=café=ascii=19` post-fix
vs `café=18` with the bug) was **EXECUTED** in a throwaway Go 1.25 program
(verified_facts §5). The one subtle point — that the existing `colOf` (byte) cannot
detect this bug and a rune-based `runeCol` is required — is empirically proven (§4)
and documented as a gotcha + in the test comment. go.mod-neutral (stdlib only),
zero file overlap with the parallel P1.M2.T1.S1. Residual risk is limited to
transcription typos, caught by the Level 4 grep-based contract checks.
