name: "P1.M3.T1.S2 — README: document the `completion` subcommand as the easy install path (Mode B)"
description: |

---

## Goal

**Feature Goal**: Add PRD §14.6's `eval`/`source` one-liner idiom to the README's "Shell completions" section as the **RECOMMENDED** install path, placed **before** the existing §14.5 manual source/copy path (which stays). This is the **Mode B changeset-level documentation sync** — the final deliverable of delta 003 that summarizes the entire `completion` feature at the README level.

**Deliverable**: A README.md edit confined to the "Shell completions" section (lines 94-128). One new recommended-path subsection (the eval/source one-liners + the embed note + `--shell`/auto-detect note) inserted **before** the existing manual source/copy instructions, which are preserved verbatim. No other file changes. **This is prose editing, not code** — match the README's existing concise tone; do NOT duplicate the PRD §14.6 blockquote rationale.

**Success Definition**: `grep -q 'skilldozer completion' README.md` exits 0 (the contract gate); both canonical one-liners (`eval "$(skilldozer completion)"` for bash/zsh, `skilldozer completion --shell fish | source` for fish) are present and correct; the binary-embeds-scripts note and the `--shell`/`$SKILLDOZER_SHELL`/`$SHELL` auto-detect note are present; the existing manual source/copy instructions (four `cp`/`source` lines, the `autoload -U compinit && compinit` note, the `install.sh` note) are **unchanged and still present**; only README.md is modified.

## User Persona (if applicable)

**Target User**: A `skilldozer` user reading the README who wants to enable shell completions with the least friction — one line in their rc file, no clone, no copy.

**Use Case**: After `go install` (or any install), the user copies the documented one-liner into their shell rc file (`eval "$(skilldozer completion)"` for bash/zsh, `| source` for fish), restarts their shell, and tab-completion works.

**User Journey**: README → "Shell completions" → sees the recommended eval/source one-liner first → pastes it into rc file → done. The manual source/copy path remains below for clone users who want edits picked up without a rebuild.

**Pain Points Addressed**: Today the README documents only the manual source/copy path (requires cloning the repo or knowing the path to `completions/*`). The §14.6 eval idiom is the easier, clone-free path — and it was missing from the README.

## Why

- **Closes the Mode B documentation gap.** The `completion` subcommand is fully functional (P1.M2.T2.S2) and tab-completable in all three shells (P1.M3.T1.S1), but the README never mentions it. This task is the changeset-level doc sync that documents the feature end-to-end at the README surface.
- **Surfaces the easy install path.** `eval "$(skilldozer completion)"` is the recommended, lowest-friction way to enable completions — especially for `go install` users with no repo clone (the binary embeds the scripts, PRD §14.6 / §12.2 decision 9). Without this doc, users only see the heavier manual copy path.
- **Keeps both delivery paths documented (PRD §14.6 lockstep note).** The eval/embed path (recommended, clone-free, needs a rebuild after editing `completions/*`) and the manual source/copy path (needs a clone, picks up edits immediately) are both valid; the README must present both, with the easy one first.
- **No rationale duplication.** The README is concise by design (PRD §15 "mirror mcpeepants README's tone"). Do NOT paste the PRD §14.6 blockquote about "a child process cannot register completions" — a short clause is the right register.

## What

A single README.md edit in the "Shell completions" section (lines 94-128). Insert the recommended eval/source path **after** the section's intro paragraph (lines 96-98) and **before** the `**bash** (one of):` manual block (line 100). Preserve the entire manual source/copy path unchanged.

**New recommended-path content (the substance that must appear):**

- **bash / zsh one-liner** (verbatim from PRD §14.6):
  ```bash
  eval "$(skilldozer completion)"
  ```
  with the note that it goes in `~/.bashrc` or `~/.zshrc`.
- **fish one-liner** (verbatim from PRD §14.6):
  ```bash
  skilldozer completion --shell fish | source
  ```
  with the note that it goes in `~/.config/fish/config.fish`.
- **Embed note** (one line): the binary embeds the completion scripts, so this works for `go install` users with no clone (consistent with §12.2 / decision 9).
- **`--shell` / auto-detect note** (one short clause): `--shell <bash|zsh|fish>` makes the eval deterministic; otherwise `skilldozer completion` auto-detects from `$SKILLDOZER_SHELL`, then `$SHELL`.
- A framing that marks this path as the **recommended** / easy one, and the manual path below as the **alternative** (clone users; picks up edits without a rebuild).

**Preserved unchanged (the existing §14.5 manual source/copy path):**

- The `**bash** (one of):` block (`source` + two `cp` lines).
- The `**zsh** (one of):` block (two `cp` lines) + the `autoload -U compinit && compinit` note.
- The `**fish**:` block (one `cp` line).
- The closing note: "`install.sh` does not install completions automatically; copy the file you want as shown above."

### Success Criteria

- [ ] `grep -q 'skilldozer completion' README.md` exits 0 (the contract gate).
- [ ] `eval "$(skilldozer completion)"` appears verbatim (the bash/zsh one-liner).
- [ ] `skilldozer completion --shell fish | source` appears verbatim (the fish one-liner).
- [ ] The recommended eval/source path is placed BEFORE the manual source/copy instructions.
- [ ] The binary-embeds-scripts note (works for `go install` users with no clone) is present.
- [ ] The `--shell` deterministic / `$SKILLDOZER_SHELL`/`$SHELL` auto-detect note is present.
- [ ] The existing manual source/copy instructions are all still present and unchanged (the four `cp`/`source` lines, `autoload -U compinit && compinit`, the install.sh note).
- [ ] Only README.md is modified; `git status --short` shows exactly `README.md`.

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact current README section (lines 94-128) was read in full and reproduced in research §1; every line that must be preserved is quoted. The `completion` subcommand's behavior — one-liners, shell-detection precedence, exit codes, embed model — was verified directly in main.go (USAGE row 107, USAGE EXAMPLES 97, detectShell 1239-1252, runCompletion 1275-1293, embed 46-61) and recorded in research §2. The two canonical one-liners are fixed by PRD §14.6 (research §3, verbatim). The contract requirements (research §5), scope boundaries (research §6), and validation greps (research §7) are pinned. The README's house tone is documented with in-section examples (research §8). An implementer who has never seen this repo can complete it in one pass: insert one recommended-path block (fixed one-liners + two short notes) before the existing manual block, preserve everything else, run the grep gates.

### Documentation & References

```yaml
# MUST READ — the verified facts (exact README section, subcommand behavior, contract, scope, validation)
- file: plan/003_3ace946c2a5c/P1M3T1S2/research/verified_facts.md
  why: "§1 = the EXACT current README 'Shell completions' section (lines 94-128), every line to preserve.
        §2 = the completion subcommand behavior (one-liners, --shell, detection precedence, exit codes,
        embed model) verified in main.go. §3 = the two canonical one-liners VERBATIM from PRD §14.6.
        §5 = the contract's OUTPUT requirements. §6 = scope (README.md ONLY). §7 = validation greps.
        §8 = README house tone."
  critical: "§3 — the one-liners are FIXED (do not rephrase; they must match PRD §14.6 exactly so `eval`
             works). §5 — do NOT duplicate the §14.6 rationale verbatim; match README tone. §6 — only
             README.md is modified."

# MUST EDIT — the single deliverable
- file: README.md
  why: "EDIT the 'Shell completions' section ONLY (lines 94-128). Insert the recommended eval/source path
        after the intro paragraph (line 98) and before the `**bash** (one of):` block (line 100). Preserve
        the entire manual source/copy path verbatim."
  pattern: "The README's house style: concise imperative second-person prose + fenced ```bash blocks.
            See the existing section's 'ships dynamic completions …' / 'copy the file you want as shown
            above' for register. A short clause (e.g. 'the binary embeds the completion scripts, so this
            needs no clone') is the right density — NOT a PRD blockquote."
  gotcha: "Do NOT remove or alter the existing manual source/copy block, the `autoload -U compinit &&
           compinit` note, or the install.sh note — they are preserved verbatim. Do NOT paste the PRD §14.6
           'child process cannot register completions' rationale. Do NOT change any other README section."

# READ-ONLY — confirms the one-liners + behavior (the source of truth for the documented text)
- file: main.go
  why: "USAGE row (107): 'completion [--shell <name>]   Emit the shell completion script for eval (§14.6)'.
        USAGE EXAMPLES (97): 'eval \"$(skilldozer completion)\"     # load completions into your shell'.
        detectShell (1239-1252): --shell → $SKILLDOZER_SHELL → basename($SHELL). runCompletion (1275-1293):
        exit 0/1/2; nothing on stdout on 1/2. embed (46-61): //go:embed's completions/*, no new dep, works
        for go install with no clone. READ-ONLY — do NOT edit main.go."
  section: "USAGE block (78-110), config.completion/completionShell (166-167), detectShell (1239-1252),
            runCompletion (1275-1293), embed vars (54-61)."

# READ-ONLY — the PRD (the authoritative contract for the one-liners + the eval rationale)
- file: PRD.md
  why: "§14.6 (h3.19): the canonical one-liners (research §3 reproduces them) + the shell-detection order +
        the embed + lockstep note. §14.5 (h3.18): the manual source/copy path the README already has (keep).
        §12.2 / decision 9: the go-install self-sufficient-binary stance the embed note references. §15
        (h2.14): README tone = concise, mirror mcpeepants."
  section: "h3.19 (§14.6), h3.18 (§14.5), h3.12 (§12.2), h2.14 (§15)."
  gotcha: "READ-ONLY. Do NOT duplicate §14.6's rationale verbatim in the README (contract §5.6 / research
           §5)."

# READ-ONLY — the parallel sibling (disjoint files; this task builds on its lockstep but does not edit them)
- file: plan/003_3ace946c2a5c/P1M3T1S1/PRP.md
  why: "P1.M3.T1.S1 edits ONLY completions/* (adds `completion` as a tab-completable subcommand). This task
        edits ONLY README.md. DISJOINT — no merge conflict. The README's recommended `eval` path documents
        the subcommand that the lockstep makes tab-completable; the two tasks are complementary, not
        overlapping."

# READ-ONLY — the previous feature PRP (the emit path the README documents)
- file: plan/003_3ace946c2a5c/P1M2.T2.S2/PRP.md
  why: "Defines run()→runCompletion→completionScript: `skilldozer completion --shell bash` emits the
        //go:embed'd bytes to stdout. The README documents exactly this emit path. READ-ONLY context."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && grep -n 'skilldozer completion' README.md ; echo "exit $?"
# exit 1  → NOT PRESENT (this is what the task adds)

$ sed -n '94,130p' README.md   # the "Shell completions" section
94:## Shell completions
96-98: intro paragraph (ships dynamic completions; tag completion via `skilldozer --relative --all`)
100-106: **bash** (one of): source / cp / cp          # KEEP
108-119: **zsh** (one of): cp / cp + autoload compinit # KEEP
121-125: **fish**: cp                                  # KEEP
127-128: `install.sh` does not install completions …   # KEEP
130: ## Usage   # next section — UNCHANGED

# main.go / completions/* / internal/* / PRD.md — UNCHANGED.
```

### Desired Codebase tree with files to be changed

```bash
README.md   # "Shell completions" section: +recommended eval/source block before the manual block
# every other file UNCHANGED (no Go, no completions/*, no PRD, no .gitignore).
```

| File | Change | Why |
|---|---|---|
| `README.md` | Insert the §14.6 eval/source recommended path (one-liners + embed note + --shell/auto-detect note) before the existing manual source/copy block; keep the manual block. | Mode B doc sync: document the easy install path for the `completion` subcommand. |

### Known Gotchas of our codebase & Library Quirks

```markdown
# GOTCHA #1 (CRITICAL — one-liners are FIXED) — The two one-liners must appear VERBATIM as in PRD §14.6:
#   eval "$(skilldozer completion)"                      # bash/zsh
#   skilldozer completion --shell fish | source          # fish
# Do NOT "improve" them (e.g. adding --shell to the bash/zsh line, or quoting tweaks). They must be
# copy-pasteable and match the binary's USAGE EXAMPLES (main.go:97) and PRD §14.6 exactly. (research §3.)

# GOTCHA #2 (CRITICAL — do NOT duplicate the rationale) — The PRD §14.6 blockquote ("a child process
# cannot register completions in the parent shell …") is for the PRD, NOT the README. The README is
# concise (PRD §15). A short clause is enough — e.g. "the binary embeds the completion scripts, so this
# works with no clone." Pasting the blockquote violates the contract (research §5.6) and the README tone.

# GOTCHA #3 (do NOT remove the manual path) — The existing §14.5 manual source/copy instructions STAY,
# verbatim: the four cp/source lines, `autoload -U compinit && compinit`, and the install.sh note. The
# eval path is RECOMMENDED and goes FIRST; the manual path is the ALTERNATIVE and stays SECOND. Deleting
# or rewriting the manual block is a scope violation. (research §4, §5.5.)

# GOTCHA #4 (placement — BEFORE the manual block) — The new recommended eval/source block is inserted
# AFTER the section's intro paragraph (line 98) and BEFORE the `**bash** (one of):` manual block (line
# 100). The manual block keeps its `**bash** (one of):` / `**zsh** (one of):` / `**fish**:` headers.

# GOTCHA #5 (the embed + --shell + auto-detect notes are REQUIRED) — The contract requires THREE notes:
#   (a) binary embeds the scripts → works for `go install` users with no clone (§12.2 decision 9);
#   (b) `--shell` makes the eval deterministic;
#   (c) else auto-detect from $SKILLDOZER_SHELL, then $SHELL.
# These can be 1-2 short sentences/clauses each — concise, README-tone. Omitting (a)/(b)/(c) fails the
# contract. (research §5.3, §5.4.)

# GOTCHA #6 (scope — README.md ONLY) — Only README.md is modified. Do NOT touch main.go, completions/*,
# PRD.md, tasks.json, prd_snapshot.md, or .gitignore. This is the Mode B doc sync; the feature + lockstep
# are done by sibling tasks. (research §6.)

# GOTCHA #7 (markdown hygiene) — Keep code fences balanced (```bash … ```). The two one-liners can share
# one fenced block or be split by shell — match the README's existing fenced-block style (each shell gets
# its own fenced block in the manual path). Do NOT break the `## Shell completions` / `## Usage` headers.

# GOTCHA #8 (no stale claim) — Do NOT claim `completion` "installs" completions or writes files. It
# EMITS the script to stdout for the shell to eval (writes nothing). If you mention what it does, say
# "emits"/"prints", not "installs". (research §3 rationale — but do NOT paste the rationale.)
```

---

## Implementation Blueprint

### Data models and structure

**None.** This is a prose/markdown edit to a single section of README.md. No code, no types, no config.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT README.md — insert the recommended eval/source path (the substance)
  - FILE: README.md
  - LOCATE: the "Shell completions" section (## Shell completions, line 94). The intro paragraph is
    lines 96-98 ("`skilldozer` ships dynamic completions … so it never goes stale as you add skills.").
    The manual block starts at line 100 ("**bash** (one of):").
  - INSERT: a recommended-path block AFTER line 98 (the blank line following the intro paragraph) and
    BEFORE line 100 ("**bash** (one of):"). The block contains:
      (a) a one-line framing that this is the recommended / easiest path (e.g. "The easiest way to load
          completions is the `completion` subcommand, which prints the script for your shell to eval:").
      (b) a fenced ```bash block with the bash/zsh one-liner VERBATIM:
              eval "$(skilldozer completion)"
          and a note it goes in ~/.bashrc or ~/.zshrc.
      (c) a fenced ```bash block (or a second line) with the fish one-liner VERBATIM:
              skilldozer completion --shell fish | source
          and a note it goes in ~/.config/fish/config.fish.
      (d) ONE-LINE EMBED NOTE (GOTCHA #5a): the binary embeds the completion scripts, so this works for
          `go install` users with no clone (§12.2 decision 9).
      (e) ONE-LINE --shell/AUTO-DETECT NOTE (GOTCHA #5b/c): `--shell <bash|zsh|fish>` makes the eval
          deterministic; otherwise `skilldozer completion` auto-detects from $SKILLDOZER_SHELL, then $SHELL.
      (f) a one-line framing that the manual source/copy path below is the alternative (clone users; it
          picks up edits to completions/* without a rebuild) — optional but consistent with PRD §14.6.
  - TONE: match the README's existing concise register (see the section's own prose). Short clauses, not
    PRD blockquotes. (GOTCHA #2, research §8.)
  - Do NOT remove or alter the manual source/copy block (lines 100-128). (GOTCHA #3.)

Task 2: VERIFY all gates (the contract's OUTPUT)
  - grep -q 'skilldozer completion' README.md                                  # contract gate — exit 0
  - grep -q 'eval "$(skilldozer completion)"' README.md                         # bash/zsh one-liner — exit 0
  - grep -q 'skilldozer completion --shell fish | source' README.md             # fish one-liner — exit 0
  - manual-path regression: the existing lines are still present (see Validation Loop Level 2)
  - scope: git status --short shows exactly README.md
  - see Validation Loop for the full gate set.
```

### Implementation Patterns & Key Details

```markdown
# The deliverable is ONE inserted block in the "Shell completions" section. Structure (anchored on the
# existing content):
#
#   ## Shell completions                                  # line 94 — UNCHANGED
#
#   `skilldozer` ships dynamic completions for bash, zsh, and fish. Tag completion is   # lines 96-98 — UNCHANGED
#   not a static list: the shell calls `skilldozer --relative --all` at completion time,
#   so it never goes stale as you add skills.
#
#   <NEW recommended eval/source block — Tasks 1(a)..(f)>                                # INSERT HERE
#
#   **bash** (one of):                                                                   # line 100 — UNCHANGED
#   ... (manual path preserved verbatim) ...
#   `install.sh` does not install completions automatically; copy the file you           # lines 127-128 — UNCHANGED
#   want as shown above.
#
#   ## Usage                                                                             # line 130 — UNCHANGED
#
# The two one-liners (FIXED — copy from PRD §14.6 / research §3):
#   bash/zsh : eval "$(skilldozer completion)"                       # ~/.bashrc / ~/.zshrc
#   fish     : skilldozer completion --shell fish | source           # ~/.config/fish/config.fish

# A concrete tone-matched example of the inserted block (the implementer may refine wording, but MUST
# include the fixed one-liners + notes a/b/c):
#
#   The easiest way to load completions is the `completion` subcommand, which prints the
#   script for your shell to eval. The binary embeds the completion scripts, so this works
#   for `go install` users with no clone.
#
#   **bash / zsh** — add to `~/.bashrc` or `~/.zshrc`:
#
#   ```bash
#   eval "$(skilldozer completion)"
#   ```
#
#   **fish** — add to `~/.config/fish/config.fish`:
#
#   ```bash
#   skilldozer completion --shell fish | source
#   ```
#
#   `--shell <bash|zsh|fish>` makes the eval deterministic; otherwise `skilldozer completion`
#   auto-detects from `$SKILLDOZER_SHELL`, then `$SHELL`.
#
#   Prefer to copy the file instead? The manual path below picks up edits to `completions/*`
#   without a rebuild.
```

Notes easy to get wrong:
- Re-wording the one-liners (GOTCHA #1) — they are fixed; copy them verbatim.
- Pasting the PRD §14.6 rationale (GOTCHA #2) — forbidden; use a short clause.
- Deleting the manual path (GOTCHA #3) — it stays; the eval path is just added before it.
- Forgetting one of the three required notes (GOTCHA #5) — embed, --shell-deterministic, auto-detect.
- Editing anything beyond README.md (GOTCHA #6) — scope is README.md only.

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **eval/source path is RECOMMENDED and goes FIRST; manual path stays SECOND.** The contract (research §5.1/§5.5) explicitly mandates this ordering and preservation of the manual path. The eval idiom is the lower-friction, clone-free path (the binary embeds the scripts); the manual path is the clone-user alternative that picks up `completions/*` edits without a rebuild (PRD §14.6 lockstep note). Both are documented; neither is removed.
2. **The one-liners are verbatim from PRD §14.6.** `eval "$(skilldozer completion)"` (bash/zsh) and `skilldozer completion --shell fish | source` (fish) must be copy-pasteable and byte-match the binary's USAGE EXAMPLES (main.go:97) and PRD §14.6. Re-wording risks breaking the `eval` (e.g. a quoting tweak). (research §3, GOTCHA #1.)
3. **The rationale is NOT duplicated.** The README's house tone is concise (PRD §15); the PRD §14.6 blockquote about "a child process cannot register completions" belongs in the PRD, not the README. A short clause ("the binary embeds the completion scripts, so this works with no clone") is the right density. (research §8, GOTCHA #2.)
4. **The three required notes (embed, --shell-deterministic, auto-detect) are mandatory.** The contract (research §5.3/§5.4) requires all three; omitting any fails the contract. They are short (1-2 clauses each). (GOTCHA #5.)
5. **Validation = grep gates + manual-path regression + scope check.** No code changes, so no build/test is semantically required; `go test ./...` is run only to prove no accidental code edits. The grep gates (`grep -q 'skilldozer completion'`, the two one-liners) are the contract's verification. (research §7.)
6. **README.md is the ONLY file touched.** This is Mode B doc sync; the feature (P1.M2.T2.S2) and lockstep (P1.M3.T1.S1) are done by disjoint sibling tasks. main.go/completions/* are read-only context; PRD.md/tasks.json/prd_snapshot.md are read-only. (research §6, GOTCHA #6.)

### Integration Points

```yaml
README.md (the deliverable):
  - section: "## Shell completions" (lines 94-128)
  - effect: "Adds the recommended eval/source one-liner path (PRD §14.6) before the existing manual
            source/copy path (PRD §14.5), which is preserved. The three required notes (embed,
            --shell-deterministic, auto-detect) are present."

DOCUMENTED FEATURE (consumed, not modified here):
  - `skilldozer completion [--shell <name>]` (P1.M2.T2.S2): emits the embedded script to stdout for eval.
  - The lockstep (P1.M3.T1.S1): makes `completion` tab-completable in all three shells.
  - The README documents the emit path that those tasks produce.

CODE: NONE. main.go / completions/* / internal/* UNCHANGED.
PRD.md / tasks.json / prd_snapshot.md: READ-ONLY. .gitignore: UNCHANGED.

PARALLEL SIBLING (no conflict):
  - P1.M3.T1.S1 edits ONLY completions/*. This task edits ONLY README.md. DISJOINT — no merge conflict.

NO DATABASE / NO ROUTES / NO CONFIG-FORMAT CHANGE / NO GO CODE / NO NEW FILES.
```

---

## Validation Loop

### Level 1: Markdown sanity (immediate, after the edit)

```bash
cd /home/dustin/projects/skilldozer

# Code fences balanced in the edited section (count ``` in the whole file — should be even):
test $(( $(grep -c '```' README.md) % 2 )) -eq 0 && echo "fences balanced OK" || echo "FAIL: unbalanced fences"

# The section headers are intact:
grep -q '^## Shell completions$' README.md && echo "header OK"
grep -q '^## Usage$' README.md && echo "next header OK"
# Expected: fences balanced; both headers present.
```

### Level 2: Contract gate + required-content + manual-path regression (the contract's OUTPUT)

```bash
cd /home/dustin/projects/skilldozer

# (a) CONTRACT GATE — `skilldozer completion` is now documented:
grep -q 'skilldozer completion' README.md && echo "contract gate OK" || echo "FAIL: no 'skilldozer completion'"

# (b) The two FIXED one-liners are present verbatim:
grep -qF 'eval "$(skilldozer completion)"' README.md && echo "bash/zsh one-liner OK" || echo "FAIL: bash/zsh one-liner"
grep -qF 'skilldozer completion --shell fish | source' README.md && echo "fish one-liner OK" || echo "FAIL: fish one-liner"

# (c) The three required NOTES are present (embed / --shell / auto-detect):
grep -qi 'embed' README.md && echo "embed note OK" || echo "FAIL: no embed note"
grep -qi -- '--shell' README.md && echo "--shell note OK" || echo "FAIL: no --shell note"
grep -q 'SKILLDOZER_SHELL' README.md && echo "auto-detect note OK" || echo "FAIL: no SKILLDOZER_SHELL note"

# (d) the recommended path comes BEFORE the manual path (eval line has a lower line number than the
#     first manual 'cp'/'source' instruction):
EVAL_LINE=$(grep -nF 'eval "$(skilldozer completion)"' README.md | head -1 | cut -d: -f1)
MANUAL_LINE=$(grep -nF 'source /path/to/skilldozer/completions/skilldozer.bash' README.md | head -1 | cut -d: -f1)
[ -n "$EVAL_LINE" ] && [ -n "$MANUAL_LINE" ] && [ "$EVAL_LINE" -lt "$MANUAL_LINE" ] \
  && echo "ordering OK (eval before manual)" || echo "FAIL: ordering"

# (e) MANUAL-PATH REGRESSION — the existing instructions are still present and unchanged:
grep -qF 'source /path/to/skilldozer/completions/skilldozer.bash' README.md && echo "bash source OK"
grep -qF 'cp completions/skilldozer.bash ~/.local/share/bash-completion/completions/skilldozer' README.md && echo "bash cp1 OK"
grep -qF 'cp completions/skilldozer.bash /etc/bash_completion.d/skilldozer' README.md && echo "bash cp2 OK"
grep -qF 'cp completions/_skilldozer ~/.zsh/completions/_skilldozer' README.md && echo "zsh cp1 OK"
grep -qF 'cp completions/_skilldozer /usr/local/share/zsh/site-functions/_skilldozer' README.md && echo "zsh cp2 OK"
grep -qF 'autoload -U compinit && compinit' README.md && echo "zsh compinit OK"
grep -qF 'cp completions/skilldozer.fish ~/.config/fish/completions/skilldozer.fish' README.md && echo "fish cp OK"
grep -qF 'install.sh` does not install completions automatically' README.md && echo "install.sh note OK"
# Expected: every check prints OK.
```

### Level 3: Scope discipline (the contract's "ONLY README.md")

```bash
cd /home/dustin/projects/skilldozer

# Exactly one file changed, and it is README.md:
git status --short
test "$(git status --short --untracked-files=no | wc -l)" -eq 1 && \
  git diff --name-only --untracked-files=no | grep -qx README.md && echo "scope OK" || echo "FAIL: scope"

# No code files touched:
git diff --name-only --untracked-files=no | grep -qE '\.go$|completions/' && echo "FAIL: code touched" || echo "no code touched OK"

# (Sanity — nothing was accidentally broken in Go land; no code changed, so this must be green:)
go build ./... && echo "go build OK"
go test ./... >/dev/null 2>&1 && echo "go test OK" || echo "go test FAIL (unexpected — recheck no .go edits)"
# Expected: scope OK; no code touched OK; go build/test OK.
```

### Level 4: Render check (optional, if a markdown viewer is available)

```bash
cd /home/dustin/projects/skilldozer

# Render-check the edited section (optional — confirms the one-liners display as code, not prose):
#   - If glow / mdcat / bat is installed:
#       glow README.md | sed -n '/Shell completions/,/## Usage/p'
#   - Or eyeball the section in your editor / GitHub preview.
# Expected: the two one-liners render inside fenced code blocks; the manual path renders unchanged.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — code fences balanced; `## Shell completions` and `## Usage` headers intact
- [ ] Level 2 PASS — contract gate (`grep -q 'skilldozer completion'`); both one-liners verbatim; all three notes (embed / --shell / auto-detect) present; eval path ordered before the manual path; every manual-path regression grep prints OK
- [ ] Level 3 PASS — exactly `README.md` changed; no `.go`/`completions/` file touched; `go build`/`go test` green (proves no accidental code edit)

### Feature Validation
- [ ] The §14.6 `eval`/`source` idiom is documented as the RECOMMENDED path
- [ ] The bash/zsh one-liner `eval "$(skilldozer completion)"` is present verbatim
- [ ] The fish one-liner `skilldozer completion --shell fish | source` is present verbatim
- [ ] The binary-embeds-scripts note (works for `go install` users with no clone) is present
- [ ] The `--shell` deterministic + `$SKILLDOZER_SHELL`/`$SHELL` auto-detect note is present
- [ ] The recommended path is placed BEFORE the manual source/copy instructions

### Code Quality / Convention Validation
- [ ] Matches the README's existing concise tone (no PRD blockquote pasted)
- [ ] The existing manual source/copy instructions are unchanged (all `cp`/`source` lines, `autoload -U compinit && compinit`, the install.sh note)
- [ ] No stale claims ("installs" → uses "emits"/"prints"; `completion` writes nothing)
- [ ] Markdown is well-formed (balanced fences, intact headers)

### Scope Discipline
- [ ] Only README.md is modified; `git status --short` shows exactly `README.md`
- [ ] main.go / completions/* / internal/* UNCHANGED (sibling/read-only scopes)
- [ ] PRD.md / tasks.json / prd_snapshot.md / .gitignore NOT touched (read-only)

---

## Anti-Patterns to Avoid

- ❌ **Don't re-word the one-liners.** They are fixed by PRD §14.6 and must match the binary's USAGE EXAMPLES (main.go:97). Copy `eval "$(skilldozer completion)"` and `skilldozer completion --shell fish | source` verbatim. (GOTCHA #1.)
- ❌ **Don't paste the PRD §14.6 rationale.** The README is concise (PRD §15); a short clause is enough. The "child process cannot register completions" blockquote belongs in the PRD, not the README. (GOTCHA #2.)
- ❌ **Don't remove or rewrite the manual source/copy path.** It stays verbatim — the eval path is added BEFORE it, not in place of it. Both paths are documented. (GOTCHA #3.)
- ❌ **Don't omit the required notes.** The contract requires the embed note (clone-free), the `--shell` deterministic note, and the `$SKILLDOZER_SHELL`/`$SHELL` auto-detect note. (GOTCHA #5.)
- ❌ **Don't edit anything besides README.md.** This is Mode B doc sync; the feature + lockstep are disjoint sibling tasks. main.go/completions/* are read-only context; PRD.md/tasks.json/prd_snapshot.md are read-only. (GOTCHA #6.)
- ❌ **Don't claim `completion` "installs" completions or writes files.** It EMITS the script to stdout for the shell to eval; it writes nothing and edits no rc files. (GOTCHA #8.)
- ❌ **Don't break markdown.** Keep code fences balanced and section headers intact; match the README's fenced-block style. (GOTCHA #7.)

---

## Confidence Score

**9.5/10** — one-pass implementation success likelihood. The task is a single-section README prose edit with the substance fully pinned: the exact current section (lines 94-128) is reproduced in research §1; the two one-liners are fixed verbatim from PRD §14.6 (and byte-checked against the binary's USAGE EXAMPLES, main.go:97); the `completion` subcommand's behavior (one-liners, shell-detection precedence `--shell` → `$SKILLDOZER_SHELL` → `basename($SHELL)`, exit codes, embed model) is verified directly in main.go (research §2); the three required notes (embed / --shell / auto-detect) and the contract's ordering/preservation requirements are explicit (research §5). The README's house tone is documented with in-section examples (research §8). The file set is fully disjoint from the parallel P1.M3.T1.S1 (completions/* only). The 0.5 reservation is for the two slips the PRP cannot fully mechanize away — paraphrasing a one-liner (caught by the Level 2 `grep -qF` verbatim checks) and accidentally editing the manual path (caught by the Level 2 manual-path regression greps) — both of which the validation gates catch immediately.
