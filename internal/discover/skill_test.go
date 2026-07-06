package discover

import (
	"path/filepath"
	"strings"
	"testing"
)

// NOTE: writeSkill is defined in discover_test.go (same package) and REUSED here.
// Do NOT redefine it. It writes content to a temp SKILL.md and returns its path.

// strEq compares two string slices by length + elements (nil and []string{} are
// both "empty" here; callers that care about nil-vs-empty assert len() directly).
func strEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- toStringSlice (table-driven; verified empirically) ---

func TestToStringSlice(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"any-slice-of-strings", []any{"a", "b"}, []string{"a", "b"}},
		{"any-slice-skips-non-strings", []any{"a", 2, "b", 3.14, true}, []string{"a", "b"}},
		{"any-slice-empty", []any{}, []string{}}, // present-but-empty -> non-nil empty (len 0)
		{"string-slice-passthrough", []string{"x", "y"}, []string{"x", "y"}},
		{"single-string", "solo", []string{"solo"}},
		{"int", 42, nil},
		{"map", map[string]any{"a": 1}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := toStringSlice(c.in)
			// Compare by length + element (the meaningful contract). This treats
			// nil and []string{} as equal (both len 0), matching the documented
			// "callers use len()" rule rather than pinning nil-vs-empty.
			if len(got) != len(c.want) {
				t.Fatalf("%s: len(got)=%d; want %d (got=%#v want=%#v)", c.name, len(got), len(c.want), got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("%s: got[%d]=%q; want %q", c.name, i, got[i], c.want[i])
				}
			}
		})
	}
}

// --- BuildSkill ---

func TestBuildSkillFull(t *testing.T) {
	// metadata lists are []any (what yaml.v3 actually produces); a non-convention
	// key ("unrelated") must be ignored.
	fm := Frontmatter{
		Name:        "example",
		Description: "An example.",
		Metadata: map[string]any{
			"keywords":  []any{"example", "demo", "skpp"},
			"category":  "meta",
			"aliases":   []any{"ex", "demo-skill"},
			"unrelated": 7, // ignored
		},
		HasFM: true,
	}
	s := BuildSkill("/a/skills/example", "example", fm)
	if s.Dir != "/a/skills/example" {
		t.Errorf("Dir=%q; want /a/skills/example", s.Dir)
	}
	if s.RelTag != "example" {
		t.Errorf("RelTag=%q; want example", s.RelTag)
	}
	if s.Name != "example" {
		t.Errorf("Name=%q; want example", s.Name)
	}
	if s.Description != "An example." {
		t.Errorf("Description=%q; want 'An example.'", s.Description)
	}
	if !strEq(s.Keywords, []string{"example", "demo", "skpp"}) {
		t.Errorf("Keywords=%v; want [example demo skpp] (real []any path)", s.Keywords)
	}
	if s.Category != "meta" {
		t.Errorf("Category=%q; want meta", s.Category)
	}
	if !strEq(s.Aliases, []string{"ex", "demo-skill"}) {
		t.Errorf("Aliases=%v; want [ex demo-skill]", s.Aliases)
	}
	if !s.HasFM {
		t.Error("HasFM=false; want true")
	}
	if s.SourceFile != "/a/skills/example/SKILL.md" {
		t.Errorf("SourceFile=%q; want /a/skills/example/SKILL.md", s.SourceFile)
	}
}

// Frontmatter{} (nil Metadata, e.g. a SKILL.md with no --- block or a read error):
// BuildSkill MUST NOT panic and must yield zero metadata while still computing
// Dir/RelTag/SourceFile. (verified_facts §7 — the load-bearing nil-safety test.)
func TestBuildSkillNoFrontmatter(t *testing.T) {
	s := BuildSkill("/a/skills/plain", "plain", Frontmatter{})
	if s.HasFM {
		t.Error("HasFM=true; want false")
	}
	if s.Name != "" || s.Description != "" {
		t.Errorf("Name=%q Description=%q; want empty", s.Name, s.Description)
	}
	if s.Category != "" {
		t.Errorf("Category=%q; want empty", s.Category)
	}
	if len(s.Keywords) != 0 || len(s.Aliases) != 0 {
		t.Errorf("Keywords=%v Aliases=%v; want empty (len 0)", s.Keywords, s.Aliases)
	}
	if s.SourceFile != "/a/skills/plain/SKILL.md" {
		t.Errorf("SourceFile=%q; want /a/skills/plain/SKILL.md", s.SourceFile)
	}
}

// metadata present but keywords/category/aliases absent -> defaults.
func TestBuildSkillMetadataWithoutConventions(t *testing.T) {
	fm := Frontmatter{
		Name:        "x",
		Description: "y",
		HasFM:       true,
		Metadata:    map[string]any{"some-other-key": "whatever"},
	}
	s := BuildSkill("/a/skills/x", "x", fm)
	if len(s.Keywords) != 0 || len(s.Aliases) != 0 || s.Category != "" {
		t.Errorf("unexpected metadata: Keywords=%v Aliases=%v Category=%q; want empty",
			s.Keywords, s.Aliases, s.Category)
	}
	if !s.HasFM || s.Name != "x" || s.Description != "y" {
		t.Errorf("scalar passthrough wrong: HasFM=%v Name=%q Description=%q", s.HasFM, s.Name, s.Description)
	}
}

// SourceFile is derived from Dir via filepath.Join (cleans a trailing slash).
func TestBuildSkillSourceFile(t *testing.T) {
	cases := []struct{ dir, want string }{
		{"/a/skills/foo", "/a/skills/foo/SKILL.md"},
		{"/a/skills/writing/reddit", "/a/skills/writing/reddit/SKILL.md"},
		{"/a/skills/trailing/", "/a/skills/trailing/SKILL.md"}, // trailing slash cleaned by Join
	}
	for _, c := range cases {
		s := BuildSkill(c.dir, "x", Frontmatter{})
		if s.SourceFile != c.want {
			t.Errorf("dir=%q: SourceFile=%q; want %q", c.dir, s.SourceFile, c.want)
		}
	}
	// Direct contract: SourceFile == filepath.Join(dir, "SKILL.md").
	s := BuildSkill("/abs/x", "x", Frontmatter{})
	if s.SourceFile != filepath.Join("/abs/x", "SKILL.md") {
		t.Errorf("SourceFile=%q != filepath.Join=%q", s.SourceFile, filepath.Join("/abs/x", "SKILL.md"))
	}
}

// End-to-end: a real SKILL.md (PRD §11-shaped) parsed by S1's ParseFrontmatter,
// then built into a Skill. Proves the genuine []any -> []string path AND that the
// folded-scalar description is carried through verbatim (trailing \n retained).
func TestBuildSkillEndToEnd(t *testing.T) {
	path := writeSkill(t, "---\nname: example\ndescription: >\n  Reference example skill for skpp.\nmetadata:\n  keywords: [example, demo, skpp]\n  category: meta\n  aliases:\n    - ex\n    - demo\n---\n# body\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if !fm.HasFM {
		t.Fatal("HasFM=false; want true (valid --- block present)")
	}

	s := BuildSkill("/skills/example", "example", fm)
	if s.Name != "example" {
		t.Errorf("Name=%q; want example", s.Name)
	}
	if !strings.HasSuffix(s.Description, "Reference example skill for skpp.\n") {
		t.Errorf("Description=%q; want folded scalar ending with '...skpp.\\n' (S1 verbatim contract)", s.Description)
	}
	if !strEq(s.Keywords, []string{"example", "demo", "skpp"}) {
		t.Errorf("Keywords=%v; want [example demo skpp] (real []any path)", s.Keywords)
	}
	if s.Category != "meta" {
		t.Errorf("Category=%q; want meta", s.Category)
	}
	if !strEq(s.Aliases, []string{"ex", "demo"}) {
		t.Errorf("Aliases=%v; want [ex demo]", s.Aliases)
	}
	if !s.HasFM {
		t.Error("HasFM=false; want true")
	}
	if s.SourceFile != "/skills/example/SKILL.md" {
		t.Errorf("SourceFile=%q; want /skills/example/SKILL.md", s.SourceFile)
	}
}
