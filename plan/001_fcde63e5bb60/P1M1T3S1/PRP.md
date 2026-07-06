# PRP — P1.M1.T3.S1: `main.go` — arg-parse skeleton + `--path` + `--version` + exit codes

> **Subtask:** P1.M1.T3.S1 — the FIRST subtask of T3 (the `main.go` entrypoint).
> **Scope:** create `main.go` (`package main`) and its white-box test
> `main_test.go`. Declare a build-time-overridable `var version = "dev"`; build a
> small extensible arg parser; wire TWO flags only — `--version`/`-v` (print
> `skpp <version>` to stdout, exit 0) and `--path`/`-p` (call `skillsdir.Find()`;
> print the resolved dir to stdout + exit 0, or print the error to stderr + exit
> 1). `--version` takes precedence over everything. Unknown flags are TOLERATED
> for now (the exit-2 matrix is P1.M5.T11). This establishes the dispatch
> structure that M2 (`--list`), M3 (`<tag>`/`--all`), M4 (`--search`/`check`),
> and M5 (`--help`/exit codes) extend.
>
> **DEPENDENCY (CONTRACT):** P1.M1.T2.S3 is assumed LANDED. Its public API is
> consumed verbatim — `skillsdir.Find() (dir string, src Source, err error)`,
> `skillsdir.ErrNotFound`, `skillsdir.Source` (with `.String()`). The S3 code is
> ALREADY on disk in `internal/skillsdir/skillsdir.go` (read in full; the file
> contains `Find`, `ErrNotFound`, `Source`, `SourceEnv`/`SourceSibling`/
> `SourceWalkUp`, and `(Source).String()` returning `"SKPP_SKILLS_DIR"` /
> `"sibling of binary"` / `"ancestor of cwd"`).
>
> **PARALLEL CONTEXT:** This PRP is authored while S3 is being implemented. S3's
> PRP and the on-disk `skillsdir.go` are treated as the contract. Do NOT modify
> `internal/skillsdir/` — only CONSUME it.

---

## Goal

**Feature Goal**: Create the `skpp` CLI entrypoint (`main.go`) with a
build-time-overridable version string and an extensible arg parser that proves
end-to-end skills-dir resolution works: `./skpp --version` prints `skpp <version>`
and exits 0; `./skpp --path` prints the absolute skills dir (resolved via
`skillsdir.Find()`) and exits 0, or prints the one-line-fix error to stderr and
exits 1. The arg-parser structure (a token switch + a precedence-aware `run`
dispatcher) is the skeleton every later flag plugs into.

**Deliverable**: Two NEW files at the repo root (no other files touched):
1. `main.go` — `package main`; `var version = "dev"`; `parseArgs(args)`; `run(args,
   stdout, stderr io.Writer) int`; `main()`.
2. `main_test.go` — `package main` (white-box); tests for `--version`/`-v`
   (incl. precedence over `--path`), `--path`/`-p` success + error, and the
   default no-args/unknown case.

**Success Definition**: `go build -o skpp .` exits 0 and produces a `./skpp`
binary; `gofmt -l main.go main_test.go` is silent; `go vet .` is clean;
`go test . -v` passes; and the PRD §13 acceptance gates pass:
- `test "$(./skpp --version)" = "skpp dev"` (default build, no ldflags).
- `test "$(./skpp --path)" = "$PWD/skills"` (after a throwaway `skills/` dir
  exists so rule 2 wins).
- ldflags override: `go build -ldflags "-X main.version=v0.0.0-test" -o skpp .`
  ⇒ `./skpp --version` prints `skpp v0.0.0-test`.
`go.mod`/`go.sum`/`PRD.md`/`internal/skillsdir/*` unchanged. No new packages
(`discover`/`resolve`/`ui`) created — those are later milestones.

---

## Why

- This subtask **proves the §8 location resolution actually works end-to-end**
  through a real binary. Until `main.go` exists and `./skpp --path` prints the
  right dir, the skillsdir package is just a library with no user-facing proof.
- It **locks the version-injection mechanism** (`var version = "dev"` + `-ldflags
  "-X main.version=..."`) that PRD §12.1's `install.sh` build command depends on.
  Getting this wrong now (e.g. a `const`, or a local var) silently breaks
  `install.sh` in P1.M6.T13. Verified the mechanism (research §1).
- It **establishes the dispatch skeleton** M2–M5 extend. A clean `parseArgs` +
  precedence-aware `run` means each later subtask is a localized edit (add a
  `case`, add a branch), not a rewrite. The item explicitly calls for this
  ("Establishes the arg-parser structure that M2-M5 extend").
- It makes `--path`/`--version` available for the rest of development: every
  later subtask's developer can run `./skpp --path` to debug discovery.

---

## What

A `package main` with:
1. `var version = "dev"` — package-level string var (NOT const, NOT local) so
   `-ldflags "-X main.version=<v>"` overrides it at link time (PRD §12.1).
2. `type config struct { version bool; path bool }` — the parsed-flags struct.
   Grown by later milestones (list/all/search/file/noColor/relative/help/tags).
3. `func parseArgs(args []string) config` — a `range`+`switch` over tokens. For
   THIS subtask: `--version`/`-v` → `cfg.version=true`; `--path`/`-p` →
   `cfg.path=true`; any other token → tolerated no-op (M5 turns unknown flags
   into exit 2 and handles subcommands/positionals). Handles flags in any order.
4. `func run(args []string, stdout, stderr io.Writer) int` — the testable
   dispatcher: precedence `--version` first (exit 0, print `skpp <version>\n`),
   then `--path` (`skillsdir.Find()`; success → `Fprintln(stdout, dir)`, exit 0;
   err → `Fprintln(stderr, err)`, exit 1), else default exit 1 (no usage text
   yet — M5 owns §6.3 no-args usage + §6.2 exit-2).
5. `func main()` — `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`.

### Success Criteria

- [ ] `var version = "dev"` is a **package-level var** (not const/local) so `-ldflags -X main.version=...` overrides it
- [ ] `./skpp --version` and `./skpp -v` both print exactly `skpp <version>\n` to stdout and exit 0
- [ ] `./skpp --version` (default build) prints exactly `skpp dev`
- [ ] ldflags build (`-X main.version=v0.0.0-test`) makes `--version` print `skpp v0.0.0-test`
- [ ] `./skpp --path` and `./skpp -p` print the resolved skills dir (single line) to stdout, exit 0, when `skillsdir.Find()` succeeds
- [ ] `./skpp --path` prints the `Find()` error (the `ErrNotFound` one-line fix) to **stderr**, prints **nothing** to stdout, exit 1, when `Find()` fails
- [ ] `--version` takes precedence: `./skpp --version --path` prints version, exits 0 (never calls `Find()`)
- [ ] Unknown flags / no args are tolerated: do not exit 2 (exit-2 deferred to M5); default to exit 1
- [ ] `--path` stdout is byte-exact: `test "$(./skpp --path)" = "$PWD/skills"` passes
- [ ] `go build -o skpp .` exits 0; `gofmt -l main.go main_test.go` silent; `go vet .` clean; `go test . -v` passes
- [ ] `go.mod`/`go.sum`/`PRD.md`/`internal/skillsdir/*` unchanged; no `discover`/`resolve`/`ui`/`install.sh`/`README` created

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for every symbol in `main.go` and `main_test.go` is
given verbatim in the Implementation Blueprint. Every load-bearing behavior was
empirically verified in the target Go 1.26.4 environment
(`research/verified_facts.md`): the ldflags override mechanism and its
package-var requirement (§1); the exact `skpp <version>\n` byte format (§2); the
byte-exact `--path` stdout requirement and stderr-vs-stdout discipline (§3); that
rule 2 (sibling-of-binary) wins for a repo-root binary next to `skills/` and
needs only a dir (no SKILL.md) (§4); why a custom parser beats Go's `flag`
package for the §6 matrix (§5); the `run(args, stdout, stderr) int` testable-CLI
pattern (§6); how to drive `skillsdir.Find()` deterministically in tests (§7);
the white-box `package main` test convention (§8); the minimal import set (§9);
the minimal exit-code semantics for this subtask (§10). The consumed
`skillsdir.Find()` contract was read in full on disk. An implementer who knows Go
but nothing about this repo can complete this in one pass from this document._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M1T3S1/research/verified_facts.md
  why: "Proves: (1) `var version = 'dev'` is ldflags-overridable AND must be a
        package var (const breaks it); the -X arg is `main.version` not the
        import path; go test sees 'dev'. (2) exact `skpp <version>\n` format.
        (3) --path must print ONLY dir to stdout (acceptance is byte-exact);
        ErrNotFound goes to stderr. (4) rule 2 wins for a repo-root binary next
        to skills/ and needs only a dir (no SKILL.md) -> throwaway skills/.gitkeep
        suffices for the gate. (5) custom parser > Go flag package for the §6
        matrix (subcommands, long+short aliases, mutual exclusivity, tolerated
        unknowns). (6) run(args,stdout,stderr) int pattern for testability.
        (7) how to make Find() succeed (SKPP_SKILLS_DIR) and fail (unset +
        t.Chdir to empty temp) deterministically in tests. (8) white-box
        `package main` test convention. (9) imports = fmt, io, os only. (10)
        minimal exit-code semantics (version=0, path=0/1, default=1, no exit-2
        yet)."
  critical: "var version MUST be a package-level var, NOT const. -X cannot
             override a const (compile error). And -X uses `main.version` (the
             package path), NOT `github.com/dabstractor/skpp.version`. See §1."

# CONTRACT — the consumed package (read in full; do NOT modify)
- file: internal/skillsdir/skillsdir.go
  why: "Contains (already on disk): Find() (dir string, src Source, err error),
        ErrNotFound (exported sentinel whose .Error() is the one-line fix
        'could not locate the skills directory: set $SKPP_SKILLS_DIR, cd into the
        skpp repo, or reinstall skpp'), Source/SourceEnv/SourceSibling/
        SourceWalkUp, and (Source).String(). main.go imports and calls
        skillsdir.Find(); prints err to stderr verbatim on failure."
  pattern: "Find() returns (absDir, src, nil) on success or ('', 0, ErrNotFound)
            on all-miss. The src is for reporting only — DO NOT print it to
            stdout (--path stdout must be byte-exact = just the dir)."
  gotcha: "Do NOT wrap or prefix the ErrNotFound message before printing — print
           err verbatim (fmt.Fprintln(stderr, err)). The message IS the user-facing
           fix (PRD §8.4/§6.4). Do NOT import skillsdir's unexported symbols
           (envVar, findEnv, etc.) — only Find/ErrNotFound/Source are exported."

# CONTRACT — the Find() signature + the main -> skillsdir.Find data flow
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "Locks Find() (dir string, src Source, err error) and the data flow
        (main.parseArgs -> skillsdir.Find -> discover.Index). 'Output discipline
        (§6.4)' note: resolve first, print nothing on failure. 'Exit codes' note:
        0 success, 1 unresolved/no-skills/unresolvable, 2 unknown-flag/mixed
        (exit-2 deferred to M5 for THIS subtask)."
  section: "Data flow", "Output discipline (§6.4)", "Exit codes"

# CONTRACT — the exact §6 CLI surface this subtask is the first slice of
- file: PRD.md
  why: "§6.1: --version/-v stdout = 'skpp <version>' single line exit 0;
        --path/-p stdout = absolute skills dir, exit 0 (1 if unresolvable).
        §6.3: --help/--version take precedence; no-args -> usage to stderr exit 1
        (usage text is M5, exit code is matched now). §6.4: skills dir unresolvable
        -> stderr concise reason + fix, exit 1; nothing to stdout. §12.1: build
        command uses -ldflags \"-X main.version=$(git describe ...)\". §13: the
        acceptance gates this subtask must pass. READ-ONLY."
  critical: "§13 acceptance gate is BYTE-EXACT: test \"$(./skpp --path)\" = "
            "\"$PWD/skills\". Any extra stdout (a source label, a trailing space,
            a missing newline) breaks it. Print ONLY fmt.Fprintln(stdout, dir)."

# REFERENCE — the testing convention to follow (white-box, same-package test)
- file: internal/skillsdir/skillsdir_test.go
  why: "The repo's established test convention: `package skillsdir` (white-box,
        same-package), t.TempDir()/t.Setenv/t.Chdir, plain t.Errorf/t.Fatalf
        (no testify), NO t.Parallel() on tests that touch env/cwd. main_test.go
        mirrors this as `package main`."
  pattern: "White-box test file alongside the code; capture output via injected
            io.Writer (*bytes.Buffer), not by redirecting os.Stdout."

# URLS — the two load-bearing stdlib mechanisms
- url: https://pkg.go.dev/cmd/link
  why: "Documents -X importpath.name=value: sets a string variable at link time.
        Confirms the symbol is addressed by package path + var name; for a
        `package main` var the path is `main` (so `main.version`), NOT the
        module import path."
  section: "-X importpath.name=value"
- url: https://pkg.go.dev/fmt#Fprintf
  why: "fmt.Fprintf(stdout, \"skpp %s\\n\", version) for byte-exact --version
        output; fmt.Fprintln(stdout, dir) for --path; fmt.Fprintln(stderr, err)
        for the error line. Fprintln adds exactly one trailing \\n."
- url: https://pkg.go.dev/os#Exit
  why: "os.Exit(code) terminates with the given code and skips deferred funcs —
        which is why run() returns an int and main() is the only os.Exit caller."
```

### Current Codebase tree (S1+S2+S3 landed; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*'
internal/skillsdir/skillsdir.go        # S1+S2+S3: Source + findEnv/findSibling/findWalkUp + Find + ErrNotFound
internal/skillsdir/skillsdir_test.go   # S1+S2+S3 tests (white-box, package skillsdir)

$ ls -A
.git/  .gitignore  LICENSE  PRD.md  go.mod  go.sum  internal/  plan/  .pi-subagents/
# go.mod: module github.com/dabstractor/skpp, go 1.25, yaml.v3 // indirect
# .gitignore: ignores /skpp, /dist, *.test, *.out, .env*, .DS_Store, .pi-subagents/
#   (skills/ is NOT ignored -> it is a tracked dir per §5)
# NO main.go, NO main_test.go yet. NO discover/resolve/ui (later milestones).
# NO skills/ dir yet (P1.M6.T12 ships skills/example/SKILL.md).
```

### Desired Codebase tree with files to be added

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/skillsdir/* — UNCHANGED)
├── main.go            # CREATE — package main: version var, parseArgs, run, main
└── main_test.go       # CREATE — package main (white-box): version/path/precedence/default tests
```

| File (created) | Responsibility | Consumes |
|---|---|---|
| `main.go` | Entry point: parse args, dispatch `--version`/`--path`, exit codes | `skillsdir.Find()`, `skillsdir.ErrNotFound` |
| `main_test.go` | Unit tests for the parser + dispatcher (captured stdout/stderr) | `main.run`, `main.parseArgs`, `main.version`, `skillsdir` (via Find) |

**No new directories. No new packages. No `go.mod`/`go.sum` change (pure stdlib +
the already-present skillsdir internal dep). `skills/` is created only as a
THROWAWAY for the acceptance smoke test, then removed (P1.M6.T12 owns the real
tree).**

### Known Gotchas of our codebase & Go toolchain

```go
// GOTCHA #1 — `var version = "dev"` MUST be a package-level VAR, not a const and
// not a function-local. `-ldflags "-X main.version=<v>"` rewrites a package-scope
// string var at link time; it CANNOT override a const (compile error) or a local.
// VERIFIED (research §1): default build -> "skpp dev"; ldflags build -> overridden.
//
//   RIGHT: var version = "dev"           // package scope, overridable by -X main.version
//   WRONG: const version = "dev"         // -X cannot override -> build error
//   WRONG: version := "dev"  (in main()) // local; -X has no target; also unexported

// GOTCHA #2 — The -X symbol path is `main.version`, NOT the module import path.
// Because main.go is `package main`, the linker addresses the var as `main.version`.
// Do NOT use `-X github.com/dabstractor/skpp.version` — that targets a different
// (nonexistent) symbol and silently does nothing. PRD §12.1 uses `-X main.version`.

// GOTCHA #3 — `--path` stdout must be BYTE-EXACT: only the dir + one newline. The
// §13 gate `test "$(./skpp --path)" = "$PWD/skills"` fails on ANY extra output.
// Do NOT print the Source label, a prefix, or a trailing space to stdout. The
// `src` return from Find() is for reporting/debug ONLY — discard it (`_, `) or use
// it for nothing on stdout. Print the Find() error to STDERR (keeps `$(...)` empty
// on failure, which §6.4 requires so `pi --skill "$(skpp bad)"` fails loudly).
//
//   RIGHT: fmt.Fprintln(stdout, dir)            // "<dir>\n" exactly
//   WRONG: fmt.Fprintf(stdout, "%s (%s)\n", dir, src)  // breaks the gate
//   WRONG: fmt.Println(dir, src)                // extra " src\n"

// GOTCHA #4 — `go test` does NOT pass ldflags, so tests see version == "dev".
// Assert against the `version` var (readable from a `package main` test file),
// e.g. wantOut := "skpp " + version + "\n". This is robust to a future default
// change; asserting the literal "skpp dev\n" is also acceptable for now.

// GOTCHA #5 — Print the Find() error VERBATIM. ErrNotFound.Error() IS the
// user-facing one-line fix (PRD §8.4). Do NOT wrap (fmt.Errorf("...: %w", err))
// or prefix ("error: ") it before printing — that corrupts the fix message the
// skillsdir package carefully authored. `fmt.Fprintln(stderr, err)` is correct.

// GOTCHA #6 — Do NOT use Go's stdlib `flag` package. It cannot represent the §6
// matrix: the `check` subcommand (bare token, no dashes), positional `<tag>` args,
// long+short aliases (`-v`/`--version`), §6.3 mutual exclusivity, AND it errors on
// unknown flags (which THIS subtask must tolerate). A hand-rolled range+switch is
// ~15 lines, extensible (M5 appends cases), and degrades gracefully. VERIFIED §5.
//   Decision: custom parser. Do NOT import "flag".

// GOTCHA #7 — `os.Exit` is NOT unit-testable (kills the process, skips defers).
// Factor ALL logic into `run(args, stdout, stderr io.Writer) int`; `main()` is the
// ONLY os.Exit caller: `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`. Tests
// call run() with *bytes.Buffer writers. Injecting the writers is ESSENTIAL —
// without it tests cannot capture --path/--version output. VERIFIED §6.

// GOTCHA #8 — main_test.go MUST be `package main` (white-box), NOT
// `package main_test`, to read the `version` var and call unexported
// parseArgs/run. This matches the repo convention (skillsdir_test.go is
// `package skillsdir`). VERIFIED §8.

// GOTCHA #9 — Tests that drive `skillsdir.Find()` MUST control env/cwd:
//   success: t.Setenv("SKPP_SKILLS_DIR", tmpDir) -> rule 1 wins deterministically.
//   error:   unset env (t.Setenv("SKPP_SKILLS_DIR","") is NOT enough if cwd has an
//            ancestor skills/) -> use the unsetEnvVar helper pattern + t.Chdir into
//            an EMPTY t.TempDir() so rule 3 ascends to / and misses. Rule 2
//            deterministically misses in `go test` (test binary in /tmp/go-buildXXX).
//            t.Setenv and t.Chdir BOTH forbid t.Parallel() — do NOT add it.
// VERIFIED §7.

// GOTCHA #10 — Exit-2 is DEFERRED to P1.M5.T11. THIS subtask must NOT exit 2 on
// unknown flags (the item: "Unknown flags are tolerated for now"). The default
// (no --version, no --path) returns exit 1 — matching the eventual §6.3 no-args
// code — WITHOUT printing usage text (usage is M5). Returning 1 (not 2) keeps
// unknowns "tolerated". --help is NOT implemented (M5); it currently falls through
// to the default. Mark the precedence slot where help slots in with a comment.

// GOTCHA #11 — `--version` takes PRECEDENCE over `--path` (PRD §6.3). Check
// version BEFORE path in run(), so `skpp --version --path` prints the version and
// NEVER calls skillsdir.Find(). (This also means a broken skills dir never hides
// the version — useful for debugging.)

// GOTCHA #12 — Do NOT create `skills/`, `internal/discover`, `internal/resolve`,
// `internal/ui`, `install.sh`, `README.md`, or any completion files. Those are
// later subtasks (M2/M3/M4/M5/M6). The ONLY files this subtask creates are
// main.go and main_test.go. `skills/` appears only as a throwaway in the
// acceptance smoke test (Level 3) and is removed afterward.

// GOTCHA #13 — The Level 3 miss-path smoke test MUST use an ISOLATED binary
// (built into a fresh temp dir with NO sibling skills/), NOT the repo-root
// ./skpp. Reason: rule 2 (findSibling) resolves <dir-of-binary>/skills. The
// repo-root binary has the throwaway skills/ sibling (created for the §13 gate),
// so rule 2 would WIN and --path would print the path + exit 0 instead of
// missing. To force ALL THREE rules to miss in the smoke test: (a) build into
// $missbin/skpp (no sibling skills/ -> rule 2 misses), (b) unset
// SKPP_SKILLS_DIR (rule 1 misses), (c) cd into an empty mktemp -d (rule 3
// ascends to / and misses). The unit test TestRunPathFailureErrNotFound is NOT
// affected (the `go test` binary lives in /tmp/go-buildXXX with no sibling
// skills/), so it authoritatively covers the miss path regardless. VERIFIED §4.
```

---

## Implementation Blueprint

### Data model — minimal config struct

No ORM/pydantic (this is Go). The only "model" is the parsed-flags struct,
designed to grow with each milestone:

```go
// config holds the parsed CLI flags. It is grown by later milestones as more of
// the §6.1/§6.2 matrix lands. For THIS subtask only version and path are set;
// every other token is a tolerated no-op (P1.M5.T11 turns unknown flags into
// exit 2 and adds subcommand/positional handling).
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	// Future (M2-M5), do NOT add yet:
	//   list, all bool; search string; check bool; file, noColor, relative, help bool; tags []string
}
```

### File 1 — `main.go` (CREATE)

Create the file with EXACTLY this content (gofmt-clean; comments explain the
extensibility hooks for M2–M5):

```go
// Command skpp resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// For this subtask (P1.M1.T3.S1) only --version/--path are wired; every other
// §6 flag is added by later milestones (M2 --list, M3 <tag>/--all, M4
// --search/check, M5 --help + exit codes). The arg parser is intentionally a
// small hand-rolled switch (not Go's `flag` package) so the full §6 matrix —
// subcommands like `check`, positional <tag> args, long+short aliases, and §6.3
// mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1T3S1/research/verified_facts.md §5.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/dabstractor/skpp/internal/skillsdir"
)

// version is the skpp version string, printed by `skpp --version`. It is
// overridden at BUILD time via ldflags (PRD §12.1 build command):
//
//	go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .
//
// The default "dev" is used by `go run` and plain `go build` (no ldflags).
//
// IMPORTANT: this MUST be a package-level var, not a const. `-X main.version=...`
// rewrites a package-scope string var at link time; it cannot override a const
// (compile error) or a function-local. Because this file is `package main`, the
// linker symbol path is `main.version` (NOT the module import path).
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// config holds the parsed CLI flags. Grown by later milestones as more of the
// PRD §6.1/§6.2 matrix lands. For this subtask only version and path are set;
// every other token is a tolerated no-op (P1.M5.T11 turns unknown flags into
// exit 2 and adds subcommand/positional handling).
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	// Future (M2-M5), do NOT add yet:
	//   list, all bool; search string; check bool; file, noColor, relative, help bool; tags []string
}

// parseArgs scans argv tokens and fills a config. Flags may appear in any order
// (PRD §6). Long forms use POSIX double-dash; short forms a single dash. Unknown
// tokens are tolerated for now (a no-op switch default); the full unknown-flag
// -> exit 2 behavior and subcommand/positional parsing land in P1.M5.T11.
//
// To add a flag in a later milestone: append a `case "--name", "-n": cfg.name =
// true` (or capture the next arg for value-taking flags like --search <q>).
func parseArgs(args []string) config {
	var c config
	for _, a := range args {
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		default:
			// Unknown flag / subcommand / positional: tolerated for now.
			// P1.M5.T11 implements: unknown flag -> exit 2 (§6.2),
			// `check` subcommand dispatch, and <tag> positional capture.
		}
	}
	return c
}

// run is the testable dispatcher. It returns the process exit code so main() can
// call os.Exit(run(...)) without tests ever invoking os.Exit. stdout/stderr are
// injected so tests capture output via *bytes.Buffer instead of the real streams.
//
// Exit codes (PRD §6; this subtask's slice):
//   - 0: --version printed; --path succeeded
//   - 1: --path failed (skills dir unresolvable); default (no recognized flag)
//   - 2: (DEFERRED to P1.M5.T11) unknown flag / mutually-exclusive modes mixed
//
// Precedence (PRD §6.3): --version (and, in M5, --help) win over everything.
func run(args []string, stdout, stderr io.Writer) int {
	c := parseArgs(args)

	// Precedence tier: --version wins over every other flag (PRD §6.3).
	// P1.M5.T11 adds --help/-h to this same tier (before --path).
	if c.version {
		fmt.Fprintf(stdout, "skpp %s\n", version)
		return 0
	}

	if c.path {
		dir, _, err := skillsdir.Find() // src is for reporting only; not printed
		if err != nil {
			// Find() returns skillsdir.ErrNotFound whose message is the
			// user-facing one-line fix (PRD §8.4/§6.4). Print it verbatim to
			// stderr (NOT stdout) so $(...) stays empty on failure.
			fmt.Fprintln(stderr, err)
			return 1
		}
		// Byte-exact: ONLY the dir + newline. The §13 acceptance gate
		// `test "$(./skpp --path)" = "$PWD/skills"` depends on this.
		fmt.Fprintln(stdout, dir)
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}
```

### File 2 — `main_test.go` (CREATE, `package main` white-box)

Create the file with EXACTLY this content. It mirrors the repo's test convention
(white-box same-package, `t.TempDir`/`t.Setenv`/`t.Chdir`, plain `t.Errorf`/
`t.Fatalf`, no testify, no `t.Parallel()` on env/cwd tests):

```go
package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// unsetSkillsEnv removes SKPP_SKILLS_DIR for the test and restores it on cleanup.
// (Mirrors internal/skillsdir/skillsdir_test.go's unsetEnvVar helper.) Forbids
// t.Parallel via t.Setenv.
func unsetSkillsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SKPP_SKILLS_DIR", "")
}

// --- parseArgs ---

func TestParseArgsEmpty(t *testing.T) {
	c := parseArgs(nil)
	if c.version || c.path {
		t.Errorf("parseArgs(nil): version=%v path=%v; want both false", c.version, c.path)
	}
}

func TestParseArgsVersionLong(t *testing.T) {
	c := parseArgs([]string{"--version"})
	if !c.version || c.path {
		t.Errorf("parseArgs(--version): version=%v path=%v; want true,false", c.version, c.path)
	}
}

func TestParseArgsVersionShort(t *testing.T) {
	c := parseArgs([]string{"-v"})
	if !c.version {
		t.Errorf("parseArgs(-v): version=false; want true")
	}
}

func TestParseArgsPathLong(t *testing.T) {
	c := parseArgs([]string{"--path"})
	if !c.path || c.version {
		t.Errorf("parseArgs(--path): path=%v version=%v; want true,false", c.path, c.version)
	}
}

func TestParseArgsPathShort(t *testing.T) {
	c := parseArgs([]string{"-p"})
	if !c.path {
		t.Errorf("parseArgs(-p): path=false; want true")
	}
}

// Flags may appear in any order (PRD §6); both long+short forms recognized.
func TestParseArgsAnyOrderBothForms(t *testing.T) {
	c := parseArgs([]string{"-p", "--version"})
	if !c.version || !c.path {
		t.Errorf("parseArgs(-p --version): version=%v path=%v; want true,true", c.version, c.path)
	}
}

// Unknown tokens are tolerated (no-op) for now; exit-2 lands in P1.M5.T11.
func TestParseArgsUnknownTolerated(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "sometag", "check"})
	if c.version || c.path {
		t.Errorf("parseArgs(unknown): version=%v path=%v; want both false (tolerated)", c.version, c.path)
	}
}

// --- run: --version / -v ---

func TestRunVersionPrintsSkppVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--version): code=%d; want 0", code)
	}
	want := "skpp " + version + "\n" // version == "dev" under `go test` (no ldflags)
	if got := out.String(); got != want {
		t.Errorf("run(--version) stdout=%q; want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--version) stderr=%q; want empty", errOut.String())
	}
}

func TestRunVersionShortFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-v"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-v): code=%d; want 0", code)
	}
	if !strings.HasPrefix(out.String(), "skpp ") {
		t.Errorf("run(-v) stdout=%q; want 'skpp <version>\\n'", out.String())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Errorf("run(-v) stdout=%q; want trailing newline", out.String())
	}
}

// --- run: --path / -p ---

// --path success: SKPP_SKILLS_DIR set to an existing dir -> rule 1 wins, Find()
// returns that dir, printed byte-exact to stdout, exit 0.
func TestRunPathSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKPP_SKILLS_DIR", dir) // rule 1 wins deterministically
	var out, errOut bytes.Buffer
	code := run([]string{"--path"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--path) success: code=%d; want 0", code)
	}
	// Find() cleans the env value via filepath.Abs, so compare to the cleaned form.
	want := filepath.Clean(dir) + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(--path) stdout=%q; want %q (byte-exact dir + newline)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--path) success stderr=%q; want empty", errOut.String())
	}
}

func TestRunPathShortFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-p"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-p): code=%d; want 0", code)
	}
	if got := out.String(); got != filepath.Clean(dir)+"\n" {
		t.Errorf("run(-p) stdout=%q; want %q", got, filepath.Clean(dir)+"\n")
	}
}

// --path failure: env unset + cwd in an empty temp tree -> all three §8 rules
// miss -> Find() returns ErrNotFound. Assert: exit 1, stdout EMPTY, stderr has
// the one-line fix (SKPP_SKILLS_DIR / cd / reinstall). Empty stdout is the §6.4
// contract that makes `pi --skill "$(skpp bad)"` fail loudly.
func TestRunPathFailureErrNotFound(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // empty tree -> rule 3 ascends to / and misses; rule 2 misses in tests
	var out, errOut bytes.Buffer
	code := run([]string{"--path"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--path) failure: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--path) failure stdout=%q; want EMPTY (§6.4: print nothing on failure)", out.String())
	}
	msg := errOut.String()
	for _, want := range []string{"SKPP_SKILLS_DIR", "cd", "reinstall"} {
		if !strings.Contains(msg, want) {
			t.Errorf("run(--path) failure stderr=%q; missing substring %q", msg, want)
		}
	}
}

// --- run: precedence ---

// --version takes precedence over --path (PRD §6.3): version printed, Find()
// never called, exit 0 — even though skills dir is unresolvable here.
func TestRunVersionPrecedenceOverPath(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // would make --path fail, but --version wins first
	var out, errOut bytes.Buffer
	code := run([]string{"--path", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--path --version): code=%d; want 0 (version precedence)", code)
	}
	want := "skpp " + version + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(--path --version) stdout=%q; want %q (version, not path)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--path --version) stderr=%q; want empty", errOut.String())
	}
}

// --- run: default (no recognized flag) ---

// No args / unknown flags: tolerated for now, exit 1 (the eventual §6.3 no-args
// code), no usage text yet (P1.M5.T11). NOT exit 2 (deferred to M5).
func TestRunDefaultNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(nil, &out, &errOut)
	if code != 1 {
		t.Errorf("run(nil): code=%d; want 1 (no-args default; usage text is M5)", code)
	}
}

func TestRunDefaultUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--frobnicate"}, &out, &errOut)
	if code != 1 {
		t.Errorf("run(--frobnicate): code=%d; want 1 (unknown tolerated; exit-2 is M5)", code)
	}
}
```

> **Copy-paste correctness:** the two blueprint files above are gofmt-clean and
> compile as-is (imports limited to exactly what each file uses: `main.go` →
> fmt/io/os/skillsdir; `main_test.go` → bytes/path/filepath/strings/testing).
> Write them verbatim.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0: PRECONDITION — confirm S3 (skillsdir.Find) is on disk and green
  - COMMAND: cd /home/dustin/projects/skpp
  - COMMAND: grep -qE 'func Find\(\) \(dir string, src Source, err error\)' internal/skillsdir/skillsdir.go
  - COMMAND: grep -qE 'var ErrNotFound = errors\.New\(' internal/skillsdir/skillsdir.go
  - COMMAND: go test ./internal/skillsdir/ >/dev/null 2>&1 && echo "skillsdir green" || echo "NOT green"
  - EXPECT: both symbols exist AND tests pass. If Find/ErrNotFound are MISSING,
            S3 has NOT landed — STOP and let S3 land first (main.go imports them).

Task 1: CREATE main.go
  - WRITE: the exact content from the Blueprint (File 1) to ./main.go.
  - CHECK: `package main`; `var version = "dev"` (package scope, NOT const);
           imports = fmt, io, os, github.com/dabstractor/skpp/internal/skillsdir;
           parseArgs (switch over --version/-v, --path/-p, default no-op);
           run(args, stdout, stderr io.Writer) int; main() -> os.Exit(run(...)).
  - GOTCHA: var version is a package VAR (const breaks -X). Do NOT import "flag".
            --path prints ONLY dir to stdout; err to stderr verbatim. version
            precedence checked before path.

Task 2: CREATE main_test.go
  - WRITE: the exact content from the Blueprint (File 2) to ./main_test.go.
  - CHECK: `package main` (white-box, NOT main_test); tests cover parseArgs
           (empty, version long/short, path long/short, any-order, unknown
           tolerated) + run (version prints "skpp <version>", -v, --path success,
           -p, --path failure ErrNotFound, version precedence over path, default
           no-args exit 1, unknown flag exit 1).
  - CHECK: imports are exactly bytes, path/filepath, strings, testing (no "os";
           the file does not use it — do not add a dead import).
  - GOTCHA: success path uses t.Setenv("SKPP_SKILLS_DIR", tmpDir); failure path
            uses unsetSkillsEnv + t.Chdir(t.TempDir()). NO t.Parallel() on any
            test that touches env/cwd.

Task 3: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w main.go main_test.go
  - COMMAND: gofmt -l main.go main_test.go   # MUST print nothing
  - COMMAND: go vet .                        # MUST be clean
  - COMMAND: go build -o skpp .              # exit 0, produces ./skpp
  - COMMAND: go test . -v                    # ALL main tests PASS
  - COMMAND: go test ./...                   # whole module still green
  - EXPECT: zero errors, zero vet findings, gofmt silent, all tests pass.

Task 4: ACCEPTANCE SMOKE TEST (PRD §13 gates) — Level 3 in Validation Loop
  - COMMAND: the Level 3 block below (build, throwaway skills/.gitkeep, the two
             §13 assertions, the ldflags override proof, then cleanup).
  - EXPECT: "PATH OK", "VERSION OK", "LDFLAGS OK" all printed; throwaway skills/
            removed afterward (P1.M6.T12 owns the real tree).

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: main.go has var version (not const), no "flag" import, --path prints
            only dir, version-precedence ordering; go.mod/go.sum/PRD.md/
            internal/skillsdir unchanged; no discover/resolve/ui/install.sh/
            README created.
```

### Implementation Patterns & Key Details

```go
// PATTERN: the testable-CLI split (run returns int; main is the only os.Exit).
//   func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
//   func run(args []string, stdout, stderr io.Writer) int { ... return code }
// WHY: os.Exit kills the process and skips defers -> untestable. Returning an
//      int from run() lets tests assert the exit code; injecting the writers
//      lets tests capture stdout/stderr via *bytes.Buffer. (Verified §6.)

// PATTERN: the extensible switch parser (range + case; default = tolerated).
//   for _, a := range args {
//       switch a {
//       case "--version", "-v": c.version = true
//       case "--path", "-p":    c.path = true
//       default: /* tolerated; M5 turns unknown flags into exit 2 */
//       }
//   }
// WHY: Go's `flag` package cannot express the §6 matrix (subcommands, long+short
//      aliases, mutual exclusivity, tolerated unknowns). A switch is ~15 lines,
//      handles any-order + both forms, and M5 extends it by appending cases.
//      (Verified §5.)

// PATTERN: precedence tier (version/help checked before everything).
//   if c.version { fmt.Fprintf(stdout, "skpp %s\n", version); return 0 }
//   if c.path { ... }
// WHY: PRD §6.3 — --version/--help take precedence. Checking version first means
//      a broken skills dir never hides the version (debugging win) and Find() is
//      never called when version is requested. M5 adds --help to this same tier.

// PATTERN: byte-exact stdout for path output; error verbatim to stderr.
//   dir, _, err := skillsdir.Find()        // src discarded (not printed)
//   if err != nil { fmt.Fprintln(stderr, err); return 1 }
//   fmt.Fprintln(stdout, dir); return 0
// WHY: §13 gate `test "$(./skpp --path)" = "$PWD/skills"` is byte-exact; §6.4
//      requires NOTHING on stdout on failure (so $(...) is empty). Fprintln adds
//      exactly one "\n". Print err verbatim — its message IS the user-facing fix.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - main.go is `package main` at the repo root (module entrypoint).
  - imports: "fmt", "io", "os", "github.com/dabstractor/skpp/internal/skillsdir".
  - consumes (from skillsdir, READ-ONLY — do not modify): Find, ErrNotFound,
    Source (+SourceEnv/SourceSibling/SourceWalkUp, (Source).String).
  - exposes (for tests, white-box): var version, parseArgs, run, type config.

BUILD (PRD §12.1, verified §1):
  - plain:    go build -o skpp .                          # version == "dev"
  - ldflags:  go build -ldflags "-X main.version=<v>" -o skpp .   # overrides
  - the -X symbol is `main.version` (package path), NOT the module import path.

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into):
  - P1.M2.T6 (--list): add `case "--list","-l": c.list=true` in parseArgs;
    add a `if c.list { ... }` branch in run (before the default).
  - P1.M3.T8 (<tag>/--all): capture positionals in parseArgs; add `--all/-a`;
    add the tag-resolution branch in run.
  - P1.M4.T9/T10 (--search/check): add `--search <q>` (value-taking: peek next
    arg) and `check` subcommand dispatch.
  - P1.M5.T11: add `--help/-h` to the version precedence tier; implement the
    no-args -> usage-to-stderr + exit 1, unknown-flag -> exit 2, and mode-
    exclusivity rules. THIS subtask leaves those as the tolerated default.

NO CHANGES TO:
  - go.mod / go.sum (pure stdlib + already-present internal/skillsdir dep)
  - PRD.md (read-only) / any tasks.json (orchestrator-owned)
  - internal/skillsdir/* (S1/S2/S3-owned; consumed verbatim)
  - any other package or file (discover/resolve/ui/install.sh/README/completions
    are later subtasks; skills/ is a throwaway for the smoke test only)
```

---

## Validation Loop

### Level 1: Format, vet, build (immediate, per file)

```bash
cd /home/dustin/projects/skpp

# Format in place, then confirm nothing is left unformatted (silent == pass)
gofmt -w main.go main_test.go
test -z "$(gofmt -l main.go main_test.go)" || { echo "FAIL: gofmt found unformatted files"; gofmt -d main.go main_test.go; exit 1; }
echo "gofmt OK"

# Vet the module root (where main lives)
go vet . || { echo "FAIL: go vet ."; exit 1; }
echo "go vet OK"

# Build the binary at the repo root (version defaults to "dev")
go build -o skpp . || { echo "FAIL: go build -o skpp ."; exit 1; }
echo "go build OK"
```

### Level 2: Unit tests (component validation)

```bash
cd /home/dustin/projects/skpp

# Run main tests verbosely — parseArgs + run (version/path/precedence/default)
go test . -v || { echo "FAIL: go test . -v"; exit 1; }

# Targeted: the precedence + byte-exact + ErrNotFound assertions
go test . -run 'TestRunVersionPrecedenceOverPath|TestRunPathSuccess|TestRunPathFailureErrNotFound|TestRunVersionPrintsSkppVersion' -v \
  || { echo "FAIL: precedence/path/version tests"; exit 1; }

# Whole module still green (skillsdir + main)
go test ./... || { echo "FAIL: go test ./..."; exit 1; }
echo "Level 2 PASS"
```

### Level 3: Acceptance smoke test (PRD §13 gates + ldflags proof)

This proves end-to-end resolution through a REAL binary. `skills/` is created as a
THROWAWAY so rule 2 (sibling-of-binary) wins; it is removed afterward (P1.M6.T12
ships the real `skills/example/SKILL.md`; §13 is re-run in P1.M6.T16).

```bash
cd /home/dustin/projects/skpp

# Rebuild clean (version == "dev")
go build -o skpp . || { echo "FAIL: build"; exit 1; }

# Throwaway skills dir so rule 2 (sibling-of-binary) resolves $PWD/skills.
# Rule 2 needs only an existing dir (no SKILL.md required — unlike rule 3).
mkdir -p skills && touch skills/.gitkeep

# §13 gate 1: --path prints exactly $PWD/skills (byte-exact via test(1)).
test "$(./skpp --path)" = "$PWD/skills" && echo "PATH OK" \
  || { echo "FAIL: --path: got '$(./skpp --path)'; want '$PWD/skills'"; rm -rf skills; exit 1; }

# §13 gate 2: --version prints exactly "skpp dev".
test "$(./skpp --version)" = "skpp dev" && echo "VERSION OK" \
  || { echo "FAIL: --version: got '$(./skpp --version)'"; rm -rf skills; exit 1; }

# Short forms work too.
test "$(./skpp -p)" = "$PWD/skills" && test "$(./skpp -v)" = "skpp dev" \
  || { echo "FAIL: short forms -p/-v"; rm -rf skills; exit 1; }
echo "SHORT-FORM OK"

# ldflags override proof (PRD §12.1 mechanism): -X main.version=<v> replaces "dev".
go build -ldflags "-X main.version=v0.0.0-test" -o skpp . \
  || { echo "FAIL: ldflags build"; rm -rf skills; exit 1; }
test "$(./skpp --version)" = "skpp v0.0.0-test" && echo "LDFLAGS OK" \
  || { echo "FAIL: ldflags override: got '$(./skpp --version)'"; rm -rf skills; exit 1; }

# --version precedence over --path (even though skills dir is resolvable here).
go build -o skpp . || { echo "FAIL: rebuild"; rm -rf skills; exit 1; }
test "$(./skpp --version --path)" = "skpp dev" && echo "PRECEDENCE OK" \
  || { echo "FAIL: --version --path should print version (precedence)"; rm -rf skills; exit 1; }

# --path failure path: force ALL THREE §8 rules to miss. The repo-root ./skpp has
# the throwaway skills/ sibling (rule 2 would win), so build an ISOLATED binary
# into a temp dir with NO sibling skills/. Then: unset env (rule 1 miss), the
# isolated binary's dir has no skills/ (rule 2 miss), and cwd is an empty temp
# tree (rule 3 ascends to / and misses) -> Find() returns ErrNotFound.
# Assert: exit 1, stdout EMPTY, stderr has the one-line fix.
# NOTE: capture the code with `; code=$?` (do NOT run this block under `set -e` —
# the binary deliberately exits 1, which set -e would treat as a script failure).
missbin="$(mktemp -d)"
go build -o "$missbin/skpp" . || { echo "FAIL: isolated miss-path build"; rm -rf "$missbin"; exit 1; }
empty="$(mktemp -d)"
( cd "$empty" && unset SKPP_SKILLS_DIR && "$missbin/skpp" --path >/tmp/skpp-path-out 2>/tmp/skpp-path-err )
code=$?
test "$code" = "1" || { echo "FAIL: --path miss exit code=$code; want 1"; cat /tmp/skpp-path-out; rm -rf "$missbin" "$empty"; exit 1; }
test ! -s /tmp/skpp-path-out || { echo "FAIL: --path miss stdout must be empty"; cat /tmp/skpp-path-out; rm -rf "$missbin" "$empty"; exit 1; }
grep -q SKPP_SKILLS_DIR /tmp/skpp-path-err && grep -q reinstall /tmp/skpp-path-err \
  || { echo "FAIL: --path miss stderr missing fix phrase"; cat /tmp/skpp-path-err; rm -rf "$missbin" "$empty"; exit 1; }
rm -rf "$missbin" "$empty"
echo "PATH-MISS OK"

# Cleanup: remove the THROWAWAY skills tree + scratch files (P1.M6.T12 owns the
# real skills/ tree; .gitignore already ignores /skpp so the binary is harmless,
# but tidy it anyway).
rm -rf skills skpp /tmp/skpp-path-out /tmp/skpp-path-err

echo "Level 3 PASS (§13 gates + ldflags + precedence + miss path)"
```

### Level 4: Scope-boundary & contract check

```bash
cd /home/dustin/projects/skpp

# main.go exists and is package main
test -f main.go || { echo "FAIL: main.go missing"; exit 1; }
grep -q '^package main' main.go || { echo "FAIL: main.go not package main"; exit 1; }

# version is a package-level VAR (not const) — ldflags-overridable (§1)
grep -qE '^var version = "' main.go || { echo "FAIL: var version missing or not a package var"; exit 1; }
! grep -qE '\bconst version\b' main.go || { echo "FAIL: version must not be const (-X cannot override)"; exit 1; }

# main calls os.Exit(run(...)) and run returns int (testable-CLI pattern)
grep -qE 'func main\(\)' main.go || { echo "FAIL: main() missing"; exit 1; }
grep -qE 'os\.Exit\(run\(' main.go || { echo "FAIL: main must call os.Exit(run(...))"; exit 1; }
grep -qE 'func run\(args \[\]string, stdout, stderr io\.Writer\) int' main.go \
  || { echo "FAIL: run signature wrong"; exit 1; }

# parseArgs is a switch over the two flag pairs (extensible for M5)
grep -qE 'func parseArgs\(args \[\]string\) config' main.go || { echo "FAIL: parseArgs missing"; exit 1; }
grep -q 'case "--version", "-v"' main.go || { echo "FAIL: --version/-v case missing"; exit 1; }
grep -q 'case "--path", "-p"' main.go || { echo "FAIL: --path/-p case missing"; exit 1; }

# --path prints ONLY dir to stdout (byte-exact); err to stderr verbatim
grep -qE 'fmt\.Fprintln\(stdout, dir\)' main.go || { echo "FAIL: --path must Fprintln(stdout, dir)"; exit 1; }
grep -qE 'fmt\.Fprintln\(stderr, err\)' main.go || { echo "FAIL: error must Fprintln(stderr, err) verbatim"; exit 1; }
# version precedence: the version `if` must come BEFORE the path `if`
awk '/if c\.version/{v=NR} /if c\.path/{p=NR} END{exit !(v && p && v<p)}' main.go \
  || { echo "FAIL: --version must be checked before --path (precedence)"; exit 1; }

# Do NOT use Go's flag package (custom parser is the decision)
! grep -q '"flag"' main.go || { echo "FAIL: do not import flag (custom parser)"; exit 1; }

# main_test.go is white-box package main with the key tests
test -f main_test.go || { echo "FAIL: main_test.go missing"; exit 1; }
grep -q '^package main' main_test.go || { echo "FAIL: main_test.go must be package main (white-box)"; exit 1; }
for tn in TestRunVersionPrintsSkppVersion TestRunVersionPrecedenceOverPath TestRunPathSuccess TestRunPathFailureErrNotFound TestRunDefaultNoArgs TestParseArgsAnyOrderBothForms; do
  grep -q "func $tn" main_test.go || { echo "FAIL: test $tn missing"; exit 1; }
done

# MUST NOT have touched go.mod / go.sum / PRD.md / skillsdir
git diff --quiet go.mod   || { echo "FAIL: go.mod changed"; exit 1; }
git diff --quiet go.sum   || { echo "FAIL: go.sum changed"; exit 1; }
git diff --quiet PRD.md   || { echo "FAIL: PRD.md changed (read-only)"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir.go      || { echo "FAIL: skillsdir.go changed"; exit 1; }
git diff --quiet internal/skillsdir/skillsdir_test.go || { echo "FAIL: skillsdir_test.go changed"; exit 1; }

# MUST NOT have created later-milestone files/packages (and skills/ must be gone
# — it was a throwaway for Level 3)
test ! -d internal/discover || { echo "FAIL: discover/ must not exist (M2)"; exit 1; }
test ! -d internal/resolve  || { echo "FAIL: resolve/ must not exist (M3)"; exit 1; }
test ! -d internal/ui       || { echo "FAIL: ui/ must not exist (M2)"; exit 1; }
test ! -f install.sh        || { echo "FAIL: install.sh must not exist (M6)"; exit 1; }
test ! -f README.md         || { echo "FAIL: README.md must not exist (M6)"; exit 1; }
test ! -d skills            || { echo "FAIL: skills/ must be removed (throwaway; M6 owns it)"; exit 1; }

echo "Level 4 PASS (scope + contract respected)"
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l main.go main_test.go` silent, `go vet .` clean, `go build -o skpp .` exit 0
- [ ] Level 2 PASS — `go test . -v` all main tests pass; `go test ./...` whole module green
- [ ] Level 3 PASS — §13 gates (`--path` = `$PWD/skills`, `--version` = `skpp dev`), short forms, ldflags override (`skpp v0.0.0-test`), version precedence, and the miss path (exit 1, empty stdout, stderr fix phrase)
- [ ] Level 4 PASS — `var version` is a package var (not const); run/parseArgs signatures correct; version checked before path; no `flag` import; white-box test file with all key tests; nothing else touched; no later-milestone files; throwaway `skills/` removed

### Feature Validation
- [ ] `./skpp --version` and `./skpp -v` print `skpp <version>\n` to stdout, exit 0
- [ ] Default build prints `skpp dev`; ldflags build prints the overridden value
- [ ] `./skpp --path` and `./skpp -p` print the resolved dir (single line) to stdout, exit 0 on success
- [ ] `./skpp --path` on an unresolvable dir prints the one-line fix to stderr, prints nothing to stdout, exit 1
- [ ] `--version` takes precedence over `--path` (Find() not called when version requested)
- [ ] Unknown flags / no args tolerated (exit 1, not exit 2; exit-2 is M5)
- [ ] `test "$(./skpp --path)" = "$PWD/skills"` passes (byte-exact)

### Code Quality / Convention Validation
- [ ] `main.go` is `package main` at repo root; imports limited to fmt/io/os/skillsdir
- [ ] `main_test.go` is white-box `package main`, mirroring skillsdir_test.go's style (t.TempDir/t.Setenv/t.Chdir, plain t.Errorf/t.Fatalf, no testify, no t.Parallel on env/cwd tests)
- [ ] `var version` is a package-level var (ldflags-overridable), documented with the §12.1 build command in its comment
- [ ] The arg parser is a documented extensible switch (M2–M5 extension points in comments)
- [ ] run() takes injected io.Writer streams (testable); main() is the only os.Exit caller
- [ ] Error printed verbatim (no wrap/prefix); --path stdout byte-exact (no source label)

### Scope Discipline
- [ ] Did NOT modify `internal/skillsdir/*` (consumed verbatim — Find/ErrNotFound/Source)
- [ ] Did NOT modify `go.mod` / `go.sum` (no new deps)
- [ ] Did NOT modify `PRD.md` (read-only) or any `tasks.json` (orchestrator-owned)
- [ ] Did NOT create `discover` / `resolve` / `ui` / `install.sh` / `README.md` / `completions/` (later milestones)
- [ ] Did NOT commit a `skills/` tree (throwaway for the smoke test only; removed; P1.M6.T12 owns it)

---

## Anti-Patterns to Avoid

- ❌ **Don't make `version` a const or a local.** `-ldflags "-X main.version=<v>"`
  rewrites a package-scope string VAR at link time; it cannot override a const
  (compile error) or a function-local. It MUST be `var version = "dev"` at package
  scope. (Verified §1.) And the -X symbol is `main.version` (the package path for
  `package main`), NOT `github.com/dabstractor/skpp.version`.
- ❌ **Don't use Go's `flag` package.** It can't express the §6 matrix: the `check`
  subcommand (bare token), positional `<tag>` args, long+short aliases, §6.3
  mutual exclusivity, AND it errors on unknown flags (which this subtask must
  tolerate). Use a hand-rolled range+switch. (Verified §5.)
- ❌ **Don't print anything but the dir to `--path` stdout.** The §13 gate
  `test "$(./skpp --path)" = "$PWD/skills"` is byte-exact. No source label, no
  prefix, no trailing space. Print the `Source` nowhere on stdout (it's for
  reporting/debug only). Print the Find() error to STDERR (so `$(...)` is empty on
  failure — §6.4). (Verified §3.)
- ❌ **Don't wrap or prefix the Find() error.** `ErrNotFound.Error()` IS the
  user-facing one-line fix (PRD §8.4). `fmt.Fprintln(stderr, err)` — verbatim.
  `fmt.Errorf("...: %w", err)` or `"error: "+err` corrupts the fix message.
- ❌ **Don't call `os.Exit` from `run()`.** `os.Exit` is untestable (kills the
  process, skips defers). Return an int from `run(args, stdout, stderr io.Writer)`
  and let `main()` be the sole `os.Exit(run(...))` caller. Inject the writers so
  tests capture output. (Verified §6.)
- ❌ **Don't implement `--help`, exit-2, or usage text.** Those are P1.M5.T11.
  This subtask tolerates unknown flags (default exit 1, not 2) and leaves the
  no-args usage slot as a clearly-marked stub. Implementing them now is scope
  creep that conflicts with M5.
- ❌ **Don't create `skills/`, `discover/`, `resolve/`, `ui/`, `install.sh`, or
  `README.md`.** Those are later subtasks. `skills/` appears ONLY as a throwaway
  in the Level 3 smoke test and is removed. The deliverable is `main.go` +
  `main_test.go` only.
- ❌ **Don't add `t.Parallel()` to tests that touch env or cwd.** `t.Setenv` and
  `t.Chdir` both forbid it. (Verified §7.) The skillsdir tests follow this; mirror
  them.
