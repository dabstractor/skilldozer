# PRP — P1.M1.T1.S1: `config.File` struct + `Load` + `Save` (yaml.v3 lenient read/write)

> **Subtask:** P1.M1.T1.S1 — the dependency root of milestone P1.M1 (`internal/config` package).
> **Scope boundary:** Creates the `internal/config` package ONLY — the `File` struct, `Load`, `Save`, and `config_test.go`. Does NOT add `Path()` / `DefaultStore()` (those are S2), does NOT wire anything into `skillsdir.Find()` (T2.S2) or `init` (M2.T2.S2). This PRP's output is a standalone, fully-tested, dependency-free leaf package.

---

## Goal

**Feature Goal**: Create the `internal/config` package — a tiny settings sidecar that reads and writes the skilldozer config file (PRD §8.1) holding only the `store` path, using the repo's existing lenient `yaml.v3` convention, with a missing file distinguishable from a broken one.

**Deliverable**: Two files under `internal/config/`:
1. `config.go` — package `config` with the exported type `File struct { Store string \`yaml:"store,omitempty"\` }` and the exported functions `Load(path string) (File, error)` and `Save(path string, f File) error`, plus a `// Package config …` doc comment.
2. `config_test.go` — table/behavioral tests covering the three contract-required cases (round-trip, unknown-keys-ignored, `fs.ErrNotExist` not masked) plus the hard on-disk-format claim and the MkdirAll parent-creation behavior.

**Success Definition**: `go build ./internal/config/...` and `go test ./internal/config/... -v` both pass; `go vet ./...` is clean; no new module appears in `go.mod`/`go.sum`; `internal/discover`'s lenient-unmarshal convention is matched verbatim; the exported surface is exactly `{File, Load, Save}`.

---

## User Persona (if applicable)

Not applicable — this is an **internal** package with no user-facing or config-surface change (the contract's DOCS section says "none"). Its only callers are two later subtasks: `skillsdir.findConfig` (P1.M1.T2.S2) reads via `Load`; the `init` flow (P1.M2.T2.S2) writes via `Save`.

---

## Why

- This package is the **read/write core of PRD §8.1** ("Configuration file"). The whole §8 config model — `findConfig` reading the store path, `init` writing it — funnels through `File`/`Load`/`Save`. Nothing downstream in P1.M1 or P1.M2 can compile or be tested until this exists.
- It locks the **on-disk format** (`store: <abs>\n`) and the **forward-compat guarantee** (unknown keys ignored) in one place, so the config file can grow fields later (`version`, `default-category`, `colors`) without forcing a coordinated binary upgrade. yaml.v3's default lenient decode is what makes that work.
- It is explicitly **NOT a catalog index** (PRD §2 constraint #1, §17). PRD §2 forbids a `skills.json`/index enumerating the skill *catalog* but permits a small **settings** file for things the filesystem cannot express (where the store lives). This package is the settings sidecar only — its doc comment must state that distinction so no one later mistakes it for catalog storage.

---

## What

A new `internal/config` package with this exact exported API:

```go
package config

type File struct {
    Store string `yaml:"store,omitempty"`
}

func Load(path string) (File, error)  // os.ReadFile + yaml.Unmarshal (lenient)
func Save(path string, f File) error  // yaml.Marshal + MkdirAll(Dir) + WriteFile 0644
```

Behavior (from the contract, verified empirically — see `research/verified_facts.md`):

- **Load**: `os.ReadFile(path)`; on error, return `(File{}, err)` **verbatim** (do NOT mask — `os.ReadFile`'s ENOENT is a `*fs.PathError` wrapping `fs.ErrNotExist`, so callers do `errors.Is(err, fs.ErrNotExist)`). Then `yaml.Unmarshal(data, &f)` with the plain helper (NO `Decoder.KnownFields(true)`); unknown keys are silently ignored (lenient), but syntactically-broken YAML is a hard error returned as-is.
- **Save**: `yaml.Marshal(&f)` → `os.MkdirAll(filepath.Dir(path), 0o755)` → `os.WriteFile(path, out, 0o644)`. Marshal output is deterministic: `File{Store:"/x"}` produces exactly `store: /x\n` (struct-field order, not sorted; no trailing `...`, no BOM).

### Success Criteria

- [ ] `internal/config/config.go` defines exported `File`, `Load`, `Save` and only those
- [ ] `File.Store` is tagged `yaml:"store,omitempty"` exactly
- [ ] `Load` returns the `os.ReadFile`/`yaml.Unmarshal` error **untouched** (testable via `errors.Is(err, fs.ErrNotExist)`)
- [ ] `Load` ignores unknown keys (file with extra keys → `Store` still set, no error)
- [ ] `Load` returns a hard error on syntactically broken YAML (does NOT swallow it)
- [ ] `Save` writes exactly `store: <v>\n` for a non-empty `Store` (verified by reading the file back)
- [ ] `Save` creates the parent directory tree (`os.MkdirAll`) when it does not exist
- [ ] `config_test.go` passes: round-trip equality, unknown-keys-ignored, `fs.ErrNotExist` not masked, exact on-disk format, parent-dir creation
- [ ] `// Package config …` doc comment present; describes it as the store-location settings sidecar, explicitly NOT a catalog index (PRD §2/§17)
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all pass; `go.mod`/`go.sum` unchanged (no new module)

---

## All Needed Context

### Context Completeness Check

**Pass.** The exact struct tag, the three function behaviors, the on-disk format string, the gotchas (empty-File `{}\n`, do-not-mask-ENOENT, no `KnownFields(true)`), the idiom to copy (`discover.go`), and the test conventions (`discover_test.go`) are all specified below with file:line references. An implementer who has never seen this repo can complete it in one pass.

### Documentation & References

```yaml
# MUST READ — the in-repo idiom this package must match, and the verified facts
- file: internal/discover/discover.go
  why: "THE canonical pattern: os.ReadFile -> yaml.Unmarshal (lenient, no KnownFields(true)) -> error returned verbatim. Package doc comment explicitly documents 'Unknown keys are ignored by yaml.v3's default (lenient) decoder'."
  pattern: "Match the read shape and the package-doc-comment style (PRD §7.3 cross-ref, lenient/strict distinction called out in prose)."
  gotcha: "discover.go returns the os.ReadFile error verbatim (ParseFrontmatter: 'data, err := os.ReadFile(path); if err != nil { return ..., err }'). Load must do the same — do NOT wrap."

- file: internal/discover/discover_test.go
  why: "THE test convention to copy: writeX(t, content) helper writing to t.TempDir(); t.Helper()+t.Fatalf setup; direct if-got!=want assertions; clear behavioral names (TestParseFrontmatterUnknownKeysIgnored, TestParseFrontmatterMissingFile)."
  pattern: "Mirror as writeConfig(t, content string) string. No testify / no external test deps (keeps yaml.v3 the only non-stdlib module)."

- file: plan/002_38acb6d28a6a/architecture/external_deps.md
  why: "§1 is the authoritative external fact sheet: the exact Load/Save code skeleton, why yaml.Unmarshal is lenient (vs opt-in Decoder.KnownFields(true)), Marshal determinism (struct-field order), and the ENOENT->fs.ErrNotExist mapping. All empirically re-verified in research/verified_facts.md."
  section: "§1 (yaml.v3 — read a struct, write a struct, ignore unknown keys)"

- file: plan/002_38acb6d28a6a/architecture/code_prd_delta.md
  why: "Gap G5 is the exact gap this subtask closes: 'No configPath() / SKILLDOZER_CONFIG / os.UserConfigDir() / cfgFile{Store} / findConfig() anywhere.' This PRP delivers the cfgFile{Store} + Load/Save half (Path()/DefaultStore()/findConfig are S2/T2)."
  section: "§2 (G5) and §10 gap index"

- file: plan/002_38acb6d28a6a/P1M1T1S1/research/verified_facts.md
  why: "Direct-execution proof of every hard claim in this PRP: exact Marshal outputs ('store: /x\\n'), unknown-keys leniency, errors.Is(fs.ErrNotExist)==true, the empty-File '{}\\n' quirk, the consumer contracts (findConfig reads, init writes)."
  critical: "Sections 1-3 lock the three testable behaviors; section 7 fixes the exported surface to exactly {File, Load, Save}."

- file: PRD.md
  why: "READ-ONLY. §8.1 is the config-file spec (format 'store: <abs>', unknown keys ignored, missing/unreadable => 'not yet configured', reuses yaml.v3). §2 constraint #1 is the 'no catalog index, but a settings sidecar is fine' rule the doc comment must echo."
  section: "§8.1 (Configuration file) and §2 constraint #1 (No catalog index — disk-discovered)"

- url: https://pkg.go.dev/gopkg.in/yaml.v3#Unmarshal
  why: "Confirms the top-level Unmarshal helper is lenient by default (unknown struct fields in the YAML are ignored). This is the function Load calls."
- url: https://pkg.go.dev/gopkg.in/yaml.v3#Decoder.KnownFields
  why: "Confirms strict (unknown-key-erroring) decoding is OPT-IN via an explicit Decoder. Load must NOT construct one. Cited in external_deps.md §1."
- url: https://pkg.go.dev/os#ReadFile
  why: "Confirms a missing path returns a *fs.PathError wrapping fs.ErrNotExist (testable via errors.Is)."
```

### Current Codebase tree (run `tree -L 2` or `ls internal/` in the repo root)

```bash
$ cd /home/dustin/projects/skilldozer && ls internal/
check/  config/   # <-- does NOT exist yet (new package — the ONLY target of this subtask)
discover/  resolve/  search/  skillsdir/  ui/
```

There is **no `internal/config` directory** today (confirmed via `find internal -type f`).
The repo already compiles green: `go.mod` pins `module github.com/dabstractor/skilldozer`,
`go 1.25`, `require gopkg.in/yaml.v3 v3.0.1` (the ONLY non-stdlib dep; already a direct
require because `internal/discover` imports it). A `type config struct` exists in
`main.go` (package `main`) — it is unrelated and in a different package, so there is
no name collision with the new `internal/config` package.

### Desired Codebase tree with files to be added and responsibility of file

```bash
internal/
├── config/
│   ├── config.go         # CREATE — package doc comment + File struct + Load + Save
│   └── config_test.go    # CREATE — round-trip / unknown-keys / ErrNotExist / exact format / MkdirAll
├── discover/   ...        # exists — READ ONLY (the idiom to copy)
└── ...                    # other packages untouched
```

**File responsibilities:**
| File | Responsibility | Owned by |
|---|---|---|
| `internal/config/config.go` | The settings sidecar: read/write the `store:` config value, lenient YAML, never mask a missing file | PRD §8.1 + this contract |
| `internal/config/config_test.go` | Lock the three contract behaviors + the on-disk format + parent-dir creation | The contract's OUTPUT §4 |

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — Do NOT mask the os.ReadFile error. Load must return it VERBATIM.
// os.ReadFile of a missing path returns *fs.PathError{Err: fs.ErrNotExist}.
// findConfig (P1.M1.T2.S2) does `errors.Is(err, fs.ErrNotExist)` to decide to
// fall through to the next §8.3 rule instead of aborting. A fmt.Errorf("load: %w",
// err) wrap technically still satisfies errors.Is, BUT the contract says "return
// the os.ReadFile error so callers can test errors.Is" — return it untouched.
//   GOOD:  return File{}, err
//   BAD:   return File{}, fmt.Errorf("load config %s: %w", path, err)  // do not
//
// GOTCHA #2 — Do NOT call KnownFields(true). Strict decoding is opt-in via an
// explicit yaml.NewDecoder + d.KnownFields(true). The plain yaml.Unmarshal(data,
// &f) helper is lenient: unknown keys (version, colors, …) are silently ignored.
// That is the forward-compat guarantee (PRD §8.1 "Unknown keys are ignored;
// room to grow"). This matches internal/discover/discover.go exactly.
//
// GOTCHA #3 — Syntactically broken YAML is a HARD error, not "lenient".
// "Lenient" means ignore unknown KEYS, NOT tolerate corrupt YAML. yaml.Unmarshal
// returns a non-nil error on e.g. "store: [unbalanced". Propagate it (return
// File{}, err). findConfig treats ANY non-ENOENT error as fall-through today, but
// Load's contract is just "return the error" — do not swallow.
//
// GOTCHA #4 — Marshal of an EMPTY File{} produces "{}\n", not a zero-byte file.
// With omitempty, every field is elided and yaml.v3 emits a flow-mapping "{}".
// Harmless: init (P1.M2.T2.S2) always writes a non-empty store; findConfig treats
// an absent store key as fall-through. Do NOT special-case it; just do not assert
// "empty File writes 0 bytes". The contract's hard format claim is about NON-empty
// Store: File{Store:"/x"} -> exactly "store: /x\n".
//
// GOTCHA #5 — Marshal order is STRUCT-FIELD order, not alphabetical. Put Store
// first (it is the only field today). When fields are added later, list them in
// the on-disk order you want; do not rely on sorting.
//
// GOTCHA #6 — Save must MkdirAll the parent BEFORE WriteFile. config.yaml lives at
// $XDG_CONFIG_HOME/skilldozer/config.yaml (S2) — the skilldozer/ dir will not
// exist on first run. os.MkdirAll(filepath.Dir(path), 0o755) is idempotent (no-op
// if the dir exists). WriteFile then succeeds. (0o755 dirs, 0o644 file — standard.)
//
// GOTCHA #7 — No new dependency. os.ReadFile/WriteFile/MkdirAll, filepath.Dir,
// errors, io/fs are stdlib; yaml.v3 is already pinned. Do NOT `go get` anything,
// do NOT `go mod tidy` (unneeded; yaml.v3 is already a direct require). go.mod and
// go.sum must remain byte-for-byte unchanged by this subtask.
```

---

## Implementation Blueprint

### Data models and structure

The entire data model is one struct (the contract fixes it verbatim):

```go
// File is the parsed skilldozer settings config (PRD §8.1). It is the unmarshal
// target for config.yaml. Unknown keys are ignored by yaml.v3's default (lenient)
// decoder, so the file can grow fields without breaking older binaries.
//
// Field order on disk is struct-field order (yaml.v3 does not sort). omitempty
// keeps an unset Store out of the written file (see GOTCHA #4 for the "{}" quirk).
type File struct {
	Store string `yaml:"store,omitempty"`
}
```

No other models, no validation layer (a missing/empty `Store` is a caller concern —
`findConfig` handles it by falling through, not by erroring here).

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CREATE internal/config/config.go
  - IMPLEMENT: package doc comment (// Package config …) describing it as the
    store-location settings sidecar and explicitly NOT a catalog index (PRD §2/§17)
  - IMPLEMENT: type File struct { Store string `yaml:"store,omitempty"` }  (EXACT tag)
  - IMPLEMENT: func Load(path string) (File, error) — os.ReadFile; on err return
    (File{}, err) VERBATIM (GOTCHA #1); else yaml.Unmarshal(data,&f) (lenient,
    NO KnownFields(true) — GOTCHA #2); broken YAML -> return (File{}, err) (GOTCHA #3)
  - IMPLEMENT: func Save(path string, f File) error — yaml.Marshal(&f); on err return
    err; os.MkdirAll(filepath.Dir(path), 0o755) (GOTCHA #6); os.WriteFile(path,out,0o644)
  - FOLLOW pattern: internal/discover/discover.go (read shape, error-return style, doc comment)
  - IMPORTS: only os, path/filepath, io/fs (if you assert in tests), gopkg.in/yaml.v3
    — do NOT add errors/fmt unless used; a bare `return File{}, err` needs none of them
  - NAMING: package config; exported File/Load/Save; no unexported helpers needed
  - PLACEMENT: internal/config/config.go (new dir; Go auto-discovers packages)

Task 2: CREATE internal/config/config_test.go
  - IMPLEMENT: writeConfig(t, content string) string helper (write to t.TempDir(),
    t.Helper()+t.Fatalf on setup error) — mirror internal/discover's writeSkill
  - TEST TestSaveLoadRoundTrip: Save(File{Store:<abs>}) then Load back; assert
    got.Store==want and err==nil on both. Use a realistic path like "/home/u/skills".
  - TEST TestSaveWritesExactFormat: Save(File{Store:"/x"}) to a temp path, ReadFile
    it back, assert string(out)=="store: /x\n" (locks the on-disk format — GOTCHA #5)
  - TEST TestLoadIgnoresUnknownKeys: write "store: /abs\nversion: 3\ncolors: red\n",
    Load, assert Store=="/abs" and err==nil (lenient — GOTCHA #2)
  - TEST TestLoadMissingFileIsErrNotExist: Load a path under t.TempDir() that does
    not exist; assert err!=nil AND errors.Is(err, fs.ErrNotExist)==true (GOTCHA #1)
  - TEST TestLoadMalformedYAMLIsHardError: write "store: [unbalanced\n", Load, assert
    err!=nil (do NOT assert the yaml message wording — library-internal, may shift)
  - TEST TestSaveCreatesParentDir: Save to filepath.Join(t.TempDir(),"a","b","config.yaml");
    assert no error AND the file exists at that nested path (GOTCHA #6)
  - FOLLOW pattern: internal/discover/discover_test.go (helper style, direct assertions,
    no testify). Mark the 3 contract-required tests clearly in a comment.
  - IMPORTS: errors, io/fs, os, path/filepath, testing, gopkg.in/yaml.v3
  - NAMING: Test<Verb><Condition> (matches discover_test.go's TestParseFrontmatter…)
  - COVERAGE: the 3 contract cases + exact-format + parent-dir + malformed-YAML
  - PLACEMENT: internal/config/config_test.go (same package — internal test is fine;
    no need for _test package external since nothing unexported is tested)

Task 3: VERIFY the package in isolation, then in the whole module
  - COMMAND: go build ./internal/config/...      (exit 0)
  - COMMAND: go test ./internal/config/... -v    (all tests pass)
  - COMMAND: go vet ./...                        (clean; existing packages unaffected)
  - COMMAND: go test ./...                       (whole module green — no regressions)
  - COMMAND: git diff --quiet go.mod go.sum && echo "deps unchanged"
    (MUST report "deps unchanged" — GOTCHA #7)
```

### Implementation Patterns & Key Details

```go
// The Load helper — match discover.go's ParseFrontmatter read shape exactly:
// read, branch on read error (return verbatim), unmarshal lenient, return.
func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// GOTCHA #1: return VERBATIM so callers can errors.Is(err, fs.ErrNotExist).
		// Do NOT wrap with fmt.Errorf.
		return File{}, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		// GOTCHA #3: syntactically broken YAML is a HARD error. Propagate it.
		// (Unknown KEYS are not an error — GOTCHA #2 — because we did not call
		// KnownFields(true); only corrupt YAML reaches here.)
		return File{}, err
	}
	return f, nil
}

// The Save helper — Marshal, ensure parent dir, write.
func Save(path string, f File) error {
	out, err := yaml.Marshal(&f) // deterministic; struct-field order; &f is required
	if err != nil {
		return err
	}
	// GOTCHA #6: config.yaml's dir will not exist on first run. MkdirAll is a no-op
	// when the dir already exists, so this is safe unconditionally.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
```

Notes that are easy to get wrong:
- `yaml.Marshal` takes a pointer (`&f`); passing `f` by value also works but the
  established form (and the format-stability claim) is on `&f`. Use `&f`.
- `filepath.Dir(path)` on a bare filename (no slashes) returns `.` — `MkdirAll(".")`
  is a no-op, so Save to a relative filename in cwd still works.
- The `omitempty` on `Store` means a written file with an empty `Store` elides the
  key entirely (Marshal then emits `"{}\n"` — GOTCHA #4). That is never produced by
  `init` and is harmless to `findConfig`; do not guard against it in Save.

### Integration Points

```yaml
MODULE GRAPH:
  - go.mod UNCHANGED. yaml.v3 is already a direct require (internal/discover imports
    it). No go get, no go mod tidy. (GOTCHA #7)

CONSUMERS (NOT built in this subtask — listed to fix the interface):
  - skillsdir.findConfig (P1.M1.T2.S2): f, err := config.Load(configPath())
      -> if errors.Is(err, fs.ErrNotExist) { fall through }  // why Load must not mask
      -> if f.Store == "" { fall through }
      -> absolutize + os.Stat(f.Store); missing -> fall through; else (absDir, SourceConfig, true)
  - init (P1.M2.T2.S2): config.Save(configPath(), config.File{Store: absStoreDir})

NO ROUTES / NO DATABASE / NO CONFIG-FIELD-ADDITIONS:
  - This subtask adds exactly one struct field (Store). DefaultStore()/Path()/the
    SKILLDOZER_CONFIG env lookup are P1.M1.T1.S2 — do NOT add them here.
```

---

## Validation Loop

### Level 1: Syntax & Style (immediate, after writing config.go)

```bash
cd /home/dustin/projects/skilldozer

gofmt -l internal/config/        # must print NOTHING (file is already gofmt-clean)
go vet ./internal/config/...      # expect exit 0
go build ./internal/config/...    # expect exit 0

# Expected: zero output / exit 0. If gofmt lists the file, run `gofmt -w internal/config/config.go`.
```

### Level 2: Unit Tests (component validation — the core gate)

```bash
cd /home/dustin/projects/skilldozer

go test ./internal/config/... -v
# Expected: 6 tests pass (the 3 contract-required + exact-format + malformed-YAML + MkdirAll).
# Each contract-required test name maps to a contract OUTPUT §4 clause:
#   TestSaveLoadRoundTrip           -> "round-trip Save->Load equality"
#   TestLoadIgnoresUnknownKeys      -> "unknown keys ignored"
#   TestLoadMissingFileIsErrNotExist-> "fs.ErrNotExist returned (not masked) when file absent"

# Confirm the three contract behaviors specifically:
go test ./internal/config/... -run 'TestSaveLoadRoundTrip|TestLoadIgnoresUnknownKeys|TestLoadMissingFileIsErrNotExist' -v
# Expected: 3 passed.
```

### Level 3: Whole-module regression + dependency invariant

```bash
cd /home/dustin/projects/skilldozer

# No regressions anywhere (existing internal/ packages + main still build & pass)
go build ./...   ; echo "build exit $?"
go vet  ./...    ; echo "vet exit $?"
go test ./...    ; echo "test exit $?"
# Expected: all exit 0.

# GOTCHA #7 invariant: go.mod / go.sum must be byte-for-byte unchanged
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
# Expected: "deps unchanged". If it changed, you accidentally added a dependency — undo it.
```

### Level 4: Behavioral spot-checks (lock the hard claims)

```bash
cd /home/dustin/projects/skilldozer

# 4a. Confirm the exact on-disk format claim directly (no test harness, raw marshal):
cat > /tmp/fmtcheck_test.go <<'EOF'
package config_test
import ("os"; "testing"; "github.com/dabstractor/skilldozer/internal/config")
func TestFmtCheck(t *testing.T){
	p := t.TempDir()+"/c.yaml"
	if err := config.Save(p, config.File{Store:"/x"}); err!=nil{ t.Fatal(err) }
	b,err := os.ReadFile(p); if err!=nil{ t.Fatal(err) }
	if string(b)!="store: /x\n" { t.Fatalf("got %q; want \"store: /x\\n\"", b) }
}
EOF
cp /tmp/fmtcheck_test.go internal/config/fmtcheck_test.go
go test ./internal/config/... -run TestFmtCheck -v
rm internal/config/fmtcheck_test.go   # remove the throwaway; keep only config_test.go
# Expected: PASS. (This duplicates TestSaveWritesExactFormat — only run if you skipped writing it.)
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — `gofmt -l` clean, `go vet ./internal/config/...` exit 0, `go build` exit 0
- [ ] Level 2 PASS — `go test ./internal/config/... -v` all pass; the 3 contract tests pass by name
- [ ] Level 3 PASS — `go build/vet/test ./...` all exit 0; `git diff go.mod go.sum` reports "deps unchanged"
- [ ] Level 4 PASS — `store: /x\n` exact-format spot-check passes (or TestSaveWritesExactFormat covers it)

### Feature Validation
- [ ] Exported surface is exactly `{File, Load, Save}` — no `Path()`, `DefaultStore()`, `SKILLDOZER_CONFIG` (those are S2)
- [ ] `File.Store` tagged `yaml:"store,omitempty"` exactly
- [ ] `Load` returns the read/unmarshal error untouched (assertable via `errors.Is(err, fs.ErrNotExist)`)
- [ ] `Load` ignores unknown keys (extra keys → Store set, no error)
- [ ] `Load` errors on broken YAML (does not swallow)
- [ ] `Save` writes exactly `store: <v>\n` for a non-empty Store; creates the parent dir tree
- [ ] Package doc comment present and calls out: settings sidecar for store location; NOT a catalog index (PRD §2/§17)

### Code Quality / Convention Validation
- [ ] Matches `internal/discover/discover.go`'s read shape (os.ReadFile → verbatim err → lenient yaml.Unmarshal)
- [ ] Matches `internal/discover/discover_test.go`'s test style (writeX helper, t.TempDir, direct assertions, no testify)
- [ ] No `Decoder.KnownFields(true)` anywhere (lenient by construction)
- [ ] No `fmt.Errorf` wrap on the Load read error (contract: return verbatim)
- [ ] No new imports beyond stdlib + yaml.v3; no `go get`; no `go mod tidy`

### Scope Discipline
- [ ] Did NOT add `Path()` / `DefaultStore()` / `SKILLDOZER_CONFIG` handling (P1.M1.T1.S2)
- [ ] Did NOT modify `internal/skillsdir` (wiring is P1.M1.T2.S2)
- [ ] Did NOT modify `main.go` / `init` (P1.M2.T2.S2)
- [ ] Did NOT modify `PRD.md` (read-only), `tasks.json`, `prd_snapshot.md`, or `.gitignore`

---

## Anti-Patterns to Avoid

- ❌ **Don't mask the missing-file error.** `fmt.Errorf("load %s: %w", path, err)` breaks the contract's "return the os.ReadFile error so callers can test errors.Is". Return `File{}, err` untouched.
- ❌ **Don't call `KnownFields(true)`.** That turns unknown keys into hard errors and kills the forward-compat guarantee (PRD §8.1). The plain `yaml.Unmarshal` helper is lenient — use it.
- ❌ **Don't swallow broken-YAML errors.** "Lenient" means ignore unknown *keys*, not tolerate corrupt YAML. Propagate the `yaml.Unmarshal` error.
- ❌ **Don't skip `os.MkdirAll(filepath.Dir(path), 0o755)` in Save.** config.yaml's parent dir will not exist on first run; WriteFile would fail with ENOENT. MkdirAll is idempotent and unconditional.
- ❌ **Don't add `Path()` / `DefaultStore()` / `SKILLDOZER_CONFIG`.** They belong to P1.M1.T1.S2. Keep this subtask to the read/write core.
- ❌ **Don't `go get` anything or `go mod tidy`.** yaml.v3 is already pinned and direct; everything else is stdlib. go.mod/go.sum must be unchanged.
- ❌ **Don't assert the yaml.v3 error *message wording* in tests.** It is library-internal and may shift across versions. Assert only `err != nil` on the malformed-YAML path (see how `discover_test.go`'s `TestParseFrontmatterMalformedYAML` does it).
- ❌ **Don't forget the `// Package config …` doc comment.** It is the ONLY documentation for this internal package and must explicitly state it is NOT a catalog index (PRD §2/§17).
- ❌ **Don't create an external `_test` package unless needed.** An internal `package config` test (`config_test.go` in `package config`) is fine — nothing unexported is being tested, but colocating keeps it simple and matches `discover_test.go`.

---

## Confidence Score

**9/10** — Every behavior is pinned to an empirically verified fact (`research/verified_facts.md`): the exact Marshal output, the lenient decode, the `errors.Is(fs.ErrNotExist)` mapping, and the MkdirAll necessity. The single idiom to follow (`internal/discover/discover.go`) already exists in-repo and is read in full. The 1-point reservation is for the one judgment call the contract leaves open: whether `Load`'s verbatim-error return should later gain a light `%w` wrap — the contract says return verbatim, so the PRP enforces that, but a future caller might prefer context. That is a non-issue for the current consumers (findConfig, init) and is explicitly out of scope here.
