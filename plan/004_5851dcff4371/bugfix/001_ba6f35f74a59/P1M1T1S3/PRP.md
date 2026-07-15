# PRP — P1.M1.T1.S3: Add `--shell` value directive + flag advertisement to the fish completion file

> **Subtask:** P1.M1.T1.S3 — the fish third of P1.M1.T1 (Issue 1: `--shell` value completion offers skill tags instead of `bash zsh fish`). Mirrors the committed S1 (bash) and the in-flight S2 (zsh) for the fish file. After S3 lands, all three completion files carry `--shell` → PRD §14.4 lockstep fully restored.
> **Scope boundary:** Edits ONLY `completions/skilldozer.fish` (the fish completion script, embedded via `//go:embed` at main.go:60). Does NOT touch the bash file (S1, committed) or the zsh file (S2, working tree); does NOT touch any `.go` file (the `//go:embed` picks up the edit automatically); does NOT change `usageText` (it already documents `--shell`); does NOT edit the README (Mode B sweep is P1.M3.T1).

---

## Goal

**Feature Goal**: Make `skilldozer --shell <TAB>` offer exactly `bash zsh fish` (PRD §14.2's fixed enum, nothing else) instead of skill tags, and make `--shell` discoverable via `skilldozer --<TAB>` — in the **fish** completion script. This is the fish-specific slice of Issue 1's three-file lockstep fix.

**Deliverable**: Edits to `completions/skilldozer.fish` only (no new files):
1. **Value directive** (after the `--store` line, ~line 50): add `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"`, preceded by an explanatory comment block (mirrors the file's existing per-flag comment style).
2. **LOCKSTEP header note** (after the last header line, ~line 15): append the `--shell` note (verbatim mirror of S1's bash header / S2's zsh header).

**Success Definition**: After `go build`, `TestEmbeddedCompletionsMatchOnDisk` passes (embedded fish bytes == on-disk file); no existing test regresses; in real fish, `complete -C "skilldozer --shell "` offers `bash zsh fish` (not skill tags) and `--shell` is advertised; `go.mod`/`go.sum` unchanged; no `.go` file edited.

---

## User Persona (if applicable)

**Target User**: fish users who tab-complete `skilldozer` invocations (especially the canonical `skilldozer --completions --shell fish | source` install idiom).

**Use Case**: A user types `skilldozer --shell ` and tabs to pick the shell for which to emit completions.

**User Journey**: `skilldozer --shell ` → `<TAB>` → fish offers `bash`, `zsh`, `fish` → user picks one → continues typing the rest of the command.

**Pain Points Addressed**: Today `<TAB>` after `--shell` offers skill tags (the opposite of §14.2's "nothing else"), because the fish file has no `--shell` directive — the dynamic-tag directive at the bottom fires for the unmatched positional and floods the menu with skills.

---

## Why

- **PRD §14.2**: "`--shell` takes a fixed enum (`bash`/`zsh`/`fish`); offer those three words, nothing else." Today the fish file has no `complete -c skilldozer -l shell` directive, so after `--shell ` fish falls through to the dynamic-tag directive → offers skill tags. A spec deviation.
- **PRD §14.4 lockstep**: completion files are frozen to `main.go parseArgs()`, which already accepts `--shell`. The fish file is missing it.
- **Decision D7** (decisions.md): `--shell` is a real, documented flag (in `usageText` OPTIONS, used in the canonical install idiom). Add it to the advertised flag list in all three files for discoverability. (PRD §14.6's 13-flag table omits `--shell` — a noted tension; D7 resolves it in favor of consistency.)
- **Lockstep with S1/S2**: the bash file already has `--shell` (S1, committed) and the zsh file gets it (S2). This PRP brings the fish file into the same state so the §14.4 "all three identical" invariant is restored.

---

## What

`completions/skilldozer.fish` gains a `--shell` directive that is the THIRD value-routing pattern in the file:

- **Value directive**: `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"`. The `-x` (exclusive) flag means the option requires a value AND suppresses file completion, so ONLY the `-a` enum is offered — the deliberate **inverse** of `--store`'s `-r` (which routes to file/dir completion). This is exactly §14.2's "offer those three words, nothing else".
- **Advertisement**: because the `--shell` directive is a top-level `complete` registration, fish automatically offers `--shell` on `skilldozer --<TAB>` (D7). No separate advertisement step — fish's flat-directive model means each `complete -l <flag>` IS the advertisement.

No behavior change for any other flag; the global `complete -c skilldozer -f` (line 18), the `--search`/`--store`/`--init` routing, the dynamic-tag default, and the long-form-only policy are all untouched.

### Success Criteria

- [ ] a `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"` directive exists immediately after the `--store` directive (line 50)
- [ ] an explanatory comment block above the directive documents `-x` (exclusive, no files) + `-a "bash zsh fish"` as the third value-routing pattern (mirrors the `--search`/`--store` comment style)
- [ ] the LOCKSTEP header (lines 9-15) ends with the `--shell` note (verbatim mirror of S1's bash header / S2's zsh header)
- [ ] `TestEmbeddedCompletionsMatchOnDisk` passes (embedded fish == on-disk file)
- [ ] in real fish, `complete -C "skilldozer --shell "` offers `bash zsh fish` (not skill tags); `skilldozer --<TAB>` offers `--shell`
- [ ] no existing test regresses; `go.mod`/`go.sum` unchanged; no `.go` file edited

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact current text of both edit sites (the `--store` directive at line 50 and the LOCKSTEP header's last line at line 15), the exact target additions (verbatim directive from the contract + a verbatim header note mirrored from S1/S2), the `-x`/`-a`/`-r` routing mechanic (why `-x -a` and NOT `-r` and NOT bare), the embed/rebuild mechanics, the automated gate (TestEmbeddedCompletionsMatchOnDisk), the manual fish repro, and the scope boundary (fish only) are all specified with line numbers and exact before/after text. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the parallel sibling PRPs (S1 bash, S2 zsh) — the pattern to mirror for fish
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M1T1S1/PRP.md
  why: "S1 (bash) is COMMITTED. Its LOCKSTEP header note (lines 17-18) is the verbatim template S3 mirrors. S1 also establishes the three-pattern value-routing model (--search nothing / --store files / --shell enum) that the fish comment block documents."
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M1T1S2/PRP.md
  why: "S2 (zsh) is being implemented in parallel; its header note append mirrors S1's verbatim — S3 does the SAME for fish. Read it to confirm the exact header-note wording to copy so all three headers stay byte-identical."

# MUST READ — the authoritative current text + exact old→new strings (verified against live HEAD 147b177)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M1T1S3/research/verified_facts.md
  why: "§0 the lockstep state (S1 committed, S2 in working tree, fish CLEAN). §1 the exact current text of the fish file (header 1-15, --store @ :50, dynamic tags @ :52-55). §2 the exact edits (directive + header note). §3 WHY -x -a is correct (the -r/-x/-a routing table). §4 the quote-style gotcha (double quotes on -a, single on -d). §5 the embed wiring. §6 the three gate tests. §7 scope. §8 the repro."
  critical: "§3 (the -x/-a/-r routing — the ONE fish-specific subtlety) and §4 (the -a double-quote vs -d single-quote distinction) are the two facts that prevent the most likely implementation errors."

# MUST READ — the issue writeup + the decision
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/issue_analysis.md
  why: "§Issue 1 is the authoritative bug writeup: the fish repro (`complete -C \"skilldozer --shell \"` → skill tags), confirmation --shell is MISSING from all three files, and the per-file fix prescription. The fish prescription is verbatim: `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a \"bash zsh fish\"`."
  section: "Issue 1 (--shell value completion offers skill tags) → Fix Surface #3 (fish)"
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/decisions.md
  why: "§D7 is the decision to ADD --shell to the advertised flag list (not just value routing) — resolves the PRD §14.6 (13-flag table) vs §14.2 (value enum) tension in favor of consistency/discoverability."
  section: "D7 (Issue 1 — --shell Added to Flag Advertisement List)"

# MUST READ — the edit site (the ONLY file S3 touches)
- file: completions/skilldozer.fish
  why: "THE edit site. Flat list of top-level `complete` directives (NOT a function, NOT an array). GLOBAL `complete -c skilldozer -f` @ :18 (no-files — load-bearing context for the -r/-x distinction). --store directive @ :50 (the insertion anchor — --shell goes right after it). Dynamic tags directive @ :52-55 (the default positional handler — UNTOUCHED). LOCKSTEP header @ :9-15 (append the --shell note after line 15)."
  pattern: "Mirror the existing value-taking flags' 'comment block above the directive' style: --search (lines 36-43, explains NO -r), --store (45-50, explains -r). --shell adds the THIRD pattern: -x -a (closed enum, no files). Match that explanatory-comment style."
  gotcha: "The directive's `-a \"bash zsh fish\"` uses DOUBLE quotes (fish tokenizes the space-separated word list); the `-d '...'` description uses SINGLE quotes (matches every other -d in the file). Do NOT normalize them to match — the two arguments legitimately differ (see verified_facts §4)."

- file: main.go
  why: "The //go:embed wiring (NO edit — confirms it picks up the file change). :60 //go:embed completions/skilldozer.fish; :61 var fishCompletion string; :1113-1121 completionScript(\"fish\") returns fishCompletion. NO eval-safe wrapper for fish (unlike zsh's zshEvalScript) — every directive is emitted verbatim, so there is nothing to strip and no strip-interaction to worry about."
  pattern: "Do NOT touch main.go. The embed is the mechanism, not an edit site."
  gotcha: "A PRE-BUILT binary holds the OLD embedded bytes. Always rebuild (go build / go test) before behavioral testing."

- file: main_test.go
  why: "The automated gates. TestEmbeddedCompletionsMatchOnDisk @ :2995 reads the on-disk file via os.ReadFile and asserts completionScript(\"fish\") == on-disk (byte identity, PRD §14.6) — go test re-embeds at compile, so it passes automatically after the edit. TestCompletionScriptMapping @ :2957 asserts the `# Fish completion for skilldozer.` header (line 1 — untouched). TestRunCompletionFishScript @ :3035 asserts `run([--completions,--shell,fish])` stdout contains `complete -c skilldozer` (the new --shell directive ALSO starts with that marker — still true). No test asserts the flag-directive set, so no test breaks."
  pattern: "The embed-match test is the automated regression gate; the fish repro (Level 3) is the behavioral gate (requires real fish)."

- url: https://fishshell.com/docs/current/cmds/complete.html
  why: "The authoritative fish `complete` builtin reference. Documents `-r/--require-parameter` (value completed via default/file completion), `-x/--exclusive` (value required, file completion suppressed — ONLY the -a list is offered), and `-a/--arguments` (the candidate values for the slot). This is the spec behind why --shell uses `-x -a \"bash zsh fish\"` and NOT `-r`."
  section: "the options table for -r, -x, -a"
- url: (PRD §14.2 / §14.4 — in PRD.md, READ-ONLY)
  why: "§14.2: --shell takes the fixed enum bash/zsh/fish, offer those three words nothing else. §14.4: completion files frozen to parseArgs (lockstep). Do NOT edit PRD.md."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && git rev-parse --short HEAD
147b177
$ git status --short completions/
 M completions/_skilldozer        # S2 (zsh) — working tree; assume DONE when S3 runs
                                  # completions/skilldozer.bash — S1, COMMITTED
                                  # completions/skilldozer.fish — CLEAN (S3's input)
$ go build ./... && echo BUILD_OK ; go vet ./... && echo VET_OK
BUILD_OK / VET_OK
# completions/skilldozer.fish: flat list of top-level `complete` directives. GLOBAL -f @ :18.
#   --store directive @ :50 (insertion anchor). Dynamic tags @ :52-55 (UNTOUCHED). Header @ :9-15.
#   Embedded via //go:embed @ main.go:60 → var fishCompletion (main.go:61).
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep).
# No new files. This subtask edits completions/skilldozer.fish ONLY (no .go file).
```

### Desired Codebase tree with files to be changed

```bash
completions/skilldozer.fish   # MODIFY — add --shell directive (+ explanatory comment) + LOCKSTEP header note
# main.go / main_test.go / go.mod / go.sum — UNCHANGED (//go:embed picks up the file edit on rebuild)
# completions/skilldozer.bash (S1, committed) / completions/_skilldozer (S2, working tree) — UNCHANGED here
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `completions/skilldozer.fish` | `--shell` value directive (routing + advertisement) + explanatory comment + LOCKSTEP note | PRD §14.2 + decision D7 |

### Known Gotchas of our codebase & Library Quirks

```bash
# GOTCHA #1 — A pre-built binary holds STALE embedded bytes. //go:embed reads the file
# at COMPILE time. If you edit completions/skilldozer.fish and run an already-built
# ./skilldozer, --completions emits the OLD script (no --shell). Always rebuild
# (go build, or just go test which compiles) before behavioral testing.

# GOTCHA #2 — The automated gate is byte-identity, not behavior. TestEmbeddedCompletionsMatchOnDisk
# asserts completionScript("fish") == the on-disk file. It passes as long as you rebuild
# after editing. It does NOT assert the --shell directive exists — so the manual fish repro
# (Level 3) is the behavioral gate that proves the routing actually works.

# GOTCHA #3 — Rebuild for the embed, but DO NOT edit any .go file. The //go:embed directive
# at main.go:60 is the mechanism; it picks up the file change automatically. Adding --shell
# to the fish file requires ZERO Go changes. (parseArgs already accepts --shell — that's
# why §14.4 says completions are frozen to parseArgs.)

# GOTCHA #4 — ONE directive does BOTH value routing AND advertisement. Fish's completion
# model is a FLAT list of top-level `complete` registrations. A `complete -c skilldozer -l shell`
# directive (a) routes `--shell`'s value slot (via -x -a) AND (b) makes `--shell` appear on
# `skilldozer --<TAB>` (the directive IS the advertisement). There is no separate "flag list"
# to update (that's a bash thing) and no separate "advertisement" mechanism (unlike zsh's
# _arguments array). So S3's ONLY functional edit is the one directive line (plus comments).

# GOTCHA #5 — Use `-x -a "bash zsh fish"`, NOT `-r` and NOT a bare flag. This is the ONE
# fish-specific subtlety. The file's OWN comment (lines 45-49) documents that `-r` BYPASSES
# the global `-f` (line 18) and offers FILES. `--shell` must NOT offer files — it must offer
# ONLY the enum. `-x` (exclusive) = require-a-value + suppress-files; combined with `-a "bash zsh
# fish"` it offers exactly the enum and nothing else. Using `-r` here would be a bug (offers files
# for --shell); using no flag (like --search) would be a bug (global -f → offers nothing → the enum
# never appears). ONLY `-x -a` is correct. (See verified_facts §3 for the 3-pattern table.)

# GOTCHA #6 — The `-a "bash zsh fish"` argument uses DOUBLE quotes; the `-d '...'` description
# uses SINGLE quotes. Do NOT "normalize" them to match. Fish tokenizes the space-separated word
# list inside the double-quoted -a into three candidates (bash, zsh, fish); single quotes on the
# -d description match every other -d in the file (descriptions are single-quoted throughout).
# Both quote styles are intentional and correct for their respective arguments.

# GOTCHA #7 — There is NO eval-safe wrapper / NO trailing self-call in the fish file (unlike
# zsh's zshEvalScript that strips the trailing `_skilldozer "$@"`). Fish sources the
# `complete` directives directly. So there is no strip interaction to worry about — every
# directive is emitted verbatim by `--completions --shell fish`. Do NOT add any wrapper.

# GOTCHA #8 — The enum order is "bash zsh fish" (PRD §14.2 + the contract + S1's bash file +
# S2's zsh file). Consistent order across all three shells avoids user surprise. Do NOT reorder.

# GOTCHA #9 — Placement is AFTER the --store directive (contract LOGIC 3; --store is @ :50).
# The contract is explicit about placement after --store. This also groups the three
# value-taking flags' comment blocks together (--search, --store, --shell) for readability.
# Do NOT group --shell with --check/--init/--completions.

# GOTCHA #10 — SCOPE: edit ONLY the fish file. bash (completions/skilldozer.bash) is S1
# (committed); zsh (completions/_skilldozer) is S2 (working tree). There is NO cross-file
# lockstep test (TestEmbeddedCompletionsMatchOnDisk compares each embed to its OWN file), so
# editing only fish breaks no test. Do NOT edit bash/zsh here.

# GOTCHA #11 — Do NOT change usageText. It already lists --shell in OPTIONS (D7: "--shell is
# a real, documented flag in usageText OPTIONS"). The gap is ONLY the completion file.

# GOTCHA #12 — No deps change. No .go file is edited, so go.mod/go.sum are byte-for-byte
# identical. The sole edited file is a shell data asset.
```

---

## Implementation Blueprint

### Data models and structure

None. This subtask edits a fish completion script (a data asset embedded into the Go binary). No Go types, no signatures change.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: ADD the --shell value directive + explanatory comment (completions/skilldozer.fish, after line 50)
  - EDIT: AFTER the `complete -c skilldozer -l store ... -r` directive (line 50) and BEFORE the
    blank line (51) / the dynamic-tags section, ADD the comment block + the directive:
        # --shell <name> (PRD §14.2): Force a shell for completion. The value is a FIXED
        # enum (bash/zsh/fish), so use `-x` (exclusive: require a value, NO file
        # completion) + `-a "bash zsh fish"` (the three candidates). This is the THIRD
        # value-routing pattern: --search = nothing (no flag), --store/--init = files
        # (-r), --shell = closed enum (-x -a). --shell is advertised (decision D7).
        complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"
  - WHY: the directive routes --shell's value slot to the fixed enum (§14.2) via -x -a (GOTCHA #5);
    its presence as a top-level complete registration advertises the flag on --<TAB> (GOTCHA #4).
  - DIRECTIVE LINE IS VERBATIM from the contract + issue_analysis. Do not alter -x / -a / the enum.
  - EXACT oldText/newText in Implementation Patterns below.

Task 2: APPEND the --shell note to the LOCKSTEP header (completions/skilldozer.fish, after line 15)
  - EDIT: AFTER the header line `# belongs to skill tags — a bare <tab> shows skills, never commands.`
    (line 15, the last header line), APPEND (verbatim mirror of S1's bash header / S2's zsh header):
        # --shell's value completes to the bash/zsh/fish enum (§14.2); --shell is
        # advertised (D7) since it is a real, documented flag in usageText OPTIONS.
  - WHY: contract DOCS §5 ("Verify the flag list comment mentions --shell") + keeps the three
    files' headers in lockstep (byte-identical wording across bash/zsh/fish).

Task 3: VERIFY — embed-match test + behavioral repro + no regression + scope discipline
  - COMMAND: go build ./...                                  (exit 0; re-embeds the edited file)
  - COMMAND: go test -run TestEmbeddedCompletionsMatchOnDisk -v ./...   (PASS — embedded fish == on-disk)
  - COMMAND: go test ./...                                   (no regression — fish content tests still pass)
  - MANUAL (requires real fish): the repro (Level 3) → complete -C "skilldozer --shell " offers bash zsh fish
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"   (GOTCHA #12)
  - COMMAND: git diff --stat -- '*.go'                       (MUST be empty — no .go file edited)
  - COMMAND: git diff --name-only completions/               (MUST list completions/skilldozer.fish;
                                                              do NOT also touch bash/zsh — GOTCHA #10)
```

### Implementation Patterns & Key Details

```fish
# Task 1 — the --shell directive (exact oldText → newText). Insert between the --store
# directive and the blank line that precedes the dynamic-tags section. The comment mirrors
# the existing per-flag comment style (--search @ 36-42, --store @ 45-49).

#   OLD (lines 50-52):
complete -c skilldozer -l store -d 'Non-interactive store path for init' -r

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per

#   NEW (insert the --shell block between --store and the blank line):
complete -c skilldozer -l store -d 'Non-interactive store path for init' -r
# --shell <name> (PRD §14.2): Force a shell for completion. The value is a FIXED
# enum (bash/zsh/fish), so use `-x` (exclusive: require a value, NO file
# completion) + `-a "bash zsh fish"` (the three candidates). This is the THIRD
# value-routing pattern: --search = nothing (no flag), --store/--init = files
# (-r), --shell = closed enum (-x -a). --shell is advertised (decision D7).
complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per

# Task 2 — the LOCKSTEP header note (exact oldText → newText). Append after line 15.

#   OLD (the last header line, line 15):
# belongs to skill tags — a bare <tab> shows skills, never commands.
#   NEW (append two lines):
# belongs to skill tags — a bare <tab> shows skills, never commands.
# --shell's value completes to the bash/zsh/fish enum (§14.2); --shell is
# advertised (D7) since it is a real, documented flag in usageText OPTIONS.
```

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **One directive does both routing + advertisement (no separate mechanism).** Unlike bash (which needs a `case "$prev"` branch AND a `compgen -W` flag-list token), fish's completion model is a flat list of top-level `complete` registrations: the `complete -c skilldozer -l shell -x -a "..."` directive (a) routes the value slot via `-x -a`, and (b) makes `--shell` appear on `--<TAB>` simply because it is registered. So S3's ONLY functional edit is the one directive line — there is no separate "flag list" to update (that's a bash thing) and no `_arguments` array (that's zsh). This is the fish-specific simplification of S1's two-edit bash approach.

2. **`-x -a` (not `-r`, not bare) — the one fish-specific subtlety.** The file's own `--store` comment documents that `-r` bypasses the global `-f` and offers FILES. `--shell` must offer ONLY the enum. `-x` (exclusive) = require-a-value + suppress-files; with `-a "bash zsh fish"` it offers exactly the three shells. Using `-r` would offer files (bug); using no flag would offer nothing via the global `-f` (bug). Only `-x -a` is correct. This is the deliberate INVERSE of `--store`'s `-r` and is exactly what the contract + issue_analysis prescribe.

3. **Header note mirrors S1/S2 verbatim.** S1's bash header (lines 17-18) and S2's zsh header both say exactly *"--shell's value completes to the bash/zsh/fish enum (§14.2); --shell is advertised (D7) since it is a real, documented flag in usageText OPTIONS."* Copy it verbatim into the fish header so the three files' headers stay byte-identical (wording divergence would be pointless review noise).

4. **Placement after `--store` (contract-pinned).** The contract LOGIC says place the directive after the `--store` line. This also groups the three value-taking flags' comment blocks together (--search, --store, --shell) for readability. Do not place `--shell` near `--check`/`--init`/`--completions` even though `--init` is also value-taking — the contract is explicit.

### Integration Points

```yaml
EMBED (the mechanism — NO edit):
  - main.go:60 //go:embed completions/skilldozer.fish  →  var fishCompletion string (main.go:61)
  - completionScript("fish") returns fishCompletion (main.go:1120); runCompletion emits it for --completions.
  - Editing the on-disk file + go build/go test re-embeds the new bytes automatically.

NO EVAL-SAFE WRAPPER (unlike zsh — NO edit, NO interaction):
  - Fish sources the `complete` directives directly. There is no self-call to strip
    (zsh's zshEvalScript is zsh-only). Every directive is emitted verbatim.

TESTS (unchanged; they gate the change):
  - TestEmbeddedCompletionsMatchOnDisk (main_test.go:2995): embedded fish == on-disk. PASS after rebuild.
  - TestCompletionScriptMapping (2957): asserts the `# Fish completion for skilldozer.` header — line 1 untouched. Still passes.
  - TestRunCompletionFishScript (3035): asserts `run([--completions,--shell,fish])` stdout contains
    `complete -c skilldozer` — the new --shell directive ALSO starts with that marker. Still passes.

NO DATABASE / NO CONFIG / NO ROUTES / NO GO SOURCE:
  - This subtask edits exactly one shell data asset. No parseArgs, no run(), no usageText, no main.go.
```

---

## Validation Loop

### Level 1: Shell sanity (immediate, after the edits)

```bash
cd /home/dustin/projects/skilldozer

# fish syntax-check the edited file (catches a broken complete directive / unbalanced quotes):
if command -v fish >/dev/null 2>&1; then
  fish -n completions/skilldozer.fish && echo "fish -n OK" || echo "FAIL: fish syntax error"
else
  echo "fish not installed — skipping fish -n (the embed-match test still gates byte-identity)"
fi
# Expected: "fish -n OK" (or the skip message if fish is absent).

# Confirm the two edits are present:
grep -F -- 'complete -c skilldozer -l shell -d ' completions/skilldozer.fish   # Expected: 1 hit (the new directive)
grep -c -- '--shell' completions/skilldozer.fish                              # Expected: >=3 (directive + comment + header note)
grep -n -- 'advertised (D7)' completions/skilldozer.fish                      # Expected: 1 hit (header note)
grep -F -- '-x -a "bash zsh fish"' completions/skilldozer.fish                # Expected: 1 hit (the enum routing)
# Expected: fish -n clean (or skipped); the greps find the additions.
```

### Level 2: The embed-match gate (the automated regression check)

```bash
cd /home/dustin/projects/skilldozer

go build ./...     ; echo "build exit $?"    # Expected: 0 (re-embeds the edited file)
go test -run TestEmbeddedCompletionsMatchOnDisk -v ./...
# Expected: PASS — completionScript("fish") == on-disk completions/skilldozer.fish.

# Whole module: the fish content tests + everything else still green:
go test ./... ; echo "test exit $?"          # Expected: 0 (no regression)
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
git diff --stat -- '*.go'                    # Expected: EMPTY (no .go file edited)
git diff --name-only completions/            # Expected: completions/skilldozer.fish (S2's zsh is already in the working tree — do NOT touch it)
# Expected: build/test exit 0; deps unchanged; no .go diff; the fish file newly changed by S3.
```

### Level 3: The behavioral repro (the actual contract — Issue 1 fixed; requires real fish)

```bash
cd /home/dustin/projects/skilldozer
command -v fish >/dev/null 2>&1 || { echo "SKIP: fish not installed (the embed-match test is the fallback gate)"; exit 0; }
go build -o /tmp/sdz . || { echo "FAIL: build"; exit 1; }

# Deterministic byte gate: the rebuilt binary's emitted script carries the --shell directive.
/tmp/sdz --completions --shell fish | grep -F -- 'complete -c skilldozer -l shell -d '
# Expected: the --shell directive line with `-x -a "bash zsh fish"`.

# Behavioral (the contract repro from PRD §Issue 1): after sourcing, `complete -C` offers the enum.
fish -c '/tmp/sdz --completions --shell fish | source; for l in (complete -C "skilldozer --shell "); echo $l; end'
# Expected (post-fix): three lines — bash, zsh, fish (each possibly annotated with the description).
# Pre-fix: offered skill tags (example, foo, writing/reddit …) — the exact bug being fixed.

# Advertisement: --shell is offered on skilldozer --<TAB> (the directive registers it).
fish -c '/tmp/sdz --completions --shell fish | source; for l in (complete -C "skilldozer --"); echo $l; end' | grep -qx -- '--shell' \
  && echo "--shell advertised OK" || echo "FAIL: --shell not advertised"

# Control: bare tag completion still works (the dynamic-tags directive is UNTOUCHED).
fish -c '/tmp/sdz --completions --shell fish | source; for l in (complete -C "skilldozer "); echo $l; end' | head -1
# Expected: a skill tag from the store (proves the --shell directive didn't break the default).

rm -f /tmp/sdz
# Expected: the --shell directive present in the emitted script; complete -C offers bash/zsh/fish;
#           --shell advertised; tag completion intact.
```

> **Note on Level 3:** the deterministic proof is the grep on the EMITTED script
> (`/tmp/sdz --completions --shell fish` output) — it confirms the rebuilt binary embeds the
> `--shell` directive with `-x -a "bash zsh fish"`. The `complete -C` check is the behavioral
> proof (requires real fish); it confirms the routing actually offers the enum, not skill tags.
> (`fish -n` in Level 1 confirms the file parses.)

### Level 4: Lockstep-awareness check (scope discipline)

```bash
cd /home/dustin/projects/skilldozer

# S3 edits ONLY the fish file. Confirm S3 did NOT touch bash/zsh and did NOT edit any .go file.
git diff --name-only completions/skilldozer.fish   # Expected: completions/skilldozer.fish (S3's sole change)
git diff --stat -- '*.go'                           # Expected: EMPTY
# (completions/_skilldozer shows as modified in the working tree, but that is S2's change, NOT S3's.
#  S3 must NOT add any further edit to _skilldozer or to skilldozer.bash.)

# The embed-match test passes for all three shells (fish now matches the edited file;
# bash matches S1's committed edit; zsh matches S2's working-tree edit):
go test -run TestEmbeddedCompletionsMatchOnDisk -v ./... 2>&1 | grep -E 'zsh|fish|bash|PASS|FAIL'
# Expected: PASS (the test loop covers all three; no per-shell breakdown in output).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `fish -n` clean (or skipped if fish absent); greps confirm the `--shell` directive, the explanatory comment, the header note, and the `-x -a "bash zsh fish"` enum routing
- [ ] Level 2 PASS — `go build` exit 0; `TestEmbeddedCompletionsMatchOnDisk` PASS; `go test ./...` exit 0; `git diff go.mod go.sum` → "deps unchanged"; no `.go` diff
- [ ] Level 3 PASS — the emitted `--completions --shell fish` script contains the `--shell` directive with `-x -a "bash zsh fish"`; `complete -C "skilldozer --shell "` offers bash/zsh/fish (not skill tags); `--shell` advertised; tag completion intact
- [ ] Level 4 PASS — only `completions/skilldozer.fish` newly changed by S3; no bash/zsh touch by S3; no `.go` touch; embed-match passes for all three shells

### Feature Validation
- [ ] a `complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"` directive exists immediately after the `--store` directive
- [ ] an explanatory comment block above the directive documents `-x` (exclusive, no files) + `-a "bash zsh fish"` as the third value-routing pattern
- [ ] the LOCKSTEP header ends with the verbatim S1/S2 mirror `--shell` note
- [ ] the enum is exactly `bash zsh fish` (order per §14.2)
- [ ] `--shell` is advertised (the directive registers it → offered on `--<TAB>`)

### Code Quality / Convention Validation
- [ ] the `--shell` directive mirrors the existing value-taking flags' shape (`complete -c skilldozer -l <flag> -d '...'`) plus the `-x -a` enum routing
- [ ] the explanatory comment mirrors the existing `--search` / `--store` comment-block style (comment above the directive)
- [ ] no Go file edited (the `//go:embed` picks up the change); `go.mod`/`go.sum` byte-for-byte identical
- [ ] minimal diff (one directive line + one comment block + one header note)

### Scope Discipline
- [ ] Did NOT touch `completions/skilldozer.bash` (S1 — committed) or `completions/_skilldozer` (S2 — working tree)
- [ ] Did NOT edit any `.go` file (main.go //go:embed, parseArgs, run(), usageText all unchanged)
- [ ] Did NOT touch the dynamic-tags directive (lines 52-55) or the global `complete -c skilldozer -f` (line 18)
- [ ] Did NOT edit the README (Mode B sweep is P1.M3.T1)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't skip the rebuild.** A pre-built binary holds stale embedded bytes. `go build` (or `go test`) re-embeds; an already-built `./skilldozer` does not.
- ❌ **Don't use `-r` for `--shell`.** `-r` bypasses the global `-f` and offers FILES (the file's own --store comment documents this). `--shell` must offer ONLY the enum → use `-x -a "bash zsh fish"`. (GOTCHA #5 / verified_facts §3.)
- ❌ **Don't use a bare directive (no `-x`, like `--search`) for `--shell`.** The global `-f` would then apply to the value slot → offer nothing → the enum never appears. Only `-x -a` offers the enum. (GOTCHA #5.)
- ❌ **Don't add a separate "advertisement" mechanism.** Fish's model is a flat list of top-level `complete` registrations: one directive does BOTH value routing AND advertisement. There is no separate flag list (that's bash) and no `_arguments` array (that's zsh). (GOTCHA #4.)
- ❌ **Don't edit any `.go` file.** The `//go:embed` is the mechanism; it picks up the file change automatically. Adding `--shell` to the fish file requires zero Go changes (parseArgs already accepts `--shell`).
- ❌ **Don't "normalize" the quote styles.** `-a "bash zsh fish"` uses DOUBLE quotes (fish tokenizes the space-separated word list); `-d '...'` uses SINGLE quotes (matches every other `-d` in the file). Both are intentional and correct. (GOTCHA #6 / verified_facts §4.)
- ❌ **Don't reorder the enum.** Use `bash zsh fish` (PRD §14.2 + the contract + S1's bash file + S2's zsh file). A different order isn't wrong functionally but diverges from the spec and the other shells.
- ❌ **Don't place the directive anywhere but after `--store`.** The contract pins placement (line 50); it also groups the three value-taking flags' comment blocks together. Do not place it near `--check`/`--init`/`--completions`.
- ❌ **Don't add an eval-safe wrapper or trailing self-call.** Unlike zsh's `zshEvalScript`, fish has NO wrapper — it sources the `complete` directives directly. (GOTCHA #7.)
- ❌ **Don't edit bash or zsh here.** bash is S1 (committed); zsh is S2 (working tree). There is no cross-file lockstep test, so editing only fish breaks nothing. (GOTCHA #10.)
- ❌ **Don't change usageText.** It already documents `--shell` (D7). The gap is only the completion file.
- ❌ **Don't add deps.** No `.go` file is edited; the sole change is a shell data asset.

---

## Confidence Score

**9.5/10** — Every edit is pinned to the exact current (HEAD `147b177`) text with before/after blocks: the `--shell` directive (verbatim from the contract + issue_analysis, mirroring the existing value-taking flags' `complete -c skilldozer -l <flag> -d '...'` shape plus the `-x -a` enum routing), the explanatory comment block (mirroring the `--search`/`--store` comment-above-directive style), and the LOCKSTEP header note (verbatim mirror of S1's bash header / S2's zsh header). The two subtleties that matter most are proven in `research/verified_facts.md`: (§3) `-x -a` is the ONLY correct form (not `-r`, which offers files; not bare, which offers nothing) — the file's own `--store` comment documents the `-r` semantics that make this the deliberate inverse, and (§4) the `-a` double-quote vs `-d` single-quote distinction is intentional, not an inconsistency to "fix". The embed/rebuild mechanic is confirmed (TestEmbeddedCompletionsMatchOnDisk gates byte-identity; no eval-safe wrapper exists for fish so there is no strip interaction), and the three gate tests are each shown to pass (the header marker is on line 1, untouched; the `complete -c skilldozer` marker is still present and now matches more). No existing test asserts the flag-directive set so nothing regresses. The 0.5 reservation is the Level 3 behavioral proof: the deterministic grep on the rebuilt binary's emitted script is rock-solid, but the `complete -C` behavioral check requires real fish to be installed (when absent, the embed-match test is the fallback gate). That combination is reliable and sufficient to prove the fix.
