---
name: e2e-triage
description: Triage E2E test failures — run locally with mise or via CI re-runs, classify flaky vs real bug. Local mode presents findings and applies fixes in-place; CI mode creates PRs for flaky fixes and GitHub issues for real bugs.
---

# E2E Triage

Triage E2E test failures with **re-run verification**. Operates in two modes (auto-detected), analyzes artifacts, and re-runs failing tests to distinguish flaky from real bugs. **Local mode** presents findings interactively and applies fixes directly in the working tree. **CI mode** creates batched PRs for flaky fixes and GitHub issues for real bugs.

## Mode Detection

The two modes share the same analysis and classification logic but differ in how results are presented and acted upon. Local mode is interactive (user reviews findings, chooses what to fix); CI mode is automated (PRs and issues created directly).

- **CI mode**: `WORKFLOW_RUN_ID` env var is set (injected by `e2e-triage.yml`)
- **Local mode**: No `WORKFLOW_RUN_ID` — user invokes `/e2e-triage` manually

---

## Local Mode

### Step L1: Parse User Input

The user provides one or more of:
- **Test name(s)** — e.g., `TestInteractiveMultiStep`
- **`--agent <agent>`** — optional, defaults to all agents that previously failed
- **A local artifact path** — skip straight to analysis of existing artifacts

**Cost warning:** Real E2E tests consume API tokens. Before running, confirm with the user unless they provided specific test names (implying intent to run).

### Step L2: First Run

```bash
mise run test:e2e --agent <agent> <TestName>
```

Capture the artifact directory from the `artifacts: <path>` output line.

### Step L3: Re-run on Failure

If the test **passes** on first run: report as passing, done for this test.

If the test **fails**: run a **second time** with the same parameters.

### Step L4: Tiebreaker (if needed)

If results are **split** (1 pass, 1 fail): run a **third time** as tiebreaker.

### Step L5: Collect Results

For each test+agent pair, record: `(test, agent, run_1_result, run_2_result, [run_3_result])`

Proceed to **Shared Analysis** (Step 1 below).

---

## CI Mode

### Step C1: Download Artifacts

**If given a run ID, URL, or "latest":**
```bash
artifact_dir=$(scripts/download-e2e-artifacts.sh <input>)
```

**If `WORKFLOW_RUN_ID` is set (automated trigger):**
```bash
artifact_dir="e2e/artifacts/ci-${WORKFLOW_RUN_ID}"
```

The download step in the workflow has already placed artifacts there.

### Step C2: Identify Failures

For each agent subdirectory in the artifact root:
1. Read `report.nocolor.txt` — list failed tests with error messages, file:line references
2. Skip agents with zero failures
3. Build failure list: `[(test_name, agent, error_line, duration, file:line)]`

### Step C3: Re-run Failing Tests via CI

For each failing test+agent pair, **sequentially**:

1. **Trigger re-run:**
   ```bash
   gh workflow run e2e-isolated.yml -f agent=<agent> -f test=<TestName>
   ```

2. **Wait for run to register** (~5s), then find the run ID:
   ```bash
   gh run list -w e2e-isolated.yml -L 1 --json databaseId -q '.[0].databaseId'
   ```

3. **Poll until complete** (check every 30s, timeout after 25 minutes):
   ```bash
   gh run view <run-id> --json status,conclusion
   ```

4. **Download re-run artifacts:**
   ```bash
   gh run download <run-id> --dir <rerun-artifact-dir>
   ```

5. **Repeat for a second re-run** (same test+agent).

### Step C4: Collect Results

For each test+agent pair, record: `(test, agent, original_result, rerun_1_result, rerun_2_result)`

Proceed to **Shared Analysis** (Step 1 below).

---

## Shared Analysis & Classification

### Step 1: Analyze Each Failure

For each failure, examine available artifacts:

1. **Read `console.log`** — what did the agent actually do? Full chronological transcript.
2. **Read test source at file:line** — what was expected?
3. **Read `entire-logs/entire.log`** — any CLI errors, panics, unexpected behavior?
4. **Read `git-log.txt` / `git-tree.txt`** — repo state at failure time
5. **Read `checkpoint-metadata/`** — corrupt or missing metadata?

### Step 2: Classify Each Failure

Use **re-run results as the primary signal**, supplemented by artifact analysis.

#### Re-run signals (strongest):

| Original | Re-run 1 | Re-run 2 | Classification |
|----------|----------|----------|----------------|
| FAIL | FAIL (same error) | FAIL (same error) | **real-bug** |
| FAIL | PASS | PASS | **flaky** |
| FAIL | PASS | FAIL | **flaky** (non-deterministic) |
| FAIL | FAIL | PASS | **flaky** (non-deterministic) |
| FAIL | FAIL (different error) | FAIL (different error) | **needs deeper analysis** — examine artifacts |

#### Strong `real-bug` signals (supplement re-runs):

- `entire.log` contains `"level":"ERROR"` or panic/stack traces
- Checkpoint metadata structurally corrupt (malformed JSON, missing `checkpoint_id`/`strategy`)
- Session state file missing or malformed when expected
- Hooks did not fire at all (no `hook invoked` log entries)
- Shadow/metadata branch has wrong tree structure
- Same test fails across 3+ agents with same non-timeout symptom
- Error references CLI code (panic in `cmd/entire/cli/`)

#### Strong `flaky` signals (unless overridden by real-bug):

- `signal: killed` (timeout)
- `context deadline exceeded` or `WaitForCheckpoint.*exceeded deadline`
- Agent asked for confirmation instead of acting
- Agent created file at wrong path / wrong name
- Agent produced no output
- Agent committed when it shouldn't have (or vice versa)
- Duration near timeout limit

#### Ambiguous cases:

Read `entire.log` carefully:
- If hooks fired correctly and metadata is valid -> lean **flaky**
- If hooks fired but produced wrong results -> lean **real-bug**

### Step 3: Cross-Agent Correlation

Before acting, check correlations using re-run data:
- Same test fails for 3+ agents, all re-runs also fail -> strong **real-bug**
- Same test fails for multiple agents, but re-runs pass -> **flaky** (shared prompt issue)
- One agent fails consistently, others pass -> agent-specific issue (still **real-bug** if re-runs confirm)

### Step 4a: Take Action — Local Mode

In local mode, present findings interactively. **Do not** create branches, PRs, or GitHub issues.

#### Present Findings Report

For each test+agent pair, print a findings block:

```
## <TestName> (<agent>) — <classification>

**Re-run results:** original=FAIL, rerun1=PASS, rerun2=PASS
**Evidence:**
- <1-2 sentence summary of what went wrong>
- <key artifact evidence: entire.log excerpt, console.log excerpt, etc.>
```

#### For `flaky` failures: describe the proposed fix

```
**Proposed fix:** <description>
  - File: <path to test file>
  - Change: <what will be modified — e.g., append "Do not ask for confirmation" to prompt>
```

Common flaky fixes (same as CI mode):
- Agent asked for confirmation -> append "Do not ask for confirmation" to prompt
- Agent wrote to wrong path -> be more explicit about paths in prompt
- Agent committed when shouldn't -> add "Do not commit" to prompt
- Checkpoint wait timeout -> increase timeout argument
- Agent timeout (signal: killed) -> increase per-test timeout, simplify prompt

#### For `real-bug` failures: describe root cause analysis

```
**Root cause analysis:**
  - Component: <hooks | session | checkpoint | strategy | agent>
  - Suspected location: <file:function>
  - Description: <what's wrong and why>
  - Proposed fix: <what code change would address it>
```

#### Ask the user

Prompt the user:

> **Should I fix these?**
> - [list of tests with classifications]
> - You can select all, specific tests, or skip.

Wait for user response before proceeding.

#### Apply fixes (if user approves)

For **flaky** fixes the user approved:
1. Apply fixes directly in the working tree (no branch creation)
2. Run verification:
   ```bash
   mise run test:e2e:canary   # Must pass
   mise run fmt && mise run lint
   ```
3. If canary fails, investigate and adjust. Report what happened to the user.

For **real-bug** fixes the user approved:
1. Apply the fix directly in the working tree (no branch creation)
2. Run relevant tests to verify:
   ```bash
   mise run test        # Unit tests
   mise run test:e2e:canary  # Canary tests
   ```
3. Report results to the user.

### Step 4b: Take Action — CI Mode

#### For `flaky` failures: Batched PR

1. Create branch `fix/e2e-flaky-<run-id-or-date>`
2. Apply fixes to ALL flaky test files (one branch, one PR):
   - Agent asked for confirmation -> append "Do not ask for confirmation" to prompt
   - Agent wrote to wrong path -> be more explicit about paths in prompt
   - Agent committed when shouldn't -> add "Do not commit" to prompt
   - Checkpoint wait timeout -> increase timeout argument
   - Agent timeout (signal: killed) -> increase per-test timeout, simplify prompt
3. Run verification:
   ```bash
   mise run test:e2e:canary   # Must pass
   mise run fmt && mise run lint
   ```
4. If canary fails, investigate and adjust. If unfixable, fall back to issue creation.
5. Commit and create PR:
   ```bash
   gh pr create \
     --title "fix(e2e): make flaky tests more resilient" \
     --body "<structured body with per-test changes, re-run evidence, run link>"
   ```

#### For `real-bug` failures: Issue (with dedup)

1. **Search existing issues first:**
   ```bash
   gh issue list --search "is:open label:e2e <TestName>" --json number,title,url
   ```

2. **If matching issue exists:** Add a comment with new evidence:
   ```bash
   gh issue comment <number> --body "<re-run results, verification details, run link, new evidence>"
   ```
   Note "Verified still failing — reproduced in N/N re-runs" plus any new diagnostic details.

3. **If no matching issue:** Create new:
   ```bash
   gh issue create \
     --title "E2E: <TestName> fails — <brief symptom>" \
     --label "bug,e2e" \
     --body "<structured body>"
   ```

   Issue body includes:
   - Test name, agent(s), CI run link (if available), re-run results
   - Failure summary (expected vs actual)
   - Root cause analysis (which CLI component: hooks, session, checkpoint, attribution, strategy)
   - Key evidence: `entire.log` excerpts, `console.log` excerpts, git state
   - Reproduction steps
   - Suspected fix location (file, function, reason)

### Step 5: Summary Report

#### Local mode

Print a summary table:
```
| Test | Agent(s) | Re-runs | Classification | Action Taken |
|------|----------|---------|---------------|-------------|
| TestFoo | claude-code | FAIL/PASS/PASS | flaky | Fixed in working tree |
| TestBar | all agents | FAIL/FAIL/FAIL | real-bug | Fix applied, tests passing |
| TestBaz | opencode | FAIL/PASS/FAIL | flaky | Skipped (user declined) |
```

No `triage-summary.json` is written in local mode.

#### CI mode

Print a summary table:
```
| Test | Agent(s) | Re-runs | Classification | Action | Link |
|------|----------|---------|---------------|--------|------|
| TestFoo | claude-code | FAIL/PASS/PASS | flaky | PR #123 | url |
| TestBar | all agents | FAIL/FAIL/FAIL | real-bug | Issue #456 (existing, commented) | url |
| TestBaz | opencode | FAIL/PASS/FAIL | flaky | PR #123 | url |
```

The "Re-runs" column shows original/rerun1/rerun2 results.

**Write `triage-summary.json`** in the artifact directory for Slack notifications:

```bash
cat > "${ARTIFACT_DIR}/triage-summary.json" << 'TEMPLATE'
{
  "actions": [
    {
      "test": "TestName",
      "agents": ["claude-code", "opencode"],
      "classification": "flaky|real-bug",
      "rerun_results": ["FAIL", "PASS", "PASS"],
      "action_type": "pr|issue|comment",
      "action_description": "PR #123|Issue #456 (new)|Issue #456 (commented)",
      "link": "https://github.com/entireio/cli/pull/123"
    }
  ]
}
TEMPLATE
```

Each entry in `actions` corresponds to one row in the summary table. Include all failures that had an action taken. The `link` field must be the full URL to the PR or issue.
