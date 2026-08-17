output "cloud_run_service_uris" {
  description = "Cloud Run service URIs behind the regional load balancer or internal service mesh."

  value = {
    ui      = google_cloud_run_v2_service.ui.uri
    demo    = google_cloud_run_v2_service.demo.uri
    auth    = google_cloud_run_v2_service.auth.uri
    job_api = google_cloud_run_v2_service.job_api.uri
  }
}

output "public_ip_address" {
  description = "Regional external Application Load Balancer IPv4 address."
  value       = google_compute_address.public.address
}

output "public_urls" {
  description = "Public HTTPS URLs served by the regional load balancer."

  value = {
    for service, hostname in local.public_hosts :
    service => "https://${hostname}"
  }
}
