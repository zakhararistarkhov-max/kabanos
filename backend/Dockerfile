# syntax=docker/dockerfile:1

# --- build stage ------------------------------------------------------------
# Compiles both the API and the worker as static binaries. Dependency download
# is a separate layer so it is cached across source-only changes.
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static, stripped binaries. tzdata is embedded via `import _ "time/tzdata"`.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# --- api runtime ------------------------------------------------------------
# distroless/static:nonroot: no shell, no package manager, runs as an
# unprivileged user — minimal attack surface and image size.
FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]

# --- worker runtime ---------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
