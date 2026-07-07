# System Context — plan 002 (skilldozer §8 config model + drift sync)

## 1. What this plan is

This is a **re-planning pass**, not a greenfield build. The `skilldozer` repo at
`~/projects/skilldozer` is already ~90% implemented and working (binary builds,
`--version`, `--path`, `--list`, tag resolution, `check`, completions, install.sh,
README, LICENSE, .gitignore all exist). Plan `001_fcde63e5bb60` produced and
executed the original `skpp` implementation; the PRD has since evolved (rename
`skpp` → `skilldozer`, plus a major new **§8 config-file + `init` model**), and the
code was only partially updated to match.

**The delta is bounded and cohesive:** bring the existing codebase into full
compliance with the current PRD, centered on the new §8 (Locating the skills
directory) and §8.2 (`init` subcommand), plus sync the residual `skpp` drift.

## 2. Verified current state (ground truth, gathered 2026-07-07)

```
go version go1.26.4   (go.mod says go 1.25 — lower bound of latest-two window, OK)
./skilldozer --version  -> skilldozer 85e1f69        ✅
./skilldozer --path     -> /home/dustin/projects/skilldozer/skills (sibling of binary) ✅
./skilldozer --list     -> shows `example` skill      ✅
./skilldozer example    -> …/skills/example (absolute) ✅
./skilldozer -f example -> …/skills/example/SKILL.md   ✅
./skilldozer nope       -> stderr only, exit 1, nothing on stdout ✅
./skilldozer check      -> OK example; 1 skills, 0 errors ✅
./skilldozer init       -> "unknown skill tag \"init\""  ❌ (init does not exist)
```

Existing internal packages (each WITH tests): `skillsdir`, `discover`,
`resolve`, `ui`, `search`, `check`. Existing non-core: `install.sh`, `README.md`,
`LICENSE`, `.gitignore`, `go.mod`, `completions/{bash,zsh,fish}`, `skills/example/SKILL.md`.

**pi flag contract confirmed** (`pi --help`, pi 0.80.3):
- `--skill <path>`  Load a skill file or directory (can be used multiple times)
- `--no-skills, -ns`  Disable skills discovery and loading
⇒ `pi --no-skills --skill "$(skilldozer example)"` loads ONLY via the explicit
path, exactly as PRD §13 requires. skilldozer outputs a **directory** by default
(pi accepts dir or SKILL.md file); `-f` emits the file path.

## 3. The gap (22 items, summarized — full detail in code_prd_delta.md)

### 3a. Core new feature — §8 config model (the bulk of the work)

`internal/skillsdir/skillsdir.go` implements the **old 3-rule** order (env →
sibling → walk-up). PRD §8.3 now mandates **5 rules**:

1. `SKILLDOZER_SKILLS_DIR` env  — ✅ already implemented (`findEnv`)
2. **Config file `store`** — ❌ ENTIRELY MISSING (new `internal/config` pkg + `findConfig`)
3. Sibling of binary (symlink-aware) — ✅ implemented, just renumbered to #3
4. Walk up from cwd — ✅ implemented, just renumbered to #4
5. None ⇒ `skilldozer is not configured; run \`skilldozer init\`` — ❌ message wrong

Missing pieces (G1–G5, G12–G13):
- New **`internal/config` package**: `File{Store string}`, `Load(path)` (yaml.v3
  lenient unmarshal — unknown keys ignored), `Save(path)` (Marshal + MkdirAll +
  WriteFile), `Path()` (`os.UserConfigDir()` + `SKILLDOZER_CONFIG` override →
  `$XDG_CONFIG_HOME/skilldozer/config.yaml`), `DefaultStore()` (`$XDG_DATA_HOME`
  hand-computed — there is **no** `os.UserDataDir()`).
- **`SourceConfig`** enum value + `"config file"` `Source.String()` case.
- **`findConfig()`** rule consuming `config.Path()` + `config.Load()`, wired into
  `Find()` at priority #2; absolutize relative `store` against the config file dir;
  return `found=false` (never hard error) on missing file / missing key / absent dir.
- **`ErrNotFound`** message → exact PRD string (the §13 gate greps for
  `` run `skilldozer init` ``, which the current message lacks).
- **`hasSkillMD`** is unexported; export it (`HasSkillMD`) so `init`'s cwd
  auto-detect can reuse it (it already implements "dir contains ≥1 SKILL.md").

### 3b. §8.2 `init` subcommand (G6–G11)

`main.go` has no `init` field, no `case "init"`, no `--store`, no `run()` branch,
no string-constant template, no stdin-TTY check. `go install` users currently have
no first-run path (sibling rule misses → ErrNotFound). `init` must:
- parse `init` (positional) + `init <dir>` + `init --store <dir>`;
- TTY-gated prompt (stdin `ModeCharDevice` — reuse the existing stdout technique,
  NOT `golang.org/x/term`, to keep yaml.v3 the only non-stdlib dep);
- cwd auto-detect (reuse `HasSkillMD`) → default `$XDG_DATA_HOME/skilldozer/skills`;
- `mkdir -p`; seed `example/SKILL.md` from a **compiled-in string constant** (NOT
  `go:embed`); write `config.yaml` via `config.Save`; print `--path` + `check`
  output; exit 0/1.
- **Bare `<tag>` path must NEVER prompt** (load-bearing for `pi --skill "$(…)"`).

### 3c. Drift sync (G18–G22)

- `skills/example/SKILL.md` still says `skpp` in 7 lines (incl. `keywords:
  [example, demo, skpp]` → inverts `--search`). PRD §11 prescribes `skilldozer`.
- `README.md`: no `init`/config anywhere; "How skilldozer finds the store" lists 3
  rules not 5; obsolete §12.2 `go install` caveat (tells users to `export
  SKILLDOZER_SKILLS_DIR`); §8 constraints don't surface the catalog-vs-settings reword.
- `completions/*`: `init` + `--store` not completable (only `check`); rename clean.
- `.gitignore`, `LICENSE`, `go.mod`, `install.sh`: ✅ compliant — verify, don't edit.

## 4. Implementation architecture (decided)

**New package `internal/config`** (dependency root; depends on nothing internal):
```
package config
type File struct { Store string `yaml:"store,omitempty"` }
func Load(path string) (File, error)        // yaml.Unmarshal; fs.ErrNotExist ⇒ "not initialized"
func Save(path string, f File) error        // yaml.Marshal + MkdirAll(dir,0755) + WriteFile(0644)
func Path() (string, error)                 // SKILLDOZER_CONFIG override, else os.UserConfigDir()/skilldozer/config.yaml
func DefaultStore() (string, error)         // XDG_DATA_HOME (abs) else ~/.local/share, + /skilldozer/skills
```
Matches the established `discover.go` lenient-unmarshal idiom (in-repo proof that
yaml.v3 ignores unknown keys by default). yaml.v3 remains the SOLE non-stdlib dep.

**Dependency DAG:**
```
internal/config ──► internal/skillsdir (findConfig rule #2; HasSkillMD export)
                └─► main.go (init: DefaultStore + Save + Path; --path label via SourceConfig)
internal/discover ──► main.go (init prints check output)   [unchanged]
```
No new packages touch discover/resolve/ui/search/check (all verified §7/§9/§10-compliant).

**`Find()` becomes:** env → config(`config.Path`+`config.Load`) → sibling → walk-up → ErrNotFound(new msg).

## 5. Suggested build order (mirrors PRD §18, adapted to the delta)

1. `internal/config` package (Load/Save/Path/DefaultStore) — the dependency root.
2. skillsdir: `SourceConfig` + label + doc comments + `HasSkillMD` export.
3. skillsdir: `findConfig()` + wire `Find()` #2 + new `ErrNotFound` message.
4. main.go: `init` parsing/dispatch/USAGE/exclusivity.
5. main.go: init flow (cwd-detect + TTY prompt + mkdir + seed template + write config).
6. main.go: init orchestration (run() branch, print --path+check, exit codes, never-prompt).
7. Asset sync: example SKILL.md; completions init/--store.
8. Final: full §13 acceptance suite; README changeset-level sync (Mode B).

## 6. Tests (implicit TDD — every subtask ships its tests)

- `internal/config`: round-trip Save→Load; unknown-key tolerance; `Path()` honors
  `SKILLDOZER_CONFIG` + `XDG_CONFIG_HOME`; `DefaultStore()` honors `XDG_DATA_HOME`.
- `internal/skillsdir`: `SourceConfig` String(); `findConfig` hit / missing-file
  fall-through / missing-key fall-through / absent-dir fall-through; env-beats-config
  precedence; new `ErrNotFound` message asserts `run \`skilldozer init\``.
- `main_test.go`: `init <dir>` / `init --store <dir>` writes config + creates store;
  init seeds template into empty store; init adopts non-empty store; init never
  clobbers; bare `<tag>` never prompts (no TTY); `--path` reports `config file`
  label; full §13 config-gate assertions.
- §13 acceptance (integration) lives in the final acceptance task.

## 7. Constraints / guardrails reaffirmed (PRD §2, §17)

- No **catalog** index/manifest. A small **settings** config file (store location)
  is explicitly permitted (§8). Never duplicate on-disk catalog data into a sidecar.
- Store never in a pi auto-discovery location; loaded ONLY via `--skill`.
- Never print to stdout on failed tag resolution. Never require a runtime (build-time
  Go only). Ship exactly one example skill.
- Keep yaml.v3 the only non-stdlib dependency (no `golang.org/x/term`, no `go:embed`).
- Do NOT touch historical `plan/` archive or `.git` — they are skpp-era records.
