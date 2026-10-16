# Renovate demo experiment

Scope: **air-hand-org/jobdeck-demo only**. This change does not configure Renovate or review bypass for jobdeck or the other services. Do not merge/apply or install an App as part of preparing these PRs.

## Policy

- The existing `main` ruleset is unchanged: PR required (zero approvals), strict `check-result` from GitHub Actions, no force pushes/deletion, **no bypass actors**.
- A separate `main-review-approval` ruleset requires one approval. Only `air-hand` (numeric user ID 45233435) and the Renovate App (resolved by slug `renovate`, public App ID 2740) may bypass this ruleset, only through PRs. Hermes is not allowed to bypass. No broad repository/admin role grants are used.
- `air-hand` was confirmed to have repository admin access; user IDs are not team IDs. Current GitHub REST documentation and pinned integrations/github v6.13.0 source both explicitly support `User`. This is a documented actor, not an invented actor type. No API ruleset write has been attempted, so runtime acceptance still needs the approved deployment.
- Repository auto-merge and squash merge are already enabled. Companion `jobdeck-demo/renovate.json` enables PR-based GitHub platform auto-merge only for minor/patch updates. Major, digest, pin, lock-file maintenance and vulnerability updates are not opted in. No approval helper bots, `autoApprove`, branch automerge, or custom merge workflow.

## Local validation

Use the locked providers, with backend disabled; do not run apply:

```sh
tofu fmt -check -recursive iac/bootstrap
tofu -chdir=iac/bootstrap init -backend=false -lockfile=readonly
tofu -chdir=iac/bootstrap validate -no-color
uv run --no-project --with 'python-hcl2<8' python scripts/verify-renovate-demo.py ../jobdeck-demo
npm exec --yes --package=renovate@44.164.10 -- renovate-config-validator --strict ../jobdeck-demo/renovate.json
```

Inspect the diff to confirm only the demo ruleset changes. Validation verifies schema, not GitHub authorization or auto-merge behavior. Renovate validation was successful under Node 26.5.1 with an engine warning (Renovate requests Node ^24.11.0); repeat under supported Node 24 in CI/operator validation if desired.

## Deferred live acceptance tests (require separate approval)

First review both PRs, install/authorize Renovate for **jobdeck-demo only** if needed, and review the IaC plan: expected new approval ruleset only, no weakening of the original `main` ruleset. Stop if the plan changes unrelated resources. Keep the normal required-check gate intact. Confirm inherited organization rules do not change the expected result.

| Actor / case | Expected result |
| --- | --- |
| air-hand PR, no approval, required check failing/pending/missing | Merge blocked |
| air-hand PR, no approval, check passes and branch current | Explicit PR-only approval bypass available; no direct push bypass |
| Renovate minor/patch PR, no approval, required check failing/pending/missing | Auto-merge must not complete |
| Renovate minor/patch PR, no approval, check passes/current | Observe whether standard GitHub auto-merge recognizes Renovate's review bypass; **unverified hypothesis**, not a guaranteed result |
| Renovate major PR, check passes | No automatic merge requested by this config; human review/decision required operationally (Renovate's actor bypass is not version-specific) |
| Hermes App PR, no approval, check passes | Approval still required; Hermes has no bypass |
| Any actor attempts direct default-branch push / force push / deletion | Blocked by unchanged baseline ruleset |

Record PR URL, head SHA, auto-merge `enabledBy`, merge state, ruleset evaluation/bypass audit actor and check-run conclusion for every case. A successful manual bypass does **not** prove platform auto-merge works. If standard auto-merge waits for approval, report the result and stop; do not silently add an approval bot, change to Renovate-managed merging, or relax CI. Actual merge tests are outside current authorization.

Existing `check-result` is skipped on successful upstream CI, which GitHub accepts as a satisfied check. Its current workflow only explicitly fails on upstream `failure`, not `cancelled`; this experiment does not change that pre-existing behavior. Test cancellation separately before claiming the aggregate gate is fail-closed for every upstream outcome.

Rollback after an approved deployment: disable minor/patch `automerge` and cancel outstanding auto-merge requests first, then remove only `demo_review_approval` through a reviewed IaC change. Never add baseline bypasses. Existing PRs may retain auto-merge requests after config changes.

## Primary references (checked 2026-10-16)

- GitHub repository rules API, actor enum and PR-only bypass: https://docs.github.com/en/rest/repos/rules#create-a-repository-ruleset
- Pinned provider actor validation: https://github.com/integrations/terraform-provider-github/blob/v6.13.0/github/resource_github_repository_ruleset.go
- Renovate platform auto-merge and required checks: https://docs.renovatebot.com/configuration-options/#platformautomerge
- Renovate review-bypass guidance (not proof of this ruleset/native-auto-merge combination): https://docs.renovatebot.com/key-concepts/automerge/#required-pull-request-reviews
