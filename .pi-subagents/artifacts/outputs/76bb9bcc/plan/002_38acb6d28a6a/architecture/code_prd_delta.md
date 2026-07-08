# Code-vs-PRD Delta Report — skilldozer v1.0 vs PRD §8 (config + `init`)

> Audit scope: the EXISTING Go codebase at `/home/dustin/projects/skilldozer` (binary built from the committed `skpp` rename) measured against the **current** `PRD.md`. The rename `skpp` → `skilldozer` is already complete in most source files. This report enumerates ONLY the remaining gaps and drifts, each cited to exact `file:line` and exact PRD section. It does **not** propose fixes — only enumerates gaps.
>
> Method: `PRD.md` read in full; `main.go`, `go.mod`, and every file under `internal/{skillsdir,discover,resolve,ui,search,check}` read in full; `README.md`, `skills/example/SKILL.md`, `completions/*` scanned for the rename / new subcommand. The pre-existing `plan/002_38acb6d28a6a/delta_prd.md` was consulted but this audit re-derives every gap from source independently.

---

## 0. Summary of top gaps (severity-ordered)

1. **§8.3 has 5 rules; code implements only 3.** The "config file `store`" rule (priority #2) is entirely absent from `internal/skillsdir/skillsdir.go`, so `Find()` never consults a config file. (`skillsdir.go:232-243`)
2. **§8.2 `skilldozer init` subcommand does not exist.** No `init` dispatch in `main.go` (`run()` `main.go:367-617`), no `init` field in the `config` struct (`main.go:122-136`), no `case "init"` in `parseArgs` (`main.go:145-261`), no `--store` flag. `go install` users have no first-run path.
3. **§6.4/§8.2 unconfigured message is wrong.** `ErrNotFound` (`skillsdir.go:221`) still prints the old `skpp`-era "set $SKILLDOZER_SKILLS_DIR, cd into the repo, or reinstall" message; PRD requires exactly `skilldozer is not configured; run \`skilldozer init\``.
4. **§8.3 `--path` label `config file` missing.** `Source.String()` (`skillsdir.go:39-52`) has no `SourceConfig` variant and no `config file` case.
5. **§8.1 config-file mechanism missing entirely.** No `configPath()`, no `SKILLDOZER_CONFIG` env handling, no `store:` unmarshal, no `os.UserConfigDir()` use anywhere in `internal/` or `main.go` (verified by grep — zero hits for `UserConfigDir`, `SKILLDOZER_CONFIG`, `SourceConfig`, `findConfig`).
6. **`skills/example/SKILL.md` still says `skpp`** (§11 drift) — the rename missed this file.
7. **README §3/§7 stale** — still tells `go install` users to `export SKILLDOZER_SKILLS_DIR`, and §7 documents the old 4-step order.
8. **Completions have no `init`** — bash/zsh/fish complete `check` only.

---

## 1. PRD §8.3 — Locating the skills directory (5 rules vs current 3)

**PRD §8.3 "Resolution priority (first hit wins)":**
> 1. `SKILLDOZER_SKILLS_DIR` env var
> 2. **Config file `store`** (§8.1) — the primary, set by `skilldozer init`.
> 3. Sibling of the running binary (symlink-aware)
> 4. Walk up from `cwd`
> 5. **None** ⇒ unconfigured: stderr one-line fix (`run \`skilldozer init\``), exit `1`.

**Current code — `internal/skillsdir/skillsdir.go`:**

- **Package doc comment, `skillsdir.go:3-7`:**
  ```
  // It implements the PRD §8 priority order:
  //  1. SKILLDOZER_SKILLS_DIR env var — if set and an existing dir, use it as-is.
  //  2. Sibling of the running binary (symlink-aware via os.Executable + EvalSymlinks).
  //  3. Walk up from the current working directory.
  ```
  **GAP:** documents only 3 rules; no config rule, no "none ⇒ unconfigured hint" step. Drifts from §8.3's 5-step ladder.

- **`Find()` doc comment, `skillsdir.go:223-231`:**
  ```
  //  1. SKILLDOZER_SKILLS_DIR env var (rule 1, findEnv).
  //  2. Sibling of the running binary, symlink-aware (rule 2, findSibling).
  //  3. Walk up from cwd (rule 3, findWalkUp).
  // … returns ("", 0, ErrNotFound)
  ```
  **GAP:** no rule 2 `findConfig`, no rule renumbering.

- **`Find()` body, `skillsdir.go:232-243`:**
  ```go
  func Find() (dir string, src Source, err error) {
      if d, s, ok := findEnv(); ok {
          return d, s, nil
      }
      if d, s, ok := findSibling(); ok {       // <-- sibling runs at priority #2; PRD wants it at #3
          return d, s, nil
      }
      if d, s, ok := findWalkUp(); ok {         // <-- walk-up at #3; PRD wants it at #4
          return d, s, nil
      }
      return "", 0, ErrNotFound                 // <-- PRD step 5; message is wrong (see gap §3)
  }
  ```
  **GAP:** the `findConfig` call is entirely missing. Sibling currently sits at priority #2 (PRD #3); walk-up at #3 (PRD #4). When the config rule is inserted it must displace both.

- **ErrNotFound message, `skillsdir.go:218-221`:**
  ```go
  // ErrNotFound is returned by Find when all three §8 rules miss. …
  var ErrNotFound = errors.New("could not locate the skills directory: set $SKILLDOZER_SKILLS_DIR, cd into the skilldozer repo, or reinstall skilldozer")
  ```
  **GAP:** doc says "all three §8 rules"; message is the OLD hint. PRD §8.2/§6.4 + §13 acceptance require EXACTLY `skilldozer is not configured; run \`skilldozer init\`` (the §13 gate greps for `run \`skilldozer init\``, which this string does NOT contain).

- **`Source` enum + `String()`, `skillsdir.go:25-52`:**
  ```go
  type Source int
  const (
      SourceEnv Source = iota     // 0
      SourceSibling               // 1
      SourceWalkUp                // 2
  )
  func (s Source) String() string {
      switch s {
      case SourceEnv:      return "SKILLDOZER_SKILLS_DIR"
      case SourceSibling:  return "sibling of binary"
      case SourceWalkUp:   return "ancestor of cwd"
      default:             return "unknown"
      }
  }
  ```
  **GAP:** no `SourceConfig` variant; `String()` has no `config file` case. PRD §8.3 ("`skilldozer --path` reports … one of the labels: `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`") requires the new label. The §13 acceptance gate `grep -q SKILLDOZER_SKILLS_DIR` (with config set) and `grep -q /tmp/skilldozer-store` (config rule wins) both depend on this.

  - Sub-gap on ordering: PRD §8.3 lists the labels in priority order env → config → sibling → walk-up. Inserting `SourceConfig` between `SourceEnv` and `SourceSibling` keeps the iota order aligned with the priority order (relevant only if any code switches on the numeric value; none currently does, but the `default`/`unknown` test in `skillsdir_test.go:57-65` will need a `SourceConfig` entry — see §6).

- **`envVar` constant, `skillsdir.go:56`:**
  ```go
  const envVar = "SKILLDOZER_SKILLS_DIR"
  ```
  **GAP:** no `configEnv = "SKILLDOZER_CONFIG"` constant and no `configPath()` helper. Sibling/walk-up helpers exist (`findEnv` `skillsdir.go:65-85`, `findSibling` `skillsdir.go:92-100`, `resolveSiblingFromExe` `skillsdir.go:104-128`, `findWalkUpAncestor` `skillsdir.go:177-194`, `findWalkUp` `skillsdir.go:202-208`); the equivalent `findConfig` does not.

**Correctly-aligned parts of the existing 3 rules (no change needed):** rule 1 `findEnv` semantics (env value names an existing dir; not `EvalSymlinks`-ed; absolutized via `filepath.Abs`; bad value falls through silently — matches §8.3 rule 1 + §8.3 "`--path` is the only way to tell which directory actually won"); rule 3 sibling symlink-awareness (`os.Executable` + `filepath.EvalSymlinks` with exe fallback on Eval error); rule 4 walk-up "first ancestor whose `skills/` subdir contains at least one SKILL.md" (`findWalkUpAncestor` + `hasSkillMD` `skillsdir.go:139-166`). These three rules' *behavior* matches PRD §8.3 rules 1/3/4; only their *priority position* and the missing rule 2 are wrong.

---

## 2. PRD §8.1 — Configuration file mechanism (entirely absent)

**PRD §8.1 requires:**
- Default path `$XDG_CONFIG_HOME/skilldozer/config.yaml` (→ `~/.config/skilldozer/config.yaml`); resolved via `os.UserConfigDir()` (stdlib honors XDG per-OS).
- Override the **file path** with `SKILLDOZER_CONFIG=<file>` (useful for tests / profiles).
- One fixed, well-known bootstrap path that does NOT depend on the store location (chicken-and-egg).
- Format YAML; minimal valid file is `store: /home/dustin/skills`. Reuses `gopkg.in/yaml.v3` (the only third-party dep — already in `go.mod`).
- Unknown keys ignored (lenient; room for default category / color prefs later).
- A **missing or unreadable** config treated as "not yet configured" and falls through to §8.3 rules 3–5 — **never a hard error**.

**Current code:** grep over `internal/` and `main.go` for `UserConfigDir`, `SKILLDOZER_CONFIG`, `SourceConfig`, `findConfig`, `configPath`, `cfgFile`, `` `yaml:"store"` `` returns **zero hits**. No config file is read, written, or parsed anywhere.

**GAPs (all NEW — nothing exists):**
- No `configPath()` helper: no `os.UserConfigDir()` call, no `SKILLDOZER_CONFIG` env lookup, no `filepath.Join(...,"skilldozer","config.yaml")`.
- No config reader: no `findConfig() (dir, src Source, found bool)` that reads `configPath()`, unmarshals `{store: <path>}`, absolutizes relative store paths (against the config file's dir per PRD §8.1), `os.Stat`s the store dir, and returns `(absDir, SourceConfig, true)` — OR returns `found=false` on ANY open/parse error, missing `store:` key, or non-existent store dir (never a hard error).
- No `cfgFile` struct with `Store string \`yaml:"store"\`` (lenient: unknown keys ignored by yaml.v3 default).
- No wiring of `findConfig` into `Find()` at priority #2 (`skillsdir.go:232-243`).
- No `SKILLDOZER_CONFIG` constant anywhere.

**`go.mod` (`go.mod:1-3`)** already pins `gopkg.in/yaml.v3 v3.0.1` and `go 1.25` — no dependency work needed; the mechanism is purely missing source.

---

## 3. PRD §8.2 — `skilldozer init` subcommand (entirely absent)

**PRD §8.2 "First-run setup" requires an `init` subcommand with:**
- **Interactive (TTY) flow:**
  1. Auto-detect from cwd: if cwd contains ≥1 `SKILL.md` at any depth, default store = cwd ("detected skills in <cwd>"); else default = `$XDG_DATA_HOME/skilldozer/skills` (→ `~/.local/share/skilldozer/skills`).
  2. Prompt "Where should skilldozer keep your skills? [<default>]" — Enter accepts default, typed path overrides. **Prompt only when stdin is a TTY** (`isatty(stdin)`).
  3. `mkdir -p` the chosen dir if missing.
  4. If the dir is empty, seed `example/SKILL.md` from a **string constant** compiled into the binary (NOT `go:embed` — "nothing about the user's collection is compiled in"). If non-empty, adopt in place; never clobber/delete.
  5. Write `config.yaml` (at `$SKILLDOZER_CONFIG` or default) with the absolute `store` path.
  6. Print the output of `skilldozer --path` (which rule won) and `skilldozer check`.
- **Non-interactive:** `skilldozer init <dir>` (positional) or `skilldozer init --store <dir>` (for scripts/CI). With no dir/`--store`, the same cwd auto-detect applies — adopt cwd if it has a SKILL.md, else the XDG default; no prompt.
- **Prompt safety (load-bearing):** `SKILLDOZER_SKILLS_DIR` set at runtime still bypasses the config entirely. The **bare `skilldozer <tag>` path NEVER prompts** — if unconfigured it prints the §6.4 hint, exits 1, writes nothing to stdout, so `pi --skill "$(skilldozer x)"` fails loudly.
- **§6.1 CLI table row:** `skilldozer init` → stdout "the configured store path", exit `0` on success / `1` on error or cancel.

**Current code — `main.go`:**

- **USAGE text, `main.go:50-98`** — the `USAGE` block (lines 54-62) and `OPTIONS` block (lines 74-82) list: `<tag>`, `--all`, `--list`, `--search`, `check`, `--path`, `--help`, `--version`, plus modifiers `--file`, `--relative`, `--no-color`. **No `init` row anywhere.** `check` appears at `main.go:59` (USAGE), `main.go:72` (EXAMPLES), `main.go:79` (OPTIONS); there is no `init` line at any of the three.
  - **GAP (§6.1):** the `init` row is missing from the help/usage block.

- **`config` struct, `main.go:122-136`:**
  ```go
  type config struct {
      version     bool
      help        bool
      path        bool
      list        bool
      all         bool
      file        bool
      relative    bool
      noColor     bool
      searchMode  bool
      searchQ     string
      check       bool     // `skilldozer check` subcommand … (§9)
      tags        []string
      unknownFlag string
  }
  ```
  **GAP:** no `init bool` field; no `initStore string` field to capture `init <dir>` / `init --store <dir>`; no `--store` value-capture plumbing (analogous to how `--search <q>` captures `c.searchQ` at `main.go:224-232`).

- **`parseArgs`, `main.go:145-261`:**
  - The `=`-form switch (`main.go:159-189`) knows `--version`/`--help`/`--path`/`--list`/`--all`/`--file`/`--relative`/`--no-color`/`--search` — **no `--store`**.
  - The short-bundle expander `expandShortBundle` (`main.go:285-360`) validates chars `v h p l a f s` only — no `--store` short form (PRD §6.2 does not define one, so this is consistent, but the value flag mechanism has no `--store` analog).
  - The main token switch (`main.go:204-243`) has `case "check":` (`main.go:234`) but **no `case "init":`**. `init` is not a reserved positional token; `skilldozer init` would fall through to the default branch (`main.go:244-258`) and be captured as a tag, then fail as an unknown tag in resolve.
  - There is no `--store` long-form handling anywhere.
  - **GAP:** `init` is not parsed as a subcommand; `--store` and positional `<dir>` after `init` are not captured.

- **`run()` dispatch, `main.go:367-617`:**
  - Precedence: help → version → unknownFlag → exclusivity → `c.path` (`main.go:413-426`) → `c.list` (`main.go:428-454`) → `c.searchMode` (`main.go:456-486`) → `c.check` (`main.go:488-545`) → `c.all` (`main.go:547-562`) → `len(c.tags) > 0` (`main.go:564-606`) → no-mode default (`main.go:612-617`).
  - **GAP:** there is **no `if c.init { … }` branch** anywhere. The entire init flow (cwd auto-detect, TTY prompt, `mkdir -p`, seed template, write config, print `--path` + `check`) has no code.
  - Sub-gap: the prompt-safety guarantee that bare `<tag>` never prompts is currently satisfied trivially (no prompt code exists at all), but once `init` lands the guarantee must be re-asserted: `init`'s prompt must be gated on `isatty(stdin)` and must NOT leak into the tag-resolution path. PRD §8.2 "Prompt safety (load-bearing)".

- **`exclusivityError`, `main.go:619-662`:**
  - Lists four families: ≥2 listing modes; tags+listing; check+tags; check+listing (`main.go:635-662`).
  - **GAP:** no `init`-related exclusivity. PRD §6.3 mutual-exclusivity + the §6.1 `init` row (init is its own mode) imply `init` must reject: `init`+tags (a positional `<dir>` after `init` is consumed as the store, not as a tag, but `init`+another-mode like `--list` should exit 2); `init`+`check`+`--all`+`--list`+`--search`+`--path`. (Exact exclusivity set is a design decision; the gap is that NONE is currently encoded.)

- **No string-constant template.** PRD §8.2 step 3 requires seeding `example/SKILL.md` from a compiled-in string constant (NOT `go:embed`). No such constant exists in `main.go` or anywhere in `internal/`. The repo's `skills/example/SKILL.md` exists on disk but is not compiled in (correctly — PRD forbids `go:embed`), so an `init`-seeded store would need the content duplicated as a `const`.

- **No `os.UserDataDir()` / `XDG_DATA_HOME` usage.** The cwd-auto-detect default `$XDG_DATA_HOME/skilldozer/skills` requires `os.UserDataDir()` (stdlib); grep returns zero hits.

**Correctly-reusable building blocks (no gap, listed so the implementer knows what exists):**
- `hasSkillMD(dir)` (`skillsdir.go:139-166`) already implements "dir contains ≥1 SKILL.md at any depth" via `WalkDir` + early-exit sentinel — exactly the §8.2 step-1 cwd auto-detect predicate. It is currently unexported and used only by walk-up; init's cwd auto-detect can reuse it (or a sibling helper).
- `isTerminal` (`main.go:96-112`) is the existing TTY check for *stdout* color; init's `isatty(stdin)` is a different stream (`os.Stdin`) but the same `ModeCharDevice` technique applies.
- `skillsdir.Find()` already returns `(dir, src, err)` and `main.go` already prints `(found via %s)` to stderr from `src.String()` (`main.go:421`) — so once `SourceConfig` exists, `--path`'s stderr line works with no `main.go` change. Confirmed: the §13 gate `grep -q SKILLDOZER_SKILLS_DIR` will pass once the env rule still wins; `grep -q /tmp/skilldozer-store` will pass once `findConfig` is wired at #2.
- `discover.Index` + `check.Check` already exist for init's step-6 "print `check` output" (`main.go:506-545` shows the rendering pattern init can mirror).

---

## 4. PRD §6.1 — CLI table verification (init row missing; rest present)

**§6.1 rows — current code status:**

| §6.1 Row | PRD stdout / exit | Code present? | Evidence |
|---|---|---|---|
| `skilldozer <tag> [<tag>...]` | one abs path/line, in order; `0` if all resolve, `1` if any fail (nothing on stdout) | ✅ | `main.go:564-606` (atomic buffered resolution, §6.4 discipline); `resolve.Resolve` `internal/resolve/resolve.go:90-148` |
| `skilldozer --all` / `-a` | one abs path/line, sorted by tag; `0` (even empty) | ✅ | `main.go:547-562`; discover.Index sorts by RelTag `internal/discover/index.go:79` |
| `skilldozer --list` / `-l` | TAG/NAME/DESCRIPTION table; `0` (`1` if no skills) | ✅ | `main.go:428-454`; `ui.PrintList` `internal/ui/ui.go:64-138` |
| `skilldozer --search <q>` / `-s <q>` | same table, filtered; `0`; `1` if no matches | ✅ | `main.go:456-486`; `search.Search` `internal/search/search.go:39-49` (searches 6 fields incl. aliases/category per §10) |
| `skilldozer check` | OK/WARN/ERROR report + summary; `0` clean / `1` if any ERROR | ✅ | `main.go:506-545`; `check.Check` `internal/check/check.go:97-159` |
| **`skilldozer init`** | the configured store path; `0` success / `1` error/cancel | ❌ **MISSING** | No `init` dispatch in `run()` `main.go:367-617`; no `init` field in `config` `main.go:122-136`; no `case "init"` in `parseArgs` `main.go:145-261`. See §3 above. |
| `skilldozer --path` / `-p` | abs path of resolved dir (stdout); `0` (`1` if unresolvable) | ✅ | `main.go:413-426`; prints dir to stdout + `(found via <src>)` to stderr. (Note: the §13 gate `test "$(./skilldozer --path)" = "$PWD/skills"` works because `$()` captures stdout only — confirmed correct.) |
| `skilldozer --help` / `-h` | help text to stdout; `0` | ✅ | `main.go:382-386` (precedence #1, "help wins"); `usageText` `main.go:48-110` |
| `skilldozer --version` / `-v` | `skilldozer <version>`; `0` | ✅ | `main.go:388-391`; `version` var `main.go:43` (ldflags-overridable) |

**§6.2 modifiers — all present and correct:**
- `--file` / `-f` (`main.go:213-214`, applied in `skillPath` `main.go:683-707`) ✅
- `--no-color` (`main.go:217-218`; gating `isTerminal(stdout) && !c.noColor` at `main.go:451`, `483`, and via `ui.PrintList`) ✅
- `--relative` (`main.go:215-216`; `skillPath` `main.go:695-699`) ✅
- Short-bundle expansion (`-fl`, `-sfoo`, etc.) via `expandShortBundle` `main.go:285-360` ✅
- `=`-form (`--flag=value`) via the `HasPrefix("--") && Contains("=")` branch `main.go:159-189` ✅

**§6.3 default behavior:**
- No args + no flag ⇒ usage to **stderr**, exit `1` ✅ (`main.go:612-617`)
- `--help`/`--version` precedence over everything ✅ (`main.go:382-391`)
- Mixing `<tag>` with `--list`/`--search`/`--all` ⇒ exit `2` ✅ (`exclusivityError` `main.go:646-660`)

**GAP:** only the `init` row (§3) is absent. Every other §6.1/§6.2/§6.3 entry is present and behaves per the table.

---

## 5. PRD §6.4 / §8.2 — Error & exit semantics drift

**§6.4 "Error semantics (critical for `$(...)` use)":**
> Skills dir cannot be located / skilldozer is unconfigured ⇒ stderr: `skilldozer is not configured; run \`skilldozer init\`` (or, if configured but the dir vanished, the concise reason + fix), exit `1`. Bare tag resolution **never** prompts.

- **`ErrNotFound` message, `skillsdir.go:221`:**
  ```
  could not locate the skills directory: set $SKILLDOZER_SKILLS_DIR, cd into the skilldozer repo, or reinstall skilldozer
  ```
  - **GAP:** message does NOT match PRD §6.4/§8.2 required string `skilldozer is not configured; run \`skilldozer init\``. The §13 acceptance gate `grep -q 'run \`skilldozer init\`'` would FAIL on the current binary.
  - **GAP:** message does not distinguish "unconfigured" (rules 1-4 all miss) from "configured but dir vanished" (PRD §6.4 allows a separate "concise reason + fix" for the latter). Currently one static string covers both. (Minor; PRD phrases the vanished-dir case as optional clarification.)

- **Bare-tag prompt-safety guarantee** is currently satisfied vacuously (no prompt code anywhere), but the contract must hold after `init` lands: §8.2 "the bare `skilldozer <tag>` path **never** prompts. If unconfigured … it writes to stderr exactly `skilldozer is not configured; run \`skilldozer init\``, exits `1`, and writes **nothing** to stdout." The current `ErrNotFound`-to-stderr path (`main.go:566-568`, `481-483`, `449-451`, `418-420`) is structurally correct (stderr only, no stdout, exit 1); only the message string drifts.

- **Unknown-tag / ambiguous-tag semantics** (`resolve.UnknownError` `internal/resolve/resolve.go:65-71`, `AmbiguousError` `internal/resolve/resolve.go:78-91`; atomic buffered flush `main.go:582-596`) match §6.4 ("one error line per problem tag", "nothing to stdout", "ambiguous lists candidate full tags"). No gap.

- **Unknown-flag exit 2** (`main.go:393-396`) and **mutually-exclusive exit 2** (`main.go:397-401`, `exclusivityError` `main.go:635-662`) match §6.3. No gap (except the missing `init` exclusivity family noted in §3).

---

## 6. Tests — `internal/skillsdir/skillsdir_test.go` drift

**`TestSourceString`, `skillsdir_test.go:57-72`:**
```go
cases := []struct{ src Source; want string }{
    {SourceEnv, "SKILLDOZER_SKILLS_DIR"},
    {SourceSibling, "sibling of binary"},
    {SourceWalkUp, "ancestor of cwd"},
    {Source(-1), "unknown"},
    {Source(99), "unknown"},
}
```
- **GAP:** no `{SourceConfig, "config file"}` case.

**`TestErrNotFoundMessageHasFix`, `skillsdir_test.go:` (last test):**
```go
for _, want := range []string{"SKILLDOZER_SKILLS_DIR", "cd", "reinstall"} {
    if !strings.Contains(msg, want) { … }
}
```
- **GAP:** asserts the OLD message substrings (`SKILLDOZER_SKILLS_DIR`, `cd`, `reinstall`). PRD requires `run \`skilldozer init\``; the test would need to flip to the new substrings (e.g. `run`, `skilldozer init`).

**Find()/rule tests** (`TestFindRuleEnvWins`, `TestFindRuleWalkUpWins`, `TestFindAllMissReturnsErrNotFound`, `skillsdir_test.go:` near the end) cover only env + walk-up; **no `findConfig`/SourceConfig tests** (valid store, missing config → fall-through, store dir absent → fall-through, `SKILLDOZER_CONFIG` override, env-beats-config precedence). All NEW.

**`main_test.go`** (not read in full, but grep-confirmed): no `init` tests — there is no `init` to test.

---

## 7. Docs / artifact drift (PRD §11, §15, §14, §12.2)

### 7.1 `skills/example/SKILL.md` — rename missed (PRD §11)

**`skills/example/SKILL.md` lines 2-20** still say `skpp` throughout:
- Line 2: `Reference example skill for skpp. Demonstrates …`
- Line 3: `how skpp resolves a tag to an absolute path. …`
- Line 6: `keywords: [example, demo, skpp]`
- Line 11: `This skill exists only so \`skpp\` has something to resolve.`
- Line 17: `skpp example                       # prints this directory's absolute path`
- Line 18: `skpp -f example                    # prints .../skills/example/SKILL.md`
- Line 19: `pi --skill "$(skpp example)"       # loads this skill into pi`

**PRD §11** prescribes the body with `skilldozer example` / `skilldozer -f example` / `pi --skill "$(skilldozer example)"` and `keywords: [example, demo, skilldozer]`. **GAP:** the on-disk example does not match §11; it is load-bearing for the §13 acceptance (`./skilldozer example` resolves, but the *content* still advertises the old name — a doc/UX drift, and the `init` seed template (§3) must use the §11-correct wording).

### 7.2 `README.md` — §3/§7/§8 stale (PRD §12.2, §8.3, §15)

- **`README.md:42-53` ("`go install` caveat"):**
  ```
  > **`go install` caveat.** A `go install`'d binary lands in
  > `$(go env GOPATH)/bin` with **no** adjacent `skills/` directory, so `skilldozer`
  > cannot auto-discover the store from there. Set the runtime override before
  > use:
  > ```bash
  > export SKILLDOZER_SKILLS_DIR=/absolute/path/to/your/cloned/skilldozer/skills
  > ```
  ```
  - **GAP (§12.2):** PRD §12.2 makes `go install` first-class: "on first use the user runs `skilldozer init` (§8.2), which creates the store and writes the config. **No clone required, no `SKILLDOZER_SKILLS_DIR` needed for normal use.** The earlier caveat … is obsolete under the config model and is removed." The README still tells users to `export SKILLDOZER_SKILLS_DIR`. This is exactly the caveat PRD §12.2 explicitly deletes.

- **`README.md:234-249` (§7 "How `skilldozer` finds the store"):**
  - Lists 4 steps: (1) env, (2) sibling, (3) walk-up, (4) "Else: fail with a one-line fix telling you how to set `SKILLDOZER_SKILLS_DIR`".
  - **GAP (§8.3):** missing the config-file rule at priority #2; the "Else" hint is wrong (should be `run \`skilldozer init\``); the `--path` label list (`README.md:248`) omits `config file`.
  - **GAP:** no mention of `skilldozer init` or `SKILLDOZER_CONFIG` (PRD §8.1/§8.2).

- **`README.md` §8 "Constraints" (the "Manifest-free" bullets):** PRD §2 constraint #1 + §17 guardrail were reworded to "no **catalog** index; a **settings** config file is fine". README's current "Manifest-free. No `skills.json`, no index file" phrasing is not wrong, but does not surface the now-permitted settings file. **GAP (§15.8, §17):** README §8 should reflect the reword (catalog vs settings distinction).

### 7.3 `completions/` — no `init` (PRD §14, §6.1)

- `completions/skilldozer.bash`: offers `check` as an exclusive subcommand (`completions/skilldozer.bash:42-48`, `59`); **no `init`**.
- `completions/skilldozer.fish`: `complete … -a 'check' …` (`completions/skilldozer.fish:34-35`); **no `init`**.
- `completions/_skilldozer` (zsh): `compadd -- "$tags[@]" check` (`completions/_skilldozer:40-41`); **no `init`**.
- **GAP (§14, §6.1):** `init` is not completable in any shell. Note `init`'s positional `<dir>` / `--store <dir>` semantics also need completion consideration (PRD §14 keeps completions simple; minimally `init` should appear as a subcommand candidate next to `check`).

### 7.4 `install.sh`

- `install.sh` build/symlink mechanics are unchanged by §8 (PRD §12.1 explicitly says "with `skilldozer init` either works — copy is no longer fatal"). PRD §12.1 step 7 prints `skilldozer example` as the verification command — `install.sh` is not the focus of this audit; no grep hits for `init`/`SKILLDOZER_SKILLS_DIR`/`go install` in it. **No gap asserted** (out of audit scope per the delta's "OUT of scope: install.sh build/symlink mechanics").

---

## 8. Cross-cutting: anything else that could drift

- **`go.mod:3` (`go 1.25`)** and **`go.mod:5`** (`require gopkg.in/yaml.v3 v3.0.1`): no new dep needed for the config mechanism. `os.UserConfigDir()` / `os.UserDataDir()` are stdlib (Go 1.16+/1.15+). **No gap.**
- **`isTerminal` (`main.go:96-112`)** checks *stdout* (`io.Writer` typed to `*os.File`). init's `isatty(stdin)` needs a separate check against `os.Stdin` (different stream, different file). The technique (`ModeCharDevice`) is identical. **Note (not a blocker):** the existing `isTerminal` cannot be reused as-is for stdin gating; init needs its own TTY check or a generalized one.
- **`skillsdir` package exports:** `Find`, `Source`, `SourceEnv/Sibling/WalkUp`, `ErrNotFound` are exported. Adding `SourceConfig` + the `config file` label is additive. **No gap.**
- **`hasSkillMD` (`skillsdir.go:139-166`) is unexported.** If `init`'s cwd auto-detect wants to reuse it from `main.go`, it must be exported (or init duplicates the predicate). **Note.**
- **No `SKILLDOZER_CONFIG` constant** anywhere — the env-var name appears only in PRD/plan docs, never in code. **GAP (§8.1).**

---

## 9. What is CORRECTLY aligned (explicit, to bound the work)

The audit confirmed these already match the current PRD and need **no** change:

- **§7 discovery & §7.2 tag resolution** — `discover.Index` (`internal/discover/index.go:23-81`), `BuildSkill`/metadata extraction (`internal/discover/skill.go:64-101`), `ParseFrontmatter` (`internal/discover/discover.go:75-127`), and `resolve.Resolve` precedence (`internal/resolve/resolve.go:90-148`: canonical → basename → name → alias → unknown, with ambiguity short-circuit) all match §7.1/§7.2/§7.3 exactly. RelTag slash-normalization, nested-skill discovery, lenient unknown-key parsing, missing-frontmatter-still-resolves-by-dir — all present.
- **§9 `check`** — `check.Check` (`internal/check/check.go:97-159`) implements name charset/length (regex `validName` `check.go:90`), description presence + 1024-char WARN (`check.go:113-138`), duplicate-name ERROR (`appendDupFindings` `check.go:158-184`), malformed-YAML-as-"invalid SKILL.md frontmatter" reframing (`check.go:118-126`). Matches §9.
- **§10 search scope** — `search.matches` (`internal/search/search.go:55-92`) searches all six fields (tag/name/description/keywords/aliases/category), individually (boundary-safe). Matches §10.
- **§6.1/§6.2/§6.3 CLI surface except `init`** — every other row/modifier/precedence rule verified present (see §4 table).
- **§6.4 atomicity** — buffered resolution (`main.go:582-596`) prints nothing to stdout on any failure. Matches §6.4.
- **§4 single third-party dep** — only `gopkg.in/yaml.v3` (`go.mod:5`); config reuses it (no new dep).

---

## 10. Gap index (one-line each, for the plan to consume)

| # | PRD ref | File:line | Gap |
|---|---|---|---|
| G1 | §8.3 | `internal/skillsdir/skillsdir.go:232-243` | `Find()` has 3 rules; PRD needs 5. No `findConfig` call at priority #2. |
| G2 | §8.3 | `internal/skillsdir/skillsdir.go:25-37`, `39-52` | No `SourceConfig` enum value; `Source.String()` has no `config file` case. |
| G3 | §8.3 | `internal/skillsdir/skillsdir.go:3-7`, `218-231` | Package/`Find()` doc comments list only 3 rules ("all three §8 rules"). |
| G4 | §6.4 / §8.2 | `internal/skillsdir/skillsdir.go:221` | `ErrNotFound` message is the OLD hint; PRD requires `skilldozer is not configured; run \`skilldozer init\``. |
| G5 | §8.1 | (absent) | No `configPath()` / `SKILLDOZER_CONFIG` / `os.UserConfigDir()` / `cfgFile{Store}` / `findConfig()` anywhere. |
| G6 | §8.2 | `main.go:122-136` | `config` struct has no `init` / `initStore` fields; no `--store` capture. |
| G7 | §8.2 | `main.go:145-261` | `parseArgs` has no `case "init"`, no `--store` handling; `init` falls through to the tag default. |
| G8 | §8.2 | `main.go:367-617` | `run()` has no `if c.init {…}` branch; entire init flow absent (cwd detect, TTY prompt, mkdir, seed, write config, print --path + check). |
| G9 | §8.2 | `main.go:635-662` | `exclusivityError` has no `init` exclusivity family. |
| G10 | §6.1 | `main.go:54-82` (USAGE/EXAMPLES/OPTIONS) | No `init` row in the help block. |
| G11 | §8.2 | (absent) | No compiled-in string-constant example template (PRD forbids `go:embed`); no `os.UserDataDir()` / `$XDG_DATA_HOME` default. |
| G12 | §8.2 | `skillsdir.go:139-166` | `hasSkillMD` (the cwd-auto-detect predicate) is unexported; init either reuses it (needs export) or duplicates. |
| G13 | §8.2 | `main.go:96-112` | `isTerminal` checks stdout; init needs `isatty(stdin)` (different stream). |
| G14 | tests | `internal/skillsdir/skillsdir_test.go:57-72` | `TestSourceString` has no `SourceConfig`/`config file` case. |
| G15 | tests | `internal/skillsdir/skillsdir_test.go` (last test) | `TestErrNotFoundMessageHasFix` asserts the OLD substrings (`SKILLDOZER_SKILLS_DIR`/`cd`/`reinstall`). |
| G16 | tests | `internal/skillsdir/skillsdir_test.go` | No `findConfig`/SourceConfig/env-beats-config precedence tests. |
| G17 | tests | `main_test.go` | No `init` tests (nothing to test yet). |
| G18 | §11 | `skills/example/SKILL.md:2,3,6,11,17,18,19` | Body still says `skpp`; rename missed this file. |
| G19 | §12.2 | `README.md:42-53` | "go install caveat" tells users to `export SKILLDOZER_SKILLS_DIR`; PRD §12.2 removes this caveat. |
| G20 | §8.3 / §15.7 | `README.md:234-249` | §7 documents old 4-step order, wrong hint, missing `config file` label, no `init`/`SKILLDOZER_CONFIG`. |
| G21 | §15.8 / §17 | `README.md` §8 Constraints | Does not surface the catalog-vs-settings reword. |
| G22 | §14 / §6.1 | `completions/{skilldozer.bash,skilldozer.fish,_skilldozer}` | `init` not completable in any shell (only `check`). |
