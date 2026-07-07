package search

import (
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

// sk builds one discover.Skill with the searchable fields set (HasFM true so all
// fields are "real"). Mirrors ui_test.go's mk but lets keywords be set. Use len()
// on Keywords (never a nil check) per the discover contract.
func sk(tag, name, desc string, keywords ...string) discover.Skill {
	return discover.Skill{
		RelTag:      tag,
		Name:        name,
		Description: desc,
		Keywords:    keywords,
		HasFM:       true,
	}
}

func TestSearchMatchByTag(t *testing.T) {
	in := []discover.Skill{sk("writing/reddit", "rp", "d")}
	out := Search("writing/reddit", in)
	if len(out) != 1 || out[0].RelTag != "writing/reddit" {
		t.Errorf("exact tag match failed: got %+v", out)
	}
}

func TestSearchMatchByTagSubstring(t *testing.T) {
	in := []discover.Skill{sk("writing/reddit", "rp", "d")}
	out := Search("redd", in)
	if len(out) != 1 {
		t.Errorf("tag substring 'redd' should match writing/reddit: got %+v", out)
	}
}

func TestSearchMatchByBasenameAsSubstring(t *testing.T) {
	in := []discover.Skill{sk("writing/reddit", "rp", "d")}
	out := Search("reddit", in) // basename is part of the relTag string
	if len(out) != 1 {
		t.Errorf("'reddit' substring of relTag should match: got %+v", out)
	}
}

func TestSearchMatchByName(t *testing.T) {
	in := []discover.Skill{sk("a", "reddit-poster", "d")}
	out := Search("poster", in)
	if len(out) != 1 {
		t.Errorf("name substring 'poster' should match: got %+v", out)
	}
}

func TestSearchMatchByDescription(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "Posts messages to social media")}
	out := Search("social", in)
	if len(out) != 1 {
		t.Errorf("description substring 'social' should match: got %+v", out)
	}
}

func TestSearchMatchByKeyword(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "d", "writing", "social")}
	out := Search("soc", in)
	if len(out) != 1 {
		t.Errorf("keyword substring 'soc' should match keyword 'social': got %+v", out)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	in := []discover.Skill{sk("Reddit", "Name", "Desc")}
	for _, q := range []string{"reddit", "REDDIT", "rEdDiT", "name", "DESC"} {
		if out := Search(q, in); len(out) != 1 {
			t.Errorf("case-insensitive query %q should match; got %+v", q, out)
		}
	}
}

func TestSearchNoMatchReturnsEmpty(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "d", "k")}
	if out := Search("zzznotfound", in); len(out) != 0 {
		t.Errorf("no-match query should return empty slice; got %+v", out)
	}
}

func TestSearchEmptyQueryMatchesAll(t *testing.T) {
	in := []discover.Skill{sk("a", "n", "d"), sk("b", "m", "e")}
	if out := Search("", in); len(out) != 2 {
		t.Errorf("empty query should match all; got %d", len(out))
	}
}

func TestSearchPreservesInputOrder(t *testing.T) {
	in := []discover.Skill{
		sk("zebra", "n", "match"),
		sk("apple", "n", "match"),
		sk("mango", "n", "unrelated"),
	}
	out := Search("match", in) // matches zebra + apple by description, in that order
	if len(out) != 2 || out[0].RelTag != "zebra" || out[1].RelTag != "apple" {
		t.Errorf("order not preserved: got %+v", out)
	}
}

func TestSearchMultipleMatchesAllReturned(t *testing.T) {
	in := []discover.Skill{
		sk("a", "x", "common"),
		sk("b", "x", "unrelated"),
		sk("c", "common", "y"),
	}
	out := Search("common", in)
	if len(out) != 2 {
		t.Errorf("expected 2 matches across desc+name; got %d: %+v", len(out), out)
	}
}

func TestSearchNoFrontmatterStillMatchesByTag(t *testing.T) {
	// HasFM false => Name/Description empty, Keywords nil, but RelTag present.
	in := []discover.Skill{{RelTag: "bare-skill", HasFM: false}}
	out := Search("bare", in)
	if len(out) != 1 {
		t.Errorf("frontmatter-less skill must still match by tag; got %+v", out)
	}
}

func TestSearchDoesNotMatchCategoryOrAliases(t *testing.T) {
	// PRD §6.1 scopes search to tag/name/description/keywords ONLY. Category and
	// Aliases are on the struct but must NOT be searched.
	in := []discover.Skill{
		{RelTag: "x", Name: "n", Description: "d", Category: "secret-cat", Aliases: []string{"secret-alias"}, HasFM: true},
	}
	if out := Search("secret", in); len(out) != 0 {
		t.Errorf("search must NOT match category/aliases (PRD §6.1 scope); got %+v", out)
	}
}

func TestSearchKeywordSubstringNotJoinBoundary(t *testing.T) {
	// Keywords are matched INDIVIDUALLY, not joined — so a query spanning a
	// boundary between two keywords must NOT match. "wriocial" is not a substring
	// of "writing" nor of "social" individually.
	in := []discover.Skill{sk("a", "n", "d", "writing", "social")}
	if out := Search("wriocial", in); len(out) != 0 {
		t.Errorf("keyword-boundary query must not match (keywords searched individually): got %+v", out)
	}
}

func TestSearchNilInput(t *testing.T) {
	if out := Search("x", nil); len(out) != 0 {
		t.Errorf("nil input should yield empty; got %+v", out)
	}
}

func TestSearchReturnsDistinctResults(t *testing.T) {
	// A skill matching in MULTIPLE fields (e.g. tag AND description) is returned
	// ONCE, not duplicated.
	in := []discover.Skill{sk("match", "n", "match")}
	out := Search("match", in)
	if len(out) != 1 {
		t.Errorf("multi-field match should return the skill once; got %d", len(out))
	}
}
