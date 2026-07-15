# Verified Facts — P1.M1.T1.S3 (fish completion `--shell`)

All line numbers verified against the live working tree at HEAD `147b177`.
S3 is the FISH third of P1.M1.T1 (Issue 1). S1 (bash) is committed; S2 (zsh)
is in the working tree (` M completions/_skilldozer`); the fish file is CLEAN —
this is S3's input.

---

## §0 — Lockstep state (what exists when S3 starts)

```
$ git rev-parse --short HEAD
147b177
$ git status --short completions/
 M completions/_skilldozer        # S2 (zsh) — being implemented in parallel; assume DONE when S3 runs
                                  # completions/skilldozer.bash — S1, COMMITTED (HEAD moved past 6fb3f7e)
                                  # completions/skilldozer.fish — CLEAN (S3's input)
```

S1 (bash) added a `--shell)` case + flag-list token + doc touches. S2 (zsh)
added a `'--shell[...]:shell:(bash zsh fish)'` array entry + header note.
S3 does the fish equivalent. After S3, all three completion files carry
`--shell` → §14.4 lockstep fully restored.

---

## §1 — The edit site: `completions/skilldozer.fish` (current, exact text)

The fish file is a FLAT LIST of top-level `complete` directives (NOT a
function like bash, NOT an autoload `_arguments` array like zsh). There is
NO trailing self-call and NO eval-safe wrapper — fish sources the directives
directly. (Critical difference from zsh S2: nothing to strip, nothing to fear.)

```
 1: # Fish completion for skilldozer.                              ← header (TestCompletionScriptMapping marker; UNTOUCHED)
...
 9-15: LOCKSTEP + long-form-only + decision-19 header notes        ← append --shell note after line 15
16: (blank)
17: # No file completion: skilldozer takes tags/flags, not paths.
18: complete -c skilldozer -f                                       ← GLOBAL -f (no-files) — load-bearing context for the -r/-x distinction
...
31: complete -c skilldozer -l check       -d '...'
34: complete -c skilldozer -l init        -d '...' -r              ← value-taking, PATH (-r → files)
35: complete -c skilldozer -l completions -d '...'                 ← bare flag (no value)
43: complete -c skilldozer -l search -d '...'                      ← value-taking, FREE-TEXT (NO -r → global -f applies → offer nothing)
50: complete -c skilldozer -l store -d 'Non-interactive store path for init' -r   ← value-taking, PATH (-r → files)
51: (blank)
52-55: dynamic tags directive (the default positional handler)
```

**S3's value-directive edit**: insert AFTER line 50 (--store) and BEFORE the
blank line 51 / the dynamic-tags section. The `--shell` directive is the
THIRD value-routing pattern in the file (see §3).

---

## §2 — The two exact edits

### Edit A — the `--shell` value directive (+ explanatory comment)

Place after the `--store` directive (line 50), before the blank line (51).
The comment mirrors the file's established "comment block above each
value-taking flag" style (--search block lines 36-42, --store block 45-49).

```
OLD (lines 50-52):
complete -c skilldozer -l store -d 'Non-interactive store path for init' -r

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per

NEW (insert the --shell block between --store and the blank line):
complete -c skilldozer -l store -d 'Non-interactive store path for init' -r
# --shell <name> (PRD §14.2): Force a shell for completion. The value is a FIXED
# enum (bash/zsh/fish), so use `-x` (exclusive: require a value, NO file
# completion) + `-a "bash zsh fish"` (the three candidates). This is the THIRD
# value-routing pattern: --search = nothing (no flag), --store/--init = files
# (-r), --shell = closed enum (-x -a). --shell is advertised (decision D7).
complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
```

The directive line is verbatim from the contract + issue_analysis:
`complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"`

### Edit B — the LOCKSTEP header note (mirror S1/S2 verbatim)

```
OLD (line 15, the last header line):
# belongs to skill tags — a bare <tab> shows skills, never commands.

NEW (append two lines):
# belongs to skill tags — a bare <tab> shows skills, never commands.
# --shell's value completes to the bash/zsh/fish enum (§14.2); --shell is
# advertised (D7) since it is a real, documented flag in usageText OPTIONS.
```

This is byte-identical to S1's bash header (lines 17-18) and S2's zsh header
append → the three files' headers stay in lockstep.

---

## §3 — WHY `-x -a` is correct (the one fish-specific subtlety)

The file's OWN comment (lines 45-49, the --store block) establishes the
codebase's mental model of fish's `-r`:

> in fish 4.x `-r` switches into "complete the option's value" mode, which
> BYPASSES the global `-f` above and offers file/dir paths for the value.

So there are THREE value-routing patterns, all already present or being added:

| Flag        | Value kind   | Directive form                  | Behavior of the value slot               |
|-------------|-------------|---------------------------------|------------------------------------------|
| `--search`  | free-text   | (no `-r`, plain flag)           | global `-f` applies → offer NOTHING      |
| `--store`/`--init` | directory | `-r`                    | `-r` bypasses `-f` → offer FILES/dirs    |
| `--shell`   | closed enum | `-x -a "bash zsh fish"` (NEW)   | `-x` = require value + NO files → offer ONLY the `-a` enum |

Fish `complete` semantics (https://fishshell.com/docs/current/cmds/complete.html):
- `-r` / `--require-parameter`: the option requires a value; the value is
  completed using the DEFAULT (file) completion.
- `-x` / `--exclusive`: the option's completions are EXCLUSIVE — it requires a
  value AND file completion is suppressed, so ONLY the `-a` list is offered.
  (Equivalently `-x` combines "require a value" with "no files".)
- `-a` / `--arguments`: the candidate values for the (value) slot.

`--shell` wants a fixed enum and NO files → `-x -a "bash zsh fish"` is exactly
right, and it is the deliberate INVERSE of `--store`'s `-r` (which WANTS files).
Using `-r` here would be a bug: it would offer files for `--shell` instead of
the enum. Using no flag (like `--search`) would be a bug: it would offer
nothing (global `-f`), so the enum never appears. ONLY `-x -a` gives the enum.

---

## §4 — The `-a` quoting: double quotes are CORRECT here (NOT single)

The directive uses `-a "bash zsh fish"` (double quotes). This is intentional
and correct: fish tokenizes the `-a` argument into three candidates (bash, zsh,
fish) by splitting on spaces inside the double-quoted string. Single quotes
would also work in fish for a space-separated word list, but the contract +
issue_analysis prescribe double quotes, and double quotes are the idiomatic
fish form for `-a` word lists (matching e.g. `complete -c git -a "add commit"`).
The `-d 'Force a shell for completion'` description stays SINGLE-quoted (matches
every other `-d` in the file — descriptions are single-quoted throughout).
Do NOT "normalize" the quotes to match; the two arguments legitimately differ.

---

## §5 — Embed wiring (the mechanism — NO edit to main.go)

```
main.go:60  //go:embed completions/skilldozer.fish
main.go:61  var fishCompletion string
main.go:1113-1121  func completionScript(shell string) (string, bool) {
                       switch shell {
                       ...
                       case "fish": return fishCompletion, true
                       }
                   }
```

Editing the on-disk fish file + `go build`/`go test` re-embeds the new bytes
into `fishCompletion` automatically. **Do NOT touch main.go.** No eval-safe
wrapper exists for fish (unlike zsh's `zshEvalScript`), so there is no strip
interaction to worry about — every directive is emitted verbatim.

---

## §6 — Tests that gate the change (all PASS after edit + rebuild)

| Test | main_test.go | What it asserts | After S3 |
|------|--------------|-----------------|----------|
| `TestCompletionScriptMapping` | :2957 | `completionScript("fish")` contains header `# Fish completion for skilldozer.` (line 1) | PASS — line 1 untouched |
| `TestEmbeddedCompletionsMatchOnDisk` | :2995 | `completionScript("fish")` == on-disk `completions/skilldozer.fish` (byte identity) | PASS — rebuild re-embeds |
| `TestRunCompletionFishScript` | :3035 | `run(["--completions","--shell","fish"])` exit 0, stdout contains `complete -c skilldozer` | PASS — the new --shell directive ALSO starts with `complete -c skilldozer` (more matches, still true) |

No test asserts the absence of `--shell`, and no test asserts the exact
flag-directive set. So nothing breaks. The byte-identity gate passes
automatically on rebuild (that's its whole point).

---

## §7 — Scope boundary (fish ONLY; do NOT touch bash/zsh/Go)

- Edit ONLY `completions/skilldozer.fish`. bash (S1, committed) and zsh (S2,
  working tree) are owned by other subtasks.
- Do NOT edit any `.go` file (main.go //go:embed, parseArgs, run, usageText —
  none). `parseArgs` already accepts `--shell`; that's why §14.4 says the
  completion files are frozen to parseArgs.
- Do NOT change `usageText` (it already documents `--shell` — decision D7).
- Do NOT edit the README (the Mode B doc sweep is P1.M3.T1).
- Do NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`,
  or `.gitignore`.
- No deps change: no `.go` file is edited → `go.mod`/`go.sum` byte-identical.

---

## §8 — Level-3 behavioral repro (fish; requires real fish)

```bash
go build -o /tmp/sdz .

# Deterministic byte gate: the rebuilt binary's emitted script carries the --shell directive.
grep -F 'complete -c skilldozer -l shell -d ' /tmp/sdz --completions --shell fish
# Expected: the --shell directive line with -x -a "bash zsh fish".

# Behavioral (fish installed): after sourcing, complete -C "skilldozer --shell " offers the enum.
fish -c '/tmp/sdz --completions --shell fish | source; for l in (complete -C "skilldozer --shell "); echo $l; end'
# Expected (post-fix): three lines — bash, zsh, fish (each with the description).
# Pre-fix: offered skill tags (example, foo, writing/reddit …).
```
