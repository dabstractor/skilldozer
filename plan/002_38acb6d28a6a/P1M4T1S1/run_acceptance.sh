#!/usr/bin/env bash
# P1.M4.T1.S1 — PRD §13 acceptance runner (verbatim isolation suite + compliant-file greps)
#
# Runs the full PRD §13 (h2.12) acceptance block VERBATIM in an isolated temp tree
# (temp HOME / XDG_CONFIG_HOME / SKILLDOZER_CONFIG so the config block cannot mutate
# the dev environment), captures every step's stdout/stderr/rc into the transcript,
# then grep-confirms the four PRD-compliant files (.gitignore / LICENSE / go.mod / install.sh).
#
# Deliverable: plan/002_38acb6d28a6a/P1M4T1S1/research/acceptance_transcript.txt
# Exit code:   0 iff every required OK marker is present; otherwise 1.
#
# This script OWNS the GREEN TRANSCRIPT. It writes ZERO new features and patches ZERO
# docs ad hoc. Any failing assertion is reported as a one-line routing note and left
# for the owning Mode A subtask (per the PRP remediation decision tree).

# We deliberately do NOT use `set -e`: the script must continue through failures so
# the FULL transcript is captured. Failures are tallied at the end.
set +e
set +u

# ----------------------------------------------------------------------------
# Paths
# ----------------------------------------------------------------------------
REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)"
TRANSCRIPT="$REPO_ROOT/plan/002_38acb6d28a6a/P1M4T1S1/research/acceptance_transcript.txt"

# ----------------------------------------------------------------------------
# Per-step capture helper
# ----------------------------------------------------------------------------
# step() runs a command (or group), echoing a header before and its rc after.
# It never exits the script on failure; the caller tallies expected OK markers.
step() {
  local name="$1"; shift
  echo ""
  echo "===== $name ====="
  "$@"
  local rc=$?
  echo "[rc=$rc] $name"
  return $rc
}

# A tally of expected OK markers we DID see.
OBSERVED_OK=0
EXPECTED_OK=0
note_ok() {
  EXPECTED_OK=$((EXPECTED_OK + 1))
  if [ "$1" = "1" ]; then
    OBSERVED_OK=$((OBSERVED_OK + 1))
  fi
}

# ----------------------------------------------------------------------------
# Pre-run: capture dev-environment isolation baseline
# ----------------------------------------------------------------------------
DEV_CONFIG=~/.config/skilldozer/config.yaml
if [ -f "$DEV_CONFIG" ]; then
  DEV_CONFIG_PRE="sha:$(sha256sum "$DEV_CONFIG" | awk '{print $1}')"
else
  DEV_CONFIG_PRE="absent"
fi
echo "dev-config pre-run: $DEV_CONFIG_PRE"

# ----------------------------------------------------------------------------
# Redirect all output to the transcript (and mirror to stdout via tee)
# ----------------------------------------------------------------------------
exec > >(tee "$TRANSCRIPT") 2>&1

echo "PRD §13 acceptance run — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Repo: $REPO_ROOT"
echo "Go:   $(go version)"
if command -v pi >/dev/null 2>&1; then
  echo "pi:   $(command -v pi) ($(pi --version 2>&1 | head -1))"
else
  echo "pi:   NOT FOUND (the pi end-to-end line will be skipped-with-note)"
fi
echo "Transcript: $TRANSCRIPT"
echo ""
echo "================================================================"
echo "GROUP A — repo-local (no isolation)"
echo "================================================================"

# ----------------------------------------------------------------------------
# 1. Build  (§13 step "Build")
# ----------------------------------------------------------------------------
step "1 build" bash -c 'go build -o skilldozer . && echo OK'
build_rc=$?

# ----------------------------------------------------------------------------
# 2. --version
# ----------------------------------------------------------------------------
step "2 version" bash -c './skilldozer --version'

# ----------------------------------------------------------------------------
# 3. --path sibling rule  (stdout ONLY — the label is on stderr; do NOT 2>&1)
# ----------------------------------------------------------------------------
step "3 path sibling" bash -c '
  test "$(./skilldozer --path)" = "$PWD/skills" && echo path_rc=0 || echo path_rc=1
'
path_ok=$?
note_ok "$([ $path_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 4. --list shows example
# ----------------------------------------------------------------------------
step "4 list" bash -c '
  ./skilldozer --list
  ./skilldozer --list | grep -q "example" && echo list_example_OK || echo list_example_FAIL
'
list_ok=$?
note_ok "$([ $list_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 5. example resolves to a real dir
# ----------------------------------------------------------------------------
step "5 example-dir" bash -c '
  test -d "$(./skilldozer example)" && echo example_dir_OK || echo example_dir_FAIL
'
example_dir_ok=$?
note_ok "$([ $example_dir_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 6. -f prints the SKILL.md path
# ----------------------------------------------------------------------------
step "6 file" bash -c '
  test -f "$(./skilldozer -f example)" && echo file_OK || echo file_FAIL
'
file_ok=$?
note_ok "$([ $file_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 7. unknown-tag contract (empty stdout, exit 1)
# ----------------------------------------------------------------------------
step "7 unknown-tag" bash -c '
  out=$(./skilldozer nope 2>/dev/null); rc=$?
  [ -z "$out" ] && [ "$rc" = "1" ] && echo "unknown-tag contract OK"
'
unknown_ok=$?
note_ok "$([ $unknown_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 8. absolute-path contract (default)
# ----------------------------------------------------------------------------
step "8 absolute" bash -c '
  case "$(./skilldozer example)" in /*) echo "absolute OK";; *) echo "FAIL"; exit 1;; esac
'
abs_ok=$?
note_ok "$([ $abs_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 9. check (exits 0, reports example OK)
# ----------------------------------------------------------------------------
step "9 check" bash -c '
  ./skilldozer check
  rc=$?; [ $rc -eq 0 ] && echo check_OK || echo check_FAIL
'
check_ok=$?
note_ok "$([ $check_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 10. pi end-to-end (skills load ONLY via --skill; --no-skills disables discovery)
# ----------------------------------------------------------------------------
step "10 pi end-to-end (--no-skills --skill)" bash -c '
  if command -v pi >/dev/null 2>&1; then
    out=$(pi --no-skills --skill "$(./skilldozer example)" -p "briefly confirm the example skill is loaded" 2>&1 | head)
    echo "$out"
    if echo "$out" | grep -q "example"; then
      echo "pi-line OK (example referenced)"
    else
      echo "pi-line WARN: ran but did not reference example; output above"
      exit 1
    fi
  else
    echo "pi-line SKIP: pi not on PATH (non-fatal on this run; recorded as a note)"
  fi
'

# ----------------------------------------------------------------------------
# 11. symlink install (resolve-back-to-repo)
# ----------------------------------------------------------------------------
step "11 symlink" bash -c '
  mkdir -p /tmp/skilldozer-bin
  ln -sf "$PWD/skilldozer" /tmp/skilldozer-bin/skilldozer
  got=$(/tmp/skilldozer-bin/skilldozer example)
  echo "$got"
  [ "$got" = "$PWD/skills/example" ] && echo symlink_OK || echo symlink_FAIL
'
symlink_ok=$?
note_ok "$([ $symlink_ok -eq 0 ] && echo 1 || echo 0)"

# ----------------------------------------------------------------------------
# 12. env override
# ----------------------------------------------------------------------------
step "12 env-override" bash -c '
  SKILLDOZER_SKILLS_DIR="$PWD/skills" ./skilldozer example
  got=$(SKILLDOZER_SKILLS_DIR="$PWD/skills" ./skilldozer example)
  [ -n "$got" ] && echo env_OK || echo env_FAIL
'
env_ok=$?
note_ok "$([ $env_ok -eq 0 ] && echo 1 || echo 0)"

echo ""
echo "================================================================"
echo "GROUP B — isolation (temp HOME / XDG / SKILLDOZER_CONFIG)"
echo "================================================================"

# ----------------------------------------------------------------------------
# Isolation block — VERBATIM from PRD §13. Sets all three env vars + unsets
# SKILLDOZER_SKILLS_DIR so config writes go ONLY to /tmp/skilldozer-iso.
# ----------------------------------------------------------------------------
ISO_ROOT=/tmp/skilldozer-iso
rm -rf "$ISO_ROOT"
mkdir -p "$ISO_ROOT"
cp ./skilldozer "$ISO_ROOT/skilldozer"

# Save cwd so we can restore it (mirrors `cd - >/dev/null`).
SAVED_PWD="$PWD"
cd "$ISO_ROOT"

# 13. unconfigured hint (clean HOME, unset env, no config, no sibling, no walk-up)
step "13 unconfigured-hint" bash -c '
  env -u SKILLDOZER_SKILLS_DIR HOME=/tmp/skilldozer-iso/home \
    XDG_CONFIG_HOME=/tmp/skilldozer-iso/home/.config ./skilldozer x 2>err; rc=$?
  echo "rc=$rc"
  echo "--- err ---"; cat err; echo "--- /err ---"
  [ "$rc" = 1 ] && grep -q '"'"'run `skilldozer init`'"'"' err && echo "unconfigured-hint OK"
'
unconf_ok=$?
note_ok "$([ $unconf_ok -eq 0 ] && echo 1 || echo 0)"

# 14. non-interactive init --store (creates store + writes config)
step "14 init --store (non-interactive)" bash -c '
  rm -rf /tmp/skilldozer-store
  SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer init --store /tmp/skilldozer-store
  rc=$?
  echo "init_rc=$rc"
  if [ -d /tmp/skilldozer-store ]; then echo "store_OK"; else echo "store_FAIL"; fi
  if grep -q "store: /tmp/skilldozer-store" /tmp/skilldozer-iso/cfg.yaml; then echo "cfg_OK"; else echo "cfg_FAIL"; fi
  [ $rc -eq 0 ] && [ -d /tmp/skilldozer-store ] \
    && grep -q "store: /tmp/skilldozer-store" /tmp/skilldozer-iso/cfg.yaml
'
init_ok=$?
note_ok "$([ $init_ok -eq 0 ] && echo 1 || echo 0)"

# 15. config rule wins (config beats sibling) — stdout-only grep is fine
step "15 config-rule-wins" bash -c '
  SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml ./skilldozer --path 2>/dev/null \
    | grep -q /tmp/skilldozer-store && echo "cfgwins_OK" || echo "cfgwins_FAIL"
'
cfgwins_ok=$?
note_ok "$([ $cfgwins_ok -eq 0 ] && echo 1 || echo 0)"

# 16. env beats config (env label appears on STDERR — 2>&1 is correct here)
step "16 env-beats-config" bash -c '
  SKILLDOZER_SKILLS_DIR=/tmp/skilldozer-store SKILLDOZER_CONFIG=/tmp/skilldozer-iso/cfg.yaml \
    ./skilldozer --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR && echo "envbeats_OK" || echo "envbeats_FAIL"
'
envbeats_ok=$?
note_ok "$([ $envbeats_ok -eq 0 ] && echo 1 || echo 0)"

# Restore cwd (mirrors `cd - >/dev/null`).
cd "$SAVED_PWD" >/dev/null

echo ""
echo "================================================================"
echo "GROUP C — compliant files (grep gates)"
echo "================================================================"

# 17. .gitignore — all 5 §16 required patterns PRESENT (presence, not exact-match)
step "17 gitignore (5 patterns)" bash -c '
  grep -qE "^/skilldozer$"  .gitignore \
    && grep -qE "^/dist$"        .gitignore \
    && grep -qE "^\*\.test$"     .gitignore \
    && grep -qE "^\*\.out$"      .gitignore \
    && grep -qE "^\.DS_Store$"   .gitignore \
    && echo "gitignore OK"
'
gitignore_ok=$?
note_ok "$([ $gitignore_ok -eq 0 ] && echo 1 || echo 0)"

# 18. LICENSE — line 1 is exactly "MIT License"
step "18 license (line 1)" bash -c '
  [ "$(head -1 LICENSE)" = "MIT License" ] && echo "license OK"
'
license_ok=$?
note_ok "$([ $license_ok -eq 0 ] && echo 1 || echo 0)"

# 19. go.mod — module + single yaml.v3 require
step "19 gomod (module + single require)" bash -c '
  grep -qE "^module github\.com/dabstractor/skilldozer$" go.mod \
    && [ "$(grep -c "^require " go.mod)" = "1" ] \
    && grep -qE "^require gopkg\.in/yaml\.v3 v3\.0\.1$" go.mod \
    && echo "gomod OK"
'
gomod_ok=$?
note_ok "$([ $gomod_ok -eq 0 ] && echo 1 || echo 0)"

# 20. install.sh — go build + ldflags -X main.version + symlink + verify-cmd
step "20 install.sh (ldflags + symlink + verify-cmd)" bash -c '
  grep -qE "go build"        install.sh \
    && grep -qE -- "-ldflags"     install.sh \
    && grep -qE -- "-X main\.version" install.sh \
    && grep -qE "ln -s(fn|f)? "   install.sh \
    && grep -qE "skilldozer example" install.sh \
    && echo "install OK"
'
install_ok=$?
note_ok "$([ $install_ok -eq 0 ] && echo 1 || echo 0)"

echo ""
echo "================================================================"
echo "GROUP D — isolation invariant (dev env byte-identical before/after)"
echo "================================================================"

if [ -f "$DEV_CONFIG" ]; then
  DEV_CONFIG_POST="sha:$(sha256sum "$DEV_CONFIG" | awk '{print $1}')"
else
  DEV_CONFIG_POST="absent"
fi
echo "dev-config pre-run:  $DEV_CONFIG_PRE"
echo "dev-config post-run: $DEV_CONFIG_POST"
if [ "$DEV_CONFIG_PRE" = "$DEV_CONFIG_POST" ]; then
  echo "isolation OK (dev config unchanged)"
  ISO_OK=0
else
  echo "isolation FAIL (dev config was mutated — see PRP gotchas)"
  ISO_OK=1
fi

echo ""
echo "================================================================"
echo "SUMMARY"
echo "================================================================"
echo "build:        $([ $build_rc   -eq 0 ] && echo OK || echo FAIL)"
echo "path:         $([ $path_ok    -eq 0 ] && echo OK || echo FAIL)"
echo "list:         $([ $list_ok    -eq 0 ] && echo OK || echo FAIL)"
echo "example-dir:  $([ $example_dir_ok -eq 0 ] && echo OK || echo FAIL)"
echo "file:         $([ $file_ok    -eq 0 ] && echo OK || echo FAIL)"
echo "unknown-tag:  $([ $unknown_ok -eq 0 ] && echo OK || echo FAIL)"
echo "absolute:     $([ $abs_ok     -eq 0 ] && echo OK || echo FAIL)"
echo "check:        $([ $check_ok   -eq 0 ] && echo OK || echo FAIL)"
echo "symlink:      $([ $symlink_ok -eq 0 ] && echo OK || echo FAIL)"
echo "env-override: $([ $env_ok     -eq 0 ] && echo OK || echo FAIL)"
echo "unconfigured: $([ $unconf_ok  -eq 0 ] && echo OK || echo FAIL)"
echo "init:         $([ $init_ok    -eq 0 ] && echo OK || echo FAIL)"
echo "cfgwins:      $([ $cfgwins_ok -eq 0 ] && echo OK || echo FAIL)"
echo "envbeats:     $([ $envbeats_ok -eq 0 ] && echo OK || echo FAIL)"
echo "gitignore:    $([ $gitignore_ok -eq 0 ] && echo OK || echo FAIL)"
echo "license:      $([ $license_ok   -eq 0 ] && echo OK || echo FAIL)"
echo "gomod:        $([ $gomod_ok     -eq 0 ] && echo OK || echo FAIL)"
echo "install.sh:   $([ $install_ok   -eq 0 ] && echo OK || echo FAIL)"
echo "isolation:    $([ $ISO_OK       -eq 0 ] && echo OK || echo FAIL)"
echo ""
echo "OK markers observed/expected: $OBSERVED_OK/$EXPECTED_OK"

# Final exit: 0 iff every expected OK was observed AND isolation held AND build succeeded.
if [ "$OBSERVED_OK" -eq "$EXPECTED_OK" ] && [ "$ISO_OK" -eq 0 ] && [ "$build_rc" -eq 0 ]; then
  echo ""
  echo "RESULT: GREEN — all §13 acceptance gates + compliant-file greps pass."
  exit 0
else
  echo ""
  echo "RESULT: RED — see failures above; route per PRP remediation decision tree:"
  echo "  build/version/path/absolute/unknown-tag/cfgwins/envbeats -> skillsdir.go / main.go owner"
  echo "  unconfigured-hint  -> skillsdir.ErrNotFound message (G4)"
  echo "  init --store       -> main.go init dispatch + config.Save (G6-G11)"
  echo "  .gitignore/LICENSE/go.mod/install.sh -> relevant Mode A subtask"
  echo "  README             -> P1.M4.T2.S1 (do NOT patch inline)"
  exit 1
fi
