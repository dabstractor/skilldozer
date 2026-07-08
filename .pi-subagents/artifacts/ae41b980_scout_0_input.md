# Task for scout

In the repo at /home/dustin/projects/skilldozer, verify the current state of the §13 acceptance criteria against the ACTUAL built binary and source. This is for a PRP for work item P1.M4.T1.S1 which executes PRD §13 acceptance end-to-end.

The repo has a built binary at /home/dustin/projects/skilldozer/skilldozer. Run this EXACT verification script (use `set +e` so it continues through failures) and capture each step's output + rc:

```bash
set +e
cd /home/dustin/projects/skilldozer
echo '== 1 build =='; go build -o skilldozer . && echo OK; echo build_rc=$?
echo '== 2 version =='; ./skilldozer --version
echo '== 3 path sibling =='; test "$(./skilldozer --path)" = "$PWD/skills"; echo path_rc=$?
echo '== 4 list =='; ./skilldozer --list
echo '== 5 example dir =='; test -d "$(./skilldozer example)"; echo example_dir_rc=$?
echo '== 6 file =='; test -f "$(./skilldozer -f example)"; echo file_rc=$?
echo '== 7 unknown-tag =='; out=$(./skilldozer nope 2>/dev/null); rc=$?; echo "rc=$rc out=[$out]"
echo '== 8 absolute =='; case "$(./skilldozer example)" in /*) echo abs_OK;; *) echo abs_FAIL;; esac
echo '== 9 check =='; ./skilldozer check; echo check_rc=$?
echo '== 10 symlink =='; mkdir -p /tmp/sd-bin && ln -sf "$PWD/skilldozer" /tmp/sd-bin/skilldozer && /tmp/sd-bin/skilldozer example; echo symlink_rc=$?
echo '== 11 env override =='; SKILLDOZER_SKILLS_DIR="$PWD/skills" ./skilldozer example; echo env_rc=$?
echo '== 12 unconfigured hint =='; mkdir -p /tmp/sd-iso/home && cp ./skilldozer /tmp/sd-iso/skilldozer && cd /tmp/sd-iso && env -u SKILLDOZER_SKILLS_DIR HOME=/tmp/sd-iso/home XDG_CONFIG_HOME=/tmp/sd-iso/home/.config ./skilldozer x 2>err; rc=$?; echo rc=$rc; grep -q 'run `skilldozer init`' err && echo hint_OK || echo hint_FAIL; cd - >/dev/null
echo '== 13 non-interactive init =='; SKILLDOZER_CONFIG=/tmp/sd-iso/cfg.yaml /tmp/sd-iso/skilldozer init --store /tmp/sd-store; echo init_rc=$?; test -d /tmp/sd-store && echo store_OK; grep -q 'store: /tmp/sd-store' /tmp/sd-iso/cfg.yaml && echo cfg_OK
echo '== 14 config rule wins =='; SKILLDOZER_CONFIG=/tmp/sd-iso/cfg.yaml /tmp/sd-iso/skilldozer --path; SKILLDOZER_CONFIG=/tmp/sd-iso/cfg.yaml /tmp/sd-iso/skilldozer --path 2>&1 | grep -q /tmp/sd-store && echo cfgwins_OK || echo cfgwins_FAIL
echo '== 15 env beats config =='; SKILLDOZER_SKILLS_DIR=/tmp/sd-store SKILLDOZER_CONFIG=/tmp/sd-iso/cfg.yaml /tmp/sd-iso/skilldozer --path 2>&1 | grep -q SKILLDOZER_SKILLS_DIR && echo envbeats_OK || echo envbeats_FAIL
```

For EACH step, report: PASS/FAIL + the actual output you observed. If a step fails, capture the exact error and note which source file likely owns it. Write findings to plan/002_38acb6d28a6a/P1M4T1S1/research/acceptance_run.md (mkdir -p the dir first). Report a concise summary back listing PASS/FAIL per step.

---
Update progress at: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/progress/ae41b980/progress.md

---
**Output:**
Write your findings to exactly this path: /home/dustin/projects/skilldozer/.pi-subagents/artifacts/outputs/ae41b980/plan/002_38acb6d28a6a/P1M4T1S1/research/scout_acceptance.txt
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