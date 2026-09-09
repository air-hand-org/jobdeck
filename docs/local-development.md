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

## Auth Source Debugging

By default, Aspire runs auth from the versioned GHCR image declared in
`environments/production.yaml`. To work on auth together with this repository,
clone `jobdeck-auth` into the fixed local checkout path:

```sh
git clone git@github.com:air-hand-org/jobdeck-auth.git services/auth
USE_LOCAL=true DEBUG=true make run
```

`USE_LOCAL=true` selects local source mode. It requires `services/auth/go.mod`,
then starts the auth server and its database migrator with `AddGoApp`;
`DEBUG=true` also enables the auth Delve server on port 2347. Without
`USE_LOCAL=true`, Aspire uses the GHCR image even if the checkout exists. The
checkout is intentionally not part of this repository's `go.work` and is
ignored by its Git worktree. CI does not set `USE_LOCAL`, so it continues to
use the GHCR image.

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
