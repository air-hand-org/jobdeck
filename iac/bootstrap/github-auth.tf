locals {
  auth_repository = "jobdeck-auth"
}

resource "github_repository_ruleset" "auth_main" {
  repository  = local.auth_repository
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
        integration_id = 15368
      }
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_repository_ruleset" "auth_release_tag" {
  repository  = local.auth_repository
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
    actor_id    = 3404920
    actor_type  = "Integration"
    bypass_mode = "exempt"
  }

  rules {
    creation = true
    update   = true
    deletion = true
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_repository_environment" "auth_production" {
  repository          = local.auth_repository
  environment         = "production"
  can_admins_bypass   = true
  prevent_self_review = false

  deployment_branch_policy {
    protected_branches     = false
    custom_branch_policies = true
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_repository_environment_deployment_policy" "auth_production_release_tag" {
  repository  = local.auth_repository
  environment = github_repository_environment.auth_production.environment
  tag_pattern = "v*.*.*"

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_actions_environment_variable" "auth_production" {
  for_each = {
    GCP_PROJECT_ID                 = "airhand-jobdeck"
    GCP_SERVICE_ACCOUNT            = "jobdeck-auth-publisher@airhand-jobdeck.iam.gserviceaccount.com"
    GCP_WORKLOAD_IDENTITY_PROVIDER = "projects/540051760819/locations/global/workloadIdentityPools/jobdeck-auth-publish/providers/github"
  }

  repository    = local.auth_repository
  environment   = github_repository_environment.auth_production.environment
  variable_name = each.key
  value         = each.value

  lifecycle {
    prevent_destroy = true
  }
}
