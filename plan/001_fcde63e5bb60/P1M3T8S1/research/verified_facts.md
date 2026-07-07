# Verified Facts — P1.M3.T8.S1 (`skpp <tag> [...]` output + §6.4 atomicity)

Verification of the proposed `main.go` + `main_test.go` changes, run against a
**verbatim copy of the real module** (M1 + M2 + resolve all LANDED on disk) in a
throwaway `/tmp/skpp_t8_verify` on go1.25 (the repo's `go.mod` version). The source
given verbatim in `../PRP.md` is what was compiled, formatted, vetted, tested, and
smoke-tested against a real built binary.

## 0. Baseline (before this subtask)

- `go build ./...` OK; `go test ./...` OK.
- Packages landed + green: `internal/discover`, `internal/skillsdir`,
  `internal/ui`, `internal/resolve`, `main`.
- **`resolve.Resolve(tag string, skills []discover.Skill) (Result, error)` is
  LANDED** (`internal/resolve/resolve.go`, commit `c7f4f99`). `Result.Skill` is a
  `discover.Skill` whose `Dir` field is the **absolute** skill-directory path (set
  by `discover.Index` from the absolute skills dir). This is the value `main`
  prints per resolved tag.
- Typed errors LANDED: `*resolve.UnknownError` (`.Error()` =
  ``unknown skill tag "foo"``) and `*resolve.AmbiguousError` (`.Error()` =
  ``ambiguous skill tag "reddit" matches: coding/reddit, writing/reddit`` —
  candidates pre-sorted by resolve). main prints these **verbatim** to stderr.

## 1. The change (4 edits to `main.go`; all additive; 0 new files)

1. `import`: add `"strings"` (stdlib) and `"github.com/dabstractor/skpp/internal/resolve"`.
2. `config`: add `tags []string` (positional `<tag>` args).
3. `parseArgs` `default` branch: capture tokens that do **not** start with `-` into
   `c.tags`; dashed unknowns (`--frobnicate`) still tolerated (no-op) → M5 exit 2.
4. `run`: new tag-resolution branch **after** the `--list` branch and **before** the
   default `return 1`. Implements the §6.4 atomicity contract via buffering.

## 2. Gates (all PASS — exact commands in PRP §Validation Loop)

| Gate | Result |
|---|---|
| `gofmt -l main.go main_test.go` | **EMPTY** (after `gofmt -w`; the only reformat was struct-field alignment — `bool` fields padded to match the `[]string` line) |
| `go vet ./...` | **clean** |
| `go build ./...` | **OK** |
| `go test ./...` | **all green**; main package = **36 PASS** (was 24 → +12 new/updated) |
| `go mod tidy` then `git diff --quiet go.mod go.sum` | **byte-identical** (no new dep: `strings` is stdlib, `resolve` is internal) |

## 3. New / updated tests (12 total; main_test.go goes 24 → 36)

**parseArgs (4):** `CapturesTagsInOrder`, `DashedUnknownNotATag`,
`TagsAndFlagsInterleave`, `UnknownTolerated` *(updated to assert positionals are
now captured as tags, not discarded)*.

**run tag-resolution (8):** `TagSingleResolvesToDir` (absolute dir + newline),
`TagMultipleInInputOrder` (input order, not sorted), `TagAtomicityUnknownPrintsNothing`
(stdout EMPTY when one tag fails), `TagAllFailMultipleErrorLines` (one stderr line
per problem tag), `TagDuplicateArgResolvesTwice`, `TagAmbiguousListsCandidates`
(stderr lists candidates, stdout empty), `TagSkillsDirUnresolvable`, `TagPathIsAbsolute`,
`VersionPrecedenceOverTag`.

## 4. End-to-end binary smoke (real built binary; `SKPP_SKILLS_DIR` store)

```
1) skpp example            -> /tmp/.../skills/example        (absolute dir path)
2) skpp reddit example     -> two lines in INPUT order
3) skpp example nope       -> stdout EMPTY, rc=1, stderr names "nope"
4) skpp nope               -> stderr: unknown skill tag "nope"; stdout empty; rc=1
5) skpp reddit (ambiguous) -> stdout EMPTY; stderr: ambiguous skill tag "reddit"
                                    matches: coding/reddit, writing/reddit; rc=1
6) skpp example example    -> two identical path lines
```

All six observed behaviors match the PRD §6.1/§6.4 contract on the real binary.

## 5. Locked decisions (load-bearing — each maps to a PRP "Known Gotcha")

1. **Positional-capture rule:** a token that does NOT start with `-` is a `<tag>`;
   a dashed token is a flag (known → sets a config field; unknown → tolerated, M5 →
   exit 2). This cleanly separates tags from flags and needs no subcommand special-
   casing in S1.
2. **Atomicity via buffering:** resolve EVERY tag first into a buffered `[]string`
   of paths; flush to stdout **only** when the whole invocation is known-good. Any
   failure → stderr lines per problem tag, **nothing** on stdout, exit 1.
3. **Default output = absolute directory path** (`res.Skill.Dir`). `--file` (→
   `SKILL.md`) and `--relative` are **S2**, not here.
4. **Error lines are verbatim** `err.Error()` from resolve's typed errors — **no
   `skpp:` prefix**, matching the `skillsdir.ErrNotFound` convention already used by
   `--path`/`--list` (which print `err` verbatim to stderr).
5. **`run` branch order:** `--version` → `--path` → `--list` → `<tag>...` → default
   `return 1`. (Mixing `<tag>` with `--list` is tolerated here — `--list` wins; the
   §6.3 exit-2 mutual-exclusivity is M5.T11.)

## 6. Scope boundary (what is NOT in this subtask)

- **OUT (P1.M3.T8.S2):** `--file`/`-f`, `--relative`, `--all`/`-a` modifiers.
- **OUT (M4):** `--search`/`-s`; the `check` subcommand.
- **OUT (M5.T11):** `--help`/`-h`; unknown-flag → exit 2; §6.3 mutual-exclusivity →
  exit 2 (tag + `--list`/`--search`/`--all`).
- **`skpp check` interim behavior:** because `check` is a non-dashed token, S1
  captures it as a tag and tries to resolve it (→ unknown, exit 1). This matches the
  pre-S1 behavior (tolerated default → exit 1) in exit code; the only change is a
  stderr line now appears. **M4.T10** adds real `check` subcommand dispatch (before
  tag resolution). A user who legitimately has a skill tagged `check` gets it
  resolved — that is correct and does not paint M4 into a corner.
