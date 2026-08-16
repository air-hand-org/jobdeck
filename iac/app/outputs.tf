output "cloud_run_service_uris" {
  description = "Cloud Run service URIs. Public DNS is added with the load-balancer slice."

  value = {
    ui      = google_cloud_run_v2_service.ui.uri
    demo    = google_cloud_run_v2_service.demo.uri
    auth    = google_cloud_run_v2_service.auth.uri
    job_api = google_cloud_run_v2_service.job_api.uri
  }
}
