// Existing jobdeck-auth governance is adopted before this root is allowed to
// apply. These imports must remain until the first successful bootstrap apply.
import {
  to = github_repository_ruleset.auth_main
  id = "jobdeck-auth:22442155"
}

import {
  to = github_repository_ruleset.auth_release_tag
  id = "jobdeck-auth:22442376"
}

import {
  to = github_repository_environment.auth_production
  id = "jobdeck-auth:production"
}

import {
  to = github_repository_environment_deployment_policy.auth_production_release_tag
  id = "jobdeck-auth:production:59393350"
}

import {
  for_each = toset([
    "GCP_PROJECT_ID",
    "GCP_SERVICE_ACCOUNT",
    "GCP_WORKLOAD_IDENTITY_PROVIDER",
  ])

  to = github_actions_environment_variable.auth_production[each.value]
  id = "jobdeck-auth:production:${each.value}"
}
