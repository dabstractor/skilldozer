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

	"github.com/dabstractor/skpp/internal/check"
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

// usageText is the full --help / no-args usage block (PRD §6.1, §6.3). It mirrors
// the STRUCTURE of mcpeepants get-server-config.sh (USAGE / EXAMPLES / OPTIONS,
// aligned columns) but lists the full skpp §6 flag matrix and the canonical
// pi --skill "$(skpp <tag>)" one-liner. It is emitted PLAIN (no ANSI):
// `skpp --help | grep` must work, §13 does not assert on help color, and tests
// use non-TTY buffers. The SAME text is printed to stdout for --help (exit 0) and
// to stderr for the no-args default (exit 1) — only the destination differs.
const usageText = `skpp — skill path printer

Resolve skill tags to on-disk skill directory paths (manifest-free).

USAGE:
  skpp <tag> [<tag>...]
  skpp --all
  skpp --list
  skpp --search <query>
  skpp check
  skpp --path
  skpp --help
  skpp --version

EXAMPLES:
  pi --skill "$(skpp example)"
  pi --skill "$(skpp writing/reddit)"
  skpp example reddit          # one absolute path per line, input order
  skpp -f example              # print the SKILL.md path
  skpp --relative --all        # every skill path, relative to the skills dir
  skpp --list                  # human-readable catalog
  skpp --search reddit         # substring search over tag/name/description/keywords
  skpp check                   # validate every skill on disk

OPTIONS:
  <tag> [<tag>...]   Resolve tags to skill directory paths (one absolute path per line)
  --all, -a          Print every skill's directory path, sorted by tag
  --list, -l         Human-readable catalog (TAG, NAME, DESCRIPTION)
  --search <q>, -s   Substring search over tag / name / description / keywords
  check              Validate every skill on disk (report OK / WARN / ERROR)
  --path, -p         Print the resolved skills directory
  --file, -f         Print the SKILL.md path instead of the directory (modifier)
  --relative         Print paths relative to the skills directory (modifier)
  --no-color         Disable ANSI color even on a TTY (modifier)
  --help, -h         Show this help message
  --version, -v      Print the skpp version

Exit codes: 0 success/help/version | 1 unresolved/no skills/unresolvable dir | 2 unknown flag / mutually-exclusive modes
`

// usage returns the help block. A tiny indirection so the constant is wrapped by
// a function (keeps the print sites uniform: fmt.Fprint(w, usage())).
func usage() string { return usageText }

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

// config holds the parsed CLI flags (PRD §6.1/§6.2 matrix). This subtask
// (P1.M5.T11.S1) completes the matrix by adding help and unknownFlag, the final
// two fields needed for the §6.3 precedence + the §6-header unknown-flag rule.
type config struct {
	version     bool     // --version / -v : print "skpp <version>" and exit 0
	help        bool     // --help / -h    : print usage to STDOUT and exit 0 (§6.1, §6.3 "help wins" tiebreak)
	path        bool     // --path / -p    : print resolved skills dir and exit 0/1
	list        bool     // --list / -l    : print the human-readable catalog table (§6.1)
	all         bool     // --all / -a     : print every skill's directory path, one per line (§6.1)
	file        bool     // --file / -f    : print the SKILL.md path instead of the dir path (§6.2)
	relative    bool     // --relative     : print paths relative to the skills dir, not absolute (§6.2)
	noColor     bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	searchMode  bool     // --search <q>/-s : substring search over tag/name/description/keywords (§6.1)
	searchQ     string   // the --search query value (consumed from the token after --search/-s)
	check       bool     // `skpp check` subcommand: validate every skill in the store (§9)
	tags        []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run
	unknownFlag string   // first unknown dashed token, "" if none (§6 header → exit 2)
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
		case "--help", "-h":
			// --help takes precedence over everything else except itself (PRD §6.3
			// "help wins" tiebreak: checked FIRST in run, before --version). Help is
			// emitted PLAIN to stdout, exit 0.
			c.help = true
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
			// value is NOT appended to c.tags (i++ skips it), and it never reaches
			// the default branch, so a dashed value (e.g. `--search -x` → query
			// "-x") is NOT mistaken for an unknown flag. If --search is the LAST
			// token (no value follows) searchMode stays false and the call falls
			// through to the no-recognized-mode default (exit 1).
			if i+1 < len(args) {
				c.searchMode = true
				c.searchQ = args[i+1]
				i++
			}
		case "check":
			// `skpp check` subcommand (PRD §9). `check` is a RESERVED positional
			// token: it selects validation mode and is NOT captured as a tag. A
			// skill literally tagged `check` cannot be resolved via `skpp check`
			// (subcommand names are reserved, as in any CLI). Captured ANYWHERE in
			// argv (so `--no-color check` still selects check); run()'s
			// exclusivity check rejects check+tags / check+mode with exit 2. A
			// nested skill `writing/check` still resolves: this case matches only
			// the EXACT token "check".
			c.check = true
		default:
			// Positional <tag> (PRD §6.1 `skpp <tag> [<tag>...]`): a token that
			// does NOT start with '-' is a tag, captured here and resolved in run.
			// A dashed token NOT in the known set is an unknown flag (PRD §6 header:
			// exit 2): capture the FIRST offender for run() to report. Do NOT collect
			// a slice of unknowns — one loud error is the §6 contract.
			if strings.HasPrefix(a, "-") {
				if c.unknownFlag == "" {
					c.unknownFlag = a
				}
			} else {
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
// Exit codes (PRD §6; final §6.1–§6.4 matrix):
//   - 0: --help printed usage to stdout; --version printed; --path/--list/--search
//     succeeded; all <tag>s resolved; --all printed the store; check passed
//   - 1: --path/--list failed or had no skills; ANY <tag> unresolved/ambiguous
//     (nothing on stdout); skills dir unresolvable; no recognized mode (usage to
//     stderr)
//   - 2: unknown flag; mutually-exclusive modes mixed (tags+mode, check+tags,
//     check+mode)
//
// Precedence (PRD §6.3 "--help / --version take precedence over everything else"
//   - the conventional help-wins tiebreak):
//     help → version → unknownFlag → exclusivity → dispatch → no-args-usage.
func run(args []string, stdout, stderr io.Writer) int {
	c := parseArgs(args)

	// 1) --help takes precedence over EVERYTHING, including --version (the
	//    "help wins" tiebreak) and unknown flags (PRD §6.3). Usage to STDOUT,
	//    exit 0. Help is PLAIN (no ANSI) unconditionally.
	if c.help {
		fmt.Fprint(stdout, usage())
		return 0
	}

	// 2) --version next (PRD §6.3: precedes everything except --help).
	if c.version {
		fmt.Fprintf(stdout, "skpp %s\n", version)
		return 0
	}

	// 3) Unknown dashed flag → exit 2 (PRD §6 header). stdout stays EMPTY (§6.4
	//    discipline: `pi --skill "$(skpp --bogus)"` must fail loudly, not pass a
	//    garbage path). Reported AFTER --help/--version so those still win.
	if c.unknownFlag != "" {
		fmt.Fprintf(stderr, "skpp: unknown flag '%s'\n", c.unknownFlag)
		return 2
	}

	// 4) Mode mutual exclusivity → exit 2 (PRD §6.3). Checked AFTER unknown-flag
	//    so `--bogus foo --list` reports the unknown flag first (both exit 2; the
	//    unknown flag is the more fundamental error). Only three families: tags+
	//    a listing mode (§6.3 explicit); check+tags; check+mode (check ignores
	//    tags so the combo is meaningless — modes are mutually exclusive).
	if bad, msg := exclusivityError(c); bad {
		fmt.Fprintln(stderr, msg)
		return 2
	}

	// 5) Normal mode dispatch (order unchanged): check → path → list →
	//    search → all → tags. Each branch body is byte-identical to
	//    pre-M5 (check is guaranteed standalone here: exclusivity caught
	//    check+tags/check+mode).

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

	// `skpp check` subcommand (PRD §9). Validates every skill in the store and
	// prints a report: one line per problem (prefixed ERROR/WARN) plus one OK line
	// per clean skill, ending with a "N skills, M errors, K warnings" summary. Exit
	// 0 if there are no ERRORs, 1 if there are any (WARNs never change the exit
	// code, so `if skpp check; then …` works as a gate). An empty store is clean
	// (0 skills, 0 errors, 0 warnings) -> exit 0 (check is validation: no skills ==
	// nothing wrong, unlike --list which exits 1 on empty).
	//
	// check is a REPORT, not a path emitter: it always prints its full findings to
	// STDOUT (pipeable to less/grep, like eslint/ruff/govet) and signals pass/fail
	// via the exit code. It is NOT subject to §6.4's "nothing on stdout on failure"
	// — that contract is for tag/path emitters used inside $(...); check never
	// participates in command substitution.
	//
	// internal/check.Check re-runs discover.ParseFrontmatter per skill to recover
	// the malformed-YAML-vs-no-frontmatter-block distinction that discover.Index
	// intentionally drops (index.go doc comment). --file/--relative/--no-color do
	// NOT apply (status report, not paths/table).
	if c.check {
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
		rep := check.Check(skills)
		// Render: status word left-padded to width 5 (OK/WARN/ERROR align); OK
		// skills get one line, problem skills get one line per finding.
		for _, sr := range rep.BySkill {
			name := sr.Skill.Name
			if name == "" {
				name = "(none)"
			}
			if len(sr.Findings) == 0 {
				fmt.Fprintf(stdout, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)
				continue
			}
			for _, f := range sr.Findings {
				fmt.Fprintf(stdout, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)
			}
		}
		fmt.Fprintf(stdout, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)
		if rep.HasErrors() {
			return 1
		}
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

	// No recognized mode → usage to STDERR, exit 1 (PRD §6.3: parity with
	// get-server-config.sh). Covers both truly-no-args and modifiers-only (e.g.
	// `skpp --no-color`): if skpp was asked to DO nothing, show usage. stdout stays
	// empty so $(...) never sees garbage.
	fmt.Fprint(stderr, usage())
	return 1
}

// exclusivityError reports whether c combines modes that PRD §6.3 forbids,
// returning a one-line stderr message. It implements EXACTLY three families:
//   - tags + a listing mode (--list/--search/--all) — PRD §6.3 explicit
//   - check + tags — `check` ignores tags, so the combo is meaningless
//   - check + a listing mode — modes are mutually exclusive
//
// Unspecified combos (e.g. --list --search with no tags) are deliberately NOT
// flagged: PRD §6.3 scopes exclusivity to tag+mode, and mode+mode-without-tags
// resolves deterministically by dispatch order (list wins today). --file/
// --relative/--no-color are MODIFIERS and never trigger exclusivity.
func exclusivityError(c config) (bad bool, msg string) {
	hasTags := len(c.tags) > 0
	if hasTags && (c.list || c.searchMode || c.all) {
		return true, "skpp: tags cannot be combined with --list/--search/--all"
	}
	if c.check && hasTags {
		return true, "skpp: 'check' cannot be combined with tag arguments"
	}
	if c.check && (c.list || c.searchMode || c.all) {
		return true, "skpp: 'check' cannot be combined with --list/--search/--all"
	}
	return false, ""
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
