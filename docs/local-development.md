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

Create an ignored `.env.local` in the repository root with a GitHub classic PAT
that has `read:packages` access to the private GHCR images and NuGet packages:

```dotenv
GITHUB_PACKAGES_TOKEN=<your-classic-PAT>
```

Authenticate the GitHub CLI with `gh auth login` and run
`gh auth setup-git --hostname github.com` to enable authenticated Git downloads
of the Go services' private `api-contracts` modules. `mise.toml` sets `GOPRIVATE`
for those modules and uses the active GitHub CLI
account as `GITHUB_ACTOR`, unless that variable is already set. The same
`GITHUB_PACKAGES_TOKEN` supplies both GHCR login and the UI's `github` NuGet source
credentials. Do not print these environment variables or commit the token.

Start the local environment through Aspire:

```sh
mise exec -- make run
```

This starts the AppHost and the local resources it defines, including the UI, job API, worker, and PostgreSQL.

The Aspire dashboard URL is printed by the Aspire CLI.

## Local Service Checkouts

By default, Aspire runs all services from the versioned GHCR images declared in
`environments/production.yaml`. To run a service from source, clone its repository
into the corresponding path below and set `USE_LOCAL=true`:

```sh
mise exec -- make clone-service-repo
USE_LOCAL=true DEBUG=true mise exec -- make run
```

`clone-service-repo` clones missing service repositories in parallel at the
`v<version>` tags declared in `environments/production.yaml`. Existing paths are
kept without fetching, switching branches, or changing local edits. Use Git in
each checkout to select a development branch when needed. Your GitHub CLI account
must have read access to the private service repositories.

`USE_LOCAL=true` selects local source mode independently for each checkout. It
requires the respective `go.mod` file for auth, job-api, job-worker, and demo,
or `services/ui/src/ui.csproj` for UI. The Go services start with `AddGoApp`, UI
starts as a local .NET project, and the job-worker forward proxy is built from
its local checkout. The TLS reverse proxies remain YARP resources.
`DEBUG=true` also enables the auth and job-worker Delve
servers on ports 2347 and 2346. A missing checkout falls back to the corresponding
versioned container image, even when other services run from source. The checkouts
are intentionally ignored by this repository's Git worktree.

## Test Local Sources

After cloning all services, run the full E2E suite with an isolated, initially
empty NuGet package and HTTP cache without clearing your normal cache:

```sh
nuget_cache_root=$(mktemp -d)
export NUGET_PACKAGES="$nuget_cache_root/packages"
export NUGET_HTTP_CACHE_PATH="$nuget_cache_root/http-cache"
mise exec -- dotnet build services/ui/src/ui.csproj
USE_LOCAL=true mise exec -- make cs-test
```

The UI project is dynamically added by the AppHost and is outside the system
solution, so build it explicitly before testing. The E2E fixture verifies that
all five services use local sources and that the worker's forward proxy is built
from its checkout; missing checkouts fail instead of silently passing via images.
The suite checks readiness, login, and job execution. Browser and development
certificate setup uses the existing `cs-test` preparation, which may require sudo.

CI keeps the existing container E2E job and runs this full local-source suite in
a separate job with fresh NuGet caches. CI supplies authentication inputs before
setup-tools loads the shared `mise.toml` environment.

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
