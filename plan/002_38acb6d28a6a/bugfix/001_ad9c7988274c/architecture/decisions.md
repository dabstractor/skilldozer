# Decisions log — bugfix round 001_ad9c7988274c

Assumptions made by the architect in lieu of asking. Override if you disagree.

## D1 — Issue 1 fix scope: swap the writer, do not refactor the duplicated block
The `runInit` check-report render (`main.go:1037-1053`) is a byte-copy of the `if c.check`
block (`main.go:566-584`). The existing comment says "do not refactor; mirror." Issue 1's fix
DIVEROES them (init renders to stderr, check stays on stdout). Decision: just swap the three
`stdout`→`stderr` writer args in runInit. Do NOT extract a shared helper — it would change
the stream for the `check` subcommand too and is out of scope for a bugfix.

## D2 — Issue 2 signal: new config field, not reusing unknownFlag
`init --store` with no value must exit 2 WITHOUT writing the config, but bare `init` (no
`--store`) legitimately has `c.initStore==""` and must prompt. Decision: add a dedicated
field (e.g. `c.storeMissingValue bool`) set in BOTH parseArgs `--store` branches (the
next-token form AND the `--store=` '='-form). Check it in `run()` after the unknown-flag
guard, before the exclusivity/init dispatch. Do NOT overload `unknownFlag` (that channel is
"first unknown dashed token"; `--store` is a KNOWN flag missing its value — different error
class, and the message must be `--store requires a value`, not `unknown flag`).

## D3 — Issue 4: append the second reserved token to c.tags (Option A)
`init init` must exit 2. Option A (capture the duplicate reserved `init`/`check` token into
`c.tags` so the init exclusivity branch already rejects it) needs no new field and reuses the
existing `'init' cannot be combined with tag arguments` path. Chosen over Option B (a
counter field) for minimal config-struct churn. A literal store dir named `init`/`check`
must still use `--store` — already documented in the `init` case comment.

## D4 — Issue 5: expandHome is a resolveStore-local helper in main.go
Placement decision (deferred by the researcher). Decision: keep `expandHome` in `main.go`
adjacent to `resolveStore`, NOT a new `internal/paths` package. Rationale: the helper has
exactly one caller (resolveStore); main.go already imports `os`/`strings`/`filepath`; a new
package for one 15-line function is over-engineering. If a second caller appears later, lift
it then. Reuse `os.UserHomeDir` (consistent with `internal/config/config.go:154`).

## D5 — Issue 6: trim to §16 exactly, including removing comments and .pi-subagents/
Per prior-round §D3, do NOT bless extras. The `.pi-subagents/` dir becomes untracked
(surfaces in git status) — that is intended; if the user wants it ignored they update §16
themselves. The §16 canonical block has NO section comments; the rewrite omits them.

## D6 — Issue 7: documentation-only (decision record), NOT a code change, NOT a PRD.md edit
`check`/`init` reservation is deliberate and documented in code. A code change to resolve a
skill named `check` would silently shadow the `skilldozer check` subcommand — a worse UX
surprise. Decision: record the reservation + workarounds in this decisions.md (D6) and surface
it in the README's skill-tag section. PRD.md is human-owned/READ-ONLY — the architect does NOT
edit it. The §7.2 note suggested by the PRD is a human action item recorded here, not executed.

## D7 — Documentation sync: Mode A per-subtask + one final Mode B sweep
Per the architect SOW §5: each implementing subtask updates the doc it directly touches
(Mode A — e.g. the `.gitignore` subtask touches nothing doc-wise but the code-comment
subtasks update inline comments). A final "Sync changeset-level documentation" task sweeps
README.md and any overview doc spanning the whole delta (Mode B), depending on all
implementing subtasks. The README's CLI/exit-code/init sections are the candidates.
