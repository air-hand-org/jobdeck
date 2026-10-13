#!/usr/bin/env bash
# Read-only, bounded diagnostics. Print sizes and selected paths, never config contents.
set -uo pipefail

run_diagnostic() {
  if ! timeout 30s "$@"; then
    printf 'Diagnostic unavailable or timed out: %s\n' "$1"
  fi
}

measure() {
  local label="$1" path="$2"
  printf '\n%s: %s\n' "$label" "$path"
  if [[ -d "$path" ]]; then
    run_diagnostic du -h --max-depth=1 -- "$path"
    run_diagnostic du --inodes --max-depth=1 -- "$path"
  else
    printf 'Directory does not exist\n'
  fi
}

printf '::group::E2E capacity: %s\n' "${1:-unspecified}"
date -u '+%Y-%m-%dT%H:%M:%SZ'
run_diagnostic df -h
run_diagnostic df -i
run_diagnostic docker system df -v

# Query only cache paths, not the complete Go environment or NuGet configuration.
if command -v go >/dev/null 2>&1; then
  for key in GOCACHE GOMODCACHE; do
    cache_path="$(timeout 10s go env "$key" 2>/dev/null)" || cache_path=""
    if [[ -n "$cache_path" ]]; then
      measure "$key" "$cache_path"
    fi
  done
fi
if command -v dotnet >/dev/null 2>&1; then
  nuget_paths="$(timeout 10s dotnet nuget locals all --list 2>/dev/null)" || nuget_paths=""
  while IFS= read -r entry; do
    case "$entry" in
      http-cache:*|global-packages:*|temp:*|plugins-cache:*)
        measure "NuGet ${entry%%:*}" "${entry#*: }"
        ;;
    esac
  done <<< "$nuget_paths"
fi
measure 'golangci-lint cache' "${XDG_CACHE_HOME:-$HOME/.cache}/golangci-lint"

workspace="${GITHUB_WORKSPACE:-$PWD}"
measure 'Service checkouts (including build outputs)' "$workspace/services"
measure 'AppHost build outputs' "$workspace/apphosts/bin"
measure 'AppHost intermediate outputs' "$workspace/apphosts/obj"
measure 'E2E build outputs and artifacts' "$workspace/tests/e2e"
measure 'Aspire home (includes dashboard state and CLI bundles)' "${ASPIRE_HOME:-$HOME/.aspire}"
if [[ -n "${ASPIRE_DASHBOARD_DATA_DIRECTORY:-}" ]]; then
  measure 'Explicit dashboard data directory' "$ASPIRE_DASHBOARD_DATA_DIRECTORY"
fi
measure 'Workspace Aspire state' "$workspace/.aspire"
measure 'AppHost Aspire state' "$workspace/apphosts/.aspire"

# Session/DCP state and None-mode dashboard SQLite databases can be temporary.
# Limit discovery to known prefixes, without listing files or traversing all /tmp.
shopt -s nullglob
temp_roots=("${TMPDIR:-/tmp}")
if [[ -n "${RUNNER_TEMP:-}" && "$RUNNER_TEMP" != "${TMPDIR:-/tmp}" ]]; then
  temp_roots+=("$RUNNER_TEMP")
fi
for temp_root in "${temp_roots[@]}"; do
  for state_path in "$temp_root"/aspire* "$temp_root"/Aspire* "$temp_root"/dcp*; do
    measure 'Temporary Aspire/DCP state' "$state_path"
  done
done
printf '::endgroup::\n'
