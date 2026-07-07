# Verified Facts — P1.M6.T14.S1 (`README.md` per §15 outline, mcpeepants tone)

All facts below were captured **from the live repo at `~/projects/skpp`** (binary
already built: `./skpp`). The README MUST document **actual** behavior, so every
example below is runnable as-is and its output is exact. The implementer should
**re-run** these commands to keep the README's example blocks truthful (do not
invent output).

---

## 1. PRD §15 outline (THE spec for the README's section order)

```
1. Title + one-liner: "Standalone skill loader for pi — resolves a skill tag to
   an absolute path for `pi --skill`."
2. Why: centralized skills, NOT in any pi discovery location, loaded only on demand.
3. Install: install.sh (symlink) / go install (+ SKPP_SKILLS_DIR caveat) / from-source.
4. Usage: pi --skill "$(skpp tag)", multi-skill, -f, --list, --search, --all, check, --path.
5. Where skills live: the skills/ dir, tag = relative dir path, discovery rules (§7).
6. Adding a skill: drop a <tag>/SKILL.md under skills/, required frontmatter, run skpp check.
7. How skpp finds the store: §8 rules + SKPP_SKILLS_DIR.
8. Constraints: manifest-free; never auto-discovered by pi; loads only via --skill.
```

The README must cover all 8 in this order. mcpeepants README is the **tone**
template; PRD §15 is the **structure** spec (skpp's README is richer than
mcpeepants' 8-section because §15 demands more).

---

## 2. mcpeepants README — the tone to mirror (full text, `~/projects/mcpeepants/README.md`)

```markdown
# mcpeepants

CLI helper for generating MCP server configurations.

## Usage

​```bash
claude --mcp-config "$(./get-server-config.sh server1 server2)"
​```

## Available Servers

- `desktop-commander` - Filesystem access, terminal execution, process management
- `playwright` - Browser automation and web scraping
- `zai-mcp-server` - Z AI model access for text analysis
- `sequential-thinking` - Reasoning scaffolding for multi-step tasks
- `context7` - Real-time documentation search for 50,000+ libraries
- `chrome-devtools` - Direct Chrome DevTools Protocol access
- `serena` - Advanced coding agent toolkit with semantic editing

## Examples

​```bash
# Generate config with browser automation and sequential thinking
claude --mcp-config "$(./get-server-config.sh playwright sequential-thinking)"
...
​```
```

**Tone takeaways (apply to skpp):**
- Plain `# Title`, single descriptive sentence under it.
- Lead with the **canonical one-liner** in the first code block (skpp's: `pi --skill "$(skpp tag)"`).
- Bulleted capability lists use `key - description`.
- Examples block uses commented `# ...` lines above each invocation.
- No badges, no emojis in headings, no marketing adjectives ("blazing fast",
  "powerful", "seamless"). No "Features:" h2. No "Why use X?" hard-sell.

`architecture/mcpeepants_patterns.md` confirms: "skpp README should follow the
same shape (PRD §15 gives the full outline): title + one-liner → why → install →
usage → where skills live → adding a skill → how skpp finds the store → constraints."

---

## 3. Actual `skpp` CLI behavior (captured; README examples must match these)

Module: `github.com/dabstractor/skpp` (go.mod). Go directive: `go 1.25`.
Version (no tags yet) prints as the short SHA: `skpp cc347c6` (dynamic — do NOT
hardcode a version string in the README; document that `--version` prints the
git-describe value).

### `--help` / `-h` (stdout, exit 0)
```
skpp — skill path printer

Resolve skill tags to on-disk skill directory paths (manifest-free).

USAGE:
  skpp <tag> [<tag>...]
  skpp --all
  skpp --list
  skpp --search <query>
  skpp check
  skpp --path
  skpp --help
  skpp --version

EXAMPLES:
  pi --skill "$(skpp example)"
  pi --skill "$(skpp writing/reddit)"
  skpp example reddit          # one absolute path per line, input order
  skpp -f example              # print the SKILL.md path
  skpp --relative --all        # every skill path, relative to the skills dir
  skpp --list                  # human-readable catalog
  skpp --search reddit         # substring search over tag/name/description/keywords
  skpp check                   # validate every skill on disk

OPTIONS:
  <tag> [<tag>...]   Resolve tags to skill directory paths (one absolute path per line)
  --all, -a          Print every skill's directory path, sorted by tag
  --list, -l         Human-readable catalog (TAG, NAME, DESCRIPTION)
  --search <q>, -s   Substring search over tag / name / description / keywords
  check              Validate every skill on disk (report OK / WARN / ERROR)
  --path, -p         Print the resolved skills directory
  --file, -f         Print the SKILL.md path instead of the directory (modifier)
  --relative         Print paths relative to the skills directory (modifier)
  --no-color         Disable ANSI color even on a TTY (modifier)
  --help, -h         Show this help message
  --version, -v      Print the skpp version

Exit codes: 0 success/help/version | 1 unresolved/no skills/unresolvable dir | 2 unknown flag / mutually-exclusive modes
```

### Command outputs (repo at `~/projects/skpp`)
| Invocation | stdout | rc |
|---|---|---|
| `skpp --version` | `skpp cc347c6` | 0 |
| `skpp --path` | `/home/dustin/projects/skpp/skills` | 0 |
| `skpp example` | `/home/dustin/projects/skpp/skills/example` | 0 |
| `skpp -f example` | `/home/dustin/projects/skpp/skills/example/SKILL.md` | 0 |
| `skpp --all` | `/home/dustin/projects/skpp/skills/example` | 0 |
| `skpp --relative example` | `example` | 0 |
| `skpp --list` | table: `TAG NAME DESCRIPTION` with wrapped description (example → 5 wrapped lines) | 0 |
| `skpp --search example` | same table format, filtered | 0 |
| `skpp --search nomatch` | (empty) + stderr `no skills matched nomatch` | 1 |
| `skpp nope` | (empty, NOTHING to stdout) + stderr `unknown skill tag "nope"` | 1 |
| `skpp` (no args) | full help to **stderr** | 1 |
| `skpp check` | `OK    example (example)` then `1 skills, 0 errors, 0 warnings` | 0 |

### `--list` exact output (the table format to show in README)
```
TAG      NAME     DESCRIPTION
example  example  Reference example skill for skpp.
                  Demonstrates the required frontmatter
                  and how skpp resolves a tag to an
                  absolute path. Safe to delete once you
                  add real skills.
```

### `check` exact output
```
OK    example (example)
1 skills, 0 errors, 0 warnings
```

### Error contract (§6.4 — load-bearing for `$(...)`)
- Unknown tag → **nothing on stdout**, error to stderr, exit 1.
  Verified: `out=$(./skpp nope 2>/dev/null); [ -z "$out" ] && echo $?` → `1`.
- This is WHY `pi --skill "$(skpp badtag)"` fails loudly instead of passing a
  garbage/empty path. The README "Usage" and "Constraints" sections must state this.

### Multi-tag + ordering
`skpp example reddit` prints one absolute path per line **in input order**.
On ANY unresolved tag → nothing printed, exit 1 (atomicity — protects `pi`).

---

## 4. Install paths to document (install.sh ALREADY EXISTS — document its behavior)

`install.sh` (PRD §12.1) exists at repo root and:
1. Builds `go build -trimpath -ldflags "-s -w -X main.version=$(git describe ...)" -o skpp .`
2. Picks target: `$SKPP_INSTALL_BIN` → `$HOME/.local/bin` → `/usr/local/bin`.
3. **Symlinks** `<target>/skpp → <repo>/skpp` (NOT copy — load-bearing for §8.2).
4. Advises PATH (prints rc-file line; never auto-edits).
5. Verifies: `"$TARGET/skpp" --version` + `"$TARGET/skpp" example`.

Three install paths for the README §Install section:
- **install.sh (recommended):** `./install.sh` → symlink into `~/.local/bin`.
  Keeps `os.Executable()` resolving back to the repo → finds `skills/` automatically.
- **go install:** `go install github.com/dabstractor/skpp@latest` lands in
  `$(go env GOPATH)/bin` with **NO** adjacent `skills/` → user MUST set
  `SKPP_SKILLS_DIR=/path/to/cloned/repo/skills`. Document PROMINENTLY (PRD §12.2).
- **from-source:** `go build -o skpp .` then run `./skpp` from the repo, OR
  symlink manually: `ln -sfn "$PWD/skpp" ~/.local/bin/skpp`.

`SKPP_INSTALL_BIN` (installer target override) is an install-time var;
`SKPP_SKILLS_DIR` (runtime skills-dir override, PRD §8 rule 1) is the one users
of a `go install`'d binary set. Do not conflate them.

---

## 5. Discovery rules to summarize (README §5 "Where skills live" + §7 "How skpp finds the store")

**Tag (canonical) = path of the skill dir relative to `skills/`, separators `/`.**
Tag resolution precedence (§7.2): exact canonical tag → basename → frontmatter
`name` → declared `metadata.aliases` → unknown.

Skills-dir location priority (§8): `SKPP_SKILLS_DIR` env → sibling of the
binary (`os.Executable()` + `EvalSymlinks`) → walk-up from cwd → fail with a fix.

A skill = any directory directly containing a `SKILL.md`. Nested skills count
(e.g. `skills/writing/reddit/SKILL.md` is tag `writing/reddit`).

---

## 6. Frontmatter conventions (README §6 "Adding a skill")

Required: `name` (1–64 chars, lowercase `a-z0-9-`, no leading/trailing/consecutive
hyphens), `description` (max 1024 chars; pi won't load a skill with no description).
skpp uses the standard optional `metadata` map for `keywords`/`category`/`aliases`.
Unknown keys ignored. `license`, `compatibility` optional.

The shipped example (`skills/example/SKILL.md`) is the concrete copy-pasteable
template to point readers at. Its frontmatter:
```
name: example
description: >
  Reference example skill for skpp. Demonstrates the required frontmatter and
  how skpp resolves a tag to an absolute path. Safe to delete once you add real skills.
metadata:
  keywords: [example, demo, skpp]
  category: meta
```

---

## 7. Constraints to restate (README §8 "Constraints", from PRD §17)

- Manifest-free (no `skills.json`/index — everything from disk).
- Skills live in a location pi does **not** auto-scan: never
  `~/.pi/agent/skills`, `~/.agents/skills`, a project `.pi/skills`/`.agents/skills`,
  a `node_modules` package, or a `package.json` with `pi.skills`.
- Loaded ONLY via `pi --skill "$(skpp <tag>)"`.
- `skpp` only ever PRINTS paths — it never installs/copies skills into `~/.pi/...`
  or `~/.agents/...`.
- Zero run-time runtime deps (build-time `go` only).

---

## 8. Completions handling (T15 not yet done — do NOT over-document)

`completions/` (bash/zsh/fish) is task P1.M6.T15.S1 (status: Planned). The README
§15 outline has NO dedicated completions section, and PRD §14 makes completions
deferrable. Therefore: **the README must not document completion commands that
don't exist yet.** If `completions/` exists when the implementer writes the
README, add a one-line "Shell completions (optional): see `completions/`"
pointer under Install. If it does not exist yet, OMIT completions entirely
(no broken references). This keeps the README truthful regardless of T15 timing.

---

## 9. Quality lever available in this environment

A `write-tech-docs` skill is available (enforces: no em dashes, no marketing
tell-words, no hedging/formulaic transitions, no narrating the codebase; ships a
linter). It is OPTIONAL and aligns with the mcpeepants plain-prose tone. The
binding spec is PRD §15 + mcpeepants tone; the skill is a quality check, not a
source of structure. Mention it in the PRP only as an optional polish step so the
implementer knows it exists, but do not let it override the PRD outline.

---

## 10. Files this task creates/touches (scope)

- **CREATE** `README.md` at repo root. ONE file. Nothing else changes.
- NO edits to PRD.md, main.go, install.sh, skills/, internal/, go.mod, .gitignore.
- The README is gitignored by nothing (it's a committed doc). `git status` after
  should show ONLY `README.md` as a new untracked file.
