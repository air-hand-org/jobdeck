# JobDeck application infrastructure

This OpenTofu root owns application-lifecycle resources: Cloud Run services,
networking, Pub/Sub delivery, the external load balancer, and records in the
delegated Cloud DNS zone. It consumes stable, non-secret bootstrap outputs from
the `bootstrap` root.

The application consists of `ui`, `demo`, `auth`, `job-api`, and `job-worker`.
Auth and job-api each run an ephemeral PostgreSQL sidecar. Job-api writes
execution requests to its outbox and publishes them to Pub/Sub; a push
subscription invokes job-worker.

Every Cloud Run service uses Direct VPC egress with `PRIVATE_RANGES_ONLY`.
The `run.app` private Cloud DNS zone resolves service URLs through Private
Google Access, so calls to internal auth and job-api services retain Cloud Run's
internal-ingress classification. Other destinations keep Cloud Run's managed
public egress: the worker's Squid sidecar allowlists only
`demo.${var.public_domain}:443`, so no Cloud NAT is needed for demo requests.

Every service revision uses the same immutable `image_tag`. Pull-request plans
use the harmless placeholder `v0.0.0`; release deployment passes the published
release tag only after the image-push job completes.

OpenTofu decrypts `iac/secrets/production.enc.yaml` through a provider function
and adds Secret Manager versions through write-only provider arguments. See
[`../secrets/README.md`](../secrets/README.md) for the one-time initialization
and rotation procedure.

The backend is configured at initialization time:

```sh
tofu init \
  -backend-config='bucket=airhand-jobdeck-bucket' \
  -backend-config='prefix=app'
```

State and saved plans are encrypted with bootstrap's `tofu-state` GCP KMS key.
The bootstrap remote state is deliberately read as unencrypted OpenTofu state:
that root creates the KMS key and must not contain secret values.

The public regional HTTPS load balancer exposes only the UI and demo hosts and
is protected by the baseline regional Cloud Armor policy.
