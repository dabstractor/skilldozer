# README Update Anchors — Scout Findings

Source files (read-only, unmodified):
- `README.md` (271 lines)
- `skills/example/SKILL.md`

All quotes below are **verbatim with line numbers**, verified against the current files on 2026-07-07.

---

## 1. Install → "B. `go install`" caveat block (README lines 36–53)

This is the block the PRP removes (PRD §9: the caveat is "obsolete under the config model").

```text
36: **B. `go install`**
37: 
38: ```bash
39: go install github.com/dabstractor/skilldozer@latest
40: ```
41: 
42: > **`go install` caveat.** A `go install`'d binary lands in
43: > `$(go env GOPATH)/bin` with **no** adjacent `skills/` directory, so `skilldozer`
44: > cannot auto-discover the store from there. Set the runtime override before
45: > use:
46: >
47: > ```bash
48: > export SKILLDOZER_SKILLS_DIR=/absolute/path/to/your/cloned/skilldozer/skills
49: > ```
50: >
51: > If you hit `skilldozer example` reporting it cannot find skills, this is the fix.
52: > (Prefer `./install.sh`, which symlinks the binary next to the repo so
53: > discovery works with no env var.)
```

Context: it sits between the "A. `./install.sh`" block (ends ~line 34) and "C. From source" (line 55+). The recommended-path callout that should be **kept** under the new model is at PRD §9 line 335: "`go install …` is a first-class install path: the binary is self-sufficient … on first use the user runs `skilldozer init`, which creates the store and writes the config. **No clone required, no `SKILLDOZER_SKILLS_DIR` needed for normal use.**"

---

## 2. "## How `skilldozer` finds the store" (README lines 234–251)

Currently a **3-rule** priority + `--path` labels paragraph. PRP turns it into a **5-rule** ladder (env → config → sibling → walk-up → fail). Verbatim:

```text
234: ## How `skilldozer` finds the store
235: 
236: `skilldozer` locates `skills/` by this priority:
237: 
238: 1. **`SKILLDOZER_SKILLS_DIR` env var**: wins if set and the directory exists. This
239:    is the override `go install` users set (see Install).
240: 2. **Sibling of the binary**: `os.Executable()` plus `EvalSymlinks()` resolves
241:    the real binary path and looks for `skills/` next to it. This is the rule a
242:    `./install.sh` symlink install relies on; a copy would break it silently.
243: 3. **Walk up from the current directory**: useful during development
244:    (`go run .` / running `./skilldozer` from a checkout).
245: 4. **Else: fail with a one-line fix** telling you how to set `SKILLDOZER_SKILLS_DIR`.
246: 
247: `skilldozer --path` reports the winning directory on stdout and the matching rule on
248: stderr — one of `SKILLDOZER_SKILLS_DIR`, `sibling of binary`, or `ancestor of cwd`.
249: The stderr label matters when `SKILLDOZER_SKILLS_DIR` is typo'd: a bad value is
250: silently ignored and discovery falls through to the sibling / walk-up rule, so
251: the `(found via …)` line is the only way to tell the env var was skipped.
```

Target shape (from PRD §8.3 + §8.4 line 226): insert a new **rule 2** "Config file `store` (§8.1)" and a new **rule 5** "None ⇒ unconfigured: stderr one-line fix (`run \`skilldozer init\``), exit 1." The `--path` label list (line 248) must grow from 3 → 4 labels: `SKILLDOZER_SKILLS_DIR`, **`config file`**, `sibling of binary`, `ancestor of cwd`. Note rule 1's "This is the override `go install` users set (see Install)" (line 239) is stale wording once the go-install caveat is gone — PRP may want to reword.

---

## 3. "## Constraints" section (README lines 253–271)

Full section verbatim. The **Manifest-free** bullet (lines 255–257) is the one to reword per PRD §9 line 419 / §A2 line 469:

```text
253: ## Constraints
254: 
255: `skilldozer` is deliberately a thin, manifest-free path printer.
256: 
257: - **Manifest-free.** No `skills.json`, no index file. Everything is resolved
258:   from the directory tree on each call.
259: - **Never auto-discovered by pi.** The skills store does **not** live in any
260:   directory pi scans. It is never:
261:   - `~/.pi/agent/skills`
262:   - `~/.agents/skills`
263:   - a project `.pi/skills` or `.agents/skills`
264:   - a `node_modules` package
265:   - a `package.json` with a `pi.skills` field
266: - **Loaded only via `--skill`.** A skill enters your context only when you ask
267:   for it explicitly: `pi --skill "$(skilldozer <tag>)"`.
268: - **`skilldozer` only ever prints paths.** It never copies or installs skills into
269:   `~/.pi/...` or `~/.agents/...`. Where the path points is up to you.
270: - **Zero runtime dependencies.** Build-time needs Go; the resulting binary
271:   stands alone.
```

Target reword (PRD §9 line 419, exact): "**Constraints:** no catalog index (disk-discovered); a settings config file is fine; never auto-discovered by pi; loads only via `--skill`." Concretely, the **Manifest-free** bullet should say "no catalog index (disk-discovered); a settings config file is fine" — distinguishing the *forbidden catalog index* from the *permitted settings sidecar*. The opening line 255 `manifest-free` wording may also need softening so it doesn't contradict the new config file. PRD §A2 line 469 notes the original "explicit user constraint" framing for catalog-free was a misattribution.

---

## 4. Forbidden-term grep against README.md — NONE FOUND ✅

```
$ grep -niE 'skilldozer init|config\.yaml|SKILLDOZER_CONFIG|config file' README.md
(no output; exit 1)
```

README currently mentions **none** of: `skilldozer init`, `config.yaml`, `SKILLDOZER_CONFIG`, or the phrase "config file". (These terms DO appear in `main.go`, `main_test.go`, `internal/config/config.go`, `PRD.md`, and `plan/.../tasks.json` — i.e., the code already implements the config model, but the README has not been updated. That is exactly the gap this PRP closes.)

---

## 5. Title one-liner + canonical "## Usage" example block (voice/flag-preservation anchors)

### Title one-liner (README line 3)

```text
3: Standalone skill loader for pi. Resolves a skill tag to an absolute path for `pi --skill`.
```

(Header is line 1 `# Skilldozer`; line 2 blank.) Match this voice/length if editing.

### "## Usage" section (README lines 75–120) — flag docs to preserve verbatim

```text
75: ## Usage
76: 
77: The canonical one-liner, first:
78: 
79: ```bash
80: pi --skill "$(skilldozer example)"
81: ```
82: 
83: Everything else, commented:
84: 
85: ```bash
86: # Resolve a tag to an absolute path (default: the skill directory)
87: skilldozer example                       # → /…/skills/example
88: 
89: # Print the SKILL.md path instead of the directory (-f / --file)
90: skilldozer -f example                    # → /…/skills/example/SKILL.md
91: 
92: # Load several skills into pi in one command
93: pi --skill "$(skilldozer writing/reddit)" --skill "$(skilldozer example)"
94: 
95: # Resolve multiple tags at once (one absolute path per line, input order)
96: skilldozer example writing/reddit
97: 
98: # Human-readable catalog and substring search
99: skilldozer --list
100: skilldozer --search reddit            # matches tag / name / description / keywords / aliases / category
101: 
102: # Print every skill path, sorted by tag
103: skilldozer --all
104: 
105: # Validate every skill on disk
106: skilldozer check
107: 
108: # Where is the resolved skills directory? (its discovery rule prints to stderr)
109: skilldozer --path                        # → /…/skills (stderr: found via sibling of binary)
110: 
111: # Print paths relative to the skills directory instead of absolute
112: skilldozer --relative example
113: 
114: # Disable ANSI color even on a TTY (for --list / --search tables)
115: skilldozer --no-color --list
116: 
117: # Version is the git-describe value (dynamic, not a fixed string)
118: skilldozer --version
119: 
120: # Short flags combine (-af) and long flags accept --flag=value (--search=reddit)
121: ```
```

Note: line 109's example output `stderr: found via sibling of binary` is one of the 3 old `--path` labels — under the config model a configured install would more likely report `found via config file`, but this is illustrative, not a contract; PRP can keep `sibling of binary` as the zero-config dev example or update it. The full flag inventory to preserve: `example` (tag), `-f/--file`, `--list`, `--search`, `--all`, `check`, `--path`, `--relative`, `--no-color`, `--version`, and the short-combine / `--flag=value` note. New flag to **add** in the Usage section under the config model: `skilldozer init [<dir>]` / `skilldozer init --store <dir>` (PRD §8.2 line 117).

---

## 6. Example skill frontmatter — `skills/example/SKILL.md`

```text
---
name: example
description: >
  Reference example skill for skilldozer. Demonstrates the required frontmatter and
  how skilldozer resolves a tag to an absolute path. Safe to delete once you add real skills.
metadata:
  keywords: [example, demo, skilldozer]
  category: meta
---
```

Schema (relevant to Usage line 100 "matches tag / name / description / keywords / aliases / category"): top-level `name`, `description` (block-scalar `>`), nested `metadata.keywords` (YAML list), `metadata.category`. No `aliases` key present in this example (but Usage line 100 lists `aliases` as a searchable field, so other skills may carry it). Body opens with `# Example Skill` and shows the three resolution forms (`skilldozer example`, `skilldozer -f example`, `pi --skill "$(skilldozer example)"`).

---

## Summary of anchors for the PRP

| # | Anchor | README lines | Action |
|---|--------|-------------|--------|
| 1 | go-install caveat block | 36–53 | Remove (replace with "first-class" wording from PRD §9:335) |
| 2 | "How … finds the store" | 234–251 | 3 rules → 5 rules; 3 `--path` labels → 4 (add `config file`) |
| 3 | Constraints / Manifest-free | 253–271 (esp. 255–258) | Reword to "no catalog index (disk-discovered); a settings config file is fine" |
| 4 | Forbidden terms | whole file | Confirmed absent (grep exit 1) — README predates the config model |
| 5 | Title (3) + Usage (75–121) | 3, 75–121 | Preserve voice/flags; add `skilldozer init` example |
| 6 | Example SKILL.md frontmatter | (separate file) | Reference schema for `--search` field list |
