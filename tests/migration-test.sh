#!/usr/bin/env bash
set -euo pipefail

POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:18}"
export TARGET_SERVICE="${TARGET_SERVICE:-job-api}"

container="$(
  docker run -d --rm \
    -e POSTGRES_USER=postgres \
    -e POSTGRES_PASSWORD=postgres \
    -e POSTGRES_DB=jobdeck \
    -p 127.0.0.1::5432 \
    "${POSTGRES_IMAGE}"
)"

cleanup() {
  docker rm -f "${container}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

port=""
for _ in {1..30}; do
  port="$(docker port "${container}" 5432/tcp | awk -F: '{print $NF}')"
  # pg_isready reports an accepting server even while POSTGRES_DB is still
  # being created. Run a query against the target database instead.
  if [[ -n "${port}" ]] && docker exec "${container}" psql -U postgres -d jobdeck -c 'SELECT 1' >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker exec "${container}" psql -U postgres -d jobdeck -c 'SELECT 1' >/dev/null 2>&1; then
  docker logs "${container}"
  exit 1
fi

export POSTGRESDB_URI="postgres://postgres:postgres@127.0.0.1:${port}/jobdeck?sslmode=disable"
(
  cd services/${TARGET_SERVICE}
  go run ./cmd/migrator
)

(
  cd services/${TARGET_SERVICE}
  go run ./cmd/migrator -down-to-0
)

(
  cd services/${TARGET_SERVICE}
  go run ./cmd/migrator
)
