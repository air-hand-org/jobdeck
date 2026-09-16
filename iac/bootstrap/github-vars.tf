locals {
  # GitHub's numeric App ID for air-hand-release-app. This ID is the ruleset
  # Integration actor ID, not its installation ID or client ID.
  release_app_integration_id = 3404920
}

# This is GitHub's public App for Actions checks. Unlike private Apps, it can
# be resolved by an installation token through the public Apps endpoint.
data "github_app" "github_actions" {
  slug = "github-actions"
}
