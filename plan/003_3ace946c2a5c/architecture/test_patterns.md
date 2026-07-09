# Test Patterns — Delta 003

## Tests that BREAK and must be rewritten

### `main_test.go:280 TestRunDefaultNoArgs`
```go
// CURRENT (asserts stderr/exit 1):
func TestRunDefaultNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(nil, &out, &errOut)
	if code != 1 { t.Errorf("run(nil): code=%d; want 1", code) }
	if out.Len() != 0 { t.Errorf("run(nil) stdout=%q; want EMPTY", out.String()) }
	if !strings.Contains(errOut.String(), "USAGE") { t.Errorf("want USAGE block on stderr") }
}
// TARGET (asserts stdout/exit 0):
//   code == 0, out.String() contains "USAGE", errOut.Len() == 0
```

### `main_test.go:1670 TestRunModifiersOnlyNoMode`
```go
// CURRENT: run([]string{"--no-color"}, ...) → code 1, stderr has USAGE, stdout empty
// TARGET: run([]string{"--no-color"}, ...) → code 0, stdout has USAGE, stderr empty
```

## New tests to write (mirror patterns)

### No-args flip (§13 "Grepability contract")
- `run(nil)` → code 0, stdout contains USAGE, stderr empty
- `run([]string{"--no-color"})` → code 0, stdout contains USAGE, stderr empty
- §13 assertion: `test -z "$(./skilldozer 2>&1 >/dev/null)"` (no-args writes NOTHING to stderr)

### Completion subcommand — parsing
- `completion` sets `c.completion=true`, no tags
- `completion --shell bash` / `completion --shell=bash` sets `completionShell="bash"`
- `completion example` (tag) → exclusivity exit 2
- `completion --list` → exclusivity exit 2
- `check completion` → exclusivity exit 2

### Completion subcommand — dispatch + exit codes
- `completion --shell bash` → stdout contains `_skilldozer_completion`, exit 0
- `completion --shell fish` → stdout contains `complete -c skilldozer`, exit 0
- `completion --shell tcsh` → stderr + exit 2, stdout empty
- No explicit + no `$SKILLDOZER_SHELL` + no `$SHELL` → stderr mentions "shell", exit 1, stdout empty
- `$SKILLDOZER_SHELL` honored (e.g. `=zsh` → zsh script)
- `basename($SHELL)` honored (e.g. `SHELL=/bin/zsh` → zsh script)

### detectShell unit test (no env mutation)
- `detectShell("bash", "", "")` → `("bash", true)`
- `detectShell("", "fish", "")` → `("fish", true)`
- `detectShell("", "", "zsh")` → `("zsh", true)`
- `detectShell("", "", "")` → `("", false)`
- `detectShell("bash", "fish", "zsh")` → `("bash", true)` (explicit wins)

### Regression guard (must stay green)
- `--help`/`-h` → stdout, exit 0, wins over everything
- `./skilldozer nope` → empty stdout, exit 1 (unchanged §6.4 contract)
- All existing exclusivity tests (check+tags, init+tags, mode+mode) unchanged
