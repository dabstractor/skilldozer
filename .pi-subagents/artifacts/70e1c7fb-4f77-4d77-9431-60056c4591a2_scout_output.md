# Code Context — test-breakage surface for the findConfig + ErrNotFound-message change

Repo: `/home/dustin/projects/skilldozer`. Read-only recon. All refs verified by reading the actual files.

## TL;DR (what the change touches and what breaks)

- **(A) Add `findConfig()` rule** → `Find()` ordering becomes env → config → sibling → walk-up. The enum (`SourceConfig`) and its label (`"config file"`) **already exist** in `skillsdir.go` (lines 35-36, 49-50) and are already covered by `TestSourceString` (`skillsdir_test.go:29-46`). Only the `findConfig()` function and its wiring into `Find()` are missing.
- **(B) Flip `ErrNotFound` message** at `skillsdir.go:234` from the long message to `"skilldozer is not configured; run \`skilldozer init\`"`.
- **7 tests assert OLD-message substrings** and **must be updated** (the new message contains NONE of `SKILLDOZER_SKILLS_DIR` / `cd` / `reinstall`).
- **7 tests become NON-HERMETIC** after findConfig is added: they neutralize only `SKILLDOZER_SKILLS_DIR` and chdir into a temp dir, but do NOT neutralize `SKILLDOZER_CONFIG` / `XDG_CONFIG_HOME`, so on a host with a real `~/.config/skilldozer/config.yaml` whose `store` exists, findConfig hits and `Find()` no longer returns `ErrNotFound`. (Same 7 tests for both issues, because they are all the "all-rules-miss → ErrNotFound" tests.)
- **No non-test Go caller of `Find()` / `ErrNotFound` other than `main.go`**; all `internal/*` hits are doc comments only. Behavior-neutral for the message flip.
- **`main.go` needs zero changes**: every `Find()` error site prints `fmt.Fprintln(stderr, err)` verbatim (no prefix), and `--path` renders the source label via `fmt.Fprintf(stderr, "(found via %s)\n", src)` where `src` is a `skillsdir.Source` whose `String()` already returns `"config file"` for `SourceConfig`.

---

## 1. Tests asserting the OLD `ErrNotFound` message / substrings (must change)

New message target: `"skilldozer is not configured; run \`skilldozer init\`"`. It contains `run` and `skilldozer init`; it does NOT contain `SKILLDOZER_SKILLS_DIR`, `cd`, or `reinstall`.

| # | Test (file:line) | Exact assertion | New-message substring that satisfies it |
|---|---|---|---|
| 1 | `TestErrNotFoundMessageHasFix` — `internal/skillsdir/skillsdir_test.go:508-514` (assertion at `:510`) | `for _, want := range []string{"SKILLDOZER_SKILLS_DIR", "cd", "reinstall"} { if !strings.Contains(msg, want) { ... } }` on `ErrNotFound.Error()` | Replace the slice with `[]string{"not configured", "run", "skilldozer init"}` (or similar). All 3 old substrings are absent in the new message. |
| 2 | `TestRunPathFailureErrNotFound` — `main_test.go:228-243` (assertion at `:240`) | `for _, want := range []string{"SKILLDOZER_SKILLS_DIR", "cd", "reinstall"} { if !strings.Contains(msg, want) { ... } }` on `errOut.String()` | Same — replace slice; e.g. `[]string{"skilldozer init", "run"}`. |
| 3 | `TestRunListSkillsDirUnresolvableExit1` — `main_test.go:368-381` (assertion at `:379`) | `if !strings.Contains(errOut.String(), "SKILLDOZER_SKILLS_DIR") { ... }` | Change to assert a substring present in the new message, e.g. `"skilldozer init"`. |
| 4 | `TestRunTagSkillsDirUnresolvable` — `main_test.go:582-595` (assertion at `:593`) | `if !strings.Contains(errOut.String(), "SKILLDOZER_SKILLS_DIR") { ... }` | Change to `"skilldozer init"`. |
| 5 | `TestRunAllSkillsDirUnresolvable` — `main_test.go:840-854` (assertion at `:851`) | `if !strings.Contains(errOut.String(), "SKILLDOZER_SKILLS_DIR") { ... }` | Change to `"skilldozer init"`. |
| 6 | `TestRunSearchSkillsDirUnresolvable` — `main_test.go:1080-1093` (assertion at `:1091`) | `if !strings.Contains(errOut.String(), "SKILLDOZER_SKILLS_DIR") { ... }` | Change to `"skilldozer init"`. |
| 7 | `TestRunCheckSkillsDirUnresolvable` — `main_test.go:1258-1272` (assertion at `:1269`) | `if !strings.Contains(errOut.String(), "SKILLDOZER_SKILLS_DIR") { ... }` | Change to `"skilldozer init"`. |

### DO NOT touch (look-alike assertions that are NOT the ErrNotFound message)
These assert the **`--path` source label** (`SourceEnv.String()` → `"SKILLDOZER_SKILLS_DIR"`), which is unrelated to the message flip and unaffected by it:
- `main_test.go:182, 198, 219` — `if got, want := errOut.String(), "(found via SKILLDOZER_SKILLS_DIR)\n"; ...` inside `TestRunPathSuccess` / `TestRunPathShortFlag` / `TestRunPathReportsSourceLabel`.
- `internal/skillsdir/skillsdir_test.go:34` — `{SourceEnv, "SKILLDOZER_SKILLS_DIR"}` inside `TestSourceString` (and `:35` `{SourceConfig, "config file"}` — already correct).

---

## 2. Tests that rely on "all rules miss → ErrNotFound" but neutralize ONLY `SKILLDOZER_SKILLS_DIR` (non-hermetic after findConfig)

`findConfig` reads `SKILLDOZER_CONFIG` env (or the default `$XDG_CONFIG_HOME/skilldozer/config.yaml` → `~/.config/skilldozer/config.yaml`) via `config.Path()`, loads via `config.Load()`, and returns `store` if it exists. None of these tests neutralize `SKILLDOZER_CONFIG` or `XDG_CONFIG_HOME`, so on a machine with a real config whose `store` dir exists, `findConfig` hits and `Find()` succeeds → these assertions break.

| # | Test (file:line) | Neutralizes | Why non-hermetic |
|---|---|---|---|
| 1 | `TestFindAllMissReturnsErrNotFound` — `internal/skillsdir/skillsdir_test.go:495-504` | `unsetEnvVar(t)` (only `SKILLDOZER_SKILLS_DIR`) + `t.Chdir(t.TempDir())`; asserts `errors.Is(err, ErrNotFound)` | No `SKILLDOZER_CONFIG`/`XDG_CONFIG_HOME` neutralization → real config makes `findConfig` win. |
| 2 | `TestRunPathFailureErrNotFound` — `main_test.go:228-243` | `unsetSkillsEnv(t)` + `t.Chdir(t.TempDir())`; asserts exit 1 + old-message substrings | Same. |
| 3 | `TestRunListSkillsDirUnresolvableExit1` — `main_test.go:368-381` | `unsetSkillsEnv(t)` + `t.Chdir(t.TempDir())` | Same. |
| 4 | `TestRunTagSkillsDirUnresolvable` — `main_test.go:582-595` | `unsetSkillsEnv(t)` + `t.Chdir(t.TempDir())` | Same. |
| 5 | `TestRunAllSkillsDirUnresolvable` — `main_test.go:840-854` | `unsetSkillsEnv(t)` + `t.Chdir(t.TempDir())` | Same. |
| 6 | `TestRunSearchSkillsDirUnresolvable` — `main_test.go:1080-1093` | `unsetSkillsEnv(t)` + `t.Chdir(t.TempDir())` | Same. |
| 7 | `TestRunCheckSkillsDirUnresolvable` — `main_test.go:1258-1272` | `unsetSkillsEnv(t)` + `t.Chdir(t.TempDir())` | Same. |

**Recommended fix (out of scope for recon, but noted):** neutralize config too — e.g. `t.Setenv("SKILLDOZER_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))` and/or `t.Setenv("XDG_CONFIG_HOME", t.TempDir())` — so findConfig deterministically misses. (`config.Load` returns `fs.ErrNotExist` verbatim per `config.go:53`, which findConfig must treat as fall-through.)

---

## 3. Helper definitions — do they neutralize `SKILLDOZER_CONFIG`?

**No. Both neutralize ONLY `SKILLDOZER_SKILLS_DIR`.**

### `unsetEnvVar` — `internal/skillsdir/skillsdir_test.go:18-39`
```go
// unsetEnvVar removes envVar for the duration of the test and restores the
// prior state on cleanup. Needed because t.Setenv can only set, never unset.
// Do NOT call t.Parallel() in any test that uses this or t.Setenv.
func unsetEnvVar(tb testing.TB) {
	tb.Helper()
	prev, had := os.LookupEnv(envVar)        // envVar == "SKILLDOZER_SKILLS_DIR" (skillsdir.go:63)
	if err := os.Unsetenv(envVar); err != nil {
		tb.Fatalf("unsetenv %s: %v", envVar, err)
	}
	tb.Cleanup(func() {
		if had {
			_ = os.Setenv(envVar, prev)
		} else {
			_ = os.Unsetenv(envVar)
		}
	})
}
```
Neutralizes **only** `envVar` (`SKILLDOZER_SKILLS_DIR`). No reference to `SKILLDOZER_CONFIG`.

### `unsetSkillsEnv` — `main_test.go:22-28`
```go
// unsetSkillsEnv removes SKILLDOZER_SKILLS_DIR for the test and restores it on cleanup.
// (Mirrors internal/skillsdir/skillsdir_test.go's unsetEnvVar helper.) Forbids
// t.Parallel via t.Setenv.
func unsetSkillsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SKILLDOZER_SKILLS_DIR", "")
}
```
Neutralizes **only** `SKILLDOZER_SKILLS_DIR` (sets it to `""`). No reference to `SKILLDOZER_CONFIG` or `XDG_CONFIG_HOME`.

> Note: `unsetSkillsEnv` sets the var to `""` rather than unsetting it. For findEnv this is equivalent (empty ⇒ fall through, `skillsdir.go:73`), but it does not help with the config rule.

---

## 4. Non-test Go callers of `skillsdir.Find()` / `skillsdir.ErrNotFound` beyond `main.go`

**None.** Only `main.go` calls `skillsdir.Find()` / uses `skillsdir.ErrNotFound`. Every `internal/*` reference is a doc comment:

- `internal/resolve/resolve.go:50` — comment: "mirroring `skillsdir.Source.String()`".
- `internal/resolve/resolve.go:81` — comment: "matching the `skillsdir.ErrNotFound` convention".
- `internal/discover/index.go:32` — comment: "input (from `skillsdir.Find`) Abs is a no-op Clean".
- `internal/check/` — no reference.
- `internal/search/` — no reference.
- `internal/ui/` — no reference.
- `internal/config/config.go:4,26,44,53,121` — comments referencing the planned `findConfig` consumer.

**Conclusion for criterion-4:** the message flip is behavior-neutral for all internal packages — they never read `ErrNotFound.Error()`. No change needed anywhere except `skillsdir.go` + the 7 tests.

---

## 5. `main.go` printing of `ErrNotFound` and the `--path` source label

**Confirmed: zero `main.go` change required for both (A) and (B).**

### (B) ErrNotFound printed verbatim, no prefix, at every `Find()` error site
All six `skillsdir.Find()` error branches print the error with `fmt.Fprintln(stderr, err)` — no `"skilldozer: "` prefix, no wrap — so the new message renders verbatim:
- `main.go:413` (`--path`)
- `main.go:433` (`--list`)
- `main.go:469` (`--search`)
- `main.go:509` (`check`)
- `main.go:550` (`--all`)
- `main.go:581` (tag-resolution)

(The `// e.g. skills dir vanished between Find and Index` prints at `:438, :474, :514, :555, :586` are the `discover.Index` error path, not `ErrNotFound`, and are unaffected.)

### (A) `--path` source label renders the `"config file"` label with no main.go change
`main.go:423`:
```go
fmt.Fprintf(stderr, "(found via %s)\n", src)
```
`src` is a `skillsdir.Source`; `%s` invokes `Source.String()` (`skillsdir.go:44-55`), which returns `"config file"` for `SourceConfig` (`skillsdir.go:49-50`). So when `findConfig` wins, stderr shows `(found via config file)` automatically — no `main.go` edit.

---

## Key types / functions (for the implementing agent)

- `skillsdir.go:234` — `var ErrNotFound = errors.New(...)` ← message flip target (B).
- `skillsdir.go:247-258` — `func Find()` ← insert `findConfig()` call between `findEnv` (line 248) and `findSibling` (line 250) (A). New ordering: `findEnv` → `findConfig` → `findSibling` → `findWalkUp` → `ErrNotFound`.
- `skillsdir.go:35-36` / `:49-50` — `SourceConfig` enum + `"config file"` label (already present; do not re-add).
- `skillsdir.go:63` — `const envVar = "SKILLDOZER_SKILLS_DIR"`.
- `config/config.go:100` — `const configEnv = "SKILLDOZER_CONFIG"`.
- `config/config.go:131-142` — `func Path() (string, error)` — reads `SKILLDOZER_CONFIG` (literal, `filepath.Clean`'d) else `os.UserConfigDir()/skilldozer/config.yaml`; returns error verbatim (findConfig must treat any error as fall-through).
- `config/config.go:58-78` — `func Load(path string) (File, error)` — returns `os.ReadFile` error VERBATIM so callers can `errors.Is(err, fs.ErrNotExist)`; broken YAML is a hard error.
- `config/config.go:34-37` — `type File struct { Store string \`yaml:"store,omitempty"\` }`.
- Per-rule shape contract: each `findXxx()` returns `(dir string, src Source, found bool)`; never errors. `findConfig` must follow the same shape: `if d exists { return d, SourceConfig, true }; return "", 0, false`.

## Architecture / data flow

`main.run()` → `skillsdir.Find()` (returns `(dir, src, err)`) → on success `discover.Index(dir)` builds the catalog; on `ErrNotFound`, `fmt.Fprintln(stderr, err)` + exit 1. `--path` additionally emits the `Source.String()` label to stderr. The config package is a pure-function-of-env settings sidecar (Path/Load/Save/DefaultStore) consumed by the new findConfig and by init. No internal package calls back into skillsdir.

## Start here

Open `internal/skillsdir/skillsdir.go` — add `findConfig()` (mirror `findEnv`'s shape, reading `config.Path()` + `config.Load()` + returning the `Store` dir if it exists) and insert its call at `Find()` line 249, then change the `ErrNotFound` message at line 234. Then update the 7 tests listed in §1 (substring assertions) and harden the 7 tests listed in §2 against the config rule.

## Residual risks / open questions
- The repo lives at `/home/dustin/projects/skilldozer`; a developer here may well have a real `~/.config/skilldozer/config.yaml`. The 7 §2 tests will fail intermittently on such a host unless they also neutralize `SKILLDOZER_CONFIG`/`XDG_CONFIG_HOME`. High-priority to fix in the same change.
- `findConfig` must treat BOTH a missing config file (`fs.ErrNotExist`) AND any `Path()` error (e.g. relative `XDG_CONFIG_HOME`, unset `HOME`) as fall-through (`found=false`), never as a hard error — otherwise the locked per-rule `(dir, src, found bool)` shape and the "all-miss → ErrNotFound" contract break.
- `config.File.Store` may be empty string in an existing-but-incomplete config.yaml; `findConfig` should treat an empty/relative/non-existent `Store` as a miss (fall through), consistent with how `findEnv` treats a non-existent dir.
