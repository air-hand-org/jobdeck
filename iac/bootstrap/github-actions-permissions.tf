resource "github_actions_repository_permissions" "public_repositories" {
  for_each = {
    actions    = github_repository.actions.name
    auth       = github_repository.auth.name
    job_api    = github_repository.job_api.name
    job_worker = github_repository.job_worker.name
    demo       = github_repository.demo.name
    ui         = github_repository.ui.name
  }

  repository           = each.value
  enabled              = true
  allowed_actions      = "all"
  sha_pinning_required = true
}
