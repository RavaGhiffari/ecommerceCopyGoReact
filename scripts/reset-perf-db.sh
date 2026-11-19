#!/bin/sh
# Drops and recreates the marketplace_perf database (load-measurement data only (make seed-load)).
# Runs INSIDE the postgres container (piped in by `make seed-load`), where
# $POSTGRES_USER is already set and psql needs no password.
#
# The name is hard-coded on purpose: this script must never be pointed at the
# dev or test database. WITH (FORCE) ends any open connection (e.g. a running
# other session) so the drop cannot hang.
set -eu

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres <<'SQL'
DROP DATABASE IF EXISTS marketplace_perf WITH (FORCE);
CREATE DATABASE marketplace_perf;
SQL
