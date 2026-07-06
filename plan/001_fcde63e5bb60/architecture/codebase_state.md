# Current Codebase State — `skpp`

## Status: GREENFIELD

The repository at `~/projects/skpp` contains ONLY:

```
skpp/
├── .git/                      # git repo, branch main, 1 commit
├── PRD.md                     # the complete spec (READ-ONLY for implementers)
└── plan/                      # this planning dir (untracked)
    └── 001_fcde63e5bb60/
        ├── architecture/      # these research docs
        └── tasks.json         # (to be generated)
```

There is NO source code, NO `go.mod`, NO `skills/`, NO README, NO LICENSE, NO
`.gitignore` yet. Everything in PRD §5 (Target repository layout) must be created
from scratch. This is a clean-room one-shot build.

## Verified environment

| Item | Value |
|---|---|
| Working dir | `/home/dustin/projects/skpp` |
| Git remote | `git@github.com:dabstractor/skpp.git` (origin) |
| Git branch | `main`, 1 commit (`Add PRD: manifest-free skill path printer`) |
| Go | 1.26.4, linux/amd64 |
| GOPATH | `/home/dustin/go` |
| pi | 0.80.3 at `/home/dustin/.local/bin/pi` |
| `~/.local/bin` | present and on PATH (where pi lives; install.sh default target) |

## Target layout (from PRD §5 — to be created)

```
skpp/
├── PRD.md                  # exists
├── README.md              # create — mirror mcpeepants style
├── LICENSE                # create — MIT
├── go.mod                 # create — module github.com/dabstractor/skpp
├── go.sum                 # create — yaml.v3
├── .gitignore             # create — /skpp, /dist, *.test, *.out, .DS_Store
├── main.go                # create — entrypoint: arg parsing, dispatch
├── internal/
│   ├── discover/discover.go    # scan skills dir, parse frontmatter, build index
│   ├── resolve/resolve.go      # tag → skill resolution rules (§7.2)
│   ├── skillsdir/skillsdir.go  # locate the skills/ dir (§8 priority order)
│   └── ui/ui.go                # --list / --search table formatting (ANSI)
├── install.sh             # create — build + symlink into PATH (§12.1)
├── completions/
│   ├── skpp.bash          # create
│   ├── _skpp              # create — zsh
│   └── skpp.fish          # create
└── skills/
    └── example/SKILL.md   # create — the ONE shipped example skill
```

## Risk areas (do first / test hardest) — per PRD §13, §18

1. **Skills dir location resolution (§8)** — symlink-aware `os.Executable()`. The
   acceptance test `test "$(./skpp --path)" = "$PWD/skills"` and the
   `/tmp/skpp-bin/skpp` symlink test both depend on this. PRD §18 says: do first.
2. **Error contract (§6.4)** — unknown/ambiguous tag ⇒ NOTHING on stdout, error
   to stderr, exit 1. This is the `$(...)` safety guarantee. Get exit codes and
   stdout/stderr discipline exactly right.
3. **Frontmatter parsing** — yaml.v3 on the `---`-delimited block; lenient on
   unknown keys; handle missing frontmatter gracefully (resolve by dir only).

## Module / build facts

- `go.mod` module path: `github.com/dabstractor/skpp`
- Go directive: latest two stable releases (go1.25 / go1.26). Since toolchain is
  go1.26.4, set `go 1.25` or `go 1.26`.
- Build command (install.sh): `go build -trimpath -ldflags "-s -w -X main.version=..." -o skpp .`
- Single third-party dep: `gopkg.in/yaml.v3`.
