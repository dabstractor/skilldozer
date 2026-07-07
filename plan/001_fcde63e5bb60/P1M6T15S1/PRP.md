# PRP — P1.M6.T15.S1: Shell completions (bash / zsh / fish) with dynamic tags

> **Subtask:** P1.M6.T15.S1 — build-order step 8 (packaging/docs cluster, final
> non-acceptance piece). Create **three** shell-completion files under
> `completions/` that complete `skpp`'s flags, the `check` subcommand, and — the
> heart of the task — **dynamic skill tags derived from disk** by calling `skpp`
> itself (manifest-free, per PRD §2.1 / §14).
>
> **Scope:** CREATE exactly **three** new files — `completions/skpp.bash`,
> `completions/_skpp` (zsh autoload), `completions/skpp.fish`. No Go code. No
> package changes. No `.gitignore` change. No edits to `install.sh` (T13) or
> `README.md` (T14) — both are COMPLETE; a pointer to this task already exists in
> `install.sh`. An optional README subsection / install.sh copy-step is noted
> below but is NOT required for this subtask's acceptance (PRD §14 explicitly
> allows the completions integration to be the one deferred deliverable).
>
> **Why this is small-but-subtle:** the three files are ~40–70 lines of shell
> each, but TWO decisions are load-bearing and easy to get wrong:
>   1. **Tags must come from `skpp --relative --all`** (canonical tags, e.g.
>      `writing/reddit`), NOT from `skpp --all` (absolute paths) with basename
>      extraction — basename collapses nested tags and makes them ambiguous.
>   2. **The completions must NOT parse a manifest** (there is none — PRD §2.1).
>      They call the binary, which reads disk. This is the key divergence from the
>      mcpeepants reference (which parses a static `servers.json`).
> A third footgun — offering tags after `check` (an exclusive subcommand, §6.3) —
> is handled in all three shells so completion never invites a guaranteed exit-2.

---

## Goal

**Feature Goal**: After sourcing/copying the right file, a user typing
`skpp <TAB>` in bash, zsh, or fish gets: flag completion when the token starts
with `-`, the `check` subcommand and all skill **tags** as positional
completions, and free-text (no completion) after `--search`/`-s`. Tags are
**dynamic** — they reflect the current on-disk `skills/` tree on every
completion, with no index file and no hardcoding.

**Deliverable**: Three new files under `completions/`:
- `completions/skpp.bash` (~55 lines) — bash `complete -F` function.
- `completions/_skpp` (~55 lines) — zsh `#compdef skpp` autoload function.
- `completions/skpp.fish` (~20 lines) — fish `complete -c skpp` directives.

No other files change.

**Success Definition** (all must hold with `skpp` on PATH and the example skill
present):
- In each shell, `skpp -<TAB>` offers every flag in the §6.1/§6.2 matrix
  (`--version -v --help -h --path -p --list -l --all -a --file -f --relative
  --no-color --search -s`).
- `skpp exa<TAB>` completes to `skpp example`.
- `skpp --search <TAB>` offers **nothing** (free-text query).
- `skpp check <TAB>` offers **nothing** (check is exclusive — no footgun).
- After temporarily adding `skills/writing/reddit/SKILL.md`,
  `skpp writing/<TAB>` completes to `writing/reddit` (proves `--relative --all`
  preserves nested tags; temp skill is removed before commit — PRD §2.4).
- No file parses a manifest; every shell derives tags via a `skpp` invocation.

## User Persona

**Target User**: A pi operator who has installed `skpp` and types it many times a
day inside `pi --skill "$(skpp <tag>)"`. They want tab-completion for tags they
half-remember and for the flag matrix.

**Use Case**: `pi --skill "$(skpp writ<TAB>`" → completes to `writing/reddit`
(or the unique match), so they don't have to remember exact tags.

**Pain Points Addressed**:
- "I don't remember the exact tag / whether it's nested." → completion shows the
  canonical tag from disk.
- "I don't remember the short flag for `--file`." → `skpp -<TAB>` lists `-f`.
- "mcpeepants completion worked but hardcoded servers." → skpp can't (no
  manifest), so completion must be dynamic and stay correct as skills are added.

## Why

- **PRD §14 — the authoritative spec for this task.** "They complete
  subcommands/flags after `skpp` / `skpp --`" and "**Tags** by invoking
  `skpp --all` (cheap, disk-derived) for positional completion. Keep them
  simple: a function that runs `skpp --all` and offers the printed paths'
  basename-or-relTag." This PRP supplies the per-shell idiom the PRD leaves
  implicit, and refines the tag source from `--all`+basename to
  `--relative --all` (relTag directly — see "Known Gotcha 1").
- **PRD §2.1 (manifest-free) is the hard constraint.** Completions must NOT read
  `skills.json`/`skills.yaml` (there is none) or any cached index. Tags come
  from `skpp`, which rebuilds the index from disk every call (fast — a directory
  walk of a small tree by a <5ms Go binary).
- **PRD §6.3 mutual exclusivity.** `check` is exclusive (check+tags → exit 2).
  Completions suppress tag offers after `check` so they never invite a
  guaranteed error.
- **Cohesion with prior/future work items:** M1–M5 (the Go binary, full §6 CLI
  surface) are landed and green; the example skill (T12), install.sh (T13), and
  README (T14) are complete. This task packages an already-stable CLI surface —
  it must NOT change any binary behavior. T16.S1 (acceptance suite) will run
  PRD §13; the completion files must not interfere (they are opt-in, sourced
  only by users). The completion flag list is frozen to main.go's `parseArgs` —
  if a later task adds a flag, the three files must be updated in lockstep
  (noted in Integration Points).

## What

User-visible behavior in each shell (after the file is sourced/installed):

1. `skpp <TAB>` → offers skill tags + the word `check` (first positional only).
2. `skpp -<TAB>` or `skpp --<TAB>` → offers the full flag matrix.
3. `skpp example <TAB>` → offers more tags (multi-tag is allowed, §6.1).
4. `skpp --search <TAB>` → offers nothing (free-text query).
5. `skpp check <TAB>` → offers nothing (check is exclusive).
6. `skpp -f <TAB>` / `skpp --relative <TAB>` / `skpp --no-color <TAB>` → these
   are modifiers; a tag is still expected, so tags are offered.
7. Tags update live: add a skill under `skills/`, and the next `<TAB>` includes
   it — no cache to bust, no manifest to rebuild.

### Success Criteria

- [ ] `completions/skpp.bash`, `completions/_skpp`, `completions/skpp.fish` exist.
- [ ] `bash -n completions/skpp.bash`, `zsh -n completions/_skpp`,
      `fish -n completions/skpp.fish` all exit 0 (syntax clean).
- [ ] Bash: simulated `COMP_WORDS=(skpp ex) COMP_CWORD=1` → `COMPREPLY` contains
      `example`; `COMP_WORDS=(skpp --)` → `COMPREPLY` contains the flag set.
- [ ] Fish: `fish -c 'source completions/skpp.fish; complete -C "skpp exa"'`
      prints `example`; `complete -C "skpp --search "` prints nothing.
- [ ] Fish: `complete -C "skpp check "` prints nothing (check suppresses tags).
- [ ] Zsh: `#compdef skpp` first line present; function loads under `compinit`
      without error; offers tags via `compadd`.
- [ ] No file reads or parses any manifest/index; each derives tags via a `skpp`
      invocation wrapped in `2>/dev/null`.
- [ ] With a temp `skills/writing/reddit/SKILL.md`, a nested-tag completion
      (`writing/`) resolves to `writing/reddit` (relTag preserved); temp skill
      removed afterward.
- [ ] `git status` shows ONLY the three new files under `completions/` — no
      changes to `main.go`, `install.sh`, `README.md`, `.gitignore`, `skills/`,
      or any `internal/` package.

## All Needed Context

### Context Completeness Check

_If someone knew nothing about this codebase, would they have everything needed
to implement this successfully?_ **Yes.** The exact flag list, the tag-source
command, and the per-shell idioms are all given verbatim in "Implementation
Blueprint". The reference files (`completion.sh`, `_mcpeepants`, `mcpeepants.fish`)
show the proven structure to mirror; the divergences (dynamic tags, `_init_completion`
fallback, `check`-suppression) are spelled out with reasons. No Go knowledge is
required; the only codebase file that must be read is `main.go` (to confirm the
flag list hasn't drifted — quoted inline below).

### Documentation & References

```yaml
# MUST READ — the authoritative spec for this task
- file: PRD.md
  section: "§14 'Shell completions' — complete subcommands/flags + tags via
            `skpp --all`; keep simple; tags offer basename-or-relTag."
  why: "This is the source of truth for WHAT the completions do. PRD §14 also
        permits deferring the install/README integration, so the 3 files alone
        satisfy the deliverable."
  critical: "PRD §14 says 'invoking `skpp --all`'. The PRP refines this to
             `skpp --relative --all` (see Gotcha 1) — that is the clean way to
             get canonical tags and is fully within §6.2 (modifiers combine with
             --all). Do NOT parse `--all`'s absolute paths by basename."

# MUST READ — why tags come from the binary, not a file (hard constraint)
- file: PRD.md
  section: "§2.1 (manifest-free) and §17 (do NOT add an index file)."
  why: "There is NO skills.json/yaml. Completions must call `skpp`, which reads
        disk. Hardcoding tags (like mcpeepants hardcodes servers) would go stale
        instantly and violate the manifest-free constraint."
  critical: "mcpeepants' static `-a name -d desc` fish lines are WRONG for skpp.
             Use `complete -a '(skpp --relative --all ...)'` (dynamic)."

# VERIFY-AGAINST — the exact flag surface to complete (authoritative)
- file: main.go
  section: "parseArgs() switch (≈ lines 160–215) — the complete, frozen flag set."
  why: "Every flag the completion must offer is a `case` here. If a future task
        adds a flag, update all three completion files in lockstep."
  critical: "Value-taking flag is ONLY `--search`/`-s` (consumes next token).
             `check` is a bare positional token (reserved subcommand). `--relative`
             and `--no-color` have NO short forms. Do not invent short flags."

# READ — the bash template to mirror (structure) and diverge from (data source)
- file: ~/projects/mcpeepants/completion.sh
  why: "`_init_completion` + `compgen -W` + `case "$prev" in --search)` structure."
  pattern: "COMPREPLY=(\$(compgen -W \"\$cands\" -- \"\$cur\")); complete -F _fn cmd"
  gotcha: "mcpeepants uses `_init_completion || return` — if the bash-completion
           package is ABSENT (minimal Linux, macOS default bash) this makes
           completion silently offer NOTHING. skpp MUST add a manual fallback to
           COMP_WORDS/COMP_CWORD. Also: mcpeepants parses servers.json; skpp calls
           `skpp --relative --all` instead."

# READ — the zsh template to mirror
- file: ~/projects/mcpeepants/_mcpeepants
  why: "`#compdef cmd` header, `_arguments -C` with `'1: :->first'` + `'*: :->args'`,
        value flags declared `'--search[...]:search query:'`, the `case "$state"`."
  pattern: "tags=(\${(f)\"\$(skpp --relative --all 2>/dev/null)\"}); compadd -- \"\$tags[@]\""
  gotcha: "Filename MUST be `_skpp` and the first line MUST be `#compdef skpp` —
           zsh binds completion functions by `_<cmdname>` on `$fpath`. A file named
           `skpp.zsh` will NOT autoload."

# READ — the fish template to mirror
- file: ~/projects/mcpeepants/mcpeepants.fish
  why: "`complete -c cmd -f`, per-flag `-s short -l long -d desc`, `-r` for value flags."
  pattern: "complete -c skpp -a '(skpp --relative --all 2>/dev/null)' -d 'skill tag'"
  gotcha: "mcpeepants hardcodes one `-a server -d desc` line per server. skpp tags
           are dynamic → ONE `-a '(...)'` line with command substitution. Use
           `__fish_seen_subcommand_from` / `__fish_is_first_arg` (real builtins)."

# REFERENCE — verified facts + design decisions for this task
- docfile: plan/001_fcde63e5bb60/P1M6T15S1/research/verified_facts.md
  why: "The exact flag table, the `--relative --all` output proof, the
        `_init_completion`-absence gotcha, the `check`-suppression footgun, the
        scope boundary (install.sh/README already done), and the validation env
        (bash/zsh/fish all installed)."

# VERIFY-AGAINST — the tag-source command behavior (already implemented)
- file: main.go
  section: "skillPath() + the `--all` branch — confirms `--relative` combines
            with `--all` and emits RelTag (the canonical tag) one per line."
  why: "`skpp --relative --all` is already correct (M3.T8.S2 landed). Completions
        consume its stdout; no binary change needed."
```

### Current Codebase tree (relevant slice)

```bash
skpp/
├── PRD.md                 # §14 = the spec for this task; §2.1 = manifest-free
├── main.go                # parseArgs() = the frozen flag set to complete
├── install.sh             # (COMPLETE) already prints a pointer to T15.S1
├── README.md              # (COMPLETE) no completions section (optional to add)
├── skills/example/SKILL.md# the tag `example` completions must resolve to
└── (no completions/ dir)  # ← THIS TASK CREATES IT + 3 files
```

### Desired Codebase tree with files to be added

```bash
skpp/
└── completions/
    ├── skpp.bash          # NEW. bash `complete -F _skpp_completion skpp`.
    │                      #   Dynamic tags via `skpp --relative --all`;
    │                      #   `_init_completion` w/ manual fallback; check-suppress.
    ├── _skpp              # NEW. zsh autoload. `#compdef skpp`; `_arguments -C`;
    │                      #   tags via `${(f)"$(skpp --relative --all)"}` + compadd.
    └── skpp.fish          # NEW. fish. `complete -c skpp -f`; flags with -d;
                           #   `--search -s -r`; tags via `complete -a '(...)'`.
```

### Known Gotchas of our codebase & Library Quirks

```bash
# CRITICAL (1): TAG SOURCE = `skpp --relative --all`, NOT `skpp --all`+basename.
#   `--all` prints ABSOLUTE paths (/home/.../skills/writing/reddit); taking the
#   basename yields `reddit`, COLLAPSING nested tags. Two skills `a/reddit` and
#   `b/reddit` would both offer `reddit` — ambiguous, and wrong (canonical tag is
#   `a/reddit`). `--relative --all` prints the relTag (canonical tag) directly:
#     $ skpp --relative --all
#     writing/reddit
#   This is spec-compliant (§6.2: --relative combines with --all) and is the
#   clean realization of PRD §14's "basename-or-relTag" intent.
TAGS_CMD="skpp --relative --all 2>/dev/null"

# CRITICAL (2): NO MANIFEST PARSING (PRD §2.1/§17). There is no skills.json.
#   mcpeepants parses servers.json / hardcodes server lines. skpp CANNOT — the
#   store is inferred from disk. Every completion must run the binary.

# CRITICAL (3): wrap the skpp call in `2>/dev/null`. If skpp is missing/broken/
#   can't find the skills dir, the completion must degrade to "no tags offered",
#   NOT dump a stderr error into the shell's completion menu.
TAGS_CMD="skpp --relative --all 2>/dev/null"   # errors swallowed → empty list

# CRITICAL (4): `check` is an EXCLUSIVE subcommand (§6.3: check+tags → exit 2).
#   Naive completions keep offering tags after `skpp check `, inviting a
#   guaranteed error. Suppress tags once `check` is seen (all three shells).

# GOTCHA (5): bash `_init_completion || return` silently disables completion if
#   the `bash-completion` package is absent. Provide a manual fallback:
#     _init_completion 2>/dev/null || {
#       COMPREPLY=(); cur="${COMP_WORDS[COMP_CWORD]}"
#       prev="${COMP_WORDS[COMP_CWORD-1]}"; cword=$COMP_CWORD; words=("${COMP_WORDS[@]}")
#     }

# GOTCHA (6): zsh file MUST be named `_skpp` (not `skpp.zsh`), first line MUST be
#   `#compdef skpp`. zsh discovers completion funcs as `_<cmd>` on `$fpath`.

# GOTCHA (7): value-taking flag is ONLY `--search`/`-s`. After it, offer NOTHING
#   (the query is free text). Do NOT complete tags there.

# GOTCHA (8): `--relative` and `--no-color` have NO short forms. Do not invent
#   `-r`/`-c`. The short set is exactly: -v -h -p -l -a -f -s.

# GOTCHA (9): completions call bare `skpp`, which does its OWN §8 skills-dir
#   resolution at completion time. So completions "just work" iff skpp is
#   installed correctly (symlink install → sibling-of-binary finds skills/).
#   No NEW resolution requirement is introduced; a misconfigured skpp just yields
#   no tags (graceful, via 2>/dev/null).

# GOTCHA (10): the `compgen`/`compadd`/`complete -a` outputs must contain NO
#   spaces (tags and flags never contain spaces). The classic unquoted-split
#   idiom `COMPREPLY=($(compgen ...))` is therefore safe (shellcheck SC2207 is
#   acceptable; optionally `mapfile -t COMPREPLY < <(compgen ...)`).
```

## Implementation Blueprint

No data models — these are shell scripts. Each file is self-contained and
follows the same 3-part shape: (a) flag set, (b) value-flag guard for
`--search`/`-s`, (c) positional tag completion from `skpp --relative --all`.

### The shared flag set (frozen to main.go `parseArgs`)

Every file offers exactly these (long, short):

```
--version -v   --help -h   --path -p   --list -l   --all -a
--file -f   --relative   --no-color   --search -s (value)   check (subcommand)
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CREATE completions/skpp.bash
  - IMPLEMENT: a `_skpp_completion` function + `complete -F _skpp_completion skpp`.
    Structure:
      1. local cur prev words cword; `_init_completion 2>/dev/null || {manual fallback}`
         (Gotcha 5 — fallback keeps completion working without bash-completion pkg).
      2. `case "$prev" in --search|-s) return 0 ;; esac`  (Gotcha 7 — free text).
      3. `if [[ "$cur" == -* ]]; then COMPREPLY=($(compgen -W "<flag set>" -- "$cur")); return 0; fi`
      4. Scan `${COMP_WORDS[@]:1:cword-1}`: if `check` seen → `return 0` (Gotcha 4);
         set have_pos if any non-flag positional seen.
      5. `tags=$(skpp --relative --all 2>/dev/null)`; `cands="$tags"`;
         `(( have_pos == 0 )) && cands="$cands check"` (check only as first token).
      6. `COMPREPLY=($(compgen -W "$cands" -- "$cur")); return 0`.
  - FOLLOW pattern: ~/projects/mcpeepants/completion.sh (structure), DIVERGING on
    data source (Gotcha 2) and `_init_completion` fallback (Gotcha 5).
  - NAMING: function `_skpp_completion`; register `complete -F _skpp_completion skpp`.
  - PLACEMENT: completions/skpp.bash.
  - GUARDRAILS: all skpp calls wrapped in `2>/dev/null`; never parse a file.

Task 2: CREATE completions/_skpp  (zsh autoload)
  - IMPLEMENT: first line `#compdef skpp`; function `_skpp() { ... }; _skpp "$@"`.
    Structure:
      1. `local -a tags; tags=(${(f)"$(skpp --relative --all 2>/dev/null)"})`
         (Gotcha 1 — relTag; `${(f)...}` splits on newlines into an array).
      2. `local -a flags=( '--version[desc]' '-v[desc]' ... '--search[..]:query:' '-s[..]:query:' )`
         — every flag with a `[description]`; `--search`/`-s` end with `:query:`
         (value-taking; zsh then routes the value slot away from $state).
      3. `_arguments -C "$flags[@]" '1: :->first' '*: :->rest' && return 0`
      4. `case "$state" in first) compadd -- "$tags[@]" check ;;`
            (plain `compadd` with `--` + the array + the literal word `check`; check
            offered only as the first positional. NOTE: do NOT use `compadd -X` —
            its group-label semantics are subtle and pty-only to verify; plain
            `compadd --` is the bulletproof, universally-correct form, and the
            group label is cosmetic anyway.)
         `rest) if (( ${words[(I)check]} )); then _message 'check takes no further args';`
         `      else compadd -- "$tags[@]"; fi ;; esac`
         (Gotcha 4 — suppress tags after `check`; `${words[(I)check]}` = last match
         index, 0 if absent. `_message` is a standard zsh completion builtin.)
  - FOLLOW pattern: ~/projects/mcpeepants/_mcpeepants (`_arguments -C`, `case $state`).
  - NAMING: file `_skpp`; function `_skpp`; header `#compdef skpp` (Gotcha 6).
  - PLACEMENT: completions/_skpp (NO extension).

Task 3: CREATE completions/skpp.fish
  - IMPLEMENT: a block of `complete -c skpp ...` directives.
      1. `complete -c skpp -f`   (disable file completion).
      2. One `complete -c skpp -s <short> -l <long> -d "<desc>"` per flag;
         `--search`/`-s` get `-r` (require-an-argument → fish treats next word as
         the query, not a positional → tags NOT offered there automatically).
      3. `complete -c skpp -n '__fish_is_first_arg' -a 'check' -d 'Validate every skill on disk'`
         (check only as first arg).
      4. `complete -c skpp -n 'not __fish_seen_subcommand_from check' \
            -a '(skpp --relative --all 2>/dev/null)' -d 'skill tag'`
         (dynamic tags; suppressed once `check` seen — Gotcha 4).
  - FOLLOW pattern: ~/projects/mcpeepants/mcpeepants.fish (directive shape), DIVERGING
    on data source (Gotcha 2 — ONE dynamic `-a '(...)'` line, not hardcoded per tag).
  - NAMING: helpers `__fish_is_first_arg`, `__fish_seen_subcommand_from` are real
    fish builtins (no need to define them).
  - PLACEMENT: completions/skpp.fish.

Task 4: RUN + VERIFY (acceptance; see Validation Loop for full commands)
  - Syntax: `bash -n`, `zsh -n`, `fish -n` on the three files (all exit 0).
  - Behavior: bash `COMP_WORDS` simulation; fish `complete -C "skpp …"` dumps;
    zsh `compinit` load smoke + manual source check.
  - Nested-tag probe: temp `skills/writing/reddit/SKILL.md`; confirm `writing/`
    completes; REMOVE temp skill (PRD §2.4 — exactly one skill ships).
```

### Implementation Patterns & Key Details

```bash
# ---- bash: _init_completion fallback (Gotcha 5) ----------------------------
_skpp_completion() {
    local cur prev words cword
    _init_completion 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"; prev="${COMP_WORDS[COMP_CWORD-1]}"
        cword=$COMP_CWORD; words=("${COMP_WORDS[@]}"); COMPREPLY=()
    }
    case "$prev" in --search|-s) return 0 ;; esac          # free-text query
    if [[ "$cur" == -* ]]; then                            # flag completion
        COMPREPLY=($(compgen -W \
          "--version -v --help -h --path -p --list -l --all -a --file -f --relative --no-color --search -s" \
          -- "$cur")); return 0
    fi
    local w i have_pos=0
    for ((i=1; i<cword; i++)); do
        [[ "${words[i]}" == "check" ]] && return 0         # Gotcha 4: check exclusive
        [[ "${words[i]}" == -* ]] && continue
        have_pos=1
    done
    local tags cands
    tags=$(skpp --relative --all 2>/dev/null)              # Gotcha 1+3: relTag, swallow err
    cands="$tags"
    (( have_pos == 0 )) && cands="$cands check"            # check only as 1st positional
    COMPREPLY=($(compgen -W "$cands" -- "$cur"))
}
complete -F _skpp_completion skpp
```

```zsh
# ---- zsh: dynamic tags + check suppression --------------------------------
#compdef skpp
_skpp() {
    local -a tags
    tags=(${(f)"$(skpp --relative --all 2>/dev/null)"})    # Gotcha 1: canonical tags
    local -a flags=(
        '--version[Print the skpp version]'   '-v[Print the skpp version]'
        '--help[Show this help message]'      '-h[Show this help message]'
        '--path[Print the resolved skills directory]' '-p[Print the resolved skills directory]'
        '--list[Human-readable catalog]'      '-l[Human-readable catalog]'
        '--all[Print every skill directory path, sorted by tag]' '-a[Print every skill dir path]'
        '--file[Print the SKILL.md path instead of the directory]' '-f[Print the SKILL.md path]'
        '--relative[Print paths relative to the skills directory]'
        '--no-color[Disable ANSI color]'
        '--search[Substring search over tag/name/description/keywords]:query:'
        '-s[Substring search over tag/name/description/keywords]:query:'
    )
    _arguments -C "$flags[@]" '1: :->first' '*: :->rest' && return 0
    case "$state" in
        first) compadd -- "$tags[@]" check ;;              # check offered as 1st token only
        rest)  if (( ${words[(I)check]} )); then           # Gotcha 4: check is exclusive
                   _message 'check takes no further arguments'
               else
                   compadd -- "$tags[@]"
               fi ;;
    esac
}
_skpp "$@"
```

```fish
# ---- fish: dynamic tags, check-only-as-first ------------------------------
complete -c skpp -f                                              # no file completion
complete -c skpp -s v -l version                       -d 'Print the skpp version'
complete -c skpp -s h -l help                          -d 'Show this help message'
complete -c skpp -s p -l path                          -d 'Print the resolved skills directory'
complete -c skpp -s l -l list                          -d 'Human-readable catalog (TAG, NAME, DESCRIPTION)'
complete -c skpp -s a -l all                           -d 'Print every skill directory path, sorted by tag'
complete -c skpp -s f -l file                          -d 'Print the SKILL.md path instead of the directory'
complete -c skpp       -l relative                     -d 'Print paths relative to the skills directory'
complete -c skpp       -l no-color                     -d 'Disable ANSI color'
complete -c skpp -s s -l search -r                     -d 'Substring search over tag/name/description/keywords'
complete -c skpp -n '__fish_is_first_arg' -a 'check'   -d 'Validate every skill on disk'
complete -c skpp -n 'not __fish_seen_subcommand_from check' \
    -a '(skpp --relative --all 2>/dev/null)'            -d 'skill tag'   # Gotcha 1+2+4
```

### Integration Points

```yaml
FILES:
  - creates: completions/skpp.bash, completions/_skpp, completions/skpp.fish
  - reads (runtime, via `skpp`): the on-disk skills/ tree (§7 discovery). No file parse.

COMPLETION REGISTRATION (user-driven; NOT automated by this task):
  - bash: `source completions/skpp.bash`, or copy to
           `~/.local/share/bash-completion/completions/skpp` (or /etc/bash_completion.d/skpp).
  - zsh:  copy `completions/_skpp` onto a `$fpath` dir (e.g. ~/.zsh/completions or
           /usr/local/share/zsh/site-functions) and ensure `autoload -U compinit && compinit`.
  - fish: copy `completions/skpp.fish` to `~/.config/fish/completions/skpp.fish`.

CONFIG:
  - env: none new. Completions honor whatever makes `skpp` work (SKPP_SKILLS_DIR etc.)
    indirectly — they just call `skpp`.

LOCKSTEP CONTRACT (future tasks):
  - If a later task adds/renames a flag in main.go parseArgs, the flag set in ALL
    THREE completion files must be updated identically. There is no single source
    of truth the shells can import; keep a comment in each file pointing at
    `main.go parseArgs()` as the canonical list.

OPTIONAL (NOT required for this subtask's acceptance — these touch COMPLETED tasks):
  - README.md: add a short "Completions" subsection under Install (PRD §15 outline
    item 3). Coordinate with T14 if added.
  - install.sh: optionally copy/symlink the three files into the user's completion
    dirs (PRD §14 "Provide an install.sh step OR a short note in README").
    install.sh ALREADY prints a pointer to T15.S1, so this is a nicety, not a need.
  Either change should be a deliberate, separately-reviewed edit; do NOT silently
  modify install.sh/README as part of this subtask.

NO CHANGES TO:
  - main.go, go.mod, any internal/* package, skills/*, .gitignore, install.sh,
    README.md, LICENSE. The binary's behavior is already correct (M1–M5 landed).
```

## Validation Loop

> **Note on testing:** this repo has Go `testing` only; there is NO shell-test
> harness (no bats, no shellcheck config). All three target shells (bash, zsh,
> fish) ARE installed on this machine, so validation is real, runnable shell
> probes — including fish's excellent headless `complete -C "skpp …"` dump.
> OPTIONAL follow-up (out of scope unless requested): add `shellcheck` /
> `shfmt` as a lint gate and a tiny `bats` smoke test. Flagged, not built.
>
> **Prerequisite:** completions call bare `skpp`, so the binary must resolve on
> PATH. Run `./install.sh` first (symlinks `skpp` into `~/.local/bin`), OR
> `export PATH="$PWD:$PATH"` for ad-hoc probes. Verify: `command -v skpp`.

### Level 1: Syntax & Style (Immediate Feedback)

```bash
cd ~/projects/skpp

# Syntax check each file without executing (fish -n == --no-execute).
bash -n completions/skpp.bash && echo "bash syntax OK"
zsh  -n completions/_skpp     && echo "zsh syntax OK"
fish -n completions/skpp.fish && echo "fish syntax OK"

# shellcheck (if installed) — bash file only (shellcheck is bash/sh).
command -v shellcheck >/dev/null 2>&1 && shellcheck completions/skpp.bash || \
  echo "(shellcheck not installed; optional)"

# Files present + named correctly (zsh MUST be `_skpp`, no extension).
test -f completions/skpp.bash && test -f completions/_skpp && test -f completions/skpp.fish \
  && head -1 completions/_skpp | grep -q '^#compdef skpp$' && echo "files + zsh header OK"

# Expected: all three syntax-OK; files present; `_skpp` first line == `#compdef skpp`.
# (shellcheck SC2207 on `COMPREPLY=($(compgen ...))` is acceptable — tags/flags have
#  no spaces. SC2317 on the fallback branch is a false positive.)
```

### Level 2: Component Validation (per-shell headless behavior)

```bash
cd ~/projects/skpp
# Ensure bare `skpp` resolves (completions call it). Pick ONE:
./install.sh >/dev/null 2>&1 || export PATH="$PWD:$PATH"
command -v skpp || { echo "FAIL: skpp not on PATH"; exit 1; }

# --- bash: simulate the completion engine via COMP_WORDS -------------------
# (function defined by sourcing the file)
source completions/skpp.bash

# positional: cur="ex" → expect "example"
COMP_WORDS=(skpp ex); COMP_CWORD=1; _skpp_completion
[[ "${COMPREPLY[*]}" == "example" ]] && echo "bash tag-completion OK" || echo "FAIL: ${COMPREPLY[*]}"

# flag: cur="--" → expect the flag set
COMP_WORDS=(skpp --); COMP_CWORD=1; _skpp_completion
case "${COMPREPLY[*]}" in *--search*--relative*--no-color*) echo "bash flag-completion OK";; *) echo "FAIL flags";; esac

# value-flag guard: prev="--search" → expect NO completion
COMP_WORDS=(skpp --search ""); COMP_CWORD=2; _skpp_completion
[[ -z "${COMPREPLY[*]}" ]] && echo "bash --search no-completion OK" || echo "FAIL: ${COMPREPLY[*]}"

# check suppression: prev="check" → expect NO tags
COMP_WORDS=(skpp check ""); COMP_CWORD=2; _skpp_completion
[[ -z "${COMPREPLY[*]}" ]] && echo "bash check-suppression OK" || echo "FAIL: ${COMPREPLY[*]}"

# --- fish: headless completion dump via `complete -C` ----------------------
# (complete -C "skpp <prefix>" prints exactly what fish would offer)
fish -c 'source completions/skpp.fish; complete -C "skpp exa"' | grep -qx example \
  && echo "fish tag-completion OK" || echo "FAIL fish tag"
fish -c 'source completions/skpp.fish; complete -C "skpp --search "' | grep -q . \
  && echo "FAIL fish --search offered something" || echo "fish --search no-completion OK"
fish -c 'source completions/skpp.fish; complete -C "skpp check "' | grep -q . \
  && echo "FAIL fish check offered tags" || echo "fish check-suppression OK"
fish -c 'source completions/skpp.fish; complete -C "skpp -"' | grep -q -- '--file' \
  && echo "fish flag-completion OK" || echo "FAIL fish flags"

# --- zsh: autoload + compinit load smoke (interactive completion needs a pty) ---
ZTMP=$(mktemp -d)
cp completions/_skpp "$ZTMP/_skpp"
zsh -c "fpath=($ZTMP \$fpath); autoload -U compinit && compinit -u; autoload -U _skpp; \
        whence -w _skpp | grep -q 'function' && echo 'zsh loads OK' || echo 'FAIL zsh load'"
rm -rf "$ZTMP"

# Expected: every line prints "... OK", none "FAIL".
```

### Level 3: Integration Testing (install as a user would + nested tags)

```bash
cd ~/projects/skpp
command -v skpp >/dev/null 2>&1 || { ./install.sh >/dev/null 2>&1 || export PATH="$PWD:$PATH"; }

# --- bash: register via the standard user dir, confirm it took -------------
mkdir -p ~/.local/share/bash-completion/completions
cp completions/skpp.bash ~/.local/share/bash-completion/completions/skpp
# Source in a fresh bash to confirm registration (doesn't pollute this shell).
bash -c 'source ~/.local/share/bash-completion/completions/skpp; complete -p skpp' \
  | grep -q '_skpp_completion' && echo "bash registered OK" || echo "FAIL bash register"

# --- fish: drop into the fish completions dir, confirm it offers tags ------
mkdir -p ~/.config/fish/completions
cp completions/skpp.fish ~/.config/fish/completions/skpp.fish
fish -c 'complete -C "skpp example"' | grep -q . && echo "fish installed OK" || echo "FAIL fish install"

# --- zsh: confirm _skpp is discoverable on fpath --------------------------
mkdir -p ~/.zsh/completions && cp completions/_skpp ~/.zsh/completions/_skpp
zsh -c 'fpath=(~/.zsh/completions $fpath); autoload -U compinit && compinit -u; \
        print -r -- "${_comps[skpp]}"' | grep -q _skpp && echo "zsh registered OK" || echo "FAIL zsh register"

# Expected: all three "... OK". (Leaves the user copies in place — harmless and
# is exactly how a user installs them. Remove if you want a pristine HOME.)

# --- NESTED-TAG PROBE (proves --relative --all preserves nesting) ---------
# PRD §2.4: ship EXACTLY ONE skill. Create a TEMP skill, test, then DELETE it.
mkdir -p skills/writing/reddit
printf -- '---\nname: reddit\ndescription: temporary nested-tag probe; deleted after test\n---\n\n# temp\n' \
  > skills/writing/reddit/SKILL.md

# bash: "writing/" should complete to "writing/reddit"
source completions/skpp.bash
COMP_WORDS=(skpp writing/); COMP_CWORD=1; _skpp_completion
case "${COMPREPLY[*]}" in *writing/reddit*) echo "bash nested-tag OK";; *) echo "FAIL bash nested: ${COMPREPLY[*]}";; esac

# fish: same probe
fish -c 'source completions/skpp.fish; complete -C "skpp writing/"' | grep -qx writing/reddit \
  && echo "fish nested-tag OK" || echo "FAIL fish nested"

# CLEANUP — do NOT commit a second skill (PRD §2.4)
rm -rf skills/writing
git status --short skills/ | grep -q . && echo "FAIL: skills/ dirty after cleanup" || echo "cleanup OK"

# Expected: nested-tag OK in both; skills/ clean afterward.
```

### Level 4: Domain-Specific Validation (end-to-end correctness)

```bash
cd ~/projects/skpp
command -v skpp >/dev/null 2>&1 || export PATH="$PWD:$PATH"

# The completion-offered tag must actually resolve via the binary (round-trip).
TAG=$(fish -c 'source completions/skpp.fish; complete -C "skpp exa"')
[ -n "$TAG" ] && [ -d "$(skpp "$TAG")" ] && echo "offered tag resolves to a real dir OK" || echo "FAIL round-trip"

# Manifest-free proof: grep the three files — none may reference a manifest/index file.
grep -rEi 'skills\.(json|ya?ml)|index\.(json|ya?ml)' completions/ \
  && echo "FAIL: a file references a manifest" || echo "manifest-free OK"

# Verbose-flag drift check: the flag tokens offered by completion must be a subset
# of what main.go actually parses (catches lockstep drift).
OFFERED=$(fish -c 'source completions/skpp.fish; complete -C "skpp --"' | grep -oE '^--?[a-z-]+' | sort -u)
for f in $OFFERED; do grep -q "\"$f\"," main.go || grep -q "\"$f\"" main.go \
  || { echo "FAIL: completion offers '$f' which main.go does not parse"; exit 1; }; done
echo "no flag drift OK"

# Expected: offered tag resolves; no manifest reference anywhere; every offered
# flag exists in main.go parseArgs.
```

## Final Validation Checklist

### Technical Validation

- [ ] Level 1 passed: `bash -n` / `zsh -n` / `fish -n` all clean; `_skpp` header is
      `#compdef skpp`; file named `_skpp` (no extension).
- [ ] Level 2 passed: bash `COMP_WORDS` probes (tag, flag, `--search` guard, `check`
      suppress); fish `complete -C` dumps; zsh compinit load — all OK.
- [ ] Level 3 passed: all three register in their standard user dirs; nested-tag
      probe (`writing/reddit`) OK in bash + fish; temp skill removed, `skills/` clean.
- [ ] Level 4 passed: offered tag resolves via `skpp`; no manifest reference in any
      file; no offered flag is absent from main.go `parseArgs`.

### Feature Validation

- [ ] All Success Criteria in "What" met.
- [ ] `skpp -<TAB>` offers the full §6.1/§6.2 flag matrix (incl. `-v -h -p -l -a -f`
      short forms; `--relative`/`--no-color` with NO short forms).
- [ ] `skpp <TAB>` offers tags + `check` (first positional); tags update live.
- [ ] `skpp --search <TAB>` offers nothing (free text).
- [ ] `skpp check <TAB>` offers nothing (exclusive subcommand — no footgun).
- [ ] Nested tags (`writing/reddit`) complete correctly (relTag preserved).
- [ ] Completions degrade gracefully (no tags, no stderr noise) when `skpp` is
      missing or can't find `skills/`.

### Code Quality Validation

- [ ] Mirrors mcpeepants completion *structure* while correctly DIVERGING
      (dynamic tags via `skpp --relative --all`, not manifest parsing/hardcoding).
- [ ] bash `_init_completion` has a manual fallback (works without bash-completion pkg).
- [ ] zsh file named `_skpp` with `#compdef skpp` (autoload-correct).
- [ ] fish uses real builtins (`__fish_is_first_arg`, `__fish_seen_subcommand_from`).
- [ ] Every `skpp` invocation wrapped in `2>/dev/null`.
- [ ] Each file has a header comment naming the install location for that shell.

### Documentation & Deployment

- [ ] Each file's header shows how to install it for that shell (source path /
      copy target / fpath).
- [ ] A `LOCKSTEP` note in each file points at `main.go parseArgs()` as the
      canonical flag list (so a future flag add updates all three).
- [ ] No claim of auto-installation (install.sh only points here; README optional).
- [ ] No new env vars introduced.

---

## Anti-Patterns to Avoid

- ❌ **Do NOT parse a manifest** (`skills.json`/`index.yaml`). There is none (PRD
  §2.1/§17). Derive tags by calling `skpp --relative --all`.
- ❌ Do NOT use `skpp --all` (absolute paths) + basename for tags — nested tags
  collapse and become ambiguous. Use `--relative --all` (relTag directly).
- ❌ Do NOT hardcode tag names (mcpeepants hardcodes servers; skpp cannot — the
  store is dynamic and manifest-free). Tags must come from a live `skpp` call.
- ❌ Do NOT omit the `2>/dev/null` on the `skpp` call — a broken/missing skpp would
  spew stderr into the completion menu.
- ❌ Do NOT use `_init_completion || return` without a fallback — it silently
  disables bash completion when the `bash-completion` package is absent.
- ❌ Do NOT name the zsh file `skpp.zsh` or omit `#compdef skpp` — zsh will not
  autoload it. The file MUST be `_skpp`.
- ❌ Do NOT offer tags after `--search`/`-s` (free-text query) or after `check`
  (exclusive subcommand → guaranteed exit 2 footgun).
- ❌ Do NOT invent short flags for `--relative`/`--no-color` (they have none).
- ❌ Do NOT silently edit `install.sh` or `README.md` (both COMPLETE) — an
  integration nicety there is optional and must be a deliberate, separate edit.
- ❌ Do NOT commit a second/temp skill used for the nested-tag probe (PRD §2.4).
- ❌ Do NOT change `main.go`, any `internal/` package, `.gitignore`, or the binary.

---

## Confidence Score

**9 / 10.** The deliverable is three small shell files whose full content is given
verbatim above, the load-bearing decisions (tag source = `--relative --all`;
manifest-free; `_init_completion` fallback; `check` suppression) are each proven
and tested with runnable commands, and all three target shells are installed on
this machine so every validation gate is real. The one point of residual risk:
zsh's *interactive* completion (the actual popup) can't be fully exercised
headlessly without a pty — Level 2 verifies the function loads and binds under
`compinit`, and Level 3 verifies it's discoverable on `$fpath`, but the live
menu is confirmed by sourcing in a real zsh (a 10-second manual step). No Go
knowledge is required of the implementer; the only codebase file to read is
`main.go` (to confirm the flag list — quoted inline here).
