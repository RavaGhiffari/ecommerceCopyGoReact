# syntax=docker/dockerfile:1

# ---- Build stage ----
# Must be >= the `go` line in go.mod. Exact tags (Phase 7e): a floating tag
# like golang:1.27-alpine changes under you (it moved to 1.27.4 / Alpine 3.24
# without any change here). Bump these on purpose, together with the Makefile
# (GO_IMAGE) and CI.
FROM golang:1.27.4-alpine3.24 AS build

WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binaries so they run in a minimal runtime image.
# ./cmd/... builds every command: /out/api and /out/seed-admin.
RUN mkdir -p /out && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

# ---- Runtime stage ----
FROM alpine:3.24.4

RUN adduser -D -u 10001 app
USER app

WORKDIR /app
COPY --from=build /out/ /app/

EXPOSE 8081

ENTRYPOINT ["/app/api"]
