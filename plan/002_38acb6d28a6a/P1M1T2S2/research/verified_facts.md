# Verified facts — P1.M1.T2.S2 (`findConfig` + wire into `Find()` + exact `ErrNotFound` message)

Every claim below is anchored to a file read in full or a command run on the live
repo at `/home/dustin/projects/skilldozer` (state: P1.M1.T2.S1 in-flight in
parallel; this PRP assumes T2.S1 lands first — see §6 boundary).

---

## 1. The two files under edit (current state — T2.S1 has LANDED; symbols stable)

`internal/skillsdir/skillsdir.go` (post-T2.S1 line numbers):
- `Find()` body (`:247-258`) — consults `findEnv` (`:248`) → `findSibling` (`:250`) → `findWalkUp`
  only; on total miss returns `("", 0, ErrNotFound)`. **T2.S2 inserts `findConfig` between `:248`
  and `:250`.**
- `var ErrNotFound` (`:234`) — message still the OLD string; **T2.S2 flips it.** (T2.S1 already
  refreshed the DOC comment above it (`:231-233`) to "every §8.3 rule misses" — no "three" remains;
  grep confirms 0 hits. T2.S2's ErrNotFound doc-comment item is therefore a no-op verify.)
- `SourceConfig` enum (`:35-36`, iota pos 1) + `case SourceConfig: return "config file"` (`:49-50`)
  + exported `HasSkillMD` (`:166`) — ALL present from T2.S1.
- Import block (`:12-17`): `errors`, `io/fs`, `os`, `path/filepath`. **T2.S2 ADDS one import:**
  `github.com/dabstractor/skilldozer/internal/config`. (No external dep — config is internal;
  go.mod/go.sum unchanged.)

`internal/skillsdir/skillsdir_test.go`:
- `TestErrNotFoundMessageHasFix` asserts OLD substrings `{"SKILLDOZER_SKILLS_DIR","cd","reinstall"}` → must flip to `{"run","skilldozer init"}`.
- `TestFindRuleEnvWins`, `TestFindRuleWalkUpWins`, `TestFindAllMissReturnsErrNotFound` — call `unsetEnvVar(t)` (neutralizes SKILLDOZER_SKILLS_DIR ONLY) + some `t.Chdir`. **Become non-hermetic once findConfig is wired** (see §4).

## 2. `main.go` needs NO change (verified)

All five `skillsdir.Find()` call sites (`main.go:408,431,467,507,548,579`) print the
error via `fmt.Fprintln(stderr, err)` — `err.Error()` verbatim, NO prefix. When
ErrNotFound's message flips, every mode (path/list/search/check/all/tag) prints the
new line automatically. `--path` also does `fmt.Fprintf(stderr, "(found via %s)\n", src)`
→ once `findConfig` returns `SourceConfig`, `(found via config file)` renders via the
Stringer T2.S1 already added. **Confirmed: zero main.go edits.**

## 3. CRITICAL — `main_test.go` ALSO breaks: SIX tests, not three (scout-re-verified)

Six `main_test.go` tests assert the OLD ErrNotFound message AND rely on "all rules
miss" with env-only neutralization (one per main mode that can hit the unconfigured path).
The first analysis caught only path/list/tag; a fresh-context scout pass over the full
67KB main_test.go found all/search/check too — they assert the IDENTICAL line and were
trivially missed:

| Test | File:line | Asserts (OLD) | Needs → NEW |
|---|---|---|---|
| `TestRunPathFailureErrNotFound` | main_test.go:228 | `[]string{"SKILLDOZER_SKILLS_DIR","cd","reinstall"}` (:240) | `{"run","skilldozer init"}` |
| `TestRunListSkillsDirUnresolvableExit1` | main_test.go:368 | `Contains(errOut,"SKILLDOZER_SKILLS_DIR")` (:379) | `Contains(...,"skilldozer init")` |
| `TestRunTagSkillsDirUnresolvable` | main_test.go:582 | `Contains(errOut,"SKILLDOZER_SKILLS_DIR")` (:593) | `Contains(...,"skilldozer init")` |
| `TestRunAllSkillsDirUnresolvable` | main_test.go:840 | `Contains(errOut,"SKILLDOZER_SKILLS_DIR")` (:851) | `Contains(...,"skilldozer init")` |
| `TestRunSearchSkillsDirUnresolvable` | main_test.go:1080 | `Contains(errOut,"SKILLDOZER_SKILLS_DIR")` (:1091) | `Contains(...,"skilldozer init")` |
| `TestRunCheckSkillsDirUnresolvable` | main_test.go:1258 | `Contains(errOut,"SKILLDOZER_SKILLS_DIR")` (:1269) | `Contains(...,"skilldozer init")` |

All six call `unsetSkillsEnv(t)` (main_test.go:22-28 — sets `SKILLDOZER_SKILLS_DIR=""` ONLY)
+ `t.Chdir(t.TempDir())`. **None neutralize the config rule** → once findConfig is wired, on
any machine with a real config (CI after `init`, dev machine, `SKILLDOZER_CONFIG` set)
`findConfig` would HIT and these tests return a real dir instead of ErrNotFound → fail.
Even on THIS machine (no config) the message-substring assertions break immediately on
the flip. **Resolution (mandated for `go test ./...` to pass):** (a) flip all six substring
assertions to the new message; (b) harden `unsetSkillsEnv` to ALSO neutralize
SKILLDOZER_CONFIG — covers all six at once since they all call it. (Same fix to `unsetEnvVar`
in skillsdir_test.go for TestFindRuleWalkUpWins / TestFindAllMissReturnsErrNotFound.)

**DO NOT touch** the `--path`-SUCCESS tests (main_test.go:182/198/219) that assert
`(found via SKILLDOZER_SKILLS_DIR)\n` — that is the SourceEnv SOURCE LABEL (src.String()),
not the ErrNotFound message; unaffected by the flip.

**No other `*_test.go` asserts the old message** (grepped internal/{check,discover,resolve,
search,ui,config}/*_test.go — zero hits). main_test.go (6) + skillsdir_test.go (1:
TestErrNotFoundMessageHasFix) are the COMPLETE breakage set (7 total).

## 4. Hermeticity gotcha — the config rule makes existing Find() tests machine-dependent

`config.Path()` reads `$SKILLDOZER_CONFIG` else `os.UserConfigDir()` → `$XDG_CONFIG_HOME/skilldozer/config.yaml` (→ `~/.config/skilldozer/config.yaml`). On THIS machine: SKILLDOZER_CONFIG unset, no `~/.config/skilldozer/config.yaml` → findConfig misses today. But any machine that ran `skilldozer init` or set the env has a real config → findConfig hits → existing "all-miss" tests break.

Tests that MUST neutralize config to stay hermetic (currently use env-only neutralization):
- skillsdir_test.go: `TestFindRuleWalkUpWins`, `TestFindAllMissReturnsErrNotFound` (and `TestFindRuleEnvWins` for consistency — env hits first so it's technically safe, but harden it).
- main_test.go: the three in §3.

`findConfig` UNIT tests and the env-beats-config precedence test SET `SKILLDOZER_CONFIG` to real fixtures intentionally (they don't neutralize).

## 5. CRITICAL — Go string-literal gotcha for the backticks in the new message

The new message is `skilldozer is not configured; run \`skilldozer init\`` (PRD §8.2 /
§6.4; §13 greps `run \`skilldozer init\``). In the contract/prose the backticks appear
escaped as `\``, but **that is markdown/shell rendering convention, NOT Go syntax.**

Empirically verified on go1.25:
- `errors.New("...run \`skilldozer init\`")` → **COMPILE ERROR**: `unknown escape` (a
  double-quoted Go string has NO `\`` escape; backslash must precede a recognized
  escape sequence).
- `errors.New("...run `skilldozer init`")` → **VALID**: a backtick is an ordinary
  character in a double-quoted string; produces exactly the bytes `run `skilldozer init``
  that the §13 grep matches (`grep -q 'run \`skilldozer init\`'` → GREP OK).

**Therefore the literal MUST use UNESCAPED backticks:**
```go
var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer init`")
```
An implementer who copies the contract's `\``-laden prose verbatim hits an instant
compile failure — the #1 one-pass stall risk for this subtask.

## 6. Status of the parallel sibling P1.M1.T2.S1: LANDED (verified at PRP-write time)

T2.S1 is COMPLETE in the live tree (confirmed by reading skillsdir.go): `SourceConfig`
enum (`:35-36`, iota value 1), `case SourceConfig: return "config file"` (`:49-50`),
exported `func HasSkillMD` (`:166`) + its refreshed doc, the 5-rule package doc comment,
the `Find()` doc comment listing rule 2 as `Config file store (SourceConfig)` (`:239`),
and the `ErrNotFound` DOC comment now reading "every §8.3 rule misses" (`:231-233` —
NO "three" remains). `TestSourceString` already has `{SourceConfig, "config file"}`.

T2.S1 explicitly did NOT touch (T2.S2's exclusive deliverables, all still pending):
- `findConfig` implementation
- `Find()` body wiring (still 3 calls: env/sibling/walkup at `:248/:250/:252`)
- `ErrNotFound` message STRING (still the OLD hint at `:234`)
- `TestErrNotFoundMessageHasFix` substrings (still OLD)

**Overlap note (ErrNotFound DOC COMMENT):** both T2.S1 and the T2.S2 contract mention the
ErrNotFound doc comment. Since T2.S1 landed and already changed "all three §8 rules" →
"every §8.3 rule misses (unconfigured)", T2.S2's doc-comment item is satisfied: treat it
as a **VERIFY gate** (`grep -c "three §8" internal/skillsdir/skillsdir.go` must be 0 —
CONFIRMED 0 in the live tree), not a re-edit. T2.S2's actual ErrNotFound edit is the
MESSAGE STRING flip only.

## 7. `findConfig` implementation — exact contract (item_description LOGIC step 3)

```go
// findConfig implements PRD §8.3 rule 2 — the config file's `store` key (§8.1).
//
// p, err := config.Path();   if err -> fall through (no bootstrap path)
// f, err := config.Load(p);  if err != nil (incl. fs.ErrNotExist) -> fall through
//                            (missing/unreadable/malformed -> NEVER a hard error)
// if f.Store == "" -> fall through (missing key)
// absolutize: if IsAbs(f.Store) use filepath.Clean(f.Store),
//             else filepath.Join(filepath.Dir(p), f.Store)  (§8.1: store may be relative to cfg)
// os.Stat the resolved store; if !existing dir -> fall through
// else return (absStore, SourceConfig, true)
func findConfig() (dir string, src Source, found bool)
```
Returns shape `(dir string, src Source, found bool)` matching findEnv/findSibling/findWalkUp.
config.Load returns os.ReadFile errors VERBATIM (not wrapped) so any read error → `err != nil`
→ fall through. findConfig does NOT import io/fs (no errors.Is needed — any error = fall through).

## 8. New tests to add (item_description OUTPUT §4)

All use `t.Setenv("SKILLDOZER_CONFIG", cfgPath)` to point findConfig at a fixture under
`t.TempDir()` (mirrors config_test.go's `writeConfig` helper + skillsdir_test.go's `makeSkill`):
1. `TestFindConfigHit` — existing store dir → SourceConfig + cleaned abs dir.
2. `TestFindConfigMissingFile` — SKILLDOZER_CONFIG → non-existent path → fall through.
3. `TestFindConfigMissingStoreKey` — `foo: bar\n` (no store) → fall through.
4. `TestFindConfigStoreDirAbsent` — `store: /nonexistent/x\n` → fall through.
5. `TestFindConfigMalformedYAML` — `store: [unclosed\n` → fall through (NOT a hard error).
6. `TestFindConfigRelativeStoreResolvedAgainstConfigDir` — relative `store` joined to
   `filepath.Dir(cfgPath)` (PRD §8.1 — bonus coverage of the relative-store rule).
7. `TestFindEnvBeatsConfig` — set BOTH SKILLDOZER_SKILLS_DIR (existing dir) AND a valid
   config; `Find()` returns SourceEnv (env beats config). Drives `Find()`, not `findConfig`.

## 9. Validation gates (verified executable on this repo)

```bash
cd /home/dustin/projects/skilldozer
gofmt -l internal/skillsdir/ main.go         # empty
go vet ./...                                  # exit 0
go test ./...                                 # ALL pass (incl. updated main_test.go + skillsdir_test.go)
go build -o /tmp/sd . && /tmp/sd --version    # builds; prints version
# §13 message grep (the load-bearing acceptance line):
env -u SKILLDOZER_SKILLS_DIR XDG_CONFIG_HOME=/tmp/nope HOME=/tmp/nope /tmp/sd x 2>/tmp/e; rc=$?
[ "$rc" = 1 ] && grep -q 'run `skilldozer init`' /tmp/e && echo "unconfigured-hint OK"
# deps unchanged:
git diff --quiet go.mod go.sum && echo "deps unchanged"
```

## 10. Confidence: high. Two non-obvious risks caught + empirically verified:
(a) backtick-escape compile error (§5), (b) main_test.go breakage + hermeticity (§3/§4).
Both are exactly the "80% stall" failures a PRP exists to prevent.
