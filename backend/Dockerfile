# syntax=docker/dockerfile:1

# ---- Build stage ----
# Must be >= the `go` line in go.mod.
FROM golang:1.27-alpine AS build

WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binaries so they run in a minimal runtime image.
# ./cmd/... builds every command: /out/api and /out/seed-admin.
RUN mkdir -p /out && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

# ---- Runtime stage ----
FROM alpine:3.20

RUN adduser -D -u 10001 app
USER app

WORKDIR /app
COPY --from=build /out/ /app/

EXPOSE 8081

ENTRYPOINT ["/app/api"]
