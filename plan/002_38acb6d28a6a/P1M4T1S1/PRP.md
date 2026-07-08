name: "P1.M4.T1.S1 — Execute §13 acceptance end-to-end and remediate any failure"
description: |
  Verification / acceptance-gate task. Run the full PRD §13 acceptance suite verbatim
  in an isolated temp HOME/XDG/SKILLDOZER_CONFIG, capture a green transcript, and
  grep-confirm the four compliant files (.gitignore / LICENSE / go.mod / install.sh).
  This is the Definition of Done for the entire phase. It OWNS the green transcript,
  NOT new features — any failing assertion routes a fix to its owning Mode A subtask.

---

## Goal

**Feature Goal**: Produce a verified-green PRD §13 acceptance transcript and a one-line-per-check confirmation that `.gitignore`, `LICENSE`, `go.mod`, and `install.sh` are PRD-compliant. The entire §13 script must pass verbatim from an isolated state (temp HOME / XDG_CONFIG_HOME / SKILLDOZER_CONFIG so the config block cannot mutate the dev environment). The pi end-to-end line must show the example skill loaded under `--no-skills --skill` (proving reliance on the explicit `--skill` path, never auto-discovery).

**Deliverable**: Two artifacts:
1. `plan/002_38acb6d28a6a/acceptance_transcript.txt` — the captured stdout/stderr/rc for every §13 step (the green transcript).
2. A one-line-per-check confirmation (appended to the transcript or in a sibling `compliant_files.txt`) that the four compliant files pass their grep gates.

**Success Definition**: (a) Every assertion in the §13 script prints its `OK` marker; (b) the transcript file exists and is non-empty; (c) the four compliant-file greps all print `OK`; (d) the dev environment (the real `~/.config/skilldozer`, the real `~/projects/skilldozer`) is byte-identical before and after the run (isolation worked); (e) no doc file was patched ad hoc (README/`init` issues route to P1.M4.T2.S1, code issues route to the owning Mode A subtask).

## User Persona (if applicable)

**Target User**: The plan orchestrator + any future contributor who needs proof the phase is done.

**Use Case**: Run the canonical acceptance suite as the gate before declaring P1 (the §8 config model + drift sync plan) complete.

**User Journey**: `bash plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh` → reads `acceptance_transcript.txt` → sees all `OK` markers → declares the phase green.

**Pain Points Addressed**: Today there is no captured, reproducible proof that §13 passes; a future regression (a re-planned subtask touching `skillsdir.go` or `main.go`) would have no baseline to compare against.

## Why

- **Definition of Done for the phase.** PRD §13 (h2.12) says "the implementer must verify all pass." P1.M1/M2/M3 landed the config model, `init`, the example-skill rename, and the completion drift. This task is the only place the WHOLE end-to-end story is asserted together, including the isolation block (§8 config) that the per-subtask unit tests only approximate.
- **The new config-specific gates live ONLY here.** The unit tests in `skillsdir_test.go` / `main_test.go` prove the functions; §13 proves the **binary** does the right thing with a real temp HOME, a real `SKILLDOZER_CONFIG` file on disk, and real cwd `cd`-arounds. These four assertions are NEW this phase and have no other home:
  - `grep -q 'run \`skilldozer init\`' err` (unconfigured hint)
  - `init --store <dir>` creates store + writes `config.yaml` (non-interactive init)
  - `--path` shows the store when config is set (config-rule-wins)
  - `--path` shows `SKILLDOZER_SKILLS_DIR` when env+config both set (env-beats-config)
- **Guards the Phase-1 contract.** A green transcript frozen now is the baseline a future Phase-2 (releases, go-install hardening) can diff against.
- **Out of scope (explicit):** this task writes ZERO new features and patches ZERO docs ad hoc. If the README is found non-compliant, the fix goes to P1.M4.T2.S1; if code is non-compliant, the fix goes back to the relevant Mode A subtask (skillsdir / main.go / completions). This task's deliverable is the **transcript**, plus a one-line routing note per failure if any occur.

## What

A single bash script (run once) that:

1. **Builds** the binary from a clean `go build -o skilldozer .`.
2. Runs the **discovery + path + error-contract + validation** block (steps 2-9 of §13) against the repo's own `skills/` sibling.
3. Runs the **pi end-to-end line** and greps the example skill's name out of pi's output (proving `--no-skills --skill` loaded it).
4. Runs the **symlink + env-override** block.
5. Runs the **config + first-run isolation block** in a temp HOME / XDG / SKILLDOZER_CONFIG (the `cd /tmp/skilldozer-iso` section).
6. **Grep-confirms** the four compliant files (`.gitignore` 5 patterns, `LICENSE` MIT line 1, `go.mod` module + single `yaml.v3` require, `install.sh` ldflags + symlink + `skilldozer example` verify-cmd).
7. Captures stdout/stderr/rc for **every** step into `plan/002_38acb6d28a6a/acceptance_transcript.txt`.

### Success Criteria

- [ ] `go build -o skilldozer .` exits 0.
- [ ] `./skilldozer --version` prints `skilldozer <something>`.
- [ ] `test "$(./skilldozer --path)" = "$PWD/skills"` passes (sibling-of-binary rule; the `(found via …)` line is on **stderr**, so `$()` captures stdout only).
- [ ] `./skilldozer --list` shows the `example` skill.
- [ ] `test -d "$(./skilldozer example)"` passes (resolves to a real dir).
- [ ] `test -f "$(./skilldozer -f example)"` passes (`-f` prints the SKILL.md path).
- [ ] unknown-tag contract: `out=$(./skilldozer nope 2>/dev/null); rc=$?` → `[ -z "$out" ] && [ "$rc" = "1" ]` → prints `unknown-tag contract OK`.
- [ ] absolute-path contract: `case "$(./skilldozer example)" in /*) echo "absolute OK";; *) echo "FAIL"; exit 1;; esac`.
- [ ] `./skilldozer check` exits 0 and reports the example as OK.
- [ ] pi line: `pi --no-skills --skill "$(./skilldozer example)" -p "briefly confirm the example skill is loaded"` references the example skill in its output (grep `example`), no error.
- [ ] symlink: `/tmp/skilldozer-bin/skilldozer example` resolves back to `$PWD/skills/example`.
- [ ] env override: `SKILLDOZER_SKILLS_DIR="$PWD/skills" ./skilldozer example` resolves.
- [ ] **unconfigured-hint**: isolated run exits 1 and `grep -q 'run \`skilldozer init\`' err` matches → prints `unconfigured-hint OK`.
- [ ] **non-interactive init**: `SKILLDOZER_CONFIG=<iso>/cfg.yaml ./skilldozer init --store <iso>/store` → store created (`test -d`) AND config written (`grep -q 'store: <iso>/store' cfg.yaml`).
- [ ] **config rule wins**: `SKILLDOZER_CONFIG=<iso>/cfg.yaml ./skilldozer --path | grep -q <iso>/store`.
- [ ] **env beats config**: `SKILLDOZER_SKILLS_DIR=<iso>/store SKILLDOZER_CONFIG=<iso>/cfg.yaml ./skilldozer --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR`.
- [ ] `.gitignore` contains all 5 §16 patterns: `/skilldozer`, `/dist`, `*.test`, `*.out`, `.DS_Store`.
- [ ] `LICENSE` line 1 is exactly `MIT License`.
- [ ] `go.mod` module is `github.com/dabstractor/skilldozer` with exactly one `require` = `gopkg.in/yaml.v3 v3.0.1`.
- [ ] `install.sh` has `go build` + `-ldflags` with `-X main.version`, a `ln -s` symlink, and echoes/runs `skilldozer example`.
- [ ] Transcript file `plan/002_38acb6d28a6a/acceptance_transcript.txt` exists and contains every step's output.

## All Needed Context

### Context Completeness Check

**Pass.** The §13 script was run verbatim on 2026-07-07 against the current `main` (binary rebuilt `go build -o skilldozer .`): **15/15 gates PASS**, all four compliant files PASS (full evidence in `research/acceptance_run.md` and `research/compliant_files.md`). The pi line was independently exercised and prints the example skill's name/description/location. The exact §13 script source is `PRD.md` lines 343-415 (heading h2.12). The isolation semantics (temp HOME / XDG / SKILLDOZER_CONFIG) and the per-grep escape gotchas are captured below. An implementer who has never seen this repo can run the script as-is and interpret a green run.

### Documentation & References

```yaml
# MUST READ — the authoritative §13 script (copy it verbatim)
- file: PRD.md
  why: "§13 (h2.12, lines 343-415) is THE acceptance script. Every assertion, every env var,
        every grep is load-bearing. Do NOT paraphrase it; copy it block-for-block. The
        isolation block (mkdir /tmp/skilldozer-iso; env -u SKILLDOZER_SKILLS_DIR HOME=...
        XDG_CONFIG_HOME=...) MUST be run exactly so the config writes go to the temp tree,
        not the dev HOME."
  section: "h2.12 (§13). Also h3.9 (§8.2 init) and h3.10 (§8.3 priority) for interpreting failures."

# MUST READ — the verified ground truth (proves 15/15 + 4/4 already pass on current main)
- file: plan/002_38acb6d28a6a/P1M4T1S1/research/acceptance_run.md
  why: "Step-by-step PASS/FAIL + actual observed stdout/stderr/rc for all 15 §13 gates,
        run 2026-07-07 against a freshly rebuilt binary. Includes the per-step owning
        source file (skillsdir.go rule numbers, main.go line refs). This is the BASELINE —
        if a future regression breaks a gate, diff the failing step against this file."
  critical: "Steps 12-15 (the config block) are the NEW assertions this phase added and the
             most failure-prone. Their owners: unconfigured-hint = skillsdir.ErrNotFound
             message; init = main.go init dispatch + config.Save; config-wins = skillsdir
             findConfig rule #2; env-beats-config = Find() precedence env→config→sibling→walk-up."

- file: plan/002_38acb6d28a6a/P1M4T1S1/research/compliant_files.md
  why: "Per-file PASS evidence + the EXACT grep commands for .gitignore (5 patterns, L2/3/5/6/16),
        LICENSE (line 1 = 'MIT License'), go.mod (module L1 + single require L5), install.sh
        (ldflags L38-40, ln -sfn L68, 'skilldozer example' L98/L101). Copy these greps verbatim."
  critical: "The .gitignore has 5 EXTRA entries beyond §16 (/build, node_modules/, venv/, .env,
             .pi-subagents/). PRD §16 is illustrative ('everything else is committed'), NOT
             exact-match — do NOT assert the file has ONLY 5 lines. Assert the 5 REQUIRED
             patterns are PRESENT (substring/line match), which they are."

# MUST READ — the system context (what this phase changed + the dependency DAG)
- file: plan/002_38acb6d28a6a/architecture/system_context.md
  why: "§2 verified pre-state; §4 the decided architecture (internal/config → skillsdir → main.go);
        §7 the reaffirmed constraints (no catalog, store never in pi auto-discovery, yaml.v3 the
        only non-stdlib dep). Use §5 'suggested build order' step 8 = THIS task."
  section: "§2 (pre-state), §4 (DAG), §5 step 8 (this task), §7 (constraints)."

- file: plan/002_38acb6d28a6a/architecture/code_prd_delta.md
  why: "§9 'What is CORRECTLY aligned' bounds what should NOT change. §10 gap index G1-G22 maps
        each former gap to its owning subtask — use this to ROUTE any new failure (do not fix
        inline). E.g. ErrNotFound message drift = G4 (skillsdir); init flow = G6-G11 (main.go)."
  section: "§9 (correctly aligned), §10 (gap→subtask routing table)."

# READ-ONLY — the previous PRP (completions) that lands BEFORE this task in the parallel batch
- file: plan/002_38acb6d28a6a/P1M3T2S1/PRP.md
  why: "P1.M3.T2.S1 adds init + --store to the three completion files. It touches NO Go code and
        NO §13 gate. Its only §13-adjacent effect: `skilldozer <TAB>` now offers `init`. This
        acceptance task does NOT assert completion behavior (§14, not §13). If the completions
        PRP is mid-flight, the §13 transcript is unaffected — verify only that the three
        completion files still pass `bash -n` / `zsh -n` / `fish -n` (optional, not a §13 gate)."
```

### Current Codebase tree

```bash
$ cd /home/dustin/projects/skilldozer
$ ls
completions/          # bash/zsh/fish (P1.M3.T2.S1 in flight; NOT a §13 gate)
go.mod  go.sum        # module github.com/dabstractor/skilldozer; single yaml.v3 dep
install.sh            # §12.1 ldflags + symlink + 'skilldozer example'  (compliant)
internal/             # config/ skillsdir/ discover/ resolve/ ui/ search/ check/  (all w/ tests)
LICENSE               # MIT                                          (compliant)
main.go  main_test.go # ~47k / ~86k; init dispatch + run() orchestration
PRD.md  README.md     # PRD read-only; README = P1.M4.T2.S1 (out of scope here)
skills/example/SKILL.md  # §11-correct (skilldozer wording) as of P1.M3.T1.S1
.gitignore            # 5 §16 patterns present (+5 extras, allowed)     (compliant)
skilldozer            # built binary (gitignored; rebuilt by step 1)
```

### Desired Codebase tree with files to be added and responsibility of file

```bash
plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh        # NEW — the §13 runner (this task's script)
plan/002_38acb6d28a6a/acceptance_transcript.txt         # NEW — captured stdout/stderr/rc per step (the deliverable)
plan/002_38acb6d28a6a/P1M4T1S1/research/*.md            # already exist (acceptance_run.md, compliant_files.md)
```

| File | Responsibility |
|---|---|
| `plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh` | Executable bash. Runs §13 verbatim in isolation, captures every step's stdout/stderr/rc into `acceptance_transcript.txt`, then runs the four compliant-file greps. Exits 0 only if every `OK` marker printed. |
| `plan/002_38acb6d28a6a/acceptance_transcript.txt` | The green transcript. Written by the script. This is the Definition-of-Done artifact for the phase. |

**No source code is touched. No docs are patched.** The only writes are the two files above (and the research/ notes, already present).

### Known Gotchas of our codebase & Library Quirks

```bash
# CRITICAL (isolation): the §13 config block writes a REAL config.yaml. PRD §13 wraps it in
#   mkdir -p /tmp/skilldozer-iso && cp ./skilldozer /tmp/skilldozer-iso/skilldozer && cd /tmp/skilldozer-iso
#   and sets HOME=/tmp/skilldozer-iso/home + XDG_CONFIG_HOME=... + SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml.
#   PRESERVE THESE EXACT ENV OVERRIDES. If you drop HOME=/ XDG=/ SKILLDOZER_CONFIG=, the run will
#   write to the dev ~/.config/skilldozer/config.yaml and corrupt the dev environment.
#   The `cd - >/dev/null` at the end restores cwd. Keep it.

# CRITICAL (the `(found via <src>)` line is on STDERR, not stdout):
#   test "$(./skilldozer --path)" = "$PWD/skills"
#   works ONLY because $() captures stdout and the label goes to stderr. Verified: stdout is
#   byte-exactly the dir path + newline; stderr is `(found via sibling of binary)`. Do not
#   `2>&1` this particular assertion or it will break.

# CRITICAL (the unconfigured-hint grep has BACKTICKS inside single quotes):
#   grep -q 'run `skilldozer init`' err
#   The backticks are LITERAL characters in the error message (skillsdir.ErrNotFound =
#   "skilldozer is not configured; run `skilldozer init`"). Single-quoting the grep pattern
#   prevents the shell from trying to command-substitute. Copy verbatim — do NOT switch to
#   double quotes or the backticks will execute.

# GOTCHA (pi line is environment-dependent): `pi` must be on PATH (v0.80.3 confirmed at
#   /home/dustin/.local/bin/pi). The line is:
#     pi --no-skills --skill "$(./skilldozer example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
#   `--no-skills` proves we rely SOLELY on the explicit --skill path. If pi is absent, SKIP with
#   a recorded note (do not fail the whole transcript) — but on this machine pi IS present and
#   the line prints the example skill's name/description/location. Grep `example` from the output.

# GOTCHA (.gitignore is NOT exact-match): PRD §16 lists 5 ILLUSTRATIVE patterns; the file has
#   5 extra (/build, node_modules/, venv/, .env, .pi-subagents/). Assert PRESENCE of the 5
#   required patterns, not absence of others. The §13 script does not grep .gitignore at all —
#   the compliant-file greps are THIS task's addition (item OUTPUT #4 in the work-item contract).

# GOTCHA (the built `skilldozer` binary is gitignored): `go build -o skilldozer .` rewrites it
#   in the working tree. That is expected (/skilldozer is in .gitignore). Do NOT git-add it.

# GOTCHA (completions may be mid-edit by P1.M3.T2.S1): the three completion files are NOT a §13
#   gate. If they are dirty/uncommitted, that does not affect this transcript. Do not touch them.
```

## Implementation Blueprint

### Data models and structure

None — this is a bash verification script. No Go types, no schemas. The only "data" is the transcript text file.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CREATE plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh
  - IMPLEMENT: a bash script that runs PRD §13 verbatim in an isolated temp tree and captures
    stdout/stderr/rc per step into plan/002_38acb6d28a6a/acceptance_transcript.txt.
  - FOLLOW pattern: the §13 block in PRD.md lines 343-415 (h2.12). Copy it block-for-block.
  - STRUCTURE:
      (a) `set +u; set +e` (the script must continue through failures so the FULL transcript is
          captured; a failure is reported, not fatal-to-capture).
      (b) Redirect ALL output to the transcript via a top-level `exec > >(tee ...) 2>&1` OR by
          wrapping each step in a helper `step() { echo "== $1 =="; shift; "$@"; echo "rc=$?"; }`.
          The helper approach gives cleaner per-step rc capture — prefer it.
      (c) Step group A (repo-local, no isolation): build, --version, --path sibling test, --list,
          example dir test, -f file test, unknown-tag contract, absolute-path contract, check,
          pi end-to-end line, symlink, env override.
      (d) Step group B (isolation): mkdir /tmp/skilldozer-iso + cp binary + cd; unconfigured-hint
          (env -u SKILLDOZER_SKILLS_DIR HOME=... XDG_CONFIG_HOME=... ./skilldozer x 2>err);
          init --store; test -d store; grep config; config-rule-wins --path grep; env-beats-config
          --path grep; cd - >/dev/null.
      (e) Step group C (compliant-file greps): .gitignore 5 patterns; LICENSE line 1; go.mod module
          + single require; install.sh ldflags + ln -s + 'skilldozer example'.
      (f) Final summary: count of OK markers; exit 0 iff all present, else 1.
  - NAMING: run_acceptance.sh (snake_case, matches the plan/ convention).
  - PLACEMENT: plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh (alongside the PRP).
  - DEPENDENCIES: the built binary (step 1 builds it), pi on PATH (skip-with-note if absent).

Task 2: RUN the script and CAPTURE the transcript
  - EXECUTE: `bash plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh`
  - OUTPUT LANDS AT: plan/002_38acb6d28a6a/acceptance_transcript.txt (the deliverable).
  - VERIFY: every step printed its OK marker; the file is non-empty; the dev ~/.config/skilldozer
    was NOT touched (isolation held).
  - This is the Definition-of-Done artifact for the phase.

Task 3: REMEDIATE any failure (CONDITIONAL — only if a step does NOT print OK)
  - DECISION TREE (route, do not fix-inline):
      build fails                → Go compile error → owning subtask of the changed file.
      --version fails            → main.go version var / ldflags → P1.M2 (main.go).
      --path sibling fails       → skillsdir.findSibling / resolveSiblingFromExe → P1.M1.T2.
      unknown-tag non-empty      → main.go tag-branch buffered flush → P1.M2 / original resolve.
      unconfigured-hint grep     → skillsdir.ErrNotFound message → P1.M1.T2.S2 (G4).
      init --store fails         → main.go init dispatch + config.Save → P1.M2.T2.S2/S3.
      config-rule-wins fails     → skillsdir.findConfig rule #2 → P1.M1.T2.S2.
      env-beats-config fails     → skillsdir.Find() precedence → P1.M1.T2.
      .gitignore / LICENSE / go.mod / install.sh non-compliant
                                  → route per work-item DOCS note: README → P1.M4.T2.S1;
                                    install.sh/LICENSE/go.mod/.gitignore → the relevant Mode A
                                    subtask (these are already verified PASS; a failure here means
                                    a regression introduced since 2026-07-07).
  - CONSTRAINT: do NOT patch docs ad hoc. Open/append a routing note to the transcript and hand
    the fix to the named subtask. This task owns the GREEN TRANSCRIPT, not new code.
  - After the owning subtask fixes, RE-RUN Task 2 to refresh the transcript.

Task 4: FINAL confirmation
  - APPEND to the transcript (or a sibling compliant_files.txt) a one-line-per-check confirmation
    for the four compliant files: `gitignore OK`, `license OK`, `gomod OK`, `install OK`.
  - These four lines are item OUTPUT #4 in the work-item contract.
```

### Implementation Patterns & Key Details

```bash
# PATTERN: per-step capture helper (gives clean rc-per-step in the transcript)
step() {
  local name="$1"; shift
  echo ""
  echo "===== $name ====="
  "$@"
  local rc=$?
  echo "[rc=$rc] $name"
  return $rc   # do not exit; the caller tallies
}

# PATTERN: the isolation block MUST set all three env vars (HOME, XDG_CONFIG_HOME,
# SKILLDOZER_CONFIG) and unset SKILLDOZER_SKILLS_DIR. Copy PRD §13 verbatim:
#   env -u SKILLDOZER_SKILLS_DIR HOME=/tmp/skilldozer-iso/home \
#     XDG_CONFIG_HOME=/tmp/skilldozer-iso/home/.config ./skilldozer x 2>err; rc=$?

# PATTERN: the unconfigured-hint grep is single-quoted (backticks are literal):
#   grep -q 'run `skilldozer init`' err && echo "unconfigured-hint OK"

# PATTERN: the --path assertions capture STDOUT only (label is on stderr):
#   test "$(./skilldozer --path)" = "$PWD/skills"   # do NOT add 2>&1 here
#   .../skilldozer --path | grep -q /tmp/skilldozer-store   # stdout only is fine
#   .../skilldozer --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR  # HERE 2>&1 is correct (grep the label)

# PATTERN: compliant-file greps (presence, not exact-match):
grep -qE '^/skilldozer$' .gitignore && grep -qE '^/dist$' .gitignore \
  && grep -qE '^\*\.test$' .gitignore && grep -qE '^\*\.out$' .gitignore \
  && grep -qE '^\.DS_Store$' .gitignore && echo "gitignore OK"
[ "$(head -1 LICENSE)" = "MIT License" ] && echo "license OK"
grep -qE '^module github\.com/dabstractor/skilldozer$' go.mod \
  && [ "$(grep -c '^require ' go.mod)" = "1" ] \
  && grep -qE '^require gopkg\.in/yaml\.v3 v3\.0\.1$' go.mod && echo "gomod OK"
grep -qE 'go build' install.sh && grep -qE -- '-ldflags' install.sh \
  && grep -qE -- '-X main\.version' install.sh && grep -qE 'ln -s(fn|f)? ' install.sh \
  && grep -qE 'skilldozer example' install.sh && echo "install OK"
```

### Integration Points

```yaml
BUILD:
  - command: "go build -o skilldozer ."
  - artifact: "./skilldozer (gitignored; rebuilt each run)"

ISOLATION (the config block writes ONLY to the temp tree):
  - temp_home: "/tmp/skilldozer-iso/home"
  - temp_xdg:  "/tmp/skilldozer-iso/home/.config"
  - temp_cfg:  "/tmp/skilldozer-iso/cfg.yaml"
  - temp_store: "/tmp/skilldozer-store (or /tmp/skilldozer-iso/store)"
  - invariant: "the dev ~/.config/skilldozer/config.yaml is byte-identical before/after"

OUTPUT ARTIFACTS:
  - transcript: "plan/002_38acb6d28a6a/acceptance_transcript.txt"
  - script:     "plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh"

NO CHANGES TO:
  - main.go / internal/** (read-only this task)
  - PRD.md (read-only, always)
  - README.md (route to P1.M4.T2.S1)
  - tasks.json / prd_snapshot.md (orchestrator-owned)
```

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
# The runner is bash — lint it before relying on it.
bash -n plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh   # syntax check (must be clean)
shellcheck -x plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh 2>/dev/null || true
# Expected: `bash -n` clean. ShellCheck may flag SC2086 (word-splitting) on the env-injection
# lines — that is acceptable (PRD §13 uses unquoted env assignments intentionally).
```

### Level 2: Unit Tests (Component Validation)

Not applicable — there is no new Go code. The existing test suite is a pre-condition sanity check:

```bash
go test ./...                                   # all internal/* tests pass
go test ./internal/skillsdir/ -run 'TestFind' -v # the 5-rule precedence tests
go test ./internal/skillsdir/ -run 'TestErrNotFound' -v # the unconfigured-message test
# Expected: PASS. If any fails, the §13 run will also fail — route to the owning subtask.
```

### Level 3: Integration Testing (System Validation)

```bash
# THIS IS THE DELIVERABLE. Run the script:
bash plan/002_38acb6d28a6a/P1M4T1S1/run_acceptance.sh

# Inspect the transcript:
cat plan/002_38acb6d28a6a/acceptance_transcript.txt

# Confirm every required OK marker is present:
grep -cE 'OK$' plan/002_38acb6d28a6a/acceptance_transcript.txt
# Expected markers (count >= 11 from §13 + 4 from compliant files = 15):
#   build OK, (version prints), path_rc=0, (list shows example), example_dir_rc=0,
#   file_rc=0, unknown-tag contract OK, absolute OK, (check rc=0),
#   (pi line references example), symlink resolves, env override resolves,
#   unconfigured-hint OK, (store_OK + cfg_OK), (cfgwins via grep), (envbeats via grep),
#   gitignore OK, license OK, gomod OK, install OK.

# Verify isolation held (the dev config was NOT mutated):
test -f ~/.config/skilldozer/config.yaml && sha256sum ~/.config/skilldozer/config.yaml
# (record before & after; they must match. If the dev config did not pre-exist, it must
#  STILL not exist after the run — the isolation block wrote to /tmp only.)

# Expected: all §13 assertions print their OK marker; transcript non-empty; dev env untouched.
```

### Level 4: Creative & Domain-Specific Validation

```bash
# The pi end-to-end line is the "does it actually work in the target runtime" check.
# Confirm pi loaded ONLY via --skill (--no-skills disables auto-discovery):
pi --no-skills --skill "$(./skilldozer example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
# Expected: output names the example skill (e.g. "The example skill is loaded." + its
# name/description/location). Grep 'example' to confirm.

# (Optional, NOT a §13 gate) completions still parse after P1.M3.T2.S1:
bash -n completions/skilldozer.bash && echo "bash-comp OK"
zsh  -n completions/_skilldozer      && echo "zsh-comp OK"
fish -n completions/skilldozer.fish  && echo "fish-comp OK"
```

## Final Validation Checklist

### Technical Validation

- [ ] `bash -n run_acceptance.sh` clean.
- [ ] `go test ./...` passes (pre-condition; not modified by this task).
- [ ] `bash run_acceptance.sh` exits 0.
- [ ] `acceptance_transcript.txt` exists, non-empty, contains every step.

### Feature Validation

- [ ] All 16 §13 success criteria (listed above) print their OK marker.
- [ ] All 4 compliant-file greps print OK.
- [ ] pi end-to-end line references the example skill under `--no-skills`.
- [ ] Isolation held: dev `~/.config/skilldozer` byte-identical before/after.
- [ ] No doc patched ad hoc (any drift routed to P1.M4.T2.S1 or the owning Mode A subtask).

### Code Quality Validation

- [ ] The runner script is self-contained, re-runnable, and idempotent (re-running overwrites the transcript cleanly).
- [ ] Temp dirs are under `/tmp` (or cleaned up); no writes outside `/tmp` + the two plan/ artifact paths.
- [ ] The transcript is human-readable (step headers, rc per step, OK markers).

### Documentation & Deployment

- [ ] The transcript IS the documentation (it is the proof).
- [ ] Any non-compliance found is recorded as a one-line routing note in the transcript (not fixed inline).

---

## Anti-Patterns to Avoid

- ❌ Don't paraphrase the §13 script — copy it verbatim. The env-var names, the grep patterns, the `cd /tmp/skilldozer-iso`, the `cd - >/dev/null` are all load-bearing.
- ❌ Don't `2>&1` the `test "$(./skilldozer --path)" = "$PWD/skills"` assertion — the `(found via …)` label is on stderr and would break the equality.
- ❌ Don't double-quote the `grep -q 'run \`skilldozer init\`'` pattern — the backticks must stay literal (single quotes).
- ❌ Don't assert `.gitignore` has ONLY 5 lines — PRD §16 is illustrative; assert the 5 required patterns are PRESENT.
- ❌ Don't patch README/install.sh/source inline on a failure — route to the owning subtask (this task owns the transcript, not fixes).
- ❌ Don't skip the isolation env vars (`HOME=`, `XDG_CONFIG_HOME=`, `SKILLDOZER_CONFIG=`) — without them the run corrupts the dev config.
- ❌ Don't `git add` the built `skilldozer` binary — it is gitignored (`/skilldozer`).
- ❌ Don't treat the completions (P1.M3.T2.S1, in-flight) as a §13 gate — they are §14, out of scope here.

---

## Confidence Score

**9/10.** As of 2026-07-07 the entire §13 suite (15/15 gates) and all four compliant files (4/4) PASS against the current `main` binary, verified by direct execution (see `research/acceptance_run.md`, `research/compliant_files.md`). The pi end-to-end line was independently exercised and prints the example skill. The remaining 1/10 is environmental: (a) a future regression in a re-planned Mode A subtask could break a gate between now and run time (the remediation decision tree in Task 3 routes any such break); (b) if `pi` is not on PATH at run time the pi line must be skipped-with-note (it is present on this machine). The script itself is straightforward bash with a well-defined transcript contract.
