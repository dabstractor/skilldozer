# Task for researcher

Research the EXTERNAL technical facts needed to implement the `skilldozer` Go CLI's new config + init model (PRD §8). Repo is at /home/dustin/projects/skilldozer (already uses gopkg.in/yaml.v3, go 1.25). You may read files in the repo for grounding but the research is about external/stdlib idioms.

Produce a research brief covering:
1. Go yaml.v3: the idiomatic way to READ a small config struct from a YAML file (yaml.Unmarshal into a struct with a `Store string` field) AND WRITE one back (yaml.Marshal). Confirm yaml.v3 ignores unknown keys when unmarshaling into a concrete struct (so the file can grow). Show minimal code patterns. Note: this is the ONLY allowed third-party dep.
2. XDG semantics on Linux: the correct default for $XDG_CONFIG_HOME (-> ~/.config) and $XDG_DATA_HOME (-> ~/.local/share) when the env var is unset. How to resolve in Go (os.UserConfigDir / os.UserHomeDir / os.Getenv). Confirm skilldozer's planned paths: config at $XDG_CONFIG_HOME/skilldozer/config.yaml, default store at $XDG_DATA_HOME/skilldozer/skills.
3. TTY detection in Go WITHOUT adding a dependency: how to detect whether stdin is a TTY using only the standard library (e.g. os.Stdin.Stat() & ModeCharDevice, or the(*os.FileState)). Also note the common alternative golang.org/x/term/IsTerminal but flag that it's an extra module — recommend the stdlib approach to keep zero extra runtime deps beyond yaml.v3, OR confirm x/term is acceptable.
4. Interactive prompt reading from stdin in Go (bufio.Reader / bufio.Scanner ReadString('\n')). Minimal pattern for 'prompt, accept default on empty Enter'.
5. os.Executable() + filepath.EvalSymlinks behavior for symlink-installed binaries on Linux (so the sibling-of-binary rule resolves back to the repo). Confirm `go install` puts the binary in $(go env GOPATH)/bin and that os.Executable returns that real path (so the sibling rule will MISS for go-install users, which is exactly why the config rule is needed).
6. Verify the pi `--skill <dir>` and `--skill <file>` contract from pi docs at /home/dustin/.pi/agent/npm/node_modules/pi-subagents/... no — instead check the pi skills doc: find and read any docs/skills.md under ~/.pi (there are several). Confirm: a skill is a dir with SKILL.md; --skill accepts a dir or file; --no-skills still allows explicit --skill. Quote the relevant lines.

Be concrete with code snippets and cite sources. Write the brief to plan/002_38acb6d28a6a/architecture/external_deps.md. Return a short summary.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/76bb9bcc/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/76bb9bcc/plan/002_38acb6d28a6a/architecture/external_deps.md
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