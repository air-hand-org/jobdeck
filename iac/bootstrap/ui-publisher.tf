# The existing `github` pool has environment-only SA bindings. Keep publishing
# in a separate pool so this repository cannot impersonate the deployer.
resource "google_iam_workload_identity_pool" "ui_publisher" {
  project                   = var.project_id
  workload_identity_pool_id = "jobdeck-ui-publish"
  display_name              = "JobDeck ui image publishing"
  description               = "Dedicated trust boundary for jobdeck-ui image publication."

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_iam_workload_identity_pool_provider" "ui_publisher" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.ui_publisher.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "JobDeck ui GitHub Actions"

  attribute_mapping = {
    "google.subject" = "assertion.sub"
    # Jobs without a GitHub Environment cannot match the publisher SA binding.
    "attribute.environment" = "'environment' in assertion ? assertion.environment : ''"
  }

  attribute_condition = join(" && ", [
    "assertion.repository == '${github_repository.ui.full_name}'",
  ])

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "ui_publisher" {
  project      = var.project_id
  account_id   = "jobdeck-ui-publisher"
  display_name = "JobDeck ui image publisher"
  description  = "Publishes images only to the jobdeck-ui Artifact Registry repository."

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_artifact_registry_repository_iam_member" "ui_publisher" {
  project    = var.project_id
  location   = google_artifact_registry_repository.service["ui"].location
  repository = google_artifact_registry_repository.service["ui"].repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.ui_publisher.email}"
}

resource "google_service_account_iam_member" "ui_publisher_wif" {
  service_account_id = google_service_account.ui_publisher.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.ui_publisher.name}/attribute.environment/${github_repository_environment.ui_production.environment}"

  depends_on = [google_iam_workload_identity_pool_provider.ui_publisher]
}
