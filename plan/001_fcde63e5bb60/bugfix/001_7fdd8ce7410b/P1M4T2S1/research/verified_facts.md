# Verified Facts — P1.M4.T2.S1 (Issue 6: reject conflicting listing modes)

Measured live at `/home/dustin/projects/skpp` (go1.25) on 2026-07-07. This subtask
runs IN PARALLEL with P1.M4.T1.S1 (parseArgs normalization, Issue 5). Every
load-bearing fact is here so the PRP can quote it verbatim.

## 0. Baseline & parallel-execution state

- **The repo is mid-flight on P1.M4.T1.S1.** `main.go`'s `parseArgs` already
  contains the normalization block (branch (a) `--flag=value`, branch (b) calling
  `expandShortBundle`), but the `expandShortBundle` helper DEFINITION is not yet
  present, so `go build ./...` currently FAILS:
  `./main.go:196:25: undefined: expandShortBundle`. This is P1.M4.T1.S1's
  in-flight state, NOT this subtask's concern. **This subtask's edits
  (`exclusivityError` + the dispatch comment) do NOT touch `parseArgs` or
  `expandShortBundle`** — they are fully independent. The build goes green once
  P1.M4.T1.S1 lands its helper; this subtask's validation gates assume that green
  state. Do NOT try to fix P1.M4.T1.S1's incomplete helper (out of scope).
- `main_test.go` currently has **98** `func Test` (pre-T1.S1; T1.M4.T1.S1 appends
  13 more when it lands). The 5 EXISTING exclusivity tests are:
  `TestRunExclusivityTagsAndList`, `...TagsAndSearch`, `...TagsAndAll`,
  `...CheckAndTags`, `...CheckAndList` (main_test.go ~1471-1535). They all still
  pass after this subtask (verified by reasoning in §4).
- `go.mod`: `module github.com/dabstractor/skpp`, `go 1.25`, sole direct dep
  `gopkg.in/yaml.v3 v3.0.1`. **No import change** for this subtask (exclusivityError
  already lives in main.go; the count uses only the existing `config` bool fields).
- Line numbers are UNSTABLE (the parallel implementer is actively editing main.go).
  Match edits by TEXT, not line number. The two target regions' text is STABLE
  (P1.M4.T1.S1 does not touch them) — captured byte-exact in §3 and §5.

## 1. The bug (issue_analysis §Issue 6 / decisions §D6)

`exclusivityError` rejects only THREE families: tags+mode, check+tags, check+mode.
It does NOT reject mode+mode combos (`--list --search`, `--all --list`,
`--path --list`). Dispatch order in `run()` (path > list > search > check > all >
tags) silently picks the first mode; a user typing `--all --list` gets `--list`
with no error. decisions §D6: extend exclusivityError to treat
`{path, list, searchMode, all}` as mutually exclusive with each other (any 2+ →
exit 2). Also fix the stale dispatch-order comment.

## 2. The contract (from the work item)

- Add a 4th family in `exclusivityError`, BEFORE the existing three: count how many
  of `{c.path, c.list, c.searchMode, c.all}` are true; if >= 2, return
  `(true, "skpp: listing modes --path/--list/--search/--all are mutually exclusive")`.
- The set is EXACTLY `{path, list, searchMode, all}`. `check` is NOT in it
  (check+mode is already caught by family 3; check+tags by family 2).
  `--file`/`--relative`/`--no-color` are MODIFIERS and never counted.
- Fix the stale comment so the dispatch order reads
  `path → list → search → check → all → tags` (it currently says
  `check → path → list → search → all → tags`, which is wrong: check is step 4,
  not step 1).
- Mode A: no docs (runtime error message, not documented config).

## 3. Current exact text of `exclusivityError` (STABLE; the function this edits)

```go
// exclusivityError reports whether c combines modes that PRD §6.3 forbids,
// returning a one-line stderr message. It implements EXACTLY three families:
//   - tags + a listing mode (--list/--search/--all) — PRD §6.3 explicit
//   - check + tags — `check` ignores tags, so the combo is meaningless
//   - check + a listing mode — modes are mutually exclusive
//
// Unspecified combos (e.g. --list --search with no tags) are deliberately NOT
// flagged: PRD §6.3 scopes exclusivity to tag+mode, and mode+mode-without-tags
// resolves deterministically by dispatch order (list wins today). --file/
// --relative/--no-color are MODIFIERS and never trigger exclusivity.
func exclusivityError(c config) (bad bool, msg string) {
	hasTags := len(c.tags) > 0
	if hasTags && (c.list || c.searchMode || c.all) {
		return true, "skpp: tags cannot be combined with --list/--search/--all"
	}
	if c.check && hasTags {
		return true, "skpp: 'check' cannot be combined with tag arguments"
	}
	if c.check && (c.list || c.searchMode || c.all) {
		return true, "skpp: 'check' cannot be combined with --list/--search/--all"
	}
	return false, ""
}
```

The paragraph "Unspecified combos (e.g. --list --search ...) are deliberately NOT
flagged" is now FALSE and MUST be removed (mode+mode is now flagged).

## 4. No-regression proof for the existing 5 exclusivity tests

With the new family checked FIRST:
- `foo --list` (tags+list): n=count{path,list,search,all}=1 → skip new family;
  hasTags&&list → family 2 → "tags cannot be combined". msg still contains
  "cannot be combined". ✓ (TestRunExclusivityTagsAndList asserts that substring.)
- `check foo` (check+tags): n=0; hasTags&&... → no (no mode); check&&hasTags →
  family 3 → "check cannot be combined with tag arguments". msg mentions "check". ✓
- `check --list` (check+mode): n=count=1 (list) → skip new; check&&(list) →
  family 4 → "check cannot be combined". ✓
- `foo --search q`, `foo --all`: same as tags+list. ✓

So all 5 existing tests pass unchanged. The new family only fires when n>=2
(two+ listing modes), which NONE of the existing tests trigger (they each set at
most ONE of {path,list,search,all}).

## 5. Current exact text of the stale dispatch comment (STABLE; the comment this edits)

```go
	// 5) Normal mode dispatch (order unchanged): check → path → list →
	//    search → all → tags. Each branch body is byte-identical to
	//    pre-M5 (check is guaranteed standalone here: exclusivity caught
	//    check+tags/check+mode).
```

The ACTUAL dispatch order in run() (from the top-level if-chain) is:
`if c.path` → `if c.list` → `if c.searchMode` → `if c.check` → `if c.all` →
`if len(c.tags) > 0`. So the correct order is **path → list → search → check →
all → tags**. The comment's "check → path → list → search → all → tags" puts
check first, which is wrong.

## 6. Composition with P1.M4.T1.S1 (after both land)

P1.M4.T1.S1 makes bundled shorts parse: `-pl` → path+list, `-la` → list+all,
`-lp` → list+path, etc. Once THIS subtask lands, those combos hit the new family
(n>=2) → exit 2 "mutually exclusive". Before T1.S1 they were "unknown flag" exit 2;
after both they are "mutually exclusive" exit 2 (a refinement; still exit 2). The
P1.M4.T1.S1 PRP explicitly anticipated this ("`-la` will set both and then
exclusivityError rejects the combo. No conflict."). Its `TestParseArgsShortBundles`
asserts the CONFIG flags are set (parseArgs-level), NOT a run() exit code, so it
still passes — parsing genuinely sets both flags; run()'s exclusivityError then
rejects the combo at dispatch time. No test conflict.

Do NOT add a `run(["-pl"])` test here — it couples to P1.M4.T1.S1's parsing. Test
only the LONG forms (`--list`, `--search`, `--all`, `--path`), which work
regardless of whether T1.S1 has landed.

## 7. The config field names (confirmed from main.go)

`config` struct fields: `path bool`, `list bool`, `all bool`, `searchMode bool`,
`check bool`, `tags []string`, plus `version/help/file/relative/noColor/unknownFlag`.
The count uses exactly `{c.path, c.list, c.searchMode, c.all}`. `searchMode` (NOT
`search`) is the field name — confirmed at main.go config decl and every
`c.searchMode = true` site.

## 8. Test plan (mirrors repo style: plain t.Errorf, no testify, no Parallel)

APPEND to main_test.go (package main, white-box). 5 new functions:
1. `TestExclusivityErrorListingModes` — direct call, table: single-mode (×4) ok,
   2-mode (×6) bad, 3-mode bad, modifier+mode (all+file, list+noColor) ok. Locks
   the {path,list,search,all} set, the >=2 threshold, and that modifiers don't count.
2. `TestRunExclusivityListAndSearch` — `run(["--list","--search","foo"])` exit 2,
   stderr "mutually exclusive", stdout empty.
3. `TestRunExclusivityAllAndList` — `run(["--all","--list"])` exit 2.
4. `TestRunExclusivityPathAndList` — `run(["--path","--list"])` exit 2.
5. `TestRunExclusivityListingModePairs` — run()-level table over all 6 pairs.

No store/fixtures needed: exclusivityError fires in run() BEFORE any dispatch
(before Find/Index), so the filesystem is never touched.

## 9. Scope / go.mod

Two text edits in `main.go` (exclusivityError body+doc; the dispatch comment) +
5 appended tests in `main_test.go`. NO new import, NO new config field, NO new
file, NO go.mod/go.sum change. `git diff --name-only` ⊆ {main.go, main_test.go}.
