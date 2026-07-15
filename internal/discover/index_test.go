package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// NOTE: this file is white-box `package discover`, so it shares scope with
// discover_test.go (writeSkill) and skill_test.go (strEq). Reuse them; do NOT
// redefine either (redeclaration is a build error). Index() is EXPORTED, so a
// black-box `package discover_test` would also work — we stay white-box to match
// discover_test.go / skill_test.go.

// makeTree builds a temp skills/ tree from a map[relTag]SKILL.md-content and
// returns its root. relTag uses '/' separators (cross-platform via FromSlash).
// A "" key writes SKILL.md directly in the root (the relTag="." edge case).
func makeTree(t *testing.T, layout map[string]string) string {
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

// Single top-level skill: full field round-trip + absolute Dir + existing SourceFile.
func TestIndexSingle(t *testing.T) {
	root := makeTree(t, map[string]string{
		"example": "---\nname: example\ndescription: demo\nmetadata:\n  keywords: [a, b]\n  category: meta\n  aliases: [ex]\n---\n# body\n",
	})
	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d; want 1 (%v)", len(got), got)
	}
	s := got[0]
	if s.RelTag != "example" {
		t.Errorf("RelTag=%q; want example", s.RelTag)
	}
	if s.Name != "example" {
		t.Errorf("Name=%q; want example", s.Name)
	}
	if !filepath.IsAbs(s.Dir) {
		t.Errorf("Dir=%q is not absolute (PRD §6.1/§13 absolute contract)", s.Dir)
	}
	if s.SourceFile != filepath.Join(s.Dir, "SKILL.md") {
		t.Errorf("SourceFile=%q; want %q", s.SourceFile, filepath.Join(s.Dir, "SKILL.md"))
	}
	if _, err := os.Stat(s.SourceFile); err != nil {
		t.Errorf("SourceFile does not exist: %v", err)
	}
	if !strEq(s.Keywords, []string{"a", "b"}) {
		t.Errorf("Keywords=%v; want [a b] (end-to-end []any -> []string)", s.Keywords)
	}
	if s.Category != "meta" {
		t.Errorf("Category=%q; want meta", s.Category)
	}
	if !strEq(s.Aliases, []string{"ex"}) {
		t.Errorf("Aliases=%v; want [ex]", s.Aliases)
	}
}

// Nested skill: relTag uses '/' separators (filepath.ToSlash), no backslash.
func TestIndexNestedRelTag(t *testing.T) {
	root := makeTree(t, map[string]string{
		"writing/reddit": "---\nname: reddit\ndescription: d\n---\nx\n",
	})
	got, _ := Index(root)
	if len(got) != 1 || got[0].RelTag != "writing/reddit" {
		t.Fatalf("got=%v; want one skill RelTag=writing/reddit (separator normalized to /)", got)
	}
	if strings.Contains(got[0].RelTag, "\\") {
		t.Errorf("RelTag contains a backslash: %q (must be /-normalized)", got[0].RelTag)
	}
}

// Returned slice is sorted by RelTag (lexicographic), not by walk visit order.
func TestIndexSortedByRelTag(t *testing.T) {
	root := makeTree(t, map[string]string{
		"zebra":      "---\nname: z\ndescription: d\n---\n",
		"apple":      "---\nname: a\ndescription: d\n---\n",
		"mango/fig":  "---\nname: f\ndescription: d\n---\n",
		"mango/beta": "---\nname: b\ndescription: d\n---\n",
	})
	got, _ := Index(root)
	var tags []string
	for _, s := range got {
		tags = append(tags, s.RelTag)
	}
	// Lexicographic by canonical tag; "mango/beta" < "mango/fig" < "zebra".
	want := []string{"apple", "mango/beta", "mango/fig", "zebra"}
	if !strEq(tags, want) {
		t.Errorf("order=%v; want %v (lexicographic by RelTag)", tags, want)
	}
}

// No-frontmatter SKILL.md is still resolved by directory (PRD §7.1): HasFM=false.
func TestIndexNoFrontmatterIncluded(t *testing.T) {
	root := makeTree(t, map[string]string{
		"plain": "# just markdown, no --- block\n",
	})
	got, _ := Index(root)
	if len(got) != 1 {
		t.Fatalf("len=%d; want 1 (no-frontmatter skill still resolved by dir, PRD §7.1)", len(got))
	}
	s := got[0]
	if s.HasFM {
		t.Error("HasFM=true; want false (no --- block)")
	}
	if s.Name != "" || s.Description != "" {
		t.Errorf("Name=%q Description=%q; want empty", s.Name, s.Description)
	}
	if s.RelTag != "plain" {
		t.Errorf("RelTag=%q; want plain", s.RelTag)
	}
}

// Malformed YAML does NOT abort the walk and is NOT propagated: the bad skill is
// included (HasFM=false) and the good sibling is kept. (verified_facts §8.)
func TestIndexMalformedYAMLNotAborted(t *testing.T) {
	root := makeTree(t, map[string]string{
		"good": "---\nname: good\ndescription: d\n---\n",
		"bad":  "---\nname: bad\nmetadata: [unbalanced\n---\nbody\n",
	})
	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v; want nil (malformed YAML must NOT abort the walk)", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d; want 2 (malformed skill still included, lenient)", len(got))
	}
	if got[0].RelTag != "bad" || got[1].RelTag != "good" {
		t.Errorf("order=%v; want [bad good]", got)
	}
	if got[0].HasFM {
		t.Error("bad: HasFM=true; want false (malformed YAML -> Frontmatter{} -> HasFM=false)")
	}
	if got[1].HasFM != true || got[1].Name != "good" {
		t.Errorf("good: HasFM=%v Name=%q; want true/good", got[1].HasFM, got[1].Name)
	}
}

// Stray files (README.md, *.bak) and subdirs without a SKILL.md are ignored.
func TestIndexIgnoresNonSkillMD(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "real"), 0o755)
	os.WriteFile(filepath.Join(root, "real/SKILL.md"), []byte("---\nname: real\ndescription: d\n---\n"), 0o644)
	// Distractions that must NOT be treated as skills.
	os.WriteFile(filepath.Join(root, "README.md"), []byte("# hi"), 0o644)
	os.MkdirAll(filepath.Join(root, "notes"), 0o755)
	os.WriteFile(filepath.Join(root, "notes/draft.txt"), []byte("draft"), 0o644)
	os.WriteFile(filepath.Join(root, "SKILL.md.bak"), []byte("bak"), 0o644)

	got, _ := Index(root)
	if len(got) != 1 || got[0].RelTag != "real" {
		t.Fatalf("got=%v; want exactly one skill 'real' (stray files/subdirs ignored)", got)
	}
}

// Empty skills dir (exists, no SKILL.md) -> nil/empty slice, nil error.
func TestIndexEmptyDir(t *testing.T) {
	root := t.TempDir() // exists, empty
	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len=%d; want 0 (empty tree -> no skills)", len(got))
	}
}

// Missing root -> error (the Stat guard; without it this returns (nil,nil)).
func TestIndexMissingRoot(t *testing.T) {
	_, err := Index(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("err=nil; want an error (missing root must propagate after the Stat guard)")
	}
}

// Root that is a regular file -> error ("not a directory").
func TestIndexRootIsFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "notadir")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := Index(f.Name()); err == nil {
		t.Fatal("err=nil; want an error (root must be a directory)")
	}
}

// Nested skills at multiple levels: writing AND writing/reddit are BOTH indexed.
func TestIndexNestedBothLevels(t *testing.T) {
	root := makeTree(t, map[string]string{
		"writing":        "---\nname: writing\ndescription: d\n---\n",
		"writing/reddit": "---\nname: reddit\ndescription: d\n---\n",
	})
	got, _ := Index(root)
	if len(got) != 2 {
		t.Fatalf("len=%d; want 2 (parent is a skill AND has a nested subskill)", len(got))
	}
	if got[0].RelTag != "writing" || got[1].RelTag != "writing/reddit" {
		t.Errorf("got=%v; want [writing writing/reddit]", got)
	}
}

// Edge case: a SKILL.md directly in the skills root -> relTag == "."
// (filepath.Rel(root, root)). Included for spec-compliance; unusual in practice.
func TestIndexRootLevelSkillMD(t *testing.T) {
	root := makeTree(t, map[string]string{"": "---\nname: root\ndescription: d\n---\n"})
	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d; want 1 (root-level SKILL.md is still a skill)", len(got))
	}
	if got[0].RelTag != "." {
		t.Errorf("RelTag=%q; want '.' (filepath.Rel(root,root) edge case)", got[0].RelTag)
	}
}

// Defensive: a RELATIVE skillsDir still yields ABSOLUTE Skill.Dir (filepath.Abs).
// Protects the PRD §6.1/§13 absolute-output contract. t.Chdir (Go 1.24+) scopes cwd.
func TestIndexRelativeInputDirStillAbsolute(t *testing.T) {
	absRoot := makeTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\n",
	})
	parent := filepath.Dir(absRoot)
	t.Chdir(parent)
	rel, err := filepath.Rel(parent, absRoot)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Index(rel)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d; want 1", len(got))
	}
	if !filepath.IsAbs(got[0].Dir) {
		t.Errorf("Dir=%q is RELATIVE; want absolute (relative input must still abs-ify)", got[0].Dir)
	}
	if got[0].RelTag != "example" {
		t.Errorf("RelTag=%q; want example", got[0].RelTag)
	}
}

// --- symlink-following (PRD §7.1: the walk now follows linked directories) ---

// mustSymlink creates target->link and skips the test if the platform/filesystem
// can't create symlinks (mirrors skillsdir_test.go's approach).
func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}
}

// A skill dir living OUTSIDE the skills root, linked in by a symlink, is
// discovered. RelTag/Dir/SourceFile are reported under the LINK NAME (what the
// user wrote), not the resolved target, and the SKILL.md is readable through it.
func TestIndexFollowsSymlinkedSkillDir(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "SKILL.md"),
		[]byte("---\nname: pooled\ndescription: d\n---\nbody\n"), 0o644)
	link := filepath.Join(root, "agent-browser-pool")
	mustSymlink(t, target, link)

	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d; want 1 (symlinked skill dir must be discovered), got=%v", len(got), got)
	}
	s := got[0]
	if s.RelTag != "agent-browser-pool" {
		t.Errorf("RelTag=%q; want agent-browser-pool (reported under the link name)", s.RelTag)
	}
	wantDir := filepath.Join(root, "agent-browser-pool")
	if s.Dir != wantDir {
		t.Errorf("Dir=%q; want %q (display path THROUGH the symlink)", s.Dir, wantDir)
	}
	if s.SourceFile != filepath.Join(wantDir, "SKILL.md") {
		t.Errorf("SourceFile=%q; want %q", s.SourceFile, filepath.Join(wantDir, "SKILL.md"))
	}
	if _, err := os.Stat(s.SourceFile); err != nil {
		t.Errorf("SourceFile not readable through symlink: %v", err)
	}
	if s.Name != "pooled" || !s.HasFM {
		t.Errorf("frontmatter not parsed through symlink: Name=%q HasFM=%v", s.Name, s.HasFM)
	}
}

// A linked-in directory that itself contains NESTED skills is fully walked, with
// relTags reported under the link prefix (vendored/...), not the resolved target.
func TestIndexSymlinkedNestedTree(t *testing.T) {
	target := t.TempDir()
	for relTag, body := range map[string]string{
		"writing":        "---\nname: writing\ndescription: d\n---\n",
		"writing/reddit": "---\nname: reddit\ndescription: d\n---\n",
	} {
		dir := filepath.Join(target, filepath.FromSlash(relTag))
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644)
	}
	root := t.TempDir()
	mustSymlink(t, target, filepath.Join(root, "vendored"))

	got, _ := Index(root)
	var tags []string
	for _, s := range got {
		tags = append(tags, s.RelTag)
	}
	want := []string{"vendored/writing", "vendored/writing/reddit"}
	if !strEq(tags, want) {
		t.Fatalf("RelTags=%v; want %v (nested skills reported under the link prefix)", tags, want)
	}
}

// Symlink cycles MUST terminate. Two cycles are planted: a link back at root, and
// a link from a subdir back at that subdir. A walk without a cycle guard would
// never return. The result is exactly the two real skills (no duplicates).
func TestIndexSymlinkCycleBroken(t *testing.T) {
	root := makeTree(t, map[string]string{
		"real": "---\nname: real\ndescription: d\n---\n",
	})
	mustSymlink(t, root, filepath.Join(root, "loop")) // -> ancestor (root)

	sub := filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "SKILL.md"), []byte("---\nname: sub\ndescription: d\n---\n"), 0o644)
	mustSymlink(t, sub, filepath.Join(sub, "inner")) // -> ancestor (sub)

	// Guard against a regression that would hang the suite: if the cycle isn't
	// broken, fail fast instead of waiting for `go test`'s 10m timeout.
	done := make(chan struct{})
	go func() { Index(root); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Index did not return within 5s; symlink cycle not broken (infinite loop)")
	}

	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	var tags []string
	for _, s := range got {
		tags = append(tags, s.RelTag)
	}
	if !strEq(tags, []string{"real", "sub"}) {
		t.Errorf("RelTags=%v; want [real sub] (cycle links must not duplicate or hang)", tags)
	}
}

// A broken (dangling) symlink is skipped silently — not an error, not a skill.
func TestIndexBrokenSymlinkSkipped(t *testing.T) {
	root := makeTree(t, map[string]string{
		"good": "---\nname: good\ndescription: d\n---\n",
	})
	mustSymlink(t, filepath.Join(t.TempDir(), "nope"), filepath.Join(root, "dangling"))

	got, err := Index(root)
	if err != nil {
		t.Fatalf("err=%v; want nil (broken symlink must be skipped, not an error)", err)
	}
	if len(got) != 1 || got[0].RelTag != "good" {
		t.Fatalf("got=%v; want exactly [good] (broken symlink ignored)", got)
	}
}

// A skills dir that is ITSELF a symlink to a directory still works; RelTag is
// relative to the link path the user passed (not the resolved target).
func TestIndexSymlinkedRoot(t *testing.T) {
	real := makeTree(t, map[string]string{
		"example": "---\nname: example\ndescription: d\n---\n",
	})
	link := filepath.Join(filepath.Dir(real), "skills-link")
	mustSymlink(t, real, link)

	got, err := Index(link)
	if err != nil {
		t.Fatalf("err=%v; want nil (a skills dir that is itself a symlink must work)", err)
	}
	if len(got) != 1 || got[0].RelTag != "example" {
		t.Fatalf("got=%v; want [example] through a symlinked root", got)
	}
	if !strings.HasPrefix(got[0].Dir, link) {
		t.Errorf("Dir=%q; want it under the symlink root %q", got[0].Dir, link)
	}
}
