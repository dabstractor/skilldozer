# parseArgs / run Analysis — `main.go`

All line numbers are from `main.go` (1335 lines) as read in this run. Every claim below was verified empirically by running the binary.

## Files Retrieved
1. `main.go` (lines 140-175) — `config` struct: the fields that carry parse results (`searchMode`, `searchQ`, `storeMissingValue`, `completionShell`, `unknownFlag`, `tags`).
2. `main.go` (lines 182-358) — `parseArgs`: the `=`-form branch (a), the short-bundle dispatch (b), and the main `switch a` (incl. `--search`/`-s`, `--store`, `--shell`, `--init`, and the `default`).
3. `main.go` (lines 361-455) — `expandShortBundle` doc + Phase 1 (validate) + Phase 2 (commit) + the `-s` value-taking tail.
4. `main.go` (lines 466-504) — `run`: precedence ladder (`help → version → unknownFlag → storeMissingValue`).
5. `main.go` (lines 744-750) — the no-recognized-mode fall-through (usage to stdout, exit 0).

---

## (1) `--search` / `-s` no-value handling — three forms

### (1a) The `=`-form — main.go:218-220
Inside the `if strings.HasPrefix(a, "--") && strings.Contains(a, "=")` block (begins main.go:202), `name`/`val` are split on the first `=`:
```go
// main.go:218-220
case "--search":
    c.searchMode = true
    c.searchQ = val
```
There is **no empty-value guard here**: `--search=` sets `searchMode=true` with an empty query (search runs with `""`, not a missing-value error). This is deliberate — `--search=` is a valid (if useless) empty query, distinct from `--search` with no following token.

### (1b) Space-separated form in the main `switch a` — main.go:289-298
```go
// main.go:288-298
case "--search", "-s":
    // Value-taking flag: consume the NEXT token verbatim as the query. The
    // value is NOT appended to c.tags (i++ skips it), and it never reaches
    // the default branch, so a dashed value (e.g. `--search -x` → query
    // "-x") is NOT mistaken for an unknown flag. If --search is the LAST
    // token (no value follows) searchMode stays false and the call falls
    // through to the no-recognized-mode default (exit 1).
    if i+1 < len(args) {
        c.searchMode = true
        c.searchQ = args[i+1]
        i++
    }
```
**No-value behavior:** if `--search`/`-s` is the LAST token (`i+1 >= len(args)`), the `if` body is skipped — `searchMode` stays `false`, `searchQ` stays `""`. Nothing else is set; the loop simply moves on. With no other mode/tag set, `run()` reaches the no-recognized-mode fall-through (usage → exit 0).

### (1c) Short-bundle form in `expandShortBundle` — main.go:434-448
The `s`-handling tail of Phase 2:
```go
// main.go:433-448
    if sIdx >= 0 {
        remainder := body[sIdx+1:]
        switch {
        case remainder != "":
            c.searchMode = true
            c.searchQ = remainder // value embedded in the bundle ("-sfoo")
        case i+1 < len(args):
            c.searchMode = true
            c.searchQ = args[i+1] // value is the next argv token ("-ls foo")
            return true, true     // caller advances i
        default:
            // 's' seen but no value anywhere: mirror the bare "-s"-no-value rule
            // (searchMode stays false). The bool flags before it remain set.
        }
    }
```
**No-value behavior:** the `default:` arm (empty remainder AND no next argv token) is a no-op — `searchMode` stays `false`, the leading bool flags (`-l`, etc.) remain applied. `return false, true` (no consumeNext). This mirrors (1b) for e.g. `-ls` as the sole token.

**Note on `bare -s`:** a lone `-s` has `len==2`, so it does NOT enter `expandShortBundle` (which requires `len(a) > 2` at main.go:259). It is handled by the main `switch a` case at main.go:288 — i.e. form (1b).

---

## (2) `--shell` no-value handling — two forms

### (2a) The `=`-form — main.go:248-253
Inside the same `=`-split block as (1a):
```go
// main.go:248-253
case "--shell":
    // `--shell=<name>`: force a shell for completion (PRD §14.6). Mirrors --store's '='-form;
    // implies completion mode (c.completion=true). No short form. NO empty-value guard (PRD §6.4
    // specifies no missing-value exit code for --shell — `--shell=` is completion=true, shell="").
    c.completion = true
    c.completionShell = val
```
No empty-value guard: `--shell=` sets `completion=true`, `completionShell=""` (auto-detect at emit time).

### (2b) Space-separated form in the main `switch a` — main.go:320-326
```go
// main.go:320-326
case "--shell":
    // `--shell <name>`: force a shell for completion (PRD §14.6). Mirrors --store's next-token
    // capture; implies completion mode (c.completion=true) when a value follows. No short form.
    // If --shell is the LAST token (no value), completion stays false — mirrors --search's
    // no-value silent behavior (PRD §6.4 specifies no missing-value exit code for --shell).
    if i+1 < len(args) {
        c.completion = true
        c.completionShell = args[i+1]
        i++
    }
```
**No-value behavior:** if `--shell` is the LAST token, the `if` is skipped — `completion` stays `false`. With no other mode/tag, `run()` reaches usage → exit 0. **Asymmetry vs `--store`:** `--store` (last token) sets `storeMissingValue=true` → exit 2; `--shell` (last token) sets nothing → exit 0. This is intentional per the PRD §6.4 "no missing-value exit code for --shell" note.

---

## (3) How `storeMissingValue` is checked in `run()` — exact code path

The signal is **set** in `parseArgs` in exactly three places (grep-confirmed: lines 229, 237, 318):
- main.go:229 — `--store=` empty value (`=`-form branch).
- main.go:237 — `--init=` empty value (`=`-form branch).
- main.go:318 — `--store` with no following token (main `switch` branch).

It is **read** once in `run()`, at main.go:499-502, as step 3.5 of the precedence ladder (AFTER help/version/unknownFlag, BEFORE exclusivityError / init dispatch):
```go
// main.go:493-502
// 3.5) --store presented without its value → exit 2 (PRD §6 header "Unknown flags
//      ⇒ error + exit 2"; ... A value-taking flag with no value is a parse error ...
//      The signal is set by parseArgs in BOTH --store no-value branches (P1.M1.T2.S1);
//      it is NOT set by bare `init` (c.initStore=="" legitimately means "prompt").
if c.storeMissingValue {
    fmt.Fprintln(stderr, "skilldozer: --store requires a value")
    return 2
}
```
**Critical ordering fact:** this check runs BEFORE `runInit` / `setupStore` / `configpkg.Save` are ever reached, so a pre-existing `config.yaml` is preserved on the error path (the "Issue 2" non-destructive guarantee). Bare `--init`/`init` does NOT set the flag (c.initStore=="" legitimately means "prompt"), so it is unaffected.

Precedence ladder in `run()` (main.go:466): `help (471) → version (477) → unknownFlag (484) → storeMissingValue (499) → exclusivity (508) → init dispatch (520) → completion dispatch (528) → normal modes → no-args-usage (749)`.

---

## (4) The misleading comment — exit 1 vs exit 0

**The comment text lives at main.go:290-293**, inside the `case "--search", "-s":` block. The exact phrasing "falls through to the no-recognized-mode default (exit 1)" is split across two lines:
```go
// main.go:290-293
// ... If --search is the LAST
// token (no value follows) searchMode stays false and the call falls
// through to the no-recognized-mode default (exit 1).
```
- Line 292 ends: `...searchMode stays false and the call falls`
- Line 293: `// through to the no-recognized-mode default (exit 1).`

The PRD's reference to `main.go:293` points at the `(exit 1)` claim. **This claim is WRONG.** The actual fall-through target is main.go:744-750:
```go
// main.go:744-750
// No recognized mode → usage to STDOUT, exit 0 (PRD §6.3 / §19 decision 17:
// bare invocation is implicit --help)...
fmt.Fprint(stdout, usage())
return 0
```
Empirically confirmed: `skilldozer --search` (and `-s`) with no value prints usage to **stdout** and exits **0**, not exit 1. The `run()` exit-code docstring (main.go:455-457) also documents exit 0 for "no-args/modifiers-only printed usage to stdout." So the fix is to change the comment's `(exit 1)` to `(exit 0)`.

---

## (5) How `--` tokens are classified in the default branch of `parseArgs`

The main `switch a` `default:` branch is at main.go:343-356:
```go
// main.go:343-356
default:
    // Positional <tag> ... A dashed token NOT in the known set is an unknown flag
    // (PRD §6 header: exit 2): capture the FIRST offender for run() to report. ...
    if strings.HasPrefix(a, "-") {
        if c.unknownFlag == "" {
            c.unknownFlag = a
        }
    } else {
        c.tags = append(c.tags, a)
    }
```
**A bare `--` token reaches this `default:` branch** because it fails BOTH upstream guards:
- main.go:202 `strings.HasPrefix(a, "--") && strings.Contains(a, "=")` → `--` has no `=`, so **false** (skips the `=`-form block).
- main.go:259 `len(a) > 2 && a[0] == '-' && a[1] != '-'` → `len("--")==2`, not `> 2`, so **false** (skips short-bundle).
- main.go:277 `switch a` → `--` matches no case → `default:`.

In `default:`, `strings.HasPrefix("--", "-")` is **true**, so `--` is recorded as an **unknown flag** (`c.unknownFlag = "--"`). In `run()`, `c.unknownFlag != ""` at main.go:484 prints `skilldozer: unknown flag '--'` to stderr and **returns 2**.

Empirically confirmed: `skilldozer --` → `skilldozer: unknown flag '--'` on stderr, exit 2. There is **no end-of-options (`--`) semantics** anywhere in `parseArgs` — the conventional "treat everything after `--` as positional" behavior is NOT implemented; `--` itself is rejected as an unknown flag, and any token after `--` is classified by the same rules as before (a leading-dash token after `--` would still be flagged unknown).

---

## Empirical confirmation matrix (commands run)

| Input | stdout | stderr | exit | Source branch |
|---|---|---|---|---|
| `--search` (last) | usage | — | **0** | (1b) → main.go:749 |
| `-s` (last) | usage | — | **0** | (1b) → main.go:749 |
| `--shell` (last) | usage | — | **0** | (2b) → main.go:749 |
| `--store` (last) | — | `--store requires a value` | **2** | main.go:318 → 501 |
| `--` (bare) | — | `unknown flag '--'` | **2** | main.go:349-352 → 485 |

(`go run` reports the child's non-zero status as its own exit 1 + an `exit status N` line; the stderr text and "exit status 2" line confirm the program's true exit code.)

---

## Start Here
Open `main.go` and read **lines 288-298** (the `--search`/`-s` case) first — that is where the misleading `(exit 1)` comment at line 293 needs to become `(exit 0)`, and it is the canonical reference for the three no-value patterns. Then main.go:343-356 (the `default:` branch) for the `--` classification issue.
