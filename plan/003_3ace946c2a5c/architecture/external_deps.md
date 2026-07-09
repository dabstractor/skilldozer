# External Dependencies & Technical Verification — Delta 003

## Go `embed` package (stdlib — NO new dependency)

### Embedding `_skilldozer` (the critical subtlety)

**VERIFIED by live test + Go source analysis:** `//go:embed completions/_skilldozer` works WITHOUT `all:` prefix.

The `_`/`.` exclusion in Go's embed package applies **only to directory-walk patterns**, not to explicit single-file paths. From `src/cmd/go/internal/load/pkg.go` (`resolveEmbed`):
- Regular file match → only checks `isBadEmbedName()`, which passes `_skilldozer` (not `.git`/`.hg`/etc.)
- Directory match → walks tree, skipping `_`/`.`-prefixed children (unless `all:`)

Live test confirmed: three `//go:embed` + three `string` vars compiles and runs correctly.

### Recommended pattern (three `string` vars)

```go
import _ "embed"

//go:embed completions/skilldozer.bash
var bashCompletion string

//go:embed completions/_skilldozer
var zshCompletion string

//go:embed completions/skilldozer.fish
var fishCompletion string

func completionScript(shell string) (string, bool) {
	switch shell {
	case "bash": return bashCompletion, true
	case "zsh":  return zshCompletion, true
	case "fish": return fishCompletion, true
	}
	return "", false
}
```

This gives compile-time-checked, zero-cost string globals + a trivial switch. No runtime `ReadFile`/error handling needed.

### Why NOT `embed.FS`
An `embed.FS` over `all:completions` would also work but adds runtime lookups + error handling for no benefit with 3 known static files. Three `string` vars is simpler and type-safe.

## Shell detection

`filepath.Base(os.Getenv("SHELL"))` is the correct idiom:
- `/bin/zsh` → `zsh`, `/usr/bin/fish` → `fish`, `/bin/bash` → `bash`
- Guard empty: `filepath.Base("")` returns `"."` → must check `os.Getenv("SHELL") == ""` first
- `os.Executable()` is NOT used here (that returns the skilldozer binary path, not the shell)
- The detection order is: explicit `--shell` → `$SKILLDOZER_SHELL` → `basename($SHELL)` → none

### Testable detection core
```go
func detectShell(explicit, envShell, loginShell string) (string, bool) {
	if explicit != "" { return explicit, true }
	if envShell != "" { return envShell, true }
	if loginShell != "" { return loginShell, true }
	return "", false
}
```
Called as `detectShell(c.completionShell, os.Getenv("SKILLDOZER_SHELL"), loginShellBase())` where `loginShellBase()` guards the empty case. This makes detection unit-testable without env mutation.

## Existing dependencies (unchanged)
- `gopkg.in/yaml.v3 v3.0.1` — the ONLY non-stdlib dependency (frontmatter parsing). No change needed.
- Go 1.25 (go.mod) — `embed` has been stable since Go 1.16.

## No new dependencies introduced by this delta
- `embed` → stdlib
- `os`, `path/filepath`, `strings`, `fmt` → stdlib (all already imported)
