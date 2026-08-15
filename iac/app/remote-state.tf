data "terraform_remote_state" "bootstrap" {
  backend = "gcs"

  config = {
    bucket = var.bootstrap_state_bucket_name
    prefix = var.bootstrap_state_prefix
  }
}
