package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// writeConfig writes content to a temp config.yaml and returns its path. Each
// fixture lives in its own t.TempDir() so they never collide. Mirrors the
// writeSkill helper in internal/discover/discover_test.go.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// --- Save/Load round trip (contract-required) ---

// TestSaveLoadRoundTrip locks contract OUTPUT §4 "round-trip Save->Load equality":
// a realistic Store value survives a Save then a Load unchanged.
func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	want := "/home/u/skills"
	if err := Save(path, File{Store: want}); err != nil {
		t.Fatalf("Save: err=%v; want nil", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: err=%v; want nil", err)
	}
	if got.Store != want {
		t.Errorf("round-trip Store=%q; want %q", got.Store, want)
	}
}

// TestLoadIgnoresUnknownKeys locks contract OUTPUT §4 "unknown keys ignored":
// extra keys (version, colors) are silently dropped by the lenient decoder and
// Store is still populated, with no error.
func TestLoadIgnoresUnknownKeys(t *testing.T) {
	path := writeConfig(t, "store: /abs\nversion: 3\ncolors: red\n")
	f, err := Load(path)
	if err != nil {
		t.Fatalf("unknown keys: err=%v; want nil (lenient)", err)
	}
	if f.Store != "/abs" {
		t.Errorf("Store=%q; want /abs (unknown keys must not block it)", f.Store)
	}
}

// TestLoadMissingFileIsErrNotExist locks contract OUTPUT §4 "fs.ErrNotExist
// returned (not masked) when file absent": a missing path returns an error that
// satisfies errors.Is(err, fs.ErrNotExist), so findConfig can fall through.
func TestLoadMissingFileIsErrNotExist(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("missing file: err=nil; want an os.ReadFile error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: err=%v; want an error satisfying errors.Is(fs.ErrNotExist)", err)
	}
}

// --- Hard on-disk format claim ---

// TestSaveWritesExactFormat locks the Marshal determinism: a non-empty Store is
// written as exactly "store: <value>\n" (struct-field order, no sorting, no BOM,
// no trailing "..."). Read back the raw bytes to verify — do not go through Load.
func TestSaveWritesExactFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Save(path, File{Store: "/x"}); err != nil {
		t.Fatalf("Save: err=%v; want nil", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: err=%v; want nil", err)
	}
	if string(out) != "store: /x\n" {
		t.Errorf("on-disk bytes=%q; want \"store: /x\\n\"", string(out))
	}
}

// --- Broken YAML is a hard error (not lenient) ---

// TestLoadMalformedYAMLIsHardError verifies "lenient" means ignore unknown KEYS,
// not tolerate corrupt YAML. A syntactically broken file returns a non-nil error.
// Assert only err != nil — the yaml.v3 message wording is library-internal.
func TestLoadMalformedYAMLIsHardError(t *testing.T) {
	path := writeConfig(t, "store: [unbalanced\n")
	_, err := Load(path)
	if err == nil {
		t.Fatal("malformed YAML: err=nil; want a yaml unmarshal error (broken YAML is a HARD error)")
	}
}

// --- Parent directory creation ---

// TestSaveCreatesParentDir verifies Save runs os.MkdirAll on the parent dir, so a
// nested config path whose directories do not yet exist (the common first-run
// case under $XDG_CONFIG_HOME/skilldozer/) is created and the file lands there.
func TestSaveCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "config.yaml")
	if err := Save(path, File{Store: "/store"}); err != nil {
		t.Fatalf("Save with missing parent dirs: err=%v; want nil (MkdirAll should create them)", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created at %s after Save: %v", path, err)
	}
}

// --- Sanity: the struct tag is exactly what the contract pins ---

// TestFileStoreTagIsExact guards against an accidental tag edit (e.g. dropping
// omitempty, or renaming the key). Marshal an empty File and a populated one and
// confirm the key name and omitempty behavior empirically.
func TestFileStoreTagIsExact(t *testing.T) {
	empty, err := yaml.Marshal(&File{})
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if string(empty) != "{}\n" {
		t.Errorf("empty File marshaled to %q; want \"{}\\n\" (omitempty elides store)", string(empty))
	}
	set, err := yaml.Marshal(&File{Store: "/s"})
	if err != nil {
		t.Fatalf("marshal set: %v", err)
	}
	if string(set) != "store: /s\n" {
		t.Errorf("File{Store:/s} marshaled to %q; want \"store: /s\\n\"", string(set))
	}
}
