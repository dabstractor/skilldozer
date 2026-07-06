# PRP — P1.M2.T4.S1: `internal/discover` — Frontmatter type + `ParseFrontmatter` (yaml.v3, lenient)

> **Subtask:** P1.M2.T4.S1 — the FIRST subtask of T4 (frontmatter model & parser,
> PRD §7.3). This is the foundation of milestone M2 (Discovery & frontmatter
> parsing); T5 (`Index()` walk) and T6 (`--list`) build on it.
> **Scope:** create `internal/discover/discover.go` (`package discover`) and its
> white-box test `internal/discover/discover_test.go`. Define the `Frontmatter`
> struct (the SKILL.md frontmatter data model) and implement `ParseFrontmatter`
> (extract the `---`-delimited YAML block, unmarshal with `gopkg.in/yaml.v3`
> LENIENTLY, return the parsed frontmatter + the markdown body). Handle missing,
> dangling, malformed, BOM-prefixed, and CRLF-encoded inputs per PRD §7.3.
>
> **SCOPE DECISION (authoritative — see verified_facts.md §13):** the task title
> says "Frontmatter/**Skill** types", but the plan's **S2 is explicitly "metadata
> extraction + Skill type"**. To avoid stealing S2's deliverable and creating
> churn, **this subtask does NOT define `type Skill struct`** (nor `toStringSlice`,
> `BuildSkill`, or `Index()`). "Skill types" in the title refers to the frontmatter
> data model that the `Skill` type is built FROM; the `Skill` struct itself lands in
> S2. The non-overlapping split: **S1 = `Frontmatter` + `ParseFrontmatter`; S2 =
> `Skill` + metadata extraction.**
>
> **DEPENDENCY:** none on disk. `internal/discover` is a LEAF library — it imports
> only `gopkg.in/yaml.v3` (already pinned in `go.mod`/`go.sum`) and the stdlib. It
> does NOT import `internal/skillsdir`, `main`, or any other skpp package. (The
> `Index()` walk in T5 will take the skills dir as a parameter and call
> `ParseFrontmatter`.)
>
> **NOTE (main.go):** M1.T3 (`main.go` + `main_test.go`) is landed and committed.
> This is noted only to prevent confusion: it is irrelevant to THIS subtask —
> `internal/discover` has NO dependency on `main.go` (it is a leaf library). Do NOT
> create or touch `main.go` here; that is M1.T3's deliverable.

---

## Goal

**Feature Goal**: Create the SKILL.md frontmatter data model and parser that the
rest of milestone M2 (and M3/M4) is built on. `ParseFrontmatter(path)` reads a
`SKILL.md`, strips a leading UTF-8 BOM, locates the YAML block between the first
two lines that are exactly `---` (tolerating CRLF), unmarshals it into a
`Frontmatter` with `gopkg.in/yaml.v3` LENIENTLY (unknown keys ignored, per PRD
§7.3), and returns `(Frontmatter, body, error)`. Missing/dangling frontmatter is
lenient (no error, `HasFM=false`); syntactically broken YAML between valid fences
is a HARD error (propagated, so `check` in M4 can report it).

**Deliverable**: Two NEW files (no other files touched):
1. `internal/discover/discover.go` — `package discover`; `type Frontmatter`; the
   `utf8BOM` var; `func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`.
2. `internal/discover/discover_test.go` — `package discover` (white-box); table +
   per-fixture tests covering: full valid frontmatter, unknown keys ignored,
   folded scalar, quoted values, no-block, no-closing-fence, empty file,
   only-fence, BOM, CRLF, malformed YAML (hard error), missing file, and the
   `HasFM` `yaml:"-"` guard.

**Success Definition**: `gofmt -l internal/discover/*.go` is silent; `go vet
./internal/discover/` is clean; `go build ./...` and `go test ./...` pass; and
`go mod tidy` produces the EXPECTED single-line `go.mod` change (the `// indirect`
marker on the yaml.v3 `require` line disappears — see verified_facts.md §14). All
12 test cases pass. No `type Skill`, no `Index()`, no `toStringSlice`, no
`main.go`/`skillsdir`/`resolve`/`ui` touch.

---

## Why

- This subtask **proves frontmatter parsing works** before anything depends on it.
  PRD §18 lists it as build-order step 2 (right after the §8 location resolution).
  Until `ParseFrontmatter` exists and is tested against every edge case in §7.3,
  the discovery pipeline (`Index()` in T5, `--list` in T6) is blocked.
- It **locks the lenient contract** (PRD §7.3: "unknown keys ignored; missing
  optional keys → defaults; no frontmatter block → resolve by directory"). Getting
  leniency wrong now — e.g. calling `dec.KnownFields(true)` (hard-errors on
  unknown keys) or panicking on a BOM/CRLF file — silently breaks every real
  SKILL.md later. Verified empirically (research §1, §5, §6, §7, §8).
- It **locks the `Frontmatter` → `Skill` contract** that S2/T5 consume verbatim.
  `Skill` (S2) is built FROM `Frontmatter` (Name/Description/Metadata/HasFM); the
  field types and yaml tags decided here propagate unchanged.
- It is **the only place yaml.v3 is imported**, flipping it from `// indirect` to
  a direct dependency (verified_facts.md §14) — a small, expected, documented
  `go.mod` change.

---

## What

A `package discover` (in `internal/discover/`, so unimportable outside the module)
containing:

1. **`var utf8BOM = []byte{0xEF, 0xBB, 0xBF}`** — the UTF-8 BOM bytes, stripped
   before fence detection.
2. **`type Frontmatter struct`** — the SKILL.md frontmatter unmarshal target:
   - `Name string` (tag `name`), `Description string` (tag `description`) —
     required Agent Skills fields.
   - `License string` (`license,omitempty`), `Compatibility string`
     (`compatibility,omitempty`) — optional scalars.
   - `Metadata map[string]any` (`metadata,omitempty`) — the spec's arbitrary map;
     holds the skpp conventions `keywords`/`category`/`aliases` (extracted by S2).
   - `AllowedTools string` (`allowed-tools,omitempty`) — a SPACE-DELIMITED string
     per the Agent Skills spec (NOT a YAML list).
   - `DisableModelInvocation bool` (`disable-model-invocation,omitempty`) — a
     boolean flag.
   - `HasFM bool` (`yaml:"-"`) — NOT a frontmatter key; records whether a
     `---`-delimited block was found. Zero value `false`.
3. **`func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`**:
   - `os.ReadFile(path)`; on error return `(Frontmatter{}, "", err)`.
   - Strip ONE leading UTF-8 BOM via `bytes.TrimPrefix`.
   - Split on `"\n"`. If `lines[0]` (trailing `\r` trimmed) != `"---"` → no
     frontmatter: return `(Frontmatter{}, string(data), nil)`.
   - Find the next line (from index 1) that (trailing `\r` trimmed) == `"---"`.
     If none → lenient, no frontmatter: return `(Frontmatter{}, string(data), nil)`.
   - `yamlBlock = join(lines[1:closeIdx], "\n")`; `body = join(lines[closeIdx+1:], "\n")`.
   - `yaml.Unmarshal([]byte(yamlBlock), &f)`. On error → HARD error: return
     `(Frontmatter{}, body, uerr)`. On success → `f.HasFM = true`; return
     `(f, body, nil)`.

### Success Criteria

- [ ] `internal/discover/discover.go` is `package discover` with exactly the
      `Frontmatter` struct (8 fields incl. `HasFM bool \`yaml:"-"\``), the
      `utf8BOM` var, and `ParseFrontmatter` with the signature `(path string)
      (fm Frontmatter, body string, err error)`.
- [ ] Production imports are EXACTLY `bytes`, `os`, `strings`, `gopkg.in/yaml.v3`
      (no `fmt`/`io`/`path/filepath` — verified_facts.md §15).
- [ ] Full valid frontmatter parses: all scalar fields set, `Metadata` is a
      non-nil map, `keywords`/`aliases` arrive as `[]interface{}` (NOT `[]string`),
      `category` as `string`, `HasFM=true`, `body` = post-fence content.
- [ ] Unknown frontmatter keys are IGNORED (no error) — PRD §7.3 leniency.
- [ ] Folded scalar `>` KEEPS its trailing `\n`; returned verbatim (no trim).
- [ ] Quoted values are unquoted; spaces preserved.
- [ ] No `---` block at all → `HasFM=false`, `body` = whole file, `err=nil`.
- [ ] Opening `---` but no closing `---` → lenient, `HasFM=false`, no error.
- [ ] Empty file → no panic, `HasFM=false`, `body=""`.
- [ ] Leading UTF-8 BOM does NOT prevent detection (`HasFM=true`).
- [ ] CRLF endings: fences detected via `\r` trim; `body` retains its `\r`.
- [ ] Malformed YAML between valid fences → HARD error returned (`HasFM=false`).
- [ ] Missing file → `os.ReadFile` error returned verbatim.
- [ ] A frontmatter key literally named `hasfm` does NOT set `HasFM` (tag `-`).
- [ ] `gofmt -l` silent; `go vet` clean; `go build ./...` + `go test ./...` pass.
- [ ] `go mod tidy` changes ONLY the yaml.v3 `require` line (`// indirect` removed);
      `go.sum` unchanged; no other files touched.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `discover.go` and `discover_test.go` is given verbatim
in the Implementation Blueprint (copy-paste clean, gofmt-clean, compiles as-is —
the algorithm was compiled and run against 8 fixtures during research). Every
load-bearing behavior was empirically verified in the target go1.26.4 + yaml.v3
v3.0.1 environment (`research/verified_facts.md`): leniency vs. broken YAML (§1/§8),
`map[string]any`→`[]interface{}` (§2), folded-scalar trailing newline (§3), quoted
values (§4), BOM stripping (§5), CRLF handling (§6), dangling fence (§7), empty
file (§9), body semantics (§10), field names/tags (§11), the `HasFM` `yaml:"-"`
field (§12), the Skill-deferral scope decision (§13), and the `go.mod`
indirect→direct change (§14). The consumed yaml.v3 contract is the standard
package-level `yaml.Unmarshal`. An implementer who knows Go but nothing about this
repo can complete this in one pass from this document._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M2T4S1/research/verified_facts.md
  why: "Proves (against yaml.v3 v3.0.1 on go1.26.4, 8 fixtures): (1) lenient =
        ignore unknown KEYS, not tolerate broken YAML; do NOT call
        KnownFields(true). (2) Metadata lists arrive as []interface{} (never
        []string) — S2's job to assert, S1 just exposes the map. (3) folded '>'
        keeps a trailing \n (return verbatim, do not trim). (4) quoted unquoted.
        (5) UTF-8 BOM MUST be stripped before fence detection. (6) CRLF: trim \r
        for the fence compare only. (7) no closing fence -> lenient, no error.
        (8) malformed YAML between fences -> HARD error. (9) empty file -> no
        panic. (10) body = non-frontmatter portion always (incl. on the error
        path). (11) field names DisableModelInvocation / AllowedTools(string).
        (12) HasFM bool yaml:\"-\" MUST be on Frontmatter. (13) SCOPE: S1 owns
        Frontmatter+ParseFrontmatter ONLY; Skill type is S2. (14) go.mod: yaml.v3
        flips // indirect -> direct; run go mod tidy. (15) package + imports +
        white-box test convention."
  critical: "Do NOT define type Skill, toStringSlice, BuildSkill, or Index() —
             those are S2/T5. Declaring Skill here steals S2's deliverable. The
             task title's 'Skill types' = the frontmatter data model the Skill
             type is built FROM, not the Skill struct itself."

# PRIOR RESEARCH (same task, earlier numbering; superseded but cross-checkable)
- file: plan/001_fcde63e5bb60/P1M2T1S1/research/verified_facts.md
  why: "The 15-fact predecessor of the P1M2T4S1 research above. All facts hold and
        were re-confirmed by the P1M2T4S1 run. Read only if you want a second
        presentation of the same conclusions; P1M2T4S1/research is authoritative."

# CONTRACT — the discover package design (Frontmatter + ParseFrontmatter signature,
# Skill struct that S2 will build, the data flow discover->resolve->ui)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "Locks ParseFrontmatter(path) (fm Frontmatter, body string, err error) and
        the Skill struct (Dir/RelTag/Name/Description/Keywords/Category/Aliases/
        HasFM/SourceFile) that S2 owns. 'Frontmatter block extraction' note: read
        file; if it does not start with --- -> no frontmatter (HasFM=false); find
        next --- ; slice between = YAML; if no closing --- -> lenient; unmarshal
        with yaml.v3, unknown keys ignored (do NOT KnownFields(true)). 'metadata
        extraction' note: toStringSlice handles []any->[]string — that is S2, NOT
        this subtask."
  section: "Core types > internal/discover", "Frontmatter block extraction", "metadata extraction"

# CONTRACT — the yaml.v3 schema + Agent Skills frontmatter spec
- file: plan/001_fcde63e5bb60/architecture/external_deps.md
  why: "§1: the frontmatter field table (name/description required; license/
        compatibility/metadata/allowed-tools/disable-model-invocation optional;
        allowed-tools is SPACE-DELIMITED -> Go string, not []string). §3: the
        recommended Frontmatter struct shape (we use the full field name
        DisableModelInvocation per verified_facts §11, but identical yaml tags).
        Confirms yaml.v3 is the ONLY third-party dep."
  section: "1. Agent Skills specification", "3. Go third-party dependency"

# CONTRACT — the §7.3 parsing rules this implements
- file: PRD.md
  why: "§7.3: slice text between first two lines that are exactly ---; no block ->
        skill still resolves by directory (HasFM=false, no error); parse with
        yaml.v3; lenient on unknown keys; missing optional keys -> defaults.
        §7.1: the Skill fields Index() will populate FROM Frontmatter (name,
        description, metadata.keywords/category/aliases). §9: check rules that
        consume HasFM/name/description (ERROR: no name/description; WARN:
        description > 1024). READ-ONLY."
  critical: "§7.3 'lenient' = ignore unknown keys. It does NOT mean tolerate
             syntactically broken YAML — that is a hard error (verified_facts §8)."

# REFERENCE — the repo's test convention (white-box, same-package)
- file: internal/skillsdir/skillsdir_test.go
  why: "The established convention: `package skillsdir` (white-box), t.TempDir()/
        t.Setenv/t.Chdir, os.WriteFile for fixtures, plain t.Errorf/t.Fatalf (NO
        testify), NO t.Parallel() (repo convention is no-Parallel across the
        board). discover_test.go mirrors this as `package discover`."
  pattern: "White-box test file alongside the code; build fixtures in t.TempDir();
            assert with t.Errorf/t.Fatalf; no external assert libs."

# URLS — the load-bearing library + spec
- url: https://pkg.go.dev/gopkg.in/yaml.v3#Unmarshal
  why: "yaml.Unmarshal(data []byte, out interface{}) — the package-level lenient
        decoder used here. Unknown struct fields are ignored by default (this is
        the leniency we rely on); call dec.KnownFields(true) to make them
        hard-errors (we do NOT). Errors on syntactically broken YAML."
- url: https://pkg.go.dev/bytes#TrimPrefix
  why: "bytes.TrimPrefix(data, utf8BOM) strips one leading BOM; a no-op when
        absent, so it is safe for BOM-free files."
- url: https://agentskills.io/specification
  why: "The Agent Skills frontmatter spec (source of truth for field names/types:
        name 1-64 lowercase a-z0-9-, description max 1024, allowed-tools
        space-delimited, disable-model-invocation boolean, metadata arbitrary map)."
```

### Current Codebase tree (M1 landed; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/skillsdir/skillsdir.go        # M1.T2: Source + findEnv/findSibling/findWalkUp + Find + ErrNotFound
internal/skillsdir/skillsdir_test.go   # M1.T2 tests (white-box, package skillsdir)

$ ls -A
.git/  .gitignore  LICENSE  PRD.md  go.mod  go.sum  internal/  plan/  .pi-subagents/
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 // indirect
# go.sum: yaml.v3 v3.0.1 present
# .gitignore: ignores /skpp, /dist, *.test, *.out, .env*, .DS_Store, .pi-subagents/
# M1.T3 (main.go + main_test.go) is landed and committed. Irrelevant here: discover
# is a leaf library with no dep on main.go.
# NO internal/discover/, resolve/, ui/. NO skills/ (P1.M6.T12 ships skills/example).
```

### Desired Codebase tree with files to be added

```bash
skpp/
├── ... (go.mod [yaml.v3 // indirect removed by go mod tidy], go.sum, .gitignore,
│        LICENSE, PRD.md, internal/skillsdir/* — UNCHANGED except the go.mod line)
└── internal/
    └── discover/
        ├── discover.go       # CREATE — package discover: Frontmatter, utf8BOM, ParseFrontmatter
        └── discover_test.go  # CREATE — package discover (white-box): 12 fixture tests
```

| File (created) | Responsibility | Imports |
|---|---|---|
| `internal/discover/discover.go` | Parse SKILL.md frontmatter (BOM/CRLF-tolerant, lenient yaml.v3) into `Frontmatter`; return body | `bytes`, `os`, `strings`, `gopkg.in/yaml.v3` |
| `internal/discover/discover_test.go` | White-box unit tests for every §7.3 edge case | `os`, `path/filepath`, `strings`, `testing` |

**One new directory (`internal/discover/`). One expected `go.mod` edit (`// indirect`
removed from the yaml.v3 line by `go mod tidy`). `go.sum` unchanged. No `main.go`,
no `skillsdir`/`resolve`/`ui` touch, no `skills/`, no `Skill` struct.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — "Lenient" means ignore unknown KEYS, NOT tolerate broken YAML.
// yaml.v3's package-level yaml.Unmarshal ignores unknown struct fields by default
// (PRD §7.3 leniency). Do NOT call dec.KnownFields(true) — that hard-errors on
// unknown keys, the OPPOSITE of what we want. BUT: syntactically broken YAML
// between valid fences (e.g. "metadata: [unbalanced") IS a hard error and MUST be
// returned, so `check` (M4) can report it. Verified (research §1, §8).
//   RIGHT: err := yaml.Unmarshal(block, &fm)   // lenient on keys, strict on syntax
//   WRONG: dec := yaml.NewDecoder(r); dec.KnownFields(true)  // hard-errors unknown keys

// GOTCHA #2 — Metadata lists arrive as []interface{} (== []any), NEVER []string.
// yaml.v3 unmarshals YAML lists into []interface{}, regardless of element type.
// So fm.Metadata["keywords"] is []any{"writing","reddit"}, not []string. S2's
// toStringSlice asserts []any -> []string. S1 only EXPOSES the map; do not try to
// type Metadata's values as []string anywhere. Verified (research §2).

// GOTCHA #3 — Folded scalar '>' KEEPS a trailing "\n". Return it VERBATIM.
// "description: >" yields "One line. Two line.\n" — the \n before the next key is
// retained. Do NOT strings.TrimSpace it here (would also corrupt a "|" literal
// block). T10's 1024-char check trims if it wants the visible length. Verified §3.
//   RIGHT: return fm.Description as-is (with trailing \n)
//   WRONG: fm.Description = strings.TrimSpace(fm.Description)  // corrupts | blocks too

// GOTCHA #4 — UTF-8 BOM (0xEF 0xBB 0xBF) MUST be stripped BEFORE fence detection.
// A BOM-prefixed file makes lines[0] == "\ufeff---", which != "---", so frontmatter
// is silently MISSED. bytes.TrimPrefix(data, utf8BOM) strips one leading BOM and is
// a no-op when absent (safe for normal files). Do NOT pass the BOM to yaml.v3
// expecting it to cope — we slice the block ourselves, so the BOM is gone first.
// Verified §5.
//   RIGHT: data = bytes.TrimPrefix(data, []byte{0xEF,0xBB,0xBF})

// GOTCHA #5 — CRLF: trim trailing "\r" for the "---" comparison ONLY.
// Windows files have "---\r\n"; strings.Split on "\n" leaves "---\r". Compare with
// strings.TrimRight(line, "\r") == "---". Do NOT strip \r from the body slice —
// the body retains its original bytes (harmless; skpp does not consume body).
// Verified §6.
//   RIGHT: if strings.TrimRight(lines[0], "\r") != "---" { /* no frontmatter */ }

// GOTCHA #6 — No closing "---" fence is LENIENT (no error), NOT a hard error.
// An opening fence with content but no closing fence returns Frontmatter{HasFM:false},
// body = whole file, err = nil. This is distinct from GOTCHA #1 (broken YAML IS a
// hard error). Also covers the "---\n"-only file. Verified §7.

// GOTCHA #7 — empty file must NOT panic. strings.Split("", "\n") returns [""]
// (len 1); lines[0] == "" != "---" -> no frontmatter, body="". Verified §9.

// GOTCHA #8 — HasFM bool MUST be a field on Frontmatter with tag yaml:"-".
// The behavior "return Frontmatter{} with HasFM=false and no error" (PRD §7.3) is
// only expressible if HasFM is a struct field. The yaml:"-" tag stops yaml.v3 from
// reading/writing a frontmatter key named "hasfm" (verified: a "hasfm: true" key
// does NOT set the field). Zero value is false, so Frontmatter{} already means
// "no frontmatter". The parser sets HasFM=true only on the successful-fence path.
// This propagates into Skill.HasFM (S2). Verified §12.
//   RIGHT: HasFM bool `yaml:"-"`

// GOTCHA #9 — allowed-tools is a SPACE-DELIMITED STRING per the Agent Skills spec,
// not a YAML list. Type it `string` (tag allowed-tools). If a file wrongly uses a
// list, yaml.v3 errors (cannot unmarshal !!seq into string) — that is a spec
// violation and a hard error is acceptable (check flags it). skpp never reads this
// field anyway. disable-model-invocation is a `bool`. Verified §11.

// GOTCHA #10 — body = the non-frontmatter portion of the file, ALWAYS.
// Whole file when there is no frontmatter; everything after the closing "---" line
// otherwise — INCLUDING on the malformed-YAML error path (return post-fence body +
// the error). This is strictly more useful than "" and consistent with the
// "body = non-frontmatter" model. Verified §10 (the run shows body="body\n" on the
// malformed path). Tests assert err != nil && HasFM==false on that path, NOT the
// body value, so this is not brittle.

// GOTCHA #11 — go.mod WILL change: yaml.v3 flips from // indirect to direct.
// Once discover.go imports gopkg.in/yaml.v3, it is a DIRECT dependency. Builds pass
// even with the stale // indirect marker, but `go mod tidy` removes the comment.
// This is the EXPECTED, legitimate go.mod diff for THIS subtask (unlike M1.T3.S1
// which promised no go.mod change because main.go is pure stdlib). go.sum unchanged.
// Verified §14. Run `go mod tidy` as the final hygiene step.

// GOTCHA #12 — Do NOT define type Skill, toStringSlice, BuildSkill, or Index().
// Those are S2 (Skill + metadata extraction) and T5 (Index walk). Declaring Skill
// here steals S2's deliverable and creates churn. S1's deliverable is EXACTLY:
// Frontmatter, utf8BOM, ParseFrontmatter. Verified §13.

// GOTCHA #13 — Production imports are bytes, os, strings, gopkg.in/yaml.v3. NOT
// fmt, io, or path/filepath. ParseFrontmatter takes a path string and reads it; it
// does not format, does not use io abstractions, and does not walk or join paths.
// Adding a dead import fails `go vet`/build. Verified §15.

// GOTCHA #14 — discover_test.go MUST be `package discover` (WHITE-BOX, same
// package), mirroring internal/skillsdir/skillsdir_test.go (which is `package
// skillsdir`). White-box matches the repo convention and lets tests assert on any
// unexported helper if added later. NO testify; NO t.Parallel() (repo convention is
// no-Parallel across the board, even where — like here — env/cwd are untouched).
// Verified §15.
```

---

## Implementation Blueprint

### Data model — `Frontmatter` struct

No ORM/pydantic (this is Go). The only "model" is the frontmatter unmarshal target:

```go
// Frontmatter is the parsed SKILL.md frontmatter (PRD §7.3, Agent Skills spec).
// It is the unmarshal target for the YAML block between the "---" fences. Unknown
// keys are ignored by yaml.v3's default (lenient) decoder. The skpp conventions
// (keywords/category/aliases) live inside the standard, spec-compliant Metadata
// map; S2 extracts them into typed fields on Skill.
type Frontmatter struct {
	Name                   string         `yaml:"name"`
	Description            string         `yaml:"description"`
	License                string         `yaml:"license,omitempty"`
	Compatibility          string         `yaml:"compatibility,omitempty"`
	Metadata               map[string]any `yaml:"metadata,omitempty"`
	AllowedTools           string         `yaml:"allowed-tools,omitempty"`
	DisableModelInvocation bool           `yaml:"disable-model-invocation,omitempty"`
	HasFM                  bool           `yaml:"-"`
}
```

### File 1 — `internal/discover/discover.go` (CREATE)

Create the file with EXACTLY this content (gofmt-clean; the algorithm was compiled
and run against 8 fixtures during research — every edge case below is verified):

```go
// Package discover scans the on-disk skills/ tree and parses SKILL.md frontmatter.
//
// This file (P1.M2.T4.S1) implements ONLY the frontmatter data model and the
// ParseFrontmatter parser. The Index() walk (T5), the Skill struct + metadata
// extraction (S2), and the toStringSlice helper are LATER subtasks — do not add
// them here.
//
// ParseFrontmatter implements PRD §7.3: it extracts the YAML block between the
// first two lines that are exactly "---" (handling a leading UTF-8 BOM and CRLF
// line endings), then unmarshals it into Frontmatter with gopkg.in/yaml.v3.
// Parsing is LENIENT in the PRD §7.3 sense: unknown frontmatter keys are silently
// ignored (yaml.v3's default). It is NOT lenient about syntactically broken YAML
// between valid fences — that is returned as an error so `check` (M4) can report
// it. A SKILL.md with no frontmatter block returns Frontmatter{HasFM:false} and no
// error (the skill still resolves by directory in T5).
package discover

import (
	"bytes"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// utf8BOM is the 3-byte UTF-8 byte-order mark (U+FEFF). Some editors prepend it to
// SKILL.md; it must be stripped before fence detection, otherwise the opening
// "---" reads as "\ufeff---" and frontmatter is silently missed. bytes.TrimPrefix
// is a no-op when the BOM is absent, so this is safe for BOM-free files.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Frontmatter is the parsed SKILL.md frontmatter (PRD §7.3, Agent Skills spec).
//
// It is the unmarshal target for the YAML block between the "---" fences. Unknown
// keys are ignored by yaml.v3's default (lenient) decoder, matching pi's behavior.
// The skpp conventions (keywords/category/aliases) live inside the standard,
// spec-compliant Metadata map; S2 extracts them into typed fields on Skill.
//
// Field types follow the Agent Skills spec:
//   - name, description: required strings (empty here means "absent in source").
//   - license, compatibility: optional scalars.
//   - allowed-tools: a SPACE-DELIMITED string per the spec (NOT a YAML list).
//   - disable-model-invocation: a boolean flag.
//
// HasFM is NOT a frontmatter key; it records whether a "---"-delimited block was
// present at all. The yaml:"-" tag stops yaml.v3 from reading/writing a key named
// "hasfm". Its zero value is false, so Frontmatter{} already means "no frontmatter".
type Frontmatter struct {
	Name                   string         `yaml:"name"`
	Description            string         `yaml:"description"`
	License                string         `yaml:"license,omitempty"`
	Compatibility          string         `yaml:"compatibility,omitempty"`
	Metadata               map[string]any `yaml:"metadata,omitempty"`
	AllowedTools           string         `yaml:"allowed-tools,omitempty"`
	DisableModelInvocation bool           `yaml:"disable-model-invocation,omitempty"`
	HasFM                  bool           `yaml:"-"`
}

// ParseFrontmatter reads the SKILL.md at path and returns its parsed frontmatter
// plus the markdown body (the non-frontmatter text). It implements PRD §7.3.
//
// Behavior (verified against yaml.v3 v3.0.1 on go1.26.4):
//
//   - No "---" block (the file does not start with a "---" line): returns
//     Frontmatter{HasFM:false}, body = the WHOLE file content, nil error. A skill
//     with no frontmatter still resolves by directory later (T5); `check` (M4)
//     flags the missing block and --list shows description as "(missing)".
//   - Opening "---" present but no closing "---": treated as NO frontmatter
//     (lenient). Same return as above.
//   - Valid fences with syntactically broken YAML between them: the yaml.v3 error
//     is returned (fm.HasFM==false). "Lenient" means ignore unknown KEYS, NOT
//     tolerate corrupt YAML.
//   - A leading UTF-8 BOM is stripped before fence detection.
//   - CRLF line endings: a trailing "\r" is ignored for the "---" comparison only
//     (the body retains its original bytes).
//
// body is always the non-frontmatter portion of the file: the whole file when
// there is no frontmatter, or everything after the closing "---" line otherwise
// (including on the malformed-YAML error path). ParseFrontmatter returns values
// VERBATIM — it does not trim the folded-scalar trailing newline from description
// (T10's length check trims if it wants the visible length).
func ParseFrontmatter(path string) (fm Frontmatter, body string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, "", err
	}
	data = bytes.TrimPrefix(data, utf8BOM)

	lines := strings.Split(string(data), "\n")
	// Must start with a line that is exactly "---" (modulo a trailing CRLF \r).
	if strings.TrimRight(lines[0], "\r") != "---" {
		return Frontmatter{}, string(data), nil // no frontmatter block
	}

	// Find the next line that is exactly "---" (the closing fence).
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		// Opening fence but no closing fence: lenient -> treat as no frontmatter.
		return Frontmatter{}, string(data), nil
	}

	yamlBlock := strings.Join(lines[1:closeIdx], "\n")
	body = strings.Join(lines[closeIdx+1:], "\n")

	var f Frontmatter
	if uerr := yaml.Unmarshal([]byte(yamlBlock), &f); uerr != nil {
		// Syntactically broken YAML between valid fences is a HARD error. Return
		// the post-fence body (useful for `check` diagnostics) plus the error.
		return Frontmatter{}, body, uerr
	}
	f.HasFM = true
	return f, body, nil
}
```

### File 2 — `internal/discover/discover_test.go` (CREATE, `package discover` white-box)

Create the file with EXACTLY this content. It mirrors the repo's test convention
(white-box same-package, `t.TempDir()`/`os.WriteFile`, plain `t.Errorf`/`t.Fatalf`,
no testify, no `t.Parallel()`). Raw string literals hold multi-line fixtures;
double-quoted strings hold the BOM (`\ufeff`) and CRLF (`\r\n`) fixtures.

```go
package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSkill writes content to a temp SKILL.md and returns its path. Each fixture
// lives in its own t.TempDir() so they never collide.
func writeSkill(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// --- ParseFrontmatter: happy paths ---

func TestParseFrontmatterFull(t *testing.T) {
	path := writeSkill(t, `---
name: my-skill
description: A short description.
license: MIT
compatibility: "Requires Python 3.11+"
metadata:
  keywords: [writing, reddit]
  category: writing
  aliases:
    - reddit-post
    - social-post
allowed-tools: shell exec
disable-model-invocation: true
---
# Body
`)
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	if !fm.HasFM {
		t.Error("HasFM=false; want true")
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name=%q; want my-skill", fm.Name)
	}
	if fm.Description != "A short description." {
		t.Errorf("Description=%q; want 'A short description.'", fm.Description)
	}
	if fm.License != "MIT" {
		t.Errorf("License=%q; want MIT", fm.License)
	}
	if fm.Compatibility != "Requires Python 3.11+" {
		t.Errorf("Compatibility=%q; want 'Requires Python 3.11+'", fm.Compatibility)
	}
	if fm.AllowedTools != "shell exec" {
		t.Errorf("AllowedTools=%q; want 'shell exec' (space-delimited string)", fm.AllowedTools)
	}
	if !fm.DisableModelInvocation {
		t.Error("DisableModelInvocation=false; want true")
	}
	if fm.Metadata == nil {
		t.Fatal("Metadata=nil; want populated map")
	}
	if got := fm.Metadata["category"]; got != "writing" {
		t.Errorf("Metadata[category]=%#v; want \"writing\"", got)
	}
	// metadata lists arrive as []any (== []interface{}), NOT []string (yaml.v3).
	kw, ok := fm.Metadata["keywords"].([]any)
	if !ok {
		t.Fatalf("Metadata[keywords] type=%T; want []any", fm.Metadata["keywords"])
	}
	if len(kw) != 2 || kw[0] != "writing" || kw[1] != "reddit" {
		t.Errorf("keywords=%#v; want [writing reddit]", kw)
	}
	if body != "# Body\n" {
		t.Errorf("body=%q; want \"# Body\\n\"", body)
	}
}

// §1: unknown frontmatter keys are silently ignored (lenient, no error).
func TestParseFrontmatterUnknownKeysIgnored(t *testing.T) {
	path := writeSkill(t, `---
name: x
description: y
future-field: whatever
another: [1, 2, 3]
---
`)
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unknown keys: err=%v; want nil (lenient)", err)
	}
	if !fm.HasFM {
		t.Error("HasFM=false; want true")
	}
	if fm.Name != "x" {
		t.Errorf("Name=%q; want x", fm.Name)
	}
}

// §3: folded scalar '>' KEEPS a trailing newline. Returned verbatim (no trim).
func TestParseFrontmatterFoldedScalarKeepsTrailingNewline(t *testing.T) {
	path := writeSkill(t, `---
description: >
  One line.
  Two line.
name: x
---
`)
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.HasSuffix(fm.Description, "\n") {
		t.Errorf("folded Description=%q; want trailing \\n (returned verbatim)", fm.Description)
	}
	if !strings.Contains(fm.Description, "One line. Two line.") {
		t.Errorf("folded Description=%q; want lines joined with a single space", fm.Description)
	}
}

// §4: quoted values are unquoted; spaces preserved.
func TestParseFrontmatterQuotedValues(t *testing.T) {
	path := writeSkill(t, "---\nname: \"my-skill\"\ncompatibility: \"Requires Python 3.11+\"\ndescription: \"d\"\n---\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name=%q; want my-skill (unquoted)", fm.Name)
	}
	if fm.Compatibility != "Requires Python 3.11+" {
		t.Errorf("Compatibility=%q; want spaces preserved", fm.Compatibility)
	}
}

// --- ParseFrontmatter: no-frontmatter cases (lenient, no error) ---

// No frontmatter block at all: HasFM=false, body=whole file, no error.
func TestParseFrontmatterNoBlock(t *testing.T) {
	content := "# just markdown\nno frontmatter here\n"
	path := writeSkill(t, content)
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("no-block: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false")
	}
	if fm.Name != "" || fm.Description != "" {
		t.Errorf("no-block: Name=%q Description=%q; want empty", fm.Name, fm.Description)
	}
	if body != content {
		t.Errorf("body=%q; want whole file %q", body, content)
	}
}

// §7: opening fence but no closing fence -> lenient, no frontmatter, no error.
func TestParseFrontmatterNoClosingFence(t *testing.T) {
	content := "---\nname: dangling\ndescription: no close\n"
	path := writeSkill(t, content)
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("no-close: err=%v; want nil (lenient)", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false (no closing fence -> lenient)")
	}
	if body != content {
		t.Errorf("body=%q; want whole file %q", body, content)
	}
}

// §9: empty file -> no panic, no frontmatter, body="".
func TestParseFrontmatterEmptyFile(t *testing.T) {
	path := writeSkill(t, "")
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("empty: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false")
	}
	if body != "" {
		t.Errorf("empty body=%q; want \"\"", body)
	}
}

// "---\n"-only file: opening fence, immediate EOF -> lenient, no frontmatter.
func TestParseFrontmatterOnlyFence(t *testing.T) {
	path := writeSkill(t, "---\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("only-fence: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false (no closing fence -> lenient)")
	}
}

// --- ParseFrontmatter: encoding robustness ---

// §5: a leading UTF-8 BOM must not prevent fence detection.
func TestParseFrontmatterLeadingBOM(t *testing.T) {
	path := writeSkill(t, "\ufeff---\nname: bom-skill\ndescription: bommed\n---\nbody\n")
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("bom: err=%v", err)
	}
	if !fm.HasFM {
		t.Fatal("bom: HasFM=false; want true (BOM stripped before fence detection)")
	}
	if fm.Name != "bom-skill" {
		t.Errorf("bom Name=%q; want bom-skill", fm.Name)
	}
	if body != "body\n" {
		t.Errorf("bom body=%q; want \"body\\n\"", body)
	}
}

// §6: CRLF line endings -> fences detected via \r trim; body retains \r.
func TestParseFrontmatterCRLF(t *testing.T) {
	path := writeSkill(t, "---\r\nname: crlf-skill\r\ndescription: crlfd\r\n---\r\n# body\r\n")
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("crlf: err=%v", err)
	}
	if !fm.HasFM {
		t.Fatal("crlf: HasFM=false; want true (CRLF fences detected via \\r trim)")
	}
	if fm.Name != "crlf-skill" {
		t.Errorf("crlf Name=%q; want crlf-skill", fm.Name)
	}
	if body != "# body\r\n" {
		t.Errorf("crlf body=%q; want \"# body\\r\\n\" (\\r retained)", body)
	}
}

// --- ParseFrontmatter: error paths ---

// §8: malformed YAML between valid fences -> HARD error (propagated). Assert the
// error is present and HasFM is false; do NOT assert the yaml.v3 message wording
// (it is library-internal and may shift across versions).
func TestParseFrontmatterMalformedYAML(t *testing.T) {
	path := writeSkill(t, "---\nname: bad\nmetadata: [unbalanced\n---\nbody\n")
	fm, _, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("malformed: err=nil; want a yaml error (broken YAML is a HARD error)")
	}
	if fm.HasFM {
		t.Error("malformed: HasFM=true; want false (unmarshal failed)")
	}
}

// Read error: nonexistent file -> os.ReadFile error returned verbatim.
func TestParseFrontmatterMissingFile(t *testing.T) {
	fm, body, err := ParseFrontmatter(filepath.Join(t.TempDir(), "nope.md"))
	if err == nil {
		t.Fatal("missing file: err=nil; want os.ReadFile error")
	}
	if fm.HasFM || body != "" {
		t.Errorf("missing file: HasFM=%v body=%q; want false/empty", fm.HasFM, body)
	}
}

// --- Frontmatter type: the HasFM yaml:"-" guard ---

// A frontmatter key literally named "hasfm" must NOT set the HasFM field (the tag
// is "-", so yaml.v3 never touches it). Verified §12.
func TestHasFMNotMappedFromKey(t *testing.T) {
	path := writeSkill(t, "---\nname: x\ndescription: y\nhasfm: true\n---\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// HasFM is true here because a valid frontmatter block WAS found — but the
	// "hasfm: true" KEY did not influence it. Re-assert the contract: a file with
	// NO frontmatter block and a stray "hasfm" line must still be HasFM=false.
	if !fm.HasFM {
		t.Error("HasFM=false; want true (a valid block was present)")
	}

	nofm := writeSkill(t, "hasfm: true\nname: stray\n")
	fm2, _, err2 := ParseFrontmatter(nofm)
	if err2 != nil {
		t.Fatalf("nofm: err=%v", err2)
	}
	if fm2.HasFM {
		t.Error("nofm HasFM=true; want false (no --- block; the stray 'hasfm:' line must not set it)")
	}
}
```

> **Copy-paste correctness:** the two blueprint files above are gofmt-clean and
> compile as-is (imports limited to exactly what each file uses: `discover.go` →
> bytes/os/strings/yaml.v3; `discover_test.go` → os/path/filepath/strings/testing).
> Write them verbatim. The algorithm was compiled and run against 8 fixtures
> during research — every asserted value traces to the run output in
> `research/verified_facts.md`.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CREATE internal/discover/discover.go
  - WRITE: the exact content from the Blueprint (File 1).
  - CHECK: `package discover`; `var utf8BOM = []byte{0xEF,0xBB,0xBF}`;
           `type Frontmatter struct` with 8 fields (Name, Description, License,
           Compatibility, Metadata map[string]any, AllowedTools string,
           DisableModelInvocation bool, HasFM bool `yaml:"-"`);
           `func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`.
  - CHECK imports == bytes, os, strings, gopkg.in/yaml.v3 (no fmt/io/path/filepath).
  - GOTCHA: do NOT call KnownFields(true) (lenient on keys). Do NOT trim the
            folded description. Strip BOM before fence detection. CRLF \r trimmed
            for fence compare only. No closing fence -> lenient (no error). Broken
            YAML -> return the yaml error. Do NOT define Skill/toStringSlice/Index.

Task 2: CREATE internal/discover/discover_test.go
  - WRITE: the exact content from the Blueprint (File 2).
  - CHECK: `package discover` (white-box, NOT discover_test); the 12 tests cover
           full, unknown-keys, folded, quoted, no-block, no-closing-fence, empty,
           only-fence, BOM, CRLF, malformed, missing-file, hasfm-guard.
  - CHECK imports == os, path/filepath, strings, testing (no yaml, no bytes).
  - GOTCHA: NO testify; NO t.Parallel() (repo convention). Fixtures via t.TempDir()
            + os.WriteFile. Assert metadata lists as []any (NOT []string). Assert
            err != nil + HasFM==false on the malformed path (NOT the body value).

Task 3: FORMAT + VET + TIDY + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/discover/discover.go internal/discover/discover_test.go
  - COMMAND: gofmt -l internal/discover/*.go   # MUST print nothing
  - COMMAND: go vet ./internal/discover/       # MUST be clean
  - COMMAND: go mod tidy   # EXPECTED: removes "// indirect" from the yaml.v3 line
  - COMMAND: go build ./...                    # exit 0 (whole module compiles)
  - COMMAND: go test ./internal/discover/ -v   # ALL discover tests PASS
  - COMMAND: go test ./...                     # whole module green (skillsdir + discover)
  - EXPECT: zero errors, zero vet findings, gofmt silent, all tests pass.

Task 4: ACCEPTANCE SMOKE TEST (Level 3 in Validation Loop)
  - COMMAND: the Level 3 block below (inline-parse the PRD §11 example frontmatter
             and assert Name/Description/Keywords round-trip).
  - EXPECT: "PRD §11 EXAMPLE OK" printed.

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: discover.go has Frontmatter + ParseFrontmatter + utf8BOM only (no
            Skill/toStringSlice/Index); imports correct; go.mod change is only the
            // indirect removal; no main.go/skillsdir/resolve/ui touch; no skills/.
```

### Implementation Patterns & Key Details

```go
// PATTERN: the lenient unmarshal (package-level yaml.Unmarshal, NOT a Decoder).
//   var f Frontmatter
//   if err := yaml.Unmarshal([]byte(yamlBlock), &f); err != nil { return ..., err }
// WHY: the package-level Unmarshal ignores unknown struct fields by default (PRD
//      §7.3 leniency). A yaml.NewDecoder + KnownFields(true) would HARD-ERROR on
//      unknown keys — the opposite of what we want. Verified §1. (A Decoder is
//      only needed for streaming/KnownFields; we need neither.)

// PATTERN: BOM strip + fence scan on a line slice (CRLF-tolerant).
//   data = bytes.TrimPrefix(data, utf8BOM)
//   lines := strings.Split(string(data), "\n")
//   if strings.TrimRight(lines[0], "\r") != "---" { /* no frontmatter */ }
//   for i := 1; i < len(lines); i++ {
//       if strings.TrimRight(lines[i], "\r") == "---" { closeIdx = i; break }
//   }
// WHY: a leading BOM makes lines[0]=="\ufeff---" (missed without TrimPrefix); CRLF
//      files leave "---\r" (missed without TrimRight("\r")). The \r is trimmed for
//      the FENCE comparison only — the body slice keeps its original bytes.
//      Verified §5, §6.

// PATTERN: body = the non-frontmatter portion, returned on EVERY path.
//   no-frontmatter -> body = string(data)            // whole file
//   frontmatter    -> body = join(lines[closeIdx+1:], "\n")
//   unmarshal-err  -> return Frontmatter{}, body, err // post-fence body + error
// WHY: a frontmatter parser's "body" is the non-frontmatter content; if there is
//      no frontmatter the whole file is body (Hugo/Jekyll convention). Returning
//      post-fence body on the error path is consistent and useful for `check`
//      diagnostics. Verified §10.

// PATTERN: HasFM as a yaml:"-" struct field (not a separate return value).
//   type Frontmatter struct { ...; HasFM bool `yaml:"-"` }
//   // no-frontmatter path returns Frontmatter{} (HasFM==false by zero value)
//   // success path sets f.HasFM = true before returning
// WHY: the (Frontmatter, body, err) signature is locked by the architecture doc;
//      HasFM must ride on the struct. yaml:"-" keeps yaml.v3 off the field (a
//      "hasfm:" key does not set it). Propagated into Skill.HasFM by S2/T5. §12.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/discover/discover.go is `package discover` (internal -> unimportable
    outside the module; correct for a CLI's private packages).
  - imports: bytes, os, strings, gopkg.in/yaml.v3.
  - exposes: type Frontmatter, var utf8BOM, func ParseFrontmatter.
  - consumes: nothing from skpp (leaf library); only yaml.v3 + stdlib.

go.mod / go.sum (EXPECTED change — verified_facts.md §14):
  - before: require gopkg.in/yaml.v3 v3.0.1 // indirect
  - after:  require gopkg.in/yaml.v3 v3.0.1        (// indirect removed by go mod tidy)
  - go.sum: UNCHANGED (yaml.v3 v3.0.1 already checksummed).
  - This is the legitimate go.mod diff for THIS subtask. Run `go mod tidy`.

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into):
  - P1.M2.T4.S2 (Skill type + metadata extraction): defines `type Skill struct`
    (Dir/RelTag/Name/Description/Keywords/Category/Aliases/HasFM/SourceFile) and a
    `toStringSlice(v any) []string` helper that asserts []any -> []string. It
    reads Frontmatter.Metadata["keywords"/"category"/"aliases"] verbatim. S1 must
    NOT pre-empt these.
  - P1.M2.T5.S1 (Index walk): WalkDir(absSkillsDir); each dir containing SKILL.md
    -> ParseFrontmatter(dir+"/SKILL.md") -> build a Skill (via S2's constructor)
    with Dir/RelTag/SourceFile from the walk + frontmatter fields. relTag uses
    filepath.Rel + filepath.ToSlash (normalize OS sep -> '/').
  - P1.M2.T6.S1 (--list): reads Skill.Description; shows "(missing)" when HasFM==false
    or Description=="".
  - P1.M4.T10.S1 (check): ERROR if !HasFM (no block) OR Name=="" OR Description=="";
    WARN if len(TrimSpace(Description)) > 1024; ERROR on duplicate Name.

NO CHANGES TO:
  - go.sum (yaml.v3 already present)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned)
  - internal/skillsdir/* (M1-owned; not imported here)
  - main.go / main_test.go (M1.T3-owned; landed & committed — do NOT touch them here;
    discover is a leaf library with no dep on main.go regardless)
  - any other package or file (resolve/ui are later milestones; skills/ is P1.M6.T12)
```

---

## Validation Loop

### Level 1: Format, vet, tidy, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass)
gofmt -w internal/discover/discover.go internal/discover/discover_test.go
test -z "$(gofmt -l internal/discover/discover.go internal/discover/discover_test.go)" \
  || { echo "FAIL: gofmt found unformatted files"; gofmt -d internal/discover/; exit 1; }
echo "gofmt OK"

# Vet the new package
go vet ./internal/discover/ || { echo "FAIL: go vet ./internal/discover/"; exit 1; }
echo "go vet OK"

# Tidy: EXPECTED change — the "// indirect" marker on the yaml.v3 line disappears.
go mod tidy
echo "--- go.mod require block after tidy ---"
grep -n 'yaml.v3' go.mod

# Build the whole module (compile check across packages)
go build ./... || { echo "FAIL: go build ./..."; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run discover tests verbosely — all 12 fixtures
go test ./internal/discover/ -v || { echo "FAIL: go test ./internal/discover/ -v"; exit 1; }

# Targeted: the load-bearing leniency + encoding + error-path assertions
go test ./internal/discover/ -run \
  'TestParseFrontmatterFull|TestParseFrontmatterUnknownKeysIgnored|TestParseFrontmatterMalformedYAML|TestParseFrontmatterLeadingBOM|TestParseFrontmatterCRLF|TestParseFrontmatterNoClosingFence|TestHasFMNotMappedFromKey' -v \
  || { echo "FAIL: load-bearing discover tests"; exit 1; }

# Whole module still green (skillsdir + discover)
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Acceptance smoke test (parse the PRD §11 example frontmatter)

This proves ParseFrontmatter correctly parses a REAL Agent Skills frontmatter
shaped exactly like the one PRD §11 ships (P1.M6.T12 owns the on-disk file; here
we inline the same frontmatter so the gate runs without that dependency).

```bash
cd /home/dustin/projects/skpp

cat > /tmp/skpp_prd11_skill.md <<'MD'
---
name: example
description: >
  Reference example skill for skpp. Demonstrates the required frontmatter and
  how skpp resolves a tag to an absolute path. Safe to delete once you add real skills.
metadata:
  keywords: [example, demo, skpp]
  category: meta
---

# Example Skill
MD

# A tiny throwaway driver that calls the package's ParseFrontmatter. Built in a
# temp dir that reuses the repo module (so it sees internal/discover).
drv="$(mktemp -d)"
go build -o "$drv/probe" ./internal/discover 2>/dev/null # not a main package; ignore
# Instead: a one-file main that imports the internal package (must live under the module).
mkdir -p cmd/probe
cat > cmd/probe/main.go <<'GO'
package main

import (
	"fmt"
	"os"

	"github.com/dabstractor/skpp/internal/discover"
)

func main() {
	fm, body, err := discover.ParseFrontmatter(os.Args[1])
	if err != nil { fmt.Println("ERR:", err); os.Exit(1) }
	fmt.Printf("HasFM=%v Name=%q\n", fm.HasFM, fm.Name)
	fmt.Printf("Desc(starts)=%q\n", fm.Description[:40])
	fmt.Printf("keywords=%#v category=%#v\n", fm.Metadata["keywords"], fm.Metadata["category"])
	fmt.Printf("body starts=%q\n", body[:20])
}
GO
go run ./cmd/probe /tmp/skpp_prd11_skill.md
rc=$?
echo "probe rc=$rc"
# Assert the round-trip: HasFM true, Name "example", keywords present as a list.
out="$(go run ./cmd/probe /tmp/skpp_prd11_skill.md)"
echo "$out" | grep -q 'HasFM=true Name="example"' \
  || { echo "FAIL: PRD §11 example did not round-trip"; echo "$out"; rm -rf cmd /tmp/skpp_prd11_skill.md; exit 1; }
echo "$out" | grep -q 'keywords=\[\]' \
  && { echo "FAIL: keywords should be a non-empty list"; rm -rf cmd /tmp/skpp_prd11_skill.md; exit 1; } \
  || true
echo "$out" | grep -Eq 'keywords=\[\]interface \{\}.*"example".*"demo".*"skpp"' \
  || echo "$out" | grep -q '"example".*"demo".*"skpp"' \
  || { echo "FAIL: keywords missing expected entries"; echo "$out"; rm -rf cmd /tmp/skpp_prd11_skill.md; exit 1; }

# Cleanup the throwaway probe (it is NOT part of this subtask's deliverable).
rm -rf cmd /tmp/skpp_prd11_skill.md
echo "PRD §11 EXAMPLE OK"
echo "Level 3 PASS"
```

> If the `grep -Eq` on the `[]interface{}` rendering is brittle across Go/yaml
> versions, the fallback `grep -q '"example".*"demo".*"skpp"'` covers it. The
> contract under test is: HasFM=true, Name=="example", keywords is a list
> containing example/demo/skpp. The cleanup removes `cmd/` so the deliverable
> remains exactly `internal/discover/*`.

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# discover.go exists and is package discover
test -f internal/discover/discover.go || { echo "FAIL: discover.go missing"; exit 1; }
grep -q '^package discover' internal/discover/discover.go || { echo "FAIL: not package discover"; exit 1; }

# Frontmatter struct: the 8 fields with correct yaml tags
grep -q 'Name .*`yaml:"name"`' internal/discover/discover.go || { echo "FAIL: Name field/tag"; exit 1; }
grep -q 'Description .*`yaml:"description"`' internal/discover/discover.go || { echo "FAIL: Description tag"; exit 1; }
grep -q 'Metadata .*map\[string\]any.*`yaml:"metadata,omitempty"`' internal/discover/discover.go || { echo "FAIL: Metadata field"; exit 1; }
grep -q 'AllowedTools .*string.*`yaml:"allowed-tools,omitempty"`' internal/discover/discover.go || { echo "FAIL: AllowedTools must be string"; exit 1; }
grep -q 'DisableModelInvocation bool.*`yaml:"disable-model-invocation,omitempty"`' internal/discover/discover.go || { echo "FAIL: DisableModelInvocation tag"; exit 1; }
grep -q 'HasFM .*bool.*`yaml:"-"`' internal/discover/discover.go || { echo "FAIL: HasFM must be yaml:\"-\""; exit 1; }

# ParseFrontmatter signature + lenient unmarshal (package-level, no KnownFields)
grep -qE 'func ParseFrontmatter\(path string\) \(fm Frontmatter, body string, err error\)' internal/discover/discover.go \
  || { echo "FAIL: ParseFrontmatter signature"; exit 1; }
grep -q 'yaml.Unmarshal' internal/discover/discover.go || { echo "FAIL: must use yaml.Unmarshal"; exit 1; }
! grep -q 'KnownFields' internal/discover/discover.go || { echo "FAIL: must NOT call KnownFields(true) (lenient on keys)"; exit 1; }

# BOM strip + CRLF-tolerant fence compare present
grep -q 'bytes.TrimPrefix(data, utf8BOM)' internal/discover/discover.go || { echo "FAIL: BOM strip missing"; exit 1; }
grep -q 'TrimRight(lines\[0\], "\\r")' internal/discover/discover.go || { echo "FAIL: CRLF fence compare missing"; exit 1; }

# Imports are EXACTLY bytes, os, strings, gopkg.in/yaml.v3
grep -qA0 'import (' internal/discover/discover.go
imp="$(sed -n '/^import (/,/^)/p' internal/discover/discover.go)"
for want in bytes os strings gopkg.in/yaml.v3; do
  echo "$imp" | grep -q "\"$want\"" || { echo "FAIL: missing import $want"; exit 1; }
done
for ban in '"fmt"' '"io"' '"path/filepath"'; do
  echo "$imp" | grep -q "$ban" && { echo "FAIL: forbidden import $ban"; exit 1; } || true
done

# SCOPE: NO Skill type, NO toStringSlice, NO Index, NO BuildSkill in discover.go
! grep -qE 'type Skill struct' internal/discover/discover.go || { echo "FAIL: Skill type must NOT exist (S2 owns it)"; exit 1; }
! grep -q 'toStringSlice' internal/discover/discover.go || { echo "FAIL: toStringSlice must NOT exist (S2)"; exit 1; }
! grep -qE 'func Index' internal/discover/discover.go || { echo "FAIL: Index() must NOT exist (T5)"; exit 1; }

# discover_test.go is white-box package discover with the key tests
test -f internal/discover/discover_test.go || { echo "FAIL: discover_test.go missing"; exit 1; }
grep -q '^package discover' internal/discover/discover_test.go || { echo "FAIL: discover_test.go must be package discover (white-box)"; exit 1; }
! grep -q 't.Parallel' internal/discover/discover_test.go || { echo "FAIL: no t.Parallel() (repo convention)"; exit 1; }
for tn in TestParseFrontmatterFull TestParseFrontmatterUnknownKeysIgnored TestParseFrontmatterFoldedScalarKeepsTrailingNewline TestParseFrontmatterNoBlock TestParseFrontmatterNoClosingFence TestParseFrontmatterEmptyFile TestParseFrontmatterLeadingBOM TestParseFrontmatterCRLF TestParseFrontmatterMalformedYAML TestHasFMNotMappedFromKey; do
  grep -q "func $tn" internal/discover/discover_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# go.mod: the ONLY allowed change is the // indirect removal from yaml.v3.
# go.sum MUST be unchanged.
git diff --name-only | grep -q '^go.sum$' && { echo "FAIL: go.sum changed (must be unchanged)"; exit 1; } || true
if git diff --quiet go.mod 2>/dev/null; then
  echo "NOTE: go.mod unchanged (yaml.v3 still // indirect) — run 'go mod tidy' to flip it to direct."
else
  # go.mod changed: it must be ONLY the indirect->direct line, no added/removed modules.
  diff <(git show HEAD:go.mod) <(cat go.mod) | grep -E '^[+-]' | grep -v '^[+-][+-]' \
    | grep -vq 'yaml.v3 v3.0.1' \
    && { echo "FAIL: go.mod changed beyond the yaml.v3 indirect->direct line"; git diff go.mod; exit 1; } \
    || echo "go.mod change OK (yaml.v3 indirect->direct)"
fi

# MUST NOT have touched PRD.md / skillsdir / main.go / main_test.go (M1.T3-owned; do not modify)
git diff --quiet PRD.md 2>/dev/null   || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go 2>/dev/null      || { echo "FAIL: skillsdir.go changed"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir_test.go 2>/dev/null || { echo "FAIL: skillsdir_test.go changed"; exit 1; }

# MUST NOT have created later-milestone files/packages
test ! -d internal/resolve  || { echo "FAIL: resolve/ must not exist (M3)"; exit 1; }
test ! -d internal/ui       || { echo "FAIL: ui/ must not exist (M2.T6)"; exit 1; }
# main.go / main_test.go are M1.T3-owned (now landed). This subtask must not MODIFY them:
git diff --quiet main.go main_test.go 2>/dev/null || { echo "FAIL: main.go/main_test.go modified (M1.T3-owned)"; exit 1; }
test ! -f install.sh        || { echo "FAIL: install.sh must not exist (M6)"; exit 1; }
test ! -f README.md         || { echo "FAIL: README.md must not exist (M6)"; exit 1; }
test ! -d skills            || { echo "FAIL: skills/ must not exist (P1.M6.T12)"; exit 1; }
test ! -d cmd               || { echo "FAIL: cmd/ (probe) must be removed (throwaway)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l internal/discover/*.go` silent, `go vet ./internal/discover/` clean, `go mod tidy` flips yaml.v3 to direct, `go build ./...` exit 0
- [ ] Level 2 PASS — `go test ./internal/discover/ -v` all 12 tests pass; `go test ./...` whole module green
- [ ] Level 3 PASS — PRD §11 example frontmatter round-trips: HasFM=true, Name=="example", keywords is a list with example/demo/skpp
- [ ] Level 4 PASS — Frontmatter has all 8 fields with correct yaml tags; ParseFrontmatter signature correct; package-level yaml.Unmarshal (no KnownFields); BOM strip + CRLF compare present; imports exactly bytes/os/strings/yaml.v3; no Skill/toStringSlice/Index; go.sum unchanged; go.mod change is only the indirect→direct line; nothing else touched

### Feature Validation
- [ ] Full valid frontmatter parses: scalars set, Metadata non-nil, keywords/aliases as `[]any`, category as string, HasFM=true, body=post-fence content
- [ ] Unknown frontmatter keys ignored (no error) — PRD §7.3 leniency
- [ ] Folded scalar `>` keeps its trailing `\n`; returned verbatim (no trim)
- [ ] Quoted values unquoted; spaces preserved
- [ ] No `---` block → HasFM=false, body=whole file, err=nil
- [ ] No closing fence → lenient, HasFM=false, no error
- [ ] Empty file → no panic, HasFM=false, body=""
- [ ] Leading UTF-8 BOM does not prevent detection
- [ ] CRLF fences detected; body retains `\r`
- [ ] Malformed YAML between fences → hard error returned (HasFM=false)
- [ ] Missing file → os.ReadFile error returned verbatim
- [ ] A `hasfm:` key does not set the `HasFM` field

### Code Quality / Convention Validation
- [ ] `discover.go` is `package discover` in `internal/discover/`; imports limited to bytes/os/strings/yaml.v3
- [ ] `discover_test.go` is white-box `package discover`, mirroring `skillsdir_test.go`'s style (t.TempDir/os.WriteFile, plain t.Errorf/t.Fatalf, no testify, no t.Parallel)
- [ ] Every behavior is documented in godoc (the Frontmatter fields, ParseFrontmatter's behavior list, the utf8BOM var)

### Scope Discipline
- [ ] Did NOT define `type Skill struct` (S2 owns it — verified_facts §13)
- [ ] Did NOT add `toStringSlice` / `BuildSkill` / `Index()` (S2/T5)
- [ ] Did NOT modify `go.sum`, `PRD.md`, any `tasks.json`, `internal/skillsdir/*`, or
      `main.go`/`main_test.go` (M1.T3-owned); and did not create `resolve`/`ui`/
      `install.sh`/`README.md`/`skills/`/`cmd/`
- [ ] The `go.mod` change is ONLY the yaml.v3 `// indirect` → direct flip (run `go mod tidy`)

---

## Anti-Patterns to Avoid

- ❌ **Don't call `dec.KnownFields(true)`.** "Lenient" (PRD §7.3) means ignore
  unknown KEYS, which the package-level `yaml.Unmarshal` does by default.
  `KnownFields(true)` hard-errors on unknown keys — the opposite of lenient. (Verified §1.)
  Leniency does NOT extend to syntactically broken YAML: that is a hard error and
  MUST be returned so `check` (M4) can report it. (Verified §8.)
- ❌ **Don't forget the UTF-8 BOM strip.** A BOM makes `lines[0]=="\ufeff---"`,
  silently misclassifying a valid SKILL.md as no-frontmatter. `bytes.TrimPrefix(data,
  utf8BOM)` is a no-op when absent. (Verified §5.)
- ❌ **Don't forget CRLF.** Windows files leave `"---\r"`. Compare with
  `strings.TrimRight(line, "\r") == "---"`. Trim `\r` for the fence compare ONLY;
  the body keeps its original bytes. (Verified §6.)
- ❌ **Don't trim the folded-scalar description.** `description: >` yields a value
  ending in `\n`. Return it verbatim; T10's length check trims if it wants visible
  length. Trimming here would also corrupt a `|` literal block. (Verified §3.)
- ❌ **Don't treat a missing closing fence as a hard error.** An opening `---` with
  no closer is LENIENT (no frontmatter, no error). That is distinct from broken
  YAML, which IS a hard error. (Verified §7 vs §8.)
- ❌ **Don't panic on empty input.** `strings.Split("", "\n")` returns `[""]` (len 1);
  `lines[0]="" != "---"` → no frontmatter, body="". Guard nothing special — it just
  works. (Verified §9.)
- ❌ **Don't make `HasFM` a return value or a yaml-mapped field.** It must be a
  struct field with `yaml:"-"` so a `hasfm:` key cannot set it and the
  `(Frontmatter, body, err)` signature stays as the architecture locks it.
  (Verified §12.)
- ❌ **Don't type `Metadata` values as `[]string`.** yaml.v3 produces `[]interface{}`
  for lists. Expose the raw `map[string]any`; S2's `toStringSlice` asserts `[]any`
  → `[]string`. (Verified §2.)
- ❌ **Don't define `type Skill`, `toStringSlice`, `BuildSkill`, or `Index()`.**
  Those are S2 (Skill + metadata extraction) and T5 (Index walk). Declaring Skill
  here steals S2's deliverable. S1's deliverable is exactly Frontmatter + utf8BOM
  + ParseFrontmatter. (Verified §13.)
- ❌ **Don't add `fmt`, `io`, or `path/filepath` imports.** ParseFrontmatter takes
  a path string and reads it; it does not format, does not use io abstractions, and
  does not walk or join paths. Dead imports fail vet/build. (Verified §15.)
- ❌ **Don't add `t.Parallel()` or testify.** The repo convention (skillsdir_test.go)
  is white-box same-package, plain `t.Errorf`/`t.Fatalf`, no Parallel. Mirror it.
  (Verified §15.)
- ❌ **Don't be surprised by the `go.mod` change.** Importing yaml.v3 in non-test
  code flips it from `// indirect` to direct; `go mod tidy` removes the marker.
  This is the EXPECTED, legitimate diff for this subtask (go.sum unchanged).
  (Verified §14.)

---

## Confidence Score

**9/10** — one-pass implementation success likelihood.

Rationale: the exact `discover.go` and `discover_test.go` source is provided
verbatim and was already compiled + run against 8 fixtures in the target
environment (go1.26.4 + yaml.v3 v3.0.1) during research; every asserted value
traces to recorded output. The single `go.mod` change is documented and expected.
The only residual risk is the Level 3 probe's `grep -E` on the `[]interface{}`
rendering being brittle across Go/yaml versions — mitigated by the documented
fallback `grep`. The core parser itself is fully de-risked.
