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
      name  = "ui"
      image = local.images.ui

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
  }
}

resource "google_cloud_run_v2_service" "demo" {
  name                 = "jobdeck-demo"
  location             = var.region
  ingress              = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"
  invoker_iam_disabled = false

  template {
    service_account = local.runtime_service_accounts.demo
    timeout         = "60s"

    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"

      network_interfaces {
        network    = google_compute_network.jobdeck.id
        subnetwork = google_compute_subnetwork.cloud_run.id
      }
    }

    containers {
      name  = "demo"
      image = local.images.demo

      ports {
        container_port = 8080
      }

      startup_probe {
        http_get {
          path = "/healthz"
          port = 8080
        }
        failure_threshold = 10
        period_seconds    = 3
        timeout_seconds   = 1
      }
    }
  }
}

resource "google_cloud_run_v2_service" "auth" {
  name                 = "jobdeck-auth"
  location             = var.region
  ingress              = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  invoker_iam_disabled = true

  template {
    service_account = local.runtime_service_accounts.auth
    timeout         = "60s"

    scaling {
      # PostgreSQL data is instance-local and intentionally ephemeral.
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
      name       = "auth"
      image      = local.images.auth
      depends_on = ["postgres"]

      ports {
        container_port = 9091
        name           = "h2c"
      }

      env {
        name  = "GRPC_PORT"
        value = "9091"
      }

      env {
        name  = "POSTGRES_HOST"
        value = "127.0.0.1"
      }

      env {
        name  = "POSTGRES_DATABASE"
        value = "jobdeck-auth"
      }

      env {
        name  = "AUTH_JOB_WORKER_CLIENT_ID"
        value = "job-worker"
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

      env {
        name = "AUTH_JWT_PRIVATE_KEY"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["auth-jwt-private-key"]
            version = "latest"
          }
        }
      }

      env {
        name = "AUTH_JOB_WORKER_SECRET_HASH"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["job-worker-client-secret-hash"]
            version = "latest"
          }
        }
      }

      env {
        name = "AUTH_TEST_USER_SECRET_HASH"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["test-user-client-secret-hash"]
            version = "latest"
          }
        }
      }

      startup_probe {
        grpc {
          port = 9091
        }
        failure_threshold = 30
        period_seconds    = 3
        timeout_seconds   = 1
      }
    }

    containers {
      name  = "postgres"
      image = local.postgres_image

      env {
        name  = "POSTGRES_DB"
        value = "jobdeck-auth"
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

resource "google_cloud_run_v2_service" "job_api" {
  name                 = "jobdeck-job-api"
  location             = var.region
  ingress              = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  invoker_iam_disabled = true

  template {
    service_account = local.runtime_service_accounts.job_api
    timeout         = "60s"

    scaling {
      # PostgreSQL data is instance-local and intentionally ephemeral.
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
      name       = "job-api"
      image      = local.images.job_api
      depends_on = ["postgres"]

      ports {
        container_port = 9090
        name           = "h2c"
      }

      env {
        name  = "GRPC_PORT"
        value = "9090"
      }

      env {
        name  = "PUBSUB_PROJECT_ID"
        value = var.project_id
      }

      env {
        name  = "PUBSUB_WORKER_TOPIC_ID"
        value = google_pubsub_topic.execution_requested.name
      }

      env {
        name  = "POSTGRES_HOST"
        value = "127.0.0.1"
      }

      env {
        name  = "POSTGRES_DATABASE"
        value = "jobdeck"
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

      env {
        name = "AUTH_JWT_PUBLIC_KEY"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["auth-jwt-public-key"]
            version = "latest"
          }
        }
      }

      startup_probe {
        grpc {
          port = 9090
        }
        failure_threshold = 30
        period_seconds    = 3
        timeout_seconds   = 1
      }
    }

    containers {
      name  = "postgres"
      image = local.postgres_image

      env {
        name  = "POSTGRES_DB"
        value = "jobdeck"
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

# The external HTTPS load balancer added by the next slice cannot authenticate
# to Cloud Run. Ingress settings above still prevent bypassing that load balancer.
resource "google_cloud_run_v2_service_iam_member" "ui_public_invoker" {
  name     = google_cloud_run_v2_service.ui.name
  location = google_cloud_run_v2_service.ui.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "demo_public_invoker" {
  name     = google_cloud_run_v2_service.demo.name
  location = google_cloud_run_v2_service.demo.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
