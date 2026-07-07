# PRP — P1.M6.T13.S1: `install.sh` (build + symlink into PATH, §12.1)

> **Subtask:** P1.M6.T13.S1 — build-order step 7 (packaging/docs cluster).
> Create `install.sh` at repo root that **builds** the `skpp` Go binary with the
> version ldflags, **symlinks** it into a PATH directory (NOT a copy — this is
> the load-bearing requirement), advises on PATH when needed, and verifies the
> install. Mirrors the *spirit* of mcpeepants `QUICK_INSTALL.sh` but does more
> (build + symlink + PATH advice + verify); it does NOT install completions
> (separate task P1.M6.T15.S1, PRD §14 allows deferral).
>
> **Scope:** CREATE exactly **one** new file — `install.sh` — and `chmod +x` it.
> No Go code. No package changes. No `.gitignore` change (the built `skpp` is
> already ignored by `/skpp`). No README edits (README is P1.M6.T14.S1).
>
> **Why this is small-but-critical:** the file is ~80 lines of bash, but ONE
> wrong decision (`cp` instead of `ln -s`) silently breaks §8.2 sibling-of-binary
> resolution for every symlink-installed user. The whole point of the script is
> to produce a symlink whose existence makes `os.Executable()` resolve back to
> the repo's `skills/` dir. Get the symlink right and everything else (§13
> acceptance symlink test) passes for free.

---

## Goal

**Feature Goal**: From a clean clone, a user runs `./install.sh` (or
`bash install.sh`) and ends up with a working `skpp` command on PATH that
resolves skill tags by reading the repo's `skills/` directory — with the version
string baked in via ldflags (not the default `"dev"`), and with clear guidance
when PATH needs updating or `go`/`sudo` is missing.

**Deliverable**: One new executable file, `install.sh`, at repo root. ~80 lines
of bash. Implements PRD §12.1 steps 1–7 exactly. No other files change.

**Success Definition** (all must hold after a clean `./install.sh`):
- A freshly-built `./skpp` binary exists at repo root (gitignored by `/skpp`).
- `./skpp --version` prints `skpp <sha-or-tag>`, NOT `skpp dev`.
- `$TARGET/skpp` is a **symbolic link** (not a regular file) whose readlink
  target is the **absolute** path `$REPO/skpp`.
- `"$TARGET/skpp" example` prints the absolute path `$REPO/skills/example`,
  exit 0 — proving the symlink → `os.Executable()` → repo `skills/` chain works
  end-to-end from outside the repo dir.
- Re-running `./install.sh` is idempotent (no duplicate PATH lines, no error,
  symlink refreshed in place).
- On this machine: target = `$HOME/.local/bin` (exists, on PATH, writable); no
  sudo needed; no PATH-advice printed.

## User Persona

**Target User**: A pi operator adopting `skpp`. They cloned the repo and want a
one-command install that leaves `skpp` on PATH and *keeps working* as they `git
pull` and rebuild.

**Use Case**: "I cloned skpp — make it a real command." → `./install.sh`, then
`skpp example` works from any directory.

**Pain Points Addressed**:
- "I copied the binary to `~/.local/bin` and now `skpp example` can't find
  `skills/`." → Fixed by **symlink**, not copy (the whole reason this script
  exists; a naive `cp` is the #1 footgun).
- "Every rebuild I have to re-copy." → Symlink means one source of truth; rebuild
  + `ln -sfn` refresh in place.
- "My version string says `dev`." → ldflags inject the git describe value.
- "I don't know if `~/.local/bin` is on PATH." → The script detects the shell
  and prints the exact rc-file line.

## Why

- **PRD §12.1 — the authoritative spec for this script.** Seven numbered steps;
  this PRP turns each into a script section. The step ordering is also the build
  order within the script (verify go → build → pick target → symlink → PATH →
  verify).
- **PRD §8.2 / §13 — the symlink install is what makes sibling-of-binary
  resolution work.** `architecture/verified_symlink_resolution.md` proves that
  `~/.local/bin/skpp → ~/projects/skpp/skpp` causes `os.Executable()` to resolve
  back to the repo. The §13 acceptance suite literally tests this:
  `ln -sf "$PWD/skpp" /tmp/skpp-bin/skpp; /tmp/skpp-bin/skpp example` must still
  print `$PWD/skills/example`. `install.sh` exists to create exactly such a link
  in a real PATH dir.
- **PRD §2 constraint 5 — one-shot buildable.** The install must work from the
  PRD alone with no further questions; this PRP supplies the bash-level detail
  the PRD leaves implicit (the `ln -sfn` gotcha, no-silent-sudo, fish PATH
  snippet, verify-via-absolute-symlink-path).
- **Decoupling from completions.** PRD §14 explicitly allows deferring
  completions. `install.sh` does NOT touch completions; it may print a one-line
  pointer. This keeps the install script small and the completions task (T15)
  independent.
- **Cohesion with prior/future work items:** M1–M5 (the Go binary, discovery,
  resolution, CLI) are all landed and green; the example skill (T12) is landed.
> This script packages that already-working binary. It must NOT change the
> binary's behavior — it only builds + symlinks it. It does not regress any T1–T12
> acceptance. The README (T14) and completions (T15) build on top of the install
> path this script establishes, so the target-dir / PATH-advice decisions here are
> the contract they reference.

## What

User-visible behavior of `./install.sh` (run from repo root):

1. Prints a short banner (mcpeepants-style) naming what it's doing.
2. If `go` is missing → prints install instructions (https://go.dev/doc/install)
   to stderr, exits 1. No build attempted.
3. Builds: `go build -trimpath -ldflags "-s -w -X main.version=$(git describe
   --tags --always 2>/dev/null || echo dev)" -o skpp .` If the build fails →
   the `go` error is on stderr, script exits non-zero, no symlink created.
4. Picks `$TARGET` (PRD §12.1 step 4): `$SKPP_INSTALL_BIN` (if set + creatable)
   → `$HOME/.local/bin` (mkdir -p) → `/usr/local/bin` (only if writable). If
   none is usable without sudo → prints the exact `sudo ln -sfn ...` command and
   exits with a clear message (no silent password prompt).
5. `ln -sfn "$REPO/skpp" "$TARGET/skpp"` (refresh in place if it exists).
6. If `$TARGET` is NOT on `$PATH`, detects `$SHELL` and prints the exact line to
   append to the matching rc file (`~/.bashrc` / `~/.zshrc` /
   `~/.config/fish/config.fish`), with fish using `fish_add_path`. Does NOT edit
   the rc file automatically.
7. Verifies by running `"$TARGET/skpp" --version` and `"$TARGET/skpp" example`
   (absolute symlink path — works even before the new PATH entry is live), then
   prints the user-facing `skpp example` to run after reloading the shell.
8. Exits 0 on success.

### Success Criteria

- [ ] `install.sh` exists at repo root and is executable (`-x`).
- [ ] Shebang is `#!/usr/bin/env bash` (or `#!/bin/bash`); uses `set -euo
      pipefail` (or equivalent careful error handling).
- [ ] `./install.sh` exits 0 on this machine (go present, `~/.local/bin`
      writable + on PATH).
- [ ] After install: `test -L "$HOME/.local/bin/skpp"` (it is a symlink, not a
      file).
- [ ] `readlink -f "$HOME/.local/bin/skpp"` ends in `/skpp` (points at the repo
      binary, absolute).
- [ ] `"$HOME/.local/bin/skpp" --version` prints `skpp <non-dev>` (the SHA/tag,
      NOT `skpp dev`).
- [ ] `"$HOME/.local/bin/skpp" example` prints `<repo>/skills/example`, exit 0.
- [ ] Re-running `./install.sh` exits 0 and does NOT duplicate anything.
- [ ] With `go` absent (`PATH=/usr/bin:/bin ./install.sh` if go lives elsewhere):
      exits 1 with the go.dev install URL on stderr, no symlink created.
- [ ] `SKPP_INSTALL_BIN=/tmp/skpp-target ./install.sh` installs the symlink into
      `/tmp/skpp-target/skpp` and that link resolves `example` correctly.
- [ ] `git status` shows ONLY `install.sh` as a new untracked file — no changes
      to `.gitignore`, `main.go`, `skills/`, or any `internal/` package.

## All Needed Context

### Context Completeness Check

_If someone knew nothing about this codebase, would they have everything needed
to implement this successfully?_ **Yes.** Every step is a single bash statement
given verbatim below. The only non-obvious decisions (symlink-not-copy, `ln
-sfn`, no-silent-sudo, absolute link target, fish `fish_add_path`) are spelled
out in "Known Gotchas" with the exact reason. No Go knowledge is needed beyond
"run the build command"; no codebase files need to be read to write the script
(they are referenced only to *verify* the script's correctness afterward).

### Documentation & References

```yaml
# MUST READ — the authoritative spec (the 7 numbered steps ARE the script sections)
- file: PRD.md
  section: "§12.1 'install.sh (mirrors mcpeepants QUICK_INSTALL.sh spirit)' —
            steps 1–7 are the implementation checklist."
  why: "This is the single source of truth for the script's behavior and the
        exact build command (step 3) and target-selection order (step 4)."
  critical: "Step 5 says SYMLINK (not copy) and 'If a symlink already exists,
             refresh it.' Implement as `ln -sfn` (see Gotcha 2). A copy silently
             breaks §8.2."

# MUST READ — why the symlink (not copy) is load-bearing
- file: plan/001_fcde63e5bb60/architecture/verified_symlink_resolution.md
  why: "Proves (Linux + macOS) that `os.Executable()`+`EvalSymlinks()`+`Dir()`
        yields the repo dir ONLY when the PATH entry is a symlink to `<repo>/skpp`.
        A copy would make `Dir()` = the copy's dir, which has no `skills/` sibling."
  critical: "This is THE reason `install.sh` exists. Copying is the #1 bug."

# READ — the install script's UX/tone template (mirror spirit, not implementation)
- file: ~/projects/mcpeepants/QUICK_INSTALL.sh
  why: "mcpeepants' SCRIPT_DIR resolution, shell detection (`basename $SHELL`),
        banner, and 'Test it:' verify block. skpp mirrors the tone."
  pattern: "SCRIPT_DIR=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")\" && pwd)\";
            CURRENT_SHELL=$(basename \"$SHELL\"); case ... in bash|zsh|fish)..."
  gotcha: "mcpeepants ONLY installs completions (copies files) and has NO build/
           symlink. Do NOT copy mcpeepants' `cp`-the-completion approach for the
           binary — skpp's binary install is `go build` + `ln -sfn`."

# READ — what mcpeepants patterns to mirror vs diverge from
- file: plan/001_fcde63e5bb60/architecture/mcpeepants_patterns.md
  section: "'Install script differences (important — do NOT copy mcpeepants blindly)'"
  why: "Explicitly lists the 7 skpp install steps and flags that completions may
        be deferred (so install.sh must not implement them)."

# VERIFY-AGAINST — the ldflags target this script populates
- file: main.go
  section: "lines 30–41 (the `var version = \"dev\"` block + its ldflags comment)"
  why: "Confirms `-X main.version=...` rewrites a package-scope `var` (NOT a
        const) and the symbol path is `main.version` because the file is
        `package main`. The build command in §12.1 step 3 matches this exactly."
  critical: "After install, `skpp --version` must NOT print `dev` — that proves
             the ldflags actually fired. (If it shows `dev`, the `$(...)` was
             quoted wrong or the build didn't run.)"

# VERIFY-AGAINST — the §13 acceptance that this script's output must satisfy
- file: PRD.md
  section: "§13 'Acceptance criteria' — the symlink + env-override block"
  why: "The acceptance suite tests the symlink behavior this script produces:
        a symlink in an arbitrary dir still resolves `example` to
        `$PWD/skills/example`; and `SKPP_SKILLS_DIR=... ./skpp example` works."

# REFERENCE — the skillsdir rule this script's symlink feeds
- file: internal/skillsdir/skillsdir.go
  section: "findSibling / resolveSiblingFromExe (PRD §8 rule 2)"
  why: "Shows exactly how the binary finds `skills/`: `Dir(EvalSymlinks(exe)) +
        \"/skills\"`. The symlink install is what makes `exe` resolve to the repo."

# REFERENCE — research notes for this task (all env probes + gotchas)
- docfile: plan/001_fcde63e5bb60/P1M6T13S1/research/verified_facts.md
  why: "Verified env facts (no git tags → SHA; ~/.local/bin on PATH; /usr/local/bin
        needs sudo; SHELL=zsh; no built binary), the exact build command, the
        `ln -sfn` gotcha, and the no-silent-sudo rule."
```

### Current Codebase tree (relevant slice)

```bash
skpp/
├── PRD.md                 # §12.1 = the spec for this script
├── main.go                # `var version = "dev"` (line 41) — ldflags target
├── go.mod                 # module github.com/dabstractor/skpp, go 1.25
├── go.sum
├── .gitignore             # already ignores `/skpp` (built binary) — no change
├── internal/…             # skillsdir §8 rule 2 consumes the symlink this makes
├── skills/example/SKILL.md# the skill `skpp example` resolves to (verify target)
└── (no install.sh yet)    # ← THIS TASK CREATES IT
```

### Desired Codebase tree with files to be added

```bash
skpp/
└── install.sh             # NEW (this task). Executable. ~80 lines bash.
                          #   Build + symlink + target-select + PATH-advice + verify.
                          #   Does NOT install completions (T15) or edit README (T14).
# (built artifact ./skpp is regenerated by this script; already gitignored)
```

### Known Gotchas of our codebase & Library Quirks

```bash
# CRITICAL (1): SYMLINK, NEVER COPY.
#   `cp skpp ~/.local/bin/skpp` makes os.Executable()'s Dir() = ~/.local/bin,
#   which has NO skills/ sibling → §8 rule 2 misses → falls to rule 3 (walk-up)
#   → only works if cwd is under the repo → broken in practice.
#   The install is load-bearing BECAUSE of the symlink. Use:
ln -sfn "$REPO_SKPP" "$TARGET/skpp"

# CRITICAL (2): `ln -sfn`, not `ln -sf`. The `-n` treats an existing
#   symlink-to-directory dest as a file (does not dereference). Without `-n`, if
#   `$TARGET/skpp` is already a symlink pointing into a directory, `ln -sf`
#   would follow it and place the link INSIDE that dir, silently corrupting the
#   install. The target here is a FILE, but `-n` is the correct defensive form.

# CRITICAL (3): the link TARGET must be ABSOLUTE ($REPO_SKPP), never relative.
#   The link lives in a different dir (~/.local/bin) than the binary, so a
#   relative link would resolve against ~/.local/bin and break.

# CRITICAL (4): version ldflags use command substitution INSIDE double quotes —
#   bash expands $(...) within "...", so the whole `-ldflags "..."` is ONE argv
#   element and `go` parses the inner `-s -w -X ...`. This is correct; do not
#   "fix" the quoting by escaping the $ (that would pass a literal string).
go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .

# GOTCHA (5): NO SILENT SUDO. If the only writable target needs root, print the
#   exact `sudo ln -sfn "$REPO_SKPP" /usr/local/bin/skpp` for the user; do not
#   auto-prompt for a password (hostile + breaks unattended installs).

# GOTCHA (6): Do NOT auto-append to rc files. Only PRINT the snippet + the file.
#   Auto-editing ~/.zshrc etc. is intrusive and duplicates lines on re-run.

# GOTCHA (7): `git describe --tags --always` with NO tags returns the short SHA
#   (verified: `cc347c6`). The `|| echo dev` fires ONLY when not a git repo.
#   So a fresh clone (no tags) → `skpp --version` prints `skpp cc347c6`. That is
#   correct and expected; do not try to force "dev" or a fake tag.

# GOTCHA (8): verify via the ABSOLUTE symlink path, not bare `skpp`.
#   `"$TARGET/skpp" example` works even before the new PATH entry is live in the
#   CURRENT shell. Bare `skpp example` may still hit a stale hash/cache until the
#   user reloads. Print BOTH: the absolute-path verify (run by the script), and
#   the bare `skpp example` (for the user to run after `exec $SHELL`).

# GOTCHA (9): shebang `#!/usr/bin/env bash` (portable) or `#!/bin/bash`. The
#   script uses bash-isms (`[[ ]]`, `${BASH_SOURCE[0]}`, arrays if any) — NOT
#   POSIX sh. mcpeepants QUICK_INSTALL.sh is `#!/bin/bash`; match that family.

# GOTCHA (10): `set -euo pipefail` is recommended, BUT the `$(git describe ...
#   2>/dev/null || echo dev)` MUST keep its `|| echo dev` even under `set -e`
#   (it already handles the failure internally). If using `set -e`, ensure the
#   go-missing branch uses an explicit `exit 1` (it does) and that `command -v
#   go` is wrapped so a missing go does not abort before printing help.
```

## Implementation Blueprint

### Script structure (no data models — this is bash; organize as named sections)

`install.sh` is a single flat bash script. Structure it as clearly-commented
sections matching PRD §12.1 steps 1–7, in that order. No functions are strictly
required, but a small `die()` helper for error+exit keeps the code clean.

```bash
#!/usr/bin/env bash
# install.sh — build skpp and symlink it into PATH (PRD §12.1).
# Mirrors mcpeepants QUICK_INSTALL.sh spirit; does MORE (build + symlink + PATH).
# Does NOT install completions (separate task, PRD §14 allows deferral).
set -euo pipefail

die() { echo "ERROR: $*" >&2; exit 1; }

# --- §12.1 step 1: cd to the script's own dir (repo root) ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# --- §12.1 step 2: verify go on PATH ---
# (see Implementation Tasks: Task 1 for the exact command + go.dev URL)

# --- §12.1 step 3: build with version ldflags ---
# go build -trimpath -ldflags "-s -w -X main.version=$(git describe ...)" -o skpp .

# --- §12.1 step 4: pick target bin dir ($SKPP_INSTALL_BIN → ~/.local/bin → /usr/local/bin) ---

# --- §12.1 step 5: SYMLINK (ln -sfn) $TARGET/skpp → $SCRIPT_DIR/skpp ---

# --- §12.1 step 6: ensure $TARGET on PATH; else print rc-file snippet ---

# --- §12.1 step 7: verify ($TARGET/skpp --version ; $TARGET/skpp example) ---

echo "Done. Run 'skpp example' (after reloading your shell if PATH changed)."
```

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: CREATE install.sh at repo root (the whole deliverable)
  - IMPLEMENT: the 7 PRD §12.1 steps as clearly-commented bash sections, in order:
      1. cd to SCRIPT_DIR (repo root): `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; cd "$SCRIPT_DIR"`
      2. go-on-PATH check: `if ! command -v go >/dev/null 2>&1; then echo "..." >&2; exit 1; fi`
         (message names https://go.dev/doc/install)
      3. BUILD (exact command, see "Known Gotcha 4"):
         `go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .`
         - On build failure (non-zero exit), let `set -e` abort with go's error;
           do NOT proceed to symlink.
      4. TARGET selection (first usable wins):
         a. `if [[ -n "${SKPP_INSTALL_BIN:-}" ]]; then TARGET="$SKPP_INSTALL_BIN"; mkdir -p "$TARGET";`
         b. `elif [[ -d "$HOME/.local/bin" || -w "$HOME" ]]; then TARGET="$HOME/.local/bin"; mkdir -p "$TARGET";`
         c. `elif [[ -w "/usr/local/bin" ]]; then TARGET="/usr/local/bin";`
         d. `else` print exact `sudo ln -sfn "$SCRIPT_DIR/skpp" /usr/local/bin/skpp`
            and `exit 1` (NO silent sudo).
      5. SYMLINK (NOT copy): `ln -sfn "$SCRIPT_DIR/skpp" "$TARGET/skpp"`
         - `$SCRIPT_DIR/skpp` is ABSOLUTE (SCRIPT_DIR is absolute from step 1).
      6. PATH check: `case ":$PATH:" in *":$TARGET:"*) ;; *) print_advice ;; esac`
         where print_advice branches on `basename "${SHELL:-}"`:
           bash → 'export PATH="$HOME/.local/bin:$PATH"'  for ~/.bashrc
           zsh  → same line for ~/.zshrc
           fish → 'fish_add_path "$HOME/.local/bin"'       for ~/.config/fish/config.fish
           *    → generic export line + "add to your shell's rc file"
         PRINT the snippet + the rc filename; do NOT auto-append.
      7. VERIFY (absolute symlink path, works pre-PATH-reload):
         `"$TARGET/skpp" --version`   # must NOT say "dev"
         `"$TARGET/skpp" example`     # must print $SCRIPT_DIR/skills/example
         Then print: "Reload your shell (or run: exec $SHELL), then: skpp example"
  - FOLLOW pattern: ~/projects/mcpeepants/QUICK_INSTALL.sh (SCRIPT_DIR, shell
    detection via `basename "$SHELL"`, banner, "Test it:" block). See gotcha: do
    NOT copy mcpeepants' cp-the-completion approach for the binary.
  - NAMING: file `install.sh` (lowercase, PRD §5 layout). Function/helper `die()`.
  - PLACEMENT: repo root (sibling of main.go, go.mod). NOT in a subdir.
  - GUARDRAILS: `set -euo pipefail`; errors→stderr (`>&2`); banner/progress→stdout;
    exit 0 only on full success; exit 1 on go-missing / build-fail / no-writable-target.

Task 2: MAKE install.sh executable
  - COMMAND: `chmod +x install.sh`
  - WHY: PRD §5 implies a runnable installer; `./install.sh` must work, not just
    `bash install.sh`. The +x bit is committed (git tracks mode).

Task 3: RUN + VERIFY (acceptance; see Validation Loop Level 3 for full commands)
  - RUN: `./install.sh`
  - ASSERT: `test -L "$HOME/.local/bin/skpp"` (symlink, not file)
  - ASSERT: `readlink -f "$HOME/.local/bin/skpp"` ends in `/skpp`
  - ASSERT: `"$HOME/.local/bin/skpp" --version` != `skpp dev`
  - ASSERT: `"$HOME/.local/bin/skpp" example` == `<repo>/skills/example`
  - ASSERT: re-run `./install.sh` exits 0, no duplicates
```

### Implementation Patterns & Key Details

```bash
# Banner (mcpeepants tone) — keep short; progress to stdout, errors to stderr.
echo "🚀 skpp install"
echo "Repo: $SCRIPT_DIR"
echo

# go-missing branch (step 2) — exit BEFORE building, message to stderr.
if ! command -v go >/dev/null 2>&1; then
  cat >&2 <<'EOF'
ERROR: 'go' was not found on PATH.
Install Go from https://go.dev/doc/install, then re-run ./install.sh.
EOF
  exit 1
fi

# Build (step 3) — the ldflags $(...) expands inside the double quotes (Gotcha 4).
go build -trimpath \
  -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
  -o skpp .
# Under `set -e`, a build failure aborts here with go's own diagnostics.
# Do NOT wrap in `|| ...` that would mask the failure.

# Target selection (step 4) — no silent sudo (Gotcha 5).
if [[ -n "${SKPP_INSTALL_BIN:-}" ]]; then
  TARGET="$SKPP_INSTALL_BIN"; mkdir -p "$TARGET"
elif [[ -d "$HOME/.local/bin" ]] || [[ -w "$HOME" ]]; then
  TARGET="$HOME/.local/bin"; mkdir -p "$TARGET"
elif [[ -w "/usr/local/bin" ]]; then
  TARGET="/usr/local/bin"
else
  cat >&2 <<EOF
ERROR: no writable install target found.
Re-run with: SKPP_INSTALL_BIN=/your/bin ./install.sh
Or (system-wide): sudo ln -sfn "$SCRIPT_DIR/skpp" /usr/local/bin/skpp
EOF
  exit 1
fi

# Symlink (step 5) — THE load-bearing line. ln -sfn, absolute target (Gotchas 1–3).
ln -sfn "$SCRIPT_DIR/skpp" "$TARGET/skpp"

# PATH advice (step 6) — detect shell, PRINT only (Gotcha 6).
case ":${PATH:-}:" in
  *":$TARGET:"*) ;;  # already on PATH, nothing to do
  *)
    sh="$(basename "${SHELL:-}")"
    case "$sh" in
      bash) echo "Add to ~/.bashrc:  export PATH=\"$TARGET:\$PATH\"" ;;
      zsh)  echo "Add to ~/.zshrc:   export PATH=\"$TARGET:\$PATH\"" ;;
      fish) echo "Add to ~/.config/fish/config.fish:  fish_add_path \"$TARGET\"" ;;
      *)    echo "Add '$TARGET' to your PATH (shell rc file)." ;;
    esac
    ;;
esac

# Verify (step 7) — absolute symlink path works pre-PATH-reload (Gotcha 8).
"$TARGET/skpp" --version
"$TARGET/skpp" example
echo
echo "Reload your shell (exec $SHELL), then:  skpp example"
```

### Integration Points

```yaml
BUILD ARTIFACT:
  - creates: "./skpp" (repo root) — already gitignored by `/skpp` (PRD §16). NO .gitignore change.
  - version: injected via ldflags into main.version (main.go line 41). Read by `skpp --version`.

FILESYSTEM:
  - creates: "$TARGET/skpp" as a SYMLINK → "$REPO/skpp" (absolute).
  - consumes (at runtime, via the binary): "$REPO/skills/" (§8 rule 2 sibling-of-binary).
  - mkdir: "$HOME/.local/bin" (or $SKPP_INSTALL_BIN) if absent — only under $HOME or an explicit env dir.

CONFIG:
  - env read: SKPP_INSTALL_BIN (optional target override). NOTHING ELSE.
  - env NOT touched: SKPP_SKILLS_DIR is a RUNTIME var for the binary, not the installer.

NO CHANGES TO:
  - .gitignore, main.go, go.mod, internal/*, skills/*, README.md, LICENSE.
  - The binary's behavior (install.sh only builds + symlinks the already-correct binary).
```

## Validation Loop

> **Note on testing:** this repo has Go `testing` only (`main_test.go`); there is
> NO shell-test harness (no bats, no shellcheck config), matching mcpeepants.
> Validation for this task is therefore **manual acceptance commands** (how PRD
> §13 itself validates, and how the rest of M6 packaging tasks will be checked).
> OPTIONAL follow-up (out of scope unless requested): add `shellcheck install.sh`
> as a lint gate or a tiny `bats` smoke test. Flagged here, not built.

### Level 1: Syntax & Style (Immediate Feedback)

```bash
cd ~/projects/skpp

# Bash syntax check (no execution) — catches unmatched quotes/braces fast.
bash -n install.sh && echo "syntax OK"

# Shellcheck (if installed) — catches the ln -sfn vs ln -sf, unquoted vars, etc.
command -v shellcheck >/dev/null 2>&1 && shellcheck install.sh || echo "(shellcheck not installed; optional)"

# Executable bit set.
test -x install.sh && echo "executable OK"

# Expected: syntax OK; executable OK. (shellcheck: SC2086 on intentional word-splitting
# is acceptable; SC2312 on `$(...)` in `set -e` is fine here. No errors.)
```

### Level 2: Unit / Component Validation (script pieces)

There are no unit tests for a shell script in this repo. Validate each piece by
isolated probe instead:

```bash
cd ~/projects/skpp

# Piece: SCRIPT_DIR resolves to the repo root.
bash -c 'SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; echo "$SCRIPT_DIR"' install.sh
# Expected: <repo root> (the dir containing this PRP's parent's parent's... repo root).

# Piece: the build command produces a non-"dev" version.
go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o /tmp/skpp-probe .
/tmp/skpp-probe --version
# Expected: skpp <sha-or-tag>, NOT "skpp dev".
rm -f /tmp/skpp-probe

# Piece: go-missing branch exits 1 with the URL.
env PATH=/usr/bin:/bin bash install.sh 2>&1 | grep -q "go.dev/doc/install" && echo "go-missing branch OK" || echo "FAIL"
# (This run must NOT create a symlink — verify: readlink "$HOME/.local/bin/skpp" unchanged.)

# Expected: each piece behaves as specified.
```

### Level 3: Integration Testing (full install + §13-style symlink contract)

```bash
cd ~/projects/skpp

# Full clean run.
./install.sh
RC=$?; [ "$RC" = "0" ] && echo "install exit OK" || { echo "FAIL exit $RC"; exit 1; }

# The PATH entry is a SYMLINK (not a copy) — the load-bearing assertion.
test -L "$HOME/.local/bin/skpp" && echo "is-symlink OK" || { echo "FAIL: not a symlink"; exit 1; }

# The symlink points at the ABSOLUTE repo binary.
case "$(readlink -f "$HOME/.local/bin/skpp")" in
  */skpp) echo "readlink OK: $(readlink -f "$HOME/.local/bin/skpp")" ;;
  *) echo "FAIL: link target = $(readlink -f "$HOME/.local/bin/skpp")"; exit 1 ;;
esac

# Version was injected (ldflags fired).
[ "$("$HOME/.local/bin/skpp" --version)" != "skpp dev" ] && echo "version-injected OK" || echo "FAIL: still dev"

# The symlink resolves example from OUTSIDE the repo (§13 symlink test, reproduced
# via the real PATH entry instead of a /tmp link).
cd /tmp
"$HOME/.local/bin/skpp" example | grep -q "/skills/example$" && echo "resolve-from-outside OK" || echo "FAIL"

# Idempotency: re-run must not error or duplicate.
./install.sh && ./install.sh && echo "idempotent OK"

# Optional env-override target (mirrors §13 SKPP_SKILLS_DIR test, but for the install target).
rm -rf /tmp/skpp-target; mkdir -p /tmp/skpp-target
SKPP_INSTALL_BIN=/tmp/skpp-target ./install.sh
test -L /tmp/skpp-target/skpp && /tmp/skpp-target/skpp example | grep -q "/skills/example$" && echo "env-target OK"
rm -rf /tmp/skpp-target

# Expected: all OK. (Cleans up after itself; leaves the real ~/.local/bin/skpp in place.)
```

### Level 4: Domain-Specific Validation (end-to-end with pi, §13 final block)

```bash
cd ~/projects/skpp

# The §13 pi end-to-end: skills load ONLY via --skill, never auto-discovered.
# (Requires `pi` on PATH; if absent, this level is skipped — it does not block
# the install.sh task, which is pure bash packaging.)
command -v pi >/dev/null 2>&1 || { echo "(pi not installed; skipping pi level — not required for install.sh)"; }

pi --no-skills --skill "$("$HOME/.local/bin/skpp" example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
# Expected: pi references the example skill / does not error. The --no-skills
# proves the skill is loaded solely via the explicit --skill path from the
# symlink-resolved binary, never via pi auto-discovery.
```

## Final Validation Checklist

### Technical Validation

- [ ] Level 1 passed: `bash -n install.sh` clean; `test -x install.sh`.
- [ ] Level 2 passed: build-piece prints non-`dev` version; go-missing branch
      prints the go.dev URL and exits 1 without creating a symlink.
- [ ] Level 3 passed: full `./install.sh`; `test -L` confirms symlink; `readlink
      -f` ends in `/skpp`; resolves `example` from `/tmp`; idempotent on re-run;
      `SKPP_INSTALL_BIN` override works.
- [ ] (Level 4) If `pi` present: `pi --no-skills --skill "$(skpp example)"` loads
      the example skill with no error.

### Feature Validation

- [ ] All Success Criteria in "What" met (8 checkboxes).
- [ ] `skpp --version` prints the SHA/tag (NOT `dev`) → ldflags fired.
- [ ] The PATH entry is a **symlink** to the **absolute** repo binary (the
      single most important assertion).
- [ ] `skpp example` resolves to `<repo>/skills/example` from any cwd.
- [ ] Error cases handled: missing `go` (exit 1 + URL), build failure (non-zero,
      no symlink), no writable target (exit 1 + sudo hint, no silent password).
- [ ] Re-running is idempotent (no duplicate PATH advice, symlink refreshed).
- [ ] No silent `sudo`; no auto-edit of rc files (only printed advice).

### Code Quality Validation

- [ ] Follows mcpeepants install-script *spirit* (banner, shell detection, verify
      block) while correctly DIVERGING (build + symlink, not cp).
- [ ] File placement: repo root (PRD §5); executable bit set.
- [ ] `set -euo pipefail` (or equivalent) so failures abort loudly.
- [ ] Streams: errors → stderr, progress → stdout (`install.sh 2>/dev/null` is
      quiet on success).
- [ ] Anti-patterns avoided (see below): no copy, no silent sudo, no rc auto-edit,
      no relative symlink, no `ln -sf` without `-n`.

### Documentation & Deployment

- [ ] Script header comment cites PRD §12.1 and notes it does NOT install
      completions (pointer to T15).
- [ ] Each of the 7 §12.1 steps is a clearly-commented section.
- [ ] The verify output tells the user to reload their shell + run `skpp example`.
- [ ] No new env vars introduced beyond `SKPP_INSTALL_BIN` (already spec'd in
      §12.1 step 4).

---

## Anti-Patterns to Avoid

- ❌ **Do NOT copy the binary** (`cp skpp ~/.local/bin/`). It breaks §8.2
  sibling-of-binary resolution silently. Always `ln -sfn`.
- ❌ Do NOT use `ln -sf` without `-n` — an existing symlink-to-dir dest gets
  dereferenced and the link lands inside it. Use `ln -sfn`.
- ❌ Do NOT make the symlink target relative — it lives in a different dir than
  the binary. Target must be the absolute `$SCRIPT_DIR/skpp`.
- ❌ Do NOT silently `sudo` or silently edit `~/.zshrc`/`~/.bashrc`. Print the
  exact command/snippet and let the user apply it.
- ❌ Do NOT escape the `$(` in the ldflags `$(git describe ...)` — bash must
  expand it inside the double-quoted `-ldflags` string.
- ❌ Do NOT swallow build failures (e.g. `go build ... || true`) — under `set -e`
  a failed build must abort before any symlink is created.
- ❌ Do NOT install completions in this script — that is task P1.M6.T15.S1 (PRD
  §14 allows deferral). A one-line pointer is fine; implementing them is scope creep.
- ❌ Do NOT print to stdout on a failed step (keep errors on stderr) so the script
  composes cleanly under redirection.
- ❌ Do NOT change `.gitignore`, `main.go`, or any `internal/` package — this task
  is additive packaging only; the binary's behavior is already correct.

---

## Confidence Score

**9 / 10.** The deliverable is a single ~80-line bash script whose every
statement is given verbatim above, the load-bearing requirement (symlink-not-copy)
is proven by `verified_symlink_resolution.md`, and the acceptance commands are
runnable as-is on this machine (go present, `~/.local/bin` on PATH and
writable). The one point of residual risk is cross-shell PATH-advice wording
(fish's `fish_add_path` vs `set -gx`) — but that branch only PRINTS guidance and
never blocks the install, so it cannot cause a silent failure. No Go knowledge
is required of the implementer; no codebase files must be read to WRITE the
script (only to verify it afterward).
