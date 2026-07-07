# PRP — P1.M4.T9.S1: `internal/search` substring filter + `--search` wiring (ui reuse)

> **Subtask:** P1.M4.T9.S1 — the `skpp --search <q>` / `-s <q>` command (PRD §6.1).
> It is the ONLY subtask of T9 ("`--search`"). It adds a case-insensitive
> **substring** search over the four PRD §6.1 fields (tag, frontmatter `name`,
> `description`, `metadata.keywords`) and renders the filtered results with the
> **same table as `--list`** by reusing `ui.PrintList`.
>
> **Scope:** four files — CREATE `internal/search/search.go` +
> `internal/search/search_test.go`; MODIFY `main.go` (add `searchMode`/`searchQ`
> to `config`, convert `parseArgs` to an index loop + add the `--search`/`-s`
> value-consuming case, add the `run` search branch, add the `internal/search`
> import) and `main_test.go` (16 new tests; existing 53 unchanged).
>
> **DEPENDENCIES (CONTRACT):** P1.M2.T5.S1 (`discover.Index`), P1.M2.T6.S1
> (`ui.PrintList` + `isTerminal`/`--no-color`), P1.M1.T2 (`skillsdir.Find`), and
> P1.M1.T3 (`main.run`/`config`/`parseArgs`) are LANDED and GREEN — all consumed
> verbatim. `ui.PrintList(w, []discover.Skill, useColor)` is the formatter to
> reuse; `discover.Index(dir)` returns the pre-sorted catalog.
>
> **SCOPE DECISION (authoritative — see research/verified_facts.md §2):** The
> filter lives in a new **`internal/search`** package that mirrors
> `internal/resolve` (a pure function over `[]discover.Skill`, own package, own
> `_test.go`). `main.run` stays a thin dispatcher. This subtask owns
> `--search`/`-s` + the `internal/search` package ONLY. It does NOT add `check`
> (T10), `--help` (M5), the §6.3 mutual-exclusivity / exit-2 logic (M5), or
> `skills/example/` (M6). It does NOT touch `internal/discover/*`,
> `internal/skillsdir/*`, `internal/resolve/*`, `internal/ui/*`, `go.mod`,
> `go.sum`, or `PRD.md`, and adds NO third-party dependency (stdlib `strings` only).

---

## Goal

**Feature Goal**: Ship `skpp --search <q>` / `-s <q>` so a user can find skills in
their manifest-free store by a case-insensitive substring over the four PRD §6.1
fields — the canonical tag, the frontmatter `name`, the `description`, and any
`metadata.keywords` entry — and see the matches as the same `TAG`/`NAME`/
`DESCRIPTION` table `--list` prints. This reuses the existing `ui.PrintList`
formatter with a filtered (still-sorted) slice, so it is a thin filter + a second
`PrintList` call, not a re-implementation of the table.

**Deliverable**: Four files (two NEW, two MODIFIED; no other files touched):
1. `internal/search/search.go` — `package search`; `func Search(query string,
   skills []discover.Skill) []discover.Skill` + the unexported `matches` helper.
   Imports only `strings` + `internal/discover`. ~50 lines.
2. `internal/search/search_test.go` — `package search` (white-box); 16 tests
   covering each field, case-insensitivity, empty-query-matches-all, order
   preservation, no-match, no-frontmatter-by-tag, category/aliases excluded,
   keyword-boundary exclusion, multi-field de-dup, nil input.
3. `main.go` — MODIFY (3 localized edits): `config` gains `searchMode`/`searchQ`;
   `parseArgs` becomes an index loop and gains `case "--search","-s"` that consumes
   the next token; `run` gains the `if c.searchMode {…}` branch (after `--list`,
   before `--all`); the import block gains `internal/search`.
4. `main_test.go` — MODIFY (append): 16 new tests (4 `parseArgs` + 12 `run`)
   using the existing `sampleStore`/`writeSkillTree`/`withTerminal`/`unsetSkillsEnv`
   helpers. Existing 53 tests UNCHANGED.

**Success Definition**: `gofmt -l internal/search/*.go main.go main_test.go` is
silent; `go vet ./...` is clean; `go build ./...` and `go test ./...` pass
(**177 tests total**: 145 baseline + 16 new search-package + 16 new main).
`go mod tidy` is a **no-op** (`go.mod`/`go.sum` unchanged — `strings` is stdlib).
`./skpp --search <q>` over a matching store prints the filtered table to stdout,
exit 0; with no matches exits 1 (stderr message, empty stdout); case-insensitive;
`--no-color`/TTY color gating shared with `--list`. No touch to
`internal/{discover,skillsdir,resolve,ui}/*`, `go.mod`/`go.sum`, `PRD.md`; no
`check`, `--help`, exit-2, or `skills/example/`.

## User Persona

**Target User**: A pi operator who keeps skills in a centralized, non-auto-discovered
store and loads them on demand via `pi --skill "$(skpp <tag>)"`.

**Use Case**: "I have a few dozen skills; I remember one is about reddit but not
its exact tag. I run `skpp --search reddit` to see which skill(s) match before I
resolve one with `skpp <tag>`."

**User Journey**: `skpp --search reddit` → table of matching skills (tag/name/desc)
→ user reads the TAG column → `pi --skill "$(skpp writing/reddit)"`.

**Pain Points Addressed**: `--list` shows the whole catalog; `--search` narrows it
to what is relevant, across multiple fields (a match can live in the description
or a keyword, not just the tag), without any index/manifest file.

## Why

- **Discoverability of a growing catalog.** As the store grows past a handful of
  skills, `--list` becomes a wall of text. `--search` lets the user find a skill
  by any remembered fragment — tag, name, description text, or a keyword — because
  the manifest-free design already keeps all of that on disk in `SKILL.md`.
- **Cheap to build, high leverage, zero new surface risk.** The formatter
  (`ui.PrintList`) and the data source (`discover.Index`) already exist and are
  green. This subtask is a pure filter between them, so it cannot regress
  `--list`/`--all`/`<tag>` — it only reads the same `[]discover.Skill`.
- **Locks the §6.1 search contract early.** Field scope (tag/name/description/
  keywords — NOT category/aliases), case-insensitivity, and the "same table as
  `--list`" rendering are now pinned by working code + tests before the
  acceptance sweep (P1.M6.T16) and completions (P1.M6.T15) lean on them.
- **Mirrors `internal/resolve`** — establishes that *every* matching concern over
  `[]discover.Skill` is a pure, self-tested `internal/*` package, keeping `main`
  a thin dispatcher.

## What

User-visible behavior (PRD §6.1):

- `skpp --search <q>` (and `-s <q>`): prints a `TAG`/`NAME`/`DESCRIPTION` table
  of every skill where `<q>` is a **case-insensitive substring** of the skill's
  tag, frontmatter `name`, `description`, or any `metadata.keywords` entry.
  Output format is **identical to `--list`** (same columns, wrapping, color rules,
  `(none)`/`(missing)` placeholders).
- Exit `0` when ≥1 skill matches; exit `1` when nothing matches (one-line message
  to stderr, nothing on stdout).
- `--no-color` suppresses ANSI even on a TTY; color is on by default when stdout
  is a TTY (same gate as `--list`).
- `--file`/`--relative` do **not** apply (search prints a table, not paths).

### Success Criteria

- [ ] `--search <q>` / `-s <q>` both work; the value token is consumed (not
      captured as a tag).
- [ ] Matches by tag, by name, by description, and by keyword are all returned.
- [ ] Matching is case-insensitive (`REDDIT` matches `reddit`).
- [ ] Field scope is **exactly** tag/name/description/keywords; `Category` and
      `Aliases` are NOT searched (negative test).
- [ ] Keywords are matched **individually** (a query spanning two keywords'
      boundary does not match — negative test).
- [ ] Order is preserved (filtered slice stays sorted by RelTag).
- [ ] Empty query `""` matches all skills (exit 0 unless store empty).
- [ ] No matches → exit 1, empty stdout, `no skills matched <q>` on stderr.
- [ ] Skills dir unresolvable → exit 1, empty stdout, the one-line fix on stderr.
- [ ] `--no-color` suppresses ANSI; TTY enables ANSI (shared with `--list`).
- [ ] `--version` still precedes `--search` (PRD §6.3).
- [ ] `go test ./...` green (177 tests); `gofmt`/`go vet` clean; `go.mod` unchanged.

## All Needed Context

### Context Completeness Check

_If someone knew nothing about this codebase, would they have everything needed to
implement this successfully?_ **Yes.** The four files are specified verbatim below
(two full new files, three exact edit snippets for `main.go`, sixteen named test
functions for `main_test.go`). The reusable formatter's signature, the data type's
fields, the dispatch order, and every semantics decision are pinned in the
research notes and the Blueprint. No external library is involved.

### Documentation & References

```yaml
# MUST READ — the authoritative spec for this command (one row of the table)
- file: PRD.md
  section: "§6.1 (the --search row) and §6.2 (--no-color modifier)"
  why: "Defines field scope (tag/name/description/keywords), case-insensitive
        substring, 'same table format as --list', and exit codes (0 on matches,
        1 if no matches)."
  critical: "Field scope is EXACTLY those four. Category and Aliases are NOT in
             the list even though they exist on discover.Skill."

# MUST READ — the formatter being reused (do NOT modify it)
- file: internal/ui/ui.go
  why: "Search calls PrintList with a FILTERED slice. Its signature, 'does not
        re-sort' contract, empty-slice-prints-nothing behavior, and color param."
  pattern: "ui.PrintList(w io.Writer, skills []discover.Skill, useColor bool)"
  gotcha: "PrintList prints NOTHING on an empty slice and does NOT decide the exit
           code — main must test len(matched)==0 and exit 1 BEFORE calling it."

# MUST READ — the data type + the pre-sorted source
- file: internal/discover/skill.go
  why: "The Skill struct fields available for matching: RelTag, Name, Description,
        Keywords (the four §6.1 fields) vs Category/Aliases (excluded)."
  pattern: "discover.Skill{ RelTag, Name, Description string; Keywords []string; ... }"
  gotcha: "Keywords is nil when absent — test with len(), not a nil check. A
           no-frontmatter skill (HasFM=false) has empty Name/Description/nil
           Keywords but a non-empty RelTag, so it is still matchable by tag."
- file: internal/discover/index.go
  why: "Index(dir) returns []Skill SORTED by RelTag. The filter iterates in order,
        so the filtered slice is already sorted — no re-sort anywhere."
  pattern: "func Index(skillsDir string) ([]Skill, error)  // sorted by RelTag"

# MUST READ — the direct precedent: a pure matching function in its own package
- file: internal/resolve/resolve.go
  why: "Search mirrors this exactly: a PURE function over []discover.Skill, in its
        own internal/ package with its own _test.go, called by main. Same shape,
        smaller scope (substring, not precedence)."
  pattern: "package resolve; func Resolve(tag, skills) (Result, error)  // pure"

# MUST READ — the three exact insertion points in the dispatcher
- file: main.go
  why: "config struct (add fields), parseArgs (range→index loop + new case), run
        dispatch (insert search branch after the --list block), import block."
  pattern: "run() order: version → path → list → [SEARCH HERE] → all → tags → default"
  gotcha: "parseArgs is currently `for _, a := range args` — a value flag (--search
           <q>) must CONSUME the next token, so convert to `for i:=0; i<len; i++`
           with an i++ skip. This is the ONLY structural parser change."

# Reference (stdlib; stable since Go 1.0 — no version concern)
- url: https://pkg.go.dev/strings#Contains
  why: "strings.Contains(haystack, needle) + strings.ToLower = case-insensitive
        substring. Lowercase the query ONCE, each field lazily."
```

### Current Codebase tree (relevant slice)

```bash
skpp/
├── go.mod                      # module github.com/dabstractor/skpp; go 1.25; dep yaml.v3 (UNCHANGED)
├── main.go                     # MODIFY: config + parseArgs + run + import
├── main_test.go                # MODIFY: append 16 tests (existing 53 unchanged)
└── internal/
    ├── discover/{skill.go,index.go,discover.go,*_test.go}   # READ-ONLY (provides Skill, Index)
    ├── resolve/{resolve.go,resolve_test.go}                 # READ-ONLY (the pattern to mirror)
    ├── skillsdir/{skillsdir.go,*_test.go}                   # READ-ONLY (provides Find)
    ├── ui/{ui.go,ui_test.go}                                # READ-ONLY (provides PrintList)
    └── search/                                              # NEW PACKAGE (this subtask)
        ├── search.go                                        # CREATE
        └── search_test.go                                   # CREATE
```

### Desired Codebase tree with files to be added/modified

```bash
internal/search/search.go        # NEW — package search; Search() + matches(); pure, stdlib-only
internal/search/search_test.go   # NEW — 16 table tests (white-box, no filesystem)
main.go                          # MODIFY — 3 localized edits (config / parseArgs / run) + 1 import
main_test.go                     # MODIFY — append 16 tests (4 parseArgs + 12 run)
```

### Known Gotchas of our codebase & Library Quirks

```go
// CRITICAL: parseArgs is a RANGE loop today. --search <q> is a VALUE flag that
// must grab args[i+1] and NOT let it fall through to the tag-capture default.
// Fix: make the loop index-based (`for i := 0; i < len(args); i++`) and `i++`
// after consuming the value. Every existing `case` stays byte-identical.

// CRITICAL: field scope is EXACTLY tag/name/description/keywords (PRD §6.1).
// Category and Aliases sit on discover.Skill right next to Keywords — do NOT
// sweep them in. There is a dedicated negative test for this.

// CRITICAL: keywords are matched INDIVIDUALLY, never strings.Join'd. A query
// spanning a boundary between two keywords must NOT match (negative test).

// GOTCHA: ui.PrintList prints NOTHING on an empty slice and does not exit. main
// MUST test len(matched)==0 → stderr msg + exit 1 BEFORE calling PrintList.

// GOTCHA: lowercasing. Lowercase the QUERY ONCE (strings.ToLower(query)) and
// reuse it; lowercase each field lazily inside Contains. Do not lowercase the
// whole Skill (would allocate + is unnecessary).

// GOTCHA: empty query "" matches ALL skills (strings.Contains(x,"") is always
// true). This is the defined behavior (PRD carves out no special case); exit 0
// unless the store itself is empty.

// GOTCHA: --search as the LAST token (no value) must NOT activate search mode
// (searchMode stays false → default exit 1). The proper "flag requires an
// argument" exit-2 is P1.M5.T11 — do not implement exit-2 here.

// GOTCHA: --file/--relative do NOT apply to --search (it prints a TABLE, like
// --list). Only --no-color / TTY-color apply (shared with --list).

// NO new third-party dependency: stdlib `strings` only. `go mod tidy` is a no-op.
```

## Implementation Blueprint

### Data models and structure

No new data models. The feature operates on the existing `discover.Skill`
(READ-ONLY) and reuses `ui.PrintList` (READ-ONLY). The only new type surface is
the `Search` function signature.

### File 1 — CREATE `internal/search/search.go` (full content)

```go
// Package search filters a []discover.Skill by a case-insensitive substring
// query over the four fields PRD §6.1 names for `skpp --search`: the tag, the
// frontmatter name, the description, and each metadata keyword. It is a PURE
// function over []discover.Skill: no filesystem, no globals, no I/O — main
// (P1.M4.T9.S1) supplies the index from discover.Index and passes the filtered
// (still-sorted) slice to ui.PrintList for the "same table format as --list"
// rendering (PRD §6.1).
//
// It mirrors internal/resolve (also a pure matching function over
// []discover.Skill, in its own package with its own _test.go) so the two matching
// concerns — precise tag resolution (resolve) and fuzzy catalog search (search) —
// stay isolated, independently unit-testable, and out of the thin main dispatcher.
package search

import (
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

// Search returns every skill in skills for which query is a case-insensitive
// substring of ANY of the four PRD §6.1 fields: RelTag (the tag), Name
// (frontmatter name), Description, or any element of Keywords. Input order is
// preserved: discover.Index sorts []Skill by RelTag, and ui.PrintList does NOT
// re-sort, so the filtered slice is displayed already-sorted.
//
// An empty query matches EVERY skill: strings.Contains(hay, "") is always true,
// so `skpp --search ""` behaves like `skpp --list` (exit 1 only if the store is
// empty). This is the natural substring semantics; the PRD carves out no special
// case for an empty query.
//
// A skill with no frontmatter (HasFM==false) has Name=="" and Description=="" and
// nil Keywords, but its RelTag is always present — so it is still discoverable by
// searching its tag, matching how resolve lets a frontmatter-less skill resolve
// by directory/basename (PRD §7.1). Only RelTag is searchable for such a skill.
func Search(query string, skills []discover.Skill) []discover.Skill {
	q := strings.ToLower(query) // lowercase the query ONCE, not per field
	var matched []discover.Skill
	for _, s := range skills {
		if matches(q, s) {
			matched = append(matched, s)
		}
	}
	return matched
}

// matches reports whether the already-lowercased query q is a case-insensitive
// substring of any searchable field of s. q is lowercased once by the caller
// (Search); each field is lowercased lazily inside Contains.
//
// Field scope is EXACTLY the four PRD §6.1 fields. It deliberately does NOT
// include Category or Aliases — both exist on discover.Skill and would be a
// tempting (but spec-violating) addition. PRD §6.1: "tag, frontmatter name,
// description, and metadata.keywords".
//
// Keywords are tested INDIVIDUALLY (not strings.Join'd): a query spanning a
// boundary between two keywords must not match (joining would create false
// positives like "wri"+"ocial" => "wriocial" across "writing","social").
func matches(q string, s discover.Skill) bool {
	if strings.Contains(strings.ToLower(s.RelTag), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Description), q) {
		return true
	}
	for _, kw := range s.Keywords {
		if strings.Contains(strings.ToLower(kw), q) {
			return true
		}
	}
	return false
}
```

### File 2 — CREATE `internal/search/search_test.go` (full content)

```go
package search

import (
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

// sk builds one discover.Skill with the searchable fields set (HasFM true so all
// fields are "real"). Mirrors ui_test.go's mk but lets keywords be set. Use len()
// on Keywords (never a nil check) per the discover contract.
func sk(tag, name, desc string, keywords ...string) discover.Skill {
	return discover.Skill{
		RelTag:      tag,
		Name:        name,
		Description: desc,
		Keywords:    keywords,
		HasFM:       true,
	}
}

func TestSearchMatchByTag(t *testing.T) {
	in := []discover.Skill{sk("writing/reddit", "rp", "d")}
	out := Search("writing/reddit", in)
	if len(out) != 1 || out[0].RelTag != "writing/reddit" {
		t.Errorf("exact tag match failed: got %+v", out)
	}
}

func TestSearchMatchByTagSubstring(t *testing.T) {
	in := []discover.Skill{sk("writing/reddit", "rp", "d")}
	out := Search("redd", in)
	if len(out) != 1 {
		t.Errorf("tag substring 'redd' should match writing/reddit: got %+v", out)
	}
}

func TestSearchMatchByBasenameAsSubstring(t *testing.T) {
	in := []discover.Skill{sk("writing/reddit", "rp", "d")}
	out := Search("reddit", in) // basename is part of the relTag string
	if len(out) != 1 {
		t.Errorf("'reddit' substring of relTag should match: got %+v", out)
	}
}

func TestSearchMatchByName(t *testing.T) {
	in := []discover.Skill{sk("a", "reddit-poster", "d")}
	out := Search("poster", in)
	if len(out) != 1 {
		t.Errorf("name substring 'poster' should match: got %+v", out)
	}
}

func TestSearchMatchByDescription(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "Posts messages to social media")}
	out := Search("social", in)
	if len(out) != 1 {
		t.Errorf("description substring 'social' should match: got %+v", out)
	}
}

func TestSearchMatchByKeyword(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "d", "writing", "social")}
	out := Search("soc", in)
	if len(out) != 1 {
		t.Errorf("keyword substring 'soc' should match keyword 'social': got %+v", out)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	in := []discover.Skill{sk("Reddit", "Name", "Desc")}
	for _, q := range []string{"reddit", "REDDIT", "rEdDiT", "name", "DESC"} {
		if out := Search(q, in); len(out) != 1 {
			t.Errorf("case-insensitive query %q should match; got %+v", q, out)
		}
	}
}

func TestSearchNoMatchReturnsEmpty(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "d", "k")}
	if out := Search("zzznotfound", in); len(out) != 0 {
		t.Errorf("no-match query should return empty slice; got %+v", out)
	}
}

func TestSearchEmptyQueryMatchesAll(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "d"), sk("b", "m", "e")}
	if out := Search("", in); len(out) != 2 {
		t.Errorf("empty query should match all; got %d", len(out))
	}
}

func TestSearchPreservesInputOrder(t *testing.T) {
	in := []discover.Skill{
		sk("zebra", "n", "match"),
		sk("apple", "n", "match"),
		sk("mango", "n", "nomatch"),
	}
	out := Search("match", in) // matches zebra + apple by description, in that order
	if len(out) != 2 || out[0].RelTag != "zebra" || out[1].RelTag != "apple" {
		t.Errorf("order not preserved: got %+v", out)
	}
}

func TestSearchMultipleMatchesAllReturned(t *testing.T) {
	in := []discover.Skill{
		sk("a", "x", "common"),
		sk("b", "x", "unrelated"),
		sk("c", "common", "y"),
	}
	out := Search("common", in)
	if len(out) != 2 {
		t.Errorf("expected 2 matches across desc+name; got %d: %+v", len(out), out)
	}
}

func TestSearchNoFrontmatterStillMatchesByTag(t *testing.T) {
	// HasFM false => Name/Description empty, Keywords nil, but RelTag present.
	in := []discover.Skill{{RelTag: "bare-skill", HasFM: false}}
	out := Search("bare", in)
	if len(out) != 1 {
		t.Errorf("frontmatter-less skill must still match by tag; got %+v", out)
	}
}

func TestSearchDoesNotMatchCategoryOrAliases(t *testing.T) {
	// PRD §6.1 scopes search to tag/name/description/keywords ONLY. Category and
	// Aliases are on the struct but must NOT be searched.
	in := []discover.Skill{
		{RelTag: "x", Name: "n", Description: "d", Category: "secret-cat", Aliases: []string{"secret-alias"}, HasFM: true},
	}
	if out := Search("secret", in); len(out) != 0 {
		t.Errorf("search must NOT match category/aliases (PRD §6.1 scope); got %+v", out)
	}
}

func TestSearchKeywordSubstringNotJoinBoundary(t *testing.T) {
	// Keywords are matched INDIVIDUALLY, not joined — so a query spanning a
	// boundary between two keywords must NOT match. "wriocial" is not a substring
	// of "writing" nor of "social" individually.
	in := []discover.Skill{sk("a", "n", "d", "writing", "social")}
	if out := Search("wriocial", in); len(out) != 0 {
		t.Errorf("keyword-boundary query must not match (keywords searched individually): got %+v", out)
	}
}

func TestSearchNilInput(t *testing.T) {
	if out := Search("x", nil); len(out) != 0 {
		t.Errorf("nil input should yield empty; got %+v", out)
	}
}

func TestSearchReturnsDistinctResults(t *testing.T) {
	// A skill matching in MULTIPLE fields (e.g. tag AND description) is returned
	// ONCE, not duplicated.
	in := []discover.Skill{sk("match", "n", "match")}
	out := Search("match", in)
	if len(out) != 1 {
		t.Errorf("multi-field match should return the skill once; got %d", len(out))
	}
}
```

### File 3 — MODIFY `main.go` (three localized edits + one import)

The file is large and otherwise unchanged; apply these EXACT edits. Run
`gofmt -w main.go` after (it will realign the `config` struct comments — that is
expected and correct).

**Edit 3a — imports** (add `internal/search` in alphabetical order):

```go
// OLD:
	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"

// NEW:
	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
	"github.com/dabstractor/skpp/internal/search"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
```

**Edit 3b — `config` struct** (add the two search fields; drop `search` from the
"future" comment):

```go
// OLD:
	noColor  bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	tags     []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run
	// Future (M4/M5), do NOT add yet:
	//   search string; check bool; help bool
}

// NEW (gofmt realigns the whole struct block; the field set is what matters):
	noColor    bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	tags       []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run
	searchMode bool     // --search <q>/-s : substring search over tag/name/description/keywords (§6.1) [NEW]
	searchQ    string   // the --search query value (consumed from the token after --search/-s) [NEW]
	// Future (M5), do NOT add yet:
	//   check bool; help bool
}
```

**Edit 3c — `parseArgs`** (convert the range loop to an index loop and add the
value-consuming `--search` case; every other `case` is unchanged):

```go
// OLD:
func parseArgs(args []string) config {
	var c config
	for _, a := range args {
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--all", "-a":
			c.all = true
		case "--file", "-f":
			c.file = true
		case "--relative":
			c.relative = true
		case "--no-color":
			c.noColor = true
		default:
			// Positional <tag> ... (unchanged comment)
			if !strings.HasPrefix(a, "-") {
				c.tags = append(c.tags, a)
			}
		}
	}
	return c
}

// NEW:
func parseArgs(args []string) config {
	var c config
	// Index-based loop (not range) so a value-taking flag (--search <q>) can
	// CONSUME the following token via i++ without it also being captured as a tag.
	// PRD §6.1/§6.2: --search/-s take exactly one value; every other flag is a bool.
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--all", "-a":
			c.all = true
		case "--file", "-f":
			c.file = true
		case "--relative":
			c.relative = true
		case "--no-color":
			c.noColor = true
		case "--search", "-s":
			// Value-taking flag: consume the NEXT token verbatim as the query. The
			// value is NOT appended to c.tags (i++ skips it). If --search is the
			// LAST token (no value follows) searchMode stays false and the call
			// falls through to the no-recognized-mode default (exit 1); the proper
			// "flag requires an argument" exit-2 is P1.M5.T11. A value starting with
			// '-' (e.g. `--search -x`) is grabbed as the literal query "-x".
			if i+1 < len(args) {
				c.searchMode = true
				c.searchQ = args[i+1]
				i++
			}
		default:
			// Positional <tag> ... (unchanged comment)
			if !strings.HasPrefix(a, "-") {
				c.tags = append(c.tags, a)
			}
		}
	}
	return c
}
```

> Keep the existing `default:`-branch comment verbatim (the one mentioning
> `--frobnicate` / P1.M5.T11 / §6.3 mutual-exclusivity). It already names `--search`.

**Edit 3d — `run` dispatch** (insert the search branch AFTER the `if c.list {…}`
block and BEFORE the `// --all mode:` block):

```go
// INSERT THIS BLOCK between the --list block and the --all block:

	// --search mode: `skpp --search <q>` / `-s <q>` (PRD §6.1). Filters the index to
	// skills where <q> is a case-insensitive substring of the tag, frontmatter name,
	// description, or any metadata keyword (internal/search), then renders the SAME
	// table as --list via ui.PrintList (PRD §6.1: "same table format as --list,
	// filtered"). The filtered slice keeps discover.Index's RelTag sort. Exit 0 with
	// the table on matches; exit 1 (stderr message, EMPTY stdout) when nothing
	// matches (PRD §6.1: "1 if no matches"). --no-color / TTY color gating is shared
	// with --list; --file/--relative do NOT apply (search prints a TABLE, not paths —
	// PRD §6.2: modifiers combine with tag/--all only).
	if c.searchMode {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		matched := search.Search(c.searchQ, skills)
		if len(matched) == 0 {
			// PRD §6.1: exit 1 "if no matches". Mirror --list's "no skills found"
			// convention: message to stderr, stdout stays clean.
			fmt.Fprintln(stderr, "no skills matched "+c.searchQ)
			return 1
		}
		ui.PrintList(stdout, matched, isTerminal(stdout) && !c.noColor)
		return 0
	}
```

### File 4 — MODIFY `main_test.go` (append these 16 tests; existing 53 unchanged)

These reuse the existing `sampleStore`, `writeSkillTree`, `withTerminal`, and
`unsetSkillsEnv` helpers (already defined in `main_test.go`). Append at end of file.

```go
// --- parseArgs: --search/-s value flag (P1.M4.T9.S1) ---

// --search <q> sets searchMode=true and captures the query; the value is NOT a tag.
func TestParseArgsSearchLong(t *testing.T) {
	c := parseArgs([]string{"--search", "reddit"})
	if !c.searchMode || c.searchQ != "reddit" {
		t.Errorf("parseArgs(--search reddit): mode=%v q=%q; want true,reddit", c.searchMode, c.searchQ)
	}
	if len(c.tags) != 0 {
		t.Errorf("--search value leaked into tags: %v", c.tags)
	}
}

// -s <q> short form behaves identically.
func TestParseArgsSearchShort(t *testing.T) {
	c := parseArgs([]string{"-s", "reddit"})
	if !c.searchMode || c.searchQ != "reddit" {
		t.Errorf("parseArgs(-s reddit): mode=%v q=%q; want true,reddit", c.searchMode, c.searchQ)
	}
}

// --search with NO following value (last token) -> searchMode stays false; falls
// to the default exit-1 path. Proper exit-2 "needs an argument" is P1.M5.T11.
func TestParseArgsSearchNoValueStaysInactive(t *testing.T) {
	c := parseArgs([]string{"--search"})
	if c.searchMode {
		t.Errorf("parseArgs(--search) with no value: searchMode=true; want false (no value consumed)")
	}
}

// --search consumes exactly ONE value; a following positional is captured as a tag.
// (Mixing search mode + a tag is an M5.T11 exclusivity error; for now searchMode
// wins in run() dispatch and tags are ignored.)
func TestParseArgsSearchConsumesOneValue(t *testing.T) {
	c := parseArgs([]string{"--search", "q", "sometag"})
	if !c.searchMode || c.searchQ != "q" {
		t.Errorf("search not captured: mode=%v q=%q", c.searchMode, c.searchQ)
	}
	if len(c.tags) != 1 || c.tags[0] != "sometag" {
		t.Errorf("tags=%v; want [sometag] (the token after the search value)", c.tags)
	}
}

// --- run: --search / -s (P1.M4.T9.S1) ---

// --search success: a query matching a skill's tag prints the filtered table,
// exit 0, no ANSI (non-TTY buffer). sampleStore has example + writing/reddit.
func TestRunSearchMatchByTag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example): code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "example") {
		t.Errorf("stdout missing 'example' row:\n%s", got)
	}
	if strings.Contains(got, "reddit") { // unmatched skill must not leak
		t.Errorf("unmatched skill 'reddit' leaked into search results:\n%s", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("non-TTY search must not emit ANSI:\n%s", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty", errOut.String())
	}
}

func TestRunSearchShortFlag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-s", "reddit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-s reddit): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "reddit") {
		t.Errorf("stdout missing matched row:\n%s", out.String())
	}
}

// --search is case-insensitive (PRD §6.1).
func TestRunSearchCaseInsensitive(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "REDDIT"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search REDDIT): code=%d; want 0 (case-insensitive)", code)
	}
	if !strings.Contains(out.String(), "reddit") {
		t.Errorf("case-insensitive query should match:\n%s", out.String())
	}
}

// --search matches by description (example has "A demo skill.").
func TestRunSearchMatchByDescription(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "demo"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search demo): code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "example") {
		t.Errorf("description match should find example:\n%s", got)
	}
	if strings.Contains(got, "reddit") {
		t.Errorf("non-matching reddit should be filtered out:\n%s", got)
	}
}

// --search matches by frontmatter name (sampleStore reddit has name reddit-poster).
func TestRunSearchMatchByName(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "poster"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search poster): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "reddit") {
		t.Errorf("name match should find reddit skill:\n%s", out.String())
	}
}

// --search matches by metadata.keywords (PRD §6.1).
func TestRunSearchMatchByKeyword(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\nmetadata:\n  keywords: [writing, social]\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "soc"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search soc): code=%d; want 0 (keyword match)", code)
	}
	if !strings.Contains(out.String(), "example") {
		t.Errorf("keyword match should find example:\n%s", out.String())
	}
}

// --search with NO matches -> exit 1, EMPTY stdout, message to stderr (PRD §6.1).
func TestRunSearchNoMatchesExit1(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "zzznotfound"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--search zzznotfound): code=%d; want 1 (no matches)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (PRD §6.1: no matches => nothing on stdout)", out.String())
	}
	if !strings.Contains(errOut.String(), "no skills matched") {
		t.Errorf("stderr=%q; want a 'no skills matched' message", errOut.String())
	}
}

// --search "" (empty query) matches ALL skills (substring semantics): exit 0, full
// table — like --list. (PRD carves out no special case for an empty query.)
func TestRunSearchEmptyQueryMatchesAll(t *testing.T) {
	dir := sampleStore(t) // 2 skills
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", ""}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search ''): code=%d; want 0 (empty matches all)", code)
	}
	got := out.String()
	if !strings.Contains(got, "example") || !strings.Contains(got, "reddit") {
		t.Errorf("empty query should list all skills:\n%s", got)
	}
}

// --search respects --no-color: suppresses ANSI even on a TTY.
func TestRunSearchNoColorSuppressesANSI(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example", "--no-color"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example --no-color): code=%d; want 0", code)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Errorf("--no-color must suppress ANSI in search:\n%s", out.String())
	}
}

// --search emits ANSI when stdout is a TTY and --no-color is absent.
func TestRunSearchColorWhenTTY(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example) tty: code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[1m") || !strings.Contains(got, "\x1b[36m") {
		t.Errorf("TTY search output should contain ANSI bold/cyan:\n%s", got)
	}
}

// --search when skills dir is unresolvable -> exit 1, empty stdout, one-line fix.
func TestRunSearchSkillsDirUnresolvable(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // all three §8 rules miss
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "x"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--search x) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("stderr=%q; want the one-line fix", errOut.String())
	}
}

// --version precedes --search (PRD §6.3).
func TestRunVersionPrecedenceOverSearch(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example --version): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (precedence over search)", got)
	}
}
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: PRECONDITION — confirm dependencies are on disk and green
  - COMMAND: cd /home/dustin/projects/skpp
  - COMMAND: grep -q 'func PrintList(w io.Writer, skills \[\]discover.Skill, useColor bool)' internal/ui/ui.go
  - COMMAND: grep -q 'func Index(skillsDir string)' internal/discover/index.go
  - COMMAND: grep -q 'type Skill struct' internal/discover/skill.go
  - COMMAND: go test ./internal/discover/ ./internal/ui/ ./internal/resolve/ ./internal/skillsdir/ >/dev/null && echo "deps green"
  - EXPECT: all three symbols exist AND those packages pass. If PrintList/Index/Skill
            are MISSING, M2 has NOT landed — STOP and let it land first.
  - COMMAND: go test . -count=1 >/dev/null && echo "main green (53 tests)"
  - EXPECT: baseline main green so the test delta (53 -> 69) is attributable to THIS task.

Task 1: CREATE internal/search/search.go
  - WRITE: the exact content from Blueprint File 1.
  - CHECK: `package search`; imports ONLY strings + internal/discover; func Search
           + unexported matches; query lowercased ONCE; four fields only (NOT
           category/aliases); keywords iterated individually (not joined).
  - GOTCHA: do NOT lowercase the whole Skill; do NOT add category/aliases; do NOT
            import os or any third-party package.

Task 2: CREATE internal/search/search_test.go
  - WRITE: the exact content from Blueprint File 2.
  - CHECK: `package search`; imports testing + internal/discover; helper sk(); 16
           tests incl. the category/aliases exclusion + keyword-boundary negatives.
  - GOTCHA: NO testify; NO t.Parallel(); pure in-memory (no filesystem).

Task 3: MODIFY main.go (apply Edits 3a-3d)
  - EDIT 3a: add "github.com/dabstractor/skpp/internal/search" to the import group.
  - EDIT 3b: add searchMode bool + searchQ string to config; drop "search string"
             from the "Future" comment (now "check bool; help bool").
  - EDIT 3c: convert parseArgs range loop -> index loop; add case "--search","-s"
             that consumes args[i+1] and i++; every other case unchanged.
  - EDIT 3d: insert the if c.searchMode {…} branch AFTER the --list block, BEFORE
             the --all block.
  - CHECK: version precedence still first; list/all/tags/default blocks unchanged.
  - GOTCHA: keep the existing default-branch comment (it already mentions --search).
            Run `gofmt -w main.go` — it realigns the config struct comments.

Task 4: MODIFY main_test.go (append the 16 tests from Blueprint File 4)
  - APPEND: the 16 named tests at end of file (4 parseArgs + 12 run).
  - CHECK: reuses sampleStore/writeSkillTree/withTerminal/unsetSkillsEnv; existing
           53 tests UNCHANGED.
  - GOTCHA: do NOT duplicate the helper definitions (they already exist). withTerminal
            mutates package state — no t.Parallel() on those tests (matches repo convention).

Task 5: FORMAT + VET + TIDY + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/search/*.go main.go main_test.go
  - COMMAND: gofmt -l internal/search/*.go main.go main_test.go   # MUST print nothing
  - COMMAND: go vet ./...                                         # MUST be clean
  - COMMAND: go mod tidy   # EXPECTED: a NO-OP (strings is stdlib; no new module)
  - COMMAND: go build ./...                                       # exit 0
  - COMMAND: go test ./internal/search/ -v                        # 16 NEW tests PASS
  - COMMAND: go test . -v                                         # 69 main tests (53 old + 16 new)
  - COMMAND: go test ./...                                        # whole module green (177 total)
  - EXPECT: zero errors, zero vet findings, gofmt silent, go.mod/go.sum unchanged.

Task 6: SMOKE + SCOPE CHECK — Levels 3 + 4 in the Validation Loop.
  - COMMAND: the Level 3 block (build, --search over a throwaway tree, no-match /
            unresolvable paths, --no-color).
  - COMMAND: the Level 4 block (scope boundaries + go.mod unchanged + the four
            READ-ONLY packages untouched).
```

### Implementation Patterns & Key Details

```go
// The WHOLE feature, end to end, is two pure functions + one dispatch branch:

// internal/search/search.go — the filter (pure, ~20 lines of logic):
func Search(query string, skills []discover.Skill) []discover.Skill {
    q := strings.ToLower(query)            // lowercase query ONCE
    var matched []discover.Skill
    for _, s := range skills {             // input order => output stays sorted
        if matches(q, s) { matched = append(matched, s) }
    }
    return matched
}
// matches() short-circuits on the first field hit (tag -> name -> desc -> any kw).

// main.go run() — the dispatch branch (mirrors the --list branch exactly):
if c.searchMode {
    dir, _, err := skillsdir.Find()
    if err != nil { fmt.Fprintln(stderr, err); return 1 }     // one-line fix; empty stdout
    skills, err := discover.Index(dir)
    if err != nil { fmt.Fprintln(stderr, err); return 1 }
    matched := search.Search(c.searchQ, skills)
    if len(matched) == 0 {                                    // MUST test BEFORE PrintList
        fmt.Fprintln(stderr, "no skills matched "+c.searchQ)  // (PrintList prints nothing
        return 1                                              //  on empty and does not exit)
    }
    ui.PrintList(stdout, matched, isTerminal(stdout) && !c.noColor)  // SAME call as --list
    return 0
}

// parseArgs — the ONE structural change (range -> index, for value consumption):
for i := 0; i < len(args); i++ {
    a := args[i]
    switch a {
    // ...all existing cases unchanged...
    case "--search", "-s":
        if i+1 < len(args) { c.searchMode = true; c.searchQ = args[i+1]; i++ } // consume value
    default:
        if !strings.HasPrefix(a, "-") { c.tags = append(c.tags, a) }          // tags unchanged
    }
}
```

### Integration Points

```yaml
NEW PACKAGE:
  - path: internal/search/
  - exports: func Search(query string, skills []discover.Skill) []discover.Skill
  - deps:    strings (stdlib) + github.com/dabstractor/skpp/internal/discover (read-only)
  - tests:   internal/search/search_test.go (white-box, 16 tests, no filesystem)

MAIN DISPATCH:
  - file: main.go
  - insert: the `if c.searchMode {…}` branch AFTER the `--list` block, BEFORE `--all`
  - precedence: version → path → list → SEARCH → all → tags → default (§6.3 exclusivity is M5)

CLI SURFACE:
  - flag: "--search <q>" / "-s <q>"  (value-taking; consumes the next token)
  - modifiers that apply: "--no-color" (shared with --list)
  - modifiers that do NOT apply: "--file", "--relative" (search prints a table)
  - exit codes: 0 on matches; 1 on no-match / unresolvable store / no-value-for-flag

CONFIG:
  - struct main.config gains: searchMode bool; searchQ string

DEPENDENCIES (go.mod): UNCHANGED. No new module; `strings` is stdlib. `go mod tidy` is a no-op.
```

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
cd /home/dustin/projects/skpp
gofmt -w internal/search/*.go main.go main_test.go
gofmt -l internal/search/*.go main.go main_test.go   # MUST print nothing
go vet ./...                                          # MUST be clean
# go mod tidy   # OPTIONAL sanity check — EXPECTED to be a no-op (diff go.mod before/after)
go build ./...                                        # exit 0
# Expected: zero output from gofmt -l; zero vet findings; build succeeds.
```

### Level 2: Unit Tests (Component Validation)

```bash
# The new pure package — fastest feedback, no filesystem.
go test ./internal/search/ -v
# Expected: 16 tests PASS (TestSearch*). If any fail, the matching logic is wrong;
#   read the failure (it names the field/scenario) and fix search.go.

# The dispatcher + parser — uses temp skill trees via the existing helpers.
go test . -run 'Search|VersionPrecedenceOverSearch' -v
# Expected: the 16 new main tests PASS (4 parseArgs + 12 run).

# Full suite — confirms NO regression in the four read-only packages.
go test ./... -count=1
# Expected: ALL PASS. Totals: . = 69 (53+16), discover = 39, resolve = 13,
#   skillsdir = 29, ui = 11, search = 16. Grand total = 177.
```

### Level 3: Integration Testing (System Validation)

```bash
cd /home/dustin/projects/skpp
go build -o skpp . && echo OK
./skpp --version                       # prints: skpp <something>

# Build a throwaway two-skill store and point skpp at it via the env override (§8 rule 1).
TMPROOT=$(mktemp -d)
mkdir -p "$TMPROOT/writing/reddit"
cat > "$TMPROOT/example/SKILL.md" <<'EOF'
---
name: example
description: A demo skill for searching.
metadata:
  keywords: [demo, search]
---
# Example
EOF
cat > "$TMPROOT/writing/reddit/SKILL.md" <<'EOF'
---
name: reddit-poster
description: Posts to reddit.
---
# Reddit
EOF

# 1) Match by tag substring -> exit 0, table on stdout, ONLY the match.
SKPP_SKILLS_DIR="$TMPROOT" ./skpp --search redd | grep -q writing/reddit && echo "tag-substring OK"

# 2) Case-insensitive.
SKPP_SKILLS_DIR="$TMPROOT" ./skpp -s REDDIT | grep -q writing/reddit && echo "case-insensitive OK"

# 3) Match by keyword.
SKPP_SKILLS_DIR="$TMPROOT" ./skpp --search demo | grep -q example && echo "keyword OK"

# 4) No matches -> exit 1, EMPTY stdout, stderr message.
out=$(SKPP_SKILLS_DIR="$TMPROOT" ./skpp --search zzz 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "no-match contract OK"

# 5) --no-color suppresses ANSI even on a TTY (force a TTY with script(1) if available).
SKPP_SKILLS_DIR="$TMPROOT" ./skpp --search example --no-color | grep -vq $'\x1b' && echo "no-color OK"

# 6) Same table shape as --list (header present).
SKPP_SKILLS_DIR="$TMPROOT" ./skpp --search example | grep -q 'TAG' && echo "table-format OK"

# 7) Empty query matches all (both skills appear).
n=$(SKPP_SKILLS_DIR="$TMPROOT" ./skpp --search '' | grep -cE 'example|reddit'); [ "$n" -ge 2 ] && echo "empty-query-matches-all OK"

rm -rf "$TMPROOT"
# Expected: every "… OK" line prints; the no-match branch empties stdout and exits 1.
```

### Level 4: Creative & Domain-Specific Validation (Scope Boundaries)

```bash
cd /home/dustin/projects/skpp

# SCOPE: go.mod / go.sum UNCHANGED (no new dependency).
git diff --name-only go.mod go.sum | (! read) && echo "go.mod/go.sum unchanged OK"
# (Or: `git diff --exit-code go.mod go.sum` prints nothing and exits 0.)

# SCOPE: the four READ-ONLY packages are untouched.
git diff --name-only internal/discover internal/skillsdir internal/resolve internal/ui | (! read) && echo "read-only packages untouched OK"

# SCOPE: no check subcommand, no --help, no skills/example created by THIS task.
./skpp check 2>&1 | grep -qi 'check' && echo "NOTE: 'check' resolves as a tag (M4.T10 not yet built) — expected"

# CONTRACT: --search respects §6.4 (nothing on stdout on the failure paths).
SKPP_SKILLS_DIR=/does/not/exist ./skpp --search x 2>/dev/null; [ $? -eq 1 ] && echo "unresolvable-store exit 1 OK"

# Expected: scope guards confirm no collateral changes; the contract guard confirms
#   the empty-stdout-on-failure discipline that makes $(...) safe.
```

## Final Validation Checklist

### Technical Validation

- [ ] Level 1 passed: `gofmt -l` silent; `go vet ./...` clean; `go build ./...` ok.
- [ ] Level 2 passed: `go test ./... -count=1` green (**177** tests).
- [ ] Level 3 passed: every `… OK` smoke line printed.
- [ ] Level 4 passed: scope guards confirm no collateral changes.
- [ ] `go mod tidy` was a no-op (`go.mod`/`go.sum` byte-identical).
- [ ] No linting errors, no type errors, no formatting drift.

### Feature Validation

- [ ] `--search <q>` and `-s <q>` both work; value consumed (not a tag).
- [ ] Matches by tag, name, description, AND keyword (positive tests).
- [ ] Case-insensitive (`REDDIT` matches `reddit`).
- [ ] Field scope is exactly tag/name/description/keywords — `Category`/`Aliases`
      excluded (negative tests `TestSearchDoesNotMatchCategoryOrAliases`).
- [ ] Keywords matched individually, not joined (negative test
      `TestSearchKeywordSubstringNotJoinBoundary`).
- [ ] No matches → exit 1, empty stdout, `no skills matched <q>` on stderr.
- [ ] Empty query matches all; order preserved (sorted by RelTag).
- [ ] `--no-color` suppresses ANSI; TTY enables ANSI (shared with `--list`).
- [ ] `--file`/`--relative` do not affect search output (table, not paths).
- [ ] Skills dir unresolvable → exit 1, empty stdout, one-line fix on stderr.
- [ ] `--version` precedes `--search`.

### Code Quality Validation

- [ ] New `internal/search` mirrors `internal/resolve` (pure, own package, own test).
- [ ] `main.run` stays a thin dispatcher; no business logic inlined.
- [ ] `parseArgs` is the only structural change (range → index loop); all other
      `case`s byte-identical.
- [ ] Existing 53 main tests + all read-only packages UNCHANGED and green.
- [ ] No new third-party dependency; stdlib `strings` only.
- [ ] Comments explain the *why* (field scope, individual-keyword matching,
      empty-query semantics, deferred exit-2).

### Documentation & Deployment

- [ ] Doc comments on `Search`/`matches` state the PRD §6.1 field scope and the
      "mirrors resolve" rationale.
- [ ] The `run` search-branch comment cross-references §6.1 and the ui reuse.
- [ ] The parseArgs `--search` case documents the deferred exit-2 (M5.T11).
- [ ] (README `--search` usage is owned by P1.M6.T14 — do NOT edit README here.)

---

## Anti-Patterns to Avoid

- ❌ Don't search `Category` or `Aliases` — PRD §6.1 scopes to tag/name/description/
  keywords only, even though both fields sit on `discover.Skill`.
- ❌ Don't `strings.Join` the keywords — match each individually (boundary safety).
- ❌ Don't lowercase the query per-field — lowercase it ONCE.
- ❌ Don't re-sort the filtered slice — `discover.Index` already sorted it and
  `ui.PrintList` does not re-sort.
- ❌ Don't call `ui.PrintList` before checking `len(matched)==0` — it prints nothing
  on empty and does not exit; `main` owns the exit-1 decision.
- ❌ Don't inline the filter in `main` — mirror `internal/resolve` (own package).
- ❌ Don't add `check`, `--help`, exit-2, or `skills/example/` — out of scope (T10/
  M5/M6). Don't modify the four read-only packages.
- ❌ Don't add a third-party dependency — `strings` is stdlib; `go.mod` stays put.
- ❌ Don't break §6.4: on every failure path (no matches, unresolvable store), keep
  stdout EMPTY and put the message on stderr.

---

**Confidence Score: 9.5/10** for one-pass implementation success. The feature is a
pure stdlib filter between two already-green components (`discover.Index` and
`ui.PrintList`); the four files are specified verbatim (two full new files, exact
edit snippets for `main.go`, named tests for `main_test.go`); every semantics
edge (field scope, keyword boundaries, empty query, no-match exit, color gating,
deferred exit-2) is pinned by a test. The 0.5 reserve is for the mechanical
`parseArgs` loop conversion (range → index) — straightforward but the one edit
most likely to introduce a typo, which `go test .` catches immediately.
