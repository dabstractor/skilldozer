# Skilldozer

Standalone skill loader for pi. Resolves a skill tag to an absolute path for `pi --skill`.

## Why

`skilldozer` gives you a centralized, on-disk catalog of pi skills addressed by short
tags. The catalog lives **deliberately outside** every directory pi scans, so
skills never bloat your context automatically. They load **only on demand** when
you pass an explicit `--skill`:

```bash
pi --skill "$(skilldozer example)"
```

If a tag is unknown, `skilldozer` prints nothing and exits 1, so the `$(...)` fails
loudly instead of handing pi an empty path.

## Install

Three paths. `./install.sh` is recommended.

**A. `./install.sh` (recommended)**

Builds the binary with version info and symlinks it into `~/.local/bin`
(or `$SKILLDOZER_INSTALL_BIN`, or `/usr/local/bin` if that is what is writable):

```bash
./install.sh
```

The install **symlinks** rather than copies. That matters: `skilldozer` resolves its
own executable path back through the symlink, which is how it finds the
adjacent `skills/` directory with no env var.

**B. `go install`**

```bash
go install github.com/dabstractor/skilldozer@latest
```

`go install` lands the binary in `$(go env GOPATH)/bin`. On first use, run
`skilldozer --init` (see First run, below) — it creates the store and writes the
config. No clone required, and no `SKILLDOZER_SKILLS_DIR` needed for normal use.

**C. From source**

```bash
go build -o skilldozer . && ./skilldozer example
```

or build, then symlink it yourself:

```bash
go build -o skilldozer .
ln -sfn "$PWD/skilldozer" ~/.local/bin/skilldozer
```

Run `./skilldozer example` from the repo, or use the symlink from anywhere.

### First run

Whichever install path you used, run `skilldozer --init` once:

```bash
skilldozer --init
```

It prompts for the directory where skilldozer should keep your skills
(defaulting to `$XDG_DATA_HOME/skilldozer/skills`, or the current directory if
it already looks like a skill store), creates it, seeds an `example/SKILL.md`
template if it is empty, and writes the config pointing at it. For scripts / CI,
skip the prompt:

```bash
skilldozer --init /path/to/store      # positional
skilldozer --init --store /path/to/store
```

`--store <dir>` implies `--init`, so it works on its own as a first-class
non-interactive form: `skilldozer --store /path/to/store` runs the full setup
and writes the config. (Use one of the forms above in scripts when you want the
intent to be self-evident; bare `--store` with an `--init` token is the canonical
shape.) Because `--store` implies `--init`, it cannot be combined with tag
arguments: `skilldozer --store /path mytag` exits 2 — it is `--init`, not a
one-off store override for a single resolution.

On success, `--init` prints exactly the configured store path to stdout — one clean
line, so `STORE="$(skilldozer --init --store /path)"` works in scripts. The
seeded/adopted status and the post-setup `--check` report go to stderr. A leading
`~` (or a bare `~`) in a typed answer or a `--store`/positional path expands to
your home directory.

## Usage

The canonical one-liner, first:

```bash
pi --skill "$(skilldozer example)"
```

Everything else, commented:

```bash
# Resolve a tag to an absolute path (default: the skill directory)
skilldozer example                       # → /…/skills/example

# Print the SKILL.md path instead of the directory (-f / --file)
skilldozer -f example                    # → /…/skills/example/SKILL.md

# Load several skills into pi in one command
pi --skill "$(skilldozer writing/reddit)" --skill "$(skilldozer example)"

# Resolve multiple tags at once (one absolute path per line, input order)
skilldozer example writing/reddit

# Human-readable catalog and substring search
skilldozer --list
skilldozer --search reddit            # matches tag / name / description / keywords / aliases / category

# Print every skill path, sorted by tag
skilldozer --all

# Validate every skill on disk
skilldozer --check

# Link an external skill directory into the store (no copy; npm-link style, §8.4)
skilldozer --link ~/projects/agent-browser   # creates <store>/agent-browser -> that dir

# Where is the resolved skills directory? (its discovery rule prints to stderr)
skilldozer --path                        # → /…/skills (stderr: found via sibling of binary)

# Print paths relative to the skills directory instead of absolute
skilldozer --relative example

# Disable ANSI color even on a TTY (for --list / --search tables)
skilldozer --no-color --list

# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'
skilldozer --version

# Short flags combine (-af) and long flags accept --flag=value (--search=reddit)
```

**Error contract.** An unknown tag prints **nothing** to stdout and exits 1
(the error goes to stderr only). That is why
`pi --skill "$(skilldozer badtag)"` fails loudly instead of loading nothing. When
multiple tags are given, any unresolved tag causes nothing to be printed and
exit 1, so `pi` never sees a partial result. The `--path`, `--list`, `--search`,
and `--all` modes are mutually exclusive — combining any two exits 2, as does
combining a tag with any of them (a tag resolves one path; those modes inspect
the whole store). `--link` is another such mode. The flags that take a value —
`--store`, `--search`, `--shell`, and `--link` — all exit 2 when given as the
last token with nothing after them, rather than guessing a value.

`skilldozer --help` lists every flag.

### Linking skills from elsewhere (`--link`)

`skilldozer --link <dir>` makes a skill directory that lives **outside** the
store available in it, **without copying** — the `npm link` / `pip install -e`
idiom for skills. You point it at any directory containing a `SKILL.md` (or a
directory of skills), and it creates a symlink `<store>/<basename>` → that
directory. Discovery already follows symlinks, so the linked skill resolves by
its name exactly like a real one:

```bash
skilldozer --link ~/projects/agent-browser
skilldozer agent-browser          # now resolves (via the symlink)
pi --skill "$(skilldozer agent-browser)"
```

`~` is expanded and the target is absolutized, so the link stays valid from any
working directory. Re-running `--link` on the same name **refreshes** an existing
symlink (re-points it); it **refuses** to overwrite a real file or directory at
that name (remove it yourself first). It also refuses a target that isn't a
directory, contains no `SKILL.md`, or is the store itself. The store must already
be configured (`skilldozer --init`).

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

So `skilldozer example`, `skilldozer writing/reddit`, `skilldozer reddit` (if unique), and
`skilldozer foo-helper` (matching a frontmatter `name`) all resolve.

There are **no reserved tag names**: bare words are always skill tags, and every
action is a `--flag` (§6.1). A skill named `check`, `init`, or `completions`
resolves normally by its tag — use `--check`, `--init`, or `--completions` to
run the action.

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
  keywords: [example, demo, skilldozer]
  category: meta
  aliases: [demo]
---

# Example Skill

Body of the skill. This is what pi loads when you pass the path.
```

- `name`: required. Lowercase `a-z0-9-`, 1-64 chars, no leading/trailing or
  consecutive hyphens.
- `description`: required (max 1024 chars). pi will not load a skill without one.
- `metadata.keywords`, `metadata.category`, `metadata.aliases`: optional.
  Unknown keys are ignored.

`skills/example/SKILL.md` is a copy-pasteable template; start from it.

When you are done, validate everything on disk:

```bash
skilldozer --check
```

Output:

```text
OK    example (example)
1 skills, 0 errors, 0 warnings
```

## How `skilldozer` finds the store

`skilldozer` locates `skills/` by this priority:

1. **`SKILLDOZER_SKILLS_DIR` env var** — override; if set and an existing dir,
   use it. Lets CI / tests / temporary redirects win without editing the config.
2. **Config file `store`** — the primary, set by `skilldozer --init`. The config
   lives at `$XDG_CONFIG_HOME/skilldozer/config.yaml` (→
   `~/.config/skilldozer/config.yaml`); override the file path with
   `SKILLDOZER_CONFIG=<file>` (handy for tests / multiple profiles). Minimal
   valid file:

   ```yaml
   store: /home/you/skills
   ```

   A missing or unreadable config is treated as "not yet configured" and falls
   through to the rules below — never a hard error. A config whose `store:` points
   at a directory that no longer exists is different: skilldozer names the missing
   path and exits 1 rather than silently falling through to a different store.
3. **Sibling of the running binary** (symlink-aware: `os.Executable()` plus
   `EvalSymlinks()`) — still lets a clone-and-build dev workflow work with no
   config. This is the rule a `./install.sh` symlink install relies on; a copy
   would break it silently.
4. **Walk up from `cwd`** — for `go run` / dev.
5. **None** ⇒ unconfigured: skilldozer prints
   `skilldozer is not configured; run \`skilldozer --init\`` to stderr, writes
   nothing to stdout, and exits 1.

`skilldozer --path` reports the winning directory on stdout and the matching rule
on stderr — one of `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`,
or `ancestor of cwd`. The stderr label matters when `SKILLDOZER_SKILLS_DIR` is
typo'd: a bad value is silently ignored and discovery falls through to a lower
rule, so the `--path` label is the only way to tell the env var was skipped.

## Shell completions

`skilldozer` ships dynamic completions for bash, zsh, and fish. Tag completion is
not a static list: the shell calls `skilldozer --relative --all` at completion time,
so it never goes stale as you add skills.

The easiest way to load completions is the `--completions` flag, which prints
the script for your shell to eval. The binary embeds the completion scripts, so
this works for `go install` users with no clone.

**bash / zsh** — add to `~/.bashrc` or `~/.zshrc`:

```bash
eval "$(skilldozer --completions)"
```

**fish** — add to `~/.config/fish/config.fish`:

```bash
skilldozer --completions --shell fish | source
```

`--shell <bash|zsh|fish>` makes the eval deterministic; otherwise
`skilldozer --completions` auto-detects from `$SKILLDOZER_SHELL`, then `$SHELL`.

Once loaded, completions are **skills-first and long-form-only**:

- `skilldozer <tab>` lists your installed skill tags (the default, most-used
  action) — never the help text or a command list. The list is recomputed from
  `skilldozer --relative --all` on every keystroke, so a newly-dropped skill is
  completable immediately.
- `skilldozer -<tab>` lists the **long-form flags only** — `--all`, `--check`,
  `--completions`, `--file`, `--help`, `--init`, `--link`, `--list`, `--no-color`,
  `--path`, `--relative`, `--search`, `--shell`, `--store`, `--version` — narrowed
  by what you type after the dash. Short aliases (`-a`, `-l`, …) stay valid for
  typing but are deliberately not advertised.
- `skilldozer --init <tab>`, `skilldozer --link <tab>`, and `skilldozer --store <tab>`
  offer directories (a path value); `skilldozer --search <tab>` offers nothing
  (free-text); `skilldozer --shell <tab>` offers the three supported shells —
  `bash`, `zsh`, and `fish`.

This works because every action that is not a skill tag is a `--flag` —
`--check`, `--init`, and `--completions` are flags, not bare subcommands — so the
bare positional namespace belongs entirely to skill tags and a `<tab>` is
unambiguous.

The emitted script also sets a shell option so that when a prefix matches two or
more skills or flags, **every** match lists on the first `<tab>` instead of the
shell freezing at the common prefix. Because the store has no index, completion
is the main way to discover skills — hiding candidates would defeat that.

This is a **session-global** option: it changes tab-completion listing for *every*
command in that shell, not just `skilldozer`, and it is set only when you load
skilldozer's completions (via the `eval`/`source` lines above). The option each
shell sets:

- **bash** — `show-all-if-ambiguous` (set on)
- **zsh** — `NO_LIST_AMBIGUOUS` (set on)
- **fish** — lists all matches by default; no option is set

Prefer your shell's stock behavior? Restore the default after loading completions:

```bash
# bash — list on the second Tab again
bind 'set show-all-if-ambiguous off'

# zsh — list only at the exact ambiguous point again
setopt LIST_AMBIGUOUS
```

Prefer to copy the file instead? The manual path below picks up edits to
`completions/*` without a rebuild.

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

then ensure this is in your `.zshrc`:

```bash
autoload -U compinit && compinit
```

**fish**:

```bash
cp completions/skilldozer.fish ~/.config/fish/completions/skilldozer.fish
```

`install.sh` does not install completions automatically; copy the file you
want as shown above.

## Constraints

`skilldozer` is deliberately a thin path printer.

- **No catalog index.** There is no `skills.json`, no manifest enumerating
  skills — the catalog is always walked from disk on each call. A *settings*
  config file (the store location, written by `skilldozer --init`) is expected and
  fine; the rule is only that catalog data already on disk is never duplicated
  into a sidecar.
- **Never auto-discovered by pi.** The skills store does **not** live in any
  directory pi scans. It is never:
  - `~/.pi/agent/skills`
  - `~/.agents/skills`
  - a project `.pi/skills` or `.agents/skills`
  - a `node_modules` package
  - a `package.json` with a `pi.skills` field
- **Loaded only via `--skill`.** A skill enters your context only when you ask
  for it explicitly: `pi --skill "$(skilldozer <tag>)"`.
- **`skilldozer` only ever prints paths.** It never copies or installs skills into
  `~/.pi/...` or `~/.agents/...`. Where the path points is up to you.
- **Zero runtime dependencies.** Build-time needs Go; the resulting binary
  stands alone.
