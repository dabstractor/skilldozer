# Go Architecture & Package Design — `skpp`

## Package map (PRD §5)

```
main.go                      # entrypoint: flag parsing, dispatch, exit codes
internal/skillsdir/skillsdir.go   # locate the skills/ dir (§8)
internal/discover/discover.go     # walk + parse frontmatter, build index (§7.1, §7.3)
internal/resolve/resolve.go       # tag → skill resolution precedence (§7.2)
internal/ui/ui.go                 # --list / --search table + ANSI (§6.1)
```

`internal/` keeps these packages unimportable by outsiders (correct for a CLI).

## Data flow (single invocation)

```
main.parseArgs
  │
  ├─ skillsdir.Find()  ──▶ (absSkillsDir string, rule Source, err error)
  │      §8 priority: SKPP_SKILLS_DIR → sibling-of-binary → walk-up-from-cwd → error
  │
  ├─ discover.Index(absSkillsDir) ──▶ []Skill, error
  │      WalkDir; each dir containing SKILL.md → parse frontmatter → Skill
  │
  ├─ resolve.Resolve(tag, index) ──▶ (Skill, error)   [per tag]
  │      §7.2 precedence: exact relTag → basename → frontmatter name → alias
  │
  └─ ui.Print* / print paths / check
```

## Core types (contract between packages)

### `internal/discover`

```go
package discover

type Skill struct {
    Dir         string   // absolute path of the skill directory
    RelTag      string   // dir path relative to skills dir, separators normalized to '/'
    Name        string   // frontmatter name ("" if missing)
    Description string   // frontmatter description ("" if missing)
    Keywords    []string // metadata.keywords, else nil
    Category    string   // metadata.category, else ""
    Aliases     []string // metadata.aliases, else nil
    HasFM       bool     // false if no --- frontmatter block found
    SourceFile  string   // absolute path to SKILL.md (Dir + "/SKILL.md")
}

// Index walks absSkillsDir and returns every skill (dir containing SKILL.md),
// sorted by RelTag for deterministic output.
func Index(absSkillsDir string) ([]Skill, error)

// ParseFrontmatter reads SKILL.md, extracts the YAML block between the first
// two lines that are exactly "---", and unmarshals into Frontmatter.
// Missing frontmatter => returns Frontmatter{} with HasFM=false, no error.
func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)
```

### `internal/skillsdir`

```go
package skillsdir

type Source int // which §8 rule won (for `skpp --path` reporting / debugging)
const (
    SourceEnv Source = iota   // SKPP_SKILLS_DIR
    SourceSibling             // sibling of running binary
    SourceWalkUp              // ancestor of cwd
)

// Find locates the skills dir per §8 priority. Returns absolute path + which
// rule won. Returns error if none found (caller prints fix hint + exit 1).
func Find() (dir string, src Source, err error)
```

### `internal/resolve`

```go
package resolve

import "github.com/dabstractor/skpp/internal/discover"

// Result is the outcome of resolving one tag.
type Result struct {
    Skill discover.Skill
    Match MatchKind // Canonical | Basename | Name | Alias
}

// Resolve applies the §7.2 precedence to a single tag against the index.
// Returns ErrUnknown (no match), ErrAmbiguous (with candidate tags), or nil.
func Resolve(tag string, skills []discover.Skill) (Result, error)

// Errors carry candidate RelTags for the ambiguous case (§6.4 lists candidates).
type AmbiguousError struct{ Tag string; Candidates []string }
type UnknownError struct{ Tag string }
```

## Key implementation notes (gotchas confirmed during research)

### Skills dir location (§8) — hardest part
1. `SKPP_SKILLS_DIR`: if set and `os.Stat` says existing dir → use it. Do NOT
   EvalSymlinks here (env points exactly where the user wants).
2. Sibling-of-binary: `exe, _ := os.Executable()`; `real, _ := filepath.EvalSymlinks(exe)`;
   `repoDir := filepath.Dir(real)`; if `repoDir/skills` exists → use it. This makes
   `~/.local/bin/skpp → ~/projects/skpp/skpp` resolve back to the repo. TEST:
   `ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp; /tmp/skpp-bin/skpp example` must still
   resolve to `$PWD/skills/example`.
3. Walk-up from cwd: for `go run` (binary in temp dir). Ascend from cwd; first
   ancestor with a `skills/` subdir containing ≥1 `SKILL.md` wins.
4. None → error + one-line fix (`set $SKPP_SKILLS_DIR`, `cd` into repo, or reinstall).

### relTag normalization
`filepath.WalkDir` yields OS-native separators. Build relTag via
`filepath.Rel(skillsDir, skillDir)` then `filepath.ToSlash(...)` so tags are
always `writing/reddit` on every platform. Canonical tag comparisons are on the
slash form.

### Frontmatter block extraction
- Read file. If it does NOT start with a line that is exactly `---` (after the
  initial newline trim) → no frontmatter (HasFM=false).
- Find the NEXT line that is exactly `---`. Slice between them = YAML block.
- If there is no closing `---`, treat as no frontmatter (lenient).
- Unmarshal YAML block with yaml.v3 into Frontmatter. Unknown keys ignored by
  default (do NOT set KnownFields(true)).

### metadata extraction (skpp conventions, not spec fields)
```go
// from Frontmatter.Metadata (map[string]any):
func toStringSlice(v any) []string  // handles []any → []string, single string, nil
keywords = toStringSlice(fm.Metadata["keywords"])
category, _ = fm.Metadata["category"].(string)
aliases  = toStringSlice(fm.Metadata["aliases"])
```

### Output discipline (§6.4) — critical
- Resolve ALL tags first, collect errors. If ANY tag fails → print one error line
  per problem to STDERR, print NOTHING to stdout, exit 1. Never partially print.
- Ambiguous → stderr lists candidate full tags.
- Buffer stdout writes; only flush when the whole invocation is known-good.

### Exit codes
- 0: success
- 1: any unresolved/ambiguous tag; no skills found (--list); skills dir unresolvable
- 2: unknown flag; mutually-exclusive modes mixed (tag + --list/--search/--all)

### ANSI color
Gate on: stdout is a TTY (`term.IsTerminal` or check `os.Stdout.Stat()` mode) AND
`--no-color` not set. `--list`/`--search` tables use color; path output never does
(it's consumed by `$(...)`). Provide a `--no-color` flag.

## Testing strategy (TDD implied per subtask)

- `internal/skillsdir`: table-driven tests with temp dirs + symlinked binary
  (use `os.Symlink` in t.TempDir()).
- `internal/discover`: temp skills tree with known SKILL.md files; assert index
  contents incl. missing-frontmatter and nested skills.
- `internal/resolve`: table over the §7.2 examples + ambiguous + unknown cases.
- main/exit codes: build a tiny test binary or test the dispatch functions with
  captured stdout/stderr.
- Acceptance: run PRD §13 commands verbatim as the final smoke test.
