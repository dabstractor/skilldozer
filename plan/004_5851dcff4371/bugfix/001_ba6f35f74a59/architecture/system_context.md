# System Context — Skilldozer Bugfix Round 2

## Project Overview

Skilldozer is a standalone Go CLI that resolves skill tags to on-disk directory
paths for `pi --skill "$(skilldozer <tag>)"`. It is a single-binary tool (no
runtime deps beyond yaml.v3) with a hand-rolled argv parser, an embedded
completion system, and a manifest-free discovery chain.

## Codebase Topology (verified)

```
main.go                         — entrypoint: parseArgs(), run() dispatcher, runInit/runCompletion
main_test.go                    — 186 unit/integration tests (parseArgs, run, completion, init)
completions/
  skilldozer.bash               — bash completion (embedded via //go:embed)
  _skilldozer                   — zsh autoload completion (embedded via //go:embed)
  skilldozer.fish               — fish completion (embedded via //go:embed)
internal/
  skillsdir/skillsdir.go        — discovery chain: Find() → findEnv/findConfig/findSibling/findWalkUp
  skillsdir/skillsdir_test.go   — 36 tests (per-rule + Find() combiner)
  config/config.go              — settings sidecar: Load(), Save(), Path(), DefaultStore()
  config/config_test.go         — 15 tests
  discover/                     — skill index: Index(), ParseFrontmatter(), Skill struct
  resolve/                      — tag→skill resolution (canonical/basename/name/alias)
  search/                       — substring search over index
  check/                        — --check validation
  ui/                           — table rendering for --list/--search
```

## Go Module

- Module path: `github.com/dabstractor/skilldozer`
- Go version: 1.24+ (uses `t.Chdir`)
- Sole external dep: `gopkg.in/yaml.v3`

## The Five Issues in This Changeset

| # | Severity | Area | Files Touched |
|---|----------|------|---------------|
| 1 | Major | `--shell` value completion offers skill tags, not `bash zsh fish` | `completions/skilldozer.bash`, `completions/_skilldozer`, `completions/skilldozer.fish` |
| 2 | Major | Configured store whose dir vanished silently falls through to different store | `internal/skillsdir/skillsdir.go`, `internal/skillsdir/skillsdir_test.go` |
| 3 | Minor | `--search`/`--shell` missing value → help/exit 0 (vs `--store` → exit 2) | `main.go`, `main_test.go` |
| 4 | Minor | POSIX `--` end-of-options separator rejected as unknown flag | `main.go`, `main_test.go` |
| 5 | Minor | README version doc inaccurate for plain `go build` | `README.md` |

## Cross-Issue Interconnections

- **Issues 1 & 3** both involve `--shell`: Issue 1 fixes completion-file value
  routing; Issue 3 fixes the parseArgs missing-value exit code. They are
  independent code paths (completion files vs Go parser) but both should land
  for a consistent `--shell` experience.
- **Issues 3 & 4** both modify `parseArgs()` in `main.go`. They touch different
  branches (Issue 3: `case "--search"/"--shell"` missing-value; Issue 4: a new
  `--` token guard before the switch). No overlap, but the implementing agent
  should apply them sequentially to avoid merge conflicts.
- **Issue 2** is self-contained in `internal/skillsdir/`. The only cross-package
  effect is that `Find()` may return a new error string, which all `main.go`
  callers already handle via `fmt.Fprintln(stderr, err); return 1`. No `main.go`
  code change is required for Issue 2.

## Embed Lockstep Constraint (Critical for Issue 1)

The three completion files are embedded at compile time via `//go:embed`:

```go
//go:embed completions/skilldozer.bash
var bashCompletion string
//go:embed completions/_skilldozer
var zshCompletion string
//go:embed completions/skilldozer.fish
var fishCompletion string
```

`TestEmbeddedCompletionsMatchOnDisk` (main_test.go:2995) asserts each embedded var
is byte-identical to its source file. After modifying any completion file, a
`go build` must be run to refresh the embed. The test then validates the
invariant automatically. No separate "rebuild embed" step is needed beyond `go
build` / `go test`.

## Test Infrastructure

- All Go tests run via `go test ./...` from the repo root.
- Tests use `t.Setenv`, `t.TempDir()`, and `t.Chdir()` (Go 1.24+) for isolation.
- `main_test.go` tests the `run()` function via injected `*bytes.Buffer` writers
  (no real I/O), and `parseArgs()` directly (pure function).
- `skillsdir_test.go` tests `findConfig()` directly (unexported, same package)
  AND `Find()` through the public API.
- The `unsetEnvVar` helper sets `SKILLDOZER_CONFIG` to a non-existent ghost file,
  which causes `config.Load` to return `fs.ErrNotExist` → `findConfig` bails at
  the "file missing" branch (line 113), NOT the "store dir absent" branch
  (line 126). This means all-miss tests are safe under the Issue 2 change.
