# Verified Facts — P1.M1.T1.S1 (--link multi-target batch conversion)

Confirmed against the live source (`main.go`, `main_test.go`) + plan/006
`codebase_state.md`. Implements PRD §8.4 batch linking. Single subtask touches
struct + parser + exclusivity + runLink + tests + usageText + README.

## 0. Git-log vs live-state discrepancy (resolved)

`git log` shows `983672e Support batch linking in --link`, but the LIVE code is
STILL single-target (verified): `linkTarget string` (singular) at main.go:177,
`c.linkTarget = args[i+1]` (assignment, not append) at :426, and runLink processes
one target (main.go:1424-1494). So this PRP genuinely drives the single→batch
conversion. Trust the live code + codebase_state.md (both agree: single-target).

## 1. The COMPLETE set of `linkTarget` references (all must rename to `linkTargets`)

**main.go** (8 sites):
- `:177` field decl `linkTarget string` → `linkTargets []string` (+ add `collectingLink bool`)
- `:271` `c.linkTarget = val` (=-form) → `append(c.linkTargets, val)`
- `:426` `c.linkTarget = args[i+1]` (next-token) → `append(c.linkTargets, args[i+1])`
- `:943` comment "by parseArgs as c.linkTarget" → update wording
- `:1417` comment "c.linkTarget is guaranteed non-empty" → "len(c.linkTargets) > 0"
- `:1437` `expandHome(c.linkTarget)` → inside loop, `expandHome(t)` (loop var)
- `:1440` error msg `... %q: ... c.linkTarget` → use the loop var / original token
- `:1446` error msg `... %q is not an existing directory ... c.linkTarget` → loop var

**main_test.go** (the parseArgs-level assertions; runLink tests use run() and don't touch the field):
- `:3496-3497` TestParseArgsLinkNextToken `c.linkTarget != "/path/to/skill"` → `c.linkTargets[0]`
- `:3513-3514` TestParseArgsLinkEquals `c.linkTarget != "/path/to/skill"` → `c.linkTargets[0]`
- `:3540-3541` TestParseArgsLinkEqualsEmpty `c.linkTarget != ""` → `len(c.linkTargets) == 0`
- `:3876-3877` TestParseArgsLinkDashedFollowerNotConsumed `c.linkTarget != ""` → `len(c.linkTargets) == 0`

## 2. TWO positional-append sites — `collectingLink` must guard BOTH

The default-position handler is NOT the only place tags are appended:
- **main.go:442** — the `default:` case `c.tags = append(c.tags, a)` (the one the contract names).
- **main.go:207** — the POSIX `--` endOfOpts handler: `if endOfOpts { c.tags = append(c.tags, a); continue }`.

For correctness (`--link a -- b` should route `b` to linkTargets, not tags), route BOTH
through collectingLink: `if c.collectingLink { c.linkTargets = append(c.linkTargets, a) } else { c.tags = append(c.tags, a) }`.
Missing the :207 site is a latent bug for the `--` separator edge case.

## 3. DESIGN DECISION — `c.link` semantics (deviates from contract (b) literal; preserves tests)

The contract (b) says "set c.link = true and c.collectingLink = true" UNCONDITIONALLY when
`--link` is seen. But the live next-token form sets `c.link = true` ONLY when a value is
consumed (the else/no-value branch leaves c.link false), and THREE existing tests assert
`c.link = false` on no-value/dashed-follower paths:
- TestParseArgsLinkNoValue (:3523): `if c.link { want false }`
- TestParseArgsLinkDashedFollowerNotConsumed (:3868): `if c.link { want false }` (+ doc comment "c.link stays false")
- TestParseArgsLinkDashedModifierFollowerIsMissingValue (:3889): `if c.link { want false }`

Setting c.link=true unconditionally would flip all three AND contradict contract (d)
("the dashed-follower guard must be preserved: `--link --check` still exits 2").

**Chosen design (Option B — minimal churn + semantically correct):**
- **next-token `--link`**: set `c.link = true` + `c.collectingLink = true` ONLY when a
  non-dashed follower is consumed (append it); else `linkMissingValue = true` (c.link and
  collectingLink stay false). Preserves all 3 c.link=false tests.
- **=-form `--link=`**: KEEP `c.link = true` unconditionally (the live =-form already does,
  and TestParseArgsLinkEqualsEmpty expects c.link=true); set `collectingLink = true`; append
  val if non-empty, else `linkMissingValue = true`.

This asymmetry (=-form c.link unconditional; next-token c.link conditional) ALREADY EXISTS in
the live code; preserving it is the minimal-churn path and matches contract (i)'s explicit
test-update list (only NextToken, Equals, WithTagsExits2). All batch requirements still hold:
`--link a b c`→[a,b,c]; `--link=a b`→[a,b]; `--link`/`--link=`→exit 2; `--link --check`→exit 2.

## 4. exclusivityError needs NO code change (collectingLink routing handles contract (e))

The `--link` block (main.go:941-953) has `if hasTags { exit 2 "cannot be combined with tag arguments" }`.
Contract (e) says "REMOVE the tag-arguments error for post-link positionals." But once
collectingLink routes post-link positionals to linkTargets (not tags), `c.tags` stays empty
for normal multi-link usage, so `hasTags` is naturally false post-link — the check only fires
for PRE-link tags (`sometag --link a`), which IS a real conflict. So contract (e) is satisfied
BY THE PARSEARGS ROUTING, with zero exclusivityError code change. Keep the hasTags check +
message as-is. (Verified: `--link /tmp/foo sometag` → linkTargets=[/tmp/foo,sometag], tags=[],
no exclusivity error → runLink validates both → exit 1.)

## 5. The missing-value guard — message change + defensive broaden (contract (f))

main.go:608-610 currently:
```go
if c.linkMissingValue {
    fmt.Fprintln(stderr, "skilldozer: --link requires a path to a skill directory")
    return 2
}
```
- Change message to the §6.4 exact text: `skilldozer: --link requires at least one path to a skill directory`.
- Broaden to also catch `c.link && len(c.linkTargets) == 0` (defensive; with Option B this
  only arises from the =-form `--link=` which already sets linkMissingValue, but the belt-
  -and-suspenders guard is cheap and matches contract (f)'s "handle c.link true but 0 targets"):
  `if c.linkMissingValue || (c.link && len(c.linkTargets) == 0) { … exit 2 }`.

**Test impact the table MISSED:** TestRunLinkNoValueExits2 (:3549) asserts the EXACT stderr
`"skilldozer: --link requires a path to a skill directory\n"` — the message change breaks it.
Update to `"… requires at least one path to a skill directory\n"`. (TestRunLinkEqualsEmptyExits2
:3564 does NOT assert the message text — only code+stdout — so it stays Same.)

## 6. runLink — single-target body (main.go:1424-1494) → loop

Current body order: (1) skillsdir.Find ONCE → (2) expandHome+Abs → (3a) os.Stat isDir →
(3b) not-store-inside (Clean + HasPrefix) → (3c) HasSkillMD → (4) Base+Join → (5) conflict
(Lstat: non-symlink refuse / symlink remove) → (6) os.Symlink → (7) stdout=linkPath,
stderr="Linked X -> Y (found via Z)".

Batch rewrite: keep (1) ONCE before the loop; wrap (2)-(7) in `for _, t := range c.linkTargets`.
Use a local `orig := t` for error messages (the original token, pre-Abs). Track `hadErr bool`;
on any per-target failure print the stderr line and continue (do NOT return early — partial
success is the contract). After the loop: `if hadErr { return 1 }; return 0`. stdout gets one
linkPath per success, in input order (the contract's order-preservation requirement).

Helpers confirmed present: `expandHome(p string) string` (main.go:1089),
`skillsdir.HasSkillMD(dir string) bool` (skillsdir.go:213), `skillsdir.Find()` (§8.3).

## 7. usageText — 3 sites (main.go:82, 98, 111)

- `:82` USAGE: `skilldozer --link <dir>` → `skilldozer --link <dir> [<dir>...]`
- `:98` EXAMPLES: add a multi-link line (e.g. `skilldozer --link ~/p/a ~/p/b ~/p/c`)
- `:111` OPTIONS: `--link <dir>` → `--link <dir> [<dir>...]` + describe batch.

## 8. The COMPLETE test-update list (6 existing tests; codebase_state.md table missed #5 and under-specified #4)

| # | Test (main_test.go) | Change |
|---|---|---|
| 1 | TestParseArgsLinkNextToken (:3491) | `c.linkTarget` → `c.linkTargets[0]` (+ assert `len==1`) |
| 2 | TestParseArgsLinkEquals (:3508) | `c.linkTarget` → `c.linkTargets[0]` |
| 3 | TestParseArgsLinkEqualsEmpty (:3535) | `c.linkTarget != ""` → `len(c.linkTargets)==0` (c.link stays true) |
| 4 | TestParseArgsLinkDashedFollowerNotConsumed (:3868) | `c.linkTarget != ""` → `len(c.linkTargets)==0` (c.link stays false — Option B) |
| 5 | TestRunLinkNoValueExits2 (:3549) | stderr message → "at least one path to a skill directory" (TABLE MISSED — exact-string assertion) |
| 6 | TestRunLinkWithTagsExits2 (:3576) | exit 2 → exit 1; `/tmp/foo`+`sometag` both fail validation (not dirs); rename suggested → TestRunLinkTrailingPositionalBecomesTargetExits1 |

Tests that stay GREEN unchanged (Option B preserves their c.link=false): TestParseArgsLinkNoValue
(:3523), TestParseArgsLinkDashedModifierFollowerIsMissingValue (:3889), TestRunLinkWithModeExits2
(:3588, `--link /tmp/foo --check` → exclusivity exit 2), TestRunLinkSuccess/Refresh/Refuse*
(n=1 batch), TestRunLinkDashedFollowerExits2NoMutation (:3946).

## 9. New tests to add (contract (j))

- TestRunLinkMultiSuccess: 2 valid dirs → both linked, stdout has both paths in order, exit 0.
- TestRunLinkMixedBatch: 2 valid + 1 invalid → valid linked, invalid on stderr, exit 1, successes persist.
- TestRunLinkSingleBadDir: 1 invalid → nothing on stdout, exit 1.
- TestParseArgsLinkCollectsMultiple: `--link a b c` → linkTargets=[a,b,c], tags empty.
- TestParseArgsLinkEqualsPlusPositionals: `--link=a b` → linkTargets=[a,b].
- TestRunLinkOrderPreservation: 3 dirs → stdout paths in input order.

Follow the existing link-test fixture style (t.TempDir store via SKILLDOZER_SKILLS_DIR,
write a SKILL.md in each src dir, assert on out.String()/errOut.String()/code).

## 10. Scope boundary & deps

- This subtask touches main.go + main_test.go + README.md (Mode A, §8.4 section ~:158 +
  quick-start ~:127). Does NOT touch the completion scripts (bash/zsh/fish multi-link dir
  completion is P1.M2.T1.S1 — a separate subtask).
- No new deps. `gopkg.in/yaml.v3` (sole dep) unaffected; all helpers (expandHome, HasSkillMD,
  filepath.*, os.*) already imported. go.mod/go.sum byte-for-byte identical.
