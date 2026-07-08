# Bug Fix Requirements

## Overview

Creative end-to-end QA of the skilldozer implementation against the PRD (this
round's scope: the §8 config-file model + the `skilldozer init` subcommand, plus
the full §6 CLI surface).

**Method:** built the binary fresh, ran the entire §13 acceptance suite
(all green), then tested as a user and as an adversary: happy paths, the
`pi --skill "$(skilldozer <tag>)"` end-to-end workflow, resolution precedence
(canonical/basename/name/alias, ambiguity, atomicity), `--search`, `check`
validation, all four §8.3 discovery rules in isolation, config edge cases
(relative store, malformed YAML, missing/non-dir env), `init` interactive
(via real pty) and non-interactive flows, frontmatter parsing (CRLF, BOM, block
scalars, length boundaries), flag combinations/exit codes, `install.sh`, and
all three shell completions (syntax-checked).

**Overall assessment: solid.** The core contract is intact — every §13
acceptance gate passes, the primary `pi --skill "$(skilldozer <tag>)"` workflow
works flawlessly, the hard constraints hold (no catalog index, not in any pi
discovery location, nothing on stdout on failed resolution, atomic multi-tag
semantics), and the §8.3 discovery priority (the PRD's "most failure-prone
area") is correct in every branch. The issues below live in the secondary
surface this round added: `init`'s output contract and flag-value parsing.

No Critical issues found (nothing blocks core functionality).

## Critical Issues (Must Fix)

None. The canonical `pi --skill "$(skilldozer <tag>)"` path, all §13
acceptance gates, discovery resolution, and the atomic-failure contract all
work correctly.

## Major Issues (Should Fix)

### Issue 1: `init` writes the `check` validation report to stdout, violating PRD §6.1's stdout contract

**Severity**: Major
**PRD Reference**: §6.1 (init row: stdout = "The configured store path.");
§6.4 (stdout discipline for `$(...)`); §8.2 step 5 ("Print the output of
`skilldozer --path` ... and `skilldozer check`.")
**Expected Behavior**: Per the authoritative §6.1 CLI contract, `skilldozer
init` prints exactly **one** line to stdout — the configured store path — so
that `STORE="$(skilldozer init --store /path)"` yields a clean, single-line
value usable by scripts/CI (the documented non-interactive form, §8.2).
**Actual Behavior**: `init` also writes the full `check` report (the per-skill
`OK`/`WARN`/`ERROR` lines plus the `N skills, M errors, K warnings` summary) to
**stdout**, so `$(skilldozer init)` captures the store path **plus** the entire
report. `main.go runInit` renders the check report with `fmt.Fprintf(stdout,
...)` ("Mirrors the `if c.check` branch render VERBATIM").
**Steps to Reproduce**:
```bash
cd /tmp && mkdir -p /tmp/A/store
SKILLDOZER_CONFIG=/tmp/A/cfg.yaml env -u SKILLDOZER_SKILLS_DIR \
  ./skilldozer init --store /tmp/A/store </dev/null 2>/dev/null
# stdout is THREE lines:
#   /tmp/A/store
#   OK    example (example)
#   1 skills, 0 errors, 0 warnings
```
**Suggested Fix**: In `runInit`, route the check-report loop and summary to
**stderr** (not stdout), leaving only the resolved store dir on stdout. This
harmonizes §8.2 step 5 ("print ... check") with the authoritative §6.1 stdout
contract and with `init`'s own pattern (the "Seeded/Adopted" status line and
the prompt already go to stderr; the check report is the same kind of
human-facing status output and is currently the lone outlier on stdout). Note
the tension: §8.2 step 5 literally says "print the output of `skilldozer
check`," and `check` itself prints to stdout — but §6.1 is labelled
"authoritative" and §6.4's whole philosophy is clean stdout, so stderr is the
correct destination for the report in the `init` context.

### Issue 2: `init --store` (and `--store`) with no value silently overwrites the config instead of erroring

**Severity**: Major
**PRD Reference**: §6 header ("Unknown flags ⇒ error + exit 2"); §8.2
(`init --store <dir>` non-interactive form); delta-PRD §2 constraint #3 ("init
is non-destructive ... never clobber or delete").
**Expected Behavior**: A value-taking flag presented without its value
(`skilldozer init --store` with nothing after `--store`) should fail fast with
an "argument required" error and a non-zero exit, like any CLI flag missing a
required value.
**Actual Behavior**: `parseArgs` only sets `c.init`/`c.initStore` for `--store`
when `i+1 < len(args)`; when `--store` is the last token it does nothing.
Because the earlier `init` token already set `c.init = true` with
`c.initStore == ""`, the invocation silently degrades to **auto-detect** init
(cwd-scan / XDG default), which then **unconditionally rewrites the config**
(`setupStore` always calls `config.Save`). An existing, valid `store:` value is
overwritten with the auto-detected path. The same "value-taking flag with no
value silently no-ops" defect affects `--search`/`-s` too (there it is harmless
— falls through to usage/exit 1 — but for `--store` inside `init` it is
destructive).
**Steps to Reproduce**:
```bash
mkdir -p /tmp/B/realstore/foo
printf -- '---\nname: foo\ndescription: real.\n---\n# x\n' > /tmp/B/realstore/foo/SKILL.md
printf 'store: /tmp/B/realstore\n' > /tmp/B/cfg.yaml
echo "BEFORE:"; cat /tmp/B/cfg.yaml          # store: /tmp/B/realstore
SKILLDOZER_CONFIG=/tmp/B/cfg.yaml env -u SKILLDOZER_SKILLS_DIR \
  ./skilldozer init --store </dev/null >/dev/null 2>&1
echo "exit=$?"                               # 0  (should be 2)
echo "AFTER:"; cat /tmp/B/cfg.yaml           # store: <auto-detected path>  ← silently overwritten
```
(When run from a cwd that contains any `SKILL.md` at depth — e.g. a tree full
of test stores — the auto-detected default is that cwd, so the config is
rewritten to point somewhere unexpected. This was reproduced against the real
default config path `~/.config/skilldozer/config.yaml` during testing.)
**Suggested Fix**: Make value-taking flags whose value is missing a hard error
(exit 2), e.g. set a "missing-argument" sentinel in the `--store`/`--search`
no-value branches and report it in `run()` (before init dispatch). At minimum,
`--store` with no value must not fall through to auto-detect init that
overwrites the config.

## Minor Issues (Nice to Fix)

### Issue 3: `tag` combined with `--path` is not rejected (silent tag drop), unlike every other mode

**Severity**: Minor
**PRD Reference**: §6.3 ("Mixing `<tag>` with `--list`/`--search`/`--all` is an
error (exit 2)"); `exclusivityError` in `main.go` (treats `--path` as a
"listing mode" for mode+mode conflicts but excludes it from the tag conflict).
**Expected Behavior**: Consistent handling of tags + an inspection mode.
**Actual Behavior**: `skilldozer foo --list` / `--search x` / `--all` all exit
2, but `skilldozer foo --path` (or `--path foo`) silently runs `--path` and
ignores the tag — even when the tag is unknown: `skilldozer NONEXISTENTTAG
--path` exits 0. Because `--path` is in the mutually-exclusive "listing modes"
set for mode+mode conflicts, its omission from the tags conflict is an internal
inconsistency that can mask a user typo (a user typing `skilldozer myskill
--path` expecting the skill's path instead receives the **store** path with no
warning).
**Steps to Reproduce**:
```bash
export SKILLDOZER_SKILLS_DIR=/tmp/store   # any store
./skilldozer NONEXISTENTTAG --path; echo "exit=$?"   # 0  (tag silently dropped)
./skilldozer NONEXISTENTTAG --list; echo "exit=$?"   # 2  (as expected)
```
**Suggested Fix**: Add `c.path` to the tag-conflict predicate in
`exclusivityError` (i.e. `hasTags && (c.path || c.list || c.searchMode ||
c.all)`), or document that `--path` is intentionally exempt.

### Issue 4: `init init` runs init instead of erroring (inconsistent with `init check`)

**Severity**: Minor
**PRD Reference**: §6.3 (mutually-exclusive modes); `parseArgs` `init` case
(suppresses capture when the next token is `check`/`init`).
**Expected Behavior**: `init` combined with the reserved subcommand token
`init` should error (exit 2), matching `init check` → exit 2.
**Actual Behavior**: `skilldozer init init </dev/null` runs init (exit 0): the
first `init` sets `c.init=true` and refuses to capture the second `init` as the
store, but the second `init` then sets `c.init=true` again (idempotent) with no
`c.check`/conflict to trip exclusivity, so init proceeds with auto-detect.
**Steps to Reproduce**:
```bash
./skilldozer init init </dev/null >/dev/null 2>&1; echo "exit=$?"   # 0
./skilldozer init check      >/dev/null 2>&1; echo "exit=$?"        # 2
```
**Suggested Fix**: Treat a second reserved-subcommand token (`check`/`init`)
after `init` as a conflict (set the corresponding mode flag so
`exclusivityError` catches it), or reject repeated `init`.

### Issue 5: Tilde (`~`) is not expanded in `init`'s interactive prompt input

**Severity**: Minor
**PRD Reference**: §8.2 step 1 (typed path overrides the default).
**Expected Behavior**: A user typing `~/myskills` at the interactive prompt
reasonably expects home-directory expansion, as in every shell.
**Actual Behavior**: `resolveStore` absolutizes the typed string with
`filepath.Abs`, which does **not** expand `~`. The literal string is stored:
`store: /<cwd>/~/myskills`, and a directory literally named `~` is created.
**Steps to Reproduce** (interactive, via pty so `stdinIsTerminal()` is true):
```bash
# type "~/myskills" at the prompt
# result: config gets `store: /tmp/E/~/myskills`; a real dir named '~' is created
```
**Suggested Fix**: Expand a leading `~/` (and a bare `~`) to `$HOME` before
`filepath.Abs` when the prompt returns a tilde-bearing value.

### Issue 6: `.gitignore` contains entries beyond the PRD §16 spec set

**Severity**: Minor
**PRD Reference**: §16 (exact 5-entry set: `/skilldozer`, `/dist`, `*.test`,
`*.out`, `.DS_Store`).
**Expected Behavior**: `.gitignore` matches the §16 spec.
**Actual Behavior**: The file carries extra entries (`/build`, `node_modules/`,
`venv/`, `.env`, `.pi-subagents/`) and section comments. (This was flagged in
the prior bugfix round, issue 3, and is still present.)
**Suggested Fix**: Trim to the §16 set, or update §16 if the extras are
intended.

### Issue 7: A skill whose canonical tag is literally `check` or `init` cannot be resolved by that tag

**Severity**: Minor
**PRD Reference**: §7.2 (canonical tag = the skill dir's relative path, no
carve-out for reserved names); `parseArgs` `check`/`init` cases.
**Expected Behavior**: Per §7.2 the canonical tag is the relTag; nothing in the
PRD exempts the names `check`/`init`.
**Actual Behavior**: `check` and `init` are reserved positional tokens
(subcommands), so `skilldozer check` runs validation and `skilldozer init`
runs setup — a skill at `skills/check/SKILL.md` (relTag `check`) is shadowed
and unresolvable via its canonical tag. Workarounds exist (resolve by
frontmatter `name`, alias, basename, or nest it e.g. `writing/check`), and the
code documents this as deliberate ("subcommand names are reserved, as in any
CLI").
**Steps to Reproduce**:
```bash
# store with skills/check/SKILL.md (name: check-skill) and skills/init/SKILL.md
./skilldozer check          # runs the check subcommand (does NOT resolve the skill)
./skilldozer check-skill    # resolves by name (fallback works)
```
**Suggested Fix**: No code change required if the team accepts reserved
subcommand names; otherwise add a PRD note to §7.2 explicitly reserving
`check`/`init` so the spec and behavior agree.

## Testing Summary

- Total tests performed: ~70 distinct scenarios across happy-path, edge-case,
  adversarial, integration (pi), and workflow categories.
- Passing: the large majority. Every §13 acceptance gate passes; the core
  `pi --skill "$(skilldozer <tag>)"` workflow is verified end-to-end; all four
  §8.3 discovery rules verified in isolation (env / config / sibling /
  walk-up / unconfigured); atomic multi-tag failure, ambiguity, name/alias
  resolution, `--search` (incl. aliases/category per §10), `check` validation
  (all §9 ERROR/WARN rules + 64/65-name and 1024/1025-description boundaries),
  CRLF/BOM/block-scalar frontmatter, `install.sh` symlink install, and
  bash/zsh/fish completion syntax all behave correctly.
- Failing: 2 Major (Issue 1, Issue 2), 5 Minor (Issues 3–7).
- Areas with good coverage: tag resolution & precedence, discovery priority,
  error/exit-code semantics, `check` validation, frontmatter robustness,
  install + symlink resolution, completions.
- Areas needing more attention: `init` output stream discipline (stdout vs
  stderr), value-taking-flag parsing (missing-value handling), and the
  `tag`+`--path` / `init init` exclusivity corners.
