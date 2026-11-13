# Common development commands. Run from the repo root.
#
# Only Docker is required: Go, sqlc and migrate all run in containers, so the
# same commands work whether or not Go is installed on the host.
# On Windows, install make with:  scoop install make
#
#   make help            list targets

# Git Bash (MSYS) rewrites arguments like "/src" into Windows paths
# ("C:/Program Files/Git/src"), which breaks `docker run -w /src`. Turn that off.
export MSYS_NO_PATHCONV := 1

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
        sqlc fmt vet test test-db test-integration seed-admin seed-dev

help:
	@echo "Stack:      up | down | logs | ps"
	@echo "Migrations: migrate-up | migrate-down | migrate-version | migrate-create name=... | migrate-force version=N"
	@echo "Code:       sqlc | fmt | vet | test | test-db | test-integration"
	@echo "Accounts:   seed-admin email=... name=... | seed-dev (demo catalog)"

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

# Creates (if missing) and migrates the separate marketplace_test database.
# Integration tests WIPE its tables, so they never touch marketplace_dev.
test-db:
	$(COMPOSE) exec -T postgres sh -s < scripts/create-test-db.sh
	$(COMPOSE) run --rm migrate-test up

# Integration tests against the Compose PostgreSQL (start it with `make up`).
# Credentials come from .env and are only used inside the container.
# -p 1: packages run one at a time because they share the test database.
test-integration: test-db
	docker run --rm --network $(NETWORK) --env-file .env -v "$(BACKEND_DIR):/src" -v gomodcache:/go/pkg/mod -v gobuildcache:/root/.cache/go-build -w /src $(GO_IMAGE) \
		sh -c 'TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/marketplace_test?sslmode=disable" go test -tags=integration -count=1 -p 1 ./...'

# ---- Accounts ----------------------------------------------------------------

# Creates an admin. Prompts for the password (or set ADMIN_PASSWORD).
# The api image must be built first (`make up`).
# make seed-admin email=admin@example.com name="Admin"
seed-admin:
ifndef email
	$(error usage: make seed-admin email=admin@example.com name="Admin")
endif
ifndef name
	$(error usage: make seed-admin email=admin@example.com name="Admin")
endif
	$(COMPOSE) run --rm -e ADMIN_PASSWORD --entrypoint /app/seed-admin api -email "$(email)" -name "$(name)"

# Loads demo categories, stores and products into an EMPTY dev database.
# Refuses APP_ENV=production; does nothing if categories already exist.
seed-dev:
	$(COMPOSE) run --rm --entrypoint /app/seed-dev api
