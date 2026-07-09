# Code Change Map — Delta 003

Exact line numbers and mirror patterns for each change site. All verified against the live codebase at HEAD (`bbd4e74`).

---

## Change A: No-args implicit help flip

### Site 1: `main.go:696-700` — the fallthrough (THE FLIP)
```go
// CURRENT (stderr/exit 1):
	fmt.Fprint(stderr, usage())
	return 1
// TARGET (stdout/exit 0):
	fmt.Fprint(stdout, usage())
	return 0
```

### Site 2: Comments to update (cite §6.3 / decision 17)
- `main.go:51` — usageText doc: "to stderr for the no-args default (exit 1)" → stdout/exit 0
- `main.go:418-421` — run() exit-code doc: remove "no recognized mode (usage to stderr)" from exit-1
- `main.go:695-698` — fallthrough comment: remove "parity with get-server-config.sh" / stderr

### Site 3: Tests to update
- `main_test.go:280 TestRunDefaultNoArgs` — assert code==0, USAGE on **stdout**, stderr **empty**
- `main_test.go:1670 TestRunModifiersOnlyNoMode` — same flip (code==0, stdout has USAGE, stderr empty)

---

## Change B: `completion` subcommand

### B1: config struct — `main.go:128-151`
Add after the `init`/`storeMissingValue` group (~line 142):
```go
	completion      bool   // `skilldozer completion` subcommand (PRD §14.6); exclusive like check/init
	completionShell string // `--shell <bash|zsh|fish>` value; "" ⇒ detect from $SKILLDOZER_SHELL/$SHELL
```

### B2: parseArgs `=`-form switch — `main.go:188-204`
Mirror `--store`'s `=`-form. Add after `case "--store":`:
```go
		case "--shell":
			c.completionShell = val
```
NOTE: do NOT auto-set `c.completion=true` — `--shell` without `completion` is an exclusivity conflict (caught by `exclusivityError`). Decision: bare `--shell` sets only `completionShell`; if no `completion` token follows, exclusivity doesn't catch it but dispatch doesn't trigger it either → it falls through to no-mode. This is the least surprising: `--shell` alone is meaningless and gets implicit help. (Alternative: treat bare `--shell` as unknown flag → exit 2. But the delta_prd doesn't specify this edge; choosing the least disruptive path.)

Actually, re-reading delta_prd M2.T1.S1: "`--shell` seen **without** `completion` as an exclusivity conflict." So add a check in exclusivityError: if `completionShell != "" && !c.completion`, return an error. OR simpler: add `--shell` handling that sets `c.completion = true` when seen (like `--store` implies `init`). The delta_prd says: "Add `--shell` handling in both forms — sets `c.completion=true` and `c.completionShell=<val>`; treat `--shell` seen without `completion` as an exclusivity conflict (it is only meaningful in completion context)."

Wait — re-reading: "sets `c.completion=true` and `c.completionShell=<val>`" — so `--shell` DOES imply `completion` (like `--store` implies `init`). And then "treat `--shell` seen without `completion` as an exclusivity conflict" means if `--shell` sets completion=true but there are ALSO tags or other modes, exclusivity catches it. Got it.

So: `--shell bash` sets both `c.completionShell = "bash"` AND `c.completion = true` (mirrors `--store` setting `c.init = true`).

### B3: parseArgs main token switch — `main.go:220-312`
Add two cases. Mirror `check` for `completion`, mirror `--store` for `--shell`:

```go
		case "completion":
			c.completion = true
		case "--shell":
			if i+1 < len(args) {
				c.completion = true
				c.completionShell = args[i+1]
				i++
			}
			// else: --shell with no value → no-mode fallthrough (implicit help);
			//   PRD §6.4 doesn't specify a missing-value exit code for --shell.
			//   Choosäng least-disruptive: silent no-op (matches --search's
			//   no-value behavior of staying false). Alternative: exit 2 like --store.
```

Actually, the delta_prd does NOT mention a missing-value guard for `--shell` (unlike `--store` which has `storeMissingValue`). And PRD §6.4 is silent. So mirror `--search`'s behavior: if no value follows, `completion` stays false (silent no-op). This is the simplest.

### B4: exclusivityError — `main.go:722-770`
Add completion block after the init block (~line 761), mirroring init:
```go
	if c.completion {
		if hasTags {
			return true, "skilldozer: 'completion' cannot be combined with tag arguments"
		}
		if c.check || c.init || c.list || c.searchMode || c.all || c.path {
			return true, "skilldozer: 'completion' cannot be combined with check/init/--path/--list/--search/--all"
		}
	}
```

### B5: run() dispatch — after init dispatch (~main.go:482)
Insert after `if c.init { return runInit(...) }` block:
```go
	if c.completion {
		return runCompletion(c, stdout, stderr)
	}
```

### B6: `//go:embed` declarations — top of main.go (after imports, before `var version`)
```go
import _ "embed"

//go:embed completions/skilldozer.bash
var bashCompletion string

//go:embed completions/_skilldozer
var zshCompletion string

//go:embed completions/skilldozer.fish
var fishCompletion string
```
**VERIFIED:** `//go:embed completions/_skilldozer` works (explicit file path bypasses `_`/`.` exclusion — confirmed by live test AND Go source analysis).

### B7: New functions — near runInit (~main.go:1057)
```go
func completionScript(shell string) (string, bool) { switch... }
func detectShell(explicit, envShell, loginShell string) (string, bool) { ... }
func runCompletion(c config, stdout, stderr io.Writer) int { ... }
```

### B8: usageText — `main.go:52-100`
Add to USAGE: `skilldozer completion [--shell <name>]`
Add to EXAMPLES: `eval "$(skilldozer completion)"`
Add to OPTIONS: `completion [--shell <name>]   Emit the shell completion script for eval (§14.6)` and `--shell <name>            Force a shell for completion (bash|zsh|fish; else auto-detect)`

---

## Change B9: Three completion files
See `completions_change_map.md` for per-file exact edits.

## Change B10: README
Add `eval "$(skilldozer completion)"` idiom to the Shell completions section (~README.md:94-129).
