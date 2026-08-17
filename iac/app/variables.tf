variable "project_id" {
  description = "GCP project that owns the JobDeck deployment."
  type        = string
}

variable "region" {
  description = "Region for Cloud Run and regional networking resources."
  type        = string
  default     = "asia-northeast1"
}

variable "bootstrap_state_bucket_name" {
  description = "GCS bucket containing the bootstrap root state to consume."
  type        = string
  default     = "airhand-jobdeck-bucket"
}

variable "bootstrap_state_prefix" {
  description = "GCS backend prefix for the bootstrap root state."
  type        = string
  default     = "bootstrap"
}

variable "image_tag" {
  description = "Immutable release tag used for every JobDeck application image."
  type        = string
  default     = "v0.0.0"

  validation {
    condition     = can(regex("^v[0-9]+\\.[0-9]+\\.[0-9]+$", var.image_tag))
    error_message = "image_tag must be a vX.Y.Z release tag."
  }
}

variable "cloud_run_subnet_cidr" {
  description = "CIDR allocated to Direct VPC egress interfaces."
  type        = string
  default     = "10.42.0.0/26"
}

variable "load_balancer_proxy_subnet_cidr" {
  description = "CIDR reserved for the regional Application Load Balancer proxy-only subnet."
  type        = string
  default     = "10.42.1.0/26"
}

variable "public_domain" {
  description = "Primary public hostname for the JobDeck UI."
  type        = string
  default     = "jobdeck.app.air-hand.com"
}
