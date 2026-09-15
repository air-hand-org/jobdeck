// GitHub repository governance is part of the platform bootstrap, but the
// GitHub App credentials remain an Environment-level trust root. The workflow
// maps GH_IAC_APP_* to the provider's GITHUB_APP_* environment variables.
provider "github" {
  owner = "air-hand-org"

  app_auth {}
}
