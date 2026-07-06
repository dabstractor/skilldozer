// Package discover scans the on-disk skills/ tree and parses SKILL.md frontmatter.
//
// This file (P1.M2.T4.S1) implements ONLY the frontmatter data model and the
// ParseFrontmatter parser. The Index() walk (T5), the Skill struct + metadata
// extraction (S2), and the toStringSlice helper are LATER subtasks — do not add
// them here.
//
// ParseFrontmatter implements PRD §7.3: it extracts the YAML block between the
// first two lines that are exactly "---" (handling a leading UTF-8 BOM and CRLF
// line endings), then unmarshals it into Frontmatter with gopkg.in/yaml.v3.
// Parsing is LENIENT in the PRD §7.3 sense: unknown frontmatter keys are silently
// ignored (yaml.v3's default). It is NOT lenient about syntactically broken YAML
// between valid fences — that is returned as an error so `check` (M4) can report
// it. A SKILL.md with no frontmatter block returns Frontmatter{HasFM:false} and no
// error (the skill still resolves by directory in T5).
package discover

import (
	"bytes"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// utf8BOM is the 3-byte UTF-8 byte-order mark (U+FEFF). Some editors prepend it to
// SKILL.md; it must be stripped before fence detection, otherwise the opening
// "---" reads as "\ufeff---" and frontmatter is silently missed. bytes.TrimPrefix
// is a no-op when the BOM is absent, so this is safe for BOM-free files.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Frontmatter is the parsed SKILL.md frontmatter (PRD §7.3, Agent Skills spec).
//
// It is the unmarshal target for the YAML block between the "---" fences. Unknown
// keys are ignored by yaml.v3's default (lenient) decoder, matching pi's behavior.
// The skpp conventions (keywords/category/aliases) live inside the standard,
// spec-compliant Metadata map; S2 extracts them into typed fields on Skill.
//
// Field types follow the Agent Skills spec:
//   - name, description: required strings (empty here means "absent in source").
//   - license, compatibility: optional scalars.
//   - allowed-tools: a SPACE-DELIMITED string per the spec (NOT a YAML list).
//   - disable-model-invocation: a boolean flag.
//
// HasFM is NOT a frontmatter key; it records whether a "---"-delimited block was
// present at all. The yaml:"-" tag stops yaml.v3 from reading/writing a key named
// "hasfm". Its zero value is false, so Frontmatter{} already means "no frontmatter".
type Frontmatter struct {
	Name                   string         `yaml:"name"`
	Description            string         `yaml:"description"`
	License                string         `yaml:"license,omitempty"`
	Compatibility          string         `yaml:"compatibility,omitempty"`
	Metadata               map[string]any `yaml:"metadata,omitempty"`
	AllowedTools           string         `yaml:"allowed-tools,omitempty"`
	DisableModelInvocation bool           `yaml:"disable-model-invocation,omitempty"`
	HasFM                  bool           `yaml:"-"`
}

// ParseFrontmatter reads the SKILL.md at path and returns its parsed frontmatter
// plus the markdown body (the non-frontmatter text). It implements PRD §7.3.
//
// Behavior (verified against yaml.v3 v3.0.1 on go1.26.4):
//
//   - No "---" block (the file does not start with a "---" line): returns
//     Frontmatter{HasFM:false}, body = the WHOLE file content, nil error. A skill
//     with no frontmatter still resolves by directory later (T5); `check` (M4)
//     flags the missing block and --list shows description as "(missing)".
//   - Opening "---" present but no closing "---": treated as NO frontmatter
//     (lenient). Same return as above.
//   - Valid fences with syntactically broken YAML between them: the yaml.v3 error
//     is returned (fm.HasFM==false). "Lenient" means ignore unknown KEYS, NOT
//     tolerate corrupt YAML.
//   - A leading UTF-8 BOM is stripped before fence detection.
//   - CRLF line endings: a trailing "\r" is ignored for the "---" comparison only
//     (the body retains its original bytes).
//
// body is always the non-frontmatter portion of the file: the whole file when
// there is no frontmatter, or everything after the closing "---" line otherwise
// (including on the malformed-YAML error path). ParseFrontmatter returns values
// VERBATIM — it does not trim the folded-scalar trailing newline from description
// (T10's length check trims if it wants the visible length).
func ParseFrontmatter(path string) (fm Frontmatter, body string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, "", err
	}
	data = bytes.TrimPrefix(data, utf8BOM)

	lines := strings.Split(string(data), "\n")
	// Must start with a line that is exactly "---" (modulo a trailing CRLF \r).
	if strings.TrimRight(lines[0], "\r") != "---" {
		return Frontmatter{}, string(data), nil // no frontmatter block
	}

	// Find the next line that is exactly "---" (the closing fence).
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		// Opening fence but no closing fence: lenient -> treat as no frontmatter.
		return Frontmatter{}, string(data), nil
	}

	yamlBlock := strings.Join(lines[1:closeIdx], "\n")
	body = strings.Join(lines[closeIdx+1:], "\n")

	var f Frontmatter
	if uerr := yaml.Unmarshal([]byte(yamlBlock), &f); uerr != nil {
		// Syntactically broken YAML between valid fences is a HARD error. Return
		// the post-fence body (useful for `check` diagnostics) plus the error.
		return Frontmatter{}, body, uerr
	}
	f.HasFM = true
	return f, body, nil
}
