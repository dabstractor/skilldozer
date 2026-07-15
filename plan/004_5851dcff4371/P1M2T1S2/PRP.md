# PRP — P1.M2.T1.S2: Rewrite `completions/_skilldozer` (zsh) + `completions/skilldozer.fish` (fish)

> **Subtask:** The zsh + fish half of P1.M2.T1 (rewrite the three completion files for decisions 19/20). Converts `completions/_skilldozer` and `completions/skilldozer.fish` from the session-003 model (offered `check`/`init`/`completion` as exclusive first-positional subcommands + advertised both `--long` and `-short` flag forms) to the session-004 model (PRD §14.1): **a bare `<tab>` shows skills only** (no command list), and **`skilldozer -<tab>` shows long-form flags only** (including the three promoted `--check`/`--init`/`--completions`).
>
> **Scope:** TWO existing files only — `completions/_skilldozer` (zsh, 61 lines) and `completions/skilldozer.fish` (fish, 52 lines). No `.go` source change (the `--flag` contract is already Complete per P1.M1.T1; `--completions` already emits correctly). No new files. Zero Go/deps changes — `//go:embed` picks up the new bytes on rebuild. The bash file is **already done** by the parallel sibling P1.M2.T1.S1 (used here as the reference pattern, especially its updated LOCKSTEP comment + `--init` directory-routing shape); `main_test.go`/`internal/*` are P1.M1.T3; README is P1.M2.T2.S1.
>
> **STATUS (verified at PRP-write time):** read both target files in full with exact line numbers; read `completions_change_map.md` §2 (zsh) + §3 (fish); read `delta_prd.md` §13 (the §2 acceptance gate — fish has ONE assertion, zsh none, beyond `go test` embed-identity); read `main_test.go::TestEmbeddedCompletionsMatchOnDisk` (table covers all three shells); read the already-rewritten `completions/skilldozer.bash` (the reference); confirmed via `main.go` usageText that all 13 long flags exist in `parseArgs()`. **Empirically verified:** `go build` + `--completions --shell fish/zsh/bash` all emit correctly; the §13 fish gate (`COMPLETIONS_FISH_OK`) passes today; the zsh `*: :->args` idiom is confirmed by `man zshcompsys` (a lone `*:` spec covers the FIRST and all subsequent positionals); all fish helpers (`__fish_prev_arg_in`, `__fish_is_first_arg`, `__fish_seen_subcommand_from`) ship with fish 4.8.0 (installed); the `-r` directory-routing pattern is proven by the existing `--store -r` directive. zsh 5.9.2 + fish 4.8.0 are both installed for smoke testing.
>
> **⚠️ CONCURRENT-IMPLEMENTATION OBSERVATION (verified at PRP-write time):** While this PRP was being authored, BOTH target files were observed to ALREADY be in the EXACT target state specified below. `git diff completions/_skilldozer completions/skilldozer.fish` shows every edit in Tasks 1-6 (zsh + fish) applied **byte-for-byte** (LOCKSTEP comments; long-form-only flag arrays; the `'*: :->args'` single state; the deleted fish subcommand offers; the re-authored `-n 'not __fish_prev_arg_in --search'` guard; byte-identical probes). **All validation gates PASS against the current on-disk content** — `zsh -n` / `fish -n` exit 0; the §13 `COMPLETIONS_FISH_OK` gate; every Level 4 spot-check (`ZSH_CHECK_OK`/`ZSH_INIT_OK`/`ZSH_COMPLETIONS_OK`/`ZSH_PROBE_OK`/`ZSH_ARGS_STATE_OK`/`FISH_CHECK_OK`/`FISH_INIT_OK`/`FISH_COMPLETIONS_OK`/`FISH_PROBE_OK`/`FISH_GUARD_OK`; all `want 0` counts return 0); and `go test ./...` fully green incl. `TestEmbeddedCompletionsMatchOnDisk`.
>
> **👉 IMPLEMENTER ACTION:** pick up this PRP by FIRST running the **Validation Loop Level 4 spot-checks + `go test ./...`**. **If they pass, the work is ALREADY COMPLETE — do NOT edit anything** (the FIND/`oldText` blocks in the Implementation Tasks describe the ORIGINAL pre-rewrite content and are retained as the authoritative transformation spec for audit/reference; applying them now is a no-op since the files already match REPLACE). **Only if a spot-check FAILS** (concurrent work reverted or diverged) should you apply the FIND→REPLACE edits exactly as written — they are correct, verified, and pin every line.

---

## Goal

**Feature Goal**: Rewrite the zsh and fish completion files so they implement PRD §14.1's skills-first / long-form-only contract, in lockstep with the already-rewritten bash file (P1.M2.T1.S1):
- **ZSH** (`completions/_skilldozer`): drop all 7 short-form `_arguments` entries (`-v`/`-h`/`-p`/`-l`/`-a`/`-f`/`-s`); add `--check`/`--init[:directory:_files]`/`--completions` to the flags array; collapse the `'1: :->first' '*: :->rest'` two-state split into a single `'*: :->args'` state whose `case` arm does `compadd -- "$tags[@]"` (so every positional offers skills, no subcommand exclusivity guard); keep the tag probe byte-identical; update the LOCKSTEP comment to match bash.
- **FISH** (`completions/skilldozer.fish`): drop all `-s X` short-option tokens from the flag matrix; add `--check` (plain), `--init … -r` (dir routing via `-r`, mirroring `--store`), `--completions` (plain); DELETE the three `__fish_is_first_arg -a 'check'/'init'/'completion'` bare-word offers; re-author ONLY the `-n` guard of the tag directive to `not __fish_prev_arg_in --search` (drop the `__fish_seen_subcommand_from check init completion` clause AND the `-s` from `--search -s`); keep the `-a '(skilldozer --relative --all 2>/dev/null)'` probe byte-identical; update the LOCKSTEP comment to match bash.

A bare `skilldozer <tab>` then yields skills only; `skilldozer -<tab>` yields the 13 long flags only; `skilldozer --init <tab>` / `--store <tab>` yields directories; `skilldozer --search <tab>` yields nothing.

**Deliverable**: Two rewritten files (`completions/_skilldozer`, `completions/skilldozer.fish`) passing the §13 fish gate (`COMPLETIONS_FISH_OK`), the content spot-checks (long-form-only + the three new flags present + probes byte-identical for both files), `go build`, and `go test ./...` (incl. `TestEmbeddedCompletionsMatchOnDisk`, which auto-satisfies for zsh+fish because `//go:embed` reads the rewritten files at build time).

**Success Definition**: `go build` succeeds (re-embeds); the §13 `COMPLETIONS_FISH_OK` grep prints; `go test ./...` is green; on both files `grep -c` for short-form advertisements returns 0; `--check`/`--init`/`--completions` are present in both flag sets; both tag probes are byte-identical; the fish `-n` guard contains `not __fish_prev_arg_in --search` and no `__fish_seen_subcommand_from`; the zsh state spec is exactly `'*: :->args'` with a single `args)` arm; `fish -c 'source ...; complete -C ...'` parses without error.

---

## User Persona (if applicable)

**Target User**: A zsh or fish user who sources/installs the completion (`cp completions/_skilldozer ~/.zsh/completions/` / `cp completions/skilldozer.fish ~/.config/fish/completions/`, or `eval "$(skilldozer --completions)"`) and presses `<tab>` after `skilldozer`.

**Use Case**: `skilldozer <tab>` to pick a skill; `skilldozer --c<tab>` to pick `--check`/`--completions`; `skilldozer --init <tab>` to pick the store directory; `skilldozer writing/<tab>` to pick a nested skill.

**User Journey**: User loads the file → `skilldozer <tab>` shows their skills (not a command list) → they pick one or type `--` to see flags → flags are long-form only → `--init`/`--store` offer directories.

**Pain Points Addressed**: today a bare `<tab>` offers `check init completion` alongside skills (clutter + shadows skills named check/init/completions); today `skilldozer -<tab>` advertises 7 short aliases that clutter the menu (PRD §14.1 rule 3: long-form-only, mirroring `--help`).

---

## Why

- **Implements decision 20 (PRD §14.1 / §17 guardrail):** completions are skills-first (bare `<tab>` = skills) and long-form-only (flags = `--long`, never short aliases). The current zsh + fish files violate both (offer subcommands on bare `<tab>`; advertise `-v -h -p -l -a -f -s`).
- **Implements decision 19 (the `--flag` conversion):** `check`/`init`/`completion` are now `--check`/`--init`/`--completions` flags (P1.M1.T1, Complete). The §14.4 lockstep invariant requires each completion file's flag set to match `parseArgs()` — so both files must advertise the three new flags and STOP offering the bare subcommands.
- **Closes the three-file lockstep (§14.4):** bash is done (P1.M2.T1.S1). This task brings zsh + fish to the identical contract so all three emit the same flag set via `--completions`. §17's completion-lockstep guardrail is satisfied for the whole changeset.
- **Keeps the embed path honest:** `TestEmbeddedCompletionsMatchOnDisk` verifies `//go:embed` bytes == on-disk bytes for ALL THREE shells; rewriting the two on-disk files + rebuilding keeps them identical (§14.6 lockstep).

---

## What

A targeted rewrite of each file:

**ZSH** (`completions/_skilldozer`): (1) drop the 7 short entries from the `_arguments` flags array; (2) add `--check`/`--init[:directory:_files]`/`--completions` before the closing `)`; (3) collapse the two-state `_arguments` spec + case block into one `'*: :->args'` state + single `args)` arm; (4) keep the tag probe byte-identical; (5) update the LOCKSTEP comment.

**FISH** (`completions/skilldozer.fish`): (1) drop `-s X` tokens from the flag matrix (incl. `-s s` on `--search`); (2) add `--check`/`--init … -r`/`--completions`; (3) DELETE the bare-subcommand offer comment + 3 directives; (4) re-author ONLY the `-n` guard of the tag directive (`not __fish_prev_arg_in --search`), keeping the `-a '(...)'` probe byte-identical; (5) update the LOCKSTEP comment. No new routing directive for `--init` — fish handles dir completion via the `-r` on the flag definition (same as the existing `--store -r`).

### Success Criteria

- [ ] **ZSH** flags array has ZERO short entries (`grep -c "'-[vhplafs]\["` = 0); contains `--check`/`--init[:directory:_files]`/`--completions`; tag probe `tags=(${(f)"$(skilldozer --relative --all 2>/dev/null)"})` byte-identical.
- [ ] **ZSH** `_arguments` line is `_arguments -C "$flags[@]" '*: :->args' && return 0`; the case block has a single `args)` arm doing `compadd -- "$tags[@]"`; no `->first`/`->rest`/`check init completion` remains.
- [ ] **FISH** flag matrix has ZERO `-s X` tokens (`grep -c -- '-s [vhplafs]'` = 0); contains `-l check`, `-l init … -r`, `-l completions`.
- [ ] **FISH** no `__fish_is_first_arg' -a 'check'/'init'/'completion'` directives remain; the tag directive `-n` guard is `not __fish_prev_arg_in --search` (no `__fish_seen_subcommand_from`, no `--search -s`); the `-a '(skilldozer --relative --all 2>/dev/null)'` probe is byte-identical.
- [ ] Both LOCKSTEP comments now cite decisions 19/20 + the long-form-only rule (matching bash's wording, shell-appropriate pronoun).
- [ ] `go build` succeeds; `./skilldozer --completions --shell fish | grep -q 'complete -c skilldozer'` prints `COMPLETIONS_FISH_OK`; `go test ./...` green (incl. `TestEmbeddedCompletionsMatchOnDisk`).
- [ ] Only `completions/_skilldozer` and `completions/skilldozer.fish` are modified.

---

## All Needed Context

### Context Completeness Check

**Pass.** Every edit is pinned to an exact line in both target files (read in full with line numbers), with before/after strings transcribed from the change map §2/§3 and verified against the live files. The one genuinely non-obvious point — that a lone zsh `'*: :->args'` `_arguments` spec (no `1:` numbered form) routes the FIRST and all subsequent positionals to the `args` state — is **empirically confirmed by `man zshcompsys`** ("*:message:action … when neither of the first two forms was provided … Any number of arguments can be completed in this fashion"). The fish `-r` directory-routing idiom is confirmed by the existing, working `--store -r` directive. The §13 gate semantics (fish has one assertion; zsh none beyond embed-identity) and the `TestEmbeddedCompletionsMatchOnDisk` auto-satisfy-on-rebuild are confirmed. The `--shell` enum-routing omission is a deliberate, documented decision (consistent with the bash sibling). An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the verified facts (exact lines, the zsh *: gotcha, the fish -r precedent, the §13 gate semantics)
- file: plan/004_5851dcff4371/P1M2T1S2/research/verified_facts.md
  why: "§0 scope + sibling boundary (bash is DONE; do NOT touch it). §1 the §13 gate: fish has ONE assertion
        (grep 'complete -c skilldozer'), zsh none beyond go test embed-identity — but PRD §14.1/§17 require
        long-form-only + the three new flags for BOTH, so Level 4 spot-checks prove faithfulness. §2 --completions
        emits for all three shells today (verified). §3 TestEmbeddedCompletionsMatchOnDisk checks ALL THREE (auto-pass
        on rebuild). §4 THE zsh gotcha: lone '*: :->args' covers the FIRST positional (man-page quote). §5 zsh
        flag-spec idioms (:directory:_files, :query:, compadd --). §6 fish -r precedent + helper builtins exist +
        -n re-author. §7 --shell OUT OF SCOPE. §8 the 13-flag list + exact descriptions. §9 LOCKSTEP comment text.
        §10 the Level 4 spot-checks. §11 live smoke."
  critical: "§4 (zsh *: ->args) and §6 (fish -r + -n re-author) are the two things most likely to be mishandled."

# MUST READ — the authoritative change map (§2 = zsh, §3 = fish, with exact line numbers + before/after)
- file: plan/004_5851dcff4371/architecture/completions_change_map.md
  why: "§2 (completions/_skilldozer) pins every zsh edit: lines 22-27 short pairs, line 34 -s, new flags before
        line 39 ')', lines 43-58 state handling, line 19 tag probe (keep), lines 12-14 LOCKSTEP. §3
        (completions/skilldozer.fish) pins every fish edit: lines 17-22 -s tokens, line 32 -s s, new flags after
        line 24, lines 43-45 delete bare offers, lines 51-52 -n re-author. NOTE: its 'General principles' #6
        mentions --shell enum routing, but §2/§3 edits + §13 assertions all omit it — omit it (verified_facts §7)."

# MUST READ — both files under edit (read in full; zsh 61 lines, fish 52 lines)
- file: completions/_skilldozer
  why: "THE zsh edit target. tag probe @19 (KEEP BYTE-IDENTICAL). flags array @21-39 (drop -x @22-27 + -s @34;
        add --check/--init[:directory:_files]/--completions before @39 ')'). _arguments spec @41 (collapse to
        '*: :->args'). case block @43-58 (collapse to single args) arm). LOCKSTEP @12-14 (append 2 paragraphs)."
  pattern: "zsh completion = an autoload function ending with `_skilldozer \"$@\"`; `#compdef skilldozer` header;
            `_arguments -C \"$flags[@]\" ... && return 0` for flags (returns 0 → short-circuit) then a `case
            \"$state\"` for positionals; tag probe `tags=(${(f)\"$(skilldozer --relative --all 2>/dev/null)\"})`."

- file: completions/skilldozer.fish
  why: "THE fish edit target. global -f @14 (KEEP). flag matrix @17-24 (drop -s tokens @17-22 + -s s @32; add
        --check/--init -r/--completions after @24). --store -r @39 (KEEP — the -r dir-routing precedent). bare
        subcommand offers @41-45 (DELETE). tag directive @47-52 (re-author -n ONLY; keep -a probe byte-identical).
        LOCKSTEP @9-11 (append 2 paragraphs)."
  pattern: "fish completion = a series of `complete -c skilldozer ...` directives; `-f` global no-files; `-l long
            -d 'desc'` for flags; `-r` to make a flag's value complete to files; `-n 'guard'` condition; `-a 'cand'
            -d 'desc'` for candidates (dynamic via command substitution)."

# MUST READ — the bash sibling PRP (the CONTRACT/reference: bash is already implemented exactly as specced)
- file: plan/004_5851dcff4371/P1M2T1S1/PRP.md
  why: "Defines the exact contract this task must match in lockstep: the 13-long-form flag list, the --init dir
        routing shape, the LOCKSTEP comment text (decisions 19/20), the --shell-omission decision (its GOTCHA #4 /
        DESIGN DECISION #3/#4), the rebuild-before-gate rule, and TestEmbeddedCompletionsMatchOnDisk auto-pass.
        Its STATUS note confirms bash is implemented and disjoint from this task."

# MUST READ — the embed-identity test (checks ALL THREE shells; auto-satisfies; do NOT touch the embed directives)
- file: main_test.go
  why: "TestEmbeddedCompletionsMatchOnDisk @2995 table = {bash,zsh,fish} × on-disk path; compares
        completionScript(shell) (the //go:embed var) to os.ReadFile of the on-disk file. After this task edits the
        two files + `go build` re-embeds, embedded == on-disk → PASS automatically for zsh+fish. Do NOT edit the
        test or main.go's //go:embed (lines 54/57/60)."
  gotcha: "Run `go build` (or `go test`, which builds) AFTER editing the files so the embed vars hold the new
           bytes — otherwise the test compares stale embedded bytes to the new files and FAILS."

# MUST READ — the §13 delta acceptance assertions (the operative gate)
- file: plan/004_5851dcff4371/delta_prd.md
  why: "§13 (lines 45-65): the ONLY fish assertion is line 48 `grep -q 'complete -c skilldozer'` (COMPLETIONS_FISH_OK);
        zsh has no §13 grep (covered by go test). The bash-only EMBED_HAS_*/LONG_FORM_ONLY_ checks (lines 62-65) use
        --shell bash and do NOT exercise fish/zsh flag content — hence the Level 4 spot-checks in this PRP."
  section: "§13 acceptance (the `--completions --shell ...` block)."

# READ-ONLY — PRD (the authority for the skills-first / long-form-only contract)
- file: PRD.md
  why: "READ-ONLY. §14.1 (h3.14) the behavior matrix + rules 1-6 (skills-first; long-form-only; --init/--store route
        to dirs; --search routes to nothing; positionals are ALWAYS skills). §14.2 (h3.15) value-taking flags.
        §14.4 (h3.17) the lockstep invariant (all three files frozen to parseArgs). §14.6 (h3.19) the 13-flag -<tab>
        table (NO --shell). §17 (h2.16) the two completion guardrails."
  section: "h3.14 (§14.1), h3.15 (§14.2), h3.17 (§14.4), h3.19 (§14.6 table), h2.16 (§17 guardrails)."

# READ-ONLY — the reference file (already rewritten by the sibling; match its shape/wording)
- file: completions/skilldozer.bash
  why: "The bash file is DONE. Read it to copy the LOCKSTEP comment text verbatim (lines 11-18), the --init dir-routing
        shape (--store|--init → dirs), and the 13-long-form flag list. Do NOT edit it."

# EXTERNAL (canonical docs — confirm the idioms; citable for the implementer)
- url: https://zsh.sourceforge.io/Doc/Release/Completion-System.html
  why: "_arguments spec forms: `*:message:action` covers all non-option arguments when no numbered `n:` form precedes
        it (the §4 gotcha). The `->state` action form + `-C` (sets $state). The `_files` action for path completion.
        `compadd` as the candidate primitive."
- url: https://fishshell.com/docs/current/completions.html
  why: "`complete -r`/`--require-parameter` routes a flag's value to file completion (the --store/--init precedent).
        `-n 'condition'` guard; `-a 'candidates'` (dynamic command substitution per keystroke). The shipped helper
        functions __fish_prev_arg_in / __fish_is_first_arg / __fish_seen_subcommand_from."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && wc -l completions/_skilldozer completions/skilldozer.fish completions/skilldozer.bash
  61 completions/_skilldozer        # THIS task (zsh)
  52 completions/skilldozer.fish    # THIS task (fish)
  69 completions/skilldozer.bash    # DONE (sibling P1.M2.T1.S1 — reference, do NOT edit)
# tooling present:
$ zsh --version   # zsh 5.9.2
$ fish --version  # fish, version 4.8.0
$ go version      # (whatever the repo pins; go build works — verified)
# main.go parseArgs has all 13 long flags (lockstep-confirmed); --completions emits for bash/zsh/fish (verified).
```

### Desired Codebase tree with files to be changed

```bash
completions/_skilldozer        # REWRITE — long-form flags array; *: :->args single state; tags offered on every
                               #            positional; 3 new flags; tag probe byte-identical; LOCKSTEP cites 19/20.
completions/skilldozer.fish    # REWRITE — long-form-only flag matrix; 3 new flags (--init -r); bare subcommand offers
                               #            DELETED; tag directive -n re-authored to not __fish_prev_arg_in --search;
                               #            probe byte-identical; LOCKSTEP cites 19/20.
# completions/skilldozer.bash — UNCHANGED (P1.M2.T1.S1, done)
# main.go / main_test.go / internal/* / go.mod / go.sum — UNCHANGED
```

| File | Responsibility |
|---|---|
| `completions/_skilldozer` | Skills-first + long-form-only zsh completion; offers the 13 long flags on `-<tab>`; offers skills on every positional `<tab>`; routes `--init`/`--store` to dirs via `:directory:_files`; routes `--search` to nothing. |
| `completions/skilldozer.fish` | Skills-first + long-form-only fish completion; `-l long` flags only; `--init`/`--store` get `-r` (dir completion); `--search` offers nothing; bare `<tab>` offers skills via the dynamic probe; no bare subcommand offers. |

### Known Gotchas of our codebase & Library Quirks

```zsh
# GOTCHA #1 (CRITICAL, zsh) — A LONE '*: :->args' spec covers the FIRST positional too.
# The current file splits positionals into '1: :->first' + '*: :->rest' (line 41). The contract collapses these to a
# single '*: :->args'. CONFIRMED CORRECT by `man zshcompsys`: "*:message:action describes how arguments ... are to be
# completed when neither of the first two forms [n: numbered, or *:optspec:] was provided. Any number of arguments
# can be completed in this fashion." With NO numbered form, the * spec handles the FIRST and every subsequent
# positional. compadd -- "$tags[@]" in the args) arm then offers skills on bare <tab> AND on later positionals.
# (verified_facts §4.) DO NOT keep a separate '1: :->first' spec — that reintroduces the first/rest distinction
# the contract removes (subcommands are gone; all positionals are skills).

# GOTCHA #2 (zsh) — _arguments -C ... && return 0 semantics. When completing a flag token (cur starts with -),
# _arguments resolves it from $flags and returns 0 → the && return 0 short-circuits (no positional case). When
# completing a positional, _arguments sets $state=args and returns NON-zero → execution falls through to the
# case "$state". KEEP -C and && return 0. (verified_facts §4.)

# GOTCHA #3 (zsh) -- :directory:_files routes the VALUE slot to dir completion. The new '--init[...]:directory:_files'
# entry mirrors the existing '--store[...]:directory:_files' (line 38) EXACTLY — the :directory:_files suffix is what
# routes --init's value to file/path completion. Do NOT add a separate positional/state handler for --init's value;
# the _arguments flag spec handles it. (verified_facts §5.)

# GOTCHA #4 (fish, CRITICAL) — -r on the flag DEFINITION is what routes --init's value to dirs; NO separate directive.
# The existing `complete -c skilldozer -l store ... -r` (line 39) ALREADY routes --store's value to file/dir completion
# via -r (fish 4.x "require-parameter" mode). Add --init with the SAME -r. Do NOT add a second `complete ... -l init -r`
# routing directive after --store — the contract §4c says "Fish handles this automatically via -r on the --init flag
# definition." One definition with -r is enough. (verified_facts §6.)

# GOTCHA #5 (fish, CRITICAL) — re-author ONLY the -n guard; keep the -a probe BYTE-IDENTICAL.
# OLD: -n 'not __fish_seen_subcommand_from check init completion; and not __fish_prev_arg_in --search -s'
# NEW: -n 'not __fish_prev_arg_in --search'
# Drop the __fish_seen_subcommand_from clause (no subcommands remain to guard against) AND drop the -s alias from
# '--search -s' (long-form-only). The -a '(skilldozer --relative --all 2>/dev/null)' probe and -d 'skill tag' MUST be
# byte-identical (PRD §14.3: the 2>/dev/null is load-bearing robustness). (verified_facts §6.)

# GOTCHA #6 (both) — REBUILD before the gate. //go:embed reads both files at BUILD time (main.go:57 _skilldozer,
# main.go:60 skilldozer.fish). After editing, `go build` (or `go test`, which builds) re-embeds the new bytes →
# TestEmbeddedCompletionsMatchOnDisk (embedded == on-disk, all three shells) passes. If you edit and run the §13 gate
# against a STALE binary, the embedded bytes are old → the gate may still pass (it only checks 'complete -c skilldozer'
# for fish) BUT TestEmbeddedCompletionsMatchOnDisk FAILS (embed != on-disk). Always `go build` after the edit.
# (verified_facts §3.)

# GOTCHA #7 (both) — --shell is NOT in the flag-offer list and is NOT routed to the enum. PRD §14.6's `skilldozer -<tab>`
# table lists exactly 13 flags (no --shell); --shell is a --completions modifier. The contract §3/§4, the change map
# §2/§3 edits, and the §13 assertions all omit it; the bash sibling omits it too (its GOTCHA #4 / DESIGN DECISION #3/#4).
# Do NOT add --shell to either flag set and do NOT add --shell value-routing. (verified_facts §7.)

# GOTCHA #8 (zsh) — KEEP the tag probe BYTE-IDENTICAL: `tags=(${(f)"$(skilldozer --relative --all 2>/dev/null)"})`
# (line 19). The ${(f)...} splits on newlines; the 2>/dev/null swallows errors (PRD §14.3). Do not "improve" it.

# GOTCHA #8a (fish) — KEEP the tag probe BYTE-IDENTICAL: the -a argument `'(skilldozer --relative --all 2>/dev/null)'`
# (the dynamic command substitution). Only the -n guard changes; -a and -d 'skill tag' are untouched.

# GOTCHA #9 (fish) — KEEP `complete -c skilldozer -f` (line 14, the global no-files rule) and the --store -r directive
# (line 39) and the --search no-r rationale comment block (lines 25-31). These are correct and load-bearing (PRD §14.1
# rule 6 + §14.2). --init -r intentionally bypasses -f (dir completion wanted); --search deliberately lacks -r.

# GOTCHA #10 (both) — No conflict with the sibling P1.M2.T1.S1 (bash, DONE) or P1.M1.T3.S2 (test-only, disjoint from
# completions/*). The shared touchpoint TestEmbeddedCompletionsMatchOnDisk is left green by all three. Land in any order.

# GOTCHA #11 (both) — Only the two completion files change. Do NOT edit completions/skilldozer.bash (done), main.go,
# main_test.go, internal/*, go.mod, go.sum. (Scope discipline.)
```

---

## Implementation Blueprint

### Data models and structure

**None.** This is a shell-script rewrite of two files. No Go types, no structs, no signatures.

### Implementation Tasks (ordered by dependencies)

> Both files are independent of each other — edit them in either order. The tasks within each file are ordered
> top-to-bottom for clarity. After BOTH files are edited, run the shared validation (Task 7).
>
> **READ FIRST — idempotency note:** the FIND/`oldText` strings below transcribe the **original** pre-rewrite file content captured at PRP-write time. As of PRP-write time the files were ALREADY observed in the target (REPLACE) state (see the ⚠️ note in the STATUS block above). Run the **Validation Loop Level 4** spot-checks first; if green, the edits are already applied and this section is a no-op/spec — skip to Task 7.

```yaml
# ════════════════════════ ZSH: completions/_skilldozer ════════════════════════

Task 1: EDIT completions/_skilldozer — the LOCKSTEP comment (lines 12-14)
  - FILE: completions/_skilldozer
  - FIND:
        # LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
        # task adds/renames a flag there, update this list — and the bash/fish files —
        # identically.
  - REPLACE with (append the same 2 paragraphs the bash file added — match bash's wording; keep the
    zsh file's existing "this list — and the bash/fish files —" pronoun):
        # LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
        # task adds/renames a flag there, update this list — and the bash/fish files —
        # identically. There is no shared source of truth the shells can import.
        # Flags are long-form-only (decision 20): short aliases stay valid at runtime
        # but are not advertised. Updated for --check/--init/--completions (decision 19):
        # these were promoted from bare subcommands so the bare positional namespace
        # belongs to skill tags — a bare <tab> shows skills, never commands.

Task 2: EDIT completions/_skilldozer — the flags array (lines 21-39): drop short forms, add 3 flags
  - (2a) Drop the 7 short-form entries (the right-hand '-x[...]' on lines 22-27). FIND the whole array open + the
         6 short pairs + the long-only entries + the --search comment/--search/-s lines + the --store comment/--store:
            local -a flags=(
                '--version[Print the skilldozer version]'   '-v[Print the skilldozer version]'
                '--help[Show this help message]'      '-h[Show this help message]'
                '--path[Print the resolved skills directory]' '-p[Print the resolved skills directory]'
                '--list[Human-readable catalog (TAG, NAME, DESCRIPTION)]' '-l[Human-readable catalog]'
                '--all[Print every skill directory path, sorted by tag]' '-a[Print every skill dir path]'
                '--file[Print the SKILL.md path instead of the directory]' '-f[Print the SKILL.md path]'
                '--relative[Print paths relative to the skills directory]'
                '--no-color[Disable ANSI color]'
                # `:query:` marks --search/-s as value-taking, so zsh routes the value
                # slot away from $state (no tag completion after them). NO completion is
                # offered for the search value (free-text query).
                '--search[Substring search over tag/name/description/keywords]:query:'
                '-s[Substring search over tag/name/description/keywords]:query:'
                # `:directory:_files` routes --store's value slot to file/path completion
                # (directories). No short form. (The OPPOSITE of --search: --store WANTS
                # path completion; --search offers nothing.)
                '--store[Non-interactive store path for init]:directory:_files'
            )
     REPLACE with (long-form only — delete every '-x[...]'; delete the standalone -s line; update the --search comment
     to drop '-s'; add --check/--init[:directory:_files]/--completions before the ')'):
            local -a flags=(
                '--version[Print the skilldozer version]'
                '--help[Show this help message]'
                '--path[Print the resolved skills directory]'
                '--list[Human-readable catalog (TAG, NAME, DESCRIPTION)]'
                '--all[Print every skill directory path, sorted by tag]'
                '--file[Print the SKILL.md path instead of the directory]'
                '--relative[Print paths relative to the skills directory]'
                '--no-color[Disable ANSI color]'
                # `:query:` marks --search as value-taking, so zsh routes the value
                # slot away from $state (no tag completion after it). NO completion is
                # offered for the search value (free-text query). No short alias is
                # advertised (decision 20: long-form-only).
                '--search[Substring search over tag/name/description/keywords]:query:'
                # `:directory:_files` routes --store's and --init's value slots to
                # file/path completion (directories). (The OPPOSITE of --search:
                # --store/--init WANT path completion; --search offers nothing.)
                '--store[Non-interactive store path for init]:directory:_files'
                # Decision 19: check/init/completions promoted from bare subcommands to
                # flags; decision 20: no short aliases advertised.
                '--check[Validate every skill on disk]'
                '--init[First-run setup: pick/create the skills store]:directory:_files'
                '--completions[Emit the shell completion script for eval]'
            )
  - This single edit drops all 7 short entries, deletes the -s line, and adds the 3 promoted flags (GOTCHA #1/#3).

Task 3: EDIT completions/_skilldozer — the _arguments spec + case block (lines 41-58): collapse to one state
  - (3a) The _arguments line. FIND:
            _arguments -C "$flags[@]" '1: :->first' '*: :->rest' && return 0
     REPLACE with (GOTCHA #1 — lone *: covers the first positional too):
            _arguments -C "$flags[@]" '*: :->args' && return 0
  - (3b) The case block. FIND:
            case "$state" in
                first)
                    # check AND init are offered ONLY as the first positional token.
                    compadd -- "$tags[@]" check init completion
                    ;;
                rest)
                    # `check` AND `init` are exclusive (PRD §6.3: either+tags → exit 2).
                    # Once one is seen, suppress tags so completion never invites a
                    # guaranteed error.
                    if (( ${words[(I)check]} || ${words[(I)init]} || ${words[(I)completion]} )); then
                        _message 'subcommand takes no tag arguments'
                    else
                        compadd -- "$tags[@]"
                    fi
                    ;;
            esac
     REPLACE with (single args) arm; positionals are ALWAYS skills; no exclusivity guard):
            case "$state" in
                args)
                    # Positionals are ALWAYS skills (decision 19: no bare subcommands),
                    # and skills are never mutually exclusive — offer them on every
                    # positional <tab>, first or later.
                    compadd -- "$tags[@]"
                    ;;
            esac
  - The 'first'/'rest' split, the `compadd ... check init completion` offer, and the `${words[(I)check]...}` exclusivity
    guard are ALL gone. The tag probe at line 19 is untouched (it feeds $tags, used by the new arm).

# ════════════════════════ FISH: completions/skilldozer.fish ════════════════════════

Task 4: EDIT completions/skilldozer.fish — the LOCKSTEP comment (lines 9-11)
  - FIND:
        # LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
        # task adds/renames a flag there, update this file — and the bash/zsh files —
        # identically.
  - REPLACE with (append the same 2 paragraphs as bash/zsh; keep the fish file's "this file — and the bash/zsh files —"
    pronoun):
        # LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
        # task adds/renames a flag there, update this file — and the bash/zsh files —
        # identically. There is no shared source of truth the shells can import.
        # Flags are long-form-only (decision 20): short aliases stay valid at runtime
        # but are not advertised. Updated for --check/--init/--completions (decision 19):
        # these were promoted from bare subcommands so the bare positional namespace
        # belongs to skill tags — a bare <tab> shows skills, never commands.

Task 5: EDIT completions/skilldozer.fish — the flag matrix (lines 17-32): drop -s tokens, add 3 flags
  - (5a) Drop the 6 `-s X` short-option tokens (lines 17-22). FIND:
            complete -c skilldozer -s v -l version  -d 'Print the skilldozer version'
            complete -c skilldozer -s h -l help     -d 'Show this help message'
            complete -c skilldozer -s p -l path     -d 'Print the resolved skills directory'
            complete -c skilldozer -s l -l list     -d 'Human-readable catalog (TAG, NAME, DESCRIPTION)'
            complete -c skilldozer -s a -l all      -d 'Print every skill directory path, sorted by tag'
            complete -c skilldozer -s f -l file     -d 'Print the SKILL.md path instead of the directory'
            complete -c skilldozer       -l relative -d 'Print paths relative to the skills directory'
            complete -c skilldozer       -l no-color -d 'Disable ANSI color'
     REPLACE with (drop every `-s X` token; keep the `-l long -d 'desc'`):
            complete -c skilldozer -l version  -d 'Print the skilldozer version'
            complete -c skilldozer -l help     -d 'Show this help message'
            complete -c skilldozer -l path     -d 'Print the resolved skills directory'
            complete -c skilldozer -l list     -d 'Human-readable catalog (TAG, NAME, DESCRIPTION)'
            complete -c skilldozer -l all      -d 'Print every skill directory path, sorted by tag'
            complete -c skilldozer -l file     -d 'Print the SKILL.md path instead of the directory'
            complete -c skilldozer -l relative -d 'Print paths relative to the skills directory'
            complete -c skilldozer -l no-color -d 'Disable ANSI color'
            # Decision 19: check/init/completions promoted from bare subcommands to flags.
            # Decision 20: long-form-only — no short aliases are advertised.
            complete -c skilldozer -l check       -d 'Validate every skill on disk'
            # --init <dir> (§8.2): like --store, its value is a directory; `-r` routes
            # the value slot to file/dir completion (the inverse of --search's nothing).
            complete -c skilldozer -l init        -d 'First-run setup: pick/create the skills store' -r
            complete -c skilldozer -l completions -d 'Emit the shell completion script for eval'
  - (5b) Drop the `-s s` from the --search line (line 32) + update its comment (lines 25-31). FIND:
            # --search/-s take a free-text query, so NO completion is offered after them.
            # We deliberately do NOT pass -r here: in fish 4.x `-r` switches into
            # "complete the option's value" mode, which BYPASSES the global `-f` above and
            # offers file names for the query. Without -r, --search/-s are treated as plain
            # flags, so after `--search ` the global `-f` (no-files) applies and nothing is
            # offered -- exactly the PRD §6.1 free-text-query behavior. (fish's -r is only a
            # completion hint; skilldozer itself enforces that --search needs a value, exit 1.)
            complete -c skilldozer -s s -l search -d 'Substring search over tag/name/description/keywords'
     REPLACE with (drop '-s s'; drop '/-s' from the comment):
            # --search takes a free-text query, so NO completion is offered after it.
            # We deliberately do NOT pass -r here: in fish 4.x `-r` switches into
            # "complete the option's value" mode, which BYPASSES the global `-f` above and
            # offers file names for the query. Without -r, --search is treated as a plain
            # flag, so after `--search ` the global `-f` (no-files) applies and nothing is
            # offered -- exactly the PRD §6.1 free-text-query behavior. (fish's -r is only a
            # completion hint; skilldozer itself enforces that --search needs a value, exit 1.)
            complete -c skilldozer -l search -d 'Substring search over tag/name/description/keywords'
  - NOTE: Task 5a inserts the 3 new flags right after --no-color (contract §4b "after line 24"); --init gets -r there
    (GOTCHA #4 — the -r on the definition is what routes the value; no separate directive). The --store -r directive
    (line 39) is UNCHANGED.

Task 6: EDIT completions/skilldozer.fish — DELETE bare subcommand offers + re-author the tag directive guard
  - (6a) DELETE the bare subcommand offers (lines 41-45, incl. the comment). FIND:
            # `check` AND `init` are EXCLUSIVE subcommands (PRD §6.3). Offer them only as
            # the first arg.
            complete -c skilldozer -n '__fish_is_first_arg' -a 'check' -d 'Validate every skill on disk'
            complete -c skilldozer -n '__fish_is_first_arg' -a 'init' -d 'First-run setup: pick/create the skills store and write the config'
            complete -c skilldozer -n '__fish_is_first_arg' -a 'completion' -d 'Emit the shell completion script for eval'
            # Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
     REPLACE with (delete the comment + 3 directives; keep the blank line + the start of the dynamic-tags comment):
            # Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
     (i.e. the 5 lines above — comment + 3 completes — are removed entirely; the following "# Dynamic tags:"
     comment remains and becomes the lead-in to the re-authored directive.)
  - (6b) Re-author the tag directive's -n guard + its comment (lines 47-52). FIND:
            # Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
            # tag — the store is manifest-free and changes as skills are added). Suppressed
            # once `check` OR `init` is seen (exclusive subcommand, PRD §6.3) AND when the
            # previous arg is --search/-s (free-text query — no tag completion there either).
            complete -c skilldozer -n 'not __fish_seen_subcommand_from check init completion; and not __fish_prev_arg_in --search -s' \
                -a '(skilldozer --relative --all 2>/dev/null)' -d 'skill tag'
     REPLACE with (GOTCHA #5 — re-author ONLY -n; drop the subcommand clause + the -s; keep -a byte-identical):
            # Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
            # tag — the store is manifest-free and changes as skills are added). Suppressed
            # only when the previous arg is --search (free-text query — no tag completion
            # there). No subcommand guard: positionals are ALWAYS skills (decision 19).
            complete -c skilldozer -n 'not __fish_prev_arg_in --search' \
                -a '(skilldozer --relative --all 2>/dev/null)' -d 'skill tag'
  - The `-a '(skilldozer --relative --all 2>/dev/null)'` probe and `-d 'skill tag'` are byte-identical (GOTCHA #8a).
    Only the comment + the -n condition changed.

# ════════════════════════ SHARED VALIDATION ════════════════════════

Task 7: VERIFY (syntax + §13 gate + embed identity + content spot-checks + smoke)
  - (zsh has no standalone syntax checker short of zsh -n; run it on the file)
  - zsh -n completions/_skilldozer                       # exit 0 (GOTCHA: zsh -n checks parse)
  - fish -n completions/skilldozer.fish                  # exit 0 (fish has a -n parse check)
  - go build -o skilldozer .                             # rebuild so //go:embed picks up new bytes (GOTCHA #6)
  - §13 fish gate:
        ./skilldozer --completions --shell fish 2>/dev/null | grep -q 'complete -c skilldozer' && echo COMPLETIONS_FISH_OK
  - go test ./...                                        # incl. TestEmbeddedCompletionsMatchOnDisk (all three shells)
  - content spot-checks (see Validation Loop Level 4 — long-form-only, 3 new flags, probes byte-identical, guards)
  - fish smoke:
        fish -c 'source completions/skilldozer.fish; complete -C "skilldozer "' 2>&1 | head   # parses + offers
  - See the Validation Loop for the full command set.
```

### Implementation Patterns & Key Details

```zsh
# The rewritten ZSH file, in full (the shape after Tasks 1-3):

#compdef skilldozer
# Zsh completion for skilldozer (autoload function).
#
# Install (one of):
#   cp completions/_skilldozer ~/.zsh/completions/_skilldozer
#   cp completions/_skilldozer /usr/local/share/zsh/site-functions/_skilldozer
# then ensure `autoload -U compinit && compinit` in your .zshrc.
#
# Tags are derived DYNAMICALLY from disk by calling `skilldozer --relative --all`
# (skilldozer is manifest-free, PRD §2.1: there is no sidecar catalog to read).
#
# LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this list — and the bash/fish files —
# identically. There is no shared source of truth the shells can import.
# Flags are long-form-only (decision 20): short aliases stay valid at runtime
# but are not advertised. Updated for --check/--init/--completions (decision 19):
# these were promoted from bare subcommands so the bare positional namespace
# belongs to skill tags — a bare <tab> shows skills, never commands.
_skilldozer() {
    local -a tags
    # Canonical relTags, one per line. ${(f)...} splits on newlines. Errors
    # swallowed: a missing/broken skilldozer yields an empty list, not a stderr dump.
    tags=(${(f)"$(skilldozer --relative --all 2>/dev/null)"})

    local -a flags=(
        '--version[Print the skilldozer version]'
        '--help[Show this help message]'
        '--path[Print the resolved skills directory]'
        '--list[Human-readable catalog (TAG, NAME, DESCRIPTION)]'
        '--all[Print every skill directory path, sorted by tag]'
        '--file[Print the SKILL.md path instead of the directory]'
        '--relative[Print paths relative to the skills directory]'
        '--no-color[Disable ANSI color]'
        # `:query:` marks --search as value-taking, so zsh routes the value
        # slot away from $state (no tag completion after it). NO completion is
        # offered for the search value (free-text query). No short alias is
        # advertised (decision 20: long-form-only).
        '--search[Substring search over tag/name/description/keywords]:query:'
        # `:directory:_files` routes --store's and --init's value slots to
        # file/path completion (directories). (The OPPOSITE of --search:
        # --store/--init WANT path completion; --search offers nothing.)
        '--store[Non-interactive store path for init]:directory:_files'
        # Decision 19: check/init/completions promoted from bare subcommands to
        # flags; decision 20: no short aliases advertised.
        '--check[Validate every skill on disk]'
        '--init[First-run setup: pick/create the skills store]:directory:_files'
        '--completions[Emit the shell completion script for eval]'
    )

    _arguments -C "$flags[@]" '*: :->args' && return 0

    case "$state" in
        args)
            # Positionals are ALWAYS skills (decision 19: no bare subcommands),
            # and skills are never mutually exclusive — offer them on every
            # positional <tab>, first or later.
            compadd -- "$tags[@]"
            ;;
    esac
}

_skilldozer "$@"
```

```fish
# The rewritten FISH file, in full (the shape after Tasks 4-6):

# Fish completion for skilldozer.
#
# Install:
#   cp completions/skilldozer.fish ~/.config/fish/completions/skilldozer.fish
#
# Tags are derived DYNAMICALLY from disk by calling `skilldozer --relative --all`
# (skilldozer is manifest-free, PRD §2.1: there is no sidecar catalog to read).
#
# LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this file — and the bash/zsh files —
# identically. There is no shared source of truth the shells can import.
# Flags are long-form-only (decision 20): short aliases stay valid at runtime
# but are not advertised. Updated for --check/--init/--completions (decision 19):
# these were promoted from bare subcommands so the bare positional namespace
# belongs to skill tags — a bare <tab> shows skills, never commands.

# No file completion: skilldozer takes tags/flags, not paths.
complete -c skilldozer -f

# Flag matrix (§6.1/§6.2). --relative and --no-color have NO short forms.
complete -c skilldozer -l version  -d 'Print the skilldozer version'
complete -c skilldozer -l help     -d 'Show this help message'
complete -c skilldozer -l path     -d 'Print the resolved skills directory'
complete -c skilldozer -l list     -d 'Human-readable catalog (TAG, NAME, DESCRIPTION)'
complete -c skilldozer -l all      -d 'Print every skill directory path, sorted by tag'
complete -c skilldozer -l file     -d 'Print the SKILL.md path instead of the directory'
complete -c skilldozer -l relative -d 'Print paths relative to the skills directory'
complete -c skilldozer -l no-color -d 'Disable ANSI color'
# Decision 19: check/init/completions promoted from bare subcommands to flags.
# Decision 20: long-form-only — no short aliases are advertised.
complete -c skilldozer -l check       -d 'Validate every skill on disk'
# --init <dir> (§8.2): like --store, its value is a directory; `-r` routes
# the value slot to file/dir completion (the inverse of --search's nothing).
complete -c skilldozer -l init        -d 'First-run setup: pick/create the skills store' -r
complete -c skilldozer -l completions -d 'Emit the shell completion script for eval'
# --search takes a free-text query, so NO completion is offered after it.
# We deliberately do NOT pass -r here: in fish 4.x `-r` switches into
# "complete the option's value" mode, which BYPASSES the global `-f` above and
# offers file names for the query. Without -r, --search is treated as a plain
# flag, so after `--search ` the global `-f` (no-files) applies and nothing is
# offered -- exactly the PRD §6.1 free-text-query behavior. (fish's -r is only a
# completion hint; skilldozer itself enforces that --search needs a value, exit 1.)
complete -c skilldozer -l search -d 'Substring search over tag/name/description/keywords'

# --store <dir> (PRD §8.2): Non-interactive store path for init. Unlike --search,
# --store's value is a DIRECTORY, so here we DO pass `-r`: in fish 4.x `-r`
# switches into "complete the option's value" mode, which BYPASSES the global
# `-f` above and offers file/dir paths for the value. This is the intentional
# INVERSE of --search's no-`-r` (free-text -> offer nothing). No short form.
complete -c skilldozer -l store -d 'Non-interactive store path for init' -r

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
# tag — the store is manifest-free and changes as skills are added). Suppressed
# only when the previous arg is --search (free-text query — no tag completion
# there). No subcommand guard: positionals are ALWAYS skills (decision 19).
complete -c skilldozer -n 'not __fish_prev_arg_in --search' \
    -a '(skilldozer --relative --all 2>/dev/null)' -d 'skill tag'
```

Notes easy to get wrong:
- **Collapse the zsh `_arguments` to ONE `*: :->args` state** (Task 3) — a lone `*:` covers the first positional too (GOTCHA #1, `man zshcompsys`). Don't keep a `1: :->first` spec; that reintroduces the dead first/rest split.
- **`--init` routing is via the flag spec/definition, not a separate handler.** zsh: the `:directory:_files` suffix on `'--init[...]'`. fish: the `-r` on the `--init` `complete` line. (GOTCHA #3/#4.)
- **Re-author ONLY the fish `-n` guard** (Task 6b) — the `-a '(skilldozer --relative --all 2>/dev/null)'` probe is byte-identical (GOTCHA #5/#8a).
- **Rebuild before the gate** so `//go:embed` re-reads both files (GOTCHA #6).
- **Do not add `--shell`** to either flag set or as a value-route (GOTCHA #7).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Collapse zsh `1: :->first` + `*: :->rest` → one `*: :->args`? → YES.** With subcommands gone, there is no first/rest distinction: every positional is a skill. A lone `*:` spec covers all positionals incl. the first (`man zshcompsys`, verified_facts §4). The contract §3c prescribes exactly this. Keeping two states would leave a dead `first)` arm + a meaningless `rest)` exclusivity guard.
2. **Where to place the 3 new fish flags? → after `--no-color` (contract §4b "after line 24").** Grouping `--check`/`--init`/`--completions` right after the existing simple-flag matrix honors the contract literally and keeps them adjacent. `--init`'s `-r` is on its own definition (GOTCHA #4); no separate `--init` routing directive is added after `--store`. (The change map §3 shows the new flags "after line 24" too.)
3. **Add `--shell` enum routing / to the flag list? → NO.** PRD §14.6's `skilldozer -<tab>` table lists exactly 13 flags (no `--shell`); `--shell` is a `--completions` modifier. The contract §3/§4, change map §2/§3 edits, §13 assertions, AND the bash sibling all omit it. Adding it is scope creep and breaks the three-way lockstep. (GOTCHA #7 / verified_facts §7.)
4. **Keep `compadd -- "$tags[@]"` vs. switch to `_describe`/`_tags`? → KEEP `compadd`.** The current file uses `compadd -- "$tags[@]"` (plain array offer); the rewrite preserves it. `_describe` would require a display/desc mapping the tags don't have (relTags are bare identifiers). Minimal change = lowest risk; the shell's native prefix-filtering does the rest (PRD §14.1 rule 2).
5. **Keep the zsh `_arguments -C ... && return 0` pattern? → YES.** It's the idiomatic short-circuit (flags resolved → return 0; positional → set `$state`, fall through to `case`). Don't "simplify" by removing `&& return 0` — that would offer tags even on `-<tab>`. (GOTCHA #2.)
6. **Update the `--search` comments to drop `/-s`? → YES.** Both files' `--search` comments reference `-s`; with `-s` no longer advertised, the comments are stale. Update them to drop `/-s` (and the zsh comment gains the "no short alias advertised" note). Cosmetic but keeps the developer-facing docs honest (contract §6 Mode A: comments ARE the docs).
7. **fish: merge `--init` into the `--store -r` line vs. separate definition? → SEPARATE definition with `-r`.** The contract §4b/§4c says add a `complete ... -l init ... -r` line (not merge into `--store`). A separate definition gives `--init` its own `-d` description and keeps `--store`'s comment block intact. (GOTCHA #4.)

### Integration Points

```yaml
EMBED (the load-bearing integration):
  - main.go's `//go:embed completions/_skilldozer` (line 57) and `//go:embed completions/skilldozer.fish` (line 60)
    read these files at BUILD time. Editing them + `go build` → the embedded vars hold the new bytes →
    `skilldozer --completions --shell zsh|fish` emits them → the §13 fish gate + the spot-checks run against the
    new content. TestEmbeddedCompletionsMatchOnDisk (embedded == on-disk, all three shells) auto-passes. (GOTCHA #6.)

LOCKSTEP (§14.4):
  - Both flag sets are frozen to main.go parseArgs() (all 13 long flags confirmed in usageText). The §17 guardrail
    (update all three files on a flag change) is honored: bash is done (P1.M2.T1.S1); this task does zsh+fish.

NO GO SOURCE / NO ROUTES / NO DATABASE / NO DEPS:
  - Two shell-script files only. go.mod/go.sum/main.go/main_test.go/internal/* unchanged.
```

---

## Validation Loop

### Level 1: Syntax (immediate, after each file's rewrite)

```bash
cd /home/dustin/projects/skilldozer

zsh  -n completions/_skilldozer      # MUST exit 0 (zsh parse check)
fish -n completions/skilldozer.fish  # MUST exit 0 (fish parse check)
# Expected: no output, exit 0 for both.
```

### Level 2: The §13 delta assertion + rebuild — REBUILD first

```bash
cd /home/dustin/projects/skilldozer

go build -o skilldozer .   # GOTCHA #6: re-embed both rewritten files

# The one §13 fish assertion:
./skilldozer --completions --shell fish 2>/dev/null | grep -q 'complete -c skilldozer' && echo COMPLETIONS_FISH_OK
# Expected: COMPLETIONS_FISH_OK printed.

# (Optional sanity: zsh emits and still starts with #compdef)
./skilldozer --completions --shell zsh 2>/dev/null | head -1   # Expected: #compdef skilldozer

rm -f skilldozer
```

### Level 3: Whole-module + embed identity (all three shells)

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # 0
go vet ./...  ; echo "vet exit $?"      # 0
go test ./...  ; echo "test exit $?"    # 0 — incl. TestEmbeddedCompletionsMatchOnDisk (embed==on-disk for bash+zsh+fish)

# Scope invariant: only the two files changed
git diff --name-only                    # Expected: completions/_skilldozer AND completions/skilldozer.fish (ONLY)
```

### Level 4: Content spot-checks + live smoke (lockstep faithfulness)

```bash
cd /home/dustin/projects/skilldozer

# ── ZSH ──
# long-form-only: no short-form _arguments entries remain
grep -c "'-[vhplafs]\[" completions/_skilldozer                # expect 0
# the three promoted flags present in the _arguments array
grep -q -- "'--check\["        completions/_skilldozer && echo ZSH_CHECK_OK
grep -q -- "'--init\["         completions/_skilldozer && echo ZSH_INIT_OK
grep -q -- "'--completions\["  completions/_skilldozer && echo ZSH_COMPLETIONS_OK
# tag probe byte-identical (PRD §14.3 robustness — the 2>/dev/null is load-bearing)
grep -q 'tags=(${(f)"$(skilldozer --relative --all 2>/dev/null)"})' completions/_skilldozer && echo ZSH_PROBE_OK
# no leftover subcommand / first-rest dead code
grep -c "check init completion\|->first\|->rest" completions/_skilldozer   # expect 0
# the lone *: :->args state spec present
grep -q "'\*: :->args'" completions/_skilldozer && echo ZSH_ARGS_STATE_OK

# ── FISH ──
# long-form-only: no `-s X` short-option tokens in flag defs
grep -c -- '-s [vhplafs]' completions/skilldozer.fish          # expect 0
# the three promoted flags present
grep -q -- '-l check'        completions/skilldozer.fish && echo FISH_CHECK_OK
grep -q -- '-l init .*-r'    completions/skilldozer.fish && echo FISH_INIT_OK
grep -q -- '-l completions'  completions/skilldozer.fish && echo FISH_COMPLETIONS_OK
# tag probe byte-identical (the -a dynamic command substitution)
grep -q -- "-a '(skilldozer --relative --all 2>/dev/null)'" completions/skilldozer.fish && echo FISH_PROBE_OK
# the re-authored -n guard (no subcommand clause, no -s alias)
grep -q "not __fish_prev_arg_in --search'" completions/skilldozer.fish && echo FISH_GUARD_OK
# no leftover bare subcommand offers
grep -c "__fish_is_first_arg' -a 'check'\|__fish_is_first_arg' -a 'init'\|__fish_is_first_arg' -a 'completion'" completions/skilldozer.fish  # expect 0

# ── LIVE SMOKE ──
# fish: parse-load + drive the completion engine headlessly (fish 4.x `complete -C`)
fish -c '
  function skilldozer; if test "$argv[1]$argv[2]" = "--relative--all"; echo writing/foo; echo example; end; end
  source completions/skilldozer.fish
  echo "bare <tab>:"; complete -C "skilldozer "
  echo "--c flags:" ; complete -C "skilldozer --c"
' 2>&1 | head
# Expected: "bare <tab>:" lists the stub skills (writing/foo, example) — NO check/init/completion.
#           "--c flags:" lists --check/--completions.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `zsh -n completions/_skilldozer` exit 0 AND `fish -n completions/skilldozer.fish` exit 0
- [ ] Level 2 PASS — `COMPLETIONS_FISH_OK` printed (after `go build`)
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (TestEmbeddedCompletionsMatchOnDisk green for all three shells); `git diff --name-only` = ONLY `completions/_skilldozer` + `completions/skilldozer.fish`
- [ ] Level 4 PASS — both files: zero short-form advertisements; `--check`/`--init`/`--completions` present; tag probes byte-identical; zsh has `'*: :->args'` single state + no `->first`/`->rest`/`check init completion`; fish has `not __fish_prev_arg_in --search` guard + no `__fish_is_first_arg -a '...'`; live smoke shows skills on bare `<tab>` + the `--c` flags on `--c<tab>`

### Feature Validation
- [ ] **ZSH** flags array = 13 long forms (zero short); `--check`/`--init[:directory:_files]`/`--completions` added; tag probe byte-identical; single `*: :->args` state offers skills on every positional
- [ ] **FISH** flag matrix = `-l long` only (zero `-s X`); `--check`/`--init -r`/`--completions` added; bare subcommand offers deleted; tag directive `-n` re-authored (probe byte-identical)
- [ ] Both LOCKSTEP comments cite decisions 19/20 + long-form-only rule (matching bash)
- [ ] A bare `skilldozer <tab>` yields skills only (zsh + fish); `skilldozer -<tab>` yields the 13 long flags only; `--init`/`--store <tab>` yields dirs; `--search <tab>` yields nothing

### Code Quality / Convention Validation
- [ ] No short forms advertised; `--shell` not added to either flag set or as a value-route (GOTCHA #7)
- [ ] Existing comment blocks preserved/updated where still accurate (--search rationale, --store -r, global -f)
- [ ] No dead code (zsh `first)`/`rest)`/exclusivity guard; fish `__fish_is_first_arg` subcommand offers all gone)
- [ ] No Go/deps changes; go.mod/go.sum/main.go/main_test.go/internal/* unchanged

### Scope Discipline
- [ ] Did NOT touch `completions/skilldozer.bash` (P1.M2.T1.S1, done)
- [ ] Did NOT touch `main.go`, `main_test.go`, `internal/*`, `go.mod`, `go.sum` (unchanged)
- [ ] Did NOT add `--shell` to either flag list or as a value-route (GOTCHA #7)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't keep a separate zsh `1: :->first` state.** A lone `*: :->args` covers the first positional too (`man zshcompsys`). Keeping two states leaves a dead `first)` arm + a meaningless exclusivity guard. (GOTCHA #1.)
- ❌ **Don't add a separate `--init` routing directive in fish.** The `-r` on the `--init` `complete` definition is what routes its value to dirs — same as the existing `--store -r`. (GOTCHA #4.)
- ❌ **Don't touch the `-a '(skilldozer --relative --all 2>/dev/null)'` fish probe.** Re-author ONLY the `-n` guard; the probe is byte-identical (PRD §14.3 robustness). (GOTCHA #5/#8a.)
- ❌ **Don't run the §13 gate / `go test` against a stale binary.** `go build` first — `//go:embed` re-reads both files at build time, and `TestEmbeddedCompletionsMatchOnDisk` compares embedded vs on-disk for all three shells. (GOTCHA #6.)
- ❌ **Don't add `--shell`** to either flag list or as a value-route. It's a `--completions` modifier, absent from the PRD §14.6 `-<tab>` table, and omitted by the contract + change map + §13 + the bash sibling. (GOTCHA #7.)
- ❌ **Don't drop the zsh `_arguments -C ... && return 0` short-circuit.** Without it, tags would be offered on `-<tab>` (the flag arm wouldn't short-circuit). (GOTCHA #2.)
- ❌ **Don't edit `completions/skilldozer.bash`, main.go, or any test.** This task is `_skilldozer` + `skilldozer.fish` ONLY. (Scope discipline.)
- ❌ **Don't add deps or Go changes.** Two shell-script files; `go.mod`/`go.sum` byte-for-byte identical.

---

## Confidence Score

**9/10** — Two-file rewrite with every edit pinned to an exact line + before/after (both files read in full with line numbers), the change map §2/§3 and the §13 acceptance gate both transcribed, and the single genuinely non-obvious point (zsh lone `*: :->args` covers the first positional) **empirically confirmed via `man zshcompsys`**. The fish `-r` directory-routing idiom is proven by the existing, working `--store -r` directive; all fish helper functions are confirmed to ship with the installed fish 4.8.0; the §13 fish gate is confirmed to pass today and after the rewrite. The `--shell` omission is a deliberate, documented decision (contract + change map + §13 + §14.6 table + bash sibling all omit it). The bash sibling is already implemented, giving a verified reference for the LOCKSTEP comment text and the `--init` routing shape. The 1-point reservation is for the zsh live-smoke step (zsh completion can't be driven fully headlessly; the authoritative proofs are the man-page confirmation + the content spot-checks, and the `zsh -n` parse check — which is strong but not a behavioral test).
