# Verified facts — P1.M2.T3.S2 (wire expandHome into resolveStore + integration test)

All facts below were confirmed by direct read/grep of the repo at PRP-write time.
Line numbers are CURRENT (the contract cites main.go:865/886 which are STALE — anchor by text).

## 1. expandHome ALREADY EXISTS (S1 landed/in-flight)

- `grep -n 'func expandHome' main.go` → `894:func expandHome(p string) string {`
- Body (main.go:894-907) matches `architecture/go_tilde_expansion.md` §"Recommended helper" verbatim:
  - `p == "~"` → `os.UserHomeDir()` (or p unchanged on error)
  - `strings.HasPrefix(p, "~/")` → `filepath.Join(home, p[2:])` (or p unchanged on error)
  - else → UNCHANGED (~user / ~foo / ~~/weird / "" / foo/bar / /abs)
- S1 also ships `TestExpandHome` + `TestExpandHomeNoHomeUnchanged` in main_test.go (after the
  TestChooseStore* family ~2428). **S2 does NOT touch those** (disjoint region).
- CONSEQUENCE for S2: the helper + its unit tests are DONE. S2 only (a) CALLS it and (b) adds an
  integration test. No helper authoring.

## 2. The wiring site — ONE source-agnostic line in resolveStore

Current resolveStore body (main.go:901-925), exact text at the edit site (lines 941-945):
```go
	store, err := chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(store)
	if err != nil {
		return "", fmt.Errorf("skilldozer init: absolutize store: %w", err)
	}
	return abs, nil
```
The fix inserts `store = expandHome(store)` (+ comment) between the `chooseStore` block and the
`filepath.Abs(store)` line. **ONE production line.**

WHY this single insertion covers ALL init path forms:
- `init <dir>` (positional, parseArgs main.go:300), `--store <dir>` (main.go:272), `--store=<dir>`
  (main.go:199) all set `c.initStore`, passed to `resolveStore(c.initStore)` (main.go:1027) as
  `haveStore`. `chooseStore` returns `haveStore` VERBATIM when non-empty (main.go:862 `return haveStore, nil`).
- The interactive typed path: `chooseStore` returns the prompt's trimmed answer verbatim
  (main.go:882 `return choice, nil`) — `stdinIsTerminal()` true, prompt fn over os.Stdin.
- ALL three sources converge into the single `store` local in resolveStore, RIGHT before the
  insertion point. So one `store = expandHome(store)` normalizes every source.
- Idempotent: expandHome is a no-op for already-absolute / tilde-free paths, so the existing
  absolute-store test (`TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`) stays green.

## 3. The integration test — MIRROR TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0

`main_test.go:2573` is the authoritative init-through-run() integration pattern. Its setup is
EXACTLY what the contract LOGIC §3 prescribes:
```go
parent := t.TempDir()
store := filepath.Join(parent, "newstore")      // <-- S2: replace with "~/sub" + HOME set
cfg := filepath.Join(t.TempDir(), "config.yaml")
t.Setenv("SKILLDOZER_CONFIG", cfg)              // redirect the config write to a temp file
t.Setenv("SKILLDOZER_SKILLS_DIR", "")           // ensure the config rule wins (env unset)
t.Chdir(t.TempDir())                            // escape the repo's walk-up rule (deterministic)

var out, errOut bytes.Buffer
code := run([]string{"init", "--store", store}, &out, &errOut)
...
f, err := configpkg.Load(cfg); f.Store == store  // config written with the (abs) store
out.String() == store+"\n"                       // §6.1: stdout = effective resolved store
```
S2 mirrors this 1:1 with TWO changes:
- `t.Setenv("HOME", home)` where `home := t.TempDir()` (a DISTINCT temp dir from cwd).
- `store := "~/sub"` (tilde-bearing) and assert the EXPANDED path `filepath.Join(home, "sub")`.

DISCRIMINATION: home (TempDir A) != cwd (TempDir C). If expandHome is WIRED → config.Store =
`A/sub`. If NOT wired → `filepath.Abs("~/sub")` = `C/~/sub` (cwd + literal ~). Since A != C, the
`f.Store == filepath.Join(home,"sub")` assertion FAILS on the un-wired code. The test cannot
pass on the bug. (Mirrors the Issue-1 "exact-equality not Contains" discipline at main_test.go:2606.)

## 4. Why the interactive typed-path is NOT separately integration-tested (and why that's OK)

- `stdinIsTerminal()` is a PLAIN FUNCTION (main.go:808 `func stdinIsTerminal() bool`), NOT a
  package var. Its doc comment (main.go:796-807) states: "It is a plain function (not a package
  var) because init's test seam is chooseStore's isTTY PARAMETER, not a global override." It reads
  `os.Stdin.Stat()` (main.go:809) — char-device check.
- `resolveStore` constructs its prompt reader from `bufio.NewReader(os.Stdin)` (main.go:933) and
  passes `stdinIsTerminal()` (main.go:941) DIRECTLY — NOT injected. `run()` (main.go:428) takes
  only (args, stdout, stderr) — no stdin seam.
- THEREFORE: driving the interactive typed-path through `run()` requires `stdinIsTerminal()`==true
  AND piped typed bytes on os.Stdin. Neither is injectable without a code change (turning
  stdinIsTerminal into a var), which is OUT OF SCOPE for S2 (S2 = wire + test, not refactor seams).
- This is FINE because of §2: the expandHome line is source-agnostic. Proving it on the `--store`
  path transitively proves it for `init <dir>`, `--store=<dir>`, AND the interactive path — they
  all flow through the SAME `store` local at the SAME insertion point.
- The interactive typed-path's UNIT behavior is already locked by: (a) S1's `TestExpandHome`/
  `TestExpandHomeNoHomeUnchanged` (the helper), and (b) `TestChooseStoreTTYTypedPathOverrides`
  (main_test.go:2395) — chooseStore returns a typed answer verbatim. The S2 integration test closes
  the one remaining gap: that the helper is actually CALLED on that return value.

## 5. Imports / deps UNCHANGED

- main.go import block: `bufio/fmt/io/os/path/filepath/strings` + internal pkgs — `expandHome` is
  already defined (S1) using os/strings/filepath; the wiring line uses only the EXISTING `store`
  local and `expandHome` (same package). NO import edit.
- main_test.go import block (lines 3-11): `bytes/errors/io/os/path/filepath/strings/testing` +
  `configpkg`. The S2 test uses bytes.Buffer, os.Stat, filepath.Join, t.Setenv/TempDir/Chdir, run,
  configpkg.Load — ALL already imported. NO import edit.
- go.mod / go.sum: module `github.com/dabstractor/skilldozer`, go 1.25, sole dep yaml.v3.
  UNCHANGED (`git diff --quiet go.mod go.sum` must hold).

## 6. Test placement — DISJOINT from S1

- S1 inserts `TestExpandHome` + `TestExpandHomeNoHomeUnchanged` after the TestChooseStore* family
  (~2428, after `TestChooseStorePropagatesPromptError` main_test.go:2421).
- S2 inserts `TestRunInitStoreTildeExpandsHome` after `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`
  (main_test.go:2573-2628), BEFORE `TestRunBareTagUnconfiguredNeverPrompts` (main_test.go:2630).
  Anchor by the unique comment `// TestRunBareTagUnconfiguredNeverPrompts — the load-bearing`.
- DISJOINT regions (2428 vs 2628); no text overlap; land in either order with S1.

## 7. Doc comment (Mode A) — the resolveStore doc @887

The doc comment (main.go:887-899) currently says chooseStore's choice is "ABSOLUTIZED via
filepath.Abs". The Mode-A edit adds: a leading "~"/"~/" is expanded to $HOME (expandHome, Issue 5)
BEFORE filepath.Abs, because filepath.Abs alone does not expand "~" (so `init ~/x` / `--store ~/x` /
`--store=~/x` / the interactive path would otherwise store `<cwd>/~/x`). The README sweep is the
final Mode-B task (P1.M3.T1) — NOT this subtask (decisions.md §D7).

## 8. Stale-line caveat (anchor by TEXT, not number)

- Contract cites `main.go:865` (resolveStore) and `main.go:886` (filepath.Abs). CURRENT lines:
  resolveStore @901, `filepath.Abs(store)` @945, `chooseStore(...)` @941, expandHome @894.
  (M1 + P1.M2.T2 shifted things; same note as S1's PRP.)
- ALL edit anchors in the PRP use unique TEXT (`abs, err := filepath.Abs(store)` + the preceding
  `chooseStore(...)` block; the doc-comment phrase "choice ABSOLUTIZED via filepath.Abs"), not line
  numbers, so they survive further shifts.
