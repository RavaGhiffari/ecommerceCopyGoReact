#!/bin/sh
# Creates the marketplace_test database if it does not exist yet.
# Runs INSIDE the postgres container (piped in by `make test-db`), where
# $POSTGRES_USER is already set and psql needs no password.
set -eu

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres <<'SQL'
SELECT 'CREATE DATABASE marketplace_test'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'marketplace_test')\gexec
SQL
