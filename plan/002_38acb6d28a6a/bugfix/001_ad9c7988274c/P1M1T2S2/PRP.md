# PRP — P1.M1.T2.S2: Guard `run()` to reject a missing `--store` value with exit 2 before init dispatch (config NOT written) [Issue 2, run-level]

> **Subtask:** P1.M1.T2.S2 — the run-level half of Issue 2 (`init --store` with no value silently overwrites the config). T2.S1 (parse-level, parallel, lands first) added the `c.storeMissingValue` field + sets it in both `--store` no-value branches. **This subtask consumes that field: it inserts one `if c.storeMissingValue { … return 2 }` guard into `run()` between the unknown-flag check and the exclusivity check, so a missing `--store` value fails fast (exit 2) BEFORE `runInit`/`setupStore`/`configpkg.Save` is reached — leaving any existing `config.yaml` byte-for-byte untouched.**
> **Scope:** Two existing files only — `main.go` (one guard block + one precedence-comment line) and `main_test.go` (4 run()-level tests). No new files. No `parseArgs` change (T2.S1 owns the signal). No `runInit`/`setupStore` change. No deps.

---

## Goal

**Feature Goal**: Make `skilldozer init --store` (and `--store=`, and bare `--store`) **fail fast with exit 2** when `--store` is presented without its value, instead of silently degrading into destructive auto-detect init that unconditionally calls `configpkg.Save` and overwrites a pre-existing valid `store:` value. The guard returns **before** `runInit` is dispatched, so the config file is never touched. This implements PRD §6 header ("Unknown flags ⇒ error + exit 2") and the delta-PRD §2 #3 constraint ("init is non-destructive … never clobber or delete").

**Deliverable**: Edits to two existing files:
1. `main.go` — insert one guard block into `run()` (new step 3.5, between the `unknownFlag` check and the `exclusivityError` check): `if c.storeMissingValue { fmt.Fprintln(stderr, "skilldozer: --store requires a value"); return 2 }`; update the `run()` precedence comment (Mode A) to list the new step in the ladder.
2. `main_test.go` — add 4 run()-level tests: `init --store`/`--store=`/bare `--store` each exit 2 with empty stdout + the exact stderr message; and a load-bearing "config NOT written" test proving a pre-existing config is preserved.

**Success Definition**: `skilldozer init --store` (no value), `--store=` (empty), and bare `--store` all exit 2 with stderr `skilldozer: --store requires a value\n` and an EMPTY stdout; a pre-existing `config.yaml`'s `store:` value is byte-for-byte unchanged after the invocation; bare `skilldozer init` (no `--store`) still proceeds to `runInit` and prompts (the signal is `false` there); `go build/vet/test ./...` green with zero regressions; `go.mod`/`go.sum` unchanged.

---

## User Persona (if applicable)

**Target User**: the operator or CI script author who runs `skilldozer init --store` and forgets the value (e.g. a trailing `--store` with nothing after it, or a copy-paste `--store=`).
**Use Case**: `STORE="$(skilldozer init --store /path)"` in a script — today, a typo'd `skilldozer init --store` silently rewrites the config to an auto-detected path; after this fix it fails loudly with exit 2 and a clear message, leaving the config intact.
**Pain Points Addressed**: silent config corruption (the most dangerous failure mode — a valid `store:` is overwritten with no warning); a value-taking flag that no-ops instead of erroring.

---

## Why

- **Issue 2 is destructive.** Today `skilldozer init --store` (trailing, no value): the `init` token set `c.init=true` with `c.initStore==""`; the trailing `--store` sets nothing (the `i+1 < len(args)` guard no-ops); the call degrades to **auto-detect init**, which `setupStore` ends with `configpkg.Save` (`main.go:979`) — **overwriting** a pre-existing valid `store:` with the auto-detected path. Reproduced in `architecture/bug_fixes_validation.md` §ISSUE 2 (exit 0, config rewritten).
- **The fix is a one-line gate that precedes the destructive call.** T2.S1 already records the distinguishing signal (`c.storeMissingValue`, true iff `--store`/`--store=` appeared with no value — `c.initStore==""` alone is insufficient because bare `init` legitimately has it). This subtask reads that signal in `run()` and returns exit 2 **before** `if c.init { return runInit(...) }`, so `setupStore`/`configpkg.Save` is never reached. (decisions.md §D2 fixes the field-vs-`unknownFlag` choice + the insertion point.)
- **It restores flag-parsing sanity.** A value-taking flag presented without its value is a parse error, not a silent no-op — matching how `--search`/every well-behaved CLI behaves and PRD §6 header ("Unknown flags ⇒ error + exit 2").
- **Zero blast radius.** Grep-confirmed: no existing run()-level test passes a no-value `--store` shape (the only `run()` test with `--store` uses a valid value), so this purely-additive guard breaks nothing (see Context §"Zero breakage").

---

## What

A single `if`-block added to `run()` plus a one-line precedence-comment update, with 4 run()-level tests. No parseArgs change (T2.S1 owns the signal), no runInit/setupStore change, no exit-code change anywhere else.

### Success Criteria

- [ ] `run(["init","--store"])` → exit 2, stdout EMPTY, stderr == `"skilldozer: --store requires a value\n"`
- [ ] `run(["--store="])` → exit 2, stdout EMPTY, stderr message (equals-form empty value)
- [ ] `run(["--store"])` (bare, no init token) → exit 2, stdout EMPTY, stderr message (was exit 1 usage — intentional improvement)
- [ ] A pre-existing config.yaml's `store:` value is byte-for-byte unchanged after `run(["init","--store"])` (the non-destructive contract)
- [ ] `run(["init"])` (bare, no `--store`) still proceeds to `runInit` (signal is false; must still prompt) — no regression
- [ ] `run(["init","--store","/x"])` still exits 0 and writes the config (valid value; signal false) — `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` unchanged
- [ ] `go build/vet/test ./...` green; `gofmt -l` clean; `go.mod`/`go.sum` unchanged; the `run()` precedence comment lists the new step

---

## All Needed Context

### Context Completeness Check

**Pass.** The single insertion site is pinned by branch content (between the `unknownFlag` block and the `exclusivityError` block in `run()`), the exact guard code + message are fixed by the contract + decisions.md §D2, the message-routing (`fmt.Fprintln` to `stderr`) mirrors the existing `exclusivityError` branch, the non-destructive chain (`setupStore`→`configpkg.Save` at `:979`, called by `runInit` at `:1012`, never reached because the guard returns at `~:438` before `runInit` at `:455`) is grep-verified, the zero-breakage surface is grep-confirmed (only one `run()` test passes `--store`, with a valid value), and the test patterns to mirror (`TestRunDefaultUnknownFlag` for exit-2+exact-stderr; `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` for the config-write setup, inverted) are read at exact line ranges. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative bug writeup + repro + site
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "§ISSUE 2 is the authoritative repro: `init --store </dev/null` with an existing
        config → exit 0, store overwritten. Site: dispatch at main.go:447 (if c.init return
        runInit) has no missing-value guard. Its 'Fix' note says reject in run() with exit 2
        BEFORE dispatching init (config must NOT be written); its 'Tests' note says add a
        run()-level test asserting exit 2 + empty stdout + config NOT written (mirror
        TestRunExclusivityInitAndCheck). NOTE: it also said keep parse+run atomic — the plan
        OVERRODE that and split into S1 (parse signal) + S2 (THIS run guard); follow the plan."
  section: "ISSUE 2 (Major)."

# MUST READ — the guard location + message (decisions.md §D2)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/decisions.md
  why: "§D2 fixes: a DEDICATED c.storeMissingValue field (NOT unknownFlag — different error
        class), checked in run() AFTER the unknown-flag guard, BEFORE the exclusivity/init
        dispatch. Message: 'skilldozer: --store requires a value'. This pins both the exact
        insertion point (between unknownFlag and exclusivity) AND the exact message string."
  section: "D2 — Issue 2 signal: new config field, not reusing unknownFlag."

# MUST READ — the verified facts (insertion point, exact code, the chain, zero-breakage, tests)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M1T2S2/research/verified_facts.md
  why: "§1 pins the insertion by BRANCH (run() line numbers drift ~+5 once T2.S1's next-token
        else arm lands — do NOT rely on line numbers); §2 is the exact guard+comment block;
        §3 the precedence-comment update; §4 the three shapes that set the signal (all exit 2
        after the guard, incl. bare --store); §5 the bare---store exit-1→exit-2 change
        (intentional); §6 the grep proof of ZERO existing-test breakage; §7 the
        setupStore→configpkg.Save chain that is never reached; §8 the 4 tests."
  critical: "§5 (bare --store exits 2, not 1 — must be test-locked) and §6 (zero breakage —
             the load-bearing safety check) are the two things most likely to be missed."

# MUST READ — the file under edit (read run() in full before editing)
- file: main.go
  why: "THE edit site. func run() at :414 (today). The dispatch ladder: help(:421) →
        version(:428) → unknownFlag(:434-437) → [INSERT GUARD HERE] → exclusivity(:440-446)
        → init dispatch(:455-456). setupStore(:957) calls configpkg.Save(:979); runInit(:998)
        calls setupStore(:1012) — all AFTER the guard, so never reached on a missing value."
  pattern: "unknownFlag branch style: fmt.Fprintf(stderr, \"skilldozer: ...\\n\", ...); return 2.
            exclusivityError branch style: fmt.Fprintln(stderr, msg); return 2 (fixed string,
            no args → Fprintln). Mirror Fprintln for the fixed '--store requires a value' msg."
  gotcha: "INSERT BY BRANCH (after the unknownFlag block, before the '// 4) Mode mutual
           exclusivity' comment), NOT by line number — T2.S1's pending next-token else arm
           shifts run() down a few lines once it lands."

# MUST READ — the test file (mirror the exit-2 + config-write patterns)
- file: main_test.go
  why: "THE other edit target. TestRunDefaultUnknownFlag (:296-312) is the exit-2 + empty-
        stdout + EXACT-stderr-line pattern to mirror. TestRunInitStoreWritesConfigCreatesStore
        PrintsPathExit0 (:2325-2365) is the run()-level init + SKILLDOZER_CONFIG + configpkg.Load
        pattern to INVERT for the 'config NOT written' test. Tests are package main (internal)."
  pattern: "var out, errOut bytes.Buffer; code := run([]string{...}, &out, &errOut); assert
            code==2, out.Len()==0, errOut.String()==exact. For config-not-written: t.Setenv
            SKILLDOZER_CONFIG=<tmp cfg>, pre-write config, t.Setenv SKILLDOZER_SKILLS_DIR=\"\",
            t.Chdir(t.TempDir()), run, then configpkg.Load(cfg).Store unchanged + bytes.Equal."
  gotcha: "Assert the message with EXACT equality (errOut.String() != want), mirroring
           TestRunDefaultUnknownFlag — the contract fixes the string verbatim. Do NOT use
           Contains (it would hide a prefix/wording drift)."

# MUST READ — the parse-level sibling PRP (defines the field this guard consumes)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M1T2S1/PRP.md
  why: "Confirms T2.S1 adds config.storeMissingValue (bool, after initStore) + sets it in BOTH
        --store no-value branches (next-token else arm + '='-form if val==\"\"), and does NOT
        add the run() guard (this subtask). Fixes the S1/S2 boundary so this PRP does not
        duplicate parseArgs edits. Bare init keeps storeMissingValue=false (must still prompt)."

# READ-ONLY — the PRD authority for exit 2 + non-destructive init
- file: PRD.md
  why: "READ-ONLY. §6 header ('Unknown flags ⇒ error + exit 2'); §8.2 (init --store <dir>
        non-interactive form); §6.4 (stdout discipline for $(...)). delta-PRD §2 #3 ('init is
        non-destructive ... never clobber or delete'). These justify exit 2 + config-preserving."
  section: "§6 header, §6.4, §8.2 (and the bugfix PRD §3.1 Issue 2 for the repro)."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/tasks.json
  why: "P1.M1.T2.S2's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. This PRP
        transcribes it; tasks.json wins on any conflict."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls main.go main_test.go go.mod
main.go        main_test.go   go.mod
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep)
# T2.S1 status (parallel): config.storeMissingValue field LANDED (:142) + '='-form setter (:193-202);
#   next-token else arm PENDING but will land per the T2.S1 PRP contract.
$ grep -n 'storeMissingValue' main.go    # field + setters (T2.S1); this subtask adds the run() reader
```

### Desired Codebase tree with files to be changed

```bash
main.go        # MODIFY — run(): +1 guard block (step 3.5, between unknownFlag and exclusivity); +1 precedence-comment line
main_test.go   # MODIFY — +4 run()-level tests (3 exit-2 shapes + 1 config-not-written)
# go.mod / go.sum — UNCHANGED (no new deps; one if-block + Fprintln use only already-imported fmt/io)
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `main.go` | `run()`: +1 `if c.storeMissingValue { fmt.Fprintln(stderr, "skilldozer: --store requires a value"); return 2 }` guard (before exclusivity); +1 precedence-comment line | Issue 2 contract + decisions.md §D2 |
| `main_test.go` | +4 run()-level tests: 3 exit-2 shapes (exact stderr) + 1 config-NOT-written (non-destructive) | QA Issue 2 (run-level) |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — INSERT BY BRANCH, not line number. T2.S1 is in flight; its pending next-token
// `else { c.storeMissingValue = true }` arm adds ~5 lines to parseArgs, shifting run() DOWN by
// ~5 lines once it lands. The contract cites run() at main.go:414 / unknownFlag at :434, but those
// are the CURRENT (pre-next-token-arm) numbers. Anchor the insertion to BRANCH CONTENT: insert
// immediately AFTER the unknownFlag block (`if c.unknownFlag != "" { ...; return 2 }` + its blank
// line) and BEFORE the `// 4) Mode mutual exclusivity → exit 2 (PRD §6.3)` comment. Branch-anchored
// edits are immune to the line drift. (verified_facts.md §1.)

// GOTCHA #2 — Place the guard BEFORE exclusivity (primary), not before init dispatch. The contract
// allows either ("before exclusivityError OR before init dispatch; either is correct as long as it
// precedes runInit"). This PRP picks BEFORE exclusivity because: (a) decisions.md §D2's phrasing
// ("after the unknown-flag guard, before the exclusivity/init dispatch") reads as "right after
// unknownFlag"; (b) a missing flag value is a flag-PARSE error like unknownFlag, so grouping
// parse-errors-before-exclusivity matches the existing "more fundamental errors first" ordering
// (the :441-443 comment already says unknownFlag precedes exclusivity so --bogus foo --list reports
// the unknown flag first); (c) it fires earlier, so `init --store --list` reports the missing value
// (more fundamental) before the exclusivity conflict. If you choose before-init-dispatch instead,
// the only behavioral difference is `init --store <conflicting-mode>` reports exclusivity first —
// either is acceptable; document which you picked.

// GOTCHA #3 — The guard is UNCONDITIONAL on c.init: `if c.storeMissingValue`, NOT
// `if c.init && c.storeMissingValue`. The contract LOGIC §3 fixes it unconditional. This means bare
// `--store` (no init token, no value — c.init=false, storeMissingValue=true) ALSO exits 2. That is
// a behavior change from the current exit-1-usage, and it is the INTENDED, more-helpful outcome
// (the bug writeup's "Suggested Fix": make missing-value flags a hard error exit 2). Cover it with
// a test. Do NOT gate on c.init — that would re-admit the bare-`--store` no-op. (verified_facts.md §5.)

// GOTCHA #4 — Message to STDERR via fmt.Fprintln; stdout stays EMPTY. The contract is explicit
// (§6.4 discipline: `pi --skill "$(skilldozer init --store)"` must fail loudly, not capture a
// partial path). Use fmt.Fprintln(stderr, "skilldozer: --store requires a value") — Fprintln adds
// the trailing \n and matches the exclusivityError branch's fmt.Fprintln(stderr, msg) style for a
// fixed string. Do NOT use fmt.Fprintf unless you have a %s arg (unknownFlag uses Fprintf only for
// its '%s' substitution). Do NOT print anything to stdout.

// GOTCHA #5 — The EXACT message is `skilldozer: --store requires a value` (contract LOGIC §3 +
// decisions.md §D2). Include the `skilldozer:` prefix (matches unknownFlag's `skilldozer: unknown
// flag '%s'`). Assert it with EXACT equality in the test (errOut.String() ==
// "skilldozer: --store requires a value\n"), mirroring TestRunDefaultUnknownFlag — do NOT use
// Contains, which would hide a wording/prefix drift.

// GOTCHA #6 — Bare `init` (no --store) must STILL PROMPT. c.storeMissingValue is false there
// (T2.S1 sets it ONLY in the two --store branches). The guard does not fire, runInit runs, the
// interactive prompt happens (PRD §8.2). Do NOT add any condition that would make bare init exit 2.
// The contract INPUT §2 note ("the signal is only meaningful when --store was used to drive init")
// describes WHY the signal exists, NOT a gate condition on c.init — the guard is unconditional
// (GOTCHA #3). A regression test (bare init still reaches runInit) is covered by the existing
// TestRunInitStoreWritesConfig... and the prompt tests; just don't break them.

// GOTCHA #7 — Zero existing-test breakage (grep-verified). The ONLY run()-level test passing
// --store is TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 (:2390), which uses a VALID
// value → storeMissingValue=false → guard does not fire → exit 0 unchanged. No run()-level test
// passes a no-value --store shape. The T2.S1 parse-level tests (TestParseArgs*…SetsSignal) call
// parseArgs, not run, so they are unaffected. Net: this guard is purely additive. (verified_facts
// §6.) Still re-run `go test ./...` to confirm.

// GOTCHA #8 — The "config NOT written" test must neutralize BOTH the env rule and the walk-up
// rule (else runInit's resolveStore could pick a different dir, though it never runs because the
// guard fires first — but set them anyway for a clean, deterministic fixture). Mirror
// TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0's setup: t.Setenv("SKILLDOZER_CONFIG",
// <tmp cfg>), t.Setenv("SKILLDOZER_SKILLS_DIR", ""), t.Chdir(t.TempDir()). Do NOT call
// unsetSkillsEnv() for the config-write tests — it points SKILLDOZER_CONFIG at a NON-EXISTENT
// path (hermeticity helper); here you WANT a real, pre-written config file to assert it survives.

// GOTCHA #9 — Do NOT touch parseArgs, the config struct, or the --store branches. Those are T2.S1
// (parallel). This subtask ONLY reads c.storeMissingValue in run() + adds tests. Touching parseArgs
// collides with T2.S1. (The field already exists at main.go:142 from T2.S1's landed portion.)

// GOTCHA #10 — No deps change. The guard is `if c.storeMissingValue { fmt.Fprintln(...); return 2 }`
// — bool field read + Fprintln + return, all already-imported (fmt, io.Writer). go.mod and go.sum
// must be byte-for-byte identical. Verify with `git diff --quiet go.mod go.sum`.
```

---

## Implementation Blueprint

### Data models and structure

**No data-model changes.** This subtask reads the `c.storeMissingValue bool` field T2.S1 added (at `main.go:142`) and returns an int exit code. No new fields, types, methods, or signatures.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — insert the guard block into run() (step 3.5)
  - FILE: main.go (func run(), between the unknownFlag block and the exclusivityError block)
  - INSERT immediately AFTER the unknownFlag block (`if c.unknownFlag != "" { ...; return 2 }` +
    its trailing blank line) and BEFORE the `// 4) Mode mutual exclusivity → exit 2 (PRD §6.3)`
    comment. Exact block:
        // 3.5) --store presented without its value → exit 2 (PRD §6 header "Unknown flags
        //      ⇒ error + exit 2"; delta-PRD §2 #3 "init is non-destructive"). A value-taking
        //      flag with no value is a parse error, NOT a silent fall-through to destructive
        //      auto-detect init. Rejecting here — BEFORE the init dispatch — means runInit /
        //      setupStore / configpkg.Save is NEVER called, so a pre-existing config.yaml's
        //      `store:` value is preserved (Issue 2). stdout stays EMPTY (§6.4 discipline).
        //      The signal is set by parseArgs in BOTH --store no-value branches (P1.M1.T2.S1);
        //      it is NOT set by bare `init` (c.initStore=="" legitimately means "prompt").
        if c.storeMissingValue {
            fmt.Fprintln(stderr, "skilldozer: --store requires a value")
            return 2
        }
  - GOTCHA #1: insert by BRANCH (after unknownFlag, before the exclusivity comment), not line
    number — T2.S1's pending next-token else arm shifts run() down.
  - GOTCHA #2: BEFORE exclusivity (primary position); document if you pick before-init instead.
  - GOTCHA #3: UNCONDITIONAL on c.init (`if c.storeMissingValue`, not `if c.init && ...`).
  - GOTCHA #4: fmt.Fprintln to stderr; nothing to stdout.
  - GOTCHA #5: exact message `skilldozer: --store requires a value` (with the `skilldozer:` prefix).

Task 2: EDIT main.go — update the run() precedence comment (Mode A)
  - FILE: main.go (the run() doc-comment precedence line, ~:412-413 today)
  - CURRENT ladder line:
        //     help → version → unknownFlag → exclusivity → dispatch → no-args-usage.
  - REPLACE with:
        //     help → version → unknownFlag → storeMissingValue (--store needs a value) → exclusivity → dispatch → no-args-usage.
  - Keep the two intro lines above it unchanged. Mode-A honesty: the comment must list every
    dispatch step or it drifts (same rule the sibling Issue-1/Issue-2-parse PRPs applied).

Task 3: EDIT main_test.go — add 3 exit-2 run()-level tests (mirror TestRunDefaultUnknownFlag)
  - FILE: main_test.go (insert near the other run()/exit-2 tests, e.g. after TestRunDefaultUnknownFlag
    at :312, or grouped with the init tests near :1736; either is fine — keep parse-level --store
    tests and run-level --store tests visually separate with a section comment)
  - ADD (package main; var out, errOut bytes.Buffer; run returns int):
    // Issue 2 (P1.M1.T2.S2): `init --store` (trailing, no value) → exit 2, empty stdout, exact
    // stderr. The destructive bug: previously degraded to auto-detect init that overwrote the config.
    func TestRunInitStoreNoValueExits2(t *testing.T) {
        var out, errOut bytes.Buffer
        code := run([]string{"init", "--store"}, &out, &errOut)
        if code != 2 {
            t.Fatalf("run(init --store): code=%d; want 2 (missing --store value, PRD §6)", code)
        }
        if out.Len() != 0 {
            t.Errorf("stdout=%q; want EMPTY (§6.4: nothing on stdout on exit-2)", out.String())
        }
        want := "skilldozer: --store requires a value\n"
        if got := errOut.String(); got != want {
            t.Errorf("stderr=%q; want %q", got, want)
        }
    }
    // Issue 2: `--store=` (empty '='-form value) → exit 2 + empty stdout + exact stderr.
    func TestRunStoreEqualsEmptyExits2(t *testing.T) {
        var out, errOut bytes.Buffer
        code := run([]string{"--store="}, &out, &errOut)
        if code != 2 { t.Fatalf("run(--store=): code=%d; want 2", code) }
        if out.Len() != 0 { t.Errorf("stdout=%q; want EMPTY", out.String()) }
        if got, want := errOut.String(), "skilldozer: --store requires a value\n"; got != want {
            t.Errorf("stderr=%q; want %q", got, want)
        }
    }
    // Issue 2: bare `--store` (no init token, no value) → exit 2. Was exit-1-usage before the
    // fix; the guard makes it a precise "requires a value" error (the bug writeup's Suggested Fix).
    func TestRunStoreBareNoValueExits2(t *testing.T) {
        var out, errOut bytes.Buffer
        code := run([]string{"--store"}, &out, &errOut)
        if code != 2 { t.Fatalf("run(--store): code=%d; want 2 (bare --store, no value)", code) }
        if out.Len() != 0 { t.Errorf("stdout=%q; want EMPTY", out.String()) }
        if got, want := errOut.String(), "skilldozer: --store requires a value\n"; got != want {
            t.Errorf("stderr=%q; want %q", got, want)
        }
    }

Task 4: EDIT main_test.go — add the load-bearing "config NOT written" test
  - FILE: main_test.go (group with Task 3's tests)
  - ADD (mirror TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0's setup, INVERTED):
    // Issue 2 (P1.M1.T2.S2): the non-destructive contract. A pre-existing config.yaml with a
    // valid `store:` must survive `init --store` (no value) byte-for-byte — the guard returns
    // before runInit/setupStore/configpkg.Save. Mirrors TestRunInitStoreWritesConfig... setup,
    // inverted: pre-write the config, run the no-value form, assert it is UNCHANGED.
    func TestRunInitStoreNoValueDoesNotWriteConfig(t *testing.T) {
        cfg := filepath.Join(t.TempDir(), "config.yaml")
        originalStore := "/tmp/B/realstore" // the value that must NOT be clobbered
        if err := os.WriteFile(cfg, []byte("store: "+originalStore+"\n"), 0o644); err != nil {
            t.Fatalf("write config: %v", err)
        }
        before, err := os.ReadFile(cfg)
        if err != nil { t.Fatalf("read config before: %v", err) }

        t.Setenv("SKILLDOZER_CONFIG", cfg)    // point config.Path at our pre-written fixture
        t.Setenv("SKILLDOZER_SKILLS_DIR", "") // env unset so config rule is the relevant one
        t.Chdir(t.TempDir())                   // escape the repo's walk-up rule (deterministic)

        var out, errOut bytes.Buffer
        code := run([]string{"init", "--store"}, &out, &errOut)
        if code != 2 {
            t.Fatalf("run(init --store): code=%d; want 2 (missing value, config must NOT be written)", code)
        }
        // §6.4: stdout stays empty.
        if out.Len() != 0 {
            t.Errorf("stdout=%q; want EMPTY", out.String())
        }
        // THE LOAD-BEARING ASSERTION: the config file is byte-for-byte unchanged.
        after, err := os.ReadFile(cfg)
        if err != nil { t.Fatalf("read config after: %v", err) }
        if !bytes.Equal(before, after) {
            t.Errorf("config was modified by a missing-value --store (must be non-destructive):\nbefore=%q\nafter =%q", before, after)
        }
        // Semantic re-check via the config loader (Store value preserved):
        f, err := configpkg.Load(cfg)
        if err != nil { t.Fatalf("config.Load: %v", err) }
        if f.Store != originalStore {
            t.Errorf("config.Store=%q; want %q (must NOT be overwritten)", f.Store, originalStore)
        }
    }
  - GOTCHA #8: set SKILLDOZER_CONFIG to the real pre-written cfg (NOT unsetSkillsEnv, which
    points it at a non-existent path). Set SKILLDOZER_SKILLS_DIR="" + t.Chdir for determinism.
  - bytes + configpkg.Load are both already imported in main_test.go (configpkg at :12).

Task 5: VERIFY in isolation + whole module + invariants
  - COMMAND: gofmt -l main.go main_test.go   (must print NOTHING)
  - COMMAND: go vet ./...                     (exit 0)
  - COMMAND: go build ./...                   (exit 0)
  - COMMAND: go test -run 'TestRun(Init)?Store|TestRunStore' -v ./...  (the 4 new tests pass)
  - COMMAND: go test ./...                    (whole module green; zero regressions — GOTCHA #7)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"  (GOTCHA #10)
  - INVARIANT: grep -c 'requires a value' main.go   (expect 2: the guard message + the precedence-comment label)
  - INVARIANT: the existing TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 still passes
    (valid --store value → storeMissingValue=false → guard does not fire → exit 0).
  - END-TO-END (the §ISSUE 2 repro, now fixed): build, write a config, run `init --store`,
    assert exit 2 + config unchanged (see Validation Loop Level 3).
```

### Implementation Patterns & Key Details

```go
// The guard (Task 1) — inserted in run() between unknownFlag and exclusivityError:
// 3.5) --store presented without its value → exit 2 ...
if c.storeMissingValue {
	fmt.Fprintln(stderr, "skilldozer: --store requires a value")
	return 2
}

// The precedence comment (Task 2) — one ladder line updated:
//     help → version → unknownFlag → storeMissingValue (--store needs a value) → exclusivity → dispatch → no-args-usage.
```

Notes easy to get wrong:
- The guard is **unconditional on `c.init`** (`if c.storeMissingValue`). Bare `--store` (no init token) also exits 2 — that's the intended improvement (GOTCHA #3/#5). Do not gate on `c.init`.
- Use `fmt.Fprintln` (not `Fprintf`) — the message is a fixed string with no args; `Fprintln` adds the `\n` and matches the `exclusivityError` branch style.
- Assert the stderr message with **exact equality** in tests (mirror `TestRunDefaultUnknownFlag`), not `Contains` — the contract fixes the string verbatim.
- The "config NOT written" test pre-writes the config and asserts `bytes.Equal(before, after)` — the strongest proof of non-destructiveness. The `configpkg.Load` check is the semantic re-assertion.

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Guard position: before exclusivity (primary), not before init dispatch.** The contract allows either. decisions.md §D2's phrasing and the "more fundamental errors first" convention (unknownFlag already precedes exclusivity) both favor before-exclusivity. It also makes `init --store --list` report the missing value before the exclusivity conflict (more fundamental). Documented; before-init-dispatch is an acceptable alternative.
2. **Unconditional on c.init? → YES.** The contract LOGIC §3 fixes `if c.storeMissingValue` (no `c.init &&`). This makes bare `--store` exit 2 (was exit 1) — the intended "missing-value flags are hard errors" fix. Gating on c.init would re-admit the bare-`--store` no-op.
3. **Message via Fprintln vs Fprintf? → Fprintln.** Fixed string, no args; matches the exclusivityError branch. Fprintf is only used where there's a `%s` (unknownFlag).
4. **Exact vs Contains stderr assertion? → EXACT.** Mirror TestRunDefaultUnknownFlag; locks the verbatim contract string and catches prefix/wording drift.
5. **Config-not-written proof: bytes.Equal vs configpkg.Load? → BOTH.** bytes.Equal is the strict byte-level proof; configpkg.Load is the semantic re-check that `Store` survived. Belt-and-suspenders for the load-bearing non-destructive contract.
6. **Touch parseArgs or the --store branches? → NO.** T2.S1 (parallel) owns the field + setters (already landed at :142, :193-202; next-token arm pending). This subtask only reads the field in run(). Touching parseArgs collides with T2.S1.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. No new imports (bool read + fmt.Fprintln + return). (GOTCHA #10)

DISPATCH (the only integration point — the guard slots into the existing run() ladder):
  before: help → version → unknownFlag → exclusivity → init-dispatch → normal-modes → no-args-usage
  after:  help → version → unknownFlag → STORE-MISSING-VALUE(guard) → exclusivity → init-dispatch → ...
  The guard reads c.storeMissingValue (set by T2.S1 in parseArgs) and returns 2 before runInit.

CONSUMERS:
  - The §13-style acceptance + the §ISSUE 2 repro: `init --store` (no value) now exits 2 + config untouched.
  - README error-contract section: swept by the final Mode B task (P1.M3.T1) — no doc file rides here
    beyond the run() precedence comment (Mode A, Task 2).

PARALLEL SIBLING (no conflict):
  - P1.M1.T2.S1 (parse signal) edits the config struct (:142) + parseArgs (:193-202, :263+). T2.S2
    edits run() (~:438) + main_test.go run()-level tests. DISJOINT regions; land in either order.
    T2.S2 depends on T2.S1's field existing (it does, :142) — the guard compiles regardless of
    whether T2.S1's next-token else arm has landed yet (the field is what's read).

NO ROUTES / NO DATABASE / NO CONFIG-FORMAT CHANGE.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after the main.go edit)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l main.go main_test.go   # must print NOTHING (run `gofmt -w` if it lists a file)
go vet ./...                     # expect exit 0
go build ./...                   # expect exit 0 (proves the guard compiles + c.storeMissingValue exists)
# Expected: zero output / exit 0.
```

### Level 2: The run()-level unit tests (the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test -run 'TestRun(Init)?Store|TestRunStore' -v ./...
# Expected: ALL 4 pass. The load-bearing assertions:
#   TestRunInitStoreNoValueExits2             -> ["init","--store"] code 2, empty stdout, exact stderr
#   TestRunStoreEqualsEmptyExits2             -> ["--store="]       code 2, empty stdout, exact stderr
#   TestRunStoreBareNoValueExits2             -> ["--store"]        code 2, empty stdout, exact stderr (was exit 1)
#   TestRunInitStoreNoValueDoesNotWriteConfig -> pre-written config byte-for-byte UNCHANGED + Store preserved
# A wrong message/prefix fails the exact-equality assert. A guard that fires for valid --store
# (storeMissingValue=false) would break the existing TestRunInitStoreWritesConfig... (run next).
```

### Level 3: Whole-module regression + the §ISSUE 2 end-to-end repro

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # 0
go vet  ./...  ; echo "vet exit $?"     # 0
go test ./...  ; echo "test exit $?"    # 0  — CRITICAL: zero regressions (the existing
                                       #   TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 + all
                                       #   parse/exclusivity/init tests still pass; GOTCHA #7)

# GOTCHA #10 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"

# Scope invariants:
grep -c 'requires a value' main.go      # expect 2 (guard message + precedence-comment label)
grep -c 'storeMissingValue' main_test.go # the new run()-level tests reference it (T2.S1's parse tests too)

# END-TO-END — the §ISSUE 2 repro, now FIXED (exit 2 + config untouched):
go build -o /tmp/sd .
tmp=$(mktemp -d) && printf 'store: /tmp/B/realstore\n' > "$tmp/cfg.yaml"
before=$(cat "$tmp/cfg.yaml")
SKILLDOZER_CONFIG="$tmp/cfg.yaml" env -u SKILLDOZER_SKILLS_DIR /tmp/sd init --store </dev/null >"$tmp/out" 2>"$tmp/err"; rc=$?
[ "$rc" = 2 ] && grep -q 'skilldozer: --store requires a value' "$tmp/err" && [ ! -s "$tmp/out" ] \
  && [ "$(cat "$tmp/cfg.yaml")" = "$before" ] && echo "Issue-2 repro FIXED: exit 2 + msg + empty stdout + config unchanged" \
  || { echo "FAIL: rc=$rc out=$(cat "$tmp/out") err=$(cat "$tmp/err") cfg=$(cat "$tmp/cfg.yaml")"; }
rm -rf "$tmp" /tmp/sd
# Expected: "Issue-2 repro FIXED: exit 2 + msg + empty stdout + config unchanged".
```

### Level 4: Behavioral spot-checks (lock the precedence + the bare-init-still-prompts guarantee)

```bash
cd /home/dustin/projects/skilldozer

# 4a. Precedence: --help and --version still WIN over the missing-value guard (PRD §6.3).
go build -o /tmp/sd .
# `init --store --help` → help wins (exit 0, usage on stdout), NOT exit 2:
/tmp/sd init --store --help >/dev/null 2>&1; echo "help-precedence exit=$? (want 0)"
# `init --store --version` → version wins (exit 0):
/tmp/sd init --store --version >/dev/null 2>&1; echo "version-precedence exit=$? (want 0)"
# Expected: both exit 0 (the guard runs AFTER help/version in run()'s ladder).

# 4b. The CRITICAL non-regression: bare `init` (no --store) must STILL PROMPT / proceed to
#     runInit (storeMissingValue is false). Feed /dev/null so a non-TTY init does NOT hang:
cfg=$(mktemp -d)/c.yaml
SKILLDOZER_CONFIG="$cfg" env -u SKILLDOZER_SKILLS_DIR /tmp/sd init </dev/null >/dev/null 2>&1; echo "bare-init exit=$? (non-0 means it ran runInit, NOT the guard; want non-0, typically 1 or 0)"
# Expected: non-2 exit (it reached runInit — the guard did NOT fire). If it exits 2, the guard
# wrongly fired for bare init (storeMissingValue was set incorrectly — a T2.S1 bug, not this subtask).

# 4c. Valid --store value still works end-to-end (guard does not fire; config written):
store=$(mktemp -d); cfg2=$(mktemp -d)/c.yaml
SKILLDOZER_CONFIG="$cfg2" env -u SKILLDOZER_SKILLS_DIR /tmp/sd init --store "$store" </dev/null >/dev/null 2>&1; echo "valid-store exit=$? (want 0)"
grep -q "store: $store" "$cfg2" && echo "valid-store config written OK" || echo "FAIL: config not written"
rm -rf "$(dirname "$cfg")" "$store" "$(dirname "$cfg2")" /tmp/sd
# Expected: exit 0 + config written (the guard is inert for a valid value).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l main.go main_test.go` empty; `go vet ./...` exit 0; `go build` exit 0
- [ ] Level 2 PASS — the 4 new run()-level tests pass (3 exit-2 shapes with exact stderr + 1 config-NOT-written)
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (zero regressions); `git diff go.mod go.sum` → "deps unchanged"; `grep -c 'requires a value' main.go` == 2; the §ISSUE 2 repro now exits 2 + config unchanged
- [ ] Level 4 PASS — `--help`/`--version` win over the guard; bare `init` still reaches runInit (non-2 exit); valid `--store <dir>` still writes the config

### Feature Validation
- [ ] `run(["init","--store"])`, `run(["--store="])`, `run(["--store"])` → exit 2, stdout EMPTY, stderr == `"skilldozer: --store requires a value\n"`
- [ ] A pre-existing config.yaml's `store:` is byte-for-byte unchanged after `init --store` (non-destructive)
- [ ] Bare `init` (no `--store`) still proceeds to runInit/prompts (signal false; no regression)
- [ ] Valid `init --store <dir>` still exits 0 and writes the config (signal false; existing test unchanged)
- [ ] The guard is placed before exclusivity (or documented if before-init) and before init dispatch

### Code Quality / Convention Validation
- [ ] Mirrors the `unknownFlag`/`exclusivityError` branch style (`fmt.Fprintln`/`Fprintf` to stderr, `return 2`)
- [ ] Tests mirror `TestRunDefaultUnknownFlag` (exact stderr) + `TestRunInitStoreWritesConfig...` (config setup)
- [ ] Exact-equality stderr assertion (not Contains)
- [ ] No new imports; no new deps; go.mod/go.sum byte-for-byte identical
- [ ] No new files; both edits to `main.go` + `main_test.go`

### Scope Discipline (the T2.S1 boundary + Issue-1 boundary)
- [ ] Did NOT touch `parseArgs`, the `config` struct, or the `--store` branches (T2.S1, parallel)
- [ ] Did NOT touch `runInit` / `setupStore` / the check-report streams (Issue 1 = P1.M1.T1.S1)
- [ ] Did NOT change any OTHER exit code (only adds the missing-value → exit 2 path)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't insert by line number.** T2.S1's pending next-token `else` arm shifts `run()` down ~5 lines. Insert by BRANCH (after the `unknownFlag` block, before the `// 4) Mode mutual exclusivity` comment). (GOTCHA #1.)
- ❌ **Don't gate the guard on `c.init`.** The contract fixes `if c.storeMissingValue` (unconditional). Bare `--store` (c.init=false) must also exit 2 — that's the intended fix. Gating on c.init re-admits the no-op. (GOTCHA #3.)
- ❌ **Don't print to stdout, and don't use `Contains` for the message.** The message goes to STDERR via `fmt.Fprintln`; stdout stays EMPTY (§6.4). Assert the exact string (`errOut.String() == "skilldozer: --store requires a value\n"`), mirroring `TestRunDefaultUnknownFlag`. (GOTCHA #4/#5.)
- ❌ **Don't drop the `skilldozer:` prefix or paraphrase the message.** The exact string is `skilldozer: --store requires a value` (contract + decisions.md §D2); the prefix matches `unknownFlag`'s style.
- ❌ **Don't forget the "config NOT written" test.** It is the load-bearing proof of the non-destructive contract — the whole point of Issue 2. Use `bytes.Equal(before, after)` + `configpkg.Load(...).Store` unchanged. (Task 4.)
- ❌ **Don't use `unsetSkillsEnv()` for the config-write test.** It points `SKILLDOZER_CONFIG` at a non-existent path (hermeticity). Here you WANT a real pre-written config to assert it survives — set `SKILLDOZER_CONFIG` to the fixture directly. (GOTCHA #8.)
- ❌ **Don't touch `parseArgs` / the `config` struct / the `--store` branches.** Those are T2.S1 (parallel). This subtask only reads `c.storeMissingValue` in `run()`. (GOTCHA #9.)
- ❌ **Don't place the guard AFTER the init dispatch.** It MUST precede `if c.init { return runInit }` or the destructive `setupStore`/`configpkg.Save` runs first. Before-exclusivity (primary) or before-init-dispatch — both precede runInit. (GOTCHA #2.)
- ❌ **Don't skip the precedence-comment update.** Mode-A honesty: the `run()` doc comment lists the dispatch ladder; adding a step without updating the comment makes the code lie about its own behavior. (Task 2.)
- ❌ **Don't add deps or imports.** A bool read + `fmt.Fprintln` + `return` use only existing constructs. go.mod/go.sum must be byte-for-byte unchanged. (GOTCHA #10.)

---

## Confidence Score

**9.5/10** — This is a one-block, purely-additive `run()` guard with the insertion point pinned by branch content, the exact message fixed by the contract + decisions.md §D2, the message-routing (`Fprintln` to stderr) mirroring the existing `exclusivityError` branch, and the non-destructive chain (`setupStore`→`configpkg.Save` at `:979`, called by `runInit` at `:1012`, never reached because the guard returns before `runInit` at `:455`) grep-verified. The zero-breakage surface is grep-confirmed (only one `run()` test passes `--store`, with a valid value; no run()-level test exercises a no-value shape). The one analytical judgment call — bare `--store` (no init token) now exits 2 instead of 1 — is explicitly resolved as the intended improvement (the bug writeup's "Suggested Fix"), test-locked, and documented. The 0.5 reservation is for the parallel-execution seam with T2.S1: the guard compiles the moment the `storeMissingValue` field exists (it does, `:142`), but the run()-level test for `init --store` (next-token shape) only passes once T2.S1 lands its next-token `else` arm — which is T2.S1's responsibility, not this subtask's, and the contract says assume T2.S1 lands fully.
