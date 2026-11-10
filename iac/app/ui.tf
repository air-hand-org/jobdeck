resource "google_cloud_run_v2_service" "ui" {
  name                 = "jobdeck-ui"
  location             = var.region
  ingress              = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"
  invoker_iam_disabled = false

  template {
    service_account = local.runtime_service_accounts.ui
    timeout         = "60s"

    scaling {
      # UI sessions are in memory and must not be distributed between instances.
      max_instance_count = 1
    }

    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"

      network_interfaces {
        network    = google_compute_network.jobdeck.id
        subnetwork = google_compute_subnetwork.cloud_run.id
      }
    }

    containers {
      name       = "ui"
      image      = local.images.ui
      depends_on = ["postgres"]

      ports {
        container_port = 8080
      }

      env {
        name  = "AUTH_GRPC_ENDPOINT"
        value = google_cloud_run_v2_service.auth.uri
      }

      env {
        name  = "JOB_API_GRPC_ENDPOINT"
        value = google_cloud_run_v2_service.job_api.uri
      }

      env {
        name  = "POSTGRES_HOST"
        value = "127.0.0.1"
      }

      env {
        name  = "POSTGRES_PORT"
        value = "5432"
      }

      env {
        name  = "POSTGRES_USER"
        value = "postgres"
      }

      env {
        name  = "POSTGRES_DB"
        value = "jobdeck-ui"
      }

      // MEMO: exec migrate in ui container
      env {
        name  = "MIGRATOR_EXECUTE"
        value = "true"
      }

      dynamic "env" {
        for_each = [
          { name = "GITHUB_OAUTH_CLIENT_ID", key = "github-oauth-client-id" },
          { name = "GITHUB_OAUTH_CLIENT_SECRET", key = "github-oauth-client-secret" },
          { name = "CERT_DATAPROTECTION_BASE64", key = "ui-dataprotection-pfx-base64" },
          { name = "CERT_OPENID_ENCRYPTION_BASE64", key = "ui-openid-encryption-pfx-base64" },
          { name = "CERT_OPENID_SIGNING_BASE64", key = "ui-openid-signing-pfx-base64" },
          { name = "POSTGRES_PASSWORD", key = "postgres-password" }
        ]

        content {
          name = env.value.name
          value_source {
            secret_key_ref {
              secret  = local.application_secrets[env.value.key]
              version = "latest"
            }
          }
        }
      }

      startup_probe {
        http_get {
          path = "/health/live"
          port = 8080
        }
        failure_threshold = 10
        period_seconds    = 3
        timeout_seconds   = 1
      }
    }

    containers {
      name  = "postgres"
      image = local.postgres_image

      env {
        name  = "POSTGRES_DB"
        value = "jobdeck-ui"
      }

      env {
        name  = "POSTGRES_USER"
        value = "postgres"
      }

      env {
        name = "POSTGRES_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["postgres-password"]
            version = "latest"
          }
        }
      }

      startup_probe {
        tcp_socket {
          port = 5432
        }
        failure_threshold = 30
        period_seconds    = 1
        timeout_seconds   = 1
      }
    }
  }
}
