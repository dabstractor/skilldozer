# External Dependencies & Verified Specs — `skpp`

All facts below were verified against authoritative sources during research.
PRD claims cross-checked: all accurate.

## 1. Agent Skills specification (agentskills.io/specification)

VERIFIED via web search of the live spec + pi's own docs. This is the source of
truth for SKILL.md frontmatter.

### Frontmatter schema

A `SKILL.md` file = YAML frontmatter delimited by lines that are exactly `---`,
followed by Markdown content.

| Field | Required | Constraints (verified) |
|---|---|---|
| `name` | YES | 1–64 chars; lowercase `a-z`, `0-9`, hyphens only; NO leading/trailing/consecutive hyphens. (Spec says name MUST match dir; pi relaxes this for shared dirs. skpp does NOT require match.) |
| `description` | YES | Max 1024 chars. Drives pi on-demand loading AND skpp `--search`. A skill with NO description is NOT loaded by pi. |
| `license` | no | License name or ref to bundled file. |
| `compatibility` | no | Max 500 chars. Environment requirements. |
| `metadata` | no | Arbitrary key-value map (spec'd optional field). Holds skpp conventions: `keywords` (list), `category` (string), `aliases` (list). Nesting lists under metadata IS spec-compliant. |
| `allowed-tools` | no | Space-delimited pre-approved tools (experimental). |
| `disable-model-invocation` | no | When `true`, hidden from system prompt; users must `/skill:name`. |

### Name validation regex (for `skpp check`, §9)

Rules: `^[a-z0-9]+(-[a-z0-9]+)*$`, length 1–64. Concretely:
- only lowercase a-z, 0-9, hyphen,
- no leading/trailing hyphen,
- no consecutive hyphens.

A Go-valid expression: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` plus a length check
1..64, plus a "no `--` substring" check (the regex already forbids consecutive
hyphens via the alternation, but verify explicitly).

### Leniency rules (matches pi behavior)

- Unknown frontmatter keys are IGNORED (do not error).
- Missing optional keys ⇒ defaults.
- No frontmatter block ⇒ skill still resolves by directory (tag/basename) but
  `check` flags it and `--list` shows description as `(missing)`.

## 2. pi (the consumer) — verified from `docs/skills.md` (pi 0.80.3)

Confirmed behaviors that drive skpp's design:

- `--skill <path>`: "Load a skill file or directory (can be used multiple times)".
  Accepts EITHER a `SKILL.md` file path OR a skill directory. Additive; works
  even with `--no-skills`. This is why skpp defaults to emitting the DIRECTORY
  (includes assets) and offers `--file` for the SKILL.md path.
- Discovery: pi scans skill locations and extracts name+description at startup.
- Progressive disclosure: only descriptions are in context; full SKILL.md loads
  on-demand via `read`.
- Name collisions warn and keep FIRST skill found.
- pi does NOT require `name` to match the directory (relaxes the standard).

## 3. Go third-party dependency — `gopkg.in/yaml.v3`

This is the ONLY third-party dependency (PRD §7.3). Justification:
- Robustly handles quoted/multiline scalars and the nested `metadata` map with
  list values.
- A hand-rolled parser is acceptable ONLY if it correctly handles quoted values
  and the metadata map; prefer yaml.v3.

Import path: `gopkg.in/yaml.v3`. Module path: `github.com/dabstractor/skpp`.

### Frontmatter struct shape (recommendation for `internal/discover`)

```go
// Minimal frontmatter model. Unknown keys ignored by yaml.v3 default.
type Frontmatter struct {
    Name             string            `yaml:"name"`
    Description      string            `yaml:"description"`
    License          string            `yaml:"license,omitempty"`
    Compatibility    string            `yaml:"compatibility,omitempty"`
    Metadata         map[string]any    `yaml:"metadata,omitempty"`  // keywords/category/aliases live here
    AllowedTools     string            `yaml:"allowed-tools,omitempty"`
    DisableModelInv  bool              `yaml:"disable-model-invocation,omitempty"`
}
```

`metadata.keywords` / `metadata.category` / `metadata.aliases` are read via the
`map[string]any` and type-asserted (keywords/aliases → `[]string`, category →
string), because they are skpp conventions, not fixed spec fields.

## 4. Go standard library APIs to use (verified available in Go 1.26)

| Need | API |
|---|---|
| Find binary, resolve symlinks | `os.Executable()` + `filepath.EvalSymlinks()` |
| Walk skills dir | `filepath.WalkDir` (uses `fs.WalkDirFunc`; skips content efficiently) |
| Path joining/separators | `path/filepath` (Join, Dir, Base, Abs); normalize OS sep → `/` for relTag |
| Output buffering / exit codes | `os.Stdout`/`os.Stderr`, `os.Exit()` |
| ANSI color | hand-rolled constants (no dep); gate on `isatty` + `--no-color` |

## 5. No other runtime dependencies

- No Node, Python, or any runtime at RUN time. Build-time `go` only.
- No network access at run time. Fully offline, disk-derived.
