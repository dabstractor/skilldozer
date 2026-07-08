# Verified Facts — P1.M2.T4.S1: Rewrite `.gitignore` to the §16 canonical 5-entry set (Issue 6)

All facts below were confirmed by direct command execution in the repo at PRP-write time
(cwd `/home/dustin/projects/skilldozer`). Each line number / byte claim is reproducible.

## 1. The §16 canonical block — EXACT bytes (the acceptance oracle)

`awk 'NR>=423 && NR<=431{printf "%d|%s|\n",NR,$0}' PRD.md`:

```
423|## 16. `.gitignore`|
424||
425|```|
426|/skilldozer|
427|/dist|
428|*.test|
429|*.out|
430|.DS_Store|
431|```|
```

So the 5 spec lines are PRD.md **lines 426–430** (the code-fence interior; 425/431 are the
fence backticks). `sed -n '426,430p' PRD.md | cat -A`:

```
/skilldozer$
/dist$
*.test$
*.out$
.DS_Store$
```

→ exactly 5 lines, each terminated by `\n`, NO trailing whitespace, NO BOM, NO trailing blank
line. The `cat -A` `$` is the line-end marker. **This is the byte-exact target for `.gitignore`.**

The acceptance gate (contract LOGIC §3 / bug_fixes_validation.md §ISSUE 6 Verification) is:

```
diff <(sed -n '426,430p' PRD.md) .gitignore
```

and it MUST produce no output. For `diff` to be empty, `.gitignore` must be byte-for-byte
identical to `sed -n '426,430p' PRD.md`, i.e. exactly:

```
/skilldozer\n/dist\n*.test\n*.out\n.DS_Store\n
```

(one trailing `\n` after `.DS_Store`, and NOTHING after it — no 6th blank line, no comment).

## 2. The current `.gitignore` — what is being removed (byte-level)

`cat -A .gitignore` (19 lines):

```
# Build artifacts$              <- COMMENT (remove)
/skilldozer$                    <- KEEP (§16)
/dist$                          <- KEEP (§16)
/build$                         <- EXTRA (remove)
*.test$                         <- KEEP (§16)
*.out$                          <- KEEP (§16)
$                               <- blank separator (remove)
# Dependency directories$       <- COMMENT (remove)
node_modules/$                  <- EXTRA (remove)
venv/$                          <- EXTRA (remove)
$                               <- blank (remove)
# Environment files$            <- COMMENT (remove)
.env$                           <- EXTRA (remove)
$                               <- blank (remove)
# OS-specific files$            <- COMMENT (remove)
.DS_Store$                      <- KEEP (§16)
$                               <- blank (remove)
# Agent runtime artifacts ...$  <- COMMENT (remove)
.pi-subagents/$                 <- EXTRA (remove — residual risk, see §4)
```

File ends with `.pi-subagents/\n` (verified via `tail -c 20 .gitignore | xxd` → `... .pi-subagents/\n`).

- **5 KEEP** (map 1:1 to PRD.md:426-430): `/skilldozer`, `/dist`, `*.test`, `*.out`, `.DS_Store`.
- **5 EXTRA entries** (remove): `/build`, `node_modules/`, `venv/`, `.env`, `.pi-subagents/`.
- **5 COMMENT lines** (remove): the `# …` lines above. §D5: the canonical block has NO comments.
- **4 blank separator lines** (remove): the empty lines between sections.

This matches the contract's "5 extra entries + 5 section-comment lines" exactly.

## 3. No code, no test, no README touches `.gitignore`

- `grep -rn 'gitignore' main.go main_test.go internal/ install.sh` → no matches. **`.gitignore` is
  not read by any Go code or install script.** bug_fixes_validation.md §ISSUE 6: "No Go test
  (.gitignore is not code)." → the **file content itself IS the acceptance check**; there is no Go
  test to add or update, and `go test ./...` is unaffected by this change.
- `grep -in 'gitignore' README.md` → NO match. (README:291 mentions `node_modules` once, but as
  incidental prose about pi skill packages — "a `node_modules` package" — NOT a `.gitignore`
  enumeration.) → **no README change** (contract DOCS: Mode A; the `.gitignore` IS the doc surface).

## 4. Residual risk: `.pi-subagents/` becomes untracked (intended per §D5 / §D3)

`git status --ignored --short` today shows `!! .pi-subagents/` and `!! skilldozer` as the only
ignored paths (the `/build`, `node_modules/`, `venv/`, `.env` entries have no on-disk files, so
removing them is cosmetic — nothing surfaces). After the rewrite:

- `.pi-subagents/` is NO LONGER ignored → it becomes **untracked** → it will surface in `git status`
  as `??`. This is **intended** per architecture/decisions.md **§D5** and prior-round **§D3**
  ("do NOT bless extras … if maintainers want the extras, they update §16 themselves").
- `skilldozer` (the locally-built binary, committed-ignored via `/skilldozer`) STAYS ignored — it is
  line 1 of the §16 set. Good: the build artifact remains ignored.

So the user-visible effect is: `.pi-subagents/` shows up untracked. This is the noted residual risk,
NOT a blocker. The implementer must NOT re-add `.pi-subagents/` (or any extra) "to be tidy" — that
re-opens Issue 6 and violates §D5.

## 5. Disjointness from the parallel sibling (P1.M2.T3.S2)

P1.M2.T3.S2 edits ONLY `main.go` (resolveStore wiring + doc) + `main_test.go` (one integration test).
Its scope-discipline checklist explicitly says "Did NOT modify … `.gitignore`". This subtask edits
ONLY `.gitignore`. → **zero file-level overlap**; the two land in either order with no merge
conflict. (Both are in `git status` as untracked/modified but touch disjoint paths.)

## 6. decisions.md §D5 (verbatim) — the decision record for this exact change

> ## D5 — Issue 6: trim to §16 exactly, including removing comments and .pi-subagents/
> Per prior-round §D3, do NOT bless extras. The `.pi-subagents/` dir becomes untracked
> (surfaces in git status) — that is intended; if the user wants it ignored they update §16
> themselves. The §16 canonical block has NO section comments; the rewrite omits them.

## 7. How to write the file byte-exactly (implementation method)

The single trailing-newline invariant is the one thing most likely to be gotten wrong. To guarantee
byte-for-byte equality with `sed -n '426,430p' PRD.md`, write the file with EXACTLY this content and a
single terminating `\n`:

```
/skilldozer
/dist
*.test
*.out
.DS_Store
```

and nothing else. Verify post-write with `diff <(sed -n '426,430p' PRD.md) .gitignore && echo OK`
(empty output + OK), `wc -l .gitignore` (== 5), and `tail -c 20 .gitignore | xxd` (ends in
`.DS_Store\n`, no double `\n`). A trailing 6th blank line or a missing final `\n` both break the diff.
