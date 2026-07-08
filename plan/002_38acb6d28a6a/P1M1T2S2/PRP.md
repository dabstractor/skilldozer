# PRP — P1.M1.T2.S2: Implement `findConfig()` rule + wire it into `Find()` at priority #2; set the exact `ErrNotFound` message

> **Subtask:** The second of two siblings that wire PRD §8.1's config-file mechanism into skills-dir discovery. T2.S1 (lands first) adds the `SourceConfig` enum/label + exports `HasSkillMD` + refreshes doc comments. **T2.S2 adds the actual `findConfig` helper, inserts it into `Find()` at priority #2 (between env and sibling), flips the `ErrNotFound` message to the PRD §6.4/§8.2 exact string, and ships the config-rule + precedence tests.**
> **Scope:** Three existing files only — `internal/skillsdir/skillsdir.go`, `internal/skillsdir/skillsdir_test.go`, `main_test.go`. No new files. One new import (`internal/config`, already landed in P1.M1.T1). No `main.go` change, no `internal/config` change, no docs.
>
> **STATUS (verified at PRP-write time):** the parallel sibling **P1.M1.T2.S1 has LANDED** — `SourceConfig` enum + the `"config file"` label + exported `HasSkillMD` + the 5-rule doc comments are all already present in `internal/skillsdir/skillsdir.go` (`SourceConfig` at `:35-36`, `case SourceConfig` at `:49-50`, `func HasSkillMD` at `:166`, `Find()` doc comment at `:237-244`, `ErrNotFound` doc at `:231-233`). So the live line numbers in this PRP are POST-T2.S1: `var ErrNotFound` is at `skillsdir.go:234`, `Find()` at `:247-258`, `findEnv` at `:74`, `findSibling` at `:101`. GOTCHA #8's doc-comment verify-gate (`grep 'three §8'` → expect 0) is already satisfied by T2.S1. The implementer can proceed immediately without waiting for T2.S1.

---

## Goal

**Feature Goal**: Complete the §8.3 5-rule discovery ladder by (a) implementing `findConfig() (dir string, src Source, found bool)` that reads the config file's `store` key via `config.Path()`+`config.Load()`, absolutizes it (relative values resolved against the config file's own dir per PRD §8.1), and returns `(absStore, SourceConfig, true)` only when it names an existing directory — falling through silently on **any** miss (no file / no `store` key / non-existent dir / unreadable / malformed YAML, never a hard error); (b) wiring it into `Find()` between `findEnv` and `findSibling` so the order is `env → config → sibling → walk-up`; (c) flipping `ErrNotFound`'s message to exactly `skilldozer is not configured; run \`skilldozer init\`` so the §13 acceptance gate (`grep -q 'run \`skilldozer init\`'`) passes.

**Deliverable**: Edits to three existing files:
1. `internal/skillsdir/skillsdir.go` — add `import ".../internal/config"`; add `findConfig()`; insert one `findConfig()` call into `Find()` between the env and sibling calls; replace the `ErrNotFound` message string with the exact new value.
2. `internal/skillsdir/skillsdir_test.go` — add 7 tests (findConfig hit/miss×4/relative-store/env-beats-config); flip `TestErrNotFoundMessageHasFix` substrings; harden `unsetEnvVar` to neutralize the config rule (hermeticity).
3. `main_test.go` — flip the **6** tests that assert the OLD ErrNotFound substrings (`TestRunPathFailureErrNotFound`, `TestRunListSkillsDirUnresolvableExit1`, `TestRunTagSkillsDirUnresolvable`, `TestRunAllSkillsDirUnresolvable`, `TestRunSearchSkillsDirUnresolvable`, `TestRunCheckSkillsDirUnresolvable`); harden `unsetSkillsEnv` to neutralize the config rule (hermeticity). *(Research-discovered scope: the contract scoped tests to skillsdir_test.go only, but main_test.go asserts the old message in **6** tests and they break on the flip — see "Why" + Gotcha #4. Mandatory for `go test ./...` to pass. A fresh-context scout re-verification confirmed the full set of 6, not just the 3 the contract's author likely had in mind.)*

**Success Definition**: `go build/vet/test ./...` all pass (including the updated main_test.go); `gofmt -l internal/skillsdir/ main.go` empty; `go.mod`/`go.sum` byte-for-byte unchanged (config is internal — no new external dep); `skilldozer --path` with a config set prints `(found via config file)` to stderr (via the `SourceConfig` label T2.S1 added — zero main.go change); the unconfigured binary writes exactly `skilldozer is not configured; run \`skilldozer init\`` to stderr and exits 1 (§13 grep passes); `SKILLDOZER_SKILLS_DIR` still beats a valid config (precedence).

---

## User Persona (if applicable)

**Not applicable** — internal package logic + one message constant. The user-facing surface (the `--path` stderr label and the unconfigured stderr line) is rendered by `main.go` generically from `Source.String()` / `err.Error()` (verified at `main.go:414` `fmt.Fprintln(stderr, err)` and `main.go:423` `fmt.Fprintf(stderr, "(found via %s)\n", src)`). No `main.go` edit, no doc edit rides here (README sync is P1.M4.T2.S1).

---

## Why

- **Closes gap G1** (`code_prd_delta.md` §1): `Find()` has 3 rules; PRD §8.3 needs 5. The `findConfig` call at priority #2 is the single missing rule. Until it lands, the config file `skilldozer init` writes (P1.M2) is never consulted — every configured user still falls through to sibling/walk-up.
- **Closes gap G4** (`code_prd_delta.md` §5): `ErrNotFound`'s message is the OLD hint. PRD §6.4/§8.2 + the §13 acceptance gate require EXACTLY `skilldozer is not configured; run \`skilldozer init\``. The §13 line `grep -q 'run \`skilldozer init\`' err` FAILS on the current binary.
- **Closes gap G5** (config-file reader): the `findConfig() (dir, src, found)` that reads `config.Path()`, unmarshals `{store: <path>}`, absolutizes relative store paths against the config file's dir (PRD §8.1), `os.Stat`s the store, and returns `(absDir, SourceConfig, true)` — or `found=false` on any miss — is the core of §8.1.
- **Closes gap G16** (tests): no `findConfig`/SourceConfig/env-beats-config precedence tests exist. The 7 new tests lock the contract.
- **Unblocks every consumer of `skillsdir.Find()`**: path/list/search/check/all/tag-resolution all flow through `Find()` (`main.go:408,431,467,507,548,579`). The config rule + new message make a configured `skilldozer` actually use its config and an unconfigured one fail loudly with the right hint — both load-bearing for the §13 acceptance suite (P1.M4.T1.S1) and for `init` (P1.M2, which writes the config this rule then reads).

---

## What

### Success Criteria

- [ ] `findConfig()` exists in `internal/skillsdir/skillsdir.go` with signature `func findConfig() (dir string, src Source, found bool)`, composed from `config.Path()` + `config.Load()`, returning `(absStore, SourceConfig, true)` on hit and `("", 0, false)` on every miss (never a hard error).
- [ ] `Find()` consults rules in order **`findEnv → findConfig → findSibling → findWalkUp`**, returning the first hit; on total miss returns `("", 0, ErrNotFound)`.
- [ ] `ErrNotFound`'s message is EXACTLY `skilldozer is not configured; run `skilldozer init`` (literal backticks — see Gotcha #1), set via `errors.New("skilldozer is not configured; run `skilldozer init`")`.
- [ ] Relative `store` values are resolved against `filepath.Dir(configPath)` (PRD §8.1); absolute values are used (cleaned) as-is.
- [ ] `go test ./...` green, including: 7 new skillsdir tests; updated `TestErrNotFoundMessageHasFix`; **6** updated main_test.go tests (path/list/tag/all/search/check unresolvable); hardened `unsetEnvVar`/`unsetSkillsEnv`.
- [ ] `go.mod`/`go.sum` unchanged; `main.go` unchanged; `internal/config` unchanged; no new files.

---

## All Needed Context

### Context Completeness Check

**Pass.** Every edit is pinned to a symbol I located in the live file (`internal/skillsdir/skillsdir.go` read in full; `skillsdir_test.go` read in full; the 3 affected `main_test.go` tests read at exact line ranges). The two non-obvious failure modes — (a) the Go backtick-escape **compile error** and (b) the `main_test.go` breakage + hermeticity risk — are empirically verified (`research/verified_facts.md` §3, §5). The `config` API consumed (`Path`, `Load`, `File.Store`) is read from the landed `internal/config/config.go`. The T2.S1 boundary (what it delivers vs. what's exclusive to T2.S2) is crisply drawn (§6). An implementer who has never seen this repo can complete it in one pass.

### Documentation & References

```yaml
# MUST READ — the authoritative gap analysis (G1/G4/G5/G16 are THIS subtask)
- file: plan/002_38acb6d28a6a/architecture/code_prd_delta.md
  why: "§1 (G1) quotes the exact Find() body (env/sibling/walkup) to edit and pins the
        Source/String site; §5 (G4) gives the EXACT old vs new ErrNotFound message and notes
        the §13 grep depends on it; §2 (G5) specifies findConfig's full contract (absolutize
        relative store against the config file's dir; os.Stat; fall through on any miss,
        never hard error); §6 (G16) lists the missing tests. §1 line 191 confirms main.go
        needs NO change (src.String() already flows to --path stderr)."
  section: "§1 (G1, Find body + Source/String), §2 (G5, findConfig spec), §5 (G4, ErrNotFound
            message), §6 (G16, tests), §10 gap index."

# MUST READ — the verified facts (line refs, gotchas empirically proven)
- file: plan/002_38acb6d28a6a/P1M1T2S2/research/verified_facts.md
  why: "§3 enumerates the 6 main_test.go tests that break (scout-re-verified; the contract MISSED these —
        scope expansion is mandatory); §4 the hermeticity risk (config rule makes existing
        Find() tests machine-dependent); §5 the backtick-escape compile-error proof (the #1
        one-pass stall); §6 the T2.S1 boundary (ErrNotFound doc comment is a VERIFY gate, not
        a re-edit); §7 the exact findConfig contract; §8 the 7 new tests."
  critical: "§3 (main_test.go breakage) and §5 (backtick escape) are the two things most
             likely to be got wrong without this file."

# MUST READ — the config API consumed (fully landed in P1.M1.T1)
- file: internal/config/config.go
  why: "THE import target. config.Path() (string,error) — SKILLDOZER_CONFIG override (literal,
        cleaned) else $XDG_CONFIG_HOME/skilldozer/config.yaml; returns os.UserConfigDir errors
        VERBATIM (not wrapped). config.Load(path) (File,error) — os.ReadFile error returned
        VERBATIM so errors.Is(err, fs.ErrNotExist) works; unknown keys ignored (lenient),
        broken YAML is a hard error. config.File{Store string `yaml:\"store,omitempty\"}`.
        Doc comments explicitly name 'findConfig (P1.M1.T2.S2) relies on this to fall through'."
  gotcha: "Load returns the ReadFile error UNWRAPPED — findConfig must treat ANY non-nil error
           as 'fall through' (it does not need errors.Is/fs.ErrNotExist). Unknown keys are
           ignored; SYNTACTICALLY BROKEN YAML is a hard error that findConfig must also turn
           into a fall-through (return false), NOT propagate."

# MUST READ — the file under edit (read in full; locate symbols by NAME, line numbers shift after T2.S1)
- file: internal/skillsdir/skillsdir.go
  why: "THE edit target (POST-T2.S1 line numbers — T2.S1 has landed). Find() body (:247-258) —
        insert findConfig between the findEnv (:248) and findSibling (:250) calls. var ErrNotFound
        (:234) — flip the message string. Import block (top, :12-17) — add the internal/config
        import. findEnv (:74-91) is the PATTERN to mirror for findConfig's shape (returns
        (dir, src, found); bad input -> false, never errors; absolutize via filepath.Abs/Clean)."
        to mirror for findConfig's shape (returns (dir, src, found); bad input -> false, never
        errors; absolutize via filepath.Abs/Clean)."
  pattern: "Per-rule helper shape: `func findXxx() (dir string, src Source, found bool)`; on
            miss return `(\"\", 0, false)`; on hit return `(absDir, SourceXxx, true)`. Never
            returns an error (the per-rule shape is locked — only Find returns err)."

# MUST READ — the test file under edit
- file: internal/skillsdir/skillsdir_test.go
  why: "THE other edit target. TestErrNotFoundMessageHasFix (~508) — flip substrings.
        unsetEnvVar (~13) — harden to neutralize SKILLDOZER_CONFIG. TestFindRuleEnvWins /
        TestFindRuleWalkUpWins / TestFindAllMissReturnsErrNotFound — become hermetic via the
        hardened helper. makeSkill helper (~280) + the writeConfig idiom (internal/config/
        config_test.go) are the fixture patterns for the new findConfig tests."
  gotcha: "findConfig tests drive config.Path() via t.Setenv(\"SKILLDOZER_CONFIG\", cfgPath)
           pointing at a fixture under t.TempDir() — NOT via injection. The env-beats-config
           test sets BOTH SKILLDOZER_SKILLS_DIR and SKILLDOZER_CONFIG."

# MUST READ — the THIRD edit target (research-discovered; the contract omitted it)
- file: main_test.go
  why: "SIX tests assert the OLD ErrNotFound message and break on the flip (one per main mode
        that can hit the unconfigured path). Each asserts a substring the new message LACKS:
          TestRunPathFailureErrNotFound       (:228) asserts {SKILLDOZER_SKILLS_DIR,cd,reinstall} (:240)
          TestRunListSkillsDirUnresolvableExit1 (:368) asserts Contains('SKILLDOZER_SKILLS_DIR') (:379)
          TestRunTagSkillsDirUnresolvable     (:582) asserts Contains('SKILLDOZER_SKILLS_DIR') (:593)
          TestRunAllSkillsDirUnresolvable     (:840) asserts Contains('SKILLDOZER_SKILLS_DIR') (:851)
          TestRunSearchSkillsDirUnresolvable  (:1080) asserts Contains('SKILLDOZER_SKILLS_DIR') (:1091)
          TestRunCheckSkillsDirUnresolvable   (:1258) asserts Contains('SKILLDOZER_SKILLS_DIR') (:1269)
        All six call unsetSkillsEnv (:22-27, env-ONLY) + t.Chdir(t.TempDir()) — non-hermetic once
        findConfig is wired (a real ~/.config/skilldozer/config.yaml would make findConfig win).
        unsetSkillsEnv (:22-27) — harden to neutralize SKILLDOZER_CONFIG (Task 8a), which covers
        all six at once since they all call it."
  critical: "Skipping this file = `go test ./...` FAILS on the message flip (6 failures). The
             contract's 'skillsdir_test.go only' scoping did not account for main_test.go's
             duplicated substring assertions. No other task owns fixing them. A fresh-context
             scout re-verification found the COMPLETE set of 6 (not just path/list/tag)."

# MUST READ — the parallel sibling PRP (defines what T2.S2 consumes from T2.S1)
- file: plan/002_38acb6d28a6a/P1M1T2S1/PRP.md
  why: "Confirms T2.S1 lands SourceConfig (iota pos 1, value 1) + 'config file' label +
        HasSkillMD export + refreshed 5-rule doc comments, and explicitly does NOT touch
        findConfig / Find() body / ErrNotFound message / TestErrNotFoundMessageHasFix. Fixes
        the boundary so T2.S2 does not duplicate T2.S1's doc-comment refresh."

# READ-ONLY — PRD (the source of truth for the exact message + 5-rule ladder)
- file: PRD.md
  why: "§8.3 enumerates the 5-rule priority + the four --path labels verbatim. §8.2/§6.4 give
        the EXACT unconfigured message `skilldozer is not configured; run \\`skilldozer init\\``
        and the 'bare tag never prompts / writes nothing to stdout' contract. §13 acceptance
        greps `run \\`skilldozer init\\`` (unconfigured) and /tmp/skilldozer-store (config wins)
        and SKILLDOZER_SKILLS_DIR (env beats config)."
  section: "h2.7 / h3.8 / h3.9 / h3.10 (selected) + h2.12 (§13 acceptance). The §8.2/§6.4
            message and the §8.3 label list are the two load-bearing sentences."

# READ-ONLY — the contract (the orchestrator owns it)
- file: plan/002_38acb6d28a6a/tasks.json
  why: "P1.M1.T2.S2's CONTRACT block (INPUT/LOGIC/OUTPUT/DOCS) is authoritative. This PRP
        transcribes it; if anything contradicts tasks.json, tasks.json wins — EXCEPT the
        research-discovered main_test.go breakage (OUTPUT §4 says skillsdir_test.go only),
        which is a mandatory consequence of the message flip, not a contradiction of intent."
```

### Current Codebase tree

```bash
$ cd /home/dustin/projects/skilldozer && tree internal/ main.go -L 2
internal/
├── check/      check.go check_test.go
├── config/     config.go config_test.go   # LANDED (T1.S1+S2): {File,Load,Save,Path,DefaultStore} — findConfig consumes Path+Load+File.Store
├── discover/   discover.go index.go skill.go + _test.go   # untouched
├── resolve/    resolve.go resolve_test.go  # only MENTIONS Source/ErrNotFound in comments
├── search/     search.go search_test.go    # untouched
├── skillsdir/  skillsdir.go skillsdir_test.go   # <-- EDIT (both)
└── ui/         ui.go ui_test.go            # untouched
main.go         # untouched — Find()/ErrNotFound consumed generically at :408-423,431,467,507,548,579
main_test.go    # <-- EDIT (6 tests + unsetSkillsEnv helper)
$ grep -rn "findConfig\|SourceConfig" --include="*.go" . | grep -v skillsdir
# (empty outside skillsdir — T2.S2 is fully self-contained at the call site; main.go consumes via Source.String()/err.Error())
```

### Desired Codebase tree with files to be added and responsibility of file

```bash
internal/
└── skillsdir/
    ├── skillsdir.go        # EDIT: +import internal/config, +findConfig(), +1 call in Find(), ErrNotFound message flip
    └── skillsdir_test.go   # EDIT: +7 tests, TestErrNotFoundMessageHasFix substrings, hardened unsetEnvVar
main_test.go                # EDIT: 6 substring-assertion flips + hardened unsetSkillsEnv
```

**No new files.** All edits are to existing files. Matches the package's one-source-file convention.

| File | T2.S2 responsibility |
|---|---|
| `internal/skillsdir/skillsdir.go` | Add `internal/config` import; add `findConfig()`; insert `findConfig()` into `Find()` at priority #2; flip `ErrNotFound` message to exact PRD §6.4 string |
| `internal/skillsdir/skillsdir_test.go` | Add 7 findConfig/precedence tests; flip `TestErrNotFoundMessageHasFix` substrings; harden `unsetEnvVar` to neutralize the config rule |
| `main_test.go` | Flip the 6 tests asserting the OLD message (path/list/tag/all/search/check); harden `unsetSkillsEnv` to neutralize the config rule |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 (CRITICAL — #1 one-pass stall) — Go double-quoted strings have NO `\`` escape.
// The new message is `skilldozer is not configured; run `skilldozer init`` (PRD §6.4/§8.2).
// The contract/prose writes it as `run \`skilldozer init\`` — that backslash is MARKDOWN/SHELL
// rendering convention, NOT Go syntax. Writing:
//     errors.New("skilldozer is not configured; run \`skilldozer init\`")
// is a COMPILE ERROR ("unknown escape"). Verified on go1.25: `\`` is not a recognized escape
// in a double-quoted string literal. The CORRECT literal uses UNESCAPED backticks (a backtick
// is an ordinary character in a double-quoted string):
//     var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer init`")
// Empirically verified: this compiles, prints exactly `run `skilldozer init``, and matches the
// §13 grep `grep -q 'run `skilldozer init`'` (GREP OK). research/verified_facts.md §5.
// Do NOT copy the contract's `\``-laden text into the source. Do NOT use a raw string literal
// (backtick-delimited) — it cannot contain backticks without concatenation; the double-quoted
// form with bare backticks is simplest and correct.

// GOTCHA #2 — findConfig must NEVER return a hard error. PRD §8.1: "A missing or unreadable
// config is treated as 'not yet configured' and falls through to §8.3 rules 3-5 — never a hard
// error." config.Load returns os.ReadFile errors VERBATIM (unwrapped) and a hard error for
// syntactically broken YAML. findConfig treats EVERY non-nil Load error (fs.ErrNotExist,
// permission denied, malformed YAML, …) as "fall through" via `if err != nil { return "",0,false }`.
// It does NOT need errors.Is/fs.ErrNotExist and does NOT import io/fs for this (the package
// already imports io/fs for WalkDir's fs.DirEntry — fine, but findConfig itself just checks
// err != nil). Same for config.Path(): any error -> fall through.

// GOTCHA #3 — Relative `store` is resolved against the CONFIG FILE's dir, not cwd. PRD §8.1:
// "store may be relative to the config file". So `store: ../skills` in a config at
// /home/u/.config/skilldozer/config.yaml resolves to /home/u/.config/skills. Implement:
//   if filepath.IsAbs(f.Store) { store = filepath.Clean(f.Store) }
//   else { store = filepath.Join(filepath.Dir(p), f.Store) }   // Join cleans + joins
// (p is the config file path from config.Path()). Do NOT use filepath.Abs(f.Store) — that
// resolves against cwd, which is wrong for a config-file-relative store.

// GOTCHA #4 (CRITICAL — research-discovered, scout-re-verified) — main_test.go ALSO breaks;
// the contract scoped tests to skillsdir_test.go only. SIX main_test.go tests (one per main mode
// that can hit the unconfigured path) assert the OLD ErrNotFound message and rely on "all rules
// miss" with ENV-ONLY neutralization:
//   TestRunPathFailureErrNotFound         (:228) asserts {"SKILLDOZER_SKILLS_DIR","cd","reinstall"} (:240)
//   TestRunListSkillsDirUnresolvableExit1 (:368) asserts Contains("SKILLDOZER_SKILLS_DIR") (:379)
//   TestRunTagSkillsDirUnresolvable       (:582) asserts Contains("SKILLDOZER_SKILLS_DIR") (:593)
//   TestRunAllSkillsDirUnresolvable       (:840) asserts Contains("SKILLDOZER_SKILLS_DIR") (:851)
//   TestRunSearchSkillsDirUnresolvable    (:1080) asserts Contains("SKILLDOZER_SKILLS_DIR") (:1091)
//   TestRunCheckSkillsDirUnresolvable     (:1258) asserts Contains("SKILLDOZER_SKILLS_DIR") (:1269)
// The message flip removes "SKILLDOZER_SKILLS_DIR"/"cd"/"reinstall" entirely → all 6 FAIL on the
// flip. They MUST be updated to the new message (Contains "skilldozer init" / "run"). Skipping
// this = `go test ./...` fails with 6 errors. No other task owns it. (verified_facts.md §3.)
// NOTE: a `--path`-success test (TestRunPathReportsSourceLabel etc., main_test.go:182/198/219)
// asserts the literal "(found via SKILLDOZER_SKILLS_DIR)\n" — that is the SourceEnv SOURCE LABEL
// (src.String()), NOT the ErrNotFound message; it is UNAFFECTED by the flip and must NOT be touched.

// GOTCHA #5 (hermeticity) — wiring findConfig makes the existing "all-miss" Find() tests
// machine-dependent. config.Path() reads $SKILLDOZER_CONFIG else os.UserConfigDir() →
// ~/.config/skilldozer/config.yaml. On THIS machine there's no config (tests pass today), but on
// any machine that ran `skilldozer init` or set the env, findConfig HITS and the all-miss tests
// return a real dir instead of ErrNotFound → test fails. Fix: harden unsetEnvVar (skillsdir_test.go)
// and unsetSkillsEnv (main_test.go) to ALSO neutralize the config rule by setting SKILLDOZER_CONFIG
// to a non-existent temp path (config.Path returns it; config.Load -> fs.ErrNotExist -> fall through).
// Safe because neutralizing config is harmless when env/sibling/walk-up hits first. Affected tests:
//   skillsdir_test.go: TestFindRuleWalkUpWins, TestFindAllMissReturnsErrNotFound (TestFindRuleEnvWins
//                      is technically safe — env hits first — but harden via the helper for consistency).
//   main_test.go: the 6 in GOTCHA #4. (research/verified_facts.md §4.)

// GOTCHA #6 — main.go needs NO change. All six Find() error sites print err via fmt.Fprintln(stderr, err)
// (err.Error() verbatim, no prefix): main.go:413 (--path), 433 (--list), 469 (--search), 509 (check),
// 550 (--all), 581 (tag). --path also does fmt.Fprintf(stderr, "(found via %s)\n", src) (:423),
// invoking Source.String() via the fmt Stringer — once findConfig returns SourceConfig, the
// "config file" label T2.S1 added renders automatically. Verified by reading main.go:408-428 and
// code_prd_delta.md §1. Do NOT touch main.go.

// GOTCHA #7 — findConfig's return shape is the LOCKED per-rule shape: (dir string, src Source,
// found bool). On miss return ("", 0, false); on hit return (absStore, SourceConfig, true). It
// never returns an error (only Find returns err). Mirror findEnv/findSibling/findWalkUp exactly.
// found==false with src's zero value is intentional — the caller (Find) ignores src on a miss.

// GOTCHA #8 — Do NOT re-edit the ErrNotFound DOC COMMENT. T2.S1 (lands first) already changed
// "all three §8 rules" → "every §8.3 rule misses (unconfigured)" and rewrote the package/Find
// doc comments to the 5-rule ladder. The T2.S2 contract also mentions the doc comment (LOGIC
// step 3), but since T2.S1 lands first, T2.S2's doc-comment item is a VERIFY gate, not a re-edit:
//   grep -c "three §8\|three rules" internal/skillsdir/skillsdir.go   # expect 0
// If (and only if) it still says "three" (T2.S1 did not land), fix it. Otherwise leave it.
// T2.S2's ACTUAL ErrNotFound edit is the MESSAGE STRING flip only. Editing the doc comment when
// T2.S1 already did it is wasted churn and risks a merge collision.

// GOTCHA #9 — Insert the findConfig CALL into Find() BETWEEN the findEnv and findSibling calls
// (priority #2), and place the findConfig DEFINITION between findEnv and findSibling in the file
// so the source reads top-to-bottom in priority order (matching the const block, String() switch,
// and TestSourceString table that T2.S1 already ordered). findSibling/walk-up BEHAVIOR is
// unchanged — only their CALL POSITION moves (sibling was #2, now #3; walk-up was #3, now #4).
// Do NOT alter findEnv/findSibling/findWalkUp/resolveSiblingFromExe/findWalkUpAncestor/HasSkillMD bodies.

// GOTCHA #10 — No new external dependency. The single new import is internal/config
// (github.com/dabstractor/skilldozer/internal/config), already in the module (landed P1.M1.T1).
// No go get, no go mod tidy. go.mod/go.sum must be byte-for-byte unchanged (config is internal,
// and it only uses yaml.v3 which is already pinned). Verify with `git diff --quiet go.mod go.sum`.

// GOTCHA #11 — The new findConfig tests must NOT call the (env-neutralizing) unsetEnvVar if they
// rely on setting SKILLDOZER_CONFIG to a real fixture — t.Setenv for the same key can be called
// twice (last wins, cleanup stack), but cleaner: findConfig UNIT tests set only SKILLDOZER_CONFIG
// (findConfig does not read SKILLDOZER_SKILLS_DIR). The env-beats-config test sets BOTH env vars
// to real dirs. The Find()-combiner tests (walk-up-wins, all-miss) use the hardened unsetEnvVar
// which neutralizes BOTH env vars.
```

---

## Implementation Blueprint

### Data models and structure

**No new data models.** `findConfig` composes two existing functions (`config.Path()`, `config.Load()`) and reads `config.File.Store` (a `string`). It returns the locked per-rule triple `(dir string, src Source, found bool)`, reusing the `Source` enum and the `SourceConfig` constant T2.S1 added. No structs, interfaces, or types.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: EDIT internal/skillsdir/skillsdir.go — add the internal/config import
  - FILE: internal/skillsdir/skillsdir.go (import block, top of file)
  - ADD one import line to the existing parenthesized import block (keep gofmt-sorted order):
      "github.com/dabstractor/skilldozer/internal/config"
    The block becomes (errors, io/fs, os, path/filepath, + the internal/config line in its
    sorted position — gofmt will place it; run gofmt after).
  - GOTCHA #10: this is the ONLY new import; config is internal, no external dep added.

Task 2: EDIT internal/skillsdir/skillsdir.go — add findConfig() between findEnv and findSibling
  - FILE: internal/skillsdir/skillsdir.go
  - PLACE the new function AFTER findEnv and BEFORE findSibling (source reads in priority order).
  - IMPLEMENT (exact contract — item_description LOGIC step 3; GOTCHA #2, #3, #7):
      // findConfig implements PRD §8.3 rule 2 — the config file's `store` key (PRD §8.1).
      //
      // It is the primary discovery rule, set by `skilldozer init`. config.Path() gives the
      // one fixed, well-known bootstrap path ($SKILLDOZER_CONFIG or $XDG_CONFIG_HOME/skilldozer/
      // config.yaml); config.Load() reads+unmarshals it (unknown keys ignored; broken YAML is a
      // hard error). findConfig treats ANY error from either as "not yet configured -> fall
      // through" — PRD §8.1: a missing/unreadable config NEVER hard-errors.
      //
      // A relative `store` is resolved against the config file's own directory (PRD §8.1:
      // store may be relative to the config file), NOT against cwd. The resolved store must
      // name an existing directory or the rule misses.
      //
      // Returns (absStore, SourceConfig, true) on a hit; ("", 0, false) otherwise so Find()
      // can fall through to the sibling rule. Never errors (locked per-rule shape).
      func findConfig() (dir string, src Source, found bool) {
          p, err := config.Path()
          if err != nil {
              return "", 0, false // no bootstrap path (e.g. relative $XDG_CONFIG_HOME) -> fall through
          }
          f, err := config.Load(p)
          if err != nil {
              return "", 0, false // missing/unreadable/malformed -> "not yet configured" -> fall through
          }
          if f.Store == "" {
              return "", 0, false // no `store` key -> fall through
          }
          var store string
          if filepath.IsAbs(f.Store) {
              store = filepath.Clean(f.Store)
          } else {
              store = filepath.Join(filepath.Dir(p), f.Store) // relative to config file's dir (PRD §8.1)
          }
          info, err := os.Stat(store)
          if err != nil || !info.IsDir() {
              return "", 0, false // store path is not an existing dir -> fall through
          }
          return store, SourceConfig, true
      }
  - GOTCHA #2: ANY Load error -> fall through (do NOT errors.Is / import io/fs for this).
  - GOTCHA #3: relative store joined to filepath.Dir(p), NOT filepath.Abs (cwd-relative).
  - GOTCHA #7: return shape (dir string, src Source, found bool); never returns error.

Task 3: EDIT internal/skillsdir/skillsdir.go — wire findConfig into Find() at priority #2
  - FILE: internal/skillsdir/skillsdir.go (Find() body)
  - INSERT one block between the findEnv and findSibling calls:
        if d, s, ok := findConfig(); ok {
            return d, s, nil
        }
  - Find() body becomes (the ONLY structural change; sibling/walk-up bodies untouched):
        func Find() (dir string, src Source, err error) {
            if d, s, ok := findEnv(); ok {
                return d, s, nil
            }
            if d, s, ok := findConfig(); ok {   // <-- NEW, priority #2 (PRD §8.3)
                return d, s, nil
            }
            if d, s, ok := findSibling(); ok {
                return d, s, nil
            }
            if d, s, ok := findWalkUp(); ok {
                return d, s, nil
            }
            return "", 0, ErrNotFound
        }
  - GOTCHA #9: place the CALL between env and sibling; sibling now runs at #3, walk-up at #4.
  - NOTE: Find()'s doc comment (refreshed by T2.S1 to the 5-rule ladder by Source label) is now
    ACCURATE — no comment change needed (T2.S1 already describes rule 2 as "Config file store
    (SourceConfig)"). If T2.S1's comment still says "3 rules", that's a T2.S1 bug — fix only the
    comment in passing, but do not touch findConfig/findEnv bodies.

Task 4: EDIT internal/skillsdir/skillsdir.go — flip the ErrNotFound message string
  - FILE: internal/skillsdir/skillsdir.go (var ErrNotFound = errors.New("…"))
  - REPLACE the message with EXACTLY (GOTCHA #1 — UNESCAPED backticks in a double-quoted string):
        var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer init`")
  - GOTCHA #1 (CRITICAL): do NOT write `\`` — that is a compile error ("unknown escape"). Use
    bare backticks inside the double-quoted literal. Verified to compile + match the §13 grep.
  - GOTCHA #8: do NOT re-edit the ErrNotFound DOC COMMENT (T2.S1 already refreshed it). Treat as
    a verify gate (Task 9): grep must show no "three §8" / "three rules".

Task 5: EDIT internal/skillsdir/skillsdir_test.go — harden unsetEnvVar (hermeticity)
  - FILE: internal/skillsdir/skillsdir_test.go (unsetEnvVar, ~line 13)
  - EXTEND unsetEnvVar to ALSO neutralize the config rule, so every test using it is hermetic
    against findConfig once it is wired into Find(). Add (after the existing env unset/cleanup):
        // Also neutralize the config-file rule (PRD §8.3 rule 2): point SKILLDOZER_CONFIG at a
        // non-existent path so findConfig deterministically misses. Required once findConfig is
        // wired into Find() — otherwise a machine with a real config (~/.config/skilldozer/
        // config.yaml, or SKILLDOZER_CONFIG set) would make the all-miss tests return a real dir.
        // Harmless when a higher-priority rule (env/sibling/walk-up) hits first.
        cfgGhost := filepath.Join(t.TempDir(), "no-config.yaml")
        prevCfg, hadCfg := os.LookupEnv("SKILLDOZER_CONFIG")
        if err := os.Setenv("SKILLDOZER_CONFIG", cfgGhost); err != nil {
            t.Fatalf("setenv SKILLDOZER_CONFIG: %v", err)
        }
        t.Cleanup(func() {
            if hadCfg {
                _ = os.Setenv("SKILLDOZER_CONFIG", prevCfg)
            } else {
                _ = os.Unsetenv("SKILLDOZER_CONFIG")
            }
        })
    (unsetEnvVar currently uses os.Unsetenv/Setenv + t.Cleanup, NOT t.Setenv, so it takes
    testing.TB — keep that; t.TempDir() works on testing.TB. filepath is already imported.)
  - GOTCHA #5: this makes TestFindRuleEnvWins / TestFindRuleWalkUpWins / TestFindAllMissReturnsErrNotFound
    hermetic automatically (they already call unsetEnvVar). No per-test change needed.
  - GOTCHA #11: the NEW findConfig tests (Task 7) set SKILLDOZER_CONFIG to real fixtures and do
    NOT call unsetEnvVar — but even if they did, a later t.Setenv("SKILLDOZER_CONFIG", real)
    would win (last-set wins). Prefer: findConfig unit tests set only SKILLDOZER_CONFIG.

Task 6: EDIT internal/skillsdir/skillsdir_test.go — flip TestErrNotFoundMessageHasFix substrings
  - FILE: internal/skillsdir/skillsdir_test.go (TestErrNotFoundMessageHasFix, ~line 507-513)
  - REPLACE the OLD substring list:
        for _, want := range []string{"SKILLDOZER_SKILLS_DIR", "cd", "reinstall"} {
    with the NEW (PRD §6.4/§8.2 message):
        for _, want := range []string{"run", "skilldozer init"} {
    Keep the loop body + t.Errorf unchanged. Optionally also assert the OLD words are GONE
    (regression guard) — optional, e.g. add a second loop over {"cd", "reinstall"} asserting
    !Contains. Keep the test focused; the contract specifies substrings {"run", "skilldozer init"}.

Task 7: EDIT internal/skillsdir/skillsdir_test.go — add 7 new tests (after the Find tests)
  - FILE: internal/skillsdir/skillsdir_test.go (append near the other Find/rule tests)
  - ADD these tests (all use t.TempDir + t.Setenv("SKILLDOZER_CONFIG", cfgPath); mirror the
    writeConfig idiom from internal/config/config_test.go and makeSkill from this file):

    // --- findConfig (PRD §8.3 rule 2 / §8.1) ---

    // writeCfg writes content to a temp config.yaml, sets SKILLDOZER_CONFIG to it, and returns
    // (cfgPath, cfgDir). Helper for the findConfig tests.
    func writeCfg(t *testing.T, content string) (cfgPath, cfgDir string) {
        t.Helper()
        cfgDir = t.TempDir()
        cfgPath = filepath.Join(cfgDir, "config.yaml")
        if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
            t.Fatalf("write %s: %v", cfgPath, err)
        }
        t.Setenv("SKILLDOZER_CONFIG", cfgPath)
        return cfgPath, cfgDir
    }

    func TestFindConfigHit(t *testing.T) {
        store := t.TempDir() // existing dir
        writeCfg(t, "store: "+store+"\n")
        got, src, found := findConfig()
        if !found { t.Fatal("findConfig existing store: found=false; want true") }
        if src != SourceConfig { t.Errorf("src=%v; want SourceConfig", src) }
        if want := filepath.Clean(store); got != want { t.Errorf("dir=%q; want cleaned %q", got, want) }
    }

    func TestFindConfigMissingFile(t *testing.T) {
        t.Setenv("SKILLDOZER_CONFIG", filepath.Join(t.TempDir(), "does-not-exist.yaml"))
        if dir, src, found := findConfig(); found {
            t.Errorf("findConfig missing file: got found=true dir=%q src=%v; want false", dir, src)
        }
    }

    func TestFindConfigMissingStoreKey(t *testing.T) {
        writeCfg(t, "foo: bar\n") // no `store:` key
        if dir, src, found := findConfig(); found {
            t.Errorf("findConfig no store key: got found=true dir=%q src=%v; want false", dir, src)
        }
    }

    func TestFindConfigStoreDirAbsent(t *testing.T) {
        writeCfg(t, "store: "+filepath.Join(t.TempDir(), "no-such-store")+"\n")
        if dir, src, found := findConfig(); found {
            t.Errorf("findConfig absent store dir: got found=true dir=%q src=%v; want false", dir, src)
        }
    }

    func TestFindConfigMalformedYAML(t *testing.T) {
        writeCfg(t, "store: [unclosed\n") // syntactically broken YAML -> Load hard-errors
        if dir, src, found := findConfig(); found {
            t.Errorf("findConfig malformed YAML: got found=true dir=%q src=%v; want false (fall through, not hard error)", dir, src)
        }
    }

    func TestFindConfigRelativeStoreResolvedAgainstConfigDir(t *testing.T) {
        // PRD §8.1: a relative `store` is resolved against the config FILE's dir, not cwd.
        cfgPath, cfgDir := writeCfg(t, "store: mystore\n")
        store := filepath.Join(cfgDir, "mystore")
        if err := os.Mkdir(store, 0o755); err != nil { t.Fatalf("mkdir %s: %v", store, err) }
        got, src, found := findConfig()
        if !found { t.Fatal("findConfig relative store: found=false; want true") }
        if src != SourceConfig { t.Errorf("src=%v; want SourceConfig", src) }
        if got != store { t.Errorf("dir=%q; want %q (joined to filepath.Dir(%q))", got, store, cfgPath) }
    }

    // --- precedence (PRD §8.3: first hit wins; env is rule 1, config is rule 2) ---

    func TestFindEnvBeatsConfig(t *testing.T) {
        // Set BOTH a valid SKILLDOZER_SKILLS_DIR and a valid config; env (rule 1) must win.
        envDir := t.TempDir()
        cfgStore := t.TempDir() // also a valid store, but config must NOT win
        cfgDir := t.TempDir()
        cfgPath := filepath.Join(cfgDir, "config.yaml")
        if err := os.WriteFile(cfgPath, []byte("store: "+cfgStore+"\n"), 0o644); err != nil {
            t.Fatalf("write %s: %v", cfgPath, err)
        }
        t.Setenv("SKILLDOZER_CONFIG", cfgPath)
        t.Setenv(envVar, envDir) // SKILLDOZER_SKILLS_DIR
        got, src, err := Find()
        if err != nil { t.Fatalf("Find() env-beats-config: err=%v; want nil", err) }
        if src != SourceEnv { t.Errorf("src=%v; want SourceEnv (env beats config)", src) }
        if want := filepath.Clean(envDir); got != want { t.Errorf("dir=%q; want env dir %q", got, want) }
    }
  - NOTE: TestFindConfigMalformedYAML is the load-bearing "never a hard error" assertion (GOTCHA #2).
  - NOTE: TestFindEnvBeatsConfig drives Find() (not findConfig) to prove precedence end-to-end.

Task 8: EDIT main_test.go — harden unsetSkillsEnv + flip the 6 substring assertions
  - FILE: main_test.go
  - (8a) EXTEND unsetSkillsEnv (:22-28) to neutralize the config rule (mirror Task 5). Current body
    is just `t.Setenv("SKILLDOZER_SKILLS_DIR", "")`. ADD (this single change covers ALL 6 tests
    below, since they all call unsetSkillsEnv):
        // Also neutralize the config-file rule (PRD §8.3 rule 2): point SKILLDOZER_CONFIG at a
        // non-existent path so findConfig deterministically misses once wired into Find().
        // Harmless when a higher-priority rule (env/sibling/walk-up) hits first.
        t.Setenv("SKILLDOZER_CONFIG", filepath.Join(t.TempDir(), "no-config.yaml"))
    (main_test.go already imports filepath + os; t.Setenv + t.TempDir are on *testing.T.)
  - (8b) TestRunPathFailureErrNotFound (:240) — replace:
        for _, want := range []string{"SKILLDOZER_SKILLS_DIR", "cd", "reinstall"} {
      with:
        for _, want := range []string{"run", "skilldozer init"} {
  - The remaining 5 tests each have the IDENTICAL assertion line — replace in EACH of them:
        if !strings.Contains(errOut.String(), "SKILLDOZER_SKILLS_DIR") {
      with:
        if !strings.Contains(errOut.String(), "skilldozer init") {
    at these exact locations (one per main mode's unresolvable test):
      (8c) TestRunListSkillsDirUnresolvableExit1 (:379)
      (8d) TestRunTagSkillsDirUnresolvable (:593)
      (8e) TestRunAllSkillsDirUnresolvable (:851)
      (8f) TestRunSearchSkillsDirUnresolvable (:1091)
      (8g) TestRunCheckSkillsDirUnresolvable (:1269)
    (Each `Contains("SKILLDOZER_SKILLS_DIR")` line is identical, so a careful find-and-replace
    scoped to each function, or 5 individual edits, is needed — do NOT blindly replace all 5 with
    one sed across the file without confirming the count is exactly 5.)
    - Also update the stale comments that say "all three §8 rules" -> "all §8.3 rules"
      (the t.Chdir comments at :230/:370/:584/:842/:1082/:1260 and the TestRunPathFailureErrNotFound
      doc comment at :224-226) for accuracy — cosmetic but keep prose honest.
  - GOTCHA #4: all 6 tests FAIL on the message flip if not updated; GOTCHA #5: they're
    non-hermetic without (8a). Both are mandatory for `go test ./...`.

Task 9: VERIFY (in isolation, then whole-module + invariants)
  - gofmt -l internal/skillsdir/ main.go        # MUST print nothing
  - go vet ./...                                 # exit 0
  - go test ./internal/skillsdir/... -v          # all pass incl. 7 new + flipped TestErrNotFoundMessageHasFix
  - go test ./...                                # ALL pass incl. the 6 updated main_test.go tests
  - git diff --quiet go.mod go.sum && echo deps unchanged
  - grep -c 'three §8\|three rules' internal/skillsdir/skillsdir.go   # expect 0 (GOTCHA #8 verify)
  - §13 acceptance: build, then run the unconfigured binary and grep the new message (Level 3).
```

### Implementation Patterns & Key Details

```go
// findConfig — the new rule-2 helper. Composes config.Path + config.Load; NEVER hard-errors.
func findConfig() (dir string, src Source, found bool) {
	p, err := config.Path()
	if err != nil {
		return "", 0, false // no bootstrap path -> fall through (PRD §8.1)
	}
	f, err := config.Load(p)
	if err != nil {
		return "", 0, false // missing/unreadable/malformed -> fall through, NEVER a hard error
	}
	if f.Store == "" {
		return "", 0, false // no `store` key -> fall through
	}
	var store string
	if filepath.IsAbs(f.Store) {
		store = filepath.Clean(f.Store) // absolute -> normalize
	} else {
		store = filepath.Join(filepath.Dir(p), f.Store) // relative -> join to config file's dir (PRD §8.1)
	}
	info, err := os.Stat(store)
	if err != nil || !info.IsDir() {
		return "", 0, false // not an existing dir -> fall through
	}
	return store, SourceConfig, true
}

// Find — the 5-rule combiner (rule 2 inserted between env and sibling).
func Find() (dir string, src Source, err error) {
	if d, s, ok := findEnv(); ok {
		return d, s, nil
	}
	if d, s, ok := findConfig(); ok { // <-- NEW, PRD §8.3 priority #2
		return d, s, nil
	}
	if d, s, ok := findSibling(); ok {
		return d, s, nil
	}
	if d, s, ok := findWalkUp(); ok {
		return d, s, nil
	}
	return "", 0, ErrNotFound
}

// ErrNotFound — EXACT PRD §6.4/§8.2 message. NOTE: bare backticks in a double-quoted
// Go string (a `\`` escape does NOT exist and is a compile error — GOTCHA #1).
var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer init`")
```

Notes easy to get wrong:
- The `errors.New` literal uses **bare backticks**, not `\``. This is the single most likely one-pass failure if an implementer copies the contract's prose verbatim.
- `findConfig` checks `err != nil` from `config.Load` — it does NOT use `errors.Is(err, fs.ErrNotExist)`. Any Load error (missing file, permission denied, broken YAML) is a fall-through. This is intentional and matches PRD §8.1 ("never a hard error").
- Relative `store` joins to `filepath.Dir(configPath)`, the config FILE's directory — NOT `filepath.Abs` (which is cwd-relative). PRD §8.1 is explicit: "store may be relative to the config file".

### DESIGN DECISIONS (the judgment calls this PRP resolves)

1. **Backticks: double-quoted with bare backticks, not a raw string literal.** A raw string (backtick-delimited) cannot contain backticks without `+` concatenation (`"run ` + "`" + `skilldozer init` + "`"`), which is ugly. The double-quoted form `"run `skilldozer init`"` is the simplest correct literal. Verified to compile and match the §13 grep.

2. **Absolute `store`: `filepath.Clean`, not verbatim.** The contract says "if IsAbs, use it". An absolute store is already absolute, but `filepath.Clean` normalizes any `..`/`./`/trailing-slash noise so the returned dir (and `--path` stdout) is canonical, matching `findEnv`'s `filepath.Abs` discipline. `filepath.Join` (the relative branch) already cleans. One-liner, harmless, idiomatic.

3. **Neutralize the config rule in the shared helpers (unsetEnvVar / unsetSkillsEnv), not per-test.** Every "which rule wins" test calls these helpers; adding config-neutralization there makes ALL of them hermetic at once, and it's harmless when a higher-priority rule hits. The alternative (per-test `t.Setenv`) is repetitive and easy to forget. The helpers already use os.Setenv+t.Cleanup / t.Setenv, so the addition is local.

4. **Expand scope into main_test.go (the contract scoped tests to skillsdir_test.go only).** The message flip is T2.S2's exclusive deliverable; it breaks **6** main_test.go tests that assert the OLD substrings (one per main mode: path/list/tag/all/search/check). "You break it, you fix it": updating them in the same changeset is mandatory for `go test ./...` to pass, and no other task owns it. Leaving them = 6 failing tests = a stalled implementation. This is not optional scope creep; it's the direct, unavoidable consequence of the message flip. A fresh-context scout re-verification found the COMPLETE set of 6 (the contract's author and my first analysis both saw only the path/list/tag trio; the all/search/check unresolvable tests assert the identical `Contains("SKILLDOZER_SKILLS_DIR")` line and were easy to miss in the 67KB main_test.go).

5. **ErrNotFound DOC COMMENT: verify, don't re-edit.** T2.S1 (lands first) already refreshed it from "all three §8 rules" to "every §8.3 rule misses (unconfigured)". T2.S2's contract also mentions the doc comment, but re-editing it when T2.S1 already did risks a collision and is wasted churn. Treat as a grep verify-gate (expect 0 hits for "three §8"); edit only if T2.S1 somehow did not land.

6. **`filepath.Join` for the relative branch (not `filepath.Abs`).** `filepath.Join(filepath.Dir(p), f.Store)` both joins AND cleans in one call, and resolves against the config file's directory per PRD §8.1. `filepath.Abs(f.Store)` would resolve against cwd — wrong for a config-relative store.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod UNCHANGED. T2.S2 adds ONE import: github.com/dabstractor/skilldozer/internal/config
    (internal, already landed P1.M1.T1; uses only yaml.v3 which is already pinned). No go get,
    no go mod tidy. (GOTCHA #10)
  - git diff --quiet go.mod go.sum MUST report "deps unchanged".

CONSUMERS (NOT built in this subtask — listed to fix the interface):
  - main.go --path/list/search/check/all/tag (NO CHANGE): all call skillsdir.Find() and print
        err via fmt.Fprintln(stderr, err) (verbatim) + --path prints (found via %s) from
        Source.String(). The new message + "config file" label render with zero main.go change.
  - init (P1.M2.T2): writes the config (config.Save) that findConfig reads; this subtask's
        findConfig is what makes a configured skilldozer actually use it.
  - §13 acceptance (P1.M4.T1.S1): greps 'run `skilldozer init`' (unconfigured) and
        /tmp/skilldozer-store (config wins) and SKILLDOZER_SKILLS_DIR (env beats config) — all
        depend on this subtask's findConfig wiring + message flip.

NO ROUTES / NO DATABASE / NO CONFIG-FIELD-ADDITIONS / NO NEW FILES.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after editing skillsdir.go)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l internal/skillsdir/ main.go   # must print NOTHING (run gofmt -w if it lists a file)
go vet ./internal/skillsdir/...        # expect exit 0
go build ./internal/skillsdir/...      # expect exit 0 (proves the import + findConfig compile;
                                       #   also proves GOTCHA #1 — a `\`` escape would FAIL here)
# Expected: zero output / exit 0.
```

### Level 2: Unit Tests (component validation — the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test ./internal/skillsdir/... -v
# Expected: ALL pass. The load-bearing assertions:
#   TestFindConfigHit               -> SourceConfig + cleaned abs dir (proves the hit path).
#   TestFindConfigMissingFile       -> found=false (config.Load fs.ErrNotExist -> fall through).
#   TestFindConfigMissingStoreKey   -> found=false (f.Store == "" -> fall through).
#   TestFindConfigStoreDirAbsent    -> found=false (os.Stat miss -> fall through).
#   TestFindConfigMalformedYAML     -> found=false (broken YAML -> fall through, NOT hard error). [GOTCHA #2]
#   TestFindConfigRelativeStoreResolvedAgainstConfigDir -> joined to filepath.Dir(cfg). [GOTCHA #3]
#   TestFindEnvBeatsConfig          -> SourceEnv wins over a valid config (precedence).
#   TestErrNotFoundMessageHasFix    -> asserts {"run","skilldozer init"} (flipped; [Task 6]).
#   TestFindRuleEnvWins / WalkUpWins / AllMissReturnsErrNotFound -> still pass (hermetic via
#     the hardened unsetEnvVar; [GOTCHA #5]).

# Isolated re-run of just the new + flipped behaviors:
go test ./internal/skillsdir/... -run 'TestFindConfig|TestFindEnvBeatsConfig|TestErrNotFound' -v
# Expected: PASS.
```

### Level 3: Whole-module regression + §13 acceptance

```bash
cd /home/dustin/projects/skilldozer

go build ./... ; echo "build exit $?"   # 0
go vet  ./...  ; echo "vet exit $?"     # 0
go test ./...  ; echo "test exit $?"    # 0  — CRITICAL: includes the 6 updated main_test.go tests

# GOTCHA #10 invariant: go.mod / go.sum byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"

# GOTCHA #8 verify: no stale "three §8 rules" doc comment remains
grep -c 'three §8\|three rules' internal/skillsdir/skillsdir.go   # expect 0

# §13 acceptance — the unconfigured message + the config-wins + env-beats-config lines:
go build -o /tmp/sd .
# unconfigured (clean HOME, no config, no skills sibling, no walk-up ancestor): hint + exit 1
tmp=$(mktemp -d)
env -u SKILLDOZER_SKILLS_DIR HOME="$tmp/home" XDG_CONFIG_HOME="$tmp/home/.config" \
  XDG_DATA_HOME="$tmp/data" /tmp/sd x 2>"$tmp/err"; rc=$?
[ "$rc" = 1 ] && grep -q 'run `skilldozer init`' "$tmp/err" && echo "unconfigured-hint OK"
# config rule wins (write a config with an existing store; --path reports it + the label)
store=$(mktemp -d); mkdir -p "$tmp/cfgdir"
printf 'store: %s\n' "$store" > "$tmp/cfgdir/config.yaml"
env -u SKILLDOZER_SKILLS_DIR SKILLDOZER_CONFIG="$tmp/cfgdir/config.yaml" /tmp/sd --path >"$tmp/out" 2>"$tmp/perr"
grep -q "$store" "$tmp/out" && grep -q 'config file' "$tmp/perr" && echo "config-rule-wins OK"
# env still beats config
env SKILLDOZER_SKILLS_DIR="$store" SKILLDOZER_CONFIG="$tmp/cfgdir/config.yaml" /tmp/sd --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR && echo "env-beats-config OK"
rm -rf "$tmp" /tmp/sd
# Expected: all three echo lines ("unconfigured-hint OK", "config-rule-wins OK", "env-beats-config OK").
```

### Level 4: Creative & Domain-Specific Validation

```bash
cd /home/dustin/projects/skilldozer

# 4a. Lock GOTCHA #1 at runtime — prove the message has LITERAL backticks (not stripped/escaped):
go build -o /tmp/sd . && env -u SKILLDOZER_SKILLS_DIR HOME="$(mktemp -d)" XDG_CONFIG_HOME="$(mktemp -d)" \
  /tmp/sd x 2>&1 | od -c | grep -q '`   s   k   i   l   l   d   o   z   e   r' && echo "backticks present OK" || echo "FAIL: no literal backtick"
rm -f /tmp/sd
# Expected: "backticks present OK" (proves the message contains a literal ` before "skilldozer init").

# 4b. Prove --path surfaces the "config file" label (the SourceConfig label T2.S1 added renders
#     via src.String() with zero main.go change):
go build -o /tmp/sd .
store=$(mktemp -d); cfg=$(mktemp -d)/config.yaml; printf 'store: %s\n' "$store" > "$cfg"
env -u SKILLDOZER_SKILLS_DIR SKILLDOZER_CONFIG="$cfg" /tmp/sd --path 2>&1 1>/dev/null | grep -q '(found via config file)' && echo "config-file-label OK"
rm -rf "$store" "$(dirname "$cfg")" /tmp/sd
# Expected: "config-file-label OK".

# 4c. Hermeticity spot-check — the all-miss tests must not depend on a real ~/.config/skilldozer:
#     temporarily CREATE a fake config at the default path and re-run the skillsdir tests; they
#     must STILL pass (because unsetEnvVar now neutralizes SKILLDOZER_CONFIG to a ghost path).
mkdir -p /tmp/fakehome/.config/skilldozer && printf 'store: /tmp/should-not-win\n' > /tmp/fakehome/.config/skilldozer/config.yaml
mkdir -p /tmp/fakehome-fakestore  # make it a REAL dir so findConfig would hit without neutralization
printf 'store: %s\n' /tmp/fakehome-fakestore > /tmp/fakehome/.config/skilldozer/config.yaml
HOME=/tmp/fakehome XDG_CONFIG_HOME=/tmp/fakehome/.config go test ./internal/skillsdir/... -run 'TestFindAllMissReturnsErrNotFound|TestFindRuleWalkUpWins' -v
# Expected: PASS even with a real default config present (proves GOTCHA #5 neutralization works).
rm -rf /tmp/fakehome /tmp/fakehome-fakestore
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l internal/skillsdir/ main.go` empty; `go vet ./internal/skillsdir/...` exit 0; `go build` exit 0 (proves GOTCHA #1 — no `\`` escape error)
- [ ] Level 2 PASS — `go test ./internal/skillsdir/... -v` all pass; the 7 new findConfig/precedence tests pass; `TestErrNotFoundMessageHasFix` asserts the new substrings
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0 (incl. the **6** updated main_test.go tests); `git diff go.mod go.sum` reports "deps unchanged"; `grep -c 'three §8' …skillsdir.go` == 0; the §13 acceptance lines all echo OK
- [ ] Level 4 PASS — the literal-backtick `od -c` check passes; `--path` shows `(found via config file)`; the all-miss tests pass even with a real default config present

### Feature Validation
- [ ] `findConfig()` exists with signature `(dir string, src Source, found bool)`; returns `(absStore, SourceConfig, true)` on hit, `("", 0, false)` on every miss
- [ ] `Find()` consults `findEnv → findConfig → findSibling → findWalkUp` and returns the first hit; total miss → `("", 0, ErrNotFound)`
- [ ] `ErrNotFound` message is EXACTLY `skilldozer is not configured; run `skilldozer init`` (literal backticks; §13 grep passes)
- [ ] Relative `store` resolves against the config file's dir (PRD §8.1); absolute `store` is cleaned
- [ ] `SKILLDOZER_SKILLS_DIR` beats a valid config (precedence); `--path` reports `config file` when config wins
- [ ] `main.go` is UNCHANGED (the message + label flow via `err.Error()` / `src.String()`)

### Code Quality / Convention Validation
- [ ] Matches the existing per-rule helper shape (`findEnv`/`findSibling`/`findWalkUp`); `findConfig` reads in priority order in the source
- [ ] Matches the existing test style (table-driven where applicable; direct got!=want assertions; `t.TempDir`/`t.Setenv`; no testify)
- [ ] Matches `internal/config/config_test.go`'s `writeConfig` idiom for the findConfig fixtures
- [ ] No new external deps; one new import (`internal/config`); `go.mod`/`go.sum` unchanged
- [ ] No new files; all edits to the 3 existing files
- [ ] Stale "all three §8 rules" prose in main_test.go comments updated to "all §8.3 rules"

### Scope Discipline (the T2.S1 boundary + the research-discovered expansion)
- [ ] Did NOT re-add `SourceConfig` / the `"config file"` label / `HasSkillMD` (T2.S1 owns those)
- [ ] Did NOT re-edit the ErrNotFound/package/Find DOC COMMENTS (T2.S1 owns those; this is a verify-gate only)
- [ ] DID update main_test.go (6 substring flips + unsetSkillsEnv hardening) — mandatory consequence of the message flip, discovered + scout-re-verified in research
- [ ] Did NOT touch `main.go`, `internal/config`, `README.md`, completions, the example skill, or `PRD.md`/`tasks.json`/`prd_snapshot.md`/`.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't write `errors.New("...\`skilldozer init\`...")`.** A `\`` is not a valid Go escape in a double-quoted string — it's a compile error. Use BARE backticks: `"run `skilldozer init`"`. (GOTCHA #1 — the #1 stall risk.)
- ❌ **Don't let `findConfig` return or propagate an error.** The per-rule shape is `(dir, src, found)`; ANY miss (missing file, missing key, missing dir, permission denied, broken YAML) → `return "", 0, false`. PRD §8.1: never a hard error. (GOTCHA #2.)
- ❌ **Don't resolve a relative `store` against cwd (`filepath.Abs`).** PRD §8.1 says store may be relative to the CONFIG FILE — use `filepath.Join(filepath.Dir(configPath), store)`. (GOTCHA #3.)
- ❌ **Don't skip main_test.go.** The contract scoped tests to skillsdir_test.go, but **6** main_test.go tests (path/list/tag/all/search/check unresolvable) assert the OLD ErrNotFound substrings and break on the flip. Updating all 6 is mandatory for `go test ./...`. (GOTCHA #4.)
- ❌ **Don't leave the all-miss tests env-only-neutralized.** Once findConfig is wired, a machine with a real config makes them fail. Harden `unsetEnvVar`/`unsetSkillsEnv` to neutralize SKILLDOZER_CONFIG too. (GOTCHA #5.)
- ❌ **Don't touch `main.go`.** Every `Find()` caller prints `err` verbatim; `--path` uses `src.String()`. The new message + "config file" label render with zero change. (GOTCHA #6.)
- ❌ **Don't re-edit the ErrNotFound doc comment.** T2.S1 already refreshed it to the 5-rule ladder. Verify (grep for "three §8" → expect 0); edit only if T2.S1 didn't land. (GOTCHA #8.)
- ❌ **Don't append `findConfig` at the end of the file or call it last in `Find()`.** It is priority #2 — place the definition between `findEnv` and `findSibling`, and the call between the env and sibling calls, so source order tracks priority order. (GOTCHA #9.)
- ❌ **Don't `go get`/`go mod tidy`/add an external dep.** The single new import is internal/config. `go.mod`/`go.sum` must be byte-for-byte unchanged. (GOTCHA #10.)
- ❌ **Don't make the findConfig unit tests call `unsetEnvVar` then rely on `SKILLDOZER_CONFIG`.** They set `SKILLDOZER_CONFIG` to a real fixture directly; the env-neutralizing helper is for the Find()-combiner tests. (GOTCHA #11.)

---

## Confidence Score

**9/10** — The contract is unusually detailed and the change is small (one helper, one inserted call, one message flip), but it carries **two non-obvious, empirically-verified failure modes** that an unguided implementer would hit: (1) the Go backtick-escape compile error (a literal copy of the contract's `\``-laden prose fails instantly), and (2) the `main_test.go` breakage + hermeticity risk that the contract's "skillsdir_test.go only" scoping omits. Both are caught and pinned with grep/runtime gates in the validation loop. The `config` API consumed is fully landed and read in full; the T2.S1 boundary is crisp; main.go's no-change claim is confirmed by reading lines 408-428. The 1-point reservation is for the parallel-execution seam with T2.S1: if T2.S1's doc-comment refresh and this PRP's "verify, don't re-edit" guidance drift (e.g. T2.S1 lands but a reviewer reverts the doc comment), the GOTCHA #8 grep gate is the backstop. The main_test.go scope expansion is a judgment call this PRP makes explicitly and defends in DESIGN DECISION #4.
