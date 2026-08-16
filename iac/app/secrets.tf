locals {
  # A write-only argument does not reveal value changes to OpenTofu. Bump the
  # corresponding number whenever the encrypted source rotates that value.
  secret_data_wo_versions = {
    "auth-jwt-private-key"          = 1
    "auth-jwt-public-key"           = 1
    "job-worker-client-secret"      = 1
    "job-worker-client-secret-hash" = 1
    "postgres-password"             = 1
    "test-user-client-secret"       = 1
    "test-user-client-secret-hash"  = 1
  }

  # `nobbs/sops` implements this as a provider function rather than a data
  # source, so the decrypted result is not retained in state. `sensitive`
  # additionally prevents an accidental plan rendering from exposing values.
  decrypted_secret_values = sensitive(
    provider::sops::file("${path.module}/../secrets/production.enc.yaml", "yaml").data.secret_values
  )
}

resource "google_secret_manager_secret_version" "application" {
  for_each = local.secret_data_wo_versions

  secret                 = local.application_secrets[each.key]
  secret_data_wo         = local.decrypted_secret_values[each.key]
  secret_data_wo_version = each.value

  lifecycle {
    precondition {
      # Key names are deliberately not secret; only make the value map
      # non-sensitive for this schema check.
      condition = (
        length(setsubtract(
          toset(nonsensitive(keys(local.decrypted_secret_values))),
          toset(keys(local.secret_data_wo_versions)),
        )) == 0 &&
        length(setsubtract(
          toset(keys(local.secret_data_wo_versions)),
          toset(nonsensitive(keys(local.decrypted_secret_values))),
        )) == 0
      )
      error_message = "The encrypted production secret source must match the bootstrap secret contract exactly."
    }
  }
}
