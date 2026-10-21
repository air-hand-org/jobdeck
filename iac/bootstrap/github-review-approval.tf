# Keep approval exemptions separate from the existing baseline CI/PR rulesets.
resource "github_repository_ruleset" "platform_review_approval" {
  for_each = toset([
    github_repository.auth.name,
    github_repository.ui.name,
    github_repository.job_api.name,
    github_repository.job_worker.name,
  ])

  repository  = each.value
  name        = "main-review-approval"
  target      = "branch"
  enforcement = "active"

  conditions {
    ref_name {
      include = ["~DEFAULT_BRANCH"]
      exclude = []
    }
  }

  bypass_actors {
    actor_id    = local.air_hand_user_id
    actor_type  = "User"
    bypass_mode = "exempt"
  }

  bypass_actors {
    actor_id    = data.github_app.demo_renovate.id
    actor_type  = "Integration"
    bypass_mode = "exempt"
  }

  rules {
    pull_request {
      required_approving_review_count   = 1
      dismiss_stale_reviews_on_push     = true
      require_code_owner_review         = contains([github_repository.auth.name, github_repository.job_api.name], each.value)
      require_last_push_approval        = false
      required_review_thread_resolution = contains([github_repository.auth.name, github_repository.job_api.name], each.value)
      allowed_merge_methods             = ["squash"]
    }
  }
}
