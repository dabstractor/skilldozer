# P1.M1.T2.S1 — Verified Facts & Scope Notes

Source of truth for the PRP. Every claim below was checked against the live repo
on 2026-07-07. Line numbers are from the **current** `internal/skillsdir/skillsdir.go`
(not the contract's slightly-stale citations).

## 1. Current Source enum + String() (the insertion site)

`internal/skillsdir/skillsdir.go`:

```go
24: // Source identifies which §8 rule located the skills directory ...
25: type Source int
26:
27: const (
28:	// SourceEnv means SKILLDOZER_SKILLS_DIR was set ...
29:	SourceEnv Source = iota
30:	// SourceSibling means the skills dir was found next to the running binary.
31:	SourceSibling
32:	// SourceWalkUp means the skills dir was found by walking up from cwd.
33:	SourceWalkUp
34: )
35:
38: func (s Source) String() string {
39:	switch s {
40:	case SourceEnv:
41:		return "SKILLDOZER_SKILLS_DIR"
42:	case SourceSibling:
43:		return "sibling of binary"
44:	case SourceWalkUp:
45:		return "ancestor of cwd"
46:	default:
47:		return "unknown"
48:	}
49: }
```

Insertion: `SourceConfig` goes between `SourceEnv` and `SourceSibling` in BOTH the
const block and the `String()` switch. Resulting iota: Env=0, Config=1,
Sibling=2, WalkUp=3. PRD §8.3 lists the four labels in this exact priority order.

## 2. The iota shift is SAFE (verified by grep — no numeric Source assumptions)

`grep -rn 'Source(0|Source(1|Source(2|Source(3|case 0|case 1|case 2|case 3'`
over `internal/skillsdir/` and `main.go` (excluding `//`, `iota`, `Source(-1)`,
`Source(99)`) returns **nothing**. Tests use Source constants BY NAME
(`SourceEnv`, `SourceSibling`, `SourceWalkUp`) plus `Source(-1)`/`Source(99)` for
the explicit out-of-range `"unknown"` case. `TestFindAllMissReturnsErrNotFound`
asserts `src != 0` — 0 is `SourceEnv`'s zero value, unchanged by the shift. →
Shifting Sibling 1→2 and WalkUp 2→3 breaks nothing.

## 3. hasSkillMD — ALL references (rename touches every one)

`grep -rn 'hasSkillMD' --include='*.go'`:

PRODUCTION (`internal/skillsdir/skillsdir.go`):
- `:140` doc-comment prose: "…so hasSkillMD does not walk the entire tree."
- `:144` doc-comment prose: "hasSkillMD reports whether dir contains …"
- `:153` declaration: `func hasSkillMD(dir string) bool {`
- `:182` the SOLE production caller: `if hasSkillMD(candidate) {` (inside
  `findWalkUpAncestor`)

TESTS (`internal/skillsdir/skillsdir_test.go` — package `skillsdir`, INTERNAL test
so it currently calls the unexported name directly):
- `:288` section comment: `// --- hasSkillMD ---`
- `:298` call in `TestHasSkillMDFoundNested`
- `:299` `t.Errorf("hasSkillMD(nested SKILL.md): …")` message
- `:305` call in `TestHasSkillMDFoundShallow`
- `:306` `t.Errorf("hasSkillMD(shallow SKILL.md): …")` message
- `:315` call in `TestHasSkillMDEmpty`
- `:316` `t.Errorf("hasSkillMD(empty skills): …")` message
- `:328` call in `TestHasSkillMDOnlyNonSkillFiles`
- `:329` `t.Errorf("hasSkillMD(only README.md): …")` message
- `:365` comment: "Rule 3: a nested SKILL.md … counts (hasSkillMD recurses)."

→ The contract's "single internal caller findWalkUpAncestor" describes the sole
PRODUCTION caller. The rename ALSO touches 4 test call sites + the test function
names already use the capitalized `HasSkillMD` form (Go requires it), so the test
FUNCTION names need no change — only the calls/prose inside them. The 4 `t.Errorf`
message strings are cosmetic but should be renamed for consistency.

Decision: **RENAME** `hasSkillMD` → `HasSkillMD` (not a thin delegate wrapper).
One symbol, no redundant indirection, idiomatic Go. A wrapper would leave two
functions and force tests to choose which to call.

## 4. External consumers of Source.String() / HasSkillMD (verified — none need changing)

- `main.go:423`: `fmt.Fprintf(stderr, "(found via %s)\n", src)` — `src` is a
  `skillsdir.Source`; `%s` invokes `Source.String()` via the fmt Stringer. Once
  `String()` returns `"config file"` for `SourceConfig`, `--path` prints
  `(found via config file)` with ZERO main.go change. **Confirmed by the
  contract DOCS §5 and by reading main.go:408-428.**
- `internal/resolve/resolve.go:50`: only MENTIONS `skillsdir.Source.String()` in a
  comment ("mirroring skillsdir.Source.String()"). No code dependency.
- `grep -rn 'SourceConfig|hasSkillMD|HasSkillMD'` outside `internal/skillsdir/`:
  **empty**. T2.S1 is fully self-contained to `internal/skillsdir/`.
- `HasSkillMD`'s consumer is P1.M2.T2.S1 (init cwd-auto-detect), which does NOT
  exist yet — nothing to update.

## 5. SCOPE BOUNDARY with the sequential sibling P1.M1.T2.S2 (load-bearing)

T2.S1 lands FIRST; T2.S2 ("Implement findConfig() rule and wire it into Find()
at priority #2; set the exact ErrNotFound message") lands SECOND and CONSUMES
`SourceConfig`.

**T2.S1 MUST NOT:**
- change the `var ErrNotFound = errors.New("…")` MESSAGE STRING. The current
  string is `"could not locate the skills directory: set $SKILLDOZER_SKILLS_DIR,
  cd into the skilldozer repo, or reinstall skilldozer"`. T2.S2 flips it to the
  PRD §8.2 `run \`skilldozer init\`` wording and updates `TestErrNotFoundMessageHasFix`.
  If T2.S1 also edits the message, the two subtasks collide.
- implement `findConfig` or wire it into `Find()`. That is T2.S2's whole job.
- touch `main.go`, `internal/config`, `README.md`, completions, or the example skill.

**T2.S1 MAY (and should):**
- update the doc-comment PROSE that says "3 rules"/"three §8 rules"/"all three"
  to the 5-rule ladder. This includes the `ErrNotFound` DOC COMMENT (line 218:
  "all three §8 rules miss") — the comment is in scope (G3), the message STRING
  is not (T2.S2). Phrasing the package/Find doc comments by Source LABEL
  (`SourceEnv`/`SourceConfig`/`SourceSibling`/`SourceWalkUp`) — not by the
  not-yet-existing `findConfig` helper — keeps the comments accurate immediately
  after T2.S1 lands AND after T2.S2 wires findConfig in.

## 6. The doc-comment "3 rules" drift (G3) — exact sites

`grep -n 'three §8\|3 rules\|all three\|rule 1\|rule 2\|rule 3' internal/skillsdir/skillsdir.go`:
- Package doc comment, lines 1-22: numbered list "1. env / 2. Sibling / 3. Walk up".
- `ErrNotFound` doc comment, line 218: "all three §8 rules miss".
- `Find()` doc comment, lines 224-231: "1. env / 2. Sibling / 3. Walk up" +
  "If all three miss …".

All three become the 5-rule ladder: env → config → sibling → walk-up →
unconfigured (ErrNotFound).

Note: the package doc comment also contains the stale parenthetical "(added in
P1.M1.T2.S3)" — there is no T2.S3 in the plan; `Find()` already exists. Since we
are rewriting that comment block, drop the parenthetical and describe `Find()` as
the existing entry point. (No subtask IDs in shipped doc comments — the existing
one is an anti-pattern to clean up while we're here.)

## 7. TestSourceString — the single test edit (G14)

`internal/skillsdir/skillsdir_test.go:57-72`:
```go
cases := []struct{ src Source; want string }{
    {SourceEnv, "SKILLDOZER_SKILLS_DIR"},
    {SourceSibling, "sibling of binary"},
    {SourceWalkUp, "ancestor of cwd"},
    {Source(-1), "unknown"},
    {Source(99), "unknown"},
}
```
Add `{SourceConfig, "config file"}` — placement after the SourceEnv row keeps the
table in priority/label order (mirrors the const block + String() switch).

## 8. Baseline + validation commands (verified working)

- `go test ./internal/skillsdir/...` → **ok** (cached, green baseline before any change).
- `go build ./...`, `go vet ./...`, `go test ./...` → the module-wide gates.
- No new deps; `go.mod`/`go.sum` byte-for-byte unchanged (T2.S1 adds zero imports).
- `gofmt -l internal/skillsdir/` → must print nothing after edits.

## 9. config package status (parallel context)

`internal/config/config.go` is ALREADY fully implemented (S1 + S2 both landed):
exports `{File, Load, Save, Path, DefaultStore}` + unexported
`configEnv = "SKILLDOZER_CONFIG"`. T2.S1 does NOT import or touch `internal/config`
— `findConfig` (which composes `config.Path()` + `config.Load()`) is T2.S2. T2.S1
only adds the `SourceConfig` enum/label that T2.S2's `findConfig` will RETURN.
