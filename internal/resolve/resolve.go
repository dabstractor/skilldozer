// Package resolve maps a user-supplied tag to a single skill using the PRD §7.2
// precedence. It is a PURE function over []discover.Skill: no filesystem, no
// global state, no I/O — it takes the already-built index as a parameter and main
// (P1.M3.T8.S1) supplies it from discover.Index().
//
// The precedence (first match wins; a later step is consulted ONLY if every
// earlier step produced nothing) is, in order:
//
//  1. Canonical — tag == skill.RelTag (case-sensitive). RelTag is unique per
//     directory, so at most one hit.
//  2. Basename  — tag == the final '/'-component of skill.RelTag (e.g. "reddit"
//     matches "writing/reddit"). >1 hit ⇒ *AmbiguousError.
//  3. Name      — tag == skill.Name (the frontmatter name). >1 ⇒ *AmbiguousError.
//  4. Alias     — tag appears in skill.Aliases (metadata.aliases). >1 ⇒ *Ambiguous.
//  5. otherwise — *UnknownError.
//
// An ambiguity at any step SHORT-CIRCUITS: *AmbiguousError is returned immediately
// and later steps are NOT consulted (a looser match cannot rescue an ambiguity;
// the caller must see the candidates per PRD §6.4).
//
// AmbiguousError.Candidates is the SORTED list of the matching skills' RelTags —
// sorted here so the error is deterministic regardless of how the caller ordered
// the input slice (PRD §6.4 wants stable stderr for scripting).
package resolve

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

// MatchKind identifies which §7.2 step resolved a tag. Its zero value is not a
// valid success; Resolve always sets it on the success path. Exported so callers
// can switch on it (e.g. --list/debug could annotate "reddit (basename)").
type MatchKind int

const (
	// Canonical means tag == skill.RelTag (step 1, exact canonical tag).
	Canonical MatchKind = iota
	// Basename means tag == the final '/'-component of skill.RelTag (step 2).
	Basename
	// Name means tag == skill.Name, the frontmatter name (step 3).
	Name
	// Alias means tag appeared in skill.Aliases (step 4).
	Alias
)

// String renders a MatchKind for logs/debug, mirroring skillsdir.Source.String().
// An out-of-range value renders as "unknown".
func (m MatchKind) String() string {
	switch m {
	case Canonical:
		return "canonical"
	case Basename:
		return "basename"
	case Name:
		return "name"
	case Alias:
		return "alias"
	default:
		return "unknown"
	}
}

// Result is the outcome of resolving one tag. The zero value Result{} is NOT a
// valid success: Resolve returns it only together with a non-nil error.
type Result struct {
	Skill discover.Skill
	Match MatchKind
}

// UnknownError is returned by Resolve when no §7.2 step matched the tag. main
// prints it to stderr and exits 1 (PRD §6.4).
type UnknownError struct {
	Tag string
}

// Error implements error. No "skpp:" prefix (main adds program context, mirroring
// skillsdir.ErrNotFound).
func (e *UnknownError) Error() string {
	return fmt.Sprintf("unknown skill tag %q", e.Tag)
}

// AmbiguousError is returned when a short tag matched >1 skill at the SAME
// precedence step. Candidates is the sorted list of the matching skills' RelTags
// (the full canonical tags the user can use to disambiguate). main lists them on
// stderr and exits 1 (PRD §6.4).
type AmbiguousError struct {
	Tag        string
	Candidates []string
}

// Error implements error. Candidates are joined with ", " for a readable line.
func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("ambiguous skill tag %q matches: %s", e.Tag, strings.Join(e.Candidates, ", "))
}

// Resolve applies the PRD §7.2 precedence to tag against skills and returns the
// single matching skill, or a typed error (*UnknownError / *AmbiguousError).
//
// It is pure: it does not touch the filesystem or mutate skills. It consults each
// precedence step only if every earlier step produced no match. An ambiguity at
// any step returns *AmbiguousError immediately (later steps are NOT consulted).
//
// Field-level gotcha: step 3 (Name) and step 4 (Alias) only consider a skill whose
// relevant field is non-empty. A skill with no frontmatter (Name=="") is never
// matched by name, and an empty alias never matches; this prevents a degenerate
// empty tag (or a missing-name skill) from spuriously resolving. RelTag and its
// basename are always non-empty for a real skill, so steps 1–2 need no guard.
func Resolve(tag string, skills []discover.Skill) (Result, error) {
	// Step 1 — exact canonical tag. RelTag is unique per directory ⇒ at most one.
	// First (only) match wins; no ambiguity is possible at this step.
	for _, s := range skills {
		if s.RelTag == tag {
			return Result{Skill: s, Match: Canonical}, nil
		}
	}

	// Step 2 — basename (final '/'-component of RelTag).
	if m := collectMatches(skills, func(s discover.Skill) bool {
		return basename(s.RelTag) == tag
	}); len(m) == 1 {
		return Result{Skill: m[0], Match: Basename}, nil
	} else if len(m) > 1 {
		return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)}
	}

	// Step 3 — frontmatter name (skip skills with no name: a missing name is not
	// a matchable name, and this guards against an empty tag matching Name=="").
	if m := collectMatches(skills, func(s discover.Skill) bool {
		return s.Name != "" && s.Name == tag
	}); len(m) == 1 {
		return Result{Skill: m[0], Match: Name}, nil
	} else if len(m) > 1 {
		return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)}
	}

	// Step 4 — declared alias.
	if m := collectMatches(skills, func(s discover.Skill) bool {
		for _, a := range s.Aliases {
			if a == tag {
				return true
			}
		}
		return false
	}); len(m) == 1 {
		return Result{Skill: m[0], Match: Alias}, nil
	} else if len(m) > 1 {
		return Result{}, &AmbiguousError{Tag: tag, Candidates: sortedRelTags(m)}
	}

	// Step 5 — nothing matched.
	return Result{}, &UnknownError{Tag: tag}
}

// collectMatches returns every skill for which pred returns true, in input order.
// It is the shared collection loop for steps 2–4 (step 1 is exact-and-unique, so
// it is inlined in Resolve). Each skill appears at most once: pred is a property
// of the skill, so it is true or false, never "twice".
func collectMatches(skills []discover.Skill, pred func(discover.Skill) bool) []discover.Skill {
	var hit []discover.Skill
	for _, s := range skills {
		if pred(s) {
			hit = append(hit, s)
		}
	}
	return hit
}

// basename returns the final '/'-component of a slash-normalized relTag (e.g.
// "writing/reddit" → "reddit"). relTag is always slash-normalized by discover
// (filepath.ToSlash), so splitting on '/' is correct on every platform and no
// OS-separator handling is needed here. A tag with no '/' is its own basename.
// Uses strings.LastIndex (zero-alloc) rather than path.Base to stay faithful to
// the item's "split on /, take last element" and avoid importing "path".
func basename(relTag string) string {
	if i := strings.LastIndex(relTag, "/"); i >= 0 {
		return relTag[i+1:]
	}
	return relTag
}

// sortedRelTags returns the RelTags of skills, sorted ascending. Used for
// AmbiguousError.Candidates so the error is deterministic regardless of the input
// slice order (PRD §6.4 wants stable stderr for scripting).
func sortedRelTags(skills []discover.Skill) []string {
	tags := make([]string, len(skills))
	for i, s := range skills {
		tags[i] = s.RelTag
	}
	sort.Strings(tags)
	return tags
}
