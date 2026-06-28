#!/usr/bin/env bash
set -euo pipefail

POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:18}"

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
  port="$(docker port "${container}" 5432/tcp | sed 's/.*://')"
  if [[ -n "${port}" ]] && docker exec "${container}" pg_isready -U postgres -d jobdeck >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker exec "${container}" pg_isready -U postgres -d jobdeck >/dev/null 2>&1; then
  docker logs "${container}"
  exit 1
fi

(
  cd services/job-api
  POSTGRESDB_URI="postgres://postgres:postgres@127.0.0.1:${port}/jobdeck?sslmode=disable" go run ./cmd/migrator
)
