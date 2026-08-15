variable "project_id" {
  description = "GCP project that owns the JobDeck deployment."
  type        = string
}

variable "region" {
  description = "Region for Cloud Run and regional networking resources."
  type        = string
  default     = "asia-northeast1"
}

variable "state_bucket_name" {
  description = "Pre-created GCS bucket used by both OpenTofu roots."
  type        = string
  default     = "airhand-jobdeck-bucket"
}

variable "bootstrap_state_prefix" {
  description = "GCS backend prefix for the bootstrap root state."
  type        = string
  default     = "bootstrap"
}
