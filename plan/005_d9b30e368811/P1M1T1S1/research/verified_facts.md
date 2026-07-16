# Verified Facts — P1.M1.T1.S1 (bash: show-all-if-ambiguous in the emitted script)

Confirmed against the live source (`completions/skilldozer.bash`, `main.go`,
`main_test.go`) and the plan/005 architecture map. Implements PRD §14.7 / decision 22
for bash. Scope: ONLY `completions/skilldozer.bash` (zsh const = S2; tests = P1.M1.T2;
README = P1.M1.T3; fish = no change).

## 0. Line-count discrepancy (resolved)

The contract and the architecture map both say the bash file is "69 lines". The LIVE
file is **67 lines** (verified `wc -l`). Both are slightly stale, but the load-bearing
claim — the LAST line is `complete -F _skilldozer_completion skilldozer` — is correct
(it is line 67). The plan/005 base already has `--shell` (lines 18-19, 44, 50) and
`--link` (lines 38, 43, 50) landed. Append AFTER line 67.

## 1. bash is emitted VERBATIM — one file edit covers both delivery paths

`runCompletion` (main.go:1499-1527):
```go
script, ok := completionScript(shell)   // "bash" → bashCompletion, verbatim
...
if shell == "zsh" { script = zshEvalScript(script) }  // ONLY zsh is derived
fmt.Fprint(stdout, script)
```
`completionScript("bash")` returns `bashCompletion` (the `//go:embed` var, main.go:1217)
byte-for-byte. So the §14.5 manual `source`/`copy` path and the §14.6 `eval
"$(skilldozer --completions)"` path produce **identical bytes**. Editing
`completions/skilldozer.bash` is the ONLY change needed for bash — no Go code, no
derivation. (zsh is different: its eval output is derived in `zshEvalScript` — that's
S2's territory, a `main.go` const edit, NOT this task.)

## 2. The embed + rebuild mechanic

- `//go:embed completions/skilldozer.bash` → `var bashCompletion string` (main.go:54-55).
- `//go:embed` reads the file at COMPILE time. After editing the on-disk file, a
  `go build` (or `go test`, which compiles) re-embeds the NEW bytes.
- A PRE-BUILT `./skilldozer` binary holds the OLD embedded bytes until rebuilt. Always
  rebuild before behavioral testing (`--completions --shell bash`).

## 3. The append (after line 67 `complete -F _skilldozer_completion skilldozer`)

Three elements (contract LOGIC a/b + OUTPUT "commented opt-out"):

**(a) Disclosure comment block** naming `show-all-if-ambiguous`, stating:
- §14.7 / decision 22 intent (list every ambiguous match on the FIRST Tab — a
  manifest-free store makes completion the primary discovery path);
- bash default is OFF (first Tab completes the common prefix + beeps; list on 2nd Tab);
- it is a READLINE SESSION-GLOBAL option (changes listing for EVERY command in the
  shell, not just skilldozer — no per-command scope);
- the `[[ $- == *i* ]] &&` guard rationale (silences `bind`'s warning when sourced
  non-interactively, e.g. an eval test harness).

**(b) Active guarded line:**
```bash
[[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'
```
`[[ $- == *i* ]]` is the standard interactive-shell guard (`$-` contains `i` only in
interactive bash; the glob `*i*` matches inside `[[ ]]`). The guard is load-bearing:
`bind` in a non-interactive shell prints a warning; the guard silences it. Completions
only matter interactively, so the option still applies where it counts.

**(c) Commented opt-out:**
```bash
# Opt-out — restore bash's stock (second-Tab) listing:
#   bind 'set show-all-if-ambiguous off'
```
PRD §14.7 requires the emitted script "provide the one-line opt-out" since the option
is session-global and the user must be able to restore stock behavior.

Exact target text is in the PRP's Implementation Patterns.

## 4. Existing tests STAY GREEN (no new test in this subtask)

- **TestEmbeddedCompletionsMatchOnDisk** (main_test.go:3139): asserts
  `completionScript("bash") == on-disk completions/skilldozer.bash` (byte identity).
  The embed var and the on-disk file MOVE TOGETHER (same file), so after a rebuild the
  test stays GREEN. It does NOT assert the option exists.
- **TestRunCompletionBashScript** (main_test.go:3163): asserts `run(["--completions",
  "--shell","bash"])` → code 0, stdout contains `_skilldozer_completion`, Go stderr
  empty. The append ADDS content after the function; `_skilldozer_completion` (defined
  at line 20) is still present; Go stderr (errOut) is unaffected by script content
  (runCompletion does `fmt.Fprint(stdout, script)`). Stays GREEN.

The NEW test that ASSERTS `show-all-if-ambiguous on` (and the opt-out token) in the
emitted output is **P1.M1.T2.S1** (a separate subtask — "Tests locking the
emitted-byte contract"). So S1's automated gate is "existing tests stay green" + the
manual CLI grep; the locking test comes next. Do NOT add the asserting test in S1.

## 5. The behavioral proof (manual, post-rebuild)

```bash
go build ./...
./skilldozer --completions --shell bash 2>/dev/null | grep -q 'show-all-if-ambiguous on'   # → found
./skilldozer --completions --shell bash 2>/dev/null | grep -q 'show-all-if-ambiguous off'  # → found (opt-out)
bash -n completions/skilldozer.bash                                                         # syntax OK
```
And the end-to-end (interactive): `eval "$(./skilldozer --completions --shell bash)"` in a
real interactive bash, then `skilldozer --c<Tab>` lists `--check`/`--completions` on the
first Tab (instead of completing `--c` + beep). (Hard to assert in CI; the grep is the
deterministic gate.)

## 6. Scope boundary (what this subtask is NOT)

- NOT zsh: the zsh eval-path option (`NO_LIST_AMBIGUOUS`) goes in `main.go`'s
  `zshEvalRegistration` const (Touch point 2) → **S2**. Do NOT edit `completions/_skilldozer`
  or any Go file here.
- NOT a new Go test: the byte-level assertion (`show-all-if-ambiguous on` in emitted
  output) is **P1.M1.T2.S1**. S1 only ensures existing tests stay green.
- NOT the README disclosure (§15): that's **P1.M1.T3.S1** (Mode B).
- NOT fish: fish lists in the pager by default (§14.7); no option to set.
- NOT any parseArgs / run() / usageText change.

## 7. Deps / build

No `.go` file is edited (the `//go:embed` picks up the file change on rebuild). No new
imports, no new deps. `go.mod`/`go.sum` byte-for-byte identical. The sole edit is a
shell data asset appended with comment + one active `bind` line + commented opt-out.
