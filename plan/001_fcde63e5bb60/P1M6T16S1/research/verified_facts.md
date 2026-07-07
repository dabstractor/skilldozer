# Verified facts — P1.M6.T16.S1 (run PRD §13 acceptance; fix regressions)

Scope: execute EVERY line of PRD §13 verbatim from a clean repo root, plus the
broader Go quality gates (`go test ./...`, `go vet ./...`, gofmt), and the M6
packaging deliverables (install.sh, completions). If any line FAILS, trace it to
the responsible package and FIX THE CODE — never weaken the test. This file is
the GROUND TRUTH captured during PRP research (2026-07-07): the exact outputs and
exit codes a green repo produces, so the implementing agent can detect drift.

## 0. Status at research time: ALL GREEN

All §13 lines pass. All `go test ./...` packages pass. `go vet ./...` clean.
Completions syntax clean. `install.sh` runs end-to-end. **At research time there
are NO regressions.** The task is therefore primarily a VERIFICATION pass: prove
the suite is green, and only fix code if a regression has appeared since (e.g.
a later merge, an env change, a rebuilt binary).

## 1. Environment (verified)

```
Working dir : /home/dustin/projects/skpp
Go          : go1.26.4 linux/amd64   (go.mod says `go 1.25`)
GOPATH      : /home/dustin/go
Module      : github.com/dabstractor/skpp   (single dep: gopkg.in/yaml.v3 v3.0.1)
pi          : 0.80.3 at /home/dustin/.local/bin/pi
~/.local/bin: present, on PATH, holds a `skpp` symlink -> ~/projects/skpp/skpp
bash/zsh/fish: all installed (/usr/bin/{bash,zsh,fish}) for completion syntax checks
git branch  : main ; HEAD = 3c4a68c "Add bash/zsh/fish completions with dynamic tags"
```

## 2. §13 acceptance — ground-truth outputs (run 2026-07-07, repo root)

Each row = a §13 line, the exact command, the verified stdout, and the exit code.
The implementing agent MUST reproduce all of these. Any divergence = a regression.

| # | Command | Verified stdout | rc |
|---|---------|-----------------|----|
| 1 | `go build -o skpp . && echo OK` | `OK` | 0 |
| 2 | `./skpp --version` | `skpp dev` | 0 |
| 3 | `test "$(./skpp --path)" = "$PWD/skills"` | (test passes, no output) | 0 |
| 4 | `./skpp --list` | table: header `TAG NAME DESCRIPTION` + one row `example  example  Reference example skill for skpp...` (wrapped) | 0 |
| 5 | `test -d "$(./skpp example)"` | (test passes) | 0 |
| 6 | `test -f "$(./skpp -f example)"` | (test passes) | 0 |
| 7 | unknown-tag contract: `out=$(./skpp nope 2>/dev/null); rc=$?` | `$out` EMPTY, `$rc`=1 | — |
| 8 | absolute-path contract: `case "$(./skpp example)" in /*) ...` | `/home/dustin/projects/skpp/skills/example` | matches `/*` |
| 9 | `./skpp check` | `OK    example (example)` + `1 skills, 0 errors, 0 warnings` | 0 |
| 10 | pi e2e: `pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded" 2>&1 \| head` | pi output referencing the **example** skill; NO error | 0 |
| 11 | symlink: `mkdir -p /tmp/skpp-bin && ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp && /tmp/skpp-bin/skpp example` | `$PWD/skills/example` (symlink resolves BACK to repo) | 0 |
| 12 | env override: `SKPP_SKILLS_DIR="$PWD/skills" ./skpp example` | `/home/dustin/projects/skpp/skills/example` | 0 |

### Notes on §13 lines (subtle points the agent must NOT trip on)

- **Line 2 (`--version` prints `skpp dev`).** §13 says "prints: skpp <something>".
  Plain `go build -o skpp .` (line 1) injects NO ldflags, so `version` stays its
  default `"dev"`. `skpp dev` satisfies §13. If you instead built via
  `install.sh` (which passes `-X main.version=$(git describe...)`), the value is
  the git tag/sha (observed `skpp 3c4a68c`). EITHER is acceptable; the assertion
  is "non-empty after `skpp `". **DO NOT assert on a specific version string.**
- **Line 1 OVERWRITES the install.sh-built binary.** If `install.sh` ran first
  (binary versioned `3c4a68c`) and you then run `go build -o skpp .` (§13 line 1),
  the repo `./skpp` is rebuilt WITHOUT ldflags -> `--version` flips to `dev`.
  Harmless and expected, but don't be confused if the PATH symlink (`~/.local/bin
  /skpp -> $PWD/skpp`) reports `dev` after running §13 line 1.
- **Line 11 (symlink test) depends on line 1.** `/tmp/skpp-bin/skpp` is a symlink
  to `$PWD/skpp`. That target only exists after `go build -o skpp .` (line 1). Run
  §13 in order. The symlink proves §8 rule 2 (sibling-of-binary) resolves through
  `os.Executable()` + `EvalSymlinks`.
- **Line 10 (pi e2e) is the "NOT auto-discovered" proof.** `--no-skills` disables
  ALL pi auto-discovery; the example skill loads ONLY because of the explicit
  `--skill "$(./skpp example)"`. If pi errored or could not see the skill, the
  store would be leaking into a discovery location (PRD §2 violation) OR the path
  skpp printed was wrong. Observed: pi successfully references the example skill.
- **§13 does NOT directly run `install.sh`.** It manually tests the symlink
  mechanism (line 11) and the env override (line 12). Running `install.sh`
  end-to-end is an EXTRA acceptance the PRP adds (§12.1 contract); see §4 below.

## 3. Go quality gates (ground truth)

```
$ go test ./...
ok  github.com/dabstractor/skpp                 0.014s
ok  github.com/dabstractor/skpp/internal/check   (cached)
ok  github.com/dabstractor/skpp/internal/discover (cached)
ok  github.com/dabstractor/skpp/internal/resolve (cached)
ok  github.com/dabstractor/skpp/internal/search  (cached)
ok  github.com/dabstractor/skpp/internal/skillsdir (cached)
ok  github.com/dabstractor/skpp/internal/ui      (cached)

$ go vet ./...
(clean, exit 0)

$ gofmt -l main.go internal/
(clean — no output = all module source is gofmt-clean)
```

Packages tested (7): `.` (main), `internal/check`, `internal/discover`,
`internal/resolve`, `internal/search`, `internal/skillsdir`, `internal/ui`.
`main_test.go` is ~53KB — extensive unit coverage of the dispatch + exit codes.
The §13 BASH suite is the authoritative INTEGRATION gate; the Go tests are the
unit gate. BOTH must stay green.

### gofmt GOTCHA (the one non-clean file — NOT a regression)

```
$ gofmt -l .   # from repo root, recurses EVERYTHING
plan/001_fcde63e5bb60/P1M6T1S1/research/validate_example_probe.go
```

This file is a RESEARCH PROBE inside `plan/` (untracked planning dir), NOT module
source — it is not imported by anything, has no package declaration that matters,
and `go vet ./...` correctly ignores it. It is NOT a shipped artifact and NOT a
regression. **Run gofmt scoped to the module**: `gofmt -l main.go internal/`
(verified clean). Do NOT "fix" the probe file as part of acceptance; if you
absolutely must, format it, but it does not gate §13. (The `plan/` tree is owned
by the orchestrator/research process.)

## 4. M6 packaging deliverables (ground truth — not literal §13, but contract)

These aren't individual §13 lines but are part of the M6 milestone the acceptance
must leave green:

- **install.sh (§12.1):** `bash -n install.sh` clean. Running it end-to-end:
  prints build banner, picks `~/.local/bin` (present+creatable), refreshes the
  symlink `~/.local/bin/skpp -> $PWD/skpp`, prints `skpp <git-describe>` +
  `$PWD/skills/example` as the verify, exits 0. Idempotent (re-run re-links).
  Builds WITH ldflags so the installed binary reports git version. WORKS.
- **completions (§14):** `bash -n completions/skpp.bash`, `zsh -n completions/_skpp`,
  `fish -n completions/skpp.fish` all clean (exit 0). (Behavioral coverage is the
  T15.S1 PRP's job; this task only confirms they didn't rot.)
- **example skill (§11):** `skills/example/SKILL.md` matches §11 verbatim;
  `skpp check` reports it OK; resolves via `skpp example` / `-f example`.
- **README (§15):** present, 6.5KB. Consistency sweep is T16.S2 (the NEXT task),
  NOT this one — do NOT edit README here.

## 5. §13-line → responsible-package map (for regression tracing)

If a §13 line fails, fix the code in the package(s) below (do NOT touch the test
expectation). All packages are under `internal/` except the entrypoint `main.go`.

| Failing §13 line | Likely responsible package(s) | Function(s) to inspect |
|------------------|-------------------------------|------------------------|
| 1 build          | (build env) `go.mod`, `main.go` imports | none — compile error; read compiler output |
| 2 --version      | `main.go` | `var version`, the `c.version` branch in `run()` |
| 3 --path         | `internal/skillsdir` | `Find()` rule 2 `resolveSibling()` (os.Executable + EvalSymlinks) |
| 4 --list         | `internal/skillsdir.Find`, `internal/discover.Index`, `internal/ui.PrintList` | dispatch `c.list` branch |
| 5 skpp example   | `internal/skillsdir`, `internal/discover`, `internal/resolve.Resolve` | tag-resolution branch in `run()` |
| 6 -f example     | `main.go skillPath()` (file vs dir) | `skillPath()` c.file branch |
| 7 unknown-tag    | `main.go` atomicity + `internal/resolve` | buffered paths + UnknownError; `run()` tag loop |
| 8 absolute path  | `internal/discover.Skill.Dir` (abs), `main.go skillPath` | `Index()` absolutizes Dir; default path |
| 9 check          | `internal/check.Check` | `Check()`, findings/levels, exit logic |
| 10 pi e2e        | path correctness (lines 5+8) OR skill not in discovery loc | if path is right, skill frontmatter (§7.3) |
| 11 symlink       | `internal/skillsdir` rule 2 | `os.Executable()` + `filepath.EvalSymlinks` |
| 12 env override  | `internal/skillsdir` rule 1 | `resolveEnv()` (`os.Getenv` + `os.Stat`) |

## 6. §6-contract spot-checks (NOT in §13 literal text; verify to catch drift)

These behaviors are unit-tested but worth a smoke-check since a regression here
would be a §6 contract violation even if §13's literal lines pass:

```
skpp example example          → 2 lines, input order, exit 0     [multi-tag, §6.1]
skpp --all                    → 1 abs path per skill, exit 0      [§6.1]
skpp --relative --all         → `example` (relTag), exit 0        [§6.2 modifier]
skpp -f --relative example    → `example/SKILL.md`, exit 0        [§6.2 combine]
skpp --bogus                  → empty stdout, stderr msg, exit 2  [unknown flag, §6 header]
skpp foo --list               → stderr msg, exit 2                [§6.3 tags+mode]
skpp check foo                → stderr msg, exit 2                [§6.3 check+tags]
skpp                          → empty stdout, usage to stderr, exit 1 [§6.3 no-args]
skpp check (on a broken store)→ ERROR lines, summary, exit 1      [§9]
```
All verified PASS during research. `check` ERROR-detection was confirmed by
temporarily pointing `SKPP_SKILLS_DIR` at a store with a missing-frontmatter
skill (got `ERROR emptydir ((none)): missing frontmatter block...`, exit 1).

## 7. Hard guardrails for THIS task (non-negotiable)

- **Do NOT weaken any §13 assertion** to make it pass. If a line fails, FIX THE
  CODE in the responsible package. The §13 suite is the authoritative gate.
- **Do NOT edit PRD.md**, `tasks.json`, `prd_snapshot.md`, `.gitignore`, README,
  or any `plan/` file. (README consistency is the NEXT task, T16.S2.) The ONLY
  files this task may modify are Go source (`main.go`, `internal/**/*.go`) or the
  packaging files (`install.sh`, `completions/*`, `skills/example/SKILL.md`) — and
  ONLY if a regression demands it. At research time NONE demands it.
- **Do NOT change the product contract** (§6 CLI matrix, §7 resolution, §8
  location, §9 check, exit codes 0/1/2). Any "fix" that changes behavior is a PRD
  deviation and must be explicitly justified in the PR description.
- **Do NOT introduce new deps** beyond yaml.v3 (PRD §4 invariant). Do NOT add
  `golang.org/x/term` or any terminal library — TTY detection uses
  `os.File.Stat()` ModeCharDevice (main.go `isTerminal`).
- **Re-run the FULL §13 suite after ANY fix** (not just the failing line) to prove
  no new regression. Re-run `go test ./...` + `go vet ./...` too.
- **Capture results**: write a run transcript (the §13 PASS/FAIL per line + exit
  codes) to this research dir as evidence. Do not delete the `skpp` binary (it's
  gitignored per §16) — but ensure `git status` shows NO unintended source churn
  if no fix was needed.
