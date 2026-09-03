# syntax=docker/dockerfile:1

FROM golang:1.26.4-bookworm AS build
WORKDIR /src

# The services use the repository Go workspace to resolve generated protobuf code.
COPY . .
RUN --mount=type=secret,id=github_contents_token \
    --mount=type=cache,target=/go/pkg/mod,id=jobdeck-go-mod \
    --mount=type=cache,target=/root/.cache/go-build,id=jobdeck-go-build \
    GOPRIVATE=github.com/air-hand-org/api-contracts \
    GIT_CONFIG_COUNT=1 \
    GIT_CONFIG_KEY_0="url.https://x-access-token:$(cat /run/secrets/github_contents_token)@github.com/.insteadOf" \
    GIT_CONFIG_VALUE_0="https://github.com/" \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/job-api ./services/job-api/cmd/server
RUN --mount=type=cache,target=/go/pkg/mod,id=jobdeck-go-mod \
    --mount=type=cache,target=/root/.cache/go-build,id=jobdeck-go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/job-api-migrator ./services/job-api/cmd/migrator
RUN --mount=type=cache,target=/go/pkg/mod,id=jobdeck-go-mod \
    --mount=type=cache,target=/root/.cache/go-build,id=jobdeck-go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/job-api-entrypoint ./services/job-api/cmd/entrypoint

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/job-api /job-api
COPY --from=build /out/job-api-migrator /job-api-migrator
COPY --from=build /out/job-api-entrypoint /job-api-entrypoint
USER nonroot:nonroot
ENTRYPOINT ["/job-api-entrypoint"]
