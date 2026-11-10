resource "github_repository" "job_worker" {
  name         = "jobdeck-job-worker"
  visibility   = "public"
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

  security_and_analysis {
    secret_scanning {
      status = "enabled"
    }
    secret_scanning_push_protection {
      status = "enabled"
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_branch" "job_worker_main" {
  repository = github_repository.job_worker.name
  branch     = "main"
}

resource "github_branch_default" "job_worker_default" {
  repository = github_repository.job_worker.name
  branch     = github_branch.job_worker_main.branch
}

resource "github_repository_ruleset" "job_worker_main" {
  repository  = github_repository.job_worker.name
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

    merge_queue {
      check_response_timeout_minutes    = 60
      grouping_strategy                 = "ALLGREEN"
      max_entries_to_build              = 5
      max_entries_to_merge              = 1
      merge_method                      = "SQUASH"
      min_entries_to_merge              = 1
      min_entries_to_merge_wait_minutes = 5
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

resource "github_repository_ruleset" "job_worker_release_tag" {
  repository  = github_repository.job_worker.name
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

resource "github_repository_environment" "job_worker_production" {
  repository          = github_repository.job_worker.name
  environment         = "production"
  prevent_self_review = false

  deployment_branch_policy {
    protected_branches     = false
    custom_branch_policies = true
  }
}

resource "github_repository_environment_deployment_policy" "job_worker_production_release_tag" {
  repository  = github_repository.job_worker.name
  environment = github_repository_environment.job_worker_production.environment
  tag_pattern = "v*.*.*"
}

resource "github_actions_environment_variable" "job_worker_production" {
  for_each = {
    GCP_PROJECT_ID                 = data.google_project.current.project_id
    GCP_SERVICE_ACCOUNT            = google_service_account.job_worker_publisher.email
    GCP_WORKLOAD_IDENTITY_PROVIDER = google_iam_workload_identity_pool_provider.job_worker_publisher.name
  }

  repository    = github_repository.job_worker.name
  environment   = github_repository_environment.job_worker_production.environment
  variable_name = each.key
  value         = each.value
}
