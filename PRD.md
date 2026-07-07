# PRD — `skpp` (skill path printer)

> **Status:** Ready for one-shot implementation. This document is the complete specification.
> **Repo:** `dabstractor/skpp` (already created and cloned at `~/projects/skpp`).
> **Scope of THIS task for the implementer:** build the tool, the example skill, docs, install, and completions described below. Do not change the product contract without updating this PRD.

---

## 1. Goal

A tiny, fast CLI called **`skpp`** that resolves a human-friendly **skill tag** to the **absolute filesystem path** of a locally-stored [Agent Skill](https://agentskills.io/specification), so it can be loaded into **pi** on demand:

```bash
pi --skill "$(skpp my-skill-tag)" --skill "$(skpp my-other-skill-tag)"
```

`skpp` is to **skills** what `mcpeepants` (`get-server-config.sh`) is to **MCP server configs**: a centralized, on-disk catalog you address by tag, surfaced through a one-liner.

### Why it exists

pi can load skills from many "official" discovery locations (see §3). The user wants a **single centralized store that is deliberately NOT one of those locations**, loaded **only** via the explicit `--skill <path>` flag. `skpp` is the resolver that turns a tag into that path.

---

## 2. Hard constraints (non-negotiable)

1. **No catalog index — disk-discovered.** There is no `skills.json` / index file enumerating the skill *catalog*; the set of skills is always computed by walking the store on each call, so dropping in a directory with a `SKILL.md` makes it instantly available — no rebuild step, no index/disk drift to debug. This is specifically about the **catalog**. It does **not** prohibit a small **settings** file for things the filesystem cannot express — today, where the store lives (see §8). Catalog data already on disk is never duplicated into a sidecar; settings are not catalog data.
2. **No auto-discovery by pi.** Skills live in a location pi does **not** scan. They load **only** through `pi --skill "$(skpp <tag>)"`. The store must never be `~/.pi/agent/skills`, `~/.agents/skills`, a project `.pi/skills` or `.agents/skills`, a `node_modules` package, or anything with a `pi.skills` entry in `package.json`.
3. **`skpp <tag>` prints exactly one absolute path** (to stdout, trailing newline) for a resolved skill — the canonical contract. Unknown tag ⇒ **nothing on stdout**, error to stderr, exit 1.
4. **No development of skills beyond one example.** Ship exactly **one** example skill to prove the pipeline. The repo is a loader, not a skill library.
5. **One-shot buildable.** An implementer must be able to produce the full deliverable from this document alone, with no further questions.

---

## 3. Background — how pi skills work (factual grounding)

Verified against pi's own docs/help (`pi --help`, `docs/skills.md`):

- A **skill** is a directory containing a `SKILL.md` file. Arbitrary sibling assets (`scripts/`, `references/`, `assets/`) are allowed.
- `SKILL.md` starts with YAML **frontmatter** delimited by `---`. Required fields: `name`, `description`. Optional: `license`, `compatibility`, `metadata` (arbitrary map), `allowed-tools`, `disable-model-invocation`.
- `--skill <path>` **"Load a skill file or directory (can be used multiple times)"** — accepts either a `SKILL.md` file path **or** a skill **directory**. It is additive and works even with `--no-skills`.
- `name` rules: 1–64 chars, lowercase `a-z0-9-`, no leading/trailing/consecutive hyphens. **Pi does not require `name` to match the directory** (it relaxes the Agent Skills standard specifically for shared skill dirs).
- `description` max 1024 chars. A skill with **no description is not loaded** by pi.
- pi discovers skills from official locations; we **deliberately use none of them** — we only ever feed pi an explicit `--skill` path.

**Decision:** `skpp` emits the skill **directory** path (not the `SKILL.md` file), because that's the natural unit (includes assets) and `--skill <dir>` is explicitly supported. A `--file` flag is provided for callers who want the `SKILL.md` path instead.

---

## 4. Recommended stack

**Go.** Rationale:

| Need | Go fit |
|---|---|
| Called inside `$(...)` many times per command → startup latency matters | Go binary starts in <5ms; Node ~50ms+ |
| Trivial install, no runtime | Single statically-linked binary; drop in `PATH` |
| Find the skills dir relative to the binary, even through a symlink | `os.Executable()` + `filepath.EvalSymlinks()` (Linux/macOS) |
| Walk dirs, parse simple YAML, format tables | `path/filepath.WalkDir`, tiny frontmatter parser (or `gopkg.in/yaml.v3`) |
| Cross-platform releases | `GOOS`/`GOARCH` matrix; `go install` / release binaries |

Alternatives considered and **rejected**:
- **TypeScript/Node/Bun** — runtime dependency, slower cold start, install friction. (pi itself is Node, so the runtime is present, but distribution and latency are worse.)
- **Rust** — equally good binary, but slower compile/more ceremony than this small CLI warrants.

> If the implementer has a strong reason to use Rust instead, the CLI contract (§6) and discovery rules (§7) stay identical; only the build steps change. **Default to Go.**

---

## 5. Target repository layout

```
skpp/
├── PRD.md                  # THIS file (already exists)
├── README.md              # User docs (mirror mcpeepants style)
├── LICENSE                # MIT (match mcpeepants conventions)
├── go.mod                 # module github.com/dabstractor/skpp
├── go.sum
├── .gitignore             # /skpp (built binary), coverage, OS files
├── main.go                # entrypoint: arg parsing, dispatch
├── internal/
│   ├── discover/
│   │   └── discover.go    # scan skills dir, parse frontmatter, build index
│   ├── resolve/
│   │   └── resolve.go     # tag → skill resolution rules (§7)
│   ├── skillsdir/
│   │   └── skillsdir.go   # locate the skills/ dir (§8 priority order)
│   └── ui/
│       └── ui.go          # --list / --search table formatting (ANSI)
├── install.sh             # build + symlink into PATH (mirrors QUICK_INSTALL.sh)
├── completions/
│   ├── skpp.bash
│   ├── _skpp              # zsh
│   └── skpp.fish
└── skills/
    └── example/           # the ONE shipped example skill
        └── SKILL.md
```

`go.mod` module path: `github.com/dabstractor/skpp`. Minimum Go: the latest two stable releases (set in `go.mod` `go` directive).

---

## 6. CLI contract (authoritative)

Binary name: **`skpp`**. Flags use POSIX double-dash long form + single-dash short forms. Unknown flags ⇒ error + exit 2.

### 6.1 Commands / flags

| Invocation | Behavior | stdout | exit |
|---|---|---|---|
| `skpp <tag> [<tag>...]` | Resolve one or more tags to skill directory paths. | One **absolute** path per line, in input order. | `0` if all resolve; `1` if **any** fail (and **nothing** is printed) |
| `skpp --all` / `-a` | All skills, directory paths. | One absolute path per line (sorted by tag). | `0` |
| `skpp --list` / `-l` | Human-readable catalog. | Table: `TAG`, `NAME`, `DESCRIPTION` (wrapped). | `0` (`1` if no skills found) |
| `skpp --search <q>` / `-s <q>` | Substring (case-insensitive) search over tag, frontmatter `name`, `description`, and `metadata.keywords`. | Same table format as `--list`, filtered. | `0`; `1` if no matches |
| `skpp check` | Validate every skill on disk (see §9). | Report: `OK` lines + any `WARN`/`ERROR` lines. | `0` if clean; `1` if any ERROR |
| `skpp init` | First-run setup (see §8.2): prompt for the skills store dir, create it if missing, write the config, seed a template if empty, validate. Non-interactive: `skpp init <dir>` / `skpp init --store <dir>`. | The configured store path. | `0` on success; `1` on error/cancel |
| `skpp --path` / `-p` | Where is `skpp` looking? | Absolute path of the resolved skills dir. | `0` (`1` if unresolvable) |
| `skpp --help` / `-h` | Usage. | Help text (to stdout). | `0` |
| `skpp --version` / `-v` | Version. | `skpp <version>` (single line). | `0` |

### 6.2 Modifiers (combine with tag resolution or `--all`)

| Flag | Effect |
|---|---|
| `--file` / `-f` | Print the `SKILL.md` file path instead of the directory path. E.g. `skpp -f example`. |
| `--no-color` | Disable ANSI color even on a TTY. |
| `--relative` | Print paths relative to the skills dir instead of absolute (machine-local convenience; default is absolute). |

### 6.3 Default behavior

- **No arguments and no flag** ⇒ print usage to **stderr**, exit `1` (parity with `get-server-config.sh`). (`skpp` with just `--help` prints usage to stdout, exit 0.)
- `--help` / `--version` take precedence over everything else.
- Mixing `<tag>` with `--list`/`--search`/`--all` is an error (exit 2): these are mutually exclusive modes.

### 6.4 Error semantics (critical for `$(...)` use)

- **Any** unresolved/ambiguous tag in a `skpp <tag>...` invocation ⇒ print **one** error line per problem tag to stderr, print **nothing** to stdout, exit `1`. This guarantees `pi --skill "$(skpp badtag)"` fails loudly rather than passing a garbage path.
- Ambiguous tag (a short name matching >1 skill) ⇒ stderr lists the candidate full tags, exit `1`.
- Skills dir cannot be located / skpp is unconfigured ⇒ stderr: `skpp is not configured; run \`skpp init\`` (or, if configured but the dir vanished, the concise reason + fix), exit `1`. Bare tag resolution **never** prompts (see §8.2), so `pi --skill "$(skpp x)"` fails loudly instead of hanging inside command substitution.

---

## 7. Skill discovery & tag resolution

### 7.1 Discovery

1. Locate the skills dir (§8).
2. Walk it recursively. A **skill** = any directory that directly contains a `SKILL.md`. (Nested skills count: `skills/writing/reddit/SKILL.md` is a skill.)
3. For each skill, parse frontmatter (§7.3) and capture:
   - `dir` — absolute path of the skill directory.
   - `relTag` — path of the skill dir **relative to** the skills dir, with OS separators normalized to `/` (e.g. `writing/reddit`). **This is the canonical tag.**
   - `name` — frontmatter `name` (may differ from dir).
   - `description` — frontmatter `description`.
   - `keywords` — `metadata.keywords` (list) if present, else `[]`.
   - `category` — `metadata.category` if present.
   - `aliases` — `metadata.aliases` (list) if present, else `[]`.

> Because everything is read from disk, there is **no index file**. `skpp` rebuilds the index on every invocation (fast: it's a directory walk of a small tree).

### 7.2 Tag resolution precedence (first match wins; later steps only consulted if earlier produced nothing)

Given an input `tag`:

1. **Exact canonical tag** — equals some skill's `relTag` (case-sensitive). Direct hit ⇒ return it.
2. **Basename** — equals the final `/`-component of some skill's `relTag` (e.g. `reddit` matches `writing/reddit`). If **>1** skill matches ⇒ ambiguous error.
3. **Frontmatter `name`** — equals some skill's `name`. If **>1** ⇒ ambiguous error.
4. **Declared alias** — appears in some skill's `metadata.aliases`. If **>1** ⇒ ambiguous error.
5. Nothing ⇒ unknown-tag error.

Examples (assume skills `skills/foo/SKILL.md` with `name: foo-helper`, and `skills/writing/reddit/SKILL.md`):

- `skpp foo` → `…/skills/foo`
- `skpp writing/reddit` → `…/skills/writing/reddit`
- `skpp reddit` → `…/skills/writing/reddit` (basename, unambiguous)
- `skpp foo-helper` → `…/skills/foo` (by `name`)

### 7.3 Frontmatter parsing

- Slice the text between the first two lines that are exactly `---` at the start of `SKILL.md`. If no frontmatter block ⇒ skill still resolves **by directory** (tag/basename) but `check` flags it and `--list` shows `description` as `(missing)`.
- Parse with `gopkg.in/yaml.v3` (robust, handles quoted/multiline scalars). This is the **only** third-party dependency. (A hand-rolled parser is acceptable if it correctly handles quoted values and the `metadata` map; prefer `yaml.v3`.)
- Be lenient: unknown frontmatter keys are ignored (matches pi behavior). Missing optional keys ⇒ defaults.

---

## 8. Locating the skills directory

`skpp` does not assume the store lives next to the binary or inside a checkout. A small settings file records where the user keeps their skills, written by `skpp init` on first use. The store can live anywhere.

### 8.1 Configuration file

- Default location: `$XDG_CONFIG_HOME/skpp/config.yaml` (→ `~/.config/skpp/config.yaml`). Override the file path with `SKPP_CONFIG=<file>` (useful for tests / multiple profiles).
- This is the **one** fixed, well-known path the binary can bootstrap from; it must not depend on the store location (chicken-and-egg: you cannot discover the config from the dir the config points at).
- Format: YAML (reuses the existing `yaml.v3` dependency). Minimal valid file:

  ```yaml
  store: /home/dustin/skills
  ```

- Unknown keys are ignored (room to grow: default category, color prefs, etc.). A missing or unreadable config is treated as "not yet configured" and falls through to §8.3 rules 3-5 — never a hard error.

### 8.2 First-run setup — `skpp init`

`skpp init` is the documented first command and the **only** place skpp prompts interactively.

Interactive (TTY) flow:

1. Prompt: "Where should skpp keep your skills?" Default: `$XDG_DATA_HOME/skpp/skills` (→ `~/.local/share/skpp/skills`).
2. `mkdir -p` the chosen dir if it does not exist.
3. If the dir is empty, seed `example/SKILL.md` as a copy-paste template (a string constant compiled into the binary — **not** `go:embed` of a directory; nothing about the user's collection is compiled in). If the dir already contains skills, adopt it in place; never clobber or delete.
4. Write `config.yaml` (at `$SKPP_CONFIG` or the default location) with the absolute `store` path.
5. Print the output of `skpp --path` (which rule won) and `skpp check`.

Non-interactive: `skpp init <dir>` or `skpp init --store <dir>` (for scripts / CI). `SKPP_SKILLS_DIR` set at runtime still bypasses the config entirely.

**Prompt safety (load-bearing):** the bare `skpp <tag>` path **never** prompts. If unconfigured (every rule in §8.3 misses), it writes to stderr exactly `skpp is not configured; run \`skpp init\``, exits `1`, and writes **nothing** to stdout — so `pi --skill "$(skpp x)"` fails loudly instead of hanging inside command substitution. Any convenience auto-prompt anywhere else must be gated on `isatty(stdin)`.

### 8.3 Resolution priority (first hit wins)

1. **`SKPP_SKILLS_DIR` env var** — override; if set and an existing dir, use it. Lets CI / tests / temporary redirects win without editing the config.
2. **Config file `store`** (§8.1) — the primary, set by `skpp init`.
3. **Sibling of the running binary** (symlink-aware: `os.Executable()` + `filepath.EvalSymlinks()`) — still lets a clone-and-build dev workflow work with zero config.
4. **Walk up from `cwd`** — for `go run` / dev.
5. **None** ⇒ unconfigured: stderr one-line fix (`run \`skpp init\``), exit `1`.

`skpp --path` reports which rule won, on stderr, with one of the labels: `SKPP_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`. This matters because a bad `SKPP_SKILLS_DIR` value is silently ignored and falls through — `--path` is the only way to tell which directory actually won. This remains the single most failure-prone area — implement and test it first (see §13 acceptance).

---

## 9. Validation — `skpp check`

Walks the store and reports problems (exit `1` if any ERROR):

- ERROR: skill dir has no `SKILL.md`.
- ERROR: frontmatter missing `name` or `description`, or `description` empty.
- ERROR: `name` violates Agent Skills rules (length/charset/consecutive hyphens).
- ERROR: duplicate frontmatter `name` across skills (pi would warn + keep first; we surface it).
- WARN: `description` > 1024 chars.
- WARN: a skill dir is empty besides `SKILL.md` (fine, just informational) — optional.

Output format: one line per skill → `OK   <relTag> (<name>)`; problem lines prefixed `ERROR`/`WARN`. Summary line at the end: `N skills, M errors, K warnings`.

---

## 10. Skill directory & frontmatter conventions

A skill under `skills/<tag>/`:

```
skills/<tag>/
├── SKILL.md          # required, valid frontmatter
├── scripts/          # optional helper scripts
├── references/       # optional on-demand docs
└── assets/           # optional static assets
```

**`SKILL.md` frontmatter** — required fields per the Agent Skills standard, plus **skpp conventions** stored under the standard `metadata` map (so nothing is non-standard):

````markdown
---
name: my-skill-tag
description: >
  One to two sentences: what this skill does and precisely when to use it.
  This field drives pi's on-demand loading AND skpp's --search.
metadata:
  keywords: [writing, reddit, social]
  category: writing
  aliases: [reddit-post, social-post]
license: MIT
compatibility: "Requires Python 3.11+"
---

# My Skill

Body instructions for the agent (loaded on-demand by pi).
````

- `name` should match the directory name where practical (but is **not required** to).
- `metadata.keywords` / `metadata.category` / `metadata.aliases` are **optional** and exist only to enrich `skpp --search` and tag aliases. They are standard-compliant (`metadata` is a spec'd optional field).
- All asset/script references inside the body use **paths relative to the skill directory** (pi resolves them against the dir we hand to `--skill`).

---

## 11. The one shipped example skill

Ship **exactly one** example so `--list`/resolution are demonstrable out of the box:

`skills/example/SKILL.md`:
````markdown
---
name: example
description: >
  Reference example skill for skpp. Demonstrates the required frontmatter and
  how skpp resolves a tag to an absolute path. Safe to delete once you add real skills.
metadata:
  keywords: [example, demo, skpp]
  category: meta
---

# Example Skill

This skill exists only so `skpp` has something to resolve.

Try:

```bash
skpp example                       # prints this directory's absolute path
skpp -f example                    # prints .../skills/example/SKILL.md
pi --skill "$(skpp example)"       # loads this skill into pi
```
````

No other skills ship in this repo.

---

## 12. Installation

### 12.1 `install.sh` (mirrors mcpeepants `QUICK_INSTALL.sh` spirit)

Behavior:

1. `cd` to the script's own directory (the repo root).
2. Verify `go` is on `PATH`; if not, print install instructions and exit `1`.
3. `go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .`
4. Pick a target bin dir in this order: `$SKPP_INSTALL_BIN` (if set) → `$HOME/.local/bin` (if present or creatable) → `/usr/local/bin` (if writable, else needs `sudo`).
5. **Symlink** (not copy) `<target>/skpp` → `<repo>/skpp`, so `os.Executable()` resolves back to the repo and finds `skills/`. If a symlink already exists, refresh it.
6. Ensure the target dir is on `PATH`; if not, print the exact `export PATH=…` line for the detected shell (`~/.bashrc` / `~/.zshrc` / `~/.config/fish/config.fish`).
7. Print a verification command: `skpp example`.

> **Why symlink, not copy:** with `skpp init` (§8) either works — copy is no longer fatal, because the store no longer has to be the binary's sibling. Symlink is still recommended for clone users: the sibling-of-binary rule then gives a zero-config default store (the repo's own `skills/`), and `git pull && go build` updates the linked binary in place. Clone users may run `skpp init` later only if they want to relocate the store.

### 12.2 `go install`

`go install github.com/dabstractor/skpp@latest` is a first-class install path: the binary is self-sufficient. It lands in `$(go env GOPATH)/bin`; on first use the user runs `skpp init` (§8.2), which creates the store and writes the config. **No clone required, no `SKPP_SKILLS_DIR` needed for normal use.** The earlier caveat ("must clone the repo and set the env var") is obsolete under the config model and is removed.

### 12.3 Releases (optional, phase 2)

If added: a GitHub Actions workflow that builds a `linux/amd64`, `linux/arm64`, `darwin/arm64`, `darwin/amd64` matrix and publishes via `gh release`. Out of scope for the initial one-shot unless trivial.

---

## 13. Acceptance criteria (the implementer must verify all pass)

From a clean clone at `~/projects/skpp`:

```bash
# Build
go build -o skpp . && echo OK
./skpp --version                      # prints: skpp <something>

# Discovery + path
test "$(./skpp --path)" = "$PWD/skills"   # sibling-of-binary rule
./skpp --list                          # shows the `example` skill
test -d "$(./skpp example)"            # resolves to a real dir
test -f "$(./skpp -f example)"         # -f prints the SKILL.md path

# Error contract: unknown tag prints nothing to stdout, exits 1
out=$(./skpp nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "unknown-tag contract OK"

# Absolute-path contract (default)
case "$(./skpp example)" in /*) echo "absolute OK";; *) echo "FAIL"; exit 1;; esac

# Validation
./skpp check                           # exits 0, reports the example as OK

# End-to-end with pi (skills loads ONLY via --skill, not auto-discovered)
pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
#   ↑ confirm pi's output references the example skill / does not error

# Symlink install works (resolve-back-to-repo)
ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp 2>/dev/null || mkdir -p /tmp/skpp-bin && ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp
/tmp/skpp-bin/skpp example             # still resolves to $PWD/skills/example
SKPP_SKILLS_DIR="$PWD/skills" ./skpp example   # env override works

# Config + first-run (§8)
mkdir -p /tmp/skpp-iso && cp ./skpp /tmp/skpp-iso/skpp && cd /tmp/skpp-iso
# unconfigured (clean HOME, no config, no skills sibling, no walk-up ancestor): hint + exit 1
env -u SKPP_SKILLS_DIR HOME=/tmp/skpp-iso/home XDG_CONFIG_HOME=/tmp/skpp-iso/home/.config \
  ./skpp x 2>err; rc=$?
[ "$rc" = 1 ] && grep -q 'run `skpp init`' err && echo "unconfigured-hint OK"
# non-interactive init creates the store + writes the config
SKPP_CONFIG=/tmp/skpp-iso/cfg.yaml ./skpp init --store /tmp/skpp-store
test -d /tmp/skpp-store                                                    # store created
grep -q 'store: /tmp/skpp-store' /tmp/skpp-iso/cfg.yaml                     # config written
# config rule wins; and env still beats config
SKPP_CONFIG=/tmp/skpp-iso/cfg.yaml ./skpp --path | grep -q /tmp/skpp-store
SKPP_SKILLS_DIR=/tmp/skpp-store SKPP_CONFIG=/tmp/skpp-iso/cfg.yaml ./skpp --path 2>&1 | grep -q SKPP_SKILLS_DIR
cd - >/dev/null
```

All of the above must pass. The pi line must show the skill loaded with **`--no-skills`** (proving we rely solely on the explicit `--skill` path, never on auto-discovery).

---

## 14. Shell completions

Ship completions for bash, zsh, fish (parity with mcpeepants). They complete:

- Subcommands/flags after `skpp ` / `skpp --`.
- **Tags** by invoking `skpp --all` (cheap, disk-derived) for positional completion.

Keep them simple: a function that runs `skpp --all` and offers the printed paths' basename-or-relTag. Provide an `install.sh` step (already in §12) OR a short note in README to source/copy the completion file. If time-boxed, completions are the **only** deliverable that may be deferred — flag it clearly in the PR if so.

---

## 15. README.md outline

Mirror the mcpeepants README's tone and structure:

1. **Title + one-liner:** "Standalone skill loader for pi — resolves a skill tag to an absolute path for `pi --skill`."
2. **Why:** centralized skills, **not** in any pi discovery location, loaded only on demand.
3. **Install:** `install.sh` (symlink) / `go install` (first-class) / from-source. First run: `skpp init` (prompts for the store dir, writes the config).
4. **Usage:** the canonical `pi --skill "$(skpp tag)"` example, multi-skill example, `-f`, `--list`, `--search`, `--all`, `check`, `--path`.
5. **Where skills live:** the `skills/` dir, the tag = relative dir path, the discovery rules (§7).
6. **Adding a skill:** drop a `<tag>/SKILL.md` under `skills/`; required frontmatter; run `skpp check`.
7. **How `skpp` finds the store:** §8 — `skpp init` writes a config pointing at the store; `SKPP_SKILLS_DIR` overrides it; sibling / walk-up are zero-config dev fallbacks.
8. **Constraints:** no catalog index (disk-discovered); a settings config file is fine; never auto-discovered by pi; loads only via `--skill`.

---

## 16. `.gitignore`

```
/skpp
/dist
*.test
*.out
.DS_Store
```

(`/skpp` ignores the locally-built binary; everything else is committed, including `skills/example/`.)

---

## 17. Constraints & guardrails (do NOT do these)

- ❌ Do **not** add a **catalog** index/manifest (e.g. `skills.json` enumerating skills). The catalog is always walked from disk. A **settings** file (store location, etc.) is expected and fine — see §8; the rule is only that catalog data already on disk is never duplicated into a sidecar.
- ❌ Do **not** place skills in any pi auto-discovery location. The store is loaded **only** via `--skill`.
- ❌ Do **not** make `skpp` install/copy skills into `~/.pi/...` or `~/.agents/...`. It only prints paths.
- ❌ Do **not** print anything to stdout on a failed/unknown tag resolution (breaks `pi --skill "$(skpp x)"`).
- ❌ Do **not** require Node, Python, or any runtime at *run* time (build-time `go` is fine).
- ❌ Do **not** ship more than the one example skill.

---

## 18. Suggested build order (for the one-shot pass)

1. `go.mod` + `internal/skillsdir` + `main.go --path` → prove location resolution (§8). **Hardest part; do first.**
2. `internal/discover` (walk + frontmatter parse) → `--list`.
3. `internal/resolve` → `skpp <tag>`, `-f`, `--all`, `--relative`.
4. `--search`, `check`.
5. `--help`/`--version`/error semantics + exit codes (§6.4).
6. Example skill + run §13 acceptance.
7. `install.sh` (symlink) + README + `.gitignore` + LICENSE.
8. Completions.

---

## 19. Decisions log (assumptions made in lieu of asking — override if you disagree)

| # | Decision | Default chosen | Rationale |
|---|---|---|---|
| 1 | Repo / binary name | `skpp` | The command as written in the request |
| 2 | Visibility | **public** | Matches mcpeepants + user's other repos |
| 3 | Language | **Go** | Static binary, instant startup, symlink-aware path resolution |
| 4 | Output unit | **directory** (default), `--file` for `SKILL.md` | `--skill <dir>` is supported & includes assets |
| 5 | Catalog index | **none** — walked from disk each call | Small, hand-edited catalog; an index would drift. *(Earlier this row claimed an "explicit user constraint" — that was a misattribution; the user did not request it. A **settings** config file is a separate concern and is now used — §8.)* |
| 6 | Canonical tag | relative dir path under `skills/`; basename/name/alias fallbacks | Inferable from disk; tolerant of common usage |
| 7 | Search metadata | `metadata.keywords`/`category`/`aliases` in frontmatter | Uses the spec's own optional `metadata` field |
| 8 | Frontmatter parser | `gopkg.in/yaml.v3` | Robust; only third-party dep |
| 9 | Install method | `go install` / release binary / `install.sh`; `skpp init` configures the store | No clone forced. The binary is self-sufficient; first run prompts for (or is told via flag/env) the store dir. Sibling / walk-up rules kept as zero-config dev fallbacks. |
| 10 | Shipped skills | exactly one `example` | Proves the pipeline; repo is a loader, not a library |
| 11 | License | MIT | Match mcpeepants conventions |
| 12 | Settings file | `$XDG_CONFIG_HOME/skpp/config.yaml`, key `store` (YAML; reuses `yaml.v3`); `SKPP_CONFIG` overrides the path | Fixed home so the binary can bootstrap without being told; YAML avoids a new dependency |
| 13 | First-run UX | `skpp init` prompts interactively; bare tags never prompt (any auto-prompt TTY-gated) | Protects the `pi --skill "$(skpp x)"` contract from hanging inside command substitution |
| 14 | Discovery order | env `SKPP_SKILLS_DIR` → config `store` → sibling of binary → walk-up → "run `skpp init`" | Env overrides config for CI/tests; heuristics kept as zero-config dev fallbacks |
