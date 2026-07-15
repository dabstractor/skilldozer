# Issue Analysis — Skilldozer Bugfix Round 2

Per-issue root cause, fix surface, and test impact. All line numbers verified
against the working tree by scout subagents.

---

## Issue 1 (MAJOR) — `--shell` value completion offers skill tags instead of `bash zsh fish`

### Root Cause

None of the three completion files reference `--shell` at all. The `--shell`
flag is real (documented in `usageText` OPTIONS at main.go:116, parsed in
`parseArgs` at main.go:248-253 and 320-326, and used in `runCompletion`), but the
frozen completion flag lists predate its addition or intentionally omit it.

When a user types `skilldozer --shell <TAB>`:
- **bash** (`completions/skilldozer.bash`): the `case "$prev"` block (line ~38)
  has no `--shell)` case → falls through to the default positional handler →
  offers skill tags via `skilldozer --relative --all`.
- **zsh** (`completions/_skilldozer`): the `_arguments` array (lines ~30-42) has
  no `'--shell[...]:shell:(bash zsh fish)'` entry → the value slot falls through
  to `*: :->args` → offers skill tags.
- **fish** (`completions/skilldozer.fish`): no `complete -c skilldozer -l shell`
  directive → the dynamic tag directive fires → offers skill tags.

### Fix Surface

Three independent files, one change each:

1. **`completions/skilldozer.bash`** (~line 38, the `case "$prev"` block):
   Add `--shell) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;`
   after the `--store|--init)` line. Optionally add `--shell` to the `compgen -W`
   flag list at line ~44-46 so `skilldozer --<TAB>` advertises it.

2. **`completions/_skilldozer`** (the `_arguments` array, ~line 30-42):
   Add `'--shell[Force a shell for completion]:shell:(bash zsh fish)'` so the
   value slot offers the enum. Optionally add `--shell` to the flag list.

3. **`completions/skilldozer.fish`** (after the existing flag directives):
   Add `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"`
   (`-x` = no file completion, `-a` = the enum values).

### PRD Tension Note

§14.6's `skilldozer -<TAB>` table enumerates 13 long flags WITHOUT `--shell`,
while §14.2 requires `--shell`'s value to complete to the enum. The minimum fix
(value routing) satisfies §14.2. Adding `--shell` to the advertised flag list is
recommended for consistency since `--shell` IS in `usageText`.

### Test Impact

- `TestEmbeddedCompletionsMatchOnDisk` (main_test.go:2995): automatically
  validates embed↔on-disk byte-identity after `go build`. PASSES if files are
  modified and rebuilt.
- `TestCompletionScriptMapping` (main_test.go:2957): checks header strings, not
  `--shell`. Still passes.
- `TestRunCompletionBashScript` / `TestRunCompletionFishScript`
  (main_test.go:3019/3035): check for marker strings. Still pass.
- **New test needed:** verify the emitted completion scripts contain `--shell`
  (e.g., `strings.Contains(script, "--shell")`). This is the Go-testable proxy
  for the shell-level behavior; the actual "tab offers bash zsh fish" behavior
  requires a running shell and is verified manually.

---

## Issue 2 (MAJOR) — Configured store whose directory has vanished silently falls through to a different store

### Root Cause

`findConfig()` (`skillsdir.go:106-128`) returns `(dir string, src Source, found
bool)` with NO error channel. When the config file loads successfully and has a
`store:` key but that store directory does not exist on disk (line 124-126),
`findConfig` returns `("", 0, false)` — indistinguishable from "no config file"
or "no store key." `Find()` (line 292) only checks `found`; on `false` it falls
through to `findSibling()` → `findWalkUp()`, potentially resolving an
**unrelated** store.

The behavior is locked by `TestFindConfigStoreDirAbsent` (skillsdir_test.go:587)
which asserts `found == false` with no error for a vanished store.

### Fix Surface

Change `findConfig()` to distinguish "store vanished" from "unconfigured":

```go
// New signature — add vanishedStore string:
func findConfig() (dir string, src Source, found bool, vanishedStore string)
```

At line 126, change:
```go
// OLD:
return "", 0, false // store path is not an existing dir -> fall through
// NEW:
return "", 0, false, store // store dir vanished — signal to Find()
```

All other return paths get `, ""` appended (no vanished store).

In `Find()` (line 292-294), change:
```go
// OLD:
if d, s, ok := findConfig(); ok {
    return d, s, nil
}
// NEW:
d, s, ok, vanished := findConfig()
if ok {
    return d, s, nil
}
if vanished != "" {
    return "", 0, fmt.Errorf("configured store %q does not exist; run `skilldozer --init` or recreate the directory", vanished)
}
```

Add a sentinel error for testability:
```go
var ErrConfiguredStoreMissing = errors.New("configured skills store directory does not exist")
```
And use `fmt.Errorf("%w: ...", ErrConfiguredStoreMissing, ...)` so tests can
`errors.Is(err, ErrConfiguredStoreMissing)`.

### Why main.go Needs No Change

Every caller of `Find()` in `main.go` already does:
```go
dir, _, err := skillsdir.Find()
if err != nil {
    fmt.Fprintln(stderr, err)
    return 1
}
```
The new error propagates verbatim to stderr, exit 1, nothing on stdout — exactly
the §6.4 contract. The `--init` flow calls `Find()` AFTER `setupStore` creates
the dir, so the error never fires post-init.

### Test Impact (Critical — Existing Tests Lock Current Behavior)

| Test | File:Line | Current Assertion | Required Change |
|------|-----------|-------------------|-----------------|
| `TestFindConfigStoreDirAbsent` | skillsdir_test.go:587 | `found == false` (no error) | Must now assert `found == false, vanishedStore != ""` (the store path). Rename to `TestFindConfigStoreVanished` for clarity. |
| `TestFindConfigHit` | :553 | 3-value destructure | Add 4th value `, _` |
| `TestFindConfigMissingFile` | :570 | 3-value destructure | Add 4th value `, _` |
| `TestFindConfigMissingStoreKey` | :578 | 3-value destructure | Add 4th value `, _` |
| `TestFindConfigMalformedYAML` | :593 | 3-value destructure | Add 4th value `, _` |
| `TestFindConfigRelativeStoreResolvedAgainstConfigDir` | :607 | 3-value destructure | Add 4th value `, _` |

**Safe tests (no change needed):**
- `TestFindAllMissReturnsErrNotFound` (:513): uses `unsetEnvVar` which points
  `SKILLDOZER_CONFIG` at a non-existent FILE → `config.Load` returns
  `fs.ErrNotExist` → `findConfig` bails at line 113 (file missing), NOT line 126
  (store dir gone). `vanishedStore` stays `""`. Test still returns `ErrNotFound`.
- All `findEnv`, `findSibling`, `findWalkUp`, `resolveSiblingFromExe`,
  `HasSkillMD`, `findWalkUpAncestor` tests: don't touch `findConfig`.

**New tests needed:**
- `TestFindErrorsOnVanishedConfiguredStore`: write a valid config with a
  non-existent store dir, call `Find()`, assert
  `errors.Is(err, ErrConfiguredStoreMissing)` and that the message names the
  configured path + the fix.
- `TestFindEnvOverridesVanishedStore`: set `SKILLDOZER_SKILLS_DIR` to an existing
  dir AND a config with a vanished store → env (rule 1) still wins → no error.

---

## Issue 3 (MINOR) — Value-taking flags missing their value: inconsistent exit codes

### Root Cause

Three value-taking flags, three different no-value behaviors:

| Flag | No-value token | Field set | run() result | Exit |
|------|---------------|-----------|-------------|------|
| `--store` | last token, no follower | `storeMissingValue = true` | `"--store requires a value"` → stderr | **2** |
| `--search`/`-s` | last token, no follower | nothing (`searchMode` stays false) | falls through to no-recognized-mode → usage to **stdout** | **0** |
| `--shell` | last token, no follower | nothing (`completion` stays false) | falls through to no-recognized-mode → usage to **stdout** | **0** |

The comment at main.go:293 claims `--search` no-value yields "exit 1" but it
actually exits 0 (the no-recognized-mode fall-through prints usage to stdout).

### Fix Surface

Make `--search` and `--shell` symmetrical with `--store`:

1. Add fields to the `config` struct (main.go:166):
   ```go
   searchMissingValue bool  // --search/-s with no value follows
   shellMissingValue  bool  // --shell with no value follows
   ```

2. In `parseArgs` main switch `case "--search", "-s":` (main.go:288):
   ```go
   if i+1 < len(args) {
       c.searchMode = true
       c.searchQ = args[i+1]
       i++
   } else {
       c.searchMissingValue = true  // NEW
   }
   ```

3. In `parseArgs` main switch `case "--shell":` (main.go:320):
   ```go
   if i+1 < len(args) {
       c.completion = true
       c.completionShell = args[i+1]
       i++
   } else {
       c.shellMissingValue = true  // NEW
   }
   ```

4. In `expandShortBundle` (main.go:444 default case): set `c.searchMissingValue = true`.

5. In `run()` (main.go:499, after `storeMissingValue` check):
   ```go
   if c.searchMissingValue {
       fmt.Fprintln(stderr, "skilldozer: --search requires a query")
       return 2
   }
   if c.shellMissingValue {
       fmt.Fprintln(stderr, "skilldozer: --shell requires a value (bash|zsh|fish)")
       return 2
   }
   ```

6. Fix the misleading comment at main.go:293: change "(exit 1)" to "(exit 2)"
   (or remove the parenthetical, since the new behavior exits 2).

### Test Impact

- Existing `--store` tests unaffected.
- New tests:
  - `TestRunSearchNoValueExits2`: `run([]string{"--search"}, ...)` → exit 2,
    stderr contains "requires a query", stdout empty.
  - `TestRunSearchShortNoValueExits2`: `run([]string{"-s"}, ...)` → exit 2.
  - `TestRunShellNoValueExits2`: `run([]string{"--shell"}, ...)` → exit 2,
    stderr contains "requires a value", stdout empty.
  - `TestParseArgsSearchMissingValue`: `parseArgs([]string{"--search"})` →
    `searchMissingValue == true`.
  - `TestParseArgsShellMissingValue`: `parseArgs([]string{"--shell"})` →
    `shellMissingValue == true`.

---

## Issue 4 (MINOR) — POSIX `--` end-of-options separator rejected as unknown flag

### Root Cause

`parseArgs` has no special handling for `--`. A bare `--` token:
1. Fails the `=`-form guard (main.go:202: no `=` in `--`).
2. Fails the short-bundle guard (main.go:259: `len("--")==2`, not `> 2`).
3. Falls through to `switch a` → `default:` (main.go:343).
4. `strings.HasPrefix("--", "-")` is true → `c.unknownFlag = "--"`.
5. `run()` prints `unknown flag '--'` to stderr, exit 2.

### Fix Surface

Add an `endOfOpts` flag to the parseArgs loop:

```go
func parseArgs(args []string) config {
    var c config
    endOfOpts := false  // NEW
    for i := 0; i < len(args); i++ {
        a := args[i]
        if endOfOpts {           // NEW: everything after -- is positional
            c.tags = append(c.tags, a)
            continue
        }
        if a == "--" {           // NEW: end-of-options separator
            endOfOpts = true
            continue
        }
        // ... existing `=`-form, short-bundle, and switch logic ...
    }
    return c
}
```

This must be placed BEFORE the `=`-form check (main.go:202) and the short-bundle
check (main.go:259) so `--` is intercepted before any other classification.

### Test Impact

- New test `TestParseArgsDashDashSeparator`: `parseArgs([]string{"--", "-x"})` →
  `tags == ["-x"]`, `unknownFlag == ""`.
- New test `TestParseArgsDashDashBeforeTag`: `parseArgs([]string{"--", "mytag"})` →
  `tags == ["mytag"]`.
- New test `TestRunDashDashResolvesTag`: `run([]string{"--", "example"})` with a
  valid store → resolves "example" tag, exit 0.
- Existing tests: `TestParseArgsDashedUnknownNotATag` asserts `--frobnicate` is
  unknown — still passes (it's not bare `--`).

---

## Issue 5 (MINOR) — README version documentation inaccurate

### Root Cause

README.md "Usage" section states:
```markdown
# Version is the git-describe value (dynamic, not a fixed string)
```

This is only true for the `./install.sh` build path (which injects
`-ldflags "-X main.version=$(git describe ...)"`). A plain `go build -o skilldozer .`
(the README's own "From source" instructions) and `go install` yield `skilldozer
dev` because the ldflags injection is not performed.

The default value `dev` is set at main.go:57: `var version = "dev"`.

### Fix Surface

Update the README.md comment line to clarify the build-path dependency:

```markdown
# Version is the git-describe value when built via ./install.sh; a plain `go build` reports `dev`
```

### Test Impact

None — this is a documentation-only change.
