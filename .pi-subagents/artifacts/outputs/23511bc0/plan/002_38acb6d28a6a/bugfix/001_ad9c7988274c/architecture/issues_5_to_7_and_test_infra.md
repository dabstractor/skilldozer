# Issues 5–7 + Testing Infrastructure — Scout Findings

Scope: reproduce + document bugfix Issues 5, 6, 7 and the testing harness in the
skilldozer Go CLI. No code changes made (scout/review pass). Every claim below is
tied to an exact `file:line` read from the current tree.

---

## ISSUE 5 — Tilde (`~`) not expanded in `init` interactive prompt

### Confirmed behavior (repro)
`skilldozer init` resolves the store path with `filepath.Abs(store)`. `filepath.Abs`
makes a path **absolute** but does **NOT** expand a leading `~`. So an interactive
typed path like `~/myskills` becomes `<cwd>/~/myskills` instead of `$HOME/myskills`.

- `filepath.Abs` docs (Go stdlib): "Abs calls Clean and, if the path is not absolute,
  joins it with the current working directory." It has no `~` semantics. Verified by
  the absence of any home-expansion helper in the codebase (see grep evidence below).
- The non-interactive `init --store <dir>` / `init <dir>` paths hit the same
  `resolveStore` → `filepath.Abs`, so they share the bug (the `--store` path is
  passed in as `c.initStore` and flows to `resolveStore` at `main.go:991`).

### Exact location
- **`main.go:865`** — `func resolveStore(haveStore string) (string, error)` is where
  the absolutization happens.
- **`main.go:886`** — `abs, err := filepath.Abs(store)` (the buggy line). `store` is
  the verbatim string returned by `chooseStore` (which returns user input UNCHANGED —
  see `main.go:860` `return choice, nil`).
- The whole `resolveStore` I/O-bearing wrapper: **`main.go:855-889`**.
- Call site: **`main.go:990-994`** inside `runInit` (`store, err := resolveStore(c.initStore)`).

### Test seam (the pure function)
- **`main.go:822`** — `func chooseStore(haveStore, cwd string, isTTY bool, defaultStore string, prompt func(label, def string) (string, error)) (string, error)`.
  `chooseStore` is pure (no I/O) and is the existing test seam for init logic. BUT it
  returns the choice **verbatim** (`main.go:860`) and does NOT absolutize — that is
  deliberately deferred to `resolveStore`. **Tilde expansion must therefore be added in
  `resolveStore` (or a helper it calls), NOT in `chooseStore`** — otherwise the pure
  function would have to take an injected home resolver, widening its signature.
  Note `chooseStore`'s doc comment at `main.go:818-821` explicitly states the verbatim
  return and that "resolveStore absolutizes it via filepath.Abs."

### Precise code change
In `resolveStore` (`main.go:865`), expand `~` to `$HOME` **before** `filepath.Abs`.
Recommended shape — add a small helper `expandHome(path string) (string, error)` and
call it between `chooseStore` and `filepath.Abs`:

```go
store, err := chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)
if err != nil {
    return "", err
}
// Issue 5: filepath.Abs does NOT expand ~. Expand a leading "~"/"~/" to $HOME
// BEFORE absolutizing, so `init ~/myskills` → $HOME/myskills, not <cwd>/~/myskills.
store, err = expandHome(store)
if err != nil {
    return "", fmt.Errorf("skilldozer init: expand home: %w", err)
}
abs, err := filepath.Abs(store)   // main.go:886 (idempotent on already-abs paths)
```

`expandHome` (new helper, ~8 lines, mirrors the `os.UserHomeDir()` pattern already used
in `internal/config/config.go:150-160`):

```go
// expandHome replaces a leading "~" or "~/" with $HOME (os.UserHomeDir).
// Non-tilde-prefixed paths are returned unchanged. A "~" with no $HOME returns
// the os.UserHomeDir error verbatim (same contract as config.DefaultStore).
func expandHome(path string) (string, error) {
    if path == "~" {
        home, err := os.UserHomeDir()
        return home, err
    }
    if strings.HasPrefix(path, "~/") {
        home, err := os.UserHomeDir()
        if err != nil {
            return "", err
        }
        return filepath.Join(home, path[2:]), nil
    }
    return path, nil
}
```

Rationale: reuse `os.UserHomeDir()` (already the established `~` source —
`internal/config/config.go:154`) so behavior is consistent with `DefaultStore`. Do NOT
shell out to a 3rd-party tilde lib (PRD §7.3: yaml.v3 is the *only* allowed 3rd-party dep;
PRD §17 forbids extra runtimes). Note a leading `~user/` (another user's home) is out of
scope — Go has no stdlib for it and PRD never requires it.

### Relevant test pattern + helper to reuse
- **Reuse the `init` test pattern**: `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`
  (`main_test.go:2325`). It sets `SKILLDOZER_CONFIG` + unsets `SKILLDOZER_SKILLS_DIR` +
  `t.Chdir(t.TempDir())`, calls `run([]string{"init","--store",store}, &out, &errOut)`,
  then asserts `configpkg.Load(cfg).Store == store` and `stdout` contains the path.
- **New test for tilde** (non-interactive path, simplest): set `t.Setenv("HOME", tmp)`
  and run `init --store "~/sub"`; assert the written config store equals
  `filepath.Join(tmp, "sub")` (abs, tilde expanded) — NOT `<cwd>/~/sub`.
  - Set HOME via `t.Setenv("HOME", ...)` exactly as `internal/config/config_test.go:275`
    does. Note `os.UserHomeDir` reads `$HOME` on Unix (override works); on plan/CI it
    must be a non-empty absolute path.
- **Interactive seam test** (optional, pure): `chooseStore` already accepts an injected
  `prompt func`; tests like `TestParseArgsInitStoreLongForm` (`main_test.go:1186`) show
  the parse-level assertions. A `chooseStore` test passing `"~/foo"` as the typed choice
  is NOT a tilt-test — `chooseStore` returns it verbatim by design; assert tilde output
  at the `expandHome`/`resolveStore` level instead.

### Grep evidence (no existing tilde handling)
```
$ grep -n "tilde\|UserHomeDir\|HomeDir\|~/\|ExpandHome\|homedir" main.go
(none)
```
Only `internal/config/config.go:154` uses `os.UserHomeDir` today; `resolveStore` has none.

---

## ISSUE 6 — `.gitignore` has extra entries beyond PRD §16 spec

### PRD §16 spec (exact, 5 entries)
From **`PRD.md:423-431`** (section "16. `.gitignore`"):
```
/skilldozer
/dist
*.test
*.out
.DS_Store
```
PRD also states (`PRD.md:432`): "everything else is committed, including `skills/example/`."

### Current `.gitignore` (read verbatim, full file)
```
# Build artifacts
/skilldozer
/dist
/build
*.test
*.out

# Dependency directories
node_modules/
venv/

# Environment files
.env

# OS-specific files
.DS_Store

# Agent runtime artifacts (transcripts, run logs, meta)
.pi-subagents/
```

### Extra entries (beyond §16 — to be REMOVED)
| # | Extra entry | Category comment also added |
|---|-------------|------------------------------|
| 1 | `/build` | under "# Build artifacts" |
| 2 | `node_modules/` | under "# Dependency directories" |
| 3 | `venv/` | under "# Dependency directories" |
| 4 | `.env` | under "# Environment files" |
| 5 | `.pi-subagents/` | under "# Agent runtime artifacts ..." |

Plus 4 section-comment lines (`# Build artifacts`, `# Dependency directories`,
`# Environment files`, `# OS-specific files`, `# Agent runtime artifacts …`) — PRD §16's
canonical block has NO comments. Per the prior round's decision
(`plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/decisions.md` §D3
"Trim .gitignore to PRD §16 spec (do NOT bless extras)"): if a contributor wants an
extra, they update §16 themselves; do not silently bless extras.

### Precise change
Rewrite `.gitignore` to exactly the 5-line PRD §16 block (no comments), byte-for-byte:
```
/skilldozer
/dist
*.test
*.out
.DS_Store
```
- The 5 kept entries map 1:1 to `PRD.md:426-430`.
- Removing `.pi-subagents/` is the notable one — it will cause `git status` to show the
  agent artifacts dir. That is intended per §D3 (do not bless extras). Risk: the local
  `.pi-subagents/` directory will surface as untracked; confirm with the parent whether
  a separate top-level ignore mechanism is desired (out of scope for this fix — see
  Residual Risks).

### Relevant test pattern / verification
There is no Go test for `.gitignore` (it is not code). Verify by diff:
`diff <(sed -n '426,430p' PRD.md) .gitignore` should produce no output post-fix. This
mirrors the §13-style "diff against spec" convention used in the acceptance gates.

---

## ISSUE 7 — A skill whose canonical tag is literally `check` or `init` cannot be resolved by that tag

### Confirmed behavior (repro)
In `parseArgs`, the tokens `"check"` and `"init"` are matched by dedicated `case`
branches BEFORE the default tag-capture branch. So `skilldozer check` sets `c.check=true`
(validation mode) and `skilldozer init` sets `c.init=true` (setup mode) — neither token
ever reaches `c.tags`, and a real skill at `skills/check/SKILL.md` or `skills/init/SKILL.md`
can NEVER be resolved by its canonical tag. The basename/frontmatter-name precedence
(`PRD.md:161-181`, §7.2 steps 1-2) is bypassed entirely at the parse layer.

### Exact location (already commented as DELIBERATE)
- **`main.go:247`** — `case "check":` with comment block (`main.go:248-256`) stating:
  "`check` is a RESERVED positional token … A skill literally tagged `check` cannot be
  resolved via `skilldozer check` (subcommand names are reserved, as in any CLI)." A
  nested skill `writing/check` still resolves (only the EXACT token "check" matches).
- **`main.go:268`** — `case "init":` with comment block (`main.go:269-280`) stating
  `init` is "a RESERVED positional token (like `check`)." The comment's "GOTCHA":
  "a store literally named `check`/`init` must be passed via `--store`."
- These are inside `parseArgs` (`main.go:153`); the `default:` tag-capture branch
  starts at **`main.go:281`**.

### Recommended approach: DOCUMENTATION-ONLY (PRD §7.2 note), not code
This is the approach that fits. Rationale:
1. The code already documents the reservation as deliberate at `main.go:248-256` and
   `main.go:269-280`. Subcommand-token reservation is standard CLI behavior (git, kubectl,
   etc. all reserve subcommand words).
2. The prior bugfix round set the precedent: `plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/decisions.md`
   §D7 ("No code change; documentation-only") for an analogous "deliberate behavior" item.
3. A code change to disambiguate `check`/`init` (e.g. "if a skill named `check` exists,
   resolve it instead") would silently change a subcommand into a tag lookup — a much
   worse UX surprise (a stray `skills/check/SKILL.md` would shadow `skilldozer check`).
   That violates the reservation's whole point.

**Precise doc change**: add a note to **PRD §7.2** (`PRD.md:161`, the "Tag resolution
precedence" section) — insert a line after the precedence list or after the Examples
block (`PRD.md:179-181`) stating:

> **Reserved tags.** The tokens `check` and `init` are reserved subcommand names (§8.2,
> §9) and are never treated as skill tags. A skill whose canonical tag is literally
> `check` or `init` is still discoverable by `--list` and `--all`, and by a *nested*
> path (e.g. `writing/check`), but cannot be resolved via `skilldozer check` /
> `skilldozer init`. This is the standard CLI subcommand-reservation rule.

(If the parent prefers the PRD note lives in a `decisions.md` §D7-style entry instead of
PRD §7.2 body, that is equivalent; both are doc-only.)

### Code alternative (document only; do NOT implement unless directed)
If a code change were ever required, the safe shape is an explicit escape flag, e.g.
`--tag check` / a leading `--` POSIX separator (`skilldozer -- check`) that forces the
following tokens into the tag list regardless of subcommand collision. This is strictly
larger scope than the bug warrants and is listed only for completeness.

### Relevant test pattern (documents the existing reservation)
- `TestParseArgsCheckSubcommand` (`main_test.go:1120`) — asserts `parseArgs(["check"])`
  sets `c.check=true` and `len(c.tags)==0` ("'check' is a subcommand, not a tag").
- `TestParseArgsCheckAfterFlag` (`main_test.go:1131`) — `check` recognized after a flag.
- `TestParseArgsCheckAndTagBothCaptured` (`main_test.go:1144`) — `check sometag` captures
  both (run() later rejects via exclusivity).
- `TestParseArgsInitSubcommand` (`main_test.go:1186`-adjacent, ~`main_test.go:1186`) and
  `TestParseArgsInitPositionalDir` (`main_test.go:1170`) — `init` reserves the token;
  `init <dir>` captures dir into `c.initStore` not tags.

No test currently asserts "a skill named check/init cannot resolve" — that negative
behavior is the gap the PRD note closes.

---

## Testing Infrastructure — how `main_test.go` asserts on stdout vs stderr

### The `run()` contract + buffer pattern (the load-bearing seam)
- **`main.go:408`** — `func run(args []string, stdout, stderr io.Writer) int`. It takes
  two `io.Writer`s and returns an exit code (int). This is THE seam: tests pass two
  `*bytes.Buffer`s and assert on their `.String()`.
- Standard test incantation (used **84×** in `main_test.go`):
  ```go
  var out, errOut bytes.Buffer
  code := run([]string{"example"}, &out, &errOut)
  ```
  Then: `out.String()` = captured **stdout** (data; must equal exact bytes for path
  emitters, e.g. `TestRunTagSingleResolvesToDir` `main_test.go:483`); `errOut.String()` =
  captured **stderr** (prose/labels; substring-checked); `code` = exit int (0/1/2).
- **Discipline enforced**: §6.4 "print NOTHING to stdout on failure" is tested by
  asserting `out.Len()==0` on error paths (e.g. `TestRunTagAtomicityUnknownPrintsNothing`
  `main_test.go:~523`, `TestRunBareTagUnconfiguredNeverPrompts` `main_test.go:~2426`).
  Init keeps stdout clean by writing the prompt dialog to **stderr** (see
  `resolveStore`'s `prompt := func(...){ return readPrompt(r, os.Stderr, label, def) }`
  at `main.go:882-884`).

### Helpers
- **`unsetSkillsEnv(t)`** — `main_test.go:28`. `t.Helper()`; sets `SKILLDOZER_SKILLS_DIR=""`
  AND points `SKILLDOZER_CONFIG` at a non-existent temp path so `findConfig` misses.
  Used **8×**. Neutralizes env + config-file rules (PRD §8.3 rules 1-2) so the walk-up/
  unconfigured paths are deterministic.
- **`writeSkillTree(t, layout map[string]string) string`** — `main_test.go:41`. `t.Helper()`;
  builds a temp `skills/` tree from `map[relTag]SKILL.md-content` (relTag uses `/`
  separators, cross-platform via `filepath.FromSlash`); `""` key writes `SKILL.md` at root.
  Returns the temp root. Used **13×**. Feeds `skillsdir.Find()` via `SKILLDOZER_SKILLS_DIR`.
- **`sampleStore(t) string`** — `main_test.go:463`. `t.Helper()`; thin wrapper over
  `writeSkillTree` returning a 2-skill store (`example`, `writing/reddit`). Used **29×**
  — the canonical fixture for tag/list/resolve tests.

### Environment setup patterns
- **`t.Setenv`** for `SKILLDOZER_CONFIG` and `SKILLDOZER_SKILLS_DIR` (auto-restored on
  test end; never raw `os.Setenv`). E.g. `main_test.go:2329-2331`:
  `t.Setenv("SKILLDOZER_CONFIG", cfg)` + `t.Setenv("SKILLDOZER_SKILLS_DIR", "")`.
- **`t.TempDir()`** for stores, configs, and parent dirs (auto-cleaned). Combined with
  `writeSkillTree`/`sampleStore` for self-contained fixtures.
- **`t.Chdir(t.TempDir())`** (used **10×**) — escapes the repo's walk-up discovery rule
  (the repo cwd HAS `skills/example/SKILL.md`, so without chdir the walk-up rule would
  resolve and mask the "unconfigured"/"env" path under test). MANDATORY for
  never-prompts / unconfigured tests (see `TestRunBareTagUnconfiguredNeverPrompts`
  comment, `main_test.go:~2416`).
- `os.UserHomeDir()`-dependent code (config, and the proposed Issue 5 `expandHome`) is
  tested by overriding `HOME` via `t.Setenv("HOME", ...)` — see
  `internal/config/config_test.go:275` (sets `HOME=""` to force the error path). For a
  positive Issue 5 test, set `HOME` to a non-empty absolute temp dir.

### Init test patterns to mirror
- **`TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`** (`main_test.go:2325`) —
  the gold-standard init integration test. Asserts: (a) `setupStore` CREATES a
  non-existent store dir (`os.Stat`); (b) config written with `store=<abs>`
  (`configpkg.Load(cfg).Store == store`); (c) stdout contains the store path; (d) exit 0.
  Note `resolveStore`'s `filepath.Abs` is *idempotent* on already-abs `store`
  (comment, `main_test.go:2343`) — so today's tests never exercise the tilde/relative
  branch. **The new tilde test must use a relative/tilde input to hit the bug.**
- **`TestParseArgsInitStoreLongForm`** (`main_test.go:1186`) — parse-level:
  `parseArgs(["init","--store","/tmp/x"])` ⇒ `c.init==true`, `c.initStore=="/tmp/x"`,
  `len(c.tags)==0`. Pure, no `run()`; the model for a parse-only assertion.
- `TestParseArgsInitPositionalDir` (`main_test.go:1170`) — `init <dir>` positional form.

### Where Issue 5's test slots in
Reuse the `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` shape but:
1. `home := t.TempDir(); t.Setenv("HOME", home)` (and unset `XDG_DATA_HOME` to avoid
   `DefaultStore` reading it).
2. `run([]string{"init","--store", filepath.Join("~","myskills")}, ...)` (note: `~` is a
   literal, so build the arg as `"~/myskills"`, not `filepath.Join`-expanded).
3. Assert `configpkg.Load(cfg).Store == filepath.Join(home, "myskills")` (tilde expanded)
   and stdout contains that abs path — NOT `<cwd>/~/myskills`.

---

## Open questions / risks
1. **Issue 6 — `.pi-subagents/` removal**: trimming `.gitignore` to §16 means the
   `.pi-subagents/` dir (agent run logs/transcripts) becomes untracked and will show in
   `git status`. Per §D3 this is intended (don't bless extras), but the parent should
   confirm there's no separate ignore mechanism expected. Flagged as a residual risk.
2. **Issue 5 — `~user` form**: the proposed `expandHome` only handles `~` and `~/...`,
   not `~otheruser/`. PRD never requires the latter; documenting as intentional scope
   limit. Also `os.UserHomeDir` on Windows reads `%USERPROFILE%` — tilde is a Unix
   convention; on non-Unix the expansion is a no-op-ish best-effort (acceptable: skilldozer
   targets Unix per the install.sh symlink flow, `main.go`/`install.sh`).
3. **Issue 7 — doc location**: PRD §7.2 body vs `decisions.md` §D7 entry. Either is
   doc-only; parent to pick. No code change recommended.

---

## Files Retrieved (exact ranges)
1. `main.go:153-313` — `parseArgs` (Issue 5 '=' form, Issue 7 `case "check"`@247 / `case "init"`@268, default tag branch @281).
2. `main.go:408-498` — `run()` dispatch (help/version/unknown/exclusivity/init/mode ladder).
3. `main.go:498-640` — `run()` list/search/check/all/tag branches (Issue 7 shadowing confirmation; atomicity contract).
4. `main.go:818-889` — `chooseStore` (pure, verbatim return @860) + `resolveStore` (I/O wrapper, `filepath.Abs` @886 — Issue 5 bug site).
5. `main.go:988-1043` — `runInit` (calls `resolveStore(c.initStore)` @991; where tilde fix takes effect).
6. `main.go:1-25` — imports (no `os.UserHomeDir` import today; the fix adds its use via the helper).
7. `internal/config/config.go:139-160` — `DefaultStore` using `os.UserHomeDir()` @154 — the existing `~`-source pattern to mirror in `expandHome`.
8. `PRD.md:161-181` — §7.2 tag-resolution precedence (Issue 7 note insertion point).
9. `PRD.md:423-432` — §16 `.gitignore` exact spec (5 entries; Issue 6 target).
10. `main_test.go:28-58` — `unsetSkillsEnv`, `writeSkillTree` helpers.
11. `main_test.go:463-523` — `sampleStore` helper + `TestRunTagSingleResolvesToDir` (the out/errOut buffer pattern).
12. `main_test.go:1120-1196` — `TestParseArgsCheck*` and `TestParseArgsInit*` (Issue 7 reservation tests + init long form).
13. `main_test.go:2325-2440` — `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` + `TestRunBareTagUnconfiguredNeverPrompts` (init integration + never-prompts patterns).
14. `plan/001_fcde63e5bb60/bugfix/001_7fdd8ce7410b/architecture/decisions.md:86-98` — §D7 "No code change; documentation-only" precedent for Issue 7.
15. `.gitignore` (full file, 23 lines) — Issue 6 current content.

## Start Here
Open **`main.go:865` (`resolveStore`)** and read down to line 889 — that is the single
function where all three of Issue 5's fix (tilde expand before `filepath.Abs` @886), the
`chooseStore` pure-seam boundary (@822), and the `runInit` call site (@991) converge.
Then `PRD.md:423` (§16) for the Issue 6 target and `PRD.md:161` (§7.2) for the Issue 7 note.
