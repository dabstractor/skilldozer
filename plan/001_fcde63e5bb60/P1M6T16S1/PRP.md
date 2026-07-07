# PRP — P1.M6.T16.S1: Run PRD §13 acceptance suite; fix regressions

> **Subtask:** P1.M6.T16.S1 — the FINAL acceptance gate of the whole build
> (build-order step "acceptance"). Run EVERY line of PRD §13 verbatim from a
> clean repo root, plus the Go quality gates (`go test ./...`, `go vet ./...`,
> gofmt) and the M6 packaging smoke (install.sh, completions). **If any line
> fails, fix THE CODE in the responsible package — never weaken the test.**
> Re-run until fully green.
>
> **Scope:** This is VERIFICATION + minimal regression repair. It is NOT a
> feature build and NOT a documentation task (README/docs consistency is the NEXT
> subtask, P1.M6.T16.S2). The ONLY files this task may touch are Go source
> (`main.go`, `internal/**/*.go`) or the packaging files (`install.sh`,
> `completions/*`, `skills/example/SKILL.md`) — and ONLY if a regression demands
> it. At research time the repo is fully green, so the expected outcome is:
> run the suite, capture a PASS transcript, change nothing.
>
> **Why this task exists (contract):** The orchestrator's
> `tasks.json P1.M6.T16.S1.context_scope` mandates it verbatim: "run EVERY
> command in PRD §13... For any failing line, trace to the responsible subtask's
> package and fix (do NOT weaken the test — fix the code). Re-run until all green.
> Also run `go test ./...` and `go vet ./...` clean." The pi end-to-end line
> specifically proves the store is NOT auto-discovered (loads ONLY via `--skill`
> under `--no-skills`).

---

## Goal

**Feature Goal**: Prove — by executing the verbatim PRD §13 suite plus the Go
quality gates — that the shipped `skpp` (binary + example skill + install.sh +
completions) fully satisfies the product contract end-to-end, from a clean clone.
Leave the repo green. Fix any regression that has appeared since the implementing
subtasks landed (none expected; one may surface from a later merge or env change).

**Deliverable**: (1) A green §13 run transcript (every line PASS + exit code),
captured to `plan/001_fcde63e5bb60/P1M6T16S1/research/acceptance_run.md` as
evidence. (2) If (and only if) a regression is found, a minimal code fix in the
responsible package, plus a re-run proving the full suite is green again and
`go test ./...` / `go vet ./...` / gofmt are clean. No new files outside the
research dir unless a fix creates one (it should not — the package layout is
complete).

**Success Definition** (all must hold):
- Every one of the 12 §13 acceptance lines (numbered below) passes with the
  verified expected output/exit code from the research file.
- `go test ./...` → all 7 packages `ok`. `go vet ./...` → clean. `gofmt -l
  main.go internal/` → clean (no output).
- `install.sh` runs end-to-end (build + symlink + verify), exit 0; the installed
  `skpp` resolves `--path` to the repo `skills/`.
- `bash -n` / `zsh -n` / `fish -n` on the three completion files → exit 0.
- If a fix was needed: `git diff` shows ONLY the minimal responsible change; no
  PRD contract (§6/§7/§8/§9, exit codes) altered; the failing line + full suite
  re-pass; no previously-passing test broken.
- `git status` shows no unintended source churn (the `plan/` planning tree and
  `tasks.json` are orchestrator-owned; the `skpp` binary is gitignored per §16).

## User Persona

**Target User**: The maintainer / reviewer who needs a single command proving the
whole repo is shippable before tagging v1.0, and any future contributor who must
confirm a change didn't break the product contract.

**Use Case**: `cd ~/projects/skpp && <acceptance runner>` → a PASS/FAIL report
per §13 line, exit 0 iff the repo is green. Run it before every release and after
every cross-cutting change.

**Pain Points Addressed**: "Did the M6 packaging (install.sh / README /
completions / example skill) silently break resolution or the `$(...)` contract?"
"I rebuilt the binary; does `--path` still find `skills/`?" → the suite answers
all of this in one run.

## Why

- **PRD §13 is the authoritative gate.** Its commands are the implementer's
  acceptance criteria; this subtask exists solely to execute them. A repo that
  passes every §13 line IS the v1.0 deliverable.
- **The §6.4 / `$(...)` contract is load-bearing and silent when it breaks.**
  `pi --skill "$(skpp tag)"` fails LOUDLY (empty `$()`, exit 1) on a bad tag —
  but only if stdout discipline + exit codes are exact. §13 lines 7 & 8 verify
  exactly this. A regression here would silently feed pi a garbage path.
- **The pi e2e line (§13 line 10) proves the §2 "not auto-discovered" constraint.**
  Under `--no-skills`, pi can ONLY see the skill via the explicit `--skill` path
  skpp prints. If the store had leaked into a discovery location, this test would
  not isolate the contract.
- **Cohesion / scope boundary:** Every implementing subtask (M1–M6, 20 subtasks)
  is `Complete`. This task is the catch-all that certifies them together. The
  NEXT task (T16.S2) does the documentation consistency sweep; THIS task must NOT
  edit README/docs (that would collide with T16.S2's scope). Completing this task
  green is the precondition for T16.S2.

## What

User-visible behavior: nothing new. The product already does everything §6/§7/§8/§9
specify. This task RUNS the existing binary through the §13 gauntlet and proves
it. The only thing that changes on disk (if anything) is a minimal code fix for a
regression — and that fix must be behavior-preserving relative to the PRD.

### Success Criteria

- [ ] All 12 §13 acceptance lines pass (see "§13 Acceptance Runner" below for the
      exact commands and verified expected outputs).
- [ ] `go test ./...` all `ok` (7 packages).
- [ ] `go vet ./...` clean.
- [ ] `gofmt -l main.go internal/` clean (the `plan/.../*.go` research probe is
      NOT module source and must NOT block this — scope gofmt to the module).
- [ ] `install.sh` runs end-to-end exit 0; installed `skpp --path` = repo skills.
- [ ] `bash -n completions/skpp.bash && zsh -n completions/_skpp && fish -n
      completions/skpp.fish` all exit 0.
- [ ] §6-contract spot-checks pass (multi-tag, `--all`/`--relative`/`--file`
      combos, unknown flag→exit 2, mixed modes→exit 2, no-args→exit 1, check on a
      broken store→exit 1). See Validation Loop Level 2.
- [ ] A run transcript is written to
      `plan/001_fcde63e5bb60/P1M6T16S1/research/acceptance_run.md`.
- [ ] If a fix landed: `git diff` is minimal, scoped to the responsible package,
      and the FULL suite + Go gates re-pass.

## All Needed Context

### Context Completeness Check

_If someone knew nothing about this codebase, would they have everything needed
to run this acceptance and fix any regression?_ **Yes.** The §13 suite commands
are quoted verbatim below WITH their verified expected outputs and exit codes,
the §13-line→responsible-package map pinpoints where to fix, the regression
diagnosis trees give the root-cause+fix for each likely failure, and the guardrails
say exactly what not to touch. The only codebase files the agent must READ are
`main.go` (dispatch + `skillPath` + `isTerminal` + `run`/`parseArgs`) and, only
if a specific line fails, the single `internal/<pkg>/<pkg>.go` named in the map.

### Documentation & References

```yaml
# MUST READ — the authoritative spec for THIS task (verbatim acceptance lines)
- file: PRD.md
  section: "§13 'Acceptance criteria' — the 12 commands below are quoted from here."
  why: "This is the SOURCE OF TRUTH for what 'green' means. Every line must pass."
  critical: "§13's own prose: 'All of the above must pass. The pi line must show
             the skill loaded with --no-skills.' Do not paraphrase the commands;
             run them verbatim (the runner below does)."

# MUST READ — the orchestrator's exact contract for this subtask
- file: plan/001_fcde63e5bb60/tasks.json
  section: "P1.M6.T16.S1.context_scope"
  why: "States verbatim: run EVERY §13 command; for any failing line, trace to the
        responsible package and FIX THE CODE (do not weaken the test); also run
        go test ./... and go vet ./... clean. This is the task's job description."
  critical: "'do NOT weaken the test — fix the code' is the core rule. A PR that
             changes a §13 assertion to make it pass is a failure of this task."

# MUST READ — ground truth: verified outputs + exit codes + diagnosis map
- docfile: plan/001_fcde63e5bb60/P1M6T16S1/research/verified_facts.md
  why: "Captured during PRP research (2026-07-07) against a GREEN repo: the exact
        stdout + exit code for each §13 line, the §13-line→package map, the gofmt
        probe GOTCHA (plan/ research file is NOT module source), env facts, and
        the hard guardrails. This is the expected-value oracle for the runner."
  critical: "Section 2 = the PASS expectations. Section 5 = where to fix. Section 7
             = what you may NOT do. Read before running."

# READ — the risk areas + the agreed package/type shapes (for diagnosis only)
- docfile: plan/001_fcde63e5bb60/architecture/codebase_state.md
  section: "Risk areas (§13/§18): skills-dir resolution (§8), the §6.4 error
            contract, frontmatter parsing."
  why: "The three areas most likely to regress. §8 sibling-of-binary backs §13
        lines 3 & 11; §6.4 backs lines 5 & 7 & 8; frontmatter backs 4 & 9."

# READ — package/type contracts + data flow (only if diagnosing a fix)
- docfile: plan/001_fcde63e5bb60/architecture/go_architecture.md
  section: "Data flow (parseArgs→Find→Index→Resolve→ui/check); Output discipline
            (§6.4); Exit codes."
  why: "If a §13 line fails, this shows the call chain to trace through."

# READ — the entrypoint dispatch (ONLY file the agent normally needs to open)
- file: main.go
  section: "run() (dispatch + precedence), parseArgs(), skillPath(), isTerminal,
            exclusivityError(), the version var."
  why: "Most §13 lines route through main.go. Exit codes 0/1/2, stdout/stderr
        discipline, the --file/--relative modifiers, and --version all live here."
  critical: "version is `var version = \"dev\"` (NOT const) so install.sh's ldflags
             `-X main.version=...` can override it. Plain `go build -o skpp .`
             (§13 line 1) does NOT pass ldflags, so --version prints `skpp dev`.
             Both satisfy §13 ('skpp <something>'). Do NOT assert a specific
             version string."

# READ — the §8 symlink resolution proof (backs §13 line 11)
- docfile: plan/001_fcde63e5bb60/architecture/verified_symlink_resolution.md
  why: "Proves os.Executable()+EvalSymlinks resolves a cross-dir symlink back to
        the repo. §13 line 11 depends on this. EvalSymlinks is REQUIRED on macOS
        even though redundant on Linux — do not strip it if line 11 fails."
```

### Current Codebase tree (relevant slice — already complete)

```bash
skpp/
├── PRD.md                       # §13 = this task's spec (READ-ONLY)
├── README.md                    # (COMPLETE) do NOT edit here (that's T16.S2)
├── LICENSE                      # MIT
├── go.mod                       # module github.com/dabstractor/skpp ; go 1.25 ; yaml.v3
├── go.sum
├── .gitignore                   # /skpp (binary), /dist, *.test, *.out, .DS_Store
├── main.go                      # entrypoint: parseArgs, run dispatch, skillPath, version
├── main_test.go                 # ~53KB unit tests (dispatch, exit codes, atomicity)
├── install.sh                   # (COMPLETE) build + symlink + verify
├── internal/
│   ├── skillsdir/skillsdir.go   # Find() §8: env → sibling-of-binary → walk-up
│   ├── discover/                # Index() walk + ParseFrontmatter + Skill type
│   ├── resolve/resolve.go       # Resolve() §7.2 precedence + Unknown/Ambiguous errors
│   ├── ui/ui.go                 # PrintList table + ANSI/--no-color
│   ├── search/search.go         # --search substring filter
│   └── check/check.go           # check validation §9 + findings/exit logic
├── completions/                 # (COMPLETE) skpp.bash / _skpp / skpp.fish
└── skills/example/SKILL.md      # the ONE shipped example skill (§11)
```

### Desired Codebase tree with files to be added

```bash
plan/001_fcde63e5bb60/P1M6T16S1/research/
└── acceptance_run.md            # NEW. The §13 PASS/FAIL transcript (evidence).
                                 #   Created by running the §13 runner below;
                                 #   one section per line: command, stdout, rc,
                                 #   PASS/FAIL. Plus the Go-gates + packaging
                                 #   results. This is the task's primary output.
# (No source files are added. A fix, if any, edits an EXISTING file in place.)
```

### Known Gotchas of our codebase & Library Quirks

```bash
# CRITICAL (1): §13 line 1 (`go build -o skpp .`) injects NO ldflags, so
#   `./skpp --version` prints `skpp dev` (the default `var version`). §13 says
#   "prints: skpp <something>" — `dev` satisfies it. install.sh's build DOES pass
#   ldflags (`-X main.version=$(git describe...)`), yielding e.g. `skpp 3c4a68c`.
#   BOTH are acceptable. DO NOT assert a specific version string in the transcript.

# CRITICAL (2): running §13 line 1 OVERWRITES an install.sh-built repo binary.
#   If install.sh ran first (binary versioned by git-describe) and you then run
#   `go build -o skpp .`, the repo `./skpp` is rebuilt WITHOUT ldflags -> --version
#   flips to `dev`. The PATH symlink (~/.local/bin/skpp -> $PWD/skpp) then also
#   reports `dev`. Harmless. Don't be confused; it's expected.

# CRITICAL (3): §13 line 11 (the /tmp/skpp-bin/skpp symlink test) depends on line
#   1 having created $PWD/skpp. Run §13 IN ORDER. The symlink proves §8 rule 2
#   (sibling-of-binary): os.Executable() + filepath.EvalSymlinks walks the symlink
#   back to the repo and finds skills/ as a sibling of the REAL binary.
#   GOTCHA within the gotcha: EvalSymlinks is REDUNDANT on Linux (/proc/self/exe
#   already gives the real path) but REQUIRED on macOS — never strip it.

# CRITICAL (4): §13 line 10 (pi e2e) is the §2 "NOT auto-discovered" proof.
#   `--no-skills` disables ALL pi discovery; the example skill loads ONLY via the
#   explicit `--skill "$(./skpp example)"`. If pi errors or can't see the skill,
#   EITHER skpp printed a wrong path (re-check lines 5/8) OR the store leaked into
#   a discovery location (~/.pi/agent/skills, a node_modules pi.skills, etc.) — the
#   latter is a PRD §2 violation, not a code fix in skpp.

# GOTCHA (5): `gofmt -l .` from repo ROOT flags
#   `plan/001_fcde63e5bb60/P1M6T1S1/research/validate_example_probe.go`. That file
#   is a RESEARCH PROBE in the (untracked) planning tree — NOT module source, not
#   imported, ignored by `go vet ./...`. It is NOT a regression and NOT shipped.
#   Scope gofmt to the MODULE: `gofmt -l main.go internal/` (verified clean). Do
#   NOT "fix" the probe as part of acceptance; the plan/ tree is orchestrator-owned.

# GOTCHA (6): the §13 unknown-tag contract (line 7) is the silent-failure guarantee.
#   `out=$(./skpp nope 2>/dev/null); rc=$?` MUST yield empty $out AND rc=1. If skpp
#   ever prints a partial path to stdout before failing, `pi --skill "$(skpp bad)"`
#   would feed pi garbage. The atomicity lives in main.go's tag-resolution branch
#   (buffer paths, flush ONLY if all resolve). Verify stdout is truly empty.

# GOTCHA (7): `check` exit code reflects ERRORS only, not WARNs. A >1024-char
#   description is a WARN (exit stays 0). A missing name/description, a bad name,
#   a duplicate name, or a no-SKILL.md dir is an ERROR (exit 1). §13 line 9 runs
#   against the clean example skill -> exit 0. To prove ERROR detection, point
#   SKPP_SKILLS_DIR at a temp broken store (see Validation Loop Level 2).

# GOTCHA (8): `skpp` with NO args prints usage to STDERR (not stdout), exit 1
#   (§6.3 parity with get-server-config.sh). `skpp --help` prints the SAME usage to
#   STDOUT, exit 0. Don't confuse the two when checking the "stdout empty" rule.

# GOTCHA (9): the `skpp` binary at repo root is gitignored (§16: `/skpp`). After
#   `go build -o skpp .`, `git status` will NOT show it. That's correct. Do not
#   `git add skpp`. Do not add it to .gitignore (already there).
```

## Implementation Blueprint

There are no data models and no new packages. This task EXECUTES acceptance and
CONDITIONALLY applies a minimal fix. The "tasks" below are ordered: (1) prep, (2)
run the full §13 + Go + packaging gauntlet, (3) if anything fails, diagnose via
the map and fix, (4) re-run + capture transcript. The bulk of the blueprint is the
**§13 Acceptance Runner** (a turnkey script) and the **Regression Diagnosis Trees**.

### §13 Acceptance Runner (turnkey — run verbatim from repo root)

This encodes every PRD §13 line with its verified expected value (from
`research/verified_facts.md` §2). Each block prints `PASS`/`FAIL` and the rc.
Exit non-zero at the end iff any line failed. Pipe the whole thing into the
transcript file (`tee research/acceptance_run.md`).

```bash
#!/usr/bin/env bash
# plan/.../P1M6T16S1 §13 acceptance runner — run from repo root (~/projects/skpp).
set -uo pipefail
pass=0; fail=0
ok()  { echo "PASS  $1"; pass=$((pass+1)); }
bad() { echo "FAIL  $1"; fail=$((fail+1)); }
CHK() { # CHK "label" actual expected
  if [ "$2" = "$3" ]; then ok "$1"; else bad "$1 (got '$2' want '$3')"; fi; }

echo "=== §13 line 1: build ==="
go build -o skpp . && ok "build" || bad "build"

echo "=== §13 line 2: --version ==="
v=$(./skpp --version 2>/dev/null)
case "$v" in skpp\ *) ok "version ('$v')";; *) bad "version ('$v' did not match 'skpp <something>')";; esac

echo "=== §13 line 3: --path = \$PWD/skills (sibling-of-binary) ==="
CHK "--path" "$(./skpp --path 2>/dev/null)" "$PWD/skills"

echo "=== §13 line 4: --list shows example ==="
./skpp --list 2>/dev/null | grep -q '^example' && ok "--list has example row" || bad "--list"

echo "=== §13 line 5: resolve example dir exists ==="
[ -d "$(./skpp example 2>/dev/null)" ] && ok "resolve dir" || bad "resolve dir"

echo "=== §13 line 6: -f example SKILL.md exists ==="
[ -f "$(./skpp -f example 2>/dev/null)" ] && ok "-f file" || bad "-f file"

echo "=== §13 line 7: unknown-tag contract (stdout empty, rc 1) ==="
out=$(./skpp nope 2>/dev/null); rc=$?
if [ -z "$out" ] && [ "$rc" = "1" ]; then ok "unknown-tag (empty stdout, rc1)"; else bad "unknown-tag (out='$out' rc=$rc)"; fi

echo "=== §13 line 8: absolute-path contract ==="
case "$(./skpp example 2>/dev/null)" in /*) ok "absolute path";; *) bad "absolute path";; esac

echo "=== §13 line 9: check (clean store → rc 0) ==="
./skpp check >/dev/null 2>&1; rc=$?
[ "$rc" = "0" ] && ok "check rc0" || bad "check rc=$rc"

echo "=== §13 line 10: pi e2e (--no-skills + explicit --skill) ==="
out=$(pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded" 2>&1 | head)
echo "$out" | grep -qi 'example' && ok "pi e2e (skill referenced, no error)" || bad "pi e2e: $out"

echo "=== §13 line 11: symlink install resolves back to repo ==="
mkdir -p /tmp/skpp-bin && ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp
CHK "symlink→repo" "$(/tmp/skpp-bin/skpp example 2>/dev/null)" "$PWD/skills/example"

echo "=== §13 line 12: SKPP_SKILLS_DIR env override ==="
CHK "env override" "$(SKPP_SKILLS_DIR="$PWD/skills" ./skpp example 2>/dev/null)" "$PWD/skills/example"

echo
echo "§13 RESULT: $pass passed, $fail failed"
[ "$fail" = "0" ]
```

> **Run it as:** `cd ~/projects/skpp && bash <runner> 2>&1 | tee
> plan/001_fcde63e5bb60/P1M6T16S1/research/acceptance_run.md`. The `tee` captures
> the transcript (the task's primary evidence artifact). The runner's own exit
> code is 0 iff every line passed.

### Regression Diagnosis Trees (only consult the matching tree if a line FAILs)

For each §13 line, IF it fails, follow this tree. **Fix the code, not the test.**
Re-run the FULL runner after any fix (a fix for line N can break line M).

**Line 1 (build) fails → compile error.** Read the compiler output. Likely: a
later edit introduced a syntax/type error, an unused import, or referenced a
symbol that was renamed. Fix in the file the compiler names. If it's a `go.mod` /
dependency issue (`yaml.v3`), `go mod tidy` then rebuild. This is NOT a §13-logic
failure; it's "the repo doesn't compile."

**Line 2 (`--version`) fails → `main.go`.** The `c.version` branch in `run()` must
`fmt.Fprintf(stdout, "skpp %s\n", version)` and `return 0`. `var version` must be
a package-scope string (ldflags override). If output is empty/wrong, the dispatch
precedence is wrong: in `run()`, `c.help` is checked first, then `c.version`. If
`--version` is being swallowed by the unknown-flag or no-args branch,
`parseArgs` lost the `case "--version", "-v"` arm.

**Line 3 (`--path` ≠ `$PWD/skills`) fails → `internal/skillsdir` rule 2.** The
binary is at `$PWD/skpp`; `os.Executable()` must return it, `EvalSymlinks` the
real path, `filepath.Dir` → `$PWD`, and `$PWD/skills` must `os.Stat` as a dir.
Root causes: (a) `skills/` was deleted/renamed — restore it (§11 example lives
there); (b) `EvalSymlinks` error swallowed wrongly and `repoDir` is wrong; (c)
rule 1 (env) or rule 3 (walk-up) is wrongly winning — check `SKPP_SKILLS_DIR`
isn't set in your shell, poisoning the test (unset it: `env -u SKPP_SKILLS_DIR`).

**Line 4 (`--list` no example row) fails → discover.Index / ui.PrintList / Find.**
Likely `discover.Index(dir)` returned empty (frontmatter parse failure on the
example skill, or the walk isn't finding `skills/example/SKILL.md`), OR
`ui.PrintList` isn't emitting the TAG column. Check `./skpp check` (line 9) for a
frontmatter ERROR on `example`. If `example` is missing from Index but present on
disk, the `WalkDir` skip logic or the `SKILL.md` filename match is broken in
`internal/discover/index.go`.

**Line 5 (`skpp example` not a dir) fails → resolve / discover / skillsdir.**
Either `Find()` failed (check `--path`/line 3), `Index()` is empty (line 4), or
`resolve.Resolve("example", skills)` returned an error. Run `./skpp example`
WITHOUT redirect to see the stderr error. If it's "unknown tag", the canonical-tag
comparison in resolve is broken (RelTag must be `example`). If stdout has a path
but it's not a dir, `discover.Skill.Dir` wasn't absolutized in `Index`/`newSkill`.

**Line 6 (`-f example` not a file) fails → main.go `skillPath()`.** The `c.file`
branch must set `p = s.SourceFile` (= `s.Dir + "/SKILL.md"`). If line 5 passes
but 6 fails, `SourceFile` is wrong (not appended, or wrong separator). Check
`internal/discover/skill.go` `newSkill` sets `SourceFile` correctly.

**Line 7 (unknown-tag: stdout not empty, or rc≠1) fails → main.go atomicity.**
This is the §6.4 contract. The tag-resolution branch must buffer ALL paths, and
ONLY `fmt.Fprintln(stdout, p)` them after the loop if `hadErr == false`. If stdout
got a partial path, paths are being printed inside the loop before the error is
known. If rc≠1, the `if hadErr { return 1 }` is missing or misplaced. See
`research/verified_facts.md` §7 + `go_architecture.md` "Output discipline".

**Line 8 (not absolute) fails → discover.Skill.Dir / skillPath.** `Index`/
`newSkill` must store `Dir` ABSOLUTE (`filepath.Abs` or built from the absolute
`skillsDir`). `skillPath` default (`c.file` false, `c.relative` false) returns
`s.Dir` untouched. If it's relative, `Dir` was stored relative, or `--relative`
defaulted on wrongly.

**Line 9 (`check` rc≠0) fails → internal/check.** Run `./skpp check` to read the
ERROR. If it flags the EXAMPLE skill (name/description/frontmatter), the shipped
`skills/example/SKILL.md` drifted from §11 — restore it verbatim (do NOT weaken
the check rule). If `check` itself errors out, `internal/check/check.go` `Check()`
or its findings/summary logic is broken.

**Line 10 (pi e2e fails) fails → FIRST re-check lines 5 & 8.** pi can only see the
skill if skpp printed the correct absolute dir. If the path is right and pi STILL
errors, either the SKILL.md frontmatter is malformed (pi requires `name` +
non-empty `description`; run `./skpp check`) OR the store leaked into a pi
discovery location (PRD §2 violation — remove it from `~/.pi/agent/skills` etc.;
NOT a skpp code fix). If pi just doesn't *mention* "example" but loads fine, the
grep is too strict — read the raw `head` output and judge.

**Line 11 (symlink ≠ repo path) fails → internal/skillsdir rule 2.** Same root as
line 3 but through a symlink. `os.Executable()` on `/tmp/skpp-bin/skpp` must
resolve to the symlink target `$PWD/skpp`, then `EvalSymlinks` to the real file,
`Dir` → `$PWD`. If it resolves to `/tmp/skpp-bin` (didn't follow the symlink),
`EvalSymlinks` was skipped or its error handled wrong. (Linux: /proc/self/exe
already gives the real path; macOS NEEDS EvalSymlinks — keep both calls.)

**Line 12 (env override fails) fails → internal/skillsdir rule 1.**
`resolveEnv()` reads `os.Getenv("SKPP_SKILLS_DIR")`, `os.Stat`s it; if an existing
dir, returns it (SourceEnv). Do NOT `EvalSymlinks` the env path (user points
exactly where they want). If line 12 fails but line 3 passes, rule 1 isn't being
tried FIRST in `Find()`, or the env value is being absolutized/cleaned wrongly.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: PREP the environment (so §13 runs from a clean, known state)
  - DO: cd ~/projects/skpp ; confirm git clean-ish (plan/ + tasks.json churn is
        orchestrator-owned and OK; skpp binary is gitignored).
  - DO: unset SKPP_SKILLS_DIR in the test shell (env -u SKPP_SKILLS_DIR) so §13
        line 3/11 exercise rule 2 (sibling-of-binary), NOT rule 1 (env). Line 12
        sets it explicitly. GOTCHA 3 above.
  - DO: confirm go, pi, bash/zsh/fish present (see verified_facts.md §1).
  - VERIFY: `git log --oneline -1` shows a real commit (HEAD = 3c4a68c at research
            time); the binary at repo root may be stale — §13 line 1 rebuilds it.

Task 2: RUN the full gauntlet (§13 + Go gates + packaging smoke)
  - RUN: the "§13 Acceptance Runner" script above; capture to
         plan/.../P1M6T16S1/research/acceptance_run.md via `tee`.
  - RUN: `go test ./...` (expect 7 `ok` lines), `go vet ./...` (clean),
         `gofmt -l main.go internal/` (clean — GOTCHA 5: do NOT run `gofmt -l .`).
  - RUN: `bash install.sh` (exit 0; refreshes ~/.local/bin/skpp symlink; prints
         a git-versioned verify). Then `skpp --path` (on PATH) == repo skills.
  - RUN: `bash -n completions/skpp.bash && zsh -n completions/_skpp &&
         fish -n completions/skpp.fish` (all exit 0).
  - RUN: the §6-contract spot-checks (Validation Loop Level 2) — optional but
         recommended to catch contract drift the literal §13 lines don't cover.
  - EXPECT: everything green (verified at research time). If so → Task 4 (skip 3).

Task 3: IF any line failed → diagnose + fix the CODE (do NOT weaken the test)
  - DIAGNOSE: open the matching "Regression Diagnosis Tree" above; trace to the
              responsible package (verified_facts.md §5 map).
  - FIX: minimal, scoped to ONE file/function. Preserve the PRD contract exactly
         (§6 CLI matrix, §7 resolution, §8 location, §9 check, exit codes 0/1/2).
         No new deps (yaml.v3 only). No `golang.org/x/term` (TTY detection uses
         os.File.Stat ModeCharDevice in main.go isTerminal).
  - DO NOT: edit PRD.md, tasks.json, prd_snapshot.md, .gitignore, README.md, or
            any plan/ file. README consistency is T16.S2. The plan/ tree + the
            validate_example_probe.go research file are orchestrator-owned.
  - RE-RUN: the FULL §13 runner (not just the fixed line) + `go test ./...` +
            `go vet ./...` + gofmt. A fix for line N must not break line M or any
            unit test. Iterate until all green.

Task 4: CAPTURE the transcript + finalize
  - WRITE: plan/001_fcde63e5bb60/P1M6T16S1/research/acceptance_run.md with the
           §13 PASS/FAIL per line (stdout + rc), the Go-gate results, the
           packaging smoke results, and — if a fix landed — a short "Regression
           found + fix" note (root cause, file changed, why it's contract-safe).
  - VERIFY: `git status` shows NO unintended source churn. If no fix was needed,
            the only new artifact is the research/acceptance_run.md transcript
            (and it lives under plan/, which is orchestrator-owned/untracked).
  - CONFIRM: the §13 runner's final line is "§13 RESULT: N passed, 0 failed" and
             its exit code is 0.
```

### Implementation Patterns & Key Details

```bash
# Pattern: how to assert the §6.4 silent-failure contract correctly (line 7).
#   Capture stdout and rc SEPARATELY; stderr is redirected away. The assertion is
#   BOTH "stdout empty" AND "rc 1". Either alone is insufficient.
out=$(./skpp nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo PASS || echo FAIL

# Pattern: the symlink test (line 11) must point at the REPO binary that §13 line
#   1 built, and assert the RESULT equals the repo path (proves resolution, not
#   just "ran"). Use CHK with the exact expected absolute path.
mkdir -p /tmp/skpp-bin && ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp
[ "$(/tmp/skpp-bin/skpp example)" = "$PWD/skills/example" ]

# Pattern: prove `check` ERROR-detection without polluting the shipped store.
#   Point SKPP_SKILLS_DIR at a TEMP broken store (outside the repo), assert exit 1,
#   then DELETE it. NEVER commit a second skill or a broken skill (PRD §2.4/§17).
mkdir -p /tmp/bk/emptydir && printf 'no frontmatter' > /tmp/bk/emptydir/SKILL.md
SKPP_SKILLS_DIR=/tmp/bk ./skpp check >/dev/null 2>&1; [ $? = 1 ] && echo "check detects ERRORs OK"
rm -rf /tmp/bk
```

### Integration Points

```yaml
NO INTEGRATION CHANGES:
  - This task changes NO config, NO routes, NO migrations, NO env vars, NO build
    flags. It runs the EXISTING product against the EXISTING spec.
  - DATABASE: none (manifest-free; no DB).
  - CONFIG: none new. SKPP_SKILLS_DIR is an EXISTING §8 rule; the runner sets it
    only for line 12 and leaves the env otherwise clean (unset) for lines 3/11.
  - DEPS: none added. yaml.v3 remains the only third-party dep (PRD §4).

FILES THIS TASK MAY TOUCH (only if a regression demands it):
  - main.go                         # dispatch / skillPath / version / isTerminal
  - internal/skillsdir/skillsdir.go # Find() §8 rules
  - internal/discover/*.go          # Index / ParseFrontmatter / Skill
  - internal/resolve/resolve.go     # Resolve() §7.2
  - internal/ui/ui.go               # PrintList
  - internal/search/search.go       # --search filter
  - internal/check/check.go         # check §9
  - install.sh, completions/*       # only if a packaging regression is found
  - skills/example/SKILL.md         # only if it drifted from §11 (restore verbatim)

FILES THIS TASK MUST NOT TOUCH:
  - PRD.md, plan/**/prd_snapshot.md, plan/**/tasks.json  (READ-ONLY / orchestrator)
  - .gitignore                                            (already correct per §16)
  - README.md                                             (T16.S2 owns docs sync)
  - plan/001_fcde63e5bb60/**/research/validate_example_probe.go (NOT module source)
  - Any other plan/ research file
```

## Validation Loop

### Level 1: Build & Static Gates (run first; fastest feedback)

```bash
cd ~/projects/skpp
go build -o skpp . && echo "build OK"        # §13 line 1
go vet ./... && echo "vet OK"                 # orchestrator-mandated; must be clean
gofmt -l main.go internal/ && echo "fmt OK"   # scoped to MODULE (GOTCHA 5); empty = clean
test -f skills/example/SKILL.md && echo "example skill present"
# Expected: all four echo. gofmt prints NOTHING before "fmt OK" (clean).
# If `gofmt -l .` (root) shows plan/.../validate_example_probe.go, IGNORE it —
# it is a research probe, not module source (verified_facts.md §3).
```

### Level 2: §13 Acceptance + §6-Contract Spot-Checks (the core gate)

```bash
cd ~/projects/skpp
# (a) Run the turnkey runner; capture the transcript.
bash <the §13 Acceptance Runner above> 2>&1 | \
  tee plan/001_fcde63e5bb60/P1M6T16S1/research/acceptance_run.md
# Expected: "§13 RESULT: 12 passed, 0 failed" and runner exit 0.

# (b) Go unit gates (orchestrator-mandated).
go test ./...        # expect 7 `ok` lines, 0 failures
go vet ./...         # clean

# (c) §6-contract spot-checks (not literal §13 lines; catch contract drift).
./skpp example example | wc -l | grep -qx 2 && echo "multi-tag OK"        # 2 lines, input order
./skpp --all >/dev/null && echo "--all OK"
./skpp --relative --all | grep -qx example && echo "--relative --all OK"
./skpp -f --relative example | grep -qx 'example/SKILL.md' && echo "-f --relative OK"
./skpp --bogus >/dev/null 2>&1; [ $? = 2 ] && echo "unknown-flag exit2 OK"
./skpp foo --list >/dev/null 2>&1; [ $? = 2 ] && echo "tags+mode exit2 OK"
./skpp check foo >/dev/null 2>&1; [ $? = 2 ] && echo "check+tags exit2 OK"
out=$(./skpp 2>/dev/null); [ -z "$out" ] && echo "no-args empty-stdout OK"
# Expected: every line prints "... OK".
```

### Level 3: Installation & Packaging Smoke (M6 deliverables)

```bash
cd ~/projects/skpp
# install.sh end-to-end (§12.1 contract — NOT a literal §13 line, but M6 ships it).
bash install.sh >/tmp/skpp-install.log 2>&1; echo "install.sh exit=$?"
grep -Eq 'Linked: .*\.local/bin/skpp' /tmp/skpp-install.log && echo "symlink created OK"
grep -Eq 'skills/example' /tmp/skpp-install.log && echo "verify line printed OK"
# Installed (on PATH) skpp must resolve --path to the repo skills (sibling-of-binary).
command -v skpp && [ "$(skpp --path)" = "$PWD/skills" ] && echo "installed --path OK"
# Completions syntax (M6/T15 deliverable — must not have rotted).
bash -n completions/skpp.bash && zsh -n completions/_skpp && fish -n completions/skpp.fish \
  && echo "completions syntax OK"
# Expected: install.sh exit 0; symlink + verify lines present; installed skpp resolves;
#           all three completion files syntax-clean.
# Cleanup the /tmp symlink from §13 line 11 if you want a pristine state (optional).
rm -f /tmp/skpp-bin/skpp; rmdir /tmp/skpp-bin 2>/dev/null || true
```

### Level 4: End-to-End pi Integration (the §2 "not auto-discovered" proof)

```bash
cd ~/projects/skpp
# §13 line 10 verbatim — the canonical proof that skills load ONLY via --skill.
pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
# Expected: pi references the `example` skill (e.g. confirms it is loaded at the
#           .../skills/example/SKILL.md path) and produces NO error. Under
#           --no-skills, pi's discovery is OFF, so the ONLY way pi sees the skill
#           is the explicit --skill path skpp printed. This isolates the §2 contract.
# If pi errors: re-check §13 lines 5/8 (path correctness) and run `./skpp check`
# (frontmatter). If the path + frontmatter are fine, the store may have leaked into
# a pi discovery location (~/.pi/agent/skills, a node_modules pi.skills entry) — that
# is a PRD §2 violation to REMOVE, not a skpp code fix.
```

## Final Validation Checklist

### Technical Validation

- [ ] Level 1 passed: `go build`, `go vet ./...`, `gofmt -l main.go internal/` all clean.
- [ ] Level 2 passed: the §13 runner reports "12 passed, 0 failed" and exits 0;
      `go test ./...` 7 `ok`; all §6 spot-checks print "... OK".
- [ ] Level 3 passed: `install.sh` exit 0 + symlink/verify lines; installed `skpp
      --path` == repo skills; all three completions syntax-clean.
- [ ] Level 4 passed: pi e2e references the example skill under `--no-skills`,
      no error.
- [ ] The transcript is written to
      `plan/001_fcde63e5bb60/P1M6T16S1/research/acceptance_run.md`.

### Feature Validation (the §13 contract — every box is a PRD requirement)

- [ ] `go build -o skpp . && echo OK` (§13).
- [ ] `./skpp --version` prints `skpp <something>` (§13).
- [ ] `test "$(./skpp --path)" = "$PWD/skills"` (§13, §8 sibling-of-binary).
- [ ] `./skpp --list` shows the example skill (§13, §6.1).
- [ ] `test -d "$(./skpp example)"` (§13, §7 resolution).
- [ ] `test -f "$(./skpp -f example)"` (§13, §6.2 `--file`).
- [ ] Unknown tag → NOTHING on stdout, exit 1 (§13, §6.4 atomicity).
- [ ] Resolved path is ABSOLUTE by default (§13, §6.1/§6.2).
- [ ] `./skpp check` exit 0 on the clean store (§13, §9).
- [ ] pi e2e loads the skill via `--no-skills --skill "$(./skpp example)"` (§13, §2).
- [ ] `/tmp/skpp-bin/skpp example` resolves back to `$PWD/skills/example` (§13, §8.2).
- [ ] `SKPP_SKILLS_DIR="$PWD/skills" ./skpp example` works (§13, §8.1).

### Code Quality Validation (only relevant IF a fix landed)

- [ ] The fix is minimal and scoped to the responsible package/file named in the
      diagnosis tree (verified_facts.md §5).
- [ ] No PRD contract altered (§6/§7/§8/§9, exit codes 0/1/2).
- [ ] No new third-party dependency (yaml.v3 only; no `golang.org/x/term`).
- [ ] No previously-passing §13 line or `go test` case broken (full re-run green).
- [ ] The §13 assertions were NOT weakened to force a pass.
- [ ] `git diff` is reviewable in one sitting; behavior change (if any) is
      explicitly justified in the transcript + PR description.

### Documentation & Deployment

- [ ] README.md was NOT edited (that is T16.S2's scope).
- [ ] No new doc files created (this is verification, not authoring).
- [ ] Any regression + fix is recorded in `research/acceptance_run.md` (root cause,
      file, contract-safety rationale) so T16.S2 and reviewers can see it.

---

## Anti-Patterns to Avoid

- ❌ **Do NOT weaken a §13 assertion to make it pass.** If a line fails, FIX THE
  CODE. Changing the test expectation is a failure of this task (and of the PRD).
- ❌ Do NOT run `gofmt -l .` from repo root and treat the `plan/.../validate_
  example_probe.go` line as a failure — it is a research probe, not module source.
  Scope gofmt to `main.go internal/`.
- ❌ Do NOT leave `SKPP_SKILLS_DIR` set in your shell while running §13 lines 3/11 —
  that makes rule 1 (env) win instead of rule 2 (sibling-of-binary), masking a
  real resolution regression. `env -u SKPP_SKILLS_DIR` for those lines.
- ❌ Do NOT assert a specific `--version` string. Plain `go build` yields `skpp dev`;
  install.sh yields `skpp <git-describe>`. Both satisfy §13 ("skpp <something>").
- ❌ Do NOT rebuild the binary with `go build -o skpp .` AFTER `install.sh` and then
  wonder why `--version` flipped to `dev` — that's GOTCHA 2 (expected).
- ❌ Do NOT edit README.md, PRD.md, tasks.json, prd_snapshot.md, .gitignore, or any
  `plan/` file. README sync is T16.S2; the rest are orchestrator/owner-controlled.
- ❌ Do NOT commit the `skpp` binary (gitignored per §16) or a temp/broken skill
  used only to prove `check` ERROR-detection.
- ❌ Do NOT re-run only the single fixed §13 line after a fix — re-run the FULL
  runner + `go test ./...` + `go vet ./...`. A fix can break another line.
- ❌ Do NOT strip `filepath.EvalSymlinks` from skillsdir rule 2 to "simplify" — it
  is redundant on Linux but REQUIRED on macOS (verified_symlink_resolution.md);
  §13 line 11 relies on the symlink following it.
- ❌ Do NOT treat pi's e2e output not literally containing "example" as a hard FAIL
  without reading it — judge whether pi successfully loaded the skill (the contract
  is "references the example skill / does not error", per §13's own prose).

---

## Confidence Score

**9.5 / 10.** This is a verification task against a spec (§13) whose every line I
ran during research and confirmed GREEN, with the exact verified outputs and exit
codes embedded in the turnkey runner and in `research/verified_facts.md`. The
§13-line→responsible-package map and the per-line diagnosis trees make any
regression that has since appeared cheap to localize and fix. The 0.5 residual
risk: (a) a regression introduced after research time (2026-07-07) that the agent
must diagnose from the trees; and (b) the pi e2e line's "references the example
skill" is a soft textual judgment (mitigated by reading the raw `head` output and
the explicit `--no-skills` isolation, which is the real §2 proof). No new feature
is built, no contract is changed, and the expected outcome (run green, capture
transcript, change nothing) is fully specified.
