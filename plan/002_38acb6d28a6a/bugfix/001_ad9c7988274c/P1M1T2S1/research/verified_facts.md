# P1.M1.T2.S1 — Verified Facts & Scope Notes (Issue 2, parse-level)

Source of truth for the PRP. Every claim verified against the live repo on
2026-07-07. Line numbers are from the CURRENT `main.go` (the contract's
citations match within ±1).

## 0. THE LOAD-BEARING SCOPE FACT — the plan SPLIT Issue 2 (overrides the arch doc)

`architecture/bug_fixes_validation.md` §ISSUE 2 "Fix ordering & dependency
notes" says: *"Issue 2 touches BOTH parseArgs (new config field) and run()
(guard) — keep the parse-level and run-level changes in the SAME subtask
(atomic)."*

**The plan OVERRODE this.** The task tree splits Issue 2 into TWO subtasks:
- **P1.M1.T2.S1 (THIS ONE):** record the `storeMissingValue` signal in parseArgs
  for BOTH `--store` no-value branches. **Parse-level only.**
- **P1.M1.T2.S2 (next):** guard `run()` to reject missing `--store` value with
  exit 2 before init dispatch (config NOT written). **run()-level only.**

→ T2.S1 MUST NOT add the run() guard, MUST NOT print any "--store requires a
value" message, MUST NOT touch exit codes. It only (a) adds the field,
(b) sets it in the two no-value branches, (c) adds parse-level tests. S2
consumes the field. The contract OUTPUT §4 is explicit: "consumed by
P1.M1.T2.S2 (run() guard)."

`architecture/decisions.md §D2` confirms the design: a dedicated
`storeMissingValue` field (NOT overloading `unknownFlag`), checked in run()
after the unknown-flag guard, before exclusivity/init dispatch.

## 1. The config struct — exact insertion site (main.go:128-152)

```go
128: type config struct {
...
149:	init        bool     // `skilldozer init [<dir>]` first-run setup ...; also set by `--store <dir>`
150:	initStore   string   // non-interactive store path: `init <dir>` positional or `--store <dir>` / `--store=<dir>`; empty ⇒ auto-detect
151:	tags        []string // positional <tag> args ...
152:	unknownFlag string   // first unknown dashed token, "" if none (§6 header → exit 2)
153: }
```
Insert `storeMissingValue bool` immediately AFTER `initStore` (line 150) so the
three --store-related fields (init, initStore, storeMissingValue) sit together.
Doc comment (Mode A): note it signals "--store/--store= seen with no value" and
that run() rejects with exit 2 (P1.M1.T2.S2); NOT set by bare `init`.

**gofmt gotcha:** `storeMissingValue` is 18 chars; the current longest field
name is `unknownFlag` (11). Adding any field longer than 11 re-aligns the TYPE
column of the ENTIRE struct (every field's type shifts right). This is normal
gofmt behavior — run `gofmt -w main.go` and expect the whole struct to
re-flow. Do NOT hand-align; let gofmt do it.

## 2. The '='-form `--store=` branch — exact code (main.go:192-197)

Inside the `--flag=value` splitter (the `strings.HasPrefix(a,"--") &&
strings.Contains(a,"=")` block):
```go
192:			case "--store":
193:				// `--store=<dir>`: non-interactive store path for init (PRD §8.2). Mirrors
194:				// --search's '='-form; implies init mode (c.init=true). No short form.
195:				c.init = true
196:				c.initStore = val
```
EXISING behavior: sets `c.init=true; c.initStore=val` UNCONDITIONALLY — including
when `val == ""` (i.e. `--store=` with nothing after `=`). The contract confirms
this: "sets c.init=true; c.initStore='' for an empty value."

CHANGE: after the two existing statements, add:
```go
				if val == "" {
					c.storeMissingValue = true
				}
```
KEEP `c.init=true` and `c.initStore=val` (contract: "you may still set c.init
true there, but run() will reject before dispatch"). Only ADD the signal.

## 3. The next-token `--store <dir>` branch — exact code (main.go:257-267)

```go
257:	case "--store":
258:		// `--store <dir>`: non-interactive store path for init (PRD §8.2). Mirrors
259:		// --search's next-token capture; implies init mode (c.init=true). No
260:		// short form. If it is the LAST token (no value follows) init stays
261:		// unset — mirrors --search-no-value (no exit-2 "needs argument" here;
262:		// the codebase defers that repo-wide).
263:		if i+1 < len(args) {
264:			c.init = true
265:			c.initStore = args[i+1]
266:			i++
267:		}
```
EXISTING behavior: the `c.init=true` is INSIDE the `if i+1 < len(args)` guard —
so when `--store` is the LAST token, NOTHING is set (silent no-op). The comment
(258-262) actively documents this no-op as "deferred/intentional" — that comment
is now WRONG (we no longer defer) and MUST be updated (Mode A honesty).

CHANGE: add an `else` arm setting the signal:
```go
		if i+1 < len(args) {
			c.init = true
			c.initStore = args[i+1]
			i++
		} else {
			c.storeMissingValue = true // --store with no following value (Issue 2)
		}
```
Do NOT set `c.init` in the else arm (the contract does not ask for it; c.init
stays false unless an `init` token also appeared — see §4).

## 4. The c.init ASYMMETRY between the two branches (PRESERVE — do not "fix")

| Input | c.init after S1 | c.initStore | storeMissingValue |
|---|---|---|---|
| `init --store` (next-token, last) | **true** (init token) | "" | **true** |
| `--store` (bare, last, no init token) | **false** (no init token, guard false) | "" | **true** |
| `--store=` (equals, empty, no init token) | **true** (equals-form sets it unconditionally) | "" | **true** |

The next-token bare-`--store` case has c.init=false while the equals-form
`--store=` has c.init=true. This is EXISTING behavior (the equals-form always
sets c.init; the next-token form only sets it when a value follows). **S1
PRESERVES it** — S1 only adds the signal. The asymmetry is HARMLESS because S2's
run() guard checks `storeMissingValue` BEFORE `if c.init { return runInit(...) }`,
so both cases exit 2 regardless of c.init. Do not "normalize" c.init in S1; the
contract does not ask for it.

## 5. The run() dispatch region — S2's insertion point (NOT S1's; boundary pin)

`main.go run()` order today (lines 420-449):
1. `:421` `if c.version { ... return 0 }`
2. `:428` `if c.unknownFlag != "" { fmt.Fprintf(stderr, "skilldozer: unknown flag '%s'\n", ...); return 2 }`
3. `:434` `if bad, msg := exclusivityError(c); bad { fmt.Fprintln(stderr, msg); return 2 }`
4. `:444-449` `if c.init { return runInit(c, stdout, stderr) }`

**S2 inserts its `storeMissingValue` guard between step 2 and step 3** (after
unknown-flag, before exclusivity/init dispatch) — per decisions.md §D2 ("after
the unknown-flag guard, before the exclusivity/init dispatch"). S1 does NOT add
this. S1's deliverable is only the field + the two parse-branch setters; S2
reads `c.storeMissingValue` and emits `skilldozer: --store requires a value` +
exit 2. Knowing where S2 lands is what lets S1 confidently set the field and
stop.

## 6. What S1 MUST NOT do (scope discipline)

- ❌ Add the run() guard or any "--store requires a value" message (S2).
- ❌ Change exit codes anywhere (S2).
- ❌ Overload `c.unknownFlag` (decisions.md §D2: different error class; the
  message must be "--store requires a value", not "unknown flag").
- ❌ Set `storeMissingValue` for bare `init` (no --store) — `c.initStore==""`
  legitimately means "prompt" there (contract CRITICAL note).
- ❌ Touch `--search`/`-s` no-value handling (architecture doc: harmless, falls
  to exit 1; out of scope for Issue 2).
- ❌ Touch the `init <dir>` positional capture (main.go:277-283) or bare `init`.
- ❌ Touch runInit / the check-report stream (that's P1.M1.T1.S1, Issue 1,
  running in parallel on DISJOINT lines ~978-1053).

## 7. The test harness + existing --store tests (the pattern to mirror)

`main_test.go` is `package main` (internal test) — `parseArgs` returns `config`
BY VALUE, and tests read fields DIRECTLY (`c.init`, `c.initStore`, etc.) with
`t.Errorf("...: got %v; want ...")`. No testify, no fixtures. Existing --store
parse tests:
- `TestParseArgsInitSubcommand` :1158 — bare `["init"]` → init=true, initStore="", tags empty.
- `TestParseArgsInitStoreLongForm` :1186 — `["init","--store","/tmp/x"]` → init=true, initStore="/tmp/x".
- `TestParseArgsInitStoreEqualsForm` :1200 — `["init","--store=/tmp/x"]` → init=true, initStore="/tmp/x".
- `TestParseArgsStoreWithoutInitToken` :1212 — `["--store","/tmp/x"]` → init=true, initStore="/tmp/x".

The `--search` no-value test (`TestParseArgsSearchNoValueStaysInactive` :901) is
the CLOSEST analog: `["--search"]` → searchMode stays false. The --store signal
test mirrors its shape but asserts the new field.

## 8. Test coverage for the "iff" (storeMissingValue true ⟺ --store with no value)

The contract OUTPUT §4: "config.storeMissingValue is true iff --store/--store=
appeared without a value." The iff needs BOTH positive and negative cases:

POSITIVE (new tests, add after :1224):
- `["init","--store"]` → storeMissingValue=true (init=true, initStore="")
- `["--store="]` → storeMissingValue=true (init=true, initStore="")
- `["--store"]` (bare) → storeMissingValue=true (init=false, initStore="")

NEGATIVE (strengthen existing tests with `if c.storeMissingValue { t.Errorf(...) }`):
- `["init"]` (bare) → storeMissingValue=false  ← the CRITICAL one (must still prompt)
- `["--store","/tmp/x"]` and `["init","--store","/tmp/x"]` and `["init","--store=/tmp/x"]`
  → storeMissingValue=false (value present)

## 9. Parallel sibling P1.M1.T1.S1 (Issue 1) — DISJOINT, no conflict

T1.S1 edits `runInit` (main.go ~978-1053, the three check-report `fmt.Fprintf`
stdout→stderr swaps + comments). T2.S1 edits the config struct (128-152) and
parseArgs (192-197, 257-267). The two edits touch DISJOINT regions of main.go +
disjoint tests (T1.S1: TestRunInitStoreWritesConfig… @2325; T2.S1: TestParseArgs*
@1158-1224). They can land in either order with no merge conflict. Both keep
go.mod/go.sum byte-for-byte unchanged (no new deps).

## 10. Baseline (verified green before any change)

- `go build ./...` → ok
- `go test ./...` → passes (the bug is behavioral — silent config overwrite —
  not a compile error; today's tests pass because none exercise the no-value
  --store branches, which is exactly the coverage gap S1 closes).
