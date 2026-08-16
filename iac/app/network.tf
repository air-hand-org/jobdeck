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
