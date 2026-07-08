# §13 Acceptance — Verification Run (P1.M4.T1.S1)

**Date:** 2026-07-07
**Repo:** /home/dustin/projects/skilldozer (branch `main`, 1 commit ahead of origin)
**Binary rebuilt:** yes (step 1 `go build -o skilldozer .` → OK, rc=0). mtime after build.
**Go toolchain:** go1.26.4-X:nodwarf5 linux/amd64
**Script:** the exact 15-step `set +e` block from the task, run verbatim. Output captured to `/tmp/sd-verify/output.txt`.

## TL;DR — 15/15 PASS

All §13 acceptance gates pass against a freshly rebuilt binary. No failures, no error attribution needed (source ownership noted below for the PRP).

| # | Gate | Result | Source owner |
|---|------|--------|--------------|
| 1 | `go build` | PASS | — |
| 2 | `--version` | PASS | `main.go:216` (`--version`/`-v`) |
| 3 | `--path` == `$PWD/skills` (sibling rule) | PASS | `internal/skillsdir/skillsdir.go` `Find`→`findSibling` / `main.go:177,223,460-490` |
| 4 | `--list` shows `example` | PASS | `main.go:179,225,488+` + `internal/discover/index.go` |
| 5 | `example` resolves to real dir | PASS | `internal/resolve/resolve.go` + `skillsdir.go` |
| 6 | `-f example` prints SKILL.md path | PASS | `main.go:229` (`--file`/`-f`) |
| 7 | unknown tag → empty stdout, rc=1 | PASS | `main.go` default tag-branch + resolve miss |
| 8 | `example` is absolute | PASS | `skillsdir.go` returns absolute paths |
| 9 | `check` exits 0 | PASS | `main.go:247` + `internal/check/check.go` |
| 10 | symlink install resolves back to repo | PASS | `skillsdir.go:167` `resolveSiblingFromExe` |
| 11 | `SKILLDOZER_SKILLS_DIR` env override | PASS | `skillsdir.go:76` `findEnv` (rule 1) |
| 12 | unconfigured hint + exit 1 | PASS | `skillsdir.ErrNotFound` → `main.go:461` |
| 13 | non-interactive `init --store` | PASS | `main.go:268,954-1011` |
| 14 | config rule wins over sibling | PASS | `skillsdir.go:106` `findConfig` (rule 2) |
| 15 | env beats config | PASS | `skillsdir.go:288` precedence: env → config → sibling → walk-up |

## Per-step detail (actual observed output)

### 1. build — PASS
```
== 1 build ==
OK
build_rc=0
```
`go build -o skilldozer .` succeeds; prints `OK`; rc=0.

### 2. version — PASS
```
== 2 version ==
skilldozer dev
```
`--version` prints `skilldozer dev` (PRD: "prints: skilldozer <something>"). Owner: `main.go:216`.

### 3. path sibling — PASS (rc=0)
```
== 3 path sibling ==
(found via sibling of binary)
path_rc=0
```
The `(found via sibling of binary)` line is on **stderr**. stdout is byte-exact:
```
/home/dustin/projects/skilldozer/skills
```
So `test "$(./skilldozer --path)" = "$PWD/skills"` holds (the stderr label does not pollute `$()`). Owner: `skillsdir.go` `Find`/`findSibling`; label printer `main.go:474-480` `fmt.Fprintf(stderr, "(found via %s)\n", src)`.

### 4. list — PASS
```
== 4 list ==
TAG      NAME     DESCRIPTION
example  example  Reference example skill for skilldozer.
                  Demonstrates the required frontmatter
                  and how skilldozer resolves a tag to a
                  absolute path. Safe to delete once you
                  add real skills.
```
The `example` skill is shown. Description is word-wrapped from the skill's own frontmatter (`skills/example/SKILL.md`).

### 5. example dir — PASS (rc=0)
`test -d "$(./skilldozer example)"` → `example_dir_rc=0`. Resolves to `/home/dustin/projects/skilldozer/skills/example`.

### 6. file — PASS (rc=0)
`test -f "$(./skilldozer -f example)"` → `file_rc=0`. `-f` prints the SKILL.md path: `/home/dustin/projects/skilldozer/skills/example/SKILL.md`.

### 7. unknown-tag — PASS
```
== 7 unknown-tag ==
rc=1 out=[]
```
`out` is byte-empty (`len=0`), rc=1. Matches PRD error contract: unknown tag prints nothing to stdout, exits 1.

### 8. absolute — PASS
```
== 8 absolute ==
abs_OK
```
`./skilldozer example` yields an absolute path (`/…`).

### 9. check — PASS
```
== 9 check ==
OK    example (example)
1 skills, 0 errors, 0 warnings
check_rc=0
```
`check` exits 0, reports `example` as OK.

### 10. symlink — PASS (rc=0)
```
== 10 symlink ==
/home/dustin/projects/skilldozer/skills/example
symlink_rc=0
```
Symlinked binary at `/tmp/sd-bin/skilldozer` still resolves back to the repo's `skills/example`. Owner: `skillsdir.go:167` `resolveSiblingFromExe` (reads the real executable path through the symlink).

### 11. env override — PASS (rc=0)
```
== 11 env override ==
/home/dustin/projects/skilldozer/skills/example
env_rc=0
```
`SKILLDOZER_SKILLS_DIR="$PWD/skills" ./skilldozer example` works. Owner: `skillsdir.go:76` `findEnv` (rule 1).

### 12. unconfigured hint — PASS
```
== 12 unconfigured hint ==
rc=1
hint_OK
```
Isolated run (clean `HOME`, unset env, no config, no sibling, no walk-up ancestor) exits rc=1 and the hint is present. `err` file contents:
```
skilldozer is not configured; run `skilldozer init`
```
`grep -q 'run \`skilldozer init\`' err` → match → `hint_OK`. Owner: `skillsdir.ErrNotFound` message printed at `main.go:461` `fmt.Fprintln(stderr, err)`.

### 13. non-interactive init — PASS
```
== 13 non-interactive init ==
Seeded example skill at /tmp/sd-store/example/SKILL.md
/tmp/sd-store
(found via config file)
OK    example (example)
1 skills, 0 errors, 0 warnings
init_rc=0
store_OK
cfg_OK
```
- `init_rc=0`
- `store_OK`: `/tmp/sd-store` created, containing `example/SKILL.md`.
- `cfg_OK`: `/tmp/sd-iso/cfg.yaml` written with exactly:
  ```
  store: /tmp/sd-store
  ```
Owner: `main.go:268` (`init` token) + `--store` flag at `main.go:258`; seeding at `main.go:954,1005`; config write nearby.

### 14. config rule wins — PASS
```
== 14 config rule wins ==
/tmp/sd-store
(found via config file)
cfgwins_OK
```
With only `SKILLDOZER_CONFIG` set (pointing at the cfg.yaml from step 13) and no env, `--path` reports `/tmp/sd-store` via `config file` — i.e. config beats the sibling rule. Owner: `skillsdir.go:106` `findConfig` (rule 2) + precedence in `Find` (`skillsdir.go:288`).

### 15. env beats config — PASS
```
== 15 env beats config ==
envbeats_OK
```
With both `SKILLDOZER_SKILLS_DIR=/tmp/sd-store` and `SKILLDOZER_CONFIG=/tmp/sd-iso/cfg.yaml` set, `--path`'s stderr label is `SKILLDOZER_SKILLS_DIR` (env wins). Owner: precedence order in `skillsdir.go` `Find` (env → config → sibling → walk-up), documented at `skillsdir.go:5-18` and `279-289`.

## Precedence proof (rules 1–4)
`internal/skillsdir/skillsdir.go` encodes the §8 order:
1. `findEnv` (`SKILLDOZER_SKILLS_DIR`, set + existing dir) — rule 1, `skillsdir.go:76`
2. `findConfig` (config `store` key) — rule 2, `skillsdir.go:106`
3. `findSibling` (sibling-of-binary, symlink-aware via `resolveSiblingFromExe`) — rule 3, `skillsdir.go:142,167`
4. `findWalkUp`/`findWalkUpAncestor` — rule 4, `skillsdir.go:231,256`
`Find` (`skillsdir.go:288`) tries them in order. Source labels (`Source.String()`, `skillsdir.go:47-58`): `SKILLDOZER_SKILLS_DIR`, `config file`, `sibling of binary`, `ancestor of cwd`.

## Notes / caveats
- The PRD §13 script also contains one **pi** end-to-end line (`pi --no-skills --skill "$(./skilldozer example)" …`). The 15-step task script does **not** include the pi invocation, so it was not exercised here. Recommend the PRP run that gate explicitly when `pi` is available, since PRD §13 says it "must pass" and must show the skill loaded under `--no-skills`.
- Working tree has **unstaged** changes only (`completions/*`, `plan/.../tasks.json`, untracked `plan/.../P1M3T2S1/`). **Nothing staged.** This verification run did not modify any tracked source.
- Build in step 1 rewrote the committed `skilldozer` binary artifact; that file is gitignored-adjacent? — it is a tracked/working-tree file but no `git add` was performed, so `noStagedFiles` holds.
