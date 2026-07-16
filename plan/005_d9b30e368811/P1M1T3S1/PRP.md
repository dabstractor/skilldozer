# PRP — P1.M1.T3.S1: README — disclose the §14.7 option set + opt-out one-liners

> **Subtask:** P1.M1.T3.S1 — the Mode B changeset-level documentation half of P1.M1 (§14.7: list every ambiguous match on the first Tab). It documents in the README what the parallel **T1.S1** (bash `show-all-if-ambiguous`, committed `5cf81d4`) + **T1.S2** (zsh `NO_LIST_AMBIGUOUS`, committed `b433a08`) already ship in the emitted completion scripts. The other disclosure half (the emitted scripts' own comments) rode with T1 as Mode A; this is the user-facing README disclosure that PRD §14.7 + §15 mandate.
> **Scope boundary:** **README.md ONLY** — a pure insertion of one disclosure paragraph + a per-shell option list + one opt-out fenced code block into the *Shell completions* section. Does NOT touch `main.go`, `completions/*`, `main_test.go`, or any `.go`/`go.mod` file. It edits no existing README line (inserts between two existing paragraphs).
> **Dependency status:** T1.S1 + T1.S2 are COMMITTED (HEAD `b433a08`), so the option strings the README discloses are already in the tree (verified — see §1). This task does NOT depend on the parallel T2.S1 (tests) landing.

---

## Goal

**Feature Goal**: Make the README's *Shell completions* section honestly disclose the session-global listing option the emitted scripts set (so the first `<tab>` lists every ambiguous match instead of halting at the common prefix), name the exact option per shell, note it affects every command in the shell, and give the one-line opt-out for users who prefer stock behavior — fulfilling PRD §14.7 / §15's "disclose in the README" requirement.

**Deliverable**: A pure insertion into `README.md` (no other file): a disclosure paragraph + a 3-bullet per-shell option list + an opt-out fenced `bash` block, placed between the existing namespace-safety paragraph (line 334 "...unambiguous.") and the "Prefer to copy the file instead?" paragraph (line 336).

**Success Definition**: `grep -q 'show-all-if-ambiguous' README.md && grep -q 'NO_LIST_AMBIGUOUS' README.md && grep -q 'LIST_AMBIGUOUS' README.md` all match; no source/test/`go.mod` file changes; the existing README structure (eval one-liners, flag list, manual-copy paths, fish source line) is byte-intact; the new prose matches the README's existing tone (user-facing, backtick-quoted commands, no internal `§X.Y` refs).

---

## User Persona (if applicable)

**Target User**: any user who runs `eval "$(skilldozer --completions)"` and then notices their shell's tab-completion lists ambiguous matches everywhere (not just for skilldozer). They need to know (a) skilldozer set it, (b) why, and (c) how to undo it.

**Use Case**: reading the README's completions section before/after installing completions, to understand the side effect and find the opt-out.

**User Journey**: install completions (eval line) → notice global listing change → re-read README → find the disclosure + opt-out one-liner → restore stock behavior if preferred.

**Pain Points Addressed**: a session-global shell option silently changed by a tool is surprising/hostile without disclosure. PRD §14.7 explicitly requires the coupling be visible and reversible.

---

## Why

- **PRD §14.7** mandates the emitted scripts set the list-ambiguous option AND "disclose the change in the emitted script's comments and in the README (§15), naming the exact option set" AND "provide the one-line opt-out". The code/comment half (T1) is done; this is the README half.
- **Decision 22** (decisions log) frames it as "disclosed + opt-out-able, since these are session-global options" — the README is the disclosure surface users actually read.
- **The store is manifest-free (§2)** → completion is the primary skill-discovery path, so hiding ambiguous candidates is a UX defect the option fixes. The README explains *why* the global coupling is worth it, so users tolerate it (or opt out informedly).
- **Touch point 4 (code_change_map.md)** is the authoritative spec for this exact insertion.

---

## What

A single insertion into `README.md` between line 334 ("unambiguous.") and line 336 ("Prefer to copy the file instead?"). The new block contains, per contract LOGIC (a)-(d):

1. **Disclosure paragraph** (a): the emitted script sets a shell option so an ambiguous prefix lists **every** match on the first `<tab>` instead of freezing at the common prefix; note WHY (manifest-free store → completion is discovery).
2. **Session-global note** (c): it changes listing for *every* command in that shell, not just `skilldozer`; set only when you load skilldozer completions (eval/source).
3. **Per-shell option list** (b): bash `show-all-if-ambiguous` (on); zsh `NO_LIST_AMBIGUOUS` (on); fish lists by default (no option set).
4. **Opt-out fenced block** (d): a ` ```bash ` block with `bind 'set show-all-if-ambiguous off'` (bash) and `setopt LIST_AMBIGUOUS` (zsh).

### Success Criteria

- [ ] `grep -q 'show-all-if-ambiguous' README.md` matches
- [ ] `grep -q 'NO_LIST_AMBIGUOUS' README.md` matches
- [ ] `grep -q 'LIST_AMBIGUOUS' README.md` matches
- [ ] The disclosure states the option is **session-global** and affects **every command**, set only on completion load
- [ ] The disclosure names bash `show-all-if-ambiguous` + zsh `NO_LIST_AMBIGUOUS`; notes fish lists by default
- [ ] The opt-out block gives `bind 'set show-all-if-ambiguous off'` (bash) + `setopt LIST_AMBIGUOUS` (zsh)
- [ ] NO existing README line is edited — pure insertion between line 334 and 336
- [ ] NO `.go` / `completions/*` / `main_test.go` / `go.mod` file changes
- [ ] The new prose mirrors the README's tone (backtick-quoted commands, em-dashes, no `§X.Y` refs)

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact insertion point (between README lines 334 and 336) is verified; the exact option strings to disclose are read directly from the committed emitted sources (§1); the contract's four content requirements (a-d) are mapped to concrete text (§3); the README tone conventions to mirror are documented (§4); and a tone-matched draft block is provided (§6). An implementer who has never seen this repo can do it in one pass by inserting the block at the named anchor.

### Documentation & References

```yaml
# MUST READ — THE source of truth (insertion point + exact option strings + draft block)
- file: plan/005_d9b30e368811/P1M1T3S1/research/verified_facts.md
  why: "THE inventory. §0 = current state (T1.S1+T1.S2 committed at HEAD b433a08; this task depends on the
        scripts, NOT on T2.S1). §1 = the EXACT option strings the emitted scripts carry (the strings the
        README must name). §2 = the verified insertion point (after line 334, before 336). §3 = contract
        LOGIC a-d mapped to text. §4 = README tone conventions. §6 = a tone-matched draft block."
  critical: "§1 (exact strings) prevents naming the wrong option; §2 (insertion anchor) prevents editing an
             existing line. §6's draft is copy-pasteable (adjust wording to taste)."

# MUST READ — the authoritative spec for this exact insertion
- file: plan/005_d9b30e368811/architecture/code_change_map.md
  why: "Touch point 4 pins the file (README.md), the section (Shell completions, 290-366), the insertion
        point (after the bullet list / namespace para ~334, before 'Prefer to copy' ~336), the content
        (session-global option, per-shell names, every-command scope, opt-out one-liners), and the verify
        grep. This PRP transcribes it; the change map is the contract."
  section: "Touch point 4 — README disclosure (Mode B)."

# MUST READ — confirms what the emitted scripts already do (so the README is accurate)
- file: completions/skilldozer.bash
  why: "READ-ONLY. Line 83 `[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'` (active) + line 85
        `#   bind 'set show-all-if-ambiguous off'` (opt-out). The README must name `show-all-if-ambiguous`
        and give the opt-out `bind 'set show-all-if-ambiguous off'` — mirroring these exact strings."
- file: main.go
  why: "READ-ONLY. The `zshEvalRegistration` const (main.go:~1260) carries `setopt NO_LIST_AMBIGUOUS`
        (active) + `#   setopt LIST_AMBIGUOUS` (opt-out). The README must name `NO_LIST_AMBIGUOUS` and
        give the opt-out `setopt LIST_AMBIGUOUS` — mirroring these exact strings."

# READ-ONLY — the README itself (the edit target + the tone to mirror)
- file: README.md
  why: "THE edit target. Shell completions section @290-366. Insertion anchor: line 334 ('unambiguous.')
        → before line 336 ('Prefer to copy the file instead?'). Mirror the section's tone: backtick-quoted
        commands, **bold** shell headers, em-dashes, `bash`-fenced code blocks, user-facing prose with NO
        internal §X.Y references."
  pattern: "The eval one-liners (302-310), the bullet list (317-328), the namespace paragraph (332-334),
            and the manual-copy paths (341-365) are the existing structures to PRESERVE and to match in style."

# READ-ONLY — the PRD authority
- file: PRD.md
  why: "READ-ONLY. §14.7 mandates the disclosure + opt-out (the README is one of the two disclosure surfaces;
        the emitted-script comments are the other). §15 (README outline, item 8) is the completions-section
        mandate. Decision 22 frames it as 'disclosed + opt-out-able'. Do NOT edit PRD.md."
  section: "h3.21 (§14.7), h2.14 (§15), h2.18 (decision 22)."

# READ-ONLY — the sibling PRP (T2.S1, tests) — confirms no conflict
- file: plan/005_d9b30e368811/P1M1T2S1/PRP.md
  why: "T2.S1 edits ONLY main_test.go (byte-level assertions on the emitted scripts). It explicitly defers
        the README to P1.M1.T3.S1 (this task). No overlap: T2.S1 = main_test.go; T3.S1 = README.md."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && git rev-parse --short HEAD
b433a08                                  # T1.S1 (bash) + T1.S2 (zsh) committed; option strings present
$ wc -l README.md
390 README.md                            # Shell completions section @290-366; insertion @335 (between 334 & 336)
# Emitted scripts already carry the option (verified):
#   completions/skilldozer.bash:83  [[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'   (active)
#   completions/skilldozer.bash:85  #   bind 'set show-all-if-ambiguous off'                 (opt-out)
#   main.go zshEvalRegistration:    setopt NO_LIST_AMBIGUOUS                                (active)
#                                  #   setopt LIST_AMBIGUOUS                                (opt-out)
# go build ./... + go test ./... → green (the option strings are present; T2.S1's tests, if landed, pass).
```

### Desired Codebase tree with files to be changed

```bash
README.md    # MODIFY — pure insertion of disclosure paragraph + option list + opt-out block (lines ~335)
# main.go / completions/* / main_test.go / go.mod / go.sum — UNCHANGED (Mode B docs; no source/test change)
```

**File responsibilities:**
| Region | Change | Source |
|---|---|---|
| README.md (Shell completions, ~335) | INSERT disclosure + option list + opt-out block | Contract LOGIC a-d / Touch point 4 |

### Known Gotchas of our codebase & Library Quirks

```markdown
<!-- GOTCHA #1 — README.md ONLY. This is Mode B changeset-level docs. Do NOT touch main.go (the
     zshEvalRegistration const — T1.S2's, committed), completions/* (T1.S1's, committed), or
     main_test.go (T2.S1's, parallel). The emitted scripts already carry the option; this task only
     documents it. Editing any source/test file is a scope violation. (research §0, §5.) -->

<!-- GOTCHA #2 — PURE INSERTION, edit no existing line. Insert BETWEEN line 334 ("unambiguous.") and
     line 336 ("Prefer to copy the file instead?"). Line 335 is the existing blank separator — keep a
     blank line on each side of the new block. Do NOT reword the namespace paragraph (332-334), the
     bullet list (317-328), the eval one-liners (302-310), or the manual-copy paths (341-365). -->

<!-- GOTCHA #3 — Name the EXACT option strings the emitted scripts use (verified, research §1):
       bash: show-all-if-ambiguous  (active: 'set ... on'; opt-out: 'set ... off')
       zsh:  NO_LIST_AMBIGUOUS       (active: setopt NO_LIST_AMBIGUOUS; opt-out: setopt LIST_AMBIGUOUS)
     Do NOT invent variants (e.g. "--show-all-if-ambiguous", "no_list_ambiguous" lowercase, "unsetopt").
     The opt-out one-liners must be runnable as-is: `bind 'set show-all-if-ambiguous off'` (bash) and
     `setopt LIST_AMBIGUOUS` (zsh). -->

<!-- GOTCHA #4 — fish gets a "lists by default; no option set" note, NOT an opt-out. §14.7 / Touch point 3:
     fish lists all matches in the pager by default, so no option is set and no opt-out exists. Including
     fish in the opt-out block (or claiming an option is set for it) would be wrong. The 3-bullet option
     list is the place to state the fish behavior. -->

<!-- GOTCHA #5 — User-facing tone: NO internal PRD section references. The README never cites "§14.7" /
     "§6.1" etc. in its body (verified — the existing Shell completions section is plain user prose).
     Keep the disclosure free of §X.Y refs; cite the option NAMES and the commands, not PRD anchors. -->

<!-- GOTCHA #6 — Fenced code block language tag is `bash` for the opt-out one-liners (both bash and zsh
     commands go in one `bash`-tagged block, matching the README's convention of using `bash` for shell
     commands throughout). Mirror the existing block style (e.g. the eval block at 302-304). -->

<!-- GOTCHA #7 — The verify gate is THREE greps (OUTPUT §4): show-all-if-ambiguous, NO_LIST_AMBIGUOUS,
     LIST_AMBIGUOUS. All three MUST match. Note LIST_AMBIGUOUS is a substring of NO_LIST_AMBIGUOUS, so a
     single grep for 'LIST_AMBIGUOUS' would match both — but the contract lists them as separate checks
     to ensure BOTH the zsh active name and the opt-out name appear. Make sure both NO_LIST_AMBIGUOUS
     (the active, in the option list) and the bare LIST_AMBIGUOUS (the opt-out, in the fenced block) are
     present. -->
```

---

## Implementation Blueprint

### Data models and structure

**None.** Documentation-only; no code, no types, no schemas.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: INSERT the disclosure block into README.md between line 334 and line 336
  - FILE: README.md. ANCHOR: the paragraph ending "...and a `<tab>` is\nunambiguous." (line 334) and the
    next paragraph "Prefer to copy the file instead? ..." (line 336).
  - INSERT (after the "...unambiguous." line, before the "Prefer to copy" line; keep a blank line each side):
      The emitted script also sets a shell option so that when a prefix matches two or
      more skills or flags, **every** match lists on the first `<tab>` instead of the
      shell freezing at the common prefix. Because the store has no index, completion
      is the main way to discover skills — hiding candidates would defeat that.

      This is a **session-global** option: it changes tab-completion listing for *every*
      command in that shell, not just `skilldozer`, and it is set only when you load
      skilldozer's completions (via the `eval`/`source` lines above). The option each
      shell sets:

      - **bash** — `show-all-if-ambiguous` (set on)
      - **zsh** — `NO_LIST_AMBIGUOUS` (set on)
      - **fish** — lists all matches by default; no option is set

      Prefer your shell's stock behavior? Restore the default after loading completions:

      ```bash
      # bash — list on the second Tab again
      bind 'set show-all-if-ambiguous off'

      # zsh — list only at the exact ambiguous point again
      setopt LIST_AMBIGUOUS
      ```
  - CONTENT MAP (contract LOGIC a-d): the 1st paragraph = (a) first-Tab list + why; the 2nd paragraph =
    (c) session-global + every-command + set-on-load; the 3-bullet list = (b) per-shell names incl. fish;
    the fenced block = (d) opt-out one-liners.
  - GOTCHA #2 (pure insertion), #3 (exact strings), #4 (fish note, no opt-out), #5 (no §X.Y refs),
    #6 (`bash` fence tag). The implementer may refine wording to better match the README's voice, but the
    four contract elements and the exact option/opt-out strings MUST remain.

Task 2: VERIFY the disclosure greps + scope invariants (the gate)
  - COMMAND: grep -q 'show-all-if-ambiguous' README.md && echo "bash name OK"        # must print OK
  - COMMAND: grep -q 'NO_LIST_AMBIGUOUS'   README.md && echo "zsh active OK"         # must print OK
  - COMMAND: grep -q 'LIST_AMBIGUOUS'      README.md && echo "zsh opt-out OK"        # must print OK
  - COMMAND: grep -q "bind 'set show-all-if-ambiguous off'" README.md && echo "bash opt-out one-liner OK"
  - COMMAND: grep -q 'setopt LIST_AMBIGUOUS' README.md && echo "zsh opt-out one-liner OK"
  - INVARIANT (scope): git diff --name-only   # MUST list ONLY README.md
  - INVARIANT (preserve): the existing anchors are intact —
      grep -q 'eval "$(skilldozer --completions)"' README.md   # eval one-liner preserved
      grep -q 'Prefer to copy the file instead'    README.md   # manual-path pivot preserved
      grep -q 'skilldozer --completions --shell fish | source' README.md  # fish source line preserved
  - INVARIANT (no source/test change): git diff --quiet main.go completions/ main_test.go go.mod go.sum && echo "code unchanged"
  - SANITY: go build ./... ; go test ./...   # both green (README change cannot break them; run to confirm)
```

### Implementation Patterns & Key Details

```markdown
<!-- The block is pure Markdown prose + one fenced `bash` block. Match the existing section's devices:
       - backtick-quoted code spans for every command/option/option-name
         (`skilldozer`, `<tab>`, `show-all-if-ambiguous`, `NO_LIST_AMBIGUOUS`, `eval`/`source`)
       - **bold** for the shell headers in the bullet list (**bash** / **zsh** / **fish**) and for the
         load-bearing word **every** / **session-global**
       - em dash `—` for parentheticals ("...discover skills — hiding candidates would defeat that.")
       - `*italics*` for emphasis on *every* (the existing section uses **bold** more than italics;
         either is fine; prefer **bold** to match)
       - a `bash`-tagged fenced block for the runnable opt-out one-liners, with `#` comments -->

<!-- The opt-out one-liners are RUNNABLE as written (copy-paste into the shell after eval). They are the
     exact inverse of what the emitted script sets:
       bash emitted:  bind 'set show-all-if-ambiguous on'   → opt-out: bind 'set show-all-if-ambiguous off'
       zsh emitted:   setopt NO_LIST_AMBIGUOUS              → opt-out: setopt LIST_AMBIGUOUS
     (setopt LIST_AMBIGUOUS is zsh's inverse of NO_LIST_AMBIGUOUS — turning the option back ON restores
      the stock "list only at the exact ambiguous point" behavior. This matches §14.7's prescribed opt-out.) -->
```

Notes easy to get wrong:
- The fenced block must use a single `bash` tag containing BOTH the bash and zsh opt-out lines (with `#` comments distinguishing them) — do NOT split into two blocks or use a `zsh` tag (the README uses `bash` for all shell commands).
- "session-global" and "every command, not just skilldozer" are the two facts PRD §14.7 most wants surfaced; do not soften or omit them.
- Do not add a §14.7 / §15 / decision-22 citation in the README body — it is user-facing (GOTCHA #5).

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Placement: after the namespace paragraph (334), before "Prefer to copy" (336)? → YES.** Touch point 4 specifies "after the bullet list (ends ~334) before 'Prefer to copy' (~336)". The natural reader flow is: bullet list (what completes) → namespace explanation (why `<tab>` is unambiguous) → §14.7 disclosure (the one global side effect + opt-out) → "Prefer to copy" (the manual alternative). Putting it earlier (inside the bullet list) would interrupt the what-completes flow; later (after the manual-copy paths) buries it.
2. **Single `bash`-tagged opt-out block with both shells, or per-shell blocks? → SINGLE block.** The README uses `bash` for all shell commands; one block with `# bash` / `# zsh` comments is the established pattern and keeps the opt-out scannable. A `zsh`-tagged block would be inconsistent.
3. **Cite PRD §14.7 in the README? → NO.** The README is user-facing and never cites internal PRD anchors in its body (verified). The disclosure names the options and commands, not the spec.
4. **Include fish in the opt-out block? → NO.** fish lists by default; no option is set, so there is nothing to opt out of. fish appears only in the 3-bullet option list with a "lists by default; no option set" note (GOTCHA #4).

### Integration Points

```yaml
NO CODE / NO TESTS / NO BUILD:
  - README.md is documentation; `go build` / `go test` are unaffected (run only as a sanity gate).
  - go.mod / go.sum UNCHANGED. No dependency of any kind.

DELIVERY-PATH CONSISTENCY (informational — not an edit here):
  - The README discloses what BOTH delivery paths produce: the §14.6 eval path (zsh derived wrapper
    sets the option; bash verbatim sets it) AND the §14.5 manual source/copy path (bash file sets it
    identically; zsh autoload does NOT — the option is eval-path-only). The README's disclosure is
    framed around "the emitted script" / "loading completions", which is accurate for the eval path
    (the primary, recommended path). The manual-copy zsh path is the optional parity the change map
    leaves alone; the README need not detail that subtlety (it would confuse users). If a future task
    adds the zsh autoload parity, this disclosure stays accurate.

OWNERSHIP (no conflicts):
  - T1.S1 (done, 5cf81d4): completions/skilldozer.bash option lines.
  - T1.S2 (done, b433a08): main.go zshEvalRegistration option lines.
  - T2.S1 (parallel): main_test.go byte-level assertions on the emitted scripts.
  - T3.S1 (this): README.md disclosure. No file overlap.
```

---

## Validation Loop

### Level 1: Disclosure presence (the core gate — contract OUTPUT §4)

```bash
cd /home/dustin/projects/skilldozer

grep -q 'show-all-if-ambiguous' README.md && echo "OK: bash option named"     || echo "FAIL"
grep -q 'NO_LIST_AMBIGUOUS'   README.md && echo "OK: zsh active named"        || echo "FAIL"
grep -q 'LIST_AMBIGUOUS'      README.md && echo "OK: zsh opt-out named"       || echo "FAIL"
# Expected: all three "OK". (LIST_AMBIGUOUS is a substring of NO_LIST_AMBIGUOUS, but the contract lists
# them separately to ensure BOTH the active name and the opt-out appear — verify the opt-out block has
# the bare `setopt LIST_AMBIGUOUS` and the option list has `NO_LIST_AMBIGUOUS`.)

# The opt-out one-liners are present verbatim (runnable):
grep -q "bind 'set show-all-if-ambiguous off'" README.md && echo "OK: bash opt-out" || echo "FAIL"
grep -q 'setopt LIST_AMBIGUOUS' README.md && echo "OK: zsh opt-out" || echo "FAIL"
# Expected: both "OK".
```

### Level 2: Content correctness (the four contract elements a-d)

```bash
cd /home/dustin/projects/skilldozer

# (a) first-Tab list-every-match framing + why (manifest-free store / discovery):
grep -qiE 'first.*tab|lists?.*(every|all).*match|common.prefix' README.md && echo "OK: (a) framing" || echo "FAIL: (a)"
# (c) session-global + every-command + set-on-load:
grep -qiE 'session.global|every command' README.md && echo "OK: (c) scope" || echo "FAIL: (c)"
# (b) fish lists-by-default note (no opt-out for fish):
grep -qiE 'fish.*default|default.*fish' README.md && echo "OK: (b) fish" || echo "FAIL: (b)"
# (d) opt-out fenced block with both one-liners (Level 1 already checked the strings).
# Expected: all "OK". (Manual eyeball: the disclosure names bash show-all-if-ambiguous + zsh NO_LIST_AMBIGUOUS
#  in a bullet list, and the fenced opt-out block has the two one-liners with `# bash` / `# zsh` comments.)
```

### Level 3: Scope + preservation invariants

```bash
cd /home/dustin/projects/skilldozer

# Only README.md changed:
git diff --name-only                  # Expected: README.md (ONLY)
git diff --quiet main.go completions/ main_test.go go.mod go.sum && echo "code unchanged" || echo "FAIL: code changed"
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"

# Existing README anchors preserved (pure insertion — nothing reworded):
grep -q 'eval "$(skilldozer --completions)"'             README.md && echo "OK: eval line"   || echo "FAIL"
grep -q 'Prefer to copy the file instead'                README.md && echo "OK: manual pivot"|| echo "FAIL"
grep -q 'skilldozer --completions --shell fish | source' README.md && echo "OK: fish source"|| echo "FAIL"
grep -q 'long-form flags only'                           README.md && echo "OK: bullet list" || echo "FAIL"
# Expected: README.md only; "code unchanged"; "deps unchanged"; all anchors "OK".

# Sanity (README cannot break these, but confirm nothing regressed):
go build ./... ; echo "build exit $?"   # Expected: 0
go test  ./... ; echo "test exit $?"    # Expected: 0
```

### Level 4: Rendered-Markdown sanity (the user-facing surface)

```bash
cd /home/dustin/projects/skilldozer

# The new block sits in the Shell completions section, between the namespace paragraph and "Prefer to copy":
sed -n '/^## Shell completions/,/^## Constraints/p' README.md | grep -nE 'show-all-if-ambiguous|NO_LIST_AMBIGUOUS|LIST_AMBIGUOUS|Prefer your shell|session-global'
# Expected: the disclosure lines appear AFTER "unambiguous." and BEFORE "Prefer to copy the file instead".

# Fence balance (the insertion added one ```bash ... ``` pair; no orphan fence):
awk '/^```/{c++} END{print "fence-pairs=" int(c/2) " (remainder " c%2 ")"}' README.md
# Expected: remainder 0 (even number of ``` fences — the document still renders as intended).
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — all three disclosure greps match (`show-all-if-ambiguous`, `NO_LIST_AMBIGUOUS`, `LIST_AMBIGUOUS`); both opt-out one-liners present verbatim
- [ ] Level 2 PASS — the four contract elements (a first-Tab framing, b per-shell names incl. fish, c session-global/every-command/set-on-load, d opt-out block) are all present
- [ ] Level 3 PASS — `git diff --name-only` = README.md only; "code unchanged"; "deps unchanged"; existing README anchors (eval line, "Prefer to copy", fish source, bullet list) preserved; `go build`/`go test` exit 0
- [ ] Level 4 PASS — the disclosure sits between the namespace paragraph and "Prefer to copy"; fence count is even (renders correctly)

### Feature Validation
- [ ] Disclosure states the emitted script sets a session-global option so ambiguous matches list on the first `<tab>`
- [ ] Names bash `show-all-if-ambiguous` + zsh `NO_LIST_AMBIGUOUS`; notes fish lists by default (no option set)
- [ ] States it affects listing for EVERY command in the shell, set only on completion load
- [ ] Opt-out fenced block gives `bind 'set show-all-if-ambiguous off'` (bash) + `setopt LIST_AMBIGUOUS` (zsh)

### Code Quality / Convention Validation
- [ ] Matches the README's existing tone (backtick-quoted commands, **bold** shell headers, em-dashes, `bash`-tagged fences)
- [ ] No internal PRD `§X.Y` references in the user-facing prose
- [ ] Pure insertion — no existing line reworded; minimal diff

### Scope Discipline (the docs-only boundary)
- [ ] Did NOT touch `main.go` (zshEvalRegistration — T1.S2's, committed)
- [ ] Did NOT touch `completions/*` (T1.S1's bash file / autoload / fish — committed/unchanged)
- [ ] Did NOT touch `main_test.go` (T2.S1's byte-level tests — parallel)
- [ ] Did NOT touch `go.mod` / `go.sum` (no dependency change)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't edit any source/test file.** This is README-only Mode B docs. The emitted scripts already carry the option (T1 committed); this task only documents it. Editing `main.go`/`completions/*`/`main_test.go` collides with T1/T2. (GOTCHA #1)
- ❌ **Don't reword existing README lines.** Pure insertion between line 334 and 336. The eval one-liners, bullet list, namespace paragraph, and manual-copy paths stay byte-intact. (GOTCHA #2)
- ❌ **Don't invent option names.** Use the exact strings the emitted scripts use: bash `show-all-if-ambiguous`, zsh `NO_LIST_AMBIGUOUS` (active) / `LIST_AMBIGUOUS` (opt-out). No `--show-all-if-ambiguous`, no lowercase, no `unsetopt`. (GOTCHA #3)
- ❌ **Don't give fish an opt-out.** fish lists by default; no option is set. fish appears only in the option bullet list with a "lists by default; no option set" note. (GOTCHA #4)
- ❌ **Don't cite PRD §14.7 / §15 / decision 22 in the README body.** It's user-facing; the existing section never cites internal anchors. Name the options and commands, not the spec. (GOTCHA #5)
- ❌ **Don't split the opt-out into per-shell fenced blocks or use a `zsh` tag.** One `bash`-tagged block with `# bash` / `# zsh` comments, matching the README's convention. (GOTCHA #6)
- ❌ **Don't soften "session-global" / "every command."** Those two facts are the core of the §14.7 disclosure mandate; a user must understand the coupling is shell-wide, not skilldozer-scoped.

---

## Confidence Score

**9.5/10** — This is the smallest possible subtask: a single pure insertion into one Markdown file at a verified anchor (between README lines 334 and 336), with the exact option/opt-out strings read directly from the committed emitted sources (`research/verified_facts.md` §1), the four contract elements (a-d) mapped to concrete text (§3), and a tone-matched draft block provided (§6). The implementing prerequisites (T1.S1 bash + T1.S2 zsh) are committed at HEAD `b433a08`, so the strings the README discloses are already in the tree. There is no code/test/build risk — the only failure mode is wrong wording or wrong option names, both of which the verified-facts §1 strings and the Level 1/2 greps guard against. The 0.5 reservation is editorial: the draft block's exact phrasing may be refined to better match the README's voice, but the contract elements and exact strings are fixed.
