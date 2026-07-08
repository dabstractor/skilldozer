# Verified facts — P1.M1.T2.S2 (Issue 2 run-level: guard `run()` to reject missing `--store` value, exit 2, config NOT written)

Every claim anchored to a file read in full or a command run on the live repo at
`/home/dustin/projects/skilldozer`. State: P1.M1.T2.S1 in-flight in parallel; the
`storeMissingValue` field + the `'='`-form setter have LANDED; the next-token `else`
arm is pending (this PRP assumes T2.S1 lands fully — see §5).

---

## 0. The one-line summary

T2.S2 adds ONE block to `run()`:
```go
if c.storeMissingValue {
    fmt.Fprintln(stderr, "skilldozer: --store requires a value")
    return 2
}
```
placed BETWEEN the `unknownFlag` check and the `exclusivityError` check (decisions.md
§D2: "after the unknown-flag guard, before the exclusivity/init dispatch"). Because it
returns before `if c.init { return runInit(...) }`, `setupStore`/`configpkg.Save` is
NEVER reached → the existing `config.yaml` is preserved (the non-destructive contract).
Plus: update the `run()` precedence comment (Mode A) + add run()-level tests.

## 1. The insertion point (current, post-T2.S1-field state — locate by branch, lines drift ~+5 after T2.S1's next-token arm lands)

`main.go` `run()` (func at `:414` today; shifts down a few lines once T2.S1's Task 3
next-token `else` arm lands — INSERT BY BRANCH, not line number):
- step 1) `--help` (`:421-425`)
- step 2) `--version` (`:428-430`)
- step 3) **`unknownFlag`** (`:434-437`): `if c.unknownFlag != "" { fmt.Fprintf(stderr, "skilldozer: unknown flag '%s'\n", c.unknownFlag); return 2 }`
- **⟵ INSERT THE GUARD HERE (new step 3.5), after the unknownFlag `}` + blank line, before the `// 4) Mode mutual exclusivity` comment**
- step 4) `exclusivityError` (`:440-446`): `if bad, msg := exclusivityError(c); bad { fmt.Fprintln(stderr, msg); return 2 }`
- init dispatch (`:455-456`): `if c.init { return runInit(c, stdout, stderr) }`
- normal-mode ladder (`:461+`)

The contract allows "before exclusivityError OR before init dispatch; either is correct
as long as it precedes runInit." **This PRP picks BEFORE exclusivity** (between
unknownFlag and exclusivity) as primary because: (a) decisions.md §D2's phrasing "after
the unknown-flag guard, before the exclusivity/init dispatch" most naturally reads as
"right after unknownFlag"; (b) it is a flag-PARSE error like unknownFlag, so grouping
parse-errors-before-exclusivity matches the existing "more fundamental errors first"
ordering (the comment at `:441-443` already says unknownFlag is checked before
exclusivity so `--bogus foo --list` reports the unknown flag first); (c) it fires
earlier, so `init --store --list` reports the missing value (more fundamental) before
the exclusivity conflict.

## 2. The exact guard + comment (the full insertion block)

```go
// 3.5) --store presented without its value → exit 2 (PRD §6 header "Unknown flags
//      ⇒ error + exit 2"; delta-PRD §2 #3 "init is non-destructive"). A value-taking
//      flag with no value is a parse error, NOT a silent fall-through to destructive
//      auto-detect init. Rejecting here — BEFORE the init dispatch — means runInit /
//      setupStore / configpkg.Save is NEVER called, so a pre-existing config.yaml's
//      `store:` value is preserved (Issue 2). stdout stays EMPTY (§6.4 discipline).
//      The signal is set by parseArgs in BOTH --store no-value branches (P1.M1.T2.S1);
//      it is NOT set by bare `init` (c.initStore=="" legitimately means "prompt").
if c.storeMissingValue {
    fmt.Fprintln(stderr, "skilldozer: --store requires a value")
    return 2
}
```

Message: EXACTLY `skilldozer: --store requires a value` (contract LOGIC §3 +
decisions.md §D2). Use `fmt.Fprintln` (adds `\n`; matches the exclusivityError branch's
`fmt.Fprintln(stderr, msg)` style for fixed strings — unknownFlag uses `Fprintf`+`\n`
only because it has a `%s` arg). The `skilldozer:` prefix matches unknownFlag's prefix.

## 3. The precedence comment update (DOCS, Mode A)

Current comment (`main.go:410-413`):
```
// Precedence (PRD §6.3 "--help / --version take precedence over everything else"
//   - the conventional help-wins tiebreak):
//     help → version → unknownFlag → exclusivity → dispatch → no-args-usage.
```
Update the ladder line to include the new guard:
```
//     help → version → unknownFlag → storeMissingValue (--store needs a value) → exclusivity → dispatch → no-args-usage.
```
(One line changed; the two intro lines stay.) Mode-A honesty: the comment must list
every step the dispatch actually performs, or it drifts.

## 4. T2.S1 status + the three shapes that set the signal (all three exit 2 after the guard)

T2.S1 has LANDED the field (`main.go:142`) + the `'='`-form setter (`:193-202`); the
next-token `else` arm (`case "--store"` at `:263+`) is PENDING but will land per the
T2.S1 PRP contract. Once T2.S1 is complete, THREE argv shapes set `storeMissingValue=true`:

| argv | c.init | c.initStore | c.storeMissingValue | run() after guard |
|---|---|---|---|---|
| `init --store` (next-token, last) | true (init token) | "" | true | **exit 2** (the destructive bug, now fixed) |
| `--store=` (equals, empty) | true ('='-form sets it) | "" | true | **exit 2** |
| `--store` (next-token, bare, no init) | false (no init token; no value) | "" | true | **exit 2** (was exit 1 usage — see §6) |
| `init` (bare, no --store) | true | "" | **false** | proceeds to runInit → PROMPTS (CRITICAL: must still prompt) |
| `init --store /x` / `--store=/x` | true | "/x" | false | proceeds to runInit → writes config |

The guard is `if c.storeMissingValue` — UNCONDITIONAL on c.init (per contract LOGIC §3).
So all three no-value shapes exit 2, including bare `--store`.

## 5. CRITICAL — bare `--store` (no init token, no value) changes from exit 1 → exit 2 (intentional improvement)

Today (pre-fix): `skilldozer --store` (bare) → no init token, next-token --store no-ops
(no else arm yet), c.init=false → falls through to no-args usage → exit 1, config
untouched. The bug writeup (§ISSUE 2) calls this "harmless."

After T2.S1+S2: the T2.S1 else arm sets storeMissingValue=true; this guard fires →
**exit 2 "skilldozer: --store requires a value"**. Both non-zero, both config-untouched.
Exit 2 with an explicit "requires a value" message is the CORRECT post-fix behavior per
the bug writeup's "Suggested Fix" ("Make value-taking flags whose value is missing a
hard error (exit 2)"). The contract's guard code (`if c.storeMissingValue`) is
unconditional, so this happens for free — it is the intended, more-helpful outcome.
**Cover it with a test** so the exit-1→exit-2 change is locked and documented.

## 6. ZERO existing-test breakage (grep-verified — the load-bearing safety check)

The ONLY run()-level test that passes `--store` is
`TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0` (`main_test.go:2390`), which
uses `["init","--store", store]` with a VALID value → storeMissingValue=false → guard
does NOT fire → exit 0 unchanged. **No existing run()-level test passes a no-value
`--store` shape** (the bug writeup confirms: "no test covers `init --store` trailing,
no value or `--store=` empty value"). The parse-level tests T2.S1 added
(`TestParseArgs*…SetsSignal`) call `parseArgs`, NOT `run`, so they are unaffected by the
run() guard. **Net: this change breaks nothing.** (Analogous to confirming the
breakage surface before editing — here the surface is empty.)

## 7. The non-destructive claim: setupStore → configpkg.Save chain (never reached)

- `func setupStore(store, configPath string)` at `main.go:957`; calls
  `configpkg.Save(configPath, configpkg.File{Store: store})` at `:979` (the unconditional
  write that clobbers the config in the buggy path).
- `func runInit(c, stdout, stderr)` at `:998`; calls `setupStore(store, cfgPath)` at `:1012`.
- The guard returns at `~:438`, BEFORE `if c.init { return runInit(...) }` at `:455`.
  ⟹ runInit is never called ⟹ setupStore is never called ⟹ configpkg.Save never runs ⟹
  the existing config.yaml is byte-for-byte preserved. The run()-level
  "config NOT written" test (§8 test 4) proves this empirically.

## 8. Tests to add (run()-level; mirror TestRunDefaultUnknownFlag + TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0)

All use `var out, errOut bytes.Buffer; code := run([]string{...}, &out, &errOut)`:

1. **TestRunInitStoreNoValueExits2** — `["init","--store"]` → code 2; stdout EMPTY
   (`out.Len()==0`); stderr == `"skilldozer: --store requires a value\n"` (exact).
2. **TestRunStoreEqualsEmptyExits2** — `["--store="]` → code 2; stdout EMPTY; stderr message.
3. **TestRunStoreBareNoValueExits2** — `["--store"]` (no init) → code 2; stdout EMPTY;
   stderr message. (Locks the §5 exit-1→exit-2 improvement.)
4. **TestRunInitStoreNoValueDoesNotWriteConfig** — THE LOAD-BEARING non-destructive test.
   Mirror TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0's setup INVERTED:
   pre-write a config (`SKILLDOZER_CONFIG=<tmp>/config.yaml` with `store: <known>`),
   `SKILLDOZER_SKILLS_DIR=""`, `t.Chdir(t.TempDir())`; `run(["init","--store"])`;
   assert code 2; then assert the config is UNCHANGED — `configpkg.Load(cfg).Store ==
   <known>` AND the file bytes are byte-identical (os.ReadFile before/after, bytes.Equal).

Message assertion: use EXACT equality (`errOut.String() != want`) mirroring
TestRunDefaultUnknownFlag (which locks the exact `skilldozer: unknown flag '...'\n` line),
not Contains — the contract fixes the message verbatim.

## 9. Validation gates (verified executable)

```bash
cd /home/dustin/projects/skilldozer
gofmt -l main.go main_test.go        # empty
go vet ./...                          # exit 0
go test -run 'TestRun(Init)?Store|TestRunStore' -v ./...   # the 4 new run()-level tests pass
go test ./...                         # whole module green (zero regressions — §6)
git diff --quiet go.mod go.sum && echo deps unchanged
grep -c 'requires a value' main.go    # expect 2 (the guard message + the precedence comment label)
# End-to-end repro from §ISSUE 2 (must now exit 2 + config untouched):
go build -o /tmp/sd . && tmp=$(mktemp -d) && printf 'store: /tmp/B/realstore\n' > "$tmp/cfg.yaml"
SKILLDOZER_CONFIG="$tmp/cfg.yaml" env -u SKILLDOZER_SKILLS_DIR /tmp/sd init --store </dev/null >/dev/null 2>&1; echo "exit=$?"; cat "$tmp/cfg.yaml"
# Expected: exit=2 ; config still `store: /tmp/B/realstore` (unchanged). rm -rf "$tmp" /tmp/sd
```

## 10. Confidence: high. Zero breakage (§6), exact insertion by branch (§1), exact message
fixed by contract+§D2, the non-destructive chain proven (§7), and the bare-`--store`
exit-1→exit-2 change explicitly analyzed as the intended improvement (§5) and test-locked (§8).
