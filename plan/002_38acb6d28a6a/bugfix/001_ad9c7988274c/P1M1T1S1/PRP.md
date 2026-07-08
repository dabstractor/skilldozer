# PRP — P1.M1.T1.S1: Route init check-report to stderr + strengthen the init stdout test (Issue 1)

> **Subtask:** P1.M1.T1.S1 — the sole subtask of P1.M1.T1 (Issue 1: `init` writes the `check` report to stdout, violating PRD §6.1).
> **Scope boundary:** A writer-argument swap on three `fmt.Fprintf` calls inside `runInit` (stdout→stderr) plus strengthening one existing test. Does NOT touch the standalone `check` subcommand, does NOT add a shared helper, does NOT change exit codes, does NOT add deps, does NOT edit the README (that is P1.M3.T1). Issue 2 (`init --store` missing-value) is a separate task (P1.M1.T2).

---

## Goal

**Feature Goal**: Make `skilldozer init` honor PRD §6.1's stdout contract — stdout carries exactly ONE line (the configured store path) so `STORE="$(skilldozer init --store /path)"` captures a clean, single-line value. The full `check` report (per-skill `OK`/`WARN`/`ERROR` lines + the `N skills, M errors, K warnings` summary) moves to stderr, where init's other human-facing status already lives.

**Deliverable**: Two surgical edits in `main.go` (`runInit`) + one strengthened assertion block in `main_test.go`:
1. `main.go:1046`, `:1050`, `:1053` — change the writer argument of the three check-report `fmt.Fprintf` calls from `stdout` to `stderr`.
2. `main.go:1031-1033` (the `(6)` comment block) and `main.go:978-980` (the runInit doc comment) — correct them to state the report goes to stderr (PRD §6.1), noting the intentional divergence from the standalone `check` subcommand.
3. `main_test.go:2325` `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` — replace the single `Contains(out, store)` assertion (which passes despite the bug) with an exact-stdout regression guard + a positive stderr-contains-summary check.

**Success Definition**: `go test ./...` passes; the strengthened test fails on the OLD code (proven by the bug existing) and passes on the NEW code; `STORE="$(./skilldozer init --store /tmp/x </dev/null)"` yields exactly one line (the path); the standalone `skilldozer check` still prints its report to stdout (`TestRunCheckCleanStore` unchanged); `go.mod`/`go.sum` unchanged.

---

## User Persona (if applicable)

**Target User**: automation/scripts/CI (and any user piping init's output). The non-interactive `skilldozer init --store <dir>` form (PRD §8.2) is meant to be captured: `STORE="$(skilldozer init --store /path)"`.

**Use Case**: A bootstrap script creates a store and captures its path in one command.

**Pain Points Addressed**: Today `$(skilldozer init)` captures the store path **plus** the entire check report, producing a multi-line, unusable `STORE` value. The fix makes init's stdout a reliable single-value capture.

---

## Why

- **PRD §6.1 is labelled "authoritative"** and its `init` row says stdout = "The configured store path." That contract is currently violated by the three `fmt.Fprintf(stdout, ...)` calls in `runInit`'s check-report block.
- **PRD §6.4's whole philosophy is clean stdout for `$(...)` use.** init's own pattern already routes every other human-facing status line to stderr (the `Seeded`/`Adopted` line, the `(found via …)` label, the prompt). The check report is the lone outlier on stdout. This fix harmonizes §8.2 step 5 ("print … check") with §6.1 by sending that report to stderr in the init context.
- **§8.2 step 5 vs §6.1 tension (resolved):** §8.2 literally says "print the output of `skilldozer check`," and `check` itself prints to stdout. But §6.1 is authoritative and §6.4 demands clean stdout, so stderr is the correct destination for the report **inside init**. The standalone `check` subcommand keeps stdout (its report IS its product).

---

## What

`skilldozer init` (both interactive and `--store` non-interactive forms) prints:
- **stdout**: exactly one line — the resolved/effective store dir (`fmt.Fprintln(stdout, dir)` at `main.go:1026`, unchanged).
- **stderr**: the `Seeded`/`Adopted` status, the `(found via <src>)` label, **and now the entire check report** (per-skill lines + summary).
- **exit code**: unchanged (0 on setup success; check findings never gate init's exit — the report is best-effort).

The standalone `skilldozer check` subcommand is **untouched** — its report stays on stdout.

### Success Criteria

- [ ] `runInit`'s three check-report `fmt.Fprintf` calls (main.go:1046/1050/1053) write to `stderr`, not `stdout`
- [ ] `main.go:1026` (`fmt.Fprintln(stdout, dir)`) is unchanged — store path stays on stdout
- [ ] The `(6)` comment block (main.go:1031-1033) and runInit doc comment (main.go:978-980) state the report goes to stderr (PRD §6.1) and note the init↔check divergence
- [ ] No shared helper extracted; the standalone `if c.check` block (main.go:557-584) is untouched
- [ ] `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` asserts stdout == exactly `store+"\n"`; no check markers on stdout; summary on stderr
- [ ] `TestRunCheckCleanStore` (main_test.go:1239) still passes — standalone check keeps its report on stdout
- [ ] `go test ./...` green; `go vet ./...` clean; `go.mod`/`go.sum` unchanged

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact three line numbers, the exact format strings, the lines that must NOT move (1026), the two comments' current text, the exact test to strengthen (and why its current assertion is too weak), and the collateral check (no other init test asserts check-report stdout) are all specified with file:line references. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
- file: main.go
  why: "THE edit site. runInit at :988; the three buggy emitters at :1046/:1050/:1053; the store-path headline at :1026 (STAYS stdout); the (6) comment block at :1031-1033; the runInit doc comment at :978-980; the standalone check branch at :557-584 (DO NOT TOUCH)."
  pattern: "Writer-argument swap: fmt.Fprintf(stdout, ...) -> fmt.Fprintf(stderr, ...) for the three report calls only."
  gotcha: "Do NOT change the format strings, args, %-5s padding, or the `continue` on the OK branch. Do NOT touch :1026 or :557-584."

- file: main_test.go
  why: "THE test to strengthen: TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 at :2325. Its current stdout assertion (:2357 `Contains(out, store)`) passes despite the bug — that is exactly why the bug shipped. Also the test harness reference: run([]string{...}, &out, &errOut) returns exit code; out.String()=stdout, errOut.String()=stderr."
  pattern: "Replace the weak Contains with exact-stdout (out.String()==store+\"\\n\") + no-markers-on-stdout + summary-on-stderr. Mirror the existing assertion style (t.Errorf with %q)."
  gotcha: "Assert on the summary marker `skills,` (always emitted when discover.Index finds >=1 skill), NOT on `OK` specifically — the test stays green if a future template tweak changes the per-skill line. The freshly-seeded store in this test has the example skill, so the summary is `1 skills, 0 errors, 0 warnings`."

- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "§ISSUE 1 is the authoritative bug writeup: confirmed repro, exact file:line of every emitter + the headline that stays, the precise fix, and the named test to strengthen."
  section: "ISSUE 1 (Major) — init writes the check report to stdout (should be stderr)"

- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/system_context.md
  why: "Documents the dispatch architecture (run()->runInit), the run() buffer test seam, the exit-code contract (init stays 0), and the cross-cutting risk that the runInit block is a byte-copy of the if c.check block — they now diverge intentionally."
  section: "runInit (main.go:988) — Issues 1, 5; Cross-cutting risk #1"

- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M1T1S1/research/verified_facts.md
  why: "Direct-from-source proof of every claim: the exact emitter lines, what must NOT move, the comment texts to correct, the strengthened-test design, and the collateral check (no other init test breaks)."

- url: https://pkg.go.dev/fmt#Fprintf
  why: "Confirms fmt.Fprintf(w io.Writer, format, args...) takes the writer as the first arg — so swapping stdout->stderr is a one-token change per call with no signature impact."

- url: (PRD §6.1/§6.4 — in PRD.md, READ-ONLY)
  why: "§6.1 init row: stdout = 'The configured store path.' §6.4: clean stdout discipline for $(...) use. These are the authority the fix enforces. Do NOT edit PRD.md."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls main.go main_test.go go.mod
main.go        main_test.go   go.mod
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep)
# No new files created by this subtask — it edits main.go and main_test.go only.
```

The repo compiles green today; the bug is behavioral (wrong stream), not a compile error.

### Desired Codebase tree with files to be changed

```bash
main.go           # MODIFY — runInit: 3 writer swaps (1046/1050/1053) + 2 comment corrections
main_test.go      # MODIFY — strengthen TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 (:2325)
# go.mod / go.sum — UNCHANGED (no new deps; this is a writer swap)
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `main.go` | Swap the 3 report emitters to stderr; correct the 2 comments | PRD §6.1/§6.4 + this contract |
| `main_test.go` | Strengthen the init stdout test to a regression guard | QA Issue 1 |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — Swap ONLY the three report emitters (:1046/:1050/:1053). Leave :1026
// (fmt.Fprintln(stdout, dir)) ALONE — that is the §6.1 store-path headline and must
// stay on stdout. A careless "replace all stdout->stderr in runInit" breaks the
// contract this fix is enforcing.
//
// GOTCHA #2 — Do NOT touch the standalone `if c.check` branch (main.go:557-584).
// It renders the IDENTICAL report to stdout and that is CORRECT for `skilldozer
// check` (its report is its stdout product; TestRunCheckCleanStore @1239 asserts
// `Contains(got,"OK")` + "2 skills, 0 errors, 0 warnings" on stdout). After this
// fix the two blocks intentionally DIVERGE (init->stderr, check->stdout).
//
// GOTCHA #3 — Do NOT extract a shared renderReport(w, ...) helper. The whole point
// of the fix is the divergence; a shared helper would re-couple them and the
// existing "do not refactor; mirror" comment on the standalone block still holds.
// Just swap the writer token three times.
//
// GOTCHA #4 — Two comments currently state the wrong stream and BOTH need correcting.
// The contract names the (6) block (:1031-1033), but the runInit-level doc comment
// (:978-980) ALSO says "the `check` report to stdout" — leave it and the code lies.
// Fix both. Cite the §6.1-authoritative vs §8.2-step-5 tension so the next reader
// understands why init and check now differ.
//
// GOTCHA #5 — The test's current `Contains(out, store)` is a TRAP: it passes on the
// buggy code (stdout still contains the path, just with the report alongside).
// The regression guard must be EXACT: `out.String() == store+"\n"`. A weaker
// `HasPrefix` or `Contains` will not catch the bug.
//
// GOTCHA #6 — Assert on the SUMMARY marker ("skills,") for the stderr positive
// check, not on "OK". discover.Index on the freshly-seeded store finds the example
// skill, so the summary is always emitted; but a future template tweak could turn
// the per-skill line from OK into WARN/ERROR. The summary is the stable signal.
//
// GOTCHA #7 — init exit code is UNCHANGED (0). The check report is best-effort;
// check findings never gate init's exit (existing comment at the `return 0` on
// :1054). Do not add any exit-code logic.
//
// GOTCHA #8 — No deps change. fmt.Fprintf, io.Writer, bytes.Buffer are all stdlib;
// the test already imports bytes/fmt/io/strings/os/path/filepath/testing. go.mod
// and go.sum must be byte-for-byte identical after this subtask.
```

---

## Implementation Blueprint

### Data models and structure

None. This subtask changes no data models — `runInit`'s signature `(c config, stdout, stderr io.Writer) int` already takes both writers as arguments. The fix uses the `stderr` argument that is already in scope; no new parameters, no new types.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: SWAP the three check-report emitters in runInit (main.go)
  - EDIT main.go:1046  fmt.Fprintf(stdout, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)
        ->  change first arg `stdout` to `stderr`  (format/args unchanged)
  - EDIT main.go:1050  fmt.Fprintf(stdout, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)
        ->  change first arg `stdout` to `stderr`
  - EDIT main.go:1053  fmt.Fprintf(stdout, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)
        ->  change first arg `stdout` to `stderr`
  - DO NOT TOUCH main.go:1026 (fmt.Fprintln(stdout, dir))  — store path stays on stdout (GOTCHA #1)
  - DO NOT TOUCH main.go:557-584 (standalone if c.check)   — keeps stdout (GOTCHA #2)
  - NAMING/PLACEMENT: no renames, no moves — same lines

Task 2: CORRECT the two runInit comments (Mode A inline doc)
  - EDIT main.go:1031-1033 (the (6) block). Current text says it "Mirrors the
    `if c.check` branch render VERBATIM (do not refactor; mirror)". Rewrite to state:
    init renders the check report to STDERR (PRD §6.1: stdout = the store path only);
    the standalone `check` subcommand keeps its report on stdout; the two diverge by
    design (do not extract a shared helper). Keep the "best-effort, non-fatal" note.
  - EDIT main.go:978-980 (runInit doc comment). Current: "...the `check` report to
    stdout (PRD §8.2 step 5)". Change to: "...the `check` report to stderr (PRD §6.1
    stdout contract; §6.1 is authoritative over §8.2 step 5's 'print check' wording)".
  - KEEP both comments accurate to the new behavior; cite PRD §6.1 as the authority.

Task 3: STRENGTHEN TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 (main_test.go:2325)
  - REPLACE the weak assertion at main_test.go:2357 (`if !strings.Contains(out.String(), store)`)
    with an Issue-1 regression guard block. Assert:
      (a) EXACT stdout:  if got := out.String(); got != store+"\n" {
                            t.Errorf("init stdout=%q; want exactly %q (§6.1: one line, the store path)", got, store+"\n") }
      (b) NO report markers on stdout: for _, m := range []string{"skills,", "OK", "errors", "warnings"} {
                            if strings.Contains(out.String(), m) {
                                t.Errorf("init stdout leaked check-report marker %q: %q", m, out.String()) } }
      (c) SUMMARY on stderr: if !strings.Contains(errOut.String(), "skills,") {
                            t.Errorf("init stderr=%q; missing the check summary (report must go to stderr)", errOut.String()) }
  - ADD a short comment naming this as the Issue-1 regression guard and explaining WHY
    the exact check is required (the old Contains passed despite the bug — GOTCHA #5).
  - KEEP all existing assertions in the test (exit 0, store created, config.Store==store).
  - DO NOT rename the test (widely referenced; contract says "strengthen", not "rename").
  - DO NOT add a new test file; edit in place.

Task 4: VERIFY in isolation + whole module
  - COMMAND: go build ./...                  (exit 0)
  - COMMAND: go vet ./...                    (clean)
  - COMMAND: go test ./...                   (all green, incl. TestRunCheckCleanStore untouched)
  - COMMAND: go test -run TestRunInitStore   -v ./...   (the strengthened test passes)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"  (GOTCHA #8)
```

### Implementation Patterns & Key Details

```go
// The three swaps (Task 1) — before / after. ONLY the first argument changes:

// :1046  BEFORE:  fmt.Fprintf(stdout, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)
//        AFTER :  fmt.Fprintf(stderr, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)

// :1050  BEFORE:  fmt.Fprintf(stdout, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)
//        AFTER :  fmt.Fprintf(stderr, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)

// :1053  BEFORE:  fmt.Fprintf(stdout, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)
//        AFTER :  fmt.Fprintf(stderr, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)

// :1026  UNCHANGED (the §6.1 headline — do not touch):
//        fmt.Fprintln(stdout, dir)

// Strengthened test assertion (Task 3) — the authoritative bug-catcher is (a):
got := out.String()
if got != store+"\n" {                                  // (a) exact stdout — catches the bug
	t.Errorf("init stdout=%q; want exactly %q (§6.1: one line, the store path)", got, store+"\n")
}
for _, m := range []string{"skills,", "OK", "errors", "warnings"} { // (b) no report on stdout
	if strings.Contains(got, m) {
		t.Errorf("init stdout leaked check-report marker %q: %q", m, got)
	}
}
if !strings.Contains(errOut.String(), "skills,") {      // (c) summary on stderr (GOTCHA #6)
	t.Errorf("init stderr=%q; missing check summary (report must be on stderr)", errOut.String())
}
```

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. No new imports (stderr is already a runInit param; the
    test already imports bytes/strings/os/path/filepath/testing). (GOTCHA #8)

DISPATCH (unchanged):
  - run() -> if c.init { return runInit(c, stdout, stderr) }  (main.go:447) — runInit keeps
    its (c, stdout, stderr io.Writer) signature; the fix just uses stderr for the report.

CONSUMERS (behavioral, not code):
  - Scripts/CI: STORE="$(skilldozer init --store /path)" now yields exactly one line.
  - Humans: still see the full check report (now on stderr, interleaved with the
    Seeded/Adopted + found-via status, which is where init's other status already lived).

NO ROUTES / NO DATABASE / NO CONFIG SCHEMA:
  - This subtask touches only output streams. No config-file format change, no discovery
    rule change, no exclusivity change (those are Issues 2-7, separate tasks).
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after the edits)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l main.go main_test.go     # must print NOTHING (already gofmt-clean)
go vet ./...                       # expect exit 0
go build ./...                     # expect exit 0

# Expected: zero output / exit 0. If gofmt lists a file, run `gofmt -w` on it.
```

### Level 2: The strengthened unit test (the core regression gate)

```bash
cd /home/dustin/projects/skilldozer

go test -run TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 -v ./...
# Expected: PASS. The exact-stdout assertion (out.String()==store+"\n") is the bug-catcher.

# Prove the new assertion actually catches the bug: temporarily revert ONE swap
# (e.g. :1046 stdout) and re-run — the test MUST fail with "stdout leaked check-report
# marker" or exact-mismatch. Then restore the swap. (This confirms the guard is load-bearing.)
```

### Level 3: Whole-module regression + the standalone-check invariant

```bash
cd /home/dustin/projects/skilldozer

go test ./...   ; echo "test exit $?"
# Expected: exit 0. Critically: TestRunCheckCleanStore (main_test.go:1239) MUST still pass —
# the standalone `check` subcommand keeps its report on stdout (untouched by this fix).

# Dependency invariant (GOTCHA #8):
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
# Expected: "deps unchanged".
```

### Level 4: End-to-end behavioral check (the user-facing contract)

```bash
cd /home/dustin/projects/skilldozer
go build -o /tmp/sdz . || { echo "FAIL: build"; exit 1; }

# Fresh store; capture init's stdout exactly.
TMP=$(mktemp -d); STORE="$TMP/store"; CFG="$TMP/cfg.yaml"
SKILLDOZER_CONFIG="$CFG" env -u SKILLDOZER_SKILLS_DIR \
  /tmp/sdz init --store "$STORE" </dev/null >/tmp/sdz.out 2>/tmp/sdz.err
echo "exit=$?"                                   # Expected: 0
echo "stdout line count: $(wc -l </tmp/sdz.out)" # Expected: 1
echo "stdout = $(cat /tmp/sdz.out)"              # Expected: exactly $STORE
echo "--- stderr (report lives here now) ---"
cat /tmp/sdz.err                                 # Expected: Seeded/Adopted + found via + OK example (example) + "1 skills, 0 errors, 0 warnings"

# The one-line-capture guarantee for scripts/CI:
CAPTURED=$(SKILLDOZER_CONFIG="$CFG" env -u SKILLDOZER_SKILLS_DIR /tmp/sdz init --store "$STORE" </dev/null 2>/dev/null)
test "$CAPTURED" = "$STORE" && echo "CAPTURE OK (single line)" || echo "FAIL: captured='$CAPTURED'"

# Control: standalone `check` STILL prints its report to stdout (unchanged).
SKILLDOZER_CONFIG="$CFG" env -u SKILLDOZER_SKILLS_DIR /tmp/sdz check >/tmp/chk.out 2>/dev/null
grep -q "skills, 0 errors" /tmp/chk.out && echo "check stdout intact" || echo "FAIL: check report missing from stdout"
rm -rf /tmp/sdz "$TMP" /tmp/sdz.out /tmp/sdz.err /tmp/chk.out
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` clean, `go vet ./...` exit 0, `go build ./...` exit 0
- [ ] Level 2 PASS — strengthened test passes; reverting a swap makes it fail (guard is load-bearing)
- [ ] Level 3 PASS — `go test ./...` exit 0; `TestRunCheckCleanStore` still green; `git diff go.mod go.sum` → "deps unchanged"
- [ ] Level 4 PASS — init stdout is exactly one line; `$(...)` capture equals the store path; standalone check report still on stdout

### Feature Validation
- [ ] main.go:1046/1050/1053 write to `stderr`; main.go:1026 unchanged (stdout)
- [ ] The `(6)` comment block (:1031-1033) and runInit doc comment (:978-980) corrected to stderr + the init↔check divergence noted
- [ ] No shared helper extracted; standalone `if c.check` (:557-584) untouched
- [ ] init exit code unchanged (0); no exit-code logic added

### Code Quality / Convention Validation
- [ ] Writer swap only — format strings, args, `%-5s` padding, `continue` all unchanged
- [ ] Strengthened test uses the existing harness (`run(..., &out, &errOut)`) and assertion style (`t.Errorf` with `%q`)
- [ ] No new imports; no new deps; go.mod/go.sum byte-for-byte identical

### Scope Discipline
- [ ] Did NOT touch Issue 2 (`init --store` missing-value; that is P1.M1.T2)
- [ ] Did NOT touch Issue 5 (tilde expansion in resolveStore; that is P1.M2.T3)
- [ ] Did NOT edit the README (cross-cutting sweep is P1.M3.T1; init section doesn't describe the report stream today)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't "replace all stdout→stderr in runInit."** That would move the §6.1 store-path headline (:1026) too, breaking the very contract this fix enforces. Swap exactly the three report emitters (:1046/:1050/:1053).
- ❌ **Don't touch the standalone `check` branch** (:557-584). Its stdout report is correct and `TestRunCheckCleanStore` depends on it. The divergence is intentional.
- ❌ **Don't extract a shared `renderReport` helper.** The divergence IS the fix; a helper re-couples them and the "do not refactor; mirror" comment still applies to the standalone block.
- ❌ **Don't leave the runInit doc comment (:978-980) saying "to stdout".** Two comments are now wrong; fix both or the code lies about its own behavior.
- ❌ **Don't keep the weak `Contains(out, store)` assertion.** It passed despite the bug — that's why the bug shipped. The guard must be exact: `out.String() == store+"\n"`.
- ❌ **Don't assert the stderr positive check on `"OK"`.** Use the summary marker `"skills,"` (always emitted when ≥1 skill) so the test survives a future per-skill-line tweak.
- ❌ **Don't add exit-code logic.** The check report is best-effort; init stays 0 regardless of findings.
- ❌ **Don't add deps or imports.** This is a writer-argument swap; everything needed (stderr, fmt, bytes, strings) is already in scope.

---

## Confidence Score

**9/10** — Every edit site is pinned to an exact line number with the exact before/after text; the fix is a three-token writer swap plus an exact-equality test guard, all verified against the built source and the QA repro (which showed stdout = three lines pre-fix). The strengthened test is provably load-bearing (the old `Contains` passed on buggy code; the new `==` cannot). The 1-point reservation is for the second comment edit (:978-980): the contract explicitly names only the `(6)` block, so a strict reading could skip the doc comment — but leaving it wrong makes the code lie, so the PRP corrects both and flags the judgment call.
