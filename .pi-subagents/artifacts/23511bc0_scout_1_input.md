# Task for scout

Reproduce and document bugfix Issues 5-7 in the skilldozer Go CLI plus the testing infrastructure. Working dir: /home/dustin/projects/skilldozer.

ISSUE 5: Tilde (~) not expanded in init interactive prompt. In main.go resolveStore() (around line 865), the chosen store path is absolutized with filepath.Abs(store) which does NOT expand ~. So `~/myskills` becomes `/<cwd>/~/myskills`. This is in the INTERACTIVE path (chooseStore → typed path). Document resolveStore's exact location and the fact that filepath.Abs does not expand tildes. Note: the test seam for this is chooseStore() (pure, takes a prompt fn) but resolveStore() is where filepath.Abs is called and where tilde expansion must be added.

ISSUE 6: .gitignore has extra entries beyond PRD §16 spec. The §16 spec set is EXACTLY: `/skilldozer`, `/dist`, `*.test`, `*.out`, `.DS_Store` (5 entries). Read the current .gitignore and list every extra entry. The current file has section comments + `/build`, `node_modules/`, `venv/`, `.env`, `.pi-subagents/`. Confirm by reading the file.

ISSUE 7: A skill whose canonical tag is literally `check` or `init` cannot be resolved by that tag. In parseArgs() the `case "check"` and `case "init"` treat them as reserved subcommand tokens, shadowing any skill at skills/check/SKILL.md. Document that the code already comments this as deliberate. The PRD's suggested fix is 'add a PRD note to §7.2' OR code change. Determine which approach fits — this is a DOC fix (update decisions/PRD note) not necessarily code, OR document the code alternative.

ALSO DOCUMENT the testing infrastructure: how main_test.go tests assert on stdout vs stderr (they pass two *bytes.Buffer to run()), the helper functions like sampleStore(t), writeSkillTree(t), unsetSkillsEnv(t), and the test patterns for init (TestRunInitStoreWritesConfigCreatesStorePrintsPathExit0 at line 2325, TestParseArgsInitStoreLongForm at line 1186). Note how tests set SKILLDOZER_CONFIG and SKILLDOZER_SKILLS_DIR via t.Setenv and use t.TempDir().

Write findings to /home/dustin/projects/skilldozer/plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/issues_5_to_7_and_test_infra.md with per-issue: confirmed repro/behavior, exact file:line, precise code change, and the relevant test pattern + helper to reuse.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/23511bc0/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/23511bc0/plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/issues_5_to_7_and_test_infra.md
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