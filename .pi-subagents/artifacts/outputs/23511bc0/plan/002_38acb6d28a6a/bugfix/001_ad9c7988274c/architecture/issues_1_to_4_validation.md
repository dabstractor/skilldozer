# Issues 1–4: Reproduction & Fix-Location Validation

Binary under test: `./skilldozer` (ELF x86-64, statically linked, built from current
`main.go` @ HEAD `414758d`; verified by matching embedded error-message strings).
Repo: `/home/dustin/projects/skilldozer`. Git working tree clean (no staged files;
only untracked PRD snapshots under `plan/...`).

All repros run against the prebuilt binary exactly as the PRD specifies.

---

## ISSUE 1 — `init` writes the check validation report to stdout

### (a) Confirmed repro output
Command (PRD exact):
```
cd /tmp && mkdir -p /tmp/A/store
SKILLDOZER_CONFIG=/tmp/A/cfg.yaml env -u SKILLDOZER_SKILLS_DIR ./skilldozer init --store /tmp/A/store </dev/null 2>/dev/null
```
stdout captured (3 lines):
```
/tmp/A/store
OK    example (example)
1 skills, 0 errors, 0 warnings
```
exit=0.

stderr (separate run, `2>/tmp/A3.err >/dev/null`) is already correct:
```
Seeded example skill at /tmp/A3/store/example/SKILL.md
(found via config file)
```

**Bug:** stdout contains 3 lines. PRD §6.1 mandates stdout carry ONLY the
configured store path (the `$(...)` headline). The `OK ...` per-skill line and the
`N skills, M errors, K warnings` summary belong on **stderr** (the check report is
informational, like the `(found via %s)` label that is already correctly on stderr).

### (b) Exact location
- **File:** `main.go`
- **Function:** `runInit(c config, stdout, stderr io.Writer) int` — defined at **line 988**.
- The check-report render block is **lines 1037–1053**.
  - Line **1046**: `fmt.Fprintf(stdout, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)` — OK line.
  - Line **1050**: `fmt.Fprintf(stdout, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)` — per-finding line.
  - Line **1053**: `fmt.Fprintf(stdout, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)` — summary.
- Line **1026** `fmt.Fprintln(stdout, dir)` — the store-path headline. **STAYS on stdout** (PRD §6.1).
- Line **1029** `fmt.Fprintf(stderr, "(found via %s)\n", src)` — already stderr, correct.

### (c) Precise change
In `runInit` (main.go lines 1046, 1050, 1053) change the `stdout` writer to
`stderr` for those three `fmt.Fprintf` calls (the check-report loop + summary).
Mirror exactly the `if c.check` branch (main.go lines 577/581/584) except that
those use `stdout`; in `runInit` they must use `stderr`. Leave line 1026
(`fmt.Fprintln(stdout, dir)`) untouched. No change to exit codes (init stays exit 0
regardless of check findings — the comment at line 1054 still holds).

### (d) Related tests
- `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (main_test.go **line
  2325**) — asserts `strings.Contains(out.String(), store)` (line 2365). This
  assertion PASSES even with the buggy extra stdout lines, so it does NOT catch
  the bug. **Strengthen**: also assert stdout contains EXACTLY one line / does NOT
  contain `"skills,"` / does NOT contain `"OK"`. Recommend adding
  `if strings.Contains(out.String(), "errors")` failure guard, and assert the
  `OK`/summary lines appear in `errOut` instead.

---

## ISSUE 2 — `init --store` with no value silently overwrites config

### (a) Confirmed repro output
Command (PRD exact, config has `skillsDir` + `preserve: true`):
```
rm -rf /tmp/B && mkdir -p /tmp/B
printf 'skillsDir: /tmp/B/EXISTING_STORE\npreserve: true\n' > /tmp/B/cfg.yaml
SKILLDOZER_CONFIG=/tmp/B/cfg.yaml env -u SKILLDOZER_SKILLS_DIR ./skilldozer init --store </dev/null
```
Before:
```
skillsDir: /tmp/B/EXISTING_STORE
preserve: true
```
Run output (note it PROMPTED then auto-resolved to the repo cwd):
```
Where should skilldozer keep your skills? [/home/dustin/projects/skilldozer]: Adopted existing store at /home/dustin/projects/skilldozer
(found via config file)
/home/dustin/projects/skilldozer
OK    skills/example (example)
1 skills, 0 errors, 0 warnings
```
exit=0.
After (config DESTROYED — `skillsDir` AND `preserve: true` both gone):
```
store: /home/dustin/projects/skilldozer
```

**`--store` with NO `init` token** (`skilldozer --store`, last token, no value):
falls through to no-args usage, exit 1, **config untouched**. So the dangerous
shape is specifically `init --store` (the `init` token sets `c.init=true`; the
trailing `--store` sets nothing).

### (b) Exact location — two places
1. **parseArgs, `case "--store":`** — `main.go` **lines 257–266**.
   The guard `if i+1 < len(args) {` at **line 263** silently skips when `--store`
   is the LAST token (no value follows). The comment (lines 258–262) explicitly
   documents this "deferred" behavior as intentional, but it is unsound when the
   `init` token already set `c.init=true`.
2. **parseArgs, `'='`-form** — `main.go` **lines 192–197** (`case "--store":`
   inside the `--flag=value` splitter). `--store=` with empty value similarly
   sets `c.init=true; c.initStore=""` and proceeds. (Lower risk: an explicit
   empty value is arguably intentional, but it lands in the same overwrite path.)
3. **run()** — `main.go` **line 447**: `if c.init { return runInit(c, stdout,
   stderr) }` dispatches with no validation that `c.initStore` is non-empty. This
   is where a missing-value guard belongs (run() already owns the
   exit-code/precedence discipline and is where `unknownFlag`/`exclusivityError`
   are checked before dispatch).

### (c) Precise change
Two complementary options (PRD should pick; both are minimal):
- **Option A (preferred, in parseArgs):** when `--store` is seen with no following
  token (and likewise `--store=` with empty value), record a missing-value error
  on the config (e.g. a new `c.storeMissingValue = true` field, or reuse the
  `unknownFlag` channel with a tailored message) and let `run()` emit a
  `skilldozer: --store requires a value` message and return exit 2. This mirrors
  the existing precedent that `--search` with no value leaves `searchMode=false`
  and falls through — EXCEPT here `c.init` is already true, so a silent fallthrough
  is destructive; an explicit error is required.
- **Option B (in run):** at main.go ~line 447, before dispatching init, if
  `c.init && c.initStore == ""` AND `--store` appeared in argv (track via a
  `c.storeSeen` bool), reject with exit 2 + a "missing value" message. (Bare
  `skilldozer init` with no `--store` at all must keep prompting — so the guard
  must distinguish "init with --store-but-no-value" from "init with no --store".)

The `c.initStore == ""` test alone is INSUFFICIENT: bare `skilldozer init`
legitimately has `c.initStore == ""` and must prompt. The distinguishing signal
must be that `--store` was actually present in argv.

### (d) Related tests
- `TestParseArgsStoreWithoutInitToken` (main_test.go **line 1212**) — covers
  `--store /tmp/x` WITH a value. Does NOT cover the no-value case.
- `TestParseArgsInitStoreLongForm` (line 1186), `TestParseArgsInitStoreEqualsForm`
  (line 1200) — both with values.
- **Gap:** no test for `init --store` (trailing, no value) or `--store=` (empty
  value). Add `TestParseArgsInitStoreNoValue` asserting the missing-value signal is
  set (Option A) and a `run()`-level test asserting exit 2 + non-destructive
  config (do NOT write the config file).

---

## ISSUE 3 — `tag + --path` not rejected

### (a) Confirmed repro output
Command (PRD exact):
```
export SKILLDOZER_SKILLS_DIR=/tmp/sk; mkdir -p /tmp/sk
./skilldozer NONEXISTENTTAG --path; echo exit=$?
```
Output:
```
/tmp/sk
(found via SKILLDOZER_SKILLS_DIR)
exit=0
```
**Bug:** exit 0; the stray `NONEXISTENTTAG` is silently ignored and `--path` wins.
Expected exit 2 (mutual exclusion). Control: `./skilldozer NONEXISTENTTAG --list`
correctly returns exit 2 with `skilldozer: tags cannot be combined with
--list/--search/--all`.

### (b) Exact location
- **File:** `main.go`
- **Function:** `exclusivityError(c config) (bad bool, msg string)` — defined at **line 686**.
- Buggy predicate — **line 702**:
  ```go
  if hasTags && (c.list || c.searchMode || c.all) {
      return true, "skilldozer: tags cannot be combined with --list/--search/--all"
  }
  ```
  `c.path` is **omitted** from the disjunction.
- Note the asymmetry: the count set at lines 695 (`{c.path, c.list, c.searchMode,
  c.all}`) and the `check + mode` predicate at **line 708**
  (`c.path || c.list || c.searchMode || c.all`) BOTH include `c.path`. Only the
  `tags +` predicate at line 702 drops it. So `check --path` is correctly rejected
  (TestRunExclusivityCheckAndPath), but `tag --path` is not.

### (c) Precise change
Add `c.path` to the predicate at main.go line 702:
```go
if hasTags && (c.path || c.list || c.searchMode || c.all) {
    return true, "skilldozer: tags cannot be combined with --path/--list/--search/--all"
}
```
(update the user-facing message string to include `--path` for consistency with
the listing-modes message at line 700). One-line predicate change + message
string update. No exit-code or ordering change.

### (d) Related tests
- `TestRunExclusivityTagsAndList` (line 1559), `TestRunExclusivityTagsAndSearch`
  (line 1574), `TestRunExclusivityTagsAndAll` (line 1586) — the sibling trio.
- `TestRunExclusivityCheckAndPath` (line 1628) — proves `check+--path` IS rejected,
  underscoring the asymmetry.
- `TestExclusivityErrorListingModes` (line 1970) — unit test of
  `exclusivityError` directly.
- **Gap:** there is NO `TestRunExclusivityTagsAndPath`. Add it (mirror
  `TestRunExclusivityTagsAndList`): `run([]string{"foo","--path"}, &out, &errOut)`
  → assert code==2, empty stdout, stderr contains "cannot be combined". Also add a
  case to `TestExclusivityErrorListingModes` table for `{tags:["foo"], path:true}`.

---

## ISSUE 4 — `init init` runs init (exit 0) instead of being rejected

### (a) Confirmed repro output
```
./skilldozer init init </dev/null >/dev/null 2>&1; echo exit=$?
```
exit=0. Config file IS written (`store: <auto>`), proving init actually ran.
Control — `./skilldozer init check </dev/null >/dev/null 2>&1; echo exit=$?` →
**exit=2** (correctly rejected by exclusivity).

**Bug:** the `init` case's GOTCHA guard swallows a following `init`/`check` as a
non-store token (so it is NOT captured into `c.initStore`), but a following
`init` then re-enters the `init` case and re-sets `c.init=true` with no conflict,
so exclusivity never fires. `check` is different: it lands in the `check` case
(`c.check=true`), and exclusivity's init branch (line 712) rejects `c.check`.

### (b) Exact location
- **File:** `main.go`
- **Function:** `parseArgs` — `case "init":` at **lines 268–284**.
  - Line 277: `c.init = true` (unconditional on the token).
  - Lines 278–283: the guard that refuses to capture `next` when it is `"check"`
    or `"init"` (`next != "check" && next != "init"` at line 281). This GUARD is
    what makes the second `init` fall through to its own `case "init":` re-entry
    rather than becoming `c.initStore`.
- **run()** dispatch — main.go **line 447**: `if c.init { return runInit(...) }`
  fires with `c.init=true` (set twice) and `c.initStore=""` (never set), so init
  runs with auto-detect.
- Exclusivity reference: `exclusivityError` init branch — main.go **lines
  711–716**: rejects `c.check || c.list || c.searchMode || c.all || c.path` but
  not a duplicate `init` (a second `init` does not set any of those; it only
  re-sets `c.init`).

### (c) Precise change
Pick ONE of:
- **Option A (in parseArgs `init` case):** after line 277, if a second reserved
  token `init`/`check` follows, record a conflict signal (e.g. set a
  `c.subcommandConflict` field or append a synthetic stray tag) so run()/exclusivity
  rejects it. Cleanest: when `next == "init"`, treat the second `init` as a stray
  positional that should trip the `init` exclusivity branch — e.g. capture it into
  `c.tags` (which the init branch at line 713 already rejects with `hasTags →
  exit 2`). Concretely: change the guard at line 281 so that `next == "init"`
  (but NOT a literal store dir, which must still use `--store`) is appended to
  `c.tags`. Then exclusivity line 713 fires `"skilldozer: 'init' cannot be
  combined with tag arguments"` (exit 2).
- **Option B (exclusivity):** add a counter/flag for repeated `init` tokens (e.g.
  `c.initCount`); in `exclusivityError` reject when `c.initCount > 1`.

Option A is more consistent with the existing design (the init branch already
rejects stray tags) and needs no new config field beyond reusing `c.tags`. The
`check` follow-token already works because it sets `c.check`, which exclusivity
catches — so only the `init` follow-token needs handling.

### (d) Related tests
- `TestParseArgsInitSubcommand` (line 1158), `TestParseArgsInitPositionalDir`
  (line 1172), `TestParseArgsInitDirNotCapturedAsTag` (line 1226) — single-init
  coverage.
- `TestRunExclusivityInitAndCheck` (line 1679) — `init check` → exit 2 (the
  control for this issue).
- **Gap:** no test for `init init`. Add `TestParseArgsInitInitDoesNotSwallow` and a
  run-level `TestRunExclusivityInitInit` asserting exit 2 + empty stdout + config
  NOT written (mirror `TestRunExclusivityInitAndCheck`).

---

## Cross-cutting notes
- **Binary provenance:** `strings ./skilldozer` contains the exact error strings
  from `main.go` (`"listing modes --path/..."`, `"'check' cannot be combined..."`,
  `"'init' cannot be combined..."`), so the binary matches HEAD source.
- **Shared render duplication:** the `runInit` check-report block (main.go
  1040–1053) is a byte-copy of the `if c.check` block (main.go 566–584). For
  Issue 1 the fix diverges them (init uses stderr, check uses stdout) — do NOT
  refactor them into a shared helper as part of this bugfix (the existing comment
  at lines 1038–1039 says "do not refactor; mirror"). Just swap the writer.
- **No staged files; working tree otherwise clean.**
