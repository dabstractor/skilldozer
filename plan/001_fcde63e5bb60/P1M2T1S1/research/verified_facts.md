# Verified Facts — P1.M2.T4.S1: Frontmatter type + ParseFrontmatter (yaml.v3, lenient)

Every claim below was **executed** against the real `gopkg.in/yaml.v3 v3.0.1` in
`go version go1.26.4-X:nodwarf5 linux/amd64` (the project's toolchain), using a
throwaway module in `/tmp` that required the SAME `yaml.v3 v3.0.1` already pinned
in `go.sum`. The exact struct + parser algorithm proposed for this PRP were
compiled and run against 9 fixture files. Raw output is summarized per fact.

The repo state at authoring time (read directly):

```
go.mod     module github.com/dabstractor/skpp ; go 1.25 ; require gopkg.in/yaml.v3 v3.0.1 // indirect
go.sum     has yaml.v3 v3.0.1 h1: + go.mod: lines (+ the check.v1 go.mod line)
internal/  only skillsdir/ (S1+S2+S3 landed); NO discover/ yet
module cache: /home/dustin/go/pkg/mod/gopkg.in/yaml.v3@v3.0.1 (present; no network needed)
```

---

## §1 — yaml.v3 is LENIENT by default: unknown keys are silently ignored

**Fixture** (excerpt): a frontmatter block containing `unknown-key: whatever`
alongside the known fields.

**Result**: `yaml.Unmarshal(block, &fm)` returned `nil` error and populated the
known fields correctly; `unknown-key` was dropped (not stored anywhere). No
`KnownFields(true)` was called.

```text
=== full+unknown+folded+quoted ===
HasFM=true Name="my-skill" Desc="One to two sentences: ..." License="MIT" ...
```

**Decision locked**: do NOT call `dec.KnownFields(true)`. The package-level
`yaml.Unmarshal` (which we use) is already lenient. This matches pi's behavior
(PRD §7.3) and the item's "lenient: ignore unknown keys". Calling
`KnownFields(true)` would make unknown keys HARD-ERROR — the opposite of what we
want. Verified the default is the lenient path.

---

## §2 — `map[string]any` receives nested lists as `[]interface{}` + scalars as `string`

**Fixture**:
```yaml
metadata:
  keywords: [writing, reddit]      # flow list
  category: writing                # scalar
  aliases:                          # block list
    - reddit-post
    - social-post
```

**Result** (`%T` of each value):

```text
  meta[keywords]=[]interface {}=[]interface {}{"writing", "reddit"}
  meta[category]=string="writing"
  meta[aliases]=[]interface {}=[]interface {}{"reddit-post", "social-post"}
```

**Decision locked**: `Metadata map[string]any` (tag `metadata,omitempty`) works
for both flow `[a, b]` and block `- a` list forms, and for scalars. yaml.v3
unmarshals YAML lists into `[]interface{}` (== `[]any` in Go 1.18+). This means
**T4.S2's metadata extraction must type-assert `[]any` → `[]string`** (NOT
`[]string` directly — yaml.v3 never produces `[]string`). This subtask (S1) only
exposes the raw `map[string]any`; S2 owns the `toStringSlice` helper. No work for
S1 here beyond exposing `Metadata` with the right type.

---

## §3 — Folded scalar `>` is handled; it KEEPS a trailing `\n`

**Fixture**:
```yaml
description: >
  One to two sentences: what this skill does
  and precisely when to use it.
```

**Result**:

```text
Desc="One to two sentences: what this skill does and precisely when to use it.\n"
```

yaml.v3 folded-scalar semantics: consecutive non-empty indented lines are joined
with a single space, and the value retains a trailing `\n` (the newline before
the next top-level key). **Gotcha for downstream** (NOT for this subtask): a
`description: >` value will be ~1 char longer than its visible text because of
the trailing `\n`. `check`/`--search` (M4) and the 1024-char validation (T10)
should `strings.TrimSpace` before length-checking if they want the visible
length. **ParseFrontmatter returns the value verbatim — do NOT trim here**
(preserve fidelity; trimming would also alter a `|` literal-block scalar).

---

## §4 — Quoted values are unquoted; spaces preserved

**Fixture**: `name: "my-skill"`, `compatibility: "Requires Python 3.11+"`.

**Result**: `Name="my-skill"` (no quotes), `Compat="Requires Python 3.11+"`
(spaces kept). yaml.v3 handles double-quoted, single-quoted, and unquoted
scalars uniformly. Nothing for S1 to do beyond letting yaml.v3 unmarshal.

---

## §5 — BOM (U+FEFF) MUST be stripped before fence detection

A UTF-8 BOM is the 3 bytes `0xEF 0xBB 0xBF`. `os.ReadFile` returns them verbatim
at the start of a BOM-marked file. If they are NOT stripped, `strings.Split`
yields `lines[0] == "\ufeff---"`, which `!= "---"` → frontmatter is NOT detected
→ a valid BOM-marked SKILL.md is misclassified as no-frontmatter.

**Verified approach** (works):

```go
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}
data = bytes.TrimPrefix(data, utf8BOM)   // strip ONE leading BOM if present
```

**Result** with a BOM-prefixed file: `HasFM=true Name="bom-skill"` — correctly
detected. `bytes.TrimPrefix` is a no-op when there is no BOM (safe for normal
files). We strip AT MOST one BOM, only at the very start — exactly the item's
"after trimming a leading BOM".

NOTE: do NOT pass the BOM to yaml.v3 expecting it to cope — we slice the inner
YAML block ourselves, so by the time yaml.v3 sees the block the BOM is long gone.
BOM handling matters ONLY for recognizing the opening `---` fence.

---

## §6 — CRLF (Windows) line endings: trim trailing `\r` for fence comparison

**Fixture**: `"---\r\nname: crlf-skill\r\n---\r\n# body\r\n"`.

**Approach**: split on `"\n"`, then compare each candidate fence line with
`strings.TrimRight(line, "\r") == "---"`. This treats `---\r` as a fence on
Windows files.

**Result**: `HasFM=true Name="crlf-skill"` — correctly detected. body retains
its `\r` (`body="# body\r\n"`) because we only strip `\r` for the FENCE
comparison, not for the body slice. That body-`\r` is harmless (skpp does not
consume body yet; S2's Skill has no Body field). Verified acceptable.

---

## §7 — No closing `---` fence ⇒ treat as NO frontmatter (lenient, no error)

**Fixture**: `"---\nname: dangling\ndescription: no close\n"` (opening fence,
content, EOF — no second `---`).

**Result**: `HasFM=false`, **no error**, body = the whole file content. This is
the item's "If no closing `---`, treat as no frontmatter (lenient)." Also covers
the `"---\n"`-only file (opening fence, immediate EOF): `lines = ["---", ""]`,
loop from i=1 finds no closing fence → no frontmatter. Verified.

**Contrast with §8**: a MISSING closing fence is lenient (no error); MALFORMED
YAML between valid fences is a hard error. These are distinct cases.

---

## §8 — Malformed YAML between valid fences ⇒ HARD ERROR (propagate it)

**Fixture**: `"---\nname: bad\nmetadata: [unbalanced\n---\nbody\n"` (an
unbalanced flow list inside the YAML block).

**Result**: `ParseFrontmatter` returns `err = yaml: line 1: did not find
expected ',' or ']'`, `fm = Frontmatter{}` (zero), body = "".

**Decision locked**: lenient means "ignore unknown KEYS", NOT "tolerate
syntactically-broken YAML". If `yaml.Unmarshal` errors, ParseFrontmatter
returns that error (wrapped context optional but keep the yaml message
visible). The item's leniency is about unknown keys (§7.3), not about corrupt
YAML. Returning the error here is correct and lets `check` (M4) flag the file.

---

## §9 — Empty file: no panic, no frontmatter, body = ""

**Fixture**: `""` (zero bytes). `strings.Split("", "\n")` returns `[""]`
(len 1). `lines[0] == ""` ≠ `"---"` → no frontmatter branch → returns
`Frontmatter{}, "", nil`. No crash. Verified. An empty SKILL.md is an oddity
but must not panic the parser.

---

## §10 — `body` semantic: WHOLE file content when there is no frontmatter

When `HasFM == false`, `body` is set to the ENTIRE file content (BOM-stripped,
otherwise byte-identical to the input). Verified: for a no-frontmatter fixture,
`body == input` was `true`.

**Rationale**: a frontmatter parser's "body" is the non-frontmatter content; if
there is no frontmatter, the whole file is body. This is the Hugo/Jekyll
convention and is strictly more useful than returning `""` (a caller wanting the
markdown never has to re-read the file). NOTE: the `Skill` struct (S2) does NOT
store body, so this is currently a convenience for direct callers / future
`check` body validation. Low-risk choice; documented for the implementer.

When `HasFM == true`, body is the text AFTER the closing `---` fence line
(everything from `lines[closeIdx+1:]` rejoined with `\n`).

---

## §11 — Struct shape: field NAME for `disable-model-invocation`

The architecture doc (`external_deps.md §3`) shows an abbreviated field name
`DisableModelInv`. The **item description** (authoritative contract for THIS
subtask) spells the field `DisableModelInvocation` with tag
`disable-model-invocation`. Verified both compile and unmarshal identically
(the tag is what yaml.v3 reads; the Go field name is free). **Decision: use the
full name `DisableModelInvocation`** (matches the item verbatim). S2 does not
read this field (it only touches `Metadata`), so the choice is low-risk, but it
is documented here so S2+ stay consistent. `AllowedTools` likewise uses tag
`allowed-tools`.

---

## §12 — `HasFM bool` MUST be on the Frontmatter struct (tag `yaml:"-"`)

The item's listed struct fields (Name, Description, License, Compatibility,
Metadata, AllowedTools, DisableModelInvocation) are the YAML-backed fields. But
the item's BEHAVIOR spec says: "return empty Frontmatter with **HasFM=false**
and NO error" — which is only expressible if `HasFM` is a field on the struct.
The architecture's `Skill.HasFM` (go_architecture.md) is propagated FROM
`Frontmatter.HasFM`. **Decision: add `HasFM bool `yaml:"-"`** (non-YAML
metadata field; `yaml:"-"` ensures yaml.v3 never tries to read/write a
frontmatter key named `hasfm`). The zero value is `false`, so the no-frontmatter
return (`Frontmatter{}`) already has `HasFM == false` without setting it; the
parser sets `fm.HasFM = true` only on the successful-fence path. Verified this
compiles and round-trips correctly (the `yaml:"-"` field was never touched by
unmarshal in any fixture).

---

## §13 — Scope: S1 owns Frontmatter + ParseFrontmatter ONLY; S2 owns the Skill type

The item TITLE says "Frontmatter/Skill types" but the item's LOGIC section
(authoritative) specifies ONLY the `Frontmatter` struct and `ParseFrontmatter`.
The plan's S2 is explicitly "metadata extraction (keywords/category/aliases) +
**Skill type**". To avoid colliding with S2, **S1 does NOT define**:

- `type Skill struct` (S2 owns it: Dir, RelTag, Name, Description, Keywords,
  Category, Aliases, HasFM, SourceFile)
- `toStringSlice(v any) []string` (S2's metadata-extraction helper)
- any `BuildSkill(...)` constructor
- `Index()` (S5 owns it)

S1's deliverable is exactly: the `Frontmatter` struct (with `HasFM`), the
`ParseFrontmatter` function, and its white-box tests. S2 will consume
`Frontmatter` (Name/Description/Metadata/HasFM) verbatim.

---

## §14 — go.mod: yaml.v3 flips from `// indirect` to direct after this subtask

`go.mod` currently lists `require gopkg.in/yaml.v3 v3.0.1 // indirect` because
no non-test code imports it yet (it was added in S1 as a forward dep). Once
`internal/discover/discover.go` imports `gopkg.in/yaml.v3`, it becomes a DIRECT
dependency. `go build`/`go vet`/`go test` all STILL PASS with the stale
`// indirect` marker (the marker is purely informational), but `go mod tidy`
removes the `// indirect` comment to reflect direct usage. **This is an EXPECTED,
legitimate go.mod change for this subtask** (unlike M1.T3.S1 which promised no
go.mod change because main.go is pure stdlib). go.sum is unchanged (yaml.v3 is
already checksummed). The implementer should run `go mod tidy` as a hygiene step;
the only diff is the `// indirect` token disappearing from the yaml.v3 line.

---

## §15 — Package + import set + test convention

- Package: `package discover` in `internal/discover/discover.go` (internal ⇒
  unimportable outside the module; correct for a CLI's private packages).
- Imports (production file): `bytes`, `os`, `strings`, `gopkg.in/yaml.v3`. That's
  it — no `io`, `fmt`, `path/filepath` needed (ParseFrontmatter takes a path
  string and reads it; it does not walk or join).
- Test file: `package discover` (WHITE-BOX, same package) — mirrors
  `internal/skillsdir/skillsdir_test.go` which is `package skillsdir`. White-box
  is needed to assert on any unexported helpers if added (none required here) and
  matches the repo convention. Tests use `t.TempDir()`, `os.WriteFile`, plain
  `t.Errorf`/`t.Fatalf`, NO testify, NO `t.Parallel()` (not strictly required to
  avoid parallel here since ParseFrontmatter touches no env/cwd, but the repo
  convention is no-Parallel across the board — follow it).
- No new directories beyond `internal/discover/`. No `main.go` touch (M1.T3.S1
  owns main.go; this subtask is a leaf library with no dependency on main).
