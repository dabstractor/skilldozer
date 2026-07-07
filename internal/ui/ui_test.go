package ui

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/dabstractor/skpp/internal/discover"
)

// mk builds one discover.Skill for table tests. fm controls HasFM (and thus the
// missing-description rule). Kept tiny so test rows stay readable.
func mk(tag, name, desc string, fm bool) discover.Skill {
	return discover.Skill{RelTag: tag, Name: name, Description: desc, HasFM: fm}
}

// colOf returns the column (0-based) of the first occurrence of substr in out,
// measured from the previous newline. Used to assert column alignment.
func colOf(out, substr string) int {
	idx := strings.Index(out, substr)
	if idx < 0 {
		return -1
	}
	return idx - (strings.LastIndex(out[:idx], "\n") + 1)
}

// runeCol returns the RUNE column (display column under the width-1-rune model)
// of substr's first occurrence within its line in out. Unlike colOf (byte offset),
// this counts runes so it reflects VISUAL alignment for multi-byte cells: byte
// padding yields uniform byte widths even when the display is misaligned, so a
// byte check is blind to the very bug displayWidth fixes (verified_facts §4).
func runeCol(out, substr string) int {
	idx := strings.Index(out, substr)
	if idx < 0 {
		return -1
	}
	lineStart := strings.LastIndex(out[:idx], "\n") + 1
	return utf8.RuneCountInString(out[lineStart:idx])
}

func TestPrintListEmptyPrintsNothing(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, nil, false)
	if buf.Len() != 0 {
		t.Errorf("empty input printed %q; want nothing", buf.String())
	}
}

func TestPrintListSingleNoColor(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("example", "example", "A demo skill.", true)}, false)
	out := buf.String()
	for _, want := range []string{"TAG", "NAME", "DESCRIPTION", "example", "A demo skill."} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("no-color output must not contain ANSI escapes:\n%s", out)
	}
	// Header precedes the data row.
	if h, d := strings.Index(out, "DESCRIPTION"), strings.Index(out, "A demo skill."); h < 0 || d < 0 || h > d {
		t.Errorf("header should precede data:\n%s", out)
	}
}

func TestPrintListColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("example", "example", "d", true)}, true)
	out := buf.String()
	for _, want := range []string{ansiBold, ansiCyan, ansiReset} {
		if !strings.Contains(out, want) {
			t.Errorf("color output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintListNoColorHasNoANSI(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("example", "example", "d", true)}, false)
	if strings.Contains(buf.String(), "\x1b") {
		t.Errorf("no-color output contains escapes:\n%s", buf.String())
	}
}

func TestPrintListMissingNameShowsNone(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{mk("plain", "", "d", true)}, false)
	if !strings.Contains(buf.String(), "(none)") {
		t.Errorf("empty name should render (none):\n%s", buf.String())
	}
}

func TestPrintListEmptyDescriptionShowsMissing(t *testing.T) {
	var buf bytes.Buffer
	// HasFM true but description empty -> "(missing)".
	PrintList(&buf, []discover.Skill{mk("a", "a", "", true)}, false)
	if !strings.Contains(buf.String(), "(missing)") {
		t.Errorf("empty description should render (missing):\n%s", buf.String())
	}
}

func TestPrintListNoFrontmatterShowsMissing(t *testing.T) {
	var buf bytes.Buffer
	// HasFM false -> "(missing)" regardless of description.
	PrintList(&buf, []discover.Skill{mk("b", "b", "", false)}, false)
	if !strings.Contains(buf.String(), "(missing)") {
		t.Errorf("no-frontmatter skill should render (missing):\n%s", buf.String())
	}
}

func TestPrintListTrimsFoldedScalarNewline(t *testing.T) {
	var buf bytes.Buffer
	// A folded-scalar description carries a trailing newline (discover.go contract).
	PrintList(&buf, []discover.Skill{mk("x", "x", "has trailing newline\n", true)}, false)
	out := buf.String()
	if !strings.Contains(out, "has trailing newline") {
		t.Errorf("description text missing:\n%s", out)
	}
	if strings.Contains(out, "\n\n") {
		t.Errorf("output has a blank line (folded newline not trimmed):\n%s", out)
	}
}

func TestPrintListWrapsLongDescription(t *testing.T) {
	var buf bytes.Buffer
	long := "Reference example skill for skpp. Demonstrates the required frontmatter and how skpp resolves a tag to an absolute path. Safe to delete once you add real skills."
	PrintList(&buf, []discover.Skill{mk("example", "example", long, true)}, false)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// header + >=2 wrapped data lines.
	if len(lines) < 3 {
		t.Fatalf("expected wrapped multi-line output, got %d lines:\n%s", len(lines), buf.String())
	}
	// Every wrapped line fits within descWrapWidth (no line overruns the column).
	descCol := colOf(buf.String(), "DESCRIPTION")
	for _, ln := range lines[1:] {
		if len(ln)-descCol > descWrapWidth {
			t.Errorf("wrapped line exceeds %d cols (descCol=%d):\n%q", descWrapWidth, descCol, ln)
		}
	}
	// All words survived (joining lines with spaces reconstructs the word stream).
	joined := strings.Join(lines, " ")
	for _, want := range []string{"Reference", "frontmatter", "real", "skills."} {
		if !strings.Contains(joined, want) {
			t.Errorf("wrapped output lost word %q:\n%s", want, joined)
		}
	}
}

func TestPrintListPreservesInputOrder(t *testing.T) {
	var buf bytes.Buffer
	// Input is zebra then apple; ui must NOT re-sort (discover.Index already did).
	PrintList(&buf, []discover.Skill{
		mk("zebra", "zebra", "z", true),
		mk("apple", "apple", "a", true),
	}, false)
	out := buf.String()
	// "zebra" appears once in the header? No — header is TAG/NAME/DESCRIPTION. The
	// tag value "zebra" first occurs in the zebra data row, which must precede apple.
	zi := strings.Index(out, "zebra")
	ai := strings.Index(out, "apple")
	if zi < 0 || ai < 0 || zi > ai {
		t.Errorf("expected zebra row before apple row (input order):\n%s", out)
	}
}

func TestPrintListColumnsAlignedAcrossRows(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{
		mk("a", "alpha", "short", true),
		mk("writing/reddit", "reddit-helper", "longer desc here", true),
	}, false)
	out := buf.String()
	descCol := colOf(out, "DESCRIPTION")
	if descCol < 0 {
		t.Fatalf("no DESCRIPTION header:\n%s", out)
	}
	// The longest tag ("writing/reddit") sets the column; "a"/"alpha" are padded so
	// every description starts at the same column under DESCRIPTION.
	for _, want := range []string{"short", "longer"} {
		if c := colOf(out, want); c != descCol {
			t.Errorf("desc %q starts at col %d; want %d (aligned under DESCRIPTION):\n%s", want, c, descCol, out)
		}
	}
	// The continuation-less first row's NAME is aligned under the NAME header.
	nameCol := colOf(out, "NAME")
	if c := colOf(out, "alpha"); c != nameCol {
		t.Errorf("'alpha' at col %d; want %d:\n%s", c, nameCol, out)
	}
}

// TestDisplayWidth: rune count is the display-width approximation (not byte len).
func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"café", 4},  // 5 bytes, 4 runes
		{"—", 1},     // 3 bytes, 1 rune
		{"a—b", 3},   // 5 bytes, 3 runes
		{"ascii", 5}, // byte == rune for ASCII
		{"", 0},
	}
	for _, c := range cases {
		if got := displayWidth(c.in); got != c.want {
			t.Errorf("displayWidth(%q)=%d; want %d (bytes=%d)", c.in, got, c.want, len(c.in))
		}
	}
}

// TestPadRightMultibyte: padding is measured in RUNES, so a multi-byte string is
// padded by (n - runeCount) spaces. With the byte-length bug, padRight("café",5)
// returns "café" (0 spaces) because len("café")==5>=5.
func TestPadRightMultibyte(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"café", 5, "café "},  // 1 space (5 - 4 runes)
		{"éé", 4, "éé  "},     // 2 spaces (4 - 2 runes)
		{"ascii", 3, "ascii"}, // already wider -> no truncation
		{"", 3, "   "},        // empty -> all padding
	}
	for _, c := range cases {
		if got := padRight(c.s, c.n); got != c.want {
			t.Errorf("padRight(%q,%d)=%q; want %q", c.s, c.n, got, c.want)
		}
	}
}

// TestWrapWordsMultibyte: wrapping measures RUNE width, so "café bar" fits in 8
// columns (4+1+3=8). With the byte-length bug, len("café")==5 makes 5+1+3=9>8 and
// it wrongly breaks into two lines.
func TestWrapWordsMultibyte(t *testing.T) {
	lines := wrapWords("café bar", 8)
	if len(lines) != 1 || lines[0] != "café bar" {
		t.Errorf("wrapWords(\"café bar\",8)=%v; want [\"café bar\"] (1 line, rune width)", lines)
	}
}

// TestPrintListColumnsAlignedForMultibyte: a multi-byte TAG (café: 4 runes/5
// bytes) and a multi-byte DESCRIPTION (—) must NOT shift the row's columns. Every
// description starts at the same DISPLAY column as the DESCRIPTION header.
//
// NOTE: this uses runeCol (rune offset), NOT the existing colOf (byte offset) —
// byte offsets are uniform under byte-padding (blind to the bug) and actually
// DIFFER under rune-padding (a rune-padded café cell is 6 bytes vs ascii's 5), so
// a byte check would pass with the bug and fail after the fix. See verified_facts §4.
func TestPrintListColumnsAlignedForMultibyte(t *testing.T) {
	var buf bytes.Buffer
	PrintList(&buf, []discover.Skill{
		mk("café", "cafe-name", "café — skill", true),
		mk("ascii", "ascii-name", "ascii skill", true),
	}, false)
	out := buf.String()

	descCol := runeCol(out, "DESCRIPTION")
	if descCol < 0 {
		t.Fatalf("no DESCRIPTION header:\n%s", out)
	}
	// Every description starts at the same DISPLAY column as the header. With the
	// byte-width bug, the café row's description lands one column early.
	for _, want := range []string{"café — skill", "ascii skill"} {
		if c := runeCol(out, want); c != descCol {
			t.Errorf("desc %q at display col %d; want %d (aligned under DESCRIPTION):\n%s", want, c, descCol, out)
		}
	}
	// The NAME column is likewise aligned (and the multi-byte tag did not shift it).
	nameCol := runeCol(out, "NAME")
	if c := runeCol(out, "cafe-name"); c != nameCol {
		t.Errorf("'cafe-name' at display col %d; want %d:\n%s", c, nameCol, out)
	}
}
