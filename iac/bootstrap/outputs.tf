output "artifact_registry_repositories" {
  value = {
    for service, repository in google_artifact_registry_repository.service :
    service => repository.name
  }
}

output "artifact_registry_host" {
  value = "${var.region}-docker.pkg.dev"
}

output "app_dns_nameservers" {
  value = google_dns_managed_zone.app.name_servers
}

output "app_dns_managed_zone" {
  value = google_dns_managed_zone.app.name
}

output "deployer_service_account" {
  value = google_service_account.deployer.email
}

output "runtime_service_accounts" {
  value = {
    for service, account in google_service_account.runtime :
    service => account.email
  }
}

output "pubsub_push_service_account" {
  value = google_service_account.pubsub_push.email
}

output "plan_service_account" {
  value = data.google_service_account.plan.email
}

output "application_secrets" {
  value = {
    for name, secret in google_secret_manager_secret.application :
    name => secret.id
  }
}

output "tofu_state_kms_key" {
  value = google_kms_crypto_key.tofu_state.id
}

output "sops_kms_key" {
  value = google_kms_crypto_key.sops.id
}
