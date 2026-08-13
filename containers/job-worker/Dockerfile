# syntax=docker/dockerfile:1

FROM golang:1.26.4-bookworm AS build
WORKDIR /src

# The services use the repository Go workspace to resolve generated protobuf code.
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,id=jobdeck-go-mod \
    --mount=type=cache,target=/root/.cache/go-build,id=jobdeck-go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/job-worker ./services/job-worker/cmd/worker

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/job-worker /job-worker
USER nonroot:nonroot
ENTRYPOINT ["/job-worker"]
