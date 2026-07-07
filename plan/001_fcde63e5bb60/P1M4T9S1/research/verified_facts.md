# Verified Facts — P1.M4.T9.S1: `--search` substring filter + ui reuse

> Research notes supporting `PRP.md`. All facts verified by reading the on-disk
> code at `~/projects/skpp` (commit at research time; M1/M2/M3 landed & green).
> No external/library research was warranted: the feature is `strings.Contains`
> over lowercased fields (Go stdlib) + reuse of the existing `ui.PrintList`.

---

## 1. What the PRD requires (§6.1, authoritative)

`skpp --search <q>` / `-s <q>`:
- **Matching:** case-insensitive **substring** search over **exactly four** fields:
  `tag`, frontmatter `name`, `description`, and `metadata.keywords`.
- **Output:** "Same table format as `--list`, filtered."
- **Exit:** `0` on matches; `1` if no matches.

Field scope is **narrow and explicit**. `Category` and `Aliases` exist on the
`discover.Skill` struct but are **NOT** in the §6.1 list — searching them would
violate the spec. (Common pitfall: since both fields sit right next to `Keywords`
on the struct, an implementer eyeballing it may sweep them in. Do not.)

## 2. Direct precedent: `internal/resolve` (the pattern to mirror)

`internal/resolve/resolve.go` is a **PURE function over `[]discover.Skill`** that
does matching (the §7.2 precedence), in its **own package** with its **own
`_test.go`**, called by `main`. Search is structurally identical: a pure matching
function over `[]discover.Skill`. Therefore the conventional home for the filter
is a new **`internal/search`** package (`search.go` + `search_test.go`), keeping
`main` a thin dispatcher. Verified: `main.run` already delegates every piece of
logic to an `internal/*` package (skillsdir/discover/resolve/ui); there is no
business logic inlined in `main`.

## 3. The existing formatter to reuse: `ui.PrintList`

Signature (internal/ui/ui.go):
```go
func PrintList(w io.Writer, skills []discover.Skill, useColor bool)
```
- **Pure formatter.** Takes a pre-sorted `[]discover.Skill`, renders the
  `TAG`/`NAME`/`DESCRIPTION` table. **Does NOT re-sort** (doc comment: "skills
  MUST already be ordered the way rows should appear").
- **Empty slice → prints nothing** (returns immediately). The CALLER (main) is
  responsible for the exit-1 "no matches" decision BEFORE calling PrintList.
- Its own doc comment already states the design intent for this task:
  *"`--search` (P1.M4.T9) reuses `PrintList` with a filtered slice."*
- Color via the `useColor` param; caller owns the TTY/`--no-color` decision.
- `--list` currently calls it as `ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)`.
  `--search` calls it identically with the **filtered** slice.

## 4. `discover.Skill` searchable fields (internal/discover/skill.go)

```go
type Skill struct {
    Dir         string
    RelTag      string   // ← the "tag" (canonical, '/'-normalized) — ALWAYS present
    Name        string   // ← "" if no frontmatter / no name field
    Description string   // ← "" if absent; may carry a folded-scalar trailing newline
    Keywords    []string // ← metadata.keywords, nil if absent; use len(), not nil-check
    Category    string   // NOT searchable (out of §6.1 scope)
    Aliases     []string // NOT searchable (out of §6.1 scope)
    HasFM       bool     // false => Name/Description empty, Keywords nil
    SourceFile  string
}
```
- `discover.Index(dir)` returns `[]Skill` **sorted by RelTag** (ascending). The
  search filter iterates in order → filtered slice stays sorted → PrintList shows
  it sorted. No re-sort needed anywhere.
- A no-frontmatter skill (`HasFM==false`) has empty Name/Description/nil Keywords
  but a **non-empty RelTag**, so it is still discoverable by searching its tag
  (mirrors resolve letting it resolve by directory/basename, PRD §7.1).

## 5. `main.go` insertion points (verified line-by-line)

- **config struct** (≈L74): currently has `version/path/list/all/file/relative/
  noColor/tags`. Its own comment lists `search string` under "Future (M4/M5), do
  NOT add yet" — this subtask removes that line and adds the search fields.
- **parseArgs** (≈L91): currently `for _, a := range args { switch a {...} }` —
  a **range loop**. A value-taking flag (`--search <q>`) needs to **consume the
  next token**, so the loop must become **index-based** (`for i := 0; ...; i++`)
  with an `i++` skip when the value is grabbed. This is the ONLY structural change
  to the parser; every existing `case` is unchanged.
- **run()** dispatch order (verified): `version → path → list → all → tags → default`.
  The search branch slots in as a **display mode alongside `list`** — insert
  immediately AFTER the `if c.list {...}` block and BEFORE the `--all` block.
  (Relative order among display modes is a no-op in practice; §6.3 mutual-
  exclusivity → exit 2 is explicitly P1.M5.T11.)
- **Imports**: add `"github.com/dabstractor/skpp/internal/search"` to the existing
  `internal/*` import group (alphabetical: after resolve, before skillsdir).

## 6. Key semantics decisions (locked, with rationale)

| # | Decision | Choice | Why |
|---|---|---|---|
| 1 | Case-insensitive substring | `strings.Contains(strings.ToLower(hay), q)`; lowercase the **query once** | Standard Go idiom; no regex/unicode-folding dep needed (tags/names are ASCII a-z0-9-) |
| 2 | Keywords matching | query is substring of **any individual** keyword (iterate, do NOT `strings.Join`) | Joining risks a false match spanning a boundary between two keywords |
| 3 | Order preservation | filter iterates input order; no re-sort | discover.Index pre-sorts; ui.PrintList does not re-sort |
| 4 | Empty query `""` | matches **all** skills (exit 0, like `--list`) | `strings.Contains(x,"")` is always true; PRD carves out no special case |
| 5 | No matches | exit **1**, message to stderr, **empty stdout** | PRD §6.1 "1 if no matches"; mirror `--list`'s "no skills found" convention |
| 6 | `--no-color` / TTY | shared with `--list` (same `isTerminal(stdout) && !c.noColor`) | PRD §6.2; search prints a TABLE |
| 7 | `--file` / `--relative` | do **NOT** apply to search | Search prints a table, not paths (PRD §6.2: modifiers combine with tag/`--all` only) |
| 8 | `--search` as last token (no value) | searchMode stays **false** → default exit-1 | Proper "flag requires an argument" exit-2 is P1.M5.T11 |
| 9 | `--search` value starting with `-` | grabbed verbatim as the query | Simple-parser convention; refining to an error is M5 |
| 10 | New package `internal/search` | YES (mirror `resolve`) | Pure, independently unit-testable, keeps main thin |

## 7. Stderr message for "no matches"

Mirror `--list`'s `fmt.Fprintln(stderr, "no skills found in "+dir)`. For search:
`fmt.Fprintln(stderr, "no skills matched "+c.searchQ)`. Concise, on one line,
names the query. Goes to stderr so stdout stays clean.

## 8. Scope boundaries (do NOT do — owned by other subtasks)

- `check` subcommand (§9) → P1.M4.T10.S1.
- `--help` text + exit-2 for unknown flags + §6.3 mutual-exclusivity → P1.M5.T11.
- `skills/example/SKILL.md` → P1.M6.T12.S1.
- Do NOT modify `internal/discover/*`, `internal/skillsdir/*`, `internal/resolve/*`,
  `internal/ui/*`, `go.mod`, `go.sum`, `PRD.md`. Read-only consumption.
- No new third-party dependency (stdlib `strings` only). `go mod tidy` is a **no-op**.

## 9. Current green baseline (to compute the test delta)

`go test ./...` is green. Per-package test counts at research time:
- `.` (main): **53**
- `internal/discover`: 39
- `internal/resolve`: 13
- `internal/skillsdir`: 29
- `internal/ui`: 11
- **Total: 145**

`gofmt -l main.go main_test.go internal/` is silent; `go vet ./...` is clean.
`go.mod`: `module github.com/dabstractor/skpp`, `go 1.25`, single dep
`gopkg.in/yaml.v3 v3.0.1` (unchanged by this task).

## 10. No external/library research needed

The feature is Go-stdlib string filtering + reuse of an existing in-repo
formatter. There is no library API, version constraint, or integration to
document. Citing `pkg.go.dev/strings` would be noise; the two functions used
(`strings.Contains`, `strings.ToLower`) are stable since Go 1.0.
