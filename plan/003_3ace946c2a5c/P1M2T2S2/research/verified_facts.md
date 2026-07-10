# Verified Facts — P1.M2.T2.S2 (`run()` dispatch + shell detection + exit codes)

> All signatures/behaviors below were read DIRECTLY from source at PRP-write time.
> Locate symbols by NAME, not line number — line numbers shift as the parallel sibling
> P1.M2.T2.S1 (embed) lands, and as other deltas accrue.

## 1. INPUT contract (what exists when this subtask starts)

### 1a. From P1.M2.T1.S1 (parsing/exclusivity) — COMPLETE (firm input, not parallel)
The local `config` struct (main.go, `type config struct`) already has:
```go
completion        bool     // `skilldozer completion` subcommand (PRD §14.6); exclusive like check/init
completionShell   string   // `--shell <bash|zsh|fish>` value; "" ⇒ detect from $SKILLDOZER_SHELL/$SHELL
```
- `parseArgs` populates them: `case "completion"` sets `c.completion = true` (main.go ~line 296);
  `--shell <dir>` / `--shell=<dir>` set `c.completion = true` + `c.completionShell = <val>`
  (main.go ~lines 227-230 and 313-318).
- `exclusivityError` ALREADY rejects completion+other-modes (main.go ~lines 809-813): when
  `c.completion` is true, it errors on stray tags OR any of check/init/list/search/all/path.
  ⇒ **When `runCompletion` is called, `c.completion` is true and NO other mode is active.**
  The dispatch sits AFTER the exclusivity gate, so runCompletion never sees a conflicting mode.

### 1b. From P1.M2.T2.S1 (embed) — being implemented in PARALLEL (treat as a CONTRACT)
- `func completionScript(shell string) (string, bool)` — ALREADY LANDED at main.go:1110
  (verified by `grep -n '^func completionScript'`). It is a pure switch: `bash`→(bashCompletion,true),
  `zsh`→(zshCompletion,true), `fish`→(fishCompletion,true), else ("",false). The three embedded
  string vars (`bashCompletion`/`zshCompletion`/`fishCompletion`) are at main.go:55/58/61.
- **This subtask CONSUMES `completionScript`** (it is uncalled until runCompletion wires it — Go
  allows unused package-level functions). Do NOT modify completionScript or the embed vars.

## 2. The run() dispatch region (WHERE to insert the slot) — verified by reading source

The relevant region of `run(args, stdout, stderr io.Writer) int` (main.go:476):

```go
	// (exclusivity gate runs here, ~line 520: exclusivityError → exit 2)

	// init dispatch (PRD §8.2). init is an exclusive mode: exclusivityError
	// above guarantees no other mode is set when c.init is true, so this
	// self-contained branch returns before the path/list/search/check/all/tags
	// ladder below. ...
	if c.init {
		return runInit(c, stdout, stderr)
	}

	// 5) Normal mode dispatch (order: path → list → search → check → all → tags). ...
	if c.path { ... }
```

**INSERT the completion dispatch BETWEEN the `if c.init { … }` block and the `// 5) Normal mode
dispatch` comment.** It is an exact mirror of the init dispatch:

```go
	if c.init {
		return runInit(c, stdout, stderr)
	}

	// completion dispatch (PRD §14.6). completion is an exclusive mode: exclusivityError
	// above guarantees no other mode is set when c.completion is true, so this self-contained
	// branch returns before the path/list/search/check/all/tags ladder below.
	if c.completion {
		return runCompletion(c, stdout, stderr)
	}

	// 5) Normal mode dispatch ...
```

- The contract's "main.go:482" / "main.go:518" line anchors are STALE (the file has grown to 1204
  lines). Anchor by SYMBOL: "immediately after the `if c.init { return runInit(c, stdout, stderr) }`
  block, immediately before the `// 5) Normal mode dispatch` comment."

## 3. The three new functions — placement at the FILE TAIL (after runInit)

- `func runInit(c config, stdout, stderr io.Writer) int` is the LAST function in main.go
  (starts main.go:1135; the file ends at line 1204 with runInit's closing `}` — verified by
  `tail -6 main.go`). APPEND `detectShell`, `loginShellBase`, and `runCompletion` immediately
  AFTER runInit (the file tail). This is the established "append at the tail" discipline and is
  DISJOINT from the parallel sibling P1.M2.T2.S1, which edits the import block + after-var-version
  (lines 44-61) + before-the-runInit-doc-comment (completionScript @1110). My append is AFTER
  runInit (line 1204) — no overlap.

### 3a. detectShell (pure; no env mutation) — verbatim from external_deps.md
```go
// detectShell resolves the target shell for `skilldozer completion` (PRD §14.6 "Shell detection",
// first wins): explicit --shell → $SKILLDOZER_SHELL → basename($SHELL). It is a PURE function of
// its three string args (the caller supplies the env reads), so detection is unit-testable without
// env mutation. Returns the first non-empty value + true, or ("", false) if all three are empty.
// NOTE: it returns the value VERBATIM — only loginShellBase lowercases (it computes the basename);
// explicit/envShell are passed through as typed, so `--shell BASH` → "BASH" → unsupported (exit 2).
func detectShell(explicit, envShell, loginShell string) (string, bool) {
	if explicit != "" {
		return explicit, true
	}
	if envShell != "" {
		return envShell, true
	}
	if loginShell != "" {
		return loginShell, true
	}
	return "", false
}
```
Called as `detectShell(c.completionShell, os.Getenv("SKILLDOZER_SHELL"), loginShellBase())`.

### 3b. loginShellBase (guards filepath.Base("") → ".")
```go
// loginShellBase returns the lowercased basename of $SHELL ("zsh" for /bin/zsh, "fish" for
// /usr/bin/fish), or "" if $SHELL is unset/empty. The empty guard is load-bearing: filepath.Base("")
// returns ".", which would otherwise pollute detection (PRD §14.6; external_deps.md §Shell detection).
// os.Executable() is NOT used here — that is the skilldozer binary path, not the shell.
func loginShellBase() string {
	s := os.Getenv("SHELL")
	if s == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(s))
}
```

### 3c. runCompletion (mirrors runInit's signature; the §6.4 exit-code ladder)
```go
// runCompletion is the `skilldozer completion` handler (PRD §14.6 / §6.4). run()'s dispatch calls it
// when c.completion is true (completion is exclusive, so no other mode is active). It resolves the
// shell, then emits the matching embedded script to stdout for `eval "$(skilldozer completion)"`.
// Exit codes (PRD §6.4): 0 on success; 1 if the shell is undetectable (no --shell, no
// $SKILLDOZER_SHELL, no usable $SHELL); 2 if the resolved shell value is not bash/zsh/fish.
// On the 1/2 paths NOTHING is written to stdout (so the §6.4 $(...) contract holds).
func runCompletion(c config, stdout, stderr io.Writer) int {
	shell, ok := detectShell(c.completionShell, os.Getenv("SKILLDOZER_SHELL"), loginShellBase())
	if !ok {
		fmt.Fprintln(stderr, "could not detect shell; pass --shell {bash|zsh|fish}")
		return 1
	}
	script, ok := completionScript(shell)
	if !ok {
		fmt.Fprintf(stderr, "skilldozer: unsupported shell '%s' (want bash|zsh|fish)\n", shell)
		return 2
	}
	fmt.Fprint(stdout, script)
	return 0
}
```

## 4. CRITICAL — `fmt.Fprint`, NOT `fmt.Fprintln`, for the script

The embedded completion scripts already END WITH a trailing newline (the on-disk files do;
`//go:embed` reads exact bytes; S1's `TestEmbeddedCompletionsMatchOnDisk` locks byte-identity).
So `fmt.Fprint(stdout, script)` emits the EXACT file bytes. `fmt.Fprintln` would append an extra
newline, breaking the §14.6 "emit identical bytes (both read the same files)" invariant. (The §13
acceptance only greps for substrings, so Fprintln would not FAIL acceptance — but Fprint is the
byte-correct choice and matches the contract.)

## 5. The exit-code ladder (PRD §6.4) — two distinct failure paths

| Situation | detectShell returns | completionScript returns | Exit | stderr | stdout |
|---|---|---|---|---|---|
| `--shell bash` / detected bash/zsh/fish | (shell, true) | (script, true) | **0** | empty | the script |
| ALL of explicit/env/login empty (no --shell, no $SKILLDOZER_SHELL, no $SHELL) | ("", **false**) | not called | **1** | "could not detect shell; pass --shell {bash\|zsh\|fish}" | empty |
| a detected but UNSUPPORTED value (e.g. `tcsh`, or `$SHELL=/bin/sh` → "sh") | (shell, **true**) | ("", **false**) | **2** | "skilldozer: unsupported shell '<v>' (want bash\|zsh\|fish)" | empty |

- **Key subtlety:** `$SHELL=/bin/sh` ⇒ loginShellBase()="sh" ⇒ detectShell returns ("sh", **true**)
  ⇒ completionScript("sh")=("false") ⇒ **exit 2** (unsupported), NOT exit 1. Only the ALL-empty
  case is exit 1. The §13 acceptance `--shell tcsh` test is the exit-2 representative.

## 6. ZERO new imports — verified

`runCompletion`/`loginShellBase` use only already-imported symbols:
- `os` (Getenv) — imported
- `fmt` (Fprint/Fprintf/Fprintln) — imported
- `io` (Writer) — imported
- `path/filepath` (Base) — imported
- `strings` (ToLower) — imported
- `completionScript` — same package (main)
- `config` — the LOCAL struct type (NOT internal/config; runInit uses the `configpkg` ALIAS for
  internal/config — irrelevant here; runCompletion takes the local `config` value `c`).

`grep` of the import block confirms bufio/_ "embed"/fmt/io/os/path/filepath/strings + internal/*
all present. **Zero new imports for main.go AND for main_test.go** (the dispatch tests use
bytes/strings/testing, already imported; t.Setenv is a testing method).

## 7. The grep substrings the tests + §13-acceptance rely on — VERIFIED present

```
$ grep -c '_skilldozer_completion' completions/skilldozer.bash   → 2   (§13 acceptance grep)
$ grep -c 'complete -c skilldozer'  completions/skilldozer.fish   → 14  (§13 acceptance grep)
$ grep -c '#compdef skilldozer'     completions/_skilldozer       → 1   (zsh-unique, for the env/login tests)
```
These three substrings are what the dispatch tests assert `strings.Contains(out.String(), …)`.

## 8. The test matrix (test_patterns.md §'Completion subcommand — dispatch' + §13)

| # | Test | Inputs | Assert |
|---|------|--------|--------|
| 1 | bash script | `run(["completion","--shell","bash"])` | code 0; stdout Contains `_skilldozer_completion`; stderr empty |
| 2 | fish script | `run(["completion","--shell","fish"])` | code 0; stdout Contains `complete -c skilldozer`; stderr empty |
| 3 | unsupported shell | `run(["completion","--shell","tcsh"])` | code 2; stdout EMPTY; stderr mentions "tcsh" |
| 4 | undetectable | `run(["completion"])` with SKILLDOZER_SHELL="" + SHELL="" | code 1; stdout EMPTY; stderr Contains "shell" |
| 5 | env shell wins | `run(["completion"])` with SKILLDOZER_SHELL="zsh" | code 0; stdout Contains `#compdef skilldozer` |
| 6 | login shell basename | `run(["completion"])` with SHELL="/bin/zsh", SKILLDOZER_SHELL="" | code 0; stdout Contains `#compdef skilldozer` |
| 7 | detectShell unit | table: ("bash","","")→("bash",t); ("","fish","")→("fish",t); ("","","zsh")→("zsh",t); ("","","")→("",f); ("bash","fish","zsh")→("bash",t) | pure, no env |
| 8 | loginShellBase unit | table: SHELL="/bin/zsh"→"zsh"; SHELL=""→""; SHELL="/usr/bin/fish"→"fish"; SHELL="/bin/ZSH"→"zsh" (lowercased) | t.Setenv |

- Test #4 (undetectable) MUST suppress the test runner's own $SHELL — set BOTH `SKILLDOZER_SHELL=""`
  AND `SHELL=""` via t.Setenv. os.Getenv returns "" for unset-or-empty, so t.Setenv("","") is
  equivalent to the §13 `env -u` for loginShellBase's `s == ""` check.
- Test #5: even if the runner's SHELL is /bin/bash, detectShell checks envShell (SKILLDOZER_SHELL)
  BEFORE loginShell (SHELL), so SKILLDOZER_SHELL=zsh wins. (Still set SKILLDOZER_SHELL explicitly.)

## 9. Sibling boundaries (do NOT cross)

- **P1.M2.T1.S1** (parsing — COMPLETE): provides c.completion/c.completionShell + exclusivity.
  Consumed, NOT modified.
- **P1.M2.T2.S1** (embed — PARALLEL): provides completionScript + the 3 embed vars. Consumed, NOT
  modified. Its code region (import block, after-var-version, completionScript @1110) is DISJOINT
  from my dispatch slot (run() ~535) + tail append (after runInit ~1204).
- **P1.M3.T1.S1**: edits completions/* (adds `completion` as a completable subcommand) + rebuilds.
  My subtask does NOT touch completions/*. (When P1.M3.T1.S1 lands, the embedded bytes change but
  completionScript keeps returning them, and the grep substrings still match.)
- **run() / exclusivityError**: I add ONE dispatch line to run() (the `if c.completion` slot). I do
  NOT modify exclusivityError (already done by P1.M2.T1.S1) or any other run() branch.

## 10. Why no external/online research is needed

This subtask is stdlib string/env handling (`os.Getenv`, `filepath.Base`, `strings.ToLower`,
`fmt.Fprint`) + the repo's own `completionScript` (S1) + the local `config` struct. The
detectShell/loginShellBase designs are FIXED VERBATIM by external_deps.md §'Shell detection'
(which verified the filepath.Base idiom). The exit-code ladder is fixed by PRD §6.4/§14.6. The
test matrix is fixed by test_patterns.md. Direct source reads (this file) are higher-fidelity
than any subagent summary; spawning online-research subagents for os.Getenv/filepath.Base would be
wasteful theater.
