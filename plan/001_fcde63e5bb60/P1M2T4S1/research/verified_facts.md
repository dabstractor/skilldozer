# Verified Facts — P1.M2.T4.S1: Frontmatter type + ParseFrontmatter (yaml.v3, lenient)

Every claim below was **executed** against the real `gopkg.in/yaml.v3 v3.0.1`
(the version pinned in the repo's `go.sum`) on `go1.26.4 linux/amd64`, using a
throwaway module that required the same `yaml.v3 v3.0.1`. The exact struct +
parser algorithm proposed for this PRP (see PRP.md §Implementation Blueprint)
were compiled and run against 8 fixture files. Raw output is summarized per fact.

This file **supersedes and consolidates** the prior research artifact at
`plan/001_fcde63e5bb60/P1M2T1S1/research/verified_facts.md` (same task, earlier
numbering) — all 15 facts there hold and were re-confirmed by the run below.

## Repo state at authoring time (read directly)

```
go.mod     module github.com/dabstractor/skpp ; go 1.25 ; require gopkg.in/yaml.v3 v3.0.1 // indirect
go.sum     yaml.v3 v3.0.1 present (h1: + go.mod: lines)
internal/  only skillsdir/ (S1+S2+S3 landed)
module cache: /home/dustin/go/pkg/mod/gopkg.in/yaml.v3@v3.0.1 (present; no network needed)
main.go    landed (M1.T3: main.go + main_test.go committed). Irrelevant to THIS
           subtask: discover is a leaf library with no dep on main.go.
```

## The run (8 fixtures, one ParseFrontmatter each)

Proposed struct + algorithm compiled clean and produced:

```text
=== full ===   (name+desc+license+compat+metadata{keywords,category,aliases}+allowed-tools+disable-model-invocation+unknown-key)
  err=<nil> HasFM=true Name="my-skill" Desc="One to two sentences: ... precisely when to use it.\n"
  License="MIT" Compat="Requires Python 3.11+" AllowedTools="shell exec" DisModInv=true
  meta[keywords]=[]interface {}={"writing", "reddit"}
  meta[category]=string="writing"
  meta[aliases]=[]interface {}={"reddit-post", "social-post"}
  body="# Body\n"
=== bom ===    (leading U+FEFF)
  err=<nil> HasFM=true Name="bom-skill"  body="body\n"
=== crlf ===   (Windows \r\n endings)
  err=<nil> HasFM=true Name="crlf-skill"  body="# body\r\n"
=== no-close ===  (opening fence, no closing fence)
  err=<nil> HasFM=false  body=<whole file>
=== malformed === (unbalanced flow list `[unbalanced` between valid fences)
  err="yaml: line 1: did not find expected ',' or ']'"  HasFM=false  body="body\n"
=== empty ===  (zero bytes)
  err=<nil> HasFM=false  body=""
=== no-fm ===  (plain markdown, no fence)
  err=<nil> HasFM=false  body=<whole file>
=== only-fence === ("---\n")
  err=<nil> HasFM=false  body="---\n"
```

## Decisions locked (each traceable to the run above)

1. **Lenient = ignore unknown KEYS, not tolerate broken YAML.** `unknown-key`
   dropped silently (no error). Malformed `[unbalanced` → hard error propagated.
   Do NOT call `dec.KnownFields(true)` (that would hard-error on unknown keys —
   the opposite of PRD §7.3). The package-level `yaml.Unmarshal` is already
   lenient. ✓ fixtures `full`, `malformed`.
2. **`Metadata map[string]any`** receives flow lists `[a, b]` and block lists as
   `[]interface{}` (== `[]any`), scalars as `string`. yaml.v3 NEVER produces
   `[]string`, so S2's `toStringSlice` must assert `[]any`→`[]string`. S1 only
   EXPOSES the raw map; S2 owns extraction. ✓ fixture `full`.
3. **Folded scalar `>` KEEPS a trailing `\n`.** `description: >` → `"...use it.\n"`.
   ParseFrontmatter returns it VERBATIM (do not trim — would also corrupt `|`
   literal blocks). T10's 1024-char check trims if it wants visible length. ✓
   fixture `full`.
4. **Quoted values unquoted; spaces preserved.** `"Requires Python 3.11+"` →
   `Requires Python 3.11+`. Nothing for S1 to do. ✓ fixture `full`.
5. **UTF-8 BOM (0xEF 0xBB 0xBF) MUST be stripped before fence detection.** Else
   `lines[0] == "\ufeff---"` and frontmatter is silently missed. Use
   `bytes.TrimPrefix(data, utf8BOM)` — no-op when absent. ✓ fixture `bom`.
6. **CRLF: trim trailing `\r` for the `---` comparison ONLY.** `body` retains its
   `\r` (harmless; skpp does not consume body). ✓ fixture `crlf`.
7. **No closing fence → lenient, NO frontmatter, NO error.** body = whole file.
   Distinct from #8. ✓ fixtures `no-close`, `only-fence`.
8. **Malformed YAML between valid fences → HARD ERROR (propagated).** The yaml.v3
   message is returned. ✓ fixture `malformed`.
9. **Empty file → no panic, HasFM=false, body="".** `strings.Split("", "\n")` →
   `[""]` (len 1); `lines[0]=""` ≠ `"---"`. ✓ fixture `empty`.
10. **`body` = the non-frontmatter portion of the file, always.** Whole file when
    no frontmatter; everything after the closing `---` otherwise — INCLUDING on
    the malformed-YAML error path (the run shows `body="body\n"` there). This is
    strictly more useful than `""` and consistent with the "body = non-frontmatter"
    model. The earlier P1M2T1S1 research observed `body=""` on that path; that was
    an artifact of computing body only on the success path. PRP specifies the
    consistent model (post-fence body returned regardless of unmarshal outcome);
    tests assert `err != nil && HasFM==false`, NOT the body value, so this is not
    brittle. ✓ fixtures `malformed`, `no-fm`, `no-close`, `empty`.
11. **Field name `DisableModelInvocation`, tag `disable-model-invocation`.**
    `AllowedTools` is a `string` (spec: space-delimited) with tag `allowed-tools`.
    Both compile + unmarshal correctly. ✓ fixture `full`.
12. **`HasFM bool `yaml:"-"`** MUST be on the Frontmatter struct. Non-YAML field;
    zero value `false` so `Frontmatter{}` already means "no frontmatter". A
    frontmatter key literally named `hasfm` must NOT set it (verified: the tag
    `yaml:"-"` keeps yaml.v3 off the field). Propagated into `Skill.HasFM` by S2/T5.
13. **SCOPE: S1 owns Frontmatter + ParseFrontmatter ONLY.** The plan's S2 is
    explicitly "metadata extraction + **Skill type**". S1 does NOT define
    `type Skill struct`, `toStringSlice`, `BuildSkill`, or `Index()` — declaring
    Skill here would steal S2's deliverable and create churn. The task title's
    "Skill types" refers to the frontmatter data model that the Skill type is
    built FROM; the Skill struct itself lands in S2. This is the
    non-overlapping split.
14. **go.mod: yaml.v3 flips `// indirect` → direct.** Once `internal/discover/
    discover.go` imports `gopkg.in/yaml.v3`, it is a DIRECT dependency. Builds
    still pass with the stale `// indirect` marker, but `go mod tidy` removes the
    comment. This is an EXPECTED, legitimate go.mod diff for THIS subtask (unlike
    M1.T3.S1 which promised no go.mod change because main.go is pure stdlib).
    go.sum is unchanged. Run `go mod tidy` as hygiene; the only diff is the
    `// indirect` token disappearing.
15. **Package + imports + test convention.** `package discover` in
    `internal/discover/discover.go`. Production imports: `bytes`, `os`, `strings`,
    `gopkg.in/yaml.v3` (no `fmt`/`io`/`path/filepath` — ParseFrontmatter takes a
    path string and reads it; it does not walk or join). Test file: `package
    discover` (WHITE-BOX, mirrors `internal/skillsdir/skillsdir_test.go`):
    `t.TempDir()` + `os.WriteFile`, plain `t.Errorf`/`t.Fatalf`, NO testify, NO
    `t.Parallel()` (repo convention is no-Parallel across the board even where
    env/cwd are untouched). No new dirs beyond `internal/discover/`. No main.go
    touch.

## Reproducing this run

The throwaway verifier lives at `/tmp/skpp_fm_verify/main.go` (built during
authoring; not part of the repo). To re-verify after implementation:

```bash
cd /home/dustin/projects/skpp
go test ./internal/discover/ -v   # the PRP's discover_test.go covers all 8 cases
```
