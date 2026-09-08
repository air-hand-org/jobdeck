#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT
export PRODUCTION_MANIFEST="$test_dir/production.yaml"

reset_manifest() {
  cat > "$PRODUCTION_MANIFEST" <<'YAML'
images:
  auth: {repository: registry.example/project/auth, version: '1.2.3'}
  ui: {repository: registry.example/project/ui, version: '1.2.3'}
  demo: {repository: registry.example/project/demo, version: '1.2.3'}
  job-api: {repository: registry.example/project/job-api, version: '1.2.3'}
  job-worker: {repository: registry.example/project/job-worker, version: '1.2.3'}
  forward-proxy: {repository: registry.example/project/forward-proxy, version: '1.2.3'}
YAML
}

images() {
  bash "$repo_root/.github/scripts/production-images.sh" "$@"
}

expect_failure() {
  local description="$1"
  shift
  if images "$@" > "$test_dir/output" 2> "$test_dir/error"; then
    printf 'FAIL: %s was accepted\n' "$description" >&2
    exit 1
  fi
}

reset_manifest
images v1.2.3 auth ui demo job-api job-worker forward-proxy |
  jq -e '. == {
    "auth": "registry.example/project/auth:v1.2.3",
    "ui": "registry.example/project/ui:v1.2.3",
    "demo": "registry.example/project/demo:v1.2.3",
    "job_api": "registry.example/project/job-api:v1.2.3",
    "job_worker": "registry.example/project/job-worker:v1.2.3",
    "forward_proxy": "registry.example/project/forward-proxy:v1.2.3"
  }' > /dev/null
expect_failure 'mismatched release' v1.2.4 auth ui

# Extracted services need not follow the monorepo release version.
yq -i '.images.auth.version = "2.0.0"' "$PRODUCTION_MANIFEST"
images v1.2.3 ui demo job-api job-worker forward-proxy |
  jq -e '.auth == "registry.example/project/auth:v2.0.0" and
    .ui == "registry.example/project/ui:v1.2.3"' > /dev/null
expect_failure 'independent auth included in local release' v1.2.3 auth ui

reset_manifest
yq -i '.images.auth.version = "v1.2.3"' "$PRODUCTION_MANIFEST"
expect_failure 'malformed version'

reset_manifest
yq -i 'del(.images."job-worker")' "$PRODUCTION_MANIFEST"
expect_failure 'missing required image'

reset_manifest
yq -i '.images.auth.repository = "registry.example/project/auth\ninjected=value"' "$PRODUCTION_MANIFEST"
expect_failure 'repository containing output injection'

printf 'Production image manifest tests passed.\n'
