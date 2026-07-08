name: "P1.M4.T2.S1 — Update README.md: init UX, config model, 5-rule priority, drop obsolete go-install caveat"
description: |
  Mode B changeset-level documentation sync. The README is the overview doc and lags the
  §8 config model that P1.M1/M2 implemented and that P1.M4.T1.S1 just proved green under §13.
  This PRP edits ONE file (README.md) so its Install / go-install / "finds the store" /
  Constraints sections match the current PRD §8.1, §8.2, §8.3, §12.2, §15.7, §15.8, §17. No Go
  code changes; no other docs. It is the final deliverable of the phase.

---

## Goal

**Feature Goal**: Bring `README.md` into compliance with the current PRD §8/§12.2/§15. Concretely: (a) document `skilldozer init` as the documented first command in §15.3 Install; (b) remove the obsolete `go install` caveat (the `export SKILLDOZER_SKILLS_DIR` block) per §12.2 and state go install is first-class; (c) replace the 3-rule "How `skilldozer` finds the store" list (§15.7) with the 5-rule §8.3 ladder, document the config file at `$XDG_CONFIG_HOME/skilldozer/config.yaml` + `SKILLDOZER_CONFIG` override, and list all four `--path` labels; (d) reword the §15.8 Constraints "Manifest-free" bullet to "no catalog index (disk-discovered); a settings config file is fine". All already-correct flag docs (`--path`/`--search`/`--list`/`--all`/`check`/`-f`/`--relative`/`--no-color`) and the existing README voice must be preserved.

**Deliverable**: A single edited `README.md` at the repo root whose Install section documents `init`, whose go-install subsection carries no caveat, whose "How `skilldozer` finds the store" section enumerates 5 rules + 4 labels + the config-file path, and whose Constraints section surfaces the catalog-vs-settings distinction. Verified by four greps (see Validation).

**Success Definition**:
- `grep -n 'skilldozer init' README.md` returns hits (init documented).
- `grep -nE 'config\.yaml|SKILLDOZER_CONFIG' README.md` returns hits (config model documented).
- `grep -n 'config file' README.md` returns the new `--path` label / rules text.
- `grep -n 'export SKILLDOZER_SKILLS_DIR' README.md` returns ZERO hits (obsolete caveat gone).
- The previously-correct flag inventory (Usage section lines ~75-120) is byte-unchanged.
- README still passes a manual read for tone/structure matching §15 (mirror the mcpeepants README's terse, example-first voice).

## User Persona (if applicable)

**Target User**: A new user installing skilldozer for the first time — especially a `go install` user who, under the old README, was told to clone the repo and `export SKILLDOZER_SKILLS_DIR`, and now should simply run `skilldozer init`.

**Use Case**: Install via `go install …@latest`, then run `skilldozer init` once to create the store and write the config — no clone, no env var.

**User Journey**: README → Install (B. go install) → "first run: `skilldozer init`" → `skilldozer init` prompts for the store dir, writes `$XDG_CONFIG_HOME/skilldozer/config.yaml` → `skilldozer example` resolves → `pi --skill "$(skilldozer example)"` works.

**Pain Points Addressed**: The current README actively misleads go-install users (tells them to set an env var the binary no longer needs), omits `init` entirely, documents a 3-rule discovery ladder that no longer matches the binary, and never acknowledges the now-permitted settings config file (so users think a config file violates the manifest-free rule).

## Why

- **The README contradicts the shipped binary.** P1.M1/M2 landed `internal/config`, the `init` subcommand, the 5-rule `Find()` precedence, the `config file` `Source.String()` label, and the exact `ErrNotFound` message `skilldozer is not configured; run \`skilldozer init\``. P1.M4.T1.S1 just proved all 15 §13 gates green against that binary. The README still describes the pre-§8 world (3 rules, env-var-only, no init). Drift audit `architecture/docs_and_assets_drift.md` §2 (items 2a-2d) and `architecture/code_prd_delta.md` G19/G20/G21 enumerate every gap.
- **go install is first-class now.** PRD §12.2 (h3.12) explicitly says: "`go install …@latest` is a first-class install path … on first use the user runs `skilldozer init` … No clone required, no `SKILLDOZER_SKILLS_DIR` needed for normal use. The earlier caveat … is obsolete under the config model and is removed." The README still presents that exact obsolete caveat as the user-facing fix (README.md:42-53).
- **Final deliverable of the phase.** Per the work-item DOCS note ([Mode B]), this subtask IS the changeset-level documentation sync; it depends on every implementing subtask by design and no further doc subtask follows.
- **Scope boundary:** README only. No source code. No other docs (install.sh / LICENSE / go.mod / .gitignore / completions / SKILL.md are all already compliant — see drift audit items 3,5,6,7 and the P1.M3 subtasks). Do NOT touch the §13 acceptance transcript owned by P1.M4.T1.S1.

## What

Four targeted edits to `README.md`. The README's existing section order, headings, and voice are preserved — only the content inside four sections changes.

### Edits (by README section)

**(1) §15.3 Install — add the `init` first-run note.** Under the existing `## Install` heading (README.md:19), after the three install paths (A/B/C), add a short "First run" callout: `First run: skilldozer init` (prompts for the store dir, writes the config). Also document the non-interactive forms `skilldozer init <dir>` and `skilldozer init --store <dir>` for scripts/CI. This is the §15.3 item "First run: `skilldozer init` (prompts for the store dir, writes the config)."

**(2) §12.2 go install — remove the obsolete caveat, state first-class.** Replace README.md:42-53 (the entire `> **`go install` caveat.** … export SKILLDOZER_SKILLS_DIR … Prefer ./install.sh …` blockquote) with a short first-class statement mirroring PRD §12.2 (h3.12): `go install` lands the binary in `$(go env GOPATH)/bin`; on first use run `skilldozer init` (§8.2), which creates the store and writes the config. No clone required, no `SKILLDOZER_SKILLS_DIR` needed for normal use. Keep the `go install github.com/dabstractor/skilldozer@latest` code fence (README.md:38-40).

**(3) §15.7 "How `skilldozer` finds the store" — 3 rules → 5 rules + config file + 4 labels.** Rewrite README.md:234-251 to enumerate the §8.3 (h3.10) ladder:
1. `SKILLDOZER_SKILLS_DIR` env var — override; if set and an existing dir, use it. Lets CI/tests/temp redirects win without editing the config.
2. Config file `store` (§8.1) — the primary, set by `skilldozer init`. Config lives at `$XDG_CONFIG_HOME/skilldozer/config.yaml` (→ `~/.config/skilldozer/config.yaml`); override the file path with `SKILLDOZER_CONFIG=<file>` (useful for tests / multiple profiles). Minimal valid file: `store: /home/you/skills`. A missing/unreadable config is "not yet configured" and falls through to rules 3-5, never a hard error.
3. Sibling of the running binary (symlink-aware: `os.Executable()` + `filepath.EvalSymlinks()`) — still lets a clone-and-build dev workflow work with zero config.
4. Walk up from `cwd` — for `go run` / dev.
5. None ⇒ unconfigured: stderr one-line fix (`run \`skilldozer init\``), exit `1`.

And rewrite the `--path` paragraph to list all FOUR labels: `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`. Keep the existing "a bad `SKILLDOZER_SKILLS_DIR` value is silently ignored and falls through — `--path` is the only way to tell which directory actually won" guidance (it is still accurate and load-bearing). Reword the stale "(see Install)" env-var note on old rule 1 (README.md:239) since go-install users no longer set the env var.

**(4) §15.8 Constraints — reword Manifest-free to catalog-vs-settings.** Edit README.md:255-258 (the `**Manifest-free.**` bullet + the opening line 255 `manifest-free path printer`) so the section surfaces PRD §15.8 / §17 (h2.16): "no catalog index (disk-discovered); a settings config file is fine." Concretely the Manifest-free bullet becomes something like: **No catalog index.** No `skills.json`, no manifest enumerating skills. The catalog is always walked from disk. A *settings* config file (store location, etc. — see above) is expected and fine; the rule is only that catalog data on disk is never duplicated into a sidecar. Keep the other Constraints bullets (Never auto-discovered by pi / Loaded only via `--skill` / only ever prints paths / Zero runtime dependencies) unchanged.

### Success Criteria

- [ ] `grep -n 'skilldozer init' README.md` → ≥1 hit.
- [ ] `grep -nE 'config\.yaml|SKILLDOZER_CONFIG' README.md` → ≥1 hit each (both the path and the override).
- [ ] `grep -n 'config file' README.md` → ≥1 hit (the new `--path` label / rules text).
- [ ] `grep -n 'export SKILLDOZER_SKILLS_DIR' README.md` → ZERO hits (caveat removed).
- [ ] `grep -nE '\`go install\` caveat' README.md` → ZERO hits.
- [ ] The Usage section flag inventory (README.md ~75-120) is byte-identical to the pre-edit version (no flag docs touched).
- [ ] The "Where skills live" and "Adding a skill" sections are unchanged (already §15.5/§15.6 compliant per drift audit §2e/§2f).
- [ ] Manual read: the README still reads in the terse, example-first mcpeepants voice (§15 "Mirror the mcpeepants README's tone and structure").

## All Needed Context

### Context Completeness Check

**Pass.** An implementer who has never seen this repo gets, below: (a) the exact current README text of every block to change, with line numbers (verified by scout, `research/readme_anchors.md`); (b) the exact target wording quoted from the PRD (§8.1 h3.8, §8.2 h3.9, §8.3 h3.10, §12.2 h3.12, §15 h2.14, §17 h2.16 — all in `<selected_prd_content>`); (c) the drift audit that maps each gap (G19/G20/G21, items 2a-2d) to its owning README region; (d) the four greps that prove done; (e) the example skill frontmatter to keep `--search` field list accurate. No guessing required.

### Documentation & References

```yaml
# MUST READ — the authoritative spec (all in <selected_prd_content>; quoted here for the implementer)
- file: PRD.md
  why: "§8.1 (h3.8) config file path + SKILLDOZER_CONFIG override + minimal `store:` YAML.
        §8.2 (h3.9) init flow (cwd auto-detect default $XDG_DATA_HOME/skilldozer/skills,
        `init <dir>` / `init --store <dir>` non-interactive, prints --path + check, never-prompts
        on bare tag). §8.3 (h3.10) the 5-rule ladder + the four --path labels. §12.2 (h3.12)
        go install is first-class; caveat removed. §15 (h2.14) the README outline + mcpeepants
        tone. §17 (h2.16) the catalog-vs-settings guardrail."
  section: "h3.8, h3.9, h3.10, h3.12, h2.14, h2.16 (all provided in <selected_prd_content>)."

# MUST READ — the verified current README anchors (verbatim, with line numbers)
- file: plan/002_38acb6d28a6a/P1M4T2S1/research/readme_anchors.md
  why: "Quotes the EXACT current text of every block to change: go-install caveat (lines 36-53),
        'How skilldozer finds the store' 3-rule + --path labels (234-251), Constraints/Manifest-free
        (253-271, esp 255-258), the title one-liner (3), and the canonical Usage block (75-120) to
        preserve verbatim. Confirms the forbidden terms (skilldozer init / config.yaml /
        SKILLDOZER_CONFIG / 'config file') are currently ABSENT (grep exit 1)."
  pattern: "Edit-in-place against these line anchors; do not reflow the whole file."

# MUST READ — the drift audit that owns this gap
- file: plan/002_38acb6d28a6a/architecture/docs_and_assets_drift.md
  why: "§2 items 2a-2d enumerate every README drift with PRD citation. 2a = no init/config
        anywhere; 2b = 3 rules not 5 + wrong hint + missing 'config file' label; 2c = obsolete
        go-install caveat; 2d = Constraints omits catalog-vs-settings reword. §2e confirms the
        flag inventory (Usage) is ALREADY correct — do not touch it."
  section: "§2 (README drift), items 2a-2d (change), 2e/2f (verify-not-change)."

- file: plan/002_38acb6d28a6a/architecture/code_prd_delta.md
  why: "G19 (README:42-53 go-install caveat), G20 (README:234-249 3-rule + missing config file
        label + no init/SKILLDOZER_CONFIG), G21 (README Constraints catalog-vs-settings reword)
        — the three gap IDs this PRP closes. §9 'What is CORRECTLY aligned' bounds what NOT to
        change."
  section: "G19/G20/G21 (gap index), §9 (correctly aligned)."

# READ-ONLY — the sibling PRP that proves the binary is already compliant
- file: plan/002_38acb6d28a6a/P1M4T1S1/PRP.md
  why: "P1.M4.T1.S1 runs the §13 acceptance suite. Its research proves the binary already
        implements init, the 5-rule Find(), the 'config file' label, and the exact ErrNotFound
        message. This README PRP documents that binary; it does not change it. If a §13 gate
        were failing, the README must still document the INTENDED behavior (the binary is the
        contract), and the failure routes to the owning Mode A subtask — not here."
  pattern: "Treat the implemented surface (init, config.yaml, SKILLDOZER_CONFIG, 5-rule order,
            exact unconfigured message) as the INPUT contract — it is what the README describes."

# REFERENCE — the example skill frontmatter (keeps --search field list accurate in Usage)
- file: skills/example/SKILL.md
  why: "The '## Adding a skill' template and the Usage --search line reference the frontmatter
        schema (name/description/metadata.keywords/metadata.category/aliases). That schema is
        unchanged by this PRP; the example already uses keywords: [example, demo, skilldozer]
        (fixed by P1.M3.T1.S1). Do not re-touch it."
```

### Current Codebase tree (run `tree` in the root of the project) to get an overview of the codebase

```bash
$ cd /home/dustin/projects/skilldozer && ls
README.md            # ← THE FILE THIS PRP EDITS (271 lines)
PRD.md               # read-only (always)
LICENSE              # compliant (do not touch)
go.mod  go.sum       # compliant (do not touch)
install.sh           # compliant (do not touch)
main.go  main_test.go        # already implements init + config (do not touch)
internal/{config,skillsdir,discover,resolve,ui,search,check}/  # do not touch
completions/{skilldozer.bash,_skilldozer,skilldozer.fish}      # compliant (P1.M3.T2.S1)
skills/example/SKILL.md       # compliant (P1.M3.T1.S1)
.gitignore            # compliant
skilldozer            # built binary (gitignored)
```

### Desired Codebase tree with files to be added and responsibility of file

```bash
README.md             # EDITED — the only file this PRP touches (4 in-place content edits)
```

No new files. No tree changes. The deliverable is the edited README.

### Known Gotchas of our codebase & Library Quirks

```markdown
<!-- CRITICAL: README.md is plain Markdown — no linter is wired up. There is NO `ruff`/`mypy`/
     `markdownlint` gate in this repo. "Validation" here is the four greps + a manual read.
     Do NOT invent a `markdownlint` command for the Validation Loop — it is not installed
     (verify: `command -v markdownlint` → not found). Use the greps in Validation Level 1. -->

<!-- CRITICAL (exactness of the unconfigured message): the binary's ErrNotFound is the literal
     string `skilldozer is not configured; run \`skilldozer init\`` (backticks literal). When you
     quote it in the README "How skilldozer finds the store" rule 5, render the backticks inside
     a code span so Markdown does not try to interpret them: `run \`skilldozer init\``. The §13
     gate `grep -q 'run \`skilldozer init\`'` runs against the BINARY's stderr, not the README —
     but the README should still show the user the exact string they will see. -->

<!-- GOTCHA (the --path label list must be FOUR, not three): the old README (line 248) lists
     `SKILLDOZER_SKILLS_DIR`, `sibling of binary`, `ancestor of cwd`. The config model adds
     `config file` (Source.String() case added by P1.M1.T2.S1). The README MUST list all four,
     in priority order: SKILLDOZER_SKILLS_DIR → config file → sibling of binary → ancestor of cwd.
     Omitting `config file` is the most likely regression. -->

<!-- GOTCHA (do NOT reword rule 1's env-var purpose away): the env var is still the override
     CI/tests/temp-redirects use (§8.3 rule 1). The stale README line 239 says "This is the
     override `go install` users set (see Install)" — that sub-clause is obsolete (go-install
     users now use init), but the env var ITSELF is not; reword to "override for CI / tests /
     temporary redirects" and DROP the "(see Install)" pointer to the removed caveat. -->

<!-- GOTCHA (keep the silent-fall-through guidance): README lines 249-251 explain that a bad
     SKILLDOZER_SKILLS_DIR value is silently ignored and --path is the only way to tell which
     rule won. That guidance is STILL correct and load-bearing (§8.3 "a bad value is silently
     ignored and falls through — --path is the only way to tell"). Preserve it verbatim or
     lightly reworded; do not delete it. -->

<!-- GOTCHA (Constraints opening line 255 says "manifest-free path printer"): softening the
     bullet to "No catalog index" can contradict the opening sentence. Reword line 255 too,
     e.g. "`skilldozer` is deliberately a thin path printer." Drop the word "manifest-free"
     from the opener OR clarify it refers only to the catalog (the §17 distinction: catalog
     index forbidden; settings config file permitted). Do not leave the opener contradicting
     the new bullet. -->

<!-- CRITICAL (scope): this is the README ONLY. Do not edit install.sh, go.mod, LICENSE,
     .gitignore, completions/*, skills/example/SKILL.md, main.go, internal/**, PRD.md,
     tasks.json, prd_snapshot.md, or the §13 transcript. All of those are either compliant,
     orchestrator-owned, or owned by a different subtask. -->
```

## Implementation Blueprint

### Data models and structure

None. This is a Markdown documentation edit. No types, no schemas, no code.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT README.md §15.3 Install — add the "First run: skilldozer init" note
  - FIND: the `## Install` heading (README.md:19) and its three sub-blocks A/B/C
    (A = ./install.sh ~20-34; B = go install 36-53; C = from-source 55+).
  - ADD (after the three install paths, before the next `##` heading): a short callout, e.g.:
        ### First run

        Whichever install path you used, run `skilldozer init` once:

        ```bash
        skilldozer init
        ```

        It prompts for the directory where skilldozer should keep your skills (defaulting to
        `$XDG_DATA_HOME/skilldozer/skills`, or the current directory if it already looks like a
        skill store), creates it, seeds an `example/SKILL.md` template if it is empty, and
        writes the config pointing at it. For scripts / CI, skip the prompt:

        ```bash
        skilldozer init /path/to/store      # positional
        skilldozer init --store /path/to/store
        ```
  - FOLLOW pattern: the existing README voice — terse, code-fence-first, one-sentence rationale.
    Match the §15.3 wording "First run: `skilldozer init` (prompts for the store dir, writes the
    config)."
  - NAMING: heading `### First run` (h3, nested under `## Install`); or a blockquote callout —
    pick whichever matches the README's existing subsection style (it currently has no h3 under
    Install, so a blockquote `> **First run.** ...` is closer to the existing voice; use that).
  - PLACEMENT: end of the `## Install` section, after the "C. From source" block.
  - DEPENDENCIES: none (this is independent of the other three edits).

Task 2: EDIT README.md §12.2 go install — remove the obsolete caveat (README.md:42-53)
  - FIND: the blockquote at README.md:42-53, i.e. EVERYTHING from
        > **`go install` caveat.** A `go install`'d binary lands in
    down through
        > discovery works with no env var.)
  - REPLACE WITH: a short first-class statement mirroring PRD §12.2 (h3.12). Example wording:
        On first use, run `skilldozer init` (see First run, above). It creates the store and
        writes the config. No clone required, and no `SKILLDOZER_SKILLS_DIR` needed for normal
        use.
  - KEEP: the code fence at README.md:38-40 (`go install github.com/dabstractor/skilldozer@latest`)
    and the `**B. \`go install\`` heading (README.md:36).
  - VERIFY after edit: `grep -nE '\`go install\` caveat|export SKILLDOZER_SKILLS_DIR' README.md`
    returns ZERO hits.
  - NAMING: none (prose).
  - PLACEMENT: in place — same position as the removed caveat (between the go-install code fence
    and the "C. From source" block).
  - DEPENDENCIES: coordinate with Task 1 (the new go-install wording says "see First run, above",
    so Task 1's First-run block must exist; if you add First-run AFTER Install's three paths, the
    go-install block precedes it — either move First-run above B, or phrase the pointer as "see
    First run below" / "run `skilldozer init` (the documented first command)"). Pick pointer
    wording consistent with final placement.

Task 3: EDIT README.md §15.7 "How `skilldozer` finds the store" — 3 rules → 5 rules + config + 4 labels
  - FIND: README.md:234-251 (heading `## How \`skilldozer\` finds the store` through the
    silent-fall-through paragraph).
  - REPLACE the 4-item ordered list (238-245) with the 5-rule §8.3 ladder (see "What" §3 above
    for exact rule text). Use the exact wording:
        1. **`SKILLDOZER_SKILLS_DIR` env var** — override; if set and an existing dir, use it.
           Lets CI / tests / temporary redirects win without editing the config.
        2. **Config file `store`** — the primary, set by `skilldozer init`. The config lives at
           `$XDG_CONFIG_HOME/skilldozer/config.yaml` (→ `~/.config/skilldozer/config.yaml`);
           override the file path with `SKILLDOZER_CONFIG=<file>` (handy for tests / multiple
           profiles). Minimal valid file:
             ```yaml
             store: /home/you/skills
           ```
           A missing or unreadable config is treated as "not yet configured" and falls through
           to the rules below — never a hard error.
        3. **Sibling of the running binary** (symlink-aware: `os.Executable()` +
           `filepath.EvalSymlinks()`) — still lets a clone-and-build dev workflow work with no
           config.
        4. **Walk up from `cwd`** — for `go run` / dev.
        5. **None** ⇒ unconfigured: skilldozer prints `skilldozer is not configured; run
           \`skilldozer init\`` to stderr, writes nothing to stdout, and exits 1.
  - REPLACE the `--path` paragraph (247-251): list all FOUR labels in priority order —
    `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`. KEEP the
    silent-fall-through guidance (a bad `SKILLDOZER_SKILLS_DIR` is silently ignored; `--path` is
    the only way to tell which rule won).
  - VERIFY after edit: `grep -n 'config file' README.md` hits (the new label); the rule list has
    exactly 5 numbered items.
  - FOLLOW pattern: the existing README bullet style (bold lead-in, one-line rationale).
  - NAMING: none (prose + one yaml code fence).
  - PLACEMENT: in place (same heading, same position in the file).
  - DEPENDENCIES: none.

Task 4: EDIT README.md §15.8 Constraints — reword Manifest-free to catalog-vs-settings
  - FIND: README.md:255-258 (the `**Manifest-free.**` bullet) and the opening line 255
    (`\`skilldozer\` is deliberately a thin, manifest-free path printer.`).
  - REPLACE the bullet with the §15.8 / §17 distinction. Example:
        - **No catalog index.** There is no `skills.json`, no manifest enumerating skills — the
          catalog is always walked from disk on each call. A *settings* config file (the store
          location, written by `skilldozer init`) is expected and fine; the rule is only that
          catalog data already on disk is never duplicated into a sidecar.
  - REWORD opening line 255 to drop the contradiction, e.g. "`skilldozer` is deliberately a thin
    path printer." (remove "manifest-free" from the opener, since the new bullet permits a
    settings file).
  - KEEP unchanged: the other Constraints bullets (Never auto-discovered by pi / Loaded only via
    `--skill` / only ever prints paths / Zero runtime dependencies) — README.md:259-271.
  - VERIFY after edit: `grep -nE 'catalog|settings' README.md` hits; the word "Manifest-free"
    is either gone or clarified.
  - NAMING: none (prose).
  - PLACEMENT: in place.
  - DEPENDENCIES: none.

Task 5: VERIFY (no edits — run the four greps + a manual read)
  - RUN the four success-criteria greps (see Validation Level 1).
  - READ the full README top-to-bottom once; confirm: voice still matches §15 (terse,
    example-first); the Usage flag inventory (75-120) is unchanged; "Where skills live" and
    "Adding a skill" are unchanged; no dangling "see Install" pointer to the removed caveat.
  - This task produces no file; it is the done-gate.
```

### Implementation Patterns & Key Details

```markdown
<!-- PATTERN: README voice. The existing README is terse and example-first: a one-line
     rationale, then a code fence. Match it. Do not add marketing prose, em dashes are fine
     (the existing README uses them), do not add headings the file does not already use. The
     §15 instruction is "Mirror the mcpeepants README's tone and structure" — the current
     skilldozer README already does; keep its cadence. -->

<!-- PATTERN: edit-in-place with the edit tool. Each of Tasks 1-4 is a bounded region with
     known start/end (the line anchors in research/readme_anchors.md). Use `edit` with the
     exact current `oldText` block and the new `newText`. Do NOT rewrite the whole file with
     `write` — that risks clobbering the Usage/Where-skills-live/Adding-a-skill sections that
     must stay byte-identical. -->

<!-- PATTERN: when quoting the unconfigured message, wrap backticks in a code span:
     `run \`skilldozer init\``. Markdown inside a code span does not re-process the inner
     backticks, so they render literally. -->

<!-- PATTERN: the config-file code fence uses the minimal valid YAML from §8.1:
     ```yaml
     store: /home/you/skills
     ```
     Do not add keys the PRD does not mention (no `category`, no `colors`) — §8.1 says unknown
     keys are ignored "room to grow" but the documented MINIMAL example is just `store:`. -->

<!-- CRITICAL (do not touch the §13 transcript): plan/002_38acb6d28a6a/acceptance_transcript.txt
     is owned by P1.M4.T1.S1. The README edit does not run §13 and must not reference the
     transcript path. -->
```

### Integration Points

```yaml
README SECTIONS (the only integration surface):
  - edit: "## Install (README.md:19-...) — add First-run note"
  - edit: "B. go install caveat (README.md:42-53) — remove, replace with first-class wording"
  - edit: "## How `skilldozer` finds the store (README.md:234-251) — 3 rules → 5 rules + config + 4 labels"
  - edit: "## Constraints / Manifest-free (README.md:255-258 + opener 255) — catalog-vs-settings reword"

NO CHANGES TO:
  - main.go / internal/** (already implement the config model — this PRP documents it, not changes it)
  - install.sh / go.mod / LICENSE / .gitignore (compliant per drift audit items 3/5/6/7)
  - completions/* (compliant per P1.M3.T2.S1)
  - skills/example/SKILL.md (compliant per P1.M3.T1.S1)
  - PRD.md (read-only, always)
  - tasks.json / prd_snapshot.md / acceptance_transcript.txt (orchestrator / sibling-subtask owned)
  - plan/002_38acb6d28a6a/** (this PRP's own research/ + PRP.md are the only plan/ writes allowed)
```

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
# There is no Markdown linter wired into this repo (verify: command -v markdownlint → not found).
# "Style" validation is the four success-criteria greps + a manual read.

cd /home/dustin/projects/skilldozer

# (1) init is documented
grep -n 'skilldozer init' README.md && echo "init-doc OK" || echo "init-doc FAIL"

# (2) config model is documented (both the path and the override)
grep -n 'config\.yaml' README.md && grep -n 'SKILLDOZER_CONFIG' README.md && echo "config-doc OK" || echo "config-doc FAIL"

# (3) the new 'config file' --path label is present
grep -n 'config file' README.md && echo "config-label OK" || echo "config-label FAIL"

# (4) the obsolete caveat is GONE (expect ZERO hits → grep exits 1 → the && branch is skipped → echo FAIL if any hit)
if grep -nq 'export SKILLDOZER_SKILLS_DIR' README.md; then echo "caveat-still-present FAIL"; else echo "caveat-removed OK"; fi
if grep -nq '`go install` caveat' README.md; then echo "caveat-heading-still-present FAIL"; else echo "caveat-heading-removed OK"; fi

# Expected: all six markers print OK (init-doc, config-doc, config-label, caveat-removed,
#           caveat-heading-removed). If any FAIL, re-read the drift audit and fix the region.
```

### Level 2: Unit Tests (Component Validation)

Not applicable — there is no code. As a structural sanity check, confirm the four target sections still parse as coherent Markdown and the untouched sections are byte-identical:

```bash
cd /home/dustin/projects/skilldozer

# The README still has exactly the 8 §15 outline headings (Why, Install, [Shell completions],
# Usage, Where skills live, Adding a skill, How skilldozer finds the store, Constraints) plus
# the title. None should have been renamed or removed.
grep -nE '^## (Why|Install|Shell completions|Usage|Where skills live|Adding a skill|How .skilldozer. finds the store|Constraints)' README.md
# Expected: 8 heading hits (Shell completions is the extra-but-harmless one §15 doesn't enumerate).

# The Usage flag inventory is unchanged (spot-check a few lines that must still be present):
grep -nF 'skilldozer --path' README.md     # --path doc
grep -nF 'skilldozer --search reddit' README.md   # --search doc
grep -nF 'skilldozer --no-color --list' README.md # --no-color doc
grep -nF 'skilldozer -f example' README.md        # -f doc
# Expected: each returns its original line number (unchanged).
```

### Level 3: Integration Testing (System Validation)

The README describes the binary; the binary's behavior is verified by P1.M4.T1.S1 (§13 acceptance). This PRP does not run the binary. The integration check is that the README's claims match what `skilldozer --help` and `skilldozer --path` actually print:

```bash
cd /home/dustin/projects/skilldozer
go build -o skilldozer . 2>/dev/null   # ensure binary is current (gitignored artifact)

# The README's "How skilldozer finds the store" lists 4 --path labels. Confirm the binary
# actually emits them (it should, per P1.M1.T2.S1 Source.String() cases):
./skilldozer --path 2>&1 | grep -oE 'found via (SKILLDOZER_SKILLS_DIR|config file|sibling of binary|ancestor of cwd)'
# Expected: one of the four labels, matching a README-documented label.

# The README's rule-5 quotes the unconfigured message. Confirm the binary emits it (isolated):
mkdir -p /tmp/sd-readme-check && cp ./skilldozer /tmp/sd-readme-check/skilldozer && cd /tmp/sd-readme-check
env -u SKILLDOZER_SKILLS_DIR HOME=/tmp/sd-readme-check/home XDG_CONFIG_HOME=/tmp/sd-readme-check/home/.config \
  ./skilldozer x 2>&1 | grep -o 'skilldozer is not configured; run `skilldozer init`'
# Expected: the exact string the README quotes in rule 5.
cd - >/dev/null

# Expected: the README's documented labels/messages match the binary's actual output. If they
# diverge, the BINARY is the contract — fix the README to match the binary (do not change code).
```

### Level 4: Creative & Domain-Specific Validation

```bash
# Manual read-through (the real "does this read well" gate). Open README.md and confirm:
#   - The Install section flows: A ./install.sh → B go install (first-class, no caveat) →
#     C from-source → First run: skilldozer init.
#   - The go-install block no longer tells users to export anything.
#   - "How skilldozer finds the store" lists 5 rules in priority order, names the config file
#     path + SKILLDOZER_CONFIG override, shows the minimal `store:` YAML, and lists 4 --path labels.
#   - Constraints distinguishes catalog index (forbidden) from settings config file (permitted).
#   - Voice matches the rest of the README (terse, example-first, em dashes ok, no marketing tone).
#   - No dangling "see Install" pointer to the removed caveat; no leftover mention of cloning
#     the repo as a go-install prerequisite.

# Optional: render the README to eyeball Markdown fidelity (if a renderer is handy):
#   glow README.md   # or: mdcat README.md   (not required; manual read suffices)
```

## Final Validation Checklist

### Technical Validation

- [ ] All four success-criteria greps pass (init / config.yaml+SKILLDOZER_CONFIG / 'config file' present; `export SKILLDOZER_SKILLS_DIR` + `` `go install` caveat`` absent).
- [ ] The 8 §15 outline headings are all still present and correctly named.
- [ ] The Usage flag inventory (`--path`/`--search`/`--list`/`--all`/`check`/`-f`/`--relative`/`--no-color`/`--version`) is byte-identical to the pre-edit README.
- [ ] "Where skills live" and "Adding a skill" sections are unchanged.
- [ ] Level 3 integration check: the README's documented `--path` labels and unconfigured message match the binary's actual output.

### Feature Validation

- [ ] Install section documents `skilldozer init` as the first command (interactive + `init <dir>` / `init --store <dir>`).
- [ ] go-install subsection states first-class, no caveat, no clone/env-var prerequisite.
- [ ] "How `skilldozer` finds the store" enumerates the 5-rule §8.3 ladder (env → config → sibling → walk-up → unconfigured) and the 4 `--path` labels.
- [ ] Config file path (`$XDG_CONFIG_HOME/skilldozer/config.yaml`), `SKILLDOZER_CONFIG` override, and minimal `store:` YAML are documented.
- [ ] Constraints section surfaces the catalog-vs-settings distinction (catalog index forbidden; settings config file permitted).
- [ ] Manual read: tone matches §15 / the existing README voice.

### Code Quality Validation

- [ ] Edits were made in-place with `edit` (not a whole-file `write` rewrite) to preserve untouched sections.
- [ ] No source code, no other docs, no orchestrator-owned files were modified.
- [ ] No fabricated tooling (no `markdownlint` / `ruff` invented for validation).
- [ ] Backticks in the unconfigured message render literally (wrapped in a code span).

### Documentation & Deployment

- [ ] The README is self-consistent (no opener contradicts a bullet; no dangling pointer to removed content).
- [ ] The README matches the shipped binary (the §13-green binary from P1.M4.T1.S1).
- [ ] This is the final deliverable of the phase — no follow-up doc subtask remains.

---

## Anti-Patterns to Avoid

- ❌ Don't rewrite the whole README with `write` — edit the four regions in place; a full rewrite risks clobbering the already-correct Usage/Where-skills-live/Adding-a-skill sections.
- ❌ Don't touch any file other than `README.md` (no code, no other docs, no orchestrator files).
- ❌ Don't leave the word "manifest-free" in the Constraints opener if the new bullet permits a settings file — fix the contradiction.
- ❌ Don't list only 3 `--path` labels — the config model adds `config file`; all four must appear.
- ❌ Don't keep the stale "(see Install)" pointer on the env-var rule — the Install section no longer tells go-install users to set the env var.
- ❌ Don't delete the silent-fall-through guidance (bad `SKILLDOZER_SKILLS_DIR` is ignored, `--path` is the only way to tell) — it is still correct and load-bearing.
- ❌ Don't invent a Markdown linter for the Validation Loop — none is installed; use the four greps + a manual read.
- ❌ Don't paraphrase the unconfigured message — the binary emits exactly `skilldozer is not configured; run \`skilldozer init\``; the README should quote it verbatim (backticks in a code span).
- ❌ Don't run or modify the §13 acceptance transcript (owned by P1.M4.T1.S1).
- ❌ Don't document config keys the PRD does not mention (no `category`/`colors` in the minimal `store:` example).

---

## Confidence Score

**9/10.** This is a pure documentation edit against a precisely-scoped, well-audited gap. The drift audit (`docs_and_assets_drift.md` §2 items 2a-2d, `code_prd_delta.md` G19/G20/G21) enumerates every change with exact line numbers and PRD citations; the scout research (`research/readme_anchors.md`) quotes the verbatim current text of every block to change and confirms the forbidden terms are currently absent; the target wording is quoted directly from the PRD (§8.1/§8.2/§8.3/§12.2/§15/§17, all in `<selected_prd_content>`); and the binary that the README describes is already §13-green (P1.M4.T1.S1). The remaining 1/10 is editorial judgment: the exact phrasing of the First-run callout, the catalog-vs-settings bullet, and the rule-1 reword are not prescribed verbatim by the PRD, so a reviewer could prefer different wording — but the four greps + manual read are an objective done-gate that catches any substantive miss.
