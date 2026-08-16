# JobDeck application infrastructure

This OpenTofu root owns application-lifecycle resources: Cloud Run services and
Jobs, networking, the external load balancer, and records in the delegated Cloud
DNS zone. It consumes stable, non-secret bootstrap outputs from the `bootstrap`
root.

The first application slice creates the shared Direct VPC egress network and
the `ui`, `demo`, `auth`, and `job-api` Cloud Run services. Auth and job-api
each run an ephemeral PostgreSQL sidecar. The worker, scheduler, public load
balancer, and DNS records are intentionally added by later slices.

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

Later changes add the worker job, load balancing, and DNS while preserving this
state and remote-state contract.
