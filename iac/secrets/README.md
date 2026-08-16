# Production secret source

`production.enc.yaml` is the version-controlled, SOPS-encrypted source for
the Secret Manager values in the bootstrap contract. It can also retain an
operator-only secret such as `test-user-client-secret`; that value is not
injected into Secret Manager until a later bootstrap change. Plaintext files
are ignored by Git.

After bootstrap has created the `jobdeck/sops` KMS key, authenticate locally
to GCP and create the initial source once:

```sh
gcloud auth application-default login
sops iac/secrets/production.enc.yaml
git add iac/secrets/production.enc.yaml
```

Opening the encrypted target itself makes SOPS evaluate the matching
`iac/secrets/*.enc.yaml` creation rule. Use the existing auth `keygen` and
`secrethash` commands to populate the key and credential values; do not reuse
E2E credentials. OpenTofu decrypts the source through the `nobbs/sops` provider
function and supplies the result directly to `secret_data_wo`; no SOPS data
source or plaintext temporary file is used.

To rotate a value, edit the encrypted file with `sops`, then increment that
secret's `secret_data_wo_version` in `iac/app/secrets.tf`. The version number
is required because the provider intentionally cannot diff write-only data.
