# Verified facts — P1.M6.T15.S1 (shell completions)

Scope: create `completions/skpp.bash`, `completions/_skpp` (zsh), `completions/skpp.fish`.
No Go code. No Go tests. Mirrors mcpeepants completion UX but diverges: skpp is
**manifest-free**, so completions derive tags dynamically by calling `skpp`
itself — never by parsing a sidecar file.

## 1. Exact CLI surface (authoritative — read from main.go `parseArgs`)

Flags (long, short, value-taking?):

| flag            | short | value? | notes                                   |
|-----------------|-------|--------|-----------------------------------------|
| `--version`     | `-v`  | no     |                                         |
| `--help`        | `-h`  | no     |                                         |
| `--path`        | `-p`  | no     |                                         |
| `--list`        | `-l`  | no     |                                         |
| `--all`         | `-a`  | no     |                                         |
| `--file`        | `-f`  | no     | modifier                                |
| `--relative`    | —     | no     | modifier (no short form)                |
| `--no-color`    | —     | no     | modifier (no short form)                |
| `--search <q>`  | `-s`  | YES    | free-text query → NO tag completion here|
| `check`         | —     | —      | bare subcommand token (reserved word)   |

Mutual exclusivity (main.go `exclusivityError`, PRD §6.3): tags+(--list/--search/
--all), check+tags, check+mode are all exit-2. `check` is matched as the EXACT
token "check" anywhere in argv; a nested skill `writing/check` still resolves.

## 2. Dynamic tag source — `skpp --relative --all` (THE key decision)

Verified output (repo, one example skill):
```
$ skpp --all
/home/dustin/projects/skpp/skills/example
$ skpp --relative --all
example
```

Use `--relative --all` (NOT `--all`):
- Emits canonical tags one per line, sorted by tag, e.g. `example`, `writing/reddit`.
- Fully spec-compliant: PRD §6.2 says `--relative` combines with `--all`.
- PRD §14 says "invoking `skpp --all` and offers the printed paths' basename-or-
  relTag" — the *intent* is to get the tags; `--relative --all` is the clean way.
- `--all` (absolute paths) would force basename extraction → nested tags like
  `writing/reddit` collapse to `reddit`, AMBIGUOUS when two skills share a
  basename. `--relative --all` keeps nested tags intact and unambiguous.

Wrap in `2>/dev/null`: a broken/missing `skpp` degrades to "offer no tags" instead
of dumping errors into the completion stream. skpp is a <5ms Go binary, so calling
it per completion request is cheap (mcpeepants re-parsed servers.json each time too).

## 3. mcpeepants reference (mirror UX, diverge on data source)

Files read in `~/projects/mcpeepants/`:
- `completion.sh` — bash. Uses `_init_completion || return` + `compgen -W`.
  GOTCHA: `_init_completion || return` FAILS entirely (offers nothing) if the
  `bash-completion` package is absent (common on minimal Linux + macOS default
  bash). skpp must provide a manual fallback to `COMP_WORDS`/`COMP_CWORD`.
- `_mcpeepants` — zsh. `#compdef mcpeepants get-server-config.sh`, `_arguments -C`
  with `'1: :->first_arg'` + `'*: :->args'`, value flags declared
  `'--search[...]:search query:'`, tags via `_values -s , servers $servers`.
- `mcpeepants.fish` — fish. `complete -c mcpeepants -f`, per-flag `-d`, hardcoded
  per-server `-a name -d desc` lines (static, because servers.json is fixed).
  skpp tags are DYNAMIC → use `complete -a '(skpp --relative --all 2>/dev/null)'`.

DIVERGENCE summary: mcpeepants parses a static manifest; skpp calls the binary.
mcpeepants offers servers separated by `-s ,` (multiple selection); skpp offers
tags one-per-line (positional `<tag> [<tag>...]`).

## 4. Filename conventions (zsh autoload is load-bearing)

- `completions/skpp.bash` — sourced.
- `completions/_skpp` — zsh autoload. The leading `_` + the `#compdef skpp` first
  line bind the function `_skpp` to the command `skpp`. File MUST be named `_skpp`
  (zsh finds completion functions by `_<cmdname>` on `$fpath`).
- `completions/skpp.fish` — sourced; fish finds it by command name.

## 5. Environment for validation (all three shells present)

```
bash: /usr/bin/bash
zsh:  /usr/bin/zsh
fish: /usr/bin/fish
go:   go1.26.4
```
Completions call bare `skpp`, so the binary must resolve on PATH during testing.
Options: run `./install.sh` first (symlinks into ~/.local/bin), or
`PATH=$PWD:$PATH` for ad-hoc probes. skpp's own skills-dir resolution (§8) then
finds `skills/` via the sibling-of-binary rule — completions add NO new resolution
requirements beyond "skpp is installed correctly".

## 6. `check`-after-words footgun (handle in all three shells)

`check` is exclusive (check+tags → exit 2). Naive completions that keep offering
tags after `skpp check ` invite the user into a guaranteed error. Suppress tags
once `check` is seen:
- bash: scan `${COMP_WORDS[@]}` for `check`; if seen, return with empty COMPREPLY.
- zsh: `(( ${words[(I)check]} ))` in the `rest` state.
- fish: condition `-n 'not __fish_seen_subcommand_from check'` on the tag line.

(Optionally also suppress after --list/--all; PRD §14 says keep simple, so the
`check` guard is the one mandatory footgun-handle. Listed as optional extra.)

## 7. Scope boundary (do NOT touch other tasks' files)

- install.sh (T13, COMPLETE) already prints: "Shell completions are not installed
  by this script — see task P1.M6.T15.S1." → the pointer exists; no install.sh
  edit REQUIRED by this subtask.
- README.md (T14, COMPLETE) has no completions section yet. PRD §14 says completions
  may be deferred and "Provide an install.sh step OR a short note in README".
  A README subsection + an install.sh copy-step are OPTIONAL enhancements, to be
  coordinated deliberately if added (they touch a completed task's file). The
  acceptance bar for THIS subtask = the 3 completion files work.
- NEVER ship a second skill (PRD §2.4). Any temp skill created for nested-tag
  validation MUST be removed before commit.
