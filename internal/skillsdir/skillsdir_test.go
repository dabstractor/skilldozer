package skillsdir

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unsetEnvVar removes envVar for the duration of the test and restores the
// prior state on cleanup. Needed because t.Setenv can only set, never unset.
// Do NOT call t.Parallel() in any test that uses this or t.Setenv.
func unsetEnvVar(tb testing.TB) {
	tb.Helper()
	prev, had := os.LookupEnv(envVar)
	if err := os.Unsetenv(envVar); err != nil {
		tb.Fatalf("unsetenv %s: %v", envVar, err)
	}
	tb.Cleanup(func() {
		if had {
			_ = os.Setenv(envVar, prev)
		} else {
			_ = os.Unsetenv(envVar)
		}
	})
}

func TestSourceString(t *testing.T) {
	cases := []struct {
		src  Source
		want string
	}{
		{SourceEnv, "SKILLDOZER_SKILLS_DIR"},
		{SourceConfig, "config file"},
		{SourceSibling, "sibling of binary"},
		{SourceWalkUp, "ancestor of cwd"},
		{Source(-1), "unknown"}, // out-of-range -> default
		{Source(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.src.String(); got != c.want {
			t.Errorf("Source(%d).String() = %q, want %q", c.src, got, c.want)
		}
	}
}

// Rule 1: env unset -> not found (fall through, no error).
func TestFindEnvUnset(t *testing.T) {
	unsetEnvVar(t)
	dir, src, found := findEnv()
	if found {
		t.Errorf("findEnv() env unset: got found=true dir=%q src=%v; want found=false", dir, src)
	}
}

// Rule 1: env set to "" -> not found (treated as no usable dir).
func TestFindEnvEmpty(t *testing.T) {
	t.Setenv(envVar, "")
	dir, src, found := findEnv()
	if found {
		t.Errorf("findEnv() env empty: got found=true dir=%q src=%v; want found=false", dir, src)
	}
}

// Rule 1: env set to an existing directory (absolute temp dir) -> found, abs path, SourceEnv.
func TestFindEnvExistingDir(t *testing.T) {
	dir := t.TempDir() // already absolute + clean
	t.Setenv(envVar, dir)
	got, src, found := findEnv()
	if !found {
		t.Fatalf("findEnv() existing dir: found=false; want true")
	}
	if src != SourceEnv {
		t.Errorf("findEnv() existing dir: src=%v; want SourceEnv", src)
	}
	if want := filepath.Clean(dir); got != want {
		t.Errorf("findEnv() existing dir: dir=%q; want %q", got, want)
	}
}

// Rule 1: env set to a path that does not exist -> not found (fall through).
func TestFindEnvNonexistent(t *testing.T) {
	t.Setenv(envVar, filepath.Join(t.TempDir(), "does-not-exist"))
	dir, src, found := findEnv()
	if found {
		t.Errorf("findEnv() nonexistent: got found=true dir=%q src=%v; want found=false", dir, src)
	}
}

// Rule 1: env set to a regular file (not a directory) -> not found (fall through).
func TestFindEnvRegularFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envVar, f)
	dir, src, found := findEnv()
	if found {
		t.Errorf("findEnv() regular file: got found=true dir=%q src=%v; want found=false", dir, src)
	}
}

// Rule 1: env set to a RELATIVE existing dir (".") -> found, absolutized via filepath.Abs.
// Proves the filepath.Abs (relative->absolute) path without chdir or cwd pollution.
func TestFindEnvRelativePathAbsolutized(t *testing.T) {
	t.Setenv(envVar, ".") // "." always exists and is a dir (the test's cwd)
	got, src, found := findEnv()
	if !found {
		t.Fatalf("findEnv() relative '.': found=false; want true")
	}
	if src != SourceEnv {
		t.Errorf("findEnv() relative '.': src=%v; want SourceEnv", src)
	}
	want, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("findEnv() relative '.': dir=%q; want absolutized %q", got, want)
	}
}

// Rule 1 CONTRACT: the env path must NOT be passed through EvalSymlinks.
// If SKILLDOZER_SKILLS_DIR points at a symlink-to-a-dir, findEnv must return the
// symlink path (made absolute/clean), NOT the resolved target. The user points
// exactly where they want.
func TestFindEnvDoesNotResolveSymlinks(t *testing.T) {
	realDir := t.TempDir()
	parent := t.TempDir()
	link := filepath.Join(parent, "link-to-skills")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}
	t.Setenv(envVar, link)
	got, src, found := findEnv()
	if !found {
		t.Fatalf("findEnv() symlink-to-dir: found=false; want true (os.Stat follows the symlink)")
	}
	if src != SourceEnv {
		t.Errorf("findEnv() symlink-to-dir: src=%v; want SourceEnv", src)
	}
	if got == realDir {
		t.Errorf("findEnv() symlink-to-dir: dir=%q == resolved target; must NOT EvalSymlinks the env path", got)
	}
	want, err := filepath.Abs(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("findEnv() symlink-to-dir: dir=%q; want symlink path (absolutized) %q", got, want)
	}
}

// makeFakeBinary creates a regular file at dir/name to stand in for a compiled
// binary. EvalSymlinks + os.Stat(Join(dir,"skills")) do not require a real ELF,
// so a 1-byte file is sufficient (research/verified_facts.md §5).
func makeFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fake binary %s: %v", p, err)
	}
	return p
}

// --- resolveSiblingFromExe (the testable core) ---

// Rule 2 CONTRACT: a symlink to the binary in a DIFFERENT dir must resolve back
// to the REAL binary's repo dir's skills/. Mirrors architecture/verified_symlink_resolution.md.
func TestResolveSiblingFromExeSymlinkCrossDir(t *testing.T) {
	// tempA holds the REAL binary + its sibling skills/
	tempA := t.TempDir()
	binary := makeFakeBinary(t, tempA, "skilldozer")
	skillsA := filepath.Join(tempA, "skills")
	if err := os.Mkdir(skillsA, 0o755); err != nil {
		t.Fatal(err)
	}
	// tempB holds a symlink to the binary (different dir, like ~/.local/bin)
	tempB := t.TempDir()
	link := filepath.Join(tempB, "skilldozer")
	if err := os.Symlink(binary, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	got, found := resolveSiblingFromExe(link)
	if !found {
		t.Fatalf("resolveSiblingFromExe(symlink): found=false; want true")
	}
	if got != skillsA {
		t.Errorf("resolveSiblingFromExe(symlink): dir=%q; want the REAL binary's skills %q", got, skillsA)
	}
	if filepath.Dir(got) != tempA {
		t.Errorf("resolveSiblingFromExe(symlink): resolved to %q, not the real binary's dir %q", filepath.Dir(got), tempA)
	}
}

// Rule 2: direct (non-symlinked) binary with a sibling skills/ also wins.
func TestResolveSiblingFromExeDirect(t *testing.T) {
	tempA := t.TempDir()
	binary := makeFakeBinary(t, tempA, "skilldozer")
	skillsA := filepath.Join(tempA, "skills")
	if err := os.Mkdir(skillsA, 0o755); err != nil {
		t.Fatal(err)
	}
	got, found := resolveSiblingFromExe(binary)
	if !found {
		t.Fatalf("resolveSiblingFromExe(direct): found=false; want true")
	}
	if got != skillsA {
		t.Errorf("resolveSiblingFromExe(direct): dir=%q; want %q", got, skillsA)
	}
}

// Rule 2: EvalSymlinks-error fallback. A non-existent exe whose parent dir DOES
// have a sibling skills/ must still win via real=exe. (Contract: 'if err, use
// exe as fallback.')
func TestResolveSiblingFromExeEvalSymlinksFallback(t *testing.T) {
	tempC := t.TempDir()
	skillsC := filepath.Join(tempC, "skills")
	if err := os.Mkdir(skillsC, 0o755); err != nil {
		t.Fatal(err)
	}
	// 'ghost' binary does not exist -> EvalSymlinks errors -> fall back to exe.
	ghost := filepath.Join(tempC, "does-not-exist-binary")
	got, found := resolveSiblingFromExe(ghost)
	if !found {
		t.Fatalf("resolveSiblingFromExe(ghost): found=false; want true (EvalSymlinks fallback to exe)")
	}
	if got != skillsC {
		t.Errorf("resolveSiblingFromExe(ghost): dir=%q; want %q (Dir(exe)/skills)", got, skillsC)
	}
}

// Rule 2: binary exists but NO sibling skills/ dir -> miss.
func TestResolveSiblingFromExeNoSkillsDir(t *testing.T) {
	tempA := t.TempDir()
	binary := makeFakeBinary(t, tempA, "skilldozer")
	// deliberately create no skills/ sibling
	if _, found := resolveSiblingFromExe(binary); found {
		t.Errorf("resolveSiblingFromExe(no skills): got found=true; want false")
	}
}

// Rule 2: sibling path 'skills' is a regular FILE, not a dir -> miss (IsDir guard).
func TestResolveSiblingFromExeSkillsIsFile(t *testing.T) {
	tempA := t.TempDir()
	binary := makeFakeBinary(t, tempA, "skilldozer")
	if err := os.WriteFile(filepath.Join(tempA, "skills"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, found := resolveSiblingFromExe(binary); found {
		t.Errorf("resolveSiblingFromExe(skills is file): got found=true; want false (IsDir guard)")
	}
}

// --- findSibling (the rule-2 entry; os.Executable exercised) ---

// Smoke test: the REAL test binary runs from a temp build dir (go-buildXXX)
// that has NO sibling skills/, so findSibling must return found=false without
// panicking. This is the only deterministic assertion possible for findSibling
// (os.Executable cannot be controlled); the symlink logic is covered by the
// resolveSiblingFromExe tests above.
func TestFindSiblingNoSkillsNextToTestBinary(t *testing.T) {
	dir, src, found := findSibling()
	if found {
		t.Errorf("findSibling(): got found=true dir=%q src=%v; want false (test binary's dir has no sibling skills/)", dir, src)
	}
}

// ---------------------------------------------------------------------------
// Rule 3 tests (walk-up) + Find() combiner tests (P1.M1.T2.S3).
// ---------------------------------------------------------------------------

// makeSkill creates <dir>/skills/<tag>/SKILL.md and returns the skills dir.
func makeSkill(t *testing.T, dir, tag string) string {
	t.Helper()
	skills := filepath.Join(dir, "skills")
	skillDir := filepath.Join(skills, tag)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return skills
}

// --- HasSkillMD ---

func TestHasSkillMDFoundNested(t *testing.T) {
	skills := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(filepath.Join(skills, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skills, "a", "b", "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasSkillMD(skills) {
		t.Errorf("HasSkillMD(nested SKILL.md): got false; want true (WalkDir recurses)")
	}
}

func TestHasSkillMDFoundShallow(t *testing.T) {
	skills := makeSkill(t, t.TempDir(), "foo")
	if !HasSkillMD(skills) {
		t.Errorf("HasSkillMD(shallow SKILL.md): got false; want true")
	}
}

func TestHasSkillMDEmpty(t *testing.T) {
	skills := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	if HasSkillMD(skills) {
		t.Errorf("HasSkillMD(empty skills): got true; want false")
	}
}

func TestHasSkillMDOnlyNonSkillFiles(t *testing.T) {
	skills := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skills, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if HasSkillMD(skills) {
		t.Errorf("HasSkillMD(only README.md): got true; want false (name must be SKILL.md)")
	}
}

// --- findWalkUpAncestor (the testable core; no cwd dependency) ---

// Rule 3: start IS the repo -> skills at start wins (cwd itself is checked first).
func TestFindWalkUpAncestorAtStart(t *testing.T) {
	root := t.TempDir()
	skills := makeSkill(t, root, "foo")
	got, found := findWalkUpAncestor(root)
	if !found {
		t.Fatalf("findWalkUpAncestor(start=repo): found=false; want true")
	}
	if got != skills {
		t.Errorf("findWalkUpAncestor: dir=%q; want %q", got, skills)
	}
}

// Rule 3: skills is several levels up -> ascent finds it.
func TestFindWalkUpAncestorDeep(t *testing.T) {
	root := t.TempDir()
	skills := makeSkill(t, root, "bar")
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, found := findWalkUpAncestor(deep)
	if !found {
		t.Fatalf("findWalkUpAncestor(deep): found=false; want true")
	}
	if got != skills {
		t.Errorf("findWalkUpAncestor(deep): dir=%q; want %q", got, skills)
	}
}

// Rule 3: a nested SKILL.md (skills/x/y/SKILL.md) counts (HasSkillMD recurses).
func TestFindWalkUpAncestorNestedSkillMD(t *testing.T) {
	root := t.TempDir()
	skills := filepath.Join(root, "skills")
	if err := os.MkdirAll(filepath.Join(skills, "x", "y"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skills, "x", "y", "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, found := findWalkUpAncestor(root)
	if !found || got != skills {
		t.Errorf("findWalkUpAncestor(nested): found=%v dir=%q; want true %q", found, got, skills)
	}
}

// Rule 3 CONTRACT: a skills/ dir that exists but has NO SKILL.md is SKIPPED and
// ascent continues to a higher ancestor that DOES have one. PRD §8.3 qualifies
// the match with "at least one SKILL.md".
func TestFindWalkUpAncestorSkipsEmptyAndContinues(t *testing.T) {
	root := t.TempDir()
	// root/a/skills = EMPTY (no SKILL.md); root/skills/foo/SKILL.md = real.
	if err := os.MkdirAll(filepath.Join(root, "a", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	realSkills := makeSkill(t, root, "foo")
	start := filepath.Join(root, "a", "sub")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	got, found := findWalkUpAncestor(start)
	if !found {
		t.Fatalf("findWalkUpAncestor(skip-empty): found=false; want true")
	}
	if got != realSkills {
		t.Errorf("findWalkUpAncestor(skip-empty): dir=%q; want the higher real skills %q", got, realSkills)
	}
}

// Rule 3: no skills anywhere up to root -> miss.
func TestFindWalkUpAncestorNoSkills(t *testing.T) {
	if _, found := findWalkUpAncestor(t.TempDir()); found {
		t.Errorf("findWalkUpAncestor(no skills): got found=true; want false")
	}
}

// Rule 3: a 'skills' entry that is a regular FILE is skipped (IsDir guard).
func TestFindWalkUpAncestorSkillsIsFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "skills"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, found := findWalkUpAncestor(root); found {
		t.Errorf("findWalkUpAncestor(skills is file): got found=true; want false")
	}
}

// --- findWalkUp (the rule-3 entry; os.Getwd exercised via t.Chdir) ---

// t.Chdir (Go 1.24+) changes cwd for the test and restores it on cleanup, so
// findWalkUp (which calls os.Getwd) is testable without global cwd pollution.

// Rule 3 via findWalkUp: chdir into a subdir of a temp repo and confirm walk-up
// resolves to the repo's skills/, returning SourceWalkUp.
func TestFindWalkUpFindsAncestor(t *testing.T) {
	root := t.TempDir()
	skills := makeSkill(t, root, "example")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	got, src, found := findWalkUp()
	if !found {
		t.Fatalf("findWalkUp(): found=false; want true")
	}
	if src != SourceWalkUp {
		t.Errorf("findWalkUp(): src=%v; want SourceWalkUp", src)
	}
	if got != skills {
		t.Errorf("findWalkUp(): dir=%q; want %q", got, skills)
	}
}

// --- Find (the public combiner) ---

// Find: rule 1 wins when SKILLDOZER_SKILLS_DIR is set to an existing dir.
func TestFindRuleEnvWins(t *testing.T) {
	unsetEnvVar(t)
	dir := t.TempDir()
	t.Setenv(envVar, dir)
	got, src, err := Find()
	if err != nil {
		t.Fatalf("Find() env set: err=%v; want nil", err)
	}
	if src != SourceEnv {
		t.Errorf("Find() env set: src=%v; want SourceEnv", src)
	}
	if want := filepath.Clean(dir); got != want {
		t.Errorf("Find() env set: dir=%q; want %q", got, want)
	}
}

// Find: rule 3 wins when env is unset and cwd has an ancestor skills/ with a
// SKILL.md. (findSibling deterministically misses in a test because the test
// binary runs from a temp build dir with no sibling skills/.)
func TestFindRuleWalkUpWins(t *testing.T) {
	unsetEnvVar(t)
	root := t.TempDir()
	skills := makeSkill(t, root, "example")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	got, src, err := Find()
	if err != nil {
		t.Fatalf("Find() walk-up: err=%v; want nil", err)
	}
	if src != SourceWalkUp {
		t.Errorf("Find() walk-up: src=%v; want SourceWalkUp", src)
	}
	if got != skills {
		t.Errorf("Find() walk-up: dir=%q; want %q", got, skills)
	}
}

// Find: all three rules miss -> ErrNotFound. (chdir into an empty temp dir; the
// walk ascends to /, which has no skills/ on this host — verified hermetic.)
func TestFindAllMissReturnsErrNotFound(t *testing.T) {
	unsetEnvVar(t)
	t.Chdir(t.TempDir())
	got, src, err := Find()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find() all-miss: err=%v; want ErrNotFound", err)
	}
	if got != "" || src != 0 {
		t.Errorf("Find() all-miss: got=%q src=%v; want \"\" and 0", got, src)
	}
}

// ErrNotFound message carries the user-facing one-line fix (PRD §8.4 / §6.4).
func TestErrNotFoundMessageHasFix(t *testing.T) {
	msg := ErrNotFound.Error()
	for _, want := range []string{"SKILLDOZER_SKILLS_DIR", "cd", "reinstall"} {
		if !strings.Contains(msg, want) {
			t.Errorf("ErrNotFound message %q missing substring %q", msg, want)
		}
	}
}
