# Local Development

This document describes the minimal local setup for JobDeck.

## Prerequisites

Install these tools before running the project:

- Aspire CLI
- mise
- aqua

Project tool versions are managed by `mise.toml` and `aqua.yaml`.

## First Setup

Run this once after cloning the repository:

```sh
make setup
```

This installs the tool versions declared for the project.

## Run Locally

Start the local environment through Aspire:

```sh
make run
```

This starts the AppHost and the local resources it defines, including the UI, job API, worker, and PostgreSQL.

The Aspire dashboard URL is printed by the Aspire CLI.

## Auth and Job Worker Source Debugging

By default, Aspire runs auth, job-worker, and its forward-proxy from the versioned
GHCR images declared in `environments/production.yaml`. To work on auth and
job-worker together with this repository, clone them into their fixed local checkout
paths:

```sh
git clone git@github.com:air-hand-org/jobdeck-auth.git services/auth
git clone git@github.com:air-hand-org/jobdeck-job-worker.git services/job-worker
USE_LOCAL=true DEBUG=true make run
```

`USE_LOCAL=true` selects local source mode independently for each checkout. It
requires `services/auth/go.mod` and/or `services/job-worker/go.mod`; auth and
job-worker then start with `AddGoApp`, while the worker's forward-proxy is built
from `services/job-worker/containers/forward-proxy`. `DEBUG=true` also enables
the auth and job-worker Delve servers on ports 2347 and 2346. Without
`USE_LOCAL=true`, or when a checkout is absent, Aspire uses the corresponding GHCR
image even if another local checkout exists. The checkouts are intentionally not
part of this repository's `go.work` and are ignored by its Git worktree. CI does
not set `USE_LOCAL`, so it continues to use GHCR images.

## Containerized Local Runs

Normal local development remains source-first: the AppHost starts Go applications and
the .NET UI directly so that Hot Reload and debugger support, including Delve, stay
fast and straightforward.

The production Dockerfiles are the canonical release-image build contract. A future
AppHost option should allow selected services to run from those Dockerfiles for
production-parity checks, without making container builds the default inner loop.

## Common Commands

```sh
make build
make test
make lint
make fmt
```

The Makefile is the source of truth for local development commands.
