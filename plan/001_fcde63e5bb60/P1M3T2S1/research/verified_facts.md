# Verified Facts — P1.M3.T2.S1 (`skpp <tag> [...]` output + §6.4 error contract)

> Directory `P1M3T2S1` == plan item `P1.M3.T8.S1` ("skpp <tag> [...] output +
> §6.4 error contract (atomicity)"). The plan id and directory name differ (renumber
> during planning); use the path given. This is milestone **M3** (Tag resolution &
> path output). It is the subtask that finally makes `skpp <tag>` print paths.

This file locks every load-bearing decision for the PRP. Every entry was confirmed
against the live codebase, the prior PRPs (contracts), `go_architecture.md`, and the
PRD. An implementer who reads only the PRP need not re-derive any of this.

---

## §0 — What this subtask IS and is NOT

**IS:** wire the three already-built packages (`skillsdir`, `discover`, `resolve`)
into `main.go` so that `skpp <tag> [<tag>...]` resolves tags to skill directory
paths and prints them with PRD §6.4 atomicity. It MODIFIES `main.go` and
`main_test.go` only. It creates NO new files and NO new packages.

**IS NOT:** it does NOT implement `--file`/`-f`, `--relative`, `--all`/`-a` (those
are P1.M3.T2.S2 — the modifiers that layer onto the atomic-output pattern this task
establishes). It does NOT implement `--list`/`--search`/`check` (M2/M4). It does NOT
implement unknown-flag→exit 2, mode-exclusivity→exit 2, or `--help`/no-args usage
text (all P1.M5.T11). It does NOT define Skill/Index/Resolve (those packages own
them; this task CONSUMES them).

**OUTPUT (item CONTRACT):** "`skpp <tag> [<tag>...]` fully working per §6.1 row 1
and §6.4. Establishes the atomic-output pattern that modifiers (S2) layer onto."

---

## §1 — HARD DEPENDENCIES (all contracts; verified absent/present on disk)

This subtask CONSUMES four packages. Three are contracts from other subtasks; one
(skillsdir) is already landed.

| Symbol | Source subtask | Status at research time | Verified |
|---|---|---|---|
| `skillsdir.Find() (dir, src Source, err)` + `ErrNotFound` | P1.M1.T2.S3 | **LANDED** | `grep Find internal/skillsdir/skillsdir.go` ✓ |
| `discover.Skill` struct (Dir/RelTag/Name/Aliases/...) | P1.M2.T4.S2 | **NOT on disk** (contract) | `grep typeSkill` → none |
| `discover.Index(absSkillsDir) ([]Skill, error)` | P1.M2.T5.S1 | **NOT on disk** (contract) | `grep func Index` → none |
| `resolve.Resolve(tag, []discover.Skill) (Result, error)` + `*UnknownError`/`*AmbiguousError` | P1.M3.T7.S1 (dir `P1M3T1S1`) | **Being implemented in parallel** | PRP read; treat as contract |

**GATE (Task 0):** Do NOT implement until `discover.Index` AND `discover.Skill`
exist on disk. M3 is scheduled after M2, so they will exist at implementation time.
Until then `go build`/`go test` fail with "undefined: discover.Index/Skill". The
resolve PRP (P1M3T1S1) has the SAME gate on `discover.Skill`.

---

## §2 — The consumed contracts (EXACT shapes — locked)

### skillsdir.Find (LANDED — read the source)
```go
func Find() (dir string, src Source, err error)   // §8 priority: env → sibling → walk-up
var ErrNotFound = errors.New("could not locate the skills directory: set $SKPP_SKILLS_DIR, cd into the skpp repo, or reinstall skpp")
```
- Returns ABSOLUTE dir + which rule won. On all-miss: `("", 0, ErrNotFound)`.
- `ErrNotFound.Error()` is the user-facing one-line fix. **Print verbatim to stderr**
  (the existing `--path` branch already does `fmt.Fprintln(stderr, err)` — mirror it).
- In tests, `t.Setenv("SKPP_SKILLS_DIR", tmpDir)` makes rule 1 win deterministically.

### discover.Skill + Index (CONTRACT — from go_architecture.md "Core types")
```go
type Skill struct {
    Dir         string   // ABSOLUTE path of the skill directory  ← what we print
    RelTag      string   // dir path relative to skills dir, separators normalized to '/'
    Name        string   // frontmatter name ("" if missing)
    Description string
    Keywords    []string
    Category    string
    Aliases     []string // metadata.aliases
    HasFM       bool
    SourceFile  string   // absolute path to SKILL.md (Dir + "/SKILL.md")
}
func Index(absSkillsDir string) ([]Skill, error)  // sorted by RelTag; lenient
```
- **`Dir` is ABSOLUTE** (Find returns absolute; Index joins it). We print `r.Skill.Dir`
  DIRECTLY — do NOT reconstruct, do NOT `filepath.Abs` it (that would mask an Index
  bug; the §13 `case ... in /*)` gate depends on Index producing absolute Dir).
- Index is LENIENT: a dir with no SKILL.md is skipped; missing frontmatter is not an
  error (HasFM=false). An EMPTY skills dir returns `([], nil)` (NOT an error). Only
  a walk failure or broken-YAML-between-fences errors. The Index-error path in main
  is therefore rare and defensive.

### resolve.Resolve + typed errors (CONTRACT — from P1M3T1S1 PRP, read in full)
```go
type Result struct { Skill discover.Skill; Match MatchKind }
func Resolve(tag string, skills []discover.Skill) (Result, error)

type UnknownError   struct{ Tag string }                 // pointer receiver Error()
type AmbiguousError struct{ Tag string; Candidates []string }  // pointer receiver Error()
```
- **Pointer receivers** → `*UnknownError` and `*AmbiguousError` satisfy `error`;
  the VALUE types do NOT. Resolve returns them as `&UnknownError{...}`.
- Extract via `errors.As(err, &ae)` / `errors.As(err, &ue)` — REQUIRED (do NOT type-
  assert; the resolve PRP's `TestResolveErrorsAs` is the contract proof).
- `Candidates` is ALREADY sorted by resolve (`sortedRelTags`) and is the list of
  matching skills' **RelTags** (full canonical tags for disambiguation).
- resolve's OWN `.Error()` text is DIFFERENT from §6.4 (see §5). Do NOT use `.Error()`
  for the stderr wording — extract `.Tag`/`.Candidates` and format per §5.

---

## §3 — The §6.4 atomicity contract (THE critical thing — item says "MOST CRITICAL")

PRD §6.4 + go_architecture.md "Output discipline (§6.4) — critical":
> Resolve ALL tags first, collect errors. If ANY tag fails → print one error line
> per problem to STDERR, print NOTHING to stdout, exit 1. Never partially print.
> Buffer stdout writes; only flush when the whole invocation is known-good.

**Why it matters:** `pi --skill "$(skpp badtag)"` captures stdout. If skpp printed a
PARTIAL result (the good tags) before hitting a bad one, `$(...)` would get a
garbage/incomplete path list and pi would load the wrong skills silently. The
contract guarantees: any failure ⇒ empty stdout + exit 1 ⇒ `$(...)` is empty ⇒ pi
errors loudly. (PRD §2.3 / §17.)

**Implementation (locked):** two-pass over the resolved slice.
1. Pass 1: `resolve.Resolve` EVERY tag into `[]resolved{tag, dir, err}` (input order).
2. Pass 2a (any err): print one stderr line per PROBLEM tag (skip the good ones),
   print NOTHING to stdout, return 1.
3. Pass 2b (all good): buffer all `dir` lines into a `strings.Builder`, then ONE
   `fmt.Fprint(stdout, buf)` so a mid-loop write failure can never leave partial
   stdout. Return 0.

The buffer is the architecture doc's literal instruction ("Buffer stdout writes;
only flush when the whole invocation is known-good"). Even though checking-all-errors-
   before-printing already makes it atomic, the buffer is the defensive guarantee.

---

## §4 — Output format (§6.1 row 1)

- Default output unit = **DIRECTORY** (`r.Skill.Dir`), NOT the SKILL.md file. (PRD §3
  decision; `--file`/`-f` in S2 swaps to SourceFile.)
- Output is **ABSOLUTE** by default. (PRD §6.1/§13: `case "$(./skpp example)" in /*)`.
  `--relative` in S2 swaps to relative.)
- **One path per line, in INPUT order** (the order the user typed the tags), NOT
  sorted by tag. discover.Index sorts by RelTag internally, but we iterate the TAGS
  (input order), so output follows tag order. `skpp b a` prints b's dir then a's dir.
- Each line is `dir + "\n"` (Fprintln). Trailing newline on the last line is fine
  (matches `--path`'s `fmt.Fprintln(stdout, dir)` and the §13 `test -d "$(./skpp x)"`).

---

## §5 — Error message format (item CONTRACT — authoritative, overrides resolve's .Error())

The item description specifies the EXACT stderr wording. This is DIFFERENT from
resolve's `.Error()` (which uses double quotes, "skill", and comma-joined matches).
main OWNS the §6.4 wording; extract fields and format here:

| Case | Format (item CONTRACT) | resolve's own .Error() (DO NOT USE for stderr) |
|---|---|---|
| Unknown | `skpp: unknown tag '<tag>'` | `unknown skill tag "<tag>"` |
| Ambiguous | `skpp: ambiguous tag '<tag>', candidates: <space-joined Candidates>` | `ambiguous skill tag "<tag>" matches: <comma-joined>` |

Notes:
- Tag is wrapped in **single quotes** `'`, not double quotes.
- Prefix is `skpp:` (program context — resolve is prefix-free, like skillsdir.ErrNotFound;
  main adds the `skpp:` context).
- Candidates are **SPACE-joined** (`strings.Join(ae.Candidates, " ")`), NOT comma-joined.
  They are already sorted (resolve's sortedRelTags).
- One line per problem tag, in INPUT order. Example: `skpp a nope1 nope2` where nope1/nope2
  fail → two stderr lines (nope1 then nope2), nothing on stdout, exit 1.

**Exact format strings (use these verbatim):**
```go
fmt.Fprintf(stderr, "skpp: unknown tag '%s'\n", ue.Tag)
fmt.Fprintf(stderr, "skpp: ambiguous tag '%s', candidates: %s\n", ae.Tag, strings.Join(ae.Candidates, " "))
```

---

## §6 — main.go integration (what changes, what stays)

### parseArgs — ADD positional tag capture (currently tolerates everything)
The current `default` case tolerates ALL unknown tokens (comment: "tolerated for
now"). This task must CAPTURE positional args as tags while KEEPING unknown flags
tolerated (exit-2 is M5's job). Rule: a token starting with `-` is a flag (known
above, or tolerated unknown); anything else is a positional `<tag>`.

```go
default:
    if strings.HasPrefix(a, "-") {
        // unknown flag: tolerated (no-op); P1.M5.T11 -> exit 2 (§6.2)
    } else {
        c.tags = append(c.tags, a)  // positional <tag> (§6.1 row 1)
    }
```
- A bare `-` (single dash) → HasPrefix true → treated as tolerated flag (no-op).
  Edge case; M5 decides its fate. Not load-bearing.
- This ADDS a `tags []string` field to `config` and makes parseArgs import `strings`.

### run — ADD a tag-resolution branch (after --path, before the no-args default)
Precedence (PRD §6.3): `--version` (and `--help` in M5) > everything. Then modes.
Existing order: version → path. ADD: version → path → **tags** → no-args default(1).

```go
if len(c.tags) > 0 {
    return resolveTags(c.tags, stdout, stderr)
}
```
- `--version` + tags ⇒ version wins (precedence), tags ignored. ✓ (§6.3)
- `--path` + tags ⇒ path wins (not forbidden by §6.3; mode-exclusivity exit-2 is M5).
- no-args ⇒ tags empty ⇒ falls through to `return 1` (unchanged no-args behavior).

### resolveTags — NEW private helper (the §6.4 atomicity implementation)
Factored out of run() for readability + as the S2 extension point. Calls
skillsdir.Find → discover.Index → resolve.Resolve per tag. Returns exit code.
See the PRP Implementation Blueprint for the verbatim source.

---

## §7 — All existing tests STILL PASS (verified by tracing)

Traced every test in main_test.go against the parseArgs/run changes:
- `TestParseArgsEmpty/Version*/Path*/AnyOrder*` — unchanged behavior. ✓
- `TestParseArgsUnknownTolerated` — input was `["--frobnicate","sometag","check"]`;
  now `sometag`/`check` become tags. The assertion (version/path false) STILL HOLDS,
  but the test's INTENT ("everything tolerated") is now inaccurate. → UPDATE it to
  flags-only input + assert tags empty, and ADD `TestParseArgsCapturesPositionalTags`.
- `TestRunVersion*/Path*/VersionPrecedenceOverPath` — unchanged. ✓
- `TestRunDefaultNoArgs` (`run(nil)`) — no tags → `return 1`. ✓
- `TestRunDefaultUnknownFlag` (`run(["--frobnicate"])`) — flag tolerated, no tags →
  `return 1`. ✓

No existing test breaks. The change is purely additive (a new mode + tag capture).

---

## §8 — Test strategy (integration via run(); real skills tree fixtures)

main's `run(args, stdout, stderr) int` is already the tested entry (existing tests
inject `*bytes.Buffer`). The tag-resolution tests are INTEGRATION tests: they build
a real on-disk skills tree, set `SKPP_SKILLS_DIR` (rule 1 wins), and call `run()`,
exercising Find→Index→Resolve→print end-to-end. This is the right level — unit tests
of Resolve/Index live in their own packages; main's job is the §6.4 wiring.

**Helpers (mirror discover_test.go's writeSkill idiom, adapted to a tree):**
- `skillsTree(t)` → temp dir + `t.Setenv("SKPP_SKILLS_DIR", dir)`; returns dir.
- `writeSkill(t, root, relTag, frontmatter, body)` → `root/<relTag>/SKILL.md`.

**Required tests (item TEST spec):**
1. multi-tag all-good → N lines in INPUT order (not sorted), exit 0, empty stderr.
2. one-bad-tag (mixed good+bad) → NOTHING on stdout, one error line per problem tag,
   exit 1.
3. ambiguous → candidates (sorted, space-joined) on stderr, nothing on stdout, exit 1.
4. stdout/stderr captured SEPARATELY (two `*bytes.Buffer`) — proven by every test.

**Extra tests (robustness):** single resolve+absolute-path check, multiple-bad-all-
reported, skills-dir-not-found (Find fails → ErrNotFound on stderr), version-
precedence-over-tags, parseArgs tag capture + flags/tags mixed.

**Gotcha — Index-of-empty:** an empty skills dir returns `([], nil)` (Index is
lenient). To avoid coupling a test to that assumption, the multi-bad test writes ONE
unrelated real skill so Index is non-empty, then queries non-existent tags.

---

## §9 — go.mod / go.sum are UNCHANGED (dependency-neutral)

main.go adds imports `errors`, `strings` (stdlib) + the INTERNAL `discover` and
`resolve` packages. `discover` already pulls `gopkg.in/yaml.v3` (flipped to direct
by P1.M2.T4.S1's `go mod tidy`) — it is ALREADY in go.mod. `resolve` is stdlib-only.
So `go mod tidy` is a NO-OP and go.mod/go.sum are byte-identical. Verify with
`git diff --quiet go.mod go.sum`. (Same property as the resolve PRP.)

---

## §10 — DOWNSTREAM EXTENSION POINTS (what this task establishes for S2)

P1.M3.T2.S2 (modifiers) layers onto the atomic-output pattern this task builds:
- `--file`/`-f`: print `r.Skill.SourceFile` instead of `r.Skill.Dir`.
- `--relative`: print path relative to the skills dir instead of absolute.
- `--all`/`-a`: iterate ALL skills (not tags), still through the same buffer+atomic
  print step.

The extension point is the SINGLE print step in resolveTags's success path
(`fmt.Fprintln(&paths, r.dir)`). S2 parameterizes WHAT is printed (Dir vs SourceFile,
abs vs rel) WITHOUT touching the resolve-all/buffer/error machinery. The PRP marks
this clearly so S2's author knows exactly where to hook in. This task must NOT add
those flags (scope theft) — it only makes the print step a clean, single location.

---

## §11 — Scope boundary (do NOT touch)

- `internal/discover/*` (M2-owned) — only IMPORT.
- `internal/resolve/*` (M3.T7-owned) — only IMPORT.
- `internal/skillsdir/*` (M1-owned) — only IMPORT.
- `go.mod`/`go.sum` — dependency-neutral (verify with git diff).
- `PRD.md` (read-only), any `tasks.json` (orchestrator-owned).
- Do NOT create `ui/`, `cmd/`, `install.sh`, `README.md`, `skills/`, or the modifiers
  (those are later milestones).
- ONLY `main.go` and `main_test.go` are modified.
