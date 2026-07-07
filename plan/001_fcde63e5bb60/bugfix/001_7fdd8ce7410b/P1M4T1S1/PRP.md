# PRP — P1.M4.T1.S1: Normalize tokens in `parseArgs` (combined shorts + `--flag=value`)

> **Subtask:** P1.M4.T1.S1 — fixes QA **Issue 5** (Minor, UX/POSIX): `parseArgs`
> matches exact whole tokens (`switch a`), so `-vh` (combined shorts) and
> `--version=x` (`=` value syntax) fall through to the unknown-flag branch and
> exit 2. Common CLIs accept both; the PRD does not mandate them, but decisions.md
> §D5 chose to support them.
>
> **Decision:** `decisions.md §D5` — normalize each token at the TOP of the
> `parseArgs` loop body, BEFORE the existing `switch a`. Support `--flag=value`
> for all long flags (value used by `--search`, ignored for bools) and `-abc`
> combined short bool flags (expand to `-a -b -c`), with `-sVALUE` attached /
> `-s VALUE` separate value-taking forms.
>
> **Scope:** SURGICAL edits to `main.go` (a normalization block inserted before
> the switch + one new unexported `expandShortBundle` helper) and ADDITIVE tests
> in `main_test.go` (13 new test functions). Mode A — README deferred to P1.M5.T3 (the flag
> surface itself does not change; only how tokens are parsed; `usageText` already
> lists the short forms). **No other file touched. No new import, no go.mod change**
> (`main.go` already imports `"strings"`).
>
> **PARALLEL CONTEXT:** the only other in-flight item is P1.M3.T1.S1 (Issue 2,
> unicode table width), which edits `internal/ui/ui.go` + `internal/ui/ui_test.go`.
> This subtask edits ONLY `main.go` + `main_test.go` — **zero file overlap**, no
> merge risk regardless of landing order. (P1.M2.T1.S1, Issue 4 search-fields, is
> COMPLETE; its `searchMode`/`searchQ` fields and `internal/search` are the
> contract this subtask consumes.)
>
> **VERIFICATION STATUS:** baseline measured live (go1.25): `go build ./...` +
> `go vet ./...` clean; `go test ./...` = **217 tests** (98 in main); `go.mod` has
> `yaml.v3` as the sole direct dep; `main.go` already imports `"strings"`. Every
> load-bearing trace (including the two-phase-commit proof for `-vz`) is in
> `research/verified_facts.md`.

---

## Goal

**Feature Goal**: Make `parseArgs` accept POSIX combined/`=`-bearing token forms so
`-vh`, `-pl`, `--version=x`, `--search=foo`, `-sfoo`, and `-ls foo` all work, while
every existing exact-token form (`--version`, `-v`, `--search foo`, `check`, bare
tags) and every unknown-flag exit-2 path behave byte-identically to today.

**Deliverable**: Surgical edits to 2 files:
1. `main.go` — insert a 2-branch normalization block at the top of the `parseArgs`
   loop body (before `switch a`); add one unexported helper `expandShortBundle`.
2. `main_test.go` — append 13 test functions (11 `parseArgs`, 2 `run` smoke).

**Success Definition**: `gofmt -l .` silent; `go vet ./...` clean; `go build ./...`
+ `go test ./...` pass (**13 NEW test functions** added — 2 of them table-driven with
4 and 5 subtests, so ~22 new `--- PASS` lines: main 98 → ~120, module 217 → ~239);
`go.mod`/`go.sum` byte-identical. `-vh`/`-pl`/`-af` set their bool flags; `--version=x`/`--path=/x`
set bools (value ignored); `--search=foo`/`-sfoo`/`-lsfoo`/`-ls foo` set
searchMode+searchQ; `-vz` exits 2 (the whole bundle is rejected — no partial flags
leak). All pre-existing parseArgs/run tests still pass unchanged.

---

## Why

- **UX / POSIX parity.** Issue 5 is Minor but real friction: a user who types
  `-vh` or `--version=1.2.3` (habits from grep/tar/git) gets a confusing
  `unknown flag` exit 2 instead of the expected behavior. The short-flag set is
  tiny (`v h p l a f s`) and only `-s` takes a value, so the normalization is small
  and low-risk (decisions.md §D5).
- **No surface change.** The flags themselves are unchanged — no new flag, no
  renamed flag, no exit-code/contract change. `usageText` already documents the
  short forms. This is purely a parser ergonomics fix (Mode A: code + tests, no
  README).
- **Locks the documented contract.** issue_analysis §Issue 5 names `-vh`,
  `--version=x`, `--search=foo`, `-afl` as the targets. This PRP makes each work
  and proves it with tests, so the contract is enforced.
- **Zero dependency cost.** `main.go` already imports `"strings"`; the fix uses
  only `strings.HasPrefix`/`Contains`/`IndexByte`. go.mod is untouched.

---

## What

`parseArgs` gains a normalization prelude (two branches, each `continue`-ing) that
runs before the existing exact-match `switch a`. A new `expandShortBundle` helper
expands combined short flags with a **two-phase (validate-then-commit)** algorithm.

### Behavior change (new forms work; old forms unchanged)

| Input | Before | After |
|---|---|---|
| `-vh` / `-pl` / `-af` | unknown flag, exit 2 | version+help / path+list / all+file set |
| `--version=x` / `--path=/x` | unknown flag, exit 2 | version / path set (value ignored) |
| `--search=foo` | unknown flag, exit 2 | searchMode=true, searchQ="foo" |
| `-sfoo` | unknown flag, exit 2 | searchMode=true, searchQ="foo" |
| `-ls foo` | unknown flag, exit 2 | list=true, searchMode=true, searchQ="foo" |
| `-lsfoo` | unknown flag, exit 2 | list=true, searchMode=true, searchQ="foo" |
| `-vz` (unknown char) | unknown flag '-vz', exit 2 | **same** — whole bundle rejected, no flags leak |
| `--version`, `-v`, `--search foo`, `check`, `<tag>` | (works) | **unchanged** |
| `-x`, `--bogus` (unknown) | exit 2 | **unchanged** |

### Success Criteria

- [ ] `parseArgs(["-vh"])` → version=true AND help=true
- [ ] `parseArgs(["-af"])`, `["-pl"]` → the two corresponding bools set
- [ ] `parseArgs(["--version=9.9"])`, `["--path=/x"]`, `["--no-color=1"]`,
      `["--relative=yes"]` → the bool set, value IGNORED
- [ ] `parseArgs(["--search=foo"])` → searchMode=true, searchQ="foo"; value NOT a tag
- [ ] `parseArgs(["--search="])` → searchMode=true, searchQ="" (empty value)
- [ ] `parseArgs(["-sfoo"])` → searchMode=true, searchQ="foo"
- [ ] `parseArgs(["-ls","foo"])` → list=true, searchMode=true, searchQ="foo" (next arg consumed)
- [ ] `parseArgs(["-lsfoo"])` → list=true, searchMode=true, searchQ="foo"
- [ ] `parseArgs(["-vz"])` → unknownFlag="-vz" AND version=false AND help=false (WHOLESALE reject)
- [ ] `parseArgs(["-sv"])` → searchMode=true, searchQ="v" (chars after `s` are the query, not flags)
- [ ] `parseArgs(["--bogus=x"])` → unknownFlag set
- [ ] `run(["--version=1.2.3"])` → prints `skpp ...`, exit 0
- [ ] `run(["-vz"])` → exit 2 (proves the wholesale reject end-to-end)
- [ ] ALL existing parseArgs/run tests pass unchanged (217 → ~239, no regressions)
- [ ] `gofmt`/`go vet`/`go build` clean; `go.mod`/`go.sum` unchanged

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT current `parseArgs` (with file:line), the EXACT old→new text for
both edits (the normalization block and the helper), and the EXACT 13 test functions are in
the Implementation Blueprint. Every algorithmic trace (`-vh`/`-vz`/`-sfoo`/`-ls foo`/
`-lsfoo`/`-sv`/`-vs`) was hand-verified against the helper (verified_facts §4). The
critical non-obvious property — that a bundle with an unknown char must be rejected
WHOLESALE (two-phase commit) so run()'s `help→version→unknownFlag` precedence can't
mask the error — is documented as the central gotcha. An implementer who knows Go
but nothing about this repo can finish in one pass by applying the two edits and
the tests verbatim._

### Documentation & References

```yaml
# MUST READ — this subtask's verified facts (every load-bearing decision)
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/P1M4T1S1/research/verified_facts.md
  why: "§0 baseline (217 tests, build/vet clean, strings ALREADY imported → no go.mod
        change). §1 root cause (switch a matches exact tokens). §2 the two-branch fix
        surface. §3 the short-flag set (v h p l a f s; --relative/--no-color are
        LONG-ONLY, never in a bundle). §4 THE two-phase-commit algorithm + why it is
        MANDATORY (run() precedence help→version→unknownFlag; a leaked version=true
        from -vz would mask the error). §5 no-regression trace for every existing
        unknown-flag test. §6 edge cases (--search=, --bogus=x, -=x, -v=x). §7
        go.mod-neutral. §8 test convention. §9 parallel-safety vs P1.M3.T1.S1. §10
        the 14-test plan."
  critical: "Two-phase commit is NON-NEGOTIABLE. A bundle with any unknown char sets
             unknownFlag and commits NOTHING — otherwise run() prints version/help
             (exit 0) for `-vz` and masks the unknown-char error. Do NOT set bool
             flags during validation; validate fully, then commit."

# CONTRACT — the decision this implements
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/decisions.md
  why: "§D5: normalize tokens in parseArgs before the switch. Support --flag=value
        (value for --search, ignored for bools), -abc combined short bools, and
        -sVALUE attached short value. Rationale: POSIX convention, low risk, tiny
        short set. READ-ONLY."
  section: "D5"

# CONTRACT — the issue root cause + fix surface + test impact
- file: plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/issue_analysis.md
  why: "Issue 5: root cause (main.go switch a exact-match), the normalize-before-switch
        fix, the short set (v h p l a f s), and the named test targets (-vh,
        --version=x, --search=foo, -afl). READ-ONLY."
  section: "Issue 5 (MINOR)"

# THE FILE BEING EDITED — parseArgs (the switch to precede) + config + run precedence
- file: main.go
  why: "parseArgs ~line 145-210 (the index loop + switch a + default branch); the
        config struct ~line 130 (fields: version/help/path/list/all/file/relative/
        noColor/searchMode/searchQ/check/tags/unknownFlag — NO new fields); run()
        precedence ~line 232 (help → version → unknownFlag → exclusivity → dispatch).
        The existing `case \"--search\",\"-s\":` stays for the separate-value form."
  pattern: "Insert the normalization block immediately after `a := args[i]` and before
            `switch a`. Each branch ends in `continue`. Add expandShortBundle at
            package scope (next to exclusivityError/skillPath)."
  gotcha: "run() checks unknownFlag AFTER version/help. So a bundle's partial bool
           flags must NEVER be committed when it has an unknown char (two-phase).
           main.go ALREADY imports \"strings\" — do NOT add an import."

# THE TEST FILE — existing parseArgs tests to keep green + helpers to reuse
- file: main_test.go
  why: "parseArgs tests (TestParseArgs*: Version/Path/List/All/File/Help long+short,
        AnyOrderBothForms, UnknownFlagCaptured, ShortUnknownCaptured, FirstUnknownWins,
        SearchLong/Short/NoValue/ConsumesOneValue, CheckSubcommand, modifiers...).
        Helpers: withTerminal, unsetSkillsEnv, writeSkillTree, sampleStore. run()
        tests: TestRunUnknownShortFlagExit2 (-z → exit 2), TestRunVersionBeatsUnknownFlag,
        TestRunHelpBeatsUnknownFlag. All these tokens are NOT =-long or len>2 bundles,
        so they are unaffected by the new branches (verified_facts §5)."
  pattern: "package main (white-box), plain t.Errorf/t.Fatalf, table-driven where
            natural, NO testify, NO t.Parallel(). APPEND new tests; do NOT edit
            existing ones."

# URLS — the load-bearing stdlib surface (all already imported via "strings")
- url: https://pkg.go.dev/strings#IndexByte
  why: "strings.IndexByte(a, '=') finds the first '=' for the --flag=value split.
        Returns -1 if absent (we guard with Contains first, so eq >= 0 always here).
        Zero-alloc, faithful to 'split on first ='."
- url: https://pkg.go.dev/strings#HasPrefix
  why: "strings.HasPrefix(a, \"--\") distinguishes long flags; combined with the byte
        check a[1]!='-' it distinguishes single-dash bundles from double-dash longs."
- url: https://pkg.go.dev/strings#Contains
  why: "strings.Contains(a, \"=\") gates the long-flag=value branch. (A long flag
        never legitimately contains '=' except as the value separator.)"
```

### Current Codebase tree (relevant slice)

```bash
$ cd /home/dustin/projects/skpp && ls main.go main_test.go
main.go        # parseArgs (~145-210), config (~130), run (~225-...), exclusivityError, skillPath — EDIT
main_test.go   # 98 tests (parseArgs + run); helpers withTerminal/unsetSkillsEnv/writeSkillTree/sampleStore — APPEND
# (internal/{discover,resolve,search,check,skillsdir,ui}, go.mod, go.sum, PRD.md — ALL UNCHANGED)
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT, ONLY)
# main.go imports: fmt, io, os, path/filepath, strings, + internal/{check,discover,resolve,search,skillsdir,ui}
#   ("strings" is ALREADY imported — NO new import needed for this subtask.)
```

### Desired Codebase tree (files touched)

```bash
skpp/
├── main.go        # MODIFY — insert normalization block in parseArgs + add expandShortBundle helper
└── main_test.go   # MODIFY (additive) — append 13 test functions
# (go.mod, go.sum, every other file — UNCHANGED; zero new dependency, zero new import)
```

| File | Change | Approx location |
|---|---|---|
| `main.go` | insert 2-branch normalization block (before `switch a`); add `expandShortBundle` helper | parseArgs loop body ~152; new func near `exclusivityError` |
| `main_test.go` | append 13 test functions (11 parseArgs + 2 run smoke) | end of file |

### Known Gotchas of our codebase & the parsing logic

```go
// GOTCHA #1 — Two-phase commit is MANDATORY (the central correctness property).
// run() precedence is help → version → unknownFlag (main.go ~232). If a bundle like
// `-vz` set version=true AND unknownFlag="-vz", run() would hit the `version` branch
// FIRST and print the version (exit 0), MASKING the unknown-char error. So a bundle
// with ANY unknown char must be rejected WHOLESALE: validate fully FIRST, and only
// commit the bool flags if the whole bundle is valid. expandShortBundle does exactly
// this (Phase 1 validate → on unknown, set unknownFlag and return BEFORE Phase 2).
//   RIGHT: Phase1 walks chars; on unknown → unknownFlag=a, return (commit nothing).
//   WRONG: set version=true as you walk, then discover 'z' is unknown (version leaks).

// GOTCHA #2 — The bundle shape guard is `len(a) > 2 && a[0]=='-' && a[1]!='-'`.
// len>2 excludes the bare short flags ("-v","-s",... len 2) which the existing switch
// owns. a[1]!='-' excludes "--..." long flags (which hit branch (a) or the switch).
// Do NOT use `strings.HasPrefix(a,"-") && !strings.HasPrefix(a,"--")` alone — you'd
// also need the len>2 guard or "-v" would be mis-routed. The byte form is clearest.

// GOTCHA #3 — Check branch (a) [--flag=value] BEFORE branch (b) [short bundle].
// `--search=foo` starts with "--" so (a) catches it; it must NOT fall through to (b)
// (which would mis-parse it as a single-dash bundle — it doesn't, because a[1]=='-',
// but ordering (a) first is the clear intent). Each branch ends in `continue`.

// GOTCHA #4 — Once `s` is seen in a bundle, the REST of the body is the query (verbatim).
// `-sv` → searchQ="v" (the 'v' is the query, NOT a flag — version stays false). `-sfoo`
// → searchQ="foo". `-lsfoo` → list=true, searchQ="foo". The FIRST non-bool char must
// be 's' or the bundle is unknown; there is no "second non-bool" to check (parsing
// stops at 's'). `s` is effectively "last among flags" by construction.

// GOTCHA #5 — `s` with NO value anywhere → searchMode stays FALSE (mirror the bare
// `-s`-no-value rule in the main switch). `-vs` as the last token → version=true,
// searchMode=false (s had no remainder and no next arg). The bools before s ARE set.
// Do NOT set searchMode=true-with-empty-query here; that diverges from the existing
// `-s`-alone behavior. (Observable run() result is the same since version wins, but
// the config should be internally consistent.)

// GOTCHA #6 — Bool long flags IGNORE the `=value`. `--version=x`, `--path=/x`,
// `--no-color=1`, `--relative=yes` all just set the bool; the value is discarded.
// ONLY `--search` consumes the value (as searchQ). `--search=` (empty value) is
// valid → searchMode=true, searchQ="" (matches the existing empty-query behavior).

// GOTCHA #7 — Unknown long=`--bogus=x` reports the WHOLE token as unknownFlag
// ("--bogus=x"), consistent with the existing "report what the user typed" pattern
// (c.unknownFlag = a). Do NOT strip to "--bogus". The test asserts unknownFlag is
// non-empty (set), not an exact string, but keep the whole-token convention.

// GOTCHA #8 — `--relative` and `--no-color` are LONG-ONLY (no short alias). They can
// NEVER appear in a bundle. The bundle's valid chars are EXACTLY {v,h,p,l,a,f} plus a
// trailing `s`+query. (issue_analysis §Issue 5 confirms the short set.) So `-rn` or
// `-nc` are NOT valid bundles — 'r'/'n' are unknown → unknownFlag.

// GOTCHA #9 — main.go ALREADY imports "strings". The fix uses strings.HasPrefix,
// strings.Contains, strings.IndexByte — all in the existing import. Do NOT add an
// import line. go.mod/go.sum are UNCHANGED. Verify `git diff --quiet go.mod go.sum`.

// GOTCHA #10 — Do NOT touch the existing switch or the default branch. They still
// handle `-v`/`-s` (len 2), `--search foo` (separate value), `check`, bare tags, and
// unknowns (`-x`, `--bogus`). The normalization is ADDITIVE: new shapes are caught
// before the switch; everything else falls through to the switch unchanged. Verified
// no-regression for every existing unknown-flag test (verified_facts §5).

// GOTCHA #11 — `consumeNext` from expandShortBundle tells the caller to `i++`. This is
// the SAME index-loop technique the existing `case "--search","-s":` uses (i++ to skip
// the consumed value). The helper returns consumeNext=true ONLY when `-s` took its
// value from the NEXT argv token (remainder empty, next exists). Embedded values
// (`-sfoo`) do NOT consume (consumeNext=false).

// GOTCHA #12 — This runs IN PARALLEL with P1.M3.T1.S1, which edits internal/ui/ui.go +
// internal/ui/ui_test.go. You edit ONLY main.go + main_test.go. Zero overlap.
```

---

## Implementation Blueprint

### Data models and structure

**No new data models, no new config fields.** This reuses the existing `config`
(version/help/path/list/all/file/relative/noColor/searchMode/searchQ/check/tags/
unknownFlag) and the existing `strings` import. The only new symbol is one
unexported helper:

```go
// expandShortBundle parses a combined short-flag token (e.g. "-vh", "-sfoo") and
// applies the flags to *c. Two-phase (validate-then-commit); see GOTCHA #1.
func expandShortBundle(c *config, a string, args []string, i int) (consumeNext, ok bool)
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — insert the normalization block in parseArgs
  - FILE: main.go, inside parseArgs's `for` loop, immediately after `a := args[i]`
           and BEFORE `switch a {`.
  - EDIT: see "Edit 1". Two branches (long `=`, short bundle), each ending in
           `continue`. Uses strings.HasPrefix/Contains/IndexByte (already imported).
  - GOTCHA: branch (a) before (b); the byte-form bundle guard len(a)>2 && a[0]=='-'
            && a[1]!='-'; bool longs ignore the value; --search takes it; unknown
            long→ unknownFlag = a (whole token).

Task 2: EDIT main.go — add the expandShortBundle helper
  - FILE: main.go, at package scope (place it right after parseArgs, or near
           exclusivityError/skillPath — any package-scope spot).
  - EDIT: see "Edit 2". Two-phase: Phase 1 validate (find sIdx or unknown → reject
           wholesale); Phase 2 commit bools [0,sIdx); then handle s (remainder /
           next arg / no-value). Returns (consumeNext, ok).
  - GOTCHA: on unknown char, set unknownFlag and return BEFORE Phase 2 (no partial
            commit). s with no value anywhere → searchMode stays false.

Task 3: EDIT main_test.go — append the 13 test functions
  - FILE: main_test.go, END of file.
  - EDIT: see "Edit 3". 12 parseArgs tests (table-driven where natural) + 2 run()
           smoke tests. Reuse sampleStore/writeSkillTree where a store is needed.
  - GOTCHA: package main (white-box); plain t.Errorf; no testify; no Parallel.
            Do NOT modify existing tests.

Task 4: VALIDATE (all gates green)
  - gofmt -w main.go main_test.go
  - test -z "$(gofmt -l .)"
  - go vet ./...   # clean
  - go build ./... # compiles
  - go test . -v   # main package: 98 → ~120 (13 new funcs, 2 table-driven)
  - go test ./...  # whole module: 217 → ~239
  - git diff --quiet go.mod go.sum   # UNCHANGED (no new import)
  - Level 3 smoke (CLI: -vh, --version=x, --search=foo, -sfoo, -vz→exit2)
  - Level 4 scope check (git diff --name-only == exactly main.go + main_test.go)
```

### Edit 1 — `main.go` normalization block (Task 1)

Locate the start of the `parseArgs` loop body. It currently looks like:

```go
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--version", "-v":
```

Insert the normalization block between `a := args[i]` and `switch a {`:

```go
	for i := 0; i < len(args); i++ {
		a := args[i]

		// Issue 5 (decisions.md §D5): normalize combined / '='-bearing tokens
		// BEFORE the exact-match switch so POSIX forms work. Each branch ends in
		// `continue`; the switch below still handles the original exact-token forms
		// (--version, -v, --search <q>, check, bare tags, and unknowns like -x).

		// (a) Long flag with '=': --flag=value. Split on the FIRST '='; bool flags
		// ignore the value (--version=x == --version), --search takes it as the
		// query, an unknown name is an unknown flag (the whole token is reported).
		if strings.HasPrefix(a, "--") && strings.Contains(a, "=") {
			eq := strings.IndexByte(a, '=')
			name, val := a[:eq], a[eq+1:]
			switch name {
			case "--version":
				c.version = true
			case "--help":
				c.help = true
			case "--path":
				c.path = true
			case "--list":
				c.list = true
			case "--all":
				c.all = true
			case "--file":
				c.file = true
			case "--relative":
				c.relative = true
			case "--no-color":
				c.noColor = true
			case "--search":
				c.searchMode = true
				c.searchQ = val
			default:
				if c.unknownFlag == "" {
					c.unknownFlag = a
				}
			}
			continue
		}

		// (b) Short bundle: -xyz (single '-', not "--", len > 2). Expand into the
		// individual short flags; -s (value-taking) may consume the next token.
		// len-2 shorts ("-v", "-s", ...) and "--..." longs fall through to the switch.
		if len(a) > 2 && a[0] == '-' && a[1] != '-' {
			if consumeNext, _ := expandShortBundle(&c, a, args, i); consumeNext {
				i++ // -s took its value from the next argv token
			}
			continue
		}

		switch a {
		case "--version", "-v":
			// ... (existing switch body unchanged) ...
```

### Edit 2 — `main.go` expandShortBundle helper (Task 2)

Add this function at package scope in `main.go` (e.g. immediately after the closing
`}` of `parseArgs`):

```go
// expandShortBundle parses a combined short-flag token `a` (e.g. "-vh", "-pl",
// "-sfoo", "-ls") and applies the resulting flags to *c. It implements Issue 5's
// short-bundle normalization (decisions.md §D5). The caller has already guaranteed
// `a` is bundle-shaped: a single leading '-', not "--", and len(a) > 2.
//
// Semantics (PRD §6 short forms; the short set is exactly v h p l a f s):
//   - v/h/p/l/a/f are BOOL flags; each sets its config field.
//   - s is the VALUE-TAKING flag (--search): once seen, the rest of the body is
//     the query (e.g. "-sfoo" -> "foo"); if the rest is empty, the NEXT argv
//     token is consumed as the query (e.g. "-ls foo" -> list + query "foo"), and
//     the caller advances i (returns consumeNext=true). If no value is available
//     at all (empty rest AND no next arg), searchMode stays false — mirroring the
//     bare "-s"-with-no-value rule in the main switch.
//   - any char that is NEITHER a bool flag NOR the leading 's' is UNKNOWN: the
//     WHOLE bundle is rejected — c.unknownFlag is set to `a` and NOTHING is
//     applied. This two-phase (validate-then-commit) design is REQUIRED because
//     run() checks unknownFlag AFTER version/help: a leaked `version=true` from a
//     partial "-vz" would make run() print the version (exit 0) and mask the
//     unknown-char error.
//
// Returns (consumeNext, ok). ok is always true for a bundle-shaped token (it was
// handled, validly or as-unknown). consumeNext=true tells the caller to i++ (the
// -s value came from the next argv token).
func expandShortBundle(c *config, a string, args []string, i int) (consumeNext, ok bool) {
	body := a[1:] // strip the single leading '-'

	// Phase 1 — validate. Walk bool flags left-to-right; the FIRST non-bool char
	// must be 's' (then the rest is the query) or it is unknown. Record where 's'
	// sits (sIdx) so Phase 2 knows where flags end and the query begins.
	sIdx := -1
	for j := 0; j < len(body); j++ {
		ch := body[j]
		if ch == 's' {
			sIdx = j
			break // 's' ends flag parsing; body[j+1:] is the query
		}
		switch ch {
		case 'v', 'h', 'p', 'l', 'a', 'f':
			// valid bool short flag (validated here; applied in Phase 2)
		default:
			// Unknown char: reject the WHOLE bundle. Commit nothing (two-phase).
			if c.unknownFlag == "" {
				c.unknownFlag = a
			}
			return false, true
		}
	}

	// Phase 2 — commit the bool flags in [0, sIdx) (or the whole body if no 's').
	end := len(body)
	if sIdx >= 0 {
		end = sIdx
	}
	for j := 0; j < end; j++ {
		switch body[j] {
		case 'v':
			c.version = true
		case 'h':
			c.help = true
		case 'p':
			c.path = true
		case 'l':
			c.list = true
		case 'a':
			c.all = true
		case 'f':
			c.file = true
		}
	}

	// Handle the value-taking 's' if it was present.
	if sIdx >= 0 {
		remainder := body[sIdx+1:]
		switch {
		case remainder != "":
			c.searchMode = true
			c.searchQ = remainder // value embedded in the bundle ("-sfoo")
		case i+1 < len(args):
			c.searchMode = true
			c.searchQ = args[i+1] // value is the next argv token ("-ls foo")
			return true, true      // caller advances i
		default:
			// 's' seen but no value anywhere: mirror the bare "-s"-no-value rule
			// (searchMode stays false). The bool flags before it remain set.
		}
	}
	return false, true
}
```

> **Copy-paste correctness:** both edits are gofmt-clean. main.go's `strings` import
> already covers HasPrefix/Contains/IndexByte. Every trace (`-vh`/`-vz`/`-sfoo`/
> `-ls foo`/`-lsfoo`/`-sv`/`-vs`/`--search=`/`--bogus=x`) was hand-verified against
> this exact source (verified_facts §4/§6).

### Edit 3 — `main_test.go` append 13 test functions (Task 3)

Append at the END of `main_test.go`. Style mirrors the file: `package main`,
plain `t.Errorf`/`t.Fatalf`, table-driven where natural, no testify, no Parallel.
Reuse `sampleStore`/`writeSkillTree` where a store is needed.

```go
// --- parseArgs: combined short flags + --flag=value (P1.M4.T1.S1, Issue 5) ---

// Combined short BOOL bundles expand into their individual flags.
func TestParseArgsShortBundles(t *testing.T) {
	cases := []struct {
		name string
		args []string
	 chk func(*testing.T, config)
	}{
		{"-vh", []string{"-vh"}, func(t *testing.T, c config) {
			if !c.version || !c.help {
				t.Errorf("-vh: version=%v help=%v; want true,true", c.version, c.help)
			}
		}},
		{"-af", []string{"-af"}, func(t *testing.T, c config) {
			if !c.all || !c.file {
				t.Errorf("-af: all=%v file=%v; want true,true", c.all, c.file)
			}
		}},
		{"-pl", []string{"-pl"}, func(t *testing.T, c config) {
			if !c.path || !c.list {
				t.Errorf("-pl: path=%v list=%v; want true,true", c.path, c.list)
			}
		}},
		{"-fl", []string{"-fl"}, func(t *testing.T, c config) {
			if !c.file || !c.list {
				t.Errorf("-fl: file=%v list=%v; want true,true", c.file, c.list)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := parseArgs(tc.args)
			tc.chk(t, c)
			// A pure-bool bundle must NOT trip unknownFlag or capture a tag.
			if c.unknownFlag != "" {
				t.Errorf("%s: unknownFlag=%q; want empty", tc.name, c.unknownFlag)
			}
		})
	}
}

// Long --flag=value: bool flags IGNORE the value (PRD §6 / decisions.md §D5).
func TestParseArgsLongEqualsBoolFlags(t *testing.T) {
	cases := []struct {
		arg  string
		chk  func(*testing.T, config)
	}{
		{"--version=9.9", func(t *testing.T, c config) {
			if !c.version {
				t.Errorf("--version=9.9: version=false; want true (value ignored)")
			}
		}},
		{"--path=/x", func(t *testing.T, c config) {
			if !c.path {
				t.Errorf("--path=/x: path=false; want true (value ignored)")
			}
		}},
		{"--no-color=1", func(t *testing.T, c config) {
			if !c.noColor {
				t.Errorf("--no-color=1: noColor=false; want true (value ignored)")
			}
		}},
		{"--relative=yes", func(t *testing.T, c config) {
			if !c.relative {
				t.Errorf("--relative=yes: relative=false; want true (value ignored)")
			}
		}},
		{"--help=anything", func(t *testing.T, c config) {
			if !c.help {
				t.Errorf("--help=anything: help=false; want true (value ignored)")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.arg, func(t *testing.T) {
			c := parseArgs([]string{tc.arg})
			tc.chk(t, c)
			if c.unknownFlag != "" {
				t.Errorf("%s: unknownFlag=%q; want empty", tc.arg, c.unknownFlag)
			}
		})
	}
}

// --search=foo sets searchMode + captures the value (which is NOT a tag).
func TestParseArgsLongEqualsSearch(t *testing.T) {
	c := parseArgs([]string{"--search=foo"})
	if !c.searchMode || c.searchQ != "foo" {
		t.Errorf("--search=foo: mode=%v q=%q; want true,foo", c.searchMode, c.searchQ)
	}
	if len(c.tags) != 0 {
		t.Errorf("--search value leaked into tags: %v", c.tags)
	}
}

// --search= (empty value) is valid -> searchMode=true, searchQ="".
func TestParseArgsLongEqualsSearchEmpty(t *testing.T) {
	c := parseArgs([]string{"--search="})
	if !c.searchMode || c.searchQ != "" {
		t.Errorf("--search=: mode=%v q=%q; want true,\"\"", c.searchMode, c.searchQ)
	}
}

// --bogus=x (unknown long with '=') -> unknownFlag set (the whole token).
func TestParseArgsLongEqualsUnknown(t *testing.T) {
	c := parseArgs([]string{"--bogus=x"})
	if c.unknownFlag == "" {
		t.Errorf("--bogus=x: unknownFlag empty; want set (whole token reported)")
	}
}

// -sfoo (attached short value) -> searchMode=true, searchQ="foo".
func TestParseArgsShortAttachedSearch(t *testing.T) {
	c := parseArgs([]string{"-sfoo"})
	if !c.searchMode || c.searchQ != "foo" {
		t.Errorf("-sfoo: mode=%v q=%q; want true,foo", c.searchMode, c.searchQ)
	}
}

// -ls foo (bundle ending in -s, value from the NEXT arg) -> list + search "foo".
func TestParseArgsShortBundleSearchNextArg(t *testing.T) {
	c := parseArgs([]string{"-ls", "foo"})
	if !c.list {
		t.Errorf("-ls foo: list=false; want true")
	}
	if !c.searchMode || c.searchQ != "foo" {
		t.Errorf("-ls foo: mode=%v q=%q; want true,foo", c.searchMode, c.searchQ)
	}
	if len(c.tags) != 0 {
		t.Errorf("-ls foo: 'foo' leaked into tags: %v", c.tags)
	}
}

// -lsfoo (bundle with attached -s value) -> list + search "foo".
func TestParseArgsShortBundleSearchAttached(t *testing.T) {
	c := parseArgs([]string{"-lsfoo"})
	if !c.list || !c.searchMode || c.searchQ != "foo" {
		t.Errorf("-lsfoo: list=%v mode=%v q=%q; want true,true,foo", c.list, c.searchMode, c.searchQ)
	}
}

// -vz (unknown char in a bundle): the WHOLE bundle is rejected — unknownFlag is
// set AND version/help are NOT leaked. (Two-phase commit; run() precedence would
// otherwise mask the error. See verified_facts §4.)
func TestParseArgsShortBundleUnknownCharRejectsWhole(t *testing.T) {
	c := parseArgs([]string{"-vz"})
	if c.unknownFlag == "" {
		t.Errorf("-vz: unknownFlag empty; want set (whole bundle rejected)")
	}
	if c.version {
		t.Errorf("-vz: version=true; want false (wholesale reject — no partial commit)")
	}
	if c.help {
		t.Errorf("-vz: help=true; want false (wholesale reject)")
	}
}

// -vs as the LAST token (s present, no value anywhere): the bool before s is set,
// searchMode stays false (mirrors the bare -s-no-value rule).
func TestParseArgsShortBundleSearchNoValue(t *testing.T) {
	c := parseArgs([]string{"-vs"})
	if !c.version {
		t.Errorf("-vs: version=false; want true (bool before s is committed)")
	}
	if c.searchMode {
		t.Errorf("-vs: searchMode=true; want false (s had no value -> stays inactive)")
	}
}

// -sv: once s is seen, the rest of the body is the query — so 'v' is the QUERY,
// not a flag (version stays false).
func TestParseArgsShortBundleSConsumesRestAsQuery(t *testing.T) {
	c := parseArgs([]string{"-sv"})
	if !c.searchMode || c.searchQ != "v" {
		t.Errorf("-sv: mode=%v q=%q; want true,\"v\" (rest after s is the query)", c.searchMode, c.searchQ)
	}
	if c.version {
		t.Errorf("-sv: version=true; want false (v after s is query, not a flag)")
	}
}

// --- run: combined shorts + --flag=value smoke (P1.M4.T1.S1) ---

// --version=1.2.3 end-to-end: prints the version line, exit 0.
func TestRunLongEqualsVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--version=1.2.3"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--version=1.2.3): code=%d; want 0", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want 'skpp <version>\\n' (value ignored)", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty", errOut.String())
	}
}

// -vz end-to-end: exit 2 (proves the wholesale reject — version does NOT mask the
// unknown char, because version was never committed).
func TestRunShortBundleUnknownExit2(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-vz"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(-vz): code=%d; want 2 (unknown char, wholesale reject)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (§6.4: nothing on stdout on exit-2)", out.String())
	}
	if !strings.Contains(errOut.String(), "unknown flag") {
		t.Errorf("stderr=%q; want an 'unknown flag' message", errOut.String())
	}
}
```

### Implementation Patterns & Key Details

```go
// PATTERN: normalize-then-continue, leaving the exact-match switch intact.
//   a := args[i]
//   if strings.HasPrefix(a, "--") && strings.Contains(a, "=") { ...; continue }   // (a)
//   if len(a) > 2 && a[0] == '-' && a[1] != '-' { ...; continue }                  // (b)
//   switch a { ... }                                                               // existing
// WHY: the switch already handles every exact-token form correctly. The new code
//      only intercepts the NEW shapes (=long, len>2 single-dash bundle) and routes
//      them; everything else falls through unchanged. This is the smallest possible
//      change and it is provably non-regressive (verified_facts §5).

// PATTERN: two-phase (validate-then-commit) for bundles.
//   Phase 1: walk chars; record sIdx or reject-on-unknown (return BEFORE committing).
//   Phase 2: commit bools [0, sIdx); then handle s.
// WHY: run() precedence is help → version → unknownFlag. A leaked version=true from
//      a partial "-vz" would print version (exit 0) and hide the unknown-char error.
//      Validating fully before committing any flag guarantees the unknown path wins.

// PATTERN: -s value resolution (remainder → next arg → none).
//   remainder := body[sIdx+1:]
//   switch { case remainder != "": ...; case i+1 < len(args): ...; consumeNext=true;
//             default: searchMode stays false }
// WHY: mirrors the existing `case "--search","-s":` (which consumes the next arg, or
//      leaves searchMode false if there is none). Embedded ("-sfoo"), separate
//      ("-ls foo"), and missing ("-vs" last) are the three cases.

// PATTERN: bool long flags ignore the =value; only --search consumes it.
//   case "--version": c.version = true   // val discarded
//   case "--search":  c.searchMode = true; c.searchQ = val
// WHY: decisions.md §D5. A bool flag has no value to take; --search is the only
//      value-taking long flag. `--search=` (empty) is a valid empty query.
```

### Integration Points

```yaml
NO NEW INTEGRATION POINTS:
  - No new types, no new exports beyond expandShortBundle (lowercase, package-private),
    no new config field, no new flag, no exit-code change, no stdout/stderr contract
    change. The CLI surface (flags, usageText, exit codes) is UNCHANGED — only the
    parser accepts more token SHAPES for the existing flags.
  - run(), exclusivityError(), skillPath(), and every mode dispatch are UNCHANGED.
    The new forms populate the SAME config fields the existing modes already read.
  - internal/{discover,resolve,search,check,skillsdir,ui} — ALL UNCHANGED.
  - go.mod/go.sum UNCHANGED (strings is already imported; no new module).

DOWNSTREAM:
  - P1.M4.T2.S1 (Issue 6, reject conflicting listing modes) extends exclusivityError;
    it reads the SAME config fields this subtask populates, so `-la` (list+all) will
    set both and then exclusivityError rejects the combo. No conflict — this subtask
    just makes `-la` parse; T2.S1 makes it exit 2. They compose cleanly.
  - P1.M5.T3.S1 (README sync) is Mode B; no flag-surface change here to document.

PARALLEL-SAFETY (vs P1.M3.T1.S1, running concurrently):
  - Files touched by THIS subtask: main.go, main_test.go.
  - Files touched by P1.M3.T1.S1: internal/ui/ui.go, internal/ui/ui_test.go.
  - ZERO overlap. Both apply cleanly regardless of landing order.
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate)

```bash
cd /home/dustin/projects/skpp

gofmt -w main.go main_test.go
test -z "$(gofmt -l .)" || { echo "FAIL: gofmt unformatted: $(gofmt -l .)"; exit 1; }
go build ./... || { echo "FAIL: go build"; exit 1; }
go vet ./...    || { echo "FAIL: go vet"; exit 1; }
echo "Level 1 PASS"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# The new parseArgs/run tests specifically.
go test . -v -run 'TestParseArgsShortBundle|TestParseArgsLongEquals|TestParseArgsShortAttachedSearch|TestParseArgsShortBundleSearch|TestRunLongEqualsVersion|TestRunShortBundleUnknown' \
  || { echo "FAIL: targeted new tests"; exit 1; }

# The 13 new functions exist (name-level guard, independent of subtest counting).
test "$(grep -cE '^func TestParseArgsShortBundle|^func TestParseArgsLongEquals|^func TestParseArgsShortAttachedSearch|^func TestRunLongEqualsVersion|^func TestRunShortBundleUnknownExit2' main_test.go)" -ge 13 \
  || { echo "FAIL: expected >=13 new test functions"; exit 1; }

# Full main package (regression: the 98 existing tests still pass alongside the new).
go test . -v || { echo "FAIL: go test ."; exit 1; }

# Whole module (regression guard across ui/search/resolve/check/discover/skillsdir —
# especially important since P1.M3.T1.S1 may have landed ui changes).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS (13 new funcs; main ~120, module ~239 incl. table subtests)"
```

### Level 3: Integration smoke test (the bug-report forms, now working)

```bash
cd /home/dustin/projects/skpp
go build -o skpp . || { echo "FAIL: build"; exit 1; }

U="$(mktemp -d)"
mkdir -p "$U/skills/example"
printf -- '---\nname: example\ndescription: A demo skill.\n---\n\n# x\n' > "$U/skills/example/SKILL.md"
export SKPP_SKILLS_DIR="$U/skills"

# --- combined short bools ---
./skpp -vh >/dev/null 2>&1; echo "-vh exit=$? (want 0: help wins)"
./skpp -pl 2>&1 | grep -q 'found via' && echo "-pl ran --path OK"

# --- --flag=value ---
./skpp --version=9.9 | grep -q '^skpp ' && echo "--version=9.9 OK (value ignored)"

# --- --search=foo / -sfoo / -ls foo ---
./skpp --search=example | grep -q 'example' && echo "--search=example OK"
./skpp -sexample | grep -q 'example' && echo "-sexample OK"
./skpp -ls example 2>&1 | grep -q 'example' && echo "-ls example OK (list+search)"

# --- -vz -> exit 2 (unknown char, wholesale reject) ---
./skpp -vz >/dev/null 2>&1; echo "-vz exit=$? (want 2: unknown char rejected)"
./skpp -vz 2>&1 | grep -q 'unknown flag' && echo "-vz reports unknown flag OK"

# --- regression: existing exact forms unchanged ---
./skpp --version | grep -q '^skpp ' && echo "--version still OK"
./skpp --search example | grep -q 'example' && echo "--search <q> still OK"
env -u SKPP_SKILLS_DIR ./skpp --bogus >/dev/null 2>&1; echo "--bogus exit=$? (want 2: unknown still rejected)"

rm -rf "$U" skpp
echo "Level 3 PASS"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# expandShortBundle helper exists (package-private).
grep -q 'func expandShortBundle(' main.go || { echo "FAIL: helper missing"; exit 1; }

# The normalization block is present before the switch (both branches).
grep -q 'strings.HasPrefix(a, "--") && strings.Contains(a, "=")' main.go || { echo "FAIL: branch (a) missing"; exit 1; }
grep -q 'len(a) > 2 && a\[0\] == .-. && a\[1\] != .-.' main.go || { echo "FAIL: branch (b) missing"; exit 1; }

# Two-phase commit: the unknown-char path returns BEFORE Phase 2 (no partial commit).
grep -q 'Commit nothing (two-phase)' main.go || { echo "FAIL: two-phase rationale comment missing"; exit 1; }

# NO new import was added (strings was already imported). Confirm the import count
# is unchanged and "strings" is present exactly once.
test "$(grep -c '\"strings\"' main.go)" -eq 1 || { echo "FAIL: strings import duplicated/missing"; exit 1; }

# The 13 new test functions are present.
for tn in TestParseArgsShortBundles TestParseArgsLongEqualsBoolFlags TestParseArgsLongEqualsSearch \
          TestParseArgsLongEqualsSearchEmpty TestParseArgsLongEqualsUnknown TestParseArgsShortAttachedSearch \
          TestParseArgsShortBundleSearchNextArg TestParseArgsShortBundleSearchAttached \
          TestParseArgsShortBundleUnknownCharRejectsWhole TestParseArgsShortBundleSearchNoValue \
          TestParseArgsShortBundleSConsumesRestAsQuery TestRunLongEqualsVersion TestRunShortBundleUnknownExit2; do
  grep -q "func $tn" main_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# go.mod / go.sum UNCHANGED (no new import, no new module).
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null || { echo "FAIL: go.mod/go.sum changed"; git diff go.mod go.sum; exit 1; }

# EXACTLY 2 files changed — nothing else.
git diff --quiet -- internal/ui/ui.go internal/ui/ui_test.go 2>/dev/null \
  || { echo "NOTE: ui files changed (likely P1.M3.T1.S1 landed — expected, not this subtask)"; }
git diff --quiet -- internal/check/check.go internal/discover/skill.go internal/resolve/resolve.go \
  internal/search/search.go internal/skillsdir/skillsdir.go 2>/dev/null \
  || { echo "FAIL: an internal package changed (out of scope)"; exit 1; }
git diff --quiet -- README.md PRD.md || { echo "FAIL: README/PRD changed (out of scope; Mode A, PRD read-only)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l .` empty, `go build ./...` compiles, `go vet ./...` clean
- [ ] Level 2 PASS — 13 new test functions present; `go test .` green (98 → ~120); `go test ./...` green (217 → ~239, incl. table subtests)
- [ ] Level 3 PASS — `-vh`/`-pl`/`--version=x`/`--search=foo`/`-sfoo`/`-ls foo` work; `-vz` exit 2; exact forms unchanged
- [ ] Level 4 PASS — helper + both branches present; two-phase comment present; `strings` imported exactly once; 13 named tests present; go.mod/go.sum unchanged; scope respected

### Feature Validation
- [ ] Combined short bool bundles (`-vh`, `-af`, `-pl`, `-fl`) set their flags
- [ ] `--flag=value` sets bool flags (value ignored) for version/help/path/list/all/file/relative/no-color
- [ ] `--search=foo` and `--search=` set searchMode + searchQ (foo / "")
- [ ] `-sfoo`, `-ls foo`, `-lsfoo` all set searchMode + the right searchQ
- [ ] `-vz` rejects the WHOLE bundle (unknownFlag set, version/help NOT leaked) → exit 2
- [ ] `-sv` treats post-`s` chars as the query (version NOT set)
- [ ] `-vs` (no value) sets version but leaves searchMode false
- [ ] All existing exact-token forms and unknown-flag paths unchanged

### Code Quality Validation
- [ ] The existing `switch a` and default branch are UNCHANGED (normalization is purely additive)
- [ ] `expandShortBundle` uses two-phase commit (validate-then-commit) — documented why
- [ ] No new config field, no new import, no new export beyond the package-private helper
- [ ] Tests mirror main_test.go style (white-box, plain assertions, table-driven, no testify/Parallel)
- [ ] Existing tests are NOT modified (only appended)

### Scope Discipline (Mode A)
- [ ] `internal/*` packages NOT modified (zero overlap with P1.M3.T1.S1's ui work)
- [ ] `README.md` NOT modified (deferred to P1.M5.T3.S1; no flag-surface change here)
- [ ] `PRD.md` / `tasks.json` / `prd_snapshot.md` NOT modified (read-only / orchestrator-owned)
- [ ] `go.mod` / `go.sum` NOT modified (strings already imported)
- [ ] `git diff --name-only` ⊆ {`main.go`, `main_test.go`} (ui files may also appear if P1.M3.T1.S1 landed — expected, separate subtask)

---

## Anti-Patterns to Avoid

- ❌ **Don't commit partial flags from a bundle with an unknown char.** run() checks
  unknownFlag AFTER version/help, so a leaked `version=true` from `-vz` would print
  the version (exit 0) and mask the error. Use two-phase (validate-then-commit):
  reject the WHOLE bundle on the first unknown char. This is the #1 correctness rule.
- ❌ **Don't replace or rewrite the existing `switch a`.** The normalization is
  ADDITIVE — insert two `continue`-ing branches before it; the switch keeps handling
  exact tokens (`-v`, `--search foo`, `check`, tags, `-x`/`--bogus` unknowns).
  Rewriting the switch risks regressions and is unnecessary.
- ❌ **Don't add a new import or a new config field.** `main.go` already imports
  `"strings"`; the fix needs only HasPrefix/Contains/IndexByte. The config already
  has every field the new forms populate. go.mod must stay unchanged.
- ❌ **Don't treat chars after `s` as flags.** Once `s` is seen, the rest of the
  bundle body is the query verbatim (`-sv` → query "v"; `-sfoo` → query "foo").
  Parsing stops at the first `s`.
- ❌ **Don't set searchMode=true for a value-less `s`.** Mirror the bare `-s`-no-value
  rule: if `s` has no remainder AND no next arg, searchMode stays false. (`-vs` last
  token → version=true, searchMode=false.)
- ❌ **Don't forget `--relative`/`--no-color` are LONG-ONLY.** They have no short
  alias and can never be in a bundle. The bundle's valid chars are exactly
  {v,h,p,l,a,f} + trailing `s`+query.
- ❌ **Don't edit existing tests.** They are unaffected by the new branches (their
  tokens are not `=`-long or len>2 bundles) and still pass. APPEND the 13 new test functions.
- ❌ **Don't change `go.mod`/`go.sum` or `README.md`.** No new dependency (strings is
  stdlib + already imported); README is Mode B (P1.M5.T3).
- ❌ **Don't touch `internal/ui/*`** (P1.M3.T1.S1 owns them this cycle — zero overlap).

---

## Confidence Score

**9/10** — The change is a 2-branch normalization prelude + one two-phase helper +
additive tests, all with verbatim source given. Every consumed contract (`config`
fields, the existing `switch a`, run() precedence, the `strings` import) is on disk
and quoted. Every algorithmic trace (`-vh`/`-vz`/`-sfoo`/`-ls foo`/`-lsfoo`/`-sv`/
`-vs`/`--search=`/`--bogus=x`) was hand-verified, and the no-regression argument for
every existing unknown-flag test is explicit (verified_facts §5). The one residual
subtlety — that two-phase commit is mandatory because of run()'s help→version→
unknownFlag precedence — is the central gotcha and is enforced by a dedicated test
(`TestParseArgsShortBundleUnknownCharRejectsWhole` asserts version stays false on
`-vz`) plus the end-to-end `TestRunShortBundleUnknownExit2`. go.mod-neutral, zero
file overlap with the parallel P1.M3.T1.S1. The -1 is for the inherent chance of a
transcription slip in the helper's two-phase control flow, caught immediately by
`go test . -v` and the Level 4 grep checks.
