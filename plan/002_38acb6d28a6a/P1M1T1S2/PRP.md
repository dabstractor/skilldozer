# PRP — P1.M1.T1.S2: `config.Path()` + `config.DefaultStore()` — XDG-aware config-file path and default store dir

> **Subtask:** P1.M1.T1.S2 — the path-resolution half of milestone P1.M1's `internal/config` package.
> **Scope boundary:** EXTENDS the `internal/config` package created by the parallel sibling **P1.M1.T1.S1** (`config.File` + `Load` + `Save`). Adds two exported functions (`Path`, `DefaultStore`) and one env-var constant (`configEnv = "SKILLDOZER_CONFIG"`). Does NOT read or write the config file (that is S1's `Load`/`Save`), does NOT wire anything into `skillsdir.Find()` (T2.S2), and does NOT touch `init` (M2.T2). This PRP's output is a pure env→path resolver layer with no filesystem side effects beyond `os.Getenv`/`os.UserConfigDir`/`os.UserHomeDir`.

---

## Goal

**Feature Goal**: Add the XDG-aware path resolution to `internal/config` so later subtasks can locate (a) the config *file* (`config.Path()`, PRD §8.1 — `$XDG_CONFIG_HOME/skilldozer/config.yaml` with a `$SKILLDOZER_CONFIG` override) and (b) the *default skills store* dir (`config.DefaultStore()`, PRD §8.2/§8.3 — `$XDG_DATA_HOME/skilldozer/skills` falling back to `~/.local/share/skilldozer/skills`). Both are pure functions of the process environment.

**Deliverable**: Two additions to files that S1 created:
1. `internal/config/config.go` — APPEND: an unexported `const configEnv = "SKILLDOZER_CONFIG"`, the exported `func Path() (string, error)`, the exported `func DefaultStore() (string, error)`, and an extension of the package doc comment to document them.
2. `internal/config/config_test.go` — APPEND: table/behavioral tests exercising every branch the contract names (override-honored-as-literal-path, XDG default, relative-`XDG_CONFIG_HOME`-rejected, absolute-`XDG_DATA_HOME`-honored, relative/empty-`XDG_DATA_HOME`-falls-back-to-`~/.local/share`, `HOME`-unset-errors).

**Success Definition**: `go build ./internal/config/...` and `go test ./internal/config/... -v` pass; `go vet ./...` clean; the whole module still builds/tests green (`go test ./...`); `go.mod`/`go.sum` byte-for-byte unchanged; the exported surface of `internal/config` is exactly `{File, Load, Save, Path, DefaultStore}`.

---

## User Persona (if applicable)

Not applicable — this is an **internal** package with no user-facing surface (the contract's DOCS section says "none"). The `$SKILLDOZER_CONFIG` env var's *user-facing* description is documented later in the README (P1.M4.T2.S1, Mode B) — do NOT duplicate it here. Its only callers are: `skillsdir.findConfig` (P1.M1.T2.S2) consumes `Path`; the `init` flow (P1.M2.T2.S1/S2) consumes `DefaultStore` + `Path`.

---

## Why

- `Path()` is the bootstrap hook for the entire §8 config model. PRD §8.1 calls the config file "the **one** fixed, well-known path the binary can bootstrap from" — every other §8.3 rule (`SKILLDOZER_SKILLS_DIR`, sibling, walk-up) is a heuristic fallback, but the config file is the persistent, authoritative source of the store location that `init` writes. `findConfig` (T2.S2) cannot compile until `Path()` exists.
- `DefaultStore()` is what makes `init` (M2.T2) useful out of the box: it gives a `go install` user a sane default (`$XDG_DATA_HOME/skilldozer/skills`) instead of requiring a `SKILLDOZER_SKILLS_DIR` env var that is easy to typo (the exact pain external_deps.md §5 documents). Without it, the `init` prompt has no sensible default.
- Closing gap **G5** (code_prd_delta.md): today there is "No `configPath()` / `SKILLDOZER_CONFIG` / `os.UserConfigDir()` / ... anywhere." S1 delivered `cfgFile{Store}` (`File` + `Load`/`Save`); S2 delivers `configPath` (named `Path`) + `SKILLDOZER_CONFIG` + the `os.UserDataDir()`-by-hand computation. `findConfig()` itself is T2.S2.
- It is a **leaf** with zero new dependencies and no filesystem mutation — the safest possible unit to land in parallel with S1's read/write core.

---

## What

Two exported functions and one constant added to `internal/config` (package `config`). Exact signatures and logic, transcribed verbatim from the contract INPUT/LOGIC:

```go
// configEnv is the environment variable that overrides the config-file location
// (PRD §8.1). Set to an absolute or relative path to redirect skilldozer at a
// different config file (useful for tests / multiple profiles). Package-internal:
// no consumer needs the symbol — Path() encapsulates the read.
const configEnv = "SKILLDOZER_CONFIG"

// Path returns the path to the skilldozer config file (PRD §8.1).
func Path() (string, error)

// DefaultStore returns the default skills store directory (PRD §8.2 / §8.3).
func DefaultStore() (string, error)
```

**`Path()` logic (contract LOGIC, verified — see `research/verified_facts.md` §1, §3):**
1. If `v := os.Getenv(configEnv); v != ""` → return `filepath.Clean(v)` **AS-IS**. The override is the *literal* config-file path — absolute OR relative — and is **NOT** joined with the config home. (`filepath.Clean` does lexical `..`/trailing-slash cleanup only; no symlink evaluation.)
2. Else `configHome, err := os.UserConfigDir()`. On error → return `("", err)` verbatim (do NOT wrap). `os.UserConfigDir()` already implements `$XDG_CONFIG_HOME → ~/.config` AND rejects a relative `$XDG_CONFIG_HOME` with a non-nil error (empirically confirmed on Linux — error text `"path in $XDG_CONFIG_HOME is relative"`, but assert only `err != nil`, never the wording).
3. Else return `filepath.Join(configHome, "skilldozer", "config.yaml")`. `filepath.Join` always returns a clean path, so no extra `filepath.Clean` is needed on this branch.

**`DefaultStore()` logic (contract LOGIC, verified — see `research/verified_facts.md` §2, §4):**
1. If `v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v)` → base = `v`.
2. Else `home, err := os.UserHomeDir()`. On error → return `("", err)` verbatim.
3. Else base = `filepath.Join(home, ".local", "share")`. (There is **no** `os.UserDataDir()` — the XDG data-home rule is computed by hand, exactly as external_deps.md §2 prescribes.)
4. Return `filepath.Join(base, "skilldozer", "skills")`.

### Success Criteria

- [ ] `internal/config/config.go` adds exported `Path() (string, error)` and `DefaultStore() (string, error)` and the unexported `const configEnv = "SKILLDOZER_CONFIG"`
- [ ] `Path()` honors a non-empty `SKILLDOZER_CONFIG` override by returning `filepath.Clean(v)` **without joining** it to the config home (absolute AND relative override both work)
- [ ] `Path()` treats an empty `SKILLDOZER_CONFIG` as unset and falls through to `os.UserConfigDir()` (empty == unset, because `os.Getenv` returns `""` for both)
- [ ] `Path()` returns `("", err)` when `os.UserConfigDir()` errors (e.g. a relative `$XDG_CONFIG_HOME`), without wrapping
- [ ] `DefaultStore()` honors an **absolute** `XDG_DATA_HOME`
- [ ] `DefaultStore()` ignores a **relative** or empty `XDG_DATA_HOME` and falls back to `~/.local/share` (XDG-spec correctness: relative `XDG_*_HOME` is invalid)
- [ ] `DefaultStore()` returns `("", err)` when `os.UserHomeDir()` errors (`HOME` unset), without wrapping
- [ ] Package doc comment extended to document `Path`/`DefaultStore` (does not contradict S1's "settings sidecar, NOT a catalog index" framing)
- [ ] `config_test.go` passes all new tests; `go build/vet/test ./...` green; `go.mod`/`go.sum` unchanged

---

## All Needed Context

### Context Completeness Check

**Pass.** Both functions are fully pinned by the contract LOGIC, and every load-bearing stdlib behavior (the relative-`XDG_CONFIG_HOME` rejection, `filepath.Clean`/`IsAbs`/`Join` semantics, the `os.UserDataDir()` absence, the `HOME=""` error path) is **empirically verified** in `research/verified_facts.md` with observed outputs. The test conventions to mirror (`t.Setenv`, the `unsetEnvVar` helper, `t.TempDir`, direct assertions, no `t.Parallel` for env-mutating tests) are read from the existing `internal/skillsdir/skillsdir_test.go`. An implementer who has never seen this repo can complete it in one pass.

### Documentation & References

```yaml
# MUST READ — the parallel sibling PRP (the CONTRACT for what config.go/config_test.go look like when S2 begins)
- file: plan/002_38acb6d28a6a/P1M1T1S1/PRP.md
  why: "S1 creates internal/config/config.go (package doc + File + Load + Save) and config_test.go. S2 EXTENDS both files — you must match S1's package-doc-comment framing, its error-return style (verbatim, no wrap), and its 'no new dependency' invariant."
  pattern: "S2 adds Path/DefaultStore to the SAME file S1 created; do not create paths.go. Append tests to the SAME config_test.go."
  gotcha: "S1's exported surface is {File, Load, Save}. S2 widens it to {File, Load, Save, Path, DefaultStore} — that is the intended delta, not scope creep."

# MUST READ — the authoritative external fact sheet (§2 is the XDG/path-resolution bible for this subtask)
- file: plan/002_38acb6d28a6a/architecture/external_deps.md
  why: "§2 gives the exact configHome()/dataHome() skeletons, confirms os.UserConfigDir implements the config-home rule (and rejects relative XDG_CONFIG_HOME), confirms there is NO os.UserDataDir() so data-home is computed by hand, and cites the XDG spec defaults (~/.config, ~/.local/share). §1 (yaml.v3) is S1's concern — read only to confirm you are NOT touching Load/Save."
  section: "§2 (XDG semantics on Linux) — the configHome/dataHome skeleton is almost line-for-line what Path/DefaultStore become."

# MUST READ — the test conventions to mirror EXACTLY
- file: internal/skillsdir/skillsdir_test.go
  why: "THE env-mutation test pattern: t.Setenv to set vars; unsetEnvVar(tb) helper for the rare unset case; t.TempDir() for controlled absolute paths; direct got!=want assertions; explicit 'do NOT call t.Parallel' comments on env tests; behavioral test names (TestFindEnvExistingDir, TestFindEnvRelativePathAbsolutized)."
  pattern: "Mirror as: t.Setenv(configEnv, ...), t.Setenv('XDG_CONFIG_HOME', tempAbs), t.Setenv('XDG_DATA_HOME', ...), t.Setenv('HOME', tempHome). No testify. Assert err==nil/!=nil plus the exact returned path string."
  gotcha: "t.Setenv CANNOT unset a var — it only sets. For SKILLDOZER_CONFIG, setting '' is equivalent to unset because Path uses os.Getenv (returns '' for both) and checks v != ''. For genuinely unsetting XDG vars where empty!=unset would matter, copy the unsetEnvVar helper — but Path/DefaultStore use os.Getenv+'!= \"\"' so empty IS unset and t.Setenv(var,'') suffices."

# MUST READ — the gap this closes (G5) and the consumer interfaces it fixes
- file: plan/002_38acb6d28a6a/architecture/code_prd_delta.md
  why: "G5 is the exact gap: 'No configPath()/SKILLDOZER_CONFIG/os.UserConfigDir()/cfgFile{Store}/findConfig() anywhere.' S1 delivered cfgFile{Store}+Load/Save; S2 delivers configPath (named Path)+SKILLDOZER_CONFIG+os.UserConfigDir()/os.UserHomeDir(). findConfig() itself is T2.S2 (out of scope here)."
  section: "§2 (G5) + §10 gap index."

# MUST READ — the empirical proof of every hard claim in this PRP
- file: plan/002_38acb6d28a6a/P1M1T1S2/research/verified_facts.md
  why: "Observed outputs for os.UserConfigDir (relative XDG_CONFIG_HOME -> error 'path in $XDG_CONFIG_HOME is relative'), os.UserHomeDir (HOME='' -> error), filepath.Clean/IsAbs/Join on every input the tests use. Locks the test assertions to ground truth."
  critical: "§1 corrects the common misconception that os.UserConfigDir does NOT validate absoluteness on Linux — it DOES. §3 documents the filepath.Clean('')='.' gotcha (moot because of the v!='' guard). §7 fixes the configEnv exported-vs-unexported decision."

# MUST READ — the contract (READ-ONLY, the orchestrator owns it)
- file: plan/002_38acb6d28a6a/tasks.json
  why: "P1.M1.T1.S2's CONTRACT block is the authoritative INPUT/LOGIC/OUTPUT. This PRP transcribes it; if anything here seems to contradict tasks.json, tasks.json wins."

# READ-ONLY — the PRD sections selected as relevant
- file: PRD.md
  why: "READ-ONLY. §8.1 (config file default location + SKILLDOZER_CONFIG override + 'the one fixed well-known path'); §8.2 (init's default-store = $XDG_DATA_HOME/skilldozer/skills); §8.3 (5-rule priority — Path is consumed by rule 2 'config file')."
  section: "h2.7 / h3.8 / h3.9 / h3.10."

- url: https://pkg.go.dev/os#UserConfigDir
  why: "Confirms: on Unix returns $XDG_CONFIG_HOME if non-empty else $HOME/.config; returns an error if $XDG_CONFIG_HOME is set but not absolute, OR if neither $XDG_CONFIG_HOME nor $HOME is defined. This is what Path() delegates to."
- url: https://pkg.go.dev/os#UserHomeDir
  why: "Confirms: returns $HOME on Unix; errors ('$HOME is not defined') if unset/empty. This is DefaultStore()'s fallback primitive (no os.UserDataDir exists)."
- url: https://specifications.freedesktop.org/basedir-spec/latest/
  why: "XDG Base Directory Specification: $XDG_CONFIG_HOME default ~/.config, $XDG_DATA_HOME default ~/.local/share, AND the rule that a set $XDG_*_HOME MUST be absolute (a relative value is invalid -> treat as unset). This is why DefaultStore guards with filepath.IsAbs."
```

### Current Codebase tree

```bash
$ cd /home/dustin/projects/skilldozer && ls internal/
check/  config/   discover/  resolve/  search/  skillsdir/  ui/
# internal/config/ is created by the PARALLEL sibling P1.M1.T1.S1 (status: Implementing).
# When S2 begins, config.go and config_test.go already exist with {File, Load, Save}.
$ grep -rn "SKILLDOZER_CONFIG\|os.UserConfigDir\|os.UserHomeDir\|os.UserDataDir\|XDG_DATA_HOME\|configPath" --include="*.go" .
# (no matches today — confirmed greenfield; G5 is real)
$ cat go.mod
module github.com/dabstractor/skilldozer
go 1.25
require gopkg.in/yaml.v3 v3.0.1   # the ONLY non-stdlib dep; S2 adds none
```

### Desired Codebase tree with files to be added and responsibility of file

```bash
internal/
├── config/
│   ├── config.go         # S1 creates (File/Load/Save) — S2 APPENDS: configEnv + Path() + DefaultStore() + doc-comment extension
│   └── config_test.go    # S1 creates (Load/Save tests) — S2 APPENDS: Path()/DefaultStore() branch tests
├── skillsdir/  ...        # exists — READ ONLY (the env-test pattern to mirror; findConfig wiring is T2.S2)
└── ...                    # other packages untouched
```

S2 creates **no new files** — it extends the two files S1 created. The package is small and cohesive (every symbol is about locating config/store paths), matching `internal/discover/discover.go` and `internal/skillsdir/skillsdir.go` (each a single source file).

| File | S2 responsibility | Owned by |
|---|---|---|
| `internal/config/config.go` | APPEND `configEnv` const + `Path()` + `DefaultStore()`; extend the package doc comment | contract LOGIC §3 + OUTPUT |
| `internal/config/config_test.go` | APPEND the Path/DefaultStore branch tests | contract OUTPUT §4 |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — os.UserConfigDir() DOES reject a relative $XDG_CONFIG_HOME on Linux.
// Empirically verified (research/verified_facts.md §1): setting XDG_CONFIG_HOME to a
// relative path makes os.UserConfigDir() return ("", error "path in $XDG_CONFIG_HOME
// is relative"). So Path() needs NO extra absoluteness check — it delegates to
// os.UserConfigDir() and returns ("", err) on any error. Do NOT re-validate. (The
// common misconception that "only Windows validates" is WRONG as of current Go.)
//
// GOTCHA #2 — There is NO os.UserDataDir(). The $XDG_DATA_HOME -> ~/.local/share rule
// must be computed by hand in DefaultStore(): os.Getenv("XDG_DATA_HOME") guarded by
// filepath.IsAbs, else os.UserHomeDir()+"/.local/share". This is exactly what
// external_deps.md §2 prescribes; do not go looking for a stdlib helper that doesn't exist.
//
// GOTCHA #3 — A relative $XDG_DATA_HOME is INVALID per the XDG spec and must be IGNORED
// (fall back to ~/.local/share), not used. The guard is `v != "" && filepath.IsAbs(v)`:
// empty/missing OR relative -> fallback. filepath.IsAbs("relative")==false and
// filepath.IsAbs("")==false (verified), so one expression handles all three.
//
// GOTCHA #4 — The SKILLDOZER_CONFIG override is returned filepath.Clean'd AS-IS, NOT
// joined with the config home. filepath.Clean("relative/cfg.yaml")=="relative/cfg.yaml"
// and filepath.Clean("/a/b/../c.yaml")=="/a/c.yaml" (lexical only, no symlink eval).
// This is the "useful for tests / multiple profiles" knob (PRD §8.1). Do NOT call
// filepath.Join(configHome, override) — that would break the relative-override use case.
//
// GOTCHA #5 — filepath.Clean("") == "." (verified). This is NEVER reached because Path
// guards with `v != ""` before calling Clean. But it means: never write
// `return filepath.Clean(os.Getenv(configEnv)), nil` unguarded — always behind the
// non-empty check. (An empty override must fall through to the XDG default, NOT return ".".)
//
// GOTCHA #6 — Empty SKILLDOZER_CONFIG is equivalent to unset. os.Getenv returns "" for
// both an unset var and a var set to "". Path checks `v != ""`, so t.Setenv(configEnv, "")
// in a test correctly simulates "unset" — you do NOT need an unsetEnvVar helper for
// SKILLDOZER_CONFIG (unlike skillsdir's envVar, which uses LookupEnv; here Getenv+"!=\"\""
// is the contract). Same for XDG_DATA_HOME in DefaultStore.
//
// GOTCHA #7 — filepath.Join always returns a Clean path. The default branch
// `filepath.Join(configHome, "skilldozer", "config.yaml")` is clean even if
// os.UserConfigDir() returned an unclean absolute value like "/x/../y" (Join collapses
// it to "/y/skilldozer/config.yaml"). No extra filepath.Clean on the default branch.
//
// GOTCHA #8 — Do NOT wrap the os.UserConfigDir / os.UserHomeDir errors. The contract
// says "on err return ('', err)". A fmt.Errorf("config home: %w", err) wrap is
// technically still errors.Is-able, but the contract wants the bare error returned.
// findConfig (T2.S2) treats ANY Path() error as "config unavailable -> fall through
// to rule 3"; it does not inspect the error type. Return it verbatim (matches S1's
// Load precedent and skillsdir's per-rule "never errors, returns found=false" style).
//
// GOTCHA #9 — Do NOT touch the filesystem. Path/DefaultStore read ONLY the environment
// (os.Getenv / os.UserConfigDir / os.UserHomeDir). They never os.Stat, os.ReadFile, or
// mkdir. File existence is a findConfig/init concern, not a path-resolution concern.
// (This is what makes them trivially unit-testable with t.Setenv and no temp files
// for the paths themselves — though tests still use t.TempDir() for controlled
// absolute XDG/HOME values.)
//
// GOTCHA #10 — No new dependency; no go mod tidy. os, path/filepath are stdlib; yaml.v3
// (S1's concern) is already pinned. go.mod/go.sum must be byte-for-byte unchanged.
// (Carries over from S1's GOTCHA #7.)
//
// GOTCHA #11 — Do NOT reuse skilldozer's SKILLDOZER_SKILLS_DIR constant here. The
// contract is explicit: "reuse skilldozer's existing SKILLDOZER_SKILLS_DIR name only
// in skillsdir, not here." config owns its OWN configEnv="SKILLDOZER_CONFIG" constant.
// The two env vars are distinct (one overrides the store DIR, the other the config FILE).
```

---

## Implementation Blueprint

### Data models and structure

**No new data models.** S2 adds only two pure functions and one string constant. It reuses S1's `File`/`Load`/`Save` unchanged (it does not even call them — `findConfig`/`init`, the consumers, compose `Path()`+`Load()`/`Path()`+`Save()` themselves).

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EXTEND internal/config/config.go (append configEnv + Path + DefaultStore)
  - PRECONDITION: S1 (P1.M1.T1.S1) is implemented — config.go exists with package
    doc comment, File, Load, Save. APPEND to it; do not rewrite S1's code.
  - IMPLEMENT: const configEnv = "SKILLDOZER_CONFIG"  (UNEXPORTED — see DESIGN DECISIONS §1;
    matches skillsdir.go's `const envVar = "SKILLDOZER_SKILLS_DIR"`)
  - IMPLEMENT: func Path() (string, error)
      if v := os.Getenv(configEnv); v != "" {
          return filepath.Clean(v), nil   // GOTCHA #4/#5: literal path AS-IS, cleaned, NOT joined
      }
      configHome, err := os.UserConfigDir()   // GOTCHA #1: rejects relative XDG_CONFIG_HOME
      if err != nil {
          return "", err                     // GOTCHA #8: verbatim, no wrap
      }
      return filepath.Join(configHome, "skilldozer", "config.yaml"), nil  // GOTCHA #7: Join cleans
  - IMPLEMENT: func DefaultStore() (string, error)
      if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {  // GOTCHA #2/#3
          return filepath.Join(v, "skilldozer", "skills"), nil
      }
      home, err := os.UserHomeDir()   // GOTCHA #2: no os.UserDataDir
      if err != nil {
          return "", err              // GOTCHA #8: verbatim
      }
      return filepath.Join(home, ".local", "share", "skilldozer", "skills"), nil
  - EXTEND the package doc comment (added by S1) with one short paragraph documenting
    Path/DefaultStore as the XDG-aware resolvers for the config-file path and default
    store dir. Do NOT contradict S1's "settings sidecar, NOT a catalog index" framing —
    Path/DefaultStore are about LOCATING config/store, squarely within that scope.
  - IMPORTS: ensure `os` and `path/filepath` are imported (S1 imports os, path/filepath,
    io/fs, gopkg.in/yaml.v3 for Load/Save). Add os/path/filepath ONLY if not already
    present; do not remove S1's imports. No new modules.
  - FOLLOW pattern: internal/skillsdir/skillsdir.go (env-var const + per-rule helper
    style; verbatim error returns; doc comments cite the PRD rule number)
  - NAMING: package config; exported Path/DefaultStore; unexported configEnv
  - PLACEMENT: append to internal/config/config.go (S1's file). Do NOT create paths.go.

Task 2: EXTEND internal/config/config_test.go (append the Path/DefaultStore tests)
  - PRECONDITION: S1's config_test.go exists (Load/Save tests). APPEND the new tests;
    do not modify S1's tests.
  - IMPLEMENT helper (if not already present from S1): a way to write nothing — these
    tests need NO temp FILES (Path/DefaultStore don't touch the FS), only t.TempDir()
    for controlled ABSOLUTE env values. Reuse S1's import block; add "os","path/filepath"
    if missing.
  - TEST TestPathSkilldozerConfigAbsoluteOverride:
      t.Setenv(configEnv, "/abs/path/to/cfg.yaml")
      t.Setenv("XDG_CONFIG_HOME", t.TempDir())   // prove override WINS over XDG
      got, err := Path()
      assert err==nil; assert got == filepath.Clean("/abs/path/to/cfg.yaml")
      (== "/abs/path/to/cfg.yaml")
  - TEST TestPathSkilldozerConfigRelativeOverrideNotJoined:   # THE critical no-join test
      t.Setenv(configEnv, "rel/sub/cfg.yaml")
      t.Setenv("XDG_CONFIG_HOME", t.TempDir())
      got, err := Path()
      assert err==nil
      assert got == "rel/sub/cfg.yaml"          # filepath.Clean of a relative path
      assert !filepath.IsAbs(got)               # proves it was NOT joined to configHome
      assert !strings.Contains(got, "skilldozer")  # proves no Join with the XDG default
  - TEST TestPathSkilldozerConfigEmptyFallsToXDG:
      t.Setenv(configEnv, "")                    # empty == unset (GOTCHA #6)
      xdg := t.TempDir()                         # controlled absolute config home
      t.Setenv("XDG_CONFIG_HOME", xdg)
      got, err := Path()
      assert err==nil; assert got == filepath.Join(xdg, "skilldozer", "config.yaml")
  - TEST TestPathRejectsRelativeXDGConfigHome:   # error path, empirically verified (§1)
      t.Setenv(configEnv, "")                    # ensure override does not short-circuit
      t.Setenv("XDG_CONFIG_HOME", "relative/not-abs")
      got, err := Path()
      assert err != nil; assert got == ""
      (do NOT assert the error message wording — stdlib-internal)
  - TEST TestDefaultStoreAbsoluteXDGDataHome:
      t.Setenv("XDG_DATA_HOME", "/abs/data")
      got, err := DefaultStore()
      assert err==nil; assert got == filepath.Join("/abs/data", "skilldozer", "skills")
  - TEST TestDefaultStoreEmptyXDGDataHomeFallsToHome:
      t.Setenv("XDG_DATA_HOME", "")
      home := t.TempDir(); t.Setenv("HOME", home)
      got, err := DefaultStore()
      assert err==nil
      assert got == filepath.Join(home, ".local", "share", "skilldozer", "skills")
  - TEST TestDefaultStoreRelativeXDGDataHomeIgnored:   # XDG-spec correctness (GOTCHA #3)
      t.Setenv("XDG_DATA_HOME", "relative/data")   # relative -> invalid -> ignored
      home := t.TempDir(); t.Setenv("HOME", home)
      got, err := DefaultStore()
      assert err==nil
      assert got == filepath.Join(home, ".local", "share", "skilldozer", "skills")  # fell back
  - TEST TestDefaultStoreHomeUnsetErrors:   # error path (Linux-specific; PRD targets Linux)
      t.Setenv("XDG_DATA_HOME", "")          # force the HOME fallback branch
      t.Setenv("HOME", "")                    # os.UserHomeDir -> error (verified §2)
      got, err := DefaultStore()
      assert err != nil; assert got == ""
      (do NOT call t.Parallel() — mutates HOME; comment this explicitly)
  - FOLLOW pattern: internal/skillsdir/skillsdir_test.go (t.Setenv, t.TempDir for
    controlled abs paths, direct got!=want assertions, no testify, behavioral names,
    explicit "no t.Parallel" on env tests)
  - IMPORTS: os, path/filepath, strings (for Contains in the no-join assertion), testing
  - NAMING: Test<Func><Branch> (mirrors skillsdir_test.go's TestFindEnv…)
  - COVERAGE: every contract OUTPUT §4 branch (override abs, override rel-no-join, XDG
    default, relative-XDG error; abs-XDG_DATA_HOME, empty->home, relative->ignored,
    home-unset error)
  - PLACEMENT: append to internal/config/config_test.go (S1's file). Internal package
    test (`package config`) so configEnv is in scope — matches S1's test-package choice.

Task 3: VERIFY the package in isolation, then the whole module
  - COMMAND: go build ./internal/config/...      (exit 0)
  - COMMAND: go test ./internal/config/... -v    (S1's tests + S2's 8 new tests all pass)
  - COMMAND: go vet ./...                        (clean)
  - COMMAND: go test ./...                       (whole module green — no regressions
                                                   in skillsdir/main/etc.)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"
    (MUST report "deps unchanged" — GOTCHA #10)
```

### Implementation Patterns & Key Details

```go
// Path — the config-file resolver (PRD §8.1). Pure function of the environment.
func Path() (string, error) {
	// Override: SKILLDOZER_CONFIG is the literal config-file path (abs or rel).
	// Cleaned lexically; NOT joined to the config home; NOT symlink-evaluated.
	// GOTCHA #5: the `v != ""` guard means filepath.Clean("") (==".") is never hit.
	if v := os.Getenv(configEnv); v != "" {
		return filepath.Clean(v), nil
	}
	// Default: os.UserConfigDir implements $XDG_CONFIG_HOME -> ~/.config for free
	// AND rejects a relative $XDG_CONFIG_HOME (GOTCHA #1). Delegate entirely.
	configHome, err := os.UserConfigDir()
	if err != nil {
		return "", err // GOTCHA #8: verbatim — findConfig treats any error as fall-through
	}
	// filepath.Join always returns a clean path (GOTCHA #7).
	return filepath.Join(configHome, "skilldozer", "config.yaml"), nil
}

// DefaultStore — the default skills store dir (PRD §8.2 / §8.3). Pure function of env.
func DefaultStore() (string, error) {
	// Honor XDG_DATA_HOME ONLY when absolute (XDG spec: a relative XDG_*_HOME is
	// invalid and must be ignored — GOTCHA #3). There is no os.UserDataDir (GOTCHA #2).
	if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, "skilldozer", "skills"), nil
	}
	// Fallback: ~/.local/share. os.UserHomeDir errors iff HOME is unset (GOTCHA #2).
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err // GOTCHA #8: verbatim
	}
	return filepath.Join(home, ".local", "share", "skilldozer", "skills"), nil
}
```

Notes that are easy to get wrong:
- `os.Getenv` returns `""` for both an unset var and a var set to `""`. Both `Path` and `DefaultStore` use `v != ""`, so `t.Setenv(var, "")` correctly simulates "unset" in tests — no `unsetEnvVar` helper needed for these two (unlike skillsdir's `findEnv`, which uses `os.LookupEnv` and distinguishes unset from empty; here the contract is `os.Getenv` + `!= ""`).
- Do **not** factor out a private `configHome()`/`dataHome()` helper unless it reads better — the contract specifies `Path`/`DefaultStore` as the exported API and the bodies are short enough to inline (8 lines each). Over-factoring adds surface the contract did not ask for.
- The `strings` import is needed **only** for the `TestPathSkilldozerConfigRelativeOverrideNotJoined` `strings.Contains(got, "skilldozer")` assertion. If you drop that assertion, drop the import. (Keeping it is recommended — it is the strongest proof the override was not joined.)

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **`configEnv` exported or unexported? → UNEXPORTED.** The contract LOGIC §3 writes it lowercase (`configEnv = "SKILLDOZER_CONFIG"`) and says "exported or package-internal as needed by main". No consumer needs the *symbol*: `findConfig` (T2.S2) calls `config.Path()`; `init` (M2.T2) calls `config.Path()`/`DefaultStore()`/`Save()`; `--path` prints the label `config file` (PRD §8.3), NOT `SKILLDOZER_CONFIG`. So the constant is an internal implementation detail of `Path()`. This matches `internal/skillsdir/skillsdir.go`'s precedent (`const envVar = "SKILLDOZER_SKILLS_DIR"`, unexported) and keeps the two packages symmetric. **Escape hatch:** if a later subtask needs the symbol (e.g. main_test.go wants to avoid hardcoding the literal), capitalize the `C` — a one-line change with no behavioral impact.

2. **New file or append to `config.go`? → APPEND.** S2 adds two short functions + one const to a package whose every symbol is about locating config/store paths. A single cohesive `config.go` matches `internal/discover/discover.go` and `internal/skillsdir/skillsdir.go` (each one source file). S1 is the dependency root (status "Implementing") and lands before S2, so `config.go` already exists when S2 begins — there is no write conflict. Do not create `paths.go`.

3. **Package doc comment update? → EXTEND, do not rewrite.** S1's doc comment frames the package as "the store-location settings sidecar ... NOT a catalog index." `Path`/`DefaultStore` resolve the *locations* of the config file and the default store — squarely within that framing. Add one short paragraph documenting the two new functions and the `SKILLDOZER_CONFIG` override; do not touch S1's "not a catalog index" sentence.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod UNCHANGED. os + path/filepath are stdlib; yaml.v3 (S1) already pinned.
    No go get, no go mod tidy. (GOTCHA #10)

CONSUMERS (NOT built in this subtask — listed to fix the interface):
  - skillsdir.findConfig (P1.M1.T2.S2):  p, err := config.Path()
        -> if err != nil { fall through to rule 3 (sibling) }   // any Path error = config unavailable
        -> f, err := config.Load(p)                              // S1's Load; ENOENT -> fall through
        -> if f.Store == "" { fall through }
        -> absolutize + os.Stat(f.Store); missing -> fall through; else (absDir, SourceConfig, true)
  - init choose-store (P1.M2.T2.S1):  def, err := config.DefaultStore()
        -> use def as the prompt default / --store fallback; on err, fail with a clear message
  - init write-config (P1.M2.T2.S2):  err := config.Save(config.Path(), config.File{Store: absStore})
        -> composes S2's Path() with S1's Save() (which MkdirAll's the parent)

NO ROUTES / NO DATABASE / NO CONFIG-FIELD-ADDITIONS:
  - S2 adds zero struct fields (Store is S1's). It adds two functions + one const.
  - S2 does NOT read or write the config file, does NOT os.Stat anything, does NOT
    touch skillsdir or main.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after appending to config.go)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l internal/config/         # must print NOTHING (run `gofmt -w` if it lists config.go)
go vet ./internal/config/...      # expect exit 0
go build ./internal/config/...    # expect exit 0
# Expected: zero output / exit 0.
```

### Level 2: Unit Tests (component validation — the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test ./internal/config/... -v
# Expected: S1's tests + S2's 8 new tests all pass. The 8 new tests map to contract OUTPUT §4:
#   TestPathSkilldozerConfigAbsoluteOverride        -> "Path honors SKILLDOZER_CONFIG override" (abs)
#   TestPathSkilldozerConfigRelativeOverrideNotJoined -> "Path honors override" (rel, NOT joined)
#   TestPathSkilldozerConfigEmptyFallsToXDG         -> "Path honors the XDG default"
#   TestPathRejectsRelativeXDGConfigHome            -> error path (os.UserConfigDir rejects rel)
#   TestDefaultStoreAbsoluteXDGDataHome             -> "DefaultStore honors an absolute XDG_DATA_HOME"
#   TestDefaultStoreEmptyXDGDataHomeFallsToHome     -> "falls back to ~/.local/share"
#   TestDefaultStoreRelativeXDGDataHomeIgnored      -> XDG-spec correctness (rel ignored)
#   TestDefaultStoreHomeUnsetErrors                 -> error path (HOME unset)

# Run just the S2 tests in isolation:
go test ./internal/config/... -run 'Path|DefaultStore' -v
# Expected: 8 passed.
```

### Level 3: Whole-module regression + dependency invariant

```bash
cd /home/dustin/projects/skilldozer

go build ./...  ; echo "build exit $?"
go vet  ./...   ; echo "vet exit $?"
go test ./...   ; echo "test exit $?"
# Expected: all exit 0. (S1's config tests, skillsdir, main, etc. all still green.)

# GOTCHA #10 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
# Expected: "deps unchanged".
```

### Level 4: Behavioral spot-checks (lock the hard claims against ground truth)

```bash
cd /home/dustin/projects/skilldozer

# 4a. Confirm the relative-XDG_CONFIG_HOME rejection and the HOME-unset error paths
# directly (re-runs the empirical probes from research/verified_facts.md §1/§2):
cat > /tmp/xdgprobe_test.go <<'EOF'
package config_test
import ("os"; "testing")
func TestProbeRelativeXDG(t *testing.T){
	os.Setenv("XDG_CONFIG_HOME","relative/x")
	_, err := os.UserConfigDir()
	if err == nil { t.Fatal("expected os.UserConfigDir to reject relative XDG_CONFIG_HOME") }
	t.Logf("UserConfigDir err (wording may vary): %v", err)
}
EOF
cp /tmp/xdgprobe_test.go internal/config/xdgprobe_test.go
go test ./internal/config/... -run TestProbeRelativeXDG -v
rm internal/config/xdgprobe_test.go   # remove the throwaway; keep only config_test.go
# Expected: PASS (logs the error). Only run if you skipped TestPathRejectsRelativeXDGConfigHome.

# 4b. Confirm the override is returned AS-IS (not joined) at runtime, end-to-end:
SKILLDOZER_CONFIG=rel/cfg.yaml go test ./internal/config/... -run TestPathSkilldozerConfigRelativeOverrideNotJoined -v
# Expected: PASS (the test sets its own env via t.Setenv, but this confirms the binary honors it).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` clean, `go vet ./internal/config/...` exit 0, `go build` exit 0
- [ ] Level 2 PASS — `go test ./internal/config/... -v` all pass; the 8 S2 tests pass by name
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0; `git diff go.mod go.sum` reports "deps unchanged"
- [ ] Level 4 PASS — the relative-`XDG_CONFIG_HOME` rejection spot-check passes (or `TestPathRejectsRelativeXDGConfigHome` covers it)

### Feature Validation
- [ ] Exported surface is exactly `{File, Load, Save, Path, DefaultStore}` (S1's three + S2's two)
- [ ] `configEnv` is the unexported const `"SKILLDOZER_CONFIG"`
- [ ] `Path()` honors a non-empty `SKILLDOZER_CONFIG` override as `filepath.Clean(v)` **without joining** (absolute AND relative both proven)
- [ ] `Path()` treats empty `SKILLDOZER_CONFIG` as unset → `os.UserConfigDir()` default
- [ ] `Path()` returns `("", err)` when `os.UserConfigDir()` errors (relative `XDG_CONFIG_HOME` or `HOME` unset), verbatim (no wrap)
- [ ] `DefaultStore()` honors an absolute `XDG_DATA_HOME`
- [ ] `DefaultStore()` ignores a relative/empty `XDG_DATA_HOME` → `~/.local/share` fallback (XDG-spec correctness)
- [ ] `DefaultStore()` returns `("", err)` when `os.UserHomeDir()` errors (`HOME` unset), verbatim
- [ ] Package doc comment extended to document `Path`/`DefaultStore` without contradicting S1's "settings sidecar, NOT a catalog index" framing

### Code Quality / Convention Validation
- [ ] Matches `internal/skillsdir/skillsdir.go`'s env-var-const + verbatim-error-return style
- [ ] Matches `internal/skillsdir/skillsdir_test.go`'s test style (`t.Setenv`, `t.TempDir`, direct assertions, no testify, no `t.Parallel` on env tests)
- [ ] No `fmt.Errorf` wrap on the `os.UserConfigDir`/`os.UserHomeDir` errors (contract: return verbatim)
- [ ] No new imports beyond stdlib (`os`, `path/filepath`, `strings` for one assertion); no `go get`; no `go mod tidy`
- [ ] Appended to S1's `config.go`/`config_test.go`; did not create new files; did not modify S1's `File`/`Load`/`Save`

### Scope Discipline
- [ ] Did NOT read/write the config file (that is S1's `Load`/`Save`, composed by consumers)
- [ ] Did NOT modify `internal/skillsdir` (the `findConfig` rule + `SourceConfig` are P1.M1.T2.S1/S2)
- [ ] Did NOT modify `main.go` / `init` (P1.M2.T2)
- [ ] Did NOT reuse the `SKILLDOZER_SKILLS_DIR` constant (that lives only in `skillsdir`; config owns its own `configEnv`)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't join the `SKILLDOZER_CONFIG` override to the config home.** `filepath.Join(configHome, override)` would break the relative-override use case (PRD §8.1 "useful for tests / multiple profiles"). Return `filepath.Clean(v)` AS-IS.
- ❌ **Don't re-validate `$XDG_CONFIG_HOME` absoluteness in `Path`.** `os.UserConfigDir()` already rejects a relative value on Linux (empirically verified — `research/verified_facts.md` §1). Delegating is correct; a second check is dead code.
- ❌ **Don't honor a relative `$XDG_DATA_HOME` in `DefaultStore`.** The XDG spec says a set `XDG_*_HOME` MUST be absolute; a relative value is invalid and the app must fall back to the default. The `filepath.IsAbs(v)` guard is load-bearing — dropping it would let a misconfigured `XDG_DATA_HOME=foo` silently produce a relative store path.
- ❌ **Don't wrap the `os.UserConfigDir`/`os.UserHomeDir` errors.** The contract says "on err return `('', err)`". `findConfig` treats ANY `Path()` error as "config unavailable → fall through"; it does not inspect the type. A `fmt.Errorf("…: %w", err)` wrap is unnecessary and diverges from the contract + S1's `Load` precedent.
- ❓ **Don't go looking for `os.UserDataDir()`.** It does not exist. The `~/.local/share` rule is computed by hand in `DefaultStore()` exactly as external_deps.md §2 prescribes.
- ❌ **Don't touch the filesystem.** `Path`/`DefaultStore` read ONLY the environment. No `os.Stat`, no `os.ReadFile`, no `MkdirAll`. Existence is a consumer concern (`findConfig`/`init`).
- ❌ **Don't reuse the `SKILLDOZER_SKILLS_DIR` constant.** The contract is explicit: that name lives only in `skillsdir`. `config` owns its own `configEnv = "SKILLDOZER_CONFIG"` (a different env var — one overrides the store DIR, the other the config FILE).
- ❌ **Don't assert the stdlib error *message wording* in tests.** `"path in $XDG_CONFIG_HOME is relative"` and `"$HOME is not defined"` are stdlib-internal and may shift across Go versions. Assert only `err != nil` + returned `""`.
- ❌ **Don't call `t.Parallel()` in any S2 test.** Every test mutates process env (`SKILLDOZER_CONFIG`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `HOME`) via `t.Setenv`, which Go forbids under `t.Parallel`. (Mirrors `skillsdir_test.go`'s explicit "do NOT call t.Parallel" comments.)
- ❌ **Don't `go get` anything or `go mod tidy`.** `os` + `path/filepath` are stdlib; yaml.v3 is already pinned. `go.mod`/`go.sum` must be unchanged.
- ❌ **Don't create `paths.go`/`paths_test.go`.** Append to S1's `config.go`/`config_test.go`. The package is small and cohesive in one file, matching `discover.go`/`skillsdir.go`.

---

## Confidence Score

**9.5/10** — Every branch of both functions is pinned to an empirically verified stdlib behavior (`research/verified_facts.md`): the relative-`XDG_CONFIG_HOME` rejection (the one fact I initially doubted and then confirmed), the `HOME=""` error path, `filepath.Clean`/`IsAbs`/`Join` on every input the tests use, and the absence of `os.UserDataDir`. The two functions are 8 lines each with no filesystem interaction, the test conventions are read directly from the in-repo `skillsdir_test.go`, and the consumer interfaces are fixed by the task tree. The 0.5 reservation is for the single documented judgment call — `configEnv` exported vs. unexported — which the PRP resolves toward unexported with a one-line escape hatch, but which a reviewer could legitimately prefer exported per the contract OUTPUT's "Exported … the SKILLDOZER_CONFIG constant name" phrasing. Either choice is behaviorally identical; only the symbol visibility differs.
