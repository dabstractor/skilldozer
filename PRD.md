# PRD — Skilldozer

> **Status:** Ready for one-shot implementation. This document is the complete specification.
> **Repo:** `dabstractor/skilldozer` (already created and cloned at `~/projects/skilldozer`).
> **Scope of THIS task for the implementer:** build the tool, the example skill, docs, install, and completions described below. Do not change the product contract without updating this PRD.

---

## 1. Goal

A tiny, fast CLI called **`skilldozer`** that resolves a human-friendly **skill tag** to the **absolute filesystem path** of a locally-stored [Agent Skill](https://agentskills.io/specification), so it can be loaded into **pi** on demand:

```bash
pi --skill "$(skilldozer my-skill-tag)" --skill "$(skilldozer my-other-skill-tag)"
```

`skilldozer` is to **skills** what `mcpeepants` (`get-server-config.sh`) is to **MCP server configs**: a centralized, on-disk catalog you address by tag, surfaced through a one-liner.

### Why it exists

pi can load skills from many "official" discovery locations (see §3). The user wants a **single centralized store that is deliberately NOT one of those locations**, loaded **only** via the explicit `--skill <path>` flag. `skilldozer` is the resolver that turns a tag into that path.

---

## 2. Hard constraints (non-negotiable)

1. **No catalog index — disk-discovered.** There is no `skills.json` / index file enumerating the skill *catalog*; the set of skills is always computed by walking the store on each call, so dropping in a directory with a `SKILL.md` makes it instantly available — no rebuild step, no index/disk drift to debug. This is specifically about the **catalog**. It does **not** prohibit a small **settings** file for things the filesystem cannot express — today, where the store lives (see §8). Catalog data already on disk is never duplicated into a sidecar; settings are not catalog data.
2. **No auto-discovery by pi.** Skills live in a location pi does **not** scan. They load **only** through `pi --skill "$(skilldozer <tag>)"`. The store must never be `~/.pi/agent/skills`, `~/.agents/skills`, a project `.pi/skills` or `.agents/skills`, a `node_modules` package, or anything with a `pi.skills` entry in `package.json`.
3. **`skilldozer <tag>` prints exactly one absolute path** (to stdout, trailing newline) for a resolved skill — the canonical contract. Unknown tag ⇒ **nothing on stdout**, error to stderr, exit 1.
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

**Decision:** `skilldozer` emits the skill **directory** path (not the `SKILL.md` file), because that's the natural unit (includes assets) and `--skill <dir>` is explicitly supported. A `--file` flag is provided for callers who want the `SKILL.md` path instead.

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
skilldozer/
├── PRD.md                  # THIS file (already exists)
├── README.md              # User docs (mirror mcpeepants style)
├── LICENSE                # MIT (match mcpeepants conventions)
├── go.mod                 # module github.com/dabstractor/skilldozer
├── go.sum
├── .gitignore             # /skilldozer (built binary), coverage, OS files
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
│   ├── skilldozer.bash
│   ├── _skilldozer              # zsh
│   └── skilldozer.fish
└── skills/
    └── example/           # the ONE shipped example skill
        └── SKILL.md
```

`go.mod` module path: `github.com/dabstractor/skilldozer`. Minimum Go: the latest two stable releases (set in `go.mod` `go` directive).

---

## 6. CLI contract (authoritative)

Binary name: **`skilldozer`**. **Every action that is not a skill tag is a `--flag`** — there are *no bare-word subcommands*, so the entire positional namespace is reserved for skill tags. This is what makes completions unambiguous: a bare `<tab>` always means "show me my skills" (§14). Flags use POSIX double-dash long form, plus single-dash short forms for the common ones. Unknown flags ⇒ error + exit 2.

> **Help & completions advertise long forms only.** `--help` lists each option as `--long`; the short aliases (`-a`, `-l`, `-s`, `-f`, `-p`, `-h`, `-v`) remain valid for fast typing but are deliberately **not** advertised in the usage text or offered by `--completions` (§14.1) — both surfaces are long-form-only, matching the double-dash convention and keeping the namespace story clean.

### 6.1 Flags & modes

| Invocation | Behavior | stdout | exit |
|---|---|---|---|
| `skilldozer <tag> [<tag>...]` | Resolve one or more tags to skill directory paths. | One **absolute** path per line, in input order. | `0` if all resolve; `1` if **any** fail (and **nothing** is printed) |
| `skilldozer --all` / `-a` | All skills, directory paths. | One absolute path per line (sorted by tag). | `0` |
| `skilldozer --list` / `-l` | Human-readable catalog. | Table: `TAG`, `NAME`, `DESCRIPTION` (wrapped). | `0` (`1` if no skills found) |
| `skilldozer --search <q>` / `-s <q>` | Substring (case-insensitive) search over tag, frontmatter `name`, `description`, and `metadata.keywords`. | Same table format as `--list`, filtered. | `0`; `1` if no matches |
| `skilldozer --check` | Validate every skill on disk (see §9). | Report: `OK` lines + any `WARN`/`ERROR` lines. | `0` if clean; `1` if any ERROR |
| `skilldozer --init [<dir>]` | First-run setup (see §8.2): prompt for the skills store dir, create it if missing, write the config, seed a template if empty, validate. Non-interactive: `skilldozer --init <dir>` or `skilldozer --init --store <dir>`. | The configured store path. | `0` on success; `1` on error/cancel |
| `skilldozer --link <dir> [<dir>...]` | Link one or more external skill directories into the store (see §8.4): pass `--link` **once**, then every following positional is a directory to link. For each, absolutize, validate it is a directory containing at least one `SKILL.md`, then create a symlink `<store>/<basename>` → `<dir>`. An existing symlink at that name is refreshed; a non-symlink (real file/dir) is refused. | One absolute link path per successfully linked directory, in input order, one per line. | `0` if all link; `1` if any fail (successful links are still created — linking is idempotent); `2` if no directory follows `--link` |
| `skilldozer --completions [--shell <name>]` | Emit the completion script for the current (or named) shell to **stdout**, for `eval "$(skilldozer --completions)"`. See §14.6. | The shell's completion script (one text blob). | `0`; `1` if shell undetectable; `2` if `--shell` value unsupported |
| `skilldozer --path` / `-p` | Where is `skilldozer` looking? | Absolute path of the resolved skills dir. | `0` (`1` if unresolvable) |
| `skilldozer --help` / `-h` | Usage. | Help text (to stdout). | `0` |
| `skilldozer --version` / `-v` | Version. | `skilldozer <version>` (single line). | `0` |

### 6.2 Modifiers (combine with tag resolution or `--all`)

| Flag | Effect |
|---|---|
| `--file` / `-f` | Print the `SKILL.md` file path instead of the directory path. E.g. `skilldozer -f example`. |
| `--no-color` | Disable ANSI color even on a TTY. |
| `--relative` | Print paths relative to the skills dir instead of absolute (machine-local convenience; default is absolute). |

### 6.3 Default behavior

- **No arguments and no flag** ⇒ print usage to **stdout**, exit `0`. Bare invocation is **implicit `--help`** (skilldozer has no default action), chosen so `skilldozer | grep …` works — the help must land on the piped stream to be grep-friendly. (`skilldozer --help` / `-h` likewise prints usage to stdout, exit 0.) *(Previously stderr/exit-1 “parity with `get-server-config.sh`”; overridden — §19, decision 17.)*
- `--help` / `--version` take precedence over everything else.
- The **mode flags** (`--check`, `--init`, `--link`, `--list`, `--search`, `--all`, `--completions`, `--path`) are mutually exclusive with each other. `--init` accepts an optional positional `<dir>` (the store to adopt); `--link` accepts **one or more** positional directories (see §8.4). All *other* positionals are skill tags, and mixing a `<tag>` with any non-`--link` mode flag is an error (exit 2). (`--link` is the sole mode that **collects** trailing positionals: once `--link` is parsed, every following non-flag token is a directory to link — never a tag — so the bare-positional namespace stays reserved for tags. Pass `--link` at most once.)
- **There are no bare-word subcommands.** Because everything that is not a tag is a `--flag`, a skill literally tagged `check`, `init`, or `completions` resolves normally — `skilldozer check` resolves the *tag* `check`, `skilldozer --check` runs validation. This is the namespace-safety guarantee that lets a bare `<tab>` always mean "skills" (§14).

### 6.4 Error semantics (critical for `$(...)` use)

> Bare no-args is **not** an error — it is implicit help (stdout, exit 0; see §6.3). The stderr / non-zero contract below applies to **genuine failures only**. That stream separation is exactly what lets `skilldozer | grep …` work: the help is on stdout, the failures stay on stderr.

- **Any** unresolved/ambiguous tag in a `skilldozer <tag>...` invocation ⇒ print **one** error line per problem tag to stderr, print **nothing** to stdout, exit `1`. This guarantees `pi --skill "$(skilldozer badtag)"` fails loudly rather than passing a garbage path.
- Ambiguous tag (a short name matching >1 skill) ⇒ stderr lists the candidate full tags, exit `1`.
- Skills dir cannot be located / skilldozer is unconfigured ⇒ stderr: `skilldozer is not configured; run \`skilldozer --init\`` (or, if configured but the dir vanished, the concise reason + fix), exit `1`. Bare tag resolution **never** prompts (see §8.2), so `pi --skill "$(skilldozer x)"` fails loudly instead of hanging inside command substitution.
- `skilldozer --completions` cannot determine the shell (no `--shell`, no `$SKILLDOZER_SHELL`, no usable `$SHELL`) ⇒ stderr: `could not detect shell; pass --shell {bash\|zsh\|fish}`, exit `1`. An unsupported `--shell <name>` value (not bash/zsh/fish) ⇒ stderr error, exit `2`. On success the script goes to **stdout** (for `eval`); nothing else.
- `skilldozer --link <dir> [<dir>...]` (§8.4): each directory is validated independently — not a directory, the store itself/inside it, or contains no `SKILL.md` ⇒ a stderr line naming that directory; a name collision at `<store>/<name>` with a non-symlink (real file or directory) is refused ⇒ a stderr line naming that directory. Directories are processed in order; each success prints its link path to stdout and each failure prints a stderr line, so a batch can yield **mixed** output (some paths on stdout, some errors on stderr). Exit `1` if **any** directory fails (successful links remain — linking is idempotent, so re-running `--link <failed-dir>` is the fix); the single-directory case reduces to the original guarantee (one bad dir ⇒ nothing on stdout, exit `1`). `--link` presented with **no** following directory at all ⇒ `skilldozer: --link requires at least one path to a skill directory`, exit `2` (nothing on stdout).

---

## 7. Skill discovery & tag resolution

### 7.1 Discovery

1. Locate the skills dir (§8).
2. Walk it recursively. A **skill** = any directory that directly contains a `SKILL.md`. (Nested skills count: `skills/writing/reddit/SKILL.md` is a skill.) Symlinked directories **are followed** — a skill reachable only through a symlink is discovered and reported under its on-disk link name (not the resolved target). Cycles are broken via a visited set of resolved (canonical) paths, so a link that points back at an ancestor terminates the walk instead of looping forever.
3. For each skill, parse frontmatter (§7.3) and capture:
   - `dir` — absolute path of the skill directory.
   - `relTag` — path of the skill dir **relative to** the skills dir, with OS separators normalized to `/` (e.g. `writing/reddit`). **This is the canonical tag.**
   - `name` — frontmatter `name` (may differ from dir).
   - `description` — frontmatter `description`.
   - `keywords` — `metadata.keywords` (list) if present, else `[]`.
   - `category` — `metadata.category` if present.
   - `aliases` — `metadata.aliases` (list) if present, else `[]`.

> Because everything is read from disk, there is **no index file**. `skilldozer` rebuilds the index on every invocation (fast: it's a directory walk of a small tree).

### 7.2 Tag resolution precedence (first match wins; later steps only consulted if earlier produced nothing)

Given an input `tag`:

1. **Exact canonical tag** — equals some skill's `relTag` (case-sensitive). Direct hit ⇒ return it.
2. **Basename** — equals the final `/`-component of some skill's `relTag` (e.g. `reddit` matches `writing/reddit`). If **>1** skill matches ⇒ ambiguous error.
3. **Frontmatter `name`** — equals some skill's `name`. If **>1** ⇒ ambiguous error.
4. **Declared alias** — appears in some skill's `metadata.aliases`. If **>1** ⇒ ambiguous error.
5. Nothing ⇒ unknown-tag error.

Examples (assume skills `skills/foo/SKILL.md` with `name: foo-helper`, and `skills/writing/reddit/SKILL.md`):

- `skilldozer foo` → `…/skills/foo`
- `skilldozer writing/reddit` → `…/skills/writing/reddit`
- `skilldozer reddit` → `…/skills/writing/reddit` (basename, unambiguous)
- `skilldozer foo-helper` → `…/skills/foo` (by `name`)

### 7.3 Frontmatter parsing

- Slice the text between the first two lines that are exactly `---` at the start of `SKILL.md`. If no frontmatter block ⇒ skill still resolves **by directory** (tag/basename) but `--check` flags it and `--list` shows `description` as `(missing)`.
- Parse with `gopkg.in/yaml.v3` (robust, handles quoted/multiline scalars). This is the **only** third-party dependency. (A hand-rolled parser is acceptable if it correctly handles quoted values and the `metadata` map; prefer `yaml.v3`.)
- Be lenient: unknown frontmatter keys are ignored (matches pi behavior). Missing optional keys ⇒ defaults.

---

## 8. Locating the skills directory

`skilldozer` does not assume the store lives next to the binary or inside a checkout. A small settings file records where the user keeps their skills, written by `skilldozer --init` on first use. The store can live anywhere.

### 8.1 Configuration file

- Default location: `$XDG_CONFIG_HOME/skilldozer/config.yaml` (→ `~/.config/skilldozer/config.yaml`). Override the file path with `SKILLDOZER_CONFIG=<file>` (useful for tests / multiple profiles).
- This is the **one** fixed, well-known path the binary can bootstrap from; it must not depend on the store location (chicken-and-egg: you cannot discover the config from the dir the config points at).
- Format: YAML (reuses the existing `yaml.v3` dependency). Minimal valid file:

  ```yaml
  store: /home/dustin/skills
  ```

- Unknown keys are ignored (room to grow: default category, color prefs, etc.). A missing or unreadable config is treated as "not yet configured" and falls through to §8.3 rules 3-5 — never a hard error.

### 8.2 First-run setup — `skilldozer --init`

`skilldozer --init` is the documented first command and the **only** place skilldozer prompts interactively.

Interactive (TTY) flow:

1. **Auto-detect from cwd first:** if the current working directory already looks like a store — it contains at least one `SKILL.md` at any depth (the store definition, §7.1) — the default store is **cwd** ("detected skills in <cwd>"). Otherwise the default is `$XDG_DATA_HOME/skilldozer/skills` (→ `~/.local/share/skilldozer/skills`). Then prompt: "Where should skilldozer keep your skills? [<default>]" — Enter accepts the default, typing a path overrides.
2. `mkdir -p` the chosen dir if it does not exist.
3. If the dir is empty, seed `example/SKILL.md` as a copy-paste template (a string constant compiled into the binary — **not** `go:embed` of a directory; nothing about the user's collection is compiled in). If the dir already contains skills, adopt it in place; never clobber or delete.
4. Write `config.yaml` (at `$SKILLDOZER_CONFIG` or the default location) with the absolute `store` path.
5. Print the output of `skilldozer --path` (which rule won) and `skilldozer --check`.

Non-interactive: `skilldozer --init <dir>` or `skilldozer --init --store <dir>` (for scripts / CI). With no `<dir>`/`--store`, the same cwd-auto-detect applies — run from a skill-containing dir and it adopts that dir as the store with no prompt; run from elsewhere and it uses the XDG default. `SKILLDOZER_SKILLS_DIR` set at runtime still bypasses the config entirely.

**Prompt safety (load-bearing):** the bare `skilldozer <tag>` path **never** prompts. If unconfigured (every rule in §8.3 misses), it writes to stderr exactly `skilldozer is not configured; run \`skilldozer --init\``, exits `1`, and writes **nothing** to stdout — so `pi --skill "$(skilldozer x)"` fails loudly instead of hanging inside command substitution. Any convenience auto-prompt anywhere else must be gated on `isatty(stdin)`.

### 8.3 Resolution priority (first hit wins)

1. **`SKILLDOZER_SKILLS_DIR` env var** — override; if set and an existing dir, use it. Lets CI / tests / temporary redirects win without editing the config.
2. **Config file `store`** (§8.1) — the primary, set by `skilldozer --init`.
3. **Sibling of the running binary** (symlink-aware: `os.Executable()` + `filepath.EvalSymlinks()`) — still lets a clone-and-build dev workflow work with zero config.
4. **Walk up from `cwd`** — for `go run` / dev.
5. **None** ⇒ unconfigured: stderr one-line fix (`run \`skilldozer --init\``), exit `1`.

`skilldozer --path` reports which rule won, on stderr, with one of the labels: `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`. This matters because a bad `SKILLDOZER_SKILLS_DIR` value is silently ignored and falls through — `--path` is the only way to tell which directory actually won. This remains the single most failure-prone area — implement and test it first (see §13 acceptance).

### 8.4 Linking an external skill — `skilldozer --link <dir> [<dir>...]`

`--link` makes one or more external skill directories available in the store **without copying them** — the `npm link` / `pip install -e` idiom for skills, **batched**. It relies on discovery following symlinks (§7.1: "a skill reachable only through a symlink is discovered and reported under its on-disk link name"), so a linked skill resolves by its link name exactly like a real one.

**Batching (headline behavior).** Pass `--link` **once**; every positional token after it is a directory to link. There is no need to repeat `--link`:

```bash
skilldozer --link ~/projects/agent-browser ~/projects/agent-builder ~/projects/mdsel
```

links all three in one invocation, printing one absolute link path per success (in input order).

Behavior:

1. Resolve the store via §8.3 (`skillsdir.Find()`). Unconfigured ⇒ stderr hint, exit `1`.
2. Collect the link targets: `--link` consumes **every** following positional as a directory (in input order). The `=`-form `--link=<dir>` supplies the first directory; further positionals add to it. **Zero** following directories ⇒ `skilldozer: --link requires at least one path to a skill directory`, exit `2` (nothing on stdout). Pass `--link` at most once.
3. **For each directory, in input order**, absolutize and link:
   1. Absolutize `<dir>` (expand `~`/`~/`, then `filepath.Abs`) so the symlink target is stable regardless of cwd when discovery later follows it.
   2. Validate `<dir>`:
      - Must be an existing directory (else stderr message naming the directory).
      - Must not be the store itself or live inside it — linking the store into itself is a no-op/footgun (else stderr message naming the directory).
      - Must contain at least one `SKILL.md` at any depth (`HasSkillMD`) — a single skill OR a directory of skills (else stderr message naming the directory).
   3. Link name = `filepath.Base(<dir>)`; link path = `<store>/<name>`.
   4. Conflict handling at the link path:
      - Nothing exists ⇒ create the symlink.
      - An existing **symlink** ⇒ **refresh** (remove + recreate), matching the `install.sh` precedent (§12.1 step 5: "If a symlink already exists, refresh it"). A symlink is a pointer the user manages; re-pointing it is not data loss.
      - A non-symlink (regular file or real directory) ⇒ **refuse** with a clear message naming the directory; the user must remove it manually. This protects real data (the §17 "never clobber" spirit, applied to `--link`).
   5. On success: print the link path (`<store>/<name>`) to stdout; print `Linked <linkPath> -> <absTarget>` plus which §8.3 store rule won to stderr.
4. **Exit code and partial success.** Processing is **not** atomic: a directory that validates is linked before the next one is even checked, so a batch can leave some symlinks created and others not. Each success prints one path to stdout; each failure prints one line to stderr. Exit `0` only if **every** directory linked; exit `1` if **any** failed (the successful links remain in place — linking is idempotent, so re-running `--link <only-the-failed-dir>` is the documented fix). This mirrors the `ln`/`git add` multi-target convention and reduces to the original single-directory guarantee as the degenerate n=1 batch (one bad dir ⇒ nothing on stdout, exit `1`).

`--link` is a mode (mutually exclusive with all other modes, §6.3). It is the sole mode that **collects** trailing positionals: once `--link` is parsed, every following non-flag token is a directory to link (never a tag), so the bare-positional namespace stays reserved for tags (§6.1). `--link=<dir>` is the `=`-form (supplies the first directory); `--link` with no following directory ⇒ exit `2`. No short form is advertised (matching `--store`/`--init`/`--check`/`--completions`).

Examples:

```bash
# Link a single skill (no copy, single source of truth)
skilldozer --link ~/projects/agent-browser
skilldozer agent-browser          # now resolves via the symlink
pi --skill "$(skilldozer agent-browser)"

# Link several at once — one --link, then the directories
skilldozer --link ~/projects/agent-browser ~/projects/agent-builder ~/projects/mdsel
# stdout: one absolute link path per success, in input order
```

---

## 9. Validation — `skilldozer --check`

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

**`SKILL.md` frontmatter** — required fields per the Agent Skills standard, plus **skilldozer conventions** stored under the standard `metadata` map (so nothing is non-standard):

````markdown
---
name: my-skill-tag
description: >
  One to two sentences: what this skill does and precisely when to use it.
  This field drives pi's on-demand loading AND skilldozer's --search.
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
- `metadata.keywords` / `metadata.category` / `metadata.aliases` are **optional** and exist only to enrich `skilldozer --search` and tag aliases. They are standard-compliant (`metadata` is a spec'd optional field).
- All asset/script references inside the body use **paths relative to the skill directory** (pi resolves them against the dir we hand to `--skill`).

---

## 11. The one shipped example skill

Ship **exactly one** example so `--list`/resolution are demonstrable out of the box:

`skills/example/SKILL.md`:
````markdown
---
name: example
description: >
  Reference example skill for skilldozer. Demonstrates the required frontmatter and
  how skilldozer resolves a tag to an absolute path. Safe to delete once you add real skills.
metadata:
  keywords: [example, demo, skilldozer]
  category: meta
---

# Example Skill

This skill exists only so `skilldozer` has something to resolve.

Try:

```bash
skilldozer example                       # prints this directory's absolute path
skilldozer -f example                    # prints .../skills/example/SKILL.md
pi --skill "$(skilldozer example)"       # loads this skill into pi
```
````

No other skills ship in this repo.

---

## 12. Installation

### 12.1 `install.sh` (mirrors mcpeepants `QUICK_INSTALL.sh` spirit)

Behavior:

1. `cd` to the script's own directory (the repo root).
2. Verify `go` is on `PATH`; if not, print install instructions and exit `1`.
3. `go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skilldozer .`
4. Pick a target bin dir in this order: `$SKILLDOZER_INSTALL_BIN` (if set) → `$HOME/.local/bin` (if present or creatable) → `/usr/local/bin` (if writable, else needs `sudo`).
5. **Symlink** (not copy) `<target>/skilldozer` → `<repo>/skilldozer`, so `os.Executable()` resolves back to the repo and finds `skills/`. If a symlink already exists, refresh it.
6. Ensure the target dir is on `PATH`; if not, print the exact `export PATH=…` line for the detected shell (`~/.bashrc` / `~/.zshrc` / `~/.config/fish/config.fish`).
7. Print a verification command: `skilldozer example`.

> **Why symlink, not copy:** with `skilldozer --init` (§8) either works — copy is no longer fatal, because the store no longer has to be the binary's sibling. Symlink is still recommended for clone users: the sibling-of-binary rule then gives a zero-config default store (the repo's own `skills/`), and `git pull && go build` updates the linked binary in place. Clone users may run `skilldozer --init` later only if they want to relocate the store.

### 12.2 `go install`

`go install github.com/dabstractor/skilldozer@latest` is a first-class install path: the binary is self-sufficient. It lands in `$(go env GOPATH)/bin`; on first use the user runs `skilldozer --init` (§8.2), which creates the store and writes the config. **No clone required, no `SKILLDOZER_SKILLS_DIR` needed for normal use.** The earlier caveat ("must clone the repo and set the env var") is obsolete under the config model and is removed.

### 12.3 Releases (optional, phase 2)

If added: a GitHub Actions workflow that builds a `linux/amd64`, `linux/arm64`, `darwin/arm64`, `darwin/amd64` matrix and publishes via `gh release`. Out of scope for the initial one-shot unless trivial.

---

## 13. Acceptance criteria (the implementer must verify all pass)

From a clean clone at `~/projects/skilldozer`:

```bash
# Build
go build -o skilldozer . && echo OK
./skilldozer --version                      # prints: skilldozer <something>

# Discovery + path
test "$(./skilldozer --path)" = "$PWD/skills"   # sibling-of-binary rule
./skilldozer --list                          # shows the `example` skill
test -d "$(./skilldozer example)"            # resolves to a real dir
test -f "$(./skilldozer -f example)"         # -f prints the SKILL.md path

# Error contract: unknown tag prints nothing to stdout, exits 1
out=$(./skilldozer nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "unknown-tag contract OK"

# Absolute-path contract (default)
case "$(./skilldozer example)" in /*) echo "absolute OK";; *) echo "FAIL"; exit 1;; esac

# Grepability contract (§6.3): no-args help is on stdout, exit 0 — pipes MUST see it
out=$(./skilldozer 2>/dev/null); rc=$?
[ "$rc" = "0" ] && printf '%s' "$out" | grep -qi 'USAGE' && echo "no-args-help-on-stdout OK" || { echo "FAIL"; exit 1; }
test -z "$(./skilldozer 2>&1 >/dev/null)"   # no-args writes NOTHING to stderr

# `--completions` flag (§14.6): emits the matching script to stdout; --shell forces one
./skilldozer --completions --shell bash 2>/dev/null | grep -q '_skilldozer_completion' && echo "completion-bash OK" || { echo "FAIL"; exit 1; }
./skilldozer --completions --shell fish 2>/dev/null | grep -q 'complete -c skilldozer' && echo "completion-fish OK" || { echo "FAIL"; exit 1; }
# detection failure (no --shell, no $SHELL) ⇒ stderr + exit 1, nothing on stdout
out=$(env -u SHELL -u SKILLDOZER_SHELL ./skilldozer --completions 2>cerr); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && grep -qi 'shell' cerr && echo "completion-no-shell OK" || { echo "FAIL"; exit 1; }
# unsupported --shell value ⇒ exit 2
./skilldozer --completions --shell tcsh >/dev/null 2>&1; [ "$?" = "2" ] && echo "completion-bad-shell OK" || { echo "FAIL"; exit 1; }

# Validation
./skilldozer --check                         # exits 0, reports the example as OK

# End-to-end with pi (skills loads ONLY via --skill, not auto-discovered)
pi --no-skills --skill "$(./skilldozer example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
#   ↑ confirm pi's output references the example skill / does not error

# Symlink install works (resolve-back-to-repo)
ln -sf "$PWD/skilldozer" /tmp/skilldozer-bin/skilldozer 2>/dev/null || mkdir -p /tmp/skilldozer-bin && ln -sf "$PWD/skilldozer" /tmp/skilldozer-bin/skilldozer
/tmp/skilldozer-bin/skilldozer example             # still resolves to $PWD/skills/example
SKILLDOZER_SKILLS_DIR="$PWD/skills" ./skilldozer example   # env override works

# Config + first-run (§8)
mkdir -p /tmp/skilldozer-iso && cp ./skilldozer /tmp/skilldozer-iso/skilldozer && cd /tmp/skilldozer-iso
# unconfigured (clean HOME, no config, no skills sibling, no walk-up ancestor): hint + exit 1
env -u SKILLDOZER_SKILLS_DIR HOME=/tmp/skilldozer-iso/home XDG_CONFIG_HOME=/tmp/skilldozer-iso/home/.config \
  ./skilldozer x 2>err; rc=$?
[ "$rc" = 1 ] && grep -q 'run `skilldozer --init`' err && echo "unconfigured-hint OK"
# non-interactive init creates the store + writes the config
SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer --init --store /tmp/skilldozer-store
test -d /tmp/skilldozer-store                                                    # store created
grep -q 'store: /tmp/skilldozer-store' /tmp/skilldozer-iso/cfg.yaml                     # config written
# config rule wins; and env still beats config
SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer --path | grep -q /tmp/skilldozer-store
SKILLDOZER_SKILLS_DIR=/tmp/skilldozer-store SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR
cd - >/dev/null

# --link: symlink an external skill dir into the store (§8.4)
rm -rf /tmp/sd-link && mkdir -p /tmp/sd-link/store /tmp/sd-link/src/linked
printf -- '---\nname: linked\ndescription: A linked skill.\n---\n# body\n' > /tmp/sd-link/src/linked/SKILL.md
test "$(SKILLDOZER_SKILLS_DIR=/tmp/sd-link/store ./skilldozer --link /tmp/sd-link/src/linked)" = "/tmp/sd-link/store/linked" && echo "link OK"
test -L /tmp/sd-link/store/linked                                                              # symlink created
SKILLDOZER_SKILLS_DIR=/tmp/sd-link/store ./skilldozer linked >/dev/null && echo "resolve-linked OK"   # resolves via the symlink
SKILLDOZER_SKILLS_DIR=/tmp/sd-link/store ./skilldozer --link /tmp/sd-link/src/linked >/dev/null && echo "link-refresh OK"   # re-link refreshes the symlink
# multi-link: one --link, then several directories (§8.4)
rm -rf /tmp/sd-link/store && mkdir -p /tmp/sd-link/store /tmp/sd-link/src/other /tmp/sd-link/notaskill
printf -- '---\nname: other\ndescription: Another linked skill.\n---\n# body\n' > /tmp/sd-link/src/other/SKILL.md
out=$(SKILLDOZER_SKILLS_DIR=/tmp/sd-link/store ./skilldozer --link /tmp/sd-link/src/linked /tmp/sd-link/src/other)
printf '%s\n' "$out" | grep -qx '/tmp/sd-link/store/linked' && printf '%s\n' "$out" | grep -qx '/tmp/sd-link/store/other' && echo "multi-link OK"   # both link paths, input order
test -L /tmp/sd-link/store/linked && test -L /tmp/sd-link/store/other                            # both symlinks created
# mixed batch: two valid + one invalid → valid ones link, exit 1, the bad dir named on stderr
out=$(SKILLDOZER_SKILLS_DIR=/tmp/sd-link/store ./skilldozer --link /tmp/sd-link/src/linked /tmp/sd-link/src/other /tmp/sd-link/notaskill 2>err); rc=$?
[ "$rc" = "1" ] && printf '%s\n' "$out" | grep -qx '/tmp/sd-link/store/linked' && grep -q 'notaskill' err && echo "multi-link-partial OK"
# single bad dir (degenerate batch): nothing on stdout, exit 1
out=$(SKILLDOZER_SKILLS_DIR=/tmp/sd-link/store ./skilldozer --link /tmp/sd-link/notaskill 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "link-non-skill-refused OK"
./skilldozer --link >/dev/null 2>&1; [ "$?" = "2" ] && echo "link-missing-value OK"               # no dir at all → exit 2
```

All of the above must pass. The pi line must show the skill loaded with **`--no-skills`** (proving we rely solely on the explicit `--skill` path, never on auto-discovery).

---

## 14. Shell completions

Ship **dynamic** completions for bash, zsh, and fish, in the `completions/` directory (§5):

| Shell | File |
|---|---|
| bash | `completions/skilldozer.bash` |
| zsh  | `completions/_skilldozer` |
| fish | `completions/skilldozer.fish` |

"Dynamic" is load-bearing: the tag list is **never** a frozen static list, because the store is manifest-free (§2, constraint 1: the catalog is always walked from disk). Completions therefore derive the tag set fresh, on every keystroke, from disk.

> **Design priority (repeated, because it's the point):** a bare `skilldozer <tab>` shows the user's **skills** — never the help menu, never a command list. That is only possible because §6.1 reserves the bare positional namespace for skill tags. The detailed behavior matrix lives in §14.1; the `--completions` sub-command that installs all of this is §14.6.

### 14.1 What they complete — **skills first**

The **headline behavior** is: a bare `<tab>` shows the user's installed **skills**, never the help text and never a command list. This is the highest-priority requirement for the completion scripts and flows directly from §6.1/§6.3: because *everything that is not a tag is a `--flag`*, the entire positional namespace belongs to skill tags. So the completion rules are simply:

| What the user types | Completion offers | Source / rule |
|---|---|---|
| `skilldozer <tab>` | **All skill tags** | `skilldozer --relative --all` (verbatim stdout) |
| `skilldozer a<tab>` | Skill tags beginning with `a` | same list, **prefix-filtered by the shell** |
| `skilldozer writing/<tab>` | Skill tags under `writing/` | same list, path-prefix-filtered |
| `skilldozer -<tab>` or `--<tab>` | **Long-form flags only** | static set (§6.1/§6.2) |
| `skilldozer --c<tab>` | `--check`, `--completions` | flags filtered to the `--c` prefix |
| `skilldozer --init <tab>` | Directories (file completion) | shell-native, like `--store` |
| `skilldozer --link <tab>` | Directories (the first skill dir to link in) | shell-native, like `--store`/`--init` |
| `skilldozer --link d1 <tab>` | Directories (a further skill dir — `--link` takes many) | shell-native; every position after `--link` completes dirs (§8.4) |
| `skilldozer --store <tab>` | Directories | shell-native |
| `skilldozer --search <tab>` | Nothing | (free-text query) |

Rules:

1. **Positional args are skills, always.** The first and every subsequent positional completes to skill tags from `skilldozer --relative --all`. A bare `<tab>` with nothing typed yields the full skill list. **No caching:** a newly-dropped skill directory is completable on the very next keystroke.
2. **Prefix filtering is the shell's native job.** The completion functions *offer* the full tag list (or the full flag list) and let the shell narrow it to whatever matches the token being typed — so `skilldozer a<tab>` shows only `a…` skills and `skilldozer --c<tab>` shows only `--c…` flags, with zero special-casing in the script. Filtering narrows the set; §14.7 additionally requires the shell to **show** that set (list every prefix match on the first Tab — never a silent halt at the common prefix).
3. **Flags complete to long form only.** `skilldozer -<tab>` (a single dash) and `skilldozer --<tab>` both offer the `--long` options and **never** the short aliases (`-a`, `-l`, …). Short forms stay valid at runtime (§6.1) for typing; they are simply not advertised by completion, mirroring the long-form-only help menu.
4. **Skills, not commands, on a bare `<tab>`.** Because the only things without a `-` prefix are skill tags, a bare `<tab>` shows skills — never the help menu, never a command/subcommand list. Commands appear only after the user types `-` (see rule 3).
5. **Modes own their positionals.** When `--init` is present, its optional `<dir>` completes to directories (like `--store`), not skills; when `--link` is present, **every** following positional completes to directories (it links many, §8.4); when any other mode flag is present, no positional is expected.
6. **Nothing else.** skilldozer takes tags/flags, never file paths, so each file sets a global no-file-completion rule (e.g. fish `complete -c skilldozer -f`) except where a flag explicitly wants a directory (`--init <dir>`, `--store <dir>`).

### 14.2 Value-taking flag handling

The value-taking flags are treated as **intentional inverses**:

- `--search`/`-s` — free-text query → offer **nothing** (offering tags here would be wrong).
- `--store <dir>` (§8.2) — a directory value → offer **file/dir completion**.
- `--init [<dir>]` (§8.2) — optional directory → offer **file/dir completion** (the store to adopt).
- `--link <dir> [<dir>...]` (§8.4) — one or more directory values → offer **file/dir completion** at every position after `--link` (the skill dirs to link in; pass `--link` once).
- `--completions --shell <name>` — `--shell` takes a fixed enum (`bash`/`zsh`/`fish`); offer those three words, nothing else.

### 14.3 Robustness

The `skilldozer --relative --all` probe must swallow its own errors (`2>/dev/null`): a missing or broken binary yields an **empty** tag list on the completion path, never a stderr dump that corrupts the shell completion UI.

### 14.4 Lockstep invariant (guardrail)

There is no shared source of truth the shells can import. The flag set in each completion file is **frozen to `main.go parseArgs()`**: if a future task adds, removes, or renames a flag there, update **all three** files identically. Carry this forward as a §17 guardrail.

### 14.5 Installation

`install.sh` (§12.1) deliberately does **not** install completions — they are a separate, shell-specific concern and the binary must not write outside its target bin dir. Users source/copy the file they want.

**bash** (one of):
```bash
source /path/to/skilldozer/completions/skilldozer.bash
cp completions/skilldozer.bash ~/.local/share/bash-completion/completions/skilldozer
cp completions/skilldozer.bash /etc/bash_completion.d/skilldozer
```

**zsh** (one of):
```bash
cp completions/_skilldozer ~/.zsh/completions/_skilldozer
cp completions/_skilldozer /usr/local/share/zsh/site-functions/_skilldozer
```
then ensure this is in `~/.zshrc`:
```bash
autoload -U compinit && compinit
```

**fish**:
```bash
cp completions/skilldozer.fish ~/.config/fish/completions/skilldozer.fish
```

> Completions are **shipped** (no longer deferrable). The earlier "may be deferred if time-boxed" caveat is obsolete.

### 14.6 `--completions` — the completion sub-command (load into your shell)

`skilldozer --completions` emits, to **stdout**, the completion script for the target shell. It is **designed to be run once from your shell's rc file** via the standard `eval`/`source` idiom — the same pattern as `zoxide init`, `starship init`, and `direnv hook`:

```bash
# bash / zsh — in ~/.bashrc / ~/.zshrc
eval "$(skilldozer --completions)"

# fish — in ~/.config/fish/config.fish
skilldozer --completions --shell fish | source
```

From then on, in every new shell, `skilldozer <tab>` is alive. The highest-priority behavior it delivers: **a bare `<tab>` lists your skills.**

#### What you get (behavior contract)

| Keystroke | Result |
|---|---|
| `skilldozer <tab>` | The full list of installed skill tags (the default, most-used action). |
| `skilldozer a<tab>` | Only skills whose tag starts with `a`. |
| `skilldozer writing/<tab>` | Only skills nested under `writing/`. |
| `skilldozer -<tab>` | The **long-form flags** (`--all`, `--check`, `--completions`, `--file`, `--help`, `--init`, `--link`, `--list`, `--no-color`, `--path`, `--relative`, `--search`, `--store`, `--version`), narrowed by what you type after the dash. |
| `skilldozer --c<tab>` | `--check`, `--completions`. |
| `skilldozer --init <tab>` | Directories (you're picking the store to adopt). |
| `skilldozer --store <tab>` | Directories. |
| `skilldozer --link <tab>` | Directories (the first skill dir to link in). |
| `skilldozer --link d1 <tab>` | Directories — `--link` takes many; every position after it completes dirs (§8.4). |
| `skilldozer --search <tab>` | Nothing — type your query freely. |

Key points:

- **Skills are the default completion target.** Nothing about the help menu or the command list appears when you press `<tab>` on a bare `skilldozer ` — you get skills, because positionals are *only* ever skill tags (§6.1). This is the whole reason `check`/`init`/`completions` were promoted to `--flags`: it keeps the bare-word namespace clean so `<tab>` is unambiguous.
- **Flags appear only after a dash.** `skilldozer -<tab>` (single dash) shows the long-form options; the shell then filters them as you keep typing. Short forms are intentionally absent from this menu (long-form-only, matching `--help`).
- **Always fresh.** The tag list is recomputed on every completion from `skilldozer --relative --all`; drop in a new skill dir and it is completable immediately — no rebuild, no cache.

#### Shell detection (first wins)

1. `--shell <bash|zsh|fish>` — explicit; required for deterministic `eval`.
2. `$SKILLDOZER_SHELL` — env override.
3. `basename("$SHELL")` — the login shell (correct in the common case).
4. None ⇒ stderr message + exit `1` (§6.4).

> **Why emit + `eval`, not "install":** a child process **cannot** register completions in the parent shell that invoked it — only the shell itself can define its completion functions, by eval'ing/sourcing the script **in its own process**. So `--completions` *emits* the script (to stdout, for the parent to `eval`); it writes **no files** and edits **no rc files**. This keeps the binary side-effect-free — fully consistent with §14.5. (A heavier `--install` mode that appends to rc files was considered and deferred; it would revisit §14.5's "binary writes nothing" stance.)

#### Embedding (self-sufficient binary)

The three scripts in `completions/` are compiled into the binary with `//go:embed` (stdlib, **no new dependency**). This makes `--completions` work for `go install` users with **no repo clone** — consistent with the "binary is self-sufficient" decision (§12.2 / decision 9). The on-disk `completions/` files remain the **single source of truth**; `completionScript` returns them verbatim (locked by `TestEmbeddedCompletionsMatchOnDisk`).

> **bash/fish vs zsh emit path.** bash and fish are emitted **verbatim** — §14.5's manual source/copy path and this flag produce identical bytes. zsh is **derived**: `runCompletion` passes the embedded autoload file through `zshEvalScript`, which (a) strips the trailing `_skilldozer "$@"` self-call and (b) appends an explicit `compdef _skilldozer skilldozer` registration plus a guarded `compinit` bootstrap. The autoload file is correct **as an fpath autoload** (the `#compdef skilldozer` header binds it; the self-call is the standard autoload idiom that fires once compsys is loaded), but it is **not eval-safe**: under `eval "$(skilldozer --completions)"` the `#compdef` header is an inert comment and the self-call runs the function immediately, hitting `_arguments` before compsys is guaranteed loaded (`_skilldozer:31: command not found: _arguments`). The §14.5 manual path (copy the file onto `fpath` + `compinit`) keeps using the verbatim autoload file; the `--completions` eval path uses the derived wrapper.

#### Lockstep (extends §14.4)

Because the scripts are baked in at build time, editing `completions/*` requires a **rebuild** for `--completions` to reflect it — whereas the §14.5 manual source/copy path picks up edits immediately. Keep both delivery paths in sync; the §14.4 flag-change rule applies to the embedded bytes too, and §14.1's "skills-first / long-form-only" behavior is part of that same lockstep contract.

### 14.7 Listing behavior — show every match, never a silent halt

A completion that completes the longest common prefix and then **stops with nothing shown** is a defect against this contract. When the token being completed matches **two or more** candidates — skill tags *or* long-form flags — the completion must surface the **full filtered candidate set** as visible hints: every candidate that begins with the typed string, listed so the user can read and choose among them.

Why this matters here specifically: the store is manifest-free (§2), so the user often does **not** know a tag ahead of time — discovery-via-completion is the primary way to find a skill. An ambiguous prefix that hides its candidates defeats that. The same applies to flags: `skilldozer --c<tab>` must show `--check` and `--completions`, not freeze at `--c`.

The requirement has two halves:

1. **Offer every match (skilldozer's job — already true; lock it).** The scripts offer the COMPLETE candidate list — tags via `skilldozer --relative --all`, flags via the frozen long-form set — and let the shell prefix-filter (§14.1 rule 2). The candidate set reaching the shell is always complete; skilldozer never filters matches out.
2. **Make the shell SHOW them on the first Tab (the fix this section adds).** Surfacing that set on the *first* Tab is governed by per-shell "list ambiguous matches" options — and on two of the three shells the default **is** the silent halt:
   - **zsh:** `LIST_AMBIGUOUS` is **on by default**, so the first Tab completes the common prefix and lists only once you have typed to the exact ambiguous point (`ag`→`agent-b`, nothing shown). `setopt NO_LIST_AMBIGUOUS` (with the default `AUTO_LIST`) makes the first Tab list **all** prefix matches immediately (verified empirically: it flips `ag<tab>` from no-list to showing both `agent-browser` and `agent-builder`). There is **no per-command scope** for this — a scoped `zstyle ':completion:*:*:_skilldozer:*' menu select` does **not** produce first-Tab listing (verified); it is a session-global option.
   - **bash:** `show-all-if-ambiguous` is **off by default** → first Tab completes the common prefix and beeps; the list appears only on the second Tab. `bind 'set show-all-if-ambiguous on'` lists all matches on the first Tab. This is a readline (session-global) setting; a completion function cannot set it, but the emitted `--completions` script can `bind` it.
   - **fish:** lists all matches in the pager by default. No action needed.

Because the zsh/bash options are **session-global** (they change listing for *every* command, not just skilldozer), enabling them is a coupling the user must be able to see and undo. The emitted `--completions` scripts therefore:

- set the option so skilldozer (and, yes, the rest of the shell) lists ambiguous matches on the first Tab — the "just works" default the user asked for;
- **disclose** the change in the emitted script's comments and in the README (§15), naming the exact option set;
- provide the one-line **opt-out** (`setopt LIST_AMBIGUOUS` for zsh, `bind 'set show-all-if-ambiguous off'` for bash), so a user who prefers the shell's stock behavior can restore it after `eval`.

The guaranteed-always-true part is #1 (the complete filtered set is always offered); #2 is the disclosed, opt-out-able enhancement that makes that set visible on the first Tab.

This complements §14.1 rule 2 ("prefix filtering is the shell's native job"): the shell filters **and lists** — the candidate set is always complete and (with the option) always visible.

---

## 15. README.md outline

Mirror the mcpeepants README's tone and structure:

1. **Title + one-liner:** "Standalone skill loader for pi — resolves a skill tag to an absolute path for `pi --skill`."
2. **Why:** centralized skills, **not** in any pi discovery location, loaded only on demand.
3. **Install:** `install.sh` (symlink) / `go install` (first-class) / from-source. First run: `skilldozer --init` (prompts for the store dir, writes the config).
4. **Usage:** the canonical `pi --skill "$(skilldozer tag)"` example, multi-skill example, `-f`, `--list`, `--search`, `--all`, `--check`, `--path`.
5. **Where skills live:** the `skills/` dir, the tag = relative dir path, the discovery rules (§7).
6. **Adding a skill:** drop a `<tag>/SKILL.md` under `skills/`; required frontmatter; run `skilldozer --check`.
7. **How `skilldozer` finds the store:** §8 — `skilldozer --init` writes a config pointing at the store; `SKILLDOZER_SKILLS_DIR` overrides it; sibling / walk-up are zero-config dev fallbacks.
8. **Shell completions:** `eval "$(skilldozer --completions)"` (or `skilldozer --completions --shell fish | source`) in your rc — a bare `<tab>` lists your skills; `skilldozer -<tab>` lists the long-form flags (§14).
9. **Constraints:** no catalog index (disk-discovered); a settings config file is fine; never auto-discovered by pi; loads only via `--skill`.

---

## 16. `.gitignore`

```
/skilldozer
/dist
*.test
*.out
.DS_Store
```

(`/skilldozer` ignores the locally-built binary; everything else is committed, including `skills/example/`.)

---

## 17. Constraints & guardrails (do NOT do these)

- ❌ Do **not** add a **catalog** index/manifest (e.g. `skills.json` enumerating skills). The catalog is always walked from disk. A **settings** file (store location, etc.) is expected and fine — see §8; the rule is only that catalog data already on disk is never duplicated into a sidecar.
- ❌ Do **not** place skills in any pi auto-discovery location. The store is loaded **only** via `--skill`.
- ❌ Do **not** make `skilldozer` install/copy skills into `~/.pi/...` or `~/.agents/...`. It only prints paths.
- ❌ Do **not** print anything to stdout on a failed/unknown tag resolution (breaks `pi --skill "$(skilldozer x)"`).
- ❌ Do **not** require Node, Python, or any runtime at *run* time (build-time `go` is fine).
- ❌ Do **not** ship more than the one example skill.
- ⚠️ **Completion lockstep:** when adding/renaming/removing a flag in `main.go parseArgs()`, update all three completion files (`skilldozer.bash`, `_skilldozer`, `skilldozer.fish`) identically — there is no shared source of truth the shells import (§14.4).
- ❌ **No bare-word subcommands.** Every non-tag action is a `--flag`; the bare positional namespace is reserved for skill tags so a bare `<tab>` unambiguously means "skills" (§6.1, §14). `check`/`init`/`completions` are `--check`/`--init`/`--completions` precisely so a skill named `check`/`init`/`completions` is never shadowed.
- ⚠️ **Completions are skills-first and long-form-only:** a bare `<tab>` shows skills (never help/commands); flag completion offers only `--long` forms, never short aliases (§14.1). Keep this invariant when editing the completion scripts.

---

## 18. Suggested build order (for the one-shot pass)

1. `go.mod` + `internal/skillsdir` + `main.go --path` → prove location resolution (§8). **Hardest part; do first.**
2. `internal/discover` (walk + frontmatter parse) → `--list`.
3. `internal/resolve` → `skilldozer <tag>`, `-f`, `--all`, `--relative`.
4. `--search`, `--check`.
5. `--help`/`--version`/error semantics + exit codes (§6.4).
6. Example skill + run §13 acceptance.
7. `install.sh` (symlink) + README + `.gitignore` + LICENSE.
8. Completions.

---

## 19. Decisions log (assumptions made in lieu of asking — override if you disagree)

| # | Decision | Default chosen | Rationale |
|---|---|---|---|
| 1 | Repo / binary name | `skilldozer` | The command as written in the request |
| 2 | Visibility | **public** | Matches mcpeepants + user's other repos |
| 3 | Language | **Go** | Static binary, instant startup, symlink-aware path resolution |
| 4 | Output unit | **directory** (default), `--file` for `SKILL.md` | `--skill <dir>` is supported & includes assets |
| 5 | Catalog index | **none** — walked from disk each call | Small, hand-edited catalog; an index would drift. *(Earlier this row claimed an "explicit user constraint" — that was a misattribution; the user did not request it. A **settings** config file is a separate concern and is now used — §8.)* |
| 6 | Canonical tag | relative dir path under `skills/`; basename/name/alias fallbacks | Inferable from disk; tolerant of common usage |
| 7 | Search metadata | `metadata.keywords`/`category`/`aliases` in frontmatter | Uses the spec's own optional `metadata` field |
| 8 | Frontmatter parser | `gopkg.in/yaml.v3` | Robust; only third-party dep |
| 9 | Install method | `go install` / release binary / `install.sh`; `skilldozer --init` configures the store | No clone forced. The binary is self-sufficient; first run prompts for (or is told via flag/env) the store dir. Sibling / walk-up rules kept as zero-config dev fallbacks. |
| 10 | Shipped skills | exactly one `example` | Proves the pipeline; repo is a loader, not a library |
| 11 | License | MIT | Match mcpeepants conventions |
| 12 | Settings file | `$XDG_CONFIG_HOME/skilldozer/config.yaml`, key `store` (YAML; reuses `yaml.v3`); `SKILLDOZER_CONFIG` overrides the path | Fixed home so the binary can bootstrap without being told; YAML avoids a new dependency |
| 13 | First-run UX | `skilldozer --init` prompts interactively; bare tags never prompt (any auto-prompt TTY-gated) | Protects the `pi --skill "$(skilldozer x)"` contract from hanging inside command substitution |
| 14 | Discovery order | env `SKILLDOZER_SKILLS_DIR` → config `store` → sibling of binary → walk-up → "run `skilldozer --init`" | Env overrides config for CI/tests; heuristics kept as zero-config dev fallbacks |
| 15 | Name | **Skilldozer** (binary `skilldozer`, env prefix `SKILLDOZER_`, module `github.com/dabstractor/skilldozer`) | Renamed from the `skpp` working title. `plan/` archive left as historical (it *was* skpp when written) |
| 16 | `--init` cwd auto-detect | If cwd contains any `SKILL.md` (any depth), default the store to cwd; else `$XDG_DATA_HOME/skilldozer/skills` | Run `skilldozer --init` inside an existing skills dir and it adopts it in place, no typing |
| 17 | No-args invocation | Bare `skilldozer` ⇒ usage to **stdout**, exit **0** (implicit `--help`) | Earlier: stderr/exit-1 (“parity with `get-server-config.sh`”). Overridden so `skilldozer \| grep …` works — the help must land on the piped stream. Only no-args is reclassified (error→help); genuine failures (unknown flag, mutually-exclusive modes, unresolved tag, unconfigured) stay on stderr, non-zero, preserving the §6.4 `$(...)` contract. |
| 18 | `--completions` delivery | **Emit script to stdout for `eval`** (`eval "$(skilldozer --completions)"`); scripts `//go:embed`-ded into the binary; **no** rc/file writes; `--shell` overrides `$SKILLDOZER_SHELL`/`$SHELL` detection | A child process cannot register completions in the parent shell — only the shell can, by eval'ing the script in its own process (idiom: `zoxide init`/`starship init`/`direnv hook`). Emitting keeps the binary side-effect-free (§14.5). Embedding makes it work for `go install` users with no clone (decision 9). `--install` (write rc) deferred — would revisit §14.5. |
| 19 | Subcommand → flag (namespace safety) | **No bare-word subcommands.** `check`/`init`/`completion` are now `--check`/`--init`/`--completions`; `completion` also renamed to the plural `--completions` | The bare positional namespace is reserved entirely for skill tags, so a skill named `check`/`init`/`completions` is never shadowed and a bare `<tab>` unambiguously means "skills" (§6.1, §14). Previously handled by "reserved subcommand shadows the tag," which was itself the namespace conflict. |
| 20 | Completions behavior | **Skills-first & long-form-only:** bare `<tab>` lists skills; `skilldozer -<tab>` lists `--long` flags (never short aliases); prefix filtering is the shell's native job | `<tab>` = skills is the headline UX the user asked for. Driving everything to `--flags` (decision 19) makes it trivially correct. Long-form-only matches the `--help` surface and keeps the menu short. |
| 21 | `--link` (external skill linking) | **`skilldozer --link <dir> [<dir>...]`** creates a symlink `<store>/<basename>` → absolutized `<dir>` for each directory (multi-target — see decision 23); refreshes an existing symlink (install.sh precedent), refuses a non-symlink (never clobber); validates each target is a dir with ≥1 `SKILL.md`, not the store itself/inside it; once `--link` is parsed every following positional is a link dir (never a tag); stdout = one link path per success, exit `0` if all link / `1` if any fail / `2` on zero dirs | The `npm link`/`pip install -e` idiom for skills — develop elsewhere, link into the store, discover via §7.1's symlink-following (no new discovery code). Avoids copying, giving a single source of truth for in-development skills. |
| 23 | `--link` multi-target (batch linking) | **Extended decision 21:** `--link` accepts one-or-more directories after a single `--link` — `skilldozer --link a b c` links all three. Processing is non-atomic (link-as-you-go): each valid dir links immediately, each failure prints a stderr line naming the dir, exit `1` if any fail (successful links remain — re-running `--link <failed-dir>` is the fix); zero dirs ⇒ exit `2`. The single-dir case is the degenerate n=1 batch, preserving the existing "one bad dir ⇒ nothing on stdout, exit 1" guarantee. | User request: linking several projects is a common setup step, and repeating `--link` per dir is needless friction. Non-atomic partial success mirrors `ln`/`git add` multi-target behavior and is safe because linking is idempotent (refresh on re-run). |
| 22 | Completion listing (ambiguous matches) | **First-Tab list-all-matches; never a silent halt at the common prefix.** skilldozer always offers the complete prefix-filtered candidate set (tags + flags); the emitted `--completions` script additionally sets the shell's list-ambiguous-matches option (zsh `NO_LIST_AMBIGUOUS`, bash `show-all-if-ambiguous on`, fish default) so matches are shown on the first Tab — disclosed + opt-out-able, since these are session-global options. | A manifest-free store makes completion the primary discovery path, so hidden candidates are a UX defect. No per-command scope exists on zsh/bash (verified: a scoped `menu select` zstyle does **not** list on the first Tab; only the global `NO_LIST_AMBIGUOUS` does), so the option must be set globally — but disclosed and reversible to respect shell sovereignty. |
