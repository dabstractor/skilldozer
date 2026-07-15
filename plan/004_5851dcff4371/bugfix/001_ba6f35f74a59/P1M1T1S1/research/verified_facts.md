# Verified Facts — P1.M1.T1.S1 (--shell routing + advertisement in bash completion)

Confirmed against the live source (`completions/skilldozer.bash`, `main.go`,
`main_test.go`) and the bugfix-round-2 architecture docs. Scope: ONLY the bash
completion file (zsh = S2, fish = S3 are sibling subtasks).

## 1. The bash file (62 lines) — exact current structure

`completions/skilldozer.bash` is embedded verbatim via `//go:embed
completions/skilldozer.bash` → `var bashCompletion string` (main.go:54-55).
`completionScript("bash")` returns it (main.go:1116). Four regions matter:

**Value-routing doc comment (lines 33-36):**
```
    # Value-taking flags: route the value slot away from tag completion.
    #   --search        -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
    #   --store/--init  -> directory value  -> complete DIRECTORIES via compgen -d.
    # (--store/--init WANT path completion, unlike --search's free-text -> nothing.)
```

**The `case "$prev" in` block (lines 37-40) — THE VALUE-ROUTING SITE:**
```
    case "$prev" in
        --search) return 0 ;;
        --store|--init) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
    esac
```
There is NO `--shell)` case today ⇒ after `--shell`, prev="--shell" matches none of
these, falls past the `case`, past the `-*` flag branch, into the tag-completion
default (line 59) ⇒ offers SKILL TAGS (the bug).

**The flag-advertisement list (lines 44-46) — THE DISCOVERY SITE:**
```
        COMPREPLY=($(compgen -W \
            "--version --help --path --list --all --file --relative --no-color --search --store --check --init --completions" \
            -- "$cur"))
```
13 flags; `--shell` absent ⇒ `skilldozer --<TAB>` does not offer `--shell`.

**LOCKSTEP header (lines 11-17):** documents the freeze to `parseArgs()` and the
long-form-only / decision-19 rationale. Currently says "Updated for
--check/--init/--completions (decision 19)". Does NOT mention --shell.

## 2. The two required edits (+ two doc-comment touches for accuracy)

**(a) Value routing — add a `--shell)` case AFTER the `--store|--init)` line (line 39):**
```bash
        --shell) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;
```
This makes `skilldozer --shell <TAB>` offer exactly `bash zsh fish` (PRD §14.2: the
fixed enum, nothing else). `return 0` prevents fall-through to tag completion.
Order in the `case` is irrelevant (--shell does not overlap --search/--store|--init).

**(b) Flag advertisement — add `--shell` to the compgen -W list (line 45):**
Place after `--store` (groups the value-taking flags: `--search --store --shell`).
Result (14 flags):
```
"--version --help --path --list --all --file --relative --no-color --search --store --shell --check --init --completions"
```
Exact position within the space-delimited list is NOT behaviorally significant
(compgen -W matches by prefix), but grouping value-taking flags is the tidy choice.
D7 rationale: --shell is a real documented flag (in usageText OPTIONS), so it should
be discoverable via `--<TAB>`.

**(c) Value-routing doc comment (lines 33-36) — add a --shell line for accuracy:**
```
    #   --shell         -> fixed enum      -> offer "bash zsh fish" via compgen -W.
```
(Mode A: the in-file doc rides with the change.)

**(d) LOCKSTEP header (lines 14-17) — mention --shell:** append a note that --shell's
value completes to the bash/zsh/fish enum (§14.2) and --shell is now advertised (D7).
Keeps the header accurate per the contract's "keep intact and accurate" requirement.

## 3. The automated gate — TestEmbeddedCompletionsMatchOnDisk (main_test.go:2995)

Asserts `completionScript("bash") == on-disk completions/skilldozer.bash` (byte
identity, PRD §14.6). `//go:embed` reads the file at COMPILE time; `go test` compiles
the package (re-running embed), so after editing the on-disk file, `go test` re-embeds
the NEW bytes ⇒ the test passes automatically. No separate `go build` is needed for the
test (go test builds), but a `go build` is the cleanest proof the embed picked up the
edit.

CRITICAL: if you edit the on-disk file and run a PRE-BUILT binary (not rebuilt), the
embedded var still holds the OLD bytes and `--completions --shell bash` emits the OLD
script. Always rebuild before behavioral testing.

## 4. No existing test asserts the bash file's flag list / case content

Grepped main_test.go: the only tests touching the bash completion are
`TestEmbeddedCompletionsMatchOnDisk` (byte identity, per-file) and
`TestRunCompletionBashScript` (run `--completions --shell bash` → stdout contains
`_skilldozer_completion`, stderr empty). Neither asserts the specific flag list or the
case block. So editing bash to add --shell breaks NO existing test. There is NO
cross-file lockstep test, so editing ONLY bash (zsh/fish in S2/S3) is safe — the
temporary cross-file divergence is expected and resolved when S2/S3 land.

## 5. The repro (contract OUTPUT §4) — the behavioral proof

```bash
bash -c 'eval "$(./skilldozer --completions --shell bash)"; \
  COMP_WORDS=(skilldozer --shell ""); COMP_CWORD=2; _skilldozer_completion; \
  echo "COMPREPLY=[${COMPREPLY[*]}]"'
# Expected (after fix): COMPREPLY=[bash zsh fish]
# (Pre-fix: COMPREPLY=[example foo writing/reddit] — skill tags.)
```
And `skilldozer --<TAB>` should now offer `--shell` among the flags.

## 6. Scope boundary (what this subtask is NOT)

- NOT zsh (`completions/_skilldozer` → P1.M1.T1.S2) or fish
  (`completions/skilldozer.fish` → P1.M1.T1.S3). Per §14.4 the three files should
  ultimately match, but S1/S2/S3 are a sequence; lockstep is restored when all three
  land. S1 edits ONLY `completions/skilldozer.bash`.
- NOT any Go source change. main.go's //go:embed + completionScript are unchanged
  (the embed picks up the edited file automatically). No parseArgs / run() change.
- NOT a usageText change (usageText already lists --shell in OPTIONS — confirmed by
  D7 "--shell is a real, documented flag in usageText OPTIONS").
- NOT a README change (Mode B final sweep is P1.M3.T1).

## 7. Deps / build

No Go imports change (no .go file is edited). `go.mod`/`go.sum` byte-for-byte
unchanged. The only edited file is `completions/skilldozer.bash` (a data asset).
