# Verified Facts — P1.M2.T1.S2 (zsh `_skilldozer` + fish `skilldozer.fish`)

All facts below were **empirically verified at PRP-write time** against the live
repo at `/home/dustin/projects/skilldozer`. The bash sibling (P1.M2.T1.S1) is
running in parallel and its file (`completions/skilldozer.bash`) is **already
rewritten** to the new contract — read in full and used here as the reference
pattern (especially the LOCKSTEP comment text and the `--init` directory-routing
shape).

---

## §0 — Scope & sibling boundary (read first)

- **This task edits exactly TWO files:** `completions/_skilldozer` (zsh, 61
  lines) and `completions/skilldozer.fish` (fish, 52 lines). Nothing else.
- **The bash file is DONE** (P1.M2.T1.S1, parallel): long-form-only 13-flag
  list, `--store|--init` dir routing, walk-loop deleted, tag probe byte-identical,
  LOCKSTEP comment cites decisions 19/20. Its `complete -F _skilldozer_completion
  skilldozer` registration + function name are preserved. **Do NOT touch it.**
- **Disjoint from P1.M1.T3.S2** (test-only; never touched `completions/*`).
- **Disjoint from P1.M2.T2.S1** (README; not a completion file).
- `main.go` / `main_test.go` / `internal/*` / `go.mod` / `go.sum` — UNCHANGED.

---

## §1 — The §13 delta acceptance gate (what the test actually checks for fish/zsh)

Verified by reading `plan/004_5851dcff4371/delta_prd.md` §13 (lines 45-65):

```bash
go build -o skilldozer . && echo BUILD_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo COMPLETIONS_BASH_OK
./skilldozer --completions --shell fish 2>/dev/null | grep -q 'complete -c skilldozer' && echo COMPLETIONS_FISH_OK   # ← ONLY fish assertion
...
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-completions' && echo EMBED_HAS_COMPLETIONS_FLAG_OK   # bash-only
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-check'         && echo EMBED_HAS_CHECK_FLAG_OK         # bash-only
! ./skilldozer --completions --shell bash 2>/dev/null | grep -Eq '\-\-version[ ]+-v' && echo LONG_FORM_ONLY_BASH_OK     # bash-only
```

**KEY:** The formal §13 gate exercises `--shell bash` for the `EMBED_HAS_*` /
`LONG_FORM_ONLY_*` checks. For **fish** the only §13 assertion is
`grep -q 'complete -c skilldozer'` (COMPLETIONS_FISH_OK). For **zsh** there is
NO §13 grep beyond the implicit `go test` embed-identity check. **HOWEVER** the
PRD §14.1/§17 guardrails + the CONTRACT (item description §3/§4) REQUIRE
long-form-only + skills-first + `--check`/`--init`/`--completions` present for
ALL THREE files identically (§14.4 lockstep). So this PRP adds **content
spot-checks** (Level 4) that mirror the bash checks to prove faithfulness even
though §13 doesn't formally assert them for fish/zsh.

**The §13 fish gate is currently PASSING** (verified):
```
$ go build -o /tmp/sdz-check . && /tmp/sdz-check --completions --shell fish | grep -q 'complete -c skilldozer' && echo COMPLETIONS_FISH_OK
COMPLETIONS_FISH_OK
```
After this rewrite it still passes (the `complete -c skilldozer -f` global line
and the tag directive both remain).

---

## §2 — `--completions` emit mechanism works for all three shells (load-bearing)

Verified by building and emitting:
```
$ go build -o /tmp/sdz-check .   # exit 0
$ /tmp/sdz-check --completions --shell fish | head -1
# Fish completion for skilldozer.
$ /tmp/sdz-check --completions --shell zsh | head -1
#compdef skilldozer
$ /tmp/sdz-check --completions --shell bash | head -1   # bash sibling already rewrote the file
```
`completionScript(shell)` (main.go:1111) returns the `//go:embed` var per shell.
Embed directives (main.go:54/57/60):
- `//go:embed completions/skilldozer.bash`
- `//go:embed completions/_skilldozer`      ← zsh (note: embeds WITHOUT `all:` — main.go:51 comment)
- `//go:embed completions/skilldozer.fish`

`runCompletion`/`detectShell`/`completionScript` wiring is the responsibility of
P1.M1/P1.M2.T2 and is **already functional** (the gate emits correctly). This
task does NOT touch main.go. The shell-detection note in main_test.go:2956
("completionScript is uncalled until P1.M2.T2.S2 wires runCompletion") is
stale — `--completions` demonstrably works today (verified above). Ignore it.

---

## §3 — `TestEmbeddedCompletionsMatchOnDisk` checks ALL THREE shells (auto-pass on rebuild)

`main_test.go:2995` `TestEmbeddedCompletionsMatchOnDisk` table:
```go
{"bash", "completions/skilldozer.bash"},
{"zsh",  "completions/_skilldozer"},     // ← THIS task's zsh file
{"fish", "completions/skilldozer.fish"}, // ← THIS task's fish file
```
It asserts `completionScript(shell) == os.ReadFile(on-disk)` for all three.
**After editing the two files + `go build`, embedded == on-disk → PASS
automatically.** Do NOT edit the test or the `//go:embed` directives.

**GOTCHA (rebuild):** `//go:embed` reads the files at BUILD time. Edit the file,
then `go build` (or `go test`, which builds) BEFORE running the §13 gate —
otherwise the embedded bytes are stale and `TestEmbeddedCompletionsMatchOnDisk`
FAILS (embed != on-disk) even though the §13 greps may pass.

---

## §4 — THE zsh gotcha: lone `'*: :->args'` covers ALL positionals incl. the FIRST

**Verified empirically from `man zshcompsys`** (zsh 5.9.2 installed):

> ```
> *:message:action
> *::message:action
> *:::message:action
>    This describes how arguments (usually non-option arguments, those not
>    beginning with - or +) are to be completed when neither of the first two
>    forms was provided.  Any number of arguments can be completed in this
>    fashion.
> ```

"when neither of the first two forms was provided" = when you did NOT supply a
`n:message:action` numbered form NOR a `*:optspec:message:action` form. The
contract's replacement uses **only** `'*: :->args'` (empty message + `->args`
action) with NO numbered spec → the `*` covers the FIRST and every subsequent
positional. `compadd -- "$tags[@]"` in the `args)` arm then offers skills on a
bare `<tab>` AND on every later positional. ✅ CONTRACT-FAITHFUL.

The current file (line 41) uses `_arguments -C "$flags[@]" '1: :->first' '*:
:->rest'` (two specs). The contract collapses these to one `*: :->args` spec
because — with subcommands gone — there is no first/rest distinction: every
positional is a skill. This is the single most non-obvious edit; it is CONFIRMED
correct.

`_arguments -C ... && return 0` semantics: when completing a flag (`-*`),
`_arguments` resolves it from `$flags` and returns 0 (short-circuit return);
when completing a positional it sets `$state=args` and returns non-zero so
execution falls through to the `case "$state"`. Keep `-C` and `&& return 0`.

---

## §5 — zsh flag-spec idioms (`:directory:_files`, `:query:`, `compadd`)

- `'--init[First-run setup: pick/create the skills store]:directory:_files'`
  → the `:directory:_files` suffix routes `--init`'s VALUE slot to
  file/directory completion, **identical** to the existing `'--store[...]:
  directory:_files'` (line 38). `_files` is the standard zsh completion action
  for paths. ✅ (Mirrors the bash `--store|--init) compgen -d` routing.)
- `'--search[Substring search...]:query:'` (line 33) — the `:query:` suffix
  marks `--search` value-taking WITHOUT offering a completion (free-text). zsh
  then routes the value slot away from `$state` (no tag completion after it).
  KEEP this line byte-identical (only DELETE its `-s` sibling at line 34).
- `compadd -- "$tags[@]"` — `--` ends options so a tag named `-x` (impossible
  for relTags, which are `[a-z0-9-/]+`) wouldn't be misread as a flag. Safe and
  idiomatic. KEEP the `--`.

---

## §6 — fish gotchas: `-r` dir routing, helper builtins, `-n` guard re-author

**fish 4.8.0 installed.** Verified all three helper functions ship with fish:
```
$ fish -c 'functions __fish_prev_arg_in'     # exists (embedded:functions/__fish_prev_arg_in.fish)
$ fish -c 'functions __fish_is_first_arg'    # exists (embedded:functions/__fish_is_first_arg.fish)
$ fish -c 'functions __fish_seen_subcommand_from'  # exists
```

- **`-r` directory routing (contract §4c):** The existing `complete -c skilldozer
  -l store -d '...' -r` (line 39) ALREADY routes `--store`'s value to
  file/dir completion via `-r`. The contract adds `--init` with the SAME `-r`
  (`complete -c skilldozer -l init -d '...' -r`). **No separate routing directive
  is needed** — fish's `-r` (require-parameter) mode handles it. This mirrors the
  existing, working `--store` precedent exactly. ✅
- **`-n` guard re-author (contract §4e):** OLD guard:
  `not __fish_seen_subcommand_from check init completion; and not __fish_prev_arg_in --search -s`
  NEW guard (drop the subcommand clause AND the `-s` alias from the prev-arg check):
  `not __fish_prev_arg_in --search`
  This suppresses tag offers ONLY when the immediately-preceding arg is `--search`
  (free-text query — no tag completion there). No subcommands remain to guard
  against (they're `--flags` now). ✅
- **`-a '(skilldozer --relative --all 2>/dev/null)'` probe — BYTE-IDENTICAL.**
  Only the `-n` condition changes. The `2>/dev/null` is load-bearing robustness
  (PRD §14.3): a broken/missing binary → empty list, not a stderr dump. ✅
- **DELETE bare subcommand offers (contract §4d):** lines 41-45 (the comment +
  three `complete -c skilldozer -n '__fish_is_first_arg' -a 'check'/'init'/
  'completion'` directives) are removed ENTIRELY. With nothing offering `check`/
  `init`/`completion` as bare words, a bare `<tab>` yields ONLY the dynamic tag
  directive → skills. ✅
- **Global no-file rule:** `complete -c skilldozer -f` (line 14) stays — it's the
  global no-file-completion rule (PRD §14.1 rule 6). `--store -r` and `--init -r`
  intentionally bypass it (the `-r` value-mode override). `--search` deliberately
  does NOT get `-r` (so the global `-f` keeps it offering nothing). KEEP all of
  this — the existing file's comments (lines 25-31, 34-38) document the rationale
  authoritatively.

---

## §7 — `--shell` enum routing is OUT OF SCOPE (consistency with bash sibling)

PRD §14.2 mentions `--completions --shell <name>` should offer `bash`/`zsh`/`fish`.
The bash sibling's PRP (P1.M2.T1.S1) **decided to OMIT** `--shell` from the flag
list and to NOT add enum routing, with documented rationale:
- The CONTRACT for THIS task (item description §3/§4) does NOT mention `--shell`
  routing in either the zsh or fish edit list.
- PRD §14.6's `skilldozer -<tab>` behavior-contract table lists exactly the 13
  flags — NO `--shell` (it's a `--completions` modifier, not a top-level menu flag).
- The §13 gate has no assertion for `--shell` enum routing in any shell.

**DECISION (consistent across all three files):** do NOT add `--shell` to the
flag-offer list and do NOT add `--shell` enum value-routing. This keeps the three
files lockstep-identical in scope and matches the bash sibling. (See PRP §"DESIGN
DECISIONS".)

---

## §8 — The exact flag list (lockstep to main.go parseArgs)

`main.go` usageText (lines 104-117) confirms the canonical long-flag set and their
descriptions. The 13 advertised long flags (no `--shell`):

```
--all --list --search --check --init --store --completions --path --file --relative --no-color --help --version
```

Short aliases (`-a -l -s -f -p -h -v`) EXIST at runtime (main.go:104-117 shows
`--all, -a` etc.) but are NOT advertised by completion (decision 20). `--check`/
`--init`/`--completions` have NO short forms.

For the zsh `_arguments` array and the fish flag matrix, the exact descriptions to
use (lifted from the current files + main.go usageText, contract §3b/§4b):

| flag | zsh description (keep existing text) | fish `-d` (keep existing text) |
|---|---|---|
| `--version` | `Print the skilldozer version` | same |
| `--help` | `Show this help message` | same |
| `--path` | `Print the resolved skills directory` | same |
| `--list` | `Human-readable catalog (TAG, NAME, DESCRIPTION)` | same |
| `--all` | `Print every skill directory path, sorted by tag` | same |
| `--file` | `Print the SKILL.md path instead of the directory` | same |
| `--relative` | `Print paths relative to the skills directory` | same |
| `--no-color` | `Disable ANSI color` | same |
| `--search` | `Substring search over tag/name/description/keywords` | same |
| `--store` | `Non-interactive store path for init` | same |
| `--check` | `Validate every skill on disk` | same |
| `--init` | `First-run setup: pick/create the skills store` | same |
| `--completions` | `Emit the shell completion script for eval` | same |

(The current zsh file DUMBED-DOWN the descriptions for the `-x` short entries,
e.g. `'-l[Human-readable catalog]'` and `'-a[Print every skill dir path]'`. After
deleting the short entries, only the full-text `--long` descriptions remain — no
truncation needed.)

---

## §9 — LOCKSTEP comment text (match the bash sibling, shell-appropriate pronoun)

The bash file's updated LOCKSTEP comment (verified in the rewritten file, the
reference for "match bash's updated comment" per contract §3e/§4f):

```
# LOCKSTEP: the flag set below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this list — and the zsh/fish files —
# identically. There is no shared source of truth the shells can import.
# Flags are long-form-only (decision 20): short aliases stay valid at runtime
# but are not advertised. Updated for --check/--init/--completions (decision 19):
# these were promoted from bare subcommands so the bare positional namespace
# belongs to skill tags — a bare <tab> shows skills, never commands.
```

For **zsh** (`_skilldozer`): keep the file's existing pronoun wording `update this
list — and the bash/fish files —` and append the same two new paragraphs.

For **fish** (`skilldozer.fish`): keep the file's existing pronoun wording `update
this file — and the bash/zsh files —` and append the same two new paragraphs.

---

## §10 — Content spot-checks (Level 4) — mirror the bash gate for fish/zsh

Since §13 only formally checks fish via `grep -q 'complete -c skilldozer'`, these
spot-checks prove faithfulness to PRD §14.1/§17 (long-form-only + skills-first +
the three new flags present) for BOTH files:

```bash
# ZSH — long-form-only: no short-form _arguments entries remain
grep -c "'-[vhplafs]\[" completions/_skilldozer          # expect 0
# ZSH — the three promoted flags present in the _arguments array
grep -q -- "'--check\["  completions/_skilldozer && echo ZSH_CHECK_OK
grep -q -- "'--init\["   completions/_skilldozer && echo ZSH_INIT_OK
grep -q -- "'--completions\[" completions/_skilldozer && echo ZSH_COMPLETIONS_OK
# ZSH — tag probe byte-identical (line 19)
grep -q 'tags=(${(f)"$(skilldozer --relative --all 2>/dev/null)"})' completions/_skilldozer && echo ZSH_PROBE_OK
# ZSH — no leftover subcommand/state 'first'/'rest' dead code
grep -c "check init completion\|->first\|->rest" completions/_skilldozer   # expect 0
# ZSH — the lone *: :->args state spec present
grep -q "'\*: :->args'" completions/_skilldozer && echo ZSH_ARGS_STATE_OK

# FISH — long-form-only: no `-s X` short-option tokens in flag defs
grep -c -- '-s [vhplafs]' completions/skilldozer.fish    # expect 0
# FISH — the three promoted flags present
grep -q -- '-l check'        completions/skilldozer.fish && echo FISH_CHECK_OK
grep -q -- '-l init .*-r'    completions/skilldozer.fish && echo FISH_INIT_OK
grep -q -- '-l completions'  completions/skilldozer.fish && echo FISH_COMPLETIONS_OK
# FISH — tag probe byte-identical (line 52)
grep -q -- "-a '(skilldozer --relative --all 2>/dev/null)'" completions/skilldozer.fish && echo FISH_PROBE_OK
# FISH — the re-authored -n guard (no subcommand clause, no -s)
grep -q "not __fish_prev_arg_in --search'" completions/skilldozer.fish && echo FISH_GUARD_OK
# FISH — no leftover bare subcommand offers
grep -c "__fish_is_first_arg' -a 'check'\|__fish_is_first_arg' -a 'init'\|__fish_is_first_arg' -a 'completion'" completions/skilldozer.fish  # expect 0
```

---

## §11 — Live smoke (optional, proves behavior end-to-end)

```bash
# zsh: bare <tab> → skills only; -<tab> → 13 long flags only
zsh -fc '
  fpath=(completions $fpath)
  autoload -U compinit; compinit -u 2>/dev/null
  # define a stub so the probe returns a known tag without the real binary
  skilldozer() { if [[ "$1$2" == --relative--all ]]; then echo writing/foo; echo example; fi; }
  autoload -U _skilldozer
  compdef _skilldozer skilldozer
  # (interactive completion can't be driven headlessly here; the content
  #  spot-checks in §10 + the man-page confirmation in §4 are the proof.)
'
# fish: emit + parse-load to confirm no syntax errors
fish -c 'source completions/skilldozer.fish; complete -C "skilldozer "' 2>&1 | head
# (fish 4.x `complete -C "cmd "` exercises the completion engine headlessly.)
```

**Note:** zsh completion can't be fully driven headlessly; the authoritative
proofs are (a) `man zshcompsys` confirmation of the `*: :->args` idiom (§4), and
(b) the content spot-checks (§10). Fish CAN be exercised via `complete -C` — run
it to confirm the file parses and offers the stub tag.

---

## §12 — CONCURRENT-IMPLEMENTATION OBSERVATION (recorded at PRP-write time)

While this PRP was being authored, BOTH target files (`completions/_skilldozer`
and `completions/skilldozer.fish`) were observed via `git status`/`git diff` to
ALREADY be in the EXACT target state specified in this PRP — a concurrent process
(either the parallel P1.M2.T1.S1 implementer operating on all three files, or a
separate worker) applied the rewrite byte-for-byte. `git diff` confirms every
edit:

**ZSH** (`completions/_skilldozer`):
- LOCKSTEP comment: +2 paragraphs (decisions 19/20) — matches Task 1 output.
- flags array: 7 short entries (`-v`/`-h`/`-p`/`-l`/`-a`/`-f`/`-s`) removed;
  comments updated; `--check`/`--init[:directory:_files]`/`--completions` added
  before `)` — matches Task 2 output.
- `_arguments -C "$flags[@]" '*: :->args' && return 0` — matches Task 3a.
- case block collapsed to single `args)` arm + `compadd -- "$tags[@]"` — Task 3b.

**FISH** (`completions/skilldozer.fish`):
- LOCKSTEP comment: +2 paragraphs — matches Task 4.
- flag matrix: 6 `-s X` tokens + `-s s` removed; comments updated; `--check`/
  `--init … -r`/`--completions` added after `--no-color` — matches Task 5.
- bare subcommand offers (`__fish_is_first_arg -a 'check'/'init'/'completion'`)
  + comment DELETED — matches Task 6a.
- tag directive `-n` re-authored to `not __fish_prev_arg_in --search` (probe
  byte-identical) — matches Task 6b.

**VALIDATION — all gates PASS against the current on-disk content:**
```
zsh -n completions/_skilldozer          → exit 0
fish -n completions/skilldozer.fish     → exit 0
go build -o /tmp/sdz-v .                → OK
./sdz-v --completions --shell fish | grep -q 'complete -c skilldozer' → COMPLETIONS_FISH_OK
# Level 4 spot-checks:
grep -c "'-[vhplafs]\[" _skilldozer     → 0   (ZSH long-form-only)
grep -c -e '->first' -e '->rest' -e 'check init completion' _skilldozer → 0  (no dead code)
grep -c -e '-s v' ... -e '-s s' skilldozer.fish → 0   (FISH long-form-only)
grep -c "__fish_is_first_arg" skilldozer.fish → 0   (no bare offers)
ZSH_CHECK_OK / ZSH_INIT_OK / ZSH_COMPLETIONS_OK / ZSH_PROBE_OK / ZSH_ARGS_STATE_OK
FISH_CHECK_OK / FISH_INIT_OK / FISH_COMPLETIONS_OK / FISH_PROBE_OK / FISH_GUARD_OK
go test ./...                           → all ok (incl. TestEmbeddedCompletionsMatchOnDisk)
```

**Conclusion:** the PRP's target is CONFIRMED correct and achievable — the disk
state proves it. The PRP's FIND/`oldText` blocks (which transcribe the ORIGINAL
pre-rewrite content) are retained as the authoritative transformation spec; an
implementer should validate-first (Level 4 + `go test ./...`) and treat the edits
as a no-op if the files already match. See the PRP's STATUS block ⚠️ note.
