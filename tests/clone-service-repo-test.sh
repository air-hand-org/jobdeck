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

cat > "${TEST_ROOT}/bin/yq" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" == 3 && "$1" == -er && "$2" == '.images[strenv(SERVICE)].version' ]]
[[ "$3" == "${TEST_REPO}/environments/production.yaml" ]]
printf '1.2.%s\n' "${#SERVICE}"
STUB

cat > "${TEST_ROOT}/bin/gh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" == 9 && "$1" == repo && "$2" == clone ]]
service=${3#air-hand-org/jobdeck-}
[[ "$3" == "air-hand-org/jobdeck-${service}" && "$4" == "${TEST_REPO}/services/${service}" ]]
[[ "$5" == -- && "$6" == --branch && "$7" == "v1.2.${#service}" && "$8" == --depth && "$9" == 1 ]]
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
sleep 0.05
mkdir -p "$4"
touch "${TEST_STATE}/${service}.finished"
STUB
chmod +x "${TEST_ROOT}/bin/gh" "${TEST_ROOT}/bin/yq"
export PATH="${TEST_ROOT}/bin:${PATH}"
export TEST_REPO="${TEST_ROOT}/repo" TEST_STATE="${TEST_ROOT}/state"
export EXPECTED_CLONES=5

# Running outside the repository also checks make -f path resolution.
(cd /tmp && make -f "${TEST_REPO}/mkfiles/services.mk" clone-service-repo)
for service in auth job-api job-worker ui demo; do
    test -f "${TEST_STATE}/${service}.finished"
done

# A repeat must preserve local edits and avoid invoking either stub.
printf 'local edit\n' > "${TEST_REPO}/services/auth/local-file"
mv "${TEST_ROOT}/bin" "${TEST_ROOT}/saved-bin"
mkdir "${TEST_ROOT}/bin"
printf '#!/usr/bin/env bash\nexit 99\n' > "${TEST_ROOT}/bin/gh"
cp "${TEST_ROOT}/bin/gh" "${TEST_ROOT}/bin/yq"
chmod +x "${TEST_ROOT}/bin/gh" "${TEST_ROOT}/bin/yq"
make -f "${TEST_REPO}/mkfiles/services.mk" clone-service-repo
test "$(cat "${TEST_REPO}/services/auth/local-file")" == 'local edit'

# Restore the stubs and remove two checkouts to test partial skip and failure.
rm -rf "${TEST_ROOT}/bin"
mv "${TEST_ROOT}/saved-bin" "${TEST_ROOT}/bin"
rm -rf "${TEST_REPO}/services/job-api" "${TEST_REPO}/services/ui"
rm "${TEST_STATE}"/*
export EXPECTED_CLONES=2 FAIL_SERVICE=job-api
if bash "${TEST_REPO}/tests/clone-service-repo.sh"; then
    printf 'Expected clone failure to propagate\n' >&2
    exit 1
fi
test -f "${TEST_STATE}/ui.finished"
test ! -e "${TEST_STATE}/auth.started"
test ! -e "${TEST_STATE}/job-worker.started"
test ! -e "${TEST_STATE}/demo.started"
test "$(cat "${TEST_REPO}/services/auth/local-file")" == 'local edit'
printf 'clone-service-repo tests passed\n'
