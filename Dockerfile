# syntax=docker/dockerfile:1

# Build stage: compile a static Go binary with the pure-Go SQLite driver
# (CGO_ENABLED=0), so the final stage can be a minimal distroless image with
# no C toolchain or shell.
#
# Base image pinned to golang:1.25 (not 1.23) because the mandated pure-Go
# SQLite driver (modernc.org/sqlite) pulls in transitive dependencies that
# require go >= 1.25 — see go.mod and 01-01-SUMMARY.md "Deviations".
FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/datechooser ./cmd/server
RUN mkdir -p /data

# Final stage: minimal, nonroot, no shell. Templates and static assets are
# embedded in the binary via go:embed, so no asset COPY is needed here.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/datechooser /datechooser
# /data is created (empty) and chowned to the nonroot uid in the build stage
# so a fresh named volume mounted here inherits writable ownership without
# requiring an init container or entrypoint chown step (OPS-03 fresh-start).
COPY --from=build --chown=65532:65532 /data /data

ENV PORT=8080
ENV DB_PATH=/data/datechooser.db

EXPOSE 8080
VOLUME ["/data"]
USER 65532:65532

ENTRYPOINT ["/datechooser"]
