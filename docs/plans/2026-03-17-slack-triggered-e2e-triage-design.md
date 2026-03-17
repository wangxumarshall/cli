# Slack-Triggered E2E Triage Design

## Goal

Allow a human to reply `triage e2e` in the Slack thread for an E2E failure alert and have GitHub Actions run the repo's existing Claude E2E triage workflow based on [`.claude/skills/e2e/triage-ci.md`](../../.claude/skills/e2e/triage-ci.md).

## Scope

This design covers:

- Slack trigger detection for a single exact-match phrase: `triage e2e`
- Hand-off from Slack to GitHub Actions
- A new GitHub Actions workflow that runs triage against an existing failed CI run
- Posting triage status and results back into the originating Slack thread

This design does not cover:

- Automatic remediation or code changes
- Running the full E2E fix pipeline
- General-purpose Slack command routing
- Local rerun verification beyond what the existing skill supports for CI run references

## Existing Context

The repo already has the core ingredients needed for the triage operation:

- [`.github/workflows/e2e.yml`](../../.github/workflows/e2e.yml) posts Slack alerts when E2E runs on `main` fail
- [`.claude/skills/e2e/triage-ci.md`](../../.claude/skills/e2e/triage-ci.md) defines the triage procedure
- [`.claude/plugins/e2e/commands/triage-ci.md`](../../.claude/plugins/e2e/commands/triage-ci.md) exposes the procedure as the `/e2e:triage-ci` command
- [`scripts/download-e2e-artifacts.sh`](../../scripts/download-e2e-artifacts.sh) already supports artifact download from a GitHub Actions run reference

The missing piece is the Slack-to-GitHub bridge.

## Architecture

The system is composed of three narrow responsibilities:

1. Slack app
   - Listen for new thread replies
   - Normalize reply text
   - Trigger only when the reply text is exactly `triage e2e`
   - Validate that the parent message is an E2E failure alert for this repository

2. Dispatch bridge
   - Read structured data from the parent Slack alert
   - Build a `repository_dispatch` payload for this repository
   - Send the dispatch event to GitHub

3. GitHub Action
   - Receive the dispatch payload
   - Check out the repository at the failed commit SHA
   - Install and authenticate Claude CLI
   - Load the local plugin directory at [`.claude/plugins/e2e`](../../.claude/plugins/e2e)
   - Invoke `/e2e:triage-ci` with the CI run URL and failed agent
   - Upload artifacts and post results back to the Slack thread

This keeps Slack focused on intent capture and routing while GitHub Actions remains the execution environment for triage.

## Trigger Contract

The Slack app should send a structured `repository_dispatch` event with custom type `slack_e2e_triage_requested`.

Recommended payload:

```json
{
  "trigger_text": "triage e2e",
  "repo": "entireio/cli",
  "branch": "main",
  "sha": "447cde1aeee938448c3edbae78242c950dc35cf0",
  "run_url": "https://github.com/entireio/cli/actions/runs/123456789",
  "run_id": "123456789",
  "failed_agents": ["cursor-cli"],
  "slack_channel": "C123456",
  "slack_thread_ts": "1742230000.123456",
  "slack_user": "U123456"
}
```

Workflow-side validation rules:

- Reject if `trigger_text` is not exactly `triage e2e`
- Reject if `run_url` or `slack_thread_ts` is missing
- Reject if the target repo or branch is unexpected
- Treat `failed_agents` as the source of truth for which agent-specific triage jobs to run

## Slack Message Requirements

The current Slack failure notification in [`.github/workflows/e2e.yml`](../../.github/workflows/e2e.yml) already includes the run details link, commit SHA, actor, and failed agent list. That is enough for a first version if the Slack app parses the parent message.

However, the safer design is to make the alert payload more machine-friendly so the Slack app does not need to scrape display text. Two acceptable options:

- Add stable metadata in the Slack message text or blocks for `run_url`, `sha`, and `failed_agents`
- Store a compact JSON blob in a Slack block element or message metadata if the chosen Slack app framework supports it

The first version can parse the existing message format, but the implementation should isolate that parsing into one small component because it is brittle compared to a structured payload.

## GitHub Workflow Design

Add a new workflow at [`.github/workflows/e2e-triage.yml`](../../.github/workflows/e2e-triage.yml).

### Triggers

- `repository_dispatch` with type `slack_e2e_triage_requested`
- `workflow_dispatch` for manual testing and debugging

### High-Level Job Flow

1. Validate dispatch payload
2. Post "triage started" reply to the Slack thread
3. Check out repository at the failed `sha`
4. Set up `mise`
5. Install Claude CLI and any required dependencies
6. Authenticate Claude using a GitHub Actions secret
7. Run the E2E triage command:

```bash
claude --plugin-dir .claude/plugins/e2e -p "/e2e:triage-ci <run_url> --agent <agent>"
```

8. Capture output to files for artifact upload
9. Post a Slack thread reply with a concise summary and a link to the triage workflow run
10. Upload triage artifacts regardless of success or failure

### Agent Fan-Out

If the alert has multiple failed agents, the workflow should fan out one matrix job per failed agent. This keeps results isolated and simplifies failure attribution in Slack and in GitHub artifacts.

### Concurrency

Use concurrency keyed by CI `run_id` or Slack thread timestamp so repeated `triage e2e` replies do not start duplicate work for the same failure thread.

## Invocation Model

This design intentionally uses the existing CI-run path in [`.claude/skills/e2e/triage-ci.md`](../../.claude/skills/e2e/triage-ci.md):

- The workflow passes the original GitHub Actions run URL to `/e2e:triage-ci`
- The skill downloads artifacts via [`scripts/download-e2e-artifacts.sh`](../../scripts/download-e2e-artifacts.sh)
- The triage workflow analyzes the failed run's artifacts instead of starting fresh E2E reruns

That keeps cost and runtime bounded for the first version.

If local rerun verification is later required for Slack-triggered triage, that should be added as a deliberate extension to the workflow and possibly to the skill behavior for CI-driven contexts.

## Slack Responses

Recommended thread messages:

- Start:
  - `Starting E2E triage for cursor-cli from CI run <url>.`
- Success:
  - Short classification summary per agent and a link to the GitHub triage workflow
- Failure:
  - Short failure reason and a link to the GitHub triage workflow

Slack replies should stay short. The full triage report belongs in workflow logs and uploaded artifacts.

## Error Handling

### Slack App

- Ignore non-thread replies
- Ignore messages whose normalized text is not exactly `triage e2e`
- Refuse to trigger if the parent message is not recognized as an E2E failure alert from this repository
- Reply in-thread with a short failure message if dispatch fails

### GitHub Workflow

- Fail fast on malformed dispatch payloads
- Fail with a clear Slack reply if checkout or Claude setup fails
- Fail with a clear Slack reply if CI artifact download fails
- Always upload raw triage output as artifacts

## Security

The Slack app should use a GitHub token scoped only to dispatch workflows on this repository.

The workflow should:

- Use the minimum required GitHub permissions
- Store Claude authentication in GitHub Actions secrets
- Avoid echoing secrets or full auth state into logs

The Slack app should validate Slack request signatures before processing events.

## Testing Strategy

### Slack App

- Unit test normalization for exact-match `triage e2e`
- Unit test parent-message validation
- Unit test extraction of `run_url`, `sha`, and `failed_agents`
- Unit test dispatch payload construction

### GitHub Workflow

- Add `workflow_dispatch` inputs mirroring the dispatch payload for manual testing
- Smoke test against a known failed E2E run URL
- Verify success path posts to Slack thread
- Verify invalid payload path exits early and reports clearly

### Non-Goals for Testing

- Do not run real E2E reruns as part of this workflow
- Do not test code-fixing behavior in this first version

## Recommended Implementation Order

1. Add the new GitHub Actions workflow with manual `workflow_dispatch`
2. Prove the workflow can run `/e2e:triage-ci` against a known failed CI run URL
3. Add Slack thread notification hooks for started/succeeded/failed states
4. Build the Slack app that validates the thread reply and sends `repository_dispatch`
5. Tighten the original E2E Slack alert format if parsing proves brittle

## Open Decisions Resolved

- Trigger phrase: exact match `triage e2e`
- Execution environment: GitHub Actions
- Triage source of truth: [`.claude/skills/e2e/triage-ci.md`](../../.claude/skills/e2e/triage-ci.md)
- Invocation surface: [`.claude/plugins/e2e/commands/triage-ci.md`](../../.claude/plugins/e2e/commands/triage-ci.md)
- Initial scope: artifact-based triage of an existing failed CI run, not automatic fixing
