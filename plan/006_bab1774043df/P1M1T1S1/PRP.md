# PRP — P1.M1.T1.S1: Convert `--link` to multi-target batch (struct, parser, exclusivity, runLink, tests)

> **Subtask:** P1.M1.T1.S1 — the whole Go implementation of PRD §8.4 batch linking in one subtask: config struct, parseArgs (both forms + the default handler), exclusivity (no change needed — see §3e), the missing-value guard, runLink (loop), usageText, and all test updates + new tests. Plus the README Mode-A doc ride-along.
> **Scope boundary:** Edits `main.go`, `main_test.go`, and `README.md` only. Does NOT touch the three completion scripts (multi-link *directory completion* is P1.M2.T1.S1, a separate subtask). No new deps.

---

## Goal

**Feature Goal**: Make `skilldozer --link <dir> [<dir>...]` link **one or more** external skill directories into the store in a single invocation (PRD §8.4): pass `--link` once, then every following positional is a directory to link — absolutized, validated, and symlinked independently, in input order, with partial success (valid links persist if others fail).

**Deliverable**: Edits to `main.go` (struct field `linkTarget string`→`linkTargets []string` + `collectingLink bool`; both parseArgs `--link` forms; both positional-append sites; the missing-value guard message; runLink rewritten as a loop; usageText 3 sites) + `main_test.go` (6 existing tests updated, 6 new tests added) + `README.md` (Mode A: §8.4 section + quick-start).

**Success Definition**: `go build ./...` + `go test ./...` pass; `--link a b c` links all three (3 paths on stdout in order, exit 0); `--link a invalid b` links a+b, fails invalid (2 paths stdout, 1 stderr, exit 1); `--link` alone → exit 2; `--link --check` → exit 2 (dashed-follower preserved); all existing link tests pass (adapted); `go.mod`/`go.sum` unchanged.

---

## User Persona (if applicable)

**Target User**: developers who keep skills in separate repos/locations and want them live-linked into the skilldozer store (the `npm link` / `pip install -e` idiom), especially in batch.

**Use Case**: `skilldozer --link ~/projects/agent-browser ~/projects/agent-builder ~/projects/mdsel` links all three in one command.

**Pain Points Addressed**: today `--link` takes a single dir, so linking N skills means N invocations; trailing positionals are rejected as tags (exit 2). Batch lets users link a whole working set at once, with partial success so one bad dir doesn't block the rest.

---

## Why

- **PRD §8.4** makes batching the headline `--link` behavior: "Pass `--link` once; every positional token after it is a directory to link." The live code is still single-target (`linkTarget string`), so this is the conversion.
- **PRD §6.1 / §6.4** define the batch contract: one absolute link path per success (input order) on stdout; exit 0 if all link, 1 if any fail (successful links remain), 2 if no dir follows `--link`; mixed stdout/stderr is allowed for partial batches.
- **Decision 21** (§19): `--link` is the sole mode that *collects* trailing positionals — once parsed, every following non-flag token is a link dir (never a tag), preserving the bare-positional-namespace-for-tags guarantee.

---

## What

`--link` becomes a batch collector:

- **Struct**: `linkTarget string` → `linkTargets []string`; add `collectingLink bool`.
- **parseArgs**: when `--link` consumes a value (next-token or `=`-form), set `c.link=true`, `c.collectingLink=true`, and append to `linkTargets`. Once `collectingLink` is set, every later positional (in BOTH the `default:` case and the `--` endOfOpts handler) routes to `linkTargets`, not `tags`. A dashed follower (`--link --check`) is NOT consumed → `linkMissingValue=true`.
- **exclusivityError**: NO code change — the collectingLink routing means post-link positionals never enter `c.tags`, so the existing `hasTags` check now only catches *pre*-link tags (the real conflict). The mode-flag check (`c.link && c.check || …`) is unchanged.
- **missing-value guard**: message → `skilldozer: --link requires at least one path to a skill directory`; broaden to `c.linkMissingValue || (c.link && len(c.linkTargets) == 0)`.
- **runLink**: resolve the store ONCE, then loop `c.linkTargets` in order (absolutize → validate → conflict → symlink → report); track `hadErr`; return 1 if any failed, else 0.
- **usageText**: `--link <dir> [<dir>...]` + a multi-link example.

### Success Criteria

- [ ] `config.linkTargets []string` + `config.collectingLink bool` replace `linkTarget string`
- [ ] both parseArgs `--link` forms append to `linkTargets` + set `collectingLink`; dashed-follower → `linkMissingValue` (not consumed)
- [ ] both positional-append sites (default :442 + endOfOpts :207) route to `linkTargets` when `collectingLink`
- [ ] missing-value guard uses the new message + the `len==0` defense; exit 2
- [ ] runLink loops over `linkTargets` (store resolved once); per-target stderr on failure, per-success stdout; exit 1 if any fail
- [ ] usageText shows `--link <dir> [<dir>...]` + batch example
- [ ] 6 existing tests updated; 6 new tests added; `go build` + `go test ./...` pass
- [ ] `--link --check` still exits 2 (dashed-follower preserved); `go.mod`/`go.sum` unchanged

---

## All Needed Context

### Context Completeness Check

**Pass.** Every edit site is pinned to a verified live line; the complete `linkTarget` reference set (8 main.go + 4 main_test.go sites) is enumerated; the two positional-append sites are flagged; the `c.link`-semantics design decision (with its test-preservation reasoning) is spelled out; exclusivityError's no-change rationale is given; the 6 test updates (incl. the 2 the table missed) and 6 new tests are listed; and runLink's exact current body (to wrap in a loop) is referenced. An implementer who has never seen this repo can do it in one pass.

### Documentation & References

```yaml
- file: main.go
  why: "THE edit site. config struct :155-185 (linkTarget :177, linkMissingValue :178). =-form case \"--link\": :266-274. next-token case \"--link\": :403-429. default handler :440-445 (c.tags append :442). endOfOpts handler :200-209 (c.tags append :207). missing-value guard :608-610. exclusivityError --link block :941-953 (NO change). runLink :1424-1494 (rewrite to loop). usageText :71-117 (--link at :82,:98,:111)."
  pattern: "Mirror --store's value-capture shape for the first target; the NEW behavior is that collectingLink then routes all later positionals to linkTargets. runLink's per-target steps (2-7) already exist verbatim — wrap them in a loop, resolve the store once before it."
  gotcha: "TWO c.tags append sites (:207 endOfOpts + :442 default) — guard BOTH with collectingLink. And c.link semantics: set true only when a value is consumed (next-token), to preserve 3 existing tests + the dashed-follower guard (see research §3)."

- file: main_test.go
  why: "19 existing link tests at :3491-3984 (full table in codebase_state.md). 6 need updates (research §8); 6 new tests to add (research §9). Harness: run([]string{...}, &out, &errOut)→int; fixture style = t.TempDir store via SKILLDOZER_SKILLS_DIR + a SKILL.md in each src dir."
  pattern: "The 2 table-misses to NOT overlook: TestRunLinkNoValueExits2 (:3549) asserts the EXACT old message (change to 'at least one path'); TestParseArgsLinkDashedFollowerNotConsumed (:3876) has a linkTarget assertion (rename to len(linkTargets)==0)."

- file: plan/006_bab1774043df/architecture/codebase_state.md
  why: "The full single-target map + the 19-test table (line numbers verified vs live). Use it as the site index; treat its 'Same/Changes' column as a hint, NOT gospel — research §8 corrects 2 entries it under-specified."
  section: "Current --link Architecture; Existing Link Tests"

- file: plan/006_bab1774043df/P1M1T1S1/research/verified_facts.md
  why: "Direct-from-source proof: the git-log/live-state discrepancy, the complete linkTarget ref set, the two-append-site gotcha, the c.link-semantics design decision (Option B), exclusivityError's no-change rationale, the 6 test updates (incl. the 2 the table missed), and runLink's loop structure."

- file: README.md
  why: "Mode A doc ride-along. §8.4 section (~:158 'Linking skills from elsewhere') + quick-start example (~:127). Document batch syntax, partial success, exit codes."
  pattern: "Mirror the PRD §8.4 examples (single + multi-link)."

- url: (PRD §8.4 + §6.1/§6.4 + decision 21 — in PRD.md, READ-ONLY)
  why: "§8.4: batch behavior, per-dir validation, conflict handling (refresh symlink / refuse non-symlink), partial success. §6.1/§6.4: stdout=one path/success in order; exit 0 all / 1 any / 2 no-dirs; mixed output allowed. Decision 21: --link collects trailing positionals. Do NOT edit PRD.md."
```

### Current Codebase tree (the relevant slice)

```bash
$ cd /home/dustin/projects/skilldozer && ls main.go main_test.go README.md go.mod
main.go   main_test.go   README.md   go.mod
# main.go: 1577 lines. config struct :155-185 (linkTarget :177). parseArgs :179-447.
#   =-form --link :266-274; next-token --link :403-429; default :440-445; endOfOpts :200-209.
#   missing-value guard :608-610; exclusivityError --link :941-953; runLink :1424-1494; usageText :71-117.
# main_test.go: link tests :3491-3984.
# go.mod: module github.com/dabstractor/skilldozer, go 1.25, require gopkg.in/yaml.v3 v3.0.1 (sole dep).
# No new files. Edits main.go + main_test.go + README.md.
```

### Desired Codebase tree with files to be changed

```bash
main.go         # MODIFY — struct (linkTargets+collectingLink), parseArgs (2 forms + 2 append sites), guard, runLink (loop), usageText
main_test.go    # MODIFY — 6 existing tests updated, 6 new tests added
README.md       # MODIFY — §8.4 batch docs + quick-start (Mode A)
# completions/{skilldozer.bash,_skilldozer,skilldozer.fish} — UNCHANGED here (multi-link dir completion = P1.M2.T1.S1)
# go.mod / go.sum — UNCHANGED (no new deps; expandHome/HasSkillMD/filepath/os already imported)
```

### Known Gotchas of our codebase & Library Quirks

```go
// GOTCHA #1 — TWO positional-append sites, not one. The default: case (main.go:442)
// AND the POSIX `--` endOfOpts handler (main.go:207) both do `c.tags = append(c.tags, a)`.
// Route BOTH through collectingLink, else `--link a -- b` puts `b` in tags (→ exclusivity
// exit 2, wrong). The contract names only the default case; the endOfOpts site is the trap.

// GOTCHA #2 — c.link SEMANTICS (deviates from contract (b) literal). The contract says
// "set c.link=true unconditionally"; the LIVE next-token form sets c.link=true ONLY when a
// value is consumed, and 3 tests assert c.link=false on no-value/dashed-follower paths.
// Setting c.link=true unconditionally would flip those 3 + contradict (d)'s "preserve the
// dashed-follower guard". CHOSEN: next-token sets c.link=true (+collectingLink) ONLY when a
// value is consumed; else linkMissingValue=true (c.link stays false). =-form KEEPS c.link=true
// unconditionally (TestParseArgsLinkEqualsEmpty expects true). This asymmetry already exists;
// preserving it = minimal churn. See research §3.

// GOTCHA #3 — exclusivityError needs NO change. The hasTags check (main.go:947-949) now only
// catches PRE-link tags (post-link positionals route to linkTargets via collectingLink, so
// c.tags stays empty). Contract (e) is satisfied by the parseArgs routing, not by editing
// exclusivityError. Do NOT remove the hasTags check — it still correctly rejects `sometag --link a`.

// GOTCHA #4 — The missing-value MESSAGE changes ("a path" → "at least one path"). TestRunLinkNoValueExits2
// (main_test.go:3549) asserts the EXACT old string → it MUST be updated (the codebase_state.md
// table marked it "Same" — a miss). Grep for the old message to catch any other exact-string assert.

// GOTCHA #5 — The dashed-follower guard is LOAD-BEARING and must survive. `--link --check`:
// next-token, args[i+1]="--check" starts with "-" → else branch → linkMissingValue=true (NOT
// consumed as a target). The guard (main.go:608) then exits 2 BEFORE exclusivity/dispatch.
// Do NOT "simplify" by consuming dashed followers — that would link a dir named "--check" or
// silently link the cwd. (commit ce01b55; preserved by Option B.)

// GOTCHA #6 — runLink is NOT atomic and MUST NOT early-return on a per-target failure. Each
// target is validated+linked independently; a failure prints one stderr line and the loop
// CONTINUES (partial success is the §8.4 contract). Track hadErr; return 1 after the loop if
// any failed. stdout gets one path per SUCCESS, in input order.

// GOTCHA #7 — Resolve the store ONCE (skillsdir.Find before the loop), not per-target. An
// unconfigured store exits 1 before any target is touched. The "(found via <src>)" stderr
// label uses the single resolved src for every success line.

// GOTCHA #8 — Per-target error messages should name the ORIGINAL token (pre-Abs), not the
// absolutized path — the user typed the original. Keep a local `orig := t` in the loop and
// use it in the "is not an existing directory" / "absolutize" messages (mirrors the current
// single-target code which uses c.linkTarget, the original token).

// GOTCHA #9 — The field rename is global. 8 main.go sites + 4 main_test.go sites reference
// linkTarget (research §1). `go build` will catch any missed main.go site (compile error);
// `go test` catches the test sites. Grep `linkTarget` after editing to confirm zero remain.

// GOTCHA #10 — Scope: this subtask does NOT touch the completion scripts. Multi-link
// DIRECTORY completion (offering dirs for every positional after --link) is P1.M2.T1.S1.
// Editing completions/* here over-reaches. (The bash file's `--link` case currently only
// completes when $prev is --link; fixing that is the sibling task.)

// GOTCHA #11 — No new deps. expandHome (main.go:1089), skillsdir.HasSkillMD (skillsdir.go:213),
// skillsdir.Find, filepath.*, os.* are all already imported. go.mod/go.sum byte-for-byte identical.
```

---

## Implementation Blueprint

### Data models and structure

```go
// config struct (main.go:176-178) — field rename + new collector flag:
link               bool     // `skilldozer --link <dir> [<dir>...]` flag (PRD §8.4): batch-link external skill dirs
linkTargets        []string // the dirs to link, in input order (next-token, =-form first value, + trailing positionals)
collectingLink     bool     // once --link has consumed a value, route later positionals to linkTargets (not tags)
linkMissingValue   bool     // --link / --link= with NO dir (or only dashed followers) → exit 2
```

No other structs change. `collectingLink` is parse-local state that lives on config (read by the default/endOfOpts handlers).

### Implementation Tasks (ordered by dependencies)

```yaml
Task 1: RENAME the config struct field + add collectingLink (main.go:176-178)
  - linkTarget string → linkTargets []string (update the doc comment to batch wording)
  - ADD collectingLink bool (doc: routes later positionals to linkTargets once --link consumed a value)
  - DEPENDENCY: all later tasks reference c.linkTargets / c.collectingLink

Task 2: parseArgs next-token form (main.go:403-429)
  - when --link seen: if i+1 < len(args) && !HasPrefix(args[i+1], "-"):
        c.link = true; c.collectingLink = true; c.linkTargets = append(c.linkTargets, args[i+1]); i++
    else: c.linkMissingValue = true   (c.link + collectingLink stay false — GOTCHA #2)
  - PRESERVE the dashed-follower guard (GOTCHA #5). Update the case's doc comment to batch wording.

Task 3: parseArgs =-form (main.go:266-274)
  - case "--link": c.link = true; c.collectingLink = true
    if val != "" { c.linkTargets = append(c.linkTargets, val) } else { c.linkMissingValue = true }

Task 4: route BOTH positional-append sites through collectingLink
  - main.go:442 (default case else-branch): if c.collectingLink { c.linkTargets = append(c.linkTargets, a) } else { c.tags = append(c.tags, a) }
  - main.go:207 (endOfOpts handler):       if c.collectingLink { c.linkTargets = append(c.linkTargets, a) } else { c.tags = append(c.tags, a) }
  - GOTCHA #1 — both sites, or `--link a -- b` misroutes b to tags.

Task 5: missing-value guard (main.go:608-610)
  - broaden + new message:
      if c.linkMissingValue || (c.link && len(c.linkTargets) == 0) {
          fmt.Fprintln(stderr, "skilldozer: --link requires at least one path to a skill directory")
          return 2
      }

Task 6: runLink — rewrite as a loop (main.go:1424-1494)
  - keep (1) skillsdir.Find ONCE before the loop (unconfigured → stderr err, return 1)
  - declare hadErr := false
  - for _, t := range c.linkTargets { orig := t
      (2) absTarget := expandHome+filepath.Abs(orig) [on err: stderr naming orig, hadErr=true, continue]
      (3a) os.Stat isDir [else stderr naming orig, hadErr=true, continue]
      (3b) not store-inside (Clean+HasPrefix) [else stderr naming absTarget, hadErr=true, continue]
      (3c) HasSkillMD [else stderr naming absTarget, hadErr=true, continue]
      (4) name=Base(absTarget); linkPath=Join(store,name)
      (5) conflict: Lstat → non-symlink refuse [stderr, hadErr, continue] / symlink remove [on err stderr, hadErr, continue]
      (6) os.Symlink [on err stderr, hadErr, continue]
      (7) fmt.Fprintln(stdout, linkPath); fmt.Fprintf(stderr, "Linked %s -> %s (found via %s)\n", linkPath, absTarget, src)
    }
  - if hadErr { return 1 }; return 0
  - GOTCHA #6 (no early return) + #7 (store once) + #8 (orig in messages)

Task 7: usageText (main.go:82, 98, 111)
  - :82 `skilldozer --link <dir>` → `skilldozer --link <dir> [<dir>...]`
  - :98 add a multi-link example line
  - :111 `--link <dir>` → `--link <dir> [<dir>...]` + batch description

Task 8: UPDATE the 6 existing tests (main_test.go) — research §8 table
  - TestParseArgsLinkNextToken (:3496): c.linkTargets[0] == "/path/to/skill" (+ len==1)
  - TestParseArgsLinkEquals (:3513): c.linkTargets[0] == "/path/to/skill"
  - TestParseArgsLinkEqualsEmpty (:3540): len(c.linkTargets)==0 (c.link stays true)
  - TestParseArgsLinkDashedFollowerNotConsumed (:3876): len(c.linkTargets)==0 (c.link stays false)
  - TestRunLinkNoValueExits2 (:3549): stderr → "…requires at least one path to a skill directory\n" (GOTCHA #4)
  - TestRunLinkWithTagsExits2 (:3576): exit 2→1; rename → TestRunLinkTrailingPositionalBecomesTargetExits1;
    /tmp/foo + sometag both fail validation → exit 1, empty stdout (use temp dirs for realism if preferred)

Task 9: ADD 6 new tests (main_test.go) — research §9
  - TestRunLinkMultiSuccess, TestRunLinkMixedBatch, TestRunLinkSingleBadDir,
    TestParseArgsLinkCollectsMultiple (--link a b c → [a,b,c], tags empty),
    TestParseArgsLinkEqualsPlusPositionals (--link=a b → [a,b]),
    TestRunLinkOrderPreservation (3 dirs → stdout in input order)
  - FOLLOW the existing link-test fixture style (t.TempDir store via SKILLDOZER_SKILLS_DIR, SKILL.md in each src dir)

Task 10: README.md Mode A (§8.4 section ~:158 + quick-start ~:127)
  - document batch syntax, partial success (valid links persist), exit codes (0 all / 1 any / 2 no dirs)

Task 11: VERIFY
  - go build ./... ; go vet ./... ; go test ./... -v
  - grep -n linkTarget main.go main_test.go   (MUST be empty — GOTCHA #9)
  - git diff --quiet go.mod go.sum && echo "deps unchanged"
  - manual: --link a b c (3 paths), --link a invalid b (2 paths + stderr, exit 1), --link (exit 2), --link --check (exit 2)
```

### Implementation Patterns & Key Details

```go
// Task 4 — the collectingLink routing at BOTH append sites (the load-bearing parser change):
// main.go:207 (endOfOpts) and main.go:442 (default else-branch) — identical guard:
if c.collectingLink {
	c.linkTargets = append(c.linkTargets, a)
} else {
	c.tags = append(c.tags, a)
}

// Task 6 — runLink loop skeleton (store once; per-target validate+link+report; no early return):
func runLink(c config, stdout, stderr io.Writer) int {
	store, src, err := skillsdir.Find()
	if err != nil { fmt.Fprintln(stderr, err); return 1 }
	hadErr := false
	for _, t := range c.linkTargets {
		orig := t
		absTarget, err := filepath.Abs(expandHome(orig))
		if err != nil { fmt.Fprintf(stderr, "skilldozer --link: absolutize target %q: %v\n", orig, err); hadErr = true; continue }
		if info, e := os.Stat(absTarget); e != nil || !info.IsDir() { fmt.Fprintf(stderr, "skilldozer --link: %q is not an existing directory\n", orig); hadErr = true; continue }
		storeAbs := filepath.Clean(store)
		if absTarget == storeAbs || strings.HasPrefix(absTarget, storeAbs+string(filepath.Separator)) { fmt.Fprintf(stderr, "skilldozer --link: %q is already inside the store %q; nothing to link\n", absTarget, store); hadErr = true; continue }
		if !skillsdir.HasSkillMD(absTarget) { fmt.Fprintf(stderr, "skilldozer --link: %q contains no SKILL.md (not a skill directory)\n", absTarget); hadErr = true; continue }
		linkPath := filepath.Join(store, filepath.Base(absTarget))
		if li, lerr := os.Lstat(linkPath); lerr == nil {
			if li.Mode()&os.ModeSymlink == 0 { fmt.Fprintf(stderr, "skilldozer --link: %q already exists and is not a symlink; refusing to overwrite (remove it first)\n", linkPath); hadErr = true; continue }
			if err := os.Remove(linkPath); err != nil { fmt.Fprintf(stderr, "skilldozer --link: remove existing symlink %q: %v\n", linkPath, err); hadErr = true; continue }
		}
		if err := os.Symlink(absTarget, linkPath); err != nil { fmt.Fprintf(stderr, "skilldozer --link: create symlink %q: %v\n", linkPath, err); hadErr = true; continue }
		fmt.Fprintln(stdout, linkPath)
		fmt.Fprintf(stderr, "Linked %s -> %s (found via %s)\n", linkPath, absTarget, src)
	}
	if hadErr { return 1 }
	return 0
}
// (Steps 2-7 are the EXISTING single-target body, verbatim, wrapped in the loop with continue-on-fail.)
```

### Integration Points

```yaml
DISPATCH (unchanged): run() → if c.link { return runLink(c, stdout, stderr) } (main.go:651). The guard at :608 fires first.
EXCLUSIVITY (unchanged): the --link hasTags check now catches only pre-link tags (collectingLink routing). No edit.
COMPLETIONS (NOT this task): bash/zsh/fish multi-link dir completion is P1.M2.T1.S1.
NO DATABASE / NO CONFIG SCHEMA / NO NEW DEPS.
```

---

## Validation Loop

### Level 1: Syntax + the rename invariant

```bash
cd /home/dustin/projects/skilldozer
gofmt -l main.go main_test.go          # empty
go vet ./...                            # clean
go build ./...                          # exit 0
grep -n "linkTarget" main.go main_test.go   # MUST be empty (GOTCHA #9 — rename is global)
# Expected: gofmt/vet/build clean; the grep prints nothing.
```

### Level 2: Unit + batch tests

```bash
cd /home/dustin/projects/skilldozer
go test ./... ; echo "test exit $?"     # Expected: 0
go test -run 'Link' -v ./...            # the 6 updated + 6 new link tests, plus the unchanged ones
# Expected: all PASS. Watch specifically: TestRunLinkNoValueExits2 (new message),
#   TestRunLinkTrailingPositionalBecomesTargetExits1 (was WithTagsExits2; exit 1),
#   TestRunLinkMultiSuccess/MixedBatch/SingleBadDir/OrderPreservation, TestParseArgsLinkCollectsMultiple/EqualsPlusPositionals.
```

### Level 3: End-to-end batch behavior (the §8.4 contract)

```bash
cd /home/dustin/projects/skilldozer
go build -o /tmp/sdz . || { echo "FAIL"; exit 1; }
TMP=$(mktemp -d); STORE="$TMP/store"; mkdir -p "$STORE" "$TMP/a" "$TMP/b" "$TMP/notaskill"
printf -- '---\nname: a\ndescription: x\n---\n# x\n' > "$TMP/a/SKILL.md"
printf -- '---\nname: b\ndescription: x\n---\n# x\n' > "$TMP/b/SKILL.md"
# multi-success: 2 dirs, both link, exit 0, paths in order
out=$(SKILLDOZER_SKILLS_DIR="$STORE" /tmp/sdz --link "$TMP/a" "$TMP/b" 2>/dev/null); rc=$?
[ "$rc" = 0 ] && printf '%s\n' "$out" | grep -qx "$STORE/a" && printf '%s\n' "$out" | grep -qx "$STORE/b" && echo "multi OK"
# mixed batch: 2 valid + 1 invalid → valid link, invalid on stderr, exit 1
out=$(SKILLDOZER_SKILLS_DIR="$STORE" /tmp/sdz --link "$TMP/a" "$TMP/b" "$TMP/notaskill" 2>/tmp/e); rc=$?
[ "$rc" = 1 ] && printf '%s\n' "$out" | grep -qx "$STORE/a" && grep -q notaskill /tmp/e && echo "mixed OK"
# single bad dir: nothing on stdout, exit 1
out=$(SKILLDOZER_SKILLS_DIR="$STORE" /tmp/sdz --link "$TMP/notaskill" 2>/dev/null); rc=$?
[ -z "$out" ] && [ "$rc" = 1 ] && echo "single-bad OK"
# no dir → exit 2 (new message)
SKILLDOZER_SKILLS_DIR="$STORE" /tmp/sdz --link >/tmp/o 2>/tmp/e; [ "$?" = 2 ] && grep -q 'at least one path' /tmp/e && echo "no-value OK"
# dashed follower → exit 2 (guard preserved)
SKILLDOZER_SKILLS_DIR="$STORE" /tmp/sdz --link --check >/dev/null 2>&1; [ "$?" = 2 ] && echo "dashed-follower OK"
rm -rf /tmp/sdz "$TMP" /tmp/o /tmp/e
# Expected: every line "...OK".
```

### Level 4: Scope discipline + deps invariant

```bash
cd /home/dustin/projects/skilldozer
git diff --name-only | grep -E 'completions/' && echo "FAIL: touched completions (P1.M2 scope)" || echo "completions untouched OK"
git diff --quiet go.mod go.sum && echo "deps unchanged" || echo "FAIL: deps changed"
# Expected: completions untouched; deps unchanged.
```

---

## Final Validation Checklist

### Technical Validation
- [ ] Level 1 PASS — gofmt/vet/build clean; zero `linkTarget` references remain
- [ ] Level 2 PASS — `go test ./...` exit 0; the 6 updated + 6 new link tests pass
- [ ] Level 3 PASS — multi/mixed/single-bad/no-value/dashed-follower all behave per §8.4
- [ ] Level 4 PASS — completions untouched; deps unchanged

### Feature Validation
- [ ] `config.linkTargets []string` + `collectingLink bool`; both parseArgs forms append + set collectingLink
- [ ] both positional-append sites (default + endOfOpts) route through collectingLink
- [ ] dashed-follower preserved (`--link --check` → exit 2); missing-value message = "at least one path"
- [ ] runLink loops (store once, no early return, hadErr→exit 1, stdout one path/success in order)
- [ ] usageText shows `--link <dir> [<dir>...]` + batch example; README §8.4 updated

### Code Quality / Convention Validation
- [ ] runLink's per-target steps are the existing single-target body, verbatim, looped (minimal new logic)
- [ ] error messages name the ORIGINAL token (orig), not the absolutized path
- [ ] no new deps; go.mod/go.sum byte-for-byte identical

### Scope Discipline
- [ ] Did NOT touch completions/{skilldozer.bash,_skilldozer,skilldozer.fish} (P1.M2.T1.S1)
- [ ] Did NOT modify PRD.md (read-only), tasks.json, prd_snapshot.md, or .gitignore

---

## Anti-Patterns to Avoid

- ❌ **Don't guard only the `default:` append site.** The `--` endOfOpts handler (main.go:207) also appends to tags; guard both with collectingLink (GOTCHA #1).
- ❌ **Don't set `c.link=true` unconditionally on the next-token no-value path.** That flips 3 existing tests' `c.link=false` assertions and contradicts the dashed-follower guarantee. Set c.link=true only when a value is consumed (GOTCHA #2).
- ❋ **Don't edit exclusivityError.** The collectingLink routing already makes post-link positionals bypass `c.tags`, so the hasTags check only catches real pre-link conflicts — contract (e) is satisfied without touching it (GOTCHA #3).
- ❌ **Don't early-return from runLink on a per-target failure.** Partial success is the §8.4 contract — `continue` on failure, track `hadErr`, return 1 after the loop (GOTCHA #6).
- ❌ **Don't resolve the store per-target.** `skillsdir.Find` once, before the loop (GOTCHA #7).
- ❌ **Don't drop the dashed-follower guard.** `--link --check` must exit 2 (linkMissingValue), never consume `--check` as a target (GOTCHA #5).
- ❌ **Don't forget the missing-value message change.** "a path" → "at least one path"; TestRunLinkNoValueExits2 asserts the exact string and must be updated (GOTCHA #4).
- ❋ **Don't trust the codebase_state.md "Same/Changes" column blindly.** It under-specified 2 tests (the message assert + the dashed-follower linkTarget rename); research §8 is authoritative.
- ❌ **Don't touch the completion scripts.** Multi-link directory completion is P1.M2.T1.S1 (GOTCHA #10).
- ❌ **Don't add deps.** All helpers are already imported (GOTCHA #11).

---

## Confidence Score

**9/10** — Every edit site is pinned to a verified live line; the complete `linkTarget` reference set is enumerated (so the rename is global and grep-verifiable); runLink's loop body is the existing single-target code verbatim (minimal new logic, low risk); the two subtle traps (the second append site, the `c.link`-semantics test churn) are explicitly resolved with reasoning; and the 6 test updates (incl. the 2 the architecture table missed) + 6 new tests are listed. The 1-point reservation is the `c.link` design deviation from contract (b)'s literal wording — the PRP chooses the minimal-churn, semantically-correct option and documents why, but an implementer following (b) literally would need to also flip 3 extra test assertions (the PRP steers them away from that).
