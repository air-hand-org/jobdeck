resource "google_cloud_run_v2_service" "job_worker" {
  name                 = "jobdeck-job-worker"
  location             = var.region
  ingress              = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  invoker_iam_disabled = false

  template {
    service_account = local.runtime_service_accounts.job_worker
    timeout         = "60s"

    vpc_access {
      egress = "ALL_TRAFFIC"

      network_interfaces {
        network    = google_compute_network.jobdeck.id
        subnetwork = google_compute_subnetwork.cloud_run.id
      }
    }

    containers {
      name  = "job-worker"
      image = local.images.job_worker

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
        name = "AUTH_CLIENT_SECRET"
        value_source {
          secret_key_ref {
            secret  = local.application_secrets["job-worker-client-secret"]
            version = "latest"
          }
        }
      }
    }
  }
}
