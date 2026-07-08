name: "P1.M3.T1.S1 — Mode B README sweep: reconcile init/error-contract sections with the bugfix changeset"
description: |

---

## Goal

**Feature Goal**: Make `README.md` coherently reflect the whole bugfix changeset (Issues 1-7) by sweeping its init ("First run") and error-contract ("**Error contract.**") sections for the user-facing behavior the fixes changed, and verifying (not duplicating) the reserved-tag note the parallel P1.M2.T5.S1 already landed. This is the one changeset-level documentation sync (SOW §5 Mode B; decisions.md §D7) that spans every implementing subtask.

**Deliverable**: Up to TWO small, voice-consistent edits to `/home/dustin/projects/skilldozer/README.md`:
1. **First run section** — add a concise note that `init` prints ONLY the store path to stdout (clean for `$(...)`), with status + the check report on stderr (Issue 1); and that a leading `~` expands to `$HOME` in typed/`--store` paths (Issue 5).
2. **Error contract** — expand the exit-2 enumeration so it states a tag combined with `--path`/`--list`/`--search`/`--all` exits 2 (Issue 3), and that `--store` with no value exits 2 (Issue 2).
Plus a **verification** (no edit) that the reserved-tag note (Issue 7) is present in "Where skills live" and consistent with the error-contract and skill-tag sections.

**Success Definition**: a user reading the README's init section learns that `STORE="$(skilldozer init --store /p)"` yields a clean single-line value and that `~` works; a user reading the error contract learns that tag+mode and bare `--store` both exit 2. The README retains its voice (zero `§`-PRD citations; bold-lead-in callouts; inline `` `code` ``). `go build/vet/test ./...` stays green (no code change). No file other than README.md is touched; the reserved-tag note is verified present, not rewritten.

---

## User Persona (if applicable)

**Target User**: A `skilldozer` user scripting first-run setup (`STORE="$(skilldozer init --store /path)"` in CI/install scripts), and any user composing flags at the command line.

**Use Case**: (1) A CI author captures init's output and needs to know it's a clean single line. (2) A user types `skilldozer myskill --path` and the docs should set the expectation that a tag + an inspection mode is an error, not a silent drop.

**User Journey**: After this sweep, the README's init section and error contract match what the binary actually does post-bugfix, so the docs and behavior no longer diverge on init's stdout, tilde handling, or the tag+mode / `--store`-no-value exit codes.

**Pain Points Addressed**: the Issue 1 fix (init stdout = store path only) is invisible to users unless documented; the Issue 3 fix (tag+`--path` now rejected) would surprise a user whose mental model came from the old README ("only the listing modes are mutually exclusive"); the Issue 2 fix (`--store` no-value → exit 2) should be discoverable.

---

## Why

- **Closes the Mode B documentation gap** (SOW §5; decisions.md §D7): each implementing subtask updated the doc it directly touches (Mode A — inline code comments + the reserved-tag note); THIS task is the one sweep that reconciles the README's overview/feature sections across the whole delta. It depends on every implementing subtask and runs last.
- **Issues 1, 2, 3, 5 changed user-visible behavior** (init stdout stream; `--store`-no-value exit code; tag+mode exit code; tilde expansion) — the README's init and error-contract sections are the canonical user-facing statements of those behaviors and must now agree with `main.go`.
- **Issues 4, 6, 7 need no new README prose** (Issue 4 is subsumed by the general exclusivity framing; Issue 6 is not user-facing; Issue 7 is already documented by P1.M2.T5.S1). The contract's "note explicitly rather than forcing an edit" rule applies to these.
- **Consumed by**: end users and the §13/§16 acceptance review — a README that contradicts the binary fails a careful reviewer even if the code is correct.

---

## What

[Mode B] Read `README.md` end-to-end and reconcile. Two warranted edits + one verification:

**(A) First run section (Issue 1 + Issue 5).** After the `skilldozer init --store /path/to/store` code block (before `## Shell completions`), add a short paragraph stating: on success `init` prints exactly the configured store path to stdout (one clean line, so `$(...)` works in scripts); the seeded/adopted status and the post-setup `check` report go to stderr; and a leading `~` (or a bare `~`) in a typed answer or a `--store`/positional path expands to the home directory.

**(B) Error contract (Issue 2 + Issue 3).** Replace the final sentence of the `**Error contract.**` callout ("The listing modes `--path`, `--list`, `--search`, and `--all` are mutually exclusive — combining any two exits 2.") with text that also states: a tag combined with any of `--path`/`--list`/`--search`/`--all` exits 2 (Issue 3); and `--store` expects a value, so `init --store` with nothing after it exits 2 rather than guessing a store (Issue 2).

**(E) Verify (no edit) the reserved-tag note (Issue 7).** Confirm the `**Reserved tag names.**` note is present in "Where skills live" and reads consistently with the error-contract (exit codes) and skill-tag (resolution) sections. Do NOT rewrite or duplicate it (it is P1.M2.T5.S1's deliverable; contract DOCS step 5).

### Success Criteria

- [ ] The "First run" section states `init` prints ONLY the store path to stdout (clean for `$(...)`) and that status/check output goes to stderr.
- [ ] The "First run" section states a leading `~` (or bare `~`) expands to `$HOME` in typed/`--store`/positional paths.
- [ ] The "**Error contract.**" callout states a tag combined with any of `--path`/`--list`/`--search`/`--all` exits 2.
- [ ] The "**Error contract.**" callout states `--store` with no value exits 2 (config is not written).
- [ ] The reserved-tag note (`**Reserved tag names.**`) is present in "Where skills live" (verified; not edited by this task).
- [ ] `grep -c '§' README.md` is still `0` (no PRD section citations added).
- [ ] Only `README.md` is modified; `main.go`/`main_test.go`/`internal/*`/`.gitignore`/`PRD.md`/`tasks.json` unchanged.
- [ ] `go build ./... && go vet ./... && go test ./...` all green (no code change — the doc-sanity check).
- [ ] Edits are minimal and in the existing README voice (plain prose, inline `` `code` ``, bold-lead-in callouts where used).

---

## All Needed Context

### Context Completeness Check

**Pass.** The two edit zones were read verbatim (current text quoted in research/verified_facts.md §2, with exact anchor strings for matching). The behavior to document was verified DIRECTLY in `main.go` at HEAD for every relevant fix: Issue 1 (runInit check-report → `stderr`; only the store-path `Fprintln(stdout, dir)` stays on stdout), Issue 2 (`storeMissingValue` → `run()` guard `skilldozer: --store requires a value` exit 2, config not written; `--search` no-value is NOT covered — accuracy note), Issue 3 (`exclusivityError` `hasTags && (c.path || c.list || c.searchMode || c.all)` → exit 2), Issue 5 (`expandHome` @ main.go:894 wired @ main.go:953 before `filepath.Abs`). The reserved-tag note (Issue 7) was confirmed ALREADY PRESENT in README "Where skills live", matching P1.M2.T5.S1's PRP verbatim. The README voice rules are grep-confirmed (`grep -c '§'` == 0; callout convention pinned by `**Error contract.**`). The parallel-execution/file-conflict consideration is analyzed (T5.S1's insertion is below both edit zones; seams pinned by anchor text, not line number). An implementer who has never seen this repo can complete it in one pass.

### Documentation & References

```yaml
# MUST READ — the verified facts (zones, landed behavior, voice, scope, conflict analysis)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M3T1S1/research/verified_facts.md
  why: "§1 = the 4 touchpoints (a-d) + consistency check (e) as a table with current-state + action.
        §2 = the VERBATIM current text of both edit zones (the exact anchor strings to match).
        §3 = the landed behavior per issue (verified in main.go) incl. the CRITICAL accuracy note
        that --search no-value is NOT exit 2 (only --store is). §4 = the reserved-tag note is
        ALREADY landed (verify only). §5 = README voice rules. §6 = the parallel-conflict analysis
        (pin seams by anchor text). §7 = scope boundary."
  critical: "§3 accuracy note: do NOT claim `--search` no-value exits 2 — only `--store` got the
             Issue 2 exit-2 fix (--search no-value still falls to usage/exit 1). §4/§6: the
             reserved-tag note is T5.S1's landed deliverable — VERIFY, do not duplicate, and do not
             touch the 'Where skills live' section. Pin every edit seam by ANCHOR TEXT, not line
             number (the file is in flux from T5.S1)."

# MUST READ — the file under edit (the ONLY deliverable)
- file: README.md
  why: "THE edit target. (A) '### First run' — insert the Issue 1 + Issue 5 note AFTER the
        `skilldozer init --store /path/to/store` code block, BEFORE `## Shell completions`. (B)
        '## Usage' '**Error contract.**' callout — REPLACE its final sentence
        ('The listing modes … combining any two exits 2.') with the Issue 2 + Issue 3 expansion.
        Read the WHOLE file end-to-end first (the contract's 'reconcile' instruction)."
  pattern: "README callout = `**Bold lead-in.**` + prose paragraph (template: `**Error contract.**`
            at the Zone B anchor). Inline `code` for commands/paths/flags. Em dashes (—) are
            voice-consistent (used elsewhere). Plain declarative prose; one idea per sentence."
  gotcha: "VOICE: `grep -c '§' README.md` is 0 — NEVER cite PRD sections ('§6.1', '§8.2') in the
           README. Use plain words ('prints the store path', 'runs setup'). The contract's '(§8.2,
           §9)' etc. point the PRP writer at the PRD, not the README reader."

# MUST READ — the parallel sibling contract (produces the reserved-tag note this task verifies)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M2T5S1/PRP.md
  why: "Defines the EXACT reserved-tag note text + placement (end of 'Where skills live'). Its OUTPUT
        says 'Consumed by: users reading the README and by the final Mode B consistency sweep
        (P1.M3.T1)'; its DOCS says 'the final Mode B sweep (P1.M3.T1) verifies README consistency
        across all 7 fixes but does NOT duplicate this note.' So this task VERIFIES that note — it
        does not rewrite it. (T5.S1 is 'Implementing' in parallel; by this task's execution time it
        is landed, so the note is present — confirmed directly in README at PRP-write time.)"
  critical: "Do NOT edit the 'Where skills live' section. T5.S1 owns it. Your edits are confined to
             '### First run' and the '## Usage' Error contract — both ABOVE T5.S1's insertion."

# READ-ONLY — the landed behavior (verify the docs match the code; do NOT edit main.go)
- file: main.go
  why: "CONFIRMS the behavior to document (all fixes landed): runInit check-report → stderr
        (main.go:~1090-1100, only `Fprintln(stdout, dir)` stays on stdout); `storeMissingValue`
        guard (main.go:461 `skilldozer: --store requires a value` exit 2); exclusivityError tags
        predicate (main.go:737 `hasTags && (c.path || c.list || c.searchMode || c.all)`); expandHome
        (main.go:894) wired into resolveStore (main.go:953) before filepath.Abs. READ-ONLY here."
  gotcha: "Do NOT edit main.go. The docs must MATCH this code; if a README claim disagrees with
           main.go, fix the README, not the code (the code is the post-fix truth)."

# READ-ONLY — the authoritative issue writeups (the changeset this sweep documents)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "Issues 1-7 with confirmed repros + exact file:line + fix. Use to double-check each README
        claim against the intended fix (esp. Issue 2's --search-is-harmless note and Issue 5's
        both-interactive-and-noninteractive scope)."
  section: "ISSUE 1, ISSUE 2, ISSUE 3, ISSUE 5 (the user-facing ones); ISSUE 7 (the note)."

# READ-ONLY — the decisions that bound this task
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/decisions.md
  why: "§D7 = THIS task (Mode B sweep depends on all implementing subtasks). §D6 = Issue 7 is
        documentation-only (the reserved-tag note is the deliverable; no PRD edit). §D1 = Issue 1
        fix scope (stderr, no refactor). Bounds what the README should and should not claim."
  section: "D7, D6, D1."

# READ-ONLY — system context (README touchpoints + exit-code contract)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/system_context.md
  why: "Enumerates the README candidate sections (init, error/exit-code) and the exit-code contract
        (0/1/2 semantics) the error-contract callout must reflect. Exit 2 = unknown flag OR
        mutually-exclusive modes mixed (and after the fixes: tag+path, init+init, --store-no-value)."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer
$ grep -n '^## \|^### ' README.md | head -20   # section boundaries (pin by TEXT, not number)
 5:## Why
19:## Install
61:### First run                       ← EDIT ZONE A (Issue 1 + Issue 5): insert note after its code block
80:## Shell completions
116:## Usage
164:**Error contract.**                 ← EDIT ZONE B (Issue 2 + Issue 3): expand its final sentence
... ## Where skills live  ← VERIFY ONLY (Issue 7 reserved-tag note; T5.S1 owns it)
# main.go / main_test.go / internal/* / .gitignore / PRD.md / tasks.json — UNCHANGED.
```
(Lines shift as T5.S1 lands its note (~line 200). Zones A and B are ABOVE that insertion, so they
are region-stable — but ALWAYS locate the seams by the anchor text in research/verified_facts.md §2.)

### Desired Codebase tree with files to be changed

```bash
README.md    # EDIT (A) First run: +1 paragraph (Issue 1 stdout/stderr + Issue 5 tilde)
             # EDIT (B) Error contract: replace final sentence (Issue 3 tag+mode + Issue 2 --store no-value)
             # VERIFY  Where skills live: reserved-tag note present + consistent (Issue 7) — NO edit
# every other file UNCHANGED.
```

| File | Change | Why |
|---|---|---|
| `README.md` | (A) +1 paragraph in "First run"; (B) replace the error contract's last sentence. | Document Issues 1, 2, 3, 5; verify Issue 7. |

### Known Gotchas of our codebase & Library Quirks

```markdown
<!-- GOTCHA #1 (VOICE — the #1 slip) — the user-facing README NEVER cites PRD sections.
     `grep -c '§' README.md` is 0. Do NOT write "§6.1", "§8.2", "§9". Use plain words
     ("prints the store path", "runs validation/setup"). The contract's "(§8.2, §9)" point the PRP
     writer at the PRD, not the README reader. -->

<!-- GOTCHA #2 (ACCURACY — do NOT overclaim --search) — only `--store` got the Issue 2 exit-2 fix.
     `--search`/`-s` with no value is NOT covered: it still falls through to usage/exit 1 (harmless).
     So the error contract must say "`--store` expects a value" (exit 2), NOT "`--store`/`--search`".
     Verified: `storeMissingValue` is set only in the two `--store` branches (main.go:201, 275). -->

<!-- GOTCHA #3 (PARALLEL FILE — T5.S1 edits README too) — the parallel sibling P1.M2.T5.S1 inserts
     the reserved-tag note at the END of "Where skills live" (~line 200). Both edit zones here
     (First run, Error contract) are ABOVE that insertion (region-disjoint). At THIS task's execution
     time T5.S1 is landed (P1.M3 depends on P1.M2.T5), so there is no concurrent write — but ALWAYS
     locate edit seams by ANCHOR TEXT (see research §2), not line number, to survive any shift.
     Do NOT touch "Where skills live" — VERIFY its note only. -->

<!-- GOTCHA #4 (init stdout is EXACTLY one line — accuracy) — after Issue 1, on the SUCCESS path
     `init` stdout is ONE line (the store path): the OK/WARN/ERROR report, the summary, the
     seeded/adopted status, and "(found via …)" are ALL on stderr. On the FAILURE path stdout is
     empty + exit 1. So `STORE="$(skilldozer init --store /p)"` is clean. State it as "exactly the
     store path" / "one clean line" — do NOT imply the report is also on stdout. -->

<!-- GOTCHA #5 (tilde scope — BOTH interactive and non-interactive) — Issue 5's expandHome runs in
     resolveStore, which BOTH the interactive prompt answer and the `--store`/positional non-
     interactive path flow through. So say "~ works in a typed answer OR a --store/positional path",
     not just one. A bare `~` and a leading `~/` both expand (to $HOME). -->

<!-- GOTCHA #6 (NO CODE / NO TEST) — this is documentation-only (Mode B). Do NOT edit main.go,
     main_test.go, internal/*, .gitignore. `go build/vet/test ./...` must be byte-for-byte unchanged
     in behavior — run them as the doc-sanity proof (contract LOGIC: "re-build the binary, re-run
     go test ./... to confirm no code regression"). -->

<!-- GOTCHA #7 (NO PRD EDIT) — PRD.md is human-owned/READ-ONLY. decisions.md §D6: the §7.2 note
     suggested by the PRD is a HUMAN action item, not this task's. Do NOT touch PRD.md, tasks.json,
     or prd_snapshot.md. -->

<!-- GOTCHA #8 (DO NOT DUPLICATE Issue 7) — the reserved-tag note is P1.M2.T5.S1's canonical
     deliverable and is ALREADY landed. This task VERIFIES it is present and consistent (with the
     error-contract exit framing and the skill-tag resolution framing) but does NOT restate or move
     it. Adding the rule to "Usage" or "First run" would violate contract DOCS step 5. -->

<!-- GOTCHA #9 (MINIMAL — let judgment decide) — the contract says "Keep edits minimal" and "If a
     section needs no change, note that explicitly rather than forcing an edit." Issues 4 (init init)
     and 6 (.gitignore) have NO direct README touchpoint — do NOT force prose for them (Issue 4 is
     subsumed by the general exclusivity framing; Issue 6 is not user-facing). State in the PRP/task
     notes that they were considered and intentionally omitted. -->
```

---

## Implementation Blueprint

### Data models and structure

**None.** This is a prose edit to a Markdown file. No types, fields, code, config, or tests. The two edited paragraphs ARE the deliverable.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: READ README.md end-to-end + confirm the reserved-tag note is present (Issue 7 verify)
  - ACTION: read the whole README; locate the three relevant zones by ANCHOR TEXT (not line number):
      (A) "### First run" and its trailing `skilldozer init --store /path/to/store` code block;
      (B) the "**Error contract.**" callout in "## Usage" and its final sentence;
      (E) the "**Reserved tag names.**" note in "## Where skills live".
  - VERIFY (E): the reserved-tag note IS present and names `check`/`init` as reserved subcommands
      with the four workarounds (--list/--all, nested path, frontmatter name, alias). If present
      and consistent with the error-contract (exit codes) and skill-tag (resolution) framing →
      NO EDIT (Issue 7 is done by T5.S1). Do NOT rewrite or move it.
  - NOTE in task output: Issues 4 (init init) and 6 (.gitignore) were considered and intentionally
      omitted (no direct README touchpoint; subsumed / not user-facing).

Task 1: EDIT (A) — append the Issue 1 + Issue 5 paragraph to "First run"
  - FILE: README.md
  - ANCHOR (locate by this text — the end of the First run code block, before the next heading):
        skilldozer init --store /path/to/store
        ```
        <blank>
        ## Shell completions
  - INSERT a new paragraph BETWEEN the closing ``` fence of the init code block and the
    `## Shell completions` heading (preserve one blank line each side). RECOMMENDED WORDING (adjust
    for voice; keep the facts — Issue 1 stdout/stderr + Issue 5 tilde):
        On success, `init` prints exactly the configured store path to stdout — one clean line, so
        `STORE="$(skilldozer init --store /path)"` works in scripts. The seeded/adopted status and
        the post-setup `check` report go to stderr. A leading `~` (or a bare `~`) in a typed answer
        or a `--store`/positional path expands to your home directory.
  - REQUIRED CONTENT (the paragraph MUST convey): (i) init stdout = ONLY the store path (one line,
    clean for `$(...)`); (ii) status + check report → stderr; (iii) `~`/`~/...` → $HOME in BOTH
    typed and --store/positional paths.
  - VOICE: plain prose, inline `code`, em dash OK, NO `§` citations.

Task 2: EDIT (B) — expand the Error contract's final sentence (Issue 2 + Issue 3)
  - FILE: README.md
  - ANCHOR (locate by this text — the final sentence of the **Error contract.** callout):
        The listing modes `--path`,
        `--list`, `--search`, and `--all` are mutually exclusive — combining any two
        exits 2.
  - REPLACE those three lines with (RECOMMENDED WORDING — Issue 3 tag+mode + Issue 2 --store):
        The `--path`, `--list`, `--search`, and `--all` modes are mutually exclusive — combining any
        two exits 2, as does combining a tag with any of them (a tag resolves one path; those modes
        inspect the whole store). `--store` expects a value: `init --store` with nothing after it
        exits 2 rather than guessing a store.
  - REQUIRED CONTENT: (i) a tag + any of --path/--list/--search/--all → exit 2; (ii) `--store` with
    no value → exit 2. ACCURACY (GOTCHA #2): do NOT mention `--search` no-value as exit 2 — only
    `--store`.
  - PRESERVE the callout's opening sentences (unknown tag → stdout empty + exit 1; multi-tag
    atomicity) UNCHANGED. Keep the `**Error contract.**` bold lead-in.

Task 3: VERIFY isolation + doc sanity (the acceptance loop)
  - go build ./... && go vet ./... && go test ./...   # all exit 0 (no code change)
  - grep -c '§' README.md                             # MUST be 0
  - git status --short README.md                      # shows " M README.md"
  - grep-based content checks (see Validation Loop Level 2)
  - manual render check: read the two edited zones + the verified reserved-tag note in context.
```

### Implementation Patterns & Key Details

```markdown
# The two edits are prose insertions/replacements at anchor-located seams. The house callout
# pattern is `**Bold lead-in.**` + prose (template: `**Error contract.**`). Edit (A) adds a NEW
# paragraph (not a callout — it's a follow-on note in the First run section, plain prose is fine);
# Edit (B) REPLACES the last sentence of an existing callout (keep the `**Error contract.**` lead-in
# and its first sentences).

# Edit (A) in context (before -> after):
#   BEFORE:
#     ```bash
#     skilldozer init /path/to/store      # positional
#     skilldozer init --store /path/to/store
#     ```
#
#     ## Shell completions
#
#   AFTER:
#     ```bash
#     skilldozer init /path/to/store      # positional
#     skilldozer init --store /path/to/store
#     ```
#
#     On success, `init` prints exactly the configured store path to stdout — one clean line, so
#     `STORE="$(skilldozer init --store /path)"` works in scripts. The seeded/adopted status and
#     the post-setup `check` report go to stderr. A leading `~` (or a bare `~`) in a typed answer
#     or a `--store`/positional path expands to your home directory.
#
#     ## Shell completions

# Edit (B) in context (before -> after):
#   BEFORE (final sentence):
#     ... so `pi` never sees a partial result. The listing modes `--path`,
#     `--list`, `--search`, and `--all` are mutually exclusive — combining any two
#     exits 2.
#
#   AFTER (final sentences):
#     ... so `pi` never sees a partial result. The `--path`, `--list`, `--search`, and `--all`
#     modes are mutually exclusive — combining any two exits 2, as does combining a tag with any
#     of them (a tag resolves one path; those modes inspect the whole store). `--store` expects a
#     value: `init --store` with nothing after it exits 2 rather than guessing a store.
```

Notes easy to get wrong:
- Claiming `--search` no-value exits 2 (GOTCHA #2) — only `--store` does.
- Implying the check report is on stdout (GOTCHA #4) — it's on stderr now; stdout is ONE line.
- Limiting tilde to one path form (GOTCHA #5) — it covers typed AND --store/positional.
- Adding a `§` citation (GOTCHA #1) — `grep -c '§'` must stay 0.
- Editing "Where skills live" or duplicating the reserved-tag note (GOTCHA #8) — verify only.
- Locating seams by line number (GOTCHA #3) — use the anchor text (the file shifts as T5.S1 lands).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Edit (A) is warranted even though the contract's (a) is conditional.** The contract says "if README describes init's stdout, ensure it says ONLY the store path" — the README does NOT currently describe init's stdout. But the Issue 1 fix's entire user-facing payoff (`$(skilldozer init)` is clean) is invisible unless documented, and (d) explicitly requires the tilde mention. So a single combined paragraph covering both is the minimal, warranted edit. (research §1.)
2. **Edit (B) is required (not conditional).** The contract's (b) ("add to the error-contract if it enumerates exit-2 cases") and (c) ("ensure tags+--path is reflected") both apply: the callout DOES enumerate exit-2 cases, and it IS silent on tags+mode. Both gaps must close. (research §1.)
3. **`--store` only, not `--store`/`--search`, in the no-value claim.** Verified in main.go: `storeMissingValue` is set only for `--store`. `--search` no-value still exits 1 (usage). Claiming otherwise would make the docs wrong. (GOTCHA #2, research §3.)
4. **Issues 4 and 6 intentionally omitted.** Issue 4 (init init) is subsumed by the general exclusivity framing the error contract already carries; Issue 6 (.gitignore) is not user-facing. The contract's "note explicitly rather than force an edit" rule applies. (GOTCHA #9, research §1/§7.)
5. **Issue 7 is verify-only.** The note is T5.S1's landed deliverable and is already present + correct. Duplicating it would violate contract DOCS step 5. (GOTCHA #8, research §4.)
6. **Anchor text, not line numbers.** T5.S1 edits README.md in parallel; pinning seams by anchor text makes the edit robust to any line shift and is clearer anyway. (GOTCHA #3, research §6.)

### Integration Points

```yaml
DOCUMENTATION (Mode B — the deliverable IS the doc edits):
  - file: README.md
  - sections: "### First run" (Edit A) and "## Usage" → "**Error contract.**" (Edit B)
  - effect: "README init + error-contract sections now match the post-bugfix binary: init stdout =
            store path only (status/check on stderr), tilde expands, tag+mode and bare --store exit 2."
  - verify-only: "## Where skills live" → "**Reserved tag names.**" (Issue 7; T5.S1 owns it).

CODE: NONE.
  - main.go / main_test.go / internal/* / .gitignore UNCHANGED. No Go file read-for-write.
  - `go build/vet/test ./...` unaffected (the contract's doc-sanity check).

PRD.md / tasks.json / prd_snapshot.md: READ-ONLY (never touched — §D6 + global FORBIDDEN OPERATIONS).

PARALLEL SIBLING (no conflict):
  - P1.M2.T5.S1 edits "Where skills live" (the reserved-tag note). This task edits "First run" +
    "Usage/Error contract" — DISJOINT regions (both above T5.S1's insertion). At execution time
    T5.S1 is landed; seams located by anchor text survive any shift.

NO DATABASE / NO ROUTES / NO CONFIG-FORMAT CHANGE / NO CODE CHANGE / NO NEW FILES.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after the edits)

```bash
cd /home/dustin/projects/skilldozer

# Voice invariant — NO PRD section citations were added:
grep -c '§' README.md          # MUST be 0
# Expected: 0.

# Render check — eyeball both edited zones in context (locate by anchor text, not line number):
sed -n "/skilldozer init --store \/path\/to\/store/,/^## Shell completions/p" README.md
sed -n '/^\*\*Error contract\.\*\*/,/^`skilldozer --help`/p' README.md
# Expected: Edit A paragraph sits between the init code block and "## Shell completions";
#           Edit B's expanded final sentences are inside the **Error contract.** callout.
```

### Level 2: Required-content checks (the contract's OUTPUT)

```bash
cd /home/dustin/projects/skilldozer

# Edit A — First run (Issue 1 + Issue 5). All must match (adjust regex to your final wording):
grep -qE 'init.*prints.*(store path|configured store path).*stdout|stdout.*store path' README.md && echo "A: stdout=store path OK"
grep -qE 'status|check report|seeded|adopted' README.md && echo "A: status/check on stderr OK"
grep -qE 'home directory|\$HOME|~/' README.md && echo "A: tilde expansion OK"
# (Confirm the stdout/stderr + tilde claims are in the FIRST RUN section, not elsewhere.)

# Edit B — Error contract (Issue 2 + Issue 3):
#   Issue 3: tag + a mode exits 2
grep -qE 'tag.*(combined|with|mixing).*(--path|--list|--search|--all)|(--path|--list|--search|--all).*tag' README.md && echo "B: tag+mode exit 2 OK"
#   Issue 2: --store no value exits 2
grep -qE -- '--store.*(value|nothing after)|init --store.*exit' README.md && echo "B: --store no-value exit 2 OK"
#   ACCURACY: --search no-value must NOT be claimed as exit 2 (only --store). Inspect the error
#   contract manually and confirm it does not say "--search with no value exits 2".

# Edit E — reserved-tag note (Issue 7) is present (VERIFY, not added by this task):
grep -q 'Reserved tag names' README.md && echo "E: reserved-tag note present OK"
grep -q '`check`' README.md && grep -q '`init`' README.md && echo "E: both names OK"
# Expected: all checks print OK; the error contract does NOT overclaim --search.
```

### Level 3: Isolation / no-regression validation (the doc-sanity check)

```bash
cd /home/dustin/projects/skilldozer

# No Go regression (the contract's "re-build the binary, re-run go test ./..."):
go build ./... ; echo "build exit $?"    # 0
go vet  ./...  ; echo "vet exit $?"      # 0
go test ./...  ; echo "test exit $?"     # 0

# Isolation: only README.md changed by THIS subtask (besides pre-existing plan/ churn):
git status --short README.md             # expect " M README.md"
git diff --name-only | grep -v '^plan/'  # README.md should be the only non-plan/ path
# Expected: build/vet/test all exit 0; README.md modified; no Go/PRD/tasks/.gitignore file changed.

# Cross-check: the README's exit-2 claims match the landed code (the docs must agree with main.go):
grep -n 'tags cannot be combined' main.go    # main.go:739 (Issue 3 — tag+mode exit 2)
grep -n 'store requires a value'  main.go    # main.go:462 (Issue 2 — --store no-value exit 2)
# Expected: both present (READ-ONLY — the README now states the same rules in prose).
```

### Level 4: Behavioral / Domain-Specific Validation (manual read-through)

```bash
cd /home/dustin/projects/skilldozer

# Read the whole README as a user would; confirm the four user-facing claims hold and nothing
# contradicts the binary. Build the binary and spot-check the behaviors the README now asserts:
go build -o /tmp/skilldozer .

# Issue 1 — init stdout is EXACTLY one line (the store path):
iso=$(mktemp -d)
out=$(SKILLDOZER_SKILLS_DIR="" SKILLDOZER_CONFIG="$iso/cfg.yaml" /tmp/skilldozer init --store "$iso/store" </dev/null 2>/dev/null)
echo "$out" | wc -l   # 1  (== the store path). README claim: "exactly the configured store path ... one clean line".

# Issue 2 — init --store with no value exits 2 (config not written):
SKILLDOZER_SKILLS_DIR="" SKILLDOZER_CONFIG="$iso/cfg2.yaml" /tmp/skilldozer init --store </dev/null >/dev/null 2>&1; echo "exit=$?"  # 2

# Issue 3 — tag + --path exits 2:
SKILLDOZER_SKILLS_DIR="$iso/store" /tmp/skilldozer NONEXISTENTTAG --path >/dev/null 2>&1; echo "exit=$?"  # 2

# Issue 5 — tilde expands (~/x -> $HOME/x):
t2=$(mktemp -d); out=$(SKILLDOZER_SKILLS_DIR="" SKILLDOZER_CONFIG="$t2/c.yaml" HOME="$t2" /tmp/skilldozer init --store '~/sub' </dev/null 2>/dev/null)
echo "$out" ; test -d "$t2/sub" && echo "tilde expanded OK (dir at $t2/sub)"   # README claim: "~ expands to home"
rm -rf "$iso" "$t2" /tmp/skilldozer
# Expected: each behavior matches the README's prose (the docs and the binary agree).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `grep -c '§' README.md` == 0; `sed` render shows clean markdown at both seams
- [ ] Level 2 PASS — all required-content greps print OK; error contract does NOT overclaim `--search`
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0; `git status --short README.md` shows `M README.md`; no non-plan/ file other than README.md changed
- [ ] Level 4 PASS — manual read-through + binary spot-checks confirm the README's init-stdout / `--store`-no-value / tag+mode / tilde claims match the binary

### Feature Validation
- [ ] First run section states init stdout = ONLY the store path (one clean line for `$(...)`); status + check report → stderr
- [ ] First run section states `~`/`~/...` expands to `$HOME` in typed AND `--store`/positional paths
- [ ] Error contract states a tag + any of `--path`/`--list`/`--search`/`--all` exits 2
- [ ] Error contract states `--store` with no value exits 2 (and does NOT claim `--search` no-value exits 2)
- [ ] Reserved-tag note verified present in "Where skills live" (not edited/duplicated)

### Code Quality / Convention Validation
- [ ] Matches README voice: plain prose, inline `` `code` ``, bold-lead-in callouts, NO `§`-PRD citations
- [ ] Edits are minimal (contract: "keep edits minimal"); Issues 4 and 6 intentionally omitted (noted)
- [ ] Markdown structure intact (blank-line discipline; no broken fences; `## Shell completions` heading preserved)
- [ ] Seams located by anchor text (robust to the parallel T5.S1 line shift)

### Documentation & Deployment
- [ ] Mode B: this IS the changeset-level documentation sync (depends on all implementing subtasks)
- [ ] No `PRD.md` edit; no code edit; no `.gitignore` edit; no new files
- [ ] No new environment variables or configuration introduced

---

## Anti-Patterns to Avoid

- ❌ **Don't claim `--search` no-value exits 2.** Only `--store` got the Issue 2 exit-2 fix; `--search` no-value still exits 1 (usage). Say "`--store` expects a value", not "`--store`/`--search`". (GOTCHA #2.)
- ❌ **Don't imply the check report is on stdout.** After Issue 1, init stdout is ONE line (the store path); the report/status go to stderr. (GOTCHA #4.)
- ❌ **Don't add `§6.1`/`§8.2`/`§9` citations.** `grep -c '§' README.md` is 0; the README never cites PRD sections. Use plain words. (GOTCHA #1.)
- ❌ **Don't edit or duplicate the reserved-tag note.** It is P1.M2.T5.S1's landed deliverable. VERIFY it; do not restate it in Usage/First run. (GOTCHA #8.)
- ❌ **Don't touch "Where skills live", main.go, main_test.go, .gitignore, PRD.md, or tasks.json.** This task edits ONLY README.md's First run + Error contract zones. (GOTCHA #3/#6/#7.)
- ❌ **Don't locate seams by line number.** The file shifts as T5.S1 lands its note. Use the anchor text in research/verified_facts.md §2. (GOTCHA #3.)
- ❌ **Don't force prose for Issues 4 and 6.** Issue 4 is subsumed by the general exclusivity framing; Issue 6 is not user-facing. Note them as intentionally omitted. (GOTCHA #9.)
- ❌ **Don't skip the doc-sanity build/test.** Run `go build/vet/test ./...` to prove no code regression from doc-adjacent work (contract LOGIC). (GOTCHA #6.)

---

## Confidence Score

**9.5/10** — Two small prose edits at anchor-located seams (exact current text quoted in research §2),
with the behavior to document verified DIRECTLY in main.go for every relevant fix (Issues 1, 2, 3, 5)
and the reserved-tag note (Issue 7) confirmed already landed. The README voice rules are grep-confirmed
(`grep -c '§'` == 0; callout convention pinned by `**Error contract.**`). The one accuracy trap
(`--search` no-value is NOT exit 2 — only `--store`) is explicitly called out at every level. The
parallel-conflict risk (T5.S1 also edits README.md) is analyzed and mitigated (disjoint regions;
anchor-text seams; T5.S1 landed before this task executes). The 0.5 reservation is for the two
one-pass slips the PRP cannot fully mechanize away: (a) an accuracy slip (claiming `--search`
no-value exits 2, or implying the report is on stdout) — both caught by Level 2/4 checks; and (b) a
voice slip (a `§` citation or a non-callout format) — caught by Level 1's `grep -c '§'` and the
render check.
