// Package skillsdir locates the on-disk skills/ directory for skilldozer.
//
// It implements the PRD §8.3 priority order (first hit wins):
//
//  1. SKILLDOZER_SKILLS_DIR env var — override; if set and an existing dir, use it as-is.
//  2. Config file `store` (PRD §8.1) — the primary, set by `skilldozer --init`.
//  3. Sibling of the running binary (symlink-aware via os.Executable + EvalSymlinks).
//  4. Walk up from the current working directory.
//  5. None ⇒ unconfigured: Find returns ErrNotFound; the caller prints a one-line
//     fix to stderr and exits 1.
//
// The public entry point is Find(), which calls the per-rule helpers in order and
// returns the first hit. Each per-rule helper returns (dir string, src Source,
// found bool) where found is true only when that rule produced a usable absolute
// directory; on found==false the src value is meaningless and the caller falls
// through to the next rule. Source.String() labels each rule for `--path` stderr
// reporting (PRD §8.3): SKILLDOZER_SKILLS_DIR, config file, sibling of binary,
// ancestor of cwd.
package skillsdir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dabstractor/skilldozer/internal/config"
)

// Source identifies which §8 rule located the skills directory. It is reported
// by `skilldozer --path` so users can tell how the dir was found.
type Source int

const (
	// SourceEnv means SKILLDOZER_SKILLS_DIR was set and pointed at an existing dir.
	SourceEnv Source = iota
	// SourceConfig means the skills dir was read from the config file's `store` key (PRD §8.1).
	SourceConfig
	// SourceSibling means the skills dir was found next to the running binary.
	SourceSibling
	// SourceWalkUp means the skills dir was found by walking up from cwd.
	SourceWalkUp
)

// String returns a human-readable label for the rule that won, used by
// `skilldozer --path` reporting. Satisfies fmt.Stringer.
func (s Source) String() string {
	switch s {
	case SourceEnv:
		return "SKILLDOZER_SKILLS_DIR"
	case SourceConfig:
		return "config file"
	case SourceSibling:
		return "sibling of binary"
	case SourceWalkUp:
		return "ancestor of cwd"
	default:
		return "unknown"
	}
}

// envVar is the environment variable consulted by rule 1. It is a package
// constant (not a parameter): the contract is "mock/replace nothing" — tests
// drive it via t.Setenv / os.Unsetenv, never via injection.
const envVar = "SKILLDOZER_SKILLS_DIR"

// findEnv implements PRD §8 rule 1.
//
// It reads SKILLDOZER_SKILLS_DIR; if the value names an existing directory it returns
// that directory as an absolute path with src=SourceEnv and found=true. The env
// path is NOT passed through filepath.EvalSymlinks: the user points exactly
// where they want (a symlink is preserved verbatim, only made absolute/clean
// via filepath.Abs). If the variable is unset, empty, or does not name an
// existing directory, it returns found=false with src's zero value so Find()
// can fall through to rule 2 — a bad env value never hard-errors.
func findEnv() (dir string, src Source, found bool) {
	val, ok := os.LookupEnv(envVar)
	if !ok || val == "" {
		return "", 0, false
	}
	info, err := os.Stat(val)
	if err != nil || !info.IsDir() {
		return "", 0, false // not an existing dir -> let the next rule try
	}
	abs, err := filepath.Abs(val)
	if err != nil {
		return "", 0, false // cwd unresolvable -> let the next rule try
	}
	return abs, SourceEnv, true
}

// findConfig implements PRD §8.3 rule 2 — the config file's `store` key (PRD §8.1).
//
// It is the primary discovery rule, set by `skilldozer --init`. config.Path() gives the
// one fixed, well-known bootstrap path ($SKILLDOZER_CONFIG or $XDG_CONFIG_HOME/skilldozer/
// config.yaml); config.Load() reads+unmarshals it (unknown keys ignored; broken YAML is a
// hard error). findConfig treats ANY error from either as "not yet configured -> fall
// through" — PRD §8.1: a missing/unreadable config NEVER hard-errors.
//
// A relative `store` is resolved against the config file's own directory (PRD §8.1:
// store may be relative to the config file), NOT against cwd. The resolved store must
// name an existing directory or the rule misses.
//
// Returns (absStore, SourceConfig, true, "") on a hit. Returns ("", 0, false, "") on the
// genuine fall-through paths (no config file / unreadable / no `store` key) so Find() can
// fall through to the sibling rule — PRD §8.1: a missing/unreadable config NEVER hard-errors.
// Returns ("", 0, false, vanishedStore) ONLY when the config file is present with a `store:`
// key whose resolved dir does not exist (§6.4 "configured but the dir vanished") — Find()
// turns that into a wrapped ErrConfiguredStoreMissing instead of falling through. Never
// errors directly (locked per-rule shape); the vanishedStore string is the signal Find() uses.
func findConfig() (dir string, src Source, found bool, vanishedStore string) {
	p, err := config.Path()
	if err != nil {
		return "", 0, false, "" // no bootstrap path (e.g. relative $XDG_CONFIG_HOME) -> fall through
	}
	f, err := config.Load(p)
	if err != nil {
		return "", 0, false, "" // missing/unreadable/malformed -> "not yet configured" -> fall through
	}
	if f.Store == "" {
		return "", 0, false, "" // no `store` key -> fall through
	}
	var store string
	if filepath.IsAbs(f.Store) {
		store = filepath.Clean(f.Store)
	} else {
		store = filepath.Join(filepath.Dir(p), f.Store) // relative to config file's dir (PRD §8.1)
	}
	info, err := os.Stat(store)
	if err != nil || !info.IsDir() {
		return "", 0, false, store // store resolved but vanished -> Find() errors (§6.4), do NOT fall through
	}
	return store, SourceConfig, true, ""
}

// findSibling implements PRD §8 rule 2 — locate <repoDir>/skills next to the
// running binary, symlink-aware. This is the rule that makes a symlink install
// work: ~/.local/bin/skilldozer -> ~/projects/skilldozer/skilldozer resolves back to the repo.
//
// It is a thin entry that asks the OS for the running binary (os.Executable)
// and delegates the symlink/dir logic to resolveSiblingFromExe. os.Executable
// cannot be controlled in a test (it returns the test binary's own path), so
// the testable core lives in resolveSiblingFromExe.
//
// Returns (candidate, SourceSibling, true) on a hit; ("", 0, false) otherwise so
// Find() (S3) can fall through to rule 3. Never errors (locked per-rule shape).
func findSibling() (dir string, src Source, found bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", 0, false // no binary path at all -> skip rule
	}
	d, ok := resolveSiblingFromExe(exe)
	if !ok {
		return "", 0, false
	}
	return d, SourceSibling, true
}

// resolveSiblingFromExe is the symlink-aware sibling-resolution core for rule 2,
// factored out so it can be unit-tested with arbitrary exe paths.
//
// PRD §8.2 sequence:
//
//	real, err := filepath.EvalSymlinks(exe)  // REQUIRED on macOS (redundant but
//	                                         //   harmless on Linux via /proc/self/exe)
//	if err != nil { real = exe }             // fall back to raw exe on EvalSymlinks error
//	repoDir := filepath.Dir(real)
//	candidate := filepath.Join(repoDir, "skills")
//	win iff os.Stat(candidate) reports an existing directory
//
// See architecture/verified_symlink_resolution.md for why EvalSymlinks must stay.
func resolveSiblingFromExe(exe string) (dir string, found bool) {
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		real = exe // EvalSymlinks could not resolve -> use exe verbatim
	}
	repoDir := filepath.Dir(real)
	candidate := filepath.Join(repoDir, "skills")
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return "", false // no existing skills/ sibling -> rule misses
	}
	return candidate, true
}

// ---------------------------------------------------------------------------
// Rule 3 — walk up from the current working directory (PRD §8.3).
//
// Used for `go run` / dev, where the binary lives in a temp build dir and
// rules 1-2 both miss. The first ancestor (including cwd) whose skills/ subdir
// contains at least one SKILL.md (at any depth) wins.
// ---------------------------------------------------------------------------

// errSkillMDFound is a sentinel error used to short-circuit filepath.WalkDir as
// soon as the first SKILL.md is found, so HasSkillMD does not walk the entire
// tree. Returning any non-nil error from a WalkDir callback stops the walk.
var errSkillMDFound = errors.New("SKILL.md found")

// HasSkillMD reports whether dir contains at least one SKILL.md at any depth.
// It walks the tree under dir but returns true as soon as it finds one (early
// exit via the errSkillMDFound sentinel). PRD §8.3 requires "at least one
// SKILL.md" — a skills/ dir with none does not count.
//
// NOTE: filepath.Glob with a "**" pattern is intentionally NOT used: Go's
// path/filepath does not support "**" (it behaves like single-level "*"), so
// Glob("skills/**/SKILL.md") matches nothing for a nested file. WalkDir is the
// correct stdlib tool and recurses to arbitrary depth.
//
// Exported because it doubles as the §8.2 cwd-auto-detect predicate: `skilldozer
// init` (P1.M2.T2.S1) uses it to decide whether the current working directory
// already looks like a store.
func HasSkillMD(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entry, keep walking
		}
		if !d.IsDir() && d.Name() == "SKILL.md" {
			found = true
			return errSkillMDFound // stop the walk
		}
		return nil
	})
	return found
}

// findWalkUpAncestor implements the ascent core of rule 3, factored out so it
// can be tested with an arbitrary start directory (os.Getwd itself cannot be
// controlled without chdir; findWalkUpAncestor takes start as a parameter).
//
// It checks start, then each ancestor, for a skills/ subdir that contains at
// least one SKILL.md. A skills/ dir that exists but has no SKILL.md is skipped
// and ascent continues (PRD §8.3 qualifies the match with "at least one
// SKILL.md"). Ascent stops at the filesystem root, where filepath.Dir(root)
// equals root.
func findWalkUpAncestor(start string) (dir string, found bool) {
	cur := filepath.Clean(start)
	for {
		candidate := filepath.Join(cur, "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if HasSkillMD(candidate) {
				return candidate, true
			}
			// skills/ exists here but has no SKILL.md -> keep ascending.
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false // reached filesystem root, no match
		}
		cur = parent
	}
}

// findWalkUp implements PRD §8 rule 3 — ascend from the current working
// directory and return the first ancestor whose skills/ subdir contains at
// least one SKILL.md. This is the rule that makes `go run` work when the binary
// is in a temp build dir and rules 1-2 miss.
//
// Returns (candidate, SourceWalkUp, true) on a hit; ("", 0, false) otherwise so
// Find() can return ErrNotFound. Never errors (matches the locked per-rule shape).
func findWalkUp() (dir string, src Source, found bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", 0, false // cwd unresolvable -> rule misses
	}
	d, ok := findWalkUpAncestor(cwd)
	if !ok {
		return "", 0, false
	}
	return d, SourceWalkUp, true
}

// ---------------------------------------------------------------------------
// Find — the public entry point (PRD §8 priority order).
// ---------------------------------------------------------------------------

// ErrNotFound is returned by Find when every §8.3 rule misses (unconfigured). Its
// message is the user-facing one-line fix (PRD §8.4 / §6.4): main prints it to
// stderr and exits 1. Print it verbatim (err.Error()); do not wrap or prefix it.
var ErrNotFound = errors.New("skilldozer is not configured; run `skilldozer --init`")

// ErrConfiguredStoreMissing is returned by Find when the config file is present and
// names a `store:` directory that does NOT exist on disk (§6.4: "configured but the
// dir vanished"). Unlike a missing config file (§8.1 fall-through), this is a loud
// error so `pi --skill "$(skilldozer x)"` fails instead of silently loading an
// unrelated sibling/walk-up store. Find wraps it via fmt.Errorf("%w: ...", ...) so
// callers can errors.Is(err, ErrConfiguredStoreMissing). Print verbatim to stderr.
var ErrConfiguredStoreMissing = errors.New("configured skills store directory does not exist")

// Find locates the skills directory per PRD §8.3 priority order (first hit wins):
//
//  1. SKILLDOZER_SKILLS_DIR env var (SourceEnv).
//  2. Config file `store` (SourceConfig).
//  3. Sibling of the running binary, symlink-aware (SourceSibling).
//  4. Walk up from cwd (SourceWalkUp).
//  5. None ⇒ unconfigured: returns ErrNotFound.
//
// NOTE (§6.4): if the config file is present with a `store:` naming a non-existent dir,
// Find returns a wrapped ErrConfiguredStoreMissing (configuring-then-vanished) instead of
// falling through — so a vanished store never silently resolves tags from an unrelated
// sibling/walk-up store. A genuinely missing/unreadable config still falls through (§8.1).
//
// The first rule to hit wins and Find returns (absDir, src, nil). If all miss it
// returns ("", 0, ErrNotFound); the caller (main) prints the error to stderr and
// exits 1.
func Find() (dir string, src Source, err error) {
	if d, s, ok := findEnv(); ok {
		return d, s, nil
	}
	d, s, ok, vanished := findConfig() // PRD §8.3 priority #2
	if ok {
		return d, s, nil
	}
	if vanished != "" {
		// Config is present with a `store:` key, but the named dir does not exist (§6.4:
		// "configured but the dir vanished"). This is NOT "unconfigured" — falling through
		// to sibling/walk-up would silently load an UNRELATED store, so fail loudly instead.
		return "", 0, fmt.Errorf("%w: configured store %q does not exist; run `skilldozer --init` or recreate the directory", ErrConfiguredStoreMissing, vanished)
	}
	if d, s, ok := findSibling(); ok {
		return d, s, nil
	}
	if d, s, ok := findWalkUp(); ok {
		return d, s, nil
	}
	return "", 0, ErrNotFound
}
