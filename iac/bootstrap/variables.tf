variable "project_id" {
  type = string
}

variable "project_number" {
  type = string
}

variable "region" {
  type    = string
  default = "asia-northeast1"
}

variable "state_bucket_name" {
  type    = string
  default = "airhand-jobdeck-bucket"
}

variable "github_repository" {
  type    = string
  default = "air-hand-org/jobdeck"
}

variable "github_wif_pool_id" {
  type    = string
  default = "github"
}

variable "app_dns_name" {
  type    = string
  default = "app.air-hand.com."
}
