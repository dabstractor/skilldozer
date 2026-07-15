// Command skilldozer resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// Wired so far (grown milestone by milestone): --version/--path (M1.T3),
// --list (M2.T6), <tag> resolution (M3.T8.S1), and the --file/--relative/--all
// modifiers (M3.T8.S2). Every other §6 flag is added by later milestones (M4
// --search/check, M5 --help + exit codes). The arg parser is intentionally a
// small hand-rolled switch (not Go's `flag` package) so the full §6 matrix —
// flags like `--check`, positional <tag> args, long+short aliases, and
// §6.3 mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1T3.S1/research/verified_facts.md §5.
package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dabstractor/skilldozer/internal/check"
	configpkg "github.com/dabstractor/skilldozer/internal/config"
	"github.com/dabstractor/skilldozer/internal/discover"
	"github.com/dabstractor/skilldozer/internal/resolve"
	"github.com/dabstractor/skilldozer/internal/search"
	"github.com/dabstractor/skilldozer/internal/skillsdir"
	"github.com/dabstractor/skilldozer/internal/ui"
)

// version is the skilldozer version string, printed by `skilldozer --version`. It is
// overridden at BUILD time via ldflags (PRD §12.1 build command):
//
//	go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skilldozer .
//
// The default "dev" is used by `go run` and plain `go build` (no ldflags).
//
// IMPORTANT: this MUST be a package-level var, not a const. `-X main.version=...`
// rewrites a package-scope string var at link time; it cannot override a const
// (compile error) or a function-local. Because this file is `package main`, the
// linker symbol path is `main.version` (NOT the module import path).
var version = "dev"

// Embedded shell-completion scripts (PRD §14.6 "Embedding (self-sufficient binary)"). The
// on-disk completions/* files are the single source of truth; these vars embed them verbatim
// so `completion` works for go install users with no repo clone (no runtime file I/O). embed is
// stdlib (Go 1.16+; no new dependency). Three string vars (not embed.FS): the files are static
// and known, so a trivial switch in completionScript is simpler than runtime ReadFile lookups.
// NOTE: completions/_skilldozer embeds WITHOUT an `all:` prefix — Go's _/. exclusion applies
// only to directory-walk patterns, not explicit file paths (verified).
//
//go:embed completions/skilldozer.bash
var bashCompletion string

//go:embed completions/_skilldozer
var zshCompletion string

//go:embed completions/skilldozer.fish
var fishCompletion string

// usageText is the full --help / no-args usage block (PRD §6.1, §6.3). It mirrors
// the STRUCTURE of mcpeepants get-server-config.sh (USAGE / EXAMPLES / OPTIONS,
// aligned columns) but lists the full skilldozer §6 flag matrix and the canonical
// pi --skill "$(skilldozer <tag>)" one-liner. It is emitted PLAIN (no ANSI):
// `skilldozer --help | grep` must work, §13 does not assert on help color, and tests
// use non-TTY buffers. The SAME text is printed to stdout for --help (exit 0) and
// for the no-args/modifiers-only default (exit 0 — implicit --help, PRD §6.3 / §19
// decision 17); genuine failures go to stderr (§6.4), never this text.
const usageText = `skilldozer — skill path printer

Resolve skill tags to on-disk skill directory paths (manifest-free).

USAGE:
  skilldozer <tag> [<tag>...]
  skilldozer --all
  skilldozer --list
  skilldozer --search <query>
  skilldozer --check
  skilldozer --init [<dir>]
  skilldozer --completions [--shell <name>]
  skilldozer --path
  skilldozer --help
  skilldozer --version

EXAMPLES:
  pi --skill "$(skilldozer example)"
  pi --skill "$(skilldozer writing/reddit)"
  skilldozer example reddit          # one absolute path per line, input order
  skilldozer -f example              # print the SKILL.md path
  skilldozer --relative --all        # every skill path, relative to the skills dir
  skilldozer --list                  # human-readable catalog
  skilldozer --search reddit         # substring search over tag/name/description/keywords/aliases/category
  skilldozer --check                   # validate every skill on disk
  skilldozer --init --store <dir>      # non-interactive first-run setup
  eval "$(skilldozer --completions)"   # load completions into your shell

Help and --completions advertise long forms only; short aliases (-a, -l, -s, -f, -p, -h, -v)
remain valid for typing but are not advertised (§6.1).

OPTIONS:
  <tag> [<tag>...]             Resolve tags to skill directory paths (one absolute path per line)
  --all, -a                    Print every skill's directory path, sorted by tag
  --list, -l                   Human-readable catalog (TAG, NAME, DESCRIPTION)
  --search <q>, -s             Substring search over tag / name / description / keywords / aliases / category
  --check                      Validate every skill on disk (report OK / WARN / ERROR)
  --init [<dir>]               First-run setup: pick/create the skills store and write the config
  --store <dir>                Non-interactive store path for init
  --completions [--shell <name>]   Emit the shell completion script for eval (§14.6)
  --shell <bash|zsh|fish>      Force a shell for completion (else auto-detect)
  --path, -p                   Print the resolved skills directory (discovery rule printed to stderr)
  --file, -f                   Print the SKILL.md path instead of the directory (modifier)
  --relative                   Print paths relative to the skills directory (modifier)
  --no-color                   Disable ANSI color even on a TTY (modifier)
  --help, -h                   Show this help message
  --version, -v                Print the skilldozer version

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
	version           bool     // --version / -v : print "skilldozer <version>" and exit 0
	help              bool     // --help / -h    : print usage to STDOUT and exit 0 (§6.1, §6.3 "help wins" tiebreak)
	path              bool     // --path / -p    : print resolved skills dir and exit 0/1
	list              bool     // --list / -l    : print the human-readable catalog table (§6.1)
	all               bool     // --all / -a     : print every skill's directory path, one per line (§6.1)
	file              bool     // --file / -f    : print the SKILL.md path instead of the dir path (§6.2)
	relative          bool     // --relative     : print paths relative to the skills dir, not absolute (§6.2)
	noColor           bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	searchMode        bool     // --search <q>/-s : substring search over tag/name/description/keywords/aliases/category (§10)
	searchQ           string   // the --search query value (consumed from the token after --search/-s)
	check             bool     // `skilldozer --check` flag: validate every skill in the store (§9)
	init              bool     // `skilldozer --init [<dir>]` flag, first-run setup (PRD §8.2); also set by `--store <dir>` (which implies init)
	initStore         string   // non-interactive store path: `--init <dir>` flag or `--store <dir>` / `--store=<dir>`; empty ⇒ auto-detect (P1.M2.T2.S3)
	storeMissingValue bool     // --store / --store= seen with NO value (Issue 2); run() rejects with exit 2 before init dispatch (config NOT written) in P1.M1.T2.S2. NOT set by bare `init` (c.initStore=="" ⇒ prompt).
	completion        bool     // `skilldozer --completions` flag (PRD §14.6); exclusive like check/init; also set by `--shell <name>` (which implies completion)
	completionShell   string   // `--shell <bash|zsh|fish>` value (PRD §14.6); "" ⇒ detect from $SKILLDOZER_SHELL/$SHELL (P1.M2.T2.S2)
	tags              []string // positional <tag> args (PRD §6.1 `skilldozer <tag> [<tag>...]`); resolved in run
	unknownFlag       string   // first unknown dashed token, "" if none (§6 header → exit 2)
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

		// Issue 5 (decisions.md §D5): normalize combined / '='-bearing tokens
		// BEFORE the exact-match switch so POSIX forms work. Each branch ends in
		// `continue`; the switch below still handles the original exact-token forms
		// (--version, -v, --search <q>, check, bare tags, and unknowns like -x).

		// (a) Long flag with '=': --flag=value. Split on the FIRST '='; bool flags
		// ignore the value (--version=x == --version), --search takes it as the
		// query, an unknown name is an unknown flag (the whole token is reported).
		if strings.HasPrefix(a, "--") && strings.Contains(a, "=") {
			eq := strings.IndexByte(a, '=')
			name, val := a[:eq], a[eq+1:]
			switch name {
			case "--version":
				c.version = true
			case "--help":
				c.help = true
			case "--path":
				c.path = true
			case "--list":
				c.list = true
			case "--all":
				c.all = true
			case "--file":
				c.file = true
			case "--relative":
				c.relative = true
			case "--no-color":
				c.noColor = true
			case "--search":
				c.searchMode = true
				c.searchQ = val
			case "--store":
				// `--store=<dir>`: non-interactive store path for init (PRD §8.2). Mirrors
				// --search's '='-form; implies init mode (c.init=true). No short form.
				// Empty value (--store=) records storeMissingValue so run() (P1.M1.T2.S2)
				// rejects with exit 2 before dispatch.
				c.init = true
				c.initStore = val
				if val == "" {
					c.storeMissingValue = true
				}
			case "--init":
				// `--init=<dir>`: non-interactive store path (mirrors --store=). Empty value (--init=)
				// records storeMissingValue so run() rejects with exit 2 (Issue 2).
				c.init = true
				c.initStore = val
				if val == "" {
					c.storeMissingValue = true
				}
			case "--check":
				c.check = true
			case "--completions":
				c.completion = true
			case "--shell":
				// `--shell=<name>`: force a shell for completion (PRD §14.6). Mirrors --store's '='-form;
				// implies completion mode (c.completion=true). No short form. NO empty-value guard (PRD §6.4
				// specifies no missing-value exit code for --shell — `--shell=` is completion=true, shell="").
				c.completion = true
				c.completionShell = val
			default:
				if c.unknownFlag == "" {
					c.unknownFlag = a
				}
			}
			continue
		}

		// (b) Short bundle: -xyz (single '-', not "--", len > 2). Expand into the
		// individual short flags; -s (value-taking) may consume the next token.
		// len-2 shorts ("-v", "-s", ...) and "--..." longs fall through to the switch.
		if len(a) > 2 && a[0] == '-' && a[1] != '-' {
			if consumeNext, _ := expandShortBundle(&c, a, args, i); consumeNext {
				i++ // -s took its value from the next argv token
			}
			continue
		}

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
		case "--check":
			// `skilldozer --check` flag (PRD §9). A bare `check` is a skill TAG (decision 19).
			c.check = true
		case "--completions":
			// `skilldozer --completions [--shell <name>]` flag (PRD §14.6). PLURAL (decision 19).
			// A bare `completion`/`completions` is a skill tag, not this flag.
			c.completion = true
		case "--store":
			// `--store <dir>`: non-interactive store path for init (PRD §8.2). Mirrors
			// --search's next-token capture; implies init mode (c.init=true) when a
			// value follows. No short form. If --store is the LAST token (no value
			// follows), record storeMissingValue so run() rejects with exit 2
			// (P1.M1.T2.S2) instead of silently no-op'ing into destructive auto-detect
			// init when an `init` token already set c.init (Issue 2).
			if i+1 < len(args) {
				c.init = true
				c.initStore = args[i+1]
				i++
			} else {
				c.storeMissingValue = true
			}
		case "--shell":
			// `--shell <name>`: force a shell for completion (PRD §14.6). Mirrors --store's next-token
			// capture; implies completion mode (c.completion=true) when a value follows. No short form.
			// If --shell is the LAST token (no value), completion stays false — mirrors --search's
			// no-value silent behavior (PRD §6.4 specifies no missing-value exit code for --shell).
			if i+1 < len(args) {
				c.completion = true
				c.completionShell = args[i+1]
				i++
			}
		case "--init":
			// `skilldozer --init [<dir>]` first-run setup (PRD §8.2). --init is the sole mode
			// accepting a positional <dir> (the store to adopt, §6.3): a following NON-dashed
			// token is c.initStore. A dashed follower (--init --store …) is left for its own
			// case. A bare `init` is a skill tag (decision 19).
			c.init = true
			if i+1 < len(args) {
				next := args[i+1]
				if !strings.HasPrefix(next, "-") {
					c.initStore = next
					i++
				}
			}
		default:
			// Positional <tag> (PRD §6.1 `skilldozer <tag> [<tag>...]`): a token that
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

// expandShortBundle parses a combined short-flag token `a` (e.g. "-vh", "-pl",
// "-sfoo", "-ls") and applies the resulting flags to *c. It implements Issue 5's
// short-bundle normalization (decisions.md §D5). The caller has already guaranteed
// `a` is bundle-shaped: a single leading '-', not "--", and len(a) > 2.
//
// Semantics (PRD §6 short forms; the short set is exactly v h p l a f s):
//   - v/h/p/l/a/f are BOOL flags; each sets its config field.
//   - s is the VALUE-TAKING flag (--search): once seen, the rest of the body is
//     the query (e.g. "-sfoo" -> "foo"); if the rest is empty, the NEXT argv
//     token is consumed as the query (e.g. "-ls foo" -> list + query "foo"), and
//     the caller advances i (returns consumeNext=true). If no value is available
//     at all (empty rest AND no next arg), searchMode stays false — mirroring the
//     bare "-s"-with-no-value rule in the main switch.
//   - any char that is NEITHER a bool flag NOR the leading 's' is UNKNOWN: the
//     WHOLE bundle is rejected — c.unknownFlag is set to `a` and NOTHING is
//     applied. This two-phase (validate-then-commit) design is REQUIRED because
//     run() checks unknownFlag AFTER version/help: a leaked `version=true` from a
//     partial "-vz" would make run() print the version (exit 0) and mask the
//     unknown-char error.
//
// Returns (consumeNext, ok). ok is always true for a bundle-shaped token (it was
// handled, validly or as-unknown). consumeNext=true tells the caller to i++ (the
// -s value came from the next argv token).
func expandShortBundle(c *config, a string, args []string, i int) (consumeNext, ok bool) {
	body := a[1:] // strip the single leading '-'

	// Phase 1 — validate. Walk bool flags left-to-right; the FIRST non-bool char
	// must be 's' (then the rest is the query) or it is unknown. Record where 's'
	// sits (sIdx) so Phase 2 knows where flags end and the query begins.
	sIdx := -1
	for j := 0; j < len(body); j++ {
		ch := body[j]
		if ch == 's' {
			sIdx = j
			break // 's' ends flag parsing; body[j+1:] is the query
		}
		switch ch {
		case 'v', 'h', 'p', 'l', 'a', 'f':
			// valid bool short flag (validated here; applied in Phase 2)
		default:
			// Unknown char: reject the WHOLE bundle. Commit nothing (two-phase).
			if c.unknownFlag == "" {
				c.unknownFlag = a
			}
			return false, true
		}
	}

	// Phase 2 — commit the bool flags in [0, sIdx) (or the whole body if no 's').
	end := len(body)
	if sIdx >= 0 {
		end = sIdx
	}
	for j := 0; j < end; j++ {
		switch body[j] {
		case 'v':
			c.version = true
		case 'h':
			c.help = true
		case 'p':
			c.path = true
		case 'l':
			c.list = true
		case 'a':
			c.all = true
		case 'f':
			c.file = true
		}
	}

	// Handle the value-taking 's' if it was present.
	if sIdx >= 0 {
		remainder := body[sIdx+1:]
		switch {
		case remainder != "":
			c.searchMode = true
			c.searchQ = remainder // value embedded in the bundle ("-sfoo")
		case i+1 < len(args):
			c.searchMode = true
			c.searchQ = args[i+1] // value is the next argv token ("-ls foo")
			return true, true     // caller advances i
		default:
			// 's' seen but no value anywhere: mirror the bare "-s"-no-value rule
			// (searchMode stays false). The bool flags before it remain set.
		}
	}
	return false, true
}

// run is the testable dispatcher. It returns the process exit code so main() can
// call os.Exit(run(...)) without tests ever invoking os.Exit. stdout/stderr are
// injected so tests capture output via *bytes.Buffer instead of the real streams.
//
// Exit codes (PRD §6; final §6.1–§6.4 matrix):
//   - 0: --help printed usage to stdout; --version printed; --path/--list/--search
//     succeeded; all <tag>s resolved; --all printed the store; check passed;
//     no-args/modifiers-only printed usage to stdout (implicit --help, §6.3)
//   - 1: --path/--list failed or had no skills; ANY <tag> unresolved/ambiguous
//     (nothing on stdout); skills dir unresolvable
//   - 2: unknown flag; mutually-exclusive modes mixed (tags+mode, check+tags,
//     check+mode)
//
// Precedence (PRD §6.3 "--help / --version take precedence over everything else"
//   - the conventional help-wins tiebreak):
//     help → version → unknownFlag → storeMissingValue (--store requires a value) → exclusivity → dispatch → no-args-usage.
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
		fmt.Fprintf(stdout, "skilldozer %s\n", version)
		return 0
	}

	// 3) Unknown dashed flag → exit 2 (PRD §6 header). stdout stays EMPTY (§6.4
	//    discipline: `pi --skill "$(skilldozer --bogus)"` must fail loudly, not pass a
	//    garbage path). Reported AFTER --help/--version so those still win.
	if c.unknownFlag != "" {
		fmt.Fprintf(stderr, "skilldozer: unknown flag '%s'\n", c.unknownFlag)
		return 2
	}

	// 3.5) --store presented without its value → exit 2 (PRD §6 header "Unknown flags
	//      ⇒ error + exit 2"; delta-PRD §2 #3 "init is non-destructive"). A value-taking
	//      flag with no value is a parse error, NOT a silent fall-through to destructive
	//      auto-detect init. Rejecting here — BEFORE the init dispatch — means runInit /
	//      setupStore / configpkg.Save is NEVER called, so a pre-existing config.yaml's
	//      `store:` value is preserved (Issue 2). stdout stays EMPTY (§6.4 discipline).
	//      The signal is set by parseArgs in BOTH --store no-value branches (P1.M1.T2.S1);
	//      it is NOT set by bare `init` (c.initStore=="" legitimately means "prompt").
	if c.storeMissingValue {
		fmt.Fprintln(stderr, "skilldozer: --store requires a value")
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

	// init dispatch (PRD §8.2). init is an exclusive mode: exclusivityError
	// above guarantees no other mode is set when c.init is true, so this
	// self-contained branch returns before the path/list/search/check/all/tags
	// ladder below. runInit orchestrates resolveStore → config.Path →
	// setupStore, then prints the --path rendering + the check report (§8.2
	// step 5). The bare-tag path (c.tags) is untouched and never prompts (§6.4).
	if c.init {
		return runInit(c, stdout, stderr)
	}

	// completion dispatch (PRD §14.6). completion is an exclusive mode (like
	// check/init): exclusivityError above guarantees no other mode is set when
	// c.completion is true, so this self-contained branch returns before the
	// path/list/search/check/all/tags ladder below. runCompletion resolves the
	// shell (--shell → $SKILLDOZER_SHELL → basename($SHELL)), emits the matching
	// embedded script to stdout (exit 0), or exits 1 (undetectable) / 2
	// (unsupported value) with nothing on stdout (PRD §6.4).
	if c.completion {
		return runCompletion(c, stdout, stderr)
	}

	// 5) Normal mode dispatch (order: path → list → search → check → all →
	//    tags). Each branch body is byte-identical to pre-M5 (any mode that
	//    reaches here is guaranteed standalone: exclusivityError caught
	//    mode+mode/check+tags/check+mode above).

	if c.path {
		dir, src, err := skillsdir.Find()
		if err != nil {
			// Find() returns skillsdir.ErrNotFound whose message is the
			// user-facing one-line fix (PRD §8.4/§6.4). Print it verbatim to
			// stderr (NOT stdout) so $(...) stays empty on failure.
			fmt.Fprintln(stderr, err)
			return 1
		}
		// Byte-exact: ONLY the dir + newline on stdout. The §13 acceptance gate
		// `test "$(./skilldozer --path)" = "$PWD/skills"` depends on this — $() captures
		// stdout only, so the stderr label below does NOT break it.
		fmt.Fprintln(stdout, dir)
		// Issue 1 (QA): report which §8 discovery rule won, to stderr. A typo'd
		// SKILLDOZER_SKILLS_DIR silently falls through to sibling/walk-up; this label
		// makes that visible without polluting stdout. Labels from Source.String().
		fmt.Fprintf(stderr, "(found via %s)\n", src)
		return 0
	}

	if c.list {
		// PRD §6.1 `skilldozer --list`: resolve the store, build the index, render the
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

	// --search mode: `skilldozer --search <q>` / `-s <q>` (PRD §10). Filters the index to
	// skills where <q> is a case-insensitive substring of the tag, frontmatter name,
	// description, any metadata keyword, any metadata alias, or the metadata category
	// (internal/search), then renders the SAME
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

	// `skilldozer --check` flag (PRD §9). Validates every skill in the store and
	// prints a report: one line per problem (prefixed ERROR/WARN) plus one OK line
	// per clean skill, ending with a "N skills, M errors, K warnings" summary. Exit
	// 0 if there are no ERRORs, 1 if there are any (WARNs never change the exit
	// code, so `if skilldozer --check; then …` works as a gate). An empty store is clean
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

	// Tag-resolution mode: `skilldozer <tag> [<tag>...]` (PRD §6.1). Resolves each tag to
	// its skill path and prints one path per line, in input order.
	//
	// ATOMICITY (PRD §6.4 — the critical-for-$(...) contract): resolve EVERY tag
	// first, buffering the resulting paths; if ANY tag fails (unknown/ambiguous),
	// print one error line per problem tag to stderr, print NOTHING to stdout, and
	// exit 1. The buffered paths are flushed ONLY when the whole invocation is
	// known-good. This makes `pi --skill "$(skilldozer bad)"` fail loudly (empty $(),
	// exit 1) instead of passing a partial or garbage path. Each error is printed
	// verbatim from resolve's typed errors — UnknownError names the tag,
	// AmbiguousError lists the candidate full tags (no "skilldozer:" prefix, matching the
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

	// No recognized mode → usage to STDOUT, exit 0 (PRD §6.3 / §19 decision 17:
	// bare invocation is implicit --help). Covers both truly-no-args and
	// modifiers-only (e.g. `skilldozer --no-color`): if skilldozer was asked to DO
	// nothing, show usage. The help lands on stdout so `skilldozer | grep …` works;
	// genuine failures stay on stderr (§6.4).
	fmt.Fprint(stdout, usage())
	return 0
}

// exclusivityError reports whether c combines modes that PRD §6.3 forbids,
// returning a one-line stderr message. It implements four families, checked in
// order (first hit wins):
//   - two or more listing modes among {--path, --list, --search, --all} — Issue 6
//     (any 2+ are mutually exclusive; the previous silent dispatch precedence was
//     surprising)
//   - tags + an inspection mode (--path/--list/--search/--all) — PRD §6.3 (Issue 3:
//     --path was omitted, silently dropping a stray tag; now uniform with the
//     --check+mode and mode+mode sets)
//   - --check + tags — `--check` ignores tags, so the combo is meaningless
//   - --check + a listing mode — modes are mutually exclusive
//
// `--check` is NOT in the listing-mode set: --check+mode is caught by the families
// below (and --check+path, too — it used to silently resolve by dispatch order
// with path winning, which was inconsistent with --check+list/--check+search/
// --check+all all exiting 2; N1 closed that asymmetry). Issue 3 (P1.M2.T1.S1)
// closed the identical asymmetry for tags+--path. --file/--relative/
// --no-color are MODIFIERS and never trigger exclusivity (they combine with a
// single mode, e.g. `--all --file`).
func exclusivityError(c config) (bad bool, msg string) {
	// Issue 6 (decisions.md §D6): any 2+ of the listing modes are mutually
	// exclusive. Count the active ones; >= 2 is an error. Checked FIRST so a
	// mode+mode combo gets the precise "listing modes" message even when tags are
	// also present. The set is exactly {path, list, searchMode, all}; check and the
	// modifiers are intentionally excluded (see the doc comment).
	n := 0
	for _, b := range []bool{c.path, c.list, c.searchMode, c.all} {
		if b {
			n++
		}
	}
	if n >= 2 {
		return true, "skilldozer: listing modes --path/--list/--search/--all are mutually exclusive"
	}
	hasTags := len(c.tags) > 0
	if hasTags && (c.path || c.list || c.searchMode || c.all) {
		return true, "skilldozer: tags cannot be combined with --path/--list/--search/--all"
	}
	if c.check && hasTags {
		return true, "skilldozer: '--check' cannot be combined with tag arguments"
	}
	if c.check && (c.completion || c.path || c.list || c.searchMode || c.all) {
		return true, "skilldozer: '--check' cannot be combined with --completions/--path/--list/--search/--all"
	}
	// --init is its own exclusive mode (PRD §6.3 / §8.2: like `--check`). It rejects the
	// listing/inspection modes AND stray tags. A single positional <dir> after `--init`
	// is consumed as the store (c.initStore) by parseArgs, so it never reaches
	// c.tags; a SECOND positional, or any positional after `--init --store`, lands in
	// c.tags and is rejected here as a stray.
	if c.init {
		if hasTags {
			return true, "skilldozer: '--init' cannot be combined with tag arguments"
		}
		if c.check || c.completion || c.list || c.searchMode || c.all || c.path {
			return true, "skilldozer: '--init' cannot be combined with --check/--completions/--list/--search/--all/--path"
		}
	}
	// --completions is its own exclusive mode (PRD §6.3 / §14.6: like --check/--init). It rejects the other
	// mode flags AND stray tags. `--completions` does no positional capture (mirrors --check), so
	// any positional after it lands in c.tags and is rejected here as a stray.
	if c.completion {
		if hasTags {
			return true, "skilldozer: '--completions' cannot be combined with tag arguments"
		}
		if c.check || c.init || c.list || c.searchMode || c.all || c.path {
			return true, "skilldozer: '--completions' cannot be combined with --check/--init/--path/--list/--search/--all"
		}
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
// only: on a (theoretical) Rel failure skilldozer falls back to the absolute path, which
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

// stdinIsTerminal reports whether os.Stdin is an interactive terminal. It is
// the stdin counterpart of the stdout isTerminal check (PRD §6.2 color gating,
// main.go:96-112): the SAME ModeCharDevice technique applied to a DIFFERENT
// stream (os.Stdin, not an io.Writer). init uses it to gate the interactive
// prompt so piped/redirected invocations never block (PRD §8.2 prompt safety).
//
// It is a plain function (not a package var) because init's test seam is
// chooseStore's isTTY PARAMETER, not a global override — see chooseStore.
// Caveat (harmless): /dev/null is also a char device, so this reports true for
// `init < /dev/null`; the immediate EOF there makes readPrompt return the
// default, so it never hangs. No golang.org/x/term (yaml.v3 stays the sole
// non-stdlib dep — external_deps.md §3).
func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// readPrompt prints the prompt (label, with [def] in brackets) to w, reads one
// line from r, and returns the trimmed answer — or def when the user just presses
// Enter (empty line) or sends EOF on an otherwise-empty line. A genuine read
// error (non-EOF) is returned. Used by init's interactive prompt (PRD §8.2).
// (external_deps.md §4 prescribes bufio.Reader.ReadString('\n') over Scanner.)
func readPrompt(r *bufio.Reader, w io.Writer, label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(w, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(w, "%s: ", label)
	}
	line, err := r.ReadString('\n') // includes the trailing '\n'
	if err != nil && err != io.EOF {
		return "", err
	}
	if s := strings.TrimSpace(line); s != "" {
		return s, nil
	}
	return def, nil // empty Enter OR EOF-with-no-text ⇒ accept default
}

// chooseStore resolves the store directory for `skilldozer --init` (PRD §8.2) via a
// 4-step decision that is fully independent of os.Stdin/os.Stdout/os.Getwd: the
// caller injects cwd, isTTY, the default store, and a prompt function, so the
// logic is unit-testable without a real terminal (the contract FACTORING).
//
// Resolution order (first applicable wins):
//  1. haveStore != "" — the non-interactive override from `init <dir>` or
//     `--store <dir>`. Returned VERBATIM; the prompt is NEVER called (scripts/CI).
//  2. auto-detect the default: if cwd already looks like a store (it contains at
//     least one SKILL.md at any depth — skillsdir.HasSkillMD, PRD §8.2 "detected
//     skills in <cwd>"), default = cwd; else default = defaultStore (the
//     $XDG_DATA_HOME/skilldozer/skills value from config.DefaultStore).
//  3. isTTY — prompt "Where should skilldozer keep your skills? [<default>]".
//     readPrompt makes empty line / EOF ⇒ default; a typed path ⇒ override.
//  4. !isTTY and no explicit haveStore — return the auto-detected default with NO
//     prompt (scripts / CI / pipes). The prompt is NEVER called.
//
// The chosen string is returned VERBATIM (it may be relative if the user typed a
// relative path); resolveStore absolutizes it via filepath.Abs. A non-nil error is
// returned ONLY on a genuine prompt read failure (a non-EOF error from the prompt
// fn); empty/EOF is "accept default", never an error.
func chooseStore(haveStore, cwd string, isTTY bool, defaultStore string, prompt func(label, def string) (string, error)) (string, error) {
	// (1) Non-interactive override: `init <dir>` / `--store <dir>`. No prompt.
	if haveStore != "" {
		return haveStore, nil
	}
	// (2) Auto-detect the default from cwd (PRD §8.2 "detected skills in <cwd>").
	def := defaultStore
	if skillsdir.HasSkillMD(cwd) {
		def = cwd
	}
	// (4) Off-TTY (pipe/file/CI): use the default, NO prompt (never blocks).
	if !isTTY {
		return def, nil
	}
	// (3) Interactive: prompt. Empty/EOF answer ⇒ def (the auto-detected default);
	// a typed path ⇒ override (returned verbatim). A genuine read error propagates.
	// readPrompt performs the empty⇒default translation for the real reader; chooseStore
	// applies the SAME rule to the prompt fn's return so the decision core is
	// self-contained regardless of which prompt fn is injected (contract §8.2).
	choice, err := prompt("Where should skilldozer keep your skills?", def)
	if err != nil {
		return "", err
	}
	if choice == "" {
		return def, nil
	}
	return choice, nil
}

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

// resolveStore is the I/O-bearing wrapper around chooseStore that run()'s init
// dispatch (P1.M2.T2.S3) calls. It supplies the real dependencies — os.Getwd(),
// config.DefaultStore(), the os.Stdin TTY check (stdinIsTerminal), and a bufio
// prompt reader over os.Stdin/os.Stderr (readPrompt) — and returns chooseStore's
// choice with a leading "~"/"~/" expanded to $HOME (expandHome, Issue 5) and then
// ABSOLUTIZED via filepath.Abs (PRD §8.2 "absolute store path"). filepath.Abs alone
// does not expand "~", so expandHome runs first — otherwise `init ~/x`, `--store ~/x`,
// `--store=~/x`, and the interactive prompt path would all store "<cwd>/~/.x". The ONE
// shared bufio.NewReader is created here and captured by the prompt closure so a
// future second prompt would reuse it (external_deps.md §4: a fresh reader per
// prompt can swallow buffered bytes).
//
// The os.Stdin / os.Stderr / os.Getwd access is confined to THIS function so the
// pure decision logic in chooseStore stays terminal-free and unit-testable. The
// prompt is written to stderr (not stdout) so init's stdout stays the bare store
// path per PRD §6.1. A genuine cwd/default/absolutize/prompt error is returned
// wrapped; an empty or EOF prompt answer is NOT an error (readPrompt ⇒ default).
func resolveStore(haveStore string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("skilldozer --init: resolve cwd: %w", err)
	}
	def, err := configpkg.DefaultStore()
	if err != nil {
		return "", fmt.Errorf("skilldozer --init: resolve default store: %w", err)
	}
	r := bufio.NewReader(os.Stdin)
	// Prompt dialog goes to stderr, not stdout. PRD §6.1 pins init's stdout to
	// exactly the configured store path; the interactive "Where should skilldozer
	// keep your skills? [default]:" line is user-facing prose, not result data,
	// so a caller doing store="$(skilldozer --init)" must not capture it.
	prompt := func(label, def string) (string, error) {
		return readPrompt(r, os.Stderr, label, def)
	}
	store, err := chooseStore(haveStore, cwd, stdinIsTerminal(), def, prompt)
	if err != nil {
		return "", err
	}
	// Expand a leading "~"/"~/" to $HOME BEFORE filepath.Abs (Issue 5). filepath.Abs
	// does not expand "~", so without this `init ~/x`, `--store ~/x`, `--store=~/x`,
	// and the interactive typed path would all store "<cwd>/~/x" and create a directory
	// literally named "~". expandHome is a no-op for absolute/tilde-free paths and
	// returns the input unchanged when $HOME is unset (fail safe).
	store = expandHome(store)
	abs, err := filepath.Abs(store)
	if err != nil {
		return "", fmt.Errorf("skilldozer --init: absolutize store: %w", err)
	}
	return abs, nil
}

// exampleSkillTemplate is the PRD §11 example skill body, compiled into the binary as a
// STRING CONSTANT (NOT go:embed — PRD §17 "nothing about the user's collection is compiled
// in"; code_prd_delta.md G11). skilldozer --init writes this verbatim into an EMPTY store's
// example/SKILL.md (PRD §8.2 step 3). NOTE: there is a SECOND copy of this exact text on
// disk at skills/example/SKILL.md (P1.M3.T1.S1's repo asset); both MUST equal PRD §11.
//
// Raw literals can't hold backticks; the §11 body has 8 (2 inline `skilldozer` + the
// ```bash fence). Splice double-quoted backtick runs between raw segments via `+`.
const exampleSkillTemplate = `---
name: example
description: >
  Reference example skill for skilldozer. Demonstrates the required frontmatter and
  how skilldozer resolves a tag to an absolute path. Safe to delete once you add real skills.
metadata:
  keywords: [example, demo, skilldozer]
  category: meta
---

# Example Skill

This skill exists only so ` + "`skilldozer`" + ` has something to resolve.

Try:

` + "```bash" + `
skilldozer example                       # prints this directory's absolute path
skilldozer -f example                    # prints .../skills/example/SKILL.md
pi --skill "$(skilldozer example)"       # loads this skill into pi
` + "```" + `
`

// setupStore creates the skills store, seeds it if empty, and writes the config. It is
// the create+seed+writeconfig half of `skilldozer --init` (PRD §8.2 steps 2-4); the
// store-CHOICE half is resolveStore (P1.M2.T2.S1), and run()'s `if c.init` dispatch
// (P1.M2.T2.S3) calls both. Both targets are INJECTED as strings (store is already
// absolute — resolveStore absolutized it; configPath is config.Path()'s result from
// run()), so this function is directly unit-testable with temp paths and needs no
// separate wrapper layer.
//
// Steps:
//
//	(a) os.MkdirAll(store, 0o755) — create the store dir if missing (PRD §8.2 step 2).
//	(b) os.ReadDir(store): if the dir is EMPTY (zero entries of any kind), seed
//	    example/SKILL.md from the compiled-in exampleSkillTemplate (PRD §8.2 step 3,
//	    §11) — MkdirAll(store/example) then WriteFile; seeded=true. "Empty" means no
//	    entries at all, NOT "no SKILL.md" (a single pre-existing file ⇒ adopt).
//	(c) If the store already contains ANYTHING, adopt it in place: NEVER clobber or
//	    delete existing files (PRD §17 guardrail); seeded stays false.
//	(d) config.Save(configPath, config.File{Store: store}) — write the config with the
//	    absolute store path (PRD §8.2 step 4). ALWAYS runs, whether seeded or adopted.
//
// Returns (seeded, nil) on success, or (false, err) on any fs failure — `seeded` is a
// SUCCESS-PATH signal (run()/S3 prints "seeded" vs "adopted"); callers MUST check err
// before reading seeded, so a config.Save failure after a successful seed still returns
// (false, err). Errors are wrapped with a "skilldozer --init: <step>: %w" prefix.
func setupStore(store, configPath string) (seeded bool, err error) {
	// (a) Ensure the store dir exists (idempotent — no-op if present).
	if err := os.MkdirAll(store, 0o755); err != nil {
		return false, fmt.Errorf("skilldozer --init: create store dir %q: %w", store, err)
	}
	// (b) Seed only if the store is EMPTY (zero entries of any kind).
	entries, err := os.ReadDir(store)
	if err != nil {
		return false, fmt.Errorf("skilldozer --init: read store dir %q: %w", store, err)
	}
	if len(entries) == 0 {
		exampleDir := filepath.Join(store, "example")
		if err := os.MkdirAll(exampleDir, 0o755); err != nil {
			return false, fmt.Errorf("skilldozer --init: create example dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(exampleDir, "SKILL.md"), []byte(exampleSkillTemplate), 0o644); err != nil {
			return false, fmt.Errorf("skilldozer --init: seed example SKILL.md: %w", err)
		}
		seeded = true
	}
	// (c) Non-empty: adopt in place. Do NOTHING to existing files (PRD §17). seeded stays false.
	// (d) Always write the config with the (already-absolute) store path.
	if err := configpkg.Save(configPath, configpkg.File{Store: store}); err != nil {
		return false, fmt.Errorf("skilldozer --init: write config %q: %w", configPath, err)
	}
	return seeded, nil
}

// completionScript returns the embedded shell-completion script for shell ("bash"/"zsh"/
// "fish") and true; for any other shell it returns ("", false). It is a pure switch over the
// package-scope //go:embed vars (PRD §14.6) — no filesystem access. The bool lets runCompletion
// (P1.M2.T2.S2) distinguish a known shell from an unsupported one (the §6.4 exit-2 path).
// The bytes are verbatim from completions/* (the single source of truth); see
// TestEmbeddedCompletionsMatchOnDisk for the byte-identity lock. runCompletion emits these
// verbatim for bash/fish but DERIVES the eval-safe zsh wrapper via zshEvalScript — the
// completionScript return value stays byte-identical to the on-disk autoload file.
func completionScript(shell string) (string, bool) {
	switch shell {
	case "bash":
		return bashCompletion, true
	case "zsh":
		return zshCompletion, true
	case "fish":
		return fishCompletion, true
	}
	return "", false
}

// zshEvalScript derives an eval-safe zsh completion wrapper from the verbatim autoload file
// (the embedded zshCompletion / completions/_skilldozer). The on-disk file is an autoload
// function: it sits on fpath, the `#compdef skilldozer` magic comment binds it to the command,
// and its trailing `_skilldozer "$@"` self-call is the standard idiom that fires the function
// when zsh first autoloads it (by which point compsys — _arguments/_files/compadd — is loaded).
//
// None of that holds under `eval "$(skilldozer --completions)"` (PRD §14.6):
//   - `#compdef skilldozer` is an INERT COMMENT when the script is eval'd (it only binds
//     autoload files that compinit scans off fpath), so the function never gets registered.
//   - the trailing `_skilldozer "$@"` self-call runs the function IMMEDIATELY in the user's
//     .zshrc, before compsys is guaranteed loaded, hitting _arguments at line 31 →
//     `_skilldozer:31: command not found: _arguments`.
//
// So we strip the self-call and append an explicit compdef registration plus a compinit
// bootstrap that is a no-op once compsys is already loaded (oh-my-zsh / prezto / a manual
// `autoload -U compinit && compinit` all define compdef). completionScript keeps returning
// the verbatim bytes (the §14.6 byte-identity lock); this derivation lives in runCompletion.
func zshEvalScript(raw string) string {
	// The autoload file ends with:   ...esac\n}\n\n_skilldozer "$@"\n
	// LastIndex (not a suffix trim) is robust to trailing-comment drift, and the function
	// body never references its own name, so the one match is the self-call. TrimRight
	// collapses the blank line(s) so the body ends cleanly at the closing brace.
	const call = "_skilldozer \"$@\"\n"
	body := raw
	if i := strings.LastIndex(raw, call); i >= 0 {
		body = strings.TrimRight(raw[:i], "\n") + "\n"
	}
	return body + zshEvalRegistration
}

// zshEvalRegistration is appended to the stripped autoload body to make it eval-safe. Its own
// const so a test can lock the exact registration contract (compdef binding + the no-op-when-
// loaded compinit bootstrap). No backticks inside: it is a Go raw string literal.
const zshEvalRegistration = `
# Register the completion for eval. The #compdef header above only binds this as an
# autoload file on fpath; under eval it is inert, so bind the function explicitly.
# compsys (_arguments/_files/compadd) is bootstrapped only if not already loaded —
# oh-my-zsh / prezto / a manual compinit all define compdef, making the compinit
# below a no-op. The autoload file's trailing self-call is intentionally omitted:
# it would fire the function at eval time, before _arguments is guaranteed to exist.
autoload -Uz compinit
(( $+functions[compdef] )) || compinit
(( $+functions[compdef] )) && compdef _skilldozer skilldozer
`

// runInit is the `skilldozer --init` orchestrator (PRD §8.2). run()'s dispatch calls it
// when c.init is true (init is exclusive, so no other mode is active). It assembles the
// three already-landed helpers — resolveStore (P1.M2.T2.S1: choose+absolutize the store),
// configpkg.Path (the config-file location), setupStore (P1.M2.T2.S2: mkdir+seed+writeconfig)
// — and then reports: the configured store path to stdout (PRD §6.1), the `--path` "found
// via" annotation to stderr, and the `check` report to stderr (PRD §6.1 stdout contract;
// §6.1 is authoritative over §8.2 step 5's 'print check' wording). Exit 0
// once create+config succeed; the check report is best-effort (NOT a gate — check findings
// do not change init's exit code, only setup failure does).
//
// The bare `skilldozer <tag>` path NEVER reaches here (c.init is false for tags), so tag
// resolution never prompts (PRD §6.4/§8.2 prompt-safety): stdin access is confined to
// resolveStore, which only init calls.
func runInit(c config, stdout, stderr io.Writer) int {
	// (1) Choose the store (haveStore != "" never blocks; resolveStore absolutizes).
	store, err := resolveStore(c.initStore)
	if err != nil {
		fmt.Fprintln(stderr, err) // one-line (resolveStore wraps with "skilldozer --init: …")
		return 1
	}
	// (2) Resolve the config-file location (pure env fn; $SKILLDOZER_CONFIG or XDG default).
	cfgPath, err := configpkg.Path()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// (3) Create the store, seed it if empty, write the config (PRD §8.2 steps 2-4).
	seeded, err := setupStore(store, cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, err) // setupStore wraps with "skilldozer --init: …"
		return 1
	}
	// (4) Report what happened. Uses `seeded` (S2's success-path signal). STDERR so §6.1's
	//     stdout headline (the store path) stays clean.
	if seeded {
		fmt.Fprintf(stderr, "Seeded example skill at %s\n", filepath.Join(store, "example", "SKILL.md"))
	} else {
		fmt.Fprintf(stderr, "Adopted existing store at %s\n", store)
	}
	// (5) Show the EFFECTIVE store + which §8.3 rule won (mirrors `skilldozer --path`, PRD
	//     §8.2 step 5). Find() runs AFTER setupStore so the just-written config is visible.
	//     In the common case dir == store and src == "config file"; if SKILLDOZER_SKILLS_DIR
	//     is set, env beats config and dir/src reflect that honestly.
	dir, src, ferr := skillsdir.Find()
	if ferr != nil {
		// Should not happen (setupStore just wrote a valid config + created the store).
		// Fall back to the configured store so §6.1 (stdout = store path) still holds.
		fmt.Fprintln(stderr, ferr)
		dir = store
	}
	// §6.1: stdout = the configured store path (== dir, the effective resolved store).
	fmt.Fprintln(stdout, dir)
	if ferr == nil {
		// Mirror `skilldozer --path`: which rule won.
		fmt.Fprintf(stderr, "(found via %s)\n", src)
	}
	// (6) `skilldozer --check` report on the effective store (PRD §8.2 step 5). init renders
	//     this report to STDERR (PRD §6.1: stdout = the store path only); the standalone
	//     `--check` flag keeps its report on stdout (its report IS its stdout product),
	//     so the two blocks intentionally DIVERGE — do NOT extract a shared helper. The
	//     report is best-effort: a discover.Index failure is non-fatal (setup succeeded).
	skills, ierr := discover.Index(dir)
	if ierr != nil {
		fmt.Fprintln(stderr, ierr)
		return 0 // setup OK; the report is best-effort
	}
	rep := check.Check(skills)
	for _, sr := range rep.BySkill {
		name := sr.Skill.Name
		if name == "" {
			name = "(none)"
		}
		if len(sr.Findings) == 0 {
			fmt.Fprintf(stderr, "%-5s %s (%s)\n", "OK", sr.Skill.RelTag, name)
			continue
		}
		for _, f := range sr.Findings {
			fmt.Fprintf(stderr, "%-5s %s (%s): %s\n", f.Level, sr.Skill.RelTag, name, f.Message)
		}
	}
	fmt.Fprintf(stderr, "%d skills, %d errors, %d warnings\n", len(skills), rep.Errors, rep.Warnings)
	return 0 // setup succeeded; check findings do not change init's exit code
}

// loginShellBase returns the lowercased basename of $SHELL — "zsh" for /bin/zsh,
// "fish" for /usr/bin/fish — or "" if $SHELL is unset/empty. It is the third-tier
// input to detectShell (PRD §14.6 "basename($SHELL)"). The empty guard is
// load-bearing: filepath.Base("") returns ".", which would otherwise pollute
// shell detection (external_deps.md §Shell detection verified this).
// os.Executable() is NOT used — that is the skilldozer binary path, not the shell.
// strings.ToLower normalizes e.g. /bin/ZSH → "zsh" so completionScript's
// case-sensitive switch still matches (explicit --shell / $SKILLDOZER_SHELL
// values pass through detectShell verbatim, not lowercased).
func loginShellBase() string {
	s := os.Getenv("SHELL")
	if s == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(s))
}

// detectShell resolves the target shell for `skilldozer --completions` (PRD §14.6
// "Shell detection", first wins): explicit --shell → $SKILLDOZER_SHELL →
// basename($SHELL). It is a PURE function of its three string args — the caller
// (runCompletion) supplies the env reads (c.completionShell,
// os.Getenv("SKILLDOZER_SHELL"), loginShellBase()) — so detection is
// unit-testable without env mutation. It returns the first non-empty value +
// true, or ("", false) if all three are empty. Values pass through VERBATIM:
// only loginShellBase lowercases (it computes the basename), so an explicit
// `--shell BASH` yields "BASH" → unsupported (completionScript is
// case-sensitive) → exit 2.
func detectShell(explicit, envShell, loginShell string) (string, bool) {
	if explicit != "" {
		return explicit, true
	}
	if envShell != "" {
		return envShell, true
	}
	if loginShell != "" {
		return loginShell, true
	}
	return "", false
}

// runCompletion is the `skilldozer --completions` handler (PRD §14.6 / §6.4).
// run()'s dispatch calls it when c.completion is true; completion is exclusive,
// so no other mode is active. It resolves the shell via detectShell, then emits
// the matching embedded script to stdout for `eval "$(skilldozer --completions)"`
// (PRD §14.6). Exit codes (PRD §6.4):
//   - 0 on success (script on stdout);
//   - 1 if the shell is undetectable (no --shell, no $SKILLDOZER_SHELL, no
//     usable $SHELL);
//   - 2 if the resolved shell value is not bash/zsh/fish (e.g. tcsh, or
//     $SHELL=/bin/sh → "sh").
//
// On the 1/2 paths NOTHING is written to stdout (the §6.4 $(...) contract:
// `eval` of an empty capture fails cleanly rather than running a partial/garbage
// script). Emitted with Fprint (NOT Fprintln). bash/fish are the verbatim
// embedded bytes (§14.6 byte-identity); zsh is a DERIVED eval-safe wrapper
// (zshEvalScript) — the on-disk autoload file ends with a `_skilldozer "$@"`
// self-call that would fire the function at eval time, before compsys is loaded
// (→ `command not found: _arguments`), and its `#compdef` header is inert eval'd.
func runCompletion(c config, stdout, stderr io.Writer) int {
	shell, ok := detectShell(c.completionShell, os.Getenv("SKILLDOZER_SHELL"), loginShellBase())
	if !ok {
		// Undetectable: all three sources empty (PRD §6.4). Nothing on stdout.
		fmt.Fprintln(stderr, "could not detect shell; pass --shell {bash|zsh|fish}")
		return 1
	}
	script, ok := completionScript(shell)
	if !ok {
		// Detected but unsupported (e.g. tcsh; or $SHELL=/bin/sh → "sh"). Nothing on stdout.
		fmt.Fprintf(stderr, "skilldozer: unsupported shell '%s' (want bash|zsh|fish)\n", shell)
		return 2
	}
	// Success. bash/fish: emit the embedded bytes verbatim. zsh: emit the DERIVED
	// eval-safe wrapper — the autoload file's trailing `_skilldozer "$@"` self-call
	// fires the function immediately under eval (→ `command not found: _arguments`
	// before compsys is loaded) and its `#compdef` header is inert eval'd, so it needs
	// an explicit compdef registration. See zshEvalScript (PRD §14.6).
	if shell == "zsh" {
		script = zshEvalScript(script)
	}
	fmt.Fprint(stdout, script)
	return 0
}
