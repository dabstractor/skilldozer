# System Context — skilldozer bugfix round 001_ad9c7988274c

## Project

`skilldozer` is a single-binary Go CLI that resolves a skill *tag* to an absolute
skill-directory path, so that `pi --skill "$(skilldozer <tag>)"` loads that skill.
It is a **path printer**: it never installs/copies skills into pi discovery
locations, never builds a catalog index/manifest, and never prints anything to
stdout on a failed resolution. The skills store lives anywhere; a settings file
records where. Module: `github.com/dabstractor/skilldozer`, Go 1.25, sole
third-party dep is `gopkg.in/yaml.v3`.

## Hard constraints that every fix MUST preserve

- **§6.4 stdout discipline (load-bearing):** any unresolved/ambiguous tag ⇒ print
  one error line per problem tag to **stderr**, print **NOTHING** to stdout, exit
  `1`. This guarantees `pi --skill "$(skilldozer badtag)"` fails loudly.
- **No catalog index on disk** (§17). Catalog is always walked from disk.
- **Bare tag resolution NEVER prompts** (§8.2). stdin access is confined to
  `resolveStore`, which only `init` calls.
- **stdlib-only besides yaml.v3** — no `golang.org/x/term`, no third-party
  tilde/homedir libs (the codebase's `stdinIsTerminal` deliberately avoids x/term).

## Source layout (files touched by this bugfix round)

```
main.go               # CLI: parseArgs, run, exclusivityError, runInit, resolveStore, setupStore, chooseStore
main_test.go          # ~2440 lines; the test harness (buffer-based run() seam)
.gitignore            # must match PRD §16 exact 5-entry spec
PRD.md                # READ-ONLY (humans own it); §7.2 note candidate for Issue 7
internal/config/      # config.Save/Load/Path/DefaultStore (DefaultStore uses os.UserHomeDir @ line 154)
internal/skillsdir/   # Find() — §8.3 discovery priority
internal/discover/    # Index() walk + Skill struct + frontmatter parse
internal/check/       # check.Check() → Report{BySkill, Errors, Warnings}
```

## The dispatch architecture (how a fix flows)

The testable core is `func run(args []string, stdout, stderr io.Writer) int`
(`main.go:408`). It is the single seam tests target: tests pass two
`*bytes.Buffer`s and assert on `.String()` + the returned exit code.

```
main() → run(os.Args[1:], os.Stdout, os.Stderr) → os.Exit(code)

run():
  1. parseArgs(args)           → config struct (all flags + tags)
  2. if c.help       → usage to STDOUT, exit 0          (help wins)
  3. if c.version    → version to STDOUT, exit 0
  4. if c.unknownFlag → error to STDERR, exit 2
  5. if exclusivityError(c) → error to STDERR, exit 2
  6. if c.init       → runInit(c, stdout, stderr)        ← Issues 1, 2, 5
  7. mode ladder: path → list → search → check → all → tags
```

### parseArgs (`main.go:153`)

Index-based loop (not range) so value-taking flags (`--search <q>`, `--store <dir>`)
can consume the next token via `i++` without it also being captured as a tag.
Three normalization layers: (a) `--flag=value` splitter, (b) short-bundle
expander `expandShortBundle`, (c) exact-match switch incl. `check`/`init` reserved
subcommand tokens. Unknown dashed token → `c.unknownFlag` (first offender only).

### exclusivityError (`main.go:686`)

Returns `(bad bool, msg string)`. Order: listing-modes count (`{path,list,searchMode,all}`,
≥2 ⇒ error); tags + listing modes; check + tags; check + modes; init + (tags|modes).
**Issue 3 site:** the tags predicate at `main.go:702` omits `c.path`, an internal
inconsistency (it IS in the count set and the check+mode set).

### runInit (`main.go:988`) — Issues 1, 5

Orchestrates: resolveStore (choose+absolutize) → configpkg.Path → setupStore
(mkdir+seed+writeconfig) → report. **Issue 1 site:** the check-report loop
(`main.go:1037-1053`) renders `OK`/`WARN`/`ERROR` lines + the summary to
**stdout** via `fmt.Fprintf(stdout, ...)`. Per §6.1 these belong on **stderr**;
only the store-path headline (`fmt.Fprintln(stdout, dir)` @ 1026) stays on stdout.

### resolveStore (`main.go:865`) — Issue 5

I/O wrapper around the pure `chooseStore`. Absolutizes via
`filepath.Abs(store)` (`main.go:886`). **Issue 5 site:** `filepath.Abs` does NOT
expand `~`, so `~/myskills` → `<cwd>/~/myskills`. Tilde expansion must run BEFORE
`filepath.Abs`. The pure seam `chooseStore` (`main.go:822`) returns its choice
verbatim; do NOT widen its signature — fix in `resolveStore`/a helper it calls.

### setupStore (`main.go:948`) — Issue 2 destructive path

ALWAYS writes the config (`configpkg.Save`) whether seeded or adopted. Issue 2's
danger: `init --store` with no value leaves `c.init=true, c.initStore=""` (the
`init` token already set init), so auto-detect runs and `setupStore` overwrites
the existing config. The guard must distinguish "init with --store-but-no-value"
(needs error) from "bare init with no --store" (must prompt) — `c.initStore==""`
alone is insufficient.

## Testing infrastructure (reuse these)

- **`run()` buffer seam:** `var out, errOut bytes.Buffer; code := run([]string{...}, &out, &errOut)`.
  `out.String()`=stdout, `errOut.String()`=stderr, `code`=exit. Used ~84×.
- **`unsetSkillsEnv(t)`** (`main_test.go:28`): neutralizes `SKILLDOZER_SKILLS_DIR` +
  `SKILLDOZER_CONFIG`. Used for unconfigured/walk-up determinism.
- **`writeSkillTree(t, map[relTag]SKILL.md-content) string`** (`main_test.go:41`):
  builds a temp skills tree; `""` key writes SKILL.md at root.
- **`sampleStore(t) string`** (`main_test.go:463`): 2-skill fixture (example,
  writing/reddit). The canonical tag/list/resolve fixture.
- **`t.Setenv`** for `SKILLDOZER_CONFIG`/`SKILLDOZER_SKILLS_DIR`/`HOME` (auto-restored).
- **`t.Chdir(t.TempDir())`** to escape the repo walk-up rule (repo cwd HAS
  skills/example/SKILL.md). MANDATORY for unconfigured/never-prompts tests.
- **`os.UserHomeDir()`-dependent code** is tested by `t.Setenv("HOME", ...)`
  (see `internal/config/config_test.go:275` which sets `HOME=""` for the error path).

## Exit-code contract (must not regress)

- `0`: success (help/version/path/list/search/check-clean/all/all-tags-resolved/init-ok)
- `1`: path/list failed, no skills, any tag unresolved/ambiguous, skills dir unresolvable,
  no recognized mode, init error/cancel
- `2`: unknown flag, mutually-exclusive modes mixed (tags+mode, check+tags, check+mode,
  init+mode/tags — and after Issue 3/4 fixes: tag+path, init+init)

## Cross-cutting risks flagged by research

1. **runInit check-report is a byte-copy of the `if c.check` block** (`main.go:566-584`).
   The existing comment says "do not refactor; mirror." For Issue 1 they DIVERGE
   (init→stderr, check→stdout). Just swap the writer; do not extract a shared helper.
2. **`.pi-subagents/` removal (Issue 6):** trimming `.gitignore` to §16's 5 entries
   makes the `.pi-subagents/` agent-artifacts dir untracked (surfaces in `git status`).
   Per the prior round's §D3 decision this is intended (do not bless extras). Residual.
3. **Issue 7 is documentation-only.** The code documents `check`/`init` reservation as
   deliberate (`main.go:248-256`, `main.go:269-280`). A code change to resolve a skill
   named `check` would silently shadow the `skilldozer check` subcommand — a worse UX
   surprise. The fix is a §7.2 note (but PRD.md is human-owned/READ-ONLY; this round
   records the decision in `architecture/decisions.md` and surfaces it, NOT a code edit).
