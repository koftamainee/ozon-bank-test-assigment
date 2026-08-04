#!/bin/sh
set -e

sed \
  -e "s|\${APP_PASSWORD}|${APP_PASSWORD}|g" \
  -e "s|\${MIGRATOR_PASSWORD}|${MIGRATOR_PASSWORD}|g" \
  /docker-entrypoint-initdb.d/init.sql.template | psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
