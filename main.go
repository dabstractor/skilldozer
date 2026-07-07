// Command skpp resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// Wired so far (grown milestone by milestone): --version/--path (M1.T3),
// --list (M2.T6), <tag> resolution (M3.T8.S1), and the --file/--relative/--all
// modifiers (M3.T8.S2). Every other §6 flag is added by later milestones (M4
// --search/check, M5 --help + exit codes). The arg parser is intentionally a
// small hand-rolled switch (not Go's `flag` package) so the full §6 matrix —
// subcommands like `check`, positional <tag> args, long+short aliases, and
// §6.3 mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1T3.S1/research/verified_facts.md §5.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
	"github.com/dabstractor/skpp/internal/search"
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
// PRD §6.1/§6.2 matrix lands. This version sets version, path, list, noColor,
// tags, and the S2 modifiers all/file/relative; every other token is a tolerated
// no-op (P1.M5.T11 turns unknown flags into exit 2 and adds subcommand handling).
type config struct {
	version    bool     // --version / -v : print "skpp <version>" and exit 0
	path       bool     // --path / -p    : print resolved skills dir and exit 0/1
	list       bool     // --list / -l    : print the human-readable catalog table (§6.1)
	all        bool     // --all / -a     : print every skill's directory path, one per line (§6.1) [NEW]
	file       bool     // --file / -f    : print the SKILL.md path instead of the dir path (§6.2) [NEW]
	relative   bool     // --relative     : print paths relative to the skills dir, not absolute (§6.2) [NEW]
	noColor    bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	tags       []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run
	searchMode bool     // --search <q>/-s : substring search over tag/name/description/keywords (§6.1) [NEW]
	searchQ    string   // the --search query value (consumed from the token after --search/-s) [NEW]
	// Future (M5), do NOT add yet:
	//   check bool; help bool
}

// parseArgs scans argv tokens and fills a config. Flags may appear in any order
// (PRD §6). Long forms use POSIX double-dash; short forms a single dash. Unknown
// dashed flags are tolerated for now (a no-op in the default branch); the full
// unknown-flag -> exit 2 behavior and §6.3 mutual-exclusivity land in P1.M5.T11.
//
// To add a flag in a later milestone: append a `case "--name", "-n": cfg.name =
// true` (or capture the next arg for value-taking flags like --search <q>).
func parseArgs(args []string) config {
	var c config
	// Index-based loop (not range) so a value-taking flag (--search <q>) can
	// CONSUME the following token via i++ without it also being captured as a tag.
	// PRD §6.1/§6.2: --search/-s take exactly one value; every other flag is a bool.
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--all", "-a":
			c.all = true
		case "--file", "-f":
			c.file = true
		case "--relative":
			c.relative = true
		case "--no-color":
			c.noColor = true
		case "--search", "-s":
			// Value-taking flag: consume the NEXT token verbatim as the query. The
			// value is NOT appended to c.tags (i++ skips it). If --search is the
			// LAST token (no value follows) searchMode stays false and the call
			// falls through to the no-recognized-mode default (exit 1); the proper
			// "flag requires an argument" exit-2 is P1.M5.T11. A value starting with
			// '-' (e.g. `--search -x`) is grabbed as the literal query "-x".
			if i+1 < len(args) {
				c.searchMode = true
				c.searchQ = args[i+1]
				i++
			}
		default:
			// Positional <tag> (PRD §6.1 `skpp <tag> [<tag>...]`): a token that
			// does NOT start with '-' is a tag, captured here and resolved in run.
			// Dashed unknowns (e.g. --frobnicate) also fall through to this default
			// and are tolerated (no-op); P1.M5.T11 turns them into exit 2 and adds
			// §6.3 mutual-exclusivity (tag mixed with --list/--search/--all).
			if !strings.HasPrefix(a, "-") {
				c.tags = append(c.tags, a)
			}
		}
	}
	return c
}

// run is the testable dispatcher. It returns the process exit code so main() can
// call os.Exit(run(...)) without tests ever invoking os.Exit. stdout/stderr are
// injected so tests capture output via *bytes.Buffer instead of the real streams.
//
// Exit codes (PRD §6; this subtask's slice):
//   - 0: --version printed; --path succeeded; --list printed the catalog; all
//     <tag>s resolved (one absolute path per line printed); --all printed the store
//   - 1: --path/--list failed; ANY <tag> unresolved/ambiguous (nothing on stdout);
//     skills dir unresolvable; default (no recognized flag)
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
		// Note: --list prints a TABLE, so the --file/--relative path modifiers do
		// NOT apply to it (PRD §6.2 header: modifiers combine with tag resolution
		// or --all).
		ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// --search mode: `skpp --search <q>` / `-s <q>` (PRD §6.1). Filters the index to
	// skills where <q> is a case-insensitive substring of the tag, frontmatter name,
	// description, or any metadata keyword (internal/search), then renders the SAME
	// table as --list via ui.PrintList (PRD §6.1: "same table format as --list,
	// filtered"). The filtered slice keeps discover.Index's RelTag sort. Exit 0 with
	// the table on matches; exit 1 (stderr message, EMPTY stdout) when nothing
	// matches (PRD §6.1: "1 if no matches"). --no-color / TTY color gating is shared
	// with --list; --file/--relative do NOT apply (search prints a TABLE, not paths —
	// PRD §6.2: modifiers combine with tag/--all only).
	if c.searchMode {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		matched := search.Search(c.searchQ, skills)
		if len(matched) == 0 {
			// PRD §6.1: exit 1 "if no matches". Mirror --list's "no skills found"
			// convention: message to stderr, stdout stays clean.
			fmt.Fprintln(stderr, "no skills matched "+c.searchQ)
			return 1
		}
		ui.PrintList(stdout, matched, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// --all mode: print every skill's directory path, one per line, SORTED by
	// canonical tag (PRD §6.1). discover.Index already sorts []Skill by RelTag, so
	// this just walks the index in order. Exit 0 even for an empty store (PRD §6.1
	// `--all` is always exit 0, unlike --list which exits 1 "if no skills found" —
	// --all is a scripting command where empty output + exit 0 is the useful shape).
	// The --file/--relative modifiers apply here too (PRD §6.2 header: "combine with
	// tag resolution or --all"), via the shared skillPath() helper.
	if c.all {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		for _, s := range skills {
			fmt.Fprintln(stdout, skillPath(s, dir, c)) // absolute dir by default; --file/--relative apply
		}
		return 0
	}

	// Tag-resolution mode: `skpp <tag> [<tag>...]` (PRD §6.1). Resolves each tag to
	// its skill path and prints one path per line, in input order.
	//
	// ATOMICITY (PRD §6.4 — the critical-for-$(...) contract): resolve EVERY tag
	// first, buffering the resulting paths; if ANY tag fails (unknown/ambiguous),
	// print one error line per problem tag to stderr, print NOTHING to stdout, and
	// exit 1. The buffered paths are flushed ONLY when the whole invocation is
	// known-good. This makes `pi --skill "$(skpp bad)"` fail loudly (empty $(),
	// exit 1) instead of passing a partial or garbage path. Each error is printed
	// verbatim from resolve's typed errors — UnknownError names the tag,
	// AmbiguousError lists the candidate full tags (no "skpp:" prefix, matching the
	// skillsdir.ErrNotFound convention used by --path/--list). The default output is
	// the skill DIRECTORY path; --file swaps to the SKILL.md path and --relative
	// makes it relative to the skills dir (applied by skillPath, PRD §6.2).
	if len(c.tags) > 0 {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		paths := make([]string, 0, len(c.tags)) // buffered; flushed only if all resolve
		hadErr := false
		for _, tag := range c.tags {
			res, rerr := resolve.Resolve(tag, skills)
			if rerr != nil {
				fmt.Fprintln(stderr, rerr) // one error line per problem tag (verbatim)
				hadErr = true
				continue
			}
			// skillPath applies --file (SourceFile vs Dir) and --relative (Rel to
			// skills dir); default is the absolute dir (PRD §6.1/§6.2).
			paths = append(paths, skillPath(res.Skill, dir, c))
		}
		if hadErr {
			return 1 // paths buffered but never written → stdout empty (§6.4)
		}
		for _, p := range paths {
			fmt.Fprintln(stdout, p) // one path per line, input order
		}
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}

// skillPath returns the path to print for a resolved skill, applying the PRD §6.2
// --file and --relative modifiers. It is the shared formatter used by BOTH the
// <tag>-resolution loop and the --all loop, so the modifiers behave identically in
// the two modes (PRD §6.2 header: "combine with tag resolution or --all").
//
// Precedence of effects:
//   - default (neither flag): the ABSOLUTE skill DIRECTORY path (s.Dir) — PRD §6.1.
//   - --file:                 the ABSOLUTE SKILL.md file path (s.SourceFile = s.Dir
//   - "/SKILL.md") — PRD §6.2.
//   - --relative:             the chosen path rewritten to be RELATIVE to the
//     skills dir (PRD §6.2 "machine-local convenience").
//   - --file --relative:      they COMBINE — a SKILL.md path relative to the skills
//     dir (e.g. "writing/reddit/SKILL.md").
//
// filepath.Rel cannot fail in practice here: both arguments are ABSOLUTE (s.Dir /
// s.SourceFile are set absolute by discover.Index; skillsDir is absolute from
// skillsdir.Find), and s.Dir is always UNDER skillsDir (it was discovered by
// walking it), so a clean relative path always exists. The err guard is defensive
// only: on a (theoretical) Rel failure skpp falls back to the absolute path, which
// is still a correct, usable answer rather than crashing.
func skillPath(s discover.Skill, skillsDir string, c config) string {
	p := s.Dir // default: absolute skill directory (PRD §6.1/§6.2)
	if c.file {
		p = s.SourceFile // --file: the SKILL.md file path (s.Dir + "/SKILL.md")
	}
	if c.relative {
		if rel, err := filepath.Rel(skillsDir, p); err == nil {
			p = rel // --relative: path relative to the skills dir
		}
	}
	return p
}
