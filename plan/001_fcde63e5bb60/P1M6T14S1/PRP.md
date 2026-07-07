# PRP — P1.M6.T14.S1: `README.md` (per §15 outline, mcpeepants tone)

> **Subtask:** P1.M6.T14.S1 — build-order step 7 (packaging/docs cluster).
> Create `README.md` at repo root that documents `skpp` for end users, following
> the **exact 8-section outline in PRD §15**, in the **plain, example-driven
> tone of the `mcpeepants` README**.
>
> **Scope:** CREATE exactly **one** new file — `README.md` — at repo root. No Go
> code. No edits to `install.sh`, `skills/`, `internal/`, `go.mod`, `.gitignore`,
> or `PRD.md`. The binary, the example skill, and `install.sh` are all already
> landed (M1–M5 + T12 + T13); this task writes the user-facing doc that explains
> how to install and use the finished product.
>
> **Why this matters:** the README is the first thing a `go install` user sees,
> and the canonical `pi --skill "$(skpp tag)"` one-liner must be discoverable in
> the first 30 lines. Get the install caveats right (especially the `go install` →
> `SKPP_SKILLS_DIR` footgun) and users succeed on their own; get them wrong and
> every `go install` user hits "can't find skills/". The README is also the place
> the §17 constraints (manifest-free; never auto-discovered by pi) are made
> **user-visible**, which is the whole product posture.

---

## Goal

**Feature Goal**: A user who clones the repo (or runs `go install`) can, **from
the README alone**, (a) install `skpp`, (b) understand the one-liner
`pi --skill "$(skpp <tag>)"`, (c) add their own skill under `skills/`, and (d)
know how `skpp` finds the store and why skills are never auto-discovered by pi —
without reading `PRD.md` or any source.

**Deliverable**: One new file, `README.md`, at repo root. Markdown. Covers PRD
§15's 8 sections in order. Mirrors the mcpeepants README's plain, example-driven
tone. Every code block in it is runnable and its documented output matches the
real `skpp` binary (captured in this PRP's research).

**Success Definition** (all must hold):
- `README.md` exists at repo root and renders cleanly as GitHub markdown.
- The literal one-liner `pi --skill "$(skpp example)"` appears in the first
  `## Usage` code block (the canonical contract, PRD §1/§3).
- All 8 PRD §15 sections are present, in order, as `##` headings.
- The `go install` path prominently documents the `SKPP_SKILLS_DIR` requirement
  (the #1 footgun — a `go install`'d binary has no adjacent `skills/`).
- `skpp install.sh`-path and `SKPP_INSTALL_BIN` are NOT conflated with the
  runtime `SKPP_SKILLS_DIR`.
- Every example's stated output matches the actual binary (re-run the commands
  from `research/verified_facts.md` §3 before finalizing).
- The §17 "never auto-discovered by pi" constraint is stated plainly in the
  Constraints section, naming the forbidden locations.
- No broken cross-references: completions are mentioned **only** if
  `completions/` exists at write time (T15 may not be done).
- `git status` shows ONLY `README.md` as a new untracked file.

## User Persona

**Target User**: A pi operator who wants a centralized, on-disk skill catalog
addressed by tag. They have `pi` installed and know what a skill is; they do NOT
want to read the PRD.

**Use Case**: "I want to load a specific skill into pi by a short tag, from a
store pi doesn't auto-scan." → `pi --skill "$(skpp my-tag)"`.

**Pain Points Addressed**:
- "Where do I point `--skill`?" → README's first block answers with the one-liner.
- "I `go install`'d and `skpp example` says it can't find skills." → the
  `SKPP_SKILLS_DIR` caveat, documented prominently.
- "How do I add my own skill?" → a copy-pasteable frontmatter block + `skpp check`.
- "Will pi pick these up automatically and clutter my context?" → Constraints
  section says no, and explains why (loads only via explicit `--skill`).

## Why

- **PRD §15 — the authoritative README outline.** Eight sections, fixed order;
  this PRP maps each to a README `##`. The one-liner and the §17 constraints are
  non-negotiable content.
- **PRD §2 constraint 5 (one-shot buildable) + §3 (pi skill loading).** The
  README is how a new user satisfies §3's contract without reading the spec. The
  canonical example must be unmissable.
- **PRD §12.2 — `go install` caveat is a documented requirement.** A `go
  install`'d binary lands in `$(go env GOPATH)/bin` with no sibling `skills/`,
  so §8 rule 2 (sibling-of-binary) misses and the user must set
  `SKPP_SKILLS_DIR`. The PRD explicitly says "Document this prominently." This is
  the single most failure-prone install path and the README's job to defuse.
- **mcpeepants parity (PRD §1, §15, decisions §1/§2/§11).** `skpp` is to skills
  what mcpeepants is to MCP server configs; the README mirrors mcpeepants' tone
  (plain, lead with the canonical one-liner, bullet capability lists, commented
  examples, no marketing fluff) while carrying a richer §15 structure.
- **Cohesion with sibling tasks:** the binary (M1–M5), example skill (T12), and
  `install.sh` (T13) are landed; completions (T15) and the §13 acceptance sweep
  (T16) follow. This README documents the **already-working** product and must
  not describe behavior that does not exist (e.g. completions before T15). It is
  the doc layer; it must not imply code changes. PRD §13's acceptance includes a
  pi end-to-end line that the README's "Usage" examples must match verbatim.

## What

User-visible artifact: `README.md` at repo root. It contains, **in this order**
(PRD §15), as `##` sections:

1. **Title + one-liner.** `# skpp`, then one sentence: *"Standalone skill loader
   for pi — resolves a skill tag to an absolute path for `pi --skill`."*
2. **Why.** A centralized skill store that is **deliberately not** in any pi
   discovery location; skills load **only on demand** via `--skill`.
3. **Install.** Three paths:
   - `./install.sh` (recommended; builds + symlinks into `~/.local/bin`; finds
     `skills/` automatically via the binary-sibling rule).
   - `go install github.com/dabstractor/skpp@latest` (**+ the `SKPP_SKILLS_DIR`
     caveat in a callout/blockquote**, prominent — see Success Definition).
   - from-source (`go build -o skpp .` / manual `ln -sfn`).
4. **Usage.** The canonical one-liner **first** (`pi --skill "$(skpp example)"`),
   then multi-skill, `-f`/`--file`, `--list`, `--search`, `--all`, `check`,
   `--path`. A short note on the error contract (unknown tag → nothing on stdout,
   exit 1) because it protects the `$(...)` use.
5. **Where skills live.** The `skills/` dir; the canonical **tag = the skill
   dir's path relative to `skills/`** (separators `/`); nested skills count; the
   §7.2 resolution precedence (exact → basename → `name` → alias) in one line each.
6. **Adding a skill.** Drop `skills/<tag>/SKILL.md`; show the required
   frontmatter (`name`, `description`) + optional `metadata` (keywords/category/
   aliases); point at `skills/example/SKILL.md` as the copy-pasteable template;
   end with `skpp check`.
7. **How `skpp` finds the store.** The §8 priority order (env → sibling-of-binary
   → walk-up-from-cwd → fail-with-fix) as a short numbered list; note
   `SKPP_SKILLS_DIR` overrides everything and that a symlink install is what
   makes the sibling rule work.
8. **Constraints.** Manifest-free; never auto-discovered by pi (name the
   forbidden locations: `~/.pi/agent/skills`, `~/.agents/skills`, project
   `.pi/skills`/`.agents/skills`, `node_modules` packages, `package.json`'s
   `pi.skills`); loads only via `--skill`; `skpp` only ever **prints** paths.

### Success Criteria

- [ ] `README.md` exists at repo root and is valid GitHub markdown.
- [ ] First `## Usage` code block contains the literal `pi --skill "$(skpp example)"`.
- [ ] All 8 §15 sections present, in order, as `##` headings.
- [ ] `go install` path documents `SKPP_SKILLS_DIR` prominently (callout/blockquote).
- [ ] `SKPP_INSTALL_BIN` (install-time) is not confused with `SKPP_SKILLS_DIR`
      (runtime); each appears only in its correct section.
- [ ] Each example's documented output matches the real binary (re-run §3 probes).
- [ ] Constraints section names the forbidden pi-discovery locations.
- [ ] No reference to `completions/` unless that dir exists at write time.
- [ ] No em-dash marketing prose; matches mcpeepants tone (plain, example-driven).
- [ ] `git status` shows only `README.md` as new/untracked.

## All Needed Context

### Context Completeness Check

_If someone knew nothing about this codebase, would they have everything needed
to implement this successfully?_ **Yes.** Every section's content is specified
below, the exact CLI output to cite is captured verbatim in
`research/verified_facts.md`, the tone model (mcpeepants README) is quoted in
full in the same file, and the structural spec (PRD §15) is reproduced inline.
No source files must be read to WRITE the README — they are referenced only to
VERIFY that the documented examples match reality. The implementer's only
judgment calls are wording (within the tone rules) and whether `completions/`
exists yet.

### Documentation & References

```yaml
# MUST READ — the authoritative structure spec (8 sections, fixed order)
- file: PRD.md
  section: "§15 'README.md outline' — the 8 numbered sections ARE the README's headings, in order."
  why: "Single source of truth for README structure and the required one-liner text."
  critical: "§15 item 1 fixes the one-liner wording; §15 item 3 requires the go install
             + SKPP_SKILLS_DIR caveat to be 'documented prominently'. §15 maps 1:1 to headings."

# MUST READ — the exact CLI behavior to cite (README examples MUST match reality)
- docfile: plan/001_fcde63e5bb60/P1M6T14S1/research/verified_facts.md
  section: "§3 'Actual skpp CLI behavior' — full --help, command outputs, table formats, error contract."
  why: "The README's code blocks must show REAL output. Cite this verbatim; re-run the
        commands before finalizing so no example drifts from the binary."
  critical: "The error contract (unknown tag → NOTHING on stdout, exit 1) is load-bearing
             for pi --skill \"$(skpp x)\"; the Usage/Constraints sections must state it."

# MUST READ — the tone/structure to mirror
- file: ~/projects/mcpeepants/README.md
  why: "mcpeepants is the named tone model (PRD §1/§15). Plain # Title + one sentence;
        lead with the canonical one-liner; `key - description` bullets; commented examples."
  pattern: "# mcpeepants \n\nCLI helper for generating MCP server configurations. \n\n## Usage ..."
  gotcha: "mcpeepants README is SHORT (~50 lines). skpp's is richer (§15 = 8 sections) but
           must keep the same plain, example-driven, no-marketing voice. No badges/emojis
           in headings, no 'Features:' hard-sell, no adjectives like 'blazing'/'seamless'."

# MUST READ — what to mirror vs diverge (confirms README shape + tone)
- file: plan/001_fcde63e5bb60/architecture/mcpeepants_patterns.md
  section: "'README tone (from mcpeepants README)' + the 'skpp README should follow the same shape' note."
  why: "Explicitly states the section order skpp must follow and that completions may be deferred."

# MUST READ — the install behavior the README documents (install.sh already exists)
- file: install.sh
  why: "The README §Install section documents what THIS script does. Read it so the documented
        target order (SKPP_INSTALL_BIN → ~/.local/bin → /usr/local/bin), symlink-not-copy, and
        PATH-advice behavior are accurate."
  critical: "install.sh uses SKPP_INSTALL_BIN (install-time target). Do NOT conflate with
             SKPP_SKILLS_DIR (runtime skills-dir override, §8 rule 1). Document each only in
             its correct section."

# VERIFY-AGAINST — the frontmatter template the README points readers at
- file: skills/example/SKILL.md
  why: "README §6 'Adding a skill' should point at this as the copy-pasteable template and
        show the same required (name, description) + optional metadata (keywords/category/aliases)
        fields. Cite its real frontmatter."

# REFERENCE — discovery/§8 rules the README summarizes (sections 5 & 7)
- file: internal/skillsdir/skillsdir.go
  section: "PRD §8 priority: env (SKPP_SKILLS_DIR) → sibling-of-binary (os.Executable + EvalSymlinks) → walk-up-from-cwd."
  why: "So the README's 'How skpp finds the store' numbered list is accurate; the symlink
        install is what makes the sibling rule win for ./install.sh users."

# REFERENCE — the §13 acceptance the README examples must satisfy
- file: PRD.md
  section: "§13 'Acceptance criteria' — the pi end-to-end line + the install/symlink assertions."
  why: "README Usage examples should match §13's verbatim invocations so docs and tests agree."

# OPTIONAL quality lever — aligns with mcpeepants plain-prose tone
- skill: write-tech-docs
  why: "Available in this env. Enforces no em dashes, no marketing tell-words, no hedging,
        ships a linter. Aligns with the required mcpeepants tone."
  critical: "OPTIONAL polish only. The BINDING spec is PRD §15 + mcpeepants tone. Do not let
             the skill's rules override the PRD's required one-liner/section content."
```

### Current Codebase tree (relevant slice)

```bash
skpp/
├── PRD.md                  # §15 = the README's structure spec; §13 = examples to match
├── README.md               # ← DOES NOT EXIST YET (this task creates it)
├── LICENSE                 # MIT (2026, Dustin Schultz) — mention if a License section is added
├── install.sh              # exists; README §Install documents its behavior
├── main.go                 # the binary; README documents its CLI (output in research §3)
├── go.mod                  # module github.com/dabstractor/skpp ; go install path lives here
├── internal/               # discovery/§7 + skillsdir/§8 — README summarizes these rules
├── skills/example/SKILL.md # the copy-pasteable skill template README §6 points at
└── (completions/ may or may not exist — T15 status Planned)
```

### Desired Codebase tree with files to be added

```bash
skpp/
└── README.md               # NEW (this task). ~120–200 lines markdown.
                           #   8 §15 sections as ## headings, mcpeepants tone.
                           #   Runnable code blocks matching the real binary.
# (NO other files change. README is committed; nothing gitignores it.)
```

### Known Gotchas of our codebase & Documentation Quirks

```markdown
<!-- CRITICAL (1): document ACTUAL behavior, never aspirational. -->
<!-- The binary is already built and its exact output is captured in
     research/verified_facts.md §3. Re-run each command before finalizing so
     the README's example blocks match byte-for-byte. A README that shows
     invented output (e.g. a tag that resolves differently) is worse than none. -->

<!-- CRITICAL (2): the go install footgun MUST be prominent. -->
<!-- `go install github.com/dabstractor/skpp@latest` puts the binary in
     $(go env GOPATH)/bin, which has NO sibling skills/ dir. §8 rule 2 then
     misses and the user must set SKPP_SKILLS_DIR=/abs/path/to/repo/skills.
     PRD §12.2 says "Document this prominently." Use a blockquote/callout, not
     a buried footnote. -->

<!-- CRITICAL (3): SKPP_INSTALL_BIN ≠ SKPP_SKILLS_DIR. -->
<!-- SKPP_INSTALL_BIN is an install.sh-time target-dir override (install.sh §4).
     SKPP_SKILLS_DIR is a RUNTIME skills-dir override (§8 rule 1). They are
     different vars for different stages. Put each ONLY in its correct section
     (Install vs How skpp finds the store) to avoid confusing users. -->

<!-- GOTCHA (4): do NOT hardcode a version string. -->
<!-- `skpp --version` prints the git-describe value (short SHA when no tags,
     e.g. "skpp cc347c6"; a tag once released). Show `skpp --version` as the
     command, not a fixed string. The version is dynamic via ldflags. -->

<!-- GOTCHA (5): state the error contract explicitly. -->
<!-- Unknown tag => NOTHING on stdout + exit 1 (verified). This is WHY
     `pi --skill "$(skpp badtag)"` fails loudly instead of passing "" to pi.
     Mention it under Usage and/or Constraints — it's the product's safety model. -->

<!-- GOTCHA (6): completions may not exist yet (T15 = Planned). -->
<!-- Do NOT document `completions/skpp.bash` etc. unless that dir exists at
     write time. A one-line "Shell completions: see completions/" pointer is OK
     only if present. Broken file references in a README erode trust. PRD §14
     explicitly allows deferring completions. -->

<!-- GOTCHA (7): tag = relative dir path, with `/` separators. -->
<!-- The canonical tag is the skill dir's path RELATIVE TO skills/, normalized
     to forward slashes (e.g. "writing/reddit"). Not the frontmatter `name`.
     State this precisely in §5 — users will assume tag == name otherwise. -->

<!-- GOTCHA (8): the forbidden pi locations (PRD §17) must be named. -->
<!-- The "never auto-discovered by pi" claim is only credible if the README
     lists where pi DOES scan and says skpp uses none of them:
     ~/.pi/agent/skills, ~/.agents/skills, project .pi/skills or .agents/skills,
     node_modules packages, package.json's pi.skills. -->

<!-- GOTCHA (9): markdown must render on GitHub. -->
<!-- Use fenced ```bash blocks, tables only if simple, blockquotes for the
     go-install callout. Avoid raw HTML. Keep lines < ~100 cols where natural. -->

<!-- GOTCHA (10): match the mcpeepants voice, not its LENGTH. -->
<!-- mcpeepants README is ~50 lines; skpp's is richer (8 mandated sections).
     Keep the VOICE (plain, lead with one-liner, commented examples, no
     marketing adjectives) while carrying the fuller §15 structure. -->
```

## Implementation Blueprint

### Structure (no code/data models — this is prose; organize as the 8 §15 headings)

`README.md` is a single Markdown file. The 8 PRD §15 items become 8 `##` sections
in order, under a `# skpp` H1 + one descriptive sentence (the §15 item-1
one-liner). Keep each section tight: enough to be self-sufficient, not so much it
stops reading like the mcpeepants tone.

```markdown
# skpp

Standalone skill loader for pi — resolves a skill tag to an absolute path for `pi --skill`.

## Why
## Install
## Usage
## Where skills live
## Adding a skill
## How `skpp` finds the store
## Constraints
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CAPTURE the authoritative CLI output (do this BEFORE writing prose)
  - RUN: the commands in research/verified_facts.md §3 against ./skpp:
      ./skpp --version ; ./skpp --path ; ./skpp example ; ./skpp -f example ;
      ./skpp --all ; ./skpp --relative example ; ./skpp --list ;
      ./skpp --search example ; ./skpp check ; ./skpp --help
  - ALSO RUN the error-contract probes:
      out=$(./skpp nope 2>/dev/null); echo "rc=$? stdout-empty=[ -z $out ]"
      ./skpp nope 2>&1 1>/dev/null   # stderr text
  - RECORD exact outputs; the README's code blocks will cite these verbatim.
  - WHY: the README must document REAL behavior (Gotcha 1). If any output differs
    from research §3, TRUST THE LIVE BINARY and update your notes — do not edit
    the binary (out of scope) or invent output.

Task 2: CREATE README.md — H1 + one-liner (PRD §15 item 1)
  - WRITE: "# skpp" then the fixed sentence (PRD §15 item 1):
    "Standalone skill loader for pi — resolves a skill tag to an absolute path
    for `pi --skill`."
  - FOLLOW pattern: mcpeepants README line 1–2 ("# mcpeepants\n\nCLI helper for...").
  - GOTCHA: do NOT add badges, tagline emojis, or a "Features" h2. mcpeepants has none.

Task 3: APPEND §Why (PRD §15 item 2) — 2–4 sentences
  - COVER: centralized skill store; deliberately NOT in any pi discovery location;
    skills load ONLY on demand via --skill. (No need to enumerate locations here;
    that's Constraints.)
  - KEEP: plain, factual. No "powerful"/"seamless".

Task 4: APPEND §Install (PRD §15 item 3) — three paths, go-install caveat prominent
  - PATH A (recommended): `./install.sh`
      - one line what it does: builds with version ldflags + symlinks into ~/.local/bin
        (or $SKPP_INSTALL_BIN, or /usr/local/bin).
      - note: symlink install (NOT copy) is what lets skpp auto-find skills/.
  - PATH B: `go install github.com/dabstractor/skpp@latest`
      - PROMINENT CALLOUT (blockquote): the binary lands in $(go env GOPATH)/bin
        with NO sibling skills/; you MUST set
        `export SKPP_SKILLS_DIR=/abs/path/to/cloned/skpp/skills`. (Gotcha 2.)
  - PATH C (from-source): `go build -o skpp .` then `./skpp ...`, or manual
        `ln -sfn "$PWD/skpp" ~/.local/bin/skpp`.
  - GOTCHA (3): do NOT mention SKPP_INSTALL_BIN here beyond path A; keep
    SKPP_SKILLS_DIR for the §How-skpp-finds-the-store section (it is runtime).
    EXCEPTION: the go-install callout names SKPP_SKILLS_DIR because it's the fix.

Task 5: APPEND §Usage (PRD §15 item 4) — one-liner FIRST, then the rest
  - FIRST code block (the canonical contract):
        ```bash
        pi --skill "$(skpp example)"
        ```
  - THEN examples (commented, mcpeepants style) covering: multi-skill
        (`pi --skill "$(skpp a)" --skill "$(skpp b)"` / `skpp a b`),
        `-f`/`--file` (prints SKILL.md path), `--list`, `--search <q>`, `--all`,
        `check`, `--path`.
  - INCLUDE a one-line error-contract note: an unknown tag prints nothing to
    stdout and exits 1, so `pi --skill "$(skpp badtag)"` fails loudly. (Gotcha 5.)
  - CITE real output from Task 1 for at least: `skpp example`, `skpp -f example`,
    `skpp --list`, `skpp check`.
  - GOTCHA (4): show `skpp --version` as a command, not a fixed string.

Task 6: APPEND §Where skills live (PRD §15 item 5)
  - COVER: the skills/ dir; canonical tag = skill-dir path relative to skills/,
    `/`-separated (Gotcha 7); nested skills count (skills/writing/reddit/SKILL.md
    → tag writing/reddit).
  - ONE LINE EACH for §7.2 resolution precedence: exact tag → basename →
    frontmatter name → declared alias → unknown.
  - EXAMPLE: assume skills/foo (name: foo-helper) and skills/writing/reddit;
    show skpp foo, skpp writing/reddit, skpp reddit, skpp foo-helper resolving.
    (PRD §7.2 examples.)

Task 7: APPEND §Adding a skill (PRD §15 item 6)
  - COVER: drop skills/<tag>/SKILL.md; required frontmatter (name, description);
    optional metadata (keywords/category/aliases); unknown keys ignored.
  - SHOW the frontmatter block (mirror skills/example/SKILL.md). Point readers
    at skills/example/SKILL.md as the copy-pasteable template.
  - END with: run `skpp check` to validate (shows the OK/summary format).

Task 8: APPEND §How skpp finds the store (PRD §15 item 7)
  - NUMBERED list of §8 priority:
      1. SKPP_SKILLS_DIR env (wins if set + exists).
      2. sibling of the binary (os.Executable + EvalSymlinks) — what ./install.sh
         relies on; why symlink-not-copy matters.
      3. walk-up from cwd (dev / go run).
      4. else: error + one-line fix.
  - NOTE: `skpp --path` reports which rule won.

Task 9: APPEND §Constraints (PRD §15 item 8)
  - COVER (PRD §17): manifest-free (no skills.json/index); never auto-discovered
    by pi — NAME the forbidden locations (Gotcha 8): ~/.pi/agent/skills,
    ~/.agents/skills, project .pi/skills or .agents/skills, node_modules
    packages, package.json's pi.skills; loads ONLY via --skill; skpp only ever
    PRINTS paths (never copies/installs into ~/.pi or ~/.agents).

Task 10: CONDITIONALLY append a one-line completions pointer (Gotcha 6)
  - IF `test -d completions`: add under §Install a one-liner "Shell completions
    (optional): see completions/ for bash/zsh/fish."
  - ELSE: OMIT entirely. Do NOT document completion commands that don't exist.

Task 11: OPTIONALLY license pointer + verify rendering
  - IF a §License or footer is added: state MIT (LICENSE file), match mcpeepants
    brevity. (Optional — §15 doesn't require it.)
  - RENDER check: preview as GitHub markdown (no raw HTML; fenced ```bash blocks;
    blockquote for the go-install callout).

Task 12: RE-RUN the success-criteria checks (see Validation Loop)
  - grep for the one-liner; confirm 8 ## sections in order; re-verify example
    outputs against ./skpp; `git status` shows only README.md new.
```

### Implementation Patterns & Key Details

```markdown
<!-- The canonical one-liner — must be unmissable, first in §Usage -->
```bash
pi --skill "$(skpp example)"
```

<!-- The go-install callout — the #1 footgun, must be prominent (blockquote) -->
> **`go install` caveat:** a `go install`'d binary lands in `$(go env GOPATH)/bin`
> with no adjacent `skills/` directory, so `skpp` cannot auto-discover the store.
> Set the runtime override before use:
> ```bash
> export SKPP_SKILLS_DIR=/absolute/path/to/your/cloned/skpp/skills
> ```
> (Prefer `./install.sh`, which symlinks the binary next to the repo so discovery
> works with no env var.)

<!-- mcpeepants-style commented example block -->
```bash
# Resolve a tag to an absolute path (default: the skill directory)
skpp example                       # → /…/skills/example

# Print the SKILL.md file path instead
skpp -f example                    # → /…/skills/example/SKILL.md

# Load several skills into pi in one command
pi --skill "$(skpp writing/reddit)" --skill "$(skpp example)"

# Human-readable catalog and substring search
skpp --list
skpp --search reddit

# Validate every skill on disk
skpp check
```

<!-- Tone rule (Gotcha 10): keep the mcpeepants voice across the richer structure.
     Plain sentences, lead with the one-liner, `key - description` bullets,
     commented examples. Avoid em dashes for emphasis, marketing adjectives,
     and formulaic transitions. -->
```

### Integration Points

```yaml
DOCUMENTATION:
  - creates: README.md (repo root). Committed; nothing gitignores it.
  - references: PRD.md (only if you link to it — optional; README should be self-sufficient),
    skills/example/SKILL.md (the copy-paste template), install.sh (the documented installer).

NO CODE CHANGES:
  - main.go, internal/*, go.mod, install.sh, skills/*, .gitignore, LICENSE: UNTOUCHED.
  - This task is pure documentation. If the binary's behavior doesn't match what the
    README needs to say, the README documents REALITY (re-run §3 probes); do NOT
    change code to match aspirational docs (out of scope; file a note instead).

COMPLETIONS COUPLING:
  - README mentions completions/ ONLY if it exists (T15 Planned). No hard dependency;
    the README is shippable with or without T15.
```

## Validation Loop

> **Note on testing:** this is a documentation deliverable; there are no unit
> tests for prose. Validation is: (1) structural grep checks against PRD §15,
> (2) re-running the cited CLI commands to prove the examples match the binary,
> (3) a markdown render sanity check. This mirrors how the rest of M6 packaging
> tasks are validated (manual acceptance + the §13 suite in T16).

### Level 1: Structural / Style (immediate)

```bash
cd ~/projects/skpp
test -f README.md && echo "exists OK"

# All 8 PRD §15 sections present, as ## headings, in order.
for h in "## Why" "## Install" "## Usage" "Where skills live" "Adding a skill" \
         "How \`skpp\` finds the store" "## Constraints"; do
  grep -qF "$h" README.md && echo "section OK: $h" || echo "MISSING: $h"
done
# Expected: 7 OK lines (title is H1; the 8 §15 items = Why..Constraints; one-liner is
# the H1 subtitle). Adjust the heading text to EXACTLY what you wrote, then re-check.

# The canonical one-liner appears (in Usage, ideally in the first code block).
grep -qF 'pi --skill "$(skpp example)"' README.md && echo "one-liner OK" || echo "FAIL: no one-liner"

# The go-install SKPP_SKILLS_DIR caveat is prominent (present + named).
grep -qF 'SKPP_SKILLS_DIR' README.md && echo "go-install caveat present OK" || echo "FAIL: caveat missing"

# No marketing tell-words (mcpeepants tone gate).
grep -nEi 'blazing|seamless|powerful|game-changing|revolutionary' README.md && echo "FAIL: marketing word" || echo "tone OK"

# Expected: all OK; no marketing hits.
```

### Level 2: Content Accuracy (examples match the real binary)

```bash
cd ~/projects/skpp

# Re-run each cited command and diff against what the README claims.
./skpp example        | grep -q "/skills/example$" && echo "example OK" || echo "FAIL"
./skpp -f example     | grep -q "/skills/example/SKILL.md$" && echo "-f OK" || echo "FAIL"
./skpp --path         | grep -q "/skills$" && echo "--path OK" || echo "FAIL"
./skpp check >/dev/null 2>&1 && echo "check exits 0 OK" || echo "FAIL"

# Error contract: unknown tag -> NOTHING on stdout, exit 1.
out=$(./skpp nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "error contract OK" || echo "FAIL"
# Ensure the README states this contract (grep the prose).
grep -qiE 'nothing.*(stdout|printed)|exit 1' README.md && echo "contract documented OK" || echo "WARN: state the error contract"

# --version documented as a command, NOT a hardcoded fixed string.
grep -qF 'skpp --version' README.md && echo "version-command OK" || echo "FAIL"

# Expected: all OK. If any example in the README shows different output than the
# binary prints, FIX THE README (the binary is authoritative; do not edit code).
```

### Level 3: Markdown Rendering & Cross-References

```bash
cd ~/projects/skpp

# No raw HTML blocks (GitHub renders fine without; keeps it portable).
! grep -nE '^<[a-z]' README.md && echo "no raw HTML OK" || echo "WARN: raw HTML line (review)"

# Code fences balanced (even number of ``` lines).
fences=$(grep -c '^```' README.md); [ $((fences % 2)) -eq 0 ] && echo "fences balanced OK" || echo "FAIL: unbalanced fences"

# Conditional: completions referenced ONLY if the dir exists.
if [ -d completions ]; then
  grep -q 'completions' README.md && echo "completions-ref OK (dir exists)" || echo "WARN: completions/ exists but not mentioned"
else
  ! grep -qi 'completions/' README.md && echo "no broken completions ref OK" || echo "FAIL: README references completions/ which does not exist"
fi

# The example skill template the README points at actually exists.
grep -q 'skills/example/SKILL.md' README.md && test -f skills/example/SKILL.md && echo "template-ref OK" || echo "WARN: check the example pointer"

# Expected: all OK (or intentional WARNs reviewed and accepted).
```

### Level 4: End-to-End (a reader can install + use from the README alone)

```bash
cd ~/projects/skpp

# A cold reader follows the README's install.sh path, then runs the canonical line.
./install.sh >/dev/null 2>&1 || true
"$HOME/.local/bin/skpp" example | grep -q "/skills/example$" && echo "install.sh path works OK" || echo "FAIL"

# The go-install caveat is actionable: setting SKPP_SKILLS_DIR makes the binary
# find the store from anywhere (simulating a go install with no sibling skills/).
SKPP_SKILLS_DIR="$PWD/skills" ./skpp example | grep -q "/skills/example$" && echo "SKPP_SKILLS_DIR override works OK" || echo "FAIL"

# (If pi is present) the README's canonical pi line runs against the example skill.
command -v pi >/dev/null 2>&1 && {
  pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded" 2>&1 | head || true
} || echo "(pi not installed; skipping pi level — not required for README task)"

# Final scope check: ONLY README.md is new/modified.
git status --porcelain | grep -qvE '^\?\? README.md$' && echo "FAIL: unexpected changes" || echo "scope OK: only README.md new"
# Expected: install path works; env override works; scope clean (only README.md untracked).
```

## Final Validation Checklist

### Technical Validation

- [ ] Level 1 passed: file exists; 8 §15 `##` sections present in order; one-liner
      present; go-install `SKPP_SKILLS_DIR` caveat present; no marketing tell-words.
- [ ] Level 2 passed: every cited example matches the real `./skpp` output;
      error contract (unknown tag → empty stdout, exit 1) both holds and is
      documented; `--version` shown as a command, not a fixed string.
- [ ] Level 3 passed: balanced code fences; no raw HTML; completions referenced
      only if `completions/` exists; example-skill pointer valid.
- [ ] Level 4 passed: `./install.sh` path resolves `example`; `SKPP_SKILLS_DIR`
      override works from any cwd; `git status` shows only `README.md` new.
- [ ] (Optional) If `pi` present: the README's canonical pi line loads the
      example skill under `--no-skills`.

### Feature Validation

- [ ] All Success Criteria in "What" met (10 checkboxes).
- [ ] The literal `pi --skill "$(skpp example)"` is in the first Usage code block.
- [ ] A reader who knows nothing of this repo can install + add a skill + use the
      one-liner from the README alone ("No Prior Knowledge" test).
- [ ] `go install` users are not silently broken (caveat is prominent).
- [ ] The §17 constraints are user-visible (manifest-free; never auto-discovered;
      loads only via `--skill`; skpp only prints paths).
- [ ] Error contract documented so `pi --skill "$(skpp badtag)"` failure mode is
      understood.

### Code Quality Validation

- [ ] Follows mcpeepants README tone (plain, example-driven, lead with one-liner)
      while carrying the fuller §15 structure.
- [ ] File placement: repo root (PRD §5).
- [ ] Markdown renders cleanly on GitHub (fenced ```bash, blockquote callout,
      `key - description` bullets).
- [ ] No em-dash emphasis, no hedging/formulaic transitions, no marketing
      adjectives (aligns with available write-tech-docs rules; optional skill).
- [ ] No invented output; no broken file references; no conflation of
      `SKPP_INSTALL_BIN` (install-time) and `SKPP_SKILLS_DIR` (runtime).

### Documentation & Deployment

- [ ] README is self-sufficient (a reader need not open PRD.md or source).
- [ ] All install paths documented; the recommended one (`./install.sh`) is clear.
- [ ] The "Adding a skill" section gives a copy-pasteable frontmatter block and
      points at `skills/example/SKILL.md`.
- [ ] No new env vars invented; only the spec'd `SKPP_SKILLS_DIR` (runtime) and
      `SKPP_INSTALL_BIN` (install-time, from install.sh) are mentioned.

---

## Anti-Patterns to Avoid

- ❌ **Do NOT invent CLI output.** Re-run the commands; the binary is
  authoritative. A README that shows fake output breaks trust instantly.
- ❌ Do NOT bury the `go install` → `SKPP_SKILLS_DIR` caveat. It is the #1
  install footgun and PRD §12.2 demands it be prominent.
- ❌ Do NOT conflate `SKPP_INSTALL_BIN` (install.sh target) with `SKPP_SKILLS_DIR`
  (runtime skills dir). Different stages, different sections.
- ❌ Do NOT hardcode a version string. `--version` output is dynamic (git describe).
- ❌ Do NOT document `completions/` commands unless that directory exists
  (T15 may not be done). Broken references are worse than omission.
- ❌ Do NOT use marketing voice ("blazing fast", "powerful", "seamless") or
  formulaic AI transitions. Match mcpeepants' plain, factual tone.
- ❌ Do NOT add a "Features:" h2, badges, or emoji-laden headings — mcpeepants
  has none, and PRD §15 fixes the section set.
- ❌ Do NOT imply the tag equals the frontmatter `name`. The canonical tag is the
  skill dir's path relative to `skills/`; `name` is only one resolution fallback.
- ❌ Do NOT enumerate pi's discovery locations only in prose — NAME them in the
  Constraints section (the "never auto-discovered" claim needs the list to be
  credible and actionable).
- ❌ Do NOT change any code, `install.sh`, `skills/`, `go.mod`, or `.gitignore`
  to make the README "true." The README documents reality; if reality is wrong,
  flag it (this task is docs-only).

---

## Confidence Score

**9 / 10.** The deliverable is a single Markdown file whose structure is fully
fixed by PRD §15 (8 ordered sections), whose tone model (mcpeepants README) is
quoted in full in `research/verified_facts.md`, and whose every code example is
captured verbatim from the already-built binary (re-runnable in §3 of the
research notes). The binding content is non-negotiable and externally specified;
the implementer's only latitude is wording within the tone rules and the
conditional completions pointer. The one residual risk is drift between the
documented examples and a future binary change — mitigated by the Level-2
"re-run and diff" gate. No Go knowledge is required; no source must be read to
WRITE the doc (only to verify examples afterward).
