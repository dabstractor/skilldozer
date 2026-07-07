# PRP — P1.M3.T2.S1: `skpp <tag> [...]` output + §6.4 error contract (atomicity)

> **Subtask:** directory `P1M3T2S1` == plan item `P1.M3.T8.S1` ("skpp <tag> [...]
> output + §6.4 error contract (atomicity)"). This is milestone **M3** (Tag
> resolution & path output). It is the subtask that finally makes `skpp <tag>`
> print paths. The plan id and directory name differ (renumber during planning);
> use the path given.
>
> **Scope:** MODIFY `main.go` (add positional `<tag>` capture to `parseArgs`, add a
> tag-resolution mode to `run`, add a `resolveTags` helper) and MODIFY `main_test.go`
> (update one test, add helpers + new tests). NO new files. NO new packages. NO
> `go.mod`/`go.sum` change.
>
> **DEPENDENCY (hard):** `discover.Skill` (P1.M2.T4.S2) and `discover.Index`
> (P1.M2.T5.S1) — neither is on disk at research time (contracts). Also consumes
> `resolve.Resolve` + `*UnknownError`/`*AmbiguousError` (P1.M3.T7.S1, dir `P1M3T1S1`,
> being implemented in parallel — treat as contract) and `skillsdir.Find` (LANDED).
> **Do not implement until `discover.Index` AND `discover.Skill` exist on disk**
> (`go build`/`go test` fail with "undefined: discover.Index/Skill" until then).
> M3 runs after M2, so they will exist at implementation time.
>
> **PARALLEL CONTEXT:** P1.M3.T7.S1 (resolve package, dir `P1M3T1S1`) is being
> implemented in parallel. Its PRP LOCKS `Resolve(tag, []discover.Skill) (Result,
> error)`, `Result{Skill, Match}`, and the pointer-receiver `*UnknownError{Tag}` /
> `*AmbiguousError{Tag, Candidates}`. This PRP CONSUMES that contract verbatim — see
> §"All Needed Context".

---

## Goal

**Feature Goal**: Make `skpp <tag> [<tag>...]` resolve one or more skill tags to
on-dill skill directory paths and print them with PRD §6.4 atomicity: resolve ALL
tags first; if any fails, print one error line per problem tag to stderr, print
NOTHING to stdout, exit 1; if all resolve, print each absolute skill directory on its
own line to stdout in input order, exit 0. This is PRD §6.1 row 1 + §6.4.

**Deliverable**: Modified `main.go` (parseArgs captures positionals as tags; `run`
dispatches a new tag-resolution mode via a `resolveTags` helper that wires
`skillsdir.Find` → `discover.Index` → `resolve.Resolve` with the §6.4 atomic-output
contract) and modified `main_test.go` (integration tests over a real skills tree,
capturing stdout/stderr separately).

**Success Definition**: `gofmt -l main.go main_test.go` silent; `go vet ./...` clean;
`go build ./...` and `go test ./...` pass; `go.mod`/`go.sum` **unchanged**. The PRD
§13 acceptance gates for `<tag>` resolution pass:
`test -d "$(./skpp example)"`, `case "$(./skpp example)" in /*)`, and the
unknown-tag contract (`out=$(./skpp nope 2>/dev/null); [ -z "$out" ] && [ "$rc" = "1" ]`).

---

## Why

- This subtask **makes the canonical contract real**: PRD §2.3 / §6.1 row 1 — "`skpp
  <tag>` prints exactly one absolute path (to stdout, trailing newline) for a
  resolved skill; unknown tag ⇒ nothing on stdout, error to stderr, exit 1." Until
  now main only did `--version`/`--path`. This wires the default resolution mode.
- It **establishes the §6.4 atomic-output pattern** (`go_architecture.md` calls §6.4
  the "MOST CRITICAL contract for `$(...)` safety"). PRD §6.4 exists so that
  `pi --skill "$(skpp badtag)"` fails loudly (empty stdout, exit 1) instead of
  passing a garbage/partial path list. Getting the buffer-all-then-flush discipline
  right here is the whole point.
- It **locks the extension point for S2** (modifiers `--file`/`-f`, `--relative`,
  `--all`/`-a`): the single print step in `resolveTags`'s success path is where S2
  swaps Dir→SourceFile / abs→rel. The resolve/buffer/error machinery stays untouched.
- It is **go.mod-neutral**: main adds only stdlib (`errors`, `strings`) + the internal
  `discover`/`resolve` packages (yaml.v3 is already a direct dep). No `go.mod` change.

---

## What

`main.go` gains:
1. Two new imports: `errors`, `strings`, plus the internal `discover` and `resolve`.
2. A `tags []string` field on `config` and positional-tag capture in `parseArgs`
   (tokens not starting with `-` → tags; unknown flags stay tolerated until M5).
3. A tag-resolution branch in `run` (after `--path`, before the no-args default):
   `if len(c.tags) > 0 { return resolveTags(c.tags, stdout, stderr) }`.
4. A new private helper `resolveTags(tags, stdout, stderr) int` implementing §6.4:
   `skillsdir.Find` → `discover.Index` → `resolve.Resolve` per tag (input order);
   on any failure, one stderr line per problem tag + nothing to stdout + exit 1;
   on all-good, buffer the absolute dirs and flush once + exit 0.

`main_test.go` gains integration tests over a real skills tree (helpers
`skillsTree`/`writeSkill`), the §6.4 atomicity cases (multi-tag in order, one-bad-
among-good prints nothing, ambiguous candidates), plus dir-not-found, version-
precedence-over-tags, absolute-path check, and parseArgs tag capture.

### Success Criteria

- [ ] `skpp example` (single good tag) prints the absolute skill dir + newline, exit 0,
      empty stderr.
- [ ] `skpp b a` (multiple good) prints b's dir then a's dir — INPUT order, not sorted.
- [ ] `skpp example nope` (one bad among good) prints NOTHING to stdout, one stderr
      line `skpp: unknown tag 'nope'`, exit 1.
- [ ] `skpp reddit` (ambiguous basename) prints NOTHING to stdout, stderr
      `skpp: ambiguous tag 'reddit', candidates: coding/reddit writing/reddit`, exit 1.
- [ ] `skpp nope1 nope2` (multiple bad) prints two stderr lines (input order), nothing
      to stdout, exit 1.
- [ ] Skills dir unresolvable → `skillsdir.ErrNotFound` (one-line fix) on stderr,
      nothing to stdout, exit 1.
- [ ] `--version` takes precedence over tags (PRD §6.3): version printed, tags ignored.
- [ ] Output is ABSOLUTE by default (§6.1/§13: `case "$(./skpp example)" in /*)`).
- [ ] stdout/stderr are captured SEPARATELY in tests (no leakage either way).
- [ ] `gofmt -l main.go main_test.go` silent; `go vet ./...` clean; `go build ./...`
      + `go test ./...` pass; `go.mod`/`go.sum` unchanged (`git diff --quiet`).
- [ ] All previously-passing main_test.go tests STILL pass (additive change).

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT edits to `main.go` and the EXACT new tests are given verbatim in the
Implementation Blueprint (copy-paste clean, gofmt-clean, compiles as-is once
`discover.Index`/`discover.Skill` exist). Every consumed contract — `skillsdir.Find`
+ `ErrNotFound` (LANDED, read in full), `discover.Skill`/`Index` (locked in
go_architecture.md), `resolve.Resolve` + typed errors (locked in P1M3T1S1 PRP) — is
re-stated below. The §6.4 atomicity discipline, the error-format divergence from
resolve's `.Error()` (§5 of verified_facts), the input-order requirement, the buffer-
then-flush guarantee, the S2 extension point, and the test strategy are all
documented with reasoning. An implementer who knows Go but nothing about this repo
can complete this in one pass (provided `discover.Index`/`discover.Skill` exist)._

### Documentation & References

```yaml
# MUST READ — this subtask's own decisions (every load-bearing choice)
- file: plan/001_fcde63e5bb60/P1M3T2S1/research/verified_facts.md
  why: "Locks: (0) scope (modify main.go/main_test.go only; no new files). (1) the
        4 consumed contracts + their on-disk status + the Task-0 gate. (2) the EXACT
        contract shapes (Find/ErrNotFound, Skill/Index, Resolve/errors). (3) the §6.4
        atomicity discipline (two-pass + buffer). (4) output format (Dir, absolute,
        input order, one/line). (5) error message format — DIVERGES from resolve's
        .Error() (single quotes, 'skpp:' prefix, space-joined candidates). (6) the
        parseArgs/run integration (tag capture, precedence, resolveTags helper).
        (7) all existing tests still pass (traced). (8) test strategy (integration
        via run, real skills tree fixtures). (9) go.mod-neutral. (10) S2 extension
        point. (11) scope boundary."
  critical: "The §6.4 error wording is an ITEM CONTRACT that OVERRIDES resolve's
             .Error(). Do NOT print resolve's error text; extract .Tag/.Candidates
             and format as 'skpp: unknown tag \\'<tag>\\'' / 'skpp: ambiguous tag
             \\'<tag>\\', candidates: <space-joined>'. Hard gate: discover.Index/
             Skill must exist first."

# CONTRACT — resolve.Resolve + typed errors (P1.M3.T7.S1, dir P1M3T1S1)
- file: plan/001_fcde63e5bb60/P1M3T1S1/PRP.md
  why: "Locks Resolve(tag, []discover.Skill) (Result, error); Result{Skill, Match};
        *UnknownError{Tag} + *AmbiguousError{Tag, Candidates} (POINTER receivers, so
        *T satisfies error; use errors.As to extract). Candidates already SORTED
        (sortedRelTags). Its 'DOWNSTREAM CONSUMERS' section states P1.M3.T8.S1 (this
        task) calls discover.Index() once then resolve.Resolve per tag, and on
        *AmbiguousError/*UnknownError prints one stderr line per problem tag, prints
        NOTHING to stdout, exits 1, using errors.As(err,&ae) to read ae.Candidates."
  section: "Goal", "All Needed Context > Known Gotchas", "Integration Points > DOWNSTREAM CONSUMERS"
  critical: "errors.As is the extraction path (NOT type-assertion). The resolve
             package's OWN .Error() text is NOT the §6.4 wording — main reformats."

# CONTRACT — discover.Skill + Index (P1.M2.T4.S2 / P1.M2.T5.S1)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "'Core types > internal/discover' locks type Skill struct { Dir (ABSOLUTE),
        RelTag, Name, Description, Keywords, Category, Aliases, HasFM, SourceFile }
        and func Index(absSkillsDir string) ([]Skill, error) (sorted by RelTag,
        lenient). 'Output discipline (§6.4)' states the buffer-all-then-flush rule.
        'Data flow' shows main: skillsdir.Find → discover.Index → resolve.Resolve
        per tag → print paths."
  section: "Core types > internal/discover", "Output discipline (§6.4)", "Data flow"
  critical: "Skill.Dir is ABSOLUTE — print it directly (do NOT filepath.Abs it; that
             masks an Index bug and the §13 'case ... in /*)' gate depends on Index
             producing absolute Dir). Index is lenient: empty dir → ([], nil)."

# CONTRACT — skillsdir.Find + ErrNotFound (LANDED — read the source)
- file: internal/skillsdir/skillsdir.go
  why: "Find() (dir, src Source, err) returns ABSOLUTE dir; on all-miss returns
        ('', 0, ErrNotFound). ErrNotFound.Error() is the one-line user fix. The
        existing --path branch already does `fmt.Fprintln(stderr, err)` — mirror it
        verbatim in resolveTags's Find-failure path."
  pattern: "fmt.Fprintln(stderr, err) for Find failure; print err verbatim."

# CONTRACT — the §6.1/§6.3/§6.4 CLI behavior this implements
- file: PRD.md
  why: "§6.1 row 1: 'skpp <tag> [<tag>...]' → one ABSOLUTE path/line, input order;
        exit 0 if all resolve, 1 if any fail (and NOTHING printed). §6.3: --version
        takes precedence over everything. §6.4: any unresolved/ambiguous ⇒ one error
        line per problem tag to stderr, NOTHING to stdout, exit 1; ambiguous ⇒ stderr
        lists candidate full tags. §2.3/§17: never print to stdout on failure. READ-ONLY."
  section: "6.1 Commands/flags (row 1)", "6.3 Default behavior", "6.4 Error semantics",
           "2. Hard constraints (#3)", "17. Constraints & guardrails"

# REFERENCE — the current main.go (the file being modified)
- file: main.go
  why: "The existing parseArgs (switch a { --version/-v, --path/-p, default: tolerated }),
        config struct, and run() (version → path → return 1) are the starting point.
        The version var + ldflags comment and the run(args,stdout,stderr)int testable
        signature MUST be preserved. The --path branch is the template for Find-error
        handling (fmt.Fprintln(stderr, err); return 1)."
  pattern: "testable run(args, stdout, stderr) int; precedence tier version>path; Find
            error printed verbatim to stderr."

# REFERENCE — the test convention (mirror it)
- file: main_test.go
  why: "Established conventions to mirror: inject *bytes.Buffer for stdout/stderr
        (separate capture), t.Setenv for SKPP_SKILLS_DIR, t.Chdir(t.TempDir()) for
        dir-not-found, plain t.Errorf/t.Fatalf, NO testify, NO t.Parallel. The
        existing TestRunPathFailureErrNotFound is the template for the Find-failure
        test (assert empty stdout + ErrNotFound substrings on stderr)."
  pattern: "bytes.Buffer for stdout+stderr; t.Setenv/t.Chdir; plain assertions."

- file: internal/discover/discover_test.go
  why: "The writeSkill helper idiom (write SKILL.md content into a temp dir) — adapt
        to a skills TREE (root/<relTag>/SKILL.md) so discover.Index walks it."
  pattern: "os.MkdirAll + os.WriteFile for on-disk fixtures; t.Helper()."

# URLS — the load-bearing stdlib surface
- url: https://pkg.go.dev/errors#As
  why: "errors.As(err, &target) extracts *resolve.AmbiguousError/*UnknownError from
        the error resolve returns (pointer receivers make *T satisfy error). Do NOT
        type-assert; errors.As is the contract resolve's TestResolveErrorsAs proves."
- url: https://pkg.go.dev/strings#Join
  why: "strings.Join(ae.Candidates, \" \") space-joins the candidate tags for the
        §6.4 ambiguous line (NOT comma — item CONTRACT)."
- url: https://pkg.go.dev/strings#Builder
  why: "strings.Builder buffers the success-path stdout so nothing reaches stdout
        unless the whole invocation is known-good (§6.4 'never partially print')."
```

### Current Codebase tree (M1 landed, M2 partial, M3.T7 in flight; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/discover/discover.go        # M2.T4.S1: Frontmatter + ParseFrontmatter (LANDED). Skill/Index NOT yet (contracts)
internal/discover/discover_test.go   # M2.T4.S1 tests
internal/skillsdir/skillsdir.go      # M1.T2: Source + Find + per-rule helpers (LANDED)
internal/skillsdir/skillsdir_test.go # M1.T2 tests
internal/resolve/resolve.go          # M3.T7.S1 (P1M3T1S1): Resolve + errors (IN FLIGHT / contract)
internal/resolve/resolve_test.go     # M3.T7.S1 tests (IN FLIGHT / contract)
main.go                              # M1.T3: --version/--path only (MODIFY HERE)
main_test.go                         # M1.T3 tests (MODIFY HERE)

$ ls -A
.git/ .gitignore LICENSE PRD.md go.mod go.sum internal/ main.go main_test.go plan/ .pi-subagents/
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT)
# NOTE: discover.Skill + discover.Index are CONTRACTS (M2.T4.S2/T5.S1) — not on disk yet.
#       This subtask cannot compile until they land (Task-0 gate). resolve is in flight.
```

### Desired Codebase tree with files modified

```bash
skpp/
├── ... (go.mod, go.sum UNCHANGED; .gitignore, LICENSE, PRD.md, internal/* UNCHANGED)
├── main.go        # MODIFY: +errors/strings/discover/resolve imports; +tags field;
│                  #        +positional capture in parseArgs; +tag-resolution branch
│                  #        in run; +resolveTags helper (§6.4 atomicity)
└── main_test.go   # MODIFY: +os import; update TestParseArgsUnknownTolerated;
                   #        +skillsTree/writeSkill helpers; +tag-resolution tests
```

| File (modified) | What changes | New imports |
|---|---|---|
| `main.go` | `config.tags`; parseArgs captures positionals; `run` tag branch; `resolveTags` helper | `errors`, `strings`, `discover`, `resolve` |
| `main_test.go` | update 1 test; +2 helpers; +8 tests | `os` |

**Two files modified. NO new files. NO new packages. NO `go.mod`/`go.sum` change.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — HARD DEPENDENCY on discover.Index + discover.Skill (M2.T5.S1/T4.S2).
// Neither is on disk at research time. main.go will import discover and call
// discover.Index(dir) + reference r.Skill.Dir. Until they exist, `go build`/`go test`
// fail with "undefined: discover.Index"/"discover.Skill". Run the Task-0 gate first.
// Do NOT define Skill/Index here (that's discover's deliverable) — only CONSUME.
//   RIGHT: import discover; skills, err := discover.Index(dir); ... r.Skill.Dir
//   WRONG: defining a local Skill type or a local index walker.

// GOTCHA #2 — §6.4 atomicity is THE contract (item: "MOST CRITICAL"). Resolve EVERY
// tag FIRST, collect per-tag errors. If ANY failed: print one error line per PROBLEM
// tag to stderr, print NOTHING to stdout, exit 1. NEVER print a partial result. Only
// when ALL resolve do you print — and you BUFFER into a strings.Builder first, then a
// single fmt.Fprint(stdout, buf). This guarantees `pi --skill "$(skpp badtag)"` gets
// an empty string + exit 1 (loud failure) instead of a garbage/partial path list.
//   RIGHT: resolve all → if anyErr { stderr per problem tag; return 1 } → buffer all → flush
//   WRONG: print each dir as you resolve it (a later bad tag leaves partial stdout)

// GOTCHA #3 — Error wording OVERRIDES resolve's .Error(). The item CONTRACT specifies
// the §6.4 stderr format, which is DIFFERENT from resolve's package .Error() text:
//   resolve UnknownError.Error()  = `unknown skill tag "foo"`     (double quotes, "skill")
//   resolve AmbiguousError.Error()= `ambiguous skill tag "x" matches: a, b` (comma-joined)
//   ITEM CONTRACT (use THIS):      `skpp: unknown tag 'foo'`      (single quotes, "skpp:")
//                                  `skpp: ambiguous tag 'x', candidates: a b` (SPACE-joined)
// Extract .Tag and .Candidates via errors.As and format yourself. Do NOT fmt.Fprintln
// the raw error. Candidates are already SORTED by resolve; join with a SPACE.
//   RIGHT: fmt.Fprintf(stderr, "skpp: unknown tag '%s'\n", ue.Tag)
//   WRONG: fmt.Fprintln(stderr, err)   // prints resolve's double-quoted text

// GOTCHA #4 — Use errors.As, NOT type-assertion. Resolve returns *UnknownError /
// *AmbiguousError (pointer receivers). Extract with:
//   var ae *resolve.AmbiguousError; if errors.As(err, &ae) { ... }
//   var ue *resolve.UnknownError;  if errors.As(err, &ue) { ... }
// Type-assertion (err.(*resolve.AmbiguousError)) works too but errors.As is the
// contract resolve's TestResolveErrorsAs proves and survives error wrapping.

// GOTCHA #5 — Output is INPUT order, NOT sorted. discover.Index returns skills sorted
// by RelTag, but you ITERATE THE TAGS the user typed, so output follows TAG order.
// `skpp b a` prints b's dir then a's dir. Do NOT sort the output by tag or by RelTag.
// (Sorting by tag would break the user's expected ordering and the input-order test.)

// GOTCHA #6 — Print r.Skill.Dir DIRECTLY (it is ABSOLUTE by contract). Do NOT wrap it
// in filepath.Abs or filepath.Clean — that would (a) mask an Index bug if Dir were
// somehow not absolute, and (b) the §13 gate `case "$(./skpp example)" in /*)` needs
// the FIRST char to be '/'. Index builds Dir from the absolute skills dir (Find
// returns absolute), so Dir is absolute. Trust the contract; print verbatim.
//   RIGHT: fmt.Fprintln(&paths, r.Skill.Dir)
//   WRONG: fmt.Fprintln(&paths, filepath.Abs(r.Skill.Dir))   // masks bugs, adds import

// GOTCHA #7 — Default unit is the DIRECTORY, not SKILL.md. PRD §3 decision; §6.1 row 1.
// Print r.Skill.Dir (the directory). --file/-f (S2) will swap to r.Skill.SourceFile —
// do NOT add that flag here (scope theft; it is P1.M3.T2.S2). The print step is the
// SINGLE location S2 will parameterize.

// GOTCHA #8 — Index-of-empty returns ([], nil), NOT an error. Index is lenient (Find
// already confirmed the dir exists; missing frontmatter is not an error). So an empty
// skills dir yields an empty index, and every queried tag is Unknown. To avoid coupling
// a test to this, the multi-bad test writes ONE unrelated real skill first.

// GOTCHA #9 — parseArgs must KEEP unknown flags tolerated (exit-2 is M5's job). Only
// capture tokens NOT starting with '-' as tags. A token starting with '-' that is not
// --version/-v/--path/-p is an unknown flag → tolerated no-op for now (M5 → exit 2).
// Do NOT turn unknown flags into errors yet (that breaks TestRunDefaultUnknownFlag and
// steals M5's deliverable).
//   RIGHT: if strings.HasPrefix(a, "-") { /* tolerated */ } else { c.tags = append(...) }
//   WRONG: treating unknown flags as errors or as tags

// GOTCHA #10 — --version precedence is preserved (checked before tags in run). And
// --path is checked before tags too (so --path+tags → path wins; mode-exclusivity
// exit-2 is M5). Do NOT reorder. The tag branch goes AFTER --path, BEFORE the no-args
// default `return 1`.

// GOTCHA #11 — go.mod/go.sum UNCHANGED. main adds errors/strings (stdlib) + the
// INTERNAL discover/resolve packages. yaml.v3 is already direct (M2.T4.S1 flipped it).
// `go mod tidy` is a no-op. Verify: `git diff --quiet go.mod go.sum`. If it changes,
// you imported something external you should not have.

// GOTCHA #12 — Tests are INTEGRATION (via run), not unit. Build a REAL skills tree
// (root/<relTag>/SKILL.md), set SKPP_SKILLS_DIR (rule 1 wins), call run(args, &out,
// &errOut), assert on the *bytes.Buffer. This exercises Find→Index→Resolve→print.
// Unit tests of Resolve/Index live in their packages. Capture stdout AND stderr in
// SEPARATE buffers (the §6.4 contract is specifically about stdout staying empty on
// failure while stderr carries the error). NO testify; NO t.Parallel() (repo convention).
```

---

## Implementation Blueprint

### The consumed contracts (LOCKED — do not redefine)

```go
// skillsdir (LANDED)
func Find() (dir string, src Source, err error)             // absolute dir; or ErrNotFound
var ErrNotFound error                                        // one-line fix message

// discover (CONTRACT — M2)
type Skill struct {
    Dir         string   // ABSOLUTE skill dir  ← print this
    RelTag      string
    Name        string
    Aliases     []string
    // ...Description, Keywords, Category, HasFM, SourceFile
}
func Index(absSkillsDir string) ([]Skill, error)            // sorted by RelTag; lenient

// resolve (CONTRACT — M3.T7, dir P1M3T1S1)
type Result struct { Skill discover.Skill; Match MatchKind }
func Resolve(tag string, skills []discover.Skill) (Result, error)
type UnknownError   struct{ Tag string }                    // *T implements error
type AmbiguousError struct{ Tag string; Candidates []string } // *T implements error; Candidates sorted
```

### Edit 1 — `main.go` imports (add errors, strings, discover, resolve)

Replace the import block:

```go
// OLD
import (
	"fmt"
	"io"
	"os"

	"github.com/dabstractor/skpp/internal/skillsdir"
)
```
```go
// NEW
import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
	"github.com/dabstractor/skpp/internal/skillsdir"
)
```

### Edit 2 — `main.go` config struct (add `tags`)

```go
// OLD
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	// Future (M2-M5), do NOT add yet:
	//   list, all bool; search string; check bool; file, noColor, relative, help bool; tags []string
}
```
```go
// NEW
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	tags    []string // positional <tag> args (PRD §6.1 row 1 — the default resolution
	// mode). Captured in parseArgs; resolved in run()'s resolveTags branch; printed
	// per the §6.4 atomic-output contract.
	// Future (M2-M5), do NOT add yet:
	//   list, all bool; search string; check bool; file, noColor, relative, help bool
}
```

### Edit 3 — `main.go` parseArgs (capture positionals as tags; keep flags tolerated)

```go
// OLD
		default:
			// Unknown flag / subcommand / positional: tolerated for now.
			// P1.M5.T11 implements: unknown flag -> exit 2 (§6.2),
			// `check` subcommand dispatch, and <tag> positional capture.
		}
```
```go
// NEW
		default:
			// A leading '-' marks a FLAG. Known flags are matched above; an unknown
			// flag (e.g. --frobnicate, -x) is tolerated here as a no-op — P1.M5.T11
			// turns unknown flags into exit 2 (PRD §6.2) and adds subcommand dispatch
			// (e.g. `check`).
			//
			// Anything NOT starting with '-' is a positional <tag> (PRD §6.1 row 1 —
			// the default resolution mode). It is captured here and resolved in run().
			if strings.HasPrefix(a, "-") {
				// unknown flag: tolerated (no-op) for now; P1.M5.T11 -> exit 2
			} else {
				c.tags = append(c.tags, a)
			}
		}
```

### Edit 4 — `main.go` run (add tag-resolution branch) + new resolveTags helper

Insert the tag-resolution branch into `run` (between the `--path` block and the
no-args `return 1`), and add the `resolveTags` helper after `run`.

```go
// OLD (the tail of run, after the --path block)
	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}
```
```go
// NEW (add the tag branch BEFORE return 1; then resolveTags after run)
	// Tag-resolution mode (PRD §6.1 row 1): positional <tag> args resolve to skill
	// directory paths. The §6.4 atomic-output contract lives inside resolveTags.
	if len(c.tags) > 0 {
		return resolveTags(c.tags, stdout, stderr)
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}

// resolveTags implements the PRD §6.1-row-1 tag-resolution mode with the §6.4
// atomic-output contract — the MOST CRITICAL contract for `$(...)` safety.
//
// It locates the skills dir (skillsdir.Find), builds the index (discover.Index),
// and resolves every tag (resolve.Resolve), collecting per-tag outcomes in INPUT
// order. Only if ALL tags resolve does it print one absolute skill.Dir per line to
// stdout (buffered, so nothing reaches stdout unless the whole invocation is
// known-good). On ANY failure it prints one error line per problem tag to stderr,
// prints NOTHING to stdout, and returns exit 1.
//
// §6.4 guarantees `pi --skill "$(skpp badtag)"` fails loudly (empty stdout + exit 1)
// rather than passing a garbage or partial path list.
//
// EXTENSION POINT (P1.M3.T2.S2 — modifiers): the success-path print step currently
// emits skill.Dir (absolute directory). --file/-f (print skill.SourceFile) and
// --relative (print relative to the skills dir) layer onto THIS step without
// touching the resolve/buffer/error machinery above.
func resolveTags(tags []string, stdout, stderr io.Writer) int {
	dir, _, err := skillsdir.Find()
	if err != nil {
		// skillsdir.ErrNotFound: its message is the one-line user fix (§8.4/§6.4).
		// Printed verbatim to stderr (same as the --path branch); stdout stays empty.
		fmt.Fprintln(stderr, err)
		return 1
	}

	skills, err := discover.Index(dir)
	if err != nil {
		// Index is lenient (missing frontmatter is not an error; Find already
		// confirmed the dir exists), so this path is rare. Honor §6.4 regardless:
		// print the error to stderr, print NOTHING to stdout, exit 1.
		fmt.Fprintln(stderr, err)
		return 1
	}

	// Pass 1 — resolve EVERY tag, keeping INPUT order so both the success output and
	// the per-tag error lines follow the order the user typed the tags.
	type resolved struct {
		tag string
		dir string // absolute skill dir; empty when err != nil
		err error
	}
	out := make([]resolved, len(tags))
	for i, tag := range tags {
		r, err := resolve.Resolve(tag, skills)
		if err != nil {
			out[i] = resolved{tag: tag, err: err}
			continue
		}
		out[i] = resolved{tag: tag, dir: r.Skill.Dir}
	}

	// §6.4 atomicity — Pass 2a: if ANY tag failed, emit one error line per PROBLEM
	// tag to stderr (in input order), print NOTHING to stdout, exit 1.
	hasErr := false
	for _, r := range out {
		if r.err != nil {
			hasErr = true
			break
		}
	}
	if hasErr {
		for _, r := range out {
			if r.err == nil {
				continue // only problem tags get a stderr line
			}
			var ae *resolve.AmbiguousError
			var ue *resolve.UnknownError
			switch {
			case errors.As(r.err, &ae):
				// §6.4: list the candidate FULL tags (space-joined) so the user can
				// disambiguate. Candidates are already sorted by resolve. NOTE: this
				// wording is the item CONTRACT and DIFFERS from ae.Error() (which is
				// comma-joined, double-quoted).
				fmt.Fprintf(stderr, "skpp: ambiguous tag '%s', candidates: %s\n",
					ae.Tag, strings.Join(ae.Candidates, " "))
			case errors.As(r.err, &ue):
				fmt.Fprintf(stderr, "skpp: unknown tag '%s'\n", ue.Tag)
			default:
				// Defensive: resolve only ever returns *UnknownError or
				// *AmbiguousError, so this is unreachable. Keep the §6.4 invariant
				// (nothing to stdout) regardless.
				fmt.Fprintf(stderr, "skpp: %s\n", r.err)
			}
		}
		return 1
	}

	// §6.4 atomicity — Pass 2b: all tags resolved. Buffer the absolute dirs (one per
	// line, input order) into a strings.Builder, then a SINGLE write so a mid-loop
	// write failure can never leave a partial stdout ("never partially print").
	var paths strings.Builder
	for _, r := range out {
		fmt.Fprintln(&paths, r.dir)
	}
	fmt.Fprint(stdout, paths.String())
	return 0
}
```

### Edit 5 — `main_test.go` imports (add `os`)

```go
// OLD
import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)
```
```go
// NEW
import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)
```

### Edit 6 — `main_test.go` update `TestParseArgsUnknownTolerated` (positionals are now tags)

```go
// OLD
// Unknown tokens are tolerated (no-op) for now; exit-2 lands in P1.M5.T11.
func TestParseArgsUnknownTolerated(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "sometag", "check"})
	if c.version || c.path {
		t.Errorf("parseArgs(unknown): version=%v path=%v; want both false (tolerated)", c.version, c.path)
	}
}
```
```go
// NEW
// Unknown FLAGS (leading '-') are tolerated (no-op) for now; exit-2 lands in
// P1.M5.T11. Positional tokens are NOT flags — they are captured as <tag> args now
// (see TestParseArgsCapturesPositionalTags). Input here is flags-only.
func TestParseArgsUnknownTolerated(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "-x"})
	if c.version || c.path {
		t.Errorf("parseArgs(unknown flags): version=%v path=%v; want both false", c.version, c.path)
	}
	if len(c.tags) != 0 {
		t.Errorf("parseArgs(unknown flags): tags=%v; want empty (flags are not tags)", c.tags)
	}
}
```

### Edit 7 — `main_test.go` ADD helpers + new tests (append at end of file)

Append the following block verbatim at the end of `main_test.go`:

```go

// ---------------------------------------------------------------------------
// Tag resolution (PRD §6.1 row 1 + §6.4 atomicity) — P1.M3.T2.S1
//
// These are INTEGRATION tests: they build a real on-disk skills tree, point
// SKPP_SKILLS_DIR at it (so skillsdir.Find() rule 1 wins deterministically), and
// call run(), exercising Find -> discover.Index -> resolve.Resolve -> print.
// stdout and stderr are captured in SEPARATE buffers — the §6.4 contract is
// specifically that stdout stays EMPTY on failure while stderr carries the error.
// ---------------------------------------------------------------------------

// skillsTree creates an empty temp dir, points SKPP_SKILLS_DIR at it (rule 1 wins),
// and returns the dir. Callers build the skills/ sub-tree with writeSkill.
func skillsTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("SKPP_SKILLS_DIR", root)
	return root
}

// writeSkill creates root/<relTag>/SKILL.md with frontmatter+body, mirroring the
// on-disk layout discover.Index() walks. relTag uses '/' separators on every
// platform (normalized internally); filepath.FromSlash makes the OS-native dir.
func writeSkill(t *testing.T, root, relTag, frontmatter, body string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(relTag))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	content := frontmatter
	if body != "" {
		content += "\n" + body
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s/SKILL.md): %v", dir, err)
	}
}

// --- parseArgs: positional tag capture ---

// Positional tokens (no leading '-') are captured as tags in input order; flags are
// not. (PRD §6.1 row 1.)
func TestParseArgsCapturesPositionalTags(t *testing.T) {
	c := parseArgs([]string{"example", "writing/reddit", "foo"})
	if len(c.tags) != 3 || c.tags[0] != "example" || c.tags[1] != "writing/reddit" || c.tags[2] != "foo" {
		t.Errorf("parseArgs(tags): tags=%v; want [example writing/reddit foo] in order", c.tags)
	}
	if c.version || c.path {
		t.Errorf("parseArgs(tags): version=%v path=%v; want false", c.version, c.path)
	}
}

// Flags and positional tags may be interleaved (PRD §6: flags in any order).
func TestParseArgsFlagsAndTagsMixed(t *testing.T) {
	c := parseArgs([]string{"--version", "example", "-p", "other"})
	if !c.version || !c.path {
		t.Errorf("version=%v path=%v; want true,true", c.version, c.path)
	}
	if len(c.tags) != 2 || c.tags[0] != "example" || c.tags[1] != "other" {
		t.Errorf("tags=%v; want [example other]", c.tags)
	}
}

// --- run: tag resolution (§6.1 row 1 + §6.4) ---

// Single good tag resolves to its ABSOLUTE skill dir, exit 0, empty stderr.
func TestRunTagSingleResolvesAbsolute(t *testing.T) {
	root := skillsTree(t)
	writeSkill(t, root, "example",
		"---\nname: example\ndescription: An example skill\n---\n", "body")
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example): code=%d; want 0", code)
	}
	want := filepath.Join(root, "example") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(example) stdout=%q; want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(example) stderr=%q; want empty", errOut.String())
	}
	// §13 absolute-path contract: output must be an absolute path.
	if got := strings.TrimRight(out.String(), "\n"); !filepath.IsAbs(got) {
		t.Errorf("run(example) stdout=%q; want absolute path (§6.1/§13)", got)
	}
}

// Multiple good tags: N lines in INPUT order (not sorted by tag), exit 0.
func TestRunTagMultipleInInputOrder(t *testing.T) {
	root := skillsTree(t)
	writeSkill(t, root, "alpha", "---\nname: alpha\n---\n", "")
	writeSkill(t, root, "beta", "---\nname: beta\n---\n", "")
	var out, errOut bytes.Buffer
	// Deliberately out of sorted order to prove INPUT order, not sorted output.
	code := run([]string{"beta", "alpha"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(beta alpha): code=%d; want 0", code)
	}
	want := filepath.Join(root, "beta") + "\n" + filepath.Join(root, "alpha") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(beta alpha) stdout=%q; want %q (input order, not sorted)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty", errOut.String())
	}
}

// §6.4 atomicity: one bad tag among good -> NOTHING on stdout, one error line per
// problem tag on stderr, exit 1. (Proves a partial result is never printed.)
func TestRunTagOneBadAmongGoodPrintsNothingToStdout(t *testing.T) {
	root := skillsTree(t)
	writeSkill(t, root, "example", "---\nname: example\n---\n", "")
	var out, errOut bytes.Buffer
	code := run([]string{"example", "nope"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(example nope): code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (§6.4: nothing on stdout on any failure)", out.String())
	}
	want := "skpp: unknown tag 'nope'\n"
	if got := errOut.String(); got != want {
		t.Errorf("stderr=%q; want %q", got, want)
	}
}

// Two bad tags -> two error lines on stderr (input order), nothing on stdout, exit 1.
// (One real skill is written so the index is non-empty; the queried tags still miss.)
func TestRunTagMultipleBadAllReported(t *testing.T) {
	root := skillsTree(t)
	writeSkill(t, root, "real", "---\nname: real\n---\n", "") // unrelated real skill
	var out, errOut bytes.Buffer
	code := run([]string{"nope1", "nope2"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(nope1 nope2): code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	want := "skpp: unknown tag 'nope1'\nskpp: unknown tag 'nope2'\n"
	if got := errOut.String(); got != want {
		t.Errorf("stderr=%q; want %q (one line per problem tag, input order)", got, want)
	}
}

// Ambiguous tag: candidates (sorted, SPACE-joined) on stderr, nothing on stdout, exit 1.
func TestRunTagAmbiguousPrintsCandidates(t *testing.T) {
	root := skillsTree(t)
	writeSkill(t, root, "writing/reddit", "---\nname: reddit-writer\n---\n", "")
	writeSkill(t, root, "coding/reddit", "---\nname: reddit-coder\n---\n", "")
	var out, errOut bytes.Buffer
	code := run([]string{"reddit"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(reddit): code=%d; want 1 (ambiguous)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	// Candidates are sorted by resolve: coding/reddit before writing/reddit; SPACE-joined.
	want := "skpp: ambiguous tag 'reddit', candidates: coding/reddit writing/reddit\n"
	if got := errOut.String(); got != want {
		t.Errorf("stderr=%q; want %q", got, want)
	}
}

// Skills dir unresolvable: Find() fails -> ErrNotFound (one-line fix) on stderr,
// nothing on stdout, exit 1. (Mirrors TestRunPathFailureErrNotFound.)
func TestRunTagSkillsDirNotFound(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // empty tree -> all §8 rules miss -> Find returns ErrNotFound
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(example) [no dir]: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	msg := errOut.String()
	for _, want := range []string{"SKPP_SKILLS_DIR", "cd", "reinstall"} {
		if !strings.Contains(msg, want) {
			t.Errorf("stderr=%q; missing substring %q (ErrNotFound fix hint)", msg, want)
		}
	}
}

// --version takes precedence over tags (PRD §6.3): version printed, tags ignored.
func TestRunVersionPrecedenceOverTags(t *testing.T) {
	root := skillsTree(t)
	writeSkill(t, root, "example", "---\nname: example\n---\n", "")
	var out, errOut bytes.Buffer
	code := run([]string{"--version", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--version example): code=%d; want 0 (version precedence)", code)
	}
	want := "skpp " + version + "\n"
	if got := out.String(); got != want {
		t.Errorf("stdout=%q; want %q (version, not tag resolution)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty", errOut.String())
	}
}
```

> **Copy-paste correctness:** every edit is against the EXACT current text of main.go
> / main_test.go (read in full at research time). The new code is gofmt-clean (tabs,
> grouped imports stdlib-then-internal) and compiles as-is once `discover.Index`/
> `discover.Skill` exist. The `resolved` struct is local to `resolveTags`. No new
> external imports. Apply edits with the `edit` tool (each OLD block is unique in its
> file); append Edit 7 at EOF.

### Implementation Patterns & Key Details

```go
// PATTERN: §6.4 two-pass atomicity (the heart of this task).
//   Pass 1: resolve ALL tags into []resolved{tag,dir,err} (input order).
//   Pass 2a: if ANY err -> stderr one line per PROBLEM tag (skip good), NOTHING to
//            stdout, return 1.
//   Pass 2b: all good -> buffer dirs into strings.Builder, ONE fmt.Fprint, return 0.
// WHY: `pi --skill "$(skpp ...)"` captures stdout. A partial print would give pi a
//      garbage path list. Resolving-all-first + buffer-then-flush guarantees empty
//      stdout on any failure (PRD §6.4 / §2.3 / §17). The buffer additionally means
//      a mid-loop write error can never leave partial stdout.

// PATTERN: errors.As for typed-error extraction (NOT type-assertion).
//   var ae *resolve.AmbiguousError
//   if errors.As(r.err, &ae) { fmt.Fprintf(stderr, "skpp: ambiguous tag '%s', ...", ae.Tag, ...) }
// WHY: resolve returns *UnknownError/*AmbiguousError (pointer receivers). errors.As
//      is the contract resolve's TestResolveErrorsAs proves and survives wrapping.

// PATTERN: §6.4 wording OVERRIDES resolve's .Error().
//   fmt.Fprintf(stderr, "skpp: unknown tag '%s'\n", ue.Tag)               // NOT ue.Error()
//   fmt.Fprintf(stderr, "skpp: ambiguous tag '%s', candidates: %s\n",
//       ae.Tag, strings.Join(ae.Candidates, " "))                         // NOT ae.Error()
// WHY: the item CONTRACT specifies single-quoted tags, 'skpp:' prefix, and
//      SPACE-joined candidates. resolve's .Error() uses double quotes and comma-join.
//      main OWNS the §6.4 stderr wording (resolve is prefix-free, like ErrNotFound).

// PATTERN: input-order iteration (NOT sorted output).
//   for i, tag := range tags { out[i] = resolve.Resolve(tag, skills) ... }
//   for _, r := range out { fmt.Fprintln(&paths, r.dir) }
// WHY: §6.1 row 1 says "in input order". discover.Index sorts by RelTag internally,
//      but we iterate the USER's tags, so output follows tag order (skpp b a -> b,a).
```

### Integration Points

```yaml
main.go PUBLIC SURFACE (unchanged): run(args, stdout, stderr) int is still the
  testable entry; main() still calls os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)).
  The version var + ldflags comment are untouched. parseArgs/run/config signatures
  are unchanged (config gains a field; parseArgs gains the strings import).

PRECEDENCE (run, PRD §6.3): --version (and --help in M5) > --path > tags > no-args(1).
  - --version + tags  -> version wins (TestRunVersionPrecedenceOverTags).
  - --path + tags     -> path wins (mode-exclusivity exit-2 is M5; not forbidden now).
  - no args           -> tags empty -> return 1 (unchanged no-args default).

go.mod / go.sum (NO change — verified_facts §9):
  - main adds errors/strings (stdlib) + internal discover/resolve. yaml.v3 is already
    direct (M2.T4.S1). `go mod tidy` is a no-op. VERIFY: git diff --quiet go.mod go.sum.

DOWNSTREAM CONSUMERS / EXTENSION POINTS:
  - P1.M3.T2.S2 (modifiers): --file/-f swaps the print step from r.Skill.Dir to
    r.Skill.SourceFile; --relative swaps absolute->relative; --all/-a iterates all
    skills instead of tags. ALL layer onto the SINGLE print step in resolveTags's
    success path (the `fmt.Fprintln(&paths, r.dir)` loop). The resolve/buffer/error
    machinery is untouched. This task makes that step a clean, single location.
  - P1.M5.T11 (CLI surface): turns unknown flags into exit 2, adds --help text +
    no-args usage, and mode-exclusivity exit-2 (tag + --list/--search/--all). The
    parseArgs default's "tolerated" branch is where M5 hooks exit-2.

NO CHANGES TO:
  - go.mod / go.sum (dependency-neutral; verify with git diff)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned)
  - internal/discover/* (M2-owned; only IMPORT)
  - internal/resolve/* (M3.T7-owned; only IMPORT)
  - internal/skillsdir/* (M1-owned; only IMPORT)
  - any other package or file (ui is M2.T6; skills/ is P1.M6.T12; install.sh/README
    are M6; the modifiers are S2)
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Task-0 GATE: confirm the hard dependencies exist on disk.
grep -q 'type Skill struct' internal/discover/discover.go \
  || { echo "FAIL GATE: discover.Skill missing (P1.M2.T4.S2 not landed)"; exit 1; }
grep -q 'func Index' internal/discover/discover.go \
  || { echo "FAIL GATE: discover.Index missing (P1.M2.T5.S1 not landed)"; exit 1; }
grep -q 'func Resolve' internal/resolve/resolve.go \
  || { echo "FAIL GATE: resolve.Resolve missing (P1.M3.T7.S1 not landed)"; exit 1; }
echo "GATE OK (dependencies present)"

# Format in place, then confirm nothing unformatted (silent == pass).
gofmt -w main.go main_test.go
test -z "$(gofmt -l main.go main_test.go)" \
  || { echo "FAIL: gofmt found unformatted files"; gofmt -d main.go main_test.go; exit 1; }
echo "gofmt OK"

# Vet the whole module (main is package main; vet ./... covers it).
go vet ./... || { echo "FAIL: go vet ./..."; exit 1; }
echo "go vet OK"

# Build the whole module (compile check; needs discover/resolve to exist).
go build ./... || { echo "FAIL: go build ./... (are discover.Index/Skill defined?)"; exit 1; }
echo "go build OK"
```

### Level 2: Unit / integration tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run the main package tests verbosely (package main == `go test .`).
go test . -v || { echo "FAIL: go test . -v"; exit 1; }

# Targeted: the load-bearing §6.4 + tag-resolution tests.
go test . -run \
  'TestRunTagSingleResolvesAbsolute|TestRunTagMultipleInInputOrder|TestRunTagOneBadAmongGoodPrintsNothingToStdout|TestRunTagMultipleBadAllReported|TestRunTagAmbiguousPrintsCandidates|TestRunTagSkillsDirNotFound|TestRunVersionPrecedenceOverTags|TestParseArgsCapturesPositionalTags' -v \
  || { echo "FAIL: load-bearing tag-resolution tests"; exit 1; }

# Whole module still green (skillsdir + discover + resolve + main).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: End-to-end CLI smoke test (PRD §13 acceptance slice for `<tag>`)

Build the real binary and run the §13 gates that this subtask owns. Requires the one
example skill — if `skills/example/SKILL.md` does not exist yet (P1.M6.T12), create a
throwaway one under a temp skills dir via SKPP_SKILLS_DIR so the gates still exercise
the code path.

```bash
cd /home/dustin/projects/skpp

go build -o skpp . || { echo "FAIL: build"; exit 1; }

# Use a throwaway skills dir with one example skill (skills/example ships in M6.T12;
# if absent, fabricate it here so the gates still run).
TMPROOT="$(mktemp -d)"
mkdir -p "$TMPROOT/example"
printf -- '---\nname: example\ndescription: An example skill\n---\nbody\n' > "$TMPROOT/example/SKILL.md"
export SKPP_SKILLS_DIR="$TMPROOT"

# §6.1 row 1: resolves to a real dir, exit 0.
test -d "$(./skpp example)" || { echo "FAIL: test -d \"\$(./skpp example)\""; exit 1; }
echo "single-resolve OK"

# §6.1 row 1 / §13: output is ABSOLUTE by default.
case "$(./skpp example)" in /*) echo "absolute OK";; *) echo "FAIL: not absolute"; exit 1;; esac

# §6.1 row 1: multiple tags -> one path/line, INPUT order.
./skpp example example | wc -l | grep -q '^2$' || { echo "FAIL: multi-tag line count"; exit 1; }
echo "multi-tag-lines OK"

# §6.4 error contract: unknown tag -> NOTHING on stdout, exit 1.
out=$(./skpp nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "unknown-tag contract OK" \
  || { echo "FAIL: unknown-tag contract (out='$out' rc=$rc)"; exit 1; }

# §6.4: one bad among good -> nothing on stdout, error on stderr, exit 1.
out=$(./skpp example nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "one-bad-among-good OK" \
  || { echo "FAIL: one-bad-among-good (out='$out' rc=$rc)"; exit 1; }

# §6.4 ambiguous: candidates on stderr, nothing on stdout.
mkdir -p "$TMPROOT/writing/reddit" "$TMPROOT/coding/reddit"
printf -- '---\nname: reddit-writer\n---\n\n' > "$TMPROOT/writing/reddit/SKILL.md"
printf -- '---\nname: reddit-coder\n---\n\n' > "$TMPROOT/coding/reddit/SKILL.md"
out=$(./skpp reddit 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "ambiguous-stdout-empty OK" \
  || { echo "FAIL: ambiguous (out='$out' rc=$rc)"; exit 1; }
err=$(./skpp reddit 2>&1 >/dev/null)
case "$err" in *"skpp: ambiguous tag 'reddit', candidates: coding/reddit writing/reddit"*) \
  echo "ambiguous-stderr OK";; *) echo "FAIL: ambiguous stderr='$err'"; exit 1;; esac

unset SKPP_SKILLS_DIR
rm -rf "$TMPROOT"
echo "Level 3 PASS"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# Only main.go and main_test.go changed; nothing else touched.
git diff --name-only | grep -Ev '^(main\.go|main_test\.go)$' \
  && { echo "FAIL: unexpected files modified"; git diff --name-only; exit 1; } || true
git diff --quiet PRD.md 2>/dev/null          || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
git diff --quiet internal/discover/discover.go   2>/dev/null || { echo "FAIL: discover.go changed (M2-owned)"; exit 1; }
git diff --quiet internal/resolve/resolve.go     2>/dev/null || { echo "FAIL: resolve.go changed (M3.T7-owned)"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go 2>/dev/null || { echo "FAIL: skillsdir.go changed (M1-owned)"; exit 1; }

# main.go imports the three internal packages + errors + strings (no surprises).
imp="$(sed -n '/^import (/,/^)/p' main.go)"
for want in errors strings fmt io os \
  github.com/dabstractor/skpp/internal/discover \
  github.com/dabstractor/skpp/internal/resolve \
  github.com/dabstractor/skpp/internal/skillsdir; do
  echo "$imp" | grep -q "\"$want\"" || { echo "FAIL: main.go missing import $want"; exit 1; }
done

# The §6.4 atomic-output contract is present in main.go.
grep -q 'func resolveTags' main.go || { echo "FAIL: resolveTags helper missing"; exit 1; }
grep -q 'errors.As' main.go        || { echo "FAIL: must use errors.As for typed errors"; exit 1; }
grep -q "skpp: unknown tag" main.go    || { echo "FAIL: §6.4 unknown wording missing"; exit 1; }
grep -q "skpp: ambiguous tag" main.go  || { echo "FAIL: §6.4 ambiguous wording missing"; exit 1; }
grep -q 'strings.Join(ae.Candidates, " ")' main.go || { echo "FAIL: candidates must be SPACE-joined"; exit 1; }
grep -q 'var paths strings.Builder' main.go || { echo "FAIL: must buffer stdout (strings.Builder)"; exit 1; }
grep -q 'r.Skill.Dir' main.go || { echo "FAIL: must print skill.Dir (default unit = directory)"; exit 1; }
# Default unit is the DIRECTORY, not SourceFile (--file is S2).
! grep -q 'SourceFile' main.go || { echo "FAIL: do not use SourceFile yet (--file is P1.M3.T2.S2)"; exit 1; }

# parseArgs captures positionals as tags; unknown flags still tolerated (no exit 2 yet).
grep -q 'c.tags = append(c.tags, a)' main.go || { echo "FAIL: positional tag capture missing"; exit 1; }
! grep -q 'os.Exit(2)' main.go || { echo "FAIL: exit 2 is P1.M5.T11, not this task"; exit 1; }

# Did NOT create later-milestone files/packages.
test ! -d internal/ui || { echo "FAIL: ui/ must not exist (M2.T6)"; exit 1; }
test ! -d cmd         || { echo "FAIL: cmd/ must not exist"; exit 1; }
test ! -f install.sh  || { echo "FAIL: install.sh must not exist (M6)"; exit 1; }
test ! -f README.md   || { echo "FAIL: README.md must not exist (M6)"; exit 1; }

# go.mod / go.sum UNCHANGED (dependency-neutral).
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null || { echo "FAIL: go.mod/go.sum changed"; git diff go.mod go.sum; exit 1; }

# The load-bearing tests exist in main_test.go.
for tn in TestRunTagSingleResolvesAbsolute TestRunTagMultipleInInputOrder \
  TestRunTagOneBadAmongGoodPrintsNothingToStdout TestRunTagMultipleBadAllReported \
  TestRunTagAmbiguousPrintsCandidates TestRunTagSkillsDirNotFound \
  TestRunVersionPrecedenceOverTags TestParseArgsCapturesPositionalTags \
  TestParseArgsFlagsAndTagsMixed; do
  grep -q "func $tn" main_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — Task-0 gate green (discover.Skill/Index + resolve.Resolve exist); `gofmt -l main.go main_test.go` silent; `go vet ./...` clean; `go build ./...` exit 0
- [ ] Level 2 PASS — `go test . -v` all main tests pass (new + updated + all pre-existing); `go test ./...` whole module green
- [ ] Level 3 PASS — §13 slice: `test -d "$(./skpp example)"`, absolute-path `case ... in /*)`, unknown-tag contract (empty stdout + exit 1), one-bad-among-good, ambiguous candidates on stderr
- [ ] Level 4 PASS — only main.go/main_test.go changed; §6.4 wording + errors.As + strings.Builder buffer + skill.Dir present; no SourceFile/exit-2; go.mod/go.sum unchanged

### Feature Validation (§6.1 row 1 + §6.4)
- [ ] Single good tag → absolute dir + newline, exit 0, empty stderr
- [ ] Multiple good tags → one path/line in INPUT order (not sorted), exit 0
- [ ] One bad among good → NOTHING on stdout, one stderr line per problem tag, exit 1
- [ ] Multiple bad → one stderr line per problem tag (input order), nothing on stdout, exit 1
- [ ] Ambiguous → `skpp: ambiguous tag '<tag>', candidates: <space-joined sorted>` on stderr, nothing on stdout, exit 1
- [ ] Skills dir unresolvable → ErrNotFound (one-line fix) on stderr, nothing on stdout, exit 1
- [ ] `--version` precedence over tags (PRD §6.3)
- [ ] stdout/stderr captured separately in tests (no leakage)

### Code Quality / Convention Validation
- [ ] main.go follows existing style (testable `run(args,stdout,stderr)int`; `parseArgs` switch; comments explain WHY)
- [ ] main_test.go mirrors repo convention (separate `*bytes.Buffer` for stdout/stderr; `t.Setenv`/`t.Chdir`; plain `t.Errorf`/`t.Fatalf`; NO testify; NO `t.Parallel`)
- [ ] The `resolveTags` success-path print step is a clean SINGLE location (the S2 extension point)
- [ ] All new exported-ish symbols documented (resolveTags godoc explains §6.4 + the S2 extension point)

### Scope Discipline
- [ ] Did NOT define Skill/Index/Resolve/Frontmatter (other packages own them — only IMPORT)
- [ ] Did NOT modify go.mod/go.sum, PRD.md, any tasks.json, internal/* (discover/resolve/skillsdir)
- [ ] Did NOT add --file/-f, --relative, --all/-a (those are P1.M3.T2.S2), --list/--search/check (M2/M4), or exit-2/--help/no-args-usage (P1.M5.T11)
- [ ] Did NOT create ui/cmd/install.sh/README.md/skills/
- [ ] Confirmed discover.Index + discover.Skill exist before implementing (Task-0 gate)

---

## Anti-Patterns to Avoid

- ❌ **Don't implement before `discover.Index`/`discover.Skill` exist.** main imports
  discover and calls Index + references r.Skill.Dir; the build fails with "undefined"
  until M2 lands. Run the Task-0 gate. (verified_facts §1.)
- ❌ **Don't print a partial result.** §6.4 is THE contract: resolve ALL tags first;
  on any failure, NOTHING to stdout. Printing each dir as you resolve it means a later
  bad tag leaves partial stdout → `pi --skill "$(skpp ...)"` gets a garbage path list.
  Two-pass + buffer. (verified_facts §3.)
- ❌ **Don't use resolve's `.Error()` for stderr.** The item CONTRACT specifies
  `skpp: unknown tag '<tag>'` / `skpp: ambiguous tag '<tag>', candidates: <space-joined>`
  (single quotes, `skpp:` prefix, SPACE-joined). resolve's `.Error()` is double-quoted,
  comma-joined, prefix-free. Extract `.Tag`/`.Candidates` via errors.As and format
  yourself. (verified_facts §5.)
- ❌ **Don't type-assert the errors.** Use `errors.As(err, &ae)` — the contract
  resolve's TestResolveErrorsAs proves, and it survives wrapping.
- ❌ **Don't sort the output by tag.** §6.1 row 1 says "input order". Iterate the
  user's tags; `skpp b a` prints b then a. (verified_facts §4.)
- ❌ **Don't `filepath.Abs`/`filepath.Clean` the Dir.** Print `r.Skill.Dir` verbatim
  (it is absolute by contract). Wrapping it masks an Index bug and the §13 `case ... in
  /*)` gate needs the first char to be `/`. (verified_facts §2.)
- ❌ **Don't add the modifiers or exit-2.** --file/-f, --relative, --all are S2;
  unknown-flag exit-2, mode-exclusivity, --help/no-args usage are M5. This task
  captures positionals as tags and tolerates unknown flags (no-op). Scope theft
  breaks parallel work. (verified_facts §0.)
- ❌ **Don't reorder the precedence.** version → path → tags → no-args(1). Putting
  tags before path would make `--path` lose to a stray positional; putting it before
  version breaks §6.3. (verified_facts §6.)
- ❌ **Don't change go.mod/go.sum.** main adds only stdlib + internal packages;
  yaml.v3 is already direct. `go mod tidy` is a no-op. (verified_facts §9.)
- ❌ **Don't use testify or `t.Parallel()` in tests.** Mirror the repo convention:
  separate `*bytes.Buffer`, `t.Setenv`/`t.Chdir`, plain assertions. (verified_facts §8.)

---

## Confidence Score

**9/10** — one-pass implementation success likelihood.

Rationale: the EXACT edits to main.go and the EXACT new tests are given verbatim
(gofmt-clean, compile as-is once `discover.Index`/`discover.Skill` exist). Every
consumed contract is locked (skillsdir LANDED + read in full; discover.Skill/Index
locked in go_architecture.md; resolve.Resolve/errors locked in the P1M3T1S1 PRP,
read in full). The change is purely additive (a new mode + tag capture); all
pre-existing main_test.go tests were traced and still pass. The §6.4 atomicity is
straightforward two-pass Go with a strings.Builder buffer. The single hard gate is
the discover.Index/Skill dependency (M2) — mitigated by the Task-0 check and by M3
being scheduled after M2. The only residual risk is the discover.Index-of-empty
behavior (`([], nil)` assumed) — mitigated by writing one real skill in the
multi-bad test so the index is non-empty.
