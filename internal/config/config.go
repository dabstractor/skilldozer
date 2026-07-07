// Package config reads and writes the skilldozer settings file (PRD §8.1), the
// small sidecar that records only the store location (the absolute path to the
// skills directory). The whole §8 config model funnels through this package:
// findConfig (P1.M1.T2.S2) reads the store via Load, and init (P1.M2.T2.S2)
// writes it via Save.
//
// This is a SETTINGS SIDECAR, not a catalog index. PRD §2 constraint #1 (and §17)
// forbid a skills.json-style catalog enumerating the skill set — skills are
// discovered by walking the on-disk store. The only thing this file persists is a
// value the filesystem cannot express (where the store lives). Do not grow this
// struct into a catalog.
//
// Parsing is LENIENT in the PRD §8.1 sense: unknown keys (version, colors, a
// future default-category, …) are silently ignored by yaml.v3's default decoder,
// so the file can gain fields without breaking older binaries. "Lenient" means
// ignore unknown KEYS, NOT tolerate syntactically broken YAML — corrupt input is
// returned as a hard error. This matches internal/discover's convention exactly.
//
// Path resolution for the settings file and the default skills store is also
// here. Path resolves the config-file location per PRD §8.1 ($XDG_CONFIG_HOME/
// skilldozer/config.yaml, with a $SKILLDOZER_CONFIG override taken as a literal
// path). DefaultStore resolves the out-of-the-box skills directory per PRD
// §8.2/§8.3 ($XDG_DATA_HOME/skilldozer/skills, falling back to ~/.local/share/
// skilldozer/skills). Both are pure functions of the environment: they read env
// vars and compute a path, they do NOT touch the filesystem (existence is a
// findConfig/init concern). Locating the file and the default store is squarely
// within the "where does this live" remit of a settings sidecar.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File is the parsed skilldozer settings config (PRD §8.1). It is the unmarshal
// target for config.yaml. Unknown keys are ignored by yaml.v3's default (lenient)
// decoder, so the file can grow fields without breaking older binaries.
//
// Field order on disk is struct-field order (yaml.v3 does not sort). omitempty
// keeps an unset Store out of the written file; an entirely-empty File marshals
// to "{}\n" (a yaml.v3 flow-mapping quirk), which is never produced by init and
// is harmless to findConfig.
type File struct {
	Store string `yaml:"store,omitempty"`
}

// Load reads and parses the config file at path. It implements PRD §8.1.
//
// The os.ReadFile error is returned VERBATIM — it is NOT wrapped with fmt.Errorf —
// so callers can distinguish a missing file from a broken one via
// errors.Is(err, fs.ErrNotExist). findConfig (P1.M1.T2.S2) relies on this to fall
// through to the next §8.3 discovery rule instead of aborting when the file does
// not exist yet.
//
// Unmarshaling uses the plain yaml.Unmarshal helper (no Decoder.KnownFields(true)),
// so unknown keys are silently ignored (lenient / forward-compatible). Syntactically
// broken YAML is a hard error and is propagated as-is.
func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return the read error untouched so callers can errors.Is(err, fs.ErrNotExist).
		return File{}, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		// Syntactically broken YAML is a HARD error (only unknown KEYS are lenient).
		return File{}, err
	}
	return f, nil
}

// Save marshals f and writes it to path, creating any missing parent directories.
// It implements PRD §8.1.
//
// The on-disk format is deterministic: a non-empty Store is written as exactly
// "store: <value>\n" (struct-field order, not sorted; no trailing "..." or BOM).
// The parent directory is created with os.MkdirAll (mode 0o755) before the write —
// config.yaml's directory (e.g. $XDG_CONFIG_HOME/skilldozer/) will not exist on
// first run, and MkdirAll is an idempotent no-op when it already exists. The file
// itself is written with mode 0o644.
func Save(path string, f File) error {
	out, err := yaml.Marshal(&f) // &f: deterministic, struct-field order
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// configEnv is the environment variable that overrides the config-file location
// (PRD §8.1). Set to an absolute or relative path to redirect skilldozer at a
// different config file (useful for tests / multiple profiles). It is read by
// Path; a non-empty value is taken as the literal config-file path (cleaned
// lexically, NOT joined to the config home). Package-internal: no consumer needs
// the symbol — Path encapsulates the read. (Mirrors skillsdir's envVar style.)
const configEnv = "SKILLDOZER_CONFIG"

// Path returns the path to the skilldozer config file (PRD §8.1 — "the one
// fixed, well-known path the binary can bootstrap from"). It is a pure function
// of the environment and reads no filesystem state.
//
// Resolution order:
//
//  1. $SKILLDOZER_CONFIG, if non-empty, is the literal config-file path: returned
//     AS-IS after filepath.Clean (lexical .. / trailing-slash cleanup only; no
//     symlink evaluation). Absolute AND relative values both work — the override
//     is NOT joined to the config home, so a relative value is usable for tests /
//     multiple profiles. (Empty == unset: os.Getenv returns "" for both, and the
//     "" guard means an empty override falls through to the XDG default rather
//     than producing filepath.Clean("") == ".".)
//  2. Otherwise $XDG_CONFIG_HOME/skilldozer/config.yaml, where the XDG config
//     home is os.UserConfigDir() (which honors $XDG_CONFIG_HOME, falls back to
//     ~/.config, and rejects a relative $XDG_CONFIG_HOME with a non-nil error).
//
// Any error from os.UserConfigDir (a relative $XDG_CONFIG_HOME, or neither
// $XDG_CONFIG_HOME nor $HOME defined) is returned VERBATIM, not wrapped:
// findConfig treats any Path error as "config unavailable -> fall through to the
// next §8.3 rule" and never inspects the error type.
func Path() (string, error) {
	if v := os.Getenv(configEnv); v != "" {
		return filepath.Clean(v), nil
	}
	configHome, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configHome, "skilldozer", "config.yaml"), nil
}

// DefaultStore returns the default skills store directory (PRD §8.2 / §8.3).
// It is a pure function of the environment and reads no filesystem state.
//
// Resolution order:
//
//  1. $XDG_DATA_HOME, if set AND absolute, is the base: $XDG_DATA_HOME/skilldozer/
//     skills. (A relative $XDG_DATA_HOME is INVALID per the XDG spec and is
//     ignored — guarded by filepath.IsAbs — so a misconfigured value never
//     produces a relative store path.)
//  2. Otherwise ~/.local/share/skilldozer/skills, where ~ is os.UserHomeDir().
//     There is no os.UserDataDir(), so the XDG data-home rule is computed by
//     hand, exactly as external_deps.md §2 prescribes.
//
// Any error from os.UserHomeDir ($HOME unset) is returned VERBATIM, not wrapped.
// This is the value init (P1.M2.T2) offers as the out-of-the-box store when no
// SKILLDOZER_SKILLS_DIR env var is set, so a go install user gets a sane default.
func DefaultStore() (string, error) {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, "skilldozer", "skills"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "skilldozer", "skills"), nil
}
