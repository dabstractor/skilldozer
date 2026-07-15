# skillsdir analysis — `findConfig` + store-dir-absent surface

Scope: `/home/dustin/projects/skilldozer/internal/skillsdir/skillsdir.go` and
`/home/dustin/projects/skilldozer/internal/skillsdir/skillsdir_test.go`.
All line numbers are 1-indexed, verified against the current files on disk.

## 1) `findConfig()` — full signature + every return statement

**Signature** — `internal/skillsdir/skillsdir.go:106`
```go
func findConfig() (dir string, src Source, found bool) {
```
Doc comment spans `skillsdir.go:92-105`. The shape is the **locked per-rule
contract** `(dir, src Source, found bool)` — never returns an `error`.

**Body + every return** — `skillsdir.go:106-128`
```go
106:func findConfig() (dir string, src Source, found bool) {
107:	p, err := config.Path()
108:	if err != nil {
109:		return "", 0, false // no bootstrap path (e.g. relative $XDG_CONFIG_HOME) -> fall through
110:	}
111:	f, err := config.Load(p)
112:	if err != nil {
113:		return "", 0, false // missing/unreadable/malformed -> "not yet configured" -> fall through
114:	}
115:	if f.Store == "" {
116:		return "", 0, false // no `store` key -> fall through
117:	}
118:	var store string
119:	if filepath.IsAbs(f.Store) {
120:		store = filepath.Clean(f.Store)
121:	} else {
122:		store = filepath.Join(filepath.Dir(p), f.Store) // relative to config file's dir (PRD §8.1)
123:	}
124:	info, err := os.Stat(store)
125:	if err != nil || !info.IsDir() {
126:		return "", 0, false // store path is not an existing dir -> fall through
127:	}
128:	return store, SourceConfig, true
129:}
```

| Return line | dir | src | found | Triggered by |
|---|---|---|---|---|
| 109 | `""` | `0` | `false` | `config.Path()` error (e.g. relative `$XDG_CONFIG_HOME`, unset `HOME`) |
| 113 | `""` | `0` | `false` | `config.Load(p)` error — **missing / unreadable / malformed YAML** (`fs.ErrNotExist` for a missing file, returned verbatim per `config.go:53`) |
| 116 | `""` | `0` | `false` | config loaded but `f.Store == ""` (no `store:` key) |
| 126 | `""` | `0` | `false` | **store path is not an existing dir** — `os.Stat` errors OR not a directory (the "vanished store" branch) |
| 128 | `store` | `SourceConfig` | `true` | hit: store resolves to an existing directory |

Only line **128** is a hit. Every other path returns the same triple
`("", 0, false)` — there is currently **no way for `findConfig` to distinguish
"store dir absent/vanished" (line 126) from "no config at all" (line 113) or
"no store key" (line 116)**. All four miss-paths collapse to `("", 0, false)`.

## 2) How `Find()` calls `findConfig()` and what happens with the result

**`Find()` — `internal/skillsdir/skillsdir.go:288-302`**
```go
288:func Find() (dir string, src Source, err error) {
289:	if d, s, ok := findEnv(); ok {
290:		return d, s, nil
291:	}
292:	if d, s, ok := findConfig(); ok { // PRD §8.3 priority #2
293:		return d, s, nil
294:	}
295:	if d, s, ok := findSibling(); ok {
296:		return d, s, nil
297:	}
298:	if d, s, ok := findWalkUp(); ok {
299:		return d, s, nil
300:	}
301:	return "", 0, ErrNotFound
302:}
```

`Find()` calls `findConfig()` exactly once, at **line 292**. It pattern-matches
`d, s, ok := findConfig()` and **only inspects the `ok` (found) boolean**:
- If `ok == true` → returns `(d, s, nil)` (the config store dir wins).
- If `ok == false` → `Find()` **discards `d` and `s` entirely** and falls
  through to `findSibling()` (line 295).

Critically: **`Find()` ignores any error/sentinel from `findConfig`** — it only
sees the `(dir, src, found bool)` triple. As long as `findConfig` keeps
returning `found == false` for a vanished store, `Find()` falls through to
sibling → walkup → `ErrNotFound`. A sentinel would only propagate to the caller
if `findConfig`'s return shape or the `Find()` call site changed.

Priority order implemented here: env (289) → config (292) → sibling (295) →
walkup (298) → `ErrNotFound` (301). `ErrNotFound` is defined at
`skillsdir.go:283`:
```go
283:var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer --init`")
```

## 3) `TestFindConfigStoreDirAbsent` — exact assertions

**`internal/skillsdir/skillsdir_test.go:586-591`**
```go
586:// Rule 2 miss: `store` names a dir that does not exist -> fall through.
587:func TestFindConfigStoreDirAbsent(t *testing.T) {
588:	writeCfg(t, "store: "+filepath.Join(t.TempDir(), "no-such-store")+"\n")
589:	if dir, src, found := findConfig(); found {
590:		t.Errorf("findConfig absent store dir: got found=true dir=%q src=%v; want false", dir, src)
591:	}
592:}
```

**What it does / asserts:**
- `writeCfg` (`skillsdir_test.go:542-550`) writes a **valid** config file
  `<tmpdir>/config.yaml` containing `store: <tmpdir2>/no-such-store`, and sets
  `SKILLDOZER_CONFIG` to that file (`t.Setenv`, line 549). So the **config file
  EXISTS and PARSES**; only the `store` dir is absent.
- It calls `findConfig()` directly (not `Find()`).
- **Single assertion (line 589):** `found` must be `false`. If `found == true`
  it fails with an `Errorf` (not `Fatalf`) reporting the unexpected `dir`/`src`.
  It asserts **nothing** about `dir`/`src` values when `found == false`, and
  **nothing about any error/sentinel** — it does not even capture an error,
  because `findConfig` returns no error today.
- This is the test that pins the **line-126 behavior** (`os.Stat` fail →
  `("", 0, false)`). It is the direct expression of "store dir absent → fall
  through, silently."

## 4) `unsetEnvVar` — what path it sets `SKILLDOZER_CONFIG` to, and the vanished-store question

**`unsetEnvVar` — `internal/skillsdir/skillsdir_test.go:11-45`** (signature at line 14)
```go
11:// unsetEnvVar removes envVar for the duration of the test and restores the
...
14:func unsetEnvVar(tb testing.TB) {
15:	tb.Helper()
...
27:	// Also neutralize the config-file rule (PRD §8.3 rule 2): point SKILLDOZER_CONFIG
28:	// at a non-existent path so findConfig deterministically misses once it is wired
29:	// into Find(). ...
33:	cfgGhost := filepath.Join(tb.TempDir(), "no-config.yaml")
34:	prevCfg, hadCfg := os.LookupEnv("SKILLDOZER_CONFIG")
35:	if err := os.Setenv("SKILLDOZER_CONFIG", cfgGhost); err != nil {
36:		tb.Fatalf("setenv SKILLDOZER_CONFIG: %v", err)
37:	}
38:	tb.Cleanup(func() {
39:		if hadCfg {
40:			_ = os.Setenv("SKILLDOZER_CONFIG", prevCfg)
41:		} else {
42:			_ = os.Unsetenv("SKILLDOZER_CONFIG")
43:		}
44:	})
45:}
```

- It also unsets `SKILLDOZER_SKILLS_DIR` (`envVar`) — lines 16-26 — and restores it.
- **The path it sets `SKILLDOZER_CONFIG` to:** `filepath.Join(tb.TempDir(),
  "no-config.yaml")` — a **NON-EXISTENT** YAML file under a fresh temp dir.
  `tb.TempDir()` is unique per test, so the file provably does not exist.

### Would a vanished-store sentinel change break `TestFindAllMissReturnsErrNotFound`?

**No.** Here is the exact test — `internal/skillsdir/skillsdir_test.go:511-523`:
```go
511:// Find: all three rules miss -> ErrNotFound. (chdir into an empty temp dir; the
512:// walk ascends to /, which has no skills/ on this host — verified hermetic.)
513:func TestFindAllMissReturnsErrNotFound(t *testing.T) {
514:	unsetEnvVar(t)
515:	t.Chdir(t.TempDir())
516:	got, src, err := Find()
517:	if !errors.Is(err, ErrNotFound) {
518:		t.Fatalf("Find() all-miss: err=%v; want ErrNotFound", err)
519:	}
520:	if got != "" || src != 0 {
521:		t.Errorf("Find() all-miss: got=%q src=%v; want \"\" and 0", got, src)
522:	}
523:}
```

Why it survives a vanished-store sentinel:
- `unsetEnvVar(t)` (line 514) sets `SKILLDOZER_CONFIG` to a **ghost config FILE
  that does not exist**.
- When `Find()` → `findConfig()` runs: `config.Path()` returns the ghost path
  (line 107-108 — `Path()` just returns the env value, does not `Stat` it), then
  `config.Load(p)` (line 111) tries to read a missing file. Per
  `internal/config/config.go:53`, `Load` returns the read error **verbatim**
  (`fs.ErrNotExist`).
- `findConfig` therefore returns at **line 113** (the "missing/unreadable/malformed
  → fall through" branch) — **it never reaches line 126** (the "store path is not
  an existing dir" branch) where a vanished-store sentinel would fire.
- A vanished-store sentinel only triggers when the config file **exists, parses,
  and has a non-empty `store`** but that dir is gone (line 124-126). This test
  never satisfies the precondition (its config file does not exist at all).
- Net: `findConfig` returns `("", 0, false)` → `Find()` falls through sibling →
  walkup (the `t.Chdir(t.TempDir())` at line 515 guarantees no `skills/` ancestor)
  → `ErrNotFound`. Test still asserts `errors.Is(err, ErrNotFound)` and passes.

**Important adjacent breakage (not asked, but flagged):** the test that a
vanished-store sentinel **would** affect is `TestFindConfigStoreDirAbsent`
(§3 above). That test writes a valid config whose `store` is a non-existent dir
and asserts `found == false` with no error captured. If the bugfix makes
`findConfig` surface a sentinel (e.g. return `found == false` **plus** a
package-level error var like `ErrStoreVanished`, or change the shape to return an
`error`), then:
- If the shape stays `(dir, src, found bool)` and `found` stays `false` with a new
  sentinel exposed another way → `TestFindConfigStoreDirAbsent` still passes (it
  only checks `found`), but gains nothing.
- If `found` semantics change or the return shape grows an `error` →
  `TestFindConfigStoreDirAbsent` (and the other findConfig-direct tests:
  `TestFindConfigHit`, `TestFindConfigMissingFile`, `TestFindConfigMissingStoreKey`,
  `TestFindConfigMalformedYAML`, `TestFindConfigRelativeStoreResolvedAgainstConfigDir`,
  lines 553-624) must be updated, and `Find()`'s call site at line 292 must decide
  whether to propagate the sentinel as a distinct error vs `ErrNotFound`.

## 5) Is `findConfig` called from anywhere besides `Find()`?

**No** — in production code `findConfig` has exactly **one call site**:
`internal/skillsdir/skillsdir.go:292` inside `Find()`. Verified via
`grep -rn "findConfig" --include="*.go" internal/ main.go`:

- **Production call sites:** `skillsdir.go:292` only.
- **Definition:** `skillsdir.go:106`.
- **Test call sites** (direct, unit-level — not routed through `Find()`):
  `skillsdir_test.go:558` (`TestFindConfigHit`), `:573`
  (`TestFindConfigMissingFile`), `:581` (`TestFindConfigMissingStoreKey`),
  `:589` (`TestFindConfigStoreDirAbsent`), `:600` (`TestFindConfigMalformedYAML`),
  `:614` (`TestFindConfigRelativeStoreResolvedAgainstConfigDir`).
- **Doc/comment references only** (not calls): `internal/config/config.go:4,26,44,53,121`.

`findConfig` is a lowercase (unexported) package-level function, so it cannot be
referenced outside `package skillsdir`. No other package calls it.

## Start Here
Open `internal/skillsdir/skillsdir.go` and read lines **106-128** (the full
`findConfig`) together with **288-302** (`Find`). The bug surface is the line-126
branch: a vanished/non-existent store dir currently collapses into the same
`("", 0, false)` as "no config at all," so `Find()` silently falls through to
sibling/walkup and may locate a *different* dir than the one the user configured.
`TestFindConfigStoreDirAbsent` (`skillsdir_test.go:587-592`) is the test that
pins the current silent behavior; `TestFindAllMissReturnsErrNotFound`
(`skillsdir_test.go:513-523`) is safe under a sentinel change because its config
file does not exist (bails at line 113, not 126).

## Constraints / risks for an implementer
- **Locked per-rule shape:** every `findXxx()` returns `(dir, src Source, found
  bool)` and **never errors**. `Find()` (line 292) only reads `found`. Introducing
  a sentinel likely means either (a) a new exported package-level `error` var
  surfaced via a side channel, or (b) changing the return shape — which forces
  updates to all six direct `findConfig` tests and the `Find()` call site.
- `config.Load` returns read errors **verbatim** (`config.go:53`), so
  `errors.Is(err, fs.ErrNotExist)` distinguishes "no config file" (line 113) from
  "store dir gone" (line 126) — that is the seam a sentinel would use.
- `findConfig` is called from exactly one production site; the blast radius of a
  shape change is `Find()` + the six unit tests + possibly `main` (how it reports
  the vanished store to the user).
