# Bug-Fix Validation — Issues 1–7 (confirmed against built binary ./skilldozer @ HEAD 414758d)

All repros confirmed by scout subagents running the PRD's exact reproduction steps.
Each entry: confirmed behavior → exact file:line → precise fix → related tests.

---

## ISSUE 1 (Major) — `init` writes the check report to stdout (should be stderr)

**Confirmed repro** (`init --store /tmp/A/store </dev/null 2>/dev/null`):
stdout = 3 lines: the store path + `OK    example (example)` + `1 skills, 0 errors, 0 warnings`.

**Site:** `main.go` `runInit` (`main.go:988`), check-report block **lines 1037–1053**:
- `main.go:1046` `fmt.Fprintf(stdout, "%-5s %s (%s)\n", "OK", ...)` — OK line
- `main.go:1050` `fmt.Fprintf(stdout, "%-5s %s (%s): %s\n", f.Level, ...)` — per-finding
- `main.go:1053` `fmt.Fprintf(stdout, "%d skills, %d errors, %d warnings\n", ...)` — summary
- `main.go:1026` `fmt.Fprintln(stdout, dir)` — store-path headline; **STAYS on stdout** (§6.1)
- `main.go:1029` `fmt.Fprintf(stderr, "(found via %s)\n", src)` — already stderr ✓

**Fix:** in runInit, change the three check-report `fmt.Fprintf` calls (1046/1050/1053)
from `stdout` → `stderr`. Do NOT touch line 1026 (store path stays on stdout). Do NOT
refactor the duplicated `if c.check` block (existing comment: "do not refactor; mirror") —
they diverge now (init→stderr, check→stdout). Exit codes unchanged (init stays 0).

**Tests:** strengthen `TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0`
(`main_test.go:2325`) — it only asserts `Contains(out, store)` which passes despite the
bug. Add: stdout must be EXACTLY one line (the path), must NOT contain "skills,"/"OK";
those lines must appear in `errOut`.

---

## ISSUE 2 (Major) — `init --store` with no value silently overwrites config

**Confirmed repro** (`init --store </dev/null`, existing config present): exit 0, the
config's pre-existing `store:` (and any other keys) is overwritten with the auto-detected
path. The dangerous shape is `init --store` (the `init` token set `c.init=true`; trailing
`--store` sets nothing). Bare `skilldozer --store` (no init token) falls through to
no-args usage, exit 1, config untouched (harmless).

**Site (parseArgs):**
- `main.go:257–266` `case "--store":` — the `if i+1 < len(args)` guard at **line 263**
  silently no-ops when `--store` is the last token. Comment (258–262) documents this as
  "deferred/intentional" — unsound when `init` already set `c.init=true`.
- `main.go:192–197` `--store=` '='-form — empty value likewise sets `c.init=true; c.initStore=""`.
- `main.go:447` `if c.init { return runInit(...) }` — dispatches with no missing-value guard.

**Fix:** record a missing-value signal when `--store` (and `--store=`) is seen with no
value, and reject in `run()` with exit 2 BEFORE dispatching init. CRITICAL: `c.initStore==""`
alone is insufficient — bare `skilldozer init` legitimately has `c.initStore==""` and MUST
prompt. The distinguishing signal is that `--store` actually appeared in argv (track a new
`c.storeSeen`/`c.storeMissingValue` field set in BOTH parseArgs `--store` branches). In
`run()`, after the unknown-flag check but before the exclusivity/init dispatch: if the
missing-value signal is set → `skilldozer: --store requires a value` to stderr, exit 2
(config must NOT be written). This mirrors §6 header "Unknown flags ⇒ error + exit 2" and
the delta-PRD "init is non-destructive" constraint.

**Tests:** no test covers `init --store` (trailing, no value) or `--store=` (empty value).
Add parse-level test asserting the missing-value signal is set, and a `run()`-level test
asserting exit 2 + empty stdout + config NOT written (mirror `TestRunExclusivityInitAndCheck`).

---

## ISSUE 3 (Minor) — `tag + --path` not rejected (silent tag drop)

**Confirmed repro** (`NONEXISTENTTAG --path`): exit 0, `--path` wins, stray tag silently
dropped. Control: `NONEXISTENTTAG --list` correctly → exit 2.

**Site:** `exclusivityError` (`main.go:686`), predicate at **line 702**:
```go
if hasTags && (c.list || c.searchMode || c.all) {   // c.path MISSING
```
Asymmetry: `c.path` IS in the count set (`main.go:695`) and the check+mode set
(`main.go:708`), but omitted only from the tags predicate at 702.

**Fix:** add `c.path` to the predicate and update the message:
```go
if hasTags && (c.path || c.list || c.searchMode || c.all) {
    return true, "skilldozer: tags cannot be combined with --path/--list/--search/--all"
}
```

**Tests:** gap — no `TestRunExclusivityTagsAndPath`. Add it (mirror
`TestRunExclusivityTagsAndList` @1559): `run([]string{"foo","--path"})` → code 2, empty
stdout, stderr contains "cannot be combined". Also add a `{tags:["foo"], path:true}` case
to `TestExclusivityErrorListingModes` (@1970).

---

## ISSUE 4 (Minor) — `init init` runs init (exit 0) instead of erroring

**Confirmed repro** (`init init </dev/null`): exit 0, config written. Control:
`init check </dev/null` → exit 2 (correctly rejected). The `init` case's GOTCHA guard
swallows a following `init`/`check` as a non-store token, but a following `init`
re-enters `case "init":` and re-sets `c.init=true` with no conflict.

**Site:** `parseArgs` `case "init":` (`main.go:268–284`):
- `main.go:277` `c.init = true` (unconditional)
- `main.go:281` guard `next != "check" && next != "init"` — refuses to capture, so the
  second `init` falls through to its own `case "init":` re-entry.
- `exclusivityError` init branch (`main.go:711–716`) rejects check/list/search/all/path
  but NOT a duplicate init.

**Fix (preferred Option A):** when the token after `init` is `"init"` (or `"check"` —
check already works via `c.check`), treat it as a conflict so exclusivity fires. Cleanest:
append the second reserved `init`/`check` token to `c.tags` (the init branch at
`main.go:713` already rejects `hasTags` → exit 2 with "'init' cannot be combined with tag
arguments"). Change the guard so `next == "init"` is captured into `c.tags` rather than
swallowed. (A literal store dir named `init` must still use `--store` — already documented.)
Alternative Option B: a `c.initCount` counter rejected in exclusivity when >1. Option A
needs no new field.

**Tests:** gap — no test for `init init`. Add `TestParseArgsInitInitDoesNotSwallow` and a
run-level `TestRunExclusivityInitInit` asserting exit 2 + empty stdout + config NOT written
(mirror `TestRunExclusivityInitAndCheck` @1679).

---

## ISSUE 5 (Minor) — Tilde (`~`) not expanded in init interactive prompt

**Confirmed behavior:** `resolveStore` absolutizes via `filepath.Abs(store)` (`main.go:886`),
which does NOT expand `~`. So `~/myskills` → `<cwd>/~/myskills` and a dir literally named
`~` is created. Affects BOTH interactive (typed path) and non-interactive (`--store ~/x`)
paths since both flow through `resolveStore`.

**Site:** `main.go:865` `resolveStore`, line **886** `abs, err := filepath.Abs(store)`.
Call site: `main.go:990-994` `runInit` → `resolveStore(c.initStore)`. Pure seam
`chooseStore` (`main.go:822`) returns choice verbatim by design — do NOT widen its signature.

**Fix:** add an `expandHome`/`expandTilde` helper (stdlib-only: `os.UserHomeDir` +
`strings.HasPrefix(p, "~/")`), call it in `resolveStore` BEFORE `filepath.Abs`. Reuse the
`os.UserHomeDir` pattern already in `internal/config/config.go:154` (DefaultStore). See
`go_tilde_expansion.md` for the exact helper + edge-case table. Helper placement:
`resolveStore`-local function in main.go (keeps the change self-contained; main.go already
imports `os`/`strings`/`filepath`).

**Tests:** add a table-driven `TestExpandHome` (HOME set/unset) + a `resolveStore`/init
integration test: `t.Setenv("HOME", tmp)`, `run([]string{"init","--store","~/sub"})`,
assert `configpkg.Load(cfg).Store == filepath.Join(tmp,"sub")` (NOT `<cwd>/~/sub`).

---

## ISSUE 6 (Minor) — `.gitignore` has extra entries beyond PRD §16 spec

**Confirmed:** §16 (`PRD.md:423-431`) spec is EXACTLY 5 entries (no comments):
```
/skilldozer
/dist
*.test
*.out
.DS_Store
```
Current `.gitignore` has 5 EXTRA entries + 5 section-comment lines:
`/build`, `node_modules/`, `venv/`, `.env`, `.pi-subagents/`.

**Fix:** rewrite `.gitignore` to the exact 5-line §16 block (no comments), byte-for-byte.
Removing `.pi-subagents/` makes the agent-artifacts dir untracked (surfaces in git status)
— intended per prior-round §D3 ("do not bless extras"); residual risk noted.

**Verification:** `diff <(sed -n '426,430p' PRD.md) .gitignore` → no output. No Go test
(.gitignore is not code).

---

## ISSUE 7 (Minor) — skill tagged `check`/`init` unresolvable by canonical tag

**Confirmed:** `parseArgs` `case "check"` (`main.go:247`) and `case "init"` (`main.go:268`)
are reserved subcommand tokens, matched BEFORE the default tag-capture branch
(`main.go:281`). A skill at `skills/check/SKILL.md` is shadowed — unresolvable by its
canonical tag, though still discoverable by `--list`/`--all`/nested path/`name`/alias.

**Approach: DOCUMENTATION-ONLY (no code change).** The code documents the reservation as
deliberate (`main.go:248-256`, `269-280`: "subcommand names are reserved, as in any CLI").
A code change to resolve a skill named `check` would silently shadow the `skilldozer check`
subcommand — a worse UX surprise. Prior-round §D7 set the "no code change; documentation-only"
precedent.

**Fix:** PRD.md is human-owned/READ-ONLY (architect must NOT edit it). Record the decision
in `architecture/decisions.md` §D7-style: `check`/`init` are reserved; document the
workarounds (nested path `writing/check`, frontmatter `name`, alias, `--list`). This task
is a decision-record write + a README/decisions cross-reference, NOT a PRD.md edit and NOT
a code change.

**Tests:** existing `TestParseArgsCheckSubcommand` (@1120) documents the reservation (asserts
`check` → `c.check=true`, `len(tags)==0`). No new code test needed; the decision record is
the deliverable.

---

## Fix ordering & dependency notes

- Issues 1, 3, 4, 5, 6, 7 are independent leaf fixes (different functions/files).
- Issue 2 touches BOTH `parseArgs` (new config field) and `run()` (guard) — keep the
  parse-level and run-level changes in the SAME subtask (atomic).
- Issue 1 and Issue 5 both touch `runInit`/`resolveStore` but different lines; safe to
  sequence but not strictly dependent.
- All fixes follow implicit TDD: write the failing test first, then implement, then pass.
