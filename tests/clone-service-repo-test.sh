#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "${TEST_ROOT}"' EXIT

mkdir -p "${TEST_ROOT}/repo/tests" "${TEST_ROOT}/repo/environments" "${TEST_ROOT}/bin" "${TEST_ROOT}/state"
cp "${SCRIPT_DIR}/clone-service-repo.sh" "${TEST_ROOT}/repo/tests/"
mkdir -p "${TEST_ROOT}/repo/mkfiles"
cp "${SCRIPT_DIR}/../mkfiles/services.mk" "${TEST_ROOT}/repo/mkfiles/services.mk"
touch "${TEST_ROOT}/repo/environments/production.yaml"

# Real Git repositories and file-only transport keep clones/fetches hermetic.
export REAL_GIT
REAL_GIT=$(command -v git)
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_ALLOW_PROTOCOL=file
export GIT_AUTHOR_NAME='Clone test' GIT_AUTHOR_EMAIL='clone-test@example.invalid'
export GIT_COMMITTER_NAME="${GIT_AUTHOR_NAME}" GIT_COMMITTER_EMAIL="${GIT_AUTHOR_EMAIL}"
export TEST_REPO="${TEST_ROOT}/repo" TEST_STATE="${TEST_ROOT}/state" TEST_REMOTE="${TEST_ROOT}/remote"
mkdir -p "${TEST_REMOTE}"
for service in auth job-api job-worker ui demo; do
    git init -q "${TEST_REMOTE}/${service}"
    printf 'initial\n' > "${TEST_REMOTE}/${service}/version"
    git -C "${TEST_REMOTE}/${service}" add version
    git -C "${TEST_REMOTE}/${service}" commit -qm initial
    git -C "${TEST_REMOTE}/${service}" tag "v1.2.${#service}"
done
printf '1.2\n' > "${TEST_STATE}/version-prefix"

cat > "${TEST_ROOT}/bin/yq" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" == 3 && "$1" == -er && "$2" == '.images[strenv(SERVICE)].version' ]]
[[ "$3" == "${TEST_REPO}/environments/production.yaml" ]]
if [[ "${MISSING_VERSION_SERVICE:-}" == "${SERVICE}" ]]; then
    printf 'null\n'
else
    printf '%s.%s\n' "$(cat "${TEST_STATE}/version-prefix")" "${#SERVICE}"
fi
STUB

cat > "${TEST_ROOT}/bin/gh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" == 9 && "$1" == repo && "$2" == clone ]]
service=${3#air-hand-org/jobdeck-}
[[ "$3" == "air-hand-org/jobdeck-${service}" && "$4" == "${TEST_REPO}/services/${service}" ]]
[[ "$5" == -- && "$6" == --branch && "$7" == "v$(cat "${TEST_STATE}/version-prefix").${#service}" && "$8" == --depth && "$9" == 1 ]]
touch "${TEST_STATE}/${service}.started"
# Every clone must start before any finishes: a sequential implementation fails.
for ((attempt = 0; attempt < 100; attempt++)); do
    started=("${TEST_STATE}"/*.started)
    if [[ "${#started[@]}" == "${EXPECTED_CLONES}" ]]; then
        break
    fi
    sleep 0.02
done
[[ "${#started[@]}" == "${EXPECTED_CLONES}" ]]
if [[ "${FAIL_SERVICE:-}" == "${service}" ]]; then
    exit 42
fi
"${REAL_GIT}" clone "file://${TEST_REMOTE}/${service}" "$4" --branch "$7" --depth 1
touch "${TEST_STATE}/${service}.finished"
STUB

cat > "${TEST_ROOT}/bin/git" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == -C && "$2" == "${TEST_REPO}/services/"* ]]
service=${2##*/}
case "$3" in
    fetch) [[ "$#" == 8 && "$4" == --no-tags && "$5" == --depth && "$6" == 1 && "$7" == origin && "$8" == "refs/tags/"* ]] ;;
    switch) [[ "$#" == 5 && "$4" == --detach ]] ;;
    *) exit 99 ;;
esac
printf '%s\n' "${*:3}" >> "${TEST_STATE}/${service}.git"
if [[ "${FAIL_GIT_SERVICE:-}" == "${service}" && "${FAIL_GIT_COMMAND:-}" == "$3" ]]; then
    exit 43
fi
exec "${REAL_GIT}" "$@"
STUB
chmod +x "${TEST_ROOT}/bin/gh" "${TEST_ROOT}/bin/yq" "${TEST_ROOT}/bin/git"
export PATH="${TEST_ROOT}/bin:${PATH}"
export EXPECTED_CLONES=5

assert_checkout() {
    local service=$1 prefix=$2
    test "$("${REAL_GIT}" -C "${TEST_REPO}/services/${service}" rev-parse HEAD)" == \
        "$("${REAL_GIT}" -C "${TEST_REMOTE}/${service}" rev-parse "v${prefix}.${#service}")"
    if "${REAL_GIT}" -C "${TEST_REPO}/services/${service}" symbolic-ref -q HEAD; then
        printf 'Expected detached checkout for %s\n' "${service}" >&2
        exit 1
    fi
}

# Running outside the repository also checks make -f path resolution.
(cd "${TEST_ROOT}" && make -f "${TEST_REPO}/mkfiles/services.mk" clone-service-repo)
for service in auth job-api job-worker ui demo; do
    test -f "${TEST_STATE}/${service}.finished"
    test ! -e "${TEST_STATE}/${service}.git"
    assert_checkout "${service}" 1.2
    # Match the production shallow tag clone: no artificial branch tracking.
    test "$("${REAL_GIT}" -C "${TEST_REPO}/services/${service}" rev-parse --is-shallow-repository)" == true
    test "$("${REAL_GIT}" -C "${TEST_REPO}/services/${service}" config --get-all remote.origin.fetch)" == \
        "+refs/tags/v1.2.${#service}:refs/tags/v1.2.${#service}"
    # Publish a new commit/tag only after the shallow initial clone.
    printf 'updated\n' > "${TEST_REMOTE}/${service}/version"
    "${REAL_GIT}" -C "${TEST_REMOTE}/${service}" commit -qam updated
    "${REAL_GIT}" -C "${TEST_REMOTE}/${service}" tag "v1.3.${#service}"
done

# Existing checkouts must fetch newly published tags and switch, not skip/clone.
printf '1.3\n' > "${TEST_STATE}/version-prefix"
printf 'local edit\n' > "${TEST_REPO}/services/auth/local-file"
rm "${TEST_STATE}"/*.started "${TEST_STATE}"/*.finished
make -f "${TEST_REPO}/mkfiles/services.mk" clone-service-repo
for service in auth job-api job-worker ui demo; do
    assert_checkout "${service}" 1.3
    test "$(cat "${TEST_STATE}/${service}.git")" == "$(printf 'fetch --no-tags --depth 1 origin refs/tags/v1.3.%s:refs/tags/v1.3.%s\nswitch --detach refs/tags/v1.3.%s' "${#service}" "${#service}" "${#service}")"
    test ! -e "${TEST_STATE}/${service}.started"
done
test "$(cat "${TEST_REPO}/services/auth/local-file")" == 'local edit'

# Requesting an older version must also switch the existing checkout.
printf '1.2\n' > "${TEST_STATE}/version-prefix"
make -f "${TEST_REPO}/mkfiles/services.mk" clone-service-repo
for service in auth job-api job-worker ui demo; do
    assert_checkout "${service}" 1.2
    test ! -e "${TEST_STATE}/${service}.started"
done

# Real fetch failure must not fall back to a locally available stale tag.
"${REAL_GIT}" -C "${TEST_REMOTE}/auth" tag -d v1.2.4
if bash "${TEST_REPO}/tests/clone-service-repo.sh"; then
    printf 'Expected missing remote tag failure to propagate\n' >&2
    exit 1
fi
"${REAL_GIT}" -C "${TEST_REMOTE}/auth" tag v1.2.4 HEAD~1
assert_checkout auth 1.2
test "$(cat "${TEST_REPO}/services/auth/local-file")" == 'local edit'

# Git must refuse a switch that would overwrite tracked local edits.
printf 'local tracked edit\n' > "${TEST_REPO}/services/auth/version"
printf '1.3\n' > "${TEST_STATE}/version-prefix"
if bash "${TEST_REPO}/tests/clone-service-repo.sh"; then
    printf 'Expected conflicting local edit failure to propagate\n' >&2
    exit 1
fi
assert_checkout auth 1.2
test "$(cat "${TEST_REPO}/services/auth/version")" == 'local tracked edit'
test "$(cat "${TEST_REPO}/services/auth/local-file")" == 'local edit'
printf 'initial\n' > "${TEST_REPO}/services/auth/version"
printf '1.2\n' > "${TEST_STATE}/version-prefix"
bash "${TEST_REPO}/tests/clone-service-repo.sh"

# Mixed existing/missing checkouts: a failed clone propagates after peers finish.
rm -rf "${TEST_REPO}/services/job-api" "${TEST_REPO}/services/ui"
rm "${TEST_STATE}"/*.git
export EXPECTED_CLONES=2 FAIL_SERVICE=job-api
if bash "${TEST_REPO}/tests/clone-service-repo.sh"; then
    printf 'Expected clone failure to propagate\n' >&2
    exit 1
fi
test -f "${TEST_STATE}/ui.finished"
for service in auth job-worker demo; do
    test ! -e "${TEST_STATE}/${service}.started"
    assert_checkout "${service}" 1.2
done
test "$(cat "${TEST_REPO}/services/auth/local-file")" == 'local edit'

# Restore all checkouts before exercising update/version failure propagation.
unset FAIL_SERVICE
export EXPECTED_CLONES=1
rm "${TEST_STATE}"/*.started "${TEST_STATE}"/*.finished
bash "${TEST_REPO}/tests/clone-service-repo.sh"
for command in fetch switch; do
    rm "${TEST_STATE}"/*.git
    if FAIL_GIT_SERVICE=auth FAIL_GIT_COMMAND="${command}" bash "${TEST_REPO}/tests/clone-service-repo.sh"; then
        printf 'Expected %s failure to propagate\n' "${command}" >&2
        exit 1
    fi
    if [[ "${command}" == fetch ]]; then
        test "$(cat "${TEST_STATE}/auth.git")" == 'fetch --no-tags --depth 1 origin refs/tags/v1.2.4:refs/tags/v1.2.4'
    fi
    test "$(cat "${TEST_STATE}/demo.git")" == "$(printf 'fetch --no-tags --depth 1 origin refs/tags/v1.2.4:refs/tags/v1.2.4\nswitch --detach refs/tags/v1.2.4')"
done
rm "${TEST_STATE}"/*.git
if MISSING_VERSION_SERVICE=auth bash "${TEST_REPO}/tests/clone-service-repo.sh"; then
    printf 'Expected missing version failure to propagate\n' >&2
    exit 1
fi
test ! -e "${TEST_STATE}/auth.git"
printf 'clone-service-repo tests passed\n'
