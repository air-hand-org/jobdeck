# Local Development

This document describes the minimal local setup for JobDeck.

## Prerequisites

Install these tools before running the project:

- Aspire CLI
- mise
- aqua

Project tool versions are managed by `mise.toml` and `aqua.yaml`.

## First Setup

Enable [mise shell integration](https://mise.jdx.dev/cli/activate.html)
so entering this repository loads its tool PATH and `mise.toml` environment
variables. The local commands below assume this integration is active.

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
make run
```

This starts the AppHost and the local resources it defines, including the UI, job API, worker, and PostgreSQL.

The Aspire dashboard URL is printed by the Aspire CLI.

## Local Service Checkouts

By default, Aspire runs all services from the versioned GHCR images declared in
`environments/production.yaml`. To run a service from source, clone its repository
into the corresponding path below and set `USE_LOCAL=true`:

```sh
make clone-service-repo
USE_LOCAL=true DEBUG=true make run
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

Local Go services use the Go executable available on the AppHost's PATH; their
own `mise.toml` files are not automatically loaded. With `GOTOOLCHAIN=auto` (the
Go default), Go can select a newer toolchain when a service's `go.mod` requires it
through its `go` or `toolchain` directive. Keep this in mind when updating Go
versions across repositories.

The UI also inherits the root .NET SDK environment. Unlike Go, .NET does not
automatically download a required SDK; a newer UI target framework or SDK
requirement must be supported by the SDKs installed for JobDeck. Keep the .NET
versions in the root and UI `mise.toml` files aligned when upgrading the UI or
Aspire.

## Test Local Sources

After cloning all services, run the full E2E suite:

```sh
USE_LOCAL=true make cs-test
```

Aspire builds the Go services and UI when starting them; no separate service
build is required. The E2E fixture verifies that
all five services use local sources and that the worker's forward proxy is built
from its checkout; missing checkouts fail instead of silently passing via images.
The suite checks readiness, login, and job execution. Browser and development
certificate setup uses the existing `cs-test` preparation, which may require sudo.

CI runs the container suite followed by this full local-source suite in the same
test job, reusing Docker images and NuGet and Go caches. Logs and traces are
uploaded separately for each mode. CI supplies authentication inputs before
setup-tools loads the shared `mise.toml` environment.

### E2E startup and runner capacity

The fixture shares one five-minute deadline across AppHost creation, build,
start, and the UI Healthy wait. E2E continues to pass `UseVolumes=false`.
pgAdmin remains enabled in both suites so capacity diagnostics can observe the
existing composition before any resource reduction.

Aspire 13.6.1's [testing factory](https://github.com/microsoft/aspire/blob/v13.6.1/src/Aspire.Hosting.Testing/DistributedApplicationFactory.cs)
already sets `DisableDashboard=true` and enables resource logging. Therefore
these E2Es do not start the dashboard or create its persistent SQLite state;
no extra dashboard persistence override is needed. This is separate from DB
volumes and resource lifetimes. The AppHost has no persistent resource lifetime
opt-ins, and the testing factory waits for resource cleanup.
For ordinary AppHost runs, [dashboard persistence](https://aspire.dev/dashboard/configuration/#data-persistence)
defaults to `Run` under `${ASPIRE_HOME:-~/.aspire}/dashboard` (or
`ASPIRE_DASHBOARD_DATA_DIRECTORY`). `ASPIRE_DASHBOARD_PERSISTENCE_MODE=None`
uses a temporary SQLite database instead; it does not eliminate disk writes.

To compare runner usage, rerun CI with debug logging enabled (`runner.debug=1`).
Read-only diagnostics run after dependency cache restore, before container E2E,
after container E2E / before local E2E, and after local E2E. The post-test
snapshots also run on failure, before artifact upload and Docker cleanup.
They include filesystem bytes/inodes, detailed Docker usage, selected Go/NuGet
cache paths, service checkouts and build outputs, and Aspire home/session state.
Only sizes and paths are printed; configuration files and credentials are not.
Commands are bounded and unavailable diagnostics do not fail the test job.
Normal CI does not run these detailed snapshots. Existing resource-log artifacts
remain available on failure, including startup timeouts, and debug runs.

Keep Docker cleanup after both suites so local-source E2E can reuse images.
Do not clear dependency caches or move cleanup between suites until the capacity
snapshots identify which interval and storage area grows. The observed
`no space left on device` does not by itself identify the occupying component
or distinguish byte exhaustion from inode exhaustion.

To investigate NuGet authentication with an empty cache, set `NUGET_PACKAGES`
and `NUGET_HTTP_CACHE_PATH` to directories under a temporary directory before
running the command. This leaves your normal caches intact.

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
