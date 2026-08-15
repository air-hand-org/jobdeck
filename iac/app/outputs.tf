output "bootstrap" {
  description = "Selected non-secret bootstrap outputs consumed by this root."

  value = {
    artifact_registry_host         = data.terraform_remote_state.bootstrap.outputs.artifact_registry_host
    artifact_registry_repositories = data.terraform_remote_state.bootstrap.outputs.artifact_registry_repositories
    app_dns_managed_zone           = data.terraform_remote_state.bootstrap.outputs.app_dns_managed_zone
    runtime_service_accounts       = data.terraform_remote_state.bootstrap.outputs.runtime_service_accounts
  }
}
