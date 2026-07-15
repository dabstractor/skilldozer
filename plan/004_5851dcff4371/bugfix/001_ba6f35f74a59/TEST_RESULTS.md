# Bug Fix Requirements

## Overview

End-to-end validation of the Skilldozer implementation against the PRD, run from a
clean rebuild (`rm -f skilldozer && go build -o skilldozer .`) with isolated
`HOME`/`XDG_CONFIG_HOME`/`SKILLDOZER_*` environments so the host's real
`~/.config/skilldozer/config.yaml` did not contaminate results.

**Overall quality: high.** The core contract is solid and the headline behaviors
all pass:

- `skilldozer <tag>` → one absolute path; unknown tag → nothing on stdout, exit 1
  (the critical `pi --skill "$(skilldozer x)"` contract).
- Multi-tag atomicity (any failure ⇒ nothing on stdout, exit 1).
- Discovery priority env → config → sibling (symlink-aware) → walk-up → unconfigured.
- Tag-resolution precedence (canonical → basename → name → alias) with correct
  ambiguity detection at each step.
- `--check` validation (missing/empty name & description, charset/length, duplicate
  names, >1024-char description) all caught.
- `--search` over tag/name/description/keywords/aliases/category, case-insensitive.
- `--init` interactive + non-interactive, cwd auto-detect, seed-vs-adopt, stdout =
  exactly the store path.
- Namespace safety: a skill literally tagged `check`/`init`/`completions` resolves
  as a tag while `--check`/`--init`/`--completions` run the actions.
- Symlink cycle guard in discovery (no hang/loop).
- `eval "$(skilldozer --completions)"` works for **bash, zsh, and fish** (zsh emits
  the derived eval-safe wrapper — verified no stderr in real zsh 5.9).
- Live `pi --no-skills --skill "$(skilldozer example)"` loads the skill.
- Full PRD §13 acceptance suite passes in a clean isolated environment.

Two Major spec-deviations were found (both explicitly specified in the PRD), plus
a few Minor items. No Critical bugs — the primary use case works reliably.

> Note on state: the zsh eval-safe fix (`runCompletion` → `zshEvalScript`) is
> present in the **working tree** (`main.go` lines ~1330-1332) and all tests pass,
> but it is **uncommitted** as of this test. `git stash` / a fresh `git checkout &&
> go build` would regress zsh to the broken verbatim-autoload output
> (`_skilldozer:31: command not found: _arguments`). Committing the working tree
> resolves this; it is not listed as a separate bug below.

---

## Critical Issues (Must Fix)

None. The core tag→path contract, error semantics, and `pi --skill` integration
all work correctly.

---

## Major Issues (Should Fix)

### Issue 1: `--shell` value completion offers skill tags instead of `bash zsh fish` (all three shells)

**Severity**: Major
**PRD Reference**: §14.2 ("`--completions --shell <name>` — `--shell` takes a fixed
enum (`bash`/`zsh`/`fish`); offer those three words, nothing else.") and §14.4
(completion files are frozen to `parseArgs()`, which accepts `--shell`).

**Expected Behavior**: After `--shell`, tab-completion offers exactly `bash`, `zsh`,
`fish` (and nothing else). `--shell` is a real, documented flag (it appears in
`usageText` OPTIONS and is used in the canonical
`skilldozer --completions --shell fish | source` install idiom).

**Actual Behavior**: None of the three completion files reference `--shell` at all.
As a result `skilldozer --shell <tab>` falls through to the default positional
handler and **offers skill tags** — the opposite of "nothing else".

**Steps to Reproduce** (with `SKILLDOZER_SKILLS_DIR` pointing at a store containing
`example`, `foo`, `writing/reddit`):

```bash
# bash — expect "bash zsh fish", get skill tags:
bash -c 'eval "$(./skilldozer --completions --shell bash)"; \
  COMP_WORDS=(skilldozer --shell ""); COMP_CWORD=2; _skilldozer_completion; \
  echo "COMPREPLY=[${COMPREPLY[*]}]"'
# → COMPREPLY=[example foo writing/reddit]

# fish — expect "bash zsh fish", get skill tags:
fish -c './skilldozer --completions --shell fish | source; \
  for l in (complete -C "skilldozer --shell "); echo $l; end' | head -2
# → example  skill tag
#   foo      skill tag

# zsh — same class of gap (no --shell entry in the _arguments spec).
```

Confirmation that `--shell` is absent from every completion file:

```bash
for f in completions/skilldozer.bash completions/_skilldozer completions/skilldozer.fish; do
  grep -q -- '--shell' "$f" && echo "$f: HAS" || echo "$f: MISSING"
done
# → all three: MISSING --shell
```

**Suggested Fix**: In each completion file, (a) route `--shell`'s value slot to the
fixed enum, and optionally (b) decide whether `--shell` belongs in the advertised
`-<tab>` flag list. Note a PRD internal tension: the §14.6 `skilldozer -<tab>` table
enumerates 13 long flags **without** `--shell`, while §14.2 requires `--shell`'s
value to complete to the enum. The minimum to satisfy §14.2 is the value routing:

- `completions/skilldozer.bash` (~line 38, the `case "$prev"` block): add
  `--shell) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;`.
  Optionally add `--shell` to the `compgen -W` flag list at line 44-46.
- `completions/_skilldozer` (zsh `_arguments` array): add an entry like
  `'--shell[Force a shell for completion]:shell:(bash zsh fish)'` so the value
  slot offers the enum.
- `completions/skilldozer.fish`: add
  `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"`
  (`-x` = no file completion, `-a` = the enum).

Per §14.4, apply identically to all three files and rebuild so the embedded bytes
(`//go:embed`) used by `--completions` stay in lockstep.

---

### Issue 2: A configured store whose directory has vanished silently falls through to a *different* store (wrong skills loaded)

**Severity**: Major
**PRD Reference**: §6.4 ("Skills dir cannot be located / skilldozer is unconfigured
⇒ stderr `skilldozer is not configured; run \`skilldozer --init\`` (**or, if
configured but the dir vanished, the concise reason + fix**), exit `1`.") and the
§6.4 preamble: the entire stderr/non-zero contract exists so that
`pi --skill "$(skilldozer x)"` **fails loudly** instead of passing a wrong path.

**Expected Behavior**: When the config file is present and names a `store:` that does
not exist on disk, skilldozer should print a concise reason + fix to stderr and exit
1 (nothing on stdout) — so `pi --skill "$(skilldozer myskill)"` fails loudly rather
than silently resolving a skill from an unrelated location.

**Actual Behavior**: `findConfig` treats "store names a non-existent dir" as a
**miss** and silently falls through to the sibling-of-binary / walk-up rules. If one
of those exists, skilldozer resolves tags from that *unrelated* store with exit 0
and no warning. The user configured store A; it vanished; skilldozer now hands pi
paths from store B with no indication.

**Steps to Reproduce** (repo binary has a sibling `skills/`; config points at a
non-existent dir):

```bash
echo 'store: /tmp/vanished-store-xyz' > /tmp/v2-cfg.yaml
env -u SKILLDOZER_SKILLS_DIR SKILLDOZER_CONFIG=/tmp/v2-cfg.yaml ./skilldozer --path
# → /home/.../skilldozer/skills
#   (found via sibling of binary)        ← silently used the SIBLING store, not the configured one
#   exit 0                               ← §6.4 implies exit 1 with a reason+fix

env -u SKILLDOZER_SKILLS_DIR SKILLDOZER_CONFIG=/tmp/v2-cfg.yaml ./skilldozer example
# → /home/.../skilldozer/skills/example  ← a path from the WRONG store, no error
```

Contrast: a typo'd `SKILLDOZER_SKILLS_DIR` is *also* silently ignored (PRD
acknowledges this and provides `--path` to detect it), but §6.4 explicitly carves
out the **config** case as deserving an error. The current code collapses the two.

**Location**: `internal/skillsdir/skillsdir.go` `findConfig()`, line ~126:
`return "", 0, false // store path is not an existing dir -> fall through`. This
behavior is locked by `internal/skillsdir/skillsdir_test.go` ("Rule 2 miss: `store`
names a dir that does not exist -> fall through"), so the change is a deliberate
contract decision — but it diverges from §6.4 and undermines the reliability goal.

**Suggested Fix**: Distinguish "config present but its `store` vanished" from "config
missing/unreadable". When `config.Load` succeeds and `f.Store != ""` but the resolved
store is not an existing directory, return a distinct sentinel error from `Find()`
(e.g. `ErrConfiguredStoreMissing`) whose message names the configured path and the
fix (`run \`skilldozer --init\`` or recreate the dir). Have `main` print it to stderr
and exit 1 (nothing on stdout), matching §6.4. Keep the genuine fall-through only for
a missing/unreadable config *file* (§8.1). Update/replace the fall-through test at
`skillsdir_test.go:586` accordingly. (If the team instead intends silent
fall-through, update §6.4 to remove the "configured but the dir vanished" clause so
the PRD and implementation agree.)

---

## Minor Issues (Nice to Fix)

### Issue 3: Value-taking flags missing their value are handled inconsistently; `--search`/`--shell` print help (exit 0) while `--store` exits 2

**Severity**: Minor
**PRD Reference**: §6.1 (flag matrix), §6.4 (error semantics). The code's own comment
is also inaccurate.

**Expected Behavior**: Consistent treatment of a value-taking flag presented without
its value. `--store` (no value) is explicitly handled as a parse error (exit 2).

**Actual Behavior**:
- `skilldozer --store` (no value) → exit 2 (`--store requires a value`). ✓
- `skilldozer --search` (no value) → prints **help to stdout**, exit 0.
- `skilldozer --shell` (no value) → prints **help to stdout**, exit 0.

The `--search` no-value branch leaves `searchMode=false`, so the invocation has no
recognized mode and falls through to the implicit-help default. Worse, the code
comment at `main.go:293` claims this path yields "exit 1", but it actually exits 0.
A user who types `skilldozer --search` expecting a search gets help instead of an
error like `--search requires a query`, and `$(skilldozer --search)` would capture
help text.

**Steps to Reproduce**:
```bash
./skilldozer --search >/dev/null 2>&1; echo "exit=$?"   # → exit 0 (comment says 1)
./skilldozer --store  >/dev/null 2>&1; echo "exit=$?"   # → exit 2
./skilldozer --shell  >/dev/null 2>&1; echo "exit=$?"   # → exit 0
```

**Suggested Fix**: Either make all value-taking flags symmetrical (treat a missing
`--search`/`--shell` value as a parse error → exit 2, mirroring `--store`), or
document the asymmetry and fix the misleading comment at `main.go:293`. The
symmetrical option is more predictable for `$(...)` use.

---

### Issue 4: POSIX `--` end-of-options separator is rejected as an unknown flag

**Severity**: Minor (very obscure — only matters for a skill whose tag begins with
`-`).
**PRD Reference**: §6.1 ("Unknown flags ⇒ error + exit 2"); POSIX convention.

**Expected Behavior**: `skilldozer -- <tag>` should treat `<tag>` as a literal
positional (end-of-options), per the widespread POSIX `--` convention, so a tag that
begins with `-` can be resolved.

**Actual Behavior**: `--` itself is classified as an unknown dashed flag.
```bash
./skilldozer -- -x   # → skilldozer: unknown flag '--', exit 2
```
A skill directory literally named e.g. `-foo` (tag `-foo`) is therefore impossible
to address, since `skilldozer -foo` is parsed as an unknown short-flag bundle and
`skilldozer -- -foo` is rejected.

**Suggested Fix**: In `parseArgs`, recognize a bare `--` token as the end-of-options
separator: set a flag that forces all subsequent tokens (even dashed ones) into
`c.tags`. Low priority — such tag names are pathological — but cheap to support.

---

### Issue 5: README "version is the git-describe value" is only true for the `install.sh` build

**Severity**: Minor (documentation accuracy).
**PRD Reference**: §12.1 (the ldflags build is specific to `install.sh`); README
"Usage" section.

**Expected Behavior**: The README states "Version is the git-describe value (dynamic,
not a fixed string)".

**Actual Behavior**: A plain `go build -o skilldozer .` (the README's own "From
source" path, and `go install`) yields `skilldozer dev`, because the `-ldflags
-X main.version=…` injection is performed only by `install.sh`. Users following the
documented "From source" instructions see `dev`, which reads as contradicting the
README.

**Suggested Fix**: Tighten the README line to "Version is the git-describe value when
built via `./install.sh`; a plain `go build` reports `dev`." (Or omit the
parenthetical.)

---

## Testing Summary

- **Total tests performed**: ~70 distinct scenarios across happy-path, edge-case,
  adversarial, integration, and full §13 acceptance (in isolated environments).
- **Passing**: the vast majority — core contract, discovery, resolution, validation,
  search, init, completions (bash/zsh/fish eval), namespace safety, pi integration,
  and all of PRD §13 in a clean environment.
- **Failing / deviations**: 2 Major (Issue 1, Issue 2), 3 Minor (Issues 3-5).
- **Areas with good coverage**: tag→path resolution & atomicity; §8.3 discovery
  priority incl. symlink/cycle handling; §7.2 precedence & ambiguity; §9 `--check`;
  `--init` flows; exit-code/precedence matrix; namespace safety; embedded-vs-on-disk
  completion byte identity; live `pi` integration.
- **Areas needing more attention**: completion value-slot handling for the
  non-advertised `--shell` flag (Issue 1); the configured-but-vanished-store error
  path (Issue 2); consistent missing-value handling for value-taking flags (Issue 3).

### Reproducibility notes

- All repros assume a fresh `rm -f skilldozer && go build -o skilldozer .` (a stale
  pre-existing binary in the repo can mask the zsh fix — see the Overview note).
- Tests that touch `~/.config/skilldozer/config.yaml` (the host's real config) were
  run with `HOME=/tmp/... XDG_CONFIG_HOME=/tmp/...` and `SKILLDOZER_CONFIG=/tmp/...`
  to isolate them; the host config legitimately overrides the sibling rule
  (config > sibling per §8.3) and is *not* a bug.
