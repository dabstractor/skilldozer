# Verified Facts — P1.M1.T1.S1 (`internal/config` package)

Empirical proof, gathered by direct execution in the skilldozer repo environment
(`go 1.25`, `gopkg.in/yaml.v3 v3.0.1`). These back the hard claims in the contract
and the PRP so the implementer does not have to re-derive them.

## 1. yaml.Marshal output is deterministic and exactly `store: <v>\n`

Ran a throwaway program with `type File struct { Store string \`yaml:"store,omitempty"\` }`:

```
yaml.Marshal(&File{Store:"/x"})                  -> "store: /x\n"
yaml.Marshal(&File{Store:"/home/dustin/skills"}) -> "store: /home/dustin/skills\n"
yaml.Marshal(&File{})                            -> "{}\n"        # omitempty drops the key entirely
```

**Implications:**
- The on-disk format is locked: struct-field order (not sorted), one `key: value\n`
  line per set field, no trailing `...` document marker, no BOM. Safe to `git diff`.
- A non-empty `File{Store: s}` always produces exactly `store: <s>\n` — this is the
  shape `findConfig` (P1.M1.T2.S2) and humans will read. Lock it with a test.
- QUIRK: an EMPTY `File{}` marshals to `"{}\n"` (yaml.v3 emits a flow-mapping for the
  zero struct when every field is `omitempty`-elided). This is harmless: `init`
  (P1.M2.T2.S2) always writes a non-empty `store`, and `findConfig` treats a file
  whose `store` key is absent as "fall through to the next rule". Do NOT special-case
  it; just be aware `os.WriteFile` of an empty `File{}` is NOT a zero-byte file.

## 2. yaml.Unmarshal is lenient by default (unknown keys ignored, no error)

```
yaml.Unmarshal([]byte("store: /abs\nversion: 3\nfuture:\n  cache: true\n"), &f)
  -> f.Store == "/abs", err == nil
```

This is yaml.v3's default. Strict decoding is OPT-IN via an explicit `Decoder` +
`d.KnownFields(true)`. The `Load` helper uses the plain `yaml.Unmarshal(data, &f)`
form and must NOT call `KnownFields(true)`, so the config file can grow new keys
(`version`, `default-category`, `colors`, …) without breaking older binaries. This
matches the in-repo idiom in `internal/discover/discover.go` (Frontmatter parsing,
same comment block).

## 3. A missing file returns an error that wraps `fs.ErrNotExist` — do NOT mask it

```
_, err := os.ReadFile("/tmp/.../does-not-exist.yaml")
  -> err = "open .../does-not-exist.yaml: no such file or directory"
     errors.Is(err, fs.ErrNotExist) == true
```

`os.ReadFile` returns a `*fs.PathError` whose `Err` field is `fs.ErrNotExist`.
`Load` must return this error VERBATIM (no wrapping, no `fmt.Errorf("load: %w")`)
so the downstream caller `findConfig` (P1.M1.T2.S2) can branch on
`errors.Is(err, fs.ErrNotExist)` and fall through to the next §8.3 rule instead of
aborting. (A light `%w` wrap still satisfies `errors.Is`, but the contract says
"return the os.ReadFile error so callers can test errors.Is" — return it untouched.)

## 4. The repo already uses this exact idiom — match it

`internal/discover/discover.go` `ParseFrontmatter` does exactly:
`os.ReadFile` -> (error returned verbatim) -> `yaml.Unmarshal([]byte, &struct)` ->
(lenient; error only on syntactically-broken YAML). The package doc comment there
explicitly documents "Unknown keys are ignored by yaml.v3's default (lenient)
decoder". `internal/config/config.go` should read the same way and carry a parallel
package doc comment (the ONLY doc for this internal package per the contract).

## 5. Test conventions to copy (from `internal/discover/discover_test.go`)

- A `writeX(t, content)` helper writes the fixture to `t.TempDir()` and returns its
  path; `t.Helper()` + `t.Fatalf` for setup errors. Each fixture gets its own temp
  dir so they never collide. → mirror as `writeConfig(t, content) string`.
- Direct-assertion style: `if got != want { t.Errorf("%q; want %q", got, want) }`,
  not `testify`. No external test deps (keeps `yaml.v3` the only non-stdlib module).
- Clear behavioral names: `TestParseFrontmatterUnknownKeysIgnored`,
  `TestParseFrontmatterMissingFile`. → `TestLoadIgnoresUnknownKeys`,
  `TestLoadMissingFileIsErrNotExist`, `TestSaveLoadRoundTrip`, etc.

## 6. No dependency change, no new module

`go.mod` already pins `require gopkg.in/yaml.v3 v3.0.1` and `go 1.25`. `os.ReadFile`,
`os.WriteFile`, `os.MkdirAll`, `filepath.Dir`, `errors`, `io/fs` are all stdlib.
Creating `internal/config` adds zero modules to the graph; `go mod tidy` is NOT
needed (and is safe to skip — yaml.v3 is already a direct require because
`internal/discover` imports it).

## 7. Consumer contracts this PRP must satisfy (so the interface is right)

- `findConfig` (P1.M1.T2.S2, `internal/skillsdir`): calls `config.Load(path)`,
  branches on `errors.Is(err, fs.ErrNotExist)` → fall through (never abort); reads
  `f.Store`; also handles "store key absent" and "store dir does not exist" itself
  (those are findConfig's concerns, NOT Load's).
- `init` (P1.M2.T2.S2, `main.go`): calls `config.Save(path, config.File{Store: absDir})`.
- So the exported surface is exactly: `config.File`, `config.Load`, `config.Save`.
  Do not add `Path()`/`DefaultStore()` here — those are P1.M1.T1.S2.
