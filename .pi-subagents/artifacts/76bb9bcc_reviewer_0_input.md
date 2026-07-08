# Task for reviewer

[Read from: /home/dustin/projects/skilldozer/plan.md, /home/dustin/projects/skilldozer/progress.md]

You are auditing the EXISTING Go codebase of the `skilldozer` CLI (repo at /home/dustin/projects/skilldozer) against its CURRENT PRD (/home/dustin/projects/skilldozer/PRD.md). The code was originally built for an older `skpp` PRD and partially updated; a new planning pass (plan 002) must capture every remaining GAP and DRIFT.

Read PRD.md fully, then read main.go and every file under internal/ (skillsdir, discover, resolve, ui, search, check). Produce a PRECISE, EXHAUSTIVE gap report.

Focus especially hard on:
1. PRD §8 (Locating the skills directory) — the code's skillsdir.go only implements 3 rules (env -> sibling -> walk-up). PRD §8.3 now requires 5 rules: (1) SKILLDOZER_SKILLS_DIR env, (2) config file `store` key, (3) sibling of binary, (4) walk-up from cwd, (5) none => 'skilldozer is not configured; run `skilldozer init`'. Document EXACTLY what must change: new config rule, renumbering, new Source label 'config file', revised ErrNotFound message.
2. PRD §8.1 config file: $XDG_CONFIG_HOME/skilldozer/config.yaml, SKILLDOZER_CONFIG override, `store:` key, yaml.v3, unknown keys ignored, missing/unreadable => treated as not-configured (never hard error). Document what is missing.
3. PRD §8.2 `init` subcommand: interactive TTY flow (cwd auto-detect -> default $XDG_DATA_HOME/skilldozer/skills -> prompt), non-interactive (`init <dir>` / `init --store <dir>`), mkdir -p, seed template example/SKILL.md if empty (string constant, NOT go:embed), write config.yaml with absolute store path, print --path + check output. Bare tag resolution NEVER prompts. Document the full init design needed.
4. PRD §6.1 CLI table — confirm `init` row is missing; confirm `--path`/`check`/modes all present and correct.
5. Any OTHER drift: main.go flag handling vs §6.1-6.4, error/exit semantics, exclusivity rules.

For each gap: cite the exact file:line(s) of the current code and the exact PRD section. Be exhaustive. Do NOT propose a fix list — just enumerate gaps precisely with code references and PRD citations.

Write your report to the file plan/002_38acb6d28a6a/architecture/code_prd_delta.md (create the dir if needed). Return a short summary of the top gaps.

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/76bb9bcc/plan/002_38acb6d28a6a/architecture/code_prd_delta.md
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