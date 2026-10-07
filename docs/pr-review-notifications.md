# PR review notifications

`.github/workflows/request-pr-review.yaml` calls the reusable Slack notification
workflow in `air-hand-org/jobdeck-actions` v1.5.0, pinned to release commit
`19271b7a2a9332bc2066ae136c0b2dc1d8751747`. It requests Hermes reviews for all
open, non-draft PRs from branches in this repository, including image promotion
PRs. Fork PRs are excluded because `pull_request` does not receive their secrets.

## Setup

Set the jobdeck repository Actions secret `SLACK_REVIEW_NOTIFIER_BOT_TOKEN` to
the **Bot User OAuth Token** (`xoxb-...`) of a notification Slack App separate
from Hermes. The App needs the `chat:write` bot scope and membership in the
notification channel. Also set `SLACK_REVIEW_CHANNEL_ID` to that channel's ID
and `SLACK_REVIEWER_USER_ID` to Hermes's Slack user ID. These real workspace
identifiers are kept in Secrets so source files and Actions logs do not publish
them.

Hermes must be in the channel and accept bot mentions via
`platforms.slack.extra.allow_bots: mentions`. Its own token must not be used
to send the request, because Hermes excludes its own messages.

The workflow uses `pull_request`; the caller controls which event types request
a review. It passes the three Repository Secrets explicitly to the released
workflow without inheriting other credentials. When updating jobdeck-actions,
update the pinned commit and version comment together.

## Events and message

| Event | Behaviour |
| --- | --- |
| `opened` | Request review if the new PR is not a draft. |
| `ready_for_review` | Request review when a draft becomes ready. |
| `reopened` | Request review again if the reopened PR is not a draft. |
| `synchronize` | Request review of the updated head if the PR is not a draft. |
| Draft PR, title/body edits, labels | No review request. |
| Fork PR | No review request. |

The reusable workflow checks the PR again through GitHub's API before sending.
A closed PR, draft PR, or event whose head SHA is no longer current is skipped.
Slack's top-level `text` contains the mention, PR URL, head SHA, triggering event
and Actions run reference. PR titles, bodies and code are not executed or
included in the request.

## Failures and duplicate requests

Requests for the same PR are serialized without cancelling an in-flight send.
GitHub can replace pending concurrency runs; a request for every intermediate
head is not guaranteed. A head can also change after the final API check, so
Hermes must re-fetch the PR and validate the head before reviewing or publishing
a result.

Reopening, becoming ready again, or manually rerunning a successful workflow can
send another request for the same head. Exactly-once delivery is not guaranteed;
Hermes should identify duplicate work by repository, PR number and head SHA.

Slack delivery uses the official `slackapi/slack-github-action` with its default
retry policy. Errors remaining after retry, including Slack `ok: false`, make
the Actions job fail. Check the workflow result and App configuration, then
rerun the failed job if needed. A timeout can occur after Slack accepted the
request, so automatic retry or manual rerun can produce duplicate notifications.
Inspect the channel before rerunning when delivery is uncertain.

A successful workflow means Slack accepted the notification, not that Hermes
finished or even started a review. This notification is separate from ordinary
CI and does not implement a required review check or release gate.

## Verification after rollout

Use a real non-draft same-repository PR containing the caller workflow. Confirm
the Actions job succeeds, the channel receives a top-level Hermes mention and
the correct PR URL/head SHA, and Hermes fetches the PR and starts reviewing.
Also confirm draft creation is silent, becoming ready triggers a request,
head updates trigger a new request, and reopening triggers a request.

The local test checks only Slack payload generation, without stubs or mock
servers. It does not verify Slack App installation, Hermes event reception or
the end-to-end review.
