# The existing `github` pool has environment-only SA bindings. Keep publishing
# in a separate pool so this repository cannot impersonate the deployer.
resource "google_iam_workload_identity_pool" "job_worker_publisher" {
  project                   = var.project_id
  workload_identity_pool_id = "jobdeck-job-worker-publish"
  display_name              = "JobDeck job-worker image publishing"
  description               = "Dedicated trust boundary for jobdeck-job-worker image publication."

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_iam_workload_identity_pool_provider" "job_worker_publisher" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.job_worker_publisher.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "JobDeck job-worker GitHub Actions"

  attribute_mapping = {
    "google.subject" = "assertion.sub"
    # Jobs without a GitHub Environment cannot match the publisher SA binding.
    "attribute.environment" = "'environment' in assertion ? assertion.environment : ''"
  }

  attribute_condition = join(" && ", [
    "assertion.repository == '${github_repository.job_worker.full_name}'",
  ])

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "job_worker_publisher" {
  project      = var.project_id
  account_id   = "jobdeck-job-worker-publisher"
  display_name = "JobDeck job-worker image publisher"
  description  = "Publishes images only to the jobdeck-job-worker Artifact Registry repository."

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_artifact_registry_repository_iam_member" "job_worker_publisher" {
  project    = var.project_id
  location   = google_artifact_registry_repository.service["job_worker"].location
  repository = google_artifact_registry_repository.service["job_worker"].repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.job_worker_publisher.email}"
}

resource "google_artifact_registry_repository_iam_member" "forward_proxy_publisher" {
  project    = var.project_id
  location   = google_artifact_registry_repository.service["forward_proxy"].location
  repository = google_artifact_registry_repository.service["forward_proxy"].repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.auth_publisher.email}"
}

resource "google_service_account_iam_member" "job_worker_publisher_wif" {
  service_account_id = google_service_account.job_worker_publisher.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.job_worker_publisher.name}/attribute.environment/${github_repository_environment.job_worker_production.environment}"

  depends_on = [google_iam_workload_identity_pool_provider.job_worker_publisher]
}
