# Verified facts — P1.M3.T7.S1 (`internal/resolve`)

> **Plan item:** `P1.M3.T7.S1` — "Resolve() + Result/MatchKind + Unknown/Ambiguous
> errors", the first (only) subtask of T7 (§7.2 precedence resolver). M3 (Tag
> resolution & path output) depends on M2, which is now COMPLETE.
>
> **Status change vs the prior research (dir `P1M3T1S1`):** that research was done
> when `discover.Skill` did **not** exist yet — its source was "compiles as-is once
> discover.Skill exists" but **was never compiled**. This re-plan runs AFTER M2
> landed, so the source has been **compiled, vetted, and tested** against the real
> module in a throwaway `/tmp/skpp_resolve_verify`. Every claim below is empirical
> (go1.25 / go.mod `go 1.25`). One factual correction: `type Skill struct` lives in
> `internal/discover/skill.go` (S2), **not** `discover.go` — the prior Task-0 gate
> grepped the wrong file.

## 0. Scope & dependencies (the contract this PRP consumes)

- **Deliverable:** `internal/resolve/resolve.go` (`package resolve`) + white-box
  `internal/resolve/resolve_test.go`. Nothing else.
- **`Resolve` is PURE:** no filesystem, no global state, no I/O. It takes the
  already-built index `[]discover.Skill` as a parameter (PRD §7.2; go_architecture
  `Resolve(tag string, skills []discover.Skill) (Result, error)`). main (P1.M3.T8.S1)
  calls `discover.Index()` then `resolve.Resolve` per `<tag>` arg.
- **Hard dependency:** `discover.Skill`. **It NOW EXISTS** in
  `internal/discover/skill.go` (P1.M2.T4.S2, LANDED & green). Resolve touches ONLY
  three of its fields: `RelTag` (steps 1 & 2), `Name` (step 3), `Aliases` (step 4).

## 1. The `discover.Skill` field shape resolve consumes (LOCKED — on disk)

Read directly from `internal/discover/skill.go` (LANDED, M2.T4.S2):

```go
type Skill struct {
    Dir         string   // absolute path of the skill directory    (unused by resolve)
    RelTag      string   // slash-normalized canonical tag            (steps 1 & 2)
    Name        string   // frontmatter name, "" if missing           (step 3)
    Description string   // unused by resolve
    Keywords    []string // unused by resolve
    Category    string   // unused by resolve
    Aliases     []string // metadata.aliases, nil if absent            (step 4)
    HasFM       bool     // unused by resolve
    SourceFile  string   // unused by resolve
}
```

- `RelTag` is **already slash-normalized** by discover (`filepath.ToSlash` in
  `index.go`) and every `Dir` is **absolute** (`filepath.Abs`). So splitting on `/`
  in resolve is correct on every OS, and a returned `Result.Skill.Dir` is already
  the absolute path main needs for `--skill` output (PRD §6.1/§13).
- `Aliases` is the normalized `[]string` from `toStringSlice` (S2): absent field ⇒
  `nil`; present-empty list ⇒ non-nil empty. Both have `len 0` — resolve's
  `range s.Aliases` is a no-op in both cases (correct: nothing to match).

## 2. The §7.2 precedence — first match wins; later steps consulted ONLY if earlier produced nothing

1. **Canonical** — `skill.RelTag == tag` (case-sensitive). RelTag is unique per
   directory ⇒ **at most one** hit. Single hit ⇒ return Canonical. (No ambiguity
   case is possible here.)
2. **Basename** — `basename(skill.RelTag) == tag` where basename = last `/`-component.
   Exactly 1 ⇒ return Basename. **>1 ⇒ *AmbiguousError** (candidates = matching RelTags).
3. **Name** — `skill.Name == tag`. 1 ⇒ Name; >1 ⇒ Ambiguous.
4. **Alias** — `tag` in `skill.Aliases`. 1 ⇒ Alias; >1 ⇒ Ambiguous.
5. nothing ⇒ **\*UnknownError**.

**Ambiguity short-circuits:** an ambiguous step returns `*AmbiguousError`
IMMEDIATELY; later steps are NOT consulted. A looser match cannot "rescue" an
ambiguity — the caller must see the candidates (PRD §6.4: list candidates, exit 1).

## 3. PRD §7.2 examples — verified END-TO-END via real discover.Index

The fixture was built **on disk** (`skills/foo` with `name: foo-helper`;
`skills/writing/reddit` with `aliases: [social]`), indexed by the **real**
`discover.Index`, then each tag resolved by **real** `resolve.Resolve`. Output:

```
index has 2 skills: foo writing/reddit
OK   foo             -> foo             (canonical)
OK   writing/reddit  -> writing/reddit  (canonical)
OK   reddit          -> writing/reddit  (basename)
OK   foo-helper      -> foo             (name)
OK   social          -> writing/reddit  (alias)
PRD §7.2 EXAMPLES OK (end-to-end via real discover.Index)
```

So the full `Index → Resolve` path is proven, not just struct-literal unit tests.

## 4. The type shape (LOCKED — go_architecture.md + item CONTRACT; now on disk)

```go
type MatchKind int
const ( Canonical MatchKind = iota; Basename; Name; Alias )

type Result struct { Skill discover.Skill; Match MatchKind }

func Resolve(tag string, skills []discover.Skill) (Result, error)

type UnknownError    struct{ Tag string }
type AmbiguousError  struct{ Tag string; Candidates []string }
```

Both error types implement `error` via a **pointer-receiver** `Error()`. `go doc`
confirms the public surface is exactly: `AmbiguousError`, `MatchKind`+consts,
`Result`+`Resolve`, `UnknownError`. (The architecture comment names them loosely
"ErrUnknown/ErrAmbiguous"; the item CONTRACT's struct types are authoritative.)

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

## 7. `go.mod` / `go.sum` are UNCHANGED by this subtask (VERIFIED)

resolve imports ONLY the stdlib (`fmt`, `sort`, `strings`) + the INTERNAL
`discover` package. `go mod tidy` was run in the throwaway copy: `go.mod` and
`go.sum` are byte-identical before/after (`diff` clean). The module's sole direct
dep remains `gopkg.in/yaml.v3 v3.0.1`.

## 8. Test convention (matches skillsdir_test.go / discover_test.go)

- `package resolve` (WHITE-BOX, same package) — mirrors `package skillsdir` and
  `package discover`.
- Plain `t.Errorf` / `t.Fatalf`. **NO testify. NO `t.Parallel()`** (repo convention
  is no-Parallel across the board; resolve tests touch no env/cwd anyway).
- Fixtures are `discover.Skill` struct literals (no `t.TempDir`, no files — resolve
  is pure, so no disk fixtures are needed for the UNIT tests; the end-to-end path is
  separately proven in this research via a throwaway cmd).
- Compare `Candidates` by length + indexed elements (explicit, no `reflect` import)
  — matches discover_test.go's manual style.
- Imports: `errors` (for the `errors.As` contract test), `testing`,
  `github.com/dabstractor/skpp/internal/discover`. Nothing else.

## 9. Full validation gate result (empirical, throwaway copy on go1.25)

Ran against `/tmp/skpp_resolve_verify` (a verbatim copy of the real module + the
new `internal/resolve/`):

```
gofmt -l internal/resolve/*.go          → (silent) ✓
go vet ./internal/resolve/              → clean ✓
go build ./...                          → OK ✓
go test ./internal/resolve/ -v          → PASS, 10 test funcs (incl 3 subtests) ✓
go test ./...                           → green (discover+skillsdir+ui+main+resolve) ✓
go mod tidy ; diff go.mod go.sum        → unchanged ✓
```

Existing test counts BEFORE this subtask (real repo): discover 31, skillsdir 29,
ui 11, main 24 = **95**. resolve ADDS 10 test funcs → **105** total after.

## 10. Scope boundary — what resolve does NOT do

- No `Index()` (P1.M2.T5.S1 — LANDED), no `Skill`/`toStringSlice` (P1.M2.T4.S2 —
  LANDED), no frontmatter parsing (P1.M2.T4.S1 — LANDED). resolve only CONSUMES them.
- No main wiring / path printing / exit codes (P1.M3.T8.S1 owns the §6.4 atomic
  "resolve all, then print nothing-on-failure" contract).
- No `--list`/`--search`/`check` (M2.T6 LANDED; M4 planned).
- Resolve returns ONE tag's outcome; the multi-tag loop + atomic stdout discipline
  is main's job (§6.4).

## 11. Confidence

The implementation is pure stdlib Go over a locked type contract, **now compiled
and tested against the real module** (not just asserted against a contract as in
the prior research). Every behavior is traceable to PRD §7.2 + go_architecture.md
and verified by 10 passing tests + an end-to-end real-Index smoke test.
go.mod-neutral. Risk is LOW.
