# Bootstrap infrastructure

This root is run by the manually-created `jobdeck-bootstrap` identity through
GitHub Actions. The state bucket, bootstrap service account, and its GitHub
Workload Identity Federation binding are the intentionally manual trust root.
The existing `github` pool/provider also remain part of that manual trust root.
Service-specific image-publishing pools are managed by this root instead.

Initialize the backend with:

```bash
tofu init \
  -backend-config='bucket=airhand-jobdeck-bucket' \
  -backend-config='prefix=bootstrap'
```

The bootstrap state intentionally contains no secret values and is not
OpenTofu-encrypted: the KMS keys used by the `app` root are created by this
root, so they cannot encrypt the state required to create them. GCS encryption,
uniform bucket-level access, public access prevention, and object versioning
protect the bucket itself.

## GitHub repository governance

This root manages the rulesets, GitHub Environments, deployment ref policies,
and non-secret Actions variables for repositories in the JobDeck platform. The
GitHub App private key is deliberately not an OpenTofu resource or state value.
It is a manually registered Environment secret in `air-hand-org/jobdeck`:

| Environment | App permission | Purpose |
| --- | --- | --- |
| `plan` | read-only | Refresh and plan governance changes on internal PRs. |
| `bootstrap` | read/write | Apply governance changes after a protected `main` push. |

Each Environment provides `GH_IAC_APP_CLIENT_ID` and
`GH_IAC_APP_PRIVATE_KEY`. The reusable OpenTofu workflow exchanges them for a
short-lived installation token with `actions/create-github-app-token`, and
passes that token to the GitHub provider as `GITHUB_TOKEN`. The app need only
be installed in repositories managed by this root. It does not need an
installation in `jobdeck` merely because the workflow executes there.

For the current `jobdeck-auth` import, an installation restricted to that
repository is sufficient. Before this root creates a new repository such as
`jobdeck-job-api`, change both Apps' organization installations to **All
repositories**. The workflow intentionally does not name a repository when it
creates the token: the `repositories` input accepts only repositories that
already exist, so it cannot grant access to a repository created by the same
OpenTofu run.

Both roots post their plan and apply results through `tfcmt`. Plan results are
updated on internal pull requests. For an apply, the workflow finds a merged
pull request associated with the triggering commit and posts there; it skips
the comment if none is found. The comment and label token is GitHub Actions'
scoped `GITHUB_TOKEN`; the nested OpenTofu command uses the read-only or
write-capable App token from the selected Environment. This keeps GitHub
comment permission separate from GitHub governance permission.

Repository resources use `prevent_destroy`. Before retiring GCP bootstrap
infrastructure, move repository state to another root (or intentionally remove
it from state while retaining the remote repository); do not run a blanket
destroy.

`tofu-state` encrypts the `app` root's OpenTofu state and plans. `sops` encrypts
the random data keys used by version-controlled SOPS secret files. They are
separate keys so that their IAM, rotation, and incident response scopes remain
independent.

After apply, delegate `jobdeck.app.air-hand.com` from the Route 53
`air-hand.com` zone to the four values in `app_dns_nameservers`.

## Auth image publisher

`auth-publisher.tf` creates a separate publishing identity for
`air-hand-org/jobdeck-auth`:

- WIF pool `jobdeck-auth-publish` and GitHub OIDC provider `github`.
- Service account `jobdeck-auth-publisher`.
- `roles/artifactregistry.writer` on the existing `jobdeck-auth` repository only.
  This grants artifact read/write access, not project-wide deployment rights.
- `roles/iam.workloadIdentityUser` on that service account for the dedicated
  pool's `publish` environment identities.

The provider requires repository ID `1360104416`, owner ID `139471655`, and
repository name `air-hand-org/jobdeck-auth`. Numeric IDs prevent reuse of names
from granting access to a different repository or organization. Jobs without an
environment cannot match the service account binding.

Do not add this repository to the existing `github` pool/provider. The existing
deployer and plan service account bindings select `attribute.environment`
within that pool; broadening its accepted repositories could authorize those
repositories to impersonate the existing identities. A separate provider in
the same pool would not isolate these pool-scoped principal sets.

Before applying, the manually managed `jobdeck-bootstrap` identity needs
`roles/iam.workloadIdentityPoolAdmin`, or equivalent custom permissions, on the
project to manage the new pool/provider. Grant this through the manual bootstrap
process; this root does not grant permissions to its own bootstrap identity.
See [Google's deployment pipeline WIF guide](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines).

After the bootstrap apply succeeds, create a GitHub Environment named `publish`
in `air-hand-org/jobdeck-auth` and set these environment variables:

| Variable | Value |
| --- | --- |
| `GCP_PROJECT_ID` | `airhand-jobdeck` |
| `GCP_SERVICE_ACCOUNT` | Output `auth_publisher_service_account` |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Output `auth_publisher_workload_identity_provider` |

Restrict the environment's deployment branches/tags to the intended release
refs (for example, release tags matching `v*`). The environment name alone does
not restrict the Git ref or workflow. The image-publishing job must declare
`environment: publish`, `contents: read`, and `id-token: write`, then authenticate
with `google-github-actions/auth` using these variables and
`token_format: access_token`. Use the resulting short-lived token with Docker
login (`username: oauth2accesstoken`) to push to:

```text
asia-northeast1-docker.pkg.dev/airhand-jobdeck/jobdeck-auth/auth
```

Use unique release tags: this registry has immutable tags enabled. No service
account key is needed. This change provisions the publishing identity; the new
repository's GitHub Environment and publishing workflow are configured
separately. Existing monorepo publishing access remains during the transition
and can be removed when image publication has moved successfully.
