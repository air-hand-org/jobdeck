locals {
  required_services = toset([
    "artifactregistry.googleapis.com",
    "certificatemanager.googleapis.com",
    "cloudkms.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "compute.googleapis.com",
    "dns.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "sts.googleapis.com",
    "cloudscheduler.googleapis.com",
  ])

  deployer_roles = toset([
    "roles/cloudscheduler.admin",
    "roles/compute.admin",
    "roles/dns.admin",
    "roles/run.admin",
  ])

  plan_roles = toset([
    "roles/viewer",
    "roles/iam.securityReviewer",
  ])

  artifact_registry_repositories = {
    ui            = "jobdeck-ui"
    auth          = "jobdeck-auth"
    job_api       = "jobdeck-job-api"
    job_worker    = "jobdeck-job-worker"
    demo          = "jobdeck-demo"
    forward_proxy = "jobdeck-forward-proxy"
  }

  secret_consumers = {
    "auth-jwt-private-key" = toset([
      "auth",
    ])
    "auth-jwt-public-key" = toset([
      "job_api",
    ])
    "job-worker-client-secret" = toset([
      "job_worker",
    ])
    "job-worker-client-secret-hash" = toset([
      "auth",
    ])
    "test-user-client-secret-hash" = toset([
      "auth",
    ])
    # The value is added separately through the write-only secret-version flow.
    "postgres-password" = toset([
      "auth",
      "job_api",
    ])
  }

  runtime_service_accounts = {
    ui         = "jobdeck-ui"
    auth       = "jobdeck-auth"
    job_api    = "jobdeck-job-api"
    job_worker = "jobdeck-worker"
    demo       = "jobdeck-demo"
  }
}

data "google_project" "current" {
  project_id = var.project_id

  depends_on = [google_project_service.required["cloudresourcemanager.googleapis.com"]]
}

resource "google_project_service" "required" {
  for_each           = local.required_services
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "service" {
  for_each      = local.artifact_registry_repositories
  project       = var.project_id
  location      = var.region
  repository_id = each.value
  description   = "JobDeck ${each.key} images"
  format        = "DOCKER"

  docker_config {
    # A release tag must always identify exactly the image that was deployed.
    immutable_tags = true
  }

  depends_on = [google_project_service.required["artifactregistry.googleapis.com"]]
}

resource "google_artifact_registry_repository_iam_member" "deployer" {
  for_each   = google_artifact_registry_repository.service
  project    = var.project_id
  location   = each.value.location
  repository = each.value.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_kms_key_ring" "jobdeck" {
  project    = var.project_id
  name       = "jobdeck"
  location   = var.region
  depends_on = [google_project_service.required["cloudkms.googleapis.com"]]
}

resource "google_kms_crypto_key" "tofu_state" {
  name            = "tofu-state"
  key_ring        = google_kms_key_ring.jobdeck.id
  rotation_period = "7776000s" # 90 days

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_kms_crypto_key" "sops" {
  name            = "sops"
  key_ring        = google_kms_key_ring.jobdeck.id
  rotation_period = "7776000s" # 90 days

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_secret_manager_secret" "application" {
  for_each  = local.secret_consumers
  project   = var.project_id
  secret_id = each.key

  replication {
    auto {}
  }

  depends_on = [google_project_service.required["secretmanager.googleapis.com"]]
}

resource "google_dns_managed_zone" "app" {
  project     = var.project_id
  name        = "jobdeck-app"
  dns_name    = var.app_dns_name
  description = "Delegated public DNS zone for JobDeck"

  depends_on = [google_project_service.required["dns.googleapis.com"]]
}

resource "google_service_account" "deployer" {
  project      = var.project_id
  account_id   = "jobdeck-deployer"
  display_name = "JobDeck application deployer"
}

data "google_service_account" "plan" {
  project    = var.project_id
  account_id = "jobdeck-plan"
}

resource "google_service_account" "runtime" {
  for_each     = local.runtime_service_accounts
  project      = var.project_id
  account_id   = each.value
  display_name = "JobDeck ${each.key} runtime"
}

resource "google_project_iam_member" "deployer" {
  for_each = local.deployer_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "plan" {
  for_each = local.plan_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${data.google_service_account.plan.email}"
}

resource "google_service_account_iam_member" "deployer_runtime" {
  for_each           = google_service_account.runtime
  service_account_id = each.value.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_storage_bucket_iam_member" "deployer_state" {
  bucket = var.state_bucket_name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_storage_bucket_iam_member" "plan_state" {
  bucket = var.state_bucket_name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${data.google_service_account.plan.email}"
}

resource "google_kms_crypto_key_iam_member" "deployer_state" {
  crypto_key_id = google_kms_crypto_key.tofu_state.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_kms_crypto_key_iam_member" "plan_state" {
  crypto_key_id = google_kms_crypto_key.tofu_state.id
  role          = "roles/cloudkms.cryptoKeyDecrypter"
  member        = "serviceAccount:${data.google_service_account.plan.email}"
}

resource "google_kms_crypto_key_iam_member" "deployer_sops" {
  crypto_key_id = google_kms_crypto_key.sops.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_secret_manager_secret_iam_member" "deployer_version_adder" {
  for_each = local.secret_consumers

  project   = var.project_id
  secret_id = google_secret_manager_secret.application[each.key].secret_id
  role      = "roles/secretmanager.secretVersionAdder"
  member    = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_secret_manager_secret_iam_member" "runtime" {
  for_each = {
    for pair in flatten([
      for secret, consumers in local.secret_consumers : [
        for consumer in consumers : {
          secret   = secret
          consumer = consumer
        }
      ]
    ]) : "${pair.consumer}:${pair.secret}" => pair
  }

  project   = var.project_id
  secret_id = google_secret_manager_secret.application[each.value.secret].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime[each.value.consumer].email}"
}

resource "google_service_account_iam_member" "deployer_wif" {
  service_account_id = google_service_account.deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${data.google_project.current.number}/locations/global/workloadIdentityPools/${var.github_wif_pool_id}/attribute.environment/production"
}

resource "google_service_account_iam_member" "plan_wif" {
  service_account_id = data.google_service_account.plan.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${data.google_project.current.number}/locations/global/workloadIdentityPools/${var.github_wif_pool_id}/attribute.environment/plan"
}
