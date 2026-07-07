// Command skpp resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// Wired so far (grown milestone by milestone): --version/--path (M1.T3) and
// --list (M2.T6). Every other §6 flag is added by later milestones (M3
// <tag>/--all, M4 --search/check, M5 --help + exit codes). The arg parser is
// intentionally a small hand-rolled switch (not Go's `flag` package) so the full
// §6 matrix — subcommands like `check`, positional <tag> args, long+short
// aliases, and §6.3 mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1.T3.S1/research/verified_facts.md §5.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
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

// isTerminal reports whether w is an interactive terminal (a character device).
// It decides whether --list/--search emit ANSI color by default (PRD §6.2: color
// is on for a TTY unless --no-color is set). It type-asserts w to *os.File and
// checks the ModeCharDevice bit, so a *bytes.Buffer (tests) or a pipe/redirect
// correctly yields false -> no color, keeping output deterministic and pipe-safe.
//
// It is a package var so tests can override it to exercise the color-enabled path
// through run() without a real terminal. NOT safe for t.Parallel (mutates package
// state); the repo convention is no t.Parallel() on such tests anyway.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// config holds the parsed CLI flags. Grown by later milestones as more of the
// PRD §6.1/§6.2 matrix lands. For this subtask version, path, list, and noColor
// are set; every other token is a tolerated no-op (P1.M5.T11 turns unknown flags
// into exit 2 and adds subcommand/positional handling).
type config struct {
	version bool // --version / -v : print "skpp <version>" and exit 0
	path    bool // --path / -p    : print resolved skills dir and exit 0/1
	list    bool // --list / -l    : print the human-readable catalog table (§6.1)
	noColor bool // --no-color     : disable ANSI color even on a TTY (§6.2)
	// Future (M3-M5), do NOT add yet:
	//   all bool; search string; check bool; file, relative, help bool; tags []string
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
		case "--list", "-l":
			c.list = true
		case "--no-color":
			c.noColor = true
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
//   - 0: --version printed; --path succeeded; --list printed the catalog
//   - 1: --path/--list failed (skills dir unresolvable, or no skills for --list);
//     default (no recognized flag)
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

	if c.list {
		// PRD §6.1 `skpp --list`: resolve the store, build the index, render the
		// table. This is the FIRST place the Find() -> discover.Index() data flow
		// is wired end-to-end (M2.T6). Exit 1 on any failure path.
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // verbatim one-line fix (PRD §6.4/§8.4)
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		if len(skills) == 0 {
			// PRD §6.1: --list exits 1 "if no skills found". Message to stderr so
			// stdout stays clean for any consumer.
			fmt.Fprintln(stderr, "no skills found in "+dir)
			return 1
		}
		// Color only when stdout is a TTY AND --no-color was not given (PRD §6.2).
		// A *bytes.Buffer (tests) / pipe / file is not a TTY -> plain output.
		ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}
