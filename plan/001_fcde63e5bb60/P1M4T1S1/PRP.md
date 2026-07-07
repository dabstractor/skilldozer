# PRP — P1.M4.T1.S1: `--search` substring filter + `ui.PrintList` reuse

> **Subtask:** P1.M4.T1.S1 (plan id) = P1.M4.T9.S1 (PRD build-order id) — the only
> subtask of T9 (`--search`, PRD §6.1). Milestone **M4** (Search & validation).
> Build-order step 4. **Small SP** — this is a FILTER + WIRING task: it adds the
> `--search`/`-s <q>` mode to `main.go`, filters `discover.Index()` output by a
> case-insensitive substring over {RelTag, Name, Description, Keywords}, and
> reuses `ui.PrintList` (P1.M2.T6.S1) to render the result. No new package, no
> new dependency.
>
> **Scope:** MODIFY `main.go` (config fields + index-loop parseArgs + run()
> search block + two unexported filter helpers + `"strings"` import) and APPEND
> tests to `main_test.go`. **Two existing files touched; nothing else.** No new
> files, no `internal/search/` package, no `go.mod`/`go.sum` change.
>
> **PARALLEL CONTEXT:** `internal/resolve/` (P1.M3.T7.S1) LANDED on disk during
> this planning pass and is green. This subtask does NOT depend on resolve
> (search never calls `resolve.Resolve`); it depends only on `discover.Index`,
> `discover.Skill`, `ui.PrintList`, and `main`'s own dispatcher. Resolve being
> present or absent does not affect this subtask's compilation.
>
> **VERIFICATION STATUS:** Baseline measured live at `/home/dustin/projects/skpp`
> (go1.25): `go build ./...` exit 0; `go test ./...` = **105 tests** (24 main +
> 31 discover + 10 resolve + 29 skillsdir + 11 ui); `go.mod` has `yaml.v3` as the
> sole direct dep; `main.go` has no `strings` import and no `search*` symbol
> (clean slate). Every load-bearing fact is in `research/verified_facts.md`.

---

## Goal

**Feature Goal**: Implement PRD §6.1 `skpp --search <q>` / `-s <q>`: build the
index via `discover.Index()`, filter it to skills where (case-insensitive) `q` is
a substring of **RelTag, Name, Description, OR any of Keywords**, and render the
filtered set through the SAME `ui.PrintList` table used by `--list`. On zero
matches, print one line to stderr and exit 1 (stdout stays empty).

**Deliverable**: MODIFY two existing files (no new files):
1. `main.go` — add `search`/`searchSet` to `config`; convert `parseArgs` to an
   index loop that captures the `--search`/`-s` value; add a `search` block to
   `run()` (after `--list`, before the default); add unexported `searchSkills` +
   `matchesQuery`; add `"strings"` to the import group.
2. `main_test.go` — append 13 tests (4 parseArgs, 5 pure-filter, 4 run() integration).

**Success Definition**: `gofmt -l *.go` silent; `go vet ./...` clean;
`go build ./...` + `go test ./...` pass (**118 tests** = 105 baseline + **13 NEW**);
`go.mod`/`go.sum` byte-identical (`go mod tidy` is a no-op). `skpp --search <q>`
prints the filtered catalog table (exit 0) when ≥1 skill matches, and prints
`no skills match "<q>"` to stderr + exit 1 with empty stdout when none do.
Mutual-exclusivity (`--search` vs `--list`/tags/`--all`) is **NOT** enforced here
(deferred to P1.M5.T11 per the item contract).

---

## Why

- PRD §18 build-order step 4. Until `--search` lands, the §6.1 CLI matrix is
  incomplete (`--list` works, but the substring catalog search does not).
- It is the cheapest possible reuse story: the entire output renderer
  (`ui.PrintList`) already exists from P1.M2.T6.S1, so this is a ~10-line filter
  + the wiring that calls it. Locking the §6.1 **search field set** (tag/name/
  description/keywords) and the **case-insensitive substring** semantics now is
  the whole point — the item calls this set "the documented contract".
- It proves the `discover.Index()` → filter → `ui.PrintList` data flow for a
  SELECTED subset, complementing `--list` (which renders the full set). The
  acceptance gate (PRD §13) exercises `--search`.
- It is **go.mod-neutral**: the only new import is the stdlib `"strings"`.

---

## What

A new `--search`/`-s <q>` mode in `main.go`:

1. **`parseArgs`** recognizes `--search`/`-s` and consumes the NEXT argv token as
   the query string. (First value-taking flag; parseArgs becomes an index loop.)
2. **`run()`** gains a `search` dispatch block: `skillsdir.Find()` →
   `discover.Index(dir)` → `searchSkills(skills, q)` → if empty, stderr + exit 1;
   else `ui.PrintList(stdout, matched, isTerminal(stdout) && !c.noColor)`, exit 0.
3. **`searchSkills`** (unexported, in main.go) filters `[]discover.Skill`,
   preserving input order (so the result stays RelTag-sorted for the table).
4. **`matchesQuery`** (unexported, in main.go) reports whether the lowercased
   query is a substring of `RelTag`, `Name`, `Description`, or any `Keyword`.

### Success Criteria

- [ ] `skpp --search <q>` and `skpp -s <q>` both work (long + short forms).
- [ ] Match is case-INsensitive substring over RelTag, Name, Description, and
      Keywords (all four; a keyword-only or description-only query still matches).
- [ ] Output is the SAME table as `--list` (reuses `ui.PrintList`), filtered to
      matches, still sorted by tag (no re-sort).
- [ ] Zero matches ⇒ exit 1, one stderr line `no skills match "<q>"`, stdout EMPTY.
- [ ] `--no-color` suppresses ANSI exactly as it does for `--list` (same gate).
- [ ] Color on a TTY, plain output on a pipe/buffer (same `isTerminal` gate as `--list`).
- [ ] Skills dir unresolvable / `Index` error ⇒ exit 1 + the one-line fix to
      stderr (same contract as `--path`/`--list`).
- [ ] Mutual-exclusivity with `--list`/tags/`--all` is **NOT** enforced (M5).
- [ ] `gofmt`/`go vet`/`go build`/`go test` clean; `go.mod`/`go.sum` unchanged.
- [ ] 13 new tests added; whole module green (118 tests).

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for every change is given verbatim in the Implementation
Blueprint (the config struct delta, the index-loop parseArgs, the run() search
block, the two filter helpers, the import line, and all 13 tests). The consumed
contracts are on disk and green: `discover.Index`/`discover.Skill`
(`internal/discover/`), `ui.PrintList` (`internal/ui/ui.go`), and `main`'s own
`run`/`parseArgs`/`config`/`isTerminal`. Baseline measured (105 tests, build OK).
Every load-bearing decision — filter lives in main.go (NOT a new package), index
loop for the value-taking flag, `searchSet` to distinguish `--search ""` from no
flag, order preserved (no re-sort), empty-result ⇒ stderr + exit 1 + empty
stdout, empty-query edge deferred to M5, go.mod-neutral — is in
`research/verified_facts.md`. An implementer who knows Go but nothing about this
repo can finish in one pass by applying the edits and tests verbatim._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical baseline (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M4T1S1/research/verified_facts.md
  why: "Measured BEFORE writing the PRP. Locks: (0) baseline = 105 tests, build
        green, resolve LANDED but NOT a dependency. (1) the consumed contracts —
        discover.Index returns []Skill sorted by RelTag (so the filtered slice
        stays sorted; NO re-sort), Skill fields search touches (RelTag/Name/
        Description/Keywords), ui.PrintList(w, skills, useColor) is the reuse
        target, main.run/parseArgs/config/isTerminal are the dispatcher to extend.
        (2) the §6.1 field set + case-insensitive substring. (3) DECISION: filter
        lives in main.go as unexported funcs (NOT a new package) — item says 'In
        main.go add', go_architecture package map has no internal/search, search
        is a leaf with no reuse, main_test.go is package main (white-box) so
        unexported funcs are testable. (4) parseArgs becomes an index loop (first
        value-taking flag). (5) searchSet distinguishes --search '' from no flag.
        (6) run() order: version > path > list > search > default. (7) empty-result
        => stderr 'no skills match %q' + exit 1 + EMPTY stdout. (8) empty-query
        edge deferred to M5 (non-crashing). (9) --search consumes next token
        literally (even flag-shaped). (10) go.mod UNCHANGED (only 'strings' added).
        (11) test convention. (12) 13-test plan."
  critical: "searchSkills must NOT re-sort (input is already RelTag-sorted by
             discover.Index; appending in iteration order preserves it). The
             empty-result branch must NOT call ui.PrintList (stdout stays empty).
             Do NOT create internal/search/."

# CONTRACT — the Skill fields search touches (on disk; READ-ONLY)
- file: internal/discover/skill.go
  why: "type Skill struct{Dir,RelTag,Name,Description,Keywords,Category,Aliases,
        HasFM,SourceFile}. Search touches ONLY RelTag/Name/Description/Keywords.
        Keywords is []string (nil if absent; test with len(), not nil-check).
        Description may carry a folded-scalar trailing newline (verbatim copy) —
        substring match is unaffected (no TrimSpace in the filter; ui.PrintList
        TrimSpaces for DISPLAY only)."
  pattern: "matchesQuery reads s.RelTag/s.Name/s.Description/s.Keywords."
  gotcha: "Do NOT search Aliases/Category/Dir — §6.1 field set is exactly the four."

# CONTRACT — Index() (the data source; on disk; READ-ONLY)
- file: internal/discover/index.go
  why: "func Index(skillsDir string) ([]Skill, error) — walks the store, returns
        []Skill SORTED by RelTag (sort.Slice on RelTag). This sort is load-bearing:
        searchSkills appends matches in iteration order, so the filtered slice is
        still tag-sorted — do NOT re-sort."
  pattern: "main's search block calls discover.Index(dir) exactly like the --list block."

# CONTRACT — the renderer to REUSE (on disk; READ-ONLY)
- file: internal/ui/ui.go
  why: "func PrintList(w io.Writer, skills []discover.Skill, useColor bool). Renders
        the TAG/NAME/DESCRIPTION table; empty slice => returns early (prints
        nothing). PRD §6.1: --search uses 'Same table format as --list' — so call
        PrintList with the FILTERED slice. Color is caller-controlled (useColor),
        so main owns the TTY/--no-color decision (already wired for --list)."
  pattern: "ui.PrintList(stdout, matched, isTerminal(stdout) && !c.noColor) — identical
            to the --list call, just on the filtered slice."
  gotcha: "Do NOT add a PrintSearch to ui. ui is a PURE formatter (its doc comment says
           so); filtering belongs in main. Reuse PrintList verbatim."

# CONTRACT — the dispatcher to extend (on disk; the file THIS subtask modifies)
- file: main.go
  why: "main.go owns parseArgs (argv->config), run() (dispatch + exit codes),
        config (parsed flags), isTerminal (TTY gate), version (ldflags var). This
        subtask ADDS to all four: config gains search/searchSet; parseArgs gains
        the --search/-s value capture (index loop); run() gains the search block;
        the filter helpers are added as unexported funcs. isTerminal + version are
        REUSED unchanged."
  pattern: "Mirror the existing --list block's shape exactly: Find() -> Index() ->
            (empty?) -> ui.PrintList(stdout, ..., isTerminal(stdout) && !c.noColor)."
  gotcha: "parseArgs is currently `for _, a := range args` — CANNOT consume a neighbor.
           Must switch to an index loop to grab args[i+1] as the query. The search
           block goes AFTER the list block (version > path > list > search > default)."

# CONTRACT — the §6.1 row this implements + §6.4 stdout discipline
- file: PRD.md
  why: "§6.1 table: 'skpp --search <q> / -s <q> | Substring (case-insensitive) search
        over tag, frontmatter name, description, and metadata.keywords. | Same table
        format as --list, filtered. | 0; 1 if no matches'. §6.4 spirit: on failure
        print NOTHING to stdout (so $(...) stays clean) — applied to the no-match path."
  section: "6.1 Commands / flags", "6.4 Error semantics"
  critical: "READ-ONLY. Do NOT edit PRD.md. Field set is EXACTLY {tag,name,description,
             keywords}; case-insensitive; exit 1 on no matches."

# REFERENCE — the architecture package map (confirms NO internal/search)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "'Package map (PRD §5)' lists exactly main + 4 internal packages (skillsdir,
        discover, resolve, ui). There is NO internal/search. The ui.go comment in the
        map says '--list / --search table' (ui owns the table; main owns filtering).
        This is why the filter lives in main.go, not a new package."
  section: "Package map (PRD §5)", "Data flow", "ANSI color"

# REFERENCE — the test convention (mirrors it for the 13 new tests)
- file: main_test.go
  why: "package main (white-box), plain t.Errorf/t.Fatalf, table-driven where natural,
        NO testify, NO t.Parallel(). Reuses the EXISTING helpers writeSkillTree
        (map[relTag]content -> temp skills/), unsetSkillsEnv, withTerminal — do NOT
        redefine them. run() integration tests build a temp store, set
        SKPP_SKILLS_DIR, capture stdout/stderr via *bytes.Buffer, assert code + output."
  pattern: "Copy the shape of TestRunListSuccess / TestRunListNoSkillsExit1 / 
            TestRunListSkillsDirUnresolvableExit1 — just swap --list for --search <q>."

# REFERENCE — pure-filter test convention (struct literals, no disk)
- file: internal/resolve/resolve_test.go
  why: "The pattern for testing a pure function over []discover.Skill WITHOUT disk
        fixtures: a package-level `var exampleSkills = []discover.Skill{...}` and
        table/cases over it. The 5 searchSkills tests mirror this with `searchFixture`."
  pattern: "package-level fixture slice of discover.Skill literals; assert len + fields."

# URLS — the load-bearing stdlib surface
- url: https://pkg.go.dev/strings#Contains
  why: "strings.Contains(haystack, needle) reports substring containment. Both
        arguments are lowercased first (needle once in searchSkills, each field in
        matchesQuery) for case-insensitive matching. Contains(x, \"\") is always true
        — the empty-query edge (deferred to M5)."
- url: https://pkg.go.dev/strings#ToLower
  why: "strings.ToLower normalizes both sides of the comparison for case-insensitive
        substring search (PRD §6.1). Lowercasing the needle ONCE (in searchSkills) and
        each field at compare time avoids re-lowercasing the needle per field."
```

### Current Codebase tree (M1 + M2 + M3.T7 COMPLETE; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*' | sort
internal/discover/discover.go        # M2.T4.S1: Frontmatter(8) + ParseFrontmatter + utf8BOM
internal/discover/discover_test.go   # M2.T4.S1 tests (package discover)
internal/discover/skill.go           # M2.T4.S2: Skill(9) + BuildSkill + toStringSlice
internal/discover/skill_test.go      # M2.T4.S2 tests
internal/discover/index.go           # M2.T5.S1: Index(skillsDir)([]Skill,error) + sort by RelTag
internal/discover/index_test.go      # M2.T5.S1 tests
internal/resolve/resolve.go          # M3.T7.S1: Resolve + MatchKind + typed errors (LANDED)
internal/resolve/resolve_test.go     # M3.T7.S1 tests (10; LANDED)
internal/skillsdir/skillsdir.go      # M1.T2: Source + Find + per-rule helpers
internal/skillsdir/skillsdir_test.go # M1.T2 tests (package skillsdir)
internal/ui/ui.go                    # M2.T6.S1: PrintList table + ANSI + padRight/wrapWords
internal/ui/ui_test.go               # M2.T6.S1 tests
main.go                              # M1.T3 + M2.T6: version/path/list dispatch   <-- MODIFY
main_test.go                         # M1.T3 + M2.T6 tests (package main)           <-- APPEND

# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT, ONLY)
# baseline (measured): go build ./... OK; go test ./... = 105 tests (24 main + 31 discover + 10
#   resolve + 29 skillsdir + 11 ui). main.go has NO "strings" import and NO search* symbol.
# NO skills/ dir yet (P1.M6.T12 ships skills/example/SKILL.md).
```

### Desired Codebase tree with files to be added/modified

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/skillsdir/*,
│        internal/discover/* [Frontmatter+Skill+Index], internal/resolve/* [LANDED],
│        internal/ui/* — ALL UNCHANGED by this subtask)
├── main.go        # MODIFY — config +search/searchSet; parseArgs index loop + --search;
│                  #          run() +search block; +searchSkills +matchesQuery; +"strings"
└── main_test.go   # MODIFY — APPEND 13 tests (4 parseArgs, 5 filter, 4 run integration)
```

| File (modified) | Change | New import |
|---|---|---|
| `main.go` | config +2 fields; parseArgs → index loop w/ `--search`/`-s` capture; run() +search block; +`searchSkills`/`matchesQuery` (unexported) | `+ "strings"` (stdlib) |
| `main_test.go` | append 13 tests; reuse existing `writeSkillTree`/`unsetSkillsEnv`/`withTerminal` | none (already imports bytes/io/os/path/filepath/strings/testing) |

**Two existing files touched. NO new files. NO `internal/search/` package. NO
`go.mod`/`go.sum` change (`"strings"` is stdlib).**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — Filter lives in main.go, NOT a new package. The item CONTRACT says
// "In main.go add `--search`/`-s <q>` mode ... then filter". go_architecture.md's
// package map (PRD §5) lists exactly 5 packages and has NO internal/search. Search
// is a LEAF (only main's --search mode uses it; no downstream reuse), and main_test.go
// is `package main` (white-box) so unexported searchSkills/matchesQuery are testable
// directly. Contrast with resolve, which earned its own package by being non-trivial
// AND reused. A ~10-line filter does not justify a package boundary.
//   RIGHT: add `func searchSkills(...)` + `func matchesQuery(...)` to main.go.
//   WRONG: create internal/search/search.go  (deviates from the documented layout).

// GOTCHA #2 — Do NOT re-sort the filtered slice. discover.Index already sorts []Skill
// by RelTag; searchSkills appends matches IN ITERATION ORDER, so the result is still
// tag-sorted — exactly what ui.PrintList expects (it renders rows in the given order).
// Re-sorting is wasted work and would mask any future ordering regression.
//   RIGHT: for _, s := range skills { if matchesQuery(s, needle) { out = append(out, s) } }
//   WRONG: sort.Slice(out, ...)  // redundant; Index already sorted the input.

// GOTCHA #3 — parseArgs must become an INDEX loop. --search/-s is the FIRST value-
// taking flag (all existing flags are boolean toggles parsed with `for _, a := range
// args`). A range loop CANNOT consume the next token. Switch to `for i := 0; i <
// len(args); i++` and do `i++` after `c.search = args[i+1]`. All existing parseArgs
// tests still pass: nil/empty ranges zero times, unknown tokens hit the same default
// no-op, order-independence is unaffected.
//   RIGHT: case "--search", "-s": c.searchSet = true; if i+1 < len(args) { c.search = args[i+1]; i++ }
//   WRONG: case "--search", "-s": c.search = a  // a IS "--search", not the query.

// GOTCHA #4 — searchSet distinguishes "--search was given" from "no --search". The
// query may legitimately be "" (e.g. --search as the last token, or --search ""). Gate
// the run() search block on `c.searchSet`, NOT on `c.search != ""`, so an empty query
// still ENTERS search mode rather than falling through to the no-args default(1).
//   RIGHT: if c.searchSet { ... }
//   WRONG: if c.search != "" { ... }  // can't tell --search "" from no flag.

// GOTCHA #5 — Empty-result path must NOT call ui.PrintList. PRD §6.1: exit 1 if no
// matches. The §6.4 "clean stdout on failure" discipline applies: print one stderr
// line, exit 1, and leave stdout EMPTY. ui.PrintList on an empty slice prints nothing
// anyway, but do NOT rely on that — branch on len(matched)==0 BEFORE calling PrintList
// so the intent is explicit and stdout is provably empty.
//   RIGHT: if len(matched) == 0 { fmt.Fprintf(stderr, "no skills match %q\n", q); return 1 }
//   WRONG: ui.PrintList(stdout, matched, ...); if len(matched)==0 { ... return 1 }  // ordering leak.

// GOTCHA #6 — Empty-query edge is DEFERRED to M5 (documented, non-crashing).
// strings.Contains(x, "") == true for every x, so `--search ""` matches ALL skills
// (degenerate, ~--list). The item's test matrix has no empty-query case, and argument
// validation (missing/empty query -> exit 2) is P1.M5.T11's job. Keep the filter
// HONEST (no special-case for ""); document the behavior in searchSkills' doc comment.
// Also: `--search` as the LAST token (no value) is guarded by `i+1 < len(args)` ->
// searchSet=true, search="", no panic; full missing-arg -> exit 2 is M5.

// GOTCHA #7 — --search consumes the NEXT token LITERALLY (even if it looks like a flag).
// `skpp --search --list` parses q="--list" and enters search mode. Refining "--search
// must not absorb a flag as its value" is argument validation -> M5. Do not try to
// peek/validate the captured token here.
//   RIGHT: c.search = args[i+1]; i++   // literal next token.
//   WRONG: if strings.HasPrefix(args[i+1], "-") { ... }  // validation is M5.

// GOTCHA #8 — Lowercase the needle ONCE, each field at compare time. Case-insensitive
// substring = Contains(ToLower(field), ToLower(q)). Lowercasing the needle once in
// searchSkills (not per field) avoids redundant work; lowercasing each field at the
// matchesQuery call site keeps the field comparison correct. Do NOT mutate s (it's a
// copy from the range, but keep the helper pure — no writes to the Skill).
//   RIGHT: needle := strings.ToLower(q); ... strings.Contains(strings.ToLower(s.RelTag), needle)
//   WRONG: strings.Contains(s.RelTag, q)  // case-SENSITIVE — violates §6.1.

// GOTCHA #9 — Do NOT search Aliases/Category/Dir/SourceFile. §6.1 field set is
// EXACTLY {tag(RelTag), name, description, keywords}. Aliases belong to tag RESOLUTION
// (§7.2, resolve.Resolve), not search. Searching extra fields would silently widen the
// contract and break the documented "search field set" the item calls a contract.
//   RIGHT: match on RelTag, Name, Description, Keywords only.
//   WRONG: also check s.Aliases / s.Category / s.Dir.

// GOTCHA #10 — go.mod/go.sum are UNCHANGED. The only new import is the STDLIB
// "strings". yaml.v3 is already the sole direct dependency. `go mod tidy` is a no-op.
// If it changes anything, you added an external import you should not have. Verify with
// `git diff --quiet go.mod go.sum` (exit 0).

// GOTCHA #11 — main_test.go is `package main` (white-box), so it can call the
// unexported searchSkills/matchesQuery directly — that is WHY the filter does not need
// its own package for testability. Mirror the existing test style: plain t.Errorf/
// t.Fatalf, table-driven where natural, NO testify, NO t.Parallel(). REUSE the existing
// writeSkillTree / unsetSkillsEnv / withTerminal helpers — do NOT redefine them.

// GOTCHA #12 — Mutual-exclusivity is NOT enforced here. PRD §6.3 says mixing <tag> with
// --list/--search/--all is an error (exit 2), but the item CONTRACT point 3 explicitly
// assigns that to P1.M5.T11 ("--search is mutually exclusive ... enforced in P1.M5.T11").
// So `--list --search` together currently just runs --list (it returns first in run()'s
// ordering). Do not add an exit-2 check. Deterministic pre-M5 behavior is acceptable.
```

---

## Implementation Blueprint

### Data model — the config delta (LOCKED)

```go
// main.go: extend the existing config struct with two fields.
type config struct {
	version bool
	path    bool
	list    bool
	noColor bool
	search    string // --search / -s <q>: the query text (may be "")
	searchSet bool   // true once --search/-s is seen (distinguishes "" from absent)
	// Future (M3-M5), do NOT add yet:
	//   all bool; check bool; file, relative, help bool; tags []string
}
```

### Edit 1 — `main.go` imports: add `"strings"`

The module path is `github.com/dabstractor/skpp` (confirmed by `go.mod`). The current
stdlib import group is `"fmt"`, `"io"`, `"os"`. Add `"strings"` to the END of the
stdlib group (it sorts after `"os"`), keeping it separate from the internal-package
group. The result is:

```go
import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
)
```

Verify the result with `gofmt -l main.go` (silent) — gofmt owns import grouping/order,
so if your editor's goimports disagrees, let `gofmt -w main.go` settle it.

### Edit 2 — `main.go` config struct: add `search` + `searchSet`

Find the existing `config` struct (in main.go) and append the two fields before the
"Future (M3-M5)" comment block:

```go
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	list    bool // --list / -l    : print the human-readable catalog table (§6.1)
	noColor bool // --no-color     : disable ANSI color even on a TTY (§6.2)
	// --search / -s <q>: substring (case-insensitive) search over tag/name/
	// description/keywords (§6.1). search is the query text (may be ""); searchSet
	// marks that the flag was seen so "" is distinguishable from "no --search".
	search    string
	searchSet bool
	// Future (M3-M5), do NOT add yet:
	//   all bool; check bool; file, relative, help bool; tags []string
}
```

### Edit 3 — `main.go` parseArgs: index loop + `--search`/`-s` value capture

Replace the entire existing `parseArgs` function body (currently a `for _, a := range
args` switch) with the index-loop version. The boolean-flag cases are UNCHANGED; only
the loop header and the new `--search` case differ:

```go
// parseArgs scans argv tokens and fills a config. Flags may appear in any order
// (PRD §6). Long forms use POSIX double-dash; short forms a single dash. Unknown
// tokens are tolerated for now (a no-op switch default); the full unknown-flag
// -> exit 2 behavior and subcommand/positional parsing land in P1.M5.T11.
//
// --search / -s is the FIRST value-taking flag: it consumes the NEXT argv token as
// the query. The loop is therefore index-based (not `for _, a := range args`) so it
// can advance past the captured value. A missing value (--search as the last token)
// leaves search="" with searchSet=true so the mode still runs without a crash; the
// missing-argument -> exit 2 rule lands in P1.M5.T11.
//
// To add a flag in a later milestone: append a `case "--name", "-n": cfg.name =
// true` (or capture the next arg for value-taking flags like --search <q>).
func parseArgs(args []string) config {
	var c config
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--no-color":
			c.noColor = true
		case "--search", "-s":
			// Value-taking flag: consume the next token as the query (literally,
			// even if it looks like a flag — validation is M5). searchSet marks
			// presence so an empty query is distinguishable from "no --search".
			c.searchSet = true
			if i+1 < len(args) {
				c.search = args[i+1]
				i++ // consume the query value
			}
		default:
			// Unknown flag / subcommand / positional: tolerated for now.
			// P1.M5.T11 implements: unknown flag -> exit 2 (§6.2),
			// `check` subcommand dispatch, and <tag> positional capture.
		}
	}
	return c
}
```

### Edit 4 — `main.go` run(): insert the search block AFTER the list block, BEFORE the default

The existing run() ends its recognized-mode chain with the `if c.list { ... }` block
followed by `return 1` (the no-args default). Insert the search block between them:

```go
	if c.searchSet {
		// PRD §6.1 `skpp --search <q>` / `-s <q>`: resolve the store, build the
		// index, filter to skills whose RelTag/Name/Description/Keywords contain
		// q as a case-insensitive substring, and render the SAME table as --list
		// (ui.PrintList). Empty result -> one stderr line + exit 1; stdout stays
		// empty (§6.1 "1 if no matches"; §6.4 clean-stdout discipline).
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // verbatim one-line fix (PRD §6.4/§8.4)
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		matched := searchSkills(skills, c.search)
		if len(matched) == 0 {
			// No match: nothing to stdout, one line to stderr (includes the query).
			fmt.Fprintf(stderr, "no skills match %q\n", c.search)
			return 1
		}
		// Same color gate as --list: TTY stdout AND no --no-color (§6.2).
		// A *bytes.Buffer (tests) / pipe / file is not a TTY -> plain output.
		ui.PrintList(stdout, matched, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
```

### Edit 5 — `main.go`: add the two filter helpers (unexported)

Add these AFTER the `run` function (or anywhere at package scope in main.go). They
are the §6.1 substring filter:

```go
// searchSkills filters skills to those whose RelTag, Name, Description, or any
// Keyword contains q as a CASE-INSENSITIVE substring (PRD §6.1 --search). The
// returned slice preserves INPUT ORDER: discover.Index sorts by RelTag, and this
// loop appends matches in place, so the filtered set is still tag-sorted for
// ui.PrintList (do NOT re-sort).
//
// An empty query ("") is a substring of every string, so it matches ALL skills
// (degenerate, ~--list). Argument validation (missing/empty query -> exit 2) is
// P1.M5.T11's job; this filter stays a pure, honest substring match.
func searchSkills(skills []discover.Skill, q string) []discover.Skill {
	needle := strings.ToLower(q) // lowercase once; each field is lowercased at compare time
	var matched []discover.Skill
	for _, s := range skills {
		if matchesQuery(s, needle) {
			matched = append(matched, s)
		}
	}
	return matched
}

// matchesQuery reports whether the lowercased needle is a substring of s.RelTag,
// s.Name, s.Description, or any of s.Keywords — the §6.1 search field set
// (NOT Aliases/Category/Dir). The needle is already lowercased by searchSkills;
// each field is lowercased here so the match is case-insensitive. Returns on the
// first hit (short-circuit). Pure: does not mutate s.
func matchesQuery(s discover.Skill, needle string) bool {
	if strings.Contains(strings.ToLower(s.RelTag), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Name), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Description), needle) {
		return true
	}
	for _, k := range s.Keywords {
		if strings.Contains(strings.ToLower(k), needle) {
			return true
		}
	}
	return false
}
```

> **Copy-paste correctness:** all five edits are gofmt-clean. The algorithm is
> `Contains(ToLower(field), ToLower(q))` over the four §6.1 fields, short-circuit
> on first hit, append in order (no re-sort). main.go's new import is exactly
> `"strings"` (stdlib) — go.mod stays unchanged. Every assertion maps to a
> verified_facts entry.

### Edit 6 — `main_test.go`: APPEND 13 tests (reuse existing helpers)

Append the following to the END of main_test.go. It reuses `writeSkillTree`,
`unsetSkillsEnv`, and the existing `discover` import (add the import if missing —
see the note below the block). Style mirrors the repo: `package main`, plain
`t.Errorf`/`t.Fatalf`, no testify, no `t.Parallel()`.

```go
// --- parseArgs: --search / -s (P1.M4.T9) ---

// --search long form consumes the next token as the query and sets searchSet.
func TestParseArgsSearchLong(t *testing.T) {
	c := parseArgs([]string{"--search", "reddit"})
	if !c.searchSet {
		t.Errorf("parseArgs(--search reddit): searchSet=false; want true")
	}
	if c.search != "reddit" {
		t.Errorf("parseArgs(--search reddit): search=%q; want reddit", c.search)
	}
}

func TestParseArgsSearchShort(t *testing.T) {
	c := parseArgs([]string{"-s", "foo"})
	if !c.searchSet || c.search != "foo" {
		t.Errorf("parseArgs(-s foo): searchSet=%v search=%q; want true, foo", c.searchSet, c.search)
	}
}

// --search may combine with other flags in any order (PRD §6). The captured value
// is the token immediately after --search, and the flag after it is NOT also parsed.
func TestParseArgsSearchAnyOrder(t *testing.T) {
	c := parseArgs([]string{"--no-color", "--search", "reddit"})
	if !c.noColor || !c.searchSet || c.search != "reddit" {
		t.Errorf("parseArgs(--no-color --search reddit): noColor=%v searchSet=%v search=%q; want true,true,reddit",
			c.noColor, c.searchSet, c.search)
	}
}

// --search as the LAST token (no query value): must NOT panic (index guard),
// searchSet=true, search="". The missing-arg -> exit 2 rule is M5's job.
func TestParseArgsSearchMissingValueNoCrash(t *testing.T) {
	c := parseArgs([]string{"--search"})
	if !c.searchSet {
		t.Errorf("parseArgs(--search): searchSet=false; want true (mode still entered)")
	}
	if c.search != "" {
		t.Errorf("parseArgs(--search): search=%q; want \"\"", c.search)
	}
}

// --- searchSkills / matchesQuery (pure filter, §6.1 field set) ---

// searchFixture mirrors discover.Index output: a []discover.Skill already SORTED by
// RelTag (coding/go < misc < writing/reddit). searchSkills must preserve this order.
var searchFixture = []discover.Skill{
	{RelTag: "coding/go", Name: "gopher", Description: "Go helper", Keywords: []string{"compile"}},
	{RelTag: "misc", Name: "", Description: "A grab bag", Keywords: nil},
	{RelTag: "writing/reddit", Name: "reddit-poster", Description: "Drafts reddit posts", Keywords: []string{"social", "marketing"}},
}

// Keyword-only match: q matches a keyword but NOT tag/name/description.
func TestSearchSkillsKeywordOnly(t *testing.T) {
	got := searchSkills(searchFixture, "marketing")
	if len(got) != 1 || got[0].RelTag != "writing/reddit" {
		t.Fatalf("searchSkills(marketing)=%+v; want exactly [writing/reddit] (keyword-only match)", got)
	}
}

// Description match: q matches the description field only.
func TestSearchSkillsDescriptionMatch(t *testing.T) {
	got := searchSkills(searchFixture, "grab bag")
	if len(got) != 1 || got[0].RelTag != "misc" {
		t.Fatalf("searchSkills(\"grab bag\")=%+v; want exactly [misc] (description match)", got)
	}
}

// Case-insensitivity: an UPPERCASE query matches lowercase field content.
// "REDDIT" matches RelTag "writing/reddit".
func TestSearchSkillsCaseInsensitive(t *testing.T) {
	got := searchSkills(searchFixture, "REDDIT")
	if len(got) != 1 || got[0].RelTag != "writing/reddit" {
		t.Fatalf("searchSkills(REDDIT)=%+v; want [writing/reddit] (case-insensitive RelTag match)", got)
	}
}

// No match: returns a nil/empty slice (caller prints "no skills match" + exit 1).
func TestSearchSkillsNoMatch(t *testing.T) {
	got := searchSkills(searchFixture, "zzz-nope-zzz")
	if len(got) != 0 {
		t.Fatalf("searchSkills(zzz-nope-zzz)=%+v; want empty (no matches)", got)
	}
}

// Preserves input order: a multi-hit query returns skills in input (RelTag-sorted)
// order — no re-sort. "i" matches all three (coding/go RelTag, misc RelTag,
// writing/reddit RelTag each contain "i").
func TestSearchSkillsPreservesOrder(t *testing.T) {
	got := searchSkills(searchFixture, "i")
	if len(got) != 3 {
		t.Fatalf("searchSkills(i)=%d matches; want 3 (one per fixture skill)", len(got))
	}
	want := []string{"coding/go", "misc", "writing/reddit"} // input order preserved
	for i, w := range want {
		if got[i].RelTag != w {
			t.Errorf("searchSkills(i)[%d].RelTag=%q; want %q (filtered slice must preserve input order)", i, got[i].RelTag, w)
		}
	}
}

// --- run: --search / -s (P1.M4.T9) ---

// --search success: a store with a matching skill -> the catalog table on stdout
// (reusing ui.PrintList), exit 0, no ANSI (non-TTY *bytes.Buffer). The query
// "showcase" matches only the keyword.
func TestRunSearchSuccess(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: A demo skill.\nkeywords:\n  - demo\n  - showcase\n---\n# body\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir) // rule 1 wins; Find() returns dir, Index finds the skill
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "showcase"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search showcase): code=%d; want 0", code)
	}
	got := out.String()
	for _, want := range []string{"TAG", "NAME", "DESCRIPTION", "example"} {
		if !strings.Contains(got, want) {
			t.Errorf("run(--search showcase) stdout missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("run(--search) on a non-TTY must not emit ANSI:\n%s", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--search) stderr=%q; want empty", errOut.String())
	}
}

func TestRunSearchShortFlag(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: demo\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-s", "demo"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-s demo): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "example") {
		t.Errorf("run(-s demo) stdout missing example:\n%s", out.String())
	}
}

// --search with NO matches: exit 1, a "no skills match" message to stderr, and
// stdout EMPTY (PRD §6.1 "1 if no matches"; §6.4 clean-stdout discipline).
func TestRunSearchNoMatchExit1(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: demo\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "nonexistent-query"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--search nonexistent-query): code=%d; want 1 (PRD §6.1 '1 if no matches')", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--search no-match) stdout=%q; want EMPTY (only stderr + exit 1)", out.String())
	}
	if !strings.Contains(errOut.String(), "no skills match") {
		t.Errorf("run(--search no-match) stderr=%q; want a 'no skills match' message", errOut.String())
	}
}

// --search when the skills dir is unresolvable -> Find() error -> exit 1, empty
// stdout, the one-line fix to stderr (same contract as --path/--list).
func TestRunSearchSkillsDirUnresolvableExit1(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // force all three §8 rules to miss
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "anything"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--search) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--search) unresolvable stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("run(--search) unresolvable stderr=%q; want the one-line fix", errOut.String())
	}
}
```

> **main_test.go import note:** the pure-filter tests reference `discover.Skill`, so
> main_test.go must import `"github.com/dabstractor/skpp/internal/discover"`. The
> current main_test.go imports are `bytes`, `io`, `os`, `path/filepath`, `strings`,
> `testing` — it does NOT yet import `discover`. ADD it to the stdlib/3rd-party
> group (goimports/gofmt will place it correctly). All other imports are already
> present. The `writeSkillTree`/`unsetSkillsEnv`/`withTerminal` helpers already
> exist at the top of main_test.go — REUSE them, do not redefine.

### Implementation Patterns & Key Details

```go
// PATTERN: parseArgs value-taking flag (index loop + i++ consume).
//   case "--search", "-s":
//       c.searchSet = true
//       if i+1 < len(args) { c.search = args[i+1]; i++ }
// WHY: a range loop cannot consume a neighbor. The index loop + manual i++ is the
//      minimal change that lets --search grab its value. The i+1<len guard prevents
//      a panic when --search is the last token. searchSet records presence so "" is
//      distinguishable from "no flag".

// PATTERN: run() mode block mirrors the --list block exactly.
//   dir, _, err := skillsdir.Find(); if err != nil { stderr; return 1 }
//   skills, err := discover.Index(dir); if err != nil { stderr; return 1 }
//   filtered := <transform>(skills)
//   if len(filtered) == 0 { stderr msg; return 1 }
//   ui.PrintList(stdout, filtered, isTerminal(stdout) && !c.noColor); return 0
// WHY: --search and --list share the Find->Index->(empty?)->PrintList skeleton; the
//      only difference is the transform (filter vs identity) and the empty message.

// PATTERN: case-insensitive substring = Contains(ToLower(field), needle).
//   needle := strings.ToLower(q)
//   strings.Contains(strings.ToLower(s.RelTag), needle)
// WHY: ToLower both sides makes the match case-insensitive (PRD §6.1). Lowercasing
//      the needle ONCE (in searchSkills) avoids redundant work per field. Pure: no
//      mutation of s.

// PATTERN: append in iteration order, do NOT re-sort.
//   for _, s := range skills { if matchesQuery(s, needle) { matched = append(matched, s) } }
// WHY: discover.Index sorts by RelTag; appending in order preserves that sort, so the
//      filtered slice is still tag-sorted for ui.PrintList. Re-sorting is wasted work.

// PATTERN: empty-result branch BEFORE the PrintList call.
//   if len(matched) == 0 { fmt.Fprintf(stderr, "no skills match %q\n", q); return 1 }
//   ui.PrintList(stdout, matched, ...)
// WHY: §6.4 clean-stdout discipline — on failure print nothing to stdout. Branching
//      before PrintList makes the empty-stdout guarantee explicit (PrintList on []
//      prints nothing anyway, but do not rely on that ordering).
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - main.go remains `package main`. NO new package. NO new directory.
  - main.go imports: fmt, io, os, strings, discover, skillsdir, ui
    (the ONLY change is adding the STDLIB "strings").
  - main.go exposes (unexported): searchSkills([]discover.Skill, string) []discover.Skill,
    matchesQuery(discover.Skill, string) bool. (parseArgs/run/config/isTerminal are
    extended in place.)

go.mod / go.sum (NO change):
  - The only new import is the STDLIB "strings". yaml.v3 is already the sole direct
    dependency. `go mod tidy` is a no-op.
  - VERIFY: `go mod tidy && git diff --quiet go.mod go.sum` exits 0.

UPSTREAM CONSUMERS (already landed; this subtask uses them read-only):
  - discover.Index(dir) ([]Skill, error)        — internal/discover/index.go
  - discover.Skill{RelTag,Name,Description,Keywords,...} — internal/discover/skill.go
  - ui.PrintList(w, skills, useColor)           — internal/ui/ui.go
  - skillsdir.Find()                            — internal/skillsdir/skillsdir.go
  - main.isTerminal / main.version              — main.go (reused unchanged)

DOWNSTREAM CONSUMERS (later subtasks plug into this):
  - P1.M5.T11 (full CLI flag matrix + exit codes): will turn `--search` mixed with
    --list/tags/--all into exit 2 (mutual exclusivity, PRD §6.3), and a missing
    --search value into exit 2. This subtask's parseArgs captures the value and sets
    searchSet so M5 can refine validation without re-architecting the wiring.
  - P1.M6.T14 (README): documents `skpp --search <q>` usage (Mode B final task).

NO CHANGES TO:
  - PRD.md, go.mod, go.sum, .gitignore, internal/* (any package), skills/ (none yet).
```

---

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
# Run after applying all edits — fix before proceeding.
gofmt -w main.go main_test.go
gofmt -l main.go main_test.go     # MUST print nothing
go vet ./...                      # MUST be clean

# Expected: zero gofmt findings, zero vet findings.
```

### Level 2: Unit Tests (Component Validation)

```bash
# The 13 new tests, plus the whole main package.
go test . -run 'Search|ParseArgsSearch|RunSearch' -v   # the new tests (parseArgs+filter+run)
go test . -v                                            # full main package (37 tests = 24 + 13)

# Whole module (regression: nothing in discover/skillsdir/ui/resolve broke).
go test ./...                                           # MUST be all ok
go test ./... -v 2>&1 | grep -c '^--- PASS'             # MUST be 118 (105 baseline + 13 new)

# Expected: all pass. If a filter test fails, check the §6.1 field set and the
# case-insensitive ToLower-on-both-sides rule. If a run() test fails, check that
# the search block is AFTER the list block and gated on c.searchSet.
```

### Level 3: Integration Testing (System Validation)

```bash
# Build the binary (with the ldflags version stamp, mirroring install.sh).
go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o /tmp/skpp .

# Seed a tiny skills store.
ROOT=$(mktemp -d)
mkdir -p "$ROOT/skills/example"
cat > "$ROOT/skills/example/SKILL.md" <<'EOF'
---
name: example
description: A demo skill for skpp.
metadata:
  keywords:
    - demo
    - showcase
  category: examples
---
# Example skill body
EOF
export SKPP_SKILLS_DIR="$ROOT/skills"

# --search hits the keyword "showcase" (keyword-only match).
/tmp/skpp --search showcase | head -3
echo "exit=$?"   # expect 0; table with TAG/NAME/DESCRIPTION header + the example row.

# --search hits the description "demo skill".
/tmp/skpp --search "demo skill" | head -3
echo "exit=$?"   # expect 0.

# --search is case-insensitive.
/tmp/skpp --search SHOWCASE | head -2
echo "exit=$?"   # expect 0.

# --search no match -> exit 1, stdout EMPTY, one stderr line.
out=$(/tmp/skpp --search zzz-nope-zzz); rc=$?
echo "stdout=[$out] exit=$rc"   # expect stdout=[] exit=1
/tmp/skpp --search zzz-nope-zsz 2>&1 >/dev/null | head -1   # expect "no skills match ..."

# -s short form works.
/tmp/skpp -s example | head -2
echo "exit=$?"   # expect 0 (matches the name "example" and the tag).

# --no-color suppresses ANSI on a TTY (force a TTY with `script` if available;
# otherwise the pipe above already proves plain output).
/tmp/skpp --no-color --search showcase | cat -v | grep -c '\^\['   # expect 0 ANSI escapes

# Skills dir unresolvable -> exit 1 + the one-line fix.
env -u SKPP_SKILLS_DIR /tmp/skpp --search anything 2>&1 >/dev/null | head -1
echo "exit=${PIPESTATUS[0]}"   # expect 1; stderr mentions SKPP_SKILLS_DIR/cd/reinstall.

cleanup() { rm -rf "$ROOT" /tmp/skpp; }
cleanup

# go.mod/go.sum unchanged (the ONLY new import is stdlib "strings").
go mod tidy && git diff --quiet go.mod go.sum && echo "go.mod/go.sum UNCHANGED"

# Expected: every command behaves as annotated.
```

### Level 4: Creative & Domain-Specific Validation

```bash
# Scope boundary check: confirm this subtask touched ONLY main.go and main_test.go,
# added NO new package, and changed NO go.mod/go.sum.
git status --porcelain                       # expect exactly:  M main.go  M main_test.go
git diff --stat                              # no other files
grep -RIn "internal/search" . 2>/dev/null    # expect NO matches (no such package)
grep -n '"strings"' main.go                  # expect the one stdlib import
grep -c 'func searchSkills\|func matchesQuery' main.go   # expect 2

# Re-run the FULL PRD §13 acceptance gate once the binary is built (this subtask
# contributes the --search rows). Not all §13 commands pass yet (later milestones),
# but the --search rows must.
# Expected: scope clean; no stray files/packages; go.mod/go.sum clean.
```

---

## Final Validation Checklist

### Technical Validation

- [ ] `gofmt -l *.go` silent; `go vet ./...` clean.
- [ ] `go build ./...` exit 0; `go test ./...` all pass (**118 tests**).
- [ ] `go mod tidy && git diff --quiet go.mod go.sum` exits 0 (UNCHANGED).
- [ ] Level 3 integration commands behave as annotated.

### Feature Validation

- [ ] `skpp --search <q>` and `skpp -s <q>` both work.
- [ ] Match is case-insensitive substring over {RelTag, Name, Description, Keywords}.
- [ ] A keyword-only query matches; a description-only query matches.
- [ ] Output is the SAME table as `--list` (reuses `ui.PrintList`), filtered + still tag-sorted.
- [ ] Zero matches ⇒ exit 1, `no skills match "<q>"` to stderr, stdout EMPTY.
- [ ] `--no-color` suppresses ANSI; color appears on a TTY (same gate as `--list`).
- [ ] Skills dir unresolvable / Index error ⇒ exit 1 + one-line fix to stderr.
- [ ] Mutual-exclusivity is NOT enforced (deferred to P1.M5.T11).

### Code Quality Validation

- [ ] Follows existing main.go / main_test.go conventions (white-box, plain assertions, no testify/Parallel).
- [ ] Filter lives in main.go (NOT a new package); reuses ui.PrintList verbatim (no PrintSearch added to ui).
- [ ] Filtered slice preserves input order (no re-sort); empty-result branches before PrintList.
- [ ] Only `"strings"` (stdlib) added; no new external dependency.
- [ ] REUSED existing helpers (writeSkillTree/unsetSkillsEnv/withTerminal), not redefined.

### Documentation & Deployment

- [ ] Code is self-documenting (doc comments on searchSkills/matchesQuery + the search block).
- [ ] No new env vars or config. No README change (DOCS: none — CLI still finalizing; item contract point 5).

---

## Anti-Patterns to Avoid

- ❌ Don't create `internal/search/` — the filter lives in main.go (item contract + architecture map).
- ❌ Don't add a `PrintSearch` to `ui` — reuse `ui.PrintList` verbatim (PRD §6.1 "same table format").
- ❌ Don't re-sort the filtered slice — `discover.Index` already sorted by RelTag; preserve order.
- ❌ Don't search Aliases/Category/Dir — §6.1 field set is EXACTLY {tag, name, description, keywords}.
- ❌ Don't make the match case-SENSITIVE — PRD §6.1 requires case-insensitive (ToLower both sides).
- ❌ Don't call `ui.PrintList` on the empty-result path — branch first so stdout stays empty (§6.4).
- ❌ Don't enforce mutual-exclusivity or missing-arg exit 2 — that is P1.M5.T11's job.
- ❌ Don't use a range loop for parseArgs — it can't consume the `--search` value (use an index loop).
- ❌ Don't gate the search block on `c.search != ""` — use `c.searchSet` (`--search ""` is a valid mode entry).
- ❌ Don't add an external import — only stdlib `"strings"`; go.mod must stay unchanged.

---

## Confidence Score

**9/10** for one-pass implementation success.

Rationale: the task is a small, well-bounded filter + wiring change to two existing
files. Every consumed contract (`discover.Index`, `discover.Skill`, `ui.PrintList`,
`main.run`/`parseArgs`/`config`/`isTerminal`) is on disk and green, with the exact
signatures quoted. The exact source for all five main.go edits and all 13 tests is
given verbatim and is gofmt-clean. The baseline (105 tests, build OK, go.mod neutral)
was measured live. The one residual risk is a copy-paste slip in the import-group
ordering or a stale assumption about a test fixture string — both caught immediately
by `go test . -v` and `gofmt -l`. The -1 is for the inherent chance of a fixture
typo in the order-preservation test (double-check the `"i"` query matches all three
fixture skills: coding/go, misc, writing/reddit each contain "i" in RelTag).
