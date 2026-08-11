output "artifact_registry_repository" {
  value = google_artifact_registry_repository.jobdeck.name
}

output "artifact_registry_host" {
  value = "${var.region}-docker.pkg.dev"
}

output "app_dns_nameservers" {
  value = google_dns_managed_zone.app.name_servers
}

output "deployer_service_account" {
  value = google_service_account.deployer.email
}

output "plan_service_account" {
  value = google_service_account.plan.email
}

output "tofu_state_kms_key" {
  value = google_kms_crypto_key.tofu_state.id
}

output "sops_kms_key" {
  value = google_kms_crypto_key.sops.id
}
