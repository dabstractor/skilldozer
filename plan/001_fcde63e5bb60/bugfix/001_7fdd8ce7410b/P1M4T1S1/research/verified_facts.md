# Verified Facts — P1.M4.T1.S1 (Issue 5: combined shorts + `--flag=value`)

Empirically grounded against the live repo at `/home/dustin/projects/skpp` (go1.25)
BEFORE writing the PRP. Every load-bearing decision is tied to a command that ran.

## 0. Baseline (the world this subtask starts in)

```
$ go build ./... && go vet ./...            # both clean
$ go test ./... -v | grep -c '^--- PASS'
217
$ go test . -v | grep -c '^--- PASS'        # main package alone
98
$ cat go.mod
module github.com/dabstractor/skpp
go 1.25
require gopkg.in/yaml.v3 v3.0.1             # the ONLY direct dependency
```

- main.go ALREADY imports `"strings"` (used by the existing default branch
  `strings.HasPrefix(a, "-")`). The normalization uses `strings.HasPrefix`,
  `strings.Contains`, `strings.IndexByte` — ALL in the existing `"strings"` import.
  ⇒ **NO new import, NO go.mod change.** Verified.
- The v1.0 CLI matrix is COMPLETE: config has version/help/path/list/all/file/
  relative/noColor/searchMode/searchQ/check/tags/unknownFlag; run() dispatches
  help→version→unknownFlag→exclusivity→check→path→list→search→all→tags; the
  `--search`/`-s` value-taking flag and the `check` subcommand are wired.

## 1. The root cause (issue_analysis.md §Issue 5; confirmed in main.go)

`parseArgs` (main.go ~line 145) is an index loop whose body is a `switch a`
(~line 152) matching EXACT whole tokens. `-vh` ≠ `case "-v"`; `--version=x` ≠
`case "--version"` → both fall to the `default` branch → `strings.HasPrefix(a,"-")`
true → `c.unknownFlag = a` → run() prints `skpp: unknown flag '...'` + exit 2
(main.go ~line 249). So today:

```
./skpp -vh            # unknown flag '-vh', exit 2   (WANT: version+help)
./skpp --version=x    # unknown flag '--version=x', exit 2  (WANT: version)
```

## 2. The fix surface (decisions.md §D5; confirmed)

Normalize each token at the TOP of the parseArgs loop body, BEFORE `switch a`.
Two new branches, each ending in `continue` so the existing switch keeps handling
only the original exact-token forms:

- **(a) Long `--flag=value`**: `strings.HasPrefix(a,"--") && strings.Contains(a,"=")`
  → split on the FIRST `=` (`strings.IndexByte`): `name=a[:eq]`, `val=a[eq+1:]`.
  Switch on `name`: bool flags ignore `val`; `--search` sets searchMode=true +
  searchQ=val; unknown name → unknownFlag (the WHOLE token `a`).
- **(b) Short bundle `-xyz`**: `len(a) > 2 && a[0]=='-' && a[1]!='-'` → delegate to
  a new `expandShortBundle` helper.

The existing switch is UNCHANGED. It still handles `-v`/`-s` (len-2 shorts),
`--search foo` (separate value), `check`, bare tags, and unknowns (`-x`, `--bogus`).

## 3. The short-flag set (LOCKED by issue_analysis + main.go switch)

Short bool flags: **v h p l a f** (→ --version/--help/--path/--list/--all/--file).
Value-taking short: **s** (→ --search). **`--relative` and `--no-color` are
LONG-ONLY** (no short alias → they can NEVER appear in a bundle). So a bundle's
valid chars are exactly {v,h,p,l,a,f} plus a trailing `s`+query.

## 4. The bundle algorithm — TWO-PHASE commit (the critical design point)

`expandShortBundle(c, a, args, i)` walks `body = a[1:]`:

- **Phase 1 (validate)**: walk left→right. Each char must be a bool flag
  (v/h/p/l/a/f). The FIRST non-bool char must be `s` (record `sIdx`, STOP —
  `body[sIdx+1:]` is the query); any OTHER char is UNKNOWN → set
  `c.unknownFlag = a`, **commit NOTHING**, return.
- **Phase 2 (commit)**: only reached if valid. Set the bool flags in `[0, sIdx)`
  (or the whole body if no `s`).
- **`s` handling** (if sIdx ≥ 0): `remainder = body[sIdx+1:]`.
  - remainder ≠ "" → searchMode=true, searchQ=remainder (e.g. `-sfoo`).
  - remainder == "" && i+1 < len(args) → searchMode=true, searchQ=args[i+1],
    return consumeNext=true (caller does `i++`; e.g. `-ls foo`).
  - remainder == "" && no next arg → searchMode stays FALSE (mirrors the bare
    `-s`-no-value rule in the main switch). The bools before `s` remain set.

### WHY two-phase is mandatory (not optional)

run() precedence is **help → version → unknownFlag** (main.go ~line 232). If a
bundle like `-vz` leaked `version=true` alongside `unknownFlag="-vz"`, run() would
hit the `version` branch FIRST and print the version (exit 0), MASKING the
unknown-char error. So a bundle with ANY unknown char must be rejected WHOLESALE:
no partial bool flags committed. The two-phase (validate-then-commit) structure
guarantees this. This is the single most important correctness property of the fix.
Traces (verified by hand against the helper below):
- `-vz` → Phase1 hits 'z' (unknown) → unknownFlag="-vz", version NOT set → run()
  exit 2. ✓ (NOT exit-0 version).
- `-vh` → valid → version=true, help=true → run() help-wins → usage stdout exit 0. ✓
- `-sfoo` → sIdx=0, remainder="foo" → searchMode=true, searchQ="foo". ✓
- `-ls foo` → sIdx=1, remainder="" → consumeNext, searchQ="foo"; list=true. ✓
- `-lsfoo` → sIdx=1, remainder="foo" → searchMode=true, searchQ="foo"; list=true. ✓
- `-sv` → sIdx=0, remainder="v" → searchQ="v" (the 'v' is the QUERY, not a flag). ✓
- `-vs` (no next arg) → version=true, searchMode=false (s had no value). ✓

## 5. No regressions to existing tests (traced)

Existing unknown-handling is UNCHANGED because the new branches only catch NEW shapes:
- `--bogus` (no `=`): (a) skips, (b) skips (a[1]=='-'), switch default → unknownFlag. ✓
- `-x`/`-z` (len 2): (b) skips (`len(a)>2` false), switch default → unknownFlag. ✓
- `--frobnicate`, `--more`: no `=`, not single-dash → switch default. ✓
- `--search` (no `=`, len 9): (a) skips, (b) skips, switch `case "--search","-s":`. ✓
- `--search` as last token (no value): switch case `if i+1<len(args)` false →
  searchMode false. ✓ (TestParseArgsSearchNoValueStaysInactive still passes.)
- `check`, bare tags: (a)/(b) skip, switch `case "check":` / default tag capture. ✓

`TestParseArgsUnknownFlagCaptured`, `TestParseArgsShortUnknownCaptured`,
`TestRunUnknownShortFlagExit2`, `TestRunDefaultUnknownFlag`,
`TestParseArgsFirstUnknownWins`, `TestRunVersionBeatsUnknownFlag`,
`TestRunHelpBeatsUnknownFlag` — ALL still pass (their tokens are not `=`-long or
len>2 bundles).

## 6. Edge cases resolved (decisions, documented)

- `--search=` (empty value): (a) name="--search", val="" → searchMode=true,
  searchQ="". (Matches the existing empty-query behavior — search.Search("",
  skills) matches all.) Documented.
- `--bogus=x` (unknown long with `=`): (a) name="--bogus" → default →
  unknownFlag="--bogus=x" (the whole token, consistent with "report what was typed").
- `-=x` / `-1` weird tokens: `-=x` len 3 → (b) bundle, body="=x", Phase1 '=' unknown
  → unknownFlag="-=x". `-1` len 2 → switch default → unknownFlag. Both rejected.
- `-v=x` (short with `=`): len 4 → (b) bundle, body="v=x", Phase1 'v' ok then '='
  unknown → unknownFlag="-v=x". (Short flags bundle, they don't take `=`; rejected.)
- `--` alone: not `=`, not single-dash bundle → switch default → unknownFlag. (Pre-
  existing; out of scope — skpp does not implement the `--` end-of-options separator.)

## 7. go.mod / go.sum UNCHANGED

No new import (strings already imported). `go mod tidy` is a no-op; verify
`git diff --quiet go.mod go.sum`.

## 8. Test convention (mirrors main_test.go — confirmed)

`package main` (white-box — can call unexported `expandShortBundle` indirectly via
`parseArgs`), plain `t.Errorf`/`t.Fatalf`, table-driven where natural, NO testify,
NO `t.Parallel()`. All new tests go through `parseArgs([...])` (or `run([...])` for
the 2 smoke tests), reusing the existing `sampleStore`/`writeSkillTree` helpers
where a store is needed. APPEND to end of main_test.go; do NOT modify existing tests.

## 9. Parallel-safety (vs P1.M3.T1.S1, in flight)

P1.M3.T1.S1 edits ONLY `internal/ui/ui.go` + `internal/ui/ui_test.go` (unicode
display width). This subtask edits ONLY `main.go` + `main_test.go`. **ZERO file
overlap.** No merge conflict possible regardless of landing order. (P1.M2.T1.S1,
Issue 4 search-fields, is already COMPLETE per plan_status.)

## 10. Test plan (13 new test FUNCTIONS → main ~120, module ~239 PASS lines)

Two are table-driven (with t.Run subtests), so the `--- PASS` line count is ~22
(13 funcs, but the two tables add 4 and 5 subtests + their parents):
parseArgs (11 funcs): ShortBundles table (vh/af/pl/fl = 4 subtests),
  LongEqualsBoolFlags table (version/path/no-color/relative/help = 5 subtests),
  LongEqualsSearch, LongEqualsSearchEmpty, LongEqualsUnknown, ShortAttachedSearch
  (-sfoo), ShortBundleSearchNextArg (-ls foo), ShortBundleSearchAttached (-lsfoo),
  ShortBundleUnknownCharRejectsWhole (-vz, version NOT set — the two-phase proof),
  ShortBundleSearchNoValue (-vs, searchMode false), ShortBundleSConsumesRestAsQuery
  (-sv, query "v").
run() smoke (2 funcs): RunLongEqualsVersion (--version=1.2.3 → version line, exit 0),
  RunShortBundleUnknownExit2 (-vz → exit 2, proves wholesale reject end-to-end).

GATE NOTE: assert the 13 function NAMES exist (grep), not a brittle exact PASS total —
subtest reporting can vary. `go test ./...` green is the real gate.
