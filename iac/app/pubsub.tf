resource "google_pubsub_topic" "execution_requested" {
  project = var.project_id
  name    = "jobdeck-execution-requested"
}

resource "google_pubsub_topic_iam_member" "job_api_publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.execution_requested.name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${local.runtime_service_accounts.job_api}"
}

resource "google_cloud_run_v2_service_iam_member" "pubsub_worker_invoker" {
  name     = google_cloud_run_v2_service.job_worker.name
  location = google_cloud_run_v2_service.job_worker.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${local.pubsub_push_service_account}"
}

resource "google_pubsub_subscription" "job_worker" {
  project = var.project_id
  name    = "jobdeck-job-worker"
  topic   = google_pubsub_topic.execution_requested.name

  # The worker acknowledges after claiming the execution. Allow a cold start
  # plus auth/job-api readiness without avoidable duplicate deliveries.
  ack_deadline_seconds = 60

  push_config {
    push_endpoint = google_cloud_run_v2_service.job_worker.uri

    oidc_token {
      service_account_email = local.pubsub_push_service_account
    }
  }

  depends_on = [google_cloud_run_v2_service_iam_member.pubsub_worker_invoker]
}
