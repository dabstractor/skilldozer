# Delta PRD — Config file + `skilldozer init` (PRD §8 rewrite)

> **Status:** Ready for one-shot implementation. This document specifies ONLY the delta from the completed `skilldozer` v1.0.
> **Repo:** `dabstractor/skilldozer` at `~/projects/skilldozer`. v1.0 already shipped (renamed from `skpp`, full §6 CLI, discovery, resolution, search, check, example skill, install.sh, README, completions).
> **Scope of THIS task:** implement the §8 config-file model + the new `skilldozer init` subcommand, plus the tightly-coupled doc/acceptance sync. Everything else is unchanged.

---

## 0. Diff summary (what actually changed, sized)

The PRD was renamed `skpp` → `skilldozer`. **The rename is already complete in code** (commit `a26da57 Rename skpp to skilldozer across codebase`: binary, env var `SKILLDOZER_SKILLS_DIR`, module path, all `internal/` packages, `install.sh`, completions, README). The ONLY substantive PRD changes that are NOT yet implemented are concentrated in **§8** (locating the skills directory) and its ripple into §2/§6/§12/§13/§17. Specifically:

| # | Change | Type | Implemented? |
|---|---|---|---|
| 1 | **§8.1 config file** — `$XDG_CONFIG_HOME/skilldozer/config.yaml`, `store:` key, `SKILLDOZER_CONFIG=<file>` override, reuses `yaml.v3`, missing→fall-through (not an error) | NEW | ❌ No code reads any config |
| 2 | **§8.2 `skilldozer init`** — interactive (TTY) + non-interactive (`init <dir>` / `init --store <dir>`); cwd auto-detect; mkdir; seed template (string const, **not** `go:embed`); write config; print `--path` + `check`; prompt-safety (bare tag never prompts) | NEW | ❌ No `init` subcommand exists |
| 3 | **§8.3 resolution priority reorder** — NEW order: env → **config `store`** → sibling → walk-up → unconfigured hint | MODIFIED | ❌ Code has old 3-rule order (env → sibling → walk-up) |
| 4 | **§6.4/§8.2 unconfigured error** — bare `skilldozer <tag>` when unconfigured prints `skilldozer is not configured; run \`skilldozer init\`` | MODIFIED | ❌ Code prints old "set $SKILLDOZER_SKILLS_DIR, cd into the repo, or reinstall" |
| 5 | **§8.3 `--path` label** — new `config file` source label (alongside `SKILLDOZER_SKILLS_DIR` / `sibling of binary` / `ancestor of cwd`) | MODIFIED | ❌ `Source.String()` has no config case |
| 6 | **§13 acceptance** — new "Config + first-run" block | NEW | ❌ Not runnable yet |
| 7 | **§2 constraint #1 + §17 guardrail** reworded — "no **catalog** index; a **settings** config file is fine" (contract clarification, not a code behavior) | DOC | code unaffected; README §8 + constraints text must reflect |
| 8 | **§12.2 go install** — now first-class (`skilldozer init` on first run), old "must set `SKILLDOZER_SKILLS_DIR`" caveat removed | DOC | README §3 must change |
| 9 | **§11 example skill body** — PRD §11 now writes `skilldozer example` etc.; `skills/example/SKILL.md` still says `skpp` | DOC | ❌ rename missed this file |

**Size:** medium feature (1 new mechanism + 1 new subcommand + bounded docs). **Not** a full rebuild. Output below is intentionally lean.

**OUT of scope (do NOT touch):** discovery (§7), frontmatter parsing, resolution precedence (§7.2), `--search`/`check`/`--list` logic, table rendering, install.sh build/symlink mechanics, completions *tag* sourcing. These are unchanged and working.

---

## 1. Goal

Make `skilldozer` self-sufficient: a user who `go install`s the binary (no clone) runs `skilldozer init` once, which creates a skills store anywhere and records its location in a config file. Thereafter every `skilldozer <tag>` resolves the store from config — no env var, no checkout required. The env var and sibling/walk-up heuristics survive as overrides/dev fallbacks.

---

## 2. Hard constraints (delta-specific, non-negotiable)

1. **Config is settings, not catalog.** The config file records ONLY where the store lives (the `store:` key). It must never enumerate skills — the catalog stays disk-walked (PRD §2 constraint #1, reworded). This is the one thing the §17 guardrail now explicitly permits.
2. **The bare `skilldozer <tag>` path never prompts and never hangs.** If unconfigured, it prints exactly `skilldozer is not configured; run \`skilldozer init\`` to stderr, writes **nothing** to stdout, exits 1. This protects `pi --skill "$(skilldozer x)"` from hanging inside command substitution. Any interactive prompt (only in `init`) must be `isatty(stdin)`-gated.
3. **`init` is non-destructive.** Adopt an existing skills dir in place; never clobber or delete. Only `mkdir -p` + seed the example template **if the dir is empty**.
4. **No new third-party dependency.** Config parsing reuses the existing `gopkg.in/yaml.v3`. `os.UserConfigDir()` resolves the XDG path (stdlib). No `go:embed`.
5. **One-shot buildable.** No new questions required.

---

## 3. Documentation impact (must not be silently omitted)

**Mode A — doc-with-work (rides with the implementing task):**
- Config rule (Task T1): updates `internal/skillsdir` package doc comment (it currently documents the 3-rule order); `Source.String()` gains the `config file` label consumed by `--path`'s stderr line.
- `init` (Task T2): `main.go` help/USAGE text gains the `init` row (mirrors the existing `check` row); `completions/skilldozer.{bash,fish}` and `completions/_skilldozer` (zsh) gain `init` as a completable subcommand next to `check`.

**Mode B — changeset-level docs (final sync task, depends on all above):**
- `README.md`: §3 Install (go install is first-class → first run is `skilldozer init`; drop the obsolete "must set `SKILLDOZER_SKILLS_DIR`" caveat for `go install`); §7 "How skilldozer finds the store" (config-first priority order + `SKILLDOZER_CONFIG`); §8 Constraints (reworded: no catalog index, settings file is fine).
- `skills/example/SKILL.md`: rename the body `skpp` → `skilldozer` (the §11 content in the PRD already says `skilldozer`; the file was missed by the rename).
- Run the new §13 "Config + first-run" acceptance block verbatim.

---

## 4. Phase / milestones / tasks

### Phase P2 — Config file + `skilldozer init` (PRD §8)

> Builds on the completed v1.0. The `Source` type and `Find() (dir, src, err)` signature in `internal/skillsdir/skillsdir.go` are extended, not replaced. Prior research: `plan/001_fcde63e5bb60/architecture/go_architecture.md` (skillsdir type shape + symlink/EvalSymlinks rationale) and `external_deps.md` (yaml.v3 is the only dep).

#### Milestone M1 — Config resolution + priority reorder (the mechanism)

##### Task T1 — `internal/skillsdir`: config rule + `Find()` reorder + unconfigured error

Add a config-backed resolution rule as §8.3 priority #2, reorder the existing rules down by one, add the `SourceConfig` variant + its `--path` label, and replace the unconfigured error message.

###### Subtask T1.S1 — config rule + SourceConfig + Find() reorder + new ErrNotFound

- **INPUT:** existing `internal/skillsdir/skillsdir.go` (rules: `findEnv` → `findSibling` → `findWalkUp`; `Source{Env,Sibling,WalkUp}`; `ErrNotFound`). Existing `gopkg.in/yaml.v3` dependency.
- **LOGIC:**
  - **Config path resolver** `func configPath() string`: if `SKILLDOZER_CONFIG` env is set, return it verbatim (absolutize via `filepath.Abs`); else `filepath.Join(os.UserConfigDir(), "skilldozer", "config.yaml")`. (`os.UserConfigDir()` honors `$XDG_CONFIG_HOME` on Linux and the correct per-OS dir elsewhere — do not hand-roll XDG.)
  - **Config reader** `func findConfig() (dir string, src Source, found bool)`: read the config file (path from `configPath()`); on any open/parse error OR missing `store:` key OR non-existent store dir, return `found=false` (PRD §8.1: a missing/unreadable config is "not yet configured" and falls through — never a hard error). Unmarshal only `store:` into a tiny struct `type cfgFile struct{ Store string \`yaml:"store"\` }`; ignore unknown keys (lenient, matches frontmatter parser). If `store` is absolute use it; if relative, resolve against the config file's dir. Stat it; existing dir ⇒ `(absDir, SourceConfig, true)`.
  - **Add `SourceConfig`** to the `Source` enum (insert before `SourceSibling`) and a `case SourceConfig: return "config file"` in `Source.String()`.
  - **Reorder `Find()`** to §8.3: rule 1 `findEnv` → rule 2 `findConfig` (NEW) → rule 3 `findSibling` → rule 4 `findWalkUp`. First hit wins. Update the `Find()` doc comment and the package doc comment to the 4-rule order.
  - **Replace `ErrNotFound` message** with EXACTLY: `skilldozer is not configured; run \`skilldozer init\`` (PRD §6.4/§8.2). Keep it a single `errors.New` so `main` prints it verbatim to stderr. (The old "set $SKILLDOZER_SKILLS_DIR, cd into the repo, or reinstall" message is obsolete.)
- **TEST (extend `skillsdir_test.go`):** (a) config with valid `store:` → `(dir, SourceConfig, true)`; (b) config missing → fall through (found=false, no error); (c) config present but `store:` dir absent → fall through; (d) `SKILLDOZER_CONFIG` override points at a custom file; (e) `Find()` returns `SourceConfig` when config wins; (f) env still beats config (`SKILLDOZER_SKILLS_DIR` set + config set ⇒ `SourceEnv`); (g) `ErrNotFound.Error()` equals the new message. Drive config path via `t.Setenv("SKILLDOZER_CONFIG", …)` and `XDG_CONFIG_HOME`.
- **OUTPUT:** `skillsdir.Find()` honors config at priority #2; `--path` reports `config file`; unconfigured invocations print the new hint. Consumed unchanged by `main.go` (which already prints `(found via %s)` to stderr from `src.String()` — Issue 1 QA — so no main change needed for the label).
- **DOCS (Mode A):** package doc comment + `Source.String()` label. No standalone doc file.

#### Milestone M2 — `skilldozer init` subcommand (the UX)

##### Task T2 — `main.go`: `skilldozer init` (interactive + non-interactive)

Implement the §8.2 first-run command as a subcommand (mirrors the existing `check` dispatch pattern in `main.go`).

###### Subtask T2.S1 — init flow (cwd detect → prompt/mkdir → seed → write config → validate)

- **INPUT:** `skillsdir.configPath()` and the `cfgFile{Store}` shape from T1.S1; `internal/discover.Index` (to detect "does cwd already look like a store" and to run `check` at the end); `internal/check` (to validate after seeding).
- **LOGIC:**
  - **Dispatch:** `init` is a positional subcommand (first arg == `"init"`), mutually exclusive with tag resolution — wire it the same way `check` is wired (existing `check bool` field + dispatch). Add `init bool` to the `config` struct and an `initStore` string captured from `init --store <dir>` or the trailing positional `init <dir>`. Add the `init` row to the USAGE/help text next to `check`.
  - **Store resolution (decide the target dir):**
    1. If `--store <dir>` or positional `<dir>` given → use it (non-interactive path; no prompt even on a TTY).
    2. Else compute a **default**: walk cwd (shallow, any depth) for a `SKILL.md`; if found → default = cwd ("detected skills in <cwd>"); else default = `filepath.Join(os.UserDataDir() or $XDG_DATA_HOME, "skilldozer", "skills")`.
    3. If stdin is a TTY (`os.Stdin.Stat()` `ModeCharDevice`) and no explicit dir → prompt `Where should skilldozer keep your skills? [<default>]`; Enter accepts default, typed path overrides. If NOT a TTY and no explicit dir → use the default silently (non-interactive).
  - **Apply:** `os.MkdirAll(store, 0o755)`. If the dir is empty (no entries), seed `example/SKILL.md` from a **string constant** compiled into the binary (the §11 example content, with `skilldozer` wording) — NOT `go:embed`. If non-empty, adopt in place (no clobber).
  - **Write config:** absolutize `store`; write `configPath()` (mkdir its parent) with `store: <abs>\n` (YAML; minimal valid file per §8.1). Preserve unknown keys if the file already existed (read, set `Store`, re-marshal) — but a fresh write of just `store:` is acceptable for v1.
  - **Report:** print the resolved store path to stdout (what `--path`'s stdout would be), then run the `check` logic over the store and print its report. Exit 0 on success, 1 on error/cancel.
  - **Prompt safety (load-bearing):** the `init` command is the ONLY place that prompts. Confirm the bare `skilldozer <tag>` path is untouched and still fails fast via the T1.S1 `ErrNotFound` message (no prompt, no stdout).
- **TEST (extend `main_test.go`):** (a) non-interactive `init --store <tmp>` creates the dir, writes `store:` to `SKILLDOZER_CONFIG`, seeds example when empty, exits 0; (b) `init <tmp>` (positional) equivalent; (c) adopting a non-empty existing dir does NOT clobber; (d) running `init` inside a cwd that contains a `SKILL.md` picks cwd as default (verify via non-interactive path / captured default, since TTY is hard to simulate — factor the default computation into a testable helper `defaultStore(cwd string) string`); (e) after `init`, a subsequent `skilldozer <tag>` resolves via the config rule (`SourceConfig`).
- **OUTPUT:** `skilldozer init` works end-to-end; `go install` users are unblocked without a clone or env var.
- **DOCS (Mode A):** `main.go` USAGE/help text gains the `init` row; `completions/*` gain `init` next to `check` (one line each).

#### Milestone M3 — Docs sync + acceptance (Mode B)

##### Task T3 — Sync changeset-level docs + run new §13 acceptance

Depends on T1.S1 and T2.S1. The final cross-cutting sweep so the changeset ships coherent, non-stale docs (Mode B).

###### Subtask T3.S1 — README / example skill / completions sync + §13 acceptance

- **INPUT:** the final binary (post T1/T2); `README.md`; `skills/example/SKILL.md`; `completions/*`; PRD §13 (new "Config + first-run" block) and §11 (example skill body wording).
- **LOGIC:**
  - **README.md §3 (Install):** `go install` is now first-class — first run is `skilldozer init` (creates the store + writes config); **remove** the obsolete "a go install'd binary has no adjacent skills/ so must set `SKILLDOZER_SKILLS_DIR`" caveat. Keep `install.sh` (symlink) and from-source. Mention `SKILLDOZER_CONFIG` as the config-path override.
  - **README.md §7 (How skilldozer finds the store):** rewrite to the §8.3 priority order (env → config `store` → sibling → walk-up → unconfigured hint) and document `skilldozer init` + `SKILLDOZER_CONFIG`.
  - **README.md §8 (Constraints):** reword to "no **catalog** index (disk-walked each call); a small **settings** config file is expected and fine." Drop any "manifest-free" phrasing that implies config is forbidden.
  - **`skills/example/SKILL.md`:** rename body tokens `skpp` → `skilldozer` (description, keywords `[example, demo, skilldozer]`, and the three `Try:` commands) to match PRD §11 exactly. (Frontmatter `name: example` is unchanged.)
  - **`completions/*`:** ensure `init` is completable alongside `check` (done in T2.S1; verify here).
  - **Acceptance:** from the repo root, run the FULL §13 suite verbatim. The pre-existing lines must still pass; the NEW "Config + first-run" block must pass:
    ```
    mkdir -p /tmp/skilldozer-iso && cp ./skilldozer /tmp/skilldozer-iso/skilldozer && cd /tmp/skilldozer-iso
    env -u SKILLDOZER_SKILLS_DIR HOME=/tmp/skilldozer-iso/home XDG_CONFIG_HOME=/tmp/skilldozer-iso/home/.config \
      ./skilldozer x 2>err; rc=$?
    [ "$rc" = 1 ] && grep -q 'run `skilldozer init`' err && echo "unconfigured-hint OK"
    SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer init --store /tmp/skilldozer-store
    test -d /tmp/skilldozer-store
    grep -q 'store: /tmp/skilldozer-store' /tmp/skilldozer-iso/cfg.yaml
    SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer --path | grep -q /tmp/skilldozer-store
    SKILLDOZER_SKILLS_DIR=/tmp/skilldozer-store SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR
    cd - >/dev/null
    ```
    Plus `go test ./...` green and `go vet ./...` clean. Fix the code (not the test) on any failure.
- **OUTPUT:** README/example/completions consistent with the shipped binary; full §13 green.
- **DOCS (Mode B):** this IS the changeset-level documentation sync.

---

## 5. Build order

1. **T1.S1** (config rule + reorder + unconfigured error) — the mechanism; do first because `init` writes the config it reads.
2. **T2.S1** (`skilldozer init`) — the UX that consumes T1.
3. **T3.S1** (docs + §13 acceptance) — final sync, depends on both.

---

## 6. Decisions log (delta-specific)

| # | Decision | Default chosen | Rationale |
|---|---|---|---|
| D1 | Config location | `$XDG_CONFIG_HOME/skilldozer/config.yaml` via `os.UserConfigDir()`; `SKILLDOZER_CONFIG` override | One fixed bootstrap path; stdlib honors XDG per-OS |
| D2 | Config format | YAML, key `store`, reuses `yaml.v3` | No new dep; matches frontmatter parser |
| 1 | Catalog index | **none** — disk-walked each call (unchanged) | A settings file is permitted; catalog data is not duplicated (PRD §2/§17 reword) |
| D3 | Resolution order | env → config → sibling → walk-up → hint | Env beats config for CI/tests; heuristics kept as zero-config dev fallbacks |
| D4 | Unconfigured behavior | bare tag prints hint, exits 1, no stdout, never prompts | Protects `pi --skill "$(skilldozer x)"` |
| D5 | `init` seed source | string constant in the binary (NOT `go:embed`) | Nothing about the user's collection is compiled in |
| D6 | `init` non-interactive | `init <dir>` and `init --store <dir>`; no-TTY uses default silently | Scriptable; CI-friendly |
| D7 | Existing-config preservation | best-effort: read, set `store`, re-marshal (fresh `store:`-only write acceptable for v1) | Lenient; avoids clobbering future keys |

