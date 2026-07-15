# Architectural Decisions — Skilldozer Bugfix Round 2

## D1: Issue 2 — findConfig Return Shape (4-value with vanishedStore)

**Decision:** Change `findConfig()` from `(dir string, src Source, found bool)` to
`(dir string, src Source, found bool, vanishedStore string)`.

**Alternatives Considered:**
1. **Side-channel sentinel** (package var set by findConfig, read by Find):
   Rejected — hidden mutable state, not thread-safe, fragile.
2. **Re-query config in Find()**: Rejected — duplicates the config.Load +
   store-resolution logic, wasteful, fragile (two code paths that must agree).
3. **Change findConfig to return error**: Rejected — breaks the "locked per-rule
   shape" contract that ALL findXxx helpers return `(dir, src, found bool)` and
   never error. An error return would require Find() to interpret errors from
   findConfig differently from the other findXxx helpers.
4. **4-value return with vanishedStore string**: ACCEPTED. Minimal disruption:
   the first three values are unchanged for all existing miss paths. Only the
   line-126 branch (store dir absent) populates vanishedStore. The 6 direct
   findConfig tests need a 4th-value `, _` append; only
   TestFindConfigStoreDirAbsent needs a semantic assertion change.

**Rationale:** The vanishedStore string is the resolved store path (already
computed by findConfig before the os.Stat check). Find() uses it in the error
message so the user sees exactly which configured path is missing.

## D2: Issue 2 — Sentinel Error Pattern

**Decision:** Define `var ErrConfiguredStoreMissing = errors.New("configured
skills store directory does not exist")` and wrap it via
`fmt.Errorf("%w: ...", ErrConfiguredStoreMissing, ...)` in Find().

**Rationale:** Tests use `errors.Is(err, ErrConfiguredStoreMissing)` to verify
the error type without matching the full message string (which includes the
dynamic store path). This follows the existing `ErrNotFound` pattern.

## D3: Issue 2 — Error Does NOT Fire When Env Overrides

**Decision:** The env rule (priority 1) is checked BEFORE findConfig (priority 2).
If `SKILLDOZER_SKILLS_DIR` is set and valid, it wins immediately and findConfig
is never called. The vanished-store error only fires when env is unset/invalid
AND config is present with a vanished store.

**Rationale:** This preserves the existing priority order (env > config > sibling
> walkup) without change. The fix only removes the silent fall-through AFTER the
config rule — once we're at config priority and the configured store is gone, we
stop rather than silently picking up a different store.

## D4: Issue 3 — Symmetrical Missing-Value Handling (exit 2 for all)

**Decision:** Make `--search`/`-s` and `--shell` no-value exit 2 with an error
message, mirroring `--store`.

**Alternatives Considered:**
1. Document the asymmetry + fix the misleading comment: Rejected — the PRD says
   "The symmetrical option is more predictable for `$(...)` use." Asymmetry
   means `$(skilldozer --search)` captures help text, which is surprising.
2. Symmetrical exit 2: ACCEPTED. Follows the existing `storeMissingValue`
   pattern exactly (bool field in config struct, checked in run() before
   exclusivity dispatch).

**Rationale:** `$(skilldozer --search)` should fail loudly (empty stdout, exit 2)
not silently capture the help text.

## D5: Issue 3 — `=`-form Empty Values Left As-Is

**Decision:** `--search=` (empty value via `=`) keeps its current behavior
(`searchMode=true, searchQ=""` → searches with empty query, matching everything).
Only the bare `--search` (no following token at all) gets the new exit-2
treatment.

**Rationale:** The PRD specifically describes the space-separated no-value case.
The `=`-form with an empty value is a distinct syntactic form that the PRD does
not mention. Changing it would be scope creep.

## D6: Issue 4 — `--` Implementation via Loop Flag

**Decision:** Use a boolean `endOfOpts` flag in the parseArgs loop rather than
early-return or slice splitting.

**Alternatives Considered:**
1. Early return: `if a == "--" { collect rest as tags; return c }`: Works but
   breaks the uniform loop structure and skips any post-loop logic (currently
   none, but fragile for future additions).
2. Loop flag: ACCEPTED. Placed before ALL token classification (before the `=`
   check, before the short-bundle check, before the switch). Clean, extensible,
   and keeps the loop structure intact.

## D7: Issue 1 — `--shell` Added to Flag Advertisement List

**Decision:** Add `--shell` to the advertised flag list in all three completion
files (not just the value routing).

**Rationale:** `--shell` is a real, documented flag (in `usageText` OPTIONS).
Its omission from the completion flag list is inconsistent. The PRD notes a
tension with §14.6's 13-flag table, but since §14.2 requires `--shell` value
completion, having the flag discoverable is the consistent choice. The §14.6
table can be updated separately if needed.

## D8: Documentation — Mode A per-subtask, Mode B final sweep

**Decision:** Each implementing subtask updates the doc it directly touches
(Mode A). A final "Sync changeset-level documentation" task sweeps README.md
and help text for cross-cutting consistency (Mode B).

**Rationale:** The completion-file header comments (Issue 1), the misleading
parseArgs comment (Issue 3), and the README version line (Issue 5) are all
per-file doc updates that ride with the implementing subtask. The README's
overall feature/coexistence story may need a consistency check after all issues
land, which is the final task's job.
