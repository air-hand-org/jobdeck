resource "google_cloud_run_v2_service" "job_worker" {
  name                 = "jobdeck-job-worker"
  location             = var.region
  ingress              = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  invoker_iam_disabled = false

  template {
    service_account = local.runtime_service_accounts.job_worker
    timeout         = "60s"

    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"

      network_interfaces {
        network    = google_compute_network.jobdeck.id
        subnetwork = google_compute_subnetwork.cloud_run.id
      }
    }

    containers {
      name       = "job-worker"
      image      = local.images.job_worker
      depends_on = ["forward-proxy"]

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
        name  = "AUTH_CLIENT_ID"
        value = "job-worker"
      }

      env {
        name  = "WORKER_HTTP_PROXY"
        value = "http://127.0.0.1:3128"
      }

      env {
        name = "AUTH_CLIENT_SECRET"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["job-worker-client-secret"]
            version = "latest"
          }
        }
      }
    }

    containers {
      name  = "forward-proxy"
      image = local.images.forward_proxy

      env {
        name  = "SQUID_ALLOWED_DEMO_HOST_AND_PORT"
        value = "demo.${var.public_domain}:443"
      }

      startup_probe {
        tcp_socket {
          port = 3128
        }
        failure_threshold = 10
        period_seconds    = 1
        timeout_seconds   = 1
      }
    }
  }
}
