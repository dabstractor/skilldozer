# PRP — P1.M2.T4.S2: `Skill` type + metadata extraction (`toStringSlice` / `newSkill`)

> **Subtask:** P1.M2.T4.S2 (plan id `P1.M2.T1.S2`) — the SECOND subtask of T4
> (the `internal/discover` frontmatter model & parser, PRD §7.1/§7.3/§10).
> **Scope:** ADD two files to the EXISTING `internal/discover` package:
> `internal/discover/skill.go` (`package discover`) defining `type Skill struct`
> (the indexed representation), `func toStringSlice(v any) []string` (the
> metadata-list converter), and `func newSkill(absDir, skillsDir string, fm
> Frontmatter, body string, hasFM bool) Skill` (the constructor); plus its
> white-box test `internal/discover/skill_test.go`. Nothing else.
>
> **DEPENDENCY (CONTRACT — treat as already landed):** S1 (P1.M2.T1.S1) creates
> `internal/discover/discover.go` with `type Frontmatter struct` (fields incl.
> `Name`, `Description`, `Metadata map[string]any`, `HasFM bool`) and
> `func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`.
> This subtask CONSUMES those two symbols verbatim. yaml.v3 is already a direct
> dependency after S1 (`go mod tidy` already flipped `// indirect` away in S1).
>
> **NEW-FILE REQUIREMENT (do NOT edit discover.go):** S1's own Level 4 validation
> ASSERTS that `internal/discover/discover.go` contains NEITHER `type Skill` NOR
> `toStringSlice`. Adding them to `discover.go` would break S1's contract. S2
> MUST place `Skill` / `toStringSlice` / `newSkill` in a NEW file
> `internal/discover/skill.go` (tests in `skill_test.go`). This also matches
> S1's package-doc statement that `discover.go` is "the frontmatter MODEL and the
> lenient fence-slicing PARSER only... the Skill struct + metadata extraction
> land in P1.M2.T4.S2."
>
> **DOWNSTREAM CONSUMERS (do not build them, just keep the contract stable):**
> P1.M2.T5 (`Index()` walk) calls `newSkill` for every dir containing a
> `SKILL.md` and returns `[]Skill`. P1.M3.T7 (`resolve.Resolve`) and P1.M2.T6
> (`ui`) READ `Skill` fields (RelTag, Name, Description, Keywords, Category,
> Aliases, HasFM). They never construct a Skill. `newSkill` stays unexported
> (same-package caller); `Skill` is exported (consumed across packages).
>
> **PARALLEL CONTEXT:** This PRP was authored while S1's implementation was in
> flight. The PRP assumes S1 lands EXACTLY as specified in
> `plan/001_fcde63e5bb60/P1M2T1S1/PRP.md` (the Frontmatter struct shape with
> `Metadata map[string]any` + `HasFM bool`, and the `ParseFrontmatter` return
> triple). S2's own implementation runs AFTER S1 lands, so `discover.go` exists.
> S2 does NOT touch `discover.go` / `discover_test.go` / `skillsdir/*` / `main.go`.

---

## Goal

**Feature Goal**: Define the `discover.Skill` value type (the central index
element) and the metadata-extraction logic that turns a parsed `Frontmatter`
into a populated `Skill`. Specifically: a `type Skill struct` with the 9 fields
the item + architecture specify (Dir, RelTag, Name, Description, Keywords,
Category, Aliases, HasFM, SourceFile); a `toStringSlice(v any) []string` helper
that robustly converts yaml.v3-decoded metadata values (`[]any` list, bare
`string`, or nil) into `[]string` without ever panicking; and a `newSkill`
constructor that computes the slash-normalized `RelTag`, extracts keywords /
category / aliases from the `metadata` map, and threads `HasFM`.

**Deliverable**: Two NEW files in the existing `internal/discover` package (NO
other files modified, NO go.mod/go.sum change — yaml.v3 is already direct):
1. `internal/discover/skill.go` — `package discover`; `import "path/filepath"`;
   `type Skill struct` (9 fields, NO yaml tags); `func toStringSlice(v any)
   []string`; `func newSkill(absDir, skillsDir string, fm Frontmatter, body
   string, hasFM bool) Skill`.
2. `internal/discover/skill_test.go` — `package discover` (white-box); the 3
   item-specified scenarios (full metadata / bare-string keyword / no metadata)
   + edge cases (nested relTag, SourceFile/Dir, HasFM propagation, name≠dir,
   no-frontmatter-block) + direct `toStringSlice` table over all input shapes.

**Success Definition**: `go build ./internal/discover/` exits 0;
`gofmt -l internal/discover/*.go` is silent; `go vet ./internal/discover/` is
clean; `go test ./internal/discover/ -v` passes (S1's parser tests AND S2's new
skill tests); `go test ./...` whole module green; `skill.go` imports ONLY
`path/filepath`; `discover.go`/`discover_test.go`/`go.mod`/`go.sum`/`skillsdir`/
`main.go` unchanged. No `Index()` (S5), no `resolve`, no `ui`.

---

## Why

- `Skill` is the **single value type** the entire downstream pipeline is built
  around. PRD §7.1 lists the per-skill fields discovery captures
  (`dir`/`relTag`/`name`/`description`/`keywords`/`category`/`aliases`); every
  one of them lives on `Skill`. Until this lands, `Index()` (S5) has nothing to
  return, `resolve.Resolve` (S7) has nothing to match against, and `--list` /
  `--search` (S6/S9) have nothing to print.
- It **locks the `[]any`→`[]string` conversion contract** that the rest of skpp
  depends on. yaml.v3 (verified) decodes `metadata.keywords: [a,b,c]` into
  `[]interface{}` (`[]any`), NEVER `[]string`. A naive `fm.Metadata["keywords"]
  .([]string)` would PANIC on every real skill. `toStringSlice` is the one place
  that type-assertion lives, so it is correct everywhere downstream.
- It **makes keywords/aliases lenient about shape** (PRD §7.1: "list if present,
  else []"): a user may write `keywords: writing` (bare scalar) or
  `keywords: [a, b]` (list) or omit it entirely. `toStringSlice` handles all
  three (verified), so a skill author's reasonable variation never breaks skpp.
- It **establishes the `RelTag` canonicalization** (`filepath.Rel` then
  `filepath.ToSlash`) that PRD §7.1 calls "the canonical tag" and §7.2 resolution
  keys off of. Getting the slash-normalization right here means `resolve` and
  `--list` never have to re-normalize.

---

## What

Additions to `package discover` (the package S1 created), in a NEW file:

1. **`type Skill struct`** — the indexed representation of one skill. 9 fields,
   NO yaml struct tags (Skill is never `yaml.Unmarshal`'d; only `Frontmatter`
   is). Fields: `Dir string` (abs skill dir), `RelTag string` (slash-normalized
   relative path = canonical tag), `Name string`, `Description string`,
   `Keywords []string`, `Category string`, `Aliases []string`, `HasFM bool`,
   `SourceFile string` (abs SKILL.md path).
2. **`func toStringSlice(v any) []string`** — converts a yaml.v3-decoded metadata
   value: `nil`→`nil`; bare `string`→`[]string{t}`; `[]any`→each `string`
   element appended (non-string elements silently skipped, never panics);
   anything else→`nil`.
3. **`func newSkill(absDir, skillsDir string, fm Frontmatter, body string,
   hasFM bool) Skill`** — the constructor: `RelTag = filepath.ToSlash(
   filepath.Rel(skillsDir, absDir))` (defensive `Base` fallback on the rare
   `Rel` error); `SourceFile = filepath.Join(absDir, "SKILL.md")`; copies
   `Name`/`Description` from `fm`; `Keywords = toStringSlice(fm.Metadata["keywords"])`;
   `Category = comma-ok string assert of fm.Metadata["category"]`; `Aliases =
   toStringSlice(fm.Metadata["aliases"])`; `HasFM = hasFM`. `body` is accepted
   (so callers pass `ParseFrontmatter`'s full triple) but currently unused.

### Success Criteria

- [ ] `internal/discover/skill.go` exists, is `package discover`, imports ONLY `path/filepath`
- [ ] `type Skill struct` has EXACTLY these 9 fields (no yaml tags): `Dir`, `RelTag`, `Name`, `Description`, `Keywords []string`, `Category string`, `Aliases []string`, `HasFM bool`, `SourceFile` (all `string` except Keywords/Aliases `[]string` and HasFM `bool`)
- [ ] `func toStringSlice(v any) []string` has that EXACT signature; returns `nil` for `nil`; `[]string{t}` for a bare `string`; a `[]string` of the string elements of a `[]any` (skipping non-strings, no panic); `nil` for any other type
- [ ] `func newSkill(absDir, skillsDir string, fm Frontmatter, body string, hasFM bool) Skill` has that EXACT signature (matches the item description verbatim)
- [ ] `newSkill` computes `RelTag` via `filepath.Rel` + `filepath.ToSlash` (verified: nested `writing/reddit` stays `writing/reddit`)
- [ ] `newSkill` sets `SourceFile = filepath.Join(absDir, "SKILL.md")` and `Dir = absDir` (exact input, no re-Abs)
- [ ] `newSkill` extracts `Keywords = toStringSlice(fm.Metadata["keywords"])`, `Category` via comma-ok `.(string)`, `Aliases = toStringSlice(fm.Metadata["aliases"])`; a nil `Metadata` map yields `Keywords=nil`/`Category=""`/`Aliases=nil` (no panic)
- [ ] `newSkill` threads `HasFM` from the passed `hasFM` param (NOT from `fm.HasFM`)
- [ ] Scenario tests pass: full metadata `keywords:[a,b,c]`+`category:x`+`aliases:[p,q]`; bare-scalar `keywords: writing` → `["writing"]`; no metadata → nil/empty + `""`
- [ ] `go build ./internal/discover/` exits 0; `gofmt -l internal/discover/*.go` silent; `go vet ./internal/discover/` clean; `go test ./internal/discover/ -v` passes (S1 + S2 tests); `go test ./...` green
- [ ] `discover.go`/`discover_test.go`/`go.mod`/`go.sum`/`internal/skillsdir/*`/`main.go`/`main_test.go` UNCHANGED; no `Index()`/`resolve`/`ui`/`skills/` created

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `skill.go` is given verbatim in the Implementation
Blueprint (File 1), and the EXACT test file is given verbatim (File 2). Every
load-bearing behavior was empirically verified against the project's real
yaml.v3 v3.0.1 in Go 1.26.4 (research/verified_facts.md §1–§12): bare-scalar →
string (§1); toStringSlice over all 6 input shapes (§2); safe category assertion
on nil/non-string (§3); `filepath.Rel`+`ToSlash` + the out-of-tree `..`/no-error
behavior (§4); SourceFile via Join (§5); the new-file requirement (§6); white-box
test package (§7); imports = path/filepath only (§8); unused `body` param (§9);
separate `hasFM` param (§10); Skill has NO yaml tags (§11); slice-comparison test
helper to avoid reflect (§12). S1's contract (`Frontmatter` fields, `ParseFrontmatter`
signature) was read from S1's PRP. The implementer needs nothing beyond this doc
and codebase access._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M2T1S2/research/verified_facts.md
  why: "Proves (against the REAL yaml.v3 v3.0.1 in Go 1.26.4): (§1) a BARE scalar
        metadata value like 'keywords: writing' decodes to a STRING, so
        toStringSlice MUST have a 'case string' branch. (§2) the full type-switch
        behavior for nil/string/[]any/mixed/empty/int — non-string elements are
        skipped, never panic; nil→true nil; []any{}→empty non-nil. (§3) the
        comma-ok 'category, _ := fm.Metadata[\"category\"].(string)' is safe on a
        nil map and on a non-string value (yields ''). (§4) filepath.Rel yields
        'writing/reddit' for nested skills and does NOT error for out-of-tree
        paths (returns '..' sequences); ToSlash is a no-op on Linux but REQUIRED
        for Windows. (§5) SourceFile = filepath.Join(absDir,'SKILL.md'). (§6) S2
        MUST use a NEW file skill.go — S1's Level 4 asserts discover.go has no
        'type Skill'/'toStringSlice'. (§7) tests are white-box 'package discover'
        because newSkill/toStringSlice are unexported. (§8) skill.go imports ONLY
        path/filepath. (§9) the unused 'body' param compiles fine (Go ignores
        unused params). (§10) hasFM is a separate param; caller passes fm.HasFM.
        (§11) Skill has NO yaml tags. (§12) use a slicesEq helper, not reflect."
  critical: "Do NOT type-assert fm.Metadata[\"keywords\"] as []string — yaml.v3
             yields []any; that panics. Use toStringSlice. Do NOT edit discover.go
             — put Skill/toStringSlice/newSkill in a NEW skill.go. Do NOT define
             Index()/resolve/ui — S5/M3/M2.T6 own those."

# CONTRACT — the Frontmatter type + ParseFrontmatter S1 produces (consume verbatim)
- file: plan/001_fcde63e5bb60/P1M2T1S1/PRP.md
  why: "Defines the exact Frontmatter struct S1 lands: Name, Description, License,
        Compatibility, Metadata map[string]any, AllowedTools, DisableModelInvocation,
        HasFM bool (yaml:'-'). And ParseFrontmatter(path) (fm Frontmatter, body
        string, err error). S2 reads fm.Name, fm.Description, fm.Metadata (indexing
        'keywords'/'category'/'aliases'), and receives fm.HasFM + body to thread
        into newSkill. S2 does NOT redefine Frontmatter or ParseFrontmatter."
  section: "Data model — the Frontmatter struct", "File 1 — discover.go (the struct + signature)"

# CONTRACT — the metadata schema + the recommended Skill/toStringSlice shape
- file: plan/001_fcde63e5bb60/architecture/external_deps.md
  why: "§1 frontmatter schema: metadata is a spec'd arbitrary key-value map;
        nesting lists (keywords/aliases) under it IS spec-compliant; keywords/
        aliases are skpp conventions read via type assertion on map[string]any
        (yaml.v3 decodes lists as []any). §3 the recommended Frontmatter struct
        shape. The metadata conventions are user-facing (README §6, final task)."
  section: "1. Agent Skills specification > Frontmatter schema", "3. Go third-party dependency"

# CONTRACT — the package map, the Skill struct, the toStringSlice signature, data flow
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "Locks: the Skill struct fields (Dir, RelTag, Name, Description, Keywords,
        Category, Aliases, HasFM, SourceFile) with the exact field-level comments;
        the toStringSlice(v any) []string signature + the metadata-extraction
        snippet (keywords=toStringSlice(...), category comma-ok assert, aliases=
        toStringSlice(...)); the relTag normalization (filepath.Rel then
        filepath.ToSlash); that Skill is consumed by Index/resolve/ui."
  section: "Core types > internal/discover", "Key implementation notes > metadata extraction", "Key implementation notes > relTag normalization"

# CONTRACT — the PRD sections this implements (§7.1 fields, §7.3 parsing leniency, §10 conventions)
- file: PRD.md
  why: "§7.1: the per-skill fields discovery captures (dir/relTag/name/description/
        keywords/category/aliases) and that relTag is the CANONICAL tag with '/'
        separators. §7.3: frontmatter parsing is lenient (S1's job) but establishes
        that Metadata is the source for keywords/category/aliases. §10: the example
        frontmatter with metadata.keywords/category/aliases — the shape S2 extracts.
        READ-ONLY."
  section: "7.1 Discovery", "7.3 Frontmatter parsing", "10. Skill directory & frontmatter conventions"

# REFERENCE — the test convention to follow (white-box, same-package, no reflect/testify)
- file: internal/skillsdir/skillsdir_test.go
  why: "The repo's established test convention: 'package skillsdir' (white-box,
        same-package), t.TempDir()/os.WriteFile/os.MkdirAll, plain t.Errorf/t.Fatalf
        (no testify), no t.Parallel(), helper funcs with t.Helper(). skill_test.go
        mirrors this as 'package discover'. Includes a table-driven test style
        (TestSourceString) reused for TestToStringSlice."
  pattern: "White-box test file alongside the code; build fixtures with os.MkdirAll +
            os.WriteFile under t.TempDir(); plain t.Errorf assertions; a t.Helper()
            fixture writer. No reflect, no testify, no t.Parallel."

# REFERENCE — the sibling package for doc-comment + var/const style
- file: internal/skillsdir/skillsdir.go
  why: "Style to mirror: a package doc comment (already in discover.go via S1 — do
        NOT repeat in skill.go); exported symbols (Skill) have a doc comment;
        unexported helpers (toStringSlice, newSkill) have doc comments explaining
        the contract and citing the behavior. Single import 'path/filepath' on its
        own line (or grouped if more added — none here)."
  pattern: "Exported types get thorough doc comments; unexported helpers documented;
            idiomatic stdlib use (filepath.Rel/ToSlash/Join/Base)."

# URLS — the load-bearing stdlib APIs (no new third-party dep in this subtask)
- url: https://pkg.go.dev/path/filepath#Rel
  why: "filepath.Rel(base, targ) returns targ relative to base. For a skill under
        skillsDir this yields the tag ('foo', 'writing/reddit'). Returns err only
        in rare cases (cross-volume / unresolvable); for out-of-tree paths it
        returns '..' sequences WITHOUT erroring (verified §4)."
  section: "func Rel"
- url: https://pkg.go.dev/path/filepath#ToSlash
  why: "filepath.ToSlash replaces OS separators with '/' — the §7.1 normalization
        that makes relTag canonical on every platform. No-op on Linux; required on
        Windows."
  section: "func ToSlash"
- url: https://pkg.go.dev/path/filepath#Join
  why: "filepath.Join(absDir, 'SKILL.md') builds SourceFile OS-correctly (cleans
        '..', avoids double separators). Byte-identical to absDir+'/SKILL.md' on
        Linux but portable."
  section: "func Join"
```

### Current Codebase tree (M1 landed; S1 discover.go assumed landed)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/skillsdir/skillsdir.go        # M1: Source + findEnv/findSibling/findWalkUp + Find + ErrNotFound
internal/skillsdir/skillsdir_test.go   # M1 tests (white-box, package skillsdir)
internal/discover/discover.go          # S1 (assumed landed): Frontmatter + ParseFrontmatter (bytes/os/strings/yaml.v3)
internal/discover/discover_test.go     # S1 (assumed landed): parser tests (white-box, package discover)
# main.go / main_test.go landed by M1.T3.S1. This subtask does NOT touch them.

$ ls -A
.git/  .gitignore  LICENSE  PRD.md  go.mod  go.sum  internal/  main.go  main_test.go  plan/  .pi-subagents/
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1
#         (S1 already removed '// indirect' via go mod tidy — yaml.v3 is DIRECT.)
# go.sum: yaml.v3 v3.0.1 checksums present (no network needed).
# NO internal/discover/skill.go yet. NO resolve/ ui/ (later milestones).
```

### Desired Codebase tree with files to be added

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, main.go, main_test.go — UNCHANGED)
├── internal/
│   ├── skillsdir/                      # UNCHANGED (M1)
│   └── discover/
│       ├── discover.go                 # S1 — UNCHANGED (Frontmatter + ParseFrontmatter)
│       ├── discover_test.go            # S1 — UNCHANGED (parser tests)
│       ├── skill.go                    # CREATE — package discover: Skill + toStringSlice + newSkill
│       └── skill_test.go               # CREATE — package discover (white-box): skill/metadata tests
└── (no other new files/packages)
```

| File (created) | Responsibility | Consumes | Consumed by |
|---|---|---|---|
| `internal/discover/skill.go` | The `Skill` value type + `toStringSlice` ([]any→[]string) + `newSkill` constructor (RelTag normalization, metadata extraction, HasFM threading) | `discover.Frontmatter` (S1), stdlib `path/filepath` | S5 (`Index()` calls newSkill), M3 (`resolve` reads Skill), M2.T6 (`ui` reads Skill) |
| `internal/discover/skill_test.go` | White-box tests: toStringSlice table; newSkill scenarios (full/bare-scalar/no-metadata) + edge cases (nested relTag, SourceFile/Dir, HasFM, name≠dir, no-frontmatter-block) | `discover.newSkill`, `discover.toStringSlice`, `discover.ParseFrontmatter` (S1) | — |

**No new packages. No edits to `discover.go`/`discover_test.go`. No go.mod/go.sum
change (yaml.v3 already direct after S1). No `Index()`/`resolve`/`ui`/`skills/`.**

### Known Gotchas of our codebase & the yaml.v3 library

```go
// GOTCHA #1 — yaml.v3 decodes YAML LISTS into []any ([]interface{}), NEVER []string.
// So fm.Metadata["keywords"] for `keywords: [a, b, c]` is []any{"a","b"}, NOT
// []string. A direct `fm.Metadata["keywords"].([]string)` PANICS on every real
// skill. toStringSlice is the ONE place that handles the []any -> []string
// conversion; everything else just calls it. VERIFIED (S1 §2, S2 §2).
//   RIGHT: Keywords: toStringSlice(fm.Metadata["keywords"])
//   WRONG: fm.Metadata["keywords"].([]string)   // panics

// GOTCHA #2 — a metadata value can be a BARE SCALAR, not just a list.
// `keywords: writing` (no brackets) decodes to the STRING "writing", not a list.
// toStringSlice MUST have a `case string: return []string{t}` branch, else a
// single-value keyword is dropped to nil. VERIFIED §1 (the item's 2nd test case).

// GOTCHA #3 — Indexing a nil map is SAFE in Go (returns the zero value), and a
// comma-ok type assertion on nil is safe too. So `fm.Metadata["category"].(string)`
// with comma-ok yields "" when Metadata is nil OR when category is a non-string
// (e.g. a number). NEVER use the single-return .(string) — it panics on nil/non-
// string. VERIFIED §3.
//   RIGHT: category, _ := fm.Metadata["category"].(string)
//   WRONG: category := fm.Metadata["category"].(string)   // panics if not a string

// GOTCHA #4 — Skill goes in a NEW FILE (skill.go), NOT discover.go. S1's Level 4
// validation greps discover.go to PROVE it has no `type Skill` and no
// `toStringSlice`. Editing discover.go breaks S1's contract. Separate file also
// keeps the parser (discover.go) and the skill model (skill.go) cleanly divided,
// matching S1's package-doc scope statement. VERIFIED §6.

// GOTCHA #5 — skill.go does NOT carry the `// Package discover` doc comment.
// Only ONE file per package should have the package doc (S1's discover.go does).
// A comment immediately before `package discover` with no blank line becomes the
// package doc and would shadow/dupe S1's. skill.go starts directly with
// `package discover`; documentation lives in per-symbol doc comments. VERIFIED §6.

// GOTCHA #6 — filepath.Rel does NOT error for out-of-tree paths; it returns ".."
// sequences. So the defensive `if err != nil` guard is for rare cases
// (cross-volume / empty path) only. Because Index() only passes in-tree absDirs,
// the guard never fires in production; the filepath.Base(absDir) fallback keeps
// RelTag non-empty regardless. Do NOT remove the guard, but do not expect it to
// trigger. VERIFIED §4.

// GOTCHA #7 — Do NOT call filepath.Abs inside newSkill. The contract is that
// absDir and skillsDir are ALREADY absolute (Index guarantees it; t.TempDir() is
// absolute in tests). Abs-ing internally would rewrite Dir away from the exact
// input and break the `Dir == absDir` assertion. VERIFIED §4.

// GOTCHA #8 — filepath.ToSlash is a no-op on Linux ('/' is already the separator)
// but is REQUIRED for Windows correctness (PRD §7.1: separators normalized to '/').
// Keep it even though the Linux tests can't observe it doing anything. VERIFIED §4.

// GOTCHA #9 — The `body` param of newSkill is currently UNUSED (Skill has no Body
// field). Go does NOT flag unused function PARAMETERS (only unused locals/imports),
// so this compiles fine under go vet/gofmt/go build. body is accepted solely so
// the caller (Index) can pass ParseFrontmatter's full return triple. Document it;
// do not `_ = body`. VERIFIED §9.

// GOTCHA #10 — `hasFM` is a SEPARATE param, not read from fm.HasFM. The item's
// signature is authoritative. Within newSkill assign the PASSED hasFM to
// Skill.HasFM; the canonical Index call site passes fm.HasFM for it. VERIFIED §10.

// GOTCHA #11 — Skill has NO yaml struct tags. Skill is the INDEXED representation
// (built from a Frontmatter), never a yaml.Unmarshal target. Only Frontmatter
// (S1) is yaml-tagged. Adding tags to Skill is dead/misleading metadata. VERIFIED §11.

// GOTCHA #12 — Do NOT define Index(), resolve.Resolve, ui.*, or a Body field.
// S5 owns Index(); M3 owns resolve; M2.T6 owns ui. Skill has exactly the 9 fields
// listed — no Body. VERIFIED (scope split, architecture package map).

// GOTCHA #13 — Comparing []string in tests: reflect.DeepEqual(nil, []string{}) is
// FALSE (a footgun). Use a tiny slicesEq helper (len + element-wise) that treats
// nil and empty as equal, plus an explicit `== nil` check where the contract
// demands a true nil (the no-metadata case). The repo uses no reflect. VERIFIED §12.
```

---

## Implementation Blueprint

### Data model — the Skill struct

No ORM/pydantic (this is Go). The single "model" is the `Skill` struct — the
indexed representation of one skill. It is NEVER yaml-decoded (only `Frontmatter`
is); hence it carries NO yaml struct tags:

```go
// Skill is a single discovered skill: the central element of the index built by
// Index (P1.M2.T5), resolved against by resolve.Resolve (P1.M3.T7), and rendered
// by the ui package (--list / --search, P1.M2.T6).
type Skill struct {
	Dir         string   // absolute path of the skill directory
	RelTag      string   // Dir relative to skillsDir, separators normalized to '/'
	Name        string   // frontmatter name ("" if absent; may differ from Dir)
	Description string   // frontmatter description ("" if absent)
	Keywords    []string // metadata.keywords, else nil
	Category    string   // metadata.category, else ""
	Aliases     []string // metadata.aliases, else nil
	HasFM       bool     // false iff the SKILL.md had no `---` frontmatter block
	SourceFile  string   // absolute path of SKILL.md (filepath.Join(Dir, "SKILL.md"))
}
```

### File 1 — `internal/discover/skill.go` (CREATE)

Create the file with EXACTLY this content (gofmt-clean; comments explain every
non-obvious decision and cite the research section that verified it). NOTE:
`skill.go` does NOT start with a `// Package discover` comment — that lives in
`discover.go` (S1). Start directly with `package discover`:

```go
package discover

import "path/filepath"

// Skill is a single discovered skill: the central element of the index built by
// Index (P1.M2.T5), resolved against by resolve.Resolve (P1.M3.T7), and rendered
// by the ui package (--list / --search, P1.M2.T6).
//
// A Skill is constructed by newSkill from a parsed Frontmatter (P1.M2.T1.S1)
// plus the skill's absolute directory and the skills-dir root. It is a pure
// value type; downstream packages read its fields and never construct one
// themselves (newSkill is unexported; only same-package Index calls it).
//
// Field meanings (PRD §7.1 / §7.3 / §10):
//
//	Dir         absolute path of the skill directory (contains SKILL.md)
//	RelTag      Dir relative to the skills dir, OS separators normalized to "/"
//	            (e.g. "writing/reddit"). This is the CANONICAL tag (PRD §7.1).
//	Name        frontmatter `name` ("" if absent; may differ from the dir name)
//	Description frontmatter `description` ("" if absent)
//	Keywords    metadata.keywords as []string (nil if absent / not a list)
//	Category    metadata.category as a string ("" if absent / not a string)
//	Aliases     metadata.aliases as []string (nil if absent / not a list)
//	HasFM       false iff the SKILL.md had NO `---` frontmatter block
//	SourceFile  absolute path of the SKILL.md (filepath.Join(Dir, "SKILL.md"))
//
// Skill is NOT a YAML-decoding target (only Frontmatter is), so it carries no
// `yaml:"..."` struct tags.
type Skill struct {
	Dir         string
	RelTag      string
	Name        string
	Description string
	Keywords    []string
	Category    string
	Aliases     []string
	HasFM       bool
	SourceFile  string
}

// newSkill builds a Skill from a parsed Frontmatter and the skill's on-disk
// location. It is the constructor Index (P1.M2.T5) calls for every directory
// that contains a SKILL.md.
//
// Parameters:
//
//	absDir     absolute path of the skill directory (Index guarantees absolute).
//	skillsDir  absolute path of the skills dir root (the WalkDir root).
//	fm         the Frontmatter parsed from absDir/SKILL.md (P1.M2.T1.S1).
//	body       the SKILL.md markdown body from ParseFrontmatter. Skill has no
//	           body field today, so body is accepted only so the caller can pass
//	           ParseFrontmatter's full return triple — reserved for future
//	           body-aware features (e.g. check). Currently unused.
//	hasFM      whether a `---` frontmatter block was found. Threaded separately
//	           from fm.HasFM per the constructor contract; the canonical Index
//	           call site passes fm.HasFM for this argument.
//
// Metadata extraction (skpp conventions, PRD §7.1 / §10, verified §1–§3):
// keywords and aliases are read from fm.Metadata via toStringSlice (tolerates a
// yaml list []any, a bare scalar string, or nil); category is the comma-ok
// string assertion (safe on a nil map and on a non-string value).
func newSkill(absDir, skillsDir string, fm Frontmatter, body string, hasFM bool) Skill {
	rel, relErr := filepath.Rel(skillsDir, absDir)
	if relErr != nil {
		// absDir is not relativizable to skillsDir (rare: cross-volume / empty
		// path). Index() only ever passes in-tree absDirs, so this guard never
		// fires in production; the Base fallback keeps RelTag non-empty. (§4)
		rel = filepath.Base(absDir)
	}
	category, _ := fm.Metadata["category"].(string)
	return Skill{
		Dir:         absDir,
		RelTag:      filepath.ToSlash(rel),
		Name:        fm.Name,
		Description: fm.Description,
		Keywords:    toStringSlice(fm.Metadata["keywords"]),
		Category:    category,
		Aliases:     toStringSlice(fm.Metadata["aliases"]),
		HasFM:       hasFM,
		SourceFile:  filepath.Join(absDir, "SKILL.md"),
	}
}

// toStringSlice converts a yaml.v3-decoded metadata value into []string.
//
// yaml.v3 (the only third-party dep, pinned in go.mod) decodes:
//   - a YAML list (flow `[a, b]` or block `- a`) into []any whose elements are
//     typically interface{} holding a string;
//   - a bare scalar (`keywords: writing`) into a plain string; (verified §1)
//   - an absent key into nil (indexing a nil map also yields nil).
//
// Mapping (verified §2 over all input shapes):
//   - nil            -> nil                 (key absent / Metadata nil)
//   - string         -> []string{that}      (bare single value)
//   - []any          -> each string element appended; non-string elements are
//                      silently skipped (lenient — never panics)
//   - anything else  -> nil                 (unexpected type, e.g. a bare number)
//
// Used for metadata.keywords and metadata.aliases. yaml.v3 NEVER produces
// []string directly, so there is no []string case.
func toStringSlice(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
```

### File 2 — `internal/discover/skill_test.go` (CREATE, `package discover` white-box)

Create the file with EXACTLY this content. It mirrors the repo's test convention
(white-box same-package, `t.TempDir()`/`os.WriteFile`/`os.MkdirAll`, plain
`t.Errorf`/`t.Fatalf`, no testify, no reflect, no `t.Parallel()`):

```go
package discover

import (
	"os"
	"path/filepath"
	"testing"
)

// slicesEq reports whether two []string slices hold the same elements in order.
// nil and an empty slice compare equal (len 0), which is the functional truth
// for all downstream consumers (range/len). Used instead of reflect.DeepEqual to
// avoid the nil-vs-empty footgun and to match the repo's no-reflect test style.
func slicesEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// writeSkillMD writes content to <skillsDir>/<tag>/SKILL.md and returns the
// absolute skill directory. tag may be nested (e.g. "writing/reddit"); MkdirAll
// creates intermediate dirs. ParseFrontmatter + newSkill touch no env/cwd, so
// these tests are fully hermetic (t.TempDir is already absolute + clean).
func writeSkillMD(t *testing.T, skillsDir, tag, content string) string {
	t.Helper()
	skillDir := filepath.Join(skillsDir, tag)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return skillDir
}

// parseAndBuild is the canonical Index()-style call: parse the SKILL.md at
// skillDir/SKILL.md (S1's ParseFrontmatter), then build a Skill (S2's newSkill).
// It exercises the S1->S2 integration seam end-to-end and passes fm.HasFM as the
// hasFM argument exactly as Index will.
func parseAndBuild(t *testing.T, skillDir, skillsDir string) Skill {
	t.Helper()
	fm, body, err := ParseFrontmatter(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	return newSkill(skillDir, skillsDir, fm, body, fm.HasFM)
}

// --- toStringSlice: direct unit tests over all input shapes ---

func TestToStringSlice(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"bare string", "writing", []string{"writing"}},
		{"list of strings", []any{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"mixed types skip non-strings", []any{"a", 1, "b"}, []string{"a", "b"}},
		{"empty list", []any{}, []string{}},
		{"unexpected type (int)", 7, nil},
		{"unexpected type (map)", map[string]any{"k": "v"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := toStringSlice(c.in)
			if !slicesEq(got, c.want) {
				t.Errorf("toStringSlice(%#v) = %#v; want %#v", c.in, got, c.want)
			}
		})
	}
}

// toStringSlice(nil) MUST return a genuine nil slice (not []string{}), per the
// item's "nil/missing -> nil" contract. slicesEq treats them equal, so assert
// nil-ness explicitly here.
func TestToStringSliceNilIsTrueNil(t *testing.T) {
	if got := toStringSlice(nil); got != nil {
		t.Errorf("toStringSlice(nil) = %#v; want a true nil []string", got)
	}
}

// --- newSkill: the three item-specified scenarios ---

// Scenario 1: metadata.keywords=[a,b,c], metadata.category=x, metadata.aliases=[p,q].
func TestNewSkillFullMetadata(t *testing.T) {
	skillsDir := t.TempDir()
	content := "---\n" +
		"name: my-skill\n" +
		"description: does things\n" +
		"metadata:\n" +
		"  keywords: [a, b, c]\n" +
		"  category: x\n" +
		"  aliases:\n" +
		"    - p\n" +
		"    - q\n" +
		"---\n# body\n"
	skillDir := writeSkillMD(t, skillsDir, "my-skill", content)
	s := parseAndBuild(t, skillDir, skillsDir)

	if s.Dir != skillDir {
		t.Errorf("Dir=%q; want %q", s.Dir, skillDir)
	}
	if s.RelTag != "my-skill" {
		t.Errorf("RelTag=%q; want %q", s.RelTag, "my-skill")
	}
	if s.Name != "my-skill" {
		t.Errorf("Name=%q; want %q", s.Name, "my-skill")
	}
	if s.Description != "does things" {
		t.Errorf("Description=%q; want %q", s.Description, "does things")
	}
	if !slicesEq(s.Keywords, []string{"a", "b", "c"}) {
		t.Errorf("Keywords=%#v; want [a b c]", s.Keywords)
	}
	if s.Category != "x" {
		t.Errorf("Category=%q; want %q", s.Category, "x")
	}
	if !slicesEq(s.Aliases, []string{"p", "q"}) {
		t.Errorf("Aliases=%#v; want [p q]", s.Aliases)
	}
	if !s.HasFM {
		t.Errorf("HasFM=false; want true")
	}
	if want := filepath.Join(skillDir, "SKILL.md"); s.SourceFile != want {
		t.Errorf("SourceFile=%q; want %q", s.SourceFile, want)
	}
}

// Scenario 2: a single-string (bare scalar) keyword -> Keywords=[writing].
func TestNewSkillBareStringKeyword(t *testing.T) {
	skillsDir := t.TempDir()
	content := "---\n" +
		"name: single\n" +
		"description: hi\n" +
		"metadata:\n" +
		"  keywords: writing\n" + // bare scalar, NOT a list
		"---\n# body\n"
	skillDir := writeSkillMD(t, skillsDir, "single", content)
	s := parseAndBuild(t, skillDir, skillsDir)

	if !slicesEq(s.Keywords, []string{"writing"}) {
		t.Errorf("Keywords=%#v; want [writing] (bare scalar wrapped into a slice)", s.Keywords)
	}
	// No category/aliases declared -> zero values.
	if s.Category != "" {
		t.Errorf("Category=%q; want empty", s.Category)
	}
	if len(s.Aliases) != 0 {
		t.Errorf("Aliases=%#v; want nil/empty", s.Aliases)
	}
}

// Scenario 3: frontmatter present but NO metadata map at all.
func TestNewSkillNoMetadata(t *testing.T) {
	skillsDir := t.TempDir()
	content := "---\n" +
		"name: bare\n" +
		"description: hi\n" +
		"---\n# body\n"
	skillDir := writeSkillMD(t, skillsDir, "bare", content)
	s := parseAndBuild(t, skillDir, skillsDir)

	// Metadata was nil/absent -> all metadata-derived fields are zero.
	if s.Keywords != nil {
		t.Errorf("Keywords=%#v; want nil (no metadata)", s.Keywords)
	}
	if s.Category != "" {
		t.Errorf("Category=%q; want empty (no metadata)", s.Category)
	}
	if s.Aliases != nil {
		t.Errorf("Aliases=%#v; want nil (no metadata)", s.Aliases)
	}
	// Name/Description still copied from the frontmatter.
	if s.Name != "bare" || s.Description != "hi" {
		t.Errorf("Name=%q Description=%q; want bare/hi", s.Name, s.Description)
	}
	if !s.HasFM {
		t.Errorf("HasFM=false; want true (frontmatter block present)")
	}
}

// --- newSkill: structural / path fields ---

// RelTag is slash-normalized for a NESTED skill (writing/reddit). On Linux '/'
// is already the separator, so this confirms the joined relative-path value;
// ToSlash guarantees '/' on Windows too.
func TestNewSkillRelTagNested(t *testing.T) {
	skillsDir := t.TempDir()
	content := "---\nname: reddit\ndescription: hi\n---\n# body\n"
	skillDir := writeSkillMD(t, skillsDir, filepath.Join("writing", "reddit"), content)
	s := parseAndBuild(t, skillDir, skillsDir)

	if s.RelTag != "writing/reddit" {
		t.Errorf("RelTag=%q; want %q", s.RelTag, "writing/reddit")
	}
	if s.Dir != skillDir {
		t.Errorf("Dir=%q; want %q", s.Dir, skillDir)
	}
}

// SourceFile is the absolute SKILL.md path (filepath.Join(absDir, "SKILL.md"));
// Dir is the absolute skill directory verbatim (no Abs/Clean re-write).
func TestNewSkillSourceFileAndDir(t *testing.T) {
	skillsDir := t.TempDir()
	skillDir := writeSkillMD(t, skillsDir, "foo", "---\nname: foo\ndescription: hi\n---\n# body\n")
	s := parseAndBuild(t, skillDir, skillsDir)

	if s.Dir != skillDir {
		t.Errorf("Dir=%q; want the exact absDir %q", s.Dir, skillDir)
	}
	if want := filepath.Join(skillDir, "SKILL.md"); s.SourceFile != want {
		t.Errorf("SourceFile=%q; want %q", s.SourceFile, want)
	}
}

// HasFM is threaded from the passed flag (true AND false), not read from fm.
func TestNewSkillHasFMPropagated(t *testing.T) {
	skillsDir := t.TempDir()
	skillDir := writeSkillMD(t, skillsDir, "foo", "---\nname: foo\ndescription: hi\n---\n# body\n")
	fm, _, err := ParseFrontmatter(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := newSkill(skillDir, skillsDir, fm, "", true); !got.HasFM {
		t.Errorf("hasFM=true -> Skill.HasFM=false; want true")
	}
	if got := newSkill(skillDir, skillsDir, fm, "", false); got.HasFM {
		t.Errorf("hasFM=false -> Skill.HasFM=true; want false")
	}
}

// Name may DIFFER from the directory name (PRD §7.1: name "may differ from dir").
func TestNewSkillNameDiffersFromDir(t *testing.T) {
	skillsDir := t.TempDir()
	// dir is "foo" but frontmatter name is "foo-helper".
	content := "---\nname: foo-helper\ndescription: hi\n---\n# body\n"
	skillDir := writeSkillMD(t, skillsDir, "foo", content)
	s := parseAndBuild(t, skillDir, skillsDir)

	if s.Name != "foo-helper" {
		t.Errorf("Name=%q; want %q (frontmatter name, not dir)", s.Name, "foo-helper")
	}
	if s.RelTag != "foo" {
		t.Errorf("RelTag=%q; want %q (derived from dir, not name)", s.RelTag, "foo")
	}
}

// No frontmatter block at all (HasFM false) still yields a usable Skill whose
// metadata-derived fields are zero and RelTag/Dir/SourceFile are correct.
func TestNewSkillNoFrontmatterBlock(t *testing.T) {
	skillsDir := t.TempDir()
	content := "# just a body\nno fences\n" // no --- block
	skillDir := writeSkillMD(t, skillsDir, "nofm", content)
	s := parseAndBuild(t, skillDir, skillsDir)

	if s.HasFM {
		t.Errorf("HasFM=true; want false (no frontmatter block)")
	}
	if s.Name != "" || s.Description != "" {
		t.Errorf("Name=%q Description=%q; want empty", s.Name, s.Description)
	}
	if s.Keywords != nil || s.Aliases != nil || s.Category != "" {
		t.Errorf("metadata fields non-zero: Keywords=%#v Aliases=%#v Category=%q", s.Keywords, s.Aliases, s.Category)
	}
	// Path fields still set from the directory.
	if s.RelTag != "nofm" {
		t.Errorf("RelTag=%q; want %q", s.RelTag, "nofm")
	}
	if want := filepath.Join(skillDir, "SKILL.md"); s.SourceFile != want {
		t.Errorf("SourceFile=%q; want %q", s.SourceFile, want)
	}
}
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: PRECONDITION — confirm S1's discover.go is landed (Frontmatter + ParseFrontmatter)
  - COMMAND: cd /home/dustin/projects/skpp
  - COMMAND: test -f internal/discover/discover.go || { echo "S1 (discover.go) not landed yet — S2 depends on it"; exit 1; }
  - COMMAND: grep -q 'type Frontmatter struct' internal/discover/discover.go
  - COMMAND: grep -qE 'func ParseFrontmatter\(path string\) \(fm Frontmatter, body string, err error\)' internal/discover/discover.go
  - COMMAND: grep -q 'Metadata ' internal/discover/discover.go   # the map[string]any field S2 reads
  - EXPECT: discover.go exists with the Frontmatter struct (incl. Metadata map[string]any +
            HasFM bool) and the ParseFrontmatter function. yaml.v3 is already a DIRECT dep
            (S1 ran `go mod tidy`). If S1 is NOT landed, STOP — S2 cannot run without it.

Task 1: CREATE internal/discover/skill.go
  - WRITE: the exact content from the Blueprint (File 1) to ./internal/discover/skill.go.
  - CHECK: `package discover`; single import `path/filepath` (ONLY this — no os/bytes/
           strings/yaml.v3); `type Skill struct` with 9 fields and NO yaml tags;
           `func newSkill(absDir, skillsDir string, fm Frontmatter, body string, hasFM bool) Skill`;
           `func toStringSlice(v any) []string`.
  - GOTCHA: do NOT start skill.go with `// Package discover` (S1's discover.go owns the
            package doc). Do NOT add yaml tags to Skill. Do NOT read fm.HasFM inside
            newSkill (use the passed hasFM). Do NOT define Index/resolve/ui.

Task 2: CREATE internal/discover/skill_test.go
  - WRITE: the exact content from the Blueprint (File 2) to ./internal/discover/skill_test.go.
  - CHECK: `package discover` (white-box, NOT discover_test); imports = os, path/filepath,
           testing (ONLY these — NO reflect/testify); tests cover: toStringSlice table
           (nil/string/[]any/mixed/empty/int/map) + toStringSlice-nil-is-true-nil;
           newSkill scenario 1 (full metadata) + scenario 2 (bare-scalar keyword) +
           scenario 3 (no metadata) + nested relTag + SourceFile/Dir + HasFM propagation +
           name≠dir + no-frontmatter-block.
  - GOTCHA: use the slicesEq helper (NOT reflect.DeepEqual). No t.Parallel(). Use
            os.MkdirAll + os.WriteFile under t.TempDir(). parseAndBuild passes fm.HasFM
            as the hasFM arg (the canonical Index call).

Task 3: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/discover/skill.go internal/discover/skill_test.go
  - COMMAND: gofmt -l internal/discover/*.go   # MUST print nothing (covers S1 + S2 files)
  - COMMAND: go vet ./internal/discover/       # MUST be clean
  - COMMAND: go build ./internal/discover/     # exit 0
  - COMMAND: go test ./internal/discover/ -v   # ALL discover tests PASS (S1 parser + S2 skill)
  - COMMAND: go test ./...                     # whole module still green
  - EXPECT: zero errors, zero vet findings, gofmt silent, all tests pass.

Task 4: SCENARIO + EDGE-CASE SMOKE TEST — Level 3 in Validation Loop
  - COMMAND: the Level 3 block below (run the 3 item scenarios + toStringSlice + edge cases).
  - EXPECT: all targeted tests pass.

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: skill.go has the exact 9-field struct, the exact signatures, single import;
            discover.go/discover_test.go/go.mod/go.sum/skillsdir/main UNCHANGED; no
            Index/resolve/ui created.
```

### Implementation Patterns & Key Details

```go
// PATTERN: the []any -> []string conversion lives in ONE place (toStringSlice).
//   Keywords: toStringSlice(fm.Metadata["keywords"])
//   Aliases:  toStringSlice(fm.Metadata["aliases"])
// WHY: yaml.v3 decodes YAML lists into []any, NEVER []string. A direct
//      `fm.Metadata["keywords"].([]string)` PANICS on every real skill.
//      Centralizing the conversion in toStringSlice means it is correct
//      everywhere and easy to audit. VERIFIED §1, §2.

// PATTERN: type switch on `any` with nil/string/[]any/default arms.
//   switch t := v.(type) {
//   case nil:     return nil
//   case string:  return []string{t}
//   case []any:   ... append each string element, skip non-strings ...
//   default:      return nil
//   }
// WHY: handles every shape yaml.v3 can produce for a metadata value (verified §2)
//      AND is panic-proof (no unchecked type assertion). The `default` arm makes
//      a weird value (e.g. `keywords: 7`) leniently nil instead of crashing.

// PATTERN: comma-ok type assertion for a scalar metadata field.
//   category, _ := fm.Metadata["category"].(string)
// WHY: safe on a nil map (indexing returns nil) AND on a non-string value
//      (returns "", false). The single-return `.(string)` would PANIC. VERIFIED §3.

// PATTERN: canonical relTag = ToSlash(Rel(skillsDir, absDir)).
//   rel, relErr := filepath.Rel(skillsDir, absDir)
//   if relErr != nil { rel = filepath.Base(absDir) }   // defensive; never fires in-tree
//   RelTag: filepath.ToSlash(rel),
// WHY: PRD §7.1 makes relTag the canonical tag with '/' separators. Rel gives the
//      relative path; ToSlash normalizes OS separators (no-op on Linux, required on
//      Windows). The Base fallback guards the rare cross-volume/empty Rel error.
//      VERIFIED §4.

// PATTERN: Skill fields carry NO yaml struct tags.
//   type Skill struct { Dir string; RelTag string; ... }   // no `yaml:"..."` anywhere
// WHY: Skill is the INDEXED representation built from a Frontmatter; yaml.Unmarshal
//      is NEVER called on a Skill. Only Frontmatter (S1) is yaml-tagged. VERIFIED §11.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/discover/skill.go is `package discover` (internal => unimportable
    outside the module). It ADDS to the package S1 created; it does NOT replace it.
  - skill.go imports (production): path/filepath (ONLY). Disjoint from discover.go's
    imports (bytes/os/strings/yaml.v3).
  - exposes: type Skill (9 fields); unexported toStringSlice, newSkill.
  - consumes: discover.Frontmatter (S1) — reads .Name, .Description, .Metadata, and
    receives .HasFM via the caller. discover.ParseFrontmatter (S1) — called only in
    tests (parseAndBuild helper), NOT in skill.go itself.

GO.MOD / GO.SUM: UNCHANGED. yaml.v3 is already a DIRECT dependency after S1
  (`go mod tidy` already removed `// indirect`). skill.go imports only stdlib
  (path/filepath), so it introduces NO new module requirement. Do NOT run
  `go mod tidy` expecting a change — there is none.

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into):
  - P1.M2.T5 (Index walk): for each dir containing SKILL.md, call
    `fm, body, err := ParseFrontmatter(filepath.Join(skillDir, "SKILL.md"))`;
    on error, collect/skip; else `skills = append(skills, newSkill(skillDir,
    skillsDir, fm, body, fm.HasFM))`. Then sort by RelTag and return []Skill.
  - P1.M3.T7 (resolve.Resolve): reads Skill.RelTag (exact + basename match),
    Skill.Name (frontmatter-name match), Skill.Aliases (alias match). Returns a
    Result{Skill, MatchKind}.
  - P1.M2.T6 (ui): reads Skill.RelTag/Name/Description/Category/Keywords/HasFM
    to render the --list / --search table; HasFM=false => show description as
    "(missing)" (PRD §7.3).

NO CHANGES TO:
  - go.mod / go.sum (byte-identical — yaml.v3 already direct)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned) / prd_snapshot.md
  - internal/discover/discover.go + discover_test.go (S1-owned; S2 must not edit)
  - internal/skillsdir/* (M1-owned; discover does not import it)
  - main.go / main_test.go (M1.T3.S1-owned; leave alone)
  - any other package or file (resolve/ui/install.sh/README/completions/skills/
    are later subtasks)
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass).
# Covers BOTH S1's files and S2's new files (whole package must be gofmt-clean).
gofmt -w internal/discover/skill.go internal/discover/skill_test.go
test -z "$(gofmt -l internal/discover/*.go)" || { echo "FAIL: gofmt found unformatted files"; gofmt -d internal/discover/*.go; exit 1; }
echo "gofmt OK"

# Vet the discover package (S1 + S2 together).
go vet ./internal/discover/ || { echo "FAIL: go vet ./internal/discover/"; exit 1; }
echo "go vet OK"

# Build the package (proves skill.go compiles against S1's Frontmatter).
go build ./internal/discover/ || { echo "FAIL: go build ./internal/discover/"; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run discover tests verbosely — BOTH S1's parser tests AND S2's new skill tests.
go test ./internal/discover/ -v || { echo "FAIL: go test ./internal/discover/ -v"; exit 1; }

# Whole module still green (skillsdir + discover + main).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Scenario + edge-case smoke test (the contract this subtask locks)

```bash
cd /home/dustin/projects/skpp

# The targeted tests proving each load-bearing behavior:
#   - toStringSlice over all input shapes (nil/string/[]any/mixed/empty/int/map)
#   - toStringSlice(nil) returns a TRUE nil (not []string{})
#   - newSkill scenario 1: full metadata keywords=[a,b,c] category=x aliases=[p,q]
#   - newSkill scenario 2: bare-scalar keyword -> [writing]
#   - newSkill scenario 3: no metadata -> nil/"" zero values
#   - newSkill nested relTag (writing/reddit), SourceFile/Dir, HasFM propagation,
#     name differs from dir, no-frontmatter-block
go test ./internal/discover/ -v \
  -run 'TestToStringSlice|TestNewSkill' \
  || { echo "FAIL: skill/metadata scenario + edge-case tests"; exit 1; }
echo "Level 3 PASS (toStringSlice + 3 scenarios + path/HasFM/name edges)"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# skill.go exists and is package discover
test -f internal/discover/skill.go || { echo "FAIL: skill.go missing"; exit 1; }
grep -q '^package discover' internal/discover/skill.go || { echo "FAIL: skill.go not package discover"; exit 1; }

# Skill struct has EXACTLY the 9 fields (no yaml tags). Check each field is present
# AND that NO yaml backtick tag appears on any Skill field line.
grep -qE '^\tDir[[:space:]]+string' internal/discover/skill.go || { echo "FAIL: Dir field"; exit 1; }
grep -qE '^\tRelTag[[:space:]]+string' internal/discover/skill.go || { echo "FAIL: RelTag field"; exit 1; }
grep -qE '^\tName[[:space:]]+string' internal/discover/skill.go || { echo "FAIL: Name field"; exit 1; }
grep -qE '^\tDescription[[:space:]]+string' internal/discover/skill.go || { echo "FAIL: Description field"; exit 1; }
grep -qE '^\tKeywords[[:space:]]+\[\]string' internal/discover/skill.go || { echo "FAIL: Keywords field"; exit 1; }
grep -qE '^\tCategory[[:space:]]+string' internal/discover/skill.go || { echo "FAIL: Category field"; exit 1; }
grep -qE '^\tAliases[[:space:]]+\[\]string' internal/discover/skill.go || { echo "FAIL: Aliases field"; exit 1; }
grep -qE '^\tHasFM[[:space:]]+bool' internal/discover/skill.go || { echo "FAIL: HasFM field"; exit 1; }
grep -qE '^\tSourceFile[[:space:]]+string' internal/discover/skill.go || { echo "FAIL: SourceFile field"; exit 1; }

# Skill struct block must contain NO yaml struct tag (yaml:"..." is only on Frontmatter).
awk '/^type Skill struct/,/^}/' internal/discover/skill.go | grep -q 'yaml:' \
  && { echo "FAIL: Skill struct must have NO yaml tags"; exit 1; } || echo "Skill: no yaml tags OK"

# Exact function signatures (match the item description + architecture contract)
grep -qE 'func toStringSlice\(v any\) \[\]string' internal/discover/skill.go \
  || { echo "FAIL: toStringSlice signature"; exit 1; }
grep -qE 'func newSkill\(absDir, skillsDir string, fm Frontmatter, body string, hasFM bool\) Skill' internal/discover/skill.go \
  || { echo "FAIL: newSkill signature"; exit 1; }

# skill.go imports ONLY path/filepath (no os/bytes/strings/yaml.v3/reflect)
test "$(go list -f '{{join .GoFiles " "}}' ./internal/discover/ 2>/dev/null | grep -q skill && go list -f '{{join .Imports " "}}' ./internal/discover/ 2>/dev/null)" \
  || true  # go list merges imports across the whole package; verify directly instead:
# Direct check: skill.go's own import block is exactly path/filepath.
sed -n '/^import (/,/^)/p; /^import "/p' internal/discover/skill.go | grep -v 'path/filepath' | grep -qE 'import|"' \
  && { echo "FAIL: skill.go must import ONLY path/filepath"; sed -n '/^import/,/^)/p' internal/discover/skill.go; exit 1; } \
  || echo "skill.go imports OK (path/filepath only)"

# newSkill MUST NOT read fm.HasFM (it uses the passed hasFM param)
awk '/func newSkill/,/^}/' internal/discover/skill.go | grep -q 'fm\.HasFM' \
  && { echo "FAIL: newSkill must use the passed hasFM, not fm.HasFM"; exit 1; } || echo "newSkill uses passed hasFM OK"

# newSkill MUST set RelTag via ToSlash(Rel(...)) and SourceFile via Join
awk '/func newSkill/,/^}/' internal/discover/skill.go | grep -q 'filepath.Rel' || { echo "FAIL: newSkill must call filepath.Rel"; exit 1; }
awk '/func newSkill/,/^}/' internal/discover/skill.go | grep -q 'filepath.ToSlash' || { echo "FAIL: newSkill must call filepath.ToSlash"; exit 1; }
awk '/func newSkill/,/^}/' internal/discover/skill.go | grep -q 'filepath.Join' || { echo "FAIL: newSkill must build SourceFile via filepath.Join"; exit 1; }

# skill_test.go is white-box package discover with the key tests
test -f internal/discover/skill_test.go || { echo "FAIL: skill_test.go missing"; exit 1; }
grep -q '^package discover' internal/discover/skill_test.go || { echo "FAIL: skill_test.go must be package discover (white-box)"; exit 1; }
# NOTE: grep for the QUOTED import form, not the bare word — the slicesEq doc
# comment legitimately mentions 'reflect.DeepEqual' to explain why it is avoided.
! grep -q '"reflect"' internal/discover/skill_test.go || { echo "FAIL: skill_test.go must not import reflect (use slicesEq)"; exit 1; }
! grep -q 'stretchr/testify' internal/discover/skill_test.go || { echo "FAIL: skill_test.go must not import testify"; exit 1; }
for tn in TestToStringSlice TestToStringSliceNilIsTrueNil TestNewSkillFullMetadata TestNewSkillBareStringKeyword TestNewSkillNoMetadata TestNewSkillRelTagNested TestNewSkillSourceFileAndDir TestNewSkillHasFMPropagated TestNewSkillNameDiffersFromDir TestNewSkillNoFrontmatterBlock; do
  grep -q "func $tn" internal/discover/skill_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# go.mod / go.sum UNCHANGED (yaml.v3 already direct after S1 — no tidy change expected)
git diff --quiet go.mod   || { echo "FAIL: go.mod changed (should be byte-identical — yaml.v3 already direct)"; exit 1; }
git diff --quiet go.sum   || { echo "FAIL: go.sum changed (should be byte-identical)"; exit 1; }

# MUST NOT have touched S1's files / skillsdir / main / PRD
git diff --quiet internal/discover/discover.go      || { echo "FAIL: discover.go changed (S1-owned)"; exit 1; }
git diff --quiet internal/discover/discover_test.go || { echo "FAIL: discover_test.go changed (S1-owned)"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go    || { echo "FAIL: skillsdir.go changed"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir_test.go || { echo "FAIL: skillsdir_test.go changed"; exit 1; }
git diff --quiet PRD.md   || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
test -f main.go && { git diff --quiet main.go || { echo "FAIL: main.go changed (M1.T3.S1-owned)"; exit 1; }; } || true

# MUST NOT have created later-milestone files/packages
! grep -q 'func Index' internal/discover/skill.go || { echo "FAIL: Index() must not exist (S5)"; exit 1; }
test ! -d internal/resolve || { echo "FAIL: resolve/ must not exist (M3)"; exit 1; }
test ! -d internal/ui      || { echo "FAIL: ui/ must not exist (M2.T6)"; exit 1; }
test ! -f install.sh       || { echo "FAIL: install.sh must not exist (M6)"; exit 1; }
test ! -f README.md        || { echo "FAIL: README.md must not exist (M6)"; exit 1; }
test ! -d skills           || { echo "FAIL: skills/ must not exist (M6 owns it)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l internal/discover/*.go` silent (S1 + S2 files), `go vet ./internal/discover/` clean, `go build ./internal/discover/` exit 0
- [ ] Level 2 PASS — `go test ./internal/discover/ -v` all discover tests pass (S1 parser + S2 skill); `go test ./...` whole module green
- [ ] Level 3 PASS — toStringSlice over all input shapes + true-nil; the 3 item scenarios (full metadata / bare-scalar keyword / no metadata); nested relTag, SourceFile/Dir, HasFM propagation, name≠dir, no-frontmatter-block
- [ ] Level 4 PASS — exact 9-field Skill struct with NO yaml tags; exact toStringSlice + newSkill signatures; skill.go imports ONLY path/filepath; newSkill uses Rel+ToSlash+Join and the passed hasFM (not fm.HasFM); white-box test file with all key tests and no reflect/testify; go.mod/go.sum/discover.go/discover_test.go/skillsdir/main/PRD unchanged; no Index/resolve/ui/skills

### Feature Validation
- [ ] `Skill` struct has the 9 fields with correct types (Dir/RelTag/Name/Description/Category/SourceFile `string`; Keywords/Aliases `[]string`; HasFM `bool`)
- [ ] `toStringSlice(nil)` → true `nil`; `toStringSlice("x")` → `["x"]`; `toStringSlice([]any{"a",1,"b"})` → `["a","b"]` (skips non-strings, no panic); `toStringSlice(7)` → `nil`
- [ ] `newSkill` scenario 1: `metadata:{keywords:[a,b,c],category:x,aliases:[p,q]}` → Keywords `[a b c]`, Category `x`, Aliases `[p q]`, plus correct Dir/RelTag/Name/Description/HasFM/SourceFile
- [ ] `newSkill` scenario 2: bare-scalar `keywords: writing` → Keywords `["writing"]` (wrapped), no panic
- [ ] `newSkill` scenario 3: no metadata → Keywords `nil`, Category `""`, Aliases `nil` (Name/Description still copied, HasFM true if a block was present)
- [ ] Nested skill `skills/writing/reddit/` → RelTag `writing/reddit`
- [ ] `SourceFile == filepath.Join(absDir, "SKILL.md")`; `Dir == absDir` (exact input)
- [ ] `HasFM` propagated from the passed `hasFM` param (true→true, false→false)
- [ ] Name may differ from the dir name (RelTag from dir, Name from frontmatter)
- [ ] No-frontmatter-block skill → HasFM false, empty Name/Description, nil metadata fields, but valid Dir/RelTag/SourceFile

### Code Quality / Convention Validation
- [ ] `skill.go` is `package discover` (internal); imports limited to `path/filepath`; does NOT repeat the `// Package discover` doc (S1 owns it)
- [ ] `skill_test.go` is white-box `package discover`, mirroring skillsdir_test.go / discover_test.go style (t.TempDir/os.MkdirAll/os.WriteFile, plain t.Errorf/t.Fatalf, slicesEq helper, no testify, no reflect, no t.Parallel)
- [ ] Exported `Skill` has a thorough doc comment (field meanings, PRD refs); unexported `toStringSlice`/`newSkill` have doc comments explaining the conversion contract and citing the research sections
- [ ] `Skill` fields carry NO yaml struct tags (it is never yaml-decoded)
- [ ] The constructor uses idiomatic stdlib (`filepath.Rel`/`ToSlash`/`Join`/`Base`) — no hand-rolled path logic

### Scope Discipline
- [ ] Did NOT edit `internal/discover/discover.go` / `discover_test.go` (S1-owned; new file `skill.go` used instead)
- [ ] Did NOT define `Index()` (S5 owns it), `resolve`, `ui`, or a `Body` field
- [ ] Did NOT modify `internal/skillsdir/*` (M1-owned), `main.go`/`main_test.go` (M1.T3.S1-owned), `PRD.md` (read-only), any `tasks.json` (orchestrator-owned)
- [ ] Did NOT modify `go.mod` / `go.sum` (byte-identical — yaml.v3 already direct after S1; skill.go adds only a stdlib import)
- [ ] Did NOT create `resolve` / `ui` / `install.sh` / `README.md` / `completions/` / `skills/` (later milestones)

---

## Anti-Patterns to Avoid

- ❌ **Don't type-assert `fm.Metadata["keywords"]` as `[]string`.** yaml.v3
  decodes YAML lists into `[]any` (`[]interface{}`), NEVER `[]string`. A direct
  `.([]string)` PANICS on every real skill. Use `toStringSlice`. Verified §1/§2.
- ❌ **Don't drop the `case string` arm from toStringSlice.** A bare scalar
  `keywords: writing` decodes to a `string`, and without that arm it collapses to
  `nil`, silently losing the value. Verified §1 (the item's 2nd scenario).
- ❌ **Don't use the single-return `.(string)` for category.** It panics when
  Metadata is nil or category is a non-string. Use the comma-ok form
  `category, _ := fm.Metadata["category"].(string)`. Verified §3.
- ❌ **Don't add `Skill` / `toStringSlice` / `newSkill` to `discover.go`.** S1's
  Level 4 greps `discover.go` to PROVE it has neither. Use a new file `skill.go`.
  Verified §6.
- ❌ **Don't repeat the `// Package discover` doc in `skill.go`.** Only one file
  per package should carry the package doc (S1's `discover.go`). A leading
  comment before `package` with no blank line becomes the package doc and
  shadows/dupe's S1's. Start `skill.go` with `package discover` directly.
- ❌ **Don't call `filepath.Abs` inside `newSkill`.** Inputs are already absolute
  (Index guarantees it; `t.TempDir()` is absolute). Abs-ing rewrites `Dir` away
  from the exact input and breaks `Dir == absDir`. Verified §4.
- ❌ **Don't drop `filepath.ToSlash`.** It's a no-op on Linux but REQUIRED for
  Windows correctness (PRD §7.1: separators normalized to `/`). Verified §4/§8.
- ❌ **Don't read `fm.HasFM` inside `newSkill`.** The `hasFM` param is authoritative
  (the item's signature). Use the passed value; the caller threads `fm.HasFM`.
  Verified §10.
- ❌ **Don't add yaml struct tags to `Skill`.** Skill is the indexed value type,
  never a `yaml.Unmarshal` target. Only `Frontmatter` is tagged. Verified §11.
- ❌ **Don't use `reflect.DeepEqual` for slice assertions in tests.** It has a
  nil-vs-empty footgun (`DeepEqual(nil, []string{})` is false). Use the
  `slicesEq` helper (len + element-wise); assert `== nil` explicitly where the
  contract demands a true nil. The repo uses no reflect. Verified §12.
- ❌ **Don't define `Index()` / `resolve` / `ui` / a `Body` field.** S5 owns
  `Index()`; M3 owns resolve; M2.T6 owns ui. Skill has exactly the 9 fields
  listed. Verified (architecture package map).
- ❌ **Don't expect a `go.mod` change.** yaml.v3 is ALREADY a direct dependency
  after S1. skill.go imports only stdlib (`path/filepath`), so it adds no module
  requirement. `go.mod`/`go.sum` stay byte-identical.

---

## Confidence Score

**9/10** — one-pass implementation success likelihood.

Rationale: every load-bearing decision (the `[]any`-never-`[]string` typing, the
bare-scalar→string case, the comma-ok category assertion, `filepath.Rel`+`ToSlash`
canonicalization with the documented `..`/no-error behavior, the new-file requirement
so S1's contract stays intact, the separate-`hasFM` param, the no-yaml-tags Skill,
the unused-`body` param compiling cleanly, and the reflect-free slice-comparison
helper) was **empirically executed** against the project's real yaml.v3 v3.0.1 in
its real Go 1.26.4 toolchain, and the exact `skill.go` and `skill_test.go` source is
provided verbatim in the Implementation Blueprint (the verification program used the
identical struct + toStringSlice + newSkill logic and produced exactly the asserted
outputs). The residual 1/10 is ordinary implementer-transcription risk (a field-name
typo, forgetting the `case string` arm, or editing `discover.go` instead of creating
`skill.go`) which the Level 4 grep-based contract check catches deterministically.
