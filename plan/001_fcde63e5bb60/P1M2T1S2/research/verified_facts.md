# Verified Facts — P1.M2.T4.S2: `Skill` type + metadata extraction (toStringSlice/newSkill)

Every claim below was **executed** against the real `gopkg.in/yaml.v3 v3.0.1` in
`go version go1.26.4-X:nodwarf5 linux/amd64` (the project's toolchain), using a
throwaway module in `/tmp` that required the SAME `yaml.v3 v3.0.1` already pinned
in the project's `go.sum` (cached in the module cache — no network). The exact
`toStringSlice` type-switch algorithm and the `newSkill` field-extraction logic
proposed for this PRP were compiled and run against 6 frontmatter fixtures +
`filepath.Rel`/`ToSlash` cases. Raw output is summarized per fact.

This subtask CONSUMES the contract S1 produces (P1.M2.T1.S1):
`discover.Frontmatter` (Name, Description, Metadata `map[string]any`, HasFM) and
`discover.ParseFrontmatter`. S1's own `research/verified_facts.md` already proved
the YAML-decoding facts that feed this subtask (lists → `[]any`, scalars →
`string`, unknown-keys leniency, BOM/CRLF handling). The facts below cover the
**NEW** surface S2 owns: `toStringSlice`, `newSkill`, and the `Skill` struct.

Repo state at authoring time (read directly):

```
go.mod     module github.com/dabstractor/skpp ; go 1.25 ; require gopkg.in/yaml.v3 v3.0.1 // indirect
go.sum     yaml.v3 v3.0.1 checksums present (no network needed)
internal/  skillsdir/ (M1 landed). discover/ does NOT exist yet — S1 creates it
           (Frontmatter + ParseFrontmatter). S2 runs AFTER S1 lands discover.go.
module cache: /home/dustin/go/pkg/mod/gopkg.in/yaml.v3@v3.0.1 (present)
```

---

## §1 — yaml.v3 decodes a BARE SCALAR metadata value to a `string` (not `[]any`)

**Fixture** (excerpt):
```yaml
metadata:
  keywords: writing          # bare scalar, NOT a list
```

**Result** (`%T` of `fm.Metadata["keywords"]`):
```text
[bare-scalar-kw] Metadata=map[string]interface {}{"keywords":"writing"}
    keywords raw=string "writing" -> toStringSlice=[]string{"writing"}
```

**Decision locked**: `metadata.keywords` (and `aliases`) can arrive as EITHER a
yaml list (`[a,b,c]` or block form → `[]any`) OR a bare scalar (`writing` →
`string`). The item's "also accept a bare string (single value)" requirement is
therefore real and load-bearing: `toStringSlice` MUST have a `case string:`
branch that wraps the single value into `[]string{t}`. Verified the bare-scalar
case yields exactly `[]string{"writing"}` via the proposed helper. This is the
second of the three test cases the item specifies.

---

## §2 — `toStringSlice` type-switch behavior (all 6 input shapes verified)

The proposed helper:

```go
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

**Verified outputs** for each input shape (raw program output):

| Input `v` | `%T` of `v` | `toStringSlice(v)` | nil? |
|---|---|---|---|
| `nil` (key absent / Metadata nil) | `<nil>` | `[]string(nil)` | yes |
| `"writing"` (bare scalar) | `string` | `[]string{"writing"}` | no |
| `[]any{"a","b","c"}` (flow/block list of strings) | `[]interface {}` | `[]string{"a","b","c"}` | no |
| `[]any{"a", 1, "b"}` (mixed: string + int + string) | `[]interface {}` | `[]string{"a","b"}` | no |
| `[]any{}` (empty flow list `[]`) | `[]interface {}` | `[]string{}` | no (empty, non-nil) |
| `7` (bare number, e.g. `category: 7` fed to keywords) | `int` | `[]string(nil)` | yes (default branch) |

**Decisions locked**:
- `nil`/missing → `nil` (matches item: "nil/missing -> nil"; matches architecture
  Skill comment "Keywords // metadata.keywords, else nil").
- bare `string` → `[]string{t}` (single-element slice; §1).
- `[]any` → each element type-asserted to `string`; **non-string elements are
  silently skipped** (lenient — a `keywords: [a, 1, b]` yields `["a","b"]`, no
  panic, no `fmt.Sprintf` coercion). This matches skpp's overall leniency and the
  yaml.v3 reality that a list slot can hold any scalar type.
- `[]any{}` (explicit empty list `keywords: []`) → `[]string{}` (empty, NON-nil).
  Documented edge: `len()` is 0 either way, so it is indistinguishable from `nil`
  for all downstream consumers (resolve checks `range`/`len`; ui prints). The
  item's three required tests do NOT include the empty-list case, so this is a
  documented behavioral note, not a tested contract. If a future caller needs
  nil-normalization, change the `case []any:` to `if len(out) == 0 { return nil }`.
- unexpected type (`int`, `map`, etc.) → `nil` via `default` (lenient, no panic).

---

## §3 — `category` type-assertion is safe on nil maps and non-string values

The item specifies `Category=Metadata["category"].(string)`. Using the comma-ok
form `category, _ := fm.Metadata["category"].(string)`:

**Verified**:
- `metadata.category: x` (scalar) → `"x"`. (S1 §2 already proved this.)
- `metadata.category: 7` (number) → comma-ok returns `("", false)` → `""`.
- `Metadata == nil` (no metadata map at all) → indexing a nil map returns the
  zero value `nil`; `nil.(string)` comma-ok → `("", false)` → `""`. **No panic.**

**Decision locked**: compute `category, _ := fm.Metadata["category"].(string)`
BEFORE the struct literal, then `Category: category`. Never use the single-return
`.(string)` (panics on nil/non-string). A missing or non-string category quietly
becomes `""`, matching the architecture Skill comment "Category // metadata.category,
else \"\"".

---

## §4 — `filepath.Rel` + `filepath.ToSlash` produce the canonical `relTag`; `Rel` rarely errors

**Verified** (`skillsDir = /home/dustin/projects/skpp/skills`):

| `absDir` | `filepath.Rel(skillsDir, absDir)` | `filepath.ToSlash(rel)` |
|---|---|---|
| `…/skills/foo` | `"foo"` | `"foo"` |
| `…/skills/writing/reddit` | `"writing/reddit"` | `"writing/reddit"` |
| `…/skills/a/b/c` | `"a/b/c"` | `"a/b/c"` |
| `/totally/elsewhere/foo` (OUT of tree) | `"../../../../../totally/elsewhere/foo"` (**err == `<nil>`!**) | (same, with `..`) |

**Decisions locked**:
- For IN-tree skills (the only case `Index()` ever builds), `filepath.Rel` yields
  the clean relative path and `filepath.ToSlash` normalizes OS separators to `/`.
  On Linux `/` is already the separator, so `ToSlash` is a no-op here — but it is
  REQUIRED for Windows correctness (PRD §7.1: "OS separators normalized to `/`").
  Keep it; it costs nothing and is the spec'd normalization.
- **`filepath.Rel` does NOT error for out-of-tree paths** — it returns a relative
  path with leading `..` components and a nil error. So the defensive `if err != nil`
  guard is for genuinely rare cases (cross-volume roots, or an empty/unresolvable
  path). Because `Index()` only passes `absDir`s it walked *under* `skillsDir`,
  this guard never fires in production. On the off chance it does, fall back to
  `filepath.ToSlash(filepath.Base(absDir))` so `RelTag` is never an empty string
  (a non-empty tag keeps downstream `--list`/resolution sane). This is a defensive
  guard, documented as such — not a tested contract.
- Do NOT call `filepath.Abs` inside `newSkill`: the contract is that `absDir` and
  `skillsDir` are ALREADY absolute (Index guarantees it; `t.TempDir()` is absolute
  in tests). Abs-ing internally would change `Dir` away from the exact input and
  break the `Dir == absDir` assertion.

---

## §5 — `SourceFile` = `filepath.Join(absDir, "SKILL.md")`

**Verified**: `filepath.Join("/x/y/foo", "SKILL.md")` → `"/x/y/foo/SKILL.md"`.

**Decision locked**: build `SourceFile` with `filepath.Join(absDir, "SKILL.md")`
NOT the literal string concat `absDir + "/SKILL.md"` the architecture comment
shows. `filepath.Join` is OS-correct (uses `os.PathSeparator`, cleans `..`, avoids
double separators) and produces byte-identical output to the concat on Linux. The
architecture's `Dir + "/SKILL.md"` is a shorthand; `filepath.Join` is the
idiomatic, portable equivalent. (Both are acceptable; `Join` is preferred.)

---

## §6 — `newSkill` goes in a SEPARATE file `skill.go` (not `discover.go`)

S1's Level 4 validation (P1.M2.T1.S1) asserts, on `internal/discover/discover.go`:

```bash
! grep -q 'type Skill ' internal/discover/discover.go
! grep -q 'toStringSlice' internal/discover/discover.go
```

i.e. S1 PROVES `discover.go` contains neither `Skill` nor `toStringSlice`. If S2
added those symbols to `discover.go`, re-running S1's contract check would FAIL.
Therefore S2 MUST place `Skill`, `toStringSlice`, and `newSkill` in a **new file**
`internal/discover/skill.go` (and its tests in `skill_test.go`). This also matches
S1's own package-doc statement: "This file (P1.M2.T4.S1) implements the frontmatter
MODEL and the lenient fence-slicing PARSER only... the Skill struct + metadata
extraction land in P1.M2.T4.S2." Separate file = the contract-respecting choice.

`skill.go` does NOT repeat the `// Package discover` doc comment — only ONE file
in a package should carry the package doc (S1's `discover.go` does). `skill.go`
starts directly with `package discover`; per-symbol doc comments carry the
documentation. (A leading comment immediately before `package` with no blank line
would become the package doc and shadow/dupe S1's — avoid.)

---

## §7 — Tests MUST be white-box `package discover` (both `newSkill` and `toStringSlice` are unexported)

The item specifies lowercase `func toStringSlice(...)` and `func newSkill(...)`
— both UNEXPORTED. To call them directly from a test, the test file must be in the
SAME package: `package discover` (white-box), mirroring S1's `discover_test.go`
and the repo's `internal/skillsdir/skillsdir_test.go` (both white-box). An
external `package discover_test` (black-box) CANNOT reference `newSkill`/
`toStringSlice`. Confirmed by Go's export rules. So `skill_test.go` is
`package discover`.

White-box also lets the test build `Frontmatter` literals directly and assert on
unexported fields if ever needed (none required here — all Skill fields are
exported). Follows the established repo convention (no `t.Parallel()`, plain
`t.Errorf`/`t.Fatalf`, `t.TempDir()`/`os.WriteFile`, no testify).

---

## §8 — Imports for `skill.go` = ONLY `path/filepath`

`toStringSlice` needs NO imports (pure type-switch on `any`). `newSkill` needs
`path/filepath` (Rel, ToSlash, Join, Base). No `os`, `bytes`, `strings`,
`gopkg.in/yaml.v3` — those are S1's parser concerns. So `skill.go`'s import block
is exactly:

```go
import "path/filepath"
```

This keeps `discover.go`'s import set (`bytes`, `os`, `strings`, `yaml.v3`) and
`skill.go`'s import set (`path/filepath`) disjoint — a clean separation that
S1's Level 4 import-exactness check (scoped to `discover.go`) continues to pass.

---

## §9 — The `body` param is accepted but unused; Go does NOT flag unused params

The item's constructor signature is `func newSkill(absDir, skillsDir string, fm
Frontmatter, body string, hasFM bool) Skill`. `Skill` has NO `Body` field
(architecture Skill struct has none; the item's field list has none). So `body`
is accepted but not stored. Go does NOT error on unused function PARAMETERS (only
unused local variables and imports). Verified: the proposed `newSkill` (which
ignores `body`) compiles cleanly under `go vet` + `gofmt` + `go build`.

**Decision locked**: accept `body` per the item's exact signature; do NOT store
it. Document in the doc comment: "`body` is the SKILL.md markdown body from
`ParseFrontmatter`; `Skill` has no body field today, so it is accepted only so
the caller can pass `ParseFrontmatter`'s full return triple — reserved for future
body-aware features (e.g. `check`)."

---

## §10 — The `hasFM` param is threaded separately from `fm.HasFM` (item contract)

The item lists `hasFM bool` as a SEPARATE parameter to `newSkill`, even though
`fm` already carries `fm.HasFM`. This is the item's authoritative signature.
Within `newSkill`, assign the PASSED `hasFM` to `Skill.HasFM` (do NOT read
`fm.HasFM`). The canonical call site (S5 `Index()`) passes `fm.HasFM` for this
argument: `newSkill(absDir, skillsDir, fm, body, fm.HasFM)`. Keeping them
separate decouples the constructor from the Frontmatter struct's internal flag
naming and matches the item verbatim. Tests pass `hasFM` directly to assert
propagation (`Skill.HasFM == hasFM`).

---

## §11 — `Skill` struct has NO yaml struct tags

`Skill` is the INDEXED representation (built by `newSkill` from a parsed
`Frontmatter`), NOT a YAML-decoding target. `yaml.Unmarshal` is NEVER called on a
`Skill`. Therefore `Skill`'s fields carry NO `` `yaml:"..."` `` tags — just plain
`string`/`[]string`/`bool` fields (architecture confirms: the Skill struct shown
in `go_architecture.md` has no tags). Only `Frontmatter` (S1) is yaml-tagged.
Adding tags to `Skill` would be dead/misleading metadata. Verified the proposed
tag-less struct compiles and is gofmt-clean.

---

## §12 — Comparing `[]string` slices in tests: avoid `reflect`, use a len-normalizing helper

The repo's test style (skillsdir_test.go, S1 discover_test.go) uses plain
`t.Errorf` field checks and `strings.Contains`, NOT `reflect.DeepEqual`. For S2
tests comparing `Keywords`/`Aliases` ([]string), `reflect.DeepEqual` has a
nil-vs-empty footgun (`DeepEqual(nil, []string{})` is `false`). Instead use a
tiny helper that compares by length+elements (so nil and empty are treated as
equal, which is the functional truth for all downstream consumers):

```go
func slicesEq(a, b []string) bool {
	if len(a) != len(b) { return false }
	for i := range a { if a[i] != b[i] { return false } }
	return true
}
```

For the explicit "missing → nil" contract (item: "nil/missing -> nil"), ALSO
assert `skill.Keywords == nil` directly on the no-metadata fixture (verified in
§2 that `toStringSlice(nil)` returns a true `nil` slice, so this assertion holds).
This keeps tests strict where the contract demands nil and lenient elsewhere.
