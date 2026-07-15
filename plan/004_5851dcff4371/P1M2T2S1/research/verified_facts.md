# Verified Facts — P1.M2.T2.S1 (README.md: --flag CLI contract sync)

Researched against the LIVE codebase (README.md read in full at 345 lines; every
bare-subcommand reference grepped with line numbers; the binary's ErrNotFound message
confirmed; the sibling PRP read as a CONTRACT). This is a documentation-only (Mode B)
subtask — no `.go`/completion-file changes. It depends on M1 (--flag code, Complete)
and M2.T1 (completion files rewritten).

---

## §1. The deliverable — README.md reflects the --flag CLI contract (decisions 19/20)

The codebase converted `check`/`init`/`completion` bare subcommands → `--check`/`--init`/
`--completions` flags (decision 19, P1.M1 Complete). README.md still documents the OLD
bare-subcommand model. This subtask syncs it: convert every bare-subcommand reference,
DELETE the now-false "Reserved tag names" paragraph, rewrite the Completions section for
skills-first + long-form-only, and reorder sections to PRD §15. The grep assertions are
the hard gate.

---

## §2. EVERY bare-subcommand reference (grepped with line numbers — exhaustive)

```
# `skilldozer init` (must → `skilldozer --init`):
 43: `skilldozer init` (see First run, below)
 63: run `skilldozer init` once:
 66: skilldozer init                              (code block)
 76: skilldozer init /path/to/store      # positional   (code)
 77: skilldozer init --store /path/to/store      (code)
 89: STORE="$(skilldozer init --store /path)"    (code, in prose)
240: `skilldozer check` ... `skilldozer init` runs first-run setup.   (Reserved para — DELETE)
297: set by `skilldozer init`. The config
315: run \`skilldozer init\`     (the unconfigured message — verbatim from the binary)
330: written by `skilldozer init`) is expected

# `skilldozer check` (must → `skilldozer --check`):
183: skilldozer check                             (Usage code block)
240: `skilldozer check` runs validation           (Reserved para — DELETE)
281: skilldozer check                             (Adding a skill code block)

# `skilldozer completion` (must → `skilldozer --completions`):
107: eval "$(skilldozer completion)"             (code)
113: skilldozer completion --shell fish | source (code)
117: `skilldozer completion` auto-detects         (prose)
```

PLUS prose references to the bare COMMAND NAMES (not the full `skilldozer X` form) that
must ALSO flip for accuracy (the grep does NOT catch these, but the contract LOGIC (a)
says "ALL bare subcommand references"):
```
 80: `--store <dir>` implies `init`
 83: bare `--store` with an `init` token is the canonical shape
 84: Because `--store` implies init ... it is an init, not a one-off store override
 88: On success, `init` prints exactly the configured store path
 90: the post-setup `check` report go to stderr
100: The easiest way to load completions is the `completion` subcommand
207: `--store` expects a value: `init --store` with nothing after it
```

---

## §3. The CRITICAL grep-safety conflict (the one-pass-stall guard)

The contract LOGIC (b) suggests replacing the Reserved paragraph with a note that *may*
include the literal `` `skilldozer check` resolves the tag ``. THAT WOULD FAIL the
contract's own grep assertion:

    ! grep -Eq 'skilldozer (init|check|completion)\b' README.md

because `skilldozer check\b` matches `` `skilldozer check` `` (the `\b` is between "check"
and the closing backtick/space). The contract's suggested phrasing "A skill named
check/init/completions resolves normally" does NOT contain `skilldozer check`, so it is
safe. RESOLUTION: the replacement one-liner must AVOID the literals `skilldozer init`,
`skilldozer check`, `skilldozer completion` entirely. Use "A skill named `check`" + the
flag forms (`--check`). Verified grep-safe phrasing (research §4 EDIT 8).

ALSO grep-safe to confirm: `skilldozer --init` does NOT contain the substring
"skilldozer init" (it is "skilldozer " + "--init"; the `-` breaks it), so the new --flag
forms pass `! grep -Eq 'skilldozer (init|check|completion)\b'` cleanly.

---

## §4. The exact edits (13, grouped; before→after text transcribed from the live README)

EDIT 1 (L43): `` `skilldozer init` (see First run `` → `` `skilldozer --init` (see First run ``
EDIT 2 (L63-66): `run \`skilldozer init\` once:` + code `skilldozer init` → `--init` (both)
EDIT 3 (L76-77): both `skilldozer init ...` code lines → `skilldozer --init ...`
EDIT 4 (L80-90): the `--store` prose paragraph — flip `init`→`--init` (×4: implies/canonical/
   prints/STORE example) and `check`→`--check` (post-setup report). Exact before/after in PRP.
EDIT 5 (L96-118): the Completions load block — `completion`→`--completions` (subcommand prose
   + 3 command literals) AND APPEND the skills-first/long-form-only description (the 13 long
   flags list + the bare-<tab>=skills + --init/--store=dirs + --search=nothing bullets +
   the namespace rationale). Exact before/after in PRP.
EDIT 6 (L183): Usage code `skilldozer check` → `skilldozer --check`
EDIT 7 (L207-208): error-contract `` `init --store` with nothing after `` → `` `--init --store` ``
EDIT 8 (L239-247): DELETE the **Reserved tag names.** paragraph; REPLACE with the grep-safe
   one-liner (no `skilldozer check/init/completion` literals — §3).
EDIT 9 (L281): Adding-a-skill code `skilldozer check` → `skilldozer --check`
EDIT 10 (L297): `` set by `skilldozer init`. `` → `` set by `skilldozer --init`. ``
EDIT 11 (L315): the unconfigured message `run \`skilldozer init\`` → `run \`skilldozer --init\``
   (matches the LIVE binary: skillsdir.go:275 `ErrNotFound` already says `--init`).
EDIT 12 (L330): `` written by `skilldozer init`) `` → `` written by `skilldozer --init`) ``
EDIT 13 (MOVE, last): cut the `## Shell completions` block (L94–L151, header→blank-line-before
   `## Usage`) and paste it after the `## How \`skilldozer\` finds the store` section (before
   `## Constraints`), making the order match PRD §15 (completions = item 8).

---

## §5. Section reordering — PRD §15 wants completions at position 8

PRD §15 (h2.14) outline order: Title → Why → Install → Usage → Where skills live →
Adding a skill → How finds store → **Shell completions** → Constraints.

Current README order has Shell completions at position 4 (right after Install/First run).
§15 wants it at position 8 (after "How finds store", before "Constraints"). The other
sections (Usage, Where skills live, Adding a skill, How finds store) are already in the
correct relative order (4-7); only completions is wedged at 4. So ONE move (completions
4→8) makes the whole README match §15. Block boundaries are unambiguous `##` headers:
- cut from `## Shell completions` (L94) through the blank line before `## Usage` (L151)
- paste after the end of `## How \`skilldozer\` finds the store` (before `## Constraints` L324)

Do the move LAST (after all content edits), because it shifts downstream line numbers.
Anchor the move by HEADER TEXT, not line numbers.

---

## §6. The grep gate (the hard validation — all 4 must print/exit cleanly)

```bash
grep -q 'skilldozer --completions' README.md           # present (EDIT 5)
! grep -q 'Reserved tag names' README.md               # removed (EDIT 8)
! grep -Eq 'skilldozer (init|check|completion)\b' README.md   # no bare (EDITS 1-12; §3 safety)
```
The combined `-E` form is word-boundary-safe against `--init`/`--check`/`--completions`
(the `-` breaks the "skilldozer <word>" substring). Run ALL THREE; any failure = a missed
reference or a grep-unsafe replacement (§3).

---

## §7. Boundary with the sibling P1.M2.T1.S2 (zsh+fish completions) — no collision

P1.M2.T1.S2 (read its PRP) rewrites `completions/_skilldozer` + `completions/skilldozer.fish`
(skills-first + long-form-only + the 3 new flags). It does NOT touch README.md. This
subtask edits ONLY README.md. Disjoint. The README's Completions section DESCRIBES the
behavior the sibling's files implement (so the description text is sourced from PRD §14.1/
§14.6, which the sibling's files also follow). Land in either order.

P1.M1.T2.S1 (usageText rewrite) edits main.go's in-code help, NOT README. P1.M1.T3.*
(main_test.go) are test-only. None touch README. So README.md is this subtask's alone.

## §8. No code/test/completion-file changes (scope discipline)

This is Mode B documentation. Do NOT edit: main.go, main_test.go, internal/*, completions/*,
go.mod, go.sum, PRD.md, tasks.json, prd_snapshot.md, .gitignore. README.md is the ONLY file
touched. The binary already emits the --flag contract (M1 Complete); the completion files are
already rewritten (M2.T1, in progress / done); this subtask only brings the README in line.
