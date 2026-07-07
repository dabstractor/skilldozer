# skpp

Standalone skill loader for pi — resolves a skill tag to an absolute path for `pi --skill`.

## Why

`skpp` gives you a centralized, on-disk catalog of pi skills addressed by short
tags. The catalog lives **deliberately outside** every directory pi scans, so
skills never bloat your context automatically. They load **only on demand** when
you pass an explicit `--skill`:

```bash
pi --skill "$(skpp example)"
```

If a tag is unknown, `skpp` prints nothing and exits 1, so the `$(...)` fails
loudly instead of handing pi an empty path.

## Install

Three paths. `./install.sh` is recommended.

**A. `./install.sh` (recommended)**

Builds the binary with version info and symlinks it into `~/.local/bin`
(or `$SKPP_INSTALL_BIN`, or `/usr/local/bin` if that is what is writable):

```bash
./install.sh
```

The install **symlinks** rather than copies. That matters: `skpp` resolves its
own executable path back through the symlink, which is how it finds the
adjacent `skills/` directory with no env var.

**B. `go install`**

```bash
go install github.com/dabstractor/skpp@latest
```

> **`go install` caveat.** A `go install`'d binary lands in
> `$(go env GOPATH)/bin` with **no** adjacent `skills/` directory, so `skpp`
> cannot auto-discover the store from there. Set the runtime override before
> use:
>
> ```bash
> export SKPP_SKILLS_DIR=/absolute/path/to/your/cloned/skpp/skills
> ```
>
> If you hit `skpp example` reporting it cannot find skills, this is the fix.
> (Prefer `./install.sh`, which symlinks the binary next to the repo so
> discovery works with no env var.)

**C. From source**

```bash
go build -o skpp . && ./skpp example
```

or build, then symlink it yourself:

```bash
go build -o skpp .
ln -sfn "$PWD/skpp" ~/.local/bin/skpp
```

Run `./skpp example` from the repo, or use the symlink from anywhere.

## Usage

The canonical one-liner, first:

```bash
pi --skill "$(skpp example)"
```

Everything else, commented:

```bash
# Resolve a tag to an absolute path (default: the skill directory)
skpp example                       # → /…/skills/example

# Print the SKILL.md path instead of the directory
skpp -f example                    # → /…/skills/example/SKILL.md

# Load several skills into pi in one command
pi --skill "$(skpp writing/reddit)" --skill "$(skpp example)"

# Resolve multiple tags at once (one absolute path per line, input order)
skpp example writing/reddit

# Human-readable catalog and substring search
skpp --list
skpp --search reddit

# Print every skill path, sorted by tag
skpp --all

# Validate every skill on disk
skpp check

# Where is the resolved skills directory?
skpp --path                        # → /…/skills

# Version is the git-describe value (dynamic, not a fixed string)
skpp --version
```

**Error contract.** An unknown tag prints **nothing** to stdout and exits 1
(the error goes to stderr only). That is why
`pi --skill "$(skpp badtag)"` fails loudly instead of loading nothing. When
multiple tags are given, any unresolved tag causes nothing to be printed and
exit 1, so `pi` never sees a partial result.

`skpp --help` lists every flag.

## Where skills live

Skills live in the `skills/` directory at the repo root. A skill is any
directory that directly contains a `SKILL.md`.

The canonical **tag** is the skill directory's path **relative to `skills/`**,
with `/` separators. It is **not** the frontmatter `name`.

```text
skills/example/SKILL.md              → tag example
skills/writing/reddit/SKILL.md       → tag writing/reddit
```

Nested skills count, so `skills/writing/reddit/SKILL.md` is addressed as
`writing/reddit`, not `reddit`.

Tag resolution tries, in order:

1. the exact canonical tag (`writing/reddit`)
2. the final path segment / basename (`reddit`)
3. the frontmatter `name`
4. a declared alias (see `metadata.aliases`)
5. else: unknown

So `skpp example`, `skpp writing/reddit`, `skpp reddit` (if unique), and
`skpp foo-helper` (matching a frontmatter `name`) all resolve.

## Adding a skill

Drop a directory under `skills/` with a `SKILL.md` whose frontmatter declares
at least `name` and `description`:

```markdown
---
name: example
description: >
  One or two sentences describing what the skill does. pi will not load a
  skill that has no description.
metadata:
  keywords: [example, demo, skpp]
  category: meta
  aliases: [demo]
---

# Example Skill

Body of the skill. This is what pi loads when you pass the path.
```

- `name` — required. Lowercase `a-z0-9-`, 1–64 chars, no leading/trailing or
  consecutive hyphens.
- `description` — required (max 1024 chars). pi will not load a skill without one.
- `metadata.keywords`, `metadata.category`, `metadata.aliases` — optional.
  Unknown keys are ignored.

`skills/example/SKILL.md` is a copy-pasteable template — start from it.

When you are done, validate everything on disk:

```bash
skpp check
```

Output:

```text
OK    example (example)
1 skills, 0 errors, 0 warnings
```

## How `skpp` finds the store

`skpp` locates `skills/` by this priority:

1. **`SKPP_SKILLS_DIR` env var** — wins if set and the directory exists. This
   is the override `go install` users set (see Install).
2. **Sibling of the binary** — `os.Executable()` plus `EvalSymlinks()` resolves
   the real binary path and looks for `skills/` next to it. This is the rule a
   `./install.sh` symlink install relies on; a copy would break it silently.
3. **Walk up from the current directory** — useful during development
   (`go run .` / running `./skpp` from a checkout).
4. **Else: fail with a one-line fix** telling you how to set `SKPP_SKILLS_DIR`.

`skpp --path` reports which directory won.

## Constraints

`skpp` is deliberately a thin, manifest-free path printer.

- **Manifest-free.** No `skills.json`, no index file. Everything is resolved
  from the directory tree on each call.
- **Never auto-discovered by pi.** The skills store does **not** live in any
  directory pi scans. It is never:
  - `~/.pi/agent/skills`
  - `~/.agents/skills`
  - a project `.pi/skills` or `.agents/skills`
  - a `node_modules` package
  - a `package.json` with a `pi.skills` field
- **Loaded only via `--skill`.** A skill enters your context only when you ask
  for it explicitly: `pi --skill "$(skpp <tag>)"`.
- **`skpp` only ever prints paths.** It never copies or installs skills into
  `~/.pi/...` or `~/.agents/...`. Where the path points is up to you.
- **Zero runtime dependencies.** Build-time needs Go; the resulting binary
  stands alone.
