# Verified Facts — P1.M1.T3.S1 (`main.go` arg-parse skeleton + `--path` + `--version`)

All facts below were **empirically verified** in the target environment
(`/home/dustin/projects/skpp`, Go 1.26.4, linux/amd64) on 2026-07-06. They are
the load-bearing decisions behind the PRP's design. Cite the §number when
justifying a choice in code comments.

---

## §1. ldflags `-X main.version=...` overrides a `var version = "dev"` package var

**The PRD §12.1 build command injects the version at link time.** Verified that
a `package main` with `var version = "dev"` is overridable by
`go build -ldflags "-X main.version=<value>"`:

```bash
$ cat > main.go <<'EOF'
package main
import ("fmt"; "os")
var version = "dev"
func main() {
    if len(os.Args) > 1 && os.Args[1] == "--version" {
        fmt.Printf("skpp %s\n", version)
    }
}
EOF
$ go build -o app .            && ./app --version | cat -A
skpp dev$                       # default build -> "skpp dev\n"
$ go build -ldflags "-X main.version=1.2.3-abc" -o app . && ./app --version | cat -A
skpp 1.2.3-abc$                 # ldflags override -> "skpp 1.2.3-abc\n"
```

**Critical details** (gotchas):
- `var version = "dev"` MUST be a **package-level** var (not `const`, not local).
  `-X` only rewrites string vars at package scope. A `const` CANNOT be overridden
  (compile error: `-X links to a const`).
- The full `-X` argument is `-X main.version=<value>` — the `main.` prefix is the
  **package path**, NOT the import path `github.com/dabstractor/skpp`. Because
  `main.go` is `package main`, the symbol is `main.version`. (If the var lived in
  a sub-package, it would be `-X github.com/dabstractor/skpp/internal/foo.version`.)
- The value has NO quotes in the ldflags string (the shell strips them; `-X
  main.version=1.2.3` not `-X main.version="1.2.3"`).
- `go test` does NOT pass ldflags, so tests see `version == "dev"`. Tests should
  assert against the `version` var (readable from a `package main` test file) so
  they are robust to future default changes, OR assert the literal `"skpp dev"`.
  Both are acceptable; asserting `"skpp "+version+"\n"` is the most robust.
- `go vet` is clean for this pattern (verified above: `vet OK`).

---

## §2. Exact `--version` output format: `skpp <version>\n` (single line, trailing newline)

PRD §6.1: stdout = `skpp <version>` (single line). Verified that `fmt.Printf("skpp
%s\n", version)` produces exactly `skpp dev\n` — one line, one trailing newline,
no leading/trailing whitespace (`cat -A` shows `skpp dev$`, the `$` = end of
line, nothing after). Use `fmt.Fprintf(stdout, "skpp %s\n", version)`.
`fmt.Println("skpp", version)` would insert a SPACE between args AND a newline,
producing `skpp dev\n` too — but `fmt.Printf` is unambiguous about the exact
bytes. Prefer `Fprintf` for byte-exact stdout that the acceptance gate
(`test "$(./skpp --version)" = "skpp dev"`) depends on.

---

## §3. `--path` prints ONLY the dir to stdout (acceptance gate is byte-exact)

The PRD §13 acceptance gate is:

```bash
test "$(./skpp --path)" = "$PWD/skills"
```

`$(...)` captures stdout; the comparison is **exact** (no trailing source label,
no extra whitespace). So `--path` MUST print **only** the absolute skills dir +
a single newline to stdout. `skillsdir.Find()` returns `(dir, src, err)`; the
`src` (Source) is for debugging/future verbose reporting and MUST NOT be printed
to stdout. Verified `fmt.Fprintln(stdout, dir)` produces exactly `<dir>\n`.

The one-line fix message from `ErrNotFound` goes to **stderr** (so `$(...)` stays
empty on failure — critical for `pi --skill "$(skpp ...)"` per §6.4). Verified:
`fmt.Fprintln(stderr, err)` where `err == skillsdir.ErrNotFound` prints the
verbatim message `could not locate the skills directory: set $SKPP_SKILLS_DIR, cd
into the skpp repo, or reinstall skpp` to stderr.

---

## §4. Rule 2 (sibling-of-binary) wins for a repo-root binary next to `skills/`

The acceptance gate runs `./skpp` (built at repo root via `go build -o skpp .`)
and expects `$PWD/skills`. Verified the resolution path:
`os.Executable()` → absolute path to `./skpp`; `filepath.EvalSymlinks` on a
non-symlink returns the same path; `filepath.Dir` → repo root; `repoDir/skills`
`os.Stat`s as a dir → **rule 2 (SourceSibling) wins** and returns `$PWD/skills`.

**Rule 2 does NOT require a `SKILL.md`** (unlike rule 3, which does via
`hasSkillMD`). Only `os.Stat(candidate).IsDir()` is checked. So a `skills/` dir
containing just a `.gitkeep` is enough for rule 2 to win. This is why the
acceptance gate creates a throwaway `skills/.gitkeep` (the real `skills/example/
SKILL.md` ships in P1.M6.T12; the exact §13 assertion is re-run in P1.M6.T16).

**Throwaway note:** `skills/.gitkeep` is a DEV ARTIFACT for THIS subtask's
smoke test only. It is NOT this subtask's deliverable and should NOT be committed
as the shipped skills tree (P1.M6.T12 owns `skills/example/SKILL.md`). After the
gate passes, `rm -rf skills/` to keep the tree clean; the real tree is created
later. `.gitignore` does NOT ignore `skills/` (verified — it ignores `/skpp`,
`/dist`, `*.test`, `.out`, `.pi-subagents/` only), so an untracked `skills/`
WOULD show in `git status` if left behind.

**Miss-path smoke-test corollary (verified):** because rule 2 resolves
`<dir-of-binary>/skills`, the repo-root `./skpp` with its throwaway `skills/`
sibling will ALWAYS succeed on `--path` (rule 2 wins). So the miss-path smoke
check CANNOT use the repo-root binary — it must build an ISOLATED binary into a
fresh temp dir (no sibling `skills/`), unset `SKPP_SKILLS_DIR`, and `cd` into an
empty temp tree so all three rules miss and `Find()` returns `ErrNotFound`.
Verified: isolated binary + unset env + empty cwd ⇒ exit 1, stdout 0 bytes,
stderr = the full `ErrNotFound` one-line fix. (The unit test
`TestRunPathFailureErrNotFound` is unaffected — the `go test` binary lives in
`/tmp/go-buildXXX` with no sibling `skills/`, so it covers the miss path
regardless.) Also: capture the exit code with `; code=$?` and do NOT run the
miss-path block under `set -e` (the binary deliberately exits 1).

---

## §5. Custom arg parser over Go's `flag` package (extensibility for the §6 matrix)

Go's stdlib `flag` package is a **poor fit** for PRD §6's full flag matrix:
- It has no native long+short aliasing (`-v` and `--version` must be registered
  as two separate flags pointing at the same `bool` var; and it treats `-version`
  and `--version` as the same flag, which is fine, but the short-form dance is
  awkward).
- It cannot represent the `check` subcommand (a bare token with NO dashes) or
  positional `<tag>` args interleaved with flags.
- It makes the §6.3 mutual-exclusivity rules (tag + `--list`/`--search`/`--all`
  ⇒ exit 2) clumsy to express.
- It greedily errors on unknown flags — but THIS subtask must TOLERATE unknown
  flags (exit-2 is deferred to P1.M5.T11).

A **hand-rolled switch-based parser** is ~15 lines, trivially extensible (M5
appends `case` entries), handles any-order + long/short aliases naturally, and
degrades gracefully (unknown tokens are a no-op until M5). This is the structure
the item description explicitly recommends ("a small custom parser may be cleaner
for the full §6 matrix — keep it extensible for M5"). **Decision: custom parser.**

Verified the switch matches both forms:
```go
switch a {
case "--version", "-v": cfg.version = true
case "--path", "-p":    cfg.path = true
}
```
A `range` over `os.Args[1:]` visits tokens in input order, so flags appearing in
ANY order are all recognized (PRD §6: "flags can appear in any order").

---

## §6. Testable CLI via `run(args, stdout, stderr io.Writer) int` + `os.Exit`

`os.Exit` cannot be unit-tested (it kills the process) and skips deferred
cleanup. The idiomatic testable-CLI pattern (used broadly in Go CLIs):

```go
func main() {
    os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
    // ... parse + dispatch; return exit code; write via stdout/stderr ...
}
```

Tests call `run([]string{"--version"}, &out, &err)` with `*bytes.Buffer` writers
and assert on the captured output + return code. **Dependency injection of the
writers is essential** — without it, `--path`/`--version` would write to the real
`os.Stdout`/`os.Stderr` and tests could not capture them cleanly. Verified this
compiles and vets clean with the ldflags scratch program.

---

## §7. `skillsdir.Find()` testability for the `--path` tests

`main_test.go` must exercise both `--path` success and error. `skillsdir.Find()`
relies on `os.Getenv(SKPP_SKILLS_DIR)`, `os.Executable`, and `os.Getwd`. How to
control each deterministically (facts proven in P1.M1.T2.S3 research):

- **Success path:** `t.Setenv("SKPP_SKILLS_DIR", tmpDir)` makes rule 1 win
  deterministically and return `tmpDir` (cleaned via `filepath.Abs`). No need to
  touch cwd or the binary.
- **Error path:** `t.Setenv("SKPP_SKILLS_DIR", "")` is NOT enough — an empty
  value makes rule 1 miss, but rule 2 (`findSibling`) inspects the TEST BINARY's
  own location and rule 3 (`findWalkUp`) ascends from cwd. To force ALL THREE to
  miss deterministically: `unsetEnvVar` (rule 1 miss) + `t.Chdir(t.TempDir())`
  into an empty temp tree (rule 3 ascends to `/`, no `skills/` → miss). Rule 2
  DETERMINISTICALLY MISSES inside `go test` because the test binary lives in
  `/tmp/go-buildXXX` with no sibling `skills/` (verified in S3 §7). So `Find()`
  returns `ErrNotFound` deterministically.
- `t.Setenv` and `t.Chdir` both forbid `t.Parallel()` — do NOT add it to any
  main test that touches env or cwd.

---

## §8. `package main` white-box test (`main_test.go`)

The skillsdir package uses white-box tests (`package skillsdir` in
`skillsdir_test.go`) to access unexported symbols. `main_test.go` follows the
same convention: declare `package main` (NOT `package main_test`) so the test can
read the `version` var and call unexported `parseArgs`/`run` directly. Verified
this is the existing repo convention (see `internal/skillsdir/skillsdir_test.go`:
`package skillsdir`).

---

## §9. Imports for `main.go` — `fmt`, `io`, `os` only (no new deps)

`main.go` needs:
- `"fmt"` — `Fprintf`/`Fprintln` for stdout/stderr.
- `"io"` — the `io.Writer` type for the injected writers in `run`.
- `"os"` — `os.Args`, `os.Stdout`, `os.Stderr`, `os.Exit`.
- `"github.com/dabstractor/skpp/internal/skillsdir"` — `Find()`, `ErrNotFound`.

No `flag`, no external deps. `go.mod`/`go.sum` UNCHANGED (skillsdir is already a
transitive dep of the module; yaml.v3 stays `// indirect`). Verified: building a
scratch main that imports an internal package works with `go build .` from the
module root.

---

## §10. Exit code semantics for THIS subtask (minimal, forward-compatible)

PRD §6 exit-code matrix (full):
- `0` success; `1` unresolved tag / no skills / skills dir unresolvable;
  `2` unknown flag / mutually-exclusive modes mixed.

THIS subtask implements ONLY `--version` (exit 0) and `--path` (exit 0 on
success, exit 1 on Find error). Everything else is deferred:
- **No-args / unknown flag / subcommand / positional:** for now, fall through to
  a default that returns **exit 1** (matches the eventual §6.3 no-args exit code)
  WITHOUT printing usage text (usage is P1.M5.T11). Returning 1 — not 2 — keeps
  "unknown flags tolerated" (the exit-2 unknown-flag error is M5).
- **`--help`/`-h`:** NOT implemented (P1.M5.T11). It currently falls through to
  the default (exit 1). The precedence tier where help slots in (same as
  version, per §6.3) is marked with a comment so M5 adds it trivially.

Verified the precedence rule: `--version` is checked before `--path`, so
`skpp --version --path` prints the version and exits 0 (version wins per §6.3).
