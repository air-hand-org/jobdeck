resource "github_repository" "actions" {
  name         = "jobdeck-actions"
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

  squash_merge_commit_title = "PR_TITLE"

  auto_init = true

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_branch" "actions_main" {
  repository = github_repository.actions.name
  branch     = "main"
}

resource "github_branch_default" "actions_default" {
  repository = github_repository.actions.name
  branch     = github_branch.actions_main.branch
}

resource "github_repository_ruleset" "actions_main" {
  repository  = github_repository.actions.name
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

resource "github_repository_ruleset" "actions_release_tag" {
  repository  = github_repository.actions.name
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

resource "github_actions_repository_access_level" "actions" {
  repository   = github_repository.actions.name
  access_level = "organization"
}
