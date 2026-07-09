# Delta PRD — Skilldozer (session 003)

> **Scope:** Implement the two PRD changes introduced in commit `bbd4e74` ("Add completion subcommand and flip bare-inv to implicit help"), which touched **PRD.md only** (+105/−5). The code is unchanged. This delta closes that gap.
> **Base:** the post-session-002 tree (P1 complete: §8 config model + `init` shipped; §13 acceptance green). Reference that work rather than re-implementing.
> **Size:** one medium feature (the `completion` subcommand) + one tiny behavior flip. Single phase, three small milestones.

---

## What actually changed (diff analysis)

Commit `bbd4e74` modified only `PRD.md`. The code/files do not yet reflect it. Two independent changes:

### Change A — Bare invocation is now implicit `--help` (PRD §6.3, §6.4; decision 17)
- **Before:** `skilldozer` with no args and no flag ⇒ usage to **stderr**, exit **1** ("parity with `get-server-config.sh`").
- **After:** ⇒ usage to **stdout**, exit **0** (implicit `--help`), so `skilldozer | grep …` works — the help must land on the piped stream.
- §6.4 gained a lead note: bare no-args is **not** an error; the stderr/non-zero contract applies to genuine failures only (unknown flag, mutually-exclusive modes, unresolved tag, unconfigured, completion-shell errors). **Only no-args is reclassified (error→help); all genuine failures stay stderr/non-zero**, preserving the §6.4 `$(...)` contract.
- §13 gained a "Grepability contract" assertion block.

### Change B — New `completion` subcommand (PRD §6.1 row, §6.3, §6.4, §14.6; decision 18; §17 guardrail)
- `skilldozer completion [--shell <name>]` emits the completion script for the target shell to **stdout**, for `eval "$(skilldozer completion)"` (idiom: `zoxide init`/`starship init`/`direnv hook`).
- `completion` is a **reserved subcommand** like `check`/`init` (mutually exclusive with tags and other modes; a skill literally tagged `completion` resolves only via its full tag).
- Shell detection (first wins): `--shell <bash|zsh|fish>` → `$SKILLDOZER_SHELL` → `basename("$SHELL")` → none ⇒ stderr + exit `1`. An unsupported `--shell` value ⇒ stderr + exit `2`. On success the script goes to stdout and **nothing else**.
- The three on-disk `completions/` scripts are compiled into the binary with `//go:embed` (stdlib `embed`, **no new dependency**) so `go install` users with no clone get `completion` for free. The on-disk files remain the single source of truth; §14.5 (manual source/copy) and this subcommand emit **identical bytes**.
- §13 gained a `completion` acceptance block (bash/fish emit, no-shell detection failure, bad-shell exit 2).
- §17 gained a "Completion lockstep" guardrail; §14.4 lockstep note extended to the embedded bytes (editing `completions/*` requires a rebuild for `completion` to reflect it).

### What did NOT change (do not re-implement)
- §14.1–§14.5 describe completion **behavior the on-disk files already implement** (dynamic tags via `skilldozer --relative --all`; `--store` dir completion vs `--search` free-text; `2>/dev/null` robustness; lockstep comments; "shipped, not deferrable"). **No change required there** except the small lockstep addition in M3.
- Everything from session 002 (§8 config, `init`, example skill, README §8 content) is the stable base.

---

## P1 — `completion` subcommand + no-args implicit help

### M1 — No-args invocation → implicit help (stdout, exit 0)
**Status:** new. Tiny, localized, foundation for the §6.4 stream-separation guarantee that Change B also relies on.

**Affected code:** `main.go` `run()` no-mode fallthrough (currently `fmt.Fprint(stderr, usage()); return 1` at ~line 696–700); doc comments at `main.go:45-52`, `:417-427`, `:695-698`; tests `TestRunDefaultNoArgs` and `TestRunModifiersOnlyNoMode` in `main_test.go` (currently assert stderr/exit-1).

**Task P1.M1.T1 — Flip no-args/modifiers-only to stdout+exit0**
- **S1 — Flip the no-mode fallthrough + update its tests + doc comments.** Change the tail of `run()` so "no recognized mode" (truly no-args **and** modifiers-only like `--no-color` alone) prints `usage()` to **stdout** and returns **0** (implicit `--help`). Update the surrounding doc comments (run() exit-code list at `:417-421`, the fallthrough comment at `:695-698`, the usageText header note at `:51` that says "stderr for the no-args default") from stderr/exit-1 to stdout/exit-0/implicit-help. Update the two tests: `TestRunDefaultNoArgs` (`run(nil, …)`) and `TestRunModifiersOnlyNoMode` (`run(["--no-color"], …)`) — assert code==0, the USAGE block lands on **stdout**, and **stderr is empty** (the §13 "no-args writes NOTHING to stderr" assertion: `test -z "$(./skilldozer 2>&1 >/dev/null)"`). Keep the genuine-failure paths untouched (unknown flag→stderr/exit 2, exclusivity→stderr/exit 2, unresolved tag→stderr/exit 1, unconfigured→stderr/exit 1). *Mode A docs: none beyond the in-code comments (the USAGE block already exists; §15 README outline does not mention no-args behavior).*

### M2 — `completion` subcommand: parse, dispatch, embed, detect shell
**Status:** new. The bulk of the delta. Depends on M1's stream discipline (completion writes the script **only** to stdout; errors **only** to stderr).

**Affected code:** `main.go` — `config` struct, `parseArgs` (token switch + `=`-form switch), `exclusivityError`, `run()` dispatch, new `//go:embed` declarations + a `runCompletion`/shell-detection helper, `usageText` (add the `completion` row + `--shell` option). Tests in `main_test.go`.

**Task P1.M2.T1 — Parsing, exclusivity, and USAGE for `completion`**
- **S1 — `completion` token + `--shell` flag + config fields + USAGE row.** Add `completion bool` and `completionShell string` to the `config` struct. In `parseArgs`: add `case "completion":` to the token switch (reserves the subcommand, mirroring `case "check"`/`case "init"` — `completion` is NOT captured as a tag). Capture an optional following positional as `c.completionShell` only when it is a non-dashed, non-reserved token (mirror `init <dir>`'s capture rule; reject `completion completion`/`completion check`/`completion init` by letting the duplicate fall into tags so exclusivity flags it). Add `--shell` handling in **both** the `=`-form switch (`--shell=bash`) and the long-form switch (`--shell bash`, consume next token) — sets `c.completion=true` and `c.completionShell=<val>`; treat `--shell` seen **without** `completion` as an exclusivity conflict (it is only meaningful in completion context). **No short form** for `--shell`. Add the §6.1 USAGE/EXAMPLES/OPTIONS lines: `skilldozer completion [--shell <name>]` and `--shell <name>    Shell for the emitted completion script (bash|zsh|fish)`. Add `completion` to `exclusivityError` as its own reserved exclusive subcommand (like `check`/`init`): `completion` + tags, or `completion` + any of {check, init, path, list, searchMode, all} ⇒ exit 2 with a one-line message. Ship `main_test.go` cases: `completion` sets c.completion and no tags; `completion bash` and `completion --shell bash` / `--shell=bash` set completionShell; `completion --list`/`completion example` (a tag) / `check completion` ⇒ exit 2. *Mode A docs: the USAGE/help block (in-code `usageText`) IS the user-facing help surface for `completion` — update it in this subtask.*

**Task P1.M2.T2 — `//go:embed`, dispatch, and shell detection**
- **S1 — Embed the three scripts + map shell→bytes.** Add `//go:embed completions/skilldozer.bash` / `_skilldozer` / `skilldozer.fish` (three `var … string`, or a single `embed.FS`) at the top of `main.go`. Provide a `completionScript(shell string) (string, bool)` mapping `bash`→bash bytes, `zsh`→zsh bytes, `fish`→fish bytes (return ok=false for any other name). The on-disk `completions/` files are the single source of truth; this embeds them verbatim. (Note: the existing `// STRING CONSTANT (NOT go:embed …)` comment at `main.go:962` is about the `init` **seed template** — user data, correctly not embedded — and is unrelated; embedding the repo's own shipped completion scripts is the §14.6 requirement and is fine.) Stdlib `embed` only — **no new dependency** (go.mod is `go 1.25`).
- **S2 — `run()` dispatch + shell detection + exit codes.** Insert `if c.completion { return runCompletion(c, stdout, stderr) }` in `run()` at the appropriate precedence slot (alongside the other exclusive subcommand dispatches, after exclusivityError and after the `c.init` block). `runCompletion`: (1) resolve shell = `c.completionShell` (if set) else `$SKILLDOZER_SHELL` else `basename(os.Getenv("SHELL"))`; (2) if no shell resolvable (all empty) ⇒ stderr `could not detect shell; pass --shell {bash|zsh|fish}`, exit **1**, nothing on stdout; (3) validate shell ∈ {bash,zsh,fish}; else stderr error naming the bad value, exit **2**, nothing on stdout; (4) `fmt.Fprint(stdout, completionScript(shell))`, return **0**. Extract a testable core for detection (e.g. `detectShell(explicit, env, loginShell string) (shell string, ok bool)`) so detection is unit-testable without env mutation. Ship `main_test.go`: `completion --shell bash` ⇒ stdout contains `_skilldozer_completion`, exit 0; `completion --shell fish` ⇒ stdout contains `complete -c skilldozer`, exit 0; `--shell tcsh` ⇒ stderr + exit 2, empty stdout; no explicit + no env + no `$SHELL` ⇒ stderr mentions "shell", exit 1, empty stdout; `$SKILLDOZER_SHELL` honored; `basename($SHELL)` honored (e.g. SHELL=…/zsh). *Mode A docs: none beyond code (the §14.6 emit semantics are runtime behavior, not a doc file).*

### M3 — Completion-file lockstep + changeset-level documentation (Mode B)
**Status:** new. Depends on M2 (the `completion` subcommand must exist before docs/lockstep reference it). Small.

**Task P1.M3.T1 — Lockstep `completion` into the three completion files + sync README**
- **S1 — Add `completion` as a completable first-arg subcommand to all three completion files.** Per §6.3 (`completion` is a reserved subcommand *like* `check`/`init`) and the §14.4 lockstep invariant, offer `completion` alongside `check`/`init` as an exclusive first-positional subcommand in `completions/skilldozer.bash`, `completions/_skilldozer`, and `completions/skilldozer.fish` (same gating as check/init: offered only as the first positional; tag completion suppressed once seen). Do **not** add `--shell` to the global flag matrix — §14.1/§14.2 define the flag matrix as §6.1/§6.2 only (`--shell` is completion-context-only), and §14.2 names exactly `--search`/`--store` as the value-taking flags. Keep tag completion (`skilldozer --relative --all`) unchanged. Structural grep asserting `completion` is offered alongside `check`/`init` in all three files; no behavioral regression. *Mode A: the completion files ARE the user-facing surface for tab-completion — updating them here satisfies doc-with-work.*
- **S2 — README: document the `completion` subcommand as the easy install path (Mode B).** The README "Shell completions" section currently documents only the manual source/copy path (§14.5). Add the §14.6 `eval`/`source` idiom as the recommended one-liner alongside it: `eval "$(skilldozer completion)"` for bash/zsh and `skilldozer completion --shell fish | source` for fish, with a one-line note that the binary embeds the scripts (works for `go install` with no clone). Note `--shell` for deterministic `eval` and the `$SKILLDOZER_SHELL`/`$SHELL` fallback. Keep the existing manual source/copy instructions (they remain valid and pick up edits without a rebuild). Do not duplicate the §14.6 rationale verbatim — match the README's existing tone. Verified by: `grep -q 'skilldozer completion' README.md`. *Mode B: this is the changeset-level documentation sync — it depends on M2 (the feature) and M3.S1 (lockstep), and it is the final deliverable of the delta.*

---

## Acceptance (the new §13 assertions must pass)

These are the exact new §13 blocks; the rest of §13 (already green in session 002) must remain green. Run from a clean build:

```bash
go build -o skilldozer . && echo OK

# Grepability contract (§6.3): no-args help is on stdout, exit 0 — pipes MUST see it
out=$(./skilldozer 2>/dev/null); rc=$?
[ "$rc" = "0" ] && printf '%s' "$out" | grep -qi 'USAGE' && echo "no-args-help-on-stdout OK" || { echo "FAIL"; exit 1; }
test -z "$(./skilldozer 2>&1 >/dev/null)"   # no-args writes NOTHING to stderr

# `completion` subcommand (§14.6): emits the matching script to stdout; --shell forces one
./skilldozer completion --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo "completion-bash OK" || { echo "FAIL"; exit 1; }
./skilldozer completion --shell fish 2>/dev/null | grep -q 'complete -c skilldozer' && echo "completion-fish OK" || { echo "FAIL"; exit 1; }
# detection failure (no --shell, no $SHELL) ⇒ stderr + exit 1, nothing on stdout
out=$(env -u SHELL -u SKILLDOZER_SHELL ./skilldozer completion 2>cerr); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && grep -qi 'shell' cerr && echo "completion-no-shell OK" || { echo "FAIL"; exit 1; }
# unsupported --shell value ⇒ exit 2
./skilldozer completion --shell tcsh >/dev/null 2>&1; [ "$?" = "2" ] && echo "completion-bad-shell OK" || { echo "FAIL"; exit 1; }
```

**Regression guard (must stay green):** `--help`/`-h` still stdout/exit 0 and still wins over everything; `completion` is mutually exclusive (`completion example`, `check completion`, `completion --list` ⇒ exit 2); genuine failures stay stderr/non-zero — in particular the `$(...)` contract: `./skilldozer nope` ⇒ empty stdout, exit 1 (unchanged from session 002); `completion` errors write **nothing** to stdout.

---

## Notes for the implementer

- **Proportional scope.** This is one medium feature + one tiny flip. Do not redesign §14 or re-plumb discovery/config (session 002 owns those and they are green).
- **No new dependency.** `embed` is stdlib (go 1.16+; go.mod is `go 1.25`). `yaml.v3` remains the only non-stdlib dep.
- **Reuse, don't rebuild.** Mirror `case "check"`/`case "init"` for the new `case "completion"`; mirror `--search`'s `=`-form + next-token capture for `--shell`; mirror the existing `exclusivityError` family style for the `completion` family.
- **Lockstep is load-bearing (§14.4/§17).** Once `completion` is a recognized token in `parseArgs`, keep the three completion files (and the embedded bytes) consistent. M3.S1 closes the on-disk half; the embedded half is correct by construction (it embeds those same files).
