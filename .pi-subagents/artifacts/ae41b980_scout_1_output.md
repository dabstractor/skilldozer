# Compliant Files Verification — PRD §13 / P1.M4.T1.S1

Scope: Verify the four "compliant files" the PRD §13 acceptance grep-confirms.
All four PASS. Below: exact PRD requirement source, exact grep command the
acceptance script should use, observed line evidence, and any extras.

Authoritative run date: 2026-07-07.

---

## PRD requirement sources (verbatim)

### §16 `.gitignore` — exactly these 5 entries (heading h2.15, PRD.md L423-435)
```
/skilldozer
/dist
*.test
*.out
.DS_Store
```
(`/skilldozer` ignores the locally-built binary; everything else committed.)

### §12.1 `install.sh` (heading h3.11, PRD.md L319-331)
Step 3 ldflags (verbatim):
  `go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skilldozer .`
Step 5: **Symlink** (not copy) `<target>/skilldozer` → `<repo>/skilldozer`.
Step 7: Print verification command `skilldozer example`.

### §1 Goal + §4 Stack (PRD.md L9-69)
go.mod module = `github.com/dabstractor/skilldozer`; single dep `gopkg.in/yaml.v3 v3.0.1`.
LICENSE = MIT (§5 layout, PRD.md L70-79: `LICENSE  # MIT (match mcpeepants conventions)`).

---

## 1. `.gitignore` — PASS

PRD §16 requires all 5 of: `/skilldozer`, `/dist`, `*.test`, `*.out`, `.DS_Store`.

Observed (cat -A-free, line-numbered via `grep -n`):
- L2  `/skilldozer`
- L3  `/dist`
- L5  `*.test`
- L6  `*.out`
- L16 `.DS_Store`

All 5 present. **PASS.**

Exact acceptance grep (one per required pattern; anchors prevent partial matches):
```bash
grep -nE '^/skilldozer$'  .gitignore   # L2
grep -nE '^/dist$'        .gitignore   # L3
grep -nE '^\*\.test$'     .gitignore   # L5
grep -nE '^\*\.out$'      .gitignore   # L6
grep -nE '^\.DS_Store$'   .gitignore   # L16
```
Or a single compound assertion (exits 0 iff all 5 present):
```bash
grep -qE '^/skilldozer$' .gitignore \
  && grep -qE '^/dist$' .gitignore \
  && grep -qE '^\*\.test$' .gitignore \
  && grep -qE '^\*\.out$' .gitignore \
  && grep -qE '^\.DS_Store$' .gitignore \
  && echo "gitignore OK"
```

### Extras (NOT in PRD §16 — informational, not required to remove)
The PRD §16 block is illustrative ("everything else is committed") and does not
forbid additional ignore rules. The file additionally contains:
- `/build` (L4)
- `node_modules/` (L9)
- `venv/` (L10)
- `.env` (L13)
- `.pi-subagents/` (L17, agent runtime artifacts)
- comment headers (`# Build artifacts`, `# Dependency directories`,
  `# Environment files`, `# OS-specific files`,
  `# Agent runtime artifacts ...`)

Risk note: `.pi-subagents/` is a reasonable repo-hygiene addition; it does not
affect any §13 acceptance step. No action required unless a future acceptance
gate asserts exact-match of `.gitignore`.

---

## 2. `LICENSE` — PASS

`head -3 LICENSE`:
```
MIT License

Copyright (c) 2026 Dustin Schultz
```
Line 1 is exactly `MIT License`. **PASS.**

Exact acceptance grep:
```bash
head -3 LICENSE | grep -q 'MIT License' && echo "license OK"
# stricter (must be line 1):
[ "$(head -1 LICENSE)" = "MIT License" ] && echo "license line1 OK"
```

---

## 3. `go.mod` — PASS

`cat go.mod`:
```
module github.com/dabstractor/skilldozer   # L1
                                           # L3 blank
go 1.25                                    # L4
                                           # (blank)
require gopkg.in/yaml.v3 v3.0.1            # L5
```
- Module = `github.com/dabstractor/skilldozer` (exact, L1). **PASS.**
- Single `require` directive = `gopkg.in/yaml.v3 v3.0.1` (exact, L5). `grep -c '^require ' go.mod` = 1. **PASS.**
- `go 1.25` directive (L4) is standard go.mod toolchain metadata, not an extra dependency — allowed.

Exact acceptance grep:
```bash
grep -qE '^module github\.com/dabstractor/skilldozer$' go.mod \
  && [ "$(grep -c '^require ' go.mod)" = "1" ] \
  && grep -qE '^require gopkg\.in/yaml\.v3 v3\.0\.1$' go.mod \
  && echo "gomod OK"
```

---

## 4. `install.sh` — PASS (all 3 sub-checks)

Full file read (102 lines). Each sub-check:

### (a) `go build` with `-ldflags` containing `-X main.version` — PASS
L38-40:
```
go build -trimpath \
  -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
  -o skilldozer .
```
Matches PRD §12.1 step 3 verbatim (including `-trimpath`, `-s -w`, git-describe-with-`|| echo dev` fallback). **PASS.**
```bash
grep -qE 'go build' install.sh \
  && grep -qE -- '-ldflags' install.sh \
  && grep -qE -- '-X main\.version' install.sh \
  && echo "install ldflags OK"
```

### (b) Creates a symlink (`ln -s`) — PASS
L68 (the load-bearing line):
```
ln -sfn "$SCRIPT_DIR/skilldozer" "$TARGET/skilldozer"
```
`ln -sfn` is `ln -s` plus `-f` (force) and `-n` (no-dereference existing dir-link).
Symlink target = `$SCRIPT_DIR/skilldozer` (absolute, repo binary), link name =
`$TARGET/skilldozer`. PRD §12.1 step 5 satisfied. **PASS.**
```bash
# Require a real symlink command (not just an echo/comment). Accept ln -s / -sf / -sfn.
grep -qE 'ln -s(fn|f)? ' install.sh && echo "install symlink OK"
```

### (c) Prints `skilldozer example` as verification command — PASS
Two qualifying sites:
- L98: `"$TARGET/skilldozer" example` — actually runs the installed binary (pre-PATH-reload verification).
- L101: `echo "Done. Reload your shell (exec \$SHELL), then run:  skilldozer example"` — echoes the literal verification command string `skilldozer example`.

PRD §12.1 step 7 ("Print a verification command: `skilldozer example`") satisfied by L101 (and L98 runs it). **PASS.**
```bash
grep -qE 'skilldozer example' install.sh && echo "install verify-cmd OK"
```

---

## Summary

| # | File            | Required by       | Result |
|---|-----------------|-------------------|--------|
| 1 | `.gitignore`    | PRD §16           | PASS (5/5 patterns; extras present, allowed) |
| 2 | `LICENSE`       | PRD §1/§5         | PASS (line 1 = "MIT License") |
| 3 | `go.mod`        | PRD §1/§4         | PASS (module + single yaml.v3 v3.0.1 require) |
| 4 | `install.sh`    | PRD §12.1         | PASS (ldflags -X main.version L38-40; ln -sfn L68; verify cmd L98/L101) |

All four compliant files satisfy their PRD requirements. No changes needed.
The only non-blocking observation: `.gitignore` carries 5 extra ignore rules
(`/build`, `node_modules/`, `venv/`, `.env`, `.pi-subagents/`) beyond the PRD §16
illustrative block; PRD §16 does not require exact-match, so these are allowed.