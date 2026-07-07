# Verified facts — P1.M6.T13.S1 (`install.sh`: build + symlink + target + PATH + verify)

Empirically checked against the live repo at `~/projects/skpp` (Go 1.25,
git repo, no tags) and the `mcpeepants` reference at `~/projects/mcpeepants`.

## Environment facts (drive target selection + PATH advice)

| Probe | Result | Implication |
|---|---|---|
| `git tag -l` | (empty — no tags) | `git describe --tags --always` returns the short SHA, e.g. `cc347c6` |
| `git describe --tags --always 2>/dev/null` | `cc347c6` (exit 0) | the `|| echo dev` fallback fires ONLY when not a git repo |
| `ls -ld ~/.local/bin` | exists, owned by user, writable | default target dir (PRD §12.1 step 4, option 2) |
| `~/.local/bin` on `$PATH`? | **YES** | no PATH-advice needed on this machine; script still must check |
| `/usr/local/bin` writable? | **NO** (needs sudo) | last-resort target; script must NOT auto-sudo |
| `$SHELL` | `/usr/bin/zsh` | PATH-advice must emit zsh (and bash/fish) snippets |
| `./skpp` built yet? | **NO** | install.sh must build it first (step 3) |

## The build command (authoritative = PRD §12.1 step 3)

```bash
go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .
```

- `-trimpath` strips local paths from the binary (reproducible).
- `-s -w` strips the symbol table + DWARF (smaller binary; safe — `--version`
  reads `main.version`, NOT debug info).
- `-X main.version=<sha>` rewrites the package-scope `var version = "dev"`
  in `main.go` (line 41) at link time. Confirmed: it MUST be a `var`, not a
  `const`; the symbol path is `main.version` (file is `package main`).
- Command substitution `$(...)` inside the double-quoted `-ldflags` string is
  standard bash and works identically inside a script and at the prompt. The
  whole `-ldflags "..."` becomes ONE argv element (correct — `go` parses the
  inner space-separated `-s -w -X ...`).
- With no tags the version printed by `skpp --version` is the short SHA; after a
  `git tag v1.0.0` it becomes `v1.0.0` (or `v1.0.0-N-gXXXXXXX` if commits past
  the tag). The `|| echo dev` only matters if someone builds outside a git
  checkout.

## The symlink contract (THE critical requirement)

PRD §12.1 step 5 + §8.2 + `architecture/verified_symlink_resolution.md`:

- The install MUST `symlink` `<target>/skpp → <repo>/skpp`, **never copy**.
- Why: at runtime `os.Executable()` → `filepath.EvalSymlinks()` →
  `filepath.Dir()` must yield the **repo dir** (which contains `skills/`).
  - On Linux, `os.Executable()` already resolves through `/proc/self/exe` to
    the real binary path; `EvalSymlinks` is redundant-but-harmless.
  - On macOS, `os.Executable()` may return the symlink path, so `EvalSymlinks`
    is REQUIRED. (This is the skillsdir code already shipped; install.sh just
    must not defeat it.)
- A COPY would put a binary at `<target>/skpp` whose `Dir()` = `<target>` (e.g.
  `~/.local/bin`), which has NO `skills/` sibling → rule 2 misses → falls to
  rule 3 (walk-up) which only works if cwd is under the repo → broken in
  practice. **Copy = silent breakage.**

### `ln` gotcha (must use `ln -sfn`)

```bash
ln -sfn "$REPO_SKPP" "$TARGET/skpp"
```

- `-s` symbolic, `-f` force (unlink existing dest first), `-n` treat an existing
  symlink-to-directory dest as a normal file (do NOT dereference it).
- Without `-n`: if `$TARGET/skpp` is ALREADY a symlink pointing at a directory,
  `ln -sf` would follow it and create the link INSIDE that dir
  (`$REPO_SKPP/skpp`), silently corrupting the install. `-n` prevents this. This
  is the single most common `ln` footgun; the symlink must target a FILE
  (`<repo>/skpp`) but defensive `-n` is correct regardless.
- The link TARGET must be an **absolute** path (`$REPO_SKPP`), never relative:
  the link lives in a different dir (`~/.local/bin`) than the binary, so a
  relative link would resolve against `~/.local/bin` and break.
- Idempotent: re-running `install.sh` rebuilds + `ln -sfn` refreshes cleanly.

## Repo-root detection (install.sh lives at repo root)

```bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"
```

- Mirrors mcpeepants `QUICK_INSTALL.sh` (uses `SCRIPT_DIR` from `BASH_SOURCE`).
- Gives the absolute repo root because `install.sh` ships at repo root (PRD §5).
- `cd` makes the `go build ... -o skpp .` output land at repo root, which is
  gitignored by `/skpp` (PRD §16 / `.gitignore`). No `.gitignore` change needed.

## Target-bin-dir selection (PRD §12.1 step 4 order)

1. `$SKPP_INSTALL_BIN` — if set AND an existing or creatable dir → use it
   (`mkdir -p`). Lets users pin a non-default location.
2. `$HOME/.local/bin` — if present OR creatable → `mkdir -p` and use it. This
   is in `$HOME`, so always writable; almost always already on PATH (it was on
   this machine). The de-facto default.
3. `/usr/local/bin` — only if writable. On this machine it is NOT writable, so
   the script must detect that and, rather than silently `sudo`, print the
   exact command for the user to run with `sudo` (or fall back to `~/.local/bin`
   with a note).

**No silent sudo.** Auto-prompting for a password from an install script is
hostile + breaks unattended installs. If sudo is genuinely required (the only
writable target needs it), print `sudo ln -sfn "$REPO_SKPP" /usr/local/bin/skpp`
and exit with a clear message.

## PATH-advice (PRD §12.1 step 6)

After selecting `$TARGET`, check `case ":$PATH:" in *":$TARGET:"*) ;; * ) print advice;; esac`.
If missing, detect shell via `basename "$SHELL"`:

| Shell | rc file | snippet to print |
|---|---|---|
| bash | `~/.bashrc` | `export PATH="$HOME/.local/bin:$PATH"` |
| zsh | `~/.zshrc` | `export PATH="$HOME/.local/bin:$PATH"` |
| fish | `~/.config/fish/config.fish` | `fish_add_path "$HOME/.local/bin"` (or `set -gx PATH $HOME/.local/bin $PATH`) |
| other/unknown | — | print the generic `export PATH=...` line + tell user to add to their shell rc |

Only PRINT the snippet + the file to append to; do not silently edit rc files
(same anti-pattern discipline as "no silent sudo"). Appending to rc files
automatically is intrusive and can duplicate lines on re-run.

## Verify command (PRD §12.1 step 7)

Strongest verification uses the **absolute path to the freshly-created symlink**
so it works even before the new PATH entry is loaded in the current shell:

```bash
"$TARGET/skpp" example      # full symlink→repo→binary→skills resolution
"$TARGET/skpp" --version    # confirms ldflags injected a version (not "dev")
```

Then ALSO print `skpp example` for the user to run after reloading their shell
(or `hash -r` / `exec $SHELL`). The §13 acceptance symlink test
(`/tmp/skpp-bin/skpp example`) is the model: a symlink in an arbitrary dir still
resolves to `$PWD/skills/example`.

## mcpeepants patterns mirrored (UX/tone, NOT implementation)

`~/projects/mcpeepants/QUICK_INSTALL.sh`:
- `#!/bin/bash`, `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"`.
- Shell detection via `CURRENT_SHELL=$(basename "$SHELL")` + `case`.
- Prints a `🚀 ... Quick Install` banner and a `Test it:` block at the end.
- BUT mcpeepants installs **completions only** (copies completion files) and has
  NO binary build / NO symlink. skpp's `install.sh` does MORE: build Go binary +
  symlink + PATH advice + verify. (Per `architecture/mcpeepants_patterns.md`.)

skpp does NOT install completions in this script — completions are a separate
task (P1.M6.T15.S1) and PRD §14 allows deferring them. `install.sh` may print a
one-line pointer to the completions task, but must not implement them.

## Go-on-PATH check (PRD §12.1 step 2)

```bash
if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: 'go' not found on PATH." >&2
  echo "Install Go from https://go.dev/doc/install and re-run." >&2
  exit 1
fi
```

`command -v` is POSIX and works in bash/zsh. Do not assume a version; just
presence (the build command itself will fail loudly on an unsupported Go).

## No shell test framework exists in this repo

- The repo has `main_test.go` (Go `testing`) only. There is NO harness for
  `.sh` scripts (no bats, no shellcheck config). mcpeepants likewise has none.
- Validation for this task is therefore **manual acceptance commands** (mirrors
  how the rest of M6 packaging tasks will be validated, and PRD §13 itself).
- Optional follow-up (NOT in scope unless requested): `shellcheck install.sh`
  as a lint gate, or a tiny `bats` smoke test. Flagged in the PRP, not built.

## Exit-code discipline for install.sh

- `0` — built + symlinked + (verify command printed or run). 
- `1` — go missing, build failed, or could not select a writable target.
- Keep all user-facing messages on the right streams (errors → stderr, banner /
  progress → stdout) so `install.sh 2>/dev/null` is quiet on success.
