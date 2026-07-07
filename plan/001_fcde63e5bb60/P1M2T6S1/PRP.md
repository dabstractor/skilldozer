# PRP — P1.M2.T6.S1: `internal/ui` `PrintList` table + ANSI/`--no-color` + `--list` wiring

> **Subtask:** P1.M2.T6.S1 — the human-readable catalog table for `skpp --list`
> (PRD §6.1). It is the ONLY subtask of T6 ("`--list` catalog output, internal/ui
> package"). It creates the `internal/ui` package, renders the `TAG`/`NAME`/
> `DESCRIPTION` table with ANSI color gated on TTY + `--no-color`, and is the
> FIRST place the full `skillsdir.Find() → discover.Index() → ui` data flow is
> wired end-to-end in `main.go`.
>
> **Scope:** four files — CREATE `internal/ui/ui.go` + `internal/ui/ui_test.go`;
> MODIFY `main.go` (add `list`/`noColor` to `config`, the `--list`/`-l` and
> `--no-color` parseArgs cases, the `isTerminal` TTY check, and the `run` `--list`
> branch) and `main_test.go` (add `io`/`os` imports, `withTerminal`/`writeSkillTree`
> helpers, and 8 new `--list`/`--no-color` tests; existing 15 tests unchanged).
>
> **DEPENDENCIES (CONTRACT):** P1.M1.T3.S1 (`main.go`/`main_test.go`) and
> P1.M2.T5.S1 (`discover.Index`) are LANDED and GREEN — both consumed verbatim.
> `discover.Index(absSkillsDir) ([]Skill, error)` returns the sorted catalog;
> `skillsdir.Find() (dir, src, err)` resolves the store. Both are READ-ONLY — do
> not modify them.
>
> **SCOPE DECISION (authoritative — see research/verified_facts.md):** This
> subtask owns `--list`/`-l` + `--no-color` ONLY. It does NOT add `--search` (T9),
> `--all`/`<tag>` (M3), `--help` (M5), the `check` subcommand, or any §6.3
> mutual-exclusivity / exit-2 logic (those remain the tolerated default). It does
> NOT touch `internal/discover/*` or `internal/skillsdir/*`, does NOT create
> `skills/example/` (P1.M6.T12), and adds NO third-party dependency (`go.mod`/
> `go.sum` unchanged — color uses raw ANSI byte strings + stdlib TTY detection).
>
> **NOTE:** `--search` (P1.M4.T9) will REUSE `ui.PrintList` with a filtered slice
> (PRD §6.1: "same table format as `--list`, filtered"). So `PrintList` is designed
> as a pure formatter over `[]discover.Skill` with color opt-in via a `useColor`
> param — no terminal inspection inside the package.

---

## Goal

**Feature Goal**: Ship the human-readable catalog (`skpp --list` / `-l`) so a user
can see every skill in their store as a `TAG`/`NAME`/`DESCRIPTION` table, with
ANSI color when stdout is a TTY (and `--no-color` to disable it). This wires the
first complete read path: `main` resolves the store via `skillsdir.Find()`, builds
the index via `discover.Index()`, and renders it via `ui.PrintList`. The table
honors the missing-data placeholders a manifest-free catalog needs: `NAME` shows
`(none)` when the frontmatter name is empty; `DESCRIPTION` shows `(missing)` when
the SKILL.md had no frontmatter block (`HasFM==false`) or an empty description.
Long descriptions wrap at a fixed width; columns align across rows; output is plain
text (no ANSI) for non-TTYs so `$(...)`, pipes, and tests are deterministic.

**Deliverable**: Four files (two NEW, two MODIFIED; no other files touched):
1. `internal/ui/ui.go` — `package ui`; `func PrintList(w io.Writer, skills
   []discover.Skill, useColor bool)` + the `padRight`/`wrapWords` helpers + ANSI
   constants. Imports only `fmt`/`io`/`strings`/`internal/discover`.
2. `internal/ui/ui_test.go` — `package ui` (white-box); 11 tests covering empty
   input, color on/off, `(none)`/`(missing)` placeholders, folded-newline trim,
   wrapping, input-order preservation, and cross-row column alignment.
3. `main.go` — MODIFY: `config` gains `list`/`noColor`; `parseArgs` gains
   `case "--list","-l"` + `case "--no-color"`; new package var `isTerminal`; `run`
   gains the `if c.list { ... }` branch. Imports gain `internal/discover` + `internal/ui`.
4. `main_test.go` — MODIFY: add `io`/`os` imports, `withTerminal` + `writeSkillTree`
   helpers, and 8 new tests (parseArgs `--list`/`-l`/`--no-color`; `run --list`
   success / short / no-skills-exit-1 / unresolvable-exit-1 / `--no-color`
   suppresses ANSI / TTY enables ANSI). Existing 15 tests UNCHANGED.

**Success Definition**: `gofmt -l internal/ui/*.go main.go main_test.go` is silent;
`go vet ./...` is clean; `go build ./...` and `go test ./...` pass (94 tests total:
31 discover + 29 skillsdir + 11 ui NEW + 23 main). `go mod tidy` is a **no-op**
(`go.mod`/`go.sum` unchanged). `./skpp --list` over a store prints the aligned table
to stdout, exit 0; over an empty store exits 1; over an unresolvable store exits 1
with the one-line fix on stderr and nothing on stdout. `--no-color` suppresses ANSI
even on a TTY. No touch to `internal/discover/*`, `internal/skillsdir/*`,
`go.mod`/`go.sum`, `PRD.md`; no `skills/`, no `--search`/`--all`/`check`/`--help`.

---

## Why

- This subtask **makes the catalog visible.** discover.Index builds the `[]Skill`;
  **T6 is the first thing that shows it to a human** (`--list`). Until now, the
  index was only observable via `go test`.
- It **wires the first end-to-end read path.** T5 explicitly deferred
  `skillsdir.Find() → discover.Index()` ("NOT added in this subtask"). T6 is where
  that data flow lands in `main.run`, so `./skpp --list` is the first real
  end-to-end CLI behavior beyond `--path`/`--version`.
- It **establishes the `ui` package** that `--search` (T9) reuses (PRD §6.1: "same
  table format as `--list`"). Designing `PrintList` as a pure formatter over
  `[]discover.Skill` with color opt-in means T9 is a thin filter + a second
  `PrintList` call, not a re-implementation.
- It **locks the color/TTY contract** (PRD §6.2): color is on for a TTY unless
  `--no-color`. The decision is stdlib-only (`os.ModeCharDevice`, no `golang.org/x/term`)
  and isolated in `main` (`isTerminal`), so the `ui` package stays deterministic
  and the dep surface stays at exactly one (`yaml.v3`, PRD §4/§7.3).
- It **surfaces missing data gracefully** — a manifest-free catalog WILL have
  no-frontmatter / empty-name skills (discover includes them as `HasFM=false`); the
  `(none)`/`(missing)` placeholders keep the table readable without lying.

---

## What

One new package + two `main` edits:

**`package ui`** — `func PrintList(w io.Writer, skills []discover.Skill, useColor bool)`:
- Empty `skills` → prints nothing (main guards `len==0` and exits 1 first; this is
  defensive).
- Computes column widths from the PLAIN content (independent of color): TAG width =
  max(`len("TAG")`, max `RelTag`); NAME width = max(`len("NAME")`, max display name).
- Per row: TAG = `RelTag`; NAME = `Name` or `"(none)"`; DESCRIPTION = trimmed
  `Description` or `"(missing)"` when `!HasFM || desc==""`.
- Header row: bold `TAG`/`NAME`/`DESCRIPTION`. Data rows: TAG cyan; NAME/DESCRIPTION
  default. Continuation (wrapped) lines: blank TAG/NAME cells.
- DESCRIPTION word-wrapped to `descWrapWidth = 40` (fixed; terminal-width detection
  would need a forbidden dep — see Gotcha #5).
- `useColor=false` → plain text (no `\x1b` bytes). Always applies `padRight` to the
  PLAIN string BEFORE painting, so escape bytes never corrupt the padding math.

**`main.go`** — the `--list` branch in `run`:
- `skillsdir.Find()` → on `ErrNotFound`, `Fprintln(stderr, err)` + exit 1.
- `discover.Index(dir)` → on error, `Fprintln(stderr, err)` + exit 1.
- `len(skills)==0` → `"no skills found in "+dir` to stderr + exit 1 (PRD §6.1).
- `ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)` + exit 0.

### Success Criteria

- [ ] `internal/ui/ui.go` is `package ui` with EXACTLY: `PrintList`, `padRight`,
      `wrapWords`, and the 3 ANSI `const` (`ansiReset`/`ansiBold`/`ansiCyan`) +
      `descWrapWidth`. Imports ONLY `fmt`, `io`, `strings`, `internal/discover`.
- [ ] `PrintList` on a single skill (`example`/`example`/`A demo skill.`, `useColor=false`)
      prints a header line (`TAG…NAME…DESCRIPTION`) and a data line containing the
      tag, name, and description, with NO `\x1b` bytes.
- [ ] `PrintList(..., useColor=true)` output contains `\x1b[1m`, `\x1b[36m`, `\x1b[0m`;
      `useColor=false` output contains none.
- [ ] Empty `Name` → cell shows `(none)`; `!HasFM` or empty `Description` → cell
      shows `(missing)`.
- [ ] A folded-scalar `Description` with a trailing `\n` does NOT create a blank
      line (`TrimSpace` applied; no `"\n\n"` in output).
- [ ] A long description wraps to multiple lines, each ≤ `descWrapWidth` chars beyond
      the description column; continuation lines are indented under the column.
- [ ] Column alignment holds across rows of differing tag/name length (every
      description starts at the same byte column as the `DESCRIPTION` header).
- [ ] `PrintList` preserves INPUT order (does NOT re-sort — discover.Index already
      sorted; ui is a pure renderer).
- [ ] `main.go` `config` has `list bool` + `noColor bool`; `parseArgs` recognizes
      `--list`/`-l` and `--no-color`; `run` has the `if c.list { ... }` branch AFTER
      `--version`/`--path` and BEFORE the default `return 1`.
- [ ] `./skpp --list` over a 1-skill store → exit 0, table on stdout; over an EMPTY
      store → exit 1, empty stdout, `"no skills found"` on stderr; over an
      unresolvable store → exit 1, empty stdout, the one-line fix on stderr.
- [ ] `--no-color` suppresses ANSI even when `isTerminal(stdout)` is true; TTY +
      no `--no-color` emits ANSI (both via the `isTerminal` package var, tested
      with the `withTerminal` override).
- [ ] `gofmt -l` silent; `go vet ./...` clean; `go build ./...` + `go test ./...`
      pass (94 tests); `go mod tidy` no-op; `go.mod`/`go.sum` unchanged; discover/
      skillsdir files untouched.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for all four files is given verbatim in the Implementation
Blueprint (gofmt-clean, compiles, and all 27 NEW/affected tests pass — the source
was compiled and run against a verbatim copy of the real module in a throwaway
`/tmp/skpp_ui_verify` during research on go1.26.4). Every load-bearing decision was
empirically verified (`research/verified_facts.md`): the stdlib-only TTY detection
(`os.ModeCharDevice`) and why `*bytes.Buffer` → not-a-TTY (§2); the pad-then-paint
ANSI alignment math (§3); the `(none)`/`(missing)` column rules (§4); fixed-width
wrapping (§5); the rendered output (§6); the end-to-end `Find→Index→ui` wiring and
the reachable empty-store exit-1 path (§7); the `--no-color`-in-scope / `--search`-
deferred boundary (§8); the exact file scope (§9); test counts (§10). The consumed
contracts — `discover.Index`/`discover.Skill` and `skillsdir.Find` — were read in
full on disk. An implementer who knows Go but nothing about this repo can complete
this in one pass from this document._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M2T6S1/research/verified_facts.md
  why: "Proves (against a verbatim module copy on go1.26.4): (1) ui.PrintList is a
        pure leaf formatter — color is opt-in via useColor, the package NEVER
        inspects the terminal. (2) TTY detection is stdlib os.ModeCharDevice (NO
        golang.org/x/term — PRD §4 keeps yaml.v3 the only dep); a *bytes.Buffer
        fails the *os.File type assertion -> false -> no color in tests by default.
        (3) ANSI padding: padRight the PLAIN string BEFORE paint, else escape bytes
        corrupt the len()-based width. (4) NAME '(none)' / DESCRIPTION '(missing)'
        when !HasFM||desc==''; TrimSpace kills the folded-scalar trailing newline.
        (5) wrapWords at fixed descWrapWidth=40 (terminal-width detection = forbidden
        dep). (6) rendered table proof + sorted order preserved by ui. (7) main wires
        Find()->Index()->PrintList for the FIRST time; empty-store exit-1 is REACHABLE
        (rules 1/2 need only a dir, not a SKILL.md). (8) --no-color is in scope;
        --search/--all/check/--help are NOT. (9) exact 4-file scope. (10) 94 tests green."
  critical: "Do NOT add golang.org/x/term (forbidden dep). Do NOT inspect the terminal
             inside ui (color decision stays in main). Do NOT pad AFTER painting
             (escape bytes break alignment). Do NOT emit ANSI when useColor=false
             (breaks pipes/$(...)/tests). Do NOT re-sort in ui (discover already did)."

# CONTRACT — the discover package design (PrintList's input + the data flow)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "Locks `func Index(absSkillsDir string) ([]Skill, error)` (line ~53) and the
        data flow `discover.Index(absSkillsDir) -> []Skill, error` (line ~23) that ui
        renders. The 'ANSI color' note: gate on stdout-is-a-TTY AND --no-color not set;
        --list/--search tables use color, PATH output never does (consumed by $(...)).
        'Exit codes': 1 = no skills found (--list) / skills dir unresolvable. The ui
        package is listed at `internal/ui/ui.go` (# --list/--search table + ANSI)."
  section: "Data flow", "ANSI color", "Exit codes", "Package map"

# PREDECESSOR — T5's landed Index + the []Skill shape PrintList consumes
- file: internal/discover/index.go
  why: "Index(skillsDir) ([]Skill, error): WalkDir, sorted by RelTag, every Dir
        absolute. PrintList consumes the returned slice AS-IS (no re-sort; the sort
        is already done). Empty store -> Index returns (nil/[], nil) -> main sees
        len==0 -> exit 1. READ-ONLY."
  pattern: "ui is downstream of Index: main calls Index, hands the []Skill to
            PrintList. ui never reads the filesystem."
  gotcha: "Index's malformed-YAML leniency means a Skill can have HasFM=false with
           empty Name/Description — that's exactly what (none)/(missing) render."

# PREDECESSOR — the Skill struct (the 9 fields PrintList reads)
- file: internal/discover/skill.go
  why: "Skill has RelTag, Name, Description, HasFM (and Keywords/Category/Aliases/
        Dir/SourceFile). PrintList reads RelTag/Name/Description/HasFM ONLY. Note
        Description is 'copied VERBATIM from Frontmatter, including a folded-scalar
        trailing newline' — ui MUST TrimSpace it or wrapping injects a blank line.
        READ-ONLY."
  gotcha: "Skill fields have NO yaml tags (built, not unmarshaled) — they're plain
           strings; len()/TrimSpace() work as expected on ASCII tags/names."

# PREDECESSOR — skillsdir.Find (the store resolution main calls first)
- file: internal/skillsdir/skillsdir.go
  why: "Find() (dir string, src Source, err error): rule 1 (SKPP_SKILLS_DIR) needs
        only an EXISTING DIR (no SKILL.md) — so an empty store IS resolvable, which
        is why main's len==0 -> exit-1 path is reachable and must be tested. On
        all-miss returns ('',0,ErrNotFound); err.Error() is the one-line fix
        (printed verbatim to stderr). READ-ONLY."
  pattern: "main's --list branch: dir,_,err := skillsdir.Find(); if err != nil {stderr;exit1}."

# PREDECESSOR — main.go (the dispatcher this subtask EXTENDS)
- file: main.go
  why: "The landed main.go (M1.T3): config{version,path}; parseArgs switch;
        run(args,stdout,stderr)int with version-precedence -> path -> default(1).
        T6 ADDS list/noColor to config, two parseArgs cases, the isTerminal var,
        and the --list branch in run. The version/path/default logic is PRESERVED
        verbatim. MODIFIED by this subtask (blueprint gives the complete new file)."
  gotcha: "Keep --version checked BEFORE --list (precedence, §6.3). Keep --path
           before --list too (path is its own mode). Put --list after path, before
           the default return 1."

# PREDECESSOR PRP — T5's downstream extension contract for T6 (what this implements)
- file: plan/001_fcde63e5bb60/P1M2T5S1/PRP.md
  why: "T5's 'DOWNSTREAM EXTENSION POINTS' locked the T6 contract verbatim:
        'P1.M2.T6.S1 (--list): ui reads the sorted []Skill; shows Description as
        \"(missing)\" when !HasFM or Description==\"\"; exits 1 when len==0.' And the
        main wiring note: 'dir,_,_ := skillsdir.Find(); skills,err := discover.Index(dir);
        if err != nil { stderr; exit 1 }.' This PRP implements exactly that."
  section: "Integration Points > DOWNSTREAM EXTENSION POINTS (P1.M2.T6.S1)"

# PREDECESSOR PRP — T3's main.go extension contract (parseArgs/run shape this extends)
- file: plan/001_fcde63e5bb60/P1M1T3.S1/PRP.md
  why: "T3's 'DOWNSTREAM EXTENSION POINTS' said: 'P1.M2.T6 (--list): add case
        \"--list\",\"-l\": c.list=true in parseArgs; add a if c.list { ... } branch in
        run (before the default).' This PRP does exactly that, plus --no-color (in
        the T6 title) + the isTerminal color gate."
  section: "Integration Points > DOWNSTREAM EXTENSION POINTS"

# CONTRACT — the PRD sections this implements
- file: PRD.md
  why: "§6.1 --list/-l: 'Human-readable catalog. Table: TAG, NAME, DESCRIPTION
        (wrapped). exit 0 (1 if no skills found)'. §6.2 --no-color: 'Disable ANSI
        color even on a TTY'. §6.3 --help/--version precedence (keep version before
        list). §6.4 nothing to stdout on failure. §13 acceptance: './skpp --list
        shows the example skill'. §4: Go; yaml.v3 is the ONLY third-party dep (so
        NO golang.org/x/term). READ-ONLY."
  critical: "§6.1 '--list exits 1 if no skills found' is a hard contract — an empty
             store MUST exit 1 (reachable because Find rules 1/2 don't require a
             SKILL.md). §4: do NOT add a terminal-detection library."

# REFERENCE — the repo's test convention (white-box, same-package, injected writers)
- file: internal/skillsdir/skillsdir_test.go
  why: "Established convention: white-box same-package test, t.TempDir/t.Setenv/
        t.Chdir, plain t.Errorf/t.Fatalf, NO testify, NO t.Parallel() on env/cwd/
        global-state tests. main_test.go mirrors this (package main); ui_test.go is
        package ui. READ-ONLY."
  pattern: "Capture output via injected *bytes.Buffer; control skills dir via
            SKPP_SKILLS_DIR (rule 1 wins deterministically)."

# URLS — the stdlib mechanisms this subtask is built from
- url: https://pkg.go.dev/os#FileStat
  why: "os.File.Stat() + FileMode&os.ModeCharDevice is the stdlib terminal test.
        A pty/tty has ModeCharDevice set; a regular file/pipe does not. Avoids
        golang.org/x/term (forbidden dep)."
  section: "type FileMode", "ModeCharDevice"
- url: https://pkg.go.dev/fmt#Fprintf
  why: "fmt.Fprintf(w, \"%s%s%s%s%s\\n\", ...) builds each table row; fmt.Fprintln
        for the error/one-liner lines. The %s form keeps ANSI bytes opaque."
- url: https://pkg.go.dev/strings#Fields
  why: "strings.Fields splits the description into words (collapses runs of
        whitespace) for wrapWords; strings.TrimSpace removes the folded-scalar
        trailing newline; strings.Repeat builds pad/indent spaces."
- url: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR
  why: "SGR (Select Graphic Rendition) sequences: ESC[1m = bold, ESC[36m = cyan,
        ESC[0m = reset. Plain byte-literal strings — no library. The reset after
        each run prevents color bleed across cells."
```

### Current Codebase tree (S1+S2+T5+M1 landed; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/discover/discover.go        # S1: Frontmatter(8) + ParseFrontmatter + utf8BOM
internal/discover/discover_test.go   # S1: 13 tests + writeSkill helper
internal/discover/skill.go           # S2: Skill(9) + BuildSkill + toStringSlice
internal/discover/skill_test.go      # S2: 6 tests + strEq helper
internal/discover/index.go           # T5: Index(skillsDir)([]Skill,error) + sort
internal/discover/index_test.go      # T5: 12 tests + makeTree helper
internal/skillsdir/skillsdir.go      # M1.T2: Find() (+rules) -> ABSOLUTE dir
internal/skillsdir/skillsdir_test.go # M1.T2: 29 tests
main.go                              # M1.T3: version/path dispatch (landed, green)
main_test.go                         # M1.T3: 15 tests

# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1
# baseline: go build ./... OK; go test ./... OK (discover + skillsdir + main green)
# NO internal/ui/ yet (THIS subtask). NO resolve/. NO skills/ (P1.M6.T12).
```

### Desired Codebase tree with files to be added/modified

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/discover/*,
│        internal/skillsdir/* — ALL UNCHANGED)
├── internal/
│   ├── discover/      # UNCHANGED (S1+S2+T5; consumed read-only)
│   ├── skillsdir/     # UNCHANGED (M1.T2; consumed read-only)
│   └── ui/
│       ├── ui.go      # CREATE — PrintList + padRight + wrapWords + ANSI consts
│       └── ui_test.go # CREATE — 11 white-box tests (helpers: mk, colOf)
├── main.go            # MODIFY — +list/noColor config, --list/--no-color cases,
│                      #          isTerminal var, run --list branch, +discover/+ui imports
└── main_test.go       # MODIFY — +io/os imports, withTerminal + writeSkillTree helpers,
                       #          8 new --list/--no-color tests (existing 15 unchanged)
```

| File | Action | Responsibility | Imports |
|---|---|---|---|
| `internal/ui/ui.go` | CREATE | Render the TAG/NAME/DESCRIPTION table (color opt-in); wrap+align | fmt, io, strings, internal/discover |
| `internal/ui/ui_test.go` | CREATE | 11 tests: empty/color/(none)/(missing)/trim/wrap/order/align | bytes, strings, testing, internal/discover |
| `main.go` | MODIFY | Wire `--list`/`-l` + `--no-color`; TTY gate; Find→Index→PrintList | fmt, io, os, discover, skillsdir, ui |
| `main_test.go` | MODIFY | Tests for --list dispatch + --no-color + TTY color path | bytes, io, os, path/filepath, strings, testing |

**Two new files + two modified files. Zero changes to `go.mod`, `go.sum`,
`internal/discover/*`, `internal/skillsdir/*`, `PRD.md`. No `resolve`/`skills`/
`install.sh`/`README`/completions.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — Keep color decision OUT of the ui package; ui takes useColor.
// ui.PrintList(w, skills, useColor bool) is a PURE formatter. The TTY/--no-color
// decision lives in main (isTerminal(stdout) && !c.noColor). Rationale: (a) tests
// pass a *bytes.Buffer, so any in-package TTY check would always be false and the
// color path would be untestable without monkey-patching; (b) keeping the package
// terminal-free makes it a clean leaf that --search (T9) reuses identically.
// VERIFIED (ui_test passes useColor=true directly; main_test overrides isTerminal).
//   RIGHT: func PrintList(w io.Writer, skills []discover.Skill, useColor bool)
//   WRONG: PrintList peeking at os.Stdout / term.IsTerminal itself

// GOTCHA #2 — TTY detection is STDLING os.ModeCharDevice; do NOT add x/term.
// PRD §4/§7.3 keep gopkg.in/yaml.v3 the ONLY third-party dep. The stdlib idiom:
// type-assert w to *os.File; check fi.Mode()&os.ModeCharDevice. A *bytes.Buffer
// fails the assertion -> false (deterministic tests); a pipe/file -> false
// (pipe-safe); a real pty -> true (color on). VERIFIED (research §2).
//   RIGHT: f,ok := w.(*os.File); ... fi.Mode()&os.ModeCharDevice != 0
//   WRONG: import "golang.org/x/term"; term.IsTerminal(int(f.Fd()))  // forbidden dep

// GOTCHA #3 — Pad the PLAIN string BEFORE painting, or escape bytes break alignment.
// paint(code,s) = code+s+reset; len() of that counts the 5+4 escape bytes, so
// padding AFTER painting over-pads and misaligns columns. padRight(plain,width)
// FIRST (uses len(plain)), THEN paint(code, padded). Trailing pad spaces land
// inside the SGR run (invisible); reset is last. VERIFIED (TestPrintListColumnsAlignedAcrossRows).
//   RIGHT: paint(ansiCyan, padRight(r.tag, tagW))
//   WRONG: padRight(paint(ansiCyan, r.tag), tagW)   // len() counts ESC bytes

// GOTCHA #4 — TrimSpace the Description (folded-scalar trailing newline).
// yaml.v3 folds `description: >` into a string ENDING IN "\n" (discover.go S1
// contract: copied verbatim, not trimmed). Without TrimSpace, the wrap output gets
// a stray blank line. ui does strings.TrimSpace(desc) before wrapping/storing.
// VERIFIED (TestPrintListTrimsFoldedScalarNewline asserts no "\n\n").
//   RIGHT: desc := strings.TrimSpace(s.Description)
//   WRONG: desc := s.Description   // folded scalar -> trailing \n -> blank line

// GOTCHA #5 — Wrap at a FIXED width; do NOT detect terminal width.
// Terminal width needs TIOCGWINSZ ioctl or golang.org/x/term (forbidden dep, #2).
// A fixed descWrapWidth=40 is deterministic, testable, fits an 80-col terminal,
// and satisfies PRD §6.1 "(wrapped)". If responsive width is ever wanted it is a
// follow-up (would need x/term or a syscall). VERIFIED (research §5).
//   RIGHT: const descWrapWidth = 40; wrapWords(desc, descWrapWidth)
//   WRONG: querying terminal columns via a syscall or x/term

// GOTCHA #6 — ui must NOT re-sort; discover.Index already sorted by RelTag.
// PrintList renders skills in INPUT order. Index sorts; --search passes a filtered
// (still-sorted) slice. Re-sorting in ui would be redundant and would surprise T9.
// VERIFIED (TestPrintListPreservesInputOrder: zebra-before-apple input is kept).
//   RIGHT: for _, r := range rows { ... }   // rows in input order
//   WRONG: sort.Slice(skills, ...) in PrintList

// GOTCHA #7 — Empty store exit-1 is REACHABLE; test it.
// skillsdir rules 1 (env) and 2 (sibling) need only an EXISTING DIR, not a
// SKILL.md. So SKPP_SKILLS_DIR=/empty/dir -> Find succeeds -> Index returns [] ->
// len==0 -> main MUST exit 1 (PRD §6.1 "1 if no skills found"). This is not dead
// code. VERIFIED (TestRunListNoSkillsExit1). PrintList itself returns early on
// empty (defensive), but main is authoritative (prints "no skills found" to stderr).
//   RIGHT: if len(skills)==0 { fmt.Fprintln(stderr, "no skills found in "+dir); return 1 }
//   WRONG: assuming Find/never-empty and skipping the len==0 check

// GOTCHA #8 — isTerminal is a package VAR (overridable in tests); not a plain func.
// To test the color-ENABLED path through run() (where stdout is a *bytes.Buffer =
// not a TTY), main_test swaps isTerminal via withTerminal(t,true)+t.Cleanup(restore).
// A plain func could not be swapped. NOT t.Parallel-safe (global state) — but the
// repo convention is no t.Parallel() on such tests. VERIFIED (TestRunListColorWhenTTY).
//   RIGHT: var isTerminal = func(w io.Writer) bool { ... }
//   WRONG: func isTerminal(w io.Writer) bool { ... }   // untestable color path

// GOTCHA #9 --no-color is IN SCOPE here; --search/--all/check/--help are NOT.
// The task title is "...+ ANSI/--no-color + --list wiring", so wire --no-color now
// (it is a §6.2 modifier that combines with --list). Do NOT add --search (T9),
// --all/-a or <tag> positionals (M3), check (M4), or --help (M5) — their cases and
// run branches stay absent (tolerated default). VERIFIED (parseArgs has exactly
// version/path/list/noColor cases; run has version/path/list branches).
//   RIGHT: case "--no-color": c.noColor = true
//   WRONG: also adding --search/--all/check/--help cases (out of scope -> drift)

// GOTCHA #10 — Preserve main's version/path/default logic VERBATIM.
// This is the FIRST main.go edit since M1.T3. Add list/noColor to config; add the
// two cases; add isTerminal; add the --list branch AFTER --path and BEFORE the
// default return 1. Do NOT touch the version var, the precedence ordering, the
// --path byte-exact Fprintln(stdout,dir), or the tolerated-default return 1.
// VERIFIED (all 15 existing main tests still pass unchanged).
//   RIGHT: ... if c.path {...} if c.list {...} return 1
//   WRONG: reordering precedence or altering the --path branch

// GOTCHA #11 — ui.go imports ONLY fmt/io/strings/discover; NO os, NO term.
// ui is a pure formatter (it takes an io.Writer). It must not import os (no TTY
// check inside) or any terminal lib. main owns os (isTerminal) + the writers.
// VERIFIED (go vet clean; go.mod unchanged).
//   RIGHT: import ("fmt"; "io"; "strings"; ".../internal/discover")
//   WRONG: import "os" in ui.go   // unused + violates the pure-formatter design
```

---

## Implementation Blueprint

### Data model — no new types in ui; `config` grows two fields

ui adds no exported types (only `PrintList`/`padRight`/`wrapWords` + `const`s). It
consumes `discover.Skill` read-only. `main.config` grows:

```go
type config struct {
	version bool   // --version / -v
	path    bool   // --path / -p
	list    bool   // --list / -l    : print the human-readable catalog table (§6.1)   [NEW]
	noColor bool   // --no-color     : disable ANSI color even on a TTY (§6.2)        [NEW]
}
```

### File 1 — `internal/ui/ui.go` (CREATE)

Create the file with EXACTLY this content (gofmt-clean; compiled + all 11 ui tests
pass against the real `discover` package in `/tmp/skpp_ui_verify`):

````go
// Package ui renders the human-readable skill catalog table for skpp's --list
// and --search modes (PRD §6.1). It is a PURE formatter: it takes a
// []discover.Skill (already discovered and sorted by the caller — discover.Index
// sorts by RelTag) and writes a TAG/NAME/DESCRIPTION table. Color is opt-in via a
// useColor parameter, so the caller (main) owns the TTY / --no-color decision and
// unit tests are fully deterministic (no real terminal required).
//
// This is the P1.M2.T6.S1 deliverable. main.go wires `--list`/`-l` and
// `--no-color` (PRD §6.1/§6.2) to call PrintList; --search (P1.M4.T9) reuses
// PrintList with a filtered slice (PRD §6.1: "same table format as --list").
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

// ANSI SGR escape sequences. Only emitted when useColor is true. The reset
// (ansiReset) is appended after every colored run so a single cell cannot bleed
// color into the next. All are plain byte-literal strings — no third-party dep.
const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiCyan  = "\x1b[36m"
)

// descWrapWidth is the column width at which the DESCRIPTION cell is word-wrapped.
// skpp deliberately does NOT detect terminal width: that needs a TIOCGWINSZ ioctl
// or golang.org/x/term, and PRD §4/§7.3 keep gopkg.in/yaml.v3 the ONLY third-party
// dependency. A fixed width keeps output deterministic and testable and fits a
// standard 80-column terminal alongside typical TAG/NAME widths.
const descWrapWidth = 40

// PrintList writes the TAG/NAME/DESCRIPTION catalog table for skills to w. It
// implements PRD §6.1 `skpp --list`. skills MUST already be ordered the way rows
// should appear: discover.Index sorts by RelTag; --search passes its filtered
// (still-sorted) slice. An empty slice prints nothing — main exits 1 "if no skills
// found" before calling this (PRD §6.1); PrintList is defensive, not authoritative.
//
// Column rules:
//   - TAG:  Skill.RelTag (the canonical '/'-normalized tag from discover).
//   - NAME: Skill.Name, or "(none)" when the frontmatter name is empty.
//   - DESCRIPTION: Skill.Description, or "(missing)" when the SKILL.md had no
//     frontmatter block (HasFM==false) OR the description is empty/blank.
//
// DESCRIPTION is word-wrapped to descWrapWidth; continuation lines leave the TAG
// and NAME cells blank (spaces) so columns stay aligned.
//
// When useColor is false (non-TTY stdout, or --no-color set), output is plain
// text — exactly what `$(...)`, pipes, log files, and tests see. Color is applied
// to the header (bold) and the TAG column (cyan); NAME/DESCRIPTION stay default.
func PrintList(w io.Writer, skills []discover.Skill, useColor bool) {
	if len(skills) == 0 {
		return
	}

	// Compute display cells + dynamic column widths from the PLAIN content, so the
	// widths are independent of whether color is on.
	tagW := len("TAG")
	nameW := len("NAME")
	type cells struct{ tag, name, desc string }
	rows := make([]cells, len(skills))
	for i, s := range skills {
		name := s.Name
		if name == "" {
			name = "(none)"
		}
		// HasFM==false OR blank description -> "(missing)". A folded-scalar
		// description may carry a trailing newline (discover.go contract);
		// TrimSpace normalizes it so it does not inject a blank line into the wrap.
		desc := strings.TrimSpace(s.Description)
		if !s.HasFM || desc == "" {
			desc = "(missing)"
		}
		if len(s.RelTag) > tagW {
			tagW = len(s.RelTag)
		}
		if len(name) > nameW {
			nameW = len(name)
		}
		rows[i] = cells{s.RelTag, name, desc}
	}

	// paint wraps s in an SGR sequence + reset when color is on; otherwise it is a
	// passthrough. padRight is applied to the PLAIN string BEFORE paint so visible
	// column alignment is unaffected by the (invisible) escape bytes (their bytes
	// would otherwise corrupt the len()-based padding math).
	paint := func(code, s string) string {
		if !useColor {
			return s
		}
		return code + s + ansiReset
	}

	const sep = "  "
	tagPad := strings.Repeat(" ", tagW)
	namePad := strings.Repeat(" ", nameW)

	// Header row: bold labels.
	fmt.Fprintf(w, "%s%s%s%s%s\n",
		paint(ansiBold, padRight("TAG", tagW)),
		sep,
		paint(ansiBold, padRight("NAME", nameW)),
		sep,
		paint(ansiBold, "DESCRIPTION"),
	)

	for _, r := range rows {
		descLines := wrapWords(r.desc, descWrapWidth)
		// First description line shares the row with the TAG + NAME cells (tag cyan).
		fmt.Fprintf(w, "%s%s%s%s%s\n",
			paint(ansiCyan, padRight(r.tag, tagW)),
			sep,
			padRight(r.name, nameW),
			sep,
			descLines[0],
		)
		// Continuation lines: blank TAG/NAME cells, plain wrapped description.
		for _, line := range descLines[1:] {
			fmt.Fprintf(w, "%s%s%s%s%s\n", tagPad, sep, namePad, sep, line)
		}
	}
}

// padRight returns s right-padded with spaces to width n. If len(s) >= n, s is
// returned unchanged (no truncation). Operates on byte length, which is correct
// for the ASCII values skpp handles (Agent Skills names are lowercase a-z0-9-;
// tags are relative dir paths of the same).
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// wrapWords word-wraps s into lines of at most width characters, breaking at
// spaces. A single word longer than width is placed on its own line (not split).
// Runs of spaces collapse to one. An empty/whitespace-only s yields a single
// empty line so callers can always index [0].
func wrapWords(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, len(words))
	cur := ""
	for _, word := range words {
		switch {
		case cur == "":
			cur = word
		case len(cur)+1+len(word) <= width:
			cur += " " + word
		default:
			lines = append(lines, cur)
			cur = word
		}
	}
	lines = append(lines, cur)
	return lines
}
````

### File 2 — `internal/ui/ui_test.go` (CREATE, `package ui` white-box)

Create the file with EXACTLY this content (gofmt-clean; all 11 tests pass):

````go
package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

// mk builds one discover.Skill for table tests. fm controls HasFM (and thus the
// missing-description rule). Kept tiny so test rows stay readable.
func mk(tag, name, desc string, fm bool) discover.Skill {
	return discover.Skill{RelTag: tag, Name: name, Description: desc, HasFM: fm}
}

// colOf returns the column (0-based) of the first occurrence of substr in out,
// measured from the previous newline. Used to assert column alignment.
func colOf(out, substr string) int {
	idx := strings.Index(out, substr)
	if idx < 0 {
		return -1
	}
	return idx - (strings.LastIndex(out[:idx], "\n") + 1)
}

func TestPrintListEmptyPrintsNothing(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, nil, false)
	if buf.Len() != 0 {
		t.Errorf("empty input printed %q; want nothing", buf.String())
	}
}

func TestPrintListSingleNoColor(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("example", "example", "A demo skill.", true)}, false)
	out := buf.String()
	for _, want := range []string{"TAG", "NAME", "DESCRIPTION", "example", "A demo skill."} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("no-color output must not contain ANSI escapes:\n%s", out)
	}
	// Header precedes the data row.
	if h, d := strings.Index(out, "DESCRIPTION"), strings.Index(out, "A demo skill."); h < 0 || d < 0 || h > d {
		t.Errorf("header should precede data:\n%s", out)
	}
}

func TestPrintListColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("example", "example", "d", true)}, true)
	out := buf.String()
	for _, want := range []string{ansiBold, ansiCyan, ansiReset} {
		if !strings.Contains(out, want) {
			t.Errorf("color output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintListNoColorHasNoANSI(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("example", "example", "d", true)}, false)
	if strings.Contains(buf.String(), "\x1b") {
		t.Errorf("no-color output contains escapes:\n%s", buf.String())
	}
}

func TestPrintListMissingNameShowsNone(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("plain", "", "d", true)}, false)
	if !strings.Contains(buf.String(), "(none)") {
		t.Errorf("empty name should render (none):\n%s", buf.String())
	}
}

func TestPrintListEmptyDescriptionShowsMissing(t *testing.T) {
	var buf bytes.Buffer
	// HasFM true but description empty -> "(missing)".
	PrintList(&buf, []discover.Skill{mk("a", "a", "", true)}, false)
	if !strings.Contains(buf.String(), "(missing)") {
		t.Errorf("empty description should render (missing):\n%s", buf.String())
	}
}

func TestPrintListNoFrontmatterShowsMissing(t *testing.T) {
	var buf bytes.Buffer
	// HasFM false -> "(missing)" regardless of description.
	PrintList(&buf, []discover.Skill{mk("b", "b", "", false)}, false)
	if !strings.Contains(buf.String(), "(missing)") {
		t.Errorf("no-frontmatter skill should render (missing):\n%s", buf.String())
	}
}

func TestPrintListTrimsFoldedScalarNewline(t *testing.T) {
	var buf bytes.Buffer
	// A folded-scalar description carries a trailing newline (discover.go contract).
	PrintList(&buf, []discover.Skill{mk("x", "x", "has trailing newline\n", true)}, false)
	out := buf.String()
	if !strings.Contains(out, "has trailing newline") {
		t.Errorf("description text missing:\n%s", out)
	}
	if strings.Contains(out, "\n\n") {
		t.Errorf("output has a blank line (folded newline not trimmed):\n%s", out)
	}
}

func TestPrintListWrapsLongDescription(t *testing.T) {
	var buf bytes.Buffer
	long := "Reference example skill for skpp. Demonstrates the required frontmatter and how skpp resolves a tag to an absolute path. Safe to delete once you add real skills."
	PrintList(&buf, []discover.Skill{mk("example", "example", long, true)}, false)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// header + >=2 wrapped data lines.
	if len(lines) < 3 {
		t.Fatalf("expected wrapped multi-line output, got %d lines:\n%s", len(lines), buf.String())
	}
	// Every wrapped line fits within descWrapWidth (no line overruns the column).
	descCol := colOf(buf.String(), "DESCRIPTION")
	for _, ln := range lines[1:] {
		if len(ln)-descCol > descWrapWidth {
			t.Errorf("wrapped line exceeds %d cols (descCol=%d):\n%q", descWrapWidth, descCol, ln)
		}
	}
	// All words survived (joining lines with spaces reconstructs the word stream).
	joined := strings.Join(lines, " ")
	for _, want := range []string{"Reference", "frontmatter", "real", "skills."} {
		if !strings.Contains(joined, want) {
			t.Errorf("wrapped output lost word %q:\n%s", want, joined)
		}
	}
}

func TestPrintListPreservesInputOrder(t *testing.T) {
	var buf bytes.Buffer
	// Input is zebra then apple; ui must NOT re-sort (discover.Index already did).
	PrintList(&buf, []discover.Skill{
		mk("zebra", "zebra", "z", true),
		mk("apple", "apple", "a", true),
	}, false)
	out := buf.String()
	// "zebra" appears once in the header? No — header is TAG/NAME/DESCRIPTION. The
	// tag value "zebra" first occurs in the zebra data row, which must precede apple.
	zi := strings.Index(out, "zebra")
	ai := strings.Index(out, "apple")
	if zi < 0 || ai < 0 || zi > ai {
		t.Errorf("expected zebra row before apple row (input order):\n%s", out)
	}
}

func TestPrintListColumnsAlignedAcrossRows(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{
		mk("a", "alpha", "short", true),
		mk("writing/reddit", "reddit-helper", "longer desc here", true),
	}, false)
	out := buf.String()
	descCol := colOf(out, "DESCRIPTION")
	if descCol < 0 {
		t.Fatalf("no DESCRIPTION header:\n%s", out)
	}
	// The longest tag ("writing/reddit") sets the column; "a"/"alpha" are padded so
	// every description starts at the same column under DESCRIPTION.
	for _, want := range []string{"short", "longer"} {
		if c := colOf(out, want); c != descCol {
			t.Errorf("desc %q starts at col %d; want %d (aligned under DESCRIPTION):\n%s", want, c, descCol, out)
		}
	}
	// The continuation-less first row's NAME is aligned under the NAME header.
	nameCol := colOf(out, "NAME")
	if c := colOf(out, "alpha"); c != nameCol {
		t.Errorf("'alpha' at col %d; want %d:\n%s", c, nameCol, out)
	}
}
````

### File 3 — `main.go` (MODIFY — write this complete file over the existing one)

The complete updated `main.go` (gofmt-clean; all 23 main tests pass). It preserves
the M1.T3 version/path/default logic verbatim and adds `list`/`noColor`, the
`isTerminal` var, the two parseArgs cases, and the `--list` branch:

````go
// Command skpp resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// Wired so far (grown milestone by milestone): --version/--path (M1.T3) and
// --list (M2.T6). Every other §6 flag is added by later milestones (M3
// <tag>/--all, M4 --search/check, M5 --help + exit codes). The arg parser is
// intentionally a small hand-rolled switch (not Go's `flag` package) so the full
// §6 matrix — subcommands like `check`, positional <tag> args, long+short
// aliases, and §6.3 mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1T3.S1/research/verified_facts.md §5.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
)

// version is the skpp version string, printed by `skpp --version`. It is
// overridden at BUILD time via ldflags (PRD §12.1 build command):
//
//	go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .
//
// The default "dev" is used by `go run` and plain `go build` (no ldflags).
//
// IMPORTANT: this MUST be a package-level var, not a const. `-X main.version=...`
// rewrites a package-scope string var at link time; it cannot override a const
// (compile error) or a function-local. Because this file is `package main`, the
// linker symbol path is `main.version` (NOT the module import path).
var version = "dev"

// isTerminal reports whether w is an interactive terminal (a character device).
// It decides whether --list/--search emit ANSI color by default (PRD §6.2: color
// is on for a TTY unless --no-color is set). It type-asserts w to *os.File and
// checks the ModeCharDevice bit, so a *bytes.Buffer (tests) or a pipe/redirect
// correctly yields false -> no color, keeping output deterministic and pipe-safe.
//
// It is a package var so tests can override it to exercise the color-enabled path
// through run() without a real terminal. NOT safe for t.Parallel (mutates package
// state); the repo convention is no t.Parallel() on such tests anyway.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// config holds the parsed CLI flags. Grown by later milestones as more of the
// PRD §6.1/§6.2 matrix lands. For this subtask version, path, list, and noColor
// are set; every other token is a tolerated no-op (P1.M5.T11 turns unknown flags
// into exit 2 and adds subcommand/positional handling).
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	list    bool // --list / -l    : print the human-readable catalog table (§6.1)
	noColor bool // --no-color     : disable ANSI color even on a TTY (§6.2)
	// Future (M3-M5), do NOT add yet:
	//   all bool; search string; check bool; file, relative, help bool; tags []string
}

// parseArgs scans argv tokens and fills a config. Flags may appear in any order
// (PRD §6). Long forms use POSIX double-dash; short forms a single dash. Unknown
// tokens are tolerated for now (a no-op switch default); the full unknown-flag
// -> exit 2 behavior and subcommand/positional parsing land in P1.M5.T11.
//
// To add a flag in a later milestone: append a `case "--name", "-n": cfg.name =
// true` (or capture the next arg for value-taking flags like --search <q>).
func parseArgs(args []string) config {
	var c config
	for _, a := range args {
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--no-color":
			c.noColor = true
		default:
			// Unknown flag / subcommand / positional: tolerated for now.
			// P1.M5.T11 implements: unknown flag -> exit 2 (§6.2),
			// `check` subcommand dispatch, and <tag> positional capture.
		}
	}
	return c
}

// run is the testable dispatcher. It returns the process exit code so main() can
// call os.Exit(run(...)) without tests ever invoking os.Exit. stdout/stderr are
// injected so tests capture output via *bytes.Buffer instead of the real streams.
//
// Exit codes (PRD §6; this subtask's slice):
//   - 0: --version printed; --path succeeded; --list printed the catalog
//   - 1: --path/--list failed (skills dir unresolvable, or no skills for --list);
//     default (no recognized flag)
//   - 2: (DEFERRED to P1.M5.T11) unknown flag / mutually-exclusive modes mixed
//
// Precedence (PRD §6.3): --version (and, in M5, --help) win over everything.
func run(args []string, stdout, stderr io.Writer) int {
	c := parseArgs(args)

	// Precedence tier: --version wins over every other flag (PRD §6.3).
	// P1.M5.T11 adds --help/-h to this same tier (before --path).
	if c.version {
		fmt.Fprintf(stdout, "skpp %s\n", version)
		return 0
	}

	if c.path {
		dir, _, err := skillsdir.Find() // src is for reporting only; not printed
		if err != nil {
			// Find() returns skillsdir.ErrNotFound whose message is the
			// user-facing one-line fix (PRD §8.4/§6.4). Print it verbatim to
			// stderr (NOT stdout) so $(...) stays empty on failure.
			fmt.Fprintln(stderr, err)
			return 1
		}
		// Byte-exact: ONLY the dir + newline. The §13 acceptance gate
		// `test "$(./skpp --path)" = "$PWD/skills"` depends on this.
		fmt.Fprintln(stdout, dir)
		return 0
	}

	if c.list {
		// PRD §6.1 `skpp --list`: resolve the store, build the index, render the
		// table. This is the FIRST place the Find() -> discover.Index() data flow
		// is wired end-to-end (M2.T6). Exit 1 on any failure path.
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
		if len(skills) == 0 {
			// PRD §6.1: --list exits 1 "if no skills found". Message to stderr so
			// stdout stays clean for any consumer.
			fmt.Fprintln(stderr, "no skills found in "+dir)
			return 1
		}
		// Color only when stdout is a TTY AND --no-color was not given (PRD §6.2).
		// A *bytes.Buffer (tests) / pipe / file is not a TTY -> plain output.
		ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}
````

### File 4 — `main_test.go` (MODIFY — write this complete file over the existing one)

The complete updated `main_test.go` (gofmt-clean; all 23 tests pass). It preserves
all 15 existing tests verbatim and adds `io`/`os` imports, the `withTerminal` +
`writeSkillTree` helpers, and the 8 new `--list`/`--no-color` tests:

````go
package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTerminal overrides the package-level isTerminal func for one test and
// restores it on cleanup. Use it to exercise the color-enabled path through
// run() without a real terminal. NOT t.Parallel-safe (mutates package state).
func withTerminal(t *testing.T, isTTY bool) {
	t.Helper()
	prev := isTerminal
	isTerminal = func(io.Writer) bool { return isTTY }
	t.Cleanup(func() { isTerminal = prev })
}

// unsetSkillsEnv removes SKPP_SKILLS_DIR for the test and restores it on cleanup.
// (Mirrors internal/skillsdir/skillsdir_test.go's unsetEnvVar helper.) Forbids
// t.Parallel via t.Setenv.
func unsetSkillsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SKPP_SKILLS_DIR", "")
}

// writeSkillTree builds a temp skills/ tree from a map[relTag]SKILL.md-content
// and returns its root. relTag uses '/' separators (cross-platform via FromSlash).
// A "" key writes SKILL.md directly in the root. Used by the --list tests to give
// skillsdir.Find() (via SKPP_SKILLS_DIR) a real store to discover.
func writeSkillTree(t *testing.T, layout map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relTag, content := range layout {
		dir := filepath.Join(root, filepath.FromSlash(relTag))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", dir, err)
		}
	}
	return root
}

// --- parseArgs ---

func TestParseArgsEmpty(t *testing.T) {
	c := parseArgs(nil)
	if c.version || c.path {
		t.Errorf("parseArgs(nil): version=%v path=%v; want both false", c.version, c.path)
	}
}

func TestParseArgsVersionLong(t *testing.T) {
	c := parseArgs([]string{"--version"})
	if !c.version || c.path {
		t.Errorf("parseArgs(--version): version=%v path=%v; want true,false", c.version, c.path)
	}
}

func TestParseArgsVersionShort(t *testing.T) {
	c := parseArgs([]string{"-v"})
	if !c.version {
		t.Errorf("parseArgs(-v): version=false; want true")
	}
}

func TestParseArgsPathLong(t *testing.T) {
	c := parseArgs([]string{"--path"})
	if !c.path || c.version {
		t.Errorf("parseArgs(--path): path=%v version=%v; want true,false", c.path, c.version)
	}
}

func TestParseArgsPathShort(t *testing.T) {
	c := parseArgs([]string{"-p"})
	if !c.path {
		t.Errorf("parseArgs(-p): path=false; want true")
	}
}

// Flags may appear in any order (PRD §6); both long+short forms recognized.
func TestParseArgsAnyOrderBothForms(t *testing.T) {
	c := parseArgs([]string{"-p", "--version"})
	if !c.version || !c.path {
		t.Errorf("parseArgs(-p --version): version=%v path=%v; want true,true", c.version, c.path)
	}
}

// Unknown tokens are tolerated (no-op) for now; exit-2 lands in P1.M5.T11.
func TestParseArgsUnknownTolerated(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "sometag", "check"})
	if c.version || c.path {
		t.Errorf("parseArgs(unknown): version=%v path=%v; want both false (tolerated)", c.version, c.path)
	}
}

func TestParseArgsListLong(t *testing.T) {
	c := parseArgs([]string{"--list"})
	if !c.list || c.version || c.path {
		t.Errorf("parseArgs(--list): list=%v; want true (others false)", c.list)
	}
}

func TestParseArgsListShort(t *testing.T) {
	c := parseArgs([]string{"-l"})
	if !c.list {
		t.Errorf("parseArgs(-l): list=false; want true")
	}
}

func TestParseArgsNoColor(t *testing.T) {
	c := parseArgs([]string{"--no-color"})
	if !c.noColor {
		t.Errorf("parseArgs(--no-color): noColor=false; want true")
	}
}

// --- run: --version / -v ---

func TestRunVersionPrintsSkppVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--version): code=%d; want 0", code)
	}
	want := "skpp " + version + "\n" // version == "dev" under `go test` (no ldflags)
	if got := out.String(); got != want {
		t.Errorf("run(--version) stdout=%q; want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--version) stderr=%q; want empty", errOut.String())
	}
}

func TestRunVersionShortFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-v"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-v): code=%d; want 0", code)
	}
	if !strings.HasPrefix(out.String(), "skpp ") {
		t.Errorf("run(-v) stdout=%q; want 'skpp <version>\\n'", out.String())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Errorf("run(-v) stdout=%q; want trailing newline", out.String())
	}
}

// --- run: --path / -p ---

// --path success: SKPP_SKILLS_DIR set to an existing dir -> rule 1 wins, Find()
// returns that dir, printed byte-exact to stdout, exit 0.
func TestRunPathSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKPP_SKILLS_DIR", dir) // rule 1 wins deterministically
	var out, errOut bytes.Buffer
	code := run([]string{"--path"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--path) success: code=%d; want 0", code)
	}
	// Find() cleans the env value via filepath.Abs, so compare to the cleaned form.
	want := filepath.Clean(dir) + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(--path) stdout=%q; want %q (byte-exact dir + newline)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--path) success stderr=%q; want empty", errOut.String())
	}
}

func TestRunPathShortFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-p"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-p): code=%d; want 0", code)
	}
	if got := out.String(); got != filepath.Clean(dir)+"\n" {
		t.Errorf("run(-p) stdout=%q; want %q", got, filepath.Clean(dir)+"\n")
	}
}

// --path failure: env unset + cwd in an empty temp tree -> all three §8 rules
// miss -> Find() returns ErrNotFound. Assert: exit 1, stdout EMPTY, stderr has
// the one-line fix (SKPP_SKILLS_DIR / cd / reinstall). Empty stdout is the §6.4
// contract that makes `pi --skill "$(skpp bad)"` fail loudly.
func TestRunPathFailureErrNotFound(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // empty tree -> rule 3 ascends to / and misses; rule 2 misses in tests
	var out, errOut bytes.Buffer
	code := run([]string{"--path"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--path) failure: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--path) failure stdout=%q; want EMPTY (§6.4: print nothing on failure)", out.String())
	}
	msg := errOut.String()
	for _, want := range []string{"SKPP_SKILLS_DIR", "cd", "reinstall"} {
		if !strings.Contains(msg, want) {
			t.Errorf("run(--path) failure stderr=%q; missing substring %q", msg, want)
		}
	}
}

// --- run: precedence ---

// --version takes precedence over --path (PRD §6.3): version printed, Find()
// never called, exit 0 — even though skills dir is unresolvable here.
func TestRunVersionPrecedenceOverPath(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // would make --path fail, but --version wins first
	var out, errOut bytes.Buffer
	code := run([]string{"--path", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--path --version): code=%d; want 0 (version precedence)", code)
	}
	want := "skpp " + version + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(--path --version) stdout=%q; want %q (version, not path)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--path --version) stderr=%q; want empty", errOut.String())
	}
}

// --- run: default (no recognized flag) ---

// No args / unknown flags: tolerated for now, exit 1 (the eventual §6.3 no-args
// code), no usage text yet (P1.M5.T11). NOT exit 2 (deferred to M5).
func TestRunDefaultNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(nil, &out, &errOut)
	if code != 1 {
		t.Errorf("run(nil): code=%d; want 1 (no-args default; usage text is M5)", code)
	}
}

func TestRunDefaultUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--frobnicate"}, &out, &errOut)
	if code != 1 {
		t.Errorf("run(--frobnicate): code=%d; want 1 (unknown tolerated; exit-2 is M5)", code)
	}
}

// --- run: --list / -l (P1.M2.T6) ---

// --list success: a store with one skill -> catalog table on stdout, exit 0, no
// ANSI (stdout is a *bytes.Buffer -> not a TTY -> plain output by default).
func TestRunListSuccess(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: A demo skill.\n---\n# body\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir) // rule 1 wins; Find() returns dir, Index finds the skill
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--list): code=%d; want 0", code)
	}
	got := out.String()
	for _, want := range []string{"TAG", "NAME", "DESCRIPTION", "example", "A demo skill."} {
		if !strings.Contains(got, want) {
			t.Errorf("run(--list) stdout missing %q:\n%s", want, got)
		}
	}
	// Default (non-TTY buffer) -> no ANSI escapes.
	if strings.Contains(got, "\x1b[") {
		t.Errorf("run(--list) on a non-TTY must not emit ANSI:\n%s", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--list) stderr=%q; want empty", errOut.String())
	}
}

func TestRunListShortFlag(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-l"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-l): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "example") {
		t.Errorf("run(-l) stdout missing the example tag:\n%s", out.String())
	}
}

// --list with NO skills (empty store) -> PRD §6.1 exit 1, stdout empty, a message
// to stderr. SKPP_SKILLS_DIR pointing at an existing-but-empty dir: rule 1 wins
// (it needs only an existing dir), Index returns [], len==0 -> exit 1.
func TestRunListNoSkillsExit1(t *testing.T) {
	t.Setenv("SKPP_SKILLS_DIR", t.TempDir()) // exists, no SKILL.md -> empty catalog
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--list) empty store: code=%d; want 1 (PRD §6.1 '1 if no skills found')", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--list) empty store stdout=%q; want empty (only the exit-1 + stderr msg)", out.String())
	}
	if !strings.Contains(errOut.String(), "no skills found") {
		t.Errorf("run(--list) empty store stderr=%q; want a 'no skills found' message", errOut.String())
	}
}

// --list when the skills dir is unresolvable -> Find() returns ErrNotFound ->
// exit 1, stdout empty, the one-line fix to stderr (same contract as --path).
func TestRunListSkillsDirUnresolvableExit1(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // force all three §8 rules to miss
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--list) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--list) unresolvable stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("run(--list) unresolvable stderr=%q; want the one-line fix", errOut.String())
	}
}

// --list with --no-color suppresses ANSI even when stdout looks like a TTY.
// Forces isTerminal=true (so color WOULD be on by default) and asserts --no-color
// still yields plain output.
func TestRunListNoColorFlagSuppressesANSI(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true) // pretend stdout is a TTY
	var out, errOut bytes.Buffer
	code := run([]string{"--list", "--no-color"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--list --no-color): code=%d; want 0", code)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Errorf("--no-color must suppress ANSI even on a TTY:\n%s", out.String())
	}
}

// --list color path: when stdout is a TTY (forced) and --no-color is absent, the
// table carries ANSI escapes. Proves the TTY gate is wired into run().
func TestRunListColorWhenTTY(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true)
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--list) tty: code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[1m") || !strings.Contains(got, "\x1b[36m") || !strings.Contains(got, "\x1b[0m") {
		t.Errorf("TTY output should contain ANSI bold/cyan/reset:\n%s", got)
	}
}
````

> **Copy-paste correctness:** all four blueprint files are gofmt-clean and compile
> against the real `internal/discover` + `internal/skillsdir`. They were written
> verbatim into `/tmp/skpp_ui_verify` (a copy of the real module) and `go build
> ./...` + `go test ./...` were run: **94 tests pass** (31 discover + 29 skillsdir
> + 11 ui NEW + 23 main), `go vet ./...` clean, `go.mod`/`go.sum` byte-identical to
> the repo. `ui.go` imports exactly fmt/io/strings/discover; `main.go` imports
> fmt/io/os/discover/skillsdir/ui.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: PRECONDITION — confirm dependencies are on disk and green
  - COMMAND: cd /home/dustin/projects/skpp
  - COMMAND: grep -q 'func Index(skillsDir string)' internal/discover/index.go
  - COMMAND: grep -q 'func Find()' internal/skillsdir/skillsdir.go
  - COMMAND: go test ./internal/discover/ ./internal/skillsdir/ >/dev/null && echo "deps green"
  - EXPECT: both symbols exist AND tests pass. If Index/Find are MISSING, T5/T2
            have NOT landed — STOP and let them land first.

Task 1: CREATE internal/ui/ui.go
  - WRITE: the exact content from the Blueprint (File 1).
  - CHECK: `package ui`; imports ONLY fmt/io/strings/internal/discover; PrintList +
           padRight + wrapWords + 3 ANSI const + descWrapWidth; padRight-then-paint;
           TrimSpace on description; (none)/(missing) rules; no os import.
  - GOTCHA: pad BEFORE paint (Gotcha #3). NO os/x-term import (Gotcha #2/#11).
            Do NOT re-sort (Gotcha #6).

Task 2: CREATE internal/ui/ui_test.go
  - WRITE: the exact content from the Blueprint (File 2).
  - CHECK: `package ui`; imports bytes/strings/testing/internal/discover; helpers
           mk + colOf; 11 tests (see success criteria).
  - GOTCHA: NO testify; NO t.Parallel(). Drive color via the useColor param
            (no TTY needed). Compare column positions with colOf.

Task 3: MODIFY main.go (overwrite with File 3)
  - WRITE: the exact content from the Blueprint (File 3) over the existing main.go.
  - CHECK: config has list+noColor; parseArgs has --list/-l + --no-color cases;
           isTerminal package var (ModeCharDevice); run --list branch AFTER --path
           and BEFORE default; imports +discover +ui; version/path/default UNCHANGED.
  - GOTCHA: keep version precedence before --list (Gotcha #10). isTerminal is a
            VAR (Gotcha #8). Empty-store len==0 -> exit 1 (Gotcha #7).

Task 4: MODIFY main_test.go (overwrite with File 4)
  - WRITE: the exact content from the Blueprint (File 4) over the existing main_test.go.
  - CHECK: +io +os imports; withTerminal + writeSkillTree helpers; 8 NEW tests;
           existing 15 tests UNCHANGED.
  - GOTCHA: withTerminal swaps isTerminal + t.Cleanup restore (Gotcha #8). NO
            t.Parallel() on env/cwd/global-state tests. writeSkillTree mirrors
            discover's makeTree but lives in package main.

Task 5: FORMAT + VET + TIDY + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w internal/ui/*.go main.go main_test.go
  - COMMAND: gofmt -l internal/ui/*.go main.go main_test.go   # MUST print nothing
  - COMMAND: go vet ./...                                      # MUST be clean
  - COMMAND: go mod tidy   # EXPECTED: a NO-OP (no new module; all imports stdlib/internal)
  - COMMAND: go build ./...                                    # exit 0
  - COMMAND: go test ./internal/ui/ -v                         # 11 NEW tests PASS
  - COMMAND: go test . -v                                      # 23 main tests PASS (15 old + 8 new)
  - COMMAND: go test ./...                                     # whole module green (94 tests)
  - EXPECT: zero errors, zero vet findings, gofmt silent, go.mod/go.sum unchanged.

Task 6: SMOKE + SCOPE CHECK — Levels 3 + 4 in Validation Loop
  - COMMAND: the Level 3 block (build, --list over a throwaway tree, empty/unresolvable paths).
  - COMMAND: the Level 4 block (scope boundaries + go.mod unchanged + discover/skillsdir untouched).
```

### Implementation Patterns & Key Details

```go
// PATTERN: color decision in main, not in ui (pure formatter).
//   // main.go
//   ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)
//   // ui.go
//   func PrintList(w io.Writer, skills []discover.Skill, useColor bool)
// WHY: tests pass a *bytes.Buffer (never a TTY), so an in-package TTY check would
//      always be false and the color path would be untestable. main owns the TTY
//      check (overrideable via the isTerminal var); ui is a clean leaf that T9
//      (--search) reuses identically. (Verified §1.)

// PATTERN: stdlib TTY detection via ModeCharDevice (no x/term).
//   var isTerminal = func(w io.Writer) bool {
//       f, ok := w.(*os.File); if !ok { return false }
//       fi, err := f.Stat(); if err != nil { return false }
//       return fi.Mode()&os.ModeCharDevice != 0
//   }
// WHY: PRD §4 keeps yaml.v3 the only third-party dep. *bytes.Buffer fails the
//      assertion -> false (deterministic tests); pipe/file -> false (pipe-safe);
//      pty -> true (color). Package VAR so tests swap it. (Verified §2.)

// PATTERN: pad the PLAIN string, THEN paint (ANSI bytes must not affect width).
//   paint(ansiCyan, padRight(r.tag, tagW))   // padRight uses len(plain) — correct
//   padRight(paint(ansiCyan, r.tag), tagW)   // WRONG: len() counts ESC bytes
// WHY: paint returns ESC[36m + s + ESC[0m; its len() is 5+len(s)+4. Padding after
//      painting over-pads and misaligns. Padding before keeps visible alignment.
//      (Verified §3; TestPrintListColumnsAlignedAcrossRows.)

// PATTERN: missing-data placeholders for a manifest-free catalog.
//   name := s.Name; if name == "" { name = "(none)" }
//   desc := strings.TrimSpace(s.Description)
//   if !s.HasFM || desc == "" { desc = "(missing)" }
// WHY: discover includes no-frontmatter skills (HasFM=false) so they resolve by
//      directory. The table must not show blank cells or lie. TrimSpace kills the
//      folded-scalar trailing newline (discover.go contract) so it doesn't force a
//      blank line. (Verified §4; TestPrintListTrimsFoldedScalarNewline.)

// PATTERN: word-wrap at a fixed width; continuation lines keep columns aligned.
//   descLines := wrapWords(r.desc, descWrapWidth)   // [0] always exists
//   // line 0: TAG | NAME | descLines[0]
//   // lines 1..: tagPad | namePad | descLine
// WHY: PRD §6.1 says "(wrapped)". Terminal-width detection is a forbidden dep, so
//      a fixed width (40) is deterministic and testable. tagPad/namePad (spaces)
//      align continuation lines under the description column. (Verified §5/§6.)
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - internal/ui/ is a NEW leaf package: ui.go + ui_test.go, `package ui`.
    imports ONLY fmt/io/strings + internal/discover. Exposes PrintList (+
    unexported padRight/wrapWords). --search (T9) imports and reuses PrintList.
  - main.go now imports internal/discover + internal/ui (in addition to fmt/io/os/
    internal/skillsdir). It consumes Index/Skill (discover) + PrintList (ui) +
    Find/ErrNotFound (skillsdir), all read-only.

go.mod / go.sum (UNCHANGED — verified_facts.md §9):
  - before/after: require gopkg.in/yaml.v3 v3.0.1   (the ONLY third-party dep)
  - ui/main use raw ANSI byte strings + stdlib os.ModeCharDevice -> NO new module.
  - `go mod tidy` is a NO-OP. go.sum unchanged.

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into):
  - P1.M4.T9 (--search): parse `--search <q>` (+`-s`); substring-filter the
    []discover.Skill over RelTag/Name/Description/Keywords; call the SAME
    ui.PrintList(stdout, filtered, useColor). No ui change needed (PRD §6.1
    "same table format as --list"). Exit 1 if no matches.
  - P1.M3.T8 (<tag>/--all): resolve via resolve.Resolve (T7); print Skill.Dir
    (or SourceFile with -f). Independent of ui.
  - P1.M5.T11: add --help/-h to the version precedence tier; implement §6.3
    mutual exclusivity (tag + --list/--search/--all -> exit 2). The --list branch
    added here is a model for those.

NO CHANGES TO:
  - go.mod / go.sum (no new module)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned) / prd_snapshot.md
  - internal/discover/* (S1/S2/T5-owned — Level 4 gate)
  - internal/skillsdir/* (M1-owned — Level 4 gate)
  - any other file (resolve is M3; skills/ is P1.M6.T12; install.sh/README/
    completions are M6)
```

---

## Validation Loop

### Level 1: Format, vet, tidy, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass).
gofmt -w internal/ui/ui.go internal/ui/ui_test.go main.go main_test.go
test -z "$(gofmt -l internal/ui/*.go main.go main_test.go)" \
  || { echo "FAIL: gofmt found unformatted files"; gofmt -d internal/ui/ main.go main_test.go; exit 1; }
echo "gofmt OK"

# Vet the whole module (ui is new; main + discover + skillsdir together).
go vet ./... || { echo "FAIL: go vet ./..."; exit 1; }
echo "go vet OK"

# Tidy: EXPECTED no-op (no new module; all imports stdlib/internal).
go mod tidy
git diff --quiet go.mod go.sum 2>/dev/null \
  && echo "go.mod/go.sum unchanged OK" \
  || { echo "FAIL: go.mod/go.sum changed (must not — no new dep)"; git diff go.mod go.sum; exit 1; }

# Build the whole module (compile check across packages, incl. the new ui pkg).
go build ./... || { echo "FAIL: go build ./..."; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run the NEW ui tests verbosely (11 tests).
go test ./internal/ui/ -v || { echo "FAIL: go test ./internal/ui/ -v"; exit 1; }

# Run main tests verbosely (15 existing + 8 NEW = 23). Confirms existing tests
# still pass AND the new --list/--no-color/TTY tests pass.
go test . -v || { echo "FAIL: go test . -v"; exit 1; }

# Targeted: the load-bearing new tests.
go test ./internal/ui/ -run \
  'TestPrintListColorEmitsANSI|TestPrintListNoColorHasNoANSI|TestPrintListColumnsAlignedAcrossRows|TestPrintListWrapsLongDescription|TestPrintListTrimsFoldedScalarNewline|TestPrintListNoFrontmatterShowsMissing' -v \
  || { echo "FAIL: load-bearing ui tests"; exit 1; }
go test . -run \
  'TestRunListSuccess|TestRunListNoSkillsExit1|TestRunListSkillsDirUnresolvableExit1|TestRunListNoColorFlagSuppressesANSI|TestRunListColorWhenTTY' -v \
  || { echo "FAIL: load-bearing main --list tests"; exit 1; }

# Whole module still green (discover + skillsdir + ui + main = 94 tests).
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: CLI smoke test (end-to-end `--list` through a real binary)

This proves the full `Find() → Index() → PrintList` path through a built binary.
The `skills/` tree is a THROWAWAY (P1.M6.T12 ships the real `skills/example/`); it
is removed afterward.

```bash
cd /home/dustin/projects/skpp

# Build clean.
go build -o skpp . || { echo "FAIL: build"; exit 1; }

# Throwaway store with one skill (so rule 1 via env wins deterministically).
store="$(mktemp -d)"
mkdir -p "$store/example"
printf -- '---\nname: example\ndescription: A demo skill for the catalog.\n---\n# body\n' > "$store/example/SKILL.md"
export SKPP_SKILLS_DIR="$store"

# --list prints the table, exit 0, and (piped to cat) has NO ANSI (non-TTY).
out="$(./skpp --list)"; code=$?
test "$code" = "0" || { echo "FAIL: --list exit=$code; want 0"; rm -rf "$store"; exit 1; }
for want in "TAG" "NAME" "DESCRIPTION" "example" "A demo skill for the catalog."; do
  echo "$out" | grep -q "$want" || { echo "FAIL: --list missing '$want'"; rm -rf "$store"; exit 1; }
done
# Piped (non-TTY) -> no ANSI escapes.
echo "$out" | grep -q $'\x1b' && { echo "FAIL: --list to a pipe must be plain"; rm -rf "$store"; exit 1; } || true
echo "--list OK"

# -l short form works identically.
./skpp -l | grep -q "example" || { echo "FAIL: -l"; rm -rf "$store"; exit 1; }
echo "-l OK"

# Empty store -> exit 1, nothing on stdout, "no skills found" on stderr.
empty="$(mktemp -d)"
SKPP_SKILLS_DIR="$empty" ./skpp --list >/tmp/skpp-list-out 2>/tmp/skpp-list-err
code=$?
test "$code" = "1" || { echo "FAIL: empty store exit=$code; want 1"; rm -rf "$store" "$empty"; exit 1; }
test ! -s /tmp/skpp-list-out || { echo "FAIL: empty store stdout must be empty"; rm -rf "$store" "$empty"; exit 1; }
grep -q "no skills found" /tmp/skpp-list-err || { echo "FAIL: empty store stderr"; rm -rf "$store" "$empty"; exit 1; }
echo "empty-store exit 1 OK"

# Unresolvable store -> exit 1, nothing on stdout, one-line fix on stderr.
( cd "$(mktemp -d)" && unset SKPP_SKILLS_DIR && "$PWD/../../../projects/skpp/skpp" --list 2>/dev/null ) 2>/dev/null
# (the unit test TestRunListSkillsDirUnresolvableExit1 covers this authoritatively;
#  the binary path is environment-dependent, so rely on the unit test for the gate.)

# --no-color is parsed and accepted (combined with --list); exit 0, plain output.
SKPP_SKILLS_DIR="$store" ./skpp --list --no-color | grep -q "example" || { echo "FAIL: --list --no-color"; rm -rf "$store" "$empty"; exit 1; }
echo "--no-color OK"

# Cleanup throwaways.
rm -rf "$store" "$empty" skpp /tmp/skpp-list-out /tmp/skpp-list-err
echo "Level 3 PASS"
```

### Level 4: Scope boundary check (do not regress deps or the module)

```bash
cd /home/dustin/projects/skpp

echo "--- ui.go owns PrintList + padRight + wrapWords (leaf formatter) ---"
grep -nE '^func |^const |^var ' internal/ui/ui.go
test "$(grep -cE '^func PrintList\(' internal/ui/ui.go)" = "1" \
  || { echo "FAIL: ui.go must define PrintList"; exit 1; }
# ui.go imports ONLY fmt/io/strings/discover (NO os, NO term, NO x/...).
! grep -A8 '^import (' internal/ui/ui.go | grep -qE '"os"|"flag"|x/term|golang.org/x' \
  || { echo "FAIL: ui.go must not import os/term/x (pure formatter)"; exit 1; }

echo "--- ui_test.go is white-box package ui with the key tests ---"
grep -q '^package ui' internal/ui/ui_test.go || { echo "FAIL: ui_test.go must be package ui"; exit 1; }
for tn in TestPrintListColorEmitsANSI TestPrintListNoColorHasNoANSI TestPrintListMissingNameShowsNone TestPrintListNoFrontmatterShowsMissing TestPrintListTrimsFoldedScalarNewline TestPrintListWrapsLongDescription TestPrintListColumnsAlignedAcrossRows; do
  grep -q "func $tn" internal/ui/ui_test.go || { echo "FAIL: ui test $tn missing"; exit 1; }
done

echo "--- main.go wiring (--list + --no-color + isTerminal) ---"
grep -q 'list    bool' main.go || { echo "FAIL: config.list missing"; exit 1; }
grep -q 'noColor bool' main.go || { echo "FAIL: config.noColor missing"; exit 1; }
grep -q 'case "--list", "-l"' main.go || { echo "FAIL: --list parseArgs case missing"; exit 1; }
grep -q 'case "--no-color"' main.go || { echo "FAIL: --no-color parseArgs case missing"; exit 1; }
grep -qE 'var isTerminal = func' main.go || { echo "FAIL: isTerminal must be a package var (testable)"; exit 1; }
grep -qE 'ModeCharDevice' main.go || { echo "FAIL: TTY check must use os.ModeCharDevice (stdlib, no x/term)"; exit 1; }
grep -q 'ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)' main.go \
  || { echo "FAIL: --list must call ui.PrintList with the TTY&&!noColor gate"; exit 1; }
# version precedence: version checked before list; path before list; list before default.
awk '/if c\.version/{v=NR} /if c\.path/{p=NR} /if c\.list/{l=NR} END{exit !(v&&p&&l&&v<p&&p<l)}' main.go \
  || { echo "FAIL: order must be version -> path -> list (precedence)"; exit 1; }
# NO forbidden dep in main either.
! grep -qE 'golang.org/x/term' main.go || { echo "FAIL: no x/term allowed"; exit 1; }

echo "--- main_test.go has the new --list tests + helpers ---"
grep -q 'func withTerminal' main_test.go || { echo "FAIL: withTerminal helper missing"; exit 1; }
grep -q 'func writeSkillTree' main_test.go || { echo "FAIL: writeSkillTree helper missing"; exit 1; }
for tn in TestRunListSuccess TestRunListNoSkillsExit1 TestRunListSkillsDirUnresolvableExit1 TestRunListNoColorFlagSuppressesANSI TestRunListColorWhenTTY TestParseArgsListShort TestParseArgsNoColor; do
  grep -q "func $tn" main_test.go || { echo "FAIL: main test $tn missing"; exit 1; }
done

echo "--- deps unchanged ---"
git diff --quiet go.mod go.sum 2>/dev/null \
  && echo "go.mod/go.sum unchanged OK" \
  || { echo "FAIL: go.mod/go.sum changed"; git diff go.mod go.sum; exit 1; }

echo "--- discover & skillsdir UNCHANGED (consumed read-only) ---"
git diff --quiet internal/discover/ internal/skillsdir/ 2>/dev/null \
  && echo "discover/skillsdir unchanged OK" \
  || { echo "FAIL: discover/ or skillsdir/ were modified (forbidden)"; git diff internal/discover internal/skillsdir; exit 1; }

echo "--- no out-of-scope files touched ---"
git status --porcelain | grep -vE 'internal/ui/ui\.go|internal/ui/ui_test\.go|^ M main\.go|^ M main_test\.go|^\?\? internal/ui/' \
  && { echo "FAIL: unexpected files changed (see above)"; exit 1; } \
  || echo "scope OK (only ui.go/ui_test.go new + main.go/main_test.go modified)"

echo "Level 4 PASS (scope + contract respected)"
```

### Level 5: Creative & domain-specific validation (the color/TTY contract, by hand)

```bash
cd /home/dustin/projects/skpp
go build -o skpp .
store="$(mktemp -d)"; mkdir -p "$store/example"
printf -- '---\nname: example\ndescription: A demo skill.\n---\nx\n' > "$store/example/SKILL.md"

# On a real TTY (run this line INTERACTIVELY in your terminal, not under set -e):
#   SKPP_SKILLS_DIR="$store" ./skpp --list            # expect a COLORED table
#   SKPP_SKILLS_DIR="$store" ./skpp --list --no-color # expect a PLAIN table
#   SKPP_SKILLS_DIR="$store" ./skpp --list | cat      # expect PLAIN (piped, non-TTY)

# Automated: confirm the --no-color flag produces byte-identical output to a pipe
# (both are "plain"). This is the §6.2 contract: --no-color == forced-non-TTY.
a="$(SKPP_SKILLS_DIR="$store" ./skpp --list --no-color)"
b="$(SKPP_SKILLS_DIR="$store" ./skpp --list | cat)"
test "$a" = "$b" && echo "no-color == piped (plain) OK" \
  || { echo "FAIL: --no-color and piped output differ"; rm -rf "$store" skpp; exit 1; }

rm -rf "$store" skpp
echo "Level 5 PASS (color/TTY contract holds)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` silent, `go vet ./...` clean, `go build ./...` exit 0
- [ ] Level 2 PASS — `go test ./...` green (94 tests: 31 discover + 29 skillsdir + 11 ui + 23 main)
- [ ] Level 3 PASS — `--list` prints the table (exit 0); empty store exit 1; `--no-color` accepted
- [ ] Level 4 PASS — scope + contract respected; `go.mod`/`go.sum`/discover/skillsdir unchanged
- [ ] Level 5 PASS — `--no-color` output == piped (non-TTY) output (the §6.2 contract)
- [ ] `go mod tidy` is a no-op (no new module — color uses raw ANSI + stdlib TTY check)

### Feature Validation
- [ ] `./skpp --list` / `-l` over a store prints the TAG/NAME/DESCRIPTION table to stdout, exit 0
- [ ] Table columns align across rows of differing tag/name width
- [ ] Long descriptions wrap (multi-line, continuation indented); no line overruns the column
- [ ] `NAME` shows `(none)` when empty; `DESCRIPTION` shows `(missing)` when `!HasFM` or empty
- [ ] `--no-color` suppresses ANSI even on a TTY; TTY (no flag) emits ANSI; pipe/file is plain
- [ ] Empty store → exit 1, empty stdout, `"no skills found"` on stderr
- [ ] Unresolvable store → exit 1, empty stdout, the one-line fix on stderr

### Code Quality / Convention Validation
- [ ] `internal/ui` is a leaf formatter (color opt-in; no terminal inspection; imports fmt/io/strings/discover)
- [ ] `ui_test.go` is white-box `package ui`; no testify; no `t.Parallel()`; color driven via `useColor`
- [ ] `main.go` preserves the M1.T3 version/path/default logic verbatim; `--list` branch ordered after `--path`
- [ ] `main_test.go` keeps all 15 existing tests unchanged; adds helpers + 8 new tests
- [ ] `isTerminal` is a package var (overrideable in tests); uses `os.ModeCharDevice` (no `x/term`)

### Documentation & Scope
- [ ] `PrintList`/`padRight`/`wrapWords` carry godoc explaining the column rules, wrap, and color contract
- [ ] `discover`/`skillsdir` files, `go.mod`/`go.sum`, `PRD.md` all UNCHANGED (Level 4)
- [ ] No `--search`/`--all`/`check`/`--help`/`<tag>`/`skills/`/`install.sh`/`README` created (deferred)

---

## Anti-Patterns to Avoid

- ❌ **Do NOT add `golang.org/x/term`.** PRD §4/§7.3 keep `yaml.v3` the only
  third-party dep. Use stdlib `os.ModeCharDevice` for the TTY check. (Gotcha #2.)
- ❌ **Do NOT inspect the terminal inside the `ui` package.** Color is opt-in via
  `useColor`; main owns the TTY/`--no-color` decision. Otherwise tests (buffer =
  never-a-TTY) can't exercise color and `--search` (T9) can't reuse the package
  cleanly. (Gotcha #1.)
- ❌ **Do NOT pad AFTER painting.** `len(paint(code,s))` counts the escape bytes and
  misaligns columns. Pad the plain string first, then paint. (Gotcha #3.)
- ❌ **Do NOT skip `TrimSpace` on the description.** A folded-scalar `description:
  >` carries a trailing `\n` (discover.go contract); without trim it injects a blank
  line. (Gotcha #4.)
- ❌ **Do NOT detect terminal width.** It needs a forbidden dep/syscall. Wrap at the
  fixed `descWrapWidth = 40`. (Gotcha #5.)
- ❌ **Do NOT re-sort in `ui`.** discover.Index already sorted by `RelTag`; `--search`
  passes a still-sorted slice. ui renders input order as given. (Gotcha #6.)
- ❌ **Do NOT skip the empty-store `len==0` check.** It is REACHABLE (rules 1/2 need
  only an existing dir) and PRD §6.1 mandates exit 1 "if no skills found". (Gotcha #7.)
- ❌ **Do NOT make `isTerminal` a plain func.** It must be a package var so tests can
  override it to exercise the color-enabled path through `run()`. (Gotcha #8.)
- ❌ **Do NOT add `--search`/`--all`/`check`/`--help` here.** Only `--list`/`-l` +
  `--no-color` are in scope. Adding others drifts from T9/M3/M4/M5. (Gotcha #9.)
- ❌ **Do NOT alter the existing `--version`/`--path` logic.** This is the first
  `main.go` edit since M1.T3; preserve version precedence, the byte-exact `--path`
  stdout, and the tolerated default `return 1`. (Gotcha #10.)
- ❌ **Do NOT import `os` in `ui.go`.** ui is a pure formatter taking an `io.Writer`;
  `os` would be an unused dead import and violates the leaf design. (Gotcha #11.)
- ❌ **Do NOT touch `internal/discover/*` or `internal/skillsdir/*`.** They are
  consumed read-only (Level 4 gate asserts unchanged).
