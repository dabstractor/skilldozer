# Task for scout

Reproduce and document bugfix Issues 1-4 in the skilldozer Go CLI against the built binary at ./skilldozer. Working dir: /home/dustin/projects/skilldozer. For EACH issue, run the exact reproduction steps from the PRD, confirm the buggy behavior, and document the EXACT code location (file:line range) and function name where the fix must go.

ISSUE 1: `init` writes the check validation report to stdout (should be stderr). In main.go runInit() (around line 988), the check report loop uses fmt.Fprintf(stdout, ...) for the OK/WARN/ERROR lines AND the summary line '%d skills, %d errors, %d warnings'. Confirm the store path (line 'fmt.Fprintln(stdout, dir)') should stay on stdout but the check report + summary must move to stderr. Run: `cd /tmp && mkdir -p /tmp/A/store; SKILLDOZER_CONFIG=/tmp/A/cfg.yaml env -u SKILLDOZER_SKILLS_DIR ./skilldozer init --store /tmp/A/store </dev/null 2>/dev/null` and capture stdout showing 3 lines.

ISSUE 2: `init --store` with no value silently overwrites config. In main.go parseArgs() the `--store` case (and `--store=` form) only sets c.init/c.initStore when i+1 < len(args); when --store is the LAST token it does nothing, so c.init is false BUT the earlier 'init' token may have set c.init=true. Actually trace carefully: does `skilldozer init --store` (init token present) end up running init with auto-detect? Run the exact repro from the PRD (/tmp/B test) and confirm the config gets overwritten. Also confirm `skilldozer --store` (no init token) behavior. Document the exact line in parseArgs and where run() should catch the missing-value.

ISSUE 3: tag + --path not rejected. exclusivityError() (main.go ~line 686) has `hasTags && (c.list || c.searchMode || c.all)` — note c.path is MISSING from that predicate (but c.path IS in the listing-modes count set). Run: `export SKILLDOZER_SKILLS_DIR=/tmp/sk; mkdir -p /tmp/sk; ./skilldozer NONEXISTENTTAG --path; echo exit=$?` (expect 0, should be 2). Document the exact predicate line.

ISSUE 4: `init init` runs init. In parseArgs() the 'init' case sets c.init=true and does NOT capture the next token if it is 'check' or 'init', but a second 'init' just re-sets c.init=true. Run: `./skilldozer init init </dev/null >/dev/null 2>&1; echo exit=$?` (expect 0). Compare with `./skilldozer init check` (expect 2). Document the exact 'init' case logic.

Write your findings to /home/dustin/projects/skilldozer/plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/issues_1_to_4_validation.md with: per-issue (a) confirmed repro output, (b) exact file:function + line range, (c) the precise code change needed, (d) which existing tests cover/relate to it.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/23511bc0/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/23511bc0/plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/issues_1_to_4_validation.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: checked
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Implement the requested change without widening scope

Required evidence: changed-files, tests-added, commands-run, residual-risks, no-staged-files

Finish with a fenced JSON block tagged `acceptance-report` in this shape:
Use empty arrays when no items apply; array fields contain strings unless object entries are shown.
```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "specific proof"
    }
  ],
  "changedFiles": [
    "src/file.ts"
  ],
  "testsAddedOrUpdated": [
    "test/file.test.ts"
  ],
  "commandsRun": [
    {
      "command": "command",
      "result": "passed",
      "summary": "short result"
    }
  ],
  "validationOutput": [
    "validation output or concise summary"
  ],
  "residualRisks": [
    "none"
  ],
  "noStagedFiles": true,
  "diffSummary": "short description of the diff",
  "reviewFindings": [
    "blocker: file.ts:12 - issue found, or no blockers"
  ],
  "manualNotes": "anything else the parent should know"
}
```