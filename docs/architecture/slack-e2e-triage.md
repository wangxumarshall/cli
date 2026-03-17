# Slack-Triggered E2E Triage

This flow lets a human reply `triage e2e` in the thread of an E2E failure alert and have GitHub Actions run the existing triage workflow.

## Flow

1. `.github/workflows/e2e.yml` posts the failure alert and includes machine-readable metadata.
2. `cmd/e2e-triage-dispatch` listens for Slack thread replies, validates the reply text, fetches the parent alert, and dispatches GitHub.
3. `.github/workflows/e2e-triage.yml` checks out the failed SHA, runs the Claude triage skill, and posts results back to the Slack thread.

The trigger is the exact normalized text `triage e2e`.

## Slack Setup

Slack app requirements:

- Event subscription for `message.channels` so the app receives public channel thread replies
- `channels:history` so the app can read the parent E2E alert message
- `chat:write` so the app can post status updates back into the thread

If you want private-channel support, add the equivalent `groups:history` event and scope as well.

## GitHub And Runtime Config

`cmd/e2e-triage-dispatch` uses these environment variables:

- `SLACK_SIGNING_SECRET`
- `SLACK_BOT_TOKEN`
- `GITHUB_TOKEN`
- `ALLOWED_REPOSITORY` or `GITHUB_REPOSITORY`
- `ADDR` optional, defaults to `:8080`
- `GITHUB_EVENT_TYPE` optional, defaults to `slack_e2e_triage_requested`
- `SLACK_API_BASE_URL` optional, defaults to `https://slack.com/api`
- `GITHUB_API_BASE_URL` optional, defaults to `https://api.github.com`
- `SLACK_REQUEST_TOLERANCE` optional, defaults to `5m`

The GitHub Actions workflow uses these secrets:

- `ANTHROPIC_API_KEY` for Claude triage
- `SLACK_BOT_TOKEN` for start and completion replies
- `GITHUB_TOKEN` for the repository dispatch and repository checkout

## Manual Fallback

If Slack dispatch is unavailable, you can run `.github/workflows/e2e-triage.yml` manually with `workflow_dispatch`.

Required inputs:

- `run_url`
- `sha`
- `failed_agents`

Optional inputs:

- `slack_channel`
- `slack_thread_ts`

This is the fallback path for ad hoc triage when you already have the failed run URL and commit SHA.
