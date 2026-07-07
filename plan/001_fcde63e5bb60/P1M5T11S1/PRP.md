# PRP — P1.M5.T11.S1: `--help` text + no-args/unknown-flag/mode-exclusivity exit codes

> **Subtask:** P1.M5.T11.S1 — the CLI-surface *gate* (PRD build-order step 5).
> It finalizes `main.go` arg parsing + dispatch so the binary matches PRD
> §6.1–§6.4 exactly: a full `--help`/`-h` (mirrors mcpeepants
> USAGE/EXAMPLES/OPTIONS structure), the §6.3 no-args → usage-to-stderr + exit 1,
> the §6-header unknown-flag → `skpp: unknown flag '<x>'` + exit 2, and the §6.3
> mode-mutual-exclusivity → exit 2 (tags + `--list`/`--search`/`--all`, plus the
> `check`-subcommand extensions). `check` dispatch itself **already landed** in
> P1.M4.T10.S1; this task only adds the `--help` text, the exit-code/error
> semantics, and the exclusivity rule around it.
>
> **Scope:** MODIFY two files — `main.go` (config + parseArgs + run preamble +
> `usageText`/`usage` + `exclusivityError`) and `main_test.go` (update 3 tests,
> lightly touch 1, append ~17). **No new files, no new packages, no new deps.**
>
> **DEPENDENCIES (all LANDED & GREEN):** P1.M4.T9.S1 (`--search`) and
> P1.M4.T10.S1 (`check`, §9) are both on disk and green. `main.go` is post-T9
> (index-loop `parseArgs`, `searchMode`/`searchQ`) AND post-T10 (inline `check`
> dispatch via `internal/check.Check`). Whole module: `go test ./...` green,
> `go vet ./...` clean, `go build ./...` exit 0. See
> `research/verified_facts.md` for the line-level baseline.

---

## Goal

**Feature Goal**: Ship the complete `skpp` CLI surface (PRD §6.1–§6.4). `skpp
--help`/`-h` prints a structured USAGE/EXAMPLES/OPTIONS block to **stdout** and
exits 0, taking precedence over `--version` (the documented "help wins"
tiebreak). `skpp` with no args (or modifiers-only, no mode) prints the SAME usage
to **stderr** and exits 1 (parity with mcpeepants `get-server-config.sh`). Any
unknown dashed flag (`--bogus`/`-x`) prints `skpp: unknown flag '<x>'` to stderr
(first offender only) and exits 2. Mixing positional `<tag>`s with
`--list`/`--search`/`--all`, or `check` with tags/other modes, prints an error to
stderr and exits 2. Everything else (version/path/list/search/check/all/tags,
including §6.4 atomicity) is unchanged.

**Deliverable**: Two MODIFIED files (no new files/packages):
1. `main.go` — `config` gains `help bool` + `unknownFlag string`; `parseArgs`
   gains `--help`/`-h` and unknown-flag capture in the default branch; `run()`
   gains a reordered preamble (help→version→unknown→exclusivity) and its trailing
   default becomes `fmt.Fprint(stderr, usage()); return 1`; new `usageText`
   const + `usage()` + `exclusivityError()` helpers. The `check` branch and
   every other mode branch body stay byte-identical. ~+70 lines.
2. `main_test.go` — UPDATE 3 tests (`TestRunDefaultNoArgs`,
   `TestRunDefaultUnknownFlag`, `TestParseArgsUnknownTolerated`→renamed), LIGHTLY
   TOUCH 1 (`TestParseArgsDashedUnknownNotATag`), and APPEND ~17 new tests.

**Success Definition**: `gofmt -l main.go main_test.go` silent; `go vet ./...`
clean; `go build ./...` exit 0; `go test ./...` green (**main grows 81 → ~98**;
whole module stays green). Empirically: `./skpp --help`→stdout usage + exit 0;
`./skpp`→stderr usage + exit 1; `./skpp --bogus`→stderr `skpp: unknown flag
'--bogus'` + exit 2; `./skpp foo --list`→stderr + exit 2; `./skpp check`→unchanged
dispatch (exit 0 on a clean store). `go mod tidy` is a no-op (stdlib only). No
touch to `internal/{discover,resolve,search,skillsdir,ui,check}/*`,
`go.mod`/`go.sum`, `PRD.md`, `tasks.json`.

## User Persona

**Target User**: A pi operator who drives skills via
`pi --skill "$(skpp <tag>)"` and occasionally introspects the CLI by hand.

**Use Case**: "I forgot the exact flag for searching" → `skpp --help` shows the
full matrix on one screen. "I mistyped `--serch`" → `skpp: unknown flag '--serch'`
(exit 2) tells me immediately, instead of a silent wrong result.

**Pain Points Addressed**: silent wrong answers (unknown flag silently dropped →
exit 1, no message); no discoverable help; ambiguous combos (`foo --list`) that
silently did something surprising.

## Why

- **Closes the §6 contract.** §6.3 (no-args, precedence) and the §6 header
  (unknown ⇒ exit 2) are the only §6 behaviors still "tolerated" (silent exit 1)
  after M1–M4. This task flips them to their final, tested shape — the gate
  before packaging/docs (M6) and the README (which documents these exit codes).
- **Makes `$(...)` and scripts safe and debuggable.** Unknown flags now fail
  loudly (exit 2) instead of silently exiting 1; help is one flag away. This is
  the UX parity the PRD demands with `get-server-config.sh`.
- **Locks the precedence + exclusivity rules in code + tests** before the
  acceptance sweep (P1.M6.T16.S1) and shell completions (P1.M6.T15.S1, which
  list these exact flags) lean on them.

## What

User-visible behavior (PRD §6.1–§6.4):

- **(a) `--help`/`-h`** → print the full usage block (USAGE / EXAMPLES / OPTIONS +
  a one-line exit-code reference) to **stdout**, exit 0. Takes precedence over
  EVERYTHING, including `--version` ("help wins" tiebreak) and unknown flags.
- **(b) No args AND no mode (incl. modifiers-only, e.g. `--no-color` alone)** →
  print the SAME usage block to **stderr**, exit 1.
- **(c) Unknown dashed flag** (any token starting with `-` that is NOT in the
  known set: `--help/-h --version/-v --path/-p --list/-l --search/-s --all/-a
  --file/-f --relative --no-color`) → print `skpp: unknown flag '<x>'` to stderr
  (the FIRST such token wins), exit 2. Positional tags are unaffected.
- **(d) Mode mutual exclusivity** → `len(tags)>0 && (--list|--search|--all)` ⇒
  stderr error, exit 2 (PRD §6.3 explicit). Extended to `check`: `check` + tags,
  and `check` + a listing mode ⇒ exit 2 (modes are mutually exclusive; `check`
  ignores tags so the combo is meaningless).
- **`check` dispatch is UNCHANGED** — `case "check":` stays as-is (captured
  anywhere), the inline `check.Check(skills)` rendering + exit code stay as-is.
  See research §5 for why we keep `case "check":` (preserves
  `TestParseArgsCheckAfterFlag`) rather than switching to first-arg detection.

### Success Criteria

- [ ] `skpp --help` and `skpp -h` print USAGE/EXAMPLES/OPTIONS to **stdout**, exit 0, stderr empty.
- [ ] The help block contains the canonical `pi --skill "$(skpp <tag>)"` example and is PLAIN (no ANSI).
- [ ] `--help` beats `--version` (`skpp --help --version` → help, exit 0; NOT the version line).
- [ ] `--version` still beats unknown flag + every mode (`skpp --version --bogus` → version, exit 0).
- [ ] `--help` beats unknown flag (`skpp --help --bogus` → help, exit 0).
- [ ] `skpp` (no args) → usage to **stderr**, exit 1, stdout empty.
- [ ] `skpp --no-color` (modifiers-only, no mode) → usage to stderr, exit 1, stdout empty.
- [ ] `skpp --bogus` → `skpp: unknown flag '--bogus'` on stderr, stdout empty, exit 2.
- [ ] `skpp -x` (short unknown) → same, exit 2.
- [ ] First unknown wins: `skpp --bogus --more` → message names `--bogus`.
- [ ] `skpp foo --list` / `foo --search q` / `foo --all` → stderr error, stdout empty, exit 2.
- [ ] `skpp check foo` (check + tag) → stderr error, exit 2.
- [ ] `skpp check --list` (check + mode) → stderr error, exit 2.
- [ ] `skpp check` → UNCHANGED: dispatches to `check.Check`; exit 0 on a clean store.
- [ ] Existing behavior preserved: version/path/list/search/all/tags precedence + §6.4 atomicity unchanged.
- [ ] `go test ./...` green; `gofmt`/`go vet` clean; `go.mod` unchanged.

## All Needed Context

### Context Completeness Check

_If someone knew nothing about this codebase, would they have everything needed
to implement this successfully?_ **Yes.** The exact `config`/`parseArgs`/`run`
edits are given verbatim below (full new `parseArgs`, the precise `run` preamble
splice, full `usageText`, full `exclusivityError`). The 3 test updates + 1 light
touch + ~17 new test functions are named and specified. The `check` branch is
left byte-identical (its real signature `func Check(skills []discover.Skill)
Report` is verified on disk). Every load-bearing decision (precedence order,
keep-`case "check":`, exclusivity families, plain-text help, exit-code mapping)
is pinned in `research/verified_facts.md`.

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M5T11S1/research/verified_facts.md
  why: "§1 the 4-behavior + check-exclusivity contract table; §2 the mcpeepants help
        STRUCTURE to mirror + the PLAIN-text color decision; §3 the current
        config/parseArgs/run WITH LINE NUMBERS; §4 check's real signature (LANDED,
        no placeholder); §5 why we KEEP case \"check\": vs first-arg detection;
        §6 the 3 existing tests that MUST change (exact current assertions + line#);
        §7 the exactly-three exclusivity families; §8 verified validation commands."
  critical: "Precedence is help → version → unknown-flag → exclusivity → dispatch →
             no-args. Inserting help ABOVE version is the one reorder that could
             surprise; existing version-precedence tests use --version alone (no
             --help) so they stay green."

# MUST READ — the authoritative §6 spec (the whole point of this task)
- file: PRD.md
  section: "§6 header (unknown ⇒ exit 2), §6.1 (--help/--version/--path rows),
            §6.3 (no-args stderr+exit1; --help/--version precedence; mutual-exclusivity
            exit 2), §6.4 (stdout-empty-on-failure discipline)."
  why: "These four sub-sections ARE the contract. §6.3 line 3 ('--help / --version
        take precedence over everything else') + the conventional help-wins
        tiebreak ⇒ check help before version. READ-ONLY."
  critical: "§6.4 applies to the NEW exit-2 paths too: NOTHING on stdout when we
             exit 2 (unknown flag / exclusivity). Keep stdout empty on every
             non-success path so pi --skill \"$(skpp …)\" never sees garbage."

# MUST READ — the current main.go (the file you edit). Read it IN FULL first.
- file: main.go
  why: "config (L40-ish), parseArgs (index loop, case \"check\":" anywhere), run (L164)
        with check inline at L267–298 calling check.Check at L278, trailing
        `return 1` at L376. skillPath helper at L399. isTerminal var + version var at top."
  pattern: "run is a flat if-chain returning int; parseArgs is a `for i:=0; …` switch."
  gotcha: "Do NOT touch any mode-branch BODY (path/list/search/check/all/tags) — only
           the preamble (before c.check) and the trailing default change. The check
           branch already calls check.Check(skills) and renders the report inline."

# CONTRACT — check's real signature (LANDED; use directly, NO placeholder)
- file: internal/check/check.go
  why: "run's check branch (L267–298) already calls it. Confirm signature is unchanged:
        func Check(skills []discover.Skill) Report; Report{BySkill,Errors,Warnings};
        Report.HasErrors() bool. This task does NOT edit this file — consume only."
  gotcha: "The prior sibling PRP (P1M5T1S1) shipped a t10CheckDelegate PLACEHOLDER
           because T10 had not landed. T10 HAS landed. Do NOT reintroduce any
           placeholder — the inline check block is correct as-is."

# REFERENCE — the help-text STRUCTURE to mirror
- file: ~/projects/mcpeepants/get-server-config.sh   # the --help branch
  why: "USAGE / EXAMPLES / OPTIONS sections, aligned OPTIONS columns, a canonical
        one-liner example. skpp mirrors the STRUCTURE (not the bash/color) and swaps
        the canonical example to pi --skill \"$(skpp <tag>)\"."
  pattern: "title line → USAGE: → EXAMPLES: (canonical one-liner + a few) → OPTIONS:
            (two aligned columns, flag … one-liner)."
  gotcha: "mcpeepants colorizes help with ANSI unconditionally; skpp emits help PLAIN
           (no ANSI) unconditionally — see research §2. Do NOT add color to the help block."

# URLS — the stdlib mechanisms used (stable since Go 1.0; no version concern)
- url: https://pkg.go.dev/fmt#Fprint
  why: "fmt.Fprint(stdout, usage()) prints the help block verbatim (usageText already
        ends in \\n; Fprint adds none). fmt.Fprintf(stderr, \"skpp: unknown flag '%s'\\n\", flag)
        for the exit-2 message; fmt.Fprintln(stderr, msg) for the exclusivity message."
- url: https://pkg.go.dev/strings#HasPrefix
  why: "strings.HasPrefix(a, \"-\") distinguishes dashed tokens (unknown-flag
        candidates) from positionals (tags) in the parseArgs default branch."
```

### Current Codebase tree (relevant slice)

```bash
skpp/
├── go.mod                      # module github.com/dabstractor/skpp; go 1.25; yaml.v3 (UNCHANGED)
├── main.go                     # MODIFY: config + parseArgs + run preamble + usageText/exclusivityError
├── main_test.go                # MODIFY: 3 updated + 1 touched + ~17 appended (other 77 unchanged)
└── internal/
    ├── discover/{skill,index,discover,*_test.go}  # READ-ONLY (Index, Skill, ParseFrontmatter)
    ├── resolve/{resolve,*_test.go}                # READ-ONLY
    ├── search/{search,*_test.go}                  # READ-ONLY
    ├── skillsdir/{skillsdir,*_test.go}            # READ-ONLY (Find)
    ├── ui/{ui,*_test.go}                          # READ-ONLY
    └── check/{check,check_test.go}                # READ-ONLY (Check, Report) — LANDED, consume only
```

### Desired Codebase tree (files added/modified)

```bash
main.go          # MODIFY — config(+help,+unknownFlag) / parseArgs(+help case, +unknown capture)
                           run(preamble reorder + trailing usage→stderr) / usageText const + usage() + exclusivityError()
main_test.go     # MODIFY — 3 updated + 1 touched + ~17 appended (reuse sampleStore/writeSkillTree/withTerminal/unsetSkillsEnv)
```
No new files. No new packages. No new dependencies.

### Known Gotchas of our codebase & Library Quirks

```go
// CRITICAL — PRECEDENCE REORDER. Current run() checks c.version FIRST (L169). This
// task inserts c.help ABOVE version ("help wins" tiebreak). Every existing
// version-precedence test uses --version WITHOUT --help (TestRunVersionPrecedenceOver*
// for Path/Tag/All/Search/Check), so they stay green — but re-run them after the edit.
// Final order: help → version → unknownFlag → exclusivity → dispatch
// (check→path→list→search→all→tags) → no-args-usage.

// CRITICAL — §6.4 stdout discipline applies to the NEW exit-2 paths too. On
// unknown-flag and exclusivity, print ONLY to stderr; stdout must be EMPTY.
// (Tests assert out.Len()==0.) This keeps `pi --skill "$(skpp --bogus)"` from
// passing a garbage path — the whole point of §6.4.

// CRITICAL — KEEP `case "check":` exactly as-is (captured anywhere → c.check=true).
// Do NOT switch to first-arg detection: TestParseArgsCheckAfterFlag (--no-color check)
// would break, and exclusivity is enforced in run() anyway. A nested skill
// `writing/check` still resolves (the case matches only the EXACT token "check").

// CRITICAL — unknown-flag capture stores the FIRST offender only
// (`if c.unknownFlag == "" { c.unknownFlag = a }`). run() reports that one. Do
// NOT collect a slice; one loud error is the §6 contract.

// GOTCHA — `--search <q>` consumes args[i+1] verbatim, INCLUDING a value starting
// with '-' (e.g. `--search -x` → query "-x"). So a dashed token grabbed as a
// search value is NOT an unknown flag (it never reaches the default branch; i++ skip).
// Leave the --search case byte-identical.

// GOTCHA — help text is PLAIN (no ANSI), unconditionally. Not gated on
// isTerminal/--no-color. `skpp --help | grep` must work; tests use non-TTY buffers.

// GOTCHA — the SAME usageText is printed to stdout (--help) AND stderr (no-args).
// Use fmt.Fprint (NOT Fprintln) so there is no double trailing newline; the
// constant already ends in exactly one '\n'.

// GOTCHA — gofmt will realign the config struct comments when fields are added;
// run `gofmt -w main.go` after editing (expected, correct).

// GOTCHA — the prior sibling PRP at plan/…/P1M5T1S1/PRP.md proposed a runCheck()
// helper + t10CheckDelegate placeholder. IGNORE both: check is inline and LANDED.
// This task keeps the check branch INLINE and unchanged.
```

## Implementation Blueprint

### Data models — `config` additions

```go
// config holds the parsed CLI flags. THIS subtask (P1.M5.T11.S1) adds the final
// two fields — help, unknownFlag — completing the §6.1–§6.4 matrix.
type config struct {
	version    bool     // --version / -v
	help       bool     // --help / -h       : print usage to STDOUT, exit 0 [NEW]
	path       bool     // --path / -p
	list       bool     // --list / -l
	all        bool     // --all / -a
	file       bool     // --file / -f
	relative   bool     // --relative
	noColor    bool     // --no-color
	searchMode bool     // --search <q> / -s
	searchQ    string   // the --search value
	check      bool     // `check` subcommand (case "check": — captured anywhere)
	tags       []string // positional <tag> args
	unknownFlag string  // first unknown dashed token, "" if none [NEW]
}
```

### File 1 — MODIFY `main.go`

Apply these edits to the current `main.go`. Run `gofmt -w main.go` after.

**Edit 1a — `usageText` const + `usage()` (add near the top, after the `version`
var).** Plain text, mirrors mcpeepants USAGE/EXAMPLES/OPTIONS. Emitted PLAIN
(no ANSI) to BOTH stdout (--help) and stderr (no-args).

```go
// usageText is the full --help / no-args usage block (PRD §6.1, §6.3). It mirrors
// the STRUCTURE of mcpeepants get-server-config.sh (USAGE / EXAMPLES / OPTIONS,
// aligned columns) but lists the full skpp §6 flag matrix and the canonical
// pi --skill "$(skpp <tag>)" one-liner. It is emitted PLAIN (no ANSI):
// `skpp --help | grep` must work, §13 does not assert on help color, and tests
// use non-TTY buffers. The SAME text is printed to stdout for --help (exit 0) and
// to stderr for the no-args default (exit 1) — only the destination differs.
const usageText = `skpp — skill path printer

Resolve skill tags to on-disk skill directory paths (manifest-free).

USAGE:
  skpp <tag> [<tag>...]
  skpp --all
  skpp --list
  skpp --search <query>
  skpp check
  skpp --path
  skpp --help
  skpp --version

EXAMPLES:
  pi --skill "$(skpp example)"
  pi --skill "$(skpp writing/reddit)"
  skpp example reddit          # one absolute path per line, input order
  skpp -f example              # print the SKILL.md path
  skpp --relative --all        # every skill path, relative to the skills dir
  skpp --list                  # human-readable catalog
  skpp --search reddit         # substring search over tag/name/description/keywords
  skpp check                   # validate every skill on disk

OPTIONS:
  <tag> [<tag>...]   Resolve tags to skill directory paths (one absolute path per line)
  --all, -a          Print every skill's directory path, sorted by tag
  --list, -l         Human-readable catalog (TAG, NAME, DESCRIPTION)
  --search <q>, -s   Substring search over tag / name / description / keywords
  check              Validate every skill on disk (report OK / WARN / ERROR)
  --path, -p         Print the resolved skills directory
  --file, -f         Print the SKILL.md path instead of the directory (modifier)
  --relative         Print paths relative to the skills directory (modifier)
  --no-color         Disable ANSI color even on a TTY (modifier)
  --help, -h         Show this help message
  --version, -v      Print the skpp version

Exit codes: 0 success/help/version | 1 unresolved/no skills/unresolvable dir | 2 unknown flag / mutually-exclusive modes
`

// usage returns the help block. A tiny indirection so the constant is wrapped by
// a function (keeps the print sites uniform: fmt.Fprint(w, usage())).
func usage() string { return usageText }
```

**Edit 1b — `config` struct** (replace with the version in "Data models" above;
gofmt realigns comments). Drop any `// Future …` comment — the matrix is complete.

**Edit 1c — `parseArgs`** (layer on top of the current index loop; ADD the
`--help` case + unknown-flag capture in the default branch; leave the `--search`
case, the `case "check":`, and every other case byte-identical):

```go
func parseArgs(args []string) config {
	var c config
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--version", "-v":
			c.version = true
		case "--help", "-h": // [NEW] precedence over everything except itself
			c.help = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--all", "-a":
			c.all = true
		case "--file", "-f":
			c.file = true
		case "--relative":
			c.relative = true
		case "--no-color":
			c.noColor = true
		case "--search", "-s":
			// Value-taking flag: consume the NEXT token verbatim as the query
			// (even if it starts with '-'); i++ skips it. If --search is the LAST
			// token, searchMode stays false → falls to the no-mode default.
			if i+1 < len(args) {
				c.searchMode = true
				c.searchQ = args[i+1]
				i++
			}
		case "check":
			// `check` subcommand (PRD §6.1/§9). Captured ANYWHERE (kept as-is).
			// run()'s exclusivity check rejects check+tags / check+mode (exit 2).
			c.check = true
		default:
			// [NEW] unknown dashed flag → capture the FIRST offender for run() to
			// report as exit 2 (PRD §6 header). Non-dashed tokens are positional
			// <tag>s (captured in input order). Do NOT collect a slice of unknowns;
			// one loud error is the §6 contract.
			if strings.HasPrefix(a, "-") {
				if c.unknownFlag == "" {
					c.unknownFlag = a
				}
			} else {
				c.tags = append(c.tags, a)
			}
		}
	}
	return c
}
```

**Edit 1d — `run()` preamble splice.** Insert the `help` + `unknownFlag` +
`exclusivity` checks around the EXISTING `c.version` block. The `c.version` block
stays; only ONE line is inserted BEFORE it (the `c.help` block). After
`c.version` (and before the existing `c.path` block) insert the `unknownFlag` and
`exclusivity` blocks. **Every mode branch body (path/list/search/check/all/tags)
is byte-identical to today.** Only the trailing default changes.

```go
func run(args []string, stdout, stderr io.Writer) int {
	c := parseArgs(args)

	// 1) --help takes precedence over EVERYTHING, including --version ("help
	//    wins" tiebreak) and unknown flags (PRD §6.3). Usage to STDOUT, exit 0.
	if c.help {
		fmt.Fprint(stdout, usage())
		return 0
	}
	// 2) --version next (PRD §6.3: precedes everything except --help). UNCHANGED
	//    block — keep its body exactly as today.
	if c.version {
		fmt.Fprintf(stdout, "skpp %s\n", version)
		return 0
	}
	// 3) Unknown dashed flag → exit 2 (PRD §6 header). stdout stays EMPTY (§6.4
	//    discipline: pi --skill "$(skpp --bogus)" must fail loudly).
	if c.unknownFlag != "" {
		fmt.Fprintf(stderr, "skpp: unknown flag '%s'\n", c.unknownFlag)
		return 2
	}
	// 4) Mode mutual exclusivity → exit 2 (PRD §6.3). Checked AFTER unknown-flag
	//    so `--bogus foo --list` reports the unknown flag first (both exit 2;
	//    unknown is the more fundamental error).
	if bad, msg := exclusivityError(c); bad {
		fmt.Fprintln(stderr, msg)
		return 2
	}

	// 5) Normal mode dispatch — UNCHANGED order & bodies from today:
	//    c.check (inline check.Check) → c.path → c.list → c.searchMode → c.all →
	//    len(c.tags)>0. Leave each block byte-identical. (check is guaranteed
	//    standalone here: exclusivity caught any check+tags / check+mode above.)

	// ... existing c.path / c.list / c.searchMode / c.check / c.all / tags blocks ...

	// 6) No recognized mode → usage to STDERR, exit 1 (PRD §6.3). Parity with
	//    get-server-config.sh. Covers both truly-no-args and modifiers-only
	//    (e.g. `skpp --no-color`): if skpp was asked to DO nothing, show usage.
	//    (REPLACES the current bare `return 1` at the end of run.)
	fmt.Fprint(stderr, usage())
	return 1
}
```

**Edit 1e — `exclusivityError` helper** (new; the exactly-three families from
research §7):

```go
// exclusivityError reports whether c combines modes that PRD §6.3 forbids,
// returning a one-line stderr message. It implements EXACTLY three families:
// tags + a listing mode (PRD §6.3 explicit); check + tags; check + a listing
// mode (the check-subcommand extensions — modes are mutually exclusive and
// `check` ignores tags). Unspecified combos (e.g. --list --search, no tags) are
// left to the deterministic dispatch order, not flagged. --file/--relative/
// --no-color are MODIFIERS and never trigger exclusivity.
func exclusivityError(c config) (bad bool, msg string) {
	hasTags := len(c.tags) > 0
	if hasTags && (c.list || c.searchMode || c.all) {
		return true, "skpp: tags cannot be combined with --list/--search/--all"
	}
	if c.check && hasTags {
		return true, "skpp: 'check' cannot be combined with tag arguments"
	}
	if c.check && (c.list || c.searchMode || c.all) {
		return true, "skpp: 'check' cannot be combined with --list/--search/--all"
	}
	return false, ""
}
```

> **Do NOT** add a `runCheck` helper or any placeholder. The existing `c.check`
> branch (inline `check.Check(skills)` + render + `HasErrors()`→exit code) is
> correct and stays as-is.

### File 2 — MODIFY `main_test.go`

**Update these 3 tests** (rename one; assertions change; locations unchanged):

```go
// === L99: RENAME TestParseArgsUnknownTolerated → TestParseArgsUnknownFlagCaptured ===
func TestParseArgsUnknownFlagCaptured(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "sometag", "othertag"})
	if c.version || c.path {
		t.Errorf("version=%v path=%v; want both false", c.version, c.path)
	}
	if c.unknownFlag != "--frobnicate" {
		t.Errorf("unknownFlag=%q; want --frobnicate (first unknown captured)", c.unknownFlag)
	}
	if len(c.tags) != 2 || c.tags[0] != "sometag" || c.tags[1] != "othertag" {
		t.Errorf("tags=%v; want [sometag othertag] (positionals still captured)", c.tags)
	}
}

// === L245: TestRunDefaultNoArgs — now usage to STDERR + exit 1 ===
func TestRunDefaultNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(nil, &out, &errOut)
	if code != 1 {
		t.Errorf("run(nil): code=%d; want 1 (no-args → stderr usage, exit 1)", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(nil) stdout=%q; want EMPTY (usage goes to stderr)", out.String())
	}
	if !strings.Contains(errOut.String(), "USAGE") {
		t.Errorf("run(nil) stderr=%q; want the USAGE block", errOut.String())
	}
}

// === L253: TestRunDefaultUnknownFlag — unknown flag now exits 2 ===
func TestRunDefaultUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--frobnicate"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(--frobnicate): code=%d; want 2 (unknown flag, PRD §6)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (§6.4: nothing on stdout on exit-2)", out.String())
	}
	want := "skpp: unknown flag '--frobnicate'\n"
	if got := errOut.String(); got != want {
		t.Errorf("stderr=%q; want %q", got, want)
	}
}
```

**Lightly touch** `TestParseArgsDashedUnknownNotATag` (L390) — add one assertion
(it still passes today because it only checks tags; make the capture explicit):

```go
func TestParseArgsDashedUnknownNotATag(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "real-tag", "-x"})
	if len(c.tags) != 1 || c.tags[0] != "real-tag" {
		t.Errorf("tags=%v; want [real-tag] (dashed tokens excluded)", c.tags)
	}
	if c.unknownFlag != "--frobnicate" { // [NEW] first of the two unknowns wins
		t.Errorf("unknownFlag=%q; want --frobnicate", c.unknownFlag)
	}
}
```

**Optionally** update the comment on `TestParseArgsCheckAndTagBothCaptured`
(L1087) to: "parseArgs captures both; run() now rejects this combo (exit 2) — see
TestRunExclusivityCheckAndTags." (Assertions unchanged; it stays green.)

**Append these ~17 new tests** (reuse existing `sampleStore`/`writeSkillTree`/
`withTerminal`/`unsetSkillsEnv` helpers; the other 77 tests are unchanged):

```go
// --- parseArgs: --help/-h, first-unknown-wins, short unknown (P1.M5.T11.S1) ---

func TestParseArgsHelpLong(t *testing.T) {
	if c := parseArgs([]string{"--help"}); !c.help {
		t.Errorf("parseArgs(--help): help=false; want true")
	}
}

func TestParseArgsHelpShort(t *testing.T) {
	if c := parseArgs([]string{"-h"}); !c.help {
		t.Errorf("parseArgs(-h): help=false; want true")
	}
}

func TestParseArgsFirstUnknownWins(t *testing.T) {
	if c := parseArgs([]string{"--bogus", "--more"}); c.unknownFlag != "--bogus" {
		t.Errorf("unknownFlag=%q; want --bogus (first unknown wins)", c.unknownFlag)
	}
}

func TestParseArgsShortUnknownCaptured(t *testing.T) {
	if c := parseArgs([]string{"-x"}); c.unknownFlag != "-x" {
		t.Errorf("unknownFlag=%q; want -x", c.unknownFlag)
	}
}

// --- run: --help / -h (P1.M5.T11.S1) ---

func TestRunHelpToStdoutExit0(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--help): code=%d; want 0", code)
	}
	got := out.String()
	for _, want := range []string{"USAGE:", "EXAMPLES:", "OPTIONS:", `pi --skill "$(skpp example)"`} {
		if !strings.Contains(got, want) {
			t.Errorf("run(--help) stdout missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("help must be PLAIN (no ANSI):\n%s", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--help) stderr=%q; want empty", errOut.String())
	}
}

func TestRunHelpShortFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-h"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-h): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "USAGE:") {
		t.Errorf("run(-h) stdout missing USAGE block:\n%s", out.String())
	}
}

func TestRunHelpBeatsVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--help", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--help --version): code=%d; want 0", code)
	}
	if strings.Contains(out.String(), "skpp "+version) {
		t.Errorf("help must beat version; got the version line:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "USAGE:") {
		t.Errorf("stdout should be the help block, not the version:\n%s", out.String())
	}
}

// --- run: no-args / modifiers-only (P1.M5.T11.S1) ---

func TestRunModifiersOnlyNoMode(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--no-color"}, &out, &errOut)
	if code != 1 {
		t.Errorf("run(--no-color): code=%d; want 1 (no mode → stderr usage)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "USAGE") {
		t.Errorf("stderr=%q; want usage block", errOut.String())
	}
}

// --- run: unknown flag → exit 2 (P1.M5.T11.S1) ---

func TestRunUnknownShortFlagExit2(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-z"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(-z): code=%d; want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if got := errOut.String(); got != "skpp: unknown flag '-z'\n" {
		t.Errorf("stderr=%q; want the exact unknown-flag line", got)
	}
}

func TestRunVersionBeatsUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--version", "--bogus"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--version --bogus): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (version beats unknown flag)", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty (version won; unknown flag not reported)", errOut.String())
	}
}

func TestRunHelpBeatsUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--help", "--bogus"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--help --bogus): code=%d; want 0 (help precedence)", code)
	}
	if !strings.Contains(out.String(), "USAGE:") {
		t.Errorf("stdout should be help, not an unknown-flag error:\n%s", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty (help won)", errOut.String())
	}
}

// --- run: mode mutual exclusivity → exit 2 (P1.M5.T11.S1) ---

func TestRunExclusivityTagsAndList(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"foo", "--list"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(foo --list): code=%d; want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "cannot be combined") {
		t.Errorf("stderr=%q; want an exclusivity message", errOut.String())
	}
}

func TestRunExclusivityTagsAndSearch(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"foo", "--search", "q"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(foo --search q): code=%d; want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
}

func TestRunExclusivityTagsAndAll(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"foo", "--all"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(foo --all): code=%d; want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
}

func TestRunExclusivityCheckAndTags(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"check", "foo"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(check foo): code=%d; want 2 (check + tag)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "check") {
		t.Errorf("stderr=%q; want a message mentioning check", errOut.String())
	}
}

func TestRunExclusivityCheckAndList(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"check", "--list"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(check --list): code=%d; want 2 (check + mode)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
}
```

> **`check` dispatch is NOT re-tested here** — `TestRunCheckCleanStore`,
> `TestRunCheckEmptyStoreExit0`, etc. (from P1.M4.T10.S1) already cover it and
> stay green unchanged. This task only adds the *exclusivity* around check.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: PRECONDITION — confirm the current baseline (read research §3/§8 first)
  - COMMAND: cd /home/dustin/projects/skpp
  - COMMAND: go test ./... -count=1 >/dev/null && echo "module green"   # whole module
  - EXPECT: green. main=81 tests; check exists & wired (grep -n 'check.Check' main.go => L278).
  - COMMAND: grep -c '^func Test' main_test.go   # EXPECT: 81
  - COMMAND: grep -q 'for i := 0; i < len(args)' main.go && grep -q 'case "check":' main.go && echo "T9+T10 baseline OK"
  - COMMAND: go vet ./... && go build ./... && echo "vet+build OK"
  - IF ANY FAILS: the baseline drifted from research §3 — re-read main.go before editing.

Task 1: MODIFY main.go — usageText + usage() (Edit 1a)
  - WRITE: the usageText const + usage() func verbatim (Blueprint Edit 1a).
  - CHECK: raw string literal ends in exactly one '\n'; contains USAGE/EXAMPLES/OPTIONS
           + the canonical pi --skill "$(skpp example)" example; NO ANSI.

Task 2: MODIFY main.go — config struct (Edit 1b)
  - REPLACE the current config struct with the 2-field-added version (help/unknownFlag);
    drop any "// Future …" comment entirely (matrix complete).
  - GOTCHA: gofmt will realign comments — run `gofmt -w main.go` after.

Task 3: MODIFY main.go — parseArgs (Edit 1c)
  - ADD: `case "--help","-h":`; unknown-flag capture in the default branch. Leave the
         --search case, `case "check":`, and every other case byte-identical.
  - CHECK: `skpp --help` → c.help true; `skpp --bogus` → c.unknownFlag="--bogus";
           `skpp --search -x` → searchQ="-x" (unchanged); `skpp check` → c.check true.

Task 4: MODIFY main.go — run() preamble splice + exclusivityError + trailing usage (Edits 1d/1e)
  - INSERT the c.help block BEFORE the existing c.version block (Edit 1d step 1).
  - INSERT the c.unknownFlag + exclusivity blocks AFTER c.version, BEFORE c.path.
  - CHANGE the trailing bare `return 1` (current end of run) to `fmt.Fprint(stderr, usage()); return 1`.
  - ADD exclusivityError (3 families exactly — Edit 1e).
  - KEEP every mode-branch BODY (path/list/search/check/all/tags) byte-identical.
  - GOTCHA: do NOT add runCheck or any placeholder — the check branch is correct as-is.

Task 5: MODIFY main_test.go — 3 updates + 1 touch + ~17 appends (File 2)
  - RENAME+UPDATE TestParseArgsUnknownTolerated → TestParseArgsUnknownFlagCaptured (L99).
  - UPDATE TestRunDefaultNoArgs (L245) + TestRunDefaultUnknownFlag (L253).
  - TOUCH TestParseArgsDashedUnknownNotATag (L390) with the unknownFlag assertion.
  - APPEND the ~17 tests from File 2 (help, no-args/modifiers-only, unknown, exclusivity).
  - CHECK: reuses sampleStore/writeSkillTree/withTerminal/unsetSkillsEnv; the other 77 tests UNCHANGED.
  - GOTCHA: NO t.Parallel() on env/cwd tests (repo convention).

Task 6: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w main.go main_test.go
  - COMMAND: gofmt -l main.go main_test.go   # MUST print nothing
  - COMMAND: go vet ./...                    # MUST be clean
  - COMMAND: go mod tidy   # EXPECTED: a NO-OP (stdlib only; no new module)
  - COMMAND: go build ./...                  # exit 0
  - COMMAND: go test . -v                    # ~98 main tests (81 + ~17)
  - COMMAND: go test ./...                   # whole module green
  - EXPECT: zero errors, zero vet findings, gofmt silent, go.mod/go.sum unchanged.

Task 7: SMOKE + SCOPE CHECK — Levels 3 & 4 in the Validation Loop.
```

### Implementation Patterns & Key Details

```go
// PATTERN: precedence tier (help highest; "help wins" tiebreak).
//   if c.help              { fmt.Fprint(stdout, usage()); return 0 }   // STDOUT, exit 0
//   if c.version           { fmt.Fprintf(stdout, "skpp %s\n", version); return 0 }
//   if c.unknownFlag != "" { fmt.Fprintf(stderr, "skpp: unknown flag '%s'\n", …); return 2 }
//   if bad, _ := exclusivityError(c); bad { fmt.Fprintln(stderr, msg); return 2 }
//   …dispatch… ; fmt.Fprint(stderr, usage()); return 1   // no-args → STDERR, exit 1
// WHY: PRD §6.3 ("--help/--version take precedence") + help-wins tiebreak. The
//      unknown-flag + exclusivity checks sit BELOW help/version (so --help/--version
//      still win) but ABOVE dispatch (so a bad combo never partially runs).

// PATTERN: same usage text, two destinations (--help→stdout exit0; no-args→stderr exit1).
//   const usageText = `…`                  // plain; ends in one '\n'
//   fmt.Fprint(stdout, usage())   // --help
//   fmt.Fprint(stderr, usage())   // no-args / modifiers-only
// WHY: PRD §6.3 parity with get-server-config.sh. fmt.Fprint (no extra '\n').

// PATTERN: keep `case "check":` (captured anywhere); enforce exclusivity in run().
//   case "check": c.check = true   // parseArgs — unchanged
//   if c.check && hasTags          { return exit2 }   // run — new
// WHY: preserves TestParseArgsCheckAfterFlag (--no-color check) and avoids loop-index
//      juggling. A nested skill writing/check still resolves (case matches only "check").

// PATTERN: unknown-flag capture = FIRST offender only, in the default branch.
//   default:
//     if strings.HasPrefix(a, "-") {
//         if c.unknownFlag == "" { c.unknownFlag = a }   // first wins
//     } else { c.tags = append(c.tags, a) }
// WHY: one loud error (PRD §6). --search's consumed value never reaches default (i++ skip).
```

### Integration Points

```yaml
CLI SURFACE (final, §6.1–§6.4):
  - flags: --help/-h [NEW], --version/-v, --path/-p, --list/-l, --search/-s,
           --all/-a, --file/-f, --relative, --no-color
  - subcommand: check (case "check": — captured anywhere; exclusivity-enforced in run)
  - exit codes: 0 success/help/version | 1 no-args/modifiers-only/no-skills/unresolved/unresolvable
                | 2 unknown flag / mutually-exclusive modes
  - stdout discipline (§6.4): EMPTY on every non-success path (incl. the new exit-2s)

CONFIG:
  - struct main.config gains: help bool; unknownFlag string

MAIN DISPATCH (run preamble order):
  - file: main.go
  - order: help → version → unknownFlag → exclusivity → (check→path→list→search→all→tags) → no-args-usage
  - new helpers: usage()/usageText, exclusivityError()
  - check branch: UNCHANGED (inline check.Check + render + HasErrors()→exit code)

DEPENDENCIES (go.mod): UNCHANGED. stdlib only (fmt/io/os/strings/path/filepath). go mod tidy is a no-op.

NO CHANGES TO:
  - internal/{discover,resolve,search,skillsdir,ui}/* ; internal/check/* (consume only)
  - go.mod / go.sum / PRD.md / tasks.json ; skills/ ; install.sh ; README.md ; completions
```

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
cd /home/dustin/projects/skpp
gofmt -w main.go main_test.go
gofmt -l main.go main_test.go   # MUST print nothing
go vet ./...                    # MUST be clean
# go mod tidy   # OPTIONAL sanity — EXPECTED no-op (diff go.mod before/after)
go build ./...                  # exit 0
# Expected: zero gofmt output; zero vet findings; build succeeds.
```

### Level 2: Unit Tests (Component Validation)

```bash
# The new parseArgs + run branches — fastest feedback.
go test . -run 'Help|NoArgs|ModifiersOnly|UnknownFlag|UnknownShortFlag|VersionBeatsUnknown|HelpBeatsUnknown|HelpBeatsVersion|Exclusivity|DefaultNoArgs|DefaultUnknownFlag|UnknownFlagCaptured|FirstUnknownWins|ShortUnknownCaptured' -v
# Expected: all the new + updated tests PASS.

# Full main suite — confirms the 4 updated tests + no regression in the other 77.
go test . -v
# Expected: ~98 main tests PASS (81 baseline + ~17 new; 3 updated + 1 touched in place).

# Whole module — confirms the read-only packages are untouched (incl. check).
go test ./... -count=1
# Expected: ALL PASS.
```

### Level 3: Integration Testing (System Validation)

```bash
cd /home/dustin/projects/skpp
go build -o skpp . && echo OK

# 1) --help → STDOUT usage, exit 0, stderr empty, NO ANSI.
out=$(./skpp --help 2>err.txt); rc=$?
[ "$rc" = "0" ] && echo "$out" | grep -q 'USAGE:' && echo "$out" | grep -q 'EXAMPLES:' \
  && echo "$out" | grep -q 'OPTIONS:' && echo "$out" | grep -qF 'pi --skill "$(skpp example)"' \
  && [ ! -s err.txt ] && ! printf '%s' "$out" | grep -q $'\x1b' && echo "HELP OK"

# 2) -h short form works identically.
./skpp -h | grep -q 'USAGE:' && echo "HELP-SHORT OK"

# 3) No args → STDERR usage, exit 1, stdout empty.
out=$(./skpp 2>err.txt); rc=$?
[ "$rc" = "1" ] && [ -z "$out" ] && grep -q 'USAGE:' err.txt && echo "NOARGS OK"

# 4) Unknown flag → exit 2, exact stderr, empty stdout.
out=$(./skpp --bogus 2>err.txt); rc=$?
[ "$rc" = "2" ] && [ -z "$out" ] && [ "$(cat err.txt)" = "skpp: unknown flag '--bogus'" ] && echo "UNKNOWN OK"
./skpp -x >/dev/null 2>&1; [ "$?" = "2" ] && echo "UNKNOWN-SHORT OK"

# 5) Precedence: --help beats --version; --version beats unknown flag.
./skpp --help --version | grep -q 'USAGE:' && ! ./skpp --help --version | grep -q "$(./skpp --version)" && echo "HELP>VERSION OK"
./skpp --version --bogus | grep -q "$(./skpp --version)" && echo "VERSION>UNKNOWN OK"

# 6) Mode mutual exclusivity → exit 2, empty stdout.
for combo in "foo --list" "foo --search q" "foo --all" "check foo" "check --list"; do
  ./skpp $combo >/dev/null 2>err.txt; rc=$?
  [ "$rc" = "2" ] && echo "EXCLUSIVITY OK: $combo" || echo "EXCLUSIVITY FAIL: $combo (rc=$rc)"
done

# 7) Regression: existing modes still work (sanity).
./skpp --version >/dev/null && ./skpp --path >/dev/null && echo "REGRESSION OK"
./skpp check >/dev/null 2>&1; [ "$?" = "0" ] && echo "CHECK-UNCHANGED OK"

# 8) go.mod unchanged (scope check).
git diff --quiet go.mod go.sum && echo "DEPS-UNCHANGED OK"
# Expected: all lines print "… OK"; nothing prints FAIL.
```

### Level 4: Creative & Domain-Specific Validation

```bash
cd /home/dustin/projects/skpp
# End-to-end with pi (skills load ONLY via --skill, not auto-discovered — PRD §13).
# Requires the example skill to exist; if P1.M6.T12.S1 hasn't landed, create a
# throwaway skills/example/SKILL.md first (out of scope for THIS task):
mkdir -p skills/example && printf -- '---\nname: example\ndescription: demo\n---\n# x\n' > skills/example/SKILL.md
pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
rm -rf skills/example   # cleanup if you created it (P1.M6 owns the real example)
# Expected: pi references the example skill and does not error (§6.4 contract intact:
# $(skpp example) emitted a clean absolute path).
```

## Final Validation Checklist

### Technical Validation

- [ ] All validation levels completed successfully.
- [ ] All tests pass: `go test ./... -count=1` (main ~98, whole module green).
- [ ] No lint/format errors: `gofmt -l main.go main_test.go` silent.
- [ ] No vet errors: `go vet ./...` clean.
- [ ] `go build ./...` exit 0.
- [ ] `go.mod`/`go.sum` unchanged (`git diff --quiet go.mod go.sum`).

### Feature Validation

- [ ] All success criteria from "What" section met.
- [ ] Manual smoke from Level 3 all print "… OK".
- [ ] Error cases handled: unknown flag → exit 2; exclusivity → exit 2; no-args → exit 1 (stderr usage).
- [ ] §6.4 stdout discipline: stdout EMPTY on every non-success path (incl. exit-2s).
- [ ] `check` dispatch unchanged (exit 0 on a clean store).

### Code Quality Validation

- [ ] Follows existing codebase patterns (index-loop parseArgs; flat if-chain `run`; verbatim stderr from typed errors).
- [ ] File placement matches the tree (only main.go + main_test.go touched).
- [ ] Anti-patterns avoided: no runCheck placeholder, no first-arg-detection churn, no help color, no slice of unknowns.
- [ ] No new dependencies.

### Documentation & Deployment

- [ ] Help block is self-documenting (USAGE/EXAMPLES/OPTIONS + exit-code line).
- [ ] Exit codes documented inline in help text (consumed by P1.M6 README + completions).
- [ ] No new env vars.

---

## Anti-Patterns to Avoid

- ❌ Don't reintroduce the `t10CheckDelegate` placeholder or a `runCheck` helper — `check` is inline and LANDED.
- ❌ Don't switch `check` to first-arg detection — it breaks `TestParseArgsCheckAfterFlag` and adds churn for no gain (exclusivity in `run()` is sufficient).
- ❌ Don't gate `--help` color on `isTerminal`/`--no-color` — help is PLAIN, unconditionally.
- ❌ Don't collect a slice of unknown flags — report only the FIRST offender.
- ❌ Don't touch any mode-branch body (path/list/search/check/all/tags) — only the preamble + trailing default change.
- ❌ Don't flag mode+mode-without-tags (e.g. `--list --search q`) — PRD §6.3 scopes exclusivity to tag+mode; leave those to deterministic dispatch.
- ❌ Don't skip validation because "it should work" — run all 4 levels.
- ❌ Don't modify `PRD.md`, `tasks.json`, `go.mod`/`go.sum`, or anything under `internal/`.

---

## Confidence Score

**9/10** for one-pass implementation success. The task is a tight, well-scoped
edit to two files against a green, fully-landed baseline. Every edit is given
verbatim; every test to update/append is named; the one prior-art PRP (P1M5T1S1)
got the contract ~90% right and is corrected here only for the post-T10 reality
(check LANDED → no placeholder; keep `case "check":` → less churn; 81-test
baseline → ~98). The residual 1/10 is the `--help`-beats-`--version` tiebreak
being a documented choice rather than a PRD-mandated one (defensible and tested;
flip the two lines in `run()` + one test if the reviewer disagrees).

## Research Artifacts

- `research/verified_facts.md` — line-level baseline, the 4 must-change tests, the
  check-handling decision (research §5), the exactly-three exclusivity families
  (§7), verified validation commands (§8).
