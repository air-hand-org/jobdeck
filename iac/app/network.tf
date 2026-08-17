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

resource "google_compute_subnetwork" "load_balancer_proxy" {
  name          = "jobdeck-lb-proxy"
  region        = var.region
  network       = google_compute_network.jobdeck.id
  ip_cidr_range = var.load_balancer_proxy_subnet_cidr
  purpose       = "REGIONAL_MANAGED_PROXY"
  role          = "ACTIVE"
}
