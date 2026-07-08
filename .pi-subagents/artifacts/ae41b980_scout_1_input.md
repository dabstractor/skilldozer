# Task for scout

In the repo at /home/dustin/projects/skilldozer, verify compliance of the four 'compliant files' that PRD §13 acceptance (work item P1.M4.T1.S1) grep-confirms at the end. This is for a PRP.

PRD is at /home/dustin/projects/skilldozer/PRD.md. First read these PRD sections to get the EXACT requirements:
- §16 .gitignore (heading h2.15) — the exact 5 required entries
- §12.1 install.sh (heading h3.11) — ldflags, symlink, prints `skilldozer example`
- §4 / §1 — go.mod module + single yaml.v3 dep; LICENSE = MIT

Then check each file and report PASS/FAIL with EXACT evidence (grep commands + line numbers):

1. `.gitignore` — get the 5 required patterns from PRD §16. Read /home/dustin/projects/skilldozer/.gitignore. Confirm all 5 are present. List any extras.

2. `LICENSE` — `head -3` must show 'MIT License'. Report.

3. `go.mod` — module must be `github.com/dabstractor/skilldozer`, only require is `gopkg.in/yaml.v3 v3.0.1`. Run `cat go.mod`.

4. `install.sh` — verify: (a) uses `go build` with `-ldflags` (look for `-X main.version` or similar), (b) creates a symlink (ln -s), (c) prints/echoes `skilldozer example` as the verification command. Read the whole file. Report each sub-check PASS/FAIL with line numbers.

For each file, provide the EXACT grep command the acceptance script should use. Write findings to plan/002_38acb6d28a6a/P1M4T1S1/research/compliant_files.md (mkdir -p the dir first). Report a concise summary back.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/ae41b980/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/ae41b980/plan/002_38acb6d28a6a/P1M4T1S1/research/scout_compliant.txt
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