# PRP — P1.M2.T3.S1: Add the `expandHome` helper (stdlib `os.UserHomeDir`) + table-driven tests (Issue 5)

> **Subtask:** A small, additive helper. `init ~/myskills` (and `--store ~/x`, the prompt) today stores the LITERAL string `<cwd>/~/myskills` and creates a dir literally named `~`, because `resolveStore` absolutizes via `filepath.Abs`, which does NOT expand `~` (Go has no stdlib tilde function — `os.Expand`/`os.ExpandEnv` handle only `$VAR`). This subtask adds the **pure-ish** normalizer `func expandHome(p string) string` to `main.go` (adjacent to `resolveStore`) plus 2 table-driven unit tests, reusing the `os.UserHomeDir` pattern already in `internal/config/config.go:154` (`DefaultStore`) so `~` semantics are consistent across the binary.
>
> **Scope (S1 ONLY):** add the helper + its doc comment + unit tests. It does NOT wire `expandHome` into `resolveStore` yet — that is **P1.M2.T3.S2** (the `store = expandHome(store)` before `filepath.Abs` at main.go:922 + an integration test through `run()`/`init`). Exposing `expandHome` as a package-level func (lowercase, `package main`) makes it directly unit-testable in-package, exactly as `chooseStore`/`parseArgs`/`resolveStore`/`setupStore` already are.
>
> **STATUS (verified at PRP-write time):** main.go import block + `chooseStore`@858 / `resolveStore` doc@887 / `resolveStore`@901 / `filepath.Abs(store)`@922 read in full; `grep -c 'os.UserHomeDir' main.go` → 0 (helper is genuinely new); `internal/config/config.go:152-159` (`DefaultStore` `os.UserHomeDir` precedent) + `config_test.go:274-279` (`TestDefaultStoreHomeUnsetErrors` — proves `t.Setenv("HOME","")`→`os.UserHomeDir` errors reliably on Linux) read in full; main_test.go `TestChooseStore*` family end (~2428) + import block read. The contract line cite `main.go:865` is STALE (M1 work + in-flight P1.M2.T2.S1 shifted lines) — this PRP anchors by the unique text `// resolveStore is the I/O-bearing wrapper`. The helper + tests are fixed verbatim by `architecture/go_tilde_expansion.md` (the authoritative research brief) and `architecture/decisions.md §D4`. The parallel sibling P1.M2.T2.S1 edits `parseArgs case "init":` (~277) + tests (~1396/1944) — DISJOINT regions in both files; no collision, land in either order.

---

## Goal

**Feature Goal**: Provide a pure-ish, package-level, stdlib-only `expandHome(p string) string` in `main.go` that expands a leading `~` or `~/` to the current user's home dir (via `os.UserHomeDir`), leaves `~user` / `~foo` / empty / relative / absolute paths UNCHANGED, and fails safe (returns p unchanged) when `$HOME` is unset — so P1.M2.T3.S2 can call it before `filepath.Abs` and `init ~/myskills` resolves to `$HOME/myskills` instead of `<cwd>/~/myskills`.

**Deliverable**: Two additive edits:
1. `main.go` — a new `func expandHome(p string) string` (+ its Mode-A doc comment) inserted immediately before the `// resolveStore is the I/O-bearing wrapper` doc comment (main.go:887), reusing the already-imported `os`/`strings`/`path/filepath`.
2. `main_test.go` — 2 new table-driven tests: `TestExpandHome` (the 10-row edge-case table, HOME set) and `TestExpandHomeNoHomeUnchanged` (3 inputs, HOME unset → unchanged).

**Success Definition**: `go build/vet/test ./...` all pass; `gofmt -l main.go main_test.go` empty; `go.mod`/`go.sum` byte-for-byte unchanged (no new imports); `expandHome("~/myskills")` with `HOME=/home/testuser` → `/home/testuser/myskills`; `expandHome("~")` → `/home/testuser`; `expandHome("~user")` → `"~user"` (unchanged); with `HOME=""`, `expandHome("~/x")` → `"~/x"` (unchanged). `resolveStore`'s body is NOT modified (that is S2).

---

## User Persona (if applicable)

**Target User**: A user who types `~/myskills` at the `init` prompt (or passes `init ~/x` / `--store ~/x`), and any tooling/acceptance suite asserting shell-like home expansion.

**Use Case**: `skilldozer init` → at the "Where should skilldozer keep your skills?" prompt the user types `~/myskills`, reasonably expecting home-dir expansion as in every shell.

**User Journey**: (today) user types `~/myskills` → config gets `store: /<cwd>/~/myskills`; a real dir named `~` is created (broken, surprising) → (after S1+S2) the value is normalized to `$HOME/myskills` BEFORE `filepath.Abs`, so the config gets the correct absolute home path and the right dir is created. **S1 ships the normalizer + locks its behavior in unit tests; S2 wires it in.**

**Pain Points Addressed**: the literal-`~` directory bug (bug_fixes_validation.md §ISSUE 5); the Go stdlib's missing tilde function; the inconsistency between `init`'s path handling and the `DefaultStore` home expansion already in the binary.

---

## Why

- **Closes bug_fixes_validation.md §ISSUE 5** (Minor): tilde is not expanded in `init`'s interactive prompt input. The fix is a normalizer that runs before `filepath.Abs`.
- **There is NO stdlib tilde function.** `os.Expand`/`os.ExpandEnv` expand only `$VAR`/`${VAR}`, NOT `~` (architecture/go_tilde_expansion.md §1). The canonical pattern is `os.UserHomeDir()` + `strings.HasPrefix(p, "~/")` + bare-`~` special case.
- **Reuses the `os.UserHomeDir` pattern already in the binary** (`internal/config/config.go:154`, `DefaultStore`) so `~` semantics are consistent across code paths (architecture/decisions.md §D4). No new package (one caller today), no new deps (PRD §17: stdlib-only besides yaml.v3).
- **Splits the work cleanly:** S1 ships a pure-ish, directly-testable helper + its unit tests (no I/O, no TTY, no fixtures); S2 does the one-line wiring + an integration test through `run()`/`init`. This PRP is S1 only.

---

## What

A new ~15-line package-level func + its doc comment + 2 table-driven tests. No signature changes, no config changes, no dispatch changes, no README change (Mode B sweep = P1.M3.T1), no `resolveStore` body change (S2).

### Success Criteria

- [ ] `func expandHome(p string) string` exists in `main.go` immediately before the `// resolveStore is the I/O-bearing wrapper` doc comment (anchored by that unique text; current line ~887).
- [ ] Behavior: `p=="~"` → `os.UserHomeDir()` (or p unchanged on error); `strings.HasPrefix(p,"~/")` → `filepath.Join(home, p[2:])` (or p unchanged on error); everything else (`~user`, `~foo`, `~foo/bar`, `~~/weird`, ``, `foo/bar`, `/abs/path`) → UNCHANGED. The guard is `HasPrefix(p, "~/")`, NOT `HasPrefix(p, "~")`.
- [ ] On `os.UserHomeDir` error (`$HOME` unset), the path is returned UNCHANGED (fail safe — never returns `""`, never crashes).
- [ ] The Mode-A doc comment covers the `~` / `~/` / `~user` / no-HOME cases AND the "MUST run before `filepath.Abs`" ordering note (verbatim from `architecture/go_tilde_expansion.md`).
- [ ] `TestExpandHome` (10-row table, `HOME=/home/testuser`) + `TestExpandHomeNoHomeUnchanged` (3 inputs, `HOME=""`) added to `main_test.go` after the `TestChooseStore*` family (~2428).
- [ ] `resolveStore`'s body is NOT modified (no `store = expandHome(store)` — that is P1.M2.T3.S2).
- [ ] `go test ./...` green; `go.mod`/`go.sum` unchanged; no new imports in either file.

---

## All Needed Context

### Context Completeness Check

**Pass.** The single edit site is pinned by the unique doc-comment text `// resolveStore is the I/O-bearing wrapper` (the only such line in main.go). The helper is fixed verbatim by `architecture/go_tilde_expansion.md` §"Recommended helper", and the 2 tests are fixed verbatim by its §"Recommended tests" — including the exact 10-row edge-case table and the no-HOME case set. Every input class in the table is traced to a contract LOGIC §3 branch in `research/verified_facts.md` §3. The no-HOME branch's reliability is NOT hypothetical — it is the SAME env condition the suite already exercises in `internal/config/config_test.go:274-279` (`TestDefaultStoreHomeUnsetErrors`, a green test depending on `t.Setenv("HOME","")`→`os.UserHomeDir` erroring). Imports are grep-confirmed already-present (`os`/`strings`/`path/filepath` in main.go; `testing` in main_test.go) — zero new imports. `grep -c 'os.UserHomeDir' main.go` → 0 proves the helper is genuinely new (no collision/rename). The `~/`→`home` trailing-slash-cleaning behavior is documented as a GOTCHA (§6 of verified_facts) so the implementer does not "fix" it. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative algorithm + edge-case table + verbatim helper + verbatim tests
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/go_tilde_expansion.md
  why: "THE research brief. §1 = no stdlib tilde fn (os.Expand/os.ExpandEnv = $VAR only).
        §2 = canonical pattern (os.UserHomeDir + HasPrefix(p,\"~/\") + bare ~). §3 = filepath.Abs
        does NOT expand ~ ⇒ expansion MUST run before it (the S2 ordering, locked here in the
        doc comment). §4 = the full edge-case table (~user/~foo/~~/weird/empty/rel/abs UNCHANGED;
        guard MUST be HasPrefix(p,\"~/\"), not HasPrefix(p,\"~\")). §5 = error handling
        (os.UserHomeDir error ⇒ return p unchanged; never \"\"). §Recommended helper = the
        EXACT func + doc comment to paste. §Recommended tests = the EXACT 10-row table + the
        no-HOME test. §Sources = the pkg.go.dev URLs (os.UserHomeDir / os.Expand / filepath.Abs /
        strings.HasPrefix) + the excluded non-stdlib libs (PRD §17)."
  critical: "The verbatim helper + verbatim tests in this file ARE the implementation — copy them.
             The ~/→home trailing-slash-cleaning (filepath.Join(home,\"\") cleans) is ACCEPTABLE,
             not a bug — do NOT replace filepath.Join with string concatenation."

# MUST READ — the placement decision (main.go adjacent to resolveStore, NOT a new package)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/decisions.md
  why: "§D4: keep expandHome in main.go adjacent to resolveStore (one caller; main.go already
        imports os/strings/filepath; a new internal/paths package for one 15-line fn is
        over-engineering — lift it only if a second caller appears). Reuse os.UserHomeDir
        (consistent with internal/config/config.go:154)."
  section: "D4."

# MUST READ — the authoritative bug writeup + repro (Issue 5)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "§ISSUE 5 is the repro: type ~/myskills at the prompt ⇒ store: /<cwd>/~/myskills, a real
        dir named ~ is created. Pins the root cause (filepath.Abs does not expand ~) and the
        prescribed fix (expand a leading ~/ and bare ~ to $HOME before filepath.Abs)."
  section: "ISSUE 5 (Minor)."

# MUST READ — the file under edit (the resolveStore region; anchor the insert by text)
- file: main.go
  why: "THE edit target. chooseStore @858 (closing brace ~885). The // resolveStore is the
        I/O-bearing wrapper doc comment @887 — INSERT expandHome (func + its doc comment)
        immediately BEFORE this line. resolveStore @901; its filepath.Abs(store) @922 — the
        S2 wiring site, DO NOT TOUCH in S1. Import block (bufio/fmt/io/os/path/filepath/strings
        + internal pkgs) — os/strings/path/filepath ALREADY present; NO import edit."
  pattern: "Package-level helper adjacent to its consumer, lowercase, package main — same shape
            as chooseStore/parseArgs/resolveStore/setupStore (all tested in-package). Error
            handling = swallow + return-input-unchanged (fail safe), DISTINCT from
            internal/config DefaultStore which PROPAGATES the os.UserHomeDir error."

# MUST READ — the test file under edit (mirror the table-driven + t.Setenv shapes exactly)
- file: main_test.go
  why: "THE other edit target + the test-pattern source. TestChooseStore* family ends at
        TestChooseStorePropagatesPromptError (~2428) — INSERT the 2 TestExpandHome* tests right
        after it (expandHome is a pure-ish store-path helper, same group as chooseStore).
        Import block: bytes/errors/io/os/path/filepath/strings/testing/configpkg — testing is
        ALREADY present; NO import edit. The t.Setenv(\"HOME\", ...) shape mirrors
        internal/config/config_test.go:242/259/275 (the cross-binary HOME-setenv precedent)."
  gotcha: "Do NOT call t.Parallel() — both tests mutate HOME via t.Setenv (t.Setenv is already
           non-parallel-safe; the config_test.go HOME tests have the same no-Parallel rule).
           The no-HOME test relies on t.Setenv(\"HOME\",\"\") → os.UserHomeDir erroring on Linux
           (PRD targets Linux) — already proven green by TestDefaultStoreHomeUnsetErrors."

# READ-ONLY — the os.UserHomeDir precedent (consistency target) + its no-HOME test (reliability proof)
- file: internal/config/config.go
  why: "DefaultStore @152-159: the EXISTING os.UserHomeDir usage this helper reuses for
        cross-binary ~ consistency (decisions.md §D4). Note the ASYMMETRY: DefaultStore
        PROPAGATES the error (return \"\", err); expandHome SWALLOWS it (return p unchanged) —
        deliberate (expandHome is a best-effort normalizer that must never crash or emit \"\")."
  section: "DefaultStore."
- file: internal/config/config_test.go
  why: "TestDefaultStoreHomeUnsetErrors @274-279: a GREEN test that sets HOME=\"\" and asserts
        os.UserHomeDir errors. PROVES the TestExpandHomeNoHomeUnchanged premise is real on this
        CI (not hypothetical). Also the t.Setenv(\"HOME\", <dir>) precedent @242/259 for the
        happy-path test."
  section: "TestDefaultStoreHomeUnsetErrors / TestDefaultStoreEmptyXDGDataHomeFallsToHome."

# READ-ONLY — the parallel sibling PRP (boundary: disjoint regions, no collision)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M2T2S1/PRP.md
  why: "Confirms P1.M2.T2.S1 (Issue 4) edits main.go parseArgs case \"init\": (~277-292) + its
        comment + 2 tests in main_test.go (~1396, ~1944). This subtask edits main.go ~886 (a NEW
        func before the resolveStore doc comment) + 2 NEW tests ~2428. DISJOINT regions in both
        files; no text-level overlap; land in either order."

# READ-ONLY — PRD (the §17 stdlib-only constraint + the §8.2 init path authority)
- file: PRD.md
  why: "READ-ONLY. §17 (stdlib-only besides yaml.v3 — excludes mitchellh/go-homedir, x/term,
        os/user). §8.2 (init path forms — the tilde-bearing input source). The bugfix PRD h3.4
        Issue 5 is the repro."
  section: "§8.2, §17 (and the bugfix PRD h3.4 Issue 5)."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/tasks.json
  why: "P1.M2.T3.S1's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. This PRP
        transcribes it; tasks.json wins on any conflict."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls main.go main_test.go internal/config/*.go go.mod
main.go        main_test.go   internal/config/config.go   internal/config/config_test.go   go.mod
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep)
$ grep -c 'os.UserHomeDir' main.go        # 0  — expandHome is genuinely new (no existing tilde logic)
$ grep -n '// resolveStore is the I/O-bearing wrapper' main.go   # the insert anchor (unique text)
```

### Desired Codebase tree with files to be changed

```bash
main.go        # ADD func expandHome(p string) string (+ Mode-A doc comment) immediately before the
               #        "// resolveStore is the I/O-bearing wrapper" doc comment (~887). NO import edit.
main_test.go   # ADD TestExpandHome (10-row table) + TestExpandHomeNoHomeUnchanged (3 inputs) after
               #        TestChooseStorePropagatesPromptError (~2428). NO import edit.
# go.mod / go.sum — UNCHANGED (os/strings/path/filepath already imported; helper is stdlib-only).
# resolveStore body — UNCHANGED in S1 (the wiring `store = expandHome(store)` is P1.M2.T3.S2).
```

| File | Change | Owner |
|---|---|---|
| `main.go` | NEW `expandHome` func + Mode-A doc comment, inserted before the `resolveStore` doc comment. No import edit. | Issue 5 contract + decisions.md §D4 + go_tilde_expansion.md |
| `main_test.go` | 2 NEW table-driven tests (`TestExpandHome`, `TestExpandHomeNoHomeUnchanged`) after the `TestChooseStore*` family. No import edit. | QA Issue 5 |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 (CRITICAL — the #1 one-pass stall) — the prefix guard MUST be
// strings.HasPrefix(p, "~/"), NOT strings.HasPrefix(p, "~"). The latter would falsely
// match "~user", "~foo", "~~/weird" and mis-expand them to <home>/user etc. The bare "~" is
// handled by its OWN `if p == "~"` branch BEFORE the prefix check. (go_tilde_expansion.md §4.)

// GOTCHA #2 — `~/` expands to `home` (NO trailing slash), because filepath.Join(home, "")
// calls filepath.Clean which strips a trailing slash. So expandHome("~/") == "/home/testuser",
// NOT "/home/testuser/". The test table row {"~/", "/home/testuser"} reflects this. This is
// ACCEPTABLE (a trailing slash is not semantically significant for a store dir; MkdirAll is
// slash-insensitive). Do NOT "fix" it with `home + p[1:]` string concat — that reintroduces
// ~user mis-expansion and skips canonicalization. filepath.Join is correct (matches DefaultStore).

// GOTCHA #3 — swallow the os.UserHomeDir error, return p UNCHANGED. Do NOT return "" on error
// (a later filepath.Abs("") yields cwd, masking the problem) and do NOT propagate the error
// (expandHome's signature is (string), not (string, error); it is a best-effort normalizer).
// This is DELIBERATELY ASYMMETRIC with internal/config.DefaultStore, which PROPAGATES the error
// (return "", err) — different role. go_tilde_expansion.md §5 + the os.UserHomeDir docs
// ("the caller may choose to ignore the error") authorize the fail-safe fallback.

// GOTCHA #4 — anchor the insert by the unique doc-comment TEXT `// resolveStore is the
// I/O-bearing wrapper`, NOT a line number. The contract cites main.go:865; the CURRENT line is
// ~887 (M1 work + the in-flight P1.M2.T2.S1 shifted things). Same for the test insert: anchor
// by the TestChooseStorePropagatesPromptError func name, not a number.

// GOTCHA #5 — do NOT modify resolveStore's body in S1. The `store = expandHome(store)` line +
// the before-filepath.Abs ordering is P1.M2.T3.S2 (with its own integration test through
// run()/init). S1 ships ONLY the helper + its unit tests. Touching the filepath.Abs(store)
// line at main.go:922 here would collide with S2's scope.

// GOTCHA #6 — both tests MUST NOT call t.Parallel(): they mutate HOME via t.Setenv (which is
// non-parallel-safe by design). The existing HOME-mutating tests in internal/config/config_test.go
// (TestDefaultStore* at 240/257/274) carry the same no-Parallel rule — mirror it.

// GOTCHA #7 — os.UserHomeDir on Linux reads $HOME (cgo-free, since Go 1.12). The no-HOME test
// premise (t.Setenv("HOME","") → os.UserHomeDir errors) is NOT hypothetical: it is the SAME
// condition the suite already exercises green in TestDefaultStoreHomeUnsetErrors
// (internal/config/config_test.go:274-279). PRD targets Linux.

// GOTCHA #8 — no deps/imports change in EITHER file. main.go already imports os/strings/
// path/filepath; main_test.go already imports testing. go.mod/go.sum must be byte-for-byte
// identical. Verify `git diff --quiet go.mod go.sum`. (decisions.md §D4.)

// GOTCHA #9 — expandHome is a package-level (lowercase) func in package main, so it is
// directly unit-testable from main_test.go (same package) WITHOUT being exported on the public
// surface — exactly how chooseStore/parseArgs/resolveStore/setupStore are tested today.
```

---

## Implementation Blueprint

### Data models and structure

**No data-model changes.** No new types, fields, methods, or signatures. `expandHome(p string) string` is a standalone, side-effect-free (env-only via `$HOME`) helper.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT main.go — add the expandHome helper + its Mode-A doc comment
  - FILE: main.go (insert immediately BEFORE the "// resolveStore is the I/O-bearing wrapper"
    doc comment, which currently sits at ~887, right after chooseStore's closing brace;
    anchor by that unique TEXT per GOTCHA #4)
  - INSERT (verbatim from architecture/go_tilde_expansion.md §"Recommended helper"; the doc
    comment is the contract DOCS Mode-A requirement covering ~ / ~/ / ~user / no-HOME + the
    before-filepath.Abs ordering note):
        // expandHome expands a leading "~" or "~/" to the current user's home directory
        // (os.UserHomeDir) so that `init ~/myskills` resolves to $HOME/myskills rather than
        // <cwd>/~/myskills. Only the CURRENT user's home is expanded ("~" and "~/..."); "~user"
        // / "~foo" are returned unchanged (other-user expansion needs cgo/os/user, out of scope).
        // filepath.Abs does NOT expand "~", so this MUST run before filepath.Abs (Issue 5).
        // If $HOME is unset, os.UserHomeDir returns an error and the path is returned unchanged
        // (fail safe — the docs say the caller may ignore the error).
        func expandHome(p string) string {
        	if p == "~" {
        		if home, err := os.UserHomeDir(); err == nil {
        			return home
        		}
        		return p
        	}
        	if strings.HasPrefix(p, "~/") {
        		if home, err := os.UserHomeDir(); err == nil {
        			return filepath.Join(home, p[2:])
        		}
        		return p
        	}
        	return p
        }
  - This is the ENTIRE production change. Verify the guard is HasPrefix(p, "~/") (GOTCHA #1),
    the ~/→home cleaning is left as-is (GOTCHA #2), and the error is swallowed (GOTCHA #3).

Task 2: EDIT main_test.go — add TestExpandHome (the 10-row edge-case table)
  - FILE: main_test.go (insert right after TestChooseStorePropagatesPromptError, the last of the
    TestChooseStore* store-path-helper family, ~2428; anchor by that func name per GOTCHA #4.
    Groups expandHome with its sibling pure-ish store-path helpers, NOT the I/O-bearing
    TestSetupStore* tests that follow.)
  - ADD (verbatim from architecture/go_tilde_expansion.md §"Recommended tests"; GOTCHA #6 no
    t.Parallel; the table locks EVERY input class from the contract LOGIC §3):
    // Issue 5 (P1.M2.T3.S1): expandHome expands a leading "~"/"~/" to $HOME (os.UserHomeDir)
    // and leaves "~user"/empty/relative/absolute unchanged. filepath.Abs does NOT expand "~",
    // so this runs before it (wired in P1.M2.T3.S2). ~/ cleans to home (filepath.Join strips the
    // trailing slash) — acceptable for a store dir.
    func TestExpandHome(t *testing.T) {
    	// Do NOT call t.Parallel() — mutates HOME.
    	t.Setenv("HOME", "/home/testuser")
    	for _, tc := range []struct{ in, want string }{
    		{"~/myskills", "/home/testuser/myskills"},
    		{"~/", "/home/testuser"},
    		{"~", "/home/testuser"},
    		{"~user", "~user"},
    		{"~foo", "~foo"},
    		{"~foo/bar", "~foo/bar"},
    		{"~~/weird", "~~/weird"},
    		{"", ""},
    		{"foo/bar", "foo/bar"},
    		{"/abs/path", "/abs/path"},
    	} {
    		if got := expandHome(tc.in); got != tc.want {
    			t.Errorf("expandHome(%q) = %q; want %q", tc.in, got, tc.want)
    		}
    	}
    }

Task 3: EDIT main_test.go — add TestExpandHomeNoHomeUnchanged (the no-HOME fail-safe)
  - FILE: main_test.go (insert immediately after TestExpandHome from Task 2)
  - ADD (verbatim from architecture/go_tilde_expansion.md §"Recommended tests"; GOTCHA #6/#7 —
    no t.Parallel; the HOME="" → os.UserHomeDir error premise is proven green by
    internal/config/config_test.go:274-279 TestDefaultStoreHomeUnsetErrors):
    // Issue 5 (P1.M2.T3.S1): with $HOME unset, os.UserHomeDir errors and expandHome returns the
    // input UNCHANGED (fail safe — never "", never crashes). The error is swallowed, NOT
    // propagated (deliberately asymmetric with internal/config.DefaultStore, which propagates).
    func TestExpandHomeNoHomeUnchanged(t *testing.T) {
    	// Do NOT call t.Parallel() — mutates HOME.
    	t.Setenv("HOME", "")
    	for _, in := range []string{"~/myskills", "~", "~/"} {
    		if got := expandHome(in); got != in {
    			t.Errorf("with no HOME, expandHome(%q) = %q; want unchanged", in, got)
    		}
    	}
    }

Task 4: VERIFY in isolation + whole module + invariants
  - gofmt -l main.go main_test.go     # MUST print nothing (run gofmt -w if it lists a file)
  - go vet ./...                      # exit 0
  - go build ./...                    # exit 0
  - go test -run 'TestExpandHome$|TestExpandHomeNoHomeUnchanged' -v ./...   # the 2 new tests pass
  - go test ./...                     # whole module green; zero regressions
  - git diff --quiet go.mod go.sum && echo deps unchanged   # GOTCHA #8
  - SCOPE-INVARIANT: resolveStore's body is byte-identical (S1 does not wire it in):
      git diff main.go | grep -E 'store = expandHome|filepath.Abs' && echo "FAIL: resolveStore touched" || echo "OK: resolveStore body unchanged"
  - REGRESSION (the chooseStore/init tests stay green — expandHome is purely additive):
      go test -run 'TestChooseStore|TestParseArgsInit|TestRunExclusivityInit' -v ./...
```

### Implementation Patterns & Key Details

```go
// The helper (Task 1) — package-level, lowercase, package main; verbatim from
// architecture/go_tilde_expansion.md. os.UserHomeDir error is SWALLOWED (return p), distinct
// from internal/config.DefaultStore which PROPAGATES it. Guard = HasPrefix(p, "~/") (not "~").
func expandHome(p string) string {
	if p == "~" { // bare-~ special case (its own branch, BEFORE the prefix check)
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p // fail safe
	}
	if strings.HasPrefix(p, "~/") { // ~/... → filepath.Join(home, rest); ~/ cleans to home
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
		return p // fail safe
	}
	return p // ~user / ~foo / ~~/weird / "" / foo/bar / /abs — UNCHANGED
}

// The consumer (P1.M2.T3.S2 — NOT this subtask) will be, in resolveStore (main.go:922):
//   store = expandHome(store)      // expand ~ BEFORE filepath.Abs (filepath.Abs does not)
//   abs, err := filepath.Abs(store)
// S1 ships ONLY the helper; do NOT add the store= line here.
```

Notes easy to get wrong:
- The guard is `HasPrefix(p, "~/")`, not `HasPrefix(p, "~")` — otherwise `~user`/`~foo` mis-expand (GOTCHA #1).
- `~/` → `home` (trailing slash cleaned by `filepath.Join`); the test row `{"~/", "/home/testuser"}` locks this; do not "fix" it with string concat (GOTCHA #2).
- The `os.UserHomeDir` error is swallowed (`return p`), never `return ""`, never propagated — deliberately asymmetric with `DefaultStore` (GOTCHA #3).
- Do not touch `resolveStore`'s body — that is S2 (GOTCHA #5).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Place in `main.go` adjacent to `resolveStore`, not a new `internal/paths` package** (architecture/decisions.md §D4). One caller (`resolveStore`, in S2); `main.go` already imports `os`/`strings`/`path/filepath`; a new package for one 15-line func is over-engineering. Lift to a package only if a second caller appears.
2. **Reuse `os.UserHomeDir`** (consistent with `internal/config/config.go:154`, `DefaultStore`) so `~` semantics match across the binary; cgo-free; PRD §17-compliant (excludes `mitchellh/go-homedir`, `x/term`, `os/user`).
3. **Swallow the `os.UserHomeDir` error (return p unchanged), do not propagate.** `expandHome` is a best-effort normalizer with a `(string)` signature; it must never crash or emit `""` (which would make `filepath.Abs("")` yield cwd). Deliberately asymmetric with `DefaultStore` (which propagates). Authorized by the `os.UserHomeDir` docs ("the caller may choose to ignore the error").
4. **`filepath.Join(home, p[2:])`, not string concat.** Matches `DefaultStore`; canonicalizes via `filepath.Clean` (so `~/` → `home`, acceptable); avoids `~user` mis-expansion (GOTCHA #2).
5. **Two table-driven tests, not a per-case func.** Mirrors `architecture/go_tilde_expansion.md` §"Recommended tests" and the codebase's `t.Setenv`-driven `TestDefaultStore*` style. The 10-row table locks every input class; the no-HOME test locks the fail-safe. `t.TempDir()` is NOT used for HOME — a fixed `/home/testuser` string is clearer for a pure string-concatenation assertion (and the no-HOME test uses `""`, not a temp dir).
6. **Test placement after the `TestChooseStore*` family (~2428).** `expandHome` is a pure-ish store-path helper (env-only via HOME), grouping with `chooseStore`, not the I/O-bearing `TestSetupStore*` tests.
7. **No README change here.** The contract DOCS assigns the README sweep to the final Mode B task (P1.M3.T1). This subtask's doc edit is the in-code `expandHome` doc comment only (Mode A, Task 1).

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod / go.sum UNCHANGED. No new imports in either file (os/strings/path/filepath already
    in main.go; testing already in main_test.go). (GOTCHA #8)

CALLER (deferred to P1.M2.T3.S2 — NOT this subtask):
  - resolveStore (main.go:901-926) will call `store = expandHome(store)` before
    `filepath.Abs(store)` at main.go:922. S1 ships ONLY the helper + unit tests; it does NOT
    add that line. (GOTCHA #5)

SURFACE:
  - expandHome is a package-level (lowercase) func in package main — directly unit-testable
    from main_test.go (same package) WITHOUT being exported. (GOTCHA #9, contract OUTPUT §4.)

DOCUMENTATION (Mode A only here):
  - The expandHome doc comment (Task 1) is the per-subtask Mode A doc edit. The README init
    section is swept by the final Mode B task (P1.M3.T1) — no doc file rides here beyond the
    in-code comment. (decisions.md §D7.)

PARALLEL SIBLING (no conflict):
  - P1.M2.T2.S1 (Issue 4) edits main.go parseArgs case "init": (~277-292) + 2 tests in
    main_test.go (~1396, ~1944). This subtask edits main.go ~886 (NEW func) + 2 NEW tests ~2428.
    DISJOINT regions in both files; no text-level overlap; land in either order.

NO ROUTES / NO DATABASE / NO CONFIG-FORMAT CHANGE / NO NEW FILES.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after the main.go edit)

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

go test -run 'TestExpandHome$|TestExpandHomeNoHomeUnchanged' -v ./...
# Expected: BOTH pass. The load-bearing rows:
#   ~/myskills -> /home/testuser/myskills ; ~ -> /home/testuser ; ~/ -> /home/testuser (cleaned)
#   ~user / ~foo / ~foo/bar / ~~/weird / "" / foo/bar / /abs/path -> UNCHANGED
#   (no HOME) ~/myskills / ~ / ~/ -> UNCHANGED (fail safe)

# Regression — the chooseStore/init tests stay green (expandHome is purely additive):
go test -run 'TestChooseStore|TestParseArgsInit|TestRunExclusivityInit|TestRunInit' -v ./...
# Expected: PASS (no behavior changed; only a new func + 2 new tests were added).
```

### Level 3: Whole-module regression + scope/disjointness invariants

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # 0
go vet  ./...  ; echo "vet exit $?"     # 0
go test ./...  ; echo "test exit $?"    # 0  — CRITICAL: zero regressions (purely additive)

# GOTCHA #8 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"

# SCOPE invariant (GOTCHA #5): resolveStore's body is NOT modified in S1 — the wiring is S2.
# Assert neither the new `store = expandHome(store)` line nor a changed filepath.Abs appears
# in the diff (S1 must leave resolveStore byte-identical):
git diff main.go | grep -E '^\+.*store = expandHome|^\+.*filepath\.Abs' \
  && echo "FAIL: resolveStore body touched (that is P1.M2.T3.S2)" \
  || echo "OK: resolveStore body unchanged (S1 = helper + tests only)"

# Disjointness invariant (P1.M2.T2.S1 edits case "init":; this edits expandHome only):
grep -c 'func expandHome' main.go            # expect 1 (the new helper)
grep -c 'next == "init"' main.go             # expect 0 here (P1.M2.T2.S1's, may be 1 once it lands)

# Expected: "deps unchanged"; "OK: resolveStore body unchanged"; expandHome present exactly once.
```

### Level 4: Behavioral spot-checks (lock the helper's contract directly)

```bash
cd /home/dustin/projects/skilldozer

# 4a. The exact contract LOGIC §3 behavior, asserted via a throwaway test program (no TTY/fixture):
cat > /tmp/eh_test.go <<'EOF'
package main
import ("fmt"; "os")
func main() {
    os.Setenv("HOME", "/home/testuser")
    for _, c := range [][2]string{
        {"~/myskills","/home/testuser/myskills"},{"~/","/home/testuser"},{"~","/home/testuser"},
        {"~user","~user"},{"~foo","~foo"},{"~foo/bar","~foo/bar"},{"~~/weird","~~/weird"},
        {"",""},{"foo/bar","foo/bar"},{"/abs/path","/abs/path"},
    } {
        got := expandHome(c[0])
        mark := "ok"; if got != c[1] { mark = "FAIL" }
        fmt.Printf("%s expandHome(%q)=%q want %q\n", mark, c[0], got, c[1])
    }
    os.Setenv("HOME", "")
    for _, in := range []string{"~/myskills","~","~/"} {
        got := expandHome(in)
        mark := "ok"; if got != in { mark = "FAIL" }
        fmt.Printf("%s no-HOME expandHome(%q)=%q want %q\n", mark, in, got, in)
    }
}
EOF
go run main.go /tmp/eh_test.go 2>/dev/null || go test -run TestExpandHome -v ./...   # the canonical check is the unit test
rm -f /tmp/eh_test.go
# Expected: every row "ok" (or, equivalently, the unit tests in Level 2 pass).

# 4b. Confirm filepath.Abs STILL does not expand ~ (the S2 rationale, locked here as context):
go run -exec sh <<'EOF' 2>/dev/null || true   # (informational; the unit tests are authoritative)
EOF
# The filepath.Abs-stays-literal fact is documented in the expandHome doc comment (Task 1) and
# proven by the existing bug (bug_fixes_validation.md §ISSUE 5). No separate assertion needed in S1.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l main.go main_test.go` empty; `go vet ./...` exit 0; `go build` exit 0
- [ ] Level 2 PASS — `TestExpandHome` + `TestExpandHomeNoHomeUnchanged` pass; the `TestChooseStore*`/`TestParseArgsInit*`/`TestRunExclusivityInit*` regression tests stay green
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (zero regressions); `git diff go.mod go.sum` → "deps unchanged"; `resolveStore` body unchanged (S1 = helper + tests only)
- [ ] Level 4 PASS — every edge-case row is "ok" (the contract LOGIC §3 table holds)

### Feature Validation
- [ ] `func expandHome(p string) string` exists in `main.go` before the `// resolveStore is the I/O-bearing wrapper` doc comment
- [ ] `expandHome("~/myskills")` (HOME=/home/testuser) → `/home/testuser/myskills`
- [ ] `expandHome("~")` → `/home/testuser`; `expandHome("~/")` → `/home/testuser` (trailing slash cleaned)
- [ ] `expandHome("~user")`, `expandHome("~foo")`, `expandHome("~foo/bar")`, `expandHome("~~/weird")`, `expandHome("")`, `expandHome("foo/bar")`, `expandHome("/abs/path")` → all UNCHANGED
- [ ] With `HOME=""`: `expandHome("~/myskills")`, `expandHome("~")`, `expandHome("~/")` → all UNCHANGED (fail safe; never `""`)
- [ ] The Mode-A doc comment covers `~` / `~/` / `~user` / no-HOME + the before-`filepath.Abs` ordering note
- [ ] `resolveStore`'s body is NOT modified (the wiring is P1.M2.T3.S2)

### Code Quality / Convention Validation
- [ ] `expandHome` is a package-level (lowercase) func in `package main` (directly unit-testable, not exported — matches `chooseStore`/`parseArgs`/`resolveStore`/`setupStore`)
- [ ] Tests are table-driven with `t.Setenv("HOME", …)`, mirroring `internal/config/config_test.go`'s `TestDefaultStore*` shape; no `t.Parallel()` (HOME mutation)
- [ ] Tests placed after the `TestChooseStore*` family (the pure-ish store-path-helper group)
- [ ] No new imports; no new deps; `go.mod`/`go.sum` byte-for-byte identical
- [ ] No new files; both edits to `main.go` + `main_test.go`

### Scope Discipline
- [ ] Did NOT widen the guard to `HasPrefix(p, "~")` (used `HasPrefix(p, "~/")` + a bare-`~` branch — GOTCHA #1)
- [ ] Did NOT replace `filepath.Join` with string concat (left the `~/`→`home` cleaning as-is — GOTCHA #2)
- [ ] Did NOT propagate the `os.UserHomeDir` error or return `""` (swallowed it, returned p — GOTCHA #3)
- [ ] Did NOT wire `expandHome` into `resolveStore` (left the `filepath.Abs(store)` line untouched — that is P1.M2.T3.S2, GOTCHA #5)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`; no README change (Mode B = P1.M3.T1)

---

## Anti-Patterns to Avoid

- ❌ **Don't use `HasPrefix(p, "~")` as the guard.** It falsely matches `~user`/`~foo`/`~~/weird`. Use `HasPrefix(p, "~/")` for the prefix case and a dedicated `if p == "~"` for the bare case. (GOTCHA #1.)
- ❌ **Don't "fix" `~/` → `home` (no trailing slash) with string concat (`home + p[1:]`).** `filepath.Join(home, p[2:])` is correct: it canonicalizes (matches `DefaultStore`) and the trailing-slash stripping is acceptable for a store dir. String concat reintroduces `~user` mis-expansion and skips `filepath.Clean`. (GOTCHA #2.)
- ❌ **Don't propagate the `os.UserHomeDir` error or return `""`.** `expandHome` is a `(string)`-returning best-effort normalizer; on error it returns p unchanged (fail safe). Returning `""` would make a later `filepath.Abs("")` yield cwd, masking the problem. This is deliberately asymmetric with `DefaultStore`. (GOTCHA #3.)
- ❌ **Don't anchor by line number.** The contract's `main.go:865` is stale (current ~887). Match the unique text `// resolveStore is the I/O-bearing wrapper`. (GOTCHA #4.)
- ❌ **Don't wire `expandHome` into `resolveStore` here.** The `store = expandHome(store)` line + the before-`filepath.Abs` ordering + the integration test are P1.M2.T3.S2. S1 ships ONLY the helper + its unit tests. (GOTCHA #5.)
- ❌ **Don't call `t.Parallel()` in the tests.** Both mutate `HOME` via `t.Setenv` (non-parallel-safe). The existing `TestDefaultStore*` HOME tests carry the same rule. (GOTCHA #6.)
- ❌ **Don't add deps/imports, a new package, or a README change.** `os`/`strings`/`path/filepath` are already imported; `internal/paths` is over-engineering for one caller (decisions.md §D4); the README sweep is Mode B (P1.M3.T1). (GOTCHA #8.)

---

## Confidence Score

**9.5/10** — This is a ~15-line additive helper + its doc comment + 2 table-driven tests. Both the helper and the tests are fixed **verbatim** by `architecture/go_tilde_expansion.md` (the authoritative research brief), including the exact 10-row edge-case table and the no-HOME case set. The edit site is pinned by unique text (`// resolveStore is the I/O-bearing wrapper`); `grep -c 'os.UserHomeDir' main.go` → 0 proves the helper is genuinely new (no collision). Every needed import (`os`/`strings`/`path/filepath` in main.go; `testing` in main_test.go) is grep-confirmed already-present — zero new imports, `go.mod`/`go.sum` unchanged. The no-HOME fail-safe branch's reliability is NOT hypothetical: it is the SAME env condition the suite already exercises green in `TestDefaultStoreHomeUnsetErrors` (internal/config/config_test.go:274-279). The helper reuses the `os.UserHomeDir` pattern already in `DefaultStore` for cross-binary consistency (decisions.md §D4). The 0.5 reservation is for the single most-likely one-pass stall — using `HasPrefix(p, "~")` instead of `HasPrefix(p, "~/")` (which would mis-expand `~user`) — which the dedicated bare-`~` branch + the GOTCHA #1 note + the `~user`/`~foo`/`~foo/bar`/`~~/weird` table rows (all asserting UNCHANGED) jointly guard against.
