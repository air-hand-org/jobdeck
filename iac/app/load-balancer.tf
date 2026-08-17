locals {
  public_hosts = {
    ui   = var.public_domain
    demo = "demo.${var.public_domain}"
  }
}

resource "google_compute_region_network_endpoint_group" "public" {
  for_each = {
    ui   = google_cloud_run_v2_service.ui.name
    demo = google_cloud_run_v2_service.demo.name
  }

  name                  = "jobdeck-${each.key}"
  region                = var.region
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = each.value
  }
}

resource "google_compute_region_security_policy" "public" {
  name        = "jobdeck-public"
  region      = var.region
  description = "Baseline protection for JobDeck public endpoints"
  type        = "CLOUD_ARMOR"
}

resource "google_compute_region_security_policy_rule" "rate_limit" {
  region          = var.region
  security_policy = google_compute_region_security_policy.public.name
  priority        = 1000
  description     = "Throttle excessive requests per source IP"
  action          = "throttle"
  preview         = false

  match {
    versioned_expr = "SRC_IPS_V1"

    config {
      src_ip_ranges = ["*"]
    }
  }

  rate_limit_options {
    conform_action = "allow"
    exceed_action  = "deny(429)"
    enforce_on_key = "IP"

    rate_limit_threshold {
      count        = 300
      interval_sec = 60
    }
  }
}

resource "google_compute_region_security_policy_rule" "allow" {
  region          = var.region
  security_policy = google_compute_region_security_policy.public.name
  priority        = 2147483647
  description     = "Allow requests below the baseline rate limit"
  action          = "allow"
  preview         = false

  match {
    versioned_expr = "SRC_IPS_V1"

    config {
      src_ip_ranges = ["*"]
    }
  }
}

resource "google_compute_region_backend_service" "public" {
  for_each = google_compute_region_network_endpoint_group.public

  name                  = "jobdeck-${each.key}"
  region                = var.region
  protocol              = "HTTP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_region_security_policy.public.id

  backend {
    group           = each.value.id
    capacity_scaler = 1.0
  }
}

resource "google_compute_region_url_map" "public" {
  name            = "jobdeck-public"
  region          = var.region
  default_service = google_compute_region_backend_service.public["ui"].id

  host_rule {
    hosts        = [local.public_hosts.ui]
    path_matcher = "ui"
  }

  path_matcher {
    name            = "ui"
    default_service = google_compute_region_backend_service.public["ui"].id
  }

  host_rule {
    hosts        = [local.public_hosts.demo]
    path_matcher = "demo"
  }

  path_matcher {
    name            = "demo"
    default_service = google_compute_region_backend_service.public["demo"].id
  }
}

resource "google_certificate_manager_dns_authorization" "public" {
  for_each = local.public_hosts

  name     = "jobdeck-${each.key}"
  location = var.region
  type     = "PER_PROJECT_RECORD"
  domain   = each.value
}

resource "google_dns_record_set" "certificate_authorization" {
  for_each = google_certificate_manager_dns_authorization.public

  managed_zone = data.terraform_remote_state.bootstrap.outputs.app_dns_managed_zone
  name         = each.value.dns_resource_record[0].name
  type         = each.value.dns_resource_record[0].type
  ttl          = 300
  rrdatas      = [each.value.dns_resource_record[0].data]
}

resource "google_certificate_manager_certificate" "public" {
  name     = "jobdeck-public"
  location = var.region

  managed {
    domains            = values(local.public_hosts)
    dns_authorizations = [for authorization in google_certificate_manager_dns_authorization.public : authorization.id]
  }

  depends_on = [google_dns_record_set.certificate_authorization]
}

resource "google_compute_address" "public" {
  name         = "jobdeck-public"
  region       = var.region
  address_type = "EXTERNAL"
  network_tier = "STANDARD"
}

resource "google_compute_region_target_https_proxy" "public" {
  name                             = "jobdeck-public"
  region                           = var.region
  url_map                          = google_compute_region_url_map.public.id
  certificate_manager_certificates = [google_certificate_manager_certificate.public.id]
}

resource "google_compute_forwarding_rule" "https" {
  name                  = "jobdeck-https"
  region                = var.region
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "443"
  target                = google_compute_region_target_https_proxy.public.id
  network               = google_compute_network.jobdeck.id
  ip_address            = google_compute_address.public.address
  network_tier          = "STANDARD"

  depends_on = [google_compute_subnetwork.load_balancer_proxy]
}

resource "google_dns_record_set" "public" {
  for_each = local.public_hosts

  managed_zone = data.terraform_remote_state.bootstrap.outputs.app_dns_managed_zone
  name         = "${each.value}."
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_address.public.address]
}
