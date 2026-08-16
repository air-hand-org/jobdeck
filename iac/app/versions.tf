terraform {
  required_version = ">= 1.10.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "= 7.43.0"
    }
    sops = {
      source  = "nobbs/sops"
      version = "= 0.3.3"
    }
  }

  backend "gcs" {}

  # bootstrap creates and protects this KMS key. Keep these provider and
  # method names stable: they are recorded in encrypted state metadata.
  encryption {
    key_provider "gcp_kms" "tofu_state" {
      kms_encryption_key = "projects/${var.project_id}/locations/${var.region}/keyRings/jobdeck/cryptoKeys/tofu-state"
      key_length         = 32
    }

    method "aes_gcm" "tofu_state" {
      keys = key_provider.gcp_kms.tofu_state
    }

    state {
      method = method.aes_gcm.tofu_state
    }

    plan {
      method = method.aes_gcm.tofu_state
    }

    # bootstrap state intentionally predates this key and remains unencrypted
    # at the OpenTofu layer. It contains no secret values.
    method "unencrypted" "bootstrap" {}

    remote_state_data_sources {
      default {
        method = method.unencrypted.bootstrap
      }
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# The provider exposes only functions. In contrast to the older data-source
# providers, its decrypted values are not recorded in OpenTofu state.
provider "sops" {}
