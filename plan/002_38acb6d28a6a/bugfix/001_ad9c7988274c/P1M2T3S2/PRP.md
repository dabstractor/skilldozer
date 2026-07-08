# PRP — P1.M2.T3.S2: Wire `expandHome` into `resolveStore` before `filepath.Abs` + integration test (Issue 5)

> **Subtask:** the WIRING half of Issue 5. S1 (P1.M2.T3.S1) already shipped the pure-ish helper `func expandHome(p string) string` (main.go:894) + its 2 table-driven unit tests. This subtask does the two things S1 deliberately deferred (see S1's GOTCHA #5): **(a)** call `expandHome` in `resolveStore` on the unified `store` local — the single line `store = expandHome(store)` — between the `chooseStore` call and the `filepath.Abs(store)` call; **(b)** an end-to-end integration test through `run()` proving `init --store ~/sub` expands `~` to `$HOME` before `filepath.Abs` (config holds `$HOME/sub`, that dir is created, stdout is `$HOME/sub`).
>
> **Why ONE line fixes every form:** `init <dir>` (parseArgs:300), `--store <dir>` (parseArgs:272), `--store=<dir>` (parseArgs:199), and the interactive typed prompt ALL converge into `c.initStore` → `resolveStore(c.initStore)` (runInit main.go:1027) → `chooseStore` returns it verbatim (main.go:862/882) → the single `store` local at the insertion point. So one `store = expandHome(store)` normalizes every source. Idempotent on already-absolute/tilde-free paths, so the existing absolute-store integration test stays green.
>
> **STATUS (verified at PRP-write time):** `grep -n 'func expandHome' main.go` → `894` (S1 landed/in-flight, as the parallel-context requires — treat S1's PRP as a CONTRACT). resolveStore body @901-925; edit site `chooseStore(...)`@941 + `filepath.Abs(store)`@945 read exactly. `stdinIsTerminal()`@808 is a NON-overridable plain func (doc @796-807: "the test seam is chooseStore's isTTY PARAMETER, not a global override") → the interactive typed-path is NOT reachable through `run()` without refactoring that seam (out of scope); the `--store` path transitively proves the fix for all forms (research/verified_facts.md §2/§4). `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`@2573 is the integration-test pattern to MIRROR (same setup the contract LOGIC §3 prescribes: SKILLDOZER_CONFIG / SKILLDOZER_SKILLS_DIR="" / t.Chdir(t.TempDir()), + HOME set). `configpkg.File.Store`@46 confirmed; main_test.go imports bytes/os/filepath/strings/testing/configpkg already present → NO import edit; go.mod/go.sum UNCHANGED. The contract cites main.go:865/886 (STALE — current 901/945); this PRP anchors by unique TEXT.

---

## Goal

**Feature Goal**: Make `resolveStore` expand a leading `~`/`~/` to `$HOME` (via the already-landed `expandHome`, Issue 5) BEFORE `filepath.Abs`, so `init ~/x`, `--store ~/x`, `--store=~/x`, and the interactive prompt all resolve to `$HOME/x` instead of the buggy `<cwd>/~/x` (which also created a directory literally named `~`). Done with ONE production line + a Mode-A doc-comment note, and locked by ONE end-to-end integration test through `run()`.

**Deliverable**: Two edits to `main.go` + one new test in `main_test.go`:
1. `main.go` resolveStore — insert `store = expandHome(store)` (+ a 5-line `// Issue 5` comment) between the `chooseStore(...)` block and `abs, err := filepath.Abs(store)` (current main.go:941-945). Anchor by the unique text `abs, err := filepath.Abs(store)` preceded by the `chooseStore` return-handling block.
2. `main.go` resolveStore doc comment (@887 region) — Mode-A note that a leading `~`/`~/` is expanded to `$HOME` (expandHome, Issue 5) before `filepath.Abs` (because `filepath.Abs` does not expand `~`).
3. `main_test.go` — `TestRunInitStoreTildeExpandsHome`: `run([]string{"init","--store","~/sub"})` with `HOME` set to a DISTINCT temp dir (so `home != cwd`) → exit 0, `config.Store == filepath.Join(home,"sub")`, that dir created, `stdout == filepath.Join(home,"sub")+"\n"`. Mirrors `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`@2573.

**Success Definition**: `go build/vet/test ./...` all green; `gofmt -l main.go main_test.go` empty; `go.mod`/`go.sum` byte-for-byte unchanged (no new imports); the new test PASSES and FAILS on the un-wired code (it asserts `config.Store == $HOME/sub`, but `filepath.Abs("~/sub")` without expandHome yields `<cwd>/~/sub`, and `home != cwd` so the equality breaks); S1's `TestExpandHome`/`TestExpandHomeNoHomeUnchanged` + the existing `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (the absolute, tilde-free sibling — the idempotency regression guard) stay green.

---

## User Persona (if applicable)

**Target User**: A user who runs `skilldozer init --store ~/myskills` (or types `~/myskills` at the interactive prompt, or `skilldozer init ~/myskills`), and any tooling/acceptance suite asserting shell-like home expansion. Also: scripts / users who legitimately pass an absolute or relative path — those must be UNCHANGED (expandHome is a no-op for them).

**Use Case**: `skilldozer init --store ~/myskills` → the user expects the config to record `$HOME/myskills` and that dir to be created, exactly as every shell expands `~`.

**User Journey**: (today) `init --store ~/myskills` → config gets `store: <cwd>/~/myskills`; a real dir `~` is created under cwd (broken, surprising, and the dir named `~` is awkward to `rm`) → (after S1+S2) `resolveStore` normalizes `~/myskills` to `$HOME/myskills` BEFORE `filepath.Abs`, so the config records the correct absolute home path and the right dir is created. **S1 shipped the normalizer; S2 wires it into the one place all init paths pass through.**

**Pain Points Addressed**: the literal-`~` directory bug (architecture/bug_fixes_validation.md §ISSUE 5); the inconsistency between `init`'s path handling and the `DefaultStore` home expansion already in the binary; the surprise that `~` works in the shell but not `skilldozer init`.

---

## Why

- **Closes architecture/bug_fixes_validation.md §ISSUE 5** (Minor): tilde is not expanded in `init`'s path input. The root cause is `filepath.Abs` does NOT expand `~` (it only joins a relative path onto cwd, so `~/x` wrongly becomes `<cwd>/~/x`). The prescribed fix is to expand a leading `~/` and bare `~` to `$HOME` BEFORE `filepath.Abs`. S1 added the helper; S2 is the wiring + the integration test that proves the bug is gone end-to-end.
- **The fix is structurally one line** because `resolveStore` is the single chokepoint: all four init path sources (`init <dir>`, `--store <dir>`, `--store=<dir>`, interactive prompt) converge into the `store` local right before `filepath.Abs`. Wiring `expandHome` there fixes all four at once — there is no per-source branch to update (research/verified_facts.md §2).
- **Idempotent and fail-safe, so it cannot regress the common case.** `expandHome` is a no-op for already-absolute / relative-tilde-free paths (the existing absolute-store integration test stays green) and returns the input unchanged when `$HOME` is unset (it never crashes or emits `""`, which would make `filepath.Abs("")` yield cwd). The no-`$HOME` fail-safe is locked at the UNIT level by S1's `TestExpandHomeNoHomeUnchanged`; no integration test for it (it would assert preserved-buggy behavior — creating a dir named `~` — which is low-value and awkward).
- **Splits the work cleanly:** S1 shipped a directly-unit-testable helper + its unit tests (no I/O, no TTY). S2 ships the one-line wiring + the end-to-end integration test. This PRP is S2 only.

---

## What

A one-line wiring change + a 5-line comment in `resolveStore`, a Mode-A doc-comment note, and one new `TestRunInit*` integration test. No signature changes, no config-format changes, no dispatch changes, no parseArgs changes (all three flag forms already set `c.initStore`), no README change (Mode B sweep = P1.M3.T1), no new helper (S1 owns `expandHome`).

### Success Criteria

- [ ] `store = expandHome(store)` is the line immediately before `abs, err := filepath.Abs(store)` in `resolveStore` (main.go), with a comment citing Issue 5 and the before-`filepath.Abs` ordering.
- [ ] The resolveStore doc comment (@887 region) notes a leading `~`/`~/` is expanded to `$HOME` (expandHome, Issue 5) before `filepath.Abs`, and why (filepath.Abs does not expand `~`).
- [ ] `TestRunInitStoreTildeExpandsHome` exists in `main_test.go` after `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (before `TestRunBareTagUnconfiguredNeverPrompts`), asserting exit 0 + `config.Store == filepath.Join(home,"sub")` + that dir created + `stdout == filepath.Join(home,"sub")+"\n"`, with `home` a DISTINCT temp dir from cwd.
- [ ] The new test FAILS on the un-wired code (remove the `store = expandHome(store)` line → the assertion breaks because `filepath.Abs("~/sub")` = `<cwd>/~/sub` and `home != cwd`). This is the "cannot pass on the buggy code" discipline (mirrors main_test.go:2606's exact-equality-not-Contains guard for Issue 1).
- [ ] `go test ./...` green (incl. S1's `TestExpandHome*` and the existing absolute-store sibling); `gofmt -l main.go main_test.go` empty; `go.mod`/`go.sum` unchanged; no new imports in either file.

---

## All Needed Context

### Context Completeness Check

**Pass.** The wiring edit site is pinned by the unique two-block text `chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)` … `abs, err := filepath.Abs(store)` (the only such adjacency in main.go). The doc-comment edit is pinned by the unique phrase `choice ABSOLUTIZED via filepath.Abs`. The helper being called (`expandHome`, main.go:894) is confirmed present (S1's deliverable) — `grep -n 'func expandHome' main.go` → 894. The integration test is a 1:1 mirror of `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (main_test.go:2573-2628) — read in full; the ONLY deltas are `t.Setenv("HOME", home)` + a tilde-bearing `store`. `configpkg.File.Store` (config.go:46) is the assertion target. Every import the test needs (bytes/os/path/filepath/strings/testing/configpkg) is grep-confirmed already present in main_test.go (lines 3-11) → zero import edits, go.mod/go.sum byte-identical. The test's discrimination power is PROVEN not assumed: `home` (TempDir A) != `cwd` (TempDir C), so `filepath.Abs("~/sub")` without expandHome = `C/~/sub` ≠ `A/sub` = `want` — the equality assertion cannot pass on the bug. The interactive-path gap is documented and closed by transitivity (one source-agnostic line; research/verified_facts.md §2/§4). An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative wiring + the verbatim call-site change
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/go_tilde_expansion.md
  why: "THE research brief. §3 = filepath.Abs does NOT expand ~ ⇒ expansion MUST run before it
        (the S2 ordering). §'Call site change in resolveStore' gives the EXACT before/after:
          before:  abs, err := filepath.Abs(store)
          after:   store = expandHome(store); abs, err := filepath.Abs(store)
        S1 already pasted the helper from this file; S2 pastes the call-site change from it."
  critical: "The call-site change is a ONE-LINE insert (store = expandHome(store)) immediately
             before the filepath.Abs(store) line — NOT a refactor of resolveStore, NOT a move of
             the logic into chooseStore. Place it AFTER chooseStore returns (so the prompt's typed
             answer is also normalized) and BEFORE filepath.Abs."

# MUST READ — the authoritative bug writeup + repro (Issue 5)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "§ISSUE 5 is the repro: a tilde-bearing init path ⇒ store: <cwd>/~/myskills, a real dir
        named ~ is created. Pins the root cause (filepath.Abs does not expand ~) and the fix
        (expand a leading ~/ and bare ~ to $HOME before filepath.Abs). The S2 integration test
        asserts exactly the 'after' state of that repro."
  section: "ISSUE 5 (Minor)."

# MUST READ — the S1 PRP (the CONTRACT for the helper S2 consumes; do NOT duplicate its work)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M2T3S1/PRP.md
  why: "Defines expandHome's EXACT behavior, signature (string)->string, placement (main.go:894),
        doc comment, and its 2 unit tests. S2 ASSUMES this is implemented verbatim. S1's GOTCHA #5
        explicitly DEFERS the resolveStore wiring + integration test to S2 — this PRP. Read it to
        know the helper's contract (no-op on absolute/tilde-free; unchanged on $HOME-unset) so the
        S2 test's expectations match."

# MUST READ — the file under edit (the resolveStore region; anchor the insert by TEXT)
- file: main.go
  why: "THE edit target. expandHome @894 (S1 — already present; do NOT redefine). chooseStore @858.
        resolveStore doc @887 (the Mode-A doc edit target; phrase 'choice ABSOLUTIZED via
        filepath.Abs'). resolveStore body @901; the wiring site is the block
          store, err := chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)
          if err != nil { return \"\", err }
          abs, err := filepath.Abs(store)            <- INSERT store = expandHome(store) BEFORE this
        (current main.go:941-945; the contract's 865/886 are STALE). stdinIsTerminal @808 is a
        plain func (NOT overridable) — see research/verified_facts.md §4 for why the interactive
        path isn't separately integration-tested. runInit @1011 calls resolveStore(c.initStore)
        @1027 — the only caller. Import block: os/strings/path/filepath ALREADY present; NO import edit."
  pattern: "Call the package-local helper on the unified store local, one line, lowercase func,
            same package — exactly how resolveStore already calls chooseStore/configpkg.DefaultStore.
            No new abstraction; no error to handle (expandHome returns string, fail-safe)."

# MUST READ — the test file under edit (MIRROR the integration-test pattern exactly)
- file: main_test.go
  why: "THE other edit target + the integration-test-pattern source. TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0
        @2573-2628 is the 1:1 mirror: same setup (SKILLDOZER_CONFIG -> temp config.yaml;
        SKILLDOZER_SKILLS_DIR=\"\"; t.Chdir(t.TempDir())), run(init --store <store>), then assert
        exit 0 + configpkg.Load(cfg).Store + os.Stat(store).IsDir + out.String()==store+\"\\n\".
        S2 copies that shape and swaps the store to ~/sub + adds t.Setenv(\"HOME\", home) with
        home a DISTINCT temp dir. INSERT the new test AFTER TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0
        and BEFORE TestRunBareTagUnconfiguredNeverPrompts @2630 (anchor by the unique comment
        '// TestRunBareTagUnconfiguredNeverPrompts — the load-bearing'). Import block lines 3-11:
        bytes/os/path/filepath/strings/testing/configpkg ALL present; NO import edit."
  gotcha: "Do NOT call t.Parallel() — the test mutates HOME + SKILLDOZER_* via t.Setenv (non-parallel-safe;
           same rule as the S1 HOME tests and the existing TestRunInitStore* tests). home MUST be a
           SEPARATE t.TempDir() from the t.Chdir(t.TempDir()) cwd, or the test loses discrimination
           (if home==cwd, filepath.Abs('~/sub')=<cwd>/~/sub could coincidentally equal home/sub)."

# READ-ONLY — the config assertion target (Store field) + the DefaultStore os.UserHomeDir precedent
- file: internal/config/config.go
  why: "File.Store @46 (`Store string \\`yaml:\"store,omitempty\"\\``) — the field the S2 test reads
        via configpkg.Load(cfg). DefaultStore @150 uses os.UserHomeDir (the precedent expandHome
        reuses); with HOME set it does NOT error, so resolveStore's def-resolution @929 won't fail
        even though haveStore != \"\" makes def unused."
  section: "File.Store; DefaultStore."

# READ-ONLY — the parallel-sibling boundary (disjoint regions; land in either order with S1)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M2T3S1/PRP.md
  why: "S1 edits main.go @894 (a NEW func) + main_test.go ~2428 (2 NEW unit tests after the
        TestChooseStore* family). S2 edits main.go resolveStore body ~941-945 + doc ~887 (a NEW
        line + comment) + main_test.go ~2628 (1 NEW integration test after TestRunInitStoreWrites*).
        DISJOINT regions in both files; no text overlap; land in either order."

# READ-ONLY — PRD (the §8.2 init path authority + §17 stdlib-only + the bugfix PRD h3.4 Issue 5)
- file: PRD.md
  why: "READ-ONLY. §8.2 (init path forms — the tilde-bearing input source; 'absolute store path').
        §17 (stdlib-only besides yaml.v3). The bugfix PRD h3.4 Issue 5 is the repro."
  section: "§8.2, §17 (and the bugfix PRD h3.4 Issue 5)."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/tasks.json
  why: "P1.M2.T3.S2's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. This PRP
        transcribes it; tasks.json wins on any conflict."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && grep -n 'func expandHome\|func resolveStore\|abs, err := filepath.Abs\|func runInit\|func TestRunInitStoreWrites' main.go main_test.go
main.go:894:func expandHome(p string) string {          # S1 — ALREADY PRESENT (do NOT redefine)
main.go:901:func resolveStore(haveStore string) (string, error) {
main.go:945:	abs, err := filepath.Abs(store)            # <- INSERT store = expandHome(store) BEFORE this
main.go:1011:func runInit(c config, stdout, stderr io.Writer) int {
main.go:1027:	store, err := resolveStore(c.initStore)     # the ONLY caller
main_test.go:2573:func TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0(t *testing.T) {  # MIRROR this
$ grep -c 'store = expandHome' main.go        # 0  — the wiring is genuinely new (this subtask)
$ go cat go.mod | head -1                      # module github.com/dabstractor/skilldozer (go 1.25, yaml.v3 sole dep)
```

### Desired Codebase tree with files to be changed

```bash
main.go        # EDIT resolveStore: insert `store = expandHome(store)` (+ 5-line Issue-5 comment) before
               #        `abs, err := filepath.Abs(store)` (current 945). EDIT the resolveStore doc comment
               #        (@887): note ~ is expanded to $HOME before filepath.Abs (Mode A). NO import edit.
main_test.go   # ADD TestRunInitStoreTildeExpandsHome after TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0
               #        (2573-2628), before TestRunBareTagUnconfiguredNeverPrompts (2630). NO import edit.
# go.mod / go.sum — UNCHANGED (expandHome already defined by S1; the wiring uses only existing locals/helper).
# expandHome (main.go:894) — UNCHANGED (S1 owns it; S2 only CALLS it).
# chooseStore / parseArgs / setupStore / runInit — UNCHANGED (the wiring is inside resolveStore only).
```

| File | Change | Owner |
|---|---|---|
| `main.go` | resolveStore: `store = expandHome(store)` before `filepath.Abs(store)` + Mode-A doc note. No import edit. | Issue 5 contract + go_tilde_expansion.md §"Call site change" |
| `main_test.go` | NEW `TestRunInitStoreTildeExpandsHome` (mirrors `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`). No import edit. | QA Issue 5 |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 (CRITICAL — the #1 one-pass stall) — place expandHome AFTER chooseStore returns and
// BEFORE filepath.Abs, NOT earlier. chooseStore returns the verbatim haveStore / typed answer
// (main.go:862/882); the ~ in that string must be expanded on the `store` local it returns INTO,
// right before filepath.Abs consumes it. Putting it inside chooseStore would break its pure/unit-
// testable contract (S1 placed the helper OUTSIDE chooseStore for exactly this reason). Putting it
// AFTER filepath.Abs is useless (filepath.Abs already mangled ~/x into <cwd>/~/x). The site is the
// gap between the two — one line. (go_tilde_expansion.md §3.)

// GOTCHA #2 — the test MUST set HOME to a DISTINCT temp dir from the t.Chdir cwd. If home == cwd,
// filepath.Abs("~/sub") (the un-wired, buggy result) = <cwd>/~/sub could coincidentally equal
// home/sub, and the test would pass on the bug (no discrimination). Use TWO t.TempDir() calls:
//   home := t.TempDir(); t.Setenv("HOME", home); ... t.Chdir(t.TempDir())  // cwd != home
// (research/verified_facts.md §3.) Mirrors the Issue-1 exact-equality-not-Contains discipline
// (main_test.go:2606): the assertion must FAIL on the buggy code, not merely pass on the fixed code.

// GOTCHA #3 — do NOT add an integration test for the interactive typed-path. stdinIsTerminal()
// (main.go:808) is a PLAIN FUNCTION, not a package var (its doc @796-807 says so: "the test seam
// is chooseStore's isTTY PARAMETER, not a global override"). resolveStore passes stdinIsTerminal()
// directly (main.go:941) and reads os.Stdin via bufio (main.go:933) — neither is injectable
// through run(). Driving the interactive path requires refactoring that seam into a var, which is
// OUT OF SCOPE (S2 = wire + test). The --store path transitively proves the fix for ALL forms
// because the wiring is one source-agnostic line on the unified `store` local (research/verified_facts.md §4).

// GOTCHA #4 — do NOT redefine or move expandHome. S1 (P1.M2.T3.S1) already defined it at
// main.go:894 with its unit tests. S2 only CALLS it. `grep -c 'func expandHome' main.go` must stay
// 1 (it is 1 today). Redefining it collides with S1.

// GOTCHA #5 — anchor by TEXT, not line number. The contract cites main.go:865/886 (resolveStore /
// filepath.Abs); the CURRENT lines are 901/945 (M1 + P1.M2.T2 shifted them). Match the unique text
// `abs, err := filepath.Abs(store)` (preceded by the chooseStore return-handling block) for the
// wiring, and `choice ABSOLUTIZED via filepath.Abs` for the doc comment. Same for the test insert:
// anchor by the func/comment names, not numbers.

// GOTCHA #6 — no deps/imports change in EITHER file. expandHome is already defined (S1) using
// os/strings/path/filepath (already imported in main.go); the wiring line uses only the existing
// `store` local and the same-package helper. main_test.go already imports bytes/os/path/filepath/
// strings/testing/configpkg. go.mod/go.sum must be byte-for-byte identical: `git diff --quiet
// go.mod go.sum` must hold. (Same invariant as S1's GOTCHA #8.)

// GOTCHA #7 — the test MUST NOT call t.Parallel(): it mutates HOME + SKILLDOZER_CONFIG +
// SKILLDOZER_SKILLS_DIR via t.Setenv (non-parallel-safe). The existing TestRunInitStore* tests
// and S1's TestExpandHome* tests carry the same no-Parallel rule.

// GOTCHA #8 — stdout in runInit is the EFFECTIVE store from skillsdir.Find() (main.go:1063), NOT
// the `store` local directly. Find() reads back the just-written config (setupStore wrote
// Store=expanded-path synchronously at main.go:1071-ish via configpkg.Save), so dir == the expanded
// path. The existing TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 proves Find() sees the
// fresh config (it asserts out.String()==store+"\n" and passes). So asserting stdout ==
// filepath.Join(home,"sub")+"\n" is correct and is the strongest end-to-end check.
```

---

## Implementation Blueprint

### Data models and structure

**No data-model changes.** No new types, fields, methods, or signatures. The change calls an existing helper on an existing local. `configpkg.File.Store` (config.go:46) is read by the test, not modified.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — wire expandHome into resolveStore (the ONE production line)
  - FILE: main.go, function resolveStore (body @901-925; the contract's "main.go:865" is STALE)
  - ANCHOR (by unique TEXT per GOTCHA #5): the block
        store, err := chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)
        if err != nil {
            return "", err
        }
        abs, err := filepath.Abs(store)
    (current main.go:941-945). This adjacency is unique in main.go.
  - REPLACE the `abs, err := filepath.Abs(store)` line (keeping the chooseStore block above it) with
    the Issue-5 comment + the wiring line + the unchanged filepath.Abs line:
        // Expand a leading "~"/"~/" to $HOME BEFORE filepath.Abs (Issue 5). filepath.Abs
        // does not expand "~", so without this `init ~/x`, `--store ~/x`, `--store=~/x`,
        // and the interactive typed path would all store "<cwd>/~/x" and create a directory
        // literally named "~". expandHome is a no-op for absolute/tilde-free paths and
        // returns the input unchanged when $HOME is unset (fail safe).
        store = expandHome(store)
        abs, err := filepath.Abs(store)
  - This is the ENTIRE production logic change. Verify: the line sits AFTER chooseStore returns
    and BEFORE filepath.Abs (GOTCHA #1); expandHome is CALLED not redefined (GOTCHA #4); no error
    to handle (expandHome returns string, fail-safe).

Task 2: EDIT main.go — Mode-A doc-comment note on resolveStore (the contract DOCS requirement)
  - FILE: main.go, the resolveStore doc comment (@887 region; anchor by the unique phrase
    "choice ABSOLUTIZED via filepath.Abs" per GOTCHA #5)
  - REPLACE:
        // prompt reader over os.Stdin/os.Stderr (readPrompt) — and returns chooseStore's
        // choice ABSOLUTIZED via filepath.Abs (PRD §8.2 "absolute store path"). The ONE
    WITH:
        // prompt reader over os.Stdin/os.Stderr (readPrompt) — and returns chooseStore's
        // choice with a leading "~"/"~/" expanded to $HOME (expandHome, Issue 5) and then
        // ABSOLUTIZED via filepath.Abs (PRD §8.2 "absolute store path"). filepath.Abs alone
        // does not expand "~", so expandHome runs first — otherwise `init ~/x`, `--store ~/x`,
        // `--store=~/x`, and the interactive prompt path would all store "<cwd>/~/x". The ONE
  - This is the per-subtask Mode-A doc edit (decisions.md §D7). The README sweep is the final
    Mode-B task (P1.M3.T1) — NOT this subtask.

Task 3: EDIT main_test.go — add TestRunInitStoreTildeExpandsHome (the integration test)
  - FILE: main_test.go; INSERT after TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0
    (main_test.go:2573-2628) and BEFORE the TestRunBareTagUnconfiguredNeverPrompts comment
    (main_test.go:2630). Anchor by the unique comment text
    "// TestRunBareTagUnconfiguredNeverPrompts — the load-bearing prompt-safety guarantee" per
    GOTCHA #5. (Groups with the other TestRunInit* integration tests; DISJOINT from S1's
    TestExpandHome* insert at ~2428.)
  - ADD (mirrors TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0@2573 1:1; GOTCHA #2
    distinct home vs cwd; GOTCHA #7 no t.Parallel):
    // TestRunInitStoreTildeExpandsHome — Issue 5 (P1.M2.T3.S2): `init --store ~/sub` (and
    // `init ~/sub` / `--store=~/sub` / the interactive prompt) must expand a leading "~" to
    // $HOME before filepath.Abs. Without the expandHome wiring in resolveStore, filepath.Abs
    // joins "~/sub" onto cwd → "<cwd>/~/sub" and a directory literally named "~" is created.
    // With the wiring, config.Store == $HOME/sub, that dir is created, and stdout (the
    // effective resolved store) == $HOME/sub. Mirrors TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0
    // (the absolute, tilde-free sibling) — same setup (SKILLDOZER_CONFIG / SKILLDOZER_SKILLS_DIR=""
    // / t.Chdir), plus HOME set to a DISTINCT temp dir so home != cwd and the assertion
    // discriminates (fails on the un-wired code). The wiring is one source-agnostic line in
    // resolveStore, so this `--store` path transitively proves the fix for `init <dir>` and
    // the interactive prompt too (stdinIsTerminal is a non-overridable plain func; see S2 PRP).
    func TestRunInitStoreTildeExpandsHome(t *testing.T) {
        // Do NOT call t.Parallel() — mutates HOME / SKILLDOZER_* env.
        home := t.TempDir()
        t.Setenv("HOME", home)                    // expandHome + configpkg.DefaultStore read $HOME
        cfg := filepath.Join(t.TempDir(), "config.yaml")
        t.Setenv("SKILLDOZER_CONFIG", cfg)        // redirect the config write to a temp file
        t.Setenv("SKILLDOZER_SKILLS_DIR", "")     // ensure the config rule wins (env unset)
        t.Chdir(t.TempDir())                      // cwd != home: without expandHome, store would be <cwd>/~/sub

        var out, errOut bytes.Buffer
        code := run([]string{"init", "--store", "~/sub"}, &out, &errOut)
        if code != 0 {
            t.Fatalf("run(init --store ~/sub): code=%d; want 0; stderr=%q", code, errOut.String())
        }

        want := filepath.Join(home, "sub") // $HOME/sub, NOT "~/sub" and NOT <cwd>/~/sub

        // The store dir was CREATED (setupStore's MkdirAll on the EXPANDED path).
        if info, err := os.Stat(want); err != nil || !info.IsDir() {
            t.Errorf("expanded store %q not created: stat err=%v (did ~ get expanded?)", want, err)
        }

        // The config holds the EXPANDED absolute store (resolveStore expandHome→filepath.Abs→config.Save).
        f, err := configpkg.Load(cfg)
        if err != nil {
            t.Fatalf("config.Load: %v", err)
        }
        if f.Store != want {
            t.Errorf("config.Store=%q; want %q (~ NOT expanded before filepath.Abs)", f.Store, want)
        }

        // §6.1: stdout carries EXACTLY one line — the EFFECTIVE resolved store ($HOME/sub).
        // skillsdir.Find() reads back the just-written config, so dir == want. On the buggy
        // code stdout would be "<cwd>/~/sub\n", failing this equality.
        if got := out.String(); got != want+"\n" {
            t.Errorf("init stdout=%q; want exactly %q", got, want+"\n")
        }
    }

Task 4: VERIFY in isolation + whole module + scope/disjointness invariants
  - gofmt -l main.go main_test.go     # MUST print nothing (run gofmt -w if it lists a file)
  - go vet ./...                      # exit 0
  - go build ./...                    # exit 0
  - go test -run 'TestRunInitStoreTildeExpandsHome' -v ./...   # the new integration test passes
  - go test -run 'TestExpandHome$|TestExpandHomeNoHomeUnchanged|TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0' -v ./...
                                      # S1's helper tests + the absolute-store sibling (idempotency) stay green
  - go test ./...                     # whole module green; zero regressions
  - git diff --quiet go.mod go.sum && echo deps unchanged   # GOTCHA #6
  - SCOPE-INVARIANT (GOTCHA #4): expandHome is defined exactly once (S1's; S2 only calls it):
      test "$(grep -c 'func expandHome' main.go)" -eq 1 && echo "OK: expandHome defined once"
  - DISCRIMINATION-INVARIANT (GOTCHA #2): the new test FAILS on the un-wired code. Temporarily
    comment out `store = expandHome(store)`, re-run the test → it MUST fail (config.Store ==
    <cwd>/~/sub != $HOME/sub); then RESTORE the line. (Or: `git stash` the main.go wiring edit
    only, run the test, see it fail, `git stash pop`.)
```

### Implementation Patterns & Key Details

```go
// The wiring (Task 1) — ONE line in resolveStore, AFTER chooseStore, BEFORE filepath.Abs.
// expandHome (S1, main.go:894) is CALLED here, not redefined. No error to handle (string->string,
// fail-safe). Idempotent on absolute/tilde-free paths; unchanged when $HOME unset.
//
//	resolveStore(haveStore) (string, error) {
//	    ...
//	    store, err := chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)
//	    if err != nil {
//	        return "", err
//	    }
//	    store = expandHome(store)          // <- THE FIX (Issue 5): expand ~ before filepath.Abs
//	    abs, err := filepath.Abs(store)    // filepath.Abs does NOT expand ~; that's why order matters
//	    if err != nil {
//	        return "", fmt.Errorf("skilldozer init: absolutize store: %w", err)
//	    }
//	    return abs, nil
//	}
//
// Why one line fixes every form: haveStore (from c.initStore) and the interactive typed answer
// BOTH become the `store` local via chooseStore's return (main.go:862 for haveStore, 882 for the
// typed answer). expandHome runs on that local regardless of source.

// The integration test (Task 3) — mirrors TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0
// (main_test.go:2573). The ONLY deltas from that sibling: (1) t.Setenv("HOME", home) with home a
// DISTINCT temp dir; (2) store input "~/sub" instead of an absolute path; (3) assert the EXPANDED
// filepath.Join(home,"sub") rather than the verbatim input. Discrimination is structural: home
// (TempDir A) != cwd (TempDir C), so the un-wired filepath.Abs("~/sub")=<cwd>/~/sub != A/sub=want.
```

Notes easy to get wrong:
- The wiring line goes AFTER the `chooseStore` error-handling block, not inside it (GOTCHA #1).
- The test's `home` and cwd MUST be different `t.TempDir()` calls (GOTCHA #2) or the test loses discrimination.
- stdout is asserted against the EXPANDED path, and that's correct because runInit prints `skillsdir.Find()` which reads the freshly-written config (GOTCHA #8).
- No interactive-path integration test — it's structurally untestable through `run()` and transitively covered (GOTCHA #3).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **One integration test (the `--store` path), not four.** The wiring is a single source-agnostic line on the unified `store` local in resolveStore; `init <dir>`, `--store <dir>`, `--store=<dir>`, and the interactive prompt all converge there (research/verified_facts.md §2). Testing the `--store` path end-to-end proves the fix for all forms; testing all four would duplicate parseArgs-routing coverage (already in `TestParseArgsInitPositionalDir`/`TestParseArgsInitStoreLongForm`/`TestParseArgsInitStoreEqualsForm`). The contract says "integration test" (singular).
2. **No interactive-path integration test.** `stdinIsTerminal()` (main.go:808) is a non-overridable plain func; `resolveStore` wires `os.Stdin` + `stdinIsTerminal()` directly (no injection seam). Driving the typed path through `run()` needs a seam refactor that is out of scope (S2 = wire + test). The `--store` path transitively proves it (one line). The interactive typed-path's UNIT behavior is locked by S1's `TestExpandHome*` + `TestChooseStoreTTYTypedPathOverrides` (main_test.go:2395). (research/verified_facts.md §4.)
3. **Assert `config.Store` AND `stdout` AND dir-created, not just one.** `config.Store` proves the expanded path reached `configpkg.Save`; `stdout` proves it propagated to the user-visible §6.1 output (via `skillsdir.Find()`); dir-created proves `setupStore`'s `MkdirAll` ran on the expanded path. Three independent observations of the same fix — the strongest end-to-end lock, mirroring the triple-assert in `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`.
4. **No no-`$HOME` integration test.** With `HOME` unset, expandHome returns the input unchanged (fail-safe), so `filepath.Abs("~/sub")` = `<cwd>/~/sub` — the preserved-buggy behavior. Asserting that creates a dir named `~` and locks the bug's preservation, which is low-value and awkward. The fail-safe is locked at the UNIT level by S1's `TestExpandHomeNoHomeUnchanged`.
5. **Idempotency regression guard = the existing absolute-store sibling.** `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (main_test.go:2573) uses an absolute, tilde-free store and asserts `config.Store == store` unchanged. With expandHome wired (a no-op for absolute paths) it stays green — that IS the idempotency proof. No new idempotency test needed.
6. **No README change here.** The contract DOCS assigns the README sweep to the final Mode B task (P1.M3.T1). This subtask's doc edit is the in-code resolveStore comment only (Mode A, Task 2). (decisions.md §D7.)

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. No new imports in either file (expandHome is already defined by S1;
    the wiring uses the existing `store` local + same-package helper; main_test.go already imports
    bytes/os/path/filepath/strings/testing/configpkg). (GOTCHA #6.)

CALLER (UNCHANGED):
  - runInit (main.go:1027) is the ONLY caller of resolveStore: `store, err := resolveStore(c.initStore)`.
    No caller change — the wiring is INSIDE resolveStore. (GOTCHA #1.)

HELPER (UNCHANGED — S1 owns it):
  - expandHome (main.go:894) is CALLED by the new wiring line; it is NOT redefined or moved.
    `grep -c 'func expandHome' main.go` stays 1. (GOTCHA #4.)

SURFACE:
  - No new exported symbols. expandHome stays package-local (lowercase), as S1 defined it.

DOCUMENTATION (Mode A only here):
  - The resolveStore doc comment (Task 2) is the per-subtask Mode A doc edit. The README init
    section is swept by the final Mode B task (P1.M3.T1) — no doc file rides here beyond the
    in-code comment. (decisions.md §D7.)

PARALLEL SIBLING (no conflict):
  - S1 (P1.M2.T3.S1) edits main.go @894 (the NEW expandHome func) + main_test.go ~2428 (2 NEW unit
    tests after the TestChooseStore* family). S2 edits main.go resolveStore body ~941-945 + doc ~887
    + main_test.go ~2628 (1 NEW integration test after TestRunInitStoreWrites*). DISJOINT regions in
    both files; no text-level overlap; land in either order.

NO ROUTES / NO DATABASE / NO CONFIG-FORMAT CHANGE / NO PARSEARGS CHANGE / NO NEW FILES.
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

### Level 2: Unit / Integration Tests (the core gate)

```bash
cd /home/dustin/projects/skilldozer

# The new integration test (the deliverable):
go test -run 'TestRunInitStoreTildeExpandsHome' -v ./...
# Expected: PASS. Asserts: exit 0; config.Store == $HOME/sub (filepath.Join(home,"sub"));
# that dir created (os.Stat IsDir); stdout == $HOME/sub+"\n". FAILS on the un-wired code
# (see Level 3 discrimination check).

# S1's helper tests + the absolute-store sibling (idempotency regression) MUST stay green:
go test -run 'TestExpandHome$|TestExpandHomeNoHomeUnchanged|TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0' -v ./...
# Expected: ALL PASS. The sibling proves expandHome is a no-op for absolute/tilde-free paths.

# The whole init / resolveStore family stays green:
go test -run 'TestRunInit|TestChooseStore|TestSetupStore|TestParseArgsInit' -v ./...
# Expected: PASS (no behavior changed beyond ~ expansion; purely additive wiring).
```

### Level 3: Whole-module regression + scope/disjointness/discrimination invariants

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # 0
go vet  ./...  ; echo "vet exit $?"     # 0
go test ./...  ; echo "test exit $?"    # 0  — CRITICAL: zero regressions (one additive line + one test)

# GOTCHA #6 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"

# SCOPE invariant (GOTCHA #4): expandHome is defined exactly ONCE (S1's; S2 only calls it).
test "$(grep -c 'func expandHome' main.go)" -eq 1 && echo "OK: expandHome defined once" \
  || echo "FAIL: expandHome redefined/moved"

# WIRING invariant: the new line is present exactly once, before filepath.Abs in resolveStore.
grep -c 'store = expandHome(store)' main.go   # expect 1

# DISCRIMINATION invariant (GOTCHA #2): the new test FAILS on the un-wired code (proves it actually
# tests the fix). Comment out the wiring line, re-run, expect FAILURE; then restore.
git stash push main.go   # stash ONLY the main.go wiring+doc edits (keep main_test.go)
go test -run 'TestRunInitStoreTildeExpandsHome' ./... ; echo "un-wired exit=$? (want NON-zero/FAIL)"
git stash pop
go test -run 'TestRunInitStoreTildeExpandsHome' ./... ; echo "wired exit=$? (want 0/PASS)"
# Expected: un-wired run FAILS (config.Store=<cwd>/~/sub != $HOME/sub); wired run PASSES.

# Disjointness invariant (S1 edits expandHome @894 + tests ~2428; S2 edits resolveStore body + test ~2628):
grep -n 'func expandHome\|store = expandHome' main.go          # 894 (S1) + ~945 (S2 wiring) — distinct
grep -n 'func TestExpandHome\|func TestRunInitStoreTildeExpandsHome' main_test.go   # distinct funcs
# Expected: "deps unchanged"; "expandHome defined once"; wiring present once; un-wired FAILS, wired PASSES.
```

### Level 4: Behavioral spot-check (the contract OUTPUT, by hand)

```bash
cd /home/dustin/projects/skilldozer
go build -o /tmp/skilldozer .

# Contract OUTPUT §4: `skilldozer init --store ~/sub` (HOME set) writes config store = $HOME/sub
# and CREATES that dir.
export HOME="$(mktemp -d)"
export SKILLDOZER_CONFIG="$HOME/cfg.yaml"
export SKILLDOZER_SKILLS_DIR=""
cd "$(mktemp -d)"                       # cwd != HOME (escape repo walk-up; deterministic)
/tmp/skilldozer init --store ~/sub ; echo "exit=$?"   # expect 0
test -d "$HOME/sub" && echo "OK: \$HOME/sub created" || echo "FAIL: dir not created"
grep -c "^store: $HOME/sub\$" "$HOME/cfg.yaml"   # expect 1 (config holds the EXPANDED path)
unset HOME SKILLDOZER_CONFIG SKILLDOZER_SKILLS_DIR

# Idempotency spot-check: an ABSOLUTE store is unchanged (expandHome no-op).
export HOME="$(mktemp -d)"; export SKILLDOZER_CONFIG="$HOME/cfg.yaml"; export SKILLDOZER_SKILLS_DIR=""
ABS="$(mktemp -d)/absstore"
/tmp/skilldozer init --store "$ABS" >/dev/null 2>&1 ; echo "exit=$?"   # 0
grep -c "^store: $ABS\$" "$HOME/cfg.yaml"   # expect 1 (absolute path verbatim, NOT mangled)
unset HOME SKILLDOZER_CONFIG SKILLDOZER_SKILLS_DIR
# Expected: both spot-checks pass. (The unit/integration tests are authoritative; this is a
# human-readable confirmation of the contract OUTPUT.)
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l main.go main_test.go` empty; `go vet ./...` exit 0; `go build` exit 0
- [ ] Level 2 PASS — `TestRunInitStoreTildeExpandsHome` passes; S1's `TestExpandHome*` + the absolute-store sibling `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` stay green
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (zero regressions); `git diff go.mod go.sum` → "deps unchanged"; expandHome defined once; the new test FAILS on the un-wired code (discrimination)
- [ ] Level 4 PASS — hand-run `init --store ~/sub` (HOME set) creates `$HOME/sub` + config `store: $HOME/sub`; absolute store unchanged

### Feature Validation
- [ ] `store = expandHome(store)` is the line immediately before `abs, err := filepath.Abs(store)` in resolveStore
- [ ] The resolveStore doc comment notes `~`/`~/` is expanded to `$HOME` (expandHome, Issue 5) before `filepath.Abs`, and why
- [ ] `TestRunInitStoreTildeExpandsHome`: `run(init --store ~/sub)` with HOME set → exit 0, `config.Store == filepath.Join(home,"sub")`, that dir created, `stdout == filepath.Join(home,"sub")+"\n"`
- [ ] The new test FAILS if the wiring line is removed (cannot pass on the buggy code)
- [ ] Idempotency holds: an absolute / tilde-free store is unchanged (the existing sibling test stays green; Level 4 spot-check)

### Code Quality / Convention Validation
- [ ] The wiring mirrors how resolveStore already calls its helpers (one line, same package, no new abstraction)
- [ ] `expandHome` is CALLED, not redefined/moved (S1 owns it)
- [ ] The test mirrors `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` exactly (same setup shape; same `configpkg.Load` + `os.Stat` + exact-equality-stdout assertions)
- [ ] The test does NOT call `t.Parallel()` (HOME / SKILLDOZER_* mutation)
- [ ] Tests placed after `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (the TestRunInit* group); DISJOINT from S1's TestExpandHome* insert
- [ ] No new imports; no new deps; `go.mod`/`go.sum` byte-for-byte identical
- [ ] No new files; both edits to `main.go` + one test in `main_test.go`

### Scope Discipline
- [ ] Did NOT place expandHome inside chooseStore or after filepath.Abs (placed it in the gap between them — GOTCHA #1)
- [ ] Did NOT make `home` == cwd in the test (used two distinct `t.TempDir()` — GOTCHA #2)
- [ ] Did NOT add an interactive-path integration test (structurally untestable through run(); transitively covered — GOTCHA #3)
- [ ] Did NOT redefine/move expandHome (S1 owns it; S2 only calls it — GOTCHA #4)
- [ ] Did NOT anchor by line number (used unique text — GOTCHA #5)
- [ ] Did NOT add deps/imports or a README change (Mode B = P1.M3.T1 — GOTCHA #6)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't place the expandHome call inside `chooseStore` or after `filepath.Abs`.** Inside chooseStore breaks its pure/unit-testable contract (S1 kept the helper OUTSIDE for this reason). After filepath.Abs is useless (filepath.Abs already mangled `~/x` into `<cwd>/~/x`). The site is the gap between chooseStore's return and filepath.Abs's call — one line. (GOTCHA #1.)
- ❌ **Don't write an integration test where `home == cwd`.** If they coincide, the un-wired `filepath.Abs("~/sub")` = `<cwd>/~/sub` could equal `home/sub`, and the test passes on the bug (no discrimination). Use two distinct `t.TempDir()` calls. The assertion must FAIL on the buggy code, not merely pass on the fixed code (mirrors main_test.go:2606's Issue-1 discipline). (GOTCHA #2.)
- ❌ **Don't add an interactive-path integration test.** `stdinIsTerminal()` is a non-overridable plain func and `resolveStore` wires `os.Stdin` directly — the typed path isn't reachable through `run()` without refactoring the seam (out of scope). The `--store` path transitively proves the fix (one source-agnostic line). (GOTCHA #3.)
- ❌ **Don't redefine or move `expandHome`.** S1 defined it at main.go:894. S2 only CALLS it. `grep -c 'func expandHome' main.go` must stay 1. (GOTCHA #4.)
- ❌ **Don't anchor by line number.** The contract's main.go:865/886 are stale (current 901/945). Match unique text (`abs, err := filepath.Abs(store)` + the preceding chooseStore block; `choice ABSOLUTIZED via filepath.Abs`). (GOTCHA #5.)
- ❌ **Don't add deps/imports, a README change, or test all four flag forms separately.** expandHome is already defined (S1) using already-imported stdlib; the README sweep is Mode B (P1.M3.T1); one `--store` integration test transitively covers all forms (the wiring is source-agnostic). (GOTCHA #6, DESIGN DECISIONS #1/#6.)
- ❌ **Don't call `t.Parallel()` in the test.** It mutates HOME + SKILLDOZER_* via t.Setenv (non-parallel-safe). (GOTCHA #7.)

---

## Confidence Score

**9.5/10** — This is a ONE-LINE production wiring (`store = expandHome(store)` before `filepath.Abs`) + a Mode-A doc note + one integration test. The helper it calls (`expandHome`, main.go:894) is already defined and unit-tested by S1 (confirmed via `grep -n 'func expandHome' main.go` → 894). The edit site is pinned by unique text (`abs, err := filepath.Abs(store)` preceded by the chooseStore return-handling block); the doc edit by `choice ABSOLUTIZED via filepath.Abs`. The integration test is a 1:1 mirror of the green `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (main_test.go:2573, read in full) with only two deltas (`t.Setenv("HOME", home)` + a `~/sub` store) — every import it needs is already present, so go.mod/go.sum stay byte-identical. The test's discrimination is PROVEN, not assumed: `home` (TempDir A) != `cwd` (TempDir C), so the un-wired `filepath.Abs("~/sub")` = `C/~/sub` ≠ `A/sub` = `want` — the assertion fails on the bug (Level 3 checks this directly). The interactive-path gap is documented and closed by transitivity (one source-agnostic line; the typed-path unit behavior is locked by S1 + `TestChooseStoreTTYTypedPathOverrides`). The 0.5 reservation is for the single most-likely one-pass stall — placing the `expandHome` call in the wrong spot (inside chooseStore, or after filepath.Abs) instead of the one-line gap between them — which GOTCHA #1 + the verbatim Task 1 oldText/newText + the Level 3 `grep -c 'store = expandHome(store)'`==1 invariant jointly guard against.
