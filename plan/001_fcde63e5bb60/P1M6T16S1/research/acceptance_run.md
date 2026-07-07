# Acceptance Run Transcript — P1.M6.T16.S1

Captured: 2026-07-07, repo root `/home/dustin/projects/skpp`, HEAD `3c4a68c`.
Environment: go1.26.4 linux/amd64, pi 0.80.3, bash/zsh/fish present. `SKPP_SKILLS_DIR`
unset for lines 3/11 (so §8 rule 2 sibling-of-binary is exercised, not rule 1).

**No regression found. No source files were modified.** The repo was green at
research time and remains green: every §13 line, every Go gate, and every M6
packaging smoke check passed on the first run. The only artifact produced by
this task is this transcript (under the orchestrator-owned `plan/` tree).

## §13 acceptance suite (12 lines, run verbatim)

=== §13 line 1: build ===
PASS  build
=== §13 line 2: --version ===
PASS  version ('skpp dev')
=== §13 line 3: --path = $PWD/skills (sibling-of-binary) ===
PASS  --path
=== §13 line 4: --list shows example ===
PASS  --list has example row
=== §13 line 5: resolve example dir exists ===
PASS  resolve dir
=== §13 line 6: -f example SKILL.md exists ===
PASS  -f file
=== §13 line 7: unknown-tag contract (stdout empty, rc 1) ===
PASS  unknown-tag (empty stdout, rc1)
=== §13 line 8: absolute-path contract ===
PASS  absolute path
=== §13 line 9: check (clean store → rc 0) ===
PASS  check rc0
=== §13 line 10: pi e2e (--no-skills + explicit --skill) ===
  (pi head output: Confirmed. The **example** skill is loaded from `/home/dustin/projects/skpp/skills/example/SKILL.md`. It's a reference/demo skill with frontmatter (`name: example`, `category: meta`) demonstrating how `skpp` resolves tags — safe to delete once you add real skills.)
PASS  pi e2e (skill referenced, no error)
=== §13 line 11: symlink install resolves back to repo ===
PASS  symlink→repo
=== §13 line 12: SKPP_SKILLS_DIR env override ===
PASS  env override

§13 RESULT: 12 passed, 0 failed
(runner exit 0)

## Go quality gates

```
$ go test ./...
ok  	github.com/dabstractor/skpp	0.014s
ok  	github.com/dabstractor/skpp/internal/check	(cached)
ok  	github.com/dabstractor/skpp/internal/discover	(cached)
ok  	github.com/dabstractor/skpp/internal/resolve	(cached)
ok  	github.com/dabstractor/skpp/internal/search	(cached)
ok  	github.com/dabstractor/skpp/internal/skillsdir	(cached)
ok  	github.com/dabstractor/skpp/internal/ui	(cached)
→ 7/7 packages ok, 0 failures        PASS

$ go vet ./...
(clean, exit 0)                       PASS

$ gofmt -l main.go internal/
(no output = all module source clean) PASS
```
Note: `gofmt -l .` (root) flags `plan/.../validate_example_probe.go`, a research
probe in the untracked planning tree, NOT module source — correctly ignored.

## §6-contract spot-checks (catch drift not covered by literal §13)

```
skpp example example   → 2 lines, input order        multi-tag OK
skpp --all             → 1 abs path/skill            --all OK
skpp --relative --all  → 'example'                   --relative --all OK
skpp -f --relative example → 'example/SKILL.md'      -f --relative OK
skpp --bogus           → empty stdout, exit 2        unknown-flag exit2 OK
skpp foo --list        → exit 2                      tags+mode exit2 OK
skpp check foo         → exit 2                      check+tags exit2 OK
skpp (no args)         → empty stdout, exit 1        no-args empty-stdout OK
skpp check (broken store via SKPP_SKILLS_DIR=/tmp/bk)
                       → ERROR lines, exit 1         check ERROR-detection OK
```
All spot-checks PASS. The `check` ERROR path was proven against a temp broken
store outside the repo (deleted after); the shipped store stays clean.

## M6 packaging smoke

```
bash install.sh                                 exit=0            PASS
  Linked: /home/dustin/.local/bin/skpp → $PWD/skpp               symlink OK
  verify line: $PWD/skills/example                              verify OK
  build WITH ldflags → --version reports 'skpp 3c4a68c'
command -v skpp && [ "$(skpp --path)" = "$PWD/skills" ]           installed --path OK
bash -n completions/skpp.bash                                    PASS
zsh -n completions/_skpp                                         PASS
fish -n completions/skpp.fish                                    PASS
→ completions syntax OK
```

## Level 4: pi end-to-end (§2 "not auto-discovered" proof)

§13 line 10 (verbatim):
`pi --no-skills --skill "$(./skpp example)" -p "briefly confirm the example skill is loaded"`

pi output (head): "Confirmed. The **example** skill is loaded from
`/home/dustin/projects/skpp/skills/example/SKILL.md`. ..." — references the
example skill, NO error. Under `--no-skills` pi discovery is OFF, so the skill
loads ONLY via the explicit `--skill` path skpp printed. §2 contract proven. PASS

## Regression found + fix

None. No regression was found; no code fix was required. `git status` shows no
source churn — only the orchestrator-owned `plan/` planning tree (untracked) and
the `plan/.../tasks.json` (orchestrator-owned) appear. The `skpp` binary at repo
root is correctly gitignored (§16) and was not staged.

## Final verdict

All 12 §13 acceptance lines PASS. `go test ./...` 7/7 ok. `go vet ./...` clean.
`gofmt -l main.go internal/` clean. `install.sh` exit 0 with symlink + verify.
All three completion files syntax-clean. §6-contract spot-checks green. pi e2e
loads the example skill under `--no-skills` via the explicit `--skill` path.

**Repo is GREEN and shippable. Acceptance gate P1.M6.T16.S1: PASS.**
