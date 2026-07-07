// Package search filters a []discover.Skill by a case-insensitive substring
// query over the four fields PRD §6.1 names for `skpp --search`: the tag, the
// frontmatter name, the description, and each metadata keyword. It is a PURE
// function over []discover.Skill: no filesystem, no globals, no I/O — main
// (P1.M4.T9.S1) supplies the index from discover.Index and passes the filtered
// (still-sorted) slice to ui.PrintList for the "same table format as --list"
// rendering (PRD §6.1).
//
// It mirrors internal/resolve (also a pure matching function over
// []discover.Skill, in its own package with its own _test.go) so the two matching
// concerns — precise tag resolution (resolve) and fuzzy catalog search (search) —
// stay isolated, independently unit-testable, and out of the thin main dispatcher.
package search

import (
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
)

// Search returns every skill in skills for which query is a case-insensitive
// substring of ANY of the four PRD §6.1 fields: RelTag (the tag), Name
// (frontmatter name), Description, or any element of Keywords. Input order is
// preserved: discover.Index sorts []Skill by RelTag, and ui.PrintList does NOT
// re-sort, so the filtered slice is displayed already-sorted.
//
// An empty query matches EVERY skill: strings.Contains(hay, "") is always true,
// so `skpp --search ""` behaves like `skpp --list` (exit 1 only if the store is
// empty). This is the natural substring semantics; the PRD carves out no special
// case for an empty query.
//
// A skill with no frontmatter (HasFM==false) has Name=="" and Description=="" and
// nil Keywords, but its RelTag is always present — so it is still discoverable by
// searching its tag, matching how resolve lets a frontmatter-less skill resolve
// by directory/basename (PRD §7.1). Only RelTag is searchable for such a skill.
func Search(query string, skills []discover.Skill) []discover.Skill {
	q := strings.ToLower(query) // lowercase the query ONCE, not per field
	var matched []discover.Skill
	for _, s := range skills {
		if matches(q, s) {
			matched = append(matched, s)
		}
	}
	return matched
}

// matches reports whether the already-lowercased query q is a case-insensitive
// substring of any searchable field of s. q is lowercased once by the caller
// (Search); each field is lowercased lazily inside Contains.
//
// Field scope is EXACTLY the four PRD §6.1 fields. It deliberately does NOT
// include Category or Aliases — both exist on discover.Skill and would be a
// tempting (but spec-violating) addition. PRD §6.1: "tag, frontmatter name,
// description, and metadata.keywords".
//
// Keywords are tested INDIVIDUALLY (not strings.Join'd): a query spanning a
// boundary between two keywords must not match (joining would create false
// positives like "wri"+"ocial" => "wriocial" across "writing","social").
func matches(q string, s discover.Skill) bool {
	if strings.Contains(strings.ToLower(s.RelTag), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Description), q) {
		return true
	}
	for _, kw := range s.Keywords {
		if strings.Contains(strings.ToLower(kw), q) {
			return true
		}
	}
	return false
}
