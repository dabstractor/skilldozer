// skill.go defines the Skill type and the metadata-extraction constructor that
// turns parsed frontmatter (S1's Frontmatter) into the typed records the rest of
// skpp consumes (PRD §7.1, §7.3). This is the P1.M2.T4.S2 deliverable: discover.go
// (S1) owns Frontmatter + ParseFrontmatter; skill.go (S2) owns Skill + the
// []any->[]string normalization; the Index() walk that ties them together is T5.
package discover

import "path/filepath"

// Skill is a resolved on-disk skill (PRD §7.1). Index() (T5) returns a []Skill;
// resolve.Resolve (T7) matches tags against it; ui.Print* (T6) renders it.
//
// It is BUILT by BuildSkill, never unmarshaled, so it carries NO yaml tags
// (unlike S1's Frontmatter, which is the unmarshal target).
//
// Field semantics:
//   - Dir:         absolute path of the skill directory (e.g. /home/u/skills/foo).
//   - RelTag:      the skill dir path relative to the skills dir, with OS
//     separators normalized to '/' (e.g. "writing/reddit"). This is
//     the CANONICAL tag (PRD §7.2 step 1). T5 computes it; S2 carries it.
//   - Name:        frontmatter `name` ("" if the block or the field is absent).
//   - Description: frontmatter `description` ("" if absent). Copied VERBATIM from
//     Frontmatter, including a folded-scalar trailing newline (S1
//     contract); T10's 1024-char check trims if it wants visible length.
//   - Keywords:    metadata.keywords as []string (nil if absent/non-list).
//   - Category:    metadata.category as string ("" if absent).
//   - Aliases:     metadata.aliases as []string (nil if absent/non-list).
//   - HasFM:       false if SKILL.md had no --- frontmatter block (from S1).
//   - SourceFile:  absolute path to SKILL.md (== filepath.Join(Dir, "SKILL.md")).
//
// yaml.v3 delivers metadata lists as []interface{} (== []any), NEVER []string;
// toStringSlice normalizes them so the typed fields are convenient for
// resolve/search. An ABSENT field yields a nil slice; a PRESENT-but-empty list
// ([]any{}) yields a non-nil empty slice. Both have len 0 -> callers MUST test
// with len(), not a nil check.
type Skill struct {
	Dir         string
	RelTag      string
	Name        string
	Description string
	Keywords    []string
	Category    string
	Aliases     []string
	HasFM       bool
	SourceFile  string
}

// toStringSlice normalizes a frontmatter metadata value into []string.
//
// yaml.v3 unmarshals YAML lists into []interface{} (== []any), NEVER []string,
// regardless of element type. This helper asserts []any -> []string so the typed
// Skill fields are convenient for resolve/search. Behavior (verified):
//   - nil           -> nil
//   - []any         -> []string, with NON-STRING elements silently skipped
//     (lenient: a stray number in `keywords:` is dropped, matching
//     the "ignore what doesn't fit" leniency of PRD §7.3)
//   - []string      -> returned as-is (defensive; yaml.v3 never produces this)
//   - single string -> []string{s} (lenient: `keywords: writing` -> ["writing"])
//   - anything else -> nil
//
// A present-but-empty list ([]any{}) yields an empty non-nil []string; an absent
// field yields nil. Both have len 0 -> callers must use len(), not a nil check.
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	case []string:
		return s
	case string:
		return []string{s}
	default:
		return nil
	}
}

// BuildSkill assembles a Skill from walk-derived location info and the parsed
// frontmatter. It performs the PRD §7.1/§7.3 metadata extraction and is the
// boundary between S1 (Frontmatter/ParseFrontmatter) and T5 (Index walk).
//
// T5 calls it once per discovered skill dir:
//
//	fm, _, err := ParseFrontmatter(filepath.Join(dir, "SKILL.md"))
//	// T5 decides how to surface `err` (malformed YAML) to `check` (M4);
//	// BuildSkill itself never errors — it works on any Frontmatter, including
//	// Frontmatter{} (HasFM=false) from a no-frontmatter or read-error skill.
//	s := BuildSkill(dir, relTag, fm)
//
// It is TOTAL: no error return, no panic — even when fm.Metadata is nil (no
// frontmatter block). Reading a missing key from a nil map returns the zero value
// (nil), and the comma-ok type assertion on nil yields ("", false). So a
// no-frontmatter skill gets a Skill with zero metadata + HasFM=false that T5 can
// still resolve by directory. Verified empirically.
//
// category uses the comma-ok assertion deliberately: a BARE fm.Metadata["category"].(string)
// would PANIC on a nil/absent value. SourceFile is derived from Dir via
// filepath.Join (== Dir + "/SKILL.md"); T5 does not pass or compute it.
func BuildSkill(dir, relTag string, fm Frontmatter) Skill {
	category, _ := fm.Metadata["category"].(string) // nil-map read is safe; comma-ok -> "",false
	return Skill{
		Dir:         dir,
		RelTag:      relTag,
		Name:        fm.Name,
		Description: fm.Description,
		Keywords:    toStringSlice(fm.Metadata["keywords"]),
		Category:    category,
		Aliases:     toStringSlice(fm.Metadata["aliases"]),
		HasFM:       fm.HasFM,
		SourceFile:  filepath.Join(dir, "SKILL.md"),
	}
}
