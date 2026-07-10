# PRP — P1.M2.T2.S2: `run()` dispatch + shell detection + exit codes

> **Subtask:** The behavior half of the `skilldozer completion` subcommand (PRD §14.6 + §6.4). Adds THREE pieces: (a) a dispatch slot in `run()` — `if c.completion { return runCompletion(c, stdout, stderr) }` — inserted immediately AFTER the `if c.init` block and BEFORE the path/list/search/check/all/tags ladder, mirroring how `init` is dispatched as an exclusive mode; (b) `detectShell(explicit, envShell, loginShell string) (string, bool)` — a pure, env-mutation-free function returning the first non-empty of the three (PRD §14.6 "Shell detection, first wins") or `("", false)` — plus its `loginShellBase()` helper that does `strings.ToLower(filepath.Base(os.Getenv("SHELL")))` with an empty guard (because `filepath.Base("")` → `"."`); (c) `runCompletion(c config, stdout, stderr io.Writer) int` — resolves the shell, then on success writes the embedded script to stdout + exit 0; on undetectable shell writes the §6.4 message to stderr + exit 1; on an unsupported value (e.g. `tcsh`, or `$SHELL=/bin/sh`→`sh`) writes the §6.4 message to stderr + exit 2. **It does NOT embed the scripts** (that is P1.M2.T2.S1's `completionScript`, already landed) and **it does NOT parse `completion`/`--shell`** (that is P1.M2.T1.S1, Complete). After this subtask, `skilldozer completion` works end-to-end and the §13 acceptance assertions pass.
>
> **Scope:** Two existing files only — `main.go` (ONE dispatch line in `run()` + THREE functions appended after `runInit` at the file tail; **zero new imports** — `os`/`fmt`/`io`/`path/filepath`/`strings` are all already imported) and `main_test.go` (8 unit tests covering the dispatch exit-code ladder + env/login detection + `detectShell`/`loginShellBase` purity; **zero new imports**). No new files. No `internal/*` change. No `completions/*` change. `go.mod`/`go.sum` byte-for-byte unchanged.
>
> **STATUS (verified at PRP-write time):** main.go + main_test.go read directly. The local `config` struct CONFIRMED to have `completion bool` + `completionShell string` (P1.M2.T1.S1 = Complete). `exclusivityError` CONFIRMED to already reject completion+other-modes (so `runCompletion` is only ever called standalone). `completionScript(shell string) (string, bool)` CONFIRMED present at main.go:1110 (P1.M2.T2.S1's code already landed; the three embed vars at 55/58/61). The `run()` init dispatch block + the `// 5) Normal mode dispatch` comment located (insertion point verified). The three grep substrings the tests/§13 rely on CONFIRMED present (`_skilldozer_completion`×2 in bash, `complete -c skilldozer`×14 in fish, `#compdef skilldozer`×1 in zsh). `runInit` CONFIRMED the last function (file ends line 1204) — append target. main.go import block CONFIRMED to contain all needed stdlib packages → zero new imports. The parallel sibling P1.M2.T2.S1 PRP was read as a CONTRACT (disjoint regions: it edits the import block + after-`var version` + before-`runInit`-doc; this subtask edits the `run()` dispatch slot + after-`runInit` tail).

---

## Goal

**Feature Goal**: Wire the `completion` subcommand end-to-end so PRD §14.6 and §6.4 hold and are provably correct: `skilldozer completion --shell bash|zsh|fish` emits the matching embedded script to stdout (exit 0); `skilldozer completion` auto-detects the shell via `--shell` → `$SKILLDOZER_SHELL` → `basename($SHELL)` (first wins); an undetectable shell (all three empty) → stderr + exit 1, nothing on stdout; an unsupported `--shell` value (not bash/zsh/fish) → stderr + exit 2, nothing on stdout. All four exit-code paths and the two auto-detection paths are locked by 8 unit tests, none of which mutate the env in the pure-function tests.

**Deliverable**: Additive edits to two existing files:
1. `main.go` — insert `if c.completion { return runCompletion(c, stdout, stderr) }` into `run()` (after the `if c.init` block, before the `// 5) Normal mode dispatch` comment); append `detectShell`, `loginShellBase`, and `runCompletion` after `runInit` (the file tail).
2. `main_test.go` — 8 tests: 6 dispatch/exit-code tests (bash→0, fish→0, tcsh→2, no-shell→1, `$SKILLDOZER_SHELL=zsh`→0, `$SHELL=/bin/zsh`→0), 1 `detectShell` table test, 1 `loginShellBase` table test.

**Success Definition**: `gofmt -l main.go main_test.go` empty; `go build/vet/test ./...` all pass; `go.mod`/`go.sum` unchanged; `run(["completion","--shell","bash"])` ⇒ code 0 + stdout Contains `_skilldozer_completion` + stderr empty; `run(["completion","--shell","fish"])` ⇒ code 0 + stdout Contains `complete -c skilldozer`; `run(["completion","--shell","tcsh"])` ⇒ code 2 + stdout empty + stderr mentions "tcsh"; `run(["completion"])` with no shell env ⇒ code 1 + stdout empty + stderr Contains "shell"; `detectShell` table (explicit>env>login; all-empty→false) passes; `loginShellBase` table (basename + lowercase + empty-guard) passes.

---

## User Persona (if applicable)

**Target User**: A user who wants shell completions and runs `eval "$(skilldozer completion)"` (or `skilldozer completion --shell fish | source`), AND scripts/CI that pin the shell with `--shell`. The detection + exit-code logic serves both without the second ever needing to set `$SHELL`.

**Use Case**: User runs `eval "$(skilldozer completion)"` with no flags → skilldozer detects `$SHELL` (or `$SKILLDOZER_SHELL`), emits the matching script, the parent shell eval's it. A deterministic install pins `--shell bash`. A typo (`--shell tcsh`) fails loudly with exit 2 (so the `$(...)` captures nothing, not a partial/garbage script).

**User Journey**: `run()` sees `c.completion` (set by parseArgs from the `completion` token / `--shell` flag) → dispatches to `runCompletion` → `detectShell` picks the shell (explicit→env→login) → `completionScript` returns the embedded bytes → stdout, exit 0. On the two failure paths nothing reaches stdout (§6.4 `$(...)` contract).

**Pain Points Addressed**: a child process cannot register completions in the parent shell — emitting-to-stdout-for-`eval` is the only correct idiom (`zoxide init`/`starship init`/`direnv hook`); a `go install` user with no clone still gets completions (the bytes are embedded, S1); deterministic `--shell` for reproducible `eval`; loud failure (exit 1/2, empty stdout) so `pi --skill "$(skilldozer completion)"`-style misuse fails instead of silently passing garbage.

---

## Why

- **Implements PRD §14.6 (Shell detection + the `completion` emit semantics) and §6.4 (the completion exit-code contract).** S1 embedded the bytes + `completionScript`; this subtask makes them reachable from `skilldozer completion` with the correct shell resolution + exit codes.
- **Closes the §13 acceptance gate for `completion`.** Four §13 assertions target THIS subtask: `completion --shell bash` grep, `completion --shell fish` grep, the undetectable-shell `env -u SHELL -u SKILLDOZER_SHELL … exit 1` case, and the `--shell tcsh` exit-2 case.
- **Honors the load-bearing §6.4 stdout/stderr separation.** The script goes to stdout ONLY on success; the two failure paths write NOTHING to stdout (so `$(...)` stays empty and `eval` fails cleanly). This is the same contract that protects `skilldozer <tag>` resolution.
- **Keeps the binary dependency-free.** Pure stdlib (`os.Getenv`/`filepath.Base`/`strings.ToLower`/`fmt.Fprint`) + the repo's own `completionScript` (S1). Zero new imports. yaml.v3 stays the sole non-stdlib module.
- **Unblocks P1.M3** (completion-file lockstep + README docs), which depends on `completion` being a working command. After this subtask `skilldozer completion` is end-to-end functional.

---

## What

A 1-line dispatch insertion, three appended functions (~30 lines total), and 8 tests. No behavior change to any other mode (the `if c.completion` slot returns before the normal-mode ladder exactly as `if c.init` does).

### Success Criteria

- [ ] `main.go` `run()` inserts `if c.completion { return runCompletion(c, stdout, stderr) }` AFTER the `if c.init { return runInit(c, stdout, stderr) }` block and BEFORE the `// 5) Normal mode dispatch` comment.
- [ ] `main.go` appends `func detectShell(explicit, envShell, loginShell string) (string, bool)` (pure; first non-empty wins; all-empty → `("", false)`), `func loginShellBase() string` (`os.Getenv("SHELL")` empty-guarded → `strings.ToLower(filepath.Base(s))`), and `func runCompletion(c config, stdout, stderr io.Writer) int` (the §6.4 exit-code ladder) after `runInit` (file tail).
- [ ] `runCompletion`: `detectShell` → `!ok` ⇒ stderr `"could not detect shell; pass --shell {bash|zsh|fish}"` + return 1 (stdout untouched); `completionScript(shell)` → `!ok` ⇒ stderr `"skilldozer: unsupported shell '<v>' (want bash|zsh|fish)"` + return 2 (stdout untouched); else `fmt.Fprint(stdout, script)` + return 0.
- [ ] `go test ./...` green including the 8 new tests; existing tests unaffected (the `if c.completion` slot is the only run() change; today `completion` falls through to no-mode, so the new dispatch is the sole behavior delta for that input).
- [ ] `go.mod`/`go.sum` unchanged; no new files; `main.go` + `main_test.go` only.

---

## All Needed Context

### Context Completeness Check

**Pass.** Every function is pinned to a verified, live symbol or a fixed external_deps.md prescription: `completionScript(shell) (string, bool)` (main.go:1110, S1 — confirmed present), the local `config` struct's `completion`/`completionShell` fields (P1.M2.T1.S1 — Complete, confirmed), `exclusivityError`'s completion handling (already rejects completion+other-modes — so `runCompletion` is only called standalone). The `detectShell` + `loginShellBase` designs are fixed VERBATIM by external_deps.md §'Shell detection' (which verified the `filepath.Base` idiom + the empty-guard). The exit-code ladder is fixed by PRD §6.4/§14.6 + the contract. The dispatch insertion point is verified by reading `run()` source. The three grep substrings are confirmed present in the embedded files. The boundary with the parallel sibling (embed) is fixed (disjoint regions). An implementer who has never seen this repo can complete it in one pass.

### Documentation & References

```yaml
# MUST READ — the verified facts (the exact code + insertion point + the exit-code ladder + the test matrix)
- file: plan/003_3ace946c2a5c/P1M2T2S2/research/verified_facts.md
  why: "§1 = INPUT contract (c.completion/c.completionShell from P1.M2.T1.S1 Complete; completionScript
        from S1 already landed @1110). §2 = the EXACT run() insertion point (after the if c.init block,
        before the '// 5) Normal mode dispatch' comment; the contract's 'main.go:482/518' is STALE).
        §3 = the THREE new functions verbatim (detectShell/loginShellBase/runCompletion) + their tail
        placement after runInit. §4 = Fprint NOT Fprintln (byte-identity). §5 = the exit-code ladder
        (undetectable→1; unsupported value→2; incl. the $SHELL=/bin/sh→sh→exit-2 subtlety). §6 = ZERO
        new imports. §7 = the verified grep substrings. §8 = the 8-test matrix. §9 = sibling boundaries."
  critical: "§4 (Fprint, not Fprintln) and §5 (the two distinct failure paths: all-empty→exit 1 vs a
             detected-but-unsupported value like tcsh/sh→exit 2) are the two things most likely to be
             mishandled. §2's stale line anchors — anchor by SYMBOL, not line number."

# MUST READ — the authoritative detectShell/loginShellBase design (verified idiom)
- file: plan/003_3ace946c2a5c/architecture/external_deps.md
  why: "§'Shell detection' fixes detectShell (pure 3-arg, first non-empty wins) + loginShellBase
        (filepath.Base + empty guard + ToLower) + the exact call
        detectShell(c.completionShell, os.Getenv('SKILLDOZER_SHELL'), loginShellBase()). It VERIFIED
        filepath.Base('/bin/zsh')→'zsh' and the filepath.Base('')→'.' gotcha. §'Existing dependencies'
        confirms os/path/filepath/strings/fmt are all already imported (zero new imports)."
  section: "Shell detection (whole section)."

# MUST READ — the exact test matrix (dispatch + detectShell unit tests)
- file: plan/003_3ace946c2a5c/architecture/test_patterns.md
  why: "§'Completion subcommand — dispatch + exit codes' lists the 6 run()-level cases (bash→0,
        fish→0, tcsh→2, no-shell→1, $SKILLDOZER_SHELL honored, basename($SHELL) honored). §'detectShell
        unit test' lists the 5 pure cases (explicit>env>login; all-empty→false). These are the
        contract test set; the PRP expands them into 8 tests (adding a loginShellBase table)."

# MUST READ — the file under edit (locate symbols by NAME; line numbers shift as S1 lands)
- file: main.go
  why: "THE edit target. run() @476: the init dispatch block (`if c.init { return runInit(c, stdout,
        stderr) }`) and the following `// 5) Normal mode dispatch` comment — INSERT the completion
        slot BETWEEN them. completionScript @1110 (CONSUMED, S1). runInit @1135 is the LAST func; the
        file ends @1204 — APPEND detectShell/loginShellBase/runCompletion after runInit. Import block
        has os/fmt/io/path/filepath/strings already (ZERO new imports). NOTE: `config` is the LOCAL
        struct; runInit uses the `configpkg` alias for internal/config (irrelevant here — runCompletion
        takes the local config value c, not internal/config)."
  pattern: "Exclusive-subcommand dispatch = a self-contained `if c.X { return runX(c, stdout, stderr) }`
            branch AFTER the exclusivity gate, BEFORE the normal-mode ladder (mirror the init dispatch
            exactly). runX(c config, stdout, stderr io.Writer) int signature = runInit's. Pure helper
            + env-reading wrapper split (detectShell is pure; loginShellBase reads $SHELL) — the
            established testability pattern (cf. chooseStore/resolveStore)."

# MUST READ — the test file under edit (mirror these helpers/shapes exactly)
- file: main_test.go
  why: "THE other edit target + the test-template source. The run() tests use `var out, errOut
        bytes.Buffer; code := run([]string{...}, &out, &errOut)` + bytes/strings.Contains + exit-code
        assertions (cf. the run check/init tests). t.Setenv is used for env-driven tests (already the
        house style). The detectShell/loginShellBase tests are PURE (no run(), no env for detectShell).
        Package is `main` (white-box) so detectShell/loginShellBase/runCompletion are directly callable."
  gotcha: "The 'no shell' dispatch test MUST t.Setenv BOTH SKILLDOZER_SHELL='' AND SHELL='' (the test
           runner's own $SHELL would otherwise leak into loginShellBase). os.Getenv returns '' for
           unset-or-empty, so t.Setenv(v,'') is equivalent to the §13 `env -u`."

# READ-ONLY — the parallel sibling PRP (defines the completionScript INPUT + the disjoint boundary)
- file: plan/003_3ace946c2a5c/P1M2T2S1/PRP.md
  why: "Defines completionScript(shell)(string,bool) + the 3 embed vars, which runCompletion CALLS.
        Confirms S1 edits the import block (adds _ \"embed\") + after-var-version (55-61) +
        before-the-runInit-doc-comment (completionScript @1110). This subtask edits the run() dispatch
        slot (~535) + AFTER runInit (~1204 tail). DISJOINT regions; the changesets compose in either
        order. S1's completionScript is uncalled until THIS subtask wires runCompletion."

# READ-ONLY — PRD (the authority for detection order + the exit-code contract)
- file: PRD.md
  why: "§14.6 (h3.19) 'Shell detection (first wins)': --shell → $SKILLDOZER_SHELL → basename($SHELL) →
        none⇒exit 1. §6.4 (h3.4): completion cannot determine shell ⇒ stderr 'could not detect shell;
        pass --shell {bash|zsh|fish}', exit 1; unsupported --shell value ⇒ stderr, exit 2; on success
        the script goes to stdout, nothing else. §13 (h2.12): the four completion acceptance assertions.
        §17: no new runtime deps. decision 18 (completion = emit for eval, no rc writes)."
  section: "h3.19 (§14.6), h3.4 (§6.4), h2.12 (§13 completion assertions), h2.16 (§17), decision 18."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls main.go main_test.go go.mod completions/
main.go        main_test.go   go.mod
completions/:  skilldozer.bash   _skilldozer   skilldozer.fish
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole non-stdlib dep)
$ grep -n 'runCompletion\|detectShell\|loginShellBase' main.go   # (empty today — purely additive)
$ grep -n '^func completionScript' main.go   # 1110 (S1, CONSUMED — already present)
```

### Desired Codebase tree with files to be changed

```bash
main.go        # INSERT: `if c.completion { return runCompletion(...) }` in run(). APPEND: detectShell + loginShellBase + runCompletion after runInit.
main_test.go   # APPEND: 8 tests (6 dispatch/exit-code + 1 detectShell table + 1 loginShellBase table)
# go.mod / go.sum — UNCHANGED (stdlib env/string ops only; no new module)
# completions/* — UNCHANGED (P1.M3.T1.S1's concern; this subtask only EMITS the already-embedded bytes)
```

| File | Change | Owner |
|---|---|---|
| `main.go` | The `completion` dispatch slot + the 3 handler functions (detection + emission + exit codes) | Issue §14.6/§6.4 contract + external_deps.md |
| `main_test.go` | Lock the 4 exit-code paths + 2 detection paths + detectShell/loginShellBase purity | test_patterns.md + §13 |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 (CRITICAL — byte-identity) — use fmt.Fprint(stdout, script), NOT fmt.Fprintln. The
// embedded completion scripts already end with a trailing newline (the on-disk files do; //go:embed
// reads exact bytes; S1's TestEmbeddedCompletionsMatchOnDisk locks it). Fprintln would append an
// extra newline, violating §14.6 "emit identical bytes". (Acceptance only greps, so Fprintln wouldn't
// FAIL §13 — but Fprint is byte-correct and matches the contract.)

// GOTCHA #2 (CRITICAL — loginShellBase empty guard) — filepath.Base("") returns ".", so you MUST
// check `os.Getenv("SHELL") == ""` BEFORE calling filepath.Base, else an unset $SHELL would feed "."
// into detectShell and produce a bogus "detected" shell. external_deps.md verified this gotcha.
// loginShellBase: s := os.Getenv("SHELL"); if s == "" { return "" }; return strings.ToLower(filepath.Base(s)).

// GOTCHA #3 (the two distinct failure paths) — do NOT conflate "undetectable" (exit 1) with
// "unsupported value" (exit 2):
//   - ALL of explicit/envShell/loginShell empty ⇒ detectShell returns ("", false) ⇒ exit 1.
//   - a DETECTED but unsupported value (e.g. `--shell tcsh`, OR `$SHELL=/bin/sh` ⇒ loginShellBase()
//     returns "sh") ⇒ detectShell returns ("sh", true) ⇒ completionScript("sh")=("false") ⇒ exit 2.
// So `$SHELL=/bin/sh` is exit 2, NOT exit 1. Only the all-empty case is exit 1.

// GOTCHA #4 — detectShell returns values VERBATIM; only loginShellBase lowercases. So `--shell BASH`
// ⇒ "BASH" ⇒ completionScript("BASH")=("false") ⇒ exit 2. This follows the contract (the user is
// responsible for a valid lowercase name; explicit/envShell pass through as typed). Do NOT add
// lowercasing to detectShell — the test_patterns detectShell unit tests use lowercase inputs and the
// contract fixes detectShell as a verbatim first-non-empty selector.

// GOTCHA #5 — the dispatch slot goes AFTER `if c.init` and BEFORE the normal-mode ladder, mirroring
// init's exclusive-mode dispatch. exclusivityError (P1.M2.T1.S1, already landed) guarantees that when
// c.completion is true, NO other mode is active — so runCompletion never sees a conflicting mode and
// the slot can `return` unconditionally. Do NOT add a guard inside runCompletion for other modes.

// GOTCHA #6 — ZERO new imports. runCompletion/loginShellBase use os/fmt/io/path/filepath/strings,
// ALL already imported. completionScript is same-package. `config` is the LOCAL struct (runInit uses
// the `configpkg` ALIAS for internal/config — irrelevant; runCompletion takes the local config value).
// main_test.go also needs ZERO new imports (bytes/strings/testing present; t.Setenv is a method).
// Do NOT run `go mod tidy` (no new deps; it could touch go.sum needlessly).

// GOTCHA #7 — anchor the dispatch insertion by SYMBOL, not the contract's "main.go:482/518" (STALE —
// the file is now 1204 lines). Insert immediately AFTER the `if c.init { return runInit(c, stdout,
// stderr) }` block and immediately BEFORE the `// 5) Normal mode dispatch` comment. Anchor the
// function append by runInit being the LAST function (file tail) — append AFTER it.

// GOTCHA #8 — the "no shell" dispatch test MUST suppress the test runner's own $SHELL. t.Setenv BOTH
// "SKILLDOZER_SHELL"="" AND "SHELL"="". os.Getenv returns "" for unset-or-empty, so t.Setenv(v,"") is
// equivalent to the §13 `env -u SHELL -u SKILLDOZER_SHELL` for loginShellBase's `s == ""` check.

// GOTCHA #9 — completionScript is uncalled until THIS subtask wires runCompletion. Before this task,
// `skilldozer completion` falls through to no-mode (stdout help, exit 0 — because c.completion is set
// but nothing dispatches on it). After this task, the `if c.completion` slot intercepts it. That is the
// SOLE behavior delta for `completion` input. (S1 could land completionScript uncalled because Go
// allows unused package-level functions.)

// GOTCHA #10 — no collision with the parallel sibling P1.M2.T2.S1 (embed). It edits the import block
// (adds _ "embed") + after-var-version (55-61) + before-the-runInit-doc (completionScript @1110).
// This subtask edits the run() dispatch slot (~535, mid-file) + after-runInit tail (~1204). DISJOINT.
// The changesets compose in either order. Do NOT touch completionScript or the embed vars (consumed).

// GOTCHA #11 — the §13 acceptance uses `env -u SHELL -u SKILLDOZER_SHELL ./skilldozer completion`
// for the exit-1 case. In a real shell `env -u` unsets the var; in Go tests t.Setenv(v,"") sets it to
// empty, which is equivalent here (loginShellBase checks `s == ""`). Do NOT try to truly unset in the
// unit test — t.Setenv("","") is the correct, hermetic equivalent.
```

---

## Implementation Blueprint

### Data models and structure

**No new types.** No struct changes (the `completion`/`completionShell` fields are P1.M2.T1.S1's, Complete). The only new "model" is the dispatch branch, expressed inline in `run()`.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — insert the completion dispatch slot in run()
  - FILE: main.go, inside `func run(args []string, stdout, stderr io.Writer) int` (@476)
  - LOCATE the `if c.init { return runInit(c, stdout, stderr) }` block and the immediately-following
    `// 5) Normal mode dispatch (order: path → list → search → check → all → tags).` comment.
  - INSERT between them (GOTCHA #5 — exclusive-mode dispatch mirroring init; GOTCHA #7 — anchor by
    symbol, the contract's line numbers are stale):
      // completion dispatch (PRD §14.6). completion is an exclusive mode (like check/init):
      // exclusivityError above guarantees no other mode is set when c.completion is true, so this
      // self-contained branch returns before the path/list/search/check/all/tags ladder below.
      // runCompletion resolves the shell (--shell → $SKILLDOZER_SHELL → basename($SHELL)), emits the
      // matching embedded script to stdout (exit 0), or exits 1 (undetectable) / 2 (unsupported value)
      // with nothing on stdout (PRD §6.4).
      if c.completion {
          return runCompletion(c, stdout, stderr)
      }
  - This is a ONE-branch insertion. Do NOT touch any other run() branch or the normal-mode ladder.

Task 2: APPEND main.go — loginShellBase (the $SHELL basename helper; guards filepath.Base(""))
  - FILE: main.go (APPEND after runInit, the file tail — currently the file ends with runInit's closing
    brace; GOTCHA #7). PLACE loginShellBase first (detectShell's caller supplies its result).
  - ADD (verbatim from external_deps.md; GOTCHA #2 — the empty guard is load-bearing):
      // loginShellBase returns the lowercased basename of $SHELL — "zsh" for /bin/zsh, "fish" for
      // /usr/bin/fish — or "" if $SHELL is unset/empty. It is the third-tier input to detectShell
      // (PRD §14.6 "basename($SHELL)"). The empty guard is load-bearing: filepath.Base("") returns
      // ".", which would otherwise pollute shell detection (external_deps.md §Shell detection verified
      // this). os.Executable() is NOT used — that is the skilldozer binary path, not the shell.
      // strings.ToLower normalizes e.g. /bin/ZSH → "zsh" so completionScript's case-sensitive switch
      // still matches (explicit --shell / $SKILLDOZER_SHELL values pass through detectShell verbatim).
      func loginShellBase() string {
          s := os.Getenv("SHELL")
          if s == "" {
              return ""
          }
          return strings.ToLower(filepath.Base(s))
      }

Task 3: APPEND main.go — detectShell (the PURE detection core; first non-empty wins)
  - FILE: main.go (APPEND immediately after loginShellBase)
  - ADD (verbatim from external_deps.md; pure, no env mutation; GOTCHA #4 — verbatim passthrough):
      // detectShell resolves the target shell for `skilldozer completion` (PRD §14.6 "Shell detection",
      // first wins): explicit --shell → $SKILLDOZER_SHELL → basename($SHELL). It is a PURE function of
      // its three string args — the caller (runCompletion) supplies the env reads (c.completionShell,
      // os.Getenv("SKILLDOZER_SHELL"), loginShellBase()) — so detection is unit-testable without env
      // mutation. It returns the first non-empty value + true, or ("", false) if all three are empty.
      // Values pass through VERBATIM: only loginShellBase lowercases (it computes the basename), so an
      // explicit `--shell BASH` yields "BASH" → unsupported (completionScript is case-sensitive) → exit 2.
      func detectShell(explicit, envShell, loginShell string) (string, bool) {
          if explicit != "" {
              return explicit, true
          }
          if envShell != "" {
              return envShell, true
          }
          if loginShell != "" {
              return loginShell, true
          }
          return "", false
      }

Task 4: APPEND main.go — runCompletion (the §6.4 exit-code ladder; mirrors runInit's signature)
  - FILE: main.go (APPEND immediately after detectShell — the file's new tail)
  - ADD (GOTCHA #1 — Fprint not Fprintln; GOTCHA #3 — two distinct failure paths; consumes S1's
    completionScript):
      // runCompletion is the `skilldozer completion` handler (PRD §14.6 / §6.4). run()'s dispatch
      // (Task 1) calls it when c.completion is true; completion is exclusive, so no other mode is
      // active. It resolves the shell via detectShell, then emits the matching embedded script to
      // stdout for `eval "$(skilldozer completion)"` (PRD §14.6). Exit codes (PRD §6.4):
      //   - 0 on success (script on stdout);
      //   - 1 if the shell is undetectable (no --shell, no $SKILLDOZER_SHELL, no usable $SHELL);
      //   - 2 if the resolved shell value is not bash/zsh/fish (e.g. tcsh, or $SHELL=/bin/sh → "sh").
      // On the 1/2 paths NOTHING is written to stdout (the §6.4 $(...) contract: `eval` of an empty
      // capture fails cleanly rather than running a partial/garbage script). The script is emitted
      // with Fprint (NOT Fprintln) so the bytes are identical to the embedded file (§14.6 byte-identity).
      func runCompletion(c config, stdout, stderr io.Writer) int {
          shell, ok := detectShell(c.completionShell, os.Getenv("SKILLDOZER_SHELL"), loginShellBase())
          if !ok {
              // Undetectable: all three sources empty (PRD §6.4). Nothing on stdout.
              fmt.Fprintln(stderr, "could not detect shell; pass --shell {bash|zsh|fish}")
              return 1
          }
          script, ok := completionScript(shell)
          if !ok {
              // Detected but unsupported (e.g. tcsh; or $SHELL=/bin/sh → "sh"). Nothing on stdout.
              fmt.Fprintf(stderr, "skilldozer: unsupported shell '%s' (want bash|zsh|fish)\n", shell)
              return 2
          }
          // Success: the matching embedded script, byte-identical to completions/* (PRD §14.6).
          fmt.Fprint(stdout, script)
          return 0
      }

Task 5: EDIT main_test.go — add the 8 tests (mirror run check/init dispatch tests + table tests)
  - FILE: main_test.go (APPEND a new block; package main — white-box. bytes/strings/testing already
    imported; t.Setenv is a testing method. GOTCHA #8 — the no-shell test sets BOTH env vars to "".)
  - (5a) Dispatch: --shell bash → exit 0, _skilldozer_completion (§13 acceptance grep):
      func TestRunCompletionBashScript(t *testing.T) {
          var out, errOut bytes.Buffer
          code := run([]string{"completion", "--shell", "bash"}, &out, &errOut)
          if code != 0 {
              t.Errorf("run(completion --shell bash): code=%d; want 0", code)
          }
          if !strings.Contains(out.String(), "_skilldozer_completion") {
              t.Errorf("stdout missing _skilldozer_completion (§13):\n%s", out.String())
          }
          if errOut.Len() != 0 {
              t.Errorf("stderr=%q; want empty on success", errOut.String())
          }
      }
  - (5b) Dispatch: --shell fish → exit 0, complete -c skilldozer (§13 acceptance grep):
      func TestRunCompletionFishScript(t *testing.T) {
          var out, errOut bytes.Buffer
          code := run([]string{"completion", "--shell", "fish"}, &out, &errOut)
          if code != 0 {
              t.Errorf("run(completion --shell fish): code=%d; want 0", code)
          }
          if !strings.Contains(out.String(), "complete -c skilldozer") {
              t.Errorf("stdout missing 'complete -c skilldozer' (§13):\n%s", out.String())
          }
      }
  - (5c) Dispatch: --shell tcsh → exit 2, EMPTY stdout, stderr mentions tcsh (§13 + §6.4):
      func TestRunCompletionUnsupportedShell(t *testing.T) {
          var out, errOut bytes.Buffer
          code := run([]string{"completion", "--shell", "tcsh"}, &out, &errOut)
          if code != 2 {
              t.Errorf("run(completion --shell tcsh): code=%d; want 2", code)
          }
          if out.Len() != 0 {
              t.Errorf("stdout=%q; want EMPTY on unsupported shell (§6.4)", out.String())
          }
          if !strings.Contains(errOut.String(), "tcsh") {
              t.Errorf("stderr=%q; want it to mention 'tcsh'", errOut.String())
          }
      }
  - (5d) Dispatch: undetectable (no --shell, no $SKILLDOZER_SHELL, no $SHELL) → exit 1, EMPTY stdout,
         stderr mentions "shell" (§13 + §6.4; GOTCHA #8 — set BOTH env vars to ""):
      func TestRunCompletionUndetectableShell(t *testing.T) {
          t.Setenv("SKILLDOZER_SHELL", "")
          t.Setenv("SHELL", "")
          var out, errOut bytes.Buffer
          code := run([]string{"completion"}, &out, &errOut)
          if code != 1 {
              t.Errorf("run(completion, no shell): code=%d; want 1", code)
          }
          if out.Len() != 0 {
              t.Errorf("stdout=%q; want EMPTY on undetectable shell (§6.4)", out.String())
          }
          if !strings.Contains(strings.ToLower(errOut.String()), "shell") {
              t.Errorf("stderr=%q; want it to mention 'shell'", errOut.String())
          }
      }
  - (5e) Detection: $SKILLDOZER_SHELL=zsh honored (envShell wins over loginShell) → exit 0, zsh #compdef:
      func TestRunCompletionEnvShellDetected(t *testing.T) {
          t.Setenv("SKILLDOZER_SHELL", "zsh")
          var out, errOut bytes.Buffer
          code := run([]string{"completion"}, &out, &errOut)
          if code != 0 {
              t.Errorf("run(completion, SKILLDOZER_SHELL=zsh): code=%d; want 0", code)
          }
          if !strings.Contains(out.String(), "#compdef skilldozer") {
              t.Errorf("stdout missing zsh #compdef header:\n%s", out.String())
          }
      }
  - (5f) Detection: basename($SHELL) honored ($SHELL=/bin/zsh → "zsh") → exit 0, zsh #compdef:
      func TestRunCompletionLoginShellDetected(t *testing.T) {
          t.Setenv("SKILLDOZER_SHELL", "")
          t.Setenv("SHELL", "/bin/zsh")
          var out, errOut bytes.Buffer
          code := run([]string{"completion"}, &out, &errOut)
          if code != 0 {
              t.Errorf("run(completion, SHELL=/bin/zsh): code=%d; want 0", code)
          }
          if !strings.Contains(out.String(), "#compdef skilldozer") {
              t.Errorf("stdout missing zsh #compdef header (basename(/bin/zsh)=zsh):\n%s", out.String())
          }
      }
  - (5g) detectShell unit test (PURE — no run(), no env; table from test_patterns.md):
      func TestDetectShell(t *testing.T) {
          cases := []struct{ explicit, env, login, wantShell string; wantOK bool }{
              {"bash", "", "", "bash", true},   // explicit wins
              {"", "fish", "", "fish", true},   // env wins
              {"", "", "zsh", "zsh", true},     // login wins
              {"", "", "", "", false},          // all empty → false
              {"bash", "fish", "zsh", "bash", true}, // explicit beats env+login
          }
          for _, tc := range cases {
              got, ok := detectShell(tc.explicit, tc.env, tc.login)
              if got != tc.wantShell || ok != tc.wantOK {
                  t.Errorf("detectShell(%q,%q,%q) = (%q,%v); want (%q,%v)",
                      tc.explicit, tc.env, tc.login, got, ok, tc.wantShell, tc.wantOK)
              }
          }
      }
  - (5h) loginShellBase unit test (t.Setenv; covers basename + lowercase + the empty guard):
      func TestLoginShellBase(t *testing.T) {
          cases := []struct{ shell, want string }{
              {"/bin/zsh", "zsh"},
              {"/usr/bin/fish", "fish"},
              {"/bin/ZSH", "zsh"},   // lowercased
              {"", ""},              // empty guard (filepath.Base("") would be ".")
          }
          for _, tc := range cases {
              t.Setenv("SHELL", tc.shell)
              if got := loginShellBase(); got != tc.want {
                  t.Errorf("loginShellBase() with SHELL=%q = %q; want %q", tc.shell, got, tc.want)
              }
          }
      }

Task 6: VERIFY (isolated, then whole-module + invariants)
  - gofmt -l main.go main_test.go     # MUST print nothing (run gofmt -w if it lists a file)
  - go vet ./...                      # exit 0
  - go build ./...                    # exit 0
  - go test -run 'Completion|DetectShell|LoginShellBase' -v ./...   # the 8 new tests pass
  - go test ./...                     # whole module green (the if c.completion slot is the only
                                      #   behavior delta for `completion` input; no other test affected)
  - git diff --quiet go.mod go.sum && echo deps unchanged   # GOTCHA #6
```

### Implementation Patterns & Key Details

```go
// loginShellBase — $SHELL basename, empty-guarded, lowercased (external_deps.md verified).
func loginShellBase() string {
	s := os.Getenv("SHELL")
	if s == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(s))
}

// detectShell — pure first-non-empty selector (no env mutation). Verbatim passthrough.
func detectShell(explicit, envShell, loginShell string) (string, bool) {
	if explicit != "" {
		return explicit, true
	}
	if envShell != "" {
		return envShell, true
	}
	if loginShell != "" {
		return loginShell, true
	}
	return "", false
}

// runCompletion — the §6.4 exit-code ladder. Fprint (not Fprintln) for byte-identity.
func runCompletion(c config, stdout, stderr io.Writer) int {
	shell, ok := detectShell(c.completionShell, os.Getenv("SKILLDOZER_SHELL"), loginShellBase())
	if !ok {
		fmt.Fprintln(stderr, "could not detect shell; pass --shell {bash|zsh|fish}")
		return 1
	}
	script, ok := completionScript(shell)
	if !ok {
		fmt.Fprintf(stderr, "skilldozer: unsupported shell '%s' (want bash|zsh|fish)\n", shell)
		return 2
	}
	fmt.Fprint(stdout, script)
	return 0
}

// The dispatch slot in run() — mirrors init's exclusive-mode dispatch exactly:
	if c.completion {
		return runCompletion(c, stdout, stderr)
	}
```

Notes easy to get wrong:
- `fmt.Fprint(stdout, script)`, NOT `Fprintln` — the embedded script already ends in `\n`; Fprintln would add a second newline (GOTCHA #1).
- `loginShellBase` checks `s == ""` BEFORE `filepath.Base` — `filepath.Base("")` is `"."` (GOTCHA #2).
- The two failure paths are distinct: all-empty ⇒ exit 1; a detected-but-unsupported value (tcsh, or `$SHELL=/bin/sh`→"sh") ⇒ exit 2 (GOTCHA #3). Do not collapse them.
- The "no shell" test sets BOTH `SKILLDOZER_SHELL=""` and `SHELL=""` (the runner's `$SHELL` would otherwise leak; GOTCHA #8).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **`detectShell` is pure (3 string args); `loginShellBase` reads `$SHELL`.** external_deps.md §'Shell detection' prescribes this split so detection is unit-testable without env mutation (the established chooseStore/resolveStore testability pattern). The caller (`runCompletion`) supplies the two env reads + `c.completionShell`.
2. **`loginShellBase` lowercases; `detectShell` passes explicit/envShell verbatim.** `$SHELL` basenames can vary in case (`/bin/ZSH`); lowercasing normalizes them so the case-sensitive `completionScript` switch matches. Explicit `--shell` / `$SKILLDOZER_SHELL` values pass through as typed — the user owns their correctness (the contract fixes `detectShell` as a verbatim selector; the test_patterns unit tests use lowercase inputs).
3. **`runCompletion` mirrors `runInit`'s `(c config, stdout, stderr io.Writer) int` signature.** Both are exclusive-subcommand handlers dispatched identically from `run()`. No new type.
4. **`Fprint`, not `Fprintln`, for the script.** §14.6 mandates byte-identity with the on-disk/embedded file (which ends in `\n`). `Fprintln` would violate it. Acceptance only greps, but byte-correctness is the contract.
5. **Two distinct exit codes (1 vs 2) per §6.4.** Undetectable (all three shell sources empty) ⇒ exit 1; an unsupported value ⇒ exit 2. A detected `sh`/`tcsh` is exit 2, not exit 1 — only the all-empty case is exit 1.
6. **Dispatch slot AFTER `if c.init`, BEFORE the normal-mode ladder.** Mirrors init's exclusive-mode dispatch; `exclusivityError` (P1.M2.T1.S1, landed) already guarantees `c.completion` is standalone, so `runCompletion` needs no internal conflict guard.
7. **Zero new imports.** All needed stdlib packages (`os`/`fmt`/`io`/`path/filepath`/`strings`) are already imported; `completionScript` is same-package; `config` is the local struct. `go.mod`/`go.sum` unchanged.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. Pure stdlib (os.Getenv/filepath.Base/strings.ToLower/fmt.Fprint) +
    same-package completionScript (S1). No new module. (GOTCHA #6)

CONSUMERS (NOT built in this subtask — listed to fix the interface):
  - The §13 acceptance suite (P1.M4 / the implementer's final gate) runs the four completion
    assertions against the real binary: `completion --shell bash | grep _skilldozer_completion`,
    `completion --shell fish | grep 'complete -c skilldozer'`, the `env -u SHELL -u SKILLDOZER_SHELL
    … exit 1` case, and `--shell tcsh … exit 2`. This subtask's runCompletion makes all four pass.
  - P1.M3.T1.S1 (completions lockstep) edits completions/* + rebuilds; the embedded bytes change but
    completionScript keeps returning them and the grep substrings still match. This subtask is the
    consumer of the (stable) completionScript API, not the bytes.

CONSUMED (already present — verified):
  - completionScript(shell string) (string, bool) — main.go:1110 (P1.M2.T2.S1).
  - config.completion bool / config.completionShell string — local struct (P1.M2.T1.S1, Complete).
  - exclusivityError's completion handling — main.go (P1.M2.T1.S1, Complete) guarantees c.completion
    is standalone when runCompletion is called.

NO ROUTES / NO DATABASE / NO CONFIG-FORMAT CHANGE / NO NEW FILES / NO completions/* EDIT.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after the main.go edits)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l main.go main_test.go   # must print NOTHING (run gofmt -w if it lists a file)
go vet ./...                    # expect exit 0
go build ./...                  # expect exit 0
# Expected: zero output / exit 0.
```

### Level 2: Unit Tests (the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test ./... -run 'Completion|DetectShell|LoginShellBase' -v
# Expected: ALL 8 pass. The load-bearing assertions:
#   TestRunCompletionBashScript          -> code 0; stdout Contains _skilldozer_completion; stderr empty.
#   TestRunCompletionFishScript          -> code 0; stdout Contains 'complete -c skilldozer'.
#   TestRunCompletionUnsupportedShell    -> code 2; stdout EMPTY; stderr mentions 'tcsh'.
#   TestRunCompletionUndetectableShell   -> code 1; stdout EMPTY; stderr Contains 'shell'.
#   TestRunCompletionEnvShellDetected    -> code 0; stdout Contains '#compdef skilldozer' (SKILLDOZER_SHELL=zsh).
#   TestRunCompletionLoginShellDetected  -> code 0; stdout Contains '#compdef skilldozer' (SHELL=/bin/zsh).
#   TestDetectShell                      -> explicit>env>login; all-empty→false (5 table cases).
#   TestLoginShellBase                   -> basename + lowercase + empty-guard (4 table cases).

# Regression — the whole suite stays green (the if c.completion slot is the only behavior delta;
# today `completion` falls through to no-mode, so no existing test asserted on it):
go test ./...   # expect exit 0
```

### Level 3: Whole-module regression + invariants + the §13 acceptance (completion slice)

```bash
cd /home/dustin/projects/skilldozer

go build -o /tmp/skilldozer-comp . ; echo "build exit $?"   # 0
go vet  ./... ; echo "vet exit $?"                          # 0
go test ./... ; echo "test exit $?"                         # 0

# GOTCHA #6 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"

# The four §13 completion acceptance assertions, against the freshly built binary:
./go-build-comp() { go build -o /tmp/skilldozer-comp . ; } ; go-build-comp
S=/tmp/skilldozer-comp
$S completion --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo "completion-bash OK" || echo FAIL
$S completion --shell fish 2>/dev/null | grep -q 'complete -c skilldozer' && echo "completion-fish OK" || echo FAIL
out=$(env -u SHELL -u SKILLDOZER_SHELL $S completion 2>/tmp/cerr); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && grep -qi 'shell' /tmp/cerr && echo "completion-no-shell OK" || echo FAIL
$S completion --shell tcsh >/dev/null 2>&1; [ "$?" = "2" ] && echo "completion-bad-shell OK" || echo FAIL
# Expected: all four "OK". (These mirror §13 verbatim; the unit tests already cover them, this is
# the end-to-end binary confirmation.)
rm -f /tmp/skilldozer-comp
```

### Level 4: Detection-order + stream-separation spot-checks

```bash
cd /home/dustin/projects/skilldozer

go build -o /tmp/skilldozer-comp .
S=/tmp/skilldozer-comp

# $SKILLDOZER_SHELL beats $SHELL (detectShell: envShell before loginShell):
SKILLDOZER_SHELL=zsh SHELL=/bin/bash $S completion 2>/dev/null | grep -q '#compdef skilldozer' \
  && echo "env-beats-login OK" || echo FAIL

# Success writes NOTHING to stderr (script is stdout-only):
err=$($S completion --shell bash 2>&1 >/dev/null); [ -z "$err" ] && echo "success-stderr-empty OK" || echo FAIL

# Undetectable writes NOTHING to stdout (§6.4 $(...) contract):
out=$(env -u SHELL -u SKILLDOZER_SHELL $S completion 2>/dev/null); [ -z "$out" ] && echo "undetectable-stdout-empty OK" || echo FAIL

rm -f /tmp/skilldozer-comp
# Expected: all OK. (These are the §6.4 stream-separation guarantees; the unit tests assert them too.)
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` clean; `go vet ./...` exit 0; `go build ./...` exit 0
- [ ] Level 2 PASS — the 8 new tests pass (6 dispatch + detectShell table + loginShellBase table)
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (zero regressions); the 4 §13 completion assertions print OK; `git diff go.mod go.sum` → "deps unchanged"
- [ ] Level 4 PASS — env-beats-login, success-stderr-empty, undetectable-stdout-empty all OK

### Feature Validation
- [ ] `run()` has `if c.completion { return runCompletion(c, stdout, stderr) }` after `if c.init`, before the normal-mode ladder
- [ ] `run(["completion","--shell","bash"])` ⇒ exit 0, stdout Contains `_skilldozer_completion`, stderr empty
- [ ] `run(["completion","--shell","fish"])` ⇒ exit 0, stdout Contains `complete -c skilldozer`
- [ ] `run(["completion","--shell","tcsh"])` ⇒ exit 2, stdout EMPTY, stderr mentions "tcsh"
- [ ] `run(["completion"])` with no shell env ⇒ exit 1, stdout EMPTY, stderr Contains "shell"
- [ ] `$SKILLDOZER_SHELL=zsh` and `$SHELL=/bin/zsh` both ⇒ exit 0, stdout Contains `#compdef skilldozer`
- [ ] `detectShell` table passes (explicit>env>login; all-empty→false)
- [ ] `loginShellBase` table passes (basename + lowercase + empty-guard)

### Code Quality / Convention Validation
- [ ] Dispatch slot mirrors init's exclusive-mode dispatch exactly
- [ ] `runCompletion` mirrors `runInit`'s `(c config, stdout, stderr io.Writer) int` signature
- [ ] `detectShell` is pure (no `os.*` inside it); `loginShellBase` confines the `$SHELL` read
- [ ] `fmt.Fprint` (not `Fprintln`) for the script (§14.6 byte-identity)
- [ ] Doc comments cite PRD §14.6, §6.4, and external_deps.md §Shell detection
- [ ] Anti-patterns avoided (see below)
- [ ] No new dependency; stdlib only; go.mod/go.sum byte-for-byte identical

### Scope Discipline
- [ ] Did NOT modify `completionScript` or the embed vars (P1.M2.T2.S1 — consumed)
- [ ] Did NOT modify `completions/*` (P1.M3.T1.S1) or the README (P1.M3.T1.S2)
- [ ] Did NOT modify `exclusivityError` or parseArgs (P1.M2.T1.S1 — Complete, consumed)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't use `fmt.Fprintln` for the script.** Use `fmt.Fprint` — the embedded script already ends in `\n`; Fprintln violates §14.6 byte-identity. (GOTCHA #1.)
- ❌ **Don't drop the `loginShellBase` empty guard.** `filepath.Base("")` returns `"."`; you MUST check `os.Getenv("SHELL") == ""` first or an unset `$SHELL` feeds a bogus `"."` into detection. (GOTCHA #2.)
- ❌ **Don't conflate the two failure paths.** All-empty ⇒ exit 1; a detected-but-unsupported value (tcsh, or `$SHELL=/bin/sh`→"sh") ⇒ exit 2. Only the all-empty case is exit 1. (GOTCHA #3.)
- ❌ **Don't add lowercasing to `detectShell`.** It returns values verbatim; only `loginShellBase` lowercases (it computes the basename). The contract + test_patterns fix `detectShell` as a verbatim selector. (GOTCHA #4.)
- ❌ **Don't add a conflict guard inside `runCompletion`.** `exclusivityError` (P1.M2.T1.S1, landed) already guarantees `c.completion` is standalone. The dispatch slot `return`s unconditionally. (GOTCHA #5.)
- ❌ **Don't add imports.** `os`/`fmt`/`io`/`path/filepath`/`strings` are all already imported; `completionScript` is same-package; `config` is the local struct. `go mod tidy` is a no-op (no new deps) — don't run it. (GOTCHA #6.)
- ❌ **Don't anchor by the contract's "main.go:482/518" line numbers.** They're stale (the file is 1204 lines). Anchor by symbol: after the `if c.init` block / before the `// 5) Normal mode dispatch` comment; append after `runInit` (the file tail). (GOTCHA #7.)
- ❌ **Don't forget to suppress `$SHELL` in the no-shell test.** Set BOTH `SKILLDOZER_SHELL=""` and `SHELL=""` via `t.Setenv`, or the runner's own `$SHELL` leaks into `loginShellBase`. (GOTCHA #8.)
- ❌ **Don't modify `completionScript`, the embed vars, `completions/*`, `exclusivityError`, `parseArgs`, or the README.** All are sibling subtasks' scope (S1 / P1.M3.T1.S1 / P1.M2.T1.S1 / P1.M3.T1.S2) — consumed, not modified.

---

## Confidence Score

**9.5/10** — one-pass implementation success likelihood. The change is purely additive (1 dispatch line + 3 appended functions + 8 tests) and consumes three already-present, verified symbols: `completionScript(shell) (string, bool)` (main.go:1110, S1 — confirmed landed), the local `config.completion`/`completionShell` fields (P1.M2.T1.S1 — Complete), and `exclusivityError`'s completion handling (already rejects completion+other-modes). The `detectShell`/`loginShellBase` designs are fixed VERBATIM by external_deps.md §'Shell detection' (which verified the `filepath.Base` idiom + the empty-guard gotcha). The exit-code ladder is fixed by PRD §6.4/§14.6 + the contract. The three grep substrings the tests/§13 rely on are confirmed present in the embedded files. Zero new imports (all stdlib packages already imported). The 0.5 reservation is for the two subtleties most likely to be mishandled — `Fprint` vs `Fprintln` (GOTCHA #1) and the exit-1-vs-exit-2 distinction (GOTCHA #3) — both explicitly locked by dedicated unit tests (TestRunCompletionUnsupportedShell asserts exit 2 + empty stdout; TestRunCompletionUndetectableShell asserts exit 1 + empty stdout), and the stale line-anchor risk (GOTCHA #7), mitigated by anchoring instructions to symbols.
