# Verified Facts — P1.M3.T8.S2 (Modifiers: `--file`/`-f`, `--relative`, `--all`/`-a`)

These facts were established empirically against a **verbatim throwaway copy of the
real module** at `/tmp/skpp_s2_verify` (go1.25). The EXACT source written into the
PRP's Implementation Blueprint (complete `main.go` + the appended `main_test.go`
block) was **copied into the throwaway, gofmt'd, vetted, built, unit-tested, and
smoke-tested against a real built binary** over a `SKPP_SKILLS_DIR` store. Every
load-bearing decision below is backed by a gate output, reproduced at the end.

## 1. What this subtask adds (the 6 edits, all ADDITIVE)

1. `main.go` import: add `"path/filepath"` (stdlib — the ONLY new import; S1 had
   fmt/io/os/strings + internal/{discover,resolve,skillsdir,ui}).
2. `config` struct: add three fields — `all bool`, `file bool`, `relative bool`.
3. `parseArgs`: add three `case` arms — `"--all","-a"`, `"--file","-f"`,
   `"--relative"`. (`--relative` has NO short form per PRD §6.2.)
4. `run`: add the `if c.all { ... }` branch AFTER `--list` and BEFORE `<tag>`.
5. `run` `<tag>` loop: change ONE line — `paths = append(paths, res.Skill.Dir)`
   becomes `paths = append(paths, skillPath(res.Skill, dir, c))`.
6. `main.go`: add the `skillPath(s, skillsDir, c)` helper function (shared formatter).

## 2. Gate output (verbatim, go1.25)

```
gofmt -w main.go main_test.go   → applied (struct fields auto-aligned)
gofmt -l main.go main_test.go   → CLEAN (no output)
go vet ./...                    → CLEAN (no output)
go build ./...                  → OK
go test .                       → ok   github.com/dabstractor/skpp  0.005s
main test COUNT (go test . -v | grep -c '^--- PASS:')  → 53   (was 36; +17)
go test ./...                   → all 5 packages green
```

`go.mod`/`go.sum` byte-identical to the pristine repo (`diff` confirms; the only
import added is `path/filepath`, which is stdlib → no new dependency →
`go mod tidy` is a no-op).

## 3. The `skillPath` helper — the load-bearing new logic

```go
func skillPath(s discover.Skill, skillsDir string, c config) string {
    p := s.Dir                       // default: absolute skill dir (PRD §6.1)
    if c.file { p = s.SourceFile }   // --file: SKILL.md path (== Dir + "/SKILL.md")
    if c.relative {
        if rel, err := filepath.Rel(skillsDir, p); err == nil { p = rel }
    }
    return p
}
```

It is the SHARED formatter for BOTH the `<tag>` loop and the `--all` loop, so the
modifiers behave identically in the two modes (PRD §6.2 header: "combine with tag
resolution or --all"). It does NOT touch `--list` (a table, not paths).

## 4. Binary smoke test (real binary, `SKPP_SKILLS_DIR` = a 2-skill store)

Store: `skills/example/SKILL.md` (name: example) + `skills/writing/reddit/SKILL.md`
(name: reddit-poster).

| Invocation | stdout | rc |
|---|---|---|
| `skpp example` | `…/skills/example` | 0 |
| `skpp -f example` | `…/skills/example/SKILL.md` | 0 |
| `test -f "$(skpp -f example)"` | PASS (§13 gate) | 0 |
| `skpp --relative example` | `example` | 0 |
| `skpp -f --relative writing/reddit` | `writing/reddit/SKILL.md` | 0 |
| `skpp --all` | `…/example` then `…/writing/reddit` (sorted) | 0 |
| `skpp --all --file` | `…/example/SKILL.md` then `…/writing/reddit/SKILL.md` | 0 |
| `skpp --all --relative` | `example` then `writing/reddit` | 0 |
| `skpp --all` (empty store) | (nothing) | 0 |
| `skpp -f example nope` | (nothing) | 1 (atomicity preserved with modifiers) |

Every output matches PRD §6.1 (`--all` sorted by tag), §6.2 (`--file` → SKILL.md,
`--relative` → relative), §6.4 (modifiers do NOT break the nothing-on-stdout-on-
failure contract), and §13 (`test -f "$(./skpp -f example)"`).

## 5. Key decisions (with the gate that backs each)

- **`--file` swaps SourceFile for Dir, one line.** PRD §6.2. `s.SourceFile` is set
  by `discover.BuildSkill` as `filepath.Join(dir, "SKILL.md")` (absolute). Verified:
  `skpp -f example` prints `…/example/SKILL.md` and `test -f` passes (§13 gate).
- **`--relative` uses `filepath.Rel(skillsDir, p)`.** Both args are absolute
  (`s.Dir`/`s.SourceFile` from discover; `skillsDir` from `skillsdir.Find`), and
  `s.Dir` is always UNDER skillsDir (discovered by walking it), so `Rel` always
  succeeds and yields the OS-separator relative path (e.g. `writing/reddit`). The
  err guard is defensive only; on a (theoretical) failure it keeps the absolute
  path. Verified: `--relative writing/reddit` → `writing/reddit`.
- **`--file --relative` COMBINE.** `p` is set to SourceFile first, THEN Rel is
  applied, yielding a relative SKILL.md path (e.g. `writing/reddit/SKILL.md`).
  Verified.
- **`--all` is a NEW mode, not tag resolution.** It walks the already-sorted
  `discover.Index` result (Index sorts by RelTag, PRD §6.1 "sorted by tag") and
  prints each via the SAME `skillPath` helper. No atomicity buffering needed
  (nothing can fail per-skill). Verified: sorted output.
- **`--all` on an empty store → exit 0 (prints nothing).** PRD §6.1 table: `--all`
  is always exit 0, UNLIKE `--list` (exit 1 "if no skills found"). `--all` is a
  scripting command where empty output + exit 0 is the useful shape. Verified.
- **`--all` branch placement: AFTER `--list`, BEFORE `<tag>`.** Groups the named
  modes (path/list/all); `<tag>` is the positional fallback. When `--all` and
  `<tag>` coexist (degenerate), `--all` wins deterministically in S2; P1.M5.T11
  turns `<tag>` + `--all` into exit 2 (§6.3 mutual-exclusivity). Verified
  (`TestRunAllPrintsAllDirsSorted` + the S1 tag tests are unchanged).
- **Modifiers do NOT affect `--list`.** `--list` prints a table; the §6.2 header
  says modifiers combine "with tag resolution or --all" (not `--list`). The
  `--list` branch is untouched. Verified (all 6 existing `--list` tests pass).
- **Modifiers do NOT break §6.4 atomicity.** `skillPath` only reformats the path
  string AFTER `resolve.Resolve` succeeds; the buffered-atomicity loop from S1 is
  unchanged (buffer paths, flush only if `!hadErr`). Verified
  (`TestRunTagFileAtomicity`: stdout EMPTY on a bad tag even with `-f`).

## 6. main.go imports (verified via `go list`)

```
fmt github.com/dabstractor/skpp/internal/discover
    github.com/dabstractor/skpp/internal/resolve
    github.com/dabstractor/skpp/internal/skillsdir
    github.com/dabstractor/skpp/internal/ui
io os path/filepath strings
```

S1's set + `path/filepath`. No third-party dep added. `gopkg.in/yaml.v3` (the only
third-party dep) stays in `go.mod` unchanged.

## 7. Test inventory (53 = 36 prior + 17 new; all PASS)

New tests (all appended; `main_test.go` imports UNCHANGED — bytes/io/os/path/
filepath/strings/testing):

- parseArgs modifiers (6): FileLong, FileShort, RelativeLong, AllLong, AllShort,
  ModifiersInterleave.
- run `<tag>` + modifiers (4): TagFilePrintsSourceFile,
  TagRelativePrintsRelativeDir, TagFileRelativeCombine, TagFileAtomicity.
- run `--all` (7): AllPrintsAllDirsSorted, AllShortFlag, AllFilePrintsAllSourceFiles,
  AllRelativePrintsAllRelative, AllEmptyStoreExit0, AllSkillsDirUnresolvable,
  VersionPrecedenceOverAll.

No `t.Parallel()`. No testify. Reuses `writeSkillTree`/`unsetSkillsEnv`/`version`/
`sampleStore` (the helper S1 added). Relative-path assertions use `filepath.FromSlash`
(cross-platform) since `filepath.Rel` emits OS separators.

## 8. Scope boundary (what is NOT in this subtask)

- No `--search` (M4.T9), no `check` subcommand (M4.T10), no `--help`/exit-2/
  mutual-exclusivity (M5.T11). `--search`/`check` are still captured as tags today
  (a `skpp check` resolves tag "check" → UnknownError → exit 1, unchanged).
- M5.T11 will turn `<tag>` mixed with `--list`/`--search`/`--all` into exit 2. In
  S2 these are TOLERATED: if `--all` + `<tag>`, `--all` wins (deterministic); if
  `--list` + `<tag>`, `--list` wins (checked first). No S2 test asserts the M5
  exit-2 behavior.
- No touch to `internal/*`, `go.mod`, `go.sum`, `.gitignore`, `LICENSE`, `PRD.md`.
  No new files. No `skills/` dir (P1.M6.T12).
