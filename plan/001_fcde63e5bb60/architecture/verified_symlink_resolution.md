# Verified: Skills-dir resolution works as designed (§8.2)

Empirically tested with Go 1.26.4 on linux/amd64 before decomposition.

## Setup (simulated install scenario)

```
real binary:  /tmp/skpp-symlink-test/realbin2
symlink:      /tmp/skpp-bindir/skpp2  ->  /tmp/skpp-symlink-test/realbin2
```

Test program did `exe, _ := os.Executable()` then `real, _ := filepath.EvalSymlinks(exe)`
then `filepath.Dir(real)`.

## Result — running the symlink from a DIFFERENT directory

```
$ /tmp/skpp-bindir/skpp2
os.Executable()              = /tmp/skpp-symlink-test/realbin2   <-- REAL path, not symlink!
EvalSymlinks(os.Executable)  = /tmp/skpp-symlink-test/realbin2   <-- already resolved
filepath.Dir(real)           = /tmp/skpp-symlink-test            <-- REAL binary's dir
```

## Conclusion (verified)

1. **On Linux**, `os.Executable()` resolves the symlink via `/proc/self/exe` and
   returns the REAL binary path directly. So `filepath.Dir(os.Executable())`
   ALREADY points at the repo containing `skills/`.
2. **`EvalSymlinks` is redundant-but-harmless on Linux, and NECESSARY on macOS**
   (where `os.Executable()` may return the symlink path). The PRD's two-call
   sequence `os.Executable()` → `filepath.EvalSymlinks()` → `filepath.Dir()` is
   therefore CORRECT and cross-platform. Implement exactly that; do not "simplify"
   by dropping EvalSymlinks (breaks macOS).
3. This is why the **symlink install** (PRD §12.1) works: `~/.local/bin/skpp →
   ~/projects/skpp/skpp` means running `skpp` resolves back to
   `~/projects/skpp/skpp`, and `Dir()` = `~/projects/skpp`, which contains `skills/`.

## §8 priority order (confirmed implementable)

```
1. SKPP_SKILLS_DIR env   -> os.Stat(existing dir)? use as-is (no EvalSymlinks)
2. sibling-of-binary     -> Dir(EvalSymlinks(os.Executable())) + "/skills" exists?
3. walk-up from cwd      -> first ancestor with skills/ containing >=1 SKILL.md
4. none                  -> stderr + exit 1
```

All four steps are independently testable with `t.TempDir()` + `os.Symlink`.
The §13 acceptance commands (`skpp --path == $PWD/skills`, and the
`/tmp/skpp-bin/skpp example` cross-dir symlink test) both exercise steps 2/3.
