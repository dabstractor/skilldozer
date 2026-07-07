# PRP — P1.M1.T2.S1: Add `SourceConfig` enum value + `config file` label + refresh doc comments + export `HasSkillMD`

> **Subtask:** P1.M1.T1's sibling-layer for skills-dir discovery — the enum/label/export/doc prep that the next subtask (**P1.M1.T2.S2**, which implements `findConfig` and wires it into `Find()` at priority #2) consumes.
> **Scope boundary:** A pure, additive refactor of **one file** (`internal/skillsdir/skillsdir.go`) and its **one test file** (`internal/skillsdir/skillsdir_test.go`). No new files, no new imports, no `main.go` change, no `internal/config` change, no `findConfig` implementation, no `ErrNotFound` message change (those are T2.S2). Adds an exported `SourceConfig` constant + its `"config file"` String label + a refreshed 5-rule doc ladder, and exports the existing `hasSkillMD` predicate as `HasSkillMD` for `init`'s future cwd-auto-detect.

---

## Goal

**Feature Goal**: Prepare `internal/skillsdir` for the §8 config-file discovery rule by (a) adding the `SourceConfig` enum value at the correct iota position so `Source` values track PRD §8.3 priority order, (b) giving it the exact `"config file"` label PRD §8.3 enumerates, (c) refreshing the package/`Find()`/`ErrNotFound` doc comments from the stale "3 rules" to the 5-rule ladder (`env → config → sibling → walk-up → unconfigured`), and (d) exporting the cwd-auto-detect predicate `hasSkillMD` → `HasSkillMD` so `skilldozer init` (P1.M2.T2.S1) can reuse it instead of duplicating it.

**Deliverable**: Edits to two existing files only:
1. `internal/skillsdir/skillsdir.go` — add `SourceConfig` to the `const` block between `SourceEnv` and `SourceSibling`; add `case SourceConfig: return "config file"` to `Source.String()`; rename `hasSkillMD` → `HasSkillMD` (exported) + update its doc comment + its single production caller (`findWalkUpAncestor`); rewrite the package doc comment, the `Find()` doc comment, and the `ErrNotFound` doc comment to the 5-rule ladder.
2. `internal/skillsdir/skillsdir_test.go` — add `{SourceConfig, "config file"}` to `TestSourceString`; update the 4 `hasSkillMD(...)` test call sites (and the section comment + `t.Errorf` messages) to `HasSkillMD`.

**Success Definition**: `go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l internal/skillsdir/` prints nothing; `go.mod`/`go.sum` byte-for-byte unchanged; the exported surface of `skillsdir` grows from `{Find, Source, SourceEnv, SourceSibling, SourceWalkUp, ErrNotFound}` to `{Find, Source, SourceEnv, SourceConfig, SourceSibling, SourceWalkUp, HasSkillMD, ErrNotFound}`; `SourceConfig`'s numeric value is `1` (Env=0, Config=1, Sibling=2, WalkUp=3); `SourceConfig.String()` returns exactly `"config file"`; and the `ErrNotFound` message STRING is untouched (left for T2.S2).

---

## User Persona (if applicable)

**Not applicable** — internal package, no user-facing surface. The contract DOCS §5 is explicit: "none — no user-facing surface (the `--path` stderr label is surfaced by `main.go` unchanged once `Source.String()` returns it; that behavior already exists at `main.go:421`)." The `"config file"` label becomes user-visible only later, transitively, when `findConfig` (T2.S2) returns `SourceConfig` and `main.go:423`'s `fmt.Fprintf(stderr, "(found via %s)\n", src)` renders it — with zero `main.go` change.

---

## Why

- **Unblocks the config rule.** T2.S2's `findConfig` must `return (absDir, SourceConfig, true)` on a hit. It cannot compile until the `SourceConfig` constant exists. This is the dependency root for the entire §8.1 config-file mechanism landing.
- **Closes gap G2** (`code_prd_delta.md`): "`Source.String()` has no `config file` case. PRD §8.3 … requires the new label. The §13 acceptance gate `grep -q /tmp/skilldozer-store` (config rule wins) depends on this."
- **Closes gap G3**: the package/`Find()` doc comments still describe "3 rules" / "all three §8 rules" while §8.3 now mandates five. The doc drift misleads the next implementer.
- **Closes gap G12**: `hasSkillMD` is "the cwd-auto-detect predicate" already used by walk-up; exporting it lets `init` (P1.M2.T2.S1) reuse the exact §8.2 "contains ≥1 SKILL.md" semantics instead of re-implementing (and subtly diverging from) them.
- **Closes gap G14**: `TestSourceString` has no `{SourceConfig, "config file"}` row.
- It is the safest possible unit to land first: **additive + rename within one package**, zero new logic, zero new imports, fully covered by the existing test file's assertions.

---

## What

Four edits to `internal/skillsdir/skillsdir.go` and two to `internal/skillsdir/skillsdir_test.go`, all pinned exactly by the contract INPUT/LOGIC and verified against the live file (see `research/verified_facts.md`).

**(a) Add `SourceConfig` to the const block** — between `SourceEnv` and `SourceSibling`:
```go
const (
	// SourceEnv means SKILLDOZER_SKILLS_DIR was set and pointed at an existing dir.
	SourceEnv Source = iota
	// SourceConfig means the skills dir was read from the config file's `store` key (PRD §8.1).
	SourceConfig
	// SourceSibling means the skills dir was found next to the running binary.
	SourceSibling
	// SourceWalkUp means the skills dir was found by walking up from cwd.
	SourceWalkUp
)
```
Resulting iota: **Env=0, Config=1, Sibling=2, WalkUp=3** (Sibling/WalkUp each shift up by 1 — verified safe; see Known Gotchas #2).

**(b) Add `case SourceConfig: return "config file"` to `Source.String()`** — placed between the `SourceEnv` and `SourceSibling` cases so the switch reads in PRD §8.3 label order:
```go
func (s Source) String() string {
	switch s {
	case SourceEnv:
		return "SKILLDOZER_SKILLS_DIR"
	case SourceConfig:
		return "config file"
	case SourceSibling:
		return "sibling of binary"
	case SourceWalkUp:
		return "ancestor of cwd"
	default:
		return "unknown"
	}
}
```
PRD §8.3 enumerates exactly these four labels (no others).

**(c) Refresh the "3 rules" doc comments to the 5-rule ladder.** Three sites (see Implementation Tasks for exact replacement text): the package doc comment, the `ErrNotFound` doc comment, and the `Find()` doc comment. Describe the ladder **by Source label**, not by the not-yet-existing `findConfig` helper, so the comments are accurate the moment T2.S1 lands and stay accurate after T2.S2 wires `findConfig` in. **Do NOT** alter the `var ErrNotFound = errors.New("…")` message string — that flip is T2.S2.

**(d) Export `hasSkillMD` → `HasSkillMD`.** Rename the function (the contract permits a thin delegate wrapper, but a direct rename is cleaner — one symbol, no indirection). Update its doc comment to note it doubles as the §8.2 cwd-auto-detect predicate, and update its single production caller `findWalkUpAncestor`.

**(e) Update `TestSourceString`** — add `{SourceConfig, "config file"}` to the cases table (after the `SourceEnv` row, to mirror priority order).

**(f) Update the renamed `hasSkillMD` references in the test file** — 4 call sites + the section comment + the `t.Errorf` message strings (cosmetic but keep them consistent). The test FUNCTION names (`TestHasSkillMD…`) already use the capitalized form Go requires, so they need no change.

### Success Criteria

- [ ] `SourceConfig` constant exists between `SourceEnv` and `SourceSibling`; numeric value is `1`
- [ ] `Source.String()` returns exactly `"config file"` for `SourceConfig`
- [ ] Package doc comment lists the 5-rule ladder (`env → config → sibling → walk-up → unconfigured`), not "3 rules"
- [ ] `Find()` doc comment lists the 5-rule ladder by Source label
- [ ] `ErrNotFound` **doc comment** no longer says "all three §8 rules"; the `ErrNotFound` **message string is unchanged** (still `"...set $SKILLDOZER_SKILLS_DIR, cd into the skilldozer repo, or reinstall skilldozer"`)
- [ ] Exported `HasSkillMD(dir string) bool` exists; `findWalkUpAncestor` calls `HasSkillMD(candidate)`
- [ ] `TestSourceString` asserts `{SourceConfig, "config file"}`
- [ ] All 4 `hasSkillMD` test call sites updated to `HasSkillMD`
- [ ] `go build/vet/test ./...` green; `gofmt -l internal/skillsdir/` empty; `go.mod`/`go.sum` unchanged

---

## All Needed Context

### Context Completeness Check

**Pass.** Every change is pinned to an exact code location I read in full (`internal/skillsdir/skillsdir.go` 1-242, `internal/skillsdir/skillsdir_test.go` 1-end), every referenced symbol's call sites are enumerated (`research/verified_facts.md` §3, §4), the iota-shift safety is grep-verified (§2), the `main.go` Stringer consumption is confirmed at line 423 (§4), and the T2.S2 boundary is crisply drawn (§5). An implementer who has never seen this repo can complete it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative gap analysis (G2/G3/G12/G14 are THIS subtask)
- file: plan/002_38acb6d28a6a/architecture/code_prd_delta.md
  why: "G2 (no SourceConfig / no 'config file' label), G3 (doc comments say '3 rules'), G12 (hasSkillMD unexported), G14 (TestSourceString has no SourceConfig case). §1 quotes the exact Source/String() code to edit; §6 quotes the exact TestSourceString table to extend. §3 line 191 confirms main.go needs NO change (the --path stderr label already flows from src.String())."
  section: "§1 (G2, Source/String site + iota-ordering sub-gap), §6 (G14 tests), §8 cross-cutting note on hasSkillMD export, §10 gap index (G2/G3/G12/G14)."

# MUST READ — the verified facts (line numbers, all hasSkillMD refs, iota-shift proof)
- file: plan/002_38acb6d28a6a/P1M1T2S1/research/verified_facts.md
  why: "Every claim in this PRP is anchored here: current Source/String line numbers, the full list of hasSkillMD references (1 prod caller + 4 test call sites + comments), the grep proof that no code switches on numeric Source values, the confirmation that main.go consumes Source.String() via fmt %s, and the T2.S2 boundary (ErrNotFound message string is OFF LIMITS)."
  critical: "§5 (SCOPE BOUNDARY with T2.S2) and §6 (the exact '3 rules' doc-drift sites) are the two things most likely to be got wrong without this file."

# MUST READ — the file under edit (read in full before editing)
- file: internal/skillsdir/skillsdir.go
  why: "THE edit target. const block ~27-34; Source.String() ~38-49; hasSkillMD decl ~144-171 (caller at ~182); package doc comment ~1-22; ErrNotFound comment+var ~218-221; Find() doc comment ~224-231."
  pattern: "Existing doc-comment style: godoc list syntax (numbered '1. 2. 3.' with leading two-space indent per godoc); per-rule helper naming findXxx; Source label returned verbatim from String(). Mirror it."
  gotcha: "The package doc comment still says '(added in P1.M1.T2.S3)' — there is no T2.S3; Find() already exists. Drop the parenthetical while rewriting the block. Do NOT cite subtask IDs in shipped doc comments."

# MUST READ — the test file under edit
- file: internal/skillsdir/skillsdir_test.go
  why: "THE other edit target. TestSourceString ~57-72 (add the SourceConfig row); the 4 hasSkillMD test calls ~298/305/315/328 + section comment ~288 + t.Errorf messages. Package is `skillsdir` (internal test) so the renamed HasSkillMD is directly callable."
  pattern: "Table-driven TestSourceString with {src, want} rows; direct got!=want assertions; no testify."
  gotcha: "Test FUNCTION names already start TestHasSkillMD (capitalized) — Go requires it — so the rename changes only the CALLS/prose inside them, not the function names."

# MUST READ — the sequential sibling PRP (defines what T2.S2 will consume from T2.S1)
- file: plan/002_38acb6d28a6a/P1M1T1S2/PRP.md
  why: "Confirms internal/config is fully landed ({File,Load,Save,Path,DefaultStore} + configEnv). T2.S1 does NOT import config; T2.S2's findConfig will compose config.Path()+config.Load() and RETURN SourceConfig. Reading this fixes the boundary: T2.S1 = enum/label/export/doc ONLY; T2.S2 = findConfig + Find() wiring + ErrNotFound message flip."

# READ-ONLY — the PRD sections selected as relevant
- file: PRD.md
  why: "READ-ONLY. §8.3 enumerates the EXACT four labels in priority order (SKILLDOZER_SKILLS_DIR, config file, sibling of binary, ancestor of cwd) — the 'config file' string is verbatim, not paraphrased. §8.1/§8.2 establish the 5-rule ladder (env → config → sibling → walk-up → unconfigured-hint) the doc comments must describe."
  section: "h2.7 / h3.8 / h3.10 (selected). The label list at the end of §8.3 is the single most load-bearing sentence for edit (b)."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/tasks.json
  why: "P1.M1.T2.S1's CONTRACT block is the authoritative INPUT/LOGIC/OUTPUT. This PRP transcribes it; if anything seems to contradict tasks.json, tasks.json wins."
```

### Current Codebase tree

```bash
$ cd /home/dustin/projects/skilldozer && tree internal/ main.go -L 2
internal/
├── check/      check.go  check_test.go
├── config/     config.go config_test.go   # FULLY LANDED (S1+S2): {File,Load,Save,Path,DefaultStore}
├── discover/   discover.go index.go skill.go + _test.go   # untouched by T2.S1
├── resolve/    resolve.go resolve_test.go  # only MENTIONS Source.String() in a comment
├── search/     search.go search_test.go    # untouched
├── skillsdir/  skillsdir.go skillsdir_test.go   # <-- THE edit targets (both files)
└── ui/         ui.go ui_test.go            # untouched
main.go         # untouched — consumes Source.String() via fmt %s at :423
$ grep -rn "SourceConfig\|hasSkillMD\|HasSkillMD" --include="*.go" . | grep -v "internal/skillsdir/"
# (empty — T2.S1 is fully self-contained; no external caller to update)
```

### Desired Codebase tree with files to be added and responsibility of file

```bash
internal/
└── skillsdir/
    ├── skillsdir.go        # EDIT: +SourceConfig const, +config file case, hasSkillMD→HasSkillMD, refreshed doc comments
    └── skillsdir_test.go   # EDIT: +{SourceConfig,"config file"} row, hasSkillMD→HasSkillMD call sites
```

**No new files.** Both edits are to existing files. Matches the package's one-source-file convention (`internal/discover`'s core is one file; `internal/config` is one file).

| File | T2.S1 responsibility |
|---|---|
| `internal/skillsdir/skillsdir.go` | Add `SourceConfig` + `"config file"` label; rename `hasSkillMD`→`HasSkillMD` + doc; refresh package/Find/ErrNotFound doc comments to 5-rule ladder |
| `internal/skillsdir/skillsdir_test.go` | Add the `SourceConfig` row to `TestSourceString`; rename the 4 `hasSkillMD` call sites + comment/messages to `HasSkillMD` |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — Insert SourceConfig BETWEEN SourceEnv and SourceSibling, not at the end.
// The contract is explicit ("immediately after SourceEnv") and PRD §8.3 lists the labels
// in priority order env → config → sibling → walk-up. Keeping the iota order aligned with
// the priority order means the const block, the String() switch, AND the TestSourceString
// table all read top-to-bottom in the order Find() consults them. Appending SourceConfig
// at the end (Sibling=1, WalkUp=2, Config=3) would compile and pass tests but diverge from
// the documented priority order — a future reader switching on the value would be misled.

// GOTCHA #2 — The iota shift (Sibling 1→2, WalkUp 2→3) is SAFE. Verified by grep: NO code
// switches on a numeric Source value. Tests use Source constants BY NAME plus Source(-1)/
// Source(99) for the explicit out-of-range "unknown" case. TestFindAllMissReturnsErrNotFound
// asserts `src != 0` — 0 is SourceEnv's zero value, unchanged by the shift. (research/
// verified_facts.md §2 has the grep transcript.) Do not "stabilize" the numbers with explicit
// assignments (SourceSibling Source = 2); iota is the project convention and explicit values
// would imply someone switches on them (nobody does).

// GOTCHA #3 — The contract says "update its single internal caller findWalkUpAncestor" for
// the hasSkillMD rename. That is the sole PRODUCTION caller. But hasSkillMD is ALSO called
// from 4 sites in skillsdir_test.go (package skillsdir, internal test). A rename that forgets
// the test call sites FAILS TO COMPILE (the unexported hasSkillMD no longer exists). Update
// all 4 call sites + the section comment "// --- HasSkillMD ---" + the t.Errorf message
// strings. (research/verified_facts.md §3 enumerates every line.)

// GOTCHA #4 — Do NOT change the `var ErrNotFound = errors.New(...)` MESSAGE STRING. The
// current string mentions "SKILLDOZER_SKILLS_DIR, cd, reinstall"; PRD §8.2 wants
// `run \`skilldozer init\``. That message flip — and the corresponding TestErrNotFoundMessageHasFix
// substring update — is T2.S2's exclusive deliverable ("set the exact ErrNotFound message").
// T2.S1 only refreshes the ErrNotFound DOC COMMENT (the "// ErrNotFound is returned …" prose)
// from "all three §8 rules" to the 5-rule reality. Editing the message string here collides
// with T2.S2. (research/verified_facts.md §5.)

// GOTCHA #5 — Describe the refreshed doc-comment ladder by Source LABEL (SourceEnv/
// SourceConfig/SourceSibling/SourceWalkUp), NOT by the not-yet-existing findConfig helper.
// findConfig is implemented by T2.S2, which lands AFTER T2.S1. If T2.S1's Find() doc comment
// says "2. Config file `store` (rule 2, findConfig)", the comment references a symbol that
// does not exist yet. godoc/gofmt do not type-check comment text (so it compiles), but the
// cleaner choice is to cite the Source label, which exists immediately and stays correct
// after T2.S2. Reserve helper-name citations for findEnv/findSibling/findWalkUp (which exist).

// GOTCHA #6 — main.go needs NO change. src is a skillsdir.Source; main.go:423 does
// fmt.Fprintf(stderr, "(found via %s)\n", src), and %s invokes Source.String() via the fmt
// Stringer interface. Once String() returns "config file" for SourceConfig, --path prints
// "(found via config file)" automatically. Verified by reading main.go:408-428 and by
// code_prd_delta.md §3 line 191. Do not touch main.go.

// GOTCHA #7 — Prefer a direct rename (hasSkillMD → HasSkillMD) over a thin exported delegate
// wrapper. The contract allows either ("rename … OR add a thin exported HasSkillMD that
// delegates"). A wrapper leaves two functions and forces the tests to pick which to call,
// and duplicates the WalkDir-with-early-exit logic description across two doc comments. One
// exported symbol is idiomatic Go and matches how Source/Find/ErrNotFound are already exported.

// GOTCHA #8 — Drop the stale "(added in P1.M1.T2.S3)" parenthetical in the package doc
// comment while you are rewriting it. There is no T2.S3 in the plan (P1.M1.T2 has only S1 and
// S2); Find() already exists at skillsdir.go:232. Describing Find() as the existing entry
// point is correct. Shipped doc comments should never cite subtask IDs — the existing one is
// an anti-pattern to clean up in passing.

// GOTCHA #9 — The "config file" label is a LITERAL, not a paraphrase. PRD §8.3 enumerates
// exactly: `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`.
// Return "config file" verbatim (lowercase, space, no quotes, no trailing period). The §13
// acceptance gate greps for the winning directory path, not the label, but the label is
// user-visible on stderr via --path and must match the PRD word-for-word.

// GOTCHA #10 — No new imports, no go mod tidy. T2.S1 changes zero import lists (the renamed
// HasSkillMD uses the same errors/io/fs/os/path/filepath already imported). go.mod/go.sum must
// be byte-for-byte unchanged. gofmt -l internal/skillsdir/ must print nothing after edits
// (godoc numbered lists need the two-space indent per item; gofmt will reflow if you mis-indent).
```

---

## Implementation Blueprint

### Data models and structure

**No data-model changes.** `Source` is an existing `type Source int`; T2.S1 adds one constant to its existing `const` block and one `case` to its existing `String()` method. `HasSkillMD` is the existing `hasSkillMD` function with a capitalized identifier. No structs, no interfaces, no new types.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT internal/skillsdir/skillsdir.go — add SourceConfig to the const block
  - FILE: internal/skillsdir/skillsdir.go (const block, currently lines ~27-34)
  - INSERT one constant + its doc comment between SourceEnv and SourceSibling:
        // SourceConfig means the skills dir was read from the config file's `store` key (PRD §8.1).
        SourceConfig
    so the block reads: SourceEnv Source = iota / SourceConfig / SourceSibling / SourceWalkUp
  - GOTCHA #1: insert BETWEEN Env and Sibling (priority order), NOT appended at the end.
  - GOTCHA #2: do NOT add explicit `= N` values; keep iota. The shift (Sibling 1→2, WalkUp 2→3)
    is grep-verified safe.
  - NAMING: SourceConfig (exported, CamelCase; matches SourceEnv/SourceSibling/SourceWalkUp).

Task 2: EDIT internal/skillsdir/skillsdir.go — add the "config file" case to Source.String()
  - FILE: internal/skillsdir/skillsdir.go (Source.String() switch, currently lines ~38-49)
  - INSERT one case between the SourceEnv and SourceSibling cases:
        case SourceConfig:
            return "config file"
  - GOTCHA #9: the literal is "config file" — verbatim per PRD §8.3, lowercase, space, no quotes.
  - PLACEMENT: between Env and Sibling cases so the switch reads in PRD §8.3 label order.

Task 3: EDIT internal/skillsdir/skillsdir.go — rename hasSkillMD → HasSkillMD (export)
  - FILE: internal/skillsdir/skillsdir.go
  - RENAME the declaration (currently ~153): `func hasSkillMD(dir string) bool {` →
        `func HasSkillMD(dir string) bool {`
  - UPDATE the doc comment (currently ~140, ~144) prose: "hasSkillMD does not walk" →
        "HasSkillMD does not walk"; "hasSkillMD reports whether" → "HasSkillMD reports whether".
    ADD one sentence noting it doubles as the §8.2 cwd-auto-detect predicate reused by init:
        "Exported because it doubles as the §8.2 cwd-auto-detect predicate: `skilldozer init`
         (P1.M2.T2.S1) uses it to decide whether the current working directory already looks
         like a store."  (Insert as a standalone doc paragraph after the existing first paragraph.)
  - UPDATE the SOLE production caller in findWalkUpAncestor (currently ~182):
        `if hasSkillMD(candidate) {` → `if HasSkillMD(candidate) {`
  - GOTCHA #3: the 4 TEST call sites are a SEPARATE task (Task 6) — do not forget them or the
    package will not compile.
  - GOTCHA #7: direct rename, NOT a delegate wrapper.

Task 4: EDIT internal/skillsdir/skillsdir.go — refresh the package doc comment to the 5-rule ladder
  - FILE: internal/skillsdir/skillsdir.go (package doc comment, currently lines 1-22)
  - REPLACE the numbered list "1. env / 2. Sibling / 3. Walk up" with the 5-rule ladder, and
    DROP the stale "(added in P1.M1.T2.S3)" parenthetical (GOTCHA #8). Target text:
        // Package skillsdir locates the on-disk skills/ directory for skilldozer.
        //
        // It implements the PRD §8.3 priority order (first hit wins):
        //
        //  1. SKILLDOZER_SKILLS_DIR env var — override; if set and an existing dir, use it as-is.
        //  2. Config file `store` (PRD §8.1) — the primary, set by `skilldozer init`.
        //  3. Sibling of the running binary (symlink-aware via os.Executable + EvalSymlinks).
        //  4. Walk up from the current working directory.
        //  5. None ⇒ unconfigured: Find returns ErrNotFound; the caller prints a one-line
        //     fix to stderr and exits 1.
        //
        // The public entry point is Find(), which calls the per-rule helpers in order and
        // returns the first hit. Each per-rule helper returns (dir string, src Source,
        // found bool) where found is true only when that rule produced a usable absolute
        // directory; on found==false the src value is meaningless and the caller falls
        // through to the next rule. Source.String() labels each rule for `--path` stderr
        // reporting (PRD §8.3): SKILLDOZER_SKILLS_DIR, config file, sibling of binary,
        // ancestor of cwd.
        package skillsdir
  - GOTCHA #5: describe rules by Source label, not by findConfig (does not exist until T2.S2).
  - PRESERVE the existing godoc list indent (two spaces before the number, four for continuation).

Task 5: EDIT internal/skillsdir/skillsdir.go — refresh ErrNotFound doc comment + Find() doc comment
  - FILE: internal/skillsdir/skillsdir.go
  - ErrNotFound DOC COMMENT (currently ~218-220): change "all three §8 rules miss" to
        "every §8.3 rule misses (unconfigured)". Target:
        // ErrNotFound is returned by Find when every §8.3 rule misses (unconfigured). Its
        // message is the user-facing one-line fix (PRD §8.4 / §6.4): main prints it to
        // stderr and exits 1. Print it verbatim (err.Error()); do not wrap or prefix it.
  - GOTCHA #4 (CRITICAL): DO NOT touch the `var ErrNotFound = errors.New("…")` STRING.
    Leave the message exactly: "could not locate the skills directory: set $SKILLDOZER_SKILLS_DIR,
    cd into the skilldozer repo, or reinstall skilldozer". T2.S2 owns the message flip.
  - Find() DOC COMMENT (currently ~224-231): replace the 3-rule list with the 5-rule ladder
    by Source label. Target:
        // Find locates the skills directory per PRD §8.3 priority order (first hit wins):
        //
        //  1. SKILLDOZER_SKILLS_DIR env var (SourceEnv).
        //  2. Config file `store` (SourceConfig).
        //  3. Sibling of the running binary, symlink-aware (SourceSibling).
        //  4. Walk up from cwd (SourceWalkUp).
        //  5. None ⇒ unconfigured: returns ErrNotFound.
        //
        // The first rule to hit wins and Find returns (absDir, src, nil). If all miss it
        // returns ("", 0, ErrNotFound); the caller (main) prints the error to stderr and
        // exits 1.
  - NOTE: Find()'s BODY is unchanged in T2.S1 (it still calls findEnv/findSibling/findWalkUp
    only). T2.S2 inserts the findConfig call at priority #2. The doc comment describes the
    target 5-rule design — accurate once T2.S2 lands.

Task 6: EDIT internal/skillsdir/skillsdir_test.go — TestSourceString + rename the hasSkillMD calls
  - FILE: internal/skillsdir/skillsdir_test.go
  - TestSourceString (currently ~57-72): ADD one row to the cases table, placed after the
        SourceEnv row to mirror priority order:
            {SourceConfig, "config file"},
  - RENAME the 4 hasSkillMD call sites to HasSkillMD (currently ~298, ~305, ~315, ~328):
        `hasSkillMD(...)` → `HasSkillMD(...)` in TestHasSkillMDFoundNested, TestHasSkillMDFoundShallow,
        TestHasSkillMDEmpty, TestHasSkillMDOnlyNonSkillFiles.
  - RENAME the section comment (currently ~288): "// --- hasSkillMD ---" → "// --- HasSkillMD ---".
  - RENAME the 4 t.Errorf message strings (currently ~299, ~306, ~316, ~329) from "hasSkillMD(…)"
    to "HasSkillMD(…)" for consistency (cosmetic; not required to compile, but keep prose honest).
  - RENAME the comment at ~365 "…(hasSkillMD recurses)" → "…(HasSkillMD recurses)".
  - GOTCHA #3: the test FUNCTION names (TestHasSkillMD*) already use the capitalized form —
    NO change needed there.
  - DO NOT touch TestErrNotFoundMessageHasFix (the message-flip test) — that is T2.S2.

Task 7: VERIFY in isolation, then whole-module + invariants
  - COMMAND: gofmt -l internal/skillsdir/        (must print NOTHING)
  - COMMAND: go vet ./internal/skillsdir/...     (exit 0)
  - COMMAND: go test ./internal/skillsdir/... -v  (all pass, incl. the new SourceConfig row +
            the renamed HasSkillMD tests)
  - COMMAND: go build ./... ; go vet ./... ; go test ./...   (whole module green — no regressions)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"   (MUST print "deps unchanged")
```

### Implementation Patterns & Key Details

```go
// Source enum — SourceConfig inserted at priority position #2 (PRD §8.3 label order).
type Source int

const (
	SourceEnv Source = iota
	SourceConfig // skills dir read from config file `store` (PRD §8.1) — added P1.M1.T2.S1
	SourceSibling
	SourceWalkUp
)

// Source.String — the four labels PRD §8.3 enumerates, in priority order.
func (s Source) String() string {
	switch s {
	case SourceEnv:
		return "SKILLDOZER_SKILLS_DIR"
	case SourceConfig:
		return "config file"
	case SourceSibling:
		return "sibling of binary"
	case SourceWalkUp:
		return "ancestor of cwd"
	default:
		return "unknown"
	}
}

// HasSkillMD — exported cwd-auto-detect predicate (PRD §8.2). Same body as the old
// hasSkillMD; only the identifier (and its doc) changed. findWalkUpAncestor now calls
// HasSkillMD(candidate); init (P1.M2.T2.S1) will call skillsdir.HasSkillMD(cwdSkills).
func HasSkillMD(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "SKILL.md" {
			found = true
			return errSkillMDFound // stop the walk (sentinel)
		}
		return nil
	})
	return found
}
```

Notes easy to get wrong:
- `Source.String()` returns the literal `"config file"` — no quotes, no capitalization, no period. PRD §8.3 is the source of truth for the exact four strings.
- The `ErrNotFound` **var** and `Find()` **body** are intentionally left calling only the 3 existing helpers. T2.S2 inserts `findConfig` into `Find()` and flips the `ErrNotFound` message. T2.S1's job is the enum/label/export/doc; pretending to wire `findConfig` here is scope creep that collides with T2.S2.

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Rename vs. delegate wrapper for HasSkillMD? → RENAME.** The contract allows either. A direct rename yields one exported symbol, no indirection, and keeps the (already-good) WalkDir+early-exit doc comment in one place. A wrapper (`func HasSkillMD(dir string) bool { return hasSkillMD(dir) }`) would leave two functions and force the existing 4 tests to either keep calling the private one (defeating the "exported" point) or migrate to the wrapper (extra churn for no benefit). Idiomatic Go exports by capitalizing.

2. **Where to place `SourceConfig` in the const block? → BETWEEN SourceEnv and SourceSibling.** The contract is explicit ("immediately after SourceEnv"). This keeps iota order == PRD §8.3 priority order, so the const block, the `String()` switch, and the `TestSourceString` table all read top-to-bottom in the order `Find()` consults them. Appending at the end would compile/pass but misrepresent priority to future readers.

3. **Do the refreshed doc comments cite `findConfig`? → NO, cite Source labels.** `findConfig` does not exist until T2.S2 lands. Citing `SourceEnv/SourceConfig/SourceSibling/SourceWalkUp` (which all exist after T2.S1) makes the comments accurate immediately and durable across T2.S2. Reserve helper-name citations for `findEnv/findSibling/findWalkUp` (which exist).

4. **Touch the `ErrNotFound` message string? → NO.** T2.S2's title is literally "set the exact ErrNotFound message". Editing it here creates a merge/sequence collision. T2.S1 refreshes only the ErrNotFound *doc comment* ("three §8 rules" → "every §8.3 rule misses"). The §13 acceptance gate that greps the new `run \`skilldozer init\`` wording runs at P1.M4.T1.S1, long after T2.S2 sets it.

5. **Drop the "(added in P1.M1.T2.S3)" parenthetical? → YES.** There is no T2.S3; `Find()` already exists. Since Task 4 rewrites that exact comment block, removing the stale parenthetical is a free correctness fix. Shipped doc comments should not cite subtask IDs.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod UNCHANGED. T2.S1 adds zero imports (errors/io/fs/os/path/filepath already present).
    No go get, no go mod tidy. (GOTCHA #10)

EXPORTED SURFACE DELTA:
  before: {Find, Source, SourceEnv, SourceSibling, SourceWalkUp, ErrNotFound}
  after:  {Find, Source, SourceEnv, SourceConfig, SourceSibling, SourceWalkUp, HasSkillMD, ErrNotFound}
  (additive only — no symbol removed; SourceEnv/Sibling/WalkUp numeric values shift but no
   caller depends on them, verified by grep)

CONSUMERS (NOT built in this subtask — listed to fix the interface):
  - skillsdir.findConfig (P1.M1.T2.S2):  returns (absDir, SourceConfig, true) on a config hit.
        Consumes the SourceConfig constant this PRP adds. Also flips the ErrNotFound message.
  - init cwd-auto-detect (P1.M2.T2.S1):  if skillsdir.HasSkillMD(cwdSkills) { default = cwd }
        Consumes the exported HasSkillMD this PRP adds (was unexported hasSkillMD).
  - main.go --path (NO CHANGE):  fmt.Fprintf(stderr, "(found via %s)\n", src) already invokes
        Source.String() via the fmt Stringer. "config file" renders automatically.

NO ROUTES / NO DATABASE / NO CONFIG-FIELD-ADDITIONS / NO NEW FILES.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after editing skillsdir.go)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l internal/skillsdir/   # must print NOTHING (run `gofmt -w internal/skillsdir/` if it lists a file)
go vet ./internal/skillsdir/... # expect exit 0
go build ./internal/skillsdir/... # expect exit 0
# Expected: zero output / exit 0. gofmt is the gate that catches a mis-indented godoc list.
```

### Level 2: Unit Tests (component validation — the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test ./internal/skillsdir/... -v
# Expected: ALL pass. The load-bearing assertions:
#   TestSourceString now includes {SourceConfig, "config file"} — proves the label.
#   TestHasSkillMDFoundNested / Shallow / Empty / OnlyNonSkillFiles still pass after the
#     hasSkillMD → HasSkillMD rename (proves all 4 call sites were updated; a missed site
#     would be a COMPILE error, not a test failure).
#   TestFindRuleEnvWins / TestFindRuleWalkUpWins / TestFindAllMissReturnsErrNotFound still
#     pass (proves the iota shift broke nothing — they assert src by NAME / zero value).
#   TestErrNotFoundMessageHasFix is UNCHANGED and still passes (proves the message string
#     was NOT touched — it still asserts the old SKILLDOZER_SKILLS_DIR/cd/reinstall substrings).

# Isolated re-run of just the two edited behaviors:
go test ./internal/skillsdir/... -run 'TestSourceString|TestHasSkillMD' -v
# Expected: PASS.
```

### Level 3: Whole-module regression + invariants

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"
go vet  ./...  ; echo "vet exit $?"
go test ./...  ; echo "test exit $?"
# Expected: all exit 0. (config, discover, resolve, check, ui, main all still green; no
# external package referenced SourceConfig/hasSkillMD, so nothing else can break.)

# GOTCHA #10 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
# Expected: "deps unchanged".

# Scope invariant: the ErrNotFound message STRING is untouched (T2.S2 owns it)
grep -c 'could not locate the skills directory: set \$SKILLDOZER_SKILLS_DIR, cd into the skilldozer repo, or reinstall skilldozer' internal/skillsdir/skillsdir.go
# Expected: 1 (the original line, unchanged). If 0, you accidentally edited the message — revert it.
```

### Level 4: Behavioral spot-checks (lock the hard claims at runtime)

```bash
cd /home/dustin/projects/skilldozer

# 4a. Prove SourceConfig == 1 and its label is exactly "config file" (locks GOTCHA #1 + #9):
cat > /tmp/srcprobe_test.go <<'EOF'
package skillsdir
import "testing"
func TestProbeSourceConfigValueAndLabel(t *testing.T) {
	if SourceConfig != 1 { t.Errorf("SourceConfig = %d, want 1 (Env=0,Config=1,Sibling=2,WalkUp=3)", SourceConfig) }
	if got := SourceConfig.String(); got != "config file" {
		t.Errorf("SourceConfig.String() = %q, want %q", got, "config file")
	}
	// label order sanity (PRD §8.3 priority order):
	if !(SourceEnv < SourceConfig && SourceConfig < SourceSibling && SourceSibling < SourceWalkUp) {
		t.Error("Source iota order != env<config<sibling<walkup")
	}
}
EOF
cp /tmp/srcprobe_test.go internal/skillsdir/srcprobe_test.go
go test ./internal/skillsdir/... -run TestProbeSourceConfigValueAndLabel -v
rm internal/skillsdir/srcprobe_test.go   # throwaway; keep only skillsdir_test.go
# Expected: PASS (proves the iota position and the verbatim label).

# 4b. Prove HasSkillMD is EXPORTED (the symbol main/init can reach). Compile a tiny external
#     package that references skillsdir.HasSkillMD — if it were still unexported, this fails:
mkdir -p /tmp/hasmdprobe && cat > /tmp/hasmdprobe/probe_test.go <<'EOF'
package hasmdprobe_test
import (
	"testing"
	"github.com/dabstractor/skilldozer/internal/skillsdir"
)
func TestHasSkillMDExported(t *testing.T) {
	// If HasSkillMD were unexported, this line is a COMPILE error (cannot refer to unexported name).
	_ = skillsdir.HasSkillMD
}
EOF
# (Run from a throwaway module to avoid polluting go.mod; or skip if internal/ is not importable
#  from outside the module — internal IS importable within github.com/dabstractor/skilldozer.
#  Simplest in-repo check: grep the exported symbol exists.)
grep -n '^func HasSkillMD' internal/skillsdir/skillsdir.go   # Expected: one match, the exported func
rm -rf /tmp/hasmdprobe /tmp/srcprobe_test.go
# Expected: the grep prints the HasSkillMD declaration line.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l internal/skillsdir/` empty, `go vet ./internal/skillsdir/...` exit 0, `go build` exit 0
- [ ] Level 2 PASS — `go test ./internal/skillsdir/... -v` all pass; `TestSourceString` covers `{SourceConfig, "config file"}`; the 4 `HasSkillMD` tests pass
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0; `git diff go.mod go.sum` reports "deps unchanged"; the `ErrNotFound` message-string grep returns 1 (untouched)
- [ ] Level 4 PASS — the `SourceConfig == 1` + `"config file"` probe passes; `^func HasSkillMD` is exported

### Feature Validation
- [ ] `SourceConfig` constant exists between `SourceEnv` and `SourceSibling`; numeric value is `1`
- [ ] `Source.String()` returns exactly `"config file"` for `SourceConfig` (verbatim PRD §8.3 label)
- [ ] Package doc comment lists the 5-rule ladder (`env → config → sibling → walk-up → unconfigured`), not "3 rules"; the stale "(added in P1.M1.T2.S3)" parenthetical is gone
- [ ] `Find()` doc comment lists the 5-rule ladder by Source label
- [ ] `ErrNotFound` **doc comment** no longer says "all three §8 rules"; the `ErrNotFound` **message string is unchanged**
- [ ] Exported `HasSkillMD(dir string) bool` exists; `findWalkUpAncestor` calls `HasSkillMD(candidate)`
- [ ] `TestSourceString` asserts `{SourceConfig, "config file"}`; the 4 `hasSkillMD` test call sites are renamed to `HasSkillMD`
- [ ] `main.go` is UNCHANGED (the `--path` label flows automatically via `src.String()`)

### Code Quality / Convention Validation
- [ ] Matches the existing `Source` enum/String() style (iota, no explicit values, doc comment per constant)
- [ ] Matches the existing doc-comment style (godoc numbered list with two-space indent)
- [ ] Matches the existing test style (table-driven `TestSourceString`, direct assertions, no testify)
- [ ] No new imports; no `go get`; no `go mod tidy`; `go.mod`/`go.sum` unchanged
- [ ] No new files; both edits are to the two existing `internal/skillsdir/` files
- [ ] No subtask IDs cited in shipped doc comments (the stale T2.S3 parenthetical removed)

### Scope Discipline (the T2.S2 boundary)
- [ ] Did NOT implement `findConfig` or wire it into `Find()` (T2.S2)
- [ ] Did NOT change the `ErrNotFound` message string (T2.S2 owns the `run \`skilldozer init\`` flip)
- [ ] Did NOT touch `TestErrNotFoundMessageHasFix` (T2.S2 updates its substrings)
- [ ] Did NOT touch `main.go`, `internal/config`, `README.md`, completions, or the example skill
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't append `SourceConfig` at the end of the const block.** Insert it between `SourceEnv` and `SourceSibling` so iota order tracks PRD §8.3 priority order. End-append compiles but misrepresents priority.
- ❌ **Don't add explicit `= N` values to "stabilize" the iota shift.** No code switches on numeric `Source` values (grep-verified). iota is the project convention; explicit values imply someone does.
- ❌ **Don't change the `ErrNotFound` message string.** T2.S2's title is "set the exact ErrNotFound message". Editing it here collides. Refresh only the *doc comment*.
- ❌ **Don't forget the 4 `hasSkillMD` test call sites.** The contract says "single internal caller" (findWalkUpAncestor) — that's the sole PRODUCTION caller. The 4 test calls in `skillsdir_test.go` (package `skillsdir`, internal test) also reference the unexported name; a rename that misses them fails to compile.
- ❌ **Don't use a delegate wrapper for `HasSkillMD`.** A direct rename is one symbol, no indirection, idiomatic. A wrapper leaves two functions and duplicates the doc.
- ❌ **Don't cite `findConfig` in the refreshed doc comments.** It does not exist until T2.S2. Cite the `Source` labels instead (they exist after T2.S1).
- ❌ **Don't paraphrase the `"config file"` label.** PRD §8.3 enumerates exactly four labels word-for-word; `"config file"` is a literal (lowercase, space, no quotes, no period).
- ❌ **Don't touch `main.go`.** `src.String()` is already invoked via `fmt.Fprintf(stderr, "(found via %s)\n", src)` at main.go:423; the new label renders with zero change.
- ❌ **Don't `go get`/`go mod tidy`/add imports.** T2.S1 changes zero import lists. `go.mod`/`go.sum` must be byte-for-byte unchanged.
- ❌ **Don't leave the stale "(added in P1.M1.T2.S3)" parenthetical.** There is no T2.S3; `Find()` already exists. Since Task 4 rewrites that comment block, drop it. Shipped doc comments cite no subtask IDs.

---

## Confidence Score

**9.5/10** — This is the smallest, most mechanical subtask in the plan: an additive enum constant + one switch case + one identifier rename + three doc-comment rewrites, all within a single package I read in full, with every call site enumerated and grep-verified (`research/verified_facts.md`). The iota-shift safety is proven by grep (no numeric `Source` consumers), the `main.go`-no-change claim is confirmed by reading lines 408-428, and the T2.S2 boundary (don't touch the `ErrNotFound` string) is crisply drawn and reinforced by a grep gate in Level 3. The 0.5 reservation is for the single human-judgment surface: the refreshed doc comments describe the 5-rule *design* before `findConfig` (T2.S2) actually wires rule #2 into `Find()` — accurate and intentional (sequential siblings), but a reviewer could prefer T2.S1 leave `Find()`'s comment at "3 rules" until T2.S2 lands. This PRP resolves toward describing the target now, since G3 explicitly asks for the 5-rule ladder and T2.S2 lands immediately after.
