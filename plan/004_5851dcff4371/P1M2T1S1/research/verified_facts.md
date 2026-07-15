# Verified Facts — P1.M2.T1.S1: Rewrite `completions/skilldozer.bash` (skills-first, long-form-only)

Plan `004_5851dcff4371` (subcommands → flags + skills-first completions). Every
claim below was read directly from the live source at `/home/dustin/projects/skilldozer`
(`completions/skilldozer.bash` read in full — 69 lines; `main_test.go::TestEmbeddedCompletionsMatchOnDisk`
read in full; `plan/004_5851dcff4371/architecture/completions_change_map.md` §1 read
in full; `delta_prd.md` §13 assertions read). This is a single-file rewrite
(`completions/skilldozer.bash` only). Zero Go/deps changes.

---

## §1 — Exact edit anchors (CURRENT completions/skilldozer.bash, 69 lines)

```
lines 11-13   the LOCKSTEP comment (UPDATE — cite decisions 19/20 + long-form-only rule)
lines 31-33   the value-routing comment (UPDATE — reflect --search) and --store|--init))
line  34      the value-routing case (EDIT: --search|-s) → --search);  --store) → --store|--init))
line  41      the compgen -W flag list (EDIT: drop 7 short forms; add --check --init --completions)
lines 46-55   the walk-guard loop + have_pos tracking (DELETE ENTIRELY — see §3)
line  61      tags=$(skilldozer --relative --all 2>/dev/null)   (KEEP BYTE-IDENTICAL)
line  62      cands="$tags"                                     (REMOVE — inline cands)
line  63      (( have_pos == 0 )) && cands="$cands check init completion"  (DELETE)
```

`grep -n 'completion\|--shell\|have_pos\|cands'` confirms the structure. The function
name `_skilldozer_completion` (line 16) and the final `complete -F _skilldozer_completion
skilldozer` (line 69) are UNCHANGED — the §13 assertion `grep -q '_skilldozer_completion'`
(delta_prd.md:47) depends on both.

---

## §2 — The flag list (line 41) — exact before/after, lockstep-verified

The §14.4 lockstep requires the bash flag set to match `main.go parseArgs()`.
Confirmed against live main.go (`case "--version"/"--help"/"--path"/"--list"/"--all"/
"--file"/"--relative"/"--no-color"/"--search"/"--store"/"--check"/"--init"/"--completions"`
all present) AND the PRD §14.6 behavior-contract table (which lists exactly these 13).

```bash
# OLD (line 41) — long + short forms, NO --check/--init/--completions:
"--version -v --help -h --path -p --list -l --all -a --file -f --relative --no-color --search -s --store"
# NEW — long-form only + the three promoted flags:
"--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions"
```

GOTCHA (the §13 LONG_FORM_ONLY assertion, delta_prd.md:65): the gate is
`! grep -Eq '\-\-version[ ]+-v'`. The OLD list has `--version -v` adjacency → grep matches →
the `!` FAILS. The NEW list has NO short forms → no `-v` anywhere → `!` SUCCEEDS →
LONG_FORM_ONLY_BASH_OK. So the new list must contain zero `-X` short tokens. Verified: the
new list has none. The §13 `grep -q '\-\-check'` and `grep -q '\-\-completions'` (delta_prd.md:62-63)
also require those two substrings — both present in the new list. ✓

`--shell` is INTENTIONALLY ABSENT from the flag list. PRD §14.6's `skilldozer -<tab>` table
lists exactly the 13 flags above (no `--shell`): `--shell` is a modifier for `--completions`,
not a top-level menu flag. Do NOT add `--shell` to the compgen -W list. (verified_facts §5.)

---

## §3 — The walk-guard loop (lines 46-55) becomes DEAD CODE → delete it wholesale

The current loop does TWO things, BOTH now obsolete:
1. `[[ "${words[i]}" == "check"||"init"||"completion" ]] && return 0` — suppress further
   offers once an exclusive subcommand appears. OBSOLETE: `check`/`init`/`completion` are
   no longer bare tokens (they're `--flags` now, and bare `check`/`init`/`completions` are
   ordinary skill tags — which are NEVER exclusive with each other: PRD §6.1
   `skilldozer <tag> [<tag>...]` accepts multiple tags).
2. `have_pos=1` tracking — used ONLY to gate line 63's subcommand offer
   (`(( have_pos == 0 )) && cands="$cands check init completion"`). OBSOLETE: line 63 is
   being deleted.

The change map confirms: "The entire have_pos / subcommand-walk logic can be simplified
to: always offer tags as positionals (skills are never exclusive with each other)."

**DELETE THE WHOLE LOOP** (the `local i have_pos=0` declaration + the `for ((i=1; i<cword;
i++)); do … done` body). Do NOT just strip the `check`/`init`/`completion` line — the
remaining `have_pos` logic would be dead and the `local i`/`have_pos` vars unused (shellcheck
noise + confusion). After deletion, the function flows: setup → value-routing case → flag
block → tag probe → `COMPREPLY=($(compgen -W "$tags" -- "$cur"))`.

`cands` is now always == `tags`, so inline it: drop the `local tags cands` + `cands="$tags"`
lines, declare `local tags`, and write `COMPREPLY=($(compgen -W "$tags" -- "$cur"))`
directly. (The contract LOGIC (d) explicitly allows: "the line is removed and the variable
flows directly.")

Trace of the simplified function against PRD §14.1 (all rows pass):
  `skilldozer <tab>`        → prev="skilldozer"/""; no value-route; cur not "-*"; tags offered. ✓
  `skilldozer a<tab>`       → same; compgen filters tags to `a*`. ✓
  `skilldozer writing/<tab>`→ same; compgen path-prefix-filters. ✓
  `skilldozer -<tab>`       → cur="-*"; flag block; 13 long flags offered. ✓
  `skilldozer --c<tab>`     → cur="--c*"; flag block; compgen narrows to --check/--completions. ✓
  `skilldozer --init <tab>` → prev="--init"; value-route `--store|--init)` → directories. ✓
  `skilldozer --store <tab>`→ prev="--store"; value-route → directories. ✓
  `skilldozer --search <tab>`→ prev="--search"; value-route `--search) return 0` → nothing. ✓
  `skilldozer example <tab>`(2nd positional) → prev="example"; no route; tags offered. ✓ (multi-tag)

---

## §4 — The value-routing case (line 34) — exact before/after

```bash
# OLD (line 34):
        --search|-s) return 0 ;;
        --store) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
# NEW:
        --search) return 0 ;;
        --store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
```
- Drop `-s` from `--search` routing: `--search|-s)` → `--search)`. (Long-form-only; -s is
  still valid at runtime but the completion routes on the long form. A user who typed `-s`
  already has prev="-s" which won't match `--search)` — but that's fine: `-s <tab>` then
  falls through to the tag block, which is harmless. The OLD behavior also only routed on
  the exact prev token; this is unchanged in spirit, just long-form-aligned.)
- Add `--init` to directory routing: `--store)` → `--store|--init)`. PRD §14.1 rule 5 +
  §14.2: `--init [<dir>]` completes to directories (the store to adopt), like `--store`.

Update the comment block above it (lines 31-33) to match: `--search` (drop `-s`),
`--store/--init` (directory).

---

## §5 — `--shell` enum routing is OUT OF SCOPE for this task (do NOT add it)

PRD §14.2 mentions "`--completions --shell <name>` — `--shell` takes a fixed enum
(bash/zsh/fish); offer those three words." But for THIS task:
- The contract LOGIC (a-f) does NOT list a `--shell)` value-route edit.
- The change map §1 (bash) does NOT list one (and neither do its §2 zsh / §3 fish sections).
- The §13 delta assertions (delta_prd.md:47,62-65) do NOT test `--shell` completion routing.
- The PRD §14.6 behavior-contract table has NO `--completions --shell <tab>` row.

So `--shell` is (a) absent from the flag-offer list (it's a --completions modifier, not a
top-level menu flag — §2) and (b) NOT routed to the enum in this changeset. Adding it would
be scope creep beyond the contract. Omit it. (If the team later wants `--completions --shell
<tab>` → bash/zsh/fish, that's a separate, consistent follow-up across all three files.)

---

## §6 — TestEmbeddedCompletionsMatchOnDisk auto-satisfies on rebuild

`main_test.go:2995 TestEmbeddedCompletionsMatchOnDisk` compares `completionScript("bash")`
(the `//go:embed completions/skilldozer.bash` var) to `os.ReadFile("completions/skilldozer.bash")`.
After this task edits the on-disk file, a `go build` re-runs the embed → the embedded var
holds the NEW bytes → embedded == on-disk → the test PASSES automatically. **No edit to the
embed declaration or the test is needed** (and neither is in this task's scope — the embed
is unchanged main.go, the test is P1.M1.T3's concern). The ONLY requirement: the on-disk
file and the embedded var must be byte-identical, which `//go:embed` guarantees by
construction. Just rebuild before running the gate.

---

## §7 — The §13 delta assertions that MUST pass (delta_prd.md §13, lines 47/62-65)

```bash
go build -o skilldozer .   # rebuild so //go:embed picks up the new bytes
./skilldozer --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo COMPLETIONS_BASH_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-completions'        && echo EMBED_HAS_COMPLETIONS_FLAG_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-check'              && echo EMBED_HAS_CHECK_FLAG_OK
! ./skilldozer --completions --shell bash 2>/dev/null | grep -Eq '\-\-version[ ]+-v'   && echo LONG_FORM_ONLY_BASH_OK
go test ./...   # incl. TestEmbeddedCompletionsMatchOnDisk
```
All four depend SOLELY on the flag-list line (§2) being the 13 long forms with no short
forms, and the function name `_skilldozer_completion` being preserved. The walk-loop
deletion (§3) and value-routing edits (§4) do not affect any of these greps. The contract
OUTPUT ("The §13 delta assertions must pass after go build") is satisfied by §2 + preserving
the function name.

---

## §8 — No conflict with the parallel sibling P1.M1.T3.S2 (test-only)

P1.M1.T3.S2 (Implementing) edits `main_test.go` + one line in
`internal/skillsdir/skillsdir_test.go` — it flips run-level dispatch/completion/help-text/
unconfigured ASSERTIONS to the --flag contract. It explicitly does NOT touch any
`completions/*` file (its PRP's scope-discipline: "Did NOT modify completions/* (those are
P1.M2.T1)"). This task edits ONLY `completions/skilldozer.bash`. **Disjoint files; no
collision.** The one shared touchpoint is `TestEmbeddedCompletionsMatchOnDisk` — T3.S2
LEAVES it green (it verifies embed↔disk identity, which this task preserves via §6), and
this task doesn't touch the test. Land in either order; rebuild before the gate.

---

## §9 — Scope discipline + zero deps

- Edit ONLY `completions/skilldozer.bash`. Do NOT touch `completions/_skilldozer` or
  `completions/skilldozer.fish` (P1.M2.T1.S2), `main.go` (the --flag contract is already
  Complete per P1.M1.T1), `main_test.go`/`internal/*` (P1.M1.T3), `README.md` (P1.M2.T2.S1),
  or the `//go:embed` declaration (unchanged).
- Do NOT modify PRD.md (read-only), tasks.json, prd_snapshot.md, or .gitignore.
- Zero Go changes, zero deps. The gate is `bash -n` (syntax) + the §13 greps (content) +
  `go build` (embed rebuild) + `go test ./...` (embed↔disk identity).

---

## §10 — Validation (verified commands)

```bash
bash -n completions/skilldozer.bash   # syntax check (baseline passes; must still pass after)
go build -o skilldozer .              # rebuild so //go:embed picks up the new bytes
# §13 assertions:
./skilldozer --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo COMPLETIONS_BASH_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-completions'        && echo EMBED_HAS_COMPLETIONS_FLAG_OK
./skilldozer --completions --shell bash 2>/dev/null | grep -q '\-\-check'              && echo EMBED_HAS_CHECK_FLAG_OK
! ./skilldozer --completions --shell bash 2>/dev/null | grep -Eq '\-\-version[ ]+-v'   && echo LONG_FORM_ONLY_BASH_OK
go test ./...                                                          # incl. TestEmbeddedCompletionsMatchOnDisk
# content spot-checks (optional, lockstep faithfulness):
grep -c -- '-v -h\|-s ' completions/skilldozer.bash   # expect 0 (no short forms advertised)
grep -q -- '--check --init --completions' completions/skilldozer.bash   # the three promoted flags, in order
grep -q 'skilldozer --relative --all 2>/dev/null' completions/skilldozer.bash   # tag probe byte-identical
```
