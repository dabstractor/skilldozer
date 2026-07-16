# PRP — P1.M1.T1.S1: bash — add `show-all-if-ambiguous` to the on-disk (== emitted) script

> **Subtask:** P1.M1.T1.S1 — the bash half of P1.M1.T1 (§14.7 / decision 22: list every ambiguous match on the first Tab; never a silent halt at the common prefix).
> **Scope boundary:** Appends a disclosure comment + one guarded `bind` line + a commented opt-out to `completions/skilldozer.bash` ONLY. bash is emitted **verbatim** by `runCompletion`, so this single file edit covers both the §14.5 manual `source` path and the §14.6 `eval` path. Does NOT touch any `.go` file (the `//go:embed` picks up the change on rebuild); does NOT add a test (the byte-level assertion is P1.M1.T2.S1); does NOT edit the README (P1.M1.T3); does NOT touch zsh (S2) or fish (no change needed).

---

## Goal

**Feature Goal**: Make the bash completion script (both the on-disk file and the `--completions --shell bash` emitted bytes) set readline's `show-all-if-ambiguous on`, so an ambiguous prefix lists **all** matches on the first Tab instead of completing the common prefix and halting silently — fulfilling PRD §14.7 for bash, with the change disclosed in comments and a one-line opt-out.

**Deliverable**: An append to `completions/skilldozer.bash` (after its last line `complete -F _skilldozer_completion skilldozer`):
1. a disclosure comment block (§14.7 intent; session-global; bash default OFF; guard rationale),
2. an active guarded line `[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'`,
3. a commented opt-out `bind 'set show-all-if-ambiguous off'`.

**Success Definition**: `go build ./...` succeeds; `TestEmbeddedCompletionsMatchOnDisk` and `TestRunCompletionBashScript` stay GREEN; after rebuild, `./skilldozer --completions --shell bash` output contains `show-all-if-ambiguous on` and the opt-out token `show-all-if-ambiguous off`; `bash -n completions/skilldozer.bash` is clean; `go.mod`/`go.sum` unchanged; no `.go` file edited.

---

## User Persona (if applicable)

**Target User**: bash users who tab-complete `skilldozer` (the primary discovery path for a manifest-free store).

**Use Case**: A user types `skilldozer --c<Tab>` and immediately sees `--check` and `--completions` (not a frozen `--c` + beep).

**Pain Points Addressed**: bash defaults to `show-all-if-ambiguous off`, so the first Tab completes only the common prefix and the candidate list appears on the second Tab — a silent halt that hides the very tags/flags the user is trying to discover.

---

## Why

- **PRD §14.7**: "A completion that completes the longest common prefix and then stops with nothing shown is a defect." The store is manifest-free (§2), so the user often doesn't know a tag ahead of time — discovery-via-completion is primary. An ambiguous prefix that hides candidates defeats that.
- **§14.7 bash half**: `show-all-if-ambiguous` is **off by default** in bash; `bind 'set show-all-if-ambiguous on'` lists all matches on the first Tab. A completion function cannot set it, but the emitted `--completions` script can `bind` it.
- **Disclosure + opt-out required**: the option is **session-global** (it changes listing for *every* command in the shell, not just skilldozer). PRD §14.7 mandates the emitted script (a) set it, (b) disclose it in comments, and (c) provide a one-line opt-out (`bind 'set show-all-if-ambiguous off'`). This subtask delivers all three for bash.
- **Decision 22**: "First-Tab list-all-matches; never a silent halt at the common prefix" — disclosed and opt-out-able.

---

## What

`completions/skilldozer.bash` gains, after the `complete -F _skilldozer_completion skilldozer` registration line, a comment block + one active `bind` + a commented opt-out. Because bash is emitted verbatim by `runCompletion`, the same bytes reach users whether they `source` the file (§14.5) or `eval "$(skilldozer --completions)"` (§14.6).

No behavior change to the completion function itself (`_skilldozer_completion` is untouched); the tag/flag candidate sets it already offers are complete (§14.7 half #1, already true). This adds half #2 — making the shell *show* them on the first Tab.

### Success Criteria

- [ ] `completions/skilldozer.bash` ends with: disclosure comment block + `[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'` + commented opt-out
- [ ] the disclosure names `show-all-if-ambiguous`, notes it is session-global, and gives the opt-out
- [ ] `go build ./...` succeeds; `bash -n completions/skilldozer.bash` clean
- [ ] `TestEmbeddedCompletionsMatchOnDisk` + `TestRunCompletionBashScript` stay GREEN
- [ ] after rebuild, `./skilldozer --completions --shell bash` output contains `show-all-if-ambiguous on` and `show-all-if-ambiguous off`
- [ ] no `.go` file edited; `go.mod`/`go.sum` unchanged

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact current last line of the file, the verbatim-emit proof (one edit covers both paths), the exact append text (comment + active line + opt-out), the embed/rebuild mechanic, the two gate tests and why they stay green, and the scope boundary (bash only; the locking test is a separate subtask) are all specified. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
- file: completions/skilldozer.bash
  why: "THE edit site (the only file touched). Currently 67 lines (contract/map say 69 — stale, but the LAST LINE `complete -F _skilldozer_completion skilldozer` is correct, at line 67). Append AFTER that line."
  pattern: "Append-only: a comment block + one active `bind` line (guarded for interactivity) + a commented opt-out. The _skilldozer_completion function (line 20) and the complete -F registration (line 67) are UNCHANGED."
  gotcha: "Line count is 67 live (not 69). The last line is the registration; append after it so the `bind` runs at source time, after the function is registered."

- file: main.go
  why: "The verbatim-emit proof (NO edit — confirming one file edit suffices). runCompletion @ :1499-1527 emits bash verbatim (only zsh is derived via zshEvalScript at :1519-1521). completionScript('bash') @ :1217 returns bashCompletion verbatim. //go:embed @ :54-55 → var bashCompletion @ :55."
  pattern: "Do NOT touch main.go. bash needs no Go change; the embed picks up the file edit on rebuild."
  gotcha: "A PRE-BUILT binary holds stale embedded bytes. Always rebuild (go build / go test) before behavioral testing."

- file: main_test.go
  why: "The two gate tests that must stay GREEN. TestEmbeddedCompletionsMatchOnDisk @ :3139 (completionScript('bash') == on-disk file; they move together). TestRunCompletionBashScript @ :3163 (run → code 0, stdout contains _skilldozer_completion, Go stderr empty — all unaffected by the append). No test asserts the option today; the locking test is P1.M1.T2.S1."
  pattern: "S1's automated gate is 'existing tests stay green' + the manual CLI grep. Do NOT add the asserting test here."

- file: plan/005_d9b30e368811/architecture/code_change_map.md
  why: "Touch point 1 is THIS task: the bash append (disclosure + guarded bind + opt-out), the verbatim-emit rationale, and the byte-identity impact (TestEmbeddedCompletionsMatchOnDisk stays green). Confirms the file is the on-disk == emitted single source."
  section: "Touch point 1 (bash: completions/skilldozer.bash)"

- file: plan/005_d9b30e368811/P1M1T1S1/research/verified_facts.md
  why: "Direct-from-source proof: the 67-line live count (resolving the 69-line discrepancy), the verbatim-emit mechanic, the exact append, why the two gate tests stay green, and the scope boundary (bash only; locking test = P1.M1.T2.S1)."

- url: (PRD §14.7 + decision 22 — in PRD.md, READ-ONLY)
  why: "§14.7: list every match on the first Tab; bash show-all-if-ambiguous is off by default; `bind 'set show-all-if-ambiguous on'`; session-global ⇒ must disclose + provide opt-out (`bind 'set show-all-if-ambiguous off'`). Decision 22 locks first-Tab-list-all. Do NOT edit PRD.md."
- url: https://www.gnu.org/software/bash/manual/html_node/Readline-Init-File-Syntax.html
  why: "Documents `set show-all-if-ambiguous on` as a readline inputrc variable (also settable at runtime via `bind`). Confirms it is a session-global readline option (no per-command scope)."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls completions/skilldozer.bash main.go main_test.go go.mod
completions/skilldozer.bash   main.go   main_test.go   go.mod
# completions/skilldozer.bash: 67 lines. _skilldozer_completion() @ :20-66; registration @ :67.
#   Already has --shell + --link (plan/005 base). Last line: `complete -F _skilldozer_completion skilldozer`.
#   Embedded via //go:embed @ main.go:54-55; emitted verbatim by runCompletion (main.go:1499-1527).
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep).
# No new files. This subtask edits completions/skilldozer.bash ONLY (no .go file).
```

### Desired Codebase tree with files to be changed

```bash
completions/skilldozer.bash   # MODIFY — append disclosure comment + guarded `bind` line + commented opt-out (after line 67)
# main.go / main_test.go / go.mod / go.sum — UNCHANGED (//go:embed picks up the file edit on rebuild)
# completions/_skilldozer (zsh) / completions/skilldozer.fish — UNCHANGED (zsh = S2; fish = no change)
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `completions/skilldozer.bash` | Append §14.7 disclosure + active `bind` + opt-out | PRD §14.7 / decision 22 |

### Known Gotchas of our codebase & Library Quirks

```bash
# GOTCHA #1 — Line count is 67 LIVE (contract/map say 69 — stale). Trust the live file:
# the LAST LINE is `complete -F _skilldozer_completion skilldozer` (verified). Append AFTER it.
# Do not anchor an edit tool's oldText to a line number; anchor to that exact last line.

# GOTCHA #2 — bash is emitted VERBATIM (unlike zsh). runCompletion (main.go:1499-1527) emits
# bashCompletion byte-for-byte; only zsh is derived via zshEvalScript. So ONE file edit covers
# both the §14.5 source path and the §14.6 eval path. Do NOT look for a bash derivation step —
# there isn't one. (The zsh option is a main.go const edit → S2, not this task.)

# GOTCHA #3 — Rebuild before behavioral testing. //go:embed reads the file at COMPILE time.
# An already-built ./skilldozer holds the OLD bytes. go build (or go test, which compiles)
# re-embeds. `--completions --shell bash` only reflects the edit after a rebuild.

# GOTCHA #4 — The `[[ $- == *i* ]] &&` guard is LOAD-BEARING. `bind` in a non-interactive
# bash prints a warning ("warning: line ... cannot be used in non-interactive context" or
# similar). The guard (true only when `$-` contains `i`) silences it for eval test harnesses
# and non-interactive sourcing. Completions only matter interactively, so the option still
# applies where it counts. Do NOT drop the guard for "simplicity".

# GOTCHA #5 — Append AFTER the `complete -F` registration, not before. Both the registration
# and the `bind` run at source time; the order between them is not behaviorally critical, but
# appending at the end is the contract-specified placement and keeps the diff minimal/obvious.

# GOTCHA #6 — SCOPE: edit ONLY the bash file. zsh (completions/_skilldozer + main.go
# zshEvalRegistration const) is S2; the locking Go test is P1.M1.T2.S1; the README disclosure
# is P1.M1.T3; fish needs no change. Editing main.go or adding a test here is a scope violation.

# GOTCHA #7 — Do NOT add the asserting test in S1. The test that asserts `show-all-if-ambiguous on`
# in the emitted output is P1.M1.T2.S1 (next subtask). S1's automated gate is the EXISTING tests
# staying green + the manual CLI grep. Adding the test now collides with T2's scope.

# GOTCHA #8 — Keep the opt-out token literally `show-all-if-ambiguous off` (a comment). The
# P1.M1.T2.S1 test (per the architecture map) greps for BOTH `show-all-if-ambiguous on` (active)
# AND `show-all-if-ambiguous off` (opt-out token). If you omit the opt-out, that future test
# cannot pass. PRD §14.7 mandates the opt-out anyway (session-global ⇒ reversible).

# GOTCHA #9 — bash -n is the syntax gate (not shellcheck). The repo's tests are Go-level
# (embed-match + content), not shellcheck. Run `bash -n completions/skilldozer.bash` after
# editing to catch a broken `[[ ]]` or unbalanced quote. (The existing file already carries
# SC2317/SC2207 notes; shellcheck cleanliness is nice-to-have, not a gate.)

# GOTCHA #10 — No deps change. No .go file is edited. go.mod/go.sum byte-for-byte identical.
# The sole edit is appending comment + one active bind line + commented opt-out to a shell asset.
```

---

## Implementation Blueprint

### Data models and structure

None. This subtask appends to a shell completion script (a data asset embedded verbatim into the Go binary). No Go types or signatures change.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: APPEND the §14.7 block to completions/skilldozer.bash (after line 67)
  - ANCHOR the edit on the exact last line: `complete -F _skilldozer_completion skilldozer`
    (NOT on a line number — the live file is 67 lines, not the 69 the contract/map cite; GOTCHA #1).
  - APPEND (exact text in Implementation Patterns):
      (a) a blank line + the disclosure comment block (§14.7 intent; session-global;
          bash default OFF; guard rationale);
      (b) the active guarded line: `[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'`;
      (c) the commented opt-out: `#   bind 'set show-all-if-ambiguous off'`.
  - DO NOT modify the _skilldozer_completion function (line 20) or anything above the registration.
  - KEEP the registration line as the last non-append line (append after it).

Task 2: VERIFY — syntax, embed-match, existing tests, manual CLI grep
  - COMMAND: bash -n completions/skilldozer.bash && echo "syntax OK"   (GOTCHA #9)
  - COMMAND: go build ./...                                           (re-embeds; exit 0)
  - COMMAND: go test -run 'TestEmbeddedCompletionsMatchOnDisk|TestRunCompletionBashScript' -v ./...
                                                                      (both GREEN)
  - COMMAND: go test ./...                                            (no regression)
  - MANUAL: ./skilldozer --completions --shell bash 2>/dev/null | grep -q 'show-all-if-ambiguous on'
            ./skilldozer --completions --shell bash 2>/dev/null | grep -q 'show-all-if-ambiguous off'
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"
  - COMMAND: git diff --stat -- '*.go'                                (MUST be empty — GOTCHA #10)
```

### Implementation Patterns & Key Details

```bash
# Task 1 — the append (exact text). Anchor oldText on the registration line; newText adds the block.

#   OLD (the current last line):
complete -F _skilldozer_completion skilldozer

#   NEW (registration line + the appended §14.7 block):
complete -F _skilldozer_completion skilldozer

# --- §14.7 listing behavior (decision 22) -------------------------------------
# skilldozer wants every ambiguous match listed on the FIRST Tab — a manifest-free
# store (PRD §2) makes completion the primary discovery path, so candidates hidden
# behind a silent common-prefix halt are a UX defect. bash defaults to
# show-all-if-ambiguous OFF: the first Tab completes the common prefix and beeps,
# and the full list appears only on the second Tab.
#
# The line below sets show-all-if-ambiguous ON so all prefix matches list on the
# first Tab. This is a READLINE SESSION-GLOBAL option: it changes listing for EVERY
# command in this shell, not just skilldozer (there is no per-command scope). The
# `[[ $- == *i* ]] &&` guard keeps this quiet when the file is sourced
# non-interactively (e.g. an eval test harness): `bind` in a non-interactive shell
# prints a warning, which the guard silences. Completions only matter interactively,
# so the option still applies where it counts.
[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'
# Opt-out — restore bash's stock (second-Tab) listing:
#   bind 'set show-all-if-ambiguous off'
```

Notes:
- Wording of the disclosure comment is flexible (it's prose); the hard requirements are
  (1) it names `show-all-if-ambiguous`, (2) notes session-global, (3) the active guarded
  `bind ... on` line is present verbatim, (4) the commented `bind ... off` opt-out is
  present. The exact text above satisfies all four and is the recommended form.
- The `[[ $- == *i* ]] &&` guard: `$-` holds bash's option flags; `i` is set only in
  interactive shells. The glob `*i*` matches inside `[[ ]]`. Standard idiom (GOTCHA #4).

### Integration Points

```yaml
EMBED (the mechanism — NO edit):
  - main.go:54 //go:embed completions/skilldozer.bash  →  var bashCompletion string (main.go:55)
  - completionScript("bash") (main.go:1217) returns bashCompletion verbatim.
  - runCompletion (main.go:1499-1527) emits it verbatim (only zsh is derived).
  - Editing the on-disk file + go build/go test re-embeds the new bytes automatically.

TESTS (unchanged; they gate the change — no NEW test in S1):
  - TestEmbeddedCompletionsMatchOnDisk (main_test.go:3139): embedded == on-disk. GREEN after rebuild.
  - TestRunCompletionBashScript (main_test.go:3163): stdout has _skilldozer_completion + Go stderr
    empty. GREEN (the function def and Go stderr are unaffected by the append).

NO DATABASE / NO CONFIG / NO ROUTES / NO GO SOURCE:
  - This subtask edits exactly one shell data asset (append-only). No parseArgs, run(), usageText.
```

---

## Validation Loop

### Level 1: Shell syntax + edit presence (immediate, after the append)

```bash
cd /home/dustin/projects/skilldozer

bash -n completions/skilldozer.bash && echo "bash -n OK" || echo "FAIL: syntax error"
# Expected: "bash -n OK".

# Confirm the three elements are present:
grep -q -- 'show-all-if-ambiguous on' completions/skilldozer.bash && echo "active ON present" || echo "FAIL"
grep -q -- 'show-all-if-ambiguous off' completions/skilldozer.bash && echo "opt-out present" || echo "FAIL"
grep -q -- '\[\[ \$- == \*i\* \]\]' completions/skilldozer.bash && echo "guard present" || echo "FAIL"
tail -1 completions/skilldozer.bash   # Expected: the commented opt-out line (or the bind line, depending on trailing newline)
# Expected: all greps succeed; the append is in place after the registration line.
```

### Level 2: The embed-match gate + existing tests (automated, post-rebuild)

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"     # Expected: 0 (re-embeds the edited file)
go test -run 'TestEmbeddedCompletionsMatchOnDisk|TestRunCompletionBashScript' -v ./...
# Expected: both PASS (embed==on-disk for bash; stdout still has _skilldozer_completion; Go stderr empty).

go test ./... ; echo "test exit $?"       # Expected: 0 (no regression)
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
git diff --stat -- '*.go'                 # Expected: EMPTY (no .go file edited)
# Expected: build/test exit 0; deps unchanged; no .go diff.
```

### Level 3: The behavioral proof (the actual §14.7 contract — manual CLI grep)

```bash
cd /home/dustin/projects/skilldozer
go build -o /tmp/sdz . || { echo "FAIL: build"; exit 1; }

# The emitted eval-path output carries the active option + the opt-out token:
/tmp/sdz --completions --shell bash 2>/dev/null | grep -q 'show-all-if-ambiguous on'  && echo "emit ON OK"  || echo "FAIL"
/tmp/sdz --completions --shell bash 2>/dev/null | grep -q 'show-all-if-ambiguous off' && echo "emit opt-out OK" || echo "FAIL"

# The emitted bytes are byte-identical to the on-disk file (verbatim emit — one edit, both paths):
diff <(/tmp/sdz --completions --shell bash 2>/dev/null) completions/skilldozer.bash && echo "emit == on-disk OK" || echo "FAIL: divergence"

# Control: the completion function + registration are intact (unchanged):
/tmp/sdz --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo "function intact OK" || echo "FAIL"

rm -f /tmp/sdz
# Expected: every line "...OK"; the emit==on-disk diff is empty (verbatim emit holds).
```

### Level 4: Scope-discipline check (only the bash file changed)

```bash
cd /home/dustin/projects/skilldozer

git diff --name-only
# Expected: ONLY completions/skilldozer.bash. (If main.go, completions/_skilldozer,
# completions/skilldozer.fish, README.md, or any test appears, you over-reached into
# S2 / P1.M1.T2 / P1.M1.T3's scope.)

# zsh/fish embed-match still holds (they're unchanged):
go test -run TestEmbeddedCompletionsMatchOnDisk -v ./... 2>&1 | grep -E 'bash|zsh|fish|PASS|FAIL'
# Expected: all three shells PASS (bash now matches the appended file; zsh/fish unchanged).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `bash -n` clean; greps confirm active `on`, opt-out `off`, and the `*i*` guard
- [ ] Level 2 PASS — `go build` exit 0; `TestEmbeddedCompletionsMatchOnDisk` + `TestRunCompletionBashScript` GREEN; `go test ./...` exit 0; deps unchanged; no `.go` diff
- [ ] Level 3 PASS — emitted output contains `show-all-if-ambiguous on` and `off`; emit == on-disk (verbatim holds); function intact
- [ ] Level 4 PASS — only `completions/skilldozer.bash` changed; embed-match passes for all three shells

### Feature Validation
- [ ] the file ends with: disclosure comment block + `[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'` + commented opt-out
- [ ] the disclosure names `show-all-if-ambiguous`, notes session-global, notes bash default OFF
- [ ] the active line and the opt-out use the exact tokens `show-all-if-ambiguous on` / `off` (the T2 test greps for both)

### Code Quality / Convention Validation
- [ ] append-only; the `_skilldozer_completion` function and `complete -F` registration are unchanged
- [ ] no Go file edited (the `//go:embed` picks up the change); `go.mod`/`go.sum` byte-for-byte identical
- [ ] the `bind` line is guarded for interactivity (`[[ $- == *i* ]] &&`)

### Scope Discipline
- [ ] Did NOT edit `main.go` or any `.go` file (zsh `zshEvalRegistration` const is S2)
- [ ] Did NOT edit `completions/_skilldozer` (zsh) or `completions/skilldozer.fish`
- [ ] Did NOT add a Go test (the byte-level assertion is P1.M1.T2.S1)
- [ ] Did NOT edit the README (the §14.7 disclosure is P1.M1.T3.S1, Mode B)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't anchor the edit on the "69-line" count.** The live file is 67 lines. Anchor `oldText` on the exact last line (`complete -F _skilldozer_completion skilldozer`), not a line number.
- ❌ **Don't edit any `.go` file.** bash is emitted verbatim; the `//go:embed` picks up the file change on rebuild. Looking for a bash-derivation step (like zsh's) is a false trail — there isn't one.
- ❌ **Don't skip the rebuild.** A pre-built binary holds stale embedded bytes. `go build` (or `go test`) re-embeds; an already-built `./skilldozer` does not.
- ❌ **Don't drop the `[[ $- == *i* ]] &&` guard.** Without it, `bind` warns when the file is sourced non-interactively (eval harnesses). The guard silences that; the option still applies in interactive shells where completions matter.
- ❋ **Don't omit the opt-out token.** PRD §14.7 mandates it (session-global ⇒ reversible), and the P1.M1.T2.S1 test will grep for `show-all-if-ambiguous off`. Omitting it fails §14.7 and blocks the next subtask.
- ❌ **Don't add the asserting Go test here.** The test that locks `show-all-if-ambiguous on` in the emitted output is P1.M1.T2.S1. S1's gate is existing tests staying green + the manual grep.
- ❌ **Don't touch zsh or fish.** zsh's option is a `main.go` const edit (S2); fish lists by default (no change). Editing them now over-reaches into sibling subtasks.
- ❋ **Don't paraphrase the option tokens.** Use the exact strings `show-all-if-ambiguous on` / `show-all-if-ambiguous off`. The disclosure comment's prose is flexible, but those two tokens (and the guard) are grepped by the next subtask's test.
- ❌ **Don't add deps.** No `.go` file is edited; the sole change is appending to a shell asset.

---

## Confidence Score

**9/10** — The edit is an append anchored on a verified exact last line; the verbatim-emit path is confirmed in `runCompletion` (one file edit covers both delivery paths); the two gate tests provably stay green (embed-match moves with the file; the function marker and Go stderr are unaffected); and the exact tokens the next subtask's test will grep for (`show-all-if-ambiguous on`/`off` + the `*i*` guard) are specified verbatim. The 1-point reservation is the disclosure comment's prose (flexible wording) — the PRP gives a complete recommended block but the exact phrasing is the implementer's call, bounded by the four hard requirements (names the option, session-global, active `on` line, commented `off` opt-out).
