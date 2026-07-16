# Verified Facts — P1.M1.T3.S1 (README: disclose the §14.7 option set + opt-out one-liners)

All facts verified directly against the working tree (HEAD `b433a08`).

---

## §0 — Current repo state (what T3.S1 executes against)

- HEAD `b433a08 "Configure zsh for immediate ambiguous listing"` = **P1.M1.T1.S2 (zsh) committed**.
- Parent `5cf81d4 "Add bash immediate ambiguous match listing"` = **P1.M1.T1.S1 (bash) committed**.
- → The emitted completion scripts ALREADY carry the §14.7 option + opt-out. This subtask (README
  disclosure) only documents what they already do; it depends on the scripts being in place (they are),
  NOT on the parallel T2.S1 (tests) landing.
- `grep` for the option strings in the emitted sources confirms presence (§1). T3.S1 edits `README.md` ONLY.

---

## §1 — Exact option strings the README must disclose (mirror the emitted scripts, verified)

### bash — `completions/skilldozer.bash` (on-disk == emitted verbatim)
```
83: [[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'   # ACTIVE (guarded for interactivity)
85: #   bind 'set show-all-if-ambiguous off'                  # OPT-OUT (commented, the disclosed one-liner)
```
- Disclosure comments at 73/76/79-80 name `show-all-if-ambiguous` and the session-global + opt-out story.
- README opt-out one-liner to give the user: `bind 'set show-all-if-ambiguous off'`

### zsh — `main.go` `zshEvalRegistration` const (appended at eval time; the on-disk autoload is unchanged)
```
37 (rel): #   setopt LIST_AMBIGUOUS        # OPT-OUT (commented)
38 (rel): setopt NO_LIST_AMBIGUOUS         # ACTIVE
```
- Disclosure comments name `NO_LIST_AMBIGUOUS`, the session-global scope, and the opt-out `setopt LIST_AMBIGUOUS`.
- README opt-out one-liner to give the user: `setopt LIST_AMBIGUOUS`

### fish — no option set
- fish lists all matches in the pager by default (§14.7). No active line, no opt-out. The README must
  NOTE this ("fish lists by default; no option is set") so the three-shell matrix is complete.

---

## §2 — The README insertion point (verified line numbers)

README.md Shell completions section spans lines 290-366. The relevant tail:

```
317-328: the skills-first / long-form-only / value-flag bullet list (3 bullets)
332-334: "This works because every action that is not a skill tag is a `--flag` ... so the
         bare positional namespace belongs entirely to skill tags and a `<tab>` is
334:     unambiguous."
335:     (blank)
336:     "Prefer to copy the file instead? The manual path below picks up edits to ..."
```

**Insertion point: AFTER line 334 ("unambiguous.") and BEFORE line 336 ("Prefer to copy").**
Line 335 is currently the blank separator. Insert the disclosure paragraph + option-set list + opt-out
fenced block there, keeping a blank line on each side (so the new block sits between the namespace-safety
explanation and the "Prefer to copy" manual-path pivot).

- The change map Touch point 4 says "after the bullet list (ends ~334) before 'Prefer to copy' (~336)" —
  the practical anchor is the "...unambiguous." line (334) and the "Prefer to copy" line (336).
- Do NOT insert INSIDE the bullet list (317-328) or between the namespace paragraph and nothing; the
  cleanest reader flow is: bullet list → namespace explanation → §14.7 disclosure → "Prefer to copy".

---

## §3 — Contract requirements (LOGIC a-d) mapped to concrete text

| Contract req | What the README must say |
|---|---|
| (a) session-global option, first-Tab list | State that the emitted script sets a shell option so ambiguous prefixes list every match on the FIRST `<tab>` (not a freeze at the common prefix). Note WHY: the store is manifest-free, so completion is how you discover skills. |
| (b) name per shell | bash `show-all-if-ambiguous`; zsh `NO_LIST_AMBIGUOUS`; fish lists by default (no option set). |
| (c) affects EVERY command, set only on load | Explicitly: session-global — changes tab-listing for every command in that shell, not just skilldozer; set only when you load skilldozer completions (eval/source). |
| (d) opt-out one-liners (fenced) | bash: `bind 'set show-all-if-ambiguous off'`; zsh: `setopt LIST_AMBIGUOUS`. In a ` ```bash ` fenced block. |

Verification gate (OUTPUT §4): `grep -q 'show-all-if-ambiguous' README.md && grep -q 'NO_LIST_AMBIGUOUS' README.md && grep -q 'LIST_AMBIGUOUS' README.md` → all three match.

---

## §4 — README tone/convention to mirror (verified from the existing section)

- User-facing prose — the README does NOT cite internal PRD section numbers (no "§14.7" in the body).
  Keep the disclosure free of `§X.Y` refs; it's for end users.
- Backtick-quoted code spans for every command/option: `` `skilldozer --completions` ``, `` `--check` ``,
  `` `show-all-if-ambiguous` ``, `` `setopt NO_LIST_AMBIGUOUS` ``.
- **Bold** for shell headers and key emphasis (e.g. **bash**, **zsh**, **fish**; **every** match).
- Em dash `—` for parentheticals (the existing prose uses `—` heavily).
- Fenced code blocks use ` ```bash ` for shell commands.
- Conversational-but-precise: "The easiest way to load completions is...", "This works because...".
  Mirror that register ("The emitted script also sets...", "Prefer your shell's stock behavior?").

---

## §5 — Scope boundary (do NOT touch)

- `main.go` (the `zshEvalRegistration` const) — already done (T1.S2, committed). READ-ONLY here.
- `completions/*` (bash/zsh/fish files) — already done (T1.S1, committed) / unchanged. READ-ONLY.
- `main_test.go` (the §14.7 byte-level test locks) — P1.M1.T2.S1's scope (parallel, in progress). NOT this task.
- Any `.go` source, `go.mod`, `go.sum` — test/doc-only delta; no source change.
- Existing README content (eval one-liners @302-310, flag list @321-325, manual-copy paths @341-365, fish
  source line @309, namespace paragraph @332-334, "Prefer to copy" paragraph @336) — PRESERVE all of it.
  T3.S1 is a PURE INSERTION between lines 334 and 336; it edits no existing line.

---

## §6 — Proposed disclosure block (draft, tone-matched — implementer may refine wording)

Insert between line 334 ("unambiguous.") and line 336 ("Prefer to copy..."):

````markdown
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
````

(Note the nested fence: the outer block above is ```` ```` ```` only to show the inner ```bash fence; in
the actual README the disclosure is plain prose + a single ```bash block — no meta-fencing.)
```
