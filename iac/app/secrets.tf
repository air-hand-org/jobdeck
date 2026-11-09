locals {
  # A write-only argument does not reveal value changes to OpenTofu. Bump the
  # corresponding number whenever the encrypted source rotates that value.
  secret_data_wo_versions = {
    "auth-jwt-private-key"          = "1"
    "auth-jwt-public-key"           = "1"
    "job-worker-client-secret"      = "1"
    "job-worker-client-secret-hash" = "1"
    "postgres-password"             = "1"
    "test-user-client-secret"       = "1"
    "test-user-client-secret-hash"  = "1"
  }

}

variable "decrypt_secrets" {
  description = "Decrypt the production source and manage Secret Manager versions."
  type        = bool
  default     = false
}

locals {
  # `nobbs/sops` implements this as a provider function rather than a data
  # source, so the decrypted result is not retained in state. Keep local lint
  # independent of KMS credentials; CI app plan/apply enables this explicitly.
  decrypted_secret_values = var.decrypt_secrets ? sensitive(
    provider::sops::file("${path.module}/../secrets/production.enc.yaml", "yaml").data.secret_values
  ) : {}
}

resource "google_secret_manager_secret_version" "application" {
  for_each = var.decrypt_secrets ? local.secret_data_wo_versions : {}

  secret                 = local.application_secrets[each.key]
  secret_data_wo         = local.decrypted_secret_values[each.key]
  secret_data_wo_version = each.value

  lifecycle {
    precondition {
      # Key names are deliberately not secret; only make the value map
      # non-sensitive for this schema check.
      condition = alltrue([
        for name in keys(local.secret_data_wo_versions) :
        contains(nonsensitive(keys(local.decrypted_secret_values)), name)
      ])
      error_message = "The encrypted production secret source must include every bootstrap secret value."
    }
  }
}
