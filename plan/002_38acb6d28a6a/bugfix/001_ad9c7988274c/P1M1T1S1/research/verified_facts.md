# Verified Facts — P1.M1.T1.S1 (Issue 1: route init check-report to stderr)

Confirmed by reading the built source (`main.go`/`main_test.go` @ HEAD), the bugfix
architecture docs, and the QA report. This subtask is a writer-argument swap plus a
test strengthening — no new data, no new deps.

## 1. The exact buggy emitters (runInit, main.go)

`runInit` is `func runInit(c config, stdout, stderr io.Writer) int` at `main.go:988`.
Its check-report block (`main.go:1031-1053`) currently renders the WHOLE check report
to **stdout** via three `fmt.Fprintf(stdout, ...)` calls:

| Line | Call | Role |
|------|------|------|
| `main.go:1046` | `fmt.Fprintf(stdout, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)` | the per-skill OK line |
| `main.go:1050` | `fmt.Fprintf(stdout, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)` | per-finding (WARN/ERROR) |
| `main.go:1053` | `fmt.Fprintf(stdout, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)` | the summary |

**Fix = change the first argument of those THREE calls from `stdout` → `stderr`.** Nothing
else about them changes (format strings, args, padding `%-5s`, the `continue` on the OK
branch, the return value). This is verified to be a pure writer swap.

## 2. What MUST NOT move (the §6.1 stdout headline)

- `main.go:1026` `fmt.Fprintln(stdout, dir)` — the configured store path. **STAYS on stdout.**
  This is the single line PRD §6.1 guarantees for init ("stdout = The configured store path.")
  so that `STORE="$(skilldozer init --store /path)"` captures a clean value.
- `main.go:1029` `fmt.Fprintf(stderr, "(found via %s)\n", src)` — already on stderr ✓ (no change).
- `main.go:1019-1024` the `Seeded ...`/`Adopted ...` status lines — already on stderr ✓ (no change).

## 3. The standalone `check` subcommand STAYS on stdout (do NOT touch it)

The `if c.check` branch at `main.go:557-584` renders the IDENTICAL report to **stdout**
(lines 577/581/584 — the same three `fmt.Fprintf(stdout, ...)` calls). This is correct for
`skilldozer check` (its report IS its stdout product; `TestRunCheckCleanStore` @ main_test.go:1248
asserts `Contains(got, "OK")` and `"2 skills, 0 errors, 0 warnings"` on stdout). After this fix
the two blocks **intentionally DIVERGE**: init→stderr, check→stdout. Do NOT extract a shared
helper (a `renderReport(w io.Writer, ...)` would defeat the whole point — the divergence IS the
fix, and the existing comment "do not refactor; mirror" still applies to the standalone block).

## 4. Comment edits (Mode A doc — inline only)

Two comments in `runInit` currently state the wrong stream and must be corrected:

- **The (6) block** at `main.go:1031-1033` (the contract-specified one):
  ```
  // (6) `skilldozer check` report on the effective store (PRD §8.2 step 5). Mirrors the
  //     `if c.check` branch render VERBATIM (do not refactor; mirror). Best-effort: a
  //     discover.Index failure is non-fatal (setup succeeded).
  ```
  → update to note the divergence: init renders the report to **STDERR** (PRD §6.1 stdout
  contract; only the store-path headline at 1026 stays on stdout), while the standalone
  `check` subcommand keeps its report on stdout.

- **The runInit-level doc comment** at `main.go:978-980` (consistency follow-on — the
  contract focuses on the (6) block, but this prose is now factually wrong):
  ```
  — and then reports: the configured store path to stdout (PRD §6.1), the `--path` "found
  via" annotation to stderr, and the `check` report to stdout (PRD §8.2 step 5).
  ```
  → change "the `check` report to stdout" → "the `check` report to stderr (PRD §6.1 stdout
  contract)". Cite the §6.1↔§8.2-step-5 tension (§6.1 is authoritative; §6.4 is clean-stdout).

## 5. The test to strengthen (the regression guard)

`TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` at `main_test.go:2325`. Today its
ONLY stdout assertion is `main_test.go:2357`:
```go
if !strings.Contains(out.String(), store) {
    t.Errorf("init stdout=%q; want it to contain the store path %q", out.String(), store)
}
```
**This assertion PASSES DESPITE THE BUG** — stdout contains the store path, just alongside
the whole check report. That is exactly why the bug shipped.

**Strengthen** by replacing/augmenting that single `Contains` with:
1. **Exact stdout** (the authoritative bug-catcher): `out.String() == store+"\n"`.
   (`fmt.Fprintln(stdout, dir)` emits exactly `dir+"\n"`; one line, nothing else.)
2. **No check-report markers on stdout** (defensive, makes intent readable): stdout must not
   contain `skills,`, `OK`, `errors`, `warnings`.
3. **Summary on stderr** (positive confirmation the report moved): `errOut.String()` must
   contain `skills,` (the summary is always emitted when `discover.Index` finds ≥1 skill —
   here the freshly-seeded store has the example skill, so `1 skills, 0 errors, 0 warnings`).

The test's existing setup (a non-existent store under a temp parent → setupStore CREATES +
SEEDS it with the compiled-in example) is exactly right: after init, `discover.Index(store)`
finds the seeded `example` skill and check produces `OK example (example)` + the summary. So
asserting `skills,` on stderr is reliable. Assert on the summary marker, not on `OK`, so the
test stays green if a future template tweak changes the per-skill line (WARN vs OK).

## 6. Collateral check — no other init test asserts check-report content on stdout

Grepped `main_test.go`: the ONLY run-level init test asserting stdout content is the one at
:2325. Every other init test asserts either exit code / exclusivity (stderr `errOut`) or
parse-level fields (`c.init`, `c.initStore`). So strengthening the single test is sufficient;
no other test breaks when the report moves to stderr. `TestRunCheckCleanStore` (:1239) asserts
on the STANDALONE `check` stdout and is unaffected (that branch keeps stdout).

## 7. Exit code + deps invariant

- init exit code is **unchanged** (0 on setup success; check findings never gate init's exit —
  the report is best-effort per the existing comment).
- `go.mod` is Go 1.25, sole dep `gopkg.in/yaml.v3`. This fix touches no imports, adds no deps.
  `go.mod`/`go.sum` stay byte-for-byte identical.

## 8. Scope boundary (what this subtask is NOT)

- It is NOT Issue 2 (init --store missing-value → exit 2; that is P1.M1.T2, a separate
  parseArgs+run() change). This subtask only moves the report and strengthens the test.
- It does NOT touch `resolveStore` (tilde expansion = Issue 5 / P1.M2.T3).
- It does NOT touch the README (the cross-cutting README sweep is P1.M3.T1; the init section
  does not currently describe the report's stream, so no per-subtask README edit is needed).
