# Task for scout

Read the file /home/dustin/projects/skilldozer/plan/002_38acb6d28a6a/architecture/system_context.md in full. Extract and report back VERBATIM (quote exact lines): (1) any mention of the 'mcpeepants README' tone/structure that PRD §15 says skilldozer's README should mirror — describe the tone/voice/structure; (2) the §8.1 config file path ($XDG_CONFIG_HOME/skilldozer/config.yaml) and SKILLDOZER_CONFIG override details; (3) the §8.2 init flow summary (cwd auto-detect default $XDG_DATA_HOME/skilldozer/skills, `init <dir>` / `init --store <dir>` non-interactive, prints --path + check output, never-prompts on bare tag); (4) the §8.3 five-rule resolution ladder and the four --path labels (SKILLDOZER_SKILLS_DIR, config file, sibling of binary, ancestor of cwd); (5) the §11 example skill frontmatter (name/description/keywords) so the README '## Adding a skill' template aligns. Report as concise quoted bullets with the section each came from. Read-only; do not modify anything.

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/29970a76/plan/002_38acb6d28a6a/P1M4T2S1/research/system_context_extract.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: attested
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Return concrete findings with file paths and severity when applicable

Required evidence: review-findings, residual-risks

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