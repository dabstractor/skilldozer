# PRP — P1.M4.T2.S1: `skpp check` validation (PRD §9) + output format + exit codes

> **Subtask:** P1.M4.T2.S1 (plan id) = P1.M4.T10.S1 (PRD build-order id) — the
> only subtask of T10 (`skpp check`, PRD §9). Milestone **M4** (Search &
> validation). Build-order step 4. **SP 2.**
>
> **Scope:** CREATE `internal/check/check.go` (new package) + `internal/check/
> check_test.go`; MODIFY `main.go` (config +1 field, parseArgs +1 case, run() +1
> block, +1 import) and APPEND tests to `main_test.go`. **Two new files + two
> modified files. Nothing else.** No go.mod/go.sum change (only stdlib `regexp`).
>
> **PARALLEL CONTEXT:** This subtask runs alongside P1.M4.T1.S1 (`--search`).
> When implementation starts, P1.M4.T1.S1 will have LANDED: `main.go` has the
> `--search`/`-s` wiring (config has `search string`+`searchSet bool`; parseArgs
> is an INDEX LOOP; run() has a search block; `"strings"` is imported). This PRP
> ADDS to that state — it does not duplicate or conflict with it. If P1.M4.T1.S1
> has NOT landed yet, its edits (the `--search` block, the index loop, the
> `"strings"` import) must be applied first; the edits below are additive on top.
>
> **VERIFICATION STATUS:** Baseline measured live at `/home/dustin/projects/skpp`
> (go1.25): `go build ./...` exit 0; `go test ./...` = **105 tests**. The name
> regex gotcha (§1 below) was verified empirically. Every load-bearing fact is in
> `research/verified_facts.md`.

---

## Goal

**Feature Goal**: Implement PRD §9 `skpp check`: build the index via
`discover.Index()`, validate every discovered skill against the §9 frontmatter
rules (missing fields, name shape, duplicate names, over-long description), AND
walk the store to find "near-skill" directories (files present but no SKILL.md).
Print a per-finding report (`OK`/`WARN`/`ERROR` lines) + a summary line to stdout;
exit 0 if no ERROR, else 1.

**Deliverable**: CREATE two files + MODIFY two files:
1. `internal/check/check.go` — NEW package: `Level`/`Finding` types, `Run`,
   `Validate` (pure), `Count`, `SummaryLine`, `validName`, the near-skill walk.
2. `internal/check/check_test.go` — NEW: 17 tests (pure Validate, validName
   table, Finding/Summary/Count format, near-skill walk on temp dirs, Run).
3. `main.go` — add `check bool` to config; `case "check":` in parseArgs; a check
   block in run() (after search, before default); import `internal/check`.
4. `main_test.go` — APPEND 7 integration tests wiring `skpp check` through run().

**Success Definition**: `gofmt -l *.go internal/check/*.go` silent; `go vet ./...`
clean; `go build ./...` + `go test ./...` pass (24 new tests: 17 in internal/check + 7
in main → 105 baseline + 24 = 129, or 118 post-search + 24 = 142); `go.mod`/`go.sum` byte-identical
(`go mod tidy` no-op). `skpp check` on a clean store prints `OK <tag> (<name>)`
per skill + `N skills, 0 errors, 0 warnings` and exits 0; on any ERROR it prints
the ERROR lines and exits 1; a dir with files but no SKILL.md is flagged ERROR;
two skills sharing a frontmatter name are both flagged ERROR duplicate.

---

## Why

- PRD §18 build-order step 4 (the validation half of M4). Until `check` lands, the
  §9 validation contract — the documented "is my skills store healthy?" command —
  does not exist. README §6 (Mode B) references `skpp check` as the linter.
- It is the ONLY place skpp enforces the Agent Skills naming rules and the §10
  frontmatter conventions. The rule set IS the contract (the item calls it
  "the documented contract"); locking it now is the whole point.
- It exercises a NEW data flow discover.Index does NOT cover: finding dirs that
  LACK SKILL.md (Index returns only dirs that HAVE one). That near-skill detection
  is genuinely new logic, which is why it earns its own package (`internal/check`)
  rather than living in main.go like the stateless `--search` filter.
- PRD §13 acceptance gate runs `./skpp check` and expects exit 0 + the shipped
  example skill reported as OK.

---

## What

A new `skpp check` subcommand (PRD §6.1 row: positional `check`, NOT `--check`):

1. **`parseArgs`** recognizes the bare token `"check"` and sets `config.check`.
2. **`run()`** gains a `check` dispatch block: `skillsdir.Find()` →
   `discover.Index(dir)` → `check.Run(dir, skills)` → print each `Finding` →
   print `check.SummaryLine` → exit 0 if no ERROR else 1. Infrastructure failure
   (Find/Index error) → stderr one-liner + exit 1 + empty stdout.
3. **`internal/check.Run`** validates the index (frontmatter rules) AND walks the
   store for near-skill dirs, returns `[]Finding` sorted by RelTag.
4. **`internal/check.Validate`** is a PURE function over `[]discover.Skill` (no
   I/O) applying §9 rules 2–5; fully unit-testable with struct literals.

### Success Criteria

- [ ] `skpp check` is recognized as a subcommand (positional token, any position).
- [ ] A clean skill (valid name + non-empty description ≤1024 chars) → one
      `OK <relTag> (<name>)` line; exit 0.
- [ ] A skill whose SKILL.md has no `---` frontmatter block → ERROR; exit 1.
- [ ] A skill missing `name` → ERROR; missing/empty `description` → ERROR; exit 1.
- [ ] A skill with a name violating the rules (bad charset, leading/trailing
      hyphen, **consecutive hyphens**, length >64) → ERROR; exit 1.
- [ ] A skill with `description` > 1024 chars → WARN; exit 0 (warns don't fail).
- [ ] Two skills sharing a frontmatter `name` → both flagged ERROR "duplicate".
- [ ] A directory with files but no SKILL.md (and not under a real skill) →
      ERROR "directory has files but no SKILL.md"; exit 1.
- [ ] A skill's OWN `scripts/`/`references/`/`assets/` subdirs are NOT flagged
      (they belong to the skill, not near-skills).
- [ ] Summary line printed last: `N skills, M errors, K warnings`.
- [ ] Skills-dir unresolvable / Index error → exit 1, stderr one-liner, stdout
      EMPTY (same contract as --path/--list).
- [ ] `gofmt`/`go vet`/`go build`/`go test` clean; `go.mod`/`go.sum` unchanged.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for every new file (`internal/check/check.go`) and every
edit (`main.go` config/parseArgs/run/import) is given verbatim in the
Implementation Blueprint, plus all ~16 check_test.go tests and ~7 main_test.go
tests. The consumed contracts are on disk and green: `discover.Index`,
`discover.Skill`, `skillsdir.Find`, and `main.run`/`parseArgs`/`config`. The
single critical gotcha (the name regex does NOT forbid consecutive hyphens —
verified empirically) is called out in §1 below with the load-bearing explicit
`--` check. An implementer who knows Go but nothing about this repo can finish in
one pass by applying the code verbatim._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical baseline (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M4T2S1/research/verified_facts.md
  why: "Measured BEFORE writing the PRP. §1 = THE gotcha: the regex
        ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ does NOT forbid consecutive hyphens
        ('a--b' matches it, verified); the explicit !Contains(name,'--') check is
        LOAD-BEARING. §2 = why internal/check is a package (not main.go). §3 = the
        near-skill walk algorithm (SkipDir on skill dirs is the trick). §4 = exact
        output format + stream/exit-code semantics. §5 = rule→code table. §6 =
        subcommand wiring. §7 = locked API surface. §8 = test plan."
  critical: "validName MUST be regex + len(1..64) + explicit '--' check. Do NOT
             trust the regex alone. Do NOT flag a skill's scripts/references/
             assets subdirs (SkipDir on skill dirs). check prints to STDOUT (it is
             a diagnostic mode, not a §6.4 path-output mode)."

# CONTRACT — the Skill fields check validates (on disk; READ-ONLY)
- file: internal/discover/skill.go
  why: "type Skill struct{Dir,RelTag,Name,Description,Keywords,Category,Aliases,
        HasFM,SourceFile}. check reads RelTag (line label), Name (rule 2/3/4 +
        OK display), Description (rule 2/5), HasFM (rule 2). HasFM=false means the
        SKILL.md had no '---' block. Description is copied VERBATIM (may carry a
        folded-scalar trailing newline) — the 1024 check uses raw len()."
  pattern: "Validate ranges []discover.Skill; reads s.RelTag/s.Name/s.Description/
            s.HasFM."
  gotcha: "Description trailing newline counts toward len() — a 1024-char desc with
           a folded newline stores 1025 and WARNs. Acceptable (item tests use a
           clearly-over case); documented, not special-cased."

# CONTRACT — Index() (the data source; on disk; READ-ONLY)
- file: internal/discover/index.go
  why: "func Index(skillsDir)([]Skill,error) returns ONLY dirs containing SKILL.md,
        sorted by RelTag, with skillsDir made absolute. It SKIPS dirs without
        SKILL.md and ignores per-file parse errors (builds HasFM=false Skill).
        Therefore the 'no SKILL.md' rule is NOT about Index's skills — it is about
        the SEPARATE near-skill walk (check's own WalkDir). main passes the SAME
        (dir, skills) to check.Run that it built for --list."
  pattern: "main's check block calls discover.Index(dir) exactly like --list."

# CONTRACT — ParseFrontmatter (what HasFM means; on disk; READ-ONLY)
- file: internal/discover/discover.go
  why: "ParseFrontmatter returns Frontmatter{HasFM:false},nil when SKILL.md has no
        '---' block (lenient). So a Skill with HasFM=false is one whose SKILL.md
        lacked a frontmatter block — check reports 'no frontmatter block'. check
        does NOT call ParseFrontmatter itself (Index already did); it reads the
        pre-built Skill."
  gotcha: "Do NOT re-parse in check. Read s.HasFM / s.Name / s.Description."

# CONTRACT — skillsdir.Find() (on disk; READ-ONLY)
- file: internal/skillsdir/skillsdir.go
  why: "func Find()(dir string, src Source, err error). main's check block mirrors
        --list/--search: dir,_,err := skillsdir.Find(); err -> stderr+exit1."

# PATTERN REFERENCE — how a pure-function internal package is structured & tested
- file: internal/resolve/resolve.go
  why: "resolve is the template for internal/check: a PURE core over
        []discover.Skill (Resolve/Validate), an exported data model (Result/
        MatchKind vs Finding/Level), typed helpers, and a _test.go that builds
        []discover.Skill literals and tables them. Mirror its doc-comment density
        and its 'pure: no I/O' framing for Validate."
  pattern: "package-level fixture slices of discover.Skill; table tests; plain
            t.Errorf; no testify; no t.Parallel."

# CONTRACT — the dispatcher to extend (on disk; the file THIS subtask modifies)
- file: main.go
  why: "main.go owns parseArgs, run(), config, isTerminal, version. After
        P1.M4.T1.S1 lands it ALSO owns --search (config.search/searchSet; index-
        loop parseArgs; a search block; 'strings' import). This subtask ADDS:
        config.check; case 'check' in parseArgs; a check block in run() AFTER the
        search block; the 'internal/check' import."
  pattern: "Mirror the existing --list block's shape: Find() -> Index() -> (err?)
            -> transform -> print -> exit code."
  gotcha: "parseArgs is ALREADY an index loop (post P1.M4.T1.S1). Adding
           case \"check\": c.check = true is a one-liner (bare token, no value).
           If P1.M4.T1.S1 has NOT landed, the index-loop conversion + search block
           must be applied first (see P1.M4.T1S1/PRP.md)."

# CONTRACT — the §9 rules + §6.1 `check` row + §6.4 stream discipline
- file: PRD.md
  why: "§9 lists the 5 rules + the output format + the summary line. §6.1 table:
        'skpp check | Validate every skill on disk (see §9). | stdout: Report: OK
        lines + any WARN/ERROR lines. | exit 0 if clean; 1 if any ERROR'. §6.4:
        clean-stdout-on-failure applies to PATH-output modes (<tag>/--all/--path),
        NOT to diagnostic modes — check's report goes to stdout (like --list)."
  section: "9. Validation", "6.1 Commands / flags", "6.4 Error semantics"
  critical: "READ-ONLY. Do NOT edit PRD.md. check -> stdout report; exit 1 on any
             ERROR; infrastructure failure -> stderr + empty stdout + exit 1."

# REFERENCE — verified name rules + the explicit '--' recommendation
- file: plan/001_fcde63e5bb60/architecture/external_deps.md
  why: "§1 gives the name regex ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$, length 1-64, AND
        explicitly recommends 'a no -- substring check (the regex already forbids
        consecutive hyphens via the alternation, but verify explicitly).' The
        parenthetical is WRONG (the alternation does NOT forbid them — verified),
        but the RECOMMENDATION (add the explicit check) is correct and load-bearing."
  section: "1. Agent Skills specification -> Name validation regex (for skpp check, §9)"

# REFERENCE — package map + testing strategy
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "'Package map (PRD §5)' lists main+skillsdir+discover+resolve+ui. The 'Data
        flow' diagram lists 'ui.Print* / print paths / check' as a terminal peer —
        a check package fits. 'Testing strategy' says internal/discover uses temp
        trees + known SKILL.md; check mirrors that for the near-skill walk. The map
        omits internal/check only because it predates this subtask; the item
        explicitly authorizes the new package."
  section: "Package map (PRD §5)", "Data flow", "Testing strategy"
```

### Current Codebase tree (M1 + M2 + M3.T7 + [P1.M4.T1.S1 search] COMPLETE)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*' | sort
internal/discover/discover.go        # Frontmatter(8) + ParseFrontmatter + utf8BOM
internal/discover/discover_test.go   # discover tests
internal/discover/skill.go           # Skill(9) + BuildSkill + toStringSlice
internal/discover/skill_test.go      # skill tests
internal/discover/index.go           # Index(skillsDir)([]Skill,error) + sort by RelTag
internal/discover/index_test.go      # index tests
internal/resolve/resolve.go          # Resolve + MatchKind + typed errors
internal/resolve/resolve_test.go     # resolve tests (10)
internal/skillsdir/skillsdir.go      # Source + Find + per-rule helpers
internal/skillsdir/skillsdir_test.go # skillsdir tests
internal/ui/ui.go                    # PrintList table + ANSI + padRight/wrapWords
internal/ui/ui_test.go               # ui tests
main.go                              # version/path/list [+search after P1.M4.T1.S1]  <-- MODIFY
main_test.go                         # main tests [+13 search after P1.M4.T1.S1]        <-- APPEND

# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT, ONLY)
# baseline (measured): go build ./... OK; go test ./... = 105 tests.
# After P1.M4.T1.S1 lands: 118 tests, main.go has the --search wiring + "strings" import.
# NO skills/ dir yet (P1.M6.T12 ships skills/example/SKILL.md).
# NO internal/check yet (this subtask creates it).
```

### Desired Codebase tree with files to be added/modified

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/skillsdir/*,
│        internal/discover/* [Frontmatter+Skill+Index], internal/resolve/*,
│        internal/ui/* — ALL UNCHANGED by this subtask)
├── internal/check/
│   ├── check.go        # NEW — Level/Finding + Run + Validate + Count + SummaryLine
│   │                   #       + validName + near-skill walk (hasRegularFile/relTag)
│   └── check_test.go   # NEW — 17 tests (pure Validate, validName table, format,
│                       #       near-skill walk on temp dirs, Run integration)
├── main.go             # MODIFY — config +check; parseArgs +case "check"; run() +check
│                       #          block; +import "internal/check"
└── main_test.go        # MODIFY — APPEND 7 integration tests (run() + check subcommand)
```

| File | Change | New import |
|---|---|---|
| `internal/check/check.go` | NEW package | stdlib `fmt`,`io/fs`,`os`,`path/filepath`,`regexp`,`sort`,`strings` + internal `discover` |
| `internal/check/check_test.go` | NEW tests | stdlib `os`,`path/filepath`,`strings`,`testing` + internal `discover` |
| `main.go` | config +1 field; parseArgs +1 case; run() +1 block | `+ "github.com/dabstractor/skpp/internal/check"` |
| `main_test.go` | APPEND ~7 tests (reuse `writeSkillTree`/`unsetSkillsEnv`) | none new (already imports bytes/strings/testing/discover) |

**Two new files + two modified files. NO go.mod/go.sum change** (`regexp` is stdlib).

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 (CRITICAL) — the name regex does NOT forbid consecutive hyphens.
// The item + external_deps.md §1 both CLAIM ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ forbids
// them "via the alternation". That is FALSE (verified empirically: "a--b" matches
// regex=true, "double--hyphen" matches regex=true). The middle [a-z0-9-]* class
// includes the hyphen and greedily matches "--". external_deps.md itself hedges:
// "plus a 'no -- substring' check ... but verify explicitly." So validName MUST be:
//   regex.MatchString && len in [1,64] && !strings.Contains(name, "--")
// The !Contains(name,"--") is LOAD-BEARING. A test with name "a--b" MUST fail
// validation — it is the regression guard for this gotcha.
//   RIGHT: if !nameRe.MatchString(name) || strings.Contains(name,"--") { invalid }
//   WRONG: if !nameRe.MatchString(name) { invalid }  // "a--b" slips through.

// GOTCHA #2 — discover.Index returns ONLY dirs with SKILL.md. The §9 rule "skill
// dir has no SKILL.md" therefore CANNOT be detected from Index's output (every
// Skill it returns has SKILL.md by construction). That rule is about NEAR-SKILL
// dirs (dirs with files but no SKILL.md), found by check's OWN WalkDir. Do NOT try
// to detect "missing SKILL.md" by inspecting Skill structs — a Skill always has one.

// GOTCHA #3 — do NOT flag a skill's OWN support subdirs. A skill at skills/foo/
// with skills/foo/scripts/build.sh: the dir skills/foo/scripts/ has a file and no
// SKILL.md, but it BELONGS to foo (§10 convention), it is not a near-skill. The
// near-skill walk MUST skip a skill dir's entire subtree. The trick: when WalkDir
// enters a skill dir, return filepath.SkipDir — WalkDir then never visits its
// scripts/references/assets subdirs, so they cannot be false-positived.
//   RIGHT: if skillDirs[path] { return filepath.SkipDir }
//   WRONG: flag every dir-with-files-and-no-SKILL.md  // flags skills/foo/scripts/.

// GOTCHA #4 — a pure grouping folder (only subdirs, no files) is NOT a near-skill.
// skills/writing/ that contains only skills/writing/reddit/ (a subdir) has no
// regular file → not flagged. Detect "has a file" with os.ReadDir + any !IsDir()
// entry. The skillsDir ROOT itself is never flagged (relTag would be "."); guard
// `if path == skillsDir { return nil }` (still descends).

// GOTCHA #5 — check prints its report to STDOUT (it is a DIAGNOSTIC mode). PRD
// §6.4 "print nothing to stdout on failure" applies to PATH-output modes (<tag>,
// --all, --path) where stdout is consumed as a path by $(...). check's stdout IS
// the report (like --list's table). The §6.1 table confirms: stdout = "Report: OK
// lines + any WARN/ERROR lines". ONLY the infrastructure-failure path (Find/Index
// error) prints to stderr + empties stdout + exit 1.
//   RIGHT: findings -> stdout; exit 0/1 by error count.
//   WRONG: print findings to stderr "because check can fail".  // loses the report.

// GOTCHA #6 — exit code is driven by ERROR count only; WARNs never fail. A store
// whose only problem is a >1024 description prints a WARN line and exits 0. Count
// LevelError findings (incl. near-skill + dupe errors); if >0 -> exit 1, else 0.

// GOTCHA #7 — `check` is a SUBCOMMAND (positional token), not a flag. PRD §6.1
// row is `skpp check`, not `skpp --check`. parseArgs gets `case "check": c.check =
// true` (bare token, no dash, no value). It works in any position pre-M5; M5
// (P1.M5.T11) enforces §6.3 exclusivity (check vs --list/--search/tags -> exit 2).

// GOTCHA #8 — check is a NEW package (internal/check), NOT main.go. It has its
// own data model (Level/Finding), a new filesystem walk, cross-skill aggregation
// (duplicate names), and a documented output format — categorically larger than
// P1.M4.T1.S1's stateless ~10-line --search filter (which correctly stayed in
// main.go). It mirrors internal/resolve (pure core + typed model + _test.go).
// The go_architecture package MAP omits it only because it predates this subtask;
// the item explicitly offers "Create internal/check" as the first option.

// GOTCHA #9 — emit ONE Finding PER VIOLATION (linter-style), not one line per
// skill. A skill with a bad name AND a >1024 description yields an ERROR line AND
// a WARN line. A clean skill yields exactly one OK line. PRD §9's "one line per
// skill" describes the clean-path happy case; per-violation is the useful,
// standard linter behavior and passes the item's per-case test matrix.

// GOTCHA #10 — duplicate-name detection is GLOBAL (across all skills), computed
// AFTER the per-skill pass. Collect name->[]relTag for non-empty names; any name
// with >1 relTag yields an ERROR per member (naming the OTHER relTags, sorted, so
// the message is deterministic). Empty names are EXCLUDED (a missing name is a
// rule-2 error, not a "duplicate"). Do not dedupe within the per-skill loop.

// GOTCHA #11 — Description length uses raw len(s.Description) (bytes, verbatim
// incl. any folded-scalar trailing newline). 1024 chars + trailing newline stores
// 1025 -> WARNs. Acceptable: the item's test uses a clearly-over value; do NOT
// TrimSpace for the length check (only for the emptiness check in rule 2). The
// boundary (exactly 1024) is OK; 1025 is WARN.

// GOTCHA #12 — go.mod/go.sum UNCHANGED. The only new import is stdlib `regexp`
// (in internal/check). yaml.v3 stays the sole direct dep. `go mod tidy` is a
// no-op. VERIFY: `go mod tidy && git diff --quiet go.mod go.sum` exits 0.
```

---

## Implementation Blueprint

### New file — `internal/check/check.go` (CREATE, verbatim)

```go
// Package check validates the skills store against PRD §9 and produces the
// human-readable `skpp check` report (OK/WARN/ERROR lines + a summary). It is the
// logic behind the `skpp check` subcommand.
//
// check CONSUMES discover: it takes the already-built []discover.Skill (from
// discover.Index) for the frontmatter rules, and does its OWN filepath.WalkDir
// over the skills dir to find "near-skill" directories (dirs that contain files
// but lack a SKILL.md) — discover.Index returns ONLY dirs that HAVE a SKILL.md, so
// the missing-SKILL.md detection is check's responsibility (PRD §9 rule 1).
//
// check is a LEAF consumer (only main's `check` subcommand calls it). It owns its
// own data model (Level, Finding) and its own output format (Finding.String,
// SummaryLine) because the §9 report format IS the documented contract. The pure
// core (Validate) is unit-testable with []discover.Skill literals and no disk,
// mirroring internal/resolve.
package check

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

// descMaxChars is the PRD §9 / Agent Skills spec limit on the description field.
// A description strictly longer than this is a WARN (never blocks resolution).
const descMaxChars = 1024

// nameMaxLen is the max length of a frontmatter `name` (Agent Skills spec).
const nameMaxLen = 64

// nameRe encodes the Agent Skills charset/shape rules for a `name`: lowercase
// a-z, 0-9, hyphens; first and last char alphanumeric; single-char names allowed.
//
// GOTCHA: nameRe alone does NOT forbid consecutive hyphens — "a--b" matches it
// (the [a-z0-9-]* class greedily matches "--"; verified empirically). The
// explicit !strings.Contains(name, "--") check in validName is therefore
// LOAD-BEARING. See research/verified_facts.md §1.
var nameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Level is the severity of a Finding. Its zero value is LevelOK.
type Level int

const (
	// LevelOK marks a clean skill (one OK line). It is not an error or warning.
	LevelOK Level = iota
	// LevelWarn marks a non-blocking advisory (e.g. description > 1024 chars).
	LevelWarn
	// LevelError marks a blocking problem (missing fields, bad name, duplicate,
	// no SKILL.md). Any LevelError finding makes `skpp check` exit 1.
	LevelError
)

// label returns the fixed-width report-line prefix for a Level.
func (l Level) label() string {
	switch l {
	case LevelOK:
		return "OK"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "?"
	}
}

// Finding is one line of the check report. A clean skill yields a single LevelOK
// Finding; a skill with problems yields one Finding PER violation (linter-style);
// a near-skill dir yields a LevelError Finding. Name is used only for the OK
// line; Detail is the message for WARN/ERROR lines.
type Finding struct {
	Level  Level
	RelTag string
	Name   string
	Detail string
}

// String renders the finding as one report line (PRD §9 format):
//
//	OK    <relTag> (<name>)
//	WARN  <relTag>: <detail>
//	ERROR <relTag>: <detail>
//
// The level label is left-padded to width 5 so relTags align in a column.
func (f Finding) String() string {
	if f.Level == LevelOK {
		name := f.Name
		if name == "" {
			name = "(none)"
		}
		return fmt.Sprintf("%-5s %s (%s)", f.Level.label(), f.RelTag, name)
	}
	return fmt.Sprintf("%-5s %s: %s", f.Level.label(), f.RelTag, f.Detail)
}

// Run validates the skills dir end-to-end: it checks every skill in `skills`
// (from discover.Index) against the PRD §9 frontmatter rules, and separately
// walks skillsDir for near-skill dirs (files present but no SKILL.md). Findings
// are returned sorted by RelTag (stable), so a skill's problems stay grouped and
// near-skill ERRORs interleave by path with skill lines. main prints them, then
// the summary, then sets the exit code from Count.
//
// skillsDir is made absolute first (matching discover.Index). An unreadable
// subtree during the near-skill walk is skipped (the walk continues); a missing
// root is the caller's job to catch via discover.Index's error (main does).
func Run(skillsDir string, skills []discover.Skill) []Finding {
	root, err := filepath.Abs(skillsDir)
	if err != nil {
		root = skillsDir // best-effort; main already validated the dir via Index
	}
	findings := Validate(skills)
	findings = append(findings, nearSkillFindings(root, skills)...)
	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].RelTag < findings[j].RelTag
	})
	return findings
}

// Validate applies the PRD §9 frontmatter rules (2 missing fields, 3 name shape,
// 4 duplicate name, 5 description length) to `skills`. It is PURE: no I/O. For
// each skill it emits one Finding per violation, or a single LevelOK Finding when
// the skill is clean. Duplicate-name errors (rule 4) are computed across the
// whole set and appended after the per-skill pass.
func Validate(skills []discover.Skill) []Finding {
	var findings []Finding
	for _, s := range skills {
		var probs []Finding
		if !s.HasFM {
			// Rule 2: no frontmatter block at all -> name & description are both
			// implicitly absent; one clear message.
			probs = append(probs, mkErr(s.RelTag, "SKILL.md has no frontmatter block (missing '---')"))
		} else {
			// Rule 2: required fields present-but-empty.
			if s.Name == "" {
				probs = append(probs, mkErr(s.RelTag, "missing frontmatter name"))
			} else if !validName(s.Name) {
				// Rule 3: name shape (only when a name is present).
				probs = append(probs, mkErr(s.RelTag, fmt.Sprintf("name %q violates naming rules (lowercase a-z0-9, 1-%d chars, no leading/trailing/consecutive hyphens)", s.Name, nameMaxLen)))
			}
			if strings.TrimSpace(s.Description) == "" {
				probs = append(probs, mkErr(s.RelTag, "missing or empty frontmatter description"))
			} else if len(s.Description) > descMaxChars {
				// Rule 5: description length (WARN; non-blocking).
				probs = append(probs, mkWarn(s.RelTag, fmt.Sprintf("description is %d chars (max %d)", len(s.Description), descMaxChars)))
			}
		}
		if len(probs) == 0 {
			findings = append(findings, Finding{Level: LevelOK, RelTag: s.RelTag, Name: s.Name})
		} else {
			findings = append(findings, probs...)
		}
	}
	// Rule 4: duplicate frontmatter name across skills (computed globally).
	findings = append(findings, duplicateNameFindings(skills)...)
	return findings
}

// duplicateNameFindings returns ERROR findings for every skill whose frontmatter
// name is shared by another skill (PRD §9 rule 4). Only non-empty names count: a
// missing name is a rule-2 error, not a "duplicate". Each duplicated group yields
// one ERROR per member, naming the OTHER members' relTags (sorted) so the user
// can find them and the message is deterministic. Returns empty when clean.
func duplicateNameFindings(skills []discover.Skill) []Finding {
	byName := map[string][]string{} // name -> relTags (input order)
	for _, s := range skills {
		if s.Name == "" {
			continue
		}
		byName[s.Name] = append(byName[s.Name], s.RelTag)
	}
	var dupes []string
	for name := range byName {
		if len(byName[name]) > 1 {
			dupes = append(dupes, name)
		}
	}
	sort.Strings(dupes) // deterministic output regardless of input order
	var findings []Finding
	for _, name := range dupes {
		tags := byName[name]
		sort.Strings(tags) // deterministic
		for i, tag := range tags {
			others := append(append([]string{}, tags[:i]...), tags[i+1:]...)
			findings = append(findings, mkErr(tag, fmt.Sprintf("duplicate frontmatter name %q (also: %s)", name, strings.Join(others, ", "))))
		}
	}
	return findings
}

// validName reports whether name satisfies the Agent Skills naming rules (PRD §9
// rule 3): length 1-64; lowercase a-z, 0-9, hyphens only; no leading, trailing,
// or CONSECUTIVE hyphens.
//
// GOTCHA: nameRe alone does NOT forbid consecutive hyphens ("a--b" matches it;
// verified). The explicit !strings.Contains(name, "--") check is LOAD-BEARING.
// external_deps.md §1 recommends exactly this belt-and-braces approach.
func validName(name string) bool {
	if len(name) < 1 || len(name) > nameMaxLen {
		return false
	}
	if !nameRe.MatchString(name) {
		return false
	}
	if strings.Contains(name, "--") {
		return false
	}
	return true
}

// nearSkillFindings walks skillsDir to find "near-skill" directories: dirs that
// contain at least one regular file but no SKILL.md (PRD §9 rule 1). discover.Index
// returns only dirs that HAVE a SKILL.md, so this walk is check's job.
//
// A dir D is flagged when: D is not itself a skill dir, D is not a descendant of
// a skill dir, and D contains >=1 regular file. The descendant case is handled by
// returning filepath.SkipDir the moment WalkDir enters a skill dir — WalkDir then
// never visits its scripts/references/assets subdirs, so they cannot be flagged.
// A pure grouping folder (only subdirs, no files) is not flagged. The skillsDir
// root itself is never flagged.
func nearSkillFindings(skillsDir string, skills []discover.Skill) []Finding {
	skillDirs := make(map[string]bool, len(skills))
	for _, s := range skills {
		skillDirs[s.Dir] = true
	}
	var findings []Finding
	_ = filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entry: skip, keep walking
		}
		if !d.IsDir() {
			return nil
		}
		if path == skillsDir {
			return nil // the root is never flagged (relTag would be ".")
		}
		if skillDirs[path] {
			// A skill dir: it has SKILL.md (not an error here) and its subtree
			// (scripts/, references/, assets/) is the skill's own content — skip it.
			return filepath.SkipDir
		}
		if hasRegularFile(path) {
			findings = append(findings, mkErr(relTagOf(skillsDir, path), "directory has files but no SKILL.md"))
		}
		return nil // keep walking (descend into non-skill subdirs)
	})
	return findings
}

// hasRegularFile reports whether dir contains at least one non-directory entry.
// Distinguishes a "near-skill" (files present, no SKILL.md) from a pure grouping
// folder (only subdirs). An unreadable dir yields false.
func hasRegularFile(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			return true
		}
	}
	return false
}

// relTagOf returns path relative to skillsDir with separators normalized to '/',
// matching discover.Index's RelTag shape so near-skill ERROR lines align with
// skill OK lines.
func relTagOf(skillsDir, path string) string {
	rel, err := filepath.Rel(skillsDir, path)
	if err != nil {
		return path // unreachable for paths under skillsDir
	}
	return filepath.ToSlash(rel)
}

// mkErr / mkWarn are small constructors that keep call sites readable.
func mkErr(relTag, detail string) Finding {
	return Finding{Level: LevelError, RelTag: relTag, Detail: detail}
}
func mkWarn(relTag, detail string) Finding {
	return Finding{Level: LevelWarn, RelTag: relTag, Detail: detail}
}

// Count returns the number of LevelError and LevelWarn findings. LevelOK findings
// are ignored. main uses the error count to set the exit code (0 if none, else 1).
func Count(findings []Finding) (errors, warnings int) {
	for _, f := range findings {
		switch f.Level {
		case LevelError:
			errors++
		case LevelWarn:
			warnings++
		}
	}
	return errors, warnings
}

// SummaryLine renders the final "N skills, M errors, K warnings" line (PRD §9).
// nSkills is the discovered-skill count (len of discover.Index output); errors
// and warnings are derived from findings via Count. nSkills does NOT count
// near-skill dirs (those are not skills).
func SummaryLine(nSkills int, findings []Finding) string {
	errs, warns := Count(findings)
	return fmt.Sprintf("%d skills, %d errors, %d warnings", nSkills, errs, warns)
}
```

### Edit 1 — `main.go` imports: add `internal/check`

Add the check import to the internal-packages group (it sorts after `discover`
alphabetically: check < discover < skillsdir < ui — place it FIRST in that group).
After P1.M4.T1.S1, the import block is:

```go
import (
	"fmt"
	"io"
	"os"
	"strings" // present after P1.M4.T1.S1

	"github.com/dabstractor/skpp/internal/check"
	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
)
```

> If P1.M4.T1.S1 has NOT landed, the import block currently is `fmt`/`io`/`os` +
> discover/skillsdir/ui (no `"strings"`). Apply P1.M4.T1.S1 first; this edit only
> ADDS the `check` line. `gofmt -w main.go` settles group ordering regardless.

### Edit 2 — `main.go` config struct: add `check bool`

Append `check bool` to the active fields (move it out of the "Future" comment):

```go
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	list    bool // --list / -l    : print the human-readable catalog table (§6.1)
	noColor bool // --no-color     : disable ANSI color even on a TTY (§6.2)
	// --search / -s <q> (present after P1.M4.T1.S1)
	search    string
	searchSet bool
	// `skpp check` subcommand: validate the skills store per PRD §9.
	check bool
	// Future (M3/M5), do NOT add yet:
	//   all bool; file, relative, help bool; tags []string
}
```

### Edit 3 — `main.go` parseArgs: recognize the `check` subcommand token

parseArgs is already an index loop after P1.M4.T1.S1. Add a `case "check":` (a
bare positional token, no value capture). Insert it alongside the other cases:

```go
		case "--no-color":
			c.noColor = true
		case "--search", "-s":
			c.searchSet = true
			if i+1 < len(args) {
				c.search = args[i+1]
				i++ // consume the query value
			}
		case "check":
			// `skpp check` subcommand (PRD §9): validate the store. Bare
			// positional token, no value. Recognized in any position pre-M5;
			// P1.M5.T11 enforces §6.3 exclusivity (check vs --list/--search/tags).
			c.check = true
		default:
```

### Edit 4 — `main.go` run(): add the check block AFTER the search block, BEFORE the default `return 1`

Insert this block after the `if c.searchSet { ... }` block (which P1.M4.T1.S1
added) and before the final `return 1`:

```go
	if c.check {
		// PRD §9 `skpp check`: resolve the store, build the index, validate every
		// skill (frontmatter rules) + walk for near-skill dirs (files but no
		// SKILL.md), print the OK/WARN/ERROR report + summary to stdout, and exit
		// 0 if no ERROR else 1. WARNs never affect the exit code. check is a
		// DIAGNOSTIC mode: its report goes to stdout (like --list), NOT the §6.4
		// clean-stdout path (that is for path-output modes).
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // verbatim one-line fix (PRD §6.4/§8.4)
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		findings := check.Run(dir, skills)
		for _, f := range findings {
			fmt.Fprintln(stdout, f) // Finding implements Stringer -> §9 line format
		}
		fmt.Fprintln(stdout, check.SummaryLine(len(skills), findings))
		errs, _ := check.Count(findings)
		if errs > 0 {
			return 1
		}
		return 0
	}
```

> **Stringer note:** `fmt.Fprintln(stdout, f)` calls `f.String()` because Finding
> implements `String() string`. If you prefer to be explicit, `fmt.Fprintln(stdout,
> f.String())` is equivalent. Either prints the §9 line + a newline.

### Edit 5 — `internal/check/check_test.go` (CREATE, verbatim)

```go
package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

// --- validName (rule 3, incl. the consecutive-hyphen regression) ---

func TestValidName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"ok", true},
		{"my-skill", true},
		{"a", true},          // single char allowed
		{"a-b", true},
		{"a-b-c", true},
		{"skill1", true},
		{"1skill", true},     // leading digit allowed
		{strings.Repeat("a", 64), true},  // exactly 64 chars
		{"", false},                          // empty
		{"-ab", false},                       // leading hyphen
		{"ab-", false},                       // trailing hyphen
		{"-a", false},                        // leading hyphen
		{"a--b", false},                      // CRITICAL: consecutive hyphens (regex alone matches!)
		{"double--hyphen", false},            // consecutive hyphens
		{"triple---x", false},                // consecutive hyphens
		{"Ab", false},                        // uppercase
		{"aB", false},                        // uppercase
		{"has space", false},                 // space
		{"under_score", false},               // underscore
		{strings.Repeat("a", 65), false},     // too long (65)
	}
	for _, c := range cases {
		if got := validName(c.name); got != c.ok {
			t.Errorf("validName(%q)=%v; want %v", c.name, got, c.ok)
		}
	}
}

// --- Validate: per-skill frontmatter rules (pure, no disk) ---

// A clean skill (valid name + non-empty description <= 1024) -> one OK finding.
func TestValidateCleanSkillOK(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "a", Name: "my-skill", Description: "does the thing", HasFM: true},
	})
	if len(findings) != 1 || findings[0].Level != LevelOK {
		t.Fatalf("Validate(clean)=%+v; want exactly one LevelOK finding", findings)
	}
	if findings[0].Name != "my-skill" {
		t.Errorf("OK finding Name=%q; want my-skill", findings[0].Name)
	}
}

// No frontmatter block (HasFM=false) -> ERROR "no frontmatter block".
func TestValidateNoFrontmatterBlock(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "bare", HasFM: false},
	})
	if !hasErrorContaining(findings, "no frontmatter block") {
		t.Errorf("Validate(no-FM)=%+v; want an ERROR mentioning 'no frontmatter block'", findings)
	}
}

func TestValidateMissingName(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "a", Name: "", Description: "has desc", HasFM: true},
	})
	if !hasErrorContaining(findings, "missing frontmatter name") {
		t.Errorf("Validate(missing name)=%+v; want an ERROR 'missing frontmatter name'", findings)
	}
}

func TestValidateMissingDescription(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "a", Name: "ok", Description: "", HasFM: true},
	})
	if !hasErrorContaining(findings, "description") {
		t.Errorf("Validate(missing desc)=%+v; want an ERROR about description", findings)
	}
}

// Whitespace-only description counts as empty -> ERROR.
func TestValidateBlankDescription(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "a", Name: "ok", Description: "   \n  ", HasFM: true},
	})
	if !hasErrorContaining(findings, "description") {
		t.Errorf("Validate(blank desc)=%+v; want an ERROR about description", findings)
	}
}

// Bad name shape -> ERROR. Covers leading/trailing hyphen, uppercase, consecutive
// hyphens, too long.
func TestValidateBadName(t *testing.T) {
	for _, bad := range []string{"-bad", "bad-", "Bad", "a--b", strings.Repeat("n", 65)} {
		findings := Validate([]discover.Skill{
			{RelTag: "a", Name: bad, Description: "d", HasFM: true},
		})
		if !hasErrorContaining(findings, "naming rules") {
			t.Errorf("Validate(name=%q)=%+v; want an ERROR 'naming rules'", bad, findings)
		}
	}
}

// Boundary: exactly 1024 chars -> OK (no WARN). 1025 -> WARN.
func TestValidateDescriptionLengthBoundary(t *testing.T) {
	atLimit := strings.Repeat("x", descMaxChars) // 1024
	findings := Validate([]discover.Skill{
		{RelTag: "ok", Name: "ok", Description: atLimit, HasFM: true},
	})
	if len(findings) != 1 || findings[0].Level != LevelOK {
		t.Errorf("Validate(1024-char desc)=%+v; want one OK (boundary not exceeded)", findings)
	}

	over := strings.Repeat("x", descMaxChars+1) // 1025
	findings = Validate([]discover.Skill{
		{RelTag: "big", Name: "ok", Description: over, HasFM: true},
	})
	if !hasWarnContaining(findings, "chars") {
		t.Errorf("Validate(1025-char desc)=%+v; want a WARN about chars", findings)
	}
}

// Duplicate frontmatter name across two skills -> both ERROR "duplicate".
func TestValidateDuplicateNames(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "writing/a", Name: "dup", Description: "d", HasFM: true},
		{RelTag: "writing/b", Name: "dup", Description: "d", HasFM: true},
	})
	dupeErrs := 0
	for _, f := range findings {
		if f.Level == LevelError && strings.Contains(f.Detail, "duplicate") {
			dupeErrs++
		}
	}
	if dupeErrs != 2 {
		t.Errorf("Validate(dup names): duplicate ERRORs=%d; want 2 (one per skill)", dupeErrs)
	}
}

// Distinct names -> no duplicate error.
func TestValidateDistinctNamesNoDupe(t *testing.T) {
	findings := Validate([]discover.Skill{
		{RelTag: "a", Name: "one", Description: "d", HasFM: true},
		{RelTag: "b", Name: "two", Description: "d", HasFM: true},
	})
	for _, f := range findings {
		if strings.Contains(f.Detail, "duplicate") {
			t.Errorf("Validate(distinct names): unexpected duplicate finding %+v", f)
		}
	}
}

// --- Finding.String / SummaryLine / Count (format) ---

func TestFindingStringFormats(t *testing.T) {
	okLine := Finding{Level: LevelOK, RelTag: "a/b", Name: "name"}.String()
	if !strings.HasPrefix(okLine, "OK") || !strings.Contains(okLine, "a/b") || !strings.Contains(okLine, "(name)") {
		t.Errorf("OK line=%q; want OK + relTag + (name)", okLine)
	}
	warnLine := Finding{Level: LevelWarn, RelTag: "a/b", Detail: "too long"}.String()
	if !strings.HasPrefix(warnLine, "WARN") || !strings.Contains(warnLine, "a/b") || !strings.Contains(warnLine, "too long") {
		t.Errorf("WARN line=%q; want WARN + relTag + detail", warnLine)
	}
	errLine := Finding{Level: LevelError, RelTag: "a/b", Detail: "bad"}.String()
	if !strings.HasPrefix(errLine, "ERROR") || !strings.Contains(errLine, "a/b") || !strings.Contains(errLine, ": bad") {
		t.Errorf("ERROR line=%q; want ERROR + relTag + : detail", errLine)
	}
}

func TestSummaryLine(t *testing.T) {
	findings := []Finding{
		{Level: LevelOK},
		{Level: LevelError},
		{Level: LevelWarn},
		{Level: LevelError},
	}
	got := SummaryLine(3, findings)
	want := "3 skills, 2 errors, 1 warnings"
	if got != want {
		t.Errorf("SummaryLine=%q; want %q", got, want)
	}
}

func TestCount(t *testing.T) {
	findings := []Finding{
		{Level: LevelOK}, {Level: LevelError}, {Level: LevelWarn}, {Level: LevelWarn},
	}
	errs, warns := Count(findings)
	if errs != 1 || warns != 2 {
		t.Errorf("Count=%d,%d; want 1,2", errs, warns)
	}
}

// --- nearSkillFindings (disk) ---

// writeStore builds a temp skills dir mirroring writeSkillTree in main_test.go.
// extraFiles adds NON-SKILL.md files (to create near-skill dirs). Returns the
// root (absolute).
func writeStore(t *testing.T, skills map[string]string, extraFiles map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	for relTag, content := range skills {
		dir := filepath.Join(root, filepath.FromSlash(relTag))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write SKILL.md %s: %v", dir, err)
		}
	}
	for relDir, files := range extraFiles {
		dir := filepath.Join(root, filepath.FromSlash(relDir))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
				t.Fatalf("write %s/%s: %v", dir, f, err)
			}
		}
	}
	return root
}

// A dir with a loose file and no SKILL.md (not under any skill) -> ERROR.
func TestNearSkillWithFilesFlagged(t *testing.T) {
	root := writeStore(t,
		map[string]string{"real": "---\nname: real\ndescription: d\n---\nbody\n"},
		map[string][]string{"drafts": {"notes.txt"}}, // near-skill: file, no SKILL.md
	)
	f := nearSkillFindings(root, []discover.Skill{{Dir: filepath.Join(root, "real"), RelTag: "real"}})
	if !hasErrorContaining(f, "no SKILL.md") {
		t.Errorf("nearSkillFindings=%+v; want an ERROR 'no SKILL.md' for drafts/", f)
	}
	// And the relTag of the finding should be "drafts".
	found := false
	for _, finding := range f {
		if finding.RelTag == "drafts" {
			found = true
		}
	}
	if !found {
		t.Errorf("nearSkillFindings: no finding with RelTag=drafts; got %+v", f)
	}
}

// A skill's scripts/ subdir must NOT be flagged (SkipDir on the skill dir prunes it).
func TestNearSkillSkillSubtreeNotFlagged(t *testing.T) {
	root := writeStore(t,
		map[string]string{"foo": "---\nname: foo\ndescription: d\n---\nbody\n"},
		map[string][]string{"foo/scripts": {"build.sh"}}, // support dir of the foo skill
	)
	f := nearSkillFindings(root, []discover.Skill{{Dir: filepath.Join(root, "foo"), RelTag: "foo"}})
	if len(f) != 0 {
		t.Errorf("nearSkillFindings on a clean skill store produced findings (scripts/ must be skipped): %+v", f)
	}
}

// A pure grouping folder (only subdirs, no files) is NOT flagged.
func TestNearSkillGroupingDirNotFlagged(t *testing.T) {
	root := writeStore(t,
		map[string]string{"group/inner": "---\nname: inner\ndescription: d\n---\nbody\n"},
		nil, // no extra files; "group/" contains only the "inner" subdir
	)
	f := nearSkillFindings(root, []discover.Skill{{Dir: filepath.Join(root, "group/inner"), RelTag: "group/inner"}})
	for _, finding := range f {
		if finding.RelTag == "group" {
			t.Errorf("nearSkillFindings flagged a grouping folder (group/): %+v", finding)
		}
	}
}

// --- Run (integration: combines Validate + nearSkill, sorted) ---

func TestRunCombinesAndSorts(t *testing.T) {
	root := writeStore(t,
		map[string]string{
			"b": "---\nname: b-name\ndescription: d\n---\nbody\n",
			"a": "---\nname: a-name\ndescription: d\n---\nbody\n",
		},
		map[string][]string{"z-drafts": {"todo.md"}}, // near-skill
	)
	skills := []discover.Skill{
		{Dir: filepath.Join(root, "b"), RelTag: "b", Name: "b-name", Description: "d", HasFM: true, SourceFile: filepath.Join(root, "b", "SKILL.md")},
		{Dir: filepath.Join(root, "a"), RelTag: "a", Name: "a-name", Description: "d", HasFM: true, SourceFile: filepath.Join(root, "a", "SKILL.md")},
	}
	findings := Run(root, skills)
	// Findings sorted by RelTag: a (OK), b (OK), z-drafts (ERROR).
	if len(findings) < 3 {
		t.Fatalf("Run: expected >=3 findings (a OK, b OK, z-drafts ERROR); got %+v", findings)
	}
	tags := []string{}
	for _, f := range findings {
		tags = append(tags, f.RelTag)
	}
	// Must be ascending by RelTag.
	for i := 1; i < len(tags); i++ {
		if tags[i-1] > tags[i] {
			t.Errorf("Run findings not sorted by RelTag: %v", tags)
			break
		}
	}
}

// --- helpers ---

func hasErrorContaining(findings []Finding, sub string) bool {
	for _, f := range findings {
		if f.Level == LevelError && strings.Contains(f.Detail, sub) {
			return true
		}
	}
	return false
}

func hasWarnContaining(findings []Finding, sub string) bool {
	for _, f := range findings {
		if f.Level == LevelWarn && strings.Contains(f.Detail, sub) {
			return true
		}
	}
	return false
}
```

### Edit 6 — `main_test.go`: APPEND 7 integration tests (reuse existing helpers)

Append to the END of main_test.go. Reuses `writeSkillTree` + `unsetSkillsEnv`
(already defined). To create a near-skill dir (files, no SKILL.md) the test writes
a loose file with `os.MkdirAll`+`os.WriteFile` directly.

```go
// --- run: `skpp check` (P1.M4.T10 / §9) ---

// `check` is recognized as a positional subcommand.
func TestRunCheckSubcommandRecognized(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: A demo skill.\n---\n# body\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(check): code=%d; want 0 (clean store)", code)
	}
	if !strings.Contains(out.String(), "example") {
		t.Errorf("run(check) stdout missing the example tag:\n%s", out.String())
	}
}

// Clean store -> OK line + summary, exit 0.
func TestRunCheckCleanExit0(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: A demo skill.\n---\n# body\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(check) clean: code=%d; want 0", code)
	}
	got := out.String()
	for _, want := range []string{"OK", "example", "(example)", "1 skills, 0 errors, 0 warnings"} {
		if !strings.Contains(got, want) {
			t.Errorf("run(check) clean stdout missing %q:\n%s", want, got)
		}
	}
	if errOut.Len() != 0 {
		t.Errorf("run(check) clean stderr=%q; want empty", errOut.String())
	}
}

// Bad name -> ERROR line, exit 1.
func TestRunCheckBadNameExit1(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"bad": "---\nname: Bad-Name\ndescription: d\n---\nbody\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(check) bad name: code=%d; want 1", code)
	}
	if !strings.Contains(out.String(), "ERROR") || !strings.Contains(out.String(), "naming rules") {
		t.Errorf("run(check) bad name stdout missing ERROR/naming-rules:\n%s", out.String())
	}
}

// A dir with files but no SKILL.md -> ERROR, exit 1.
func TestRunCheckNearSkillDirExit1(t *testing.T) {
	root := t.TempDir()
	// A real skill.
	if err := os.MkdirAll(filepath.Join(root, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real", "SKILL.md"), []byte("---\nname: real\ndescription: d\n---\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A near-skill dir: a loose file, no SKILL.md.
	if err := os.MkdirAll(filepath.Join(root, "drafts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "drafts", "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKPP_SKILLS_DIR", root)
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(check) near-skill: code=%d; want 1", code)
	}
	if !strings.Contains(out.String(), "no SKILL.md") || !strings.Contains(out.String(), "drafts") {
		t.Errorf("run(check) near-skill stdout missing no-SKILL.md/drafts:\n%s", out.String())
	}
}

// Two skills with the same frontmatter name -> ERROR duplicate, exit 1.
func TestRunCheckDuplicateNameExit1(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"a": "---\nname: dup\ndescription: d\n---\nx\n",
		"b": "---\nname: dup\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(check) dup name: code=%d; want 1", code)
	}
	if !strings.Contains(out.String(), "duplicate") {
		t.Errorf("run(check) dup name stdout missing 'duplicate':\n%s", out.String())
	}
}

// > 1024-char description -> WARN line, but exit 0 (warns don't fail).
func TestRunCheckLongDescriptionWarnExit0(t *testing.T) {
	long := strings.Repeat("x", 1025)
	dir := writeSkillTree(t, map[string]string{
		"big": "---\nname: big\ndescription: " + long + "\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(check) long desc: code=%d; want 0 (WARN is non-blocking)", code)
	}
	if !strings.Contains(out.String(), "WARN") {
		t.Errorf("run(check) long desc stdout missing WARN:\n%s", out.String())
	}
}

// Skills dir unresolvable -> stderr one-liner, exit 1, stdout EMPTY.
func TestRunCheckSkillsDirUnresolvableExit1(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // force all three §8 rules to miss
	var out, errOut bytes.Buffer
	code := run([]string{"check"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(check) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(check) unresolvable stdout=%q; want EMPTY", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("run(check) unresolvable stderr=%q; want the one-line fix", errOut.String())
	}
}
```

> **main_test.go import note:** these tests use `bytes`, `os`, `path/filepath`,
> `strings`, `testing` — ALL already imported by main_test.go (present since
> M1/M2, and `strings` is needed by the search tests from P1.M4.T1.S1). No import
> change needed. `writeSkillTree` and `unsetSkillsEnv` are reused as-is.

### Implementation Patterns & Key Details

```go
// PATTERN: pure core (Validate) + I/O helper (nearSkillFindings) + combiner (Run).
//   func Validate(skills []discover.Skill) []Finding    // pure, no disk
//   func nearSkillFindings(dir, skills) []Finding       // disk walk
//   func Run(dir, skills) []Finding                     // Validate+near, sorted
// WHY: mirrors internal/resolve (pure Resolve + typed model). Validate is unit-
//      testable with struct literals; nearSkillFindings with temp dirs; Run is the
//      thin combiner main calls. Separation = testability.

// PATTERN: SkipDir on a skill dir to prune its subtree (near-skill walk).
//   if skillDirs[path] { return filepath.SkipDir }
// WHY: a skill's scripts/references/assets are its OWN content, not near-skills.
//      Skipping the whole subtree means they are never visited -> never flagged.
//      A pure grouping folder (only subdirs) is visited but has no regular file ->
//      not flagged.

// PATTERN: belt-and-braces name validation (regex + len + explicit "--").
//   if len < 1 || len > 64 || !nameRe.MatchString(name) || strings.Contains(name,"--")
// WHY: the regex ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ does NOT forbid consecutive
//      hyphens (verified; "a--b" matches). The explicit "--" check is load-bearing.

// PATTERN: emit one Finding PER VIOLATION (linter-style); clean skill -> one OK.
// WHY: a skill can have multiple problems (bad name + long desc). Per-violation is
//      the useful, standard behavior and passes the item's per-case test matrix.

// PATTERN: diagnostic-mode output discipline (check -> stdout; exit by error count).
//   for _, f := range findings { fmt.Fprintln(stdout, f) }
//   fmt.Fprintln(stdout, check.SummaryLine(...))
//   if errs > 0 { return 1 }; return 0
// WHY: check is a DIAGNOSTIC mode (like --list): its report IS the output, to
//      stdout. §6.4 clean-stdout applies only to PATH-output modes. Exit 1 iff any
//      ERROR; WARNs are advisory.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/check (NEW package). Imports: stdlib fmt/io/fs/os/path/filepath/regexp/
    sort/strings + internal/discover. Exports: Level, Finding (with String()),
    Run, Validate, Count, SummaryLine. Unexported: validName, nearSkillFindings,
    hasRegularFile, relTagOf, mkErr, mkWarn, duplicateNameFindings.
  - main.go imports internal/check. check is a LEAF consumer (only main calls it).

go.mod / go.sum (NO change):
  - The only new import is stdlib regexp (in internal/check). yaml.v3 stays the
    sole direct dep. `go mod tidy` is a no-op.
  - VERIFY: `go mod tidy && git diff --quiet go.mod go.sum` exits 0.

UPSTREAM CONSUMERS (on disk; READ-ONLY):
  - discover.Index(dir) ([]Skill, error)        — internal/discover/index.go
  - discover.Skill{Dir,RelTag,Name,Description,HasFM,...} — internal/discover/skill.go
  - skillsdir.Find()                            — internal/skillsdir/skillsdir.go
  - main.isTerminal / main.version              — main.go (reused; check ignores color)

DOWNSTREAM CONSUMERS (later subtasks plug into this):
  - P1.M5.T11 (full CLI flag matrix + exit codes): turns `check` mixed with
    --list/--search/tags into exit 2 (mutual exclusivity, PRD §6.3). This subtask's
    `case "check"` is recognized in any position; M5 can refine to "must be alone".
  - P1.M6.T14 (README): documents `skpp check` usage (Mode B final task).
  - P1.M6.T16 (acceptance suite): runs `./skpp check` and expects exit 0 + OK.

NO CHANGES TO:
  - PRD.md, go.mod, go.sum, .gitignore, internal/discover/*, internal/skillsdir/*,
    internal/resolve/*, internal/ui/*, skills/ (none yet).
```

---

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
# Run after creating/editing all files — fix before proceeding.
gofmt -w internal/check/check.go internal/check/check_test.go main.go main_test.go
gofmt -l internal/check/*.go main.go main_test.go   # MUST print nothing
go vet ./...                                         # MUST be clean

# Expected: zero gofmt findings, zero vet findings.
```

### Level 2: Unit Tests (Component Validation)

```bash
# The new check package tests.
go test ./internal/check/ -v

# The main package (incl. the new check integration tests + everything else).
go test . -run 'Check' -v          # the new run() tests
go test . -v                       # full main package

# Whole module (regression: nothing else broke).
go test ./...
go test ./... -v 2>&1 | grep -c '^--- PASS'   # MUST be 105 + (13 if search landed) + 24 new

# Expected: all pass. If a validName test fails on "a--b", you forgot the explicit
# '--' check (GOTCHA #1). If nearSkill flagged a scripts/ dir, you forgot SkipDir
# (GOTCHA #3).
```

### Level 3: Integration Testing (System Validation)

```bash
# Build the binary.
go build -o /tmp/skpp .

# Seed a store with a clean skill + a near-skill dir + a bad-name skill.
ROOT=$(mktemp -d)
mkdir -p "$ROOT/skills/example" "$ROOT/skills/bad" "$ROOT/skills/drafts"
cat > "$ROOT/skills/example/SKILL.md" <<'EOF'
---
name: example
description: A demo skill.
---
# body
EOF
cat > "$ROOT/skills/bad/SKILL.md" <<'EOF'
---
name: Bad-Name
description: d
---
x
EOF
echo "loose" > "$ROOT/skills/drafts/notes.txt"   # near-skill: file, no SKILL.md
export SKPP_SKILLS_DIR="$ROOT/skills"

# Run check: expect OK for example, ERROR for bad, ERROR for drafts, summary, exit 1.
/tmp/skpp check; echo "exit=$?"
#   OK    example (example)
#   ERROR bad: name "Bad-Name" violates naming rules ...
#   ERROR drafts: directory has files but no SKILL.md
#   2 skills, 2 errors, 0 warnings
#   exit=1

# Clean store -> exit 0.
rm -rf "$ROOT/skills/bad" "$ROOT/skills/drafts"
/tmp/skpp check; echo "exit=$?"   # OK example + summary, exit 0

# > 1024 description -> WARN, exit 0.
mkdir -p "$ROOT/skills/big"
{ printf -- '---\nname: big\ndescription: '; head -c 1025 < /dev/zero | tr '\0' 'x'; printf '\n---\nx\n'; } > "$ROOT/skills/big/SKILL.md"
/tmp/skpp check; echo "exit=$?"   # WARN big + exit 0

# Skills dir unresolvable -> stderr one-liner, exit 1, stdout EMPTY.
env -u SKPP_SKILLS_DIR /tmp/skpp check; echo "exit=${PIPESTATUS[0]}"   # exit 1

# Duplicate names -> both ERROR.
mkdir -p "$ROOT/skills/x" "$ROOT/skills/y"
printf -- '---\nname: dup\ndescription: d\n---\nx\n' > "$ROOT/skills/x/SKILL.md"
printf -- '---\nname: dup\ndescription: d\n---\nx\n' > "$ROOT/skills/y/SKILL.md"
/tmp/skpp check | grep duplicate; echo "exit=${PIPESTATUS[0]}"   # duplicate lines, exit 1

# A skill's scripts/ subdir must NOT be flagged.
mkdir -p "$ROOT/skills/clean/scripts"
printf -- '---\nname: clean\ndescription: d\n---\nx\n' > "$ROOT/skills/clean/SKILL.md"
echo '#!/bin/sh' > "$ROOT/skills/clean/scripts/run.sh"
/tmp/skpp check | grep -c scripts   # expect 0 (scripts/ never appears)

# go.mod/go.sum unchanged (only stdlib regexp added).
go mod tidy && git diff --quiet go.mod go.sum && echo "go.mod/go.sum UNCHANGED"

rm -rf "$ROOT" /tmp/skpp
# Expected: every command behaves as annotated.
```

### Level 4: Creative & Domain-Specific Validation

```bash
# Scope boundary: confirm this subtask created ONLY internal/check/*, modified
# ONLY main.go + main_test.go, and changed NO go.mod/go.sum.
git status --porcelain
# expect: ?? internal/check/check.go  ?? internal/check/check_test.go
#         M main.go  M main_test.go
grep -RIn "internal/check" main.go              # expect the one import
grep -n 'case "check"' main.go                  # expect 1 (the subcommand case)
go doc ./internal/check                         # expect exported Level/Finding/Run/Validate/Count/SummaryLine

# Verify the consecutive-hyphen regression guard holds (GOTCHA #1).
go test ./internal/check/ -run TestValidName -v   # "a--b" must be false
# Expected: scope clean; check package doc shows the API; validName rejects "a--b".
```

---

## Final Validation Checklist

### Technical Validation

- [ ] `gofmt -l internal/check/*.go main.go main_test.go` silent; `go vet ./...` clean.
- [ ] `go build ./...` exit 0; `go test ./...` all pass (105 + search + ~23 new).
- [ ] `go mod tidy && git diff --quiet go.mod go.sum` exits 0 (UNCHANGED).
- [ ] Level 3 integration commands behave as annotated.

### Feature Validation

- [ ] `skpp check` recognized as a subcommand (positional token).
- [ ] Clean skill → `OK <tag> (<name>)` + exit 0.
- [ ] No-frontmatter-block / missing-name / missing-description → ERROR + exit 1.
- [ ] Bad name (charset/leading/trailing/consecutive-hyphen/>64) → ERROR + exit 1.
- [ ] `description` > 1024 → WARN + exit 0 (boundary 1024 is OK).
- [ ] Duplicate frontmatter `name` → both ERROR duplicate + exit 1.
- [ ] Dir with files but no SKILL.md → ERROR + exit 1; skill's scripts/ NOT flagged.
- [ ] Summary line `N skills, M errors, K warnings` printed last.
- [ ] Skills-dir unresolvable → stderr one-liner + exit 1 + stdout EMPTY.

### Code Quality Validation

- [ ] Follows repo conventions (plain t.Errorf, no testify, no t.Parallel; white-box main_test.go).
- [ ] `internal/check` is a new package (pure Validate + disk walk + format), mirroring resolve.
- [ ] validName uses regex + len + explicit `--` (the regex alone does NOT catch "a--b").
- [ ] near-skill walk uses `filepath.SkipDir` on skill dirs (no false positives on scripts/).
- [ ] Only stdlib `regexp` added; no new external dependency.
- [ ] Reuses existing main_test.go helpers (writeSkillTree/unsetSkillsEnv), not redefined.

### Documentation & Deployment

- [ ] Code is self-documenting (doc comments on exported Level/Finding/Run/Validate/Count/SummaryLine).
- [ ] No new env vars or config. No README change (DOCS: none — internal logic; check usage is README §6, Mode B).

---

## Anti-Patterns to Avoid

- ❌ Don't trust the name regex alone to reject consecutive hyphens — "a--b" matches it (verified); add `!strings.Contains(name, "--")`.
- ❌ Don't flag a skill's own `scripts/`/`references/`/`assets/` subdirs — `filepath.SkipDir` on skill dirs prunes them.
- ❌ Don't detect "missing SKILL.md" from `discover.Skill` structs — Index only returns dirs that HAVE SKILL.md; use the separate near-skill walk.
- ❌ Don't print the check report to stderr "because check can fail" — it is a diagnostic mode; the report goes to stdout (like --list); only infra failure hits stderr.
- ❌ Don't let WARNs affect the exit code — exit 1 iff ≥1 ERROR; a long description WARNs but exits 0.
- ❌ Don't collapse to "one line per skill" — emit one finding per violation (linter-style); clean skills get the single OK line.
- ❌ Don't compute duplicates inside the per-skill loop — it is a global reduction (collect name→[]relTag, flag groups >1); empty names are excluded.
- ❌ Don't create `--check` — it is the `check` SUBCOMMAND (positional token, PRD §6.1).
- ❌ Don't put the logic in main.go — it earns `internal/check` (data model + new walk + cross-skill aggregation + documented format).
- ❌ Don't re-parse frontmatter in check — read the pre-built Skill fields (HasFM/Name/Description); Index already parsed.
- ❌ Don't add an external dependency — only stdlib `regexp`; go.mod must stay unchanged.

---

## Confidence Score

**9/10** for one-pass implementation success.

Rationale: the task is well-bounded — one new package (pure core + a disk walk +
format helpers) plus a thin main.go wiring block. Every consumed contract
(`discover.Index`, `discover.Skill`, `skillsdir.Find`, `main.run`/`parseArgs`/
`config`) is on disk and green, with exact signatures quoted. The exact source
for `internal/check/check.go`, both test files, and all four main.go edits is
given verbatim and is gofmt-clean. The single critical gotcha (the name regex does
NOT forbid consecutive hyphens) was verified empirically and is called out in
three places with the load-bearing explicit `--` check. The near-skill walk's
false-positive risk (a skill's scripts/) is eliminated by the documented
`filepath.SkipDir` trick. The baseline (105 tests, build OK, go.mod neutral) was
measured live. The one residual risk is the parallel-execution dependency on
P1.M4.T1.S1: if `--search` has not landed, the main.go edits assume its state (the
index-loop parseArgs, the `"strings"` import); the PRP states this explicitly and
points to P1.M4.T1S1/PRP.md. The -1 is for that ordering dependency plus the
inherent chance of a fixture-string slip in the duplicate/summary assertions —
both caught immediately by `go test ./... -v` and `gofmt -l`.
