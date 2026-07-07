# Verified facts — P1.M3.T7.S1 (`internal/resolve`)

> Output path uses the orchestrator directory `P1M3T1S1`; this is plan item
> **P1.M3.T7.S1** ("Resolve() + Result/MatchKind + Unknown/Ambiguous errors"),
> the first (only) subtask of T7 (§7.2 precedence resolver). M3 depends on M2.

## 0. Scope & dependencies (the contract this PRP consumes)

- **Deliverable:** `internal/resolve/resolve.go` (`package resolve`) + white-box
  `internal/resolve/resolve_test.go`. Nothing else.
- **`Resolve` is PURE:** no filesystem, no global state, no I/O. It takes the
  already-built index `[]discover.Skill` as a parameter (PRD §7.2; go_architecture
  `Resolve(tag string, skills []discover.Skill) (Result, error)`). main (P1.M3.T8.S1)
  calls `discover.Index()` then `resolve.Resolve` per tag.
- **Hard dependency:** `discover.Skill`. That struct is **NOT yet implemented** at
  research time — it is P1.M2.T4.S2's deliverable ("metadata extraction + Skill
  type"). Its field shape is LOCKED in `go_architecture.md` ("Core types >
  internal/discover") and re-stated in the P1.M2.T4.S1 PRP's "DOWNSTREAM EXTENSION
  POINTS". Because Resolve takes `[]discover.Skill` as a **parameter**, the resolve
  tests build `discover.Skill` literals directly and need **neither Index() nor a
  skills/ tree** — only the Skill TYPE (S2) must exist. M3 runs after M2, so S2/T5
  are complete when this PRP is implemented.

## 1. The `discover.Skill` field shape resolve consumes (LOCKED — go_architecture.md)

```go
type Skill struct {
    Dir         string   // absolute path of the skill directory     (unused by resolve)
    RelTag      string   // slash-normalized canonical tag             (steps 1 & 2)
    Name        string   // frontmatter name, "" if missing            (step 3)
    Description string   // unused by resolve
    Keywords    []string // unused by resolve
    Category    string   // unused by resolve
    Aliases     []string // metadata.aliases, nil if absent             (step 4)
    HasFM       bool     // unused by resolve
    SourceFile  string   // unused by resolve
}
```
Resolve touches ONLY: `RelTag` (step 1 exact, step 2 basename), `Name` (step 3),
`Aliases` (step 4). `RelTag` is **already slash-normalized** by discover
(`filepath.ToSlash`), so splitting on `/` is correct on every OS (no need for
`path/filepath` or OS-separator handling in resolve).

## 2. The §7.2 precedence — first match wins; later steps consulted ONLY if earlier produced nothing

1. **Canonical** — `skill.RelTag == tag` (case-sensitive). RelTag is unique per
   directory ⇒ **at most one** hit. Single hit ⇒ return Canonical. (No ambiguity
   case is possible here.)
2. **Basename** — `basename(skill.RelTag) == tag` where basename = last `/`-component.
   Exactly 1 ⇒ return Basename. **>1 ⇒ AmbiguousError** (candidates = the matching
   RelTags).
3. **Name** — `skill.Name == tag`. 1 ⇒ Name; >1 ⇒ Ambiguous.
4. **Alias** — `tag` in `skill.Aliases`. 1 ⇒ Alias; >1 ⇒ Ambiguous.
5. nothing ⇒ **UnknownError**.

**Ambiguity short-circuits:** an ambiguous step returns AmbiguousError IMMEDIATELY;
later steps are NOT consulted. A looser match cannot "rescue" an ambiguity — the
caller must see the ambiguous error (PRD §6.4: list candidates, exit 1).

## 3. PRD §7.2 examples (the table resolve tests EXACTLY)

Setup (PRD §7.2): `skills/foo/SKILL.md` with `name: foo-helper`, and
`skills/writing/reddit/SKILL.md`. Verified trace:

| tag             | step1 RelTag? | step2 basename? | step3 name? | result                  |
|-----------------|---------------|-----------------|-------------|-------------------------|
| `foo`           | ✓ "foo"       | (skipped)       | —           | Canonical → foo         |
| `writing/reddit`| ✓             | (skipped)       | —           | Canonical → writing/reddit |
| `reddit`        | ✗             | ✓ 1 match       | —           | Basename → writing/reddit |
| `foo-helper`    | ✗             | ✗ (foo,reddit)  | ✓ 1 match   | Name → foo              |

Note: step 1 wins over step 2 for `foo` even though basename("foo")=="foo" would
also match — that is the whole point of "first match wins".

## 4. The type shape (LOCKED — go_architecture.md + item CONTRACT)

```go
type MatchKind int
const ( Canonical MatchKind = iota; Basename; Name; Alias )

type Result struct { Skill discover.Skill; Match MatchKind }

func Resolve(tag string, skills []discover.Skill) (Result, error)

type UnknownError    struct{ Tag string }
type AmbiguousError  struct{ Tag string; Candidates []string }
```
Both error types implement `error` via `.Error()`. The architecture comment names
them loosely "ErrUnknown/ErrAmbiguous"; the item CONTRACT is authoritative: the
named struct types above. Resolve returns them by pointer (`&UnknownError{...}`).

## 5. Design decisions (defensible, documented as gotchas in the PRP)

- **Pointer receivers** for the error types (`func (e *UnknownError) Error()`).
  Stdlib convention for data-carrying errors (`*url.Error`, `*json.SyntaxError`).
  Enables `errors.As(err, &target)` (which main, P1.M3.T8.S1, will use) and avoids
  copying the `Candidates` slice. The VALUE type does not implement `error` — only
  the pointer does. This is intended.
- **`MatchKind.String()`** added (canonical/basename/name/alias/unknown) mirroring
  `skillsdir.Source.String()` (the established codebase convention for an int enum).
  Out-of-range ⇒ "unknown". Low-cost, idiomatic, aids --list/debug.
- **`basename` via `strings.LastIndex(relTag, "/")`** (not `path.Base`). The item
  CONTRACT says "split on /, take last element"; LastIndex is the zero-alloc
  equivalent and avoids importing `path` (we already import `strings`). For a clean
  slash-normalized RelTag, LastIndex and path.Base are identical; LastIndex is
  faithful to the spec wording.
- **Candidates are SORTED (`sort.Strings`)** for deterministic stderr regardless of
  the input slice ordering. PRD §6.4 wants stable candidate listing for scripting;
  sorting makes `AmbiguousError.Candidates` order-independent of how the caller
  built the index. Tests pass input in REVERSE sorted order to prove the sort.
- **Empty-field guard on steps 3 & 4:** step 3 only counts a skill whose `Name !=
  ""`; an empty alias never matches. A skill with NO frontmatter (Name=="") is not
  matchable by name, and a degenerate empty tag (`""`) cannot spuriously resolve to
  a missing-name skill. RelTag and its basename are always non-empty for a real
  skill, so steps 1–2 need no guard.

## 6. Error message format (sensible defaults; main owns final §6.4 stderr)

- `UnknownError.Error()`  = `unknown skill tag %q`            → ``unknown skill tag "foo"``
- `AmbiguousError.Error()` = `ambiguous skill tag %q matches: %s` with
  `strings.Join(Candidates, ", ")` →
  ``ambiguous skill tag "reddit" matches: coding/reddit, writing/reddit``
- NO `skpp:` prefix (matches `skillsdir.ErrNotFound`, which is prefix-free — main
  adds program context). main (P1.M3.T8.S1) may use these verbatim or reformat per
  §6.4; the contract under test is the TYPE + FIELDS, not the exact prose.

## 7. `go.mod` / `go.sum` are UNCHANGED by this subtask

resolve imports ONLY the stdlib (`fmt`, `sort`, `strings`) + the INTERNAL
`discover` package. It imports NO new external module: yaml.v3 is already a direct
dependency (flipped `// indirect`→direct by P1.M2.T4.S1's `go mod tidy`). Therefore
`go mod tidy` is a no-op here and `go.mod`/`go.sum` are byte-identical before/after.
(This is the clean contrast to S1, whose single go.mod line was expected.)

## 8. Test convention (matches skillsdir_test.go / discover_test.go)

- `package resolve` (WHITE-BOX, same package) — mirrors `package skillsdir` and
  `package discover`.
- Plain `t.Errorf` / `t.Fatalf`. **NO testify. NO `t.Parallel()`** (repo convention
  is no-Parallel across the board; resolve tests touch no env/cwd anyway).
- Fixtures are `discover.Skill` struct literals (no `t.TempDir`, no files — resolve
  is pure, so no disk fixtures are needed).
- Compare `Candidates` by length + indexed elements (explicit, no `reflect` import)
  — matches discover_test.go's manual `kw[0]`/`kw[1]` style.
- Imports: `errors` (for the `errors.As` contract test), `testing`,
  `github.com/dabstractor/skpp/internal/discover`. Nothing else.

## 9. Scope boundary — what resolve does NOT do

- No `Index()` (P1.M2.T5.S1), no `Skill`/`toStringSlice` (P1.M2.T4.S2), no
  frontmatter parsing (P1.M2.T4.S1).
- No main wiring / path printing / exit codes (P1.M3.T8.S1 owns the §6.4 atomic
  "resolve all, then print nothing-on-failure" contract).
- No `--list`/`--search`/`check` (M2.T6, M4).
- Resolve returns ONE tag's outcome; the multi-tag loop + atomic stdout discipline
  is main's job (§6.4).

## 10. Confidence

The implementation is pure stdlib Go over a locked type contract. Every behavior is
traceable to PRD §7.2 + go_architecture.md. The exact `resolve.go` and
`resolve_test.go` source is given verbatim in the PRP (gofmt-clean, compiles as-is
once `discover.Skill` exists). go.mod-neutral. Risk is low.
