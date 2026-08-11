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
client-side encrypted: the KMS key used by app-state encryption is created by
this root. GCS encryption, uniform bucket-level access, public access
prevention, and object versioning protect the bucket itself.

After apply, delegate `app.air-hand.com` from the Route 53 `air-hand.com` zone
to the four values in `app_dns_nameservers`.
