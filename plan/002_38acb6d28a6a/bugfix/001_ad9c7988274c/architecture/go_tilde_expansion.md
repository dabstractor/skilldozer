# Go tilde (~) expansion — research brief (Issue 5)

Applies to user-typed paths in `init <dir>`, `--store <dir>`, `--store=<dir>`, and the
interactive prompt. All flow through `resolveStore` → `filepath.Abs` (`main.go:886`).

## Key findings

1. **No stdlib tilde function exists.** `os.Expand`/`os.ExpandEnv` handle only `$VAR`/`${VAR}`,
   NOT `~`. There is no `os.ExpandHome` / `filepath.ExpandHome`. This is a well-known Go gap.
2. **Canonical pattern:** `os.UserHomeDir()` (reads `$HOME` on Unix / `%USERPROFILE%` on
   Windows, cgo-free, since Go 1.12) + `strings.HasPrefix(p, "~/")` prefix check + bare `"~"`
   special case.
3. **`filepath.Abs` does NOT expand `~`** — it only joins a relative path onto cwd, so
   `~/x` wrongly becomes `<cwd>/~/x`. Expansion MUST run BEFORE `filepath.Abs`.
4. **Edge cases — expand ONLY `~` and `~/...` for the current user:**
   - `~/myskills` → `$HOME/myskills`
   - `~/` → `$HOME/`
   - `~` → `$HOME`
   - `~user`, `~foo`, `~foo/bar`, `~~/weird` → UNCHANGED (second char not `/`; other-user
     expansion needs cgo/os/user — out of scope, PRD never requires it)
   - ``, `foo/bar`, `/abs/path` → UNCHANGED
   - Guard MUST be `strings.HasPrefix(p, "~/")` (not `HasPrefix(p, "~")`).
5. **`os.UserHomeDir()` error handling:** returns `("", error)` when `$HOME` unset. Docs say
   the caller "may choose to ignore the error." Idiomatic CLI behavior: return the path
   UNCHANGED (fail safe; never crash or emit empty string). Returning `""` would make a later
   `filepath.Abs("")` yield cwd, masking the problem.

## Recommended helper (place in main.go near resolveStore; stdlib-only, ~20 lines)

Reuses the `os.UserHomeDir` pattern already in `internal/config/config.go:154` (DefaultStore)
so `~` semantics are consistent across the binary. main.go already imports `os`, `strings`,
`path/filepath` — no new imports.

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

Call site change in `resolveStore` (`main.go:886`):
```go
// before:  abs, err := filepath.Abs(store)
// after:   expand ~ first, then absolutize
store = expandHome(store)
abs, err := filepath.Abs(store)
```

## Recommended tests (table-driven; HOME via t.Setenv)

```go
func TestExpandHome(t *testing.T) {
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
	t.Setenv("HOME", "")
	for _, in := range []string{"~/myskills", "~", "~/"} {
		if got := expandHome(in); got != in {
			t.Errorf("with no HOME, expandHome(%q) = %q; want unchanged", in, got)
		}
	}
}
```

Note: `~/` expands to `/home/testuser` because `filepath.Join(home, "")` cleans the trailing
slash. That is acceptable (a trailing slash is not semantically significant for a store dir).

## Sources
- [`os.UserHomeDir`](https://pkg.go.dev/os#UserHomeDir) — home-dir primitive; "caller may
  choose to ignore the error" authorizes the fail-safe fallback.
- [`os.Expand`](https://pkg.go.dev/os#Expand) / [`os.ExpandEnv`](https://pkg.go.dev/os#ExpandEnv)
  — proof they expand only `$VAR`/`${VAR}` (the no-stdlib-tilde gap).
- [`path/filepath.Abs`](https://pkg.go.dev/path/filepath#Abs) — joins onto cwd, no `~` (justifies
  expand-before-Abs ordering).
- [`strings.HasPrefix`](https://pkg.go.dev/strings#HasPrefix) — the prefix primitive.
- Excluded (PRD §17: stdlib-only besides yaml.v3): `mitchellh/go-homedir`, `golang.org/x/term`,
  `os/user` (cgo). The codebase's `stdinIsTerminal` comment already documents avoiding x/term.
