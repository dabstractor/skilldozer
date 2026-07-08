# Task for researcher

Research the idiomatic Go approach for expanding a leading tilde (~) in a filesystem path string, for a CLI tool that accepts a user-typed path like '~/myskills' or '~'. I need to know:

1. Does the Go standard library provide a function to expand '~' to the user's home directory? (e.g. is there an os.ExpandHome, filepath.ExpandHome, or similar? Confirm there is NO such stdlib function in the current Go standard library — this is a well-known gap.)
2. What is the canonical manual pattern? Specifically: use os.UserHomeDir() (returns $HOME on Unix, %USERPROFILE% on Windows) and strings.HasPrefix(p, "~/") to replace the prefix. Also handle a bare '~' (just the tilde alone) → home dir.
3. Edge cases: should '~user' (tilde + other username) be expanded? (Generally NO for simple CLI tools — only '~/' and '~' for the current user.) What about '~foo' (tilde + non-slash)? (Should be left alone — it's a relative path, not a home reference.)
4. Confirm that filepath.Abs does NOT expand tildes (it only makes a path absolute relative to cwd).
5. Is there any concern with os.UserHomeDir() returning an error (empty HOME env)? How should a CLI handle that — fall back to leaving the path as-is?

Provide a short code snippet of the recommended expandTilde helper function (Go), and cite the relevant Go stdlib docs. This is for a project whose ONLY third-party dependency is gopkg.in/yaml.v3 (it must stay stdlib-only besides that), so do NOT recommend any third-party tilde-expansion library.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/23511bc0/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/23511bc0/plan/002_38acb6d28a6a/bugfix/001_ad9c7988274c/architecture/go_tilde_expansion.md
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