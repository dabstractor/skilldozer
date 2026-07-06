# PRP — P1.M2.T4.S2: `internal/discover` — `Skill` type + metadata extraction (`toStringSlice`, `BuildSkill`)

> **Subtask:** P1.M2.T4.S2 — the SECOND subtask of T4 (frontmatter model & parser,
> PRD §7.3). It builds directly on **S1** (the `Frontmatter` type + `ParseFrontmatter`,
> landed & committed in `internal/discover/discover.go`) and feeds **T5** (the
> `Index()` walk, not yet built) and **T6/T7/T10** (`--list`, `resolve`, `check`).
>
> **Scope:** create `internal/discover/skill.go` (`package discover`) and its
> white-box test `internal/discover/skill_test.go`. Define the `Skill` struct
> (the typed on-disk skill record, PRD §7.1) and the metadata-extraction
> constructor that turns S1's `Frontmatter` into it: `toStringSlice` (normalizes
> yaml.v3's `[]any` metadata lists into `[]string`) and `BuildSkill` (assembles a
> `Skill` from walk-derived location info + parsed frontmatter).
>
> **SCOPE DECISION (authoritative — see verified_facts.md §12):** S1's anti-pattern
> list explicitly assigned `type Skill`, `toStringSlice`, and `BuildSkill` to S2,
> and reserved `Index()` for T5. This subtask owns EXACTLY those three symbols.
> It does **NOT** implement `Index()` (the `WalkDir` scan — T5), does **NOT** touch
> `discover.go` / `discover_test.go` (S1-owned — see the "no-touch" gate in Level 4),
> and does **NOT** add any `resolve`/`ui`/`main` code (later milestones). The
> non-overlapping split: **S1 = `Frontmatter` + `ParseFrontmatter`; S2 = `Skill` +
> `toStringSlice` + `BuildSkill`; T5 = `Index()`.**
>
> **DEPENDENCY:** depends on S1's `Frontmatter` type (same package, already
> landed). `internal/discover` remains a LEAF library — `skill.go` imports ONLY
> `path/filepath` (stdlib). It does NOT import `yaml.v3` (S1 already does, in
> `discover.go`; the package shares it), so S2 adds **NO** module dependency →
> `go.mod`/`go.sum` are UNCHANGED (unlike S1's indirect→direct flip).
>
> **NOTE (main.go):** M1.T3 (`main.go` + `main_test.go`) is landed, green, and
> irrelevant here — `discover` is a leaf library with no dependency on `main.go`.
> Do NOT touch `main.go`/`main_test.go`.

---

## Goal

**Feature Goal**: Define the typed skill record that the rest of skpp operates on,
and the single constructor that produces it from S1's parsed frontmatter. After
S2, every downstream consumer (T5 `Index`, T6 `--list`, T7 `resolve`, T10 `check`)
works with `[]Skill` and typed `[]string` fields instead of raw `map[string]any`.
`BuildSkill(dir, relTag, fm)` extracts the skpp conventions (`keywords`/
`category`/`aliases`) from `Frontmatter.Metadata` — which yaml.v3 delivers as
`[]interface{}` (== `[]any`), never `[]string` — and assembles a fully-populated
`Skill`. The extraction is **total**: it never errors or panics, even when
frontmatter is absent (`Metadata` is nil), so a no-frontmatter skill still gets a
`Skill` (`HasFM=false`, zero metadata) that T5 can resolve by directory.

**Deliverable**: Two NEW files (no other files touched):
1. `internal/discover/skill.go` — `package discover`; `type Skill struct` (9
   fields, no yaml tags); `func toStringSlice(v any) []string` (unexported);
   `func BuildSkill(dir, relTag string, fm Frontmatter) Skill`.
2. `internal/discover/skill_test.go` — `package discover` (white-box); a
   table-driven `TestToStringSlice` (8 cases) + `TestBuildSkill*` cases (full
   extraction, nil-metadata safety, metadata-without-conventions, `SourceFile`
   derivation, and a real `ParseFrontmatter`→`BuildSkill` end-to-end).

**Success Definition**: `gofmt -l internal/discover/*.go` is silent; `go vet
./internal/discover/` is clean; `go build ./...` and `go test ./...` pass (S1's
12 tests + S2's new tests, all green); `go mod tidy` is a **no-op** (go.mod/go.sum
unchanged). `go doc ./internal/discover Skill` and `go doc ./internal/discover
BuildSkill` show exported, documented symbols. No `Index()`, no touch to
`discover.go`/`discover_test.go`/`main.go`/`skillsdir`/`resolve`/`ui`.

---

## Why

- This subtask **closes the S1↔T5 seam.** S1 parses frontmatter into
  `Frontmatter` (with `Metadata map[string]any`); T5 needs typed `[]Skill`. S2 is
  the single place that bridges them, so the `[]any`→`[]string` normalization and
  the `Skill` field semantics are locked in ONE spot instead of leaking into T5.
- It **locks the `[]any` trap before it bites.** yaml.v3 unmarshals YAML lists
  into `[]interface{}`, never `[]string` (re-verified: `metadata[keywords]
  type=[]interface{}`). If `resolve`/`search` ever did `fm.Metadata["keywords"].([]string)`
  they would panic. `toStringSlice` centralizes the safe assertion. (verified_facts §2.)
- It **locks the nil-metadata safety.** A `SKILL.md` with no `---` block (or a
  read error) yields `Frontmatter{}` with `Metadata == nil`. `BuildSkill` must
  build a usable `Skill` from that without panicking. Reading a missing key from a
  nil map + a comma-ok type assertion is safe; a BARE `.(string)` on nil would
  PANIC. Verified (verified_facts §7, §8).
- It **keeps `go.mod` clean.** Unlike S1 (which flipped yaml.v3 indirect→direct),
  S2 imports only stdlib (`path/filepath`). `go mod tidy` changes nothing.

---

## What

Two new files in the existing `package discover` (`internal/discover/`):

1. **`type Skill struct`** — the typed on-disk skill record (PRD §7.1). Built by
   `BuildSkill`, never unmarshaled → **no yaml tags** (unlike `Frontmatter`).
   Fields (exact, per `architecture/go_architecture.md`):
   - `Dir string` — absolute path of the skill directory.
   - `RelTag string` — dir path relative to the skills dir, OS separators
     normalized to `/` (the **canonical tag**, PRD §7.2 step 1). T5 computes it;
     S2 just carries it.
   - `Name string` — frontmatter `name` (`""` if absent).
   - `Description string` — frontmatter `description` (`""` if absent), copied
     **verbatim** incl. a folded-scalar trailing newline (S1 contract; T10 trims).
   - `Keywords []string` — `metadata.keywords` (`nil` if absent/non-list).
   - `Category string` — `metadata.category` (`""` if absent).
   - `Aliases []string` — `metadata.aliases` (`nil` if absent/non-list).
   - `HasFM bool` — `false` if `SKILL.md` had no `---` block (copied from S1).
   - `SourceFile string` — absolute path to `SKILL.md` (== `filepath.Join(Dir,
     "SKILL.md")`; derived, not passed by T5).
2. **`func toStringSlice(v any) []string`** (unexported) — normalizes a
   `Metadata` value to `[]string`:
   - `nil` → `nil`.
   - `[]any` → `[]string`, **non-string elements silently skipped** (lenient).
   - `[]string` → returned as-is (defensive; yaml.v3 never emits this).
   - `string` → `[]string{s}` (lenient: a scalar where a list is expected).
   - anything else → `nil`.
   - A present-but-empty list (`[]any{}`) → non-nil empty `[]string{}`; an absent
     field → `nil`. Both are `len 0` → callers MUST test with `len()`, not nil.
3. **`func BuildSkill(dir, relTag string, fm Frontmatter) Skill`** — the
   constructor. Extracts `keywords`/`aliases` via `toStringSlice`, `category` via
   the **comma-ok** `.(string)` assertion, copies `Name`/`Description`/`HasFM`,
   and derives `SourceFile` via `filepath.Join(dir, "SKILL.md")`. **Total**: no
   error return, no panic on nil `Metadata`.

### Success Criteria

- [ ] `internal/discover/skill.go` is `package discover` with EXACTLY: `type Skill
      struct` (9 fields, NO yaml tags), `func toStringSlice(v any) []string`,
      `func BuildSkill(dir, relTag string, fm Frontmatter) Skill`.
- [ ] `skill.go` imports ONLY `path/filepath` (no `fmt`, no `yaml.v3`, no `strings`).
- [ ] `toStringSlice(nil)`→`nil`; `[]any{"a","b"}`→`["a","b"]`;
      `[]any{"a",2,"b"}`→`["a","b"]` (non-strings skipped); `"solo"`→`["solo"]`;
      `42`→`nil`; `[]string{"x","y"}`→passthrough; `[]any{}`→empty (len 0).
- [ ] `BuildSkill` on a full `Frontmatter` populates all 9 fields correctly:
      `Keywords`/`Aliases` as `[]string` from the `[]any` metadata, `Category` as
      string, `HasFM` copied, `SourceFile == filepath.Join(dir, "SKILL.md")`.
- [ ] `BuildSkill(Frontmatter{})` (nil `Metadata`) does NOT panic: zero metadata,
      `HasFM=false`, `SourceFile` still computed. (Critical — nil-map read +
      comma-ok assertion must be safe.)
- [ ] `BuildSkill` with `Metadata` present but no `keywords`/`category`/`aliases`
      → defaults (`nil`/`""`/`nil`), `Name`/`Description`/`HasFM` still set.
- [ ] A real folded-scalar `description: >` flows `ParseFrontmatter`→`BuildSkill`
      verbatim (trailing `\n` retained on `Skill.Description`).
- [ ] `skill_test.go` is white-box `package discover`; reuses `writeSkill` from
      `discover_test.go` (does NOT redefine it); NO testify, NO `t.Parallel()`.
- [ ] `gofmt -l` silent; `go vet ./internal/discover/` clean; `go build ./...` +
      `go test ./...` pass (S1's 12 + S2's new); `go mod tidy` no-op (go.mod/go.sum
      unchanged); `discover.go`/`discover_test.go`/`main*`/`skillsdir*` untouched.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `skill.go` and `skill_test.go` is given verbatim in the
Implementation Blueprint (gofmt-clean, compiles as-is — the algorithm was compiled
and run in TWO throwaway `/tmp` modules during research). Every load-bearing
behavior was empirically verified against `yaml.v3 v3.0.1` on `go1.26.4`
(`research/verified_facts.md`): the `[]any`→`[]string` assertion (§2), non-string
skipping (§3), nil-vs-empty (§4), single-string (§5), the nil-metadata panic-safety
(§7), the comma-ok `category` assertion (§8), `SourceFile` via `filepath.Join` (§9),
the folded-scalar verbatim copy (§10), the S1↔T5 boundary (§11), the scope split
(§12), the `path/filepath`-only import (§13), and the test conventions (§14/§15).
The consumed `Frontmatter` contract is S1's landed `discover.go` (read directly).
An implementer who knows Go but nothing about this repo can complete this in one
pass from this document._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M2T4S2/research/verified_facts.md
  why: "Proves (against yaml.v3 v3.0.1 on go1.26.4, two /tmp runs): (1) Skill =
        9 fields, NO yaml tags (built, not unmarshaled). (2) yaml.v3 lists arrive
        as []interface{} (== []any), NEVER []string -> toStringSlice asserts
        []any->[]string. (3) non-string list elements are SKIPPED (lenient). (4)
        nil->nil; empty []any{}->non-nil empty; both len 0 -> callers use len().
        (5) single string -> [s]. (6) []string passthrough is defensive only.
        (7) BuildSkill is TOTAL: nil Metadata does NOT panic (nil-map read returns
        zero value; comma-ok assertion on nil yields \"\",false). (8) category
        MUST use comma-ok assertion, NOT bare .(string) (would PANIC on nil/absent).
        (9) SourceFile = filepath.Join(dir,\"SKILL.md\"). (10) folded-scalar
        description copied VERBATIM (trailing \\n kept). (11) BuildSkill is the
        S1<->T5 seam; it owns NO error policy (T5 handles ParseFrontmatter errors).
        (12) SCOPE: S2 = Skill+toStringSlice+BuildSkill ONLY (no Index/T5). (13)
        skill.go imports ONLY path/filepath -> NO go.mod/go.sum change. (14/15)
        white-box package discover; reuse writeSkill from discover_test.go; no
        testify; no t.Parallel()."
  critical: "Do NOT implement Index() (T5 owns it). Do NOT touch discover.go or
             discover_test.go (S1 owns them). Do NOT add yaml tags to Skill (it is
             built, not unmarshaled). Do NOT use a bare type assertion for category
             (panics on nil) — use the comma-ok form."

# CONTRACT — the discover package design (the exact Skill struct + the
# toStringSlice/BuildSkill signatures + the data flow discover->resolve->ui)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "Locks the Skill struct verbatim (Dir/RelTag/Name/Description/Keywords/
        Category/Aliases/HasFM/SourceFile) and the 'metadata extraction' note:
        toStringSlice handles []any->[]string (single string, nil); category,_ =
        fm.Metadata[\"category\"].(string); aliases = toStringSlice(...). Also
        locks ParseFrontmatter(path) (fm Frontmatter, body string, err error) from
        S1 that BuildSkill consumes, and the data flow Index()->[]Skill->resolve."
  section: "Core types > internal/discover", "metadata extraction", "Data flow"

# PREDECESSOR — S1's landed code (the Frontmatter type BuildSkill consumes) + its
# exact contract (folded-scalar verbatim, HasFM semantics, []any metadata)
- file: internal/discover/discover.go
  why: "The Frontmatter struct (8 fields: Name/Description/License/Compatibility/
        Metadata map[string]any/AllowedTools string/DisableModelInvocation bool/
        HasFM bool yaml:\"-\") and ParseFrontmatter(path)(Frontmatter,body,error).
        BuildSkill reads ONLY Name/Description/Metadata/HasFM (all present). S1's
        doc guarantees: Description returned verbatim (folded '>' keeps trailing
        \\n); Metadata is map[string]any with lists as []any; HasFM=false when no
        --- block (and Metadata is then nil). READ-ONLY — do not modify."
  pattern: "S1's Frontmatter is the unmarshal target (yaml tags); Skill (S2) is the
            built record (no yaml tags). Same package, different roles."
  gotcha: "Frontmatter.Metadata is nil when there is no frontmatter block —
           BuildSkill must tolerate that (nil-map read is safe; see verified_facts §7)."

# PREDECESSOR RESEARCH — the []any fact and the leniency model (cross-checkable)
- file: plan/001_fcde63e5bb60/P1M2T4S1/research/verified_facts.md
  why: "§2: Metadata lists arrive as []interface{} (== []any), NEVER []string —
        this is WHY toStringSlice exists. §3: folded scalar keeps trailing \\n
        (BuildSkill copies it verbatim). §12: HasFM propagates into Skill.HasFM.
        §13: the S1/S2/T5 scope split (S2 = Skill + metadata extraction). All
        facts re-confirmed by S2's own runs."

# CONTRACT — the metadata field conventions (keywords/category/aliases under the
# spec-compliant metadata map) + the yaml.v3 schema
- file: plan/001_fcde63e5bb60/architecture/external_deps.md
  why: "§1: frontmatter field table (metadata is the spec's arbitrary map; the skpp
        conventions keywords/category/aliases live INSIDE it, so nothing is
        non-standard). Confirms yaml.v3 is the only third-party dep (already
        imported by S1; S2 adds none)."
  section: "1. Agent Skills specification"

# CONTRACT — the PRD sections this implements
- file: PRD.md
  why: "§7.1: the Skill fields Index() populates FROM Frontmatter (name,
        description, metadata.keywords/category/aliases) — S2's BuildSkill is the
        extraction. §7.2: RelTag is the canonical tag; Aliases feed step 4; Name
        feeds step 3; Keywords feed --search. §7.3: leniency (unknown keys ignored;
        missing optional keys -> defaults) — toStringSlice's skip-non-strings and
        nil->defaults embody this. §9: check consumes HasFM/Name/Description.
        §10: the metadata conventions (keywords/category/aliases). READ-ONLY."
  critical: "§7.3 'lenient' = ignore unknown keys / missing -> defaults. toStringSlice
             applies the same spirit to list ELEMENTS (skip non-strings) and to
             absent fields (nil/\"\")."

# REFERENCE — the repo's test convention (white-box, same-package, shared helpers)
- file: internal/discover/discover_test.go
  why: "Defines writeSkill(t, content) string — the fixture helper skill_test.go
        REUSES (same package; do NOT redefine it). Also the convention template:
        package discover (white-box), t.TempDir()+os.WriteFile, plain
        t.Errorf/t.Fatalf, NO testify, NO t.Parallel(). READ-ONLY — do not modify."
  pattern: "White-box test file; reuse cross-file helpers in the same package; build
            fixtures in t.TempDir(); assert with t.Errorf/t.Fatalf."

# URLS — the load-bearing library + spec
- url: https://pkg.go.dev/path/filepath#Join
  why: "filepath.Join(dir, \"SKILL.md\") — the idiomatic, separator-cleaning path
        join used to derive SourceFile. Cleans a trailing slash on dir."
- url: https://go.dev/ref/spec#Type_assertions
  why: "The comma-ok type assertion 'v, ok := x.(T)' — used for category (safe on
        nil/absent; returns \"\",false). A bare 'x.(T)' PANICS on a non-matching
        value (incl. nil), so the comma-ok form is mandatory here."
- url: https://agentskills.io/specification
  why: "The Agent Skills frontmatter spec (metadata is the spec'd optional arbitrary
        map; skpp's keywords/category/aliases live inside it, fully standard)."
```

### Current Codebase tree (S1 landed; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/discover/discover.go        # S1: Frontmatter(8 fields) + ParseFrontmatter + utf8BOM
internal/discover/discover_test.go   # S1: 12 white-box tests + writeSkill helper (package discover)
internal/skillsdir/skillsdir.go      # M1.T2: Source + Find + findEnv/findSibling/findWalkUp
internal/skillsdir/skillsdir_test.go # M1.T2 tests (white-box, package skillsdir)
main.go / main_test.go               # M1.T3: arg-parse + --path + --version (landed, green)

$ ls -A internal/discover/
discover.go  discover_test.go
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1
#         (yaml.v3 is DIRECT — S1's indirect->direct flip already landed; S2 changes NOTHING here)
# go.sum: yaml.v3 v3.0.1 (unchanged by S2)
# baseline: `go build ./...` OK ; `go test ./...` OK (skillsdir + discover + main all green)
# NO internal/resolve/, ui/. NO skills/ (P1.M6.T12). NO Index() anywhere yet (T5).
```

### Desired Codebase tree with files to be added

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/skillsdir/*, main.go,
│        main_test.go — ALL UNCHANGED; internal/discover/discover.go &
│        discover_test.go — UNCHANGED [S1-owned, see Level 4 gate])
└── internal/
    └── discover/
        ├── discover.go       # UNCHANGED (S1: Frontmatter + ParseFrontmatter + utf8BOM)
        ├── discover_test.go  # UNCHANGED (S1: 12 tests + writeSkill helper)
        ├── skill.go          # CREATE — Skill struct + toStringSlice + BuildSkill
        └── skill_test.go     # CREATE — white-box tests for toStringSlice + BuildSkill
```

| File (created) | Responsibility | Imports |
|---|---|---|
| `internal/discover/skill.go` | Define `Skill`; normalize `[]any`→`[]string` (`toStringSlice`); build `Skill` from location + `Frontmatter` (`BuildSkill`) | `path/filepath` |
| `internal/discover/skill_test.go` | White-box tests: `toStringSlice` table (8 cases) + `BuildSkill` (full / nil-metadata / no-conventions / `SourceFile` / end-to-end) | `path/filepath`, `strings`, `testing` |

**Two new files in the existing `internal/discover/`. Zero changes to `go.mod`,
`go.sum`, `discover.go`, `discover_test.go`, `main.go`, `main_test.go`, or any
other file. No `Index()`, no `resolve`/`ui`/`skills/`.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — yaml.v3 lists arrive as []interface{} (== []any), NEVER []string.
// "metadata: { keywords: [a, b] }" unmarshals into Metadata["keywords"] as
// []any{"a","b"}, NOT []string. If resolve/search ever did
// fm.Metadata["keywords"].([]string) it would PANIC. toStringSlice centralizes the
// safe []any->[]string assertion. Verified (research §2; run prints
// type=[]interface{}).
//   RIGHT: Keywords: toStringSlice(fm.Metadata["keywords"])
//   WRONG: fm.Metadata["keywords"].([]string)  // PANIC at runtime

// GOTCHA #2 — category MUST use the comma-ok assertion, NOT a bare .(string).
// A bare assertion panics when the value is absent or the wrong type. The
// metadata map is nil when there is no frontmatter block; reading a missing key
// from a nil map returns nil (safe), but a BARE nil.(string) PANICS. The comma-ok
// form yields ("", false) instead.
//   RIGHT: category, _ := fm.Metadata["category"].(string)   // "" when absent; no panic
//   WRONG: category := fm.Metadata["category"].(string)       // PANICS on nil/absent

// GOTCHA #3 — Reading a nil map is safe; BuildSkill must work on Frontmatter{}.
// A SKILL.md with no --- block (or a read error) yields Frontmatter{} with
// Metadata == nil. var m map[string]any; m["x"] returns the zero value (nil), no
// panic; combined with the comma-ok assertion (GOTCHA #2) this is fully safe. So
// BuildSkill never errors and never panics — it produces a Skill with zero
// metadata + HasFM=false + SourceFile still computed. Verified (research §7).
//   IMPLICATION: BuildSkill has NO error return. T5 calls it unconditionally.

// GOTCHA #4 — Skill has NO yaml tags (it is BUILT, not unmarshaled).
// Frontmatter (S1) is the unmarshal target and carries yaml tags. Skill is
// constructed by BuildSkill from already-parsed values, so it must NOT have yaml
// tags (they'd be meaningless and misleading). It's a plain data struct.
//   RIGHT: type Skill struct { Dir string; RelTag string; ... }   // no tags
//   WRONG: type Skill struct { Name string `yaml:"name"`; ... }   // Skill is never unmarshaled

// GOTCHA #5 — Non-string list elements are SKIPPED (lenient), not coerced.
// "keywords: [a, 2, b]" -> []any{"a", 2, "b"} -> toStringSlice -> ["a","b"]. The
// integer is dropped silently. This matches PRD §7.3's leniency ("ignore what
// doesn't fit") and avoids importing fmt to stringify. A pure int/map value -> nil.
// Verified (research §3).
//   RIGHT: if str, ok := e.(string); ok { out = append(out, str) }
//   WRONG: out = append(out, fmt.Sprintf("%v", e))  // coerces; needs fmt import; surprises

// GOTCHA #6 — nil vs empty: callers MUST use len(), not a nil check.
// Absent field -> nil ([]string(nil)); present-but-empty list ([]any{}) ->
// non-nil empty ([]string{}). Both have len 0. Code that does `if s.Keywords == nil`
// to mean "absent" is WRONG. Use `len(s.Keywords)`. Documented on the Skill type.
// Verified (research §4).
//   RIGHT: if len(s.Keywords) > 0 { ... }
//   WRONG: if s.Keywords != nil { ... }   // treats []any{}-sourced empty as "present"

// GOTCHA #7 — Folded-scalar description is copied VERBATIM (trailing \n kept).
// S1 returns Description verbatim ("desc: >" yields "...\n"). BuildSkill copies it
// onto Skill.Description unchanged. Do NOT TrimSpace it (would corrupt a "|"
// literal block and fight S1's contract). T10's 1024-char check trims if it wants
// the visible length. Verified end-to-end (research §10; run prints trailing \n).
//   RIGHT: Description: fm.Description
//   WRONG: Description: strings.TrimSpace(fm.Description)

// GOTCHA #8 — SourceFile is DERIVED from Dir, not passed by T5.
// The architecture note says SourceFile == "Dir + /SKILL.md". BuildSkill computes
// it via filepath.Join(dir, "SKILL.md") (cleans a trailing slash; idiomatic). T5
// calls BuildSkill(dir, relTag, fm) and never sets SourceFile itself. Verified
// (research §9).
//   RIGHT: SourceFile: filepath.Join(dir, "SKILL.md")
//   NOTE: do NOT take sourceFile as a BuildSkill param — minimal signature, one source of truth.

// GOTCHA #9 — []string passthrough in toStringSlice is DEFENSIVE ONLY.
// yaml.v3 NEVER produces []string (always []any). The `case []string:` branch
// exists only so a hand-built or future-typed Metadata value doesn't surprise us.
// It returns the slice as-is. Do not rely on yaml.v3 ever hitting this branch.
// Verified (research §6).

// GOTCHA #10 — BuildSkill owns NO error policy; that's T5's job.
// ParseFrontmatter returns (Frontmatter, body, err). On malformed YAML, err != nil
// and fm == Frontmatter{} (HasFM=false). T5 decides whether to surface that to
// `check` (M4), skip the skill, or build a HasFM=false Skill. BuildSkill just
// transforms whatever Frontmatter it gets. Do NOT add error handling to BuildSkill.
// Verified (research §11).
//   RIGHT (T5): fm, _, err := ParseFrontmatter(path); ... s := BuildSkill(dir, relTag, fm)

// GOTCHA #11 — Reuse writeSkill from discover_test.go; do NOT redefine it.
// skill_test.go is package discover (white-box), so it shares scope with
// discover_test.go, which already defines writeSkill(t, content) string. Redefining
// it in skill_test.go is a compile error ("redeclared"). Reuse it. Verified (§14).
//   RIGHT: path := writeSkill(t, `---\n...`)   // from discover_test.go
//   WRONG: func writeSkill(...) {...}          // redeclared -> build fails

// GOTCHA #12 — skill.go imports ONLY path/filepath. NO fmt, NO yaml.v3, NO strings.
// toStringSlice is pure (no formatting); BuildSkill uses filepath.Join (the only
// import); Skill is a struct. yaml.v3 is already imported by discover.go in the
// SAME package — skill.go references the Frontmatter type but never calls yaml, so
// it needs no yaml import. A dead import fails vet/build. Because no new MODULE is
// required, go.mod/go.sum are UNCHANGED (unlike S1). Verified (§13).
//   RIGHT: import "path/filepath"
//   WRONG: import ("fmt"; "gopkg.in/yaml.v3")  // fmt unused; yaml already in-package via discover.go

// GOTCHA #13 — Do NOT implement Index(), and do NOT touch S1's files.
// Index() (the WalkDir scan + relTag via filepath.Rel/ToSlash + sorting) is T5.
// discover.go and discover_test.go are S1's landed, green deliverables — modifying
// them risks S1's contract and is forbidden (Level 4 gate asserts they're
// unchanged). S2's deliverable lives entirely in skill.go + skill_test.go.
// Verified (§12).
```

---

## Implementation Blueprint

### Data model — `Skill` struct

No ORM/pydantic (this is Go). The "model" is the typed skill record (built, not
unmarshaled — note the absence of yaml tags, unlike S1's `Frontmatter`):

```go
// Skill is a resolved on-disk skill (PRD §7.1). Built by BuildSkill, never
// unmarshaled -> NO yaml tags (unlike Frontmatter).
type Skill struct {
	Dir         string   // absolute path of the skill directory
	RelTag      string   // dir relative to skills dir, separators normalized to '/'
	Name        string   // frontmatter name ("" if absent)
	Description string   // frontmatter description ("" if absent); verbatim incl. trailing \n
	Keywords    []string // metadata.keywords (nil if absent/non-list); test with len()
	Category    string   // metadata.category ("" if absent)
	Aliases     []string // metadata.aliases (nil if absent/non-list); test with len()
	HasFM       bool     // false if SKILL.md had no --- block (from S1's Frontmatter)
	SourceFile  string   // == filepath.Join(Dir, "SKILL.md"); derived, not passed by T5
}
```

### File 1 — `internal/discover/skill.go` (CREATE)

Create the file with EXACTLY this content (gofmt-clean; compiled and run in two
`/tmp` modules during research — every behavior below is verified):

```go
// skill.go defines the Skill type and the metadata-extraction constructor that
// turns parsed frontmatter (S1's Frontmatter) into the typed records the rest of
// skpp consumes (PRD §7.1, §7.3). This is the P1.M2.T4.S2 deliverable: discover.go
// (S1) owns Frontmatter + ParseFrontmatter; skill.go (S2) owns Skill + the
// []any->[]string normalization; the Index() walk that ties them together is T5.
package discover

import "path/filepath"

// Skill is a resolved on-disk skill (PRD §7.1). Index() (T5) returns a []Skill;
// resolve.Resolve (T7) matches tags against it; ui.Print* (T6) renders it.
//
// It is BUILT by BuildSkill, never unmarshaled, so it carries NO yaml tags
// (unlike S1's Frontmatter, which is the unmarshal target).
//
// Field semantics:
//   - Dir:         absolute path of the skill directory (e.g. /home/u/skills/foo).
//   - RelTag:      the skill dir path relative to the skills dir, with OS
//     separators normalized to '/' (e.g. "writing/reddit"). This is
//     the CANONICAL tag (PRD §7.2 step 1). T5 computes it; S2 carries it.
//   - Name:        frontmatter `name` ("" if the block or the field is absent).
//   - Description: frontmatter `description` ("" if absent). Copied VERBATIM from
//     Frontmatter, including a folded-scalar trailing newline (S1
//     contract); T10's 1024-char check trims if it wants visible length.
//   - Keywords:    metadata.keywords as []string (nil if absent/non-list).
//   - Category:    metadata.category as string ("" if absent).
//   - Aliases:     metadata.aliases as []string (nil if absent/non-list).
//   - HasFM:       false if SKILL.md had no --- frontmatter block (from S1).
//   - SourceFile:  absolute path to SKILL.md (== filepath.Join(Dir, "SKILL.md")).
//
// yaml.v3 delivers metadata lists as []interface{} (== []any), NEVER []string;
// toStringSlice normalizes them so the typed fields are convenient for
// resolve/search. An ABSENT field yields a nil slice; a PRESENT-but-empty list
// ([]any{}) yields a non-nil empty slice. Both have len 0 -> callers MUST test
// with len(), not a nil check.
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

// toStringSlice normalizes a frontmatter metadata value into []string.
//
// yaml.v3 unmarshals YAML lists into []interface{} (== []any), NEVER []string,
// regardless of element type. This helper asserts []any -> []string so the typed
// Skill fields are convenient for resolve/search. Behavior (verified):
//   - nil           -> nil
//   - []any         -> []string, with NON-STRING elements silently skipped
//     (lenient: a stray number in `keywords:` is dropped, matching
//     the "ignore what doesn't fit" leniency of PRD §7.3)
//   - []string      -> returned as-is (defensive; yaml.v3 never produces this)
//   - single string -> []string{s} (lenient: `keywords: writing` -> ["writing"])
//   - anything else -> nil
//
// A present-but-empty list ([]any{}) yields an empty non-nil []string; an absent
// field yields nil. Both have len 0 -> callers must use len(), not a nil check.
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	case []string:
		return s
	case string:
		return []string{s}
	default:
		return nil
	}
}

// BuildSkill assembles a Skill from walk-derived location info and the parsed
// frontmatter. It performs the PRD §7.1/§7.3 metadata extraction and is the
// boundary between S1 (Frontmatter/ParseFrontmatter) and T5 (Index walk).
//
// T5 calls it once per discovered skill dir:
//
//	fm, _, err := ParseFrontmatter(filepath.Join(dir, "SKILL.md"))
//	// T5 decides how to surface `err` (malformed YAML) to `check` (M4);
//	// BuildSkill itself never errors — it works on any Frontmatter, including
//	// Frontmatter{} (HasFM=false) from a no-frontmatter or read-error skill.
//	s := BuildSkill(dir, relTag, fm)
//
// It is TOTAL: no error return, no panic — even when fm.Metadata is nil (no
// frontmatter block). Reading a missing key from a nil map returns the zero value
// (nil), and the comma-ok type assertion on nil yields ("", false). So a
// no-frontmatter skill gets a Skill with zero metadata + HasFM=false that T5 can
// still resolve by directory. Verified empirically.
//
// category uses the comma-ok assertion deliberately: a BARE fm.Metadata["category"].(string)
// would PANIC on a nil/absent value. SourceFile is derived from Dir via
// filepath.Join (== Dir + "/SKILL.md"); T5 does not pass or compute it.
func BuildSkill(dir, relTag string, fm Frontmatter) Skill {
	category, _ := fm.Metadata["category"].(string) // nil-map read is safe; comma-ok -> "",false
	return Skill{
		Dir:         dir,
		RelTag:      relTag,
		Name:        fm.Name,
		Description: fm.Description,
		Keywords:    toStringSlice(fm.Metadata["keywords"]),
		Category:    category,
		Aliases:     toStringSlice(fm.Metadata["aliases"]),
		HasFM:       fm.HasFM,
		SourceFile:  filepath.Join(dir, "SKILL.md"),
	}
}
```

### File 2 — `internal/discover/skill_test.go` (CREATE, `package discover` white-box)

Create the file with EXACTLY this content. It mirrors the repo's test convention
(white-box same-package, `t.TempDir()`/`os.WriteFile` via the shared `writeSkill`
helper, plain `t.Errorf`/`t.Fatalf`, no testify, no `t.Parallel()`). It REUSES
`writeSkill` from `discover_test.go` (same package — do not redefine it).

```go
package discover

import (
	"path/filepath"
	"strings"
	"testing"
)

// NOTE: writeSkill is defined in discover_test.go (same package) and REUSED here.
// Do NOT redefine it. It writes content to a temp SKILL.md and returns its path.

// strEq compares two string slices by length + elements (nil and []string{} are
// both "empty" here; callers that care about nil-vs-empty assert len() directly).
func strEq(a, b []string) bool {
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

// --- toStringSlice (table-driven; verified empirically) ---

func TestToStringSlice(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"any-slice-of-strings", []any{"a", "b"}, []string{"a", "b"}},
		{"any-slice-skips-non-strings", []any{"a", 2, "b", 3.14, true}, []string{"a", "b"}},
		{"any-slice-empty", []any{}, []string{}}, // present-but-empty -> non-nil empty (len 0)
		{"string-slice-passthrough", []string{"x", "y"}, []string{"x", "y"}},
		{"single-string", "solo", []string{"solo"}},
		{"int", 42, nil},
		{"map", map[string]any{"a": 1}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := toStringSlice(c.in)
			// Compare by length + element (the meaningful contract). This treats
			// nil and []string{} as equal (both len 0), matching the documented
			// "callers use len()" rule rather than pinning nil-vs-empty.
			if len(got) != len(c.want) {
				t.Fatalf("%s: len(got)=%d; want %d (got=%#v want=%#v)", c.name, len(got), len(c.want), got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("%s: got[%d]=%q; want %q", c.name, i, got[i], c.want[i])
				}
			}
		})
	}
}

// --- BuildSkill ---

func TestBuildSkillFull(t *testing.T) {
	// metadata lists are []any (what yaml.v3 actually produces); a non-convention
	// key ("unrelated") must be ignored.
	fm := Frontmatter{
		Name:        "example",
		Description: "An example.",
		Metadata: map[string]any{
			"keywords":  []any{"example", "demo", "skpp"},
			"category":  "meta",
			"aliases":   []any{"ex", "demo-skill"},
			"unrelated": 7, // ignored
		},
		HasFM: true,
	}
	s := BuildSkill("/a/skills/example", "example", fm)
	if s.Dir != "/a/skills/example" {
		t.Errorf("Dir=%q; want /a/skills/example", s.Dir)
	}
	if s.RelTag != "example" {
		t.Errorf("RelTag=%q; want example", s.RelTag)
	}
	if s.Name != "example" {
		t.Errorf("Name=%q; want example", s.Name)
	}
	if s.Description != "An example." {
		t.Errorf("Description=%q; want 'An example.'", s.Description)
	}
	if !strEq(s.Keywords, []string{"example", "demo", "skpp"}) {
		t.Errorf("Keywords=%v; want [example demo skpp] (real []any path)", s.Keywords)
	}
	if s.Category != "meta" {
		t.Errorf("Category=%q; want meta", s.Category)
	}
	if !strEq(s.Aliases, []string{"ex", "demo-skill"}) {
		t.Errorf("Aliases=%v; want [ex demo-skill]", s.Aliases)
	}
	if !s.HasFM {
		t.Error("HasFM=false; want true")
	}
	if s.SourceFile != "/a/skills/example/SKILL.md" {
		t.Errorf("SourceFile=%q; want /a/skills/example/SKILL.md", s.SourceFile)
	}
}

// Frontmatter{} (nil Metadata, e.g. a SKILL.md with no --- block or a read error):
// BuildSkill MUST NOT panic and must yield zero metadata while still computing
// Dir/RelTag/SourceFile. (verified_facts §7 — the load-bearing nil-safety test.)
func TestBuildSkillNoFrontmatter(t *testing.T) {
	s := BuildSkill("/a/skills/plain", "plain", Frontmatter{})
	if s.HasFM {
		t.Error("HasFM=true; want false")
	}
	if s.Name != "" || s.Description != "" {
		t.Errorf("Name=%q Description=%q; want empty", s.Name, s.Description)
	}
	if s.Category != "" {
		t.Errorf("Category=%q; want empty", s.Category)
	}
	if len(s.Keywords) != 0 || len(s.Aliases) != 0 {
		t.Errorf("Keywords=%v Aliases=%v; want empty (len 0)", s.Keywords, s.Aliases)
	}
	if s.SourceFile != "/a/skills/plain/SKILL.md" {
		t.Errorf("SourceFile=%q; want /a/skills/plain/SKILL.md", s.SourceFile)
	}
}

// metadata present but keywords/category/aliases absent -> defaults.
func TestBuildSkillMetadataWithoutConventions(t *testing.T) {
	fm := Frontmatter{
		Name:        "x",
		Description: "y",
		HasFM:       true,
		Metadata:    map[string]any{"some-other-key": "whatever"},
	}
	s := BuildSkill("/a/skills/x", "x", fm)
	if len(s.Keywords) != 0 || len(s.Aliases) != 0 || s.Category != "" {
		t.Errorf("unexpected metadata: Keywords=%v Aliases=%v Category=%q; want empty",
			s.Keywords, s.Aliases, s.Category)
	}
	if !s.HasFM || s.Name != "x" || s.Description != "y" {
		t.Errorf("scalar passthrough wrong: HasFM=%v Name=%q Description=%q", s.HasFM, s.Name, s.Description)
	}
}

// SourceFile is derived from Dir via filepath.Join (cleans a trailing slash).
func TestBuildSkillSourceFile(t *testing.T) {
	cases := []struct{ dir, want string }{
		{"/a/skills/foo", "/a/skills/foo/SKILL.md"},
		{"/a/skills/writing/reddit", "/a/skills/writing/reddit/SKILL.md"},
		{"/a/skills/trailing/", "/a/skills/trailing/SKILL.md"}, // trailing slash cleaned by Join
	}
	for _, c := range cases {
		s := BuildSkill(c.dir, "x", Frontmatter{})
		if s.SourceFile != c.want {
			t.Errorf("dir=%q: SourceFile=%q; want %q", c.dir, s.SourceFile, c.want)
		}
	}
	// Direct contract: SourceFile == filepath.Join(dir, "SKILL.md").
	s := BuildSkill("/abs/x", "x", Frontmatter{})
	if s.SourceFile != filepath.Join("/abs/x", "SKILL.md") {
		t.Errorf("SourceFile=%q != filepath.Join=%q", s.SourceFile, filepath.Join("/abs/x", "SKILL.md"))
	}
}

// End-to-end: a real SKILL.md (PRD §11-shaped) parsed by S1's ParseFrontmatter,
// then built into a Skill. Proves the genuine []any -> []string path AND that the
// folded-scalar description is carried through verbatim (trailing \n retained).
func TestBuildSkillEndToEnd(t *testing.T) {
	path := writeSkill(t, "---\nname: example\ndescription: >\n  Reference example skill for skpp.\nmetadata:\n  keywords: [example, demo, skpp]\n  category: meta\n  aliases:\n    - ex\n    - demo\n---\n# body\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if !fm.HasFM {
		t.Fatal("HasFM=false; want true (valid --- block present)")
	}

	s := BuildSkill("/skills/example", "example", fm)
	if s.Name != "example" {
		t.Errorf("Name=%q; want example", s.Name)
	}
	if !strings.HasSuffix(s.Description, "Reference example skill for skpp.\n") {
		t.Errorf("Description=%q; want folded scalar ending with '...skpp.\\n' (S1 verbatim contract)", s.Description)
	}
	if !strEq(s.Keywords, []string{"example", "demo", "skpp"}) {
		t.Errorf("Keywords=%v; want [example demo skpp] (real []any path)", s.Keywords)
	}
	if s.Category != "meta" {
		t.Errorf("Category=%q; want meta", s.Category)
	}
	if !strEq(s.Aliases, []string{"ex", "demo"}) {
		t.Errorf("Aliases=%v; want [ex demo]", s.Aliases)
	}
	if !s.HasFM {
		t.Error("HasFM=false; want true")
	}
	if s.SourceFile != "/skills/example/SKILL.md" {
		t.Errorf("SourceFile=%q; want /skills/example/SKILL.md", s.SourceFile)
	}
}
```

> **Copy-paste correctness:** the two blueprint files above are gofmt-clean and
> compile as-is against the real `discover.go` (imports limited to exactly what
> each file uses: `skill.go` → `path/filepath`; `skill_test.go` →
> `path/filepath`/`strings`/`testing`). The algorithm was compiled and run in two
> `/tmp` modules during research — every asserted value traces to recorded output
> in `research/verified_facts.md`. Note `skill_test.go` reuses `writeSkill` from
> `discover_test.go` (same package) — it must NOT redefine it.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CREATE internal/discover/skill.go
  - WRITE: the exact content from the Blueprint (File 1).
  - CHECK: `package discover`; `import "path/filepath"` (ONLY);
           `type Skill struct` with 9 fields and NO yaml tags;
           `func toStringSlice(v any) []string` with the 5-case switch;
           `func BuildSkill(dir, relTag string, fm Frontmatter) Skill`.
  - GOTCHA: category MUST use comma-ok assertion (bare .(string) PANICS on nil).
            Non-string list elements are SKIPPED (no fmt). SourceFile via
            filepath.Join. Do NOT add yaml tags to Skill. Do NOT implement Index().

Task 2: CREATE internal/discover/skill_test.go
  - WRITE: the exact content from the Blueprint (File 2).
  - CHECK: `package discover` (white-box, NOT skill_test); imports path/filepath,
           strings, testing. REUSES writeSkill from discover_test.go (do NOT
           redefine it — redeclaration is a build error).
  - CHECK tests: TestToStringSlice (8 subtests), TestBuildSkillFull,
           TestBuildSkillNoFrontmatter (nil-safety), TestBuildSkillMetadataWithoutConventions,
           TestBuildSkillSourceFile, TestBuildSkillEndToEnd (folded-scalar verbatim).
  - GOTCHA: NO testify; NO t.Parallel() (repo convention). Compare slices by
            len()+elements (strEq), NOT reflect.DeepEqual (avoids pinning nil-vs-empty).

Task 3: FORMAT + VET + TIDY + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/discover/skill.go internal/discover/skill_test.go
  - COMMAND: gofmt -l internal/discover/*.go   # MUST print nothing
  - COMMAND: go vet ./internal/discover/       # MUST be clean
  - COMMAND: go mod tidy   # EXPECTED: a NO-OP (go.mod/go.sum unchanged — S2 adds no module dep)
  - COMMAND: go build ./...                    # exit 0 (whole module compiles)
  - COMMAND: go test ./internal/discover/ -v   # S2's NEW tests + S1's 12 tests ALL PASS
  - COMMAND: go test ./...                     # whole module green (skillsdir + discover + main)
  - EXPECT: zero errors, zero vet findings, gofmt silent, all tests pass, go.mod/go.sum unchanged.

Task 4: EXPORTED-API SMOKE TEST (Level 3 in Validation Loop)
  - COMMAND: go doc ./internal/discover Skill ; go doc ./internal/discover BuildSkill
  - EXPECT: both print their godoc (Skill fields + BuildSkill behavior); Skill and
            BuildSkill are exported; toStringSlice is NOT (lowercase, unexported).

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: skill.go has Skill+toStringSlice+BuildSkill only (no Index); imports
            exactly path/filepath; discover.go/discover_test.go UNCHANGED;
            go.mod/go.sum UNCHANGED; no main.go/skillsdir/resolve/ui touch; no skills/.
```

### Implementation Patterns & Key Details

```go
// PATTERN: the comma-ok type assertion for a maybe-absent metadata scalar.
//   category, _ := fm.Metadata["category"].(string)
// WHY: Metadata may be nil (no frontmatter) or the key may be absent or
//      non-string. A bare `fm.Metadata["category"].(string)` PANICS in any of
//      those cases. The comma-ok form yields ("", false) — safe and the right
//      default. Reading a missing key from a nil map is itself safe (returns the
//      zero value nil). Verified §7/§8.
//   WRONG: category := fm.Metadata["category"].(string)   // PANICS on nil/absent

// PATTERN: normalize []any -> []string once, centrally (toStringSlice).
//   case []any:
//       out := make([]string, 0, len(s))
//       for _, e := range s { if str, ok := e.(string); ok { out = append(out, str) } }
//       return out
// WHY: yaml.v3 ALWAYS produces []interface{} for YAML lists (verified). Asserting
//      []any->[]string in ONE place (toStringSlice) keeps Skill fields typed and
//      prevents panic-prone .([]string) assertions scattered across resolve/search.
//      Skipping non-string elements (no fmt) is the lenient PRD §7.3 spirit.

// PATTERN: a total constructor (no error return) that tolerates absent frontmatter.
//   func BuildSkill(dir, relTag string, fm Frontmatter) Skill { ... }
// WHY: T5 calls ParseFrontmatter, which can return (Frontmatter{}, err) for a
//      no-frontmatter or malformed file. T5 owns the ERROR policy (surface to
//      `check`, skip, or build a HasFM=false skill); BuildSkill just transforms.
//      Returning no error keeps T5's call site simple and makes BuildSkill usable
//      on ANY Frontmatter. Verified §11.

// PATTERN: derive SourceFile from Dir (single source of truth).
//   SourceFile: filepath.Join(dir, "SKILL.md"),
// WHY: the architecture note says SourceFile == "Dir + /SKILL.md". Computing it in
//      BuildSkill (not passing it in) means T5 has one less value to get right and
//      Dir/SourceFile can never disagree. filepath.Join cleans separators/trailing
//      slash. Verified §9.

// PATTERN: reuse cross-file test helpers in the same white-box package.
//   // skill_test.go (package discover) calls writeSkill from discover_test.go
//   path := writeSkill(t, "---\n...")
// WHY: Go compiles all *_test.go in a package together; helpers are shared. This
//      avoids duplicating writeSkill (DRY) and is the established repo convention
//      (skillsdir_test.go/discover_test.go both self-contain their helpers; here
//      the package already has one, so reuse it). Redefining it is a build error.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/discover/ contains discover.go (S1, UNCHANGED) + skill.go (S2, NEW)
    + discover_test.go (S1, UNCHANGED) + skill_test.go (S2, NEW). All `package
    discover` (internal -> unimportable outside the module; correct for a CLI).
  - skill.go imports ONLY path/filepath (stdlib). It references the in-package
    Frontmatter type from discover.go but does NOT import yaml.v3 (discover.go
    already does; the package shares it).
  - exposes (exported): type Skill, func BuildSkill. Unexported: toStringSlice.

go.mod / go.sum (UNCHANGED — verified_facts.md §13):
  - before/after: require gopkg.in/yaml.v3 v3.0.1   (yaml.v3 already DIRECT from S1)
  - go.sum: UNCHANGED. `go mod tidy` is a NO-OP (S2 adds no module; path/filepath is stdlib).

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into):
  - P1.M2.T5.S1 (Index walk): WalkDir(absSkillsDir); each dir containing SKILL.md
    -> fm, _, err := ParseFrontmatter(filepath.Join(dir, "SKILL.md")) ->
    s := BuildSkill(dir, relTag, fm). relTag = filepath.ToSlash(filepath.Rel(
    skillsDir, skillDir)). T5 owns: the walk, relTag computation, sorting by
    RelTag, and the err policy (malformed YAML -> surface to `check` M4). BuildSkill
    gives T5 a one-line call; T5 never touches frontmatter fields directly.
  - P1.M2.T6.S1 (--list): reads Skill.Name/RelTag/Description; shows "(missing)"
    when !HasFM or Description=="".
  - P1.M3.T7.S1 (resolve): §7.2 precedence over Skill.RelTag (exact), the
    basename of RelTag, Skill.Name, and Skill.Aliases.
  - P1.M4.T9.S1 (--search): substring over Skill.RelTag/Name/Description AND
    Skill.Keywords (the []string S2 normalized from []any).
  - P1.M4.T10.S1 (check): ERROR if !Skill.HasFM (no block) OR Name=="" OR
    Description==""; WARN if len(TrimSpace(Description)) > 1024; ERROR on
    duplicate Skill.Name.

NO CHANGES TO:
  - go.mod / go.sum (S2 adds no module)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned)
  - internal/discover/discover.go + discover_test.go (S1-owned — Level 4 gate asserts unchanged)
  - internal/skillsdir/* (M1-owned)
  - main.go / main_test.go (M1.T3-owned)
  - any other package or file (resolve/ui are later milestones; skills/ is P1.M6.T12;
    Index() is T5 — do NOT create it here)
```

---

## Validation Loop

### Level 1: Format, vet, tidy, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass).
gofmt -w internal/discover/skill.go internal/discover/skill_test.go
test -z "$(gofmt -l internal/discover/*.go)" \
  || { echo "FAIL: gofmt found unformatted files"; gofmt -d internal/discover/; exit 1; }
echo "gofmt OK"

# Vet the package (skill.go + discover.go together).
go vet ./internal/discover/ || { echo "FAIL: go vet ./internal/discover/"; exit 1; }
echo "go vet OK"

# Tidy: EXPECTED no-op for S2 (path/filepath is stdlib; yaml.v3 already direct).
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null \
  && echo "go.mod/go.sum unchanged OK" \
  || { echo "FAIL: go.mod/go.sum changed (S2 must not touch them)"; git diff go.mod go.sum; exit 1; }

# Build the whole module (compile check across packages).
go build ./... || { echo "FAIL: go build ./..."; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run discover tests verbosely — S2's NEW tests + S1's 12 tests together.
go test ./internal/discover/ -v || { echo "FAIL: go test ./internal/discover/ -v"; exit 1; }

# Targeted: the load-bearing S2 assertions (nil-safety, []any path, folded scalar).
go test ./internal/discover/ -run \
  'TestToStringSlice|TestBuildSkillFull|TestBuildSkillNoFrontmatter|TestBuildSkillMetadataWithoutConventions|TestBuildSkillSourceFile|TestBuildSkillEndToEnd' -v \
  || { echo "FAIL: load-bearing S2 tests"; exit 1; }

# Whole module still green (skillsdir + discover + main).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Exported-API smoke test (library subtask — no main to run)

S2 is a leaf library, so the "acceptance smoke" is confirming the exported API is
present, documented, and behaves via the end-to-end test (already in Level 2).

```bash
cd /home/dustin/projects/skpp

# Skill is exported with its field godoc.
go doc ./internal/discover Skill | grep -q 'type Skill struct' \
  || { echo "FAIL: Skill not exported/documented"; exit 1; }
go doc ./internal/discover Skill | grep -qE 'Keywords +\[\]string' \
  || { echo "FAIL: Skill.Keywords field missing from godoc"; exit 1; }

# BuildSkill is exported with its behavior godoc.
go doc ./internal/discover BuildSkill | grep -qE 'func BuildSkill\(dir, relTag string, fm Frontmatter\) Skill' \
  || { echo "FAIL: BuildSkill signature wrong/undocumented"; exit 1; }

# toStringSlice is UNEXPORTED (lowercase) — go doc must NOT find it as a symbol.
go doc ./internal/discover toStringSlice 2>/dev/null \
  && { echo "FAIL: toStringSlice should be unexported"; exit 1; } \
  || echo "toStringSlice correctly unexported"

# End-to-end behavior is covered by TestBuildSkillEndToEnd (Level 2). Re-run it
# alone to confirm the real ParseFrontmatter -> BuildSkill -> []string path.
go test ./internal/discover/ -run TestBuildSkillEndToEnd -v \
  || { echo "FAIL: end-to-end (folded scalar + []any path)"; exit 1; }
echo "Level 3 PASS"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# skill.go exists and is package discover
test -f internal/discover/skill.go || { echo "FAIL: skill.go missing"; exit 1; }
grep -q '^package discover' internal/discover/skill.go || { echo "FAIL: skill.go not package discover"; exit 1; }

# Skill struct: 9 fields, NO yaml tags.
grep -qE 'type Skill struct' internal/discover/skill.go || { echo "FAIL: Skill type missing"; exit 1; }
for f in 'Dir ' 'RelTag ' 'Name ' 'Description ' 'Keywords +\[\]string' 'Category +string' 'Aliases +\[\]string' 'HasFM +bool' 'SourceFile '; do
  grep -qE "$f" internal/discover/skill.go || { echo "FAIL: Skill field /$f/ missing"; exit 1; }
done
! grep -qE '\`yaml:' internal/discover/skill.go || { echo "FAIL: Skill must have NO yaml tags (built, not unmarshaled)"; exit 1; }

# toStringSlice: the 5-case switch + non-string skip.
grep -qE 'func toStringSlice\(v any\) \[\]string' internal/discover/skill.go || { echo "FAIL: toStringSlice signature"; exit 1; }
grep -q 'case nil:' internal/discover/skill.go || { echo "FAIL: nil case"; exit 1; }
grep -q 'case \[\]any:' internal/discover/skill.go || { echo "FAIL: []any case"; exit 1; }
grep -q 'case \[\]string:' internal/discover/skill.go || { echo "FAIL: []string defensive case"; exit 1; }
grep -q 'case string:' internal/discover/skill.go || { echo "FAIL: single-string case"; exit 1; }
grep -qE 'if str, ok := e\.\(string\); ok' internal/discover/skill.go || { echo "FAIL: non-string skip (comma-ok element assert)"; exit 1; }

# BuildSkill: signature + comma-ok category + filepath.Join SourceFile + verbatim Description.
grep -qE 'func BuildSkill\(dir, relTag string, fm Frontmatter\) Skill' internal/discover/skill.go \
  || { echo "FAIL: BuildSkill signature"; exit 1; }
grep -qE 'category, _ := fm\.Metadata\["category"\]\.\(string\)' internal/discover/skill.go \
  || { echo "FAIL: category must use comma-ok assertion (bare .(string) panics on nil)"; exit 1; }
grep -qE 'SourceFile: +filepath\.Join\(dir, "SKILL\.md"\)' internal/discover/skill.go \
  || { echo "FAIL: SourceFile must be filepath.Join(dir, \"SKILL.md\")"; exit 1; }
grep -qE 'Description: +fm\.Description' internal/discover/skill.go \
  || { echo "FAIL: Description must be copied verbatim (no TrimSpace)"; exit 1; }

# Imports are EXACTLY path/filepath.
imp="$(sed -n '/^import (/,/^)/p' internal/discover/skill.go)"
echo "$imp" | grep -q '"path/filepath"' || { echo "FAIL: missing import path/filepath"; exit 1; }
for ban in '"fmt"' '"strings"' '"gopkg.in/yaml.v3"' '"os"'; do
  echo "$imp" | grep -q "$ban" && { echo "FAIL: forbidden import $ban in skill.go"; exit 1; } || true
done
# (skill.go may use the single-import form `import "path/filepath"` — accept that too.)
grep -qE '^import "path/filepath"$' internal/discover/skill.go || \
  grep -q '"path/filepath"' internal/discover/skill.go || { echo "FAIL: path/filepath import not found"; exit 1; }

# SCOPE: NO Index() anywhere new; NO BuildSkill/toStringSlice accidentally in discover.go.
! grep -qE 'func Index' internal/discover/skill.go || { echo "FAIL: Index() must NOT exist (T5)"; exit 1; }
! grep -qE 'type Skill struct' internal/discover/discover.go || { echo "FAIL: Skill must live in skill.go, not discover.go"; exit 1; }
! grep -q 'toStringSlice' internal/discover/discover.go || { echo "FAIL: toStringSlice must live in skill.go"; exit 1; }

# skill_test.go is white-box package discover with the key tests.
test -f internal/discover/skill_test.go || { echo "FAIL: skill_test.go missing"; exit 1; }
grep -q '^package discover' internal/discover/skill_test.go || { echo "FAIL: skill_test.go must be package discover (white-box)"; exit 1; }
! grep -q 'func writeSkill' internal/discover/skill_test.go || { echo "FAIL: skill_test.go must NOT redefine writeSkill (reuse from discover_test.go)"; exit 1; }
! grep -q 't.Parallel' internal/discover/skill_test.go || { echo "FAIL: no t.Parallel() (repo convention)"; exit 1; }
for tn in TestToStringSlice TestBuildSkillFull TestBuildSkillNoFrontmatter TestBuildSkillMetadataWithoutConventions TestBuildSkillSourceFile TestBuildSkillEndToEnd; do
  grep -q "func $tn" internal/discover/skill_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# S1's files MUST be unchanged (git diff clean). go.mod/go.sum MUST be unchanged.
git diff --quiet internal/discover/discover.go 2>/dev/null      || { echo "FAIL: discover.go changed (S1-owned)"; exit 1; }
git diff --quiet internal/discover/discover_test.go 2>/dev/null || { echo "FAIL: discover_test.go changed (S1-owned)"; exit 1; }
git diff --quiet go.mod go.sum 2>/dev/null                       || { echo "FAIL: go.mod/go.sum changed (S2 must not touch them)"; exit 1; }

# MUST NOT have touched PRD.md / skillsdir / main.go (M1-owned).
git diff --quiet PRD.md 2>/dev/null                                 || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go 2>/dev/null        || { echo "FAIL: skillsdir.go changed"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir_test.go 2>/dev/null   || { echo "FAIL: skillsdir_test.go changed"; exit 1; }
git diff --quiet main.go main_test.go 2>/dev/null                   || { echo "FAIL: main.go/main_test.go modified (M1.T3-owned)"; exit 1; }

# MUST NOT have created later-milestone files/packages.
test ! -d internal/resolve || { echo "FAIL: resolve/ must not exist (M3)"; exit 1; }
test ! -d internal/ui      || { echo "FAIL: ui/ must not exist (M2.T6)"; exit 1; }
test ! -f install.sh       || { echo "FAIL: install.sh must not exist (M6)"; exit 1; }
test ! -f README.md        || { echo "FAIL: README.md must not exist (M6)"; exit 1; }
test ! -d skills           || { echo "FAIL: skills/ must not exist (P1.M6.T12)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l internal/discover/*.go` silent, `go vet ./internal/discover/` clean, `go mod tidy` is a **no-op** (go.mod/go.sum unchanged), `go build ./...` exit 0
- [ ] Level 2 PASS — `go test ./internal/discover/ -v` passes (S2's new tests + S1's 12); `go test ./...` whole module green
- [ ] Level 3 PASS — `Skill` and `BuildSkill` are exported + documented; `toStringSlice` is unexported; `TestBuildSkillEndToEnd` passes (folded scalar + real `[]any` path)
- [ ] Level 4 PASS — `Skill` has 9 fields and NO yaml tags; `toStringSlice` has the 5-case switch with non-string skip; `BuildSkill` uses comma-ok `category` + `filepath.Join` `SourceFile` + verbatim `Description`; `skill.go` imports only `path/filepath`; `discover.go`/`discover_test.go`/`go.mod`/`go.sum`/`main*`/`skillsdir*` unchanged; no `Index()`/`resolve`/`ui`/`skills/`

### Feature Validation
- [ ] `toStringSlice(nil)`→nil; `[]any{...}`→`[]string` (non-strings skipped); `"solo"`→`["solo"]`; `42`→nil; `[]any{}`→empty (len 0)
- [ ] `BuildSkill` on full frontmatter: all 9 fields correct; `Keywords`/`Aliases` as `[]string` from `[]any`; `Category` as string; `HasFM` copied; `SourceFile == filepath.Join(dir,"SKILL.md")`
- [ ] `BuildSkill(Frontmatter{})` (nil Metadata) does NOT panic: zero metadata, `HasFM=false`, `SourceFile` computed
- [ ] `BuildSkill` with metadata lacking conventions → defaults (`nil`/`""`/`nil`)
- [ ] Folded-scalar `description: >` carried `ParseFrontmatter`→`BuildSkill` verbatim (trailing `\n` on `Skill.Description`)

### Code Quality / Convention Validation
- [ ] `skill.go` is `package discover` in `internal/discover/`; imports limited to `path/filepath`
- [ ] `skill_test.go` is white-box `package discover`; reuses `writeSkill` from `discover_test.go`; no testify, no `t.Parallel()`
- [ ] Every exported symbol (`Skill`, `BuildSkill`) and the helper (`toStringSlice`) has godoc explaining its contract and the `[]any`/nil gotchas

### Scope Discipline
- [ ] Did NOT implement `Index()` (T5 owns it — verified_facts §12)
- [ ] Did NOT modify `discover.go` or `discover_test.go` (S1-owned), `go.mod`/`go.sum` (S2 adds no module), or any `main*`/`skillsdir*`/`PRD.md`/`tasks.json`
- [ ] Did NOT create `resolve`/`ui`/`install.sh`/`README.md`/`skills/`

---

## Anti-Patterns to Avoid

- ❌ **Don't use a bare type assertion for `category`.** `fm.Metadata["category"].(string)`
  PANICS when the value is nil (no frontmatter → `Metadata` is nil) or non-string. Use
  the comma-ok form `category, _ := fm.Metadata["category"].(string)` (→ `""` when
  absent). The nil-map READ itself is safe (returns the zero value); only the
  assertion is dangerous. (Verified §7/§8.)
- ❌ **Don't assert `[]string` on a metadata list.** yaml.v3 ALWAYS produces
  `[]interface{}` (== `[]any`) for YAML lists, never `[]string`. A `.([]string)`
  assertion panics. Run ALL list values through `toStringSlice`. (Verified §2.)
- ❌ **Don't add yaml tags to `Skill`.** `Skill` is built by `BuildSkill`, never
  unmarshaled. yaml tags belong only on `Frontmatter` (S1's unmarshal target). (Verified §1.)
- ❌ **Don't coerce non-string list elements.** `[a, 2, b]` → `["a","b"]` (skip the
  int). Using `fmt.Sprintf("%v", e)` would coerce (surprising) AND require importing
  `fmt`. Skipping matches PRD §7.3 leniency and keeps the import set to one. (Verified §3/§13.)
- ❌ **Don't TrimSpace the description.** `description: >` yields a value ending in
  `\n`; S1 returns it verbatim and S2 must copy it verbatim. Trimming would corrupt
  a `|` literal block and fight S1's contract; T10 trims if it wants visible length.
  (Verified §10.)
- ❌ **Don't make `BuildSkill` return an error.** It is total (works on any
  `Frontmatter`, incl. nil metadata). Error policy for malformed YAML belongs to T5
  (it calls `ParseFrontmatter` and decides how to surface `err` to `check`). (Verified §11.)
- ❌ **Don't take `sourceFile` as a `BuildSkill` parameter.** Derive it from `dir`
  via `filepath.Join(dir, "SKILL.md")` (single source of truth; matches the
  architecture's "Dir + /SKILL.md"). (Verified §9.)
- ❌ **Don't test slices with `reflect.DeepEqual` if you also want to be robust to
  nil-vs-empty.** `reflect.DeepEqual([]string(nil), []string{})` is `false`, which
  would pin the test to an unspecified distinction. Compare by `len()`+elements
  (the `strEq` helper); the documented contract is "callers use `len()`". (Verified §4/§6.)
- ❌ **Don't redefine `writeSkill` in `skill_test.go`.** It's already defined in
  `discover_test.go` (same `package discover`). Redefining it is a compile error
  ("redeclared in this block"). Reuse it. (Verified §14.)
- ❌ **Don't import `fmt`, `strings`, `yaml.v3`, or `os` in `skill.go`.**
  `toStringSlice` is pure; `BuildSkill` needs only `filepath.Join`; `Skill` is a
  struct. yaml.v3 is already imported by `discover.go` in the same package (skill.go
  references the `Frontmatter` TYPE but never calls yaml). Dead imports fail vet.
  Because no new MODULE is required, `go.mod`/`go.sum` stay unchanged. (Verified §13.)
- ❌ **Don't implement `Index()` or touch S1's files.** `Index()` is T5 (the
  WalkDir scan, relTag via `filepath.Rel`/`ToSlash`, sorting). `discover.go` and
  `discover_test.go` are S1's landed, green deliverables; modifying them is forbidden
  (Level 4 gate). S2's deliverable is exactly `skill.go` + `skill_test.go`. (Verified §12.)
- ❌ **Don't add `t.Parallel()` or testify.** The repo convention
  (`skillsdir_test.go`/`discover_test.go`) is white-box same-package, plain
  `t.Errorf`/`t.Fatalf`, no Parallel. Mirror it. (Verified §14/§15.)

---

## Confidence Score

**9/10** — one-pass implementation success likelihood.

Rationale: the exact `skill.go` and `skill_test.go` source is provided verbatim and
was already compiled and run in TWO throwaway `/tmp` modules (one minimal, one a
faithful verbatim copy of S1's `discover.go` + the proposed additions) against the
real `yaml.v3 v3.0.1` on `go1.26.4`; every asserted value — including the load-bearing
`[]any`→`[]string` path, the nil-metadata panic-safety, the comma-ok `category`
assertion, the `filepath.Join` `SourceFile`, and the folded-scalar verbatim copy
through the full `ParseFrontmatter`→`BuildSkill` pipeline — traces to recorded output
in `research/verified_facts.md`. The baseline is green (`go build ./...` + `go test
./...` pass; yaml.v3 already direct). `go.mod`/`go.sum` are unchanged (S2 adds only a
stdlib import). The only residual risk is the Level 4 import-check `grep` being
brittle to the single-line vs grouped `import` form — mitigated by the documented
fallback `grep`. The core logic is fully de-risked.
