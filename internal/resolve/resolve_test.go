package resolve

import (
	"errors"
	"testing"

	"github.com/dabstractor/skpp/internal/discover"
)

// exampleSkills mirrors the PRD §7.2 example setup EXACTLY: a top-level skill
// `foo` whose frontmatter name is `foo-helper`, and a nested skill
// `writing/reddit`. Only RelTag/Name/Aliases influence resolution; Dir/SourceFile
// are filled with realistic absolute paths so a returned Result.Skill is usable by
// main. reddit is given an alias "social" so the example fixture also exercises
// the alias step without needing a second fixture.
var exampleSkills = []discover.Skill{
	{Dir: "/repo/skills/foo", RelTag: "foo", Name: "foo-helper", SourceFile: "/repo/skills/foo/SKILL.md"},
	{Dir: "/repo/skills/writing/reddit", RelTag: "writing/reddit", Name: "reddit-poster", Aliases: []string{"social"}, SourceFile: "/repo/skills/writing/reddit/SKILL.md"},
}

// TestResolveExamples is THE PRD §7.2 examples table (the item's required test),
// plus the alias step on the same fixture. Each row asserts both the resolved
// RelTag and the MatchKind.
func TestResolveExamples(t *testing.T) {
	cases := []struct {
		tag       string
		wantRel   string
		wantMatch MatchKind
	}{
		{"foo", "foo", Canonical},                       // exact RelTag
		{"writing/reddit", "writing/reddit", Canonical}, // exact RelTag (nested)
		{"reddit", "writing/reddit", Basename},          // basename, unambiguous
		{"foo-helper", "foo", Name},                     // frontmatter name
		{"social", "writing/reddit", Alias},             // declared alias
	}
	for _, c := range cases {
		got, err := Resolve(c.tag, exampleSkills)
		if err != nil {
			t.Errorf("Resolve(%q): err=%v; want nil", c.tag, err)
			continue
		}
		if got.Match != c.wantMatch {
			t.Errorf("Resolve(%q): Match=%v; want %v", c.tag, got.Match, c.wantMatch)
		}
		if got.Skill.RelTag != c.wantRel {
			t.Errorf("Resolve(%q): RelTag=%q; want %q", c.tag, got.Skill.RelTag, c.wantRel)
		}
	}
}

// TestResolveAmbiguous exercises the >1-match case at each of steps 2/3/4. Input
// is deliberately passed in REVERSE sorted order to prove sortedRelTags sorts the
// Candidates regardless of input ordering.
func TestResolveAmbiguous(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		// skills listed REVERSE-sorted so Candidates sorting is observable.
		skills []discover.Skill
		want   []string // expected sorted Candidates
	}{
		{
			name: "basename",
			tag:  "reddit",
			skills: []discover.Skill{
				{RelTag: "writing/reddit", Name: "a"},
				{RelTag: "coding/reddit", Name: "b"},
			},
			want: []string{"coding/reddit", "writing/reddit"},
		},
		{
			name: "name",
			tag:  "dup",
			skills: []discover.Skill{
				{RelTag: "beta", Name: "dup"},
				{RelTag: "alpha", Name: "dup"},
			},
			want: []string{"alpha", "beta"},
		},
		{
			name: "alias",
			tag:  "shared",
			skills: []discover.Skill{
				{RelTag: "beta", Aliases: []string{"shared"}},
				{RelTag: "alpha", Aliases: []string{"shared"}},
			},
			want: []string{"alpha", "beta"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := Resolve(c.tag, c.skills)
			if err == nil {
				t.Fatalf("Resolve(%q) [%s]: err=nil res=%+v; want *AmbiguousError", c.tag, c.name, res)
			}
			ae, ok := err.(*AmbiguousError)
			if !ok {
				t.Fatalf("Resolve(%q) [%s]: err type=%T; want *AmbiguousError", c.tag, c.name, err)
			}
			if ae.Tag != c.tag {
				t.Errorf("Tag=%q; want %q", ae.Tag, c.tag)
			}
			if len(ae.Candidates) != len(c.want) {
				t.Fatalf("Candidates=%v; want %v", ae.Candidates, c.want)
			}
			for i, want := range c.want {
				if ae.Candidates[i] != want {
					t.Errorf("Candidates[%d]=%q; want %q (full=%v)", i, ae.Candidates[i], want, ae.Candidates)
				}
			}
		})
	}
}

// TestResolveUnknown: a tag matching nothing, and an empty/nil index, both yield
// *UnknownError{Tag: tag}. No panic on nil/empty input.
func TestResolveUnknown(t *testing.T) {
	// Unknown tag against the example index.
	res, err := Resolve("nope", exampleSkills)
	if err == nil {
		t.Fatalf("Resolve(nope): err=nil res=%+v; want *UnknownError", res)
	}
	ue, ok := err.(*UnknownError)
	if !ok {
		t.Fatalf("Resolve(nope): err type=%T; want *UnknownError", err)
	}
	if ue.Tag != "nope" {
		t.Errorf("Tag=%q; want nope", ue.Tag)
	}

	// Empty index ⇒ unknown (range over nil/empty is a no-op).
	if _, err := Resolve("anything", nil); err == nil {
		t.Fatal("Resolve(anything, nil): err=nil; want *UnknownError")
	}
	if _, err := Resolve("anything", []discover.Skill{}); err == nil {
		t.Fatal("Resolve(anything, []): err=nil; want *UnknownError")
	}
}

// TestResolvePrecedence: first-match-wins. A tag that matches at an EARLIER step
// must resolve there even if it would also match a later step.
func TestResolvePrecedence(t *testing.T) {
	// Canonical beats Name: skill A has RelTag "x"; skill B has Name "x".
	// tag "x" must resolve to A at step 1 (Canonical), NOT B at step 3 (Name).
	skills := []discover.Skill{
		{RelTag: "x", Name: "a-name", Dir: "/s/x"},
		{RelTag: "y", Name: "x"},
	}
	got, err := Resolve("x", skills)
	if err != nil {
		t.Fatalf("Resolve(x) precedence: err=%v; want nil", err)
	}
	if got.Match != Canonical {
		t.Errorf("Match=%v; want Canonical (step 1 beats step 3)", got.Match)
	}
	if got.Skill.RelTag != "x" {
		t.Errorf("RelTag=%q; want x", got.Skill.RelTag)
	}

	// Canonical beats Basename: a top-level skill "foo" — tag "foo" is BOTH the
	// exact RelTag (step 1) AND its own basename (step 2). Step 1 must win.
	// (Covered by TestResolveExamples "foo"→Canonical; restated for clarity.)
	got2, err := Resolve("foo", exampleSkills)
	if err != nil || got2.Match != Canonical {
		t.Errorf("Resolve(foo): match=%v err=%v; want Canonical (step 1 beats step 2)", got2.Match, err)
	}
}

// TestResolveEmptyTagGuard: a skill with Name=="" (no frontmatter) must NOT match
// step 3, so a degenerate empty tag ("") yields *UnknownError, not a Name hit.
// Also: a skill whose only alias is "" never matches.
func TestResolveEmptyTagGuard(t *testing.T) {
	skills := []discover.Skill{
		{RelTag: "nofm", Name: ""}, // no frontmatter → Name empty
	}
	res, err := Resolve("", skills)
	if err == nil {
		t.Fatalf("Resolve(\"\"): err=nil res=%+v; want *UnknownError (empty Name must not match)", res)
	}
	if _, ok := err.(*UnknownError); !ok {
		t.Fatalf("Resolve(\"\"): err type=%T; want *UnknownError", err)
	}

	// Sanity: a NON-empty tag still resolves normally on the same fixture by basename.
	if _, err := Resolve("nofm", skills); err != nil {
		t.Errorf("Resolve(nofm): err=%v; want nil (basename match)", err)
	}
}

// TestResolveDuplicateAliasCountedOnce: a skill whose Aliases lists the same tag
// twice still counts as ONE match (collectMatches appends each skill at most once),
// so a single such skill resolves cleanly rather than being misread as ambiguous.
func TestResolveDuplicateAliasCountedOnce(t *testing.T) {
	skills := []discover.Skill{
		{RelTag: "alpha", Aliases: []string{"dup", "dup"}},
	}
	got, err := Resolve("dup", skills)
	if err != nil {
		t.Fatalf("Resolve(dup): err=%v; want nil (duplicate alias counts once)", err)
	}
	if got.Match != Alias || got.Skill.RelTag != "alpha" {
		t.Errorf("Resolve(dup): match=%v rel=%q; want Alias/alpha", got.Match, got.Skill.RelTag)
	}
}

// TestResolveErrorsAs: the typed errors must be extractable via errors.As — the
// contract main (P1.M3.T8.S1) relies on to branch on error type and read Candidates.
func TestResolveErrorsAs(t *testing.T) {
	ambig := []discover.Skill{
		{RelTag: "writing/reddit"},
		{RelTag: "coding/reddit"},
	}
	_, err := Resolve("reddit", ambig)

	var ae *AmbiguousError
	if !errors.As(err, &ae) {
		t.Fatalf("errors.As(*AmbiguousError)=false for %T; want true", err)
	}
	if ae.Tag != "reddit" || len(ae.Candidates) != 2 {
		t.Errorf("extracted AmbiguousError=%+v; want Tag=reddit, 2 candidates", ae)
	}

	_, err = Resolve("nope", exampleSkills)
	var ue *UnknownError
	if !errors.As(err, &ue) {
		t.Fatalf("errors.As(*UnknownError)=false for %T; want true", err)
	}
	if ue.Tag != "nope" {
		t.Errorf("extracted UnknownError.Tag=%q; want nope", ue.Tag)
	}

	// Negative: an UnknownError must NOT masquerade as an AmbiguousError.
	_, err = Resolve("nope", exampleSkills)
	var wrong *AmbiguousError
	if errors.As(err, &wrong) {
		t.Error("errors.As(*AmbiguousError)=true on an UnknownError; want false")
	}
}

// TestErrorMessages: exact .Error() text (we own the format strings). main may
// reformat for §6.4, but the package's own rendering must be stable.
func TestErrorMessages(t *testing.T) {
	if got := (&UnknownError{Tag: "foo"}).Error(); got != `unknown skill tag "foo"` {
		t.Errorf("UnknownError.Error()=%q; want %q", got, `unknown skill tag "foo"`)
	}
	got := (&AmbiguousError{Tag: "reddit", Candidates: []string{"coding/reddit", "writing/reddit"}}).Error()
	want := `ambiguous skill tag "reddit" matches: coding/reddit, writing/reddit`
	if got != want {
		t.Errorf("AmbiguousError.Error()=%q; want %q", got, want)
	}
}

// TestMatchKindString mirrors skillsdir's TestSourceString: each constant renders,
// and an out-of-range value renders as "unknown".
func TestMatchKindString(t *testing.T) {
	cases := []struct {
		m    MatchKind
		want string
	}{
		{Canonical, "canonical"},
		{Basename, "basename"},
		{Name, "name"},
		{Alias, "alias"},
		{MatchKind(-1), "unknown"},
		{MatchKind(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("MatchKind(%d).String()=%q; want %q", c.m, got, c.want)
		}
	}
}

// TestBasename: the private slash-split helper (covers the no-slash and
// multi-level cases directly, independent of Resolve).
func TestBasename(t *testing.T) {
	cases := []struct{ relTag, want string }{
		{"writing/reddit", "reddit"},
		{"foo", "foo"}, // no slash → whole string
		{"a/b/c", "c"}, // multi-level → last
		{"", ""},       // degenerate
	}
	for _, c := range cases {
		if got := basename(c.relTag); got != c.want {
			t.Errorf("basename(%q)=%q; want %q", c.relTag, got, c.want)
		}
	}
}
