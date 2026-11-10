# syntax=docker/dockerfile:1

# ---- Build stage ----
# Matches `go 1.27.1` in go.mod. Adjust the tag if your registry doesn't have it.
FROM golang:1.27-alpine AS build

WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary so it runs in a minimal runtime image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

# ---- Runtime stage ----
FROM alpine:3.20

RUN adduser -D -u 10001 app
USER app

WORKDIR /app
COPY --from=build /out/api /app/api

EXPOSE 8081

ENTRYPOINT ["/app/api"]
