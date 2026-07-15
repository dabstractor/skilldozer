# PRP — P1.M2.T2.S1: Update README.md — command refs, remove Reserved section, completions section

> **Subtask:** The Mode B (changeset-level) documentation sync for the `--flag` CLI contract (decisions 19/20). The codebase converted `check`/`init`/`completion` bare subcommands → `--check`/`--init`/`--completions` flags (P1.M1, Complete) and rewrote the three completion files skills-first + long-form-only (P1.M2.T1). README.md still documents the OLD bare-subcommand model. This subtask: (a) converts every bare-subcommand reference to its `--flag`; (b) DELETES the now-false "Reserved tag names" paragraph; (c) rewrites the Completions section for `--completions` + skills-first/long-form-only; (d) reorders sections to PRD §15. **Documentation-only — no `.go`/completion-file/test changes.** README.md is the sole file touched.
>
> **Scope:** ONE existing file — `README.md` (345 lines). No code, no tests, no completions, no go.mod. Depends on all implementing subtasks (M1 + M2.T1); runs last.
>
> **STATUS (verified at PRP-write time):** README.md read in full; every bare-subcommand reference grepped with exact line numbers (`skilldozer init` ×9, `skilldozer check` ×3, `skilldozer completion` ×3, plus 8 prose command-name refs). The binary's `ErrNotFound` (skillsdir.go:275) confirmed to already emit `run \`skilldozer --init\`` — so README's quoted message (L315) must match. The contract's suggested Reserved-paragraph replacement was checked against the grep gate and a **grep-safety conflict** was found and resolved (the literal `` `skilldozer check` `` would fail `! grep -Eq 'skilldozer (init|check|completion)\b'`; use the grep-safe phrasing). The sibling P1.M2.T1.S2 PRP was read — it touches only `completions/*`, never README (disjoint).

---

## Goal

**Feature Goal**: Make README.md accurately document the `--flag` CLI contract — zero stale bare-subcommand references, the "Reserved tag names" paragraph removed (it is now factually wrong: there are NO reserved names), the Completions section describing the skills-first + long-form-only behavior, and section ordering matching PRD §15.

**Deliverable**: Additive + subtractive edits to ONE file (`README.md`): ~12 command-reference conversions, 1 paragraph deletion + replacement, 1 completions-section rewrite (commands + new skills-first description), 1 section move (completions → position 8).

**Success Definition**: all 3 grep assertions pass (`grep -q 'skilldozer --completions'`; `! grep -q 'Reserved tag names'`; `! grep -Eq 'skilldozer (init|check|completion)\b'`); no `.go`/completion/test files changed; README section order matches PRD §15.

---

## User Persona (if applicable)

**Target User**: A reader of the README who installs skilldozer and follows its examples — every code block / prose command they copy must match the actual `--flag` binary (so `skilldozer --init`, not the now-invalid `skilldozer init`).

**Use Case**: `eval "$(skilldozer --completions)"` to load completions; `skilldozer --check` to validate; `skilldozer --init` to set up — all copied from the README.

**User Journey**: User reads README → Install → runs `skilldozer --init` → loads `eval "$(skilldozer --completions)"` → presses `<tab>` → sees their skills (per the README's skills-first description).

**Pain Points Addressed**: README examples that no longer work (bare subcommands are now tags/errors); a "Reserved tag names" paragraph that contradicts the new namespace-safety model; missing description of the headline `<tab>` = skills UX.

---

## Why

- **Accuracy.** The binary's CLI surface changed (decisions 19/20); the README is the canonical user doc and must match. Stale `skilldozer init` examples now either resolve as a *tag* named `init` or are misleading.
- **Closes the §15 outline gap.** PRD §15 is prescriptive ("Mirror the mcpeepants README's tone and structure"); the README's section order and the Completions description must match it.
- **Completes the changeset (Mode B).** This is the final documentation sweep that depends on all implementing subtasks; it is the last thing a reader sees.
- **No code risk.** Pure documentation — cannot break the build or tests; the only gate is grep + human reading.

---

## What

Targeted edits to README.md:

1. Convert ~12 `skilldozer init`/`check`/`completion` literals → `--init`/`--check`/`--completions` (Install, First run, Usage, Adding a skill, How finds store, Constraints).
2. Convert 8 prose command-name refs (`init`/`check`/`completion` used as command names) → their `--flag` forms (the `--store` paragraph, the completions load prose, the error contract).
3. DELETE the "Reserved tag names" paragraph (L239-247) → replace with a grep-safe one-line note.
4. Rewrite the Completions section (L96-118): `completion`→`--completions` commands + APPEND the skills-first/long-form-only description.
5. MOVE the `## Shell completions` section to position 8 (after "How finds store", before "Constraints") per PRD §15.

### Success Criteria

- [ ] Every `skilldozer init`/`check`/`completion` literal in README.md is now `--init`/`--check`/`--completions`.
- [ ] Every prose command-name ref (`init`/`check`/`completion` as commands) is flipped to the `--flag` form.
- [ ] The "Reserved tag names" paragraph is GONE; a grep-safe one-line note replaces it.
- [ ] The Completions section documents: `--completions` load commands (bash/zsh eval + fish source), the skills-first `<tab>` behavior, the 13 long-form flags on `-<tab>`, `--init`/`--store`→dirs + `--search`→nothing.
- [ ] README section order matches PRD §15 (completions is item 8, before Constraints).
- [ ] `grep -q 'skilldozer --completions' README.md` → exit 0.
- [ ] `! grep -q 'Reserved tag names' README.md` → exit 0 (i.e. no match).
- [ ] `! grep -Eq 'skilldozer (init|check|completion)\b' README.md` → exit 0 (no bare).
- [ ] Only `README.md` is modified; no `.go`/completion/test/go.mod changes.

---

## All Needed Context

### Context Completeness Check

**Pass.** Every edit is pinned to exact line numbers + before/after text transcribed from the live README (read in full). The 3 grep assertions are stated verbatim. The one non-obvious trap — that the contract's *suggested* Reserved-paragraph replacement containing `` `skilldozer check` `` would FAIL the contract's own grep — is identified and resolved with a grep-safe phrasing (`research/verified_facts.md` §3). The binary's current `ErrNotFound` message (`--init`, skillsdir.go:275) is confirmed so the quoted message (L315) matches the real output. The section-move boundaries are unambiguous `##` headers. The sibling boundary (completions files only, never README) is fixed. An implementer who has never seen this repo can do it in one pass by applying the 13 exact edits.

### Documentation & References

```yaml
# MUST READ — the verified facts (every line, the grep-safety fix, exact edits, the move)
- file: plan/004_5851dcff4371/P1M2T2S1/research/verified_facts.md
  why: "§2 the EXHAUSTIVE grep of every bare-subcommand reference (with line numbers) — this is
        the checklist; miss one and the grep gate fails. §3 THE grep-safety conflict: the
        contract's suggested one-liner may contain `skilldozer check` which fails
        `! grep -Eq 'skilldozer (init|check|completion)\b'`; use the grep-safe phrasing. §4 the
        13 exact before→after edits. §5 the §15 section move (completions 4→8, header-anchored).
        §6 the 3 grep assertions. §7 sibling boundary. §8 scope discipline (README only)."
  critical: "§2 (exhaustive line list) and §3 (grep-safe replacement) are the two things most
             likely to cause a one-pass stall (a missed reference fails the gate; a grep-unsafe
             replacement fails it too)."

# MUST READ — the authoritative change map (exact lines + old→new + the grep gate)
- file: plan/004_5851dcff4371/architecture/test_doc_change_map.md
  why: "§README (lines 108-150) is the authoritative change map: the ~16-line table (old→new),
        the Reserved-paragraph DELETE (L239-247), the Completions-section update, and the 3 grep
        assertions. NOTE: its suggested one-liner must be checked for grep-safety (verified_facts §3);
        its line numbers are the contract's and match the live README (verified)."
  section: "## README.md (345 lines) — whole section."

# MUST READ — the file under edit (read in full; the before-text of every edit is transcribed)
- file: README.md
  why: "THE edit target. The 13 edits' before-text is in the Implementation Tasks below, transcribed
        verbatim. Key regions: Install/First run (L19-92), Shell completions (L94-151), Usage
        (L152-211), Where skills live incl the Reserved para (L212-247), Adding a skill (L248-290),
        How finds store (L291-323), Constraints (L324-345)."

# MUST READ — PRD §15 (the README outline authority) + §14 (the completions behavior to describe)
- file: PRD.md
  why: "READ-ONLY. §15 (h2.14) the outline + the exact eval/source one-liners + the skills-first/
        long-form-only description to mirror. §14.1 (h3.14) the behavior matrix (bare <tab>=skills;
        -<tab>=13 long flags; --init/--store=dirs; --search=nothing). §14.6 (h3.19) the --completions
        flag + shell detection. §6.1/§6.3 (h3.1/h3.3) the no-bare-subcommands rule. §17 (h2.16) the
        guardrails. decisions 19 (subcommand→flag) + 20 (skills-first/long-form-only)."
  section: "h2.14 (§15), h3.14 (§14.1), h3.19 (§14.6), h3.1/h3.3 (§6.1/§6.3), h2.16 (§17), h2.18 (decisions 19/20)."

# READ-ONLY — the binary message (confirms the README's quoted "unconfigured" line must say --init)
- file: internal/skillsdir/skillsdir.go
  why: "Line 275: ErrNotFound = `skilldozer is not configured; run \`skilldozer --init\``. The README
        quotes this message verbatim (L315); it must say --init to match the live binary."

# READ-ONLY — the sibling PRP (boundary: completions files only, never README)
- file: plan/004_5851dcff4371/P1M2T1S2/PRP.md
  why: "Confirms P1.M2.T1.S2 rewrites completions/_skilldozer + completions/skilldozer.fish (skills-first
        + long-form-only + the 3 new flags). It does NOT touch README.md. The README's Completions
        DESCRIPTION mirrors the behavior those files implement (sourced from PRD §14, which both follow).
        Disjoint; land in either order."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && wc -l README.md && ls completions/
345 README.md
completions/: skilldozer.bash  _skilldozer  skilldozer.fish   # rewritten by P1.M2.T1 — DO NOT touch here
# bare-subcommand refs today (all must go):
$ grep -nE 'skilldozer (init|check|completion)\b' README.md | wc -l   # 15 hits across the file
```

### Desired Codebase tree with files to be changed

```bash
README.md   # EDIT: ~12 command refs → --flags; DELETE Reserved para (+ grep-safe note); rewrite
            #       Completions section (commands + skills-first desc); MOVE completions → §15 pos 8.
# ALL other files UNCHANGED (main.go, main_test.go, internal/*, completions/*, go.mod, go.sum)
```

| File | Change |
|---|---|
| `README.md` | Sync to the `--flag` CLI contract (decisions 19/20): convert refs, remove Reserved, rewrite completions, reorder to §15. |

### Known Gotchas of our codebase & Library Quirks

```markdown
<!-- GOTCHA #1 (CRITICAL — the one-pass-stall) — the grep gate is word-boundary-strict. -->
<!-- `! grep -Eq 'skilldozer (init|check|completion)\b' README.md` matches the literal "skilldozer <word>" -->
<!-- followed by a boundary. The NEW `--flag` forms are SAFE: "skilldozer --init" does NOT contain the -->
<!-- substring "skilldozer init" (the `-` breaks it). BUT a replacement that keeps `skilldozer check` -->
<!-- (e.g. in an illustrative "`skilldozer check` resolves the tag") WOULD FAIL the gate. The -->
<!-- Reserved-paragraph replacement MUST avoid the literals `skilldozer init/check/completion` entirely -->
<!-- (use "A skill named `check`" + the `--flag` forms). (research §3.) -->

<!-- GOTCHA #2 — convert the PROSE command-name refs too, not just the `skilldozer X` literals. -->
<!-- The grep only catches "skilldozer <word>", but lines like "`--store` implies init" (L80/84), -->
<!-- "On success, `init` prints" (L88), "the post-setup `check` report" (L90), "the `completion` -->
<!-- subcommand" (L100), "`init --store` with nothing after it" (L207) reference the command by bare -->
<!-- NAME. For accuracy they must also flip to `--init`/`--check`/`--completions`. (research §2.) -->

<!-- GOTCHA #3 — the quoted "unconfigured" message (L315) must match the LIVE binary. -->
<!-- internal/skillsdir/skillsdir.go:275 ErrNotFound already says `run \`skilldozer --init\``. -->
<!-- README L315 currently quotes `run \`skilldozer init\`` — flip to `--init` so the doc matches -->
<!-- what users actually see. (Do NOT change the binary; change the README to match it.) -->

<!-- GOTCHA #4 — do content edits BEFORE the section move (EDIT 13). The move shifts every -->
<!-- downstream line number, so any line-anchored edit done after the move would target the wrong -->
<!-- line. Anchor edits by TEXT (the before-strings are unique); do the move LAST, anchored by -->
<!-- `##` headers (not line numbers). (research §5.) -->

<!-- GOTCHA #5 — the move boundaries are `##` headers (unambiguous). Cut from `## Shell completions` -->
<!-- through the blank line immediately before `## Usage`; paste after the last line of -->
<!-- `## How \`skilldozer\` finds the store` (i.e. immediately before `## Constraints`). This makes -->
<!-- completions §15 item 8. Do NOT split the block or duplicate a header. -->

<!-- GOTCHA #6 — the 13-flag list in the Completions description must be EXACTLY these, alphabetical: -->
<!-- --all, --check, --completions, --file, --help, --init, --list, --no-color, --path, --relative, -->
<!-- --search, --store, --version (PRD §14.6 table; matches the rewritten completion files). -->
<!-- `--shell` is NOT in this list (it is a --completions modifier, not a top-level flag). -->

<!-- GOTCHA #7 — documentation-only. Do NOT edit main.go, main_test.go, internal/*, completions/*, -->
<!-- go.mod, go.sum, PRD.md, tasks.json, prd_snapshot.md, or .gitignore. README.md is the sole file. -->

<!-- GOTCHA #8 — no conflict with the sibling P1.M2.T1.S2 (completions files) or P1.M1.* (code/tests). -->
<!-- None of them touch README.md. Land in any order. -->
```

---

## Implementation Blueprint

### Data models and structure

**None.** This is a Markdown documentation edit. No types, no code, no signatures.

### Implementation Tasks (ordered by dependencies — content edits first, move last)

> Apply EDITS 1-12 (content) first — each is anchored by unique before-text. Then apply EDIT 13
> (the section move) last, anchored by `##` headers (the move shifts line numbers).

```yaml
EDIT 1 — L43 (go install prose): convert `skilldozer init`
  - FIND: `skilldozer init` (see First run, below) — it creates the store and writes the
  - REPLACE: `skilldozer --init` (see First run, below) — it creates the store and writes the

EDIT 2 — L63-66 (First run header + code): convert both
  - FIND:
        Whichever install path you used, run `skilldozer init` once:

        ```bash
        skilldozer init
        ```
  - REPLACE:
        Whichever install path you used, run `skilldozer --init` once:

        ```bash
        skilldozer --init
        ```

EDIT 3 — L76-77 (non-interactive forms code): convert both
  - FIND:
        skilldozer init /path/to/store      # positional
        skilldozer init --store /path/to/store
  - REPLACE:
        skilldozer --init /path/to/store      # positional
        skilldozer --init --store /path/to/store

EDIT 4 — L80-90 (the `--store` prose paragraph): flip `init`→`--init` (×4) + `check`→`--check`
  - FIND (the whole paragraph):
        `--store <dir>` implies `init`, so it works on its own as a first-class
        non-interactive form: `skilldozer --store /path/to/store` runs the full setup
        and writes the config. (Use one of the forms above in scripts when you want the
        intent to be self-evident; bare `--store` with an `init` token is the canonical
        shape.) Because `--store` implies init, it cannot be combined with tag
        arguments: `skilldozer --store /path mytag` exits 2 — it is an init, not a
        one-off store override for a single resolution.

        On success, `init` prints exactly the configured store path to stdout — one clean
        line, so `STORE="$(skilldozer init --store /path)"` works in scripts. The
        seeded/adopted status and the post-setup `check` report go to stderr. A leading
        `~` (or a bare `~`) in a typed answer or a `--store`/positional path expands to
        your home directory.
  - REPLACE:
        `--store <dir>` implies `--init`, so it works on its own as a first-class
        non-interactive form: `skilldozer --store /path/to/store` runs the full setup
        and writes the config. (Use one of the forms above in scripts when you want the
        intent to be self-evident; bare `--store` with an `--init` token is the canonical
        shape.) Because `--store` implies `--init`, it cannot be combined with tag
        arguments: `skilldozer --store /path mytag` exits 2 — it is `--init`, not a
        one-off store override for a single resolution.

        On success, `--init` prints exactly the configured store path to stdout — one clean
        line, so `STORE="$(skilldozer --init --store /path)"` works in scripts. The
        seeded/adopted status and the post-setup `--check` report go to stderr. A leading
        `~` (or a bare `~`) in a typed answer or a `--store`/positional path expands to
        your home directory.

EDIT 5 — L96-118 (Completions load block): convert `completion`→`--completions` + APPEND the
         skills-first/long-form-only description (GOTCHA #6 — the exact 13-flag list, alphabetical)
  - FIND (L96-118, the load instructions block):
        `skilldozer` ships dynamic completions for bash, zsh, and fish. Tag completion is
        not a static list: the shell calls `skilldozer --relative --all` at completion time,
        so it never goes stale as you add skills.

        The easiest way to load completions is the `completion` subcommand, which prints
        the script for your shell to eval. The binary embeds the completion scripts, so
        this works for `go install` users with no clone.

        **bash / zsh** — add to `~/.bashrc` or `~/.zshrc`:

        ```bash
        eval "$(skilldozer completion)"
        ```

        **fish** — add to `~/.config/fish/config.fish`:

        ```bash
        skilldozer completion --shell fish | source
        ```

        `--shell <bash|zsh|fish>` makes the eval deterministic; otherwise
        `skilldozer completion` auto-detects from `$SKILLDOZER_SHELL`, then `$SHELL`.
  - REPLACE (note: the first 3-line intro stays; the rest is updated + a new description appended):
        `skilldozer` ships dynamic completions for bash, zsh, and fish. Tag completion is
        not a static list: the shell calls `skilldozer --relative --all` at completion time,
        so it never goes stale as you add skills.

        The easiest way to load completions is the `--completions` flag, which prints
        the script for your shell to eval. The binary embeds the completion scripts, so
        this works for `go install` users with no clone.

        **bash / zsh** — add to `~/.bashrc` or `~/.zshrc`:

        ```bash
        eval "$(skilldozer --completions)"
        ```

        **fish** — add to `~/.config/fish/config.fish`:

        ```bash
        skilldozer --completions --shell fish | source
        ```

        `--shell <bash|zsh|fish>` makes the eval deterministic; otherwise
        `skilldozer --completions` auto-detects from `$SKILLDOZER_SHELL`, then `$SHELL`.

        Once loaded, completions are **skills-first and long-form-only**:

        - `skilldozer <tab>` lists your installed skill tags (the default, most-used
          action) — never the help text or a command list. The list is recomputed from
          `skilldozer --relative --all` on every keystroke, so a newly-dropped skill is
          completable immediately.
        - `skilldozer -<tab>` lists the **long-form flags only** — `--all`, `--check`,
          `--completions`, `--file`, `--help`, `--init`, `--list`, `--no-color`,
          `--path`, `--relative`, `--search`, `--store`, `--version` — narrowed by what
          you type after the dash. Short aliases (`-a`, `-l`, …) stay valid for typing
          but are deliberately not advertised.
        - `skilldozer --init <tab>` and `skilldozer --store <tab>` offer directories
          (the store to adopt); `skilldozer --search <tab>` offers nothing (free-text).

        This works because every action that is not a skill tag is a `--flag` —
        `--check`, `--init`, and `--completions` are flags, not bare subcommands — so the
        bare positional namespace belongs entirely to skill tags and a `<tab>` is
        unambiguous.

EDIT 6 — L183 (Usage code): convert `skilldozer check`
  - FIND (the line under "# Validate every skill on disk"):
        skilldozer check
  - REPLACE:
        skilldozer --check

EDIT 7 — L207-208 (error contract): convert `init --store`
  - FIND:
        the
        whole store). `--store` expects a value: `init --store` with nothing after
        it exits 2 rather than guessing a store.
  - REPLACE:
        the
        whole store). `--store` expects a value: `--init --store` with nothing after
        it exits 2 rather than guessing a store.

EDIT 8 — L239-247 (Reserved tag names paragraph): DELETE + grep-safe one-line note (GOTCHA #1)
  - FIND (the whole paragraph):
        **Reserved tag names.** `check` and `init` are subcommand names, so they never resolve as
        skill tags: `skilldozer check` runs validation and `skilldozer init` runs first-run setup.
        That is the standard CLI rule — a subcommand name takes precedence over a positional
        argument. A skill whose canonical tag collides (`skills/check/SKILL.md`, tag `check`) is
        still fully usable, just not via that one tag: it appears in `--list` and `--all`, and
        resolves by a nested path (`writing/check`), by its frontmatter `name`, or by a declared
        alias. To point `init` at a store directory literally named `check` or `init`, pass it with
        `--store` rather than as a positional argument.
  - REPLACE (GOTCHA #1 — NO `skilldozer init/check/completion` literals; grep-safe):
        There are **no reserved tag names**: bare words are always skill tags, and every
        action is a `--flag` (§6.1). A skill named `check`, `init`, or `completions`
        resolves normally by its tag — use `--check`, `--init`, or `--completions` to
        run the action.

EDIT 9 — L281 (Adding a skill code): convert `skilldozer check`
  - FIND (the line under "validate everything on disk:"):
        skilldozer check
  - REPLACE:
        skilldozer --check

EDIT 10 — L297 (How finds store prose): convert `skilldozer init`
  - FIND: 2. **Config file `store`** — the primary, set by `skilldozer init`. The config
  - REPLACE: 2. **Config file `store`** — the primary, set by `skilldozer --init`. The config

EDIT 11 — L315 (unconfigured message — match the live binary, GOTCHA #3): convert `init`→`--init`
  - FIND:    `skilldozer is not configured; run \`skilldozer init\`` to stderr, writes
  - REPLACE: `skilldozer is not configured; run \`skilldozer --init\`` to stderr, writes

EDIT 12 — L330 (Constraints prose): convert `skilldozer init`
  - FIND:  config file (the store location, written by `skilldozer init`) is expected and
  - REPLACE:  config file (the store location, written by `skilldozer --init`) is expected and

EDIT 13 — MOVE the `## Shell completions` section to §15 position 8 (LAST; GOTCHA #4/#5)
  - CUT the contiguous block from the header `## Shell completions` through the blank line
    immediately before `## Usage` (currently L94-L151, but anchor by HEADER, not number).
  - PASTE it after the LAST line of the `## How \`skilldozer\` finds the store` section —
    i.e. immediately before the `## Constraints` header. Keep a blank line on each side of
    the moved block so the Markdown structure stays clean (no doubled headers, no missing
    blank lines).
  - RESULTING §15 order: Title → Why → Install → Usage → Where skills live → Adding a skill
    → How finds store → **Shell completions** → Constraints. (research §5.)
  - Do this AFTER EDITS 1-12 (the move shifts line numbers; content edits are text-anchored).
```

### Implementation Patterns & Key Details

```markdown
<!-- The conversion is mechanical (s/init/--init/ etc.) EXCEPT two judgment calls: -->

<!-- 1. The Reserved-paragraph replacement (EDIT 8) must be GREP-SAFE. The contract's suggested -->
<!--    "A skill named check/init/completions resolves normally" is safe (no `skilldozer X`). -->
<!--    Do NOT write "`skilldozer check` resolves the tag check" — that fails the gate. -->

<!-- 2. The Completions description (EDIT 5) must list EXACTLY the 13 alphabetical long flags -->
<!--    (--all … --version), describe skills-first (<tab>=skills), long-form-only (-<tab>=long -->
<!--    flags, no short aliases), and the --init/--store→dirs + --search→nothing routing. Source: -->
<!--    PRD §14.1/§14.6, which the rewritten completion files (sibling) also implement. -->
```

Notes easy to get wrong:
- **Every** `skilldozer init`/`check`/`completion` literal must go (research §2 is the checklist). A single missed reference fails the grep gate.
- The Reserved replacement must avoid the `skilldozer <word>` literal (GOTCHA #1) — use "A skill named `check`" + the `--flag`.
- The quoted "unconfigured" message (EDIT 11) must say `--init` to match the live binary (GOTCHA #3).
- Do the move LAST, header-anchored (GOTCHA #4/#5).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Convert prose command-name refs too (not just `skilldozer X` literals).** The grep only catches the full form, but the contract LOGIC (a) says "ALL bare subcommand references." Lines like "`--store` implies init" / "the `completion` subcommand" describe the command by bare name and would read as stale. Flipping them to `--init`/`--completions` makes the README fully accurate. (research §2; GOTCHA #2.)
2. **Grep-safe Reserved replacement (EDIT 8).** The contract's literal suggested text is safe, but an illustrative "`skilldozer check` resolves the tag" is NOT (fails the gate). The chosen phrasing ("A skill named `check` … use `--check`") conveys the same point (no reserved names; tags vs flags) without the failing literal. (research §3; GOTCHA #1.)
3. **Append the skills-first description to the Completions section (EDIT 5), don't just convert commands.** The contract LOGIC (c) requires describing the headline `<tab>`=skills UX. The 13-flag list is sourced verbatim from PRD §14.6 (and matches the sibling's rewritten completion files). `--shell` is deliberately excluded (it's a `--completions` modifier, not a top-level flag). (GOTCHA #6.)
4. **MOVE the completions section to §15 position 8.** PRD §15 is prescriptive and lists completions as item 8 (after "How finds store"). The current README has it at position 4. One move (4→8) makes the whole README match §15 (the other sections are already in relative order). Done last, header-anchored, to avoid line-number churn. (research §5; GOTCHA #4/#5.)
5. **Error-contract example `init --store` → `--init --store` (EDIT 7).** Preserves the original "explicit script form" example structure (the canonical `--init --store` pair). `--store` alone would also be correct, but `--init --store` matches the surrounding "self-evident intent" prose.
6. **README only; no code/test/completion changes.** Mode B documentation. The binary already emits the `--flag` contract; this subtask only brings the doc in line. (research §8; GOTCHA #7.)

### Integration Points

```yaml
FILES TOUCHED:
  - README.md ONLY. (GOTCHA #7)

DEPENDENCIES (already satisfied — this subtask consumes their outputs, does not change them):
  - P1.M1 (--flag code, Complete): the binary now parses --check/--init/--completions.
  - P1.M2.T1 (completion files rewritten): the README's Completions description mirrors their behavior.
  - internal/skillsdir/skillsdir.go:275 (ErrNotFound): the README's quoted "unconfigured" message
    matches it (already --init).

NO ROUTES / NO DATABASE / NO CODE / NO TESTS / NO NEW FILES / NO DEPS.
```

---

## Validation Loop

### Level 1: Markdown sanity (immediate, after the edits)

```bash
cd /home/dustin/projects/skilldozer

# No code to compile/lint — README is Markdown. Sanity-check structure:
grep -c '^## ' README.md          # count top-level sections (should be ~9 per §15)
grep -n '^## ' README.md          # list them IN ORDER — verify §15 sequence:
                                 # Title(one-liner) / Why / Install / Usage / Where skills live /
                                 # Adding a skill / How finds store / Shell completions / Constraints
# Expected: Shell completions appears AFTER "How `skilldozer` finds the store" and BEFORE Constraints.
```

### Level 2: The grep gate (the hard validation — all 3 must pass)

```bash
cd /home/dustin/projects/skilldozer

# (a) new --completions commands are present (EDIT 5):
grep -q 'skilldozer --completions' README.md && echo "PASS: --completions present" || echo "FAIL"

# (b) the Reserved paragraph is gone (EDIT 8):
! grep -q 'Reserved tag names' README.md && echo "PASS: Reserved removed" || echo "FAIL: still present"

# (c) NO bare subcommands remain (EDITS 1-12; GOTCHA #1 grep-safety):
! grep -Eq 'skilldozer (init|check|completion)\b' README.md && echo "PASS: no bare subcommands" || echo "FAIL: bare ref remains"
# Expected: all three print PASS.
```

### Level 3: Content spot-checks (faithfulness to the --flag contract)

```bash
cd /home/dustin/projects/skilldozer

# The eval/source one-liners are now --completions:
grep -q 'eval "$(skilldozer --completions)"' README.md && echo "eval OK"
grep -q 'skilldozer --completions --shell fish | source' README.md && echo "fish OK"

# The skills-first description is present (EDIT 5):
grep -q 'skills-first and long-form-only' README.md && echo "skills-first desc OK"
grep -q -- '--all`, `--check`, `--completions`' README.md && echo "13-flag list OK"

# The Reserved paragraph's REPLACEMENT is present and grep-safe (EDIT 8):
grep -q 'no reserved tag names' README.md && echo "replacement note OK"
# (and it must NOT contain the failing literal — re-confirm gate (c) above passes.)

# The quoted unconfigured message matches the binary (EDIT 11):
grep -q 'run \\`skilldozer --init\\`' README.md && echo "unconfigured msg OK" \
  || grep -q 'run .skilldozer --init.' README.md && echo "unconfigured msg OK"

# Spot-check a few converted refs:
grep -q '`skilldozer --init` (see First run' README.md && echo "L43 OK"
grep -q 'skilldozer --check' README.md && echo "check→--check OK"
# Expected: all print OK.
```

### Level 4: Exhaustive no-bare sweep + ordering confirm (the belt-and-suspenders)

```bash
cd /home/dustin/projects/skilldozer

# 4a. The combined word-boundary grep must return NOTHING (the authoritative no-bare check):
grep -En 'skilldozer (init|check|completion)\b' README.md && echo "FAIL: list above are bare refs" || echo "PASS: zero bare subcommand refs"

# 4b. Also confirm no stray bare command-NAME prose slipped through (informational; not gated):
grep -nE '`init`|`check`|`completion`' README.md | grep -viE 'init\b.*--init|--check|--completions|initialize|initial' || echo "PASS: no stale bare-name prose"

# 4c. Section ordering matches PRD §15 (completions is item 8, after "How finds store"):
awk '/^## /{print NR": "$0}' README.md
# Expected: the "## Shell completions" line number is GREATER than "## How `skilldozer` finds the store"
# and LESS than "## Constraints".

# 4d. Human read-through: open README.md, confirm every code-block command + prose command is a --flag,
#     the Completions section reads well, and the Reserved paragraph is replaced by the one-line note.
# Expected: 4a PASS; 4b PASS (or only intentional `--flag` matches); 4c ordering correct; 4d reads clean.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — README has ~9 `##` sections; the §15 sequence is correct (completions after "How finds store")
- [ ] Level 2 PASS — all 3 grep assertions print PASS (`--completions` present; `Reserved tag names` absent; no `skilldozer (init|check|completion)\b`)
- [ ] Level 3 PASS — eval/fish one-liners, skills-first description, 13-flag list, replacement note, unconfigured message all present
- [ ] Level 4 PASS — exhaustive no-bare sweep returns nothing; ordering confirmed; human read-through clean

### Feature Validation
- [ ] Every `skilldozer init`/`check`/`completion` literal → `--init`/`--check`/`--completions`
- [ ] Every prose command-name ref (`init`/`check`/`completion` as commands) → `--flag` form
- [ ] "Reserved tag names" paragraph deleted; grep-safe one-line note replaces it
- [ ] Completions section: `--completions` load commands + skills-first/long-form-only description (13 flags)
- [ ] Section order matches PRD §15 (completions = item 8)

### Scope Discipline
- [ ] ONLY `README.md` modified
- [ ] Did NOT touch main.go, main_test.go, internal/*, completions/*, go.mod, go.sum
- [ ] Did NOT modify PRD.md (read-only), tasks.json, prd_snapshot.md, or .gitignore
- [ ] The quoted "unconfigured" message matches the live binary (`--init`, skillsdir.go:275)

---

## Anti-Patterns to Avoid

- ❌ **Don't write `skilldozer check`/`init`/`completion` in the Reserved replacement.** The grep gate `! grep -Eq 'skilldozer (init|check|completion)\b'` catches it. Use "A skill named `check`" + the `--flag` form. (GOTCHA #1.)
- ❌ **Don't convert only the `skilldozer X` literals.** The 8 prose command-name refs (`init`/`check`/`completion` as commands) must also flip to `--flag` for accuracy. (GOTCHA #2.)
- ❌ **Don't quote the old unconfigured message.** The binary now says `run \`skilldozer --init\`` (skillsdir.go:275); README must match. (GOTCHA #3.)
- ❌ **Don't do the move before the content edits.** The move shifts line numbers; do it LAST, header-anchored. (GOTCHA #4/#5.)
- ❌ **Don't edit any file but README.md.** This is Mode B documentation; the code/completions/tests are done. (GOTCHA #7.)
- ❌ **Don't add `--shell` to the 13-flag list.** It is a `--completions` modifier, not a top-level flag (PRD §14.6). (GOTCHA #6.)
- ❌ **Don't drop or duplicate a `##` header in the move.** Cut one contiguous block; paste it whole before `## Constraints`, with blank lines on both sides. (GOTCHA #5.)

---

## Confidence Score

**9/10** — This is a documentation-only edit to one Markdown file, with every change pinned to exact line numbers + before/after text transcribed from the live README, and a hard grep gate that deterministically verifies completeness. The one non-obvious trap — that the contract's *suggested* Reserved-paragraph replacement could contain `` `skilldozer check` `` and fail the gate — is identified and resolved with a grep-safe phrasing (`research/verified_facts.md` §3). The binary's current `--init` message is confirmed so the quoted line matches. The section move is header-anchored (unambiguous boundaries) and done last. The 1-point reservation is for the human-read-through subjectivity (does the rewritten Completions section read naturally?) and the small risk of an overlooked prose command-name ref that the grep (which only catches the full `skilldozer X` form) wouldn't catch — mitigated by the explicit 8-ref prose list in research §2 and the Level 4b informational sweep.
