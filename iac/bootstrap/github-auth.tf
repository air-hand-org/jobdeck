resource "github_repository" "auth" {
  name         = "jobdeck-auth"
  visibility   = "private"
  has_issues   = true
  has_projects = true
  has_wiki     = true

  has_discussions             = false
  allow_merge_commit          = false
  allow_squash_merge          = true
  allow_rebase_merge          = false
  allow_auto_merge            = true
  allow_update_branch         = false
  delete_branch_on_merge      = true
  web_commit_signoff_required = false

  squash_merge_commit_message = "COMMIT_MESSAGES"
  squash_merge_commit_title   = "PR_TITLE"

  lifecycle {
    prevent_destroy = true

    # The GitHub provider's import/read path reports these enabled settings as
    # false, despite GitHub returning true. Keep the existing repository
    # import stable; repository resources created by this root do not use this
    # exception.
    ignore_changes = [
      ignore_vulnerability_alerts_during_read,
      merge_commit_message,
      merge_commit_title,
    ]
  }
}

resource "github_repository_ruleset" "auth_main" {
  repository  = github_repository.auth.name
  name        = "main"
  target      = "branch"
  enforcement = "active"

  conditions {
    ref_name {
      include = ["~DEFAULT_BRANCH"]
      exclude = []
    }
  }

  rules {
    deletion         = true
    non_fast_forward = true

    pull_request {
      required_approving_review_count   = 0
      dismiss_stale_reviews_on_push     = false
      require_code_owner_review         = false
      require_last_push_approval        = false
      required_review_thread_resolution = false
      # GitHub's "require additional approval for un-attributed changes"
      # setting is enabled on this ruleset, but provider v6.13 does not expose
      # it. It remains configured in GitHub until the provider supports it.
      allowed_merge_methods = ["merge", "squash", "rebase"]
    }

    required_status_checks {
      strict_required_status_checks_policy = true
      do_not_enforce_on_create             = false

      required_check {
        context        = "check-result"
        integration_id = data.github_app.github_actions.id
      }
    }
  }

}

resource "github_repository_ruleset" "auth_release_tag" {
  repository  = github_repository.auth.name
  name        = "release-tag"
  target      = "tag"
  enforcement = "active"

  conditions {
    ref_name {
      include = ["refs/tags/v*.*.*"]
      exclude = []
    }
  }

  bypass_actors {
    actor_id    = local.release_app_integration_id
    actor_type  = "Integration"
    bypass_mode = "exempt"
  }

  rules {
    creation         = true
    update           = true
    deletion         = true
    non_fast_forward = true
  }
}

resource "github_repository_environment" "auth_production" {
  repository          = github_repository.auth.name
  environment         = "production"
  can_admins_bypass   = true
  prevent_self_review = false

  deployment_branch_policy {
    protected_branches     = false
    custom_branch_policies = true
  }

}

resource "github_repository_environment_deployment_policy" "auth_production_release_tag" {
  repository  = github_repository.auth.name
  environment = github_repository_environment.auth_production.environment
  tag_pattern = "v*.*.*"
}

resource "github_actions_environment_variable" "auth_production" {
  for_each = {
    GCP_PROJECT_ID                 = data.google_project.current.project_id
    GCP_SERVICE_ACCOUNT            = google_service_account.auth_publisher.email
    GCP_WORKLOAD_IDENTITY_PROVIDER = google_iam_workload_identity_pool_provider.auth_publisher.name
  }

  repository    = github_repository.auth.name
  environment   = github_repository_environment.auth_production.environment
  variable_name = each.key
  value         = each.value

}
