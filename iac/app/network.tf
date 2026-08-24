locals {
  # https://cloud.google.com/vpc/docs/configure-private-google-access
  private_google_access_ipv4_cidr = "199.36.153.8/30"
}

resource "google_compute_network" "jobdeck" {
  name                    = "jobdeck"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "cloud_run" {
  name                     = "jobdeck-cloud-run"
  region                   = var.region
  network                  = google_compute_network.jobdeck.id
  ip_cidr_range            = var.cloud_run_subnet_cidr
  private_ip_google_access = true
}

# Keep calls to Cloud Run's default URLs on the VPC while services use
# PRIVATE_RANGES_ONLY egress. The wildcard preserves the original run.app host
# for TLS SNI and Host routing, while the apex resolves to the PGA VIP.
resource "google_dns_managed_zone" "run_app_private" {
  name        = "jobdeck-run-app-private"
  dns_name    = "run.app."
  description = "Routes Cloud Run service URLs through Private Google Access"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = google_compute_network.jobdeck.id
    }
  }
}

resource "google_dns_record_set" "run_app_private" {
  managed_zone = google_dns_managed_zone.run_app_private.name
  name         = "run.app."
  type         = "A"
  ttl          = 300
  rrdatas = [
    for host_number in range(4) :
    cidrhost(local.private_google_access_ipv4_cidr, host_number)
  ]
}

resource "google_dns_record_set" "run_app_private_wildcard" {
  managed_zone = google_dns_managed_zone.run_app_private.name
  name         = "*.run.app."
  type         = "CNAME"
  ttl          = 300
  rrdatas      = [google_dns_record_set.run_app_private.name]
}

resource "google_compute_subnetwork" "load_balancer_proxy" {
  name          = "jobdeck-lb-proxy"
  region        = var.region
  network       = google_compute_network.jobdeck.id
  ip_cidr_range = var.load_balancer_proxy_subnet_cidr
  purpose       = "REGIONAL_MANAGED_PROXY"
  role          = "ACTIVE"
}
