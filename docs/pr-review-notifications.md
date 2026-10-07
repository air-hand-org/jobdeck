# Hermes review requests

JobDeck uses Hermes to help review changes before they reach `main`. Review
requests are sent automatically so starting a review does not depend on someone
remembering to ask, and an updated PR can be reviewed against its new revision.

The GitHub Slack integration does not deliver its PR mentions in a form Hermes
recognizes. [The review request workflow](../.github/workflows/request-pr-review.yaml)
sends an explicit Slack message containing a Hermes mention, the PR URL, and the
requested head SHA. This gives Hermes a clear request and identifies the change
to review. Shared notification code lives in jobdeck-actions.

Requests cover non-draft PRs targeting `main` from branches in this repository,
including image promotion PRs. Opening, becoming ready, reopening, or updating a
PR's head requests review. Fork PRs are excluded. This scope matches the normal
CI and review path into `main`.

The workflow starts a review; notification success alone does not establish that
Hermes completed it or that the change is safe to merge. Check the review result
on the PR alongside CI and human review. Notification failures are visible in
Actions, and retries can produce duplicate requests.

## Connecting Hermes

Use a notifier Slack App separate from Hermes. Hermes excludes its own posts,
so its token cannot be used to send these requests. The notifier needs
`chat:write` and membership in the notification channel. Hermes must also be in
that channel and accept bot mentions with
`platforms.slack.extra.allow_bots: mentions`.

Configure these Repository Secrets:

| Secret | Value |
| --- | --- |
| `SLACK_REVIEW_NOTIFIER_BOT_TOKEN` | Notifier App's Bot User OAuth Token |
| `SLACK_REVIEW_CHANNEL_ID` | Channel receiving the review requests |
| `SLACK_REVIEWER_USER_ID` | Hermes's Slack user ID |

Tokens and workspace identifiers stay out of public source files and logs.
Confirm the connection with a real PR: a request must reach Slack, and Hermes
must fetch the PR and start reviewing. A successful notification job confirms
only the first part.

For delivery, retry, and duplicate handling, see the
[jobdeck-actions notification documentation](https://github.com/air-hand-org/jobdeck-actions/blob/main/docs/slack-pr-review-request.md).
