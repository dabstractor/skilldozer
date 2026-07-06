# System Context — `skpp` (skill path printer)

## What `skpp` is

A tiny Go CLI that resolves a human-friendly **skill tag** to the **absolute filesystem
path** of a locally-stored [Agent Skill](https://agentskills.io/specification), so pi
can load it on demand:

```bash
pi --skill "$(skpp my-skill-tag)" --skill "$(skpp my-other-skill-tag)"
```

`skpp` is to **skills** what `mcpeepants` (`get-server-config.sh`) is to **MCP server
configs**: a centralized, on-disk catalog you address by tag, surfaced through a one-liner.

## The single most important architectural invariant

```
pi ──$(skpp <tag>)──▶ absolute path ──▶ pi --skill <path>
```

`skpp` is a **pure path resolver**. It NEVER:
- installs or copies skills anywhere,
- writes to `~/.pi/...` or `~/.agents/...`,
- auto-discovers or registers itself with pi,
- prints anything to stdout on failure (the `$(...)` contract — §6.4).

It only reads the disk and prints one absolute path per resolved tag.

## System boundary

```
┌─────────────────────────────────────────────────────────┐
│  user shell                                             │
│    pi --skill "$(skpp tag)"                             │
│         │                                               │
│         └─ skpp binary (Go, static, <5ms start)         │
│              │                                          │
│              ├─ locate skills/ dir  (§8 priority)       │
│              ├─ walk dir, parse SKILL.md frontmatter    │
│              ├─ resolve tag (canonical/basename/name/   │
│              │            alias precedence, §7.2)       │
│              └─ print absolute dir path to stdout       │
│                                                         │
│  pi ── loads ──▶ <store>/skills/<tag>/   (SKILL.md +    │
│                  optional scripts/references/assets)    │
└─────────────────────────────────────────────────────────┘
```

The store (`<store>/skills/`) is DELIBERATELY not one of pi's auto-discovery
locations (verified list below). It loads only through the explicit `--skill`
flag, even when pi's discovery is on.

## How pi discovers vs. how skpp feeds pi (verified)

Pi's skill auto-discovery locations (verified from pi 0.80.3 `docs/skills.md`):

- Global: `~/.pi/agent/skills/`, `~/.agents/skills/`
- Project (after trust): `.pi/skills/`, `.agents/skills/` (cwd + ancestors to git root)
- Packages: `skills/` dirs or `pi.skills` entries in `package.json`
- Settings: `skills` array in settings.json
- CLI: `--skill <path>` (repeatable, additive, works even with `--no-skills`)

**`skpp`'s store is NONE of the first five.** It is a standalone `skills/` dir
adjacent to the skpp binary (or pointed at by `SKPP_SKILLS_DIR`). It reaches pi
exclusively via the `--skill` path. The acceptance test proves this by loading
the example skill under `pi --no-skills --skill "$(skpp example)"`.

## Verified environment facts

| Fact | Value | Source |
|---|---|---|
| Go version available | 1.26.4 (linux/amd64) | `go version` |
| pi version | 0.80.3 at `/home/dustin/.local/bin/pi` | `pi --version` |
| GOPATH | `/home/dustin/go` | `go env GOPATH` |
| Repo | `~/projects/skpp`, remote `git@github.com:dabstractor/skpp.git` | `git remote -v` |
| Current contents | only `PRD.md` + `.git` + `plan/` (greenfield) | `ls` |

## Key relationships to other tools

- **pi**: the consumer. skpp emits paths pi's `--skill` flag accepts (dir or
  SKILL.md file). No pi API coupling — purely a string contract on stdout.
- **mcpeepants** (`~/projects/mcpeepants`): the sibling tool whose style/conventions
  skpp mirrors (README tone, install.sh spirit, bash/zsh/fish completions). See
  `mcpeepants_patterns.md`. NOTE: mcpeepants is bash + `servers.json` manifest;
  skpp is Go + manifest-free. The convergence is UX shape, not implementation.
