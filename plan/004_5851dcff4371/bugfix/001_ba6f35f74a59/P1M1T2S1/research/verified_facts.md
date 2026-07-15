# P1.M1.T2.S1 — Verified Facts & Site Inventory (Issue 2: vanished-configured-store)

Every line number below is the CURRENT `internal/skillsdir/skillsdir.go` / `skillsdir_test.go`,
verified by grep + read on 2026-07-15. The contract's line numbers are ACCURATE this time
(no drift). HEAD = the bugfix-2 working tree.

## 0. Ownership boundary (no conflicts — verified)

| Region | File:lines | Owner | Status |
|---|---|---|---|
| findConfig (body + doc) | skillsdir.go:92-128 | **T2.S1 (this)** | — |
| Find() (body + doc) + ErrNotFound + sentinel | skillsdir.go:272-301 | **T2.S1** | — |
| findConfig test call sites + renamed/new tests | skillsdir_test.go:555-624 + new | **T2.S1** | — |
| completions/skilldozer.fish | (the parallel sibling) | **T1.S3** (fish completion, Issue 1) | in-progress |

The parallel sibling **P1.M1.T1.S3 edits ONLY `completions/skilldozer.fish`** (the fish
completion script, `//go:embed`'d at main.go:60). It does NOT touch any `.go` file. →
DISJOINT from `internal/skillsdir/`. Land in either order; no merge conflict.

## 1. findConfig — EXACT current code (skillsdir.go:92-128)

Doc comment 92-105; signature 106; body 107-128. Five return paths:
```go
106: func findConfig() (dir string, src Source, found bool) {
107: 	p, err := config.Path()
108: 	if err != nil {
109: 		return "", 0, false // no bootstrap path (e.g. relative $XDG_CONFIG_HOME) -> fall through
110: 	}
111: 	f, err := config.Load(p)
112: 	if err != nil {
113: 		return "", 0, false // missing/unreadable/malformed -> "not yet configured" -> fall through
114: 	}
115: 	if f.Store == "" {
116: 		return "", 0, false // no `store` key -> fall through
117: 	}
118: 	var store string
119: 	if filepath.IsAbs(f.Store) {
120: 		store = filepath.Clean(f.Store)
121: 	} else {
122: 		store = filepath.Join(filepath.Dir(p), f.Store) // relative to config file's dir (PRD §8.1)
123: 	}
124: 	info, err := os.Stat(store)
125: 	if err != nil || !info.IsDir() {
126: 		return "", 0, false // store path is not an existing dir -> fall through   ← THE BUG
127: 	}
128: 	return store, SourceConfig, true
129: }
```
**The 5 return paths and their NEW 4th value:**
| Line | Current | New 4th value | Why |
|---|---|---|---|
| 109 | `("", 0, false)` | `""` | config.Path() err — no store resolved |
| 113 | `("", 0, false)` | `""` | config.Load err — no store resolved |
| 116 | `("", 0, false)` | `""` | f.Store == "" — no store to stat |
| 126 | `("", 0, false)` | **`store`** | store WAS resolved (118-123) then os.Stat failed → VANISHED |
| 128 | `(store, SourceConfig, true)` | `""` | HIT — store exists, nothing vanished |

The `store` variable (declared 118, assigned 119-123) is IN SCOPE at line 126 — it holds the
resolved-but-nonexistent path that becomes the vanishedStore return. (decisions.md §D1 rationale.)

## 2. ErrNotFound + Find() — EXACT current code (skillsdir.go:272-301)

```go
272: // ErrNotFound is returned by Find when every §8.3 rule misses (unconfigured). Its
273: // message is the user-facing one-line fix (PRD §8.4 / §6.4): main prints it to
274: // stderr and exits 1. Print it verbatim (err.Error()); do not wrap or prefix it.
275: var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer --init`")
...
277: // Find locates the skills directory per PRD §8.3 priority order (first hit wins):
278: //  1. SKILLDOZER_SKILLS_DIR env var (SourceEnv).
279: //  2. Config file `store` (SourceConfig).
280: //  3. Sibling of the running binary, symlink-aware (SourceSibling).
281: //  4. Walk up from cwd (SourceWalkUp).
282: //  5. None ⇒ unconfigured: returns ErrNotFound.
283: // The first rule to hit wins and Find returns (absDir, src, nil). If all miss it
284: // returns ("", 0, ErrNotFound); the caller (main) prints the error to stderr and
285: // exits 1.
288: func Find() (dir string, src Source, err error) {
289: 	if d, s, ok := findEnv(); ok {
290: 		return d, s, nil
291: 	}
292: 	if d, s, ok := findConfig(); ok { // PRD §8.3 priority #2
293: 		return d, s, nil
294: 	}
295: 	if d, s, ok := findSibling(); ok {
296: 		return d, s, nil
297: 	}
298: 	if d, s, ok := findWalkUp(); ok {
299: 		return d, s, nil
300: 	}
301: 	return "", 0, ErrNotFound
302: }
```
- The findConfig call is at line **292** (the contract's citation is exact).
- ErrNotFound (275) is where the NEW sentinel `ErrConfiguredStoreMissing` goes (right after it,
  decisions.md §D2 — same package-level-sentinel pattern).

## 3. CRITICAL: `fmt` is NOT imported today — must be ADDED

Current skillsdir.go imports (verified):
```go
import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dabstractor/skilldozer/internal/config"
)
```
The ONLY `fmt.` reference today is in a COMMENT (`// Satisfies fmt.Stringer` at line 46). The new
`fmt.Errorf("%w: ...", ErrConfiguredStoreMissing, vanished)` in Find() REQUIRES `"fmt"` in the
import block. ADD it alphabetically (after "errors", before "io/fs"):
```go
	"errors"
	"fmt"
	"io/fs"
```
This is the single non-obvious prerequisite — miss it and `go build` fails with "undefined: fmt".

## 4. main.go callers — NO main.go change (verified)

7 sites call `skillsdir.Find()`: main.go:541, 564, 600, 640, 681, 712, 1213. Each handles the
error identically (verified at 712-716):
```go
dir, _, err := skillsdir.Find()
if err != nil {
    fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
    return 1
}
```
→ The new wrapped ErrConfiguredStoreMissing propagates through the EXISTING `err` handling
unchanged: printed verbatim to stderr, exit 1, nothing on stdout. **Zero main.go edits.** The
error message itself is the user-facing doc (contract DOCS §5).

## 5. The 6 findConfig test call sites (skillsdir_test.go) — append a 4th value

| Line | Test | Current | New |
|---|---|---|---|
| 558 | TestFindConfigHit | `got, src, found := findConfig()` | `got, src, found, _ := findConfig()` |
| 573 | TestFindConfigMissingFile | `if dir, src, found := findConfig(); found {` | `if dir, src, found, _ := findConfig(); found {` |
| 581 | TestFindConfigMissingStoreKey | `if dir, src, found := findConfig(); found {` | `if dir, src, found, _ := findConfig(); found {` |
| 589 | TestFindConfigStoreDirAbsent → **rename TestFindConfigStoreVanished** | `if dir, src, found := findConfig(); found {` | `if dir, src, found, vanished := findConfig(); ...` (semantic change — see §6) |
| 600 | TestFindConfigMalformedYAML | `if dir, src, found := findConfig(); found {` | `if dir, src, found, _ := findConfig(); found {` |
| 614 | TestFindConfigRelativeStoreResolvedAgainstConfigDir | `got, src, found := findConfig()` | `got, src, found, _ := findConfig()` |

## 6. TestFindConfigStoreDirAbsent (587-596) → rename + semantic change

Current (asserts fall-through only):
```go
587: func TestFindConfigStoreDirAbsent(t *testing.T) {
588: 	writeCfg(t, "store: "+filepath.Join(t.TempDir(), "no-such-store")+"\n")
589: 	if dir, src, found := findConfig(); found {
590: 		t.Errorf("findConfig absent store dir: got found=true dir=%q src=%v; want false", dir, src)
591: 	}
592: }
```
NEW (TestFindConfigStoreVanished): assert found==false AND vanishedStore == the configured
non-existent path. Capture the path into a var so the assertion can name it:
```go
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
```

## 7. Why TestFindAllMissReturnsErrNotFound (513) stays GREEN (the safety proof)

From skillsdir_research.md §4 (verified): `unsetEnvVar` (skillsdir_test.go:14-42) sets
`SKILLDOZER_CONFIG` to a GHOST config FILE PATH (a non-existent file). So in that test:
config.Path() returns the ghost path (non-empty) → config.Load() returns fs.ErrNotExist →
findConfig bails at **line 113** (the config.Load-err branch), which returns vanishedStore=`""`.
It NEVER reaches line 126 (the vanished-store branch). → Find() still returns ErrNotFound.
**TestFindAllMissReturnsErrNotFound is unaffected.** (The only test the sentinel changes is
TestFindConfigStoreDirAbsent, which the contract renames.)

## 8. The design decisions (decisions.md §D1-D3) — summarized

- **D1 (4-value return):** Chosen over side-channel sentinel / re-query / error-return. The first
  3 values are UNCHANGED for all miss paths; only line 126 populates vanishedStore. findConfig
  STILL "never errors" (locked per-rule shape) — it returns the vanishedStore string; Find()
  constructs the error. So the findConfig doc's "Never errors (locked per-rule shape)" stays true.
- **D2 (sentinel + %w):** `ErrConfiguredStoreMissing` + `fmt.Errorf("%w: ...", ErrConfiguredStoreMissing, ...)`
  so tests use `errors.Is(err, ErrConfiguredStoreMissing)` (not message matching). Mirrors ErrNotFound.
- **D3 (env still wins):** findEnv (priority 1) runs BEFORE findConfig. If SKILLDOZER_SKILLS_DIR is
  set+valid, findEnv wins and findConfig is never called → vanished error never fires. This is the
  TestFindEnvOverridesVanishedStore guarantee. Priority order is UNCHANGED.

## 9. Baseline (verified green for the package)

- `go test ./internal/skillsdir/...` → passes today (the bug is behavioral — silent wrong-store
  fall-through — and the existing TestFindConfigStoreDirAbsent LOCKS the buggy fall-through).
  T2.S1 REPLACES that assertion with the vanished assertion + adds Find()-level tests.
- go.mod/go.sum unchanged (the only new import is stdlib `fmt`).

## 10. The §6.4 distinction this enforces (the WHY)

PRD §6.4: "skilldozer is unconfigured ⇒ `run \`skilldozer --init\`` **(or, if configured but the
dir vanished, the concise reason + fix)**, exit 1." Today both collapse to silent fall-through
(or ErrNotFound). The fix splits them:
- **Config file missing/unreadable/no-store-key** (lines 109/113/116) → genuine "not yet
  configured" → fall through to sibling/walk-up (§8.1: never a hard error). UNCHANGED.
- **Config file present with `store:` naming a non-existent dir** (line 126) → "configured but
  vanished" → LOUD error (wrapped ErrConfiguredStoreMissing) naming the path + the fix. NEW.
This stops `pi --skill "$(skilldozer myskill)"` from silently resolving a skill from an
UNRELATED store (sibling/walk-up) when the configured store vanished — the §6.4 reliability goal.
