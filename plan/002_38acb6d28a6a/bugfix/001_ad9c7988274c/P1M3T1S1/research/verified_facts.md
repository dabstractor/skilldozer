# Verified Facts — P1.M3.T1.S1 (Mode B README sweep for bugfix changeset 001)

Single deliverable: reconcile `README.md` with the bugfix changeset (Issues 1-7).
All facts read directly from source at PRP-write time. Repo: `/home/dustin/projects/skilldozer`.

## §0 — What this task IS (Mode B doc sweep) and is NOT

- **IS**: the one changeset-level documentation sync (SOW §5 Mode B; decisions.md §D7). It reads
  README.md end-to-end and makes the README coherently reflect the WHOLE bugfix delta.
- **is NOT**: a per-feature doc edit (those rode with each implementing subtask — Mode A). It does
  NOT restate the reserved-tag note (P1.M2.T5.S1 owns that). It does NOT edit code or PRD.md.
- **MUST run after** all implementing subtasks (it depends on P1.M1 + P1.M2). At execution time
  Issues 1-6 are landed in main.go and T5.S1's note is landed in README.md.

## §1 — The 4 documentation touchpoints (from contract LOGIC a-d) + consistency check (e)

| # | Issue | README zone | Contract requirement | Current README state | Action |
|---|---|---|---|---|---|
| a | 1 (Major) | "### First run" | "if README describes init's stdout, ensure it says ONLY the store path" | Does NOT describe init's stdout at all (only what init DOES). | **WARRANTED ADD**: a concise note that stdout = the store path (one clean line, for `$(...)`); status + check report → stderr. The payoff of the Issue 1 fix is user-visible only if documented. |
| b | 2 (Major) | "**Error contract.**" | "--store-missing-value exit-2 rule mentioned or implied" | Enumerates exit-2 only for mutually-exclusive modes. | **REQUIRED ADD**: `--store` with no value → exit 2 (config not written). |
| c | 3 (Minor) | "**Error contract.**" | "ensure tags+--path is reflected" | Says modes are mutually exclusive with EACH OTHER; silent on tags+mode. | **REQUIRED ADD**: a `<tag>` + any of --path/--list/--search/--all → exit 2. |
| d | 5 (Minor) | "### First run" | "mention ~ in init paths where appropriate" | No mention of tilde. | **WARRANTED ADD**: `~`/`~/...` expands to $HOME in typed and `--store`/positional paths. |
| e | 7 (Minor) | "Where skills live" | "reserved-tag note must read consistently with error-contract and skill-tag sections" | **T5.S1 ALREADY LANDED the note** (verified present, matches T5.S1 PRP verbatim). | **VERIFY ONLY** — do NOT duplicate. Confirm it is present and consistent; no edit. |

Issues 4 (init init), 6 (.gitignore) have NO direct README touchpoint (covered by general exclusivity
framing / not user-facing). The contract LOGIC does not list them for the README. Decision: no
init-init-specific note (the error-contract's general "mutually exclusive" framing subsumes it).

## §2 — Current README text of the two edit zones (verbatim, for anchor matching)

### Zone A — "### First run" (the init section)
```
It prompts for the directory where skilldozer should keep your skills
(defaulting to `$XDG_DATA_HOME/skilldozer/skills`, or the current directory if
it already looks like a skill store), creates it, seeds an `example/SKILL.md`
template if it is empty, and writes the config pointing at it. For scripts / CI,
skip the prompt:

```bash
skilldozer init /path/to/store      # positional
skilldozer init --store /path/to/store
```
```
- INSERT a new paragraph AFTER the `skilldozer init --store /path/to/store` code block's closing
  fence, BEFORE the `## Shell completions` heading. (Zone A is ABOVE T5.S1's insertion ~line 200, so
  its line numbers are stable; but pin the seam by anchor text: the `init --store /path/to/store`
  line + the following fence + blank line + `## Shell completions`.)

### Zone B — "**Error contract.**" (in "## Usage")
```
**Error contract.** An unknown tag prints **nothing** to stdout and exits 1
(the error goes to stderr only). That is why
`pi --skill "$(skilldozer badtag)"` fails loudly instead of loading nothing. When
multiple tags are given, any unresolved tag causes nothing to be printed and
exit 1, so `pi` never sees a partial result. The listing modes `--path`,
`--list`, `--search`, and `--all` are mutually exclusive — combining any two
exits 2.
```
- REPLACE the final 3 lines ("The listing modes `--path`, ... exits 2.") with expanded text covering
  tags+mode (Issue 3) AND `--store` no-value (Issue 2). Keep the opening sentences (unknown tag →
  stdout empty + exit 1; multi-tag atomicity) UNCHANGED.

## §3 — Landed behavior to document (verified directly in main.go @ HEAD)

- **Issue 1 (runInit, main.go:1057):** check-report OK/WARN/ERROR lines + the `N skills, M errors, K
  warnings` summary now render to **stderr** (3 `fmt.Fprintf(stderr, …)` calls). ONLY `fmt.Fprintln
  (stdout, dir)` (the store path, main.go:~1095) stays on stdout. Seeded/Adopted status + "(found
  via …)" are already stderr. ⇒ on success, `skilldozer init` stdout is EXACTLY one line (the store
  path); on failure, stdout is empty + exit 1. So `STORE="$(skilldozer init --store /p)"` is clean.
- **Issue 2 (parseArgs + run guard, main.go:461):** `--store` / `--store=` with no value sets
  `c.storeMissingValue`; `run()` rejects with `skilldozer: --store requires a value` → stderr,
  **exit 2**, BEFORE init dispatch (config is NOT written). Bare `skilldozer init` (no `--store`)
  still legitimately has `c.initStore==""` and prompts — the signal distinguishes the two.
  **CRITICAL ACCURACY NOTE:** `--search`/`-s` with no value is NOT covered by this fix — it still
  falls through to usage/exit 1 (harmless). Do NOT claim `--search` no-value exits 2 in the README.
- **Issue 3 (exclusivityError, main.go:737):** `if hasTags && (c.path || c.list || c.searchMode ||
  c.all)` → exit 2, message `tags cannot be combined with --path/--list/--search/--all`. So a tag +
  ANY of the four inspection modes now exits 2 (previously only --list/--search/--all did; --path
  silently dropped the tag).
- **Issue 5 (expandHome, main.go:894; wired at main.go:953):** `expandHome` (stdlib
  `os.UserHomeDir` + `strings.HasPrefix(p, "~/")`) runs in `resolveStore` BEFORE `filepath.Abs`.
  A leading `~/` and a bare `~` expand to `$HOME`. Affects BOTH the interactive typed answer AND
  `--store`/positional non-interactive paths (both flow through resolveStore).
- **Issue 4 (parseArgs, main.go:292):** a duplicate reserved `init` token is captured as a conflict
  → exit 2 (message `'init' cannot be combined with tag arguments`). NOT separately documented
  (subsumed by the general exclusivity framing).
- **Issue 6 (.gitignore):** trimmed to the §16 5-entry set. Not README-facing.

## §4 — T5.S1 reserved-tag note (Issue 7) — ALREADY LANDED, verify only

The note IS present in "Where skills live" (read directly), matching T5.S1's PRP recommended text:
> **Reserved tag names.** `check` and `init` are subcommand names, so they never resolve as skill
> tags: `skilldozer check` runs validation and `skilldozer init` runs first-run setup. …

Per contract DOCS step 5 + T5.S1's PRP: "the final Mode B sweep (P1.M3.T1) verifies README
consistency across all 7 fixes but does NOT duplicate this note." So this task VERIFIES the note is
present + consistent with the error-contract (exit-2 framing) and the skill-tag section, and does
NOT rewrite it. The error-contract talks about exit CODES; the reserved-tag note talks about tag
RESOLUTION — they do not conflict (check/init are subcommands, neither tags nor modes).

## §5 — README voice rules (verified; MUST hold after edits)

- `grep -c '§' README.md` == 0 — the user-facing README NEVER cites PRD section numbers. Use plain
  words ("prints the store path", "runs validation"), never "§6.1"/"§8.2". (The contract's "(§8.2,
  §9)" etc. are pointers for the PRP writer, NOT README citations.)
- Callouts use a `**Bold lead-in.**` + prose paragraph (e.g. `**Error contract.**` at the Zone B
  anchor). No blockquotes/tables/subheadings for inline notes.
- Inline `` `code` `` for commands/paths/flags/tags. Em dashes (—) are used elsewhere in the README,
  so voice-consistent. Plain declarative prose; one idea per sentence.
- "Keep edits minimal and in the existing README voice." (contract LOGIC). "If a section needs no
  change, note that explicitly rather than forcing an edit." (contract DOCS).

## §6 — Parallel-execution / file-conflict consideration

T5.S1 (the parallel sibling, "Implementing") edits README.md — it inserts the reserved-tag note at
the END of "Where skills live" (~line 200). This task ALSO edits README.md. BUT:
- This task's two zones (First run ~69-77, Error contract ~164-170) are ABOVE T5.S1's insertion, so
  they are REGION-disjoint and (for Zone A/B) line-stable.
- At THIS task's EXECUTION time, T5.S1 will have completed (P1.M3 depends on P1.M2.T5), so the note
  is landed and stable. No concurrent write.
- ROBUSTNESS: pin ALL edit seams by ANCHOR TEXT (the `init --store /path/to/store` line + fence +
  `## Shell completions`; the "The listing modes … exits 2." sentence), NOT by line number, so the
  edit survives any line shifts from T5.S1 (or any other late landing).
- DO NOT touch the "Where skills live" section (T5.S1's territory) — only VERIFY it.

## §7 — Scope boundary (what NOT to touch)

- `main.go` / `main_test.go` / `internal/*` — UNCHANGED (all fixes landed; this is doc-only).
- `PRD.md`, `tasks.json`, `prd_snapshot.md` — READ-ONLY (human/orchestrator-owned).
- `.gitignore` — already fixed by P1.M2.T4.S1; do not touch.
- The "Where skills live" reserved-tag note — T5.S1 owns it; VERIFY only, do not duplicate.
- The "How `skilldozer` finds the store" section (5-rule priority + config model) — already current
  (updated in the prior non-bugfix plan); verify it is consistent, no changeset-relevant drift.
- Validation: `go build ./... && go vet ./... && go test ./...` must stay green (no code change →
  proof of isolation). Re-build + re-test is the contract's "doc sanity" check.
