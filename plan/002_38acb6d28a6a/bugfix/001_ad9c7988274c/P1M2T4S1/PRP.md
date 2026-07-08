# PRP — P1.M2.T4.S1: Rewrite `.gitignore` to the §16 canonical 5-entry set (Issue 6)

> **Subtask:** a static, single-file rewrite of the repo-root `.gitignore` so it byte-for-byte
> matches the PRD §16 spec block (`PRD.md:426-430`). No code change, no Go test, no README change,
> no new files. The acceptance oracle is a `diff` against the spec lines; the file content itself is
> the test (architecture/bug_fixes_validation.md §ISSUE 6: "No Go test (.gitignore is not code)").
>
> **Why byte-exact matters:** the gate `diff <(sed -n '426,430p' PRD.md) .gitignore` must emit
> NOTHING, so a trailing blank line, a missing final `\n`, a stray comment, or any preserved extra
> all FAIL. The §16 block is EXACTLY 5 lines, each `\n`-terminated, no comments (architecture/decisions.md
> §D5: "the §16 canonical block has NO section comments; the rewrite omits them").
>
> **STATUS (verified at PRP-write time):** current `.gitignore` = 19 lines (5 KEEP / 5 EXTRA /
> 5 COMMENT / 4 blank). `sed -n '426,430p' PRD.md | cat -A` = the 5 KEEP lines each `$`-terminated,
> no trailing blank. `grep -rn 'gitignore' main.go main_test.go internal/ install.sh` → no matches
> (no code reads it). `grep -in 'gitignore' README.md` → no matches (README:291's `node_modules`
> mention is incidental pi-ecosystem prose, not a `.gitignore` enumeration). Parallel sibling
> P1.M2.T3.S2 edits ONLY `main.go`+`main_test.go` (its checklist: "Did NOT modify … `.gitignore`")
> → **zero file-level overlap**; land in either order. `git status --ignored` today shows only
> `.pi-subagents/` + `skilldozer` ignored; removing `.pi-subagents/` makes that dir untracked
> (surfaces in `git status`) — **intended** per §D5/§D3 ("do NOT bless extras"). All numbers below
> are reproducible (see research/verified_facts.md).

---

## Goal

**Feature Goal**: Bring the repo-root `.gitignore` into exact spec compliance with PRD §16 by
overwriting it with the canonical 5-entry block — `/skilldozer`, `/dist`, `*.test`, `*.out`,
`.DS_Store` — each newline-terminated, with NO section comments and NO extra entries.

**Deliverable**: ONE overwritten file, `/home/dustin/projects/skilldozer/.gitignore`, whose bytes
are identical to `sed -n '426,430p' PRD.md` (the §16 code-fence interior). Concretely the file's
entire content is:

```
/skilldozer
/dist
*.test
*.out
.DS_Store
```

terminated by exactly one trailing `\n` (POSIX text file), nothing before/after.

**Success Definition**: `diff <(sed -n '426,430p' PRD.md) .gitignore` produces **no output** (exit
0); `wc -l .gitignore` prints `5`; `git status` no longer ignores `.pi-subagents/` (it surfaces as
untracked — the intended §D5 residual risk); `go test ./...` is unaffected (no code change).

## User Persona (if applicable)

**Target User**: QA / acceptance harness running the §16 gate, and any maintainer expecting the
repo to match its own spec.

**Use Case**: The bug-fix QA round (this changeset) flags (Issue 6, h3.5) that `.gitignore` carries
5 extras + 5 comments beyond §16; this subtask trims it to spec so the §16 gate passes.

**User Journey**: (today) the §16 acceptance `diff` emits a delta (5 extras + comments) → (after)
the `diff` is empty; `.gitignore` is a 5-line, comment-free, spec-faithful file.

**Pain Points Addressed**: the undocumented deviation between repo state and the PRD spec; the
noise of section comments and non-spec extras in a file the PRD pins to exactly 5 lines.

## Why

- **Closes architecture/bug_fixes_validation.md §ISSUE 6 (Minor)** and the bugfix PRD h3.5: the
  `.gitignore` carries entries beyond the §16 spec set. The fix is to trim, not to bless.
- **architecture/decisions.md §D5 + prior-round §D3 are explicit and binding:** "do NOT bless
  extras … the §16 canonical block has NO section comments; the rewrite omits them." PRD.md is
  read-only/human-owned, so the ONLY action that resolves the discrepancy without editing the PRD
  is to bring `.gitignore` into spec. If maintainers want an extra, they update §16 themselves.
- **The residual risk is noted, not a blocker:** removing `.pi-subagents/` makes the
  agent-artifacts dir untracked (it will surface in `git status` as `??`). This is intentional and
  is the documented trade-off of not blessing extras. The implementer must NOT re-add it "to be tidy".
- **Cheap and isolated:** one file, no code, no test, no README, no deps. Zero risk of regression
  to the Go build/test surface; the change is invisible to `go build/vet/test` (`.gitignore` is not
  code — confirmed: nothing in `main.go`/`main_test.go`/`internal/`/`install.sh` reads it).

## What

A single overwrite of `.gitignore` with the exact §16 block. No comment lines, no section headers,
no blank separators, no `/build`, no `node_modules/`, no `venv/`, no `.env`, no `.pi-subagents/`.
The file's entire byte content becomes `/skilldozer\n/dist\n*.test\n*.out\n.DS_Store\n`.

### Success Criteria

- [ ] `.gitignore` content is byte-for-byte identical to `sed -n '426,430p' PRD.md` (the §16 block),
      i.e. the 5 entries in spec order, each `\n`-terminated, exactly one trailing `\n`, nothing else.
- [ ] `diff <(sed -n '426,430p' PRD.md) .gitignore` prints nothing and exits 0.
- [ ] `wc -l .gitignore` prints `5` (no trailing 6th blank line; no missing final newline).
- [ ] The 5 EXTRA entries (`/build`, `node_modules/`, `venv/`, `.env`, `.pi-subagents/`) and ALL 5
      `#`-comment lines and ALL blank separators are GONE.
- [ ] `.pi-subagents/` is no longer ignored (`git check-ignore .pi-subagents` exits 1) — surfaces as
      untracked (intended §D5 residual risk); the `skilldozer` build binary stays ignored via `/skilldozer`.
- [ ] No other file is touched (no README, no Go code, no test, no `PRD.md`, no `tasks.json`).
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` remain green (unchanged — not code).

## All Needed Context

### Context Completeness Check

**Pass.** The deliverable is a deterministic byte-exact file overwrite whose target content is
reproducible verbatim from `sed -n '426,430p' PRD.md` (printed with `cat -A` to pin the line-ends:
5 lines, each `$`-terminated, no trailing blank, no BOM). The current `.gitignore` is dumped in full
(`cat -A` + `xxd`) so the implementer knows exactly what is being removed (5 extras + 5 comments +
4 blanks). The acceptance command (`diff <(sed -n '426,430p' PRD.md) .gitignore`) is stated, runnable,
and its pass condition (empty output) is unambiguous. It is confirmed there is no Go test, no Go
code, no install script, and no README content that reads or enumerates `.gitignore`, so the change
is fully isolated to that one file. The residual risk (`.pi-subagents/` becomes untracked) is named
and traced to the binding decision (§D5/§D3). An implementer who has never seen this repo can do it
in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative spec block (THE acceptance oracle source)
- file: PRD.md
  why: "§16 (lines 423-431) is the canonical 5-entry spec. The code-fence INTERIOR is lines
        426-430 (/skilldozer / /dist / *.test / *.out / .DS_Store). The acceptance gate is
        `diff <(sed -n '426,430p' PRD.md) .gitignore` → empty. REPRODUCE the target bytes with
        `sed -n '426,430p' PRD.md | cat -A` before writing (pin: 5 lines, each \\n-terminated,
        NO trailing blank line). Do NOT type the entries from memory — copy them from the PRD."
  section: "§16 (`.gitignore`), lines 426-430."
  critical: "READ-ONLY file. Never edit PRD.md. The 425/431 backticks are the FENCE, not entries —
             the .gitignore holds ONLY lines 426-430."

# MUST READ — the authoritative bug writeup + the exact acceptance command
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/bug_fixes_validation.md
  why: "§ISSUE 6 enumerates the 5 extras (/build, node_modules/, venv/, .env, .pi-subagents/) and
        the 5 comment lines, states the fix (rewrite to the exact 5-line block, no comments,
        byte-for-byte), and pins the verification: `diff <(sed -n '426,430p' PRD.md) .gitignore`
        → no output. Also: 'No Go test (.gitignore is not code).' → no test to add."
  section: "ISSUE 6 (Minor)."

# MUST READ — the binding decision (trim exactly; do NOT bless extras; comments removed too)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/decisions.md
  why: "§D5 is the decision of record: 'trim to §16 exactly, INCLUDING removing comments and
        .pi-subagents/. Per prior-round §D3, do NOT bless extras. The .pi-subagents/ dir becomes
        untracked (surfaces in git status) — that is intended.' This forbids re-adding any extra
        and forbids keeping the comments. The §16 block has NO section comments by design."
  section: "D5 (and the prior-round D3 it cites — 'do NOT bless extras … the spec is explicit about
            the 5-entry set … bringing the file into spec compliance is the only action that
            resolves the discrepancy without modifying PRD.md')."

# MUST READ — the file under edit (the ONLY deliverable)
- file: .gitignore
  why: "THE edit target — a single overwrite. Current content = 19 lines (5 KEEP mapping 1:1 to
        PRD.md:426-430; 5 EXTRA; 5 COMMENT; 4 blank separators). File currently ends with
        `.pi-subagents/\\n`. After the overwrite it must end with `.DS_Store\\n` and be exactly
        5 lines."
  pattern: "A plain-text ignore file. There is no code structure to follow — only the spec bytes.
            Use the PRD §16 block verbatim (sed it out), NOT a hand-typed list."

# READ-ONLY — confirms isolation (no code/test/README reads .gitignore)
- file: main.go
  why: "Confirm via `grep -n gitignore main.go` → no matches. The .gitignore change cannot affect
        any Go symbol, build, or test. (Same for main_test.go, internal/, install.sh.)"
  section: "(grep only — .gitignore is not referenced anywhere in Go code.)"
  gotcha: "If a future change makes Go read .gitignore, THIS subtask's 'no test' assumption breaks;
           out of scope for this bugfix round — the §16 gate is the sole acceptance check today."

# READ-ONLY — the parallel sibling boundary (disjoint paths; land in either order)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/P1M2T3S2/PRP.md
  why: "P1.M2.T3.S2 edits ONLY main.go + main_test.go (tilde expansion). Its scope-discipline
        checklist explicitly says 'Did NOT modify … .gitignore'. This subtask edits ONLY
        .gitignore. DISJOINT file sets → no merge conflict; land in either order."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/tasks.json
  why: "P1.M2.T4.S1's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. This PRP
        transcribes it; tasks.json wins on any conflict."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && cat -A .gitignore   # 19 lines; the whole file
# Build artifacts$
/skilldozer$                  # KEEP (§16 line 426)
/dist$                        # KEEP (§16 line 427)
/build$                       # EXTRA — remove
*.test$                       # KEEP (§16 line 428)
*.out$                        # KEEP (§16 line 429)
$                             # blank — remove
# Dependency directories$     # COMMENT — remove
node_modules/$                # EXTRA — remove
venv/$                        # EXTRA — remove
$                             # blank — remove
# Environment files$          # COMMENT — remove
.env$                         # EXTRA — remove
$                             # blank — remove
# OS-specific files$          # COMMENT — remove
.DS_Store$                    # KEEP (§16 line 430)
$                             # blank — remove
# Agent runtime artifacts (transcripts, run logs, meta)$   # COMMENT — remove
.pi-subagents/$               # EXTRA — remove (residual risk, see §4)

$ sed -n '426,430p' PRD.md    # the EXACT target bytes (the acceptance oracle)
/skilldozer
/dist
*.test
*.out
.DS_Store
$ git status --ignored --short   # today: only .pi-subagents/ + skilldozer ignored
!! .pi-subagents/
!! skilldozer
```

### Desired Codebase tree with files to be changed

```bash
.gitignore      # OVERWRITE — replace all 19 lines with exactly the §16 5-line block (no comments,
                #          no extras, one trailing newline). This is the ONLY file changed.
# main.go / main_test.go / internal/ / install.sh / README.md / go.mod / go.sum — UNCHANGED.
# PRD.md / tasks.json / prd_snapshot.md — READ-ONLY (never touched).
```

| File | Change | Owner |
|---|---|---|
| `.gitignore` | Overwrite with the exact PRD §16 (PRD.md:426-430) 5-line block; remove 5 extras + 5 comments + 4 blanks. | Issue 6 contract + decisions.md §D5 |

### Known Gotchas of our codebase & Library Quirks

```bash
# GOTCHA #1 (CRITICAL — the #1 one-pass stall) — byte-for-byte means byte-for-byte. The acceptance
# `diff <(sed -n '426,430p' PRD.md) .gitignore` must emit NOTHING. That requires .gitignore to be
# EXACTLY: /skilldozer\n/dist\n*.test\n*.out\n.DS_Store\n — ONE trailing newline, NO 6th blank line,
# NO missing final newline, NO leading BOM, NO trailing spaces. A trailing blank line (so common when
# pasting into an editor) makes `diff` emit `5a6 > ` and FAILS the gate. After writing, verify with
# `tail -c 20 .gitignore | xxd` (must end in `2e44535f53746f7265 0a` = `.DS_Store\n`, no double `\n`).
# (research/verified_facts.md §1/§7.)

# GOTCHA #2 — do NOT re-add .pi-subagents/ (or any extra) "to be tidy". decisions.md §D5 is binding:
# "do NOT bless extras … .pi-subagents/ becomes untracked — that is intended." Re-adding it re-opens
# Issue 6 and violates the decision record. The dir surfacing as untracked in `git status` is the
# NOTED residual risk, not a bug to silently fix. If a maintainer wants it ignored, THEY update §16.

# GOTCHA #3 — do NOT keep the comments. §D5: "the §16 canonical block has NO section comments; the
# rewrite omits them." So drop ALL five `#`-prefixed lines AND the blank separators. The result is a
# bare 5-line file. (Keeping comments would make `diff` against the 5 spec lines emit the comment
# lines as extras → FAIL.)

# GOTCHA #4 — order and spelling are pinned by the spec. The 5 entries must appear in PRD §16 order
# (/skilldozer, /dist, *.test, *.out, .DS_Store) and spelled exactly as in PRD.md:426-430 (note the
# leading slashes on the first two; NO trailing slash on .DS_Store; *.test/*.out are glob patterns).
# Do NOT "normalize" (e.g. adding trailing slashes, or `.DS_Store/`). Copy the bytes from the PRD.

# GOTCHA #5 — there is NO Go test to add and NO code to change. "No Go test (.gitignore is not code)"
# (bug_fixes_validation.md §ISSUE 6). Confirmed: `grep -rn gitignore main.go main_test.go internal/
# install.sh` → nothing reads it. Do not invent a test; the file content IS the acceptance check.
# `go test ./...` must remain green simply because nothing changed in Go land.

# GOTCHA #6 — no README change. Mode A: the .gitignore IS the doc/config surface, and §16 has no
# comments by design. `grep -in gitignore README.md` → no match (README:291's `node_modules` is
# incidental pi-ecosystem prose, not a .gitignore enumeration). The README sweep is a separate Mode B
# task (P1.M3.T1) and does not touch .gitignore content anyway.

# GOTCHA #7 — do NOT copy the markdown code-fence backticks. PRD.md:425 and :431 are the ``` fences;
# the .gitignore holds ONLY the interior lines 426-430. Including a ``` line makes `diff` FAIL.

# GOTCHA #8 — reproduce the target from the PRD, don't hand-type it. Run `sed -n '426,430p' PRD.md`
# to see the exact bytes, and verify your written file against it with the diff command. This is the
# single most reliable way to avoid a transcription error in spelling/order/line-ends.
```

## Implementation Blueprint

### Data models and structure

**None.** This is a static text-file rewrite. No types, fields, signatures, config format, code, or
tests are involved. The §16 5-entry block is the entire "data model."

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: OVERWRITE ./.gitignore with the exact §16 block (the ONE and ONLY production action)
  - FILE: .gitignore (repo root)
  - ACTION: replace the entire 19-line contents with EXACTLY these 5 lines, in this order, each
            newline-terminated, and a single trailing newline after .DS_Store (nothing else):
                /skilldozer
                /dist
                *.test
                *.out
                .DS_Store
  - SOURCE OF TRUTH: PRD.md:426-430. Reproduce with `sed -n '426,430p' PRD.md` and diff your
            written file against it — do NOT hand-type (GOTCHA #8).
  - REMOVE (all of these, fully):
        comments : # Build artifacts / # Dependency directories / # Environment files /
                    # OS-specific files / # Agent runtime artifacts (transcripts, run logs, meta)
        extras   : /build, node_modules/, venv/, .env, .pi-subagents/      (GOTCHA #2: do NOT re-add)
        blanks   : the 4 empty separator lines between sections
  - KEEP (map 1:1 to §16): /skilldozer, /dist, *.test, *.out, .DS_Store
  - BYTE INVARIANTS (GOTCHA #1/#3/#4/#7): no comment lines; no blank lines; no code-fence ``` ;
            exact spelling/order; exactly one trailing \n; file size = 5 lines.

Task 2: VERIFY the byte-exact gate + isolation invariants (the acceptance loop)
  - GATE (THE acceptance check; must be EMPTY):
        diff <(sed -n '426,430p' PRD.md) .gitignore && echo "§16 gate: PASS (no diff)"
  - LINE COUNT:      wc -l .gitignore          # expect 5
  - TAIL BYTES:      tail -c 20 .gitignore | xxd   # ends in ".DS_Store\n" (0a), no double newline
  - RESIDUAL RISK (intended): git check-ignore .pi-subagents ; echo "exit=$?"   # expect 1 (untracked now)
  - BUILD-ARTIFACT STILL IGNORED: git check-ignore skilldozer ; echo "exit=$?"  # expect 0 (/skilldozer kept)
  - NO CODE REGRESSION: go build ./... && go vet ./... && go test ./...   # all exit 0 (unchanged)
  - ISOLATION: git status --short .gitignore   # shows modified: .gitignore only; no other path changed
```

### Implementation Patterns & Key Details

```bash
# The deliverable is a single overwrite. Two equally-correct ways to produce byte-exact bytes:
#
# (A) PREFERRED — derive the target from the PRD so there is zero transcription risk:
#       sed -n '426,430p' PRD.md > .gitignore
#     (sed writes exactly the 5 spec lines each \n-terminated, single trailing \n. This is provably
#      byte-identical to the gate's left-hand side, so `diff` is empty by construction.)
#
# (B) Write the 5 lines verbatim via the write tool / heredoc, then DIFF against the PRD:
#       cat > .gitignore <<'EOF'
#       /skilldozer
#       /dist
#       *.test
#       *.out
#       .DS_Store
#       EOF
#     (the quoted 'EOF' prevents shell glob expansion of *.test/*.out — GOTCHA #4; ensure the editor
#      does not append a trailing blank line — GOTCHA #1.)
#
# Either way, the acceptance command is identical and authoritative:
#       diff <(sed -n '426,430p' PRD.md) .gitignore   # MUST print nothing
#
# After writing, ALWAYS run the byte-checks (Task 2): wc -l == 5, xxd tail ends in .DS_Store\n,
# git check-ignore .pi-subagents exits 1 (untracked), go test ./... stays green.
```

Notes easy to get wrong:
- A trailing blank line is the most common one-pass failure — verify with `xxd` tail (GOTCHA #1).
- Do not preserve any comment or extra "for context" — §D5 forbids comments AND extras (GOTCHA #2/#3).
- Derive from the PRD (`sed`), don't hand-type, to avoid a spelling/order slip (GOTCHA #8).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Overwrite, not surgical edit.** The current file interleaves comments + extras + blanks with the
   5 keepers in a non-contiguous way. Editing in place (removing 14 scattered lines) is more error-
   prone and harder to byte-verify than a single overwrite. The PRP prescribes OVERWRITE with the
   spec block. (decisions.md §D5: "the rewrite omits them" — "rewrite" is the operative verb.)
2. **Derive the bytes from the PRD.** `sed -n '426,430p' PRD.md > .gitignore` is provably byte-
   identical to the gate's left-hand side, so the acceptance `diff` is empty by construction. This
   removes all transcription risk (spelling, order, line-ends). Hand-typing is acceptable only if the
   diff is then run and clean. (GOTCHA #8.)
3. **No Go test, by design.** `.gitignore` is not code; nothing reads it (grep-confirmed). The §16
   `diff` gate is the sole acceptance check. Inventing a test would be scope-creep and unhelpful.
   (bug_fixes_validation.md §ISSUE 6: "No Go test (.gitignore is not code).")
4. **Accept the `.pi-subagents/` residual risk.** Removing `.pi-subagents/` makes the agent-artifacts
   dir untracked. This is intended (§D5/§D3) and is the documented trade-off of not blessing extras.
   The PRP does NOT re-add it. The 4 other extras (`/build`, `node_modules/`, `venv/`, `.env`) have
   no on-disk files, so removing them is purely cosmetic (nothing surfaces in `git status`).
5. **No README change.** Mode A: the `.gitignore` IS the doc/config surface and §16 has no comments.
   README does not enumerate `.gitignore` (grep-confirmed). The README sweep is the separate Mode B
   task P1.M3.T1, which does not alter `.gitignore` content.

### Integration Points

```yaml
GIT:
  - effect: ".gitignore now matches PRD §16 exactly (5 entries, no comments/extras)."
  - residual: ".pi-subagents/ becomes UNTRACKED (surfaces in `git status`) — intended per §D5/§D3."
  - preserved: "the locally-built `skilldozer` binary stays ignored via the `/skilldozer` entry."

CODE: NONE.
  - go.mod / go.sum UNCHANGED. No Go file is read or written. `go build/vet/test ./...` unaffected.
  - "No Go test (.gitignore is not code)." (bug_fixes_validation.md §ISSUE 6.)

DOCUMENTATION (Mode A only):
  - The .gitignore IS the doc/config surface; §16 has no comments by design (§D5). No separate doc edit.
  - README sweep is the final Mode B task (P1.M3.T1) — no doc file rides here.

PARALLEL SIBLING (no conflict):
  - P1.M2.T3.S2 edits ONLY main.go + main_test.go. This subtask edits ONLY .gitignore. DISJOINT paths;
    land in either order with no merge conflict.

NO DATABASE / NO ROUTES / NO CONFIG-FORMAT CHANGE / NO PARSEARGS CHANGE / NO NEW FILES.
```

## Validation Loop

### Level 1: Syntax & Style (immediate, after the overwrite)

```bash
cd /home/dustin/projects/skilldozer

# THE acceptance oracle — must print NOTHING (exit 0):
diff <(sed -n '426,430p' PRD.md) .gitignore && echo "§16 gate: PASS (no diff)"

# Byte hygiene:
wc -l .gitignore                 # expect 5
tail -c 20 .gitignore | xxd      # must end in ".DS_Store\n" (hex: ...44535f53746f7265 0a); no double \n
# Expected: "§16 gate: PASS (no diff)"; wc -l == 5; xxd tail ends in a single 0a after .DS_Store.
```

### Level 2: Unit / Component Tests (N/A — .gitignore is not code)

```bash
cd /home/dustin/projects/skilldozer

# There is no Go test for .gitignore (bug_fixes_validation.md §ISSUE 6: "No Go test").
# The §16 diff (Level 1) is the component check. Confirm nothing in Go land references it:
grep -rn 'gitignore' main.go main_test.go internal/ install.sh || echo "OK: no code reads .gitignore"
# Expected: "OK: no code reads .gitignore" (grep exits 1 = no matches).
```

### Level 3: Integration / Repo-State Validation

```bash
cd /home/dustin/projects/skilldozer

# No Go regression (nothing changed in Go land, but prove it):
go build ./... ; echo "build exit $?"    # 0
go vet  ./...  ; echo "vet exit $?"      # 0
go test ./...  ; echo "test exit $?"     # 0

# Residual risk (intended per §D5/§D3): .pi-subagents/ is NO LONGER ignored (now untracked).
git check-ignore .pi-subagents ; echo "exit=$?"   # expect 1 (NOT ignored) — surfaces as ?? in status
# Build artifact STILL ignored (the /skilldozer entry is §16 line 1):
git check-ignore skilldozer      ; echo "exit=$?"   # expect 0 (still ignored)

# Isolation: only .gitignore changed in the working tree (besides pre-existing plan/ churn).
git status --short .gitignore    # expect " M .gitignore"
# Expected: build/vet/test all exit 0; .pi-subagents no longer ignored (exit 1); skilldozer still
# ignored (exit 0); only .gitignore shows as modified.
```

### Level 4: Behavioral / Domain-Specific Validation (the contract OUTPUT, by hand)

```bash
cd /home/dustin/projects/skilldozer

# Contract OUTPUT: the file IS the spec. Show the final contents and re-run the gate end-to-end:
cat .gitignore                   # the 5 §16 lines, in order, no comments
diff <(sed -n '426,430p' PRD.md) .gitignore && echo "FINAL: .gitignore == PRD §16 byte-for-byte"

# Confirm the extras are gone and the keepers remain:
for e in '/skilldozer' '/dist' '\*.test' '\*.out' '.DS_Store'; do
  grep -qxF "$e" .gitignore && echo "KEEP present: $e" || echo "MISSING: $e"
done
for e in '/build' 'node_modules/' 'venv/' '.env' '.pi-subagents/'; do
  grep -qxF "$e" .gitignore && echo "EXTRA STILL PRESENT (FAIL): $e" || echo "extra removed: $e"
done
# Expected: all 5 KEEP present; all 5 EXTRA removed; FINAL line printed; .pi-subagents/ untracked.
```

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `diff <(sed -n '426,430p' PRD.md) .gitignore` prints nothing; `wc -l .gitignore` == 5; `xxd` tail ends in `.DS_Store\n` (single `\n`)
- [ ] Level 2 PASS — `grep -rn gitignore main.go main_test.go internal/ install.sh` → no matches (no code reads it)
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (no Go regression); `.pi-subagents/` no longer ignored (`git check-ignore` exit 1); `skilldozer` binary still ignored (exit 0)
- [ ] Level 4 PASS — all 5 §16 entries present (`grep -qxF`); all 5 extras removed; `cat .gitignore` shows exactly the 5 spec lines, no comments

### Feature Validation
- [ ] `.gitignore` is byte-for-byte identical to `sed -n '426,430p' PRD.md` (the §16 block)
- [ ] The 5 extras (`/build`, `node_modules/`, `venv/`, `.env`, `.pi-subagents/`) and all 5 comment lines and all blank separators are GONE
- [ ] The `.pi-subagents/` untracking is the intended §D5 residual risk (NOT re-added; surfaced in `git status` as `??`)
- [ ] The `skilldozer` build binary remains ignored via the `/skilldozer` entry
- [ ] Only `.gitignore` changed; no other file touched

### Code Quality / Convention Validation
- [ ] Follows the existing convention of a single `.gitignore` at repo root (no new ignore files)
- [ ] File ends with exactly one trailing newline (POSIX text file); no trailing blank line
- [ ] No code, no test, no README, no `PRD.md`, no `tasks.json`, no `prd_snapshot.md`, no `go.mod`/`go.sum` change

### Documentation & Deployment
- [ ] Mode A: the `.gitignore` IS the doc/config surface; §16 has no comments by design (no separate doc edit) — §D5
- [ ] No README change (README does not enumerate `.gitignore`; the sweep is Mode B task P1.M3.T1)
- [ ] No new environment variables or configuration introduced

---

## Anti-Patterns to Avoid

- ❌ **Don't leave a trailing blank line.** Byte-for-byte means exactly one trailing `\n` after `.DS_Store`.
  A 6th empty line makes the `diff` emit `5a6 > ` and FAILS the §16 gate. Verify with `xxd` tail.
  (GOTCHA #1.)
- ❌ **Don't re-add `.pi-subagents/` (or any extra).** §D5 is binding: "do NOT bless extras …
  .pi-subagents/ becomes untracked — that is intended." Re-adding it re-opens Issue 6. (GOTCHA #2.)
- ❌ **Don't keep the comments or blank separators.** §D5: the canonical block has NO section comments;
  the rewrite omits them. (GOTCHA #3.)
- ❌ **Don't hand-type the entries.** Reproduce from `sed -n '426,430p' PRD.md` and diff against it, to
  avoid a spelling/order/line-end slip. (GOTCHA #8.)
- ❌ **Don't copy the markdown code-fence backticks.** The `.gitignore` holds ONLY the interior lines
  426-430, not the ``` fences on 425/431. (GOTCHA #7.)
- ❌ **Don't add a Go test or edit any Go/README/install file.** `.gitignore` is not code; nothing reads
  it (grep-confirmed). The §16 `diff` is the sole acceptance check. (GOTCHA #5/#6.)
- ❌ **Don't edit `PRD.md`, `tasks.json`, or `prd_snapshot.md`.** Read-only / orchestrator-owned.

---

## Confidence Score

**9.7/10** — This is a single-file overwrite whose target bytes are reproducible verbatim from
`sed -n '426,430p' PRD.md` (the §16 block, printed with `cat -A`: 5 lines, each `\n`-terminated, no
trailing blank, no BOM). The acceptance oracle (`diff <(sed -n '426,430p' PRD.md) .gitignore` → empty)
is unambiguous and runnable. The current file is fully dumped (`cat -A` + `xxd`) so the implementer
knows exactly what is removed (5 extras + 5 comments + 4 blanks). It is grep-confirmed that NO Go
code/test/install-script/README reads or enumerates `.gitignore`, so the change is fully isolated and
`go build/vet/test ./...` cannot regress. The residual risk (`.pi-subagents/` becomes untracked) is
named, traced to the binding decision (§D5/§D3), and explicitly NOT to be worked around. The
parallel sibling (P1.M2.T3.S2) touches only `main.go`+`main_test.go` → zero file overlap. The 0.3
reservation is for the single most-likely one-pass stall — a stray trailing blank line or a
transcription slip (wrong order/spelling/line-end) — which the PRP's "derive from the PRD via `sed`,
then diff" method + the `xxd` tail check + GOTCHA #1/#4/#8 jointly eliminate.
