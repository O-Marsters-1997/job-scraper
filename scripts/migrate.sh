#!/bin/sh
set -eu

dsn="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE:-disable}"

goose -dir ./migrations postgres "$dsn" up
psql "$dsn" -v ON_ERROR_STOP=1 -q -f ./seed/scoring_options.sql
