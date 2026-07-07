# Verified Facts — P1.M4.T2.S1 (`skpp check`, PRD §9)

Measured live at `/home/dustin/projects/skpp` (go1.25) on 2026-07-06, BEFORE this
subtask is implemented. Every load-bearing decision is recorded here so the PRP
can quote it verbatim. This subtask runs IN PARALLEL with P1.M4.T1.S1 (`--search`);
the PRP assumes P1.M4.T1.S1's outputs exist when implementation starts.

## 0. Baseline (current on-disk state)

- `go build ./...` → exit 0. `go test ./...` → **105 tests**, all PASS
  (24 main + 31 discover + 10 resolve + 29 skillsdir + 11 ui).
- `go.mod`: `module github.com/dabstractor/skpp`, `go 1.25`,
  sole direct dep `gopkg.in/yaml.v3 v3.0.1`.
- Files on disk (READ-ONLY contracts this subtask consumes):
  - `internal/discover/discover.go` — `Frontmatter`(8 fields) + `ParseFrontmatter`
    (lenient; returns `Frontmatter{HasFM:false}` + nil err when no `---` block).
  - `internal/discover/skill.go` — `Skill`(Dir,RelTag,Name,Description,Keywords,
    Category,Aliases,HasFM,SourceFile) + `BuildSkill`.
  - `internal/discover/index.go` — `Index(skillsDir)([]Skill,error)`: WalkDir,
    returns ONLY dirs that contain SKILL.md, sorted by RelTag. Makes skillsDir
    absolute. SKIPS dirs without SKILL.md (so `check` must do its OWN walk for
    those — see §3). Ignores per-SKILL.md parse errors (builds HasFM=false Skill).
  - `internal/skillsdir/skillsdir.go` — `Find()(dir,src,err)`.
  - `internal/ui/ui.go` — `PrintList(w,skills,useColor)` (NOT reused by check;
    check has its own §9 line format).
  - `internal/resolve/resolve.go` — pattern reference (pure function over
    `[]discover.Skill`, own data model `Result`/`MatchKind`, own typed errors).
  - `main.go` — `config`, `parseArgs`, `run(args,stdout,stderr)int`, `isTerminal`,
    `version`. Wired: --version/--path/--list. **After P1.M4.T1.S1 lands** it ALSO
    has `--search`/`-s` (config gains `search string`+`searchSet bool`; parseArgs
    is an INDEX LOOP; a search block in run(); `"strings"` imported).

## 1. THE critical gotcha — the name regex does NOT forbid consecutive hyphens

The item description and external_deps.md §1 BOTH claim the regex
`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` forbids consecutive hyphens "via the
alternation". **This is FALSE.** Verified empirically with Go's `regexp`:

```
"a--b"           regex=true   contains--=true    ← MATCHES the regex!
"double--hyphen" regex=true   contains--=true    ← MATCHES the regex!
"triple---x"     regex=true   contains--=true    ← MATCHES the regex!
```

Why: the middle `[a-z0-9-]*` character class INCLUDES the hyphen, so it greedily
matches runs of hyphens. The alternation `([a-z0-9-]*[a-z0-9])?` does NOT force a
non-hyphen between hyphens (contrast with the OTHER regex in external_deps.md,
`^[a-z0-9]+(-[a-z0-9]+)*$`, which DOES forbid `--` because each `-` must be
followed by `[a-z0-9]+`).

**DECISION (matches external_deps.md's own recommendation):** use the regex
`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` (handles leading/trailing/single-char/uppercase
correctly) PLUS an EXPLICIT `!strings.Contains(name, "--")` check. The explicit
check is LOAD-BEARING, not redundant. external_deps.md §1 says exactly this:
"A Go-valid expression: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` plus a length check
1..64, plus a 'no `--` substring' check (the regex already forbids consecutive
hyphens via the alternation, but verify explicitly)."

Full name-validation rule (VERIFIED-correct):
1. `len(name) >= 1 && len(name) <= 64`
2. `nameRe.MatchString(name)` where `nameRe = ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
3. `!strings.Contains(name, "--")`

Verified regex behavior (all correct EXCEPT consecutive hyphens):
- valid: "ok","my-skill","a","ab","a-b","a-b-c","skill1","1skill","valid-name-here"
- invalid (regex=false): "-ab","ab-","-a","a-","Ab","aB","" (empty)
- 64×'a' → regex=true (length handled separately)
- "a--b" → regex=true ← the bug the explicit check must catch

## 2. Package placement — `internal/check` (NEW package), NOT main.go

DECISION: create `internal/check` (new package). Rationale:
- The item explicitly offers "Create `internal/check` (or a check.go in an
  existing package)" — `internal/check` is the FIRST option.
- check has a distinct DATA MODEL (`Level`, `Finding`) + a NEW filesystem walk
  (near-skill detection) + cross-skill aggregation (duplicate names) + a
  documented output format (§9). It is ~100 LOC, categorically larger than
  P1.M4.T1.S1's ~10-line stateless `--search` filter (which correctly stayed in
  main.go).
- It mirrors `internal/resolve`: a pure-function core (`Validate`) testable with
  `[]discover.Skill` literals and NO disk, plus a small I/O helper.
- The go_architecture.md "Data flow" diagram already lists `check` as a terminal
  peer of `ui.Print*`/`resolve` — a package fits. The package MAP omits it only
  because check hadn't been designed when the map was written.
- main.go stays thin: `Find()` → `Index()` → `check.Run()` → render → exit code.

CONTRAST with the search PRP's "stay in main" decision: search was a LEAF
stateless filter with no new data model and no I/O. check is none of those.

## 3. The near-skill walk (PRD §9 rule 1: "dir has no SKILL.md")

discover.Index returns ONLY dirs that HAVE SKILL.md, so detecting dirs that LACK
SKILL.md is check's job. The item: "walk skills/; for every immediate-or-nested
dir that has any file but no SKILL.md, emit ERROR 'no SKILL.md'."

NAIVE reading (flag every dir without SKILL.md that has a file) is WRONG: it would
false-positive on a skill's OWN support subdirs (`skills/foo/scripts/build.sh` —
`scripts/` has a file, no SKILL.md, but belongs to the `foo` skill).

CORRECT algorithm (VERIFIED by reasoning through the §10 convention):
1. Build a set of skill-dir absolute paths from discover.Index (each `Skill.Dir`).
2. WalkDir over skillsDir. For each directory D (skip the root itself):
   - If D is a skill dir → `filepath.SkipDir` (it has SKILL.md; AND we must NOT
     descend into it, because its `scripts/`/`references/`/`assets/` subdirs are
     the skill's own content, not near-skills).
   - Else if D contains ≥1 REGULAR file (non-directory entry, via `os.ReadDir`)
     → emit ERROR "directory has files but no SKILL.md" with D's relTag.
   - Else (D has only subdirs) → a pure grouping folder; not flagged. Keep walking.

`filepath.SkipDir` on a skill dir is the load-bearing trick: it prevents the walk
from EVER visiting a skill's support subdirs, so they cannot be false-positived.
Grouping folders (only subdirs) are correctly NOT flagged.

relTag for a near-skill dir = `filepath.ToSlash(filepath.Rel(skillsDir, D))`,
matching discover.Index's RelTag shape so ERROR lines align with OK lines.

Edge: the skillsDir ROOT itself is never flagged (relTag would be "."; a loose
file in `skills/` root is unusual and not in the test matrix). Guard: `if path ==
skillsDir { return nil }` (still descends).

## 4. Output format (PRD §9) — the documented contract

PRD §9 / §6.1 table (`skpp check` row): stdout = "Report: OK lines + any
WARN/ERROR lines"; exit = "0 if clean; 1 if any ERROR".

Line format (DEFINITIVE — check owns it):
- OK:    `OK    <relTag> (<name>)`
- WARN:  `WARN  <relTag>: <detail>`
- ERROR: `ERROR <relTag>: <detail>`

Implementation: `fmt.Sprintf("%-5s %s (%s)", level, relTag, name)` for OK and
`fmt.Sprintf("%-5s %s: %s", level, relTag, detail)` for WARN/ERROR. The `%-5s`
left-pads the label to width 5 ("OK"→"OK   ", "WARN"→"WARN ", "ERROR"→"ERROR") so
relTags align in a column. NOTE: PRD §9 shows `OK   <relTag>` (3 spaces); the
`%-5s `+space format renders OK with 4 spaces (aligned under ERROR). The PRD's
3-space form is illustrative (it is NOT byte-aligned with ERROR); check prioritizes
column alignment (standard linter convention). There is NO byte-exact acceptance
gate for the OK line (unlike `--path`), so tests assert SUBSTRINGS, not exact bytes.

Summary line (DEFINITIVE): `N skills, M errors, K warnings\n`
- N = len(discover.Index result) = discovered skills (does NOT count near-skills).
- M = count of LevelError findings (includes near-skill + dupe errors).
- K = count of LevelWarn findings.

Stream + exit code:
- check's full report (findings + summary) → STDOUT (check is a DIAGNOSTIC mode;
  its output is the point, like `--list`'s table). This is NOT the §6.4
  clean-stdout-on-failure path (that applies to PATH-output modes: `<tag>`,
  `--all`, `--path`). The §6.1 table confirms: stdout = the report.
- Infrastructure failure (Find()/Index() error) → one-line fix to STDERR, exit 1,
  stdout EMPTY (same as --list/--path; the store can't be read at all).
- Validation result → report to stdout; exit 0 if M==0 else 1. WARNs never affect
  the exit code.

## 5. Validation rules — precise mapping (PRD §9 ↔ code)

| §9 rule | Severity | Condition (on a discover.Skill `s`) |
|---|---|---|
| 1 no SKILL.md | ERROR | from the near-skill walk (NOT from `s` — Index skills always have SKILL.md) |
| 2 missing name/desc | ERROR | `!s.HasFM` → "no frontmatter block"; else `s.Name==""` → "missing name"; else `TrimSpace(s.Description)==""` → "missing/empty description" |
| 3 name shape | ERROR | `s.Name != "" && !validName(s.Name)` (see §1: regex+len+explicit `--`) |
| 4 duplicate name | ERROR | name appears in ≥2 skills (computed globally; empty names excluded) |
| 5 desc too long | WARN | `len(s.Description) > 1024` |

Rule 6 (WARN: skill dir empty besides SKILL.md) is OPTIONAL per PRD §9 and NOT in
the item's test matrix → DEFERRED (not implemented; documented as optional).

Emit ONE Finding PER VIOLATION (linter-style). A clean skill → exactly one OK
Finding. A skill may appear on multiple lines (e.g. bad name ERROR + long desc
WARN). Findings sorted by RelTag (stable) so a skill's problems stay grouped.

## 6. Wiring as `skpp check` SUBCOMMAND (not a flag)

PRD §6.1 row: `skpp check` (positional subcommand, NOT `--check`). So:
- `config` gains `check bool`.
- `parseArgs` (already an INDEX LOOP after P1.M4.T1.S1) adds
  `case "check": c.check = true` (bare token, no value capture).
- `run()` adds a check block AFTER the search block, BEFORE the default `return 1`.
- main.go imports `"github.com/dabstractor/skpp/internal/check"`.

Precedence: --version wins (existing). check is a mode peer of --list/--search;
multi-mode mixing (`check --list`) is tolerated pre-M5 (M5 enforces §6.3 exit 2).

## 7. API surface of `internal/check` (LOCKED)

```go
package check

type Level int
const ( LevelOK Level = iota; LevelWarn; LevelError )

type Finding struct {
    Level  Level
    RelTag string
    Name   string  // frontmatter name for the OK line; "" otherwise
    Detail string  // message for WARN/ERROR; "" for OK
}
func (f Finding) String() string          // one report line (§4 format)
func Run(skillsDir string, skills []discover.Skill) []Finding  // full report, sorted
func Validate(skills []discover.Skill) []Finding               // pure frontmatter rules (no I/O)
func Count(findings []Finding) (errors, warnings int)
func SummaryLine(nSkills int, findings []Finding) string       // "N skills, M errors, K warnings"
```

main calls: `findings := check.Run(dir, skills)`; loop `fmt.Fprintln(stdout, f)`;
`fmt.Fprintln(stdout, check.SummaryLine(len(skills), findings))`;
`errs,_ := check.Count(findings); return map[bool]int{true:0,false:1}[errs==0]`.

## 8. Test plan (mirrors repo style: plain t.Errorf, no testify, no t.Parallel)

`internal/check/check_test.go` (package check) — ~16 tests:
- pure `Validate`: clean→OK; no-FM→ERROR; missing-name→ERROR; missing-desc→ERROR;
  bad-name (table incl. "a--b" regression)→ERROR; 65-char-name→ERROR; 1025-desc→
  WARN; 1024-desc→OK (boundary); duplicate-names→ERROR×2; distinct-names→no-dupe.
- `validName` table (ok/leading/trailing/consecutive/uppercase/empty/len).
- `Finding.String`/`SummaryLine`/`Count` format tests.
- `nearSkillFindings` (disk): files-no-SKILL.md→ERROR; skill's scripts/ NOT
  flagged (SkipDir); grouping dir (only subdirs) NOT flagged.
- `Run` integration: combines Validate+nearSkill, sorted by relTag.

`main_test.go` (package main, white-box) — ~7 tests appended:
- clean store → OK line + summary on stdout, exit 0.
- bad name → ERROR, exit 1.
- near-skill dir (files, no SKILL.md) → ERROR, exit 1.
- two skills same name → ERROR duplicate, exit 1.
- >1024 description → WARN, exit 0 (warns don't fail).
- `check` recognized as positional subcommand.
- skills dir unresolvable → stderr one-liner, exit 1, stdout EMPTY.

Expected total after this subtask: 118 (post-search baseline) + ~23 = ~141 tests.

## 9. Dependencies / go.mod

`internal/check` imports: stdlib `fmt`,`io/fs`,`os`,`path/filepath`,`regexp`,
`sort`,`strings` + internal `discover`. NO new external dependency. yaml.v3 stays
the sole direct dep. `go mod tidy` is a no-op → `git diff --quiet go.mod go.sum`
exits 0. (regexp is stdlib, not external.)
