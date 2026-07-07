# PRP — P1.M4.T2.S1: Reject conflicting listing modes + fix stale dispatch comment (Issue 6)

> **Subtask:** P1.M4.T2.S1 — fixes QA **Issue 6** (Minor, UX):
> `exclusivityError` rejects tags+mode, check+tags, check+mode but NOT mode+mode
> combos (`--list --search`, `--all --list`, `--path --list`). Dispatch order
> silently picks the first mode. decisions.md §D6: treat
> `{path, list, searchMode, all}` as mutually exclusive (any 2+ → exit 2). Also
> fix a stale dispatch-order comment.
>
> **Scope:** SURGICAL edits to `main.go` (the `exclusivityError` function body +
> doc comment, and one dispatch comment) and ADDITIVE tests in `main_test.go`
> (5 new test functions). Mode A — no docs (the exclusivity message is runtime
> output, not documented config). **No other file touched. No new import, no new
> config field, no go.mod change.**
>
> **PARALLEL CONTEXT:** P1.M4.T1.S1 (Issue 5, parseArgs normalization) is being
> implemented concurrently. It edits `parseArgs` + adds `expandShortBundle`.
> **This subtask edits `exclusivityError` + the dispatch comment — a DIFFERENT,
> non-overlapping region of main.go.** At the time of writing, the repo build is
> BROKEN mid-flight (`expandShortBundle` called in parseArgs but its definition
> not yet added by T1.S1). That breakage is T1.S1's state, NOT this subtask's
> concern. This subtask's two text edits apply cleanly regardless; its validation
> gates assume P1.M4.T1.S1 has completed (build green). See research §0.
>
> **VERIFICATION STATUS:** the exact current text of both target regions was
> captured byte-exact (research §3, §5); both are STABLE (T1.S1 does not touch
> them). The no-regression proof for all 5 existing exclusivity tests is in
> research §4. Every load-bearing fact is in `research/verified_facts.md`.

---

## Goal

**Feature Goal**: Make `exclusivityError` reject any combination of two+ listing
modes (`--path`/`--list`/`--search`/`--all`) with exit 2 and a clear message, so
`--list --search foo`, `--all --list`, `--path --list` no longer silently pick one
mode by dispatch order. Also correct the stale dispatch-order comment to match the
actual code order (`path → list → search → check → all → tags`).

**Deliverable**: Two text edits in `main.go` + 5 appended test functions in
`main_test.go`:
1. `main.go` `exclusivityError` — add a 4th family (count of
   `{c.path, c.list, c.searchMode, c.all}` ≥ 2 → error) BEFORE the existing three;
   rewrite the doc comment (now four families; remove the now-false "mode+mode
   deliberately NOT flagged" paragraph).
2. `main.go` dispatch comment — fix the order to
   `path → list → search → check → all → tags`.
3. `main_test.go` — append 5 tests (1 direct `exclusivityError` table + 4 `run`
   end-to-end, one of them table-driven over all 6 mode pairs).

**Success Definition**: `gofmt -l .` silent; `go vet ./...` clean; `go build ./...`
+ `go test ./...` pass (5 NEW test functions; 0 regressions — all 5 existing
`TestRunExclusivity*` tests still green); `go.mod`/`go.sum` byte-identical.
`--list --search foo` / `--all --list` / `--path --list` → exit 2, stderr contains
"mutually exclusive", stdout empty. Single modes and modifier+mode combos
(`--all --file`, `--list --no-color`) still work (exit 0/1 as before).

---

## Why

- **UX / predictability.** Issue 6 is Minor but real surprise: a user typing
  `--all --list` expecting `--all` (or an error) gets `--list` silently. A loud
  exit 2 is consistent with the existing tags+mode and check+mode exclusivity
  families (decisions §D6).
- **Locks the documented contract.** issue_analysis §Issue 6 names `--list --search`,
  `--all --list`, `--path --list` as the targets. This PRP makes each exit 2 and
  proves it with tests.
- **Doc hygiene.** The dispatch comment claiming `check → path → ...` is wrong
  (check is step 4). Fixing it prevents future maintainers from relying on a
  false order. Harmless today (exclusivity guarantees standalone modes) but
  misleading.
- **Zero cost.** No new flag, no new field, no new import, no exit-code/contract
  change for any currently-valid invocation. Pure tightening of an existing gate.

---

## What

`exclusivityError` gains a 4th family, checked FIRST. The existing three families
are byte-identical and stay in the same order after it.

### Behavior change (new combos rejected; everything else unchanged)

| Input | Before | After |
|---|---|---|
| `--list --search foo` | runs `--list` (exit 0/1), ignores `--search` | **exit 2**, "listing modes ... mutually exclusive" |
| `--all --list` | runs `--list`, ignores `--all` | **exit 2** |
| `--path --list` | runs `--path`, ignores `--list` | **exit 2** |
| `--path --all`, `--list --all`, `--search --all`, `--path --search` | first mode wins | **exit 2** |
| `foo --list` (tags+mode) | exit 2 "tags cannot be combined" | **unchanged** (new family sees n=1, skips) |
| `check --list` (check+mode) | exit 2 "check cannot be combined" | **unchanged** (n=1, skips; family 4 fires) |
| `--all --file` (mode+modifier) | exit 0 (prints paths) | **unchanged** (`--file` not counted) |
| `--list --no-color` | exit 0/1 (table) | **unchanged** (`--no-color` not counted) |
| `--list` (single mode) | works | **unchanged** (n=1) |

### Success Criteria

- [ ] `exclusivityError` returns `(true, "skpp: listing modes --path/--list/--search/--all are mutually exclusive")` when ≥2 of `{path,list,searchMode,all}` are set.
- [ ] `exclusivityError` returns `(false, "")` for: no modes, exactly one mode, mode+modifier.
- [ ] The new family is checked BEFORE the existing three (so `--list --search foo` gets the "listing modes" message).
- [ ] `run(["--list","--search","foo"])` → exit 2, stderr contains "mutually exclusive", stdout EMPTY.
- [ ] `run(["--all","--list"])` → exit 2; `run(["--path","--list"])` → exit 2.
- [ ] All 6 pairs of `{path,list,search,all}` exit 2 via run().
- [ ] All 5 existing `TestRunExclusivity*` tests pass unchanged.
- [ ] The dispatch comment reads `path → list → search → check → all → tags`.
- [ ] `gofmt`/`go vet`/`go build` clean; `go.mod`/`go.sum` unchanged; `git diff --name-only` ⊆ {main.go, main_test.go}.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT current text of both target regions is quoted verbatim below
(research §3, §5; captured byte-exact, stable under the parallel T1.S1 work), with
the exact old→new replacement for each. The 5 new test functions are given
verbatim. The consumed contract (`config` field names: `path`/`list`/`searchMode`/
`all`/`check`/`tags`) is confirmed. The no-regression proof for the 5 existing
exclusivity tests is explicit (research §4). An implementer who knows Go but
nothing about this repo can finish in one pass by applying the two edits and
appending the tests._

### Documentation & References

```yaml
# MUST READ — this subtask's verified facts
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/P1M4T2S1/research/verified_facts.md
  why: "§0 baseline + the parallel-build-broken caveat (T1.S1 mid-flight; NOT our
        bug). §1/§2 the bug + contract. §3 the EXACT current exclusivityError text
        (stable). §4 the no-regression proof for the 5 existing exclusivity tests.
        §5 the EXACT stale-comment text. §6 composition with T1.S1 (-la etc. become
        exit-2 after both land; do NOT add a -pl run test). §7 field names
        (searchMode, not search). §8 the 5-test plan. §9 scope."
  critical: "The set is EXACTLY {c.path, c.list, c.searchMode, c.all} — check is NOT
             in it (check+mode is family 4 below; check+tags is family 3). Modifiers
             (--file/--relative/--no-color) are NEVER counted. Put the new family
             FIRST. Remove the now-false 'mode+mode deliberately NOT flagged' paragraph."

# CONTRACT — the decision this implements
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/decisions.md
  why: "§D6: extend exclusivityError to treat {path, list, searchMode, all} as
        mutually exclusive (any 2+ → exit 2). Also fix the stale dispatch-order
        comment. READ-ONLY."
  section: "D6"

# CONTRACT — the issue root cause + fix + test impact
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/issue_analysis.md
  why: "Issue 6: root cause (exclusivityError misses mode+mode), the 4th-family fix
        over {path,list,searchMode,all}, the stale-comment fix, and the named test
        targets (--list --search, --all --list, --path --list). READ-ONLY."
  section: "Issue 6 (MINOR)"

# THE FILE BEING EDITED — exclusivityError + the dispatch comment
- file: main.go
  why: "exclusivityError (~line 530+, stable text in research §3) is the function to
        extend; the dispatch comment (~line 313, stable text in research §5) is the
        comment to fix. run() calls exclusivityError at ~line 308 (step 4, AFTER
        unknownFlag, BEFORE dispatch) — unchanged. config fields path/list/all/
        searchMode/check/tags confirmed at the config struct decl."
  pattern: "Add the count block at the TOP of exclusivityError's body (before
            hasTags); keep the three existing families byte-identical beneath it."
  gotcha: "Line numbers SHIFT because the parallel T1.M4.T1.S1 is editing parseArgs
           above. Match by TEXT (the oldText blocks below), NOT by line number."

# THE TEST FILE — existing exclusivity tests to keep green + helpers
- file: main_test.go
  why: "The 5 existing exclusivity tests (TestRunExclusivityTagsAndList/Search/All,
        TestRunExclusivityCheckAndTags/List ~line 1471-1535) set at most ONE of
        {path,list,search,all}, so the new family (n>=2) never fires for them ->
        they pass unchanged. Helpers (bytes/strings/testing) already imported."
  pattern: "package main (white-box), plain t.Errorf/t.Fatalf, table-driven where
            natural, NO testify, NO t.Parallel(). APPEND the 5 new tests; do NOT
            edit existing ones."

# CONTRACT — PRD §6.3 (mutual exclusivity scope) + §6.4 (clean stdout on exit 2)
- file: PRD.md
  why: "§6.3 scopes exclusivity to tag+mode and check+X; this subtask EXTENDS it to
        mode+mode (decisions §D6 blesses the extension). §6.4: on exit 2 print
        NOTHING to stdout (so $(...) stays clean). READ-ONLY — do NOT edit PRD.md."
  section: "6.3 Default behavior", "6.4 Error semantics"
```

### Current Codebase tree (relevant slice)

```bash
$ cd /home/dustin/projects/skpp && ls main.go main_test.go
main.go        # exclusivityError (~530) + dispatch comment (~313) — EDIT (text-stable regions)
main_test.go   # 98 tests incl. 5 TestRunExclusivity* — APPEND 5
# (internal/*, go.mod, go.sum, PRD.md, README.md — ALL UNCHANGED)
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT, ONLY)
# NOTE: build may be transiently broken by parallel P1.M4.T1.S1 (expandShortBundle
#   undefined). That is T1.S1's state; this subtask is independent of it.
```

### Desired Codebase tree (files touched)

```bash
skpp/
├── main.go        # MODIFY — exclusivityError (+4th family + doc) + dispatch comment (order fix)
└── main_test.go   # MODIFY (additive) — append 5 test functions
# (go.mod, go.sum, every other file — UNCHANGED; zero new dependency, zero new import)
```

| File | Change | Region |
|---|---|
| `main.go` | add count-block + rewrite doc comment | `exclusivityError` (~530) |
| `main.go` | fix dispatch order in comment | dispatch comment (~313, before `if c.path`) |
| `main_test.go` | append 5 test functions | end of file |

### Known Gotchas of our codebase & the fix

```go
// GOTCHA #1 — The set is EXACTLY {c.path, c.list, c.searchMode, c.all}. Do NOT
// include c.check: check+mode is already caught by the existing family
// (c.check && (c.list||c.searchMode||c.all)), and check+tags by its family. Do
// NOT include modifiers (--file/--relative/--no-color): they legitimately combine
// with a single mode (e.g. `--all --file`). The field is `searchMode` (NOT
// `search`).
//   RIGHT: for _, b := range []bool{c.path, c.list, c.searchMode, c.all} { if b { n++ } }
//   WRONG: include c.check / c.file / c.relative / c.noColor in the count.

// GOTCHA #2 — Put the new family FIRST (before hasTags). For `--list --search foo`
// (list+search, foo consumed by --search so NOT a tag), the new family fires and
// gives the precise "listing modes" message. If it were last, `foo`-bearing combos
// would hit the tags+mode family first with a less-accurate message. Order also
// keeps the existing tests' messages byte-identical (they set <=1 listing mode, so
// the new family's n>=2 never fires for them — research §4).

// GOTCHA #3 — Remove the now-FALSE doc paragraph. exclusivityError's current doc
// says "Unspecified combos (e.g. --list --search with no tags) are deliberately NOT
// flagged ... mode+mode resolves by dispatch order (list wins today)." That is now
// the OPPOSITE of the behavior — mode+mode IS flagged. Delete/rewrite it (the new
// doc lists four families and notes mode+mode is an error per Issue 6).

// GOTCHA #4 — Line numbers are UNSTABLE. The parallel P1.M4.T1.S1 is editing
// parseArgs (above exclusivityError) and adding expandShortBundle, so the line
// numbers in this PRP (~313, ~530) are approximate and WILL shift. Match the edits
// by the exact TEXT blocks below (they are stable — T1.S1 does not touch
// exclusivityError or the dispatch comment).

// GOTCHA #5 — The build may be transiently broken when you start (P1.M4.T1.S1
// mid-flight: `expandShortBundle` called but undefined). That is NOT this subtask's
// bug. Apply the two text edits regardless; `go build`/`go test` go green once
// T1.S1 lands its helper. If you must validate in isolation, the exclusivityError
// logic is independent of expandShortBundle.

// GOTCHA #6 — Do NOT add a `run(["-pl"])` / `run(["-la"])` test. Those bundled
// shorts depend on P1.M4.T1.S1's expandShortBundle having landed. Test only the
// LONG forms (--list, --search, --all, --path), which work regardless of T1.S1.
// (After both land, `-pl`→path+list→exit 2 "mutually exclusive" composes cleanly;
//  T1.S1's TestParseArgsShortBundles checks CONFIG flags only, not run() exit, so
//  no conflict. Documented in research §6, not tested here.)

// GOTCHA #7 — exclusivityError fires in run() BEFORE any dispatch (before
// Find/Index), so the new run() tests need NO store/fixtures: the filesystem is
// never touched. Keep them store-free (fast, deterministic).

// GOTCHA #8 — No new import. exclusivityError already lives in main.go; the count
// uses only existing config bool fields. `go mod tidy` is a no-op. Verify
// `git diff --quiet go.mod go.sum`.
```

---

## Implementation Blueprint

### Data models and structure

**No new data models, no new config fields, no new imports.** This reuses the
existing `config` (path/list/all/searchMode/check/tags/...) and edits one existing
function + one comment.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — extend exclusivityError (4th family + doc comment)
  - FILE: main.go, function exclusivityError (stable text in research §3).
  - EDIT: see "Edit 1". Add a count block at the TOP of the body (before hasTags)
           returning the "listing modes mutually exclusive" message when n>=2;
           rewrite the doc comment to four families and remove the false paragraph.
  - GOTCHA: set is exactly {path,list,searchMode,all}; check NOT included; new
            family FIRST; field is searchMode.

Task 2: EDIT main.go — fix the stale dispatch-order comment
  - FILE: main.go, the "// 5) Normal mode dispatch" comment (stable text in §5).
  - EDIT: see "Edit 2". Change the order to "path → list → search → check → all →
           tags" and refresh the parenthetical (exclusivity now catches mode+mode).

Task 3: EDIT main_test.go — append the 5 test functions
  - FILE: main_test.go, END of file.
  - EDIT: see "Edit 3". 1 direct exclusivityError table + 4 run() tests (one
           table-driven over all 6 mode pairs). No store needed.
  - GOTCHA: package main (white-box); plain t.Errorf; no testify; no Parallel.
            Do NOT modify existing tests; do NOT add bundled-short run tests.

Task 4: VALIDATE (all gates green; assumes P1.M4.T1.S1 complete so build is green)
  - gofmt -w main.go main_test.go; test -z "$(gofmt -l .)"
  - go vet ./...; go build ./...
  - go test . -v   # 5 new funcs + all existing incl. the 5 TestRunExclusivity*
  - go test ./...  # whole module
  - git diff --quiet go.mod go.sum   # UNCHANGED
  - Level 3 smoke (--list --search foo → exit 2; --all --list → exit 2; etc.)
  - Level 4 scope (git diff --name-only ⊆ {main.go, main_test.go})
```

### Edit 1 — `main.go` exclusivityError (Task 1)

Replace the entire current `exclusivityError` (doc comment + function) with:

**oldText** (the exact current text — research §3):

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

**newText**:

```go
// exclusivityError reports whether c combines modes that PRD §6.3 forbids,
// returning a one-line stderr message. It implements four families, checked in
// order (first hit wins):
//   - two or more listing modes among {--path, --list, --search, --all} — Issue 6
//     (any 2+ are mutually exclusive; the previous silent dispatch precedence was
//     surprising)
//   - tags + a listing mode (--list/--search/--all) — PRD §6.3 explicit
//   - check + tags — `check` ignores tags, so the combo is meaningless
//   - check + a listing mode — modes are mutually exclusive
//
// `check` is NOT in the listing-mode set: check+mode is caught by the family
// below, and check+path resolves by dispatch order (path wins) — out of scope
// here. --file/--relative/--no-color are MODIFIERS and never trigger exclusivity
// (they combine with a single mode, e.g. `--all --file`).
func exclusivityError(c config) (bad bool, msg string) {
	// Issue 6 (decisions.md §D6): any 2+ of the listing modes are mutually
	// exclusive. Count the active ones; >= 2 is an error. Checked FIRST so a
	// mode+mode combo gets the precise "listing modes" message even when tags are
	// also present. The set is exactly {path, list, searchMode, all}; check and the
	// modifiers are intentionally excluded (see the doc comment).
	n := 0
	for _, b := range []bool{c.path, c.list, c.searchMode, c.all} {
		if b {
			n++
		}
	}
	if n >= 2 {
		return true, "skpp: listing modes --path/--list/--search/--all are mutually exclusive"
	}
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

### Edit 2 — `main.go` dispatch comment (Task 2)

**oldText** (the exact current text — research §5):

```go
	// 5) Normal mode dispatch (order unchanged): check → path → list →
	//    search → all → tags. Each branch body is byte-identical to
	//    pre-M5 (check is guaranteed standalone here: exclusivity caught
	//    check+tags/check+mode).
```

**newText**:

```go
	// 5) Normal mode dispatch (order: path → list → search → check → all →
	//    tags). Each branch body is byte-identical to pre-M5 (any mode that
	//    reaches here is guaranteed standalone: exclusivityError caught
	//    mode+mode/check+tags/check+mode above).
```

> **Why this is correct:** the actual `run()` if-chain is `if c.path` → `if c.list`
> → `if c.searchMode` → `if c.check` → `if c.all` → `if len(c.tags) > 0`, i.e.
> path → list → search → check → all → tags. The old comment put `check` first,
> which was wrong. The parenthetical now also notes mode+mode is caught (Issue 6).

### Edit 3 — `main_test.go` append 5 test functions (Task 3)

Append at the END of `main_test.go`. Style mirrors the file: `package main`,
plain `t.Errorf`/`t.Fatalf`, table-driven where natural, no testify, no Parallel.
No store needed (exclusivityError fires before any dispatch).

```go
// --- exclusivityError: listing-mode mutual exclusivity (P1.M4.T2.S1, Issue 6) ---

// exclusivityError directly: 2+ listing modes → bad; exactly 1 (or none) → ok;
// modifiers never count. Locks the {path,list,search,all} set, the >=2 threshold,
// and that --file/--no-color are invisible to the check.
func TestExclusivityErrorListingModes(t *testing.T) {
	cases := []struct {
		name string
		c    config
		bad  bool
	}{
		{"none", config{}, false},
		{"only path", config{path: true}, false},
		{"only list", config{list: true}, false},
		{"only search", config{searchMode: true}, false},
		{"only all", config{all: true}, false},
		{"path+list", config{path: true, list: true}, true},
		{"path+search", config{path: true, searchMode: true}, true},
		{"path+all", config{path: true, all: true}, true},
		{"list+search", config{list: true, searchMode: true}, true},
		{"list+all", config{list: true, all: true}, true},
		{"search+all", config{searchMode: true, all: true}, true},
		{"path+list+all (3 modes)", config{path: true, list: true, all: true}, true},
		// modifiers + a single mode are NOT exclusive (modifiers don't count):
		{"all+file (modifier)", config{all: true, file: true}, false},
		{"list+noColor (modifier)", config{list: true, noColor: true}, false},
		{"path+relative (modifier)", config{path: true, relative: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad, msg := exclusivityError(tc.c)
			if bad != tc.bad {
				t.Errorf("exclusivityError(%s)=bad=%v,msg=%q; want bad=%v", tc.name, bad, msg, tc.bad)
			}
			if bad && !strings.Contains(msg, "mutually exclusive") {
				t.Errorf("(%s) msg=%q; want it to contain 'mutually exclusive'", tc.name, msg)
			}
		})
	}
}

// --- run: two listing modes → exit 2 (P1.M4.T2.S1, Issue 6) ---

// --list --search foo → exit 2 (two listing modes). No store needed:
// exclusivityError fires in run() before any dispatch, so the filesystem is
// untouched. stderr names the conflicting family; stdout stays empty (§6.4).
func TestRunExclusivityListAndSearch(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--list", "--search", "foo"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(--list --search foo): code=%d; want 2 (Issue 6)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty (§6.4)", out.String())
	}
	if !strings.Contains(errOut.String(), "mutually exclusive") {
		t.Errorf("stderr=%q; want a 'mutually exclusive' message", errOut.String())
	}
}

// --all --list → exit 2.
func TestRunExclusivityAllAndList(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--list"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(--all --list): code=%d; want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "mutually exclusive") {
		t.Errorf("stderr=%q; want a 'mutually exclusive' message", errOut.String())
	}
}

// --path --list → exit 2.
func TestRunExclusivityPathAndList(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--path", "--list"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(--path --list): code=%d; want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "mutually exclusive") {
		t.Errorf("stderr=%q; want a 'mutually exclusive' message", errOut.String())
	}
}

// All 6 pairs of {path,list,search,all} → exit 2 via run() (set-completeness
// guard). Long forms only (no bundled shorts — those depend on P1.M4.T1.S1).
func TestRunExclusivityListingModePairs(t *testing.T) {
	pairs := [][]string{
		{"--path", "--list"},
		{"--path", "--search", "x"},
		{"--path", "--all"},
		{"--list", "--search", "x"},
		{"--list", "--all"},
		{"--search", "x", "--all"},
	}
	for _, args := range pairs {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run(args, &out, &errOut)
			if code != 2 {
				t.Fatalf("run(%v): code=%d; want 2", args, code)
			}
			if out.Len() != 0 {
				t.Errorf("stdout=%q; want empty", out.String())
			}
			if !strings.Contains(errOut.String(), "mutually exclusive") {
				t.Errorf("stderr=%q; want 'mutually exclusive'", errOut.String())
			}
		})
	}
}
```

> **main_test.go import note:** these tests use only `bytes`, `strings`, `testing`
> — ALL already imported by main_test.go. No import change.

### Implementation Patterns & Key Details

```go
// PATTERN: count-then-threshold for the mode+mode family.
//   n := 0
//   for _, b := range []bool{c.path, c.list, c.searchMode, c.all} { if b { n++ } }
//   if n >= 2 { return true, "..." }
// WHY: the set is small and fixed; a count is clearer than enumerating 6 pair
//      conditions (C(4,2)). The slice literal makes the set self-documenting and
//      trivially auditable (exactly the four listing modes; check + modifiers
//      visibly absent).

// PATTERN: new family FIRST, existing three byte-identical beneath.
// WHY: the existing tests assert specific messages ("cannot be combined") for
//      tags+mode / check+X combos. Those combos set <=1 listing mode, so the new
//      n>=2 family skips them and they fall through to their original family with
//      the original message. Putting the new family first gives mode+mode combos
//      the precise "listing modes" message without disturbing the others.

// PATTERN: comment fixes are text-matched, not line-matched.
// WHY: the parallel P1.M4.T1.S1 is editing parseArgs above, shifting line numbers.
//      The exclusivityError function and the dispatch comment are text-stable
//      (T1.S1 does not touch them), so oldText matching is robust.
```

### Integration Points

```yaml
NO NEW INTEGRATION POINTS:
  - No new types, no new exports, no new config field, no new flag, no new import.
    exclusivityError's SIGNATURE is unchanged: (c config) (bad bool, msg string).
    The only change is one new return path inside it (the count block) + a doc/comment
    refresh.
  - run() is UNCHANGED (it already calls exclusivityError at step 4). The mode
    dispatch branches are UNCHANGED.
  - internal/* packages — ALL UNCHANGED. go.mod/go.sum UNCHANGED.

DOWNSTREAM / COMPOSITION:
  - P1.M4.T1.S1 (parseArgs normalization, concurrent): once it lands, bundled
    shorts like `-pl`/`-la`/`-lp` parse to 2 listing modes and then THIS subtask's
    new family rejects them (exit 2 "mutually exclusive"). Composes cleanly; no
    conflict (T1.S1's tests assert config flags, not run() exit). Not tested here
    (no coupling to T1.S1's parser).
  - P1.M5.T3.S1 (README sync): Mode A here — no README change (the exclusivity
    message is runtime output; the flag surface is unchanged).

PARALLEL-SAFETY (vs P1.M4.T1.S1):
  - This subtask edits: exclusivityError (~530) + the dispatch comment (~313) in
    main.go, and appends tests in main_test.go.
  - P1.M4.T1.S1 edits: parseArgs (~150-210) + adds expandShortBundle in main.go,
    and appends tests in main_test.go.
  - NON-OVERLAPPING regions in main.go (exclusivityError vs parseArgs). Both
    append tests at end of main_test.go — apply both append sets; order does not
    matter (distinct function names). The transient build break (expandShortBundle
    undefined) is T1.S1's in-flight state; this subtask is independent of it.
```

---

## Validation Loop

> **Precondition:** these gates require a GREEN build. If P1.M4.T1.S1 is still
> mid-flight (`go build` fails on `undefined: expandShortBundle`), that breakage is
> NOT this subtask's — apply the two text edits + tests anyway; the gates pass
> once T1.S1 lands its helper. To isolate THIS subtask's correctness in the
> meantime, the `exclusivityError` unit test (`TestExclusivityErrorListingModes`)
> exercises the logic directly (no build dependency on expandShortBundle).

### Level 1: Format, vet, build

```bash
cd /home/dustin/projects/skpp

gofmt -w main.go main_test.go
test -z "$(gofmt -l .)" || { echo "FAIL: gofmt unformatted: $(gofmt -l .)"; exit 1; }
go build ./... || { echo "FAIL: go build (if 'undefined: expandShortBundle', that is P1.M4.T1.S1 mid-flight, not this subtask)"; exit 1; }
go vet ./...    || { echo "FAIL: go vet"; exit 1; }
echo "Level 1 PASS"
```

### Level 2: Unit tests

```bash
cd /home/dustin/projects/skpp

# The new tests specifically.
go test . -v -run 'TestExclusivityErrorListingModes|TestRunExclusivityListAndSearch|TestRunExclusivityAllAndList|TestRunExclusivityPathAndList|TestRunExclusivityListingModePairs' \
  || { echo "FAIL: targeted new tests"; exit 1; }

# The 5 new functions exist.
for tn in TestExclusivityErrorListingModes TestRunExclusivityListAndSearch TestRunExclusivityAllAndList \
          TestRunExclusivityPathAndList TestRunExclusivityListingModePairs; do
  grep -q "func $tn" main_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# REGRESSION: the 5 existing exclusivity tests still pass (unchanged messages).
go test . -v -run 'TestRunExclusivity' || { echo "FAIL: existing exclusivity tests regressed"; exit 1; }

# Full main package + whole module.
go test . -v   || { echo "FAIL: go test ."; exit 1; }
go test ./...  || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS (5 new funcs; 0 regressions)"
```

### Level 3: Integration smoke (the bug-report combos, now rejected)

```bash
cd /home/dustin/projects/skpp
go build -o skpp . || { echo "FAIL: build"; exit 1; }

U="$(mktemp -d)"; mkdir -p "$U/skills/example"
printf -- '---\nname: example\ndescription: A demo skill.\n---\n\n# x\n' > "$U/skills/example/SKILL.md"
export SKPP_SKILLS_DIR="$U/skills"

# --- the three named bug-report combos now exit 2 ---
./skpp --list --search foo >/dev/null 2>&1; echo "--list --search exit=$? (want 2)"
./skpp --all --list          >/dev/null 2>&1; echo "--all --list exit=$? (want 2)"
./skpp --path --list         >/dev/null 2>&1; echo "--path --list exit=$? (want 2)"

# stderr names the family; stdout is empty.
out=$(./skpp --list --search foo 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = 2 ] && echo "stdout-empty + exit 2 OK"
./skpp --list --search foo 2>&1 >/dev/null | grep -q 'mutually exclusive' && echo "message OK"

# --- regression: single modes + modifier+mode still work ---
./skpp --list >/dev/null 2>&1; echo "--list exit=$? (want 0)"
./skpp --all --file | grep -q 'example' && echo "--all --file still works (modifier not counted)"
./skpp foo --list >/dev/null 2>&1; echo "foo --list exit=$? (want 2, tags+mode unchanged)"
./skpp check --list >/dev/null 2>&1; echo "check --list exit=$? (want 2, check+mode unchanged)"

rm -rf "$U" skpp
echo "Level 3 PASS"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# The new family is present and FIRST (before hasTags).
grep -q 'for _, b := range \[\]bool{c.path, c.list, c.searchMode, c.all}' main.go || { echo "FAIL: count block missing"; exit 1; }
grep -q 'listing modes --path/--list/--search/--all are mutually exclusive' main.go || { echo "FAIL: message missing"; exit 1; }

# The stale paragraph is GONE (mode+mode is now flagged).
! grep -q 'deliberately NOT' main.go || { echo "FAIL: stale 'deliberately NOT flagged' paragraph remains"; exit 1; }

# The dispatch comment now has the correct order.
grep -q 'order: path → list → search → check → all →' main.go || { echo "FAIL: dispatch order not fixed"; exit 1; }

# NO new import (none needed). go.mod/go.sum UNCHANGED.
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null || { echo "FAIL: go.mod/go.sum changed"; exit 1; }

# EXACTLY main.go + main_test.go changed (internal/* untouched).
git diff --quiet -- internal/ README.md PRD.md 2>/dev/null || { echo "FAIL: out-of-scope file changed"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l .` empty, `go build ./...` compiles (once T1.S1 complete), `go vet ./...` clean
- [ ] Level 2 PASS — 5 new test functions present; `go test .` green; `go test ./...` green; the 5 existing `TestRunExclusivity*` unchanged
- [ ] Level 3 PASS — `--list --search foo` / `--all --list` / `--path --list` exit 2; single modes + modifier+mode unchanged; tags+mode / check+mode unchanged
- [ ] Level 4 PASS — count block + message present; stale paragraph gone; dispatch order fixed; go.mod/go.sum unchanged; scope respected

### Feature Validation
- [ ] `exclusivityError` returns the "listing modes mutually exclusive" error for any 2+ of {path,list,searchMode,all}
- [ ] `exclusivityError` returns ok for: no modes, one mode, mode+modifier
- [ ] `--list --search foo`, `--all --list`, `--path --list` → exit 2 + "mutually exclusive" + empty stdout
- [ ] All 6 mode pairs exit 2 via run()
- [ ] tags+mode, check+tags, check+mode still exit 2 with their ORIGINAL messages (no regression)
- [ ] The dispatch comment reads `path → list → search → check → all → tags`

### Code Quality Validation
- [ ] The three existing exclusivity families are byte-identical (only the new family + doc added)
- [ ] The count uses a self-documenting `[]bool{...}` slice literal (set is auditable)
- [ ] No new config field, no new import, no new export, no signature change
- [ ] Tests mirror main_test.go style (white-box, plain assertions, table-driven, no testify/Parallel)
- [ ] Existing tests are NOT modified (only appended)

### Scope Discipline (Mode A)
- [ ] `internal/*` NOT modified
- [ ] `README.md` NOT modified (Mode A; runtime message, not documented config)
- [ ] `PRD.md` / `tasks.json` / `prd_snapshot.md` NOT modified
- [ ] `go.mod` / `go.sum` NOT modified
- [ ] `git diff --name-only` ⊆ {`main.go`, `main_test.go`}

---

## Anti-Patterns to Avoid

- ❌ **Don't include `check` or modifiers in the count set.** The set is EXACTLY
  `{c.path, c.list, c.searchMode, c.all}`. check+mode is already a separate family;
  `--file`/`--relative`/`--no-color` legitimately combine with one mode.
- ❌ **Don't put the new family after the existing three.** It must be FIRST so
  mode+mode combos get the precise "listing modes" message and the existing tests'
  messages stay byte-identical (they set ≤1 listing mode).
- ❌ **Don't leave the now-false doc paragraph.** "mode+mode deliberately NOT
  flagged" is the OPPOSITE of the new behavior — delete/rewrite it.
- ❌ **Don't edit the three existing families.** They are byte-identical before and
  after. Only the new count block (at the top) and the doc comment change.
- ❌ **Don't match edits by line number.** Line numbers shift under the parallel
  P1.M4.T1.S1 work. Match by the exact text blocks (they are stable).
- ❌ **Don't add a `run(["-pl"])` / `run(["-la"])` test.** Those depend on
  P1.M4.T1.S1's `expandShortBundle`. Test long forms only.
- ❌ **Don't "fix" the transient `undefined: expandShortBundle` build break.** That
  is P1.M4.T1.S1's in-flight state, out of this subtask's scope.
- ❌ **Don't change `go.mod`/`go.sum`, `README.md`, or any `internal/*` file.** No
  new dependency/import; README is Mode A; the fix is entirely in main.go/main_test.go.

---

## Confidence Score

**9/10** — The change is small and surgical: one count-block + doc rewrite in
`exclusivityError`, one comment order-fix, and 5 additive tests. The exact current
text of both target regions was captured byte-exact and is stable under the
parallel P1.M4.T1.S1 work (which touches a different region). The no-regression
proof for all 5 existing exclusivity tests is explicit (they set ≤1 listing mode,
so the new n≥2 family never fires for them). The consumed contract (`config` field
names) is confirmed. The one residual risk is the parallel-execution timing: if
the implementer starts while P1.M4.T1.S1 is still mid-flight, `go build` will fail
on `undefined: expandShortBundle` — but that is explicitly NOT this subtask's bug,
the two text edits apply regardless, and the `TestExclusivityErrorListingModes`
unit test exercises the logic directly without depending on the parser. The -1 is
for that timing dependency plus the inherent chance of a stale oldText if the
parallel work somehow touches exclusivityError (it does not, per its PRP scope).
