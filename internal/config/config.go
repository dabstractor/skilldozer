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
