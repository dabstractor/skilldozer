# Docs & Assets Drift Audit (plan 002) — NON-CORE files vs current PRD.md

Scope: audit every NON-CORE file for drift/gap against the CURRENT `PRD.md`.
Repo: `/home/dustin/projects/skilldozer`. The code was renamed `skpp` → `skilldozer`;
docs/assets may lag. **This report enumerates drift only — no fixes applied.**

Legend: ❌ = drift/gap (must change). ✅ = compliant. ⚠️ = minor / defensible but flagged.

Method: each file was read in full; `grep -i skpp`, `grep 'init|--store|config\.yaml|SKILLDOZER_CONFIG'`
were run repo-wide to cross-check. PRD section numbers cited inline.

---

## Summary table

| # | File | Verdict | Headline drift |
|---|------|---------|----------------|
| 1 | `skills/example/SKILL.md` | ❌ | 7 lines still say `skpp` (incl. `metadata.keywords`) vs PRD §11 |
| 2 | `README.md` | ❌ | No `skilldozer init`, no config file/`SKILLDOZER_CONFIG`; obsolete `go install` caveat; store-priority list has 3 rules, not 5 |
| 3 | `install.sh` | ✅ | Matches PRD §12.1 step-for-step (ldflags, symlink, `skilldozer example`) |
| 4 | `completions/*` | ❌ | `init` subcommand & `--store` flag NOT completed (all 3 files); tags use `--relative --all` vs §14 literal `--all` |
| 5 | `.gitignore` | ✅ | Byte-identical to PRD §16 (5 entries) |
| 6 | `LICENSE` | ✅ | MIT (PRD §19) |
| 7 | `go.mod` | ✅ | module `github.com/dabstractor/skilldozer`; `go 1.25` = lower bound of latest-two window (installed Go 1.26.4) |

---

## 1. `skills/example/SKILL.md` — ❌ DRIFT (7 lines say `skpp`)

**PRD §11** gives the canonical content (says `skilldozer`, `keywords: [example, demo, skilldozer]`).
Current file still uses the `skpp` working title in **7 places**:

| Line | Current (quote) | Must become (per §11) |
|------|-----------------|-----------------------|
| 4 | `  Reference example skill for skpp. Demonstrates the required frontmatter and` | `  Reference example skill for skilldozer. Demonstrates the required frontmatter and` |
| 5 | `  how skpp resolves a tag to an absolute path. Safe to delete once you add real skills.` | `  how skilldozer resolves a tag to an absolute path. Safe to delete once you add real skills.` |
| 7 | `  keywords: [example, demo, skpp]` | `  keywords: [example, demo, skilldozer]` |
| 13 | `This skill exists only so \`skpp\` has something to resolve.` | `This skill exists only so \`skilldozer\` has something to resolve.` |
| 18 | `skpp example                       # prints this directory's absolute path` | `skilldozer example                       # prints this directory's absolute path` |
| 19 | `skpp -f example                    # prints .../skills/example/SKILL.md` | `skilldozer -f example                    # prints .../skills/example/SKILL.md` |
| 20 | `pi --skill "$(skpp example)"       # loads this skill into pi` | `pi --skill "$(skilldozer example)"       # loads this skill into pi` |

Frontmatter `name: example` (line 2) and `category: meta` (line 8) are unchanged and already match §11.
Lines 4–5 (description) and line 13 (body) are also re-branded.

**`--search` impact (PRD §6.1 / §7.1):** line 7 declares `keywords: [example, demo, skpp]`.
Because `skilldozer --search` matches `metadata.keywords` (PRD §6.1 table), the current keyword
set means `skilldozer --search skilldozer` returns **no match** while `skilldozer --search skpp`
matches — the inverse of the intended/PRD-conformant behavior. Fixing line 7 restores parity with
PRD §11 and the README template (`## Adding a skill` uses `[example, demo, skilldozer]`).
Also note: `description` (lines 4–5) drives both pi on-demand loading and `--search`, so the stale
`skpp` token there also pollutes search relevance.

This is the **only** non-core file that still contains `skpp` (confirmed by repo-wide `grep -i skpp`;
all other `skpp` hits are historical `plan/` artifacts and `.git/logs/HEAD`, both out of scope).

---

## 2. `README.md` — ❌ DRIFT (missing §8 config model + stale go-install caveat)

All 8 PRD §15 outline sections are **present** as headings (Why, Install, Usage, Where skills live,
Adding a skill, How `skilldozer` finds the store, Constraints — plus an extra "Shell completions"
section that §15 doesn't enumerate but is harmless). The drift is **content** inside §3/§7/§8.

### 2a. No `skilldozer init`, no config file anywhere — ❌

Repo-wide `grep 'init|config|store'` on README.md returns **zero** hits for the `init` subcommand,
the config file, or `SKILLDOZER_CONFIG`. The only `config`/`init`/`store` tokens are:
- `compinit` and `~/.config/fish/completions/...` (shell-completion install notes — unrelated)
- the words "store" in the §7 heading and the §8 "skills store" constraints line

PRD §15 items 3 & 7 require documenting the config model:
- **§15.3 (Install):** "First run: `skilldozer init` (prompts for the store dir, writes the config)." → MISSING.
- **§15.7 (How `skilldozer` finds the store):** "`skilldozer init` writes a config pointing at the store;
  `SKILLDOZER_SKILLS_DIR` overrides it; sibling / walk-up are zero-config dev fallbacks." → MISSING.
- **§8.1:** config file at `$XDG_CONFIG_HOME/skilldozer/config.yaml`, `SKILLDOZER_CONFIG=<file>` override. → MISSING.

### 2b. "How `skilldozer` finds the store" lists 3 rules, not 5 — ❌ (PRD §8.3)

Current README (lines ~236–244) priority list:
```
1. SKILLDOZER_SKILLS_DIR env var
2. Sibling of the binary
3. Walk up from the current directory
4. Else: fail with a one-line fix telling you how to set SKILLDOZER_SKILLS_DIR.
```
PRD §8.3 requires **5** rules (config rule is the new primary):
```
1. SKILLDOZER_SKILLS_DIR env var
2. Config file store (§8.1) — the primary, set by skilldozer init    ← MISSING
3. Sibling of the binary
4. Walk up from cwd
5. None ⇒ 'skilldozer is not configured; run `skilldozer init`'       ← MISSING
```
Two problems: (i) the **config-file rule** is entirely absent; (ii) the **unconfigured message** in
the README (rule 4) tells the user to "set `SKILLDOZER_SKILLS_DIR`", but PRD §8.3 rule 5 / §6.4 mandates
the exact message `skilldozer is not configured; run \`skilldozer init\``. The README also omits the
`config file` `--path` stderr label (PRD §8.3 enumerates four labels: `SKILLDOZER_SKILLS_DIR`,
`config file`, `sibling of binary`, `ancestor of cwd`).

### 2c. "B. go install" caveat is OBSOLETE — ❌ (PRD §12.2)

README lines ~38–46 carry this caveat block:
```
> **`go install` caveat.** A `go install`'d binary lands in
> `$(go env GOPATH)/bin` with **no** adjacent `skills/` directory, so `skilldozer`
> cannot auto-discover the store from there. Set the runtime override before
> use:
>   export SKILLDOZER_SKILLS_DIR=/absolute/path/to/your/cloned/skilldozer/skills
```
PRD §12.2 states go install is now **first-class**: "on first use the user runs `skilldozer init`
(§8.2), which creates the store and writes the config. **No clone required, no
`SKILLDOZER_SKILLS_DIR` needed for normal use.** The earlier caveat ('must clone the repo and set
the env var') is **obsolete under the config model and is removed.**" The README still presents the
obsolete caveat as the user-facing fix. This caveat should be replaced by "first run: `skilldozer init`".

### 2d. Constraints section omits "settings config file is fine" — ⚠️ minor (PRD §15.8)

README "## Constraints" leads with "**Manifest-free.** No `skills.json`, no index file." PRD §15.8
phrasing is: "no catalog index (disk-discovered); **a settings config file is fine**." The README never
acknowledges that the §8 settings/config file exists and is permitted (contrasts with the no-index rule).
Worth a one-line clarification so users don't think a config file violates the manifest-free rule.

### 2e. Flags coverage — ✅

All task-listed flags ARE documented: `--path`, `--search`, `--list`, `--all`, `check`, `-f`/`--file`,
`--relative`, `--no-color` (see the `## Usage` block, lines ~78–112). `SKILLDOZER_SKILLS_DIR` is
documented (§3B and §7). The `## Adding a skill` template correctly uses
`keywords: [example, demo, skilldozer]` — which makes the SKILL.md `skpp` keyword (item 1) the
inconsistency, not the README template.

### 2f. Note: README §7 example output is stale-because-of-item-1

The §7 `check` example shows `OK    example (example)` — that is independent of the SKILL.md rename
(the `name` field is unchanged) so it remains accurate. No drift there; flagged only to avoid
re-touching it when fixing item 1.

---

## 3. `install.sh` — ✅ COMPLIANT (PRD §12.1)

Step-by-step vs PRD §12.1:

| §12.1 step | PRD requirement | install.sh | Match |
|-----------|-----------------|-----------|-------|
| 1 | `cd` to script's own dir | `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; cd "$SCRIPT_DIR"` | ✅ |
| 2 | verify `go` on PATH; else print instructions, exit 1 | `command -v go` check + heredoc instructions + `exit 1` | ✅ |
| 3 | `go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skilldozer .` | Identical ldflags string, byte-for-byte | ✅ exact |
| 4 | target order: `$SKILLDOZER_INSTALL_BIN` → `~/.local/bin` → `/usr/local/bin` | Same order; no silent sudo (prints sudo hint instead) | ✅ |
| 5 | **Symlink** (not copy) `<target>/skilldozer` → `<repo>/skilldozer` | `ln -sfn "$SCRIPT_DIR/skilldozer" "$TARGET/skilldozer"` | ✅ (uses `-sfn`, absolute target) |
| 6 | ensure target on PATH else print exact export line | `case "$PATH"` + per-shell `export PATH=…` / `fish_add_path` | ✅ |
| 7 | print verification command `skilldozer example` | runs `"$TARGET/skilldozer" --version` then `"$TARGET/skilldozer" example`, and prints `then run: skilldozer example` | ✅ |

Minor observation (not a drift): step 7 actually **executes** the verification via the absolute
symlink path rather than merely *printing* the command. This is a strict superset of "print a
verification command" and matches the PRD's intent. No change needed.

No `skpp` references (verified). install.sh is fully aligned with §12.1.

---

## 4. `completions/` (skilldozer.bash, _skilldozer, skilldozer.fish) — ❌ DRIFT

### 4a. No `skpp` references — ✅
Repo-wide `grep -i skpp` finds no hits in any completion file. Rename applied cleanly here.

### 4b. `init` subcommand is NOT completed (all 3 files) — ❌ (PRD §14 + §6.1 + §8.2)
PRD §14: "They complete: Subcommands/flags after `skilldozer`." §6.1 lists `init` as a first-class
subcommand; §8.2 documents `init --store <dir>`. The delta plan (Mode A, T2.S1) explicitly requires
"`init` as a completable subcommand next to `check`."

Each file currently completes ONLY `check` as a subcommand:
- `skilldozer.bash`: `(( have_pos == 0 )) && cands="$cands check"` — `init` absent.
- `_skilldozer` (zsh): `compadd -- "$tags[@]" check` — `init` absent.
- `skilldozer.fish`: `complete -c skilldozer -n '__fish_is_first_arg' -a 'check' …` — `init` absent.

(The `grep 'init'` hits in these files — `_init_completion`, `compinit` — are bash/zsh completion
**library** helpers, false positives, not the `init` subcommand.)

`init` must be added alongside `check` in all three files (same "offer only as first positional,
suppress further completion once seen" treatment, since `init` is likewise exclusive — see §6.3).

### 4c. `--store` flag is NOT completed (all 3 files) — ❌ (PRD §8.2 / §6.1)
`skilldozer init --store <dir>` is a documented non-interactive flag. None of the three files list
`--store` in their flag matrix (bash `compgen -W` list, zsh `flags` array, fish `complete` directives).
Flag coverage that IS present and correct: `--version/-v`, `--help/-h`, `--path/-p`, `--list/-l`,
`--all/-a`, `--file/-f`, `--relative`, `--no-color`, `--search/-s`. ✅ for those; `--store` is the gap.

### 4d. Tag source is `skilldozer --relative --all`, not literal `skilldozer --all` — ⚠️ minor (PRD §14)
PRD §14: "Tags ... by invoking `skilldozer --all`." All three files instead call
`skilldozer --relative --all`:
- bash: `tags=$(skilldozer --relative --all 2>/dev/null)`
- zsh: `tags=(${(f)"$(skilldozer --relative --all 2>/dev/null)"})`
- fish: `-a '(skilldozer --relative --all 2>/dev/null)'`

This is a **defensible** deviation — `--relative` yields the relTag form the user actually types for
completion, so it is arguably *better* than plain `--all` (which prints absolute paths). But it is not
the literal PRD §14 invocation. Flagging so the implementer can consciously confirm the choice rather
than treat it as a bug. No functional drift.

---

## 5. `.gitignore` — ✅ COMPLIANT (PRD §16)

Current contents (5 lines):
```
/skilldozer
/dist
*.test
*.out
.DS_Store
```
PRD §16 specifies **exactly** these 5 entries (no comments, no sections). Byte-for-byte match.
No diff. (Historical note: plan/001 reduced this from a 9-entry file to the 5 PRD entries; that fix
landed. No regression.)

---

## 6. `LICENSE` — ✅ COMPLIANT (PRD §19)

MIT License, header `MIT License` / `Copyright (c) 2026 Dustin Schultz`, full standard MIT text
(Permission…, NO WARRANTY / "AS IS" paragraph). PRD §19 decision #11: "License: MIT." ✅.

---

## 7. `go.mod` — ✅ COMPLIANT (PRD §5 / §19 #15)

Current contents:
```
module github.com/dabstractor/skilldozer

go 1.25

require gopkg.in/yaml.v3 v3.0.1
```
- **Module path:** `github.com/dabstractor/skilldozer` — matches PRD §5 + §19 #15 exactly. ✅
- **go directive:** `go 1.25`. PRD §5: "Minimum Go: the latest two stable releases."
  Installed toolchain on this host is `go1.26.4` (verified via `go version`). The latest-two-stable
  window as of 2026-07-07 is therefore **1.26 + 1.25**, and `go 1.25` is the *lower bound* of that
  window — i.e. the minimum the module requires while still covering the two newest releases.
  This is consistent with the Go `go` directive's "minimum supported version" semantics and with
  PRD §5. ✅ (No drift. If a maintainer instead reads "latest two" as "set directive to the newest
  release", they might bump to `go 1.26`, but that is not required by the PRD wording.)
- **Dependency:** single direct dep `gopkg.in/yaml.v3 v3.0.1` — matches PRD §7.3 / §8.1 "only
  third-party dependency". ✅

---

## Cross-cutting notes for the implementer

1. **Item 1 is the highest-signal, lowest-effort fix** and is the only file with a literal `skpp`
   residue in the non-core set. It also fixes a functional `--search` inversion (keyword `skpp` vs
   the expected `skilldozer`).
2. **Item 2 (README)** is the largest doc gap: the entire §8 config model + `skilldozer init` UX is
   absent, and the go-install caveat actively contradicts PRD §12.2. The §7 store-priority list must
   grow from 3 → 5 rules.
3. **Item 4 (completions)** `init`/`--store` gaps are small, mechanical, and must be kept LOCKSTEP
   across all three shell files (they each carry a "LOCKSTEP to main.go parseArgs()" comment).
4. **Items 3, 5, 6, 7 are compliant** — verify-not-edit. The only judgment call is item 4d
   (`--relative --all` vs `--all`), which is functionally fine.
5. Out-of-scope `skpp` hits (do NOT touch): `plan/` archive (`delta_prd.md`, `prd_snapshot.md`,
   `prd_index.txt`, P1*/PRP.md, bugfix tasks.json/snapshots) and `.git/logs/*` — these are historical
   records; PRD §19 #15 explicitly states "`plan/` archive left as historical (it *was* skpp when
   written)".