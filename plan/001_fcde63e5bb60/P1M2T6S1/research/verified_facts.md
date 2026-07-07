# Verified Facts — P1.M2.T6.S1 (`internal/ui` PrintList + `--list`/`--no-color` wiring)

> Empirical verification of the `--list` catalog table, ANSI/`--no-color`, and the
> `Find() → discover.Index() → ui.PrintList()` wiring. Method: a verbatim copy of
> the real module (`go.mod`/`go.sum`/`internal/`/`main.go`/`main_test.go`) into a
> throwaway `/tmp/skpp_ui_verify`, then the four deliverable files added/modified
> and `go build ./...` + `go test ./...` run. **Environment: go1.26.4 linux/amd64.**
>
> **Bottom line:** all 27 tests pass (11 new `ui` + 16 `main`), `go vet ./...`
> clean, `gofmt -l` silent, and **`go.mod`/`go.sum` are byte-identical to the
> repo** (this subtask adds NO third-party dependency). The blueprint source in the
> PRP is the exact, gofmt-clean source that was compiled and tested here.

## §1. The `ui` package is a leaf formatter; color is opt-in (no TTY in the package)

`ui.PrintList(w io.Writer, skills []discover.Skill, useColor bool)` takes an
injected `io.Writer` and a `useColor bool`. The package NEVER inspects stdout or
the terminal itself — the caller (`main`) decides color. This makes the package
fully deterministic in tests (pass `useColor=true`/`false`, capture via
`*bytes.Buffer`) and keeps `internal/ui` a pure leaf library that imports only
`fmt`, `io`, `strings`, and `internal/discover`. Verified: `ui_test.go` asserts
both the color-on (ANSI escapes present) and color-off (no escapes) paths without
touching a real terminal.

## §2. TTY detection is stdlib-only (`os.ModeCharDevice`); NO `golang.org/x/term`

PRD §4/§7.3 keep `gopkg.in/yaml.v3` the **only** third-party dep. Adding
`golang.org/x/term` for `term.IsTerminal` would pull an indirect dep chain and
violate that constraint. The stdlib idiom works:

```go
var isTerminal = func(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok { return false }            // *bytes.Buffer (tests), pipes -> false
    fi, err := f.Stat()
    if err != nil { return false }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

Verified behaviors:
- A `*bytes.Buffer` (the test writer) fails the `*os.File` type assertion →
  `false` → **no color in tests by default, regardless of where `go test` runs.**
  This is the load-bearing property: test output is deterministic without a TTY.
- A regular file / pipe (`os.Stdout` redirected) has `ModeCharDevice == 0` →
  `false` → no color (correct: `skpp --list | cat` is plain).
- A real terminal (`os.Stdout` to a pty) has `ModeCharDevice != 0` → `true` → color.

Because tests pass a buffer, `run()` always computes `useColor=false` there. To
exercise the color path THROUGH `run()`, `main_test.go` overrides the package var
`isTerminal` via a `withTerminal(t, true)` helper (swap + `t.Cleanup` restore). NOT
`t.Parallel`-safe (package-state mutation), but the repo convention is no
`t.Parallel()` on env/cwd/global-state tests. Verified: `TestRunListColorWhenTTY`
and `TestRunListNoColorFlagSuppressesANSI` both pass.

## §3. ANSI padding math: pad the PLAIN string BEFORE painting (else escape bytes corrupt width)

The trap: `paint(code, s)` returns `ESC[36m` + `s` + `ESC[0m`. `len()` of that
counts the escape bytes (5 + len(s) + 4), so padding AFTER painting misaligns
columns. The fix: `padRight(plain, width)` FIRST (uses `len(plain)`, correct), THEN
`paint(code, paddedPlain)`. The trailing pad spaces land INSIDE the SGR run
(invisible) and the reset is at the very end. Verified column alignment in
`TestPrintListColumnsAlignedAcrossRows`: every description starts at the same byte
column as the `DESCRIPTION` header, even with a long tag (`writing/reddit`) setting
a wide TAG column.

## §4. Column rules (the placeholders for missing data)

From the T5 downstream-extension contract + PRD §7.1:
- **TAG** = `Skill.RelTag` (canonical `/`-normalized tag; always set by discover).
- **NAME** = `Skill.Name`; if empty → `"(none)"`.
- **DESCRIPTION** = `Skill.Description`; if `!HasFM` OR blank → `"(missing)"`.

A folded-scalar YAML description (`description: >`) carries a **trailing newline**
(the discover.go S1 contract — copied verbatim, not trimmed). `strings.TrimSpace`
normalizes it so it does NOT inject a blank line into the wrap output. Verified by
`TestPrintListTrimsFoldedScalarNewline` (asserts no `"\n\n"` in output).

## §5. Word-wrap at a FIXED width (terminal-width detection deliberately omitted)

PRD §6.1 says DESCRIPTION is "(wrapped)". skpp wraps at a fixed
`descWrapWidth = 40`:

- Terminal-width detection needs a TIOCGWINSZ ioctl or `golang.org/x/term`
  (forbidden dep, §2). A fixed width keeps output deterministic and testable.
- 40 fits an 80-column terminal beside typical TAG/NAME widths.
- `wrapWords` breaks at spaces; a word longer than the width goes on its own line
  (not split); runs of spaces collapse; empty input → `[""]` so the caller can
  always index `[0]`.
- Continuation lines emit blank TAG/NAME cells (spaces) so columns stay aligned.

Verified: the long example-skill description (one line in source) wraps to 5 data
lines, each ≤ 40 chars beyond the description column (`TestPrintListWrapsLongDescription`).

## §6. Rendered output (proof — `go run` over a 3-skill temp tree, `useColor=false`)

```
TAG             NAME           DESCRIPTION
example         example        Reference example skill for skpp.
                               Demonstrates the required frontmatter
                               and how skpp resolves a tag to an
                               absolute path. Safe to delete once you
                               add real skills.
plain           (none)         (missing)
writing/reddit  reddit-helper  Write a Reddit post.
```

- Rows sorted by `RelTag` (discover.Index sorts; ui preserves input order —
  `TestPrintListPreservesInputOrder`).
- The long tag `writing/reddit` (14 chars) sets the TAG column; `example`/`plain`
  are padded; NAME column padded to `reddit-helper` width.
- `plain` (no frontmatter) → NAME `(none)`, DESCRIPTION `(missing)`.

With `useColor=true` (piped through `cat -v` to show escapes):
```
^[[1mTAG             ^[[0m  ^[[1mNAME         ^[[0m  ^[[1mDESCRIPTION^[[0m
^[[36mexample       ^[[0m  example        Reference example skill for skpp.
```
Header bold (`ESC[1m`), TAG column cyan (`ESC[36m`), each run reset (`ESC[0m`).

## §7. main.go wiring: the FIRST end-to-end `Find() → Index() → ui` path

T5 explicitly deferred this wiring ("NOT added in this subtask"). T6 is where it
lands. The `run()` `--list` branch:
1. `dir, _, err := skillsdir.Find()` → on `ErrNotFound`, `Fprintln(stderr, err)` + exit 1.
2. `skills, err := discover.Index(dir)` → on error, `Fprintln(stderr, err)` + exit 1
   (defensive: the dir could vanish between Find and Index).
3. `len(skills) == 0` → PRD §6.1 exit 1 (`"no skills found in "+dir` to stderr).
4. `ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)` → exit 0.

**The empty-store exit-1 path is REACHABLE and tested** (PRD §6.1 "1 if no skills
found"): `SKPP_SKILLS_DIR` (rule 1) and sibling-of-binary (rule 2) need only an
existing dir — NOT a SKILL.md. So an empty-but-existing store makes `Find()` return
a dir and `Index()` return `[]`, len==0 → exit 1. Verified by
`TestRunListNoSkillsExit1` (stdout empty, stderr message, exit 1).

## §8. `--no-color` is in scope here; `--help`/`--search`/`--all` are NOT

The task title is "ui.PrintList table + ANSI/--no-color + --list wiring", so
`--no-color` (PRD §6.2) is wired now (it is a modifier that combines with `--list`).
`--search` (T9), `--all`/`<tag>` (M3), and `--help` (M5) remain deferred — their
`case` arms and `run` branches are NOT added. Verified: `parseArgs` has exactly the
version/path/list/noColor cases (plus the tolerated default); `run` has the
version/path/list branches (plus the silent default return 1).

## §9. Files touched (and NOT touched) — scope boundary

| File | Action | Verified |
|---|---|---|
| `internal/ui/ui.go` | CREATE | `package ui`; imports fmt/io/strings/discover; `PrintList` + `padRight` + `wrapWords` |
| `internal/ui/ui_test.go` | CREATE | `package ui`; 11 tests; helpers `mk`, `colOf` |
| `main.go` | MODIFY | added `list`/`noColor` to `config`; `case "--list","-l"`,`case "--no-color"`; `isTerminal` var; `run` `--list` branch; imports discover+ui |
| `main_test.go` | MODIFY | added `io`/`os` imports; `withTerminal` + `writeSkillTree` helpers; 8 new tests; existing 15 tests UNCHANGED |
| `go.mod`/`go.sum` | NONE | `diff` vs repo = empty (no new dep) |
| `internal/discover/*` | NONE | consumed read-only (`[]discover.Skill`) |
| `internal/skillsdir/*` | NONE | consumed read-only (`Find()`) |

## §10. Test counts after this subtask

- `internal/discover`: 13 + 6 + 12 = 31 (unchanged).
- `internal/skillsdir`: 29 (unchanged).
- `internal/ui`: 11 NEW.
- `main` (`.`, root): 15 existing + 8 NEW (`TestParseArgsListLong/Short`,
  `TestParseArgsNoColor`, `TestRunListSuccess`, `TestRunListShortFlag`,
  `TestRunListNoSkillsExit1`, `TestRunListSkillsDirUnresolvableExit1`,
  `TestRunListNoColorFlagSuppressesANSI`, `TestRunListColorWhenTTY`) = 23.
- **Total: 94 tests, all green.** `go vet ./...` clean, `gofmt -l` silent.
