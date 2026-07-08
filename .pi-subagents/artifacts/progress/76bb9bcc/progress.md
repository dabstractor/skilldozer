# Progress — 76bb9bcc

## 002_38acb6d28a6a — Research: external/stdlib facts for config + init model (PRD §8)

- **Status:** DONE. Research brief written.
- **Output:** `plan/002_38acb6d28a6a/architecture/external_deps.md`
  (authoritative path: `.pi-subagents/artifacts/outputs/76bb9bcc/plan/002_38acb6d28a6a/architecture/external_deps.md`)
- **Scope:** external + Go stdlib idioms only. No source files changed.
  Dependency invariant preserved: `gopkg.in/yaml.v3` stays the sole non-stdlib module.

### Key results
1. **yaml.v3**: plain `yaml.Unmarshal` is lenient → unknown keys ignored (config file can grow). `Load`/`Save` patterns drafted. Confirmed in-repo at `internal/discover/discover.go`.
2. **XDG**: defaults `~/.config` (config), `~/.local/share` (data). `os.UserConfigDir()` is free; there is NO `os.UserDataDir()` — compute data-home by hand. Planned paths confirmed spec-correct.
3. **TTY**: stdlib `os.Stdin.Stat()&os.ModeCharDevice` — repo already uses this pattern (`main.go`). Do NOT add `golang.org/x/term`.
4. **Prompts**: `bufio.NewReader(os.Stdin).ReadString('\n')`; empty/EOF ⇒ accept default.
5. **os.Executable + EvalSymlinks**: repo already implements symlink-aware sibling rule (`skillsdir.go`). `go install` copies to `$GOPATH/bin` (no sibling skills/) → justifies the new config rule.
6. **pi --skill**: skill = dir with SKILL.md; pi takes dir OR SKILL.md path ("parent of SKILL.md / dirname"). Confirmed via skilldozer README + pi-subagents README. GAP: `--no-skills` + explicit `--skill` interaction not found on disk — verify with `pi --help`.

### Open / gaps
- Capture pi `--skill`/`--no-skills` exact flag text via `pi --help | grep -i skill` before any skilldozer help copy relies on it.
