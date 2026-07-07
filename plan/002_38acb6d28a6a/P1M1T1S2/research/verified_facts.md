# P1.M1.T1.S2 — Verified facts (empirical)

Empirically verified on the target host (Linux, go 1.25,
`github.com/dabstractor/skilldozer`) by running small `go run` probes. Every
claim below was observed directly; nothing is from memory.

---

## 1. `os.UserConfigDir()` — the XDG_CONFIG_HOME rule (and its error paths)

Source: https://pkg.go.dev/os#UserConfigDir. Observed behavior on Linux:

| Env state | `os.UserConfigDir()` result |
|---|---|
| `XDG_CONFIG_HOME="/custom/xdg"` (absolute) | `("/custom/xdg", nil)` — returned **verbatim, NOT cleaned** |
| `XDG_CONFIG_HOME="/custom/x/../y"` (absolute, unclean) | `("/custom/x/../y", nil)` — still verbatim |
| `XDG_CONFIG_HOME=""`, `HOME="/home/u"` | `("/home/u/.config", nil)` |
| `XDG_CONFIG_HOME=""`, `HOME=""` | `("", error: "neither $XDG_CONFIG_HOME nor $HOME are defined")` |
| `XDG_CONFIG_HOME="relative/config"` (RELATIVE) | `("", error: "path in $XDG_CONFIG_HOME is relative")` |

**LOAD-BEARING CONCLUSION (corrects a common misconception):** On Linux,
`os.UserConfigDir()` DOES reject a relative `$XDG_CONFIG_HOME` with a non-nil
error. The external_deps.md §2 claim ("If `$XDG_CONFIG_HOME` is set but not
absolute, it returns an error") and the contract's RESEARCH NOTE are both
CORRECT. There is NO need for `config.Path()` to re-validate absoluteness — it
delegates to `os.UserConfigDir()` and returns `("", err)` on any error.

> NOTE: the error *message wording* (`"path in $XDG_CONFIG_HOME is relative"`,
> `"neither $XDG_CONFIG_HOME nor $HOME are defined"`) is stdlib-internal and may
> shift across Go versions. Tests assert ONLY `err != nil` + returned `""`, never
> the message text.

---

## 2. `os.UserHomeDir()` — the HOME rule

Source: https://pkg.go.dev/os#UserHomeDir. Observed:

| Env state | `os.UserHomeDir()` result |
|---|---|
| `HOME="/home/u"` | `("/home/u", nil)` |
| `HOME=""` | `("", error: "$HOME is not defined")` |

There is **no `os.UserDataDir()`** (confirmed: not in the `os` package). So the
`$XDG_DATA_HOME -> ~/.local/share` rule must be computed by hand in
`config.DefaultStore()`, exactly as the contract specifies.

---

## 3. `filepath.Clean()` on the `SKILLDOZER_CONFIG` override

`Path()` returns `filepath.Clean(v)` of the override **AS-IS** (no `filepath.Join`
with the config home). Observed `filepath.Clean` outputs:

| input | `filepath.Clean` |
|---|---|
| `"/tmp/foo/config.yaml"` | `"/tmp/foo/config.yaml"` |
| `"relative/config.yaml"` | `"relative/config.yaml"` |
| `"./x.yaml"` | `"x.yaml"` |
| `"x.yaml"` | `"x.yaml"` |
| `"/a/b/../c.yaml"` | `"/a/c.yaml"` (lexical `..` collapse, no symlink eval) |
| `"/tmp/foo/"` | `"/tmp/foo"` (trailing slash stripped) |
| `""` | `"."` — **GOTCHA** — but the contract's `v != ""` guard means Clean is
        never called on an empty value. Path() never returns `"."` in practice. |

**LOAD-BEARING CONCLUSION:** the override is cleaned but NOT joined and NOT
symlink-evaluated. The user's literal path (abs or relative) wins verbatim modulo
lexical cleaning. This is the "useful for tests / multiple profiles" knob from
PRD §8.1.

---

## 4. `filepath.IsAbs()` — the `XDG_DATA_HOME` guard for `DefaultStore()`

`DefaultStore()` honors `XDG_DATA_HOME` only when it is absolute (XDG spec: a
relative `XDG_*_HOME` is invalid and must be ignored). Observed `filepath.IsAbs`:

| input | `filepath.IsAbs` |
|---|---|
| `"/abs/path"` | `true` |
| `"/"` | `true` |
| `"relative"` | `false` |
| `"./also-rel"` | `false` |
| `""` | `false` |

So the guard `v != "" && filepath.IsAbs(v)` correctly:
- uses an absolute `XDG_DATA_HOME`,
- ignores a relative `XDG_DATA_HOME` (XDG-spec correctness — fall back to
  `~/.local/share`),
- ignores an empty/unset `XDG_DATA_HOME` (fall back to `~/.local/share`).

---

## 5. `filepath.Join` always returns a Clean path

`filepath.Join("/custom/x/../y", "skilldozer", "config.yaml")` →
`"/custom/y/skilldozer/config.yaml"` (the `..` is collapsed by Join). So the
DEFAULT path returned by `Path()` (the `filepath.Join(configHome, "skilldozer",
"config.yaml")` branch) is always clean even if `os.UserConfigDir()` returned an
unclean absolute value. No extra `filepath.Clean` needed on the default branch.

---

## 6. Consumers (NOT built here — listed to fix the interface)

From the contract OUTPUT §4 + plan/task tree:

| Consumer (later subtask) | Calls | Why `Path`/`DefaultStore` must not mask errors |
|---|---|---|
| `skillsdir.findConfig` (P1.M1.T2.S2) | `path, err := config.Path()` then `config.Load(path)` | if `Path` errors (e.g. relative `XDG_CONFIG_HOME`), `findConfig` treats it as "config unavailable" and falls through to rule 3 (sibling). Must propagate `("", err)`. |
| `init` choose-store (P1.M2.T2.S1) | `def, err := config.DefaultStore()` | init needs the default store dir for its prompt / `--store` fallback. On error (HOME unset), init should fail with a clear message, not a panic. |
| `init` write-config (P1.M2.T2.S2) | `config.Save(config.Path(), config.File{Store: …})` | writes config.yaml at the resolved path (reusing S1's `Save`, which MkdirAll's the parent). |

`Path`/`DefaultStore` never read the config FILE (that's `Load`, S1) and never
touch the filesystem beyond `os.Getenv` / `os.UserConfigDir` / `os.UserHomeDir`.
They are pure env->path resolvers.

---

## 7. Exported surface delta vs. S1

S1's `internal/config` exports exactly `{File, Load, Save}`. S2 ADDS:

- `func Path() (string, error)`
- `func DefaultStore() (string, error)`
- the `SKILLDOZER_CONFIG` env-var constant.

**Constant name decision:** the contract LOGIC §3 writes it lowercase
(`configEnv = "SKILLDOZER_CONFIG"`) and says "exported or package-internal as
needed by main". No consumer needs the SYMBOL (findConfig/init call the
functions, which encapsulate the env read; `--path` prints the label `config
file`, NOT `SKILLDOZER_CONFIG`). So **unexported `configEnv`** matches
`internal/skillsdir/skillsdir.go`'s precedent (`const envVar =
"SKILLDOZER_SKILLS_DIR"`). If a future consumer needs the symbol, capitalize the
`C` — a one-line change. (See PRP "DESIGN DECISIONS" §1.)

---

## 8. No new dependency; no new file required

- `os.Getenv`, `os.UserConfigDir`, `os.UserHomeDir`, `filepath.Clean`,
  `filepath.IsAbs`, `filepath.Join` are all **stdlib**.
- S2 EXTENDS the two files S1 creates (`internal/config/config.go`,
  `internal/config/config_test.go`). It does not create a new file — the package
  is small and cohesive (every symbol is about locating config/store paths),
  matching `internal/discover/discover.go` and `internal/skillsdir/skillsdir.go`
  (each a single source file). S1 is the dependency root (status "Implementing")
  and is implemented BEFORE S2, so `config.go` already exists when S2 begins — no
  write conflict.
- `go.mod`/`go.sum` remain byte-for-byte unchanged (GOTCHA from S1 carries over).
