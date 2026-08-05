#!/bin/sh
set -e

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -v app_password="$APP_PASSWORD" \
  -v migrator_password="$MIGRATOR_PASSWORD" \
  -f /scripts/init.sql
