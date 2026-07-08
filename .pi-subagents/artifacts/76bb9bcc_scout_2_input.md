# Task for scout

Audit the NON-CORE files of the `skilldozer` repo (/home/dustin/projects/skilldozer) for drift against the CURRENT PRD.md. The code was renamed from `skpp` to `skilldozer` but docs/assets may lag.

Check and report drift/gaps for EACH of:
1. skills/example/SKILL.md — it currently says 'skpp' in multiple places (description, keywords, body). The PRD §11 gives the canonical content (says 'skilldozer', keywords [example, demo, skilldozer], category meta). Document every line that must change. Note this also affects --search (keyword 'skpp').
2. README.md — compare against PRD §15 outline (8 sections). Does it mention `skilldozer init` and the config model (§8)? Does it cover --path, --search, --list, --all, check, -f, --relative, --no-color? Does it describe SKILLDOZER_SKILLS_DIR and the config file? List missing/outdated sections.
3. install.sh — compare against PRD §12.1. Does it use the exact go build ldflags? Does it symlink? Does it print 'skilldozer example' verification? Note any deviation.
4. completions/ (skilldozer.bash, _skilldozer, skilldozer.fish) — do they reference 'skpp'? Do they complete the `init` subcommand and --store/--file/--relative/--no-color flags per §14? PRD §14 says tags come from `skilldozer --all`.
5. .gitignore — must match PRD §16 exactly (5 entries: /skilldozer, /dist, *.test, *.out, .DS_Store). Report current contents and any diff.
6. LICENSE — must be MIT per §19. Confirm.
7. go.mod — module github.com/dabstractor/skilldozer, go directive. PRD says 'latest two stable releases'. Report current go directive.

Read each file. For each: quote current relevant lines and cite the PRD section it must satisfy. Do NOT fix anything — just enumerate the precise drift. Write findings to plan/002_38acb6d28a6a/architecture/docs_and_assets_drift.md. Return a short summary of the top drift items.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/76bb9bcc/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/76bb9bcc/plan/002_38acb6d28a6a/architecture/docs_and_assets_drift.md
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