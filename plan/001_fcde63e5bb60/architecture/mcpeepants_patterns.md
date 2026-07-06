# Conventions to Mirror — `mcpeepants` reference (`~/projects/mcpeepants`)

`skpp` mirrors mcpeepants' **UX shape and tone**, NOT its implementation.
mcpeepants = bash + `servers.json` manifest. skpp = Go + manifest-free.

## Examined reference artifacts

| File | What we mirror |
|---|---|
| `README.md` | Tone: short, example-driven, leading with the canonical one-liner. |
| `QUICK_INSTALL.sh` | Install-script spirit: detect shell, install completion, print verify cmd. skpp's `install.sh` additionally builds the Go binary + symlinks. |
| `completion.sh` (bash) | Bash completion registering the `skpp` command + flags + dynamic tags. |
| `_mcpeepants` (zsh) | Zsh `#compdef` style; `_arguments` for flags; dynamic positional completion. |
| `mcpeepants.fish` (fish) | `complete -c skpp` lines for flags + tag completions. |
| `get-server-config.sh` | Arg-parsing + help-text structure; error→stderr discipline. |

## README tone (from mcpeepants README)

```markdown
# mcpeepants

CLI helper for generating MCP server configurations.

## Usage

claude --mcp-config "$(./get-server-config.sh server1 server2)"

## Available Servers
- server-key - one-line description
```

**skpp README should follow the same shape** (PRD §15 gives the full outline):
title + one-liner → why → install → usage (the canonical `pi --skill "$(skpp tag)"`)
→ where skills live → adding a skill → how skpp finds the store → constraints.

## Install script differences (important — do NOT copy mcpeepants blindly)

mcpeepants `QUICK_INSTALL.sh` only installs **shell completions** (it copies
completion files). skpp's `install.sh` (PRD §12.1) does MORE:

1. `cd` to script's own dir (repo root).
2. Verify `go` on PATH; else print install instructions, exit 1.
3. `go build -trimpath -ldflags "-s -w -X main.version=$(git describe ...)" -o skpp .`
4. Pick target bin dir: `$SKPP_INSTALL_BIN` → `$HOME/.local/bin` → `/usr/local/bin`.
5. **SYMLINK** (not copy) `<target>/skpp` → `<repo>/skpp`. Refresh if exists.
   (Copying breaks §8.2 sibling-of-binary rule. This is the critical difference.)
6. Ensure target dir on PATH; else print exact `export PATH=…` for detected shell.
7. Print verification: `skpp example`.

Then optionally install completions (mcpeepants-style shell detection). Completions
may be deferred (PRD §14) but flag clearly in the PR if so.

## Completion patterns to adapt

mcpeepants completions hardcode the server list (from servers.json). skpp MUST NOT
hardcode tags (manifest-free!). Instead, dynamic completion calls `skpp --all`
(cheap disk walk) and offers the printed paths' relTag/basename. Pattern:

- bash: `_skpp_completion()` runs `skpp --all`, splits lines, offers as `compgen`.
- zsh: `_skpp()` uses `_call_program` / `$(skpp --all)` for positional args.
- fish: `complete -c skpp -a "(skpp --all)"` with function wrapping.

Flags to complete: `--all/-a --list/-l --search/-s --path/-p --file/-f --version/-v
--help/-h --no-color --relative` + subcommand `check`.

## Files NOT in mcpeepants that skpp needs

- `LICENSE` (MIT) — mcpeepants has none; PRD §5/§19 require MIT for skpp.
- `go.mod`/`go.sum` — Go project (mcpeepants is bash, has package.json only).
- `internal/` Go packages — no analog in mcpeepants.

## .gitignore (PRD §16)

```
/skpp
/dist
*.test
*.out
.DS_Store
```

(`/skpp` = the locally-built binary; everything else committed, incl. `skills/example/`.)
