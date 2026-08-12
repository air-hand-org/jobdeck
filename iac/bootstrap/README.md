# Bootstrap infrastructure

This root is run by the manually-created `jobdeck-bootstrap` identity through
GitHub Actions. The state bucket, bootstrap service account, and its GitHub
Workload Identity Federation binding are the intentionally manual trust root.

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

`tofu-state` encrypts the `app` root's OpenTofu state and plans. `sops` encrypts
the random data keys used by version-controlled SOPS secret files. They are
separate keys so that their IAM, rotation, and incident response scopes remain
independent.

After apply, delegate `jobdeck.app.air-hand.com` from the Route 53
`air-hand.com` zone to the four values in `app_dns_nameservers`.
