# PRP — P1.M2.T3.S1: Clarify version string documentation in README (Issue 5)

> **Subtask:** A single-line, documentation-only fix. The README.md `## Usage` section's ` ```bash ` example block contains the comment `# Version is the git-describe value (dynamic, not a fixed string)` (README.md:136). That is only true for the `./install.sh` build path (which injects `-ldflags "-X main.version=$(git describe ...)"` at install.sh:40). The README's own "From source" path C (`go build -o skilldozer .`) and "go install" path B do NOT inject ldflags, so they yield `skilldozer dev` (the default `var version = "dev"` at main.go:44). The old comment therefore contradicts the very install instructions the README documents. Fix: replace the comment with a line that accurately describes **both** build paths.
>
> **Scope (S1 ONLY):** edit **README.md line 136** — replace one bash comment. No code changes (contract OUTPUT §4 + DOCS §5: "The README change is the entire deliverable. No code changes."). This is a **Mode A** targeted documentation fix; the later P1.M3.T1.S1 Mode B sweep covers broader changeset consistency but does not conflict with this authoritative line fix.
>
> **STATUS (verified at PRP-write time):** README.md:136 read in context (the `## Usage` → ` ```bash ` "Everything else, commented" block, lines 118-150); `grep -c 'Version is the git-describe' README.md` → **1** (the line is unique — a single `edit` matches exactly once). main.go `var version = "dev"` verified at **main.go:44** (the contract RESEARCH NOTE + issue_analysis §Issue 5 cite main.go:57 — **drifted**; anchor by the symbol, not the number; the README comment does not cite a line number so this is context-only). install.sh:40 verified as the SOLE `-X main.version=$(git describe ...)` injection site. The README `## Install` three-path table (A install.sh / B go install / C from source, lines 19-58) confirmed: only path A injects the version. The parallel sibling P1.M2.T2.S1 edits main.go + main_test.go; this subtask edits README.md ONLY — **zero file-level overlap**, no collision.

---

## Goal

**Feature Goal**: Make the README's "version" comment accurate for ALL three documented build paths, so a user who follows the README's own "From source" (path C) or "go install" (path B) instructions and sees `skilldozer dev` is not confused by a comment claiming the version is "the git-describe value."

**Deliverable**: One-line edit to `README.md` (line 136): replace the bash comment `# Version is the git-describe value (dynamic, not a fixed string)` with `# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'`. The line stays a `#`-prefixed bash comment inside the existing ` ```bash ` Usage example block.

**Success Definition**: README.md:136 reads with the corrected wording; the OLD wording appears nowhere in README.md; the line is still a `#`-prefixed comment inside an intact ` ```bash ` fence; `git diff --name-only` shows ONLY `README.md`; `go build/vet/test ./...` stays green (proving no code was touched).

---

## User Persona (if applicable)

**Target User**: A user who builds skilldozer from source (or via `go install`) and runs `skilldozer --version`, and any reader of the README trying to understand what the version string means.

**Use Case**: User follows README "From source" (path C: `go build -o skilldozer .`) → runs `skilldozer --version` → sees `dev` → (today) reads the README comment "Version is the git-describe value (dynamic, not a fixed string)" and is confused (their build said `dev`, not a git-describe value).

**User Journey**: (today) `go build` → `skilldozer --version` prints `dev`; the README comment implies it should be a git-describe value → confusion/mistrust → (after fix) the comment clearly states a plain `go build` reports `dev`, matching observed behavior.

**Pain Points Addressed**: a self-contradicting README (the "From source" path it documents produces output its own Usage comment misdescribes); documentation accuracy for the secondary build paths.

---

## Why

- **Closes bug_fixes_validation.md / issue_analysis.md §Issue 5** (Minor — documentation accuracy). The README comment is only true for the `./install.sh` build.
- **Removes a self-contradiction.** The README documents three install paths (A `./install.sh`, B `go install`, C `go build`), then claims the version is "the git-describe value" — but paths B and C report `dev`. Users following the documented "From source" instructions see `dev` and reasonably conclude something is broken.
- **Cheapest possible accuracy fix.** One line; no code change; no behavior change; no test change. The contract DOCS §5 makes this an explicit Mode A documentation fix (the README change IS the deliverable).
- **Consistency with the build system.** Only `install.sh` performs the `-ldflags -X main.version=…` injection (install.sh:40); the new comment reflects that one fact accurately.

---

## What

A single bash-comment line replacement in the README's `## Usage` ` ```bash ` example block. No code, no config, no install.sh, no completions, no help text.

### Success Criteria

- [ ] README.md:136 reads `# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'` (the contract LOGIC §3 exact wording — single quotes, not backticks).
- [ ] The OLD line `# Version is the git-describe value (dynamic, not a fixed string)` appears NOWHERE in README.md.
- [ ] The line is still a `#`-prefixed bash comment inside the existing ` ```bash ` Usage example block (the fence and comment structure are intact).
- [ ] `git diff --name-only` → ONLY `README.md` (no code files touched).
- [ ] `go build ./... && go vet ./... && go test ./...` → all green (unaffected by a README edit; proves scope discipline).
- [ ] No changes to main.go, install.sh, main_test.go, completions/*, PRD.md, tasks.json, prd_snapshot.md, or .gitignore.

---

## All Needed Context

### Context Completeness Check

**Pass.** This is a one-line README comment replacement. The target line is pinned uniquely (`grep -c` → 1) and read in its full surrounding context (the `## Usage` ` ```bash ` block, README.md:118-150). The exact OLD and NEW text are fixed verbatim by the contract LOGIC §3 (authoritative — single-quoted form, not the backticked form in issue_analysis.md's Fix Surface). The factual basis (only `./install.sh` injects the version; `go build`/`go install` report `dev`) is verified against install.sh:40 (the sole `-X main.version=$(git describe ...)` site) and the README's own three-path `## Install` table (lines 19-58). The only non-obvious points are: (a) the line MUST remain a `#`-prefixed bash comment inside the ` ```bash ` fence (not converted to prose); (b) use single quotes (`'go build'`, `'dev'`) per the contract, not markdown backticks (which render as literal backticks inside a bash fence and are stylistically odd in a comment). The contract/issue cite `main.go:57` for the `version` var has **drifted** to main.go:44 — irrelevant to the edit (the README comment cites no line number) but noted for accuracy. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative bug writeup + the prescribed comment wording
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/issue_analysis.md
  why: "§Issue 5 (line 298) is the authoritative analysis: the README comment is only true
        for ./install.sh; a plain go build / go install yields 'dev'. Root cause = the
        ldflags injection (install.sh) is the sole version source. Fix Surface = the new
        comment line. Test Impact = 'None — this is a documentation-only change.'"
  critical: "issue_analysis's Fix Surface uses markdown BACKTICKS around `./install.sh` and
             `go build`. The CONTRACT LOGIC §3 (tasks.json, authoritative) uses SINGLE QUOTES
             and NO quotes around install.sh. Use the CONTRACT's single-quoted form — it is
             cleaner inside a ```bash fence (backticks render as literal backtick chars)."
  section: "Issue 5 (MINOR)."

# MUST READ — the file under edit (read the Usage ```bash block in full before editing)
- file: README.md
  why: "THE edit target. Line 136 = the comment, inside the `## Usage` section's ```bash
        'Everything else, commented' example block (opens ~line 109 ` ```bash `, the comment
        at 136, the block closes ~line 138 ` ``` `). Read 118-150 to confirm the fence +
        comment context. The line is UNIQUE (`grep -c` → 1), so a single `edit` oldText match
        is unambiguous."
  pattern: "The Usage block is a ```bash fence of commented example commands. Each line is a
            `#`-prefixed comment or a bare command. The version comment precedes the
            `skilldozer --version` example line (137). Keep the `#` prefix and the fence."
  gotcha: "Do NOT convert the line to markdown prose, do NOT remove the `#`, do NOT move it out
           of the ```bash fence. It stays a bash comment in the example block (contract LOGIC §3)."

# MUST READ — the contract (the orchestrator owns it; LOGIC §3 wording is authoritative)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/tasks.json
  why: "P1.M2.T3.S1's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. LOGIC §3 fixes
        the EXACT replacement text (single-quoted form). OUTPUT §4 + DOCS §5 = README.md only,
        no code changes. tasks.json wins on any conflict with issue_analysis.md."
  section: "P1.M2.T3.S1 CONTRACT."

# READ-ONLY — the version var (context only; anchor by symbol, the cite drifted)
- file: main.go
  why: "CONTEXT for WHY the comment is wrong. `var version = \"dev\"` is the default (currently
        main.go:44 — the contract + issue_analysis cite main.go:57, DRIFTED; anchor by the
        symbol `var version = \"dev\"`). This default is what paths B/C report; only
        install.sh's -X override changes it. The README comment cites NO line number, so the
        drift does not affect the edit."
  section: "var version (grep `var version`)."

# READ-ONLY — the sole version-injection site (proves only path A injects)
- file: install.sh
  why: "CONTEXT. Line 40 is the ONLY `-ldflags \"-s -w -X main.version=$(git describe --tags
        --always 2>/dev/null || echo dev)\"` injection. This is why ./install.sh (path A) reports
        the git-describe value while go build (path C) / go install (path B) report dev."
  section: "§12.1 step 3 build (line ~34-40)."

# READ-ONLY — the README Install section (the three paths the comment must cover)
- file: README.md
  why: "CONTEXT. `## Install` (lines 19-58) documents three paths: A ./install.sh (injects
        version), B go install (dev), C From source / go build (dev). The new comment must be
        accurate for ALL three. Read 19-58 to confirm the commands users actually run."
  section: "## Install (A/B/C paths)."

# READ-ONLY — the parallel sibling PRP (boundary: disjoint files, no collision)
- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M2T2S1/PRP.md
  why: "Confirms P1.M2.T2.S1 (Issue 4, POSIX `--`) edits main.go (parseArgs) + main_test.go
        (5 tests). This subtask edits README.md ONLY. ZERO file-level overlap; lands in any order."

# READ-ONLY — PRD (the §12.1 ldflags-build authority + the bugfix-2 Issue 5 repro)
- file: PRD.md
  why: "READ-ONLY. §12.1 pins that the ldflags build is specific to install.sh. The bugfix-2
        requirements doc h3.4 Issue 5 is the repro. Do NOT edit PRD.md."
  section: "§12.1 (and the bugfix-2 PRD h3.4 Issue 5)."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls README.md main.go install.sh
README.md   main.go   install.sh
$ grep -n 'Version is the git-describe' README.md        # 136 — the unique target line
$ grep -c 'Version is the git-describe' README.md        # 1   — a single edit matches exactly once
$ grep -n 'var version' main.go                          # 44:var version = "dev"  (cite drifted from 57)
$ grep -n 'X main.version' install.sh                    # 40 — the sole version-injection site
```

### Desired Codebase tree with files to be changed

```bash
README.md      # EDIT line 136: replace the version comment (OLD → NEW, contract LOGIC §3 wording).
# main.go / install.sh / main_test.go / completions/* — UNCHANGED (documentation-only fix).
```

| File | Change | Owner |
|---|---|---|
| `README.md` | Line 136 comment rewrite: `# Version is the git-describe value (dynamic, not a fixed string)` → `# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'`. Stays a `#` comment in the ` ```bash ` Usage block. | Issue 5 contract LOGIC §3 + issue_analysis.md §Issue 5 |

### Known Gotchas of our codebase & Library Quirks

```bash
# GOTCHA #1 (the wording is fixed — use the CONTRACT's single-quoted form, not backticks).
# The contract LOGIC §3 (tasks.json, authoritative) wording is:
#   # Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'
# i.e. SINGLE QUOTES around 'go build' and 'dev', and NO quotes around ./install.sh.
# issue_analysis.md's Fix Surface shows the line with markdown BACKTICKS — DO NOT use backticks.
# Backticks inside a ```bash fence render as literal backtick characters and are stylistically odd
# in a bash comment; single quotes read naturally and are command-substitution-free. tasks.json wins.

# GOTCHA #2 — the line MUST stay a `#`-prefixed bash comment inside the ```bash Usage example
# block. Do NOT convert it to markdown prose, do NOT remove the `#`, do NOT relocate it outside
# the fence. (contract LOGIC §3: "Keep the line as a bash comment (prefixed with #) in the Usage
# examples block.") It immediately precedes the `skilldozer --version` example line.

# GOTCHA #3 — the line is UNIQUE in README.md (`grep -c` → 1). A single `edit` call with the OLD
# text as oldText matches exactly once. No need for surrounding context in oldText — the single
# line is unambiguous.

# GOTCHA #4 — main.go line drift is IRRELEVANT to the edit. The contract RESEARCH NOTE and
# issue_analysis cite `main.go:57` for `var version`, but it is now main.go:44. The README comment
# cites NO line number, so this drift does not affect the edit. Do NOT "correct" the README to
# cite a line number (it never did).

# GOTCHA #5 — README.md ONLY. No code changes (contract OUTPUT §4 + DOCS §5). Do NOT touch
# main.go, install.sh, main_test.go, completions/*, PRD.md, tasks.json, prd_snapshot.md, .gitignore.
# `git diff --name-only` MUST list ONLY README.md.

# GOTCHA #6 — do NOT unbalance the ```bash fence. The edit changes one line of COMMENT text
# inside the fence; it does not add/remove fence markers. After the edit, the fence still opens
# (```bash, ~line 109) and closes (```, ~line 138) exactly once each. A quick read of the block
# confirms this.

# GOTCHA #7 — this is a MODE A targeted doc fix, NOT the Mode B sweep. The later P1.M3.T1.S1
# sweeps README/help/completions for whole-changeset consistency. This PRP fixes the version
# comment authoritatively; the later sweep confirms it (and covers other lines). No conflict.
```

---

## Implementation Blueprint

### Data models and structure

**None.** Documentation-only. No code, no types, no config.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT README.md — replace the version comment line (the entire deliverable)
  - FILE: README.md (line 136, inside the ## Usage ```bash "Everything else, commented" block)
  - FIND (the exact current line; it is UNIQUE — grep -c → 1):
        # Version is the git-describe value (dynamic, not a fixed string)
  - REPLACE with (contract LOGIC §3 authoritative wording; GOTCHA #1 single quotes, NOT backticks):
        # Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'
  - That is the ENTIRE change. The line stays a `#`-prefixed bash comment in the ```bash block
    (GOTCHA #2); it still immediately precedes the `skilldozer --version` example line (137).

Task 2: VERIFY — the edit is correct, scoped, and breaks nothing
  - grep -c 'Version is the git-describe value when built via ./install.sh' README.md   # expect 1 (NEW present)
  - grep -c 'dynamic, not a fixed string' README.md                                      # expect 0 (OLD gone)
  - grep -n 'Version is the git-describe' README.md                                      # expect exactly line 136, NEW wording
  - git diff --name-only                                                                  # expect ONLY README.md (GOTCHA #5)
  - git diff --quiet go.mod go.sum main.go install.sh main_test.go && echo "code unchanged"  # GOTCHA #5
  - go build ./... && go vet ./... && go test ./...        # all green (proves no code touched)
  - (sanity) read README.md lines ~109-138 to confirm the ```bash fence is intact and the line is
    still a `#` comment preceding `skilldozer --version` (GOTCHA #6).
```

### Implementation Patterns & Key Details

```bash
# README.md:136 — before:
# Version is the git-describe value (dynamic, not a fixed string)
skilldozer --version

# README.md:136 — after (contract LOGIC §3; single quotes, not backticks):
# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'
skilldozer --version
```

Notes easy to get wrong:
- Use the **contract's single-quoted wording**, not issue_analysis's backticked form (GOTCHA #1).
- Keep the `#` prefix and the ` ```bash ` fence (GOTCHA #2); do not convert to prose.
- Edit **README.md only** — no code files (GOTCHA #5).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Single-quoted wording (contract LOGIC §3) over backticked wording (issue_analysis Fix Surface).** The contract is authoritative; single quotes render cleanly inside a ` ```bash ` comment (backticks would show as literal backtick chars). This is the only substantive choice, and the contract already made it — follow it verbatim.
2. **Keep the line as a bash comment in the example block** (contract LOGIC §3), not as markdown prose. It sits with the other commented examples and precedes the `skilldozer --version` command it annotates. Moving it to prose would detach it from its example and change the README's established structure.
3. **No `go build`/`go install`/`./install.sh` repro added to validation.** The change is text-only; the factual claim (path A injects, B/C report dev) is already verified against install.sh:40 and the README's own install table. Running the three builds would add noise without testing the edit. The `go test ./...` green check is solely a scope-discipline guard (proves no code was touched), not a test of the comment.
4. **Mode A targeted fix, not deferred to P1.M3.T1.S1.** The contract DOCS §5 makes this subtask a Mode A documentation fix whose deliverable IS the README change. P1.M3.T1.S1 (Mode B sweep) is broader and later; this fixes the line authoritatively now. No conflict.

### Integration Points

```yaml
FILES TOUCHED:
  - README.md ONLY (line 136). No code, no config, no install.sh, no completions. (GOTCHA #5)

BUILD SYSTEM (unchanged — context only):
  - install.sh:40 remains the SOLE -X main.version=$(git describe ...) injection (path A).
  - main.go:44 `var version = "dev"` remains the default (paths B/C). No change to either.

DOCUMENTATION (this IS the deliverable):
  - Mode A targeted fix to the README version comment (contract DOCS §5). P1.M3.T1.S1 (Mode B
    sweep) covers broader changeset consistency later; this lands independently and is
    authoritative for this line.

PARALLEL SIBLING (no conflict):
  - P1.M2.T2.S1 (Issue 4) edits main.go + main_test.go; P1.M2.T1.S1 (Issue 3, complete) edited
    main.go + main_test.go. This subtask edits README.md ONLY. ZERO file-level overlap; lands in
    any order.

NO ROUTES / NO DATABASE / NO CONFIG / NO COMPLETIONS / NO HELP TEXT / NO NEW FILES.
```

---

## Validation Loop

### Level 1: Syntax & Style (the edit itself)

```bash
cd /home/dustin/projects/skilldozer

# The NEW line is present exactly once; the OLD line is GONE:
grep -c 'Version is the git-describe value when built via ./install.sh' README.md   # expect 1
grep -c 'dynamic, not a fixed string' README.md                                      # expect 0
grep -n 'Version is the git-describe' README.md                                      # expect line 136, NEW wording

# (sanity) the ```bash fence around the Usage block is intact (one open, one close) and the
# line is still a `#` comment preceding `skilldozer --version`:
sed -n '136,137p' README.md
# Expected:
#   # Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'
#   skilldozer --version
```

### Level 2: Scope discipline (NO code files touched)

```bash
cd /home/dustin/projects/skilldozer

git diff --name-only                       # expect ONLY: README.md
git diff --quiet go.mod go.sum && echo "go.mod/go.sum unchanged"
git diff --quiet main.go && echo "main.go unchanged"
git diff --quiet install.sh && echo "install.sh unchanged"
git diff --quiet main_test.go && echo "main_test.go unchanged"
# Expected: README.md is the only changed file; all code files unchanged.
```

### Level 3: Build/test still green (proves the README edit broke nothing)

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"      # 0
go vet  ./...  ; echo "vet exit $?"        # 0
go test ./...  ; echo "test exit $?"       # 0  — a README edit cannot affect this; green = scope held
# Expected: all exit 0. (This is a scope-discipline guard, not a test of the comment.)
```

### Level 4: Accuracy spot-check (the comment's factual claim holds)

```bash
cd /home/dustin/projects/skilldozer

# Confirm the comment's claim matches reality across the three README-documented build paths:
#   Path A (./install.sh) injects the git-describe value; Paths B/C (go build / go install) report dev.

# (a) install.sh:40 is the sole -X main.version injection (path A — the comment's "built via ./install.sh"):
grep -n 'X main.version' install.sh   # expect line 40: -ldflags "... -X main.version=$(git describe ...)"

# (b) the default version var is "dev" (paths B/C — the comment's "a plain 'go build' reports 'dev'"):
grep -n 'var version = "dev"' main.go  # expect main.go:44

# (c) README's own Install section documents go build (path C) with NO ldflags:
sed -n '46,56p' README.md              # expect "## C. From source" + `go build -o skilldozer .` (no -ldflags)

# (d) live check (optional): a plain go build really does report dev:
go build -o /tmp/sdz-vcheck . && /tmp/sdz-vcheck --version; echo "exit=$?"
# Expected: prints "dev" (or "skilldozer version dev") — matches the comment's claim.
rm -f /tmp/sdz-vcheck
# Expected: all four spot-checks confirm the comment is now accurate.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — the NEW comment line is present exactly once (line 136); the OLD wording is GONE; the ` ```bash ` fence is intact and the line is still a `#` comment preceding `skilldozer --version`
- [ ] Level 2 PASS — `git diff --name-only` = ONLY `README.md`; go.mod/go.sum/main.go/install.sh/main_test.go unchanged
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (scope held; no code touched)
- [ ] Level 4 PASS — install.sh:40 (sole `-X main.version` injection) + main.go:44 (`var version = "dev"`) + README path C (`go build`, no ldflags) all confirm the comment's claim; a live plain `go build` reports `dev`

### Feature Validation
- [ ] README.md:136 reads `# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'` (contract LOGIC §3 exact wording)
- [ ] The OLD line `# Version is the git-describe value (dynamic, not a fixed string)` appears nowhere in README.md
- [ ] The comment accurately describes all three README-documented build paths (A injects; B/C report dev)

### Code Quality / Convention Validation
- [ ] The line follows the README's established "commented example" style (`#` prefix in the ` ```bash ` Usage block)
- [ ] Single quotes (not backticks) used per the contract — reads cleanly inside the bash fence
- [ ] No fence imbalance introduced; the example block still opens/closes exactly once

### Scope Discipline
- [ ] Edited README.md ONLY (contract OUTPUT §4 + DOCS §5)
- [ ] Did NOT touch main.go, install.sh, main_test.go, completions/*, or any code file
- [ ] Did NOT modify PRD.md (read-only), tasks.json, prd_snapshot.md, or .gitignore
- [ ] Did NOT use issue_analysis's backticked wording (used the contract's single-quoted form)
- [ ] Did NOT convert the comment to prose or relocate it out of the ` ```bash ` block

---

## Anti-Patterns to Avoid

- ❌ **Don't use the backticked wording from issue_analysis.md's Fix Surface.** The contract LOGIC §3 (tasks.json, authoritative) uses single quotes (`'go build'`, `'dev'`) and no quotes around `./install.sh`. Backticks render as literal backtick chars inside a ` ```bash ` fence. (GOTCHA #1.)
- ❌ **Don't convert the line to markdown prose or remove the `#`.** It stays a bash comment in the ` ```bash ` Usage example block (contract LOGIC §3). (GOTCHA #2.)
- ❌ **Don't touch any code file.** This is documentation-only (contract OUTPUT §4 + DOCS §5). `git diff --name-only` must be ONLY `README.md`. (GOTCHA #5.)
- ❌ **Don't "correct" a main.go line number in the README.** The comment never cited one; the `main.go:57` drift is context-only and irrelevant to the edit. (GOTCHA #4.)
- ❌ **Don't unbalance the ` ```bash ` fence.** The edit swaps one comment line's text; it does not add/remove fence markers. (GOTCHA #6.)
- ❌ **Don't defer this to P1.M3.T1.S1 or widen it.** This is the authoritative Mode A fix for the version comment; the later Mode B sweep is broader and non-conflicting. (GOTCHA #7.)

---

## Confidence Score

**9.8/10** — This is a single-line bash-comment replacement in README.md, with the exact OLD and NEW text fixed verbatim by the contract LOGIC §3 (authoritative, single-quoted wording). The target line is grep-confirmed unique (`grep -c` → 1), so one `edit` call matches exactly once with no surrounding-context ambiguity. The factual basis (only `./install.sh` injects the version via install.sh:40; `go build`/`go install` report the `dev` default at main.go:44) is verified directly against install.sh, main.go, and the README's own three-path `## Install` table. The change is documentation-only — no code, no tests, no behavior — so the only failure modes are stylistic (using backticks instead of single quotes, dropping the `#`, or unbalancing the fence), each explicitly guarded by a GOTCHA. Zero file-level overlap with the parallel siblings (P1.M2.T2.S1 / P1.M2.T1.S1 edit main.go + main_test.go; this edits README.md only). The 0.2 reservation is purely for the trivial mechanical risk of a copy-paste typo in the replacement string, which the Level 1 grep checks (NEW present once, OLD gone) immediately catch.
