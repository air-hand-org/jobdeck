# syntax=docker/dockerfile:1

FROM golang:1.26.4-bookworm AS build
WORKDIR /src

# The services use the repository Go workspace to resolve generated protobuf code.
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/auth ./services/auth/cmd/server

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/auth /auth
EXPOSE 9091
USER nonroot:nonroot
ENTRYPOINT ["/auth"]
