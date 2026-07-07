# Verified Facts — P1.M4.T1.S1 (`--search` substring filter + ui reuse)

Empirically confirmed against the live repo at `/home/dustin/projects/skpp`
(go1.25) BEFORE writing the PRP. Every load-bearing decision below is grounded
in a command that was actually run.

## 0. Baseline state (the world this subtask starts in)

```
$ go build ./...            # exit 0
$ go test ./... -v | grep -c '^--- PASS'
105
$ for p in . internal/discover internal/resolve internal/skillsdir internal/ui; do \
    n=$(go test ./$p/ -v 2>&1 | grep -c '^--- PASS'); echo "$p: $n"; done
.: 24                  # main (parseArgs + run: version/path/list + helpers)
internal/discover: 31  # Frontmatter + Skill + Index
internal/resolve: 10   # §7.2 precedence resolver — LANDED (parallel P1.M3.T7.S1)
internal/skillsdir: 29 # §8 dir location
internal/ui: 11        # PrintList table + ANSI
```

- `internal/resolve/resolve.go` + `resolve_test.go` ARE on disk (the parallel
  P1.M3.T7.S1 implementation landed). **This subtask does NOT depend on resolve**
  (search never calls `resolve.Resolve`); it depends only on `discover.Index`,
  `discover.Skill`, `ui.PrintList`, and `main`'s own dispatcher.
- `go.mod`: `module github.com/dabstractor/skpp`, `go 1.25`,
  `require gopkg.in/yaml.v3 v3.0.1` (the ONLY direct dependency).
- `main.go` does NOT yet import `"strings"` and has NO `search*` symbol
  (`grep` confirmed: clean slate for this subtask).

## 1. The consumed contracts (read-only; on disk, green)

- **`discover.Index(skillsDir string) ([]Skill, error)`** (`internal/discover/index.go`):
  walks the store, returns `[]Skill` sorted by `RelTag` (deterministic). This
  sort is load-bearing: `searchSkills` appends matches in iteration order, so the
  filtered slice is STILL tag-sorted — no re-sort needed.
- **`discover.Skill`** (`internal/discover/skill.go`) — the field set search touches:
  `RelTag string` (slash-normalized canonical tag), `Name string` (frontmatter
  name, `""` if absent), `Description string` (verbatim incl. possible folded-
  scalar trailing newline — substring match is unaffected), `Keywords []string`
  (nil if absent/non-list; `len()` to test). `Aliases`/`Category`/`Dir`/`HasFM`/
  `SourceFile` are NOT searched.
- **`ui.PrintList(w io.Writer, skills []discover.Skill, useColor bool)`**
  (`internal/ui/ui.go`): renders the TAG/NAME/DESCRIPTION table; empty slice →
  prints nothing (returns early). This is the EXACT reuse target — PRD §6.1 says
  `--search` uses "Same table format as `--list`".
- **`main.run` / `main.parseArgs` / `main.config` / `main.isTerminal`**
  (`main.go`): the dispatcher to extend. `parseArgs` is currently a simple
  `for _, a := range args { switch a {...} }`; ALL existing flags are boolean
  toggles, so `--search` is the FIRST value-taking flag and needs an index loop
  to consume the next token as the query.

## 2. The search field set (PRD §6.1 contract, LOCKED)

`skpp --search <q>` → substring (case-INsensitive) of `q` in ANY of:
**RelTag, Name, Description, or any of Keywords**. Match on first hit (short-
circuit). Confirmed against PRD §6.1 table row:
> `skpp --search <q>` / `-s <q>` | Substring (case-insensitive) search over tag,
> frontmatter `name`, `description`, and `metadata.keywords`.

Exit code (§6.1): `0` on match; `1` if no matches. Output: the SAME table as
`--list`, filtered (reuse `ui.PrintList`).

## 3. Where the filter logic lives — main.go (NOT a new package)

DECISION: `searchSkills` + `matchesQuery` are UNEXPORTED funcs in `main.go`,
tested from `main_test.go` (`package main`, white-box — already the convention).
Reasoning (all confirmed):
1. The item CONTRACT literally says "In **main.go** add `--search`/`-s <q>` mode.
   Build the index, then filter...".
2. `go_architecture.md` "Package map (PRD §5)" lists exactly FIVE packages
   (`main`, `internal/skillsdir`, `internal/discover`, `internal/resolve`,
   `internal/ui`). There is NO `internal/search`. Adding one would deviate from
   the documented layout.
3. Search is a LEAF: only `main`'s `--search` mode consumes it. No downstream
   subtask reuses it (T8 tag-resolution uses `resolve`; T10 check validates, not
   filters). So it needs no importable/shared type.
4. `main_test.go` is `package main` → can call unexported `searchSkills`/
   `matchesQuery` directly. Testability is preserved WITHOUT a new package.
5. It is a ~10-line filter (a loop + 4 `strings.Contains`). Contrast with
   `resolve`, which earned its own package by being non-trivial (5-step
   precedence + typed errors) AND reused by main's tag mode.

Do NOT create `internal/search/`. Do NOT extract the filter into `ui` (ui is a
pure formatter; mixing filtering in would violate its single responsibility and
the existing doc comment "It is a PURE formatter").

## 4. parseArgs becomes an index loop (FIRST value-taking flag)

`--search`/`-s` takes the NEXT token as the query. The current `for _, a :=
range args` cannot consume a neighbor, so parseArgs switches to an index loop
(`for i := 0; i < len(args); i++`) and does `i++` after capturing `args[i+1]`.
This is the ONLY structural change to parseArgs; the boolean-flag cases are
unchanged. Existing parseArgs tests (Empty/Version/Path/List/NoColor/AnyOrder/
UnknownTolerated) all still pass: an index loop over nil/empty ranges zero
times, unknown tokens hit the same `default` no-op, and the order-independence
test (`-p --version`) is unaffected.

## 5. config gains `search string` + `searchSet bool`

`searchSet` distinguishes "`--search` was given" from "no `--search`" because the
query itself may legitimately be `""`. The run() search block is gated on
`c.searchSet` (NOT on `c.search != ""`), so `--search ""` enters search mode
rather than falling through to the no-args default.

## 6. run() precedence — search slots AFTER list, BEFORE the default

Existing order: `version → path → list → default(1)`. Search is inserted as a
new block after the list block: `version → path → list → search → default(1)`.
- `--version` still wins over everything (PRD §6.3) — search block is below it.
- `--search` with no other mode → hits the search block.
- `--list --search` together → list wins (returns first). This is a DETERMINISTIC
  pre-M5 behavior; mutual-exclusivity (mixing → exit 2) is explicitly P1.M5.T11's
  job (item contract point 3). No existing test mixes these, so no regression.

## 7. Empty-result path: stderr + exit 1, stdout EMPTY

`searchSkills` returns `nil`/`[]` when nothing matches. run() then prints ONE
line to stderr and exits 1 WITHOUT calling `ui.PrintList` — so stdout stays
empty (the §6.4 "clean stdout on failure" discipline, applied to search).
Message wording (LOCKED): `no skills match %q` where `%q` is the query, e.g.
`no skills match "nonexistent-query"`. (Consistent in tone with the existing
`--list` "no skills found in <dir>" message; includes the query for usefulness.)

## 8. Empty-query edge (deferred to M5, documented, non-crashing)

`strings.Contains(x, "") == true` for every string, so `--search ""` matches ALL
skills (degenerate, ~`--list`). The item's test matrix does not include empty
query, and argument-level validation (missing/empty query → exit 2) is M5's job.
The filter is kept HONEST (no special-case): it documents this behavior in its
doc comment. `--search` as the LAST token (no value following) is guarded by an
`i+1 < len(args)` check → sets `searchSet=true`, `search=""`, no crash; full
missing-arg → exit 2 lands in M5.

## 9. `--search` consumes the NEXT token literally (even if flag-shaped)

`skpp --search --list` parses `q = "--list"` and enters search mode (query
"--list"). This is literal next-token consumption; refining "--search must not
absorb a flag as its value" is argument validation → M5. Documented as a gotcha.

## 10. go.mod / go.sum are UNCHANGED

The only new import is the STDLIB `"strings"` (for `strings.ToLower` +
`strings.Contains`). yaml.v3 is already the sole direct dependency. `go mod
tidy` is a no-op; `git diff --quiet go.mod go.sum` exits 0. If it changes
anything, the implementer added an external import they should not have.

## 11. Test convention (mirrors the repo — confirmed in main_test.go)

`package main` (white-box), plain `t.Errorf`/`t.Fatalf`, table-driven where
natural, NO testify, NO `t.Parallel()`. On-disk fixtures via the existing
`writeSkillTree(t, map[relTag]content)` helper (already in main_test.go) for
run() integration tests; pure-filter tests use `discover.Skill` struct literals
(mirrors `resolve_test.go`'s `exampleSkills`). New tests append to the END of
main_test.go; the existing `writeSkillTree`/`unsetSkillsEnv`/`withTerminal`
helpers are reused (do NOT redefine them).

## 12. Test plan (13 new tests → total 118)

parseArgs (4): SearchLong, SearchShort, SearchAnyOrder,
  SearchMissingValueNoCrash.
filter, pure (5): KeywordOnly, DescriptionMatch, CaseInsensitive, NoMatch,
  PreservesOrder.
run() integration (4): SearchSuccess (reuses table), SearchShortFlag,
  NoMatchExit1 (exit 1 + stderr + EMPTY stdout), SkillsDirUnresolvableExit1.
