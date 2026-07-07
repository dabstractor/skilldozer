# PRP — P1.M3.T8.S1: `main` tag-resolution path output + §6.4 atomicity

> **Subtask:** P1.M3.T8.S1 — the first subtask of T8 ("main tag-resolution path
> output + modifiers", PRD §7.2/§6.1/§6.4). This is milestone **M3** (Tag resolution
> & path output), build-order step 3 (right after T7 shipped the pure resolver).
> S2 (`--file`/`-f`, `--relative`, `--all`/`-a`) builds directly on top of this.
>
> **Scope:** **MODIFY** `main.go` (4 additive edits: +`strings`/`resolve` imports,
> +`tags []string` config field, +positional-tag capture in `parseArgs`, +the
> `<tag>`-resolution branch in `run`) and `main_test.go` (update 1 test, add 11 new
> tests). **No new files.** No touch to `internal/discover/*`, `internal/skillsdir/*`,
> `internal/ui/*`, `internal/resolve/*`, `go.mod`, `go.sum`, or `PRD.md`.
>
> **DEPENDENCIES (hard, now SATISFIED):** `resolve.Resolve` is **LANDED** and green
> (`internal/resolve/resolve.go`, commit `c7f4f99`, P1.M3.T7.S1). `discover.Index`
> (M2.T5) and `skillsdir.Find` (M1.T2) are LANDED and green. This subtask is purely
> the `main` wiring that calls them and enforces the §6.4 output contract.
>
> **VERIFICATION STATUS (authoritative):** the EXACT source in the Implementation
> Blueprint has been **compiled, gofmt'd, vetted, unit-tested, and smoke-tested
> against a real built binary** in a throwaway `/tmp/skpp_t8_verify` on go1.25 (a
> verbatim copy of the real module). See `research/verified_facts.md` for the full
> gate output: gofmt silent, vet clean, build OK, main package **36 tests pass**
> (was 24; +12 new/updated), whole module green, `go.mod`/`go.sum` byte-identical,
> AND the six end-to-end `SKPP_SKILLS_DIR` binary behaviors (single/multi-order/
> atomicity/unknown/ambiguous/duplicate) all match PRD §6.1/§6.4.

---

## Goal

**Feature Goal**: Wire `skpp <tag> [<tag>...]` (PRD §6.1) end-to-end in `main.go`:
parse positional `<tag>` args, resolve the store via `skillsdir.Find()`, build the
index via `discover.Index()`, resolve each tag via `resolve.Resolve()`, and print
**one absolute skill-directory path per line, in input order** to stdout. Critically,
enforce the **§6.4 atomicity / `$()`-safety contract**: resolve **every** tag first,
buffering the output; if **any** tag fails (unknown or ambiguous), print **one error
line per problem tag** to stderr, print **nothing** to stdout, and exit 1 — so
`pi --skill "$(skpp badtag)"` fails loudly (empty `$()`, exit 1) rather than passing
a partial or garbage path. Only when all tags resolve are the buffered paths flushed
to stdout.

**Deliverable**: Two **MODIFIED** files (no new files):
1. `main.go` — add `"strings"` + `internal/resolve` imports; add `tags []string` to
   `config`; capture non-dashed positionals as tags in `parseArgs`; add the
   `<tag>`-resolution branch to `run` (after `--list`, before the default `return 1`)
   implementing the buffered-atomicity contract.
2. `main_test.go` — update `TestParseArgsUnknownTolerated` (positionals are now
   captured as tags); add 3 new `parseArgs` tests + 8 new `run` tag-resolution
   tests (single/multiple-input-order/atomicity/all-fail/duplicate/ambiguous/
   unresolvable/absolute/version-precedence). All 24 existing tests preserved.

**Success Definition**: `gofmt -l main.go main_test.go` is silent; `go vet ./...` is
clean; `go build ./...` and `go test ./...` pass (**main package 36 tests**: the
existing 24 + 12 new/updated). `go mod tidy` is a **no-op** (`go.mod`/`go.sum`
unchanged — `strings` is stdlib, `resolve` is internal). A real built binary over a
`SKPP_SKILLS_DIR` store prints one absolute dir path per resolved tag in input
order; on any failure prints nothing to stdout and one stderr line per problem tag,
exit 1. No touch to `internal/*`, `go.mod`, `go.sum`, `PRD.md`; no new files; no
`--file`/`--relative`/`--all`/`--search`/`check`/`--help`.

---

## Why

- This subtask **makes `resolve` do something a user can see.** T7 shipped the pure
  precedence resolver and proved §7.2 in isolation. T8.S1 is where `skpp example`
  actually prints a path — the canonical product contract (PRD §1: "resolves a
  human-friendly skill tag to the absolute filesystem path").
- It **locks the §6.4 `$()`-safety contract** that the entire product hinges on.
  `pi --skill "$(skpp tag)"` only works if `skpp` prints exactly one clean path on
  success and **nothing** on failure. The atomicity design (resolve-all-then-flush)
  is the load-bearing decision and is verified by both unit test and binary smoke.
- It **establishes the `<tag>` positional-capture rule** (`parseArgs` default
  branch: non-dashed token → tag) that every later positional feature inherits
  (`--all` reuse, `check` subcommand, §6.3 exclusivity in M5).
- It is **go.mod-neutral**: `main.go` gains one stdlib import (`strings`) and one
  internal import (`resolve`); no new third-party dependency.
- It is the **last M3 step before the modifiers (S2)**: S2 adds `--file`/`-f`,
  `--relative`, and `--all`/`-a` as thin variations on this exact branch (swap which
  string is printed per result; add an `--all` mode that feeds `Index` order into
  the same printer). Getting the buffered-atomicity loop right now makes S2 trivial.

---

## What

Two `main` edits (the complete verified source is in the Implementation Blueprint):

**`parseArgs`** — the `default` branch changes from "tolerate everything" to
"distinguish positionals from flags":
- A token that does **not** start with `-` is a positional `<tag>` → appended to
  `c.tags`.
- A token that starts with `-` is a flag: a known one sets its config field (as
  today); an unknown one (`--frobnicate`) is still tolerated (no-op) → exit-2 lands
  in M5.T11.

**`run`** — a new branch `if len(c.tags) > 0 { ... }` placed **after** `--list` and
**before** the default `return 1`:
1. `skillsdir.Find()` → on `ErrNotFound`, print the one-line fix to stderr, exit 1,
   nothing on stdout (identical to `--path`/`--list`).
2. `discover.Index(dir)` → on error, stderr + exit 1, nothing on stdout.
3. Loop every tag: `resolve.Resolve(tag, skills)`. On success append
   `res.Skill.Dir` to a **buffered** `paths []string`. On error print the verbatim
   `err.Error()` to stderr (one line per problem tag), set `hadErr`, `continue`.
4. If `hadErr`: `return 1` — the buffered `paths` is **never written**, so stdout
   stays empty (§6.4). Else: write one path per line (input order), `return 0`.

`config` gains one field: `tags []string`.

### Success Criteria

- [ ] `config` has a `tags []string` field; `parseArgs` captures non-dashed tokens
      into `c.tags` in input order and leaves dashed unknowns tolerated (no-op).
- [ ] `run` has the `if len(c.tags) > 0 { ... }` branch **after** `--list` and
      **before** the default `return 1`; it calls `skillsdir.Find()` →
      `discover.Index()` → `resolve.Resolve()` per tag.
- [ ] **Atomicity:** when any tag fails, **nothing** is written to stdout; one error
      line per problem tag goes to stderr; exit 1. (Verified by
      `TestRunTagAtomicityUnknownPrintsNothing` + binary smoke.)
- [ ] On full success, one **absolute** skill-directory path is printed per tag, in
      **input order** (not sorted), exit 0. (Verified by
      `TestRunTagMultipleInInputOrder` + `TestRunTagPathIsAbsolute`.)
- [ ] Ambiguous tag → stderr lists the candidate full tags (resolve pre-sorts them),
      nothing on stdout, exit 1. (Verified by `TestRunTagAmbiguousListsCandidates`.)
- [ ] Skills dir unresolvable + tags → exit 1, empty stdout, the one-line fix on
      stderr. (Verified by `TestRunTagSkillsDirUnresolvable`.)
- [ ] `--version` still precedes tag mode (PRD §6.3). (Verified by
      `TestRunVersionPrecedenceOverTag`.)
- [ ] `main.go` imports exactly: `fmt`, `io`, `os`, `strings`, `internal/discover`,
      `internal/resolve`, `internal/skillsdir`, `internal/ui`.
- [ ] `gofmt -l main.go main_test.go` silent; `go vet ./...` clean; `go build ./...`
      + `go test ./...` pass (main = 36); `go mod tidy` no-op; `go.mod`/`go.sum`
      unchanged; nothing under `internal/` touched.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `main.go` (complete file) and the precise `main_test.go`
edits (1 replace + 1 append) are given verbatim in the Implementation Blueprint and
have been **compiled, gofmt'd, vetted, unit-tested, and smoke-tested against a real
built binary** in `/tmp/skpp_t8_verify` (go1.25, verbatim module copy). Every
load-bearing decision is documented in `research/verified_facts.md` and the Known
Gotchas below: the positional-capture rule; the buffered-atomicity loop; the
absolute-dir default output; verbatim error lines (no `skpp:` prefix); the `run`
branch ordering; the S2/M4/M5 scope boundary. The consumed contracts —
`resolve.Resolve`/`Result`/`*UnknownError`/`*AmbiguousError`, `discover.Index`/
`discover.Skill`, `skillsdir.Find` — are LANDED on disk and were read in full. An
implementer who knows Go but nothing about this repo can complete this in one pass
by applying the two edits verbatim._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M3T8S1/research/verified_facts.md
  why: "Proves (against a verbatim module copy on go1.25): (1) the 4 main.go edits
        compile + pass gofmt + vet; (2) the §6.4 atomicity loop works — buffered
        paths, flushed only on full success, stdout EMPTY on any failure; (3) main
        goes 24→36 tests; (4) go.mod/go.sum byte-identical (no new dep); (5) the
        six end-to-end binary behaviors (single/multi-order/atomicity/unknown/
        ambiguous/duplicate) all match §6.1/§6.4; (6) the positional-capture rule;
        (7) the verbatim-error-line decision; (8) the S2/M4/M5 scope boundary,
        incl. the `skpp check` interim behavior."
  critical: "The EXACT main.go in the blueprint is gofmt-clean AS WRITTEN (run
             `gofmt -w` to be safe; the only reformat found in research was struct-
             field alignment). Write the test edits verbatim. Do NOT add --file/
             --relative/--all (S2), --search/check (M4), or --help/exit-2 (M5)."

# CONTRACT — resolve.Resolve / Result / typed errors (LANDED; consumed verbatim by main)
- file: internal/resolve/resolve.go
  why: "The brain main calls once per <tag>. Signature:
        func Resolve(tag string, skills []discover.Skill) (Result, error).
        Result{Skill discover.Skill; Match MatchKind}. On success main prints
        res.Skill.Dir (the ABSOLUTE skill dir — set by discover.Index). On failure
        it returns *UnknownError{Tag} (.Error() = `unknown skill tag \"foo\"`) or
        *AmbiguousError{Tag, Candidates} (.Error() = `ambiguous skill tag \"reddit\"
        matches: coding/reddit, writing/reddit` — Candidates pre-SORTED by resolve).
        main prints these VERBATIM to stderr (no reformatting, no skpp: prefix).
        READ-ONLY — do NOT modify resolve."
  pattern: "res, err := resolve.Resolve(tag, skills); if err != nil { stderr; hadErr=true; continue }"
  gotcha: "main does NOT need errors.As here — it just prints err.Error() per
           problem tag. (errors.As is how a FUTURE verbose/debug mode would branch
           on type; not needed for the §6.4 contract.)"

# CONTRACT — discover.Index / discover.Skill (LANDED; the index main builds + the Dir it prints)
- file: internal/discover/index.go
  why: "func Index(absSkillsDir string) ([]discover.Skill, error): WalkDir, sorted
        by RelTag, every Dir ABSOLUTE. main calls it ONCE (not per tag) and passes
        the []Skill to resolve.Resolve. Empty store -> Index returns ([], nil) ->
        every tag resolves to *UnknownError -> hadErr -> exit 1 (correct: an empty
        store cannot resolve any tag). READ-ONLY."
  pattern: "skills, err := discover.Index(dir); if err != nil { stderr; exit 1 }"
- file: internal/discover/skill.go
  why: "Skill.Dir is the ABSOLUTE skill-directory path (PRD §6.1 default output).
        Skill.SourceFile is the SKILL.md path (--file, S2, prints that instead).
        READ-ONLY."
  gotcha: "Dir is already absolute and Clean'd by Index — main prints it verbatim
           with Fprintln (path + newline); no further cleaning needed."

# CONTRACT — skillsdir.Find (the store resolution main calls first)
- file: internal/skillsdir/skillsdir.go
  why: "Find() (dir string, src Source, err error). On all-miss returns ('', 0,
        ErrNotFound); err.Error() is the user-facing one-line fix (printed verbatim
        to stderr, same as --path/--list). src is for reporting only — not printed.
        READ-ONLY."
  pattern: "dir, _, err := skillsdir.Find(); if err != nil { fmt.Fprintln(stderr, err); return 1 }"

# CONTRACT — the dispatcher this subtask EXTENDS (read in full before editing)
- file: main.go
  why: "The LANDED main.go (M1.T3 + M2.T6): config{version,path,list,noColor};
        parseArgs switch; run(args,stdout,stderr)int with version-precedence -> path
        -> list -> default(1). T8.S1 ADDS tags to config, the positional-capture in
        parseArgs's default, +strings/+resolve imports, and the <tag> branch in run.
        The version/path/list/default logic is PRESERVED verbatim. MODIFIED by this
        subtask (blueprint gives the complete new file)."
  gotcha: "Keep the branch ORDER: version -> path -> list -> <tag> -> default(1).
           --version must stay first (§6.3 precedence). Put <tag> AFTER --list and
           BEFORE the default return 1."

# PREDECESSOR PRP — T7's downstream extension contract for T8 (what this implements)
- file: plan/001_fcde63e5bb60/P1M3T7S1/PRP.md
  why: "T7's 'DOWNSTREAM CONSUMERS' locked the T8 contract verbatim: 'P1.M3.T8.S1
        (main tag-resolution output): calls discover.Index() once, then
        resolve.Resolve(tag, index) per <tag> arg. On *AmbiguousError/*UnknownError
        it prints one stderr line per problem tag, prints NOTHING to stdout, exits 1.
        Uses errors.As(err, &ae) to read ae.Candidates for the list-candidates
        requirement.' This PRP implements exactly that (errors.As is optional here —
        resolve already formats candidates into .Error(); main prints verbatim)."
  section: "Integration Points > DOWNSTREAM CONSUMERS"

# CONTRACT — the architecture design (locks the data flow + output discipline)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "'Data flow' shows main.parseArgs -> skillsdir.Find -> discover.Index ->
        resolve.Resolve(tag, index) per tag -> 'print paths'. 'Output discipline
        (§6.4) — critical' states verbatim: 'Resolve ALL tags first, collect errors.
        If ANY tag fails -> print one error line per problem to STDERR, print NOTHING
        to stdout, exit 1. Never partially print.' and 'Buffer stdout writes; only
        flush when the whole invocation is known-good.' This is EXACTLY the buffered
        loop this PRP implements. 'Exit codes': 0 success; 1 unresolved/ambiguous
        tag; 2 (M5) unknown flag/exclusivity."
  section: "Data flow", "Output discipline (§6.4)", "Exit codes"

# CONTRACT — the PRD sections this implements
- file: PRD.md
  why: "§6.1 `skpp <tag> [<tag>...]`: 'One ABSOLUTE path per line, in INPUT order.
        exit 0 if all resolve; 1 if any fail (and NOTHING is printed).' §6.4: 'Any
        unresolved/ambiguous tag -> print one error line per problem tag to stderr,
        print nothing to stdout, exit 1. Ambiguous -> stderr lists candidate full
        tags.' §6.3: --help/--version precedence. §13 acceptance: `test -d
        \"$(./skpp example)\"`, absolute-path contract, unknown-tag contract.
        READ-ONLY."
  critical: "§6.1 'nothing is printed [on any failure]' is THE contract — buffered
             output, flush only on full success. §6.4 lists candidates on stderr
             (resolve pre-sorts them)."

# REFERENCE — the repo's test convention (white-box main, injected writers, SKPP_SKILLS_DIR)
- file: main_test.go
  why: "Established convention: `package main` (white-box), `*bytes.Buffer` for
        stdout/stderr, `t.Setenv(\"SKPP_SKILLS_DIR\", dir)` (rule 1 wins
        deterministically), `t.Chdir(t.TempDir())` to force all §8 rules to miss,
        `writeSkillTree`/`withTerminal` helpers, plain t.Errorf/t.Fatalf, NO
        testify, NO t.Parallel(). The new tests mirror this exactly (they reuse
        `writeSkillTree`, `unsetSkillsEnv`, `version`). MODIFIED by this subtask."

# URLS — the stdlib surface this subtask is built from
- url: https://pkg.go.dev/strings#HasPrefix
  why: "strings.HasPrefix(a, \"-\") is the positional-vs-flag test in parseArgs.
        A tag never starts with '-'; a flag always does. Cheap, obvious, no import
        beyond the already-needed strings."
- url: https://pkg.go.dev/fmt#Fprintln
  why: "fmt.Fprintln(w, x) writes x + newline. Used for every output line: the
        buffered paths (stdout), and the verbatim error lines / one-line fix
        (stderr). Fprintln(stdout, p) gives the byte-exact 'path + newline' the §13
        `test -d \"$(./skpp example)\"` gate depends on."
```

### Current Codebase tree (M1 + M2 + resolve LANDED; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*' | sort
internal/discover/discover.go        # M2.T4.S1: Frontmatter + ParseFrontmatter
internal/discover/discover_test.go
internal/discover/skill.go           # M2.T4.S2: Skill + BuildSkill (Dir is ABSOLUTE)
internal/discover/skill_test.go
internal/discover/index.go           # M2.T5.S1: Index(dir)([]Skill,error) + sort
internal/discover/index_test.go
internal/skillsdir/skillsdir.go      # M1.T2: Source + Find + per-rule helpers
internal/skillsdir/skillsdir_test.go
internal/ui/ui.go                    # M2.T6.S1: PrintList table + ANSI
internal/ui/ui_test.go
internal/resolve/resolve.go          # M3.T7.S1: Resolve + Result + typed errors  [LANDED]
internal/resolve/resolve_test.go
main.go                              # M1.T3 + M2.T6: version/path/list dispatch  [THIS subtask MODIFIES]
main_test.go                         # M1.T3 + M2.T6: 24 tests                    [THIS subtask MODIFIES]

# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT)
# baseline: go build ./... OK; go test ./... OK (discover + skillsdir + ui + resolve + main green)
# resolve.Resolve LANDED. NO skills/ dir yet (P1.M6.T12 ships skills/example).
```

### Desired Codebase tree with files to be modified

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/discover/*,
│        internal/skillsdir/*, internal/ui/*, internal/resolve/* — ALL UNCHANGED)
├── main.go         # MODIFY — +strings/+resolve imports, +tags field, +positional
│                   #          capture in parseArgs, +<tag>-resolution branch in run
└── main_test.go    # MODIFY — update TestParseArgsUnknownTolerated; +3 parseArgs
                    #          tag tests; +8 run tag-resolution tests (24 -> 36)
```

| File | Action | Responsibility | Imports added |
|---|---|---|---|
| `main.go` | MODIFY | Wire `skpp <tag> [...]`: capture positionals, resolve, buffered-atomic output | `strings`, `internal/resolve` |
| `main_test.go` | MODIFY | Tests for tag capture + resolution + §6.4 atomicity + ambiguity | (none — reuses existing bytes/filepath/strings/testing) |

**Two modified files. Zero new files. Zero changes to `go.mod`, `go.sum`,
`internal/*`, `PRD.md`. No `skills/`, `install.sh`, `README`, completions,
`--file`/`--relative`/`--all`/`--search`/`check`/`--help`.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — Positional vs flag: a token NOT starting with '-' is a <tag>.
// parseArgs's default branch now does `if !strings.HasPrefix(a, "-") { c.tags =
// append(c.tags, a) }`. A dashed token (--frobnicate, -x) is a FLAG: known flags
// set their config field earlier in the switch; unknown dashed flags fall through
// to the default and are tolerated (no-op) -> exit-2 lands in M5.T11. This needs
// NO subcommand special-casing in S1. VERIFIED (TestParseArgsDashedUnknownNotATag).
//   RIGHT: if !strings.HasPrefix(a, "-") { c.tags = append(c.tags, a) }
//   WRONG: capturing "--frobnicate" as a tag, or special-casing "check" now.

// GOTCHA #2 — ATOMICITY via buffering; never print partial output (PRD §6.4).
// Resolve EVERY tag first into a buffered `paths []string`. On any per-tag error,
// print the verbatim err to stderr, set hadErr, CONTINUE (so every problem tag gets
// its own stderr line). Only AFTER the loop: if hadErr -> return 1 (paths NEVER
// written -> stdout empty); else flush all paths. Printing incrementally to stdout
// would leak a partial result on a later failure and break `pi --skill "$(skpp a b)"`
// when b is bad. VERIFIED (TestRunTagAtomicityUnknownPrintsNothing: stdout EMPTY).
//   RIGHT: paths := make([]string,0,len(c.tags)); ...; if hadErr { return 1 }; for p ...
//   WRONG: fmt.Fprintln(stdout, res.Skill.Dir) inside the loop   // leaks on later fail

// GOTCHA #3 — One error line PER PROBLEM TAG (not just the first).
// PRD §6.4: "print one error line per problem tag". So on error you CONTINUE the
// loop (not return), so a second bad tag also gets its stderr line. The resolvable
// tags between them produce NO stderr line (they just populate paths, which is then
// discarded). VERIFIED (TestRunTagAllFailMultipleErrorLines: 2 tags -> 2 stderr lines).
//   RIGHT: if rerr != nil { fmt.Fprintln(stderr, rerr); hadErr = true; continue }
//   WRONG: return 1 on the first error   // only one stderr line; later bad tags silenced

// GOTCHA #4 — Print err.Error() VERBATIM; no "skpp:" prefix.
// skillsdir.ErrNotFound is printed verbatim by --path/--list (no prefix); match that
// convention for tag errors. resolve already formats them readably:
//   *UnknownError  -> `unknown skill tag "foo"`
//   *AmbiguousError -> `ambiguous skill tag "reddit" matches: coding/reddit, writing/reddit`
// (Candidates are pre-SORTED by resolve, so stderr is stable for scripting, §6.4.)
// main does NOT need errors.As to satisfy §6.4 — it just prints err. VERIFIED (binary
// smoke: stderr matches the resolve .Error() text exactly).
//   RIGHT: fmt.Fprintln(stderr, rerr)
//   WRONG: fmt.Fprintf(stderr, "skpp: %v\n", rerr)   // inconsistent with --path/--list

// GOTCHA #5 — Default output is the skill DIRECTORY path (res.Skill.Dir), ABSOLUTE.
// PRD §6.1: "One absolute path per line". discover.Index sets Skill.Dir to the
// absolute, Clean'd skill dir, so main prints it verbatim (Fprintln -> path + \n).
// --file (-> Skill.SourceFile, the SKILL.md path) and --relative are S2 — do NOT
// add them here. VERIFIED (TestRunTagPathIsAbsolute + binary `skpp example`).
//   RIGHT: paths = append(paths, res.Skill.Dir)
//   WRONG: printing res.Skill.SourceFile, or re-Clean'ing Dir, or adding --file now.

// GOTCHA #6 — run branch ORDER: version -> path -> list -> <tag> -> default(1).
// The <tag> branch goes AFTER --list and BEFORE the default return 1. --version
// stays FIRST (§6.3 precedence). Keep the existing --path/--list branches verbatim.
// Mixing <tag> with --list is TOLERATED here (--list wins because it is checked
// first); the §6.3 exit-2 mutual-exclusivity is M5.T11. VERIFIED
// (TestRunVersionPrecedenceOverTag; all existing version/path/list tests unchanged).
//   RIGHT: ... if c.path {...} if c.list {...} if len(c.tags)>0 {...} return 1
//   WRONG: putting <tag> before --list, or reordering the precedence tier.

// GOTCHA #7 — Find()/Index() errors short-circuit BEFORE the tag loop.
// If the skills dir is unresolvable or Index() errors, print the one-line fix /
// error to stderr and exit 1 — do NOT enter the per-tag loop (there is no index to
// resolve against). stdout stays empty (same contract as --path/--list). VERIFIED
// (TestRunTagSkillsDirUnresolvable).
//   RIGHT: dir,_,err := skillsdir.Find(); if err!=nil { stderr; return 1 }; ...Index...
//   WRONG: looping tags with a nil/empty index and relying on per-tag UnknownError
//          (loses the one-line-fix message and is the wrong failure shape).

// GOTCHA #8 — Duplicate tag in argv is NOT an error; it resolves twice.
// `skpp example example` resolves "example" twice and prints two identical path
// lines. Do NOT dedupe (PRD §6.1 says "in input order" — duplicates are the user's
// explicit choice). VERIFIED (TestRunTagDuplicateArgResolvesTwice).
//   RIGHT: loop c.tags verbatim; append per resolution.
//   WRONG: deduping tags, or erroring on a repeated tag.

// GOTCHA #9 — `skpp check` interim: "check" is a non-dashed token, so S1 captures
// it as a tag and tries to resolve it (-> *UnknownError -> exit 1). This matches
// the pre-S1 exit code (tolerated default -> exit 1); the only change is a stderr
// line now appears. M4.T10 adds real `check` subcommand dispatch (handled BEFORE
// tag resolution). A user who legitimately tags a skill "check" gets it resolved —
// correct, and does not paint M4 into a corner. Do NOT special-case "check" here.
// VERIFIED (TestParseArgsCapturesTagsInOrder treats arbitrary positionals uniformly).

// GOTCHA #10 — gofmt aligns the config struct fields; run `gofmt -w`.
// Adding `tags []string` (longer than `bool`) makes gofmt pad the bool fields'
// comments to align with the []string line. In research the only gofmt reformat was
// exactly this struct alignment. Run `gofmt -w main.go main_test.go` after editing;
// `gofmt -l` must then be empty. VERIFIED (research §2: gofmt -l empty after -w).
//   RIGHT: version bool     // ...   (padded to align under tags []string // ...)
//   WRONG: leaving the struct un-aligned (gofmt -l will flag main.go).

// GOTCHA #11 — go.mod/go.sum are UNCHANGED.
// main.go gains one STDLIB import (strings) and one INTERNAL import (resolve).
// No new third-party dependency. `go mod tidy` is a no-op. If it changes anything,
// something is wrong (you added an external import). VERIFIED (research §2: go.mod/
// go.sum byte-identical). Confirm with `git diff --quiet go.mod go.sum`.

// GOTCHA #12 — Reuse the existing test helpers; do NOT add new imports to main_test.go.
// The new tests reuse `writeSkillTree`, `unsetSkillsEnv`, and the `version` var
// already in main_test.go, plus bytes/filepath/strings/testing already imported.
// Add NO new import. A small `sampleStore(t)` helper (2 skills: example +
// writing/reddit) is added to keep the run tests readable. VERIFIED (main_test.go
// imports unchanged; compiles clean).
```

---

## Implementation Blueprint

### Data model — `config` gains one field; no new types

```go
type config struct {
	version bool     // --version / -v
	path    bool     // --path / -p
	list    bool     // --list / -l
	noColor bool     // --no-color
	tags    []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run   [NEW]
}
```

No new types. The branch consumes `resolve.Result`/`resolve.Resolve` and
`discover.Index`/`discover.Skill`/`skillsdir.Find` verbatim.

### File 1 — `main.go` (MODIFY — write this complete file over the existing one)

The complete updated `main.go` (gofmt-clean; all 36 main tests pass; **compiled +
vetted + smoke-tested** against the real module in `/tmp/skpp_t8_verify`). It
preserves the M1.T3 + M2.T6 version/path/list/default logic verbatim and adds the
two imports, the `tags` field, the positional capture, and the `<tag>` branch:

````go
// Command skpp resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// Wired so far (grown milestone by milestone): --version/--path (M1.T3) and
// --list (M2.T6). Every other §6 flag is added by later milestones (M3
// <tag>/--all, M4 --search/check, M5 --help + exit codes). The arg parser is
// intentionally a small hand-rolled switch (not Go's `flag` package) so the full
// §6 matrix — subcommands like `check`, positional <tag> args, long+short
// aliases, and §6.3 mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1T3.S1/research/verified_facts.md §5.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
)

// version is the skpp version string, printed by `skpp --version`. It is
// overridden at BUILD time via ldflags (PRD §12.1 build command):
//
//	go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .
//
// The default "dev" is used by `go run` and plain `go build` (no ldflags).
//
// IMPORTANT: this MUST be a package-level var, not a const. `-X main.version=...`
// rewrites a package-scope string var at link time; it cannot override a const
// (compile error) or a function-local. Because this file is `package main`, the
// linker symbol path is `main.version` (NOT the module import path).
var version = "dev"

// isTerminal reports whether w is an interactive terminal (a character device).
// It decides whether --list/--search emit ANSI color by default (PRD §6.2: color
// is on for a TTY unless --no-color is set). It type-asserts w to *os.File and
// checks the ModeCharDevice bit, so a *bytes.Buffer (tests) or a pipe/redirect
// correctly yields false -> no color, keeping output deterministic and pipe-safe.
//
// It is a package var so tests can override it to exercise the color-enabled path
// through run() without a real terminal. NOT safe for t.Parallel (mutates package
// state); the repo convention is no t.Parallel() on such tests anyway.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// config holds the parsed CLI flags. Grown by later milestones as more of the
// PRD §6.1/§6.2 matrix lands. For this subtask version, path, list, noColor, and
// tags are set; every other token is a tolerated no-op (P1.M5.T11 turns unknown
// flags into exit 2 and adds subcommand handling).
type config struct {
	version bool     // --version / -v : print "skpp <version>" and exit 0
	path    bool     // --path / -p    : print resolved skills dir and exit 0/1
	list    bool     // --list / -l    : print the human-readable catalog table (§6.1)
	noColor bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	tags    []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run [NEW]
	// Future (S2/M4/M5), do NOT add yet:
	//   all bool; search string; check bool; file, relative, help bool
}

// parseArgs scans argv tokens and fills a config. Flags may appear in any order
// (PRD §6). Long forms use POSIX double-dash; short forms a single dash. Unknown
// dashed flags are tolerated for now (a no-op in the default branch); the full
// unknown-flag -> exit 2 behavior and §6.3 mutual-exclusivity land in P1.M5.T11.
//
// To add a flag in a later milestone: append a `case "--name", "-n": cfg.name =
// true` (or capture the next arg for value-taking flags like --search <q>).
func parseArgs(args []string) config {
	var c config
	for _, a := range args {
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--no-color":
			c.noColor = true
		default:
			// Positional <tag> (PRD §6.1 `skpp <tag> [<tag>...]`): a token that
			// does NOT start with '-' is a tag, captured here and resolved in run.
			// Dashed unknowns (e.g. --frobnicate) also fall through to this default
			// and are tolerated (no-op); P1.M5.T11 turns them into exit 2 and adds
			// §6.3 mutual-exclusivity (tag mixed with --list/--search/--all). The
			// --file/--relative/--all modifiers land in P1.M3.T8.S2.
			if !strings.HasPrefix(a, "-") {
				c.tags = append(c.tags, a)
			}
		}
	}
	return c
}

// run is the testable dispatcher. It returns the process exit code so main() can
// call os.Exit(run(...)) without tests ever invoking os.Exit. stdout/stderr are
// injected so tests capture output via *bytes.Buffer instead of the real streams.
//
// Exit codes (PRD §6; this subtask's slice):
//   - 0: --version printed; --path succeeded; --list printed the catalog; all
//     <tag>s resolved (one absolute path per line printed)
//   - 1: --path/--list failed; ANY <tag> unresolved/ambiguous (nothing on stdout);
//     skills dir unresolvable; default (no recognized flag)
//   - 2: (DEFERRED to P1.M5.T11) unknown flag / mutually-exclusive modes mixed
//
// Precedence (PRD §6.3): --version (and, in M5, --help) win over everything.
func run(args []string, stdout, stderr io.Writer) int {
	c := parseArgs(args)

	// Precedence tier: --version wins over every other flag (PRD §6.3).
	// P1.M5.T11 adds --help/-h to this same tier (before --path).
	if c.version {
		fmt.Fprintf(stdout, "skpp %s\n", version)
		return 0
	}

	if c.path {
		dir, _, err := skillsdir.Find() // src is for reporting only; not printed
		if err != nil {
			// Find() returns skillsdir.ErrNotFound whose message is the
			// user-facing one-line fix (PRD §8.4/§6.4). Print it verbatim to
			// stderr (NOT stdout) so $(...) stays empty on failure.
			fmt.Fprintln(stderr, err)
			return 1
		}
		// Byte-exact: ONLY the dir + newline. The §13 acceptance gate
		// `test "$(./skpp --path)" = "$PWD/skills"` depends on this.
		fmt.Fprintln(stdout, dir)
		return 0
	}

	if c.list {
		// PRD §6.1 `skpp --list`: resolve the store, build the index, render the
		// table. This is the FIRST place the Find() -> discover.Index() data flow
		// is wired end-to-end (M2.T6). Exit 1 on any failure path.
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // verbatim one-line fix (PRD §6.4/§8.4)
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		if len(skills) == 0 {
			// PRD §6.1: --list exits 1 "if no skills found". Message to stderr so
			// stdout stays clean for any consumer.
			fmt.Fprintln(stderr, "no skills found in "+dir)
			return 1
		}
		// Color only when stdout is a TTY AND --no-color was not given (PRD §6.2).
		// A *bytes.Buffer (tests) / pipe / file is not a TTY -> plain output.
		ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// Tag-resolution mode: `skpp <tag> [<tag>...]` (PRD §6.1). Resolves each tag to
	// its absolute skill dir path and prints one path per line, in input order.
	//
	// ATOMICITY (PRD §6.4 — the critical-for-$(...) contract): resolve EVERY tag
	// first, buffering the resulting paths; if ANY tag fails (unknown/ambiguous),
	// print one error line per problem tag to stderr, print NOTHING to stdout, and
	// exit 1. The buffered paths are flushed ONLY when the whole invocation is
	// known-good. This makes `pi --skill "$(skpp bad)"` fail loudly (empty $(),
	// exit 1) instead of passing a partial or garbage path. Each error is printed
	// verbatim from resolve's typed errors — UnknownError names the tag,
	// AmbiguousError lists the candidate full tags (no "skpp:" prefix, matching the
	// skillsdir.ErrNotFound convention used by --path/--list). The default output is
	// the skill DIRECTORY path; --file/--relative modifiers land in P1.M3.T8.S2.
	if len(c.tags) > 0 {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		paths := make([]string, 0, len(c.tags)) // buffered; flushed only if all resolve
		hadErr := false
		for _, tag := range c.tags {
			res, rerr := resolve.Resolve(tag, skills)
			if rerr != nil {
				fmt.Fprintln(stderr, rerr) // one error line per problem tag (verbatim)
				hadErr = true
				continue
			}
			paths = append(paths, res.Skill.Dir) // absolute dir path (PRD §6.1 default)
		}
		if hadErr {
			return 1 // paths buffered but never written → stdout empty (§6.4)
		}
		for _, p := range paths {
			fmt.Fprintln(stdout, p) // one absolute path per line, input order
		}
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}
````

### File 2 — `main_test.go` (MODIFY — two edits: 1 replace + 1 append)

**Edit A — REPLACE** the existing `TestParseArgsUnknownTolerated` (its assertions
still hold, but the test must now reflect that non-dashed positionals are captured
as tags rather than discarded):

````go
// Dashed unknown flags are tolerated (no-op; exit-2 is M5). Non-dashed positional
// tokens are now captured as <tag>s (so "sometag"/"check" land in c.tags rather
// than being discarded). Only version/path stay false here.
func TestParseArgsUnknownTolerated(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "sometag", "check"})
	if c.version || c.path {
		t.Errorf("parseArgs(unknown): version=%v path=%v; want both false", c.version, c.path)
	}
	// Non-dashed positionals are captured as tags; the dashed --frobnicate is excluded.
	if len(c.tags) != 2 || c.tags[0] != "sometag" || c.tags[1] != "check" {
		t.Errorf("parseArgs tags=%v; want [sometag check] (positionals captured)", c.tags)
	}
}
````

**Edit B — APPEND** to the end of `main_test.go` (the 3 new parseArgs tests + the
`sampleStore` helper + the 8 new `run` tag-resolution tests). Reuses the existing
`writeSkillTree`, `unsetSkillsEnv`, and `version`; adds **no new import**:

````go

// --- parseArgs: positional <tag> capture (P1.M3.T8.S1) ---

// Positional <tag> args (non-dashed tokens) are captured in INPUT order (PRD §6.1).
func TestParseArgsCapturesTagsInOrder(t *testing.T) {
	c := parseArgs([]string{"foo", "writing/reddit"})
	if len(c.tags) != 2 || c.tags[0] != "foo" || c.tags[1] != "writing/reddit" {
		t.Errorf("tags=%v; want [foo writing/reddit] in input order", c.tags)
	}
}

// Dashed unknowns are NOT tags (they are tolerated flags); only the positional is captured.
func TestParseArgsDashedUnknownNotATag(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "real-tag", "-x"})
	if len(c.tags) != 1 || c.tags[0] != "real-tag" {
		t.Errorf("tags=%v; want [real-tag] (dashed tokens excluded)", c.tags)
	}
}

// Tags and recognized flags may interleave (PRD §6: flags appear in any order).
func TestParseArgsTagsAndFlagsInterleave(t *testing.T) {
	c := parseArgs([]string{"--no-color", "a", "-l", "b"})
	if !c.list || !c.noColor || len(c.tags) != 2 || c.tags[0] != "a" || c.tags[1] != "b" {
		t.Errorf("config=%+v; want list+noColor true and tags=[a b]", c)
	}
}

// --- run: <tag> resolution (P1.M3.T8.S1) ---

// sampleStore builds a store with a top-level `example` and a nested
// `writing/reddit` skill, returning the skills dir (set via SKPP_SKILLS_DIR rule 1).
func sampleStore(t *testing.T) string {
	t.Helper()
	return writeSkillTree(t, map[string]string{
		"example":        "---\nname: example\ndescription: A demo skill.\n---\n# body\n",
		"writing/reddit": "---\nname: reddit-poster\ndescription: Posts to reddit.\n---\n# body\n",
	})
}

// Single tag resolves to its absolute skill DIRECTORY path on stdout, exit 0, no
// stderr. The default output is the dir, not SKILL.md (--file is P1.M3.T8.S2).
func TestRunTagSingleResolvesToDir(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example): code=%d; want 0", code)
	}
	want := filepath.Join(dir, "example") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(example) stdout=%q; want %q (absolute dir + newline)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(example) stderr=%q; want empty", errOut.String())
	}
}

// Multiple tags -> one path per line, in INPUT order (not sorted), exit 0. `reddit`
// resolves by basename to writing/reddit; `example` by canonical tag.
func TestRunTagMultipleInInputOrder(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"reddit", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(reddit example): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	if lines[0] != filepath.Join(dir, "writing", "reddit") {
		t.Errorf("lines[0]=%q; want the reddit dir (input order preserved)", lines[0])
	}
	if lines[1] != filepath.Join(dir, "example") {
		t.Errorf("lines[1]=%q; want the example dir (input order preserved)", lines[1])
	}
}

// ATOMICITY (§6.4): one unknown tag among resolvable ones -> NOTHING on stdout, one
// stderr line per problem tag, exit 1. The resolvable tag must NOT leak to stdout.
func TestRunTagAtomicityUnknownPrintsNothing(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example", "nope"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(example nope): code=%d; want 1 (atomic failure)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (§6.4: nothing printed on failure)", out.String())
	}
	if !strings.Contains(errOut.String(), "nope") {
		t.Errorf("stderr=%q; want an error line naming 'nope'", errOut.String())
	}
}

// All tags fail -> one stderr line per problem tag, nothing on stdout, exit 1.
func TestRunTagAllFailMultipleErrorLines(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"nope1", "nope2"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(nope1 nope2): code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY", out.String())
	}
	errLines := strings.Split(strings.TrimRight(errOut.String(), "\n"), "\n")
	if len(errLines) != 2 {
		t.Fatalf("want 2 stderr lines (one per problem tag); got %d: %q", len(errLines), errOut.String())
	}
}

// A tag repeated in argv resolves each time; output repeats. Not an error.
func TestRunTagDuplicateArgResolvesTwice(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example example): code=%d; want 0", code)
	}
	want := strings.Repeat(filepath.Join(dir, "example")+"\n", 2)
	if got := out.String(); got != want {
		t.Errorf("stdout=%q; want two identical path lines:\n%s", got, want)
	}
}

// Ambiguous tag (basename collision) -> stderr lists the candidate full tags,
// NOTHING on stdout, exit 1 (PRD §6.4).
func TestRunTagAmbiguousListsCandidates(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"writing/reddit": "---\nname: a\ndescription: d\n---\nx\n",
		"coding/reddit":  "---\nname: b\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"reddit"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(reddit) ambiguous: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (ambiguous => nothing on stdout)", out.String())
	}
	msg := errOut.String()
	for _, want := range []string{"reddit", "coding/reddit", "writing/reddit"} {
		if !strings.Contains(msg, want) {
			t.Errorf("stderr=%q; missing candidate %q", msg, want)
		}
	}
}

// Skills dir unresolvable + tags -> exit 1, nothing on stdout, the one-line fix on
// stderr (same contract as --path/--list).
func TestRunTagSkillsDirUnresolvable(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // all three §8 rules miss
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(example) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("stderr=%q; want the one-line fix", errOut.String())
	}
}

// The resolved path is ABSOLUTE (PRD §6.1 default; --relative is P1.M3.T8.S2).
func TestRunTagPathIsAbsolute(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example): code=%d; want 0", code)
	}
	if p := strings.TrimRight(out.String(), "\n"); !filepath.IsAbs(p) {
		t.Errorf("resolved path %q is not absolute (discover.Skill.Dir should be absolute)", p)
	}
}

// --version precedes tag-resolution mode even when a tag is present (PRD §6.3).
func TestRunVersionPrecedenceOverTag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example --version): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (precedence over tag mode)", got)
	}
}
````

> **Copy-paste correctness:** both edits are gofmt-clean and were compiled + tested
> verbatim against the real module in `/tmp/skpp_t8_verify` (go1.25). main.go imports
> exactly fmt/io/os/strings/discover/resolve/skillsdir/ui; main_test.go imports are
> unchanged (bytes/io/os/path/filepath/strings/testing). Apply Edit A (replace the
> one test) and Edit B (append the block) and run the gates. The buffered-atomicity
> loop and the §6.4 stdout-empty contract trace directly to PRD §6.4 +
> go_architecture.md "Output discipline"; every assertion maps to a verified_facts entry.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0 (GATE): CONFIRM resolve.Resolve is LANDED before starting
  - COMMAND: grep -q 'func Resolve' internal/resolve/resolve.go
  - EXPECT: match (P1.M3.T7.S1 LANDED resolve, commit c7f4f99). If it does NOT exist,
            STOP: this subtask cannot compile until T7 is done. (T7 is COMPLETE, so
            this passes.) Also confirm the typed-error shapes:
            grep -n 'unknown skill tag\|ambiguous skill tag' internal/resolve/resolve.go
            (main prints these .Error() strings verbatim to stderr.)

Task 1: MODIFY main.go  (write the complete File 1 over the existing main.go)
  - WRITE: the exact content from the Blueprint (File 1).
  - CHECK the 4 additions are present:
      (a) imports include "strings" + "github.com/dabstractor/skpp/internal/resolve";
      (b) config struct has `tags []string`;
      (c) parseArgs default branch: `if !strings.HasPrefix(a, "-") { c.tags = append(c.tags, a) }`;
      (d) run has `if len(c.tags) > 0 { ... }` AFTER --list and BEFORE the default
          return 1, with the buffered paths + hadErr atomicity loop.
  - CHECK the version/path/list/default branches are PRESERVED verbatim.
  - GOTCHA: buffered atomicity (resolve all, flush only if !hadErr); verbatim error
            lines (no skpp: prefix); default output = res.Skill.Dir (absolute); Find/
            Index errors short-circuit before the loop; run `gofmt -w` (struct align).

Task 2: MODIFY main_test.go  (Edit A: replace TestParseArgsUnknownTolerated;
                             Edit B: append the 11 new tests + sampleStore helper)
  - REPLACE: the old TestParseArgsUnknownTolerated with the Blueprint's Edit A version.
  - APPEND: the Edit B block (3 parseArgs tests + sampleStore + 8 run tests).
  - CHECK: main_test.go imports UNCHANGED (bytes/io/os/path/filepath/strings/testing);
           the 12 new/updated tests are present; all 24 prior tests preserved verbatim.
  - GOTCHA: NO new imports; NO testify; NO t.Parallel(); reuse writeSkillTree/
            unsetSkillsEnv/version; SKPP_SKILLS_DIR (rule 1) + t.Chdir(TempDir) to
            force misses. Compare path output with filepath.Join (cross-platform).

Task 3: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w main.go main_test.go
  - COMMAND: gofmt -l main.go main_test.go   # MUST print nothing
  - COMMAND: go vet ./...                     # MUST be clean
  - COMMAND: go build ./...                   # exit 0 (whole module compiles)
  - COMMAND: go test . -v                     # ALL 36 main tests PASS (was 24)
  - COMMAND: go test ./...                    # whole module green
  - COMMAND: go mod tidy && git diff --quiet go.mod go.sum   # go.mod/go.sum UNCHANGED
  - EXPECT: zero errors, zero vet findings, gofmt silent, 36 main tests pass, no go.mod change.

Task 4: ACCEPTANCE SMOKE TEST (Level 3 in Validation Loop)
  - COMMAND: the Level 3 block below (build a real binary, point SKPP_SKILLS_DIR at a
             2-skill store, run the 6 §6.1/§6.4 scenarios, assert stdout/stderr/rc).
  - EXPECT: single -> absolute dir; multi -> input order; one-bad -> stdout EMPTY rc=1;
            unknown -> stderr `unknown skill tag "nope"`; ambiguous -> stderr lists
            candidates rc=1; duplicate -> repeats.

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: main.go has exactly the 4 additions (no --file/--relative/--all/--search/
            check/--help); imports correct; go.mod/go.sum unchanged; nothing under
            internal/ touched; no new files.
```

### Implementation Patterns & Key Details

```go
// PATTERN: positional-vs-flag test in parseArgs default branch.
//   default:
//       if !strings.HasPrefix(a, "-") { c.tags = append(c.tags, a) }
// WHY: a tag never starts with '-'; a flag always does. Known flags are matched by
//      earlier `case` arms; unknown dashed flags fall through here and are tolerated
//      (no-op) until M5 turns them into exit 2. Needs no subcommand special-casing.

// PATTERN: buffered atomicity (the §6.4 contract).
//   paths := make([]string, 0, len(c.tags))
//   hadErr := false
//   for _, tag := range c.tags {
//       res, rerr := resolve.Resolve(tag, skills)
//       if rerr != nil { fmt.Fprintln(stderr, rerr); hadErr = true; continue }
//       paths = append(paths, res.Skill.Dir)
//   }
//   if hadErr { return 1 }          // paths NEVER flushed -> stdout empty
//   for _, p := range paths { fmt.Fprintln(stdout, p) }
//   return 0
// WHY: PRD §6.4 — resolve ALL first, print NOTHING on any failure. Buffering +
//      flush-only-on-full-success guarantees `pi --skill "$(skpp a b)"` gets either
//      both paths or an empty $() + exit 1, never a partial/garbage path. CONTINUE
//      on error (not return) so every problem tag gets its own stderr line.

// PATTERN: verbatim error lines (no prefix), matching skillsdir convention.
//   fmt.Fprintln(stderr, rerr)   // rerr is *resolve.UnknownError or *AmbiguousError
// WHY: --path/--list print skillsdir.ErrNotFound verbatim (no "skpp:"); tag errors
//      match. resolve already formats them readably and pre-sorts Candidates, so
//      stderr is stable for scripting (§6.4). No errors.As needed for the contract.

// PATTERN: Find()/Index() errors short-circuit before the tag loop.
//   dir, _, err := skillsdir.Find();  if err != nil { fmt.Fprintln(stderr, err); return 1 }
//   skills, err := discover.Index(dir); if err != nil { fmt.Fprintln(stderr, err); return 1 }
// WHY: without a store/index there is nothing to resolve against; printing the
//      one-line fix (Find) or the Index error to stderr + exit 1 is the correct
//      failure shape, identical to --path/--list. stdout stays empty (§6.4).
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - main.go imports fmt, io, os, strings, internal/discover, internal/resolve,
    internal/skillsdir, internal/ui. (resolve is NEW in main's import set.)
  - main consumes: skillsdir.Find(), discover.Index(), discover.Skill.Dir,
    resolve.Resolve() / Result / *UnknownError / *AmbiguousError (via err.Error()).

go.mod / go.sum (NO change — verified_facts §2):
  - main gains one STDLIB import (strings) + one INTERNAL import (resolve). No new
    third-party dependency. `go mod tidy` is a no-op.
  - VERIFY: `go mod tidy && git diff --quiet go.mod go.sum` exits 0.

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into — designed for reuse):

  - P1.M3.T8.S2 (modifiers --file/-f, --relative, --all/-a):
      * --file/-f:  change ONE line — print res.Skill.SourceFile instead of
        res.Skill.Dir. Add `file bool` to config + a parseArgs case. The buffered
        atomicity loop is UNCHANGED.
      * --relative: print a path RELATIVE to the skills dir instead of absolute.
        Compute once: rel, _ := filepath.Rel(dir, res.Skill.Dir). (Needs importing
        path/filepath in main.go — currently absent; S2 adds it.)
      * --all/-a:   a NEW mode (not tag resolution) that feeds discover.Index's
        already-sorted []Skill into the SAME path-printer (one dir per line). Reuse
        the `for _, p := range paths { fmt.Fprintln(stdout, p) }` shape. Mutually
        exclusive with <tag> (§6.3, exit 2 in M5).

  - P1.M4.T9 (--search) and P1.M4.T10 (check): do NOT touch the <tag> branch.
    --search reuses ui.PrintList (already wired). `check` adds subcommand dispatch
    BEFORE the tag branch in run (so `skpp check` stops resolving "check" as a tag).

  - P1.M5.T11 (full CLI surface): adds --help/-h to the precedence tier (before
    --path); turns unknown dashed flags into exit 2; enforces §6.3 mutual-
    exclusivity (tag + --list/--search/--all -> exit 2). The tag branch itself is
    unchanged — M5 only adds a guard that rejects `c.tags` mixed with list/search/all.

NO CHANGES TO:
  - internal/discover/*, internal/skillsdir/*, internal/ui/*, internal/resolve/*
    (all consumed READ-ONLY).
  - go.mod, go.sum, .gitignore, LICENSE, PRD.md.
  - No new files; no skills/ dir (P1.M6.T12); no install.sh/README/completions.
```

---

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
cd /home/dustin/projects/skpp
gofmt -w main.go main_test.go        # apply formatting (struct-field alignment is the only change)
gofmt -l main.go main_test.go        # MUST print nothing
go vet ./...                         # MUST be clean
# Expected: zero output from gofmt -l; zero vet findings.
```

### Level 2: Unit Tests (Component Validation)

```bash
# The new tag-resolution tests in isolation
go test . -run 'TestRunTag|TestParseArgsCaptures|TestParseArgsDashed|TestParseArgsTagsAndFlags|TestParseArgsUnknownTolerated|TestRunVersionPrecedenceOverTag' -v

# Whole main package (expect 36 PASS: prior 24 + 12 new/updated)
go test . -v

# Whole module
go test ./...
# Expected: all pass. main package = 36 tests.
```

### Level 3: Integration Testing (System Validation — real binary, §13-style)

```bash
cd /home/dustin/projects/skpp
go build -o /tmp/skpp_t8_bin .
S=$(mktemp -d)/skills
mkdir -p "$S/example" "$S/writing/reddit" "$S/coding/reddit"
printf -- '---\nname: example\ndescription: A demo skill.\n---\n# body\n' > "$S/example/SKILL.md"
printf -- '---\nname: reddit-poster\ndescription: Posts to reddit.\n---\n# body\n' > "$S/writing/reddit/SKILL.md"
printf -- '---\nname: b\ndescription: d\n---\nx\n'                   > "$S/coding/reddit/SKILL.md"

# 1) single tag -> absolute dir path
SKPP_SKILLS_DIR="$S" /tmp/skpp_t8_bin example

# 2) two tags -> input order (reddit resolves by basename to writing/reddit)
SKPP_SKILLS_DIR="$S" /tmp/skpp_t8_bin reddit example

# 3) ATOMICITY: one bad among good -> stdout EMPTY, exit 1
out=$(SKPP_SKILLS_DIR="$S" /tmp/skpp_t8_bin example nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "atomicity OK" || echo "ATOMICITY FAIL: out=[$out] rc=$rc"

# 4) unknown -> stderr names the tag, stdout empty, exit 1
SKPP_SKILLS_DIR="$S" /tmp/skpp_t8_bin nope 2>&1 1>/dev/null

# 5) ambiguous -> stdout EMPTY, stderr lists sorted candidates, exit 1
out=$(SKPP_SKILLS_DIR="$S" /tmp/skpp_t8_bin reddit 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "ambiguous-empty OK" || echo "FAIL: out=[$out] rc=$rc"
SKPP_SKILLS_DIR="$S" /tmp/skpp_t8_bin reddit 2>&1 1>/dev/null   # -> ambiguous skill tag "reddit" matches: coding/reddit, writing/reddit

# 6) end-to-end with pi (skills load ONLY via --skill, never auto-discovered) — P1.M6 ships the example skill
# pi --no-skills --skill "$(/tmp/skpp_t8_bin example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
# (Defer the pi line until skills/example exists (P1.M6.T12); the binary contract above is this subtask's gate.)

rm -f /tmp/skpp_t8_bin; rm -rf "${S%/*}"
# Expected: 1) one absolute path; 2) two paths in input order; 3) "atomicity OK";
#           4) `unknown skill tag "nope"`; 5) "ambiguous-empty OK" + candidate line.
```

### Level 4: Creative & Domain-Specific Validation (scope boundary)

```bash
cd /home/dustin/projects/skpp
# main.go imports are EXACTLY the expected set (resolve + strings are the only adds):
go list -f '{{join .Imports " "}}' .
# Expected to include: fmt io os strings github.com/dabstractor/skpp/internal/discover
#                      github.com/dabstractor/skpp/internal/resolve
#                      github.com/dabstractor/skpp/internal/skillsdir
#                      github.com/dabstractor/skpp/internal/ui

# No out-of-scope flags leaked into main.go:
! grep -nE '\-\-(file|relative|all|search|help)\b|c\.all|c\.search|c\.file|c\.relative|c\.help' main.go \
  && echo "scope OK (no S2/M4/M5 flags in main.go)"

# go.mod/go.sum UNCHANGED:
go mod tidy && git diff --quiet go.mod go.sum && echo "go.mod/go.sum unchanged"

# Nothing under internal/ was touched, and no new files were created:
git status --porcelain internal/   # Expected: empty
git status --porcelain | grep -E '\?\?' | grep -vE '\.pi-subagents|plan/' || echo "no stray new files"
# Expected: "scope OK", "go.mod/go.sum unchanged", empty internal/ status, no stray files.
```

---

## Final Validation Checklist

### Technical Validation

- [ ] All 4 validation levels completed successfully.
- [ ] `gofmt -l main.go main_test.go` is silent (run `gofmt -w` first).
- [ ] `go vet ./...` is clean.
- [ ] `go build ./...` succeeds.
- [ ] `go test ./...` passes; main package = **36 tests** (prior 24 + 12 new/updated).
- [ ] `go mod tidy` is a no-op; `git diff --quiet go.mod go.sum` exits 0.

### Feature Validation (PRD §6.1 / §6.4)

- [ ] `skpp <tag>` prints one **absolute** skill-directory path + newline, exit 0.
- [ ] `skpp a b` prints one path per line **in input order** (not sorted), exit 0.
- [ ] **Atomicity:** any unresolved/ambiguous tag ⇒ **nothing** on stdout, one error
      line per problem tag on stderr, exit 1 (Level 3 step 3).
- [ ] Unknown tag ⇒ stderr `unknown skill tag "<tag>"`, stdout empty, exit 1.
- [ ] Ambiguous tag ⇒ stderr lists the candidate full tags (sorted), stdout empty,
      exit 1 (Level 3 step 5).
- [ ] Duplicate tag in argv resolves twice (repeats the path), not an error.
- [ ] Skills dir unresolvable + tags ⇒ exit 1, empty stdout, one-line fix on stderr.
- [ ] `--version` precedes tag mode even with a tag present (PRD §6.3).

### Code Quality & Scope Validation

- [ ] `main.go` has exactly the 4 additions (imports, `tags` field, positional
      capture, `<tag>` branch); version/path/list/default logic preserved verbatim.
- [ ] `main.go` imports == fmt, io, os, strings, internal/{discover,resolve,skillsdir,ui}.
- [ ] No `--file`/`--relative`/`--all`/`--search`/`check`/`--help` added (S2/M4/M5).
- [ ] `main_test.go` imports unchanged; reuses `writeSkillTree`/`unsetSkillsEnv`/`version`;
      no testify, no `t.Parallel()`.
- [ ] Nothing under `internal/` touched; no new files created; `go.mod`/`go.sum`/
      `PRD.md` untouched.

---

## Anti-Patterns to Avoid

- ❌ Don't print paths **incrementally** inside the resolve loop — buffer them and
  flush only after all resolve (§6.4 atomicity; a later failure must not leak a
  partial result to stdout).
- ❌ Don't `return 1` on the **first** bad tag — `continue` so every problem tag
  gets its own stderr line (PRD §6.4: "one error line per problem tag").
- ❌ Don't add a `skpp:` prefix to tag errors — print `err.Error()` verbatim, matching
  the `skillsdir.ErrNotFound` convention used by `--path`/`--list`.
- ❌ Don't print `res.Skill.SourceFile` or re-`Clean` the path — `Skill.Dir` is
  already absolute and clean; `--file`/`--relative` are S2.
- ❌ Don't special-case the `check` subcommand here — S1 captures it as a tag (exit
  1, same as before); M4.T10 adds real dispatch.
- ❌ Don't add `--file`/`--relative`/`--all`/`--search`/`check`/`--help` — out of
  scope (S2/M4/M5). Don't touch `internal/*`, `go.mod`, `go.sum`, `PRD.md`.
- ❌ Don't dedupe tags or error on a repeated tag — duplicates resolve twice by design.
- ❌ Don't reorder the `run` precedence tier — version → path → list → `<tag>` → default.
- ❌ Don't add testify or `t.Parallel()` — repo convention is plain `t.Errorf`/`t.Fatalf`, no Parallel.
- ❌ Don't skip `gofmt -w` — adding the longer `tags []string` line triggers struct-field alignment.

---

**Confidence Score: 10/10** for one-pass implementation success. The exact `main.go`
(complete file) and the precise `main_test.go` edits (1 replace + 1 append) are given
verbatim and have been **compiled, gofmt'd, vetted, unit-tested (main 24→36), and
smoke-tested against a real built binary** in `/tmp/skpp_t8_verify` (go1.25). The
critical §6.4 atomicity contract is verified at both the unit level
(`TestRunTagAtomicityUnknownPrintsNothing`, `TestRunTagAmbiguousListsCandidates`) and
the binary level (stdout EMPTY on any failure, exit 1). `go.mod`/`go.sum` are
byte-identical. An implementer who applies the two edits verbatim and runs the four
validation levels will ship this in one pass.
