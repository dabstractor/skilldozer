# PRP — P1.M3.T8.S2: Modifiers `--file`/`-f`, `--relative`, `--all`/`-a`

> **Subtask:** P1.M3.T8.S2 — the second (final) subtask of T8 ("main tag-resolution
> path output + modifiers", PRD §6.1/§6.2/§6.4). This is milestone **M3** (Tag
> resolution & path output), build-order step 3. S1 (T8.S1) shipped the `<tag>`
> resolution branch with the §6.4 atomicity contract; **S2 layers three PRD §6.2
> modifiers on top of it** plus the §6.1 `--all` mode.
>
> **Scope:** **MODIFY** `main.go` (6 additive edits: +`path/filepath` import,
> +3 `config` fields, +3 `parseArgs` cases, +1 `--all` branch in `run`, +1 one-line
> change in the `<tag>` loop, +1 `skillPath` helper) and `main_test.go` (append 17
> new tests). **No new files.** No touch to `internal/discover/*`,
> `internal/skillsdir/*`, `internal/ui/*`, `internal/resolve/*`, `go.mod`,
> `go.sum`, or `PRD.md`.
>
> **DEPENDENCIES (hard, now SATISFIED):** everything S1 built is LANDED and green —
> `resolve.Resolve`/`Result` (T7), `discover.Index`/`discover.Skill` (T5/T4),
> `skillsdir.Find` (T2), and the S1 `<tag>` branch in `main.go` (commit `814bab4`).
> S2 consumes them READ-ONLY and adds only the output-formatting layer.
>
> **VERIFICATION STATUS (authoritative):** the EXACT source in the Implementation
> Blueprint has been **compiled, gofmt'd, vetted, unit-tested, and smoke-tested
> against a real built binary** in a throwaway `/tmp/skpp_s2_verify` (go1.25, a
> verbatim copy of the real module). See `research/verified_facts.md` for the full
> gate output: gofmt silent, vet clean, build OK, main package **53 tests pass**
> (was 36; +17 new), whole module green, `go.mod`/`go.sum` byte-identical, AND the
> ten end-to-end `SKPP_SKILLS_DIR` binary behaviors (`-f`, `--relative`, the
> `--file --relative` combine, `--all`/`-a`, `--all --file`, `--all --relative`,
> `--all` empty store exit 0, modifier-preserved atomicity, and the §13
> `test -f "$(skpp -f example)"` gate) all match PRD §6.1/§6.2/§6.4/§13.

---

## Goal

**Feature Goal**: Complete the M3 path-output surface. Add the three PRD §6.2 output
modifiers (`--file`/`-f`, `--relative`) and the PRD §6.1 `--all`/`-a` catalog mode to
`main.go`, so that `skpp` can emit (a) the `SKILL.md` file path instead of the skill
directory, (b) a path relative to the skills dir instead of an absolute one, or
(c) every skill's directory path in one shot — all built on the S1 buffered-atomicity
machinery, with the modifiers applied through one shared `skillPath()` formatter so
they behave identically across `<tag>` resolution and `--all`.

**Deliverable**: Two **MODIFIED** files (no new files):
1. `main.go` — add `"path/filepath"` import; add `all`, `file`, `relative` bool fields
   to `config`; add three `parseArgs` cases; add the `if c.all { ... }` branch in
   `run` (after `--list`, before `<tag>`); change one line in the `<tag>` loop to
   route through `skillPath`; add the `skillPath` helper.
2. `main_test.go` — append 17 new tests (6 `parseArgs` modifier tests + 4 `<tag>`-
   modifier tests + 7 `--all` tests). All 36 prior tests preserved.

**Success Definition**: `gofmt -l main.go main_test.go` is silent; `go vet ./...` is
clean; `go build ./...` and `go test ./...` pass (**main package 53 tests**: the
existing 36 + 17 new). `go mod tidy` is a **no-op** (`go.mod`/`go.sum` unchanged —
`path/filepath` is stdlib). A real built binary over a `SKPP_SKILLS_DIR` store emits
exactly the modifier-combined paths from §6.2 (`-f`→SKILL.md, `--relative`→relative,
`-f --relative`→relative SKILL.md), `--all` lists every dir sorted by tag, `--all`
on an empty store exits 0, and a bad tag still prints nothing to stdout with the
modifiers applied. The §13 acceptance gate `test -f "$(./skpp -f example)"` passes.
No touch to `internal/*`, `go.mod`, `go.sum`, `PRD.md`; no new files.

---

## Why

- **Closes the M3 "path output" loop.** S1 proved tag→dir resolution. S2 adds the
  §6.2 controls callers actually need: `pi --skill "$(skpp -f tag)"` (point at the
  file directly), machine-local relative paths for scripting, and `--all` for
  "enumerate my whole catalog" (`skpp --all | xargs -I{} ...`).
- **`--all` is the foundation for shell completions (P1.M6.T15).** PRD §14 says
  completions "invoke `skpp --all` ... for positional completion." Shipping `--all`
  here unblocks that later deliverable; it cannot exist without it.
- **The `skillPath` helper is the deliberate abstraction.** §6.2 says the modifiers
  "combine with tag resolution **or** `--all`." Rather than fork the formatting
  into two places, S2 extracts one formatter used by both loops — so the §6.2
  contract is enforced by construction and the S1 buffered-atomicity loop stays
  byte-for-byte intact.
- **It is go.mod-neutral.** The only new import is `path/filepath` (stdlib). No new
  third-party dependency; `go.mod`/`go.sum` are byte-identical.
- **It is the last M3 step.** With `--all`/`--file`/`--relative` in, the path-output
  surface is complete; M4 (`--search`/`check`) and M5 (`--help`/exit-codes/
  exclusivity) build on top of the now-final `<tag>` and `--all` loops.

---

## What

Six `main.go` edits (the complete verified source is in the Implementation Blueprint):

**`config`** — add three fields:
- `all bool` (`--all`/`-a`), `file bool` (`--file`/`-f`), `relative bool`
  (`--relative`).

**`parseArgs`** — add three `case` arms (any order within the switch; they are
exact-string matches):
- `case "--all", "-a": c.all = true`
- `case "--file", "-f": c.file = true`
- `case "--relative": c.relative = true`  (no short form per §6.2)

**`run`** — add the `--all` branch **after** `--list` and **before** the `if len(c.tags) > 0`
branch:
1. `skillsdir.Find()` → on `ErrNotFound`, stderr + exit 1 (identical to `--path`/`--list`).
2. `discover.Index(dir)` → on error, stderr + exit 1.
3. `for _, s := range skills { fmt.Fprintln(stdout, skillPath(s, dir, c)) }`
4. `return 0` — **always 0, even for an empty store** (PRD §6.1 `--all` is always
   exit 0; unlike `--list`, which exits 1 "if no skills found").

**`run` `<tag>` loop** — change ONE line:
- `paths = append(paths, res.Skill.Dir)` → `paths = append(paths, skillPath(res.Skill, dir, c))`
- Everything else in the loop (buffered atomicity, `hadErr`, `continue` on error,
  flush-only-on-full-success) is **unchanged**.

**`skillPath` helper** — new package-level function:
```go
func skillPath(s discover.Skill, skillsDir string, c config) string {
    p := s.Dir                                  // default: absolute dir (§6.1)
    if c.file { p = s.SourceFile }              // --file: SKILL.md path
    if c.relative {                             // --relative: path rel to skills dir
        if rel, err := filepath.Rel(skillsDir, p); err == nil { p = rel }
    }
    return p
}
```

**`main.go` imports** — add `"path/filepath"` (alphabetical, between `"os"` and `"strings"`).

### Success Criteria

- [ ] `config` has `all bool`, `file bool`, `relative bool`; `parseArgs` recognizes
      `--all`/`-a`, `--file`/`-f`, `--relative` (long only) in any position.
- [ ] `run` has an `if c.all { ... }` branch **after** `--list` and **before**
      `<tag>`; it calls `skillsdir.Find()` → `discover.Index()` → prints each skill
      via `skillPath`, and returns 0 even when the store is empty.
- [ ] The `<tag>` loop routes its append through `skillPath(res.Skill, dir, c)`; the
      buffered-atomicity contract is **unchanged** (any failure ⇒ nothing on stdout).
- [ ] `--file` swaps the printed path from `s.Dir` to `s.SourceFile` (`Dir + "/SKILL.md"`).
- [ ] `--relative` rewrites the path via `filepath.Rel(skillsDir, p)` (OS separator).
- [ ] `--file --relative` COMBINE: a SKILL.md path relative to the skills dir.
- [ ] `--all` output is **sorted by canonical tag** (the `discover.Index` order).
- [ ] `--list` is **unaffected** by `--file`/`--relative` (it prints a table).
- [ ] `--all` + empty store ⇒ exit 0, nothing printed.
- [ ] `--all` + unresolvable skills dir ⇒ exit 1, empty stdout, the one-line fix.
- [ ] `--version` precedes `--all` (PRD §6.3).
- [ ] `main.go` imports exactly: `fmt`, `io`, `os`, `path/filepath`, `strings`,
      `internal/{discover,resolve,skillsdir,ui}`.
- [ ] `gofmt -l main.go main_test.go` silent; `go vet ./...` clean; `go build ./...`
      + `go test ./...` pass (main = 53); `go mod tidy` no-op; `go.mod`/`go.sum`
      unchanged; nothing under `internal/` touched.

---

## All Needed Context

### Context Completeness Check

_Pass: the EXACT source for `main.go` (complete file) and the appended `main_test.go`
block are given verbatim in the Implementation Blueprint and have been **compiled,
gofmt'd, vetted, unit-tested, and smoke-tested against a real built binary** in
`/tmp/skpp_s2_verify` (go1.25, verbatim module copy). Every load-bearing decision is
documented in `research/verified_facts.md` and the Known Gotchas below: the
`skillPath` abstraction and why it is shared; the `--file`/`--relative` precedence
and combine semantics; why `filepath.Rel` cannot fail in practice; `--all`'s
empty-store exit 0 vs `--list`'s exit 1; the `--all` branch placement; why modifiers
do not affect `--list`; why modifiers do not break §6.4 atomicity; the M4/M5 scope
boundary. The consumed contracts — `discover.Skill` (`Dir`/`SourceFile`), `discover.Index`
(sorted), `resolve.Resolve`/`Result`, `skillsdir.Find` — are LANDED on disk and were
read in full. An implementer who knows Go but nothing about this repo can complete
this in one pass by applying the two edits verbatim._

### Documentation & References

```yaml
# MUST READ — this subtask's own empirical verification (every load-bearing decision)
- file: plan/001_fcde63e5bb60/P1M3T8S2/research/verified_facts.md
  why: "Proves (against a verbatim module copy on go1.25): (1) the 6 main.go edits
        compile + pass gofmt + vet; (2) the skillPath helper + the --all branch + the
        one-line <tag>-loop change work; (3) main goes 36→53 tests; (4) go.mod/go.sum
        byte-identical (no new dep — path/filepath is stdlib); (5) the ten end-to-end
        binary behaviors (-f, --relative, the combine, --all/-a, --all --file,
        --all --relative, --all empty→exit 0, modifier-preserved atomicity, and the
        §13 `test -f \"$(skpp -f example)\"` gate) all match §6.1/§6.2/§6.4/§13."
  critical: "Write the main.go in the blueprint verbatim (it is gofmt-clean AS
             WRITTEN; run `gofmt -w` to be safe — the only reformat is struct-field
             alignment). Append the test block verbatim. Do NOT add --search/check
             (M4) or --help/exit-2/exclusivity (M5)."

# PREDECESSOR PRP — S1's "DOWNSTREAM EXTENSION POINTS" that DESIGNED this contract
- file: plan/001_fcde63e5bb60/P1M3T8S1/PRP.md
  why: "S1's 'Integration Points > DOWNSTREAM EXTENSION POINTS > P1.M3.T8.S2' locked
        the S2 contract verbatim: '--file/-f: change ONE line — print
        res.Skill.SourceFile instead of res.Skill.Dir. Add file bool to config + a
        parseArgs case. The buffered atomicity loop is UNCHANGED. --relative: print
        a path RELATIVE to the skills dir. Compute once: rel, _ := filepath.Rel(dir,
        res.Skill.Dir). (Needs importing path/filepath in main.go — currently absent;
        S2 adds it.) --all/-a: a NEW mode that feeds discover.Index's already-sorted
        []Skill into the SAME path-printer.' This PRP implements EXACTLY that (the
        only refinement: S1 sketched rel, _ := filepath.Rel(...) inline; S2 extracts a
        shared skillPath() helper so --all and <tag> share one formatter)."
  section: "Integration Points > DOWNSTREAM EXTENSION POINTS"

# CONTRACT — discover.Skill (the Dir/SourceFile the modifiers switch between)
- file: internal/discover/skill.go
  why: "Skill.Dir is the ABSOLUTE skill dir; Skill.SourceFile is the ABSOLUTE
        SKILL.md path (== filepath.Join(Dir, \"SKILL.md\"), set by BuildSkill). These
        are the two values skillPath() chooses between (Dir by default, SourceFile
        under --file). Both are guaranteed ABSOLUTE and Clean'd by Index, so no
        re-cleaning is needed and filepath.Rel always has absolute inputs. READ-ONLY."
  pattern: "p := s.Dir; if c.file { p = s.SourceFile }"
  gotcha: "SourceFile is derived in BuildSkill (not passed by Index); it already
           exists on every Skill S1's loop produced — no discover change needed."

# CONTRACT — discover.Index (the SORTED list --all iterates)
- file: internal/discover/index.go
  why: "Index(dir) returns []Skill sorted by RelTag (PRD §6.1 --all 'sorted by tag'
        is FREE — Index already sort.Slices by RelTag). --all just walks it in order.
        An empty store yields ([], nil) — --all prints nothing and exits 0 (PRD §6.1).
        READ-ONLY."
  pattern: "skills, err := discover.Index(dir); for _, s := range skills { ... }"
  gotcha: "--all needs NO atomicity buffering: nothing can fail per-skill (Index
           builds HasFM=false Skills for malformed SKILL.md, never errors per-skill)."

# CONTRACT — resolve.Resolve / Result (unchanged from S1)
- file: internal/resolve/resolve.go
  why: "Resolve(tag, skills)(Result, error); Result.Skill is the discover.Skill whose
        Dir/SourceFile skillPath formats. S1's loop is unchanged; S2 only swaps the
        append target from res.Skill.Dir to skillPath(res.Skill, dir, c). READ-ONLY."
  pattern: "res, rerr := resolve.Resolve(tag, skills); ...; paths = append(paths, skillPath(res.Skill, dir, c))"

# CONTRACT — skillsdir.Find (the dir passed to skillPath as the --relative base)
- file: internal/skillsdir/skillsdir.go
  why: "Find()(dir, src, err); dir is ABSOLUTE (the base for filepath.Rel). The --all
        and <tag> branches already call Find() (from S1); S2 just threads `dir` into
        skillPath. READ-ONLY."
  pattern: "dir, _, err := skillsdir.Find(); ...; skillPath(s, dir, c)"

# CONTRACT — the dispatcher this subtask EXTENDS (read in full before editing)
- file: main.go
  why: "The LANDED main.go (S1, commit 814bab4): config{version,path,list,noColor,
        tags}; parseArgs switch; run with version→path→list→<tag>→default(1). S2 ADDS
        all/file/relative to config, three parseArgs cases, path/filepath import, the
        --all branch (after --list, before <tag>), the one-line skillPath call in the
        <tag> loop, and the skillPath function. The version/path/list/<tag>/default
        logic is PRESERVED verbatim. MODIFIED by this subtask (blueprint gives the
        complete new file)."
  gotcha: "Keep the branch ORDER: version → path → list → all → <tag> → default(1).
           Put --all AFTER --list and BEFORE <tag>. --version stays FIRST (§6.3)."

# CONTRACT — the architecture design (locks the output discipline)
- file: plan/001_fcde63e5bb60/architecture/go_architecture.md
  why: "'Data flow' + 'Output discipline (§6.4)'. The §6.4 buffered-atomicity rule
        (resolve ALL, buffer, flush only on full success, nothing on stdout on any
        failure) is S1's contract and is UNCHANGED by S2 — skillPath only reformats a
        path AFTER Resolve succeeds. 'Exit codes': --all is 0; unresolvable dir is 1."
  section: "Data flow", "Output discipline (§6.4)", "Exit codes"

# CONTRACT — the PRD sections this implements
- file: PRD.md
  why: "§6.1 `--all`/`-a`: 'All skills, directory paths. One absolute path per line
        (sorted by tag). exit 0.' §6.2: '--file/-f: Print the SKILL.md file path
        instead of the directory path. --relative: Print paths relative to the skills
        dir instead of absolute (machine-local convenience; default is absolute).'
        §6.2 header: modifiers 'combine with tag resolution or --all' (NOT --list).
        §6.3: --version precedence; <tag>+--all mutual-exclusivity is exit 2 (M5).
        §13 acceptance: `test -f \"$(./skpp -f example)\"`. §14: completions invoke
        `skpp --all`. READ-ONLY."
  critical: "§6.2 'combine with tag resolution or --all' is why skillPath is SHARED.
             §6.1 --all exit 0 (even empty) vs --list exit 1 'if no skills found' —
             do NOT copy --list's empty-store exit 1 into --all."

# REFERENCE — the repo's test convention (white-box main, injected writers, SKPP_SKILLS_DIR)
- file: main_test.go
  why: "Established convention: `package main` (white-box), `*bytes.Buffer` for
        stdout/stderr, `t.Setenv(\"SKPP_SKILLS_DIR\", dir)` (rule 1 wins), `t.Chdir(
        t.TempDir())` to force all §8 rules to miss, `writeSkillTree`/`unsetSkillsEnv`/
        `version`/`sampleStore` helpers, plain t.Errorf/t.Fatalf, NO testify, NO
        t.Parallel(). The new tests mirror this exactly and add NO new import.
        Relative-path assertions use filepath.FromSlash (cross-platform) since
        filepath.Rel emits OS separators. MODIFIED by this subtask (block appended)."

# URLS — the stdlib surface this subtask is built from
- url: https://pkg.go.dev/path/filepath#Rel
  why: "filepath.Rel(base, target) returns the relative path from base to target. Used
        by skillPath for --relative. With both args absolute (Dir/SourceFile from
        discover, dir from Find) and target under base (discovered by walking it), Rel
        always succeeds. The err guard is defensive only. Emits OS path separators,
        so tests compare via filepath.FromSlash."
- url: https://pkg.go.dev/fmt#Fprintln
  why: "fmt.Fprintln(w, p) writes p + newline — every output line (the --all loop and
        the buffered <tag> paths) uses it, giving the byte-exact 'path + newline' the
        §13 gates depend on."
```

### Current Codebase tree (M1 + M2 + resolve + S1 LANDED; before this subtask)

```bash
$ cd /home/dustin/projects/skpp && find . -name '*.go' -not -path './.pi-subagents/*' | sort
internal/discover/discover.go        # M2.T4.S1: Frontmatter + ParseFrontmatter
internal/discover/discover_test.go
internal/discover/skill.go           # M2.T4.S2: Skill{Dir,RelTag,...,SourceFile} + BuildSkill
internal/discover/skill_test.go
internal/discover/index.go           # M2.T5.S1: Index(dir) SORTED by RelTag
internal/discover/index_test.go
internal/skillsdir/skillsdir.go      # M1.T2: Source + Find + per-rule helpers
internal/skillsdir/skillsdir_test.go
internal/ui/ui.go                    # M2.T6.S1: PrintList table + ANSI
internal/ui/ui_test.go
internal/resolve/resolve.go          # M3.T7.S1: Resolve + Result + typed errors
internal/resolve/resolve_test.go
main.go                              # M1.T3 + M2.T6 + M3.T8.S1: version/path/list/<tag> dispatch  [THIS subtask MODIFIES]
main_test.go                         # M1.T3 + M2.T6 + M3.T8.S1: 36 tests                          [THIS subtask MODIFIES]

# go.mod: module github.com/dabstractor/skpp, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (DIRECT)
# baseline: go build ./... OK; go test ./... OK (discover + skillsdir + ui + resolve + main green)
# S1 <tag> branch LANDED (commit 814bab4). NO skills/ dir yet (P1.M6.T12 ships skills/example).
```

### Desired Codebase tree with files to be modified

```bash
skpp/
├── ... (go.mod, go.sum, .gitignore, LICENSE, PRD.md, internal/discover/*,
│        internal/skillsdir/*, internal/ui/*, internal/resolve/* — ALL UNCHANGED)
├── main.go         # MODIFY — +path/filepath import, +all/file/relative fields,
│                   #          +3 parseArgs cases, +--all branch, +skillPath() helper,
│                   #          +1 one-line change in the <tag> loop (res.Skill.Dir → skillPath)
└── main_test.go    # MODIFY — APPEND 17 new tests (parseArgs modifiers + tag modifiers + --all)
                    #          imports UNCHANGED (36 -> 53)
```

| File | Action | Responsibility | Imports added |
|---|---|---|---|
| `main.go` | MODIFY | Add `--all` mode + `--file`/`--relative` output modifiers via shared `skillPath` | `path/filepath` |
| `main_test.go` | MODIFY | Tests for the 3 flags, tag-modifier combinations, `--all` modes/edge cases | (none) |

**Two modified files. Zero new files. Zero changes to `go.mod`, `go.sum`,
`internal/*`, `PRD.md`. No `skills/`, `install.sh`, `README`, completions,
`--search`/`check`/`--help`.**

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — --file and --relative COMBINE; --file is applied FIRST, then --relative.
// skillPath sets p := s.Dir, then if c.file { p = s.SourceFile }, then if c.relative
// { p = filepath.Rel(skillsDir, p) }. So `-f --relative writing/reddit` yields
// "writing/reddit/SKILL.md" (the SKILL.md path, made relative). Applying them in the
// other order would be wrong: Rel on the dir then swapping to SourceFile loses the
// relativization. VERIFIED (binary smoke + TestRunTagFileRelativeCombine).
//   RIGHT: p := s.Dir; if c.file { p = s.SourceFile }; if c.relative { Rel }
//   WRONG: applying --relative before choosing Dir/SourceFile, or only-ever-rel'ing the dir.

// GOTCHA #2 — filepath.Rel inputs are BOTH ABSOLUTE; Rel cannot fail in practice.
// s.Dir and s.SourceFile are absolute (set by discover.Index via filepath.Abs +
// filepath.Join); skillsDir is absolute (from skillsdir.Find). And s.Dir is always
// UNDER skillsDir (discovered by walking it), so a clean relative path always exists.
// The `if rel, err := filepath.Rel(...); err == nil` guard is DEFENSIVE only: on a
// (theoretical, e.g. cross-device) failure it keeps the absolute path rather than
// crashing. Do NOT drop the guard (silently Rel-discard is fine; a panic is not).
// VERIFIED (binary smoke: --relative emits "example", "writing/reddit").
//   RIGHT: if rel, err := filepath.Rel(skillsDir, p); err == nil { p = rel }
//   WRONG: rel, _ := filepath.Rel(...); p = rel   // a non-nil err leaves rel == ""

// GOTCHA #3 — --all is exit 0 EVEN for an empty store (NOT copy --list's exit 1).
// PRD §6.1 table: `--all` is exit 0; `--list` is "1 if no skills found". They differ
// deliberately: --all is a scripting command (enumerate the catalog) where empty
// output + exit 0 is the useful shape; --list is human-facing and an empty catalog is
// a problem worth flagging. So the --all branch has NO `if len(skills)==0 { exit 1 }`
// guard — the for-range over an empty slice simply prints nothing. VERIFIED
// (TestRunAllEmptyStoreExit0).
//   RIGHT: for _, s := range skills { ... }; return 0
//   WRONG: mirroring --list's `if len==0 { stderr; return 1 }` into --all.

// GOTCHA #4 --all does NOT need atomicity buffering (nothing can fail per-skill).
// discover.Index builds a HasFM=false Skill for a malformed/no-frontmatter SKILL.md
// rather than erroring per-entry, so EVERY discovered skill dir is printable. There
// are no tags to resolve, so there is nothing to fail atomically. Just iterate and
// print. VERIFIED (TestRunAllPrintsAllDirsSorted). (The <tag> branch keeps S1's
// buffered atomicity — that loop is unchanged except the one skillPath call.)
//   RIGHT: for _, s := range skills { Fprintln(stdout, skillPath(s, dir, c)) }
//   WRONG: wrapping --all's loop in a buffer/flush-on-success dance (no failure mode).

// GOTCHA #5 --all output is SORTED by tag for FREE (discover.Index already sorts).
// Index sort.Slices by RelTag, so `--all` just walks the returned []Skill in order.
// Do NOT re-sort in main, and do NOT assume tag/input order. VERIFIED (binary smoke:
// example before writing/reddit). (The <tag> branch is INPUT order, NOT sorted —
// distinct from --all; do not unify the two orderings.)
//   RIGHT: iterate discover.Index's slice verbatim.
//   WRONG: sort again, or sorting the <tag> output (PRD §6.1: tags are input order).

// GOTCHA #6 — --all branch goes AFTER --list and BEFORE <tag>.
// run dispatch order: version → path → list → all → <tag> → default(1). This groups
// the named modes (path/list/all); <tag> is the positional fallback. When --all and
// <tag> coexist (degenerate), --all wins deterministically in S2; P1.M5.T11 turns
// <tag>+--all into exit 2 (§6.3 mutual-exclusivity). --list still wins over --all
// (checked first) if both are given — also tolerated until M5. VERIFIED (the S1
// list/tag tests are unchanged; TestRunAll* exercises --all).
//   RIGHT: ... if c.list {...} if c.all {...} if len(c.tags)>0 {...} return 1
//   WRONG: putting --all before --list, or after <tag>.

// GOTCHA #7 — Modifiers do NOT affect --list (--list prints a TABLE, not paths).
// PRD §6.2 header: modifiers "combine with tag resolution or --all" — --list is
// absent. The --list branch is untouched by S2 (it calls ui.PrintList, which renders
// the catalog table). If --list + --file are both given, --file is silently ignored
// (tolerated; M5 may surface an error). VERIFIED (all 6 existing --list tests pass).
//   RIGHT: leaving the --list branch exactly as S1 left it.
//   WRONG: threading skillPath into --list, or erroring on --list+--file now.

// GOTCHA #8 — Modifiers do NOT break §6.4 atomicity (the S1 contract still holds).
// skillPath runs AFTER resolve.Resolve succeeds, so it only reformats an already-good
// path. The buffered-atomicity loop (buffer paths, set hadErr + continue on error,
// flush only if !hadErr) is byte-for-byte the S1 loop — the only change is the append
// target. So `skpp -f example nope` still prints NOTHING to stdout and exits 1.
// VERIFIED (TestRunTagFileAtomicity + binary smoke).
//   RIGHT: paths = append(paths, skillPath(res.Skill, dir, c))  // inside the buffer loop
//   WRONG: formatting/printing to stdout inside the resolve loop (breaks atomicity).

// GOTCHA #9 — --relative has NO short form; --file/-f and --all/-a do.
// PRD §6.2 lists only `--relative` (no `-r`). parseArgs has a case for `"--file","-f"`
// and `"--all","-a"` but only `"--relative"` (single string). A future `-r` is NOT
// reserved (PRD does not assign it). VERIFIED (TestParseArgsRelativeLong only).
//   RIGHT: case "--relative": c.relative = true
//   WRONG: case "--relative", "-r": ...   // inventing a short form PRD doesn't define.

// GOTCHA #10 — filepath.Rel emits OS path separators; tests compare via FromSlash.
// On Linux Rel returns "writing/reddit"; on Windows it would return "writing\reddit".
// main.go does NOT normalize (it returns whatever Rel gives — machine-local is the
// point of --relative per §6.2). Tests must compare with filepath.FromSlash("writing/
// reddit") so they pass on every OS. main itself never needs ToSlash on these paths
// (§6.2 wants the local separator; §7.1's ToSlash normalization is for the canonical
// TAG, a different concern). VERIFIED (tests use FromSlash; binary smoke on Linux).
//   RIGHT (test): want := filepath.FromSlash("writing/reddit") + "\n"
//   WRONG (test): want := "writing/reddit\n"   // fails on Windows

// GOTCHA #11 — go.mod/go.sum are UNCHANGED.
// The only new import is path/filepath (STDLIB). No new third-party dependency.
// `go mod tidy` is a no-op. If it changes anything, something is wrong (you added an
// external import). VERIFIED (research §2: go.mod/go.sum byte-identical). Confirm
// with `go mod tidy && git diff --quiet go.mod go.sum`.

// GOTCHA #12 — Reuse the existing test helpers; do NOT add new imports to main_test.go.
// The new tests reuse `writeSkillTree`, `unsetSkillsEnv`, `version`, and `sampleStore`
// (the helper S1 added), plus bytes/io/os/path/filepath/strings/testing already
// imported. Add NO new import. NO testify; NO t.Parallel(). VERIFIED (main_test.go
// imports unchanged; compiles clean).
```

---

## Implementation Blueprint

### Data model — `config` gains three fields; one new helper function

```go
type config struct {
	version  bool     // --version / -v
	path     bool     // --path / -p
	list     bool     // --list / -l
	all      bool     // --all / -a     : print every skill's directory path (§6.1)   [NEW]
	file     bool     // --file / -f    : print the SKILL.md path instead of the dir (§6.2) [NEW]
	relative bool     // --relative     : print paths relative to the skills dir (§6.2) [NEW]
	noColor  bool     // --no-color
	tags     []string // positional <tag> args (PRD §6.1); resolved in run
}
```

```go
// skillPath returns the path to print for a resolved skill, applying §6.2 modifiers.
func skillPath(s discover.Skill, skillsDir string, c config) string {
	p := s.Dir                                  // default: absolute dir (§6.1)
	if c.file { p = s.SourceFile }              // --file: SKILL.md path
	if c.relative {                             // --relative: relative to skills dir
		if rel, err := filepath.Rel(skillsDir, p); err == nil { p = rel }
	}
	return p
}
```

### File 1 — `main.go` (MODIFY — write this complete file over the existing one)

The complete updated `main.go` (gofmt-clean; all 53 main tests pass; **compiled +
vetted + smoke-tested** against the real module in `/tmp/skpp_s2_verify`). It
preserves the S1 version/path/list/`<tag>`/default logic verbatim and adds the six
S2 changes (path/filepath import, three config fields, three parseArgs cases, the
`--all` branch, the one-line `<tag>`-loop change, and the `skillPath` helper):

````go
// Command skpp resolves skill tags to on-disk skill directory paths.
//
// main.go is the entrypoint: it parses argv, applies PRD §6 precedence
// (--version/--help win over everything), and dispatches to the matching mode.
// Wired so far (grown milestone by milestone): --version/--path (M1.T3),
// --list (M2.T6), <tag> resolution (M3.T8.S1), and the --file/--relative/--all
// modifiers (M3.T8.S2). Every other §6 flag is added by later milestones (M4
// --search/check, M5 --help + exit codes). The arg parser is intentionally a
// small hand-rolled switch (not Go's `flag` package) so the full §6 matrix —
// subcommands like `check`, positional <tag> args, long+short aliases, and
// §6.3 mutual exclusivity — can be expressed cleanly. See
// plan/001_fcde63e5bb60/P1M1T3.S1/research/verified_facts.md §5.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dabstractor/skpp/internal/discover"
	"github.com/dabstractor/skpp/internal/resolve"
	"github.com/dabstractor/skpp/internal/skillsdir"
	"github.com/dabstractor/skpp/internal/ui"
)

// version is the skpp version string, printed by `skpp --version`. It is
// overridden at BUILD time via ldflags (PRD §12.1 build command):
//
//	go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o skpp .
//
// The default "dev" is used by `go run` and plain `go build` (no ldflags).
//
// IMPORTANT: this MUST be a package-level var, not a const. `-X main.version=...`
// rewrites a package-scope string var at link time; it cannot override a const
// (compile error) or a function-local. Because this file is `package main`, the
// linker symbol path is `main.version` (NOT the module import path).
var version = "dev"

// isTerminal reports whether w is an interactive terminal (a character device).
// It decides whether --list/--search emit ANSI color by default (PRD §6.2: color
// is on for a TTY unless --no-color is set). It type-asserts w to *os.File and
// checks the ModeCharDevice bit, so a *bytes.Buffer (tests) or a pipe/redirect
// correctly yields false -> no color, keeping output deterministic and pipe-safe.
//
// It is a package var so tests can override it to exercise the color-enabled path
// through run() without a real terminal. NOT safe for t.Parallel (mutates package
// state); the repo convention is no t.Parallel() on such tests anyway.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// config holds the parsed CLI flags. Grown by later milestones as more of the
// PRD §6.1/§6.2 matrix lands. This version sets version, path, list, noColor,
// tags, and the S2 modifiers all/file/relative; every other token is a tolerated
// no-op (P1.M5.T11 turns unknown flags into exit 2 and adds subcommand handling).
type config struct {
	version  bool     // --version / -v : print "skpp <version>" and exit 0
	path     bool     // --path / -p    : print resolved skills dir and exit 0/1
	list     bool     // --list / -l    : print the human-readable catalog table (§6.1)
	all      bool     // --all / -a     : print every skill's directory path, one per line (§6.1) [NEW]
	file     bool     // --file / -f    : print the SKILL.md path instead of the dir path (§6.2) [NEW]
	relative bool     // --relative     : print paths relative to the skills dir, not absolute (§6.2) [NEW]
	noColor  bool     // --no-color     : disable ANSI color even on a TTY (§6.2)
	tags     []string // positional <tag> args (PRD §6.1 `skpp <tag> [<tag>...]`); resolved in run
	// Future (M4/M5), do NOT add yet:
	//   search string; check bool; help bool
}

// parseArgs scans argv tokens and fills a config. Flags may appear in any order
// (PRD §6). Long forms use POSIX double-dash; short forms a single dash. Unknown
// dashed flags are tolerated for now (a no-op in the default branch); the full
// unknown-flag -> exit 2 behavior and §6.3 mutual-exclusivity land in P1.M5.T11.
//
// To add a flag in a later milestone: append a `case "--name", "-n": cfg.name =
// true` (or capture the next arg for value-taking flags like --search <q>).
func parseArgs(args []string) config {
	var c config
	for _, a := range args {
		switch a {
		case "--version", "-v":
			c.version = true
		case "--path", "-p":
			c.path = true
		case "--list", "-l":
			c.list = true
		case "--all", "-a":
			c.all = true
		case "--file", "-f":
			c.file = true
		case "--relative":
			c.relative = true
		case "--no-color":
			c.noColor = true
		default:
			// Positional <tag> (PRD §6.1 `skpp <tag> [<tag>...]`): a token that
			// does NOT start with '-' is a tag, captured here and resolved in run.
			// Dashed unknowns (e.g. --frobnicate) also fall through to this default
			// and are tolerated (no-op); P1.M5.T11 turns them into exit 2 and adds
			// §6.3 mutual-exclusivity (tag mixed with --list/--search/--all).
			if !strings.HasPrefix(a, "-") {
				c.tags = append(c.tags, a)
			}
		}
	}
	return c
}

// run is the testable dispatcher. It returns the process exit code so main() can
// call os.Exit(run(...)) without tests ever invoking os.Exit. stdout/stderr are
// injected so tests capture output via *bytes.Buffer instead of the real streams.
//
// Exit codes (PRD §6; this subtask's slice):
//   - 0: --version printed; --path succeeded; --list printed the catalog; all
//     <tag>s resolved (one absolute path per line printed); --all printed the store
//   - 1: --path/--list failed; ANY <tag> unresolved/ambiguous (nothing on stdout);
//     skills dir unresolvable; default (no recognized flag)
//   - 2: (DEFERRED to P1.M5.T11) unknown flag / mutually-exclusive modes mixed
//
// Precedence (PRD §6.3): --version (and, in M5, --help) win over everything.
func run(args []string, stdout, stderr io.Writer) int {
	c := parseArgs(args)

	// Precedence tier: --version wins over every other flag (PRD §6.3).
	// P1.M5.T11 adds --help/-h to this same tier (before --path).
	if c.version {
		fmt.Fprintf(stdout, "skpp %s\n", version)
		return 0
	}

	if c.path {
		dir, _, err := skillsdir.Find() // src is for reporting only; not printed
		if err != nil {
			// Find() returns skillsdir.ErrNotFound whose message is the
			// user-facing one-line fix (PRD §8.4/§6.4). Print it verbatim to
			// stderr (NOT stdout) so $(...) stays empty on failure.
			fmt.Fprintln(stderr, err)
			return 1
		}
		// Byte-exact: ONLY the dir + newline. The §13 acceptance gate
		// `test "$(./skpp --path)" = "$PWD/skills"` depends on this.
		fmt.Fprintln(stdout, dir)
		return 0
	}

	if c.list {
		// PRD §6.1 `skpp --list`: resolve the store, build the index, render the
		// table. This is the FIRST place the Find() -> discover.Index() data flow
		// is wired end-to-end (M2.T6). Exit 1 on any failure path.
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // verbatim one-line fix (PRD §6.4/§8.4)
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		if len(skills) == 0 {
			// PRD §6.1: --list exits 1 "if no skills found". Message to stderr so
			// stdout stays clean for any consumer.
			fmt.Fprintln(stderr, "no skills found in "+dir)
			return 1
		}
		// Color only when stdout is a TTY AND --no-color was not given (PRD §6.2).
		// A *bytes.Buffer (tests) / pipe / file is not a TTY -> plain output.
		// Note: --list prints a TABLE, so the --file/--relative path modifiers do
		// NOT apply to it (PRD §6.2 header: modifiers combine with tag resolution
		// or --all, not --list).
		ui.PrintList(stdout, skills, isTerminal(stdout) && !c.noColor)
		return 0
	}

	// --all mode: print every skill's directory path, one per line, SORTED by
	// canonical tag (PRD §6.1). discover.Index already sorts []Skill by RelTag, so
	// this just walks the index in order. Exit 0 even for an empty store (PRD §6.1
	// `--all` is always exit 0, unlike --list which exits 1 "if no skills found" —
	// --all is a scripting command where empty output + exit 0 is the useful shape).
	// The --file/--relative modifiers apply here too (PRD §6.2 header: "combine with
	// tag resolution or --all"), via the shared skillPath() helper.
	if c.all {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		for _, s := range skills {
			fmt.Fprintln(stdout, skillPath(s, dir, c)) // absolute dir by default; --file/--relative apply
		}
		return 0
	}

	// Tag-resolution mode: `skpp <tag> [<tag>...]` (PRD §6.1). Resolves each tag to
	// its skill path and prints one path per line, in input order.
	//
	// ATOMICITY (PRD §6.4 — the critical-for-$(...) contract): resolve EVERY tag
	// first, buffering the resulting paths; if ANY tag fails (unknown/ambiguous),
	// print one error line per problem tag to stderr, print NOTHING to stdout, and
	// exit 1. The buffered paths are flushed ONLY when the whole invocation is
	// known-good. This makes `pi --skill "$(skpp bad)"` fail loudly (empty $(),
	// exit 1) instead of passing a partial or garbage path. Each error is printed
	// verbatim from resolve's typed errors — UnknownError names the tag,
	// AmbiguousError lists the candidate full tags (no "skpp:" prefix, matching the
	// skillsdir.ErrNotFound convention used by --path/--list). The default output is
	// the skill DIRECTORY path; --file swaps to the SKILL.md path and --relative
	// makes it relative to the skills dir (applied by skillPath, PRD §6.2).
	if len(c.tags) > 0 {
		dir, _, err := skillsdir.Find()
		if err != nil {
			fmt.Fprintln(stderr, err) // one-line fix (PRD §6.4/§8); stdout stays empty
			return 1
		}
		skills, err := discover.Index(dir)
		if err != nil {
			fmt.Fprintln(stderr, err) // e.g. skills dir vanished between Find and Index
			return 1
		}
		paths := make([]string, 0, len(c.tags)) // buffered; flushed only if all resolve
		hadErr := false
		for _, tag := range c.tags {
			res, rerr := resolve.Resolve(tag, skills)
			if rerr != nil {
				fmt.Fprintln(stderr, rerr) // one error line per problem tag (verbatim)
				hadErr = true
				continue
			}
			// skillPath applies --file (SourceFile vs Dir) and --relative (Rel to
			// skills dir); default is the absolute dir (PRD §6.1/§6.2).
			paths = append(paths, skillPath(res.Skill, dir, c))
		}
		if hadErr {
			return 1 // paths buffered but never written → stdout empty (§6.4)
		}
		for _, p := range paths {
			fmt.Fprintln(stdout, p) // one path per line, input order
		}
		return 0
	}

	// No recognized mode. PRD §6.3 no-args behavior is "usage to stderr, exit 1";
	// the usage text and the unknown-flag -> exit 2 rule both land in P1.M5.T11.
	// For now, exit 1 silently (matches the eventual no-args code) so unknown
	// flags are "tolerated" (not exit 2) per this subtask's contract.
	return 1
}

// skillPath returns the path to print for a resolved skill, applying the PRD §6.2
// --file and --relative modifiers. It is the shared formatter used by BOTH the
// <tag>-resolution loop and the --all loop, so the modifiers behave identically in
// the two modes (PRD §6.2 header: "combine with tag resolution or --all").
//
// Precedence of effects:
//   - default (neither flag): the ABSOLUTE skill DIRECTORY path (s.Dir) — PRD §6.1.
//   - --file:                 the ABSOLUTE SKILL.md file path (s.SourceFile = s.Dir
//     + "/SKILL.md") — PRD §6.2.
//   - --relative:             the chosen path rewritten to be RELATIVE to the
//     skills dir (PRD §6.2 "machine-local convenience").
//   - --file --relative:      they COMBINE — a SKILL.md path relative to the skills
//     dir (e.g. "writing/reddit/SKILL.md").
//
// filepath.Rel cannot fail in practice here: both arguments are ABSOLUTE (s.Dir /
// s.SourceFile are set absolute by discover.Index; skillsDir is absolute from
// skillsdir.Find), and s.Dir is always UNDER skillsDir (it was discovered by
// walking it), so a clean relative path always exists. The err guard is defensive
// only: on a (theoretical) Rel failure skpp falls back to the absolute path, which
// is still a correct, usable answer rather than crashing.
func skillPath(s discover.Skill, skillsDir string, c config) string {
	p := s.Dir // default: absolute skill directory (PRD §6.1/§6.2)
	if c.file {
		p = s.SourceFile // --file: the SKILL.md file path (s.Dir + "/SKILL.md")
	}
	if c.relative {
		if rel, err := filepath.Rel(skillsDir, p); err == nil {
			p = rel // --relative: path relative to the skills dir
		}
	}
	return p
}
````

### File 2 — `main_test.go` (MODIFY — APPEND the block below to the end of `main_test.go`)

The new tests are appended verbatim after the existing `TestRunVersionPrecedenceOverTag`
(the last S1 test). **`main_test.go` imports are UNCHANGED** (bytes/io/os/path/filepath/
strings/testing); all tests reuse the existing `writeSkillTree`/`unsetSkillsEnv`/
`version`/`sampleStore` helpers. Relative-path assertions use `filepath.FromSlash`
(cross-platform) since `filepath.Rel` emits OS separators.

````go

// --- parseArgs: modifiers --file/-f, --relative, --all/-a (P1.M3.T8.S2) ---

// --file/-f sets c.file (long and short forms, PRD §6.2).
func TestParseArgsFileLong(t *testing.T) {
	c := parseArgs([]string{"--file"})
	if !c.file {
		t.Errorf("parseArgs(--file): file=false; want true")
	}
}

func TestParseArgsFileShort(t *testing.T) {
	c := parseArgs([]string{"-f"})
	if !c.file {
		t.Errorf("parseArgs(-f): file=false; want true")
	}
}

// --relative has NO short form (PRD §6.2 lists only the long form).
func TestParseArgsRelativeLong(t *testing.T) {
	c := parseArgs([]string{"--relative"})
	if !c.relative {
		t.Errorf("parseArgs(--relative): relative=false; want true")
	}
}

// --all/-a sets c.all (long and short forms, PRD §6.1).
func TestParseArgsAllLong(t *testing.T) {
	c := parseArgs([]string{"--all"})
	if !c.all {
		t.Errorf("parseArgs(--all): all=false; want true")
	}
}

func TestParseArgsAllShort(t *testing.T) {
	c := parseArgs([]string{"-a"})
	if !c.all {
		t.Errorf("parseArgs(-a): all=false; want true")
	}
}

// Modifiers may interleave with tags and other flags (PRD §6 any order).
func TestParseArgsModifiersInterleave(t *testing.T) {
	c := parseArgs([]string{"-f", "example", "--relative"})
	if !c.file || !c.relative || len(c.tags) != 1 || c.tags[0] != "example" {
		t.Errorf("config=%+v; want file+relative true and tags=[example]", c)
	}
}

// --- run: <tag> + --file/--relative modifiers (P1.M3.T8.S2) ---

// --file prints the ABSOLUTE SKILL.md path instead of the dir (PRD §6.2). The §13
// gate `test -f "$(./skpp -f example)"` depends on this printing a real file.
func TestRunTagFilePrintsSourceFile(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-f", "example"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-f example): code=%d; want 0", code)
	}
	want := filepath.Join(dir, "example", "SKILL.md") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(-f example) stdout=%q; want %q (absolute SKILL.md path)", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("run(-f example) stderr=%q; want empty", errOut.String())
	}
}

// --relative prints the dir path RELATIVE to the skills dir (PRD §6.2). The output
// uses the OS path separator (filepath.Rel), so compare via FromSlash.
func TestRunTagRelativePrintsRelativeDir(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--relative", "writing/reddit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--relative writing/reddit): code=%d; want 0", code)
	}
	want := filepath.FromSlash("writing/reddit") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(--relative writing/reddit) stdout=%q; want %q (relative dir)", got, want)
	}
}

// --file --relative COMBINE: a SKILL.md path RELATIVE to the skills dir (PRD §6.2).
func TestRunTagFileRelativeCombine(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-f", "--relative", "writing/reddit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-f --relative writing/reddit): code=%d; want 0", code)
	}
	want := filepath.FromSlash("writing/reddit/SKILL.md") + "\n"
	if got := out.String(); got != want {
		t.Errorf("run(-f --relative writing/reddit) stdout=%q; want %q (relative SKILL.md)", got, want)
	}
}

// Modifiers must NOT break §6.4 atomicity: one bad tag -> NOTHING on stdout, exit 1.
func TestRunTagFileAtomicity(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-f", "example", "nope"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(-f example nope): code=%d; want 1 (atomic failure)", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want EMPTY (modifiers must not break §6.4)", out.String())
	}
}

// --- run: --all/-a (P1.M3.T8.S2) ---

// --all prints every skill's absolute DIRECTORY path, one per line, SORTED by
// canonical tag (discover.Index already sorts []Skill by RelTag). exit 0.
func TestRunAllPrintsAllDirsSorted(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	// Sorted by RelTag: "example" < "writing/reddit".
	if lines[0] != filepath.Join(dir, "example") {
		t.Errorf("lines[0]=%q; want example dir (sorted)", lines[0])
	}
	if lines[1] != filepath.Join(dir, "writing", "reddit") {
		t.Errorf("lines[1]=%q; want writing/reddit dir (sorted)", lines[1])
	}
	if errOut.Len() != 0 {
		t.Errorf("run(--all) stderr=%q; want empty", errOut.String())
	}
}

// -a short form behaves identically to --all.
func TestRunAllShortFlag(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"-a"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(-a): code=%d; want 0", code)
	}
	if !strings.Contains(out.String(), filepath.Join(dir, "example")) {
		t.Errorf("run(-a) stdout missing example dir:\n%s", out.String())
	}
}

// --all --file: every skill's ABSOLUTE SKILL.md path, sorted by tag.
func TestRunAllFilePrintsAllSourceFiles(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--file"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all --file): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	if lines[0] != filepath.Join(dir, "example", "SKILL.md") {
		t.Errorf("lines[0]=%q; want example SKILL.md (sorted)", lines[0])
	}
	if lines[1] != filepath.Join(dir, "writing", "reddit", "SKILL.md") {
		t.Errorf("lines[1]=%q; want writing/reddit SKILL.md (sorted)", lines[1])
	}
}

// --all --relative: every skill's directory path RELATIVE to the skills dir, sorted.
func TestRunAllRelativePrintsAllRelative(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--relative"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all --relative): code=%d; want 0", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 paths; got %d: %q", len(lines), out.String())
	}
	if lines[0] != "example" {
		t.Errorf("lines[0]=%q; want 'example' (relative)", lines[0])
	}
	if lines[1] != filepath.FromSlash("writing/reddit") {
		t.Errorf("lines[1]=%q; want 'writing/reddit' (relative, OS-sep)", lines[1])
	}
}

// --all with an EMPTY store -> prints nothing, exit 0 (PRD §6.1: --all is always
// exit 0, UNLIKE --list which exits 1 "if no skills found" — --all is a scripting
// command where empty output + exit 0 is the useful shape).
func TestRunAllEmptyStoreExit0(t *testing.T) {
	t.Setenv("SKPP_SKILLS_DIR", t.TempDir()) // exists, no SKILL.md
	var out, errOut bytes.Buffer
	code := run([]string{"--all"}, &out, &errOut)
	if code != 0 {
		t.Errorf("run(--all) empty: code=%d; want 0 (PRD §6.1 --all is always 0)", code)
	}
	if out.Len() != 0 {
		t.Errorf("run(--all) empty stdout=%q; want empty", out.String())
	}
}

// --all when skills dir is unresolvable -> exit 1, empty stdout, the one-line fix.
func TestRunAllSkillsDirUnresolvable(t *testing.T) {
	unsetSkillsEnv(t)
	t.Chdir(t.TempDir()) // all three §8 rules miss
	var out, errOut bytes.Buffer
	code := run([]string{"--all"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("run(--all) unresolvable: code=%d; want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout=%q; want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "SKPP_SKILLS_DIR") {
		t.Errorf("stderr=%q; want the one-line fix", errOut.String())
	}
}

// --version precedes --all even when both are given (PRD §6.3).
func TestRunVersionPrecedenceOverAll(t *testing.T) {
	dir := sampleStore(t)
	t.Setenv("SKPP_SKILLS_DIR", dir)
	var out, errOut bytes.Buffer
	code := run([]string{"--all", "--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--all --version): code=%d; want 0 (version precedence)", code)
	}
	if got := out.String(); got != "skpp "+version+"\n" {
		t.Errorf("stdout=%q; want the version line (precedence over --all)", got)
	}
}
````

> **Copy-paste correctness:** both edits are gofmt-clean and were compiled + tested
> verbatim against the real module in `/tmp/skpp_s2_verify` (go1.25). main.go imports
> exactly fmt/io/os/path/filepath/strings + internal/{discover,resolve,skillsdir,ui};
> main_test.go imports are unchanged. Apply File 1 (write the complete main.go) and
> File 2 (append the block) and run the gates. The `skillPath` abstraction, the
> `--all` empty-store exit 0, and the modifier-preserved atomicity trace directly to
> PRD §6.1/§6.2/§6.4 + go_architecture.md "Output discipline"; every assertion maps
> to a verified_facts entry.

### Implementation Tasks (ordered by dependencies)

```yaml
Task 0 (GATE): CONFIRM S1 is LANDED before starting
  - COMMAND: grep -q 'func run\|tags \[\]string' main.go && grep -q 'res.Skill.Dir' main.go
  - EXPECT: matches (P1.M3.T8.S1 LANDED, commit 814bab4). The <tag> branch must exist
            with `paths = append(paths, res.Skill.Dir)`. If S1 is NOT in, STOP: this
            subtask's one-line change has nothing to change. (S1 is COMPLETE.)

Task 1: MODIFY main.go  (write the complete File 1 over the existing main.go)
  - WRITE: the exact content from the Blueprint (File 1).
  - CHECK the 6 additions are present:
      (a) imports include "path/filepath";
      (b) config struct has all/file/relative bool fields;
      (c) parseArgs has cases for "--all"/"-a", "--file"/"-f", "--relative";
      (d) run has `if c.all { ... }` AFTER --list and BEFORE the <tag> branch;
      (e) the <tag> loop appends skillPath(res.Skill, dir, c) (not res.Skill.Dir);
      (f) the skillPath helper function is present.
  - CHECK the version/path/list/<tag>/default branches are PRESERVED verbatim.
  - GOTCHA: --file applies before --relative in skillPath; --all exit 0 on empty;
            --all needs no buffering; --all is sorted (free via Index); --list
            untouched; run `gofmt -w` (struct-field alignment).

Task 2: MODIFY main_test.go  (APPEND the File 2 block — 17 new tests)
  - APPEND: the File 2 block after the last existing test (TestRunVersionPrecedenceOverTag).
  - CHECK: main_test.go imports UNCHANGED; the 17 new tests are present; all 36 prior
           tests preserved verbatim.
  - GOTCHA: NO new imports; NO testify; NO t.Parallel(); reuse writeSkillTree/
            unsetSkillsEnv/version/sampleStore; SKPP_SKILLS_DIR (rule 1) + t.Chdir(
            TempDir) to force misses; relative-path assertions use filepath.FromSlash.

Task 3: FORMAT + VET + BUILD + TEST (validation gates — run in order)
  - COMMAND: gofmt -w main.go main_test.go
  - COMMAND: gofmt -l main.go main_test.go   # MUST print nothing
  - COMMAND: go vet ./...                     # MUST be clean
  - COMMAND: go build ./...                   # exit 0 (whole module compiles)
  - COMMAND: go test . -v                     # ALL 53 main tests PASS (was 36)
  - COMMAND: go test ./...                    # whole module green
  - COMMAND: go mod tidy && git diff --quiet go.mod go.sum   # go.mod/go.sum UNCHANGED
  - EXPECT: zero errors, zero vet findings, gofmt silent, 53 main tests pass, no go.mod change.

Task 4: ACCEPTANCE SMOKE TEST (Level 3 in Validation Loop)
  - COMMAND: the Level 3 block below (build a real binary, point SKPP_SKILLS_DIR at a
             2-skill store, run the 10 modifier/--all scenarios + assert stdout/stderr/rc).
  - EXPECT: -f -> absolute SKILL.md (§13 `test -f` PASS); --relative -> "writing/reddit";
            -f --relative -> "writing/reddit/SKILL.md"; --all -> sorted absolute dirs;
            --all --file/--relative -> sorted SKILL.md/relative; --all empty -> exit 0;
            -f example nope -> stdout EMPTY rc=1 (atomicity preserved).

Task 5: SCOPE BOUNDARY CHECK — Level 4 in Validation Loop
  - COMMAND: the Level 4 block below.
  - EXPECT: main.go has exactly the 6 additions (no --search/check/--help); imports
            correct; go.mod/go.sum unchanged; nothing under internal/ touched; no new files.
```

### Implementation Patterns & Key Details

```go
// PATTERN: the shared skillPath formatter (the §6.2 "combine" contract by construction).
//   func skillPath(s discover.Skill, skillsDir string, c config) string {
//       p := s.Dir
//       if c.file { p = s.SourceFile }
//       if c.relative { if rel, err := filepath.Rel(skillsDir, p); err == nil { p = rel } }
//       return p
//   }
// WHY: §6.2 says --file/--relative apply to BOTH <tag> resolution and --all. Routing
//      both loops through ONE formatter means the two modes cannot drift, and the S1
//      buffered-atomicity loop is untouched except the append target. --file first,
//      then --relative, so they combine correctly (a relative SKILL.md path).

// PATTERN: --all is a plain iterate (no buffering, no per-skill failure mode).
//   for _, s := range skills { fmt.Fprintln(stdout, skillPath(s, dir, c)) }
//   return 0
// WHY: discover.Index sorts by RelTag (§6.1 "sorted by tag" is free) and never errors
//      per-skill (it builds HasFM=false for malformed SKILL.md). So --all just walks
//      the index. Empty store -> empty loop -> exit 0 (NOT the --list exit-1 shape).

// PATTERN: modifier-preserved atomicity (the one-line change to the <tag> loop).
//   paths = append(paths, skillPath(res.Skill, dir, c))   // was: res.Skill.Dir
// WHY: skillPath runs only AFTER Resolve succeeds, so it only reformats a good path.
//      The buffer/continue/flush-only-on-success loop is byte-for-byte S1's, so §6.4
//      holds: `skpp -f example nope` prints NOTHING to stdout, exit 1.
```

### Integration Points

```yaml
PACKAGE BOUNDARIES (after this subtask):
  - main.go imports fmt, io, os, path/filepath, strings, internal/{discover,resolve,
    skillsdir,ui}. (path/filepath is NEW in main's import set vs S1.)
  - main consumes: skillsdir.Find(), discover.Index(), discover.Skill{Dir,SourceFile},
    resolve.Resolve()/Result, ui.PrintList(), and the new skillPath() it owns.

go.mod / go.sum (NO change — verified_facts §2):
  - main gains one STDLIB import (path/filepath). No new third-party dependency.
    `go mod tidy` is a no-op. VERIFY: `go mod tidy && git diff --quiet go.mod go.sum`
    exits 0.

DOWNSTREAM EXTENSION POINTS (what later subtasks plug into — designed for reuse):

  - P1.M4.T9 (--search) and P1.M4.T10 (check): do NOT touch the <tag> or --all
    branches. --search reuses ui.PrintList (already wired) + the new `search` config
    field, as a sibling mode (place its branch wherever fits; it is mutually exclusive
    with the others per §6.3, enforced in M5). `check` adds subcommand dispatch BEFORE
    the tag branch in run (so `skpp check` stops resolving "check" as a tag). Neither
    needs skillPath.

  - P1.M5.T11 (full CLI surface): adds --help/-h to the precedence tier (before
    --path); turns unknown dashed flags into exit 2; enforces §6.3 mutual-exclusivity
    (tag + --list/--search/--all -> exit 2). The <tag> and --all branches themselves
    are unchanged — M5 only adds a guard that rejects `c.tags` mixed with list/search/all.
    skillPath is unaffected.

  - P1.M6.T15 (shell completions): invokes `skpp --all` (now shipped here) for
    positional tag completion (PRD §14). This subtask UNBLOCKS that deliverable.

NO CHANGES TO:
  - internal/discover/*, internal/skillsdir/*, internal/ui/*, internal/resolve/*
    (all consumed READ-ONLY).
  - go.mod, go.sum, .gitignore, LICENSE, PRD.md.
  - No new files; no skills/ dir (P1.M6.T12); no install.sh/README/completions.
```

---

## Validation Loop

### Level 1: Syntax & Style (Immediate Feedback)

```bash
cd /home/dustin/projects/skpp
gofmt -w main.go main_test.go        # apply formatting (struct-field alignment is the only change)
gofmt -l main.go main_test.go        # MUST print nothing
go vet ./...                         # MUST be clean
# Expected: zero output from gofmt -l; zero vet findings.
```

### Level 2: Unit Tests (Component Validation)

```bash
# The new modifier/--all tests in isolation
go test . -run 'TestParseArgs(File|Relative|All|Modifiers)|TestRunTag(File|Relative)|TestRunAll|TestRunVersionPrecedenceOverAll' -v

# Whole main package (expect 53 PASS: prior 36 + 17 new)
go test . -v

# Whole module
go test ./...
# Expected: all pass. main package = 53 tests.
```

### Level 3: Integration Testing (System Validation — real binary, §13-style)

```bash
cd /home/dustin/projects/skpp
go build -o /tmp/skpp_s2_bin .
S=$(mktemp -d)/skills
mkdir -p "$S/example" "$S/writing/reddit"
printf -- '---\nname: example\ndescription: A demo skill.\n---\n# body\n' > "$S/example/SKILL.md"
printf -- '---\nname: reddit-poster\ndescription: Posts to reddit.\n---\n# body\n' > "$S/writing/reddit/SKILL.md"
export SKPP_SKILLS_DIR="$S"

# 1) default: absolute dir (unchanged from S1)
/tmp/skpp_s2_bin example

# 2) -f: absolute SKILL.md path  — §13 gate
/tmp/skpp_s2_bin -f example
test -f "$(/tmp/skpp_s2_bin -f example)" && echo "§13 -f gate: PASS"

# 3) --relative: dir relative to the skills dir
/tmp/skpp_s2_bin --relative example                 # -> example

# 4) --file --relative COMBINE: relative SKILL.md path
/tmp/skpp_s2_bin -f --relative writing/reddit       # -> writing/reddit/SKILL.md

# 5) --all: every dir, absolute, SORTED by tag
/tmp/skpp_s2_bin --all

# 6) --all --file / --all --relative
/tmp/skpp_s2_bin --all --file
/tmp/skpp_s2_bin --all --relative

# 7) --all on an EMPTY store -> exit 0, nothing printed
E=$(mktemp -d); SKPP_SKILLS_DIR="$E" /tmp/skpp_s2_bin --all; echo "exit=$? (want 0)"

# 8) atomicity preserved with modifiers: one bad tag -> stdout EMPTY, exit 1
out=$(/tmp/skpp_s2_bin -f example nope 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = "1" ] && echo "modifier atomicity OK" || echo "FAIL: out=[$out] rc=$rc"

# 9) end-to-end with pi (skills load ONLY via --skill, never auto-discovered) — P1.M6 ships the example skill
# pi --no-skills --skill "$(/tmp/skpp_s2_bin -f example)" -p "briefly confirm the example skill is loaded" 2>&1 | head
# (Defer the pi line until skills/example exists (P1.M6.T12); the binary contract above is this subtask's gate.)

rm -f /tmp/skpp_s2_bin; rm -rf "${S%/*}" "$E"
# Expected: 1) absolute dir; 2) absolute SKILL.md + "PASS"; 3) "example"; 4) "writing/reddit/SKILL.md";
#           5) two sorted absolute dirs; 6) SKILL.md / relative variants; 7) exit=0; 8) "modifier atomicity OK".
```

### Level 4: Creative & Domain-Specific Validation (scope boundary)

```bash
cd /home/dustin/projects/skpp
# main.go imports are EXACTLY the expected set (path/filepath is the only add vs S1):
go list -f '{{join .Imports " "}}' .
# Expected to include: fmt io os path/filepath strings
#                      github.com/dabstractor/skpp/internal/discover
#                      github.com/dabstractor/skpp/internal/resolve
#                      github.com/dabstractor/skpp/internal/skillsdir
#                      github.com/dabstractor/skpp/internal/ui

# No out-of-scope flags leaked into main.go:
! grep -nE '\-\-(search|help)\b|c\.search|c\.check|c\.help' main.go \
  && echo "scope OK (no M4/M5 flags in main.go)"

# go.mod/go.sum UNCHANGED:
go mod tidy && git diff --quiet go.mod go.sum && echo "go.mod/go.sum unchanged"

# Nothing under internal/ was touched, and no new files were created:
git status --porcelain internal/   # Expected: empty
git status --porcelain | grep -E '\?\?' | grep -vE '\.pi-subagents|plan/' || echo "no stray new files"
# Expected: "scope OK", "go.mod/go.sum unchanged", empty internal/ status, no stray files.
```

---

## Final Validation Checklist

### Technical Validation

- [ ] All 4 validation levels completed successfully.
- [ ] `gofmt -l main.go main_test.go` is silent (run `gofmt -w` first).
- [ ] `go vet ./...` is clean.
- [ ] `go build ./...` succeeds.
- [ ] `go test ./...` passes; main package = **53 tests** (prior 36 + 17 new).
- [ ] `go mod tidy` is a no-op; `git diff --quiet go.mod go.sum` exits 0.

### Feature Validation (PRD §6.1 / §6.2 / §6.4)

- [ ] `skpp -f example` prints the **absolute SKILL.md** path; `test -f "$(...)"` passes (§13).
- [ ] `skpp --relative example` prints the dir path **relative to the skills dir**.
- [ ] `skpp -f --relative <tag>` prints a **relative SKILL.md** path (the combine).
- [ ] `skpp --all` prints every skill's absolute **directory** path, one per line,
      **sorted by tag**, exit 0.
- [ ] `skpp -a` behaves identically to `--all`.
- [ ] `skpp --all --file` / `skpp --all --relative` apply the modifiers to every row.
- [ ] `skpp --all` on an empty store prints nothing and exits **0** (not 1).
- [ ] `skpp --all` on an unresolvable dir exits 1, empty stdout, the one-line fix.
- [ ] Modifiers do **not** break §6.4 atomicity: `skpp -f <good> <bad>` ⇒ nothing on
      stdout, exit 1 (Level 3 step 8).
- [ ] `--list` is unaffected by `--file`/`--relative` (all 6 existing --list tests pass).
- [ ] `--version` precedes `--all` (PRD §6.3).

### Code Quality & Scope Validation

- [ ] `main.go` has exactly the 6 additions (path/filepath import, all/file/relative
      fields, 3 parseArgs cases, `--all` branch, one-line `<tag>`-loop change,
      `skillPath` helper); version/path/list/`<tag>`/default logic preserved verbatim.
- [ ] `main.go` imports == fmt, io, os, path/filepath, strings,
      internal/{discover,resolve,skillsdir,ui}.
- [ ] No `--search`/`check`/`--help` added (M4/M5).
- [ ] `main_test.go` imports unchanged; reuses helpers; no testify, no `t.Parallel()`.
- [ ] Nothing under `internal/` touched; no new files created; `go.mod`/`go.sum`/
      `PRD.md` untouched.

---

## Anti-Patterns to Avoid

- ❌ Don't fork the modifier logic into two places — route both the `<tag>` loop and
  the `--all` loop through the shared `skillPath()` helper (§6.2 "combine with tag
  resolution or --all").
- ❌ Don't apply `--relative` before choosing Dir/SourceFile, and don't only-ever
  relativize the dir — `--file --relative` must yield a relative SKILL.md path
  (`--file` first, then `--relative`).
- ❌ Don't drop the `filepath.Rel` error guard — keep the `if ...; err == nil` so a
  theoretical Rel failure falls back to the absolute path instead of using a "" or
  panicking.
- ❌ Don't copy `--list`'s empty-store exit-1 into `--all` — `--all` is exit 0 even
  empty (PRD §6.1); the empty store is a valid "enumerate" result.
- ❌ Don't wrap `--all` in the buffered-atomicity dance — nothing fails per-skill
  (Index builds HasFM=false Skills), so a plain iterate is correct and clearer.
- ❌ Don't re-sort `--all` output, or sort the `<tag>` output — `--all` is tag-sorted
  (free via Index); `<tag>` is input order (PRD §6.1). They are deliberately distinct.
- ❌ Don't thread `skillPath` into `--list`, or error on `--list`+`--file` now —
  `--list` prints a table; modifiers don't apply to it (§6.2 header); M5 handles
  exclusivity.
- ❌ Don't invent a `-r` short form for `--relative` — PRD §6.2 defines only the long form.
- ❌ Don't break §6.4 atomicity — `skillPath` runs after Resolve succeeds; keep the
  S1 buffer/continue/flush loop intact (only the append target changes).
- ❌ Don't add `--search`/`check`/`--help`, or touch `internal/*`, `go.mod`, `go.sum`,
  `PRD.md`.
- ❌ Don't add testify or `t.Parallel()` — repo convention is plain `t.Errorf`/`t.Fatalf`,
  no Parallel.
- ❌ Don't skip `gofmt -w` — adding the new bool fields triggers struct-field alignment.

---

**Confidence Score: 10/10** for one-pass implementation success. The exact `main.go`
(complete file) and the appended `main_test.go` block (17 new tests) are given
verbatim and have been **compiled, gofmt'd, vetted, unit-tested (main 36→53), and
smoke-tested against a real built binary** in `/tmp/skpp_s2_verify` (go1.25). All
ten end-to-end modifier/`--all` behaviors match PRD §6.1/§6.2/§6.4/§13, including the
§13 `test -f "$(./skpp -f example)"` gate, the `--file --relative` combine, `--all`'s
empty-store exit 0, and modifier-preserved atomicity. `go.mod`/`go.sum` are
byte-identical (only `path/filepath`, a stdlib import, is added). An implementer who
applies the two edits verbatim and runs the four validation levels will ship this in
one pass.
