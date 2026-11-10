#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/.." && pwd)
SERVICES=(auth job-api job-worker ui demo)
PIDS=()

mkdir -p "${REPO_ROOT}/services"

for service in "${SERVICES[@]}"; do
    destination="${REPO_ROOT}/services/${service}"
    (
        version=$(SERVICE="${service}" yq -er '.images[strenv(SERVICE)].version' "${REPO_ROOT}/environments/production.yaml")
        if [[ -z "${version}" || "${version}" == null ]]; then
            printf 'Missing production version for %s\n' "${service}" >&2
            exit 1
        fi
        if [[ -e "${destination}" || -L "${destination}" ]]; then
            printf 'Switch %s to v%s\n' "${service}" "${version}"
            # A shallow tag clone tracks only its initial tag, not new releases.
            git -C "${destination}" fetch --no-tags --depth 1 origin "refs/tags/v${version}:refs/tags/v${version}"
            git -C "${destination}" switch --detach "refs/tags/v${version}"
        else
            printf 'Cloning %s at v%s\n' "${service}" "${version}"
            gh repo clone "air-hand-org/jobdeck-${service}" "${destination}" -- --branch "v${version}" --depth 1
        fi
    ) &
    PIDS+=("$!")
done

status=0
for pid in "${PIDS[@]}"; do
    if ! wait "${pid}"; then
        status=1
    fi
done
exit "${status}"
