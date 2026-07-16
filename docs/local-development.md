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

## Common Commands

```sh
make build
make test
make lint
make fmt
```

The Makefile is the source of truth for local development commands.

