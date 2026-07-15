// index.go implements discover.Index — the on-disk skills/ walk that ties S1's
// ParseFrontmatter (discover.go) and S2's BuildSkill (skill.go) into the []Skill
// the rest of skilldozer consumes (PRD §7.1). This is the P1.M2.T5.S1 deliverable.
// discover.go (S1) owns the frontmatter model/parser; skill.go (S2) owns the Skill
// type + metadata extraction; index.go (T5) owns the recursive directory scan
// (symlink-following and cycle-guarded), the relTag normalization, the sort, and
// the parse-error policy. It is the data source for
// T6 (--list), T7 (resolve), T9 (--search), and T10 (check).
package discover

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Index walks the skills directory at skillsDir and returns every skill it
// contains, as a []Skill sorted by canonical tag (RelTag) for deterministic
// output. It implements PRD §7.1 discovery (manifest-free: the catalog is rebuilt
// from disk on every call — there is no index file).
//
// A "skill" is any directory that directly contains a SKILL.md file; nested skills
// count (skills/writing/reddit/SKILL.md is a skill whose RelTag is
// "writing/reddit"). relTag is the skill dir path relative to skillsDir, with OS
// separators normalized to '/' via filepath.ToSlash — so tags are "writing/reddit"
// on every platform (PRD §7.2 step 1; go_architecture.md "relTag normalization").
//
// skillsDir is made absolute first (filepath.Abs), so every Skill.Dir is an
// absolute path — the contract behind PRD §6.1 ("absolute path") and the §13
// acceptance gate (`case "$(./skilldozer example)" in /*)`). On the canonical absolute
// input (from skillsdir.Find) Abs is a no-op Clean.
//
// Error policy (the decision S2's PRP assigned to T5; see research/verified_facts.md §8):
//   - skillsDir missing, unreadable, or not a directory -> returned as the error.
//     (The caller, main, prints it to stderr and exits 1, PRD §6.4/§8.4.)
//   - A per-entry error (an unreadable subtree) is SKIPPED; the walk continues.
//   - Malformed YAML inside a SKILL.md does NOT abort the walk and is NOT
//     propagated: ParseFrontmatter returns (Frontmatter{}, body, err); Index
//     ignores err and builds a HasFM=false Skill via BuildSkill so the skill is
//     still resolvable by directory/basename (PRD §7.1). check (M4/T10) can
//     re-run ParseFrontmatter(s.SourceFile) to distinguish "malformed YAML" from
//     "no frontmatter block" (idempotent; no rework).
//
// SYMLINKS ARE FOLLOWED. Unlike filepath.WalkDir (which skips linked dirs by
// default), this walk resolves every entry with filepath.EvalSymlinks and recurses
// into linked directories, so a skill dir reachable only through a symlink is still
// discovered. Each visited path is tracked in TWO coordinate systems:
//   - displayPath: the path AS IT APPEARS under skillsDir (symlinks preserved). It
//     backs RelTag (relative to root) and Skill.Dir, so the tags and absolute
//     paths skilldozer reports match what the user wrote on disk (e.g. a skill
//     reached via a link named "agent-browser-pool" is reported under that name,
//     not under the resolved target).
//   - realPath: the EvalSymlinks canonical target. It is what is actually read, and
//     its string form is the cycle-detection key.
// Cycles are broken by a visited set of canonical real paths: a symlink that points
// at an ancestor (or any already-entered dir) resolves to a string already in the
// set and is skipped, so the walk always terminates. (Bind-mount cycles, which
// share an inode but not a path, are out of scope — same as the prior behavior.)
//
// An empty skills dir (no SKILL.md anywhere) yields a nil slice and a nil error;
// callers test with len() (e.g. --list exits 1 "if no skills found").
func Index(skillsDir string) ([]Skill, error) {
	root, err := filepath.Abs(skillsDir)
	if err != nil {
		return nil, err
	}
	// Stat-guard BEFORE the walk: a missing root must propagate as an error, not
	// be swallowed into (nil, nil). os.Stat (not Lstat) follows a symlink, so a
	// skills dir that is itself a link to a directory is accepted. See
	// research/verified_facts.md Run 1 (the bug) vs Run 2 (the fix).
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New(root + ": not a directory")
	}

	// realRoot is the EvalSymlinks target of root: what to READ entries from and
	// the seed of the cycle-detection set. It usually equals a Clean of root (no-
	// op) and only differs when root itself crosses a symlink. Fall back to root
	// on error (EvalSymlinks can fail on exotic filesystems).
	realRoot, rerr := filepath.EvalSymlinks(root)
	if rerr != nil {
		realRoot = root
	}

	var skills []Skill
	visited := map[string]bool{realRoot: true} // canonical real path -> already entered (cycle guard)

	// walk descends realPath (what to read) while threading displayPath (what the
	// user sees, for RelTag/Skill.Dir). See the Index doc comment for the contract.
	var walk func(displayPath, realPath string)
	walk = func(displayPath, realPath string) {
		entries, derr := os.ReadDir(realPath)
		if derr != nil {
			return // unreadable dir -> skip, keep walking (mirrors the prior per-entry policy)
		}
		for _, e := range entries {
			name := e.Name()
			childDisplay := filepath.Join(displayPath, name)
			childReal := filepath.Join(realPath, name)
			isDir := e.IsDir()

			// Follow symlinks: resolve to the canonical target so linked dirs are
			// recursed into and deduped/cycle-broken by real path. Broken or
			// unresolvable links are skipped (Stat/EvalSymlinks error -> continue).
			if e.Type()&fs.ModeSymlink != 0 {
				resolved, rer := filepath.EvalSymlinks(childReal)
				if rer != nil {
					continue
				}
				childReal = resolved
				st, ser := os.Stat(childReal)
				if ser != nil {
					continue
				}
				isDir = st.IsDir()
			}

			// A skill is a directory directly containing a SKILL.md. A directory
			// (plain OR a link to a dir) literally named "SKILL.md" is neither a
			// skill nor recursed into — matches the original IsDir guard.
			if name == "SKILL.md" {
				if isDir {
					continue
				}
				rel, rer := filepath.Rel(root, displayPath)
				if rer != nil {
					continue // displayPath is always under root; unreachable
				}
				relTag := filepath.ToSlash(rel)
				// Lenient parse: malformed/frontmatter-less SKILL.md still yields a
				// resolvable HasFM=false Skill. err is dropped here on purpose (see
				// the doc comment); check (M4) re-parses s.SourceFile to recover it.
				fm, _, _ := ParseFrontmatter(childDisplay)
				skills = append(skills, BuildSkill(displayPath, relTag, fm))
				continue
			}

			if !isDir {
				continue
			}
			// Cycle guard: EvalSymlinks canonicalizes, so a link back at an ancestor
			// resolves to a path already in `visited` -> stop, don't loop forever.
			if visited[childReal] {
				continue
			}
			visited[childReal] = true
			walk(childDisplay, childReal)
		}
	}
	walk(root, realRoot)

	// Deterministic output: sort by canonical tag (PRD §6.1 --all "sorted by tag").
	sort.Slice(skills, func(i, j int) bool { return skills[i].RelTag < skills[j].RelTag })
	return skills, nil
}
