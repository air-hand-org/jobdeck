# The existing `github` pool has environment-only SA bindings. Keep publishing
# in a separate pool so this repository cannot impersonate the deployer.
resource "google_iam_workload_identity_pool" "auth_publisher" {
  project                   = var.project_id
  workload_identity_pool_id = "jobdeck-auth-publish"
  display_name              = "JobDeck auth image publishing"
  description               = "Dedicated trust boundary for jobdeck-auth image publication."

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_iam_workload_identity_pool_provider" "auth_publisher" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.auth_publisher.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "JobDeck auth GitHub Actions"

  attribute_mapping = {
    "google.subject" = "assertion.sub"
    # Jobs without a GitHub Environment cannot match the publisher SA binding.
    "attribute.environment" = "'environment' in assertion ? assertion.environment : ''"
  }

  # Numeric IDs prevent name reuse from authorizing another repo or owner.
  attribute_condition = join(" && ", [
    "assertion.repository == 'air-hand-org/jobdeck-auth'",
  ])

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "auth_publisher" {
  project      = var.project_id
  account_id   = "jobdeck-auth-publisher"
  display_name = "JobDeck auth image publisher"
  description  = "Publishes images only to the jobdeck-auth Artifact Registry repository."

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_artifact_registry_repository_iam_member" "auth_publisher" {
  project    = var.project_id
  location   = google_artifact_registry_repository.service["auth"].location
  repository = google_artifact_registry_repository.service["auth"].repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.auth_publisher.email}"
}

resource "google_service_account_iam_member" "auth_publisher_wif" {
  service_account_id = google_service_account.auth_publisher.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.auth_publisher.name}/attribute.environment/production"

  depends_on = [google_iam_workload_identity_pool_provider.auth_publisher]
}
