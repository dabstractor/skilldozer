package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTerminal overrides the package-level isTerminal func for one test and
// restores it on cleanup. Use it to exercise the color-enabled path through
// run() without a real terminal. NOT t.Parallel-safe (mutates package state).
func withTerminal(t *testing.T, isTTY bool) {
	t.Helper()
	prev := isTerminal
	isTerminal = func(io.Writer) bool { return isTTY }
	t.Cleanup(func() { isTerminal = prev })
}

// unsetSkillsEnv removes SKPP_SKILLS_DIR for the test and restores it on cleanup.
// (Mirrors internal/skillsdir/skillsdir_test.go's unsetEnvVar helper.) Forbids
// t.Parallel via t.Setenv.
func unsetSkillsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SKPP_SKILLS_DIR", "")
}

// writeSkillTree builds a temp skills/ tree from a map[relTag]SKILL.md-content
// and returns its root. relTag uses '/' separators (cross-platform via FromSlash).
// A "" key writes SKILL.md directly in the root. Used by the --list tests to give
// skillsdir.Find() (via SKPP_SKILLS_DIR) a real store to discover.
func writeSkillTree(t *testing.T, layout map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relTag, content := range layout {
		dir := filepath.Join(root, filepath.FromSlash(relTag))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", dir, err)
		}
	}
	return root
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
// Dashed unknown flags are tolerated (no-op; exit-2 is M5). Non-dashed positional
// tokens are now captured as <tag>s (so "sometag"/"check" land in c.tags rather
// than being discarded). Only version/path stay false here.
func TestParseArgsUnknownTolerated(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "sometag", "check"})
	if c.version || c.path {
		t.Errorf("parseArgs(unknown): version=%v path=%v; want both false", c.version, c.path)
	}
	// Non-dashed positionals are captured as tags; the dashed --frobnicate is excluded.
	if len(c.tags) != 2 || c.tags[0] != "sometag" || c.tags[1] != "check" {
		t.Errorf("parseArgs tags=%v; want [sometag check] (positionals captured)", c.tags)
	}
}

func TestParseArgsListLong(t *testing.T) {
	c := parseArgs([]string{"--list"})
	if !c.list || c.version || c.path {
		t.Errorf("parseArgs(--list): list=%v; want true (others false)", c.list)
	}
}

func TestParseArgsListShort(t *testing.T) {
	c := parseArgs([]string{"-l"})
	if !c.list {
		t.Errorf("parseArgs(-l): list=false; want true")
	}
}

func TestParseArgsNoColor(t *testing.T) {
	c := parseArgs([]string{"--no-color"})
	if !c.noColor {
		t.Errorf("parseArgs(--no-color): noColor=false; want true")
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

// --- run: --list / -l (P1.M2.T6) ---

// --list success: a store with one skill -> catalog table on stdout, exit 0, no
// ANSI (stdout is a *bytes.Buffer -> not a TTY -> plain output by default).
func TestRunListSuccess(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: A demo skill.\n---\n# body\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir) // rule 1 wins; Find() returns dir, Index finds the skill
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--list): code=%d; want 0", code)
	}
	got := out.String()
	for _, want := range []string{"TAG", "NAME", "DESCRIPTION", "example", "A demo skill."} {
		if !strings.Contains(got, want) {
			t.Errorf("run(--list) stdout missing %q:\n%s", want, got)
		}
	}
	// Default (non-TTY buffer) -> no ANSI escapes.
	if strings.Contains(got, "\x1b[") {
		t.Errorf("run(--list) on a non-TTY must not emit ANSI:\n%s", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--list) stderr=%q; want empty", errOut.String())
	}
}

func TestRunListShortFlag(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-l"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-l): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "example") {
		t.Errorf("run(-l) stdout missing the example tag:\n%s", out.String())
	}
}

// --list with NO skills (empty store) -> PRD §6.1 exit 1, stdout empty, a message
// to stderr. SKPP_SKILLS_DIR pointing at an existing-but-empty dir: rule 1 wins
// (it needs only an existing dir), Index returns [], len==0 -> exit 1.
func TestRunListNoSkillsExit1(t *testing.T) {
	t.Setenv("SKPP_SKILLS_DIR", t.TempDir()) // exists, no SKILL.md -> empty catalog
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--list) empty store: code=%d; want 1 (PRD §6.1 '1 if no skills found')", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--list) empty store stdout=%q; want empty (only the exit-1 + stderr msg)", out.String())
	}
	if !strings.Contains(errOut.String(), "no skills found") {
		t.Errorf("run(--list) empty store stderr=%q; want a 'no skills found' message", errOut.String())
	}
}

// --list when the skills dir is unresolvable -> Find() returns ErrNotFound ->
// exit 1, stdout empty, the one-line fix to stderr (same contract as --path).
func TestRunListSkillsDirUnresolvableExit1(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // force all three §8 rules to miss
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--list) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--list) unresolvable stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("run(--list) unresolvable stderr=%q; want the one-line fix", errOut.String())
	}
}

// --list with --no-color suppresses ANSI even when stdout looks like a TTY.
// Forces isTerminal=true (so color WOULD be on by default) and asserts --no-color
// still yields plain output.
func TestRunListNoColorFlagSuppressesANSI(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true) // pretend stdout is a TTY
	var out, errOut bytes.Buffer
	code := run([]string{"--list", "--no-color"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--list --no-color): code=%d; want 0", code)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Errorf("--no-color must suppress ANSI even on a TTY:\n%s", out.String())
	}
}

// --list color path: when stdout is a TTY (forced) and --no-color is absent, the
// table carries ANSI escapes. Proves the TTY gate is wired into run().
func TestRunListColorWhenTTY(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true)
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--list) tty: code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[1m") || !strings.Contains(got, "\x1b[36m") || !strings.Contains(got, "\x1b[0m") {
		t.Errorf("TTY output should contain ANSI bold/cyan/reset:\n%s", got)
	}
}

// --- parseArgs: positional <tag> capture (P1.M3.T8.S1) ---

// Positional <tag> args (non-dashed tokens) are captured in INPUT order (PRD §6.1).
func TestParseArgsCapturesTagsInOrder(t *testing.T) {
	c := parseArgs([]string{"foo", "writing/reddit"})
	if len(c.tags) != 2 || c.tags[0] != "foo" || c.tags[1] != "writing/reddit" {
		t.Errorf("tags=%v; want [foo writing/reddit] in input order", c.tags)
	}
}

// Dashed unknowns are NOT tags (they are tolerated flags); only the positional is captured.
func TestParseArgsDashedUnknownNotATag(t *testing.T) {
	c := parseArgs([]string{"--frobnicate", "real-tag", "-x"})
	if len(c.tags) != 1 || c.tags[0] != "real-tag" {
		t.Errorf("tags=%v; want [real-tag] (dashed tokens excluded)", c.tags)
	}
}

// Tags and recognized flags may interleave (PRD §6: flags appear in any order).
func TestParseArgsTagsAndFlagsInterleave(t *testing.T) {
	c := parseArgs([]string{"--no-color", "a", "-l", "b"})
	if !c.list || !c.noColor || len(c.tags) != 2 || c.tags[0] != "a" || c.tags[1] != "b" {
		t.Errorf("config=%+v; want list+noColor true and tags=[a b]", c)
	}
}

// --- run: <tag> resolution (P1.M3.T8.S1) ---

// sampleStore builds a store with a top-level `example` and a nested
// `writing/reddit` skill, returning the skills dir (set via SKPP_SKILLS_DIR rule 1).
func sampleStore(t *testing.T) string {
	t.Helper()
	return writeSkillTree(t, map[string]string{
		"example":        "---\nname: example\ndescription: A demo skill.\n---\n# body\n",
		"writing/reddit": "---\nname: reddit-poster\ndescription: Posts to reddit.\n---\n# body\n",
	})
}

// Single tag resolves to its absolute skill DIRECTORY path on stdout, exit 0, no
// stderr. The default output is the dir, not SKILL.md (--file is P1.M3.T8.S2).
func TestRunTagSingleResolvesToDir(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example): code=%d; want 0", code)
	}
	want := filepath.Join(dir, "example") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(example) stdout=%q; want %q (absolute dir + newline)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(example) stderr=%q; want empty", errOut.String())
	}
}

// Multiple tags -> one path per line, in INPUT order (not sorted), exit 0. `reddit`
// resolves by basename to writing/reddit; `example` by canonical tag.
func TestRunTagMultipleInInputOrder(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"reddit", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(reddit example): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	if lines[0] != filepath.Join(dir, "writing", "reddit") {
		t.Errorf("lines[0]=%q; want the reddit dir (input order preserved)", lines[0])
	}
	if lines[1] != filepath.Join(dir, "example") {
		t.Errorf("lines[1]=%q; want the example dir (input order preserved)", lines[1])
	}
}

// ATOMICITY (§6.4): one unknown tag among resolvable ones -> NOTHING on stdout, one
// stderr line per problem tag, exit 1. The resolvable tag must NOT leak to stdout.
func TestRunTagAtomicityUnknownPrintsNothing(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example", "nope"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(example nope): code=%d; want 1 (atomic failure)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (§6.4: nothing printed on failure)", out.String())
	}
	if !strings.Contains(errOut.String(), "nope") {
		t.Errorf("stderr=%q; want an error line naming 'nope'", errOut.String())
	}
}

// All tags fail -> one stderr line per problem tag, nothing on stdout, exit 1.
func TestRunTagAllFailMultipleErrorLines(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"nope1", "nope2"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(nope1 nope2): code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY", out.String())
	}
	errLines := strings.Split(strings.TrimRight(errOut.String(), "\n"), "\n")
	if len(errLines) != 2 {
		t.Fatalf("want 2 stderr lines (one per problem tag); got %d: %q", len(errLines), errOut.String())
	}
}

// A tag repeated in argv resolves each time; output repeats. Not an error.
func TestRunTagDuplicateArgResolvesTwice(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example example): code=%d; want 0", code)
	}
	want := strings.Repeat(filepath.Join(dir, "example")+"\n", 2)
	if got := out.String(); got != want {
		t.Errorf("stdout=%q; want two identical path lines:\n%s", got, want)
	}
}

// Ambiguous tag (basename collision) -> stderr lists the candidate full tags,
// NOTHING on stdout, exit 1 (PRD §6.4).
func TestRunTagAmbiguousListsCandidates(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"writing/reddit": "---\nname: a\ndescription: d\n---\nx\n",
		"coding/reddit":  "---\nname: b\ndescription: d\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"reddit"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(reddit) ambiguous: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (ambiguous => nothing on stdout)", out.String())
	}
	msg := errOut.String()
	for _, want := range []string{"reddit", "coding/reddit", "writing/reddit"} {
		if !strings.Contains(msg, want) {
			t.Errorf("stderr=%q; missing candidate %q", msg, want)
		}
	}
}

// Skills dir unresolvable + tags -> exit 1, nothing on stdout, the one-line fix on
// stderr (same contract as --path/--list).
func TestRunTagSkillsDirUnresolvable(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // all three §8 rules miss
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(example) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("stderr=%q; want the one-line fix", errOut.String())
	}
}

// The resolved path is ABSOLUTE (PRD §6.1 default; --relative is P1.M3.T8.S2).
func TestRunTagPathIsAbsolute(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example): code=%d; want 0", code)
	}
	if p := strings.TrimRight(out.String(), "\n"); !filepath.IsAbs(p) {
		t.Errorf("resolved path %q is not absolute (discover.Skill.Dir should be absolute)", p)
	}
}

// --version precedes tag-resolution mode even when a tag is present (PRD §6.3).
func TestRunVersionPrecedenceOverTag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"example", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(example --version): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (precedence over tag mode)", got)
	}
}

// --- parseArgs: modifiers --file/-f, --relative, --all/-a (P1.M3.T8.S2) ---

// --file/-f sets c.file (long and short forms, PRD §6.2).
func TestParseArgsFileLong(t *testing.T) {
	c := parseArgs([]string{"--file"})
	if !c.file {
		t.Errorf("parseArgs(--file): file=false; want true")
	}
}

func TestParseArgsFileShort(t *testing.T) {
	c := parseArgs([]string{"-f"})
	if !c.file {
		t.Errorf("parseArgs(-f): file=false; want true")
	}
}

// --relative has NO short form (PRD §6.2 lists only the long form).
func TestParseArgsRelativeLong(t *testing.T) {
	c := parseArgs([]string{"--relative"})
	if !c.relative {
		t.Errorf("parseArgs(--relative): relative=false; want true")
	}
}

// --all/-a sets c.all (long and short forms, PRD §6.1).
func TestParseArgsAllLong(t *testing.T) {
	c := parseArgs([]string{"--all"})
	if !c.all {
		t.Errorf("parseArgs(--all): all=false; want true")
	}
}

func TestParseArgsAllShort(t *testing.T) {
	c := parseArgs([]string{"-a"})
	if !c.all {
		t.Errorf("parseArgs(-a): all=false; want true")
	}
}

// Modifiers may interleave with tags and other flags (PRD §6 any order).
func TestParseArgsModifiersInterleave(t *testing.T) {
	c := parseArgs([]string{"-f", "example", "--relative"})
	if !c.file || !c.relative || len(c.tags) != 1 || c.tags[0] != "example" {
		t.Errorf("config=%+v; want file+relative true and tags=[example]", c)
	}
}

// --- run: <tag> + --file/--relative modifiers (P1.M3.T8.S2) ---

// --file prints the ABSOLUTE SKILL.md path instead of the dir (PRD §6.2). The §13
// gate `test -f "$(./skpp -f example)"` depends on this printing a real file.
func TestRunTagFilePrintsSourceFile(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-f", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-f example): code=%d; want 0", code)
	}
	want := filepath.Join(dir, "example", "SKILL.md") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(-f example) stdout=%q; want %q (absolute SKILL.md path)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(-f example) stderr=%q; want empty", errOut.String())
	}
}

// --relative prints the dir path RELATIVE to the skills dir (PRD §6.2). The output
// uses the OS path separator (filepath.Rel), so compare via FromSlash.
func TestRunTagRelativePrintsRelativeDir(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--relative", "writing/reddit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--relative writing/reddit): code=%d; want 0", code)
	}
	want := filepath.FromSlash("writing/reddit") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(--relative writing/reddit) stdout=%q; want %q (relative dir)", got, want)
	}
}

// --file --relative COMBINE: a SKILL.md path RELATIVE to the skills dir (PRD §6.2).
func TestRunTagFileRelativeCombine(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-f", "--relative", "writing/reddit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-f --relative writing/reddit): code=%d; want 0", code)
	}
	want := filepath.FromSlash("writing/reddit/SKILL.md") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(-f --relative writing/reddit) stdout=%q; want %q (relative SKILL.md)", got, want)
	}
}

// Modifiers must NOT break §6.4 atomicity: one bad tag -> NOTHING on stdout, exit 1.
func TestRunTagFileAtomicity(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-f", "example", "nope"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(-f example nope): code=%d; want 1 (atomic failure)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (modifiers must not break §6.4)", out.String())
	}
}

// --- run: --all/-a (P1.M3.T8.S2) ---

// --all prints every skill's absolute DIRECTORY path, one per line, SORTED by
// canonical tag (discover.Index already sorts []Skill by RelTag). exit 0.
func TestRunAllPrintsAllDirsSorted(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	// Sorted by RelTag: "example" < "writing/reddit".
	if lines[0] != filepath.Join(dir, "example") {
		t.Errorf("lines[0]=%q; want example dir (sorted)", lines[0])
	}
	if lines[1] != filepath.Join(dir, "writing", "reddit") {
		t.Errorf("lines[1]=%q; want writing/reddit dir (sorted)", lines[1])
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--all) stderr=%q; want empty", errOut.String())
	}
}

// -a short form behaves identically to --all.
func TestRunAllShortFlag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-a"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-a): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), filepath.Join(dir, "example")) {
		t.Errorf("run(-a) stdout missing example dir:\n%s", out.String())
	}
}

// --all --file: every skill's ABSOLUTE SKILL.md path, sorted by tag.
func TestRunAllFilePrintsAllSourceFiles(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--file"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all --file): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	if lines[0] != filepath.Join(dir, "example", "SKILL.md") {
		t.Errorf("lines[0]=%q; want example SKILL.md (sorted)", lines[0])
	}
	if lines[1] != filepath.Join(dir, "writing", "reddit", "SKILL.md") {
		t.Errorf("lines[1]=%q; want writing/reddit SKILL.md (sorted)", lines[1])
	}
}

// --all --relative: every skill's directory path RELATIVE to the skills dir, sorted.
func TestRunAllRelativePrintsAllRelative(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--relative"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all --relative): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	if lines[0] != "example" {
		t.Errorf("lines[0]=%q; want 'example' (relative)", lines[0])
	}
	if lines[1] != filepath.FromSlash("writing/reddit") {
		t.Errorf("lines[1]=%q; want 'writing/reddit' (relative, OS-sep)", lines[1])
	}
}

// --all with an EMPTY store -> prints nothing, exit 0 (PRD §6.1: --all is always
// exit 0, UNLIKE --list which exits 1 "if no skills found" — --all is a scripting
// command where empty output + exit 0 is the useful shape).
func TestRunAllEmptyStoreExit0(t *testing.T) {
	t.Setenv("SKPP_SKILLS_DIR", t.TempDir()) // exists, no SKILL.md
	var out, errOut bytes.Buffer
	code := run([]string{"--all"}, &out, &errOut)
	if code != 0 {
		t.Errorf("run(--all) empty: code=%d; want 0 (PRD §6.1 --all is always 0)", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--all) empty stdout=%q; want empty", out.String())
	}
}

// --all when skills dir is unresolvable -> exit 1, empty stdout, the one-line fix.
func TestRunAllSkillsDirUnresolvable(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // all three §8 rules miss
	var out, errOut bytes.Buffer
	code := run([]string{"--all"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--all) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("stderr=%q; want the one-line fix", errOut.String())
	}
}

// --version precedes --all even when both are given (PRD §6.3).
func TestRunVersionPrecedenceOverAll(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all --version): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (precedence over --all)", got)
	}
}

// --- parseArgs: --search/-s value flag (P1.M4.T9.S1) ---

// --search <q> sets searchMode=true and captures the query; the value is NOT a tag.
func TestParseArgsSearchLong(t *testing.T) {
	c := parseArgs([]string{"--search", "reddit"})
	if !c.searchMode || c.searchQ != "reddit" {
		t.Errorf("parseArgs(--search reddit): mode=%v q=%q; want true,reddit", c.searchMode, c.searchQ)
	}
	if len(c.tags) != 0 {
		t.Errorf("--search value leaked into tags: %v", c.tags)
	}
}

// -s <q> short form behaves identically.
func TestParseArgsSearchShort(t *testing.T) {
	c := parseArgs([]string{"-s", "reddit"})
	if !c.searchMode || c.searchQ != "reddit" {
		t.Errorf("parseArgs(-s reddit): mode=%v q=%q; want true,reddit", c.searchMode, c.searchQ)
	}
}

// --search with NO following value (last token) -> searchMode stays false; falls
// to the default exit-1 path. Proper exit-2 "needs an argument" is P1.M5.T11.
func TestParseArgsSearchNoValueStaysInactive(t *testing.T) {
	c := parseArgs([]string{"--search"})
	if c.searchMode {
		t.Errorf("parseArgs(--search) with no value: searchMode=true; want false (no value consumed)")
	}
}

// --search consumes exactly ONE value; a following positional is captured as a tag.
// (Mixing search mode + a tag is an M5.T11 exclusivity error; for now searchMode
// wins in run() dispatch and tags are ignored.)
func TestParseArgsSearchConsumesOneValue(t *testing.T) {
	c := parseArgs([]string{"--search", "q", "sometag"})
	if !c.searchMode || c.searchQ != "q" {
		t.Errorf("search not captured: mode=%v q=%q", c.searchMode, c.searchQ)
	}
	if len(c.tags) != 1 || c.tags[0] != "sometag" {
		t.Errorf("tags=%v; want [sometag] (the token after the search value)", c.tags)
	}
}

// --- run: --search / -s (P1.M4.T9.S1) ---

// --search success: a query matching a skill's tag prints the filtered table,
// exit 0, no ANSI (non-TTY buffer). sampleStore has example + writing/reddit.
func TestRunSearchMatchByTag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example): code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "example") {
		t.Errorf("stdout missing 'example' row:\n%s", got)
	}
	if strings.Contains(got, "reddit") { // unmatched skill must not leak
		t.Errorf("unmatched skill 'reddit' leaked into search results:\n%s", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("non-TTY search must not emit ANSI:\n%s", got)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr=%q; want empty", errOut.String())
	}
}

func TestRunSearchShortFlag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-s", "reddit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-s reddit): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "reddit") {
		t.Errorf("stdout missing matched row:\n%s", out.String())
	}
}

// --search is case-insensitive (PRD §6.1).
func TestRunSearchCaseInsensitive(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "REDDIT"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search REDDIT): code=%d; want 0 (case-insensitive)", code)
	}
	if !strings.Contains(out.String(), "reddit") {
		t.Errorf("case-insensitive query should match:\n%s", out.String())
	}
}

// --search matches by description (example has "A demo skill.").
func TestRunSearchMatchByDescription(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "demo"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search demo): code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "example") {
		t.Errorf("description match should find example:\n%s", got)
	}
	if strings.Contains(got, "reddit") {
		t.Errorf("non-matching reddit should be filtered out:\n%s", got)
	}
}

// --search matches by frontmatter name (sampleStore reddit has name reddit-poster).
func TestRunSearchMatchByName(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "poster"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search poster): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), "reddit") {
		t.Errorf("name match should find reddit skill:\n%s", out.String())
	}
}

// --search matches by metadata.keywords (PRD §6.1).
func TestRunSearchMatchByKeyword(t *testing.T) {
	dir := writeSkillTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\nmetadata:\n  keywords: [writing, social]\n---\nx\n",
	})
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "soc"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search soc): code=%d; want 0 (keyword match)", code)
	}
	if !strings.Contains(out.String(), "example") {
		t.Errorf("keyword match should find example:\n%s", out.String())
	}
}

// --search with NO matches -> exit 1, EMPTY stdout, message to stderr (PRD §6.1).
func TestRunSearchNoMatchesExit1(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "zzznotfound"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--search zzznotfound): code=%d; want 1 (no matches)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (PRD §6.1: no matches => nothing on stdout)", out.String())
	}
	if !strings.Contains(errOut.String(), "no skills matched") {
		t.Errorf("stderr=%q; want a 'no skills matched' message", errOut.String())
	}
}

// --search "" (empty query) matches ALL skills (substring semantics): exit 0, full
// table — like --list. (PRD carves out no special case for an empty query.)
func TestRunSearchEmptyQueryMatchesAll(t *testing.T) {
	dir := sampleStore(t) // 2 skills
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", ""}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search ''): code=%d; want 0 (empty matches all)", code)
	}
	got := out.String()
	if !strings.Contains(got, "example") || !strings.Contains(got, "reddit") {
		t.Errorf("empty query should list all skills:\n%s", got)
	}
}

// --search respects --no-color: suppresses ANSI even on a TTY.
func TestRunSearchNoColorSuppressesANSI(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example", "--no-color"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example --no-color): code=%d; want 0", code)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Errorf("--no-color must suppress ANSI in search:\n%s", out.String())
	}
}

// --search emits ANSI when stdout is a TTY and --no-color is absent.
func TestRunSearchColorWhenTTY(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	withTerminal(t, true)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example) tty: code=%d; want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[1m") || !strings.Contains(got, "\x1b[36m") {
		t.Errorf("TTY search output should contain ANSI bold/cyan:\n%s", got)
	}
}

// --search when skills dir is unresolvable -> exit 1, empty stdout, one-line fix.
func TestRunSearchSkillsDirUnresolvable(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // all three §8 rules miss
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "x"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--search x) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("stderr=%q; want the one-line fix", errOut.String())
	}
}

// --version precedes --search (PRD §6.3).
func TestRunVersionPrecedenceOverSearch(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--search", "example", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--search example --version): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (precedence over search)", got)
	}
}
