# PRP — P1.M1.T2.S1: Record a `storeMissingValue` signal in `parseArgs` for both `--store` no-value branches (Issue 2, parse-level)

> **Subtask:** P1.M1.T2.S1 — the parse-level half of Issue 2 (`init --store` with no value silently overwrites the config). Adds one `config` field and sets it in the two `--store` no-value branches.
> **Scope boundary:** **Parse-level ONLY.** Adds the `storeMissingValue` field + the two setters + parse-level tests. Does NOT add the `run()` exit-2 guard, does NOT print "--store requires a value", does NOT change exit codes, does NOT touch `--search`, does NOT overload `unknownFlag`, does NOT touch runInit (Issue 1 = P1.M1.T1.S1, parallel, disjoint). The `run()` guard is the **next** subtask P1.M1.T2.S2, which consumes this field.

---

## Goal

**Feature Goal**: Give `parseArgs` a way to remember that `--store` (or `--store=`) was seen **without a value**, so the next subtask (P1.M1.T2.S2) can reject it with exit 2 **before** dispatching the destructive auto-detect init that silently overwrites an existing `config.yaml` `store:` value (PRD §6 header "Unknown flags ⇒ error + exit 2"; delta-PRD §2 #3 "init is non-destructive … never clobber").

**Deliverable**: Edits to two existing files only:
1. `main.go` — add `storeMissingValue bool` to the `config` struct (`main.go:128`) with a doc comment; set `c.storeMissingValue = true` in the two `--store` no-value branches (the next-token `case "--store":` `else` arm at `main.go:257-267`, and the `'='`-form `if val == ""` at `main.go:192-197`); update the now-stale next-token comment that called the no-op "intentional/deferred".
2. `main_test.go` — add 3 positive parse-level tests asserting the signal is set for each no-value shape, and strengthen 4 existing tests (bare `init` + the three value-present `--store` tests) with a `storeMissingValue == false` assertion to nail the "iff".

**Success Definition**: `config.storeMissingValue` is `true` **iff** `--store`/`--store=` appeared without a value (bare `init` with no `--store` keeps `storeMissingValue == false` so it still prompts); `go build/vet/test ./...` green; `gofmt -l main.go main_test.go` empty; `go.mod`/`go.sum` unchanged; no `run()` change (exit codes untouched — that is S2).

---

## User Persona (if applicable)

**Not applicable directly** — this subtask adds an internal parse-time signal with no user-visible behavior change yet (the field is consumed by S2, which is what produces the user-facing exit 2 + message). The eventual user is the operator/script who typo's `skilldozer init --store` (forgetting the value) and today silently corrupts their config; S2 stops them, and S1 is the signal S2 needs to do that.

---

## Why

- **Issue 2 is destructive.** Today `skilldozer init --store` (trailing, no value): the `init` token set `c.init=true` with `c.initStore==""`, the trailing `--store` sets nothing (the `i+1 < len(args)` guard at `main.go:263` silently no-ops), so the call degrades to **auto-detect init**, which `setupStore` always ends with `config.Save` — **overwriting** a pre-existing valid `store:` with the auto-detected path. Confirmed repro in `architecture/bug_fixes_validation.md` §ISSUE 2.
- **`c.initStore==""` alone cannot distinguish the bug from the feature.** Bare `skilldozer init` (no `--store`) legitimately has `c.initStore==""` and MUST prompt (PRD §8.2). The distinguishing signal is: **did `--store` actually appear in argv with no value?** That is exactly what `storeMissingValue` records. The contract CRITICAL note and `decisions.md §D2` both call this out.
- **`unknownFlag` is the wrong channel.** It means "first unknown dashed token"; `--store` is a KNOWN flag missing its VALUE — a different error class whose message must be `--store requires a value`, not `unknown flag` (`decisions.md §D2`). A dedicated field keeps the two error classes disjoint.
- **It is the dependency root for S2.** S2's `run()` guard (`if c.storeMissingValue { … exit 2 }`) cannot exist until the field and its setters do. Landing S1 first lets S2 be a tiny, reviewable one-block addition.

---

## What

A pure, additive parse-time change — no behavior shifts yet (no exit code, no message, no dispatch change; all of that is S2). Three edits to `main.go`, tests in `main_test.go`:

**(a) Add the field.** In the `config` struct (`main.go:128-152`), insert `storeMissingValue bool` immediately after `initStore` (keeps the three `--store`-related fields — `init`, `initStore`, `storeMissingValue` — together). Doc comment (Mode A): signals `--store`/`--store=` seen with no value; `run()` rejects with exit 2 (P1.M1.T2.S2); NOT set by bare `init`.

**(b) Set it in the next-token `--store` branch (`main.go:257-267`).** Add an `else` arm to the existing `if i+1 < len(args)` guard: `else { c.storeMissingValue = true }`. Do NOT set `c.init` there (preserve existing behavior — see Gotcha #4). Update the branch comment (it currently calls the silent no-op "deferred/intentional repo-wide", which is now false).

**(c) Set it in the `'='`-form `--store=` branch (`main.go:192-197`).** After the existing `c.init = true; c.initStore = val`, add `if val == "" { c.storeMissingValue = true }`. Keep `c.init=true` there (the `'='`-form sets it unconditionally; S2 gates on `storeMissingValue` before dispatch, so it's harmless).

**(d) Tests.** 3 new positive parse tests + 4 strengthened negatives (see Implementation Tasks).

### Success Criteria

- [ ] `config.storeMissingValue bool` field exists, documented, placed after `initStore`
- [ ] `parseArgs(["init","--store"])` → `storeMissingValue=true` (and `init=true`, `initStore=""`)
- [ ] `parseArgs(["--store="])` → `storeMissingValue=true` (and `init=true`, `initStore=""`)
- [ ] `parseArgs(["--store"])` (bare) → `storeMissingValue=true` (and `init=false`, `initStore=""`)
- [ ] `parseArgs(["init"])` (bare, no `--store`) → `storeMissingValue == false` (the CRITICAL guard — must still prompt)
- [ ] `parseArgs(["--store","/tmp/x"])`, `["init","--store","/tmp/x"]`, `["init","--store=/tmp/x"]` → `storeMissingValue == false` (value present)
- [ ] `c.unknownFlag` is untouched by the `--store` branches (separate error class)
- [ ] The next-token `--store` branch comment no longer claims the no-op is "deferred/intentional"
- [ ] `go build/vet/test ./...` green; `gofmt -l` clean; `go.mod`/`go.sum` unchanged; **no `run()` edit**

---

## All Needed Context

### Context Completeness Check

**Pass.** Both edit sites are pinned to exact current line numbers with exact before/after code (`research/verified_facts.md` §2, §3); the field placement, doc-comment text, the c.init asymmetry to preserve (§4), the S1/S2 boundary (§0, §5), the tests to mirror (§7) and the exact iff coverage (§8) are all enumerated. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative bug writeup + the plan-split note
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "§ISSUE 2 is the authoritative repro + site: parseArgs case '--store' at main.go:257-266 (guard at :263 silently no-ops), the '='-form at :192-197, dispatch at :447. Its 'Fix ordering' note says keep parse+run atomic — BUT the plan SPLIT this into S1 (parse) + S2 (run guard); follow the PLAN (research/verified_facts.md §0). Its Tests note: 'Add parse-level test asserting the missing-value signal is set' — that parse-level test is THIS subtask; the run()-level test is S2."
  section: "ISSUE 2 (Major) — init --store with no value silently overwrites config."

# MUST READ — the field-vs-unknownFlag decision + the run() guard location
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/decisions.md
  why: "§D2 fixes the design: a DEDICATED c.storeMissingValue bool (NOT unknownFlag), set in BOTH parseArgs --store branches, checked in run() AFTER the unknown-flag guard and BEFORE exclusivity/init dispatch. The message is '--store requires a value'. This pins both the field name AND the S2 insertion point (so S1 knows exactly where its consumer lands)."
  section: "D2 — Issue 2 signal: new config field, not reusing unknownFlag."

# MUST READ — the verified facts (line numbers, exact code, the c.init asymmetry, the iff tests)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M1T2S1/research/verified_facts.md
  why: "Every claim anchored: current line numbers for the struct + both branches, the exact before/after edits, the c.init asymmetry table to PRESERVE (§4 — do not 'fix' it), the run() dispatch order so you know where S2 lands (§5), the test harness pattern (§7), and the positive+negative iff coverage (§8)."
  critical: "§0 (the plan SPLIT Issue 2; S1 is parse-only) and §6 (what S1 must NOT do) are the two scope guards most likely to be violated without this file."

# MUST READ — the edit target (read the struct + both --store branches before editing)
- file: main.go
  why: "THE edit site. config struct :128-152; '='-form --store case :192-197 (sets c.init/initStore unconditionally); next-token --store case :257-267 (c.init is INSIDE the i+1<len guard). run() dispatch :428-449 is where S2 lands (read to confirm the boundary, do NOT edit)."
  pattern: "Existing branch style: each --store case has a multi-line comment citing PRD §8.2; setters are bare field assigns. The next-token comment (:258-262) currently LIES ('deferred/intentional repo-wide') — update it."
  gotcha: "The '='-form and next-token branches set c.init DIFFERENTLY (equals always true; next-token only when a value follows). Preserve this — S2's guard makes it moot. See Gotcha #4."

# MUST READ — the test file (mirror the --store parse tests)
- file: main_test.go
  why: "THE other edit target. Existing --store parse tests at :1158-1224 (TestParseArgsInitSubcommand, TestParseArgsInitStoreLongForm, TestParseArgsInitStoreEqualsForm, TestParseArgsStoreWithoutInitToken). The --search no-value test (TestParseArgsSearchNoValueStaysInactive :901) is the closest analog for the no-value shape. Add the 3 new tests after :1224; strengthen the 4 existing tests with the negative assertion."
  pattern: "package main (internal test); parseArgs returns config BY VALUE; read fields directly; t.Errorf('...: got %v; want ...'). No testify, no fixtures."
  gotcha: "Tests are in package main, so the unexported storeMissingValue field IS directly assertable. No need to export it or add an accessor."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/tasks.json
  why: "P1.M1.T2.S1's CONTRACT block is authoritative INPUT/LOGIC/OUTPUT. This PRP transcribes it; tasks.json wins on any conflict."

# READ-ONLY (context) — the parallel sibling PRP (Issue 1), to confirm no write conflict
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M1T1S1/PRP.md
  why: "T1.S1 edits runInit (main.go ~978-1053) + TestRunInitStoreWritesConfig… (:2325). T2.S1 edits the config struct (128-152) + parseArgs (192-197, 257-267) + TestParseArgs* (:1158-1224). DISJOINT regions — land in either order, no merge conflict. Reading it confirms S1 must NOT touch runInit/check-report streams."

# READ-ONLY — the PRD authority for the eventual exit-2 (S2 enforces it; S1 just feeds it)
- file: PRD.md
  why: "READ-ONLY. §6 header ('Unknown flags ⇒ error + exit 2'); §8.2 (init --store <dir> non-interactive form); delta-PRD §2 #3 ('init is non-destructive … never clobber'). These justify why a missing --store value must hard-error; S1 supplies the signal, S2 enforces the exit. Do NOT edit PRD.md."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls main.go main_test.go go.mod
main.go        main_test.go   go.mod
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep)
# No new files created by this subtask — it edits main.go and main_test.go only.
$ go test ./...   # green today (the bug is behavioral; no test exercises the no-value --store branches yet)
```

### Desired Codebase tree with files to be changed

```bash
main.go        # MODIFY — config struct: +storeMissingValue field; parseArgs: 2 setters + 1 comment fix
main_test.go   # MODIFY — +3 positive parse tests; strengthen 4 existing tests with the negative assertion
# go.mod / go.sum — UNCHANGED (no new deps; pure bool field + branch logic)
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `main.go` | +1 field on `config`; +1 `else` arm (next-token `--store`); +1 `if val==""` block (`'='`-form `--store`); 1 comment correction | Issue 2 contract + decisions.md §D2 |
| `main_test.go` | +3 parse tests (signal set); strengthen 4 existing tests (signal NOT set) | QA Issue 2 (parse-level) |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — gofmt re-aligns the WHOLE config struct when you add storeMissingValue.
// It is 18 chars; the current longest field name is unknownFlag (11). Adding any field
// longer than 11 shifts every field's TYPE column right. This is normal gofmt behavior —
// just run `gofmt -w main.go` and expect the whole struct to re-flow. Do NOT hand-align
// and do NOT try to "minimize the diff" by shortening the name; the contract mandates
// `storeMissingValue`.

// GOTCHA #2 — Place the field after initStore (NOT at the end, NOT after unknownFlag).
// The three --store-related fields (init / initStore / storeMissingValue) belong together:
// init = "was --store/init seen?", initStore = "what value?", storeMissingValue = "was the
// value missing?". A reader scanning the init block sees the full --store picture. (gofmt
// re-alignment is identical regardless of position — see #1.)

// GOTCHA #3 — Do NOT add the run() guard. This is the #1 scope trap. The architecture doc
// §ISSUE 2 said "keep parse+run atomic in one subtask" — the plan OVERRODE that and split
// Issue 2 into S1 (parse signal, THIS) + S2 (run guard). S1 sets the field; S2 reads it and
// emits "skilldozer: --store requires a value" + exit 2 BEFORE `if c.init { return runInit }`.
// If you add the guard here, you collide with S2. (research/verified_facts.md §0, §5.)

// GOTCHA #4 — PRESERVE the c.init asymmetry; do not "normalize" it. The '='-form --store=
// sets c.init=true UNCONDITIONALLY (even for empty val); the next-token --store sets c.init
// ONLY when a value follows (it's inside the i+1<len guard). So `--store` (bare) leaves
// c.init=false while `--store=` leaves c.init=true. This is EXISTING behavior; S1 only ADDS
// the storeMissingValue signal. The asymmetry is harmless: S2's run() guard checks
// storeMissingValue BEFORE the c.init dispatch, so both shapes exit 2 regardless. The
// contract does NOT ask you to touch c.init. (research/verified_facts.md §4 has the table.)

// GOTCHA #5 — Do NOT overload c.unknownFlag. decisions.md §D2 is explicit: unknownFlag is
// "first unknown dashed token" (message: "unknown flag '%s'"); --store is a KNOWN flag
// missing its VALUE (message: "--store requires a value"). Different error class, different
// message. Use the dedicated storeMissingValue field. Do not set unknownFlag in the --store
// branches.

// GOTCHA #6 — The next-token --store comment (main.go:258-262) currently LIES: it says the
// silent no-op is "mirrors --search-no-value (no exit-2 'needs argument' here; the codebase
// defers that repo-wide)". After S1, we NO LONGER defer — the else arm records the signal.
// Leaving the comment as-is makes the code lie about its own behavior. Update it to state
// the signal is recorded for run() to reject (P1.M1.T2.S2). (Same Mode-A honesty rule the
// sibling Issue-1 PRP applied to its stale comments.)

// GOTCHA #7 — bare `init` (no --store) must NOT set the signal. This is the contract's
// CRITICAL note: c.initStore=="" legitimately means "prompt" for bare `init`/`init <dir>`.
// storeMissingValue is set ONLY in the two --store branches when the value is missing.
// Do not set it in the `case "init":` block (main.go:268-284) or anywhere else. The negative
// test on TestParseArgsInitSubcommand (bare init) guards exactly this.

// GOTCHA #8 — Do NOT touch --search/-s no-value handling. The architecture doc notes the
// same "value-taking flag with no value silently no-ops" defect affects --search too, but
// there it is harmless (falls through to usage/exit 1). Issue 2's scope is --store ONLY.
// The TestParseArgsSearchNoValueStaysInactive test (:901) must keep passing unchanged.

// GOTCHA #9 — Tests are in package main (internal test). parseArgs returns config BY VALUE,
// and the unexported storeMissingValue field is directly readable in the test. No need to
// export the field or add a getter. Mirror the existing --store test style exactly
// (t.Errorf with %q/%v, no testify).

// GOTCHA #10 — No deps change. storeMissingValue is a plain bool; the if/else and val==""
// checks use only already-imported constructs. go.mod and go.sum must be byte-for-byte
// identical after this subtask. (Carries over from the sibling Issue-1 PRP's GOTCHA #8.)
```

---

## Implementation Blueprint

### Data models and structure

One new field on the existing `config` struct (no new types, no new methods, no signature changes):

```go
// inside type config struct {...} (main.go:128), placed AFTER initStore:
storeMissingValue bool // --store / --store= seen with NO value (Issue 2); run() rejects with
                       // exit 2 before init dispatch (config NOT written) in P1.M1.T2.S2.
                       // NOT set by bare `init` — c.initStore=="" legitimately means "prompt".
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — add the storeMissingValue field to the config struct
  - FILE: main.go (type config struct, main.go:128-152)
  - INSERT one field immediately AFTER the initStore line (main.go:150), BEFORE tags:
        storeMissingValue bool // --store / --store= seen with NO value (Issue 2); run() rejects
                               // with exit 2 before init dispatch (config NOT written) in
                               // P1.M1.T2.S2. NOT set by bare `init` (c.initStore=="" ⇒ prompt).
  - GOTCHA #1: run `gofmt -w main.go` after — the whole struct re-aligns (storeMissingValue=18
    > unknownFlag=11). Expected; do not hand-align.
  - GOTCHA #2: place after initStore, not at the end.
  - NAMING: storeMissingValue (lowercase, unexported — matches version/path/init/initStore/
    unknownFlag; tests are package main so they can read it).

Task 2: EDIT main.go — set the signal in the '='-form --store= branch
  - FILE: main.go (case "--store": inside the --flag=value splitter, main.go:192-197)
  - CURRENT:
        case "--store":
            // `--store=<dir>`: ... Mirrors --search's '='-form; implies init mode (c.init=true). No short form.
            c.init = true
            c.initStore = val
  - ADD after c.initStore = val:
        if val == "" {
            // `--store=` with no value (Issue 2): record the signal so run() (P1.M1.T2.S2)
            // rejects with exit 2 before dispatch. c.init stays true (the '='-form sets it);
            // run() gates on storeMissingValue first.
            c.storeMissingValue = true
        }
  - PRESERVE c.init=true and c.initStore=val (GOTCHA #4). Only ADD the if-block.
  - UPDATE the case comment to note the empty-value signal (one line).

Task 3: EDIT main.go — set the signal in the next-token --store branch + fix the stale comment
  - FILE: main.go (case "--store": in the main switch, main.go:257-267)
  - CURRENT:
        case "--store":
            // `--store <dir>`: ... Mirrors --search's next-token capture; implies init mode
            // (c.init=true). No short form. If it is the LAST token (no value follows) init
            // stays unset — mirrors --search-no-value (no exit-2 "needs argument" here; the
            // codebase defers that repo-wide).
            if i+1 < len(args) {
                c.init = true
                c.initStore = args[i+1]
                i++
            }
  - REPLACE the comment AND add an else arm:
        case "--store":
            // `--store <dir>`: non-interactive store path for init (PRD §8.2). Mirrors
            // --search's next-token capture; implies init mode (c.init=true) when a value
            // follows. No short form. If --store is the LAST token (no value follows),
            // record storeMissingValue so run() rejects with exit 2 (P1.M1.T2.S2) instead
            // of silently no-op'ing into destructive auto-detect init when an `init` token
            // already set c.init (Issue 2).
            if i+1 < len(args) {
                c.init = true
                c.initStore = args[i+1]
                i++
            } else {
                c.storeMissingValue = true
            }
  - GOTCHA #4: do NOT add c.init=true in the else arm (preserve the asymmetry).
  - GOTCHA #6: the OLD comment's "deferred/intentional repo-wide" claim is GONE — the new
    comment describes the signal-based rejection.

Task 4: EDIT main_test.go — add 3 positive parse-level tests (signal IS set)
  - FILE: main_test.go (insert AFTER TestParseArgsStoreWithoutInitToken, ~line 1224)
  - ADD (mirror the existing --store test style; package main; direct field asserts):
    // Issue 2 (P1.M1.T2.S1): `init --store` (last token, no value) records the signal.
    // c.init=true (init token); initStore=""; run() (S2) rejects before dispatch.
    func TestParseArgsInitStoreLongFormNoValueSetsSignal(t *testing.T) {
        c := parseArgs([]string{"init", "--store"})
        if !c.init { t.Errorf("init --store: init=false; want true (init token set it)") }
        if c.initStore != "" { t.Errorf("init --store: initStore=%q; want empty", c.initStore) }
        if !c.storeMissingValue { t.Errorf("init --store: storeMissingValue=false; want true") }
    }
    // Issue 2: `--store=` (empty '='-form value) records the signal. c.init=true ('='-form sets it).
    func TestParseArgsInitStoreEqualsFormEmptyValueSetsSignal(t *testing.T) {
        c := parseArgs([]string{"--store="})
        if !c.init { t.Errorf("--store=: init=false; want true ('='-form implies init)") }
        if c.initStore != "" { t.Errorf("--store=: initStore=%q; want empty", c.initStore) }
        if !c.storeMissingValue { t.Errorf("--store=: storeMissingValue=false; want true (empty value)") }
    }
    // Issue 2: bare `--store` (last token, no init token) records the signal. c.init=false here
    // (no init token; next-token branch sets c.init only when a value follows). run()'s guard
    // (S2) exits 2 regardless of c.init.
    func TestParseArgsStoreNoValueNoInitTokenSetsSignal(t *testing.T) {
        c := parseArgs([]string{"--store"})
        if c.init { t.Errorf("--store (bare): init=true; want false (no init token, no value)") }
        if !c.storeMissingValue { t.Errorf("--store (bare): storeMissingValue=false; want true") }
    }

Task 5: EDIT main_test.go — strengthen 4 existing tests with the NEGATIVE assertion (signal NOT set)
  - FILE: main_test.go (4 existing tests; add ONE assertion line to each)
  - TestParseArgsInitSubcommand (:1158, bare ["init"]) — the CRITICAL guard:
        if c.storeMissingValue { t.Errorf("parseArgs(init): storeMissingValue=true; want false (no --store token; must still prompt)") }
  - TestParseArgsInitStoreLongForm (:1186, ["init","--store","/tmp/x"]):
        if c.storeMissingValue { t.Errorf("init --store /tmp/x: storeMissingValue=true; want false (value present)") }
  - TestParseArgsInitStoreEqualsForm (:1200, ["init","--store=/tmp/x"]):
        if c.storeMissingValue { t.Errorf("init --store=/tmp/x: storeMissingValue=true; want false (value present)") }
  - TestParseArgsStoreWithoutInitToken (:1212, ["--store","/tmp/x"]):
        if c.storeMissingValue { t.Errorf("--store /tmp/x: storeMissingValue=true; want false (value present)") }
  - These complete the "iff" (storeMissingValue true ⟺ --store with no value). Do NOT change
    any existing assertion — only ADD the negative check.
  - GOTCHA #7: the TestParseArgsInitSubcommand assertion is the load-bearing one (bare init
    must still prompt). Do not skip it.

Task 6: VERIFY in isolation + whole module + invariants
  - COMMAND: gofmt -l main.go main_test.go   (must print NOTHING; run gofmt -w if it lists a file)
  - COMMAND: go vet ./...                     (exit 0)
  - COMMAND: go build ./...                   (exit 0)
  - COMMAND: go test -run 'TestParseArgs(Init|Store)' -v ./...  (the 3 new + 4 strengthened pass)
  - COMMAND: go test ./...                    (whole module green; --search test unchanged)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"  (GOTCHA #10)
  - INVARIANT: grep -c 'storeMissingValue' main.go   (expect ≥4: 1 field + 2 setters + comment refs)
  - INVARIANT: grep -c 'requires a value' main.go    (expect 0 — S1 does NOT add the message; S2 does)
```

### Implementation Patterns & Key Details

```go
// The field (Task 1) — placed after initStore in the config struct:
init               bool     // `skilldozer init [<dir>]` ...; also set by `--store <dir>`
initStore          string   // non-interactive store path: ...; empty ⇒ auto-detect
storeMissingValue  bool     // --store / --store= seen with NO value (Issue 2); run() rejects
                            // with exit 2 before init dispatch (config NOT written) in P1.M1.T2.S2.
                            // NOT set by bare `init` (c.initStore=="" ⇒ prompt).
tags        []string // ...
// (gofmt re-aligns every field's type column to storeMissingValue's width — expected.)

// The two setters (Tasks 2 + 3) — ONLY the signal is added; existing assigns preserved:

// '='-form (main.go:192-197):
c.init = true
c.initStore = val
if val == "" {
	c.storeMissingValue = true
}

// next-token (main.go:257-267):
if i+1 < len(args) {
	c.init = true
	c.initStore = args[i+1]
	i++
} else {
	c.storeMissingValue = true
}
```

Notes easy to get wrong:
- The `'='`-form and next-token branches differ in whether they set `c.init` on the no-value path (equals→true, next-token-bare→false). **Preserve both** — S1 only adds the signal. S2's guard runs before `if c.init`, so the c.init value is irrelevant once `storeMissingValue` is set.
- The signal is **parse-level only**. Nothing in `run()` changes in S1. The user still sees the old (buggy) behavior end-to-end until S2 lands — that's the intended sequencing (S1 feeds S2).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Field name + placement? → `storeMissingValue`, after `initStore`.** The contract + decisions.md §D2 fix the name. Placement after `initStore` keeps the three `--store` fields cohesive; gofmt alignment is identical regardless of position.
2. **Add the run() guard here? → NO.** The plan split Issue 2 into S1 (parse) + S2 (run guard); the architecture doc's "atomic" advice was overridden. S1 is the dependency root for S2. Adding the guard here collides with S2.
3. **Set `c.init` in the next-token `else` arm to match the equals-form? → NO.** Preserve the existing asymmetry (Gotcha #4). The contract doesn't ask for it, and S2's pre-dispatch guard makes it moot. Normalizing it would be scope creep with zero behavioral benefit.
4. **Fix the stale next-token comment? → YES.** It currently claims the no-op is "deferred/intentional repo-wide", which is now false (we record the signal). Mode-A honesty (same rule the sibling Issue-1 PRP applied). One comment rewrite, in scope.
5. **Export `storeMissingValue` for external tests? → NO.** `main_test.go` is `package main` (internal test); the unexported field is directly readable. Exporting would widen the public API needlessly.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. No new imports (a bool field + if/else + val=="" use only
    already-imported constructs). (GOTCHA #10)

DISPATCH (unchanged in S1; S2 inserts the guard here):
  run() order today: version(:421) -> unknownFlag(:428) -> exclusivity(:434) -> init(:444).
  S2 inserts `if c.storeMissingValue { fmt.Fprintln(stderr, "skilldozer: --store requires a value"); return 2 }`
  BETWEEN unknownFlag(:428) and exclusivity(:434) — per decisions.md §D2. S1 does NOT add it.

CONSUMERS:
  - run() guard (P1.M1.T2.S2): reads c.storeMissingValue; exit 2; config NOT written.
    Message: "skilldozer: --store requires a value". This is the ONLY consumer.

PARALLEL SIBLING (no conflict):
  - P1.M1.T1.S1 (Issue 1) edits runInit (main.go ~978-1053) + TestRunInitStoreWritesConfig (:2325).
    T2.S1 edits config struct (128-152) + parseArgs (192-197, 257-267) + TestParseArgs* (:1158-1224).
    DISJOINT regions; land in either order.

NO ROUTES / NO DATABASE / NO CONFIG-FILE FORMAT CHANGE.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after the main.go edits)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l main.go main_test.go   # must print NOTHING (run `gofmt -w` if it lists a file)
go vet ./...                     # expect exit 0
go build ./...                   # expect exit 0
# Expected: zero output / exit 0. gofmt will have re-aligned the whole config struct (Gotcha #1).
```

### Level 2: The parse-level unit tests (the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test -run 'TestParseArgs(Init|Store)' -v ./...
# Expected: ALL pass. The load-bearing assertions:
#   TestParseArgsInitStoreLongFormNoValueSetsSignal      -> ["init","--store"] signal=true
#   TestParseArgsInitStoreEqualsFormEmptyValueSetsSignal -> ["--store="]      signal=true
#   TestParseArgsStoreNoValueNoInitTokenSetsSignal       -> ["--store"]       signal=true
#   TestParseArgsInitSubcommand (strengthened)           -> ["init"]          signal=false (CRITICAL)
#   TestParseArgsInitStoreLongForm/EqualsForm/StoreWithoutInitToken (strengthened)
#                                                        -> value present     signal=false
# A missed setter would leave signal=false on a no-value input -> the positive test fails.
# A spurious setter (e.g. in the init case) would set signal on bare init -> the CRITICAL
# negative test fails.

# Control: the --search no-value test is UNCHANGED and still passes (Gotcha #8):
go test -run TestParseArgsSearchNoValueStaysInactive -v ./...
```

### Level 3: Whole-module regression + invariants

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"
go vet  ./...  ; echo "vet exit $?"
go test ./...  ; echo "test exit $?"
# Expected: all exit 0. (runInit/check tests, discover, skillsdir, config all still green —
# S1 touches only parseArgs + the config struct.)

# GOTCHA #10 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
# Expected: "deps unchanged".

# Scope invariants (prove S1 stayed parse-level):
grep -c 'storeMissingValue' main.go      # expect >= 4 (1 field + 2 setters + comment refs)
grep -c 'requires a value' main.go       # expect 0  (S1 adds NO message; S2 does)
git diff main.go | grep -c '^[+-].*runInit\|^[+-].*return 2'  # expect 0  (no run()/exit-code change)
```

### Level 4: Behavioral spot-check (lock the signal against the parser at runtime)

```bash
cd /home/dustin/projects/skilldozer
# Prove all three no-value shapes set the signal and bare init does NOT, via a throwaway probe:
cat > /tmp/sigprobe_test.go <<'EOF'
package main
import "testing"
func TestProbeStoreMissingValueSignal(t *testing.T) {
	cases := []struct{ args []string; want bool }{
		{[]string{"init", "--store"}, true},
		{[]string{"--store="}, true},
		{[]string{"--store"}, true},
		{[]string{"init"}, false},          // bare init — MUST stay false
		{[]string{"--store", "/x"}, false}, // value present
		{[]string{"init", "--store=/x"}, false},
	}
	for _, c := range cases {
		got := parseArgs(c.args).storeMissingValue
		if got != c.want {
			t.Errorf("parseArgs(%v).storeMissingValue=%v; want %v", c.args, got, c.want)
		}
	}
}
EOF
cp /tmp/sigprobe_test.go main_sigprobe_test.go
go test -run TestProbeStoreMissingValueSignal -v ./...
rm main_sigprobe_test.go   # throwaway; keep only main_test.go
# Expected: PASS (locks the iff table from research/verified_facts.md §8).

# NOTE: the END-TO-END bug is NOT yet fixed after S1 alone — `init --store` still overwrites
# the config because run() has no guard yet. That is EXPECTED (S2 adds the guard). Do NOT
# assert end-to-end exit-2 here; that is S2's gate. S1's gate is purely the parse signal.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` clean, `go vet ./...` exit 0, `go build ./...` exit 0
- [ ] Level 2 PASS — the 3 new + 4 strengthened parse tests pass; `TestParseArgsSearchNoValueStaysInactive` unchanged
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0; `git diff go.mod go.sum` → "deps unchanged"; `grep 'requires a value' main.go` → 0
- [ ] Level 4 PASS — the probe table (3 true + 3 false) passes

### Feature Validation
- [ ] `config.storeMissingValue bool` field exists, documented, placed after `initStore`
- [ ] `["init","--store"]`, `["--store="]`, `["--store"]` → `storeMissingValue == true`
- [ ] `["init"]`, `["--store","/x"]`, `["init","--store","/x"]`, `["init","--store=/x"]` → `storeMissingValue == false`
- [ ] `c.unknownFlag` untouched by the `--store` branches; `--search` no-value path untouched
- [ ] The next-token `--store` comment no longer claims the no-op is "deferred/intentional"

### Code Quality / Convention Validation
- [ ] Field naming matches the existing lowercase-bool convention (`version`, `path`, `init`, …)
- [ ] Tests mirror the existing `TestParseArgs*` style (`parseArgs(...)` → direct field `t.Errorf`)
- [ ] No new imports; no new deps; go.mod/go.sum byte-for-byte identical
- [ ] No new files; both edits are to `main.go` + `main_test.go`

### Scope Discipline (the S1/S2 boundary)
- [ ] Did NOT add the `run()` exit-2 guard (S2)
- [ ] Did NOT print "--store requires a value" (S2)
- [ ] Did NOT change any exit code (S2)
- [ ] Did NOT touch `--search`/`-s` no-value handling (out of scope)
- [ ] Did NOT touch `runInit` / the check-report stream (P1.M1.T1.S1, Issue 1, parallel)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't add the `run()` guard.** The plan split Issue 2 into S1 (parse signal) + S2 (run guard). S1 sets the field; S2 consumes it. Adding the guard here collides with S2.
- ❌ **Don't overload `c.unknownFlag`.** It's a different error class ("unknown flag" vs "--store requires a value"). Use the dedicated `storeMissingValue` field (decisions.md §D2).
- ❌ **Don't "normalize" the c.init asymmetry.** The equals-form sets c.init=true unconditionally; the next-token form sets it only when a value follows. Preserve both — S2's pre-dispatch guard makes it moot, and the contract doesn't ask you to touch c.init.
- ❌ **Don't set `storeMissingValue` for bare `init`.** `c.initStore==""` legitimately means "prompt" when there's no `--store` token. The signal is set ONLY in the two `--store` branches when the value is missing. The `TestParseArgsInitSubcommand` negative assertion guards this.
- ❋ **Don't go looking for a place to also fix `--search` no-value.** It's noted in the arch doc but it's harmless (exit 1) and out of scope. Leave `TestParseArgsSearchNoValueStaysInactive` alone.
- ❌ **Don't leave the stale next-token comment saying the no-op is "deferred/intentional repo-wide".** After S1 we record the signal; the comment must say so or the code lies (Mode-A honesty).
- ❌ **Don't touch runInit or the check-report streams.** That's Issue 1 (P1.M1.T1.S1), running in parallel on disjoint lines.
- ❌ **Don't export `storeMissingValue`.** `main_test.go` is `package main`; the unexported field is directly readable. Exporting widens the API needlessly.
- ❌ **Don't hand-align the struct or shorten the field name to "minimize the diff".** Run `gofmt -w`; the whole-struct re-alignment is expected and correct.
- ❌ **Don't add deps or imports.** A bool field + `if/else` + `val == ""` use only existing constructs. go.mod/go.sum must be byte-for-byte unchanged.

---

## Confidence Score

**9.5/10** — This is a 1-point, purely additive parse-time change pinned to exact current line numbers with exact before/after code, the field name fixed by the contract + decisions.md §D2, the S1/S2 boundary crisply drawn (parse-only; the run() guard is S2), and the c.init asymmetry explicitly documented as "preserve, don't fix." The iff test coverage (3 positive + 4 negative, including the CRITICAL bare-`init` guard) is enumerated, and the test harness (`package main`, direct field reads) is read directly from the existing `TestParseArgs*` tests. The 0.5 reservation is for the single human-judgment surface: the next-token branch comment rewrite — the contract names the field/comment scope but a strict reader could leave the stale "deferred/intentional" comment untouched; this PRP rewrites it (Mode-A honesty, same rule the sibling Issue-1 PRP applied), which a reviewer could legitimately defer.
