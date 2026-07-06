package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSkill writes content to a temp SKILL.md and returns its path. Each fixture
// lives in its own t.TempDir() so they never collide.
func writeSkill(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// --- ParseFrontmatter: happy paths ---

func TestParseFrontmatterFull(t *testing.T) {
	path := writeSkill(t, `---
name: my-skill
description: A short description.
license: MIT
compatibility: "Requires Python 3.11+"
metadata:
  keywords: [writing, reddit]
  category: writing
  aliases:
    - reddit-post
    - social-post
allowed-tools: shell exec
disable-model-invocation: true
---
# Body
`)
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	if !fm.HasFM {
		t.Error("HasFM=false; want true")
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name=%q; want my-skill", fm.Name)
	}
	if fm.Description != "A short description." {
		t.Errorf("Description=%q; want 'A short description.'", fm.Description)
	}
	if fm.License != "MIT" {
		t.Errorf("License=%q; want MIT", fm.License)
	}
	if fm.Compatibility != "Requires Python 3.11+" {
		t.Errorf("Compatibility=%q; want 'Requires Python 3.11+'", fm.Compatibility)
	}
	if fm.AllowedTools != "shell exec" {
		t.Errorf("AllowedTools=%q; want 'shell exec' (space-delimited string)", fm.AllowedTools)
	}
	if !fm.DisableModelInvocation {
		t.Error("DisableModelInvocation=false; want true")
	}
	if fm.Metadata == nil {
		t.Fatal("Metadata=nil; want populated map")
	}
	if got := fm.Metadata["category"]; got != "writing" {
		t.Errorf("Metadata[category]=%#v; want \"writing\"", got)
	}
	// metadata lists arrive as []any (== []interface{}), NOT []string (yaml.v3).
	kw, ok := fm.Metadata["keywords"].([]any)
	if !ok {
		t.Fatalf("Metadata[keywords] type=%T; want []any", fm.Metadata["keywords"])
	}
	if len(kw) != 2 || kw[0] != "writing" || kw[1] != "reddit" {
		t.Errorf("keywords=%#v; want [writing reddit]", kw)
	}
	if body != "# Body\n" {
		t.Errorf("body=%q; want \"# Body\\n\"", body)
	}
}

// §1: unknown frontmatter keys are silently ignored (lenient, no error).
func TestParseFrontmatterUnknownKeysIgnored(t *testing.T) {
	path := writeSkill(t, `---
name: x
description: y
future-field: whatever
another: [1, 2, 3]
---
`)
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unknown keys: err=%v; want nil (lenient)", err)
	}
	if !fm.HasFM {
		t.Error("HasFM=false; want true")
	}
	if fm.Name != "x" {
		t.Errorf("Name=%q; want x", fm.Name)
	}
}

// §3: folded scalar '>' KEEPS a trailing newline. Returned verbatim (no trim).
func TestParseFrontmatterFoldedScalarKeepsTrailingNewline(t *testing.T) {
	path := writeSkill(t, `---
description: >
  One line.
  Two line.
name: x
---
`)
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.HasSuffix(fm.Description, "\n") {
		t.Errorf("folded Description=%q; want trailing \\n (returned verbatim)", fm.Description)
	}
	if !strings.Contains(fm.Description, "One line. Two line.") {
		t.Errorf("folded Description=%q; want lines joined with a single space", fm.Description)
	}
}

// §4: quoted values are unquoted; spaces preserved.
func TestParseFrontmatterQuotedValues(t *testing.T) {
	path := writeSkill(t, "---\nname: \"my-skill\"\ncompatibility: \"Requires Python 3.11+\"\ndescription: \"d\"\n---\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name=%q; want my-skill (unquoted)", fm.Name)
	}
	if fm.Compatibility != "Requires Python 3.11+" {
		t.Errorf("Compatibility=%q; want spaces preserved", fm.Compatibility)
	}
}

// --- ParseFrontmatter: no-frontmatter cases (lenient, no error) ---

// No frontmatter block at all: HasFM=false, body=whole file, no error.
func TestParseFrontmatterNoBlock(t *testing.T) {
	content := "# just markdown\nno frontmatter here\n"
	path := writeSkill(t, content)
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("no-block: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false")
	}
	if fm.Name != "" || fm.Description != "" {
		t.Errorf("no-block: Name=%q Description=%q; want empty", fm.Name, fm.Description)
	}
	if body != content {
		t.Errorf("body=%q; want whole file %q", body, content)
	}
}

// §7: opening fence but no closing fence -> lenient, no frontmatter, no error.
func TestParseFrontmatterNoClosingFence(t *testing.T) {
	content := "---\nname: dangling\ndescription: no close\n"
	path := writeSkill(t, content)
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("no-close: err=%v; want nil (lenient)", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false (no closing fence -> lenient)")
	}
	if body != content {
		t.Errorf("body=%q; want whole file %q", body, content)
	}
}

// §9: empty file -> no panic, no frontmatter, body="".
func TestParseFrontmatterEmptyFile(t *testing.T) {
	path := writeSkill(t, "")
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("empty: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false")
	}
	if body != "" {
		t.Errorf("empty body=%q; want \"\"", body)
	}
}

// "---\n"-only file: opening fence, immediate EOF -> lenient, no frontmatter.
func TestParseFrontmatterOnlyFence(t *testing.T) {
	path := writeSkill(t, "---\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("only-fence: err=%v; want nil", err)
	}
	if fm.HasFM {
		t.Error("HasFM=true; want false (no closing fence -> lenient)")
	}
}

// --- ParseFrontmatter: encoding robustness ---

// §5: a leading UTF-8 BOM must not prevent fence detection.
func TestParseFrontmatterLeadingBOM(t *testing.T) {
	path := writeSkill(t, "\ufeff---\nname: bom-skill\ndescription: bommed\n---\nbody\n")
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("bom: err=%v", err)
	}
	if !fm.HasFM {
		t.Fatal("bom: HasFM=false; want true (BOM stripped before fence detection)")
	}
	if fm.Name != "bom-skill" {
		t.Errorf("bom Name=%q; want bom-skill", fm.Name)
	}
	if body != "body\n" {
		t.Errorf("bom body=%q; want \"body\\n\"", body)
	}
}

// §6: CRLF line endings -> fences detected via \r trim; body retains \r.
func TestParseFrontmatterCRLF(t *testing.T) {
	path := writeSkill(t, "---\r\nname: crlf-skill\r\ndescription: crlfd\r\n---\r\n# body\r\n")
	fm, body, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("crlf: err=%v", err)
	}
	if !fm.HasFM {
		t.Fatal("crlf: HasFM=false; want true (CRLF fences detected via \\r trim)")
	}
	if fm.Name != "crlf-skill" {
		t.Errorf("crlf Name=%q; want crlf-skill", fm.Name)
	}
	if body != "# body\r\n" {
		t.Errorf("crlf body=%q; want \"# body\\r\\n\" (\\r retained)", body)
	}
}

// --- ParseFrontmatter: error paths ---

// §8: malformed YAML between valid fences -> HARD error (propagated). Assert the
// error is present and HasFM is false; do NOT assert the yaml.v3 message wording
// (it is library-internal and may shift across versions).
func TestParseFrontmatterMalformedYAML(t *testing.T) {
	path := writeSkill(t, "---\nname: bad\nmetadata: [unbalanced\n---\nbody\n")
	fm, _, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("malformed: err=nil; want a yaml error (broken YAML is a HARD error)")
	}
	if fm.HasFM {
		t.Error("malformed: HasFM=true; want false (unmarshal failed)")
	}
}

// Read error: nonexistent file -> os.ReadFile error returned verbatim.
func TestParseFrontmatterMissingFile(t *testing.T) {
	fm, body, err := ParseFrontmatter(filepath.Join(t.TempDir(), "nope.md"))
	if err == nil {
		t.Fatal("missing file: err=nil; want os.ReadFile error")
	}
	if fm.HasFM || body != "" {
		t.Errorf("missing file: HasFM=%v body=%q; want false/empty", fm.HasFM, body)
	}
}

// --- Frontmatter type: the HasFM yaml:"-" guard ---

// A frontmatter key literally named "hasfm" must NOT set the HasFM field (the tag
// is "-", so yaml.v3 never touches it). Verified §12.
func TestHasFMNotMappedFromKey(t *testing.T) {
	path := writeSkill(t, "---\nname: x\ndescription: y\nhasfm: true\n---\n")
	fm, _, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// HasFM is true here because a valid frontmatter block WAS found — but the
	// "hasfm: true" KEY did not influence it. Re-assert the contract: a file with
	// NO frontmatter block and a stray "hasfm" line must still be HasFM=false.
	if !fm.HasFM {
		t.Error("HasFM=false; want true (a valid block was present)")
	}

	nofm := writeSkill(t, "hasfm: true\nname: stray\n")
	fm2, _, err2 := ParseFrontmatter(nofm)
	if err2 != nil {
		t.Fatalf("nofm: err=%v", err2)
	}
	if fm2.HasFM {
		t.Error("nofm HasFM=true; want false (no --- block; the stray 'hasfm:' line must not set it)")
	}
}
