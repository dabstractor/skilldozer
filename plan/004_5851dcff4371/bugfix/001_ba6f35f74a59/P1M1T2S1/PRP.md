# PRP — P1.M1.T2.S1: Distinguish vanished-store from unconfigured in `findConfig` and propagate through `Find` (Issue 2)

> **Subtask:** P1.M1.T2.S1 — the sole subtask of P1.M1.T2 (Issue 2: a configured store whose directory has vanished silently falls through to a *different* store). Makes `findConfig` report a vanished `store:` dir as a distinct condition, and `Find` return a wrapped sentinel error instead of silently picking up an unrelated sibling/walk-up store.
> **Scope boundary:** Edits ONLY `internal/skillsdir/skillsdir.go` (add `"fmt"` import, the `ErrConfiguredStoreMissing` sentinel, the `findConfig` 4th return value, the `Find` vanished-check, two doc-comment updates) and `internal/skillsdir/skillsdir_test.go` (6 call-site `, _` appends, 1 rename+semantic change, 2 new Find()-level tests). Does NOT touch `main.go` (callers already handle `err != nil` generically), does NOT touch completions (Issue 1 = T1.S3, parallel, fish file), does NOT touch PRD.md/README (the error message IS the user-facing doc).

---

## Goal

**Feature Goal**: When the config file is present and names a `store:` directory that does not exist on disk, `skillsdir.Find()` returns a distinct, wrapped sentinel error (`ErrConfiguredStoreMissing`) naming the configured path and the fix — instead of silently falling through to the sibling/walk-up rules and resolving tags from an *unrelated* store. This enforces PRD §6.4 ("if configured but the dir vanished, the concise reason + fix … exit 1") so `pi --skill "$(skilldozer myskill)"` fails loudly rather than passing a wrong-path.

**Deliverable**: Edits to two existing files (no new files):
1. `internal/skillsdir/skillsdir.go` — add `"fmt"` to imports; add `var ErrConfiguredStoreMissing = errors.New(...)` near `ErrNotFound`; widen `findConfig`'s signature with a 4th return `vanishedStore string` (populated only on the vanished-store branch at line 126; `""` on the other 4 paths); add the vanished-check to `Find()` (4-value capture → `if ok` → `if vanished != "" { return wrapped ErrConfiguredStoreMissing }`); update the `findConfig` and `Find` doc comments.
2. `internal/skillsdir/skillsdir_test.go` — append `, _` (or `, vanished`) to the 6 `findConfig()` call sites; rename `TestFindConfigStoreDirAbsent` → `TestFindConfigStoreVanished` and assert `found==false && vanished==configuredPath`; add `TestFindErrorsOnVanishedConfiguredStore` and `TestFindEnvOverridesVanishedStore`.

**Success Definition**: `go test ./internal/skillsdir/...` passes; `go build ./...` + `go vet ./...` clean; `go.mod`/`go.sum` unchanged; `findConfig` has the 4-value signature; `Find()` returns a wrapped `ErrConfiguredStoreMissing` (verifiable via `errors.Is`) on a vanished configured store; `TestFindAllMissReturnsErrNotFound` stays GREEN (the ghost-config path bails at line 113, never reaching 126); `main.go` is byte-identical.

---

## User Persona (if applicable)

**Target User**: any user whose configured store directory was deleted/moved (e.g. an external drive unmounted, a rename, a fresh checkout without the data dir). Today they get *silently wrong* skills (resolved from an unrelated sibling/walk-up store) with exit 0; after this fix they get a one-line reason + fix on stderr, exit 1, nothing on stdout.

**Pain Points Addressed**: the §6.4 reliability contract — `pi --skill "$(skilldozer x)"` must fail loudly, not pass a garbage/wrong path. Today a vanished store defeats that: it silently loads a different store's skills.

---

## Why

- **PRD §6.4 explicitly distinguishes the two cases**: "skilldozer is unconfigured ⇒ `run \`skilldozer --init\`` **(or, if configured but the dir vanished, the concise reason + fix)**, exit 1." Today both collapse — the vanished case silently falls through. This subtask splits them.
- **PRD §8.1** (the genuine fall-through): "A missing or unreadable config is treated as 'not yet configured' and falls through to §8.3 rules 3-5 — never a hard error." That stays true for a *missing config file* (lines 109/113/116); only the *present-config-but-vanished-store* case (line 126) becomes a loud error.
- **Closing the silent-wrong-store failure mode**: today `findConfig` returns `("", 0, false)` for a vanished store, so `Find` tries sibling/walk-up and may return a *different* store with no warning — exactly the surprise §6.4 exists to prevent.
- **decisions.md §D1-§D3** fixed the design: 4-value `findConfig` return (D1, not an error-return that breaks the "locked per-rule shape"); sentinel + `%w` wrap (D2, so `errors.Is` works); env still wins (D3, findEnv runs first).

---

## What

A minimal, surgical change confined to the skillsdir package:

1. **Imports**: add `"fmt"` (needed for the new `fmt.Errorf("%w: ...")`; not imported today).
2. **Sentinel**: `var ErrConfiguredStoreMissing = errors.New("configured skills store directory does not exist")`, placed right after `ErrNotFound` (line 275).
3. **`findConfig` signature** (line 106): add a 4th return `vanishedStore string`. The 5 return paths become: 3 fall-throughs `("", 0, false, "")`; the vanished branch (126) `("", 0, false, store)`; the hit (128) `(store, SourceConfig, true, "")`.
4. **`Find()` vanished-check** (lines 292-294): capture the 4th value; `if ok { return ...nil }`; `if vanished != "" { return "", 0, fmt.Errorf("%w: configured store %q does not exist; run \`skilldozer --init\` or recreate the directory", ErrConfiguredStoreMissing, vanished) }` — BEFORE the sibling/walk-up calls.
5. **Doc comments**: `findConfig` (the "Returns …" sentence) and `Find` (add the vanished-store error path) — Mode A.
6. **Tests**: 6 call-site `, _` appends; `TestFindConfigStoreDirAbsent` → `TestFindConfigStoreVanished` (semantic assertion); 2 new Find()-level tests.

### Success Criteria

- [ ] `findConfig` signature is `(dir string, src Source, found bool, vanishedStore string)`; the vanished branch (126) returns `store`; the other 4 paths return `""`
- [ ] `ErrConfiguredStoreMissing` sentinel exists, wrapped via `fmt.Errorf("%w: ...")` in `Find()`
- [ ] `Find()` returns a wrapped `ErrConfiguredStoreMissing` (assertable via `errors.Is`) on a present-config + vanished-store, BEFORE sibling/walk-up
- [ ] A missing/unreadable/no-store-key config (lines 109/113/116) STILL falls through (no error) — §8.1 unchanged
- [ ] `findEnv` (priority 1) still wins over a vanished config — env-set+valid ⇒ no error (TestFindEnvOverridesVanishedStore)
- [ ] `TestFindAllMissReturnsErrNotFound` stays GREEN (ghost config bails at line 113)
- [ ] `"fmt"` added to skillsdir.go imports; `main.go` byte-identical; `go.mod`/`go.sum` unchanged
- [ ] `go test ./internal/skillsdir/...` + `go build ./...` + `go vet ./...` all pass

---

## All Needed Context

### Context Completeness Check

**Pass.** Every edit site is pinned to its exact current line with before→after text in `research/verified_facts.md` §1-§6; the `fmt`-import prerequisite (§3), the main.go-no-change proof (§4), the `TestFindAllMissReturnsErrNotFound` safety proof (§7), and the design decisions (§8) are all documented. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the verified site inventory (exact current lines + before/after + the fmt-import prerequisite)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M1T2S1/research/verified_facts.md
  why: "THE source of truth. §1 = findConfig's 5 return paths + the per-path 4th value. §2 = Find()+ErrNotFound exact code. §3 = CRITICAL: fmt is NOT imported today; must be ADDED. §4 = main.go callers handle err generically (NO main.go change). §5 = the 6 test call sites. §6 = the renamed test's new assertion. §7 = why TestFindAllMissReturnsErrNotFound stays green. §8 = decisions D1-D3."
  critical: "§3 (add fmt import) and §7 (the ghost-config safety proof) are the two facts that prevent the most likely errors: a build failure on undefined fmt, and a wrongly-'fixed' all-miss test."

# MUST READ — the design decisions (the 4-value shape, the sentinel, env-still-wins)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/decisions.md
  why: "§D1 chose the 4-value return over side-channel/re-query/error-return (findConfig STILL 'never errors' — it returns a string; Find constructs the error). §D2 chose sentinel + %w wrap (errors.Is works). §D3: env runs first so it still overrides the vanished config. These pin the exact shape — do not deviate."
  section: "D1, D2, D3 (Issue 2)."

# MUST READ — the architecture research (the §6.4 distinction + the all-miss safety analysis)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/skillsdir_research.md
  why: "Maps every findConfig return path; proves unsetEnvVar points SKILLDOZER_CONFIG at a ghost file (so TestFindAllMissReturnsErrNotFound bails at line 113, not 126 — stays green); identifies TestFindConfigStoreDirAbsent (587) as the ONE test the sentinel changes. Confirms line 126 is the 'vanished store' seam."
  section: "§3 (return-path table), §4 (unsetEnvVar + all-miss safety)."

# MUST READ — the authoritative bug writeup
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/issue_analysis.md
  why: "§Issue 2 is the authoritative repro (the --path example showing silent sibling-store use) + the exact fix prescription (distinguish vanished-store from unconfigured; sentinel + reason/fix; keep genuine fall-through for missing config file). PRD §6.4 + §8.1 are the authority."
  section: "Issue 2 (Major)."

# MUST READ — the edit target (read the findConfig + Find regions before editing)
- file: internal/skillsdir/skillsdir.go
  why: "THE edit target. findConfig doc 92-105 + body 106-128 (5 return paths; store var at 118-123; os.Stat at 124; THE BUG at 126). ErrNotFound + sentinel zone 272-275. Find doc 277-287 + body 288-302 (findConfig call at 292). Imports: needs fmt ADDED (see §3)."
  pattern: "Existing per-rule shape: every findXxx returns (dir, src, found bool) and never errors. findConfig's 4th value is the ONE sanctioned deviation (decisions D1) — it stays non-erroring; Find constructs the error. Sentinel sits beside ErrNotFound (same package-level pattern)."
  gotcha: "fmt is NOT in the import block today — ADD it or go build fails. The 'store' var (118-123) is the value returned as vanishedStore at 126 — it is already computed before os.Stat."

- file: internal/skillsdir/skillsdir_test.go
  why: "THE other edit target. 6 findConfig() call sites (558/573/581/589/600/614) need a 4th value. TestFindConfigStoreDirAbsent (587) → rename + semantic change. TestFindAllMissReturnsErrNotFound (513) STAYS (do not touch — it stays green). The writeCfg helper (writeCfg) sets SKILLDOZER_CONFIG to a real test file; the unsetEnvVar helper sets it to a GHOST file (these two are mutually exclusive — pick per test)."
  pattern: "Tests are package skillsdir (internal) — read unexported findConfig/findEnv directly. writeCfg(t, content) writes a temp config.yaml + sets SKILLDOZER_CONFIG. unsetEnvVar(t) unsets SKILLDOZER_SKILLS_DIR AND points SKILLDOZER_CONFIG at a ghost file. errors.Is for sentinel checks; strings.Contains for message-substring checks."
  gotcha: "Do NOT use unsetEnvVar in the new Find()-level tests that need a SPECIFIC config — it clobbers SKILLDOZER_CONFIG. Neutralize SKILLDOZER_SKILLS_DIR with t.Setenv(envVar, \"\") instead (findEnv treats empty == unset)."

# READ-ONLY — the parallel sibling PRP (Issue 1, fish completion) — confirms the disjoint boundary
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M1T1S3/PRP.md
  why: "T1.S3 edits ONLY completions/skilldozer.fish (//go:embed'd at main.go:60); it touches NO .go file. DISJOINT from internal/skillsdir/. Land in either order; no merge conflict."

# READ-ONLY — the PRD authority
- file: PRD.md
  why: "READ-ONLY (but this is the bugfix-2 PRD context — the selected sections ARE the issue_analysis). §6.4 ('configured but the dir vanished → concise reason + fix, exit 1'); §8.1 ('missing/unreadable config → fall through, never a hard error'). These two sentences are the exact contract this fix enforces. Do NOT edit PRD.md."
  section: "h2.2/h3.1 (Issue 2) in the bugfix requirements doc."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/tasks.json
  why: "P1.M1.T2.S1's CONTRACT block is authoritative INPUT/LOGIC/OUTPUT. This PRP transcribes it; tasks.json wins on conflict."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && wc -l internal/skillsdir/skillsdir.go internal/skillsdir/skillsdir_test.go
~302 / ~650   (findConfig 92-128; ErrNotFound 272-275; Find 288-302; 6 findConfig tests 555-624)
$ go test ./internal/skillsdir/...   # green today (the bug is behavioral; TestFindConfigStoreDirAbsent LOCKS the buggy fall-through)
$ grep -n '"fmt"' internal/skillsdir/skillsdir.go   # NOTHING — fmt is NOT imported (must add)
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep).
# main.go: 7 skillsdir.Find() callers, all `if err != nil { fmt.Fprintln(stderr, err); return 1 }`.
```

### Desired Codebase tree with files to be changed

```bash
internal/skillsdir/skillsdir.go       # MODIFY — +fmt import; +ErrConfiguredStoreMissing sentinel; findConfig 4-value; Find vanished-check; 2 doc comments
internal/skillsdir/skillsdir_test.go  # MODIFY — 6 call-site `, _` appends; rename TestFindConfigStoreDirAbsent→...Vanished + assertion; +2 new Find()-level tests
# main.go — UNCHANGED (callers handle err generically)
# go.mod / go.sum — UNCHANGED (only stdlib fmt added)
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `skillsdir.go` (findConfig) | 4-value signature; vanished branch returns `store`; 4 other paths `""` | Contract LOGIC b + decisions D1 |
| `skillsdir.go` (Find) | 4-value capture + vanished→wrapped ErrConfiguredStoreMissing | Contract LOGIC c + decisions D2 |
| `skillsdir.go` (sentinel + import + docs) | `ErrConfiguredStoreMissing`; `"fmt"`; 2 doc comments | Contract LOGIC a + DOCS |
| `skillsdir_test.go` | 6 call sites; 1 rename+assertion; 2 new tests | Contract LOGIC d-g |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — fmt is NOT imported today. The new fmt.Errorf("%w: ...", ErrConfiguredStoreMissing,
// vanished) REQUIRES "fmt" in the import block. Today skillsdir.go imports only errors/io/fs/os/
// path/filepath/config. ADD "fmt" (alphabetically after "errors", before "io/fs"). Miss this and
// `go build` fails with "undefined: fmt". (research/verified_facts.md §3.)

// GOTCHA #2 — findConfig STILL "never errors" (locked per-rule shape). It returns a 4th STRING
// value (vanishedStore); Find() is what constructs the error. Do NOT change findConfig to return
// an error — that breaks the per-rule symmetry with findEnv/findSibling/findWalkUp and is explicitly
// REJECTED in decisions.md §D1 (alternative 3). The 4-value return is the sanctioned deviation.

// GOTCHA #3 — The vanishedStore is returned ONLY at line 126 (the os.Stat-failed branch). The other
// 4 return paths (109/113/116/128) return "". The distinction is critical: lines 109/113/116 mean
// "config file missing/unreadable/no-store-key" → genuine fall-through (§8.1); line 126 means
// "config present, store resolved, but dir gone" → vanished (§6.4). Do NOT populate vanishedStore
// on the fall-through paths or Find will spuriously error on a plain missing config.

// GOTCHA #4 — Place the vanished-check in Find() AFTER `if ok { return ...nil }` but BEFORE the
// sibling/walk-up calls. If you put it after sibling/walk-up, the silent-wrong-store bug returns
// (Find would resolve the unrelated sibling store with exit 0 before checking vanished). The
// order MUST be: findEnv → findConfig → (if ok return) → (if vanished error) → findSibling → ...
// (decisions.md §D3 + the contract LOGIC c ordering.)

// GOTCHA #5 — Env (priority 1) STILL wins. findEnv runs before findConfig; if SKILLDOZER_SKILLS_DIR
// is set+valid, findConfig is never called and vanished never fires. TestFindEnvOverridesVanishedStore
// proves this. Do NOT move the vanished-check before findEnv — that would break the priority order.

// GOTCHA #6 — TestFindAllMissReturnsErrNotFound (513) STAYS GREEN; do not touch it. unsetEnvVar
// points SKILLDOZER_CONFIG at a GHOST file → config.Load returns fs.ErrNotExist → findConfig bails
// at line 113 (config.Load-err), returning vanished="". It NEVER reaches line 126. (research/
// verified_facts.md §7; skillsdir_research.md §4.) Editing it to "fix" a non-bug is scope creep.

// GOTCHA #7 — In the new Find()-level tests, do NOT use unsetEnvVar to neutralize SKILLDOZER_SKILLS_DIR.
// unsetEnvVar ALSO clobbers SKILLDOZER_CONFIG (points it at a ghost), which would conflict with
// writeCfg's real test config. Use t.Setenv(envVar, "") to neutralize ONLY the skills-dir env
// (findEnv treats val=="" as unset via its `!ok || val == ""` guard). writeCfg sets SKILLDOZER_CONFIG.

// GOTCHA #8 — Go scoping in Find(): after `d, s, ok, vanished := findConfig()`, the later
// `if d, s, ok := findSibling(); ok {` REDECLARES d/s/ok in the if-init scope (shadowing) — legal.
// All 4 findConfig values are USED (d/s/ok in `if ok { return d,s,nil }`; vanished in the check),
// so there is no "declared and not used" error. Do not rename to avoid the shadow.

// GOTCHA #9 — The error message is the user-facing doc (contract DOCS §5). It MUST contain: the
// configured path (%q → vanished) AND the fix "skilldozer --init". TestFindErrorsOnVanishedConfiguredStore
// asserts errors.Is(err, ErrConfiguredStoreMissing) (type) + the message substrings. Use %w (not %s)
// so errors.Is works (decisions D2). The wrapped message reads: "configured skills store directory
// does not exist: configured store \"<path>\" does not exist; run `skilldozer --init` or recreate
// the directory".

// GOTCHA #10 — main.go is UNCHANGED. Its 7 Find() callers all do `if err != nil { fmt.Fprintln(stderr,
// err); return 1 }` (verified at main.go:712-716). The wrapped error propagates through that existing
// handling verbatim — printed to stderr, exit 1, nothing on stdout. Do NOT add a skillsdir.ErrConfiguredStoreMissing
// special-case in main.go; the generic handler is correct and matches §6.4.

// GOTCHA #11 — gofmt -l must print nothing. The 4-value returns will re-flow if alignment matters
// (Go return lists don't column-align within a single func signature, so no reflow expected). Run
// gofmt -w if it lists a file. No deps change (only stdlib fmt). go.mod/go.sum byte-for-byte identical.
```

---

## Implementation Blueprint

### Data models and structure

One new package-level sentinel error + one widened function signature. No structs, no types, no config-field changes. `findConfig`'s 4th return value is a plain `string`.

```go
// NEW sentinel (beside ErrNotFound):
var ErrConfiguredStoreMissing = errors.New("configured skills store directory does not exist")

// WIDENED signature:
func findConfig() (dir string, src Source, found bool, vanishedStore string)
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: ADD "fmt" to internal/skillsdir/skillsdir.go imports (GOTCHA #1)
  - FILE: internal/skillsdir/skillsdir.go, the import block (top of file).
  - CURRENT:
        import (
        	"errors"
        	"io/fs"
        	"os"
        	"path/filepath"

        	"github.com/dabstractor/skilldozer/internal/config"
        )
  - ADD "fmt" alphabetically (after "errors", before "io/fs"):
        import (
        	"errors"
        	"fmt"
        	"io/fs"
        	"os"
        	"path/filepath"

        	"github.com/dabstractor/skilldozer/internal/config"
        )
  - WHY: Task 4's fmt.Errorf needs it. Today fmt is NOT imported (only referenced in a comment).

Task 2: ADD the ErrConfiguredStoreMissing sentinel (LOGIC a + decisions D2)
  - FILE: internal/skillsdir/skillsdir.go. Place IMMEDIATELY AFTER the ErrNotFound var (line 275).
  - ADD:
        // ErrConfiguredStoreMissing is returned by Find when the config file is present and
        // names a `store:` directory that does NOT exist on disk (§6.4: "configured but the
        // dir vanished"). Unlike a missing config file (§8.1 fall-through), this is a loud
        // error so `pi --skill "$(skilldozer x)"` fails instead of silently loading an
        // unrelated sibling/walk-up store. Find wraps it via fmt.Errorf("%w: ...", ...) so
        // callers can errors.Is(err, ErrConfiguredStoreMissing). Print verbatim to stderr.
        var ErrConfiguredStoreMissing = errors.New("configured skills store directory does not exist")
  - NAMING: ErrConfiguredStoreMissing (contract + decisions D2). Mirrors the ErrNotFound sentinel style.

Task 3: WIDEN findConfig to the 4-value signature + set vanishedStore on the vanished branch (LOGIC b)
  - FILE: internal/skillsdir/skillsdir.go, findConfig (106-128).
  - EDIT signature (106): func findConfig() (dir string, src Source, found bool, vanishedStore string) {
  - EDIT the 5 return paths:
      109:  return "", 0, false, ""        // (was: return "", 0, false)
      113:  return "", 0, false, ""        // (was: return "", 0, false)
      116:  return "", 0, false, ""        // (was: return "", 0, false)
      126:  return "", 0, false, store     // (was: return "", 0, false)  ← THE FIX (vanishedStore = resolved store)
      128:  return store, SourceConfig, true, ""   // (was: return store, SourceConfig, true)
  - PRESERVE the store-resolution logic (118-123) and the os.Stat (124) UNCHANGED. The `store`
    var is already in scope at 126 — that is exactly the vanishedStore value.
  - GOTCHA #2/#3: do NOT return an error; do NOT populate vanishedStore on the 4 non-vanished paths.

Task 4: ADD the vanished-check to Find() (LOGIC c)
  - FILE: internal/skillsdir/skillsdir.go, Find (288-302).
  - REPLACE lines 292-294:
        if d, s, ok := findConfig(); ok { // PRD §8.3 priority #2
        	return d, s, nil
        }
    WITH:
        d, s, ok, vanished := findConfig() // PRD §8.3 priority #2
        if ok {
        	return d, s, nil
        }
        if vanished != "" {
        	// Config is present with a `store:` key, but the named dir does not exist (§6.4:
        	// "configured but the dir vanished"). This is NOT "unconfigured" — falling through
        	// to sibling/walk-up would silently load an UNRELATED store, so fail loudly instead.
        	return "", 0, fmt.Errorf("%w: configured store %q does not exist; run `skilldozer --init` or recreate the directory", ErrConfiguredStoreMissing, vanished)
        }
  - GOTCHA #4/#8: the vanished-check goes AFTER `if ok` and BEFORE findSibling. The later
    `if d, s, ok := findSibling(); ok {` legally shadows d/s/ok (redeclares in if-init scope).
  - PRESERVE findEnv (289-291), findSibling (295-297), findWalkUp (298-300), the final ErrNotFound (301).

Task 5: UPDATE the findConfig + Find doc comments (Mode A / DOCS)
  - findConfig doc (the "Returns …" sentence, ~104-105). CURRENT:
        // Returns (absStore, SourceConfig, true) on a hit; ("", 0, false) otherwise so Find()
        // can fall through to the sibling rule. Never errors (locked per-rule shape).
    REPLACE WITH:
        // Returns (absStore, SourceConfig, true, "") on a hit. Returns ("", 0, false, "") on the
        // genuine fall-through paths (no config file / unreadable / no `store` key) so Find() can
        // fall through to the sibling rule — PRD §8.1: a missing/unreadable config NEVER hard-errors.
        // Returns ("", 0, false, vanishedStore) ONLY when the config file is present with a `store:`
        // key whose resolved dir does not exist (§6.4 "configured but the dir vanished") — Find()
        // turns that into a wrapped ErrConfiguredStoreMissing instead of falling through. Never
        // errors directly (locked per-rule shape); the vanishedStore string is the signal Find() uses.
  - Find doc (the numbered list + the "If all miss" sentence, ~277-287). Add a note after the
    numbered list, before the "first rule to hit" paragraph:
        // NOTE (§6.4): if the config file is present with a `store:` naming a non-existent dir,
        // Find returns a wrapped ErrConfiguredStoreMissing (configuring-then-vanished) instead of
        // falling through — so a vanished store never silently resolves tags from an unrelated
        // sibling/walk-up store. A genuinely missing/unreadable config still falls through (§8.1).
  - PRESERVE the rest of both doc comments (the priority list, the ErrNotFound description).

Task 6: UPDATE the 6 findConfig test call sites (LOGIC d) + rename/assert TestFindConfigStoreDirAbsent (LOGIC e)
  - FILE: internal/skillsdir/skillsdir_test.go.
  - 5 call sites — append ", _" (TestFindConfigHit@558, MissingFile@573, MissingStoreKey@581,
    MalformedYAML@600, RelativeStoreResolvedAgainstConfigDir@614). Example:
        // BEFORE: got, src, found := findConfig()
        // AFTER:  got, src, found, _ := findConfig()
        // BEFORE: if dir, src, found := findConfig(); found {
        // AFTER:  if dir, src, found, _ := findConfig(); found {
  - RENAME TestFindConfigStoreDirAbsent (587) → TestFindConfigStoreVanished AND change the assertion
    (LOGIC e). NEW body:
        func TestFindConfigStoreVanished(t *testing.T) {
        	store := filepath.Join(t.TempDir(), "no-such-store")
        	writeCfg(t, "store: "+store+"\n")
        	dir, src, found, vanished := findConfig()
        	if found {
        		t.Errorf("findConfig vanished store: got found=true dir=%q src=%v; want false", dir, src)
        	}
        	if vanished != store {
        		t.Errorf("findConfig vanished store: vanished=%q; want %q (the configured path)", vanished, store)
        	}
        }
  - GOTCHA #6: do NOT touch TestFindAllMissReturnsErrNotFound (513) — it stays green.

Task 7: ADD TestFindErrorsOnVanishedConfiguredStore (LOGIC f)
  - FILE: internal/skillsdir/skillsdir_test.go. Place near the other Find() combiner tests
    (after TestFindAllMissReturnsErrNotFound / TestErrNotFoundMessageHasFix).
  - BODY:
        // Issue 2: a present config whose `store:` names a non-existent dir makes Find() return
        // a wrapped ErrConfiguredStoreMissing (NOT fall through to sibling/walk-up). §6.4: "configured
        // but the dir vanished" → reason + fix, exit 1, nothing on stdout.
        func TestFindErrorsOnVanishedConfiguredStore(t *testing.T) {
        	t.Setenv(envVar, "")                            // neutralize SKILLDOZER_SKILLS_DIR (env must NOT override) — GOTCHA #7
        	t.Chdir(t.TempDir())                            // hermetic: no walk-up ancestor skills/ (moot — Find errors before walk-up)
        	store := filepath.Join(t.TempDir(), "no-such-store")
        	writeCfg(t, "store: "+store+"\n")               // present config, vanished store
        	dir, src, err := Find()
        	if !errors.Is(err, ErrConfiguredStoreMissing) {
        		t.Fatalf("Find() vanished store: err=%v; want ErrConfiguredStoreMissing", err)
        	}
        	if dir != "" || src != 0 {
        		t.Errorf("Find() vanished store: dir=%q src=%v; want \"\" and 0", dir, src)
        	}
        	msg := err.Error()
        	if !strings.Contains(msg, store) {
        		t.Errorf("Find() vanished store: message %q missing the configured path %q", msg, store)
        	}
        	if !strings.Contains(msg, "skilldozer --init") {
        		t.Errorf("Find() vanished store: message %q missing the fix 'skilldozer --init'", msg)
        	}
        }
  - NOTE: writeCfg sets SKILLDOZER_CONFIG to the real test config; t.Setenv(envVar,"") neutralizes
    ONLY the skills-dir env (GOTCHA #7). Find() errors at the vanished-check BEFORE findSibling/
    findWalkUp run, so cwd is moot (t.Chdir is belt-and-suspenders hermeticity).

Task 8: ADD TestFindEnvOverridesVanishedStore (LOGIC g)
  - FILE: internal/skillsdir/skillsdir_test.go. Place after TestFindErrorsOnVanishedConfiguredStore.
  - BODY:
        // Issue 2 (decisions D3): SKILLDOZER_SKILLS_DIR (priority 1) wins over a vanished config
        // store (priority 2) — findEnv runs first, findConfig is never called, no error.
        func TestFindEnvOverridesVanishedStore(t *testing.T) {
        	envStore := t.TempDir()                         // an EXISTING dir for the env override
        	t.Setenv(envVar, envStore)
        	writeCfg(t, "store: "+filepath.Join(t.TempDir(), "no-such-store")+"\n") // present config, vanished store
        	got, src, err := Find()
        	if err != nil {
        		t.Fatalf("Find() env-override: err=%v; want nil (env beats vanished config)", err)
        	}
        	if src != SourceEnv {
        		t.Errorf("Find() env-override: src=%v; want SourceEnv", src)
        	}
        	if want := filepath.Clean(envStore); got != want {
        		t.Errorf("Find() env-override: dir=%q; want %q", got, want)
        	}
        }

Task 9: VERIFY build + vet + the package test suite (the gate)
  - COMMAND: gofmt -l internal/skillsdir/   (must print NOTHING)
  - COMMAND: go build ./...                 (exit 0 — proves fmt import + signature compile)
  - COMMAND: go vet ./...                   (exit 0)
  - COMMAND: go test ./internal/skillsdir/... -v   (all pass, incl. the renamed + 2 new tests; TestFindAllMissReturnsErrNotFound stays green)
  - COMMAND: go test ./...                  (whole module green — main.go callers unchanged)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"
  - COMMAND: git diff --quiet main.go && echo "main.go unchanged"   (GOTCHA #10)
```

### Implementation Patterns & Key Details

```go
// The sentinel (Task 2) — beside ErrNotFound, same pattern:
var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer --init`")
var ErrConfiguredStoreMissing = errors.New("configured skills store directory does not exist")

// findConfig's vanished branch (Task 3) — the ONE line that changes semantics:
//   BEFORE:  return "", 0, false // store path is not an existing dir -> fall through
//   AFTER:   return "", 0, false, store // store vanished -> Find() errors (§6.4), do NOT fall through
// (`store` is the resolved path from 118-123; it is already computed before os.Stat at 124.)

// Find()'s vanished-check (Task 4) — order is load-bearing (GOTCHA #4):
d, s, ok, vanished := findConfig() // PRD §8.3 priority #2
if ok {
	return d, s, nil
}
if vanished != "" {
	return "", 0, fmt.Errorf("%w: configured store %q does not exist; run `skilldozer --init` or recreate the directory", ErrConfiguredStoreMissing, vanished)
}
// ... then findSibling, findWalkUp, ErrNotFound (unchanged)
```

Notes easy to get wrong:
- `findConfig` still has the `(dir, src, found)` *prefix* — only a 4th value is appended. The existing 6 tests just need `, _` (or `, vanished`); none of their first-3-value assertions change (except the renamed test's new vanished assertion).
- `%w` (not `%s`) wraps the sentinel so `errors.Is` works (decisions D2). The message includes both `%q` (the path) and the literal fix string so the test's two `strings.Contains` checks pass.
- `findEnv` runs before `findConfig`, so env-override is free (decisions D3) — no special handling in Find() for the env-beats-vanished case.

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **4-value return vs error-return? → 4-value (decisions D1).** An error return would break the per-rule `(dir, src, found)` symmetry and require Find() to treat findConfig's errors specially. The 4-value return keeps the first 3 values identical for all miss paths; only line 126 populates vanishedStore.
2. **Sentinel + %w vs a typed error? → sentinel + %w (decisions D2).** Mirrors ErrNotFound; `errors.Is` works; the dynamic path is in the message, not matched by type. Tests assert type via `errors.Is` + message via `strings.Contains`.
3. **Vanished-check placement? → after `if ok`, before sibling (GOTCHA #4).** Must run BEFORE sibling/walk-up or the silent-wrong-store bug returns. After findEnv (priority order, decisions D3).
4. **Neutralize SKILLDOZER_SKILLS_DIR in the new tests? → t.Setenv(envVar, ""), NOT unsetEnvVar (GOTCHA #7).** unsetEnvVar clobbers SKILLDOZER_CONFIG (ghost file), conflicting with writeCfg. `t.Setenv(envVar, "")` neutralizes only the skills-dir env (findEnv treats "" == unset).
5. **Rename vs keep TestFindConfigStoreDirAbsent? → rename to TestFindConfigStoreVanished (contract LOGIC e).** The old name ("DirAbsent") described the buggy fall-through; the new name ("StoreVanished") describes the new semantics. The assertion changes from "found==false" to "found==false && vanished==configuredPath".

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. The only new import is stdlib fmt. (GOTCHA #11)

CALLERS (unchanged — GOTCHA #10):
  - main.go: 7 skillsdir.Find() callers (541/564/600/640/681/712/1213), all:
        if err != nil { fmt.Fprintln(stderr, err); return 1 }
    The wrapped ErrConfiguredStoreMissing propagates verbatim → stderr, exit 1, empty stdout.
    ZERO main.go edits.

PACKAGE INTERNALS:
  - findConfig's new 4th value is consumed ONLY by Find() (1 production site). The 6 test
    sites append ", _" (or ", vanished"). No other caller of findConfig exists (verified).

NO ROUTES / NO DATABASE / NO CONFIG SCHEMA / NO COMPLETIONS:
  - T2.S1 touches only the skillsdir package. The config package (config.Path/config.Load) is
    consumed unchanged. Completions are Issue 1 (T1.S3, parallel, disjoint).
```

---

## Validation Loop

### Level 1: Syntax & Style + build/vet (immediate)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l internal/skillsdir/   # must print NOTHING
go build ./...                 # expect exit 0 (proves the fmt import + 4-value signature compile)
go vet ./...                   # expect exit 0
# Expected: zero output / exit 0. If go build says "undefined: fmt" you forgot Task 1.
```

### Level 2: The package unit tests (the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test ./internal/skillsdir/... -v
# Expected: ALL pass. Load-bearing assertions:
#   TestFindConfigStoreVanished (renamed)       -> found=false AND vanished==configured path
#   TestFindErrorsOnVanishedConfiguredStore      -> errors.Is(err, ErrConfiguredStoreMissing) + msg has path + "skilldozer --init"
#   TestFindEnvOverridesVanishedStore            -> err==nil, src==SourceEnv (env beats vanished config)
#   TestFindAllMissReturnsErrNotFound (513)      -> STILL GREEN (ghost config bails at 113, not 126)
#   TestFindConfigHit/MissingFile/MissingStoreKey/MalformedYAML/RelativeStore -> the 5 ", _" call sites compile + pass

# Isolated re-run of just the new/renamed tests:
go test ./internal/skillsdir/... -run 'Vanished|ErrorsOnVanished|EnvOverridesVanished' -v
# Expected: 3 passed.
```

### Level 3: Whole-module regression + invariants

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # Expected: 0
go vet  ./...  ; echo "vet exit $?"     # Expected: 0
go test ./...  ; echo "test exit $?"    # Expected: 0 (main.go callers unchanged; no other package regresses)

# GOTCHA #10/#11 invariants:
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
git diff --quiet main.go && echo "main.go unchanged" || echo "FAIL: main.go changed"
# Expected: "deps unchanged" + "main.go unchanged".

# Scope invariants:
grep -c 'ErrConfiguredStoreMissing' internal/skillsdir/skillsdir.go           # expect >= 3 (sentinel + doc + fmt.Errorf)
grep -c 'vanishedStore' internal/skillsdir/skillsdir.go                       # expect >= 2 (signature + the vanished return)
grep -c '"fmt"' internal/skillsdir/skillsdir.go                               # expect 1 (the added import)
```

### Level 4: End-to-end behavioral check (the §6.4 contract)

```bash
cd /home/dustin/projects/skilldozer
go build -o /tmp/sdz . || { echo "FAIL: build"; exit 1; }

TMP=$(mktemp -d); CFG="$TMP/cfg.yaml"; printf 'store: %s/no-such\n' "$TMP" > "$CFG"

# (a) Configured-but-vanished → exit 1, nothing on stdout, reason+fix on stderr (NOT a sibling/walk-up path):
out=$(env -u SKILLDOZER_SKILLS_DIR SKILLDOZER_CONFIG="$CFG" /tmp/sdz --path 2>/tmp/err); code=$?
[ "$code" = 1 ] && [ -z "$out" ] && grep -q "does not exist" /tmp/err && grep -q "skilldozer --init" /tmp/err \
  && echo "OK: vanished store → exit 1, empty stdout, reason+fix on stderr" || echo "FAIL: code=$code out=$out err=$(cat /tmp/err)"

# (b) Control: env override beats vanished config → exit 0, env store on stdout:
ENVSTORE="$TMP/envstore"; mkdir -p "$ENVSTORE"
out=$(SKILLDOZER_SKILLS_DIR="$ENVSTORE" SKILLDOZER_CONFIG="$CFG" /tmp/sdz --path 2>/dev/null); code=$?
[ "$code" = 0 ] && [ "$out" = "$ENVSTORE" ] && echo "OK: env overrides vanished config (exit 0)" || echo "FAIL: code=$code out=$out"

# (c) Control: missing config file → still falls through (NOT the vanished error) — §8.1:
out=$(env -u SKILLDOZER_SKILLS_DIR SKILLDOZER_CONFIG="$TMP/no-such-cfg.yaml" /tmp/sdz --path 2>/tmp/err2); code=$?
([ "$code" = 0 ] || grep -q "not configured" /tmp/err2) && echo "OK: missing config → fall-through (not vanished error)" || echo "FAIL: code=$code err=$(cat /tmp/err2)"

rm -rf /tmp/sdz "$TMP" /tmp/err /tmp/err2
# Expected: all three print "OK". (a) is the bugfix; (b) proves env-still-wins (D3); (c) proves §8.1 fall-through survives.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` clean, `go build ./...` exit 0, `go vet ./...` exit 0
- [ ] Level 2 PASS — renamed TestFindConfigStoreVanished + 2 new tests pass; TestFindAllMissReturnsErrNotFound stays green; the 5 `, _` call sites compile + pass
- [ ] Level 3 PASS — `go test ./...` exit 0; `git diff go.mod go.sum` → "deps unchanged"; `git diff main.go` → "main.go unchanged"
- [ ] Level 4 PASS — vanished store → exit 1 + reason/fix on stderr + empty stdout; env overrides → exit 0; missing config → still falls through

### Feature Validation
- [ ] `findConfig` has the 4-value signature; the vanished branch (126) returns `store`; the other 4 paths return `""`
- [ ] `ErrConfiguredStoreMissing` sentinel exists beside `ErrNotFound`; wrapped via `fmt.Errorf("%w: ...")` in Find()
- [ ] `Find()` returns the wrapped sentinel (assertable via `errors.Is`) on present-config + vanished-store, BEFORE sibling/walk-up
- [ ] Missing/unreadable/no-store-key config (109/113/116) STILL falls through (§8.1) — no error
- [ ] findEnv (priority 1) still wins (TestFindEnvOverridesVanishedStore) — no error
- [ ] `"fmt"` added to skillsdir.go imports

### Code Quality / Convention Validation
- [ ] Sentinel mirrors ErrNotFound style; doc comments cite §6.4/§8.1
- [ ] Tests use `errors.Is` (type) + `strings.Contains` (message), matching the existing test style
- [ ] No new deps (only stdlib fmt); go.mod/go.sum byte-for-byte identical
- [ ] Minimal diff (1 import + 1 sentinel + 5 return-value appends + 1 Find block + 2 doc comments + 6 test edits + 2 new tests)

### Scope Discipline (the Issue 1 / main.go / PRD boundaries)
- [ ] Did NOT touch `main.go` (callers handle err generically — verified)
- [ ] Did NOT touch `completions/*` (Issue 1 = T1.S3, parallel, fish file)
- [ ] Did NOT touch `internal/config` (consumed unchanged)
- [ ] Did NOT touch `TestFindAllMissReturnsErrNotFound` (stays green — ghost config bails at 113)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't forget to add `"fmt"` to the imports.** It's not imported today; the new `fmt.Errorf` needs it. Miss it → `go build` fails with "undefined: fmt". (GOTCHA #1)
- ❌ **Don't make `findConfig` return an error.** That breaks the per-rule `(dir, src, found)` symmetry and is explicitly REJECTED (decisions D1). It returns a 4th STRING; Find constructs the error. (GOTCHA #2)
- ❌ **Don't populate `vanishedStore` on the fall-through paths (109/113/116).** Only line 126 (os.Stat failed AFTER a store was resolved) is "vanished". The fall-throughs mean "no config / no store key" (§8.1, silent) — populating vanished there makes Find spuriously error on a plain missing config. (GOTCHA #3)
- ❌ **Don't put the vanished-check after sibling/walk-up.** It MUST go `findConfig → if ok → if vanished(error) → findSibling`. Putting it later resurrects the silent-wrong-store bug. (GOTCHA #4)
- ❌ **Don't touch `TestFindAllMissReturnsErrNotFound`.** It stays green: unsetEnvVar points SKILLDOZER_CONFIG at a ghost file → findConfig bails at line 113 (config.Load err), never reaching 126. "Fixing" it is editing a non-bug. (GOTCHA #6)
- ❌ **Don't use `unsetEnvVar` in the new Find()-level tests.** It clobbers SKILLDOZER_CONFIG (ghost file), conflicting with writeCfg. Use `t.Setenv(envVar, "")` to neutralize only the skills-dir env. (GOTCHA #7)
- ❌ **Don't use `%s` instead of `%w`** in the fmt.Errorf — `%w` is what makes `errors.Is(err, ErrConfiguredStoreMissing)` work (decisions D2). The test relies on it.
- ❌ **Don't add a `skillsdir.ErrConfiguredStoreMissing` special-case in main.go.** The generic `fmt.Fprintln(stderr, err); return 1` handler is correct and matches §6.4. (GOTCHA #10)
- ❌ **Don't move the vanished-check before findEnv.** Env is priority 1; it must still override the vanished config (decisions D3, TestFindEnvOverridesVanishedStore). (GOTCHA #5)
- ❌ **Don't add deps.** Only stdlib `fmt` is added; go.mod/go.sum byte-for-byte identical. (GOTCHA #11)

---

## Confidence Score

**9.5/10** — Every edit site is pinned to its exact current line with before→after text (`research/verified_facts.md` §1-§6); the design is fixed by decisions.md §D1-§D3 (4-value return, sentinel+%w, env-still-wins); the `fmt`-import prerequisite, the main.go-no-change proof, the `TestFindAllMissReturnsErrNotFound` safety proof, and the disjoint boundary with the parallel T1.S3 (fish completion) are all verified. The change is small and surgical (1 import + 1 sentinel + 5 return-value appends + 1 Find block + 2 doc comments + 6 test edits + 2 new tests), and the existing test harness (writeCfg/unsetEnvVar/errors.Is/strings.Contains) directly supports the new tests. The 0.5 reservation is for the one judgment call a reviewer could second-guess: whether the `findConfig` doc-comment rewrite (Task 5) should be as verbose as specified — the PRP writes it for completeness (Mode A), but a strict reader could keep it shorter. The logic, signatures, and tests are pinned with no ambiguity.
