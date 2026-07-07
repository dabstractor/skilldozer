package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// ---------------------------------------------------------------------------
// Path / DefaultStore tests (P1.M1.T1.S2).
//
// Every test below mutates process env via t.Setenv, so NONE may call
// t.Parallel (mirrors internal/skillsdir/skillsdir_test.go). t.Setenv cannot
// unset, but Path/DefaultStore use os.Getenv + `!=""`, so t.Setenv(var, "")
// correctly simulates "unset" for these two functions (empty == unset).
// Path/DefaultStore read ONLY env vars (no filesystem), so no temp FILES are
// needed — t.TempDir() is used only to obtain controlled ABSOLUTE env values.
// ---------------------------------------------------------------------------

// Path: a non-empty SKILLDOZER_CONFIG override is returned filepath.Clean'd,
// honored over $XDG_CONFIG_HOME. Proves the override branch wins.
func TestPathSkilldozerConfigAbsoluteOverride(t *testing.T) {
	// Do NOT call t.Parallel() — mutates SKILLDOZER_CONFIG / XDG_CONFIG_HOME.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // prove override WINS over XDG
	t.Setenv(configEnv, "/abs/path/to/cfg.yaml")
	got, err := Path()
	if err != nil {
		t.Fatalf("Path() override abs: err=%v; want nil", err)
	}
	if want := filepath.Clean("/abs/path/to/cfg.yaml"); got != want {
		t.Errorf("Path() override abs: got=%q; want %q", got, want)
	}
}

// Path: a RELATIVE SKILLDOZER_CONFIG override is returned AS-IS (cleaned), NOT
// joined to the config home. This is THE critical no-join test (PRD §8.1 "useful
// for tests / multiple profiles"). Asserts the result is relative and contains
// no "skilldozer" segment, proving it never touched the XDG default.
func TestPathSkilldozerConfigRelativeOverrideNotJoined(t *testing.T) {
	// Do NOT call t.Parallel() — mutates SKILLDOZER_CONFIG / XDG_CONFIG_HOME.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(configEnv, "rel/sub/cfg.yaml")
	got, err := Path()
	if err != nil {
		t.Fatalf("Path() override rel: err=%v; want nil", err)
	}
	if want := filepath.Clean("rel/sub/cfg.yaml"); got != want {
		t.Errorf("Path() override rel: got=%q; want %q", got, want)
	}
	if filepath.IsAbs(got) {
		t.Errorf("Path() override rel: got=%q is absolute; must stay relative (NOT joined to configHome)", got)
	}
	if strings.Contains(got, "skilldozer") {
		t.Errorf("Path() override rel: got=%q contains \"skilldozer\"; override must NOT be joined to the XDG default", got)
	}
}

// Path: an EMPTY SKILLDOZER_CONFIG is equivalent to unset (os.Getenv returns
// "" for both, and the `!=""` guard treats them the same), so Path falls
// through to the os.UserConfigDir() default honoring $XDG_CONFIG_HOME.
func TestPathSkilldozerConfigEmptyFallsToXDG(t *testing.T) {
	// Do NOT call t.Parallel() — mutates SKILLDOZER_CONFIG / XDG_CONFIG_HOME.
	t.Setenv(configEnv, "") // empty == unset
	xdg := t.TempDir()      // controlled absolute config home
	t.Setenv("XDG_CONFIG_HOME", xdg)
	got, err := Path()
	if err != nil {
		t.Fatalf("Path() empty override: err=%v; want nil", err)
	}
	if want := filepath.Join(xdg, "skilldozer", "config.yaml"); got != want {
		t.Errorf("Path() empty override: got=%q; want %q", got, want)
	}
}

// Path: a relative $XDG_CONFIG_HOME is rejected by os.UserConfigDir() with a
// non-nil error, and Path propagates it verbatim with an empty path. Asserts
// only err != nil (the stdlib error wording is not part of the contract).
func TestPathRejectsRelativeXDGConfigHome(t *testing.T) {
	// Do NOT call t.Parallel() — mutates SKILLDOZER_CONFIG / XDG_CONFIG_HOME.
	t.Setenv(configEnv, "") // ensure the override does not short-circuit
	t.Setenv("XDG_CONFIG_HOME", "relative/not-abs")
	got, err := Path()
	if err == nil {
		t.Fatalf("Path() relative XDG_CONFIG_HOME: err=nil; want a non-nil error from os.UserConfigDir")
	}
	if got != "" {
		t.Errorf("Path() relative XDG_CONFIG_HOME: got=%q; want \"\" on error", got)
	}
}

// DefaultStore: an absolute $XDG_DATA_HOME is honored as the base dir.
func TestDefaultStoreAbsoluteXDGDataHome(t *testing.T) {
	// Do NOT call t.Parallel() — mutates XDG_DATA_HOME.
	t.Setenv("XDG_DATA_HOME", "/abs/data")
	got, err := DefaultStore()
	if err != nil {
		t.Fatalf("DefaultStore() abs XDG_DATA_HOME: err=%v; want nil", err)
	}
	if want := filepath.Join("/abs/data", "skilldozer", "skills"); got != want {
		t.Errorf("DefaultStore() abs XDG_DATA_HOME: got=%q; want %q", got, want)
	}
}

// DefaultStore: an EMPTY $XDG_DATA_HOME falls through to ~/.local/share.
func TestDefaultStoreEmptyXDGDataHomeFallsToHome(t *testing.T) {
	// Do NOT call t.Parallel() — mutates XDG_DATA_HOME / HOME.
	t.Setenv("XDG_DATA_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := DefaultStore()
	if err != nil {
		t.Fatalf("DefaultStore() empty XDG_DATA_HOME: err=%v; want nil", err)
	}
	if want := filepath.Join(home, ".local", "share", "skilldozer", "skills"); got != want {
		t.Errorf("DefaultStore() empty XDG_DATA_HOME: got=%q; want %q", got, want)
	}
}

// DefaultStore: a RELATIVE $XDG_DATA_HOME is invalid per the XDG spec and is
// IGNORED — the function falls back to ~/.local/share rather than producing a
// relative store path.
func TestDefaultStoreRelativeXDGDataHomeIgnored(t *testing.T) {
	// Do NOT call t.Parallel() — mutates XDG_DATA_HOME / HOME.
	t.Setenv("XDG_DATA_HOME", "relative/data") // relative -> invalid -> ignored
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := DefaultStore()
	if err != nil {
		t.Fatalf("DefaultStore() relative XDG_DATA_HOME: err=%v; want nil", err)
	}
	if want := filepath.Join(home, ".local", "share", "skilldozer", "skills"); got != want {
		t.Errorf("DefaultStore() relative XDG_DATA_HOME: got=%q; want %q (must fall back to ~/.local/share)", got, want)
	}
}

// DefaultStore: an unset/empty $HOME makes os.UserHomeDir() error, and
// DefaultStore propagates it verbatim with an empty path. (Linux-specific;
// PRD targets Linux.) Asserts only err != nil.
func TestDefaultStoreHomeUnsetErrors(t *testing.T) {
	// Do NOT call t.Parallel() — mutates XDG_DATA_HOME / HOME.
	t.Setenv("XDG_DATA_HOME", "") // force the HOME fallback branch
	t.Setenv("HOME", "")          // os.UserHomeDir -> error
	got, err := DefaultStore()
	if err == nil {
		t.Fatalf("DefaultStore() HOME unset: err=nil; want a non-nil error from os.UserHomeDir")
	}
	if got != "" {
		t.Errorf("DefaultStore() HOME unset: got=%q; want \"\" on error", got)
	}
}
