# Common development commands. Run from the repo root.
#
# Only Docker is required: Go, sqlc and migrate all run in containers, so the
# same commands work whether or not Go is installed on the host.
# On Windows, install make with:  scoop install make
#
#   make help            list targets

COMPOSE     := docker compose
GO_IMAGE    := golang:1.27-alpine
SQLC_IMAGE  := sqlc/sqlc:1.29.0
NETWORK     := goframework_default
BACKEND_DIR := $(CURDIR)/backend

# `$(GO) go test ./...` runs `go test` inside a Go container with backend/
# mounted. The two named volumes cache downloaded modules and build output.
GO := docker run --rm -v "$(BACKEND_DIR):/src" -v gomodcache:/go/pkg/mod -v gobuildcache:/root/.cache/go-build -w /src $(GO_IMAGE)

.PHONY: help up down logs ps \
        migrate-up migrate-down migrate-version migrate-create migrate-force \
        sqlc fmt vet test test-integration

help:
	@echo "Stack:      up | down | logs | ps"
	@echo "Migrations: migrate-up | migrate-down | migrate-version | migrate-create name=... | migrate-force version=N"
	@echo "Code:       sqlc | fmt | vet | test | test-integration"

# ---- Stack -------------------------------------------------------------------

up:
	$(COMPOSE) up -d --build

# Stops containers but KEEPS the database volume. (`down -v` would delete data.)
down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

# ---- Migrations (golang-migrate, see the `migrate` service in compose) -------

migrate-up:
	$(COMPOSE) run --rm migrate up

# Rolls back ONE migration. Run it again to go further back.
migrate-down:
	$(COMPOSE) run --rm migrate down 1

migrate-version:
	$(COMPOSE) run --rm migrate version

# make migrate-create name=create_users_table
migrate-create:
ifndef name
	$(error usage: make migrate-create name=create_users_table)
endif
	$(COMPOSE) run --rm migrate create -ext sql -dir /migrations -seq -digits 6 $(name)

# Only for a "dirty" database after a migration failed halfway: fix the SQL,
# repair the schema by hand, then mark the version as clean.
# make migrate-force version=1
migrate-force:
ifndef version
	$(error usage: make migrate-force version=N)
endif
	$(COMPOSE) run --rm migrate force $(version)

# ---- Code --------------------------------------------------------------------

sqlc:
	docker run --rm -v "$(BACKEND_DIR):/src" -w /src $(SQLC_IMAGE) generate

fmt:
	$(GO) gofmt -l -w .

vet:
	$(GO) go vet ./...

# Unit tests: no database needed.
test:
	$(GO) go test -count=1 ./...

# Integration tests run against the Compose PostgreSQL (start it with `make up`).
# Credentials come from .env and are only used inside the container.
test-integration:
	docker run --rm --network $(NETWORK) --env-file .env -v "$(BACKEND_DIR):/src" -v gomodcache:/go/pkg/mod -v gobuildcache:/root/.cache/go-build -w /src $(GO_IMAGE) \
		sh -c 'TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/$$POSTGRES_DB?sslmode=disable" go test -tags=integration -count=1 ./...'
