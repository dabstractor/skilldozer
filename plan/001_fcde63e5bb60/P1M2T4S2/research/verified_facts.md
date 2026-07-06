# Verified Facts — P1.M2.T4.S2: `Skill` type + metadata extraction (`toStringSlice`, `BuildSkill`)

Every claim below was **executed** against the real `gopkg.in/yaml.v3 v3.0.1`
(the version pinned in the repo's `go.sum`) on `go1.26.4 linux/amd64`, using TWO
throwaway modules in `/tmp` (never touching the repo). The exact `Skill` struct +
`toStringSlice` + `BuildSkill` proposed for this PRP were compiled and run; raw
output is summarized per fact.

## Repo state at authoring time (read directly)

```
go version go1.26.4-X:nodwarf5 linux/amd64
go.mod     module github.com/dabstractor/skpp ; go 1.25 ; require gopkg.in/yaml.v3 v3.0.1
           (yaml.v3 is ALREADY DIRECT — the S1 indirect→direct flip landed; S2 adds NO module dep)
go.sum     yaml.v3 v3.0.1 present (unchanged by S2)
internal/discover/discover.go      S1 landed: Frontmatter(8 fields) + ParseFrontmatter + utf8BOM
internal/discover/discover_test.go S1 landed: 12 white-box tests (package discover)
main.go / main_test.go              M1.T3 landed (builds + tests green)
baseline: `go build ./...` OK ; `go test ./...` OK (skillsdir + discover + main all green)
```

## Run 1 — `toStringSlice` + `BuildSkill` unit behavior (`/tmp/skpp_skill_verify`)

A throwaway module with a minimal `Frontmatter` twin + the proposed `Skill` /
`toStringSlice` / `BuildSkill`, fed a **real** `yaml.Unmarshal` block (so the
`metadata.keywords` list is a genuine `[]any`, not a hand-built `[]string`).

```text
metadata[keywords] type=[]interface {} value=[]interface {}{"example", "demo", "skpp"}
FULL Skill={Dir:/abs/skills/example RelTag:example Name:example Description:d
            Keywords:[example demo skpp] Category:meta Aliases:[a b] HasFM:true
            SourceFile:/abs/skills/example/SKILL.md}
NILMETA Skill={Dir:/abs/skills/plain RelTag:plain Name: Description: Keywords:[]
               Category: Aliases:[] HasFM:false SourceFile:/abs/skills/plain/SKILL.md}
   ↑ Frontmatter{} (nil Metadata) DID NOT PANIC; all metadata zero-valued; SourceFile still computed.
TTS "nil"         -> len=0 []string(nil)
TTS "[]any-str"   -> len=2 []string{"a", "b"}
TTS "[]any-mixed" -> len=2 []string{"a", "b"}        # non-string elements (2, true) SKIPPED
TTS "[]any-empty" -> len=0 []string{}                 # present-but-empty -> non-nil empty
TTS "single-str"  -> len=1 []string{"solo"}
TTS "int"         -> len=0 []string(nil)
TTS "[]string"    -> len=2 []string{"x", "y"}         # defensive passthrough (yaml.v3 never emits this)
```

## Run 2 — full `ParseFrontmatter`→`BuildSkill` end-to-end (`/tmp/skpp_skill_e2e`)

A throwaway module containing a **faithful verbatim copy** of S1's `discover.go`
(`Frontmatter` all 8 fields + `ParseFrontmatter` + `utf8BOM`) PLUS the proposed
`Skill`/`toStringSlice`/`BuildSkill`. Writes a real `SKILL.md` with a folded-scalar
`description: >`, parses it, and builds a `Skill`.

```text
raw Description="Reference example skill for skpp.\n"   (folded scalar KEEPS trailing \n — S1 contract)
E2E Skill: Name="example" Category="meta" HasFM=true SourceFile="/tmp/skppe2e…/SKILL.md"
E2E Keywords=[example demo skpp] Aliases=[ex demo]
E2E Description ends with newline: true
E2E ALL ASSERTIONS PASS
```

## Decisions locked (each traceable to a run above)

1. **`type Skill struct` — 9 fields, NO yaml tags.** `Skill` is built
   programmatically by `BuildSkill`, never unmarshaled, so it carries no `yaml:`
   tags (unlike `Frontmatter`). Fields: `Dir`, `RelTag`, `Name`, `Description`,
   `Keywords []string`, `Category string`, `Aliases []string`, `HasFM bool`,
   `SourceFile`. Matches `architecture/go_architecture.md` verbatim. ✓ Run 1 FULL.
2. **`toStringSlice` asserts `[]any`→`[]string`.** yaml.v3 unmarshals YAML lists
   into `[]interface{}` (== `[]any`), NEVER `[]string` (re-confirmed: Run 1 prints
   `type=[]interface {}`). The `case []any:` branch builds a new `[]string`. ✓ Run 1.
3. **Non-string list elements are SKIPPED (lenient).** `[a, 2, b, true]`→`[a b]`.
   This matches PRD §7.3's "ignore what doesn't fit" leniency and needs no `fmt`.
   A pure-`int`/`map` value → `nil`. ✓ Run 1 (`[]any-mixed`, `int`).
4. **`nil` → `nil`; empty `[]any{}` → non-nil empty `[]string{}`.** Both are
   `len 0`. Callers MUST test with `len()`, not a nil check. Documented on `Skill`.
   ✓ Run 1 (`nil` vs `[]any-empty`).
5. **Single string → `[]string{s}`.** `keywords: writing` (a scalar, not a list)
   → `["writing"]`. Lenient: tolerates a non-list scalar. ✓ Run 1 (`single-str`).
6. **`case []string:` passthrough is defensive only.** yaml.v3 never produces it,
   but the branch costs nothing and guards against hand-built inputs. ✓ Run 1.
7. **`BuildSkill` is TOTAL: never errors, never panics — even on nil `Metadata`.**
   Reading a missing key from a nil `map[string]any` returns the zero value (`nil`),
   and the comma-ok type assertion `c, _ := m["category"].(string)` on `nil` yields
   `("", false)`. So a no-frontmatter skill (`Frontmatter{}`) builds fine with
   zero metadata + `HasFM=false` + `SourceFile` still computed. ✓ Run 1 NILMETA.
8. **`category` uses the comma-ok assertion (`_ , ok`), NOT a bare `.(string)`.**
   A bare assertion on a missing/nil key would PANIC. `category, _ :=
   fm.Metadata["category"].(string)` is safe and returns `""` when absent. Matches
   `architecture/go_architecture.md` exactly. ✓ Run 1 (FULL→"meta", NILMETA→"").
9. **`SourceFile = filepath.Join(dir, "SKILL.md")`.** Derived from `Dir` (the
   architecture note says `Dir + "/SKILL.md"`); `filepath.Join` cleans a trailing
   slash and is the idiomatic join. T5 does NOT pass or compute `SourceFile`.
   ✓ Run 1 + Run 2.
10. **Folded-scalar `description` flows through `BuildSkill` VERBATIM (trailing
    `\n` retained).** S1 returns `Description` verbatim; S2 copies it onto `Skill`
    unchanged. T10's 1024-char check trims if it wants visible length. ✓ Run 2.
11. **`BuildSkill` is the S1↔T5 boundary; it does NOT own error policy.** T5 calls
    `ParseFrontmatter` (gets `(Frontmatter, body, err)`); if `err != nil`
    (malformed YAML) T5 decides how to surface it to `check` (M4). `BuildSkill`
    works on ANY `Frontmatter`, including the `Frontmatter{}` from a read error or
    no-frontmatter file, so it stays out of T5's error-handling domain. ✓ Run 1/2.
12. **SCOPE: S2 owns `Skill` + `toStringSlice` + `BuildSkill` ONLY.** It does NOT
    implement `Index()` (the WalkDir scan — that is T5), does NOT touch
    `discover.go`/`discover_test.go` (S1-owned), and does NOT add any resolve/ui
    code. New files: `internal/discover/skill.go` + `skill_test.go`. S1's
    anti-pattern list explicitly assigned these three symbols to S2. ✓
13. **`skill.go` imports ONLY `path/filepath`.** No `fmt`, no `yaml.v3` (S1 already
    imports it in `discover.go`; the package shares it — `skill.go` references the
    `Frontmatter` type but does not call yaml). `path/filepath` is stdlib → NO
    `go.mod`/`go.sum` change for S2 (unlike S1's indirect→direct flip). ✓
14. **`skill_test.go` is white-box `package discover`** (mirrors
    `discover_test.go`/`skillsdir_test.go`): `t.TempDir`/`os.WriteFile` via the
    shared `writeSkill` helper (defined in `discover_test.go`, same package — DO
    NOT redefine it), plain `t.Errorf`/`t.Fatalf`, NO testify, NO `t.Parallel()`.
    Imports: `path/filepath`, `strings`, `testing`. ✓
15. **No symbol collision with `discover_test.go`.** S1's file defines `writeSkill`
    + `TestParseFrontmatter*` + `TestHasFMNotMappedFromKey`. S2's file defines
    `strEq`, `TestToStringSlice`, `TestBuildSkill*` — no overlap. Both compile in
    `package discover`. ✓

## Reproducing these runs

```bash
cd /home/dustin/projects/skpp
go test ./internal/discover/ -v        # S2's skill_test.go + S1's discover_test.go together
go doc ./internal/discover Skill       # exported Skill type + its godoc
go doc ./internal/discover BuildSkill  # exported constructor + its godoc
```

The throwaway verifiers lived at `/tmp/skpp_skill_verify` and `/tmp/skpp_skill_e2e`
(built during authoring; not part of the repo).
