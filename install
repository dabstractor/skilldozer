#!/usr/bin/env bash
# install.sh — build skilldozer and symlink it into PATH (PRD §12.1).
#
# Mirrors mcpeepants QUICK_INSTALL.sh spirit (banner, shell detection, verify
# block) but does MORE: it BUILDS the binary with version ldflags and SYMLINKS
# it into a PATH dir (never copies — the symlink is load-bearing for §8.2
# sibling-of-binary skill resolution).
#
# Does NOT install completions: that is a separate task (P1.M6.T15.S1); PRD §14
# allows deferral. A pointer is printed at the end.
set -euo pipefail

die() { echo "ERROR: $*" >&2; exit 1; }

# --- §12.1 step 1: cd to the script's own dir (repo root) --------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 skilldozer install"
echo "Repo: $SCRIPT_DIR"
echo

# --- §12.1 step 2: verify go on PATH -----------------------------------------
# Exit BEFORE building; print install instructions to stderr.
if ! command -v go >/dev/null 2>&1; then
  cat >&2 <<'EOF'
ERROR: 'go' was not found on PATH.
Install Go from https://go.dev/doc/install, then re-run ./install.sh.
EOF
  exit 1
fi

# --- §12.1 step 3: build with version ldflags --------------------------------
# The $(git describe ...) expands INSIDE the double-quoted -ldflags string
# (Gotcha 4): do not escape the $. `|| echo dev` only fires outside a git repo.
# Under `set -e`, a build failure aborts here with go's own diagnostics; no
# symlink is created.
go build -trimpath \
  -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
  -o skilldozer .

# --- §12.1 step 4: pick target bin dir (first usable wins) -------------------
# Override → ~/.local/bin → /usr/local/bin (only if writable) → fail with hint.
# NO silent sudo (Gotcha 5): if root is required, print the exact command.
if [[ -n "${SKILLDOZER_INSTALL_BIN:-}" ]]; then
  TARGET="$SKILLDOZER_INSTALL_BIN"
  mkdir -p "$TARGET"
elif [[ -d "$HOME/.local/bin" ]] || [[ -w "$HOME" ]]; then
  TARGET="$HOME/.local/bin"
  mkdir -p "$TARGET"
elif [[ -w "/usr/local/bin" ]]; then
  TARGET="/usr/local/bin"
else
  cat >&2 <<EOF
ERROR: no writable install target found.
Re-run with: SKILLDOZER_INSTALL_BIN=/your/bin ./install.sh
Or (system-wide): sudo ln -sfn "$SCRIPT_DIR/skilldozer" /usr/local/bin/skilldozer
EOF
  exit 1
fi

# --- §12.1 step 5: SYMLINK (ln -sfn) $TARGET/skilldozer → $SCRIPT_DIR/skilldozer ---------
# THE load-bearing line (Gotchas 1–3):
#  - symlink, NEVER copy (cp breaks §8.2 sibling resolution silently)
#  - `ln -sfn`, not `ln -sf` (-n treats an existing symlink-to-dir dest as a
#    file; defensive even though our dest is a file)
#  - ABSOLUTE target ($SCRIPT_DIR/skilldozer); relative would resolve against $TARGET
ln -sfn "$SCRIPT_DIR/skilldozer" "$TARGET/skilldozer"

echo "Linked: $TARGET/skilldozer -> $SCRIPT_DIR/skilldozer"

# --- §12.1 step 6: ensure $TARGET on PATH; else PRINT rc-file snippet --------
# Detect shell via basename of $SHELL; PRINT only — never auto-edit rc files
# (Gotcha 6): auto-editing is intrusive and duplicates lines on re-run.
case ":${PATH:-}:" in
  *":$TARGET:"*) ;;  # already on PATH
  *)
    sh="$(basename "${SHELL:-}")"
    case "$sh" in
      bash)
        echo "Add to ~/.bashrc:  export PATH=\"$TARGET:\$PATH\"" ;;
      zsh)
        echo "Add to ~/.zshrc:   export PATH=\"$TARGET:\$PATH\"" ;;
      fish)
        echo "Add to ~/.config/fish/config.fish:  fish_add_path \"$TARGET\"" ;;
      *)
        echo "Add '$TARGET' to your PATH (your shell's rc file)." ;;
    esac
    ;;
esac

# --- §12.1 step 7: verify (absolute symlink path works pre-PATH-reload) ------
# Use the ABSOLUTE symlink path (Gotcha 8): it works even before the new PATH
# entry is live in the current shell; bare `skilldozer` may hit a stale hash until reload.
echo
echo "Verify:"
"$TARGET/skilldozer" --version
"$TARGET/skilldozer" example

echo
echo "Done. Reload your shell (exec \$SHELL), then run:  skilldozer example"
echo "(Shell completions are not installed by this script — see task P1.M6.T15.S1.)"
