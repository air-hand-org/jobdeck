# syntax=docker/dockerfile:1

FROM golang:1.26.4-bookworm AS build
WORKDIR /src

# The services use the repository Go workspace to resolve generated protobuf code.
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/job-api ./services/job-api/cmd/server

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/job-api /job-api
USER nonroot:nonroot
ENTRYPOINT ["/job-api"]
