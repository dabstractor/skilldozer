# Verified facts — P1.M2.T3.S1: `expandHome` helper + table-driven tests (Issue 5)

Scope: **add `func expandHome(p string) string` to `main.go` (adjacent to `resolveStore`) +
2 table-driven tests in `main_test.go`.** This subtask does NOT wire it into `resolveStore`
(that is P1.M2.T3.S2). All facts below were read directly from the current tree
(main.go 1094 lines, main_test.go 2648 lines) on 2026-07-07.

---

## 1. The exact insertion site (anchor by TEXT; contract line numbers are stale)

Contract says "adjacent to resolveStore (main.go:865)". **865 is STALE** — the M1 work and
the in-flight P1.M2.T2.S1 shift things. The CURRENT, verified anchors (anchored by unique
text, not number):

```
main.go:858  func chooseStore(haveStore, cwd string, isTTY bool, defaultStore string, prompt ...) (string, error) {
main.go:887  // resolveStore is the I/O-bearing wrapper around chooseStore that run()'s init
main.go:901  func resolveStore(haveStore string) (string, error) {
main.go:922      abs, err := filepath.Abs(store)
main.go:926      return abs, nil
```

`chooseStore`'s closing `}` is at ~885; the `// resolveStore is the I/O-bearing wrapper`
doc comment begins at **887**. **Insert `expandHome` (helper + its doc comment) immediately
BEFORE that doc comment** — i.e. right after chooseStore's closing brace, so it sits adjacent
to its sole future consumer (`resolveStore`, P1.M2.T3.S2). The unique text to anchor on is the
line starting `// resolveStore is the I/O-bearing wrapper`.

`grep -c 'os.UserHomeDir' main.go` → **0** today. There is NO existing tilde/home logic in
main.go; `expandHome` is genuinely additive (no collision, no rename).

---

## 2. Imports — ZERO new imports (contract LOGIC §3 + decisions.md §D4 confirmed)

**main.go import block (verified):**
```go
import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dabstractor/skilldozer/internal/check"
	configpkg "github.com/dabstractor/skilldozer/internal/config"
	... internal/{discover,resolve,search,skillsdir,ui}
)
```
`expandHome` uses `os.UserHomeDir`, `strings.HasPrefix`, `filepath.Join` — **all already
imported.** No edit to the import block.

**main_test.go import block (verified):** `bytes, errors, io, os, path/filepath, strings,
testing, configpkg`. `TestExpandHome`/`TestExpandHomeNoHomeUnchanged` use only `*testing.T`,
`t.Setenv`, `t.Errorf` — **all already imported.** No edit to the import block.

`go.mod`/`go.sum` are byte-for-byte UNCHANGED (sole dep `gopkg.in/yaml.v3 v3.0.1`, go 1.25).

---

## 3. The helper — verbatim from architecture/go_tilde_expansion.md (the authoritative brief)

```go
// expandHome expands a leading "~" or "~/" to the current user's home directory
// (os.UserHomeDir) so that `init ~/myskills` resolves to $HOME/myskills rather than
// <cwd>/~/myskills. Only the CURRENT user's home is expanded ("~" and "~/..."); "~user"
// / "~foo" are returned unchanged (other-user expansion needs cgo/os/user, out of scope).
// filepath.Abs does NOT expand "~", so this MUST run before filepath.Abs (Issue 5).
// If $HOME is unset, os.UserHomeDir returns an error and the path is returned unchanged
// (fail safe — the docs say the caller may ignore the error).
func expandHome(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
		return p
	}
	return p
}
```

**Behavior table (the contract LOGIC §3, traced):**
| input p            | output                                  | why                                           |
|--------------------|-----------------------------------------|-----------------------------------------------|
| `~`                | `os.UserHomeDir()` (or p if err)        | bare-`~` special case                         |
| `~/myskills`       | `filepath.Join(home, "myskills")`       | `HasPrefix(p,"~/")` → strip 2 → Join          |
| `~/`               | `home` (trailing slash CLEANED by Join) | `filepath.Join(home, "")` cleans → see §6     |
| `~user`,`~foo`,    | UNCHANGED                               | second char ≠ `/`; other-user needs cgo/user  |
| `~foo/bar`,`~~/weird` | UNCHANGED                            | same                                          |
| ``, `foo/bar`,     | UNCHANGED                               | no `~` prefix                                 |
| `/abs/path`        | UNCHANGED                               | absolute, no `~`                              |

The guard is `strings.HasPrefix(p, "~/")` — NOT `HasPrefix(p, "~")` (the latter would
falsely match `~user`).

---

## 4. The tests — verbatim from architecture/go_tilde_expansion.md §"Recommended tests"

```go
func TestExpandHome(t *testing.T) {
	// Do NOT call t.Parallel() — mutates HOME.
	t.Setenv("HOME", "/home/testuser")
	for _, tc := range []struct{ in, want string }{
		{"~/myskills", "/home/testuser/myskills"},
		{"~/", "/home/testuser"},
		{"~", "/home/testuser"},
		{"~user", "~user"},
		{"~foo", "~foo"},
		{"~foo/bar", "~foo/bar"},
		{"~~/weird", "~~/weird"},
		{"", ""},
		{"foo/bar", "foo/bar"},
		{"/abs/path", "/abs/path"},
	} {
		if got := expandHome(tc.in); got != tc.want {
			t.Errorf("expandHome(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestExpandHomeNoHomeUnchanged(t *testing.T) {
	// Do NOT call t.Parallel() — mutates HOME.
	t.Setenv("HOME", "")
	for _, in := range []string{"~/myskills", "~", "~/"} {
		if got := expandHome(in); got != in {
			t.Errorf("with no HOME, expandHome(%q) = %q; want unchanged", in, got)
		}
	}
}
```

**Placement:** after `TestChooseStorePropagatesPromptError` (main_test.go ~2428, the last of
the `TestChooseStore*` store-path-helper family) and before the `// TestSetupStoreEmptyDir...`
comment block. `expandHome` is a pure-ish store-path helper (env-only via HOME), so it groups
with the `TestChooseStore*` family, not the I/O-bearing `TestSetupStore*` tests.

---

## 5. The `os.UserHomeDir` precedent + the no-HOME error behavior is ALREADY tested

**internal/config/config.go:152-159 (`DefaultStore`) — the existing, consistent pattern:**
```go
home, err := os.UserHomeDir()
if err != nil {
	return "", err
}
return filepath.Join(home, ".local", "share", "skilldozer", "skills"), nil
```
`expandHome` reuses `os.UserHomeDir` for cross-binary `~` consistency (decisions.md §D4).

**The no-HOME branch is reliable — proven by internal/config/config_test.go:274-279
(`TestDefaultStoreHomeUnsetErrors`):**
```go
t.Setenv("XDG_DATA_HOME", "") // force the HOME fallback branch
t.Setenv("HOME", "")          // os.UserHomeDir -> error
got, err := DefaultStore()
if err == nil {
	t.Fatalf("DefaultStore() HOME unset: err=nil; want a non-nil error from os.UserHomeDir")
}
```
This existing green test DEPENDS on `t.Setenv("HOME","")` → `os.UserHomeDir()` returning a
non-nil error on Linux (PRD targets Linux; the config test header says so). So
`TestExpandHomeNoHomeUnchanged`'s premise is not hypothetical — it is the same env condition
the suite already exercises. `os.UserHomeDir` is cgo-free, reads `$HOME` on Unix, since Go 1.12.

Note the contract LOGIC §3 asymmetry vs `DefaultStore`: `DefaultStore` PROPAGATES the error
(returns `"", err`); `expandHome` SWALLOWS it (returns p unchanged). That is deliberate and
correct — `expandHome` is a best-effort normalizer that must NEVER crash or emit `""`
(returning `""` would make a later `filepath.Abs("")` yield cwd, masking the problem). The
go_tilde_expansion.md §5 finding + the `os.UserHomeDir` docs ("the caller may choose to
ignore the error") authorize the fail-safe fallback.

---

## 6. GOTCHA — `~/` expands to `home` (trailing slash CLEANED), not `home/`

`filepath.Join(home, "")` calls `filepath.Clean`, which strips a trailing slash. So
`expandHome("~/")` returns `/home/testuser`, NOT `/home/testuser/`. The test table
(`{"~/", "/home/testuser"}`) reflects this. This is ACCEPTABLE — a trailing slash is not
semantically significant for a store directory (the store is later absolutized + created via
`MkdirAll`, which is slash-insensitive). Do NOT "fix" this by using `home + p[1:]` string
concatenation — that would reintroduce `~user` mis-expansion and skip `filepath.Clean`
(canonicalization). `filepath.Join` is the correct, consistent choice (matches `DefaultStore`).

---

## 7. Subtask boundary — S1 = helper + unit tests ONLY; S2 wires it in

This PRP (P1.M2.T3.S1) adds **only** the helper + its table-driven unit tests. It does **NOT**:
- modify `resolveStore`'s body (the `store = expandHome(store)` + ordering is **P1.M2.T3.S2**);
- touch the `filepath.Abs(store)` line at main.go:922 (S2);
- add an integration test through `run()`/`init` (S2 — that needs a pty/TTY fixture, out of
  scope for the pure-helper unit tests here).

Exposing `expandHome` as a **package-level** func (lowercase, same `package main`) makes it
directly unit-testable from `main_test.go` (same package) WITHOUT exporting it on the public
surface — matching how `chooseStore`, `parseArgs`, `resolveStore`, `setupStore` are all tested
in-package today. The contract OUTPUT §4 requires this.

---

## 8. Disjointness from the parallel sibling P1.M2.T2.S1 (no collision)

P1.M2.T2.S1 (Issue 4, `init init`) edits:
- main.go `parseArgs` `case "init":` guard (~277-292) + its comment (~278-285);
- main_test.go `TestParseArgsInitInitCapturedAsTag` (~1396) + `TestRunExclusivityInitInit` (~1944).

This subtask edits:
- main.go: a NEW `expandHome` func inserted ~886 (between `chooseStore`'s close and the
  `resolveStore` doc comment) — nowhere near `parseArgs` (~277);
- main_test.go: 2 NEW tests inserted ~2428 (after the `TestChooseStore*` family) — nowhere
  near the `Init`/`Exclusivity` tests (~1396/1944).

The regions are disjoint in BOTH files; no text-level overlap; they land in either order.
Both are additive (new func / new tests), so neither can break the other's grep anchors.

---

## 9. Sources (already in architecture/go_tilde_expansion.md §"Sources" — transcribed)

- `os.UserHomeDir` — https://pkg.go.dev/os#UserHomeDir — home-dir primitive; "caller may
  choose to ignore the error" authorizes the fail-safe fallback.
- `os.Expand` / `os.ExpandEnv` — https://pkg.go.dev/os#Expand — proof they expand ONLY
  `$VAR`/`${VAR}` (the no-stdlib-tilde gap).
- `path/filepath.Abs` — https://pkg.go.dev/path/filepath#Abs — joins onto cwd, NO `~`
  (justifies the expand-before-Abs ordering that S2 will rely on).
- `strings.HasPrefix` — https://pkg.go.dev/strings#HasPrefix — the prefix primitive.
- EXCLUDED (PRD §17: stdlib-only besides yaml.v3): `mitchellh/go-homedir`,
  `golang.org/x/term`, `os/user` (cgo).
