# Task for scout

Read the current README at /home/dustin/projects/skilldozer/README.md (271 lines) and the example skill at /home/dustin/projects/skilldozer/skills/example/SKILL.md. I'm writing a PRP to update the README for the config model.

Report back concisely (quote verbatim with line numbers):
1. The EXACT current text of README.md lines 36-53 (the '## Install' → 'B. go install' caveat block to be removed). 
2. The EXACT current text of README.md lines 234-251 (the '## How skilldozer finds the store' 3-rule + --path labels paragraph, to become 5 rules).
3. The current '## Constraints' section (lines ~253-271) verbatim — especially the 'Manifest-free' bullet to reword to 'no catalog index (disk-discovered); a settings config file is fine'.
4. Confirm README does NOT mention `skilldozer init`, `config.yaml`, `SKILLDOZER_CONFIG`, or 'config file' anywhere (grep).
5. The title one-liner (line ~3) and the canonical '## Usage' example block (lines ~75-120) so flag docs are preserved and voice matched.
6. The example skill's frontmatter (name/description/keywords) from skills/example/SKILL.md.

Read-only; do not modify anything.

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/fc32afe3/plan/002_38acb6d28a6a/P1M4T2S1/research/readme_anchors.md
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