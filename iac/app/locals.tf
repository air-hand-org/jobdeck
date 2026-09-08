locals {
  runtime_service_accounts    = data.terraform_remote_state.bootstrap.outputs.runtime_service_accounts
  application_secrets         = data.terraform_remote_state.bootstrap.outputs.application_secrets
  pubsub_push_service_account = data.terraform_remote_state.bootstrap.outputs.pubsub_push_service_account

  production = yamldecode(file("${path.module}/../../environments/production.yaml"))

  images = {
    for name, image in local.production.images :
    replace(name, "-", "_") => "${local.production.registry}/jobdeck-${name}/${name}:v${image.version}"
  }

  # Cloud Run runs linux/amd64 images. Pin the multi-architecture manifest so
  # PostgreSQL sidecar updates are deliberate, like application image releases.
  postgres_image = "postgres:18.4-bookworm@sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382"
}
