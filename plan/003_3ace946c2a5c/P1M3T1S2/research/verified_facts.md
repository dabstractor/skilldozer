# Verified Facts — P1.M3.T1.S2 (README `completion` subcommand doc, Mode B)

All anchors cross-checked against the live files at PRP-write time. This is a
**README.md-only documentation edit** (Mode B changeset-level doc sync).

## §1 — The README "Shell completions" section today (exact, lines 94–128)

```
94: ## Shell completions
95:
96: `skilldozer` ships dynamic completions for bash, zsh, and fish. Tag completion is
97: not a static list: the shell calls `skilldozer --relative --all` at completion time,
98: so it never goes stale as you add skills.
99:
100:**bash** (one of):
101:
102:```bash
103:source /path/to/skilldozer/completions/skilldozer.bash
104:cp completions/skilldozer.bash ~/.local/share/bash-completion/completions/skilldozer
105:cp completions/skilldozer.bash /etc/bash_completion.d/skilldozer
106:```
107:
108:**zsh** (one of):
109:
110:```bash
111:cp completions/_skilldozer ~/.zsh/completions/_skilldozer
112:cp completions/_skilldozer /usr/local/share/zsh/site-functions/_skilldozer
113:```
114:
115:then ensure this is in your `.zshrc`:
116:
117:```bash
118:autoload -U compinit && compinit
119:```
120:
121:**fish**:
122:
123:```bash
124:cp completions/skilldozer.fish ~/.config/fish/completions/skilldozer.fish
125:```
126:
127:`install.sh` does not install completions automatically; copy the file you
128:want as shown above.
```

- **Section spans `## Shell completions` (line 94) through the install.sh note (line 128).**
- The next section, `## Usage`, begins at line 130.
- **No occurrence of the `completion` SUBCOMMAND** exists in the README today
  (`grep -q 'skilldozer completion' README.md` → NOT PRESENT). The current section
  documents ONLY the §14.5 manual source/copy path. **This task adds the §14.6
  eval/source idiom.**

## §2 — The `completion` subcommand behavior (verified in main.go)

- **USAGE row (main.go:107):** `completion [--shell <name>]   Emit the shell completion script for eval (§14.6)`.
- **USAGE EXAMPLES (main.go:97):** `eval "$(skilldozer completion)"     # load completions into your shell`.
- **USAGE SYNOPSIS (main.go:82):** `skilldozer completion [--shell <name>]`.
- **`--shell` flag (main.go:108, 228-233, 314-321):** `--shell <bash|zsh|fish>` — force a shell for completion; implies completion mode.
- **Shell detection — PRD §14.6, first wins** (detectShell, main.go:1239-1252):
  1. `--shell <name>` — explicit; required for deterministic eval.
  2. `$SKILLDOZER_SHELL` — env override.
  3. `basename("$SHELL")` — the login shell (lowercased).
  4. None ⇒ stderr message + exit `1`.
- **Exit codes (runCompletion, main.go:1275-1293):** 0 success (script on stdout); 1 if shell undetectable; 2 if resolved shell not bash/zsh/fish (e.g. tcsh, or `$SHELL=/bin/sh` → `sh`). On 1/2 **nothing** is written to stdout (the `$(...)` contract).
- **Embedding (main.go:46-61):** the three `completions/*` files are `//go:embed`'d into the binary (stdlib, **no new dependency**). `completion` therefore works for `go install` users with **no repo clone** — consistent with §12.2 / decision 9 ("binary is self-sufficient"). The on-disk files remain the single source of truth; `completionScript()` (main.go:1121) returns the embedded bytes verbatim.

## §3 — PRD §14.6 canonical one-liners (the exact forms to document)

From PRD §14.6 (h3.19), verbatim:

```bash
# bash / zsh (~/.bashrc / ~/.zshrc)
eval "$(skilldozer completion)"

# fish (~/.config/fish/config.fish)
skilldozer completion --shell fish | source
```

> **Why emit + eval, not "install":** a child process cannot register completions in the
> parent shell that invoked it — only the shell itself can, by eval'ing/sourcing the script
> in its own process. So `completion` emits the script to stdout (for the parent to eval);
> it writes no files and edits no rc files. Same idiom as `zoxide init`, `starship init`,
> `direnv hook`. **(This rationale is NOT to be duplicated verbatim in the README — match
> the README's existing concise tone; at most a short clause.)**

## §4 — PRD §14.5 (the existing manual path — KEEP, do not remove)

PRD §14.5 (h3.18) IS the manual source/copy path already in the README (lines 100-128). The
contract explicitly says **keep the existing manual source/copy instructions** — they remain
valid and pick up edits to `completions/*` immediately (no rebuild needed), whereas the
eval/embed path requires a rebuild after editing `completions/*` (PRD §14.6 lockstep note).

## §5 — Contract requirements (the OUTPUT the implementer must produce)

1. Add the §14.6 `eval`/`source` idiom as the **RECOMMENDED** one-liner, placed **BEFORE** the
   manual source/copy instructions.
2. The one-liners (fixed, from §3 above):
   - bash/zsh: `eval "$(skilldozer completion)"` — add to `~/.bashrc` or `~/.zshrc`.
   - fish: `skilldozer completion --shell fish | source` — add to `~/.config/fish/config.fish`.
3. **One-line note:** the binary embeds the scripts (works for `go install` users with no clone
   — consistent with §12.2 decision 9).
4. Note `--shell` for deterministic eval, and the `$SKILLDOZER_SHELL` / `$SHELL` fallback for
   auto-detection.
5. **Keep the existing manual source/copy instructions** (they remain valid; pick up edits
   without a rebuild).
6. **Do NOT duplicate the §14.6 rationale verbatim** — match the README's existing concise tone.
7. **Verified by:** `grep -q 'skilldozer completion' README.md`.

## §6 — Scope boundaries (do NOT)

- **ONLY README.md is modified** — the "Shell completions" section (lines 94-128). Nothing else.
- **Do NOT touch** main.go, `completions/*`, PRD.md, `tasks.json`, `prd_snapshot.md`, `.gitignore`.
- This is **Mode B** (changeset-level doc sync) — the FINAL deliverable of delta 003. It depends
  on the feature (P1.M2.T2.S2) and the lockstep (P1.M3.T1.S1).
- The sibling P1.M3.T1.S1 edits ONLY `completions/*`. This task edits ONLY README.md → **DISJOINT**,
  no merge conflict.

## §7 — Validation (README docs task)

- `grep -q 'skilldozer completion' README.md` (the contract gate) → exits 0.
- `grep -q 'eval "$(skilldozer completion)"' README.md` → the bash/zsh one-liner present.
- `grep -q 'skilldozer completion --shell fish | source' README.md` → the fish one-liner present.
- The manual source/copy instructions are STILL present (regression: the four `cp`/`source`
  lines + the `autoload -U compinit && compinit` note + the install.sh note remain).
- Markdown sanity: the code fences are balanced; no broken section headers.
- `go test ./...` is unaffected (no code change) — run only to prove no accidental edits.

## §8 — Tone reference (the README's house style)

The README is concise, imperative, second-person, with short prose paragraphs and fenced
`bash` blocks. Examples of the existing tone in the "Shell completions" section: "ships
dynamic completions for bash, zsh, and fish", "copy the file you want as shown above".
Match this — do NOT paste the PRD's blockquote rationale. A short clause (e.g. "the binary
embeds the completion scripts, so this needs no clone") is the right register.
