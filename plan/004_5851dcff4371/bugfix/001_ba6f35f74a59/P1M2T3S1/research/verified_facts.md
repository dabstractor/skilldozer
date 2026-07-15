# Verified facts — P1.M2.T3.S1: Clarify version string documentation in README (Issue 5)

Scope: **a single-line comment edit in `README.md`.** Documentation-only — NO code
changes (contract OUTPUT §4 + DOCS §5). The README change IS the entire deliverable.

All facts below read directly from the current tree on 2026-07-07.

---

## 1. The exact edit (one line; anchor by TEXT — it is unique in README.md)

**File:** `README.md`, line **136**.

The line lives INSIDE the `## Usage` section's ` ```bash ` fenced code block (the
"Everything else, commented:" example block). It is a **bash comment** (prefixed `#`).
It MUST stay a `#`-prefixed bash comment inside that fence — do NOT convert it to
markdown prose or strip the `#` (contract LOGIC §3: "Keep the line as a bash comment
(prefixed with #) in the Usage examples block").

| | text |
|---|---|
| **OLD (README.md:136)** | `# Version is the git-describe value (dynamic, not a fixed string)` |
| **NEW (authoritative — contract LOGIC §3)** | `# Version is the git-describe value when built via ./install.sh; a plain 'go build' reports 'dev'` |

`grep -c 'Version is the git-describe' README.md` → **1** (the line is unique; a
single `edit`-tool replacement with the OLD text as `oldText` matches exactly once).

The contract's exact wording (single quotes around `'go build'` and `'dev'`, NO quotes
around `./install.sh`) is AUTHORITATIVE — it wins over issue_analysis.md's Fix Surface,
which uses markdown backticks. **Use the contract's single-quoted form**, not backticks:
backticks inside a ` ```bash ` fence render as literal backtick characters and are
stylistically odd in a bash comment (single quotes read naturally and are
command-substitution-free).

---

## 2. Why the old line is wrong — the three build paths

The README `## Install` section (lines 19-58) documents THREE install paths:

| Path | Command | Version injected? | `skilldozer --version` |
|---|---|---|---|
| **A. `./install.sh`** (recommended) | `./install.sh` | **YES** | the `git describe` value |
| **B. `go install`** | `go install github.com/dabstractor/skilldozer@latest` | no | `dev` |
| **C. From source** | `go build -o skilldozer .` | no | `dev` |

Only path A injects the version. **install.sh:40** is the sole injection site:
```sh
go build -trimpath \
  -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
  -o skilldozer .
```
Paths B and C run a plain `go build`/`go install` with NO `-ldflags`, so the `version`
var keeps its default and the binary reports `dev`. The old README comment claimed the
version is "the git-describe value (dynamic, not a fixed string)" — true ONLY for path A,
contradicting the very "From source" (C) instructions the README itself documents
(issue_analysis.md §Issue 5). The new line accurately describes BOTH build paths.

---

## 3. The `version` var — main.go:44 (NOT 57; the contract/issue cite drifted)

Both the contract RESEARCH NOTE and issue_analysis.md §Issue 5 cite `main.go:57`, but
`var version = "dev"` is currently at **main.go:44** (the go:embed completion directives
at 54-59 pushed things; line numbers drifted). **Anchor by the symbol** `var version = "dev"`,
not a line number. (This drift does NOT affect the README edit — it is context only; the
README comment does not cite a line number.)

`grep -n 'var version' main.go` → `44:var version = "dev"`. The default `"dev"` is exactly
what paths B/C report; the `-X main.version=…` injection overrides it only for path A.

---

## 4. This is a Mode A documentation fix (the README change IS the deliverable)

- Contract DOCS §5: "[Mode A] This subtask IS a documentation fix. The README change is the
  entire deliverable. No code changes."
- Contract OUTPUT §4: "Modified README.md with the corrected version documentation line."
- So: edit README.md ONLY. Do NOT touch main.go, install.sh, main_test.go, completions/*, etc.

**Relationship to P1.M3.T1.S1** (the later Mode B "changeset-level documentation sync"
sweep): that task sweeps README.md + help text + completion headers for whole-changeset
consistency. This subtask is the **targeted Mode A fix** for the version-comment line
specifically. No conflict: this PRP fixes the line authoritatively; the later sweep would
merely confirm it (and address other lines). Land this independently.

---

## 5. Disjointness from the parallel sibling (no collision)

The parallel sibling **P1.M2.T2.S1** (Issue 4, POSIX `--` separator) edits **main.go**
(parseArgs loop head + a parse-local bool + a doc comment) and **main_test.go** (5 new
tests). The completed **P1.M2.T1.S1** (Issue 3, missing-value) also edited main.go +
main_test.go.

This subtask edits **README.md ONLY** (line 136). There is **zero file-level overlap**
with either sibling (neither touches README.md). No merge collision; lands in any order.

---

## 6. Validation approach (doc-only; no unit test for the change itself)

There is no code under test for this change, so the validation is:
1. **grep the README** — the NEW line is present exactly once; the OLD line is GONE.
2. **grep the README** — the line is still inside the ` ```bash ` fence and still a `#`
   comment (i.e. the fence + comment structure is intact).
3. **`git diff --name-only`** → ONLY `README.md` (scope discipline: no code files touched).
4. **`go build/vet/test ./...`** → still green — this PROVES no code was changed (the build
   is unaffected by a README edit; if it regressed, a code file was touched by mistake).
5. (Optional) **markdown render sanity** — the ` ```bash ` fence still opens/closes cleanly
   (a stray edit could unbalance it). A quick `awk` fence-balance check or a visual read of
   the surrounding block confirms it.

No `tasks.json`/`PRD.md`/`prd_snapshot.md`/`.gitignore` changes (all read-only / owned by
humans/orchestrator).
