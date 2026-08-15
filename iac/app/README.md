# JobDeck application infrastructure

This OpenTofu root owns application-lifecycle resources: Cloud Run services and
Jobs, networking, the external load balancer, and records in the delegated Cloud
DNS zone. It consumes stable, non-secret bootstrap outputs from the `bootstrap`
root.

The backend is configured at initialization time:

```sh
tofu init \
  -backend-config='bucket=airhand-jobdeck-bucket' \
  -backend-config='prefix=app'
```

State and saved plans are encrypted with bootstrap's `tofu-state` GCP KMS key.
The bootstrap remote state is deliberately read as unencrypted OpenTofu state:
that root creates the KMS key and must not contain secret values.

`iac/app` has no deployable resources yet. Subsequent changes add Cloud Run,
load balancing, and DNS while preserving this state and remote-state contract.
