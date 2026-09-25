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

## Local Service Checkouts

By default, Aspire runs all services from the versioned GHCR images declared in
`environments/production.yaml`. To run a service from source, clone its repository
into the corresponding path below and set `USE_LOCAL=true`:

```sh
git clone git@github.com:air-hand-org/jobdeck-auth.git services/auth
git clone git@github.com:air-hand-org/jobdeck-job-api.git services/job-api
git clone git@github.com:air-hand-org/jobdeck-job-worker.git services/job-worker
git clone git@github.com:air-hand-org/jobdeck-ui.git services/ui
git clone git@github.com:air-hand-org/jobdeck-demo.git services/demo
USE_LOCAL=true DEBUG=true make run
```

`USE_LOCAL=true` selects local source mode independently for each checkout. It
It requires the respective `go.mod` file for auth, job-api, job-worker, and demo,
or `services/ui/src/ui.csproj` for UI. The Go services start with `AddGoApp`, UI
starts as a local .NET project, and the auth / job-worker proxies are built from
their local checkout. `DEBUG=true` also enables the auth and job-worker Delve
servers on ports 2347 and 2346. A missing checkout falls back to the corresponding
versioned container image, even when other services run from source. The checkouts
are intentionally ignored by this repository's Git worktree. CI does not set
`USE_LOCAL`, so it continues to use container images.

## Containerized Local Runs

The default local composition is image-first, so a system-repository checkout does
not require every service repository. Use a checkout under `services/` together
with `USE_LOCAL=true` when changing that service and needing source-level debugging
or Hot Reload.

## Common Commands

```sh
make build
make test
make lint
make fmt
```

The Makefile is the source of truth for local development commands.
