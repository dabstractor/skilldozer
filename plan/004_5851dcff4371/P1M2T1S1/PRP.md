# PRP — P1.M2.T1.S1: Rewrite `completions/skilldozer.bash` (skills-first, long-form-only)

> **Subtask:** The bash half of P1.M2.T1 (rewrite the three completion files for decision 19/20). Converts `completions/skilldozer.bash` from the session-003 model (offered `check`/`init`/`completion` as exclusive first-positional subcommands + advertised both `--long` and `-short` flag forms) to the session-004 model (PRD §14.1): **a bare `<tab>` shows skills only** (no command list), and **`skilldozer -<tab>` shows long-form flags only** (including the three promoted `--check`/`--init`/`--completions`).
>
> **Scope:** ONE existing file only — `completions/skilldozer.bash` (69 lines). No `.go` source change (the `--flag` contract is already Complete per P1.M1.T1). No new files. Zero Go/deps changes — `//go:embed` picks up the new bytes on rebuild. The zsh/fish files are P1.M2.T1.S2; `main_test.go`/`internal/*` are P1.M1.T3; README is P1.M2.T2.S1.
>
> **STATUS (verified at PRP-write time):** read `completions/skilldozer.bash` (full, 69 lines), `completions_change_map.md` §1 (full), `delta_prd.md` §13 assertions (lines 47/62-65), and `main_test.go::TestEmbeddedCompletionsMatchOnDisk` (full). The parallel sibling P1.M1.T3.S2 is test-only and explicitly does NOT touch `completions/*` — no collision. `bash -n` passes on the current file (baseline); main.go's parseArgs has all 13 long flags (lockstep-confirmed).

---

## Goal

**Feature Goal**: Rewrite `completions/skilldozer.bash` so it implements PRD §14.1's skills-first / long-form-only contract: (a) the flag-offer list is the 13 long forms only (drop all 7 short aliases; add `--check`/`--init`/`--completions`); (b) `--init <dir>` routes to directory completion like `--store`; (c) `--search` value-routing drops the `-s` alias; (d) the entire subcommand walk-guard / `have_pos` machinery is deleted (subcommands no longer exist; positionals are always skills, and skills are never mutually exclusive); (e) the tag probe is byte-identical. A bare `skilldozer <tab>` then yields skills only; `skilldozer -<tab>` yields the 13 long flags only.

**Deliverable**: A rewritten `completions/skilldozer.bash` (one file) passing `bash -n`, the §13 delta assertions (the four `--completions --shell bash` greps), and `go test ./...` (incl. `TestEmbeddedCompletionsMatchOnDisk`, which auto-satisfies because `//go:embed` reads the rewritten file at build time).

**Success Definition**: `bash -n completions/skilldozer.bash` exits 0; `go build` succeeds (re-embeds); the four §13 greps all print their `*_OK` token; `go test ./...` is green; `grep -c -- '-v\b\|-h\b\|-p\b\|-l\b\|-a\b\|-f\b\|-s\b'` on the file returns 0 short-form advertisements; the tag probe line `tags=$(skilldozer --relative --all 2>/dev/null)` is byte-identical; the function name `_skilldozer_completion` is preserved.

---

## User Persona (if applicable)

**Target User**: A bash user who sources/installs the completion (`source completions/skilldozer.bash`) and presses `<tab>` after `skilldozer`.

**Use Case**: `skilldozer <tab>` to pick a skill; `skilldozer --c<tab>` to pick `--check`/`--completions`; `skilldozer --init <tab>` to pick the store directory.

**User Journey**: User sources the file → `skilldozer <tab>` shows their skills (not a command list) → they pick one or type `--` to see flags → flags are long-form only.

**Pain Points Addressed**: today a bare `<tab>` offers `check init completion` alongside skills (clutter + shadows skills named check/init); today `skilldozer -<tab>` offers 7 short aliases that clutter the menu (PRD §14.1 rule 3: long-form-only, mirroring `--help`).

---

## Why

- **Implements decision 20 (PRD §14.1 / §17 guardrail):** completions are skills-first (bare `<tab>` = skills) and long-form-only (flags = `--long`, never short aliases). The current bash file violates both (offers subcommands on bare `<tab>`; advertises `-v -h -p -l -a -f -s`).
- **Implements decision 19 (the `--flag` conversion):** `check`/`init`/`completion` are now `--check`/`--init`/`--completions` flags (P1.M1.T1, Complete). The §14.4 lockstep invariant requires the bash flag set to match `parseArgs()` — so the file must advertise the three new flags and STOP offering the bare subcommands.
- **Satisfies the §13 delta acceptance gate** (delta_prd.md:47,62-65): the four `--completions --shell bash` greps (`_skilldozer_completion` present; `--completions` present; `--check` present; no `--version -v` adjacency) all depend on this rewrite.
- **Keeps the embed path honest:** `TestEmbeddedCompletionsMatchOnDisk` verifies `//go:embed` bytes == on-disk bytes; rewriting the on-disk file + rebuilding keeps them identical (§14.6 lockstep).

---

## What

A targeted rewrite of `completions/skilldozer.bash`: (1) replace the flag list with the 13 long forms; (2) update the value-routing case (`--search)` + `--store|--init)`); (3) delete the entire walk-guard loop + `have_pos` + `cands` machinery (now dead code); (4) keep the tag probe byte-identical; (5) update the LOCKSTEP comment + the value-routing comment. The function name and the `complete -F` registration are unchanged.

### Success Criteria

- [ ] Flag list (line 41) is exactly `"--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions"` (13 long forms; zero short aliases).
- [ ] Value-routing case (line 34) is `--search) return 0 ;;` and `--store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;`.
- [ ] The walk-guard loop + `local i have_pos=0` + the `cands` variable are DELETED; the function offers tags directly via `COMPREPLY=($(compgen -W "$tags" -- "$cur"))`.
- [ ] The tag probe line `tags=$(skilldozer --relative --all 2>/dev/null)` is byte-identical.
- [ ] The function name `_skilldozer_completion` and the `complete -F _skilldozer_completion skilldozer` line are preserved.
- [ ] The LOCKSTEP comment cites decisions 19/20 and the long-form-only rule.
- [ ] `bash -n completions/skilldozer.bash` exits 0; `go build` succeeds; the four §13 greps all pass; `go test ./...` green.
- [ ] Only `completions/skilldozer.bash` is modified.

---

## All Needed Context

### Context Completeness Check

**Pass.** Every edit is pinned to an exact line in the 69-line file (read in full), with before/after strings transcribed from the change map §1 and verified against the live file. The one non-obvious point — that the walk loop becomes ENTIRELY dead code (not just the `check`/`init`/`completion` line) and must be deleted wholesale — is traced against all 9 PRD §14.1 behavior rows in `research/verified_facts.md` §3. The §13 gate semantics (the `--version[ ]+-v` regex) and the `TestEmbeddedCompletionsMatchOnDisk` auto-satisfy-on-rebuild are confirmed. The `--shell` enum-routing omission is a deliberate, documented decision (§5). An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the verified facts (exact lines, the dead-loop gotcha, the §13 gate semantics)
- file: plan/004_5851dcff4371/P1M2T1S1/research/verified_facts.md
  why: "§1 the exact line anchors. §2 the flag list before/after + the §13 LONG_FORM_ONLY regex semantics
        (the new list must have ZERO short tokens). §3 THE gotcha: the walk-guard loop (lines 46-55) is
        ENTIRELY dead code after removing subcommands — delete it wholesale (not just the check/init/
        completion line); traces all 9 §14.1 behavior rows on the simplified function. §4 the value-routing
        before/after. §5 --shell enum routing is OUT OF SCOPE (do NOT add it). §6 TestEmbeddedCompletions
        MatchOnDisk auto-satisfies on rebuild. §7 the exact §13 greps. §8 disjoint from sibling P1.M1.T3.S2."
  critical: "§2 (the §13 regex) and §3 (delete the whole loop, not one line) are the two things most likely
             to be mishandled."

# MUST READ — the authoritative change map (§1 = THIS task, with exact line numbers + before/after)
- file: plan/004_5851dcff4371/architecture/completions_change_map.md
  why: "§1 (`completions/skilldozer.bash`) pins every edit: line 41 flag list, line 34 value-routing, lines
        46-55 walk-guard, line 63 subcommand offers, line 61 tag probe (keep), lines 11-13 LOCKSTEP comment.
        The 'Post-change verification' section is the §13 gate. NOTE: its 'General principles' #6 mentions
        --shell enum routing, but §1's bash edits do NOT list it and no §13 assertion tests it — omit it
        (verified_facts §5)."

# MUST READ — the file under edit (read in full; 69 lines)
- file: completions/skilldozer.bash
  why: "THE edit target. _init_completion setup (keep). Value-routing case @34. Flag block @41 (compgen -W).
        Walk-guard loop @46-55 (DELETE). Tag probe @61 (KEEP BYTE-IDENTICAL). cands/have_pos @62-63 (DELETE).
        `complete -F _skilldozer_completion skilldozer` @69 (KEEP — §13 greps '_skilldozer_completion')."
  pattern: "bash completion = a function registered via `complete -F <fn> <cmd>`; _init_completion for cur/prev/
            words/cword with a manual fallback; value-routing via `case \"$prev\"`; flag offer via
            `compgen -W \"...\"`; tag probe via `$(skilldozer --relative --all 2>/dev/null)`."

# MUST READ — the §13 delta acceptance assertions this task must satisfy
- file: plan/004_5851dcff4371/delta_prd.md
  why: "Lines 47 + 62-65 are the operative gate: `grep -q '_skilldozer_completion'` (function name preserved),
        `grep -q '\\-\\-completions'` + `grep -q '\\-\\-check'` (the two new flags present in the offer),
        `! grep -Eq '\\-\\-version[ ]+-v'` (no short-form adjacency = long-form-only). All four depend solely
        on the flag-list line + the function name."
  section: "§13 acceptance (the `--completions --shell bash` block, lines 47/62-65)."

# MUST READ — the embed-identity test (auto-satisfies; do NOT touch it or the embed declaration)
- file: main_test.go
  why: "TestEmbeddedCompletionsMatchOnDisk @2995 compares completionScript('bash') (the //go:embed var) to
        os.ReadFile('completions/skilldozer.bash'). After this task edits the on-disk file + `go build`
        re-embeds, embedded == on-disk → PASS automatically. Do NOT edit the test or main.go's //go:embed."
  gotcha: "Run `go build` (or `go test`, which builds) AFTER editing the file so the embed var holds the new
           bytes — otherwise the test compares stale embedded bytes to the new file and FAILS."

# READ-ONLY — the parallel sibling PRP (boundary: test-only, does NOT touch completions/*)
- file: plan/004_5851dcff4371/P1M1T3S2/PRP.md
  why: "Confirms P1.M1.T3.S2 edits main_test.go + one line in internal/skillsdir/skillsdir_test.go (flips
        run-level/completion/help-text/unconfigured ASSERTIONS to the --flag contract). Its scope-discipline
        explicitly excludes completions/*. Disjoint from this task; land in either order. It LEAVES
        TestEmbeddedCompletionsMatchOnDisk green (this task preserves it via §6)."

# READ-ONLY — PRD (the authority for the skills-first / long-form-only contract)
- file: PRD.md
  why: "READ-ONLY. §14.1 (h3.14) the behavior matrix + rules 1-6 (skills-first; long-form-only; --init/--store
        route to dirs; --search routes to nothing). §14.2 (h3.15) value-taking flag handling. §14.4 (h3.17)
        the lockstep invariant. §14.6 (h3.19) the 13-flag -<tab> table (NO --shell). §17 (h2.16) the two
        completion guardrails. delta_prd.md decisions 19/20."
  section: "h3.14 (§14.1), h3.15 (§14.2), h3.17 (§14.4), h3.19 (§14.6 table), h2.16 (§17 guardrails)."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/004_5851dcff4371/tasks.json
  why: "P1.M2.T1.S1's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. This PRP transcribes it;
        tasks.json wins on any conflict — EXCEPT the --shell-routing omission (verified_facts §5: the contract
        a-f list + change map §1 + §13 assertions all omit it, so omitting is contract-faithful, not a
        divergence)."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls completions/skilldozer.bash
completions/skilldozer.bash   # 69 lines; the ONLY file this task edits
$ wc -l completions/skilldozer.bash   # 69
$ bash -n completions/skilldozer.bash && echo "baseline bash -n OK"
# main.go parseArgs has all 13 long flags (lockstep-confirmed); the --flag contract is Complete (P1.M1.T1).
```

### Desired Codebase tree with files to be changed

```bash
completions/skilldozer.bash   # REWRITE — long-form flag list; --init joins --store dir routing; walk-loop deleted;
                              #            tag probe byte-identical; LOCKSTEP comment cites decisions 19/20
# completions/_skilldozer, completions/skilldozer.fish — UNCHANGED (P1.M2.T1.S2)
# main.go / main_test.go / internal/* / go.mod / go.sum — UNCHANGED
```

| File | Responsibility |
|---|---|
| `completions/skilldozer.bash` | Skills-first + long-form-only bash completion; routes `--init`/`--store` to dirs; offers the 13 long flags on `-<tab>`; offers skills on bare `<tab>`. |

### Known Gotchas of our codebase & Library Quirks

```bash
# GOTCHA #1 (CRITICAL) — DELETE THE WHOLE WALK-GUARD LOOP (lines 46-55), not just the check/init/completion line.
# After decision 19 removed the bare subcommands, the loop's TWO purposes are both dead:
#   (a) the `[[ check/init/completion ]] && return 0` subcommand suppression — those tokens are now --flags /
#       ordinary tags (tags are never mutually exclusive: PRD §6.1 accepts multiple tags);
#   (b) the `have_pos=1` tracking — it gated ONLY line 63's `cands="$cands check init completion"`, which is
#       being deleted. Leaving a stripped loop with unused `have_pos`/`local i` is dead code + shellcheck noise.
# The change map confirms: "always offer tags as positionals". Delete `local i have_pos=0` + the entire
# `for ((i=1; i<cword; i++)); do … done`. Inline `cands` → `COMPREPLY=($(compgen -W "$tags" -- "$cur"))`.
# (verified_facts §3 traces all 9 §14.1 rows on the simplified function.)

# GOTCHA #2 — The §13 LONG_FORM_ONLY gate is a REGEX, not a substring. delta_prd.md:65 is
# `! grep -Eq '\-\-version[ ]+-v'`. The OLD list has `--version -v` adjacency → matches → `!` FAILS. The NEW
# list must have ZERO short tokens anywhere (not just no `--version -v` adjacency). The 13-long-form list has
# none → `!` SUCCEEDS → LONG_FORM_ONLY_BASH_OK. Do not leave ANY `-v`/`-h`/`-p`/`-l`/`-a`/`-f`/`-s` in the compgen -W.
# (verified_facts §2.)

# GOTCHA #3 — REBUILD before the gate. //go:embed reads completions/skilldozer.bash at BUILD time. After editing
# the file, `go build` (or `go test`, which builds) re-embeds the new bytes → TestEmbeddedCompletionsMatchOnDisk
# (embedded == on-disk) passes. If you edit the file and run the §13 greps against a STALE binary, the embedded
# bytes are old → the greps may still pass (they only check flag substrings) but TestEmbeddedCompletionsMatchOnDisk
# FAILS (embed != on-disk). Always `go build` after the edit. (verified_facts §6.)

# GOTCHA #4 — --shell is NOT in the flag-offer list and is NOT routed to the enum. PRD §14.6's `skilldozer -<tab>`
# table lists exactly the 13 flags (no --shell) — --shell is a --completions modifier, not a top-level menu flag.
# The contract a-f list, change map §1, and §13 assertions all omit --shell routing. Do NOT add `--shell` to the
# compgen -W list and do NOT add a `--shell)` value-route case. (verified_facts §5.)

# GOTCHA #5 — KEEP the function name `_skilldozer_completion` and the `complete -F _skilldozer_completion
# skilldozer` registration line. The §13 gate greps `_skilldozer_completion` (delta_prd.md:47) and the embed
# identity test depends on the whole file. Renaming the function breaks the registration + the grep.

# GOTCHA #6 — KEEP the tag probe BYTE-IDENTICAL: `tags=$(skilldozer --relative --all 2>/dev/null)`. PRD §14.3
# (robustness): the `2>/dev/null` is load-bearing (a broken binary degrades to empty tags, not a stderr dump).
# Do not "improve" it (no `--path`, no caching, no error reporting).

# GOTCHA #6a — The SC2207 comment (`# SC2207 (mapfile preferred) is acceptable here: tags and flags never
# contain spaces, so word-splitting is safe.`) stays accurate (tags are relTags, no spaces) — keep it above the
# final COMPREPLY line. The earlier `# SC2317` false-positive comment in the _init_completion fallback also stays.

# GOTCHA #7 — bash -n is the syntax gate (no Go compile for a .bash file). Run it after the rewrite. The current
# file passes bash -n (baseline); the rewrite must too. Common break: unbalanced parens in the deleted loop, or
# a stray `local i`/`have_pos` reference left after deleting the loop.

# GOTCHA #8 — No conflict with the parallel sibling P1.M1.T3.S2 (test-only; its PRP's scope-discipline excludes
# completions/*). The one shared touchpoint, TestEmbeddedCompletionsMatchOnDisk, is LEFT green by T3.S2 and
# PRESERVED green by this task (§6). Land in either order.

# GOTCHA #9 — Only `completions/skilldozer.bash` changes. The zsh/fish files are P1.M2.T1.S2 (a separate task
# with its own PRP) — do NOT edit them here, even though the change map describes all three. The §13 bash greps
# only exercise the bash file; the zsh/fish greps (`complete -c skilldozer` for fish) are S2's gate.
```

---

## Implementation Blueprint

### Data models and structure

**None.** This is a shell-script rewrite. No Go types, no structs, no signatures.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT completions/skilldozer.bash — the flag list (line 41)
  - FILE: completions/skilldozer.bash (the compgen -W inside the `if [[ "$cur" == -* ]]` block)
  - FIND (GOTCHA #2 — the §13 regex target):
        COMPREPLY=($(compgen -W \
            "--version -v --help -h --path -p --list -l --all -a --file -f --relative --no-color --search -s --store" \
            -- "$cur"))
  - REPLACE with (13 long forms; zero short aliases; add --check --init --completions):
        COMPREPLY=($(compgen -W \
            "--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions" \
            -- "$cur"))
  - This single edit satisfies §13 greps --completions, --check, and the LONG_FORM_ONLY regex.

Task 2: EDIT completions/skilldozer.bash — the value-routing case (line 34) + its comment (lines 31-33)
  - (2a) The case block. FIND:
        case "$prev" in
            --search|-s) return 0 ;;
            --store) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
        esac
    REPLACE with (drop -s; add --init to dir routing):
        case "$prev" in
            --search) return 0 ;;
            --store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
        esac
  - (2b) The comment above it (lines 31-33). FIND:
        # Value-taking flags: route the value slot away from tag completion.
        #   --search/-s  -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
        #   --store      -> directory value  -> complete DIRECTORIES via compgen -d.
        # (--store WANTS path completion, unlike --search's free-text -> nothing.)
    REPLACE with:
        # Value-taking flags: route the value slot away from tag completion.
        #   --search        -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
        #   --store/--init  -> directory value  -> complete DIRECTORIES via compgen -d.
        # (--store/--init WANT path completion, unlike --search's free-text -> nothing.)

Task 3: EDIT completions/skilldozer.bash — DELETE the walk-guard loop + have_pos + cands (lines 46-63)
  - FIND the entire block (GOTCHA #1 — delete the WHOLE loop, not one line):
        # Walk earlier words: `check` AND `init` are EXCLUSIVE subcommands (PRD §6.3 —
        # either +tags → exit 2), so once one appears, offer nothing further. Track
        # whether any non-flag positional was seen so they are only ever offered
        # as the FIRST positional token.
        local i have_pos=0
        for ((i=1; i<cword; i++)); do
            [[ "${words[i]}" == "check" || "${words[i]}" == "init" || "${words[i]}" == "completion" ]] && return 0
            [[ "${words[i]}" == -* ]] && continue
            have_pos=1
        done

        # Tags straight from the binary (canonical relTags, one per line). Errors
        # swallowed: a missing/broken skilldozer degrades to "no tags" instead of spewing
        # into the completion menu.
        local tags cands
        tags=$(skilldozer --relative --all 2>/dev/null)
        cands="$tags"
        (( have_pos == 0 )) && cands="$cands check init completion"
        # SC2207 (mapfile preferred) is acceptable here: tags and flags never
        # contain spaces, so word-splitting is safe.
        COMPREPLY=($(compgen -W "$cands" -- "$cur"))
  - REPLACE with (no walk loop; positionals are always skills; cands inlined to tags; tag probe + SC2207 comment
    preserved byte-identical — GOTCHA #6/#6a):
        # Tags straight from the binary (canonical relTags, one per line). Errors
        # swallowed: a missing/broken skilldozer degrades to "no tags" instead of spewing
        # into the completion menu. Positionals are ALWAYS skills (decision 19: no bare
        # subcommands), and skills are never mutually exclusive — offer them on every
        # positional <tab>, first or later.
        local tags
        tags=$(skilldozer --relative --all 2>/dev/null)
        # SC2207 (mapfile preferred) is acceptable here: tags and flags never
        # contain spaces, so word-splitting is safe.
        COMPREPLY=($(compgen -W "$tags" -- "$cur"))
  - GOTCHA #1: the `local i have_pos=0`, the entire `for` loop, the `cands` variable, and the `(( have_pos…))`
    line are ALL gone. The tag probe `tags=$(skilldozer --relative --all 2>/dev/null)` is byte-identical.

Task 4: EDIT completions/skilldozer.bash — the LOCKSTEP comment (lines 11-13)
  - FIND:
        # LOCKSTEP: the flag set below is frozen to `main.go parseArgs()`. If a future
        # task adds/renames a flag there, update this list — and the zsh/fish files —
        # identically. There is no shared source of truth the shells can import.
  - REPLACE with (cite decisions 19/20 + long-form-only rule + skills-first rationale):
        # LOCKSTEP: the flag set below is frozen to `main.go parseArgs()`. If a future
        # task adds/renames a flag there, update this list — and the zsh/fish files —
        # identically. There is no shared source of truth the shells can import.
        # Flags are long-form-only (decision 20): short aliases stay valid at runtime
        # but are not advertised. Updated for --check/--init/--completions (decision 19):
        # these were promoted from bare subcommands so the bare positional namespace
        # belongs to skill tags — a bare <tab> shows skills, never commands.

Task 5: VERIFY (syntax + §13 gate + embed identity + content spot-checks)
  - bash -n completions/skilldozer.bash                 # exit 0 (GOTCHA #7)
  - go build -o skilldozer .                            # rebuild so //go:embed picks up new bytes (GOTCHA #3)
  - §13 gate (all four must print their *_OK):
        ./skilldozer --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo COMPLETIONS_BASH_OK
        ./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-completions'        && echo EMBED_HAS_COMPLETIONS_FLAG_OK
        ./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-check'              && echo EMBED_HAS_CHECK_FLAG_OK
        ! ./skilldozer --completions --shell bash 2>/dev/null | grep -Eq '\-\-version[ ]+-v'   && echo LONG_FORM_ONLY_BASH_OK
  - go test ./...                                       # incl. TestEmbeddedCompletionsMatchOnDisk (auto-pass on rebuild)
  - content spot-checks:
        grep -c -E '(^| )-[vhplafs]( |")' completions/skilldozer.bash    # expect 0 (no short forms advertised)
        grep -q -- '--check --init --completions' completions/skilldozer.bash   # the 3 promoted flags, in order
        grep -q 'tags=$(skilldozer --relative --all 2>/dev/null)' completions/skilldozer.bash  # probe byte-identical
```

### Implementation Patterns & Key Details

```bash
# The rewritten function, in full (the shape after Tasks 1-4):

_skilldozer_completion() {
    local cur prev words cword
    _init_completion 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        cword=$COMP_CWORD
        words=("${COMP_WORDS[@]}")
        COMPREPLY=()
    }

    # Value-taking flags: route the value slot away from tag completion.
    #   --search        -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
    #   --store/--init  -> directory value  -> complete DIRECTORIES via compgen -d.
    # (--store/--init WANT path completion, unlike --search's free-text -> nothing.)
    case "$prev" in
        --search) return 0 ;;
        --store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
    esac

    # Flag completion when the current token starts with '-' (long-form only — decision 20).
    if [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W \
            "--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions" \
            -- "$cur"))
        return 0
    fi

    # Tags straight from the binary (canonical relTags, one per line). Errors
    # swallowed: a missing/broken skilldozer degrades to "no tags" instead of spewing
    # into the completion menu. Positionals are ALWAYS skills (decision 19: no bare
    # subcommands), and skills are never mutually exclusive — offer them on every
    # positional <tab>, first or later.
    local tags
    tags=$(skilldozer --relative --all 2>/dev/null)
    # SC2207 (mapfile preferred) is acceptable here: tags and flags never
    # contain spaces, so word-splitting is safe.
    COMPREPLY=($(compgen -W "$tags" -- "$cur"))
    return 0
}
complete -F _skilldozer_completion skilldozer
```

Notes easy to get wrong:
- **Delete the whole walk loop** (Task 3), not just the `check`/`init`/`completion` line — `have_pos`/`local i`/`cands` are all dead once the subcommand offer is gone (GOTCHA #1).
- **Zero short forms in the flag list** — the §13 regex `--version[ ]+-v` is satisfied by having NO `-X` tokens at all, not just no `--version -v` adjacency (GOTCHA #2).
- **Rebuild before the gate** so `//go:embed` re-reads the file (GOTCHA #3).
- **Do not add `--shell`** to the flag list or as a value-route (GOTCHA #4).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Delete the walk loop wholesale vs. strip one line.** Delete it all. Both its purposes (subcommand suppression + `have_pos` for the subcommand offer) are dead after decision 19. A stripped loop leaves unused `have_pos`/`local i` — dead code + shellcheck noise. The change map's "always offer tags as positionals" confirms. (GOTCHA #1.)
2. **Inline `cands` vs. keep the variable.** Inline (`COMPREPLY=($(compgen -W "$tags" …))`). `cands` was only ever `tags` possibly + subcommands; with subcommands gone it's always `tags`. The contract LOGIC (d) explicitly allows "the variable flows directly". Less indirection.
3. **Add `--shell` enum routing? → NO.** The contract a-f list, the change map §1 bash edits, the §13 assertions, and the PRD §14.6 `-<tab>` table all omit it. PRD §14.2 mentions it in prose but no behavior-contract row or test requires it for this changeset. Adding it is scope creep; omit it consistently with the zsh/fish tasks (which also omit it). (GOTCHA #4 / verified_facts §5.)
4. **`--shell` in the flag-offer list? → NO.** PRD §14.6's `skilldozer -<tab>` table lists exactly 13 flags (no `--shell`); `--shell` is a `--completions` modifier, not a top-level menu flag. The contract's 13-flag list matches the table exactly. (GOTCHA #4.)
5. **Touch the `_init_completion` fallback / SC2317 comment? → NO.** Unchanged; it's correct (the fallback runs when bash-completion is absent). Keep it and the SC2207 comment verbatim.

### Integration Points

```yaml
EMBED (the load-bearing integration):
  - main.go's `//go:embed completions/skilldozer.bash` (unchanged) reads this file at BUILD time.
    Editing the file + `go build` → the embedded var holds the new bytes → `skilldozer --completions
    --shell bash` emits them → the §13 greps run against the new content. TestEmbeddedCompletionsMatchOnDisk
    (embedded == on-disk) auto-passes. (GOTCHA #3.)

LOCKSTEP (§14.4):
  - The 13-flag list is frozen to main.go parseArgs() (all 13 confirmed present). The §17 guardrail
    (update all three files on a flag change) is honored: this task does bash; P1.M2.T1.S2 does zsh+fish.

NO GO SOURCE / NO ROUTES / NO DATABASE / NO DEPS:
  - One .bash file only. go.mod/go.sum/main.go/main_test.go/internal/* unchanged.
```

---

## Validation Loop

### Level 1: Syntax (immediate, after the rewrite)

```bash
cd /home/dustin/projects/skilldozer

bash -n completions/skilldozer.bash   # MUST exit 0 (the .bash syntax gate; GOTCHA #7)
# Expected: no output, exit 0.
```

### Level 2: The §13 delta assertions (the core gate) — REBUILD first

```bash
cd /home/dustin/projects/skilldozer

go build -o skilldozer .   # GOTCHA #3: re-embed the rewritten file

./skilldozer --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo COMPLETIONS_BASH_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-completions'        && echo EMBED_HAS_COMPLETIONS_FLAG_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-check'              && echo EMBED_HAS_CHECK_FLAG_OK
! ./skilldozer --completions --shell bash 2>/dev/null | grep -Eq '\-\-version[ ]+-v'   && echo LONG_FORM_ONLY_BASH_OK
# Expected: all four *_OK tokens printed.
rm -f skilldozer
```

### Level 3: Whole-module + embed identity

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # 0
go vet ./...  ; echo "vet exit $?"      # 0
go test ./...  ; echo "test exit $?"    # 0 — incl. TestEmbeddedCompletionsMatchOnDisk (embed==on-disk)

# Scope invariant: only the bash file changed
git diff --name-only                    # Expected: completions/skilldozer.bash (ONLY)
```

### Level 4: Content spot-checks + live tab-completion smoke (lockstep faithfulness)

```bash
cd /home/dustin/projects/skilldozer

# No short forms advertised anywhere in the flag offer:
grep -c -E '(^| )-[vhplafs]( |")' completions/skilldozer.bash   # expect 0
# The three promoted flags, in order, in the flag list:
grep -q -- '--check --init --completions' completions/skilldozer.bash && echo "promoted-flags-present OK"
# Tag probe byte-identical (PRD §14.3 robustness — the 2>/dev/null is load-bearing):
grep -q 'tags=$(skilldozer --relative --all 2>/dev/null)' completions/skilldozer.bash && echo "tag-probe-identical OK"
# No leftover subcommand-offer / have_pos dead code:
grep -c 'have_pos\|check init completion' completions/skilldozer.bash   # expect 0

# Live smoke (optional — proves the behavior matrix end-to-end). Source + drive via COMP_WORDS:
go build -o /tmp/sdz .
# enable completion in a subshell and probe the candidate computation:
bash -c '
  source completions/skilldozer.bash
  # bare <tab>: skills only (no check/init/completion, no -v/-h/...)
  COMP_WORDS=(skilldozer ""); COMP_CWORD=1; _skilldozer_completion; echo "bare offers: ${COMPREPLY[*]}"
  # -<tab>: 13 long flags only
  COMP_WORDS=(skilldozer -); COMP_CWORD=1; _skilldozer_completion; echo "flag offers: ${COMPREPLY[*]}"
'
# Expected: "bare offers:" = the skill tags (e.g. example + any others); NO check/init/completion/-x.
#           "flag offers:" = the 13 --long flags; NO -v/-h/-p/-l/-a/-f/-s.
rm -f /tmp/sdz
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `bash -n completions/skilldozer.bash` exit 0
- [ ] Level 2 PASS — all four §13 greps print their `*_OK` (after `go build`)
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (TestEmbeddedCompletionsMatchOnDisk green); `git diff --name-only` = ONLY `completions/skilldozer.bash`
- [ ] Level 4 PASS — zero short-form advertisements; `--check --init --completions` present in order; tag probe byte-identical; no `have_pos`/`check init completion` dead code; live smoke shows skills on bare `<tab>` + 13 long flags on `-<tab>`

### Feature Validation
- [ ] Flag list is the 13 long forms (zero short aliases); `--check`/`--init`/`--completions` added
- [ ] `--search)` routing (no `-s`); `--store|--init)` directory routing
- [ ] Walk-guard loop + `have_pos` + `cands` deleted; tags offered directly on every positional
- [ ] Tag probe `tags=$(skilldozer --relative --all 2>/dev/null)` byte-identical
- [ ] Function name `_skilldozer_completion` + `complete -F` registration preserved
- [ ] LOCKSTEP comment cites decisions 19/20 + long-form-only rule

### Code Quality / Convention Validation
- [ ] No short forms advertised; `--shell` not added (out of scope per contract + §14.6 table)
- [ ] SC2207/SC2317 explanatory comments preserved
- [ ] No dead code (`have_pos`/`local i`/`cands` all removed)
- [ ] No Go/deps changes; go.mod/go.sum/main.go/main_test.go/internal/* unchanged

### Scope Discipline
- [ ] Did NOT touch `completions/_skilldozer` or `completions/skilldozer.fish` (P1.M2.T1.S2)
- [ ] Did NOT touch `main.go`, `main_test.go`, `internal/*`, `go.mod`, `go.sum` (P1.M1 / unchanged)
- [ ] Did NOT add `--shell` to the flag list or as a value-route (GOTCHA #4)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't strip just the `check`/`init`/`completion` line from the walk loop.** The whole loop (plus `have_pos`, `local i`, `cands`) is dead after decision 19 — delete it all or leave dead code + shellcheck noise. (GOTCHA #1.)
- ❌ **Don't leave any short form in the flag list.** The §13 regex `--version[ ]+-v` is satisfied by ZERO short tokens, not just no `--version -v` pair. (GOTCHA #2.)
- ❌ **Don't run the §13 greps against a stale binary.** `go build` first — `//go:embed` re-reads the file at build time, and `TestEmbeddedCompletionsMatchOnDisk` compares embedded vs on-disk. (GOTCHA #3.)
- ❌ **Don't add `--shell`** to the flag list or as a value-route. It's a `--completions` modifier, absent from the PRD §14.6 `-<tab>` table, and omitted by the contract + change map + §13. (GOTCHA #4.)
- ❌ **Don't rename `_skilldozer_completion`** or drop the `complete -F` line. The §13 grep and the embed identity depend on them. (GOTCHA #5.)
- ❌ **Don't "improve" the tag probe.** `tags=$(skilldozer --relative --all 2>/dev/null)` is byte-identical by contract (LOGIC (e)); the `2>/dev/null` is load-bearing robustness (§14.3). (GOTCHA #6.)
- ❌ **Don't edit the zsh/fish files, main.go, or any test.** This task is `completions/skilldozer.bash` ONLY. (Scope discipline.)
- ❌ **Don't add deps or Go changes.** One `.bash` file; `go.mod`/`go.sum` byte-for-byte identical.

---

## Confidence Score

**9/10** — Single-file rewrite with every edit pinned to an exact line + before/after (read in full), the change map §1 and the §13 delta assertions both transcribed, and the one non-obvious point (delete the whole walk loop, not one line) traced against all 9 PRD §14.1 behavior rows. The §13 LONG_FORM_ONLY regex semantics and the `TestEmbeddedCompletionsMatchOnDisk` auto-satisfy-on-rebuild are confirmed. The `--shell` omission is a deliberate, documented decision (contract + change map + §13 + §14.6 table all omit it). The 1-point reservation is for the live-smoke step (Level 4), which depends on the bash-completion helper being loadable in the probe subshell — if `_init_completion` is absent, the manual fallback still sets `cur`/`prev` so the smoke works, but it's the one step that depends on the shell environment rather than the file content alone.
