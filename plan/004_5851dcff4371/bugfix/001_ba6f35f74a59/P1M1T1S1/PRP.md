# PRP — P1.M1.T1.S1: Add `--shell` value routing + flag advertisement to the bash completion file

> **Subtask:** P1.M1.T1.S1 — the bash third of P1.M1.T1 (Issue 1: `--shell` value completion offers skill tags instead of `bash zsh fish`).
> **Scope boundary:** Edits ONLY `completions/skilldozer.bash` (the bash completion script, embedded via `//go:embed`). Does NOT touch the zsh file (`completions/_skilldozer` → S2) or the fish file (`completions/skilldozer.fish` → S3); does NOT touch any `.go` file (the `//go:embed` picks up the edit automatically); does NOT change `usageText` (it already documents `--shell`); does NOT edit the README (Mode B sweep is P1.M3.T1).

---

## Goal

**Feature Goal**: Make `skilldozer --shell <TAB>` offer exactly `bash zsh fish` (PRD §14.2's fixed enum, nothing else) instead of skill tags, and make `--shell` discoverable via `skilldozer --<TAB>` — in the **bash** completion script.

**Deliverable**: Edits to `completions/skilldozer.bash` only (no new files):
1. **Value routing** (~line 39): add a `--shell)` case to the `case "$prev" in` block → `COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0`.
2. **Flag advertisement** (~line 45): add `--shell` to the `compgen -W` flag list (decision D7 — `--shell` is a real, documented flag).
3. **In-file doc accuracy**: add a `--shell` line to the value-routing comment block (~lines 33-36) and a `--shell` note to the LOCKSTEP header (~lines 14-17).

**Success Definition**: After `go build`, `TestEmbeddedCompletionsMatchOnDisk` passes (embedded bytes == on-disk file); the contract repro yields `COMPREPLY=[bash zsh fish]`; `skilldozer --<TAB>` offers `--shell`; no existing test regresses; `go.mod`/`go.sum` unchanged.

---

## User Persona (if applicable)

**Target User**: bash users who tab-complete `skilldozer` invocations (especially the `skilldozer --completions --shell <shell> | source` install idiom).

**Use Case**: A user types `skilldozer --shell ` and tabs to pick the shell.

**Pain Points Addressed**: Today tab after `--shell` offers skill tags (the opposite of §14.2's "nothing else"), and `--shell` isn't even offered after `--`, so users must know the flag exists.

---

## Why

- **PRD §14.2**: "`--shell` takes a fixed enum (`bash`/`zsh`/`fish`); offer those three words, nothing else." Today the bash file has no `--shell` case, so the value slot falls through to tag completion — a spec deviation.
- **PRD §14.4 lockstep**: completion files are frozen to `parseArgs()`, which accepts `--shell`. The bash file is missing it.
- **Decision D7**: `--shell` is a real, documented flag (in `usageText` OPTIONS, used in the canonical install idiom). Its omission from the advertised flag list is inconsistent, so D7 adds it for discoverability. (PRD §14.6's 13-flag table omits `--shell` — a noted tension; D7 resolves it in favor of consistency. That table can be updated separately if ever desired.)

---

## What

`completions/skilldozer.bash` gains `--shell` handling in two places:

- **Value routing**: a new `--shell)` case in the `case "$prev" in` block offers the fixed enum `bash zsh fish` and returns (no fall-through to tags).
- **Flag advertisement**: `--shell` appears in the `compgen -W` long-form flag list, so `skilldozer --<TAB>` offers it.

No behavior change for any other flag; the tag-completion default, the `--search`/`--store`/`--init` routing, and the long-form-only policy are untouched.

### Success Criteria

- [ ] `case "$prev" in` has a `--shell)` case: `COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;`
- [ ] the `compgen -W` flag list string contains `--shell` (14 flags)
- [ ] the value-routing doc comment + LOCKSTEP header mention `--shell`
- [ ] `TestEmbeddedCompletionsMatchOnDisk` passes (embedded == on-disk bash file)
- [ ] the contract repro yields `COMPREPLY=[bash zsh fish]`; `skilldozer --<TAB>` offers `--shell`
- [ ] no existing test regresses; `go.mod`/`go.sum` unchanged; no `.go` file edited

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact current text of both edit sites (the `case` block and the flag list), the exact target additions, the embed/rebuild mechanics, the automated gate (TestEmbeddedCompletionsMatchOnDisk), the manual repro, and the scope boundary (bash only) are all specified with line numbers. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
- file: completions/skilldozer.bash
  why: "THE edit site (the only file touched). case \"$prev\" in @ :37-40 (add --shell) after the --store|--init) line :39). compgen -W flag list @ :44-46 (add --shell to the 13-flag string :45). Value-routing doc comment @ :33-36 + LOCKSTEP header @ :11-17 (mention --shell). 62 lines total."
  pattern: "Mirror the existing --store|--init) case shape: 'COMPREPLY=($(compgen -W \"...\" -- \"$cur\")); return 0'. The only difference is the word list (bash zsh fish) instead of compgen -d."
  gotcha: "Order in the case block is irrelevant (--shell doesn't overlap the other patterns), but place it right after --store|--init) per the contract. The flag-list position is also not behaviorally significant (compgen -W prefix-matches); group --shell after --store for tidiness."

- file: main.go
  why: "The //go:embed wiring (NO edit — just confirming it picks up the file change automatically). :54 //go:embed completions/skilldozer.bash; :55 var bashCompletion string; :1116 completionScript(\"bash\") returns bashCompletion. Editing the on-disk file + go build/go test re-embeds the new bytes."
  pattern: "Do NOT touch main.go. The embed is the mechanism, not an edit site."
  gotcha: "A PRE-BUILT binary holds the OLD embedded bytes. Always rebuild (go build / go test) before behavioral testing."

- file: main_test.go
  why: "The automated gate: TestEmbeddedCompletionsMatchOnDisk @ :2995 asserts completionScript(\"bash\") == on-disk completions/skilldozer.bash (byte identity, PRD §14.6). go test re-embeds at compile, so the test passes automatically after the edit. TestRunCompletionBashScript @ :3020 checks --completions --shell bash stdout contains _skilldozer_completion (still true). No test asserts the flag list / case content, so no test breaks."
  pattern: "The embed-match test is the automated regression gate; the bash repro (Level 3) is the behavioral gate."

- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/decisions.md
  why: "§D7 is the decision to ADD --shell to the advertised flag list (not just value routing) — resolves the PRD §14.6 (13-flag table) vs §14.2 (value enum) tension in favor of consistency/discoverability."
  section: "D7 (Issue 1 — --shell Added to Flag Advertisement List)"

- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/architecture/issue_analysis.md
  why: "§Issue 1 is the authoritative bug writeup: the repro (bash COMP_WORDS), confirmation --shell is MISSING from all three files today, and the per-file fix prescription (the bash case line + the flag-list addition)."
  section: "Issue 1 (--shell value completion offers skill tags)"

- file: plan/004_5851dcff4371/bugfix/001_ba6f35f74a59/P1M1T1S1/research/verified_facts.md
  why: "Direct-from-source proof: exact current text of every site, the target additions, the embed/rebuild mechanic, the no-test-breaks check, and the scope boundary (bash only; zsh/fish are S2/S3)."

- url: (PRD §14.2 / §14.4 — in PRD.md, READ-ONLY)
  why: "§14.2: --shell takes the fixed enum bash/zsh/fish, offer those three words nothing else. §14.4: completion files frozen to parseArgs (lockstep). Do NOT edit PRD.md."
- url: https://www.gnu.org/software/bash/manual/html_node/Programmable-Completion-Builtins.html
  why: "compgen -W wordlist -- cur is the bash primitive for offering a fixed word list (used by the existing --store|--init case and reused here for the enum)."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls completions/skilldozer.bash main.go main_test.go go.mod
completions/skilldozer.bash   main.go   main_test.go   go.mod
# completions/skilldozer.bash: 62 lines. case "$prev" in @ :37-40 (--search, --store|--init; NO --shell).
#   compgen -W flag list @ :44-46 (13 flags, no --shell). Embedded via //go:embed @ main.go:54-55.
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep).
# No new files. This subtask edits completions/skilldozer.bash ONLY (no .go file).
```

### Desired Codebase tree with files to be changed

```bash
completions/skilldozer.bash   # MODIFY — add --shell case (value routing) + --shell in flag list + 2 doc-comment touches
# main.go / main_test.go / go.mod / go.sum — UNCHANGED (//go:embed picks up the file edit on rebuild)
# completions/_skilldozer (zsh) / completions/skilldozer.fish — UNCHANGED here (S2/S3)
```

**File responsibilities:**
| File | Change | Owner |
|---|---|---|
| `completions/skilldozer.bash` | `--shell` value routing + advertisement + doc accuracy | PRD §14.2 + decision D7 |

### Known Gotchas of our codebase & Library Quirks

```bash
# GOTCHA #1 — A pre-built binary holds STALE embedded bytes. //go:embed reads the file
# at COMPILE time. If you edit completions/skilldozer.bash and run an already-built
# ./skilldozer, --completions --shell bash emits the OLD script (no --shell). Always
# rebuild (go build, or just go test which compiles) before behavioral testing.

# GOTCHA #2 — The automated gate is byte-identity, not behavior. TestEmbeddedCompletionsMatchOnDisk
# asserts completionScript("bash") == the on-disk file. It passes as long as you rebuild
# after editing. It does NOT assert the --shell case exists — so the manual repro (Level 3)
# is the behavioral gate that proves the routing actually works.

# GOTCHA #3 — Rebuild for the embed, but DO NOT edit any .go file. The //go:embed directive
# at main.go:54 is the mechanism; it picks up the file change automatically. Adding --shell
# to the bash file requires ZERO Go changes. (parseArgs already accepts --shell — that's
# why §14.4 says completions are frozen to parseArgs.)

# GOTCHA #4 — The `return 0` on the --shell case is LOAD-BEARING. Without it, prev="--shell"
# would still fall through to the tag-completion default (line 59) and offer skill tags.
# Every case in the block ends with `return 0` for exactly this reason.

# GOTCHA #5 — Place --shell AFTER --store|--init) in the case (per contract). Order is
# technically irrelevant (--shell doesn't overlap --search or --store|--init), but follow
# the contract's placement. In the flag LIST, exact position is also irrelevant (compgen -W
# prefix-matches); group --shell after --store (value-taking flags together) for tidiness.

# GOTCHA #6 — SCOPE: edit ONLY the bash file. zsh (completions/_skilldozer) is S2; fish
# (completions/skilldozer.fish) is S3. After S1 alone the three files temporarily diverge
# (bash has --shell, zsh/fish don't). That's EXPECTED — S1/S2/S3 are a sequence and §14.4
# lockstep is restored when all three land. There is NO cross-file lockstep test, so editing
# only bash breaks no test. Do NOT edit zsh/fish here.

# GOTCHA #7 — Do NOT change usageText. It already lists --shell in OPTIONS (D7: "--shell is
# a real, documented flag in usageText OPTIONS"). The gap is ONLY the completion file.

# GOTCHA #8 — Keep the enum order "bash zsh fish" (matches PRD §14.2 and the contract repro).
# It's also the order completionScript/detectShell use; consistency avoids user surprise.

# GOTCHA #9 — No deps change. No .go file is edited, so go.mod/go.sum are byte-for-byte
# identical. The sole edited file is a shell data asset.
```

---

## Implementation Blueprint

### Data models and structure

None. This subtask edits a shell completion script (a data asset embedded into the Go binary). No Go types, no signatures change.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: ADD the --shell value-routing case (completions/skilldozer.bash, ~line 39)
  - EDIT the `case "$prev" in` block: AFTER the `--store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;` line,
    ADD:
        --shell) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;
  - WHY: routes the --shell value slot to the fixed enum (§14.2); `return 0` prevents fall-through to tags (GOTCHA #4).
  - EXACT oldText/newText in Implementation Patterns below.

Task 2: ADD --shell to the flag-advertisement list (~line 45)
  - EDIT the compgen -W string: insert `--shell` after `--store` (groups value-taking flags).
    OLD: "--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions"
    NEW: "--version --help --path --list --all --file --relative --no-color --search --store --shell --check --init --completions"
  - WHY: D7 — --shell is a real documented flag; make it discoverable via --<TAB>.

Task 3: ADD a --shell line to the value-routing doc comment (~lines 33-36)
  - EDIT the comment block: add a line documenting the --shell routing, e.g.:
        #   --shell         -> fixed enum      -> offer "bash zsh fish" via compgen -W.
  - WHY: Mode A — keep the in-file doc accurate (the block already documents --search and --store/--init).

Task 4: UPDATE the LOCKSTEP header (~lines 14-17) to mention --shell
  - EDIT the header note (currently "Updated for --check/--init/--completions (decision 19): ..."):
    append/adjust so it notes --shell's value completes to the bash/zsh/fish enum (§14.2)
    and --shell is advertised (D7). Keep the LOCKSTEP + long-form-only + decision-19 content intact.
  - WHY: contract DOCS §5 — "Verify it still mentions all flags including --shell."

Task 5: VERIFY — embed-match test + behavioral repro + no regression
  - COMMAND: go build ./...                                  (exit 0; re-embeds the edited file)
  - COMMAND: go test -run TestEmbeddedCompletionsMatchOnDisk -v ./...   (PASS — embedded == on-disk)
  - COMMAND: go test ./...                                   (no regression — the bash content tests still pass)
  - MANUAL: the contract repro (Level 3) → COMPREPLY=[bash zsh fish]
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"   (GOTCHA #9)
  - COMMAND: git diff --stat -- '*.go'                       (MUST be empty — no .go file edited)
```

### Implementation Patterns & Key Details

```bash
# Task 1 — the value-routing case (exact oldText → newText). The `return 0` is load-bearing.

#   OLD (the two existing cases, lines 38-39):
        --search) return 0 ;;
        --store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
#   NEW (add the --shell case right after --store|--init):
        --search) return 0 ;;
        --store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
        --shell) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;

# Task 2 — the flag list (exact oldText → newText). --shell inserted after --store.

#   OLD (line 45):
            "--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions" \
#   NEW:
            "--version --help --path --list --all --file --relative --no-color --search --store --shell --check --init --completions" \

# Task 3 — value-routing doc comment (add a --shell line). Example:
#   OLD:
    #   --search        -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
    #   --store/--init  -> directory value  -> complete DIRECTORIES via compgen -d.
#   NEW (append a --shell line):
    #   --search        -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
    #   --store/--init  -> directory value  -> complete DIRECTORIES via compgen -d.
    #   --shell         -> fixed enum       -> offer "bash zsh fish" via compgen -W.

# Task 4 — LOCKSTEP header. Keep lines 11-13 (the freeze/lockstep statement) and the
# long-form-only note; extend the decision-19 note (lines 15-17) to also mention --shell,
# e.g. append: "--shell's value completes to the bash/zsh/fish enum (§14.2); --shell is
# advertised (D7)." Wording is flexible; the requirement is that the header remains
# accurate and mentions --shell.
```

### Integration Points

```yaml
EMBED (the mechanism — NO edit):
  - main.go:54 //go:embed completions/skilldozer.bash  →  var bashCompletion string (main.go:55)
  - completionScript("bash") returns bashCompletion (main.go:1116); runCompletion emits it for --completions.
  - Editing the on-disk file + go build/go test re-embeds the new bytes automatically.

TESTS (unchanged; they gate the change):
  - TestEmbeddedCompletionsMatchOnDisk (main_test.go:2995): embedded == on-disk. PASS after rebuild.
  - TestRunCompletionBashScript (main_test.go:3020): --completions --shell bash stdout has
    _skilldozer_completion. Still true (the marker is unchanged).

NO DATABASE / NO CONFIG / NO ROUTES / NO GO SOURCE:
  - This subtask edits exactly one shell data asset. No parseArgs, no run(), no usageText.
```

---

## Validation Loop

### Level 1: Syntax & shell sanity (immediate, after the edits)

```bash
cd /home/dustin/projects/skilldozer

# bash syntax-check the edited file (catches a broken case/esac or unbalanced quote):
bash -n completions/skilldozer.bash && echo "bash -n OK" || echo "FAIL: syntax error"
# Expected: "bash -n OK".

# Confirm the two edits are present:
grep -n -- '--shell) COMPREPLY=' completions/skilldozer.bash          # Expected: 1 hit (the new case)
grep -c -- '--shell' completions/skilldozer.bash                      # Expected: >=3 (case + flag list + 2 doc comments)
grep -n -- '--search --store --shell --check' completions/skilldozer.bash   # Expected: 1 hit (flag list)
# Expected: bash -n clean; the greps find the additions.
```

### Level 2: The embed-match gate (the automated regression check)

```bash
cd /home/dustin/projects/skilldozer

go build ./...     ; echo "build exit $?"    # Expected: 0 (re-embeds the edited file)
go test -run TestEmbeddedCompletionsMatchOnDisk -v ./...
# Expected: PASS — completionScript("bash") == on-disk completions/skilldozer.bash.

# Whole module: the bash content tests + everything else still green:
go test ./... ; echo "test exit $?"          # Expected: 0 (no regression)
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
git diff --stat -- '*.go'                    # Expected: EMPTY (no .go file edited)
# Expected: build/test exit 0; deps unchanged; no .go diff.
```

### Level 3: The behavioral repro (the actual contract — Issue 1 fixed)

```bash
cd /home/dustin/projects/skilldozer
go build -o /tmp/sdz . || { echo "FAIL: build"; exit 1; }

# Issue 1 repro: after --shell, offer exactly bash zsh fish (NOT skill tags).
bash -c 'eval "$(/tmp/sdz --completions --shell bash)"; \
  COMP_WORDS=(skilldozer --shell ""); COMP_CWORD=2; _skilldozer_completion; \
  echo "COMPREPLY=[${COMPREPLY[*]}]"'
# Expected: COMPREPLY=[bash zsh fish]
# (Pre-fix this printed skill tags like [example foo writing/reddit].)

# Advertisement: skilldozer --<TAB> now offers --shell among the flags.
bash -c 'eval "$(/tmp/sdz --completions --shell bash)"; \
  COMP_WORDS=(skilldozer --); COMP_CWORD=1; _skilldozer_completion; \
  echo "${COMPREPLY[*]}"' | tr ' ' '\n' | grep -qx -- '--shell' && echo "--shell advertised OK" || echo "FAIL: --shell not in -<tab>"
# Expected: "--shell advertised OK".

# Control: bare tag completion still works (unchanged default branch).
bash -c 'eval "$(/tmp/sdz --completions --shell bash)"; \
  COMP_WORDS=(skilldozer ""); COMP_CWORD=1; _skilldozer_completion; \
  echo "tags=[${COMPREPLY[*]}]"'
# Expected: the skill tags from the store (proves the --shell case didn't break the default).

rm -f /tmp/sdz
# Expected: COMPREPLY=[bash zsh fish]; --shell advertised; tag completion intact.
```

### Level 4: Lockstep-awareness check (scope discipline)

```bash
cd /home/dustin/projects/skilldozer

# S1 edits ONLY bash. zsh/fish are S2/S3 (separate subtasks). Confirm this subtask did
# NOT touch them (the temporary cross-file divergence is expected and resolved by S2/S3):
git diff --name-only
# Expected: ONLY completions/skilldozer.bash. (If zsh/fish appear, you over-reached into S2/S3.)

# The embed-match test still passes for zsh/fish (they're unchanged, so embedded==on-disk trivially):
go test -run TestEmbeddedCompletionsMatchOnDisk -v ./... 2>&1 | grep -E 'zsh|fish|bash|PASS|FAIL'
# Expected: all three shells PASS (bash now matches the edited file; zsh/fish unchanged).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `bash -n` clean; greps confirm the `--shell)` case, the flag-list entry, and the doc lines
- [ ] Level 2 PASS — `go build` exit 0; `TestEmbeddedCompletionsMatchOnDisk` PASS; `go test ./...` exit 0; `git diff go.mod go.sum` → "deps unchanged"; no `.go` diff
- [ ] Level 3 PASS — repro yields `COMPREPLY=[bash zsh fish]`; `--shell` advertised via `--<TAB>`; tag completion still works
- [ ] Level 4 PASS — only `completions/skilldozer.bash` changed; embed-match passes for all three shells

### Feature Validation
- [ ] `case "$prev" in` has the `--shell)` case (enum + `return 0`)
- [ ] the `compgen -W` flag list contains `--shell` (14 flags)
- [ ] value-routing doc comment + LOCKSTEP header mention `--shell`
- [ ] the enum is exactly `bash zsh fish` (order per §14.2)

### Code Quality / Convention Validation
- [ ] the `--shell)` case mirrors the existing `--store|--init)` case shape (`COMPREPLY=($(compgen ...)); return 0`)
- [ ] no Go file edited (the `//go:embed` picks up the change); `go.mod`/`go.sum` byte-for-byte identical
- [ ] minimal diff (one case line, one list token, two doc-comment touches)

### Scope Discipline
- [ ] Did NOT touch `completions/_skilldozer` (zsh → S2) or `completions/skilldozer.fish` (fish → S3)
- [ ] Did NOT edit any `.go` file (main.go //go:embed, parseArgs, run(), usageText all unchanged)
- [ ] Did NOT edit the README (Mode B sweep is P1.M3.T1)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't skip the rebuild.** A pre-built binary holds stale embedded bytes. `go build` (or `go test`) re-embeds; an already-built `./skilldozer` does not.
- ❌ **Don't drop the `return 0`** on the `--shell)` case. Without it, `prev="--shell"` falls through to tag completion — the exact bug you're fixing.
- ❌ **Don't edit any `.go` file.** The `//go:embed` is the mechanism; it picks up the file change automatically. Adding `--shell` to the bash file requires zero Go changes (parseArgs already accepts `--shell`).
- ❌ **Don't edit zsh or fish here.** They are S2/S3. Editing them now over-reaches and creates review noise; the §14.4 lockstep is restored when all three land (S1→S2→S3).
- ❋ **Don't forget the doc comments.** The value-routing block (33-36) and LOCKSTEP header (11-17) should mention `--shell` — leaving them stale makes the file lie about itself (Mode A).
- ❌ **Don't reorder the enum.** Use `bash zsh fish` (PRD §14.2 + the contract repro). A different order isn't wrong functionally but diverges from the spec and the other shells.
- ❌ **Don't change usageText.** It already documents `--shell` (D7). The gap is only the completion file.
- ❌ **Don't add deps.** No `.go` file is edited; the sole change is a shell data asset.

---

## Confidence Score

**9/10** — Every edit site is pinned to a verified live line with exact before/after text; the `--shell)` case mirrors an existing case shape one line above it; the embed/rebuild mechanic is confirmed (TestEmbeddedCompletionsMatchOnDisk gates byte-identity); no existing test asserts the bash content so nothing regresses; and the bash repro is the deterministic behavioral proof. The 1-point reservation is the LOCKSTEP-header wording (Task 4), which is flexible prose — the contract requires only that it "mentions all flags including --shell," so the PRP gives a concrete suggestion but leaves exact phrasing to the implementer.
