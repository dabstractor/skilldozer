# Research: Idiomatic Go tilde (`~`) expansion for CLI paths

> Scope: a `skilldozer`-style Go CLI (Go 1.25, module `github.com/dabstractor/skilldozer`)
> whose **only** third-party dependency is `gopkg.in/yaml.v3`. The recommendation must be
> **stdlib-only** (no `mitchellh/go-homedir`, no `golang.org/x/sys`, no `os/user`).
>
> Applies to user-typed paths in this codebase: `init <dir>`, `--store <dir>`,
> `--store=<dir>`, and the `SKILLDOZER_SKILLS_DIR` env value — i.e. the strings that today
> flow straight into `filepath.Abs` (see `resolveStore` in `main.go`). Tilde expansion must
> run **before** `filepath.Abs`.

---

## Summary

The Go standard library has **no** tilde-expansion function: there is no `os.ExpandHome`,
`filepath.ExpandHome`, or anything like it. (`os.Expand` / `os.ExpandEnv` expand only
`$VAR` / `${VAR}` environment-variable references — **not** `~`.) The canonical,
stdlib-only pattern is a ~15-line helper that pairs `os.UserHomeDir()` (which reads `$HOME`
on Unix / `%USERPROFILE%` on Windows) with a `strings.HasPrefix(p, "~/")` prefix check,
plus a special case for a bare `"~"`. It expands **only** `~` and `~/...` for the current
user; `~user`, `~foo`, and `~foo/bar` are left untouched (they are relative paths, not home
references). `filepath.Abs` does **not** expand tildes — it only joins a relative path onto
the current working directory, so `~/x` would wrongly become `<cwd>/~/x` unless you expand
first. If `$HOME` is unset, `os.UserHomeDir()` returns `("", error)`; the official docs
explicitly say the caller "may choose to ignore the error," so the idiomatic CLI behavior is
to **leave the path unchanged** on that error (fail safe, never crash).

---

## Findings

### 1. There is NO stdlib tilde-expansion function. (Confirmed gap.)

Go provides `os.Expand` and `os.ExpandEnv`, but they handle **only** shell-style variable
substitution — `${var}` and `$var` — via a caller-supplied mapping (or the live environment
for `ExpandEnv`). Neither recognizes `~`. There is no `os.ExpandHome`, no
`filepath.ExpandHome`, and no tilde handling in `path/filepath` at all. This is a
long-standing, well-known gap that every Go CLI ends up filling itself.

```go
// os package
func Expand(s string, mapping func(string) string) string   // $VAR / ${VAR} only
func ExpandEnv(s string) string                             // $VAR / ${VAR} from env only
// (no ExpandHome / HomeExpand / similar exists)
```
— [`os` package docs](https://pkg.go.dev/os#Expand) · [`os.ExpandEnv`](https://pkg.go.dev/os#ExpandEnv)

### 2. The canonical manual pattern: `os.UserHomeDir()` + `strings.HasPrefix(p, "~/")`.

`os.UserHomeDir()` (since Go 1.12) returns the current user's home directory with no cgo
and no `/etc/passwd` lookup:

- **Unix (incl. macOS):** the `$HOME` environment variable.
- **Windows:** `%USERPROFILE%`.
- **Plan 9:** `$home`.
- If the expected variable is unset, it returns `("", error)`.

Because it is purely env-based, it is fast, cgo-free, and matches exactly what a shell does
for `~` (shells also expand `~` from `$HOME`). This is why it — and not `os/user.Current()`
(which can require cgo and a passwd lookup) — is the right primitive for a CLI. The prefix
swap is done with `strings.HasPrefix`:

```go
home, err := os.UserHomeDir()
// ~/x  -> home + p[1:]   (p[1:] keeps the leading '/', so "home" + "/x")
// ~    -> home
```
— [`os.UserHomeDir`](https://pkg.go.dev/os#UserHomeDir) · [`strings.HasPrefix`](https://pkg.go.dev/strings#HasPrefix)

### 3. Edge cases: `~user`, `~foo`, `~foo/bar` are NOT expanded.

A simple CLI should expand **only** the current user's home: the bare `"~"` and the `"~/"`
prefix. Everything else starting with `~` is left untouched:

| Input        | Output              | Why                                                              |
|--------------|---------------------|------------------------------------------------------------------|
| `~/myskills` | `$HOME/myskills`    | home-prefix case                                                 |
| `~/`         | `$HOME/`            | home-prefix case (empty remainder)                               |
| `~`          | `$HOME`             | bare-tilde case                                                  |
| `` (empty)   | ``                  | passed through                                                   |
| `~user`      | `~user` (unchanged) | `~+name` is a different user's home; not handled by simple CLIs  |
| `~foo`       | `~foo` (unchanged)  | `~` not followed by `/` ⇒ a relative path, not a home reference  |
| `~foo/bar`   | `~foo/bar`          | same: second char is not `/`                                     |
| `foo/bar`    | `foo/bar`           | no tilde                                                         |
| `/abs/path`  | `/abs/path`         | already absolute                                                 |
| `~~/weird`   | `~~/weird`          | second char is `~`, not `/`                                      |

Key correctness point: the guard must be `strings.HasPrefix(p, "~/")` — i.e. the `~` must be
immediately followed by a `/` (or be exactly `"~"`). Do **not** write `strings.HasPrefix(p,
"~")`, which would wrongly rewrite `~foo` and `~user`. Full POSIX `~user` expansion (looking
up another user's home via `getpwnam`) is deliberately out of scope — it needs cgo/`os/user`
and is unnecessary for a single-user CLI that only cares about the invoker's own `~`.

### 4. `filepath.Abs` does NOT expand tildes.

`filepath.Abs` only converts a path to an absolute one by joining it with the **current
working directory** when it is not already absolute:

> "Abs returns an absolute representation of path. If the path is not absolute it will be
> joined with the current working directory to turn it into an absolute path."

It has no concept of `~`. Consequence: if you call `filepath.Abs("~/myskills")` directly, you
get `<cwd>/~/myskills` (a literal directory literally named `~`), **not**
`$HOME/myskills`. Therefore tilde expansion **must run before** `filepath.Abs`. In this
codebase, `resolveStore` calls `filepath.Abs(store)` on the `init`/`--store` value and on the
`config.File.Store` string — so the expand step must precede that call.
— [`path/filepath.Abs`](https://pkg.go.dev/path/filepath#Abs)

### 5. `os.UserHomeDir()` error handling: fall back to the path unchanged.

`os.UserHomeDir()` returns `("", error)` when `$HOME` (Unix) / `%USERPROFILE%` (Windows) is
unset. The package docs explicitly bless ignoring it:

> "If the expected variable is not set in the environment, UserHomeDir returns both an error
> and the empty string, so the caller may choose to ignore the error."

For a CLI, the right behavior on that error is to **return the path unchanged** (fail safe).
A user typing `~/myskills` with no `$HOME` is an exotic, near-unreal scenario, and silently
leaving the literal path — rather than crashing, erroring, or producing an empty string — is
the least-surprising, non-destructive choice. (If you instead returned `""` on error, a later
`filepath.Abs("")` would yield the cwd, masking the problem; returning the unchanged `~/...`
keeps the problem visible and honest.)

---

## Recommended helper: `expandTilde`

Drop this into the package that owns path normalization (e.g. `internal/skillsdir` or a new
`internal/paths`, alongside the existing `filepath.Abs` usage in `resolveStore`). It is
stdlib-only (`os` + `strings`), cgo-free, and ~20 lines including the doc comment.

```go
package paths

import (
	"os"
	"strings"
)

// expandTilde expands a leading '~' to the current user's home directory.
//
//   "~/foo"        -> $HOME/foo
//   "~/"           -> $HOME/
//   "~"            -> $HOME
//   "~user"        -> returned UNCHANGED (other-user expansion is intentionally
//                     out of scope; it would require os/user/cgo + getpwnam)
//   "~foo"         -> returned UNCHANGED ('~' not followed by '/' is a relative
//                     path, not a home reference)
//   "" / "foo/bar" -> returned UNCHANGED
//
// If the home directory cannot be determined (e.g. $HOME unset), the path is
// returned UNCHANGED — os.UserHomeDir's docs say the caller "may choose to
// ignore the error"; we fail safe rather than crash or emit an empty path.
//
// IMPORTANT: expand tildes BEFORE filepath.Abs. filepath.Abs does not expand
// '~'; on its own it would turn "~/foo" into "<cwd>/~/foo".
func expandTilde(p string) string {
	if p == "" {
		return p
	}
	// Bare "~" alone -> home.
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p // home unavailable: leave as-is
	}
	// "~/..." -> home + the "/..." remainder (p[1:] keeps its leading '/').
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:] // "/home/user" + "/myskills"
		}
		return p // home unavailable: leave as-is
	}
	// "~user", "~foo", "~foo/bar", relative, absolute -> untouched.
	return p
}
```

### Usage in this codebase

Call `expandTilde` immediately before the existing `filepath.Abs` in `resolveStore`
(`main.go`), so user-typed `init`/`--store` values — and any `SKILLDOZER_SKILLS_DIR`-sourced
value — are normalized first:

```go
// in resolveStore, replace:
//   abs, err := filepath.Abs(store)
// with:
abs, err := filepath.Abs(expandTilde(store))
```

(Where exactly to centralize it — a new `internal/paths` package vs. a function inside
`internal/skillsdir` next to `Find` — is an implementation/placement decision for the
parent/worker, intentionally **not** made here. The research only confirms the algorithm and
the before-`filepath.Abs` ordering.)

---

## Recommended tests (table-driven; HOME-dependent cases use `t.Setenv`)

```go
func TestExpandTilde(t *testing.T) {
	t.Setenv("HOME", "/home/testuser") // also sets %USERPROFILE% on Windows via Go's shim
	for _, tc := range []struct{ in, want string }{
		{"~/myskills", "/home/testuser/myskills"},
		{"~/", "/home/testuser/"},
		{"~", "/home/testuser"},
		{"~user", "~user"},        // other user: unchanged
		{"~foo", "~foo"},          // ~ not followed by '/': unchanged
		{"~foo/bar", "~foo/bar"},  // unchanged
		{"~~/weird", "~~/weird"},  // unchanged
		{"", ""},                  // unchanged
		{"foo/bar", "foo/bar"},    // no tilde: unchanged
		{"/abs/path", "/abs/path"},// absolute: unchanged
	} {
		if got := expandTilde(tc.in); got != tc.want {
			t.Errorf("expandTilde(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// When $HOME is unset, every tilde form is returned unchanged (fail-safe).
func TestExpandTilde_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	// On Unix os.UserHomeDir treats an empty $HOME like an unset one (returns error).
	for _, in := range []string{"~/myskills", "~", "~/"} {
		if got := expandTilde(in); got != in {
			t.Errorf("with no HOME, expandTilde(%q) = %q; want it unchanged", in, got)
		}
	}
}
```

> `t.Setenv("HOME", "")` works in the test because `os.UserHomeDir()` reads `HOME` live.
> (Tests using `t.Setenv` cannot run under `t.Parallel()`; the rest of this repo follows the
> no-`t.Parallel()` convention for global-state tests already — see the `isTerminal` note in
> `main.go`.)

---

## Manual verification trace (snippet logic)

The available subagent toolset is `read`/`write`/`contact_supervisor`/`intercom` only — it
includes **no shell execution and no `web_search`**, so the snippet could not be compiled/run
live. Correctness was instead verified by hand-tracing each branch against the documented
Go stdlib semantics above. With `HOME=/home/testuser`:

| Input        | Branch hit                          | Computed result          | Expected |
|--------------|-------------------------------------|--------------------------|----------|
| `~/myskills` | `HasPrefix("~/")` → `home + p[1:]`  | `/home/testuser`+`/myskills` | ✅ |
| `~/`         | `HasPrefix("~/")` → `home + p[1:]`  | `/home/testuser`+`/`     | ✅ |
| `~`          | `p == "~"` → `home`                 | `/home/testuser`         | ✅ |
| `~foo`       | none (2nd char `f`≠`/`, not `~`)    | `~foo` unchanged         | ✅ |
| `~user/x`    | none (2nd char `u`≠`/`)             | `~user/x` unchanged      | ✅ |
| `` / `foo/bar` / `/abs/x` / `~~/weird` | fallthrough | unchanged | ✅ |

With `HOME` unset: `os.UserHomeDir()` → `("", err)`, so the `if ... ; err == nil` guard is
false in both branches → input returned unchanged. ✅

A scratch harness was staged at `/tmp/tildecheck/` (a `main.go` exercising these cases under
`HOME` set vs `env -u HOME`) for the implementing agent to run with `go run`, but it was not
executed by this research subagent due to the tool limitation.

---

## Sources

- **Kept (Go stdlib docs — stable, Go 1 compatibility promise):**
  - [`os.UserHomeDir`](https://pkg.go.dev/os#UserHomeDir) — the home-dir primitive ($HOME /
    %USERPROFILE%); the explicit "caller may choose to ignore the error" sentence authorizes
    the fail-safe fallback (Finding 2 & 5).
  - [`os.Expand`](https://pkg.go.dev/os#Expand) / [`os.ExpandEnv`](https://pkg.go.dev/os#ExpandEnv)
    — proof they expand **only** `$VAR`/`${VAR}`, establishing the no-stdlib-tilde gap
    (Finding 1).
  - [`path/filepath.Abs`](https://pkg.go.dev/path/filepath#Abs) — proof it joins onto cwd
    and does not expand `~`, justifying expand-before-Abs ordering (Finding 4).
  - [`strings.HasPrefix`](https://pkg.go.dev/strings#HasPrefix) — the prefix primitive for
    the canonical pattern (Finding 2).
- **Dropped:** third-party tilde libraries (e.g. `mitchellh/go-homedir`) and
  `golang.org/x/term`/`os/user` — excluded by the hard constraint that the project stay
  stdlib-only besides `gopkg.in/yaml.v3` (also noted in `main.go`'s `stdinIsTerminal`
  comment: "No golang.org/x/term"). They would also pull in cgo / extra deps for no benefit.

## Gaps

- **Live execution not performed.** The snippet was verified by manual trace, not by `go
  run`/`go test`, because this research subagent has no shell-exec or web tool. The logic is
  simple and depends only on stable stdlib behavior, so confidence is high, but the
  implementing agent should run the `/tmp/tildecheck` harness (or the recommended unit tests)
  to confirm. Suggested next step: add `expandTilde` + the two tests to the codebase and run
  `go test ./...`.
- **Windows `%USERPROFILE%` behavior** is asserted from the docs but not exercised on a
  Windows host (same tool limitation). The `t.Setenv("HOME", ...)` test covers Unix; on
  Windows `os.UserHomeDir` reads `USERPROFILE` instead, so the Windows-specific env var
  would need to be set in a Windows test.
- **Placement decision deferred** (new `internal/paths` package vs. a fn in
  `internal/skillsdir` next to `Find`) — left to the parent/worker; this brief only nails the
  algorithm and the before-`filepath.Abs` ordering.
