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
    "roles/certificatemanager.editor",
    "roles/cloudscheduler.admin",
    "roles/compute.admin",
    "roles/dns.admin",
    "roles/run.admin",