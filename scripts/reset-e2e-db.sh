#!/bin/sh
# Drops and recreates the marketplace_e2e database (browser E2E tests only).
# Runs INSIDE the postgres container (piped in by `make e2e-reset`), where
# $POSTGRES_USER is already set and psql needs no password.
#
# The name is hard-coded on purpose: this script must never be pointed at the
# dev or test database. WITH (FORCE) ends any open connection (e.g. a running
# api-e2e container) so the drop cannot hang.
set -eu

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres <<'SQL'
DROP DATABASE IF EXISTS marketplace_e2e WITH (FORCE);
CREATE DATABASE marketplace_e2e;
SQL
