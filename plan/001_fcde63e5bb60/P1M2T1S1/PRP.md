# PRP — P1.M2.T4.S1: `Frontmatter` type + `ParseFrontmatter` (yaml.v3, lenient)

> **Subtask:** P1.M2.T4.S1 (plan id `P1.M2.T1.S1`) — the FIRST subtask of T4
> (the `internal/discover` frontmatter model & parser, PRD §7.3).
> **Scope:** create `internal/discover/discover.go` (`package discover`) defining
> `type Frontmatter struct` (the YAML frontmatter model) and
> `func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`
> (lenient fence-slicing + yaml.v3 unmarshal), plus its white-box test
> `internal/discover/discover_test.go`. Nothing else.
>
> **DEPENDENCY (CONTRACT):** `go.mod` already pins `gopkg.in/yaml.v3 v3.0.1`
> (added indirect in P1.M1.T1.S1; the module is in the local cache — no network).
> This subtask is what makes that dep DIRECT (`go mod tidy` flips the
> `// indirect` marker). No other package is consumed: `discover` is a LEAF
> library. It does NOT depend on `skillsdir`, `main`, or anything else.
>
> **DOWNSTREAM CONSUMERS (do not build them, just keep the contract stable):**
> P1.M2.T4.S2 (metadata extraction + `Skill` type) reads `Frontmatter.Name`,
> `.Description`, `.Metadata` (a `map[string]any`), `.HasFM`. P1.M2.T5
> (`Index()` walk) calls `ParseFrontmatter` on each `SKILL.md`. The `Skill`
> struct itself is OWNED BY S2 — S1 does NOT define it (see §13 of verified_facts).
>
> **PARALLEL CONTEXT:** This subtask runs while P1.M1.T3.S1 (`main.go`) is being
> implemented. The two are fully independent: S1 creates `internal/discover/`,
> T3.S1 creates `main.go` at the repo root. Different packages, different dirs,
> no shared symbols. S1 does NOT touch `main.go`; T3.S1 does NOT touch `discover`.
> S1's validation is scoped to `./internal/discover/` so it never depends on
> `main.go` existing. `go test ./...` works whether or not main.go has landed.

---

## Goal

**Feature Goal**: Create the `internal/discover` package's frontmatter model and
parser — the component that turns a `SKILL.md` file into a typed `Frontmatter`
struct (plus the markdown `body`) using `gopkg.in/yaml.v3`, while being LENIENT:
unknown frontmatter keys are silently ignored (matches pi), a missing frontmatter
block is not an error (the skill still resolves by directory), a missing closing
`---` fence is treated as no-frontmatter, and only genuinely malformed YAML
between valid fences is surfaced as an error. BOM-prefixed and CRLF (Windows)
files are handled so a valid skill is never misclassified.

**Deliverable**: Two NEW files (no other files modified except the expected
`go.mod` `// indirect` cleanup):
1. `internal/discover/discover.go` — `package discover`; package doc comment;
   `var fence = "---"`; `var utf8BOM = []byte{0xEF, 0xBB, 0xBF}`;
   `type Frontmatter struct` (Name, Description, License, Compatibility,
   Metadata `map[string]any`, AllowedTools, DisableModelInvocation, HasFM);
   `func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`.
2. `internal/discover/discover_test.go` — `package discover` (white-box); tests
   for: full frontmatter; no frontmatter; unknown keys ignored; multiline folded
   scalar (`>`); quoted values; BOM-prefixed; CRLF; no closing fence (lenient);
   malformed YAML (error); empty file; only-open-fence.

**Success Definition**: `go build ./internal/discover/` exits 0;
`gofmt -l internal/discover/*.go` is silent; `go vet ./internal/discover/` is
clean; `go test ./internal/discover/ -v` passes (all cases above); `go test ./...`
whole module still green; `go.mod`'s yaml.v3 line loses its `// indirect` marker
after `go mod tidy` (the ONLY go.mod change). `skillsdir`/`main.go`/`PRD.md`
unchanged. No `Skill` struct, no `Index()`, no `toStringSlice` — those are S2/S5.

---

## Why

- This subtask is the **foundational parser** every discovery feature builds on.
  PRD §7.1 (discovery) captures `name`/`description`/`keywords`/`category`/
  `aliases` per skill — ALL of them come out of `ParseFrontmatter`. Until this
  lands, `Index()` (S5), `--list` (S6), `--search` (S9), tag resolution (S7), and
  `check` (S10) have nothing to read.
- It **locks the leniency contract** that matches pi's own behavior (PRD §7.3:
  "unknown frontmatter keys are ignored"). Getting this wrong — e.g. calling
  `KnownFields(true)` — would make skpp reject every real-world SKILL.md that
  carries an extra field, breaking the entire tool. Verified the default is
  lenient (research §1).
- It **establishes the `Frontmatter` struct shape** that S2's `Skill` builder and
  S5's `Index()` consume verbatim. Defining it now, with the exact field names and
  yaml tags the item specifies, means S2/S5 are pure consumers (no struct edits).
- It **resolves the edge cases that would otherwise cause silent misclassification**:
  UTF-8 BOM (a BOM-marked file's first line is `\ufeff---`, not `---`), CRLF
  endings (`---\r` ≠ `---`), and dangling opening fences. All three were
  reproduced and fixed in research (§5, §6, §7).

---

## What

A `package discover` (internal — unimportable outside the module) with:

1. **`type Frontmatter struct`** — the YAML frontmatter model. Fields (yaml tags
   in backticks): `Name` (`name`), `Description` (`description`), `License`
   (`license,omitempty`), `Compatibility` (`compatibility,omitempty`), `Metadata`
   (`metadata,omitempty`, type `map[string]any`), `AllowedTools`
   (`allowed-tools,omitempty`), `DisableModelInvocation`
   (`disable-model-invocation,omitempty`), `HasFM` (`yaml:"-"`). `HasFM` is NOT a
   YAML field — it is set by the parser to record whether a `---` block was found
   (S2 propagates it to `Skill.HasFM`).
2. **`func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)`**:
   - `os.ReadFile(path)`; on error return `(Frontmatter{}, "", err)`.
   - Strip ONE leading UTF-8 BOM (`bytes.TrimPrefix(data, utf8BOM)`).
   - Split into lines on `"\n"`; for each line, treat a trailing `"\r"` as
     stripped when comparing to the `---` fence (CRLF tolerance).
   - If the first line is NOT exactly `---` → **no frontmatter**: return
     `(Frontmatter{}, wholeFileContent, nil)` — `HasFM` stays false, NO error.
   - Else scan for the NEXT line that is exactly `---`. If none → **no frontmatter
     (lenient)**: same return as above (whole file as body, no error).
   - Else: the lines between the two fences are the YAML block; the lines after
     the closing fence are the body. `yaml.Unmarshal([]byte(yamlBlock), &fm)`;
     on error return `(Frontmatter{}, "", err)` (propagate the yaml error). On
     success set `fm.HasFM = true` and return `(fm, body, nil)`.

### Success Criteria

- [ ] `internal/discover/discover.go` exists, is `package discover`, imports only `bytes`, `os`, `strings`, `gopkg.in/yaml.v3`
- [ ] `type Frontmatter struct` has exactly the 8 fields above with the exact yaml tags (Name `name`, Description `description`, License `license,omitempty`, Compatibility `compatibility,omitempty`, Metadata `metadata,omitempty` of type `map[string]any`, AllowedTools `allowed-tools,omitempty`, DisableModelInvocation `disable-model-invocation,omitempty`, HasFM `yaml:"-"`)
- [ ] `func ParseFrontmatter(path string) (fm Frontmatter, body string, err error)` has that EXACT signature (matches go_architecture.md contract)
- [ ] The parser does NOT call `KnownFields(true)` — unknown keys are ignored (verified: an `unknown-key:` line produces no error and does not appear in the struct)
- [ ] A SKILL.md with full frontmatter parses all fields; `HasFM == true`; `body` is the text after the closing fence
- [ ] A SKILL.md with NO frontmatter returns `Frontmatter{}` (`HasFM == false`), the whole file as `body`, and `nil` error
- [ ] Unknown frontmatter keys are silently ignored (no error)
- [ ] A multiline folded-scalar `description: >` is parsed to a single space-joined string (yaml.v3 semantics; a trailing `\n` is retained — documented, not trimmed)
- [ ] Quoted values (`name: "x"`) are unquoted correctly
- [ ] A BOM-prefixed file is still detected as having frontmatter (BOM stripped before fence check)
- [ ] A CRLF (`\r\n`) file is still detected as having frontmatter (trailing `\r` tolerated on fence lines)
- [ ] A file with an opening `---` but NO closing `---` is treated as no-frontmatter (lenient, no error)
- [ ] A file with MALFORMED YAML between valid fences returns the yaml error (leniency is about unknown keys, not broken syntax)
- [ ] An empty file returns `(Frontmatter{}, "", nil)` — no panic
- [ ] `go build ./internal/discover/` exits 0; `gofmt -l internal/discover/*.go` silent; `go vet ./internal/discover/` clean; `go test ./internal/discover/ -v` passes; `go test ./...` green
- [ ] After `go mod tidy`, `go.mod`'s yaml.v3 line has NO `// indirect` (the ONLY go.mod change); `go.sum` unchanged
- [ ] `internal/skillsdir/*`, `main.go`, `PRD.md` unchanged; no `Skill` struct / `Index()` / `toStringSlice` / `resolve` / `ui` created

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `discover.go` is given verbatim in the Implementation
Blueprint (File 1), and the EXACT test file is given verbatim (File 2). Every
load-bearing behavior was empirically verified in the project's real Go 1.26.4
toolchain against the real `gopkg.in/yaml.v3 v3.0.1` (research/verified_facts.md
§1–§15): leniency-by-default (§1); `map[string]any` list/scalar typing (§2);
folded-scalar trailing-`\n` (§3); quoted values (§4); BOM stripping (§5); CRLF
tolerance (§6); no-closing-fence leniency (§7); malformed-YAML error (§8);
empty-file safety (§9); body-as-whole-file semantic (§10); the
`DisableModelInvocation` vs `DisableModelInv` naming decision (§11); the `HasFM`
field requirement (§12); the S1-vs-S2 scope split (§13); the go.mod
indirect→direct flip (§14); the package/import/test convention (§15). The
consumed `go.mod`/`go.sum` were read in full. An implementer who knows Go but
nothing about this repo can complete this in one pass from this document._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M2T1S1/research/verified_facts.md
  why: "Proves (against the REAL yaml.v3 v3.0.1 in Go 1.26.4): (§1) default
        yaml.Unmarshal is LENIENT — unknown keys ignored, do NOT call
        KnownFields(true). (§2) map[string]any receives flow/block lists as
        []interface{} and scalars as string — S2 must type-assert []any, not
        []string. (§3) folded scalar '>' joins lines with spaces AND keeps a
        trailing \\n — do NOT trim in the parser. (§4) quoted values unquoted.
        (§5) UTF-8 BOM (0xEF 0xBB 0xBF) MUST be bytes.TrimPrefix'd before fence
        detection or a BOM-marked file is misclassified as no-frontmatter.
        (§6) CRLF: TrimRight(line,'\\r') for the fence comparison only.
        (§7) no closing '---' => lenient no-frontmatter, no error. (§8) malformed
        YAML between valid fences => HARD error, propagate it. (§9) empty file =>
        no panic. (§10) body = whole file content when HasFM=false. (§11) field
        name is DisableModelInvocation (full), tag disable-model-invocation.
        (§12) HasFM bool yaml:'-' MUST be on the struct. (§13) S1 owns ONLY
        Frontmatter+ParseFrontmatter; S2 owns Skill. (§14) go mod tidy flips
        yaml.v3 from // indirect to direct (expected change). (§15) package
        discover, imports bytes/os/strings/yaml.v3, white-box test convention."
  critical: "Do NOT call KnownFields(true) — that makes unknown keys HARD-ERROR,
             the opposite of lenient. Do NOT trim the folded-scalar trailing \\n.
             Do NOT define Skill/toStringSlice/Index — S2/S5 own those."

# CONTRACT — the frontmatter schema + the recommended struct shape
- file: plan/001_fcde63e5bb60/architecture/external_deps.md
  why: "§1 frontmatter schema (name/description/license/compatibility/metadata/
        allowed-tools/disable-model-invocation constraints + leniency rules);
        §3 the recommended Go Frontmatter struct shape (yaml tags). The item's
        field list matches this; the ONLY addition is HasFM (yaml:'-')."
  critical: "§1 name regex (for check in M4, NOT this subtask): ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$
             length 1-64. description max 1024. This subtask PARSES only — it does
             NOT validate name/description constraints (that is M4 check, S10).
             ParseFrontmatter returns whatever yaml.v3 gives, even an over-long
             description or a bad name."

# CONTRACT — the package map, data flow, and the ParseFrontmatter signature
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "Locks: package map (internal/discover is a leaf library); the
        ParseFrontmatter(path string) (fm Frontmatter, body string, err error)
        signature; the Skill struct that S2 builds FROM Frontmatter (Dir, RelTag,
        Name, Description, Keywords, Category, Aliases, HasFM, SourceFile);
        'Frontmatter block extraction' note (BOM/newline trim, next '---' line,
        lenient on no closing fence, do NOT KnownFields(true)); 'metadata
        extraction' note (toStringSlice is S2's job, not S1's)."
  section: "Core types > internal/discover", "Key implementation notes > Frontmatter block extraction", "Key implementation notes > metadata extraction"

# CONTRACT — the PRD section this implements (§7.3 frontmatter parsing)
- file: PRD.md
  why: "§7.3: slice between the first two lines that are exactly '---' at the
        start of SKILL.md; no frontmatter => skill still resolves by dir, check
        flags it, --list shows description as '(missing)'; parse with yaml.v3
        (handles quoted/multiline scalars); lenient — unknown keys ignored;
        missing optional keys => defaults. §7.1: the per-skill fields discovery
        captures (name/description/keywords/category/aliases) — ALL come from
        ParseFrontmatter's output. §10: the example frontmatter shape. READ-ONLY."
  section: "7.3 Frontmatter parsing", "7.1 Discovery", "10. Skill directory & frontmatter conventions"

# REFERENCE — the test convention to follow (white-box, same-package)
- file: internal/skillsdir/skillsdir_test.go
  why: "The repo's established test convention: `package skillsdir` (white-box,
        same-package), t.TempDir()/os.WriteFile, plain t.Errorf/t.Fatalf (no
        testify), no t.Parallel(). discover_test.go mirrors this as
        `package discover`. ParseFrontmatter touches no env/cwd, so tests are
        fully hermetic with just t.TempDir."
  pattern: "White-box test file alongside the code; build fixtures with
            os.WriteFile(filepath.Join(t.TempDir(), 'SKILL.md'), []byte(content), 0644);
            assert on the returned struct fields + body + err (errors.Is for the
            malformed case, or strings.Contains on err.Error())."

# REFERENCE — the sibling landed package (for doc-comment + var-constant style)
- file: internal/skillsdir/skillsdir.go
  why: "Style to mirror: a package doc comment (// Package discover ...); named
        constants for magic strings (skillsdir uses `const envVar = ...`; we use
        `var fence = \"---\"` and `var utf8BOM = ...` — vars not consts because
        []byte can't be const); per-symbol doc comments explaining the contract."
  pattern: "Package doc comment at top; exported symbols have doc comments;
            unexported helpers documented; sentinel values as package-level vars."

# URLS — the two load-bearing mechanisms
- url: https://pkg.go.dev/gopkg.in/yaml.v3#Unmarshal
  why: "yaml.Unmarshal parses the YAML block into the Frontmatter struct. Default
        behavior is LENIENT (unknown fields ignored). To make it strict you'd use
        a Decoder + KnownFields(true) — we explicitly do NOT. Returns *yaml.TypeError
        or *yaml.SyntaxError on bad YAML (both satisfy the error interface; we
        just return err)."
  section: "func Unmarshal"
- url: https://pkg.go.dev/bytes#TrimPrefix
  why: "bytes.TrimPrefix(data, utf8BOM) strips a single leading UTF-8 BOM if
        present, and is a no-op if not — exactly the 'trim a leading BOM' step."
- url: https://pkg.go.dev/strings#Split
  why: "strings.Split(s, '\\n') carves the file into lines for fence detection;
        strings.TrimRight(line, '\\r') makes the fence comparison CRLF-tolerant."
```

### Current Codebase tree (M1 fully landed; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/skillsdir/skillsdir.go        # S1+S2+S3: Source + findEnv/findSibling/findWalkUp + Find + ErrNotFound
internal/skillsdir/skillsdir_test.go   # S1+S2+S3 tests (white-box, package skillsdir)
# main.go / main_test.go are being created by M1.T3.S1 IN PARALLEL — may or may not
#   be on disk yet. This subtask does NOT depend on them either way.

$ ls -A
.git/  .gitignore  LICENSE  PRD.md  go.mod  go.sum  internal/  plan/  .pi-subagents/
# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 // indirect
# go.sum: yaml.v3 v3.0.1 h1: + go.mod: lines already present (no network needed)
# module cache: /home/dustin/go/pkg/mod/gopkg.in/yaml.v3@v3.0.1 (present)
# NO internal/discover/ yet. NO resolve/ ui/ (later milestones).
# NO skills/ dir yet (P1.M6.T12 ships skills/example/SKILL.md).
```

### Desired Codebase tree with files to be added

```bash
skpp/
├── ... (go.mod, go.sum [// indirect removed after tidy], .gitignore, LICENSE, PRD.md — UNCHANGED except go.mod marker)
├── internal/
│   ├── skillsdir/                      # UNCHANGED (M1)
│   └── discover/
│       ├── discover.go                 # CREATE — package discover: Frontmatter + ParseFrontmatter
│       └── discover_test.go            # CREATE — package discover (white-box): parser tests
└── (main.go / main_test.go may appear via M1.T3.S1 — leave alone)
```

| File (created) | Responsibility | Consumes | Consumed by |
|---|---|---|---|
| `internal/discover/discover.go` | The frontmatter model + the lenient fence-slicing YAML parser | `gopkg.in/yaml.v3`, stdlib `bytes`/`os`/`strings` | S2 (`Skill` builder), S5 (`Index()`), M4 (`check`) |
| `internal/discover/discover_test.go` | White-box tests for all parser cases (full/no-fm/unknown/folded/quoted/BOM/CRLF/no-close/malformed/empty) | `discover.ParseFrontmatter`, `discover.Frontmatter` | — |

**No new packages beyond `internal/discover`. No `Skill`/`Index()`/`toStringSlice`
(S2/S5). No `resolve`/`ui`/`main.go` touch. The only non-create change is
`go mod tidy` removing `// indirect` from the yaml.v3 require line.**

### Known Gotchas of our codebase & the yaml.v3 library

```go
// GOTCHA #1 — Do NOT call KnownFields(true). yaml.v3 is LENIENT by default: the
// package-level yaml.Unmarshal ignores unknown struct fields. KnownFields(true)
// (a Decoder method) makes unknown keys HARD-ERROR — the exact opposite of
// PRD §7.3 ("unknown frontmatter keys are ignored"). VERIFIED (research §1): an
// `unknown-key: whatever` line produced no error and was silently dropped.
//   RIGHT: yaml.Unmarshal([]byte(block), &fm)        // lenient
//   WRONG: dec := yaml.NewDecoder(r); dec.KnownFields(true); dec.Decode(&fm)

// GOTCHA #2 — Lenient about UNKNOWN KEYS, NOT about malformed YAML. A missing
// CLOSING fence => no frontmatter, no error (lenient). But MALFORMED YAML
// BETWEEN valid fences (e.g. `metadata: [unbalanced`) => yaml.Unmarshal returns
// a real error (e.g. `yaml: line 1: did not find expected ',' or ']'`).
// ParseFrontmatter MUST propagate that error. These are two different cases;
// do not conflate them. VERIFIED §7 vs §8.

// GOTCHA #3 — A UTF-8 BOM (bytes 0xEF 0xBB 0xBF = U+FEFF) at the start of the
// file makes lines[0] == "\ufeff---", which != "---" => frontmatter NOT detected.
// Strip ONE leading BOM with bytes.TrimPrefix(data, utf8BOM) BEFORE splitting
// into lines. TrimPrefix is a no-op when there's no BOM (safe for normal files).
// We strip at most one BOM, only at the very start. VERIFIED §5.
//   var utf8BOM = []byte{0xEF, 0xBB, 0xBF}
//   data = bytes.TrimPrefix(data, utf8BOM)

// GOTCHA #4 — CRLF (\r\n) line endings: split on "\n" leaves a trailing "\r" on
// each line, so lines[i] == "---\r" != "---". Compare fences with
// strings.TrimRight(line, "\r") == fence. Strip the "\r" ONLY for the fence
// comparison — leave body bytes alone (a CRLF body retains its \r; harmless,
// skpp doesn't consume body). VERIFIED §6.

// GOTCHA #5 — Folded scalar `>` (and literal `|`) KEEP a trailing "\n" in the
// parsed value (yaml.v3 semantics). E.g. `description: >\n  a\n  b` parses to
// "a b\n" (note the trailing newline). ParseFrontmatter returns this VERBATIM —
// do NOT TrimSpace. Downstream (M4 check, 1024-char rule) trims if it wants the
// visible length. Trimming here would corrupt a `|` literal block. VERIFIED §3.

// GOTCHA #6 — yaml.v3 unmarshals YAML lists into []interface{} (== []any), NEVER
// []string. So Metadata["keywords"] is []any{"writing","reddit"}, not []string.
// S2's toStringSlice type-asserts []any -> []string. S1 only exposes the raw
// map[string]any; it does NOT pre-convert. Do not try to make Metadata hold
// []string directly — yaml.v3 won't populate it. VERIFIED §2.

// GOTCHA #7 — `HasFM bool` MUST be a field on Frontmatter with tag `yaml:"-"`
// (NOT a YAML key). The item's behavior says "return empty Frontmatter with
// HasFM=false" — only expressible if HasFM is a field. `yaml:"-"` ensures
// yaml.v3 never reads/writes a key named "hasfm". The zero value is false, so
// Frontmatter{} already has HasFM==false; the parser sets fm.HasFM=true only on
// the successful-fence path. S2 propagates Frontmatter.HasFM -> Skill.HasFM.
// VERIFIED §12.

// GOTCHA #8 — Field name is `DisableModelInvocation` (FULL name, matching the
// item description), tag `disable-model-invocation`. The architecture doc
// abbreviates it `DisableModelInv`; ignore that — the ITEM is authoritative.
// S2 does not read this field, so the name is low-risk, but use the full name
// for consistency with the item. VERIFIED §11 (both compile; tag is what matters).

// GOTCHA #9 — `body` when HasFM==false is the WHOLE file content (BOM-stripped),
// NOT "". Rationale: the non-frontmatter content is the body; with no frontmatter
// the whole file is body (Hugo/Jekyll convention; strictly more useful than "").
// When HasFM==true, body = text after the closing fence line. Skill (S2) does
// NOT store body, so this is a convenience for direct callers / future check.
// VERIFIED §10.

// GOTCHA #10 — strings.Split("", "\n") returns [""] (len 1), NOT []. So an empty
// file hits the no-frontmatter branch cleanly (lines[0]=="" != "---") and returns
// (Frontmatter{}, "", nil). No special-case needed; no panic. VERIFIED §9.

// GOTCHA #11 — Do NOT define `type Skill struct`, `toStringSlice`, `Index()`,
// or any resolver/ui code. S1 owns Frontmatter + ParseFrontmatter ONLY. S2 owns
// Skill + metadata extraction; S5 owns Index(). Defining Skill here collides
// with S2's contract. VERIFIED §13 (scope split).

// GOTCHA #12 — `go mod tidy` will remove the `// indirect` comment from the
// yaml.v3 require line in go.mod (this subtask makes yaml.v3 a DIRECT dep, since
// discover.go imports it). This is the ONLY expected go.mod change; go.sum is
// unchanged. `go build`/`vet`/`test` pass even WITHOUT tidying (the marker is
// informational), but run `go mod tidy` for hygiene. VERIFIED §14.

// GOTCHA #13 — Scope validation to ./internal/discover/ so this subtask never
// depends on main.go (M1.T3.S1, parallel). `go test ./internal/discover/` and
// `go build ./internal/discover/` build the leaf library alone. `go test ./...`
// is a bonus whole-module check that works whether or not main.go has landed
// (if main.go is absent, the repo root simply has no package to build there;
// internal/* packages still build/test independently).
```

---

## Implementation Blueprint

### Data model — the Frontmatter struct

No ORM/pydantic (this is Go). The single "model" is the Frontmatter struct. It
is the faithful representation of the SKILL.md frontmatter schema (PRD §3/§10)
PLUS the non-YAML `HasFM` flag the parser sets:

```go
// Frontmatter is the typed view of a SKILL.md's YAML frontmatter. Unknown YAML
// keys are ignored (lenient, matches pi). Missing optional keys => zero values.
// HasFM is NOT a YAML field (yaml:"-"); the parser sets it to record whether a
// `---` block was found. S2's Skill builder propagates HasFM -> Skill.HasFM.
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

Create the file with EXACTLY this content (gofmt-clean; comments explain every
non-obvious decision and cite the research section that verified it):

```go
// Package discover walks the skills/ tree and parses SKILL.md frontmatter.
//
// This file (P1.M2.T4.S1) implements the frontmatter MODEL and the lenient
// fence-slicing PARSER only. The directory-walking Index() lands in P1.M2.T5;
// the Skill struct + metadata extraction land in P1.M2.T4.S2.
//
// Leniency (PRD §7.3, matches pi): unknown frontmatter keys are silently
// ignored (yaml.v3 default — do NOT call KnownFields(true)); a missing
// frontmatter block, a missing closing `---` fence, or an empty file are all
// treated as "no frontmatter" (HasFM=false) and return NO error — the skill
// still resolves by directory elsewhere. Only genuinely malformed YAML between
// valid fences is surfaced as an error.
package discover

import (
	"bytes"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// fence is the exact line that opens and closes a frontmatter block. A line
// counts as a fence iff, after stripping a trailing CR (CRLF tolerance), it
// equals this string. (PRD §7.3: "lines that are exactly `---`".)
var fence = "---"

// utf8BOM is the UTF-8 byte-order mark (U+FEFF). A SKILL.md saved by some
// editors begins with these 3 bytes; left in place they make the first line
// "\ufeff---" != "---" and the frontmatter is silently missed. ParseFrontmatter
// strips at most one leading BOM before fence detection. (research §5)
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Frontmatter is the typed view of a SKILL.md's YAML frontmatter (PRD §3/§10).
//
// Field/tag mapping (tag in backticks is what yaml.v3 reads):
//
//	Name                   <- name
//	Description            <- description
//	License                <- license                  (optional)
//	Compatibility          <- compatibility            (optional)
//	Metadata               <- metadata                 (optional, arbitrary map;
//	                                                   skpp conventions keywords/
//	                                                   category/aliases live here)
//	AllowedTools           <- allowed-tools            (optional)
//	DisableModelInvocation <- disable-model-invocation (optional)
//
// Unknown keys are ignored (lenient). Missing optional keys => Go zero values.
//
// HasFM is NOT a YAML field (yaml:"-"). ParseFrontmatter sets it to true iff a
// valid `--- ... ---` block was found, so downstream code (S2's Skill builder,
// --list, check) can distinguish "no frontmatter" from "frontmatter with empty
// fields". S2 propagates HasFM -> Skill.HasFM.
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

// ParseFrontmatter reads path (a SKILL.md), extracts the YAML frontmatter block
// between the first two lines that are exactly `---`, unmarshals it into a
// Frontmatter with gopkg.in/yaml.v3, and returns the markdown body that follows
// the closing fence.
//
// Algorithm (PRD §7.3 + architecture "Frontmatter block extraction"):
//
//  1. os.ReadFile(path). On error, return (Frontmatter{}, "", err).
//  2. Strip ONE leading UTF-8 BOM if present (bytes.TrimPrefix). (research §5)
//  3. Split into lines on "\n". For fence comparison only, a trailing "\r" is
//     tolerated (CRLF files): strings.TrimRight(line, "\r") == fence. (research §6)
//  4. If the first line is NOT a fence => NO frontmatter: return
//     (Frontmatter{}, wholeFileContent, nil). HasFM stays false. No error.
//     (A missing-frontmatter skill still resolves by directory; check flags it.)
//  5. Scan for the NEXT fence line. If none => NO frontmatter (lenient): same
//     return as step 4 (whole file as body, no error). (research §7)
//  6. The lines between the two fences are the YAML block; the lines after the
//     closing fence are the body. yaml.Unmarshal(block, &fm). On error return
//     (Frontmatter{}, "", err) — malformed YAML is a real error (NOT lenient).
//     (research §8)
//  7. Set fm.HasFM = true; return (fm, body, nil).
//
// body semantic: when HasFM==true it is the text after the closing fence; when
// HasFM==false it is the whole file content (BOM-stripped). (research §10)
//
// Leniency is about UNKNOWN KEYS, not broken syntax. yaml.v3 is lenient by
// default; do NOT call KnownFields(true) (that would make unknown keys error).
// (research §1)
func ParseFrontmatter(path string) (fm Frontmatter, body string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, "", err
	}
	// Strip a single leading UTF-8 BOM so a BOM-marked file still detects its
	// frontmatter fence. No-op when there is no BOM. (research §5)
	data = bytes.TrimPrefix(data, utf8BOM)

	s := string(data)
	lines := strings.Split(s, "\n")

	// isFence reports whether line is exactly "---", tolerating a trailing CR
	// from CRLF files. (research §6)
	isFence := func(line string) bool { return strings.TrimRight(line, "\r") == fence }

	// No lines at all, or the first line is not a fence => no frontmatter.
	// (strings.Split("", "\n") returns [""], so an empty file lands here too —
	// research §9. Return the whole content as body, no error.)
	if len(lines) == 0 || !isFence(lines[0]) {
		return Frontmatter{}, s, nil
	}

	// Find the next fence line (the closing delimiter). The opening fence is
	// lines[0]; scan from index 1.
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if isFence(lines[i]) {
			closeIdx = i
			break
		}
	}
	// No closing fence => treat as no frontmatter (lenient). (research §7)
	if closeIdx == -1 {
		return Frontmatter{}, s, nil
	}

	// YAML block = lines strictly between the fences; body = lines after the
	// closing fence. Rejoin with "\n" (we split on "\n", so this round-trips
	// except for CRLF bodies, which keep their "\r" — harmless; research §6).
	yamlBlock := strings.Join(lines[1:closeIdx], "\n")
	body = strings.Join(lines[closeIdx+1:], "\n")

	// Lenient unmarshal: unknown keys ignored by default (do NOT KnownFields(true),
	// research §1). Malformed YAML => real error (research §8).
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return Frontmatter{}, "", err
	}

	fm.HasFM = true
	return fm, body, nil
}
```

### File 2 — `internal/discover/discover_test.go` (CREATE, `package discover` white-box)

Create the file with EXACTLY this content. It mirrors the repo's test convention
(white-box same-package, `t.TempDir()`/`os.WriteFile`, plain `t.Errorf`/`t.Fatalf`,
no testify, no `t.Parallel()`):

```go
package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSkill writes content to a fresh temp SKILL.md and returns its path.
// ParseFrontmatter touches no env/cwd, so these tests are fully hermetic.
func writeSkill(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return p
}

// --- full frontmatter: all fields parsed, HasFM=true, body after fence ---

func TestParseFullFrontmatter(t *testing.T) {
	content := "---\n" +
		"name: my-skill\n" +
		"description: A skill that does things.\n" +
		"license: MIT\n" +
		"compatibility: Requires Python 3.11+\n" +
		"allowed-tools: read write\n" +
		"disable-model-invocation: true\n" +
		"metadata:\n" +
		"  category: writing\n" +
		"---\n" +
		"# Body\n" +
		"Instructions.\n"
	fm, body, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter full: err=%v; want nil", err)
	}
	if !fm.HasFM {
		t.Errorf("HasFM=false; want true")
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name=%q; want %q", fm.Name, "my-skill")
	}
	if fm.Description != "A skill that does things." {
		t.Errorf("Description=%q; want the plain sentence", fm.Description)
	}
	if fm.License != "MIT" {
		t.Errorf("License=%q; want MIT", fm.License)
	}
	if fm.Compatibility != "Requires Python 3.11+" {
		t.Errorf("Compatibility=%q", fm.Compatibility)
	}
	if fm.AllowedTools != "read write" {
		t.Errorf("AllowedTools=%q; want %q", fm.AllowedTools, "read write")
	}
	if !fm.DisableModelInvocation {
		t.Errorf("DisableModelInvocation=false; want true")
	}
	if got, _ := fm.Metadata["category"].(string); got != "writing" {
		t.Errorf("metadata.category=%q; want %q", got, "writing")
	}
	wantBody := "# Body\nInstructions.\n"
	if body != wantBody {
		t.Errorf("body=%q; want %q", body, wantBody)
	}
}

// --- no frontmatter: HasFM=false, whole file as body, no error ---

func TestParseNoFrontmatter(t *testing.T) {
	content := "# Just a body\nno fences at all\n"
	fm, body, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter no-fm: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Errorf("HasFM=true; want false")
	}
	if fm.Name != "" || fm.Description != "" || fm.Metadata != nil {
		t.Errorf("no-fm returned non-zero fields: %+v", fm)
	}
	if body != content {
		t.Errorf("body=%q; want the whole file content %q", body, content)
	}
}

// --- unknown keys are IGNORED (lenient — do NOT KnownFields(true)) ---

func TestParseUnknownKeysIgnored(t *testing.T) {
	content := "---\n" +
		"name: foo\n" +
		"description: hi\n" +
		"some-unknown-key: whatever\n" +
		"another: [1, 2, 3]\n" +
		"---\n# body\n"
	fm, _, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter unknown-keys: err=%v; want nil (lenient)", err)
	}
	if !fm.HasFM {
		t.Errorf("HasFM=false; want true")
	}
	if fm.Name != "foo" {
		t.Errorf("Name=%q; want foo (known fields still parsed)", fm.Name)
	}
}

// --- multiline folded scalar description (yaml '>') ---

func TestParseFoldedScalarDescription(t *testing.T) {
	content := "---\n" +
		"name: foo\n" +
		"description: >\n" +
		"  One to two sentences: what this skill does\n" +
		"  and precisely when to use it.\n" +
		"---\n# body\n"
	fm, _, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter folded: err=%v; want nil", err)
	}
	// yaml.v3 folds consecutive indented lines with a single space AND keeps a
	// trailing newline. ParseFrontmatter returns it verbatim (no trim).
	wantDesc := "One to two sentences: what this skill does and precisely when to use it.\n"
	if fm.Description != wantDesc {
		t.Errorf("Description=%q; want %q (folded, trailing newline retained)", fm.Description, wantDesc)
	}
}

// --- quoted values are unquoted ---

func TestParseQuotedValues(t *testing.T) {
	content := "---\n" +
		"name: \"my-skill\"\n" +
		"description: 'It has spaces and a \" quote'\n" +
		"---\n# body\n"
	fm, _, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter quoted: err=%v; want nil", err)
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name=%q; want my-skill (double-quotes stripped)", fm.Name)
	}
	if fm.Description != `It has spaces and a " quote` {
		t.Errorf("Description=%q; want the single-quoted value unquoted", fm.Description)
	}
}

// --- metadata: flow list, block list, scalar (yaml.v3 -> []any / string) ---

func TestParseMetadataShapes(t *testing.T) {
	content := "---\n" +
		"name: foo\n" +
		"description: hi\n" +
		"metadata:\n" +
		"  keywords: [writing, reddit]\n" +
		"  category: writing\n" +
		"  aliases:\n" +
		"    - reddit-post\n" +
		"    - social-post\n" +
		"---\n# body\n"
	fm, _, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter metadata: err=%v; want nil", err)
	}
	if fm.Metadata == nil {
		t.Fatalf("Metadata=nil; want populated map")
	}
	// yaml.v3 gives []any for lists (NOT []string) — S2 type-asserts. Here we
	// only confirm the values arrive at all.
	if kw, ok := fm.Metadata["keywords"].([]any); !ok || len(kw) != 2 {
		t.Errorf("metadata.keywords=%#v; want []any of len 2", fm.Metadata["keywords"])
	}
	if cat, _ := fm.Metadata["category"].(string); cat != "writing" {
		t.Errorf("metadata.category=%#v; want string \"writing\"", fm.Metadata["category"])
	}
	if al, ok := fm.Metadata["aliases"].([]any); !ok || len(al) != 2 {
		t.Errorf("metadata.aliases=%#v; want []any of len 2", fm.Metadata["aliases"])
	}
}

// --- BOM-prefixed file still detects frontmatter ---

func TestParseBOMPrefixed(t *testing.T) {
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("---\nname: bom-skill\ndescription: hi\n---\n# body\n")...)
	p := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	fm, body, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("ParseFrontmatter BOM: err=%v; want nil", err)
	}
	if !fm.HasFM {
		t.Fatalf("HasFM=false; want true (BOM must be stripped before fence check)")
	}
	if fm.Name != "bom-skill" {
		t.Errorf("Name=%q; want bom-skill", fm.Name)
	}
	if body != "# body\n" {
		t.Errorf("body=%q; want \"# body\\n\"", body)
	}
}

// --- CRLF (Windows) line endings: fence comparison tolerates trailing \r ---

func TestParseCRLF(t *testing.T) {
	content := "---\r\nname: crlf-skill\r\ndescription: hi\r\n---\r\n# body\r\n"
	fm, _, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter CRLF: err=%v; want nil", err)
	}
	if !fm.HasFM {
		t.Fatalf("HasFM=false; want true (CRLF fence lines must be tolerated)")
	}
	if fm.Name != "crlf-skill" {
		t.Errorf("Name=%q; want crlf-skill", fm.Name)
	}
}

// --- opening fence but NO closing fence => lenient no-frontmatter, no error ---

func TestParseNoClosingFenceLenient(t *testing.T) {
	content := "---\nname: dangling\ndescription: no close\n"
	fm, body, err := ParseFrontmatter(writeSkill(t, content))
	if err != nil {
		t.Fatalf("ParseFrontmatter no-close: err=%v; want nil (lenient)", err)
	}
	if fm.HasFM {
		t.Errorf("HasFM=true; want false (no closing fence => no frontmatter)")
	}
	// Whole file content returned as body.
	if body != content {
		t.Errorf("body=%q; want whole file content", body)
	}
}

// --- opening fence then immediate EOF (only "---\n") => no frontmatter ---

func TestParseOnlyOpeningFence(t *testing.T) {
	fm, _, err := ParseFrontmatter(writeSkill(t, "---\n"))
	if err != nil {
		t.Fatalf("ParseFrontmatter only-open: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Errorf("HasFM=true; want false (no closing fence)")
	}
}

// --- malformed YAML between valid fences => HARD error propagated ---

func TestParseMalformedYAMLErrors(t *testing.T) {
	content := "---\nname: bad\nmetadata: [unbalanced\n---\n# body\n"
	fm, body, err := ParseFrontmatter(writeSkill(t, content))
	if err == nil {
		t.Fatalf("ParseFrontmatter malformed: err=nil; want a yaml error")
	}
	// Lenient about unknown KEYS, NOT about broken syntax.
	if !strings.Contains(err.Error(), "yaml") && !strings.Contains(err.Error(), "did not find") {
		t.Errorf("err=%q; want a yaml parse error", err.Error())
	}
	// On error, return zero Frontmatter and empty body (do NOT half-fill).
	if fm.HasFM || fm.Name != "" {
		t.Errorf("malformed returned non-zero fm: %+v; want Frontmatter{}", fm)
	}
	if body != "" {
		t.Errorf("malformed body=%q; want empty", body)
	}
}

// --- read error: nonexistent file => os.ReadFile error propagated ---

func TestParseNonexistentFile(t *testing.T) {
	fm, body, err := ParseFrontmatter(filepath.Join(t.TempDir(), "nope.md"))
	if err == nil {
		t.Fatalf("ParseFrontmatter missing file: err=nil; want an os error")
	}
	if !os.IsNotExist(err) {
		t.Errorf("err=%v; want an os.IsNotExist error", err)
	}
	if fm.HasFM || body != "" {
		t.Errorf("missing-file returned non-zero fm/body: %+v %q", fm, body)
	}
}

// --- empty file: no panic, no frontmatter, empty body ---

func TestParseEmptyFile(t *testing.T) {
	fm, body, err := ParseFrontmatter(writeSkill(t, ""))
	if err != nil {
		t.Fatalf("ParseFrontmatter empty: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Errorf("HasFM=true; want false")
	}
	if body != "" {
		t.Errorf("body=%q; want empty", body)
	}
}
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: PRECONDITION — confirm go.mod has yaml.v3 and the module cache has it
  - COMMAND: cd /home/dustin/projects/skpp
  - COMMAND: grep -q 'gopkg.in/yaml.v3 v3.0.1' go.mod
  - COMMAND: test -d "$(go env GOMODCACHE)/gopkg.in/yaml.v3@v3.0.1" && echo "yaml.v3 cached"
  - EXPECT: yaml.v3 is pinned in go.mod AND present in the module cache (no network).
            If the cache is missing, `go mod download` fetches it (go.sum already has it).

Task 1: CREATE internal/discover/discover.go
  - WRITE: the exact content from the Blueprint (File 1) to ./internal/discover/discover.go.
  - CHECK: `package discover`; imports = bytes, os, strings, gopkg.in/yaml.v3 (ONLY these 4);
           var fence = "---"; var utf8BOM = []byte{0xEF,0xBB,0xBF};
           type Frontmatter struct with the 8 fields and exact yaml tags;
           func ParseFrontmatter(path string) (fm Frontmatter, body string, err error).
  - GOTCHA: do NOT call KnownFields(true). Do NOT trim the folded-scalar description.
            Do NOT define Skill/toStringSlice/Index. HasFM has yaml:"-".

Task 2: CREATE internal/discover/discover_test.go
  - WRITE: the exact content from the Blueprint (File 2) to ./internal/discover/discover_test.go.
  - CHECK: `package discover` (white-box, NOT discover_test); tests cover: full
           frontmatter (all fields + body), no frontmatter (whole-file body),
           unknown keys ignored, folded scalar '>', quoted values, metadata shapes
           (flow/block list + scalar), BOM-prefixed, CRLF, no closing fence
           (lenient), only-open-fence, malformed YAML (error), nonexistent file
           (os error), empty file (no panic).
  - GOTCHA: metadata lists assert as []any (yaml.v3 never produces []string).
            No t.Parallel() (repo convention). Use os.WriteFile + t.TempDir().

Task 3: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/discover/discover.go internal/discover/discover_test.go
  - COMMAND: gofmt -l internal/discover/*.go   # MUST print nothing
  - COMMAND: go vet ./internal/discover/       # MUST be clean
  - COMMAND: go build ./internal/discover/     # exit 0
  - COMMAND: go test ./internal/discover/ -v   # ALL discover tests PASS
  - COMMAND: go test ./...                     # whole module still green
  - EXPECT: zero errors, zero vet findings, gofmt silent, all tests pass.

Task 4: TIDY go.mod (yaml.v3 flips indirect -> direct; the ONLY go.mod change)
  - COMMAND: go mod tidy
  - COMMAND: grep 'gopkg.in/yaml.v3' go.mod    # the '// indirect' marker is GONE
  - COMMAND: git diff --stat go.sum            # go.sum unchanged (empty diff)
  - EXPECT: go.mod yaml.v3 line shows `require gopkg.in/yaml.v3 v3.0.1` with NO
            `// indirect`. go.sum byte-identical.

Task 5: LENIENCY + EDGE-CASE SMOKE TEST — Level 3 in Validation Loop
  - COMMAND: the Level 3 block below (run the targeted tests proving leniency,
             BOM, CRLF, folded scalar, malformed-YAML-error, no-close leniency).
  - EXPECT: all targeted tests pass.

Task 6: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: discover.go has the exact field/tag set, no KnownFields(true), no
            Skill/toStringSlice/Index; imports limited to the 4; skillsdir/main/
            PRD.md unchanged; go.mod's only change is the indirect marker; no
            resolve/ui/skills/ created.
```

### Implementation Patterns & Key Details

```go
// PATTERN: lenient-by-default unmarshal (do NOT KnownFields(true)).
//   if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
//       return Frontmatter{}, "", err
//   }
// WHY: yaml.v3's package-level Unmarshal ignores unknown struct fields by default.
//      That IS the leniency PRD §7.3 requires ("unknown frontmatter keys are
//      ignored"). Calling KnownFields(true) would flip it to strict and reject
//      every real-world SKILL.md with an extra field. Verified research §1.

// PATTERN: fence detection tolerant of BOM + CRLF.
//   data = bytes.TrimPrefix(data, utf8BOM)                 // strip ONE leading BOM
//   lines := strings.Split(string(data), "\n")
//   isFence := func(line string) bool { return strings.TrimRight(line, "\r") == fence }
// WHY: a BOM makes line[0] == "\ufeff---" (missed frontmatter); CRLF leaves a
//      trailing "\r" (line == "---\r"). Both fixed without altering body bytes.
//      Verified research §5, §6.

// PATTERN: two-stage leniency (missing fence = OK; broken YAML = error).
//   if len(lines)==0 || !isFence(lines[0]) { return Frontmatter{}, s, nil }   // no open fence
//   closeIdx := findNextFence(lines, 1)
//   if closeIdx == -1 { return Frontmatter{}, s, nil }                         // no close fence (lenient)
//   if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
//       return Frontmatter{}, "", err                                          // broken YAML (error)
//   }
// WHY: PRD §7.3 — "If no frontmatter block => skill still resolves by directory"
//      (no error). But malformed YAML is a genuine parse failure the caller
//      (check, M4) must surface. Verified research §7 vs §8.

// PATTERN: non-YAML metadata field via yaml:"-".
//   type Frontmatter struct {
//       ...
//       HasFM bool `yaml:"-"`   // set by the parser, never read/written by yaml
//   }
// WHY: HasFM records whether a `---` block was found (distinct from "frontmatter
//      with empty fields"). yaml:"-" keeps yaml.v3 from touching it. The zero
//      value (false) is the no-frontmatter state. Verified research §12.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/discover/discover.go is `package discover` (internal => unimportable
    outside the module; correct for a CLI's private packages).
  - imports (production): bytes, os, strings, gopkg.in/yaml.v3. (4 only.)
  - exposes: type Frontmatter (8 fields incl. HasFM); func ParseFrontmatter.
  - consumes: NOTHING in this repo (leaf library). Only yaml.v3 + stdlib.

GO.MOD (the ONE expected change):
  - before: require gopkg.in/yaml.v3 v3.0.1 // indirect
  - after:  require gopkg.in/yaml.v3 v3.0.1        (// indirect removed by `go mod tidy`)
  - go.sum: unchanged (yaml.v3 already checksummed in P1.M1.T1.S1).

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into):
  - P1.M2.T4.S2 (Skill + metadata extraction): defines `type Skill struct`
    (Dir, RelTag, Name, Description, Keywords, Category, Aliases, HasFM,
    SourceFile) and `toStringSlice(v any) []string`; builds a Skill from a
    Frontmatter by reading fm.Name, fm.Description, fm.Metadata (type-asserting
    keywords/aliases as []any -> []string, category as string), fm.HasFM. S2 does
    NOT re-parse; it consumes S1's Frontmatter verbatim.
  - P1.M2.T5 (Index walk): calls ParseFrontmatter(absSkillDir + "/<tag>/SKILL.md")
    for every dir containing a SKILL.md; propagates fm into the Skill builder.
  - P1.M4.T10 (check): uses ParseFrontmatter to detect missing frontmatter
    (HasFM=false => flag), validate name against ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$
    (1-64), and description <= 1024 (TrimSpace first to handle folded-scalar
    trailing newline — research §3).

NO CHANGES TO:
  - go.sum (byte-identical after tidy)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned) / prd_snapshot.md
  - internal/skillsdir/* (M1-owned; discover does not import it)
  - main.go / main_test.go (M1.T3.S1-owned; parallel, leave alone)
  - any other package or file (resolve/ui/install.sh/README/completions/skills/
    are later subtasks)
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass)
gofmt -w internal/discover/discover.go internal/discover/discover_test.go
test -z "$(gofmt -l internal/discover/*.go)" || { echo "FAIL: gofmt found unformatted files"; gofmt -d internal/discover/*.go; exit 1; }
echo "gofmt OK"

# Vet the discover package
go vet ./internal/discover/ || { echo "FAIL: go vet ./internal/discover/"; exit 1; }
echo "go vet OK"

# Build the leaf library (no main needed; proves it compiles standalone)
go build ./internal/discover/ || { echo "FAIL: go build ./internal/discover/"; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run discover tests verbosely — all parser cases
go test ./internal/discover/ -v || { echo "FAIL: go test ./internal/discover/ -v"; exit 1; }

# Whole module still green (skillsdir + discover; + main if M1.T3.S1 has landed).
# Works whether or not main.go exists (a root with no .go files is simply skipped).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Leniency + edge-case smoke test (the contract this subtask locks)

```bash
cd /home/dustin/projects/skpp

# The targeted tests proving each load-bearing behavior:
#   - unknown keys ignored (lenient; NOT KnownFields(true))
#   - folded scalar '>' parsed (trailing newline retained, not trimmed)
#   - BOM-prefixed file detects frontmatter
#   - CRLF file detects frontmatter
#   - no closing fence => lenient no-frontmatter (no error)
#   - malformed YAML => error propagated (leniency is about keys, not syntax)
go test ./internal/discover/ -v \
  -run 'TestParseUnknownKeysIgnored|TestParseFoldedScalarDescription|TestParseBOMPrefixed|TestParseCRLF|TestParseNoClosingFenceLenient|TestParseMalformedYAMLErrors|TestParseFullFrontmatter|TestParseNoFrontmatter' \
  || { echo "FAIL: leniency/edge-case tests"; exit 1; }
echo "Level 3 PASS (leniency + BOM + CRLF + folded + malformed-YAML + no-close)"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# discover.go exists and is package discover
test -f internal/discover/discover.go || { echo "FAIL: discover.go missing"; exit 1; }
grep -q '^package discover' internal/discover/discover.go || { echo "FAIL: discover.go not package discover"; exit 1; }

# Exact field/tag set (8 fields; HasFM has yaml:"-")
grep -q 'Name                   string         `yaml:"name"`' internal/discover/discover.go || { echo "FAIL: Name field/tag"; exit 1; }
grep -q 'Description            string         `yaml:"description"`' internal/discover/discover.go || { echo "FAIL: Description field/tag"; exit 1; }
grep -q 'License                string         `yaml:"license,omitempty"`' internal/discover/discover.go || { echo "FAIL: License field/tag"; exit 1; }
grep -q 'Compatibility          string         `yaml:"compatibility,omitempty"`' internal/discover/discover.go || { echo "FAIL: Compatibility field/tag"; exit 1; }
grep -q 'Metadata               map\[string\]any `yaml:"metadata,omitempty"`' internal/discover/discover.go || { echo "FAIL: Metadata field/tag"; exit 1; }
grep -q 'AllowedTools           string         `yaml:"allowed-tools,omitempty"`' internal/discover/discover.go || { echo "FAIL: AllowedTools field/tag"; exit 1; }
grep -q 'DisableModelInvocation bool           `yaml:"disable-model-invocation,omitempty"`' internal/discover/discover.go || { echo "FAIL: DisableModelInvocation field/tag"; exit 1; }
grep -q 'HasFM                  bool           `yaml:"-"`' internal/discover/discover.go || { echo "FAIL: HasFM field/tag (yaml:\"-\")"; exit 1; }

# Exact function signature (matches go_architecture.md contract)
grep -qE 'func ParseFrontmatter\(path string\) \(fm Frontmatter, body string, err error\)' internal/discover/discover.go \
  || { echo "FAIL: ParseFrontmatter signature"; exit 1; }

# MUST NOT call KnownFields(true) — leniency depends on the default
! grep -q 'KnownFields' internal/discover/discover.go || { echo "FAIL: KnownFields found (breaks leniency)"; exit 1; }

# MUST NOT trim the folded-scalar description (preserve fidelity)
! grep -qE 'TrimSpace.*Description|Description.*TrimSpace' internal/discover/discover.go || { echo "FAIL: do not trim Description"; exit 1; }

# Imports limited to the 4 expected (bytes, os, strings, yaml.v3) — no fmt/io/filepath
test "$(go list -f '{{join .Imports \" \"}}' ./internal/discover/ 2>/dev/null)" = "bytes os strings gopkg.in/yaml.v3" \
  || { echo "FAIL: imports must be exactly [bytes os strings gopkg.in/yaml.v3]"; go list -f '{{join .Imports \" \"}}' ./internal/discover/; exit 1; }

# MUST NOT define Skill / toStringSlice / Index (S2/S5 own those)
! grep -q 'type Skill ' internal/discover/discover.go || { echo "FAIL: Skill struct must not exist (S2)"; exit 1; }
! grep -q 'func Index' internal/discover/discover.go || { echo "FAIL: Index() must not exist (S5)"; exit 1; }
! grep -q 'toStringSlice' internal/discover/discover.go || { echo "FAIL: toStringSlice must not exist (S2)"; exit 1; }

# discover_test.go is white-box package discover with the key tests
test -f internal/discover/discover_test.go || { echo "FAIL: discover_test.go missing"; exit 1; }
grep -q '^package discover' internal/discover/discover_test.go || { echo "FAIL: discover_test.go must be package discover (white-box)"; exit 1; }
for tn in TestParseFullFrontmatter TestParseNoFrontmatter TestParseUnknownKeysIgnored TestParseFoldedScalarDescription TestParseQuotedValues TestParseMetadataShapes TestParseBOMPrefixed TestParseCRLF TestParseNoClosingFenceLenient TestParseMalformedYAMLErrors TestParseEmptyFile; do
  grep -q "func $tn" internal/discover/discover_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# go.mod: yaml.v3 is now DIRECT (// indirect removed by tidy) — the ONLY change
grep -q '^require gopkg.in/yaml.v3 v3.0.1$' go.mod \
  || grep -q '^	gopkg.in/yaml.v3 v3.0.1$' go.mod \
  || { echo "FAIL: go.mod yaml.v3 must be direct (no // indirect)"; cat go.mod; exit 1; }
! grep -q 'yaml.v3 v3.0.1 // indirect' go.mod || { echo "FAIL: yaml.v3 still // indirect (run go mod tidy)"; exit 1; }

# MUST NOT have touched go.sum / PRD.md / skillsdir / main (if present)
git diff --quiet go.sum   || { echo "FAIL: go.sum changed (should be byte-identical)"; exit 1; }
git diff --quiet PRD.md   || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go      || { echo "FAIL: skillsdir.go changed"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir_test.go || { echo "FAIL: skillsdir_test.go changed"; exit 1; }
# main.go is owned by M1.T3.S1 (parallel); only check if it exists to avoid coupling
test -f main.go && { git diff --quiet main.go || { echo "FAIL: main.go changed (owned by M1.T3.S1)"; exit 1; }; } || true

# MUST NOT have created later-milestone files/packages
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
- [ ] Level 1 PASS — `gofmt -l internal/discover/*.go` silent, `go vet ./internal/discover/` clean, `go build ./internal/discover/` exit 0
- [ ] Level 2 PASS — `go test ./internal/discover/ -v` all discover tests pass; `go test ./...` whole module green
- [ ] Level 3 PASS — leniency (unknown keys ignored), folded scalar `>`, BOM, CRLF, no-closing-fence leniency, malformed-YAML error all verified by targeted tests
- [ ] Level 4 PASS — exact 8-field struct with exact tags; `ParseFrontmatter` signature; no `KnownFields(true)`; imports = bytes/os/strings/yaml.v3; no Skill/Index/toStringSlice; white-box test file with all key tests; go.mod yaml.v3 direct; go.sum/skillsdir/main/PRD unchanged; no later-milestone files

### Feature Validation
- [ ] Full frontmatter parses all 8 paths (Name/Description/License/Compatibility/Metadata/AllowedTools/DisableModelInvocation/HasFM); `HasFM == true`; body = text after closing fence
- [ ] No frontmatter → `Frontmatter{}` (`HasFM == false`), whole file as body, `nil` error
- [ ] Unknown keys silently ignored (no error; known fields still parsed)
- [ ] Folded scalar `>` description parsed (trailing `\n` retained, NOT trimmed)
- [ ] Quoted values unquoted (double + single quotes)
- [ ] BOM-prefixed file detects frontmatter (BOM stripped before fence check)
- [ ] CRLF file detects frontmatter (trailing `\r` tolerated on fence lines)
- [ ] No closing fence → lenient no-frontmatter (no error)
- [ ] Malformed YAML between valid fences → error propagated
- [ ] Empty file → `(Frontmatter{}, "", nil)` (no panic)
- [ ] Nonexistent file → os.ReadFile error propagated

### Code Quality / Convention Validation
- [ ] `discover.go` is `package discover` (internal); imports limited to bytes/os/strings/yaml.v3
- [ ] `discover_test.go` is white-box `package discover`, mirroring skillsdir_test.go's style (t.TempDir/os.WriteFile, plain t.Errorf/t.Fatalf, no testify, no t.Parallel)
- [ ] Package doc comment + per-symbol doc comments explain the leniency contract and cite the research sections
- [ ] Magic strings (`"---"`, the BOM bytes) are named package-level vars (`fence`, `utf8BOM`), not inlined — mirrors skillsdir's `const envVar`
- [ ] The parser returns zero `Frontmatter{}` and empty body on ANY error path (no half-filled struct)

### Scope Discipline
- [ ] Did NOT define `type Skill struct` / `toStringSlice` / `Index()` (S2/S5 own those)
- [ ] Did NOT modify `internal/skillsdir/*` (M1-owned; discover does not import it)
- [ ] Did NOT modify `main.go` / `main_test.go` (M1.T3.S1-owned, parallel)
- [ ] Did NOT modify `PRD.md` (read-only) or any `tasks.json` (orchestrator-owned)
- [ ] Did NOT modify `go.sum` (byte-identical); the ONLY go.mod change is `// indirect` removed from yaml.v3 (via `go mod tidy`)
- [ ] Did NOT create `resolve` / `ui` / `install.sh` / `README.md` / `completions/` / `skills/` (later milestones)

---

## Anti-Patterns to Avoid

- ❌ **Don't call `KnownFields(true)`.** yaml.v3 is lenient by default (unknown
  keys ignored). `KnownFields(true)` makes unknown keys HARD-ERROR — the opposite
  of PRD §7.3. Verified research §1.
- ❌ **Don't conflate "lenient about unknown keys" with "tolerate broken YAML".**
  A missing closing fence → no frontmatter, no error (lenient). But malformed
  YAML between valid fences → real error. Two different cases. Verified §7/§8.
- ❌ **Don't forget to strip the UTF-8 BOM.** A BOM-marked file's first line is
  `\ufeff---`, which `!= "---"` → frontmatter silently missed. `bytes.TrimPrefix`
  first. Verified §5.
- ❌ **Don't compare fence lines without tolerating a trailing `\r`.** CRLF files
  have `---\r` lines. `strings.TrimRight(line, "\r") == fence`. Verified §6.
- ❌ **Don't trim the folded-scalar description.** yaml.v3 keeps a trailing `\n`
  on `>`/`|` scalars. Return it verbatim; trimming corrupts `|` blocks and steals
  the M4 check's job. Verified §3.
- ❌ **Don't expect `Metadata["keywords"]` to be `[]string`.** yaml.v3 produces
  `[]interface{}` (`[]any`). S2 type-asserts. S1 only exposes the raw map.
  Verified §2.
- ❌ **Don't omit `HasFM` or tag it as a YAML key.** It must be `HasFM bool
  `yaml:"-"`` — a parser-set flag, not a frontmatter field. Verified §12.
- ❌ **Don't define `Skill` / `toStringSlice` / `Index()` here.** S2 owns Skill +
  metadata extraction; S5 owns Index. Defining them now collides with S2/S5.
  Verified §13.
- ❌ **Don't half-fill the struct on an error.** On malformed YAML or read error,
  return `Frontmatter{}` (zero) and `""` body — never a partially-populated struct
  that a caller might mistake for valid data.
- ❌ **Don't depend on `main.go` existing.** Scope validation to
  `./internal/discover/`. This subtask is a leaf library; it builds and tests
  standalone whether or not M1.T3.S1 has landed.

---

## Confidence Score

**9/10** — one-pass implementation success likelihood.

Rationale: every load-bearing decision (leniency default, BOM/CRLF handling,
folded-scalar trailing newline, `map[string]any` typing, `HasFM` field, malformed-
vs-missing-fence distinction, the indirect→direct go.mod flip, the S1/S2 scope
split) was **empirically executed** against the project's real yaml.v3 v3.0.1 in
its real Go 1.26.4 toolchain, and the exact `discover.go` and `discover_test.go`
source is provided verbatim in the Implementation Blueprint (the verification
program used the identical struct + algorithm and produced exactly the asserted
outputs). The residual 1/10 is ordinary implementer-transcription risk (a typo in
a yaml tag, a forgotten `go mod tidy`) which the Level 4 grep-based contract check
catches deterministically.
