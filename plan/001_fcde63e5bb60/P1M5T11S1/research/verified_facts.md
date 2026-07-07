# Verified facts — P1.M5.T11.S1 (`--help` + no-args/unknown-flag/mode-exclusivity exit codes)

> All facts read directly from the working tree at `/home/dustin/projects/skpp` on
> 2026-07-07. Every quoted fragment is the CURRENT on-disk state. Go toolchain:
> `go1.26.4-X:nodwarf5 linux/amd64`. This supersedes `P1M5T1S1/research/verified_facts.md`,
> which was authored against an OLDER baseline (post-T9, pre-T10, 69 main tests).
> **The current baseline is post-T9 AND post-T10: 81 main tests, `internal/check`
> exists and is wired inline in `run()`.**

## 1. What this subtask owns (contract from the plan + PRD §6)

The CLI-surface *gate* (build-order step 5). Four behaviors, each with a pinned
stdout/stderr/exit-code discipline. (`check` dispatch already landed in T10.S1;
this task only adds the **exclusivity** rule around it + `--help`.)

| Input                                  | stdout            | stderr                       | exit | PRD        |
| -------------------------------------- | ----------------- | ---------------------------- | ---- | ---------- |
| `skpp --help` / `-h`                   | full usage        | (empty)                      | 0    | §6.1, §6.3 |
| `skpp` (no args / no mode)             | (empty)           | usage                        | 1    | §6.3       |
| `skpp --bogus` (unknown dashed token)  | (empty)           | `skpp: unknown flag '<x>'`   | 2    | §6 header |
| `skpp foo --list` (tags + a mode)      | (empty)           | error line                   | 2    | §6.3       |
| `skpp check foo` (check + tag)         | (empty)           | error line                   | 2    | §6.3 (ext) |
| `skpp check` (subcommand)              | check report      | (empty unless Find err)      | 0/1  | §6.1, §9   |

**Precedence decision (help-wins tiebreak):** `--help` beats `--version` beats
unknown-flag beats exclusivity beats normal dispatch beats no-args. PRD §6.3
("--help / --version take precedence over everything else") lists them as a tier;
the conventional + defensible tiebreak is help-before-version (most CLIs honor
`--help` as the highest), so check `c.help` BEFORE `c.version` in `run()`.

## 2. mcpeepants `get-server-config.sh` help-text STRUCTURE (the thing to mirror)

Read in full at `~/projects/mcpeepants/get-server-config.sh` (the `--help`
branch). It prints, in order: title + one-line desc → **USAGE:** → **EXAMPLES:**
(canonical `claude --mcp-config "$($0 …)"` one-liner + a few) → **OPTIONS:**
(two aligned columns, flag … one-liner). skpp mirrors the STRUCTURE and swaps the
canonical example to `pi --skill "$(skpp example)"`.

**Color decision: help is PLAIN (no ANSI), unconditionally.** Four reasons:
(a) `skpp --help | grep` must work; (b) §13 acceptance never asserts on help
color; (c) tests use non-TTY `*bytes.Buffer` so color would be noise; (d) the
`--no-color` flag is a *modifier* for the path/table emitters, not a help-text
knob. Do NOT gate help color on `isTerminal`/`--no-color`.

## 3. Current `main.go` (read in full; 410 lines)

`config` struct fields (parseArgs output), IN ORDER:
`version, path, list, all, file, relative, noColor, tags, searchMode, searchQ,
check`. **MISSING (this task adds):** `help bool`, `unknownFlag string`.

`parseArgs` (index loop `for i := 0; i < len(args); i++`): a `case` per known
flag; `--search/-s` consumes `args[i+1]` + `i++`; `case "check":` sets `c.check`
(**captured anywhere**, not first-arg-only); the `default` branch is:
```go
default:
    if !strings.HasPrefix(a, "-") {
        c.tags = append(c.tags, a)
    }
    // (dashed unknowns are SILENTLY DROPPED today — this task captures them)
```

`run()` (L164) precedence, CURRENT:
`c.version`(L169) → `c.path`(L174) → `c.list`(L189) → `c.searchMode`(L227) →
`c.check`(L267, calls `check.Check(skills)` at L278, renders inline) → `c.all`(L308)
→ `len(c.tags)>0`(L327) → `return 1`(L376, **silent** no-args default).

**This task REORDERS the preamble to:**
`c.help` → `c.version` → `c.unknownFlag` → `exclusivityError(c)` →
`c.check` → `c.path` → `c.list` → `c.searchMode` → `c.all` → tags → no-args-usage.
Every mode-branch BODY stays byte-identical; only the preamble grows and the
trailing `return 1` becomes `fmt.Fprint(stderr, usage()); return 1`.

## 4. `check` is LANDED — use the real signature, NO placeholder

`internal/check/check.go` exports (read in full):
```go
func Check(skills []discover.Skill) Report
type Report struct { BySkill []SkillReport; Errors int; Warnings int }
func (r Report) HasErrors() bool
type SkillReport struct { Skill discover.Skill; Findings []Finding }
type Finding struct { Level Severity; Message string }
// Severity.String() => "OK" | "WARN" | "ERROR"
```
`run()` ALREADY renders it inline (L267–298): one `OK <relTag> (<name>)` line per
clean skill, one `ERROR/WARN …: <msg>` line per finding, then
`"%d skills, %d errors, %d warnings"`. Exit 0 clean / 1 if `rep.HasErrors()`.
**Keep this block INLINE** (do not extract `runCheck`; minimal diff, tests pass).

## 5. `check` handling decision: KEEP `case "check":` (anywhere) + enforce exclusivity in `run()`

The sibling PRP (`P1M5T1S1`) proposed first-arg detection
(`if args[0]=="check" { start=1 }`). **We REJECT that** for the current baseline:
the on-disk tests `TestParseArgsCheckAfterFlag` (`--no-color check` → check works)
and `TestParseArgsCheckSubcommand` assert check-is-recognized-anywhere, and
first-arg detection would BREAK them (and add loop-index juggling). Instead:

- Keep `case "check":` exactly as-is (captured anywhere → `c.check=true`).
- Add an `exclusivityError(c)` check in `run()` that rejects check+tags and
  check+mode combos (→ exit 2). parseArgs still captures both; `run()` enforces.
- This keeps `TestParseArgsCheckSubcommand`, `TestParseArgsCheckAfterFlag`, AND
  `TestParseArgsCheckAndTagBothCaptured` GREEN (they test parseArgs capture, not
  run dispatch). Only a run-level exclusivity test (`TestRunExclusivityCheckAndTags`)
  is ADDED; the existing parseArgs test's *comment* ("M5 makes this exit 2") is now
  satisfied at the run layer.

A top-level skill literally tagged `check` stays unresolvable via `skpp check`
(reserved subcommand word) — already documented in the current `case "check":`
comment. A nested skill `writing/check` still resolves: `case "check":` matches
only the EXACT token `check`, not the slash-containing token `writing/check`.

## 6. The 4 existing tests that MUST change (exact current assertions)

```go
// L99 TestParseArgsUnknownTolerated  — rename TestParseArgsUnknownFlagCaptured
c := parseArgs([]string{"--frobnicate", "sometag", "othertag"})
// currently: asserts version/path false + tags=[sometag othertag].
// ADD: assert c.unknownFlag == "--frobnicate" (first unknown captured).

// L245 TestRunDefaultNoArgs  — currently: code==1 only.
// CHANGE: also assert errOut contains "USAGE:" AND out.Len()==0.

// L253 TestRunDefaultUnknownFlag  — currently: code==1 ("exit-2 is M5").
// CHANGE: code==2; out.Len()==0; errOut == "skpp: unknown flag '--frobnicate'\n".

// L390 TestParseArgsDashedUnknownNotATag  — currently: tags=[real-tag].
// OPTIONAL ADD: assert c.unknownFlag == "--frobnicate" (first of two unknowns).
```
`TestParseArgsCheckAndTagBothCaptured` (L1089) and `TestRunTagStillResolvesAlongsideCheck`
(L1266) STAY GREEN unchanged (parseArgs captures both; single-tag-no-check resolves).
Update the L1087 comment only (run now enforces exit 2).

## 7. The exactly-three exclusivity families (PRD §6.3 + check extension)

```go
hasTags := len(c.tags) > 0
if hasTags && (c.list || c.searchMode || c.all) { exit2 }   // PRD §6.3 explicit
if c.check && hasTags                                       { exit2 }   // check ignores tags
if c.check && (c.list || c.searchMode || c.all)            { exit2 }   // modes mutually exclusive
```
NOT flagged (deliberate, PRD §6.3 scopes exclusivity to tag+mode): mode+mode
without tags (e.g. `--list --search q`, `--list --all`) → deterministic dispatch
(list wins today). `--file/--relative/--no-color` are MODIFIERS, never trigger.

## 8. Verified validation commands (run from repo root)

```bash
go test ./... -count=1        # whole module green (main=81, check=18, +others)
go vet ./...                  # clean
go build ./...                # exit 0
gofmt -l main.go main_test.go # silent (run `gofmt -w` after edits)
```
Baseline whole-module = main(81) + check(18) + discover + resolve + search +
skillsdir + ui, all green today. After this task: main grows 81 → ~98 (4 updated,
~17 appended; none removed).
