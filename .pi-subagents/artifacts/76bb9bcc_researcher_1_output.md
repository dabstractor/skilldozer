# Research: External/stdlib facts for skilldozer config + init model (PRD §8)

**Scope.** External + Go standard-library idioms needed to implement the new
config-file + `init` model for the `skilldozer` Go CLI. The repo
(`github.com/dabstractor/skilldozer`, go 1.25) already pins
`gopkg.in/yaml.v3 v3.0.1` as its **only** third-party dependency (confirmed in
`go.mod`/`go.sum`). Every recommendation below keeps that invariant: yaml.v3
remains the sole non-stdlib module. Where the repo already does the right thing,
that is called out so the new code matches the established style.

---

## Summary

All six facts check out and align with existing repo patterns. yaml.v3 ignores
unknown keys by default (so a config file can grow without breaking older
binaries), and `Marshal`/`Unmarshal` round-trip a tiny struct cleanly. XDG
defaults are `~/.config` and `~/.local/share`; `os.UserConfigDir()` implements
the config-home default for free, but there is **no** `os.UserDataDir()`, so the
data-home default must be computed by hand. TTY detection, interactive stdin
reading, and the symlink-aware binary-path resolution are all already done in
the repo with the idiomatic stdlib approach — the new code should reuse those
exact patterns. The pi `--skill` "skill = dir with SKILL.md" model is confirmed
by the on-disk pi-subagents README and the project's own README; the only thing
not verifiable from an on-disk doc is the `--no-skills` + explicit `--skill`
interaction, which should be confirmed with `pi --help`.

---

## Findings

### 1. yaml.v3 — read a struct, write a struct, ignore unknown keys

**The repo already uses this exact idiom.** `internal/discover/discover.go`
unmarshals SKILL.md frontmatter with `yaml.Unmarshal([]byte(yamlBlock), &f)`
into a `Frontmatter` struct tagged `yaml:"name"`, `yaml:"metadata,omitempty"`,
etc., and its doc comment explicitly states the key guarantee we rely on:

> "Unknown keys are ignored by yaml.v3's default (lenient) decoder, matching pi's behavior." — `internal/discover/discover.go`

**READ pattern (small config struct):**

```go
package config

type File struct {
    Store string `yaml:"store"`
}

// Load reads the config file at path. A missing file is a distinct, expected
// condition (first run / not yet initialized); callers test errors.Is(err,
// fs.ErrNotExist) to decide whether to fall back to defaults rather than abort.
func Load(path string) (File, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return File{}, err // os.ReadFile wraps ENOENT as *fs.PathError -> fs.ErrNotExist
    }
    var f File
    if err := yaml.Unmarshal(data, &f); err != nil {
        return File{}, err // syntactically broken YAML is a hard error (unknown KEYS are not)
    }
    return f, nil
}
```

**WRITE pattern (Marshal + MkdirAll + WriteFile):**

```go
// Save serializes f to path, creating the parent config directory if needed.
// yaml.Marshal returns deterministic output (struct-field order, not sorted).
func Save(path string, f File) error {
    out, err := yaml.Marshal(&f)
    if err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return err
    }
    return os.WriteFile(path, out, 0o644)
}
```

A `File{Store: "/home/u/.local/share/skilldozer/skills"}` marshals to exactly:

```yaml
store: /home/u/.local/share/skilldozer/skills
```

**Unknown keys ARE ignored (the forward-compat guarantee).** `yaml.Unmarshal`
does **not** enable strict decoding. Strict mode exists, but only via an explicit
`Decoder`:

```go
d := yaml.NewDecoder(bytes.NewReader(data))
d.KnownFields(true) // <-- opt-in: now unknown keys return an error
if err := d.Decode(&f); err != nil { ... }
```

Because `Load` uses the plain `yaml.Unmarshal` helper, it never calls
`KnownFields(true)`, so a file like

```yaml
store: /abs/path
version: 3          # added later — ignored by today's binary
future:
  cache: true       # ignored by today's binary
```

unmarshals cleanly into `File{Store: "/abs/path"}`. **This is what lets the
config file grow** without forcing a coordinated binary upgrade: old binaries
silently ignore new keys, and new binaries with a larger `File` struct read the
extra fields when present. [Source: yaml.v3 `Decoder.KnownFields`
docs](https://pkg.go.dev/gopkg.in/yaml.v3#Decoder.KnownFields); default
`Unmarshal` behavior is lenient. Confirmed in-repo at
`internal/discover/discover.go`.

**Marshal notes for writing a stable, diff-friendly file:**
- yaml.v3 marshals keys in **struct-definition order** (not alphabetical). Put
  the fields in the order you want them on disk.
- `omitempty` keeps optional fields out of the written file. For the single
  `store` field, omit `omitempty` only if you want an empty `store:` line to
  always be present; otherwise tag it `yaml:"store,omitempty"`.
- yaml.v3 does not emit a trailing document marker or BOM; the output is plain
  `key: value\n`. Safe to `git diff`.

---

### 2. XDG semantics on Linux — defaults and Go resolution

**XDG Base Directory Specification defaults (when the env var is unset or empty):**

| Var | Default if unset/empty |
|-----|------------------------|
| `$XDG_CONFIG_HOME` | `$HOME/.config` |
| `$XDG_DATA_HOME`  | `$HOME/.local/share` |
| `$XDG_CACHE_HOME` | `$HOME/.cache` |
| `$XDG_RUNTIME_DIR` | (no default; must be set by the OS/login manager) |

[Source: XDG Base Directory Specification, freedesktop.org](https://specifications.freedesktop.org/basedir-spec/latest/).
The spec also requires that if an `XDG_*_HOME` var is **set, it must be
absolute**; a relative value is invalid and the application should treat it as
unset (fall back to the default).

**Go resolution — config home is free; data home is NOT.**

`os.UserConfigDir()` implements the config-home XDG rule for you:

> "On Unix (including macOS), it returns `$XDG_CONFIG_HOME` if non-empty, else
> `$HOME/.config`. … If `$XDG_CONFIG_HOME` is set but not absolute, it returns
> an error." — [Go `os` package docs](https://pkg.go.dev/os#UserConfigDir)

So `os.UserConfigDir()` is the correct, spec-compliant source for the config
root. There is **no** `os.UserDataDir()` in the standard library, so the
data-home default must be computed explicitly:

```go
// configHome returns $XDG_CONFIG_HOME or ~/.config. Prefer os.UserConfigDir():
// it already encodes the XDG default AND rejects a relative $XDG_CONFIG_HOME.
func configHome() (string, error) {
    return os.UserConfigDir()
}

// dataHome returns $XDG_DATA_HOME if it is an absolute path, else ~/.local/share.
// There is no os.UserDataDir(); this mirrors the XDG spec rule by hand.
func dataHome() (string, error) {
    if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {
        return v, nil
    }
    home, err := os.UserHomeDir() // → $HOME (or %USERPROFILE%); errors if unset
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".local", "share"), nil
}
```

(`os.UserHomeDir()` returns `$HOME` on Unix and errors if it is unset; it is the
right primitive for the fallback. [Source: Go `os.UserHomeDir`
docs](https://pkg.go.dev/os#UserHomeDir).)

**skilldozer's planned paths — confirmed spec-correct:**

| Purpose | Planned path | Expanded default |
|---------|--------------|------------------|
| Config file | `$XDG_CONFIG_HOME/skilldozer/config.yaml` | `~/.config/skilldozer/config.yaml` |
| Default store dir | `$XDG_DATA_HOME/skilldozer/skills` | `~/.local/share/skilldozer/skills` |

Both follow the XDG convention exactly: a per-application subdirectory
(`skilldozer/`) under the relevant XDG root, then the file/sub-tree. This is the
same shape systemd, kubectl, docker, and most modern CLIs use. The full
on-disk location is therefore:

```
configHome, _ := os.UserConfigDir()
configPath := filepath.Join(configHome, "skilldozer", "config.yaml")

dataHome, _ := dataHome()
defaultStore := filepath.Join(dataHome, "skilldozer", "skills")
```

---

### 3. TTY detection with the standard library (no extra dep)

**The repo already does this** — `main.go` defines:

```go
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
```

This is the idiomatic zero-dependency check: `os.File.Stat()` returns an
`os.FileMode`; the `os.ModeCharDevice` bit (`os.ModeDevice | os.ModeCharDevice`
under the hood) is set for character-special files such as a real terminal.

**For stdin specifically** (to decide whether to show the interactive `init`
prompts), the same primitive applies directly:

```go
func stdinIsTTY() bool {
    fi, err := os.Stdin.Stat()
    if err != nil {
        return false
    }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

[Source: Go `os.FileMode` / `os.ModeCharDevice`
docs](https://pkg.go.dev/os#FileMode). This correctly reports **false** for the
two cases that matter most for a CLI: a pipe (`skilldozer init < file`,
`echo y | skilldozer init`) and a regular-file redirect. When false, `init`
should **not block on stdin** — apply defaults non-interactively (or error out
if a required value is missing) rather than hang waiting for input that will
never arrive.

**Known caveat.** `/dev/null` is *also* a character device, so this heuristic
reports `true` for `skilldozer init < /dev/null`. For the prompt/skip decision
this is harmless (an empty read on /dev/null yields immediate EOF, which the
prompt reader treats as "accept default"). The precise alternative is the real
`isatty(0)` ioctl via `golang.org/x/term.IsTerminal(int(os.Stdin.Fd()))`.

**Recommendation: keep the stdlib approach.** `golang.org/x/term` (and its
`golang.org/x/sys` dependency) would be an *additional* runtime module beyond
the yaml.v3-only constraint. The `ModeCharDevice` heuristic is already the
established repo pattern for stdout color gating; reusing it for the stdin
prompt decision keeps behavior consistent and adds zero dependencies. Reserve
`x/term` only if `/dev/null`-as-TTY proves to be a real problem (it has not).

---

### 4. Interactive prompt reading from stdin (bufio.Reader.ReadString)

The right primitive for a single-line "prompt → accept default on empty Enter"
is `bufio.Reader.ReadString('\n')`. (Prefer it over `bufio.Scanner`: Scanner is
built for streaming many lines, has a configurable but finite token-size limit,
and does not make the "empty line ⇒ default" intent as clean.)

```go
// readPrompt prints label + [def] to w, reads one line from r, and returns the
// trimmed answer — or def when the user just presses Enter (empty line) or
// sends EOF (Ctrl-D) on an otherwise-empty line. A genuine read error is returned.
func readPrompt(r *bufio.Reader, w io.Writer, label, def string) (string, error) {
    if def != "" {
        fmt.Fprintf(w, "%s [%s]: ", label, def)
    } else {
        fmt.Fprintf(w, "%s: ", label)
    }
    line, err := r.ReadString('\n')     // includes the trailing '\n'
    if err != nil && err != io.EOF {
        return "", err
    }
    if s := strings.TrimSpace(line); s != "" {
        return s, nil
    }
    return def, nil // empty Enter OR EOF-with-no-text ⇒ accept default
}

// Usage inside init, guarded by the stdin-TTY check from §3:
r := bufio.NewReader(os.Stdin)
store, err := readPrompt(r, os.Stdout, "Skills directory", defaultStore)
```

Notes:
- `ReadString('\n')` returns `(line, error)` where `error` is `io.EOF` if the
  delimiter is not found before end of input. Treat a bare `io.EOF` with empty
  text as "accept default", not a hard error — that is what makes
  `skilldozer init < /dev/null` and `echo | skilldozer init` behave like "press
  Enter" instead of crashing. [Source: Go `bufio.Reader.ReadString`
  docs](https://pkg.go.dev/bufio#Reader.ReadString).
- Wrap stdin in **one** `bufio.NewReader` and reuse it for all prompts; creating
  a fresh reader per prompt can swallow buffered bytes from the previous read.
- Gate the whole prompt block behind `stdinIsTTY()` (§3) so piped/redirected
  invocations never block.

---

### 5. os.Executable() + filepath.EvalSymlinks — symlink-installed binaries

**The repo already implements this** as discovery rule 2 in
`internal/skillsdir/skillsdir.go`:

```go
func resolveSiblingFromExe(exe string) (dir string, found bool) {
    real, err := filepath.EvalSymlinks(exe)
    if err != nil {
        real = exe // EvalSymlinks could not resolve -> use exe verbatim
    }
    repoDir := filepath.Dir(real)
    candidate := filepath.Join(repoDir, "skills")
    info, err := os.Stat(candidate)
    if err != nil || !info.IsDir() {
        return "", false
    }
    return candidate, true
}
```

**Why EvalSymlinks is required (platform split):**
- `os.Executable()` returns "the path of the running binary". On **Linux** it
  reads `/proc/self/exe`, which is itself a symlink that the kernel points at the
  *real* executable — so it already resolves the install symlink chain and
  `EvalSymlinks` is a redundant-but-harmless no-op. On **macOS** it uses
  `_NSGetExecutablePath`, which can return the *symlink* path
  (`~/.local/bin/skilldozer`), so `filepath.EvalSymlinks` is the step that walks
  back to the real binary sitting next to `skills/`. Keeping `EvalSymlinks` in
  both cases makes the sibling rule correct on both OSes.
  [Source: Go `os.Executable` docs](https://pkg.go.dev/os#Executable),
  [Go `filepath.EvalSymlinks` docs](https://pkg.go.dev/path/filepath#EvalSymlinks).

**`./install.sh` symlink install → sibling rule HITS.**
`install.sh` symlinks `~/.local/bin/skilldozer` → `<repo>/skilldozer`. After
`EvalSymlinks`, the resolved path is `<repo>/skilldozer`, so `filepath.Dir(...)/skills`
is `<repo>/skills`. This is exactly why the README recommends `./install.sh`.

**`go install` → sibling rule MISSES (the justification for the config rule).**
`go install github.com/dabstractor/skilldozer@latest` compiles the package and
**copies** the resulting binary into `$(go env GOPATH)/bin` (default `~/go/bin`).
There is no symlink, so `os.Executable()` returns the real path
`~/go/bin/skilldozer` (and `EvalSymlinks` is a no-op). The directory
`~/go/bin/skills` does not exist, so the sibling lookup fails; the walk-up rule
also fails unless the user happens to be inside the repo. With no
`SKILLDOZER_SKILLS_DIR` set, discovery today falls all the way through to
`ErrNotFound`. **This is precisely why the new config-file rule is needed** — it
gives `go install` users a persistent, discoverable default store
(`$XDG_DATA_HOME/skilldozer/skills`) and a config file to record it, instead of
requiring a shell env var that is easy to typo (the README's documented caveat).
[Source: Go `go install` / GOPATH docs](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies);
`go install` places output in `$GOBIN` (default `$GOPATH/bin`).

**Precedence implication for the new model.** The config file slots in as an
authoritative source of the store path that sits between the env override and
the heuristics. Concretely, the discovery order becomes (env var → config-file
`store` → sibling → walk-up), where the config-file `store` is what `init`
writes after the user confirms the default `$XDG_DATA_HOME/skilldozer/skills`
(or types a custom path). This keeps `./install.sh` symlink users working
unchanged (their sibling rule still wins if config is absent) while finally
giving `go install` users a sane default.

---

### 6. The pi `--skill` / skills contract

Verified against two on-disk sources.

**(a) The skill model — "a skill is a directory that directly contains a
`SKILL.md`."** From the project README (`README.md`, "Where skills live"):

> "Skills live in the `skills/` directory at the repo root. A skill is any
> directory that directly contains a `SKILL.md`."

And the canonical invocation that skilldozer exists to serve (`README.md`,
"Why"):

> "They load **only on demand** when you pass an explicit `--skill`:
> `pi --skill "$(skilldozer example)"`"

**(b) pi identifies a skill by its SKILL.md path, and treats the dir and the
SKILL.md file as equivalent (dir = parent of SKILL.md).** From the pi-subagents
package README (`~/.pi/agent/npm/node_modules/pi-subagents/README.md`,
"Skills"):

> "Skills are `SKILL.md` files made available to an agent. The prompt includes
> skill metadata and the file location; the agent reads the full skill file only
> when the task matches."

with the available-skill shape giving a SKILL.md **file** path:

> "`<location>/absolute/path/to/safe-bash/SKILL.md</location>`"

and the resolution rule that makes dir-or-file interchangeable:

> "When a skill file references a relative path, resolve it against the skill
> directory (parent of SKILL.md / dirname of the path) …"

That "parent of SKILL.md / dirname of the path" sentence is the proof that pi
accepts **both** a directory (the default `skilldozer <tag>` output →
`pi --skill "<dir>"`) **and** the `SKILL.md` file itself (the `skilldozer -f
<tag>` output → `pi --skill "<file>"`). skilldozer's `-f`/`--file` modifier
exists precisely to feed pi the file form.

The same README documents the programmatic skill controls, which mirror the CLI
flags: `{ skill: "tmux, safe-bash" }` enables specific skills and
`{ skill: false }` disables skill auto-discovery while still letting an explicit
`--skill` path load.

**Could NOT be confirmed from an on-disk doc:** the exact pi CLI flag spelling
and the "--no-skills still allows explicit `--skill`" interaction. I searched
the standard locations (`~/.pi/docs/skills.md`, `~/.pi/agent/docs/*`,
`~/.pi/agent/npm/node_modules/pi/docs/*`, the `pi-subagents` docs/ and skills/
trees, and `~/.pi/agent/extensions/*`) and did not find a standalone pi-CLI
`docs/skills.md`. The flags are properties of the `pi` binary (a Go CLI), not of
the `pi-subagents` npm package whose README/SKILL.md I did read. **Recommended
verification:** run `pi --help | grep -iE 'skill'` to capture the authoritative
`--skill <path>` / `--no-skills` text before relying on the `--no-skills` +
explicit `--skill` interaction in any skilldozer help copy. The *skill identity
model* (dir with SKILL.md; pi takes a dir or a SKILL.md path via `--skill`) is
solidly confirmed by (a) and (b) above.

---

## Sources

**Kept (primary / authoritative):**
- `go.mod` + `go.sum` — confirms `gopkg.in/yaml.v3 v3.0.1` is the sole non-stdlib dependency; go 1.25.
- `internal/discover/discover.go` — in-repo proof that yaml.v3 `Unmarshal` ignores unknown keys by default; the established read idiom to match.
- `main.go` (`isTerminal`) — in-repo proof that the stdlib `ModeCharDevice` TTY check is the established pattern (no `x/term`).
- `internal/skillsdir/skillsdir.go` (`resolveSiblingFromExe`) — in-repo proof that `os.Executable()` + `filepath.EvalSymlinks()` is the established symlink-aware sibling rule.
- `README.md` — the pi `--skill` contract skilldozer relies on ("skill = dir with SKILL.md", canonical one-liner).
- `~/.pi/agent/npm/node_modules/pi-subagents/README.md` ("Skills" section) — confirms pi identifies a skill by its `SKILL.md` file path and that "skill directory (parent of SKILL.md / dirname of the path)" makes dir/file interchangeable.
- [yaml.v3 docs — Decoder.KnownFields](https://pkg.go.dev/gopkg.in/yaml.v3#Decoder.KnownFields) — strict mode is opt-in; default `Unmarshal` is lenient (unknown keys ignored).
- [os.UserConfigDir](https://pkg.go.dev/os#UserConfigDir) / [os.UserHomeDir](https://pkg.go.dev/os#UserHomeDir) — config-home default is free; there is no data-home helper.
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/) — `$XDG_CONFIG_HOME`→`~/.config`, `$XDG_DATA_HOME`→`~/.local/share`; set values must be absolute.
- [os.FileMode / ModeCharDevice](https://pkg.go.dev/os#FileMode), [bufio.Reader.ReadString](https://pkg.go.dev/bufio#Reader.ReadString), [os.Executable](https://pkg.go.dev/os#Executable), [filepath.EvalSymlinks](https://pkg.go.dev/path/filepath#EvalSymlinks) — the stdlib primitives behind findings 3–5.
- [go install / GOPATH docs](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies) — `go install` copies the binary to `$GOBIN` (default `$GOPATH/bin`), which has no sibling `skills/`.

**Dropped:**
- `~/.pi/agent/npm/node_modules/pi-subagents/skills/pi-subagents/SKILL.md` — read; it documents subagent *orchestration*, not the pi `--skill` flag. Not relevant to point 6's flag contract.

## Gaps

- **pi `--no-skills` + explicit `--skill` interaction:** not located as an on-disk `docs/skills.md`. The skill *identity* model (dir with SKILL.md; pi takes a dir or a SKILL.md path) is confirmed, but the exact flag semantics should be captured from `pi --help` before any skilldozer help text asserts the `--no-skills`/`--skill` interaction. Next step: `pi --help | grep -iE 'skill'`.
- **`os.UserConfigDir` cross-platform mapping** (Windows `%AppData%`, macOS `~/Library/Application Support`) is noted for completeness but the PRD targets Linux; only the Linux XDG mapping is load-bearing here.
- The `/dev/null`-reports-as-TTY edge case of the `ModeCharDevice` heuristic is documented; not a blocker (EOF ⇒ default).

## Implementation guidance (one-screen recap for the worker)

1. New `internal/config` package: `File{ Store string \`yaml:"store,omitempty"\` }`, `Load(path)` (plain `yaml.Unmarshal`, treat `fs.ErrNotExist` as "not initialized"), `Save(path)` (`yaml.Marshal` + `MkdirAll` + `WriteFile 0644`). Matches `discover.go`'s lenient-unmarshal convention.
2. Paths: `configPath = filepath.Join(must(os.UserConfigDir()), "skilldozer", "config.yaml")`; `defaultStore = filepath.Join(must(dataHome()), "skilldozer", "skills")` (no `os.UserDataDir()` — compute it).
3. `init` flow: guard prompts with the existing `ModeCharDevice` stdin check; read answers with one shared `bufio.NewReader(os.Stdin).ReadString('\n')` ("empty/EOF ⇒ default"). Write the config via `Save`.
4. Discovery precedence: env var → config `store` → existing sibling rule → walk-up. Reuse `resolveSiblingFromExe` and `findWalkUpAncestor` unchanged.
5. Keep `gopkg.in/yaml.v3` the only non-stdlib dependency. Do **not** add `golang.org/x/term`.