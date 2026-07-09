# System Context — Delta 003

## What this is

A **delta task** against an existing, working Go CLI (`skilldozer`). The codebase is NOT greenfield — it is a complete, building implementation from sessions 001 + 002. All tests currently pass (`go test ./...` = OK).

## The delta

Commit `bbd4e74` ("Add completion subcommand and flip bare-inv to implicit help") modified **only `PRD.md`** (+105/−5 lines). The code is unchanged. Two independent changes must now be implemented:

### Change A — No-args invocation is now implicit `--help` (stdout, exit 0)
- **Before:** `skilldozer` with no args → usage to **stderr**, exit **1**
- **After:** → usage to **stdout**, exit **0** (implicit `--help`), so `skilldozer | grep …` works
- Only no-args is reclassified; genuine failures stay stderr/non-zero (§6.4)

### Change B — New `completion` subcommand (PRD §14.6)
- `skilldozer completion [--shell <name>]` emits shell completion script to **stdout** for `eval "$(skilldozer completion)"`
- Reserved subcommand like `check`/`init` (mutually exclusive)
- Shell detection: `--shell` → `$SKILLDOZER_SHELL` → `basename($SHELL)` → exit 1
- Three completion scripts compiled in via `//go:embed` (stdlib, no new dep)
- Unsupported `--shell` value → exit 2

## Verified ground truth (binary probes)

| Assertion | Current | Required | Status |
|---|---|---|---|
| No-args: stdout contains USAGE | NO | YES | ❌ |
| No-args: exit code | 1 | 0 | ❌ |
| No-args: stderr empty | NO | YES | ❌ |
| `completion --shell bash` emits script | NO | YES | ❌ |
| `completion --shell fish` emits script | NO | YES | ❌ |
| `completion` treated as reserved subcommand | NO (treated as tag) | YES | ❌ |
| `completion --shell tcsh` exits 2 | YES (coincidental — it's unknown flag path) | YES | ⚠️ |
| All existing tests pass | YES | YES | ✅ |

## Repository facts
- **Module:** `github.com/dabstractor/skilldozer` (Go 1.25)
- **Sole dependency:** `gopkg.in/yaml.v3 v3.0.1`
- **Structure:** `main.go` (1127 lines) + `internal/{check,config,discover,resolve,search,skillsdir,ui}/` + `main_test.go` (2737 lines)
- **Built binary:** `./skilldozer` exists (committed in `.gitignore` as `/skilldozer`)
- **Completions:** `completions/{skilldozer.bash,_skilldozer,skilldozer.fish}` (shipped, NOT yet embedding `completion` subcommand)
- **No `//go:embed` directives anywhere** in the codebase yet
- **No `completion` token** anywhere in `main.go` yet
