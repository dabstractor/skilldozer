# PRP — P1.M3.T7.S1: `internal/resolve` — Resolve() + Result/MatchKind + Unknown/Ambiguous errors

> **Subtask:** P1.M3.T7.S1 — the (only) subtask of T7 (§7.2 precedence resolver,
> PRD §7.2). This is milestone **M3** (Tag resolution & path output); T8 (main
> tag-resolution output + §6.4 error contract) builds directly on this package's
> public API. Output directory is the orchestrator's `P1M3T1S1` (this is plan item
> `P1.M3.T7.S1`; directory name differs from plan id — use the path given).
>
> **Scope:** create `internal/resolve/resolve.go` (`package resolve`) and its
> white-box test `internal/resolve/resolve_test.go`. Define `MatchKind`
> (Canonical/Basename/Name/Alias), `Result{Skill, Match}`, the typed errors
> `*UnknownError` / `*AmbiguousError`, and the pure function
> `Resolve(tag, skills) (Result, error)` implementing PRD §7.2 precedence.
>
> **DEPENDENCY (hard):** `discover.Skill`. That struct is the deliverable of
> **P1.M2.T4.S2** ("metadata extraction + Skill type") — it does NOT exist at
> research time. Its field shape is LOCKED in `go_architecture.md` and is
> re-stated verbatim in §"All Needed Context" below. Because `Resolve` takes the
> index `[]discover.Skill` as a **parameter**, the resolve tests construct
> `discover.Skill` literals directly and need **neither `Index()` nor a `skills/`
> tree** — only the `Skill` TYPE. M3 is scheduled after M2, so S2 (and T5) are
> complete when this PRP runs. **Do not implement this subtask until
> `internal/discover.Skill` exists** (build fails otherwise).
>
> **PARALLEL CONTEXT:** P1.M2.T4.S1 (Frontmatter + ParseFrontmatter) is already
> LANDED on disk (`internal/discover/discover.go` exists, tests green). This PRP
> does NOT touch it. It only IMPORTS `discover` for the `Skill` type.

---

## Goal

**Feature Goal**: Implement the PRD §7.2 tag-resolution precedence as a pure,
I/O-free function over `[]discover.Skill`. Given a `tag`, resolve it to exactly
one skill via (1) exact canonical `RelTag`, (2) `RelTag` basename, (3) frontmatter
`Name`, (4) declared `Alias` — first match wins, >1 match at a step ⇒
`*AmbiguousError`, nothing ⇒ `*UnknownError`. This is the brain `main`'s
tag-resolution mode (P1.M3.T8.S1) calls once per `<tag>` argument.

**Deliverable**: Two NEW files (nothing else touched):
1. `internal/resolve/resolve.go` — `package resolve`; `type MatchKind int` +
   constants; `type Result struct`; `type UnknownError`/`AmbiguousError` (pointer-
   receiver `error`); `func Resolve(tag string, skills []discover.Skill) (Result, error)`;
   three private helpers (`basename`, `collectMatches`, `sortedRelTags`).
2. `internal/resolve/resolve_test.go` — `package resolve` (white-box); the PRD §7.2
   examples table tested exactly, plus ambiguous (basename/name/alias), unknown,
   precedence-first-match-wins, empty-tag guard, `errors.As`, error messages, and
   `MatchKind.String()`.

**Success Definition**: `gofmt -l internal/resolve/*.go` silent; `go vet
./internal/resolve/` clean; `go build ./...` and `go test ./...` pass;
`go.mod`/`go.sum` **unchanged** (resolve adds no external dependency — verified
fact §7). All resolve tests pass, including the four PRD §7.2 example rows
(foo→canonical, writing/reddit→canonical, reddit→basename, foo-helper→name).

---

## Why

- This subtask **proves §7.2 resolution works in isolation** before `main` wires
  it into the CLI. PRD §18 lists it as build-order step 3 (right after discovery).
  Until `Resolve` exists and is tested against the §7.2 examples + ambiguity +
  unknown cases, the tag-resolution mode (T8) is blocked.
- It **locks the typed-error contract** that `main` (P1.M3.T8.S1) and `--list`/
  `check` (M2.T6/M4.T10) depend on: `*AmbiguousError{Tag, Candidates}` carries the
  disambiguating full tags (PRD §6.4 lists candidates on stderr), and
  `*UnknownError{Tag}` signals "nothing matched". Getting the type shape or the
  pointer-vs-value receiver wrong now silently breaks `errors.As` in main later.
- It **locks the §7.2 precedence ordering** (canonical beats basename beats name
  beats alias; first match wins). The examples table is the regression anchor.
- It is **go.mod-neutral**: resolve imports only the stdlib + the internal
  `discover` package, so unlike P1.M2.T4.S1 there is NO `go.mod`/`go.sum` change.

---

## What

A `package resolve` (in `internal/resolve/`, unimportable outside the module)
containing:

1. **`type MatchKind int`** with constants `Canonical`, `Basename`, `Name`, `Alias`
   (in that `iota` order), plus a `String()` method (mirrors `skillsdir.Source.String()`).
2. **`type Result struct { Skill discover.Skill; Match MatchKind }`** — the success
   outcome. Zero value `Result{}` is NOT a valid success (returned alongside a
   non-nil error).
3. **`type UnknownError struct{ Tag string }`** and
   **`type AmbiguousError struct{ Tag string; Candidates []string }`**, each with a
   pointer-receiver `Error() string`. `*UnknownError` and `*AmbiguousError` satisfy
   `error`; the value types do NOT (intended — stdlib convention).
4. **`func Resolve(tag string, skills []discover.Skill) (Result, error)`** applying
   the §7.2 precedence (see verified_facts §2). Pure: no I/O, no mutation.
5. Three private helpers: `basename(relTag)` (last `/`-component),
   `collectMatches(skills, kind, pred)` (the shared collect loop for steps 2–4),
   `sortedRelTags(skills)` (sorted RelTags for `AmbiguousError.Candidates`).

### Success Criteria

- [ ] `internal/resolve/resolve.go` is `package resolve` with exactly: `MatchKind`
      + 4 constants + `String()`; `Result`; `UnknownError`/`AmbiguousError` with
      pointer-receiver `Error()`; `Resolve`; helpers `basename`/`collectMatches`/
      `sortedRelTags`.
- [ ] Production imports are EXACTLY `fmt`, `sort`, `strings`,
      `github.com/dabstractor/skpp/internal/discover` (no `path`, no `errors` in
      production — `errors` is test-only).
- [ ] PRD §7.2 examples pass exactly: `foo`→Canonical/foo, `writing/reddit`→
      Canonical/writing-reddit, `reddit`→Basename/writing-reddit,
      `foo-helper`→Name/foo.
- [ ] Ambiguous basename/name/alias each return `*AmbiguousError` with the sorted
      matching RelTags in `Candidates` and the input `tag` in `Tag`.
- [ ] Unknown tag and empty/nil `skills` both return `*UnknownError{Tag: tag}`.
- [ ] First-match-wins: a tag matching both canonical and name (or canonical and
      basename) resolves at the EARLIER step.
- [ ] Empty-name guard: a skill with `Name==""` is never matched by step 3 (so an
      empty tag, or a no-frontmatter skill, does not spuriously resolve).
- [ ] `*AmbiguousError` / `*UnknownError` are extractable via `errors.As` (the
      contract `main` will rely on).
- [ ] `gofmt -l` silent; `go vet` clean; `go build ./...` + `go test ./...` pass;
      `go.mod`/`go.sum` unchanged (`git diff --quiet go.mod go.sum`).

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `resolve.go` and `resolve_test.go` is given verbatim
in the Implementation Blueprint (copy-paste clean, gofmt-clean, compiles as-is once
`discover.Skill` exists — the algorithm is straightforward stdlib Go over a locked
type contract). The consumed contract is `discover.Skill` (field shape locked in
go_architecture.md; only `RelTag`/`Name`/`Aliases` are read). Every load-bearing
decision — pointer receivers, candidate sorting, basename via LastIndex, the
empty-field guard, ambiguity short-circuit, go.mod-neutrality, the test convention
— is documented in `research/verified_facts.md` and the Known Gotchas below. An
implementer who knows Go but nothing about this repo can complete this in one pass
from this document (provided `discover.Skill` already exists in the module)._

### Documentation & References

```yaml
# MUST READ — this subtask's own decisions (every load-bearing choice)
- file: plan/001_fcde63e5bb60/P1M3T1S1/research/verified_facts.md
  why: "Locks: (0) resolve is pure, takes []discover.Skill as a param. (1) the
        Skill field shape resolve consumes (RelTag/Name/Aliases only; RelTag is
        already slash-normalized). (2) the §7.2 precedence + ambiguity short-circuit.
        (3) the 4-row examples table traced step-by-step. (4) the type shape
        (MatchKind/Result/errors). (5) design decisions: pointer receivers,
        MatchKind.String(), basename via strings.LastIndex, sorted Candidates,
        empty-field guard. (6) error message format. (7) go.mod/go.sum UNCHANGED.
        (8) white-box test convention, no testify/Parallel. (9) scope boundary."
  critical: "The hard dependency is discover.Skill (P1.M2.T4.S2). Do NOT implement
             this subtask until internal/discover.Skill exists. go.mod is neutral
             (no external dep added) — contrast with P1.M2.T4.S1 which flipped
             yaml.v3 to direct."

# CONTRACT — the discover.Skill type shape (LOCKED; consumed verbatim by resolve)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "'Core types > internal/discover' defines type Skill struct (Dir, RelTag,
        Name, Description, Keywords, Category, Aliases, HasFM, SourceFile) AND
        'Core types > internal/resolve' defines Result{Skill, Match}, MatchKind
        (Canonical|Basename|Name|Alias), Resolve(tag, skills)(Result,error),
        AmbiguousError{Tag, Candidates}, UnknownError{Tag}. The data-flow note
        shows main calling discover.Index() then resolve.Resolve per tag."
  section: "Core types > internal/resolve", "Core types > internal/discover", "Data flow"
  critical: "The architecture comment says Resolve 'Returns ErrUnknown/ErrAmbiguous'
             — that is LOOSE naming for the typed structs UnknownError/AmbiguousError
             (the item CONTRACT is authoritative). Resolve signature is
             (tag string, skills []discover.Skill) (Result, error)."

# CONTRACT — the §7.2 precedence + §6.4 error semantics this implements
- file: PRD.md
  why: "§7.2: the 5-step precedence (canonical→basename→name→alias→unknown), the
        '>1 ⇒ ambiguous' rule, and the 4 examples (foo/writing-reddit/reddit/
        foo-helper). §6.4: ambiguous tag ⇒ stderr lists candidate FULL tags, exit 1;
        any unresolved/ambiguous ⇒ print nothing to stdout (main's job, but resolve
        supplies the Candidates). §7.1: the Skill fields Index() populates. READ-ONLY."
  section: "7.2 Tag resolution precedence", "6.4 Error semantics", "7.1 Discovery"

# PRIOR PRP (CONTRACT for what resolve consumes) — discover package design
- file: plan/001_fcde63e5bb60/P1M2T4S1/PRP.md
  why: "Locks Frontmatter + ParseFrontmatter (S1) AND documents in 'DOWNSTREAM
        EXTENSION POINTS' that P1.M2.T4.S2 defines type Skill struct with EXACTLY
        the fields in go_architecture.md, and that T5's Index() returns skills
        sorted by RelTag. resolve relies on both: the Skill fields and the
        sorted-by-RelTag input (resolve ALSO sorts Candidates itself, so output is
        deterministic even if a future caller passes unsorted input)."
  section: "Integration Points > DOWNSTREAM EXTENSION POINTS"

# REFERENCE — the repo's enum-stringer + test conventions (mirror these exactly)
- file: internal/skillsdir/skillsdir.go
  why: "Two conventions to mirror: (1) Source int + Source.String() switch — the
        exact pattern for MatchKind.String(). (2) small factored private helpers
        (resolveSiblingFromExe, findWalkUpAncestor) that are testable in isolation
        — the pattern for basename/collectMatches/sortedRelTags."
  pattern: "int enum with a switch-based String(); private helpers factored out."

- file: internal/skillsdir/skillsdir_test.go
  why: "The established test convention: `package skillsdir` (white-box), plain
        t.Errorf/t.Fatalf, table-driven cases, NO testify, NO t.Parallel().
        resolve_test.go mirrors this as `package resolve`."
  pattern: "White-box same-package test; table-driven; plain assertions; no Parallel."

# URLS — the load-bearing stdlib surface
- url: https://pkg.go.dev/strings#LastIndex
  why: "strings.LastIndex(s, \"/\") finds the last slash for basename extraction.
        Returns -1 when absent → the whole string is the basename. Zero-alloc,
        faithful to the item's 'split on /, take last element'."
- url: https://pkg.go.dev/sort#Strings
  why: "sort.Strings(candidates) makes AmbiguousError.Candidates deterministic
        regardless of input slice order (PRD §6.4 wants stable stderr)."
- url: https://pkg.go.dev/errors#As
  why: "errors.As(err, &target) is how main (P1.M3.T8.S1) will extract the typed
        errors. Pointer receivers make *UnknownError/*AmbiguousError satisfy error,
        so errors.As works. The test verifies this contract."
```

### Current Codebase tree (M1 + M2.T4.S1 landed; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/discover/discover.go        # M2.T4.S1: Frontmatter + utf8BOM + ParseFrontmatter (LANDED)
internal/discover/discover_test.go   # M2.T4.S1 tests (white-box, package discover)
internal/skillsdir/skillsdir.go      # M1.T2: Source + Find + per-rule helpers
internal/skillsdir/skillsdir_test.go # M1.T2 tests (white-box, package skillsdir)
main.go                              # M1.T3: arg-parse skeleton + --path + --version
main_test.go                         # M1.T3 tests

$ ls -A
.git/ .gitignore LICENSE PRD.md go.mod go.sum internal/ main.go main_test.go plan/ .pi-subagents/
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT after S1 tidy)
# NOTE: internal/discover/discover.go currently has Frontmatter + ParseFrontmatter ONLY.
#       type Skill struct is NOT yet defined (it lands in P1.M2.T4.S2). This subtask
#       cannot compile until S2 lands — see DEPENDENCY above.
# NO internal/resolve/, ui/. NO skills/ (P1.M6.T12 ships skills/example).
```

### Desired Codebase tree with files to be added

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/skillsdir/*,
│        internal/discover/* [now incl. Skill once S2 lands], main.go, main_test.go
│        — ALL UNCHANGED by this subtask)
└── internal/
    └── resolve/
        ├── resolve.go       # CREATE — package resolve: MatchKind, Result, errors, Resolve, helpers
        └── resolve_test.go  # CREATE — package resolve (white-box): §7.2 table + ambiguous + unknown + ...
```

| File (created) | Responsibility | Imports |
|---|---|---|
| `internal/resolve/resolve.go` | Resolve one tag to a skill via §7.2 precedence; typed errors | `fmt`, `sort`, `strings`, `github.com/dabstractor/skpp/internal/discover` |
| `internal/resolve/resolve_test.go` | White-box tests: §7.2 examples, ambiguous, unknown, precedence, guard, errors.As | `errors`, `testing`, `github.com/dabstractor/skpp/internal/discover` |

**One new directory (`internal/resolve/`). NO `go.mod`/`go.sum` change (resolve adds
no external dependency). No `main.go`, no `discover`/`skillsdir`/`ui` touch, no
`skills/`, no `Index()`.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — HARD DEPENDENCY on discover.Skill (P1.M2.T4.S2). It does not exist at
// research time. resolve.go imports "github.com/dabstractor/skpp/internal/discover"
// and references discover.Skill{RelTag, Name, Aliases, ...}. If you implement this
// subtask before S2 lands, `go build ./internal/resolve/` fails with "undefined:
// discover.Skill". M3 is scheduled after M2, so S2/T5 are done when this runs. Do
// NOT define Skill here (that steals S2's deliverable) — only CONSUME it.
//   RIGHT: import discover; use discover.Skill in Result + Resolve signature.
//   WRONG: define type Skill in resolve.go (S2 owns it; would shadow/conflict).

// GOTCHA #2 — Pointer receivers for the error types. *UnknownError and
// *AmbiguousError implement error; the VALUE types do NOT. Return them as
// &UnknownError{...} / &AmbiguousError{...}. This is the stdlib convention
// (*url.Error, *json.SyntaxError, *os.PathError all use pointer receivers) and it
// is REQUIRED for errors.As to extract them cleanly in main. Do NOT use value
// receivers — then both value and pointer would implement error, but the
// AmbiguousError carries a slice (not comparable with ==) and pointer is standard.
//   RIGHT: func (e *AmbiguousError) Error() string { ... }; return &AmbiguousError{...}
//   WRONG: func (e AmbiguousError) Error() string   // value receiver

// GOTCHA #3 — Ambiguity SHORT-CIRCUITS. If step 2 (basename) matches >1 skill,
// return *AmbiguousError immediately; do NOT fall through to step 3 (name). A
// looser match must never "rescue" an ambiguity — the caller must see the
// ambiguous error so it can list candidates and exit 1 (PRD §6.4). Each step is
// consulted ONLY if every earlier step produced NOTHING (0 matches). The
// collectMatches helper returns the match slice; Resolve decides 0/1/>1.

// GOTCHA #4 — First match wins; canonical beats basename beats name beats alias.
// For tag "foo" against a top-level skill with RelTag "foo": step 1 (exact RelTag)
// hits and returns Canonical — even though basename("foo")=="foo" would ALSO match
// at step 2. Step 1 must be checked FIRST and must RETURN on its (single) hit. This
// is the PRD §7.2 "first match wins; later steps only consulted if earlier
// produced nothing" rule. The foo→Canonical example row is the regression anchor.

// GOTCHA #5 — basename via strings.LastIndex, NOT path.Base. The item CONTRACT
// says "split on /, take last element". RelTag is ALREADY slash-normalized by
// discover (filepath.ToSlash), so splitting on "/" is correct on every OS. Use
// strings.LastIndex(relTag, "/"); if -1, the whole string is the basename. This
// avoids importing "path" (we already import "strings") and is faithful to the
// spec wording. For a clean RelTag, LastIndex and path.Base are identical.
//   RIGHT: if i := strings.LastIndex(relTag, "/"); i >= 0 { return relTag[i+1:] }
//   WRONG: path.Base(relTag)   // works, but adds an import and re-cleans the path

// GOTCHA #6 — Candidates are SORTED (sort.Strings) for deterministic stderr.
// PRD §6.4 wants stable candidate listing for scripting. resolve sorts
// AmbiguousError.Candidates itself so the output is deterministic EVEN IF a future
// caller passes an unsorted index (defensive — Index() already sorts by RelTag).
// The tests pass the fixture in REVERSE sorted order to prove the sort runs.
//   RIGHT: sort.Strings(tags) inside sortedRelTags.
//   WRONG: return matches in iteration order   // nondeterministic if input unsorted

// GOTCHA #7 — Empty-field guard on steps 3 (Name) and 4 (alias). A skill with NO
// frontmatter has Name=="". It must NOT match step 3 — a "frontmatter name" that is
// absent is not a matchable name. Guard with `s.Name != "" && s.Name == tag`. This
// ALSO prevents a degenerate empty tag ("") from spuriously resolving to a
// missing-name skill (Resolve("", [Skill{Name:""}]) ⇒ UnknownError, not Name match).
// RelTag and its basename are always non-empty for a real skill, so steps 1–2 need
// no guard. (main never passes an empty tag, but resolve must be correct in isolation.)
//   RIGHT: pred: func(s) bool { return s.Name != "" && s.Name == tag }
//   WRONG: pred: func(s) bool { return s.Name == tag }   // "" matches missing-name skills

// GOTCHA #8 — Duplicate alias within ONE skill counts ONCE. A skill whose Aliases
// contains ["x","x"] still matches "x" exactly once (the predicate returns true;
// collectMatches appends the skill once). Do NOT break-then-re-add. A skill is in
// the match set at most once per step (it either has the property or not).
//   RIGHT: pred loops Aliases, returns true on first ==tag; collectMatches appends once.

// GOTCHA #9 — NO "skpp:" prefix on error messages. skillsdir.ErrNotFound is
// prefix-free ("could not locate the skills directory: ...") — main adds program
// context. Match that convention: UnknownError.Error() = `unknown skill tag %q`;
// AmbiguousError.Error() = `ambiguous skill tag %q matches: <joined>`. main
// (P1.M3.T8.S1) owns the final §6.4 stderr wording; these are sensible defaults
// and the test asserts their exact text.

// GOTCHA #10 — go.mod/go.sum are UNCHANGED. resolve imports only the stdlib
// (fmt/sort/strings) + the INTERNAL discover package. yaml.v3 is ALREADY a direct
// dependency (flipped by P1.M2.T4.S1's go mod tidy). So `go mod tidy` is a no-op
// and go.mod/go.sum are byte-identical before/after. This is the clean contrast to
// S1. If `go mod tidy` DOES change anything, something is wrong (you added an
// external import you should not have). Verify with `git diff --quiet go.mod go.sum`.

// GOTCHA #11 — resolve_test.go is WHITE-BOX (`package resolve`), NOT
// `package resolve_test`. Mirrors internal/skillsdir/skillsdir_test.go (`package
// skillsdir`) and discover_test.go (`package discover`). White-box lets tests call
// the private helpers (basename/sortedRelTags) if desired and matches the repo
// convention. NO testify; NO t.Parallel() (repo convention is no-Parallel
// everywhere). Fixtures are discover.Skill struct LITERALS — no t.TempDir, no
// files (resolve is pure, so no disk fixtures are needed, unlike discover/skillsdir).

// GOTCHA #12 — Compare Candidates by length + indexed elements, NOT reflect.DeepEqual.
// discover_test.go checks metadata lists manually (kw[0]/kw[1]); mirror that style
// and avoid the reflect import. Keep test imports to exactly: errors, testing,
// discover. (errors is needed only for the errors.As contract test.)
```

---

## Implementation Blueprint

### Data model — the type shape (LOCKED by go_architecture.md + item CONTRACT)

```go
package resolve

import "github.com/dabstractor/skpp/internal/discover"

// MatchKind: which §7.2 step resolved the tag.
type MatchKind int
const ( Canonical MatchKind = iota; Basename; Name; Alias )

// Result: the success outcome.
type Result struct {
	Skill discover.Skill
	Match MatchKind
}

// Typed errors (pointer receivers → *T implements error).
type UnknownError   struct{ Tag string }
type AmbiguousError struct{ Tag string; Candidates []string }
```

### File 1 — `internal/resolve/resolve.go` (CREATE)

Create the file with EXACTLY this content (gofmt-clean; pure stdlib + discover):

```go
// Package resolve maps a user-supplied tag to a single skill using the PRD §7.2
// precedence. It is a PURE function over []discover.Skill: no filesystem, no
// global state, no I/O — it takes the already-built index as a parameter and main
// (P1.M3.T8.S1) supplies it from discover.Index().
//
// The precedence (first match wins; a later step is consulted ONLY if every
// earlier step produced nothing) is, in order:
//
//  1. Canonical — tag == skill.RelTag (case-sensitive). RelTag is unique per
//     directory, so at most one hit.
//  2. Basename  — tag == the final '/'-component of skill.RelTag (e.g. "reddit"
//     matches "writing/reddit"). >1 hit ⇒ *AmbiguousError.
//  3. Name      — tag == skill.Name (the frontmatter name). >1 ⇒ *AmbiguousError.
//  4. Alias     — tag appears in skill.Aliases (metadata.aliases). >1 ⇒ *Ambiguous.
//  5. otherwise — *UnknownError.
//
// An ambiguity at any step SHORT-CIRCUITS: *AmbiguousError is returned immediately
// and later steps are NOT consulted (a looser match cannot rescue an ambiguity;
// the caller must see the candidates per PRD §6.4).
//
// AmbiguousError.Candidates is the SORTED list of the matching skills' RelTags —
// sorted here so the error is deterministic regardless of how the caller ordered
// the input slice (PRD §6.4 wants stable stderr for scripting).
package resolve

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

// MatchKind identifies which §7.2 step resolved a tag. Its zero value is not a
// valid success; Resolve always sets it on the success path. Exported so callers
// can switch on it (e.g. --list/debug could annotate "reddit (basename)").
type MatchKind int

const (
	// Canonical means tag == skill.RelTag (step 1, exact canonical tag).
	Canonical MatchKind = iota
	// Basename means tag == the final '/'-component of skill.RelTag (step 2).
	Basename
	// Name means tag == skill.Name, the frontmatter name (step 3).
	Name
	// Alias means tag appeared in skill.Aliases (step 4).
	Alias
)

// String renders a MatchKind for logs/debug, mirroring skillsdir.Source.String().
// An out-of-range value renders as "unknown".
func (m MatchKind) String() string {
	switch m {
	case Canonical:
		return "canonical"
	case Basename:
		return "basename"
	case Name:
		return "name"
	case Alias:
		return "alias"
	default:
		return "unknown"
	}
}

// Result is the outcome of resolving one tag. The zero value Result{} is NOT a
// valid success: Resolve returns it only together with a non-nil error.
type Result struct {
	Skill discover.Skill
	Match MatchKind
}

// UnknownError is returned by Resolve when no §7.2 step matched the tag. main
// prints it to stderr and exits 1 (PRD §6.4).
type UnknownError struct {
	Tag string
}

// Error implements error. No "skpp:" prefix (main adds program context, mirroring
// skillsdir.ErrNotFound).
func (e *UnknownError) Error() string {
	return fmt.Sprintf("unknown skill tag %q", e.Tag)
}

// AmbiguousError is returned when a short tag matched >1 skill at the SAME
// precedence step. Candidates is the sorted list of the matching skills' RelTags
// (the full canonical tags the user can use to disambiguate). main lists them on
// stderr and exits 1 (PRD §6.4).
type AmbiguousError struct {
	Tag        string
	Candidates []string
}

// Error implements error. Candidates are joined with ", " for a readable line.
func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("ambiguous skill tag %q matches: %s", e.Tag, strings.Join(e.Candidates, ", "))
}

// Resolve applies the PRD §7.2 precedence to tag against skills and returns the
// single matching skill, or a typed error (*UnknownError / *AmbiguousError).
//
// It is pure: it does not touch the filesystem or mutate skills. It consults each
// precedence step only if every earlier step produced no match. An ambiguity at
// any step returns *AmbiguousError immediately (later steps are NOT consulted).
//
// Field-level gotcha: step 3 (Name) and step 4 (Alias) only consider a skill whose
// relevant field is non-empty. A skill with no frontmatter (Name=="") is never
// matched by name, and an empty alias never matches; this prevents a degenerate
// empty tag (or a missing-name skill) from spuriously resolving. RelTag and its
// basename are always non-empty for a real skill, so steps 1–2 need no guard.
func Resolve(tag string, skills []discover.Skill) (Result, error) {
	// Step 1 — exact canonical tag. RelTag is unique per directory ⇒ at most one.
	// First (only) match wins; no ambiguity is possible at this step.
	for _, s := range skills {
		if s.RelTag == tag {
			return Result{Skill: s, Match: Canonical}, nil
		}
	}

	// Step 2 — basename (final '/'-component of RelTag).
	if m := collectMatches(skills, func(s discover.Skill) bool {
		return basename(s.RelTag) == tag
	}); len(m) == 1 {
		return Result{Skill: m[0], Match: Basename}, nil
	} else if len(m) > 1 {
		return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)}
	}

	// Step 3 — frontmatter name (skip skills with no name: a missing name is not
	// a matchable name, and this guards against an empty tag matching Name=="").
	if m := collectMatches(skills, func(s discover.Skill) bool {
		return s.Name != "" && s.Name == tag
	}); len(m) == 1 {
		return Result{Skill: m[0], Match: Name}, nil
	} else if len(m) > 1 {
		return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)}
	}

	// Step 4 — declared alias.
	if m := collectMatches(skills, func(s discover.Skill) bool {
		for _, a := range s.Aliases {
			if a == tag {
				return true
			}
		}
		return false
	}); len(m) == 1 {
		return Result{Skill: m[0], Match: Alias}, nil
	} else if len(m) > 1 {
		return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)}
	}

	// Step 5 — nothing matched.
	return Result{}, &UnknownError{Tag: tag}
}

// collectMatches returns every skill for which pred returns true, in input order.
// It is the shared collection loop for steps 2–4 (step 1 is exact-and-unique, so
// it is inlined in Resolve). Each skill appears at most once: pred is a property
// of the skill, so it is true or false, never "twice".
func collectMatches(skills []discover.Skill, pred func(discover.Skill) bool) []discover.Skill {
	var hit []discover.Skill
	for _, s := range skills {
		if pred(s) {
			hit = append(hit, s)
		}
	}
	return hit
}

// basename returns the final '/'-component of a slash-normalized relTag (e.g.
// "writing/reddit" → "reddit"). relTag is always slash-normalized by discover
// (filepath.ToSlash), so splitting on '/' is correct on every platform and no
// OS-separator handling is needed here. A tag with no '/' is its own basename.
// Uses strings.LastIndex (zero-alloc) rather than path.Base to stay faithful to
// the item's "split on /, take last element" and avoid importing "path".
func basename(relTag string) string {
	if i := strings.LastIndex(relTag, "/"); i >= 0 {
		return relTag[i+1:]
	}
	return relTag
}

// sortedRelTags returns the RelTags of skills, sorted ascending. Used for
// AmbiguousError.Candidates so the error is deterministic regardless of the input
// slice order (PRD §6.4 wants stable stderr for scripting).
func sortedRelTags(skills []discover.Skill) []string {
	tags := make([]string, len(skills))
	for i, s := range skills {
		tags[i] = s.RelTag
	}
	sort.Strings(tags)
	return tags
}
```

### File 2 — `internal/resolve/resolve_test.go` (CREATE, `package resolve` white-box)

Create the file with EXACTLY this content. It mirrors the repo's test convention
(white-box same-package, plain `t.Errorf`/`t.Fatalf`, table-driven, no testify, no
`t.Parallel()`). Fixtures are `discover.Skill` struct literals (resolve is pure —
no disk fixtures needed).

```go
package resolve

import (
	"errors"
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

// exampleSkills mirrors the PRD §7.2 example setup EXACTLY: a top-level skill
// `foo` whose frontmatter name is `foo-helper`, and a nested skill
// `writing/reddit`. Only RelTag/Name/Aliases influence resolution; Dir/SourceFile
// are filled with realistic absolute paths so a returned Result.Skill is usable by
// main. reddit is given an alias "social" so the example fixture also exercises
// the alias step without needing a second fixture.
var exampleSkills = []discover.Skill{
	{Dir: "/repo/skills/foo", RelTag: "foo", Name: "foo-helper", SourceFile: "/repo/skills/foo/SKILL.md"},
	{Dir: "/repo/skills/writing/reddit", RelTag: "writing/reddit", Name: "reddit-poster", Aliases: []string{"social"}, SourceFile: "/repo/skills/writing/reddit/SKILL.md"},
}

// TestResolveExamples is THE PRD §7.2 examples table (the item's required test),
// plus the alias step on the same fixture. Each row asserts both the resolved
// RelTag and the MatchKind.
func TestResolveExamples(t *testing.T) {
	cases := []struct {
		tag       string
		wantRel   string
		wantMatch MatchKind
	}{
		{"foo", "foo", Canonical},                       // exact RelTag
		{"writing/reddit", "writing/reddit", Canonical}, // exact RelTag (nested)
		{"reddit", "writing/reddit", Basename},          // basename, unambiguous
		{"foo-helper", "foo", Name},                     // frontmatter name
		{"social", "writing/reddit", Alias},             // declared alias
	}
	for _, c := range cases {
		got, err := Resolve(c.tag, exampleSkills)
		if err != nil {
			t.Errorf("Resolve(%q): err=%v; want nil", c.tag, err)
			continue
		}
		if got.Match != c.wantMatch {
			t.Errorf("Resolve(%q): Match=%v; want %v", c.tag, got.Match, c.wantMatch)
		}
		if got.Skill.RelTag != c.wantRel {
			t.Errorf("Resolve(%q): RelTag=%q; want %q", c.tag, got.Skill.RelTag, c.wantRel)
		}
	}
}

// TestResolveAmbiguous exercises the >1-match case at each of steps 2/3/4. Input
// is deliberately passed in REVERSE sorted order to prove sortedRelTags sorts the
// Candidates regardless of input ordering.
func TestResolveAmbiguous(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		// skills listed REVERSE-sorted so Candidates sorting is observable.
		skills []discover.Skill
		want   []string // expected sorted Candidates
	}{
		{
			name: "basename",
			tag:  "reddit",
			skills: []discover.Skill{
				{RelTag: "writing/reddit", Name: "a"},
				{RelTag: "coding/reddit", Name: "b"},
			},
			want: []string{"coding/reddit", "writing/reddit"},
		},
		{
			name: "name",
			tag:  "dup",
			skills: []discover.Skill{
				{RelTag: "beta", Name: "dup"},
				{RelTag: "alpha", Name: "dup"},
			},
			want: []string{"alpha", "beta"},
		},
		{
			name: "alias",
			tag:  "shared",
			skills: []discover.Skill{
				{RelTag: "beta", Aliases: []string{"shared"}},
				{RelTag: "alpha", Aliases: []string{"shared"}},
			},
			want: []string{"alpha", "beta"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := Resolve(c.tag, c.skills)
			if err == nil {
				t.Fatalf("Resolve(%q) [%s]: err=nil res=%+v; want *AmbiguousError", c.tag, c.name, res)
			}
			ae, ok := err.(*AmbiguousError)
			if !ok {
				t.Fatalf("Resolve(%q) [%s]: err type=%T; want *AmbiguousError", c.tag, c.name, err)
			}
			if ae.Tag != c.tag {
				t.Errorf("Tag=%q; want %q", ae.Tag, c.tag)
			}
			if len(ae.Candidates) != len(c.want) {
				t.Fatalf("Candidates=%v; want %v", ae.Candidates, c.want)
			}
			for i, want := range c.want {
				if ae.Candidates[i] != want {
					t.Errorf("Candidates[%d]=%q; want %q (full=%v)", i, ae.Candidates[i], want, ae.Candidates)
				}
			}
		})
	}
}

// TestResolveUnknown: a tag matching nothing, and an empty/nil index, both yield
// *UnknownError{Tag: tag}. No panic on nil/empty input.
func TestResolveUnknown(t *testing.T) {
	// Unknown tag against the example index.
	res, err := Resolve("nope", exampleSkills)
	if err == nil {
		t.Fatalf("Resolve(nope): err=nil res=%+v; want *UnknownError", res)
	}
	ue, ok := err.(*UnknownError)
	if !ok {
		t.Fatalf("Resolve(nope): err type=%T; want *UnknownError", err)
	}
	if ue.Tag != "nope" {
		t.Errorf("Tag=%q; want nope", ue.Tag)
	}

	// Empty index ⇒ unknown (range over nil/empty is a no-op).
	if _, err := Resolve("anything", nil); err == nil {
		t.Fatal("Resolve(anything, nil): err=nil; want *UnknownError")
	}
	if _, err := Resolve("anything", []discover.Skill{}); err == nil {
		t.Fatal("Resolve(anything, []): err=nil; want *UnknownError")
	}
}

// TestResolvePrecedence: first-match-wins. A tag that matches at an EARLIER step
// must resolve there even if it would also match a later step.
func TestResolvePrecedence(t *testing.T) {
	// Canonical beats Name: skill A has RelTag "x"; skill B has Name "x".
	// tag "x" must resolve to A at step 1 (Canonical), NOT B at step 3 (Name).
	skills := []discover.Skill{
		{RelTag: "x", Name: "a-name", Dir: "/s/x"},
		{RelTag: "y", Name: "x"},
	}
	got, err := Resolve("x", skills)
	if err != nil {
		t.Fatalf("Resolve(x) precedence: err=%v; want nil", err)
	}
	if got.Match != Canonical {
		t.Errorf("Match=%v; want Canonical (step 1 beats step 3)", got.Match)
	}
	if got.Skill.RelTag != "x" {
		t.Errorf("RelTag=%q; want x", got.Skill.RelTag)
	}

	// Canonical beats Basename: a top-level skill "foo" — tag "foo" is BOTH the
	// exact RelTag (step 1) AND its own basename (step 2). Step 1 must win.
	// (Covered by TestResolveExamples "foo"→Canonical; restated for clarity.)
	got2, err := Resolve("foo", exampleSkills)
	if err != nil || got2.Match != Canonical {
		t.Errorf("Resolve(foo): match=%v err=%v; want Canonical (step 1 beats step 2)", got2.Match, err)
	}
}

// TestResolveEmptyTagGuard: a skill with Name=="" (no frontmatter) must NOT match
// step 3, so a degenerate empty tag ("") yields *UnknownError, not a Name hit.
// Also: a skill whose only alias is "" never matches.
func TestResolveEmptyTagGuard(t *testing.T) {
	skills := []discover.Skill{
		{RelTag: "nofm", Name: ""}, // no frontmatter → Name empty
	}
	res, err := Resolve("", skills)
	if err == nil {
		t.Fatalf("Resolve(\"\"): err=nil res=%+v; want *UnknownError (empty Name must not match)", res)
	}
	if _, ok := err.(*UnknownError); !ok {
		t.Fatalf("Resolve(\"\"): err type=%T; want *UnknownError", err)
	}

	// Sanity: a NON-empty tag still resolves normally on the same fixture by basename.
	if _, err := Resolve("nofm", skills); err != nil {
		t.Errorf("Resolve(nofm): err=%v; want nil (basename match)", err)
	}
}

// TestResolveDuplicateAliasCountedOnce: a skill whose Aliases lists the same tag
// twice still counts as ONE match (collectMatches appends each skill at most once),
// so a single such skill resolves cleanly rather than being misread as ambiguous.
func TestResolveDuplicateAliasCountedOnce(t *testing.T) {
	skills := []discover.Skill{
		{RelTag: "alpha", Aliases: []string{"dup", "dup"}},
	}
	got, err := Resolve("dup", skills)
	if err != nil {
		t.Fatalf("Resolve(dup): err=%v; want nil (duplicate alias counts once)", err)
	}
	if got.Match != Alias || got.Skill.RelTag != "alpha" {
		t.Errorf("Resolve(dup): match=%v rel=%q; want Alias/alpha", got.Match, got.Skill.RelTag)
	}
}

// TestResolveErrorsAs: the typed errors must be extractable via errors.As — the
// contract main (P1.M3.T8.S1) relies on to branch on error type and read Candidates.
func TestResolveErrorsAs(t *testing.T) {
	ambig := []discover.Skill{
		{RelTag: "writing/reddit"},
		{RelTag: "coding/reddit"},
	}
	_, err := Resolve("reddit", ambig)

	var ae *AmbiguousError
	if !errors.As(err, &ae) {
		t.Fatalf("errors.As(*AmbiguousError)=false for %T; want true", err)
	}
	if ae.Tag != "reddit" || len(ae.Candidates) != 2 {
		t.Errorf("extracted AmbiguousError=%+v; want Tag=reddit, 2 candidates", ae)
	}

	_, err = Resolve("nope", exampleSkills)
	var ue *UnknownError
	if !errors.As(err, &ue) {
		t.Fatalf("errors.As(*UnknownError)=false for %T; want true", err)
	}
	if ue.Tag != "nope" {
		t.Errorf("extracted UnknownError.Tag=%q; want nope", ue.Tag)
	}

	// Negative: an UnknownError must NOT masquerade as an AmbiguousError.
	_, err = Resolve("nope", exampleSkills)
	var wrong *AmbiguousError
	if errors.As(err, &wrong) {
		t.Error("errors.As(*AmbiguousError)=true on an UnknownError; want false")
	}
}

// TestErrorMessages: exact .Error() text (we own the format strings). main may
// reformat for §6.4, but the package's own rendering must be stable.
func TestErrorMessages(t *testing.T) {
	if got := (&UnknownError{Tag: "foo"}).Error(); got != `unknown skill tag "foo"` {
		t.Errorf("UnknownError.Error()=%q; want %q", got, `unknown skill tag "foo"`)
	}
	got := (&AmbiguousError{Tag: "reddit", Candidates: []string{"coding/reddit", "writing/reddit"}}).Error()
	want := `ambiguous skill tag "reddit" matches: coding/reddit, writing/reddit`
	if got != want {
		t.Errorf("AmbiguousError.Error()=%q; want %q", got, want)
	}
}

// TestMatchKindString mirrors skillsdir's TestSourceString: each constant renders,
// and an out-of-range value renders as "unknown".
func TestMatchKindString(t *testing.T) {
	cases := []struct {
		m    MatchKind
		want string
	}{
		{Canonical, "canonical"},
		{Basename, "basename"},
		{Name, "name"},
		{Alias, "alias"},
		{MatchKind(-1), "unknown"},
		{MatchKind(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("MatchKind(%d).String()=%q; want %q", c.m, got, c.want)
		}
	}
}

// TestBasename: the private slash-split helper (covers the no-slash and
// multi-level cases directly, independent of Resolve).
func TestBasename(t *testing.T) {
	cases := []struct{ relTag, want string }{
		{"writing/reddit", "reddit"},
		{"foo", "foo"}, // no slash → whole string
		{"a/b/c", "c"}, // multi-level → last
		{"", ""},       // degenerate
	}
	for _, c := range cases {
		if got := basename(c.relTag); got != c.want {
			t.Errorf("basename(%q)=%q; want %q", c.relTag, got, c.want)
		}
	}
}
```

> **Copy-paste correctness:** both blueprint files are gofmt-clean and compile
> as-is once `discover.Skill` exists (resolve.go imports fmt/sort/strings/discover;
> resolve_test.go imports errors/testing/discover — exactly what each file uses,
> nothing dead). Write them verbatim. The algorithm traces directly to PRD §7.2 +
> go_architecture.md; every assertion maps to a verified_facts entry.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0 (GATE): CONFIRM discover.Skill EXISTS before starting
  - COMMAND: grep -q 'type Skill struct' internal/discover/discover.go
  - EXPECT: match (P1.M2.T4.S2 landed the Skill type). If it does NOT exist, STOP:
            this subtask cannot compile until S2 is done. (M3 runs after M2, so it
            will exist at implementation time.)

Task 1: CREATE internal/resolve/resolve.go
  - WRITE: the exact content from the Blueprint (File 1).
  - CHECK: `package resolve`; `type MatchKind int` + Canonical/Basename/Name/Alias
           (iota in that order) + String(); `type Result struct{ Skill; Match }`;
           `type UnknownError struct{ Tag }` + pointer-receiver Error();
           `type AmbiguousError struct{ Tag; Candidates }` + pointer-receiver Error();
           `func Resolve(tag string, skills []discover.Skill) (Result, error)`;
           helpers basename/collectMatches/sortedRelTags (private).
  - CHECK imports == fmt, sort, strings, github.com/dabstractor/skpp/internal/discover.
  - GOTCHA: step 1 returns on first RelTag hit (canonical, ≤1). Steps 2-4 use
            collectMatches; >1 ⇒ AmbiguousError (short-circuit). Step 3 guards
            Name != "". basename via strings.LastIndex. Candidates sorted. Pointer
            receivers. Do NOT define Skill/Index.

Task 2: CREATE internal/resolve/resolve_test.go
  - WRITE: the exact content from the Blueprint (File 2).
  - CHECK: `package resolve` (white-box); the 9 tests (examples table, ambiguous
           x3, unknown, precedence, empty-tag-guard, duplicate-alias, errors.As,
           error-messages, MatchKind-string, basename).
  - CHECK imports == errors, testing, github.com/dabstractor/skpp/internal/discover.
  - GOTCHA: NO testify; NO t.Parallel() (repo convention). Fixtures are
            discover.Skill literals (no disk). Ambiguous fixtures passed REVERSE-
            sorted to prove the Candidate sort. Compare Candidates by length +
            indexed elements (no reflect).

Task 3: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/resolve/resolve.go internal/resolve/resolve_test.go
  - COMMAND: gofmt -l internal/resolve/*.go   # MUST print nothing
  - COMMAND: go vet ./internal/resolve/       # MUST be clean
  - COMMAND: go build ./...                    # exit 0 (whole module compiles)
  - COMMAND: go test ./internal/resolve/ -v   # ALL resolve tests PASS
  - COMMAND: go test ./...                     # whole module green
  - COMMAND: go mod tidy && git diff --quiet go.mod go.sum   # go.mod/go.sum UNCHANGED
  - EXPECT: zero errors, zero vet findings, gofmt silent, all tests pass, no go.mod change.

Task 4: ACCEPTANCE SMOKE TEST (Level 3 in Validation Loop)
  - COMMAND: the Level 3 block below (a throwaway main that builds exampleSkills,
             resolves the four §7.2 tags, and asserts the MatchKind + RelTag).
  - EXPECT: "PRD §7.2 EXAMPLES OK" printed.

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: resolve.go has MatchKind/Result/errors/Resolve/helpers only; imports
            correct; go.mod/go.sum unchanged; no Skill/Index in resolve; nothing
            else touched.
```

### Implementation Patterns & Key Details

```go
// PATTERN: step 1 inlined (exact + unique); steps 2-4 share collectMatches.
//   for _, s := range skills { if s.RelTag == tag { return Result{s, Canonical}, nil } }
//   if m := collectMatches(skills, pred); len(m) == 1 { return Result{m[0], kind}, nil
//   } else if len(m) > 1 { return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)} }
// WHY: step 1 can match at most one skill (RelTag is unique per directory), so it
//      has no ambiguity case and is simplest inlined. Steps 2-4 share the exact
//      "collect, then 1=>hit / >1=>ambiguous" shape, so collectMatches DRYs the
//      loop (mirrors skillsdir's factored-helper style). The MatchKind is fixed
//      per step, so it is passed to the Result constructor at the call site.

// PATTERN: ambiguity short-circuits (later steps NOT consulted).
//   if len(m) > 1 { return Result{}, &AmbiguousError{...} }   // return, do not fall through
// WHY: PRD §7.2 — a later step is consulted ONLY if earlier produced NOTHING. An
//      ambiguity IS a result (an error), so it stops the chain. A looser match
//      must never mask an ambiguity (the caller needs the candidates, §6.4).

// PATTERN: pointer-receiver error types + errors.As.
//   type AmbiguousError struct{ Tag string; Candidates []string }
//   func (e *AmbiguousError) Error() string { ... }
//   return &AmbiguousError{Tag: tag, Candidates: ...}
//   // caller: var ae *AmbiguousError; errors.As(err, &ae)
// WHY: stdlib convention (*url.Error, *json.SyntaxError). *T implements error so
//      errors.As extracts it (the contract main relies on). Value types would also
//      work but pointer is standard for data-carrying errors and avoids copying
//      the Candidates slice.

// PATTERN: sorted Candidates for deterministic stderr.
//   func sortedRelTags(skills) []string { tags := ...; sort.Strings(tags); return tags }
// WHY: PRD §6.4 wants stable candidate listing. Index() sorts by RelTag, so
//      iteration order is already sorted in practice — but sorting here makes the
//      error deterministic EVEN IF a caller passes unsorted input (defensive), and
//      lets tests pass reverse-sorted fixtures to prove it.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/resolve/resolve.go is `package resolve` (internal → unimportable
    outside the module; correct for a CLI's private packages).
  - imports: fmt, sort, strings, github.com/dabstractor/skpp/internal/discover.
  - exposes: MatchKind (+ constants + String), Result, UnknownError, AmbiguousError,
    Resolve. (basename/collectMatches/sortedRelTags are private.)
  - consumes: discover.Skill (the TYPE only — fields RelTag/Name/Aliases).

go.mod / go.sum (NO change — verified_facts §7):
  - resolve imports ONLY stdlib + the INTERNAL discover package. yaml.v3 is already
    a direct dependency (flipped by P1.M2.T4.S1). `go mod tidy` is a no-op.
  - VERIFY: `go mod tidy && git diff --quiet go.mod go.sum` exits 0.

DOWNSTREAM CONSUMERS (what later subtasks plug into):
  - P1.M3.T8.S1 (main tag-resolution output): calls discover.Index() once, then
    resolve.Resolve(tag, index) per <tag> arg. On *AmbiguousError/*UnknownError it
    prints one stderr line per problem tag (using .Error() or reformatting per
    §6.4), prints NOTHING to stdout, exits 1. Uses errors.As(err, &ae) to read
    ae.Candidates for the "list candidate full tags" requirement.
  - P1.M2.T6.S1 (--list): does NOT call Resolve (it lists all skills). May use
    MatchKind annotations in debug/verbose modes (optional).
  - P1.M4.T10.S1 (check): does NOT call Resolve directly (it validates, not
    resolves), but consumes the same discover.Skill index.

NO CHANGES TO:
  - go.mod / go.sum (go.mod-neutral — the whole point; verify with git diff)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned)
  - internal/discover/* (M2-owned; resolve only IMPORTS it)
  - internal/skillsdir/* (M1-owned; not imported here)
  - main.go / main_test.go (M1.T3-owned; T8 wires Resolve later, not here)
  - any other package or file (ui is a later milestone; skills/ is P1.M6.T12)
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass)
gofmt -w internal/resolve/resolve.go internal/resolve/resolve_test.go
test -z "$(gofmt -l internal/resolve/resolve.go internal/resolve/resolve_test.go)" \
  || { echo "FAIL: gofmt found unformatted files"; gofmt -d internal/resolve/; exit 1; }
echo "gofmt OK"

# Vet the new package
go vet ./internal/resolve/ || { echo "FAIL: go vet ./internal/resolve/"; exit 1; }
echo "go vet OK"

# Build the whole module (compile check across packages — requires discover.Skill)
go build ./... || { echo "FAIL: go build ./... (is discover.Skill defined yet?)"; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run resolve tests verbosely — all subtests
go test ./internal/resolve/ -v || { echo "FAIL: go test ./internal/resolve/ -v"; exit 1; }

# Targeted: the load-bearing §7.2 examples + ambiguity + errors.As
go test ./internal/resolve/ -run \
  'TestResolveExamples|TestResolveAmbiguous|TestResolveUnknown|TestResolvePrecedence|TestResolveErrorsAs' -v \
  || { echo "FAIL: load-bearing resolve tests"; exit 1; }

# Whole module still green (skillsdir + discover + resolve)
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Acceptance smoke test (resolve the four PRD §7.2 tags end-to-end)

This proves Resolve returns the exact MatchKind + RelTag for each §7.2 example row,
using the example fixture inlined from the PRD (no skills/ tree needed — Resolve is
pure). It builds a throwaway main under the module so it can import the internal
package, then cleans up.

```bash
cd /home/dustin/projects/skpp

mkdir -p cmd/probe
cat > cmd/probe/main.go <<'GO'
package main

import (
	"fmt"
	"os"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
)

func main() {
	skills := []discover.Skill{
		{Dir: "/repo/skills/foo", RelTag: "foo", Name: "foo-helper", SourceFile: "/repo/skills/foo/SKILL.md"},
		{Dir: "/repo/skills/writing/reddit", RelTag: "writing/reddit", Name: "reddit-poster", Aliases: []string{"social"}, SourceFile: "/repo/skills/writing/reddit/SKILL.md"},
	}
	for _, tag := range []string{"foo", "writing/reddit", "reddit", "foo-helper"} {
		r, err := resolve.Resolve(tag, skills)
		if err != nil { fmt.Println("ERR", tag, err); os.Exit(1) }
		fmt.Printf("%-15s -> %-15s %s\n", tag, r.Skill.RelTag, r.Match)
	}
}
GO

out="$(go run ./cmd/probe)"
rc=$?
rm -rf cmd   # throwaway — NOT part of the deliverable
echo "$out"
[ $rc -eq 0 ] || { echo "FAIL: probe exited $rc"; exit 1; }

# Assert the four §7.2 example rows exactly (tag -> relTag matchKind).
echo "$out" | grep -Eq '^foo[[:space:]]+->[[:space:]]+foo[[:space:]]+canonical$'         || { echo "FAIL: foo -> canonical"; exit 1; }
echo "$out" | grep -Eq '^writing/reddit[[:space:]]+->[[:space:]]+writing/reddit[[:space:]]+canonical$' || { echo "FAIL: writing/reddit -> canonical"; exit 1; }
echo "$out" | grep -Eq '^reddit[[:space:]]+->[[:space:]]+writing/reddit[[:space:]]+basename$' || { echo "FAIL: reddit -> basename"; exit 1; }
echo "$out" | grep -Eq '^foo-helper[[:space:]]+->[[:space:]]+foo[[:space:]]+name$'        || { echo "FAIL: foo-helper -> name"; exit 1; }

echo "PRD §7.2 EXAMPLES OK"
echo "Level 3 PASS"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# resolve.go exists and is package resolve
test -f internal/resolve/resolve.go || { echo "FAIL: resolve.go missing"; exit 1; }
grep -q '^package resolve' internal/resolve/resolve.go || { echo "FAIL: not package resolve"; exit 1; }

# MatchKind + constants + String
grep -q 'type MatchKind int' internal/resolve/resolve.go || { echo "FAIL: MatchKind"; exit 1; }
for c in Canonical Basename Name Alias; do
  grep -qE "^\s*$c MatchKind" internal/resolve/resolve.go || { echo "FAIL: constant $c"; exit 1; }
done
grep -qE 'func \(m MatchKind\) String\(\)' internal/resolve/resolve.go || { echo "FAIL: MatchKind.String"; exit 1; }

# Result struct
grep -qE 'type Result struct' internal/resolve/resolve.go || { echo "FAIL: Result"; exit 1; }
grep -q 'Skill discover.Skill' internal/resolve/resolve.go || { echo "FAIL: Result.Skill"; exit 1; }
grep -q 'Match MatchKind' internal/resolve/resolve.go || { echo "FAIL: Result.Match"; exit 1; }

# Error types: pointer-receiver Error()
grep -q 'type UnknownError struct' internal/resolve/resolve.go || { echo "FAIL: UnknownError"; exit 1; }
grep -q 'type AmbiguousError struct' internal/resolve/resolve.go || { echo "FAIL: AmbiguousError"; exit 1; }
grep -q 'Candidates \[\]string' internal/resolve/resolve.go || { echo "FAIL: AmbiguousError.Candidates"; exit 1; }
grep -qE 'func \(e \*UnknownError\) Error\(\)' internal/resolve/resolve.go || { echo "FAIL: *UnknownError.Error (pointer receiver)"; exit 1; }
grep -qE 'func \(e \*AmbiguousError\) Error\(\)' internal/resolve/resolve.go || { echo "FAIL: *AmbiguousError.Error (pointer receiver)"; exit 1; }
# value receivers must NOT exist
! grep -qE 'func \(e UnknownError\) Error\(\)' internal/resolve/resolve.go || { echo "FAIL: must use POINTER receiver for UnknownError"; exit 1; }

# Resolve signature
grep -qE 'func Resolve\(tag string, skills \[\]discover.Skill\) \(Result, error\)' internal/resolve/resolve.go \
  || { echo "FAIL: Resolve signature"; exit 1; }

# basename via strings.LastIndex (NOT path.Base); Candidates sorted
grep -q 'strings.LastIndex' internal/resolve/resolve.go || { echo "FAIL: basename must use strings.LastIndex"; exit 1; }
! grep -q 'path.Base' internal/resolve/resolve.go || { echo "FAIL: do not use path.Base"; exit 1; }
grep -q 'sort.Strings' internal/resolve/resolve.go || { echo "FAIL: Candidates must be sorted (sort.Strings)"; exit 1; }

# Imports are EXACTLY fmt, sort, strings, discover
imp="$(sed -n '/^import (/,/^)/p' internal/resolve/resolve.go)"
for want in fmt sort strings github.com/dabstractor/skpp/internal/discover; do
  echo "$imp" | grep -q "\"$want\"" || { echo "FAIL: missing import $want"; exit 1; }
done
for ban in '"path"' '"errors"' '"reflect"' '"path/filepath"' '"os"'; do
  echo "$imp" | grep -q "$ban" && { echo "FAIL: forbidden import $ban in resolve.go"; exit 1; } || true
done

# SCOPE: NO Skill type, NO Index, NO ParseFrontmatter in resolve.go (those are discover's)
! grep -qE 'type Skill struct' internal/resolve/resolve.go || { echo "FAIL: Skill type must NOT exist (discover owns it)"; exit 1; }
! grep -qE 'func Index' internal/resolve/resolve.go || { echo "FAIL: Index() must NOT exist (discover T5)"; exit 1; }

# resolve_test.go is white-box package resolve with no Parallel/testify
test -f internal/resolve/resolve_test.go || { echo "FAIL: resolve_test.go missing"; exit 1; }
grep -q '^package resolve' internal/resolve/resolve_test.go || { echo "FAIL: resolve_test.go must be package resolve (white-box)"; exit 1; }
! grep -q 't.Parallel' internal/resolve/resolve_test.go || { echo "FAIL: no t.Parallel() (repo convention)"; exit 1; }
! grep -q 'testify' internal/resolve/resolve_test.go || { echo "FAIL: no testify"; exit 1; }
# test imports are exactly errors, testing, discover
timp="$(sed -n '/^import (/,/^)/p' internal/resolve/resolve_test.go)"
for want in errors testing github.com/dabstractor/skpp/internal/discover; do
  echo "$timp" | grep -q "\"$want\"" || { echo "FAIL: test missing import $want"; exit 1; }
done
for ban in '"reflect"' '"strings"' '"fmt"'; do
  echo "$timp" | grep -q "$ban" && { echo "FAIL: forbidden test import $ban"; exit 1; } || true
done
# key tests present
for tn in TestResolveExamples TestResolveAmbiguous TestResolveUnknown TestResolvePrecedence TestResolveEmptyTagGuard TestResolveErrorsAs TestErrorMessages TestMatchKindString; do
  grep -q "func $tn" internal/resolve/resolve_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# go.mod / go.sum UNCHANGED (go.mod-neutral — the defining property of this subtask)
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null || { echo "FAIL: go.mod/go.sum changed (resolve must be dependency-neutral)"; git diff go.mod go.sum; exit 1; }

# MUST NOT have touched PRD.md / discover / skillsdir / main (owned by other subtasks)
git diff --quiet PRD.md 2>/dev/null || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
git diff --quiet internal/discover/discover.go 2>/dev/null   || { echo "FAIL: discover.go changed (M2-owned)"; exit 1; }
git diff --quiet internal/discover/discover_test.go 2>/dev/null || { echo "FAIL: discover_test.go changed"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go 2>/dev/null  || { echo "FAIL: skillsdir.go changed"; exit 1; }
git diff --quiet main.go main_test.go 2>/dev/null || { echo "FAIL: main.go/main_test.go modified"; exit 1; }

# MUST NOT have created later-milestone files/packages
test ! -d internal/ui || { echo "FAIL: ui/ must not exist (M2.T6)"; exit 1; }
test ! -d cmd         || { echo "FAIL: cmd/ (probe) must be removed (throwaway)"; exit 1; }
test ! -f install.sh  || { echo "FAIL: install.sh must not exist (M6)"; exit 1; }
test ! -f README.md   || { echo "FAIL: README.md must not exist (M6)"; exit 1; }
test ! -d skills      || { echo "FAIL: skills/ must not exist (P1.M6.T12)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l internal/resolve/*.go` silent, `go vet ./internal/resolve/` clean, `go build ./...` exit 0
- [ ] Level 2 PASS — `go test ./internal/resolve/ -v` all tests pass; `go test ./...` whole module green
- [ ] Level 3 PASS — the four PRD §7.2 tags resolve to the exact (relTag, MatchKind): foo→canonical, writing/reddit→canonical, reddit→basename, foo-helper→name
- [ ] Level 4 PASS — types/signatures correct; pointer-receiver errors; basename via LastIndex; sorted Candidates; imports exact; go.mod/go.sum unchanged; nothing else touched

### Feature Validation
- [ ] PRD §7.2 examples table passes exactly (the item's required test)
- [ ] Ambiguous basename/name/alias each return `*AmbiguousError` with sorted `Candidates` and the input `tag`
- [ ] Unknown tag AND empty/nil `skills` both return `*UnknownError{Tag: tag}`
- [ ] First-match-wins: canonical beats basename and name (precedence respected)
- [ ] Empty-name guard: a `Name==""` skill never matches step 3 (empty tag ⇒ unknown)
- [ ] Duplicate alias within one skill counts once (no false ambiguity)
- [ ] `*AmbiguousError`/`*UnknownError` extractable via `errors.As` (main's contract)
- [ ] Error `.Error()` text is stable; `MatchKind.String()` renders all 4 + "unknown"

### Code Quality / Convention Validation
- [ ] `resolve.go` is `package resolve` in `internal/resolve/`; imports limited to fmt/sort/strings/discover
- [ ] `resolve_test.go` is white-box `package resolve`, mirroring `skillsdir_test.go`/`discover_test.go` (plain t.Errorf/t.Fatalf, no testify, no t.Parallel, struct-literal fixtures)
- [ ] Every exported type/function is documented in godoc (MatchKind constants, Result, the error types, Resolve's precedence list, the helpers)

### Scope Discipline
- [ ] Did NOT define `type Skill`, `Index`, `ParseFrontmatter`, or `toStringSlice` (discover owns those)
- [ ] Did NOT modify `go.mod`/`go.sum` (go.mod-neutral), `PRD.md`, any `tasks.json`, `internal/discover/*`, `internal/skillsdir/*`, or `main.go`/`main_test.go`
- [ ] Did NOT create `ui`/`cmd`/`install.sh`/`README.md`/`skills/`
- [ ] Confirmed `discover.Skill` exists before implementing (Task 0 gate)

---

## Anti-Patterns to Avoid

- ❌ **Don't implement before `discover.Skill` exists.** resolve imports it; the
  build fails with "undefined: discover.Skill" until P1.M2.T4.S2 lands. (verified_facts §0.)
  M3 runs after M2, so it will exist — but run the Task 0 gate.
- ❌ **Don't use value receivers for the error types.** Pointer receivers are the
  stdlib convention and are REQUIRED for `errors.As` to be the natural extraction
  path in main. Return `&UnknownError{}`/`&AmbiguousError{}`. (verified_facts §5.)
- ❌ **Don't let ambiguity fall through.** An ambiguous step returns
  `*AmbiguousError` IMMEDIATELY; later steps are NOT consulted. A looser match must
  never mask an ambiguity — the caller needs the candidates (PRD §6.4).
- ❌ **Don't check later steps before earlier ones.** Canonical beats basename beats
  name beats alias; first match wins. Step 1 returns on its single hit even if step 2
  would also match (e.g. `foo` is both the exact RelTag and its own basename).
- ❌ **Don't use `path.Base` for basename.** Use `strings.LastIndex(relTag, "/")`.
  The item says "split on /, take last element"; RelTag is already slash-normalized;
  and LastIndex avoids importing `path` (we already import `strings`).
- ❌ **Don't leave `Candidates` unsorted.** `sort.Strings` makes the error
  deterministic regardless of input order (PRD §6.4 wants stable stderr). Tests pass
  reverse-sorted fixtures to prove it.
- ❌ **Don't match `Name==""` in step 3.** A skill with no frontmatter has an empty
  Name; it is not matchable by name. Guard with `s.Name != "" && s.Name == tag`.
  This also prevents a degenerate empty tag from spuriously resolving.
- ❌ **Don't double-count a skill.** `collectMatches` appends each skill at most once
  per step (the predicate is a boolean property). A skill with a duplicate alias
  still counts once.
- ❌ **Don't add a "skpp:" prefix to error messages.** Match `skillsdir.ErrNotFound`
  (prefix-free; main adds program context). The test asserts exact text, so keep the
  format strings as specified.
- ❌ **Don't change `go.mod`/`go.sum`.** resolve imports only stdlib + internal
  `discover`; yaml.v3 is already direct. If `go mod tidy` changes anything, you
  imported something external you should not have. (verified_facts §7.)
- ❌ **Don't use `reflect`, `testify`, or `t.Parallel()` in tests.** Mirror the repo
  convention: white-box same-package, plain assertions, manual Candidate comparison.
- ❌ **Don't define `Skill`/`Index` in resolve, or wire Resolve into `main.go`.**
  Those are discover's and T8's deliverables. resolve only consumes `discover.Skill`
  and exposes `Resolve`.

---

## Confidence Score

**9/10** — one-pass implementation success likelihood.

Rationale: the exact `resolve.go` and `resolve_test.go` source is provided verbatim
(gofmt-clean, compiles as-is once `discover.Skill` exists). The algorithm is pure
stdlib Go over a type contract locked in `go_architecture.md`; every assertion maps
to PRD §7.2 + verified_facts. The package is `go.mod`-neutral (no external dep). The
single hard gate is the `discover.Skill` dependency (P1.M2.T4.S2) — mitigated by the
Task 0 check and by M3 being scheduled after M2. The only residual risk is the Level 3
probe's `grep -E` on the aligned output being whitespace-sensitive — mitigated by the
`[[:space:]]+` patterns and the cleanup of the throwaway `cmd/` dir.
