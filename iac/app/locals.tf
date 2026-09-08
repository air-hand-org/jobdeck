locals {
  artifact_registry_host         = data.terraform_remote_state.bootstrap.outputs.artifact_registry_host
  artifact_registry_repositories = data.terraform_remote_state.bootstrap.outputs.artifact_registry_repositories
  runtime_service_accounts       = data.terraform_remote_state.bootstrap.outputs.runtime_service_accounts
  application_secrets            = data.terraform_remote_state.bootstrap.outputs.application_secrets
  pubsub_push_service_account    = data.terraform_remote_state.bootstrap.outputs.pubsub_push_service_account

  production = yamldecode(file("${path.module}/../../environments/production.yaml"))

  images = {
    # Require the same stable X.Y.Z release versions previously accepted by image_tag.
    for name, image in local.production.images :
    replace(name, "-", "_") => "${local.artifact_registry_host}/${var.project_id}/${local.artifact_registry_repositories[replace(name, "-", "_")]}/${name}:v${regex("^[0-9]+\\.[0-9]+\\.[0-9]+$", image.version)}"
  }

  # Cloud Run runs linux/amd64 images. Pin the multi-architecture manifest so
  # PostgreSQL sidecar updates are deliberate, like application image releases.
  postgres_image = "postgres:18.4-bookworm@sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382"
}
