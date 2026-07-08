# system_context.md — Verbatim Extract for README Task P1M4T2S1

Source file read in full: `plan/002_38acb6d28a6a/architecture/system_context.md` (147 lines).
Scope note: This document is a **re-planning pass (plan 002)**, not a copy of the PRD.
It summarizes/cites PRD sections but does **not** reproduce most of them verbatim. Quoted
lines below are exact; gaps are called out explicitly with pointers to where the real
verbatim text lives.

---

## (1) PRD §15 "mcpeepants README" tone/structure — ⚠️ NOT PRESENT in this file

`system_context.md` contains **zero** mentions of "mcpeepants", "§15", "README tone",
"voice", or anything describing a README the skilldozer README should mirror. Grep for
`mcpeepants|§15|README tone|README should` → **No matches found** (verified across the
whole file).

Where the real §15 / mcpeepants content actually lives (for the next agent):
- `plan/001_fcde63e5bb60/architecture/mcpeepants_patterns.md` — mcpeepants reference patterns (historical archive).
- `plan/002_38acb6d28a6a/prd_snapshot.md` — current PRD, where §15 verbatim text resides.
- `plan/002_38acb6d28a6a/delta_prd.md` — PRD delta for this plan.

`system_context.md` only references the README as a **drift-sync target** (§3c, quoted
under §3 below): "README.md: no `init`/config anywhere; 'How skilldozer finds the store'
lists 3 rules not 5; ... §8 constraints don't surface the catalog-vs-settings reword."
→ No tone/voice/structure guidance is given here.

---

## (2) §8.1 config file path + SKILLDOZER_CONFIG override

Verbatim, `system_context.md` §3a (lines 58–59):
> `Path()` (`os.UserConfigDir()` + `SKILLDOZER_CONFIG` override →
> `$XDG_CONFIG_HOME/skilldozer/config.yaml`), `DefaultStore()` (`$XDG_DATA_HOME` ...

Verbatim, §4 decided-architecture code block (lines 101–103):
> `type File struct { Store string \`yaml:"store,omitempty"\` }`
> `func Path() (string, error)                 // SKILLDOZER_CONFIG override, else os.UserConfigDir()/skilldozer/config.yaml`
> `func DefaultStore() (string, error)         // XDG_DATA_HOME (abs) else ~/.local/share, + /skilldozer/skills`

Verbatim, §6 test spec (line 132):
> `SKILLDOZER_CONFIG` + `XDG_CONFIG_HOME`; `DefaultStore()` honors `XDG_DATA_HOME`.

Note: No verbatim "§8.1" section heading exists in this file; the path/override spec is
stated only via the package contract above. Note also `Path()` uses `os.UserConfigDir()`
(which already honors `$XDG_CONFIG_HOME`), with `SKILLDOZER_CONFIG` as an override on top.

---

## (3) §8.2 init flow summary

Verbatim, `system_context.md` §3b (lines 72–82):
> `main.go` has no `init` field, no `case "init"`, no `--store`, no `run()` branch,
> no string-constant template, no stdin-TTY check. `go install` users currently have
> no first-run path (sibling rule misses → ErrNotFound). `init` must:
> - parse `init` (positional) + `init <dir>` + `init --store <dir>`;
> - TTY-gated prompt (stdin `ModeCharDevice` — reuse the existing stdout technique,
>   NOT `golang.org/x/term`, to keep yaml.v3 the only non-stdlib dep);
> - cwd auto-detect (reuse `HasSkillMD`) → default `$XDG_DATA_HOME/skilldozer/skills`;
> - `mkdir -p`; seed `example/SKILL.md` from a **compiled-in string constant** (NOT
>   `go:embed`); write `config.yaml` via `config.Save`; print `--path` + `check`
>   output; exit 0/1.
> - **Bare `<tag>` path must NEVER prompt** (load-bearing for `pi --skill "$(…)"`).

This matches every item the task asked for: cwd auto-detect default, `init <dir>` /
`init --store <dir>` non-interactive forms, prints `--path` + `check`, never-prompts on
bare tag.

---

## (4) §8.3 five-rule resolution ladder + four --path labels

Verbatim, `system_context.md` §3a (lines 47–51):
> sibling → walk-up). PRD §8.3 now mandates **5 rules**:
>
> 1. `SKILLDOZER_SKILLS_DIR` env  — ✅ already implemented (`findEnv`)
> 2. **Config file `store`** — ❌ ENTIRELY MISSING (new `internal/config` pkg + `findConfig`)
> 3. Sibling of binary (symlink-aware) — ✅ implemented, just renumbered to #3
> 4. Walk up from cwd — ✅ implemented, just renumbered to #4
> 5. None ⇒ `skilldozer is not configured; run \`skilldozer init\`` — ❌ message wrong

Verbatim `Find()` re-statement, §4 (line 116):
> **`Find()` becomes:** env → config(`config.Path`+`config.Load`) → sibling → walk-up → ErrNotFound(new msg).

Four `--path` labels: the file explicitly names only the **new** one. §3a line 61:
> - **`SourceConfig`** enum value + `"config file"` `Source.String()` case.

Mapping the task's four labels to the rules (rules 1–4 are the "found" cases; rule 5 is the
error/`None` case with no label):
- `SKILLDOZER_SKILLS_DIR`  → rule 1 (env)
- `config file`            → rule 2 (the new `SourceConfig` label, quoted above)
- `sibling of binary`      → rule 3
- `ancestor of cwd`        → rule 4 ("walk up from cwd")

The env/sibling/walk-up label *strings* are not spelled out verbatim in this file; only
`"config file"` is. The exact enum string values live in `internal/skillsdir` source /
the PRD.

---

## (5) §11 example skill frontmatter (name/description/keywords)

`system_context.md` quotes frontmatter only as a **drift correction**, §3c (lines 86–87):
> - `skills/example/SKILL.md` still says `skpp` in 7 lines (incl. `keywords:
>   [example, demo, skpp]` → inverts `--search`). PRD §11 prescribes `skilldozer`.

So the only verbatim frontmatter line captured here is:
> `keywords: [example, demo, skpp]`   (current/WRONG value — §11 wants `skilldozer`)

`name:` and `description:` are **NOT quoted** in this file, and the corrected §11 target
values are not given verbatim (only "prescribes `skilldozer`"). For the exact
name/description/keywords the README "## Adding a skill" template must align to, read
the actual `skills/example/SKILL.md` and/or PRD §11 in `prd_snapshot.md`.

---

## Summary of verbatim-coverage gaps (important)

| Task item | Verbatim in system_context.md? | Where to get it instead |
|---|---|---|
| (1) §15 mcpeepants README tone | ❌ absent entirely | `prd_snapshot.md` §15; `plan/001.../mcpeepants_patterns.md` |
| (2) §8.1 config path / override | ⚠️ partial (package contract quotes) | `prd_snapshot.md` §8.1 |
| (3) §8.2 init flow | ✅ full summary, §3b | also `prd_snapshot.md` §8.2 |
| (4) §8.3 5-rule ladder | ✅ full list, §3a; only `config file` label spelled | `prd_snapshot.md` §8.3 + `internal/skillsdir` for label strings |
| (5) §11 frontmatter | ⚠️ only `keywords:` quoted, wrong value | `skills/example/SKILL.md`; `prd_snapshot.md` §11 |

Conclusion for the README-writing agent: `system_context.md` gives the §8.2 init flow and
§8.3 resolution ladder fully, the config path partially, and **nothing** on §15 README
tone — that must be pulled from `prd_snapshot.md` / the mcpeepants archive directly.
