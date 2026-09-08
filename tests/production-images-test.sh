#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT
export PRODUCTION_MANIFEST="$test_dir/production.yaml"

reset_manifest() {
  cat > "$PRODUCTION_MANIFEST" <<'YAML'
images:
  auth: {version: '1.2.3'}
  ui: {version: '1.2.3'}
  demo: {version: '1.2.3'}
  job-api: {version: '1.2.3'}
  job-worker: {version: '1.2.3'}
  forward-proxy: {version: '1.2.3'}
YAML
}

images() {
  bash "$repo_root/.github/scripts/production-images.sh" "$@"
}

expect_failure() {
  local description="$1"
  shift
  if "$@" > "$test_dir/output" 2> "$test_dir/error"; then
    printf 'FAIL: %s was accepted\n' "$description" >&2
    exit 1
  fi
}

reset_manifest
images v1.2.3 auth ui demo job-api job-worker forward-proxy |
  jq -e '. == {
    "auth": "1.2.3",
    "ui": "1.2.3",
    "demo": "1.2.3",
    "job_api": "1.2.3",
    "job_worker": "1.2.3",
    "forward_proxy": "1.2.3"
  }' > /dev/null
expect_failure 'mismatched release' images v1.2.4 auth ui

# Extracted services need not follow the monorepo release version.
yq -i '.images.auth.version = "2.0.0"' "$PRODUCTION_MANIFEST"
images v1.2.3 ui demo job-api job-worker forward-proxy |
  jq -e '.auth == "2.0.0" and .ui == "1.2.3"' > /dev/null
expect_failure 'independent auth included in local release' images v1.2.3 auth ui

# Resolve references from bootstrap outputs, including custom repository names.
versions="$(images)"
repositories='{"auth":"registry-auth","ui":"registry-ui","demo":"registry-demo","job_api":"registry-api","job_worker":"registry-worker","forward_proxy":"registry-proxy"}'
resolve_images() {
  jq -cne --arg registry "$1" --arg project example-project \
    --argjson repositories "$2" --argjson versions "$versions" \
    -f "$repo_root/.github/scripts/image-references.jq"
}
resolve_images europe-west1-docker.pkg.dev "$repositories" |
  jq -e '. == {
    "auth": "europe-west1-docker.pkg.dev/example-project/registry-auth/auth:v2.0.0",
    "ui": "europe-west1-docker.pkg.dev/example-project/registry-ui/ui:v1.2.3",
    "demo": "europe-west1-docker.pkg.dev/example-project/registry-demo/demo:v1.2.3",
    "job_api": "europe-west1-docker.pkg.dev/example-project/registry-api/job-api:v1.2.3",
    "job_worker": "europe-west1-docker.pkg.dev/example-project/registry-worker/job-worker:v1.2.3",
    "forward_proxy": "europe-west1-docker.pkg.dev/example-project/registry-proxy/forward-proxy:v1.2.3"
  }' > /dev/null
expect_failure 'missing bootstrap repository' resolve_images europe-west1-docker.pkg.dev \
  "$(jq -c 'del(.auth)' <<< "$repositories")"
expect_failure 'invalid registry host' resolve_images 'registry.example/injected' "$repositories"

reset_manifest
yq -i '.images.auth.version = "v1.2.3"' "$PRODUCTION_MANIFEST"
expect_failure 'malformed version' images

reset_manifest
yq -i 'del(.images."job-worker")' "$PRODUCTION_MANIFEST"
expect_failure 'missing required image' images

reset_manifest
yq -i '.images.auth.repository = "registry.example/project/auth"' "$PRODUCTION_MANIFEST"
expect_failure 'repository field in version-only manifest' images

printf 'Production image manifest tests passed.\n'
